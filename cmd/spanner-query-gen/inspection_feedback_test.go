package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/goccy/go-yaml"
)

func mixedInspectionConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `CREATE TABLE T (Id INT64 NOT NULL) PRIMARY KEY (Id)`)
	path := filepath.Join(dir, "querygen.yaml")
	writeTestFile(t, path, `version: v1alpha
go: {package: db}
catalogs:
- {name: app, kind: spanner, ddl: schema.sql}
queries:
- name: Bad
  catalog: app
  kind: sql
  sql: SELECT * FROM UNNEST(1)
  result: {struct: BadRow}
- name: Good
  catalog: app
  kind: sql
  sql: SELECT Id FROM T
  result: {struct: GoodRow}
- name: AnotherBad
  catalog: app
  kind: sql
  sql: SELECT Missing FROM T
  result: {struct: AnotherBadRow}
- name: Count
  catalog: app
  kind: sql
  sql: SELECT COUNT(*) AS Total FROM T
  result: {struct: CountRow}
`)
	return path
}

func TestExplainPlanContinuesAfterAnalysisErrors(t *testing.T) {
	for _, output := range []string{"json", "yaml", "summary"} {
		t.Run(output, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run([]string{"explain-plan", "--config", mixedInspectionConfig(t), "--output", output}, &stdout, &stderr)
			if err == nil {
				t.Fatal("partial inspection must return nonzero")
			}
			if output == "summary" {
				for _, want := range []string{"Query: Bad", "Error:", "Query: Good", "Id: INT64", "Query: AnotherBad", "Query: Count", "Total: INT64"} {
					if !strings.Contains(stdout.String(), want) {
						t.Fatalf("missing %q:\n%s", want, stdout.String())
					}
				}
				return
			}
			data := stdout.Bytes()
			if output == "yaml" {
				var err error
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					t.Fatal(err)
				}
			}
			var report struct {
				Queries []struct {
					Name, Status, Error string
					Fields              []struct{ Name string }
				}
				PlanningErrors []vetDiagnostic `json:"planning_errors"`
			}
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatalf("decode partial report: %v\n%s", err, data)
			}
			if len(report.Queries) != 4 || len(report.PlanningErrors) != 2 {
				t.Fatalf("report = %+v", report)
			}
			for i, q := range report.Queries {
				if i%2 == 0 {
					if q.Status != "error" || q.Error == "" || len(q.Fields) != 0 {
						t.Fatalf("failed target = %+v", q)
					}
				} else if q.Status != "ok" || len(q.Fields) != 1 {
					t.Fatalf("successful target = %+v", q)
				}
			}
		})
	}
}

func TestVetContinuesAndRetainsWarnings(t *testing.T) {
	for _, output := range []string{"json", "yaml", "text"} {
		t.Run(output, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "querygen.yaml")
			writeExternalDatasetFixture(t, dir, path, false)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			data = bytes.Replace(data, []byte("queries:\n"), []byte("queries:\n- name: Bad\n  catalog: app\n  kind: sql\n  sql: SELECT * FROM UNNEST(1)\n  result: {struct: BadRow}\n"), 1)
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			if err := run([]string{"vet", "--config", path, "--output", output}, &stdout, &stderr); err == nil {
				t.Fatal("invalid target must fail vet")
			}
			if output == "text" {
				if stdout.Len() != 0 || !strings.Contains(stderr.String(), "error: query Bad") || !strings.Contains(stderr.String(), "external-dataset-data-boost-permission-note") {
					t.Fatalf("stdout=%s\nstderr=%s", &stdout, &stderr)
				}
				return
			}
			if stderr.Len() != 0 {
				t.Fatalf("structured vet wrote stderr: %s", &stderr)
			}
			data = stdout.Bytes()
			if output == "yaml" {
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					t.Fatal(err)
				}
			}
			var report vetReport
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, d := range report.Diagnostics {
				if d.Scope == "query" && d.Name == "Bad" && d.Severity == "error" && !d.Suppressible && !d.Suppressed {
					found = true
				}
			}
			if !found || len(report.Warnings) == 0 || len(report.Suppressions) == 0 {
				t.Fatalf("lost error/warnings/suppressions: %+v", report)
			}
		})
	}
}

