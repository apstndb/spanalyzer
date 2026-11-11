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

func TestMutationCountRejectsContradictoryIndexRows(t *testing.T) {
	a := mutationTestAnalyzer(t, `CREATE TABLE T(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id);
CREATE INDEX ByV ON T(V);
CREATE NULL_FILTERED INDEX FilteredByV ON T(V);`)
	data := []byte(`[{"delete":{"table":"T","keySet":{"keys":[["1"]]}}}]`)
	for _, facts := range []MutationCountRowFacts{
		{Exists: mutationTestBool(false), IndexRows: map[string]int64{"ByV": 1}},
		{Rows: mutationTestInt(1), IndexRows: map[string]int64{"ByV": 0}},
		{Exists: mutationTestBool(true), Old: map[string]interface{}{"V": nil}, IndexRows: map[string]int64{"FilteredByV": 1}},
		{Exists: mutationTestBool(true), Old: map[string]interface{}{"V": "a"}, IndexRows: map[string]int64{"FilteredByV": 0}},
		{IndexRows: map[string]int64{"FilteredByV": 2}},
	} {
		if _, err := a.AnalyzeMutationsJSON(data, MutationCountOptions{Facts: []MutationCountRowFacts{facts}}); err == nil {
			t.Fatal("index rows contradicting cardinality or known membership accepted", facts)
		}
	}
	// A NULL-filtered index need not contain an existing row. Unknown values
	// must retain that conditional membership rather than imply one entry.
	for _, count := range []int64{0, 1} {
		if _, err := a.AnalyzeMutationsJSON(data, MutationCountOptions{Facts: []MutationCountRowFacts{{Exists: mutationTestBool(true), IndexRows: map[string]int64{"ByV": 1, "FilteredByV": count}}}}); err != nil {
			t.Fatal("valid conditional membership rejected", err)
		}
	}
	if _, err := a.AnalyzeMutationsJSON(data, MutationCountOptions{Facts: []MutationCountRowFacts{{Exists: mutationTestBool(true), Old: map[string]interface{}{"V": nil}, IndexRows: map[string]int64{"ByV": 1, "FilteredByV": 0}}}}); err != nil {
		t.Fatal(err)
	}
	// Ordinary indexes must agree on cardinality even without an explicit Rows
	// fact. A range's unknown cardinality cannot reconcile unequal assertions.
	a = mutationTestAnalyzer(t, `CREATE TABLE T(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id);
CREATE INDEX ByV ON T(V); CREATE INDEX ById ON T(Id);`)
	data = []byte(`[{"delete":{"table":"T","keySet":{"all":true}}}]`)
	if _, err := a.AnalyzeMutationsJSON(data, MutationCountOptions{Facts: []MutationCountRowFacts{{IndexRows: map[string]int64{"ByV": 1, "ById": 2}}}}); err == nil {
		t.Fatal("ordinary indexes cannot assert different deleted-row counts")
	}
	a = mutationTestAnalyzer(t, `CREATE TABLE P(Id INT64 NOT NULL) PRIMARY KEY(Id);
CREATE TABLE C(Id INT64 NOT NULL,N INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id,N), INTERLEAVE IN PARENT P ON DELETE CASCADE;
CREATE INDEX ByV ON C(V);`)
	data = []byte(`[{"delete":{"table":"P","keySet":{"keys":[["1"]]}}}]`)
	if _, err := a.AnalyzeMutationsJSON(data, MutationCountOptions{Facts: []MutationCountRowFacts{{Exists: mutationTestBool(false), CascadeIndexRows: map[string]int64{"ByV": 1}}}}); err == nil {
		t.Fatal("absent parent cannot assert cascading index removals")
	}
}

func TestMutationCountDeleteShapeBounds(t *testing.T) {
	a := mutationTestAnalyzer(t, `CREATE TABLE T(Id INT64 NOT NULL) PRIMARY KEY(Id);`)
	for _, bounds := range []MutationCountInterval{{Lower: 0}, {Lower: 0, Upper: mutationTestInt(2)}} {
		if _, err := a.AnalyzeWriteShape(MutationWriteShape{Operation: "delete", Table: "T", RowBounds: &bounds}, MutationCountOptions{}); err == nil {
			t.Fatal("one-key delete shape accepted cardinality above one", bounds)
		}
	}
	bounds := MutationCountInterval{Lower: 0, Upper: mutationTestInt(1)}
	if _, err := a.AnalyzeWriteShape(MutationWriteShape{Operation: "delete", Table: "T", RowBounds: &bounds}, MutationCountOptions{}); err != nil {
		t.Fatal(err)
	}
}
