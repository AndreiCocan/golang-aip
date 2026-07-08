package ordering_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/AndreiCocan/golang-aip/ordering"
	"github.com/AndreiCocan/golang-aip/ordering/ast"
)

// Tree-building helpers. Positions are ignored by the bulk grammar tests
// (the positions subtest pins them separately), so the helpers leave them
// zero.

func asc(segments ...string) ast.Field  { return ast.Field{Segments: segments} }
func desc(segments ...string) ast.Field { return ast.Field{Segments: segments, Desc: true} }

func orderBy(fields ...ast.Field) *ast.OrderBy {
	return &ast.OrderBy{Fields: fields}
}

func TestParse(t *testing.T) {
	t.Parallel()

	ignorePositions := []cmp.Option{
		cmpopts.IgnoreFields(ast.OrderBy{}, "Source"),
		cmpopts.IgnoreFields(ast.Field{}, "Pos"),
	}

	tests := []struct {
		name    string
		orderBy string
		want    *ast.OrderBy
	}{
		{"empty", "", orderBy()},
		{"blank", " \t\r\n ", orderBy()},
		{"single field", "title", orderBy(asc("title"))},
		{"single field desc", "title desc", orderBy(desc("title"))},
		{"desc is case-insensitive upper", "title DESC", orderBy(desc("title"))},
		{"desc is case-insensitive mixed", "title Desc", orderBy(desc("title"))},
		{"two fields", "foo,bar", orderBy(asc("foo"), asc("bar"))},
		{"spec example", "foo, bar desc", orderBy(asc("foo"), desc("bar"))},
		{
			"spec whitespace equivalence",
			" foo , bar desc ",
			orderBy(asc("foo"), desc("bar")),
		},
		{"compact", "foo,bar desc", orderBy(asc("foo"), desc("bar"))},
		{"dotted path", "address.street", orderBy(asc("address", "street"))},
		{"deep dotted path", "a.b.c", orderBy(asc("a", "b", "c"))},
		{
			"dotted desc then plain",
			"author.name desc, title",
			orderBy(desc("author", "name"), asc("title")),
		},
		{"spaces around dots", "foo . bar", orderBy(asc("foo", "bar"))},
		{"desc is positional: bare field named desc", "desc", orderBy(asc("desc"))},
		{"desc field ordered descending", "desc desc", orderBy(desc("desc"))},
		{"asc is positional: bare field named asc", "asc", orderBy(asc("asc"))},
		{"redundant spaces before desc", "title    desc", orderBy(desc("title"))},
		{
			"tabs and newlines as whitespace",
			"foo\t,\nbar desc",
			orderBy(asc("foo"), desc("bar")),
		},
		{
			"duplicates preserved by parse",
			"title, title",
			orderBy(asc("title"), asc("title")),
		},
		{
			"contradictory duplicates preserved by parse",
			"title, title desc",
			orderBy(asc("title"), desc("title")),
		},
		{"permissive segment bytes", "foo-bar_baz*", orderBy(asc("foo-bar_baz*"))},
		{"non-ASCII segment", "café desc", orderBy(desc("café"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ordering.Parse(tt.orderBy)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.orderBy, err)
			}

			if diff := cmp.Diff(tt.want, got, ignorePositions...); diff != "" {
				t.Errorf("Parse(%q) tree mismatch (-want +got):\n%s", tt.orderBy, diff)
			}
		})
	}

	t.Run("positions", func(t *testing.T) {
		t.Parallel()

		got, err := ordering.Parse(" foo , bar.baz desc")
		if err != nil {
			t.Fatalf("Parse() error: %v", err)
		}

		want := &ast.OrderBy{
			Source: " foo , bar.baz desc",
			Fields: []ast.Field{
				{Pos: 1, Segments: []string{"foo"}},
				{Pos: 7, Segments: []string{"bar", "baz"}, Desc: true},
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Parse() tree mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			orderBy string
			wantPos int
			wantMsg string
		}{
			{"lone comma", ",", 0, `expected field, got ","`},
			{"leading comma", ",title", 0, `expected field, got ","`},
			{"trailing comma", "title,", 6, "expected field, got end of order by"},
			{"double comma", "title,,bar", 6, `expected field, got ","`},
			{"leading dot", ".title", 0, `expected field, got "."`},
			{
				"trailing dot",
				"title.",
				6,
				`expected field segment after ".", got end of order by`,
			},
			{"double dot", "a..b", 2, `expected field segment after ".", got "."`},
			{
				"asc suffix rejected",
				"title asc",
				6,
				`ascending is the default; only "desc" may follow a field`,
			},
			{
				"asc suffix rejected case-insensitively",
				"title ASC",
				6,
				`ascending is the default; only "desc" may follow a field`,
			},
			{
				"stray word after field",
				"title foo",
				6,
				`expected "desc", ",", or end of order by, got "foo"`,
			},
			{
				"stray word after desc",
				"title desc extra",
				11,
				`expected ",", got "extra"`,
			},
			{
				"double desc",
				"title desc desc",
				11,
				`expected ",", got "desc"`,
			},
			{"comparator character", "title=1", 5, `unexpected character '='`},
			{"quote character", `"title"`, 0, `unexpected character '"'`},
			{"parenthesis character", "min(title)", 3, `unexpected character '('`},
			{"control character", "a\x01b", 1, `unexpected character '\x01'`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := ordering.Parse(tt.orderBy)
				if err == nil {
					t.Fatalf("Parse(%q) succeeded, want error", tt.orderBy)
				}

				if !errors.Is(err, ordering.ErrInvalidOrderBy) {
					t.Errorf(
						"Parse(%q) error does not match ErrInvalidOrderBy: %v",
						tt.orderBy,
						err,
					)
				}

				var parseErr *ordering.ParseError
				if !errors.As(err, &parseErr) {
					t.Fatalf("Parse(%q) error is %T, want *ParseError", tt.orderBy, err)
				}

				if parseErr.OrderBy != tt.orderBy {
					t.Errorf("ParseError.OrderBy = %q, want %q", parseErr.OrderBy, tt.orderBy)
				}

				if parseErr.Pos != tt.wantPos {
					t.Errorf("ParseError.Pos = %d, want %d", parseErr.Pos, tt.wantPos)
				}

				if parseErr.Message != tt.wantMsg {
					t.Errorf("ParseError.Message = %q, want %q", parseErr.Message, tt.wantMsg)
				}
			})
		}
	})
}
