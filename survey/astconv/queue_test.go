package astconv

import (
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/survey/infoschem"
)

func TestToDDLStatementsRejectsQueue(t *testing.T) {
	t.Parallel()
	// A queue and its generated TVFs must not disappear from reconstruction
	// merely because they are neither BASE TABLE nor FUNCTION.
	schema := &Schema{
		Tables: []*infoschem.Table{{TableName: "Tasks", TableType: "QUEUE"}},
		Routines: []*infoschem.Routine{
			{RoutineName: "RECEIVE_Tasks", SpecificName: "RECEIVE_Tasks", RoutineType: "TABLE FUNCTION"},
			{RoutineName: "RENEWLEASE_Tasks", SpecificName: "RENEWLEASE_Tasks", RoutineType: "TABLE FUNCTION"},
		},
	}
	ddls, err := schema.ToDDLStatements()
	if err == nil || !strings.Contains(err.Error(), `unsupported queue "Tasks"`) ||
		!strings.Contains(err.Error(), "CREATE QUEUE") {
		t.Fatalf("ToDDLStatements() = %v, %v; want explicit queue support boundary", ddls, err)
	}
	if len(ddls) != 0 {
		t.Fatalf("ToDDLStatements() returned partial DDL: %v", ddls)
	}
}

func TestToDDLStatementsSkipsSystemQueueMetadata(t *testing.T) {
	t.Parallel()
	schema := &Schema{Tables: []*infoschem.Table{{
		TableSchema: "SPANNER_SYS", TableName: "InternalQueue", TableType: "QUEUE",
	}}}
	ddls, err := schema.ToDDLStatements()
	if err != nil || len(ddls) != 0 {
		t.Fatalf("system objects must stay excluded: %v, %v", ddls, err)
	}
}
