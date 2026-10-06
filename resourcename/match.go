package resourcename

// Match reports whether name matches pattern, segment by segment. Each
// literal segment of pattern must be equal to the segment of name at the
// same position. Each "{variable}" segment of pattern matches one non-empty
// segment of name, also the [Wildcard] "-".
//
// Match ignores the service name of a full name ("//service/..."). Match
// returns false when the numbers of segments are different, when pattern is
// empty or a full name, when pattern has a wildcard segment, or when name
// has a variable segment.
//
// Match does not validate name or pattern. Use [Validate] and
// [ValidatePattern] for that.
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
