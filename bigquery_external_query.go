package spanalyzer

import (
	"fmt"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	googlesql "github.com/goccy/go-googlesql"
)

func (c *BigQueryGoogleSQLCatalog) addExternalQueryTVF() error {
	if c.externalQueryTVFRegistered {
		return nil
	}
	stringType, err := c.TypeFactory.GetString()
	if err != nil {
		return err
	}
	stringArg, err := newFunctionArgumentType(stringType)
	if err != nil {
		return err
	}
	resultArg, err := googlesql.NewFunctionArgumentTypeAnyRelation()
	if err != nil {
		return err
	}
	sigOpts, err := googlesql.NewFunctionSignatureOptions()
	if err != nil {
		return err
	}
	twoArgSig, err := googlesql.NewFunctionSignature2(resultArg, []*googlesql.FunctionArgumentType{
		stringArg,
		stringArg,
	}, 0, sigOpts)
	if err != nil {
		return err
	}
	threeArgSig, err := googlesql.NewFunctionSignature2(resultArg, []*googlesql.FunctionArgumentType{
		stringArg,
		stringArg,
		stringArg,
	}, 0, sigOpts)
	if err != nil {
		return err
	}
	tvf, err := googlesql.NewTableValuedFunctionFromImpl(
		&bigQueryExternalQueryTVF{catalog: c},
		[]string{"EXTERNAL_QUERY"},
		"BigQuery",
		[]*googlesql.FunctionSignature{twoArgSig, threeArgSig},
		&googlesql.TableValuedFunctionOptions{},
	)
	if err != nil {
		return err
	}
	if err := c.SimpleCatalog.AddOwnedTableValuedFunction(tvf); err != nil {
		return err
	}
	c.externalQueryTVFRegistered = true
	return nil
}

type bigQueryExternalQueryTVF struct {
	googlesql.TableValuedFunctionCallbackDefaults

	catalog *BigQueryGoogleSQLCatalog
}

func (f *bigQueryExternalQueryTVF) Resolve(
	_ *googlesql.AnalyzerOptions,
	actualArguments []*googlesql.TVFInputArgumentType,
	_ *googlesql.FunctionSignature,
	_ googlesql.CatalogNode,
	typeFactory *googlesql.TypeFactory,
) (*googlesql.TVFSignature, error) {
	if len(actualArguments) < 2 || len(actualArguments) > 3 {
		return nil, fmt.Errorf("EXTERNAL_QUERY requires 2 or 3 arguments, got %d", len(actualArguments))
	}
	if len(actualArguments) == 3 {
		return nil, fmt.Errorf("EXTERNAL_QUERY options argument is currently not supported for static analysis")
	}

	connectionID, err := extractStringArgument(actualArguments[0])
	if err != nil {
		return nil, fmt.Errorf("EXTERNAL_QUERY connection argument: %w", err)
	}
	spannerSQL, err := extractStringArgument(actualArguments[1])
	if err != nil {
		return nil, fmt.Errorf("EXTERNAL_QUERY SQL argument: %w", err)
	}

	rowType, err := f.findRowType(connectionID, spannerSQL)
	if err != nil {
		return nil, err
	}
	resultSchema, err := bigQueryTVFRelationFromSpannerRowType(typeFactory, rowType)
	if err != nil {
		return nil, err
	}
	return googlesql.NewTVFSignature(actualArguments, resultSchema, nil)
}

func (f *bigQueryExternalQueryTVF) findRowType(connectionID, spannerSQL string) (*spannerpb.StructType, error) {
	if f.catalog == nil {
		return nil, fmt.Errorf("EXTERNAL_QUERY callback has no BigQuery catalog")
	}
	if f.catalog.externalQueryRowTypes == nil || f.catalog.externalQueryRowTypes[connectionID] == nil || f.catalog.externalQueryRowTypes[connectionID][spannerSQL] == nil {
		return nil, fmt.Errorf("no prepared Spanner row type for connection %q and SQL %q", connectionID, spannerSQL)
	}
	return f.catalog.externalQueryRowTypes[connectionID][spannerSQL], nil
}

func extractStringArgument(arg *googlesql.TVFInputArgumentType) (string, error) {
	isScalar, err := arg.IsScalar()
	if err != nil {
		return "", err
	}
	if !isScalar {
		return "", fmt.Errorf("expected scalar argument")
	}
	expr, err := arg.ScalarExpr()
	if err != nil {
		return "", err
	}
	literal, ok := expr.(*googlesql.ResolvedLiteral)
	if !ok {
		return "", fmt.Errorf("expected string literal, got non-literal expression")
	}
	val, err := literal.Value()
	if err != nil {
		return "", err
	}
	isNull, err := val.IsNull()
	if err != nil {
		return "", err
	}
	if isNull {
		return "", fmt.Errorf("expected string literal, got NULL")
	}
	return val.StringValue()
}

func bigQueryTVFRelationFromSpannerRowType(typeFactory *googlesql.TypeFactory, rowType *spannerpb.StructType) (*googlesql.TVFRelation, error) {
	if rowType == nil {
		return nil, fmt.Errorf("nil Spanner row type")
	}
	columns := make([]*googlesql.TVFSchemaColumn, 0, len(rowType.Fields))
	for _, field := range rowType.Fields {
		if _, err := bigQuerySQLTypeFromSpannerType(field.Type); err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}
		spec, err := TypeSpecFromSpannerPB(field.Type)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}
		typ, err := typeSpecToGoogleSQLTypeWithProto(typeFactory, spec, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}
		columns = append(columns, &googlesql.TVFSchemaColumn{Name: field.Name, Type_: typ})
	}
	return googlesql.NewTVFRelation(columns)
}

func bigQuerySQLTypeFromSpannerType(t *spannerpb.Type) (string, error) {
	if t == nil {
		return "", fmt.Errorf("nil Spanner type")
	}
	switch t.Code {
	case spannerpb.TypeCode_BOOL:
		return "BOOL", nil
	case spannerpb.TypeCode_INT64:
		return "INT64", nil
	case spannerpb.TypeCode_FLOAT32, spannerpb.TypeCode_FLOAT64:
		return "FLOAT64", nil
	case spannerpb.TypeCode_TIMESTAMP:
		return "TIMESTAMP", nil
	case spannerpb.TypeCode_DATE:
		return "DATE", nil
	case spannerpb.TypeCode_STRING:
		return "STRING", nil
	case spannerpb.TypeCode_BYTES:
		return "BYTES", nil
	case spannerpb.TypeCode_NUMERIC:
		return "NUMERIC", nil
	case spannerpb.TypeCode_JSON:
		return "JSON", nil
	case spannerpb.TypeCode_ARRAY:
		elem, err := bigQuerySQLTypeFromSpannerType(t.ArrayElementType)
		if err != nil {
			return "", err
		}
		return "ARRAY<" + elem + ">", nil
	case spannerpb.TypeCode_STRUCT:
		return "", fmt.Errorf("STRUCT is unsupported for Spanner federated query output")
	case spannerpb.TypeCode_PROTO:
		return "", fmt.Errorf("PROTO is unsupported for Spanner federated query output")
	case spannerpb.TypeCode_ENUM:
		return "", fmt.Errorf("ENUM is unsupported for Spanner federated query output")
	default:
		return "", fmt.Errorf("unsupported Spanner type %s for Spanner federated query output", t.Code)
	}
}
