package filtering

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AndreiCocan/golang-aip/filtering/ast"
)

// Check validates a parsed filter against a schema and resolves it into a
// [Checked] filter: field paths are verified and annotated with their
// types, literals are converted to typed values, wildcard strings become
// patterns, and declared functions are arity- and type-checked (and
// expanded, for macro functions).
//
// Errors are [*CheckError] values matching [ErrInvalidFilter], except for
// errors returned by function expanders, which are wrapped unchanged so
// that internal failures are not reported as invalid filters.
//
// schema must not be nil; use NewSchema() for a schema with no filterable
// fields.
func Check(filter *ast.Filter, schema *Schema) (*Checked, error) {
	if filter == nil || filter.Expr == nil {
		return &Checked{}, nil
	}

	c := &checker{schema: schema, source: filter.Source}

	expr, err := c.expression(filter.Expr)
	if err != nil {
		return nil, err
	}

	return &Checked{Expr: expr}, nil
}

type checker struct {
	schema *Schema
	source string
}

func (c *checker) errorf(pos int, format string, args ...any) error {
	return &CheckError{Filter: c.source, Pos: pos, Message: fmt.Sprintf(format, args...)}
}

// expression checks a conjunction. Both AND and sequence adjacency conjoin,
// so the two grammar levels collapse into one [And] node; nested
// conjunctions are flattened and adjacent [Search] operands merged.
func (c *checker) expression(e *ast.Expression) (Expr, error) {
	var operands []Expr

	for _, s := range e.Sequences {
		for _, f := range s.Factors {
			x, err := c.factor(f)
			if err != nil {
				return nil, err
			}

			operands = appendConjunct(operands, x)
		}
	}

	if len(operands) == 1 {
		return operands[0], nil
	}

	return &And{Operands: operands}, nil
}

// appendConjunct adds x to a conjunction's operand list, flattening nested
// [And] nodes and merging adjacent [Search] operands.
func appendConjunct(operands []Expr, x Expr) []Expr {
	switch x := x.(type) {
	case *And:
		for _, op := range x.Operands {
			operands = appendConjunct(operands, op)
		}

		return operands
	case *Search:
		if len(operands) > 0 {
			if last, ok := operands[len(operands)-1].(*Search); ok {
				last.Terms = append(last.Terms, x.Terms...)

				return operands
			}
		}
	}

	return append(operands, x)
}

// factor checks a disjunction of terms.
func (c *checker) factor(f *ast.Factor) (Expr, error) {
	if len(f.Terms) == 1 {
		return c.term(f.Terms[0])
	}

	operands := make([]Expr, 0, len(f.Terms))
	for _, t := range f.Terms {
		x, err := c.term(t)
		if err != nil {
			return nil, err
		}

		operands = append(operands, x)
	}

	return &Or{Operands: operands}, nil
}

// term checks an optionally negated simple expression.
func (c *checker) term(t *ast.Term) (Expr, error) {
	x, err := c.simple(t.Simple)
	if err != nil {
		return nil, err
	}

	if t.Neg != ast.NegationNone {
		return &Not{Operand: x}, nil
	}

	return x, nil
}

func (c *checker) simple(s ast.Simple) (Expr, error) {
	switch s := s.(type) {
	case *ast.Composite:
		return c.expression(s.Expr)
	case *ast.Restriction:
		return c.restriction(s)
	default:
		return nil, fmt.Errorf("filtering: unhandled simple node %T", s)
	}
}

func (c *checker) restriction(r *ast.Restriction) (Expr, error) {
	if r.Op == ast.ComparatorNone {
		return c.global(r.Comparable)
	}

	op := operatorOf(r.Op)
	switch left := r.Comparable.(type) {
	case *ast.Member:
		field, err := c.resolveField(left, op == OpHas)
		if err != nil {
			return nil, err
		}

		return c.fieldComparison(field, op, r)
	case *ast.Function:
		call, err := c.resolveCall(left)
		if err != nil {
			return nil, err
		}

		if op == OpHas {
			return nil, c.errorf(
				r.OpPos,
				"the has operator (:) is not supported on function results",
			)
		}

		if isOrdering(op) && !orderable(call.Result.Kind) {
			return nil, c.errorf(
				r.OpPos,
				"%v results do not support ordering comparisons",
				call.Result.Kind,
			)
		}

		right, err := c.resolveValue(call.Result, op, r.Arg)
		if err != nil {
			return nil, err
		}

		return &Comparison{Left: call, Op: op, Right: right}, nil
	default:
		return nil, fmt.Errorf("filtering: unhandled comparable node %T", left)
	}
}

