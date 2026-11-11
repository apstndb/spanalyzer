package spanalyzer

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

const mutationTestDDL = `CREATE TABLE Events(Shard INT64 NOT NULL,Id INT64 NOT NULL,Active STRING(MAX),Note STRING(MAX),Extra STRING(MAX)) PRIMARY KEY(Shard,Id);
CREATE INDEX ByActive ON Events(Active) STORING(Note);`

func mutationTestAnalyzer(t *testing.T, ddl string) *MutationCountAnalyzer {
	t.Helper()
	a, err := NewMutationCountAnalyzer(ddl)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func assertMutationInterval(t *testing.T, q MutationCountInterval, lo int64, hi *int64) {
	t.Helper()
	if q.Lower != lo || (q.Upper == nil) != (hi == nil) || q.Upper != nil && *q.Upper != *hi {
		t.Fatalf("got [%d,%v], want [%d,%v]", q.Lower, q.Upper, lo, hi)
	}
}
func mutationTestInt(n int64) *int64 { return &n }
func mutationTestBool(b bool) *bool  { return &b }

func TestMutationCountPartialUpdates(t *testing.T) {
	a := mutationTestAnalyzer(t, mutationTestDDL)
	for _, tc := range []struct {
		column, value   string
		lo, hi, refined int64
	}{{"Active", "a", 3, 6, 3}, {"Active", "b", 3, 6, 6}, {"Note", "n", 5, 5, 5}, {"Extra", "x", 3, 3, 3}} {
		t.Run(tc.column+tc.value, func(t *testing.T) {
			payload := []byte(`[{"update":{"table":"Events","columns":["Shard","Id","` + tc.column + `"],"values":[["1","2","` + tc.value + `"]]}}]`)
			r, err := a.AnalyzeMutationsJSON(payload, MutationCountOptions{Facts: []MutationCountRowFacts{{Exists: mutationTestBool(true), Old: map[string]interface{}{"Active": "a", "Note": "n", "Extra": "x"}}}})
			if err != nil {
				t.Fatal(err)
			}
			assertMutationInterval(t, r.Bounds.Count, tc.lo, mutationTestInt(tc.hi))
			assertMutationInterval(t, r.Effective, tc.refined, mutationTestInt(tc.refined))
			if r.Bounds.GateEligible || r.CommitSizeLimitChecked || r.QuotaEnforcementVerified {
				t.Fatal("model cannot claim unmeasured enforcement or bytes")
			}
			if len(r.Bounds.Results[0].IndexEffects) != 1 {
				t.Fatal("index effect missing")
			}
		})
	}
}

func TestMutationCountAllWriteOperations(t *testing.T) {
	a := mutationTestAnalyzer(t, mutationTestDDL)
	for _, tc := range []struct {
		kind   string
		lo, hi int64
	}{{"insert", 6, 6}, {"update", 4, 7}, {"insertOrUpdate", 4, 7}, {"replace", 4, 7}} {
		payload := []byte(`[{"` + tc.kind + `":{"table":"Events","columns":["Shard","Id","Active","Note"],"values":[["1","2","new","new"]]}}]`)
		r, err := a.AnalyzeMutationsJSON(payload, MutationCountOptions{})
		if err != nil {
			t.Fatal(err)
		}
		// Four base cells plus a two-cell stored-index insert; existing rows
		// can also delete their prior index entry. UPDATE's lower bound can be
		// only its base cells when both index values already match.
		assertMutationInterval(t, r.Bounds.Count, tc.lo, mutationTestInt(tc.hi))
	}
	r, err := a.AnalyzeMutationsJSON([]byte(`[{"delete":{"table":"Events","keySet":{"ranges":[{"startClosed":["1","1"],"endClosed":["1","3"]}]}}}]`), MutationCountOptions{Facts: []MutationCountRowFacts{{Rows: mutationTestInt(3)}}})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Effective, 4, mutationTestInt(4))
}

