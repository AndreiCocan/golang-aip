package fieldbehavior

import (
	"slices"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Clear clears each field of msg that has one or more of the given
// behaviors, at any depth. It also clears such fields in the messages inside
// repeated fields and map values, because the annotations of a nested
// message do not depend on its parent.
//
// Use it on a create or update payload to drop the fields that a client
// cannot write: clear OUTPUT_ONLY, and IDENTIFIER on create. Clear returns
// no error for the dropped values. A nil msg has no fields, so Clear does
// nothing.
func Clear(msg proto.Message, behaviors ...annotations.FieldBehavior) {
	if msg == nil {
		return
	}

	clearMessage(msg.ProtoReflect(), behaviors)
}

// clearMessage clears each field of m that has one of behaviors, and does
// the same in the nested messages of the other fields.
func clearMessage(m protoreflect.Message, behaviors []annotations.FieldBehavior) {
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if hasAny(fd, behaviors) {
			m.Clear(fd)

			return true
		}

		rangeNested(fd, v, "", func(nested protoreflect.Message, _ string) {
			clearMessage(nested, behaviors)
		})

		return true
	})
}

// hasAny reports whether the field is annotated with at least one of the
// given behaviors.
func hasAny(fd protoreflect.FieldDescriptor, behaviors []annotations.FieldBehavior) bool {
	return slices.ContainsFunc(Get(fd), func(b annotations.FieldBehavior) bool {
		return slices.Contains(behaviors, b)
	})
}
