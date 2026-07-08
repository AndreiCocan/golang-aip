package ordering

import (
	"fmt"
	"strings"
)

// Schema declares which dotted field paths an order_by may sort by.
// [Check] validates every order_by against a schema, so the schema doubles
// as the allowlist of orderable fields.
//
// A Schema is immutable once built and safe for concurrent use.
type Schema struct {
	fields map[string]struct{}
}

// NewSchema builds a schema from dotted field paths, such as "create_time"
// or "author.name". Orderable fields are declared by their full path;
// there is no need to declare intermediate segments.
//
// Schemas are programmer input, so NewSchema panics instead of returning
// an error: on an empty path, an empty path segment, or a duplicate path.
func NewSchema(paths ...string) *Schema {
	s := &Schema{fields: make(map[string]struct{}, len(paths))}
	for _, p := range paths {
		if p == "" {
			panic("ordering: field declared with an empty name")
		}

		for seg := range strings.SplitSeq(p, ".") {
			if seg == "" {
				panic(fmt.Sprintf("ordering: field %q has an empty segment", p))
			}
		}

		if _, ok := s.fields[p]; ok {
			panic(fmt.Sprintf("ordering: field %q declared twice", p))
		}

		s.fields[p] = struct{}{}
	}

	return s
}
