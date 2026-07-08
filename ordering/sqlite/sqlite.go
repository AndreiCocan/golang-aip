package sqlite

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AndreiCocan/golang-aip/ordering"
)

// ErrUnsupported is the sentinel matched by every error for an order_by
// that checked successfully but uses constructs this dialect cannot
// express in SQLite: field paths without a column mapping. Services
// usually surface it like [ordering.ErrInvalidOrderBy], as an
// INVALID_ARGUMENT response.
var ErrUnsupported = errors.New("sqlite: unsupported order by")

// Option configures a [Transpile] call.
type Option func(*transpiler)

// Column maps the ordering field path (dotted, like "author.name") to a
// column name. Without a mapping only top-level fields are rendered, as
// identically named columns.
func Column(path, column string) Option {
	return func(t *transpiler) { t.columns[path] = quoteIdent(column) }
}

// ColumnExpr maps the ordering field path to a raw SQL expression,
// inserted verbatim, for example a json_extract call for a JSON-backed
// field. The expression must be a scalar; it is the caller's
// responsibility that it is well-formed SQL.
func ColumnExpr(path, expr string) Option {
	return func(t *transpiler) { t.columns[path] = expr }
}

// Transpile renders a checked order_by as a SQLite ORDER BY fragment,
// without the ORDER BY keyword. An empty order_by yields an empty
// fragment:
//
//	frag, err := sqlite.Transpile(checked)
//	...
//	if frag != "" {
//		query += " ORDER BY " + frag
//	}
//
// By default each top-level field maps to the identically named column;
// [Column] and [ColumnExpr] override that, and are required for nested
// paths, which have no natural column. Descending keys render with DESC;
// ascending keys carry no suffix. NULLs follow SQLite's default placement:
// NULL sorts as the smallest value, first under ascending and last under
// descending order.
//
// Order_bys using field paths without a column mapping fail with an error
// matching [ErrUnsupported].
func Transpile(checked *ordering.Checked, opts ...Option) (string, error) {
	if checked == nil || len(checked.Fields) == 0 {
		return "", nil
	}

	t := &transpiler{columns: make(map[string]string)}
	for _, opt := range opts {
		opt(t)
	}

	var sql strings.Builder

	for i, f := range checked.Fields {
		if i > 0 {
			sql.WriteString(", ")
		}

		column, err := t.column(&f)
		if err != nil {
			return "", err
		}

		sql.WriteString(column)

		if f.Desc {
			sql.WriteString(" DESC")
		}
	}

	return sql.String(), nil
}

type transpiler struct {
	columns map[string]string
}

// column resolves a field path to its SQL rendering. Mappings registered
// with [Column] or [ColumnExpr] win; an unmapped top-level field renders
// as the identically named column, and an unmapped nested path fails,
// since no flat column holds it.
func (t *transpiler) column(field *ordering.Field) (string, error) {
	path := field.Path()
	if column, ok := t.columns[path]; ok {
		return column, nil
	}

	if len(field.Segments) > 1 {
		return "", fmt.Errorf(
			"%w: nested field %q requires a Column or ColumnExpr mapping",
			ErrUnsupported,
			path,
		)
	}

	return quoteIdent(path), nil
}

// quoteIdent quotes a SQL identifier.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
