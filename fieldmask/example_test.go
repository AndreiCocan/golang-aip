package fieldmask_test

import (
	"fmt"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/AndreiCocan/golang-aip/fieldmask"
	"github.com/AndreiCocan/golang-aip/internal/testproto"
)

// An update handler. Normalize the mask, in case a REST client wrote the
// JSON names of the fields. Validate it with CheckUpdate before the
// database read, to fail fast on a bad path. Then merge the masked fields
// of the payload into the stored resource, and store the result. Errors
// are INVALID_ARGUMENT.
func Example() {
	mask := &fieldmaskpb.FieldMask{Paths: []string{"title", "pageCount"}}
	payload := &testproto.Book{Title: "The Go Programming Language, 2nd Edition", PageCount: 400}

	mask, err := fieldmask.Normalize(mask, &testproto.Book{})
	if err != nil {
		fmt.Println(err)

		return
	}

	if err := fieldmask.CheckUpdate(mask, &testproto.Book{}); err != nil {
		fmt.Println(err)

		return
	}

	stored := &testproto.Book{ // In a service: read it from the database.
		Name:      "shelves/1/books/1",
		Title:     "The Go Programming Language",
		Isbn:      "978-0134190440",
		PageCount: 380,
	}

	if err := fieldmask.Update(mask, stored, payload); err != nil {
		fmt.Println(err)

		return
	}

	// The fields outside the mask keep their stored values.
	fmt.Println(stored.GetTitle(), stored.GetPageCount(), stored.GetIsbn())
	// Output: The Go Programming Language, 2nd Edition 400 978-0134190440
}

// An update handler merges the masked fields of the request payload into
// the stored resource. Server-managed fields survive even when the payload
// tries to write them.
func ExampleUpdate() {
	stored := &testproto.Book{
		Name:       "shelves/1/books/1",
		Title:      "The Go Programming Language",
		CreateTime: "2026-01-01T00:00:00Z",
	}
	payload := &testproto.Book{
		Title:      "The Go Programming Language, 2nd Edition",
		CreateTime: "counterfeit",
	}
	mask := &fieldmaskpb.FieldMask{Paths: []string{"title", "create_time"}}

	if err := fieldmask.Update(mask, stored, payload); err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(stored.GetTitle())
	fmt.Println(stored.GetCreateTime())
	// Output:
	// The Go Programming Language, 2nd Edition
	// 2026-01-01T00:00:00Z
}

// A REST client can write the mask with the JSON names of the fields.
// Normalize converts them to the proto names that Prune and Update expect.
func ExampleNormalize() {
	mask := &fieldmaskpb.FieldMask{Paths: []string{"createTime", "labels.myKey"}}

	normalized, err := fieldmask.Normalize(mask, &testproto.Book{})
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(normalized.GetPaths())
	// Output: [create_time labels.myKey]
}

// A get handler strips the response down to the fields the read mask
// names.
func ExamplePrune() {
	book := &testproto.Book{
		Name:   "shelves/1/books/1",
		Title:  "The Go Programming Language",
		Author: &testproto.Author{Name: "Alan Donovan", Age: 50},
	}
	mask := &fieldmaskpb.FieldMask{Paths: []string{"title", "author.name"}}

	if err := fieldmask.Prune(mask, book); err != nil {
		fmt.Println(err)

		return
	}

	fmt.Printf("%q %q %d\n", book.GetTitle(), book.GetAuthor().GetName(), book.GetAuthor().GetAge())
	// Output: "The Go Programming Language" "Alan Donovan" 0
}

// The JSON has the fields that the read mask names, also those that have
// their default value, such as the zero page count.
func ExampleMarshalJSON() {
	book := &testproto.Book{
		Name:  "shelves/1/books/1",
		Title: "The Go Programming Language",
	}
	mask := &fieldmaskpb.FieldMask{Paths: []string{"title", "page_count"}}

	data, err := fieldmask.MarshalJSON(mask, book)
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(string(data))
	// Output: {"title":"The Go Programming Language","pageCount":0}
}
