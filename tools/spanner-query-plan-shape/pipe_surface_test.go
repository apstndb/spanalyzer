package main

import (
	"strings"
	"testing"
)

func TestLoadQueriesPipeSurfaceMatchesManifest(t *testing.T) {
	queries, err := loadQueries("pipe_surface", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "pipe_surface", err)
	}
	if got, want := len(queries), 15; got != want {
		t.Fatalf("pipe surface query count = %d, want %d", got, want)
	}
	seen := make(map[string]struct{}, len(queries))
	for _, query := range queries {
		if !strings.HasPrefix(query.Label, "pipe-surface/accepted/") {
			t.Errorf("pipe surface label %q has wrong prefix", query.Label)
		}
		if _, duplicate := seen[query.Label]; duplicate {
			t.Errorf("duplicate pipe surface label %q", query.Label)
		}
		seen[query.Label] = struct{}{}
	}

	manifest := loadExpectationManifest(t, "pipe_surface_expectations.json")
	if got, want := len(manifest.Queries), len(queries); got != want {
		t.Errorf("pipe surface manifest plans = %d, want %d", got, want)
	}
	if got := len(manifest.ExpectedQueryErrors); got != 0 {
		t.Errorf("pipe surface manifest errors = %d, want 0", got)
	}
}
