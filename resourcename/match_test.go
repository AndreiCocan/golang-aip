package resourcename

import "testing"

func TestMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		pattern string
		want    bool
	}{
		{"variable match", "publishers/123", "publishers/{publisher}", true},
		{
			"nested variable match",
			"publishers/123/books/45",
			"publishers/{publisher}/books/{book}",
			true,
		},
		{"literal only match", "publishers", "publishers", true},
		{"wildcard satisfies variable", "publishers/-", "publishers/{publisher}", true},
		{
			"full name matches by path",
			"//library.example.com/publishers/123",
			"publishers/{publisher}",
			true,
		},

		{
			"literal mismatch",
			"publishers/123/shelves/45",
			"publishers/{publisher}/books/{book}",
			false,
		},
		{"name too short", "publishers/123", "publishers/{publisher}/books/{book}", false},
		{"name too long", "publishers/123/books/45", "publishers/{publisher}", false},
		{"empty name", "", "publishers/{publisher}", false},
		{"empty pattern", "publishers/123", "", false},
		{
			"empty segment does not satisfy variable",
			"publishers//books/1",
			"publishers/{publisher}/books/{book}",
			false,
		},
		{"variable in name", "publishers/{publisher}", "publishers/{publisher}", false},
		{"wildcard in pattern", "publishers/123", "publishers/-", false},
		{
			"full pattern",
			"//library.example.com/publishers/123",
			"//library.example.com/publishers/{publisher}",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := Match(tt.input, tt.pattern); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.input, tt.pattern, got, tt.want)
			}
		})
	}
}
