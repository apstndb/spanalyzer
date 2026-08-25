package infoschem

import "testing"

func TestTableMetaRegistryMutationIsolation(t *testing.T) {
	all := AllTableMetas()
	all[0].Schema = "mutated"
	all[0].Columns[0].Name = "MUTATED"

	byName, ok := TableMetaByName("CHANGE_STREAMS")
	if !ok {
		t.Fatal("CHANGE_STREAMS metadata not found")
	}
	if byName.Schema != "INFORMATION_SCHEMA" {
		t.Errorf("schema = %q, want INFORMATION_SCHEMA", byName.Schema)
	}
	if byName.Columns[0].Name != "CHANGE_STREAM_CATALOG" {
		t.Errorf("first column = %q, want CHANGE_STREAM_CATALOG", byName.Columns[0].Name)
	}

	byName.Columns[0].Name = "MUTATED_AGAIN"
	fresh := AllTableMetas()
	if fresh[0].Columns[0].Name != "CHANGE_STREAM_CATALOG" {
		t.Errorf("fresh registry column = %q, want CHANGE_STREAM_CATALOG", fresh[0].Columns[0].Name)
	}
}
