//go:build integration && omni

package main

import (
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/proto"
)

func TestIntegrationRemoteFunctionPlansOnOmni(t *testing.T) {
	ddls, err := parseBuiltInDDLs("remote-function-schema.sql", remoteFunctionDDL)
	if err != nil {
		t.Fatal(err)
	}
	clients := openOmniClients(t, ddls)
	cases := queryCasesByLabel(t, remoteFunctionQueries)

	for _, label := range []string{"remote-function/literal-input", "remote-function/table-input"} {
		var first *spannerpb.QueryPlan
		for version := firstOptimizerVersion; version <= latestOptimizerVersion; version++ {
			plan, err := analyzeVersionedSearchGraphPlan(t, clients.Client, cases[label], version)
			if err != nil {
				t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", label, version, err)
			}
			if got := countPlanNodesWithMetadata(plan, "TVF", map[string]string{"Name": "spanalyzer_remote.remote_add"}); got != 1 {
				t.Errorf("%s v%d remote-function TVF count = %d, want 1", label, version, got)
			}
			if first == nil {
				first = proto.Clone(plan).(*spannerpb.QueryPlan)
			} else if !proto.Equal(first, plan) {
				t.Errorf("%s v%d plan differs from v1", label, version)
			}
		}
	}

	literal, err := analyzeVersionedSearchGraphPlan(t, clients.Client, cases["remote-function/literal-input"], latestOptimizerVersion)
	if err != nil {
		t.Fatal(err)
	}
	if got := countPlanNodes(literal, "Compute", ""); got != 1 {
		t.Errorf("literal-input Compute count = %d, want 1", got)
	}
	if got := countPlanNodes(literal, "Unit Relation", ""); got != 1 {
		t.Errorf("literal-input Unit Relation count = %d, want 1", got)
	}
	if got := countPlanNodes(literal, "Scan", ""); got != 0 {
		t.Errorf("literal-input Scan count = %d, want 0", got)
	}

	tableInput, err := analyzeVersionedSearchGraphPlan(t, clients.Client, cases["remote-function/table-input"], latestOptimizerVersion)
	if err != nil {
		t.Fatal(err)
	}
	if got := countPlanNodesWithMetadata(tableInput, "Scan", map[string]string{
		"scan_target": "RemoteFunctionInputs",
		"scan_type":   "TableScan",
	}); got != 1 {
		t.Errorf("table-input TableScan count = %d, want 1", got)
	}
	if got := countPlanChildLinks(tableInput, "PassThroughVars"); got != 1 {
		t.Errorf("table-input PassThroughVars count = %d, want 1", got)
	}
}
