package astconv

import (
	"testing"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/apstndb/spanemuboost"
)

func TestDDLDependencyReapplicationOnEmulator(t *testing.T) {
	if testing.Short() {
		t.Skip("requires the pinned Emulator")
	}
	runtime := spanemuboost.NewLazyRuntime(spanemuboost.BackendEmulator, spanemuboost.WithContainerImage(pinnedEmulatorImage(t)))
	t.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			t.Error(err)
		}
	})
	// Emulator 1.5.59 executes SQL UDFs but returned zero ROUTINES rows for
	// the named UDF fixture. Test its recoverable sequence/view dependencies;
	// Omni additionally tests live UDF metadata and function-to-view ordering.
	assertDDLDependencyReapplication(t, runtime, false)
}

func assertDDLDependencyReapplication(t *testing.T, runtime spanemuboost.RuntimeHandle, includeFunctions bool) {
	t.Helper()
	original := []string{
		`CREATE SEQUENCE S OPTIONS (sequence_kind = 'bit_reversed_positive')`,
		`CREATE TABLE T (Id INT64 NOT NULL DEFAULT (GET_NEXT_SEQUENCE_VALUE(SEQUENCE S))) PRIMARY KEY (Id)`,
		`CREATE SCHEMA app`,
		`CREATE VIEW ZBase SQL SECURITY INVOKER AS SELECT 1 AS Id`,
		`CREATE VIEW ALeaf SQL SECURITY INVOKER AS SELECT z.Id FROM ZBase z`,
	}
	views := map[string]int64{"ALeaf": 1}
	if includeFunctions {
		original = append(original, `CREATE FUNCTION app.F(x INT64) RETURNS INT64 SQL SECURITY INVOKER AS (x + 1)`, `CREATE VIEW V SQL SECURITY INVOKER AS SELECT app.F(1) AS Id`)
		views["V"] = 2
	}
	control := spanemuboost.SetupClients(t, runtime, spanemuboost.EnableDatabaseAutoConfigOnly(), spanemuboost.WithRandomDatabaseID(), spanemuboost.WithSetupDDLs(original), spanemuboost.ForceSchemaTeardown())
	for _, fromLive := range []bool{false, true} {
		name := "from_ast"
		if fromLive {
			name = "from_live"
		}
		t.Run(name, func(t *testing.T) {
			var schema *Schema
			var err error
			if fromLive {
				schema, err = LoadSchema(t.Context(), control.Client)
			} else {
				schema, err = FromDDLStatements(parseDDLs(t, original...))
			}
			if err != nil {
				t.Fatal(err)
			}
			replica := spanemuboost.SetupClients(t, runtime, spanemuboost.EnableDatabaseAutoConfigOnly(), spanemuboost.WithRandomDatabaseID(), spanemuboost.ForceSchemaTeardown())
			schema.DatabaseName = replica.DatabaseID
			exported, err := schema.ToDDLStatements()
			if err != nil {
				t.Fatal(err)
			}
			statements := make([]string, len(exported))
			for i, ddl := range exported {
				statements[i] = ddl.SQL()
			}
			op, err := replica.DatabaseClient.UpdateDatabaseDdl(t.Context(), &databasepb.UpdateDatabaseDdlRequest{Database: replica.DatabasePath(), Statements: statements})
			if err != nil {
				t.Fatal(err)
			}
			if err := op.Wait(t.Context()); err != nil {
				t.Fatalf("reapply exported DDL: %v\n%v", err, statements)
			}
			for view, want := range views {
				iter := replica.Client.Single().Query(t.Context(), spanner.NewStatement("SELECT Id FROM "+view))
				row, err := iter.Next()
				iter.Stop()
				if err != nil {
					t.Fatal(err)
				}
				var got int64
				if err := row.Column(0, &got); err != nil || got != want {
					t.Fatalf("view %s = %d, %v", view, got, err)
				}
			}
		})
	}
}
