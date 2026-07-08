// Package sqlite translates checked order_by expressions into SQLite
// ORDER BY clauses. It is the reference dialect for
// [github.com/AndreiCocan/golang-aip/ordering]: a pure string builder
// with no driver dependency, usable with any database/sql SQLite driver.
//
// [Transpile] renders an [ordering.Checked] order_by as an ORDER BY
// fragment, without the ORDER BY keyword. Top-level fields map to
// identically named columns; [Column] and [ColumnExpr] override the
// mapping for renamed columns, nested paths, and JSON-backed fields.
// Nested paths without a mapping fail with errors matching
// [ErrUnsupported].
//
// SQLite's natural comparators apply: numeric columns order numerically,
// text columns lexicographically by their collation, and NULL sorts as
// the smallest value, first under ascending and last under descending
// order.
package sqlite
