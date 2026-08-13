//go:build integration && omni

package main

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanemuboost"
	"google.golang.org/protobuf/proto"
)

func TestIntegrationFullTextSearchPlansOnOmni(t *testing.T) {
	clients := openSearchGraphOmniClients(t, fullTextSearchDDLs(t))
	cases := queryCasesByLabel(t, fullTextSearchQueries)
	query := cases["full-text-search/gql-search-traversal"]

	for version := 1; version <= 8; version++ {
		plan, err := analyzeVersionedSearchGraphPlan(t, clients.Client, query, version)
		if version <= 4 {
			if err == nil || !strings.Contains(err.Error(), "SEARCH is only supported in queries with query optimizer version 6 or above") {
				t.Errorf("GQL SEARCH traversal v%d error = %v, want optimizer-version capability error", version, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", query.Label, version, err)
		}
		if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{
			"scan_target": "SearchSingersByBio",
			"scan_type":   "SearchIndexScan",
		}); got != 1 {
			t.Errorf("GQL SEARCH traversal v%d SearchIndexScan count = %d, want 1", version, got)
		}
		if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{
			"scan_target": "SearchCollaborations",
			"scan_type":   "TableScan",
		}); got != 1 {
			t.Errorf("GQL SEARCH traversal v%d edge TableScan count = %d, want 1", version, got)
		}
	}

	for version := 1; version <= 8; version++ {
		plans := map[string]*spannerpb.QueryPlan{}
		for _, label := range []string{
			"full-text-search/facet-single-count",
			"full-text-search/facet-search-only-control",
			"full-text-search/facet-multiple",
			"full-text-search/facet-result-page-control",
		} {
			plan, err := analyzeVersionedSearchGraphPlan(t, clients.Client, cases[label], version)
			if version <= 4 {
				if err == nil || !strings.Contains(err.Error(), "SEARCH is only supported in queries with query optimizer version 6 or above") {
					t.Errorf("AnalyzeQuery(%s, v%d) error = %v, want version capability error", label, version, err)
				}
				continue
			}
			if err != nil {
				t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", label, version, err)
			}
			plans[label] = plan
			if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{
				"scan_target": "SearchAlbumsFacetIndex",
				"scan_type":   "SearchIndexScan",
			}); got != 1 {
				t.Errorf("%s v%d facet SearchIndexScan count = %d, want 1", label, version, got)
			}
			if got := countPlanChildLinks(plan, "Search Predicate"); got != 1 {
				t.Errorf("%s v%d Search Predicate count = %d, want 1", label, version, got)
			}
			if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{"scan_target": "SearchAlbums", "scan_type": "TableScan"}); got != 0 {
				t.Errorf("%s v%d base-table back-join count = %d, want 0", label, version, got)
			}
		}
		if version <= 4 {
			continue
		}
		if got := countPlanNodes(plans["full-text-search/facet-single-count"], "Aggregate", ""); got != 2 {
			t.Errorf("single facet v%d Aggregate count = %d, want 2", version, got)
		}
		if got := countPlanNodes(plans["full-text-search/facet-search-only-control"], "Aggregate", ""); got != 0 {
			t.Errorf("single-facet control v%d Aggregate count = %d, want 0", version, got)
		}
		multi := plans["full-text-search/facet-multiple"]
		for displayName, want := range map[string]int{"SpoolBuild": 1, "SpoolScan": 3, "Aggregate": 2, "Array Unnest": 1, "Sort Limit": 2} {
			if got := countPlanNodes(multi, displayName, ""); got != want {
				t.Errorf("multi-facet v%d %s count = %d, want %d", version, displayName, got, want)
			}
		}
		page := plans["full-text-search/facet-result-page-control"]
		for _, displayName := range []string{"SpoolBuild", "SpoolScan", "Aggregate", "Array Unnest"} {
			if got := countPlanNodes(page, displayName, ""); got != 0 {
				t.Errorf("facet page control v%d %s count = %d, want 0", version, displayName, got)
			}
		}
	}

	for version := 1; version <= 8; version++ {
		labels := []string{
			"full-text-search/enhanced-query",
			"full-text-search/enhanced-query-control",
			"full-text-search/enhanced-query-required-hint",
			"full-text-search/enhanced-query-timeout-hint",
			"full-text-search/phonetic-composition",
			"full-text-search/phonetic-search-only-control",
		}
		plans := map[string]*spannerpb.QueryPlan{}
		for _, label := range labels {
			plan, err := analyzeVersionedSearchGraphPlan(t, clients.Client, cases[label], version)
			if version <= 4 {
				if err == nil || !strings.Contains(err.Error(), "SEARCH is only supported in queries with query optimizer version 6 or above") {
					t.Errorf("AnalyzeQuery(%s, v%d) error = %v, want version capability error", label, version, err)
				}
				continue
			}
			if err != nil {
				t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", label, version, err)
			}
			plans[label] = plan
		}
		if version <= 4 {
			continue
		}

		for _, label := range labels[:4] {
			plan := plans[label]
			if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{
				"scan_target": "SearchAlbumsTitleIndex",
				"scan_type":   "SearchIndexScan",
			}); got != 1 {
				t.Errorf("%s v%d SearchIndexScan count = %d, want 1", label, version, got)
			}
			if got := countPlanChildLinks(plan, "Search Predicate"); got != 1 {
				t.Errorf("%s v%d Search Predicate child-link count = %d, want 1", label, version, got)
			}
		}
		enhanced := plans["full-text-search/enhanced-query"]
		control := plans["full-text-search/enhanced-query-control"]
		if proto.Equal(enhanced, control) {
			t.Errorf("enhanced query v%d unexpectedly equals its false/default control", version)
		}
		if !planShortDescriptionContains(enhanced, "enhance_query: true") || !planShortDescriptionContains(control, "enhance_query: false") {
			t.Errorf("enhanced query v%d lacks explicit true/false conversion descriptions", version)
		}
		for _, label := range []string{"full-text-search/enhanced-query-required-hint", "full-text-search/enhanced-query-timeout-hint"} {
			if !proto.Equal(plans[label], enhanced) {
				t.Errorf("%s v%d plan differs from enhanced query without statement hint", label, version)
			}
		}

		phonetic := plans["full-text-search/phonetic-composition"]
		phoneticControl := plans["full-text-search/phonetic-search-only-control"]
		for _, plan := range []*spannerpb.QueryPlan{phonetic, phoneticControl} {
			if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{
				"scan_target": "SearchAlbumsPhoneticIndex",
				"scan_type":   "SearchIndexScan",
			}); got != 1 {
				t.Errorf("phonetic v%d SearchIndexScan count = %d, want 1", version, got)
			}
		}
		if got := countPlanNodesAnyKind(phonetic, "Search Predicate"); got != 2 {
			t.Errorf("phonetic composition v%d Search Predicate node count = %d, want 2", version, got)
		}
		if got := countPlanNodesAnyKind(phoneticControl, "Search Predicate"); got != 1 {
			t.Errorf("phonetic control v%d Search Predicate node count = %d, want 1", version, got)
		}
		if got := countPlanNodesAnyKind(phonetic, "Function"); got != 1 || !planShortDescriptionContains(phonetic, "ArtistNameSoundex_Tokens") {
			t.Errorf("phonetic composition v%d missing combined search-predicate Function", version)
		}
		if got := countPlanNodesAnyKind(phoneticControl, "Function"); got != 0 {
			t.Errorf("phonetic control v%d Function count = %d, want 0", version, got)
		}
	}
}

