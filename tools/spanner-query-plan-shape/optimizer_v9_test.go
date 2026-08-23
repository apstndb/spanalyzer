package main

import (
	"strings"
	"testing"
)

func TestLoadQueriesOptimizerV9MatchesManifest(t *testing.T) {
	queries, err := loadQueries("optimizer_v9", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "optimizer_v9", err)
	}
	if got, want := len(queries), 7; got != want {
		t.Fatalf("optimizer v9 query count = %d, want %d", got, want)
	}
	for _, query := range queries {
		if !strings.HasPrefix(query.Label, "optimizer-v9/") {
			t.Errorf("optimizer v9 label %q has wrong prefix", query.Label)
		}
	}

	manifest := loadExpectationManifest(t, "optimizer_v9_expectations.json")
	if got, want := len(manifest.Queries), 6; got != want {
		t.Errorf("optimizer v9 manifest plans = %d, want %d", got, want)
	}
	if got, want := len(manifest.ExpectedQueryErrors), 1; got != want {
		t.Errorf("optimizer v9 manifest errors = %d, want %d", got, want)
	}
}
