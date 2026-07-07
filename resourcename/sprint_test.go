package resourcename

import "testing"

func TestSprint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pattern   string
		variables []string
		want      string
	}{
		{
			name:      "substitutes variables",
			pattern:   "publishers/{publisher}/books/{book}",
			variables: []string{"123", "45"},
			want:      "publishers/123/books/45",
		},
		{
			name:    "literal only pattern",
			pattern: "publishers",
			want:    "publishers",
		},
		{
			name:      "extra variables ignored",
			pattern:   "publishers/{publisher}",
			variables: []string{"123", "unused"},
			want:      "publishers/123",
		},
		{
			name:      "missing variable leaves empty segment",
			pattern:   "publishers/{publisher}/books/{book}",
			variables: []string{"123"},
			want:      "publishers/123/books/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := Sprint(tt.pattern, tt.variables...); got != tt.want {
				t.Errorf("Sprint(%q, %v) = %q, want %q", tt.pattern, tt.variables, got, tt.want)
			}
		})
	}
}
