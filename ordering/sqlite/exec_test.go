package sqlite_test

import (
	"database/sql"
	"testing"

	"github.com/google/go-cmp/cmp"
	_ "modernc.org/sqlite"

	"github.com/AndreiCocan/golang-aip/ordering"
	"github.com/AndreiCocan/golang-aip/ordering/sqlite"
)

// newBookDB seeds an in-memory database with fixture books. The NULL
// rating marks an unset field, matching how a service stores absent
// optional data.
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
			id         INTEGER PRIMARY KEY,
			title      TEXT    NOT NULL,
			page_count INTEGER NOT NULL,
			rating     REAL,
			author     TEXT    NOT NULL
		)`
	if _, err := db.ExecContext(t.Context(), schema); err != nil {
		t.Fatalf("creating table: %v", err)
	}

	rows := [][]any{
		{1, "war and peace", 1225, 4.5, `{"name": "Tolstoy"}`},
		{2, "anna karenina", 864, nil, `{"name": "Tolstoy"}`},
		{3, "hamlet", 160, 4.0, `{"name": "Shakespeare"}`},
		{4, "ulysses", 730, 4.0, `{"name": "Joyce"}`},
	}
	for _, row := range rows {
		_, err := db.ExecContext(
			t.Context(),
			"INSERT INTO books (id, title, page_count, rating, author) VALUES (?, ?, ?, ?, ?)",
			row...,
		)
		if err != nil {
			t.Fatalf("inserting fixture row: %v", err)
		}
	}

	return db
}

func TestTranspile_execution(t *testing.T) {
	t.Parallel()

	schema := ordering.NewSchema("title", "page_count", "rating", "author.name")

	tests := []struct {
		name    string
		orderBy string
		opts    []sqlite.Option
		wantIDs []int
	}{
		{"ascending is the default", "page_count", nil, []int{3, 4, 2, 1}},
		{"descending", "page_count desc", nil, []int{1, 2, 4, 3}},
		{"strings order lexicographically", "title", nil, []int{2, 3, 4, 1}},
		{
			"secondary key breaks ties",
			"rating desc, title",
			nil,
			[]int{1, 3, 4, 2},
		},
		{
			"null sorts as the smallest value",
			"rating, title",
			nil,
			[]int{2, 3, 4, 1},
		},
		{
			"nested field via column expr",
			"author.name desc, page_count",
			[]sqlite.Option{
				sqlite.ColumnExpr("author.name", "json_extract(author, '$.name')"),
			},
			[]int{2, 1, 3, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := newBookDB(t)

			checked, err := ordering.Compile(tt.orderBy, schema)
			if err != nil {
				t.Fatalf("Compile(%q) error: %v", tt.orderBy, err)
			}

			frag, err := sqlite.Transpile(checked, tt.opts...)
			if err != nil {
				t.Fatalf("Transpile(%q) error: %v", tt.orderBy, err)
			}

			rows, err := db.QueryContext(
				t.Context(),
				"SELECT id FROM books ORDER BY "+frag,
			)
			if err != nil {
				t.Fatalf("querying with fragment %q: %v", frag, err)
			}

			defer func() {
				if err := rows.Close(); err != nil {
					t.Errorf("closing rows: %v", err)
				}
			}()

			var gotIDs []int

			for rows.Next() {
				var id int
				if err := rows.Scan(&id); err != nil {
					t.Fatalf("scanning row: %v", err)
				}

				gotIDs = append(gotIDs, id)
			}

			if err := rows.Err(); err != nil {
				t.Fatalf("iterating rows: %v", err)
			}

			if diff := cmp.Diff(tt.wantIDs, gotIDs); diff != "" {
				t.Errorf("order_by %q row order mismatch (-want +got):\n%s", tt.orderBy, diff)
			}
		})
	}
}
