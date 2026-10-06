package resourcename

import "iter"

// HasAncestor reports whether ancestor is a strict ancestor of name, at any
// depth: each segment of ancestor is equal to the segment of name at the
// same position, and name has more segments. A name is not its own
// ancestor. A [Wildcard] "-" segment in ancestor matches any segment of
// name.
//
// The two names must have the same form. Two relative names compare by
// their paths. Two full names must also have the same service name. A full
// name and a relative name are never ancestor and descendant.
func HasAncestor(name, ancestor string) bool {
	if name == "" || ancestor == "" || name == ancestor {
		return false
	}

	var parentScanner, nameScanner Scanner
	parentScanner.Init(ancestor)
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

// Ancestor returns the start of name that pattern matches. Each literal
// segment of pattern must be equal to the segment of name at the same
// position, and each "{variable}" segment matches any segment. The result is
// a substring of name up to the last segment that pattern matched. For a
// full name, it starts with the "//service" prefix.
//
// Ancestor returns "" and false when name or pattern is empty, when a
// literal segment is different, when pattern has more segments than name,
// or when pattern has a [Wildcard] "-" segment.
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

// Parents returns an iterator over the prefixes of name, from the shortest
// ("publishers") to the longest ("publishers/1/books"). It does not yield
// name. For a full name, the prefixes do not have the service name.
//
// Each yielded value is a substring of name, so it allocates no memory.
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
