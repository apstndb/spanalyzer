package spannersys

import (
	"time"

	"cloud.google.com/go/civil"
)

// --- Standalone tables (no interval variants) ---

// ActiveQueriesSummary represents SPANNER_SYS.ACTIVE_QUERIES_SUMMARY.
type ActiveQueriesSummary struct {
	ActiveCount        int64     `spanner:"ACTIVE_COUNT"`
	OldestStartTime    time.Time `spanner:"OLDEST_START_TIME"`
	CountOlderThan1s   int64     `spanner:"COUNT_OLDER_THAN_1S"`
	CountOlderThan10s  int64     `spanner:"COUNT_OLDER_THAN_10S"`
	CountOlderThan100s int64     `spanner:"COUNT_OLDER_THAN_100S"`
}

// OldestActiveQuery represents SPANNER_SYS.OLDEST_ACTIVE_QUERIES.
type OldestActiveQuery struct {
	StartTime       time.Time `spanner:"START_TIME"`
	TextFingerprint int64     `spanner:"TEXT_FINGERPRINT"`
	Text            string    `spanner:"TEXT"`
	TextTruncated   bool      `spanner:"TEXT_TRUNCATED"`
	SessionID       string    `spanner:"SESSION_ID"`
	QueryID         string    `spanner:"QUERY_ID"`
	ClientIPAddress *string   `spanner:"CLIENT_IP_ADDRESS"`
	APIClientHeader *string   `spanner:"API_CLIENT_HEADER"`
	UserAgentHeader *string   `spanner:"USER_AGENT_HEADER"`
	ServerRegion    *string   `spanner:"SERVER_REGION"`
	Priority        *string   `spanner:"PRIORITY"`
	TransactionType *string   `spanner:"TRANSACTION_TYPE"`
}

// ActivePartitionedDML represents SPANNER_SYS.ACTIVE_PARTITIONED_DMLS.
type ActivePartitionedDML struct {
	Text                         string    `spanner:"TEXT"`
	TextFingerprint              string    `spanner:"TEXT_FINGERPRINT"`
	SessionID                    string    `spanner:"SESSION_ID"`
	NumPartitionsTotal           int64     `spanner:"NUM_PARTITIONS_TOTAL"`
	NumPartitionsComplete        int64     `spanner:"NUM_PARTITIONS_COMPLETE"`
	NumTrivialPartitionsComplete int64     `spanner:"NUM_TRIVIAL_PARTITIONS_COMPLETE"`
	Progress                     string    `spanner:"PROGRESS"`
	RowsProcessed                int64     `spanner:"ROWS_PROCESSED"`
	StartTimestamp               time.Time `spanner:"START_TIMESTAMP"`
	LastUpdateTimestamp          time.Time `spanner:"LAST_UPDATE_TIMESTAMP"`
}

// QueryRecommendation represents SPANNER_SYS.QUERY_RECOMMENDATIONS.
type QueryRecommendation struct {
	GenerationTime       time.Time `spanner:"GENERATION_TIME"`
	TextFingerprint      int64     `spanner:"TEXT_FINGERPRINT"`
	QueryRecommendations string    `spanner:"QUERY_RECOMMENDATIONS"`
}

// RowDeletionPolicy represents SPANNER_SYS.ROW_DELETION_POLICIES.
type RowDeletionPolicy struct {
	TableName               string     `spanner:"TABLE_NAME"`
	ProcessedWatermark      time.Time  `spanner:"PROCESSED_WATERMARK"`
	UndeletableRows         int64      `spanner:"UNDELETABLE_ROWS"`
	MinUndeletableTimestamp *time.Time `spanner:"MIN_UNDELETABLE_TIMESTAMP"`
}

// SchemaRecommendation represents SPANNER_SYS.SCHEMA_RECOMMENDATIONS.
type SchemaRecommendation struct {
	GenerationTime        time.Time `spanner:"GENERATION_TIME"`
	Fingerprint           string    `spanner:"FINGERPRINT"`
	SchemaRecommendations string    `spanner:"SCHEMA_RECOMMENDATIONS"`
}

