//go:build integration && omni

package main

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIntegrationAIPlanCapabilityBoundaryOnOmni(t *testing.T) {
	ddls, err := parseBuiltInDDLs("ai-plan-schema.sql", docsDDL)
	if err != nil {
		t.Fatal(err)
	}
	clients := openSearchGraphOmniClients(t, ddls)
	cases := queryCasesByLabel(t, aiPlanQueries)
	candidates := []string{
		"ai-plan/classify-projection",
		"ai-plan/if-filter",
		"ai-plan/score-order-limit",
	}
	controls := []string{
		"ai-plan/classify-case-control",
		"ai-plan/if-scalar-filter-control",
		"ai-plan/score-scalar-order-limit-control",
	}
	for version := 1; version <= 8; version++ {
		for _, label := range candidates {
			if _, err := analyzeVersionedSearchGraphPlan(t, clients.Client, cases[label], version); status.Code(err) != codes.Internal {
				t.Errorf("AnalyzeQuery(%s, v%d) error = %v, want Omni Internal capability boundary", label, version, err)
			}
		}
		for _, label := range controls {
			plan, err := analyzeVersionedSearchGraphPlan(t, clients.Client, cases[label], version)
			if err != nil {
				t.Fatalf("AnalyzeQuery(%s, v%d) control error = %v", label, version, err)
			}
			if got := countPlanNodes(plan, "TVF", ""); got != 0 {
				t.Errorf("%s v%d control TVF count = %d, want 0", label, version, got)
			}
		}
	}
}
