package resourcename

import "strings"

// Join combines resource name fragments into a single resource name,
// dropping empty segments and redundant slashes. If the first element is a
// full name ("//host/..."), the service host is kept; a host on any later
// element is dropped. Joining no non-empty segments returns "/".
func Join(elems ...string) string {
	segments := make([]string, 0, len(elems))

	for elemIndex, elem := range elems {
		var sc Scanner
		sc.Init(elem)

		// The "//host" prefix is only known after the first Scan, which
		// returns false for a host-only fragment; capture the host before
		// deciding whether any path segments follow.
		more := sc.Scan()
		if elemIndex == 0 && sc.Full() {
			segments = append(segments, "//"+sc.ServiceName())
		}

		for ; more; more = sc.Scan() {
			segment := sc.Segment()
			if segment == "" {
				continue
			}

			segments = append(segments, string(segment))
		}
	}

	if len(segments) == 0 {
		return "/"
	}

	return strings.Join(segments, "/")
}
