// Package fieldmask validates and applies the field masks of
// resource-oriented APIs: the `google.protobuf.FieldMask update_mask` of
// Update requests and the read masks of partial responses.
//
// The package has these entry points:
//
//   - [Update] merges the masked fields of a request payload into the
//     stored resource.
//   - [Prune] removes from a response the fields that are not in the mask.
//   - [MarshalJSON] writes the JSON of the fields that a read mask covers,
//     with the default values of the covered fields.
//   - [CheckUpdate] and [CheckRead] validate the paths of an update mask
//     and of a read mask. Call them before you read the resource, to fail
//     early.
//
// All of them resolve paths against the proto descriptors of the message,
// so there is no schema to declare.
//
// # Paths
//
// A mask path names a field of the resource, with dots traversing into
// nested messages: "author.name". Map entries are addressed by key,
// including backtick quoting for keys with problematic characters:
// "labels.lang", "labels.`k8s.io/name`", "reviews.smith.rating". A quoted
// key is a literal, so "labels.`*`" addresses the entry keyed "*". Repeated
// fields and maps are otherwise selected as a whole; paths cannot index
// into their elements. A read mask can go through a repeated message field
// to name a field of each element, such as "books.title" in a list
// response; an update mask cannot. The "*" path, alone, selects the entire
// resource: full replacement in an update, all fields in a read.
//
// Paths use the proto names of the fields. [Normalize] converts a mask
// that uses the JSON names, such as "createTime", which REST clients tend
// to write.
//
// An omitted mask means different defaults on the two sides: an update
// falls back to the implied mask of the payload's populated fields, while
// a read returns the full resource.
//
// # Field behavior
//
// [Update] obeys the google.api.field_behavior annotations of the message:
// an OUTPUT_ONLY field keeps its stored value, whatever the mask says, and an
// IMMUTABLE or IDENTIFIER field rejects a change. See [Update] for the
// rules.
//
// [ErrInvalidFieldMask] and [ErrImmutable] report bad client input. Test
// for them with [errors.Is].
package fieldmask
