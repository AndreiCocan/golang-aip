package ast

// OrderBy is the root of a parsed order_by expression. An empty order_by
// is valid and yields no fields: the service's default order applies.
type OrderBy struct {
	// Source is the order_by string the tree was parsed from. Positions
	// throughout the tree are byte offsets into Source.
	Source string
	Fields []Field
}

// Field is one ordering key as written: a dotted field path and whether
// the "desc" suffix reversed its direction. The tree is grammar-faithful:
// fields appear in written order, duplicates included.
type Field struct {
	// Pos is the byte offset of the field's first segment.
	Pos int
	// Segments holds the dotted path split at the dots: "author.name"
	// parses to {"author", "name"}.
	Segments []string
	// Desc reports whether the field carries the "desc" suffix. Ascending
	// is the default and has no suffix.
	Desc bool
}

// Path returns the dotted field path, such as "author.name".
func (f *Field) Path() string {
	n := len(f.Segments) - 1
	for _, s := range f.Segments {
		n += len(s)
	}

	b := make([]byte, 0, n)

	for i, s := range f.Segments {
		if i > 0 {
			b = append(b, '.')
		}

		b = append(b, s...)
	}

	return string(b)
}
