package fieldmask

import (
	"fmt"
	"slices"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// WildcardPath is the special field mask path meaning every field: full
// replacement in an update mask, all fields in a read mask.
const WildcardPath = "*"

// Check validates the mask's paths against the message type of msg. It
// reports, with an error matching [ErrInvalidFieldMask], paths that name
// unknown fields, traverse values that have no subfields, address repeated
// field elements, use malformed backtick quoting, or combine the "*"
// wildcard with other paths.
//
// A nil or empty mask is valid: an omitted mask has a meaning of its own in
// both updates and reads. [Update] and [Prune] run the same validation, so
// Check is for failing fast before fetching the resource.
func Check(mask *fieldmaskpb.FieldMask, msg proto.Message) error {
	paths := mask.GetPaths()
	if slices.Contains(paths, WildcardPath) {
		if len(paths) > 1 {
			return fmt.Errorf(
				"%w: the wildcard cannot be combined with other paths",
				ErrInvalidFieldMask,
			)
		}

		return nil
	}

	md := msg.ProtoReflect().Descriptor()
	for _, path := range paths {
		if err := checkPath(md, path); err != nil {
			return fmt.Errorf("%w: path %q: %w", ErrInvalidFieldMask, path, err)
		}
	}

	return nil
}

// IsFullReplacement reports whether the mask is the wildcard mask, which
// requests a full replacement of the resource in an update.
func IsFullReplacement(mask *fieldmaskpb.FieldMask) bool {
	return len(mask.GetPaths()) == 1 && mask.GetPaths()[0] == WildcardPath
}
