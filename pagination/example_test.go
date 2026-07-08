package pagination_test

import (
	"fmt"
	"slices"

	"github.com/AndreiCocan/golang-aip/pagination"
)

// Example pages through a List method's collection. The library only
// parses and mints tokens and resolves the page size; seeking to the
// cursor and fetching one row too many stay in the service.
func Example() {
	// The collection, already in the List method's order.
	books := []string{"1984", "Dune", "Emma", "Ivanhoe", "Ulysses"}

	// The cursor is the service's own bookmark, never seen by clients:
	// the ordering key of the last row a page served. The next page
	// resumes right after it. Any gob-encodable struct works.
	type cursor struct {
		LastTitle string
	}

	// listBooks is the service's List method.
	listBooks := func(pageSize int32, pageToken, filter string) ([]string, string, error) {
		// Decode the request's page_token. The empty token of a first
		// request is valid and carries no cursor. The filter is
		// checksummed into every token minted here, so a client cannot
		// reuse a token with a different filter: that fails with
		// ErrInvalidPageToken, as does a tampered token. Both mean
		// INVALID_ARGUMENT.
		token, err := pagination.Parse(pageToken, filter)
		if err != nil {
			return nil, "", err
		}

		// Resolve the request's page_size: unset means the default (2),
		// anything above the maximum (1000) is capped to it.
		size, err := pagination.PageSize(pageSize, 2, 1000)
		if err != nil {
			return nil, "", err
		}

		// Recover the cursor. ok is false on a first page, which starts
		// at the top of the collection; otherwise the page starts right
		// after the cursor's row.
		start := 0

		var cur cursor

		ok, err := token.Cursor(&cur)
		if err != nil {
			return nil, "", err
		}

		if ok {
			start = slices.Index(books, cur.LastTitle) + 1
		}

		// Fetch one row more than the page size. Getting it proves
		// another page exists; a short read means this page is the last
		// one and the next_page_token must be empty.
		page := books[start:min(start+int(size)+1, len(books))]
		if len(page) <= int(size) {
			return page, "", nil
		}

		// Drop the extra row and mint the next page's token from the
		// last row actually served.
		page = page[:size]

		next, err := token.Next(cursor{LastTitle: page[len(page)-1]})

		return page, next, err
	}

	// The client's side of the contract: send back the token unchanged,
	// stop when it comes back empty.
	pageToken := ""
	for {
		page, next, err := listBooks(0, pageToken, `author = "tolkien"`)
		if err != nil {
			fmt.Println(err)

			return
		}

		fmt.Println(page)

		if next == "" {
			break
		}

		pageToken = next
	}
	// Output:
	// [1984 Dune]
	// [Emma Ivanhoe]
	// [Ulysses]
}

func ExamplePageSize() {
	// The client left page_size unset.
	size, err := pagination.PageSize(0, 25, 1000)
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(size)

	// The client asked for more than the service allows.
	size, err = pagination.PageSize(5000, 25, 1000)
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(size)
	// Output:
	// 25
	// 1000
}
