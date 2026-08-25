package spannersys

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"cloud.google.com/go/spanner"
)

// AuditReport summarizes one complete live SPANNER_SYS surface checked by
// Audit. Registry entries missing from that successful observation are kept as
// known absent, separately from unknown advertised tables or columns.
type AuditReport struct {
	RegisteredTables   int
	AdvertisedTables   int
	AdvertisedColumns  int
	CheckedTables      int
	DecodedRows        int
	ObservedColumns    []ColumnObservation
	UnknownTables      []string
	UnknownColumns     map[string][]string
	KnownAbsentColumns map[string][]string
	OrdinalMismatches  []ColumnOrdinalMismatch
	TypeMismatches     []ColumnTypeMismatch
}

// ColumnObservation is one complete SPANNER_SYS column-metadata tuple returned
// by INFORMATION_SCHEMA.COLUMNS.
type ColumnObservation struct {
	TableName       string
	ColumnName      string
	SpannerType     string
	OrdinalPosition int
}

// ColumnOrdinalMismatch reports a disagreement between the live column order
// and the relative declaration order of the registered columns that target
// advertises. Registered columns absent from the target do not create a shift.
type ColumnOrdinalMismatch struct {
	TableName       string
	ColumnName      string
	ObservedOrdinal int
	ExpectedOrdinal int
}

// ColumnTypeMismatch reports a disagreement between a live raw type and the
// canonical structural type derived from the registered decoder.
type ColumnTypeMismatch struct {
	TableName    string
	ColumnName   string
	ObservedType string
	ExpectedType string
}

// HasDrift reports whether the live surface disagrees with the registered name,
// relative-order, or structural type contracts.
func (r *AuditReport) HasDrift() bool {
	return len(r.UnknownTables) > 0 || len(r.UnknownColumns) > 0 ||
		len(r.OrdinalMismatches) > 0 || len(r.TypeMismatches) > 0
}

// Audit discovers and retains the live SPANNER_SYS name, raw type, and ordinal
// surface and verifies that every advertised table and column is represented by
// the package structs in the same relative order. It also selects at most one
// row from each advertised known table and decodes it into the corresponding
// struct. Empty tables still receive a queryability check.
func Audit(ctx context.Context, client *spanner.Client) (*AuditReport, error) {
	registry, err := tableRegistry()
	if err != nil {
		return nil, err
	}

	discovered, err := discoverColumns(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("discover SPANNER_SYS columns: %w", err)
	}

	report, err := buildAuditReport(registry, discovered)
	if err != nil {
		return nil, fmt.Errorf("build SPANNER_SYS audit report: %w", err)
	}

	grouped := groupObservations(discovered)
	tableNames := make([]string, 0, len(grouped))
	for tableName := range grouped {
		tableNames = append(tableNames, tableName)
	}
	sort.Strings(tableNames)

	for _, tableName := range tableNames {
		columns := grouped[tableName]
		rowType, ok := registry[tableName]
		if !ok {
			continue
		}

		registered := structColumns(rowType)
		selected := make([]string, 0, len(columns))
		for _, column := range columns {
			if !registered[column.ColumnName] {
				continue
			}
			selected = append(selected, column.ColumnName)
		}
		if len(selected) == 0 {
			continue
		}

		query := fmt.Sprintf(
			"SELECT %s FROM SPANNER_SYS.`%s` AS t LIMIT 1",
			quoteIdentifiers(selected),
			tableName,
		)
		iter := client.Single().Query(ctx, spanner.NewStatement(query))
		destination := reflect.New(reflect.SliceOf(rowType))
		err := spanner.SelectAll(iter, destination.Interface(), spanner.WithLenient())
		iter.Stop()
		if err != nil {
			return report, fmt.Errorf("query and decode SPANNER_SYS.%s: %w", tableName, err)
		}
		report.CheckedTables++
		report.DecodedRows += destination.Elem().Len()
	}

	return report, nil
}

