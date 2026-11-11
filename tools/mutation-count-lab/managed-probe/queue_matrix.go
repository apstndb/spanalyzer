package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func queueMatrixDDL() []string {
	return []string{
		"CREATE PROTO BUNDLE (`examples.shipping.Order`,`examples.shipping.Order.Address`,`examples.shipping.Order.Item`)",
		"CREATE TABLE PlainRows(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id)",
		"CREATE TABLE IndexedRows(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id)",
		"CREATE INDEX ByValue ON IndexedRows(V)",
		"CREATE TABLE ParentRows(ParentId INT64 NOT NULL,Note STRING(MAX)) PRIMARY KEY(ParentId)",
		"CREATE TABLE ChildRows(ParentId INT64 NOT NULL,ChildId INT64 NOT NULL,Note STRING(MAX)) PRIMARY KEY(ParentId,ChildId),INTERLEAVE IN PARENT ParentRows ON DELETE CASCADE",
		"CREATE INDEX ChildNote ON ChildRows(Note)",
		"CREATE QUEUE Messages(Id STRING(MAX) NOT NULL,Payload BYTES(MAX) NOT NULL) PRIMARY KEY(Id)",
		"CREATE QUEUE TwoKeyMessages(Shard INT64 NOT NULL,Id STRING(MAX) NOT NULL,Payload BYTES(MAX) NOT NULL) PRIMARY KEY(Shard,Id)",
		"CREATE QUEUE ThreeKeyMessages(Shard INT64 NOT NULL,Bucket INT64 NOT NULL,Id STRING(MAX) NOT NULL,Payload BYTES(MAX) NOT NULL) PRIMARY KEY(Shard,Bucket,Id)",
		"CREATE QUEUE TextMessages(Id STRING(MAX) NOT NULL,Payload STRING(MAX) NOT NULL) PRIMARY KEY(Id)",
		"CREATE QUEUE JsonMessages(Id STRING(MAX) NOT NULL,Payload JSON NOT NULL) PRIMARY KEY(Id)",
		"CREATE QUEUE ProtoMessages(Id STRING(MAX) NOT NULL,Payload `examples.shipping.Order` NOT NULL) PRIMARY KEY(Id)",
		"CREATE QUEUE ChildMessages(ParentId INT64 NOT NULL,Id STRING(MAX) NOT NULL,Payload BYTES(MAX) NOT NULL) PRIMARY KEY(ParentId,Id),INTERLEAVE IN PARENT ParentRows ON DELETE CASCADE",
		"CREATE QUEUE DeepMessages(ParentId INT64 NOT NULL,ChildId INT64 NOT NULL,Id STRING(MAX) NOT NULL,Payload BYTES(MAX) NOT NULL) PRIMARY KEY(ParentId,ChildId,Id),INTERLEAVE IN PARENT ChildRows ON DELETE CASCADE",
	}
}

func queueProtoPayload(b []byte, s string) (proto.Message, error) {
	var set descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(b, &set); err != nil {
		return nil, err
	}
	files, err := protodesc.NewFiles(&set)
	if err != nil {
		return nil, err
	}
	d, err := files.FindDescriptorByName("examples.shipping.Order")
	if err != nil {
		return nil, err
	}
	m := dynamicpb.NewMessage(d.(protoreflect.MessageDescriptor))
	if s != "" {
		m.Set(m.Descriptor().Fields().ByName("order_number"), protoreflect.ValueOfString(s))
	}
	return m, nil
}

type queueMatrixPlanCase struct {
	ID       string `json:"id"`
	Table    string `json:"table"`
	Kind     string `json:"kind"`
	State    string `json:"before_state"`
	Expected string `json:"expected"`
}

type queueFixture struct {
	name, payload string
	keys          []string
}

func queueFixtures() []queueFixture {
	return []queueFixture{
		{"Messages", "bytes", []string{"Id"}},
		{"TwoKeyMessages", "bytes", []string{"Shard", "Id"}},
		{"ThreeKeyMessages", "bytes", []string{"Shard", "Bucket", "Id"}},
		{"TextMessages", "string", []string{"Id"}},
		{"JsonMessages", "json", []string{"Id"}},
		{"ProtoMessages", "proto", []string{"Id"}},
		{"ChildMessages", "bytes", []string{"ParentId", "Id"}},
		{"DeepMessages", "bytes", []string{"ParentId", "ChildId", "Id"}},
	}
}

