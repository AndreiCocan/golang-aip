package fieldbehavior_test

import (
	"fmt"

	"google.golang.org/genproto/googleapis/api/annotations"

	"github.com/AndreiCocan/golang-aip/fieldbehavior"
	"github.com/AndreiCocan/golang-aip/internal/testproto"
)

// A create handler discards the server-managed fields a client has no
// business writing, without erroring.
func ExampleClear() {
	payload := &testproto.Book{
		Name:       "shelves/1/books/1",
		Title:      "The Go Programming Language",
		CreateTime: "2026-01-01T00:00:00Z",
	}
	fieldbehavior.Clear(payload,
		annotations.FieldBehavior_OUTPUT_ONLY,
		annotations.FieldBehavior_IDENTIFIER,
	)
	fmt.Println(payload.GetName(), payload.GetCreateTime(), payload.GetTitle())
	// Output:   The Go Programming Language
}

// A create handler rejects requests that omit required fields.
func ExampleValidateRequired() {
	req := &testproto.CreateBookRequest{
		Parent: "shelves/1",
		Book:   &testproto.Book{},
	}
	err := fieldbehavior.ValidateRequired(req)
	fmt.Println(err)
	// Output: missing required field: book.title
}
