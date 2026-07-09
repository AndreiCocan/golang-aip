// Package fieldbehavior reads and enforces the google.api.field_behavior
// annotations of resource-oriented APIs: the REQUIRED, OUTPUT_ONLY,
// INPUT_ONLY, IMMUTABLE, and IDENTIFIER designations that protos attach to
// fields.
//
// [Get] and [Has] look behaviors up on a field descriptor. The remaining
// functions enforce the behaviors a service must uphold at runtime:
//
//   - [Clear] drops annotated fields from a message. Clearing OUTPUT_ONLY
//     (plus IDENTIFIER on create) from a request payload discards
//     server-managed fields without erroring, as required of services.
//
//   - [Copy] carries annotated fields from one message to another, for
//     example restoring OUTPUT_ONLY fields from the stored resource after
//     a full replacement.
//
//   - [ValidateRequired] checks a create request's required fields;
//     [ValidateRequiredWithMask] checks an update's, where a required
//     field may be omitted as long as the field mask does not cover it.
//
//     if err := fieldbehavior.ValidateRequired(req); err != nil { … }
//     fieldbehavior.Clear(req.GetBook(),
//     annotations.FieldBehavior_OUTPUT_ONLY,
//     annotations.FieldBehavior_IDENTIFIER,
//     )
//
// Validation errors match [ErrMissingRequired] with [errors.Is] and name
// every missing field; services should surface them as an INVALID_ARGUMENT
// response.
//
// A nil message is a programming error everywhere a descriptor is needed to
// do the work: [Copy], [ValidateRequired], and [ValidateRequiredWithMask]
// panic on one. Only [Clear] accepts it, since a message with no fields has
// nothing to clear.
//
// Field mask handling itself, including how IMMUTABLE is enforced during
// an update, lives in the fieldmask package; this package is the annotation
// layer under it.
package fieldbehavior
