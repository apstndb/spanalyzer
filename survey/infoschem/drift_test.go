package infoschem_test

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanemuboost"
	"github.com/apstndb/spanner-emulator-survey/astconv"
	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/apstndb/spanner-emulator-survey/spannersys"
	"github.com/cloudspannerecosystem/memefish/ast"
	"google.golang.org/api/iterator"
)

// TestDrift_EmulatorTableMetas starts the emulator and verifies that the
// registry covers every table and column exposed by its INFORMATION_SCHEMA.
//
// The registry intentionally models a production superset, so target-specific
// absences are not drift: emulator and Omni may omit valid real-Spanner columns.
// Target removals require comparison with a target-specific baseline, not this
// one-sided coverage check.
func TestDrift_EmulatorTableMetas(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping emulator drift test in short mode")
	}

	// Pin to the emulator version that meta.go was calibrated against.
	// Update this (and re-run the diff) when upgrading.
	const emulatorImage = "gcr.io/cloud-spanner-emulator/emulator:1.5.56"

	ctx := context.Background()

	env, err := spanemuboost.RunWithClients(
		ctx, spanemuboost.BackendEmulator,
		spanemuboost.WithContainerImage(emulatorImage),
		spanemuboost.WithSetupDDLs([]string{uuidFixtureDDL}),
	)
	if err != nil {
		t.Fatalf("RunWithClients: %v", err)
	}
	defer func() { _ = env.Close() }()

	_ = verifyTableMetas(ctx, t, env.Client, "emulator")
	verifyUUIDFixture(ctx, t, env.Client, uuidFixtureTable)
}

func TestDrift_OmniTableMetas(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping omni drift test in short mode")
	}

	ctx := context.Background()

	env, err := spanemuboost.RunWithClients(
		ctx,
		spanemuboost.BackendOmni,
		spanemuboost.WithSetupDDLs([]string{
			"CREATE CHANGE STREAM DefaultRetentionProbe FOR ALL",
			uuidFixtureDDL,
		}),
	)
	if err != nil {
		t.Fatalf("RunWithClients: %v", err)
	}
	defer func() { _ = env.Close() }()

	schema := verifyTableMetas(ctx, t, env.Client, "omni")
	verifyOmittedChangeStreamRetention(t, schema, "DefaultRetentionProbe")
	verifyUUIDFixture(ctx, t, env.Client, uuidFixtureTable)
	verifySpannerSys(ctx, t, env.Client, "omni")
}

func TestDrift_RealTableMetas(t *testing.T) {
	required := os.Getenv("REQUIRE_REAL_SPANNER_DRIFT") == "1"
	if testing.Short() && !required {
		t.Skip("skipping optional real Spanner drift test in short mode")
	}
	db, err := realSpannerDatabaseForDrift(
		os.Getenv("TEST_REAL_SPANNER_DATABASE"),
		required,
	)
	if err != nil {
		t.Fatal(err)
	}
	if db == "" {
		t.Skip("skipping real Spanner drift test; set TEST_REAL_SPANNER_DATABASE")
	}

	ctx := context.Background()
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		t.Fatalf("spanner.NewClient: %v", err)
	}
	defer client.Close()

	_ = verifyTableMetas(ctx, t, client, "real_spanner")
	verifySpannerSys(ctx, t, client, "real_spanner")
}

