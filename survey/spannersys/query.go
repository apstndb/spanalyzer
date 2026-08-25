// Package spannersys provides Go structs for SPANNER_SYS tables.
package spannersys

import "fmt"

// Interval represents a SPANNER_SYS stats time interval.
type Interval string

// Known intervals.
const (
	IntervalMinute   Interval = "MINUTE"
	Interval10Minute Interval = "10MINUTE"
	IntervalHour     Interval = "HOUR"
)

// AllIntervals returns all standard intervals.
func AllIntervals() []Interval {
	return []Interval{IntervalMinute, Interval10Minute, IntervalHour}
}

// ValidateInterval reports whether interval names a supported SPANNER_SYS
// table suffix. It prevents arbitrary text from becoming part of an identifier.
func ValidateInterval(interval Interval) error {
	switch interval {
	case IntervalMinute, Interval10Minute, IntervalHour:
		return nil
	default:
		return fmt.Errorf("unsupported SPANNER_SYS interval %q", interval)
	}
}

// query builds "SELECT * FROM SPANNER_SYS.<prefix><interval>".
func query(prefix string, interval Interval) (string, error) {
	if err := ValidateInterval(interval); err != nil {
		return "", err
	}
	return fmt.Sprintf("SELECT * FROM SPANNER_SYS.%s%s", prefix, interval), nil
}

// QueryStatsTopQuery returns a query for QUERY_STATS_TOP_<interval>.
func QueryStatsTopQuery(interval Interval) (string, error) {
	return query("QUERY_STATS_TOP_", interval)
}

// QueryStatsTotalQuery returns a query for QUERY_STATS_TOTAL_<interval>.
func QueryStatsTotalQuery(interval Interval) (string, error) {
	return query("QUERY_STATS_TOTAL_", interval)
}

// ReadStatsTopQuery returns a query for READ_STATS_TOP_<interval>.
func ReadStatsTopQuery(interval Interval) (string, error) {
	return query("READ_STATS_TOP_", interval)
}

// ReadStatsTotalQuery returns a query for READ_STATS_TOTAL_<interval>.
func ReadStatsTotalQuery(interval Interval) (string, error) {
	return query("READ_STATS_TOTAL_", interval)
}

// TxnStatsTopQuery returns a query for TXN_STATS_TOP_<interval>.
func TxnStatsTopQuery(interval Interval) (string, error) {
	return query("TXN_STATS_TOP_", interval)
}

// TxnStatsTotalQuery returns a query for TXN_STATS_TOTAL_<interval>.
func TxnStatsTotalQuery(interval Interval) (string, error) {
	return query("TXN_STATS_TOTAL_", interval)
}

// LockStatsTopQuery returns a query for LOCK_STATS_TOP_<interval>.
func LockStatsTopQuery(interval Interval) (string, error) {
	return query("LOCK_STATS_TOP_", interval)
}

// LockStatsTotalQuery returns a query for LOCK_STATS_TOTAL_<interval>.
func LockStatsTotalQuery(interval Interval) (string, error) {
	return query("LOCK_STATS_TOTAL_", interval)
}

// ColumnOperationsStatsQuery returns a query for COLUMN_OPERATIONS_STATS_<interval>.
func ColumnOperationsStatsQuery(interval Interval) (string, error) {
	return query("COLUMN_OPERATIONS_STATS_", interval)
}

// TableOperationsStatsQuery returns a query for TABLE_OPERATIONS_STATS_<interval>.
func TableOperationsStatsQuery(interval Interval) (string, error) {
	return query("TABLE_OPERATIONS_STATS_", interval)
}

// SplitStatsTopQuery returns a query for SPLIT_STATS_TOP_<interval>.
func SplitStatsTopQuery(interval Interval) (string, error) {
	return query("SPLIT_STATS_TOP_", interval)
}

