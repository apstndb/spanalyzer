//go:build integration && omni

package main

import (
	"strings"
	"testing"
)

func TestIntegrationDMLSurfaceOnOmni(t *testing.T) {
	ddls, err := parseBuiltInDDLs("dml-schema.sql", dmlDDL)
	if err != nil {
		t.Fatal(err)
	}
	clients := openSearchGraphOmniClients(t, ddls)
	plansByLabel := make(map[string][]optimizerVersionShape)
	for version := 1; version <= 8; version++ {
		for _, query := range dmlQueries {
			plan, err := analyzeVersionedSearchGraphPlan(t, clients.Client, query, version)
			if query.Label == "dml/insert-assert-rows-modified" {
				if err == nil || !strings.Contains(err.Error(), "ASSERT_ROWS_MODIFIED is not supported") {
					t.Errorf("AnalyzeQuery(%s, v%d) error = %v, want capability error", query.Label, version, err)
				}
				continue
			}
			if err != nil {
				t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", query.Label, version, err)
			}
			operation, table := dmlMutationMetadata(query.Label)
			if got := countPlanNodesWithMetadata(plan, "Apply Mutations", map[string]string{
				"operation_type": operation,
				"table":          table,
			}); got != 1 {
				t.Errorf("%s v%d Apply Mutations(%s, %s) count = %d, want 1", query.Label, version, operation, table, got)
			}
			plansByLabel[query.Label] = append(plansByLabel[query.Label], optimizerVersionShape{
				version: version,
				shape:   compactPlanTree(plan, true, false),
			})
		}
	}

	sensitiveGroups := map[string][]string{
		"dml/insert-on-conflict-do-nothing":        {"v1-v2", "v3-v6", "v7-v8"},
		"dml/insert-on-conflict-target-do-nothing": {"v1-v2", "v3-v8"},
		"dml/insert-on-conflict-unique-do-nothing": {"v1-v2", "v3-v8"},
		"dml/insert-on-conflict-do-update-where":   {"v1-v6", "v7-v8"},
		"dml/insert-on-conflict-select":            {"v1-v2", "v3-v8"},
		"dml/update-subquery":                      {"v1-v4", "v5-v8"},
		"dml/delete-subquery":                      {"v1-v4", "v5-v8"},
		"dml/delete-then-return":                   {"v1-v4", "v5-v8"},
	}
	for label, shapes := range plansByLabel {
		want, sensitive := sensitiveGroups[label]
		if !sensitive {
			want = []string{"v1-v8"}
		}
		groups := groupOptimizerVersionShapes(shapes)
		if len(groups) != len(want) {
			t.Errorf("%s version groups = %v, want %v", label, optimizerVersionGroupLabels(groups), want)
			continue
		}
		for i := range groups {
			if groups[i].label != want[i] {
				t.Errorf("%s version groups = %v, want %v", label, optimizerVersionGroupLabels(groups), want)
				break
			}
		}
	}
}

func dmlMutationMetadata(label string) (operation, table string) {
	switch {
	case strings.HasPrefix(label, "dml/insert-"):
		operation = "INSERT"
	case strings.HasPrefix(label, "dml/update-"):
		operation = "UPDATE"
	case strings.HasPrefix(label, "dml/delete-"):
		operation = "DELETE"
	}
	table = "Singers"
	if label == "dml/insert-fans-default-key-then-return" {
		table = "Fans"
	} else if label == "dml/update-array" {
		table = "Concerts"
	}
	return operation, table
}

func optimizerVersionGroupLabels(groups []optimizerVersionShapeGroup) []string {
	labels := make([]string, len(groups))
	for i, group := range groups {
		labels[i] = group.label
	}
	return labels
}
