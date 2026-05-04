package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/go-googlesql-spanner-poc"
	"github.com/goccy/go-yaml"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRunModeSpannerTypeExpressionReturnsType(t *testing.T) {
	analyzer, err := spanalyzer.NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}

	out, err := runMode(analyzer, "spanner_type", "expression", "1", "json")
	if err != nil {
		t.Fatalf("runMode() error = %v", err)
	}
	if strings.Contains(out, "fields") {
		t.Fatalf("runMode() output contains row fields, want scalar Type:\n%s", out)
	}
	var typ spannerpb.Type
	if err := protojson.Unmarshal([]byte(out), &typ); err != nil {
		t.Fatalf("unmarshal Type output: %v\n%s", err, out)
	}
	if got, want := typ.Code, spannerpb.TypeCode_INT64; got != want {
		t.Fatalf("typ.Code = %s, want %s", got, want)
	}
}

func TestRunModesDefaultReturnsSpannerType(t *testing.T) {
	analyzer, err := spanalyzer.NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}

	out, err := runModes(analyzer, nil, "expression", "1", defaultOutputFormat)
	if err != nil {
		t.Fatalf("runModes() error = %v", err)
	}
	if strings.Contains(out, "{") {
		t.Fatalf("runModes() output looks like JSON, want default YAML:\n%s", out)
	}
	jsonOut, err := yaml.YAMLToJSON([]byte(out))
	if err != nil {
		t.Fatalf("YAMLToJSON() error = %v\n%s", err, out)
	}
	var typ spannerpb.Type
	if err := protojson.Unmarshal(jsonOut, &typ); err != nil {
		t.Fatalf("unmarshal Type output: %v\n%s", err, out)
	}
	if got, want := typ.Code, spannerpb.TypeCode_INT64; got != want {
		t.Fatalf("typ.Code = %s, want %s", got, want)
	}
}

func TestRunBigQueryModesDefaultReturnsBigQueryType(t *testing.T) {
	analyzer, err := spanalyzer.NewBigQueryAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}

	out, err := runBigQueryModes(analyzer, nil, "query", "SELECT 1 AS n", defaultOutputFormat)
	if err != nil {
		t.Fatalf("runBigQueryModes() error = %v", err)
	}
	if strings.Contains(out, "{") {
		t.Fatalf("runBigQueryModes() output looks like JSON, want default YAML:\n%s", out)
	}
	if !strings.Contains(out, `name: "n"`) || !strings.Contains(out, "type: INTEGER") {
		t.Fatalf("runBigQueryModes() output = %q, want BigQuery TableSchema YAML", out)
	}
}

func TestRunBigQueryModeJSON(t *testing.T) {
	analyzer, err := spanalyzer.NewBigQueryAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}

	out, err := runBigQueryMode(analyzer, "bigquery_type", "expression", "IF(TRUE, 1, 2)", "json")
	if err != nil {
		t.Fatalf("runBigQueryMode() error = %v", err)
	}
	if !strings.Contains(out, `"type": "INTEGER"`) {
		t.Fatalf("runBigQueryMode() output = %q, want JSON containing INTEGER type", out)
	}
}

func TestRunModeSpannerTypeYAML(t *testing.T) {
	analyzer, err := spanalyzer.NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}

	out, err := runMode(analyzer, "spanner_type", "expression", "1", "yaml")
	if err != nil {
		t.Fatalf("runMode() error = %v", err)
	}
	if !strings.Contains(out, "code: INT64") {
		t.Fatalf("runMode() output = %q, want YAML containing code: INT64", out)
	}
	if strings.Contains(out, "{") {
		t.Fatalf("runMode() output looks like JSON, want YAML:\n%s", out)
	}
}

func TestRunModeAnalyzeReturnsResolvedAST(t *testing.T) {
	analyzer, err := spanalyzer.NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}

	out, err := runMode(analyzer, "analyze", "query", "SELECT 1 AS n", "json")
	if err != nil {
		t.Fatalf("runMode() error = %v", err)
	}
	if strings.Contains(out, `"fields"`) {
		t.Fatalf("runMode() output contains Spanner row type, want resolved AST:\n%s", out)
	}
	if !strings.Contains(out, "QueryStmt") {
		t.Fatalf("runMode() output = %q, want resolved AST debug string containing QueryStmt", out)
	}
}

func TestReadDDLEmptyPath(t *testing.T) {
	path, ddl, err := readDDL("")
	if err != nil {
		t.Fatalf("readDDL() error = %v", err)
	}
	if path == "" {
		t.Fatal("readDDL() path is empty, want synthetic path")
	}
	if ddl != "" {
		t.Fatalf("readDDL() ddl = %q, want empty", ddl)
	}
}

func TestReadDDLFile(t *testing.T) {
	ddlPath := filepath.Join(t.TempDir(), "schema.sql")
	const ddl = "CREATE TABLE T (Id INT64 NOT NULL) PRIMARY KEY(Id);"
	if err := os.WriteFile(ddlPath, []byte(ddl), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	path, gotDDL, err := readDDL(ddlPath)
	if err != nil {
		t.Fatalf("readDDL() error = %v", err)
	}
	if path != ddlPath {
		t.Fatalf("readDDL() path = %q, want %q", path, ddlPath)
	}
	if gotDDL != ddl {
		t.Fatalf("readDDL() ddl = %q, want %q", gotDDL, ddl)
	}
}
