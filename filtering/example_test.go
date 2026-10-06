package filtering_test

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AndreiCocan/golang-aip/filtering"
)

// The usual service path. Build the schema once, from the aip tags of the
// domain type. Then compile the filter of each List request, and give the
// checked filter to a dialect, such as a PostgreSQL one, to query the
// storage.
func Example() {
	type Book struct {
		Name       string    `aip:"name,filter"`
		Title      string    `aip:"title,filter,search"`
		Pages      int64     `aip:"page_count,filter"`
		CreateTime time.Time `aip:"create_time,filter"`
	}

	schema := filtering.SchemaFromTags(Book{})

	checked, err := filtering.Compile(
		`title = "War*" AND create_time > "2021-01-01T00:00:00Z"`, schema,
	)
	if err != nil {
		fmt.Println(err) // See the invalid filter example.

		return
	}

	// Each literal has the type of its field: "War*" is a wildcard pattern
	// and the date is a timestamp.
	for _, operand := range checked.Expr.(*filtering.And).Operands {
		comparison := operand.(*filtering.Comparison)
		field := comparison.Left.(*filtering.Field)
		fmt.Println(field.Path(), comparison.Op, comparison.Right.Kind)
	}
	// Output:
	// title = pattern
	// create_time > timestamp
}

// A filter that is not valid gives an error that matches ErrInvalidFilter.
// The client sent it, so return INVALID_ARGUMENT with the message, which
// gives the position of the problem. Any other error is a bug in the
// service, such as an expander that fails.
func ExampleCompile_invalidFilter() {
	schema := filtering.NewSchema(filtering.IntField("page_count"))

	for _, filter := range []string{
		`page_count > "many"`, // a value of the wrong type
		`revision > 1`,        // a field that is not in the schema
		`page_count >`,        // a syntax error
	} {
		_, err := filtering.Compile(filter, schema)
		fmt.Println(errors.Is(err, filtering.ErrInvalidFilter), err)
	}
	// Output:
	// true invalid filter: expected an integer, got "many" at position 13
	// true invalid filter: unknown filter field "revision" at position 0
	// true invalid filter: expected value, got end of filter at position 12
}

// NewSchema declares the fields one by one, for fields that the aip tags
// cannot describe, such as enums, messages, and repeated messages. The
// table shows the checked form of each part of the filter syntax.
func ExampleNewSchema() {
	schema := filtering.NewSchema(
		filtering.StringField("title"),
		filtering.EnumField("state", "ACTIVE", "DELETED"),
		filtering.DurationField("read_time"),
		filtering.TimestampField("delete_time"),
		filtering.MessageField("author", filtering.StringField("name")),
		filtering.RepeatedField(filtering.StringField("tags")),
		filtering.RepeatedField(filtering.MessageField("chapters", filtering.StringField("title"))),
		filtering.MapField(filtering.StringField("labels")),
	)

	// describe writes a checked filter as text. A dialect walks the tree
	// in the same way to write its query language.
	var describe func(filtering.Expr) string

	describe = func(e filtering.Expr) string {
		var (
			name     string
			operands []filtering.Expr
		)

		switch e := e.(type) {
		case *filtering.Comparison:
			return fmt.Sprintf("%s %v %v", e.Left.(*filtering.Field).Path(), e.Op, e.Right.Kind)
		case *filtering.Search:
			return fmt.Sprintf("search %q", e.Terms)
		case *filtering.Not:
			return "NOT(" + describe(e.Operand) + ")"
		case *filtering.And:
			name, operands = "AND", e.Operands
		case *filtering.Or:
			name, operands = "OR", e.Operands
		}

		parts := make([]string, len(operands))
		for i, operand := range operands {
			parts[i] = describe(operand)
		}

		return name + "(" + strings.Join(parts, ", ") + ")"
	}

	for _, filter := range []string{
		`title = "*peace"`,                    // a * makes a wildcard pattern
		`state = ACTIVE AND read_time > 1.5s`, // an enum name, and seconds
		`delete_time = null`,                  // the field is not set
		`author.name = Hugo`,                  // a field of a message
		`author:*`,                            // the field is set
		`tags:classic`,                        // the list contains the value
		`chapters.title:Intro`,                // an element has the value
		`labels:env`,                          // the map has the key
		`labels.env = prod`,                   // the value of a key
		`tags:a OR tags:b AND -author:*`,      // OR binds tighter than AND
		`Hugo "New York"`,                     // bare words search the entry
	} {
		checked, err := filtering.Compile(filter, schema)
		if err != nil {
			fmt.Println(err)

			continue
		}

		fmt.Println(describe(checked.Expr))
	}
	// Output:
	// title = pattern
	// AND(state = enum, read_time > duration)
	// delete_time = null
	// author.name = string
	// author : *
	// tags : string
	// chapters.title : string
	// labels.env : *
	// labels.env = string
	// AND(OR(tags : string, tags : string), NOT(author : *))
	// search ["Hugo" "New York"]
}