func verifySpannerSys(ctx context.Context, t *testing.T, client *spanner.Client, target string) {
	t.Helper()
	const (
		wantAdvertisedTables  = 50
		wantAdvertisedColumns = 539
		wantKnownAbsent       = 8
	)

	report, err := spannersys.Audit(ctx, client)
	if err != nil {
		t.Fatalf("SPANNER_SYS audit (%s): %v", target, err)
	}
	knownAbsentColumns := 0
	for _, columns := range report.KnownAbsentColumns {
		knownAbsentColumns += len(columns)
	}
	if report.AdvertisedTables != wantAdvertisedTables ||
		report.AdvertisedColumns != wantAdvertisedColumns ||
		knownAbsentColumns != wantKnownAbsent {
		t.Errorf(
			"SPANNER_SYS surface (%s): got %d tables / %d columns / %d known absent, want %d / %d / %d",
			target,
			report.AdvertisedTables,
			report.AdvertisedColumns,
			knownAbsentColumns,
			wantAdvertisedTables,
			wantAdvertisedColumns,
			wantKnownAbsent,
		)
	}
	t.Logf(
		"SPANNER_SYS audit (%s): registered_tables=%d advertised_tables=%d advertised_columns=%d checked_tables=%d decoded_rows=%d known_absent_columns=%d",
		target,
		report.RegisteredTables,
		report.AdvertisedTables,
		report.AdvertisedColumns,
		report.CheckedTables,
		report.DecodedRows,
		knownAbsentColumns,
	)
	if report.HasDrift() {
		t.Errorf(
			"SPANNER_SYS drift (%s): unknown_tables=%v unknown_columns=%v ordinal_mismatches=%v type_mismatches=%v",
			target,
			report.UnknownTables,
			report.UnknownColumns,
			report.OrdinalMismatches,
			report.TypeMismatches,
		)
	}
}

func realSpannerDatabaseForDrift(database string, required bool) (string, error) {
	if database != "" {
		return database, nil
	}
	if required {
		return "", fmt.Errorf("TEST_REAL_SPANNER_DATABASE is required when REQUIRE_REAL_SPANNER_DRIFT=1")
	}
	return "", nil
}

func TestRealSpannerDatabaseForDrift(t *testing.T) {
	tests := []struct {
		name     string
		database string
		required bool
		wantErr  bool
	}{
		{name: "configured", database: "projects/p/instances/i/databases/d"},
		{name: "optional and absent"},
		{name: "required and absent", required: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := realSpannerDatabaseForDrift(tt.database, tt.required)
			if (err != nil) != tt.wantErr {
				t.Fatalf("realSpannerDatabaseForDrift() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.database {
				t.Errorf("realSpannerDatabaseForDrift() = %q, want %q", got, tt.database)
			}
		})
	}
}

func verifyTableMetas(ctx context.Context, t *testing.T, client *spanner.Client, targetName string) *astconv.Schema {
	t.Helper()

	metadata, err := infoschem.DiscoverColumnMetadata(ctx, client)
	if err != nil {
		t.Fatalf("DiscoverColumnMetadata: %v", err)
	}

	// Loader discovery additionally removes explicitly marked rolling columns
	// that are advertised before they become queryable.
	discovered, err := infoschem.DiscoverColumns(ctx, client)
	if err != nil {
		t.Fatalf("DiscoverColumns: %v", err)
	}
	verifyRollingColumnsQueryable(ctx, t, client, discovered)

	// Check each TableMeta.
	allMetas := infoschem.AllTableMetas()
	var issues []string

	for _, meta := range allMetas {
		actual, tableExists := metadata[meta.Name]
		if !tableExists {
			// If it's in our meta but not in the database, that's fine (e.g. emulator missing a table)
			continue
		}

		// Verify our meta covers all columns the database actually has
		metaCols := make(map[string]infoschem.ColumnMeta)
		for _, c := range meta.Columns {
			metaCols[c.Name] = c
		}

		var extra []string
		for columnName, actualColumn := range actual {
			registeredColumn, ok := metaCols[columnName]
			if !ok {
				extra = append(extra, columnName)
				continue
			}
			if registeredColumn.SpannerType != actualColumn.SpannerType {
				issues = append(issues, fmt.Sprintf(
					"TABLE %s COLUMN %s: %s type = %s, registry = %s",
					meta.Name, columnName, targetName,
					actualColumn.SpannerType, registeredColumn.SpannerType,
				))
			}
			expectedOrdinal := registeredColumn.OrdinalPosition
			if override, ok := targetOrdinalOverrides[targetName][meta.Name+"."+columnName]; ok {
				expectedOrdinal = override
			}
			if expectedOrdinal != actualColumn.OrdinalPosition {
				issues = append(issues, fmt.Sprintf(
					"TABLE %s COLUMN %s: %s ordinal = %d, expected = %d",
					meta.Name, columnName, targetName,
					actualColumn.OrdinalPosition, expectedOrdinal,
				))
			}
		}

		if len(extra) > 0 {
			sort.Strings(extra)
			issues = append(issues, fmt.Sprintf(
				"TABLE %s: %s has columns not in TableMeta: %s",
				meta.Name, targetName, strings.Join(extra, ", "),
			))
		}
	}

	// Also check for INFORMATION_SCHEMA tables that exist in the target but
	// are not in our AllTableMetas registry.
	metaNames := make(map[string]bool)
	for _, m := range allMetas {
		metaNames[m.Name] = true
	}
	var unknownTables []string
	for tableName := range metadata {
		if !metaNames[tableName] {
			unknownTables = append(unknownTables, tableName)
		}
	}
	if len(unknownTables) > 0 {
		sort.Strings(unknownTables)
		issues = append(issues, fmt.Sprintf(
			"UNKNOWN TABLES in %s not in AllTableMetas: %s",
			targetName, strings.Join(unknownTables, ", "),
		))
	}

	if len(issues) > 0 {
		sort.Strings(issues)
		t.Errorf("drift detected between infoschem.AllTableMetas() and %s:\n  %s",
			targetName, strings.Join(issues, "\n  "))
	}

	schema, err := astconv.LoadSchema(ctx, client)
	if err != nil {
		t.Errorf("LoadSchema against %s: %v", targetName, err)
	}
	return schema
}

