package main

import (
	"strings"
	"testing"
)

func TestNgramSearchCases(t *testing.T) {
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
}
