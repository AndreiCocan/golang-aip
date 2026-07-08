// Package ast declares the syntactic tree for order_by expressions.
//
// An [OrderBy] is a comma-separated list of fields, each a dotted path
// with an optional "desc" suffix. The tree is purely syntactic: no schema
// validation has happened, and duplicate fields are preserved as written.
// All positions are byte offsets into the original order_by string.
package ast
