//go:build integration && omni

package main

import (
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanemuboost"
)

func TestIntegrationFullTextSearchGQLTraversalOnOmni(t *testing.T) {
	clients := openSearchGraphOmniClients(t, fullTextSearchDDLs(t))
	query := queryCasesByLabel(t, fullTextSearchQueries)["full-text-search/gql-search-traversal"]

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
	return analyzePlan(t.Context(), client, query)
}
