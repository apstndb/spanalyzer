package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/go-googlesql-spanner-poc"
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

	out, err := runModes(analyzer, nil, "expression", "1", "json")
	if err != nil {
		t.Fatalf("runModes() error = %v", err)
	}
	var typ spannerpb.Type
	if err := protojson.Unmarshal([]byte(out), &typ); err != nil {
		t.Fatalf("unmarshal Type output: %v\n%s", err, out)
	}
	if got, want := typ.Code, spannerpb.TypeCode_INT64; got != want {
		t.Fatalf("typ.Code = %s, want %s", got, want)
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
