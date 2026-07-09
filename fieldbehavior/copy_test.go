package fieldbehavior_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/AndreiCocan/golang-aip/fieldbehavior"
	"github.com/AndreiCocan/golang-aip/internal/testproto"
)

func TestCopy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		dst       *testproto.Book
		src       *testproto.Book
		behaviors []annotations.FieldBehavior
		want      *testproto.Book
	}{
		{
			name: "output_only copied from src, other fields kept",
			dst: &testproto.Book{
				Title:      "kept",
				CreateTime: "old",
				Author:     &testproto.Author{Name: "kept", Verified: false},
			},
			src: &testproto.Book{
				Title:      "ignored",
				CreateTime: "new",
				Author:     &testproto.Author{Name: "ignored", Verified: true},
			},
			behaviors: []annotations.FieldBehavior{annotations.FieldBehavior_OUTPUT_ONLY},
			want: &testproto.Book{
				Title:      "kept",
				CreateTime: "new",
				Author:     &testproto.Author{Name: "kept", Verified: true},
			},
		},
		{
			name:      "unpopulated src clears annotated dst field",
			dst:       &testproto.Book{Title: "kept", CreateTime: "old"},
			src:       &testproto.Book{},
			behaviors: []annotations.FieldBehavior{annotations.FieldBehavior_OUTPUT_ONLY},
			want:      &testproto.Book{Title: "kept"},
		},
		{
			name:      "nested annotated field copied even when dst lacks the parent",
			dst:       &testproto.Book{Title: "kept"},
			src:       &testproto.Book{Author: &testproto.Author{Name: "ignored", Verified: true}},
			behaviors: []annotations.FieldBehavior{annotations.FieldBehavior_OUTPUT_ONLY},
			want: &testproto.Book{
				Title:  "kept",
				Author: &testproto.Author{Verified: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fieldbehavior.Copy(tt.dst, tt.src, tt.behaviors...)

			if diff := cmp.Diff(tt.want, tt.dst, protocmp.Transform()); diff != "" {
				t.Fatalf("Copy() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	// Copy documents that it shares message values rather than deep-copying
	// them, so a later write to src is visible through dst.
	t.Run("message values are copied by reference", func(t *testing.T) {
		t.Parallel()

		src := &testproto.Book{Author: &testproto.Author{Name: "Alan Donovan"}}
		dst := &testproto.Book{}

		fieldbehavior.Copy(dst, src, annotations.FieldBehavior_OPTIONAL)
		src.GetAuthor().Age = 51

		if got := dst.GetAuthor().GetAge(); got != 51 {
			t.Fatalf("dst author age = %d, want 51 through the shared message", got)
		}
	})

	t.Run("nil message panics", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if recover() == nil {
				t.Fatal("Copy() with a nil message did not panic")
			}
		}()

		fieldbehavior.Copy(nil, nil, annotations.FieldBehavior_OUTPUT_ONLY)
	})

	t.Run("mismatched message types panic", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if recover() == nil {
				t.Fatal("Copy() with mismatched message types did not panic")
			}
		}()

		fieldbehavior.Copy(
			&testproto.Book{},
			&testproto.Author{},
			annotations.FieldBehavior_OUTPUT_ONLY,
		)
	})
}