// SplitHotnessStatsTop represents SPANNER_SYS.SPLIT_HOTNESS_STATS_TOP_MINUTE.
type SplitHotnessStatsTop struct {
	IntervalEnd    time.Time `spanner:"INTERVAL_END"`
	SplitStart     string    `spanner:"SPLIT_START"`
	SplitLimit     string    `spanner:"SPLIT_LIMIT"`
	AffectedTables []string  `spanner:"AFFECTED_TABLES"`
	Hotness        int64     `spanner:"HOTNESS"`
}

// SupportedOptimizerVersion represents SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS.
type SupportedOptimizerVersion struct {
	Version     int64      `spanner:"VERSION"`
	ReleaseDate civil.Date `spanner:"RELEASE_DATE"`
	IsDefault   bool       `spanner:"IS_DEFAULT"`
}

// TableSizesStats represents SPANNER_SYS.TABLE_SIZES_STATS_1HOUR.
type TableSizesStats struct {
	IntervalEnd              time.Time `spanner:"INTERVAL_END"`
	TableName                string    `spanner:"TABLE_NAME"`
	UsedBytes                float64   `spanner:"USED_BYTES"`
	UsedSSDBytes             float64   `spanner:"USED_SSD_BYTES"`
	UsedHDDBytes             float64   `spanner:"USED_HDD_BYTES"`
	ColumnarUsedBytes        *float64  `spanner:"COLUMNAR_USED_BYTES"`
	ColumnarUsedSSDBytes     *float64  `spanner:"COLUMNAR_USED_SSD_BYTES"`
	ColumnarUsedHDDBytes     *float64  `spanner:"COLUMNAR_USED_HDD_BYTES"`
	ColumnarCoverageRatio    *float64  `spanner:"COLUMNAR_COVERAGE_RATIO"`
	ColumnarSSDCoverageRatio *float64  `spanner:"COLUMNAR_SSD_COVERAGE_RATIO"`
	ColumnarHDDCoverageRatio *float64  `spanner:"COLUMNAR_HDD_COVERAGE_RATIO"`
}

// TableSizesStatsPerLocalityGroup represents SPANNER_SYS.TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR.
type TableSizesStatsPerLocalityGroup struct {
	IntervalEnd   time.Time `spanner:"INTERVAL_END"`
	TableName     string    `spanner:"TABLE_NAME"`
	LocalityGroup string    `spanner:"LOCALITY_GROUP"`
	UsedBytes     float64   `spanner:"USED_BYTES"`
	UsedSSDBytes  float64   `spanner:"USED_SSD_BYTES"`
	UsedHDDBytes  float64   `spanner:"USED_HDD_BYTES"`
}

// Task represents SPANNER_SYS.TASKS.
type Task struct {
	TaskName           string    `spanner:"TASK_NAME"`
	ProcessedWatermark time.Time `spanner:"PROCESSED_WATERMARK"`
	LastRunStatus      string    `spanner:"LAST_RUN_STATUS"`
	UndeletableRows    int64     `spanner:"UNDELETABLE_ROWS"`
}

// UserSplitPoint represents SPANNER_SYS.USER_SPLIT_POINTS.
type UserSplitPoint struct {
	TableName  string    `spanner:"TABLE_NAME"`
	IndexName  string    `spanner:"INDEX_NAME"`
	Initiator  string    `spanner:"INITIATOR"`
	SplitKey   string    `spanner:"SPLIT_KEY"`
	ExpireTime time.Time `spanner:"EXPIRE_TIME"`
}

