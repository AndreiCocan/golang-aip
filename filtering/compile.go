package filtering

// Compile parses filter and checks it against schema in one step. It is
// the entry point for the common service path:
//
//	checked, err := filtering.Compile(req.GetFilter(), bookSchema)
//	if errors.Is(err, filtering.ErrInvalidFilter) {
//		return nil, status.Error(codes.InvalidArgument, err.Error())
//	}
//
// Use [Parse] and [Check] separately when the syntactic tree is of
// interest.
func Compile(filter string, schema *Schema) (*Checked, error) {
	parsed, err := Parse(filter)
	if err != nil {
		return nil, err
	}

	return Check(parsed, schema)
}
