package filtering

// Kind classifies both the declared type of a schema field and the resolved
// type of a checked [Value]. The kinds up to and including KindMap appear in
// schemas; KindNull, KindStar, and KindPattern appear only in values, where
// the schema context has given a literal one of those special meanings.
type Kind int

const (
	// KindInvalid is the zero Kind and never appears in a checked filter.
	KindInvalid Kind = iota
	// KindString is a text field or literal.
	KindString
	// KindInt is a 64-bit signed integer field or literal.
	KindInt
	// KindFloat is a 64-bit floating-point field or literal.
	KindFloat
	// KindBool is a boolean field or the literal true or false.
	KindBool
	// KindEnum is a field restricted to a set of named values, or one of
	// those names. Enum names are case-sensitive.
	KindEnum
	// KindTimestamp is a point-in-time field or an RFC 3339 literal.
	KindTimestamp
	// KindDuration is a duration field or a seconds literal such as "20s".
	KindDuration
	// KindMessage is a structured field with named subfields.
	KindMessage
	// KindRepeated is a list field. Its element type is [Type.Elem].
	KindRepeated
	// KindMap is a string-keyed map field. Its value type is [Type.Elem].
	KindMap
	// KindNull is the literal null, valid against message-backed fields
	// (messages, timestamps, durations) to test unsetness.
	KindNull
	// KindStar is the bare * argument of a has restriction: a presence
	// test, as in `author:*`.
	KindStar
	// KindPattern is a string wildcard pattern such as "*.foo". The
	// resolved segments are in [Value.Pattern].
	KindPattern
)

// String returns a human-readable name for the kind.
func (k Kind) String() string {
	switch k {
	case KindString:
		return "string"
	case KindInt:
		return "int"
	case KindFloat:
		return "float"
	case KindBool:
		return "bool"
	case KindEnum:
		return "enum"
	case KindTimestamp:
		return "timestamp"
	case KindDuration:
		return "duration"
	case KindMessage:
		return "message"
	case KindRepeated:
		return "repeated"
	case KindMap:
		return "map"
	case KindNull:
		return "null"
	case KindStar:
		return "*"
	case KindPattern:
		return "pattern"
	default:
		return "invalid"
	}
}

// Type is the resolved type of a schema field. For KindRepeated and KindMap,
// Elem is the element or value type; for KindEnum, Enum lists the valid
// value names.
type Type struct {
	Kind Kind
	Elem *Type
	Enum []string

	// msg holds the subfields of a KindMessage type.
	msg *messageType
}

type messageType struct {
	fields map[string]Type
}
