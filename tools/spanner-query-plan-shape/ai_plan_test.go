package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAIPlanCasesAndManifest(t *testing.T) {
	queries, err := loadQueries("ai_plan", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(queries), 6; got != want {
		t.Fatalf("AI PLAN query count = %d, want %d", got, want)
	}
	seen := make(map[string]string, len(queries))
	for _, query := range queries {
		if _, duplicate := seen[query.Label]; duplicate {
			t.Errorf("duplicate AI PLAN label %q", query.Label)
		}
		seen[query.Label] = query.SQL
	}
	for label, function := range map[string]string{
		"ai-plan/classify-projection": "AI.CLASSIFY(",
		"ai-plan/if-filter":           "AI.IF(",
		"ai-plan/score-order-limit":   "AI.SCORE(",
	} {
		if !strings.Contains(seen[label], function) {
			t.Errorf("%s missing %s", label, function)
		}
	}

	data, err := os.ReadFile(filepath.Join("testdata", "ai_plan_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
		ExpectedQueryErrors []json.RawMessage `json:"expected_query_errors"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "v0alpha1" || len(manifest.Queries) != len(queries) || len(manifest.ExpectedQueryErrors) != 0 {
		t.Fatalf("AI manifest summary = version %q, %d queries, %d errors", manifest.Version, len(manifest.Queries), len(manifest.ExpectedQueryErrors))
	}
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("AI manifest label %q absent from selector", expectation.Label)
		}
		if len(expectation.Patterns) == 0 {
			t.Errorf("AI manifest label %q has no patterns", expectation.Label)
		}
		delete(seen, expectation.Label)
	}
	if len(seen) != 0 {
		t.Errorf("AI selector labels absent from manifest: %v", seen)
	}
}
