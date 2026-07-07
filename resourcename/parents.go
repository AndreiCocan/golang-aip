package resourcename

import "iter"

// HasParent reports whether name is strictly a descendant of parent: parent
// must align segment-by-segment with a proper prefix of name. A name is
// never its own parent. A [Wildcard] "-" segment in parent matches any
// segment of name at that position.
//
// The two names must agree on form: two relative names compare by path
// only; two full names must share the same service host; a mix of one full
// and one relative name is never a parent relationship.
func HasParent(name, parent string) bool {
	if name == "" || parent == "" || name == parent {
		return false
	}

	var parentScanner, nameScanner Scanner
	parentScanner.Init(parent)
	nameScanner.Init(name)

	for parentScanner.Scan() {
		if !nameScanner.Scan() {
			return false
		}

		if parentScanner.Segment().IsWildcard() {
			continue
		}

		if parentScanner.Segment() != nameScanner.Segment() {
			return false
		}
	}

	// A parent is a strict ancestor: name needs at least one segment beyond
	// parent. String inequality alone does not guarantee that (wildcards
	// make distinct strings cover the same depth).
	if !nameScanner.Scan() {
		return false
	}

	if parentScanner.Full() != nameScanner.Full() {
		return false
	}

	if parentScanner.Full() {
		return parentScanner.ServiceName() == nameScanner.ServiceName()
	}

	return true
}

// Ancestor returns the prefix of name that pattern covers when pattern
// aligns with the start of name: literal pattern segments must be equal and
// each "{variable}" accepts whatever appears in name at that position. The
// returned string is a slice of name through the last segment that pattern
// consumed, so for a full name it includes the "//host" prefix.
//
// Returns "" and false when name or pattern is empty, when a literal
// segment disagrees, when pattern is deeper than name, or when pattern
// contains a [Wildcard] "-" (not supported in patterns).
func Ancestor(name, pattern string) (string, bool) {
	if name == "" || pattern == "" {
		return "", false
	}

	var nameScanner, patternScanner Scanner
	nameScanner.Init(name)
	patternScanner.Init(pattern)

	for patternScanner.Scan() {
		if !nameScanner.Scan() {
			return "", false
		}

		segment := patternScanner.Segment()

		switch {
		case segment.IsWildcard():
			return "", false
		case !segment.IsVariable() && segment != nameScanner.Segment():
			return "", false
		}
	}

	return name[:nameScanner.End()], true
}

// Parents returns an iterator over every intermediate parent path of name,
// from the shallowest ("publishers") to the deepest
// ("publishers/1/books"). The name itself is never yielded. For a full
// name, only path substrings after the service host are yielded.
//
// Each yielded value is a slice of name and shares its storage.
func Parents(name string) iter.Seq[string] {
	return func(yield func(string) bool) {
		var sc Scanner
		sc.Init(name)

		if !sc.Scan() {
			return
		}

		start := sc.Start()
		if sc.End() != len(name) && !yield(name[start:sc.End()]) {
			return
		}

		for sc.Scan() {
			if sc.End() != len(name) && !yield(name[start:sc.End()]) {
				return
			}
		}
	}
}
