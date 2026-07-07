package ast

// Filter is the root of a parsed filter. A filter may be empty, in which
// case Expr is nil and the filter matches everything.
type Filter struct {
	// Source is the filter string the tree was parsed from. Positions
	// throughout the tree are byte offsets into Source.
	Source string
	Expr   *Expression
}

// Expression is a conjunction: its sequences are joined by the AND keyword.
// An expression with a single sequence is just that sequence.
type Expression struct {
	Sequences []*Sequence
}

// Pos returns the byte offset of the expression's first token.
func (e *Expression) Pos() int { return e.Sequences[0].Pos() }

// Sequence is one or more whitespace-separated factors. A sequence is a
// conjunction like AND, but it additionally expresses that the factors
// belong together: fuzzy-match backends may rank results by factor
// proximity. A sequence with a single factor is just that factor.
type Sequence struct {
	Factors []*Factor
}

// Pos returns the byte offset of the sequence's first token.
func (s *Sequence) Pos() int { return s.Factors[0].Pos() }

// Factor is a disjunction: its terms are joined by the OR keyword. A factor
// with a single term is just that term.
type Factor struct {
	Terms []*Term
}

// Pos returns the byte offset of the factor's first token.
func (f *Factor) Pos() int { return f.Terms[0].Pos() }

// Negation is the way a [Term] is negated, if at all.
type Negation int

const (
	// NegationNone leaves the term as-is.
	NegationNone Negation = iota
	// NegationNot negates with the NOT keyword: `NOT a`.
	NegationNot
	// NegationMinus negates with a minus prefix: `-a`.
	NegationMinus
)

// Term is an optionally negated restriction or composite. The two negation
// styles NOT and - are interchangeable.
type Term struct {
	// NegPos is the byte offset of the negation token. Defined only when
	// Neg is not NegationNone.
	NegPos int
	Neg    Negation
	Simple Simple
}

// Pos returns the byte offset of the term's first token, including its
// negation prefix.
func (t *Term) Pos() int {
	if t.Neg != NegationNone {
		return t.NegPos
	}

	return t.Simple.Pos()
}

// Simple is a [*Restriction] or a [*Composite].
type Simple interface {
	Pos() int
	isSimple()
}

// Comparator is the relational operator of a [Restriction].
type Comparator int

const (
	// ComparatorNone marks a global restriction: a bare comparable with no
	// operator and no argument, such as `prod`.
	ComparatorNone Comparator = iota
	// ComparatorEquals is `=`.
	ComparatorEquals
	// ComparatorNotEquals is `!=`.
	ComparatorNotEquals
	// ComparatorLess is `<`.
	ComparatorLess
	// ComparatorLessEquals is `<=`.
	ComparatorLessEquals
	// ComparatorGreater is `>`.
	ComparatorGreater
	// ComparatorGreaterEquals is `>=`.
	ComparatorGreaterEquals
	// ComparatorHas is `:`, the presence/containment operator.
	ComparatorHas
)

// String returns the operator as written in a filter, or "" for
// ComparatorNone.
func (c Comparator) String() string {
	switch c {
	case ComparatorEquals:
		return "="
	case ComparatorNotEquals:
		return "!="
	case ComparatorLess:
		return "<"
	case ComparatorLessEquals:
		return "<="
	case ComparatorGreater:
		return ">"
	case ComparatorGreaterEquals:
		return ">="
	case ComparatorHas:
		return ":"
	default:
		return ""
	}
}

// Restriction relates a comparable to an argument: `create_time > "2021"`.
// When Op is ComparatorNone the restriction is global, a bare comparable
// such as `prod`, and Arg is nil.
type Restriction struct {
	Comparable Comparable
	// OpPos is the byte offset of the comparator token. Defined only when
	// Op is not ComparatorNone.
	OpPos int
	Op    Comparator
	Arg   Arg
}

// Pos returns the byte offset of the restriction's first token.
func (r *Restriction) Pos() int { return r.Comparable.Pos() }

func (*Restriction) isSimple() {}

// Composite is a parenthesized expression, used to group terms or override
// precedence.
type Composite struct {
	// Lparen and Rparen are the byte offsets of the parentheses.
	Lparen int
	Expr   *Expression
	Rparen int
}

// Pos returns the byte offset of the opening parenthesis.
func (c *Composite) Pos() int { return c.Lparen }

func (*Composite) isSimple() {}
func (*Composite) isArg()    {}

// Comparable is the left-hand side of a [Restriction]: a [*Member] or a
// [*Function].
type Comparable interface {
	Pos() int
	isComparable()
}

// Arg is the right-hand side of a [Restriction] or a function argument: a
// [*Member], a [*Function], or a [*Composite].
type Arg interface {
	Pos() int
	isArg()
}

// Value is one literal token: a bare TEXT word or a quoted STRING. In the
// filter grammar TEXT cannot contain whitespace or dots, so a decimal number
// like `2.5` arrives as a [Member] with two values joined by a dot.
type Value struct {
	// ValuePos is the byte offset of the token, including the opening quote
	// of a quoted string.
	ValuePos int
	// Text is the literal text. For a quoted string it is the content
	// between the quotes, which the grammar defines no escaping for.
	Text string
	// Quoted reports whether the value was written as a quoted string.
	// Quoting protects text from being interpreted as a keyword, a number,
	// or any other typed literal.
	Quoted bool
}

// Pos returns the byte offset of the value's token.
func (v *Value) Pos() int { return v.ValuePos }

// Member is a value optionally qualified by dotted fields: `a`, `a.b.c`, or
// `m."key.with.dots"`. Depending on schema context a member names a field
// path or spells a literal (`2.5` is the member `2` dot `5`).
type Member struct {
	Value  Value
	Fields []Value
}

// Pos returns the byte offset of the member's first token.
func (m *Member) Pos() int { return m.Value.ValuePos }

// Text returns the member as written, with segments joined by dots. Quoted
// segments are returned without their quotes.
func (m *Member) Text() string {
	if len(m.Fields) == 0 {
		return m.Value.Text
	}

	n := len(m.Value.Text)
	for _, f := range m.Fields {
		n += 1 + len(f.Text)
	}

	b := make([]byte, 0, n)

	b = append(b, m.Value.Text...)
	for _, f := range m.Fields {
		b = append(b, '.')
		b = append(b, f.Text...)
	}

	return string(b)
}

func (*Member) isComparable() {}
func (*Member) isArg()        {}

// Function is a call with a possibly dot-qualified name: `regex(m.key,
// '^.*prod.*$')` or `math.mem('30mb')`.
type Function struct {
	// Name holds the dotted name segments. The values are never quoted.
	Name []Value
	// Lparen and Rparen are the byte offsets of the parentheses.
	Lparen int
	Args   []Arg
	Rparen int
}

// Pos returns the byte offset of the function's first name segment.
func (f *Function) Pos() int { return f.Name[0].ValuePos }

// Text returns the function's dotted name.
func (f *Function) Text() string {
	m := Member{Value: f.Name[0], Fields: f.Name[1:]}

	return m.Text()
}

func (*Function) isComparable() {}
func (*Function) isArg()        {}
