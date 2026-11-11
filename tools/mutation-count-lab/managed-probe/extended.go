package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	dbpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/status"
)

type commitFunc func(string, string, string, string, []string, []*spanner.Mutation, ...string)

func extendedCases(ctx context.Context, c *managedClients, r *result, commit commitFunc, largeDML bool) error {
	cols := []string{"Shard", "Id", "Active", "Note"}
	id := int64(200000)
	seed := func(table string, n int64) error {
		_, err := c.Client.Apply(ctx, []*spanner.Mutation{spanner.Insert(table, cols, []interface{}{int64(1), n, "a", "n"})})
		return err
	}
	for _, table := range []string{"Bare", "OneIndex", "TwoIndexes", "StoreIndex", "NullIndex", "CompositeIndex"} {
		for _, kind := range []string{"upsert", "replace"} {
			for _, present := range []bool{false, true} {
				for _, column := range []string{"Active", "Note"} {
					id++
					if present {
						if err := seed(table, id); err != nil {
							return err
						}
					}
					cs := []string{"Shard", "Id", column}
					vs := []interface{}{int64(1), id, "changed"}
					m := spanner.InsertOrUpdate(table, cs, vs)
					if kind == "replace" {
						m = spanner.Replace(table, cs, vs)
					}
					commit(fmt.Sprintf("partial/%s/%s/present=%v/%s", kind, table, present, column), table, kind, fmt.Sprintf("partial-present=%v:a,n", present), cs, []*spanner.Mutation{m})
				}
			}
		}
		for _, column := range []string{"Active", "Note"} {
			for _, same := range []bool{false, true} {
				id++
				if err := seed(table, id); err != nil {
					return err
				}
				v := "changed"
				if same {
					if column == "Active" {
						v = "a"
					} else {
						v = "n"
					}
				}
				commit(fmt.Sprintf("dml/update/%s/%s/same=%v", table, column, same), table, "dml_update", "present:a,n", nil, nil, fmt.Sprintf("UPDATE %s SET %s='%s' WHERE Shard=1 AND Id=%d", table, column, v, id))
			}
		}
		id++
		commit("dml/insert-full/"+table, table, "dml_insert", "absent", cols, nil, fmt.Sprintf("INSERT INTO %s(Shard,Id,Active,Note) VALUES(1,%d,'a','n')", table, id))
		id++
		if err := seed(table, id); err != nil {
			return err
		}
		commit("dml/upsert-partial/"+table, table, "dml_upsert", "present:a,n", []string{"Shard", "Id", "Note"}, nil, fmt.Sprintf("INSERT OR UPDATE INTO %s(Shard,Id,Note) VALUES(1,%d,'changed')", table, id))
		id++
		if err := seed(table, id); err != nil {
			return err
		}
		commit("dml/delete-key/"+table, table, "dml_delete", "1 existing indexable row", nil, nil, fmt.Sprintf("DELETE FROM %s WHERE Shard=1 AND Id=%d", table, id))
		id += 10
		for n := id; n < id+3; n++ {
			if err := seed(table, n); err != nil {
				return err
			}
		}
		commit("dml/delete-range3/"+table, table, "dml_delete", "3 existing indexable rows", nil, nil, fmt.Sprintf("DELETE FROM %s WHERE Shard=1 AND Id BETWEEN %d AND %d", table, id, id+2))
	}
	// Same-value upserts distinguish the written-column shape from old/new
	// index-key equality independently of the initial two-index full updates.
	for _, table := range []string{"OneIndex", "StoreIndex", "NullIndex"} {
		for _, column := range []string{"Active", "Note"} {
			id++
			if err := seed(table, id); err != nil {
				return err
			}
			v := "a"
			if column == "Note" {
				v = "n"
			}
			cs := []string{"Shard", "Id", column}
			commit("partial/upsert-same/"+table+"/"+column, table, "upsert", "present:a,n", cs, []*spanner.Mutation{spanner.InsertOrUpdate(table, cs, []interface{}{int64(1), id, v})})
		}
	}
	parentID := int64(500000)
	if _, err := c.Client.Apply(ctx, []*spanner.Mutation{spanner.Insert("ParentRows", []string{"ParentId"}, []interface{}{parentID}), spanner.Insert("ChildRows", []string{"ParentId", "Id", "Note"}, []interface{}{parentID, int64(1), "n"}), spanner.Insert("ChildRows", []string{"ParentId", "Id", "Note"}, []interface{}{parentID, int64(2), "n"}), spanner.Insert("ChildRows", []string{"ParentId", "Id", "Note"}, []interface{}{parentID, int64(3), "n"})}); err != nil {
		return err
	}
	commit("replace/interleaved-cascade", "ParentRows", "replace", "parent +3 indexed children", []string{"ParentId"}, []*spanner.Mutation{spanner.Replace("ParentRows", []string{"ParentId"}, []interface{}{parentID})})
	valuesSQL := func(start, rows int) string {
		var b strings.Builder
		b.WriteString("INSERT INTO Bare(Shard,Id,Active,Note) VALUES ")
		for n := range rows {
			if n > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(&b, "(1,%d,'a','n')", start+n)
		}
		return b.String()
	}
	if largeDML {
		commit("limit/dml-single-80004", "Bare", "dml_insert_many", "20001 new rows", nil, nil, valuesSQL(300000, 20001))
		commit("limit/dml-two-statements-120000", "Bare", "dml_insert_many", "30000 new rows; two statements of 60000 cells", nil, nil, valuesSQL(400000, 15000), valuesSQL(420000, 15000))
	} else {
		for _, id := range []string{"limit/dml-single-80004", "limit/dml-two-statements-120000"} {
			r.Observations = append(r.Observations, observation{SchemaPhase: len(r.SchemaPhases) - 1, ID: id, Status: "NOT_RUN", Error: "large literal DML boundary controls omitted after an interrupted attempt; no runtime conclusion"})
		}
	}
	return queueCases(ctx, c, r, commit)
}

