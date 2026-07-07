package filtering

import "strings"

// Checked is a filter that [Check] validated and resolved against a
// [Schema]. It is the contract consumed by dialect packages: every literal
// is typed, every field path is resolved, and only the node types below can
// occur. A nil Expr means the filter was empty and matches everything.
type Checked struct {
	Expr Expr
}

// Expr is a node of a checked filter: [*And], [*Or], [*Not], [*Comparison],
// or [*Search]. The set is closed; dialects can type-switch exhaustively.
type Expr interface {
	isExpr()
}

// And is a conjunction of two or more operands. Conjunctions are flattened:
// both the AND keyword and whitespace-adjacent sequence factors produce a
// single And node, and adjacent global search terms are merged into one
// [*Search] operand.
type And struct {
	Operands []Expr
}

func (*And) isExpr() {}

// Or is a disjunction of two or more operands. In filters OR binds tighter
// than AND.
type Or struct {
	Operands []Expr
}

func (*Or) isExpr() {}

// Not negates its operand. Both `NOT a` and `-a` produce a Not node.
type Not struct {
	Operand Expr
}

func (*Not) isExpr() {}

// Operator is the comparator of a [Comparison].
type Operator int

const (
	// OpEquals is `=`. With a KindPattern value it is a wildcard match,
	// and with a KindNull value an unset test.
	OpEquals Operator = iota + 1
	// OpNotEquals is `!=`, the negation of OpEquals.
	OpNotEquals
	// OpLess is `<`.
	OpLess
	// OpLessEquals is `<=`.
	OpLessEquals
	// OpGreater is `>`.
	OpGreater
	// OpGreaterEquals is `>=`.
	OpGreaterEquals
	// OpHas is `:`. Against a repeated field it tests containment;
	// against a map or message field path it tests presence or value; and
	// with a KindStar value it tests presence of the field itself.
	OpHas
)

// String returns the operator as written in a filter.
func (o Operator) String() string {
	switch o {
	case OpEquals:
		return "="
	case OpNotEquals:
		return "!="
	case OpLess:
		return "<"
	case OpLessEquals:
		return "<="
	case OpGreater:
		return ">"
	case OpGreaterEquals:
		return ">="
	case OpHas:
		return ":"
	default:
		return "invalid"
	}
}

// Comparison relates a field or function call to a typed literal, such as
// `create_time > <timestamp>`. Bare boolean restrictions are normalized to
// comparisons: the filter `published` checks to `published = true`.
type Comparison struct {
	Left  Operand
	Op    Operator
	Right Value
}

func (*Comparison) isExpr() {}

// Operand is the left-hand side of a [Comparison]: a [*Field] or a
// [*FuncCall].
type Operand interface {
	isOperand()
}

// Field is a resolved field path. Each segment carries its type, so a
// dialect can see where a path crosses a message, map, or repeated
// boundary: `chapters.title` has a KindRepeated first segment and a
// KindString second one.
type Field struct {
	Segments []FieldSegment
}

// FieldSegment is one step of a field path. For a map access the Name is
// the map key and Type the map's value type.
type FieldSegment struct {
	Name string
	Type Type
}

// Path returns the dotted field path, such as "author.name".
func (f *Field) Path() string {
	names := make([]string, len(f.Segments))
	for i, s := range f.Segments {
		names[i] = s.Name
	}

	return strings.Join(names, ".")
}

// Type returns the type of the final segment: the type the field's values
// have.
func (f *Field) Type() Type {
	return f.Segments[len(f.Segments)-1].Type
}

func (*Field) isOperand() {}
func (*Field) isFuncArg() {}

// FuncCall is a call to a schema-declared function that has no expander:
// the dialect is expected to translate it natively, or reject it. Args
// holds resolved arguments; Result is the declared result type.
type FuncCall struct {
	Name   string
	Args   []FuncArg
	Result Type
}

func (*FuncCall) isOperand() {}
func (*FuncCall) isFuncArg() {}

// FuncArg is a resolved function argument: a [*Field], a literal [Value],
// or a nested [*FuncCall].
type FuncArg interface {
	isFuncArg()
}

// Search holds global restrictions: bare terms with no field and no
// comparator, such as the filter `Hugo 1862`. The service-wide semantics
// are a fuzzy match across the entry; dialects without a text-search
// capability should reject it.
type Search struct {
	Terms []string
}

func (*Search) isExpr() {}
