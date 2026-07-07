package resourcename

// ContainsWildcard reports whether any path segment of name is the
// [Wildcard] "-". Services that do not support reading across collections
// use it to detect and reject wildcard names; the service host of a full
// name is not considered.
func ContainsWildcard(name string) bool {
	var sc Scanner
	sc.Init(name)

	for sc.Scan() {
		if sc.Segment().IsWildcard() {
			return true
		}
	}

	return false
}
