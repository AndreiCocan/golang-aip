// Package filtering parses and validates AIP-160 filter expressions, the
// `string filter` field of List requests in resource-oriented APIs.
//
// The package does not query storage. It turns a filter string into a
// checked tree in which each field path and each literal has a type. A
// dialect package then translates the checked tree into the query language
// of one storage backend.
//
// [Parse] turns a string into a syntax tree. [Check] turns a syntax tree
// into a [CheckedFilter], validated against a [Schema]. [Compile] does both.
// The schema declares the fields that a filter can name, with their types,
// and the functions that a filter can call. It is also the allowlist: Check
// rejects a filter that names a field that the schema does not declare.
// [NewSchema] builds a schema from declarations such as [StringField] and
// [Func]. [SchemaFromTags] builds it from the aip struct tags of a domain
// type.
//
// A checked filter has only five node kinds: [And], [Or], [Not],
// [Comparison], and [Search]. Each literal is a typed [Value], and each
// field path is a [Field] whose segments have their types. [Walk] goes
// through the tree.
//
// Each error for a malformed or invalid filter matches [ErrInvalidFilter]
// with [errors.Is], and holds the byte offset of the problem. Such an
// error is bad client input.
//
// # Supported syntax
//
// Parse accepts the full filter grammar of AIP-160:
//
//   - the comparators =, !=, <, <=, >, and >=,
//   - the has operator (:) for repeated fields, maps, messages, and
//     presence tests,
//   - AND, OR, NOT, and the - prefix for negation,
//   - parentheses, dotted field paths, function calls, and bare search
//     terms.
//
// OR binds tighter than AND. This is the opposite of most programming
// languages.
//
// The type of the field gives the type of the literal that it is compared
// with:
//
//   - a timestamp is an RFC 3339 string. Quote it, because the colons of
//     the time would split the token.
//   - a duration is a number of seconds, such as 20s or 1.5s.
//   - an enum value is a case-sensitive name.
//   - null tests a message, timestamp, or duration field for no value.
//
// A * inside a string compared with = or != is a wildcard. The grammar has
// no escape for a literal asterisk.
//
// A bare term with no field and no comparator, such as `Hugo` or
// `New York`, is a valid filter. Check turns it into a [Search] node. The
// dialect decides how such terms match.
//
// # Functions
//
// Declare each filter function in the schema with [Func]. Check rejects a
// call to a function that the schema does not declare. A function with a
// [FuncExpand] option is a macro: Check replaces the call with an ordinary
// expression tree, so all dialects support it. Check keeps a call to a
// function without an expander as a [FuncCall] node. The dialect then
// translates it, or rejects it.
//
// # Writing a dialect
//
// A dialect reads a [CheckedFilter] and makes what its backend needs, such
// as a WHERE clause. There is no interface to implement: type-switch over
// the five node kinds. Reject what the backend cannot express, such as
// [Search], [FuncCall], or a has restriction on a repeated field, with a
// clear error. Do not approximate it.
package filtering
