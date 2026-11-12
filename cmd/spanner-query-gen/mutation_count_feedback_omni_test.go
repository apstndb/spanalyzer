//go:build integration && omni

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanemuboost"
	"github.com/apstndb/spannerplan/plantree/reference"
)

func TestIntegrationMutationCountFeedbackOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1")
	}
	querygenOmniIntegrationRequireRuntime(t)
	dir := t.TempDir()
	ddl := []string{}
	for _, name := range []string{"Bare", "OneIndex", "TwoIndexes", "NullIndex", "StoredIndex"} {
		ddl = append(ddl, fmt.Sprintf("CREATE TABLE %s(Shard INT64 NOT NULL,Id INT64 NOT NULL,Active BOOL,Note STRING(MAX)) PRIMARY KEY(Shard,Id)", name))
	}
	ddl = append(ddl, "CREATE INDEX OneActive ON OneIndex(Active)", "CREATE INDEX TwoActive ON TwoIndexes(Active)", "CREATE INDEX TwoNote ON TwoIndexes(Note)", "CREATE NULL_FILTERED INDEX NullActive ON NullIndex(Active)", "CREATE INDEX StoredActive ON StoredIndex(Active) STORING(Note)", "CREATE TABLE Owners(Id INT64 NOT NULL) PRIMARY KEY(Id)", "CREATE TABLE ForeignEvents(Shard INT64 NOT NULL,Id INT64 NOT NULL,Owner INT64,Note STRING(MAX),CONSTRAINT EventOwner FOREIGN KEY(Owner) REFERENCES Owners(Id)) PRIMARY KEY(Shard,Id)", "CREATE TABLE AutomaticEvents(Shard INT64 NOT NULL,Id INT64 NOT NULL,Generated INT64 AS (Shard+Id) STORED,Defaulted STRING(MAX) DEFAULT('d')) PRIMARY KEY(Shard,Id)")
	if err := os.WriteFile(filepath.Join(dir, "schema.sql"), []byte(strings.Join(ddl, ";\n")+";"), 0600); err != nil {
		t.Fatal(err)
	}
	config := querygen.QueryCodegenConfig{Package: "db", Client: querygen.GoStructTargetSpanner, Schemas: []querygen.QueryCodegenSchema{{Name: "app", Dialect: "spanner", DDL: "schema.sql"}}, Queries: []querygen.QueryCodegenQuery{{Name: "ForeignSeek", Kind: "sql", Catalog: "app", SQL: "SELECT Shard,Id FROM ForeignEvents@{FORCE_INDEX=EventOwner} WHERE Owner=@owner", Params: []querygen.QueryCodegenParam{{Name: "owner", Type: "INT64"}}, ResultStruct: "ForeignRow"}}}
	inputs := querygen.MutationCountInputs{Writes: map[string]map[string]querygen.MutationCountWriteInput{}}
	full := []string{"Shard", "Id", "Active", "Note"}
	type scenario struct {
		name, table, operation string
		columns                []string
		old                    bool
		want                   int64
	}
	cases := []scenario{{"InsertBare", "Bare", "insert", full, false, 4}, {"InsertOne", "OneIndex", "insert", full, false, 5}, {"InsertTwo", "TwoIndexes", "insert", full, false, 6}, {"InsertKeys", "TwoIndexes", "insert", []string{"Shard", "Id"}, false, 4}, {"InsertNullKeys", "NullIndex", "insert", []string{"Shard", "Id"}, false, 2}, {"InsertStored", "StoredIndex", "insert", full, false, 6}, {"UpdateNote", "OneIndex", "update", []string{"Note"}, true, 3}, {"UpdateStoredNote", "StoredIndex", "update", []string{"Note"}, true, 5}, {"UpdateActive", "OneIndex", "update", []string{"Active"}, true, 5}, {"UpdateStoredActive", "StoredIndex", "update", []string{"Active"}, true, 6}, {"UpsertExisting", "OneIndex", "insert_or_update", []string{"Shard", "Id", "Active"}, true, -1}, {"UpsertMissing", "OneIndex", "insert_or_update", []string{"Shard", "Id", "Active"}, false, -1}, {"ReplacePartial", "OneIndex", "replace", []string{"Shard", "Id", "Active"}, true, -1}, {"DeleteEvent", "OneIndex", "delete", nil, true, -1}, {"InsertForeign", "ForeignEvents", "insert", []string{"Shard", "Id", "Owner", "Note"}, false, 5}, {"InsertAutomatic", "AutomaticEvents", "insert", []string{"Shard", "Id"}, false, -1}}
	for n, c := range cases {
		id := n + 1
		write := querygen.QueryCodegenWrite{Name: c.name, Catalog: "app", Table: c.table, Operation: c.operation, InputStruct: c.name + "Input", Methods: []string{"mutation", "dml"}}
		if c.operation == "replace" {
			write.Methods = []string{"mutation"}
		}
		if c.operation == "update" {
			write.Update.Columns = c.columns
		} else {
			write.Insert.Columns = c.columns
		}
		config.Writes = append(config.Writes, write)
		values := map[string]interface{}{"Shard": "1", "Id": fmt.Sprint(id)}
		for _, column := range c.columns {
			switch column {
			case "Active":
				values[column] = true
			case "Note":
				values[column] = "after"
			case "Owner":
				values[column] = "7"
			}
		}
		seed := []string{fmt.Sprintf("DELETE FROM %s WHERE Shard=1 AND Id=%d", c.table, id)}
		if c.old {
			seed = append(seed, fmt.Sprintf("INSERT INTO %s(Shard,Id,Active,Note) VALUES(1,%d,FALSE,'before')", c.table, id))
		}
		if c.table == "ForeignEvents" {
			seed = append(seed, "DELETE FROM Owners WHERE Id=7", "INSERT INTO Owners(Id) VALUES(7)")
		}
		inputs.Writes[c.name] = map[string]querygen.MutationCountWriteInput{}
		for _, method := range write.Methods {
			inputs.Writes[c.name][method] = querygen.MutationCountWriteInput{Values: values, SeedDML: seed}
		}
	}
	plan := querygen.BuildQueryCodegenReportPlan(config, dir)
	if err := querygen.RefinePlanMutationCounts(plan, config, dir, inputs); err != nil {
		t.Fatal(err)
	}
	runtime := querygenOmniRuntime(t)
	report, err := buildPlanReportWithRuntime(t.Context(), config, plan, dir, planReportOptions{Backend: "omni", Format: reference.FormatCurrent, RenderMode: reference.RenderModePlan, Summary: true, MeasureMutationCounts: true, MutationInputs: inputs}, runtime)
	if err != nil {
		t.Fatal(err)
	}
	provenance, err := spanemuboost.RuntimeImageProvenance(t.Context(), runtime)
	if err != nil {
		t.Fatal(err)
	}
	provenanceJSON, err := json.Marshal(provenance)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("mutation_feedback_provenance=%s", provenanceJSON)
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("mutation_feedback_report=%s", data)
	for _, warning := range report.Warnings {
		if warning.ID == "mutation-count-measurement-failed" || warning.ID == "plan-report-index-observation-error" {
			t.Error(warning)
		}
	}
	var physical string
	for _, q := range report.Queries {
		if q.Name == "ForeignSeek" {
			if q.Status != "ok" {
				t.Error(q.Error)
			}
			for _, path := range q.AccessPaths {
				if slices.Contains(path.InferredConstraintSelectors, "EventOwner:referencing") {
					if path.BaseTable != "ForeignEvents" || path.Managed == nil || !*path.Managed || path.DeclaredKeyCount == nil || *path.DeclaredKeyCount != 1 || !slices.Contains(path.KeyColumns, "Owner") {
						t.Errorf("missing observed scan facts: %+v", path)
					}
					physical = path.CatalogIndexName
				}
			}
			continue
		}
		if q.MutationCountError != "" {
			t.Errorf("%s: %s", q.Name, q.MutationCountError)
		}
		if len(q.MutationCounts) == 0 {
			t.Errorf("%s: missing counts", q.Name)
		}
		for method, count := range q.MutationCounts {
			if count.Observation == nil || !count.Observation.Successful || !count.Observation.StatsPresent || count.Observation.Count == nil || count.Effective.Lower != *count.Observation.Count || count.Effective.Upper == nil || *count.Effective.Upper != *count.Observation.Count || len(count.Disagreements) > 0 {
				t.Errorf("%s/%s: not a matching observed point: %+v", q.Name, method, count)
			}
			for _, c := range cases {
				if c.name == q.Name && c.want >= 0 && count.Observation != nil && count.Observation.Count != nil && *count.Observation.Count != c.want {
					t.Errorf("%s/%s: count=%d want=%d", q.Name, method, *count.Observation.Count, c.want)
				}
			}
			if q.Name == "InsertForeign" {
				found := false
				for _, index := range count.Indexes {
					if index.Name == physical && slices.Contains(index.Constraints, "EventOwner:referencing") {
						found = true
					}
				}
				if !found {
					t.Error("plan scans and mutation accounting did not share the physical index mapping")
				}
			}
		}
	}
	if physical == "" {
		t.Error("foreign-key backing-index scan was not catalogued")
	}
}
