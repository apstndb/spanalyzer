package main

import (
	"strings"
	"testing"
)

func TestStatementSurfaceCases(t *testing.T) {
	assertRetainedSurfaceLabels(t, "statement_surface", "statement-surface/")

	ddls, err := loadDDLs("statement_surface", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ddls) != 0 {
		t.Fatalf("statement surface DDL count = %d, want 0", len(ddls))
	}
}

func TestDMLCases(t *testing.T) {
	assertRetainedSurfaceLabels(t, "dml", "dml/")
}

func TestRemoteFunctionCases(t *testing.T) {
	assertRetainedSurfaceLabels(t, "remote_function", "remote-function/")
	ddls, err := loadDDLs("remote_function", nil)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{
		"CREATE SCHEMA spanalyzer_remote",
		"CREATE TABLE RemoteFunctionInputs",
		"CREATE FUNCTION spanalyzer_remote.remote_add",
		"LANGUAGE REMOTE",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("remote_function DDL missing %q", want)
		}
	}
}

func assertRetainedSurfaceLabels(t *testing.T, selector, prefix string) {
	t.Helper()
	queries, err := loadQueries(selector, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool, len(queries))
	for _, query := range queries {
		if !strings.HasPrefix(query.Label, prefix) {
			t.Errorf("selector %s label %q lacks prefix %q", selector, query.Label, prefix)
		}
		if seen[query.Label] {
			t.Errorf("selector %s has duplicate label %q", selector, query.Label)
		}
		seen[query.Label] = true
	}
}
