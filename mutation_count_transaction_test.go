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

func TestMutationCountMixedDisjointCompositeKeys(t *testing.T) {
	a := mutationTestAnalyzer(t, `CREATE TABLE Events(Shard INT64 NOT NULL,Id INT64 NOT NULL,Note STRING(MAX)) PRIMARY KEY(Shard,Id);`)
	input := MutationCountTransaction{
		Statements: []string{"UPDATE Events SET Note='n' WHERE Shard=1 AND Id=2"},
		Mutations:  json.RawMessage(`[{"insert":{"table":"Events","columns":["Shard","Id"],"values":[["1","3"]]}}]`),
	}
	r, err := a.AnalyzeTransaction(input, MutationCountOptions{Facts: []MutationCountRowFacts{
		{Rows: mutationTestInt(1), Exists: mutationTestBool(true), Old: map[string]interface{}{"Note": "old"}}, {},
	}})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Effective, 0, mutationTestInt(5))
	assertMutationInterval(t, r.OldRows.Statements[0].Count, 3, mutationTestInt(3))
	assertMutationInterval(t, r.OldRows.BufferedCommit.Count, 2, mutationTestInt(2))
	if !slices.Contains(r.Bounds.Reasons, "mixed_stage_disjoint_primary_keys_upper_bound") || r.Bounds.LimitVerdict != "within_each_model_boundary_only" || r.Bounds.GateEligible {
		t.Fatal(r.Bounds)
	}
	// Key extraction must not silently turn the predicate into a row-count
	// assertion, a written PK, or an exact additive transaction estimate.
	unknownRows, err := a.AnalyzeTransaction(input, MutationCountOptions{})
	if err != nil || unknownRows.Effective.Upper != nil {
		t.Fatal(unknownRows, err)
	}
	ops, err := a.dmlOperations(input.Statements)
	if err != nil || len(ops) != 1 || len(ops[0].Columns) != 1 || len(ops[0].Values) != 1 {
		t.Fatal("predicate changed the written columns", ops, err)
	}
}

