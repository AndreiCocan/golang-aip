package resourcename

import "testing"

func TestJoin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		elems []string
		want  string
	}{
		{"two relative names", []string{"publishers/1", "books/2"}, "publishers/1/books/2"},
		{"single name", []string{"publishers/1"}, "publishers/1"},
		{"no elements", nil, "/"},
		{"empty elements", []string{"", ""}, "/"},
		{
			"redundant slashes dropped",
			[]string{"publishers/1/", "/books/2"},
			"publishers/1/books/2",
		},
		{
			"full first element keeps host",
			[]string{"//library.example.com/publishers/1", "books/2"},
			"//library.example.com/publishers/1/books/2",
		},
		{
			"full non-first element loses host",
			[]string{"publishers/1", "//library.example.com/books/2"},
			"publishers/1/books/2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := Join(tt.elems...); got != tt.want {
				t.Errorf("Join(%v) = %q, want %q", tt.elems, got, tt.want)
			}
		})
	}
}
