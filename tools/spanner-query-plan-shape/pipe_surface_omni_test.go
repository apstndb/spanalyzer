//go:build integration && omni

package main

import (
	"strconv"
	"testing"
)

func TestIntegrationPipeSurfaceVersionMatrixOnOmni(t *testing.T) {
	ddls, err := parseBuiltInDDLs("pipe-surface-schema.sql", docsDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	clients := openOmniClients(t, ddls)

	wantNode := map[string]string{
		"pipe-surface/accepted/select":      "Scan",
		"pipe-surface/accepted/extend":      "Function",
		"pipe-surface/accepted/set":         "Function",
		"pipe-surface/accepted/drop":        "Scan",
		"pipe-surface/accepted/rename":      "Scan",
		"pipe-surface/accepted/as":          "Scan",
		"pipe-surface/accepted/where":       "Filter Scan",
		"pipe-surface/accepted/aggregate":   "Aggregate",
		"pipe-surface/accepted/join":        "Distributed Cross Apply",
		"pipe-surface/accepted/order-by":    "Sort",
		"pipe-surface/accepted/limit":       "Limit",
		"pipe-surface/accepted/union":       "Union All",
		"pipe-surface/accepted/intersect":   "Aggregate",
		"pipe-surface/accepted/except":      "Aggregate",
		"pipe-surface/accepted/tablesample": "Random Id Assign",
	}

	for version := firstOptimizerVersion; version <= latestOptimizerVersion; version++ {
		t.Run("v"+strconv.Itoa(version), func(t *testing.T) {
			for _, query := range pipeSurfaceQueries {
				query := query
				query.SQL = withOptimizerVersionStatementHint(query.SQL, version)
				plan, err := analyzePlan(t.Context(), clients.Client, query)
				if err != nil {
					t.Errorf("AnalyzeQuery(%s, v%d) error = %v", query.Label, version, err)
					continue
				}
				nodeName := wantNode[query.Label]
				if got := countPlanNodesAnyKind(plan, nodeName); got == 0 {
					t.Errorf("%s v%d %s count = 0, want positive", query.Label, version, nodeName)
				}
			}
		})
	}
}
