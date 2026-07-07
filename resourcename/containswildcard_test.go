package resourcename

import "testing"

func TestContainsWildcard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		{"publishers/-/books/1", true},
		{"publishers/1/books/-", true},
		{"-", true},
		{"publishers/123", false},
		{"books/a-b", false},
		{"", false},
		{"//library.example.com/publishers/-", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			if got := ContainsWildcard(tt.input); got != tt.want {
				t.Errorf("ContainsWildcard(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
