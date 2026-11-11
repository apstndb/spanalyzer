package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	dbpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanalyzer/tools/internal/runtimepins"
	"github.com/apstndb/spanemuboost"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type caseKey struct{}
type observation struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`
	Table        string            `json:"table,omitempty"`
	Columns      []string          `json:"columns,omitempty"`
	State        string            `json:"before_state,omitempty"`
	Status       string            `json:"status"`
	Error        string            `json:"error,omitempty"`
	StatsPresent bool              `json:"stats_present"`
	Count        *int64            `json:"mutation_count"`
	Rows         []int64           `json:"dml_rows,omitempty"`
	Mutations    []json.RawMessage `json:"mutations,omitempty"`
	Statements   []string          `json:"statements,omitempty"`
}
type metadata struct {
	Table, Name, Type, State      string
	Unique, NullFiltered, Managed bool
	Columns                       []string
	Stored                        []string
}
type extraSchema struct{ DDL, Status, Error string }
type result struct {
	Platform      string        `json:"platform"`
	Status        string        `json:"status"`
	FatalError    string        `json:"fatal_error,omitempty"`
	Backend       string        `json:"backend"`
	Image         string        `json:"configured_image"`
	Started       time.Time     `json:"started_at"`
	Finished      time.Time     `json:"finished_at"`
	DDL           []string      `json:"ddl"`
	Extra         []extraSchema `json:"extra_schema"`
	Indexes       []metadata    `json:"indexes"`
	Observations  []observation `json:"observations"`
	CleanupErrors []string      `json:"cleanup_errors,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() (runErr error) {
	backend := flag.String("backend", "omni", "owned local backend: omni or emulator")
	metadataOnly := flag.Bool("metadata-only", false, "create the fixture and collect index metadata without writes")
	out := flag.String("out", "", "result file")
	flag.Parse()
	if *out == "" {
		return fmt.Errorf("out required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	root, err := runtimepins.FindRepositoryRoot(".")
	if err != nil {
		return err
	}
	image, err := runtimepins.ImageForHost(root, *backend)
	if err != nil {
		return err
	}
	r := result{Backend: *backend, Image: image, Platform: runtime.GOOS + "/" + runtime.GOARCH, Started: time.Now().UTC()}
	defer func() {
		if len(r.CleanupErrors) > 0 && runErr == nil {
			runErr = fmt.Errorf("local fixture cleanup failed: %v", r.CleanupErrors)
		}
		r.Finished = time.Now().UTC()
		r.Status = "complete"
		if runErr != nil {
			r.Status = "partial"
			r.FatalError = runErr.Error()
		}
		b, e := json.MarshalIndent(r, "", "  ")
		if e == nil {
			e = os.WriteFile(*out, append(b, '\n'), 0o600)
		}
		if e != nil && runErr == nil {
			runErr = e
		}
	}()
	var mu sync.Mutex
	captured := map[string][]json.RawMessage{}
	interceptor := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if q, ok := req.(*spannerpb.CommitRequest); ok {
			if id, ok := ctx.Value(caseKey{}).(string); ok {
				var ms []json.RawMessage
				for _, m := range q.Mutations {
					b, e := protojson.Marshal(m)
					if e != nil {
						return e
					}
					ms = append(ms, b)
				}
				mu.Lock()
				captured[id] = ms
				mu.Unlock()
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
	rt, err := spanemuboost.Run(ctx, spanemuboost.Backend(*backend), spanemuboost.WithContainerImage(image))
	if err != nil {
		return err
	}
	defer func() {
		if e := rt.Close(); e != nil {
			r.CleanupErrors = append(r.CleanupErrors, "runtime: "+e.Error())
		}
	}()
	for _, name := range []string{"Bare", "OneIndex", "TwoIndexes", "StoreIndex", "NullIndex", "CompositeIndex"} {
		r.DDL = append(r.DDL, fmt.Sprintf("CREATE TABLE %s (Shard INT64 NOT NULL, Id INT64 NOT NULL, Active STRING(MAX), Note STRING(MAX)) PRIMARY KEY (Shard, Id)", name))
	}
	r.DDL = append(r.DDL,
		"CREATE INDEX OneActive ON OneIndex(Active)",
		"CREATE INDEX TwoActive ON TwoIndexes(Active)", "CREATE INDEX TwoNote ON TwoIndexes(Note)",
		"CREATE INDEX StoreActive ON StoreIndex(Active) STORING(Note)",
		"CREATE NULL_FILTERED INDEX NullActive ON NullIndex(Active) STORING(Note)",
		"CREATE INDEX CompositeActiveNote ON CompositeIndex(Active, Note)",
		"CREATE TABLE DefaultsOnly (Shard INT64 NOT NULL, Id INT64 NOT NULL, Active STRING(MAX), Note STRING(MAX) DEFAULT ('default')) PRIMARY KEY (Shard,Id)",
		"CREATE TABLE GeneratedOnly (Shard INT64 NOT NULL, Id INT64 NOT NULL, Active STRING(MAX), Note STRING(MAX), G INT64 AS (Id+1) STORED) PRIMARY KEY (Shard,Id)",
		"CREATE TABLE BothAuto (Shard INT64 NOT NULL, Id INT64 NOT NULL, Active STRING(MAX), Note STRING(MAX) DEFAULT ('default'), G INT64 AS (Id+1) STORED) PRIMARY KEY (Shard,Id)",
		"CREATE TABLE Owners (OwnerId INT64 NOT NULL, Alias STRING(MAX)) PRIMARY KEY(OwnerId)",
		"CREATE TABLE ForeignEvents (Shard INT64 NOT NULL, Id INT64 NOT NULL, OwnerId INT64, Note STRING(MAX), CONSTRAINT FK_Events_Owner FOREIGN KEY(OwnerId) REFERENCES Owners(OwnerId)) PRIMARY KEY(Shard,Id)",
		"CREATE TABLE ParentRows (ParentId INT64 NOT NULL) PRIMARY KEY(ParentId)",
		"CREATE TABLE ChildRows (ParentId INT64 NOT NULL, Id INT64 NOT NULL, Note STRING(MAX)) PRIMARY KEY(ParentId,Id), INTERLEAVE IN PARENT ParentRows ON DELETE CASCADE",
		"CREATE INDEX ChildNote ON ChildRows(Note)",
	)
	clients, err := spanemuboost.OpenClients(ctx, rt, spanemuboost.EnableDatabaseAutoConfigOnly(), spanemuboost.WithRandomDatabaseID(), spanemuboost.WithSetupDDLs(r.DDL), spanemuboost.WithClientOptionsForClient(option.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(interceptor))))
	if err != nil {
		return err
	}
	defer func() {
		if e := clients.Close(); e != nil {
			r.CleanupErrors = append(r.CleanupErrors, "clients: "+e.Error())
		}
	}()
	for _, ddl := range []string{
		"CREATE TABLE SharedForeign (Shard INT64 NOT NULL, Id INT64 NOT NULL, OwnerId INT64, Note STRING(MAX), CONSTRAINT FK_Shared_A FOREIGN KEY(OwnerId) REFERENCES Owners(OwnerId), CONSTRAINT FK_Shared_B FOREIGN KEY(OwnerId) REFERENCES Owners(OwnerId)) PRIMARY KEY(Shard,Id)",
		"CREATE TABLE AliasForeign (Id INT64 NOT NULL, OwnerAlias STRING(MAX), CONSTRAINT FK_Alias FOREIGN KEY(OwnerAlias) REFERENCES Owners(Alias)) PRIMARY KEY(Id)",
		"CREATE TABLE InfoForeign (Shard INT64 NOT NULL, Id INT64 NOT NULL, OwnerId INT64, Note STRING(MAX), CONSTRAINT FK_Info FOREIGN KEY(OwnerId) REFERENCES Owners(OwnerId) NOT ENFORCED) PRIMARY KEY(Shard,Id)",
	} {
		s := extraSchema{DDL: ddl}
		op, e := clients.DatabaseClient.UpdateDatabaseDdl(ctx, &dbpb.UpdateDatabaseDdlRequest{Database: clients.DatabasePath(), Statements: []string{ddl}})
		if e == nil {
			e = op.Wait(ctx)
		}
		s.Status = status.Code(e).String()
		if e != nil {
			s.Error = status.Convert(e).Message()
		}
		r.Extra = append(r.Extra, s)
	}
	seed := func(ms ...*spanner.Mutation) error { _, e := clients.Client.Apply(ctx, ms); return e }
	commit := func(id, table, kind, before string, cols []string, ms []*spanner.Mutation, sqls ...string) {
		o := observation{ID: id, Table: table, Kind: kind, State: before, Columns: cols, Statements: sqls}
		callCtx := context.WithValue(ctx, caseKey{}, id)
		resp, e := clients.Client.ReadWriteTransactionWithOptions(callCtx, func(txctx context.Context, tx *spanner.ReadWriteTransaction) error {
			o.Rows = nil
			for _, q := range sqls {
				n, e := tx.Update(txctx, spanner.NewStatement(q))
				if e != nil {
					return e
				}
				o.Rows = append(o.Rows, n)
			}
			return tx.BufferWrite(ms)
		}, spanner.TransactionOptions{CommitOptions: spanner.CommitOptions{ReturnCommitStats: true}})
		o.Status = status.Code(e).String()
		if e != nil {
			o.Error = status.Convert(e).Message()
		}
		if resp.CommitStats != nil {
			n := resp.CommitStats.MutationCount
			o.Count = &n
			o.StatsPresent = true
		}
		mu.Lock()
		o.Mutations = captured[id]
		mu.Unlock()
		r.Observations = append(r.Observations, o)
	}
	if !*metadataOnly {
		var id int64 = 10
		cols := []string{"Shard", "Id", "Active", "Note"}
		for _, table := range []string{"Bare", "OneIndex", "TwoIndexes", "StoreIndex", "NullIndex", "CompositeIndex", "DefaultsOnly", "GeneratedOnly", "BothAuto"} {
			for _, keysOnly := range []bool{false, true} {
				id += 10
				cs := cols
				vs := []interface{}{int64(1), id, "a", "n"}
				if keysOnly {
					cs = cols[:2]
					vs = vs[:2]
				}
				commit(fmt.Sprintf("insert/%s/keys=%v", table, keysOnly), table, "insert", "absent", cs, []*spanner.Mutation{spanner.Insert(table, cs, vs)})
			}
			for _, col := range []string{"Note", "Active"} {
				for _, same := range []bool{false, true} {
					id += 10
					if e := seed(spanner.Insert(table, cols, []interface{}{int64(1), id, "a", "n"})); e != nil {
						return e
					}
					v := "changed"
					if same {
						if col == "Note" {
							v = "n"
						} else {
							v = "a"
						}
					}
					cs := []string{"Shard", "Id", col}
					commit(fmt.Sprintf("update/%s/%s/same=%v", table, col, same), table, "update", "present:a,n", cs, []*spanner.Mutation{spanner.Update(table, cs, []interface{}{int64(1), id, v})})
				}
			}
		}
		for _, operation := range []string{"upsert", "replace"} {
			for _, present := range []bool{false, true} {
				for _, keysOnly := range []bool{false, true} {
					id += 10
					if present {
						if e := seed(spanner.Insert("TwoIndexes", cols, []interface{}{int64(1), id, "old", "old"})); e != nil {
							return e
						}
					}
					cs := cols
					vs := []interface{}{int64(1), id, "new", "new"}
					if keysOnly {
						cs = cols[:2]
						vs = vs[:2]
					}
					m := spanner.InsertOrUpdate("TwoIndexes", cs, vs)
					if operation == "replace" {
						m = spanner.Replace("TwoIndexes", cs, vs)
					}
					commit(fmt.Sprintf("%s/present=%v/keys=%v", operation, present, keysOnly), "TwoIndexes", operation, fmt.Sprintf("present=%v", present), cs, []*spanner.Mutation{m})
				}
			}
		}
		for _, table := range []string{"Bare", "TwoIndexes", "StoreIndex", "NullIndex"} {
			id += 10
			commit("delete/missing/"+table, table, "delete", "absent", nil, []*spanner.Mutation{spanner.Delete(table, spanner.Key{int64(1), id})})
			id += 10
			for offset := int64(0); offset < 3; offset++ {
				if e := seed(spanner.Insert(table, cols, []interface{}{int64(1), id + offset, "a", "n"})); e != nil {
					return e
				}
			}
			commit("delete/range3/"+table, table, "delete_range", "3 existing indexable rows", nil, []*spanner.Mutation{spanner.Delete(table, spanner.KeyRange{Start: spanner.Key{int64(1), id}, End: spanner.Key{int64(1), id + 2}, Kind: spanner.ClosedClosed})})
		}
		id += 10
		if e := seed(spanner.Insert("NullIndex", cols, []interface{}{int64(1), id, nil, "n"})); e != nil {
			return e
		}
		commit("null-index/null-to-value", "NullIndex", "update", "active=NULL", []string{"Shard", "Id", "Active"}, []*spanner.Mutation{spanner.Update("NullIndex", []string{"Shard", "Id", "Active"}, []interface{}{int64(1), id, "a"})})
		id += 10
		if e := seed(spanner.Insert("NullIndex", cols, []interface{}{int64(1), id, "a", "n"})); e != nil {
			return e
		}
		commit("null-index/value-to-null", "NullIndex", "update", "active=a", []string{"Shard", "Id", "Active"}, []*spanner.Mutation{spanner.Update("NullIndex", []string{"Shard", "Id", "Active"}, []interface{}{int64(1), id, nil})})
		id += 10
		if e := seed(spanner.Insert("ParentRows", []string{"ParentId"}, []interface{}{id})); e != nil {
			return e
		}
		for c := int64(1); c <= 3; c++ {
			if e := seed(spanner.Insert("ChildRows", []string{"ParentId", "Id", "Note"}, []interface{}{id, c, "n"})); e != nil {
				return e
			}
		}
		commit("delete/interleaved-cascade", "ParentRows", "delete", "parent +3 indexed children", nil, []*spanner.Mutation{spanner.Delete("ParentRows", spanner.Key{id})})
		for _, table := range []string{"Bare", "TwoIndexes", "DefaultsOnly", "GeneratedOnly", "BothAuto"} {
			id += 10
			commit("dml/insert-keys/"+table, table, "dml_insert", "absent", cols[:2], nil, fmt.Sprintf("INSERT INTO %s (Shard,Id) VALUES(1,%d)", table, id))
		}
		id += 10
		if e := seed(spanner.Insert("OneIndex", cols, []interface{}{int64(1), id, "a", "n"})); e != nil {
			return e
		}
		commit("mutations/double-update", "OneIndex", "two_updates", "present:a,n", []string{"Shard", "Id", "Active"}, []*spanner.Mutation{spanner.Update("OneIndex", []string{"Shard", "Id", "Active"}, []interface{}{int64(1), id, "b"}), spanner.Update("OneIndex", []string{"Shard", "Id", "Active"}, []interface{}{int64(1), id, "a"})})
		id += 10
		if e := seed(spanner.Insert("OneIndex", cols, []interface{}{int64(1), id, "a", "n"})); e != nil {
			return e
		}
		commit("dml/double-update", "OneIndex", "dml_two_updates", "present:a,n", nil, nil, fmt.Sprintf("UPDATE OneIndex SET Active='b' WHERE Shard=1 AND Id=%d", id), fmt.Sprintf("UPDATE OneIndex SET Active='a' WHERE Shard=1 AND Id=%d", id))
		id += 10
		if e := seed(spanner.Insert("Owners", []string{"OwnerId", "Alias"}, []interface{}{id, fmt.Sprintf("owner%d", id)})); e != nil {
			return e
		}
		for _, table := range []string{"ForeignEvents", "SharedForeign", "InfoForeign"} {
			id2 := id + 1
			commit("fk/insert/"+table, table, "insert", "referenced owner exists", []string{"Shard", "Id", "OwnerId", "Note"}, []*spanner.Mutation{spanner.Insert(table, []string{"Shard", "Id", "OwnerId", "Note"}, []interface{}{int64(1), id2, id, "n"})})
		}
		id += 10
		commit("fk/invalid-reference", "ForeignEvents", "insert", "no referenced owner", []string{"Shard", "Id", "OwnerId", "Note"}, []*spanner.Mutation{spanner.Insert("ForeignEvents", []string{"Shard", "Id", "OwnerId", "Note"}, []interface{}{int64(1), id, int64(-999), "n"})})
		var large []*spanner.Mutation
		for n := int64(100000); n < 120001; n++ {
			large = append(large, spanner.Insert("Bare", cols, []interface{}{int64(1), n, "a", "n"}))
		}
		commit("limit/80004", "Bare", "insert_many", "20001 new rows", cols, large)
	}
	iter := clients.Client.Single().Query(ctx, spanner.NewStatement("SELECT TABLE_NAME, INDEX_NAME, INDEX_TYPE, INDEX_STATE, IS_UNIQUE, IS_NULL_FILTERED, SPANNER_IS_MANAGED FROM INFORMATION_SCHEMA.INDEXES WHERE TABLE_SCHEMA = '' AND INDEX_TYPE <> 'PRIMARY_KEY' ORDER BY TABLE_NAME,INDEX_NAME"))
	err = iter.Do(func(row *spanner.Row) error {
		var m metadata
		if e := row.Columns(&m.Table, &m.Name, &m.Type, &m.State, &m.Unique, &m.NullFiltered, &m.Managed); e != nil {
			return e
		}
		r.Indexes = append(r.Indexes, m)
		return nil
	})
	if err != nil {
		return err
	}
	for i := range r.Indexes {
		m := &r.Indexes[i]
		q := spanner.NewStatement("SELECT COLUMN_NAME, ORDINAL_POSITION FROM INFORMATION_SCHEMA.INDEX_COLUMNS WHERE TABLE_SCHEMA = '' AND TABLE_NAME=@t AND INDEX_NAME=@i ORDER BY ORDINAL_POSITION")
		q.Params = map[string]interface{}{"t": m.Table, "i": m.Name}
		err = clients.Client.Single().Query(ctx, q).Do(func(row *spanner.Row) error {
			var name string
			var ord spanner.NullInt64
			if e := row.Columns(&name, &ord); e != nil {
				return e
			}
			if ord.Valid {
				m.Columns = append(m.Columns, name)
			} else {
				m.Stored = append(m.Stored, name)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
