// Package ordering parses and validates order_by expressions, the
// `string order_by` field of List requests in resource-oriented APIs.
//
// The package does not sort. It turns an order_by string into a checked
// list of sort keys. A dialect package then translates the list into the
// query language of one storage backend, such as an ORDER BY clause.
//
// [Parse] turns a string into a syntax tree. [Check] turns a syntax tree
// into a [CheckedOrderBy], validated against a [Schema]. [Compile] does
// both. The schema declares the field paths that an order_by can name. It
// is also the allowlist: Check rejects an order_by that names a field that
// the schema does not declare. [NewSchema] builds a schema from paths, and
// [SchemaFromTags] builds it from the aip struct tags of a domain type.
//
// A checked order_by is a list of [Key] values. Each key is a dotted field
// path with a direction. An empty order_by is valid and gives no keys. It
// means the default order of the service.
//
// Each error for a malformed or invalid order_by matches
// [ErrInvalidOrderBy] with [errors.Is], and holds the byte offset of the
// problem. Such an error is bad client input.
//
// # Supported syntax
//
// An order_by is a comma-separated list of fields: "foo,bar". The default
// direction is ascending. A "desc" suffix makes a field descending:
// "foo, bar desc". The suffix is case-insensitive. Extra whitespace has no
// effect, so "foo, bar desc", " foo , bar desc ", and "foo,bar desc" are
// equal. Dots name subfields: "address.street".
//
// An "asc" suffix is not part of the syntax, and Parse rejects it. Only the
// position of "desc" makes it a suffix: a field can have the name "desc",
// and "desc desc" sorts it descending.
//
// [Check] merges exact duplicate keys (same path, same direction) into the
// first one. It rejects an order_by that sorts one path both ascending and
// descending. How the values of a field compare is a property of the
// storage, so the dialect decides it.
package ordering
