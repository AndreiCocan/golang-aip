package fieldbehavior

import (
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// ValidateRequired returns the path of each field of msg that has the
// REQUIRED annotation and no value, in a [*RequiredFieldsError] that matches
// [ErrMissingRequired]. It returns nil when all required fields have a
// value. A field has a value when:
//
//   - it is a scalar without explicit presence and it is not zero,
//   - it has explicit presence and it is present, or
//   - it is a repeated field or a map and it is not empty.
//
// ValidateRequired also validates each populated nested message, also inside
// repeated fields and map values.
//
// Use it for a create request, where all required fields must have a value.
// For an update request, use [ValidateRequiredWithMask]. A nil msg has no
// descriptor, so ValidateRequired panics.
func ValidateRequired(msg proto.Message) error {
	return newRequiredFieldsError(missingRequired(msg.ProtoReflect(), ""))
}

// ValidateRequiredWithMask is [ValidateRequired] for the fields that the
// mask covers only. In an update, a required field can have no value when
// the mask does not cover it.
//
// A mask path covers the field that it names and all the fields below it:
// "author" also validates the required subfields of author, also when msg
// has no author. A nil or empty mask means the implied mask of the populated
// fields. The "*" path covers all fields.
//
// ValidateRequiredWithMask ignores a path that names an unknown field. Use
// fieldmask.CheckUpdate to validate the paths. It also ignores a map key that
// is in backticks or that is an integer: only plain string keys resolve.
// A nil msg has no descriptor, so ValidateRequiredWithMask panics.
func ValidateRequiredWithMask(msg proto.Message, mask *fieldmaskpb.FieldMask) error {
	m := msg.ProtoReflect()

	if len(mask.GetPaths()) == 0 {
		var missing []string

		m.Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
			missing = append(missing, missingInField(m, fd, string(fd.Name()))...)

			return true
		})

		return newRequiredFieldsError(missing)
	}

	root := &maskNode{children: map[string]*maskNode{}}
	for _, path := range mask.GetPaths() {
		root.insert(strings.Split(path, "."))
	}

	return newRequiredFieldsError(missingInCovered(m, root, ""))
}

// newRequiredFieldsError returns a [*RequiredFieldsError] with the missing
// paths, or nil when there are none.
func newRequiredFieldsError(missing []string) error {
	if len(missing) == 0 {
		return nil
	}

	return &RequiredFieldsError{Paths: missing}
}

// maskNode is one segment of a field mask path tree. A terminal node covers
// the whole subtree below its path.
type maskNode struct {
	// children holds the node of each next segment, by its name.
	children map[string]*maskNode
	// terminal reports whether a path of the mask ends at this node.
	terminal bool
}

// insert adds the path of segments below n.
func (n *maskNode) insert(segments []string) {
	if len(segments) == 0 {
		n.terminal = true

		return
	}

	child, ok := n.children[segments[0]]
	if !ok {
		child = &maskNode{children: map[string]*maskNode{}}
		n.children[segments[0]] = child
	}

	child.insert(segments[1:])
}

// missingRequired returns the paths of the missing required fields of the
// message, at any depth.
func missingRequired(m protoreflect.Message, prefix string) []string {
	fields := m.Descriptor().Fields()

	missing := make([]string, 0, fields.Len())
	for i := range fields.Len() {
		fd := fields.Get(i)
		missing = append(missing, missingInField(m, fd, joinPath(prefix, string(fd.Name())))...)
	}

	return missing
}

// missingInField returns the path of fd when fd is required and has no
// value. When fd has a value, it returns the paths of the missing required
// fields in the messages that fd holds.
func missingInField(m protoreflect.Message, fd protoreflect.FieldDescriptor, path string) []string {
	if !m.Has(fd) {
		if Has(fd, annotations.FieldBehavior_REQUIRED) {
			return []string{path}
		}

		return nil
	}

	var missing []string

	rangeNested(fd, m.Get(fd), path, func(nested protoreflect.Message, nestedPath string) {
		missing = append(missing, missingRequired(nested, nestedPath)...)
	})

	return missing
}

// missingInCoveredField is [missingInField] for a field that a mask path
// covers completely. It also goes into an unset singular message, because
// the path covers all fields below it: the mask "author" validates
// author.name also when the message has no author, as the mask
// "author.name" does.
func missingInCoveredField(
	m protoreflect.Message,
	fd protoreflect.FieldDescriptor,
	path string,
) []string {
	if missing := missingInField(m, fd, path); len(missing) > 0 {
		return missing
	}

	// A populated field was already walked by checkField, and only singular
	// messages have subfields a mask can reach when unset.
	if m.Has(fd) || fd.IsMap() || fd.IsList() || fd.Kind() != protoreflect.MessageKind {
		return nil
	}

	return missingRequired(m.Get(fd).Message(), path)
}

// missingInCovered returns the missing required paths among the fields of m
// that the mask tree covers.
func missingInCovered(m protoreflect.Message, node *maskNode, prefix string) []string {
	var missing []string

	for segment, child := range node.children {
		if segment == "*" {
			missing = append(missing, missingRequired(m, prefix)...)

			continue
		}

		fd := m.Descriptor().Fields().ByName(protoreflect.Name(segment))
		if fd == nil {
			continue
		}

		path := joinPath(prefix, segment)
		if child.terminal {
			missing = append(missing, missingInCoveredField(m, fd, path)...)

			continue
		}

		switch {
		case fd.IsMap():
			missing = append(missing, missingInCoveredMapKeys(m, fd, child, path)...)
		case fd.IsList():
			// No coverage below repeated fields: masks cannot traverse them.
		case fd.Kind() == protoreflect.MessageKind:
			// Get on an unpopulated field yields an empty message, so
			// covered required subfields of an unset parent still report
			// as missing.
			missing = append(missing, missingInCovered(m.Get(fd).Message(), child, path)...)
		}
	}

	return missing
}

// missingInCoveredMapKeys returns the missing required paths among the
// entries of a string-keyed map of messages that the mask tree names.
func missingInCoveredMapKeys(
	m protoreflect.Message,
	fd protoreflect.FieldDescriptor,
	node *maskNode,
	path string,
) []string {
	if fd.MapKey().Kind() != protoreflect.StringKind ||
		fd.MapValue().Kind() != protoreflect.MessageKind {
		return nil
	}

	mp := m.Get(fd).Map()

	var missing []string

	for key, child := range node.children {
		mk := protoreflect.ValueOfString(key).MapKey()
		if !mp.Has(mk) {
			continue
		}

		keyPath := path + "." + key
		if child.terminal {
			missing = append(missing, missingRequired(mp.Get(mk).Message(), keyPath)...)

			continue
		}

		missing = append(missing, missingInCovered(mp.Get(mk).Message(), child, keyPath)...)
	}

	return missing
}

// joinPath joins a parent path and a segment with a dot. When prefix is
// empty, it returns segment.
func joinPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}

	return prefix + "." + segment
}
