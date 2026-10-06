package fieldmask

import "errors"

// ErrInvalidFieldMask matches, with [errors.Is], every error that reports a
// field mask that does not apply to its message type. Such an error is bad
// client input. A mask does not apply when a path:
//
//   - names an unknown field,
//   - goes into a value that has no subfields,
//   - names the elements of a repeated field,
//   - has a segment with incorrect quotes, or
//   - is the "*" wildcard together with other paths.
var ErrInvalidFieldMask = errors.New("invalid field mask")

// ErrImmutable matches, with [errors.Is], every error that reports an
// [Update] that would change a field with the IMMUTABLE or IDENTIFIER
// annotation. Such an error is bad client input.
var ErrImmutable = errors.New("cannot change immutable field")
