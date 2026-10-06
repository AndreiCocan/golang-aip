package fieldbehavior_test

import (
	"errors"
	"fmt"

	"google.golang.org/genproto/googleapis/api/annotations"

	"github.com/AndreiCocan/golang-aip/fieldbehavior"
	"github.com/AndreiCocan/golang-aip/internal/testproto"
)

// A create handler checks the required fields first: a missing one is an
// INVALID_ARGUMENT error. Then it removes the fields that the server
// manages, such as the name and the create time, before it stores the
// resource.
func Example() {
	req := &testproto.CreateBookRequest{
		Parent: "shelves/1",
		Book: &testproto.Book{
			Name:       "shelves/1/books/mine", // The server chooses the name.
			Title:      "The Go Programming Language",
			CreateTime: "1999-01-01T00:00:00Z", // The server sets it.
		},
	}

	if err := fieldbehavior.ValidateRequired(req); err != nil {
		fmt.Println(err) // Return INVALID_ARGUMENT.

		return
	}

	book := req.GetBook()
	fieldbehavior.Clear(book,
		annotations.FieldBehavior_OUTPUT_ONLY,
		annotations.FieldBehavior_IDENTIFIER,
	)

	fmt.Printf("%q %q %q\n", book.GetName(), book.GetTitle(), book.GetCreateTime())
	// Output: "" "The Go Programming Language" ""
}

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

// A create handler lists every missing field, for example as one field
// violation per field in an INVALID_ARGUMENT response.
func ExampleRequiredFieldsError() {
	err := fieldbehavior.ValidateRequired(new(testproto.CreateBookRequest))

	if rf, ok := errors.AsType[*fieldbehavior.RequiredFieldsError](err); ok {
		fmt.Println(rf.Paths)
	}
	// Output: [parent book]
}
