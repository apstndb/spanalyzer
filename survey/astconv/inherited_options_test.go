package astconv

import (
	"testing"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func TestBuildColumnDef_SkipsInheritedLocalityGroupKeepsAllowCommitTimestamp(t *testing.T) {
	schema := liveColumnOptionSchema(
		&infoschem.ColumnOption{
			TableName:   "Events",
			ColumnName:  "CreatedAt",
			OptionName:  "locality_group",
			OptionType:  "STRING",
			OptionValue: "'default'",
			IsInherited: new(true),
		},
		&infoschem.ColumnOption{
			TableName:   "Events",
			ColumnName:  "CreatedAt",
			OptionName:  "allow_commit_timestamp",
			OptionType:  "BOOL",
			OptionValue: "TRUE",
		},
	)

	col := reconstructSingleColumn(t, schema)
	if col.Options == nil || len(col.Options.Records) != 1 {
		t.Fatalf("column options = %#v, want only allow_commit_timestamp", col.Options)
	}
	if got := col.Options.Records[0].Name.Name; got != "allow_commit_timestamp" {
		t.Errorf("option name = %q, want allow_commit_timestamp", got)
	}
}

func TestBuildColumnDef_OmitsOptionsWhenEveryRowIsInherited(t *testing.T) {
	schema := liveColumnOptionSchema(
		&infoschem.ColumnOption{
			TableName:   "Events",
			ColumnName:  "CreatedAt",
			OptionName:  "locality_group",
			OptionType:  "STRING",
			OptionValue: "'default'",
			IsInherited: new(true),
		},
	)

	col := reconstructSingleColumn(t, schema)
	if col.Options != nil {
		t.Fatalf("column options = %#v, want no OPTIONS clause when every row is inherited", col.Options)
	}
}

func TestBuildColumnDef_EmitsExplicitLocalityGroup(t *testing.T) {
	schema := liveColumnOptionSchema(
		&infoschem.ColumnOption{
			TableName:   "Events",
			ColumnName:  "CreatedAt",
			OptionName:  "locality_group",
			OptionType:  "STRING",
			OptionValue: "'ssd_only'",
			IsInherited: new(false),
		},
	)

	col := reconstructSingleColumn(t, schema)
	if col.Options == nil || len(col.Options.Records) != 1 {
		t.Fatalf("column options = %#v, want explicit locality_group", col.Options)
	}
	if got := col.Options.Records[0].Name.Name; got != "locality_group" {
		t.Errorf("option name = %q, want locality_group", got)
	}
	value, ok := col.Options.Records[0].Value.(*ast.StringLiteral)
	if !ok || value.Value != "ssd_only" {
		t.Errorf("locality_group value = %#v, want StringLiteral(ssd_only)", col.Options.Records[0].Value)
	}
}

func TestBuildColumnDef_NilIsInheritedKeepsCurrentBehavior(t *testing.T) {
	schema := liveColumnOptionSchema(
		&infoschem.ColumnOption{
			TableName:   "Events",
			ColumnName:  "CreatedAt",
			OptionName:  "locality_group",
			OptionType:  "STRING",
			OptionValue: "'default'",
		},
		&infoschem.ColumnOption{
			TableName:   "Events",
			ColumnName:  "CreatedAt",
			OptionName:  "allow_commit_timestamp",
			OptionType:  "BOOL",
			OptionValue: "TRUE",
		},
	)

	col := reconstructSingleColumn(t, schema)
	if col.Options == nil || len(col.Options.Records) != 2 {
		t.Fatalf("column options = %#v, want both nil-IsInherited options", col.Options)
	}
	if got := col.Options.Records[0].Name.Name; got != "locality_group" {
		t.Errorf("first option = %q, want locality_group", got)
	}
	if got := col.Options.Records[1].Name.Name; got != "allow_commit_timestamp" {
		t.Errorf("second option = %q, want allow_commit_timestamp", got)
	}
}

func TestToLocalityGroupsDDL_SkipsInheritedStorageEmitsExplicit(t *testing.T) {
	schema := &Schema{
		LocalityGroupOptions: []*infoschem.LocalityGroupOption{
			{
				LocalityGroupName: "archive",
				OptionName:        "storage",
				OptionValue:       new("'hdd'"),
				IsInherited:       new(true),
			},
			{
				LocalityGroupName: "hot",
				OptionName:        "storage",
				OptionValue:       new("'ssd'"),
				IsInherited:       new(false),
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
	group, ok := ddls[0].(*ast.CreateLocalityGroup)
	if !ok {
		t.Fatalf("DDL type = %T, want *ast.CreateLocalityGroup", ddls[0])
	}
	if group.Name.Name != "hot" {
		t.Fatalf("emitted locality group = %q, want hot", group.Name.Name)
	}
	if group.Options == nil || len(group.Options.Records) != 1 {
		t.Fatalf("hot options = %#v, want explicit storage", group.Options)
	}
	value, ok := group.Options.Records[0].Value.(*ast.StringLiteral)
	if !ok || value.Value != "ssd" {
		t.Errorf("hot storage = %#v, want StringLiteral(ssd)", group.Options.Records[0].Value)
	}
}

func TestToLocalityGroupsDDL_NilIsInheritedKeepsCurrentBehavior(t *testing.T) {
	schema := &Schema{
		LocalityGroupOptions: []*infoschem.LocalityGroupOption{
			{
				LocalityGroupName: "archive",
				OptionName:        "storage",
				OptionValue:       new("'hdd'"),
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
	if group.Name.Name != "archive" {
		t.Fatalf("emitted locality group = %q, want archive", group.Name.Name)
	}
	if group.Options == nil || len(group.Options.Records) != 1 {
		t.Fatalf("archive options = %#v, want storage", group.Options)
	}
}

func liveColumnOptionSchema(opts ...*infoschem.ColumnOption) *Schema {
	return &Schema{
		Tables: []*infoschem.Table{
			{TableName: "Events", TableType: "BASE TABLE"},
		},
		Columns: []*infoschem.Column{
			{
				TableName:       "Events",
				ColumnName:      "CreatedAt",
				OrdinalPosition: 1,
				SpannerType:     "TIMESTAMP",
				IsNullable:      "YES",
			},
		},
		ColumnOptions: opts,
	}
}

func reconstructSingleColumn(t *testing.T, schema *Schema) *ast.ColumnDef {
	t.Helper()
	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	if got := len(ddls); got != 1 {
		t.Fatalf("DDL statements = %d, want 1", got)
	}
	ct, ok := ddls[0].(*ast.CreateTable)
	if !ok {
		t.Fatalf("DDL type = %T, want *ast.CreateTable", ddls[0])
	}
	if len(ct.Columns) != 1 {
		t.Fatalf("columns = %d, want 1", len(ct.Columns))
	}
	return ct.Columns[0]
}
