package fieldmask_test

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/AndreiCocan/golang-aip/fieldmask"
	"github.com/AndreiCocan/golang-aip/internal/testproto"
)

func TestCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		paths   []string
		nilMask bool
		wantErr string // substring of the error, "" means valid
	}{
		{name: "nil mask", nilMask: true},
		{name: "empty mask", paths: []string{}},
		{name: "top-level scalar", paths: []string{"title"}},
		{name: "top-level message", paths: []string{"author"}},
		{name: "message subfield", paths: []string{"author.name"}},
		{name: "whole map", paths: []string{"labels"}},
		{name: "map string key", paths: []string{"labels.lang"}},
		{name: "backticked map key", paths: []string{"labels.`k8s.io/name`"}},
		{name: "integer map key", paths: []string{"editions.5"}},
		{name: "non-canonical integer map key", paths: []string{"editions.05"}},
		{name: "backticked wildcard map key", paths: []string{"labels.`*`"}},
		{name: "backticked field name", paths: []string{"`title`"}},
		{name: "map key then subfield", paths: []string{"reviews.smith.rating"}},
		{name: "whole repeated field", paths: []string{"shelves"}},
		{name: "several paths", paths: []string{"title", "author.name", "labels"}},
		{name: "duplicate paths", paths: []string{"title", "title"}},
		{name: "wildcard alone", paths: []string{"*"}},

		{name: "empty path", paths: []string{""}, wantErr: "empty"},
		{name: "unknown field", paths: []string{"nope"}, wantErr: `"nope"`},
		{name: "unknown subfield", paths: []string{"author.nope"}, wantErr: `"author.nope"`},
		{name: "index into repeated", paths: []string{"shelves.0"}, wantErr: "repeated"},
		{name: "traversal into repeated", paths: []string{"shelves.name"}, wantErr: "repeated"},
		{name: "traversal into scalar", paths: []string{"title.foo"}, wantErr: `"title.foo"`},
		{
			name:    "traversal into scalar map value",
			paths:   []string{"labels.lang.x"},
			wantErr: `"labels.lang.x"`,
		},
		{name: "empty segment", paths: []string{"author..name"}, wantErr: "empty"},
		{
			name:    "non-integer key for an integer map",
			paths:   []string{"editions.x"},
			wantErr: "integer",
		},
		{name: "unterminated backtick", paths: []string{"labels.`oops"}, wantErr: "backtick"},
		{name: "backtick inside a segment", paths: []string{"la`bels"}, wantErr: "backtick"},
		{
			name:    "missing dot after backtick quote",
			paths:   []string{"labels.`k`x"},
			wantErr: "backtick",
		},
		{name: "trailing dot", paths: []string{"author."}, wantErr: "empty"},
		{name: "wildcard with other paths", paths: []string{"*", "title"}, wantErr: "wildcard"},
		{name: "wildcard as subfield", paths: []string{"author.*"}, wantErr: "wildcard"},
		{name: "wildcard as an unquoted map key", paths: []string{"labels.*"}, wantErr: "wildcard"},
		{name: "backticked wildcard as a field name", paths: []string{"`*`"}, wantErr: `"*"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mask *fieldmaskpb.FieldMask
			if !tt.nilMask {
				mask = &fieldmaskpb.FieldMask{Paths: tt.paths}
			}

			err := fieldmask.Check(mask, &testproto.Book{})
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Check() = %v, want nil", err)
				}

				return
			}

			if err == nil {
				t.Fatalf("Check() = nil, want error containing %q", tt.wantErr)
			}

			if !errors.Is(err, fieldmask.ErrInvalidFieldMask) {
				t.Fatalf("error %v does not match ErrInvalidFieldMask", err)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestIsFullReplacement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mask *fieldmaskpb.FieldMask
		want bool
	}{
		{"nil mask", nil, false},
		{"wildcard", &fieldmaskpb.FieldMask{Paths: []string{"*"}}, true},
		{"plain paths", &fieldmaskpb.FieldMask{Paths: []string{"title"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := fieldmask.IsFullReplacement(tt.mask); got != tt.want {
				t.Fatalf("IsFullReplacement() = %v, want %v", got, tt.want)
			}
		})
	}
}
