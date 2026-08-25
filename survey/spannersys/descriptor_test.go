package spannersys

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestRegistryDescriptorsCoverKnownSurface(t *testing.T) {
	tables, err := registryDescriptors()
	if err != nil {
		t.Fatalf("registryDescriptors: %v", err)
	}
	if len(tables) != 51 {
		t.Fatalf("table descriptors = %d, want 51", len(tables))
	}
	registry, err := tableRegistry()
	if err != nil {
		t.Fatalf("tableRegistry: %v", err)
	}

	tableNames := make([]string, 0, len(tables))
	columnCount := 0
	shapes := make(map[string]bool)
	for _, table := range tables {
		tableNames = append(tableNames, table.Name)
		wantColumnOrder := orderedStructColumns(registry[table.Name])
		if len(table.Columns) != len(wantColumnOrder) {
			t.Fatalf("%s columns = %d, want %d", table.Name, len(table.Columns), len(wantColumnOrder))
		}
		for i, column := range table.Columns {
			columnCount++
			if column.Name != wantColumnOrder[i] {
				t.Errorf("%s column %d = %s, want %s", table.Name, i, column.Name, wantColumnOrder[i])
			}
			canonical, err := canonicalSpannerType(column.Type)
			if err != nil {
				t.Fatalf("canonicalSpannerType(%s.%s): %v", table.Name, column.Name, err)
			}
			shapes[canonical] = true
		}
	}
	if columnCount != 547 {
		t.Errorf("column descriptors = %d, want 547", columnCount)
	}
	if !sort.StringsAreSorted(tableNames) {
		t.Errorf("table descriptors are not sorted: %v", tableNames)
	}

	wantShapes := map[string]bool{
		"BOOL":               true,
		"INT64":              true,
		"FLOAT64":            true,
		"STRING(MAX)":        true,
		"BYTES(MAX)":         true,
		"DATE":               true,
		"TIMESTAMP":          true,
		"ARRAY<INT64>":       true,
		"ARRAY<STRING(MAX)>": true,
		"ARRAY<STRUCT<COLUMN STRING(MAX), LOCK_MODE STRING(MAX), TRANSACTION_TAG STRING(MAX)>>":                                                                                  true,
		"ARRAY<STRUCT<TABLE_NAME STRING(MAX), INSERT_OR_UPDATE_COUNT INT64, INSERT_OR_UPDATE_BYTES INT64>>":                                                                      true,
		"ARRAY<STRUCT<COUNT INT64, MEAN FLOAT64, SUM_OF_SQUARED_DEVIATION FLOAT64, NUM_FINITE_BUCKETS INT64, GROWTH_FACTOR FLOAT64, SCALE FLOAT64, BUCKET_COUNTS ARRAY<INT64>>>": true,
		"ARRAY<STRUCT<percentile INT64, value_at_percentile INT64>>":                                                                                                             true,
		"ARRAY<STRUCT<percentile INT64, value_at_percentile FLOAT64>>":                                                                                                           true,
	}
	if !reflect.DeepEqual(shapes, wantShapes) {
		t.Errorf("canonical shapes =\n%v\nwant\n%v", sortedKeys(shapes), sortedKeys(wantShapes))
	}

	wantExpanded := []string{
		"QUERY_STATS_TOP_MINUTE",
		"QUERY_STATS_TOP_10MINUTE",
		"QUERY_STATS_TOP_HOUR",
		"QUERY_PROFILES_TOP_MINUTE",
		"QUERY_PROFILES_TOP_10MINUTE",
		"QUERY_PROFILES_TOP_HOUR",
	}
	tableSet := make(map[string]bool, len(tableNames))
	for _, tableName := range tableNames {
		tableSet[tableName] = true
	}
	for _, tableName := range wantExpanded {
		if !tableSet[tableName] {
			t.Errorf("expanded registry missing %s", tableName)
		}
	}

	wantNested := map[string]string{
		"LOCK_STATS_TOP_MINUTE.SAMPLE_LOCK_REQUESTS":                                    "ARRAY<STRUCT<COLUMN STRING(MAX), LOCK_MODE STRING(MAX), TRANSACTION_TAG STRING(MAX)>>",
		"TXN_STATS_TOP_MINUTE.OPERATIONS_BY_TABLE":                                      "ARRAY<STRUCT<TABLE_NAME STRING(MAX), INSERT_OR_UPDATE_COUNT INT64, INSERT_OR_UPDATE_BYTES INT64>>",
		"QUERY_STATS_TOP_MINUTE.LATENCY_DISTRIBUTION":                                   "ARRAY<STRUCT<COUNT INT64, MEAN FLOAT64, SUM_OF_SQUARED_DEVIATION FLOAT64, NUM_FINITE_BUCKETS INT64, GROWTH_FACTOR FLOAT64, SCALE FLOAT64, BUCKET_COUNTS ARRAY<INT64>>>",
		"VECTOR_INDEX_METRICS_HISTORY.CLUSTER_SIZE_PERCENTILES":                         "ARRAY<STRUCT<percentile INT64, value_at_percentile INT64>>",
		"VECTOR_INDEX_METRICS_HISTORY.CLUSTER_AVERAGE_DISTANCE_TO_CENTROID_PERCENTILES": "ARRAY<STRUCT<percentile INT64, value_at_percentile FLOAT64>>",
	}
	for path, want := range wantNested {
		parts := strings.SplitN(path, ".", 2)
		column := findDescriptorColumn(t, tables, parts[0], parts[1])
		if got := mustCanonicalType(t, column.Type); got != want {
			t.Errorf("%s canonical type = %q, want %q", path, got, want)
		}
	}
}

