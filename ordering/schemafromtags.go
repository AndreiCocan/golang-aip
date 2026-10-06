package ordering

import (
	"fmt"
	"reflect"

	"github.com/AndreiCocan/golang-aip/aiptag"
)

// SchemaFromTags builds a schema from the aip tags of the struct v, such as
// Book{}. Each field with the order option becomes an orderable field, with
// the API name of its tag. See [github.com/AndreiCocan/golang-aip/aiptag]
// for the tag.
//
// Schemas are programmer input, so SchemaFromTags panics instead of returning
// an error: when v is not a struct, when a tag is not valid, or when two
// orderable fields have the same name.
func SchemaFromTags(v any) *Schema {
	t := reflect.TypeOf(v)
	if t == nil || t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("ordering: SchemaFromTags: %T is not a struct", v))
	}

	var paths []string

	for f := range t.Fields() {
		tag, ok, err := aiptag.Parse(f)
		if err != nil {
			panic("ordering: SchemaFromTags: " + err.Error())
		}

		if ok && tag.Orderable {
			paths = append(paths, tag.Name)
		}
	}

	return NewSchema(paths...)
}
