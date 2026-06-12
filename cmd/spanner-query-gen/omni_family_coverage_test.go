//go:build integration && omni

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanalyzer/plancontract"
	"github.com/apstndb/spanemuboost"
	"github.com/apstndb/spannerplan/plantree/reference"
)

// familyCoverageDDL extends the basic integration schema with shapes needed
// to observe operator families that plain scan/join queries never produce:
// interleaved children for colocated (anti) semi applies, a non-interleaved
// table for bloom-filter hash joins, and a search index for full-text search
// operators.
const familyCoverageDDL = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  Rating INT64 NOT NULL
) PRIMARY KEY (SingerId);

CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  Title STRING(MAX)
) PRIMARY KEY (SingerId, AlbumId),
  INTERLEAVE IN PARENT Singers ON DELETE CASCADE;

CREATE TABLE Songs (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  TrackId INT64 NOT NULL,
  SongName STRING(MAX)
) PRIMARY KEY (SingerId, AlbumId, TrackId),
  INTERLEAVE IN PARENT Albums ON DELETE CASCADE;

CREATE TABLE Concerts (
  VenueId INT64 NOT NULL,
  SingerId INT64 NOT NULL,
  ConcertDate TIMESTAMP NOT NULL
) PRIMARY KEY (VenueId, SingerId, ConcertDate);

CREATE TABLE SearchAlbums (
  SingerId INT64 NOT NULL,
  AlbumId STRING(MAX) NOT NULL,
  AlbumTitle STRING(MAX),
  AlbumTitle_Tokens TOKENLIST AS (TOKENIZE_FULLTEXT(AlbumTitle)) HIDDEN
) PRIMARY KEY(SingerId, AlbumId);

CREATE SEARCH INDEX SearchAlbumsTitleIndex ON SearchAlbums(AlbumTitle_Tokens);
`

const familyCoverageConfigYAML = `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
  ddl: schema.sql
queries:
- name: UnitRelation
  catalog: app
  kind: sql
  sql: SELECT 1 AS x
  result: {struct: UnitRow}
- name: EmptyRelation
  catalog: app
  kind: sql
  sql: SELECT SingerId FROM Singers WHERE FALSE
  result: {struct: EmptyRow}
- name: HavingFilter
  catalog: app
  kind: sql
  sql: SELECT Rating, COUNT(*) AS c FROM Singers GROUP BY Rating HAVING COUNT(*) > 1
  result: {struct: HavingRow}
- name: MinorSortAlbums
  catalog: app
  kind: sql
  sql: SELECT SingerId, AlbumId, Title FROM Albums ORDER BY SingerId, Title
  result: {struct: MinorSortRow}
- name: SpooledCte
  catalog: app
  kind: sql
  sql: WITH r AS (SELECT SingerId FROM Singers WHERE Rating > 3) SELECT SingerId FROM r UNION ALL SELECT SingerId FROM r
  result: {struct: SpoolRow}
- name: BloomFilterHashJoin
  catalog: app
  kind: sql
  sql: SELECT s.FirstName, c.VenueId FROM Singers AS s JOIN@{JOIN_METHOD=HASH_JOIN} Concerts AS c ON s.SingerId = c.SingerId
  result: {struct: BloomRow}
- name: ColocatedSemiApply
  catalog: app
  kind: sql
  sql: SELECT a.Title FROM Albums AS a WHERE EXISTS (SELECT 1 FROM Songs AS so WHERE so.SingerId = a.SingerId AND so.AlbumId = a.AlbumId)
  result: {struct: SemiRow}
- name: ColocatedAntiSemiApply
  catalog: app
  kind: sql
  sql: SELECT a.Title FROM Albums AS a WHERE NOT EXISTS (SELECT 1 FROM Songs AS so WHERE so.SingerId = a.SingerId AND so.AlbumId = a.AlbumId)
  result: {struct: AntiSemiRow}
