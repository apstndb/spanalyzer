package spanalyzer

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/proto"
)

func TestSpannerSysManifestProjection(t *testing.T) {
	manifest, definitions, err := parseSpannerSysManifest(embeddedSpannerSysManifest)
	if err != nil {
		t.Fatalf("parseSpannerSysManifest() error = %v", err)
	}
	if got, want := len(manifest.Tables), 51; got != want {
		t.Fatalf("manifest table count = %d, want %d", got, want)
	}
	if got, want := len(definitions), 50; got != want {
		t.Fatalf("projected table count = %d, want %d", got, want)
	}

	definitionIndex := 0
	projectedColumns := 0
	absentColumns := 0
	for _, manifestTable := range manifest.Tables {
		if !manifestTable.Project {
			for _, column := range manifestTable.Columns {
				if column.Project {
					t.Fatalf("non-projecting table %s contains projecting column %s", manifestTable.Name, column.Name)
				}
				absentColumns++
			}
			continue
		}
		if definitionIndex >= len(definitions) {
			t.Fatalf("manifest projects table %s beyond definition count", manifestTable.Name)
		}
		definition := definitions[definitionIndex]
		definitionIndex++
		if definition.name != manifestTable.Name {
			t.Fatalf("definition table = %q, want %q", definition.name, manifestTable.Name)
		}
		definitionColumn := 0
		for _, manifestColumn := range manifestTable.Columns {
			if !manifestColumn.Project {
				absentColumns++
				continue
			}
			projectedColumns++
			if definitionColumn >= len(definition.columns) {
				t.Fatalf("definition %s lacks projected column %s", definition.name, manifestColumn.Name)
			}
			got := definition.columns[definitionColumn]
			definitionColumn++
			if got.name != manifestColumn.Name {
				t.Fatalf("definition %s column %d = %q, want %q", definition.name, definitionColumn, got.name, manifestColumn.Name)
			}
			want, err := spannerSysTypeSpec(manifestColumn.Type)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.typ, want) {
				t.Fatalf("definition %s.%s type = %#v, want %#v", definition.name, got.name, got.typ, want)
			}
		}
		if definitionColumn != len(definition.columns) {
			t.Fatalf("definition %s has %d columns, manifest projects %d", definition.name, len(definition.columns), definitionColumn)
		}
	}
	if definitionIndex != len(definitions) {
		t.Fatalf("visited %d definitions, have %d", definitionIndex, len(definitions))
	}
	if got, want := projectedColumns, 539; got != want {
		t.Errorf("projected column count = %d, want %d", got, want)
	}
	if got, want := absentColumns, 8; got != want {
		t.Errorf("absent column count = %d, want %d", got, want)
	}
}

func TestCatalogProjectsSpannerSysManifestExactly(t *testing.T) {
	manifest, _, err := parseSpannerSysManifest(embeddedSpannerSysManifest)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := BuildSchemaCatalog("schema.sql", "")
	if err != nil {
		t.Fatal(err)
	}

	projectedTableCount := 0
	projectedColumnCount := 0
	for _, manifestTable := range manifest.Tables {
		name := spannerSysName + "." + manifestTable.Name
		table := catalog.Tables[name]
		if !manifestTable.Project {
			if table != nil {
				t.Errorf("known-absent table %s was projected", name)
			}
			continue
		}
		projectedTableCount++
		if table == nil {
			t.Fatalf("projected table %s is absent", name)
		}
		columnIndex := 0
		for _, manifestColumn := range manifestTable.Columns {
			if !manifestColumn.Project {
				if column, _ := table.Column(manifestColumn.Name); column != nil {
					t.Errorf("known-absent column %s.%s was projected", name, manifestColumn.Name)
				}
				continue
			}
			projectedColumnCount++
			if columnIndex >= len(table.Columns) {
				t.Fatalf("table %s lacks projected column %s", name, manifestColumn.Name)
			}
			column := table.Columns[columnIndex]
			columnIndex++
			if column.Name != manifestColumn.Name {
				t.Fatalf("table %s column %d = %q, want %q", name, columnIndex, column.Name, manifestColumn.Name)
			}
			wantType, err := spannerSysTypeSpec(manifestColumn.Type)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(column.Type, wantType) {
				t.Fatalf("table %s column %s type = %#v, want %#v", name, column.Name, column.Type, wantType)
			}
			if column.NotNull {
				t.Errorf("table %s column %s is NotNull; decoder nullable evidence must not become SQL nullability", name, column.Name)
			}
		}
		if columnIndex != len(table.Columns) {
			t.Fatalf("table %s has %d columns, manifest projects %d", name, len(table.Columns), columnIndex)
		}
	}

	actualTableCount := 0
	for name := range catalog.Tables {
		if strings.HasPrefix(name, spannerSysName+".") {
			actualTableCount++
		}
	}
	if actualTableCount != projectedTableCount {
		t.Errorf("catalog SPANNER_SYS table count = %d, want %d", actualTableCount, projectedTableCount)
	}
	if got, want := projectedColumnCount, 539; got != want {
		t.Errorf("catalog SPANNER_SYS column count = %d, want %d", got, want)
	}
}

