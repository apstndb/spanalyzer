package astconv

import (
	"testing"

	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func TestRoundtripGenericOptions(t *testing.T) {
	inputs := []string{
		`ALTER DATABASE survey SET OPTIONS (columnar_policy = 'disabled', schema_drop_protection_inactivity_period = '7d', schema_drop_protection_usage_lowerbound = 100)`,
		`CREATE TABLE Dictionary (
  Key STRING(MAX) NOT NULL,
  Value ARRAY<STRING(MAX)> NOT NULL,
) PRIMARY KEY (Key), OPTIONS (fulltext_dictionary_table = true, fulltext_dictionary_staleness = '5s')`,
		`CREATE TABLE Documents (
  Id INT64 NOT NULL,
  Body STRING(MAX),
) PRIMARY KEY (Id), OPTIONS (columnar_policy = 'disabled')`,
		`CREATE INDEX DocumentsByBody ON Documents(Body) OPTIONS (columnar_policy = 'disabled')`,
	}

	statements := make([]ast.DDL, 0, len(inputs))
	for _, input := range inputs {
		statement, err := memefish.ParseDDL("", input)
		if err != nil {
			t.Fatalf("ParseDDL(%q): %v", input, err)
		}
		statements = append(statements, statement)
	}

	schema, err := FromDDLStatements(statements)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if got := len(schema.DatabaseOptions); got != 3 {
		t.Fatalf("DatabaseOptions count = %d, want 3", got)
	}
	if got := len(schema.TableOptions); got != 3 {
		t.Fatalf("TableOptions count = %d, want 3", got)
	}
	if got := len(schema.IndexOptions); got != 1 {
		t.Fatalf("IndexOptions count = %d, want 1", got)
	}

	databaseOptions := make(map[string]string, len(schema.DatabaseOptions))
	for _, option := range schema.DatabaseOptions {
		databaseOptions[option.OptionName] = option.OptionType + ":" + option.OptionValue
	}
	for name, want := range map[string]string{
		"columnar_policy":                          `STRING:"disabled"`,
		"schema_drop_protection_inactivity_period": `STRING:"7d"`,
		"schema_drop_protection_usage_lowerbound":  "INT64:100",
	} {
		if got := databaseOptions[name]; got != want {
			t.Errorf("database option %s = %q, want %q", name, got, want)
		}
	}

	tableOptions := make(map[string]string, len(schema.TableOptions))
	for _, option := range schema.TableOptions {
		tableOptions[option.OptionName] = option.OptionType + ":" + option.OptionValue
	}
	for name, want := range map[string]string{
		"fulltext_dictionary_table":     "BOOL:TRUE",
		"fulltext_dictionary_staleness": `STRING:"5s"`,
		"columnar_policy":               `STRING:"disabled"`,
	} {
		if got := tableOptions[name]; got != want {
			t.Errorf("table option %s = %q, want %q", name, got, want)
		}
	}
	if got := schema.IndexOptions[0].OptionType + ":" + schema.IndexOptions[0].OptionValue; got != `STRING:"disabled"` {
		t.Errorf("index columnar_policy = %q, want %q", got, `STRING:"disabled"`)
	}

	// Managed Spanner exposes STRING values in TABLE_OPTIONS without SQL
	// quotes. Reconstruct from that live shape, not only from AST-shaped values.
	for _, option := range schema.TableOptions {
		switch option.OptionName {
		case "fulltext_dictionary_staleness":
			option.OptionValue = "5s"
		case "columnar_policy":
			option.OptionValue = "disabled"
		}
	}

	reconstructed, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	want := map[string]string{
		"database":   `OPTIONS (columnar_policy = "disabled", schema_drop_protection_inactivity_period = "7d", schema_drop_protection_usage_lowerbound = 100)`,
		"Dictionary": `OPTIONS (fulltext_dictionary_table = true, fulltext_dictionary_staleness = "5s")`,
		"Documents":  `OPTIONS (columnar_policy = "disabled")`,
		"index":      `OPTIONS (columnar_policy = "disabled")`,
	}
	got := make(map[string]string)
	for _, statement := range reconstructed {
		switch statement := statement.(type) {
		case *ast.AlterDatabase:
			got["database"] = statement.Options.SQL()
		case *ast.CreateTable:
			got[leafName(statement.Name)] = statement.Options.SQL()
		case *ast.CreateIndex:
			got["index"] = statement.Options.SQL()
		}
	}
	for object, wantSQL := range want {
		if gotSQL := got[object]; gotSQL != wantSQL {
			t.Errorf("%s options = %q, want %q", object, gotSQL, wantSQL)
		}
	}
}
