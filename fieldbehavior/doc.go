// Package fieldbehavior reads and enforces the google.api.field_behavior
// annotations of resource-oriented APIs: the REQUIRED, OUTPUT_ONLY,
// INPUT_ONLY, IMMUTABLE, and IDENTIFIER designations that protos attach to
// fields.
//
// [Get] and [Has] read the behaviors of a field descriptor. The other
// functions apply the behaviors at run time:
//
//   - [Clear] removes the fields with given behaviors from a message. For
//     example, clear OUTPUT_ONLY, and IDENTIFIER on create, from a request
//     payload. Then the fields that the server manages go away without an
//     error.
//   - [Copy] copies the fields with given behaviors from one message to
//     another. For example, put back the OUTPUT_ONLY fields of the stored
//     resource after a full replacement.
//   - [ValidateRequired] checks the required fields of a create request.
//     [ValidateRequiredWithMask] checks those of an update, where a
//     required field can be absent when the field mask does not cover it.
//
// A validation error matches [ErrMissingRequired] with [errors.Is], and
// names each missing field. Such an error is bad client input. The error
// is a [*RequiredFieldsError] with the paths of all the missing fields.
//
// [Copy], [ValidateRequired], and [ValidateRequiredWithMask] need a
// descriptor, so they panic on a nil message. Only [Clear] accepts a nil
// message, because it has no fields to clear.
//
// The fieldmask package uses this package to enforce the annotations
// during an update.
package fieldbehavior
