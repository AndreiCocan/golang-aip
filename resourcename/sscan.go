package resourcename

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// Sscan reads name with pattern, and stores the value of each "{variable}"
// segment of pattern in the next pointer of variables. Each literal segment
// of pattern must be equal to the segment of name at the same position.
// Sscan ignores the service name of a full name ("//service/...").
//
// Give one non-nil pointer for each variable segment of pattern. Sscan
// returns an error when:
//
//   - pattern is a full resource name,
//   - a pointer of variables is nil,
//   - name has a variable segment,
//   - a literal segment of pattern is not equal to the segment of name,
//   - name has fewer or more segments than pattern, or
//   - the number of pointers is not the number of variable segments.
//
// The error does not match [ErrInvalidName]. When Sscan returns an error,
// it can have set some of the variables.
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
		if nameSegment.IsVariable() {
			return fmt.Errorf(
				"segment %s: name has a variable segment %s",
				patternSegment,
				nameSegment,
			)
		}

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

		*variables[i] = string(nameSegment)
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
