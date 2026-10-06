// Package aiptag reads the aip struct tag, which gives the API name of a
// field of a domain type and the List features that the field supports.
//
// A tag is the API name of the field, followed by options separated by
// commas, such as `aip:"title,filter,order"`. The options are:
//
//   - filter: a filter can name the field.
//   - order: an order_by can name the field.
//   - search: a bare search term of a filter matches the field.
//
// The tag does not hold a resource name pattern.
//
// [Parse] reads the tag of one struct field. The filtering and ordering
// packages of this module, and backend packages in other modules, use it so
// that all of them read the same tag the same way.
package aiptag
