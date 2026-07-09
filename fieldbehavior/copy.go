package fieldbehavior

import (
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Copy copies every field annotated with any of the given behaviors from
// src to dst. An annotated field unpopulated in src is cleared in dst, so
// after Copy the annotated fields of dst mirror src exactly.
//
// Copy recurses through singular message fields (creating them in dst when
// src carries annotated values below), but not through repeated fields or
// maps, where no element correspondence between dst and src exists. Message,
// repeated, and map values are copied by reference, not deep-copied.
//
// dst and src must share a message type; Copy panics otherwise, as does
// passing a nil message. Copying OUTPUT_ONLY from the stored resource onto
// an incoming payload restores server-managed fields the client cannot set.
func Copy(dst, src proto.Message, behaviors ...annotations.FieldBehavior) {
	copyMessage(dst.ProtoReflect(), src.ProtoReflect(), behaviors)
}

func copyMessage(dst, src protoreflect.Message, behaviors []annotations.FieldBehavior) {
	fields := dst.Descriptor().Fields()
	for i := range fields.Len() {
		fd := fields.Get(i)
		if hasAny(fd, behaviors) {
			if src.Has(fd) {
				dst.Set(fd, src.Get(fd))
			} else {
				dst.Clear(fd)
			}

			continue
		}

		if fd.Kind() != protoreflect.MessageKind || fd.IsMap() || fd.IsList() {
			continue
		}

		if !src.Has(fd) && !dst.Has(fd) {
			continue
		}

		copyMessage(dst.Mutable(fd).Message(), src.Get(fd).Message(), behaviors)
	}
}
