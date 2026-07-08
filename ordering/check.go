package ordering

import (
	"fmt"

	"github.com/AndreiCocan/golang-aip/ordering/ast"
)

// Check validates a parsed order_by against a schema and resolves it into
// a [Checked] order_by: every field path must be declared in the schema,
// exact duplicates (same path, same direction) are merged into their first
// occurrence, and a path ordered in two contradictory directions is
// rejected.
//
// Errors are [*CheckError] values matching [ErrInvalidOrderBy], carrying
// the byte offset of the offending field.
//
// schema must not be nil; use NewSchema() for a schema with no orderable
// fields.
func Check(orderBy *ast.OrderBy, schema *Schema) (*Checked, error) {
	checked := &Checked{}
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
		checked.Fields = append(checked.Fields, Field{Segments: f.Segments, Desc: f.Desc})
	}

	return checked, nil
}
