package astconv

import (
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func parseDDLs(t *testing.T, ddls ...string) []ast.DDL {
	t.Helper()

	statements := make([]ast.DDL, 0, len(ddls))
	for _, ddl := range ddls {
		statement, err := memefish.ParseDDL("", ddl)
		if err != nil {
			t.Fatalf("ParseDDL(%q): %v", ddl, err)
		}
		statements = append(statements, statement)
	}
	return statements
}

func ddlSQL(t *testing.T, ddls []ast.DDL) string {
	t.Helper()

	statements := make([]string, 0, len(ddls))
	for _, ddl := range ddls {
		statements = append(statements, ddl.SQL())
	}
	return strings.Join(statements, "\n")
}

func TestRoundtrip_ColumnScopedPrivileges(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"CREATE ROLE analyst",
		"GRANT SELECT(secret), INSERT(write_only), UPDATE(modified_at) ON TABLE Records TO ROLE analyst",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.RoleTableGrants) != 0 {
		t.Fatalf("RoleTableGrants = %d, want 0 for column-scoped grants", len(schema.RoleTableGrants))
	}
	if len(schema.RoleColumnGrants) != 3 {
		t.Fatalf("RoleColumnGrants = %d, want 3", len(schema.RoleColumnGrants))
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	for _, want := range []string{
		"GRANT SELECT(secret) ON TABLE Records TO ROLE analyst",
		"GRANT INSERT(write_only) ON TABLE Records TO ROLE analyst",
		"GRANT UPDATE(modified_at) ON TABLE Records TO ROLE analyst",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated DDL missing %q; got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "GRANT SELECT ON TABLE Records") ||
		strings.Contains(got, "GRANT INSERT ON TABLE Records") ||
		strings.Contains(got, "GRANT UPDATE ON TABLE Records") {
		t.Errorf("column-scoped grants were widened; got:\n%s", got)
	}

	reconstructed, err := FromDDLStatements(ddls)
	if err != nil {
		t.Fatalf("reconstruct generated DDL: %v", err)
	}
	if len(reconstructed.RoleTableGrants) != 0 || len(reconstructed.RoleColumnGrants) != 3 {
		t.Fatalf("reconstructed grants = table %d, column %d; want table 0, column 3", len(reconstructed.RoleTableGrants), len(reconstructed.RoleColumnGrants))
	}
}

func TestToRolesDDL_SuppressesInheritedColumnRows(t *testing.T) {
	schema := &Schema{
		RoleTableGrants: []*infoschem.RoleTableGrant{
			{TableName: "Records", PrivilegeType: "SELECT", Grantee: "analyst"},
		},
		RoleColumnGrants: []*infoschem.RoleColumnGrant{
			{TableName: "Records", ColumnName: "secret", PrivilegeType: "SELECT", Grantee: "analyst"},
			{TableName: "Records", ColumnName: "write_only", PrivilegeType: "INSERT", Grantee: "analyst"},
		},
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	if !strings.Contains(got, "GRANT SELECT ON TABLE Records TO ROLE analyst") {
		t.Fatalf("table-wide SELECT grant missing; got:\n%s", got)
	}
	if strings.Contains(got, "SELECT(secret)") {
		t.Fatalf("inherited SELECT column row should be suppressed; got:\n%s", got)
	}
	if !strings.Contains(got, "GRANT INSERT(write_only) ON TABLE Records TO ROLE analyst") {
		t.Fatalf("explicit INSERT column grant missing; got:\n%s", got)
	}
}

func TestFromDDLStatements_Revoke(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"CREATE ROLE analyst",
		"GRANT SELECT(secret) ON TABLE Records TO ROLE analyst",
		"GRANT SELECT ON TABLE Records TO ROLE analyst",
		"GRANT SELECT ON ALL TABLES IN SCHEMA app TO ROLE analyst",
		"GRANT SELECT ON ALL VIEWS IN SCHEMA app TO ROLE analyst",
		"GRANT SELECT ON ALL CHANGE STREAMS IN SCHEMA app TO ROLE analyst",
		"GRANT USAGE ON SCHEMA app TO ROLE analyst",
		"GRANT SELECT ON VIEW Report TO ROLE analyst",
		"GRANT SELECT ON CHANGE STREAM Changes TO ROLE analyst",
		"GRANT EXECUTE ON TABLE FUNCTION ReadChanges TO ROLE analyst",
		"GRANT ROLE base TO ROLE analyst",
		"REVOKE SELECT(secret) ON TABLE Records FROM ROLE analyst",
		"REVOKE SELECT ON TABLE Records FROM ROLE analyst",
		"REVOKE SELECT ON ALL TABLES IN SCHEMA app FROM ROLE analyst",
		"REVOKE SELECT ON ALL VIEWS IN SCHEMA app FROM ROLE analyst",
		"REVOKE SELECT ON ALL CHANGE STREAMS IN SCHEMA app FROM ROLE analyst",
		"REVOKE USAGE ON SCHEMA app FROM ROLE analyst",
		"REVOKE SELECT ON VIEW Report FROM ROLE analyst",
		"REVOKE SELECT ON CHANGE STREAM Changes FROM ROLE analyst",
		"REVOKE EXECUTE ON TABLE FUNCTION ReadChanges FROM ROLE analyst",
		"REVOKE ROLE base FROM ROLE analyst",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.RoleTableGrants) != 0 || len(schema.RoleColumnGrants) != 0 {
		t.Fatalf("table grants after revoke = %d table, %d column; want 0, 0", len(schema.RoleTableGrants), len(schema.RoleColumnGrants))
	}
	if len(schema.AllSchemaGrants) != 0 {
		t.Fatalf("AllSchemaGrants after revoke = %d, want 0", len(schema.AllSchemaGrants))
	}
	if len(schema.SchemaGrants) != 0 || len(schema.RoleChangeStreamGrants) != 0 ||
		len(schema.RoleRoutineGrants) != 0 || len(schema.RoleGrantees) != 0 {
		t.Fatalf("non-table grants remain after revoke: schema=%d change-stream=%d routine=%d role=%d",
			len(schema.SchemaGrants),
			len(schema.RoleChangeStreamGrants),
			len(schema.RoleRoutineGrants),
			len(schema.RoleGrantees),
		)
	}
}

func TestFromDDLStatements_RevokeTablePrivilegeLeavesColumnGrant(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"GRANT SELECT ON TABLE Records TO ROLE analyst",
		"GRANT SELECT(secret) ON TABLE Records TO ROLE analyst",
		"REVOKE SELECT ON TABLE Records FROM ROLE analyst",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}
	if len(schema.RoleTableGrants) != 0 {
		t.Fatalf("RoleTableGrants = %d, want 0", len(schema.RoleTableGrants))
	}
	if len(schema.RoleColumnGrants) != 1 || schema.RoleColumnGrants[0].ColumnName != "secret" {
		t.Fatalf("RoleColumnGrants = %+v, want the explicit secret column grant", schema.RoleColumnGrants)
	}
}

