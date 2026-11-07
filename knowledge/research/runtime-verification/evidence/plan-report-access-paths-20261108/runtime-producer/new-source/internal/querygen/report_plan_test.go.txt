package querygen

import (
	"path/filepath"
	"testing"
)

func TestReportPlanIsolatesStaticAndCatalogFailures(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "good.sql"), "CREATE TABLE T (Id INT64 NOT NULL) PRIMARY KEY (Id)")
	config := QueryCodegenConfig{Package: "db", Schemas: []QueryCodegenSchema{
		{Name: "good", Dialect: "spanner", DDL: "good.sql"},
		{Name: "bad", Dialect: "spanner", DDL: "missing.sql"},
	}, Queries: []QueryCodegenQuery{
		{Name: "WrongLimit", Catalog: "good", SQL: "SELECT Id FROM T LIMIT @n", ResultStruct: "WrongRow", Params: []QueryCodegenParam{{Name: "n", Type: "STRING"}}},
		{Name: "Good", Catalog: "good", SQL: "SELECT Id FROM T", ResultStruct: "GoodRow"},
		{Name: "BadCatalog", Catalog: "bad", SQL: "SELECT Id FROM T", ResultStruct: "OtherRow"},
	}}
	plan := BuildQueryCodegenReportPlan(config, dir)
	if len(plan.Queries) != 3 || plan.Queries[0].PlanningError == "" || plan.Queries[1].PlanningError != "" || plan.Queries[2].PlanningError == "" {
		t.Fatalf("report targets = %#v", plan.Queries)
	}
	if _, err := BuildQueryCodegenPlan(config, dir); err == nil {
		t.Fatal("strict generation must still reject the invalid config")
	}
}

func TestReportPlanIncludesGeneratedDMLWritesAndSkipsMutations(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), "CREATE TABLE T (Id INT64 NOT NULL, Value STRING(MAX)) PRIMARY KEY (Id)")
	config := QueryCodegenConfig{Package: "db", Schemas: []QueryCodegenSchema{{Name: "app", Dialect: "spanner", DDL: "schema.sql"}}, Writes: []QueryCodegenWrite{
		{Name: "Remove", Catalog: "app", Table: "T", Operation: "delete", InputStruct: "RemoveInput", Methods: []string{"dml"}},
		{Name: "Replace", Catalog: "app", Table: "T", Operation: "replace", InputStruct: "ReplaceInput", Methods: []string{"mutation"}},
	}}
	plan := BuildQueryCodegenReportPlan(config, dir)
	if len(plan.Queries) != 2 {
		t.Fatalf("targets = %#v", plan.Queries)
	}
	dml, mutation := plan.Queries[0], plan.Queries[1]
	if dml.PlanningError != "" || dml.ReportScope != "write" || dml.Result != "row_count" || dml.SQL == "" || len(dml.Params) != 1 {
		t.Fatalf("DML target = %#v", dml)
	}
	if mutation.PlanningError != "" || mutation.ReportSkipReason == "" {
		t.Fatalf("mutation target = %#v", mutation)
	}
}

func TestPublicDMLRowCountDoesNotRequireRowStruct(t *testing.T) {
	config, err := ParseQueryCodegenConfigYAML([]byte(`version: v1alpha
go: {package: db}
catalogs:
- {name: app, kind: spanner, ddl: schema.sql}
queries:
- name: Remove
  catalog: app
  kind: sql
  sql: DELETE FROM T WHERE Id = @id
  params: [{name: id, type: INT64}]
  result: {cardinality: row_count}
`))
	if err != nil || len(config.Queries) != 1 || config.Queries[0].Result != "row_count" {
		t.Fatalf("DML config = %#v, %v", config, err)
	}
}
