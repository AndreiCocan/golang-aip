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

// String returns the segment as it appears in a path, in backticks when it
// is quoted.
func (s pathSegment) String() string {
	if s.quoted {
		return "`" + s.value + "`"
	}

	return s.value
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

// pathPosition is the place in a message type that a path reached, while
// the path is resolved segment by segment. Exactly one field describes the
// place: a message whose fields the next segment can name, a map whose key
// comes next, a repeated field, or a scalar that no path can go into.
type pathPosition struct {
	message  protoreflect.MessageDescriptor
	mapField protoreflect.FieldDescriptor
	repeated bool
	scalar   bool
}

// step moves the position by one path segment. throughRepeated tells
// whether the segment after a repeated message field can name a field of
// each element, which only a read mask allows.
func (pos pathPosition) step(segment pathSegment, throughRepeated bool) (pathPosition, error) {
	// Quoting makes "*" a literal map key rather than the wildcard.
	if !segment.quoted && segment.value == WildcardPath {
		return pathPosition{}, errors.New("the wildcard is only valid as the entire mask")
	}

	switch {
	case pos.mapField != nil:
		return pos.stepMapKey(segment.value)
	case pos.message != nil:
		fd := pos.message.Fields().ByName(protoreflect.Name(segment.value))
		if fd == nil {
			return pathPosition{}, fmt.Errorf("unknown field %q", segment.value)
		}

		return positionAt(fd, throughRepeated), nil
	case pos.repeated:
		if _, err := strconv.Atoi(segment.value); err == nil {
			return pathPosition{}, errors.New(
				"cannot address elements of a repeated field by index",
			)
		}

		return pathPosition{}, errors.New("cannot traverse a repeated field")
	default:
		return pathPosition{}, errors.New("field has no subfields")
	}
}

// stepMapKey moves the position past a map key segment.
func (pos pathPosition) stepMapKey(segment string) (pathPosition, error) {
	if _, err := parseMapKey(pos.mapField.MapKey(), segment); err != nil {
		return pathPosition{}, err
	}

	value := pos.mapField.MapValue()
	if value.Kind() == protoreflect.MessageKind {
		return pathPosition{message: value.Message()}, nil
	}

	return pathPosition{scalar: true}, nil
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

// mapKey converts a path segment into a key of the map's key kind.
// [CheckUpdate] or [CheckRead] validated the segment, so the conversion
// cannot fail. mapKey also makes the segment canonical, so that the paths
// "editions.05" and "editions.5" name the same entry.
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

// newMaskTree builds the path tree of a mask that [CheckUpdate] or
// [CheckRead] validated.
// Nodes are keyed by segment value: quoting distinguishes a literal from
// the wildcard while a path is resolved, and has served its purpose by now.
func newMaskTree(paths []string) *maskNode {
	root := &maskNode{children: map[string]*maskNode{}}

	for _, path := range paths {
		// The mask is valid, so the path splits.
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

// positionAt returns the position at the value of a field. When
// throughRepeated is true, the position of a repeated message field is the
// message type of its elements, so that the next segment names a field of
// each element.
func positionAt(fd protoreflect.FieldDescriptor, throughRepeated bool) pathPosition {
	switch {
	case fd.IsMap():
		return pathPosition{mapField: fd}
	case fd.IsList() && throughRepeated && fd.Kind() == protoreflect.MessageKind:
		return pathPosition{message: fd.Message()}
	case fd.IsList():
		return pathPosition{repeated: true}
	case fd.Kind() == protoreflect.MessageKind:
		return pathPosition{message: fd.Message()}
	default:
		return pathPosition{scalar: true}
	}
}

// checkPath resolves one non-wildcard path against a message type.
// throughRepeated tells whether the path can name the fields of the
// elements of a repeated message field.
func checkPath(md protoreflect.MessageDescriptor, path string, throughRepeated bool) error {
	segments, err := splitPath(path)
	if err != nil {
		return err
	}

	pos := pathPosition{message: md}
	for _, segment := range segments {
		pos, err = pos.step(segment, throughRepeated)
		if err != nil {
			return err
		}
	}

	return nil
}
