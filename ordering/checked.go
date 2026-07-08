package ordering

import "strings"

// Checked is an order_by that [Check] validated against a [Schema]. It is
// the contract consumed by dialect packages: every field path is declared
// in the schema, exact duplicates are merged, and no field appears in two
// directions. No fields means the order_by was empty and the service's
// default order applies.
type Checked struct {
	Fields []Field
}

// Field is one validated ordering key: a field path and its direction.
type Field struct {
	// Segments holds the dotted path split at the dots: "author.name"
	// becomes {"author", "name"}.
	Segments []string
	// Desc reports whether the key orders descending. Ascending is the
	// default.
	Desc bool
}

// Path returns the dotted field path, such as "author.name".
func (f *Field) Path() string {
	return strings.Join(f.Segments, ".")
}
