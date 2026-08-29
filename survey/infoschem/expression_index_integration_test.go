package infoschem_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/apstndb/spanalyzer/survey/astconv"
	"github.com/apstndb/spanalyzer/survey/infoschem"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestExpressionIndex_RealSpanner(t *testing.T) {
	if os.Getenv("REQUIRE_REAL_SPANNER_EXPRESSION_INDEX") != "1" {
		t.Skip("skipping managed-Spanner expression-index fixture; run mise run test-expression-index-real")
	}
	databaseName := os.Getenv("TEST_REAL_SPANNER_DATABASE")
	if databaseName == "" {
		t.Fatal("TEST_REAL_SPANNER_DATABASE is required when REQUIRE_REAL_SPANNER_EXPRESSION_INDEX=1")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	admin, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		t.Fatalf("database.NewDatabaseAdminClient failed (code=%s)", status.Code(err))
	}
	defer func() {
		if err := admin.Close(); err != nil {
			t.Errorf("close managed-Spanner database admin client: %v", err)
		}
	}()
	client, err := spanner.NewClient(ctx, databaseName)
	if err != nil {
		t.Fatalf("spanner.NewClient failed (code=%s)", status.Code(err))
	}
	defer client.Close()

	verifyExpressionIndexAccepted(
		ctx,
		t,
		admin,
		client,
		databaseName,
		fmt.Sprintf("ExpressionIndexProbe_%d", time.Now().UnixNano()),
	)
}

func verifyExpressionIndexRejected(
	ctx context.Context,
	t *testing.T,
	admin *database.DatabaseAdminClient,
	databaseName string,
	tableName string,
) {
	t.Helper()

	if err := updateExpressionIndexDDL(ctx, admin, databaseName, expressionIndexTableDDL(tableName)); err != nil {
		t.Fatalf("create expression-index rejection fixture table: %v", err)
	}
	defer func() {
		if err := updateExpressionIndexDDL(ctx, admin, databaseName, "DROP TABLE "+tableName); err != nil {
			t.Errorf("drop expression-index rejection fixture table: %v", err)
		}
	}()

	indexName := tableName + "ByCity"
	err := updateExpressionIndexDDL(ctx, admin, databaseName, fmt.Sprintf(
		"CREATE INDEX %s ON %s((JSON_VALUE(VenueData.address.city)))",
		indexName,
		tableName,
	))
	if err == nil {
		t.Fatal("expression-index DDL succeeded, want pinned Emulator syntax rejection")
	}
	if status.Code(err) != codes.InvalidArgument || !strings.Contains(strings.ToLower(err.Error()), "syntax error") {
		t.Fatalf("expression-index rejection = %v, want InvalidArgument syntax error", err)
	}
}

