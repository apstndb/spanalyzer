package spanalyzer_test

import (
	"encoding/json"
	"fmt"

	"github.com/apstndb/spanalyzer"
)

func ExampleMutationCountAnalyzer_AnalyzeTransaction() {
	a, err := spanalyzer.NewMutationCountAnalyzer(`
CREATE TABLE Events(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id);
CREATE QUEUE Tasks(Id STRING(36) NOT NULL,Payload BYTES(MAX) NOT NULL) PRIMARY KEY(Id);`)
	if err != nil {
		panic(err)
	}
	r, err := a.AnalyzeTransaction(spanalyzer.MutationCountTransaction{
		Statements: []string{"INSERT INTO Events(Id,V) VALUES(1,'x')"},
		Mutations:  json.RawMessage(`[{"send":{"queue":"Tasks","key":["task-1"],"payload":"eA=="}}]`),
	}, spanalyzer.MutationCountOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Println(r.Scope)
	fmt.Println(len(r.Bounds.Statements), r.Bounds.BufferedCommit != nil)
	fmt.Println(r.Effective.Upper == nil)
	// Output:
	// mixed_dml_mutation_transaction
	// 1 true
	// true
}

func ExampleMutationCountAnalyzer_AnalyzeTransaction_disjointKeys() {
	a, err := spanalyzer.NewMutationCountAnalyzer(`
CREATE TABLE Events(Shard INT64 NOT NULL,Id INT64 NOT NULL,Note STRING(MAX)) PRIMARY KEY(Shard,Id);`)
	if err != nil {
		panic(err)
	}
	rows, exists := int64(1), true
	r, err := a.AnalyzeTransaction(spanalyzer.MutationCountTransaction{
		Statements: []string{"UPDATE Events SET Note='n' WHERE Shard=1 AND Id=2"},
		Mutations:  json.RawMessage(`[{"insert":{"table":"Events","columns":["Shard","Id"],"values":[["1","3"]]}}]`),
	}, spanalyzer.MutationCountOptions{Facts: []spanalyzer.MutationCountRowFacts{
		{Rows: &rows, Exists: &exists, Old: map[string]interface{}{"Note": "old"}}, {},
	}})
	if err != nil {
		panic(err)
	}
	fmt.Println(r.Effective.Lower, *r.Effective.Upper)
	fmt.Println(r.OldRows.Statements[0].Count.Lower, r.OldRows.BufferedCommit.Count.Lower)
	// Output:
	// 0 5
	// 3 2
}