func discoverColumns(ctx context.Context, client *spanner.Client) ([]ColumnObservation, error) {
	type columnRow struct {
		TableName       string `spanner:"TABLE_NAME"`
		ColumnName      string `spanner:"COLUMN_NAME"`
		SpannerType     string `spanner:"SPANNER_TYPE"`
		OrdinalPosition int64  `spanner:"ORDINAL_POSITION"`
	}

	iter := client.Single().Query(ctx, spanner.NewStatement(columnDiscoveryQuery))
	defer iter.Stop()

	var rows []columnRow
	if err := spanner.SelectAll(iter, &rows); err != nil {
		return nil, err
	}
	discovered := make([]ColumnObservation, 0, len(rows))
	for _, row := range rows {
		discovered = append(discovered, ColumnObservation{
			TableName:       row.TableName,
			ColumnName:      row.ColumnName,
			SpannerType:     row.SpannerType,
			OrdinalPosition: int(row.OrdinalPosition),
		})
	}
	return discovered, nil
}

func buildAuditReport(
	registry map[string]reflect.Type,
	observations []ColumnObservation,
) (*AuditReport, error) {
	observed := append([]ColumnObservation(nil), observations...)
	sort.Slice(observed, func(i, j int) bool {
		if observed[i].TableName != observed[j].TableName {
			return observed[i].TableName < observed[j].TableName
		}
		if observed[i].OrdinalPosition != observed[j].OrdinalPosition {
			return observed[i].OrdinalPosition < observed[j].OrdinalPosition
		}
		return observed[i].ColumnName < observed[j].ColumnName
	})
	grouped := groupObservations(observed)
	report := &AuditReport{
		RegisteredTables:   len(registry),
		AdvertisedTables:   len(grouped),
		AdvertisedColumns:  len(observed),
		ObservedColumns:    observed,
		UnknownColumns:     make(map[string][]string),
		KnownAbsentColumns: make(map[string][]string),
	}

	for tableName, columns := range grouped {
		rowType, ok := registry[tableName]
		if !ok {
			report.UnknownTables = append(report.UnknownTables, tableName)
			continue
		}

		registered := structColumns(rowType)
		observedByName := make(map[string]ColumnObservation, len(columns))
		for _, column := range columns {
			observedByName[column.ColumnName] = column
			if !registered[column.ColumnName] {
				report.UnknownColumns[tableName] = append(report.UnknownColumns[tableName], column.ColumnName)
			}
		}

		expectedOrdinal := 0
		for _, columnName := range orderedStructColumns(rowType) {
			column, ok := observedByName[columnName]
			if !ok {
				continue
			}
			expectedOrdinal++
			if column.OrdinalPosition != expectedOrdinal {
				report.OrdinalMismatches = append(report.OrdinalMismatches, ColumnOrdinalMismatch{
					TableName:       tableName,
					ColumnName:      columnName,
					ObservedOrdinal: column.OrdinalPosition,
					ExpectedOrdinal: expectedOrdinal,
				})
			}
		}
	}

	registryNames := make([]string, 0, len(registry))
	for tableName := range registry {
		registryNames = append(registryNames, tableName)
	}
	sort.Strings(registryNames)
	for _, tableName := range registryNames {
		observedNames := make(map[string]bool)
		for _, column := range grouped[tableName] {
			observedNames[column.ColumnName] = true
		}
		for _, columnName := range orderedStructColumns(registry[tableName]) {
			if !observedNames[columnName] {
				report.KnownAbsentColumns[tableName] = append(report.KnownAbsentColumns[tableName], columnName)
			}
		}
	}

	sort.Strings(report.UnknownTables)
	sort.Slice(report.OrdinalMismatches, func(i, j int) bool {
		if report.OrdinalMismatches[i].TableName != report.OrdinalMismatches[j].TableName {
			return report.OrdinalMismatches[i].TableName < report.OrdinalMismatches[j].TableName
		}
		return report.OrdinalMismatches[i].ExpectedOrdinal < report.OrdinalMismatches[j].ExpectedOrdinal
	})
	descriptors, err := descriptorsFromRegistry(registry)
	if err != nil {
		return nil, err
	}
	report.TypeMismatches, err = compareLiveTypes(descriptors, observed)
	if err != nil {
		return nil, err
	}
	if len(report.UnknownColumns) == 0 {
		report.UnknownColumns = nil
	}
	if len(report.KnownAbsentColumns) == 0 {
		report.KnownAbsentColumns = nil
	}
	return report, nil
}

func groupObservations(observations []ColumnObservation) map[string][]ColumnObservation {
	grouped := make(map[string][]ColumnObservation)
	for _, observation := range observations {
		grouped[observation.TableName] = append(grouped[observation.TableName], observation)
	}
	return grouped
}

