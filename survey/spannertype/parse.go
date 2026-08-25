// Package spannertype provides a parser for Spanner type strings.
package spannertype

import (
	"fmt"
	"strings"

	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

// ParseSchemaType parses a SPANNER_TYPE string (e.g. "STRING(MAX)",
// "ARRAY<INT64>", or "PROTO<example.Message>") into an ast.SchemaType node.
//
// It delegates to memefish.ParseSchemaType, which is exported since memefish v0.8.0.
// INFORMATION_SCHEMA wraps protocol-buffer and enum names in PROTO<...> and
// ENUM<...>, while Spanner DDL uses the enclosed named type directly. Normalize
// those metadata-only wrappers before parsing, including inside ARRAY.
func ParseSchemaType(s string) (ast.SchemaType, error) {
	return memefish.ParseSchemaType("", normalizeNamedTypeWrappers(strings.TrimSpace(s)))
}

// ParseFunctionType parses a Spanner function parameter or return type. The
// function grammar additionally permits types such as INTERVAL and unsized
// STRING/BYTES, which are not valid column schema types.
func ParseFunctionType(s string) (ast.SchemaType, error) {
	typeText := normalizeNamedTypeWrappers(strings.TrimSpace(s))
	ddl, err := memefish.ParseDDL(
		"",
		"CREATE FUNCTION __type_probe(__value "+typeText+") RETURNS BOOL AS (TRUE)",
	)
	if err != nil {
		return nil, err
	}
	function, ok := ddl.(*ast.CreateFunction)
	if !ok || len(function.Params) != 1 || function.Params[0].Type == nil {
		return nil, fmt.Errorf("failed to extract function type")
	}
	return function.Params[0].Type, nil
}

func normalizeNamedTypeWrappers(s string) string {
	for _, prefix := range []string{"PROTO<", "ENUM<"} {
		if strings.HasPrefix(s, prefix) && strings.HasSuffix(s, ">") {
			return strings.TrimSpace(s[len(prefix) : len(s)-1])
		}
	}
	if strings.HasPrefix(s, "ARRAY<") && strings.HasSuffix(s, ">") {
		item := strings.TrimSpace(s[len("ARRAY<") : len(s)-1])
		return "ARRAY<" + normalizeNamedTypeWrappers(item) + ">"
	}
	return s
}