func queueCases(ctx context.Context, c *managedClients, r *result, commit commitFunc) error {
	for _, ddl := range []string{
		"CREATE QUEUE Messages(Id STRING(36) NOT NULL,Payload BYTES(MAX) NOT NULL) PRIMARY KEY(Id)",
		"CREATE QUEUE TextMessages(Id STRING(36) NOT NULL,Payload STRING(MAX) NOT NULL) PRIMARY KEY(Id)",
		"CREATE QUEUE JsonMessages(Id STRING(36) NOT NULL,Payload JSON NOT NULL) PRIMARY KEY(Id)",
		"CREATE QUEUE ReceiveMessages(Id STRING(36) NOT NULL,Payload BYTES(MAX) NOT NULL) PRIMARY KEY(Id)",
		"CREATE QUEUE ChildMessages(ParentId INT64 NOT NULL,Id STRING(36) NOT NULL,Payload BYTES(MAX) NOT NULL) PRIMARY KEY(ParentId,Id),INTERLEAVE IN PARENT ParentRows ON DELETE CASCADE",
	} {
		x := extraSchema{DDL: ddl}
		op, err := c.DatabaseClient.UpdateDatabaseDdl(ctx, &dbpb.UpdateDatabaseDdlRequest{Database: c.DatabasePath(), Statements: []string{ddl}})
		if err == nil {
			err = op.Wait(ctx)
		}
		x.Status = status.Code(err).String()
		if err != nil {
			x.Error = redact(err.Error())
		}
		r.Extra = append(r.Extra, x)
		if err != nil {
			return fmt.Errorf("queue fixture schema: %s", status.Code(err))
		}
	}
	if err := captureSchemaPhase(ctx, c, r); err != nil {
		return err
	}
	future := time.Now().UTC().Add(24 * time.Hour)
	for _, tc := range []struct {
		queue   string
		payload interface{}
	}{{"Messages", []byte("x")}, {"TextMessages", "x"}, {"JsonMessages", spanner.NullJSON{Value: map[string]interface{}{"x": 1}, Valid: true}}} {
		for _, explicit := range []bool{false, true} {
			key := fmt.Sprintf("send-%v", explicit)
			var options []spanner.SendOption
			if explicit {
				options = append(options, spanner.WithDeliveryTime(future))
			}
			commit(fmt.Sprintf("queue/send/%s/explicit=%v", tc.queue, explicit), tc.queue, "send", "absent", nil, []*spanner.Mutation{spanner.Send(tc.queue, spanner.Key{key}, tc.payload, options...)})
		}
	}
	for _, kind := range []string{"insert", "update", "upsert", "replace", "delete", "ack"} {
		key := "wire-" + kind
		present := kind != "insert"
		if present {
			if _, err := c.Client.Apply(ctx, []*spanner.Mutation{spanner.Send("Messages", spanner.Key{key}, []byte("old"), spanner.WithDeliveryTime(future))}); err != nil {
				return err
			}
		}
		cols := []string{"Id", "Payload", "DeliverTime"}
		vs := []interface{}{key, []byte("new"), future}
		var m *spanner.Mutation
		switch kind {
		case "insert":
			m = spanner.Insert("Messages", cols, vs)
		case "update":
			m = spanner.Update("Messages", cols[:2], vs[:2])
		case "upsert":
			m = spanner.InsertOrUpdate("Messages", cols, vs)
		case "replace":
			m = spanner.Replace("Messages", cols, vs)
		case "delete":
			m = spanner.Delete("Messages", spanner.Key{key})
		case "ack":
			m = spanner.Ack("Messages", spanner.Key{key})
		}
		commit("queue/wire/"+kind, "Messages", kind, fmt.Sprintf("queue-present=%v", present), cols, []*spanner.Mutation{m})
	}
	commit("queue/ack/missing", "Messages", "ack", "absent", nil, []*spanner.Mutation{spanner.Ack("Messages", spanner.Key{"missing"})})
	for _, kind := range []string{"insert", "update", "upsert", "replace", "delete"} {
		key := "dml-" + kind
		if kind != "insert" {
			if _, err := c.Client.Apply(ctx, []*spanner.Mutation{spanner.Send("Messages", spanner.Key{key}, []byte("old"), spanner.WithDeliveryTime(future))}); err != nil {
				return err
			}
		}
		var sql string
		switch kind {
		case "insert":
			sql = "INSERT INTO Messages(Id,Payload,DeliverTime) VALUES('" + key + "',b'new',TIMESTAMP_ADD(CURRENT_TIMESTAMP(),INTERVAL 1 DAY))"
		case "update":
			sql = "UPDATE Messages SET Payload=b'new' WHERE Id='" + key + "'"
		case "upsert":
			sql = "INSERT OR UPDATE INTO Messages(Id,Payload,DeliverTime) VALUES('" + key + "',b'new',TIMESTAMP_ADD(CURRENT_TIMESTAMP(),INTERVAL 1 DAY))"
		case "replace":
			sql = "INSERT OR REPLACE INTO Messages(Id,Payload,DeliverTime) VALUES('" + key + "',b'new',TIMESTAMP_ADD(CURRENT_TIMESTAMP(),INTERVAL 1 DAY))"
		case "delete":
			sql = "DELETE FROM Messages WHERE Id='" + key + "'"
		}
		commit("queue/dml/"+kind, "Messages", "dml_"+kind, "queue row fixture", nil, nil, sql)
	}
	// Omitting DeliverTime is separately observed rather than assigning its
	// default an ordinary column price before confirming queue accounting.
	commit("queue/dml/insert-no-delivery", "Messages", "dml_insert", "absent", nil, nil, "INSERT INTO Messages(Id,Payload) VALUES('dml-default',b'x')")
	commit("queue/wire/insert-no-delivery", "Messages", "insert", "absent", nil, []*spanner.Mutation{spanner.Insert("Messages", []string{"Id", "Payload"}, []interface{}{"wire-default", []byte("x")})})
	if _, err := c.Client.Apply(ctx, []*spanner.Mutation{spanner.Insert("ParentRows", []string{"ParentId"}, []interface{}{int64(500001)}), spanner.Send("ChildMessages", spanner.Key{int64(500001), "child"}, []byte("x"), spanner.WithDeliveryTime(future))}); err != nil {
		return err
	}
	commit("queue/interleaved-parent-delete", "ParentRows", "delete", "parent +1 queued child", nil, []*spanner.Mutation{spanner.Delete("ParentRows", spanner.Key{int64(500001)})})
	if _, err := c.Client.Apply(ctx, []*spanner.Mutation{spanner.Send("ReceiveMessages", spanner.Key{"receive"}, []byte("x"))}); err != nil {
		return err
	}
	token := ""
	for _, tc := range []struct{ id, sql string }{{"receive", "SELECT SpannerLeaseToken FROM RECEIVE_ReceiveMessages(max_duration=>'2s',max_batch_size=>1)"}, {"renew", "SELECT SpannerNewLeaseToken FROM RENEWLEASE_ReceiveMessages(lease_tokens=>[@token])"}} {
		cx, cancel := context.WithTimeout(ctx, 15*time.Second)
		q := spanner.NewStatement(tc.sql)
		if tc.id == "renew" {
			q.Params = map[string]interface{}{"token": token}
		}
		it := c.Client.Single().Query(cx, q)
		rows := 0
		var last error
		for {
			row, err := it.Next()
			if err != nil {
				if err != iterator.Done {
					last = err
				}
				break
			}
			rows++
			if tc.id == "receive" {
				if err := row.Column(0, &token); err != nil {
					last = err
				}
				break
			}
		}
		it.Stop()
		cancel()
		r.QueueReads = append(r.QueueReads, queueRead{ID: tc.id, SQL: tc.sql, Status: status.Code(last).String(), Rows: rows, StatsScope: "no Commit RPC; not an observed zero mutation count"})
		if last != nil {
			return fmt.Errorf("queue %s: %s", tc.id, status.Code(last))
		}
	}
	commit("queue/ack/received", "ReceiveMessages", "ack", "received row", nil, []*spanner.Mutation{spanner.Ack("ReceiveMessages", spanner.Key{"receive"})})
	return nil
}
