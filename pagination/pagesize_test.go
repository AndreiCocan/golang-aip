package pagination_test

import (
	"errors"
	"testing"

	"github.com/AndreiCocan/golang-aip/pagination"
)

func TestPageSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		requested   int32
		defaultSize int32
		maxSize     int32
		want        int32
		wantErr     bool
	}{
		{"zero requested uses default", 0, 25, 100, 25, false},
		{"positive passthrough", 10, 25, 100, 10, false},
		{"requested equal to max", 100, 25, 100, 100, false},
		{"requested above max capped", 500, 25, 100, 100, false},
		{"max zero means no cap", 5000, 25, 0, 5000, false},
		{"default above max capped", 0, 200, 100, 100, false},
		{"negative requested", -1, 25, 100, 0, true},
		{"zero default", 10, 0, 100, 0, true},
		{"negative default", 10, -25, 100, 0, true},
		{"negative max", 10, 25, -100, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := pagination.PageSize(tt.requested, tt.defaultSize, tt.maxSize)
			if tt.wantErr {
				if !errors.Is(err, pagination.ErrInvalidPageSize) {
					t.Errorf(
						"PageSize(%d, %d, %d) error = %v, want ErrInvalidPageSize",
						tt.requested, tt.defaultSize, tt.maxSize, err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"PageSize(%d, %d, %d) error: %v",
					tt.requested, tt.defaultSize, tt.maxSize, err,
				)
			}

			if got != tt.want {
				t.Errorf(
					"PageSize(%d, %d, %d) = %d, want %d",
					tt.requested, tt.defaultSize, tt.maxSize, got, tt.want,
				)
			}
		})
	}
}