// VectorIndexStats represents the Enterprise and Enterprise Plus table
// SPANNER_SYS.VECTOR_INDEX_STATS documented at
// https://docs.cloud.google.com/spanner/docs/introspection/vector-index-statistics.
// The managed and Omni targets tested on 2026-08-25 did not advertise this
// edition-gated table, so its schema is documentation-backed rather than
// live-decoded in this repository.
type VectorIndexStats struct {
	StartTime                                   time.Time                    `spanner:"START_TIME"`
	VectorIndexName                             string                       `spanner:"VECTOR_INDEX_NAME"`
	NumLeaves                                   int64                        `spanner:"NUM_LEAVES"`
	NumClustersSampled                          int64                        `spanner:"NUM_CLUSTERS_SAMPLED"`
	NumZeroSizeClustersSampled                  int64                        `spanner:"NUM_ZERO_SIZE_CLUSTERS_SAMPLED"`
	ClusterSizePercentiles                      []*ClusterSizePercentile     `spanner:"CLUSTER_SIZE_PERCENTILES"`
	ClusterAverageDistanceToCentroidPercentiles []*ClusterDistancePercentile `spanner:"CLUSTER_AVERAGE_DISTANCE_TO_CENTROID_PERCENTILES"`
}

// VectorIndexMetricsHistory represents SPANNER_SYS.VECTOR_INDEX_METRICS_HISTORY.
type VectorIndexMetricsHistory struct {
	VectorIndexName                             string                       `spanner:"VECTOR_INDEX_NAME"`
	StartTime                                   time.Time                    `spanner:"START_TIME"`
	CompletionTime                              time.Time                    `spanner:"COMPLETION_TIME"`
	RowsScanned                                 int64                        `spanner:"ROWS_SCANNED"`
	ClustersSampled                             int64                        `spanner:"CLUSTERS_SAMPLED"`
	ZeroSizeClustersSampled                     int64                        `spanner:"ZERO_SIZE_CLUSTERS_SAMPLED"`
	MinClusterSize                              int64                        `spanner:"MIN_CLUSTER_SIZE"`
	MaxClusterSize                              int64                        `spanner:"MAX_CLUSTER_SIZE"`
	ClusterSizePercentiles                      []*ClusterSizePercentile     `spanner:"CLUSTER_SIZE_PERCENTILES"`
	ClusterAverageDistanceToCentroidPercentiles []*ClusterDistancePercentile `spanner:"CLUSTER_AVERAGE_DISTANCE_TO_CENTROID_PERCENTILES"`
	NumLeaves                                   int64                        `spanner:"NUM_LEAVES"`
	NumBranches                                 int64                        `spanner:"NUM_BRANCHES"`
}

// GraphOperationExecutionStatus represents
// SPANNER_SYS.GRAPH_OPERATION_EXECUTION_STATUS.
type GraphOperationExecutionStatus struct {
	QueryID             *string    `spanner:"QUERY_ID"`
	QueryText           *string    `spanner:"QUERY_TEXT"`
	StartTimestamp      *time.Time `spanner:"START_TIMESTAMP"`
	LastUpdateTimestamp *time.Time `spanner:"LAST_UPDATE_TIMESTAMP"`
	Progress            *float64   `spanner:"PROGRESS"`
	Status              *string    `spanner:"STATUS"`
	ErrorMessage        *string    `spanner:"ERROR_MESSAGE"`
}

// --- Interval-based tables (shared struct for MINUTE/10MINUTE/HOUR) ---

