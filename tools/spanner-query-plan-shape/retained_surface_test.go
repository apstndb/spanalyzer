package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type retainedSurfaceManifest struct {
	Version string `json:"version"`
	Queries []struct {
		Label    string            `json:"label"`
		Patterns []json.RawMessage `json:"patterns"`
	} `json:"queries"`
	ExpectedQueryErrors []struct {
		Label    string `json:"label"`
		Contains string `json:"contains"`
	} `json:"expected_query_errors"`
}

func TestStatementSurfaceCasesAndManifest(t *testing.T) {
	assertRetainedSurface(t, "statement_surface", "statement-surface/", "statement_surface_expectations.json", 3, 7)

	ddls, err := loadDDLs("statement_surface", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ddls) != 0 {
		t.Fatalf("statement surface DDL count = %d, want 0", len(ddls))
	}
}

func TestDMLCasesAndManifest(t *testing.T) {
	assertRetainedSurface(t, "dml", "dml/", "dml_expectations.json", 28, 1)
}

func assertRetainedSurface(t *testing.T, selector, prefix, manifestName string, wantPlans, wantErrors int) {
	t.Helper()
	queries, err := loadQueries(selector, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool, len(queries))
	for _, query := range queries {
		if !strings.HasPrefix(query.Label, prefix) {
			t.Errorf("selector %s label %q lacks prefix %q", selector, query.Label, prefix)
		}
		if seen[query.Label] {
			t.Errorf("selector %s has duplicate label %q", selector, query.Label)
		}
		seen[query.Label] = true
	}

	data, err := os.ReadFile(filepath.Join("testdata", manifestName))
	if err != nil {
		t.Fatal(err)
	}
	var manifest retainedSurfaceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "v0alpha1" || len(manifest.Queries) != wantPlans || len(manifest.ExpectedQueryErrors) != wantErrors {
		t.Fatalf("%s manifest summary = version %q, %d plans, %d errors", selector, manifest.Version, len(manifest.Queries), len(manifest.ExpectedQueryErrors))
	}
	manifestLabels := make(map[string]bool, len(queries))
	for _, expectation := range manifest.Queries {
		if len(expectation.Patterns) == 0 {
			t.Errorf("manifest label %q has no patterns", expectation.Label)
		}
		manifestLabels[expectation.Label] = true
	}
	for _, expectation := range manifest.ExpectedQueryErrors {
		if expectation.Contains == "" {
			t.Errorf("manifest error label %q has empty substring", expectation.Label)
		}
		manifestLabels[expectation.Label] = true
	}
	for label := range seen {
		if !manifestLabels[label] {
			t.Errorf("selector label %q absent from manifest", label)
		}
	}
	for label := range manifestLabels {
		if !seen[label] {
			t.Errorf("manifest label %q absent from selector", label)
		}
	}
}
