package spanalyzer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestInformationSchemaManifestProjection(t *testing.T) {
	manifest, definitions, err := parseInformationSchemaManifest(embeddedInformationSchemaManifest)
	if err != nil {
		t.Fatalf("parseInformationSchemaManifest() error = %v", err)
	}
	if got, want := len(manifest.Tables), 48; got != want {
		t.Fatalf("manifest table count = %d, want %d", got, want)
	}
	if got, want := len(definitions), len(manifest.Tables); got != want {
		t.Fatalf("definition table count = %d, want %d", got, want)
	}

	projectedColumns := 0
	rollingColumns := 0
	docsOnlyColumns := 0
	rollingPaths := make(map[string]bool)
	docsOnlyPaths := make(map[string]bool)
	for tableIndex, manifestTable := range manifest.Tables {
		definition := definitions[tableIndex]
		if definition.name != manifestTable.Name {
			t.Fatalf("definition[%d].name = %q, want %q", tableIndex, definition.name, manifestTable.Name)
		}
		definitionColumn := 0
		for _, manifestColumn := range manifestTable.Columns {
			switch manifestColumn.EvidenceStatus {
			case "rolling":
				rollingColumns++
				rollingPaths[manifestTable.Name+"."+manifestColumn.Name] = true
			case "docs_only_absent":
				docsOnlyColumns++
				docsOnlyPaths[manifestTable.Name+"."+manifestColumn.Name] = true
			}
			if !manifestColumn.Project {
				continue
			}
			projectedColumns++
			if definitionColumn >= len(definition.columns) {
				t.Fatalf("definition %q lacks projected column %q", definition.name, manifestColumn.Name)
			}
			got := definition.columns[definitionColumn]
			if got.name != manifestColumn.Name {
				t.Fatalf("definition %q column %d = %q, want %q", definition.name, definitionColumn, got.name, manifestColumn.Name)
			}
			wantType, err := informationSchemaColumnType(manifestColumn)
			if err != nil {
				t.Fatalf("informationSchemaColumnType(%s.%s) error = %v", definition.name, manifestColumn.Name, err)
			}
			if !reflect.DeepEqual(got.typ, wantType) {
				t.Fatalf("definition %s.%s type = %#v, want %#v", definition.name, got.name, got.typ, wantType)
			}
			definitionColumn++
		}
		if definitionColumn != len(definition.columns) {
			t.Fatalf("definition %q has %d columns, manifest projects %d", definition.name, len(definition.columns), definitionColumn)
		}
	}
	if got, want := projectedColumns, 308; got != want {
		t.Errorf("projected column count = %d, want %d", got, want)
	}
	if got, want := rollingColumns, 2; got != want {
		t.Errorf("rolling column count = %d, want %d", got, want)
	}
	if got, want := docsOnlyColumns, 9; got != want {
		t.Errorf("docs-only column count = %d, want %d", got, want)
	}
	wantRolling := map[string]bool{
		"INDEXES.SEARCH_UNNEST":    true,
		"INDEX_COLUMNS.EXPRESSION": true,
	}
	if !reflect.DeepEqual(rollingPaths, wantRolling) {
		t.Errorf("rolling columns = %#v, want %#v", rollingPaths, wantRolling)
	}
	wantDocsOnly := map[string]bool{
		"ROLE_TABLE_GRANTS.GRANTOR":         true,
		"ROLE_TABLE_GRANTS.IS_GRANTABLE":    true,
		"ROLE_COLUMN_GRANTS.GRANTOR":        true,
		"ROLE_COLUMN_GRANTS.IS_GRANTABLE":   true,
		"ROLE_MODEL_GRANTS.GRANTOR":         true,
		"ROLE_MODEL_GRANTS.IS_GRANTABLE":    true,
		"ROLE_ROUTINE_GRANTS.GRANTOR":       true,
		"ROLE_ROUTINE_GRANTS.IS_GRANTABLE":  true,
		"TABLE_SYNONYMS.SYNONYM_TABLE_NAME": true,
	}
	if !reflect.DeepEqual(docsOnlyPaths, wantDocsOnly) {
		t.Errorf("docs-only columns = %#v, want %#v", docsOnlyPaths, wantDocsOnly)
	}
}

func TestInformationSchemaManifestRejectsStaleHash(t *testing.T) {
	var manifest informationSchemaManifest
	if err := json.Unmarshal(embeddedInformationSchemaManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Tables[0].Columns[0].Name += "_CHANGED"
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = parseInformationSchemaManifest(data)
	if err == nil || !strings.Contains(err.Error(), "content_sha256") {
		t.Fatalf("parseInformationSchemaManifest() error = %v, want stale content_sha256", err)
	}
}

func TestInformationSchemaManifestRejectsDuplicateKeys(t *testing.T) {
	data := bytes.Replace(
		embeddedInformationSchemaManifest,
		[]byte(`"schema_version": "v0alpha2",`),
		[]byte(`"schema_version": "v0alpha2", "schema_version": "v0alpha2",`),
		1,
	)
	_, _, err := parseInformationSchemaManifest(data)
	if err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("parseInformationSchemaManifest() error = %v, want duplicate key", err)
	}
}

