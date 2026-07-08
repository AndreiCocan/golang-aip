package pagination

import "errors"

// ErrInvalidPageToken is returned by [Parse] for a page token that is
// malformed, corrupted, or minted for a different request, and by
// [Token.Cursor] when the cursor was encoded from a different shape than
// the one being decoded into. Services should map it to an
// INVALID_ARGUMENT response:
//
//	if errors.Is(err, pagination.ErrInvalidPageToken) {
//		return nil, status.Error(codes.InvalidArgument, err.Error())
//	}
var ErrInvalidPageToken = errors.New("invalid page token")

// ErrInvalidPageSize is returned by [PageSize] for a negative requested
// page size, and for a configuration with a non-positive default or a
// negative maximum. Services should map a negative request to an
// INVALID_ARGUMENT response.
var ErrInvalidPageSize = errors.New("invalid page size")

// ErrInvalidCursor reports a programming error in the service, never bad
// client input: a cursor value that cannot be encoded by [Token.Next], or
// a destination that cannot be decoded into by [Token.Cursor]. It should
// not be mapped to INVALID_ARGUMENT.
var ErrInvalidCursor = errors.New("invalid cursor")
