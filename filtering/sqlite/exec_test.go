package sqlite_test

import (
	"database/sql"
	"testing"

	"github.com/google/go-cmp/cmp"
	_ "modernc.org/sqlite"

	"github.com/AndreiCocan/golang-aip/filtering"
	"github.com/AndreiCocan/golang-aip/filtering/sqlite"
)

// newBookDB seeds an in-memory database with fixture books. NULLs mark
// unset fields, matching how a service stores absent optional data.
func newBookDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening in-memory sqlite: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing database: %v", err)
		}
	})
	// Every pooled connection would get its own empty in-memory database;
	// pin the pool to one connection so all queries see the fixtures.
	db.SetMaxOpenConns(1)

	const schema = `
		CREATE TABLE books (
			id           INTEGER PRIMARY KEY,
			display_name TEXT    NOT NULL,
			page_count   INTEGER NOT NULL,
			rating       REAL,
			published    INTEGER NOT NULL,
			create_time  TEXT,
			read_time    REAL,
			state        TEXT    NOT NULL,
			author_name  TEXT,
			labels       TEXT
		)`
	if _, err := db.ExecContext(t.Context(), schema); err != nil {
		t.Fatalf("creating table: %v", err)
	}

	rows := [][]any{
		{
			1,
			"war and peace",
			1225,
			4.5,
			1,
			"2020-06-01T00:00:00Z",
			7200.0,
			"ACTIVE",
			"Tolstoy",
			`{"env":"prod"}`,
		},
		{
			2,
			"hunt for red october",
			656,
			3.9,
			1,
			"2021-03-01T00:00:00Z",
			5400.0,
			"DRAFT",
			"Clancy",
			`{"env":"dev"}`,
		},
		{3, "readme.md", 1, nil, 0, nil, nil, "DELETED", nil, nil},
		{4, "what?.md", 10, 2.0, 0, "2022-01-01T00:00:00Z", 60.0, "DRAFT", nil, nil},
		{5, "whatX.md", 5, 1.0, 0, "2019-01-01T00:00:00Z", 30.0, "DRAFT", nil, nil},
	}
	for _, r := range rows {
		if _, err := db.ExecContext(
			t.Context(),
			`INSERT INTO books VALUES (?,?,?,?,?,?,?,?,?,?)`,
			r...); err != nil {
			t.Fatalf("seeding row %v: %v", r[0], err)
		}
	}

	return db
}

