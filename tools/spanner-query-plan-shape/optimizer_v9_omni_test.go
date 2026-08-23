//go:build integration && omni

package main

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestIntegrationOptimizerV9OnOmni(t *testing.T) {
	ddls, err := parseBuiltInDDLs("optimizer-v9-schema.sql", dmlDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	clients := openOmniClients(t, ddls)
	cases := queryCasesByLabel(t, optimizerV9Queries)
	mustAnalyze := func(t *testing.T, label string) *spannerpb.QueryPlan {
		t.Helper()
		plan, err := analyzePlan(t.Context(), clients.Client, cases[label])
		if err != nil {
			t.Fatalf("AnalyzeQuery(%s) error = %v", label, err)
		}
		return plan
	}

	v8 := mustAnalyze(t, "optimizer-v9/dca-input-column-pruning/v8-control")
	if got := countPlanNodesWithMetadata(v8, "Distributed Cross Apply", map[string]string{"execution_method": "Row"}); got != 1 {
		t.Errorf("v8 row DCA count = %d, want 1", got)
	}
	if planHasChildVariableContaining(v8, "__row_id") {
		t.Error("v8 plan unexpectedly contains an __row_id child variable")
	}

	v9 := mustAnalyze(t, "optimizer-v9/dca-input-column-pruning/v9")
	if got := countPlanNodesWithMetadata(v9, "Distributed Cross Apply", map[string]string{"execution_method": "Batch"}); got != 1 {
		t.Errorf("v9 batch DCA count = %d, want 1", got)
	}
	if !planHasChildVariableContaining(v9, "__row_id") {
		t.Error("v9 plan lacks the __row_id batch field")
	}
	if !planHasChildVariableContaining(v9, "restored_") {
		t.Error("v9 plan lacks a restored input-column variable")
	}
	if !planHasTypedNullBatchField(v9, "__row_id") {
		t.Error("v9 Create Batch __row_id field is not defined by Constant <typed null>")
	}

	limit := mustAnalyze(t, "optimizer-v9/dca-input-column-pruning-limit/v9")
	if got := countPlanNodesAnyKind(limit, "Limit"); got == 0 {
		t.Error("v9 LIMIT control has no Limit operator")
	}
	indexUnion := mustAnalyze(t, "optimizer-v9/index-union-aggressiveness/v9")
	if got := countPlanNodesAnyKind(indexUnion, "Union All"); got == 0 {
		t.Error("v9 index-union probe has no Union All operator")
	}
	v8Delete := mustAnalyze(t, "optimizer-v9/dml-delete-then-return/v8-control")
	v9Delete := mustAnalyze(t, "optimizer-v9/dml-delete-then-return/v9")
	for label, plan := range map[string]*spannerpb.QueryPlan{"v8": v8Delete, "v9": v9Delete} {
		if got := countPlanNodesWithMetadata(plan, "Apply Mutations", map[string]string{"operation_type": "DELETE", "table": "Singers"}); got != 1 {
			t.Errorf("%s DELETE Apply Mutations count = %d, want 1", label, got)
		}
	}
	if v8Shape, v9Shape := compactPlanTree(v8Delete, true, false), compactPlanTree(v9Delete, true, false); v8Shape == v9Shape {
		t.Errorf("DELETE THEN RETURN v8 and v9 shapes are equal; want a version boundary: %s", v8Shape)
	}

	if _, err := analyzePlan(t.Context(), clients.Client, cases["optimizer-v9/unsupported/v10"]); err == nil || !strings.Contains(err.Error(), "Query optimizer version: 10 is not supported") {
		t.Errorf("optimizer v10 error = %v, want unsupported-version boundary", err)
	}
}

func planHasChildVariableContaining(plan *spannerpb.QueryPlan, fragment string) bool {
	for _, node := range plan.GetPlanNodes() {
		for _, link := range node.GetChildLinks() {
			if strings.Contains(link.GetVariable(), fragment) {
				return true
			}
		}
	}
	return false
}

func planHasTypedNullBatchField(plan *spannerpb.QueryPlan, field string) bool {
	nodes := make(map[int32]*spannerpb.PlanNode, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		nodes[node.GetIndex()] = node
	}
	for _, node := range plan.GetPlanNodes() {
		if node.GetDisplayName() != "Create Batch" {
			continue
		}
		for _, link := range node.GetChildLinks() {
			if !strings.Contains(link.GetVariable(), field) {
				continue
			}
			child := nodes[link.GetChildIndex()]
			if child.GetDisplayName() == "Constant" && child.GetShortRepresentation().GetDescription() == "<typed null>" {
				return true
			}
		}
	}
	return false
}
