package resourcename

// Wildcard is the segment "-", standing for an unspecified resource ID when
// reading across collections (e.g. "publishers/-/books/123").
const Wildcard = "-"

// Segment is one path component of a resource name or pattern, between "/"
// characters. It is one of three kinds:
//
//   - a literal such as "publishers" or "les-miserables",
//   - the [Wildcard] "-", or
//   - (patterns only) a braced variable such as "{publisher}".
type Segment string

// IsVariable reports whether the segment is a pattern variable of the form
// "{name}" with a non-empty name. It is false for the [Wildcard] and for
// literals.
func (s Segment) IsVariable() bool {
	return len(s) > 2 && s[0] == '{' && s[len(s)-1] == '}'
}

// IsWildcard reports whether the segment is exactly the [Wildcard] "-".
func (s Segment) IsWildcard() bool {
	return s == Wildcard
}

// VariableName returns the name of a pattern variable without its braces:
// "publisher" for the segment "{publisher}". It returns "" when the segment
// is not a variable, see [Segment.IsVariable].
func (s Segment) VariableName() string {
	if !s.IsVariable() {
		return ""
	}

	return string(s[1 : len(s)-1])
}
