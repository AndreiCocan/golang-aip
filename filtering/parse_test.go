package filtering_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/AndreiCocan/golang-aip/filtering"
	"github.com/AndreiCocan/golang-aip/filtering/ast"
)

// Tree-building helpers. Positions are ignored by the bulk grammar tests
// (TestParsePositions pins them separately), so the helpers leave them zero.

func txt(s string) ast.Value { return ast.Value{Text: s} }
func str(s string) ast.Value { return ast.Value{Text: s, Quoted: true} }

func member(head ast.Value, fields ...ast.Value) *ast.Member {
	return &ast.Member{Value: head, Fields: fields}
}

func function(name []string, args ...ast.Arg) *ast.Function {
	f := &ast.Function{Args: args}
	for _, n := range name {
		f.Name = append(f.Name, txt(n))
	}

	return f
}

func global(c ast.Comparable) *ast.Restriction {
	return &ast.Restriction{Comparable: c}
}

func restrict(c ast.Comparable, op ast.Comparator, arg ast.Arg) *ast.Restriction {
	return &ast.Restriction{Comparable: c, Op: op, Arg: arg}
}

func term(s ast.Simple) *ast.Term { return &ast.Term{Simple: s} }

func notTerm(
	s ast.Simple,
) *ast.Term {
	return &ast.Term{Neg: ast.NegationNot, Simple: s}
}

func minusTerm(
	s ast.Simple,
) *ast.Term {
	return &ast.Term{Neg: ast.NegationMinus, Simple: s}
}
func factor(terms ...*ast.Term) *ast.Factor      { return &ast.Factor{Terms: terms} }
func seq(factors ...*ast.Factor) *ast.Sequence   { return &ast.Sequence{Factors: factors} }
func expr(seqs ...*ast.Sequence) *ast.Expression { return &ast.Expression{Sequences: seqs} }

// one wraps a single simple into the full single-term filter tree.
func one(s ast.Simple) *ast.Filter {
	return &ast.Filter{Expr: expr(seq(factor(term(s))))}
}

// oneTerm wraps a single term into the full filter tree.
func oneTerm(t *ast.Term) *ast.Filter {
	return &ast.Filter{Expr: expr(seq(factor(t)))}
}

func composite(e *ast.Expression) *ast.Composite { return &ast.Composite{Expr: e} }

