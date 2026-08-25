package astconv

import (
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func TestRoundtrip_NamedSchemaSupportedObjects(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"CREATE SCHEMA app",
		`CREATE TABLE app.Parents (
  Id INT64 NOT NULL,
  Code STRING(20),
  CreatedAt TIMESTAMP OPTIONS (allow_commit_timestamp = true),
  Location STRING(MAX) NOT NULL PLACEMENT KEY,
  SYNONYM (ParentsAlias),
  CONSTRAINT PositiveId CHECK (Id > 0),
) PRIMARY KEY (Id), OPTIONS (locality_group = 'regional')`,
		`CREATE TABLE app.Children (
  Id INT64 NOT NULL,
  ParentId INT64 NOT NULL,
  Tokens TOKENLIST,
  CONSTRAINT ChildParent FOREIGN KEY (ParentId) REFERENCES app.Parents (Id),
) PRIMARY KEY (Id),
  INTERLEAVE IN PARENT app.Parents ON DELETE CASCADE`,
		"CREATE INDEX app.ByParent ON app.Children (ParentId DESC) STORING (Id)",
		"CREATE SEARCH INDEX app.SearchChildren ON app.Children (Tokens)",
		"CREATE VIEW app.ChildIds SQL SECURITY INVOKER AS SELECT Id FROM app.Children",
		"CREATE SEQUENCE app.ChildSequence OPTIONS (sequence_kind = 'bit_reversed_positive')",
		"CREATE FUNCTION app.AddOne(value INT64) RETURNS INT64 SQL SECURITY INVOKER AS (value + 1)",
		"CREATE ROLE analyst",
		"GRANT SELECT ON TABLE app.Children TO ROLE analyst",
		"GRANT SELECT ON VIEW app.ChildIds TO ROLE analyst",
		"GRANT SELECT, UPDATE ON SEQUENCE app.ChildSequence TO ROLE analyst",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	for _, table := range schema.Tables {
		if table.TableType == "BASE TABLE" && table.TableSchema != "app" {
			t.Errorf("table %s schema = %q, want app", table.TableName, table.TableSchema)
		}
	}
	for _, column := range schema.Columns {
		if column.TableSchema != "app" {
			t.Errorf("column %s.%s schema = %q, want app", column.TableName, column.ColumnName, column.TableSchema)
		}
	}
	for _, index := range schema.Indexes {
		if index.TableSchema != "app" {
			t.Errorf("index %s schema = %q, want app", index.IndexName, index.TableSchema)
		}
	}
	for _, indexColumn := range schema.IndexColumns {
		if indexColumn.TableSchema != "app" {
			t.Errorf("index column %s.%s schema = %q, want app", indexColumn.IndexName, indexColumn.ColumnName, indexColumn.TableSchema)
		}
	}
	for _, constraint := range schema.TableConstraints {
		if constraint.ConstraintSchema != "app" || constraint.TableSchema != "app" {
			t.Errorf("table constraint %+v lost app schema", constraint)
		}
	}
	for _, usage := range schema.KeyColumnUsage {
		if usage.ConstraintSchema != "app" || usage.TableSchema != "app" {
			t.Errorf("key-column usage %+v lost app schema", usage)
		}
	}
	for _, option := range schema.ColumnOptions {
		if option.TableSchema != "app" {
			t.Errorf("column option %+v lost app schema", option)
		}
	}
	for _, option := range schema.TableOptions {
		if option.TableSchema != "app" {
			t.Errorf("table option %+v lost app schema", option)
		}
	}
	if len(schema.PlacementKeyColumns) != 1 ||
		schema.PlacementKeyColumns[0].TableSchema != "app" {
		t.Fatalf("PlacementKeyColumns = %+v, want app placement key", schema.PlacementKeyColumns)
	}
	if len(schema.TableSynonyms) != 1 ||
		schema.TableSynonyms[0].TableSchema != "app" ||
		schema.TableSynonyms[0].SynonymSchema != "app" {
		t.Fatalf("TableSynonyms = %+v, want same-schema app synonym", schema.TableSynonyms)
	}
	if len(schema.Sequences) != 1 || schema.Sequences[0].Schema != "app" {
		t.Fatalf("Sequences = %+v, want app sequence", schema.Sequences)
	}
	if len(schema.Routines) != 1 ||
		schema.Routines[0].RoutineSchema != "app" ||
		schema.Routines[0].SpecificSchema != "app" {
		t.Fatalf("Routines = %+v, want app routine/specific schemas", schema.Routines)
	}
	if len(schema.Parameters) != 1 || schema.Parameters[0].SpecificSchema != "app" {
		t.Fatalf("Parameters = %+v, want app function parameter", schema.Parameters)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	for _, want := range []string{
		"CREATE TABLE app.Parents",
		"SYNONYM (ParentsAlias)",
		"PLACEMENT KEY",
		"allow_commit_timestamp = true",
		"OPTIONS (locality_group = \"regional\")",
		"CREATE TABLE app.Children",
		"REFERENCES app.Parents (Id)",
		"INTERLEAVE IN PARENT app.Parents",
		"CREATE INDEX app.ByParent ON app.Children",
		"CREATE SEARCH INDEX app.SearchChildren ON app.Children",
		"CREATE VIEW app.ChildIds",
		"CREATE SEQUENCE app.ChildSequence",
		"CREATE FUNCTION app.AddOne",
		"GRANT SELECT ON TABLE app.Children",
		"GRANT SELECT ON VIEW app.ChildIds",
		"GRANT SELECT, UPDATE ON SEQUENCE app.ChildSequence",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated DDL missing %q:\n%s", want, got)
		}
	}
	if strings.Index(got, "CREATE TABLE app.Parents") > strings.Index(got, "CREATE TABLE app.Children") {
		t.Errorf("interleave/FK parent emitted after child:\n%s", got)
	}
}

func TestRoundtrip_NamedSchemaLeafCollisionsRemainIsolated(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"CREATE TABLE app.Records (Id INT64, AppValue STRING(MAX)) PRIMARY KEY (Id)",
		"CREATE TABLE audit.Records (Id INT64, AuditValue BYTES(MAX)) PRIMARY KEY (Id)",
		"CREATE INDEX app.ByValue ON app.Records (AppValue)",
		"CREATE INDEX audit.ByValue ON audit.Records (AuditValue)",
		"CREATE VIEW app.Report SQL SECURITY INVOKER AS SELECT AppValue FROM app.Records",
		"CREATE VIEW audit.Report SQL SECURITY INVOKER AS SELECT AuditValue FROM audit.Records",
		"CREATE SEQUENCE app.Counter OPTIONS (start_with_counter = 10)",
		"CREATE SEQUENCE audit.Counter OPTIONS (start_with_counter = 20)",
		"CREATE FUNCTION app.Normalize(value INT64) RETURNS INT64 AS (value)",
		"CREATE FUNCTION audit.Normalize(value STRING(MAX)) RETURNS STRING(MAX) AS (value)",
		"GRANT SELECT ON SEQUENCE app.Counter TO ROLE analyst",
		"GRANT UPDATE ON SEQUENCE audit.Counter TO ROLE analyst",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	for _, want := range []string{
		"app.Records (",
		"AppValue STRING(MAX)",
		"audit.Records (",
		"AuditValue BYTES(MAX)",
		"app.ByValue ON app.Records(AppValue",
		"audit.ByValue ON audit.Records(AuditValue",
		"app.Report",
		"audit.Report",
		"CREATE SEQUENCE app.Counter START COUNTER WITH 10",
		"CREATE SEQUENCE audit.Counter START COUNTER WITH 20",
		"CREATE FUNCTION app.Normalize (value INT64) RETURNS INT64",
		"CREATE FUNCTION audit.Normalize (value STRING(MAX)) RETURNS STRING(MAX)",
		"GRANT SELECT ON SEQUENCE app.Counter TO ROLE analyst",
		"GRANT UPDATE ON SEQUENCE audit.Counter TO ROLE analyst",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated DDL missing isolated identity %q:\n%s", want, got)
		}
	}
}