type queueMatrixRunner struct {
	ctx         context.Context
	c           *managedClients
	r           *result
	commit      commitFunc
	record      func(observation)
	descriptors []byte
	future      time.Time
	next        int64
	plan        []queueMatrixPlanCase
	expected    map[string]string
}

func (q *queueMatrixRunner) key(f queueFixture) spanner.Key {
	q.next++
	k := make(spanner.Key, len(f.keys))
	for n := range k {
		k[n] = int64(1)
	}
	k[len(k)-1] = fmt.Sprintf("m-%07d", q.next)
	return k
}

func (q *queueMatrixRunner) payload(f queueFixture, size int) interface{} {
	s := strings.Repeat("x", size)
	switch f.payload {
	case "string":
		return s
	case "json":
		return spanner.NullJSON{Value: map[string]interface{}{"x": s}, Valid: true}
	case "proto":
		m, err := queueProtoPayload(q.descriptors, s)
		if err != nil {
			panic(err)
		}
		return m
	default:
		return []byte(s)
	}
}

func (q *queueMatrixRunner) seed(ms ...*spanner.Mutation) error {
	if q.c == nil || len(ms) == 0 {
		return nil
	}
	cx, cancel := context.WithTimeout(q.ctx, 45*time.Second)
	defer cancel()
	_, err := q.c.Client.Apply(cx, ms)
	return err
}

func (q *queueMatrixRunner) seedMessages(f queueFixture, keys []spanner.Key) error {
	var ms []*spanner.Mutation
	for _, k := range keys {
		ms = append(ms, spanner.Send(f.name, k, q.payload(f, 1), spanner.WithDeliveryTime(q.future)))
	}
	return q.seed(ms...)
}

func (q *queueMatrixRunner) run(id, table, kind, state, expected string, ms []*spanner.Mutation, sqls ...string) {
	q.plan = append(q.plan, queueMatrixPlanCase{id, table, kind, state, expected})
	q.expected[id] = expected
	if q.c != nil {
		q.commit(id, table, kind, state, nil, ms, sqls...)
	}
}

func matrixKeySQL(k spanner.Key) string {
	var parts []string
	for _, v := range k {
		switch v := v.(type) {
		case int64:
			parts = append(parts, fmt.Sprint(v))
		case string:
			parts = append(parts, "'"+v+"'")
		}
	}
	return strings.Join(parts, ",")
}

func (q *queueMatrixRunner) payloadSQL(f queueFixture) string {
	switch f.payload {
	case "string":
		return "'x'"
	case "json":
		return `JSON '{"x":"x"}'`
	case "proto":
		m, _ := queueProtoPayload(q.descriptors, "x")
		b, _ := proto.Marshal(m)
		return "CAST(FROM_BASE64('" + base64.StdEncoding.EncodeToString(b) + "') AS `examples.shipping.Order`)"
	default:
		return "b'x'"
	}
}

func (q *queueMatrixRunner) insertSQL(f queueFixture, keys []spanner.Key, delivery bool) string {
	cols := append(append([]string(nil), f.keys...), "Payload")
	if delivery {
		cols = append(cols, "DeliverTime")
	}
	var rows []string
	for _, k := range keys {
		s := matrixKeySQL(k) + "," + q.payloadSQL(f)
		if delivery {
			s += ",TIMESTAMP '" + q.future.Format(time.RFC3339Nano) + "'"
		}
		rows = append(rows, "("+s+")")
	}
	return "INSERT INTO " + f.name + "(" + strings.Join(cols, ",") + ") VALUES " + strings.Join(rows, ",")
}

func queueKeyPredicate(f queueFixture, k spanner.Key) string {
	var terms []string
	for n, c := range f.keys {
		terms = append(terms, c+"="+matrixKeySQL(spanner.Key{k[n]}))
	}
	return strings.Join(terms, " AND ")
}

