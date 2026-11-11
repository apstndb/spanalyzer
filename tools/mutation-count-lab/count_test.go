package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
)

const simpleDDL = `CREATE TABLE Events (Shard INT64 NOT NULL, Id INT64 NOT NULL, Active STRING(MAX), Note STRING(MAX)) PRIMARY KEY(Shard,Id);
CREATE INDEX ByActive ON Events(Active) STORING(Note);`

func testSchema(t *testing.T, ddl string) *schemaModel {
	t.Helper()
	s, e := schemaFromDDL(ddl)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func assertCount(t *testing.T, q quantity, lo, hi int64) {
	t.Helper()
	if q.Lower != lo || q.Upper == nil || *q.Upper != hi {
		t.Fatalf("count=%+v upper=%v, want [%d,%d]", q, valueOr(q.Upper, -1), lo, hi)
	}
}
func updateOp(column string, v interface{}) writeOperation {
	return writeOperation{Kind: "update", Table: "Events", Columns: []string{"Shard", "Id", column}, Values: map[string]interface{}{"shard": "1", "id": "2", strings.ToLower(column): v}, Rows: fixed(1), Facts: rowFacts{Old: map[string]interface{}{"active": "a", "note": "n"}}, Channel: "mutation"}
}

func TestDataDependentIndexCosts(t *testing.T) {
	s := testSchema(t, simpleDDL)
	for _, tc := range []struct {
		name, column string
		value        interface{}
		cost         int64
	}{{"same key", "Active", "a", 3}, {"changed key", "Active", "b", 6}, {"same stored", "Note", "n", 5}, {"changed stored", "Note", "b", 5}} {
		t.Run(tc.name, func(t *testing.T) {
			op := updateOp(tc.column, tc.value)
			r, e := estimate(s, op, "row-image")
			if e != nil {
				t.Fatal(e)
			}
			assertCount(t, r.Count, tc.cost, tc.cost)
			if r.GateEligible {
				t.Fatal("calibration cannot authorize a quota gate")
			}
		})
	}
	// The same payload has different costs with different old values.
	op := updateOp("Active", "a")
	a, _ := estimate(s, op, "row-image")
	op.Facts.Old["active"] = "b"
	b, _ := estimate(s, op, "row-image")
	assertCount(t, a.Count, 3, 3)
	assertCount(t, b.Count, 6, 6)
	r, _ := estimate(s, op, "bounds")
	assertCount(t, r.Count, 3, 6)
}
func TestNullFilteredAndCompositeIndexes(t *testing.T) {
	s := testSchema(t, strings.Replace(simpleDDL, "CREATE INDEX", "CREATE NULL_FILTERED INDEX", 1))
	op := updateOp("Active", "a")
	op.Facts.Old["active"] = nil
	r, _ := estimate(s, op, "row-image")
	assertCount(t, r.Count, 5, 5)
	op = updateOp("Active", nil)
	r, _ = estimate(s, op, "row-image")
	assertCount(t, r.Count, 4, 4)
	s = testSchema(t, strings.Replace(simpleDDL, "(Active) STORING(Note)", "(Active,Note)", 1))
	op = writeOperation{Kind: "insert", Table: "Events", Columns: []string{"Shard", "Id", "Active", "Note"}, Values: map[string]interface{}{"shard": "1", "id": "2", "active": "a", "note": "n"}, Rows: fixed(1), Channel: "mutation"}
	r, _ = estimate(s, op, "row-image")
	assertCount(t, r.Count, 5, 5)
}
func TestDefaultsAndGeneratedAreChannelScoped(t *testing.T) {
	s := testSchema(t, `CREATE TABLE Events(Shard INT64 NOT NULL,Id INT64 NOT NULL,Note STRING(MAX) DEFAULT('n'), G INT64 AS(Id+1) STORED) PRIMARY KEY(Shard,Id);`)
	op := writeOperation{Kind: "insert", Table: "Events", Columns: []string{"Shard", "Id"}, Values: map[string]interface{}{"shard": "1", "id": "2"}, Rows: fixed(1), Channel: "mutation"}
	r, _ := estimate(s, op, "row-image")
	assertCount(t, r.Count, 6, 6)
	op.Channel = "dml"
	r, _ = estimate(s, op, "row-image")
	assertCount(t, r.Count, 4, 4)
	r, _ = estimate(s, op, "bounds")
	assertCount(t, r.Count, 2, 6)
	// Updating an input of a generated column cannot be priced by CREATE shape.
	s = testSchema(t, `CREATE TABLE Events(Shard INT64 NOT NULL,Id INT64 NOT NULL,Active INT64,G INT64 AS(Active+1) STORED) PRIMARY KEY(Shard,Id);`)
	op = updateOp("Active", "7")
	r, _ = estimate(s, op, "bounds")
	if r.Count.Upper != nil {
		t.Fatal("generated update dependency must stay unknown")
	}
}
func TestDeleteRangeAndCascade(t *testing.T) {
	s := testSchema(t, strings.Replace(simpleDDL, " STORING(Note)", "", 1)+"CREATE INDEX ByNote ON Events(Note);")
	op := writeOperation{Kind: "delete", Table: "Events", Rows: fixed(3), DeleteUnits: fixed(1), Channel: "mutation"}
	r, _ := estimate(s, op, "bounds")
	assertCount(t, r.Count, 7, 7)
	op.Rows = fixed(0)
	r, _ = estimate(s, op, "bounds")
	assertCount(t, r.Count, 1, 1)
	s = testSchema(t, `CREATE TABLE Parents(Id INT64 NOT NULL) PRIMARY KEY(Id);
CREATE TABLE Children(Id INT64 NOT NULL,C INT64 NOT NULL) PRIMARY KEY(Id,C), INTERLEAVE IN PARENT Parents ON DELETE CASCADE;
CREATE TABLE Grandchildren(Id INT64 NOT NULL,C INT64 NOT NULL,G INT64 NOT NULL,N STRING(MAX)) PRIMARY KEY(Id,C,G), INTERLEAVE IN PARENT Children ON DELETE CASCADE;
CREATE INDEX ByN ON Grandchildren(N);`)
	op = writeOperation{Kind: "delete", Table: "Parents", Rows: fixed(1), DeleteUnits: fixed(1), Channel: "mutation"}
	r, _ = estimate(s, op, "bounds")
	if r.Count.Upper != nil {
		t.Fatal("grandchild index fanout cannot be omitted")
	}
	op.Facts.CascadeIndexRows = map[string]int64{"ByN": 3}
	r, _ = estimate(s, op, "bounds")
	assertCount(t, r.Count, 4, 4)
}
func TestBufferedUpdatesCoalesceButDMLDoesNot(t *testing.T) {
	s := testSchema(t, strings.Replace(simpleDDL, " STORING(Note)", "", 1))
	a, b := updateOp("Active", "b"), updateOp("Active", "a")
	r, e := analyze(s, []writeOperation{a, b}, "row-image")
	if e != nil {
		t.Fatal(e)
	}
	assertCount(t, r.Count, 6, 6)
	a.Channel = "dml"
	b.Channel = "dml"
	b.Facts.Old["active"] = "b"
	r, e = analyze(s, []writeOperation{a, b}, "row-image")
	if e != nil {
		t.Fatal(e)
	}
	assertCount(t, r.Count, 10, 10)
	if r.LimitScope != "dml_statements_separately" {
		t.Fatal(r.LimitScope)
	}
}
func TestUncertaintyOverflowAndQuotaScope(t *testing.T) {
	q := multiply(fixed(math.MaxInt64), fixed(2))
	if !q.Overflow || q.Upper != nil {
		t.Fatal(q)
	}
	assertCount(t, multiply(uncertain(0, nil), fixed(0)), 0, 0)
	s := testSchema(t, `CREATE TABLE Events(Shard INT64 NOT NULL,Id INT64 NOT NULL,Active STRING(MAX),Note STRING(MAX)) PRIMARY KEY(Shard,Id);`)
	op := writeOperation{Kind: "insert", Table: "Events", Columns: []string{"Shard", "Id", "Active", "Note"}, Rows: fixed(20001), Channel: "mutation"}
	r, e := analyze(s, []writeOperation{op}, "bounds")
	if e != nil {
		t.Fatal(e)
	}
	assertCount(t, r.Count, 80004, 80004)
	if r.LimitVerdict != "model_exceeds_mutation_limit" {
		t.Fatal(r.LimitVerdict)
	}
	op.Channel = "dml"
	op.Rows = fixed(15000)
	second := op
	second.Statement = 1
	r, e = analyze(s, []writeOperation{op, second}, "bounds")
	if e != nil {
		t.Fatal(e)
	}
	assertCount(t, r.Count, 120000, 120000)
	if r.LimitVerdict != "within_each_model_statement_limit_only" {
		t.Fatal(r.LimitVerdict)
	}
	for _, step := range r.Results {
		if step.LimitVerdict != "within_model_mutation_limit_only" {
			t.Fatal(step.LimitVerdict)
		}
	}
	// Splitting one multi-row statement into normalized rows must not reset
	// the quota per row. Only actual DML statement boundaries reset it.
	r, e = analyze(s, []writeOperation{op, op}, "bounds")
	if e != nil {
		t.Fatal(e)
	}
	if r.LimitVerdict != "model_exceeds_dml_statement_limit" {
		t.Fatal(r.LimitVerdict)
	}
	ops, e := dmlOperations(s, "UPDATE Events SET Active='a' WHERE TRUE")
	if e != nil {
		t.Fatal(e)
	}
	r, e = analyze(s, ops, "bounds")
	if e != nil {
		t.Fatal(e)
	}
	if r.Count.Upper != nil {
		t.Fatal("WHERE must not imply one row")
	}
}

func TestManagedIndexesAreObservedAndShared(t *testing.T) {
	s := testSchema(t, `CREATE TABLE Owners(Id INT64 NOT NULL) PRIMARY KEY(Id);
CREATE TABLE Events(Id INT64 NOT NULL,Owner INT64,CONSTRAINT FK_A FOREIGN KEY(Owner) REFERENCES Owners(Id),CONSTRAINT FK_B FOREIGN KEY(Owner) REFERENCES Owners(Id)) PRIMARY KEY(Id);
CREATE TABLE Info(Id INT64 NOT NULL,Owner INT64,CONSTRAINT FK_Info FOREIGN KEY(Owner) REFERENCES Owners(Id) NOT ENFORCED) PRIMARY KEY(Id);`)
	if s.potentialManaged("Events") != 2 || s.potentialManaged("Info") != 0 {
		t.Fatal("DDL-only must express sharing uncertainty and informational FK behavior")
	}
	observed, e := s.withObservedIndexes([]observedIndex{{Table: "Events", Name: "backend_owned_name", Type: "INDEX", State: "READ_WRITE", Managed: true, NullFiltered: true, Columns: []string{"Owner"}}})
	if e != nil {
		t.Fatal(e)
	}
	i := observed.Indexes["events"]
	if len(i) != 1 || len(i[0].Constraints) != 2 {
		t.Fatalf("shared aliases lost: %+v", i)
	}
	if observed.potentialManaged("Events") != 0 {
		t.Fatal("do not count observed and hypothetical indexes twice")
	}
	s = testSchema(t, simpleDDL)
	if _, e := s.withObservedIndexes(nil); e == nil {
		t.Fatal("incomplete snapshot accepted")
	}
}

func TestWireMutationValidation(t *testing.T) {
	s := testSchema(t, `CREATE TABLE Events(Id INT64 NOT NULL,Required STRING(3) NOT NULL,Stamp TIMESTAMP,Allowed TIMESTAMP OPTIONS(allow_commit_timestamp=true),G INT64 AS(Id+1) STORED) PRIMARY KEY(Id);`)
	for _, tc := range []struct {
		name, input string
		valid       bool
	}{
		{"valid", `[{"insert":{"table":"Events","columns":["Id","Required"],"values":[["1","yes"]]}}]`, true},
		{"unknown table", `[{"insert":{"table":"Missing","columns":["Id"],"values":[["1"]]}}]`, false},
		{"unknown column", `[{"update":{"table":"Events","columns":["Id","Missing"],"values":[["1",true]]}}]`, false},
		{"arity", `[{"update":{"table":"Events","columns":["Id"],"values":[[]]}}]`, false},
		{"int wire encoding", `[{"update":{"table":"Events","columns":["Id"],"values":[[1]]}}]`, false},
		{"missing PK", `[{"update":{"table":"Events","columns":["Required"],"values":[["yes"]]}}]`, false},
		{"missing required insert", `[{"insert":{"table":"Events","columns":["Id"],"values":[["1"]]}}]`, false},
		{"missing required replace", `[{"replace":{"table":"Events","columns":["Id"],"values":[["1"]]}}]`, false},
		{"partial update", `[{"update":{"table":"Events","columns":["Id"],"values":[["1"]]}}]`, true},
		{"not null", `[{"update":{"table":"Events","columns":["Id","Required"],"values":[["1",null]]}}]`, false},
		{"too long", `[{"update":{"table":"Events","columns":["Id","Required"],"values":[["1","long"]]}}]`, false},
		{"no timestamp permission", `[{"update":{"table":"Events","columns":["Id","Stamp"],"values":[["1","spanner.commit_timestamp()"]]}}]`, false},
		{"timestamp allowed", `[{"update":{"table":"Events","columns":["Id","Allowed"],"values":[["1","spanner.commit_timestamp()"]]}}]`, true},
		{"generated", `[{"update":{"table":"Events","columns":["Id","G"],"values":[["1","2"]]}}]`, false},
		{"oneof", `[{"insert":{},"update":{}}]`, false},
		{"unknown field", `[{"update":{"table":"Events","columns":["Id"],"values":[["1"]]},"extra":1}]`, false},
		{"range boundaries", `[{"delete":{"table":"Events","keySet":{"ranges":[{}]}}}]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, e := decodeMutations(s, []byte(tc.input))
			if (e == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, e)
			}
		})
	}
}
func TestCommandAndMissingStats(t *testing.T) {
	dir := t.TempDir()
	ddl := dir + "/schema.sql"
	if e := os.WriteFile(ddl, []byte(simpleDDL), 0600); e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	if e := run([]string{"-ddl", ddl, "-dml", "UPDATE Events SET Active='a' WHERE TRUE"}, &out); e != nil {
		t.Fatal(e)
	}
	var a analysis
	if e := json.Unmarshal(out.Bytes(), &a); e != nil {
		t.Fatal(e)
	}
	if a.Count.Upper != nil {
		t.Fatal("unknown rows became scalar")
	}
	r := probeReceipt{Status: "complete", DDL: []string{simpleDDL}, Observations: []probeObservation{{ID: "test", Status: "OK", State: "absent", Statements: []string{"INSERT INTO Events(Shard,Id) VALUES(1,2)"}, Rows: []int64{1}}}}
	c, e := compareReceipt(r, "ddl")
	if e != nil {
		t.Fatal(e)
	}
	if c.Scores["bounds"].Unavailable != 1 || c.Scores["bounds"].Observed != 0 {
		t.Fatal("nil stats must not become zero")
	}
}

func TestRetainedRuntimeCalibration(t *testing.T) {
	for _, backend := range []string{"omni", "emulator"} {
		t.Run(backend, func(t *testing.T) {
			r, e := readReceipt("testdata/" + backend + ".json.gz")
			if e != nil {
				t.Fatal(e)
			}
			for _, provider := range []string{"ddl", "observed"} {
				c, e := compareReceipt(r, provider)
				if e != nil {
					t.Fatal(e)
				}
				if len(c.Cases) != 85 {
					t.Fatalf("lost cases: %d", len(c.Cases))
				}
				if backend == "omni" {
					if c.Scores["bounds"].Outside != 0 || c.Scores["row-image"].Outside != 0 {
						t.Fatalf("bounds/calibration disagreed: %+v %+v", c.Scores["bounds"], c.Scores["row-image"])
					}
					if provider == "observed" && c.Scores["row-image"].Exact != 84 {
						t.Fatalf("calibration did not account for every successful case: %+v", c.Scores["row-image"])
					}
					if c.Scores["shape"].Outside == 0 {
						t.Fatal("baseline must expose counterexamples")
					}
				} else {
					if c.Scores["bounds"].Unavailable != 84 {
						t.Fatal(c.Scores["bounds"])
					}
				}
			}
		})
	}
}
