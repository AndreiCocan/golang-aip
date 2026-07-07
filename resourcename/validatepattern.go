package resourcename

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidPattern matches every error returned by [ValidatePattern] via
// [errors.Is]. The wrapped message states the specific rule that was
// violated.
var ErrInvalidPattern = errors.New("invalid resource name pattern")

// ValidatePattern reports whether pattern is a well-formed resource name
// pattern such as "publishers/{publisher}/books/{book}".
//
// Rules:
//   - The pattern must be non-empty and relative; full "//host/..." forms
//     are rejected because patterns are written relative to a service.
//   - Every segment must be non-empty.
//   - The [Wildcard] "-" must not be hard-coded in a pattern; use a
//     "{variable}" where a value is supplied at that position.
//   - Variables must be snake_case ("[a-z][_a-z0-9]*[a-z0-9]"), must not end
//     in "_id" (the variable for a book is "{book}", not "{book_id}"), and
//     must be unique within the pattern.
//   - Literal segments name collections or singletons and must be lower
//     camel case alphanumeric ("[a-z][a-zA-Z0-9]*").
//
// The returned error matches [ErrInvalidPattern] and carries a
// human-readable reason.
func ValidatePattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("%w: empty", ErrInvalidPattern)
	}

	var sc Scanner
	sc.Init(pattern)

	seenVariables := map[string]bool{}

	var i int
	for sc.Scan() {
		i++
		segment := sc.Segment()

		switch {
		case segment == "":
			return fmt.Errorf("%w: segment %d is empty", ErrInvalidPattern, i)
		case segment.IsWildcard():
			return fmt.Errorf(
				"%w: segment %d: wildcards are not allowed in patterns",
				ErrInvalidPattern,
				i,
			)
		case segment.IsVariable():
			name := segment.Literal()
			if err := validatePatternVariable(name); err != nil {
				return fmt.Errorf("%w: segment %q: %w", ErrInvalidPattern, segment, err)
			}

			if seenVariables[name] {
				return fmt.Errorf("%w: segment %q: duplicate variable", ErrInvalidPattern, segment)
			}

			seenVariables[name] = true
		case !isCollectionIdentifier(string(segment)):
			return fmt.Errorf(
				"%w: segment %q: literals must be lower camel case alphanumeric",
				ErrInvalidPattern,
				segment,
			)
		}
	}

	if sc.Full() {
		return fmt.Errorf("%w: patterns must not be full resource names", ErrInvalidPattern)
	}

	return nil
}

// minVariableLength is the shortest legal pattern variable name: the format
// "[a-z][_a-z0-9]*[a-z0-9]" requires a first and a last character.
const minVariableLength = 2

// validatePatternVariable checks the name between the braces of a pattern
// variable: snake_case per "[a-z][_a-z0-9]*[a-z0-9]" and no "_id" suffix.
func validatePatternVariable(name string) error {
	if len(name) < minVariableLength {
		return errors.New("variable must be at least two characters")
	}

	if name[0] < 'a' || name[0] > 'z' {
		return errors.New("variable must begin with a lowercase letter")
	}

	last := name[len(name)-1]
	if !isLowerAlphanumeric(last) {
		return errors.New("variable must end with a lowercase letter or a digit")
	}

	for _, character := range []byte(name) {
		if character != '_' && !isLowerAlphanumeric(character) {
			return errors.New("variable must be snake_case")
		}
	}

	if strings.HasSuffix(name, "_id") {
		return errors.New("variable must not use an _id suffix")
	}

	return nil
}

// isCollectionIdentifier reports whether s is a valid literal pattern
// segment: lower camel case alphanumeric ("[a-z][a-zA-Z0-9]*").
func isCollectionIdentifier(s string) bool {
	if len(s) == 0 || s[0] < 'a' || s[0] > 'z' {
		return false
	}

	for _, character := range []byte(s) {
		switch {
		case 'a' <= character && character <= 'z':
		case 'A' <= character && character <= 'Z':
		case '0' <= character && character <= '9':
		default:
			return false
		}
	}

	return true
}

// isLowerAlphanumeric reports whether c is a lowercase ASCII letter or digit.
func isLowerAlphanumeric(c byte) bool {
	return 'a' <= c && c <= 'z' || '0' <= c && c <= '9'
}
