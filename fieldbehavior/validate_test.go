package fieldbehavior_test

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/AndreiCocan/golang-aip/fieldbehavior"
	"github.com/AndreiCocan/golang-aip/internal/testproto"
)

func TestValidateRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		msg         proto.Message
		wantMissing string // "" means no error
	}{
		{
			name: "fully populated book is valid",
			msg:  fullBook(),
		},
		{
			name: "missing title",
			msg: func() *testproto.Book {
				b := fullBook()
				b.Title = ""

				return b
			}(),
			wantMissing: "title",
		},
		{
			name: "populated author with empty name",
			msg: func() *testproto.Book {
				b := fullBook()
				b.Author.Name = ""

				return b
			}(),
			wantMissing: "author.name",
		},
		{
			name: "unset optional author does not require its subfields",
			msg: func() *testproto.Book {
				b := fullBook()
				b.Author = nil

				return b
			}(),
		},
		{
			name: "repeated element missing its required field",
			msg: &testproto.Book{
				Title:           "kept",
				FeaturedReviews: []*testproto.Review{{Text: "great", Rating: 5}, {Rating: 2}},
			},
			wantMissing: "featured_reviews[1].text",
		},
		{
			name: "map value missing its required field",
			msg: &testproto.Book{
				Title:   "kept",
				Reviews: map[string]*testproto.Review{"smith": {Rating: 5}},
			},
			wantMissing: "reviews.smith.text",
		},
		{
			name:        "request message with missing resource",
			msg:         &testproto.CreateBookRequest{Parent: "shelves/1"},
			wantMissing: "book",
		},
		{
			name: "request message with invalid nested resource",
			msg: &testproto.CreateBookRequest{
				Parent: "shelves/1",
				Book:   &testproto.Book{},
			},
			wantMissing: "book.title",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := fieldbehavior.ValidateRequired(tt.msg)
			assertMissing(t, err, tt.wantMissing)
		})
	}
}

func TestValidateRequiredWithMask(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		msg         proto.Message
		paths       []string
		nilMask     bool
		wantMissing string
	}{
		{
			name:    "nil mask skips unpopulated required fields",
			msg:     &testproto.Book{},
			nilMask: true,
		},
		{
			name: "nil mask still validates inside populated messages",
			msg: &testproto.Book{
				Author: &testproto.Author{Age: 50},
			},
			nilMask:     true,
			wantMissing: "author.name",
		},
		{
			name:        "masked empty required field fails",
			msg:         &testproto.Book{},
			paths:       []string{"title"},
			wantMissing: "title",
		},
		{
			name:        "masked required subfield of unset parent fails",
			msg:         &testproto.Book{},
			paths:       []string{"author.name"},
			wantMissing: "author.name",
		},
		{
			name:  "sibling subfield mask does not cover the required one",
			msg:   &testproto.Book{Author: &testproto.Author{Age: 50}},
			paths: []string{"author.age"},
		},
		{
			name:        "prefix covers nested required fields",
			msg:         &testproto.Book{Author: &testproto.Author{Age: 50}},
			paths:       []string{"author"},
			wantMissing: "author.name",
		},
		{
			// The prefix "author" must reach author.name whether or not the
			// message carries an author, exactly as the "author.name" path
			// above does.
			name:        "prefix covers nested required fields of an unset parent",
			msg:         &testproto.Book{},
			paths:       []string{"author"},
			wantMissing: "author.name",
		},
		{
			name:  "integer-keyed map paths are skipped",
			msg:   &testproto.Book{Editions: map[int32]string{1: "first"}},
			paths: []string{"editions.1"},
		},
		{
			name:        "wildcard validates everything",
			msg:         &testproto.Book{},
			paths:       []string{"*"},
			wantMissing: "title",
		},
		{
			name: "masked map entry validates its required subfields",
			msg: &testproto.Book{
				Reviews: map[string]*testproto.Review{"smith": {Rating: 5}},
			},
			paths:       []string{"reviews.smith"},
			wantMissing: "reviews.smith.text",
		},
		{
			name:  "subfield mask below a map key skips its required siblings",
			msg:   &testproto.Book{Reviews: map[string]*testproto.Review{"smith": {Rating: 5}}},
			paths: []string{"reviews.smith.rating"},
		},
		{
			name:  "mask for an absent map entry has nothing to validate",
			msg:   &testproto.Book{},
			paths: []string{"reviews.smith"},
		},
		{
			name:  "unknown paths are ignored",
			msg:   fullBook(),
			paths: []string{"nope", "author.nope"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mask *fieldmaskpb.FieldMask
			if !tt.nilMask {
				mask = &fieldmaskpb.FieldMask{Paths: tt.paths}
			}

			err := fieldbehavior.ValidateRequiredWithMask(tt.msg, mask)
			assertMissing(t, err, tt.wantMissing)
		})
	}

	// Descending into an unset parent must not pile the parent's subfields
	// on top of the parent's own error: a missing book is one complaint,
	// not one per field the book would have had.
	t.Run("missing required message reports itself, not its subfields", func(t *testing.T) {
		t.Parallel()

		msg := &testproto.CreateBookRequest{Parent: "shelves/1"}
		mask := &fieldmaskpb.FieldMask{Paths: []string{"book"}}

		err := fieldbehavior.ValidateRequiredWithMask(msg, mask)
		assertMissing(t, err, "book")

		if strings.Contains(err.Error(), "book.title") {
			t.Fatalf("error %q reports a subfield of the missing message", err)
		}
	})
}

// assertMissing checks that err reports the given missing required field
// path, or that err is nil when wantMissing is empty.
func assertMissing(t *testing.T, err error, wantMissing string) {
	t.Helper()

	if wantMissing == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		return
	}

	if err == nil {
		t.Fatalf("want error for missing %q, got nil", wantMissing)
	}

	if !errors.Is(err, fieldbehavior.ErrMissingRequired) {
		t.Fatalf("error %v does not match ErrMissingRequired", err)
	}

	if !strings.Contains(err.Error(), wantMissing) {
		t.Fatalf("error %q does not name field %q", err, wantMissing)
	}
}
