package resourcename

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHasParent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		parent string
		want   bool
	}{
		{"direct parent", "publishers/1/books/2", "publishers/1", true},
		{"grandparent", "publishers/1/books/2", "publishers", true},
		{"wildcard parent", "publishers/1/books/2", "publishers/-", true},
		{"wildcard mid parent", "publishers/1/books/2", "publishers/-/books", true},
		{
			"full names same host",
			"//h.example.com/publishers/1/books/2",
			"//h.example.com/publishers/1",
			true,
		},

		{"self is not parent", "publishers/1/books/2", "publishers/1/books/2", false},
		{"sibling", "publishers/1", "publishers/2", false},
		{"parent deeper than name", "publishers/1", "publishers/1/books/2", false},
		{"empty name", "", "publishers/1", false},
		{"empty parent", "publishers/1/books/2", "", false},
		{
			"full name relative parent",
			"//h.example.com/publishers/1/books/2",
			"publishers/1",
			false,
		},
		{
			"relative name full parent",
			"publishers/1/books/2",
			"//h.example.com/publishers/1",
			false,
		},
		{
			"full names different hosts",
			"//a.example.com/publishers/1/books/2",
			"//b.example.com/publishers/1",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := HasParent(tt.input, tt.parent); got != tt.want {
				t.Errorf("HasParent(%q, %q) = %v, want %v", tt.input, tt.parent, got, tt.want)
			}
		})
	}
}

func TestAncestor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		pattern string
		want    string
		wantOK  bool
	}{
		{"proper prefix", "publishers/1/books/2", "publishers/{publisher}", "publishers/1", true},
		{
			"whole name",
			"publishers/1/books/2",
			"publishers/{publisher}/books/{book}",
			"publishers/1/books/2",
			true,
		},
		{
			"full name keeps host",
			"//h.example.com/publishers/1/books/2",
			"publishers/{publisher}",
			"//h.example.com/publishers/1",
			true,
		},

		{"literal mismatch", "shelves/1", "publishers/{publisher}", "", false},
		{
			"pattern deeper than name",
			"publishers/1",
			"publishers/{publisher}/books/{book}",
			"",
			false,
		},
		{"wildcard in pattern", "publishers/1", "publishers/-", "", false},
		{"empty name", "", "publishers/{publisher}", "", false},
		{"empty pattern", "publishers/1", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := Ancestor(tt.input, tt.pattern)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf(
					"Ancestor(%q, %q) = %q, %v, want %q, %v",
					tt.input,
					tt.pattern,
					got,
					ok,
					tt.want,
					tt.wantOK,
				)
			}
		})
	}
}

func TestParents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			"nested name",
			"publishers/1/books/2",
			[]string{"publishers", "publishers/1", "publishers/1/books"},
		},
		{"single segment", "publishers", nil},
		{"empty", "", nil},
		{"full name yields path prefixes", "//h.example.com/publishers/1", []string{"publishers"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := slices.Collect(Parents(tt.input))
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Parents(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestParents_earlyBreak(t *testing.T) {
	t.Parallel()

	var got []string
	for parent := range Parents("publishers/1/books/2") {
		got = append(got, parent)

		break
	}

	if diff := cmp.Diff([]string{"publishers"}, got); diff != "" {
		t.Errorf("Parents() with early break mismatch (-want +got):\n%s", diff)
	}
}
