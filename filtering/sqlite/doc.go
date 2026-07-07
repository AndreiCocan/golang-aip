// Package sqlite translates checked AIP-160 filters into SQLite WHERE
// clauses. It is the reference dialect for
// [github.com/AndreiCocan/golang-aip/filtering]: a pure string builder
// with no driver dependency, usable with any database/sql SQLite driver.
//
// [Transpile] renders a [filtering.Checked] filter as a WHERE-clause
// fragment with ? placeholders plus its bind arguments. Top-level fields
// map to identically named columns; [Column] and [ColumnExpr] override the
// mapping for renamed columns, nested paths, and JSON-backed maps. Bare
// search terms match as case-insensitive substrings of the columns
// registered with [SearchColumns] and [SearchColumnExpr]. Constructs
// SQLite cannot express (search terms without registered search columns,
// pass-through functions, containment on repeated fields) fail with
// errors matching [ErrUnsupported].
//
// Timestamps are bound as RFC 3339 UTC text by default (see [TimeLayout]),
// booleans as 0/1, durations as seconds, and wildcard patterns as
// case-sensitive GLOB matches. Because unset columns are SQL NULL, a
// comparison on an unset field never matches, including !=, which is
// exactly the AIP traversal semantics for unset fields.
package sqlite