func (q *queueMatrixRunner) regular() error {
	if err := q.seed(spanner.Insert("ParentRows", []string{"ParentId", "Note"}, []interface{}{int64(1), "n"}), spanner.Insert("ChildRows", []string{"ParentId", "ChildId", "Note"}, []interface{}{int64(1), int64(1), "n"})); err != nil {
		return err
	}
	for _, f := range queueFixtures() {
		for _, size := range []int{0, 1, 8192} {
			for _, delivery := range []string{"default", "past", "future"} {
				var opts []spanner.SendOption
				if delivery != "default" {
					t := q.future
					if delivery == "past" {
						t = t.Add(-48 * time.Hour)
					}
					opts = []spanner.SendOption{spanner.WithDeliveryTime(t)}
				}
				q.run(fmt.Sprintf("send/%s/bytes=%d/delivery=%s", f.name, size, delivery), f.name, "send", "absent", "OK", []*spanner.Mutation{spanner.Send(f.name, q.key(f), q.payload(f, size), opts...)})
			}
		}
		for _, same := range []bool{true, false} {
			k := q.key(f)
			if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
				return err
			}
			size := 1
			if !same {
				size = 2
			}
			q.run(fmt.Sprintf("send/duplicate/%s/same=%v", f.name, same), f.name, "send", "existing message", "reject", []*spanner.Mutation{spanner.Send(f.name, k, q.payload(f, size), spanner.WithDeliveryTime(q.future))})
		}
		for _, present := range []bool{false, true} {
			for _, ignore := range []bool{false, true} {
				k := q.key(f)
				if present {
					if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
						return err
					}
				}
				expected := "OK"
				if !present && !ignore {
					expected = "reject"
				}
				q.run(fmt.Sprintf("ack/%s/present=%v/ignore=%v", f.name, present, ignore), f.name, "ack", fmt.Sprintf("queue-present=%v", present), expected, []*spanner.Mutation{spanner.Ack(f.name, k, spanner.WithIgnoreNotFound(ignore))})
			}
		}
		for _, n := range []int{2, 10} {
			var sends, acks []*spanner.Mutation
			for range n {
				k := q.key(f)
				sends = append(sends, spanner.Send(f.name, k, q.payload(f, 1), spanner.WithDeliveryTime(q.future)))
			}
			q.run(fmt.Sprintf("batch/send/%s/n=%d", f.name, n), f.name, "send", "all absent", "OK", sends)
			// Independent seed keys prevent a failed preceding Send control from
			// turning the Ack observation into an unintended missing-row test.
			var keys []spanner.Key
			for range n {
				keys = append(keys, q.key(f))
			}
			if err := q.seedMessages(f, keys); err != nil {
				return err
			}
			for _, k := range keys {
				acks = append(acks, spanner.Ack(f.name, k))
			}
			q.run(fmt.Sprintf("batch/ack/%s/n=%d", f.name, n), f.name, "ack", fmt.Sprintf("%d existing messages", n), "OK", acks)
		}
		for _, n := range []int{1, 2, 10} {
			for _, delivery := range []bool{false, true} {
				var keys []spanner.Key
				for range n {
					keys = append(keys, q.key(f))
				}
				q.run(fmt.Sprintf("dml/insert/%s/n=%d/delivery=%v", f.name, n, delivery), f.name, "dml_insert", "all absent", "OK", nil, q.insertSQL(f, keys, delivery))
			}
		}
		for _, n := range []int{0, 1, 3, 10} {
			var keys []spanner.Key
			var where []string
			for range n {
				keys = append(keys, q.key(f))
			}
			if err := q.seedMessages(f, keys); err != nil {
				return err
			}
			for _, k := range keys {
				where = append(where, "("+queueKeyPredicate(f, k)+")")
			}
			if n == 0 {
				where = []string{queueKeyPredicate(f, q.key(f))}
			}
			q.run(fmt.Sprintf("dml/delete/%s/n=%d", f.name, n), f.name, "dml_delete", fmt.Sprintf("%d matched messages", n), "OK", nil, "DELETE FROM "+f.name+" WHERE "+strings.Join(where, " OR "))
		}
	}
	f := queueFixtures()[0]
	q.run("send/bytes-65536", f.name, "send", "absent", "OK", []*spanner.Mutation{spanner.Send(f.name, q.key(f), q.payload(f, 65536), spanner.WithDeliveryTime(q.future))})
	k := q.key(f)
	k[0] = strings.Repeat("k", 1024)
	q.run("send/key-1024-bytes", f.name, "send", "absent", "OK", []*spanner.Mutation{spanner.Send(f.name, k, []byte("x"), spanner.WithDeliveryTime(q.future))})
	q.run("send/json-null", "JsonMessages", "send", "absent", "OK", []*spanner.Mutation{spanner.Send("JsonMessages", spanner.Key{"json-null"}, spanner.NullJSON{Value: nil, Valid: true}, spanner.WithDeliveryTime(q.future))})
	for _, payload := range []interface{}{nil, int64(1)} {
		q.run(fmt.Sprintf("send/invalid-payload/%T", payload), f.name, "send", "absent", "reject", []*spanner.Mutation{spanner.Send(f.name, q.key(f), payload)})
	}
	for n, k := range []spanner.Key{{}, {nil}} {
		q.run(fmt.Sprintf("send/invalid-key/%d", n), f.name, "send", "no valid target key", "reject", []*spanner.Mutation{spanner.Send(f.name, k, []byte("x"))})
	}
	// INT64 and STRING both use a protobuf string scalar on the wire. The
	// queue's STRING key interprets this SDK int64 as the valid string "1".
	q.run("send/int64-string-wire-key", f.name, "send", "valid STRING wire key", "OK", []*spanner.Mutation{spanner.Send(f.name, spanner.Key{int64(1)}, []byte("x"))})
	return nil
}