// TestTranspileExecution runs transpiled filters against a real SQLite
// database and asserts which fixture rows match, proving the generated
// SQL means what the filter means, not just that it looks right.
func TestTranspile_execution(t *testing.T) {
	t.Parallel()

	opts := []sqlite.Option{
		sqlite.Column("author.name", "author_name"),
		// The author message has no column of its own; its presence is
		// its one subfield's presence.
		sqlite.Column("author", "author_name"),
		sqlite.ColumnExpr("labels.env", "json_extract(labels, '$.env')"),
		sqlite.SearchColumns("display_name", "author_name"),
	}

	db := newBookDB(t)
	for _, tt := range []struct {
		name    string
		filter  string
		wantIDs []int
	}{
		{
			name:    "empty filter matches everything",
			filter:  "",
			wantIDs: []int{1, 2, 3, 4, 5},
		},
		{
			name:    "string equality",
			filter:  `display_name = "readme.md"`,
			wantIDs: []int{3},
		},
		{
			name:    "integer range",
			filter:  "page_count >= 10 AND page_count <= 700",
			wantIDs: []int{2, 4},
		},
		{
			name:    "float comparison skips null rows",
			filter:  "rating > 4.0",
			wantIDs: []int{1},
		},
		{
			name:    "not-equals skips null rows",
			filter:  "rating != 4.5",
			wantIDs: []int{2, 4, 5},
		},
		{
			name:    "bool equality",
			filter:  "published = true",
			wantIDs: []int{1, 2},
		},
		{
			name:    "negation",
			filter:  "NOT published = true",
			wantIDs: []int{3, 4, 5},
		},
		{
			name:    "suffix wildcard",
			filter:  `display_name = "*.md"`,
			wantIDs: []int{3, 4, 5},
		},
		{
			name:    "glob metacharacter matches literally",
			filter:  `display_name = "what?.md"`,
			wantIDs: []int{4},
		},
		{
			name:    "wildcard inequality",
			filter:  `display_name != "*.md"`,
			wantIDs: []int{1, 2},
		},
		{
			name:    "timestamp comparison skips null rows",
			filter:  `create_time > "2021-01-01T00:00:00Z"`,
			wantIDs: []int{2, 4},
		},
		{
			name:    "timestamp presence",
			filter:  "create_time:*",
			wantIDs: []int{1, 2, 4, 5},
		},
		{
			name:    "null test",
			filter:  "create_time = null",
			wantIDs: []int{3},
		},
		{
			name:    "duration comparison",
			filter:  "read_time > 5400s",
			wantIDs: []int{1},
		},
		{
			name:    "enum disjunction",
			filter:  "state = ACTIVE OR state = DELETED",
			wantIDs: []int{1, 3},
		},
		{
			name:    "or binds tighter than and",
			filter:  "published = true AND state = ACTIVE OR state = DRAFT",
			wantIDs: []int{1, 2},
		},
		{
			name:    "mapped nested field",
			filter:  `author.name = "Tolstoy"`,
			wantIDs: []int{1},
		},
		{
			name:    "message presence",
			filter:  "author:*",
			wantIDs: []int{1, 2},
		},
		{
			name:    "negated message presence",
			filter:  "NOT author:*",
			wantIDs: []int{3, 4, 5},
		},
		{
			name:    "message null test",
			filter:  "author = null",
			wantIDs: []int{3, 4, 5},
		},
		{
			name:    "map value equality",
			filter:  "labels.env = prod",
			wantIDs: []int{1},
		},
		{
			name:    "map key presence",
			filter:  "labels:env",
			wantIDs: []int{1, 2},
		},
		{
			name:    "search matches case-insensitively",
			filter:  "TOLSTOY",
			wantIDs: []int{1},
		},
		{
			name:    "search matches any configured column",
			filter:  "red",
			wantIDs: []int{2},
		},
		{
			name:    "search terms may match different columns",
			filter:  "october Clancy",
			wantIDs: []int{2},
		},
		{
			name:    "search terms must all match somewhere",
			filter:  "war Clancy",
			wantIDs: nil,
		},
		{
			name:    "search matches despite other columns being null",
			filter:  "md",
			wantIDs: []int{3, 4, 5},
		},
		{
			name:    "negated search does not match null columns",
			filter:  "NOT Tolstoy",
			wantIDs: []int{2},
		},
		{
			name:    "search underscore matches literally",
			filter:  `"t_.md"`,
			wantIDs: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checked, err := filtering.Compile(tt.filter, bookSchema)
			if err != nil {
				t.Fatalf("Compile(%q) error = %v", tt.filter, err)
			}

			whereClause, args, err := sqlite.Transpile(checked, opts...)
			if err != nil {
				t.Fatalf("Transpile(%q) error = %v", tt.filter, err)
			}

			query := "SELECT id FROM books"
			if whereClause != "" {
				query += " WHERE "
				query += whereClause
			}

			query += " ORDER BY id"

			rows, err := db.QueryContext(t.Context(), query, args...)
			if err != nil {
				t.Fatalf("querying %q (from filter %q): %v", query, tt.filter, err)
			}

			defer func() {
				if err := rows.Close(); err != nil {
					t.Errorf("closing rows: %v", err)
				}
			}()

			var got []int

			for rows.Next() {
				var id int
				if err := rows.Scan(&id); err != nil {
					t.Fatalf("scanning: %v", err)
				}

				got = append(got, id)
			}

			if err := rows.Err(); err != nil {
				t.Fatalf("iterating: %v", err)
			}

			if diff := cmp.Diff(tt.wantIDs, got); diff != "" {
				t.Errorf("filter %q (query %q, args %v) matched wrong rows (-want +got):\n%s",
					tt.filter, query, args, diff)
			}
		})
	}
}
