package sqlite_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/AndreiCocan/golang-aip/filtering"
	"github.com/AndreiCocan/golang-aip/filtering/sqlite"
)

// recentCutoff is the fixed timestamp the recent() test macro expands to.
var recentCutoff = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

var bookSchema = filtering.NewSchema(
	filtering.String("display_name"),
	filtering.Int("page_count"),
	filtering.Float("rating"),
	filtering.Bool("published"),
	filtering.Timestamp("create_time"),
	filtering.Duration("read_time"),
	filtering.Enum("state", "DRAFT", "ACTIVE", "DELETED"),
	filtering.Message("author",
		filtering.String("name"),
	),
	filtering.Repeated(filtering.String("tags")),
	filtering.Map(filtering.String("labels")),
	filtering.Function("recent",
		filtering.Expand(func(s *filtering.Schema, _ []filtering.Value) (filtering.Expr, error) {
			f, err := s.Field("create_time")
			if err != nil {
				return nil, err
			}

			return &filtering.Comparison{
				Left:  f,
				Op:    filtering.OpGreater,
				Right: filtering.TimestampValue(recentCutoff),
			}, nil
		}),
	),
	filtering.Function("hasPrefix",
		filtering.Args(filtering.KindString, filtering.KindString),
		filtering.Returns(filtering.KindBool),
	),
)

