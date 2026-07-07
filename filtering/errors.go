package filtering

import (
	"errors"
	"fmt"
)

// ErrInvalidFilter is the sentinel matched by every error returned for a
// filter that is syntactically malformed or fails schema checking. Services
// should map errors matching this sentinel to an INVALID_ARGUMENT response:
//
//	if errors.Is(err, filtering.ErrInvalidFilter) {
//		return nil, status.Error(codes.InvalidArgument, err.Error())
//	}
//
// Errors not matching ErrInvalidFilter indicate a bug in the calling
// service, such as an invalid schema or a misbehaving function expander.
var ErrInvalidFilter = errors.New("invalid filter")

// ParseError reports a syntax error in a filter. It matches
// [ErrInvalidFilter] with [errors.Is] and carries the byte offset of the
// offending token for precise user-facing messages.
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

// CheckError reports a filter that parsed but failed validation against a
// [Schema]: an unknown field, a type mismatch, or an unsupported operation.
// It matches [ErrInvalidFilter] with [errors.Is] and carries the byte offset
// of the offending token.
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
