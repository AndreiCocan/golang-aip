// Package ordering parses and validates order_by expressions, the
// `string order_by` field of List requests in resource-oriented APIs.
//
// The package is the backend-neutral base of a two-layer design:
//
//   - This package turns an order_by string into a checked list of
//     ordering keys.
//   - Separate dialect packages, such as [ordering/sqlite], translate
//     that list into a storage backend's query language. The checked
//     order_by is the contract between the two layers; anyone can
//     implement a dialect for another backend against it.
//
// The pipeline is [Parse] (string to syntactic tree), then [Check]
// (syntactic tree to validated keys, resolved against a [Schema]);
// [Compile] runs both. The schema declares the orderable field paths and
// doubles as the allowlist: order_bys referencing undeclared fields fail
// Check.
//
//	schema := ordering.NewSchema(
//		"display_name",
//		"create_time",
//		"author.name",
//	)
//	checked, err := ordering.Compile(req.GetOrderBy(), schema)
//
// A [Checked] order_by is a list of [Field] keys, each a dotted path with
// a direction. An empty order_by is valid, yields no keys, and means the
// service's default order.
//
// All errors for malformed or invalid order_bys match [ErrInvalidOrderBy]
// with [errors.Is] and carry the byte offset of the problem; services
// should surface them as an INVALID_ARGUMENT response.
//
// # Supported syntax
//
// An order_by is a comma-separated list of fields: "foo,bar". Ascending
// is the default; a "desc" suffix (matched case-insensitively) reverses a
// field: "foo, bar desc". Redundant whitespace is insignificant, so
// "foo, bar desc", " foo , bar desc ", and "foo,bar desc" are equivalent.
// Subfields are addressed with dots: "address.street".
//
// An explicit "asc" suffix is not part of the order_by syntax and is
// rejected. The "desc" keyword is positional: a field named "desc"
// remains referencable, and "desc desc" orders it descending.
//
// [Check] merges exact duplicate keys (same path, same direction) into
// their first occurrence, and rejects an order_by that sorts one path
// both ascending and descending. How each field's values compare (its
// natural comparator) is the storage backend's concern, so it lives in
// the dialect packages.
package ordering
