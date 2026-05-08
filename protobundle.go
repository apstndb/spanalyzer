package spanalyzer

import (
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type ProtoDescriptorSet struct {
	files    *protoregistry.Files
	fileSet  *descriptorpb.FileDescriptorSet
	messages map[string]protoreflect.MessageDescriptor
	enums    map[string]protoreflect.EnumDescriptor
}

func LoadProtoDescriptorSetFiles(paths []string) (*ProtoDescriptorSet, error) {
	merged := &descriptorpb.FileDescriptorSet{}
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var set descriptorpb.FileDescriptorSet
		if err := proto.Unmarshal(b, &set); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		merged.File = append(merged.File, set.File...)
	}
	files, err := protodesc.NewFiles(merged)
	if err != nil {
		return nil, err
	}
	out := &ProtoDescriptorSet{
		files:    files,
		fileSet:  merged,
		messages: map[string]protoreflect.MessageDescriptor{},
		enums:    map[string]protoreflect.EnumDescriptor{},
	}
	files.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		collectFileDescriptors(out, file)
		return true
	})
	return out, nil
}

func collectFileDescriptors(out *ProtoDescriptorSet, file protoreflect.FileDescriptor) {
	for i := 0; i < file.Messages().Len(); i++ {
		collectMessageDescriptors(out, file.Messages().Get(i))
	}
	for i := 0; i < file.Enums().Len(); i++ {
		enum := file.Enums().Get(i)
		out.enums[string(enum.FullName())] = enum
	}
}

func collectMessageDescriptors(out *ProtoDescriptorSet, msg protoreflect.MessageDescriptor) {
	out.messages[string(msg.FullName())] = msg
	for i := 0; i < msg.Messages().Len(); i++ {
		collectMessageDescriptors(out, msg.Messages().Get(i))
	}
	for i := 0; i < msg.Enums().Len(); i++ {
		enum := msg.Enums().Get(i)
		out.enums[string(enum.FullName())] = enum
	}
}

func (c *Catalog) LoadProtoDescriptorSetFiles(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	descriptors, err := LoadProtoDescriptorSetFiles(paths)
	if err != nil {
		return err
	}
	c.ProtoDescriptors = descriptors
	c.resolveProtoTypeCodes()
	return nil
}

func (c *Catalog) resolveProtoTypeCodes() {
	for _, table := range c.Tables {
		for _, col := range table.Columns {
			c.resolveProtoTypeCode(col.Type)
		}
	}
}

func (c *Catalog) resolveProtoTypeCode(spec *TypeSpec) {
	if spec == nil {
		return
	}
	switch spec.Code {
	case spannerpb.TypeCode_ARRAY:
		c.resolveProtoTypeCode(spec.ArrayElement)
	case spannerpb.TypeCode_STRUCT:
		for _, field := range spec.StructFields {
			c.resolveProtoTypeCode(field.Type)
		}
	case spannerpb.TypeCode_PROTO:
		if c.ProtoDescriptors != nil && c.ProtoDescriptors.enums[spec.ProtoTypeFQN] != nil {
			spec.Code = spannerpb.TypeCode_ENUM
		}
	}
}

func (c *Catalog) protoShadowStructType(fqn string) (*TypeSpec, error) {
	if c.ProtoDescriptors == nil {
		return nil, fmt.Errorf("proto descriptor set is required for %s", fqn)
	}
	if !c.ProtoTypes[fqn] {
		return nil, fmt.Errorf("proto type %s is not in the active proto bundle", fqn)
	}
	msg := c.ProtoDescriptors.messages[fqn]
	if msg == nil {
		return nil, fmt.Errorf("proto message %s not found in descriptor set", fqn)
	}
	return c.protoMessageShadowStructType(msg), nil
}

func (c *Catalog) protoMessageShadowStructType(msg protoreflect.MessageDescriptor) *TypeSpec {
	fields := msg.Fields()
	structFields := make([]StructField, 0, fields.Len())
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		spec, ok := c.protoFieldType(field)
		if !ok {
			continue
		}
		structFields = append(structFields, StructField{Name: string(field.Name()), Type: spec})
	}
	return &TypeSpec{Code: spannerpb.TypeCode_STRUCT, StructFields: structFields}
}

func (c *Catalog) protoFieldType(field protoreflect.FieldDescriptor) (*TypeSpec, bool) {
	elem, ok := c.protoSingularFieldType(field)
	if !ok {
		return nil, false
	}
	if field.IsList() {
		return &TypeSpec{Code: spannerpb.TypeCode_ARRAY, ArrayElement: elem}, true
	}
	return elem, true
}

func (c *Catalog) protoSingularFieldType(field protoreflect.FieldDescriptor) (*TypeSpec, bool) {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return &TypeSpec{Code: spannerpb.TypeCode_BOOL}, true
	case protoreflect.BytesKind:
		return &TypeSpec{Code: spannerpb.TypeCode_BYTES}, true
	case protoreflect.DoubleKind:
		return &TypeSpec{Code: spannerpb.TypeCode_FLOAT64}, true
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return &TypeSpec{Code: spannerpb.TypeCode_INT64}, true
	case protoreflect.StringKind:
		return &TypeSpec{Code: spannerpb.TypeCode_STRING}, true
	case protoreflect.MessageKind, protoreflect.GroupKind:
		msg := field.Message()
		if msg == nil || !c.ProtoTypes[string(msg.FullName())] {
			return nil, false
		}
		return c.protoMessageShadowStructType(msg), true
	case protoreflect.EnumKind:
		enum := field.Enum()
		if enum == nil || !c.ProtoTypes[string(enum.FullName())] {
			return nil, false
		}
		return &TypeSpec{Code: spannerpb.TypeCode_ENUM, ProtoTypeFQN: string(enum.FullName())}, true
	default:
		return nil, false
	}
}

func normalizeProtoTypeName(name string) string {
	return strings.Trim(strings.TrimPrefix(name, "."), "`")
}
