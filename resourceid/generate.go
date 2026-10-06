package resourceid

import "github.com/google/uuid"

// Generate returns a new resource ID that the system makes: a lowercase
// UUIDv7 string, such as "0190163d-8694-7d9b-8080-3f1a2b3c4d5e". Use it for
// the last segment of a resource name when the "{resource}_id" field of a
// Create request has no value.
//
// UUIDv7 values sort by time: an ID made later sorts after an ID made
// before. This helps storage locality and pagination.
//
// Most generated IDs start with a digit, so [ValidateUserSettable] rejects
// them. Validate only the IDs that users give.
//
// Generate is safe for concurrent use. It panics when the system cannot
// supply random bytes.
func Generate() string {
	return uuid.Must(uuid.NewV7()).String()
}
