package fieldbehavior

import (
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// ValidateRequired returns a [*RequiredFieldsError], which matches
// [ErrMissingRequired], with the path of every required field of the message
// that is not populated. It returns nil when all required fields are
// populated. A required field is populated when it has a value: non-zero for
// scalars without explicit presence, present for messages, non-empty for
// repeated fields and maps. Nested messages are validated wherever they are
// populated, including inside repeated fields and map values.
//
// Use it to validate create requests, where every required field must be
// provided; for update requests use [ValidateRequiredWithMask]. Passing a
// nil message panics, since there is no descriptor to validate against.
func ValidateRequired(msg proto.Message) error {
	return newRequiredFieldsError(missingRequired(msg.ProtoReflect(), ""))
}

// ValidateRequiredWithMask is [ValidateRequired] restricted to the fields
// the mask covers, per the update rule that a required field may be omitted
// as long as it is absent from the field mask.
//
// A mask path covers the field it names and, by prefix, everything below
// it: "author" also validates the required subfields of author, whether or
// not the message carries an author. A nil or empty mask means the implied
// mask of populated fields, and a "*" path covers everything. Unknown paths
// are skipped; validating mask paths themselves is the fieldmask package's
// concern. Map keys in paths must be plain string keys; backtick-escaped or
// integer keys are not resolved and are skipped.
//
// Passing a nil message panics, since there is no descriptor to validate
// against.
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
	children map[string]*maskNode
	terminal bool
}

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

// missingInField returns the path of one field when it is required and missing,
// or, when it is populated, the missing paths of every nested message below
// it.
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

// missingInCoveredField checks a field that a terminal mask path covers. Beyond
// [checkField] it descends into an unset singular message, because a path
// covers everything below it: a mask of "author" validates author.name even
// when the message carries no author, matching the "author.name" path that
// names the subfield directly.
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

// joinPath joins a parent path and a segment with a dot, omitting the dot
// for a root segment.
func joinPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}

	return prefix + "." + segment
}
