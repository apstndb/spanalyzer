package astconv

import (
	"strings"
	"testing"

	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func TestModelOptionRoundTripPreservesArraysAndEscapes(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "array",
			sql:  "CREATE MODEL M INPUT (x FLOAT64) OUTPUT (y FLOAT64) REMOTE OPTIONS (endpoints = ['https://example.com/a', 'https://example.com/b'])",
		},
		{
			name: "escaped backslash",
			sql:  "CREATE MODEL M INPUT (x FLOAT64) OUTPUT (y FLOAT64) REMOTE OPTIONS (endpoint = 'https://example.com/a\\\\b')",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ddls, err := memefish.ParseDDLs("", test.sql)
			if err != nil {
				t.Fatalf("ParseDDLs: %v", err)
			}
			schema, err := FromDDLStatements(ddls)
			if err != nil {
				t.Fatalf("FromDDLStatements: %v", err)
			}
			if len(schema.ModelOptions) != 1 {
				t.Fatalf("model options = %d, want 1", len(schema.ModelOptions))
			}
			if test.name == "array" && schema.ModelOptions[0].OptionType != "ARRAY" {
				t.Fatalf("option type = %q, want ARRAY", schema.ModelOptions[0].OptionType)
			}
			out, err := schema.ToDDLStatements()
			if err != nil {
				t.Fatalf("ToDDLStatements: %v", err)
			}
			again, err := FromDDLStatements(out)
			if err != nil {
				t.Fatalf("FromDDLStatements(round trip): %v", err)
			}
			if len(again.ModelOptions) != 1 {
				t.Fatalf("round-trip options = %d, want 1", len(again.ModelOptions))
			}
			if again.ModelOptions[0].OptionType != schema.ModelOptions[0].OptionType || again.ModelOptions[0].OptionValue != schema.ModelOptions[0].OptionValue {
				t.Fatalf("option = %+v, want %+v\nSQL: %s", again.ModelOptions[0], schema.ModelOptions[0], out[0].SQL())
			}
			if strings.Contains(out[0].SQL(), `a\\\\b`) {
				t.Fatalf("SQL() doubled the escape: %s", out[0].SQL())
			}
		})
	}
}

func TestOptionStringMetadataPreservesContent(t *testing.T) {
	for _, value := range []string{"123", "true", "false", "[1]", "NULL", "-42", "1.5", `a\b`, "'leading", "trailing'"} {
		t.Run(value, func(t *testing.T) {
			expr := parseOptionValue("STRING", value)
			literal, ok := expr.(*ast.StringLiteral)
			if !ok || literal.Value != value {
				t.Fatalf("STRING %q reconstructed as %T %s", value, expr, expr.SQL())
			}
		})
	}
}

func TestOptionLiteralRoundTrip(t *testing.T) {
	for _, sql := range []string{
		`'123'`, `'true'`, `'[1]'`, `'NULL'`, `'a\\b'`, `'it\'s quoted'`,
		`r'a\b'`, `'''first
second'''`, `b'\x41'`, `-42`, `+42`, `-1.5`, `true`, `NULL`,
		`[]`, `['a\\b', 'it\'s quoted', NULL]`, `[-1, 2, NULL]`, `ARRAY<STRING>['a']`,
	} {
		t.Run(sql, func(t *testing.T) {
			expr, err := memefish.ParseExpr("", sql)
			if err != nil {
				t.Fatal(err)
			}
			kind := inferOptionType(expr)
			got := parseOptionValue(kind, expr.SQL())
			if got.SQL() != expr.SQL() || inferOptionType(got) != kind {
				t.Fatalf("%s %s round-tripped to %s %s", kind, expr.SQL(), inferOptionType(got), got.SQL())
			}
		})
	}
	for _, kind := range []string{"ARRAY", "ARRAY<STRING>", "array<string>"} {
		if _, ok := parseOptionValue(kind, `['one', 'two']`).(*ast.ArrayLiteral); !ok {
			t.Fatalf("%s metadata did not reconstruct as ARRAY", kind)
		}
	}
	for _, kind := range []string{"NULL", "BOOL", "INT64", "ARRAY<STRING>"} {
		if _, ok := parseOptionValue(kind, "NULL").(*ast.NullLiteral); !ok {
			t.Fatalf("%s NULL metadata did not reconstruct as NULL", kind)
		}
	}
}
