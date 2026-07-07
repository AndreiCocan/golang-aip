package resourcename

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// Sscan parses name according to pattern, storing the value of each
// "{variable}" segment into successive variables. Literal pattern segments
// must be equal to the corresponding name segment. A full name
// ("//host/...") is parsed by its path only; the service host is ignored.
//
// The number of variables must equal the number of variable segments in
// pattern, and every pointer must be non-nil.
//
// An error is returned when pattern is a full resource name, a variable
// pointer is nil, a literal segment differs, name has fewer or more
// segments than pattern, or the variable counts do not line up. On error,
// some variables may already have been assigned.
func Sscan(name, pattern string, variables ...*string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("parse resource name %q with pattern %q: %w", name, pattern, err)
		}
	}()

	if strings.HasPrefix(pattern, "//") {
		return errors.New("pattern must not be a full resource name")
	}

	for i, v := range variables {
		if v == nil {
			return fmt.Errorf("variable %d: nil pointer", i)
		}
	}

	var nameScanner, patternScanner Scanner
	nameScanner.Init(name)
	patternScanner.Init(pattern)

	var i int

	for patternScanner.Scan() {
		if !nameScanner.Scan() {
			return fmt.Errorf("segment %s: %w", patternScanner.Segment(), io.ErrUnexpectedEOF)
		}

		nameSegment, patternSegment := nameScanner.Segment(), patternScanner.Segment()
		if !patternSegment.IsVariable() {
			// Compare raw segments: a braced name segment ("{books}") must
			// not satisfy the literal "books".
			if patternSegment != nameSegment {
				return fmt.Errorf("segment %s: got %s", patternSegment, nameSegment)
			}

			continue
		}

		if i > len(variables)-1 {
			return fmt.Errorf("segment %s: too few variables", patternSegment)
		}

		*variables[i] = nameSegment.Literal()
		i++
	}

	if nameScanner.Scan() {
		return errors.New("got trailing segments in name")
	}

	if i != len(variables) {
		return fmt.Errorf("too many variables: got %d but expected %d", len(variables), i)
	}

	return nil
}