// QueryProfilesTopQuery returns a query for QUERY_PROFILES_TOP_<interval>.
func QueryProfilesTopQuery(interval Interval) (string, error) {
	return query("QUERY_PROFILES_TOP_", interval)
}

// --- Fixed-interval queries ---

// ActiveQueriesSummaryQuery returns a query for ACTIVE_QUERIES_SUMMARY.
func ActiveQueriesSummaryQuery() string {
	return "SELECT * FROM SPANNER_SYS.ACTIVE_QUERIES_SUMMARY"
}

// OldestActiveQueriesQuery returns a query for OLDEST_ACTIVE_QUERIES.
func OldestActiveQueriesQuery() string {
	return "SELECT * FROM SPANNER_SYS.OLDEST_ACTIVE_QUERIES"
}

// ActivePartitionedDMLsQuery returns a query for ACTIVE_PARTITIONED_DMLS.
func ActivePartitionedDMLsQuery() string {
	return "SELECT * FROM SPANNER_SYS.ACTIVE_PARTITIONED_DMLS"
}

// QueryRecommendationsQuery returns a query for QUERY_RECOMMENDATIONS.
func QueryRecommendationsQuery() string {
	return "SELECT * FROM SPANNER_SYS.QUERY_RECOMMENDATIONS"
}

// RowDeletionPoliciesQuery returns a query for ROW_DELETION_POLICIES.
func RowDeletionPoliciesQuery() string {
	return "SELECT * FROM SPANNER_SYS.ROW_DELETION_POLICIES"
}

// SchemaRecommendationsQuery returns a query for SCHEMA_RECOMMENDATIONS.
func SchemaRecommendationsQuery() string {
	return "SELECT * FROM SPANNER_SYS.SCHEMA_RECOMMENDATIONS"
}

// SplitHotnessStatsTopMinuteQuery returns a query for SPLIT_HOTNESS_STATS_TOP_MINUTE.
func SplitHotnessStatsTopMinuteQuery() string {
	return "SELECT * FROM SPANNER_SYS.SPLIT_HOTNESS_STATS_TOP_MINUTE"
}

// SupportedOptimizerVersionsQuery returns a query for SUPPORTED_OPTIMIZER_VERSIONS.
func SupportedOptimizerVersionsQuery() string {
	return "SELECT * FROM SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS"
}

// TableSizesStats1HourQuery returns a query for TABLE_SIZES_STATS_1HOUR.
func TableSizesStats1HourQuery() string {
	return "SELECT * FROM SPANNER_SYS.TABLE_SIZES_STATS_1HOUR"
}

// TableSizesStatsPerLocalityGroup1HourQuery returns a query for TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR.
func TableSizesStatsPerLocalityGroup1HourQuery() string {
	return "SELECT * FROM SPANNER_SYS.TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR"
}

// TasksQuery returns a query for TASKS.
func TasksQuery() string {
	return "SELECT * FROM SPANNER_SYS.TASKS"
}

// UserSplitPointsQuery returns a query for USER_SPLIT_POINTS.
func UserSplitPointsQuery() string {
	return "SELECT * FROM SPANNER_SYS.USER_SPLIT_POINTS"
}

// VectorIndexStatsQuery returns a query for VECTOR_INDEX_STATS.
func VectorIndexStatsQuery() string {
	return "SELECT * FROM SPANNER_SYS.VECTOR_INDEX_STATS"
}

// VectorIndexMetricsHistoryQuery returns a query for VECTOR_INDEX_METRICS_HISTORY.
func VectorIndexMetricsHistoryQuery() string {
	return "SELECT * FROM SPANNER_SYS.VECTOR_INDEX_METRICS_HISTORY"
}

// GraphOperationExecutionStatusQuery returns a query for GRAPH_OPERATION_EXECUTION_STATUS.
func GraphOperationExecutionStatusQuery() string {
	return "SELECT * FROM SPANNER_SYS.GRAPH_OPERATION_EXECUTION_STATUS"
}
