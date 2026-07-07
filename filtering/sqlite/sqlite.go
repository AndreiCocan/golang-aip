package sqlite

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AndreiCocan/golang-aip/filtering"
)

// ErrUnsupported is the sentinel matched by every error for a filter that
// checked successfully but uses constructs this dialect cannot express in
// SQLite: search terms without configured search columns, pass-through
// functions, repeated-field containment, or field paths without a column
// mapping. Services usually surface it like [filtering.ErrInvalidFilter],
// as an INVALID_ARGUMENT response.
var ErrUnsupported = errors.New("sqlite: unsupported filter")

// defaultTimeLayout is how timestamp arguments are formatted: RFC 3339
// seconds in UTC, which compares correctly as text when the column stores
// the same layout.
const defaultTimeLayout = "2006-01-02T15:04:05Z"

// Option configures a [Transpile] call.
type Option func(*transpiler)

// Column maps the filter field path (dotted, like "author.name") to a
// column name. Without a mapping only top-level scalar fields are
// rendered, as identically named columns.
func Column(path, column string) Option {
	return func(t *transpiler) { t.columns[path] = quoteIdent(column) }
}

// ColumnExpr maps the filter field path to a raw SQL expression, inserted
// verbatim, for example a json_extract call for a map field. The
// expression must be a scalar; it is the caller's responsibility that it
// is well-formed SQL.
func ColumnExpr(path, expr string) Option {
	return func(t *transpiler) { t.columns[path] = expr }
}

// SearchColumns registers columns as search targets, unlocking bare
// search terms: each term matches when it appears as a substring of at
// least one target. Matching is case-insensitive for ASCII via SQLite's
// default LIKE; a connection with PRAGMA case_sensitive_like on changes
// that. Calls accumulate, with [SearchColumnExpr]. Services restricting
// search to certain columns like this must document which fields they
// consider.
func SearchColumns(cols ...string) Option {
	return func(t *transpiler) {
		for _, col := range cols {
			t.searchTargets = append(t.searchTargets, quoteIdent(col))
		}
	}
}

// SearchColumnExpr registers a raw SQL expression as a search target,
// inserted verbatim, for example a json_extract call for a JSON-backed
// field. The expression must be a scalar; it is the caller's
// responsibility that it is well-formed SQL. Calls accumulate, with
// [SearchColumns].
func SearchColumnExpr(expr string) Option {
	return func(t *transpiler) { t.searchTargets = append(t.searchTargets, expr) }
}

// TimeLayout sets the [time.Time.Format] layout for timestamp arguments,
// which are always rendered in UTC. The default is RFC 3339 seconds,
// "2006-01-02T15:04:05Z". The layout must match how the column stores
// timestamps, or comparisons will silently misbehave.
func TimeLayout(layout string) Option {
	return func(t *transpiler) { t.timeLayout = layout }
}

// Transpile renders a checked filter as a SQLite WHERE-clause fragment
// with ? placeholders, returning the fragment and its arguments. An empty
// filter yields an empty fragment and no arguments:
//
//	frag, args, err := sqlite.Transpile(checked)
//	...
//	if frag != "" {
//		query += " WHERE " + frag
//	}
//	rows, err := db.Query(query, args...)
//
// By default each top-level field maps to the identically named column;
// [Column] and [ColumnExpr] override that, and are required for nested
// paths, which have no natural column. Wildcard patterns render as GLOB
// (case-sensitive, like the = they refine), booleans as 0/1, durations as
// seconds, and null comparisons and presence tests as IS [NOT] NULL. SQL
// null semantics then match AIP filtering: a comparison on an unset
// (NULL) column does not match, even under !=. Bare search terms render
// as case-insensitive substring matches over the columns registered with
// [SearchColumns] and [SearchColumnExpr]; every term must match at
// least one of them.
//
// Filters using constructs SQLite cannot express fail with an error
// matching [ErrUnsupported].
func Transpile(checked *filtering.Checked, opts ...Option) (string, []any, error) {
	if checked == nil || checked.Expr == nil {
		return "", nil, nil
	}

	t := &transpiler{
		columns:    make(map[string]string),
		timeLayout: defaultTimeLayout,
	}
	for _, opt := range opts {
		opt(t)
	}

	if err := t.expr(checked.Expr); err != nil {
		return "", nil, err
	}

	return t.sql.String(), t.args, nil
}