func TestRegistryDescriptorsRetainDecoderNullabilitySeparately(t *testing.T) {
	tables, err := registryDescriptors()
	if err != nil {
		t.Fatalf("registryDescriptors: %v", err)
	}

	clientIP := findDescriptorColumn(t, tables, "OLDEST_ACTIVE_QUERIES", "CLIENT_IP_ADDRESS")
	if !clientIP.DecoderNullable {
		t.Error("CLIENT_IP_ADDRESS decoder nullability = false, want true")
	}
	if got := mustCanonicalType(t, clientIP.Type); got != "STRING(MAX)" {
		t.Errorf("CLIENT_IP_ADDRESS type = %s, want STRING(MAX)", got)
	}

	lockSamples := findDescriptorColumn(t, tables, "LOCK_STATS_TOP_MINUTE", "SAMPLE_LOCK_REQUESTS")
	if lockSamples.DecoderNullable {
		t.Error("SAMPLE_LOCK_REQUESTS column decoder nullability = true, want false")
	}
	if !lockSamples.Type.ElementDecoderNullable {
		t.Error("SAMPLE_LOCK_REQUESTS element decoder nullability = false, want true")
	}
	if got := mustCanonicalType(t, lockSamples.Type); strings.Contains(got, "NULL") {
		t.Errorf("decoder nullability leaked into canonical type %q", got)
	}
}

func TestRegistryDescriptorsAreFreshAndDeterministic(t *testing.T) {
	first, err := registryDescriptors()
	if err != nil {
		t.Fatalf("first registryDescriptors: %v", err)
	}
	second, err := registryDescriptors()
	if err != nil {
		t.Fatalf("second registryDescriptors: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("registryDescriptors is not deterministic")
	}
	first[0].Name = "MUTATED"
	third, err := registryDescriptors()
	if err != nil {
		t.Fatalf("third registryDescriptors: %v", err)
	}
	if third[0].Name == "MUTATED" {
		t.Error("registryDescriptors returned shared mutable state")
	}
}

func TestDescriptorsFromRegistryRejectsUnsupportedShapes(t *testing.T) {
	type namedString string
	type missingTag struct{ Value string }
	type duplicateTag struct {
		First  string `spanner:"DUP"`
		Second string `spanner:"DUP"`
	}
	type unexportedTopLevel struct {
		//nolint:unused // Intentionally exercises rejection of tagged unexported fields.
		value string `spanner:"VALUE"`
	}
	type nestedMissingTag struct {
		Value struct{ Missing string } `spanner:"VALUE"`
	}
	type nestedDuplicateTag struct {
		Value struct {
			First  string `spanner:"DUP"`
			Second string `spanner:"DUP"`
		} `spanner:"VALUE"`
	}
	type nestedUnexported struct {
		Value struct {
			//nolint:unused // Intentionally exercises rejection of tagged unexported fields.
			value string `spanner:"VALUE"`
		} `spanner:"VALUE"`
	}
	type nestedArray struct {
		Value [][]int64 `spanner:"VALUE"`
	}
	type recursive struct {
		Next *recursive `spanner:"NEXT"`
	}
	type recursiveRow struct {
		Value recursive `spanner:"VALUE"`
	}
	type emptyStruct struct {
		Value struct{} `spanner:"VALUE"`
	}
	type unsupportedMap struct {
		Value map[string]string `spanner:"VALUE"`
	}
	type unsupportedNamedScalar struct {
		Value namedString `spanner:"VALUE"`
	}

	tests := []struct {
		name     string
		rowType  reflect.Type
		contains string
	}{
		{name: "nil row type", contains: "nil row type"},
		{name: "non struct row", rowType: reflect.TypeFor[string](), contains: "is not a struct"},
		{name: "missing top-level tag", rowType: reflect.TypeFor[missingTag](), contains: "has no spanner tag"},
		{name: "duplicate top-level tag", rowType: reflect.TypeFor[duplicateTag](), contains: "duplicate spanner tag"},
		{name: "unexported top-level field", rowType: reflect.TypeFor[unexportedTopLevel](), contains: "is not exported"},
		{name: "missing nested tag", rowType: reflect.TypeFor[nestedMissingTag](), contains: "has no spanner tag"},
		{name: "duplicate nested tag", rowType: reflect.TypeFor[nestedDuplicateTag](), contains: "duplicate spanner tag"},
		{name: "unexported nested field", rowType: reflect.TypeFor[nestedUnexported](), contains: "is not exported"},
		{name: "nested array", rowType: reflect.TypeFor[nestedArray](), contains: "nested arrays"},
		{name: "recursive struct", rowType: reflect.TypeFor[recursiveRow](), contains: "recursive struct"},
		{name: "empty struct", rowType: reflect.TypeFor[emptyStruct](), contains: "empty struct"},
		{name: "map", rowType: reflect.TypeFor[unsupportedMap](), contains: "unsupported Go decoder type"},
		{name: "named scalar", rowType: reflect.TypeFor[unsupportedNamedScalar](), contains: "unsupported Go decoder type"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := descriptorsFromRegistry(map[string]reflect.Type{"TEST": test.rowType})
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("descriptorsFromRegistry() error = %v, want containing %q", err, test.contains)
			}
		})
	}
}

