package fieldmask

import (
	"fmt"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/AndreiCocan/golang-aip/fieldbehavior"
)

// Update merges the masked fields of src into dst. src is the payload of an
// update request, and dst is the stored resource. A masked field gets the
// value that it has in src. When the field has no value in src, Update
// clears it in dst.
//
// The paths of the mask name fields of the resource. Update validates the
// mask with [CheckUpdate] first. A nil or empty mask means the implied mask
// of the populated fields of src, so Update never clears a field. The "*"
// mask requests a full replacement: each field gets the value that it has in
// src, populated or not.
//
// The field behavior annotations limit what a mask can do:
//
//   - An OUTPUT_ONLY field keeps its stored value, at any depth, and Update
//     returns no error for it.
//   - An IMMUTABLE or IDENTIFIER field accepts its current value. A different
//     value gives an error that matches [ErrImmutable]. With the "*" mask, an
//     unpopulated IMMUTABLE or IDENTIFIER field keeps its stored value.
//   - Update does not enforce the annotations inside repeated fields and map
//     values, because a replaced element has no stored element to compare
//     with.
//
// dst and src must have the same message type. If not, Update panics. When
// Update returns an error, dst does not change. After a successful Update,
// dst shares no data with src. Update is not safe for concurrent use on the
// same dst.
func Update(mask *fieldmaskpb.FieldMask, dst, src proto.Message) error {
	if dst.ProtoReflect().Descriptor() != src.ProtoReflect().Descriptor() {
		panic("fieldmask: Update dst and src must share a message type")
	}

	if err := CheckUpdate(mask, dst); err != nil {
		return err
	}

	merged := proto.Clone(dst)
	to, from := merged.ProtoReflect(), src.ProtoReflect()

	switch {
	case IsWildcard(mask):
		replaceAll(to, from)
	case len(mask.GetPaths()) == 0:
		from.Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
			setField(to, from, fd)

			return true
		})
	default:
		tree := newMaskTree(mask.GetPaths())
		mergeTree(to, from, tree)
	}

	// The merge shares values with src. Clone before enforceBehaviors
	// changes them.
	merged = proto.Clone(merged)
	if err := enforceBehaviors(merged.ProtoReflect(), dst.ProtoReflect(), ""); err != nil {
		return err
	}

	proto.Reset(dst)
	proto.Merge(dst, merged)

	return nil
}

// replaceAll gives each field of dst the value that it has in src. It does
// not touch OUTPUT_ONLY fields. It keeps an IMMUTABLE or IDENTIFIER field
// that has no value in src. It writes an IMMUTABLE or IDENTIFIER field that
// has a value in src, and [enforceBehaviors] then rejects a changed value.
func replaceAll(dst, src protoreflect.Message) {
	fields := dst.Descriptor().Fields()
	for i := range fields.Len() {
		fd := fields.Get(i)
		if isOutputOnly(fd) {
			continue
		}

		if rejectsChange(fd) && !src.Has(fd) {
			continue
		}

		setField(dst, src, fd)
	}
}

// setField gives fd in dst the value that it has in src. When fd has no
// value in src, setField clears it in dst.
func setField(dst, src protoreflect.Message, fd protoreflect.FieldDescriptor) {
	if src.Has(fd) {
		dst.Set(fd, src.Get(fd))
	} else {
		dst.Clear(fd)
	}
}

// mergeTree merges the fields of src that node covers into dst, for one
// message level, and continues into the levels below.
func mergeTree(dst, src protoreflect.Message, node *maskNode) {
	for segment, child := range node.children {
		// The mask is valid, so the field exists.
		fd := dst.Descriptor().Fields().ByName(protoreflect.Name(segment))
		if child.terminal {
			setField(dst, src, fd)

			continue
		}

		switch {
		case fd.IsMap():
			mergeMapKeys(dst, src, fd, child)
		case fd.Kind() == protoreflect.MessageKind:
			if !dst.Has(fd) && !src.Has(fd) {
				continue
			}

			mergeTree(dst.Mutable(fd).Message(), src.Get(fd).Message(), child)
		}
	}
}