// A function must be declared in the schema. A function with an expander
// is a macro: Compile replaces the call with a normal condition, so every
// dialect supports it. A function without an expander stays in the checked
// filter as a call, for the dialect to translate into a function of its
// database.
func ExampleFunc() {
	schema := filtering.NewSchema(
		filtering.StringField("title"),
		filtering.TimestampField("create_time"),

		// recent() becomes create_time > now - 24h.
		filtering.Func(
			"recent",
			filtering.FuncExpand(
				func(s *filtering.Schema, _ []filtering.Value) (filtering.Expr, error) {
					field, err := s.Field("create_time")
					if err != nil {
						return nil, err
					}

					now := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC) // time.Now() in a service

					return &filtering.Comparison{
						Left:  field,
						Op:    filtering.OpGreater,
						Right: filtering.TimestampValue(now.Add(-24 * time.Hour)),
					}, nil
				},
			),
		),

		// hasPrefix(field, text) stays a call.
		filtering.Func("hasPrefix",
			filtering.FuncArgs(filtering.KindString, filtering.KindString),
			filtering.FuncReturns(filtering.KindBool),
		),
	)

	checked, err := filtering.Compile(`recent() AND hasPrefix(title, "War")`, schema)
	if err != nil {
		fmt.Println(err)

		return
	}

	and := checked.Expr.(*filtering.And)

	macro := and.Operands[0].(*filtering.Comparison)
	fmt.Println(
		macro.Left.(*filtering.Field).Path(),
		macro.Op,
		macro.Right.Time.Format(time.RFC3339),
	)

	// A call by itself is a condition: the call = true.
	call := and.Operands[1].(*filtering.Comparison)
	fn := call.Left.(*filtering.FuncCall)
	fmt.Println(fn.Name, fn.Args[0].(*filtering.Field).Path(), fn.Args[1].(filtering.Value).Text)
	// Output:
	// create_time > 2020-12-31T00:00:00Z
	// hasPrefix title War
}

// Walk visits each node of a checked filter. Use it to inspect a filter
// before the query: for example, to join a table only when the filter
// names one of its fields, or to refuse a field to some users.
func ExampleWalk() {
	schema := filtering.NewSchema(
		filtering.BoolField("published"),
		filtering.MessageField("author", filtering.StringField("name")),
	)

	checked, err := filtering.Compile(
		`published = true AND (author.name = Hugo OR author.name = Zola)`, schema,
	)
	if err != nil {
		fmt.Println(err)

		return
	}

	joinAuthors := false

	filtering.Walk(checked.Expr, func(e filtering.Expr) bool {
		if c, ok := e.(*filtering.Comparison); ok {
			if field, ok := c.Left.(*filtering.Field); ok && field.Segments[0].Name == "author" {
				joinAuthors = true
			}
		}

		return true // Visit the children too.
	})

	fmt.Println(joinAuthors)
	// Output:
	// true
}

// Compile is Parse then Check. Call them separately to parse a filter once
// and check it against more than one schema, such as two versions of an
// API.
func ExampleCheck() {
	parsed, err := filtering.Parse(`rating > 4`)
	if err != nil {
		fmt.Println(err)

		return
	}

	v1 := filtering.NewSchema(filtering.StringField("title"))
	v2 := filtering.NewSchema(filtering.StringField("title"), filtering.FloatField("rating"))

	if _, err := filtering.Check(parsed, v1); err != nil {
		fmt.Println("v1:", err)
	}

	if checked, err := filtering.Check(parsed, v2); err == nil {
		fmt.Println("v2:", checked.Expr.(*filtering.Comparison).Right.Float)
	}
	// Output:
	// v1: invalid filter: unknown filter field "rating" at position 0
	// v2: 4
}
