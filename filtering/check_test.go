package filtering_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/AndreiCocan/golang-aip/filtering"
)

// checkTime is the fixed timestamp the recent() test macro expands to.
var checkTime = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

// checkSchema declares the book-shaped resource all Check tests run
// against.
var checkSchema = filtering.NewSchema(
	filtering.StringField("display_name"),
	filtering.IntField("page_count"),
	filtering.FloatField("rating"),
	filtering.BoolField("published"),
	filtering.TimestampField("create_time"),
	filtering.DurationField("read_time"),
	filtering.EnumField("state", "DRAFT", "ACTIVE", "DELETED"),
	filtering.MessageField("author",
		filtering.StringField("name"),
		filtering.BoolField("verified"),
	),
	filtering.RepeatedField(filtering.StringField("tags")),
	filtering.RepeatedField(filtering.MessageField("chapters",
		filtering.StringField("title"),
		filtering.IntField("pages"),
	)),
	filtering.MapField(filtering.StringField("labels")),
	filtering.Func(
		"recent",
		filtering.FuncExpand(
			func(s *filtering.Schema, _ []filtering.Value) (filtering.Expr, error) {
				f, err := s.Field("create_time")
				if err != nil {
					return nil, err
				}

				return &filtering.Comparison{
					Left:  f,
					Op:    filtering.OpGreater,
					Right: filtering.TimestampValue(checkTime),
				}, nil
			},
		),
	),
	filtering.Func(
		"titled",
		filtering.FuncArgs(filtering.KindString),
		filtering.FuncExpand(
			func(s *filtering.Schema, args []filtering.Value) (filtering.Expr, error) {
				f, err := s.Field("display_name")
				if err != nil {
					return nil, err
				}

				return &filtering.Comparison{Left: f, Op: filtering.OpEquals, Right: args[0]}, nil
			},
		),
	),
	filtering.Func("hasPrefix",
		filtering.FuncArgs(filtering.KindString, filtering.KindString),
		filtering.FuncReturns(filtering.KindBool),
	),
	filtering.Func("title", filtering.FuncReturns(filtering.KindString)),
)

// Expected-tree helpers.

func typ(kind filtering.Kind) filtering.Type { return filtering.Type{Kind: kind} }

func elemTyp(kind, elem filtering.Kind) filtering.Type {
	e := typ(elem)

	return filtering.Type{Kind: kind, Elem: &e}
}

func enumTyp(values ...string) filtering.Type {
	return filtering.Type{Kind: filtering.KindEnum, Enum: values}
}

func seg(name string, t filtering.Type) filtering.FieldSegment {
	return filtering.FieldSegment{Name: name, Type: t}
}

func fld(segments ...filtering.FieldSegment) *filtering.Field {
	return &filtering.Field{Segments: segments}
}

func cmpr(
	left filtering.Operand,
	op filtering.Operator,
	right filtering.Value,
) *filtering.Comparison {
	return &filtering.Comparison{Left: left, Op: op, Right: right}
}

func pattern(parts ...filtering.PatternPart) filtering.Value {
	return filtering.Value{Kind: filtering.KindPattern, Pattern: parts}
}

func lit(s string) filtering.PatternPart { return filtering.PatternPart{Literal: s} }
func anyPart() filtering.PatternPart     { return filtering.PatternPart{Wildcard: true} }
func search(terms ...string) *filtering.Search {
	return &filtering.Search{Terms: terms}
}

var (
	stateType    = enumTyp("DRAFT", "ACTIVE", "DELETED")
	displayName  = fld(seg("display_name", typ(filtering.KindString)))
	pageCount    = fld(seg("page_count", typ(filtering.KindInt)))
	rating       = fld(seg("rating", typ(filtering.KindFloat)))
	published    = fld(seg("published", typ(filtering.KindBool)))
	createTime   = fld(seg("create_time", typ(filtering.KindTimestamp)))
	readTime     = fld(seg("read_time", typ(filtering.KindDuration)))
	stateField   = fld(seg("state", stateType))
	authorField  = fld(seg("author", typ(filtering.KindMessage)))
	tagsField    = fld(seg("tags", elemTyp(filtering.KindRepeated, filtering.KindString)))
	chaptersType = elemTyp(filtering.KindRepeated, filtering.KindMessage)
)

