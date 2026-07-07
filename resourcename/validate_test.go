package resourcename

import (
	"errors"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid relative names.
		{"typical name", "publishers/123/books/les-miserables", false},
		{"collection and id", "publishers/123", false},
		{"single segment", "publishers", false},
		{"camel case segment", "bookShelves/big5", false},
		{"dotted segment", "peers/example.com", false},
		{"max length dns label", "publishers/" + strings.Repeat("a", 63), false},

		// Valid wildcard names.
		{"wildcard parent", "publishers/-/books/123", false},
		{"wildcard terminal", "publishers/123/books/-", false},

		// Valid full names.
		{"full name", "//library.example.com/publishers/123", false},
		{"full name host only", "//library.example.com", false},

		// Invalid: structure.
		{"empty", "", true},
		{"empty segment", "publishers//books", true},
		{"trailing slash", "publishers/123/", true},
		{"only slash", "/", true},

		// Invalid: variables in concrete names.
		{"variable segment", "publishers/{publisher}", true},

		// Invalid: character set.
		{"underscore in segment", "book_shelves/123", true},
		{"space", "publishers/1 3", true},
		{"non-ascii", "publishers/café", true},
		{"percent", "publishers/foo%2fbar", true},
		{"leading hyphen segment", "publishers/-abc", true},
		{"trailing hyphen segment", "publishers/abc-", true},
		{"too long dns label", "publishers/" + strings.Repeat("a", 64), true},

		// Invalid: full name host.
		{"full name bad host", "//not a host/publishers/123", true},
		{"full name empty host", "///publishers/123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, ErrInvalidName) {
				t.Fatalf(
					"Validate(%q) error = %v, want errors.Is(err, ErrInvalidName)",
					tt.input,
					err,
				)
			}
		})
	}
}
