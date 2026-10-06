package resourceid_test

import (
	"fmt"

	"github.com/AndreiCocan/golang-aip/resourceid"
	"github.com/AndreiCocan/golang-aip/resourcename"
)

// A create handler. When the client gives the ID of the new resource,
// validate it: a bad ID is an INVALID_ARGUMENT error. When the client
// gives no ID, generate one. Then build the name of the resource.
func Example() {
	const pattern = "publishers/{publisher}/books/{book}"

	// The client gave the ID.
	bookID := "les-miserables"
	if err := resourceid.ValidateUserSettable(bookID); err != nil {
		fmt.Println(err) // Return INVALID_ARGUMENT.

		return
	}

	fmt.Println(resourcename.Sprint(pattern, "123", bookID))

	// The client gave no ID, so the server generates a UUID.
	generated := resourcename.Sprint(pattern, "123", resourceid.Generate())
	fmt.Println(resourcename.Match(generated, pattern))
	// Output:
	// publishers/123/books/les-miserables
	// true
}

func ExampleValidateUserSettable() {
	fmt.Println(resourceid.ValidateUserSettable("les-miserables"))
	fmt.Println(resourceid.ValidateUserSettable("Les-Miserables"))
	// Output:
	// <nil>
	// invalid resource ID: must begin with a lowercase letter
}

func ExampleGenerate() {
	id := resourceid.Generate()
	fmt.Println(len(id))
	// Output:
	// 36
}
