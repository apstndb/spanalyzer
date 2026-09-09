package astconv

import (
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func TestToDatabaseDDL_LiveShapedFiltersNonCanonicalOptions(t *testing.T) {
	schema := &Schema{
		DatabaseName: "survey_db",
		DatabaseOptions: []*infoschem.DatabaseOption{
			{OptionName: "enable_key_visualizer", OptionType: "BOOL", OptionValue: "TRUE"},
			{OptionName: "database_dialect", OptionType: "STRING", OptionValue: "GOOGLE_STANDARD_SQL"},
			{OptionName: "default_sequence_kind", OptionType: "STRING", OptionValue: "BIT_REVERSED_POSITIVE"},
		},
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	if got := len(ddls); got != 1 {
		t.Fatalf("DDL statements = %d, want 1", got)
	}
	ad, ok := ddls[0].(*ast.AlterDatabase)
	if !ok {
		t.Fatalf("DDL type = %T, want *ast.AlterDatabase", ddls[0])
	}
	if ad.Name.Name != "survey_db" {
		t.Errorf("database name = %q, want survey_db", ad.Name.Name)
	}
	if ad.Options == nil || len(ad.Options.Records) != 1 {
		t.Fatalf("database options = %#v, want only default_sequence_kind", ad.Options)
	}
	if ad.Options.Records[0].Name.Name != "default_sequence_kind" {
		t.Errorf("option name = %q, want default_sequence_kind", ad.Options.Records[0].Name.Name)
	}
}

func TestToDatabaseDDL_MissingNameWithSettableOptionReturnsError(t *testing.T) {
	schema := &Schema{
		DatabaseOptions: []*infoschem.DatabaseOption{
			{OptionName: "optimizer_version", OptionType: "INT64", OptionValue: "4"},
		},
	}

	_, err := schema.ToDDLStatements()
	if err == nil {
		t.Fatal("ToDDLStatements() error = nil, want error when settable options exist without a database name")
	}
	if !strings.Contains(err.Error(), "database name") {
		t.Errorf("ToDDLStatements() error = %v, want database name", err)
	}
}

func TestToDatabaseDDL_OnlyNonCanonicalOptionsEmitsNothing(t *testing.T) {
	schema := &Schema{
		DatabaseOptions: []*infoschem.DatabaseOption{
			{OptionName: "enable_key_visualizer", OptionType: "BOOL", OptionValue: "TRUE"},
			{OptionName: "database_dialect", OptionType: "STRING", OptionValue: "GOOGLE_STANDARD_SQL"},
		},
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	if got := len(ddls); got != 0 {
		t.Fatalf("DDL statements = %d, want 0", got)
	}
}

func TestToDatabaseDDL_GenericNonDenylistedOptionEmitted(t *testing.T) {
	schema := &Schema{
		DatabaseName: "survey_db",
		DatabaseOptions: []*infoschem.DatabaseOption{
			{OptionName: "schema_drop_protection_inactivity_period", OptionType: "STRING", OptionValue: "7d"},
		},
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	if got := len(ddls); got != 1 {
		t.Fatalf("DDL statements = %d, want 1", got)
	}
	ad, ok := ddls[0].(*ast.AlterDatabase)
	if !ok {
		t.Fatalf("DDL type = %T, want *ast.AlterDatabase", ddls[0])
	}
	if ad.Options == nil || len(ad.Options.Records) != 1 {
		t.Fatalf("database options = %#v, want schema_drop_protection_inactivity_period", ad.Options)
	}
	if ad.Options.Records[0].Name.Name != "schema_drop_protection_inactivity_period" {
		t.Errorf("option name = %q, want schema_drop_protection_inactivity_period", ad.Options.Records[0].Name.Name)
	}
}
