package resourceid

import (
	"errors"
	"fmt"
)

// ErrInvalidID matches, with [errors.Is], every error of
// [ValidateUserSettable]. Use it to tell a malformed resource ID from
// other failures. The message of the error gives the rule that the ID
// does not obey.
var ErrInvalidID = errors.New("invalid resource ID")

// ValidateUserSettable returns an error when id is not a valid resource ID
// that a user can set. Call it when a user gives the last segment of a
// resource name, usually in the "{resource}_id" field of a Create request.
// Reject the request when it returns an error.
//
// The id must match ^[a-z][a-z0-9-]{2,61}[a-z0-9]$. Thus:
//
//   - It has 4 to 63 characters, as AIP-133 recommends.
//   - The first character is a lowercase ASCII letter.
//   - The last character is a lowercase letter or a digit, not a hyphen.
//   - The other characters are lowercase letters, digits, or hyphens.
//
// A UUID gets no special treatment. A UUID that starts with a hex digit
// fails. A lowercase UUID that starts with a hex letter passes.
//
// The message of the error is for humans. It is correct to send it to the
// client.
func ValidateUserSettable(id string) error {
	if len(id) < 4 || len(id) > 63 {
		return fmt.Errorf("%w: must be between 4 and 63 characters", ErrInvalidID)
	}

	if id[0] < 'a' || id[0] > 'z' {
		return fmt.Errorf("%w: must begin with a lowercase letter", ErrInvalidID)
	}

	if id[len(id)-1] == '-' {
		return fmt.Errorf("%w: must end with a lowercase letter or a digit", ErrInvalidID)
	}

	for position, character := range id {
		switch {
		case 'a' <= character && character <= 'z':
		case '0' <= character && character <= '9':
		case character == '-':
		default:
			return fmt.Errorf(
				"%w: must contain only lowercase letters, digits, and hyphens, got %q at position %d",
				ErrInvalidID,
				character,
				position,
			)
		}
	}

	return nil
}
