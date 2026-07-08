package pagination

import "fmt"

// PageSize resolves a request's page_size field against the service's
// default and maximum. A zero requested size means the client left the
// field unset and yields defaultSize; a size above maxSize is capped to
// maxSize; a maxSize of zero disables the cap. A negative requested size
// is a client error matching [ErrInvalidPageSize], which services should
// surface as an INVALID_ARGUMENT response.
//
// defaultSize must be positive and maxSize non-negative; a configuration
// violating either also fails with [ErrInvalidPageSize].
func PageSize(requested, defaultSize, maxSize int32) (int32, error) {
	switch {
	case defaultSize <= 0:
		return 0, fmt.Errorf("%w: default size %d is not positive", ErrInvalidPageSize, defaultSize)
	case maxSize < 0:
		return 0, fmt.Errorf("%w: max size %d is negative", ErrInvalidPageSize, maxSize)
	case requested < 0:
		return 0, fmt.Errorf("%w: page size %d is negative", ErrInvalidPageSize, requested)
	}

	size := requested
	if size == 0 {
		size = defaultSize
	}

	if maxSize > 0 && size > maxSize {
		size = maxSize
	}

	return size, nil
}
