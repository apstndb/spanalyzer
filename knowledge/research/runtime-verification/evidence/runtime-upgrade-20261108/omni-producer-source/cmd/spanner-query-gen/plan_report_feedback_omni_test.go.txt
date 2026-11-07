//go:build integration && omni

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanalyzer/plancontract"
	"github.com/apstndb/spanemuboost"
	"github.com/apstndb/spannerplan/plantree/reference"
)

type feedbackBorrowedPlanAnalyzer struct{ planReportAnalyzer }

// SetupClients owns cleanup; the collector borrows the same local test DB.
func (feedbackBorrowedPlanAnalyzer) Close() error { return nil }

func TestIntegrationPlanReportFeedbackOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	querygenOmniIntegrationRequireRuntime(t)
	runtime := querygenOmniRuntime(t)
	ddl := `CREATE TABLE T (Id INT64 NOT NULL, Value STRING(MAX)) PRIMARY KEY (Id);
CREATE TABLE Events (Shard INT64 NOT NULL, CreatedAt TIMESTAMP NOT NULL, Id INT64 NOT NULL, Owner STRING(MAX)) PRIMARY KEY (Shard, CreatedAt, Id);
CREATE TABLE Surrogates (Id INT64 NOT NULL, CreatedAt TIMESTAMP, Owner STRING(MAX)) PRIMARY KEY (Id);`
	clients := spanemuboost.SetupClients(t, runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(querygenIntegrationDDLs(t, "feedback.sql", ddl)),
		spanemuboost.WithClientConfig(spanner.ClientConfig{QueryOptions: spanner.QueryOptions{Options: &spannerpb.ExecuteSqlRequest_QueryOptions{OptimizerVersion: "9"}}}),
	)
	if _, err := clients.Client.Apply(t.Context(), []*spanner.Mutation{spanner.Insert("T", []string{"Id", "Value"}, []interface{}{int64(1), "before"})}); err != nil {
		t.Fatal(err)
	}
	analyzer := &spannerPlanReportAnalyzer{clients: clients}
	for _, sql := range []string{
		"@{OPTIMIZER_VERSION=9} UPDATE T SET Value=@value WHERE Id=@id",
		"@{OPTIMIZER_VERSION=9} DELETE FROM T WHERE Id=@id",
		"@{OPTIMIZER_VERSION=9} INSERT INTO T (Id, Value) VALUES (@id, @value)",
	} {
		params, err := planReportParamValues([]querygen.QueryCodegenParam{{Name: "id", Type: "INT64"}, {Name: "value", Type: "STRING"}})
		if err != nil {
			t.Fatal(err)
		}
		stmt := spanner.NewStatement(sql)
		stmt.Params = params
		plan, err := analyzer.AnalyzePlan(t.Context(), stmt, true)
		if err != nil || len(plan.GetPlanNodes()) == 0 {
			t.Fatalf("PLAN %s: %v", sql, err)
		}
		row, err := clients.Client.Single().ReadRow(t.Context(), "T", spanner.Key{int64(1)}, []string{"Value"})
		if err != nil {
			t.Fatal(err)
		}
		var value string
		if err := row.Column(0, &value); err != nil || value != "before" {
			t.Fatalf("DML PLAN changed data: %q, %v", value, err)
		}
	}
	t.Run("NestedStructArrayParameter", func(t *testing.T) {
		params, err := planReportParamValues([]querygen.QueryCodegenParam{{Name: "items", Type: "ARRAY<STRUCT<id INT64, nested STRUCT<label STRING, tags ARRAY<STRING>>>>"}})
		if err != nil {
			t.Fatal(err)
		}
		stmt := spanner.NewStatement("@{OPTIMIZER_VERSION=9} SELECT item.id, item.nested.label, item.nested.tags FROM UNNEST(@items) AS item JOIN T ON T.Id=item.id")
		stmt.Params = params
		plan, err := analyzer.AnalyzePlan(t.Context(), stmt, false)
		if err != nil || len(plan.GetPlanNodes()) == 0 {
			t.Fatalf("nested array PLAN: %v", err)
		}
	})
	params, err := planReportParamValues([]querygen.QueryCodegenParam{{Name: "shard", Type: "INT64"}, {Name: "start", Type: "TIMESTAMP"}, {Name: "owner", Type: "STRING"}, {Name: "id", Type: "INT64"}})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, sql, target string
		seek, residual    []string
		full              bool
	}{
		{"LeadingKeyAndRange", "SELECT Id FROM Events WHERE Shard=@shard AND CreatedAt>=@start AND Owner=@owner", "Events", []string{"CreatedAt", "Shard"}, []string{"Owner"}, false},
		{"SkippedMiddleKey", "SELECT Id FROM Events WHERE Shard=@shard AND Id=@id", "Events", []string{"Shard"}, []string{"Id"}, false},
		{"SurrogateNonKeyFilters", "SELECT Id FROM Surrogates WHERE Owner=@owner AND CreatedAt>=@start", "Surrogates", nil, []string{"CreatedAt", "Owner"}, true},
		{"MixedSeekAndFullScan", "SELECT e.Id,l.Id AS latest FROM Events AS e CROSS JOIN (SELECT Id FROM Surrogates ORDER BY CreatedAt DESC LIMIT 1) AS l WHERE e.Shard=@shard AND e.CreatedAt>=@start", "Events", []string{"CreatedAt", "Shard"}, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := spanner.NewStatement("@{OPTIMIZER_VERSION=9} " + tc.sql)
			stmt.Params = params
			plan, err := analyzer.AnalyzePlan(t.Context(), stmt, false)
			if err != nil {
				t.Fatal(err)
			}
			paths := plancontract.NormalizeAccessPaths(plan, nil)
			t.Logf("access paths: %+v", paths)
			found := false
			for _, path := range paths {
				if path.ScanTarget != tc.target {
					continue
				}
				found = true
				if !path.ConditionsComplete || path.FullScan != tc.full || !slices.Equal(path.SeekColumns, tc.seek) || !slices.Equal(path.ResidualColumns, tc.residual) {
					t.Fatalf("access path: %+v", path)
				}
			}
			if !found {
				t.Fatalf("target %s missing in %+v", tc.target, paths)
			}
			if tc.name == "MixedSeekAndFullScan" {
				if !slices.ContainsFunc(paths, func(p plancontract.AccessPath) bool {
					return p.ScanTarget == "Surrogates" && p.FullScan && !p.HasSeekCondition
				}) {
					t.Fatalf("separate full scan missing: %+v", paths)
				}
			}
		})
	}
	t.Run("EndToEndReportCollection", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "schema.sql"), []byte(ddl), 0600); err != nil {
			t.Fatal(err)
		}
		config := querygen.QueryCodegenConfig{Package: "db", Schemas: []querygen.QueryCodegenSchema{{Name: "app", Dialect: "spanner", DDL: "schema.sql"}}, Queries: []querygen.QueryCodegenQuery{
			{Name: "Optional", Catalog: "app", Kind: "sql", ResultStruct: "OptionalRow", SQL: "SELECT Id FROM T WHERE TRUE /*?optional:id*/ AND Id=@id /*?end*/ /*?optional:value*/ AND Value=@value /*?end*/", Params: []querygen.QueryCodegenParam{{Name: "id", Type: "INT64", Optional: "omit_when_null"}, {Name: "value", Type: "STRING", Optional: "omit_when_null"}}},
			{Name: "Delete", Catalog: "app", Kind: "sql", Result: "row_count", SQL: "DELETE FROM T WHERE Id=@id", Params: []querygen.QueryCodegenParam{{Name: "id", Type: "INT64"}}},
			{Name: "UpdateReturning", Catalog: "app", Kind: "sql", Result: "row_set", ResultStruct: "UpdatedRow", SQL: "UPDATE T SET Value=@value WHERE Id=@id THEN RETURN Id", Params: []querygen.QueryCodegenParam{{Name: "id", Type: "INT64"}, {Name: "value", Type: "STRING"}}},
			{Name: "BadLimit", Catalog: "app", Kind: "sql", ResultStruct: "BadRow", SQL: "SELECT Id FROM T LIMIT @limit", Params: []querygen.QueryCodegenParam{{Name: "limit", Type: "STRING"}}},
			{Name: "Nested", Catalog: "app", Kind: "sql", ResultStruct: "NestedRow", SQL: "SELECT item.id FROM UNNEST(@items) AS item", Params: []querygen.QueryCodegenParam{{Name: "items", Type: "ARRAY<STRUCT<id INT64>>"}}},
			{Name: "Range", Catalog: "app", Kind: "sql", ResultStruct: "EventRow", SQL: "SELECT Id FROM Events WHERE Shard=@shard AND CreatedAt>=@start", Params: []querygen.QueryCodegenParam{{Name: "shard", Type: "INT64"}, {Name: "start", Type: "TIMESTAMP"}}},
		}, Writes: []querygen.QueryCodegenWrite{{Name: "Remove", Catalog: "app", Table: "T", Operation: "delete", InputStruct: "RemoveInput", Methods: []string{"dml"}}}}
		model := querygen.BuildQueryCodegenReportPlan(config, dir)
		open := func(context.Context, spanemuboost.RuntimeHandle, querygen.QueryCodegenSchema, string, *spanalyzer.ProtoDescriptorSet, planReportOptimizer) (planReportAnalyzer, error) {
			return feedbackBorrowedPlanAnalyzer{analyzer}, nil
		}
		report, err := buildPlanReportWithAnalyzer(t.Context(), config, model, dir, planReportOptions{Backend: "omni", Format: reference.FormatCurrent, RenderMode: reference.RenderModePlan, AnnotateSeekability: true, Optimizer: planReportOptimizerEnvironment{Version: "9"}}, runtime, open)
		if err != nil || report.TargetSummary.Planned != 9 || report.TargetSummary.Errors != 1 || len(report.Queries) != 10 {
			t.Fatalf("report = %+v, %v", report, err)
		}
		ids := map[string]bool{}
		for _, query := range report.Queries {
			if ids[query.TargetID] {
				t.Fatalf("duplicate target %s", query.TargetID)
			}
			ids[query.TargetID] = true
			if query.Name == "BadLimit" && query.Status != "error" {
				t.Fatalf("static failure lost: %+v", query)
			}
			if query.Name == "Range" && (!strings.Contains(query.Plan, "2/3") || len(query.AccessPaths) != 1 || query.AccessPaths[0].SeekableKeySize != "2") {
				t.Fatalf("annotated range = %+v", query)
			}
		}
		if !ids["query/Optional"] || !ids["write/Remove"] {
			t.Fatalf("canonical identities missing: %v", ids)
		}
		var output bytes.Buffer
		if err := writePlanReport(&output, "json", report); err != nil {
			t.Fatal(err)
		}
		row, err := clients.Client.Single().ReadRow(t.Context(), "T", spanner.Key{int64(1)}, []string{"Value"})
		if err != nil {
			t.Fatal(err)
		}
		var value string
		if err := row.Column(0, &value); err != nil || value != "before" {
			t.Fatalf("report DML changed data: %q, %v", value, err)
		}
	})
}