// QueryStatsTop represents SPANNER_SYS.QUERY_STATS_TOP_{MINUTE,10MINUTE,HOUR}.
type QueryStatsTop struct {
	IntervalEnd                           time.Time              `spanner:"INTERVAL_END"`
	Text                                  string                 `spanner:"TEXT"`
	TextTruncated                         bool                   `spanner:"TEXT_TRUNCATED"`
	TextFingerprint                       int64                  `spanner:"TEXT_FINGERPRINT"`
	ExecutionCount                        int64                  `spanner:"EXECUTION_COUNT"`
	AvgLatencySeconds                     float64                `spanner:"AVG_LATENCY_SECONDS"`
	AvgRows                               float64                `spanner:"AVG_ROWS"`
	AvgBytes                              float64                `spanner:"AVG_BYTES"`
	AvgRowsScanned                        float64                `spanner:"AVG_ROWS_SCANNED"`
	AvgCPUSeconds                         float64                `spanner:"AVG_CPU_SECONDS"`
	CancelledOrDisconnectedExecutionCount int64                  `spanner:"CANCELLED_OR_DISCONNECTED_EXECUTION_COUNT"`
	TimedOutExecutionCount                int64                  `spanner:"TIMED_OUT_EXECUTION_COUNT"`
	AllFailedExecutionCount               int64                  `spanner:"ALL_FAILED_EXECUTION_COUNT"`
	AllFailedAvgLatencySeconds            float64                `spanner:"ALL_FAILED_AVG_LATENCY_SECONDS"`
	RequestTag                            *string                `spanner:"REQUEST_TAG"`
	AvgBytesWritten                       float64                `spanner:"AVG_BYTES_WRITTEN"`
	AvgRowsWritten                        float64                `spanner:"AVG_ROWS_WRITTEN"`
	StatementCount                        int64                  `spanner:"STATEMENT_COUNT"`
	LatencyDistribution                   []*LatencyDistribution `spanner:"LATENCY_DISTRIBUTION"`
	RunInRWTransactionExecutionCount      int64                  `spanner:"RUN_IN_RW_TRANSACTION_EXECUTION_COUNT"`
	QueryType                             *string                `spanner:"QUERY_TYPE"`
	AvgMemoryPeakUsageBytes               float64                `spanner:"AVG_MEMORY_PEAK_USAGE_BYTES"`
	AvgMemoryUsagePercentage              float64                `spanner:"AVG_MEMORY_USAGE_PERCENTAGE"`
	AvgQueryPlanCreationTimeSecs          float64                `spanner:"AVG_QUERY_PLAN_CREATION_TIME_SECS"`
	AvgFilesystemDelaySecs                float64                `spanner:"AVG_FILESYSTEM_DELAY_SECS"`
	AvgRemoteServerCalls                  float64                `spanner:"AVG_REMOTE_SERVER_CALLS"`
	AvgRowsSpooled                        float64                `spanner:"AVG_ROWS_SPOOLED"`
	AvgDiskIOCost                         float64                `spanner:"AVG_DISK_IO_COST"`
	LatencyDistributionJSONString         *string                `spanner:"LATENCY_DISTRIBUTION_JSON_STRING"`
	AvgColumnarReadShare                  *float64               `spanner:"AVG_COLUMNAR_READ_SHARE"`
	QueryOptimizerVersions                []int64                `spanner:"QUERY_OPTIMIZER_VERSIONS"`
	StatisticsPackageNames                []string               `spanner:"STATISTICS_PACKAGE_NAMES"`
}

// QueryStatsTotal represents SPANNER_SYS.QUERY_STATS_TOTAL_{MINUTE,10MINUTE,HOUR}.
type QueryStatsTotal struct {
	IntervalEnd                           time.Time              `spanner:"INTERVAL_END"`
	ExecutionCount                        int64                  `spanner:"EXECUTION_COUNT"`
	AvgLatencySeconds                     float64                `spanner:"AVG_LATENCY_SECONDS"`
	AvgRows                               float64                `spanner:"AVG_ROWS"`
	AvgBytes                              float64                `spanner:"AVG_BYTES"`
	AvgRowsScanned                        float64                `spanner:"AVG_ROWS_SCANNED"`
	AvgCPUSeconds                         float64                `spanner:"AVG_CPU_SECONDS"`
	CancelledOrDisconnectedExecutionCount int64                  `spanner:"CANCELLED_OR_DISCONNECTED_EXECUTION_COUNT"`
	TimedOutExecutionCount                int64                  `spanner:"TIMED_OUT_EXECUTION_COUNT"`
	AllFailedExecutionCount               int64                  `spanner:"ALL_FAILED_EXECUTION_COUNT"`
	AllFailedAvgLatencySeconds            float64                `spanner:"ALL_FAILED_AVG_LATENCY_SECONDS"`
	AvgBytesWritten                       float64                `spanner:"AVG_BYTES_WRITTEN"`
	AvgRowsWritten                        float64                `spanner:"AVG_ROWS_WRITTEN"`
	LatencyDistribution                   []*LatencyDistribution `spanner:"LATENCY_DISTRIBUTION"`
	RunInRWTransactionExecutionCount      int64                  `spanner:"RUN_IN_RW_TRANSACTION_EXECUTION_COUNT"`
	AvgMemoryPeakUsageBytes               float64                `spanner:"AVG_MEMORY_PEAK_USAGE_BYTES"`
	AvgMemoryUsagePercentage              float64                `spanner:"AVG_MEMORY_USAGE_PERCENTAGE"`
	AvgQueryPlanCreationTimeSecs          float64                `spanner:"AVG_QUERY_PLAN_CREATION_TIME_SECS"`
	AvgFilesystemDelaySecs                float64                `spanner:"AVG_FILESYSTEM_DELAY_SECS"`
	AvgRemoteServerCalls                  float64                `spanner:"AVG_REMOTE_SERVER_CALLS"`
	AvgRowsSpooled                        float64                `spanner:"AVG_ROWS_SPOOLED"`
	AvgDiskIOCost                         float64                `spanner:"AVG_DISK_IO_COST"`
	LatencyDistributionJSONString         *string                `spanner:"LATENCY_DISTRIBUTION_JSON_STRING"`
	AvgColumnarReadShare                  *float64               `spanner:"AVG_COLUMNAR_READ_SHARE"`
}