type transpiler struct {
	columns       map[string]string
	searchTargets []string
	timeLayout    string
	sql           strings.Builder
	args          []any
}

func unsupportedf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrUnsupported, fmt.Sprintf(format, args...))
}

func (t *transpiler) expr(e filtering.Expr) error {
	switch e := e.(type) {
	case *filtering.And:
		return t.junction(e.Operands, " AND ")
	case *filtering.Or:
		return t.junction(e.Operands, " OR ")
	case *filtering.Not:
		t.sql.WriteString("NOT (")

		if err := t.expr(e.Operand); err != nil {
			return err
		}

		t.sql.WriteString(")")

		return nil
	case *filtering.Comparison:
		return t.comparison(e)
	case *filtering.Search:
		return t.search(e)
	default:
		return unsupportedf("unknown filter node %T", e)
	}
}

func (t *transpiler) junction(operands []filtering.Expr, sep string) error {
	t.sql.WriteString("(")

	for i, op := range operands {
		if i > 0 {
			t.sql.WriteString(sep)
		}

		if err := t.expr(op); err != nil {
			return err
		}
	}

	t.sql.WriteString(")")

	return nil
}

// search renders a Search node: every term must match at least one
// search target, so terms join with AND and, within a term, targets with
// OR. A term matching on a NULL target column yields SQL NULL, so the
// row does not match, including under NOT, consistent with the
// package's unset-field semantics.
func (t *transpiler) search(s *filtering.Search) error {
	if len(t.searchTargets) == 0 {
		return unsupportedf(
			"search terms (%s) require SearchColumns or SearchColumnExpr",
			strings.Join(s.Terms, " "),
		)
	}

	if len(s.Terms) > 1 {
		t.sql.WriteString("(")
	}

	for i, term := range s.Terms {
		if i > 0 {
			t.sql.WriteString(" AND ")
		}

		t.searchTerm(term)
	}

	if len(s.Terms) > 1 {
		t.sql.WriteString(")")
	}

	return nil
}

// searchTerm renders one term as an OR across the search targets, each a
// substring LIKE with the term's %, _, and \ escaped.
func (t *transpiler) searchTerm(term string) {
	if len(t.searchTargets) > 1 {
		t.sql.WriteString("(")
	}

	for i, target := range t.searchTargets {
		if i > 0 {
			t.sql.WriteString(" OR ")
		}

		t.sql.WriteString(target)
		t.sql.WriteString(` LIKE ? ESCAPE '\'`)
		t.args = append(t.args, "%"+likeEscape(term)+"%")
	}

	if len(t.searchTargets) > 1 {
		t.sql.WriteString(")")
	}
}

