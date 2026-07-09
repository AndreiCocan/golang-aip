package fieldmask

import "errors"

// ErrInvalidFieldMask is the sentinel matched by every error reporting a
// field mask that cannot be applied to its message type: a path naming an
// unknown field, traversing a value that has no subfields, addressing a
// repeated field's elements, quoting a segment malformedly, or combining
// the "*" wildcard with other paths. Services should map errors matching
// this sentinel to an INVALID_ARGUMENT response:
//
//	if errors.Is(err, fieldmask.ErrInvalidFieldMask) {
//		return nil, status.Error(codes.InvalidArgument, err.Error())
//	}
var ErrInvalidFieldMask = errors.New("invalid field mask")

// ErrImmutable is the sentinel matched by every error reporting an [Update]
// that would change a field annotated IMMUTABLE or IDENTIFIER. Services
// should map errors matching this sentinel to an INVALID_ARGUMENT response.
var ErrImmutable = errors.New("cannot change immutable field")