func TestAnalyzerProjectsEverySpannerSysTableInLiveOrdinalOrder(t *testing.T) {
	manifest, _, err := parseSpannerSysManifest(embeddedSpannerSysManifest)
	if err != nil {
		t.Fatal(err)
	}
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range manifest.Tables {
		if !table.Project {
			continue
		}
		t.Run(table.Name, func(t *testing.T) {
			rowType, err := analyzer.RowTypeForStatement(fmt.Sprintf("SELECT * FROM SPANNER_SYS.`%s`", table.Name))
			if err != nil {
				t.Fatalf("RowTypeForStatement() error = %v", err)
			}
			projected := make([]int, 0, len(table.Columns))
			for index, column := range table.Columns {
				if column.Project {
					projected = append(projected, index)
				}
			}
			if got, want := len(rowType.Fields), len(projected); got != want {
				t.Fatalf("field count = %d, want %d", got, want)
			}
			for fieldIndex, manifestIndex := range projected {
				column := table.Columns[manifestIndex]
				field := rowType.Fields[fieldIndex]
				if field.Name != column.Name {
					t.Fatalf("field %d name = %q, want %q", fieldIndex, field.Name, column.Name)
				}
				wantType, err := spannerSysTypeSpec(column.Type)
				if err != nil {
					t.Fatal(err)
				}
				wantPB, err := wantType.SpannerPB()
				if err != nil {
					t.Fatal(err)
				}
				if !proto.Equal(field.Type, wantPB) {
					t.Fatalf("field %s type = %v, want %v", field.Name, field.Type, wantPB)
				}
			}
		})
	}
}

func TestSpannerSysLiveTypeRepairs(t *testing.T) {
	catalog, err := BuildSchemaCatalog("schema.sql", "")
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"ACTIVE_PARTITIONED_DMLS.TEXT_FINGERPRINT",
		"ACTIVE_PARTITIONED_DMLS.PROGRESS",
	} {
		column := spannerSysCatalogColumn(t, catalog, path)
		if column.Type.Code != spannerpb.TypeCode_STRING || !column.Type.Max {
			t.Errorf("%s type = %#v, want STRING(MAX)", path, column.Type)
		}
	}

	lockRequests := spannerSysCatalogColumn(t, catalog, "LOCK_STATS_TOP_MINUTE.SAMPLE_LOCK_REQUESTS")
	if lockRequests.Type.Code != spannerpb.TypeCode_ARRAY || lockRequests.Type.ArrayElement == nil {
		t.Fatalf("lock request type = %#v, want ARRAY<STRUCT<...>>", lockRequests.Type)
	}
	lockFields := lockRequests.Type.ArrayElement.StructFields
	if got, want := len(lockFields), 3; got != want {
		t.Fatalf("lock request field count = %d, want %d", got, want)
	}
	if got, want := []string{lockFields[0].Name, lockFields[1].Name, lockFields[2].Name}, []string{"COLUMN", "LOCK_MODE", "TRANSACTION_TAG"}; !reflect.DeepEqual(got, want) {
		t.Errorf("lock request fields = %v, want %v", got, want)
	}
	for _, field := range lockFields {
		if field.Type.Code != spannerpb.TypeCode_STRING || !field.Type.Max {
			t.Errorf("lock request field %s type = %#v, want STRING(MAX)", field.Name, field.Type)
		}
	}

	operations := spannerSysCatalogColumn(t, catalog, "TXN_STATS_TOP_MINUTE.OPERATIONS_BY_TABLE")
	if operations.Type.Code != spannerpb.TypeCode_ARRAY || operations.Type.ArrayElement == nil || len(operations.Type.ArrayElement.StructFields) == 0 {
		t.Fatalf("operations-by-table type = %#v, want non-empty ARRAY<STRUCT<...>>", operations.Type)
	}
	if got, want := operations.Type.ArrayElement.StructFields[0].Name, "TABLE_NAME"; got != want {
		t.Errorf("operations-by-table first field = %q, want %q", got, want)
	}
	percentiles := spannerSysCatalogColumn(t, catalog, "VECTOR_INDEX_METRICS_HISTORY.CLUSTER_SIZE_PERCENTILES")
	if percentiles.Type.Code != spannerpb.TypeCode_ARRAY || percentiles.Type.ArrayElement == nil || len(percentiles.Type.ArrayElement.StructFields) < 2 {
		t.Fatalf("vector percentile type = %#v, want two-field ARRAY<STRUCT<...>>", percentiles.Type)
	}
	if got, want := percentiles.Type.ArrayElement.StructFields[1].Type.Code, spannerpb.TypeCode_INT64; got != want {
		t.Errorf("vector percentile value type = %s, want %s", got, want)
	}
}

func TestSpannerSysRetiredManualEntriesAreAbsent(t *testing.T) {
	catalog, err := BuildSchemaCatalog("schema.sql", "")
	if err != nil {
		t.Fatal(err)
	}
	if table := catalog.Tables[spannerSysName+".VECTOR_INDEX_STATS"]; table != nil {
		t.Error("VECTOR_INDEX_STATS was projected")
	}
	for _, path := range []string{
		"TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR.USED_BYTES",
		"VECTOR_INDEX_METRICS_HISTORY.NUM_CLUSTERS_SAMPLED",
		"VECTOR_INDEX_METRICS_HISTORY.NUM_ZERO_SIZE_CLUSTERS_SAMPLED",
	} {
		parts := strings.Split(path, ".")
		table := catalog.Tables[spannerSysName+"."+parts[0]]
		if table == nil {
			t.Fatalf("table %s is absent", parts[0])
		}
		if column, _ := table.Column(parts[1]); column != nil {
			t.Errorf("retired manual column %s was projected", path)
		}
	}
}

func spannerSysCatalogColumn(t *testing.T, catalog *Catalog, path string) *Column {
	t.Helper()
	parts := strings.Split(path, ".")
	if len(parts) != 2 {
		t.Fatalf("invalid SPANNER_SYS column path %q", path)
	}
	table := catalog.Tables[spannerSysName+"."+parts[0]]
	if table == nil {
		t.Fatalf("table %s is absent", parts[0])
	}
	column, _ := table.Column(parts[1])
	if column == nil {
		t.Fatalf("column %s is absent", path)
	}
	return column
}