// global checks a restriction with no comparator. A bare member is a search
// term, even when its text names a declared field: AIP-160 defines no
// boolean shorthand, so `published` searches for the word while
// `published = true` tests the field. A bare call to a bool-returning
// function is a condition.
func (c *checker) global(operand ast.Comparable) (Expr, error) {
	switch g := operand.(type) {
	case *ast.Member:
		return &Search{Terms: []string{g.Text()}}, nil
	case *ast.Function:
		decl, err := c.lookupFunc(g)
		if err != nil {
			return nil, err
		}

		if decl.expand != nil {
			return c.expandCall(g, decl)
		}

		if decl.result.Kind != KindBool {
			return nil, c.errorf(
				g.Pos(),
				"function %q returns %v and cannot be used as a condition",
				decl.name,
				decl.result.Kind,
			)
		}

		call, err := c.passthroughCall(g, decl)
		if err != nil {
			return nil, err
		}

		return &Comparison{Left: call, Op: OpEquals, Right: BoolValue(true)}, nil
	default:
		return nil, fmt.Errorf("filtering: unhandled comparable node %T", g)
	}
}

// resolveField resolves a member as a field path against the schema. has
// reports whether the path is used with the has operator, which is the only
// operator allowed to traverse into repeated fields.
func (c *checker) resolveField(m *ast.Member, has bool) (*Field, error) {
	root, ok := c.schema.fields[m.Value.Text]
	if !ok {
		return nil, c.errorf(m.Pos(), "unknown filter field %q", m.Text())
	}

	field := &Field{Segments: make([]FieldSegment, 0, 1+len(m.Fields))}
	field.Segments = append(field.Segments, FieldSegment{Name: m.Value.Text, Type: root})

	t := root
	prev := m.Value
	crossedRepeated := false

	for _, sv := range m.Fields {
		switch t.Kind {
		case KindMessage:
			sub, ok := t.msg.fields[sv.Text]
			if !ok {
				return nil, c.errorf(sv.Pos(), "unknown field %q in %q", sv.Text, field.Path())
			}

			t = sub
		case KindMap:
			t = *t.Elem
		case KindRepeated:
			if !has {
				return nil, c.errorf(
					prev.Pos(),
					"cannot traverse repeated field %q without the has operator (:)",
					field.Path(),
				)
			}

			if crossedRepeated {
				return nil, c.errorf(
					prev.Pos(),
					"cannot traverse more than one repeated field in %q",
					field.Path(),
				)
			}

			crossedRepeated = true

			elem := *t.Elem
			if elem.Kind != KindMessage {
				return nil, c.errorf(
					sv.Pos(),
					"repeated field %q has %v elements, not messages",
					field.Path(),
					elem.Kind,
				)
			}

			sub, ok := elem.msg.fields[sv.Text]
			if !ok {
				return nil, c.errorf(sv.Pos(), "unknown field %q in %q", sv.Text, field.Path())
			}

			t = sub
		default:
			return nil, c.errorf(
				sv.Pos(),
				"field %q of type %v has no subfields",
				field.Path(),
				t.Kind,
			)
		}

		field.Segments = append(field.Segments, FieldSegment{Name: sv.Text, Type: t})
		prev = sv
	}

	return field, nil
}

// fieldComparison checks a comparison whose left-hand side is a field.
func (c *checker) fieldComparison(field *Field, op Operator, r *ast.Restriction) (Expr, error) {
	t := field.Type()
	if op == OpHas {
		return c.hasComparison(field, r)
	}

	switch t.Kind {
	case KindRepeated:
		return nil, c.errorf(
			r.OpPos,
			"repeated field %q supports only the has operator (:)",
			field.Path(),
		)
	case KindMap:
		return nil, c.errorf(
			r.OpPos,
			"map field %q must be accessed by key, like %s.key",
			field.Path(),
			field.Path(),
		)
	default:
		// Scalar and message kinds continue below.
	}

	if isOrdering(op) && !orderable(t.Kind) {
		return nil, c.errorf(
			r.OpPos,
			"field %q of type %v does not support ordering comparisons",
			field.Path(),
			t.Kind,
		)
	}

	right, err := c.resolveValue(t, op, r.Arg)
	if err != nil {
		return nil, err
	}

	return &Comparison{Left: field, Op: op, Right: right}, nil
}

