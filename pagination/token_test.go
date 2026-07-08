package pagination_test

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/AndreiCocan/golang-aip/pagination"
)

// bookCursor is the cursor shape most token tests round-trip.
type bookCursor struct {
	PublishTime time.Time
	ID          string
}

type authorCursor struct {
	Name string
	Born *int
}

type shelfPosition struct {
	Row  int
	Slot int
}

type shelfCursor struct {
	Position shelfPosition
	Tags     []string
	Counts   map[string]int
}

// renamedCursor is bookCursor after an ID field rename: gob would decode
// an old token into it with the renamed field silently zeroed.
type renamedCursor struct {
	PublishTime time.Time
	BookID      string
}

// retypedCursor is bookCursor after an ID field type change.
type retypedCursor struct {
	PublishTime time.Time
	ID          int64
}

// sameShapeCursor is structurally identical to bookCursor under a
// different type name.
type sameShapeCursor struct {
	PublishTime time.Time
	ID          string
}

// gridCursor has an array field.
type gridCursor struct {
	Cells [2]int
}

// linkedCursor is a recursive type, exercising the shape checksum's cycle
// guard.
type linkedCursor struct {
	ID   string
	Prev *linkedCursor
}

// unencodableCursor has a field gob cannot encode.
type unencodableCursor struct {
	Done chan struct{}
}

// opaqueCursor has no exported fields and no GobEncoder, so gob has
// nothing to encode.
type opaqueCursor struct {
	id string
}

// mint parses an empty token under args and mints a next-page token from
// cursor.
func mint(t *testing.T, cursor any, args ...any) string {
	t.Helper()

	first, err := pagination.Parse("", args...)
	if err != nil {
		t.Fatalf("Parse(%q, %v) error: %v", "", args, err)
	}

	token, err := first.Next(cursor)
	if err != nil {
		t.Fatalf("Next(%+v) error: %v", cursor, err)
	}

	return token
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("request args", func(t *testing.T) {
		t.Parallel()

		filter := `author = "hugo"`

		tests := []struct {
			name      string
			mintArgs  []any
			parseArgs []any
			wantErr   bool
		}{
			{
				"same args accepted",
				[]any{filter, "publish_time desc"},
				[]any{filter, "publish_time desc"},
				false,
			},
			{
				"no args on both sides accepted",
				nil,
				nil,
				false,
			},
			{
				"changed arg rejected",
				[]any{filter},
				[]any{`author = "verne"`},
				true,
			},
			{
				"reordered args rejected",
				[]any{filter, "publish_time desc"},
				[]any{"publish_time desc", filter},
				true,
			},
			{
				"dropped arg rejected",
				[]any{filter, "publish_time desc"},
				[]any{filter},
				true,
			},
			{
				"added arg rejected",
				[]any{filter},
				[]any{filter, "publish_time desc"},
				true,
			},
			{
				"resplit args rejected",
				[]any{"ab", "c"},
				[]any{"a", "bc"},
				true,
			},
			{
				"pointer arg hashes as its value",
				[]any{&filter},
				[]any{filter},
				false,
			},
			{
				"mixed arg types accepted",
				[]any{"shelves/1", int32(3), true},
				[]any{"shelves/1", int32(3), true},
				false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				token := mint(t, bookCursor{ID: "books/1"}, tt.mintArgs...)

				_, err := pagination.Parse(token, tt.parseArgs...)
				if tt.wantErr {
					if !errors.Is(err, pagination.ErrInvalidPageToken) {
						t.Errorf(
							"Parse(token, %v) error = %v, want ErrInvalidPageToken",
							tt.parseArgs, err,
						)
					}

					return
				}

				if err != nil {
					t.Errorf("Parse(token, %v) error: %v", tt.parseArgs, err)
				}
			})
		}
	})

	t.Run("corrupt token", func(t *testing.T) {
		t.Parallel()

		minted := mint(t, bookCursor{ID: "books/1"})

		raw, err := base64.RawURLEncoding.DecodeString(minted)
		if err != nil {
			t.Fatalf("decoding minted token: %v", err)
		}

		wrongVersion := append([]byte{}, raw...)
		wrongVersion[0] = 2

		tests := []struct {
			name  string
			token string
		}{
			{"not base64", "not/valid/base64!"},
			{"base64 of garbage", base64.RawURLEncoding.EncodeToString([]byte("garbage"))},
			{"header only", base64.RawURLEncoding.EncodeToString(raw[:17])},
			{"wrong version byte", base64.RawURLEncoding.EncodeToString(wrongVersion)},
			{"truncated", minted[:len(minted)/2]},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				// Truncation inside the payload is only detectable when
				// the cursor is decoded, so corruption must surface as
				// ErrInvalidPageToken from Parse or from Cursor.
				parsed, err := pagination.Parse(tt.token)
				if err == nil {
					var dst bookCursor

					_, err = parsed.Cursor(&dst)
				}

				if !errors.Is(err, pagination.ErrInvalidPageToken) {
					t.Errorf(
						"Parse(%q) pipeline error = %v, want ErrInvalidPageToken",
						tt.token,
						err,
					)
				}
			})
		}
	})
}

