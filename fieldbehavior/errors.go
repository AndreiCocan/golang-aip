package fieldbehavior

import (
	"errors"
	"strings"
)

// ErrMissingRequired matches, with [errors.Is], every error that reports a
// field with the REQUIRED annotation and no value. Such an error is bad
// client input.
var ErrMissingRequired = errors.New("missing required field")

// RequiredFieldsError reports the fields with the REQUIRED annotation that
// have no value. [ValidateRequired] and [ValidateRequiredWithMask] return
// it. It matches [ErrMissingRequired] with [errors.Is]. Use
// [errors.AsType] to get the paths, for example to make one field
// violation for each missing field.
type RequiredFieldsError struct {
	// Paths are the paths of the missing fields, from the root of the
	// validated message, such as "title" or "author.name". A path into a
	// repeated field has the index of the element in brackets, such as
	// "chapters[2].title". A path into a map has the key as a segment,
	// such as "reviews.smith.rating". There is at least one path.
	//
	// The order can change between calls when the message has maps of
	// messages, or when the error comes from [ValidateRequiredWithMask].
	// Sort the paths when the order is important.
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
