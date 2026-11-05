package spannersys

import (
	"errors"
	"slices"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestColumnsNamedByRejection(t *testing.T) {
	selected := []string{"INTERVAL_END", "COMMIT_LATENCY_DISTRIBUTION", "PARTICIPANTS_DISTRIBUTION"}
	tests := []struct {
		name string
		err  error
		want []string
	}{
		{
			name: "not found inside alias form",
			err:  status.Error(codes.InvalidArgument, "Name COMMIT_LATENCY_DISTRIBUTION not found inside t [at 1:561]\n...t.`REPEATABLE_READ_PESSIMISTIC_TXN_COUNT`, t.`COMMIT_LATENCY_DISTRIBUTION`..."),
			want: []string{"COMMIT_LATENCY_DISTRIBUTION"},
		},
		{
			name: "unrecognized name form",
			err:  status.Error(codes.InvalidArgument, "Unrecognized name: PARTICIPANTS_DISTRIBUTION [at 1:10]"),
			want: []string{"PARTICIPANTS_DISTRIBUTION"},
		},
		{
			name: "named column was not selected",
			err:  status.Error(codes.InvalidArgument, "Name OTHER not found inside t"),
			want: nil,
		},
		{
			name: "non InvalidArgument code",
			err:  status.Error(codes.Unavailable, "Name COMMIT_LATENCY_DISTRIBUTION not found inside t"),
			want: nil,
		},
		{
			name: "closed transaction is not a rejection",
			err:  status.Error(codes.InvalidArgument, closedTransactionErr),
			want: nil,
		},
		{
			name: "plain error",
			err:  errors.New("Name COMMIT_LATENCY_DISTRIBUTION not found inside t"),
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := columnsNamedByRejection(tt.err, selected); !slices.Equal(got, tt.want) {
				t.Fatalf("columnsNamedByRejection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExcludeRejectedOrUnqueryablePrefersRejectionOverProbes(t *testing.T) {
	selected := []string{"INTERVAL_END", "COMMIT_LATENCY_DISTRIBUTION", "PARTICIPANTS_DISTRIBUTION"}
	probeCalls := 0
	kept, excluded, err := excludeRejectedOrUnqueryable(
		"TXN_STATS_TOTAL_HOUR",
		selected,
		status.Error(codes.InvalidArgument, "Name COMMIT_LATENCY_DISTRIBUTION not found inside t"),
		func(_, _ string) error { probeCalls++; return nil },
	)
	if err != nil {
		t.Fatalf("excludeRejectedOrUnqueryable() error = %v", err)
	}
	if probeCalls != 0 {
		t.Fatalf("probe calls = %d, want 0 when the rejection names a selected column", probeCalls)
	}
	if want := []string{"COMMIT_LATENCY_DISTRIBUTION"}; !slices.Equal(excluded, want) {
		t.Fatalf("excluded = %v, want %v", excluded, want)
	}
	if want := []string{"INTERVAL_END", "PARTICIPANTS_DISTRIBUTION"}; !slices.Equal(kept, want) {
		t.Fatalf("kept = %v, want %v", kept, want)
	}
}

func TestExcludeRejectedOrUnqueryableFallsBackToProbes(t *testing.T) {
	selected := []string{"INTERVAL_END", "COMMIT_LATENCY_DISTRIBUTION"}
	kept, excluded, err := excludeRejectedOrUnqueryable(
		"TXN_STATS_TOTAL_HOUR",
		selected,
		status.Error(codes.InvalidArgument, "some other rejection"),
		func(_, column string) error {
			if column == "COMMIT_LATENCY_DISTRIBUTION" {
				return status.Error(codes.InvalidArgument, "rejected by probe")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("excludeRejectedOrUnqueryable() error = %v", err)
	}
	if want := []string{"COMMIT_LATENCY_DISTRIBUTION"}; !slices.Equal(excluded, want) {
		t.Fatalf("excluded = %v, want %v", excluded, want)
	}
	if want := []string{"INTERVAL_END"}; !slices.Equal(kept, want) {
		t.Fatalf("kept = %v, want %v", kept, want)
	}
}