// hasComparison checks a has restriction. The has operator tests
// containment on repeated fields, key presence or key value on maps, field
// presence on messages, and, with the * argument, presence of any field.
// Map keys and message fields named on the right-hand side are normalized
// into the field path: `labels:env` becomes the path labels.env with a
// presence test.
func (c *checker) hasComparison(field *Field, r *ast.Restriction) (Expr, error) {
	lit, err := c.argLiteral(r.Arg)
	if err != nil {
		return nil, err
	}

	t := field.Type()

	star := !lit.quoted && lit.text == "*"
	if star {
		return &Comparison{Left: field, Op: OpHas, Right: StarValue()}, nil
	}

	switch t.Kind {
	case KindRepeated:
		elem := *t.Elem
		if elem.Kind == KindMessage {
			return nil, c.errorf(
				lit.pos,
				"repeated message field %q requires a subfield, like %s.field:value",
				field.Path(),
				field.Path(),
			)
		}

		right, err := c.literalValue(elem, OpHas, lit)
		if err != nil {
			return nil, err
		}

		return &Comparison{Left: field, Op: OpHas, Right: right}, nil
	case KindMap:
		extended := extend(field, FieldSegment{Name: lit.text, Type: *t.Elem})

		return &Comparison{Left: extended, Op: OpHas, Right: StarValue()}, nil
	case KindMessage:
		sub, ok := t.msg.fields[lit.text]
		if !ok {
			return nil, c.errorf(lit.pos, "unknown field %q in %q", lit.text, field.Path())
		}

		extended := extend(field, FieldSegment{Name: lit.text, Type: sub})

		return &Comparison{Left: extended, Op: OpHas, Right: StarValue()}, nil
	default:
		if hasCollectionAncestor(field) {
			right, err := c.literalValue(t, OpHas, lit)
			if err != nil {
				return nil, err
			}

			return &Comparison{Left: field, Op: OpHas, Right: right}, nil
		}

		return nil, c.errorf(
			lit.pos,
			"field %q of type %v supports only the presence test :*",
			field.Path(),
			t.Kind,
		)
	}
}

// extend returns a copy of field with one more segment.
func extend(field *Field, segment FieldSegment) *Field {
	segments := make([]FieldSegment, 0, len(field.Segments)+1)
	segments = append(segments, field.Segments...)
	segments = append(segments, segment)

	return &Field{Segments: segments}
}

// hasCollectionAncestor reports whether any non-final segment of the path
// is a repeated or map field, which makes a has restriction on the final
// scalar a containment or map-value test rather than nonsense.
func hasCollectionAncestor(field *Field) bool {
	for _, s := range field.Segments[:len(field.Segments)-1] {
		if s.Type.Kind == KindRepeated || s.Type.Kind == KindMap {
			return true
		}
	}

	return false
}

// lookupFunc finds the declaration of a called function.
func (c *checker) lookupFunc(f *ast.Function) (*funcDecl, error) {
	name := f.Text()

	decl, ok := c.schema.funcs[name]
	if !ok {
		return nil, c.errorf(f.Pos(), "unknown filter function %q", name)
	}

	return decl, nil
}

// resolveCall checks a pass-through function call used inside a
// comparison. Macro functions expand to conditions and cannot appear
// there.
func (c *checker) resolveCall(f *ast.Function) (*FuncCall, error) {
	decl, err := c.lookupFunc(f)
	if err != nil {
		return nil, err
	}

	if decl.expand != nil {
		return nil, c.errorf(
			f.Pos(),
			"function %q expands to a condition and cannot be used in a comparison",
			decl.name,
		)
	}

	return c.passthroughCall(f, decl)
}

// passthroughCall resolves the arguments of a non-macro call.
func (c *checker) passthroughCall(f *ast.Function, decl *funcDecl) (*FuncCall, error) {
	args, _, err := c.callArgs(f, decl)
	if err != nil {
		return nil, err
	}

	return &FuncCall{Name: decl.name, Args: args, Result: decl.result}, nil
}

// expandCall resolves a macro call's arguments and rewrites the call using
// its expander. Expander errors are wrapped unchanged: an expander reports
// an invalid filter by returning a [*CheckError] itself.
func (c *checker) expandCall(f *ast.Function, decl *funcDecl) (Expr, error) {
	args, positions, err := c.callArgs(f, decl)
	if err != nil {
		return nil, err
	}

	values := make([]Value, len(args))
	for i, a := range args {
		v, ok := a.(Value)
		if !ok {
			return nil, c.errorf(
				positions[i],
				"argument %d of %q must be a literal value",
				i+1,
				decl.name,
			)
		}

		values[i] = v
	}

	x, err := decl.expand(c.schema, values)
	if err != nil {
		return nil, fmt.Errorf("expanding filter function %q: %w", decl.name, err)
	}

	if x == nil {
		return nil, fmt.Errorf("expanding filter function %q: expander returned nil", decl.name)
	}

	return x, nil
}

