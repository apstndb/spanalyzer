package spanalyzer

import (
	"errors"
	"fmt"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	googlesql "github.com/goccy/go-googlesql"
)

var ErrStatementHasNoRowType = errors.New("statement has no row type")

type Analyzer struct {
	catalog   *Catalog
	googleSQL *GoogleSQLCatalog
	helper    *GoogleSQLHelper
}

func NewAnalyzer(schema *Catalog) (*Analyzer, error) {
	return NewAnalyzerWithOptions(schema)
}

func NewAnalyzerWithOptions(schema *Catalog, options ...AnalyzerOption) (*Analyzer, error) {
	googleSQLCatalog, err := BuildGoogleSQLCatalogFromSpannerCatalog(schema, options...)
	if err != nil {
		return nil, err
	}
	return NewAnalyzerFromGoogleSQLCatalog(googleSQLCatalog)
}

func NewAnalyzerFromGoogleSQLCatalog(googleSQLCatalog *GoogleSQLCatalog) (*Analyzer, error) {
	if googleSQLCatalog == nil {
		return nil, fmt.Errorf("nil GoogleSQL catalog")
	}
	return &Analyzer{
		catalog:   googleSQLCatalog.SpannerCatalog,
		googleSQL: googleSQLCatalog,
		helper:    googleSQLCatalog.Helper(),
	}, nil
}

func NewAnalyzerFromDDL(path, ddlSQL string) (*Analyzer, error) {
	return NewAnalyzerFromDDLWithOptions(path, ddlSQL)
}

func NewAnalyzerFromDDLWithOptions(path, ddlSQL string, options ...AnalyzerOption) (*Analyzer, error) {
	return NewAnalyzerFromDDLWithProtoDescriptorFilesAndOptions(path, ddlSQL, nil, options...)
}

func NewAnalyzerFromDDLWithProtoDescriptorFiles(path, ddlSQL string, protoDescriptorPaths []string) (*Analyzer, error) {
	return NewAnalyzerFromDDLWithProtoDescriptorFilesAndOptions(path, ddlSQL, protoDescriptorPaths)
}

func NewAnalyzerFromDDLWithProtoDescriptorFilesAndOptions(path, ddlSQL string, protoDescriptorPaths []string, options ...AnalyzerOption) (*Analyzer, error) {
	googleSQLCatalog, err := BuildGoogleSQLCatalogFromDDL(path, ddlSQL, protoDescriptorPaths, options...)
	if err != nil {
		return nil, err
	}
	return NewAnalyzerFromGoogleSQLCatalog(googleSQLCatalog)
}

func (a *Analyzer) RowTypeForStatement(sql string) (*spannerpb.StructType, error) {
	out, err := a.helper.AnalyzeStatement(sql)
	if err != nil {
		return nil, err
	}
	return RowTypeFromAnalyzerOutput(out, a.catalog)
}

func (a *Analyzer) RowTypeForExpression(sql string) (*spannerpb.StructType, error) {
	typ, err := a.TypeForExpression(sql)
	if err != nil {
		return nil, err
	}
	return &spannerpb.StructType{
		Fields: []*spannerpb.StructType_Field{
			{Name: "expression", Type: typ},
		},
	}, nil
}

func (a *Analyzer) TypeForExpression(sql string) (*spannerpb.Type, error) {
	out, err := a.helper.AnalyzeExpression(sql)
	if err != nil {
		return nil, err
	}
	return TypeFromAnalyzerOutput(out)
}

func (a *Analyzer) ParseDebugString(sqlMode, sql string) (string, error) {
	return a.helper.ParseDebugString(sqlMode, sql)
}

func (a *Analyzer) Unparse(sqlMode, sql string) (string, error) {
	return a.helper.Unparse(sqlMode, sql)
}

func (a *Analyzer) ResolvedASTDebugString(sqlMode, sql string) (string, error) {
	return a.helper.ResolvedASTDebugString(sqlMode, sql)
}

func (a *Analyzer) AddQueryParameter(name string, spec *TypeSpec) error {
	typ, err := a.googleSQL.TypeSpecToGoogleSQLType(spec)
	if err != nil {
		return err
	}
	if err := a.googleSQL.AnalyzerOptions.SetParameterMode(googlesql.ParameterModeParameterNamed); err != nil {
		return err
	}
	return a.googleSQL.AnalyzerOptions.AddQueryParameter(name, typ)
}

func (a *Analyzer) AddPositionalQueryParameter(spec *TypeSpec) error {
	typ, err := a.googleSQL.TypeSpecToGoogleSQLType(spec)
	if err != nil {
		return err
	}
	if err := a.googleSQL.AnalyzerOptions.SetParameterMode(googlesql.ParameterModeParameterPositional); err != nil {
		return err
	}
	return a.googleSQL.AnalyzerOptions.AddPositionalQueryParameter(typ)
}
