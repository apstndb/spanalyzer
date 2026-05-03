package spanalyzer

import (
	"fmt"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func ParseTypeSpec(path, typeSQL string) (*TypeSpec, error) {
	typ, err := memefish.ParseType(path, typeSQL)
	if err != nil {
		return parseSchemaTypeSpecFromColumn(path, typeSQL)
	}
	spec, err := typeToTypeSpec(typ)
	if err != nil {
		return parseSchemaTypeSpecFromColumn(path, typeSQL)
	}
	return spec, nil
}

func parseSchemaTypeSpecFromColumn(path, typeSQL string) (*TypeSpec, error) {
	ddl := "CREATE TABLE _ParamType (_Key INT64 NOT NULL, _Value " + typeSQL + ") PRIMARY KEY(_Key);"
	catalog, err := BuildSchemaCatalog(path, ddl)
	if err != nil {
		return nil, err
	}
	table := catalog.Tables["_ParamType"]
	if table == nil {
		return nil, fmt.Errorf("could not parse type %q", typeSQL)
	}
	column, _ := table.Column("_Value")
	if column == nil {
		return nil, fmt.Errorf("could not parse type %q", typeSQL)
	}
	return column.Type, nil
}

func typeToTypeSpec(t ast.Type) (*TypeSpec, error) {
	switch t := t.(type) {
	case *ast.SimpleType:
		return scalarTypeToTypeSpec(t.Name, nil, false)
	case *ast.ArrayType:
		elem, err := typeToTypeSpec(t.Item)
		if err != nil {
			return nil, err
		}
		return &TypeSpec{Code: spannerpb.TypeCode_ARRAY, ArrayElement: elem}, nil
	case *ast.StructType:
		fields := make([]StructField, 0, len(t.Fields))
		for _, field := range t.Fields {
			spec, err := typeToTypeSpec(field.Type)
			if err != nil {
				return nil, err
			}
			name := ""
			if field.Ident != nil {
				name = field.Ident.Name
			}
			fields = append(fields, StructField{Name: name, Type: spec})
		}
		return &TypeSpec{Code: spannerpb.TypeCode_STRUCT, StructFields: fields}, nil
	case *ast.NamedType:
		return &TypeSpec{Code: spannerpb.TypeCode_PROTO, ProtoTypeFQN: normalizeProtoTypeName(identPathString(t.Path))}, nil
	default:
		return nil, fmt.Errorf("unsupported type syntax %T", t)
	}
}
