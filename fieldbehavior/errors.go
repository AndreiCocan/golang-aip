package fieldbehavior

import "errors"

// ErrMissingRequired is the sentinel matched by every error returned for a
// field annotated REQUIRED that carries no value. Services should map
// errors matching this sentinel to an INVALID_ARGUMENT response:
//
//	if errors.Is(err, fieldbehavior.ErrMissingRequired) {
//		return nil, status.Error(codes.InvalidArgument, err.Error())
//	}
var ErrMissingRequired = errors.New("missing required field")
