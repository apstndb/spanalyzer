//go:build integration

package main

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanemuboost"
)

func TestIntegrationSQLChoicesOnEmulator(t *testing.T) {
	querygenIntegrationRequireContainerRuntime(t)
	querygenIntegrationSQLChoices(t, querygenIntegrationRuntime, false)
}

type sqlChoiceBorrowedAnalyzer struct{ planReportAnalyzer }

// SetupClients owns the client and database; collection only borrows them.
func (sqlChoiceBorrowedAnalyzer) Close() error { return nil }

func querygenIntegrationSQLChoices(t *testing.T, runtime spanemuboost.RuntimeHandle, checkPlans bool) {
	t.Helper()
	config, dir := sqlChoiceTestConfig(t)
	plan, err := querygen.BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatal(err)
	}
	clients := spanemuboost.SetupClients(t, runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(querygenIntegrationDDLs(t, "schema.sql", sqlChoiceTestDDL)),
	)
	var mutations []*spanner.Mutation
	for _, row := range []struct {
		id, shard, seconds int64
		active, pending    string
	}{{1, 1, 1, "1", "0"}, {2, 1, 2, "1", "1"}, {3, 1, 3, "0", "1"}, {4, 1, 4, "1", "0"}, {5, 2, 2, "1", "1"}} {
		mutations = append(mutations, spanner.Insert("Events", []string{"Id", "Shard", "CreatedAt", "Active", "Pending"}, []interface{}{row.id, row.shard, time.Unix(row.seconds, 0).UTC(), row.active, row.pending}))
	}
	if _, err := clients.Client.Apply(t.Context(), mutations); err != nil {
		t.Fatal(err)
	}
	want := map[string][]int64{
		"both/active": {2}, "both/pending": {2, 3},
		"start/active": {2, 4}, "start/pending": {2, 3},
		"end/active": {1, 2}, "end/pending": {2, 3},
	}
	for _, target := range querygen.ExpandPlanReportQueries(plan.Queries) {
		t.Run(target.ReportVariant.Label, func(t *testing.T) {
			stmt := spanner.NewStatement(target.SQL)
			values, err := planReportParamValues(target.Params)
			if err != nil {
				t.Fatal(err)
			}
			stmt.Params = values
			if _, ok := stmt.Params["start"]; ok {
				stmt.Params["start"] = time.Unix(2, 0).UTC()
			}
			if _, ok := stmt.Params["end"]; ok {
				stmt.Params["end"] = time.Unix(3, 0).UTC()
			}
			var got []int64
			if err := clients.Client.Single().Query(t.Context(), stmt).Do(func(row *spanner.Row) error {
				var id int64
				if err := row.Column(0, &id); err != nil {
					return err
				}
				got = append(got, id)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			slices.Sort(got) // No ORDER BY: compare membership on both backends.
			key := target.ReportVariant.ChoiceAssignments["bounds"] + "/" + target.ReportVariant.ChoiceAssignments["flag"]
			if !reflect.DeepEqual(got, want[key]) {
				t.Fatalf("%s rows = %v, want %v", key, got, want[key])
			}
			if checkPlans {
				p, err := clients.Client.Single().AnalyzeQuery(t.Context(), stmt)
				if err != nil || len(p.GetPlanNodes()) == 0 {
					t.Fatalf("PLAN %s: %v", key, err)
				}
			}
		})
	}
	if checkPlans {
		open := func(context.Context, spanemuboost.RuntimeHandle, querygen.QueryCodegenSchema, string, *spanalyzer.ProtoDescriptorSet, planReportOptimizer) (planReportAnalyzer, error) {
			return sqlChoiceBorrowedAnalyzer{&spannerPlanReportAnalyzer{clients: clients}}, nil
		}
		report, err := buildPlanReportWithAnalyzer(t.Context(), config, plan, dir, planReportOptions{Backend: "omni", Summary: true}, runtime, open)
		if err != nil || report.TargetSummary.Planned != 6 || report.TargetSummary.Errors != 0 || len(report.Queries) != 6 {
			t.Fatalf("SQL choice collection = %+v, %v", report, err)
		}
	}
}
