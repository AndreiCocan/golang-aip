package pagination

import "errors"

// ErrInvalidPageToken is bad client input. [ParseToken] returns it for a page
// token that is malformed, damaged, or made for a different request.
// [Token.Cursor] returns it when the cursor has a different shape than the
// destination.
var ErrInvalidPageToken = errors.New("invalid page token")

// ErrInvalidPageSize is returned by [ResolvePageSize] for a negative requested
// page size, which is bad client input. ResolvePageSize also returns it for
// a configuration with a default that is not positive or a negative maximum.
var ErrInvalidPageSize = errors.New("invalid page size")

// ErrInvalidCursor is a programming error in the service, never bad client
// input. [Token.Next] returns it for a cursor value that it cannot encode.
// [Token.Cursor] returns it for a destination that it cannot decode into.
var ErrInvalidCursor = errors.New("invalid cursor")
