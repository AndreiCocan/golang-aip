// Package resourcename parses, formats, and validates the resource names of
// resource-oriented APIs.
//
// A resource name is a path of segments with slashes between them, such as
// "publishers/123/books/les-miserables". A full resource name starts with
// "//" and a service name: "//library.example.com/publishers/123". A
// pattern is a template for resource names, with snake_case variables in
// braces: "publishers/{publisher}/books/{book}".
//
// [Validate] validates a resource name, and [ValidatePattern] validates a
// pattern. Their errors match [ErrInvalidName] and [ErrInvalidPattern].
// [Sscan] reads the variables of a name with a pattern, [Sprint] makes a
// name from a pattern, and [Match] compares a name with a pattern. [Join],
// [HasAncestor], [Ancestor], and [Parents] work on the hierarchy of a name.
// [ContainsWildcard] finds the "-" segment of a read across collections.
// [Scanner] and [Segment] read a name one segment at a time, without
// allocation.
package resourcename
