package spanalyzer_test

import (
	"fmt"

	spanalyzer "github.com/apstndb/spanalyzer"
)

func ExampleMutationCountAnalyzer_AnalyzeQueueRead() {
	analyzer, err := spanalyzer.NewMutationCountAnalyzer(`CREATE QUEUE Tasks(
  Id STRING(36) NOT NULL, Payload BYTES(MAX) NOT NULL) PRIMARY KEY(Id);`)
	if err != nil {
		panic(err)
	}
	report, err := analyzer.AnalyzeQueueRead("SELECT * FROM RECEIVE_Tasks(max_duration=>'1s')")
	if err != nil {
		panic(err)
	}
	fmt.Println(report.Scope)
	fmt.Println("upper bound known:", report.Effective.Upper != nil)
	// Output:
	// queue_internal_queries
	// upper bound known: false
}
