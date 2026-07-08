package ast_test

import (
	"testing"

	"github.com/AndreiCocan/golang-aip/ordering/ast"
)

func TestField_Path(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		segments []string
		want     string
	}{
		{"single segment", []string{"title"}, "title"},
		{"dotted path", []string{"author", "name"}, "author.name"},
		{"deep path", []string{"a", "b", "c"}, "a.b.c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := &ast.Field{Segments: tt.segments}
			if got := f.Path(); got != tt.want {
				t.Errorf("Path() = %q, want %q", got, tt.want)
			}
		})
	}
}
