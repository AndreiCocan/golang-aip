package resourceid

import (
	"errors"
	"fmt"
)

// ErrInvalid matches every error returned by [ValidateUserSettable] via
// [errors.Is]. Use it to distinguish a malformed resource ID from other
// failures; the wrapped message states the specific rule that was violated.
var ErrInvalid = errors.New("invalid resource ID")

// ValidateUserSettable reports whether id is a well-formed user-settable
// resource ID. Call it when an end user supplies the trailing segment of a
// resource name (typically the "{resource}_id" field on a Create request)
// and reject the request if a non-nil error is returned.
//
// The id must match ^[a-z][a-z0-9-]{2,61}[a-z0-9]$, that is:
//   - 4 to 63 characters, per AIP-133's recommendation,
//   - first character a lowercase ASCII letter,
//   - last character a lowercase letter or digit (no trailing hyphen),
//   - remaining characters lowercase letters, digits, or hyphens.
//
// UUIDs get no special treatment: one that starts with a hex digit fails,
// while a lowercase UUID that starts with a hex letter passes.
//
// The returned error carries a human-readable reason suitable to propagate
// to the caller, e.g. in an INVALID_ARGUMENT status message.
func ValidateUserSettable(id string) error {
	if len(id) < 4 || len(id) > 63 {
		return fmt.Errorf("%w: must be between 4 and 63 characters", ErrInvalid)
	}

	if id[0] < 'a' || id[0] > 'z' {
		return fmt.Errorf("%w: must begin with a lowercase letter", ErrInvalid)
	}

	if id[len(id)-1] == '-' {
		return fmt.Errorf("%w: must end with a lowercase letter or a digit", ErrInvalid)
	}

	for position, character := range id {
		switch {
		case 'a' <= character && character <= 'z':
		case '0' <= character && character <= '9':
		case character == '-':
		default:
			return fmt.Errorf(
				"%w: must contain only lowercase letters, digits, and hyphens, got %q at position %d",
				ErrInvalid,
				character,
				position,
			)
		}
	}

	return nil
}
