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

// CheckUpdate validates the mask's paths against the message type of msg. It
// reports, with an error matching [ErrInvalidFieldMask], paths that name
// unknown fields, traverse values that have no subfields, address repeated
// field elements, use malformed backtick quoting, or combine the "*"
// wildcard with other paths.
//
// A nil or empty mask is valid: an omitted mask has a meaning of its own in
// both updates and reads. [Update] runs the same validation, so CheckUpdate is
// for failing fast before fetching the resource. Use [CheckRead] for a read
// mask.
func CheckUpdate(mask *fieldmaskpb.FieldMask, msg proto.Message) error {
	return check(mask, msg, false)
}

// CheckRead validates the paths of a read mask against the message type of
// msg. It is [CheckUpdate], except that a path can go through a repeated message
// field to name a field of each element: "books.title" selects the title of
// every book of a list response. A path still cannot address an element by
// its index, or go through a repeated field of scalars.
//
// [Prune] runs the same validation, so CheckRead is for failing fast before
// the response is computed.
func CheckRead(mask *fieldmaskpb.FieldMask, msg proto.Message) error {
	return check(mask, msg, true)
}

// check validates the mask's paths against the message type of msg.
// throughRepeated tells whether a path can name the fields of the elements
// of a repeated message field.
func check(mask *fieldmaskpb.FieldMask, msg proto.Message, throughRepeated bool) error {
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
		if err := checkPath(md, path, throughRepeated); err != nil {
			return fmt.Errorf("%w: path %q: %w", ErrInvalidFieldMask, path, err)
		}
	}

	return nil
}

// IsWildcard reports whether the mask is the wildcard mask, which
// requests a full replacement of the resource in an update.
func IsWildcard(mask *fieldmaskpb.FieldMask) bool {
	return len(mask.GetPaths()) == 1 && mask.GetPaths()[0] == WildcardPath
}
