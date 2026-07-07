// Package filtering parses, validates, and resolves AIP-160 filter
// expressions, the `string filter` field of List requests in
// resource-oriented APIs.
//
// The package is the backend-neutral base of a two-layer design:
//
//   - This package turns a filter string into a checked, fully typed
//     expression tree.
//   - Separate dialect packages, such as [filtering/sqlite], translate
//     that tree into a storage backend's query language. The checked tree
//     is the contract between the two layers; anyone can implement a
//     dialect for another backend against it.
//
// The pipeline is [Parse] (string to syntactic tree), then [Check]
// (syntactic tree to typed tree, validated against a [Schema]); [Compile]
// runs both. The schema declares the filterable fields with their types
// and doubles as the allowlist: filters referencing undeclared fields fail
// Check.
//
//	schema := filtering.NewSchema(
//		filtering.String("display_name"),
//		filtering.Timestamp("create_time"),
//		filtering.Enum("state", "ACTIVE", "DELETED"),
//	)
//	checked, err := filtering.Compile(req.GetFilter(), schema)
//
// A [Checked] filter contains only five node kinds ([And], [Or], [Not],
// [Comparison], and [Search]) with every literal resolved to a typed
// [Value] and every field path resolved to a [Field] whose segments carry
// their types. [Walk] traverses the tree.
//
// All errors for malformed or invalid filters match [ErrInvalidFilter]
// with [errors.Is] and carry the byte offset of the problem; services
// should surface them as an INVALID_ARGUMENT response.
//
// # Supported syntax
//
// The full official filter grammar is parsed: comparators (=, !=, <, <=,
// >, >=), the has operator (:) for repeated fields, maps, messages, and
// presence tests, AND / OR / NOT (and the - negation prefix), parentheses,
// dotted field traversal, functions, and bare search terms. Note that OR
// binds tighter than AND, the opposite of most programming languages.
//
// Literals are typed by the field they are compared against: RFC 3339
// timestamps (quote them; the time-of-day colons would otherwise split
// the token), seconds durations like 20s or 1.5s, integers, floats with
// exponents, true/false, case-sensitive enum names, and null for
// message-backed fields. A * inside a string compared with = or != is a
// wildcard; the grammar defines no escape for a literal asterisk.
//
// Bare terms with no field and no comparator (`Hugo`, `New York`) are
// valid filters that check into [Search] nodes. How, and whether, they
// match is a dialect decision: backends without a text-search capability
// reject them.
//
// # Functions
//
// Filter functions must be declared in the schema with [Function]; calls
// to undeclared functions fail Check. A function declared with [Expand] is
// a macro: at Check time the call is rewritten into an ordinary expression
// tree, so it works on every dialect. A function without an expander
// passes through as a [FuncCall] node for the dialect to translate
// natively, or reject.
//
// # Writing a dialect
//
// A dialect package consumes a [Checked] filter and produces whatever its
// backend needs: a SQL WHERE clause, a search query, a plan. There is no
// interface to implement: expose whatever entry point suits the backend
// and type-switch over the five node kinds. Reject what the backend cannot
// express (commonly [Search], [FuncCall], and has restrictions on
// repeated fields) with clear errors rather than approximating. See
// filtering/sqlite for a reference implementation.
package filtering