// mergeMapKeys merges the entries of the map field fd that node names by
// key, from src into dst.
func mergeMapKeys(dst, src protoreflect.Message, fd protoreflect.FieldDescriptor, node *maskNode) {
	dstMap := dst.Mutable(fd).Map()
	srcMap := src.Get(fd).Map()

	for key, child := range node.children {
		mk := mapKey(fd.MapKey(), key)
		if child.terminal {
			if srcMap.Has(mk) {
				dstMap.Set(mk, srcMap.Get(mk))
			} else {
				dstMap.Clear(mk)
			}

			continue
		}

		// The mask is valid, so the value below a key is a message.
		if !dstMap.Has(mk) && !srcMap.Has(mk) {
			continue
		}

		srcValue := dstMap.NewValue().Message()
		if srcMap.Has(mk) {
			srcValue = srcMap.Get(mk).Message()
		}

		mergeTree(dstMap.Mutable(mk).Message(), srcValue, child)
	}
}

// enforceBehaviors applies the field behavior annotations after a merge.
// merged is the result of the merge, and stored is the resource before the
// update. An OUTPUT_ONLY field gets its stored value back. An IMMUTABLE or
// IDENTIFIER field with a changed value gives an error that matches
// [ErrImmutable]. prefix is the path of merged from the root, for the error.
func enforceBehaviors(merged, stored protoreflect.Message, prefix string) error {
	fields := merged.Descriptor().Fields()
	for i := range fields.Len() {
		fd := fields.Get(i)
		path := joinPath(prefix, string(fd.Name()))

		switch {
		case isOutputOnly(fd):
			setField(merged, stored, fd)
		case rejectsChange(fd):
			if !merged.Get(fd).Equal(stored.Get(fd)) {
				return fmt.Errorf("%w: %s", ErrImmutable, path)
			}
		case fd.IsMap() || fd.IsList():
			// The annotations inside map values and repeated elements are
			// not enforced: a replaced element has no stored element.
		case fd.Kind() == protoreflect.MessageKind:
			if !mayNeedEnforcement(merged, stored, fd) {
				continue
			}

			err := enforceBehaviors(merged.Mutable(fd).Message(), stored.Get(fd).Message(), path)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// mayNeedEnforcement reports whether the message field fd can hold a value
// that [enforceBehaviors] must enforce. It prevents enforceBehaviors from
// making empty messages in merged for subtrees that have no annotated
// values.
func mayNeedEnforcement(merged, stored protoreflect.Message, fd protoreflect.FieldDescriptor) bool {
	if merged.Has(fd) {
		return true
	}

	return stored.Has(fd) && hasAnnotatedValue(stored.Get(fd).Message())
}

// hasAnnotatedValue reports whether the message has, at any depth, a
// populated field with an annotation that [enforceBehaviors] enforces:
// OUTPUT_ONLY, IMMUTABLE, or IDENTIFIER.
func hasAnnotatedValue(m protoreflect.Message) bool {
	found := false

	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if isOutputOnly(fd) || rejectsChange(fd) {
			found = true

			return false
		}

		if fd.Kind() == protoreflect.MessageKind && !fd.IsMap() && !fd.IsList() {
			found = hasAnnotatedValue(v.Message())

			return !found
		}

		return true
	})

	return found
}

// isOutputOnly reports whether the field has the OUTPUT_ONLY annotation:
// the server sets it, and a client cannot.
func isOutputOnly(fd protoreflect.FieldDescriptor) bool {
	return fieldbehavior.Has(fd, annotations.FieldBehavior_OUTPUT_ONLY)
}

// rejectsChange reports whether a client cannot change the field after
// creation: the field has the IMMUTABLE or the IDENTIFIER annotation.
func rejectsChange(fd protoreflect.FieldDescriptor) bool {
	return fieldbehavior.Has(fd, annotations.FieldBehavior_IMMUTABLE) ||
		fieldbehavior.Has(fd, annotations.FieldBehavior_IDENTIFIER)
}

// joinPath joins a parent path and a segment with a dot. When prefix is
// empty, it returns segment.
func joinPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}

	return prefix + "." + segment
}
