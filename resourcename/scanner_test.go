package resourcename

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestScanner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		wantSegments []string
		wantFull     bool
		wantService  string
	}{
		{
			name:         "relative name",
			input:        "publishers/123/books/les-miserables",
			wantSegments: []string{"publishers", "123", "books", "les-miserables"},
		},
		{
			name:         "single segment",
			input:        "publishers",
			wantSegments: []string{"publishers"},
		},
		{
			name:         "empty input",
			input:        "",
			wantSegments: nil,
		},
		{
			name:         "full name",
			input:        "//library.example.com/publishers/123",
			wantSegments: []string{"publishers", "123"},
			wantFull:     true,
			wantService:  "library.example.com",
		},
		{
			name:         "full name host only",
			input:        "//library.example.com",
			wantSegments: nil,
			wantFull:     true,
			wantService:  "library.example.com",
		},
		{
			name:         "leading slash is skipped",
			input:        "/publishers/123",
			wantSegments: []string{"publishers", "123"},
		},
		{
			name:         "empty segment surfaced",
			input:        "publishers//books",
			wantSegments: []string{"publishers", "", "books"},
		},
		{
			name:         "trailing slash yields empty segment",
			input:        "publishers/",
			wantSegments: []string{"publishers", ""},
		},
		{
			name:         "wildcard and variable segments",
			input:        "publishers/-/books/{book}",
			wantSegments: []string{"publishers", "-", "books", "{book}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var sc Scanner
			sc.Init(tt.input)

			var segments []string
			for sc.Scan() {
				segments = append(segments, string(sc.Segment()))
			}

			if diff := cmp.Diff(tt.wantSegments, segments); diff != "" {
				t.Errorf("segments mismatch (-want +got):\n%s", diff)
			}

			if sc.Full() != tt.wantFull {
				t.Errorf("Full() = %v, want %v", sc.Full(), tt.wantFull)
			}

			if sc.ServiceName() != tt.wantService {
				t.Errorf("ServiceName() = %q, want %q", sc.ServiceName(), tt.wantService)
			}
		})
	}
}

func TestScanner_startEnd(t *testing.T) {
	t.Parallel()

	const input = "publishers/123/books"

	var sc Scanner
	sc.Init(input)

	type span struct{ start, end int }

	var spans []span
	for sc.Scan() {
		spans = append(spans, span{sc.Start(), sc.End()})
		if got, want := input[sc.Start():sc.End()], string(sc.Segment()); got != want {
			t.Errorf("input[Start():End()] = %q, want Segment() %q", got, want)
		}
	}

	want := []span{{0, 10}, {11, 14}, {15, 20}}
	if diff := cmp.Diff(want, spans, cmp.AllowUnexported(span{})); diff != "" {
		t.Errorf("spans mismatch (-want +got):\n%s", diff)
	}
}

func TestScanner_reuseAfterInit(t *testing.T) {
	t.Parallel()

	var sc Scanner

	sc.Init("//library.example.com/publishers/123")

	for sc.Scan() {
	}

	sc.Init("shelves/1")

	var segments []string
	for sc.Scan() {
		segments = append(segments, string(sc.Segment()))
	}

	if diff := cmp.Diff([]string{"shelves", "1"}, segments); diff != "" {
		t.Errorf("segments mismatch (-want +got):\n%s", diff)
	}

	if sc.Full() {
		t.Error("Full() = true after re-Init with relative name, want false")
	}

	if sc.ServiceName() != "" {
		t.Errorf("ServiceName() = %q after re-Init, want empty", sc.ServiceName())
	}
}
