package infoschem

import (
	"errors"
	"slices"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestExcludeUnqueryableColumns(t *testing.T) {
	tests := []struct {
		name          string
		probe         func(table, column string) error
		wantExcluded  []string
		wantRemaining []string
		wantErr       bool
		wantRetry     bool
	}{
		{
			name:          "all queryable",
			wantRemaining: []string{"COL_A", "COL_B"},
		},
		{
			name: "one excluded",
			probe: func(_, column string) error {
				if column == "COL_B" {
					return status.Error(codes.InvalidArgument, "Unrecognized name: COL_B")
				}
				return nil
			},
			wantExcluded:  []string{"COL_B"},
			wantRemaining: []string{"COL_A"},
			wantRetry:     true,
		},
		{
			name:          "non-InvalidArgument probe error propagates",
			probe:         func(_, _ string) error { return errors.New("transport failed") },
			wantRemaining: []string{"COL_A", "COL_B"},
			wantErr:       true,
		},
		{
			name: "closed-transaction error propagates",
			probe: func(_, _ string) error {
				return status.Error(codes.InvalidArgument, closedTransactionErr)
			},
			wantRemaining: []string{"COL_A", "COL_B"},
			wantErr:       true,
		},
		{
			name:          "nothing excluded means no retry",
			wantRemaining: []string{"COL_A", "COL_B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discovered := DiscoveredColumns{
				"COLUMN_OPTIONS": {"COL_A": true, "COL_B": true},
				"TABLES":         {"TABLE_NAME": true},
			}
			probe := tt.probe
			if probe == nil {
				probe = func(_, _ string) error { return nil }
			}
			calls := 0
			excluded, err := excludeUnqueryableColumns(discovered, "COLUMN_OPTIONS", func(table, column string) error {
				calls++
				if table != "COLUMN_OPTIONS" {
					t.Fatalf("probe table = %q, want COLUMN_OPTIONS", table)
				}
				return probe(table, column)
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("excludeUnqueryableColumns() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if calls < 1 {
					t.Fatalf("probe calls = %d, want at least 1", calls)
				}
				if excluded != nil {
					t.Errorf("excluded = %v, want nil on error", excluded)
				}
			} else if calls != 2 {
				t.Fatalf("probe calls = %d, want 2", calls)
			}
			if !slices.Equal(excluded, tt.wantExcluded) {
				t.Errorf("excluded = %v, want %v", excluded, tt.wantExcluded)
			}
			gotRetry := len(excluded) > 0 && err == nil
			if gotRetry != tt.wantRetry {
				t.Errorf("retry = %t, want %t", gotRetry, tt.wantRetry)
			}
			var remaining []string
			for name := range discovered["COLUMN_OPTIONS"] {
				remaining = append(remaining, name)
			}
			slices.Sort(remaining)
			if !slices.Equal(remaining, tt.wantRemaining) {
				t.Errorf("remaining = %v, want %v", remaining, tt.wantRemaining)
			}
			if !discovered["TABLES"]["TABLE_NAME"] {
				t.Fatal("columns of other tables must not be probed or deleted")
			}
		})
	}
}