func planShortDescriptionContains(plan *spannerpb.QueryPlan, substring string) bool {
	for _, node := range plan.GetPlanNodes() {
		if strings.Contains(node.GetShortRepresentation().GetDescription(), substring) {
			return true
		}
	}
	return false
}

func TestIntegrationVectorSearchPlansOnOmni(t *testing.T) {
	clients := openSearchGraphOmniClients(t, append([]string(nil), vectorSearchDDLs...))
	cases := queryCasesByLabel(t, vectorSearchQueries)
	query := cases["vector-search/ann-gql-next-traversal"]

	for version := 1; version <= 8; version++ {
		plan, err := analyzeVersionedSearchGraphPlan(t, clients.Client, query, version)
		if err != nil {
			t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", query.Label, version, err)
		}
		for _, scanType := range []string{"VectorIndexMetadataScan", "VectorIndexRootScan", "VectorIndexLeafScan"} {
			if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{
				"scan_target": "VectorDocumentsByEmbedding",
				"scan_type":   scanType,
			}); got != 1 {
				t.Errorf("ANN GQL NEXT traversal v%d %s count = %d, want 1", version, scanType, got)
			}
		}
		if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{
			"scan_target": "VectorRelated",
			"scan_type":   "TableScan",
		}); got != 1 {
			t.Errorf("ANN GQL NEXT traversal v%d edge TableScan count = %d, want 1", version, got)
		}
		if got := countPlanNodes(plan, "Distributed Cross Apply", ""); got < 3 {
			t.Errorf("ANN GQL NEXT traversal v%d Distributed Cross Apply count = %d, want at least 3", version, got)
		}

		for _, tt := range []struct {
			label  string
			target string
		}{
			{label: "vector-search/ann-dot-product", target: "VectorDocumentsByDotProduct"},
			{label: "vector-search/ann-euclidean-distance", target: "VectorDocumentsByEuclidean"},
		} {
			plan, err := analyzeVersionedSearchGraphPlan(t, clients.Client, cases[tt.label], version)
			if err != nil {
				t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", tt.label, version, err)
			}
			for _, scanType := range []string{"VectorIndexRootScan", "VectorIndexLeafScan"} {
				if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{
					"scan_target": tt.target,
					"scan_type":   scanType,
				}); got != 1 {
					t.Errorf("%s v%d %s count = %d, want 1", tt.label, version, scanType, got)
				}
			}
		}
	}

	for version := 1; version <= 8; version++ {
		for _, tt := range []struct {
			label      string
			wantVector bool
		}{
			{label: "vector-search/hybrid-rrf", wantVector: true},
			{label: "vector-search/hybrid-rrf-exact-control"},
		} {
			t.Run(tt.label+"/v"+strconv.Itoa(version), func(t *testing.T) {
				plan, err := analyzeVersionedSearchGraphPlan(t, clients.Client, cases[tt.label], version)
				if version <= 4 {
					if err == nil || !strings.Contains(err.Error(), "SCORE is only supported in queries with query optimizer version 6 or above") {
						t.Errorf("AnalyzeQuery(%s, v%d) error = %v, want SCORE version capability error", tt.label, version, err)
					}
					return
				}
				if err != nil {
					t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", tt.label, version, err)
				}
				iteratorType := "Hash"
				if version == 5 {
					iteratorType = "Stream"
				}
				assertHybridRRFTopology(t, plan, tt.wantVector, iteratorType)
			})
		}
	}
}

