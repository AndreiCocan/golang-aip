package resourcename

// ContainsWildcard reports whether a segment of name is the [Wildcard]
// "-". A service that does not support reads across collections uses it to
// reject wildcard names. ContainsWildcard ignores the service name of a full
// name.
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
