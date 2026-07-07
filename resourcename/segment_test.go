package resourcename

import "testing"

func TestSegment_IsVariable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		segment Segment
		want    bool
	}{
		{"{publisher}", true},
		{"{a}", true},
		{"{book_shelf}", true},
		{"publishers", false},
		{"-", false},
		{"", false},
		{"{}", false},
		{"{unclosed", false},
		{"unopened}", false},
		{"pre{fix}", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.segment), func(t *testing.T) {
			t.Parallel()

			if got := tt.segment.IsVariable(); got != tt.want {
				t.Errorf("Segment(%q).IsVariable() = %v, want %v", tt.segment, got, tt.want)
			}
		})
	}
}

func TestSegment_IsWildcard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		segment Segment
		want    bool
	}{
		{"-", true},
		{"--", false},
		{"a-b", false},
		{"", false},
		{"{-}", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.segment), func(t *testing.T) {
			t.Parallel()

			if got := tt.segment.IsWildcard(); got != tt.want {
				t.Errorf("Segment(%q).IsWildcard() = %v, want %v", tt.segment, got, tt.want)
			}
		})
	}
}

func TestSegment_Literal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		segment Segment
		want    string
	}{
		{"{publisher}", "publisher"},
		{"publishers", "publishers"},
		{"-", "-"},
		{"", ""},
		{"{}", "{}"},
	}

	for _, tt := range tests {
		t.Run(string(tt.segment), func(t *testing.T) {
			t.Parallel()

			if got := tt.segment.Literal(); got != tt.want {
				t.Errorf("Segment(%q).Literal() = %q, want %q", tt.segment, got, tt.want)
			}
		})
	}
}