func TestMutationCountMixedPrimaryKeyProof(t *testing.T) {
	for _, tc := range []struct {
		name, sql, mutation string
		disjoint            bool
	}{
		{"insert", "INSERT INTO A(Id,V) VALUES(1,'a')", `{"insert":{"table":"A","columns":["Id","V"],"values":[["2","b"]]}}`, true},
		{"multiple-dml-rows", "INSERT INTO A(Id,V) VALUES(1,'a'),(2,'b')", `{"update":{"table":"A","columns":["Id","V"],"values":[["3","b"]]}}`, true},
		{"reversed-and-extra-filter", "UPDATE A SET V='a' WHERE (1 = id) AND (V='old' OR V='other')", `{"insertOrUpdate":{"table":"a","columns":["Id","V"],"values":[["2","b"]]}}`, true},
		{"buffered-replace", "INSERT INTO A(Id,V) VALUES(1,'a')", `{"replace":{"table":"A","columns":["Id","V"],"values":[["2","b"]]}}`, true},
		{"dml-delete", "DELETE FROM A WHERE Id=1", `{"insert":{"table":"A","columns":["Id","V"],"values":[["2","b"]]}}`, true},
		{"buffered-point-delete", "INSERT INTO A(Id,V) VALUES(1,'a')", `{"delete":{"table":"A","keySet":{"keys":[["2"]]}}}`, true},
		{"same-key-normalized", "UPDATE A SET V='a' WHERE Id=1", `{"update":{"table":"A","columns":["Id","V"],"values":[["01","b"]]}}`, false},
		{"one-overlapping-row", "INSERT INTO A(Id,V) VALUES(1,'a'),(2,'b')", `{"update":{"table":"A","columns":["Id","V"],"values":[["3","b"],["2","b"]]}}`, false},
		{"parameter", "UPDATE A SET V='a' WHERE Id=@id", `{"update":{"table":"A","columns":["Id","V"],"values":[["2","b"]]}}`, false},
		{"range", "UPDATE A SET V='a' WHERE Id<2", `{"update":{"table":"A","columns":["Id","V"],"values":[["3","b"]]}}`, false},
		{"or", "UPDATE A SET V='a' WHERE Id=1 OR Id=2", `{"update":{"table":"A","columns":["Id","V"],"values":[["3","b"]]}}`, false},
		{"not", "UPDATE A SET V='a' WHERE NOT(Id=1)", `{"update":{"table":"A","columns":["Id","V"],"values":[["2","b"]]}}`, false},
		{"expression", "UPDATE A SET V='a' WHERE Id=1+0", `{"update":{"table":"A","columns":["Id","V"],"values":[["2","b"]]}}`, false},
		{"qualified-target", "UPDATE A AS x SET V='a' WHERE x.Id=1", `{"update":{"table":"A","columns":["Id","V"],"values":[["2","b"]]}}`, false},
		{"insert-select", "INSERT INTO A(Id,V) SELECT 1,'a'", `{"update":{"table":"A","columns":["Id","V"],"values":[["2","b"]]}}`, false},
		{"buffered-range-delete", "INSERT INTO A(Id,V) VALUES(1,'a')", `{"delete":{"table":"A","keySet":{"ranges":[{"startClosed":["2"],"endClosed":["3"]}]}}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := mutationTestAnalyzer(t, mixedMutationDDL)
			input := MutationCountTransaction{Statements: []string{tc.sql}, Mutations: json.RawMessage("[" + tc.mutation + "]")}
			ops, err := a.dmlOperations(input.Statements)
			if err != nil {
				t.Fatal(err)
			}
			buffered, err := decodeMutationCountInput(a.schema, input.Mutations)
			if err != nil {
				t.Fatal(err)
			}
			facts := make([]MutationCountRowFacts, len(ops)+len(buffered))
			for i := range facts {
				// An old image and cardinality must never manufacture a
				// predicate key for the unresolved cases above.
				facts[i] = MutationCountRowFacts{Rows: mutationTestInt(1)}
				if i < len(ops) && ops[i].Kind == "update" {
					facts[i].Exists = mutationTestBool(true)
					facts[i].Old = map[string]interface{}{"Id": "1", "V": "old"}
				}
			}
			r, err := a.AnalyzeTransaction(input, MutationCountOptions{Facts: facts})
			if err != nil {
				t.Fatal(err)
			}
			if tc.disjoint {
				if r.Effective.Upper == nil || r.Effective.Lower != 0 || r.Bounds.BufferedCommit.Count.Upper == nil || !slices.Contains(r.Bounds.Reasons, "mixed_stage_disjoint_primary_keys_upper_bound") {
					t.Fatal(r)
				}
			} else if r.Effective.Upper != nil || r.Bounds.BufferedCommit.Count.Upper != nil || !slices.Contains(r.Bounds.Reasons, "mixed_stage_related_writes_not_calibrated") {
				t.Fatal(r)
			}
		})
	}
}

func TestMutationCountMixedDisjointKeyGuards(t *testing.T) {
	for _, ddl := range []string{
		mixedMutationDDL + `CREATE TABLE C(Id INT64 NOT NULL,FOREIGN KEY(Id) REFERENCES A(Id)) PRIMARY KEY(Id);`,
		mixedMutationDDL + `CREATE TABLE C(Id INT64 NOT NULL,FOREIGN KEY(Id) REFERENCES A(Id) NOT ENFORCED) PRIMARY KEY(Id);`,
		mixedMutationDDL + `CREATE TABLE C(Id INT64 NOT NULL) PRIMARY KEY(Id), INTERLEAVE IN PARENT A ON DELETE CASCADE;`,
		mixedMutationDDL + `ALTER TABLE A ADD COLUMN Extra STRING(MAX);`,
	} {
		a := mutationTestAnalyzer(t, ddl)
		r, err := a.AnalyzeTransaction(MutationCountTransaction{
			Statements: []string{"INSERT INTO A(Id,V) VALUES(1,'x')"},
			Mutations:  json.RawMessage(`[{"update":{"table":"A","columns":["Id","V"],"values":[["2","y"]]}}]`),
		}, MutationCountOptions{})
		if err != nil || r.Effective.Upper != nil || r.Bounds.BufferedCommit.Count.Upper != nil {
			t.Fatal(ddl, r, err)
		}
	}
	a := mutationTestAnalyzer(t, `CREATE TABLE A(Shard INT64 NOT NULL,Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Shard,Id);`)
	r, err := a.AnalyzeTransaction(MutationCountTransaction{
		Statements: []string{"UPDATE A SET V='a' WHERE Id=1"},
		Mutations:  json.RawMessage(`[{"update":{"table":"A","columns":["Shard","Id","V"],"values":[["2","3","b"]]}}]`),
	}, MutationCountOptions{Facts: []MutationCountRowFacts{{Rows: mutationTestInt(1), Old: map[string]interface{}{"Shard": "1"}}, {}}})
	if err != nil || r.Effective.Upper != nil {
		t.Fatal("partial PK or old-row data was treated as a complete predicate", r, err)
	}
}

func TestMutationCountMixedScalarKeys(t *testing.T) {
	for _, tc := range []struct {
		name, keyType, literal, wire string
		disjoint                     bool
	}{
		{"signed-int", "INT64", "-1", `"1"`, true},
		{"string", "STRING(MAX)", "'one'", `"two"`, true},
		{"same-string", "STRING(MAX)", "'one'", `"one"`, false},
		{"bool", "BOOL", "TRUE", "false", true},
		{"date", "DATE", "'2026-11-12'", `"2026-11-13"`, true},
		{"bytes-unresolved", "BYTES(MAX)", "b'one'", `"dHdv"`, false},
		{"timestamp-unresolved", "TIMESTAMP", "TIMESTAMP '2026-11-12T00:00:00Z'", `"2026-11-13T00:00:00Z"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := mutationTestAnalyzer(t, "CREATE TABLE A(Id "+tc.keyType+" NOT NULL,V STRING(MAX)) PRIMARY KEY(Id);")
			r, err := a.AnalyzeTransaction(MutationCountTransaction{
				Statements: []string{"UPDATE A SET V='new' WHERE Id=" + tc.literal},
				Mutations:  json.RawMessage(`[{"insert":{"table":"A","columns":["Id"],"values":[[` + tc.wire + `]]}}]`),
			}, MutationCountOptions{Facts: []MutationCountRowFacts{{Rows: mutationTestInt(1)}, {}}})
			if err != nil || (r.Effective.Upper != nil) != tc.disjoint {
				t.Fatal(r, err)
			}
		})
	}
}