// ReadStatsTop represents SPANNER_SYS.READ_STATS_TOP_{MINUTE,10MINUTE,HOUR}.
type ReadStatsTop struct {
	IntervalEnd                      time.Time `spanner:"INTERVAL_END"`
	ReadColumns                      []string  `spanner:"READ_COLUMNS"`
	Fprint                           int64     `spanner:"FPRINT"`
	ExecutionCount                   int64     `spanner:"EXECUTION_COUNT"`
	AvgRows                          float64   `spanner:"AVG_ROWS"`
	AvgBytes                         float64   `spanner:"AVG_BYTES"`
	AvgCPUSeconds                    float64   `spanner:"AVG_CPU_SECONDS"`
	AvgLockingDelaySeconds           float64   `spanner:"AVG_LOCKING_DELAY_SECONDS"`
	AvgClientWaitSeconds             float64   `spanner:"AVG_CLIENT_WAIT_SECONDS"`
	AvgLeaderRefreshDelaySeconds     float64   `spanner:"AVG_LEADER_REFRESH_DELAY_SECONDS"`
	RequestTag                       *string   `spanner:"REQUEST_TAG"`
	RunInRWTransactionExecutionCount int64     `spanner:"RUN_IN_RW_TRANSACTION_EXECUTION_COUNT"`
	ReadType                         *string   `spanner:"READ_TYPE"`
	AvgDiskIOCost                    float64   `spanner:"AVG_DISK_IO_COST"`
}

// ReadStatsTotal represents SPANNER_SYS.READ_STATS_TOTAL_{MINUTE,10MINUTE,HOUR}.
type ReadStatsTotal struct {
	IntervalEnd                      time.Time `spanner:"INTERVAL_END"`
	ExecutionCount                   int64     `spanner:"EXECUTION_COUNT"`
	AvgRows                          float64   `spanner:"AVG_ROWS"`
	AvgBytes                         float64   `spanner:"AVG_BYTES"`
	AvgCPUSeconds                    float64   `spanner:"AVG_CPU_SECONDS"`
	AvgLockingDelaySeconds           float64   `spanner:"AVG_LOCKING_DELAY_SECONDS"`
	AvgClientWaitSeconds             float64   `spanner:"AVG_CLIENT_WAIT_SECONDS"`
	AvgLeaderRefreshDelaySeconds     float64   `spanner:"AVG_LEADER_REFRESH_DELAY_SECONDS"`
	RunInRWTransactionExecutionCount int64     `spanner:"RUN_IN_RW_TRANSACTION_EXECUTION_COUNT"`
	AvgDiskIOCost                    float64   `spanner:"AVG_DISK_IO_COST"`
}

