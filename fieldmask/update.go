package fieldmask

import (
	"fmt"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/AndreiCocan/golang-aip/fieldbehavior"
)

// Update merges the masked fields of src, an update request's payload, into
// dst, the stored resource. A masked field takes src's value; unpopulated
// in src, it is cleared. Both messages must share one type; Update panics
// otherwise. On error dst is left unmodified, and on success it shares no
// data with src.
//
// The mask's paths address fields of the resource, [CheckUpdate]-validated first.
// A nil or empty mask means the implied mask of src's populated fields,
// which updates without ever clearing. The "*" mask requests full
// replacement: every field takes src's value, populated or not.
//
// Field behavior annotations bound what any mask can do. OUTPUT_ONLY
// fields keep their stored value, at any depth, without erroring, no
// matter how the mask or payload names them. IMMUTABLE and IDENTIFIER
// fields accept their current value as a no-op but reject any change with
// an error matching [ErrImmutable]; under a "*" mask, leaving them
// unpopulated preserves them instead of clearing. Annotations inside
// repeated fields and map entries are not enforced, since replaced
// elements have no stored counterpart to compare against.
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

	// The merge shares values with src; clone before the sweep mutates any.
	merged = proto.Clone(merged)
	if err := enforceBehaviors(merged.ProtoReflect(), dst.ProtoReflect(), ""); err != nil {
		return err
	}

	proto.Reset(dst)
	proto.Merge(dst, merged)

	return nil
}

// replaceAll gives every field of w the value it has in s, except that
// output-only fields are never touched and immutable fields unpopulated in
// s are preserved rather than cleared. Immutable fields populated in s are
// written; the behavior sweep rejects them if that changed anything.
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

// setField gives fd in w the value it has in s: set when populated,
// cleared when not.
func setField(dst, src protoreflect.Message, fd protoreflect.FieldDescriptor) {
	if src.Has(fd) {
		dst.Set(fd, src.Get(fd))
	} else {
		dst.Clear(fd)
	}
}

// mergeTree applies the masked paths of one message level.
func mergeTree(dst, src protoreflect.Message, node *maskNode) {
	for segment, child := range node.children {
		// Check has passed: the field exists.
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

// mergeMapKeys applies masked paths that address entries of a map field by
// key.
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

		// Below a key the value is a message; Check has passed.
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

// enforceBehaviors enforces field behavior annotations after a merge: work
// is the merged resource, orig the stored one. Output-only fields get
// their stored values back, and a changed immutable field is an error.
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
			// Annotations inside map entries and repeated elements are
			// not enforced: replaced elements have no stored counterpart.
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

// mayNeedEnforcement reports whether a message field can hold anything for
// enforceBehaviors to enforce, avoiding the creation of empty messages in
// work for subtrees without annotated values.
func mayNeedEnforcement(merged, stored protoreflect.Message, fd protoreflect.FieldDescriptor) bool {
	if merged.Has(fd) {
		return true
	}

	return stored.Has(fd) && hasAnnotatedValue(stored.Get(fd).Message())
}

// hasAnnotatedValue reports whether the message holds, at any depth, a
// populated field carrying one of the annotations the update sweep
// enforces: OUTPUT_ONLY, IMMUTABLE, or IDENTIFIER. Populated fields without
// them give the sweep nothing to do.
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

// isOutputOnly reports whether the field is server-managed output.
func isOutputOnly(fd protoreflect.FieldDescriptor) bool {
	return fieldbehavior.Has(fd, annotations.FieldBehavior_OUTPUT_ONLY)
}

// rejectsChange reports whether the field rejects changes after creation,
// which covers both the IMMUTABLE and IDENTIFIER annotations.
func rejectsChange(fd protoreflect.FieldDescriptor) bool {
	return fieldbehavior.Has(fd, annotations.FieldBehavior_IMMUTABLE) ||
		fieldbehavior.Has(fd, annotations.FieldBehavior_IDENTIFIER)
}

// joinPath joins a parent path and a segment with a dot, omitting the dot
// for a root segment.
func joinPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}

	return prefix + "." + segment
}