func TestFromDDLStatements_RevokeSchemaUsageIsScopedToGranteeAndSchema(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"GRANT USAGE ON SCHEMA app TO ROLE analyst",
		"GRANT USAGE ON SCHEMA audit TO ROLE analyst",
		"GRANT USAGE ON SCHEMA app TO ROLE reporter",
		"REVOKE USAGE ON SCHEMA app FROM ROLE analyst",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}
	if len(schema.SchemaGrants) != 2 {
		t.Fatalf("SchemaGrants = %+v, want audit/analyst and app/reporter", schema.SchemaGrants)
	}
	got := map[string]bool{}
	for _, grant := range schema.SchemaGrants {
		got[grant.SchemaName+"/"+grant.Grantee] = true
	}
	for _, want := range []string{"audit/analyst", "app/reporter"} {
		if !got[want] {
			t.Errorf("remaining schema grants = %+v, missing %s", schema.SchemaGrants, want)
		}
	}
}

func TestRoundtrip_NamedSchemaTableGrant(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"GRANT SELECT ON TABLE app.Records TO ROLE analyst",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}
	if len(schema.RoleTableGrants) != 1 || schema.RoleTableGrants[0].TableSchema != "app" {
		t.Fatalf("RoleTableGrants = %+v, want app.Records", schema.RoleTableGrants)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	if got := ddlSQL(t, ddls); !strings.Contains(got, "GRANT SELECT ON TABLE app.Records TO ROLE analyst") {
		t.Fatalf("generated DDL lost named-schema table grant:\n%s", got)
	}
}

