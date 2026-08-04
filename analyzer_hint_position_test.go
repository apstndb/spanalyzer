package spanalyzer

import (
	"strings"
	"testing"
)

func TestGoogleSQLHintPositionsParseAndRoundTrip(t *testing.T) {
	catalog, err := BuildGoogleSQLCatalogFromDDL("schema.sql", "", nil)
	if err != nil {
		t.Fatalf("BuildGoogleSQLCatalogFromDDL() error = %v", err)
	}
	helper := catalog.Helper()

	tests := []struct {
		name string
		sql  string
	}{
		{"statement", `@{a=1} SELECT 1`},
		{"select", `SELECT @{a=1} 1`},
		{"order_by", `SELECT 1 ORDER @{a=1} BY 1`},
		{"set_operation_first", `SELECT 1 UNION @{a=1} ALL SELECT 2`},
		// The parser preserves this hint in the AST; the analyzer enforces the
		// same-level set-operation restriction in the focused test below.
		{"set_operation_second_same_level", `SELECT 1 UNION ALL SELECT 2 UNION @{a=1} ALL SELECT 3`},
		{"sql_exists", `SELECT EXISTS @{a=1} (SELECT 1)`},
		{"in_subquery", `SELECT 1 IN @{a=1} (SELECT 1)`},
		{"like_any_subquery", `SELECT 'x' LIKE ANY @{a=1} (SELECT 'x')`},
		{"like_some_subquery", `SELECT 'x' LIKE SOME @{a=1} (SELECT 'x')`},
		{"like_all_subquery", `SELECT 'x' LIKE ALL @{a=1} (SELECT 'x')`},
		{"like_some_multi_hint_subquery", `SELECT 'x' LIKE SOME @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT 'x')`},
		{"like_all_multi_hint_subquery", `SELECT 'x' LIKE ALL @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT 'x')`},
		{"quantified_subquery", `SELECT 1 = ANY @{a=1} (SELECT 1)`},
		{"quantified_some_subquery", `SELECT 1 = SOME @{a=1} (SELECT 1)`},
		{"quantified_all_subquery", `SELECT 1 = ALL @{a=1} (SELECT 1)`},
		{"quantified_some_multi_hint_subquery", `SELECT 1 = SOME @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT 1)`},
		{"quantified_all_multi_hint_subquery", `SELECT 1 = ALL @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT 1)`},
		{"partition_by", `SELECT ROW_NUMBER() OVER (PARTITION @{a=1} BY 1)`},
		{"function_call", `SELECT ABS(1) @{a=1}`},
		{"tvf", `SELECT * FROM MissingTVF() @{a=1}`},
		{"gql_return", `GRAPH FinGraph MATCH (p:Person) RETURN @{a=1} p.id`},
		{"gql_order_by", `GRAPH FinGraph MATCH (p:Person) RETURN p.id ORDER @{a=1} BY p.id`},
		{"gql_with", `GRAPH FinGraph MATCH (p:Person) WITH @{a=1} p RETURN p.id`},
		{"gql_value", `GRAPH FinGraph MATCH (p:Person) RETURN VALUE @{a=1} { MATCH (q:Person) RETURN q.id LIMIT 1 } AS v`},
		{"gql_exists", `GRAPH FinGraph MATCH (p:Person) RETURN EXISTS @{a=1} { MATCH (q:Person) RETURN q.id } AS v`},
		{"pipe_log", `SELECT 1 |> LOG @{a=1}`},
		{"insert_target", `INSERT INTO T @{a=1} (id) VALUES (1)`},
		{"delete_target", `DELETE FROM T @{a=1} WHERE TRUE`},
		{"update_target", `UPDATE T @{a=1} SET id = 1 WHERE TRUE`},
		{"dml_statement_pdml", `@{PDML_MAX_PARALLELISM=1} UPDATE T SET id = id WHERE FALSE`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			debug, err := helper.ParseDebugString("query", tt.sql)
			if err != nil {
				t.Fatalf("ParseDebugString() error = %v", err)
			}
			if !strings.Contains(debug, "Hint") {
				t.Fatalf("parse tree does not retain a hint node:\n%s", debug)
			}

			unparsed, err := helper.Unparse("query", tt.sql)
			if err != nil {
				t.Fatalf("Unparse() error = %v", err)
			}
			if !strings.Contains(unparsed, "@{") {
				t.Fatalf("unparsed SQL does not retain the hint: %s", unparsed)
			}
			if _, err := helper.Parse("query", unparsed); err != nil {
				t.Fatalf("Parse(unparsed) error = %v\nunparsed SQL: %s", err, unparsed)
			}
		})
	}
}