// TxnStatsTop represents SPANNER_SYS.TXN_STATS_TOP_{MINUTE,10MINUTE,HOUR}.
type TxnStatsTop struct {
	IntervalEnd                        time.Time              `spanner:"INTERVAL_END"`
	Fprint                             int64                  `spanner:"FPRINT"`
	ReadColumns                        []string               `spanner:"READ_COLUMNS"`
	WriteConstructiveColumns           []string               `spanner:"WRITE_CONSTRUCTIVE_COLUMNS"`
	WriteDeleteTables                  []string               `spanner:"WRITE_DELETE_TABLES"`
	CommitAttemptCount                 int64                  `spanner:"COMMIT_ATTEMPT_COUNT"`
	CommitFailedPreconditionCount      int64                  `spanner:"COMMIT_FAILED_PRECONDITION_COUNT"`
	CommitAbortCount                   int64                  `spanner:"COMMIT_ABORT_COUNT"`
	AvgParticipants                    float64                `spanner:"AVG_PARTICIPANTS"`
	AvgTotalLatencySeconds             float64                `spanner:"AVG_TOTAL_LATENCY_SECONDS"`
	AvgCommitLatencySeconds            float64                `spanner:"AVG_COMMIT_LATENCY_SECONDS"`
	AvgBytes                           float64                `spanner:"AVG_BYTES"`
	CommitRetryCount                   int64                  `spanner:"COMMIT_RETRY_COUNT"`
	TransactionTag                     *string                `spanner:"TRANSACTION_TAG"`
	OperationsByTable                  []*OperationsByTable   `spanner:"OPERATIONS_BY_TABLE"`
	TotalLatencyDistribution           []*LatencyDistribution `spanner:"TOTAL_LATENCY_DISTRIBUTION"`
	AttemptCount                       int64                  `spanner:"ATTEMPT_COUNT"`
	SerializablePessimisticTxnCount    int64                  `spanner:"SERIALIZABLE_PESSIMISTIC_TXN_COUNT"`
	RepeatableReadOptimisticTxnCount   int64                  `spanner:"REPEATABLE_READ_OPTIMISTIC_TXN_COUNT"`
	OperationsByTableJSONString        *string                `spanner:"OPERATIONS_BY_TABLE_JSON_STRING"`
	TotalLatencyDistributionJSONString *string                `spanner:"TOTAL_LATENCY_DISTRIBUTION_JSON_STRING"`
	SerializableOptimisticTxnCount     int64                  `spanner:"SERIALIZABLE_OPTIMISTIC_TXN_COUNT"`
	RepeatableReadPessimisticTxnCount  int64                  `spanner:"REPEATABLE_READ_PESSIMISTIC_TXN_COUNT"`
}

// TxnStatsTotal represents SPANNER_SYS.TXN_STATS_TOTAL_{MINUTE,10MINUTE,HOUR}.
type TxnStatsTotal struct {
	IntervalEnd                        time.Time              `spanner:"INTERVAL_END"`
	CommitAttemptCount                 int64                  `spanner:"COMMIT_ATTEMPT_COUNT"`
	CommitFailedPreconditionCount      int64                  `spanner:"COMMIT_FAILED_PRECONDITION_COUNT"`
	CommitAbortCount                   int64                  `spanner:"COMMIT_ABORT_COUNT"`
	AvgParticipants                    float64                `spanner:"AVG_PARTICIPANTS"`
	AvgTotalLatencySeconds             float64                `spanner:"AVG_TOTAL_LATENCY_SECONDS"`
	AvgCommitLatencySeconds            float64                `spanner:"AVG_COMMIT_LATENCY_SECONDS"`
	AvgBytes                           float64                `spanner:"AVG_BYTES"`
	CommitRetryCount                   int64                  `spanner:"COMMIT_RETRY_COUNT"`
	OperationsByTable                  []*OperationsByTable   `spanner:"OPERATIONS_BY_TABLE"`
	TotalLatencyDistribution           []*LatencyDistribution `spanner:"TOTAL_LATENCY_DISTRIBUTION"`
	AttemptCount                       int64                  `spanner:"ATTEMPT_COUNT"`
	SerializablePessimisticTxnCount    int64                  `spanner:"SERIALIZABLE_PESSIMISTIC_TXN_COUNT"`
	RepeatableReadOptimisticTxnCount   int64                  `spanner:"REPEATABLE_READ_OPTIMISTIC_TXN_COUNT"`
	OperationsByTableJSONString        *string                `spanner:"OPERATIONS_BY_TABLE_JSON_STRING"`
	TotalLatencyDistributionJSONString *string                `spanner:"TOTAL_LATENCY_DISTRIBUTION_JSON_STRING"`
	SerializableOptimisticTxnCount     int64                  `spanner:"SERIALIZABLE_OPTIMISTIC_TXN_COUNT"`
	RepeatableReadPessimisticTxnCount  int64                  `spanner:"REPEATABLE_READ_PESSIMISTIC_TXN_COUNT"`
}

