// Package pagination mints and parses the opaque page tokens of AIP-158
// pagination, the page_size, page_token, and next_page_token fields of
// List requests in resource-oriented APIs.
//
// The package is a token codec and a page-size policy; it issues no
// queries. The service keeps a keyset cursor, a small struct holding the
// ordering key of the last row it served, and pages by seeking past it.
// [Parse] decodes a request's page_token into a [Token], [Token.Cursor]
// recovers the cursor to seek from, and [Token.Next] mints the
// next_page_token from the last row of the page being served. [PageSize]
// resolves the request's page_size against the service's default and
// maximum.
//
//	token, err := pagination.Parse(req.GetPageToken(), req.GetFilter())
//	size, err := pagination.PageSize(req.GetPageSize(), 25, 1000)
//
//	var cur bookCursor
//	ok, err := token.Cursor(&cur)
//	// Seek past cur when ok. Fetch size+1 rows and serve size of them:
//	// an extra row means another page exists, so mint its token from the
//	// last row served; otherwise return "" as the next_page_token.
//	next, err := token.Next(bookCursor{PublishTime: last.PublishTime, ID: last.ID})
//
// The cursor may be any gob-encodable Go value. Its shape is private to
// the service: tokens are opaque to clients and only the service that
// minted one decodes it. They are tamper-evident, not encrypted, so a
// cursor must not carry secrets. The checksums are unkeyed and catch
// accidents, not forgery, so a token must not carry authority either:
// the service must authorize every request independently of its token.
//
// AIP-158 requires a token to fail when request arguments it depends on
// change between pages. The requestArgs passed to [Parse] are checksummed
// into every token minted from it, so replaying a token under a different
// parent, filter, or order_by fails with [ErrInvalidPageToken]. Leave
// page_size out of the args, and skip too when the service implements
// AIP-158 skip: both may change between pages. Changing the service's
// cursor struct invalidates outstanding tokens the same way; to also
// invalidate them on changes the checksums cannot see, such as reordering
// an unchanged struct's meaning, include a version constant in the args.
//
// [ErrInvalidPageToken] and [ErrInvalidPageSize] report bad client input;
// services should surface them as an INVALID_ARGUMENT response. Every
// token failure, whether corruption, changed request arguments, or a
// changed cursor shape, is deliberately the same bare
// [ErrInvalidPageToken], because its message reaches clients; when all
// outstanding tokens fail after a deploy, suspect a cursor or argument
// change on the service side.
// [ErrInvalidCursor] reports a bug in the service itself and should not
// be mapped to a client error.
package pagination
