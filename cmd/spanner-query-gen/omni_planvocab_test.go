//go:build integration && omni

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/encoding/protojson"
)

const planvocabCheckBinaryEnv = "SPANALYZER_PLANVOCAB_CHECK_BIN"

// querygenOmniCheckPlanVocabulary lets the root plancontract module inspect
// live querygen plans without adding an unpublished sibling-module dependency
// to cmd/spanner-query-gen. Normal integration runs leave the hook disabled.
func querygenOmniCheckPlanVocabulary(t testing.TB, label string, plan *spannerpb.QueryPlan) {
	t.Helper()
	checker := os.Getenv(planvocabCheckBinaryEnv)
	if checker == "" {
		return
	}
	planJSON, err := protojson.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal %s plan for planvocab: %v", label, err)
	}
	envelope, err := json.Marshal(struct {
		QueryLabel string          `json:"query_label"`
		Plan       json.RawMessage `json:"plan"`
	}{
		QueryLabel: label,
		Plan:       planJSON,
	})
	if err != nil {
		t.Fatalf("marshal %s plan envelope for planvocab: %v", label, err)
	}
	cmd := exec.CommandContext(t.Context(), checker, "--format", "json")
	cmd.Stdin = bytes.NewReader(envelope)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("planvocab check failed for %s: %v\n%s", label, err, output)
	}
}
