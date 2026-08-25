package astconv

import (
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func TestRoundtrip_AnonymousPrimaryKey(t *testing.T) {
	ddl := `CREATE TABLE t (
  a INT64,
  b INT64,
  PRIMARY KEY (a, b)
)`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.Tables) != 1 {
		t.Fatalf("Tables = %d, want 1", len(schema.Tables))
	}

	var pkCols []string
	for _, ic := range schema.IndexColumns {
		if ic.TableName == "t" && ic.IndexName == "PRIMARY_KEY" {
			pkCols = append(pkCols, ic.ColumnName)
		}
	}
	if len(pkCols) != 2 || pkCols[0] != "a" || pkCols[1] != "b" {
		t.Errorf("PK columns = %v, want [a b]", pkCols)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var b strings.Builder
	for _, d := range ddls {
		b.WriteString(d.SQL())
		b.WriteByte('\n')
	}
	got := b.String()
	if !strings.Contains(got, "PRIMARY KEY (a ASC, b ASC)") {
		t.Errorf("generated DDL missing PRIMARY KEY; got:\n%s", got)
	}
}

func TestRoundtrip_AllInSchemaGrants(t *testing.T) {
	ddls := []string{
		`CREATE TABLE t (a INT64) PRIMARY KEY (a)`,
		`CREATE ROLE r`,
		`GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA s TO ROLE r`,
		`GRANT SELECT ON ALL VIEWS IN SCHEMA s TO ROLE r`,
		`GRANT SELECT ON ALL CHANGE STREAMS IN SCHEMA s TO ROLE r`,
	}

	var stmts []ast.DDL
	for _, d := range ddls {
		stmt, err := memefish.ParseDDL("", d)
		if err != nil {
			t.Fatalf("ParseDDL(%q): %v", d, err)
		}
		stmts = append(stmts, stmt)
	}

	schema, err := FromDDLStatements(stmts)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.AllSchemaGrants) != 4 {
		t.Fatalf("AllSchemaGrants = %d, want 4", len(schema.AllSchemaGrants))
	}

	want := map[string]bool{
		"TABLES:SELECT":         false,
		"TABLES:INSERT":         false,
		"VIEWS:SELECT":          false,
		"CHANGE_STREAMS:SELECT": false,
	}
	for _, g := range schema.AllSchemaGrants {
		key := g.ObjectType + ":" + g.PrivilegeType
		if _, ok := want[key]; !ok {
			t.Errorf("unexpected AllSchemaGrant: %s", key)
			continue
		}
		want[key] = true
		if g.SchemaName != "s" {
			t.Errorf("SchemaName = %q, want s", g.SchemaName)
		}
		if g.Grantee != "r" {
			t.Errorf("Grantee = %q, want r", g.Grantee)
		}
	}
	for k, ok := range want {
		if !ok {
			t.Errorf("missing AllSchemaGrant: %s", k)
		}
	}

	out, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var generated []string
	for _, d := range out {
		generated = append(generated, d.SQL())
	}
	got := strings.Join(generated, "\n")

	for _, fragment := range []string{
		"GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA s TO ROLE r",
		"GRANT SELECT ON ALL VIEWS IN SCHEMA s TO ROLE r",
		"GRANT SELECT ON ALL CHANGE STREAMS IN SCHEMA s TO ROLE r",
	} {
		if !strings.Contains(got, fragment) {
			t.Errorf("generated DDL missing %q; got:\n%s", fragment, got)
		}
	}
}

func TestRoundtrip_ModelColumnsAndOptions(t *testing.T) {
	ddl := `CREATE MODEL MyModel INPUT (x INT64) OUTPUT (y STRING(MAX)) REMOTE OPTIONS (endpoint = 'e')`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var got []string
	for _, d := range ddls {
		got = append(got, d.SQL())
	}
	sql := strings.Join(got, "\n")

	if !strings.Contains(sql, "INPUT (x INT64)") {
		t.Errorf("generated DDL missing INPUT column; got:\n%s", sql)
	}
	if !strings.Contains(sql, "OUTPUT (y STRING(MAX))") {
		t.Errorf("generated DDL missing OUTPUT column; got:\n%s", sql)
	}
}

func TestRoundtrip_SelfReferentialForeignKey(t *testing.T) {
	ddls := []string{
		`CREATE TABLE Nodes (
  Id INT64 NOT NULL,
  ParentId INT64,
  CONSTRAINT FK_Self FOREIGN KEY (ParentId) REFERENCES Nodes (Id)
) PRIMARY KEY (Id)`,
	}

	var stmts []ast.DDL
	for _, d := range ddls {
		stmt, err := memefish.ParseDDL("", d)
		if err != nil {
			t.Fatalf("ParseDDL(%q): %v", d, err)
		}
		stmts = append(stmts, stmt)
	}

	schema, err := FromDDLStatements(stmts)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	out, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var got []string
	for _, d := range out {
		got = append(got, d.SQL())
	}
	sql := strings.Join(got, "\n")

	if !strings.Contains(sql, "REFERENCES Nodes (Id)") {
		t.Errorf("self-referential FK should reference Nodes; got:\n%s", sql)
	}
	if strings.Contains(sql, "REFERENCES PK_Nodes") {
		t.Errorf("self-referential FK should not reference constraint name; got:\n%s", sql)
	}
}

func TestToRolesDDL_AllSchemaGrantValidation(t *testing.T) {
	schema := &Schema{
		AllSchemaGrants: []*infoschem.AllSchemaGrant{
			{
				ObjectType:    "TABLES",
				PrivilegeType: "EXECUTE",
				SchemaName:    "s",
				Grantee:       "r",
			},
		},
	}

	if _, err := schema.ToDDLStatements(); err == nil {
		t.Fatal("expected error for invalid privilege type on TABLES, got nil")
	}

	schema.AllSchemaGrants[0].ObjectType = "VIEWS"
	schema.AllSchemaGrants[0].PrivilegeType = "INSERT"
	if _, err := schema.ToDDLStatements(); err == nil {
		t.Fatal("expected error for invalid privilege type on VIEWS, got nil")
	}
}
