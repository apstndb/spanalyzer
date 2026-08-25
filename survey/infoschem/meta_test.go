package infoschem

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAllTableMetas_Count(t *testing.T) {
	metas := AllTableMetas()
	if got := len(metas); got != 48 {
		t.Errorf("AllTableMetas() returned %d tables, want 48", got)
	}
}

func TestAllTableMetas_RegistryInvariants(t *testing.T) {
	tableNames := make(map[string]bool)
	for _, meta := range AllTableMetas() {
		if meta.Schema != "INFORMATION_SCHEMA" {
			t.Errorf("table %q schema = %q, want INFORMATION_SCHEMA", meta.Name, meta.Schema)
		}
		if tableNames[meta.Name] {
			t.Errorf("duplicate table metadata for %q", meta.Name)
		}
		tableNames[meta.Name] = true

		columnNames := make(map[string]bool)
		for _, column := range meta.Columns {
			if columnNames[column.Name] {
				t.Errorf("table %q has duplicate column metadata for %q", meta.Name, column.Name)
			}
			columnNames[column.Name] = true
		}
	}
}

func TestTableMetaByName(t *testing.T) {
	tests := []struct {
		name     string
		wantOK   bool
		wantCols int
	}{
		{"COLUMNS", true, 21},
		{"TABLES", true, 9},
		{"INDEXES", true, 14},
		{"VIEWS", true, 5},
		{"NONEXISTENT", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, ok := TableMetaByName(tt.name)
			if ok != tt.wantOK {
				t.Fatalf("TableMetaByName(%q) ok = %v, want %v", tt.name, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := len(meta.Columns); got != tt.wantCols {
				t.Errorf("table %s has %d columns, want %d", tt.name, got, tt.wantCols)
			}
		})
	}
}

func TestFilterQueryableRollingColumns(t *testing.T) {
	tests := []struct {
		name             string
		probeErr         error
		wantSearchUnnest bool
		wantErr          bool
	}{
		{name: "queryable", wantSearchUnnest: true},
		{name: "advertised before query support", probeErr: status.Error(codes.InvalidArgument, "unknown name")},
		{name: "backend reports unimplemented", probeErr: status.Error(codes.Unimplemented, "not implemented")},
		{name: "unrelated failure is preserved", probeErr: errors.New("transport failed"), wantSearchUnnest: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discovered := DiscoveredColumns{
				"INDEXES": {"SEARCH_UNNEST": true},
			}
			calls := 0
			err := filterQueryableRollingColumns(discovered, func(tableName, columnName string) error {
				calls++
				if tableName != "INDEXES" || columnName != "SEARCH_UNNEST" {
					t.Fatalf("probe(%q, %q), want INDEXES.SEARCH_UNNEST", tableName, columnName)
				}
				return tt.probeErr
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("filterQueryableRollingColumns() error = %v, wantErr %v", err, tt.wantErr)
			}
			if calls != 1 {
				t.Fatalf("probe calls = %d, want 1", calls)
			}
			if got := discovered["INDEXES"]["SEARCH_UNNEST"]; got != tt.wantSearchUnnest {
				t.Errorf("SEARCH_UNNEST present = %v, want %v", got, tt.wantSearchUnnest)
			}
		})
	}
}

func TestFilterQueryableRollingColumns_IgnoresOrdinaryColumns(t *testing.T) {
	discovered := DiscoveredColumns{
		"INDEXES": {"FILTER": true},
	}
	err := filterQueryableRollingColumns(discovered, func(_, _ string) error {
		t.Fatal("ordinary column was probed")
		return nil
	})
	if err != nil {
		t.Fatalf("filterQueryableRollingColumns: %v", err)
	}
}

func TestColumnNames(t *testing.T) {
	metadata := DiscoveredColumnMetadata{
		"INDEXES": {
			"INDEX_NAME":    {Name: "INDEX_NAME", SpannerType: "STRING(MAX)", OrdinalPosition: 4},
			"SEARCH_UNNEST": {Name: "SEARCH_UNNEST", SpannerType: "ARRAY<STRING(MAX)>", OrdinalPosition: 14},
		},
	}

	discovered := columnNames(metadata)
	if !discovered["INDEXES"]["INDEX_NAME"] || !discovered["INDEXES"]["SEARCH_UNNEST"] {
		t.Fatalf("columnNames() = %#v", discovered)
	}
	delete(discovered["INDEXES"], "INDEX_NAME")
	if _, ok := metadata["INDEXES"]["INDEX_NAME"]; !ok {
		t.Fatal("columnNames returned a map aliased to metadata")
	}
}

func TestTableMeta_Query(t *testing.T) {
	meta, ok := TableMetaByName("COLUMNS")
	if !ok {
		t.Fatal("COLUMNS not found")
	}

	discovered := DiscoveredColumns{
		"COLUMNS": {
			"TABLE_SCHEMA": true,
			"TABLE_NAME":   true,
			"COLUMN_NAME":  true,
			// Simulating missing ON_UPDATE_EXPRESSION
		},
	}

	q, err := meta.Query(discovered)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(q, "ON_UPDATE_EXPRESSION") {
		t.Error("Query should not include ON_UPDATE_EXPRESSION since it wasn't discovered")
	}
	if !strings.Contains(q, "`TABLE_SCHEMA`") {
		t.Error("Query should include TABLE_SCHEMA")
	}
	if !strings.HasPrefix(q, "SELECT ") {
		t.Error("Query should start with SELECT")
	}

	// Now simulate discovery of ON_UPDATE_EXPRESSION
	discovered["COLUMNS"]["ON_UPDATE_EXPRESSION"] = true
	q2, err := meta.Query(discovered)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q2, "ON_UPDATE_EXPRESSION") {
		t.Error("Query should include ON_UPDATE_EXPRESSION now")
	}
}

func TestTableMeta_Query_MissingTable(t *testing.T) {
	meta, ok := TableMetaByName("ROLES")
	if !ok {
		t.Fatal("ROLES not found")
	}

	discovered := DiscoveredColumns{} // Empty, ROLES is not available

	if _, err := meta.Query(discovered); err == nil {
		t.Error("Query should fail if table is not discovered")
	}
}
