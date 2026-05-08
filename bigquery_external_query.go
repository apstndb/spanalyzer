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
	rowType, err := f.nextPreparedRowType()
	if err != nil {
		return nil, err
	}
	resultSchema, err := bigQueryTVFRelationFromSpannerRowType(typeFactory, rowType)
	if err != nil {
		return nil, err
	}
	return googlesql.NewTVFSignature(actualArguments, resultSchema, nil)
}

func (f *bigQueryExternalQueryTVF) nextPreparedRowType() (*spannerpb.StructType, error) {
	if f.catalog == nil {
		return nil, fmt.Errorf("EXTERNAL_QUERY callback has no BigQuery catalog")
	}
	i := f.catalog.externalQueryPendingIndex
	if i < 0 || i >= len(f.catalog.externalQueryPendingRowTypes) {
		return nil, fmt.Errorf("EXTERNAL_QUERY callback has no prepared Spanner row type")
	}
	rowType := f.catalog.externalQueryPendingRowTypes[i]
	f.catalog.externalQueryPendingIndex++
	if rowType == nil {
		return nil, fmt.Errorf("EXTERNAL_QUERY callback prepared nil Spanner row type")
	}
	return rowType, nil
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
