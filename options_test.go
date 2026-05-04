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
