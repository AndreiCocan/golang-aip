package resourcename

import "strings"

// Sprint makes a resource name from pattern: it replaces each "{variable}"
// segment with the next value of variables, in order.
//
// Sprint does not validate. It ignores extra values, and it makes an empty
// segment for a variable that has no value. To make a name that [Validate]
// accepts, give one valid value for each variable segment.
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
