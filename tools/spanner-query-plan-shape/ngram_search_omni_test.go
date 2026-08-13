//go:build integration && omni

package main

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestIntegrationNgramSearchVersionMatrixOnOmni(t *testing.T) {
	ddls, err := parseBuiltInDDLs("ngram-search-schema.sql", ngramSearchDDL)
	if err != nil {
		t.Fatal(err)
	}
	clients := openOmniClients(t, ddls)
	cases := queryCasesByLabel(t, ngramSearchQueries)

	baseLabels := []string{
		"ngram-search/pattern/like-contains-literal/base-table",
		"ngram-search/pattern/starts-with-literal/base-table",
		"ngram-search/pattern/ends-with-literal/base-table",
		"ngram-search/pattern/regexp-contains-literal/base-table",
	}
	acceleratedLabels := []string{
		"ngram-search/pattern/like-contains-literal/search-index",
		"ngram-search/pattern/starts-with-literal/search-index",
		"ngram-search/pattern/ends-with-literal/search-index",
		"ngram-search/pattern/regexp-contains-literal/search-index",
	}
	fuzzyLabels := []string{
		"ngram-search/fuzzy/search-only",
		"ngram-search/fuzzy/score-limit",
	}
	boundaryLabels := []string{
		"ngram-search/pattern/like-parameter/search-index",
		"ngram-search/pattern/like-too-short/search-index",
	}

	for version := 1; version <= 8; version++ {
		for _, label := range baseLabels {
			plan, err := analyzeVersionedSearchGraphPlan(t, clients.Client, cases[label], version)
			if err != nil {
				t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", label, version, err)
			}
			assertNgramBaseControl(t, plan)
		}
		for _, labels := range [][]string{fuzzyLabels, acceleratedLabels, boundaryLabels} {
			for _, label := range labels {
				plan, err := analyzeVersionedSearchGraphPlan(t, clients.Client, cases[label], version)
				if version <= 4 {
					if err == nil || !strings.Contains(err.Error(), "only supported in queries with query optimizer version 6 or above") {
						t.Errorf("AnalyzeQuery(%s, v%d) error = %v, want version capability error", label, version, err)
					}
					continue
				}
				if err != nil {
					t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", label, version, err)
				}
				switch {
				case strings.HasPrefix(label, "ngram-search/fuzzy/"):
					assertNgramFuzzyPlan(t, plan, strings.HasSuffix(label, "/score-limit"))
				case strings.Contains(label, "-literal/search-index") && !strings.Contains(label, "too-short"):
					assertNgramAcceleratedPattern(t, plan)
				default:
					assertNgramIneligiblePattern(t, plan)
				}
			}
		}
	}
}

func assertNgramBaseControl(t *testing.T, plan *spannerpb.QueryPlan) {
	t.Helper()
	if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{"scan_target": "NgramAlbums", "scan_type": "TableScan"}); got != 1 {
		t.Errorf("base-table Scan count = %d, want 1", got)
	}
	if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{"scan_type": "SearchIndexScan"}); got != 0 {
		t.Errorf("base control SearchIndexScan count = %d, want 0", got)
	}
	if got := countPlanChildLinks(plan, "Search Predicate"); got != 0 {
		t.Errorf("base control Search Predicate count = %d, want 0", got)
	}
	if got := countPlanChildLinks(plan, "Residual Condition"); got != 1 {
		t.Errorf("base control Residual Condition count = %d, want 1", got)
	}
}

func assertNgramFuzzyPlan(t *testing.T, plan *spannerpb.QueryPlan, scored bool) {
	t.Helper()
	if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{"scan_target": "NgramAlbumsFuzzyIndex", "scan_type": "SearchIndexScan"}); got != 1 {
		t.Errorf("fuzzy SearchIndexScan count = %d, want 1", got)
	}
	if got := countPlanChildLinks(plan, "Search Predicate"); got != 1 {
		t.Errorf("fuzzy Search Predicate count = %d, want 1", got)
	}
	if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{"scan_target": "NgramAlbums", "scan_type": "TableScan"}); got != 0 {
		t.Errorf("fuzzy base-table back-join Scan count = %d, want 0", got)
	}
	if scored {
		if got := countPlanNodesWithMetadata(plan, "Sort Limit", map[string]string{"call_type": "Local"}); got != 1 {
			t.Errorf("scored fuzzy local Sort Limit count = %d, want 1", got)
		}
		if got := countPlanNodesWithMetadata(plan, "Limit", map[string]string{"call_type": "Global"}); got != 1 {
			t.Errorf("scored fuzzy global Limit count = %d, want 1", got)
		}
	}
}

func assertNgramAcceleratedPattern(t *testing.T, plan *spannerpb.QueryPlan) {
	t.Helper()
	if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{"scan_target": "NgramAlbumsPatternIndex", "scan_type": "SearchIndexScan"}); got != 1 {
		t.Errorf("accelerated SearchIndexScan count = %d, want 1", got)
	}
	if got := countPlanChildLinks(plan, "Search Predicate"); got != 1 {
		t.Errorf("accelerated Search Predicate count = %d, want 1", got)
	}
	if got := countPlanChildLinks(plan, "Residual Condition"); got != 1 {
		t.Errorf("accelerated Residual Condition count = %d, want 1", got)
	}
	if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{"scan_target": "NgramAlbums", "scan_type": "TableScan"}); got != 0 {
		t.Errorf("accelerated pattern base-table back-join count = %d, want 0", got)
	}
}

func assertNgramIneligiblePattern(t *testing.T, plan *spannerpb.QueryPlan) {
	t.Helper()
	if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{
		"Full scan":   "true",
		"scan_target": "NgramAlbumsPatternIndex",
		"scan_type":   "SearchIndexScan",
	}); got != 1 {
		t.Errorf("ineligible forced-index full Scan count = %d, want 1", got)
	}
	if got := countPlanChildLinks(plan, "Search Predicate"); got != 0 {
		t.Errorf("ineligible pattern Search Predicate count = %d, want 0", got)
	}
	if got := countPlanChildLinks(plan, "Residual Condition"); got != 1 {
		t.Errorf("ineligible pattern Residual Condition count = %d, want 1", got)
	}
}

func countPlanChildLinks(plan *spannerpb.QueryPlan, linkType string) int {
	count := 0
	for _, node := range plan.GetPlanNodes() {
		for _, link := range node.GetChildLinks() {
			if link.GetType() == linkType {
				count++
			}
		}
	}
	return count
}