// callArgs resolves a call's arguments against the declared parameter
// kinds. Each argument may be a field reference of the declared kind, a
// literal of the declared kind, or a nested pass-through call returning
// it. The returned positions parallel the arguments, for error reporting.
func (c *checker) callArgs(f *ast.Function, decl *funcDecl) ([]FuncArg, []int, error) {
	if len(f.Args) != len(decl.args) {
		return nil, nil, c.errorf(
			f.Pos(),
			"function %q takes %d arguments, got %d",
			decl.name,
			len(decl.args),
			len(f.Args),
		)
	}

	if len(f.Args) == 0 {
		return nil, nil, nil
	}

	args := make([]FuncArg, 0, len(f.Args))

	positions := make([]int, 0, len(f.Args))
	for i, a := range f.Args {
		want := decl.args[i]

		var arg FuncArg

		switch a := a.(type) {
		case *ast.Member:
			if field, err := c.resolveField(a, false); err == nil {
				if field.Type().Kind != want {
					return nil, nil, c.errorf(
						a.Pos(),
						"argument %d of %q must be %v, field %q is %v",
						i+1,
						decl.name,
						want,
						field.Path(),
						field.Type().Kind,
					)
				}

				arg = field

				break
			}

			lit, err := c.argLiteral(a)
			if err != nil {
				return nil, nil, err
			}

			v, err := c.literalValue(Type{Kind: want}, OpEquals, lit)
			if err != nil {
				return nil, nil, err
			}

			arg = v
		case *ast.Function:
			nested, err := c.resolveCall(a)
			if err != nil {
				return nil, nil, err
			}

			if nested.Result.Kind != want {
				return nil, nil, c.errorf(a.Pos(), "argument %d of %q must be %v, %q returns %v",
					i+1, decl.name, want, nested.Name, nested.Result.Kind)
			}

			arg = nested
		default:
			return nil, nil, c.errorf(a.Pos(), "parenthesized function arguments are not supported")
		}

		args = append(args, arg)
		positions = append(positions, a.Pos())
	}

	return args, positions, nil
}

// literal is the raw text of a comparison argument.
type literal struct {
	text   string
	quoted bool
	pos    int
}

// argLiteral extracts the literal text of a comparison argument. Per AIP
// filtering semantics the right-hand side of a comparator accepts only
// literals: field references there are undistinguishable from strings and
// are treated as text.
func (c *checker) argLiteral(arg ast.Arg) (literal, error) {
	switch a := arg.(type) {
	case *ast.Member:
		return literal{
			text:   a.Text(),
			quoted: len(a.Fields) == 0 && a.Value.Quoted,
			pos:    a.Pos(),
		}, nil
	case *ast.Function:
		return literal{}, c.errorf(a.Pos(), "functions are not allowed as comparison arguments")
	case *ast.Composite:
		return literal{}, c.errorf(a.Pos(), "parenthesized comparison arguments are not supported")
	default:
		return literal{}, fmt.Errorf("filtering: unhandled argument node %T", a)
	}
}

// resolveValue resolves a comparison argument against the expected type.
func (c *checker) resolveValue(t Type, op Operator, arg ast.Arg) (Value, error) {
	lit, err := c.argLiteral(arg)
	if err != nil {
		return Value{}, err
	}

	return c.literalValue(t, op, lit)
}

