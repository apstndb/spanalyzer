package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPlanReportValidatesContractsBeforeAnalyzingSQL(t *testing.T) {
	dir := t.TempDir()
	// The DDL path deliberately does not exist. A contract error must win
	// before catalog analysis can open it, much less start an Omni backend.
	configPath := filepath.Join(dir, "querygen.yaml")
	if err := os.WriteFile(configPath, []byte(`version: v1alpha
go:
  package: db
catalogs:
- name: spanner
  kind: spanner
  ddl: missing.sql
queries:
- name: Q
  catalog: spanner
  kind: sql
  sql: SELECT 1 AS Id
  result:
    struct: QRow
`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "contracts.yaml"), []byte(`version: v1alpha-plan-contracts
contracts:
- name: Invalid
  target: query/Q
  use: [not_a_contract]
`), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := run([]string{"plan-report", "--config", configPath, "--contracts", "contracts.yaml"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "unsupported predefined plan contract") {
		t.Fatalf("run() error = %v, want contract validation before catalog analysis", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("run() output = %s, want no report on invalid input", stdout.String())
	}
}
