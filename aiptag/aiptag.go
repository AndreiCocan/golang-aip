package aiptag

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Key is the key of the struct tag that [Parse] reads.
const Key = "aip"

// ErrInvalidTag matches every error of [Parse] with [errors.Is].
var ErrInvalidTag = errors.New("invalid aip tag")

// Tag is the content of the aip tag of one struct field.
type Tag struct {
	// Name is the API name of the field, such as "display_name".
	Name string
	// Filterable tells that a filter can name the field.
	Filterable bool
	// Orderable tells that an order_by can name the field.
	Orderable bool
	// Searchable tells that a bare search term matches the field.
	Searchable bool
}

// Parse reads the aip tag of f. It reports false when f has no aip tag.
//
// It returns an error that matches [ErrInvalidTag] when the name is empty,
// or when an option is unknown or occurs twice.
func Parse(f reflect.StructField) (Tag, bool, error) {
	value, ok := f.Tag.Lookup(Key)
	if !ok {
		return Tag{}, false, nil
	}

	name, options, hasOptions := strings.Cut(value, ",")
	if name == "" {
		return Tag{}, true, fmt.Errorf("%w: field %s: empty name", ErrInvalidTag, f.Name)
	}

	tag := Tag{Name: name}
	if !hasOptions {
		return tag, true, nil
	}

	seen := make(map[string]bool)

	for option := range strings.SplitSeq(options, ",") {
		if seen[option] {
			return Tag{}, true, fmt.Errorf("%w: field %s: option %q occurs twice",
				ErrInvalidTag, f.Name, option)
		}

		seen[option] = true

		switch option {
		case "filter":
			tag.Filterable = true
		case "order":
			tag.Orderable = true
		case "search":
			tag.Searchable = true
		default:
			return Tag{}, true, fmt.Errorf("%w: field %s: unknown option %q",
				ErrInvalidTag, f.Name, option)
		}
	}

	return tag, true, nil
}
