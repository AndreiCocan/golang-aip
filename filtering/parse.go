package filtering

import (
	"fmt"

	"github.com/AndreiCocan/golang-aip/filtering/ast"
)

// maxNestingDepth bounds parenthesis and function-call nesting so that
// adversarial filters cannot exhaust the stack.
const maxNestingDepth = 50

// Parse parses filter into its syntactic tree, without consulting any
// schema. An empty (or blank) filter is valid and yields a [ast.Filter]
// with a nil expression.
//
// Errors are [*ParseError] values matching [ErrInvalidFilter], carrying the
// byte offset of the offending token.
func Parse(filter string) (*ast.Filter, error) {
	tokens, err := lex(filter)
	if err != nil {
		return nil, err
	}

	p := &parser{filter: filter, tokens: tokens}
	if p.peek().kind == tokenEOF {
		return &ast.Filter{Source: filter}, nil
	}

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if tok := p.peek(); tok.kind != tokenEOF {
		return nil, p.errorf(tok.pos, "unexpected %s", tokenDesc(tok))
	}

	return &ast.Filter{Source: filter, Expr: expr}, nil
}

type parser struct {
	filter string
	tokens []token
	i      int
	depth  int
}

// peek returns the current token without consuming it.
func (p *parser) peek() token { return p.tokens[p.i] }

// peekNext returns the token after the current one, or the final EOF token.
func (p *parser) peekNext() token {
	if p.i+1 < len(p.tokens) {
		return p.tokens[p.i+1]
	}

	return p.tokens[len(p.tokens)-1]
}

// next consumes and returns the current token. The EOF token is never
// consumed.
func (p *parser) next() token {
	tok := p.tokens[p.i]
	if tok.kind != tokenEOF {
		p.i++
	}

	return tok
}

func (p *parser) errorf(pos int, format string, args ...any) error {
	return &ParseError{Filter: p.filter, Pos: pos, Message: fmt.Sprintf(format, args...)}
}

func isKeywordText(s string) bool { return s == "AND" || s == "OR" || s == "NOT" }

// atKeyword reports whether the current token is the bare keyword kw. A
// keyword immediately followed by "(" is not a keyword but a (discouraged
// yet grammatical) function name, such as `AND(b)`.
func (p *parser) atKeyword(kw string) bool {
	tok := p.peek()
	if tok.kind != tokenText || tok.text != kw {
		return false
	}

	next := p.peekNext()

	return next.kind != tokenLparen || next.pos != tok.end
}

// parseExpression parses `sequence {WS AND WS sequence}`.
func (p *parser) parseExpression() (*ast.Expression, error) {
	sequences, err := parseKeywordSeparated(p, "AND", p.parseSequence)
	if err != nil {
		return nil, err
	}

	return &ast.Expression{Sequences: sequences}, nil
}

// parseKeywordSeparated parses `elem {WS kw WS elem}`, the loop shared by
// the AND and OR grammar levels.
func parseKeywordSeparated[T any](p *parser, kw string, parse func() (T, error)) ([]T, error) {
	elem, err := parse()
	if err != nil {
		return nil, err
	}

	elems := []T{elem}

	for p.atKeyword(kw) {
		tok := p.peek()
		if !tok.spaceBefore {
			return nil, p.errorf(tok.pos, "%q must be preceded by whitespace", kw)
		}

		p.next()

		if !p.peek().spaceBefore {
			return nil, p.errorf(p.peek().pos, "%q must be followed by whitespace", kw)
		}

		if elem, err = parse(); err != nil {
			return nil, err
		}

		elems = append(elems, elem)
	}

	return elems, nil
}

// parseSequence parses `factor {WS factor}`.
func (p *parser) parseSequence() (*ast.Sequence, error) {
	factor, err := p.parseFactor()
	if err != nil {
		return nil, err
	}

	factors := []*ast.Factor{factor}

	for p.startsFactor() {
		if tok := p.peek(); !tok.spaceBefore {
			return nil, p.errorf(tok.pos, "expected whitespace between filter terms")
		}

		if factor, err = p.parseFactor(); err != nil {
			return nil, err
		}

		factors = append(factors, factor)
	}

	return &ast.Sequence{Factors: factors}, nil
}

// startsFactor reports whether the current token can begin a new factor of
// the running sequence. The AND and OR keywords cannot: they continue the
// enclosing expression or factor instead.
func (p *parser) startsFactor() bool {
	switch p.peek().kind {
	case tokenText:
		return !p.atKeyword("AND") && !p.atKeyword("OR")
	case tokenString, tokenMinus, tokenLparen:
		return true
	default:
		return false
	}
}

