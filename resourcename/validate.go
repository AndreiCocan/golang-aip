package resourcename

import (
	"errors"
	"fmt"
)

// ErrInvalidName matches every error returned by [Validate] via [errors.Is].
// The wrapped message states the specific rule that was violated.
var ErrInvalidName = errors.New("invalid resource name")

// Validate reports whether name is a well-formed concrete resource name.
// Both relative names ("publishers/123") and full names
// ("//library.example.com/publishers/123") are accepted; pattern syntax is
// rejected; for patterns, use [ValidatePattern].
//
// Rules:
//   - The name must be non-empty and every segment must be non-empty.
//   - Each segment must be a valid RFC 1123 host name: dot-separated labels
//     of letters, digits, and hyphens, with no leading or trailing hyphen
//     and at most 63 characters per label.
//   - The [Wildcard] "-" is allowed as a whole segment, standing for an
//     unspecified ID when reading across collections.
//   - Variable segments ("{name}") are rejected: a concrete name has no
//     placeholders.
//   - For a full name, the service host must be a valid DNS name.
//
// The returned error matches [ErrInvalidName] and carries a human-readable
// reason suitable to propagate to the caller.
func Validate(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty", ErrInvalidName)
	}

	var sc Scanner
	sc.Init(name)

	var i int
	for sc.Scan() {
		i++
		segment := sc.Segment()

		switch {
		case segment == "":
			return fmt.Errorf("%w: segment %d is empty", ErrInvalidName, i)
		case segment.IsWildcard():
			continue
		case segment.IsVariable():
			return fmt.Errorf(
				"%w: segment %q: concrete names must not contain variables",
				ErrInvalidName,
				segment,
			)
		case !isRFC1123Name(string(segment)):
			return fmt.Errorf("%w: segment %q: not a valid identifier", ErrInvalidName, segment)
		}
	}

	if sc.Full() && !isRFC1123Name(sc.ServiceName()) {
		return fmt.Errorf("%w: service %q: not a valid DNS name", ErrInvalidName, sc.ServiceName())
	}

	return nil
}
