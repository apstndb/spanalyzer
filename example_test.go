package spanalyzer_test

import (
	"fmt"

	"github.com/apstndb/spanalyzer"
)

func ExampleAnalyzer_RowTypeForStatement() {
	analyzer, err := spanalyzer.NewAnalyzerFromDDL("schema.sql", `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  Name STRING(MAX),
) PRIMARY KEY (SingerId);
`)
	if err != nil {
		panic(err)
	}

	rowType, err := analyzer.RowTypeForStatement("SELECT SingerId, Name FROM Singers")
	if err != nil {
		panic(err)
	}
	for _, field := range rowType.Fields {
		fmt.Printf("%s %s\n", field.Name, field.Type.Code)
	}

	// Output:
	// SingerId INT64
	// Name STRING
}
