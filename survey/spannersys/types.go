package spannersys

// LockSampleRequest represents a STRUCT element in LOCK_STATS_TOP_*.SAMPLE_LOCK_REQUESTS.
type LockSampleRequest struct {
	Column         string `spanner:"COLUMN"`
	LockMode       string `spanner:"LOCK_MODE"`
	TransactionTag string `spanner:"TRANSACTION_TAG"`
}

// OperationsByTable represents a STRUCT element in TXN_STATS_*.OPERATIONS_BY_TABLE.
type OperationsByTable struct {
	TableName           string `spanner:"TABLE_NAME"`
	InsertOrUpdateCount int64  `spanner:"INSERT_OR_UPDATE_COUNT"`
	InsertOrUpdateBytes int64  `spanner:"INSERT_OR_UPDATE_BYTES"`
}

// LatencyDistribution represents a STRUCT element in *_STATS_*.LATENCY_DISTRIBUTION
// or TOTAL_LATENCY_DISTRIBUTION.
type LatencyDistribution struct {
	Count                 int64   `spanner:"COUNT"`
	Mean                  float64 `spanner:"MEAN"`
	SumOfSquaredDeviation float64 `spanner:"SUM_OF_SQUARED_DEVIATION"`
	NumFiniteBuckets      int64   `spanner:"NUM_FINITE_BUCKETS"`
	GrowthFactor          float64 `spanner:"GROWTH_FACTOR"`
	Scale                 float64 `spanner:"SCALE"`
	BucketCounts          []int64 `spanner:"BUCKET_COUNTS"`
}

// ClusterSizePercentile represents a STRUCT element in
// VECTOR_INDEX_METRICS_HISTORY.CLUSTER_SIZE_PERCENTILES.
type ClusterSizePercentile struct {
	Percentile        int64 `spanner:"percentile"`
	ValueAtPercentile int64 `spanner:"value_at_percentile"`
}

// ClusterDistancePercentile represents a STRUCT element in
// VECTOR_INDEX_METRICS_HISTORY.CLUSTER_AVERAGE_DISTANCE_TO_CENTROID_PERCENTILES.
type ClusterDistancePercentile struct {
	Percentile        int64   `spanner:"percentile"`
	ValueAtPercentile float64 `spanner:"value_at_percentile"`
}
