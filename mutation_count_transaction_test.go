package spanalyzer

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

const mixedMutationDDL = `CREATE TABLE A(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id);
CREATE TABLE B(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id);`

func TestMutationCountMixedBoundariesAndObservation(t *testing.T) {
	a := mutationTestAnalyzer(t, mixedMutationDDL)
	input := MutationCountTransaction{
		Statements: []string{"UPDATE A SET V='a' WHERE TRUE", "UPDATE A SET V='b' WHERE TRUE"},
		Mutations:  json.RawMessage(`[{"insert":{"table":"B","columns":["Id","V"],"values":[["1","x"]]}}]`),
	}
	opts := MutationCountOptions{Facts: []MutationCountRowFacts{{Rows: mutationTestInt(30000)}, {Rows: mutationTestInt(30000)}, {}}}
	r, err := a.AnalyzeTransaction(input, opts)
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Effective, 120002, mutationTestInt(120002))
	if r.Scope != "mixed_dml_mutation_transaction" || r.Bounds.LimitVerdict != "within_each_model_boundary_only" || len(r.Bounds.Statements) != 2 {
		t.Fatal(r)
	}
	assertMutationInterval(t, r.Bounds.BufferedCommit.Count, 2, mutationTestInt(2))
	opts.Observation = &MutationCountObservation{Runtime: "test-input-only", ObservedAt: "2026-11-12T00:00:00Z", SchemaSHA256: r.SchemaSHA256, InputSHA256: r.InputSHA256, Scope: r.Scope, Successful: true, StatsPresent: true, Count: mutationTestInt(120002)}
	measured, err := a.AnalyzeTransaction(input, opts)
	if err != nil || measured.EffectiveBasis != "same_execution_observation" {
		t.Fatal(measured, err)
	}
	// A cumulative measurement must not turn into a buffered or statement count.
	assertMutationInterval(t, measured.Bounds.BufferedCommit.Count, 2, mutationTestInt(2))
	assertMutationInterval(t, measured.Bounds.Statements[0].Count, 60000, mutationTestInt(60000))
	for _, subset := range []string{"mutation_api_commit", "dml_transaction"} {
		opts.Observation.Scope = subset
		if _, err := a.AnalyzeTransaction(input, opts); err == nil {
			t.Fatal("accepted subset observation", subset)
		}
	}
	opts.Observation = nil
	opts.Facts[0].Rows = mutationTestInt(40001)
	r, err = a.AnalyzeTransaction(input, opts)
	if err != nil || r.Bounds.LimitVerdict != "model_exceeds_transaction_component_limit" {
		t.Fatal(r, err)
	}
}

func TestMutationCountMixedBufferedLimit(t *testing.T) {
	a := mutationTestAnalyzer(t, mixedMutationDDL)
	var values []string
	for n := range 40001 {
		values = append(values, fmt.Sprintf(`["%d","x"]`, n))
	}
	r, err := a.AnalyzeTransaction(MutationCountTransaction{
		Statements: []string{"INSERT INTO A(Id,V) VALUES(1,'x')"},
		Mutations:  json.RawMessage(`[{"insert":{"table":"B","columns":["Id","V"],"values":[` + strings.Join(values, ",") + `]}}]`),
	}, MutationCountOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Bounds.BufferedCommit.Count, 80002, mutationTestInt(80002))
	if r.Bounds.BufferedCommit.Verdict != "model_exceeds_mutation_limit" || r.Bounds.LimitVerdict != "model_exceeds_transaction_component_limit" {
		t.Fatal(r.Bounds)
	}
}

func TestMutationCountMixedRelatedWrites(t *testing.T) {
	for _, tc := range []struct{ name, ddl, target string }{
		{"same-table", mixedMutationDDL, "A"},
		{"foreign-key", mixedMutationDDL + `CREATE TABLE C(Id INT64 NOT NULL,V STRING(MAX),FOREIGN KEY(Id) REFERENCES A(Id)) PRIMARY KEY(Id);`, "C"},
		{"interleave", mixedMutationDDL + `CREATE TABLE C(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id), INTERLEAVE IN PARENT A ON DELETE CASCADE;`, "C"},
		{"transitive", mixedMutationDDL + `CREATE TABLE C(Id INT64 NOT NULL,V STRING(MAX),FOREIGN KEY(Id) REFERENCES A(Id)) PRIMARY KEY(Id); CREATE TABLE D(Id INT64 NOT NULL,V STRING(MAX),FOREIGN KEY(Id) REFERENCES C(Id)) PRIMARY KEY(Id);`, "D"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := mutationTestAnalyzer(t, tc.ddl)
			r, err := a.AnalyzeTransaction(MutationCountTransaction{
				Statements: []string{"INSERT INTO A(Id,V) VALUES(1,'x')"},
				Mutations:  json.RawMessage(`[{"update":{"table":"` + tc.target + `","columns":["Id","V"],"values":[["1","y"]]}}]`),
			}, MutationCountOptions{})
			if err != nil {
				t.Fatal(err)
			}
			assertMutationInterval(t, r.Effective, 0, nil)
			assertMutationInterval(t, r.Bounds.BufferedCommit.Count, 0, nil)
			if r.Bounds.LimitVerdict != "indeterminate" || !slices.Contains(r.Bounds.Reasons, "mixed_stage_related_writes_not_calibrated") {
				t.Fatal(r.Bounds)
			}
		})
	}
}

func TestMutationCountMixedInputIdentity(t *testing.T) {
	a := mutationTestAnalyzer(t, mixedMutationDDL)
	sql := []string{"INSERT INTO A(Id,V) VALUES(1,'x')"}
	mutations := json.RawMessage(`[{"insert":{"table":"B","columns":["Id","V"],"values":[["1","x"]]}}]`)
	base, err := a.AnalyzeTransaction(MutationCountTransaction{Statements: sql, Mutations: mutations}, MutationCountOptions{})
	if err != nil {
		t.Fatal(err)
	}
	equivalent, err := a.AnalyzeTransaction(MutationCountTransaction{Statements: sql, Mutations: json.RawMessage(`[{"insert": {"values": [["1", "x"]], "columns": ["Id", "V"], "table": "B"}}]`)}, MutationCountOptions{})
	if err != nil || base.InputSHA256 != equivalent.InputSHA256 {
		t.Fatal("object layout must not change input identity", equivalent, err)
	}
	for _, input := range []MutationCountTransaction{
		{Mutations: mutations}, {Statements: sql}, {Statements: sql, Mutations: json.RawMessage(`[]`)},
		{Statements: []string{"SELECT 1"}, Mutations: mutations},
		{Statements: sql, Mutations: json.RawMessage(`[{"insert":{"table":"Missing"}}]`)},
	} {
		if _, err := a.AnalyzeTransaction(input, MutationCountOptions{}); err == nil {
			t.Fatal("invalid mixed input accepted", input)
		}
	}
	for _, changed := range []MutationCountTransaction{
		{Statements: []string{sql[0] + " "}, Mutations: mutations},
		{Statements: sql, Mutations: json.RawMessage(`[{"insert":{"table":"B","columns":["Id","V"],"values":[["2","x"]]}}]`)},
	} {
		r, err := a.AnalyzeTransaction(changed, MutationCountOptions{})
		if err != nil || r.InputSHA256 == base.InputSHA256 {
			t.Fatal("changed input reused identity", r, err)
		}
	}
}
