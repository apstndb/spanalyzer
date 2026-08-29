//go:build integration && omni

package main

import (
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestIntegrationExpressionIndexesOnOmni(t *testing.T) {
	clients := openOmniClients(t, expressionIndexDDLs)
	cases := queryCasesByLabel(t, expressionIndexQueries)
	mustAnalyze := func(t *testing.T, label string) *spannerpb.QueryPlan {
		t.Helper()
		plan, err := analyzePlan(t.Context(), clients.Client, cases[label])
		if err != nil {
			t.Fatalf("AnalyzeQuery(%s) error = %v", label, err)
		}
		return plan
	}

	for _, tt := range []struct {
		name          string
		label         string
		index         string
		variableParts []string
	}{
		{
			name:          "automatic selection",
			label:         "expression-index/auto-city",
			index:         "ExpressionIndexVenuesByCity",
			variableParts: []string{"_ExpressionIndex_", "ByCity_0"},
		},
		{
			name:          "forced selection",
			label:         "expression-index/force-city",
			index:         "ExpressionIndexVenuesByCity",
			variableParts: []string{"_ExpressionIndex_", "ByCity_0"},
		},
		{
			name:          "composite column and expression key",
			label:         "expression-index/composite-name-state",
			index:         "ExpressionIndexVenuesByNameState",
			variableParts: []string{"VenueName", "_ExpressionIndex_", "ByNameState_0"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			plan := mustAnalyze(t, tt.label)
			if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{
				"scan_target": tt.index,
				"scan_type":   "IndexScan",
			}); got != 1 {
				t.Errorf("%s index scan count = %d, want 1", tt.index, got)
			}
			if !planHasChildLinkType(plan, "Scan", "Seek Condition") {
				t.Errorf("%s plan lacks Scan Seek Condition", tt.label)
			}
			for _, fragment := range tt.variableParts {
				if !planHasChildVariableContaining(plan, fragment) {
					t.Errorf("%s plan lacks child variable fragment %q", tt.label, fragment)
				}
			}
			if !planHasChildLinkType(plan, "Distributed Union", "Split Range") {
				t.Errorf("%s plan lacks Distributed Union Split Range", tt.label)
			}
		})
	}

	t.Run("base table control", func(t *testing.T) {
		plan := mustAnalyze(t, "expression-index/base-table-control")
		if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{
			"scan_target": "ExpressionIndexVenues",
			"scan_type":   "TableScan",
		}); got != 1 {
			t.Errorf("base-table scan count = %d, want 1", got)
		}
		if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{"scan_type": "IndexScan"}); got != 0 {
			t.Errorf("base-table control index scan count = %d, want 0", got)
		}
	})
}

func planHasChildLinkType(plan *spannerpb.QueryPlan, parentDisplayName, linkType string) bool {
	for _, node := range plan.GetPlanNodes() {
		if node.GetDisplayName() != parentDisplayName {
			continue
		}
		for _, link := range node.GetChildLinks() {
			if link.GetType() == linkType {
				return true
			}
		}
	}
	return false
}
