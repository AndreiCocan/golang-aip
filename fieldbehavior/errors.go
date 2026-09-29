package fieldbehavior

import (
	"errors"
	"strings"
)

// ErrMissingRequired is the sentinel matched by every error returned for a
// field annotated REQUIRED that carries no value. Services should map
// errors matching this sentinel to an INVALID_ARGUMENT response:
//
//	if errors.Is(err, fieldbehavior.ErrMissingRequired) {
//		return nil, status.Error(codes.InvalidArgument, err.Error())
//	}
var ErrMissingRequired = errors.New("missing required field")

// RequiredFieldsError reports the fields annotated REQUIRED that carry no
// value. [ValidateRequired] and [ValidateRequiredWithMask] return it. It
// matches [ErrMissingRequired] with [errors.Is]. Use [errors.AsType] to get
// the paths, for example to build one field violation per missing field:
//
//	if rf, ok := errors.AsType[*fieldbehavior.RequiredFieldsError](err); ok {
//		for _, path := range rf.Paths { … }
//	}
type RequiredFieldsError struct {
	// Paths are the dot-separated paths of the missing fields from the
	// validated message, such as "book.title". There is at least one path.
	// The order is not stable across calls when the message has maps of
	// messages or when the error comes from [ValidateRequiredWithMask]; sort
	// the paths when the order matters.
	Paths []string
}

// Error returns one line per missing field, each "missing required field: "
// followed by the path.
func (e *RequiredFieldsError) Error() string {
	lines := make([]string, len(e.Paths))
	for i, path := range e.Paths {
		lines[i] = ErrMissingRequired.Error() + ": " + path
	}

	return strings.Join(lines, "\n")
}

// Is reports whether target is [ErrMissingRequired].
func (e *RequiredFieldsError) Is(target error) bool {
	return target == ErrMissingRequired
}