func TestFromDDLStatements_IndexAndTableSchemasMustMatch(t *testing.T) {
	tests := []struct {
		name string
		ddl  string
	}{
		{
			name: "regular index",
			ddl:  "CREATE INDEX app.ByValue ON audit.Records (Value)",
		},
		{
			name: "search index",
			ddl:  "CREATE SEARCH INDEX app.SearchRecords ON audit.Records (Tokens)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromDDLStatements(parseDDLs(t, tt.ddl))
			if err == nil || !strings.Contains(err.Error(), "must use the same schema") {
				t.Fatalf("FromDDLStatements error = %v, want same-schema error", err)
			}
		})
	}
}

func TestFromDDLStatements_NamedSchemaInterleavedIndexesFailExplicitly(t *testing.T) {
	tests := []struct {
		name string
		ddl  string
		want string
	}{
		{
			name: "regular index",
			ddl:  "CREATE INDEX app.ByValue ON app.Records (Value), INTERLEAVE IN Parents",
			want: "unsupported named-schema interleaved index",
		},
		{
			name: "search index",
			ddl:  "CREATE SEARCH INDEX app.SearchRecords ON app.Records (Tokens), INTERLEAVE IN Parents",
			want: "unsupported named-schema interleaved search index",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromDDLStatements(parseDDLs(t, tt.ddl))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("FromDDLStatements error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestRoundtrip_CrossSchemaForeignKeyOrdersQualifiedParent(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"CREATE TABLE app.Child (Id INT64, ParentId INT64, FOREIGN KEY (ParentId) REFERENCES audit.Parent (Id)) PRIMARY KEY (Id)",
		"CREATE TABLE audit.Parent (Id INT64) PRIMARY KEY (Id)",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	parent := strings.Index(got, "CREATE TABLE audit.Parent")
	child := strings.Index(got, "CREATE TABLE app.Child")
	if parent < 0 || child < 0 || parent > child {
		t.Fatalf("qualified FK parent was not emitted before child:\n%s", got)
	}
	if !strings.Contains(got, "REFERENCES audit.Parent (Id)") {
		t.Fatalf("qualified FK target was not preserved:\n%s", got)
	}
}

func TestRoundtrip_SameLeafNamedForeignKeysRemainIsolated(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"CREATE TABLE shared.Parent (Id INT64, Code STRING(10)) PRIMARY KEY (Id)",
		"CREATE TABLE app.Child (P INT64, FOREIGN KEY (P) REFERENCES shared.Parent (Id)) PRIMARY KEY (P)",
		"CREATE TABLE audit.Child (C STRING(10), FOREIGN KEY (C) REFERENCES shared.Parent (Code)) PRIMARY KEY (C)",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	tableSQL := make(map[string]string)
	for _, ddl := range ddls {
		table, ok := ddl.(*ast.CreateTable)
		if !ok {
			continue
		}
		tableSQL[table.Name.SQL()] = table.SQL()
	}
	for tableName, wantReference := range map[string]string{
		"app.Child":   "REFERENCES shared.Parent (Id)",
		"audit.Child": "REFERENCES shared.Parent (Code)",
	} {
		if !strings.Contains(tableSQL[tableName], wantReference) {
			t.Errorf("%s DDL = %q, want %q", tableName, tableSQL[tableName], wantReference)
		}
	}
}

func TestToDDLStatements_NamedVectorIndexStillFailsExplicitly(t *testing.T) {
	schema := &Schema{Indexes: []*infoschem.Index{
		{TableSchema: "app", TableName: "Records", IndexName: "Vec", IndexType: "VECTOR_INDEX"},
	}}

	_, err := schema.ToDDLStatements()
	if err == nil || !strings.Contains(err.Error(), "unsupported named-schema vector index") {
		t.Fatalf("ToDDLStatements error = %v, want named vector-index error", err)
	}
}