func (q *queueMatrixRunner) composition() error {
	f := queueFixtures()[0]
	for _, ignore := range []bool{false, true} {
		gone := q.key(f)
		if err := q.seedMessages(f, []spanner.Key{gone}); err != nil {
			return err
		}
		if err := q.seed(spanner.Ack(f.name, gone)); err != nil {
			return err
		}
		missingWant := "reject"
		if ignore {
			missingWant = "OK"
		}
		q.run(fmt.Sprintf("composition/ack-already-acked/ignore=%v", ignore), f.name, "ack", "previously acknowledged message", missingWant, []*spanner.Mutation{spanner.Ack(f.name, gone, spanner.WithIgnoreNotFound(ignore))})
		k := q.key(f)
		if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
			return err
		}
		want := "reject"
		if ignore {
			want = "OK"
		}
		q.run(fmt.Sprintf("composition/ack-present-and-missing/ignore=%v", ignore), f.name, "ack", "one present, one missing", want, []*spanner.Mutation{spanner.Ack(f.name, k), spanner.Ack(f.name, q.key(f), spanner.WithIgnoreNotFound(ignore))})
		k = q.key(f)
		if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
			return err
		}
		q.run(fmt.Sprintf("composition/duplicate-ack/ignore=%v", ignore), f.name, "ack", "one existing message", want, []*spanner.Mutation{spanner.Ack(f.name, k), spanner.Ack(f.name, k, spanner.WithIgnoreNotFound(ignore))})
	}
	for _, present := range []bool{false, true} {
		for _, order := range []string{"send-ack", "ack-send"} {
			k := q.key(f)
			if present {
				if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
					return err
				}
			}
			send := spanner.Send(f.name, k, []byte("x"), spanner.WithDeliveryTime(q.future))
			ack := spanner.Ack(f.name, k)
			ms := []*spanner.Mutation{send, ack}
			if order == "ack-send" {
				ms = []*spanner.Mutation{ack, send}
			}
			// Composition is exploratory: duplicate/existence checks may occur
			// before or after coalescing. Record either semantic outcome explicitly.
			q.run(fmt.Sprintf("composition/%s/present=%v", order, present), f.name, "mixed_queue", fmt.Sprintf("queue-present=%v", present), "observe", ms)
		}
	}
	k := q.key(f)
	m := spanner.Send(f.name, k, []byte("x"), spanner.WithDeliveryTime(q.future))
	q.run("composition/duplicate-send", f.name, "send", "absent", "observe", []*spanner.Mutation{m, m})
	for _, table := range []string{"PlainRows", "IndexedRows"} {
		for _, n := range []int{1, 3} {
			q.next++
			ms := []*spanner.Mutation{spanner.Insert(table, []string{"Id", "V"}, []interface{}{q.next, "x"})}
			for range n {
				ms = append(ms, spanner.Send(f.name, q.key(f), []byte("x"), spanner.WithDeliveryTime(q.future)))
			}
			q.run(fmt.Sprintf("composition/%s-plus-send/n=%d", table, n), table, "mixed_tables", "new table row and messages", "OK", ms)
		}
	}
	k = q.key(f)
	q.run("composition/dml-insert-delete", f.name, "dml_mixed", "absent", "OK", nil, q.insertSQL(f, []spanner.Key{k}, true), "DELETE FROM Messages WHERE "+queueKeyPredicate(f, k))
	k = q.key(f)
	if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
		return err
	}
	q.run("composition/dml-delete-insert", f.name, "dml_mixed", "present", "OK", nil, "DELETE FROM Messages WHERE "+queueKeyPredicate(f, k), q.insertSQL(f, []spanner.Key{k}, true))
	k = q.key(f)
	q.run("composition/dml-insert-buffered-ack", f.name, "mixed_channels", "absent", "observe", []*spanner.Mutation{spanner.Ack(f.name, k)}, q.insertSQL(f, []spanner.Key{k}, true))
	q.next++
	q.run("composition/dml-table-buffered-send", f.name, "mixed_channels", "new table row and message", "OK", []*spanner.Mutation{spanner.Send(f.name, q.key(f), []byte("x"), spanner.WithDeliveryTime(q.future))}, fmt.Sprintf("INSERT INTO PlainRows(Id,V) VALUES(%d,'x')", q.next))
	return nil
}

