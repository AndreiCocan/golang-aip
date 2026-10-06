// Package fieldmask validates and applies the field masks of
// resource-oriented APIs: the `google.protobuf.FieldMask update_mask` of
// Update requests and the read masks of partial responses.
//
// [Update] merges the masked fields of a request payload into the stored
// resource, [Prune] strips a response down to the masked fields, and
// [CheckUpdate] and [CheckRead] validate the paths of an update mask and of a read
// mask on their own, for failing fast before the resource is fetched. All
// of them resolve paths against the message's own proto descriptors, so
// there is no schema to declare.
//
//	if err := fieldmask.Check(req.GetUpdateMask(), &pb.Book{}); err != nil { … }
//	stored := fetch(…)
//	if err := fieldmask.Update(req.GetUpdateMask(), stored, req.GetBook()); err != nil { … }
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
// Updates honor the message's google.api.field_behavior annotations, via
// the fieldbehavior package: OUTPUT_ONLY fields silently keep their stored
// values no matter what the mask says, and IMMUTABLE or IDENTIFIER fields
// reject changes. See [Update] for the exact rules.
//
// [ErrInvalidFieldMask] and [ErrImmutable] report bad client input and
// match with [errors.Is]; services should surface both as an
// INVALID_ARGUMENT response.
package fieldmask