func TestFromDDLStatements_NamedSchemaRevokeIsQualified(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"CREATE VIEW Report SQL SECURITY INVOKER AS SELECT 1",
		"CREATE VIEW app.Report SQL SECURITY INVOKER AS SELECT 2",
		"GRANT SELECT ON VIEW Report TO ROLE analyst",
		"GRANT SELECT ON VIEW app.Report TO ROLE analyst",
		"GRANT SELECT(secret) ON TABLE app.Records TO ROLE analyst",
		"GRANT SELECT(secret) ON TABLE audit.Records TO ROLE analyst",
		"REVOKE SELECT ON VIEW app.Report FROM ROLE analyst",
		"REVOKE SELECT(secret) ON TABLE app.Records FROM ROLE analyst",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}
	if len(schema.RoleTableGrants) != 1 ||
		schema.RoleTableGrants[0].TableSchema != "" ||
		schema.RoleTableGrants[0].TableName != "Report" {
		t.Fatalf("RoleTableGrants = %+v, want only default Report", schema.RoleTableGrants)
	}
	if len(schema.RoleColumnGrants) != 1 ||
		schema.RoleColumnGrants[0].TableSchema != "audit" ||
		schema.RoleColumnGrants[0].TableName != "Records" {
		t.Fatalf("RoleColumnGrants = %+v, want only audit.Records", schema.RoleColumnGrants)
	}
}

func TestRoundtrip_NamedSchemaChangeStreamAndTableFunctionGrants(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"GRANT SELECT ON CHANGE STREAM app.Changes TO ROLE analyst",
		"GRANT SELECT ON CHANGE STREAM audit.Changes TO ROLE analyst",
		"GRANT EXECUTE ON TABLE FUNCTION app.ReadRows TO ROLE analyst",
		"GRANT EXECUTE ON TABLE FUNCTION audit.ReadRows TO ROLE analyst",
		"REVOKE SELECT ON CHANGE STREAM app.Changes FROM ROLE analyst",
		"REVOKE EXECUTE ON TABLE FUNCTION app.ReadRows FROM ROLE analyst",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}
	if len(schema.RoleChangeStreamGrants) != 1 ||
		schema.RoleChangeStreamGrants[0].ChangeStreamSchema != "audit" ||
		schema.RoleChangeStreamGrants[0].ChangeStreamName != "Changes" {
		t.Fatalf("RoleChangeStreamGrants = %+v, want only audit.Changes", schema.RoleChangeStreamGrants)
	}
	if len(schema.RoleRoutineGrants) != 1 ||
		schema.RoleRoutineGrants[0].SpecificSchema != "audit" ||
		schema.RoleRoutineGrants[0].SpecificName != "ReadRows" {
		t.Fatalf("RoleRoutineGrants = %+v, want only audit.ReadRows", schema.RoleRoutineGrants)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	for _, want := range []string{
		"GRANT SELECT ON CHANGE STREAM audit.Changes TO ROLE analyst",
		"GRANT EXECUTE ON TABLE FUNCTION audit.ReadRows TO ROLE analyst",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated DDL missing %q; got:\n%s", want, got)
		}
	}
}

