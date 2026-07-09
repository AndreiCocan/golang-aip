package fieldmask_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/AndreiCocan/golang-aip/fieldmask"
	"github.com/AndreiCocan/golang-aip/internal/testproto"
)

// storedBook returns the resource as the service would have it in storage,
// server-managed fields included.
func storedBook() *testproto.Book {
	return &testproto.Book{
		Name:       "shelves/1/books/1",
		Title:      "The Go Programming Language",
		Author:     &testproto.Author{Name: "Alan Donovan", Age: 50, Verified: true},
		Isbn:       "978-0134190440",
		CreateTime: "2026-01-01T00:00:00Z",
		Labels:     map[string]string{"lang": "en", "topic": "go"},
		Shelves:    []string{"shelves/1"},
		Reviews:    map[string]*testproto.Review{"smith": {Text: "great", Rating: 5}},
		PageCount:  380,
		Editions:   map[int32]string{1: "first", 2: "second"},
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		paths   []string
		nilMask bool
		src     *testproto.Book
		want    *testproto.Book // ignored when wantErr is set
		wantErr error
	}{
		{
			name:  "masked scalar is replaced",
			paths: []string{"title"},
			src:   &testproto.Book{Title: "The Go Programming Language, 2nd Edition"},
			want: func() *testproto.Book {
				b := storedBook()
				b.Title = "The Go Programming Language, 2nd Edition"

				return b
			}(),
		},
		{
			name:  "masked subfield leaves siblings alone",
			paths: []string{"author.name"},
			src:   &testproto.Book{Author: &testproto.Author{Name: "Brian Kernighan", Age: 1}},
			want: func() *testproto.Book {
				b := storedBook()
				b.Author.Name = "Brian Kernighan"

				return b
			}(),
		},
		{
			name:  "masked field unset in the payload is cleared",
			paths: []string{"labels"},
			src:   &testproto.Book{},
			want: func() *testproto.Book {
				b := storedBook()
				b.Labels = nil

				return b
			}(),
		},
		{
			name:    "omitted mask updates the populated fields",
			nilMask: true,
			src:     &testproto.Book{Title: "New Title", PageCount: 400},
			want: func() *testproto.Book {
				b := storedBook()
				b.Title = "New Title"
				b.PageCount = 400

				return b
			}(),
		},
		{
			name:    "omitted mask never clears",
			nilMask: true,
			src:     &testproto.Book{Title: "New Title"},
			want: func() *testproto.Book {
				b := storedBook()
				b.Title = "New Title"

				return b
			}(),
		},
		{
			name:  "output-only path is ignored without error",
			paths: []string{"create_time", "title"},
			src:   &testproto.Book{CreateTime: "2027-01-01T00:00:00Z", Title: "New Title"},
			want: func() *testproto.Book {
				b := storedBook()
				b.Title = "New Title"

				return b
			}(),
		},
		{
			name:  "output-only survives replacement of its parent",
			paths: []string{"author"},
			src:   &testproto.Book{Author: &testproto.Author{Name: "Brian Kernighan"}},
			want: func() *testproto.Book {
				b := storedBook()
				b.Author = &testproto.Author{Name: "Brian Kernighan", Verified: true}

				return b
			}(),
		},
		{
			// The payload clears author, so only the stored resource can
			// still say that its output-only subfield was set.
			name:  "output-only survives clearing its parent",
			paths: []string{"author"},
			src:   &testproto.Book{},
			want: func() *testproto.Book {
				b := storedBook()
				b.Author = &testproto.Author{Verified: true}

				return b
			}(),
		},
		{
			name:  "immutable field set to its current value is a no-op",
			paths: []string{"isbn", "title"},
			src:   &testproto.Book{Isbn: "978-0134190440", Title: "New Title"},
			want: func() *testproto.Book {
				b := storedBook()
				b.Title = "New Title"

				return b
			}(),
		},
		{
			name:    "immutable field change is rejected",
			paths:   []string{"isbn"},
			src:     &testproto.Book{Isbn: "changed"},
			wantErr: fieldmask.ErrImmutable,
		},
		{
			name:    "immutable field cleared explicitly is rejected",
			paths:   []string{"isbn"},
			src:     &testproto.Book{},
			wantErr: fieldmask.ErrImmutable,
		},
		{
			name:    "identifier change is rejected",
			paths:   []string{"name"},
			src:     &testproto.Book{Name: "shelves/1/books/2"},
			wantErr: fieldmask.ErrImmutable,
		},
		{
			name:    "implied mask still rejects immutable changes",
			nilMask: true,
			src:     &testproto.Book{Isbn: "changed"},
			wantErr: fieldmask.ErrImmutable,
		},
		{
			name:  "wildcard replaces everything writable",
			paths: []string{"*"},
			src: &testproto.Book{
				Title:  "New Title",
				Author: &testproto.Author{Name: "Brian Kernighan"},
			},
			want: &testproto.Book{
				Name:       "shelves/1/books/1",
				Title:      "New Title",
				Author:     &testproto.Author{Name: "Brian Kernighan", Verified: true},
				Isbn:       "978-0134190440",
				CreateTime: "2026-01-01T00:00:00Z",
			},
		},
		{
			name:    "wildcard with changed immutable is rejected",
			paths:   []string{"*"},
			src:     &testproto.Book{Title: "New Title", Isbn: "changed"},
			wantErr: fieldmask.ErrImmutable,
		},
		{
			name:    "unknown path is rejected",
			paths:   []string{"nope"},
			src:     &testproto.Book{},
			wantErr: fieldmask.ErrInvalidFieldMask,
		},
		{
			name:  "map entry is updated by key",
			paths: []string{"labels.lang"},
			src:   &testproto.Book{Labels: map[string]string{"lang": "fr"}},
			want: func() *testproto.Book {
				b := storedBook()
				b.Labels["lang"] = "fr"

				return b
			}(),
		},
		{
			name:  "map entry absent from the payload is deleted",
			paths: []string{"labels.lang"},
			src:   &testproto.Book{},
			want: func() *testproto.Book {
				b := storedBook()
				delete(b.GetLabels(), "lang")

				return b
			}(),
		},
		{
			name:  "subfield below a map key leaves the rest of the entry",
			paths: []string{"reviews.smith.rating"},
			src:   &testproto.Book{Reviews: map[string]*testproto.Review{"smith": {Rating: 4}}},
			want: func() *testproto.Book {
				b := storedBook()
				b.Reviews["smith"].Rating = 4

				return b
			}(),
		},
		{
			name:  "map entry with an integer key is updated",
			paths: []string{"editions.2"},
			src:   &testproto.Book{Editions: map[int32]string{2: "revised"}},
			want: func() *testproto.Book {
				b := storedBook()
				b.Editions[2] = "revised"

				return b
			}(),
		},
		{
			name:  "map entry with a non-canonical integer key is updated",
			paths: []string{"editions.02"},
			src:   &testproto.Book{Editions: map[int32]string{2: "revised"}},
			want: func() *testproto.Book {
				b := storedBook()
				b.Editions[2] = "revised"

				return b
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mask *fieldmaskpb.FieldMask
			if !tt.nilMask {
				mask = &fieldmaskpb.FieldMask{Paths: tt.paths}
			}

			dst := storedBook()

			err := fieldmask.Update(mask, dst, tt.src)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Update() = %v, want %v", err, tt.wantErr)
				}

				if diff := cmp.Diff(storedBook(), dst, protocmp.Transform()); diff != "" {
					t.Fatalf("Update() modified dst on error (-want +got):\n%s", diff)
				}

				return
			}

			if err != nil {
				t.Fatalf("Update() = %v, want nil", err)
			}

			if diff := cmp.Diff(tt.want, dst, protocmp.Transform()); diff != "" {
				t.Fatalf("Update() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	// Update merges values out of src, so the two could easily end up
	// sharing the messages, maps, and slices below them. They must not: the
	// payload belongs to the caller, and the resource outlives the request.
	t.Run("shares no data with src", func(t *testing.T) {
		t.Parallel()

		src := &testproto.Book{
			Title:   "New Title",
			Author:  &testproto.Author{Name: "Brian Kernighan"},
			Labels:  map[string]string{"lang": "fr"},
			Reviews: map[string]*testproto.Review{"smith": {Text: "fine", Rating: 3}},
		}
		before := proto.Clone(src)

		dst := storedBook()
		mask := &fieldmaskpb.FieldMask{Paths: []string{"title", "author", "labels", "reviews"}}

		if err := fieldmask.Update(mask, dst, src); err != nil {
			t.Fatalf("Update() = %v, want nil", err)
		}

		if diff := cmp.Diff(before, src, protocmp.Transform()); diff != "" {
			t.Fatalf("Update() modified src (-want +got):\n%s", diff)
		}

		// Writing to the updated resource must not reach back into src.
		dst.GetAuthor().Name = "changed"
		dst.GetLabels()["lang"] = "de"
		dst.GetReviews()["smith"].Text = "changed"

		if diff := cmp.Diff(before, src, protocmp.Transform()); diff != "" {
			t.Fatalf("writing to dst changed src (-want +got):\n%s", diff)
		}
	})

	t.Run("mismatched message types panic", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if recover() == nil {
				t.Fatal("Update() with mismatched message types did not panic")
			}
		}()

		_ = fieldmask.Update(nil, &testproto.Book{}, &testproto.Author{})
	})

	// Backtick quoting is what lets a map key carry a character the path
	// syntax claims for itself, the wildcard included.
	t.Run("backticked wildcard map key", func(t *testing.T) {
		t.Parallel()

		dst := &testproto.Book{Labels: map[string]string{"*": "star", "lang": "en"}}
		src := &testproto.Book{Labels: map[string]string{"*": "changed"}}

		mask := &fieldmaskpb.FieldMask{Paths: []string{"labels.`*`"}}
		if err := fieldmask.Update(mask, dst, src); err != nil {
			t.Fatalf("Update() = %v, want nil", err)
		}

		want := &testproto.Book{Labels: map[string]string{"*": "changed", "lang": "en"}}
		if diff := cmp.Diff(want, dst, protocmp.Transform()); diff != "" {
			t.Fatalf("Update() mismatch (-want +got):\n%s", diff)
		}
	})
}
