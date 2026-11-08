package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/internal/querygen"
)

const sqlChoiceTestDDL = `CREATE TABLE Events (
  Id INT64 NOT NULL, Shard INT64, CreatedAt TIMESTAMP,
  Active STRING(MAX), Pending STRING(MAX)
) PRIMARY KEY (Id)`

const sqlChoiceTestYAML = `version: v1alpha
go:
  package: db
emit:
  spanner:
    mutations: true
catalogs:
- {name: app, kind: spanner, ddl: schema.sql}
queries:
- name: FindEvents
  catalog: app
  kind: sql
  sql: |
    SELECT Id FROM Events WHERE Shard = @shard
    /*?choice:bounds*/ AND CreatedAt BETWEEN @start AND @end /*?end*/
    /*?choice:flag*/ AND Active = "1" /*?end*/
  params:
  - {name: shard, type: INT64}
  - {name: start, type: TIMESTAMP}
  - {name: end, type: TIMESTAMP}
  - name: bounds
    type: STRING
    optional: sql_choice
    default: both
    choices:
      both: " AND CreatedAt BETWEEN @start AND @end"
      start: " AND CreatedAt >= @start"
      end: " AND CreatedAt <= @end"
  - name: flag
    type: STRING
    optional: sql_choice
    default: active
    choices:
      active: ' AND Active = "1"'
      pending: ' AND Pending = "1"'
  result: {cardinality: many, struct: EventRow}
`

func sqlChoiceTestConfig(t testing.TB) (querygen.QueryCodegenConfig, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "schema.sql"), []byte(sqlChoiceTestDDL), 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := querygen.ParseQueryCodegenConfigYAML([]byte(sqlChoiceTestYAML))
	if err != nil {
		t.Fatal(err)
	}
	return config, dir
}

func TestSQLChoiceCLIAndPlanInputs(t *testing.T) {
	config, dir := sqlChoiceTestConfig(t)
	configFile := filepath.Join(dir, "querygen.yaml")
	if err := os.WriteFile(configFile, []byte(sqlChoiceTestYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if err := run([]string{"generate", "--config", configFile, "--out", filepath.Join(dir, "generated.go")}, &out, &stderr); err != nil {
		t.Fatalf("generate: %v, %s", err, stderr.String())
	}
	if err := run([]string{"check", "--config", configFile, "--out", filepath.Join(dir, "generated.go")}, &out, &stderr); err != nil {
		t.Fatalf("check: %v, %s", err, stderr.String())
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module sqlchoicefixture\n\ngo 1.26.2\n\nrequire cloud.google.com/go/spanner v1.95.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	compile := exec.Command("go", "build", ".")
	compile.Dir = dir
	compile.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile generated SDK-facing builder: %v\n%s", err, output)
	}
	plan, err := querygen.BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatal(err)
	}
	targets := querygen.ExpandPlanReportQueries(plan.Queries)
	if len(targets) != 6 {
		t.Fatalf("PLAN targets = %d", len(targets))
	}
	for _, target := range targets {
		values, err := planReportParamValues(target.Params)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := values["bounds"]; ok {
			t.Fatal("SQL selector became a bind parameter")
		}
		if _, ok := values["flag"]; ok {
			t.Fatal("column selector became a bind parameter")
		}
		for _, name := range target.ReportVariant.AbsentParams {
			if _, ok := values[name]; ok {
				t.Fatalf("omitted bind %q was supplied in %s", name, target.ReportVariant.Label)
			}
		}
	}
	values, err := planReportParamValues([]querygen.QueryCodegenParam{{Name: "selector", Optional: " SQL_CHOICE "}})
	if err != nil || values != nil {
		t.Fatalf("selector-only params = %v, %v", values, err)
	}
	schema, err := configSchemaBytes("json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatal(err)
	}
	properties := doc["$defs"].(map[string]interface{})["query_param"].(map[string]interface{})["properties"].(map[string]interface{})
	enum := properties["optional"].(map[string]interface{})["enum"].([]interface{})
	if !slices.Contains(interfaceStrings(enum), "sql_choice") {
		t.Fatal("config schema excludes sql_choice")
	}
	if !strings.Contains(string(schema), "v1alpha") {
		t.Fatal("preview schema version disappeared")
	}
}
