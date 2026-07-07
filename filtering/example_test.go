package filtering_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/AndreiCocan/golang-aip/filtering"
)

func ExampleCompile() {
	schema := filtering.NewSchema(
		filtering.String("display_name"),
		filtering.Timestamp("create_time"),
	)

	checked, err := filtering.Compile(
		`display_name = "war*" AND create_time > "2021-02-14T10:00:00Z"`, schema)
	if err != nil {
		fmt.Println(err)

		return
	}

	and := checked.Expr.(*filtering.And)
	for _, operand := range and.Operands {
		comparison := operand.(*filtering.Comparison)
		field := comparison.Left.(*filtering.Field)
		fmt.Println(field.Path(), comparison.Op, comparison.Right.Kind)
	}
	// Output:
	// display_name = pattern
	// create_time > timestamp
}

func ExampleCompile_invalidFilter() {
	schema := filtering.NewSchema(filtering.String("display_name"))
	_, err := filtering.Compile(`unknown_field = "x"`, schema)
	fmt.Println(errors.Is(err, filtering.ErrInvalidFilter))
	fmt.Println(err)
	// Output:
	// true
	// invalid filter: unknown filter field "unknown_field" at position 0
}

func ExampleWalk() {
	schema := filtering.NewSchema(
		filtering.Bool("published"),
		filtering.Float("rating"),
	)

	checked, err := filtering.Compile("published = true AND (rating > 4.0 OR rating < 1.0)", schema)
	if err != nil {
		fmt.Println(err)

		return
	}

	comparisons := 0

	filtering.Walk(checked.Expr, func(e filtering.Expr) bool {
		if _, ok := e.(*filtering.Comparison); ok {
			comparisons++
		}

		return true
	})
	fmt.Println(comparisons)
	// Output:
	// 3
}

func ExampleFunction() {
	// A macro function: expanded at Check time, so every dialect
	// supports it without backend-specific code.
	schema := filtering.NewSchema(
		filtering.Timestamp("create_time"),
		filtering.Function(
			"createdBefore",
			filtering.Args(filtering.KindTimestamp),
			filtering.Expand(
				func(s *filtering.Schema, args []filtering.Value) (filtering.Expr, error) {
					field, err := s.Field("create_time")
					if err != nil {
						return nil, err
					}

					return &filtering.Comparison{
						Left:  field,
						Op:    filtering.OpLess,
						Right: args[0],
					}, nil
				},
			),
		),
	)

	checked, err := filtering.Compile(`createdBefore("2021-01-01T00:00:00Z")`, schema)
	if err != nil {
		fmt.Println(err)

		return
	}

	comparison := checked.Expr.(*filtering.Comparison)
	field := comparison.Left.(*filtering.Field)
	fmt.Println(field.Path(), comparison.Op, comparison.Right.Time.Format(time.RFC3339))
	// Output:
	// create_time < 2021-01-01T00:00:00Z
}