func TestMutationCountMixedEveryStatementAndTable(t *testing.T) {
	a := mutationTestAnalyzer(t, mixedMutationDDL)
	input := MutationCountTransaction{
		Statements: []string{"UPDATE A SET V='x' WHERE Id=1", "DELETE FROM A WHERE Id=2"},
		Mutations:  json.RawMessage(`[{"update":{"table":"A","columns":["Id","V"],"values":[["2","y"]]}}]`),
	}
	options := MutationCountOptions{Facts: []MutationCountRowFacts{{Rows: mutationTestInt(1)}, {Rows: mutationTestInt(1)}, {}}}
	r, err := a.AnalyzeTransaction(input, options)
	if err != nil || r.Effective.Upper != nil {
		t.Fatal("a later statement overlaps", r, err)
	}
	input.Statements[1] = "UPDATE B SET V='x' WHERE TRUE"
	r, err = a.AnalyzeTransaction(input, options)
	if err != nil || r.Effective.Upper == nil || *r.Effective.Upper != 6 {
		t.Fatal("unrelated table should not require a key proof", r, err)
	}
	input.Mutations = json.RawMessage(`[{"update":{"table":"A","columns":["Id","V"],"values":[["2","y"]]}},{"update":{"table":"B","columns":["Id","V"],"values":[["2","y"]]}}]`)
	options.Facts = append(options.Facts, MutationCountRowFacts{})
	r, err = a.AnalyzeTransaction(input, options)
	if err != nil || r.Effective.Upper != nil {
		t.Fatal("every overlapping table needs a key proof", r, err)
	}
}

func TestMutationCountMixedDisjointKeysPreserveUnknownComponents(t *testing.T) {
	a := mutationTestAnalyzer(t, mixedMutationDDL)
	r, err := a.AnalyzeTransaction(MutationCountTransaction{
		Statements: []string{"INSERT INTO A(Id,V) VALUES(1,'x')"},
		Mutations:  json.RawMessage(`[{"delete":{"table":"A","keySet":{"keys":[["2"]]}}},{"update":{"table":"A","columns":["Id","V"],"values":[["3","y"]]}}]`),
	}, MutationCountOptions{})
	if err != nil || r.Effective.Upper != nil || r.Bounds.BufferedCommit.Count.Upper != nil || !slices.Contains(r.Bounds.Reasons, "mixed_delete_write_interaction_not_calibrated") {
		t.Fatal("disjoint stage keys must not erase a buffered-subset uncertainty", r, err)
	}
	a = mutationTestAnalyzer(t, `CREATE QUEUE Q(Id STRING(36) NOT NULL,Payload BYTES(MAX) NOT NULL) PRIMARY KEY(Id);`)
	r, err = a.AnalyzeTransaction(MutationCountTransaction{
		Statements: []string{"INSERT INTO Q(Id,Payload) VALUES('a',b'x')"},
		Mutations:  json.RawMessage(`[{"send":{"queue":"Q","key":["b"],"payload":"eA=="}}]`),
	}, MutationCountOptions{})
	if err != nil || r.Effective.Upper != nil || !slices.Contains(r.Bounds.Reasons, "mixed_stage_related_writes_not_calibrated") {
		t.Fatal("different message keys must not bypass queue accounting", r, err)
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
