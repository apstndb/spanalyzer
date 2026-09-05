package querygen

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildQueryCodegenPlanRejectsUnsupportedClient(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), "CREATE TABLE T (Id INT64) PRIMARY KEY (Id);")
	config := QueryCodegenConfig{
		Package: "db", Client: "unsupported",
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{Name: "GetID", Catalog: "spanner", SQL: "SELECT Id FROM T"}},
	}
	_, genErr := GenerateQueryCode(config, dir)
	_, planErr := BuildQueryCodegenPlan(config, dir)
	if genErr == nil || !strings.Contains(genErr.Error(), "unsupported Go struct target") {
		t.Fatalf("GenerateQueryCode() error = %v", genErr)
	}
	if planErr == nil || planErr.Error() != genErr.Error() {
		t.Fatalf("BuildQueryCodegenPlan() error = %v, want %v", planErr, genErr)
	}
}
