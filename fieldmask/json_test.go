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

func TestMarshalJSON(t *testing.T) {
	t.Parallel()

	// allFields is the JSON of a book with only a title, every field
	// written.
	const allFields = `{"name":"","title":"Go","isbn":"","createTime":"","importToken":"",` +
		`"labels":{},"shelves":[],"reviews":{},"pageCount":0,"editions":{},"featuredReviews":[]}`

	tests := []struct {
		name    string
		paths   []string
		nilMask bool
		msg     *testproto.Book
		want    string // ignored when wantErr is set
		wantErr error
	}{
		{
			name:    "omitted mask writes every field",
			nilMask: true,
			msg:     &testproto.Book{Title: "Go"},
			want:    allFields,
		},
		{
			name:  "wildcard writes every field",
			paths: []string{"*"},
			msg:   &testproto.Book{Title: "Go"},
			want:  allFields,
		},
		{
			name:  "leaves out the fields that the mask does not cover",
			paths: []string{"title"},
			msg:   storedBook(),
			want:  `{"title":"The Go Programming Language"}`,
		},
		{
			name:  "writes a covered field that has its default value",
			paths: []string{"title", "page_count", "labels", "shelves"},
			msg:   &testproto.Book{Title: "Go"},
			want:  `{"title":"Go","labels":{},"shelves":[],"pageCount":0}`,
		},
		{
			name:  "leaves out a covered message field that is unset",
			paths: []string{"author"},
			msg:   &testproto.Book{Title: "Go"},
			want:  `{}`,
		},
		{
			name:  "writes a covered subfield that has its default value",
			paths: []string{"author.verified"},
			msg:   &testproto.Book{Author: &testproto.Author{Name: "Alan Donovan"}},
			want:  `{"author":{"verified":false}}`,
		},
		{
			name:  "single map entry",
			paths: []string{"labels.lang"},
			msg:   storedBook(),
			want:  `{"labels":{"lang":"en"}}`,
		},
		{
			name:  "non-canonical integer-keyed map entry",
			paths: []string{"editions.02"},
			msg:   storedBook(),
			want:  `{"editions":{"2":"second"}}`,
		},
		{
			name:  "subfield below a map key",
			paths: []string{"reviews.smith.rating"},
			msg:   storedBook(),
			want:  `{"reviews":{"smith":{"rating":5}}}`,
		},
		{
			name:  "field of each element of a repeated field",
			paths: []string{"featured_reviews.rating"},
			msg: &testproto.Book{FeaturedReviews: []*testproto.Review{
				{Text: "great", Rating: 5},
				{Text: "unrated"},
			}},
			want: `{"featuredReviews":[{"rating":5},{"rating":0}]}`,
		},
		{
			name:    "unknown path is rejected",
			paths:   []string{"nope"},
			msg:     storedBook(),
			wantErr: fieldmask.ErrInvalidFieldMask,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mask *fieldmaskpb.FieldMask
			if !tt.nilMask {
				mask = &fieldmaskpb.FieldMask{Paths: tt.paths}
			}

			original := proto.Clone(tt.msg)

			got, err := fieldmask.MarshalJSON(mask, tt.msg)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("MarshalJSON() = %v, want %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("MarshalJSON() = %v, want nil", err)
			}

			if string(got) != tt.want {
				t.Errorf("MarshalJSON() = %s, want %s", got, tt.want)
			}

			if diff := cmp.Diff(original, tt.msg, protocmp.Transform()); diff != "" {
				t.Errorf("MarshalJSON() modified msg (-want +got):\n%s", diff)
			}
		})
	}
}
