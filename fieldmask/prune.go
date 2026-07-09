package fieldmask

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// Prune clears every field of msg the mask does not cover, implementing the
// read mask of a partial response. A covered field is kept whole; a path
// into a message or map keeps only the named subfields or entries.
//
// A nil or empty mask, and the explicit "*" mask, mean the full resource:
// Prune keeps everything. The mask is [Check]-validated first, and on error
// msg is left unmodified. Field behavior annotations play no role in
// reading, so output-only fields are as readable as any other.
func Prune(mask *fieldmaskpb.FieldMask, msg proto.Message) error {
	if err := Check(mask, msg); err != nil {
		return err
	}

	if len(mask.GetPaths()) == 0 || IsFullReplacement(mask) {
		return nil
	}

	pruneTree(msg.ProtoReflect(), newMaskTree(mask.GetPaths()))

	return nil
}

// pruneTree clears the fields of one message level that the mask tree does
// not cover, recursing into covered subtrees.
func pruneTree(m protoreflect.Message, node *maskNode) {
	var uncovered []protoreflect.FieldDescriptor

	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		child, ok := node.children[string(fd.Name())]
		if !ok {
			uncovered = append(uncovered, fd)

			return true
		}

		if child.terminal {
			return true
		}

		switch {
		case fd.IsMap():
			pruneMapKeys(v.Map(), fd, child)
		case fd.Kind() == protoreflect.MessageKind:
			pruneTree(v.Message(), child)
		}

		return true
	})

	for _, fd := range uncovered {
		m.Clear(fd)
	}
}

// pruneMapKeys clears the entries of a map field that the mask tree does
// not name, recursing into named entries with covered subfields. Segments
// are canonicalised into map keys before they are matched, so that the
// integer-keyed path "editions.05" names the same entry as "editions.5".
func pruneMapKeys(mp protoreflect.Map, fd protoreflect.FieldDescriptor, node *maskNode) {
	covered := make(map[string]*maskNode, len(node.children))
	for segment, child := range node.children {
		covered[mapKey(fd.MapKey(), segment).String()] = child
	}

	var uncovered []protoreflect.MapKey

	mp.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		child, ok := covered[k.String()]
		if !ok {
			uncovered = append(uncovered, k)

			return true
		}

		if !child.terminal {
			// Below a key the value is a message; Check has passed.
			pruneTree(v.Message(), child)
		}

		return true
	})

	for _, k := range uncovered {
		mp.Clear(k)
	}
}
