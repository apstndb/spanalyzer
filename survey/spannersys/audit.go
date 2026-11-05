package spannersys

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const closedTransactionErr = "cannot use a closed transaction"

// AuditReport summarizes one complete live SPANNER_SYS surface checked by
// Audit. Registry entries missing from that successful observation are kept as
// known absent, separately from unknown advertised tables or columns.
// AdvertisedNotQueryable lists columns advertised by INFORMATION_SCHEMA that
// the owning view rejected in this transaction; they remain counted in
// AdvertisedColumns and are not drift.
type AuditReport struct {
	RegisteredTables       int
	AdvertisedTables       int
	AdvertisedColumns      int
	CheckedTables          int
	DecodedRows            int
	ObservedColumns        []ColumnObservation
	UnknownTables          []string
	UnknownColumns         map[string][]string
	KnownAbsentColumns     map[string][]string
	AdvertisedNotQueryable map[string][]string
	OrdinalMismatches      []ColumnOrdinalMismatch
	TypeMismatches         []ColumnTypeMismatch
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
//
// Discovery and the per-table probes share one read-only transaction. During a
// managed-service rollout, consecutive single-use reads can be served by
// replicas that disagree about which columns exist, so a column advertised by
// discovery could otherwise be rejected by the probe that selects it. On
// InvalidArgument, Audit probes each selected column in that transaction,
// retries without advertised-but-not-queryable columns, and records them in
// AdvertisedNotQueryable without treating them as drift.
func Audit(ctx context.Context, client *spanner.Client) (*AuditReport, error) {
	registry, err := tableRegistry()
	if err != nil {
		return nil, err
	}

	txn := client.ReadOnlyTransaction()
	defer txn.Close()

	discovered, err := discoverSurfaceCaptureColumns(ctx, txn)
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

		decoded, excluded, err := decodeOneRowExcludingUnqueryable(ctx, txn, tableName, rowType, selected)
		if err != nil {
			return report, fmt.Errorf("query and decode SPANNER_SYS.%s: %w", tableName, err)
		}
		if len(excluded) > 0 {
			if report.AdvertisedNotQueryable == nil {
				report.AdvertisedNotQueryable = make(map[string][]string)
			}
			report.AdvertisedNotQueryable[tableName] = excluded
		}
		report.CheckedTables++
		report.DecodedRows += decoded
	}

	return report, nil
}

// rejectedNamePatterns match the GoogleSQL name-resolution errors Spanner
// returns when a selected column does not exist in the owning view. A match is
// honored only for a column this audit actually selected.
var rejectedNamePatterns = []*regexp.Regexp{
	regexp.MustCompile(`Unrecognized name: ([A-Za-z_][A-Za-z0-9_]*)`),
	regexp.MustCompile(`Name ([A-Za-z_][A-Za-z0-9_]*) not found inside`),
}

// columnsNamedByRejection returns the selected columns that an InvalidArgument
// rejection explicitly names, in selection order. During a managed-service
// rollout each request can be served by a server with a different view of a
// SPANNER_SYS table, so an individual per-column probe can succeed while the
// multi-column statement is rejected; the rejection is then the only
// request-consistent evidence of which column is not queryable.
func columnsNamedByRejection(err error, selected []string) []string {
	if err == nil || spanner.ErrCode(err) != codes.InvalidArgument || isClientClosedTransactionError(err) {
		return nil
	}
	message := status.Convert(err).Message()
	named := make(map[string]bool)
	for _, pattern := range rejectedNamePatterns {
		for _, match := range pattern.FindAllStringSubmatch(message, -1) {
			named[match[1]] = true
		}
	}
	var result []string
	for _, column := range selected {
		if named[column] {
			result = append(result, column)
		}
	}
	return result
}

