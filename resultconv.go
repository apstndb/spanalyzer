package spanalyzer

import (
	"fmt"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	googlesql "github.com/goccy/go-googlesql"
)

func RowTypeFromAnalyzerOutput(out *googlesql.AnalyzerOutput, schema *Catalog) (*spannerpb.StructType, error) {
	stmt, err := out.ResolvedStatement()
	if err != nil {
		return nil, err
	}
	switch s := stmt.(type) {
	case *googlesql.ResolvedQueryStmt:
		return RowTypeFromResolvedQuery(s, schema)
	case *googlesql.ResolvedInsertStmt:
		returning, err := s.Returning()
		if err != nil || returning == nil {
			return nil, ErrStatementHasNoRowType
		}
		return rowTypeFromReturningClause(returning, schema)
	case *googlesql.ResolvedUpdateStmt:
		returning, err := s.Returning()
		if err != nil || returning == nil {
			return nil, ErrStatementHasNoRowType
		}
		return rowTypeFromReturningClause(returning, schema)
	case *googlesql.ResolvedDeleteStmt:
		returning, err := s.Returning()
		if err != nil || returning == nil {
			return nil, ErrStatementHasNoRowType
		}
		return rowTypeFromReturningClause(returning, schema)
	default:
		return nil, ErrStatementHasNoRowType
	}
}

func rowTypeFromReturningClause(returning *googlesql.ResolvedReturningClause, schema *Catalog) (*spannerpb.StructType, error) {
	n, err := returning.OutputColumnListSize()
	if err != nil {
		return nil, err
	}
	fields := make([]*spannerpb.StructType_Field, 0, n)
	for i := int32(0); i < n; i++ {
		outCol, err := returning.OutputColumnList2(i)
		if err != nil {
			return nil, err
		}
		name, err := outCol.Name()
		if err != nil {
			return nil, err
		}
		resolvedCol, err := outCol.Column()
		if err != nil {
			return nil, err
		}
		gsType, err := resolvedCol.Type()
		if err != nil {
			return nil, err
		}
		pbType, ok, err := directProtoColumnType(schema, resolvedCol)
		if err != nil {
			return nil, err
		}
		if !ok {
			pbType, err = SpannerTypeFromGoogleSQLType(gsType)
		}
		if err != nil {
			return nil, err
		}
		fields = append(fields, &spannerpb.StructType_Field{Name: name, Type: pbType})
	}
	return &spannerpb.StructType{Fields: fields}, nil
}

func RowTypeFromResolvedQuery(query *googlesql.ResolvedQueryStmt, schema *Catalog) (*spannerpb.StructType, error) {
	n, err := query.OutputColumnListSize()
	if err != nil {
		return nil, err
	}
	fields := make([]*spannerpb.StructType_Field, 0, n)
	for i := int32(0); i < n; i++ {
		outCol, err := query.OutputColumnList2(i)
		if err != nil {
			return nil, err
		}
		name, err := outCol.Name()
		if err != nil {
			return nil, err
		}
		resolvedCol, err := outCol.Column()
		if err != nil {
			return nil, err
		}
		gsType, err := resolvedCol.Type()
		if err != nil {
			return nil, err
		}
		pbType, ok, err := directProtoColumnType(schema, resolvedCol)
		if err != nil {
			return nil, err
		}
		if !ok {
			pbType, err = SpannerTypeFromGoogleSQLType(gsType)
		}
		if err != nil {
			return nil, err
		}
		fields = append(fields, &spannerpb.StructType_Field{Name: name, Type: pbType})
	}
	return &spannerpb.StructType{Fields: fields}, nil
}

func TypeFromAnalyzerOutput(out *googlesql.AnalyzerOutput) (*spannerpb.Type, error) {
	expr, err := out.ResolvedExpr()
	if err != nil {
		return nil, err
	}
	typ, err := expr.Type()
	if err != nil {
		return nil, err
	}
	return SpannerTypeFromGoogleSQLType(typ)
}

func directProtoColumnType(schema *Catalog, col *googlesql.ResolvedColumn) (*spannerpb.Type, bool, error) {
	if schema == nil {
		return nil, false, nil
	}
	tableName, err := col.TableName()
	if err != nil {
		return nil, false, err
	}
	columnName, err := col.Name()
	if err != nil {
		return nil, false, err
	}
	table := schema.Tables[tableName]
	if table == nil {
		return nil, false, nil
	}
	c, _ := table.Column(columnName)
	if c == nil || c.Type == nil {
		return nil, false, nil
	}
	switch c.Type.Code {
	case spannerpb.TypeCode_PROTO, spannerpb.TypeCode_ENUM:
		t, err := c.Type.SpannerPB()
		return t, err == nil, err
	default:
		return nil, false, nil
	}
}

func TypeSpecFromSpannerPB(t *spannerpb.Type) (*TypeSpec, error) {
	if t == nil {
		return nil, fmt.Errorf("nil Spanner protobuf type")
	}
	switch t.Code {
	case spannerpb.TypeCode_ARRAY:
		elem, err := TypeSpecFromSpannerPB(t.ArrayElementType)
		if err != nil {
			return nil, err
		}
		return &TypeSpec{Code: spannerpb.TypeCode_ARRAY, ArrayElement: elem}, nil
	case spannerpb.TypeCode_STRUCT:
		fields := make([]StructField, 0, len(t.GetStructType().GetFields()))
		for _, field := range t.GetStructType().GetFields() {
			spec, err := TypeSpecFromSpannerPB(field.Type)
			if err != nil {
				return nil, err
			}
			fields = append(fields, StructField{Name: field.Name, Type: spec})
		}
		return &TypeSpec{Code: spannerpb.TypeCode_STRUCT, StructFields: fields}, nil
	case spannerpb.TypeCode_PROTO, spannerpb.TypeCode_ENUM:
		return &TypeSpec{Code: t.Code, ProtoTypeFQN: t.ProtoTypeFqn}, nil
	default:
		return &TypeSpec{Code: t.Code}, nil
	}
}
