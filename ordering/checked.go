package ordering

import "strings"

// CheckedOrderBy is an order_by that [Check] validated against a [Schema]. It is
// the contract consumed by dialect packages: every field path is declared
// in the schema, exact duplicates are merged, and no field appears in two
// directions. No fields means the order_by was empty and the service's
// default order applies.
type CheckedOrderBy struct {
	Keys []Key
}

// Key is one validated ordering key: a field path and its direction.
type Key struct {
	// Segments holds the dotted path split at the dots: "author.name"
	// becomes {"author", "name"}.
	Segments []string
	// Desc reports whether the key orders descending. Ascending is the
	// default.
	Desc bool
}

// Path returns the dotted field path, such as "author.name".
func (k *Key) Path() string {
	return strings.Join(k.Segments, ".")
}
