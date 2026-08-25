package spannersys

import (
	"reflect"
	"testing"
)

func TestTableRegistry(t *testing.T) {
	registry, err := tableRegistry()
	if err != nil {
		t.Fatalf("tableRegistry: %v", err)
	}
	if got := len(registry); got != 51 {
		t.Fatalf("registered tables = %d, want 51", got)
	}

	for tableName, rowType := range registry {
		columns := structColumns(rowType)
		if len(columns) != rowType.NumField() {
			t.Errorf(
				"SPANNER_SYS.%s has %d tagged columns for %d fields",
				tableName,
				len(columns),
				rowType.NumField(),
			)
		}
	}

	vectorStats, ok := registry["VECTOR_INDEX_STATS"]
	if !ok {
		t.Fatal("VECTOR_INDEX_STATS is not registered")
	}
	columns := structColumns(vectorStats)
	for _, column := range []string{
		"START_TIME",
		"VECTOR_INDEX_NAME",
		"NUM_LEAVES",
		"NUM_CLUSTERS_SAMPLED",
		"NUM_ZERO_SIZE_CLUSTERS_SAMPLED",
		"CLUSTER_SIZE_PERCENTILES",
		"CLUSTER_AVERAGE_DISTANCE_TO_CENTROID_PERCENTILES",
	} {
		if !columns[column] {
			t.Errorf("VECTOR_INDEX_STATS missing column %s", column)
		}
	}
	for fieldName, wantType := range map[string]reflect.Type{
		"NumClustersSampled":                          reflect.TypeFor[int64](),
		"ClusterSizePercentiles":                      reflect.TypeFor[[]*ClusterSizePercentile](),
		"ClusterAverageDistanceToCentroidPercentiles": reflect.TypeFor[[]*ClusterDistancePercentile](),
	} {
		field, ok := vectorStats.FieldByName(fieldName)
		if !ok {
			t.Errorf("VectorIndexStats missing Go field %s", fieldName)
			continue
		}
		if field.Type != wantType {
			t.Errorf("VectorIndexStats.%s type = %s, want %s", fieldName, field.Type, wantType)
		}
	}
}

