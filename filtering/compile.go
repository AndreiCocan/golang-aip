package filtering

// Compile parses filter and checks it against schema in one step. It
// returns the errors of [Parse] and [Check]. Use Parse and Check
// separately when you need the syntax tree.
func Compile(filter string, schema *Schema) (*CheckedFilter, error) {
	parsed, err := Parse(filter)
	if err != nil {
		return nil, err
	}

	return Check(parsed, schema)
}