func verifyExpressionIndexAccepted(
	ctx context.Context,
	t *testing.T,
	admin *database.DatabaseAdminClient,
	client *spanner.Client,
	databaseName string,
	tableName string,
) {
	t.Helper()

	cityIndex := tableName + "ByCity"
	mixedIndex := tableName + "ByNameState"
	noWrapIndex := tableName + "ByCityNoWrap"
	createdIndexes := make([]string, 0, 3)
	tableCreated := false
	cleanup := func(cleanupCtx context.Context) error {
		for len(createdIndexes) > 0 {
			last := len(createdIndexes) - 1
			indexName := createdIndexes[last]
			if err := updateExpressionIndexDDL(cleanupCtx, admin, databaseName, "DROP INDEX "+indexName); err != nil {
				return fmt.Errorf("drop index %s: %w", indexName, err)
			}
			createdIndexes = createdIndexes[:last]
		}
		if tableCreated {
			if err := updateExpressionIndexDDL(cleanupCtx, admin, databaseName, "DROP TABLE "+tableName); err != nil {
				return fmt.Errorf("drop table %s: %w", tableName, err)
			}
			tableCreated = false
		}
		return nil
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := cleanup(cleanupCtx); err != nil {
			t.Errorf("cleanup expression-index fixture: %v", err)
		}
	}()

	if err := updateExpressionIndexDDL(ctx, admin, databaseName, expressionIndexTableDDL(tableName)); err != nil {
		t.Fatalf("create expression-index fixture table: %v", err)
	}
	tableCreated = true

	acceptedDDLs := []struct {
		name string
		sql  string
	}{
		{
			name: cityIndex,
			sql: fmt.Sprintf(
				"CREATE INDEX %s ON %s((JSON_VALUE(VenueData.address.city)))",
				cityIndex,
				tableName,
			),
		},
		{
			name: mixedIndex,
			sql: fmt.Sprintf(
				"CREATE INDEX %s ON %s(VenueName, (JSON_VALUE(VenueData.address.state)))",
				mixedIndex,
				tableName,
			),
		},
		{
			name: noWrapIndex,
			sql: fmt.Sprintf(
				"CREATE INDEX %s ON %s(JSON_VALUE(VenueData.address.city))",
				noWrapIndex,
				tableName,
			),
		},
	}
	for _, ddl := range acceptedDDLs {
		if err := updateExpressionIndexDDL(ctx, admin, databaseName, ddl.sql); err != nil {
			t.Fatalf("create expression-index fixture %s: %v", ddl.name, err)
		}
		createdIndexes = append(createdIndexes, ddl.name)
	}

	descendingName := tableName + "ByCityDesc"
	descendingErr := updateExpressionIndexDDL(ctx, admin, databaseName, fmt.Sprintf(
		"CREATE INDEX %s ON %s((JSON_VALUE(VenueData.address.city)) DESC)",
		descendingName,
		tableName,
	))
	if descendingErr == nil {
		createdIndexes = append(createdIndexes, descendingName)
		t.Fatal("descending GoogleSQL expression-index key succeeded, want current syntax boundary")
	}
	if status.Code(descendingErr) != codes.InvalidArgument || !strings.Contains(strings.ToLower(descendingErr.Error()), "syntax error") {
		t.Errorf("descending expression-index error = %v, want InvalidArgument syntax error", descendingErr)
	}

	volatileName := tableName + "ByCurrentTimestamp"
	volatileErr := updateExpressionIndexDDL(ctx, admin, databaseName, fmt.Sprintf(
		"CREATE INDEX %s ON %s((CURRENT_TIMESTAMP()))",
		volatileName,
		tableName,
	))
	if volatileErr == nil {
		createdIndexes = append(createdIndexes, volatileName)
		t.Fatal("non-deterministic expression index succeeded, want semantic rejection")
	}
	if status.Code(volatileErr) != codes.InvalidArgument || !strings.Contains(strings.ToLower(volatileErr.Error()), "non-deterministic expression") {
		t.Errorf("non-deterministic expression-index error = %v, want semantic rejection", volatileErr)
	}

	rows := loadExpressionIndexColumns(ctx, t, client, tableName)
	assertExpressionIndexRows(t, rows, cityIndex, mixedIndex, noWrapIndex)
	assertExpressionIndexColumnsAreNotTableColumns(ctx, t, client, tableName)
	assertExpressionIndexCanonicalDDL(ctx, t, admin, databaseName, tableName, cityIndex, noWrapIndex)
	assertExpressionIndexReconstructionFailsClosed(ctx, t, client, cityIndex)
	if err := cleanup(ctx); err != nil {
		t.Fatalf("cleanup expression-index fixture: %v", err)
	}
	assertExpressionIndexFixtureRemoved(ctx, t, client, tableName)
}

