package fieldbehavior

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// rangeNested calls f for every message that the field holds in v, along
// with that message's path: the message itself for a singular message
// field, each element of a repeated message field, and each value of a map
// of messages. Fields that hold no messages are skipped, so f is never
// called for scalars, maps of scalars, or repeated scalars.
//
// It is the one place that knows how the three message-bearing field shapes
// are traversed, shared by the recursive walks of [Clear] and
// [ValidateRequired].
func rangeNested(
	fd protoreflect.FieldDescriptor,
	v protoreflect.Value,
	path string,
	f func(nested protoreflect.Message, nestedPath string),
) {
	switch {
	case fd.IsMap():
		if fd.MapValue().Kind() != protoreflect.MessageKind {
			return
		}

		v.Map().Range(func(k protoreflect.MapKey, mv protoreflect.Value) bool {
			f(mv.Message(), path+"."+k.String())

			return true
		})
	case fd.IsList():
		if fd.Kind() != protoreflect.MessageKind {
			return
		}

		list := v.List()
		for i := range list.Len() {
			f(list.Get(i).Message(), fmt.Sprintf("%s[%d]", path, i))
		}
	case fd.Kind() == protoreflect.MessageKind:
		f(v.Message(), path)
	}
}
