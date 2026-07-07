// Package resourceid validates and generates the trailing ID segment of
// resource names in resource-oriented APIs.
//
// Use [ValidateUserSettable] to check an ID supplied by an end user, such as
// the "{resource}_id" field on a Create request. Use [NewSystemGenerated] to
// mint an ID when the user left that field unset. Validation failures match
// [ErrInvalid].
package resourceid