func expressionIndexTableDDL(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE %s (
  Id INT64 NOT NULL,
  VenueName STRING(MAX),
  VenueData JSON
) PRIMARY KEY (Id)`, tableName)
}

func updateExpressionIndexDDL(
	ctx context.Context,
	admin *database.DatabaseAdminClient,
	databaseName string,
	statement string,
) error {
	op, err := admin.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   databaseName,
		Statements: []string{statement},
	})
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

func loadExpressionIndexColumns(
	ctx context.Context,
	t *testing.T,
	client *spanner.Client,
	tableName string,
) []*infoschem.IndexColumn {
	t.Helper()
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_SCHEMA, TABLE_NAME, INDEX_NAME, INDEX_TYPE, COLUMN_NAME,
                     ORDINAL_POSITION, COLUMN_ORDERING, IS_NULLABLE, SPANNER_TYPE,
                     EXPRESSION
                FROM INFORMATION_SCHEMA.INDEX_COLUMNS
               WHERE TABLE_SCHEMA = '' AND TABLE_NAME = @table_name
                 AND INDEX_NAME != 'PRIMARY_KEY'
               ORDER BY INDEX_NAME, ORDINAL_POSITION`,
		Params: map[string]any{"table_name": tableName},
	})
	defer iter.Stop()
	var rows []*infoschem.IndexColumn
	if err := spanner.SelectAll(iter, &rows, spanner.WithLenient()); err != nil {
		t.Fatalf("load expression-index metadata: %v", err)
	}
	return rows
}

func assertExpressionIndexRows(
	t *testing.T,
	rows []*infoschem.IndexColumn,
	cityIndex string,
	mixedIndex string,
	noWrapIndex string,
) {
	t.Helper()
	if len(rows) != 4 {
		t.Fatalf("expression-index metadata rows = %d, want 4", len(rows))
	}
	wantExpressions := map[string][]string{
		cityIndex:   {"(JSON_VALUE(VenueData.address.city))"},
		mixedIndex:  {"", "(JSON_VALUE(VenueData.address.state))"},
		noWrapIndex: {"JSON_VALUE(VenueData.address.city)"},
	}
	seen := make(map[string]int)
	for _, row := range rows {
		position := seen[row.IndexName]
		seen[row.IndexName]++
		want, ok := wantExpressions[row.IndexName]
		if !ok || position >= len(want) {
			t.Errorf("unexpected expression-index metadata row: %#v", row)
			continue
		}
		if row.OrdinalPosition == nil || *row.OrdinalPosition != int64(position+1) {
			t.Errorf("%s ordinal = %v, want %d", row.IndexName, row.OrdinalPosition, position+1)
		}
		if row.ColumnOrdering == nil || *row.ColumnOrdering != "ASC" {
			t.Errorf("%s ordering = %v, want ASC", row.IndexName, row.ColumnOrdering)
		}
		if want[position] == "" {
			if row.ColumnName != "VenueName" || row.Expression != nil {
				t.Errorf("ordinary mixed-index key = %#v, want VenueName with NULL expression", row)
			}
			continue
		}
		if row.Expression == nil || *row.Expression != want[position] {
			t.Errorf("%s expression = %v, want %q", row.IndexName, row.Expression, want[position])
		}
		if wantPrefix := "_ExpressionIndex_" + row.IndexName + "_"; !strings.HasPrefix(row.ColumnName, wantPrefix) {
			t.Errorf("%s internal column = %q, want prefix %q", row.IndexName, row.ColumnName, wantPrefix)
		}
	}
	for indexName, want := range wantExpressions {
		if seen[indexName] != len(want) {
			t.Errorf("%s metadata rows = %d, want %d", indexName, seen[indexName], len(want))
		}
	}
}

