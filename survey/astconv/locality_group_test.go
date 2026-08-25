package astconv

import (
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func TestRoundtripLocalityGroupPreservesOptionExpressions(t *testing.T) {
	stmt, err := memefish.ParseDDL("", "CREATE LOCALITY GROUP archive OPTIONS (storage = 'hdd', replica_count = 3, enabled = TRUE)")
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}
	if got := len(schema.LocalityGroups); got != 1 {
		t.Fatalf("locality groups = %d, want 1", got)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	group, ok := ddls[0].(*ast.CreateLocalityGroup)
	if !ok {
		t.Fatalf("DDL type = %T, want *ast.CreateLocalityGroup", ddls[0])
	}
	stringValue, ok := group.Options.Records[0].Value.(*ast.StringLiteral)
	if !ok || stringValue.Value != "hdd" {
		t.Errorf("string option = %#v, want StringLiteral(hdd)", group.Options.Records[0].Value)
	}
	if got := group.Options.Records[1].Value.SQL(); got != "3" {
		t.Errorf("integer option SQL = %q, want %q", got, "3")
	}
	if got := group.Options.Records[2].Value.SQL(); got != "TRUE" {
		t.Errorf("boolean option SQL = %q, want %q", got, "TRUE")
	}
}

func TestLocalityGroupsFromInformationSchemaOptions(t *testing.T) {
	schema := &Schema{
		LocalityGroupOptions: []*infoschem.LocalityGroupOption{
			{
				LocalityGroupName: "archive",
				OptionName:        "storage",
				OptionValue:       strPtr("'hdd'"),
			},
		},
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	if got := len(ddls); got != 1 {
		t.Fatalf("DDL statements = %d, want 1", got)
	}
	group := ddls[0].(*ast.CreateLocalityGroup)
	value, ok := group.Options.Records[0].Value.(*ast.StringLiteral)
	if group.Name.Name != "archive" || !ok || value.Value != "hdd" {
		t.Errorf("INFORMATION_SCHEMA-shaped reconstruction = %s", group.SQL())
	}
}

func TestLocalityGroupsFromInformationSchemaNullOptions(t *testing.T) {
	schema := &Schema{
		LocalityGroupOptions: []*infoschem.LocalityGroupOption{
			{
				LocalityGroupName: "archive",
				OptionName:        "storage",
				OptionValue:       strPtr("'ssd'"),
			},
			{
				LocalityGroupName: "archive",
				OptionName:        "ssd_to_hdd_spill_timespan",
			},
			{
				LocalityGroupName: "inherited",
				OptionName:        "storage",
			},
		},
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	if got := len(ddls); got != 2 {
		t.Fatalf("DDL statements = %d, want 2", got)
	}

	archive := ddls[0].(*ast.CreateLocalityGroup)
	if archive.Options == nil || len(archive.Options.Records) != 1 {
		t.Fatalf("archive options = %#v, want only non-NULL storage", archive.Options)
	}
	storage, ok := archive.Options.Records[0].Value.(*ast.StringLiteral)
	if !ok || storage.Value != "ssd" {
		t.Errorf("archive storage = %#v, want StringLiteral(ssd)", archive.Options.Records[0].Value)
	}

	inherited := ddls[1].(*ast.CreateLocalityGroup)
	if inherited.Options != nil {
		t.Errorf("all-NULL locality group options emitted as %s", inherited.Options.SQL())
	}
}

func TestLocalityGroupsRejectMalformedEmulatorOption(t *testing.T) {
	schema := &Schema{
		LocalityGroupOptions: []*infoschem.LocalityGroupOption{
			{
				LocalityGroupName: "archive",
				OptionName:        "inflash",
				OptionValue:       strPtr("BOOL"),
			},
		},
	}

	_, err := schema.ToDDLStatements()
	if err == nil || !strings.Contains(err.Error(), "malformed emulator metadata") {
		t.Fatalf("ToDDLStatements() error = %v, want malformed emulator metadata", err)
	}
}

func TestRoundtripOptionlessLocalityGroup(t *testing.T) {
	stmt, err := memefish.ParseDDL("", "CREATE LOCALITY GROUP archive")
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}
	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}
	if got := len(schema.LocalityGroupOptions); got != 0 {
		t.Fatalf("locality group options = %d, want 0", got)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	group := ddls[0].(*ast.CreateLocalityGroup)
	if group.Options != nil {
		t.Errorf("optionless locality group has options: %s", group.Options.SQL())
	}
}
