package fieldmask

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// MarshalJSON returns the JSON of the fields of msg that the read mask
// covers, as [Prune] would keep them. A covered field is written also when
// it has its default value: a false bool, a zero number, an empty string,
// an empty list or map. A covered message field that is unset is left out.
// The fields that the mask does not cover are left out.
//
// A nil or empty mask, and the "*" mask, cover every field. The JSON is
// the canonical protobuf JSON mapping, with the JSON names of the fields,
// without insignificant whitespace. A path into a message of the
// google.protobuf package, such as a Timestamp, covers the whole message,
// because the JSON of such a message is not an object of its fields.
//
// MarshalJSON returns an error that matches [ErrInvalidFieldMask] for the
// masks that [CheckRead] rejects, and an error when msg cannot be written
// as JSON. msg is not modified.
func MarshalJSON(mask *fieldmaskpb.FieldMask, msg proto.Message) ([]byte, error) {
	if err := CheckRead(mask, msg); err != nil {
		return nil, err
	}

	data, err := protojson.MarshalOptions{EmitDefaultValues: true}.Marshal(msg)
	if err != nil {
		return nil, err
	}

	if len(mask.GetPaths()) > 0 && !IsWildcard(mask) {
		data, err = filterMessage(
			data,
			msg.ProtoReflect().Descriptor(),
			newMaskTree(mask.GetPaths()),
		)
		if err != nil {
			return nil, fmt.Errorf("fieldmask: filter JSON: %w", err)
		}
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		return nil, fmt.Errorf("fieldmask: compact JSON: %w", err)
	}

	return compact.Bytes(), nil
}

// filterMessage removes from data, the JSON of a message of type md, the
// members that the mask tree node does not cover.
func filterMessage(data []byte, md protoreflect.MessageDescriptor, node *maskNode) ([]byte, error) {
	if !isObject(data) || isWellKnown(md) {
		return data, nil
	}

	return filterObject(data, func(key string, value []byte) ([]byte, bool, error) {
		fd := md.Fields().ByJSONName(key)
		if fd == nil {
			return nil, false, nil
		}

		child, ok := node.children[string(fd.Name())]
		if !ok {
			return nil, false, nil
		}

		if child.terminal {
			return value, true, nil
		}

		var err error

		switch {
		case fd.IsMap():
			value, err = filterMapEntries(value, fd, child)
		case fd.IsList():
			value, err = filterList(value, fd.Message(), child)
		default:
			value, err = filterMessage(value, fd.Message(), child)
		}

		return value, true, err
	})
}

// filterMapEntries removes from data, the JSON of the map field fd, the
// entries that the mask tree node does not cover. Segments are
// canonicalised into map keys before they are matched, as in [Prune].
func filterMapEntries(
	data []byte,
	fd protoreflect.FieldDescriptor,
	node *maskNode,
) ([]byte, error) {
	if !isObject(data) {
		return data, nil
	}

	covered := make(map[string]*maskNode, len(node.children))
	for segment, child := range node.children {
		covered[mapKey(fd.MapKey(), segment).String()] = child
	}

	return filterObject(data, func(key string, value []byte) ([]byte, bool, error) {
		k, err := parseMapKey(fd.MapKey(), key)
		if err != nil {
			// No segment can name the key, so the mask does not cover it.
			return nil, false, nil //nolint:nilerr
		}

		child, ok := covered[k.String()]
		if !ok {
			return nil, false, nil
		}

		if child.terminal {
			return value, true, nil
		}

		// Below a key the value is a message; CheckRead has passed.
		value, err = filterMessage(value, fd.MapValue().Message(), child)

		return value, true, err
	})
}

// filterList filters each element of data, the JSON of a repeated message
// field of type md, by the mask tree node.
func filterList(data []byte, md protoreflect.MessageDescriptor, node *maskNode) ([]byte, error) {
	var elements []json.RawMessage
	if err := json.Unmarshal(data, &elements); err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	buf.WriteByte('[')

	for i, element := range elements {
		filtered, err := filterMessage(element, md, node)
		if err != nil {
			return nil, err
		}

		if i > 0 {
			buf.WriteByte(',')
		}

		buf.Write(filtered)
	}

	buf.WriteByte(']')

	return buf.Bytes(), nil
}

// filterObject calls keep on each member of the JSON object data, in
// order, and returns the object of the members that keep keeps, with the
// values that it returns.
func filterObject(
	data []byte,
	keep func(key string, value []byte) ([]byte, bool, error),
) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))

	// The opening brace.
	if _, err := dec.Token(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	buf.WriteByte('{')

	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}

		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("object key %v is not a string", token)
		}

		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, err
		}

		filtered, kept, err := keep(key, value)
		if err != nil {
			return nil, err
		}

		if !kept {
			continue
		}

		encodedKey, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}

		if buf.Len() > 1 {
			buf.WriteByte(',')
		}

		buf.Write(encodedKey)
		buf.WriteByte(':')
		buf.Write(filtered)
	}

	buf.WriteByte('}')

	return buf.Bytes(), nil
}

// isObject reports whether data is a JSON object.
func isObject(data []byte) bool {
	trimmed := bytes.TrimSpace(data)

	return len(trimmed) > 0 && trimmed[0] == '{'
}

// isWellKnown reports whether md is in the google.protobuf package, whose
// messages can have a JSON that is not an object of their fields.
func isWellKnown(md protoreflect.MessageDescriptor) bool {
	return strings.HasPrefix(string(md.FullName()), "google.protobuf.")
}
