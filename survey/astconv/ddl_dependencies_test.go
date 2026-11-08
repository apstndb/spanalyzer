package astconv

import (
	"strings"
	"testing"
)

func TestDDLDependencies(t *testing.T) {
	tests := []struct {
		name string
		ddls []string
		want []string
	}{
		{
			name: "sequence default",
			ddls: []string{
				`CREATE TABLE T (Id INT64 NOT NULL DEFAULT (GET_NEXT_SEQUENCE_VALUE(SEQUENCE S))) PRIMARY KEY (Id)`,
				`CREATE SEQUENCE S OPTIONS (sequence_kind = 'bit_reversed_positive')`,
			},
			want: []string{"sequence:s", "relation:t"},
		},
		{
			name: "function and view chain",
			ddls: []string{
				`CREATE VIEW ALeaf SQL SECURITY INVOKER AS SELECT z.Id FROM ZBase z`,
				`CREATE VIEW ZBase SQL SECURITY INVOKER AS SELECT app.F(1) AS Id`,
				`CREATE FUNCTION app.F(x INT64) RETURNS INT64 SQL SECURITY INVOKER AS (app.G(x))`,
				`CREATE FUNCTION app.G(x INT64) RETURNS INT64 SQL SECURITY INVOKER AS (x + 1)`,
				`CREATE SCHEMA app`,
			},
			want: []string{"schema:app", "function:app\x00g", "function:app\x00f", "relation:zbase", "relation:aleaf"},
		},
		{
			name: "locality group options",
			ddls: []string{
				`CREATE TABLE T (Id INT64 NOT NULL) PRIMARY KEY (Id), OPTIONS (locality_group = 'archive')`,
				`ALTER DATABASE db SET OPTIONS (default_locality_group = 'archive')`,
				`CREATE LOCALITY GROUP archive OPTIONS (storage = 'hdd')`,
			},
			// The original family order is the tie-breaker after archive.
			want: []string{"locality group:archive", "", "relation:t"},
		},
		{
			name: "case insensitive references retain spelling",
			ddls: []string{
				`CREATE VIEW V SQL SECURITY INVOKER AS SELECT t.Id FROM basetable t`,
				`CREATE TABLE BaseTable (Id INT64 NOT NULL) PRIMARY KEY (Id)`,
			},
			want: []string{"relation:basetable", "relation:v"},
		},
		{
			name: "CTE is scoped and sequential",
			ddls: []string{
				`CREATE VIEW V SQL SECURITY INVOKER AS WITH V AS (SELECT b.Id FROM Base b) SELECT v.Id FROM V v`,
				`CREATE VIEW Base SQL SECURITY INVOKER AS SELECT 1 AS Id`,
			},
			want: []string{"relation:base", "relation:v"},
		},
		{
			name: "range variable path shadows a schema relation",
			ddls: []string{
				`CREATE SCHEMA app`,
				`CREATE TABLE T (Id INT64 NOT NULL, Items ARRAY<INT64>) PRIMARY KEY (Id)`,
				`CREATE VIEW app.Items SQL SECURITY INVOKER AS SELECT item AS Id FROM T AS app, app.Items AS item`,
			},
			want: []string{"schema:app", "relation:t", "relation:app\x00items"},
		},
		{
			name: "derived table alias shadows a schema relation",
			ddls: []string{
				`CREATE SCHEMA app`,
				`CREATE TABLE T (Id INT64 NOT NULL, Items ARRAY<INT64>) PRIMARY KEY (Id)`,
				`CREATE VIEW app.Items SQL SECURITY INVOKER AS SELECT item AS Id FROM (SELECT t.Items FROM T t) AS app, app.Items AS item`,
			},
			want: []string{"schema:app", "relation:t", "relation:app\x00items"},
		},
		{
			name: "UNNEST alias shadows a schema relation",
			ddls: []string{
				`CREATE SCHEMA app`,
				`CREATE VIEW app.Items SQL SECURITY INVOKER AS SELECT item AS Id FROM UNNEST([STRUCT([1] AS Items)]) AS app, app.Items AS item`,
			},
			want: []string{"schema:app", "relation:app\x00items"},
		},
		{
			name: "partial export allows external references",
			ddls: []string{`CREATE VIEW V SQL SECURITY INVOKER AS SELECT t.Id FROM ExternalTable t`},
			want: []string{"relation:v"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := FromDDLStatements(parseDDLs(t, tt.ddls...))
			if err != nil {
				t.Fatal(err)
			}
			out, err := schema.ToDDLStatements()
			if err != nil {
				t.Fatal(err)
			}
			if len(out) != len(tt.want) {
				t.Fatalf("got %d statements, want %d", len(out), len(tt.want))
			}
			for i, ddl := range out {
				key, _ := ddlIdentity(ddl)
				if key != tt.want[i] {
					t.Errorf("statement %d = %s (%s), want %s", i, key, ddl.SQL(), tt.want[i])
				}
			}
		})
	}
}

func TestDDLDependencyErrors(t *testing.T) {
	for _, tt := range []struct {
		name, want string
		ddls       []string
	}{
		{"view cycle", "dependency cycle", []string{
			`CREATE VIEW A SQL SECURITY INVOKER AS SELECT b.Id FROM B b`,
			`CREATE VIEW B SQL SECURITY INVOKER AS SELECT a.Id FROM A a`,
		}},
		{"self reference in CTE definition", "dependency cycle", []string{
			`CREATE VIEW V SQL SECURITY INVOKER AS WITH V AS (SELECT v.Id FROM V v) SELECT v.Id FROM V v`,
		}},
		{"duplicate SQL object", "duplicate DDL object", []string{
			`CREATE SEQUENCE S OPTIONS (sequence_kind = 'bit_reversed_positive')`,
			`CREATE SEQUENCE s OPTIONS (sequence_kind = 'bit_reversed_positive')`,
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ddls := parseDDLs(t, tt.ddls...)
			var first string
			for range 5 {
				out, err := orderDDLDependencies(ddls)
				if err == nil || !strings.Contains(err.Error(), tt.want) || out != nil {
					t.Fatalf("output = %v, error = %v, want %q", out, err, tt.want)
				}
				if first != "" && err.Error() != first {
					t.Fatalf("nondeterministic error: %s / %s", first, err)
				}
				first = err.Error()
			}
		})
	}
}

func TestDDLDependencyStableTies(t *testing.T) {
	ddls := parseDDLs(t, `CREATE VIEW Z SQL SECURITY INVOKER AS SELECT 1 AS Id`, `CREATE VIEW A SQL SECURITY INVOKER AS SELECT 2 AS Id`)
	out, err := orderDDLDependencies(ddls)
	if err != nil {
		t.Fatal(err)
	}
	for i := range ddls {
		if out[i] != ddls[i] {
			t.Fatal("independent DDL was reordered")
		}
	}
}