func assertHybridRRFTopology(t *testing.T, plan *spannerpb.QueryPlan, wantVector bool, iteratorType string) {
	t.Helper()
	index := make(map[int32]*spannerpb.PlanNode, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		index[node.GetIndex()] = node
	}
	matching := func(displayName string, metadata map[string]string) []int32 {
		var indexes []int32
		for _, node := range plan.GetPlanNodes() {
			if node.GetKind() != spannerpb.PlanNode_RELATIONAL || node.GetDisplayName() != displayName {
				continue
			}
			matches := true
			for key, want := range metadata {
				value := node.GetMetadata().GetFields()[key]
				if value == nil || value.GetStringValue() != want {
					matches = false
					break
				}
			}
			if matches {
				indexes = append(indexes, node.GetIndex())
			}
		}
		return indexes
	}
	reachable := func(from, target int32) bool {
		seen := map[int32]bool{}
		stack := []int32{from}
		for len(stack) > 0 {
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if current == target {
				return true
			}
			if seen[current] {
				continue
			}
			seen[current] = true
			node := index[current]
			if node == nil {
				t.Fatalf("plan child index %d has no node", current)
			}
			for _, link := range node.GetChildLinks() {
				stack = append(stack, link.GetChildIndex())
			}
		}
		return false
	}
	one := func(name string, indexes []int32) int32 {
		t.Helper()
		if len(indexes) != 1 {
			t.Fatalf("%s node count = %d, want 1", name, len(indexes))
		}
		return indexes[0]
	}

	search := one("hybrid search-index scan", matching("Scan", map[string]string{
		"scan_target": "VectorDocumentsByBody",
		"scan_type":   "SearchIndexScan",
	}))
	hasSearchPredicate := false
	for _, link := range index[search].GetChildLinks() {
		if link.GetType() == "Search Predicate" {
			hasSearchPredicate = true
		}
	}
	if !hasSearchPredicate {
		t.Error("hybrid SearchIndexScan lacks a Search Predicate child")
	}

	retrieval := []int32{search}
	for _, scanType := range []string{"VectorIndexRootScan", "VectorIndexLeafScan"} {
		nodes := matching("Scan", map[string]string{
			"scan_target": "VectorDocumentsByDotProduct",
			"scan_type":   scanType,
		})
		if wantVector {
			retrieval = append(retrieval, one(scanType, nodes))
		} else if len(nodes) != 0 {
			t.Errorf("exact hybrid control %s count = %d, want 0", scanType, len(nodes))
		}
	}
	if !wantVector {
		retrieval = append(retrieval, one("hybrid base-table scan", matching("Scan", map[string]string{
			"scan_target": "VectorDocuments",
			"scan_type":   "TableScan",
		})))
		if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{"scan_type": "VectorIndexMetadataScan"}); got != 0 {
			t.Errorf("exact hybrid control VectorIndexMetadataScan count = %d, want 0", got)
		}
	}

	var fusionUnion int32 = -1
	for _, candidate := range matching("Union All", nil) {
		containsAll := true
		for _, target := range retrieval {
			containsAll = containsAll && reachable(candidate, target)
		}
		if containsAll {
			fusionUnion = candidate
			break
		}
	}
	if fusionUnion < 0 {
		t.Fatal("no Union All contains both hybrid retrieval branches")
	}

	var fusionAggregate int32 = -1
	for _, candidate := range matching("Aggregate", map[string]string{"iterator_type": iteratorType}) {
		if reachable(candidate, fusionUnion) {
			fusionAggregate = candidate
			break
		}
	}
	if fusionAggregate < 0 {
		t.Fatalf("no %s Aggregate contains the hybrid Union All", iteratorType)
	}
	for _, candidate := range matching("Sort Limit", map[string]string{"call_type": "Global"}) {
		if reachable(candidate, fusionAggregate) {
			return
		}
	}
	t.Fatal("no global Sort Limit contains the hybrid Aggregate")
}

