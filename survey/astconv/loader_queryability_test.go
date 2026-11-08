package astconv

import (
	"errors"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanalyzer/survey/infoschem"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type schemaRowStream struct {
	rows []*spanner.Row
	err  error
	next int
}

func (s *schemaRowStream) Next() (*spanner.Row, error) {
	if s.next < len(s.rows) {
		row := s.rows[s.next]
		s.next++
		return row, nil
	}
	if s.err != nil {
		return nil, s.err
	}
	return nil, iterator.Done
}

func (s *schemaRowStream) Stop() {}

func (s *schemaRowStream) Do(f func(*spanner.Row) error) error {
	for _, row := range s.rows {
		if err := f(row); err != nil {
			return err
		}
	}
	return s.err
}

func TestSchemaRowsCommitOnlyCompleteReads(t *testing.T) {
	type item struct{ Name string }
	row, err := spanner.NewRow([]string{"Name"}, []any{"new"})
	if err != nil {
		t.Fatal(err)
	}
	dest := []*item{{Name: "old"}}
	failed := status.Error(codes.InvalidArgument, "Unrecognized name: IS_INHERITED")
	if err := selectSchemaRowsAtomically(&schemaRowStream{rows: []*spanner.Row{row}, err: failed}, &dest); !errors.Is(err, failed) {
		t.Fatalf("read error = %v, want original failure", err)
	}
	if len(dest) != 1 || dest[0].Name != "old" {
		t.Fatalf("failed read changed destination: %+v", dest)
	}
	if err := selectSchemaRowsAtomically(&schemaRowStream{rows: []*spanner.Row{row}}, &dest); err != nil {
		t.Fatal(err)
	}
	if len(dest) != 1 || dest[0].Name != "new" {
		t.Fatalf("retry kept stale or partial rows: %+v", dest)
	}
}

func TestSchemaCompatibilityRetryCannotDropIdentityColumns(t *testing.T) {
	for _, table := range []string{"COLUMN_OPTIONS", "LOCALITY_GROUP_OPTIONS", "TABLES"} {
		discovered := infoschem.DiscoveredColumns{table: {"TABLE_NAME": true, "IS_INHERITED": true}}
		candidates := compatibilityFlagCandidates(table, discovered)
		identityFailure := status.Error(codes.InvalidArgument, "Unrecognized name: TABLE_NAME")
		if names := infoschem.ColumnsNamedByRejection(identityFailure, candidates[table]); len(names) != 0 {
			t.Fatalf("%s identity column became an exclusion candidate: %v", table, names)
		}
		wantFlag := table != "TABLES"
		if candidates[table]["IS_INHERITED"] != wantFlag {
			t.Fatalf("%s compatibility candidates = %v", table, candidates)
		}
		delete(candidates[table], "IS_INHERITED")
		if !discovered[table]["IS_INHERITED"] || !discovered[table]["TABLE_NAME"] {
			t.Fatal("candidate probing mutated the source discovery map")
		}
	}
}