func TestToken_Cursor(t *testing.T) {
	t.Parallel()

	t.Run("round trip", func(t *testing.T) {
		t.Parallel()

		born := 1970

		tests := []struct {
			name   string
			cursor any
			dst    any
			want   any
		}{
			{
				"time and string fields",
				bookCursor{
					PublishTime: time.Date(2021, 2, 14, 9, 30, 0, 0, time.UTC),
					ID:          "books/les-miserables",
				},
				&bookCursor{},
				&bookCursor{
					PublishTime: time.Date(2021, 2, 14, 9, 30, 0, 0, time.UTC),
					ID:          "books/les-miserables",
				},
			},
			{
				"nil pointer field",
				authorCursor{Name: "hugo"},
				&authorCursor{},
				&authorCursor{Name: "hugo"},
			},
			{
				"non-nil pointer field",
				authorCursor{Name: "hugo", Born: &born},
				&authorCursor{},
				&authorCursor{Name: "hugo", Born: &born},
			},
			{
				"nested struct, slice, and map fields",
				shelfCursor{
					Position: shelfPosition{Row: 3, Slot: 7},
					Tags:     []string{"classics", "french"},
					Counts:   map[string]int{"read": 12, "unread": 3},
				},
				&shelfCursor{},
				&shelfCursor{
					Position: shelfPosition{Row: 3, Slot: 7},
					Tags:     []string{"classics", "french"},
					Counts:   map[string]int{"read": 12, "unread": 3},
				},
			},
			{
				"cursor passed as pointer",
				&bookCursor{ID: "books/notre-dame"},
				&bookCursor{},
				&bookCursor{ID: "books/notre-dame"},
			},
			{
				"array field",
				gridCursor{Cells: [2]int{3, 7}},
				&gridCursor{},
				&gridCursor{Cells: [2]int{3, 7}},
			},
			{
				"recursive type",
				linkedCursor{ID: "books/2", Prev: &linkedCursor{ID: "books/1"}},
				&linkedCursor{},
				&linkedCursor{ID: "books/2", Prev: &linkedCursor{ID: "books/1"}},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				parsed, err := pagination.Parse(mint(t, tt.cursor))
				if err != nil {
					t.Fatalf("Parse(token) error: %v", err)
				}

				ok, err := parsed.Cursor(tt.dst)
				if err != nil {
					t.Fatalf("Cursor() error: %v", err)
				}

				if !ok {
					t.Fatal("Cursor() = false, want true for a minted token")
				}

				if diff := cmp.Diff(tt.want, tt.dst); diff != "" {
					t.Errorf("Cursor() mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("first page", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			token func() (pagination.Token, error)
		}{
			{
				"parsed empty token",
				func() (pagination.Token, error) { return pagination.Parse("") },
			},
			{
				"zero token",
				func() (pagination.Token, error) { return pagination.Token{}, nil },
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				tok, err := tt.token()
				if err != nil {
					t.Fatalf("building token: %v", err)
				}

				dst := bookCursor{ID: "untouched"}

				ok, err := tok.Cursor(&dst)
				if err != nil {
					t.Fatalf("Cursor() error: %v", err)
				}

				if ok {
					t.Error("Cursor() = true, want false for a first page")
				}

				if dst.ID != "untouched" {
					t.Errorf("Cursor() modified dst to %+v, want it untouched", dst)
				}
			})
		}
	})

	t.Run("shape drift", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			dst     any
			wantErr bool
		}{
			{"same type accepted", &bookCursor{}, false},
			{"same shape under another name accepted", &sameShapeCursor{}, false},
			{"renamed field rejected", &renamedCursor{}, true},
			{"retyped field rejected", &retypedCursor{}, true},
			{"unrelated shape rejected", &shelfCursor{}, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				token := mint(t, bookCursor{
					PublishTime: time.Date(2021, 2, 14, 9, 30, 0, 0, time.UTC),
					ID:          "books/les-miserables",
				})

				parsed, err := pagination.Parse(token)
				if err != nil {
					t.Fatalf("Parse(token) error: %v", err)
				}

				ok, err := parsed.Cursor(tt.dst)
				if tt.wantErr {
					if !errors.Is(err, pagination.ErrInvalidPageToken) {
						t.Errorf("Cursor(%T) error = %v, want ErrInvalidPageToken", tt.dst, err)
					}

					return
				}

				if err != nil {
					t.Fatalf("Cursor(%T) error: %v", tt.dst, err)
				}

				if !ok {
					t.Errorf("Cursor(%T) = false, want true", tt.dst)
				}
			})
		}
	})

	t.Run("corrupt payload", func(t *testing.T) {
		t.Parallel()

		raw, err := base64.RawURLEncoding.DecodeString(mint(t, bookCursor{ID: "books/1"}))
		if err != nil {
			t.Fatalf("decoding minted token: %v", err)
		}

		// Keep the header intact so the shape checksum still matches, but
		// replace the gob payload with garbage.
		corrupt := append(append([]byte{}, raw[:17]...), "garbage"...)

		parsed, err := pagination.Parse(base64.RawURLEncoding.EncodeToString(corrupt))
		if err != nil {
			t.Fatalf("Parse(corrupt) error: %v", err)
		}

		var dst bookCursor
		if _, err := parsed.Cursor(&dst); !errors.Is(err, pagination.ErrInvalidPageToken) {
			t.Errorf("Cursor() error = %v, want ErrInvalidPageToken", err)
		}
	})

	t.Run("invalid dst", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			dst  any
		}{
			{"nil", nil},
			{"typed nil pointer", (*bookCursor)(nil)},
			{"not a pointer", bookCursor{}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				parsed, err := pagination.Parse(mint(t, bookCursor{ID: "books/1"}))
				if err != nil {
					t.Fatalf("Parse(token) error: %v", err)
				}

				if _, err := parsed.Cursor(tt.dst); !errors.Is(err, pagination.ErrInvalidCursor) {
					t.Errorf("Cursor(%#v) error = %v, want ErrInvalidCursor", tt.dst, err)
				}
			})
		}

		t.Run("rejected even on a first page", func(t *testing.T) {
			t.Parallel()

			var tok pagination.Token

			if _, err := tok.Cursor(nil); !errors.Is(err, pagination.ErrInvalidCursor) {
				t.Errorf("Cursor(nil) error = %v, want ErrInvalidCursor", err)
			}
		})
	})
}

