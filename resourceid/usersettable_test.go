package resourceid

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateUserSettable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		// Valid IDs.
		{"min length 4", "abcd", false},
		{"letters and digits", "ab12", false},
		{"letters digits hyphens", "les-miserables", false},
		{"hyphen separated digits", "a-1-2", false},
		{"max length 63", "a" + strings.Repeat("b", 61) + "c", false},
		{"max length 63 all letters", strings.Repeat("a", 63), false},
		{"lowercase hex uuid starting with letter", "deadbeef-0000-4000-8000-000000000000", false},

		// Invalid: length. AIP-133 recommends 4 to 63 characters.
		{"empty", "", true},
		{"single letter", "a", true},
		{"two letters", "ab", true},
		{"three letters", "abc", true},
		{"too long 64", strings.Repeat("a", 64), true},

		// Invalid: first character.
		{"starts with digit", "1abc", true},
		{"starts with hyphen", "-abc", true},
		{"uuid starting with digit", "0f47ac10-58cc-4372-a567-0e02b2c3d479", true},

		// Invalid: last character.
		{"ends with hyphen", "abcd-", true},
		{"min length ending with hyphen", "abc-", true},
		{"max length ending with hyphen", strings.Repeat("a", 62) + "-", true},

		// Invalid: character set.
		{"uppercase first", "Abcd", true},
		{"uppercase middle", "aBcd", true},
		{"uppercase uuid", "DEADBEEF-0000-4000-8000-000000000000", true},
		{"underscore", "ab_cd", true},
		{"space", "ab cd", true},
		{"dot", "ab.cd", true},
		{"slash", "ab/cd", true},
		{"non-ascii letter", "café", true},
		{"non-ascii symbol", "ab£cd", true},
		{"emoji", "ab😀cd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateUserSettable(tt.id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateUserSettable(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, ErrInvalid) {
				t.Fatalf(
					"ValidateUserSettable(%q) error = %v, want errors.Is(err, ErrInvalid)",
					tt.id,
					err,
				)
			}
		})
	}
}