// parseFactor parses `term {WS OR WS term}`.
func (p *parser) parseFactor() (*ast.Factor, error) {
	terms, err := parseKeywordSeparated(p, "OR", p.parseTerm)
	if err != nil {
		return nil, err
	}

	return &ast.Factor{Terms: terms}, nil
}

// parseTerm parses `[(NOT WS | MINUS)] simple`.
func (p *parser) parseTerm() (*ast.Term, error) {
	switch tok := p.peek(); {
	case tok.kind == tokenMinus:
		p.next()

		if next := p.peek(); next.spaceBefore || next.kind == tokenEOF {
			return nil, p.errorf(tok.pos, `"-" must be directly followed by an expression`)
		}

		simple, err := p.parseSimple()
		if err != nil {
			return nil, err
		}

		return &ast.Term{NegPos: tok.pos, Neg: ast.NegationMinus, Simple: simple}, nil
	case p.atKeyword("NOT"):
		if next := p.peekNext(); !next.spaceBefore || next.kind == tokenEOF {
			return nil, p.errorf(tok.pos, `"NOT" must be followed by whitespace and an expression`)
		}

		p.next()

		simple, err := p.parseSimple()
		if err != nil {
			return nil, err
		}

		return &ast.Term{NegPos: tok.pos, Neg: ast.NegationNot, Simple: simple}, nil
	default:
		simple, err := p.parseSimple()
		if err != nil {
			return nil, err
		}

		return &ast.Term{Simple: simple}, nil
	}
}

// parseSimple parses a restriction or a parenthesized composite.
func (p *parser) parseSimple() (ast.Simple, error) {
	if p.peek().kind == tokenLparen {
		return p.parseComposite()
	}

	return p.parseRestriction()
}

// parseComposite parses `( expression )`.
func (p *parser) parseComposite() (*ast.Composite, error) {
	lparen := p.peek()
	if err := p.push(lparen.pos); err != nil {
		return nil, err
	}
	defer p.pop()

	p.next()

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	rparen := p.peek()
	if rparen.kind != tokenRparen {
		return nil, p.errorf(rparen.pos, `expected ")", got %s`, tokenDesc(rparen))
	}

	p.next()

	return &ast.Composite{Lparen: lparen.pos, Expr: expr, Rparen: rparen.pos}, nil
}

// parseRestriction parses `comparable [comparator arg]`.
func (p *parser) parseRestriction() (*ast.Restriction, error) {
	left, err := p.parseComparable()
	if err != nil {
		return nil, err
	}

	op := comparatorOf(p.peek().kind)
	if op == ast.ComparatorNone {
		return &ast.Restriction{Comparable: left}, nil
	}

	opTok := p.next()

	arg, err := p.parseArg()
	if err != nil {
		return nil, err
	}

	return &ast.Restriction{Comparable: left, OpPos: opTok.pos, Op: op, Arg: arg}, nil
}

// comparatorOf maps a comparator token to its [ast.Comparator], or
// ComparatorNone for any other token.
func comparatorOf(kind tokenKind) ast.Comparator {
	switch kind {
	case tokenEquals:
		return ast.ComparatorEquals
	case tokenNotEquals:
		return ast.ComparatorNotEquals
	case tokenLess:
		return ast.ComparatorLess
	case tokenLessEquals:
		return ast.ComparatorLessEquals
	case tokenGreater:
		return ast.ComparatorGreater
	case tokenGreaterEquals:
		return ast.ComparatorGreaterEquals
	case tokenHas:
		return ast.ComparatorHas
	default:
		return ast.ComparatorNone
	}
}

// parseComparable parses `member | function`. The two are distinguished by
// lookahead: a dotted name chain immediately followed by "(" is a function
// call.
func (p *parser) parseComparable() (ast.Comparable, error) {
	head := p.peek()
	if head.kind != tokenText && head.kind != tokenString {
		return nil, p.errorf(head.pos, "expected value, got %s", tokenDesc(head))
	}

	p.next()

	fields, err := p.parseDotFields()
	if err != nil {
		return nil, err
	}

	segments := make([]token, 0, 1+len(fields))
	segments = append(segments, head)
	segments = append(segments, fields...)

	last := segments[len(segments)-1]
	if lparen := p.peek(); lparen.kind == tokenLparen && lparen.pos == last.end {
		for _, s := range segments {
			if s.kind == tokenString {
				return nil, p.errorf(s.pos, "function name segments must not be quoted")
			}
		}

		return p.parseFunctionCall(segments)
	}

	if head.kind == tokenText && isKeywordText(head.text) {
		return nil, p.errorf(
			head.pos,
			"%q is a keyword and cannot start a value; quote it to match it literally",
			head.text,
		)
	}

	member := &ast.Member{Value: valueOf(head)}
	for _, f := range segments[1:] {
		member.Fields = append(member.Fields, valueOf(f))
	}

	return member, nil
}