func TestTranspile(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name     string
		filter   string
		opts     []sqlite.Option
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "empty filter",
			filter:   "",
			wantSQL:  "",
			wantArgs: nil,
		},
		{
			name:     "string equality",
			filter:   `display_name = "war"`,
			wantSQL:  `"display_name" = ?`,
			wantArgs: []any{"war"},
		},
		{
			name:     "string inequality",
			filter:   `display_name != "war"`,
			wantSQL:  `"display_name" <> ?`,
			wantArgs: []any{"war"},
		},
		{
			name:     "integer comparison",
			filter:   "page_count > 100",
			wantSQL:  `"page_count" > ?`,
			wantArgs: []any{int64(100)},
		},
		{
			name:     "float comparison",
			filter:   "rating >= 4.5",
			wantSQL:  `"rating" >= ?`,
			wantArgs: []any{4.5},
		},
		{
			name:     "bool true is one",
			filter:   "published = true",
			wantSQL:  `"published" = ?`,
			wantArgs: []any{int64(1)},
		},
		{
			name:     "bool false is zero",
			filter:   "published != false",
			wantSQL:  `"published" <> ?`,
			wantArgs: []any{int64(0)},
		},
		{
			name:     "enum as its name",
			filter:   "state = ACTIVE",
			wantSQL:  `"state" = ?`,
			wantArgs: []any{"ACTIVE"},
		},
		{
			name:     "timestamp as RFC 3339 UTC text",
			filter:   `create_time > "2021-02-14T11:00:00+01:00"`,
			wantSQL:  `"create_time" > ?`,
			wantArgs: []any{"2021-02-14T10:00:00Z"},
		},
		{
			name:     "timestamp with custom layout",
			filter:   `create_time > "2021-02-14T10:00:00Z"`,
			opts:     []sqlite.Option{sqlite.TimeLayout("2006-01-02 15:04:05")},
			wantSQL:  `"create_time" > ?`,
			wantArgs: []any{"2021-02-14 10:00:00"},
		},
		{
			name:     "duration as seconds",
			filter:   "read_time > 90s",
			wantSQL:  `"read_time" > ?`,
			wantArgs: []any{90.0},
		},
		{
			name:     "fractional duration",
			filter:   "read_time <= 1.5s",
			wantSQL:  `"read_time" <= ?`,
			wantArgs: []any{1.5},
		},
		{
			name:     "and flattens",
			filter:   "page_count > 100 rating > 4.0 AND published = true",
			wantSQL:  `("page_count" > ? AND "rating" > ? AND "published" = ?)`,
			wantArgs: []any{int64(100), 4.0, int64(1)},
		},
		{
			name:     "or binds tighter than and",
			filter:   "published = true AND state = ACTIVE OR state = DRAFT",
			wantSQL:  `("published" = ? AND ("state" = ? OR "state" = ?))`,
			wantArgs: []any{int64(1), "ACTIVE", "DRAFT"},
		},
		{
			name:     "not",
			filter:   "NOT published = true",
			wantSQL:  `NOT ("published" = ?)`,
			wantArgs: []any{int64(1)},
		},
		{
			name:     "minus negation",
			filter:   "-published = true",
			wantSQL:  `NOT ("published" = ?)`,
			wantArgs: []any{int64(1)},
		},
		{
			name:     "suffix wildcard uses glob",
			filter:   `display_name = "*.foo"`,
			wantSQL:  `"display_name" GLOB ?`,
			wantArgs: []any{"*.foo"},
		},
		{
			name:     "wildcard inequality",
			filter:   `display_name != "war*"`,
			wantSQL:  `"display_name" NOT GLOB ?`,
			wantArgs: []any{"war*"},
		},
		{
			name:     "glob metacharacters escaped in literals",
			filter:   `display_name = "a?[b]*"`,
			wantSQL:  `"display_name" GLOB ?`,
			wantArgs: []any{"a[?][[]b]*"},
		},
		{
			name:     "null equality",
			filter:   "author = null",
			wantSQL:  `"author" IS NULL`,
			wantArgs: nil,
		},
		{
			name:     "null inequality",
			filter:   "create_time != null",
			wantSQL:  `"create_time" IS NOT NULL`,
			wantArgs: nil,
		},
		{
			name:     "presence test",
			filter:   "display_name:*",
			wantSQL:  `"display_name" IS NOT NULL`,
			wantArgs: nil,
		},
		{
			name:     "renamed column",
			filter:   `display_name = "war"`,
			opts:     []sqlite.Option{sqlite.Column("display_name", "title")},
			wantSQL:  `"title" = ?`,
			wantArgs: []any{"war"},
		},
		{
			name:     "mapped nested field",
			filter:   `author.name = "Hugo"`,
			opts:     []sqlite.Option{sqlite.Column("author.name", "author_name")},
			wantSQL:  `"author_name" = ?`,
			wantArgs: []any{"Hugo"},
		},
		{
			name:     "column expression",
			filter:   "labels.env = prod",
			opts:     []sqlite.Option{sqlite.ColumnExpr("labels.env", "json_extract(labels, '$.env')")},
			wantSQL:  `json_extract(labels, '$.env') = ?`,
			wantArgs: []any{"prod"},
		},
		{
			name:     "has map key with expression",
			filter:   "labels:env",
			opts:     []sqlite.Option{sqlite.ColumnExpr("labels.env", "json_extract(labels, '$.env')")},
			wantSQL:  `json_extract(labels, '$.env') IS NOT NULL`,
			wantArgs: nil,
		},
		{
			name:     "has map value with expression",
			filter:   "labels.env:prod",
			opts:     []sqlite.Option{sqlite.ColumnExpr("labels.env", "json_extract(labels, '$.env')")},
			wantSQL:  `json_extract(labels, '$.env') = ?`,
			wantArgs: []any{"prod"},
		},
		{
			name:     "macro function",
			filter:   "recent()",
			wantSQL:  `"create_time" > ?`,
			wantArgs: []any{"2021-01-01T00:00:00Z"},
		},
		{
			name:     "search term single column",
			filter:   "Hugo",
			opts:     []sqlite.Option{sqlite.SearchColumns("display_name")},
			wantSQL:  `"display_name" LIKE ? ESCAPE '\'`,
			wantArgs: []any{"%Hugo%"},
		},
		{
			name:     "search term multiple columns",
			filter:   "Hugo",
			opts:     []sqlite.Option{sqlite.SearchColumns("display_name", "author_name")},
			wantSQL:  `("display_name" LIKE ? ESCAPE '\' OR "author_name" LIKE ? ESCAPE '\')`,
			wantArgs: []any{"%Hugo%", "%Hugo%"},
		},
		{
			name:    "search terms all must match",
			filter:  "foo bar",
			opts:    []sqlite.Option{sqlite.SearchColumns("display_name", "author_name")},
			wantSQL: `(("display_name" LIKE ? ESCAPE '\' OR "author_name" LIKE ? ESCAPE '\')` + ` AND ("display_name" LIKE ? ESCAPE '\' OR "author_name" LIKE ? ESCAPE '\'))`,
			wantArgs: []any{
				"%foo%", "%foo%",
				"%bar%", "%bar%",
			},
		},
		{
			name:     "quoted phrase is one search term",
			filter:   `"war and peace"`,
			opts:     []sqlite.Option{sqlite.SearchColumns("display_name")},
			wantSQL:  `"display_name" LIKE ? ESCAPE '\'`,
			wantArgs: []any{"%war and peace%"},
		},
		{
			name:     "like metacharacters escaped in search terms",
			filter:   `"50%_off"`,
			opts:     []sqlite.Option{sqlite.SearchColumns("display_name")},
			wantSQL:  `"display_name" LIKE ? ESCAPE '\'`,
			wantArgs: []any{`%50\%\_off%`},
		},
		{
			name:     "negated search",
			filter:   "NOT Hugo",
			opts:     []sqlite.Option{sqlite.SearchColumns("display_name")},
			wantSQL:  `NOT ("display_name" LIKE ? ESCAPE '\')`,
			wantArgs: []any{"%Hugo%"},
		},
		{
			name:     "search column expression",
			filter:   "prod",
			opts:     []sqlite.Option{sqlite.SearchColumnExpr("json_extract(labels, '$.env')")},
			wantSQL:  `json_extract(labels, '$.env') LIKE ? ESCAPE '\'`,
			wantArgs: []any{"%prod%"},
		},
		{
			name:   "search columns and expressions accumulate",
			filter: "Hugo",
			opts: []sqlite.Option{
				sqlite.SearchColumns("display_name"),
				sqlite.SearchColumnExpr("json_extract(labels, '$.env')"),
			},
			wantSQL:  `("display_name" LIKE ? ESCAPE '\' OR json_extract(labels, '$.env') LIKE ? ESCAPE '\')`,
			wantArgs: []any{"%Hugo%", "%Hugo%"},
		},
		{
			name:     "search mixed with comparison",
			filter:   "Hugo published = true",
			opts:     []sqlite.Option{sqlite.SearchColumns("display_name")},
			wantSQL:  `("display_name" LIKE ? ESCAPE '\' AND "published" = ?)`,
			wantArgs: []any{"%Hugo%", int64(1)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checked, err := filtering.Compile(tt.filter, bookSchema)
			if err != nil {
				t.Fatalf("Compile(%q) error = %v", tt.filter, err)
			}

			gotSQL, gotArgs, err := sqlite.Transpile(checked, tt.opts...)
			if err != nil {
				t.Fatalf("Transpile(%q) error = %v", tt.filter, err)
			}

			if gotSQL != tt.wantSQL {
				t.Errorf("Transpile(%q) SQL = %s, want %s", tt.filter, gotSQL, tt.wantSQL)
			}

			if diff := cmp.Diff(tt.wantArgs, gotArgs); diff != "" {
				t.Errorf("Transpile(%q) args mismatch (-want +got):\n%s", tt.filter, diff)
			}
		})
	}

	t.Run("unsupported", func(t *testing.T) {
		t.Parallel()

		for _, tt := range []struct {
			name   string
			filter string
			opts   []sqlite.Option
		}{
			{name: "search terms without configured search columns", filter: "Hugo"},
			{name: "bare field name without search columns", filter: "published"},
			{name: "unmapped nested field", filter: `author.name = "x"`},
			{name: "unmapped map key", filter: "labels.env = prod"},
			{name: "repeated containment", filter: "tags:go"},
			{name: "repeated containment despite mapping", filter: "tags:go", opts: []sqlite.Option{sqlite.Column("tags", "tags")}},
			{name: "unmapped repeated presence", filter: "tags:*"},
			{name: "pass-through function", filter: `hasPrefix(display_name, "go")`},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				checked, err := filtering.Compile(tt.filter, bookSchema)
				if err != nil {
					t.Fatalf("Compile(%q) error = %v", tt.filter, err)
				}

				_, _, err = sqlite.Transpile(checked, tt.opts...)
				if err == nil {
					t.Fatalf("Transpile(%q) succeeded, want error", tt.filter)
				}

				if !errors.Is(err, sqlite.ErrUnsupported) {
					t.Errorf("Transpile(%q) error = %v, want ErrUnsupported", tt.filter, err)
				}
			})
		}
	})
}
