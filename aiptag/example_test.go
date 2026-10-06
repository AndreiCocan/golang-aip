package aiptag_test

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/AndreiCocan/golang-aip/aiptag"
	"github.com/AndreiCocan/golang-aip/filtering"
	"github.com/AndreiCocan/golang-aip/ordering"
)

// Tag the domain type once. The filtering and ordering packages read the
// tags to build their schemas, so a field that a client can filter or sort
// by is declared in one place.
func Example() {
	type Book struct {
		// The resource name, such as "books/123". A search term can match it.
		Name string `aip:"name,filter,order,search"`
		// A filter and an order_by can name the title.
		Title string `aip:"title,filter,order"`
		// Only an order_by can name the create time.
		CreateTime time.Time `aip:"create_time,order"`
		// Without a tag, the API cannot name the field.
		Revision int64
	}

	filterSchema := filtering.SchemaFromTags(Book{})
	orderSchema := ordering.SchemaFromTags(Book{})

	_, err := filtering.Compile(`title = "War*"`, filterSchema)
	fmt.Println("filter by title:", err == nil)

	_, err = filtering.Compile(`create_time > "2021-01-01T00:00:00Z"`, filterSchema)
	fmt.Println("filter by create_time:", err == nil)

	_, err = ordering.Compile("create_time desc, title", orderSchema)
	fmt.Println("order by create_time and title:", err == nil)
	// Output:
	// filter by title: true
	// filter by create_time: false
	// order by create_time and title: true
}

func ExampleParse() {
	type Book struct {
		Name     string `aip:"name,filter,order,search"`
		Title    string `aip:"title,filter"`
		Revision int64
	}

	for f := range reflect.TypeFor[Book]().Fields() {
		tag, ok, err := aiptag.Parse(f)
		if err != nil {
			fmt.Println(err)

			return
		}

		if !ok {
			fmt.Println(f.Name, "has no aip tag")

			continue
		}

		fmt.Printf("%s: %+v\n", f.Name, tag)
	}
	// Output:
	// Name: {Name:name Filterable:true Orderable:true Searchable:true}
	// Title: {Name:title Filterable:true Orderable:false Searchable:false}
	// Revision has no aip tag
}

func ExampleParse_invalidTag() {
	type Book struct {
		Title string `aip:"title,sort"`
	}

	_, _, err := aiptag.Parse(reflect.TypeFor[Book]().Field(0))
	fmt.Println(errors.Is(err, aiptag.ErrInvalidTag))
	// Output:
	// true
}
