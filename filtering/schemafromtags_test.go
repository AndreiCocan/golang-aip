package filtering_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/AndreiCocan/golang-aip/filtering"
)

// taggedBook is a domain type with a field of each supported Go type.
type taggedBook struct {
	Name       string            `aip:"name,filter,search"`
	Title      string            `aip:"title,filter,order,search"`
	Pages      int32             `aip:"page_count,filter"`
	Rating     float64           `aip:"rating,filter"`
	Published  bool              `aip:"published,filter"`
	CreateTime time.Time         `aip:"create_time,filter"`
	ReadTime   time.Duration     `aip:"read_time,filter"`
	Tags       []string          `aip:"tags,filter"`
	Labels     map[string]string `aip:"labels,filter"`
	Summary    string            `aip:"summary,search"`
	Revision   int64
}

func TestSchemaFromTags(t *testing.T) {
	t.Parallel()

	schema := filtering.SchemaFromTags(taggedBook{},
		filtering.Func("recent", filtering.FuncReturns(filtering.KindBool)),
	)

	tests := []struct {
		path string
		want filtering.Type
	}{
		{"name", typ(filtering.KindString)},
		{"title", typ(filtering.KindString)},
		{"page_count", typ(filtering.KindInt)},
		{"rating", typ(filtering.KindFloat)},
		{"published", typ(filtering.KindBool)},
		{"create_time", typ(filtering.KindTimestamp)},
		{"read_time", typ(filtering.KindDuration)},
		{"tags", elemTyp(filtering.KindRepeated, filtering.KindString)},
		{"labels", elemTyp(filtering.KindMap, filtering.KindString)},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			field, err := schema.Field(tt.path)
			if err != nil {
				t.Fatalf("Field(%q) error = %v", tt.path, err)
			}

			if diff := cmp.Diff(
				tt.want,
				field.Type(),
				cmpopts.IgnoreUnexported(filtering.Type{}),
			); diff != "" {
				t.Errorf("Field(%q) type mismatch (-want +got):\n%s", tt.path, diff)
			}
		})
	}

	t.Run("fields without the filter option", func(t *testing.T) {
		t.Parallel()

		for _, path := range []string{"summary", "Revision"} {
			if _, err := schema.Field(path); err == nil {
				t.Errorf("Field(%q) error = nil, want an error", path)
			}
		}
	})

	t.Run("extra declarations", func(t *testing.T) {
		t.Parallel()

		got, err := filtering.Compile("recent()", schema)
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}

		want := &filtering.CheckedFilter{Expr: &filtering.Comparison{
			Left:  &filtering.FuncCall{Name: "recent", Result: typ(filtering.KindBool)},
			Op:    filtering.OpEquals,
			Right: filtering.BoolValue(true),
		}}
		if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(filtering.Type{})); diff != "" {
			t.Errorf("Compile() mismatch (-want +got):\n%s", diff)
		}
	})
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
		{"unsupported type", struct {
			Size uint `aip:"size,filter"`
		}{}},
		{"map with non-string keys", struct {
			Counts map[int]string `aip:"counts,filter"`
		}{}},
		{"slice of slices", struct {
			Grid [][]string `aip:"grid,filter"`
		}{}},
		{"search on a non-string", struct {
			Pages int64 `aip:"page_count,search"`
		}{}},
		{"duplicate name", struct {
			A string `aip:"title,filter"`
			B string `aip:"title,filter"`
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

			filtering.SchemaFromTags(tt.v)
		})
	}
}
