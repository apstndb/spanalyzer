package astconv

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/apstndb/spanemuboost"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func TestRowDeletionPolicyAlterMatchesInline(t *testing.T) {
	for _, name := range []string{"Events", "app.Events"} {
		t.Run(name, func(t *testing.T) {
			create := "CREATE TABLE " + name + " (Id INT64 NOT NULL, CreatedAt TIMESTAMP) PRIMARY KEY (Id)"
			inline, err := FromDDLStatements(parseDDLs(t, create+", ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 30 DAY))"))
			if err != nil {
				t.Fatal(err)
			}
			altered, err := FromDDLStatements(parseDDLs(t, create, "ALTER TABLE "+name+" ADD ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 30 DAY))"))
			if err != nil {
				t.Fatal(err)
			}
			want, err := inline.ToDDLStatements()
			if err != nil {
				t.Fatal(err)
			}
			got, err := altered.ToDDLStatements()
			if err != nil {
				t.Fatal(err)
			}
			if ddlSQL(t, got) != ddlSQL(t, want) || strings.Contains(ddlSQL(t, got), "ALTER TABLE") {
				t.Fatalf("inline:\n%s\naltered:\n%s", ddlSQL(t, want), ddlSQL(t, got))
			}
		})
	}
}

func TestRowDeletionPolicyAlterRejectsInvalidTransitions(t *testing.T) {
	create := `CREATE TABLE Events (Id INT64 NOT NULL, CreatedAt TIMESTAMP) PRIMARY KEY (Id)`
	add := `ALTER TABLE Events ADD ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 30 DAY))`
	for _, tt := range []struct {
		name, want string
		ddls       []string
	}{
		{"missing target", "has not been created", []string{add}},
		{"view is not a table", "not a base table", []string{`CREATE VIEW Events SQL SECURITY INVOKER AS SELECT CURRENT_TIMESTAMP() AS CreatedAt`, add}},
		{"duplicate ADD", "already has a row deletion policy", []string{create, add, add}},
		{"inline then ADD", "already has a row deletion policy", []string{create + ", ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 7 DAY))", add}},
		{"DROP remains unsupported", "unsupported ALTER TABLE operation", []string{create, add, "ALTER TABLE Events DROP ROW DELETION POLICY"}},
		{"REPLACE remains unsupported", "unsupported ALTER TABLE operation", []string{create, add, "ALTER TABLE Events REPLACE ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 7 DAY))"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := FromDDLStatements(parseDDLs(t, tt.ddls...)); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v, want %q", err, tt.want)
			}
		})
	}
	ddls := parseDDLs(t, create)
	ddls = append(ddls, &ast.AlterTable{Name: path("Events"), TableAlteration: &ast.AddRowDeletionPolicy{}})
	if _, err := FromDDLStatements(ddls); err == nil || !strings.Contains(err.Error(), "requires a column and interval") {
		t.Fatalf("malformed policy error = %v", err)
	}
}

func TestRowDeletionPolicyAlterKeepsSchemaIdentity(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		`CREATE TABLE Events (Id INT64 NOT NULL, CreatedAt TIMESTAMP) PRIMARY KEY (Id)`,
		`CREATE TABLE app.Events (Id INT64 NOT NULL, CreatedAt TIMESTAMP) PRIMARY KEY (Id)`,
		`ALTER TABLE app.Events ADD ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 30 DAY))`,
	))
	if err != nil {
		t.Fatal(err)
	}
	if schema.Tables[0].RowDeletionPolicyExpression != nil || schema.Tables[1].RowDeletionPolicyExpression == nil {
		t.Fatalf("policy applied to the wrong schema: %+v", schema.Tables)
	}
}

func TestRowDeletionPolicyAlterReappliesOnEmulator(t *testing.T) {
	if testing.Short() {
		t.Skip("requires pinned Emulator")
	}
	runtime := spanemuboost.NewLazyRuntime(spanemuboost.BackendEmulator, spanemuboost.WithContainerImage(pinnedEmulatorImage(t)))
	t.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			t.Error(err)
		}
	})
	statements := []string{`CREATE SCHEMA app`, `CREATE TABLE app.Events (Id INT64 NOT NULL, CreatedAt TIMESTAMP) PRIMARY KEY (Id)`, `ALTER TABLE app.Events ADD ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 30 DAY))`}
	control := spanemuboost.SetupClients(t, runtime, spanemuboost.EnableDatabaseAutoConfigOnly(), spanemuboost.WithRandomDatabaseID(), spanemuboost.WithSetupDDLs(statements), spanemuboost.ForceSchemaTeardown())
	for _, fromLive := range []bool{false, true} {
		var schema *Schema
		var err error
		if fromLive {
			schema, err = LoadSchema(t.Context(), control.Client)
		} else {
			schema, err = FromDDLStatements(parseDDLs(t, statements...))
		}
		if err != nil {
			t.Fatal(err)
		}
		exported, err := schema.ToDDLStatements()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(ddlSQL(t, exported), "ROW DELETION POLICY") || strings.Contains(ddlSQL(t, exported), "ALTER TABLE") {
			t.Fatalf("policy was lost: %s", ddlSQL(t, exported))
		}
		ddls := make([]string, len(exported))
		for i, ddl := range exported {
			ddls[i] = ddl.SQL()
		}
		replica := spanemuboost.SetupClients(t, runtime, spanemuboost.EnableDatabaseAutoConfigOnly(), spanemuboost.WithRandomDatabaseID(), spanemuboost.ForceSchemaTeardown())
		op, err := replica.DatabaseClient.UpdateDatabaseDdl(t.Context(), &databasepb.UpdateDatabaseDdlRequest{Database: replica.DatabasePath(), Statements: ddls})
		if err != nil {
			t.Fatal(err)
		}
		if err := op.Wait(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
}