func TestGenerationStillRejectsMixedInspectionInput(t *testing.T) {
	path := mixedInspectionConfig(t)
	for _, command := range []string{"generate", "check"} {
		var out, stderr bytes.Buffer
		if err := run([]string{command, "--config", path}, &out, &stderr); err == nil || out.Len() != 0 {
			t.Fatalf("%s emitted partial code: %v, %s", command, err, &out)
		}
	}
}

func TestInspectionPreservesGlobalValidationErrors(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `CREATE TABLE T (Id INT64 NOT NULL) PRIMARY KEY (Id)`)
	config := querygen.QueryCodegenConfig{Package: "db", Client: querygen.GoStructTargetSpanner,
		Schemas: []querygen.QueryCodegenSchema{{Name: "app", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []querygen.QueryCodegenQuery{
			{Name: "First", Catalog: "app", SQL: "SELECT Id FROM T", ResultStruct: "SharedRow"},
			{Name: "Second", Catalog: "app", SQL: "SELECT CAST(Id AS STRING) AS Id FROM T", ResultStruct: "SharedRow"},
		},
	}
	inspection := buildInspectionPlan(config, dir)
	if inspection.cause == nil || len(inspection.Errors) != 1 || inspection.Errors[0].Scope != "plan" || len(inspection.Plan.Queries) != 2 {
		t.Fatalf("global validation was lost: %+v", inspection)
	}
	for _, q := range inspection.Plan.Queries {
		if q.PlanningError != "" || len(q.Fields) != 1 {
			t.Fatalf("individual result was lost: %+v", q)
		}
	}
	config.Queries = config.Queries[:1]
	config.Schemas = append(config.Schemas, querygen.QueryCodegenSchema{Name: "unrelated", Dialect: "spanner", DDL: "missing.sql"})
	inspection = buildInspectionPlan(config, dir)
	if inspection.cause == nil || len(inspection.Errors) != 1 || inspection.Errors[0].Scope != "plan" || len(inspection.Plan.Queries[0].Fields) != 1 {
		t.Fatalf("catalog failure hid valid result or disappeared: %+v", inspection)
	}
}

func TestInspectionKeepsWritesSeparateFromQueryTargets(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `CREATE TABLE T (Id INT64 NOT NULL) PRIMARY KEY (Id)`)
	config := querygen.QueryCodegenConfig{Package: "db", Client: querygen.GoStructTargetSpanner,
		Schemas: []querygen.QueryCodegenSchema{{Name: "app", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []querygen.QueryCodegenQuery{{Name: "Good", Catalog: "app", SQL: "SELECT Id FROM T", ResultStruct: "Row"}},
		Writes: []querygen.QueryCodegenWrite{
			{Name: "BadWrite", Catalog: "app", Table: "Missing", Operation: "delete", InputStruct: "BadInput", Methods: []string{"dml"}},
			{Name: "GoodWrite", Catalog: "app", Table: "T", Operation: "delete", InputStruct: "GoodInput", Methods: []string{"mutation"}},
		},
	}
	inspection := buildInspectionPlan(config, dir)
	if inspection.cause == nil || len(inspection.Errors) != 1 || inspection.Errors[0].Scope != "write" || inspection.Errors[0].Name != "BadWrite" || len(inspection.Plan.Queries) != 1 || len(inspection.Plan.Writes) != 1 || inspection.Plan.Writes[0].Name != "GoodWrite" {
		t.Fatalf("write/query scopes: %+v, plan=%+v", inspection, inspection.Plan)
	}
}

type inspectionFailWriter struct{ err error }

func (w inspectionFailWriter) Write([]byte) (int, error) { return 0, w.err }

func TestInspectionDoesNotHideOutputFailures(t *testing.T) {
	path := mixedInspectionConfig(t)
	sentinel := errors.New("output unavailable")
	for _, command := range []string{"explain-plan", "vet"} {
		err := run([]string{command, "--config", path, "--output", "json"}, inspectionFailWriter{sentinel}, &bytes.Buffer{})
		if !errors.Is(err, sentinel) {
			t.Fatalf("%s error = %v, want output failure", command, err)
		}
	}
}
