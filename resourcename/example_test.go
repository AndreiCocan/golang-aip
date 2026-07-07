package resourcename_test

import (
	"fmt"

	"github.com/AndreiCocan/golang-aip/resourcename"
)

func ExampleValidate() {
	fmt.Println(resourcename.Validate("publishers/123/books/les-miserables"))
	fmt.Println(resourcename.Validate("publishers/123/books/"))
	// Output:
	// <nil>
	// invalid resource name: segment 4 is empty
}

func ExampleValidatePattern() {
	fmt.Println(resourcename.ValidatePattern("publishers/{publisher}/books/{book}"))
	fmt.Println(resourcename.ValidatePattern("publishers/{publisher_id}"))
	// Output:
	// <nil>
	// invalid resource name pattern: segment "{publisher_id}": variable must not use an _id suffix
}

func ExampleMatch() {
	fmt.Println(resourcename.Match("publishers/123", "publishers/{publisher}"))
	fmt.Println(resourcename.Match("shelves/123", "publishers/{publisher}"))
	// Output:
	// true
	// false
}

func ExampleSscan() {
	var publisher, book string
	if err := resourcename.Sscan(
		"publishers/123/books/les-miserables",
		"publishers/{publisher}/books/{book}",
		&publisher, &book,
	); err != nil {
		panic(err)
	}

	fmt.Println(publisher, book)
	// Output:
	// 123 les-miserables
}

func ExampleSprint() {
	fmt.Println(resourcename.Sprint("publishers/{publisher}/books/{book}", "123", "les-miserables"))
	// Output:
	// publishers/123/books/les-miserables
}

func ExampleJoin() {
	fmt.Println(resourcename.Join("publishers/123", "books/les-miserables"))
	// Output:
	// publishers/123/books/les-miserables
}

func ExampleHasParent() {
	fmt.Println(resourcename.HasParent("publishers/123/books/les-miserables", "publishers/123"))
	// Output:
	// true
}

func ExampleAncestor() {
	ancestor, ok := resourcename.Ancestor(
		"publishers/123/books/les-miserables",
		"publishers/{publisher}",
	)
	fmt.Println(ancestor, ok)
	// Output:
	// publishers/123 true
}

func ExampleParents() {
	for parent := range resourcename.Parents("publishers/123/books/les-miserables") {
		fmt.Println(parent)
	}
	// Output:
	// publishers
	// publishers/123
	// publishers/123/books
}

func ExampleContainsWildcard() {
	fmt.Println(resourcename.ContainsWildcard("publishers/-/books/123"))
	// Output:
	// true
}