- name: FullTextSearch
  catalog: app
  kind: sql
  sql: SELECT AlbumId FROM SearchAlbums WHERE SEARCH(AlbumTitle_Tokens, "friday")
  result: {struct: SearchRow}
- name: BernoulliSample
  catalog: app
  kind: sql
  sql: SELECT SingerId FROM Singers TABLESAMPLE BERNOULLI (10 PERCENT)
  result: {struct: BernoulliRow}
- name: ReservoirSample
  catalog: app
  kind: sql
  sql: SELECT SingerId FROM Singers TABLESAMPLE RESERVOIR (5 ROWS)
  result: {struct: ReservoirRow}
`

// familyCoverageExpectedFamilies lists the operator families each query was
// added to observe. The assertion is subset-based so unrelated plan details
// (distributed union shapes, filter scans) can evolve without breaking the
// test, while a silent loss of a target family classification still fails.
var familyCoverageExpectedFamilies = map[string][]string{
	"UnitRelation":  {"unit_relation"},
	"EmptyRelation": {"empty_relation"},
	// The unhinted GROUP BY aggregate method follows available input
	// orderings: with no index on Rating in familyCoverageDDL the plan
	// deterministically uses a hash aggregate (verified 3x on this Omni
	// build); an index on the grouping key would switch it to a stream
	// aggregate.
	"HavingFilter":           {"filter", "hash_aggregate"},
	"MinorSortAlbums":        {"minor_sort"},
	"SpooledCte":             {"spool_build", "spool_scan", "union_all", "union_input"},
	"BloomFilterHashJoin":    {"hash_join", "bloom_filter_build"},
	"ColocatedSemiApply":     {"semi_apply"},
	"ColocatedAntiSemiApply": {"anti_semi_apply"},
	"FullTextSearch":         {"search_predicate", "search_query_conversion_tvf", "verify_determinism"},
	// TABLESAMPLE produces Random ID Assign combined with a Filter
	// (Bernoulli) or a Sort plus Limit (Reservoir), matching the official
	// query-operators-unary documentation.
	"BernoulliSample": {"random_id_assign", "filter"},
	"ReservoirSample": {"random_id_assign", "full_sort", "limit"},
}

// TestIntegrationPlanReportOperatorFamilyCoverageOnOmni verifies on a live
// Spanner Omni backend that plans designed to produce rarely-observed
// operator families classify into those families with no unknown relational
// operators and no classification warnings.
func TestIntegrationPlanReportOperatorFamilyCoverageOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	querygenIntegrationRequireContainerRuntime(t)

	dir := t.TempDir()
	writeIntegrationTestFile(t, filepath.Join(dir, "schema.sql"), familyCoverageDDL)
	config, err := querygen.ParseQueryCodegenConfigYAML([]byte(familyCoverageConfigYAML))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v", err)
	}
	plan, err := querygen.BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}

	runtime := spanemuboost.NewLazyRuntime(spanemuboost.BackendOmni)
	t.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			t.Errorf("failed to close Spanner Omni runtime: %v", err)
		}
	})
	report, err := buildPlanReportWithRuntime(t.Context(), config, plan, dir, planReportOptions{
		Backend:    "omni",
		Format:     reference.FormatCurrent,
		RenderMode: reference.RenderModePlan,
	}, runtime)
	if err != nil {
		t.Fatalf("buildPlanReportWithRuntime() error = %v", err)
	}
	if got, want := len(report.Queries), len(familyCoverageExpectedFamilies); got != want {
		t.Fatalf("plan-report query count = %d, want %d", got, want)
	}
	for _, query := range report.Queries {
		query := query
		t.Run(query.Name, func(t *testing.T) {
			if query.Status != "ok" {
				t.Fatalf("status = %q, error = %q", query.Status, query.Error)
			}
			if got := query.OperatorFamilyCounts["unknown"]; got != 0 {
				t.Errorf("unknown operator family count = %d, want 0\nplan:\n%s", got, query.Plan)
			}
			if len(query.ClassificationNotes) != 0 {
				t.Errorf("classification warnings = %+v, want none", query.ClassificationNotes)
			}
			observed := make(map[string]bool, len(query.OperatorFamilies))
			for _, family := range query.OperatorFamilies {
				observed[family] = true
			}
			for _, family := range familyCoverageExpectedFamilies[query.Name] {
				if !observed[family] {
					t.Errorf("expected operator family %q was not observed; families = %v\nplan:\n%s", family, query.OperatorFamilies, query.Plan)
				}
			}
		})
	}
}

// dmlFamilyCoverageCases pin the operator families observed for DML plans on
// Spanner Omni. DML targets are excluded from plan-report's public config,
// so these go through ReadWriteTransaction.AnalyzeQuery directly (PLAN mode,
// nothing is executed) and classify the raw plan with plancontract. Note
// that the RowCount and MiniBatch* operators are unrelated to DML despite
// the row_count config-mode naming: they are undocumented SELECT back-join
// operators observed only on Cloud Spanner optimizer v5, never on Omni.
var dmlFamilyCoverageCases = []struct {
	Name     string
	SQL      string
	Expected []string
}{
	{"InsertValues", "INSERT INTO Singers (SingerId, FirstName, Rating) VALUES (1, 'A', 1)", []string{"apply_mutations", "union_all", "union_input", "unit_relation"}},
	{"InsertThenReturn", "INSERT INTO Singers (SingerId, FirstName, Rating) VALUES (2, 'B', 1) THEN RETURN SingerId", []string{"apply_mutations", "serialize_result"}},
	{"UpdateWhere", "UPDATE Singers SET FirstName = 'C' WHERE SingerId = 1", []string{"apply_mutations", "scan"}},
	{"DeleteWhere", "DELETE FROM Singers WHERE SingerId = 1", []string{"apply_mutations", "scan"}},
	{"InsertSelect", "INSERT INTO Singers (SingerId, FirstName, Rating) SELECT SingerId + 100, FirstName, Rating FROM Singers", []string{"apply_mutations", "distributed_union", "scan"}},
}

// TestIntegrationDMLOperatorFamilyCoverageOnOmni verifies on live Spanner
// Omni that DML execution plans classify without unknown operator families
// or classification warnings, and that Apply Mutations is observed for every
// DML shape.
func TestIntegrationDMLOperatorFamilyCoverageOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	querygenIntegrationRequireContainerRuntime(t)

	runtime := spanemuboost.NewLazyRuntime(spanemuboost.BackendOmni)
	t.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			t.Errorf("failed to close Spanner Omni runtime: %v", err)
		}
	})
	clients := spanemuboost.SetupClients(t, runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(querygenIntegrationDDLs(t, "schema.sql", familyCoverageDDL)),
	)
	for _, c := range dmlFamilyCoverageCases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			var operators []plancontract.Operator
			_, err := clients.Client.ReadWriteTransaction(t.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				plan, err := txn.AnalyzeQuery(ctx, spanner.NewStatement(c.SQL))
				if err != nil {
					return err
				}
				operators = plancontract.NormalizeOperators(plan)
				return nil
			})
			if err != nil {
				t.Fatalf("AnalyzeQuery(%s) error = %v", c.SQL, err)
			}
			counts := plancontract.OperatorFamilyCounts(operators)
			if got := counts["unknown"]; got != 0 {
				t.Errorf("unknown operator family count = %d, want 0", got)
			}
			if warnings := plancontract.ClassificationWarnings(operators); len(warnings) != 0 {
				t.Errorf("classification warnings = %+v, want none", warnings)
			}
			observed := make(map[string]bool)
			for _, family := range plancontract.ObservedOperatorFamilies(operators) {
				observed[family] = true
			}
			for _, family := range c.Expected {
				if !observed[family] {
					t.Errorf("expected operator family %q was not observed; families = %v", family, plancontract.ObservedOperatorFamilies(operators))
				}
			}
		})
	}
}
