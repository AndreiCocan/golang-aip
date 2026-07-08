package pagination

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"strconv"
	"strings"
)

// hashArgs checksums the request arguments a token must be consistent
// with. Arguments are hashed through their fmt representation with a
// length prefix, so distinct splits of the same text stay distinct;
// pointers are dereferenced so *string and string arguments hash alike.
// No arguments hash to zero, which keeps a zero Token interchangeable
// with Parse("").
func hashArgs(args []any) uint64 {
	if len(args) == 0 {
		return 0
	}

	var b strings.Builder

	for _, arg := range args {
		v := reflect.ValueOf(arg)
		for v.Kind() == reflect.Pointer && !v.IsNil() {
			v = v.Elem()
		}

		s := ""
		if v.IsValid() {
			s = fmt.Sprintf("%v", v.Interface())
		}

		b.WriteString(strconv.Itoa(len(s)))
		b.WriteString(":")
		b.WriteString(s)
	}

	return hashString(b.String())
}

// hashShape checksums the structure a cursor is encoded from, so a token
// minted from one shape cannot silently decode into another: gob matches
// struct fields by name and would zero a renamed field instead of
// failing. The checksum covers exported field names and types
// recursively, not type names, so moving a cursor type between packages
// keeps old tokens valid.
func hashShape(t reflect.Type) uint64 {
	var b strings.Builder

	writeShape(&b, t, map[reflect.Type]bool{})

	return hashString(b.String())
}

// hashString is 64-bit FNV-1a over the bytes of s. hash.Hash documents
// that Write never returns an error, hence the discarded returns.
func hashString(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))

	return h.Sum64()
}

// writeShape writes a structural description of t. Types already on the
// path are cycles, terminated to keep recursion finite. Structs with no
// exported fields are opaque to reflection, typically because they encode
// through GobEncoder like time.Time, and are described by name instead.
func writeShape(b *strings.Builder, t reflect.Type, path map[reflect.Type]bool) {
	if path[t] {
		b.WriteString("cycle;")

		return
	}

	path[t] = true
	defer delete(path, t)

	switch t.Kind() {
	case reflect.Pointer:
		b.WriteString("*")
		writeShape(b, t.Elem(), path)
	case reflect.Slice:
		b.WriteString("[]")
		writeShape(b, t.Elem(), path)
	case reflect.Array:
		b.WriteString("[")
		b.WriteString(strconv.Itoa(t.Len()))
		b.WriteString("]")
		writeShape(b, t.Elem(), path)
	case reflect.Map:
		b.WriteString("map[")
		writeShape(b, t.Key(), path)
		b.WriteString("]")
		writeShape(b, t.Elem(), path)
	case reflect.Struct:
		writeStructShape(b, t, path)
	default:
		b.WriteString(t.Kind().String())
		b.WriteString(";")
	}
}

func writeStructShape(b *strings.Builder, t reflect.Type, path map[reflect.Type]bool) {
	exported := make([]reflect.StructField, 0, t.NumField())

	for f := range t.Fields() {
		if f.IsExported() {
			exported = append(exported, f)
		}
	}

	if len(exported) == 0 {
		b.WriteString(t.String())
		b.WriteString(";")

		return
	}

	b.WriteString("struct{")

	for _, f := range exported {
		b.WriteString(f.Name)
		b.WriteString(" ")
		writeShape(b, f.Type, path)
	}

	b.WriteString("}")
}