func TestCanonicalSpannerTypeRejectsMalformedDescriptors(t *testing.T) {
	intType := scalarDescriptor(scalarInt64)
	arrayType := typeDescriptor{Kind: typeKindArray, Element: &intType}
	tests := []struct {
		name       string
		descriptor typeDescriptor
	}{
		{name: "unknown kind", descriptor: typeDescriptor{Kind: "unknown"}},
		{name: "unknown scalar", descriptor: typeDescriptor{Kind: typeKindScalar, Scalar: "NUMERIC"}},
		{name: "scalar with element", descriptor: typeDescriptor{Kind: typeKindScalar, Scalar: scalarInt64, Element: &intType}},
		{name: "array without element", descriptor: typeDescriptor{Kind: typeKindArray}},
		{name: "nested array", descriptor: typeDescriptor{Kind: typeKindArray, Element: &arrayType}},
		{name: "empty struct", descriptor: typeDescriptor{Kind: typeKindStruct}},
		{name: "empty field name", descriptor: typeDescriptor{Kind: typeKindStruct, Fields: []structFieldDescriptor{{Type: intType}}}},
		{name: "duplicate field", descriptor: typeDescriptor{Kind: typeKindStruct, Fields: []structFieldDescriptor{{Name: "X", Type: intType}, {Name: "X", Type: intType}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := canonicalSpannerType(test.descriptor); err == nil {
				t.Fatal("canonicalSpannerType() error = nil")
			}
		})
	}
}

func TestCompareLiveTypes(t *testing.T) {
	descriptors := []tableDescriptor{{
		Name: "KNOWN",
		Columns: []columnDescriptor{
			{Name: "A", Type: scalarDescriptor(scalarStringMax), DecoderNullable: true},
			{Name: "B", Type: scalarDescriptor(scalarInt64)},
			{Name: "ABSENT", Type: scalarDescriptor(scalarBool)},
		},
	}}
	mismatches, err := compareLiveTypes(descriptors, []ColumnObservation{
		{TableName: "KNOWN", ColumnName: "A", SpannerType: "STRING(MAX)", OrdinalPosition: 1},
		{TableName: "KNOWN", ColumnName: "B", SpannerType: "FLOAT64", OrdinalPosition: 2},
		{TableName: "KNOWN", ColumnName: "UNKNOWN", SpannerType: "BOOL", OrdinalPosition: 3},
		{TableName: "UNKNOWN", ColumnName: "A", SpannerType: "STRING(MAX)", OrdinalPosition: 1},
	})
	if err != nil {
		t.Fatalf("compareLiveTypes: %v", err)
	}
	want := []ColumnTypeMismatch{{
		TableName: "KNOWN", ColumnName: "B", ObservedType: "FLOAT64", ExpectedType: "INT64",
	}}
	if !reflect.DeepEqual(mismatches, want) {
		t.Errorf("mismatches = %#v, want %#v", mismatches, want)
	}
}

func TestCompareLiveTypesRejectsInvalidDescriptors(t *testing.T) {
	descriptors := []tableDescriptor{
		{Name: "DUP"},
		{Name: "DUP"},
	}
	if _, err := compareLiveTypes(descriptors, nil); err == nil {
		t.Fatal("compareLiveTypes() duplicate-table error = nil")
	}
}

func findDescriptorColumn(
	t *testing.T,
	tables []tableDescriptor,
	tableName string,
	columnName string,
) columnDescriptor {
	t.Helper()
	for _, table := range tables {
		if table.Name != tableName {
			continue
		}
		for _, column := range table.Columns {
			if column.Name == columnName {
				return column
			}
		}
	}
	t.Fatalf("descriptor missing %s.%s", tableName, columnName)
	return columnDescriptor{}
}

func mustCanonicalType(t *testing.T, descriptor typeDescriptor) string {
	t.Helper()
	canonical, err := canonicalSpannerType(descriptor)
	if err != nil {
		t.Fatalf("canonicalSpannerType: %v", err)
	}
	return canonical
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
