package infoschem_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	databasepb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/apstndb/spanalyzer/survey/astconv"
	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

const uuidFixtureCleanupTimeout = 5 * time.Minute

const uuidFixtureTable = "UUIDDefaultProbe"

const uuidFixtureDDL = `CREATE TABLE UUIDDefaultProbe (
  Id UUID NOT NULL DEFAULT (NEW_UUID()),
  Payload STRING(MAX),
) PRIMARY KEY (Id)`

func updateUUIDFixtureDDL(ctx context.Context, admin *database.DatabaseAdminClient, databaseName, statement string) error {
	op, err := admin.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   databaseName,
		Statements: []string{statement},
	})
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

func verifyUUIDFixture(ctx context.Context, t *testing.T, client *spanner.Client, tableName string) {
	t.Helper()

	columnIter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME, ORDINAL_POSITION,
                     COLUMN_DEFAULT, DATA_TYPE, IS_NULLABLE, SPANNER_TYPE,
                     IS_GENERATED, GENERATION_EXPRESSION, IS_HIDDEN
                FROM INFORMATION_SCHEMA.COLUMNS
               WHERE TABLE_SCHEMA = '' AND TABLE_NAME = @table_name
               ORDER BY ORDINAL_POSITION`,
		Params: map[string]any{"table_name": tableName},
	})
	defer columnIter.Stop()
	var columns []*infoschem.Column
	if err := spanner.SelectAll(columnIter, &columns, spanner.WithLenient()); err != nil {
		t.Fatalf("load UUID column metadata: %v", err)
	}
	if len(columns) != 2 {
		t.Fatalf("UUID fixture columns = %d, want 2", len(columns))
	}
	id := columns[0]
	if id.ColumnName != "Id" {
		t.Fatalf("first UUID fixture column = %q, want Id", id.ColumnName)
	}
	// Spanner leaves the SQL-standard DATA_TYPE field NULL for UUID and exposes
	// the usable type name through SPANNER_TYPE.
	if id.DataType != nil {
		t.Errorf("UUID DATA_TYPE = %q, want SQL NULL", *id.DataType)
	}
	if id.SpannerType != "UUID" {
		t.Errorf("UUID SPANNER_TYPE = %q, want UUID", id.SpannerType)
	}
	if id.ColumnDefault == nil || *id.ColumnDefault != "NEW_UUID()" {
		t.Errorf("UUID COLUMN_DEFAULT = %v, want NEW_UUID()", id.ColumnDefault)
	}

	indexIter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_SCHEMA, TABLE_NAME, INDEX_NAME, INDEX_TYPE, COLUMN_NAME,
                     ORDINAL_POSITION, COLUMN_ORDERING, IS_NULLABLE, SPANNER_TYPE
                FROM INFORMATION_SCHEMA.INDEX_COLUMNS
               WHERE TABLE_SCHEMA = '' AND TABLE_NAME = @table_name
                 AND INDEX_NAME = 'PRIMARY_KEY'
               ORDER BY ORDINAL_POSITION`,
		Params: map[string]any{"table_name": tableName},
	})
	defer indexIter.Stop()
	var indexColumns []*infoschem.IndexColumn
	if err := spanner.SelectAll(indexIter, &indexColumns, spanner.WithLenient()); err != nil {
		t.Fatalf("load UUID primary-key metadata: %v", err)
	}
	if len(indexColumns) != 1 || indexColumns[0].ColumnName != "Id" {
		t.Fatalf("UUID primary-key columns = %#v, want [Id]", indexColumns)
	}

	schema := &astconv.Schema{
		Tables:       []*infoschem.Table{{TableName: tableName, TableType: "BASE TABLE"}},
		Columns:      columns,
		IndexColumns: indexColumns,
	}
	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("reconstruct UUID fixture: %v", err)
	}
	if len(ddls) != 1 {
		t.Fatalf("reconstructed UUID statements = %d, want 1", len(ddls))
	}
	table, ok := ddls[0].(*ast.CreateTable)
	if !ok {
		t.Fatalf("reconstructed UUID statement type = %T, want *ast.CreateTable", ddls[0])
	}
	if len(table.Columns) != 2 || table.Columns[0].Type.SQL() != "UUID" {
		t.Fatalf("reconstructed UUID columns = %s", table.SQL())
	}
	defaultExpr, ok := table.Columns[0].DefaultSemantics.(*ast.ColumnDefaultExpr)
	if !ok || defaultExpr.Expr.SQL() != "NEW_UUID()" {
		t.Errorf("reconstructed UUID default = %s", table.Columns[0].SQL())
	}
	if len(table.PrimaryKeys) != 1 || table.PrimaryKeys[0].Name.Name != "Id" {
		t.Errorf("reconstructed UUID primary key = %s", table.SQL())
	}
}

func TestUUID_RealSpanner(t *testing.T) {
	if os.Getenv("REQUIRE_REAL_SPANNER_UUID") != "1" {
		t.Skip("skipping managed-Spanner UUID fixture; run mise run test-uuid-real")
	}
	databaseName := os.Getenv("TEST_REAL_SPANNER_DATABASE")
	if databaseName == "" {
		t.Fatal("TEST_REAL_SPANNER_DATABASE is required when REQUIRE_REAL_SPANNER_UUID=1")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	tableName := fmt.Sprintf("UUIDDefaultProbe_%d", time.Now().UnixNano())
	ddl := fmt.Sprintf(`CREATE TABLE %s (
  Id UUID NOT NULL DEFAULT (NEW_UUID()),
  Payload STRING(MAX),
) PRIMARY KEY (Id)`, tableName)

	admin, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		t.Fatalf("database.NewDatabaseAdminClient: %v", err)
	}
	defer func() {
		if err := admin.Close(); err != nil {
			t.Errorf("close managed-Spanner database admin client: %v", err)
		}
	}()
	update := func(opCtx context.Context, statement string) error {
		return updateUUIDFixtureDDL(opCtx, admin, databaseName, statement)
	}
	if err := update(ctx, ddl); err != nil {
		t.Fatalf("create managed-Spanner UUID fixture: %v", err)
	}
	created := true
	defer func() {
		if err := runUUIDFixtureFallbackCleanup(&created, tableName, uuidFixtureCleanupTimeout, update); err != nil {
			t.Errorf("drop managed-Spanner UUID fixture: %v", err)
		}
	}()

	client, err := spanner.NewClient(ctx, databaseName)
	if err != nil {
		t.Fatalf("spanner.NewClient: %v", err)
	}
	defer client.Close()
	verifyUUIDFixture(ctx, t, client, tableName)

	if err := update(ctx, uuidFixtureDropStatement(tableName)); err != nil {
		t.Fatalf("drop managed-Spanner UUID fixture: %v", err)
	}
	created = false
	type countRow struct {
		Count int64 `spanner:"OBJECT_COUNT"`
	}
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*) AS OBJECT_COUNT
                FROM INFORMATION_SCHEMA.TABLES
               WHERE TABLE_SCHEMA = '' AND TABLE_NAME = @table_name`,
		Params: map[string]any{"table_name": tableName},
	})
	defer iter.Stop()
	var counts []countRow
	if err := spanner.SelectAll(iter, &counts); err != nil {
		t.Fatalf("verify managed-Spanner UUID fixture cleanup: %v", err)
	}
	if len(counts) != 1 || counts[0].Count != 0 {
		t.Errorf("managed-Spanner UUID fixture remains after cleanup: %#v", counts)
	}
}
