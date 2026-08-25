package spannersys

import (
	"strings"
	"testing"
)

func TestIntervalQueriesValidateIntervals(t *testing.T) {
	queries := map[string]func(Interval) (string, error){
		"query stats top":          QueryStatsTopQuery,
		"query stats total":        QueryStatsTotalQuery,
		"read stats top":           ReadStatsTopQuery,
		"read stats total":         ReadStatsTotalQuery,
		"transaction stats top":    TxnStatsTopQuery,
		"transaction stats total":  TxnStatsTotalQuery,
		"lock stats top":           LockStatsTopQuery,
		"lock stats total":         LockStatsTotalQuery,
		"column operations stats":  ColumnOperationsStatsQuery,
		"table operations stats":   TableOperationsStatsQuery,
		"split stats top":          SplitStatsTopQuery,
		"query profiles stats top": QueryProfilesTopQuery,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			for _, interval := range AllIntervals() {
				got, err := query(interval)
				if err != nil {
					t.Fatalf("query(%q): %v", interval, err)
				}
				if !strings.HasSuffix(got, string(interval)) {
					t.Errorf("query(%q) = %q, does not end in interval", interval, got)
				}
			}
			for _, interval := range []Interval{"", "HOUR; DROP TABLE users", "DAY"} {
				got, err := query(interval)
				if err == nil {
					t.Errorf("query(%q) = %q, want error", interval, got)
				}
			}
		})
	}
}

func TestGraphOperationExecutionStatusQuery(t *testing.T) {
	want := "SELECT * FROM SPANNER_SYS.GRAPH_OPERATION_EXECUTION_STATUS"
	if got := GraphOperationExecutionStatusQuery(); got != want {
		t.Errorf("GraphOperationExecutionStatusQuery() = %q, want %q", got, want)
	}
}

func TestVectorIndexStatsQuery(t *testing.T) {
	want := "SELECT * FROM SPANNER_SYS.VECTOR_INDEX_STATS"
	if got := VectorIndexStatsQuery(); got != want {
		t.Errorf("VectorIndexStatsQuery() = %q, want %q", got, want)
	}
}
