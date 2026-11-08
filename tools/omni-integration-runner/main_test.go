package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestIntegrationSteps(t *testing.T) {
	steps := integrationSteps("/repo")
	if len(steps) != 3 {
		t.Fatalf("integration step count = %d, want 3", len(steps))
	}
	if steps[0].name != "plan-shape" || steps[1].name != "query-generator" || steps[2].name != "schema-reconstruction" {
		t.Fatalf("integration steps = %#v", steps)
	}
	for _, current := range steps {
		if len(current.args) == 0 || current.args[0] != "test" {
			t.Errorf("step %q arguments = %v", current.name, current.args)
		}
	}
}

func TestUnknownOmniVersionFailsBeforeRuntimeStartup(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{"--omni-version", "unregistered"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), `version "unregistered" is not pinned`) {
		t.Fatalf("run error = %v", err)
	}
}

func TestGitSourceState(t *testing.T) {
	head, _, err := gitSourceState(context.Background(), "../..")
	if err != nil {
		t.Fatal(err)
	}
	if len(head) != 40 {
		t.Fatalf("source HEAD = %q, want 40 hex characters", head)
	}
}

func TestWriteReport(t *testing.T) {
	var output bytes.Buffer
	want := report{SchemaVersion: "v0alpha1", Status: "ok", Image: "example:tag@sha256:digest"}
	if err := writeReport(&output, want); err != nil {
		t.Fatal(err)
	}
	var got report
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != want.SchemaVersion || got.Status != want.Status || got.Image != want.Image {
		t.Fatalf("decoded report = %#v, want %#v", got, want)
	}
}
