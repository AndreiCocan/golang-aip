package resourcename

import (
	"errors"
	"testing"
)

func TestValidatePattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		// Valid patterns.
		{"top-level collection", "publishers/{publisher}", false},
		{"nested collections", "publishers/{publisher}/books/{book}", false},
		{"singleton suffix", "users/{user}/settings", false},
		{"literal only", "publishers", false},
		{"camel case collection", "bookShelves/{book_shelf}", false},
		{"variable with digits", "shards/{shard_1}", false},
		{"two letter variable", "as/{ab}", false},

		// Invalid: structure.
		{"empty", "", true},
		{"empty segment", "publishers//{book}", true},
		{"trailing slash", "publishers/{publisher}/", true},
		{"full pattern", "//library.example.com/publishers/{publisher}", true},
		{"wildcard segment", "publishers/-/books/{book}", true},

		// Invalid: variable format.
		{"uppercase variable", "publishers/{Publisher}", true},
		{"hyphenated variable", "publishers/{pub-lisher}", true},
		{"non-ascii variable", "publishers/{café}", true},
		{"single char variable", "publishers/{p}", true},
		{"leading underscore variable", "publishers/{_publisher}", true},
		{"trailing underscore variable", "publishers/{publisher_}", true},
		{"leading digit variable", "publishers/{1publisher}", true},
		{"id suffix variable", "publishers/{publisher_id}", true},
		{"empty braces", "publishers/{}", true},
		{"unclosed brace", "publishers/{publisher", true},

		// Invalid: duplicate variables.
		{"duplicate variables", "projects/{abc}/topics/{abc}", true},

		// Invalid: literal segments must be camelCase alphanumeric.
		{"hyphenated literal", "book-shelves/{book_shelf}", true},
		{"underscore literal", "book_shelves/{book_shelf}", true},
		{"dotted literal", "book.shelves/{book_shelf}", true},
		{"uppercase literal", "Publishers/{publisher}", true},
		{"leading digit literal", "1publishers/{publisher}", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidatePattern(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePattern(%q) error = %v, wantErr %v", tt.pattern, err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, ErrInvalidPattern) {
				t.Fatalf(
					"ValidatePattern(%q) error = %v, want errors.Is(err, ErrInvalidPattern)",
					tt.pattern,
					err,
				)
			}
		})
	}
}
