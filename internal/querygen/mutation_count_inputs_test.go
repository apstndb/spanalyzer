package querygen

import (
	"path/filepath"
	"testing"

	spanalyzer "github.com/apstndb/spanalyzer"
)

func TestMutationInputsNarrowBothAPIsAndRejectMisspelledTargets(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `CREATE TABLE Events(Id INT64 NOT NULL,Active STRING(MAX),Note STRING(MAX)) PRIMARY KEY(Id); CREATE INDEX EventsByActive ON Events(Active);`)
	config := QueryCodegenConfig{Package: "db", Client: GoStructTargetSpanner, Schemas: []QueryCodegenSchema{{Name: "app", DDL: "schema.sql"}}, Writes: []QueryCodegenWrite{{Name: "UpdateActive", Catalog: "app", Table: "Events", Operation: "update", InputStruct: "Input", Update: QueryCodegenWriteUpdate{Columns: []string{"Active"}}, Methods: []string{"mutation", "dml"}}}}
	plan, err := BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatal(err)
	}
	exists := true
	one := int64(1)
	input := MutationCountWriteInput{Values: map[string]interface{}{"Id": "1", "Active": "after"}, Facts: []spanalyzer.MutationCountRowFacts{{Exists: &exists, Rows: &one, Old: map[string]interface{}{"Id": "1", "Active": "before", "Note": "n"}}}}
	if err := RefinePlanMutationCounts(plan, config, dir, MutationCountInputs{Writes: map[string]map[string]MutationCountWriteInput{"UpdateActive": {"mutation": input, "dml": input}}}); err != nil {
		t.Fatal(err)
	}
	// Both channels charge the identifying primary-key cell in these retained
	// managed/Omni update controls, as well as the updated value and index rows.
	for method, want := range map[string]int64{"mutation": 4, "dml": 4} {
		r := plan.Writes[0].MutationCounts[method]
		if r.Effective.Lower != want || r.Effective.Upper == nil || *r.Effective.Upper != want || r.OldRows == nil {
			t.Fatalf("%s refinement = %+v", method, r)
		}
	}
	for _, inputs := range []MutationCountInputs{
		{Writes: map[string]map[string]MutationCountWriteInput{"Typo": {"mutation": input}}},
		{Writes: map[string]map[string]MutationCountWriteInput{"UpdateActive": {"typo": input}}},
		{Catalogs: map[string]spanalyzer.MutationCountIndexSnapshot{"Typo": {}}},
		{Catalogs: map[string]spanalyzer.MutationCountIndexSnapshot{"app": {SchemaSHA256: "stale"}}},
	} {
		if err := RefinePlanMutationCounts(plan, config, dir, inputs); err == nil {
			t.Fatal("invalid mutation inputs accepted", inputs)
		}
	}
}

func TestReportPlanRetainsMutationOnlyCounts(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `CREATE TABLE Events(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id);`)
	config := QueryCodegenConfig{Package: "db", Client: GoStructTargetSpanner, Schemas: []QueryCodegenSchema{{Name: "app", DDL: "schema.sql"}}, Writes: []QueryCodegenWrite{{Name: "InsertEvent", Catalog: "app", Table: "Events", Operation: "insert", InputStruct: "Input", Insert: QueryCodegenWriteInsert{Columns: []string{"Id", "V"}}, Methods: []string{"mutation"}}}}
	plan := BuildQueryCodegenReportPlan(config, dir)
	if len(plan.Writes) != 1 || plan.Writes[0].MutationCounts["mutation"].Effective.Lower != 2 || len(plan.Queries) != 1 || plan.Queries[0].ReportSkipReason == "" {
		t.Fatalf("report omitted independent write accounting: %+v", plan)
	}
}
