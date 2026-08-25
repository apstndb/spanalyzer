package astconv

import (
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/survey/infoschem"
)

func TestRoundtrip_PlacementKey(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"CREATE TABLE Singers (SingerId INT64 NOT NULL, Location STRING(MAX) NOT NULL PLACEMENT KEY HIDDEN) PRIMARY KEY (SingerId)",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.PlacementKeyColumns) != 1 {
		t.Fatalf("PlacementKeyColumns = %d, want 1", len(schema.PlacementKeyColumns))
	}
	placementKey := schema.PlacementKeyColumns[0]
	if placementKey.TableName != "Singers" || placementKey.ColumnName != "Location" {
		t.Fatalf("PlacementKeyColumns[0] = %+v, want Singers.Location", placementKey)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	if !strings.Contains(got, "Location STRING(MAX) NOT NULL PLACEMENT KEY HIDDEN") {
		t.Fatalf("generated DDL lost PLACEMENT KEY; got:\n%s", got)
	}

	reconstructed, err := FromDDLStatements(ddls)
	if err != nil {
		t.Fatalf("reconstruct generated DDL: %v", err)
	}
	if len(reconstructed.PlacementKeyColumns) != 1 {
		t.Fatalf("reconstructed PlacementKeyColumns = %d, want 1", len(reconstructed.PlacementKeyColumns))
	}
}

func TestToDDLStatements_PlacementKeyMissingColumnReturnsError(t *testing.T) {
	schema := &Schema{
		Tables: []*infoschem.Table{
			{TableName: "Missing", TableType: "BASE TABLE"},
		},
		Columns: []*infoschem.Column{
			{TableName: "Missing", ColumnName: "Other"},
		},
		PlacementKeyColumns: []*infoschem.PlacementKeyColumn{
			{TableName: "Missing", ColumnName: "Location"},
		},
	}

	_, err := schema.ToDDLStatements()
	if err == nil || !strings.Contains(err.Error(), "references missing column") {
		t.Fatalf("ToDDLStatements error = %v, want missing-column error", err)
	}
}

func TestRoundtrip_SequenceGrants(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"CREATE ROLE analyst",
		"GRANT SELECT, UPDATE ON SEQUENCE seq1, app.seq2 TO ROLE analyst",
		"GRANT SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA app TO ROLE analyst",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.SequenceGrants) != 4 {
		t.Fatalf("SequenceGrants = %d, want 4", len(schema.SequenceGrants))
	}
	if len(schema.AllSchemaGrants) != 2 {
		t.Fatalf("AllSchemaGrants = %d, want 2", len(schema.AllSchemaGrants))
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	got := ddlSQL(t, ddls)
	for _, want := range []string{
		"GRANT SELECT, UPDATE ON SEQUENCE seq1 TO ROLE analyst",
		"GRANT SELECT, UPDATE ON SEQUENCE app.seq2 TO ROLE analyst",
		"GRANT SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA app TO ROLE analyst",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated DDL missing %q; got:\n%s", want, got)
		}
	}

	reconstructed, err := FromDDLStatements(ddls)
	if err != nil {
		t.Fatalf("reconstruct generated DDL: %v", err)
	}
	if len(reconstructed.SequenceGrants) != 4 || len(reconstructed.AllSchemaGrants) != 2 {
		t.Fatalf("reconstructed grants = sequence %d, all-schema %d; want 4, 2",
			len(reconstructed.SequenceGrants), len(reconstructed.AllSchemaGrants))
	}
}

func TestFromDDLStatements_RevokeSequenceGrantsIsScoped(t *testing.T) {
	schema, err := FromDDLStatements(parseDDLs(t,
		"GRANT SELECT, UPDATE ON SEQUENCE seq1, app.seq2 TO ROLE analyst",
		"GRANT SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA app TO ROLE analyst",
		"REVOKE UPDATE ON SEQUENCE seq1 FROM ROLE analyst",
		"REVOKE SELECT ON SEQUENCE app.seq2 FROM ROLE analyst",
		"REVOKE UPDATE ON ALL SEQUENCES IN SCHEMA app FROM ROLE analyst",
	))
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	wantSequence := map[string]bool{
		".seq1:SELECT":    false,
		"app.seq2:UPDATE": false,
	}
	for _, grant := range schema.SequenceGrants {
		key := grant.SequenceSchema + "." + grant.SequenceName + ":" + grant.PrivilegeType
		if _, ok := wantSequence[key]; !ok {
			t.Errorf("unexpected sequence grant: %+v", grant)
			continue
		}
		wantSequence[key] = true
	}
	for key, found := range wantSequence {
		if !found {
			t.Errorf("missing sequence grant %s", key)
		}
	}

	if len(schema.AllSchemaGrants) != 1 ||
		schema.AllSchemaGrants[0].ObjectType != "SEQUENCES" ||
		schema.AllSchemaGrants[0].PrivilegeType != "SELECT" {
		t.Fatalf("remaining all-sequences grants = %+v, want app SELECT", schema.AllSchemaGrants)
	}
}

func TestToRolesDDL_InvalidSequenceGrantReturnsError(t *testing.T) {
	schema := &Schema{
		SequenceGrants: []*infoschem.SequenceGrant{
			{SequenceName: "seq1", PrivilegeType: "DELETE", Grantee: "analyst"},
		},
	}

	_, err := schema.ToDDLStatements()
	if err == nil || !strings.Contains(err.Error(), "invalid sequence privilege") {
		t.Fatalf("ToDDLStatements error = %v, want invalid sequence privilege", err)
	}
}
