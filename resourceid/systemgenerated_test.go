package resourceid

import (
	"strings"
	"testing"
)

func TestNewSystemGenerated(t *testing.T) {
	t.Parallel()

	id := NewSystemGenerated()

	if len(id) != 36 {
		t.Fatalf("NewSystemGenerated() = %q, want 36 characters, got %d", id, len(id))
	}

	for _, position := range []int{8, 13, 18, 23} {
		if id[position] != '-' {
			t.Errorf("NewSystemGenerated() = %q, want %q at position %d", id, '-', position)
		}
	}

	for position, character := range id {
		switch position {
		case 8, 13, 18, 23:
		default:
			if !strings.ContainsRune("0123456789abcdef", character) {
				t.Errorf(
					"NewSystemGenerated() = %q, want lowercase hex at position %d, got %q",
					id,
					position,
					character,
				)
			}
		}
	}

	if id[14] != '7' {
		t.Errorf(
			"NewSystemGenerated() = %q, want version nibble '7' at position 14, got %q",
			id,
			id[14],
		)
	}

	if !strings.ContainsRune("89ab", rune(id[19])) {
		t.Errorf(
			"NewSystemGenerated() = %q, want variant character in [89ab] at position 19, got %q",
			id,
			id[19],
		)
	}
}

func TestNewSystemGenerated_unique(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)

	for range 1000 {
		id := NewSystemGenerated()
		if seen[id] {
			t.Fatalf("NewSystemGenerated() returned duplicate id %q", id)
		}

		seen[id] = true
	}
}
