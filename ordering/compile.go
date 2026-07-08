package ordering

// Compile parses orderBy and checks it against schema in one step. It is
// the entry point for the common service path:
//
//	checked, err := ordering.Compile(req.GetOrderBy(), bookSchema)
//	if errors.Is(err, ordering.ErrInvalidOrderBy) {
//		return nil, status.Error(codes.InvalidArgument, err.Error())
//	}
//
// Use [Parse] and [Check] separately when the syntactic tree is of
// interest.
func Compile(orderBy string, schema *Schema) (*Checked, error) {
	parsed, err := Parse(orderBy)
	if err != nil {
		return nil, err
	}

	return Check(parsed, schema)
}
