package fieldbehavior_test

import (
	"testing"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/AndreiCocan/golang-aip/fieldbehavior"
	"github.com/AndreiCocan/golang-aip/internal/testproto"
)

// fullBook returns a Book with every field populated, shared by the
// package's tests as the canonical starting state.
func fullBook() *testproto.Book {
	return &testproto.Book{
		Name:        "shelves/1/books/1",
		Title:       "The Go Programming Language",
		Author:      &testproto.Author{Name: "Alan Donovan", Age: 50, Verified: true},
		Isbn:        "978-0134190440",
		CreateTime:  "2026-01-01T00:00:00Z",
		ImportToken: "secret",
		Labels:      map[string]string{"lang": "en"},
		Shelves:     []string{"shelves/1"},
		Reviews:     map[string]*testproto.Review{"smith": {Text: "great", Rating: 5}},
		PageCount:   380,
		Editions:    map[int32]string{1: "first"},
	}
}

func bookField(t *testing.T, name protoreflect.Name) protoreflect.FieldDescriptor {
	t.Helper()

	fd := (&testproto.Book{}).ProtoReflect().Descriptor().Fields().ByName(name)
	if fd == nil {
		t.Fatalf("field %q not found on Book", name)
	}

	return fd
}

func TestGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		field protoreflect.Name
		want  []annotations.FieldBehavior
	}{
		{"name", []annotations.FieldBehavior{annotations.FieldBehavior_IDENTIFIER}},
		{"title", []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED}},
		{"isbn", []annotations.FieldBehavior{annotations.FieldBehavior_IMMUTABLE}},
		{"create_time", []annotations.FieldBehavior{annotations.FieldBehavior_OUTPUT_ONLY}},
		{"import_token", []annotations.FieldBehavior{annotations.FieldBehavior_INPUT_ONLY}},
		{"labels", []annotations.FieldBehavior{annotations.FieldBehavior_OPTIONAL}},
	}
	for _, tt := range tests {
		t.Run(string(tt.field), func(t *testing.T) {
			t.Parallel()

			got := fieldbehavior.Get(bookField(t, tt.field))
			if len(got) != len(tt.want) {
				t.Fatalf("Get() = %v, want %v", got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("Get() = %v, want %v", got, tt.want)
				}
			}
		})
	}

	t.Run("field without annotations", func(t *testing.T) {
		t.Parallel()

		fd := (&fieldmaskpb.FieldMask{}).ProtoReflect().Descriptor().Fields().ByName("paths")
		if got := fieldbehavior.Get(fd); got != nil {
			t.Fatalf("Get() = %v, want nil", got)
		}
	})
}

func TestHas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		field protoreflect.Name
		want  annotations.FieldBehavior
		ok    bool
	}{
		{
			"output_only field has OUTPUT_ONLY",
			"create_time",
			annotations.FieldBehavior_OUTPUT_ONLY,
			true,
		},
		{
			"output_only field lacks REQUIRED",
			"create_time",
			annotations.FieldBehavior_REQUIRED,
			false,
		},
		{"identifier field has IDENTIFIER", "name", annotations.FieldBehavior_IDENTIFIER, true},
		{"optional field lacks IMMUTABLE", "labels", annotations.FieldBehavior_IMMUTABLE, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := fieldbehavior.Has(bookField(t, tt.field), tt.want); got != tt.ok {
				t.Fatalf("Has(%s, %v) = %v, want %v", tt.field, tt.want, got, tt.ok)
			}
		})
	}
}
