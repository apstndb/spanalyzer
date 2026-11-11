package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type queueOtherWrite struct {
	ID     string `json:"id"`
	API    string `json:"api"`
	SQL    string `json:"sql,omitempty"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	// PartitionedUpdate reports an estimated lower bound, not an exact row
	// count. Keep the original capture tag and qualify its meaning explicitly.
	Rows       *int64            `json:"affected_rows,omitempty"`
	Groups     []queueBatchGroup `json:"groups,omitempty"`
	StatsScope string            `json:"stats_scope"`
	Count      *int64            `json:"mutation_count"`
}
type queueBatchGroup struct {
	Indexes   []int32 `json:"indexes"`
	Status    string  `json:"status"`
	Committed bool    `json:"committed"`
	Error     string  `json:"error,omitempty"`
}

func (q *queueMatrixRunner) limitControls() error {
	f := queueFixtures()[0]
	for _, n := range []int{16000, 16001} {
		var ms []*spanner.Mutation
		for range n {
			ms = append(ms, spanner.Send(f.name, q.key(f), []byte("q"), spanner.WithDeliveryTime(q.future)))
		}
		expected := "OK"
		if n == 16001 {
			expected = "reject"
		}
		q.run(fmt.Sprintf("quota/send/n=%d", n), f.name, "send", "all absent; boundary hypothesis uses prior five-unit observation", expected, ms)
	}
	for _, n := range []int{40000, 40001} {
		var keys []spanner.Key
		var ms []*spanner.Mutation
		for range n {
			keys = append(keys, q.key(f))
		}
		for start := 0; start < len(keys); start += 5000 {
			end := min(start+5000, len(keys))
			if err := q.seedMessages(f, keys[start:end]); err != nil {
				return fmt.Errorf("ack boundary seed: %w", err)
			}
		}
		for _, k := range keys {
			ms = append(ms, spanner.Ack(f.name, k))
		}
		expected := "OK"
		if n == 40001 {
			expected = "reject"
		}
		q.run(fmt.Sprintf("quota/ack/n=%d", n), f.name, "ack", fmt.Sprintf("%d present messages; boundary hypothesis uses prior two-unit observation", n), expected, ms)
	}
	return nil
}

func (q *queueMatrixRunner) clearMessages() error {
	cx, cancel := context.WithTimeout(q.ctx, 3*time.Minute)
	defer cancel()
	// An earlier rejected boundary write can leave many seeded messages.
	// Bounded cleanup statements must not themselves exceed the DML limit.
	for {
		var keys []string
		err := q.c.Client.Single().Query(cx, spanner.NewStatement("SELECT Id FROM Messages LIMIT 1000")).Do(func(row *spanner.Row) error {
			var key string
			if err := row.Column(0, &key); err != nil {
				return err
			}
			keys = append(keys, key)
			return nil
		})
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			return nil
		}
		_, err = q.c.Client.ReadWriteTransaction(cx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
			stmt := spanner.NewStatement("DELETE FROM Messages WHERE Id IN UNNEST(@keys)")
			stmt.Params = map[string]interface{}{"keys": keys}
			_, err := tx.Update(ctx, stmt)
			return err
		})
		if err != nil {
			return err
		}
	}
}

type queueReceiver struct {
	it         *spanner.RowIterator
	cancel     context.CancelFunc
	keys       []string
	tokens     []string
	expiration time.Time
}

func (x *queueReceiver) close() { x.it.Stop(); x.cancel() }

func (q *queueMatrixRunner) receive(id string, n int) (x *queueReceiver, err error) {
	cx, cancel := context.WithTimeout(q.ctx, 30*time.Second)
	sql := fmt.Sprintf("SELECT Id,SpannerLeaseToken,SpannerLeaseExpirationTimestamp FROM RECEIVE_Messages(max_duration=>'20m',max_batch_size=>%d)", n)
	x = &queueReceiver{it: q.c.Client.Single().Query(cx, spanner.NewStatement(sql)), cancel: cancel}
	o := queueRead{ID: id, SQL: sql, StatsScope: "streaming query; no caller CommitStats"}
	defer func() {
		o.Status = status.Code(err).String()
		if err != nil {
			o.Error = redact(err.Error())
			x.close()
		}
		q.r.QueueReads = append(q.r.QueueReads, o)
	}()
	for len(x.tokens) < n {
		row, e := x.it.Next()
		if e != nil {
			return x, e
		}
		var k, t string
		var exp time.Time
		if e := row.Columns(&k, &t, &exp); e != nil {
			return x, e
		}
		x.keys = append(x.keys, k)
		x.tokens = append(x.tokens, t)
		if x.expiration.IsZero() || exp.Before(x.expiration) {
			x.expiration = exp
		}
		o.Rows++
	}
	return x, nil
}

func (q *queueMatrixRunner) renew(id string, tokens []string) queueRead {
	sql := "SELECT SpannerNewLeaseToken FROM RENEWLEASE_Messages(lease_tokens=>@tokens)"
	o := queueRead{ID: id, SQL: sql, StatsScope: "query response exposes no caller CommitStats"}
	cx, cancel := context.WithTimeout(q.ctx, 15*time.Second)
	defer cancel()
	stmt := spanner.NewStatement(sql)
	stmt.Params = map[string]interface{}{"tokens": tokens}
	it := q.c.Client.Single().Query(cx, stmt)
	defer it.Stop()
	var err error
	for {
		row, e := it.Next()
		if e == iterator.Done {
			break
		}
		if e != nil {
			err = e
			break
		}
		o.Rows++
		var token spanner.NullString
		if e := row.Column(0, &token); e != nil {
			err = e
			break
		}
		o.Renewed = o.Renewed || token.Valid && token.StringVal != ""
	}
	o.Status = status.Code(err).String()
	if err != nil {
		o.Error = redact(err.Error())
		for _, t := range tokens {
			o.Error = redactLeaseError(status.Error(status.Code(err), o.Error), t)
		}
	}
	q.r.QueueReads = append(q.r.QueueReads, o)
	return o
}

func (q *queueMatrixRunner) seedReady(n int) error {
	if err := q.clearMessages(); err != nil {
		return err
	}
	var ms []*spanner.Mutation
	for i := range n {
		ms = append(ms, spanner.Send("Messages", spanner.Key{fmt.Sprintf("ready-%d", i)}, []byte("x")))
	}
	return q.seed(ms...)
}

func waitQueueExpiration(ctx context.Context, t time.Time) error {
	delay := time.Until(t) + time.Second
	if delay <= 0 {
		return nil
	}
	if delay > 20*time.Second {
		return fmt.Errorf("unexpected lease duration exceeds bounded expiry control")
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *queueMatrixRunner) leaseStates() error {
	for _, n := range []int{1, 3} {
		if err := q.seedReady(n); err != nil {
			return err
		}
		x, err := q.receive(fmt.Sprintf("lease/batch-receive/n=%d", n), n)
		if err != nil {
			return err
		}
		renewed := q.renew(fmt.Sprintf("lease/batch-renew/n=%d", n), x.tokens)
		var ms []*spanner.Mutation
		for _, k := range x.keys {
			ms = append(ms, spanner.Ack("Messages", spanner.Key{k}))
		}
		q.run(fmt.Sprintf("lease/batch-ack/n=%d", n), "Messages", "ack", fmt.Sprintf("%d received messages; renewal status=%s", n, renewed.Status), "OK", ms)
		x.close()
	}
	for _, redeliver := range []bool{false, true} {
		if err := q.seedReady(1); err != nil {
			return err
		}
		x, err := q.receive(fmt.Sprintf("lease/expiry-receive/redeliver=%v", redeliver), 1)
		if err != nil {
			return err
		}
		if err := waitQueueExpiration(q.ctx, x.expiration); err != nil {
			x.close()
			return err
		}
		x.close()
		if redeliver {
			y, err := q.receive("lease/redelivery", 1)
			if err != nil {
				return err
			}
			if y.keys[0] != x.keys[0] || y.tokens[0] == x.tokens[0] {
				y.close()
				return fmt.Errorf("redelivery identity/token control failed")
			}
			y.close()
		}
		q.run(fmt.Sprintf("lease/ack-after-expiry/redeliver=%v", redeliver), "Messages", "ack", "lease expiration passed before Ack", "OK", []*spanner.Mutation{spanner.Ack("Messages", spanner.Key{x.keys[0]})})
	}
	if err := q.seedReady(1); err != nil {
		return err
	}
	x, err := q.receive("lease/receive-before-ack-renew", 1)
	if err != nil {
		return err
	}
	q.run("lease/ack-before-renew", "Messages", "ack", "received message", "OK", []*spanner.Mutation{spanner.Ack("Messages", spanner.Key{x.keys[0]})})
	q.renew("lease/renew-after-ack", x.tokens)
	x.close()
	q.renew("lease/renew-malformed-token", []string{"not-a-token"})
	return nil
}

func (q *queueMatrixRunner) readWriteTVF() error {
	for _, write := range []bool{false, true} {
		if err := q.seedReady(1); err != nil {
			return err
		}
		id := fmt.Sprintf("lease/receive-in-rw/normal-write=%v", write)
		o := observation{ID: id, SchemaPhase: len(q.r.SchemaPhases) - 1, Table: "Messages", Kind: "queue_tvf_rw", State: "one ready message", Queries: []string{"SELECT Id FROM RECEIVE_Messages(max_duration=>'1s',max_batch_size=>1)"}}
		cx, cancel := context.WithTimeout(context.WithValue(q.ctx, caseKey{}, id), 20*time.Second)
		resp, err := q.c.Client.ReadWriteTransactionWithOptions(cx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
			o.QueryRows = nil
			it := tx.Query(ctx, spanner.NewStatement(o.Queries[0]))
			var rows int64
			_, err := it.Next()
			if err != nil && err != iterator.Done {
				it.Stop()
				return err
			}
			if err == nil {
				rows = 1
			}
			it.Stop()
			o.QueryRows = append(o.QueryRows, rows)
			if write {
				q.next++
				return tx.BufferWrite([]*spanner.Mutation{spanner.Insert("PlainRows", []string{"Id", "V"}, []interface{}{q.next, "x"})})
			}
			return nil
		}, spanner.TransactionOptions{CommitOptions: spanner.CommitOptions{ReturnCommitStats: true}})
		cancel()
		o.Status = status.Code(err).String()
		if err != nil {
			o.Error = redact(err.Error())
		}
		if resp.CommitStats != nil {
			n := resp.CommitStats.MutationCount
			o.Count = &n
			o.StatsPresent = true
		}
		q.record(o)
	}
	return nil
}

func (q *queueMatrixRunner) renewReadWrite() error {
	for _, ack := range []bool{false, true} {
		id := fmt.Sprintf("lease/renew-in-rw/ack=%v", ack)
		q.plan = append(q.plan, queueMatrixPlanCase{id, "Messages", "queue_renew_rw", "fresh lease; optional buffered Ack", "observe"})
		if q.c == nil {
			continue
		}
		if err := q.seedReady(1); err != nil {
			return err
		}
		x, err := q.receive(id+"/receive", 1)
		if err != nil {
			return err
		}
		sql := "SELECT SpannerNewLeaseToken FROM RENEWLEASE_Messages(lease_tokens=>@tokens)"
		o := observation{ID: id, SchemaPhase: len(q.r.SchemaPhases) - 1, Table: "Messages", Kind: "queue_renew_rw", State: "one fresh unacknowledged lease", Queries: []string{sql}}
		cx, cancel := context.WithTimeout(context.WithValue(q.ctx, caseKey{}, id), 20*time.Second)
		resp, err := q.c.Client.ReadWriteTransactionWithOptions(cx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
			stmt := spanner.NewStatement(sql)
			stmt.Params = map[string]interface{}{"tokens": x.tokens}
			it := tx.Query(ctx, stmt)
			var rows int64
			for {
				_, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					it.Stop()
					return err
				}
				rows++
			}
			it.Stop()
			o.QueryRows = []int64{rows}
			if ack {
				return tx.BufferWrite([]*spanner.Mutation{spanner.Ack("Messages", spanner.Key{x.keys[0]})})
			}
			return nil
		}, spanner.TransactionOptions{CommitOptions: spanner.CommitOptions{ReturnCommitStats: true}})
		cancel()
		x.close()
		o.Status = status.Code(err).String()
		if err != nil {
			o.Error = redactLeaseError(err, x.tokens[0])
		}
		if resp.CommitStats != nil {
			n := resp.CommitStats.MutationCount
			o.Count = &n
			o.StatsPresent = true
		}
		q.record(o)
	}
	return nil
}

func (q *queueMatrixRunner) nonCommitAPIs() error {
	for _, n := range []int{0, 3} {
		if err := q.seedReady(n); err != nil {
			return err
		}
		cx, cancel := context.WithTimeout(q.ctx, 45*time.Second)
		rows, err := q.c.Client.PartitionedUpdate(cx, spanner.NewStatement("DELETE FROM Messages WHERE TRUE"))
		cancel()
		o := queueOtherWrite{ID: fmt.Sprintf("pdml/delete/n=%d", n), API: "PartitionedUpdate", SQL: "DELETE FROM Messages WHERE TRUE", Status: status.Code(err).String(), StatsScope: "API returns an estimated affected-row lower bound, not CommitStats; mutation count remains unknown"}
		if err == nil {
			o.Rows = &rows
		} else {
			o.Error = redact(err.Error())
		}
		q.r.OtherWrites = append(q.r.OtherWrites, o)
	}
	groups := []*spanner.MutationGroup{
		{Mutations: []*spanner.Mutation{spanner.Send("Messages", spanner.Key{"batchwrite-1"}, []byte("x"), spanner.WithDeliveryTime(q.future))}},
		{Mutations: []*spanner.Mutation{spanner.Send("Messages", spanner.Key{"batchwrite-2"}, []byte("x"), spanner.WithDeliveryTime(q.future)), spanner.Send("Messages", spanner.Key{"batchwrite-3"}, []byte("x"), spanner.WithDeliveryTime(q.future))}},
		{Mutations: []*spanner.Mutation{spanner.Ack("Messages", spanner.Key{"batchwrite-missing"})}},
	}
	cx, cancel := context.WithTimeout(q.ctx, 45*time.Second)
	defer cancel()
	it := q.c.Client.BatchWrite(cx, groups)
	defer it.Stop()
	o := queueOtherWrite{ID: "batchwrite/send-and-missing-ack", API: "BatchWrite", StatsScope: "BatchWriteResponse has no CommitStats field; per-group status is not a zero count"}
	var err error
	for {
		r, e := it.Next()
		if e == iterator.Done {
			break
		}
		if e != nil {
			err = e
			break
		}
		g := queueBatchGroup{Indexes: r.Indexes, Status: codes.Code(r.GetStatus().GetCode()).String(), Committed: r.CommitTimestamp != nil}
		if r.GetStatus().GetCode() != 0 {
			g.Error = redact(r.GetStatus().GetMessage())
		}
		o.Groups = append(o.Groups, g)
	}
	o.Status = status.Code(err).String()
	if err != nil {
		o.Error = redact(err.Error())
	}
	q.r.OtherWrites = append(q.r.OtherWrites, o)
	return nil
}

func (q *queueMatrixRunner) profileEmpty() error {
	if err := q.clearMessages(); err != nil {
		return err
	}
	mode := spannerpb.ExecuteSqlRequest_PROFILE
	cx, cancel := context.WithTimeout(q.ctx, 10*time.Second)
	defer cancel()
	sql := "SELECT Id FROM RECEIVE_Messages(max_duration=>'1s',max_batch_size=>1)"
	it := q.c.Client.Single().QueryWithOptions(cx, spanner.NewStatement(sql), spanner.QueryOptions{Mode: &mode})
	defer it.Stop()
	o := queueRead{ID: "lease/empty-receive-profile", SQL: sql, StatsScope: "PROFILE query statistics are not caller CommitStats"}
	var err error
	for {
		_, e := it.Next()
		if e == iterator.Done {
			break
		}
		if e != nil {
			err = e
			break
		}
		o.Rows++
	}
	for k := range it.QueryStats {
		o.QueryStatsKeys = append(o.QueryStatsKeys, k)
	}
	sort.Strings(o.QueryStatsKeys)
	o.Status = status.Code(err).String()
	if err != nil {
		o.Error = redact(err.Error())
	}
	q.r.QueueReads = append(q.r.QueueReads, o)
	return nil
}

func (q *queueMatrixRunner) remaining() error {
	if err := q.clearMessages(); err != nil {
		return err
	}
	leaseErr := queueLeaseVerification(q.ctx, q.c, q.r, q.commit, "Messages")
	// Keep collecting independent controls after a lease spelling fails. The
	// final run status still records that failure instead of hiding it.
	for _, run := range []func() error{q.leaseStates, q.readWriteTVF, q.nonCommitAPIs, q.profileEmpty} {
		if err := run(); err != nil {
			return err
		}
	}
	var unexpected []string
	for _, o := range q.r.Observations {
		want, ok := q.expected[o.ID]
		if !ok || want == "observe" {
			continue
		}
		bad := want == "OK" && o.Status != "OK" || want == "reject" && (o.Status == "OK" || o.Status == "DeadlineExceeded" || o.Status == "Unavailable" || o.Status == "Canceled")
		if bad {
			unexpected = append(unexpected, o.ID+": "+o.Status)
		}
	}
	if leaseErr != nil {
		return leaseErr
	}
	if len(unexpected) > 0 {
		// An observed price or semantic transition is a research result, not an
		// unattempted flow. Keep failed hypotheses separate from completeness.
		q.r.HypothesisMismatches = unexpected
	}
	return nil
}
