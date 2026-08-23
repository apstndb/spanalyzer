//go:build integration && omni

package main

import (
	"strconv"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestIntegrationRewriterSurfaceVersionMatrixOnOmni(t *testing.T) {
	ddls, err := parseBuiltInDDLs("rewriter-surface-schema.sql", rewriterSurfaceDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	clients := openOmniClients(t, ddls)

	cases := queryCasesByLabel(t, rewriterSurfaceQueries)
	analyze := func(t *testing.T, label string, version int) (*spannerpb.QueryPlan, error) {
		t.Helper()
		query, ok := cases[label]
		if !ok {
			t.Fatalf("rewriter surface query %q is missing", label)
		}
		query.SQL = withOptimizerVersionStatementHint(query.SQL, version)
		return analyzePlan(t.Context(), clients.Client, query)
	}
	mustAnalyze := func(t *testing.T, label string, version int) *spannerpb.QueryPlan {
		t.Helper()
		plan, err := analyze(t, label, version)
		if err != nil {
			t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", label, version, err)
		}
		return plan
	}

	for version := firstOptimizerVersion; version <= latestOptimizerVersion; version++ {
		t.Run("v"+strconv.Itoa(version), func(t *testing.T) {
			first := mustAnalyze(t, "rewriter-surface/accepted/array-first", version)
			if !planHasDescriptionContaining(first, "ARRAY_AT_OFFSET") {
				t.Errorf("ARRAY_FIRST v%d plan does not expose ARRAY_AT_OFFSET lowering", version)
			}
			last := mustAnalyze(t, "rewriter-surface/accepted/array-last", version)
			if !planHasDescriptionContaining(last, "ARRAY_AT_ORDINAL") {
				t.Errorf("ARRAY_LAST v%d plan does not expose ARRAY_AT_ORDINAL lowering", version)
			}

			for _, label := range []string{
				"rewriter-surface/accepted/array-min",
				"rewriter-surface/accepted/array-max",
			} {
				plan := mustAnalyze(t, label, version)
				for _, nodeName := range []string{"Scalar Subquery", "Sort Limit", "Filter", "Array Unnest"} {
					if got := countPlanNodesAnyKind(plan, nodeName); got == 0 {
						t.Errorf("%s v%d %s count = 0, want positive", label, version, nodeName)
					}
				}
			}

			for label, wantNodes := range map[string][]string{
				"rewriter-surface/accepted/array-slice":                {"Array Subquery", "Filter", "Array Unnest"},
				"rewriter-surface/accepted/array-filter-with-index":    {"Array Subquery", "Filter", "Array Unnest"},
				"rewriter-surface/accepted/array-transform-with-index": {"Array Subquery", "Array Unnest"},
				"rewriter-surface/accepted/array-includes-value":       {"Scalar Subquery", "Aggregate", "Filter", "Array Unnest"},
				"rewriter-surface/accepted/array-includes-lambda":      {"Scalar Subquery", "Aggregate", "Filter", "Array Unnest"},
				"rewriter-surface/accepted/array-includes-any":         {"Scalar Subquery", "Aggregate", "Filter", "Array Unnest"},
				"rewriter-surface/accepted/array-includes-all":         {"Scalar Subquery", "Aggregate", "Cross Apply", "Array Unnest"},
			} {
				plan := mustAnalyze(t, label, version)
				for _, nodeName := range wantNodes {
					if got := countPlanNodesAnyKind(plan, nodeName); got == 0 {
						t.Errorf("%s v%d %s count = 0, want positive", label, version, nodeName)
					}
				}
			}

			dotProduct := mustAnalyze(t, "rewriter-surface/accepted/dot-product", version)
			if !planHasDescriptionContaining(dotProduct, "DOT_PRODUCT") {
				t.Errorf("DOT_PRODUCT v%d plan lacks the function expression", version)
			}
			arrayConcatAgg := mustAnalyze(t, "rewriter-surface/accepted/array-concat-agg-order-limit", version)
			for _, nodeName := range []string{"Aggregate", "Sort Limit", "Array Unnest", "Minor Sort"} {
				if got := countPlanNodesAnyKind(arrayConcatAgg, nodeName); got == 0 {
					t.Errorf("ARRAY_CONCAT_AGG ORDER BY LIMIT v%d %s count = 0, want positive", version, nodeName)
				}
			}

			view := mustAnalyze(t, "rewriter-surface/accepted/view", version)
			viewControl := mustAnalyze(t, "rewriter-surface/accepted/view-control", version)
			if got, want := compactPlanTree(view, true, false), compactPlanTree(viewControl, true, false); got != want {
				t.Errorf("view and inline control shapes differ at v%d:\nview: %s\ncontrol: %s", version, got, want)
			}
			nestedView := mustAnalyze(t, "rewriter-surface/accepted/nested-view", version)
			nestedViewControl := mustAnalyze(t, "rewriter-surface/accepted/nested-view-control", version)
			if got, want := compactPlanTree(nestedView, true, false), compactPlanTree(nestedViewControl, true, false); got != want {
				t.Errorf("nested view and inline control shapes differ at v%d:\nview: %s\ncontrol: %s", version, got, want)
			}
			insertValues := mustAnalyze(t, "rewriter-surface/accepted/insert-values-multi-row", version)
			for nodeName, wantCount := range map[string]int{"Apply Mutations": 1, "Union All": 1, "Union Input": 2, "Unit Relation": 2} {
				if got := countPlanNodesAnyKind(insertValues, nodeName); got != wantCount {
					t.Errorf("multi-row INSERT VALUES v%d %s count = %d, want %d", version, nodeName, got, wantCount)
				}
			}

			for label, wantError := range map[string]string{
				"rewriter-surface/unsupported/aggregation-threshold": "Unexpected keyword WITH",
				"rewriter-surface/unsupported/anonymization":         "Unexpected keyword WITH",
				"rewriter-surface/unsupported/differential-privacy":  "Unexpected keyword WITH",
				"rewriter-surface/unsupported/flatten":               "Function not found: FLATTEN",
				"rewriter-surface/unsupported/like-any":              "LIKE ANY is not supported",
				"rewriter-surface/unsupported/quantified-comparison": "Quantified comparisons (= ANY) are not supported",
				"rewriter-surface/unsupported/nulliferror":           "Unsupported built-in function: nulliferror",
				"rewriter-surface/unsupported/typeof":                "Function not found: TYPEOF",
				"rewriter-surface/unsupported/multiway-unnest":       "The UNNEST operator supports exactly one argument",
				"rewriter-surface/unsupported/pipe-assert":           "Pipe ASSERT not supported",
				"rewriter-surface/unsupported/pipe-describe":         "Pipe DESCRIBE not supported",
				"rewriter-surface/unsupported/pipe-if":               "Pipe IF not supported",
				"rewriter-surface/unsupported/hop":                   "Table-valued function not found: HOP",
				"rewriter-surface/unsupported/tumble":                "Table-valued function not found: TUMBLE",
				"rewriter-surface/unsupported/nested-array-update":   "Nested updates on table columns of type ARRAY are not supported",
			} {
				if _, err := analyze(t, label, version); err == nil || !strings.Contains(err.Error(), wantError) {
					t.Errorf("AnalyzeQuery(%s, v%d) error = %v, want containing %q", label, version, err, wantError)
				}
			}
		})
	}
}

func planHasDescriptionContaining(plan *spannerpb.QueryPlan, want string) bool {
	for _, node := range plan.GetPlanNodes() {
		if strings.Contains(node.GetShortRepresentation().GetDescription(), want) {
			return true
		}
	}
	return false
}

func countPlanNodesAnyKind(plan *spannerpb.QueryPlan, displayName string) int {
	count := 0
	for _, node := range plan.GetPlanNodes() {
		if node.GetDisplayName() == displayName {
			count++
		}
	}
	return count
}
