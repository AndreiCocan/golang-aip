package ordering_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/AndreiCocan/golang-aip/ordering"
)

// bookSchema is the schema most Check tests validate against.
var bookSchema = ordering.NewSchema("title", "create_time", "author.name")

// ascKey and descKey build checked ordering keys.

func ascKey(segments ...string) ordering.Field {
	return ordering.Field{Segments: segments}
}

func descKey(segments ...string) ordering.Field {
	return ordering.Field{Segments: segments, Desc: true}
}

func checked(fields ...ordering.Field) *ordering.Checked {
	return &ordering.Checked{Fields: fields}
}

func TestCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		orderBy string
		want    *ordering.Checked
	}{
		{"empty", "", checked()},
		{"single field", "title", checked(ascKey("title"))},
		{"single field desc", "title desc", checked(descKey("title"))},
		{"dotted path", "author.name desc", checked(descKey("author", "name"))},
		{
			"order preserved",
			"create_time desc, title",
			checked(descKey("create_time"), ascKey("title")),
		},
		{
			"exact duplicate merged",
			"title, title",
			checked(ascKey("title")),
		},
		{
			"exact descending duplicate merged",
			"title desc, title desc",
			checked(descKey("title")),
		},
		{
			"merge keeps first position",
			"title, create_time, title",
			checked(ascKey("title"), ascKey("create_time")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := ordering.Parse(tt.orderBy)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.orderBy, err)
			}

			got, err := ordering.Check(parsed, bookSchema)
			if err != nil {
				t.Fatalf("Check(%q) error: %v", tt.orderBy, err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Check(%q) mismatch (-want +got):\n%s", tt.orderBy, diff)
			}
		})
	}

	t.Run("nil order by", func(t *testing.T) {
		t.Parallel()

		got, err := ordering.Check(nil, bookSchema)
		if err != nil {
			t.Fatalf("Check(nil) error: %v", err)
		}

		if diff := cmp.Diff(checked(), got); diff != "" {
			t.Errorf("Check(nil) mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			orderBy string
			wantPos int
			wantMsg string
		}{
			{"unknown field", "nope", 0, `unknown ordering field "nope"`},
			{
				"unknown nested field",
				"author.age",
				0,
				`unknown ordering field "author.age"`,
			},
			{
				"declared path prefix is not orderable",
				"author",
				0,
				`unknown ordering field "author"`,
			},
			{
				"unknown field after known one",
				"title, nope desc",
				7,
				`unknown ordering field "nope"`,
			},
			{
				"contradictory directions",
				"title, title desc",
				7,
				`field "title" is ordered both ascending and descending`,
			},
			{
				"contradictory directions reversed",
				"title desc, title",
				12,
				`field "title" is ordered both ascending and descending`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				parsed, err := ordering.Parse(tt.orderBy)
				if err != nil {
					t.Fatalf("Parse(%q) error: %v", tt.orderBy, err)
				}

				_, err = ordering.Check(parsed, bookSchema)
				if err == nil {
					t.Fatalf("Check(%q) succeeded, want error", tt.orderBy)
				}

				if !errors.Is(err, ordering.ErrInvalidOrderBy) {
					t.Errorf(
						"Check(%q) error does not match ErrInvalidOrderBy: %v",
						tt.orderBy,
						err,
					)
				}

				var checkErr *ordering.CheckError
				if !errors.As(err, &checkErr) {
					t.Fatalf("Check(%q) error is %T, want *CheckError", tt.orderBy, err)
				}

				if checkErr.OrderBy != tt.orderBy {
					t.Errorf("CheckError.OrderBy = %q, want %q", checkErr.OrderBy, tt.orderBy)
				}

				if checkErr.Pos != tt.wantPos {
					t.Errorf("CheckError.Pos = %d, want %d", checkErr.Pos, tt.wantPos)
				}

				if checkErr.Message != tt.wantMsg {
					t.Errorf("CheckError.Message = %q, want %q", checkErr.Message, tt.wantMsg)
				}
			})
		}
	})
}

func TestField_Path(t *testing.T) {
	t.Parallel()

	f := descKey("author", "name")
	if got := f.Path(); got != "author.name" {
		t.Errorf("Path() = %q, want %q", got, "author.name")
	}
}

func TestCompile(t *testing.T) {
	t.Parallel()

	got, err := ordering.Compile("author.name desc, title", bookSchema)
	if err != nil {
		t.Fatalf("Compile() error: %v", err)
	}

	want := checked(descKey("author", "name"), ascKey("title"))
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Compile() mismatch (-want +got):\n%s", diff)
	}

	t.Run("parse error propagates", func(t *testing.T) {
		t.Parallel()

		_, err := ordering.Compile("title asc", bookSchema)

		var parseErr *ordering.ParseError
		if !errors.As(err, &parseErr) {
			t.Fatalf("Compile() error is %T, want *ParseError", err)
		}
	})

	t.Run("check error propagates", func(t *testing.T) {
		t.Parallel()

		_, err := ordering.Compile("nope", bookSchema)

		var checkErr *ordering.CheckError
		if !errors.As(err, &checkErr) {
			t.Fatalf("Compile() error is %T, want *CheckError", err)
		}
	})
}