func TestGoogleSQLHintPositionsRejectedByParser(t *testing.T) {
	catalog, err := BuildGoogleSQLCatalogFromDDL("schema.sql", "", nil)
	if err != nil {
		t.Fatalf("BuildGoogleSQLCatalogFromDDL() error = %v", err)
	}
	helper := catalog.Helper()

	tests := []struct {
		name    string
		sql     string
		wantErr string
	}{
		{"in_value_list", `SELECT 1 IN @{a=1} (1, 2)`, "IN clause with value list"},
		{"in_unnest", `SELECT 1 IN @{a=1} UNNEST([1])`, "IN clause with UNNEST"},
		{"like_any_value_list", `SELECT 'x' LIKE ANY @{a=1} ('x')`, "LIKE clause with value list"},
		{"like_some_value_list", `SELECT 'x' LIKE SOME @{a=1} ('x')`, "LIKE clause with value list"},
		{"like_all_value_list", `SELECT 'x' LIKE ALL @{a=1} ('x')`, "LIKE clause with value list"},
		{"quantified_value_list", `SELECT 1 = ANY @{a=1} (1)`, "ANY/SOME/ALL clause with value list"},
		{"quantified_some_value_list", `SELECT 1 = SOME @{a=1} (1)`, "ANY/SOME/ALL clause with value list"},
		{"quantified_all_value_list", `SELECT 1 = ALL @{a=1} (1)`, "ANY/SOME/ALL clause with value list"},
		{"gql_set_operation", `GRAPH FinGraph MATCH (p:Person) RETURN p.id UNION @{a=1} ALL MATCH (q:Person) RETURN q.id`, "Expected keyword ALL or keyword DISTINCT"},
		{"gql_subpath_leading_hint", `GRAPH FinGraph MATCH ( @{a=1} (p:Person) ) RETURN p.id`, "Hint cannot be used at beginning of path pattern"},
		{"gql_edge_edge_hint", `GRAPH FinGraph MATCH (a:Person)-[e]->@{a=1}-[f]->(b:Person) RETURN a.id`, "Hint cannot be used in between two GraphEdgePatterns"},
		{"pipe_finish", `SELECT 1 |> FINISH @{a=1}`, "Expected keyword JOIN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := helper.Parse("query", tt.sql)
			if err == nil {
				t.Fatal("Parse() succeeded, want rejection")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Parse() error = %q, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestGoogleSQLSetOperationHintAfterFirstOperationRejectedByAnalyzer(t *testing.T) {
	catalog, err := BuildGoogleSQLCatalogFromDDL("schema.sql", "", nil)
	if err != nil {
		t.Fatalf("BuildGoogleSQLCatalogFromDDL() error = %v", err)
	}
	helper := catalog.Helper()
	const sql = `SELECT 1 UNION ALL SELECT 2 UNION @{a=1} ALL SELECT 3`

	if _, err := helper.Parse("query", sql); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	_, err = helper.AnalyzeStatement(sql)
	if err == nil {
		t.Fatal("AnalyzeStatement() succeeded, want same-level set-operation hint rejection")
	}
	if want := "Hints on set operations must appear on the first"; !strings.Contains(err.Error(), want) {
		t.Fatalf("AnalyzeStatement() error = %q, want substring %q", err, want)
	}
}