func TestMutationCountUnknownShapeValuesAreNotNull(t *testing.T) {
	a := mutationTestAnalyzer(t, strings.Replace(mutationTestDDL, "CREATE INDEX", "CREATE NULL_FILTERED INDEX", 1))
	r, err := a.AnalyzeWriteShape(MutationWriteShape{Operation: "insert", Table: "Events", Columns: []string{"Shard", "Id", "Active", "Note"}}, MutationCountOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Bounds.Count, 4, mutationTestInt(6))
	r, err = a.AnalyzeWriteShape(MutationWriteShape{Operation: "insert", Table: "Events", Columns: []string{"Shard", "Id"}}, MutationCountOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Bounds.Count, 2, mutationTestInt(2))
}

func TestMutationCountObservationScopeAndDisagreement(t *testing.T) {
	a := mutationTestAnalyzer(t, mutationTestDDL)
	data := []byte(`[{"update":{"table":"Events","columns":["Shard","Id","Active"],"values":[["1","2","a"]]}}]`)
	base, err := a.AnalyzeMutationsJSON(data, MutationCountOptions{})
	if err != nil {
		t.Fatal(err)
	}
	o := &MutationCountObservation{Runtime: "managed", ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), SchemaSHA256: base.SchemaSHA256, InputSHA256: base.InputSHA256, Scope: base.Scope, Successful: true, StatsPresent: true, Count: mutationTestInt(3)}
	r, err := a.AnalyzeMutationsJSON(data, MutationCountOptions{Observation: o})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Effective, 3, mutationTestInt(3))
	assertMutationInterval(t, r.Bounds.Count, 3, mutationTestInt(6))
	o.InputSHA256 = "other-execution-shape"
	if _, err := a.AnalyzeMutationsJSON(data, MutationCountOptions{Observation: o}); err == nil {
		t.Fatal("foreign observation accepted")
	}
	o.InputSHA256 = base.InputSHA256
	o.Count = mutationTestInt(100)
	r, err = a.AnalyzeMutationsJSON(data, MutationCountOptions{Observation: o})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Disagreements) == 0 || r.EffectiveBasis == "same_execution_observation" {
		t.Fatal("disagreement hidden")
	}
	o.StatsPresent = false
	o.Count = nil
	r, err = a.AnalyzeMutationsJSON(data, MutationCountOptions{Observation: o})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Effective, 3, mutationTestInt(6))
}

func TestMutationCountDMLStatementBoundaries(t *testing.T) {
	a := mutationTestAnalyzer(t, `CREATE TABLE T(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id);`)
	r, err := a.AnalyzeDML([]string{"UPDATE T SET V='a' WHERE TRUE", "UPDATE T SET V='b' WHERE TRUE"}, MutationCountOptions{Facts: []MutationCountRowFacts{{Rows: mutationTestInt(30000)}, {Rows: mutationTestInt(30000)}}})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Bounds.Count, 120000, mutationTestInt(120000))
	if r.Bounds.LimitVerdict != "within_each_model_statement_limit_only" || len(r.Bounds.Statements) != 2 {
		t.Fatal(r.Bounds)
	}
	r, err = a.AnalyzeDML([]string{"UPDATE T SET V='a' WHERE TRUE"}, MutationCountOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Bounds.Count, 0, nil)
	if _, err := a.AnalyzeDML([]string{"INSERT OR UPDATE INTO T(Id,V) VALUES(1,'a')"}, MutationCountOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AnalyzeDML([]string{"INSERT OR REPLACE INTO T(Id,V) VALUES(1,'a')"}, MutationCountOptions{}); err == nil {
		t.Fatal("Spanner DML has no INSERT OR REPLACE")
	}
}

