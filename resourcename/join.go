package resourcename

import "strings"

// Join joins resource name fragments into one resource name. It removes
// empty segments and extra slashes. When the first fragment is a full name
// ("//service/..."), the result keeps its service name. Join removes the
// service name of a later fragment. When there are no non-empty segments,
// Join returns "/".
func Join(elems ...string) string {
	segments := make([]string, 0, len(elems))

	for elemIndex, elem := range elems {
		var sc Scanner
		sc.Init(elem)

		// The "//service" prefix is only known after the first Scan, which
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
