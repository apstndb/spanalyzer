package querygen

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritePlanMutationCountBounds(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `CREATE TABLE Events(Shard INT64 NOT NULL,Id INT64 NOT NULL,Active STRING(MAX),Note STRING(MAX),Extra STRING(MAX)) PRIMARY KEY(Shard,Id); CREATE INDEX ByActive ON Events(Active) STORING(Note);`)
	config := QueryCodegenConfig{Package: "db", Client: GoStructTargetSpanner, Schemas: []QueryCodegenSchema{{Name: "app", DDL: "schema.sql"}}}
	for _, column := range []string{"Active", "Note", "Extra"} {
		config.Writes = append(config.Writes, QueryCodegenWrite{Name: "Update" + column, Catalog: "app", Table: "Events", Operation: "update", InputStruct: column + "Input", Update: QueryCodegenWriteUpdate{Columns: []string{column}}, Methods: []string{"mutation", "dml"}})
	}
	plan, err := BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, write := range plan.Writes {
		if write.MutationCountError != "" {
			t.Fatal(write.MutationCountError)
		}
		if len(write.MutationCounts) != 2 {
			t.Fatal("missing per-API accounting")
		}
		m := write.MutationCounts["mutation"].Bounds.Count
		d := write.MutationCounts["dml"].Bounds.Count
		if d.Lower != 0 {
			t.Fatal("generated full-key DML can match no row")
		}
		switch write.Name {
		case "UpdateActive":
			if m.Lower != 3 || m.Upper == nil || *m.Upper != 6 {
				t.Fatal(m)
			}
		case "UpdateNote":
			if m.Lower != 5 || m.Upper == nil || *m.Upper != 5 {
				t.Fatal(m)
			}
		case "UpdateExtra":
			if m.Lower != 3 || m.Upper == nil || *m.Upper != 3 {
				t.Fatal(m)
			}
		}
	}
	b, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"mutation_counts"`) || !strings.Contains(string(b), `"index_effects"`) {
		t.Fatal("structured explain projection lost count evidence")
	}
}

func TestWritePlanKeepsUnavailableCountDiagnostic(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `CREATE TABLE T(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id); CREATE TABLE IF NOT EXISTS T(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id);`)
	config := QueryCodegenConfig{Package: "db", Client: GoStructTargetSpanner, Schemas: []QueryCodegenSchema{{Name: "app", DDL: "schema.sql"}}, Writes: []QueryCodegenWrite{{Name: "Insert", Catalog: "app", Table: "T", Operation: "insert", InputStruct: "Input", Insert: QueryCodegenWriteInsert{Columns: []string{"Id", "V"}}, Methods: []string{"mutation"}}}}
	plan, err := BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Writes) != 1 || plan.Writes[0].MutationCountError == "" {
		t.Fatal("unsupported count catalog must remain explicit without breaking valid DTO planning")
	}
}
