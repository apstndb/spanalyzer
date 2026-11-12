package spanalyzer

import "testing"

func TestIndexCatalogSharesObservedFactsWithoutExposingMutableState(t *testing.T) {
	a, err := NewMutationCountAnalyzer(`CREATE TABLE Owners(Id INT64 NOT NULL) PRIMARY KEY(Id); CREATE TABLE Events(Id INT64 NOT NULL,Owner INT64,CONSTRAINT EventOwner FOREIGN KEY(Owner) REFERENCES Owners(Id)) PRIMARY KEY(Id);`)
	if err != nil {
		t.Fatal(err)
	}
	a, err = a.WithObservedIndexes(a.SchemaSHA256(), []MutationCountIndexObservation{{Table: "Events", Name: "physical_generated_name", Type: "INDEX", State: "READ_WRITE", Columns: []string{"Owner"}, Managed: true, NullFiltered: true}})
	if err != nil {
		t.Fatal(err)
	}
	indexes := a.IndexCatalog()
	if len(indexes) != 1 || len(indexes[0].Constraints) != 1 || indexes[0].Constraints[0] != "EventOwner:referencing" || indexes[0].Origin != "observed_snapshot" {
		t.Fatalf("unexpected index facts: %+v", indexes)
	}
	indexes[0].Keys[0] = "wrong"
	indexes[0].Constraints[0] = "wrong"
	*indexes[0].ObservedUnique = true
	again := a.IndexCatalog()
	if again[0].Keys[0] != "Owner" || again[0].Constraints[0] != "EventOwner:referencing" || *again[0].ObservedUnique {
		t.Fatal("index facts alias caller state", again)
	}
}

func TestDeleteShapeAcceptsOnlyPrimaryKeyScenarioValues(t *testing.T) {
	a, err := NewMutationCountAnalyzer(`CREATE TABLE Events(Id INT64 NOT NULL,Note STRING(MAX)) PRIMARY KEY(Id);`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.AnalyzeWriteShape(MutationWriteShape{Operation: "delete", Table: "Events", Values: map[string]interface{}{"Id": "1"}}, MutationCountOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.AnalyzeWriteShape(MutationWriteShape{Operation: "delete", Table: "Events", Values: map[string]interface{}{"Note": "n"}}, MutationCountOptions{})
	if err == nil {
		t.Fatal("non-key delete value accepted")
	}
	_, err = a.AnalyzeWriteShape(MutationWriteShape{Operation: "delete", Table: "Events", Columns: []string{"Note"}, Values: map[string]interface{}{"Note": "n"}}, MutationCountOptions{})
	if err == nil {
		t.Fatal("written columns on delete accepted")
	}
}
