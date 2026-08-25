package astconv

import (
	"strings"
	"testing"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func TestRoundtrip_AnonymousChecksAndForeignKeys(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		`CREATE TABLE ParentA (
  Id INT64 NOT NULL,
) PRIMARY KEY (Id)`,
		`CREATE TABLE ParentB (
  Id INT64 NOT NULL,
) PRIMARY KEY (Id)`,
		`CREATE TABLE Child (
  Id INT64 NOT NULL,
  AId INT64,
  BId INT64,
  CHECK (Id > 0),
  CHECK (Id < 100),
  FOREIGN KEY (AId) REFERENCES ParentA (Id),
  FOREIGN KEY (BId) REFERENCES ParentB (Id),
) PRIMARY KEY (Id)`,
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	if strings.Count(got, "CHECK (") != 2 {
		t.Fatalf("generated DDL has %d CHECK constraints, want 2:\n%s", strings.Count(got, "CHECK ("), got)
	}
	if strings.Count(got, "FOREIGN KEY (") != 2 {
		t.Fatalf("generated DDL has %d foreign keys, want 2:\n%s", strings.Count(got, "FOREIGN KEY ("), got)
	}
	if strings.Contains(got, "CONSTRAINT") {
		t.Fatalf("anonymous constraint received an emitted name:\n%s", got)
	}
}

func TestRoundtrip_ForeignKeyPreservesNonPrimaryReferencedColumns(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		`CREATE TABLE Parent (
  Id INT64 NOT NULL,
  ExternalA INT64,
  ExternalB INT64,
) PRIMARY KEY (Id)`,
		`CREATE TABLE Child (
  Id INT64 NOT NULL,
  LocalA INT64,
  LocalB INT64,
  CONSTRAINT FK_External FOREIGN KEY (LocalA, LocalB) REFERENCES Parent (ExternalB, ExternalA),
) PRIMARY KEY (Id)`,
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	if !strings.Contains(got, "FOREIGN KEY (LocalA, LocalB) REFERENCES Parent (ExternalB, ExternalA)") {
		t.Fatalf("non-primary referenced columns changed:\n%s", got)
	}
}

func TestRoundtrip_AlterTableAddConstraint(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		`CREATE SCHEMA app`,
		`CREATE TABLE app.Parent (
  Id INT64 NOT NULL,
  ExternalId INT64,
) PRIMARY KEY (Id)`,
		`CREATE TABLE app.Child (
  Id INT64 NOT NULL,
  ParentExternalId INT64,
) PRIMARY KEY (Id)`,
		`ALTER TABLE app.Child ADD CONSTRAINT FK_External
  FOREIGN KEY (ParentExternalId) REFERENCES app.Parent (ExternalId) ON DELETE CASCADE`,
		`ALTER TABLE app.Child ADD CHECK (ParentExternalId > 0)`,
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	if !strings.Contains(got, "CONSTRAINT FK_External FOREIGN KEY (ParentExternalId) REFERENCES app.Parent (ExternalId) ON DELETE CASCADE") {
		t.Fatalf("ALTER TABLE foreign key was not folded into CREATE TABLE:\n%s", got)
	}
	if !strings.Contains(got, "CHECK (ParentExternalId > 0)") {
		t.Fatalf("ALTER TABLE check constraint was not folded into CREATE TABLE:\n%s", got)
	}
	if strings.Contains(got, "ALTER TABLE") {
		t.Fatalf("reconstruction should emit canonical CREATE TABLE constraints:\n%s", got)
	}
}

func TestFromDDLStatements_AlterTableFailsClosed(t *testing.T) {
	for _, tt := range []struct {
		name string
		ddls []string
		want string
	}{
		{
			name: "target not created",
			ddls: []string{`ALTER TABLE Missing ADD CHECK (Id > 0)`},
			want: "has not been created",
		},
		{
			name: "unsupported operation",
			ddls: []string{
				`CREATE TABLE T (Id INT64 NOT NULL) PRIMARY KEY (Id)`,
				`ALTER TABLE T ADD COLUMN Extra INT64`,
			},
			want: "unsupported ALTER TABLE operation",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromDDLStatements(parseDDLs(t, tt.ddls...))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("FromDDLStatements error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestToTablesDDL_CompositeForeignKeyUsesReferencedKeyOrder(t *testing.T) {
	position := func(value int64) *int64 {
		return &value
	}
	ordinal := func(value int64) *int64 {
		return &value
	}

	schema := &Schema{
		Tables: []*infoschem.Table{
			{TableName: "Child", TableType: "BASE TABLE"},
			{TableName: "Parent", TableType: "BASE TABLE"},
		},
		Columns: []*infoschem.Column{
			{TableName: "Child", ColumnName: "Id", OrdinalPosition: 1, SpannerType: "INT64"},
			{TableName: "Child", ColumnName: "LocalA", OrdinalPosition: 2, SpannerType: "INT64"},
			{TableName: "Child", ColumnName: "LocalB", OrdinalPosition: 3, SpannerType: "INT64"},
			{TableName: "Parent", ColumnName: "K", OrdinalPosition: 1, SpannerType: "INT64"},
			{TableName: "Parent", ColumnName: "B", OrdinalPosition: 2, SpannerType: "INT64"},
		},
		TableConstraints: []*infoschem.TableConstraint{
			{
				TableName:      "Child",
				ConstraintName: "FK_Child_Parent",
				ConstraintType: "FOREIGN KEY",
				Enforced:       "YES",
			},
		},
		ReferentialConstraints: []*infoschem.ReferentialConstraint{
			{
				ConstraintName:         "FK_Child_Parent",
				UniqueConstraintName:   "PRIMARY_KEY",
				UniqueConstraintSchema: "",
				DeleteRule:             "NO ACTION",
			},
		},
		KeyColumnUsage: []*infoschem.KeyColumnUsage{
			{
				ConstraintName:             "FK_Child_Parent",
				TableName:                  "Child",
				ColumnName:                 "LocalB",
				OrdinalPosition:            2,
				PositionInUniqueConstraint: position(1),
			},
			{
				ConstraintName:  "PRIMARY_KEY",
				TableName:       "Parent",
				ColumnName:      "B",
				OrdinalPosition: 2,
			},
			{
				ConstraintName:             "FK_Child_Parent",
				TableName:                  "Child",
				ColumnName:                 "LocalA",
				OrdinalPosition:            1,
				PositionInUniqueConstraint: position(2),
			},
			{
				ConstraintName:  "PRIMARY_KEY",
				TableName:       "Parent",
				ColumnName:      "K",
				OrdinalPosition: 1,
			},
		},
		ConstraintColumnUsage: []*infoschem.ConstraintColumnUsage{
			{ConstraintName: "FK_Child_Parent", TableName: "Parent", ColumnName: "K"},
			{ConstraintName: "FK_Child_Parent", TableName: "Parent", ColumnName: "B"},
		},
		ConstraintTableUsage: []*infoschem.ConstraintTableUsage{
			{ConstraintName: "FK_Child_Parent", TableName: "Parent"},
		},
		IndexColumns: []*infoschem.IndexColumn{
			{TableName: "Parent", IndexName: "PRIMARY_KEY", ColumnName: "K", OrdinalPosition: ordinal(1)},
			{TableName: "Parent", IndexName: "PRIMARY_KEY", ColumnName: "B", OrdinalPosition: ordinal(2)},
		},
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var createTables []*ast.CreateTable
	for _, ddl := range ddls {
		if table, ok := ddl.(*ast.CreateTable); ok {
			createTables = append(createTables, table)
		}
	}
	if len(createTables) != 2 {
		t.Fatalf("generated %d tables, want 2", len(createTables))
	}
	if got := leafName(createTables[0].Name); got != "Parent" {
		t.Fatalf("first table = %q, want Parent", got)
	}

	childSQL := createTables[1].SQL()
	if !strings.Contains(childSQL, "FOREIGN KEY (LocalA, LocalB) REFERENCES Parent (B, K)") {
		t.Fatalf("composite foreign key order was not reconstructed from key metadata:\n%s", childSQL)
	}
}

func TestToTablesDDL_ForeignKeyCycleReturnsError(t *testing.T) {
	schema := &Schema{
		Tables: []*infoschem.Table{
			{TableName: "A", TableType: "BASE TABLE"},
			{TableName: "B", TableType: "BASE TABLE"},
		},
		TableConstraints: []*infoschem.TableConstraint{
			{TableName: "A", ConstraintName: "FK_A_B", ConstraintType: "FOREIGN KEY"},
			{TableName: "B", ConstraintName: "FK_B_A", ConstraintType: "FOREIGN KEY"},
		},
		ReferentialConstraints: []*infoschem.ReferentialConstraint{
			{ConstraintName: "FK_A_B", UniqueConstraintName: "PRIMARY_KEY"},
			{ConstraintName: "FK_B_A", UniqueConstraintName: "PRIMARY_KEY"},
		},
		ConstraintTableUsage: []*infoschem.ConstraintTableUsage{
			{ConstraintName: "FK_A_B", TableName: "B"},
			{ConstraintName: "FK_B_A", TableName: "A"},
		},
	}

	_, err := schema.ToDDLStatements()
	if err == nil {
		t.Fatal("ToDDLStatements succeeded for a foreign-key dependency cycle")
	}
	if !strings.Contains(err.Error(), "table dependency cycle") ||
		!strings.Contains(err.Error(), "A") || !strings.Contains(err.Error(), "B") {
		t.Fatalf("cycle error = %v, want table names and context", err)
	}
}

func TestToTablesDDL_DoesNotGuessReferencedTableFromConstraintName(t *testing.T) {
	schema := &Schema{
		Tables: []*infoschem.Table{{TableName: "Child", TableType: "BASE TABLE"}},
		TableConstraints: []*infoschem.TableConstraint{
			{TableName: "Child", ConstraintName: "FK_Child", ConstraintType: "FOREIGN KEY"},
		},
		ReferentialConstraints: []*infoschem.ReferentialConstraint{
			{ConstraintName: "FK_Child", UniqueConstraintName: "Parent"},
		},
	}

	_, err := schema.ToDDLStatements()
	if err == nil || !strings.Contains(err.Error(), "no referenced table metadata") {
		t.Fatalf("ToDDLStatements error = %v, want missing referenced table metadata", err)
	}
}

func TestToTablesDDL_CompositeForeignKeyWithoutOrdinalMetadataReturnsError(t *testing.T) {
	schema := &Schema{
		Tables: []*infoschem.Table{
			{TableName: "Parent", TableType: "BASE TABLE"},
			{TableName: "Child", TableType: "BASE TABLE"},
		},
		TableConstraints: []*infoschem.TableConstraint{
			{TableName: "Child", ConstraintName: "FK_Child", ConstraintType: "FOREIGN KEY"},
		},
		ReferentialConstraints: []*infoschem.ReferentialConstraint{
			{ConstraintName: "FK_Child", UniqueConstraintName: "PK_Parent"},
		},
		ConstraintTableUsage: []*infoschem.ConstraintTableUsage{
			{ConstraintName: "FK_Child", TableName: "Parent"},
		},
		KeyColumnUsage: []*infoschem.KeyColumnUsage{
			{ConstraintName: "FK_Child", TableName: "Child", ColumnName: "A", OrdinalPosition: 1},
			{ConstraintName: "FK_Child", TableName: "Child", ColumnName: "B", OrdinalPosition: 2},
		},
		ConstraintColumnUsage: []*infoschem.ConstraintColumnUsage{
			{ConstraintName: "FK_Child", TableName: "Parent", ColumnName: "X"},
			{ConstraintName: "FK_Child", TableName: "Parent", ColumnName: "Y"},
		},
	}

	_, err := schema.ToDDLStatements()
	if err == nil || !strings.Contains(err.Error(), "cannot determine referenced column order") {
		t.Fatalf("ToDDLStatements error = %v, want explicit composite-order error", err)
	}
}

func TestToTablesDDL_QualifiedSystemTableDoesNotHideUserTable(t *testing.T) {
	schema := &Schema{
		Tables: []*infoschem.Table{
			{TableSchema: "INFORMATION_SCHEMA", TableName: "COLUMNS", TableType: "BASE TABLE"},
			{TableName: "COLUMNS", TableType: "BASE TABLE"},
		},
		Columns: []*infoschem.Column{
			{TableName: "COLUMNS", ColumnName: "Id", OrdinalPosition: 1, SpannerType: "INT64"},
		},
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	var createTables []*ast.CreateTable
	for _, ddl := range ddls {
		if table, ok := ddl.(*ast.CreateTable); ok {
			createTables = append(createTables, table)
		}
	}
	if len(createTables) != 1 || leafName(createTables[0].Name) != "COLUMNS" {
		t.Fatalf("generated tables = %v, want default-schema user COLUMNS only", createTables)
	}
}

func TestToTablesDDL_RowDeletionPolicyErrorIncludesContext(t *testing.T) {
	expression := "NOT_A_ROW_DELETION_POLICY()"
	schema := &Schema{
		Tables: []*infoschem.Table{{TableName: "Events", TableType: "BASE TABLE", RowDeletionPolicyExpression: &expression}},
	}

	_, err := schema.ToDDLStatements()
	if err == nil {
		t.Fatal("ToDDLStatements succeeded for invalid row-deletion policy metadata")
	}
	if !strings.Contains(err.Error(), "Events") || !strings.Contains(err.Error(), expression) {
		t.Fatalf("row-deletion error = %v, want table and expression context", err)
	}
}

func TestRoundtrip_OmittedPrimaryKeyRemainsOmitted(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		`CREATE TABLE Pkless (Name STRING(MAX), Rank INT64)`,
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}
	if len(schema.Indexes) != 0 || len(schema.IndexColumns) != 0 || len(schema.TableConstraints) != 0 {
		t.Fatalf(
			"omitted primary key produced metadata: indexes=%d index_columns=%d constraints=%d",
			len(schema.Indexes),
			len(schema.IndexColumns),
			len(schema.TableConstraints),
		)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	if strings.Contains(got, "PRIMARY KEY") || strings.Contains(got, "rowid") {
		t.Fatalf("omitted primary key was expanded or changed:\n%s", got)
	}
	reentered, err := FromDDLStatements(ddls)
	if err != nil {
		t.Fatalf("FromDDLStatements(generated AST): %v", err)
	}
	if len(reentered.TableConstraints) != 0 || len(reentered.Indexes) != 0 {
		t.Fatalf(
			"generated omitted-key AST was reclassified: constraints=%d indexes=%d",
			len(reentered.TableConstraints),
			len(reentered.Indexes),
		)
	}
}

func TestRoundtrip_ZeroColumnPrimaryKeyPreservesSingletonSemantics(t *testing.T) {
	parsed := parseDDLs(t,
		`CREATE TABLE Singleton (Name STRING(MAX)) PRIMARY KEY ()`,
	)
	createTable := parsed[0].(*ast.CreateTable)
	if createTable.PrimaryKeyRparen.Invalid() {
		t.Fatal("parser did not distinguish explicit empty primary key from omission")
	}

	schema, err := FromDDLStatements(parsed)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}
	if len(schema.Indexes) != 0 || len(schema.IndexColumns) != 0 {
		t.Fatalf("zero-column primary key produced index metadata: indexes=%d index_columns=%d", len(schema.Indexes), len(schema.IndexColumns))
	}
	if len(schema.TableConstraints) != 1 || schema.TableConstraints[0].ConstraintType != "PRIMARY KEY" {
		t.Fatalf("zero-column primary key constraints = %#v, want one PRIMARY KEY marker", schema.TableConstraints)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	if !strings.Contains(got, "PRIMARY KEY ()") {
		t.Fatalf("zero-column primary key missing from generated DDL:\n%s", got)
	}
	generated := ddls[0].(*ast.CreateTable)
	if len(generated.TableConstraints) != 1 {
		t.Fatalf("generated DDL has %d table constraints, want 1", len(generated.TableConstraints))
	}
	if pk, ok := generated.TableConstraints[0].Constraint.(*ast.TablePrimaryKey); !ok || len(pk.Columns) != 0 {
		t.Fatalf("generated table constraint = %#v, want zero-column primary key", generated.TableConstraints[0].Constraint)
	}
}

func TestToTablesDDL_LiveZeroColumnPrimaryKeyMetadata(t *testing.T) {
	for _, tc := range []struct {
		name    string
		indexes []*infoschem.Index
	}{
		{name: "managed_and_omni"},
		{
			name: "emulator_v1_5_56",
			indexes: []*infoschem.Index{{
				TableName:  "Singleton",
				IndexName:  "PRIMARY_KEY",
				IndexType:  "PRIMARY_KEY",
				IsUnique:   true,
				IndexState: strPtr("READ_WRITE"),
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema := &Schema{
				Tables:  []*infoschem.Table{{TableName: "Singleton", TableType: "BASE TABLE"}},
				Indexes: tc.indexes,
				Columns: []*infoschem.Column{{
					TableName:       "Singleton",
					ColumnName:      "Name",
					OrdinalPosition: 1,
					SpannerType:     "STRING(MAX)",
				}},
				TableConstraints: []*infoschem.TableConstraint{{
					ConstraintName:    "PK_Singleton",
					TableName:         "Singleton",
					ConstraintType:    "PRIMARY KEY",
					IsDeferrable:      "NO",
					InitiallyDeferred: "NO",
					Enforced:          "YES",
				}},
			}

			ddls, err := schema.ToDDLStatements()
			if err != nil {
				t.Fatalf("ToDDLStatements: %v", err)
			}
			got := ddlSQL(t, ddls)
			if !strings.Contains(got, "PRIMARY KEY ()") {
				t.Fatalf("live zero-column primary-key metadata changed to omission:\n%s", got)
			}
		})
	}
}

func TestToTablesDDL_RestoresIdentityColumnKind(t *testing.T) {
	for _, identityKind := range []string{
		"BIT_REVERSED_POSITIVE",
		"BIT_REVERSED_POSITIVE_SEQUENCE",
	} {
		t.Run(identityKind, func(t *testing.T) {
			schema := &Schema{
				Tables: []*infoschem.Table{{TableName: "Events", TableType: "BASE TABLE"}},
				Columns: []*infoschem.Column{
					{
						TableName:       "Events",
						ColumnName:      "EventId",
						OrdinalPosition: 1,
						SpannerType:     "INT64",
						IsIdentity:      strPtr("YES"),
						IdentityKind:    &identityKind,
					},
				},
			}

			ddls, err := schema.ToDDLStatements()
			if err != nil {
				t.Fatalf("ToDDLStatements: %v", err)
			}
			got := ddlSQL(t, ddls)
			if !strings.Contains(got, "BIT_REVERSED_POSITIVE") {
				t.Fatalf("identity kind missing from generated DDL:\n%s", got)
			}
		})
	}
}
