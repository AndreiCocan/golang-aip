package aiptag_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/AndreiCocan/golang-aip/aiptag"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		tag    reflect.StructTag
		want   aiptag.Tag
		wantOK bool
	}{
		{
			name:   "no tag",
			tag:    `json:"id"`,
			want:   aiptag.Tag{},
			wantOK: false,
		},
		{
			name:   "name only",
			tag:    `aip:"title"`,
			want:   aiptag.Tag{Name: "title"},
			wantOK: true,
		},
		{
			name: "all options",
			tag:  `aip:"name,filter,order,search"`,
			want: aiptag.Tag{
				Name:       "name",
				Filterable: true,
				Orderable:  true,
				Searchable: true,
			},
			wantOK: true,
		},
		{
			name:   "options in any order",
			tag:    `aip:"title,search,filter"`,
			want:   aiptag.Tag{Name: "title", Filterable: true, Searchable: true},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok, err := aiptag.Parse(reflect.StructField{Name: "F", Tag: tt.tag})
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if ok != tt.wantOK {
				t.Errorf("Parse() ok = %v, want %v", ok, tt.wantOK)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Parse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tag  reflect.StructTag
	}{
		{"empty name", `aip:",filter"`},
		{"unknown option", `aip:"title,sort"`},
		{"duplicate option", `aip:"title,filter,filter"`},
		{"option with a value", `aip:"title,filter=yes"`},
		{"pattern option", `aip:"name,pattern=books/{book}"`},
		{"empty option", `aip:"title,,filter"`},
		{"trailing comma", `aip:"title,"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, ok, err := aiptag.Parse(reflect.StructField{Name: "F", Tag: tt.tag})
			if !errors.Is(err, aiptag.ErrInvalidTag) {
				t.Errorf("Parse() error = %v, want ErrInvalidTag", err)
			}

			if !ok {
				t.Error("Parse() ok = false, want true")
			}
		})
	}
}