func TestMutationCountQueues(t *testing.T) {
	ddl := "-- CREATE QUEUE Fake\nCREATE QUEUE Tasks(Id STRING(36) NOT NULL,Payload BYTES(MAX) NOT NULL) PRIMARY KEY(Id); CREATE TABLE Notes(Id INT64 NOT NULL,V STRING(MAX) DEFAULT('CREATE QUEUE')) PRIMARY KEY(Id);"
	a := mutationTestAnalyzer(t, ddl)
	for _, data := range []string{`[{"send":{"queue":"Tasks","key":["a"],"payload":"eA=="}}]`, `[{"ack":{"queue":"Tasks","key":["a"]}}]`} {
		r, err := a.AnalyzeMutationsJSON([]byte(data), MutationCountOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if r.Bounds.Count.Upper != nil {
			t.Fatal("queue internal effects silently treated as ordinary table")
		}
	}
	for _, data := range []string{
		`[{"insert":{"table":"Tasks","columns":["Id","Payload"],"values":[["a","eA=="]]}}]`,
		`[{"update":{"table":"Tasks","columns":["Id","Payload"],"values":[["a","eA=="]]}}]`,
		`[{"insertOrUpdate":{"table":"Tasks","columns":["Id","Payload"],"values":[["a","eA=="]]}}]`,
		`[{"replace":{"table":"Tasks","columns":["Id","Payload"],"values":[["a","eA=="]]}}]`,
		`[{"delete":{"table":"Tasks","keySet":{"keys":[["a"]]}}}]`,
	} {
		if _, err := a.AnalyzeMutationsJSON([]byte(data), MutationCountOptions{}); err == nil {
			t.Fatal("ordinary wire queue write accepted:", data)
		}
	}
	for _, sql := range []string{"UPDATE Tasks SET Payload=b'x' WHERE Id='a'", "INSERT OR UPDATE INTO Tasks(Id,Payload) VALUES('a',b'x')"} {
		if _, err := a.AnalyzeDML([]string{sql}, MutationCountOptions{}); err == nil {
			t.Fatal("unsupported queue DML accepted:", sql)
		}
	}
	for _, sql := range []string{"INSERT INTO Tasks(Id,Payload) VALUES('a',b'x')", "DELETE FROM Tasks WHERE Id='a'"} {
		r, err := a.AnalyzeDML([]string{sql}, MutationCountOptions{})
		if err != nil || r.Bounds.Count.Upper != nil {
			t.Fatal("supported queue DML needs unknown accounting:", r, err)
		}
	}
	for _, sql := range []string{"SELECT * FROM RECEIVE_Tasks(max_duration=>'1s')", "SELECT * FROM RENEWLEASE_Tasks(lease_tokens=>['test'])"} {
		r, err := a.AnalyzeQueueRead(sql)
		if err != nil {
			t.Fatal(err)
		}
		if r.Effective.Upper != nil || r.Scope == "mutation_api_commit" {
			t.Fatal(r)
		}
	}
	if _, err := a.AnalyzeQueueRead("SELECT * FROM RECEIVE_Unknown()"); err == nil {
		t.Fatal("unknown queue accepted")
	}
	if _, err := a.AnalyzeMutationsJSON([]byte(`[{"send":{"queue":"Notes","key":["1"],"payload":"eA=="}}]`), MutationCountOptions{}); err == nil {
		t.Fatal("ordinary table accepted as queue")
	}
	if _, err := NewMutationCountAnalyzer("CREATE QUEUE Bad(Id INT64 NOT NULL,Wrong BYTES(MAX)) PRIMARY KEY(Id)"); err == nil {
		t.Fatal("missing Payload accepted")
	}
}

func TestMutationCountQueueOpaqueProtoPayload(t *testing.T) {
	a := mutationTestAnalyzer(t, "CREATE PROTO BUNDLE (`examples.shipping.Order`); CREATE QUEUE Tasks(Id STRING(36) NOT NULL,Payload `examples.shipping.Order` NOT NULL) PRIMARY KEY(Id)")
	for _, value := range []string{"", "CgF4"} {
		r, err := a.AnalyzeMutationsJSON([]byte(`[{"send":{"queue":"Tasks","key":["a"],"payload":"`+value+`"}}]`), MutationCountOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if r.Effective.Upper != nil {
			t.Fatal("opaque proto is not a queue-accounting formula")
		}
	}
	if _, err := a.AnalyzeMutationsJSON([]byte(`[{"send":{"queue":"Tasks","key":["a"],"payload":"invalid-base64"}}]`), MutationCountOptions{}); err == nil {
		t.Fatal("invalid proto wire envelope accepted")
	}
}

func TestMutationCountDMLIgnoreAndSourceCardinality(t *testing.T) {
	a := mutationTestAnalyzer(t, "CREATE TABLE T(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id)")
	sql := []string{"INSERT OR IGNORE INTO T(Id,V) VALUES(1,'x')"}
	r, err := a.AnalyzeDML(sql, MutationCountOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Bounds.Count, 0, mutationTestInt(2))
	for _, exists := range []bool{false, true} {
		r, err := a.AnalyzeDML(sql, MutationCountOptions{Facts: []MutationCountRowFacts{{Exists: &exists}}})
		if err != nil {
			t.Fatal(err)
		}
		want := int64(2)
		if exists {
			want = 0
		}
		assertMutationInterval(t, r.Effective, want, &want)
	}
	selectSQL := []string{"INSERT INTO T(Id,V) SELECT Id,'x' FROM UNNEST([1,2,3]) AS Id"}
	r, err = a.AnalyzeDML(selectSQL, MutationCountOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Effective, 0, nil)
	r, err = a.AnalyzeDML(selectSQL, MutationCountOptions{Facts: []MutationCountRowFacts{{Rows: mutationTestInt(3)}}})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Effective, 6, mutationTestInt(6))
	if _, err := a.AnalyzeDML(selectSQL, MutationCountOptions{Facts: []MutationCountRowFacts{{Exists: mutationTestBool(true)}}}); err == nil {
		t.Fatal("one old image cannot describe an INSERT SELECT group")
	}
}

func TestMutationCountStaticValidation(t *testing.T) {
	a := mutationTestAnalyzer(t, `CREATE TABLE T(Id INT64 NOT NULL,Required STRING(2) NOT NULL,Stamp TIMESTAMP,G INT64 AS(Id+1) STORED) PRIMARY KEY(Id);`)
	for _, data := range []string{
		`[{"insertOrUpdate":{"table":"T","columns":["Id"],"values":[["1"]]}}]`,
		`[{"update":{"table":"T","columns":["Id","G"],"values":[["1","2"]]}}]`,
		`[{"update":{"table":"T","columns":["Id","Stamp"],"values":[["1","spanner.commit_timestamp()"]]}}]`,
		`[{"update":{"table":"T","columns":["Id"],"values":[[1]]}}]`,
	} {
		if _, err := a.AnalyzeMutationsJSON([]byte(data), MutationCountOptions{}); err == nil {
			t.Fatal("invalid payload accepted:", data)
		}
	}
}

func ExampleMutationCountAnalyzer_AnalyzeMutationsJSON() {
	a, err := NewMutationCountAnalyzer("CREATE TABLE Events(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id)")
	if err != nil {
		panic(err)
	}
	r, err := a.AnalyzeMutationsJSON([]byte(`[{"insert":{"table":"Events","columns":["Id","V"],"values":[["1","x"]]}}]`), MutationCountOptions{})
	if err != nil {
		panic(err)
	}
	b, err := json.Marshal(r.Effective)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	// Output:
	// {"lower":2,"upper":2}
}

func ExampleMutationCountAnalyzer_AnalyzeMutationsJSON_indexEffects() {
	a, err := NewMutationCountAnalyzer(`CREATE TABLE T(Id INT64 NOT NULL,K STRING(MAX),S STRING(MAX),Other STRING(MAX)) PRIMARY KEY(Id);
CREATE INDEX ByK ON T(K) STORING(S);`)
	if err != nil {
		panic(err)
	}
	exists := true
	for _, column := range []string{"Other", "S", "K"} {
		data := []byte(`[{"update":{"table":"T","columns":["Id","` + column + `"],"values":[["1","new"]]}}]`)
		r, err := a.AnalyzeMutationsJSON(data, MutationCountOptions{Facts: []MutationCountRowFacts{{Exists: &exists, Old: map[string]interface{}{"K": "old", "S": "old"}}}})
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s: [%d,%d]\n", column, r.Effective.Lower, *r.Effective.Upper)
	}
	// Output:
	// Other: [2,2]
	// S: [4,4]
	// K: [5,5]
}