func TestToken_Next(t *testing.T) {
	t.Parallel()

	t.Run("zero token mints like a parsed empty token", func(t *testing.T) {
		t.Parallel()

		var first pagination.Token

		token, err := first.Next(bookCursor{ID: "books/1"})
		if err != nil {
			t.Fatalf("Next() error: %v", err)
		}

		if _, err := pagination.Parse(token); err != nil {
			t.Errorf("Parse(token) error = %v, want a zero Token to mint like Parse(%q)", err, "")
		}
	})

	t.Run("carries request args across pages", func(t *testing.T) {
		t.Parallel()

		args := []any{`author = "hugo"`}

		second, err := pagination.Parse(mint(t, bookCursor{ID: "books/1"}, args...), args...)
		if err != nil {
			t.Fatalf("Parse(second, %v) error: %v", args, err)
		}

		third, err := second.Next(bookCursor{ID: "books/2"})
		if err != nil {
			t.Fatalf("Next() error: %v", err)
		}

		if _, err := pagination.Parse(third, args...); err != nil {
			t.Errorf("Parse(third, %v) error = %v, want the args checksum carried over", args, err)
		}

		if _, err := pagination.Parse(third); err == nil {
			t.Error("Parse(third) error = nil, want ErrInvalidPageToken without the original args")
		}
	})

	t.Run("invalid cursor", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			cursor any
		}{
			{"nil", nil},
			{"typed nil pointer", (*bookCursor)(nil)},
			{"channel field", unencodableCursor{Done: make(chan struct{})}},
			{"no exported fields", opaqueCursor{id: "books/1"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var tok pagination.Token

				_, err := tok.Next(tt.cursor)
				if !errors.Is(err, pagination.ErrInvalidCursor) {
					t.Errorf("Next(%#v) error = %v, want ErrInvalidCursor", tt.cursor, err)
				}
			})
		}
	})
}
