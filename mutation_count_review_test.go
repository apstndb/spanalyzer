package spanalyzer

import "testing"

func TestMutationCountRejectsContradictoryCardinality(t *testing.T) {
	a := mutationTestAnalyzer(t, `CREATE TABLE T(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id);`)
	for _, tc := range []struct {
		sql  string
		rows int64
	}{
		{"INSERT INTO T(Id,V) VALUES(1,'a')", 0},
		{"INSERT INTO T(Id,V) VALUES(1,'a')", 2},
		{"INSERT OR IGNORE INTO T(Id,V) VALUES(1,'a')", 2},
	} {
		if _, err := a.AnalyzeDML([]string{tc.sql}, MutationCountOptions{Facts: []MutationCountRowFacts{{Rows: &tc.rows}}}); err == nil {
			t.Fatal("cardinality outside normalized row bounds was accepted", tc.sql, tc.rows)
		}
	}
	for _, exists := range []bool{false, true} {
		rows := int64(1)
		if exists {
			rows = 0
		}
		if _, err := a.AnalyzeMutationsJSON([]byte(`[{"delete":{"table":"T","keySet":{"keys":[["1"]]}}}]`), MutationCountOptions{Facts: []MutationCountRowFacts{{Rows: &rows, Exists: &exists}}}); err == nil {
			t.Fatal("contradictory delete existence and cardinality were accepted")
		}
	}
	if _, err := a.AnalyzeDML([]string{"UPDATE T SET V='a' WHERE Id=1"}, MutationCountOptions{Facts: []MutationCountRowFacts{{Rows: mutationTestInt(1), Exists: mutationTestBool(false)}}}); err == nil {
		t.Fatal("contradictory DML update existence and cardinality were accepted")
	}
}

func TestMutationCountCascadeDeleteOverlap(t *testing.T) {
	a := mutationTestAnalyzer(t, `CREATE TABLE P(Id INT64 NOT NULL) PRIMARY KEY(Id);
CREATE TABLE C(Id INT64 NOT NULL,N INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id,N), INTERLEAVE IN PARENT P ON DELETE CASCADE;
CREATE INDEX ByV ON C(V);`)
	r, err := a.AnalyzeMutationsJSON([]byte(`[
{"delete":{"table":"P","keySet":{"keys":[["1"]]}}},
{"delete":{"table":"C","keySet":{"keys":[["1","1"]]}}}
]`), MutationCountOptions{Facts: []MutationCountRowFacts{
		{Exists: mutationTestBool(true), CascadeIndexRows: map[string]int64{"ByV": 1}},
		{Exists: mutationTestBool(true)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	// Pre-commit child facts do not prove that a later delete removes another
	// index entry after its parent already cascaded. Do not add exact old-row
	// effects across this unmodeled operation interaction.
	assertMutationInterval(t, r.Effective, 0, mutationTestInt(4))
}
