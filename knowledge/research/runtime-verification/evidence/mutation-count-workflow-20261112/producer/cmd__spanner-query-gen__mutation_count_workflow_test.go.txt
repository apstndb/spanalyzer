package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/apstndb/spanalyzer/internal/querygen"
)

func TestRunExplainPlanMutationWorkflowFixture(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{
		"explain-plan", "--config", filepath.Join("..", "..", "testdata", "mutation-count", "querygen.yaml"),
		"--mutation-facts-file", "scenarios.json", "--output", "json", "--stable",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("documented offline workflow: %v\n%s", err, &stderr)
	}
	var plan querygen.QueryCodegenPlan
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Writes) != 3 {
		t.Fatalf("expected three configured writes, got %d", len(plan.Writes))
	}
	for _, write := range plan.Writes {
		for method, count := range write.MutationCounts {
			if count.OldRows == nil || count.Observation != nil || count.Effective.Upper == nil {
				t.Fatalf("%s/%s: offline facts did not stay distinct from measurement: %+v", write.Name, method, count)
			}
			want := int64(6)
			if write.Name == "UpdateExtra" {
				want = 3
			}
			if count.Effective.Lower != want || *count.Effective.Upper != want {
				t.Fatalf("%s/%s: conditional interval %+v, want %d", write.Name, method, count.Effective, want)
			}
		}
		wantMethods := 2
		if write.Name == "ReplaceEvent" {
			wantMethods = 1
		}
		if len(write.MutationCounts) != wantMethods {
			t.Fatalf("%s: expected %d API analyses, got %d", write.Name, wantMethods, len(write.MutationCounts))
		}
	}
}