func TestAuditReportHasDrift(t *testing.T) {
	for _, test := range []struct {
		name   string
		report AuditReport
		want   bool
	}{
		{name: "none"},
		{name: "known absent is not drift", report: AuditReport{KnownAbsentColumns: map[string][]string{"KNOWN": {"ABSENT"}}}},
		{name: "unknown table", report: AuditReport{UnknownTables: []string{"NEW_STATS"}}, want: true},
		{name: "unknown column", report: AuditReport{UnknownColumns: map[string][]string{"QUERY_STATS_TOP_MINUTE": {"NEW_METRIC"}}}, want: true},
		{name: "ordinal mismatch", report: AuditReport{OrdinalMismatches: []ColumnOrdinalMismatch{{TableName: "KNOWN", ColumnName: "B"}}}, want: true},
		{name: "type mismatch", report: AuditReport{TypeMismatches: []ColumnTypeMismatch{{TableName: "KNOWN", ColumnName: "B"}}}, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.report.HasDrift(); got != test.want {
				t.Errorf("HasDrift() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestBuildAuditReportChecksActivePartitionedDMLTypes(t *testing.T) {
	registry := map[string]reflect.Type{
		"ACTIVE_PARTITIONED_DMLS": reflect.TypeFor[ActivePartitionedDML](),
	}
	report := mustBuildAuditReport(t, registry, []ColumnObservation{
		{TableName: "ACTIVE_PARTITIONED_DMLS", ColumnName: "TEXT_FINGERPRINT", SpannerType: "INT64", OrdinalPosition: 1},
		{TableName: "ACTIVE_PARTITIONED_DMLS", ColumnName: "PROGRESS", SpannerType: "FLOAT64", OrdinalPosition: 2},
	})

	want := []ColumnTypeMismatch{
		{TableName: "ACTIVE_PARTITIONED_DMLS", ColumnName: "PROGRESS", ObservedType: "FLOAT64", ExpectedType: "STRING(MAX)"},
		{TableName: "ACTIVE_PARTITIONED_DMLS", ColumnName: "TEXT_FINGERPRINT", ObservedType: "INT64", ExpectedType: "STRING(MAX)"},
	}
	if !reflect.DeepEqual(report.TypeMismatches, want) {
		t.Errorf("type mismatches = %#v, want %#v", report.TypeMismatches, want)
	}
}

func TestBuildAuditReportRetainsMetadataAndSeparatesKnownAbsence(t *testing.T) {
	type row struct {
		First   string `spanner:"FIRST"`
		Missing string `spanner:"MISSING"`
		Last    int64  `spanner:"LAST"`
	}
	registry := map[string]reflect.Type{"KNOWN": reflect.TypeFor[row]()}
	observations := []ColumnObservation{
		{TableName: "KNOWN", ColumnName: "LAST", SpannerType: "INT64", OrdinalPosition: 2},
		{TableName: "KNOWN", ColumnName: "FIRST", SpannerType: "STRING(MAX)", OrdinalPosition: 1},
	}

	report := mustBuildAuditReport(t, registry, observations)
	if report.HasDrift() {
		t.Fatalf("HasDrift() = true, report = %#v", report)
	}
	if report.AdvertisedTables != 1 || report.AdvertisedColumns != 2 {
		t.Errorf("advertised surface = %d tables / %d columns, want 1 / 2", report.AdvertisedTables, report.AdvertisedColumns)
	}
	if want := []string{"MISSING"}; !reflect.DeepEqual(report.KnownAbsentColumns["KNOWN"], want) {
		t.Errorf("known absent = %v, want %v", report.KnownAbsentColumns["KNOWN"], want)
	}
	if got := report.ObservedColumns; !reflect.DeepEqual(got, []ColumnObservation{
		{TableName: "KNOWN", ColumnName: "FIRST", SpannerType: "STRING(MAX)", OrdinalPosition: 1},
		{TableName: "KNOWN", ColumnName: "LAST", SpannerType: "INT64", OrdinalPosition: 2},
	}) {
		t.Errorf("observed columns = %#v", got)
	}
}

func TestBuildAuditReportDetectsRelativeOrderDrift(t *testing.T) {
	type row struct {
		First string `spanner:"FIRST"`
		Last  int64  `spanner:"LAST"`
	}
	registry := map[string]reflect.Type{"KNOWN": reflect.TypeFor[row]()}
	report := mustBuildAuditReport(t, registry, []ColumnObservation{
		{TableName: "KNOWN", ColumnName: "LAST", SpannerType: "INT64", OrdinalPosition: 1},
		{TableName: "KNOWN", ColumnName: "FIRST", SpannerType: "STRING(MAX)", OrdinalPosition: 2},
	})

	want := []ColumnOrdinalMismatch{
		{TableName: "KNOWN", ColumnName: "FIRST", ObservedOrdinal: 2, ExpectedOrdinal: 1},
		{TableName: "KNOWN", ColumnName: "LAST", ObservedOrdinal: 1, ExpectedOrdinal: 2},
	}
	if !reflect.DeepEqual(report.OrdinalMismatches, want) {
		t.Errorf("ordinal mismatches = %#v, want %#v", report.OrdinalMismatches, want)
	}
}

func TestBuildAuditReportCompleteEmptySurfaceIsKnownAbsent(t *testing.T) {
	type row struct {
		Only string `spanner:"ONLY"`
	}
	report := mustBuildAuditReport(t,
		map[string]reflect.Type{"KNOWN": reflect.TypeFor[row]()},
		nil,
	)
	if report.HasDrift() {
		t.Fatalf("HasDrift() = true, report = %#v", report)
	}
	if want := []string{"ONLY"}; !reflect.DeepEqual(report.KnownAbsentColumns["KNOWN"], want) {
		t.Errorf("known absent = %v, want %v", report.KnownAbsentColumns["KNOWN"], want)
	}
}

func mustBuildAuditReport(
	t *testing.T,
	registry map[string]reflect.Type,
	observations []ColumnObservation,
) *AuditReport {
	t.Helper()
	report, err := buildAuditReport(registry, observations)
	if err != nil {
		t.Fatalf("buildAuditReport: %v", err)
	}
	return report
}

func TestQuoteIdentifiersQualifiesColumns(t *testing.T) {
	got := quoteIdentifiers([]string{"GENERATION_TIME", "QUERY_RECOMMENDATIONS"})
	want := "t.`GENERATION_TIME`, t.`QUERY_RECOMMENDATIONS`"
	if got != want {
		t.Errorf("quoteIdentifiers() = %q, want %q", got, want)
	}
}
