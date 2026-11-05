package spannersys

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTableRegistry(t *testing.T) {
	registry, err := tableRegistry()
	if err != nil {
		t.Fatalf("tableRegistry: %v", err)
	}
	if got := len(registry); got != 53 {
		t.Fatalf("registered tables = %d, want 53", got)
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
		{name: "advertised not queryable is not drift", report: AuditReport{AdvertisedNotQueryable: map[string][]string{"QUERY_STATS_TOP_MINUTE": {"CPU_SECONDS_DISTRIBUTION"}}}},
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

func TestExcludeUnqueryableSelectedColumns(t *testing.T) {
	tests := []struct {
		name      string
		probe     func(table, column string) error
		wantKept  []string
		wantExcl  []string
		wantErr   bool
		wantRetry bool
	}{
		{
			name:     "all queryable",
			wantKept: []string{"COL_A", "COL_B"},
		},
		{
			name: "one excluded",
			probe: func(_, column string) error {
				if column == "COL_B" {
					return status.Error(codes.InvalidArgument, "Name COL_B not found inside t")
				}
				return nil
			},
			wantKept:  []string{"COL_A"},
			wantExcl:  []string{"COL_B"},
			wantRetry: true,
		},
		{
			name:    "non-InvalidArgument probe error propagates",
			probe:   func(_, _ string) error { return errors.New("transport failed") },
			wantErr: true,
		},
		{
			name: "closed-transaction error propagates",
			probe: func(_, _ string) error {
				return status.Error(codes.InvalidArgument, closedTransactionErr)
			},
			wantErr: true,
		},
		{
			name:     "nothing excluded means no retry",
			wantKept: []string{"COL_A", "COL_B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			probe := tt.probe
			if probe == nil {
				probe = func(_, _ string) error { return nil }
			}
			calls := 0
			kept, excluded, err := excludeUnqueryableSelectedColumns("QUERY_STATS_TOP_MINUTE", []string{"COL_A", "COL_B"}, func(table, column string) error {
				calls++
				if table != "QUERY_STATS_TOP_MINUTE" {
					t.Fatalf("probe table = %q, want QUERY_STATS_TOP_MINUTE", table)
				}
				return probe(table, column)
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("excludeUnqueryableSelectedColumns() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if calls < 1 {
					t.Fatalf("probe calls = %d, want at least 1", calls)
				}
				if kept != nil || excluded != nil {
					t.Errorf("kept, excluded = %v, %v, want nil, nil on error", kept, excluded)
				}
				return
			}
			if calls != 2 {
				t.Fatalf("probe calls = %d, want 2", calls)
			}
			if !slices.Equal(kept, tt.wantKept) {
				t.Errorf("kept = %v, want %v", kept, tt.wantKept)
			}
			if !slices.Equal(excluded, tt.wantExcl) {
				t.Errorf("excluded = %v, want %v", excluded, tt.wantExcl)
			}
			gotRetry := len(excluded) > 0 && err == nil
			if gotRetry != tt.wantRetry {
				t.Errorf("retry = %t, want %t", gotRetry, tt.wantRetry)
			}
		})
	}
}
