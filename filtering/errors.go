package filtering

import (
	"errors"
	"fmt"
)

// ErrInvalidFilter matches, with [errors.Is], every error that reports a
// filter with a syntax error or a filter that does not match its schema.
// Such an error is bad client input. An error that does not match
// ErrInvalidFilter comes from an [Expander] of the service. Check returns
// it without a change.
var ErrInvalidFilter = errors.New("invalid filter")

// ParseError reports a syntax error in a filter. It matches
// [ErrInvalidFilter] with [errors.Is]. It holds the byte offset of the
// token with the error, for precise messages to the user.
type ParseError struct {
	// Filter is the complete filter being parsed.
	Filter string
	// Pos is the byte offset in Filter where the error was detected.
	Pos int
	// Message describes the error, without position information.
	Message string
}

// Error returns the message with its position, prefixed by
// [ErrInvalidFilter].
func (e *ParseError) Error() string {
	return fmt.Sprintf("%v: %s at position %d", ErrInvalidFilter, e.Message, e.Pos)
}

// Unwrap makes the error match [ErrInvalidFilter].
func (e *ParseError) Unwrap() error { return ErrInvalidFilter }

// CheckError reports a filter that parsed but does not match its [Schema]:
// an unknown field, a type mismatch, or an operation that the type does
// not support. It matches [ErrInvalidFilter] with [errors.Is]. It holds
// the byte offset of the token with the error.
type CheckError struct {
	// Filter is the complete filter being checked, when known.
	Filter string
	// Pos is the byte offset in Filter where the error was detected.
	Pos int
	// Message describes the error, without position information.
	Message string
}

// Error returns the message with its position, prefixed by
// [ErrInvalidFilter].
func (e *CheckError) Error() string {
	return fmt.Sprintf("%v: %s at position %d", ErrInvalidFilter, e.Message, e.Pos)
}

// Unwrap makes the error match [ErrInvalidFilter].
func (e *CheckError) Unwrap() error { return ErrInvalidFilter }
