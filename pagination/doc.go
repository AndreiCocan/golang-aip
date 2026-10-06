// Package pagination mints and parses the opaque page tokens of AIP-158
// pagination, the page_size, page_token, and next_page_token fields of
// List requests in resource-oriented APIs.
//
// The package encodes tokens and applies a page size policy. It does not
// send queries. A token holds a cursor: a value that the service makes,
// such as the sort key of the last row of a page. The package has these
// entry points:
//
//   - [ParseToken] decodes the page_token of a request into a [Token].
//   - [Token.Cursor] decodes the cursor of the token.
//   - [Token.Next] makes the next_page_token from a cursor.
//   - [ResolvePageSize] applies the default and the maximum of the service
//     to the page_size of a request.
//
// The cursor can be any Go value that gob can encode. Its shape is private
// to the service. A token is opaque to clients, and only the service that
// made it decodes it. A token shows changes, but it is not encrypted, so a
// cursor must not hold secrets. The checksums have no key. They find
// accidents, not forgery. Thus a token must not give authority: the
// service must authorize each request without the token.
//
// AIP-158 requires a token to fail when a request argument that it uses
// changes between pages. [ParseToken] keeps a checksum of its requestArgs
// in the [Token], [Token.Next] writes it into the next token, and the next
// ParseToken compares it with the checksum of its own requestArgs. Thus a
// token that comes back with a different
// parent, filter, or order_by fails with [ErrInvalidPageToken]. Do not
// put page_size or skip in the arguments, because both can change between
// pages. A change to the cursor type of the service also makes the old
// tokens fail. For a change that the checksums cannot find, such as a new
// meaning of an unchanged type, put a version constant in the arguments.
//
// [ErrInvalidPageToken] and [ErrInvalidPageSize] report bad client input.
// Each token failure gives the same [ErrInvalidPageToken], because its
// message goes to the client: damage, changed request arguments, and a
// changed cursor shape look the same. When all old tokens fail after a
// deployment, look for a cursor or argument change in the service.
// [ErrInvalidCursor] reports a bug in the service, not bad client input.
package pagination
