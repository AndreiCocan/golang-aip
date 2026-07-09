package fieldbehavior

import (
	"slices"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Clear clears every field of the message, recursively, that is annotated
// with any of the given behaviors. Fields of nested messages are cleared
// wherever they appear, including inside repeated fields and map values,
// since a nested message's annotations are independent of its parent's.
//
// Clearing OUTPUT_ONLY (and, on create, IDENTIFIER) from a request payload
// implements the rule that clients cannot write server-managed fields:
// their values are dropped without error. A nil message has no fields to
// clear, and is a no-op rather than a panic.
func Clear(msg proto.Message, behaviors ...annotations.FieldBehavior) {
	if msg == nil {
		return
	}

	clearMessage(msg.ProtoReflect(), behaviors)
}

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
