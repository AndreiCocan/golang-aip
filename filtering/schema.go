package filtering

import (
	"errors"
	"fmt"
	"strings"
)

// Schema declares which fields a filter may reference and which functions
// it may call, with their types. [Check] validates every filter against a
// schema, so the schema doubles as the allowlist of filterable fields.
//
// A Schema is immutable once built and safe for concurrent use.
type Schema struct {
	fields map[string]Type
	funcs  map[string]*funcDecl
}

// Decl is one schema declaration: a [FieldDecl] or a [FuncDecl].
type Decl interface {
	declare(s *Schema)
}

// NewSchema builds a schema from field and function declarations.
//
// Schemas are programmer input, so NewSchema panics instead of returning an
// error: on a duplicate or empty name, an enum without values, or an
// invalid function declaration.
func NewSchema(decls ...Decl) *Schema {
	s := &Schema{
		fields: make(map[string]Type),
		funcs:  make(map[string]*funcDecl),
	}
	for _, d := range decls {
		d.declare(s)
	}

	return s
}

// Field resolves a dotted field path, such as "author.name", against the
// schema. It is intended for function expanders that need to build
// [Comparison] nodes. Map keys may be traversed like fields; repeated
// fields may be crossed.
func (s *Schema) Field(path string) (*Field, error) {
	field := &Field{}
	rest := path
	fields := s.fields

	var typ Type

	for first := true; rest != ""; first = false {
		name, tail, _ := strings.Cut(rest, ".")
		rest = tail

		if !first {
			var ok bool

			fields, ok = subfields(typ)
			if !ok {
				return nil, fmt.Errorf("field %q of %q is not a message", name, path)
			}
		}

		var ok bool

		typ, ok = fields[name]
		if !ok && fields != nil {
			return nil, fmt.Errorf("unknown field %q in %q", name, path)
		}

		if fields == nil {
			// Inside a map: any name is a key of the map's value type.
			typ = *field.Segments[len(field.Segments)-1].Type.Elem
		}

		field.Segments = append(field.Segments, FieldSegment{Name: name, Type: typ})
	}

	if len(field.Segments) == 0 {
		return nil, errors.New("empty field path")
	}

	return field, nil
}

// subfields returns the field set to resolve the next path segment in: the
// subfields of a message, or nil for a map, whose keys are unrestricted.
// The second result reports whether the type can be traversed at all.
func subfields(t Type) (map[string]Type, bool) {
	switch t.Kind {
	case KindMessage:
		return t.msg.fields, true
	case KindMap:
		return nil, true
	case KindRepeated:
		return subfields(*t.Elem)
	default:
		return nil, false
	}
}

// FieldDecl declares one filterable field. Build one with [String], [Int],
// [Float], [Bool], [Enum], [Timestamp], [Duration], [Message], [Repeated],
// or [Map].
type FieldDecl struct {
	name string
	typ  Type
}

func (d FieldDecl) declare(s *Schema) {
	if d.name == "" {
		panic("filtering: field declared with an empty name")
	}

	if _, ok := s.fields[d.name]; ok {
		panic(fmt.Sprintf("filtering: field %q declared twice", d.name))
	}

	s.fields[d.name] = d.typ
}

// String declares a string field.
func String(name string) FieldDecl {
	return FieldDecl{name: name, typ: Type{Kind: KindString}}
}

// Int declares a 64-bit signed integer field.
func Int(name string) FieldDecl {
	return FieldDecl{name: name, typ: Type{Kind: KindInt}}
}

// Float declares a 64-bit floating-point field.
func Float(name string) FieldDecl {
	return FieldDecl{name: name, typ: Type{Kind: KindFloat}}
}

// Bool declares a boolean field.
func Bool(name string) FieldDecl {
	return FieldDecl{name: name, typ: Type{Kind: KindBool}}
}

// Timestamp declares a point-in-time field, filtered with RFC 3339
// literals such as "2021-02-14T10:00:00Z".
func Timestamp(name string) FieldDecl {
	return FieldDecl{name: name, typ: Type{Kind: KindTimestamp}}
}

// Duration declares a duration field, filtered with seconds literals such
// as 20s or 1.5s.
func Duration(name string) FieldDecl {
	return FieldDecl{name: name, typ: Type{Kind: KindDuration}}
}

// Enum declares a field restricted to the given case-sensitive value
// names. NewSchema panics if no values are given.
func Enum(name string, values ...string) FieldDecl {
	if len(values) == 0 {
		panic(fmt.Sprintf("filtering: enum field %q declared without values", name))
	}

	return FieldDecl{name: name, typ: Type{Kind: KindEnum, Enum: values}}
}

// Message declares a structured field with the given subfields, traversed
// with dots: `author.name = "Hugo"`.
func Message(name string, fields ...FieldDecl) FieldDecl {
	msg := &messageType{fields: make(map[string]Type, len(fields))}
	for _, f := range fields {
		if f.name == "" {
			panic(fmt.Sprintf("filtering: subfield of %q declared with an empty name", name))
		}

		if _, ok := msg.fields[f.name]; ok {
			panic(fmt.Sprintf("filtering: subfield %q of %q declared twice", f.name, name))
		}

		msg.fields[f.name] = f.typ
	}

	return FieldDecl{name: name, typ: Type{Kind: KindMessage, msg: msg}}
}

