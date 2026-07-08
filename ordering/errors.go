package ordering

import (
	"errors"
	"fmt"
)

// ErrInvalidOrderBy is the sentinel matched by every error returned for an
// order_by that is syntactically malformed or fails schema checking.
// Services should map errors matching this sentinel to an INVALID_ARGUMENT
// response:
//
//	if errors.Is(err, ordering.ErrInvalidOrderBy) {
//		return nil, status.Error(codes.InvalidArgument, err.Error())
//	}
var ErrInvalidOrderBy = errors.New("invalid order by")

// ParseError reports a syntax error in an order_by. It matches
// [ErrInvalidOrderBy] with [errors.Is] and carries the byte offset of the
// offending token for precise user-facing messages.
type ParseError struct {
	// OrderBy is the complete order_by being parsed.
	OrderBy string
	// Pos is the byte offset in OrderBy where the error was detected.
	Pos int
	// Message describes the error, without position information.
	Message string
}

// Error returns the message with its position, prefixed by
// [ErrInvalidOrderBy].
func (e *ParseError) Error() string {
	return fmt.Sprintf("%v: %s at position %d", ErrInvalidOrderBy, e.Message, e.Pos)
}

// Unwrap makes the error match [ErrInvalidOrderBy].
func (e *ParseError) Unwrap() error { return ErrInvalidOrderBy }

// CheckError reports an order_by that parsed but failed validation against
// a [Schema]: an unknown field or a field ordered in two contradictory
// directions. It matches [ErrInvalidOrderBy] with [errors.Is] and carries
// the byte offset of the offending field.
type CheckError struct {
	// OrderBy is the complete order_by being checked, when known.
	OrderBy string
	// Pos is the byte offset in OrderBy where the error was detected.
	Pos int
	// Message describes the error, without position information.
	Message string
}

// Error returns the message with its position, prefixed by
// [ErrInvalidOrderBy].
func (e *CheckError) Error() string {
	return fmt.Sprintf("%v: %s at position %d", ErrInvalidOrderBy, e.Message, e.Pos)
}

// Unwrap makes the error match [ErrInvalidOrderBy].
func (e *CheckError) Unwrap() error { return ErrInvalidOrderBy }
