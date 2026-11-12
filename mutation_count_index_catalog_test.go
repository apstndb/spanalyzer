package spanalyzer

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestObservedIndexJSONPreservesNullFilteringAndLegacySpelling(t *testing.T) {
	a, err := NewMutationCountAnalyzer(`CREATE TABLE Owners(Id INT64 NOT NULL) PRIMARY KEY(Id); CREATE TABLE Events(Id INT64 NOT NULL,Owner INT64,CONSTRAINT EventOwner FOREIGN KEY(Owner) REFERENCES Owners(Id)) PRIMARY KEY(Id);`)
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{
		`{"table":"Events","name":"physical","type":"INDEX","state":"READ_WRITE","columns":["Owner"],"managed":true,"null_filtered":true}`,
		`{"Table":"Events","Name":"physical","Type":"INDEX","State":"READ_WRITE","Columns":["Owner"],"Managed":true,"NullFiltered":true}`,
	} {
		var observed MutationCountIndexObservation
		if err := json.Unmarshal([]byte(data), &observed); err != nil {
			t.Fatal(err)
		}
		model, err := a.WithObservedIndexes(a.SchemaSHA256(), []MutationCountIndexObservation{observed})
		if err != nil {
			t.Fatal(err)
		}
		r, err := model.AnalyzeWriteShape(MutationWriteShape{Operation: "insert", Table: "Events", Columns: []string{"Id"}, Values: map[string]interface{}{"Id": "1"}}, MutationCountOptions{})
		if err != nil || r.Effective.Lower != 1 || r.Effective.Upper == nil || *r.Effective.Upper != 1 {
			t.Fatalf("NULL_FILTERED membership changed after JSON decoding: %+v %v", r, err)
		}
		encoded, err := json.Marshal(observed)
		if err != nil || !strings.Contains(string(encoded), `"null_filtered":true`) {
			t.Fatalf("noncanonical snapshot encoding: %s %v", encoded, err)
		}
	}
	for _, data := range []string{`{"null_filtered":false,"NullFiltered":true}`, `{"null_filterd":true}`} {
		var observed MutationCountIndexObservation
		if err := json.Unmarshal([]byte(data), &observed); err == nil {
			t.Fatal("ambiguous or misspelled index attribute accepted", data)
		}
	}
}

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