func fullTextSearchDDLs(t *testing.T) []string {
	t.Helper()
	ddls, err := parseBuiltInDDLs("full-text-search-schema.sql", fullTextSearchDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	return ddls
}

func openSearchGraphOmniClients(t *testing.T, ddls []string) *spanemuboost.Clients {
	t.Helper()
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	image := strings.TrimSpace(os.Getenv("SPANALYZER_OMNI_IMAGE"))
	if image == "" {
		t.Fatal("set SPANALYZER_OMNI_IMAGE to the pinned Spanner Omni image under test")
	}
	runtime := spanemuboost.NewLazyRuntime(
		spanemuboost.BackendOmni,
		spanemuboost.WithContainerImage(image),
	)
	t.Cleanup(func() { _ = runtime.Close() })
	clients, err := spanemuboost.OpenClients(t.Context(), runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(ddls),
	)
	if err != nil {
		t.Fatalf("OpenClients() error = %v", err)
	}
	t.Cleanup(func() { _ = clients.Close() })
	return clients
}

func analyzeVersionedSearchGraphPlan(t *testing.T, client *spanner.Client, query queryCase, version int) (*spannerpb.QueryPlan, error) {
	t.Helper()
	query.SQL = withOptimizerVersionStatementHint(query.SQL, version)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	return analyzePlan(ctx, client, query)
}
