package ast

import "testing"

func TestComparator_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		comp Comparator
		want string
	}{
		{"none", ComparatorNone, ""},
		{"equals", ComparatorEquals, "="},
		{"not equals", ComparatorNotEquals, "!="},
		{"less", ComparatorLess, "<"},
		{"less equals", ComparatorLessEquals, "<="},
		{"greater", ComparatorGreater, ">"},
		{"greater equals", ComparatorGreaterEquals, ">="},
		{"has", ComparatorHas, ":"},
		{"out of range", Comparator(99), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.comp.String(); got != tt.want {
				t.Errorf("Comparator(%d).String() = %q, want %q", tt.comp, got, tt.want)
			}
		})
	}
}

func TestMember_Text(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		member Member
		want   string
	}{
		{
			name:   "bare value",
			member: Member{Value: Value{Text: "a"}},
			want:   "a",
		},
		{
			name:   "dotted path",
			member: Member{Value: Value{Text: "a"}, Fields: []Value{{Text: "b"}, {Text: "c"}}},
			want:   "a.b.c",
		},
		{
			name: "quoted field keeps no quotes",
			member: Member{
				Value:  Value{Text: "m"},
				Fields: []Value{{Text: "key.with.dots", Quoted: true}},
			},
			want: "m.key.with.dots",
		},
		{
			name:   "number as dotted members",
			member: Member{Value: Value{Text: "2"}, Fields: []Value{{Text: "5"}}},
			want:   "2.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.member.Text(); got != tt.want {
				t.Errorf("Member.Text() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFunction_Text(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   Function
		want string
	}{
		{
			name: "single segment",
			fn:   Function{Name: []Value{{Text: "regex"}}},
			want: "regex",
		},
		{
			name: "dotted name",
			fn:   Function{Name: []Value{{Text: "math"}, {Text: "mem"}}},
			want: "math.mem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.fn.Text(); got != tt.want {
				t.Errorf("Function.Text() = %q, want %q", got, tt.want)
			}
		})
	}
}
