package fieldbehavior_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/AndreiCocan/golang-aip/fieldbehavior"
	"github.com/AndreiCocan/golang-aip/internal/testproto"
)

func TestClear(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		msg       *testproto.Book
		behaviors []annotations.FieldBehavior
		want      *testproto.Book
	}{
		{
			name:      "output_only clears top-level and nested",
			msg:       fullBook(),
			behaviors: []annotations.FieldBehavior{annotations.FieldBehavior_OUTPUT_ONLY},
			want: func() *testproto.Book {
				b := fullBook()
				b.CreateTime = ""
				b.Author.Verified = false

				return b
			}(),
		},
		{
			name: "output_only and identifier clear the name too",
			msg:  fullBook(),
			behaviors: []annotations.FieldBehavior{
				annotations.FieldBehavior_OUTPUT_ONLY,
				annotations.FieldBehavior_IDENTIFIER,
			},
			want: func() *testproto.Book {
				b := fullBook()
				b.Name = ""
				b.CreateTime = ""
				b.Author.Verified = false

				return b
			}(),
		},
		{
			name:      "input_only clears the import token",
			msg:       fullBook(),
			behaviors: []annotations.FieldBehavior{annotations.FieldBehavior_INPUT_ONLY},
			want: func() *testproto.Book {
				b := fullBook()
				b.ImportToken = ""

				return b
			}(),
		},
		{
			name:      "unmatched behavior changes nothing",
			msg:       fullBook(),
			behaviors: []annotations.FieldBehavior{annotations.FieldBehavior_UNORDERED_LIST},
			want:      fullBook(),
		},
		{
			name:      "unset nested message is safe",
			msg:       &testproto.Book{Title: "bare"},
			behaviors: []annotations.FieldBehavior{annotations.FieldBehavior_OUTPUT_ONLY},
			want:      &testproto.Book{Title: "bare"},
		},
		{
			// A nested message carries its own annotations wherever it sits,
			// map values and repeated elements included.
			name: "annotated fields cleared inside map values and repeated elements",
			msg: &testproto.Book{
				Title:           "The Go Programming Language",
				Author:          &testproto.Author{Name: "Alan Donovan"},
				Reviews:         map[string]*testproto.Review{"smith": {Text: "great", Rating: 5}},
				FeaturedReviews: []*testproto.Review{{Text: "superb", Rating: 4}},
			},
			behaviors: []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED},
			want: &testproto.Book{
				Author:          &testproto.Author{},
				Reviews:         map[string]*testproto.Review{"smith": {Rating: 5}},
				FeaturedReviews: []*testproto.Review{{Rating: 4}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fieldbehavior.Clear(tt.msg, tt.behaviors...)

			if diff := cmp.Diff(tt.want, tt.msg, protocmp.Transform()); diff != "" {
				t.Fatalf("Clear() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("nil message is safe", func(t *testing.T) {
		t.Parallel()

		fieldbehavior.Clear(nil, annotations.FieldBehavior_OUTPUT_ONLY)
	})
}