func verifyOmittedChangeStreamRetention(t *testing.T, schema *astconv.Schema, name string) {
	t.Helper()
	if schema == nil {
		return
	}
	for _, option := range schema.ChangeStreamOptions {
		if option.ChangeStreamName == name {
			t.Errorf("omitted retention produced CHANGE_STREAM_OPTIONS row: %#v", option)
		}
	}
	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	for _, ddl := range ddls {
		stream, ok := ddl.(*ast.CreateChangeStream)
		if !ok || stream.Name.Name != name {
			continue
		}
		if stream.Options != nil {
			t.Errorf("omitted retention was materialized as %s", stream.Options.SQL())
		}
		return
	}
	t.Errorf("reconstructed DDL missing change stream %s", name)
}

// targetOrdinalOverrides records intentional target-specific layouts for the
// exact images pinned by the drift tests. The registry itself retains the
// managed-Spanner/Omni order used as the production superset.
var targetOrdinalOverrides = map[string]map[string]int{
	"emulator": {
		"COLUMNS.IS_HIDDEN":                   11,
		"COLUMNS.GENERATION_EXPRESSION":       12,
		"COLUMNS.ON_UPDATE_EXPRESSION":        13,
		"COLUMNS.IS_STORED":                   14,
		"COLUMNS.SPANNER_STATE":               15,
		"COLUMNS.IS_IDENTITY":                 16,
		"COLUMNS.IDENTITY_GENERATION":         17,
		"COLUMNS.IDENTITY_KIND":               18,
		"COLUMNS.IDENTITY_START_WITH_COUNTER": 19,
		"COLUMNS.IDENTITY_SKIP_RANGE_MIN":     20,
		"COLUMNS.IDENTITY_SKIP_RANGE_MAX":     21,
		"SCHEMATA.SCHEMA_OWNER":               4,
		"SCHEMATA.PROTO_BUNDLE":               5,
	},
}

func verifyRollingColumnsQueryable(
	ctx context.Context,
	t *testing.T,
	client *spanner.Client,
	discovered infoschem.DiscoveredColumns,
) {
	t.Helper()

	for _, meta := range infoschem.AllTableMetas() {
		hasRollingColumn := false
		for _, column := range meta.Columns {
			if column.Rolling && discovered[meta.Name][column.Name] {
				hasRollingColumn = true
				break
			}
		}
		if !hasRollingColumn {
			continue
		}

		query, err := meta.Query(discovered)
		if err != nil {
			t.Fatalf("build rolling-column query for %s: %v", meta.Name, err)
		}
		iter := client.Single().Query(ctx, spanner.NewStatement(query+" LIMIT 0"))
		_, err = iter.Next()
		iter.Stop()
		if err != nil && err != iterator.Done {
			t.Errorf("query advertised rolling columns from %s: %v", meta.Name, err)
		}
	}
}