func quoteIdentifiers(columns []string) string {
	quoted := make([]string, len(columns))
	for i, column := range columns {
		// Qualify every column because QUERY_RECOMMENDATIONS and
		// SCHEMA_RECOMMENDATIONS each contain a column with the same name as
		// the table. Unqualified resolution treats that name as the row STRUCT.
		quoted[i] = "t.`" + column + "`"
	}
	return strings.Join(quoted, ", ")
}

func structColumns(rowType reflect.Type) map[string]bool {
	columns := make(map[string]bool, rowType.NumField())
	for i := 0; i < rowType.NumField(); i++ {
		if name := rowType.Field(i).Tag.Get("spanner"); name != "" {
			columns[name] = true
		}
	}
	return columns
}

func orderedStructColumns(rowType reflect.Type) []string {
	columns := make([]string, 0, rowType.NumField())
	for i := 0; i < rowType.NumField(); i++ {
		if name := rowType.Field(i).Tag.Get("spanner"); name != "" {
			columns = append(columns, name)
		}
	}
	return columns
}

func tableRegistry() (map[string]reflect.Type, error) {
	registry := make(map[string]reflect.Type, 51)
	add := func(name string, row any) error {
		rowType := reflect.TypeOf(row)
		if rowType.Kind() != reflect.Struct {
			return fmt.Errorf("SPANNER_SYS.%s row type %T is not a struct", name, row)
		}
		if _, exists := registry[name]; exists {
			return fmt.Errorf("duplicate SPANNER_SYS table %s", name)
		}
		registry[name] = rowType
		return nil
	}

	fixed := []struct {
		name string
		row  any
	}{
		{"ACTIVE_QUERIES_SUMMARY", ActiveQueriesSummary{}},
		{"OLDEST_ACTIVE_QUERIES", OldestActiveQuery{}},
		{"ACTIVE_PARTITIONED_DMLS", ActivePartitionedDML{}},
		{"QUERY_RECOMMENDATIONS", QueryRecommendation{}},
		{"ROW_DELETION_POLICIES", RowDeletionPolicy{}},
		{"SCHEMA_RECOMMENDATIONS", SchemaRecommendation{}},
		{"SPLIT_HOTNESS_STATS_TOP_MINUTE", SplitHotnessStatsTop{}},
		{"SUPPORTED_OPTIMIZER_VERSIONS", SupportedOptimizerVersion{}},
		{"TABLE_SIZES_STATS_1HOUR", TableSizesStats{}},
		{"TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR", TableSizesStatsPerLocalityGroup{}},
		{"TASKS", Task{}},
		{"USER_SPLIT_POINTS", UserSplitPoint{}},
		{"VECTOR_INDEX_STATS", VectorIndexStats{}},
		{"VECTOR_INDEX_METRICS_HISTORY", VectorIndexMetricsHistory{}},
		{"GRAPH_OPERATION_EXECUTION_STATUS", GraphOperationExecutionStatus{}},
	}
	for _, table := range fixed {
		if err := add(table.name, table.row); err != nil {
			return nil, err
		}
	}

	intervalTables := []struct {
		prefix string
		row    any
	}{
		{"QUERY_STATS_TOP_", QueryStatsTop{}},
		{"QUERY_STATS_TOTAL_", QueryStatsTotal{}},
		{"READ_STATS_TOP_", ReadStatsTop{}},
		{"READ_STATS_TOTAL_", ReadStatsTotal{}},
		{"TXN_STATS_TOP_", TxnStatsTop{}},
		{"TXN_STATS_TOTAL_", TxnStatsTotal{}},
		{"LOCK_STATS_TOP_", LockStatsTop{}},
		{"LOCK_STATS_TOTAL_", LockStatsTotal{}},
		{"COLUMN_OPERATIONS_STATS_", ColumnOperationsStats{}},
		{"TABLE_OPERATIONS_STATS_", TableOperationsStats{}},
		{"SPLIT_STATS_TOP_", SplitStatsTop{}},
		{"QUERY_PROFILES_TOP_", QueryProfilesTop{}},
	}
	for _, table := range intervalTables {
		for _, interval := range AllIntervals() {
			if err := add(table.prefix+string(interval), table.row); err != nil {
				return nil, err
			}
		}
	}

	return registry, nil
}
