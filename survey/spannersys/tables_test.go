package spannersys

import (
	"testing"

	"cloud.google.com/go/spanner"
)

func TestTransactionAveragesDecodeNullAndZero(t *testing.T) {
	for _, nullable := range []bool{true, false} {
		value := spanner.NullFloat64{Float64: 0, Valid: !nullable}
		row, err := spanner.NewRow([]string{"AVG_PARTICIPANTS", "AVG_TOTAL_LATENCY_SECONDS"}, []any{value, value})
		if err != nil {
			t.Fatal(err)
		}
		var top TxnStatsTop
		var total TxnStatsTotal
		for _, dst := range []any{&top, &total} {
			if err := row.ToStructLenient(dst); err != nil {
				t.Fatalf("decode %T nullable=%t: %v", dst, nullable, err)
			}
		}
		for _, got := range []*float64{top.AvgParticipants, top.AvgTotalLatencySeconds, total.AvgParticipants, total.AvgTotalLatencySeconds} {
			if (got == nil) != nullable || (got != nil && *got != 0) {
				t.Fatalf("average = %v, nullable=%t", got, nullable)
			}
		}
	}
}

func TestLockDistributionDecodesStructPointers(t *testing.T) {
	row, err := spanner.NewRow([]string{"LOCK_WAIT_DISTRIBUTION"}, []any{[]*LatencyDistribution{{Count: 3, BucketCounts: []int64{1, 2}}}})
	if err != nil {
		t.Fatal(err)
	}
	var top LockStatsTop
	var total LockStatsTotal
	for _, dst := range []any{&top, &total} {
		if err := row.ToStructLenient(dst); err != nil {
			t.Fatalf("decode %T: %v", dst, err)
		}
	}
	for _, got := range [][]*LatencyDistribution{top.LockWaitDistribution, total.LockWaitDistribution} {
		if len(got) != 1 || got[0] == nil || got[0].Count != 3 || len(got[0].BucketCounts) != 2 {
			t.Fatalf("distribution = %#v", got)
		}
	}
}

func TestBatchOperationDecodesUnfinishedResult(t *testing.T) {
	row, err := spanner.NewRow([]string{"RESULT", "STATUS"}, []any{spanner.NullString{}, "RUNNING"})
	if err != nil {
		t.Fatal(err)
	}
	var got BatchOperation
	if err := row.ToStructLenient(&got); err != nil {
		t.Fatal(err)
	}
	if got.Result != nil || got.Status == nil || *got.Status != "RUNNING" {
		t.Fatalf("unfinished result = %#v", got)
	}
}