// parseDotFields parses `{DOT field}`, returning the field tokens. Fields
// may be TEXT (including keywords) or quoted strings, the latter allowing
// map keys that contain dots.
func (p *parser) parseDotFields() ([]token, error) {
	var fields []token

	for p.peek().kind == tokenDot {
		p.next()

		field := p.peek()
		if field.kind != tokenText && field.kind != tokenString {
			return nil, p.errorf(field.pos, `expected field after ".", got %s`, tokenDesc(field))
		}

		p.next()

		fields = append(fields, field)
	}

	return fields, nil
}

// parseFunctionCall parses `( [argList] )` after the already-consumed name
// segments.
func (p *parser) parseFunctionCall(name []token) (*ast.Function, error) {
	lparen := p.peek()
	if err := p.push(lparen.pos); err != nil {
		return nil, err
	}
	defer p.pop()

	p.next()

	fn := &ast.Function{Lparen: lparen.pos}
	for _, n := range name {
		fn.Name = append(fn.Name, valueOf(n))
	}

	if p.peek().kind != tokenRparen {
		for {
			arg, err := p.parseArg()
			if err != nil {
				return nil, err
			}

			fn.Args = append(fn.Args, arg)

			if p.peek().kind != tokenComma {
				break
			}

			p.next()
		}
	}

	rparen := p.peek()
	if rparen.kind != tokenRparen {
		return nil, p.errorf(rparen.pos, `expected ")", got %s`, tokenDesc(rparen))
	}

	p.next()

	fn.Rparen = rparen.pos

	return fn, nil
}

// parseArg parses `comparable | composite`. As a practical extension beyond
// the letter of the grammar, a "-" directly followed by a number is folded
// into a negative numeric literal, so that `a = -30` works even though
// negation is otherwise a term-level construct.
func (p *parser) parseArg() (ast.Arg, error) {
	switch tok := p.peek(); tok.kind {
	case tokenLparen:
		return p.parseComposite()
	case tokenMinus:
		next := p.peekNext()
		if next.kind != tokenText || next.pos != tok.end ||
			next.text == "" || next.text[0] < '0' || next.text[0] > '9' {
			return nil, p.errorf(tok.pos, `expected a number after "-"`)
		}

		p.next()
		head := p.next()
		member := &ast.Member{Value: ast.Value{ValuePos: tok.pos, Text: "-" + head.text}}

		fields, err := p.parseDotFields()
		if err != nil {
			return nil, err
		}

		for _, f := range fields {
			member.Fields = append(member.Fields, valueOf(f))
		}

		return member, nil
	default:
		left, err := p.parseComparable()
		if err != nil {
			return nil, err
		}

		arg, ok := left.(ast.Arg)
		if !ok {
			return nil, fmt.Errorf("filtering: comparable node %T is not an argument", left)
		}

		return arg, nil
	}
}

// push enters a nesting level, failing once the filter nests too deeply.
func (p *parser) push(pos int) error {
	p.depth++
	if p.depth > maxNestingDepth {
		return p.errorf(pos, "filter is nested more than %d levels deep", maxNestingDepth)
	}

	return nil
}

func (p *parser) pop() { p.depth-- }

// valueOf converts a TEXT or STRING token to an [ast.Value].
func valueOf(tok token) ast.Value {
	return ast.Value{ValuePos: tok.pos, Text: tok.text, Quoted: tok.kind == tokenString}
}

// tokenDesc describes a token for an error message.
func tokenDesc(tok token) string {
	switch tok.kind {
	case tokenEOF:
		return "end of filter"
	case tokenText, tokenString:
		return fmt.Sprintf("%q", tok.text)
	case tokenDot:
		return `"."`
	case tokenComma:
		return `","`
	case tokenLparen:
		return `"("`
	case tokenRparen:
		return `")"`
	case tokenMinus:
		return `"-"`
	case tokenEquals:
		return `"="`
	case tokenNotEquals:
		return `"!="`
	case tokenLess:
		return `"<"`
	case tokenLessEquals:
		return `"<="`
	case tokenGreater:
		return `">"`
	case tokenGreaterEquals:
		return `">="`
	case tokenHas:
		return `":"`
	default:
		return "unknown token"
	}
}
