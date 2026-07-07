package resourceid

import "github.com/google/uuid"

// NewSystemGenerated returns a new system-generated resource ID: a lowercase
// UUIDv7 string, e.g. "0190163d-8694-7d9b-8080-3f1a2b3c4d5e". Use it to fill
// the trailing segment of a resource name when the user left the
// "{resource}_id" field on a Create request unset.
//
// UUIDv7 values are time-ordered, so IDs generated later sort after earlier
// ones, a friendly property for storage locality and pagination.
//
// System-generated IDs are not user-settable IDs: most begin with a digit and
// would therefore fail [ValidateUserSettable]. Validate only the IDs users
// supply, not the ones this function produces.
func NewSystemGenerated() string {
	return uuid.Must(uuid.NewV7()).String()
}
