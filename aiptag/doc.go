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
// [Parse] reads the tag of one struct field into a [Tag]. Use it in each
// package that reads the tag, so that all of them read it the same way.
package aiptag
