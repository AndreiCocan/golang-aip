package ordering_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/AndreiCocan/golang-aip/ordering"
)

// taggedBook is a domain type with orderable and other fields.
type taggedBook struct {
	Name       string            `aip:"name,filter,order"`
	Title      string            `aip:"title,filter,order,search"`
	CreateTime time.Time         `aip:"create_time,order"`
	Labels     map[string]string `aip:"labels,filter"`
	Revision   int64
}

func TestSchemaFromTags(t *testing.T) {
	t.Parallel()

	schema := ordering.SchemaFromTags(taggedBook{})

	tests := []struct {
		orderBy string
		want    *ordering.CheckedOrderBy
		wantErr error
	}{
		{
			orderBy: "name, title desc, create_time",
			want: &ordering.CheckedOrderBy{Keys: []ordering.Key{
				{Segments: []string{"name"}},
				{Segments: []string{"title"}, Desc: true},
				{Segments: []string{"create_time"}},
			}},
		},
		{orderBy: "labels", wantErr: ordering.ErrInvalidOrderBy},
		{orderBy: "Revision", wantErr: ordering.ErrInvalidOrderBy},
	}

	for _, tt := range tests {
		t.Run(tt.orderBy, func(t *testing.T) {
			t.Parallel()

			got, err := ordering.Compile(tt.orderBy, schema)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Compile() error = %v, want %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Compile() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSchemaFromTagsPanics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		v    any
	}{
		{"not a struct", "book"},
		{"pointer to a struct", &taggedBook{}},
		{"nil", nil},
		{"invalid tag", struct {
			Title string `aip:"title,sort"`
		}{}},
		{"duplicate name", struct {
			A string `aip:"title,order"`
			B string `aip:"title,order"`
		}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recover() == nil {
					t.Error("SchemaFromTags() did not panic")
				}
			}()

			ordering.SchemaFromTags(tt.v)
		})
	}
}
