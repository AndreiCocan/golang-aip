package filtering

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLex(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		filter string
		want   []token
	}{
		{
			name:   "empty",
			filter: "",
			want: []token{
				{kind: tokenEOF, pos: 0, end: 0},
			},
		},
		{
			name:   "only whitespace",
			filter: " \t ",
			want: []token{
				{kind: tokenEOF, pos: 3, end: 3, spaceBefore: true},
			},
		},
		{
			name:   "single word",
			filter: "prod",
			want: []token{
				{kind: tokenText, pos: 0, end: 4, text: "prod"},
				{kind: tokenEOF, pos: 4, end: 4},
			},
		},
		{
			name:   "words separated by whitespace",
			filter: "New York",
			want: []token{
				{kind: tokenText, pos: 0, end: 3, text: "New"},
				{kind: tokenText, pos: 4, end: 8, text: "York", spaceBefore: true},
				{kind: tokenEOF, pos: 8, end: 8},
			},
		},
		{
			name:   "equals without spaces",
			filter: "a=1",
			want: []token{
				{kind: tokenText, pos: 0, end: 1, text: "a"},
				{kind: tokenEquals, pos: 1, end: 2},
				{kind: tokenText, pos: 2, end: 3, text: "1"},
				{kind: tokenEOF, pos: 3, end: 3},
			},
		},
		{
			name:   "all comparators",
			filter: "<= < >= > != = :",
			want: []token{
				{kind: tokenLessEquals, pos: 0, end: 2},
				{kind: tokenLess, pos: 3, end: 4, spaceBefore: true},
				{kind: tokenGreaterEquals, pos: 5, end: 7, spaceBefore: true},
				{kind: tokenGreater, pos: 8, end: 9, spaceBefore: true},
				{kind: tokenNotEquals, pos: 10, end: 12, spaceBefore: true},
				{kind: tokenEquals, pos: 13, end: 14, spaceBefore: true},
				{kind: tokenHas, pos: 15, end: 16, spaceBefore: true},
				{kind: tokenEOF, pos: 16, end: 16},
			},
		},
		{
			name:   "dotted member",
			filter: "a.b.c",
			want: []token{
				{kind: tokenText, pos: 0, end: 1, text: "a"},
				{kind: tokenDot, pos: 1, end: 2},
				{kind: tokenText, pos: 2, end: 3, text: "b"},
				{kind: tokenDot, pos: 3, end: 4},
				{kind: tokenText, pos: 4, end: 5, text: "c"},
				{kind: tokenEOF, pos: 5, end: 5},
			},
		},
		{
			name:   "decimal number splits on dot",
			filter: "2.5",
			want: []token{
				{kind: tokenText, pos: 0, end: 1, text: "2"},
				{kind: tokenDot, pos: 1, end: 2},
				{kind: tokenText, pos: 2, end: 3, text: "5"},
				{kind: tokenEOF, pos: 3, end: 3},
			},
		},
		{
			name:   "parens and comma",
			filter: "f(a, b)",
			want: []token{
				{kind: tokenText, pos: 0, end: 1, text: "f"},
				{kind: tokenLparen, pos: 1, end: 2},
				{kind: tokenText, pos: 2, end: 3, text: "a"},
				{kind: tokenComma, pos: 3, end: 4},
				{kind: tokenText, pos: 5, end: 6, text: "b", spaceBefore: true},
				{kind: tokenRparen, pos: 6, end: 7},
				{kind: tokenEOF, pos: 7, end: 7},
			},
		},
		{
			name:   "minus is its own token at word start",
			filter: "-file",
			want: []token{
				{kind: tokenMinus, pos: 0, end: 1},
				{kind: tokenText, pos: 1, end: 5, text: "file"},
				{kind: tokenEOF, pos: 5, end: 5},
			},
		},
		{
			name:   "minus inside a word stays in the word",
			filter: "2012-04-21",
			want: []token{
				{kind: tokenText, pos: 0, end: 10, text: "2012-04-21"},
				{kind: tokenEOF, pos: 10, end: 10},
			},
		},
		{
			name:   "colon splits words",
			filter: "map:key",
			want: []token{
				{kind: tokenText, pos: 0, end: 3, text: "map"},
				{kind: tokenHas, pos: 3, end: 4},
				{kind: tokenText, pos: 4, end: 7, text: "key"},
				{kind: tokenEOF, pos: 7, end: 7},
			},
		},
		{
			name:   "double-quoted string",
			filter: `a = "foo bar"`,
			want: []token{
				{kind: tokenText, pos: 0, end: 1, text: "a"},
				{kind: tokenEquals, pos: 2, end: 3, spaceBefore: true},
				{kind: tokenString, pos: 4, end: 13, text: "foo bar", spaceBefore: true},
				{kind: tokenEOF, pos: 13, end: 13},
			},
		},
		{
			name:   "single-quoted string may hold double quotes",
			filter: `msg != 'he said "hi"'`,
			want: []token{
				{kind: tokenText, pos: 0, end: 3, text: "msg"},
				{kind: tokenNotEquals, pos: 4, end: 6, spaceBefore: true},
				{kind: tokenString, pos: 7, end: 21, text: `he said "hi"`, spaceBefore: true},
				{kind: tokenEOF, pos: 21, end: 21},
			},
		},
		{
			name:   "empty string literal",
			filter: `a = ""`,
			want: []token{
				{kind: tokenText, pos: 0, end: 1, text: "a"},
				{kind: tokenEquals, pos: 2, end: 3, spaceBefore: true},
				{kind: tokenString, pos: 4, end: 6, text: "", spaceBefore: true},
				{kind: tokenEOF, pos: 6, end: 6},
			},
		},
		{
			name:   "string directly after word",
			filter: `file:".java"`,
			want: []token{
				{kind: tokenText, pos: 0, end: 4, text: "file"},
				{kind: tokenHas, pos: 4, end: 5},
				{kind: tokenString, pos: 5, end: 12, text: ".java"},
				{kind: tokenEOF, pos: 12, end: 12},
			},
		},
		{
			name:   "wildcard star is plain text",
			filter: "m.foo:*",
			want: []token{
				{kind: tokenText, pos: 0, end: 1, text: "m"},
				{kind: tokenDot, pos: 1, end: 2},
				{kind: tokenText, pos: 2, end: 5, text: "foo"},
				{kind: tokenHas, pos: 5, end: 6},
				{kind: tokenText, pos: 6, end: 7, text: "*"},
				{kind: tokenEOF, pos: 7, end: 7},
			},
		},
		{
			name:   "unicode text",
			filter: "name = héllo",
			want: []token{
				{kind: tokenText, pos: 0, end: 4, text: "name"},
				{kind: tokenEquals, pos: 5, end: 6, spaceBefore: true},
				{kind: tokenText, pos: 7, end: 13, text: "héllo", spaceBefore: true},
				{kind: tokenEOF, pos: 13, end: 13},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := lex(tt.filter)
			if err != nil {
				t.Fatalf("lex(%q) error = %v", tt.filter, err)
			}

			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(token{})); diff != "" {
				t.Errorf("lex(%q) mismatch (-want +got):\n%s", tt.filter, diff)
			}
		})
	}

	t.Run("errors", func(t *testing.T) {
		t.Parallel()

		for _, tt := range []struct {
			name    string
			filter  string
			wantPos int
		}{
			{name: "unterminated double-quoted string", filter: `a = "foo`, wantPos: 4},
			{name: "unterminated single-quoted string", filter: `a = 'foo"`, wantPos: 4},
			{name: "exclamation without equals", filter: "a ! b", wantPos: 2},
			{name: "exclamation at end", filter: "a!", wantPos: 1},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := lex(tt.filter)
				if err == nil {
					t.Fatalf("lex(%q) succeeded, want error", tt.filter)
				}

				if !errors.Is(err, ErrInvalidFilter) {
					t.Errorf("lex(%q) error = %v, want ErrInvalidFilter", tt.filter, err)
				}

				var perr *ParseError
				if !errors.As(err, &perr) {
					t.Fatalf("lex(%q) error = %T, want *ParseError", tt.filter, err)
				}

				if perr.Pos != tt.wantPos {
					t.Errorf(
						"lex(%q) error position = %d, want %d",
						tt.filter,
						perr.Pos,
						tt.wantPos,
					)
				}
			})
		}
	})
}
