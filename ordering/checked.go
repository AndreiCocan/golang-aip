package ordering

import "strings"

// CheckedOrderBy is an order_by that [Check] validated against a [Schema].
// Dialect packages read it: the schema declares each path, Check merged the
// exact duplicates, and no path has two directions. No keys means that the
// order_by was empty, and the default order of the service applies.
type CheckedOrderBy struct {
	Keys []Key
}

// Key is one validated sort key: a field path and its direction.
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
