package ordering

// Compile parses orderBy and checks it against schema in one step. It
// returns the errors of [Parse] and [Check]. Use Parse and Check
// separately when you need the syntax tree.
func Compile(orderBy string, schema *Schema) (*CheckedOrderBy, error) {
	parsed, err := Parse(orderBy)
	if err != nil {
		return nil, err
	}

	return Check(parsed, schema)
}
