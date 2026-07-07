// Package ast declares the syntactic tree for AIP-160 filter expressions.
//
// The node types mirror the official filter grammar: a [Filter] holds an
// optional [Expression], which is a conjunction (AND) of sequences; a
// [Sequence] is one or more whitespace-separated factors; a [Factor] is a
// disjunction (OR) of terms; and a [Term] optionally negates a [Restriction]
// or a parenthesized [Composite]. Because OR joins terms while AND joins
// sequences, OR binds tighter than AND, the opposite of most programming
// languages.
//
// The tree is purely syntactic: literals are untyped text, and no schema
// validation has happened. All positions are byte offsets into the original
// filter string.
package ast
