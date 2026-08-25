package spannertype

import (
	"testing"

	"github.com/cloudspannerecosystem/memefish/ast"
)

func TestParseSchemaType(t *testing.T) {
	tests := []struct {
		input   string
		wantSQL string
	}{
		{"BOOL", "BOOL"},
		{"INT64", "INT64"},
		{"FLOAT32", "FLOAT32"},
		{"FLOAT64", "FLOAT64"},
		{"NUMERIC", "NUMERIC"},
		{"STRING(MAX)", "STRING(MAX)"},
		{"STRING(100)", "STRING(100)"},
		{"BYTES(MAX)", "BYTES(MAX)"},
		{"BYTES(256)", "BYTES(256)"},
		{"DATE", "DATE"},
		{"TIMESTAMP", "TIMESTAMP"},
		{"JSON", "JSON"},
		{"ARRAY<INT64>", "ARRAY<INT64>"},
		{"ARRAY<STRING(MAX)>", "ARRAY<STRING(MAX)>"},
		{"ARRAY<FLOAT64>", "ARRAY<FLOAT64>"},
		{"PROTO<examples.shipping.Order>", "examples.shipping.`Order`"},
		{"ENUM<examples.shipping.Order.Status>", "examples.shipping.`Order`.Status"},
		{"ARRAY<PROTO<examples.shipping.Order>>", "ARRAY<examples.shipping.`Order`>"},
		{"ARRAY<ENUM<examples.shipping.Order.Status>>", "ARRAY<examples.shipping.`Order`.Status>"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			schemaType, err := ParseSchemaType(tt.input)
			if err != nil {
				t.Fatalf("ParseSchemaType(%q) error: %v", tt.input, err)
			}
			if schemaType == nil {
				t.Fatalf("ParseSchemaType(%q) returned nil", tt.input)
			}
			got := schemaType.SQL()
			if got != tt.wantSQL {
				t.Errorf("ParseSchemaType(%q).SQL() = %q, want %q", tt.input, got, tt.wantSQL)
			}
		})
	}
}

func TestParseSchemaType_Roundtrip(t *testing.T) {
	types := []string{
		"BOOL", "INT64", "FLOAT64", "STRING(MAX)", "STRING(255)",
		"BYTES(MAX)", "DATE", "TIMESTAMP", "JSON", "NUMERIC",
		"ARRAY<INT64>", "ARRAY<STRING(MAX)>",
	}

	for _, input := range types {
		t.Run(input, func(t *testing.T) {
			st, err := ParseSchemaType(input)
			if err != nil {
				t.Fatalf("ParseSchemaType(%q) error: %v", input, err)
			}

			// Roundtrip: SQL() -> Parse again
			sql := st.SQL()
			st2, err := ParseSchemaType(sql)
			if err != nil {
				t.Fatalf("ParseSchemaType(%q) roundtrip error: %v", sql, err)
			}

			if st2.SQL() != sql {
				t.Errorf("roundtrip mismatch: %q -> %q -> %q", input, sql, st2.SQL())
			}
		})
	}
}

func TestParseSchemaType_Error(t *testing.T) {
	// Syntax errors should fail (not unknown type names, since memefish accepts NamedType for PROTO types)
	_, err := ParseSchemaType("STRING(")
	if err == nil {
		t.Error("expected error for malformed type, got nil")
	}
}

func TestParseSchemaType_TypeAssertion(t *testing.T) {
	// Scalar
	st, _ := ParseSchemaType("BOOL")
	if _, ok := st.(*ast.ScalarSchemaType); !ok {
		t.Errorf("BOOL should be ScalarSchemaType, got %T", st)
	}

	// Sized
	st, _ = ParseSchemaType("STRING(MAX)")
	if _, ok := st.(*ast.SizedSchemaType); !ok {
		t.Errorf("STRING(MAX) should be SizedSchemaType, got %T", st)
	}

	// Array
	st, _ = ParseSchemaType("ARRAY<INT64>")
	if _, ok := st.(*ast.ArraySchemaType); !ok {
		t.Errorf("ARRAY<INT64> should be ArraySchemaType, got %T", st)
	}
}

func TestParseFunctionType(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "INTERVAL", want: "INTERVAL"},
		{input: "STRING", want: "STRING"},
		{input: "BYTES", want: "BYTES"},
		{input: "STRUCT<value INT64>", want: "STRUCT<value INT64>"},
		{input: "ARRAY<STRUCT<value INT64>>", want: "ARRAY<STRUCT<value INT64>>"},
		{input: "PROTO<examples.Message>", want: "examples.Message"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, err := ParseFunctionType(test.input)
			if err != nil {
				t.Fatalf("ParseFunctionType(%q): %v", test.input, err)
			}
			if got.SQL() != test.want {
				t.Errorf("ParseFunctionType(%q).SQL() = %q, want %q", test.input, got.SQL(), test.want)
			}
		})
	}
}