func TestParse(t *testing.T) {
	t.Parallel()

	ignorePositions := []cmp.Option{
		cmpopts.IgnoreFields(ast.Filter{}, "Source"),
		cmpopts.IgnoreFields(ast.Value{}, "ValuePos"),
		cmpopts.IgnoreFields(ast.Term{}, "NegPos"),
		cmpopts.IgnoreFields(ast.Restriction{}, "OpPos"),
		cmpopts.IgnoreFields(ast.Composite{}, "Lparen", "Rparen"),
		cmpopts.IgnoreFields(ast.Function{}, "Lparen", "Rparen"),
	}

	for _, tt := range []struct {
		name   string
		filter string
		want   *ast.Filter
	}{
		{
			name:   "empty filter",
			filter: "",
			want:   &ast.Filter{},
		},
		{
			name:   "whitespace-only filter",
			filter: "  \t ",
			want:   &ast.Filter{},
		},
		{
			name:   "global restriction",
			filter: "prod",
			want:   one(global(member(txt("prod")))),
		},
		{
			name:   "quoted global restriction",
			filter: `"New York"`,
			want:   one(global(member(str("New York")))),
		},
		{
			name:   "global dotted member",
			filter: "expr.type_map.1.type",
			want:   one(global(member(txt("expr"), txt("type_map"), txt("1"), txt("type")))),
		},
		{
			name:   "quoted field segment",
			filter: `m."key.with.dots"`,
			want:   one(global(member(txt("m"), str("key.with.dots")))),
		},
		{
			name:   "keyword as field segment",
			filter: "f.AND",
			want:   one(global(member(txt("f"), txt("AND")))),
		},
		{
			name:   "equals",
			filter: "package=com.google",
			want: one(restrict(
				member(txt("package")),
				ast.ComparatorEquals,
				member(txt("com"), txt("google")),
			)),
		},
		{
			name:   "not equals with string",
			filter: "msg != 'hello'",
			want: one(restrict(
				member(txt("msg")),
				ast.ComparatorNotEquals,
				member(str("hello")),
			)),
		},
		{
			name:   "less than",
			filter: "a<3",
			want:   one(restrict(member(txt("a")), ast.ComparatorLess, member(txt("3")))),
		},
		{
			name:   "less or equal",
			filter: "a <= 4",
			want:   one(restrict(member(txt("a")), ast.ComparatorLessEquals, member(txt("4")))),
		},
		{
			name:   "greater than",
			filter: "1 > 0",
			want:   one(restrict(member(txt("1")), ast.ComparatorGreater, member(txt("0")))),
		},
		{
			name:   "greater or equal decimals",
			filter: "2.5 >= 2.4",
			want: one(restrict(
				member(txt("2"), txt("5")),
				ast.ComparatorGreaterEquals,
				member(txt("2"), txt("4")),
			)),
		},
		{
			name:   "has",
			filter: "map:key",
			want:   one(restrict(member(txt("map")), ast.ComparatorHas, member(txt("key")))),
		},
		{
			name:   "has star",
			filter: "m.foo:*",
			want: one(restrict(
				member(txt("m"), txt("foo")),
				ast.ComparatorHas,
				member(txt("*")),
			)),
		},
		{
			name:   "date text argument",
			filter: "create_time > 2021-02-14",
			want: one(restrict(
				member(txt("create_time")),
				ast.ComparatorGreater,
				member(txt("2021-02-14")),
			)),
		},
		{
			name:   "timestamp string argument",
			filter: `create_time > "2021-02-14T10:00:00Z"`,
			want: one(restrict(
				member(txt("create_time")),
				ast.ComparatorGreater,
				member(str("2021-02-14T10:00:00Z")),
			)),
		},
		{
			name:   "wildcard argument",
			filter: `a = "*.foo"`,
			want:   one(restrict(member(txt("a")), ast.ComparatorEquals, member(str("*.foo")))),
		},
		{
			name:   "negative integer argument",
			filter: "a = -30",
			want:   one(restrict(member(txt("a")), ast.ComparatorEquals, member(txt("-30")))),
		},
		{
			name:   "negative decimal argument",
			filter: "a = -2.5",
			want: one(restrict(
				member(txt("a")),
				ast.ComparatorEquals,
				member(txt("-2"), txt("5")),
			)),
		},
		{
			name:   "sequence",
			filter: "New York Giants",
			want: &ast.Filter{Expr: expr(seq(
				factor(term(global(member(txt("New"))))),
				factor(term(global(member(txt("York"))))),
				factor(term(global(member(txt("Giants"))))),
			))},
		},
		{
			name:   "and of sequences",
			filter: "a b AND c",
			want: &ast.Filter{Expr: expr(
				seq(
					factor(term(global(member(txt("a"))))),
					factor(term(global(member(txt("b"))))),
				),
				seq(factor(term(global(member(txt("c")))))),
			)},
		},
		{
			name:   "or of terms",
			filter: "a < 10 OR a >= 100",
			want: &ast.Filter{Expr: expr(seq(factor(
				term(restrict(member(txt("a")), ast.ComparatorLess, member(txt("10")))),
				term(restrict(member(txt("a")), ast.ComparatorGreaterEquals, member(txt("100")))),
			)))},
		},
		{
			name:   "or binds tighter than and",
			filter: "a AND b OR c",
			want: &ast.Filter{Expr: expr(
				seq(factor(term(global(member(txt("a")))))),
				seq(factor(
					term(global(member(txt("b")))),
					term(global(member(txt("c")))),
				)),
			)},
		},
		{
			name:   "sequence with composite or",
			filter: "New York (Giants OR Yankees)",
			want: &ast.Filter{Expr: expr(seq(
				factor(term(global(member(txt("New"))))),
				factor(term(global(member(txt("York"))))),
				factor(term(composite(expr(seq(factor(
					term(global(member(txt("Giants")))),
					term(global(member(txt("Yankees")))),
				)))))),
			))},
		},
		{
			name:   "not composite",
			filter: "NOT (a OR b)",
			want: oneTerm(notTerm(composite(expr(seq(factor(
				term(global(member(txt("a")))),
				term(global(member(txt("b")))),
			)))))),
		},
		{
			name:   "minus restriction",
			filter: `-file:".java"`,
			want: oneTerm(minusTerm(restrict(
				member(txt("file")),
				ast.ComparatorHas,
				member(str(".java")),
			))),
		},
		{
			name:   "minus number is negation of a global restriction",
			filter: "-30",
			want:   oneTerm(minusTerm(global(member(txt("30"))))),
		},
		{
			name:   "parenthesized sequence and conjunction",
			filter: "(a b) AND c",
			want: &ast.Filter{Expr: expr(
				seq(factor(term(composite(expr(seq(
					factor(term(global(member(txt("a"))))),
					factor(term(global(member(txt("b"))))),
				)))))),
				seq(factor(term(global(member(txt("c")))))),
			)},
		},
		{
			name:   "nested parens",
			filter: "((a))",
			want:   one(composite(expr(seq(factor(term(composite(expr(seq(factor(term(global(member(txt("a")))))))))))))),
		},
		{
			name:   "function without arguments",
			filter: "recent()",
			want:   one(global(function([]string{"recent"}))),
		},
		{
			name:   "function with arguments",
			filter: "regex(m.key, '^.*prod.*$')",
			want: one(global(function(
				[]string{"regex"},
				member(txt("m"), txt("key")),
				member(str("^.*prod.*$")),
			))),
		},
		{
			name:   "qualified function name",
			filter: "math.mem('30mb')",
			want:   one(global(function([]string{"math", "mem"}, member(str("30mb"))))),
		},
		{
			name:   "function as argument",
			filter: "experiment.rollout <= cohort(request.user)",
			want: one(restrict(
				member(txt("experiment"), txt("rollout")),
				ast.ComparatorLessEquals,
				function([]string{"cohort"}, member(txt("request"), txt("user"))),
			)),
		},
		{
			name:   "nested function argument",
			filter: "f(a, g(b))",
			want: one(global(function(
				[]string{"f"},
				member(txt("a")),
				function([]string{"g"}, member(txt("b"))),
			))),
		},
		{
			name:   "composite function argument",
			filter: "f((a OR b))",
			want: one(global(function(
				[]string{"f"},
				composite(expr(seq(factor(
					term(global(member(txt("a")))),
					term(global(member(txt("b")))),
				)))),
			))),
		},
		{
			name:   "negative number as function argument",
			filter: "f(-30)",
			want:   one(global(function([]string{"f"}, member(txt("-30"))))),
		},
		{
			name:   "keyword-named function",
			filter: "a AND(b)",
			want: &ast.Filter{Expr: expr(seq(
				factor(term(global(member(txt("a"))))),
				factor(term(global(function([]string{"AND"}, member(txt("b")))))),
			))},
		},
		{
			name:   "NOT adjacent to paren is a function call",
			filter: "NOT(a)",
			want:   one(global(function([]string{"NOT"}, member(txt("a"))))),
		},
		{
			name:   "minus quoted string",
			filter: `-"foo"`,
			want:   oneTerm(minusTerm(global(member(str("foo"))))),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := filtering.Parse(tt.filter)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.filter, err)
			}

			if diff := cmp.Diff(tt.want, got, ignorePositions...); diff != "" {
				t.Errorf("Parse(%q) mismatch (-want +got):\n%s", tt.filter, diff)
			}
		})
	}

	t.Run("positions", func(t *testing.T) {
		t.Parallel()
		// NOT a.b != "x": every position pinned.
		//     0123456789...
		got, err := filtering.Parse(`NOT a.b != "x"`)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		want := &ast.Filter{
			Source: `NOT a.b != "x"`,
			Expr: &ast.Expression{Sequences: []*ast.Sequence{{
				Factors: []*ast.Factor{{Terms: []*ast.Term{{
					NegPos: 0,
					Neg:    ast.NegationNot,
					Simple: &ast.Restriction{
						Comparable: &ast.Member{
							Value:  ast.Value{ValuePos: 4, Text: "a"},
							Fields: []ast.Value{{ValuePos: 6, Text: "b"}},
						},
						OpPos: 8,
						Op:    ast.ComparatorNotEquals,
						Arg: &ast.Member{
							Value: ast.Value{ValuePos: 11, Text: "x", Quoted: true},
						},
					},
				}}}},
			}}},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Parse() position mismatch (-want +got):\n%s", diff)
		}

		// -f(x): negation, function parens.
		got, err = filtering.Parse("-f(x)")
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		want = &ast.Filter{Source: "-f(x)", Expr: &ast.Expression{Sequences: []*ast.Sequence{{
			Factors: []*ast.Factor{{Terms: []*ast.Term{{
				NegPos: 0,
				Neg:    ast.NegationMinus,
				Simple: &ast.Restriction{
					Comparable: &ast.Function{
						Name:   []ast.Value{{ValuePos: 1, Text: "f"}},
						Lparen: 2,
						Args:   []ast.Arg{&ast.Member{Value: ast.Value{ValuePos: 3, Text: "x"}}},
						Rparen: 4,
					},
				},
			}}}},
		}}}}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Parse() position mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()

		for _, tt := range []struct {
			name    string
			filter  string
			wantPos int
		}{
			{name: "missing argument", filter: "a =", wantPos: 3},
			{name: "leading comparator", filter: "= 1", wantPos: 0},
			{name: "unclosed paren", filter: "(a", wantPos: 2},
			{name: "trailing paren", filter: "a )", wantPos: 2},
			{name: "leading keyword", filter: "AND b", wantPos: 0},
			{name: "trailing keyword", filter: "a AND", wantPos: 5},
			{name: "and without leading space", filter: "(a)AND (b)", wantPos: 3},
			{name: "empty parens", filter: "recent ()", wantPos: 8},
			{name: "double dot", filter: "a..b", wantPos: 2},
			{name: "leading dot", filter: ".a", wantPos: 0},
			{name: "trailing dot", filter: "a.", wantPos: 2},
			{name: "minus before non-number argument", filter: "a = -foo", wantPos: 4},
			{name: "minus followed by space", filter: "- a", wantPos: 0},
			{name: "not without space is member error", filter: "NOT", wantPos: 0},
			{name: "trailing comma in arguments", filter: "f(a,)", wantPos: 4},
			{name: "missing factor whitespace", filter: "(a)(b)", wantPos: 3},
			{name: "unterminated string via parse", filter: `a = "foo`, wantPos: 4},
			{name: "two comparators", filter: "a = = b", wantPos: 4},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := filtering.Parse(tt.filter)
				if err == nil {
					t.Fatalf("Parse(%q) succeeded, want error", tt.filter)
				}

				if !errors.Is(err, filtering.ErrInvalidFilter) {
					t.Errorf("Parse(%q) error = %v, want ErrInvalidFilter", tt.filter, err)
				}

				var perr *filtering.ParseError
				if !errors.As(err, &perr) {
					t.Fatalf("Parse(%q) error = %T, want *ParseError", tt.filter, err)
				}

				if perr.Pos != tt.wantPos {
					t.Errorf(
						"Parse(%q) error = %q, position = %d, want %d",
						tt.filter,
						err,
						perr.Pos,
						tt.wantPos,
					)
				}

				if perr.Filter != tt.filter {
					t.Errorf("Parse(%q) error Filter = %q, want the input", tt.filter, perr.Filter)
				}
			})
		}
	})

	t.Run("nesting limit", func(t *testing.T) {
		t.Parallel()

		deep := strings.Repeat("(", 200) + "a" + strings.Repeat(")", 200)

		_, err := filtering.Parse(deep)
		if err == nil {
			t.Fatal("Parse() of 200-deep nesting succeeded, want error")
		}

		if !errors.Is(err, filtering.ErrInvalidFilter) {
			t.Errorf("Parse() error = %v, want ErrInvalidFilter", err)
		}

		ok := strings.Repeat("(", 20) + "a" + strings.Repeat(")", 20)
		if _, err := filtering.Parse(ok); err != nil {
			t.Errorf("Parse() of 20-deep nesting error = %v, want success", err)
		}
	})
}
