package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNgramSearchCasesAndManifest(t *testing.T) {
	ddls, err := loadDDLs("ngram_search", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(ddls), 3; got != want {
		t.Fatalf("ngram DDL count = %d, want %d", got, want)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{
		"TOKENIZE_SUBSTRING", "TOKENIZE_NGRAMS", "LOWER(AlbumTitle)",
		"NgramAlbumsFuzzyIndex", "NgramAlbumsPatternIndex", "STORING (AlbumTitle)",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("ngram schema missing %q", want)
		}
	}

	queries, err := loadQueries("ngram_search", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(queries), 12; got != want {
		t.Fatalf("ngram query count = %d, want %d", got, want)
	}
	seen := make(map[string]queryCase, len(queries))
	for _, query := range queries {
		if _, duplicate := seen[query.Label]; duplicate {
			t.Errorf("duplicate ngram label %q", query.Label)
		}
		seen[query.Label] = query
	}
	if got := seen["ngram-search/pattern/like-parameter/search-index"].Params["pattern"]; got != "%999%" {
		t.Errorf("LIKE parameter = %v, want %%999%%", got)
	}
	for _, predicate := range []string{"like-contains-literal", "starts-with-literal", "ends-with-literal", "regexp-contains-literal"} {
		indexed := seen["ngram-search/pattern/"+predicate+"/search-index"].SQL
		base := seen["ngram-search/pattern/"+predicate+"/base-table"].SQL
		if got := strings.Replace(indexed, "NgramAlbumsPatternIndex", "_BASE_TABLE", 1); got != base {
			t.Errorf("%s pair differs beyond FORCE_INDEX:\nindexed: %s\nbase: %s", predicate, indexed, base)
		}
	}

	data, err := os.ReadFile(filepath.Join("testdata", "ngram_search_expectations.json"))
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
	if manifest.Version != "v0alpha1" {
		t.Errorf("manifest version = %q", manifest.Version)
	}
	if len(manifest.ExpectedQueryErrors) != 0 {
		t.Errorf("default ngram manifest has %d expected errors", len(manifest.ExpectedQueryErrors))
	}
	if got, want := len(manifest.Queries), len(queries); got != want {
		t.Fatalf("manifest query count = %d, want %d", got, want)
	}
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("manifest label %q is absent from selector", expectation.Label)
		}
		if len(expectation.Patterns) == 0 {
			t.Errorf("manifest label %q has no patterns", expectation.Label)
		}
		delete(seen, expectation.Label)
	}
	if len(seen) != 0 {
		t.Errorf("selector has %d labels absent from manifest: %v", len(seen), seen)
	}
}
