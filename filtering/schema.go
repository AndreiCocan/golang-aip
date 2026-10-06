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
// Schemas are programmer input, so NewSchema panics instead of returning an
// error: on a duplicate or empty name, or an invalid function declaration.
// The field constructors, such as [EnumField] and [MessageField], panic on their own
// invalid input.
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

// subfieldType returns the type of the path segment name below a field of type t:
// a subfield of a message, the value of any key of a map, or a subfield of
// the message elements of a repeated field. It reports false when t has no
// such segment.
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

// FieldDecl declares one filterable field. Build one with [StringField], [IntField],
// [FloatField], [BoolField], [EnumField], [TimestampField], [DurationField], [MessageField], [RepeatedField],
// or [MapField].
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

// RepeatedField declares a list field whose elements are described by elem,
// which contributes both the field's name and the element type:
// RepeatedField(String("tags")) is a list of strings named "tags". RepeatedField
// fields are queried with the has operator: `tags:go`.
func RepeatedField(elem FieldDecl) FieldDecl {
	if elem.typ.Kind == KindRepeated || elem.typ.Kind == KindMap {
		panic(
			fmt.Sprintf("filtering: repeated field %q of repeated or map element type", elem.name),
		)
	}

	t := elem.typ

	return FieldDecl{name: elem.name, typ: Type{Kind: KindRepeated, Elem: &t}}
}

// MapField declares a string-keyed map field whose values are described by
// value, which contributes both the field's name and the value type:
// MapField(String("labels")) is a map from string keys to string values named
// "labels". Maps are queried by key: `labels:env`, `labels.env = prod`.
func MapField(value FieldDecl) FieldDecl {
	if value.typ.Kind == KindRepeated || value.typ.Kind == KindMap {
		panic(fmt.Sprintf("filtering: map field %q of repeated or map value type", value.name))
	}

	t := value.typ

	return FieldDecl{name: value.name, typ: Type{Kind: KindMap, Elem: &t}}
}

// Expander rewrites a function call into a checked filter expression at
// Check time. It receives the schema being checked against (use
// [Schema.Field] to resolve field paths) and the call's literal arguments.
//
// Returning an error fails the Check; return a [*CheckError] to report an
// invalid filter, or any other error to report an internal problem.
type Expander func(s *Schema, args []Value) (Expr, error)

// FuncDecl declares a filter function. Build one with [Func].
type FuncDecl struct {
	name    string
	args    []Kind
	result  Kind
	expand  Expander
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

	s.funcs[d.name] = &declaredFunc{
		name:   d.name,
		args:   d.args,
		result: Type{Kind: d.result},
		expand: d.expand,
	}
}

type declaredFunc struct {
	name   string
	args   []Kind
	result Type
	expand Expander
}

// FuncOption configures a [Func] declaration.
type FuncOption func(*FuncDecl)

// Func declares a filter function, callable as `name(args...)`. Names
// may be dotted, like "math.mem". A function either carries an [FuncExpand]
// rewrite (a macro every dialect supports) or is passed through to the
// dialect as a [FuncCall] to translate natively.
//
// Per AIP filtering semantics a service must document the functions it
// supports; an undeclared function fails [Check].
func Func(name string, opts ...FuncOption) FuncDecl {
	d := FuncDecl{name: name}
	for _, opt := range opts {
		opt(&d)
	}

	return d
}

// FuncArgs declares the function's parameter kinds; calls must match the exact
// arity. Only scalar kinds are allowed.
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

// FuncReturns declares the function's result kind. Only scalar kinds are
// allowed. A function used as a bare restriction, like `overdue()`, must
// return KindBool.
func FuncReturns(kind Kind) FuncOption {
	return func(d *FuncDecl) {
		if !isScalarKind(kind) {
			d.declErr = fmt.Sprintf("function %q declares a non-scalar %v result", d.name, kind)
		}

		d.result = kind
	}
}

// FuncExpand attaches a macro expander: at Check time the call is rewritten
// into the returned expression, so the function works on every dialect
// without backend support. Expanded functions must take literal arguments
// and return bool.
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
