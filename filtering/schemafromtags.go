package filtering

import (
	"fmt"
	"reflect"
	"time"

	"github.com/AndreiCocan/golang-aip/aiptag"
)

// SchemaFromTags builds a schema from the aip tags of the struct v, such as
// Book{}, and from the extra declarations decls, such as functions. Each
// field with the filter option becomes a field of the schema, with the API
// name of its tag and the type of its Go type:
//
//   - string: [StringField]
//   - int, int8, int16, int32, int64: [IntField]
//   - float32, float64: [FloatField]
//   - bool: [BoolField]
//   - [time.Time]: [TimestampField]
//   - [time.Duration]: [DurationField]
//   - a slice of one of the types above: [RepeatedField]
//   - a map with string keys and values of one of the types above: [MapField]
//
// The search option does not change the schema: a [Search] node does not
// name fields. A dialect reads the search option of the tags to match the
// terms. See [github.com/AndreiCocan/golang-aip/aiptag] for the tag.
//
// Schemas are programmer input, so SchemaFromTags panics instead of returning
// an error: when v is not a struct, when a tag is not valid, when a
// filterable field has a Go type that is not in the list above, when a
// field with the search option is not a string, or when two declarations
// have the same name.
func SchemaFromTags(v any, decls ...Decl) *Schema {
	t := reflect.TypeOf(v)
	if t == nil || t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("filtering: SchemaFromTags: %T is not a struct", v))
	}

	fields := make([]Decl, 0, t.NumField()+len(decls))

	for f := range t.Fields() {
		tag, ok, err := aiptag.Parse(f)
		if err != nil {
			panic("filtering: SchemaFromTags: " + err.Error())
		}

		if !ok {
			continue
		}

		if tag.Searchable && f.Type.Kind() != reflect.String {
			panic(fmt.Sprintf(
				"filtering: SchemaFromTags: field %s has the search option and is not a string",
				f.Name,
			))
		}

		if !tag.Filterable {
			continue
		}

		decl, ok := declFor(tag.Name, f.Type)
		if !ok {
			panic(fmt.Sprintf(
				"filtering: SchemaFromTags: field %s has the type %v, which a filter cannot compare",
				f.Name,
				f.Type,
			))
		}

		fields = append(fields, decl)
	}

	return NewSchema(append(fields, decls...)...)
}

// declFor returns the declaration of a field with the API name and the
// Go type t. It reports false when no declaration fits t.
func declFor(name string, t reflect.Type) (FieldDecl, bool) {
	if decl, ok := scalarDeclFor(name, t); ok {
		return decl, true
	}

	switch t.Kind() {
	case reflect.Slice:
		elem, ok := scalarDeclFor(name, t.Elem())

		return RepeatedField(elem), ok
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return FieldDecl{}, false
		}

		value, ok := scalarDeclFor(name, t.Elem())

		return MapField(value), ok
	default:
		return FieldDecl{}, false
	}
}

// scalarDeclFor returns the declaration of a scalar field with the API name
// and the Go type t. It reports false when t is not a scalar type.
func scalarDeclFor(name string, t reflect.Type) (FieldDecl, bool) {
	switch t {
	case reflect.TypeFor[time.Time]():
		return TimestampField(name), true
	case reflect.TypeFor[time.Duration]():
		return DurationField(name), true
	}

	switch t.Kind() {
	case reflect.String:
		return StringField(name), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return IntField(name), true
	case reflect.Float32, reflect.Float64:
		return FloatField(name), true
	case reflect.Bool:
		return BoolField(name), true
	default:
		return FieldDecl{}, false
	}
}
