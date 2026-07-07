// Package resourcename parses, formats, and validates resource names in
// resource-oriented APIs.
//
// A resource name is a slash-separated path such as
// "publishers/123/books/les-miserables", optionally prefixed by a service
// host in the full form "//library.example.com/publishers/123". A pattern is
// a template for such names, with snake_case variables in braces:
// "publishers/{publisher}/books/{book}".
//
// Validate concrete names with [Validate] and patterns with
// [ValidatePattern]; failures match [ErrInvalidName] and [ErrInvalidPattern]
// respectively. Parse names with [Sscan], format them with [Sprint], and
// test them against a pattern with [Match]. [Join], [HasParent], [Ancestor],
// and [Parents] manipulate the hierarchy, and [ContainsWildcard] detects the
// "-" segment used when reading across collections. The low-level [Scanner]
// and [Segment] types walk a name without allocating.
package resourcename
