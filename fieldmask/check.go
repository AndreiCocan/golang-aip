package fieldmask

import (
	"fmt"
	"slices"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// WildcardPath is the field mask path that selects all fields. In an update
// mask it requests a full replacement. In a read mask it requests the full
// resource. It must be the only path of the mask.
const WildcardPath = "*"

// CheckUpdate validates the paths of an update mask against the message
// type of msg. It returns an error that matches [ErrInvalidFieldMask] when a
// path:
//
//   - names an unknown field,
//   - goes into a value that has no subfields,
//   - goes into a repeated field,
//   - has incorrect backtick quotes, or
//   - is the "*" wildcard together with other paths.
//
// A nil or empty mask is valid, because an omitted mask has its own meaning.
// [Update] does the same validation. Call CheckUpdate before you read the
// stored resource, to fail early. For a read mask, use [CheckRead].
func CheckUpdate(mask *fieldmaskpb.FieldMask, msg proto.Message) error {
	return check(mask, msg, false)
}

// CheckRead validates the paths of a read mask against the message type of
// msg. It is [CheckUpdate], except that a path can go through a repeated
// message field to name a field of each element: "books.title" selects the
// title of each book of a list response. A path cannot name an element by its
// index, or go through a repeated field of scalars.
//
// [Prune] does the same validation. Call CheckRead before you compute the
// response, to fail early.
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

// IsWildcard reports whether the only path of the mask is [WildcardPath].
// Such a mask requests a full replacement in an update, and the full
// resource in a read.
func IsWildcard(mask *fieldmaskpb.FieldMask) bool {
	return len(mask.GetPaths()) == 1 && mask.GetPaths()[0] == WildcardPath
}
