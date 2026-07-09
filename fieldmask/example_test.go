package fieldmask_test

import (
	"fmt"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/AndreiCocan/golang-aip/fieldmask"
	"github.com/AndreiCocan/golang-aip/internal/testproto"
)

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