func TestCheck(t *testing.T) {
	t.Parallel()

	opts := []cmp.Option{cmpopts.IgnoreUnexported(filtering.Type{})}

	for _, tt := range []struct {
		name   string
		filter string
		want   *filtering.CheckedFilter
	}{
		{
			name:   "empty filter",
			filter: "",
			want:   &filtering.CheckedFilter{},
		},
		{
			name:   "string equality with quoted literal",
			filter: `display_name = "war"`,
			want:   &filtering.CheckedFilter{Expr: cmpr(displayName, filtering.OpEquals, filtering.StringValue("war"))},
		},
		{
			name:   "string equality with text literal",
			filter: "display_name = war",
			want:   &filtering.CheckedFilter{Expr: cmpr(displayName, filtering.OpEquals, filtering.StringValue("war"))},
		},
		{
			name:   "suffix wildcard",
			filter: `display_name = "*.foo"`,
			want:   &filtering.CheckedFilter{Expr: cmpr(displayName, filtering.OpEquals, pattern(anyPart(), lit(".foo")))},
		},
		{
			name:   "prefix wildcard with not equals",
			filter: `display_name != "foo*"`,
			want:   &filtering.CheckedFilter{Expr: cmpr(displayName, filtering.OpNotEquals, pattern(lit("foo"), anyPart()))},
		},
		{
			name:   "inner wildcard",
			filter: `display_name = "a*b"`,
			want:   &filtering.CheckedFilter{Expr: cmpr(displayName, filtering.OpEquals, pattern(lit("a"), anyPart(), lit("b")))},
		},
		{
			name:   "bare star wildcard",
			filter: "display_name = *",
			want:   &filtering.CheckedFilter{Expr: cmpr(displayName, filtering.OpEquals, pattern(anyPart()))},
		},
		{
			name:   "integer comparison",
			filter: "page_count > 100",
			want:   &filtering.CheckedFilter{Expr: cmpr(pageCount, filtering.OpGreater, filtering.IntValue(100))},
		},
		{
			name:   "negative integer",
			filter: "page_count = -30",
			want:   &filtering.CheckedFilter{Expr: cmpr(pageCount, filtering.OpEquals, filtering.IntValue(-30))},
		},
		{
			name:   "quoted integer",
			filter: `page_count = "42"`,
			want:   &filtering.CheckedFilter{Expr: cmpr(pageCount, filtering.OpEquals, filtering.IntValue(42))},
		},
		{
			name:   "float comparison with decimal",
			filter: "rating >= 4.5",
			want:   &filtering.CheckedFilter{Expr: cmpr(rating, filtering.OpGreaterEquals, filtering.FloatValue(4.5))},
		},
		{
			name:   "float with exponent",
			filter: "rating < 2.997e9",
			want:   &filtering.CheckedFilter{Expr: cmpr(rating, filtering.OpLess, filtering.FloatValue(2.997e9))},
		},
		{
			name:   "integer literal on float field",
			filter: "rating >= 4",
			want:   &filtering.CheckedFilter{Expr: cmpr(rating, filtering.OpGreaterEquals, filtering.FloatValue(4))},
		},
		{
			name:   "bool equality",
			filter: "published = true",
			want:   &filtering.CheckedFilter{Expr: cmpr(published, filtering.OpEquals, filtering.BoolValue(true))},
		},
		{
			name:   "bool not equals false",
			filter: "published != false",
			want:   &filtering.CheckedFilter{Expr: cmpr(published, filtering.OpNotEquals, filtering.BoolValue(false))},
		},
		{
			name:   "bare bool field name is a search term",
			filter: "published",
			want:   &filtering.CheckedFilter{Expr: search("published")},
		},
		{
			name:   "negated bare bool field name",
			filter: "-published",
			want:   &filtering.CheckedFilter{Expr: &filtering.Not{Operand: search("published")}},
		},
		{
			name:   "bare nested bool field name is a search term",
			filter: "author.verified",
			want:   &filtering.CheckedFilter{Expr: search("author.verified")},
		},
		{
			name:   "enum equality",
			filter: "state = ACTIVE",
			want:   &filtering.CheckedFilter{Expr: cmpr(stateField, filtering.OpEquals, filtering.EnumValue("ACTIVE"))},
		},
		{
			name:   "quoted enum inequality",
			filter: `state != "DRAFT"`,
			want:   &filtering.CheckedFilter{Expr: cmpr(stateField, filtering.OpNotEquals, filtering.EnumValue("DRAFT"))},
		},
		{
			name:   "timestamp RFC 3339",
			filter: `create_time > "2021-02-14T10:00:00Z"`,
			want: &filtering.CheckedFilter{Expr: cmpr(createTime, filtering.OpGreater,
				filtering.TimestampValue(time.Date(2021, 2, 14, 10, 0, 0, 0, time.UTC)))},
		},
		{
			name:   "timestamp with offset",
			filter: `create_time = "2012-04-21T11:30:00-04:00"`,
			want: &filtering.CheckedFilter{Expr: cmpr(createTime, filtering.OpEquals,
				filtering.TimestampValue(time.Date(2012, 4, 21, 11, 30, 0, 0, time.FixedZone("", -4*60*60))))},
		},
		{
			name:   "duration seconds",
			filter: "read_time > 300s",
			want:   &filtering.CheckedFilter{Expr: cmpr(readTime, filtering.OpGreater, filtering.DurationValue(300*time.Second))},
		},
		{
			name:   "fractional duration",
			filter: "read_time <= 1.5s",
			want:   &filtering.CheckedFilter{Expr: cmpr(readTime, filtering.OpLessEquals, filtering.DurationValue(1500*time.Millisecond))},
		},
		{
			name:   "message null equality",
			filter: "author = null",
			want:   &filtering.CheckedFilter{Expr: cmpr(authorField, filtering.OpEquals, filtering.NullValue())},
		},
		{
			name:   "timestamp null inequality",
			filter: "create_time != null",
			want:   &filtering.CheckedFilter{Expr: cmpr(createTime, filtering.OpNotEquals, filtering.NullValue())},
		},
		{
			name:   "nested field",
			filter: `author.name = "Hugo"`,
			want: &filtering.CheckedFilter{Expr: cmpr(
				fld(seg("author", typ(filtering.KindMessage)), seg("name", typ(filtering.KindString))),
				filtering.OpEquals, filtering.StringValue("Hugo"),
			)},
		},
		{
			name:   "map value equality",
			filter: "labels.env = prod",
			want: &filtering.CheckedFilter{Expr: cmpr(
				fld(seg("labels", elemTyp(filtering.KindMap, filtering.KindString)), seg("env", typ(filtering.KindString))),
				filtering.OpEquals, filtering.StringValue("prod"),
			)},
		},
		{
			name:   "quoted map key",
			filter: `labels."my.key" = x`,
			want: &filtering.CheckedFilter{Expr: cmpr(
				fld(seg("labels", elemTyp(filtering.KindMap, filtering.KindString)), seg("my.key", typ(filtering.KindString))),
				filtering.OpEquals, filtering.StringValue("x"),
			)},
		},
		{
			name:   "has on repeated scalar",
			filter: "tags:go",
			want:   &filtering.CheckedFilter{Expr: cmpr(tagsField, filtering.OpHas, filtering.StringValue("go"))},
		},
		{
			name:   "has star on repeated",
			filter: "tags:*",
			want:   &filtering.CheckedFilter{Expr: cmpr(tagsField, filtering.OpHas, filtering.PresenceValue())},
		},
		{
			name:   "has on repeated message field",
			filter: "chapters.title:intro",
			want: &filtering.CheckedFilter{Expr: cmpr(
				fld(seg("chapters", chaptersType), seg("title", typ(filtering.KindString))),
				filtering.OpHas, filtering.StringValue("intro"),
			)},
		},
		{
			name:   "has map key normalizes to star",
			filter: "labels:env",
			want: &filtering.CheckedFilter{Expr: cmpr(
				fld(seg("labels", elemTyp(filtering.KindMap, filtering.KindString)), seg("env", typ(filtering.KindString))),
				filtering.OpHas, filtering.PresenceValue(),
			)},
		},
		{
			name:   "has map key star",
			filter: "labels.env:*",
			want: &filtering.CheckedFilter{Expr: cmpr(
				fld(seg("labels", elemTyp(filtering.KindMap, filtering.KindString)), seg("env", typ(filtering.KindString))),
				filtering.OpHas, filtering.PresenceValue(),
			)},
		},
		{
			name:   "has map key value",
			filter: "labels.env:prod",
			want: &filtering.CheckedFilter{Expr: cmpr(
				fld(seg("labels", elemTyp(filtering.KindMap, filtering.KindString)), seg("env", typ(filtering.KindString))),
				filtering.OpHas, filtering.StringValue("prod"),
			)},
		},
		{
			name:   "has message field normalizes to star",
			filter: "author:name",
			want: &filtering.CheckedFilter{Expr: cmpr(
				fld(seg("author", typ(filtering.KindMessage)), seg("name", typ(filtering.KindString))),
				filtering.OpHas, filtering.PresenceValue(),
			)},
		},
		{
			name:   "has star on message",
			filter: "author:*",
			want:   &filtering.CheckedFilter{Expr: cmpr(authorField, filtering.OpHas, filtering.PresenceValue())},
		},
		{
			name:   "has star on scalar is presence",
			filter: "display_name:*",
			want:   &filtering.CheckedFilter{Expr: cmpr(displayName, filtering.OpHas, filtering.PresenceValue())},
		},
		{
			name:   "bare term",
			filter: "Hugo",
			want:   &filtering.CheckedFilter{Expr: search("Hugo")},
		},
		{
			name:   "quoted bare term keeps spaces",
			filter: `"New York"`,
			want:   &filtering.CheckedFilter{Expr: search("New York")},
		},
		{
			name:   "bare dotted member is one term",
			filter: "a.b.c",
			want:   &filtering.CheckedFilter{Expr: search("a.b.c")},
		},
		{
			name:   "quoted field name is a search term",
			filter: `"published"`,
			want:   &filtering.CheckedFilter{Expr: search("published")},
		},
		{
			name:   "sequence of terms merges",
			filter: "foo bar",
			want:   &filtering.CheckedFilter{Expr: search("foo", "bar")},
		},
		{
			name:   "and of terms merges",
			filter: "foo AND bar",
			want:   &filtering.CheckedFilter{Expr: search("foo", "bar")},
		},
		{
			name:   "or of terms stays disjunction",
			filter: "foo OR bar",
			want:   &filtering.CheckedFilter{Expr: &filtering.Or{Operands: []filtering.Expr{search("foo"), search("bar")}}},
		},
		{
			name:   "negated term",
			filter: "NOT foo",
			want:   &filtering.CheckedFilter{Expr: &filtering.Not{Operand: search("foo")}},
		},
		{
			name:   "search terms and comparison",
			filter: "New York rating > 4.0",
			want: &filtering.CheckedFilter{Expr: &filtering.And{Operands: []filtering.Expr{
				search("New", "York"),
				cmpr(rating, filtering.OpGreater, filtering.FloatValue(4)),
			}}},
		},
		{
			name:   "and flattens across parens",
			filter: "(published = true rating > 4.0) AND state = ACTIVE",
			want: &filtering.CheckedFilter{Expr: &filtering.And{Operands: []filtering.Expr{
				cmpr(published, filtering.OpEquals, filtering.BoolValue(true)),
				cmpr(rating, filtering.OpGreater, filtering.FloatValue(4)),
				cmpr(stateField, filtering.OpEquals, filtering.EnumValue("ACTIVE")),
			}}},
		},
		{
			name:   "or binds tighter than and",
			filter: "published = true AND state = ACTIVE OR state = DRAFT",
			want: &filtering.CheckedFilter{Expr: &filtering.And{Operands: []filtering.Expr{
				cmpr(published, filtering.OpEquals, filtering.BoolValue(true)),
				&filtering.Or{Operands: []filtering.Expr{
					cmpr(stateField, filtering.OpEquals, filtering.EnumValue("ACTIVE")),
					cmpr(stateField, filtering.OpEquals, filtering.EnumValue("DRAFT")),
				}},
			}}},
		},
		{
			name:   "not composite",
			filter: "NOT (published = true OR rating > 4.0)",
			want: &filtering.CheckedFilter{Expr: &filtering.Not{Operand: &filtering.Or{Operands: []filtering.Expr{
				cmpr(published, filtering.OpEquals, filtering.BoolValue(true)),
				cmpr(rating, filtering.OpGreater, filtering.FloatValue(4)),
			}}}},
		},
		{
			name:   "macro expands",
			filter: "recent()",
			want:   &filtering.CheckedFilter{Expr: cmpr(createTime, filtering.OpGreater, filtering.TimestampValue(checkTime))},
		},
		{
			name:   "macro with literal argument",
			filter: `titled("war")`,
			want:   &filtering.CheckedFilter{Expr: cmpr(displayName, filtering.OpEquals, filtering.StringValue("war"))},
		},
		{
			name:   "negated macro",
			filter: "NOT recent()",
			want: &filtering.CheckedFilter{Expr: &filtering.Not{
				Operand: cmpr(createTime, filtering.OpGreater, filtering.TimestampValue(checkTime)),
			}},
		},
		{
			name:   "bare pass-through bool function",
			filter: `hasPrefix(display_name, "go")`,
			want: &filtering.CheckedFilter{Expr: cmpr(
				&filtering.FuncCall{
					Name:   "hasPrefix",
					Args:   []filtering.FuncArg{displayName, filtering.StringValue("go")},
					Result: typ(filtering.KindBool),
				},
				filtering.OpEquals, filtering.BoolValue(true),
			)},
		},
		{
			name:   "pass-through function comparison",
			filter: `title() = "war"`,
			want: &filtering.CheckedFilter{Expr: cmpr(
				&filtering.FuncCall{Name: "title", Result: typ(filtering.KindString)},
				filtering.OpEquals, filtering.StringValue("war"),
			)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := filtering.Parse(tt.filter)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.filter, err)
			}

			got, err := filtering.Check(parsed, checkSchema)
			if err != nil {
				t.Fatalf("Check(%q) error = %v", tt.filter, err)
			}

			if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
				t.Errorf("Check(%q) mismatch (-want +got):\n%s", tt.filter, diff)
			}
		})
	}

	t.Run("errors", func(t *testing.T) {
		t.Parallel()

		for _, tt := range []struct {
			name    string
			filter  string
			wantPos int
		}{
			{name: "unknown field", filter: "missing = 1", wantPos: 0},
			{name: "unknown nested field", filter: "author.nope = 1", wantPos: 7},
			{name: "traversal through scalar", filter: "display_name.x = 1", wantPos: 13},
			{name: "left-hand side must be a field", filter: "2.5 >= 2.4", wantPos: 0},
			{name: "non-numeric int literal", filter: "page_count = foo", wantPos: 13},
			{name: "ordering on bool", filter: "published > true", wantPos: 10},
			{name: "ordering on enum", filter: "state < ACTIVE", wantPos: 6},
			{name: "non-numeric float literal", filter: `rating = "x"`, wantPos: 9},
			{name: "quoted bool literal", filter: `published = "true"`, wantPos: 12},
			{name: "unknown enum value", filter: "state = BOGUS", wantPos: 8},
			{name: "wildcard with ordering", filter: `display_name > "a*"`, wantPos: 15},
			{name: "null on scalar", filter: "display_name = null", wantPos: 15},
			{name: "repeated without has", filter: "tags = go", wantPos: 5},
			{name: "traversing repeated without has", filter: `chapters.title = "x"`, wantPos: 0},
			{name: "has on repeated message without field", filter: "chapters:x", wantPos: 9},
			{name: "has on scalar with value", filter: "display_name:foo", wantPos: 13},
			{name: "message compared to string", filter: `author = "x"`, wantPos: 9},
			{name: "map compared without key", filter: "labels = x", wantPos: 7},
			{name: "duration without suffix", filter: "read_time > 300", wantPos: 12},
			{name: "duration too long", filter: "read_time > 9999999999s", wantPos: 12},
			{name: "negative duration too long", filter: "read_time > -9999999999s", wantPos: 12},
			{name: "invalid timestamp", filter: `create_time > "not-a-time"`, wantPos: 14},
			{name: "date-only timestamp", filter: "create_time >= 2021-02-14", wantPos: 15},
			{name: "unknown function", filter: "nope()", wantPos: 0},
			{name: "wrong arity", filter: "hasPrefix(display_name)", wantPos: 0},
			{name: "macro in comparison", filter: "recent() = true", wantPos: 0},
			{name: "bare non-bool function", filter: "title()", wantPos: 0},
			{name: "macro argument must be literal", filter: "titled(display_name)", wantPos: 7},
			{name: "composite argument", filter: "display_name = (foo)", wantPos: 15},
			{name: "function argument to comparison", filter: "display_name = title()", wantPos: 15},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				parsed, err := filtering.Parse(tt.filter)
				if err != nil {
					t.Fatalf("Parse(%q) error = %v", tt.filter, err)
				}

				_, err = filtering.Check(parsed, checkSchema)
				if err == nil {
					t.Fatalf("Check(%q) succeeded, want error", tt.filter)
				}

				if !errors.Is(err, filtering.ErrInvalidFilter) {
					t.Errorf("Check(%q) error = %v, want ErrInvalidFilter", tt.filter, err)
				}

				var cerr *filtering.CheckError
				if !errors.As(err, &cerr) {
					t.Fatalf("Check(%q) error = %T, want *CheckError", tt.filter, err)
				}

				if cerr.Pos != tt.wantPos {
					t.Errorf(
						"Check(%q) error = %q, position = %d, want %d",
						tt.filter,
						err,
						cerr.Pos,
						tt.wantPos,
					)
				}

				if cerr.Filter != tt.filter {
					t.Errorf("Check(%q) error Filter = %q, want the input", tt.filter, cerr.Filter)
				}
			})
		}
	})

	t.Run("expander error", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("boom")
		schema := filtering.NewSchema(
			filtering.Func(
				"explode",
				filtering.FuncExpand(
					func(_ *filtering.Schema, _ []filtering.Value) (filtering.Expr, error) {
						return nil, wantErr
					},
				),
			),
		)

		parsed, err := filtering.Parse("explode()")
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		_, err = filtering.Check(parsed, schema)
		if !errors.Is(err, wantErr) {
			t.Errorf("Check() error = %v, want to wrap the expander error", err)
		}

		if errors.Is(err, filtering.ErrInvalidFilter) {
			t.Errorf("Check() error = %v, must not claim the filter is invalid", err)
		}
	})
}
