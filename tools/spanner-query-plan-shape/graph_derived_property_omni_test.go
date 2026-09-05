//go:build integration && omni

package main

import (
	"testing"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/api/iterator"
)

func TestIntegrationGraphDerivedPropertyResultTypesOnOmni(t *testing.T) {
	ddls := []string{
		`CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX),
) PRIMARY KEY (SingerId)`,
		`CREATE PROPERTY GRAPH MyGraph
  NODE TABLES (
    Singers
      LABEL Singer
      PROPERTIES (
        SingerId,
        CONCAT(FirstName, ' ', LastName) AS FullName,
        LENGTH(FirstName) AS FirstNameLength
      )
  )`,
	}
	clients := openOmniClients(t, ddls)
	if _, err := clients.Client.Apply(t.Context(), []*spanner.Mutation{
		spanner.Insert("Singers", []string{"SingerId", "FirstName", "LastName"}, []any{int64(1), "Ada", "Lovelace"}),
	}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	iter := clients.Client.Single().Query(t.Context(), spanner.Statement{
		SQL: "GRAPH MyGraph MATCH (n:Singer) RETURN n.FullName, n.FirstNameLength",
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		t.Fatal("Query returned no rows")
	}
	if err != nil {
		t.Fatalf("Query Next() error = %v", err)
	}
	if row == nil {
		t.Fatal("Query Next() row = nil")
	}
	if iter.Metadata == nil || iter.Metadata.RowType == nil {
		t.Fatal("RowIterator.Metadata.RowType is nil after Next")
	}
	fields := iter.Metadata.RowType.Fields
	if len(fields) != 2 {
		t.Fatalf("RowType fields = %d, want 2", len(fields))
	}
	if fields[0].GetType().GetCode() != spannerpb.TypeCode_STRING {
		t.Errorf("FullName type = %s, want STRING", fields[0].GetType().GetCode())
	}
	if fields[1].GetType().GetCode() != spannerpb.TypeCode_INT64 {
		t.Errorf("FirstNameLength type = %s, want INT64", fields[1].GetType().GetCode())
	}
}
