package filtering_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/AndreiCocan/golang-aip/filtering"
)

// neverExpand is a placeholder expander for declaration-level tests; the
// declarations it appears in are invalid, so it is never called.
func neverExpand(*filtering.Schema, []filtering.Value) (filtering.Expr, error) {
	return nil, errors.New("never called")
}

func TestNewSchema(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name  string
		build func()
		want  string
	}{
		{
			name:  "duplicate field",
			build: func() { filtering.NewSchema(filtering.String("a"), filtering.Int("a")) },
			want:  `field "a" declared twice`,
		},
		{
			name:  "empty field name",
			build: func() { filtering.NewSchema(filtering.String("")) },
			want:  "empty name",
		},
		{
			name:  "enum without values",
			build: func() { filtering.NewSchema(filtering.Enum("state")) },
			want:  `enum field "state" declared without values`,
		},
		{
			name:  "repeated of repeated",
			build: func() { filtering.NewSchema(filtering.Repeated(filtering.Repeated(filtering.String("x")))) },
			want:  "repeated or map",
		},
		{
			name:  "function without returns or expand",
			build: func() { filtering.NewSchema(filtering.Function("f")) },
			want:  "must declare Returns or Expand",
		},
		{
			name: "duplicate function",
			build: func() {
				filtering.NewSchema(
					filtering.Function("f", filtering.Returns(filtering.KindBool)),
					filtering.Function("f", filtering.Returns(filtering.KindBool)),
				)
			},
			want: `function "f" declared twice`,
		},
		{
			name: "non-scalar argument kind",
			build: func() {
				filtering.NewSchema(filtering.Function("f",
					filtering.Args(filtering.KindRepeated), filtering.Returns(filtering.KindBool)))
			},
			want: "non-scalar",
		},
		{
			name: "expander with non-bool result",
			build: func() {
				filtering.NewSchema(filtering.Function("f",
					filtering.Returns(filtering.KindString),
					filtering.Expand(neverExpand),
				))
			},
			want: "must return bool",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("NewSchema did not panic")
				}

				if msg, ok := r.(string); !ok || !strings.Contains(msg, tt.want) {
					t.Errorf("NewSchema panic = %v, want it to contain %q", r, tt.want)
				}
			}()

			tt.build()
		})
	}
}

func TestSchema_Field(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name     string
		path     string
		wantPath string
		wantKind filtering.Kind
		wantErr  bool
	}{
		{name: "top-level", path: "display_name", wantPath: "display_name", wantKind: filtering.KindString},
		{name: "nested", path: "author.name", wantPath: "author.name", wantKind: filtering.KindString},
		{name: "map key", path: "labels.env", wantPath: "labels.env", wantKind: filtering.KindString},
		{name: "across repeated", path: "chapters.pages", wantPath: "chapters.pages", wantKind: filtering.KindInt},
		{name: "unknown", path: "missing", wantErr: true},
		{name: "unknown nested", path: "author.missing", wantErr: true},
		{name: "through scalar", path: "display_name.x", wantErr: true},
		{name: "empty", path: "", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			field, err := checkSchema.Field(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Field(%q) succeeded, want error", tt.path)
				}

				return
			}

			if err != nil {
				t.Fatalf("Field(%q) error = %v", tt.path, err)
			}

			if field.Path() != tt.wantPath {
				t.Errorf("Field(%q).Path() = %q, want %q", tt.path, field.Path(), tt.wantPath)
			}

			if field.Type().Kind != tt.wantKind {
				t.Errorf(
					"Field(%q).Type().Kind = %v, want %v",
					tt.path,
					field.Type().Kind,
					tt.wantKind,
				)
			}
		})
	}
}
