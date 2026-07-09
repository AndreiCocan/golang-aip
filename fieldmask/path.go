package fieldmask

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// pathSegment is one dot-separated segment of a field mask path. Quoting a
// segment in backticks makes it a literal, so that a map key can carry
// characters the path syntax would otherwise claim: the dots of
// "labels.`k8s.io/name`", or the "*" of "labels.`*`".
type pathSegment struct {
	value  string
	quoted bool
}

// splitPath splits a field mask path into its dot-separated segments. A
// segment may be backtick-quoted to carry problematic characters, dots
// included: "labels.`k8s.io/name`" has the two segments "labels" and
// "k8s.io/name", the second one quoted.
func splitPath(path string) ([]pathSegment, error) {
	if path == "" {
		return nil, errors.New("empty path")
	}

	var segments []pathSegment

	i := 0
	for {
		if i >= len(path) {
			return nil, errors.New("empty segment")
		}

		if path[i] == '`' {
			closing := strings.IndexByte(path[i+1:], '`')
			if closing < 0 {
				return nil, errors.New("unterminated backtick quote")
			}

			segment := path[i+1 : i+1+closing]
			if segment == "" {
				return nil, errors.New("empty segment")
			}

			segments = append(segments, pathSegment{value: segment, quoted: true})

			// Continue past the closing backtick.
			i += 1 + closing + 1
			if i == len(path) {
				return segments, nil
			}

			if path[i] != '.' {
				return nil, errors.New("expected '.' after backtick quote")
			}

			i++

			continue
		}

		j := i
		for j < len(path) && path[j] != '.' && path[j] != '`' {
			j++
		}

		if j < len(path) && path[j] == '`' {
			return nil, errors.New("backtick quote must span a whole segment")
		}

		segment := path[i:j]
		if segment == "" {
			return nil, errors.New("empty segment")
		}

		segments = append(segments, pathSegment{value: segment})

		if j == len(path) {
			return segments, nil
		}

		i = j + 1
	}
}

// cursor is a position inside a message type while resolving a path,
// segment by segment. Exactly one of the fields describes the position:
// a message whose fields can be named, a map whose key comes next, or a
// terminal value that no path may traverse.
type cursor struct {
	message  protoreflect.MessageDescriptor
	mapField protoreflect.FieldDescriptor
	repeated bool
	scalar   bool
}

// step advances the cursor by one path segment.
func (c cursor) step(segment pathSegment) (cursor, error) {
	// Quoting makes "*" a literal map key rather than the wildcard.
	if !segment.quoted && segment.value == WildcardPath {
		return cursor{}, errors.New("the wildcard is only valid as the entire mask")
	}

	switch {
	case c.mapField != nil:
		return c.stepMapKey(segment.value)
	case c.message != nil:
		fd := c.message.Fields().ByName(protoreflect.Name(segment.value))
		if fd == nil {
			return cursor{}, fmt.Errorf("unknown field %q", segment.value)
		}

		return cursorAt(fd), nil
	case c.repeated:
		if _, err := strconv.Atoi(segment.value); err == nil {
			return cursor{}, errors.New("cannot address elements of a repeated field by index")
		}

		return cursor{}, errors.New("cannot traverse a repeated field")
	default:
		return cursor{}, errors.New("field has no subfields")
	}
}

// stepMapKey advances the cursor past a map key segment.
func (c cursor) stepMapKey(segment string) (cursor, error) {
	if _, err := parseMapKey(c.mapField.MapKey(), segment); err != nil {
		return cursor{}, err
	}

	value := c.mapField.MapValue()
	if value.Kind() == protoreflect.MessageKind {
		return cursor{message: value.Message()}, nil
	}

	return cursor{scalar: true}, nil
}

// parseMapKey converts a path segment into a key of the map's key kind.
// Only string and integer keys can appear in a field mask path.
func parseMapKey(kd protoreflect.FieldDescriptor, segment string) (protoreflect.MapKey, error) {
	switch kd.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(segment).MapKey(), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		k, err := strconv.ParseInt(segment, 10, 32)
		if err != nil {
			return protoreflect.MapKey{}, fmt.Errorf("map key %q is not an integer", segment)
		}

		return protoreflect.ValueOfInt32(int32(k)).MapKey(), nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		k, err := strconv.ParseInt(segment, 10, 64)
		if err != nil {
			return protoreflect.MapKey{}, fmt.Errorf("map key %q is not an integer", segment)
		}

		return protoreflect.ValueOfInt64(k).MapKey(), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		k, err := strconv.ParseUint(segment, 10, 32)
		if err != nil {
			return protoreflect.MapKey{}, fmt.Errorf("map key %q is not an integer", segment)
		}

		return protoreflect.ValueOfUint32(uint32(k)).MapKey(), nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		k, err := strconv.ParseUint(segment, 10, 64)
		if err != nil {
			return protoreflect.MapKey{}, fmt.Errorf("map key %q is not an integer", segment)
		}

		return protoreflect.ValueOfUint64(k).MapKey(), nil
	default:
		return protoreflect.MapKey{}, errors.New(
			"only string and integer map keys can be addressed",
		)
	}
}

// mapKey converts a path segment into a key of the map's key kind. Check
// has validated the segment, so the conversion cannot fail. It also
// canonicalises the segment, so that the paths "editions.05" and
// "editions.5" address the one entry.
func mapKey(kd protoreflect.FieldDescriptor, segment string) protoreflect.MapKey {
	v, err := parseMapKey(kd, segment)
	if err != nil {
		panic(fmt.Sprintf("fieldmask: %v", err))
	}

	return v
}

// maskNode is one segment of a field mask path tree. A terminal node
// covers the whole subtree below its path.
type maskNode struct {
	children map[string]*maskNode
	terminal bool
}

// newMaskTree builds the path tree of an already [Check]-validated mask.
// Nodes are keyed by segment value: quoting distinguishes a literal from
// the wildcard while a path is resolved, and has served its purpose by now.
func newMaskTree(paths []string) *maskNode {
	root := &maskNode{children: map[string]*maskNode{}}

	for _, path := range paths {
		// Check has passed: the path splits.
		segments, err := splitPath(path)
		if err != nil {
			panic(fmt.Sprintf("fieldmask: path %q: %v", path, err))
		}

		values := make([]string, len(segments))
		for i, segment := range segments {
			values[i] = segment.value
		}

		root.insert(values)
	}

	return root
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

// cursorAt positions a cursor at the value of a field.
func cursorAt(fd protoreflect.FieldDescriptor) cursor {
	switch {
	case fd.IsMap():
		return cursor{mapField: fd}
	case fd.IsList():
		return cursor{repeated: true}
	case fd.Kind() == protoreflect.MessageKind:
		return cursor{message: fd.Message()}
	default:
		return cursor{scalar: true}
	}
}

// checkPath resolves one non-wildcard path against a message type.
func checkPath(md protoreflect.MessageDescriptor, path string) error {
	segments, err := splitPath(path)
	if err != nil {
		return err
	}

	c := cursor{message: md}
	for _, segment := range segments {
		c, err = c.step(segment)
		if err != nil {
			return err
		}
	}

	return nil
}
