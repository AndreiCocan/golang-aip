package resourcename

import "strings"

// Scanner walks a resource name from left to right, yielding one [Segment]
// per call to [Scanner.Scan]. It performs no validation: empty segments are
// surfaced as empty [Segment] values, and any bytes between slashes are
// returned as-is.
//
// A name starting with "//" is a full resource name: the text between "//"
// and the next "/" is the service host, reported by [Scanner.ServiceName]
// and [Scanner.Full] rather than as a path segment. A single leading "/" is
// skipped.
//
// The zero value is ready for [Scanner.Init]. A Scanner may be reused by
// calling Init again. Not safe for concurrent use.
type Scanner struct {
	name                     string
	start, end               int
	serviceStart, serviceEnd int
	full                     bool
}

// Init binds name to the scanner and rewinds it. Init must be called before
// the first [Scanner.Scan].
func (s *Scanner) Init(name string) {
	s.name = name
	s.start, s.end = 0, 0
	s.serviceStart, s.serviceEnd = 0, 0
	s.full = false
}

// Scan advances to the next segment, reporting whether one is available. On
// the first call it detects the "//host" prefix of a full name.
func (s *Scanner) Scan() bool {
	switch s.end {
	case len(s.name):
		return false
	case 0:
		switch {
		case strings.HasPrefix(s.name, "//"):
			s.full = true
			s.start = 2

			hostLen := strings.IndexByte(s.name[s.start:], '/')
			if hostLen == -1 {
				s.serviceStart, s.serviceEnd = s.start, len(s.name)
				s.start, s.end = len(s.name), len(s.name)

				return false
			}

			s.serviceStart, s.serviceEnd = s.start, s.start+hostLen
			s.start = s.serviceEnd + 1
		case strings.HasPrefix(s.name, "/"):
			s.start = 1
		}
	default:
		s.start = s.end + 1
	}

	if nextSlash := strings.IndexByte(s.name[s.start:], '/'); nextSlash == -1 {
		s.end = len(s.name)
	} else {
		s.end = s.start + nextSlash
	}

	return true
}

// Segment returns the current segment. It shares storage with the scanned
// name; no allocation is performed. Defined only after [Scanner.Scan]
// returned true.
func (s *Scanner) Segment() Segment {
	return Segment(s.name[s.start:s.end])
}

// Start returns the index in the scanned name of the first byte of the
// current segment. Defined only after [Scanner.Scan] returned true.
func (s *Scanner) Start() int {
	return s.start
}

// End returns the index just past the last byte of the current segment, so
// name[Start():End()] is the current segment. Defined only after
// [Scanner.Scan] returned true.
func (s *Scanner) End() int {
	return s.end
}

// Full reports whether the scanned name is a full resource name, that is,
// starts with "//" followed by a service host.
func (s *Scanner) Full() bool {
	return s.full
}

// ServiceName returns the service host of a full resource name, or the empty
// string for a relative name.
func (s *Scanner) ServiceName() string {
	return s.name[s.serviceStart:s.serviceEnd]
}
