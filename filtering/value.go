package filtering

import "time"

// Value is a literal that [Check] resolved against the schema: the right-
// hand side of a comparison or a literal function argument. Kind says which
// of the typed fields is meaningful; all others are zero.
//
// Values are produced by Check, and by the exported constructors for use in
// function expanders and tests.
type Value struct {
	Kind Kind

	// Str is the value of a KindString value, or the name of a KindEnum
	// value.
	Str string
	// Int is the value of a KindInt value.
	Int int64
	// Float is the value of a KindFloat value.
	Float float64
	// Bool is the value of a KindBool value.
	Bool bool
	// Time is the value of a KindTimestamp value.
	Time time.Time
	// Duration is the value of a KindDuration value.
	Duration time.Duration
	// Pattern holds the segments of a KindPattern value, in order.
	Pattern []PatternPart
}

func (Value) isFuncArg() {}

// PatternPart is one segment of a wildcard string pattern. Either Wildcard
// is true and the part matches any run of characters (including none), or
// Literal must match exactly.
type PatternPart struct {
	Wildcard bool
	Literal  string
}

// StringValue returns a KindString value.
func StringValue(v string) Value { return Value{Kind: KindString, Str: v} }

// IntValue returns a KindInt value.
func IntValue(v int64) Value { return Value{Kind: KindInt, Int: v} }

// FloatValue returns a KindFloat value.
func FloatValue(v float64) Value { return Value{Kind: KindFloat, Float: v} }

// BoolValue returns a KindBool value.
func BoolValue(v bool) Value { return Value{Kind: KindBool, Bool: v} }

// EnumValue returns a KindEnum value with the given name.
func EnumValue(name string) Value { return Value{Kind: KindEnum, Str: name} }

// TimestampValue returns a KindTimestamp value.
func TimestampValue(t time.Time) Value { return Value{Kind: KindTimestamp, Time: t} }

// DurationValue returns a KindDuration value.
func DurationValue(d time.Duration) Value { return Value{Kind: KindDuration, Duration: d} }

// NullValue returns the KindNull value, the literal null.
func NullValue() Value { return Value{Kind: KindNull} }

// StarValue returns the KindStar value, the presence test *.
func StarValue() Value { return Value{Kind: KindStar} }