func assertExpressionIndexColumnsAreNotTableColumns(
	ctx context.Context,
	t *testing.T,
	client *spanner.Client,
	tableName string,
) {
	t.Helper()
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COLUMN_NAME
                FROM INFORMATION_SCHEMA.COLUMNS
               WHERE TABLE_SCHEMA = '' AND TABLE_NAME = @table_name
               ORDER BY ORDINAL_POSITION`,
		Params: map[string]any{"table_name": tableName},
	})
	defer iter.Stop()
	var rows []struct {
		ColumnName string `spanner:"COLUMN_NAME"`
	}
	if err := spanner.SelectAll(iter, &rows); err != nil {
		t.Fatalf("load expression-index fixture columns: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("expression-index fixture table columns = %d, want 3", len(rows))
	}
	for _, row := range rows {
		if strings.HasPrefix(row.ColumnName, "_ExpressionIndex_") {
			t.Errorf("internal expression-index key leaked into COLUMNS as %q", row.ColumnName)
		}
	}
}

func assertExpressionIndexCanonicalDDL(
	ctx context.Context,
	t *testing.T,
	admin *database.DatabaseAdminClient,
	databaseName string,
	tableName string,
	cityIndex string,
	noWrapIndex string,
) {
	t.Helper()
	response, err := admin.GetDatabaseDdl(ctx, &databasepb.GetDatabaseDdlRequest{Database: databaseName})
	if err != nil {
		t.Fatalf("GetDatabaseDdl for expression-index fixture: %v", err)
	}
	want := map[string]string{
		cityIndex:   "((JSON_VALUE(VenueData.address.city)))",
		noWrapIndex: "(JSON_VALUE(VenueData.address.city))",
	}
	for indexName, expression := range want {
		found := false
		for _, statement := range response.Statements {
			if strings.Contains(statement, "CREATE INDEX "+indexName+" ON "+tableName) {
				found = true
				if !strings.Contains(statement, expression) {
					t.Errorf("canonical DDL for %s = %q, want expression fragment %q", indexName, statement, expression)
				}
				break
			}
		}
		if !found {
			t.Errorf("GetDatabaseDdl is missing expression index %s", indexName)
		}
	}
}

func assertExpressionIndexReconstructionFailsClosed(
	ctx context.Context,
	t *testing.T,
	client *spanner.Client,
	cityIndex string,
) {
	t.Helper()
	loaded, err := astconv.LoadSchema(ctx, client)
	if err != nil {
		t.Fatalf("LoadSchema with expression index: %v", err)
	}
	partial := &astconv.Schema{}
	for _, index := range loaded.Indexes {
		if index.IndexName == cityIndex {
			partial.Indexes = append(partial.Indexes, index)
		}
	}
	for _, column := range loaded.IndexColumns {
		if column.IndexName == cityIndex {
			partial.IndexColumns = append(partial.IndexColumns, column)
		}
	}
	if len(partial.Indexes) != 1 || len(partial.IndexColumns) != 1 {
		t.Fatalf("loaded expression index has indexes=%d columns=%d, want 1/1", len(partial.Indexes), len(partial.IndexColumns))
	}
	_, err = partial.ToDDLStatements()
	if err == nil || !strings.Contains(err.Error(), "unsupported expression index") {
		t.Fatalf("expression-index reconstruction error = %v, want fail-closed boundary", err)
	}
}

func assertExpressionIndexFixtureRemoved(
	ctx context.Context,
	t *testing.T,
	client *spanner.Client,
	tableName string,
) {
	t.Helper()
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*) AS OBJECT_COUNT
                FROM (
                  SELECT TABLE_NAME AS OBJECT_NAME
                    FROM INFORMATION_SCHEMA.TABLES
                   WHERE TABLE_SCHEMA = '' AND TABLE_NAME = @table_name
                  UNION ALL
                  SELECT INDEX_NAME AS OBJECT_NAME
                    FROM INFORMATION_SCHEMA.INDEXES
                   WHERE TABLE_SCHEMA = '' AND TABLE_NAME = @table_name
                )`,
		Params: map[string]any{"table_name": tableName},
	})
	defer iter.Stop()
	var counts []struct {
		ObjectCount int64 `spanner:"OBJECT_COUNT"`
	}
	if err := spanner.SelectAll(iter, &counts); err != nil {
		t.Fatalf("verify expression-index fixture cleanup: %v", err)
	}
	if len(counts) != 1 || counts[0].ObjectCount != 0 {
		t.Errorf("expression-index fixture remains after cleanup: %#v", counts)
	}
}