// literalValue converts literal text to a typed value of kind t.Kind.
func (c *checker) literalValue(t Type, op Operator, lit literal) (Value, error) {
	if !lit.quoted && lit.text == "null" {
		return c.nullLiteral(t, op, lit)
	}

	switch t.Kind {
	case KindString:
		if strings.ContainsRune(lit.text, '*') {
			if isOrdering(op) {
				return Value{}, c.errorf(
					lit.pos,
					"wildcard patterns support only = and != comparisons",
				)
			}

			return patternValue(lit.text), nil
		}

		return StringValue(lit.text), nil
	case KindInt:
		n, err := strconv.ParseInt(lit.text, 10, 64)
		if err != nil {
			return Value{}, c.errorf(lit.pos, "expected an integer, got %q", lit.text)
		}

		return IntValue(n), nil
	case KindFloat:
		f, err := strconv.ParseFloat(lit.text, 64)
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return Value{}, c.errorf(lit.pos, "expected a number, got %q", lit.text)
		}

		return FloatValue(f), nil
	case KindBool:
		if !lit.quoted {
			switch lit.text {
			case "true":
				return BoolValue(true), nil
			case "false":
				return BoolValue(false), nil
			}
		}

		return Value{}, c.errorf(lit.pos, "expected true or false, got %q", lit.text)
	case KindEnum:
		if slices.Contains(t.Enum, lit.text) {
			return EnumValue(lit.text), nil
		}

		return Value{}, c.errorf(
			lit.pos,
			"invalid value %q; valid values are %s",
			lit.text,
			strings.Join(t.Enum, ", "),
		)
	case KindTimestamp:
		return c.timestampLiteral(lit)
	case KindDuration:
		return c.durationLiteral(lit)
	case KindMessage:
		return Value{}, c.errorf(
			lit.pos,
			"message fields can only be compared to null or tested with the has operator (:)",
		)
	default:
		return Value{}, fmt.Errorf("filtering: unhandled value kind %v", t.Kind)
	}
}

// nullLiteral checks a null literal: valid only against message-backed
// fields (messages, timestamps, durations) and only with = or !=.
func (c *checker) nullLiteral(t Type, op Operator, lit literal) (Value, error) {
	switch {
	case t.Kind != KindMessage && t.Kind != KindTimestamp && t.Kind != KindDuration:
		return Value{}, c.errorf(
			lit.pos,
			"null is not a valid %v value; quote it to match the text \"null\"",
			t.Kind,
		)
	case op != OpEquals && op != OpNotEquals:
		return Value{}, c.errorf(lit.pos, "null supports only = and !=")
	}

	return NullValue(), nil
}

// timestampLiteral parses an RFC 3339 timestamp literal, the only timestamp
// form AIP-160 admits.
func (c *checker) timestampLiteral(lit literal) (Value, error) {
	if ts, err := time.Parse(time.RFC3339, lit.text); err == nil {
		return TimestampValue(ts), nil
	}

	return Value{}, c.errorf(
		lit.pos,
		"invalid timestamp %q; use RFC 3339, like 2021-02-14T10:00:00Z",
		lit.text,
	)
}

// durationLiteral parses a seconds duration literal such as 20s or 1.5s.
func (c *checker) durationLiteral(lit literal) (Value, error) {
	if num, ok := strings.CutSuffix(lit.text, "s"); ok {
		secs, err := strconv.ParseFloat(num, 64)
		if err == nil && !math.IsInf(secs, 0) && !math.IsNaN(secs) {
			return DurationValue(time.Duration(secs * float64(time.Second))), nil
		}
	}

	return Value{}, c.errorf(
		lit.pos,
		"invalid duration %q; use seconds with an s suffix, like 20s or 1.5s",
		lit.text,
	)
}

// patternValue splits a wildcard string into pattern parts. Consecutive
// wildcards collapse into one.
func patternValue(s string) Value {
	var parts []PatternPart

	for {
		i := strings.IndexByte(s, '*')
		if i == -1 {
			if s != "" {
				parts = append(parts, PatternPart{Literal: s})
			}

			break
		}

		if i > 0 {
			parts = append(parts, PatternPart{Literal: s[:i]})
		}

		if len(parts) == 0 || !parts[len(parts)-1].Wildcard {
			parts = append(parts, PatternPart{Wildcard: true})
		}

		s = s[i+1:]
	}

	return Value{Kind: KindPattern, Pattern: parts}
}

// operatorOf maps a syntactic comparator to a checked [Operator].
func operatorOf(cmp ast.Comparator) Operator {
	switch cmp {
	case ast.ComparatorEquals:
		return OpEquals
	case ast.ComparatorNotEquals:
		return OpNotEquals
	case ast.ComparatorLess:
		return OpLess
	case ast.ComparatorLessEquals:
		return OpLessEquals
	case ast.ComparatorGreater:
		return OpGreater
	case ast.ComparatorGreaterEquals:
		return OpGreaterEquals
	case ast.ComparatorHas:
		return OpHas
	default:
		return 0
	}
}

// isOrdering reports whether op is one of <, <=, >, >=.
func isOrdering(op Operator) bool {
	switch op {
	case OpLess, OpLessEquals, OpGreater, OpGreaterEquals:
		return true
	default:
		return false
	}
}

// orderable reports whether values of kind k have a defined order.
func orderable(k Kind) bool {
	switch k {
	case KindString, KindInt, KindFloat, KindTimestamp, KindDuration:
		return true
	default:
		return false
	}
}
