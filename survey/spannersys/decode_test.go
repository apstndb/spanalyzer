package spannersys

import (
	"testing"

	"cloud.google.com/go/spanner"
)

func TestRowDecodeActivePartitionedDML(t *testing.T) {
	row, err := spanner.NewRow(
		[]string{"TEXT", "TEXT_FINGERPRINT", "SESSION_ID", "NUM_PARTITIONS_TOTAL", "NUM_PARTITIONS_COMPLETE", "NUM_TRIVIAL_PARTITIONS_COMPLETE", "PROGRESS", "ROWS_PROCESSED"},
		[]any{"UPDATE T SET C = 1", "42", "session", int64(10), int64(4), int64(1), "0.4", int64(99)},
	)
	if err != nil {
		t.Fatalf("NewRow: %v", err)
	}

	var got ActivePartitionedDML
	if err := row.ToStructLenient(&got); err != nil {
		t.Fatalf("ToStructLenient: %v", err)
	}
	if got.TextFingerprint != "42" || got.Progress != "0.4" {
		t.Errorf("decoded ActivePartitionedDML = %#v", got)
	}
}

func TestRowDecodeOperationsByTable(t *testing.T) {
	type sourceOperationsByTable struct {
		Table               string `spanner:"TABLE_NAME"`
		InsertOrUpdateCount int64  `spanner:"INSERT_OR_UPDATE_COUNT"`
		InsertOrUpdateBytes int64  `spanner:"INSERT_OR_UPDATE_BYTES"`
	}
	type destination struct {
		OperationsByTable []*OperationsByTable `spanner:"OPERATIONS_BY_TABLE"`
	}

	row, err := spanner.NewRow(
		[]string{"OPERATIONS_BY_TABLE"},
		[]any{[]*sourceOperationsByTable{{Table: "Accounts", InsertOrUpdateCount: 2, InsertOrUpdateBytes: 128}}},
	)
	if err != nil {
		t.Fatalf("NewRow: %v", err)
	}

	var got destination
	if err := row.ToStruct(&got); err != nil {
		t.Fatalf("ToStruct: %v", err)
	}
	if len(got.OperationsByTable) != 1 || got.OperationsByTable[0].TableName != "Accounts" {
		t.Errorf("decoded operations = %#v", got.OperationsByTable)
	}
}

func TestRowDecodeTableSizesStatsPerLocalityGroup(t *testing.T) {
	row, err := spanner.NewRow(
		[]string{"TABLE_NAME", "LOCALITY_GROUP", "USED_BYTES", "USED_SSD_BYTES", "USED_HDD_BYTES"},
		[]any{"Accounts", "archive", 12.5, 10.0, 2.5},
	)
	if err != nil {
		t.Fatalf("NewRow: %v", err)
	}

	var got TableSizesStatsPerLocalityGroup
	if err := row.ToStructLenient(&got); err != nil {
		t.Fatalf("ToStructLenient: %v", err)
	}
	if got.UsedBytes != 12.5 {
		t.Errorf("used bytes = %v, want 12.5", got.UsedBytes)
	}
}
