// Package resourceid validates and generates the trailing ID segment of
// resource names in resource-oriented APIs.
//
// Use [ValidateUserSettable] to check an ID supplied by an end user, such as
// the "{resource}_id" field on a Create request. Use [Generate] to make
// an ID when that field has no value. Validation errors match
// [ErrInvalidID].
package resourceid
