package sqlite_test

import (
	"errors"
	"testing"

	"github.com/AndreiCocan/golang-aip/ordering"
	"github.com/AndreiCocan/golang-aip/ordering/sqlite"
)

// bookSchema declares the orderable fields the transpile tests use.
var bookSchema = ordering.NewSchema("title", "create_time", "author.name")

// compile compiles orderBy against bookSchema, failing the test on error.
func compile(t *testing.T, orderBy string) *ordering.Checked {
	t.Helper()

	checked, err := ordering.Compile(orderBy, bookSchema)
	if err != nil {
		t.Fatalf("Compile(%q) error: %v", orderBy, err)
	}

	return checked
}

func TestTranspile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		orderBy string
		opts    []sqlite.Option
		want    string
	}{
		{"empty", "", nil, ""},
		{"single field", "title", nil, `"title"`},
		{"single field desc", "title desc", nil, `"title" DESC`},
		{
			"multiple fields",
			"create_time desc, title",
			nil,
			`"create_time" DESC, "title"`,
		},
		{
			"column override",
			"title desc",
			[]sqlite.Option{sqlite.Column("title", "book_title")},
			`"book_title" DESC`,
		},
		{
			"column override quotes embedded quote",
			"title",
			[]sqlite.Option{sqlite.Column("title", `weird"col`)},
			`"weird""col"`,
		},
		{
			"column expr inserted verbatim",
			"author.name desc, title",
			[]sqlite.Option{
				sqlite.ColumnExpr("author.name", "json_extract(author, '$.name')"),
			},
			`json_extract(author, '$.name') DESC, "title"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := sqlite.Transpile(compile(t, tt.orderBy), tt.opts...)
			if err != nil {
				t.Fatalf("Transpile(%q) error: %v", tt.orderBy, err)
			}

			if got != tt.want {
				t.Errorf("Transpile(%q) = %q, want %q", tt.orderBy, got, tt.want)
			}
		})
	}

	t.Run("nil order by", func(t *testing.T) {
		t.Parallel()

		got, err := sqlite.Transpile(nil)
		if err != nil {
			t.Fatalf("Transpile(nil) error: %v", err)
		}

		if got != "" {
			t.Errorf("Transpile(nil) = %q, want empty", got)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		t.Parallel()

		_, err := sqlite.Transpile(compile(t, "author.name"))
		if err == nil {
			t.Fatal("Transpile() succeeded, want error")
		}

		if !errors.Is(err, sqlite.ErrUnsupported) {
			t.Errorf("Transpile() error does not match ErrUnsupported: %v", err)
		}

		want := `sqlite: unsupported order by: nested field "author.name" requires a Column or ColumnExpr mapping`
		if err.Error() != want {
			t.Errorf("Transpile() error = %q, want %q", err.Error(), want)
		}
	})
}
