package resourcename

import "testing"

func TestSscan(t *testing.T) {
	t.Parallel()

	t.Run("parses variables", func(t *testing.T) {
		t.Parallel()

		var publisher, book string

		err := Sscan(
			"publishers/123/books/45",
			"publishers/{publisher}/books/{book}",
			&publisher,
			&book,
		)
		if err != nil {
			t.Fatalf("Sscan() error = %v", err)
		}

		if publisher != "123" || book != "45" {
			t.Errorf("Sscan() parsed publisher=%q book=%q, want 123 and 45", publisher, book)
		}
	})

	t.Run("parses variables from full name", func(t *testing.T) {
		t.Parallel()

		var publisher string

		err := Sscan("//library.example.com/publishers/123", "publishers/{publisher}", &publisher)
		if err != nil {
			t.Fatalf("Sscan() error = %v", err)
		}

		if publisher != "123" {
			t.Errorf("Sscan() parsed publisher=%q, want 123", publisher)
		}
	})

	t.Run("literal only pattern", func(t *testing.T) {
		t.Parallel()

		if err := Sscan("publishers", "publishers"); err != nil {
			t.Fatalf("Sscan() error = %v", err)
		}
	})

	tests := []struct {
		name         string
		input        string
		pattern      string
		numVariables int
	}{
		{"literal mismatch", "shelves/1", "publishers/{publisher}", 1},
		{"name too short", "publishers/1", "publishers/{publisher}/books/{book}", 2},
		{"name too long", "publishers/1/books/2", "publishers/{publisher}", 1},
		{"too few variables", "publishers/1/books/2", "publishers/{publisher}/books/{book}", 1},
		{"too many variables", "publishers/1", "publishers/{publisher}", 2},
		{
			"full pattern",
			"//library.example.com/publishers/1",
			"//library.example.com/publishers/{publisher}",
			1,
		},
		{
			"braced name segment does not match literal",
			"{publishers}/1",
			"publishers/{publisher}",
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			variables := make([]*string, tt.numVariables)
			for i := range variables {
				variables[i] = new(string)
			}

			if err := Sscan(tt.input, tt.pattern, variables...); err == nil {
				t.Errorf("Sscan(%q, %q) error = nil, want non-nil", tt.input, tt.pattern)
			}
		})
	}

	t.Run("nil variable pointer", func(t *testing.T) {
		t.Parallel()

		if err := Sscan("publishers/1", "publishers/{publisher}", nil); err == nil {
			t.Error("Sscan() error = nil, want non-nil for nil variable pointer")
		}
	})
}
