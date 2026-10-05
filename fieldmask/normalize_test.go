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

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		paths   []string
		nilMask bool
		want    []string // ignored when wantErr is set
		wantErr error
	}{
		{name: "nil mask", nilMask: true, want: []string{}},
		{name: "empty mask", paths: []string{}, want: []string{}},
		{name: "wildcard", paths: []string{"*"}, want: []string{"*"}},
		{name: "proto name", paths: []string{"create_time"}, want: []string{"create_time"}},
		{name: "JSON name", paths: []string{"createTime"}, want: []string{"create_time"}},
		{
			name:  "JSON names in several paths",
			paths: []string{"title", "pageCount", "featuredReviews"},
			want:  []string{"title", "page_count", "featured_reviews"},
		},
		{name: "subfield", paths: []string{"author.name"}, want: []string{"author.name"}},
		{
			name:  "map key keeps its case",
			paths: []string{"labels.pageCount"},
			want:  []string{"labels.pageCount"},
		},
		{
			name:  "field below a map key",
			paths: []string{"reviews.smith.rating"},
			want:  []string{"reviews.smith.rating"},
		},
		{
			name:  "quoted map key",
			paths: []string{"labels.`k8s.io/name`"},
			want:  []string{"labels.`k8s.io/name`"},
		},
		{name: "quoted field name", paths: []string{"`title`"}, want: []string{"`title`"}},
		{
			name:  "integer map key keeps its form",
			paths: []string{"editions.05"},
			want:  []string{"editions.05"},
		},

		{name: "unknown field", paths: []string{"nope"}, wantErr: fieldmask.ErrInvalidFieldMask},
		{
			name:    "quoted JSON name",
			paths:   []string{"`createTime`"},
			wantErr: fieldmask.ErrInvalidFieldMask,
		},
		{
			name:    "JSON name into a repeated field",
			paths:   []string{"featuredReviews.text"},
			wantErr: fieldmask.ErrInvalidFieldMask,
		},
		{name: "empty path", paths: []string{""}, wantErr: fieldmask.ErrInvalidFieldMask},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mask *fieldmaskpb.FieldMask
			if !tt.nilMask {
				mask = &fieldmaskpb.FieldMask{Paths: tt.paths}
			}

			got, err := fieldmask.Normalize(mask, &testproto.Book{})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Normalize() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			want := &fieldmaskpb.FieldMask{Paths: tt.want}
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("Normalize() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNormalizeKeepsTheMask(t *testing.T) {
	t.Parallel()

	mask := &fieldmaskpb.FieldMask{Paths: []string{"createTime"}}

	if _, err := fieldmask.Normalize(mask, &testproto.Book{}); err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	want := &fieldmaskpb.FieldMask{Paths: []string{"createTime"}}
	if diff := cmp.Diff(want, mask, protocmp.Transform()); diff != "" {
		t.Errorf("mask changed (-want +got):\n%s", diff)
	}
}
