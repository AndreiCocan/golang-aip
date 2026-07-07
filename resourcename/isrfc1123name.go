package resourcename

// isRFC1123Name reports whether s is a valid host name per RFC 1123
// section 2.1: dot-separated labels of 1 to 63 characters, each made of
// ASCII letters, digits, and hyphens, neither starting nor ending with a
// hyphen. Unlike RFC 952, a label may start with a digit. The total length
// must not exceed 253 characters, or 254 with a trailing dot (the
// presentation form of the terminal empty label), so that the wire format
// stays within its 255-octet limit.
//
// [Validate] applies it to each path segment and to the service host of a
// full resource name. Segments keep DNS case-insensitivity: uppercase
// letters are accepted, as collection identifiers are lowerCamelCase per
// AIP-122.
func isRFC1123Name(s string) bool {
	l := len(s)
	if l == 0 || l > 254 || l == 254 && s[l-1] != '.' {
		return false
	}

	if s[l-1] == '.' {
		s = s[:l-1]
	}

	for {
		label := s

		dot := -1

		for i := range len(s) {
			if s[i] == '.' {
				label, dot = s[:i], i

				break
			}
		}

		if !isRFC1123Label(label) {
			return false
		}

		if dot == -1 {
			return true
		}

		s = s[dot+1:]
	}
}

// isRFC1123Label reports whether label is a valid RFC 1123 host name label:
// 1 to 63 ASCII letters, digits, and hyphens, with no hyphen at either end.
func isRFC1123Label(label string) bool {
	if len(label) == 0 || len(label) > 63 {
		return false
	}

	if label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}

	for i := range len(label) {
		c := label[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9', c == '-':
		default:
			return false
		}
	}

	return true
}
