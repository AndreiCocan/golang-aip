package resourcename

import "strings"

// Sprint formats pattern into a concrete resource name by substituting
// successive variables for its "{variable}" segments, in order.
//
// Sprint is lenient: extra variables are ignored, and a variable segment
// with no corresponding value becomes an empty segment. Use [Sscan]'s
// counterpart contract (one value per variable segment) to produce a name
// that passes [Validate].
func Sprint(pattern string, variables ...string) string {
	var totalVarLen int
	for _, v := range variables {
		totalVarLen += len(v)
	}

	var result strings.Builder
	result.Grow(len(pattern) + totalVarLen)

	var sc Scanner
	sc.Init(pattern)

	var variable int

	first := true

	for sc.Scan() {
		if !first {
			result.WriteByte('/')
		}

		first = false

		segment := sc.Segment()
		if segment.IsVariable() {
			if variable < len(variables) {
				result.WriteString(variables[variable])
				variable++
			}

			continue
		}

		result.WriteString(string(segment))
	}

	return result.String()
}
