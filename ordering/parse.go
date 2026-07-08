package ordering

import (
	"fmt"
	"strings"

	"github.com/AndreiCocan/golang-aip/ordering/ast"
)

// Parse parses orderBy into its syntactic tree, without consulting any
// schema. An empty (or blank) order_by is valid and yields an
// [ast.OrderBy] with no fields, meaning the service's default order.
//
// The syntax is a comma-separated list of dotted field paths, each with an
// optional "desc" suffix; ascending is the default and has no suffix.
// Redundant whitespace is insignificant. The suffix is matched
// case-insensitively, and only in suffix position, so a field named "desc"
// remains referencable. An explicit "asc" suffix is not part of the syntax
// and is rejected.
//
// Errors are [*ParseError] values matching [ErrInvalidOrderBy], carrying
// the byte offset of the offending token.
func Parse(orderBy string) (*ast.OrderBy, error) {
	tokens, err := lex(orderBy)
	if err != nil {
		return nil, err
	}

	p := &parser{orderBy: orderBy, tokens: tokens}
	ob := &ast.OrderBy{Source: orderBy}

	if p.peek().kind == tokenEOF {
		return ob, nil
	}

	for {
		field, err := p.parseField()
		if err != nil {
			return nil, err
		}

		ob.Fields = append(ob.Fields, field)

		tok := p.peek()
		if tok.kind == tokenEOF {
			return ob, nil
		}

		if tok.kind != tokenComma {
			return nil, p.errorf(tok.pos, `expected ",", got %s`, tokenDesc(tok))
		}

		p.next()
	}
}

// tokenKind enumerates the tokens of the order_by syntax.
type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenText
	tokenComma
	tokenDot
)

type token struct {
	kind tokenKind
	// pos is the byte offset of the token in the order_by string.
	pos int
	// text is the word of a TEXT token.
	text string
}

// lastControlByte is the highest ASCII control character; bytes above it
// are printable ASCII or part of a multi-byte UTF-8 rune.
const lastControlByte = 0x1f

// isTextByte reports whether c may appear inside a TEXT token. TEXT stops
// at whitespace, separators, and control characters, and excludes the
// structural bytes of the filter syntax for consistent tokens across the
// module; everything else, including "-", "*", and all non-ASCII bytes,
// is text.
func isTextByte(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '(', ')', ',', '.', '<', '>', '=', '!', ':', '"', '\'':
		return false
	}

	return c > lastControlByte
}

// lex splits orderBy into tokens, ending with a tokenEOF token. It fails
// with a [*ParseError] on any byte that is neither whitespace, a
// separator, nor text.
func lex(orderBy string) ([]token, error) {
	var tokens []token

	i := 0
	for i < len(orderBy) {
		c := orderBy[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == ',':
			tokens = append(tokens, token{kind: tokenComma, pos: i})
			i++
		case c == '.':
			tokens = append(tokens, token{kind: tokenDot, pos: i})
			i++
		case isTextByte(c):
			start := i
			for i < len(orderBy) && isTextByte(orderBy[i]) {
				i++
			}

			tokens = append(tokens, token{kind: tokenText, pos: start, text: orderBy[start:i]})
		default:
			return nil, &ParseError{
				OrderBy: orderBy,
				Pos:     i,
				Message: fmt.Sprintf("unexpected character %q", c),
			}
		}
	}

	return append(tokens, token{kind: tokenEOF, pos: len(orderBy)}), nil
}

type parser struct {
	orderBy string
	tokens  []token
	i       int
}

// peek returns the current token without consuming it.
func (p *parser) peek() token { return p.tokens[p.i] }

// next consumes the current token. The EOF token is never consumed.
func (p *parser) next() {
	if p.tokens[p.i].kind != tokenEOF {
		p.i++
	}
}

func (p *parser) errorf(pos int, format string, args ...any) error {
	return &ParseError{OrderBy: p.orderBy, Pos: pos, Message: fmt.Sprintf(format, args...)}
}

// parseField parses `segment {"." segment} ["desc"]`.
func (p *parser) parseField() (ast.Field, error) {
	head := p.peek()
	if head.kind != tokenText {
		return ast.Field{}, p.errorf(head.pos, "expected field, got %s", tokenDesc(head))
	}

	p.next()

	field := ast.Field{Pos: head.pos, Segments: []string{head.text}}

	for p.peek().kind == tokenDot {
		p.next()

		seg := p.peek()
		if seg.kind != tokenText {
			return ast.Field{}, p.errorf(
				seg.pos,
				`expected field segment after ".", got %s`,
				tokenDesc(seg),
			)
		}

		p.next()

		field.Segments = append(field.Segments, seg.text)
	}

	// A word after the path can only be the "desc" suffix. The keyword is
	// positional: in path position "desc" is an ordinary segment.
	if tok := p.peek(); tok.kind == tokenText {
		switch {
		case strings.EqualFold(tok.text, "desc"):
			p.next()

			field.Desc = true
		case strings.EqualFold(tok.text, "asc"):
			return ast.Field{}, p.errorf(
				tok.pos,
				`ascending is the default; only "desc" may follow a field`,
			)
		default:
			return ast.Field{}, p.errorf(
				tok.pos,
				`expected "desc", ",", or end of order by, got %s`,
				tokenDesc(tok),
			)
		}
	}

	return field, nil
}

// tokenDesc describes a token for an error message.
func tokenDesc(tok token) string {
	switch tok.kind {
	case tokenEOF:
		return "end of order by"
	case tokenText:
		return fmt.Sprintf("%q", tok.text)
	case tokenDot:
		return `"."`
	case tokenComma:
		return `","`
	default:
		return "unknown token"
	}
}
