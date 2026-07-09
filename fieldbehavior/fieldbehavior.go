package fieldbehavior

import (
	"slices"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Get returns the field behaviors annotated on the field, in declaration
// order. It returns nil for a field without behavior annotations.
func Get(field protoreflect.FieldDescriptor) []annotations.FieldBehavior {
	opts, ok := field.Options().(*descriptorpb.FieldOptions)
	if !ok || opts == nil {
		return nil
	}

	behaviors, ok := proto.GetExtension(opts, annotations.E_FieldBehavior).([]annotations.FieldBehavior)
	if !ok {
		return nil
	}

	return behaviors
}

// Has reports whether the field is annotated with the given behavior.
func Has(field protoreflect.FieldDescriptor, want annotations.FieldBehavior) bool {
	return slices.Contains(Get(field), want)
}
