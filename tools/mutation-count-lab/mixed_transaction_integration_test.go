//go:build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/tools/internal/runtimepins"
	"github.com/apstndb/spanemuboost"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

// This opt-in probe always owns its local container; it never attaches to an
// endpoint or uses managed credentials. The runtime owns the random database.
func TestIntegrationMixedMutationPrimaryKeys(t *testing.T) {
	backend := os.Getenv("SPANALYZER_MIXED_MUTATION_BACKEND")
	if backend == "" {
		t.Skip("set SPANALYZER_MIXED_MUTATION_BACKEND=omni or emulator")
	}
	if backend != "omni" && backend != "emulator" {
		t.Fatal("unsupported local backend", backend)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	root, err := runtimepins.FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	image, err := runtimepins.ImageForHost(root, backend)
	if err != nil {
		t.Fatal(err)
	}
	rt, err := spanemuboost.Run(ctx, spanemuboost.Backend(backend), spanemuboost.WithContainerImage(image))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := rt.Close(); err != nil {
			t.Error(err)
		}
	}()
	t.Logf("MIXED_RUNTIME backend=%s image=%s started=%s", backend, image, time.Now().UTC().Format(time.RFC3339))
	ddls := []string{
		"CREATE TABLE Events(Shard INT64 NOT NULL,Id INT64 NOT NULL,Note STRING(MAX)) PRIMARY KEY(Shard,Id)",
		"CREATE TABLE Indexed(Shard INT64 NOT NULL,Id INT64 NOT NULL,Active STRING(MAX)) PRIMARY KEY(Shard,Id)",
		"CREATE UNIQUE INDEX ByActive ON Indexed(Active)",
	}
	type captureKey struct{}
	var mu sync.Mutex
	captured := map[string][]json.RawMessage{}
	interceptor := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if id, ok := ctx.Value(captureKey{}).(string); ok {
			if commit, ok := req.(*spannerpb.CommitRequest); ok {
				var mutations []json.RawMessage
				for _, m := range commit.Mutations {
					b, err := protojson.Marshal(m)
					if err != nil {
						return err
					}
					mutations = append(mutations, b)
				}
				mu.Lock()
				captured[id] = mutations
				mu.Unlock()
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
	// Run already bootstraps the Emulator instance; Omni owns its fixed
	// instance too. Reopening creates only this fixture's random database.
	clients, err := spanemuboost.OpenClients(ctx, rt, spanemuboost.EnableDatabaseAutoConfigOnly(), spanemuboost.WithRandomDatabaseID(), spanemuboost.SkipSchemaTeardown(), spanemuboost.WithSetupDDLs(ddls), spanemuboost.WithClientOptionsForClient(option.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(interceptor))))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := clients.Close(); err != nil {
			t.Error(err)
		}
	}()
	a, err := spanalyzer.NewMutationCountAnalyzer(ddls[0] + ";" + ddls[1] + ";" + ddls[2] + ";")
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"feedback", "update", "upsert", "replace", "delete", "dml-insert", "dml-delete", "same-key", "parameter", "partial-key", "unique-key-reuse"} {
		t.Run(name, func(t *testing.T) {
			shard := int64(i + 1)
			table := "Events"
			cols := []string{"Shard", "Id", "Note"}
			seed := []*spanner.Mutation{spanner.Insert(table, cols, []interface{}{shard, int64(2), "old"})}
			statement := spanner.NewStatement(fmt.Sprintf("UPDATE Events SET Note='n' WHERE Shard=%d AND Id=2", shard))
			writes := []*spanner.Mutation{spanner.Insert(table, cols[:2], []interface{}{shard, int64(3)})}
			wantKeys := []int64{2, 3}
			unknown := false
			switch name {
			case "update", "replace", "delete":
				seed = append(seed, spanner.Insert(table, cols, []interface{}{shard, int64(3), "old"}))
				switch name {
				case "update":
					writes[0] = spanner.Update(table, cols, []interface{}{shard, int64(3), "m"})
				case "replace":
					writes[0] = spanner.Replace(table, cols, []interface{}{shard, int64(3), "m"})
				case "delete":
					writes[0] = spanner.Delete(table, spanner.Key{shard, int64(3)})
					wantKeys = []int64{2}
				}
			case "upsert":
				writes[0] = spanner.InsertOrUpdate(table, cols, []interface{}{shard, int64(3), "m"})
			case "dml-insert":
				seed = nil
				statement = spanner.NewStatement(fmt.Sprintf("INSERT INTO Events(Shard,Id) VALUES(%d,2)", shard))
			case "dml-delete":
				statement = spanner.NewStatement(fmt.Sprintf("DELETE FROM Events WHERE Shard=%d AND Id=2", shard))
				wantKeys = []int64{3}
			case "same-key":
				writes[0] = spanner.Update(table, cols, []interface{}{shard, int64(2), "m"})
				wantKeys, unknown = []int64{2}, true
			case "parameter":
				statement = spanner.Statement{SQL: "UPDATE Events SET Note='n' WHERE Shard=@s AND Id=@id", Params: map[string]interface{}{"s": shard, "id": int64(2)}}
				unknown = true
			case "partial-key":
				// Other shards already exist; restrict the runtime by an
				// expression the static equality proof does not solve.
				statement = spanner.NewStatement(fmt.Sprintf("UPDATE Events SET Note='n' WHERE Shard+0=%d AND Id=2", shard))
				unknown = true
			case "unique-key-reuse":
				table = "Indexed"
				cols = []string{"Shard", "Id", "Active"}
				seed = []*spanner.Mutation{spanner.Insert(table, cols, []interface{}{shard, int64(2), "a"})}
				statement = spanner.NewStatement(fmt.Sprintf("UPDATE Indexed SET Active='b' WHERE Shard=%d AND Id=2", shard))
				writes[0] = spanner.Insert(table, cols, []interface{}{shard, int64(3), "a"})
			}
			if len(seed) > 0 {
				if _, err := clients.Client.Apply(ctx, seed); err != nil {
					t.Fatal(err)
				}
			}
			var rows int64
			resp, err := clients.Client.ReadWriteTransactionWithOptions(context.WithValue(ctx, captureKey{}, name), func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
				var err error
				rows, err = tx.Update(ctx, statement)
				if err != nil {
					return err
				}
				return tx.BufferWrite(writes)
			}, spanner.TransactionOptions{CommitOptions: spanner.CommitOptions{ReturnCommitStats: true}})
			if err != nil || rows != 1 {
				t.Fatal("mixed commit", rows, err)
			}
			mu.Lock()
			wire, err := json.Marshal(captured[name])
			mu.Unlock()
			if err != nil {
				t.Fatal(err)
			}
			r, err := a.AnalyzeTransaction(spanalyzer.MutationCountTransaction{Statements: []string{statement.SQL}, Mutations: wire}, spanalyzer.MutationCountOptions{Facts: []spanalyzer.MutationCountRowFacts{{Rows: &rows}, {}}})
			if err != nil || (r.Effective.Upper == nil) != unknown {
				t.Fatal("static bound", r, err)
			}
			var count *int64
			if resp.CommitStats != nil {
				count = &resp.CommitStats.MutationCount
				if r.Effective.Upper != nil && *count > *r.Effective.Upper {
					t.Fatal("observation exceeds upper bound", *count, r.Effective)
				}
				if name == "feedback" && *count != 5 {
					t.Fatal("feedback control count", *count)
				}
			} else if backend == "omni" {
				t.Fatal("Omni did not return requested CommitStats")
			}
			var keys []int64
			it := clients.Client.Single().Query(ctx, spanner.Statement{SQL: "SELECT Id FROM " + table + " WHERE Shard=@s ORDER BY Id", Params: map[string]interface{}{"s": shard}})
			defer it.Stop()
			for {
				row, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				var id int64
				if err := row.Column(0, &id); err != nil {
					t.Fatal(err)
				}
				keys = append(keys, id)
			}
			if !slices.Equal(keys, wantKeys) {
				t.Fatal("committed keys", keys, "want", wantKeys)
			}
			b, err := json.Marshal(struct {
				Case     string                           `json:"case"`
				Rows     int64                            `json:"dml_rows"`
				Count    *int64                           `json:"mutation_count"`
				Interval spanalyzer.MutationCountInterval `json:"effective"`
				Keys     []int64                          `json:"final_ids"`
			}{name, rows, count, r.Effective, keys})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("MIXED_RESULT %s", b)
		})
	}
}