func TestToRolesDDL_InvalidMetadataReturnsErrors(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
		want   string
	}{
		{
			name: "unknown table privilege",
			schema: &Schema{RoleTableGrants: []*infoschem.RoleTableGrant{
				{TableName: "Records", PrivilegeType: "TRUNCATE", Grantee: "analyst"},
			}},
			want: "invalid table privilege",
		},
		{
			name: "unknown column privilege",
			schema: &Schema{RoleColumnGrants: []*infoschem.RoleColumnGrant{
				{TableName: "Records", ColumnName: "secret", PrivilegeType: "DELETE", Grantee: "analyst"},
			}},
			want: "invalid column privilege",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.schema.ToDDLStatements()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ToDDLStatements error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestRoundtrip_RemoteFunctionMetadata(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"CREATE FUNCTION RemoteLookup(values ARRAY<STRING(MAX)>) RETURNS ARRAY<STRING(MAX)> NOT DETERMINISTIC LANGUAGE REMOTE OPTIONS (endpoint = 'https://example.test')",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.Routines) != 1 || len(schema.Parameters) != 1 {
		t.Fatalf("routines/parameters = %d/%d, want 1/1", len(schema.Routines), len(schema.Parameters))
	}
	routine := schema.Routines[0]
	if routine.Language != "REMOTE" || routine.Determinism != "NOT DETERMINISTIC" || routine.Remote {
		t.Fatalf("routine metadata = language %q, determinism %q, remote %v; want REMOTE, NOT DETERMINISTIC, false",
			routine.Language,
			routine.Determinism,
			routine.Remote,
		)
	}
	if routine.RoutineBody != "REMOTE" {
		t.Errorf("RoutineBody = %q, want REMOTE", routine.RoutineBody)
	}
	if got := *schema.Parameters[0].SpannerType; got != "ARRAY<STRING(MAX)>" {
		t.Errorf("parameter SPANNER_TYPE = %q, want ARRAY<STRING(MAX)>", got)
	}
	if got := *routine.SpannerType; got != "ARRAY<STRING(MAX)>" {
		t.Errorf("return SPANNER_TYPE = %q, want ARRAY<STRING(MAX)>", got)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	for _, want := range []string{
		"RETURNS ARRAY<STRING(MAX)>",
		"NOT DETERMINISTIC",
		"LANGUAGE REMOTE",
		"OPTIONS (endpoint = \"https://example.test\")",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated remote function missing %q; got:\n%s", want, got)
		}
	}
}

func TestToFunctionsDDL_PrefersSpannerType(t *testing.T) {
	routineType := "ARRAY<STRING(MAX)>"
	parameterType := "ARRAY<STRING(MAX)>"
	schema := &Schema{
		Routines: []*infoschem.Routine{
			{
				SpecificName: "ExactTypes",
				RoutineName:  "ExactTypes",
				RoutineType:  "FUNCTION",
				SpannerType:  &routineType,
				DataType:     strPtr("ARRAY"),
			},
		},
		Parameters: []*infoschem.Parameter{
			{
				SpecificName:    "ExactTypes",
				OrdinalPosition: 1,
				ParameterName:   strPtr("values"),
				SpannerType:     &parameterType,
				DataType:        strPtr("ARRAY"),
			},
		},
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	var function *ast.CreateFunction
	for _, ddl := range ddls {
		if candidate, ok := ddl.(*ast.CreateFunction); ok {
			function = candidate
		}
	}
	if function == nil {
		t.Fatal("CREATE FUNCTION not found")
	}
	if got := function.Params[0].Type.SQL(); got != parameterType {
		t.Errorf("parameter type = %q, want %q", got, parameterType)
	}
	if got := function.ReturnType.SQL(); got != routineType {
		t.Errorf("return type = %q, want %q", got, routineType)
	}
}

func TestToFunctionsDDL_UsesFunctionTypeGrammar(t *testing.T) {
	parameterType := "INTERVAL"
	returnType := "STRING"
	schema := &Schema{
		Routines: []*infoschem.Routine{
			{
				SpecificName: "FunctionTypes",
				RoutineName:  "FunctionTypes",
				RoutineType:  "FUNCTION",
				SpannerType:  &returnType,
			},
		},
		Parameters: []*infoschem.Parameter{
			{
				SpecificName:    "FunctionTypes",
				OrdinalPosition: 1,
				ParameterName:   strPtr("value"),
				SpannerType:     &parameterType,
			},
		},
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	function := ddls[0].(*ast.CreateFunction)
	if got := function.Params[0].Type.SQL(); got != "INTERVAL" {
		t.Errorf("parameter type = %q, want INTERVAL", got)
	}
	if got := function.ReturnType.SQL(); got != "STRING" {
		t.Errorf("return type = %q, want STRING", got)
	}
}

func TestToFunctionsDDL_LiveRemoteMetadataWithoutSyntaxDetailsReturnsError(t *testing.T) {
	schema := &Schema{Routines: []*infoschem.Routine{
		{
			SpecificName: "RemoteLookup",
			RoutineName:  "RemoteLookup",
			RoutineType:  "FUNCTION",
			RoutineBody:  "EXTERNAL",
		},
	}}

	_, err := schema.ToDDLStatements()
	if err == nil || !strings.Contains(err.Error(), "refusing to guess remote-function syntax") {
		t.Fatalf("ToDDLStatements error = %v, want explicit missing remote metadata error", err)
	}
}

func TestToFunctionsDDL_RemoteClauseMetadataFallback(t *testing.T) {
	schema := &Schema{
		Routines: []*infoschem.Routine{
			{
				SpecificName: "RemoteLookup",
				RoutineName:  "RemoteLookup",
				RoutineType:  "FUNCTION",
				RoutineBody:  "EXTERNAL",
			},
		},
		RoutineOptions: []*infoschem.RoutineOption{
			{SpecificName: "RemoteLookup", OptionName: "LANGUAGE", OptionValue: "'REMOTE'"},
			{SpecificName: "RemoteLookup", OptionName: "DETERMINISM", OptionValue: "'NOT DETERMINISTIC'"},
			{SpecificName: "RemoteLookup", OptionName: "endpoint", OptionType: "STRING", OptionValue: "'https://example.test'"},
		},
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	for _, want := range []string{"NOT DETERMINISTIC", "LANGUAGE REMOTE", "OPTIONS (endpoint = \"https://example.test\")"} {
		if !strings.Contains(got, want) {
			t.Errorf("generated remote function missing %q; got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "LANGUAGE =") || strings.Contains(got, "DETERMINISM =") {
		t.Errorf("clause metadata leaked into OPTIONS; got:\n%s", got)
	}
}

func TestToFunctionsDDL_NamedSchemaFunction(t *testing.T) {
	returnType := "INT64"
	schema := &Schema{Routines: []*infoschem.Routine{
		{
			SpecificSchema: "app",
			SpecificName:   "LookupValue",
			RoutineSchema:  "app",
			RoutineName:    "LookupValue",
			RoutineType:    "FUNCTION",
			SpannerType:    &returnType,
		},
	}}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	if got := ddlSQL(t, ddls); !strings.Contains(got, "CREATE FUNCTION app.LookupValue () RETURNS INT64") {
		t.Fatalf("generated DDL lost named-schema function:\n%s", got)
	}
}

func TestToFunctionsDDL_InconsistentSchemasReturnError(t *testing.T) {
	schema := &Schema{Routines: []*infoschem.Routine{
		{
			SpecificSchema: "audit",
			SpecificName:   "LookupValue",
			RoutineSchema:  "app",
			RoutineName:    "LookupValue",
			RoutineType:    "FUNCTION",
		},
	}}

	_, err := schema.ToDDLStatements()
	if err == nil || !strings.Contains(err.Error(), "inconsistent routine schema") {
		t.Fatalf("ToDDLStatements error = %v, want inconsistent-schema error", err)
	}
}

func TestToRolesDDL_ModelGrantUnsupported(t *testing.T) {
	schema := &Schema{
		RoleModelGrants: []*infoschem.RoleModelGrant{
			{ModelName: "Classifier", PrivilegeType: "EXECUTE", Grantee: "analyst"},
		},
	}

	_, err := schema.ToDDLStatements()
	if err == nil {
		t.Fatal("ToDDLStatements succeeded for an unrepresentable model grant")
	}
	for _, want := range []string{"model grant", "Classifier", "analyst", "no model-privilege AST node"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err, want)
		}
	}
}
