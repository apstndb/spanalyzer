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
