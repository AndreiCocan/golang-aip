package fieldbehavior

import (
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Copy copies each field that has one or more of the given behaviors from
// src to dst. When such a field has no value in src, Copy clears it in dst.
// After Copy, the annotated fields of dst are equal to those of src.
//
// Copy goes into singular message fields, and makes them in dst when src
// has annotated values below them. Copy does not go into repeated fields or
// maps, because the elements of dst and src do not correspond. Copy does not
// deep-copy message, repeated, and map values: dst and src then share them.
//
// Use it to put back the OUTPUT_ONLY fields of the stored resource into a
// payload before a full replacement. dst and src must have the same message
// type and must not be nil. If not, Copy panics.
func Copy(dst, src proto.Message, behaviors ...annotations.FieldBehavior) {
	copyMessage(dst.ProtoReflect(), src.ProtoReflect(), behaviors)
}

// copyMessage sets each field of dst that has one of behaviors to its value
// in src, or clears it when src does not have it. It does the same in the
// nested messages of the other fields.
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