func (q *queueMatrixRunner) deletesAndRestrictions() error {
	f := queueFixtures()[0]
	for _, mode := range []string{"UPDATE", "IGNORE", "REPLACE"} {
		sql := strings.Replace(q.insertSQL(f, []spanner.Key{q.key(f)}, true), "INSERT INTO", "INSERT OR "+mode+" INTO", 1)
		id, want := "unsupported/dml-insert-or-"+strings.ToLower(mode), "reject"
		if mode == "IGNORE" {
			id, want = "dml/insert-or-ignore-new-key", "OK"
		}
		q.run(id, f.name, "dml_insert", "absent", want, nil, sql)
	}
	for _, column := range []string{"Payload", "DeliverTime"} {
		k := q.key(f)
		if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
			return err
		}
		value := "b'x'"
		if column == "DeliverTime" {
			value = "CURRENT_TIMESTAMP()"
		}
		q.run("unsupported/dml-update-"+column, f.name, "dml_update", "present", "reject", nil, "UPDATE Messages SET "+column+"="+value+" WHERE "+queueKeyPredicate(f, k))
	}
	for _, kind := range []string{"insert", "update", "upsert", "replace", "delete"} {
		k := q.key(f)
		var ms []*spanner.Mutation
		if kind != "insert" {
			if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
				return err
			}
		}
		cols := []string{"Id", "Payload", "DeliverTime"}
		vs := []interface{}{k[0], []byte("x"), q.future}
		switch kind {
		case "insert":
			ms = []*spanner.Mutation{spanner.Insert(f.name, cols, vs)}
		case "update":
			ms = []*spanner.Mutation{spanner.Update(f.name, cols, vs)}
		case "upsert":
			ms = []*spanner.Mutation{spanner.InsertOrUpdate(f.name, cols, vs)}
		case "replace":
			ms = []*spanner.Mutation{spanner.Replace(f.name, cols, vs)}
		case "delete":
			ms = []*spanner.Mutation{spanner.Delete(f.name, k)}
		}
		q.run("unsupported/wire-"+kind, f.name, kind, "isolated key", "reject", ms)
	}
	for _, n := range []int{1, 3, 10} {
		var keys []spanner.Key
		for range n {
			keys = append(keys, q.key(f))
		}
		if err := q.seedMessages(f, keys); err != nil {
			return err
		}
		lo, hi := keys[0][0].(string), keys[n-1][0].(string)
		q.run(fmt.Sprintf("dml/delete-range/n=%d", n), f.name, "dml_delete", fmt.Sprintf("%d matched messages", n), "OK", nil, "DELETE FROM Messages WHERE Id BETWEEN '"+lo+"' AND '"+hi+"'")
	}
	for _, n := range []int{0, 1, 3, 10} {
		if q.c != nil {
			if err := q.clearMessages(); err != nil {
				return err
			}
		}
		var keys []spanner.Key
		for range n {
			keys = append(keys, q.key(f))
		}
		if err := q.seedMessages(f, keys); err != nil {
			return err
		}
		q.run(fmt.Sprintf("dml/delete-all/n=%d", n), f.name, "dml_delete", fmt.Sprintf("%d total messages in isolated queue", n), "OK", nil, "DELETE FROM Messages WHERE TRUE")
	}
	q.run("unsupported/dml-null-payload", f.name, "dml_insert", "absent", "reject", nil, "INSERT INTO Messages(Id,Payload) VALUES('null-payload',NULL)")
	q.run("unsupported/dml-null-delivery", f.name, "dml_insert", "absent", "reject", nil, "INSERT INTO Messages(Id,Payload,DeliverTime) VALUES('null-delivery',b'x',NULL)")
	return nil
}