// Repeated declares a list field whose elements are described by elem,
// which contributes both the field's name and the element type:
// Repeated(String("tags")) is a list of strings named "tags". Repeated
// fields are queried with the has operator: `tags:go`.
func Repeated(elem FieldDecl) FieldDecl {
	if elem.typ.Kind == KindRepeated || elem.typ.Kind == KindMap {
		panic(
			fmt.Sprintf("filtering: repeated field %q of repeated or map element type", elem.name),
		)
	}

	t := elem.typ

	return FieldDecl{name: elem.name, typ: Type{Kind: KindRepeated, Elem: &t}}
}

// Map declares a string-keyed map field whose values are described by
// value, which contributes both the field's name and the value type:
// Map(String("labels")) is a map from string keys to string values named
// "labels". Maps are queried by key: `labels:env`, `labels.env = prod`.
func Map(value FieldDecl) FieldDecl {
	if value.typ.Kind == KindRepeated || value.typ.Kind == KindMap {
		panic(fmt.Sprintf("filtering: map field %q of repeated or map value type", value.name))
	}

	t := value.typ

	return FieldDecl{name: value.name, typ: Type{Kind: KindMap, Elem: &t}}
}

// ExpandFunc rewrites a function call into a checked filter expression at
// Check time. It receives the schema being checked against (use
// [Schema.Field] to resolve field paths) and the call's literal arguments.
//
// Returning an error fails the Check; return a [*CheckError] to report an
// invalid filter, or any other error to report an internal problem.
type ExpandFunc func(s *Schema, args []Value) (Expr, error)

// FuncDecl declares a filter function. Build one with [Function].
type FuncDecl struct {
	name    string
	args    []Kind
	result  Kind
	expand  ExpandFunc
	declErr string
}

func (d FuncDecl) declare(s *Schema) {
	if d.declErr != "" {
		panic("filtering: " + d.declErr)
	}

	if d.name == "" {
		panic("filtering: function declared with an empty name")
	}

	if _, ok := s.funcs[d.name]; ok {
		panic(fmt.Sprintf("filtering: function %q declared twice", d.name))
	}

	if d.result == KindInvalid {
		if d.expand == nil {
			panic(fmt.Sprintf("filtering: function %q must declare Returns or Expand", d.name))
		}

		d.result = KindBool
	}

	if d.expand != nil && d.result != KindBool {
		panic(fmt.Sprintf("filtering: function %q has an expander and must return bool", d.name))
	}

	s.funcs[d.name] = &funcDecl{
		name:   d.name,
		args:   d.args,
		result: Type{Kind: d.result},
		expand: d.expand,
	}
}

type funcDecl struct {
	name   string
	args   []Kind
	result Type
	expand ExpandFunc
}

// FunctionOption configures a [Function] declaration.
type FunctionOption func(*FuncDecl)

// Function declares a filter function, callable as `name(args...)`. Names
// may be dotted, like "math.mem". A function either carries an [Expand]
// rewrite (a macro every dialect supports) or is passed through to the
// dialect as a [FuncCall] to translate natively.
//
// Per AIP filtering semantics a service must document the functions it
// supports; an undeclared function fails [Check].
func Function(name string, opts ...FunctionOption) FuncDecl {
	d := FuncDecl{name: name}
	for _, opt := range opts {
		opt(&d)
	}

	return d
}

// Args declares the function's parameter kinds; calls must match the exact
// arity. Only scalar kinds are allowed.
func Args(kinds ...Kind) FunctionOption {
	return func(d *FuncDecl) {
		for _, k := range kinds {
			if !scalarKind(k) {
				d.declErr = fmt.Sprintf("function %q declares a non-scalar %v argument", d.name, k)
			}
		}

		d.args = kinds
	}
}

// Returns declares the function's result kind. Only scalar kinds are
// allowed. A function used as a bare restriction, like `overdue()`, must
// return KindBool.
func Returns(kind Kind) FunctionOption {
	return func(d *FuncDecl) {
		if !scalarKind(kind) {
			d.declErr = fmt.Sprintf("function %q declares a non-scalar %v result", d.name, kind)
		}

		d.result = kind
	}
}

// Expand attaches a macro expander: at Check time the call is rewritten
// into the returned expression, so the function works on every dialect
// without backend support. Expanded functions must take literal arguments
// and return bool.
func Expand(fn ExpandFunc) FunctionOption {
	return func(d *FuncDecl) { d.expand = fn }
}

// scalarKind reports whether k is a scalar kind usable as a function
// argument or result.
func scalarKind(k Kind) bool {
	switch k {
	case KindString, KindInt, KindFloat, KindBool, KindTimestamp, KindDuration:
		return true
	default:
		return false
	}
}
