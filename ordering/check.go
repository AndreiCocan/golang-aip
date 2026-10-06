package ordering

import (
	"fmt"

	"github.com/AndreiCocan/golang-aip/ordering/ast"
)

// Check validates a parsed order_by against schema and returns the
// [CheckedOrderBy]. Each field path must be declared in the schema. Check
// merges exact duplicates (same path, same direction) into the first one,
// and rejects a path that has two directions.
//
// Errors are [*CheckError] values matching [ErrInvalidOrderBy], carrying
// the byte offset of the offending field.
//
// schema must not be nil; use NewSchema() for a schema with no orderable
// fields.
func Check(orderBy *ast.OrderBy, schema *Schema) (*CheckedOrderBy, error) {
	checked := &CheckedOrderBy{}
	if orderBy == nil {
		return checked, nil
	}

	directions := make(map[string]bool, len(orderBy.Fields))

	for _, f := range orderBy.Fields {
		path := f.Path()
		if _, ok := schema.fields[path]; !ok {
			return nil, &CheckError{
				OrderBy: orderBy.Source,
				Pos:     f.Pos,
				Message: fmt.Sprintf("unknown ordering field %q", path),
			}
		}

		if desc, seen := directions[path]; seen {
			if desc != f.Desc {
				return nil, &CheckError{
					OrderBy: orderBy.Source,
					Pos:     f.Pos,
					Message: fmt.Sprintf(
						"field %q is ordered both ascending and descending",
						path,
					),
				}
			}

			// An exact duplicate merges into its first occurrence.
			continue
		}

		directions[path] = f.Desc
		checked.Keys = append(checked.Keys, Key{Segments: f.Segments, Desc: f.Desc})
	}

	return checked, nil
}