// LockStatsTop represents SPANNER_SYS.LOCK_STATS_TOP_{MINUTE,10MINUTE,HOUR}.
type LockStatsTop struct {
	IntervalEnd                  time.Time            `spanner:"INTERVAL_END"`
	RowRangeStartKey             []byte               `spanner:"ROW_RANGE_START_KEY"`
	LockWaitSeconds              float64              `spanner:"LOCK_WAIT_SECONDS"`
	SampleLockRequests           []*LockSampleRequest `spanner:"SAMPLE_LOCK_REQUESTS"`
	SampleLockRequestsJSONString *string              `spanner:"SAMPLE_LOCK_REQUESTS_JSON_STRING"`
}

// LockStatsTotal represents SPANNER_SYS.LOCK_STATS_TOTAL_{MINUTE,10MINUTE,HOUR}.
type LockStatsTotal struct {
	IntervalEnd          time.Time `spanner:"INTERVAL_END"`
	TotalLockWaitSeconds float64   `spanner:"TOTAL_LOCK_WAIT_SECONDS"`
}

// ColumnOperationsStats represents SPANNER_SYS.COLUMN_OPERATIONS_STATS_{MINUTE,10MINUTE,HOUR}.
type ColumnOperationsStats struct {
	IntervalEnd              time.Time `spanner:"INTERVAL_END"`
	TableName                string    `spanner:"TABLE_NAME"`
	ColumnName               string    `spanner:"COLUMN_NAME"`
	QueryCount               int64     `spanner:"QUERY_COUNT"`
	ReadCount                int64     `spanner:"READ_COUNT"`
	WriteCount               int64     `spanner:"WRITE_COUNT"`
	IsQueryCacheMemoryCapped bool      `spanner:"IS_QUERY_CACHE_MEMORY_CAPPED"`
}

// TableOperationsStats represents SPANNER_SYS.TABLE_OPERATIONS_STATS_{MINUTE,10MINUTE,HOUR}.
type TableOperationsStats struct {
	IntervalEnd    time.Time `spanner:"INTERVAL_END"`
	TableName      string    `spanner:"TABLE_NAME"`
	ReadQueryCount int64     `spanner:"READ_QUERY_COUNT"`
	WriteCount     int64     `spanner:"WRITE_COUNT"`
	DeleteCount    int64     `spanner:"DELETE_COUNT"`
}

// SplitStatsTop represents SPANNER_SYS.SPLIT_STATS_TOP_{MINUTE,10MINUTE,HOUR}.
type SplitStatsTop struct {
	IntervalEnd         time.Time `spanner:"INTERVAL_END"`
	SplitStart          string    `spanner:"SPLIT_START"`
	SplitLimit          string    `spanner:"SPLIT_LIMIT"`
	AffectedTables      []string  `spanner:"AFFECTED_TABLES"`
	CPUUsageScore       int64     `spanner:"CPU_USAGE_SCORE"`
	UnsplittableReasons []string  `spanner:"UNSPLITTABLE_REASONS"`
}

// QueryProfilesTop represents SPANNER_SYS.QUERY_PROFILES_TOP_{MINUTE,10MINUTE,HOUR}.
type QueryProfilesTop struct {
	IntervalEnd     time.Time `spanner:"INTERVAL_END"`
	TextFingerprint int64     `spanner:"TEXT_FINGERPRINT"`
	LatencySeconds  float64   `spanner:"LATENCY_SECONDS"`
	QueryProfile    string    `spanner:"QUERY_PROFILE"`
}
