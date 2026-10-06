package resourcename

import (
	"errors"
	"fmt"
)

// ErrInvalidName matches each error of [Validate] with [errors.Is]. The
// message of the error gives the rule that the name does not obey.
var ErrInvalidName = errors.New("invalid resource name")

// Validate returns an error when name is not a valid resource name. It
// accepts relative names ("publishers/123") and full names
// ("//library.example.com/publishers/123"). It rejects patterns: for a
// pattern, use [ValidatePattern].
//
// The rules are:
//
//   - The name and each of its segments are not empty.
//   - Each segment is a valid RFC 1123 host name: labels of letters,
//     digits, and hyphens with dots between them. A label does not start or
//     end with a hyphen, and has 63 characters or fewer.
//   - The [Wildcard] "-" is a valid segment. It stands for any ID in a
//     read across collections.
//   - A variable segment, such as "{book}", is not valid.
//   - The service name of a full name is a valid DNS name.
//
// The error matches [ErrInvalidName]. Its message is for humans, and it is
// correct to send it to the client.
func Validate(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty", ErrInvalidName)
	}

	var sc Scanner
	sc.Init(name)

	var i int
	for sc.Scan() {
		i++
		segment := sc.Segment()

		switch {
		case segment == "":
			return fmt.Errorf("%w: segment %d is empty", ErrInvalidName, i)
		case segment.IsWildcard():
			continue
		case segment.IsVariable():
			return fmt.Errorf(
				"%w: segment %q: concrete names must not contain variables",
				ErrInvalidName,
				segment,
			)
		case !isRFC1123Name(string(segment)):
			return fmt.Errorf("%w: segment %q: not a valid identifier", ErrInvalidName, segment)
		}
	}

	if sc.Full() && !isRFC1123Name(sc.ServiceName()) {
		return fmt.Errorf("%w: service %q: not a valid DNS name", ErrInvalidName, sc.ServiceName())
	}

	return nil
}
