package spanalyzer_test

import (
	"fmt"

	spanalyzer "github.com/apstndb/spanalyzer"
)

func ExampleMutationCountAnalyzer_IndexCatalog() {
	ddl := `CREATE TABLE Owners(Id INT64 NOT NULL) PRIMARY KEY(Id);
CREATE TABLE Events(Id INT64 NOT NULL,Owner INT64,
  CONSTRAINT EventOwner FOREIGN KEY(Owner) REFERENCES Owners(Id)) PRIMARY KEY(Id);`
	analyzer, err := spanalyzer.NewMutationCountAnalyzer(ddl)
	if err != nil {
		panic(err)
	}
	// This synthetic snapshot illustrates the caller's completeness/freshness
	// assertion. Real generated names must come from the observed database.
	analyzer, err = analyzer.WithObservedIndexes(analyzer.SchemaSHA256(), []spanalyzer.MutationCountIndexObservation{{Table: "Events", Name: "observed_fk_index", Type: "INDEX", State: "READ_WRITE", Columns: []string{"Owner"}, Managed: true, NullFiltered: true}})
	if err != nil {
		panic(err)
	}
	index := analyzer.IndexCatalog()[0]
	fmt.Println(index.Table, index.Keys, index.Origin)
	fmt.Println(index.Constraints)
	// Output:
	// Events [Owner] observed_snapshot
	// [EventOwner:referencing]
}
