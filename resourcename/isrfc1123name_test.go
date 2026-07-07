package resourcename

import (
	"strings"
	"testing"
)

// hostname builds a valid RFC 1123 name of exactly n bytes out of 63-byte
// labels, so length-boundary cases can be expressed by number alone.
func hostname(n int) string {
	var b strings.Builder

	for b.Len() < n {
		if b.Len() > 0 {
			b.WriteByte('.')
		}

		rem := min(n-b.Len(), 63)

		b.WriteString(strings.Repeat("a", rem))
	}

	return b.String()
}

func TestIsRFC1123Name(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty", "", false},
		{"single label", "a", true},
		{"two labels", "example.com", true},
		{"three labels", "a.b.c", true},
		{"uppercase accepted", "A.COM", true},
		{"digit-leading label", "1a.example", true},
		{"all digits", "127.0.0.1", true},
		{"hyphen inside label", "a-b.example", true},

		{"trailing dot", "a.", true},
		{"trailing dot multi", "example.com.", true},
		{"lone dot", ".", false},
		{"leading dot", ".com", false},
		{"empty middle label", "a..b", false},

		{"label starts with hyphen", "-a.com", false},
		{"label ends with hyphen", "a-.com", false},
		{"underscore rejected", "a_b.com", false},
		{"space rejected", "a b.com", false},

		{"label of 63", strings.Repeat("a", 63), true},
		{"label of 64", strings.Repeat("a", 64), false},

		{"total 253", hostname(253), true},
		{"total 253 plus trailing dot", hostname(253) + ".", true},
		{"total 254 no trailing dot", hostname(254), false},
		{"total 255", hostname(255), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isRFC1123Name(tt.input); got != tt.want {
				t.Errorf("isRFC1123Name(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
