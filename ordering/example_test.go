package ordering_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/AndreiCocan/golang-aip/ordering"
)

// The usual service path. Build the schema once, from the aip tags of the
// domain type. Then compile the order_by of each List request, and give
// the checked order_by to a dialect, which sorts the query. An empty
// order_by gives no keys: use the default order of the service.
func Example() {
	type Book struct {
		Name       string    `aip:"name,order"`
		Title      string    `aip:"title,order"`
		CreateTime time.Time `aip:"create_time,order"`
	}

	schema := ordering.SchemaFromTags(Book{})

	checked, err := ordering.Compile("create_time desc, title", schema)
	if err != nil {
		fmt.Println(err) // See the invalid order_by example.

		return
	}

	for _, field := range checked.Keys {
		fmt.Println(field.Path(), field.Desc)
	}
	// Output:
	// create_time true
	// title false
}

// An order_by that is not valid gives an error that matches
// ErrInvalidOrderBy. The client sent it, so return INVALID_ARGUMENT with
// the message, which gives the position of the problem.
func ExampleCompile_invalidOrderBy() {
	schema := ordering.NewSchema("title", "create_time")

	for _, orderBy := range []string{
		"rating",                 // a field that is not in the schema
		"title, title desc",      // a field in two directions
		"title asc",              // ascending is the default, without a suffix
		"create_time desc desc,", // a syntax error
	} {
		_, err := ordering.Compile(orderBy, schema)
		fmt.Println(errors.Is(err, ordering.ErrInvalidOrderBy), err)
	}
	// Output:
	// true invalid order by: unknown ordering field "rating" at position 0
	// true invalid order by: field "title" is ordered both ascending and descending at position 7
	// true invalid order by: ascending is the default; only "desc" may follow a field at position 6
	// true invalid order by: expected ",", got "desc" at position 17
}

// NewSchema declares the orderable fields by their dotted paths, such as a
// field of a nested message. The suffix desc is not case-sensitive, and a
// field given twice in the same direction counts once.
func ExampleNewSchema() {
	schema := ordering.NewSchema("author.name", "create_time")

	checked, err := ordering.Compile("author.name DESC, create_time, author.name desc", schema)
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, field := range checked.Keys {
		fmt.Println(field.Segments, field.Desc)
	}
	// Output:
	// [author name] true
	// [create_time] false
}
