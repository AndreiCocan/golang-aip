package ordering

import (
	"errors"
	"fmt"
)

// ErrInvalidOrderBy matches, with [errors.Is], every error that reports an
// order_by with a syntax error or an order_by that does not match its
// schema. Such an error is bad client input.
var ErrInvalidOrderBy = errors.New("invalid order by")

// ParseError reports a syntax error in an order_by. It matches
// [ErrInvalidOrderBy] with [errors.Is]. It holds the byte offset of the
// token with the error, for precise messages to the user.
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

// CheckError reports an order_by that parsed but does not match its
// [Schema]: an unknown field, or a field in two opposite directions. It
// matches [ErrInvalidOrderBy] with [errors.Is]. It holds the byte offset
// of the field with the error.
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
