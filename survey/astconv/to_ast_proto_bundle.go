package astconv

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// toProtoBundleDDL generates CREATE PROTO BUNDLE from SCHEMATA.PROTO_BUNDLE.
// Explicitly populated ProtoBundleTypes still take precedence, but when absent
// we decode SCHEMATA.PROTO_BUNDLE and reconstruct the bundled type names.
func (s *Schema) toProtoBundleDDL() ([]ast.DDL, error) {
	typeNames, err := s.protoBundleTypeNames()
	if err != nil {
		return nil, err
	}
	if len(typeNames) == 0 {
		return nil, nil
	}

	types := make([]*ast.NamedType, 0, len(typeNames))
	for _, typeName := range typeNames {
		types = append(types, namedTypeFromProtoTypeName(typeName))
	}

	return []ast.DDL{
		&ast.CreateProtoBundle{
			Types: &ast.ProtoBundleTypes{
				Types: types,
			},
		},
	}, nil
}

func (s *Schema) protoBundleTypeNames() ([]string, error) {
	if len(s.ProtoBundleTypes) > 0 {
		return s.ProtoBundleTypes, nil
	}

	seen := make(map[string]struct{})
	var typeNames []string
	for _, schema := range s.Schemata {
		if len(schema.ProtoBundle) == 0 {
			continue
		}
		if isSystemSchemaName(schema.SchemaName) {
			continue
		}
		names, err := extractProtoBundleTypes(schema.ProtoBundle)
		if err != nil {
			return nil, fmt.Errorf("decode proto bundle for schema %q: %w", schema.SchemaName, err)
		}
		for _, name := range names {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			typeNames = append(typeNames, name)
		}
	}

	sort.Strings(typeNames)
	return typeNames, nil
}

func extractProtoBundleTypes(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var descriptors descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(raw, &descriptors); err != nil {
		return nil, err
	}

	var typeNames []string
	for _, file := range descriptors.GetFile() {
		pkg := file.GetPackage()
		for _, enum := range file.GetEnumType() {
			typeNames = append(typeNames, qualifyProtoTypeName(pkg, enum.GetName()))
		}
		for _, message := range file.GetMessageType() {
			typeNames = append(typeNames, extractDescriptorTypeNames(pkg, nil, message)...)
		}
	}

	return typeNames, nil
}

func extractDescriptorTypeNames(pkg string, parents []string, message *descriptorpb.DescriptorProto) []string {
	if message == nil {
		return nil
	}

	currentPath := append(append([]string(nil), parents...), message.GetName())
	var typeNames []string
	typeNames = append(typeNames, qualifyProtoTypeName(pkg, strings.Join(currentPath, ".")))

	for _, enum := range message.GetEnumType() {
		typeNames = append(typeNames, qualifyProtoTypeName(pkg, strings.Join(append(currentPath, enum.GetName()), ".")))
	}
	for _, nested := range message.GetNestedType() {
		typeNames = append(typeNames, extractDescriptorTypeNames(pkg, currentPath, nested)...)
	}

	return typeNames
}

func qualifyProtoTypeName(pkg, name string) string {
	if pkg == "" {
		return name
	}
	if name == "" {
		return pkg
	}
	return pkg + "." + name
}

func namedTypeFromProtoTypeName(typeName string) *ast.NamedType {
	if strings.HasPrefix(typeName, "`") && strings.HasSuffix(typeName, "`") {
		return &ast.NamedType{Path: []*ast.Ident{ident(strings.Trim(typeName, "`"))}}
	}
	parts := strings.Split(typeName, ".")
	for _, part := range parts {
		if token.QuoteSQLIdent(part) != part {
			return &ast.NamedType{Path: []*ast.Ident{ident(typeName)}}
		}
	}
	return &ast.NamedType{Path: path(parts...).Idents}
}
