package filtering_test

import (
	"fmt"
	"testing"

	"github.com/AndreiCocan/golang-aip/filtering"
)

func nodeName(e filtering.Expr) string {
	switch e.(type) {
	case *filtering.And:
		return "and"
	case *filtering.Or:
		return "or"
	case *filtering.Not:
		return "not"
	case *filtering.Comparison:
		return "comparison"
	case *filtering.Search:
		return "search"
	default:
		return fmt.Sprintf("%T", e)
	}
}

func TestWalk(t *testing.T) {
	t.Parallel()

	checked, err := filtering.Compile(
		`published = true AND (rating > 4.0 OR NOT state = DELETED) war`, checkSchema)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	var order []string

	filtering.Walk(checked.Expr, func(e filtering.Expr) bool {
		order = append(order, nodeName(e))

		return true
	})

	want := []string{"and", "comparison", "or", "comparison", "not", "comparison", "search"}
	if fmt.Sprint(order) != fmt.Sprint(want) {
		t.Errorf("Walk() visit order = %v, want %v", order, want)
	}

	// Returning false prunes the subtree.
	order = nil

	filtering.Walk(checked.Expr, func(e filtering.Expr) bool {
		order = append(order, nodeName(e))

		return nodeName(e) != "or"
	})

	want = []string{"and", "comparison", "or", "search"}
	if fmt.Sprint(order) != fmt.Sprint(want) {
		t.Errorf("Walk() pruned visit order = %v, want %v", order, want)
	}

	t.Run("nil expr", func(t *testing.T) {
		t.Parallel()

		called := false

		filtering.Walk(nil, func(filtering.Expr) bool {
			called = true

			return true
		})

		if called {
			t.Error("Walk(nil) called f")
		}
	})
}
