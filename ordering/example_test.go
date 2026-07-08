package ordering_test

import (
	"errors"
	"fmt"

	"github.com/AndreiCocan/golang-aip/ordering"
)

func ExampleCompile() {
	schema := ordering.NewSchema("display_name", "create_time")

	checked, err := ordering.Compile("create_time desc, display_name", schema)
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, field := range checked.Fields {
		fmt.Println(field.Path(), field.Desc)
	}
	// Output:
	// create_time true
	// display_name false
}

func ExampleCompile_invalidOrderBy() {
	schema := ordering.NewSchema("display_name")
	_, err := ordering.Compile("unknown_field desc", schema)
	fmt.Println(errors.Is(err, ordering.ErrInvalidOrderBy))
	fmt.Println(err)
	// Output:
	// true
	// invalid order by: unknown ordering field "unknown_field" at position 0
}
