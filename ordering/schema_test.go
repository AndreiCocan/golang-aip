package ordering_test

import (
	"testing"

	"github.com/AndreiCocan/golang-aip/ordering"
)

func TestNewSchema(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		// Building a schema from distinct dotted paths must not panic.
		ordering.NewSchema("title", "create_time", "author.name", "a.b.c")
	})

	t.Run("panics", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			paths []string
			want  string
		}{
			{
				"empty path",
				[]string{""},
				"ordering: field declared with an empty name",
			},
			{
				"duplicate path",
				[]string{"title", "title"},
				`ordering: field "title" declared twice`,
			},
			{
				"empty leading segment",
				[]string{".title"},
				`ordering: field ".title" has an empty segment`,
			},
			{
				"empty trailing segment",
				[]string{"title."},
				`ordering: field "title." has an empty segment`,
			},
			{
				"empty middle segment",
				[]string{"a..b"},
				`ordering: field "a..b" has an empty segment`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				defer func() {
					got := recover()
					if got == nil {
						t.Fatalf("NewSchema(%q) did not panic", tt.paths)
					}

					if got != tt.want {
						t.Errorf("NewSchema(%q) panic = %v, want %q", tt.paths, got, tt.want)
					}
				}()

				ordering.NewSchema(tt.paths...)
			})
		}
	})
}
