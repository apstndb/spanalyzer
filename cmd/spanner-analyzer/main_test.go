package main

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/go-googlesql-spanner-poc"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRunModeAnalyzeExpressionReturnsType(t *testing.T) {
	analyzer, err := spanalyzer.NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}

	out, err := runMode(analyzer, "analyze", "expression", "1", "json")
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