func TestInformationSchemaManifestRejectsNoncanonicalObservationPath(t *testing.T) {
	manifest := decodedInformationSchemaManifest(t)
	manifest.Source.SelectedObservationPath = "survey/infoschem/evidence/managed/latest.json"
	data := encodedInformationSchemaManifest(t, manifest)
	_, _, err := parseInformationSchemaManifest(data)
	if err == nil || !strings.Contains(err.Error(), "selected observation path") {
		t.Fatalf("parseInformationSchemaManifest() error = %v, want observation path failure", err)
	}
}

func TestInformationSchemaManifestRequiresProtoProjectionOverride(t *testing.T) {
	manifest := decodedInformationSchemaManifest(t)
	var found bool
	for tableIndex := range manifest.Tables {
		for columnIndex := range manifest.Tables[tableIndex].Columns {
			column := &manifest.Tables[tableIndex].Columns[columnIndex]
			if strings.HasPrefix(column.RawType, "PROTO<") {
				column.ProjectedType = ""
				found = true
			}
		}
	}
	if !found {
		t.Fatal("manifest contains no PROTO raw type")
	}
	data := encodedInformationSchemaManifest(t, manifest)
	_, _, err := parseInformationSchemaManifest(data)
	if err == nil || !strings.Contains(err.Error(), "requires an explicit projected_type") {
		t.Fatalf("parseInformationSchemaManifest() error = %v, want missing projected_type", err)
	}
}

func TestAnalyzerProjectsEveryInformationSchemaTableInManifestOrder(t *testing.T) {
	manifest, _, err := parseInformationSchemaManifest(embeddedInformationSchemaManifest)
	if err != nil {
		t.Fatal(err)
	}
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	for _, table := range manifest.Tables {
		t.Run(table.Name, func(t *testing.T) {
			rowType, err := analyzer.RowTypeForStatement(fmt.Sprintf("SELECT * FROM INFORMATION_SCHEMA.`%s`", table.Name))
			if err != nil {
				t.Fatalf("RowTypeForStatement() error = %v", err)
			}
			projected := make([]informationSchemaManifestColumn, 0, len(table.Columns))
			for _, column := range table.Columns {
				if column.Project {
					projected = append(projected, column)
				}
			}
			if got, want := len(rowType.Fields), len(projected); got != want {
				t.Fatalf("field count = %d, want %d", got, want)
			}
			for index, column := range projected {
				wantType, err := informationSchemaColumnType(column)
				if err != nil {
					t.Fatal(err)
				}
				assertField(t, rowType.Fields[index], column.Name, wantType.Code)
				if wantType.Code == spannerpb.TypeCode_ARRAY {
					if rowType.Fields[index].Type.ArrayElementType == nil {
						t.Fatalf("field %s has nil array element type", column.Name)
					}
					if got, want := rowType.Fields[index].Type.ArrayElementType.Code, wantType.ArrayElement.Code; got != want {
						t.Fatalf("field %s array element code = %s, want %s", column.Name, got, want)
					}
				}
			}
		})
	}
}

func TestInformationSchemaDocsOnlyColumnsAreNotProjected(t *testing.T) {
	catalog, err := BuildSchemaCatalog("schema.sql", "")
	if err != nil {
		t.Fatal(err)
	}
	for tableName, columnNames := range map[string][]string{
		"ROLE_TABLE_GRANTS":   {"GRANTOR", "IS_GRANTABLE"},
		"ROLE_COLUMN_GRANTS":  {"GRANTOR", "IS_GRANTABLE"},
		"ROLE_MODEL_GRANTS":   {"GRANTOR", "IS_GRANTABLE"},
		"ROLE_ROUTINE_GRANTS": {"GRANTOR", "IS_GRANTABLE"},
		"TABLE_SYNONYMS":      {"SYNONYM_TABLE_NAME"},
	} {
		table := catalog.Tables[informationSchemaName+"."+tableName]
		if table == nil {
			t.Fatalf("table %s is absent", tableName)
		}
		for _, columnName := range columnNames {
			if column, _ := table.Column(columnName); column != nil {
				t.Errorf("docs-only column %s.%s was projected", tableName, columnName)
			}
		}
	}
}

func decodedInformationSchemaManifest(t *testing.T) informationSchemaManifest {
	t.Helper()
	var manifest informationSchemaManifest
	if err := json.Unmarshal(embeddedInformationSchemaManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func encodedInformationSchemaManifest(t *testing.T, manifest informationSchemaManifest) []byte {
	t.Helper()
	hash, err := hashInformationSchemaManifestTables(manifest.Tables)
	if err != nil {
		t.Fatal(err)
	}
	manifest.ContentSHA256 = hash
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
