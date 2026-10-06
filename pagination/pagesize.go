package pagination

import "fmt"

// ResolvePageSize applies the default and the maximum of the service to the
// page_size field of a request:
//
//   - A requested size of zero means that the client did not set the
//     field. It gives defaultSize.
//   - A requested size above maxSize gives maxSize. A maxSize of zero
//     means no maximum.
//   - A negative requested size is bad client input. It gives an error
//     that matches [ErrInvalidPageSize].
//
// defaultSize must be positive, and maxSize must not be negative. Another
// configuration also gives an error that matches [ErrInvalidPageSize].
func ResolvePageSize(requested, defaultSize, maxSize int32) (int32, error) {
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
