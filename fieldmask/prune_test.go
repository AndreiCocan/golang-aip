package fieldmask_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/AndreiCocan/golang-aip/fieldmask"
	"github.com/AndreiCocan/golang-aip/internal/testproto"
)

func TestPrune(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		paths   []string
		nilMask bool
		want    *testproto.Book // ignored when wantErr is set
		wantErr error
	}{
		{
			name:    "omitted mask keeps every field",
			nilMask: true,
			want:    storedBook(),
		},
		{
			name:  "empty mask keeps every field",
			paths: []string{},
			want:  storedBook(),
		},
		{
			name:  "wildcard keeps every field",
			paths: []string{"*"},
			want:  storedBook(),
		},
		{
			name:  "single field",
			paths: []string{"title"},
			want:  &testproto.Book{Title: storedBook().GetTitle()},
		},
		{
			name:  "subfield keeps only that subfield",
			paths: []string{"author.name"},
			want:  &testproto.Book{Author: &testproto.Author{Name: "Alan Donovan"}},
		},
		{
			name:  "several paths",
			paths: []string{"title", "author"},
			want: &testproto.Book{
				Title:  storedBook().GetTitle(),
				Author: storedBook().GetAuthor(),
			},
		},
		{
			name:  "whole map",
			paths: []string{"labels"},
			want:  &testproto.Book{Labels: storedBook().GetLabels()},
		},
		{
			name:  "single map entry",
			paths: []string{"labels.lang"},
			want:  &testproto.Book{Labels: map[string]string{"lang": "en"}},
		},
		{
			name:  "single integer-keyed map entry",
			paths: []string{"editions.1"},
			want:  &testproto.Book{Editions: map[int32]string{1: "first"}},
		},
		{
			// A non-canonical integer key must name the entry it parses to,
			// rather than matching nothing and pruning the entry away.
			name:  "non-canonical integer-keyed map entry",
			paths: []string{"editions.02"},
			want:  &testproto.Book{Editions: map[int32]string{2: "second"}},
		},
		{
			name:  "backticked map key",
			paths: []string{"labels.`lang`"},
			want:  &testproto.Book{Labels: map[string]string{"lang": "en"}},
		},
		{
			name:  "subfield below a map key",
			paths: []string{"reviews.smith.rating"},
			want:  &testproto.Book{Reviews: map[string]*testproto.Review{"smith": {Rating: 5}}},
		},
		{
			name:  "whole repeated field",
			paths: []string{"shelves"},
			want:  &testproto.Book{Shelves: []string{"shelves/1"}},
		},
		{
			name:  "output-only fields are readable",
			paths: []string{"create_time"},
			want:  &testproto.Book{CreateTime: storedBook().GetCreateTime()},
		},
		{
			name:    "unknown path is rejected",
			paths:   []string{"nope"},
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

			msg := storedBook()

			err := fieldmask.Prune(mask, msg)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Prune() = %v, want %v", err, tt.wantErr)
				}

				if diff := cmp.Diff(storedBook(), msg, protocmp.Transform()); diff != "" {
					t.Fatalf("Prune() modified msg on error (-want +got):\n%s", diff)
				}

				return
			}

			if err != nil {
				t.Fatalf("Prune() = %v, want nil", err)
			}

			if diff := cmp.Diff(tt.want, msg, protocmp.Transform()); diff != "" {
				t.Fatalf("Prune() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
