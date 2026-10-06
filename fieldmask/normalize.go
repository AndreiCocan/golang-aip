package fieldmask

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// Normalize returns a copy of the mask in which every path names its
// fields by their proto names. Use it on a mask that a client wrote with
// the JSON names of the fields, such as "createTime" or "author.name", so
// that [CheckUpdate], [Prune], and [Update] accept it: "createTime" becomes
// "create_time".
//
// Only field segments change. A map key, such as "myKey" in
// "labels.myKey", and a backtick-quoted segment stay as they are. A
// segment that is the proto name of a field is not changed, also when it
// is the JSON name of a different field. The "*" path stays as it is.
//
// Normalize returns an empty mask for a nil mask. The two have the same
// meaning in [CheckUpdate], [Prune], and [Update]. It returns an error that
// matches [ErrInvalidFieldMask] when a path does not resolve against the
// message type of msg, for the same causes as [CheckRead]. Thus it accepts
// a path through a repeated message field, such as "books.displayName",
// which only a read mask allows; [CheckUpdate] rejects it. It does not detect a
// "*" path together with other paths; [CheckUpdate] and [CheckRead] do.
func Normalize(mask *fieldmaskpb.FieldMask, msg proto.Message) (*fieldmaskpb.FieldMask, error) {
	md := msg.ProtoReflect().Descriptor()

	paths := make([]string, len(mask.GetPaths()))
	for i, path := range mask.GetPaths() {
		if path == WildcardPath {
			paths[i] = path

			continue
		}

		normalized, err := normalizePath(md, path)
		if err != nil {
			return nil, fmt.Errorf("%w: path %q: %w", ErrInvalidFieldMask, path, err)
		}

		paths[i] = normalized
	}

	return &fieldmaskpb.FieldMask{Paths: paths}, nil
}

// normalizePath resolves one path against a message type, as [checkPath]
// does, and returns it with the proto name of each field segment.
func normalizePath(md protoreflect.MessageDescriptor, path string) (string, error) {
	segments, err := splitPath(path)
	if err != nil {
		return "", err
	}

	normalized := make([]string, len(segments))

	pos := pathPosition{message: md}
	for i, segment := range segments {
		if pos.message != nil && !segment.quoted {
			segment.value = protoName(pos.message, segment.value)
		}

		pos, err = pos.step(segment, true)
		if err != nil {
			return "", err
		}

		normalized[i] = segment.String()
	}

	return strings.Join(normalized, "."), nil
}

// protoName returns the proto name of the field of md that name names, by
// its proto name first and then by its JSON name. It returns name
// unchanged when no field has that name.
func protoName(md protoreflect.MessageDescriptor, name string) string {
	fields := md.Fields()
	if fields.ByName(protoreflect.Name(name)) != nil {
		return name
	}

	if fd := fields.ByJSONName(name); fd != nil {
		return string(fd.Name())
	}

	return name
}