// decodeOneRowExcludingUnqueryable selects at most one row of the given
// columns and decodes it into rowType. When the service rejects the statement
// with InvalidArgument, the columns the rejection names are excluded; if it
// names none, every remaining column is probed with the same statement shape
// and the columns the owning view rejects are excluded. The selection is then
// retried. Each round excludes at least one column, so the loop is bounded by
// the number of selected columns. It returns the decoded row count and the
// sorted excluded column names.
func decodeOneRowExcludingUnqueryable(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction,
	tableName string,
	rowType reflect.Type,
	selected []string,
) (int, []string, error) {
	var excluded []string
	for {
		if len(selected) == 0 {
			return 0, excluded, nil
		}
		query := fmt.Sprintf(
			"SELECT %s FROM SPANNER_SYS.`%s` AS t LIMIT 1",
			quoteIdentifiers(selected),
			tableName,
		)
		iter := txn.Query(ctx, spanner.NewStatement(query))
		destination := reflect.New(reflect.SliceOf(rowType))
		err := spanner.SelectAll(iter, destination.Interface(), spanner.WithLenient())
		iter.Stop()
		if err == nil {
			return destination.Elem().Len(), excluded, nil
		}
		if spanner.ErrCode(err) != codes.InvalidArgument || isClientClosedTransactionError(err) {
			return 0, excluded, err
		}
		kept, roundExcluded, probeErr := excludeRejectedOrUnqueryable(tableName, selected, err, func(table, column string) error {
			return probeSpannerSysColumn(ctx, txn, table, column)
		})
		if probeErr != nil {
			return 0, excluded, probeErr
		}
		if len(roundExcluded) == 0 {
			if len(excluded) > 0 {
				return 0, excluded, fmt.Errorf("still rejected after excluding %v: %w", excluded, err)
			}
			return 0, excluded, err
		}
		excluded = append(excluded, roundExcluded...)
		sort.Strings(excluded)
		selected = kept
	}
}

// excludeRejectedOrUnqueryable prefers the columns named by the rejection and
// falls back to individual probes when the rejection names none of them.
func excludeRejectedOrUnqueryable(
	tableName string,
	selected []string,
	rejection error,
	probe func(table, column string) error,
) (kept, excluded []string, err error) {
	if named := columnsNamedByRejection(rejection, selected); len(named) > 0 {
		namedSet := make(map[string]bool, len(named))
		for _, column := range named {
			namedSet[column] = true
		}
		for _, column := range selected {
			if !namedSet[column] {
				kept = append(kept, column)
			}
		}
		sort.Strings(named)
		return kept, named, nil
	}
	return excludeUnqueryableSelectedColumns(tableName, selected, probe)
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
	for field := range rowType.Fields() {
		if name := field.Tag.Get("spanner"); name != "" {
			columns[name] = true
		}
	}
	return columns
}

func orderedStructColumns(rowType reflect.Type) []string {
	columns := make([]string, 0, rowType.NumField())
	for field := range rowType.Fields() {
		if name := field.Tag.Get("spanner"); name != "" {
			columns = append(columns, name)
		}
	}
	return columns
}

func tableRegistry() (map[string]reflect.Type, error) {
	registry := make(map[string]reflect.Type, 53)
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
		{"ACTIVE_BATCH_OPERATIONS", BatchOperation{}},
		{"BATCH_OPERATIONS", BatchOperation{}},
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

func excludeUnqueryableSelectedColumns(
	tableName string,
	selected []string,
	probe func(table, column string) error,
) (kept, excluded []string, err error) {
	for _, column := range selected {
		if err := probe(tableName, column); err != nil {
			if isClientClosedTransactionError(err) {
				return nil, nil, fmt.Errorf("probe %s.%s: %w", tableName, column, err)
			}
			switch spanner.ErrCode(err) {
			case codes.InvalidArgument, codes.Unimplemented:
				excluded = append(excluded, column)
			default:
				return nil, nil, fmt.Errorf("probe %s.%s: %w", tableName, column, err)
			}
			continue
		}
		kept = append(kept, column)
	}
	sort.Strings(excluded)
	return kept, excluded, nil
}

// probeSpannerSysColumn selects one column with the same statement shape as
// the table probe. LIMIT 0 is deliberately not used: the stats views can
// accept a LIMIT 0 name resolution while still rejecting the column once a
// row is read, so the probe must exercise the same path as the real query.
func probeSpannerSysColumn(ctx context.Context, txn *spanner.ReadOnlyTransaction, tableName, columnName string) error {
	query := fmt.Sprintf("SELECT t.`%s` FROM SPANNER_SYS.`%s` AS t LIMIT 1", columnName, tableName)
	iter := txn.Query(ctx, spanner.NewStatement(query))
	defer iter.Stop()
	_, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
}

// isClientClosedTransactionError prevents a local transaction-lifecycle error
// from being mistaken for a backend rollout where a column is advertised
// before its owning view accepts it.
func isClientClosedTransactionError(err error) bool {
	return spanner.ErrCode(err) == codes.InvalidArgument && status.Convert(err).Message() == closedTransactionErr
}
