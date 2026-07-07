package filtering

// Walk traverses a checked expression in depth-first pre-order, calling f
// for every node. If f returns false the node's children are skipped.
// [*Comparison] and [*Search] nodes are leaves; function-call arguments are
// not traversed.
//
// Walk is a convenience for dialects and tooling that scan a filter, for
// example to collect referenced fields or detect unsupported constructs. A
// nil expr (an empty filter) is allowed and never calls f.
func Walk(expr Expr, f func(Expr) bool) {
	if expr == nil || !f(expr) {
		return
	}

	switch x := expr.(type) {
	case *And:
		for _, op := range x.Operands {
			Walk(op, f)
		}
	case *Or:
		for _, op := range x.Operands {
			Walk(op, f)
		}
	case *Not:
		Walk(x.Operand, f)
	}
}