// likeEscape escapes the LIKE metacharacters in a search term so it
// matches literally.
func likeEscape(term string) string {
	term = strings.ReplaceAll(term, `\`, `\\`)
	term = strings.ReplaceAll(term, `%`, `\%`)

	return strings.ReplaceAll(term, `_`, `\_`)
}

func (t *transpiler) comparison(c *filtering.Comparison) error {
	field, ok := c.Left.(*filtering.Field)
	if !ok {
		if call, isCall := c.Left.(*filtering.FuncCall); isCall {
			return unsupportedf("function %q has no SQLite translation", call.Name)
		}

		return unsupportedf("unknown comparison operand %T", c.Left)
	}

	column, err := t.column(field, c.Op)
	if err != nil {
		return err
	}

	switch c.Right.Kind {
	case filtering.KindNull:
		t.sql.WriteString(column)

		if c.Op == filtering.OpEquals {
			t.sql.WriteString(" IS NULL")
		} else {
			t.sql.WriteString(" IS NOT NULL")
		}

		return nil
	case filtering.KindStar:
		t.sql.WriteString(column)
		t.sql.WriteString(" IS NOT NULL")

		return nil
	case filtering.KindPattern:
		t.sql.WriteString(column)

		if c.Op == filtering.OpNotEquals {
			t.sql.WriteString(" NOT GLOB ?")
		} else {
			t.sql.WriteString(" GLOB ?")
		}

		t.args = append(t.args, globPattern(c.Right.Pattern))

		return nil
	default:
		// Scalar values render as a plain bind argument.
	}

	t.sql.WriteString(column)
	t.sql.WriteString(" ")
	t.sql.WriteString(sqlOperator(c.Op))
	t.sql.WriteString(" ?")
	t.args = append(t.args, t.arg(c.Right))

	return nil
}

// column resolves a field path to its SQL rendering. Mappings registered
// with [Column] or [ColumnExpr] win; an unmapped top-level field renders
// as the identically named column unless it is a repeated or map field,
// which no flat column holds. The has operator additionally rejects
// containment tests, which SQLite cannot express on a scalar column.
func (t *transpiler) column(field *filtering.Field, op filtering.Operator) (string, error) {
	kind := field.Type().Kind
	if op == filtering.OpHas && (kind == filtering.KindRepeated || kind == filtering.KindMap) {
		return "", unsupportedf(
			"containment on field %q; only the presence test %s:* is supported, with a column mapping",
			field.Path(),
			field.Path(),
		)
	}

	path := field.Path()
	if column, ok := t.columns[path]; ok {
		return column, nil
	}

	if len(field.Segments) > 1 {
		return "", unsupportedf("nested field %q requires a Column or ColumnExpr mapping", path)
	}

	if kind == filtering.KindRepeated || kind == filtering.KindMap {
		return "", unsupportedf("%v field %q requires a ColumnExpr mapping", kind, path)
	}

	return quoteIdent(path), nil
}

// arg converts a checked value to a bind argument. Booleans become 0/1,
// enums their name, timestamps text in the configured layout, and
// durations seconds.
func (t *transpiler) arg(v filtering.Value) any {
	switch v.Kind {
	case filtering.KindInt:
		return v.Int
	case filtering.KindFloat:
		return v.Float
	case filtering.KindBool:
		if v.Bool {
			return int64(1)
		}

		return int64(0)
	case filtering.KindTimestamp:
		return v.Time.UTC().Format(t.timeLayout)
	case filtering.KindDuration:
		return v.Duration.Seconds()
	default: // KindString, KindEnum
		return v.Str
	}
}

// globPattern renders pattern parts as a GLOB pattern, escaping the GLOB
// metacharacters *, ?, and [ in literal parts with character classes.
func globPattern(parts []filtering.PatternPart) string {
	var b strings.Builder

	for _, p := range parts {
		if p.Wildcard {
			b.WriteString("*")

			continue
		}

		for _, r := range p.Literal {
			switch r {
			case '*', '?', '[':
				b.WriteString("[")
				b.WriteRune(r)
				b.WriteString("]")
			default:
				b.WriteRune(r)
			}
		}
	}

	return b.String()
}

func sqlOperator(op filtering.Operator) string {
	switch op {
	case filtering.OpNotEquals:
		return "<>"
	case filtering.OpLess:
		return "<"
	case filtering.OpLessEquals:
		return "<="
	case filtering.OpGreater:
		return ">"
	case filtering.OpGreaterEquals:
		return ">="
	default: // OpEquals, and OpHas on map values and repeated crossings
		return "="
	}
}

// quoteIdent quotes a SQL identifier.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
