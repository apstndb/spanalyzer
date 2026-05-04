package spanalyzer

import (
	"testing"

	googlesql "github.com/goccy/go-googlesql"
)

func TestDefaultAnalyzerProductModeIsExternal(t *testing.T) {
	catalog, err := BuildGoogleSQLCatalogFromDDL("schema.sql", "", nil)
	if err != nil {
		t.Fatalf("BuildGoogleSQLCatalogFromDDL() error = %v", err)
	}
	assertProductMode(t, catalog, googlesql.ProductModeProductExternal)
}

func TestWithProductModeOverridesDefault(t *testing.T) {
	catalog, err := BuildGoogleSQLCatalogFromDDL(
		"schema.sql",
		"",
		nil,
		WithProductMode(googlesql.ProductModeProductInternal),
	)
	if err != nil {
		t.Fatalf("BuildGoogleSQLCatalogFromDDL() error = %v", err)
	}
	assertProductMode(t, catalog, googlesql.ProductModeProductInternal)
}

func TestSpannerAnalyzerDisablesPipeSyntaxFeature(t *testing.T) {
	catalog, err := BuildGoogleSQLCatalogFromDDL("schema.sql", "", nil)
	if err != nil {
		t.Fatalf("BuildGoogleSQLCatalogFromDDL() error = %v", err)
	}
	assertLanguageFeature(t, catalog.AnalyzerOptions, googlesql.LanguageFeatureFeaturePipes, false)
	assertLanguageFeature(t, catalog.AnalyzerOptions, googlesql.LanguageFeatureFeatureStatementWithPipeOperators, false)
}

func TestBigQueryAnalyzerEnablesPipeSyntaxFeature(t *testing.T) {
	catalog, err := BuildBigQueryGoogleSQLCatalogFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("BuildBigQueryGoogleSQLCatalogFromDDL() error = %v", err)
	}
	assertLanguageFeature(t, catalog.AnalyzerOptions, googlesql.LanguageFeatureFeaturePipes, true)
}

func TestSpannerAnalyzerRejectsPipeSyntax(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	if _, err := analyzer.RowTypeForStatement("SELECT 1 AS n |> SELECT n"); err == nil {
		t.Fatalf("RowTypeForStatement() error = nil, want pipe syntax error")
	}
}

func TestSpannerAnalyzerCanEnableMaximumDevelopmentLanguageFeatures(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDLWithOptions(
		"schema.sql",
		"",
		WithMaximumDevelopmentLanguageFeatures(true),
	)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDLWithOptions() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("SELECT 1 AS n |> SELECT n")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := rowType.Fields[0].Name, "n"; got != want {
		t.Fatalf("field name = %q, want %q", got, want)
	}
}

func assertProductMode(t *testing.T, catalog *GoogleSQLCatalog, want googlesql.ProductMode) {
	t.Helper()
	lang, err := catalog.AnalyzerOptions.Language()
	if err != nil {
		t.Fatalf("AnalyzerOptions.Language() error = %v", err)
	}
	got, err := lang.ProductMode()
	if err != nil {
		t.Fatalf("LanguageOptions.ProductMode() error = %v", err)
	}
	if got != want {
		t.Fatalf("ProductMode() = %s, want %s", got, want)
	}
}

func assertLanguageFeature(t *testing.T, opts *googlesql.AnalyzerOptions, feature googlesql.LanguageFeature, want bool) {
	t.Helper()
	lang, err := opts.Language()
	if err != nil {
		t.Fatalf("AnalyzerOptions.Language() error = %v", err)
	}
	got, err := lang.LanguageFeatureEnabled(feature)
	if err != nil {
		t.Fatalf("LanguageFeatureEnabled(%s) error = %v", feature, err)
	}
	if got != want {
		t.Fatalf("LanguageFeatureEnabled(%s) = %t, want %t", feature, got, want)
	}
}
