package resourcename

// Match reports whether name matches pattern segment-for-segment: literal
// segments must be equal, and each "{variable}" in pattern accepts any
// single non-empty segment of name, including the [Wildcard] "-".
//
// A full name ("//host/...") is matched by its path only; the service host
// is ignored. The match fails when the segment counts differ, when pattern
// is empty or a full name, when pattern hard-codes a wildcard, or when name
// itself contains variable segments.
//
// Match reports well-formedness of neither argument; validate separately
// with [Validate] and [ValidatePattern] where needed.
func Match(name, pattern string) bool {
	var nameScanner, patternScanner Scanner
	nameScanner.Init(name)
	patternScanner.Init(pattern)

	for patternScanner.Scan() {
		if !nameScanner.Scan() {
			return false
		}

		nameSegment, patternSegment := nameScanner.Segment(), patternScanner.Segment()

		switch {
		case nameSegment.IsVariable():
			return false
		case patternSegment.IsWildcard():
			return false
		case patternSegment.IsVariable():
			if nameSegment == "" {
				return false
			}
		case nameSegment != patternSegment:
			return false
		}
	}

	if nameScanner.Scan() || patternScanner.Segment() == "" || patternScanner.Full() {
		return false
	}

	return true
}
