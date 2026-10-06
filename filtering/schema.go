package filtering

import (
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
	funcs  map[string]*declaredFunc
}

// Decl is one schema declaration: a [FieldDecl] or a [FuncDecl].
type Decl interface {
	declare(s *Schema)
}

// NewSchema builds a schema from field and function declarations.
//
// A schema is programmer input, so NewSchema panics instead of returning an
// error: on a duplicate or empty name, or on a function declaration that is
// not valid. The field constructors, such as [EnumField] and
// [MessageField], panic on their own input that is not valid.
func NewSchema(decls ...Decl) *Schema {
	s := &Schema{
		fields: make(map[string]Type),
		funcs:  make(map[string]*declaredFunc),
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
	names := strings.Split(path, ".")

	t, ok := s.fields[names[0]]
	if !ok {
		return nil, fmt.Errorf("unknown field %q in %q", names[0], path)
	}

	field := &Field{Segments: make([]FieldSegment, 0, len(names))}
	field.Segments = append(field.Segments, FieldSegment{Name: names[0], Type: t})

	for _, name := range names[1:] {
		if t, ok = subfieldType(t, name); !ok {
			return nil, fmt.Errorf("unknown field %q in %q", name, path)
		}

		field.Segments = append(field.Segments, FieldSegment{Name: name, Type: t})
	}

	return field, nil
}

// subfieldType returns the type of the path segment name below a field of
// type t: a subfield of a message, the value of any key of a map, or a
// subfield of the message elements of a repeated field. It reports false
// when t has no such segment.
func subfieldType(t Type, name string) (Type, bool) {
	switch t.Kind {
	case KindMessage:
		sub, ok := t.msg.fields[name]

		return sub, ok
	case KindMap:
		return *t.Elem, true
	case KindRepeated:
		if t.Elem.Kind != KindMessage {
			return Type{}, false
		}

		return subfieldType(*t.Elem, name)
	default:
		return Type{}, false
	}
}

// FieldDecl declares one field that a filter can name. Build one with
// [StringField], [IntField], [FloatField], [BoolField], [EnumField],
// [TimestampField], [DurationField], [MessageField], [RepeatedField], or
// [MapField].
type FieldDecl struct {
	// name is the API name of the field.
	name string
	// typ is the type of the field.
	typ Type
}

// declare adds the field to s. It panics when the name is empty, or when s
// already has a field with the name.
func (d FieldDecl) declare(s *Schema) {
	if d.name == "" {
		panic("filtering: field declared with an empty name")
	}

	if _, ok := s.fields[d.name]; ok {
		panic(fmt.Sprintf("filtering: field %q declared twice", d.name))
	}

	s.fields[d.name] = d.typ
}

// StringField declares a string field.
func StringField(name string) FieldDecl {
	return FieldDecl{name: name, typ: Type{Kind: KindString}}
}

// IntField declares a 64-bit signed integer field.
func IntField(name string) FieldDecl {
	return FieldDecl{name: name, typ: Type{Kind: KindInt}}
}

// FloatField declares a 64-bit floating-point field.
func FloatField(name string) FieldDecl {
	return FieldDecl{name: name, typ: Type{Kind: KindFloat}}
}

// BoolField declares a boolean field.
func BoolField(name string) FieldDecl {
	return FieldDecl{name: name, typ: Type{Kind: KindBool}}
}

// TimestampField declares a point-in-time field, filtered with RFC 3339
// literals such as "2021-02-14T10:00:00Z".
func TimestampField(name string) FieldDecl {
	return FieldDecl{name: name, typ: Type{Kind: KindTimestamp}}
}

// DurationField declares a duration field, filtered with seconds literals such
// as 20s or 1.5s.
func DurationField(name string) FieldDecl {
	return FieldDecl{name: name, typ: Type{Kind: KindDuration}}
}

// EnumField declares a field restricted to the given case-sensitive value
// names. EnumField panics if no values are given.
func EnumField(name string, values ...string) FieldDecl {
	if len(values) == 0 {
		panic(fmt.Sprintf("filtering: enum field %q declared without values", name))
	}

	return FieldDecl{name: name, typ: Type{Kind: KindEnum, Enum: values}}
}

// MessageField declares a structured field with the given subfields, traversed
// with dots: `author.name = "Hugo"`.
func MessageField(name string, fields ...FieldDecl) FieldDecl {
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

// RepeatedField declares a list field. elem gives both the name of the
// field and the type of its elements: RepeatedField(StringField("tags")) is
// a list of strings with the name "tags". A filter tests a repeated field
// with the has operator: `tags:go`.
func RepeatedField(elem FieldDecl) FieldDecl {
	if elem.typ.Kind == KindRepeated || elem.typ.Kind == KindMap {
		panic(
			fmt.Sprintf("filtering: repeated field %q of repeated or map element type", elem.name),
		)
	}

	t := elem.typ

	return FieldDecl{name: elem.name, typ: Type{Kind: KindRepeated, Elem: &t}}
}

// MapField declares a map field with string keys. value gives both the name
// of the field and the type of its values: MapField(StringField("labels"))
// is a map from string keys to string values with the name "labels". A
// filter names a map value by its key: `labels:env`, `labels.env = prod`.
func MapField(value FieldDecl) FieldDecl {
	if value.typ.Kind == KindRepeated || value.typ.Kind == KindMap {
		panic(fmt.Sprintf("filtering: map field %q of repeated or map value type", value.name))
	}

	t := value.typ

	return FieldDecl{name: value.name, typ: Type{Kind: KindMap, Elem: &t}}
}

// Expander replaces a call to a macro function with a checked filter
// expression, during [Check]. It receives the schema of the check and the
// literal arguments of the call. Use [Schema.Field] to resolve field paths.
//
// An error from an Expander stops Check. Return a [*CheckError] for an
// invalid filter. Any other error tells about an internal problem.
type Expander func(s *Schema, args []Value) (Expr, error)

// FuncDecl declares a filter function. Build one with [Func].
type FuncDecl struct {
	// name is the name of the function, which can have dots.
	name string
	// args holds the kind of each argument.
	args []Kind
	// result is the kind of the result, or KindInvalid when no option set
	// it.
	result Kind
	// expand rewrites a call, or is nil.
	expand Expander
	// declErr is the error of an option that is not valid, or "".
	declErr string
}

// declare adds the function to s. It panics when an option is not valid,
// when the name is empty, when s already has a function with the name,
// when the function has neither a result nor an expander, and when a
// function with an expander does not return bool.
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

	s.funcs[d.name] = &declaredFunc{
		name:   d.name,
		args:   d.args,
		result: Type{Kind: d.result},
		expand: d.expand,
	}
}

// declaredFunc is a function of a [Schema], after [FuncDecl.declare] checked
// it.
type declaredFunc struct {
	// name is the name of the function.
	name string
	// args holds the kind of each argument.
	args []Kind
	// result is the type of the result.
	result Type
	// expand rewrites a call, or is nil.
	expand Expander
}

// FuncOption sets an option of a [Func] declaration.
type FuncOption func(*FuncDecl)

// Func declares a filter function that a filter calls as `name(args...)`.
// The name can have dots, such as "math.mem". A function with the
// [FuncExpand] option is a macro, and all dialects support it. [Check] keeps
// a call to a function without it as a [FuncCall], for the dialect to
// translate.
//
// AIP-160 requires a service to document the functions that it supports.
// Check rejects a call to a function that the schema does not declare.
func Func(name string, opts ...FuncOption) FuncDecl {
	d := FuncDecl{name: name}
	for _, opt := range opts {
		opt(&d)
	}

	return d
}

// FuncArgs declares the kinds of the arguments of the function. A call must
// have exactly that number of arguments. Only scalar kinds are valid.
func FuncArgs(kinds ...Kind) FuncOption {
	return func(d *FuncDecl) {
		for _, k := range kinds {
			if !isScalarKind(k) {
				d.declErr = fmt.Sprintf("function %q declares a non-scalar %v argument", d.name, k)
			}
		}

		d.args = kinds
	}
}

// FuncReturns declares the kind of the result of the function. Only scalar
// kinds are valid. A function that a filter calls alone, such as
// `overdue()`, must return KindBool.
func FuncReturns(kind Kind) FuncOption {
	return func(d *FuncDecl) {
		if !isScalarKind(kind) {
			d.declErr = fmt.Sprintf("function %q declares a non-scalar %v result", d.name, kind)
		}

		d.result = kind
	}
}

// FuncExpand makes the function a macro: [Check] replaces each call with
// the expression that fn returns. Thus all dialects support the function.
// The arguments of a call must be literals, and the function must return
// bool.
func FuncExpand(fn Expander) FuncOption {
	return func(d *FuncDecl) { d.expand = fn }
}

// isScalarKind reports whether k is a scalar kind usable as a function
// argument or result.
func isScalarKind(k Kind) bool {
	switch k {
	case KindString, KindInt, KindFloat, KindBool, KindTimestamp, KindDuration:
		return true
	default:
		return false
	}
}