func (q *queueMatrixRunner) cascades() error {
	for _, deep := range []bool{false, true} {
		for _, n := range []int{0, 1, 3, 10} {
			for _, method := range []string{"mutation-delete", "dml-delete", "replace"} {
				q.next++
				parent := q.next + 1000000
				ms := []*spanner.Mutation{spanner.Insert("ParentRows", []string{"ParentId", "Note"}, []interface{}{parent, "old"})}
				if deep {
					ms = append(ms, spanner.Insert("ChildRows", []string{"ParentId", "ChildId", "Note"}, []interface{}{parent, int64(1), "n"}))
				}
				queue := "ChildMessages"
				if deep {
					queue = "DeepMessages"
				}
				for c := range n {
					k := spanner.Key{parent, fmt.Sprintf("child-%d", c)}
					if deep {
						k = spanner.Key{parent, int64(1), fmt.Sprintf("child-%d", c)}
					}
					ms = append(ms, spanner.Send(queue, k, []byte("x"), spanner.WithDeliveryTime(q.future)))
				}
				if err := q.seed(ms...); err != nil {
					return err
				}
				var write []*spanner.Mutation
				var sql []string
				switch method {
				case "mutation-delete":
					write = []*spanner.Mutation{spanner.Delete("ParentRows", spanner.Key{parent})}
				case "replace":
					write = []*spanner.Mutation{spanner.Replace("ParentRows", []string{"ParentId", "Note"}, []interface{}{parent, "new"})}
				case "dml-delete":
					sql = []string{fmt.Sprintf("DELETE FROM ParentRows WHERE ParentId=%d", parent)}
				}
				q.run(fmt.Sprintf("cascade/%s/%s/n=%d", queue, method, n), "ParentRows", method, fmt.Sprintf("parent and %d queued children; deep=%v", n, deep), "OK", write, sql...)
			}
		}
	}
	return nil
}

func printQueueMatrixPlan(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	q := newQueueMatrixRunner(context.Background(), nil, &result{}, nil, nil, b)
	if err := q.writes(); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Status string                `json:"status"`
		DDL    []string              `json:"ddl"`
		Cases  []queueMatrixPlanCase `json:"cases"`
	}{"prepared_not_executed", queueMatrixDDL(), q.plan})
}

func newQueueMatrixRunner(ctx context.Context, c *managedClients, r *result, commit commitFunc, record func(observation), b []byte) *queueMatrixRunner {
	return &queueMatrixRunner{ctx: ctx, c: c, r: r, commit: commit, record: record, descriptors: b, future: time.Now().UTC().Add(24 * time.Hour), expected: map[string]string{}}
}

func (q *queueMatrixRunner) writes() error {
	for _, f := range []func() error{q.regular, q.composition, q.deletesAndRestrictions, q.cascades, q.limitControls} {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

func runQueueMatrix(ctx context.Context, c *managedClients, r *result, commit commitFunc, record func(observation), b []byte) error {
	q := newQueueMatrixRunner(ctx, c, r, commit, record, b)
	if err := q.writes(); err != nil {
		return err
	}
	return q.remaining()
}
