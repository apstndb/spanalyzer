package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/status"
)

// Each spelling uses its own message, so a failed/expired token cannot affect
// the next spelling. Tokens stay in memory and are removed from error text.
func queueLeaseVerification(ctx context.Context, c *managedClients, r *result, commit commitFunc, queue string) error {
	verified := 0
	for _, tc := range []struct{ id, projection, argument string }{
		{"projected-scalar", "SpannerNewLeaseToken", "[@token]"},
		{"star-scalar", "*", "[@token]"},
		{"star-array", "*", "@tokens"},
	} {
		key := "lease-" + tc.id
		commit("queue/lease/send/"+tc.id, queue, "send", "absent", nil, []*spanner.Mutation{spanner.Send(queue, spanner.Key{key}, []byte("x"))})
		token := ""
		var receiver *spanner.RowIterator
		var receiverCancel context.CancelFunc
		read := func(id string, q spanner.Statement, receive bool) (e error) {
			cx, cancel := context.WithTimeout(ctx, 15*time.Second)
			it := c.Client.Single().Query(cx, q)
			if receive {
				// Match the earlier controlled lease experiment: keep the stream
				// open through renewal so cancellation cannot explain a failure.
				receiver, receiverCancel = it, cancel
			} else {
				defer cancel()
				defer it.Stop()
			}
			x := queueRead{ID: id, SQL: q.SQL, StatsScope: "no Commit RPC; not an observed zero mutation count"}
			defer func() {
				x.Status = status.Code(e).String()
				if e != nil {
					x.Error = redactLeaseError(e, token)
				}
				r.QueueReads = append(r.QueueReads, x)
				fmt.Fprintln(os.Stderr, id, x.Status)
			}()
			for {
				row, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				if err != nil {
					return err
				}
				x.Rows++
				if receive {
					return row.ColumnByName("SpannerLeaseToken", &token)
				}
				var next spanner.NullString
				if err := row.ColumnByName("SpannerNewLeaseToken", &next); err != nil {
					return err
				}
				x.Renewed = x.Renewed || next.Valid && next.StringVal != ""
			}
		}
		if err := read("receive/"+tc.id, spanner.NewStatement("SELECT SpannerLeaseToken FROM RECEIVE_"+queue+"(max_duration=>'20m',max_batch_size=>1)"), true); err != nil || token == "" {
			receiver.Stop()
			receiverCancel()
			return fmt.Errorf("minimal lease fixture could not receive a token")
		}
		q := spanner.NewStatement("SELECT " + tc.projection + " FROM RENEWLEASE_" + queue + "(lease_tokens=>" + tc.argument + ")")
		if tc.id == "star-array" {
			q.Params = map[string]interface{}{"tokens": []string{token}}
		} else {
			q.Params = map[string]interface{}{"token": token}
		}
		_ = read("renew/"+tc.id, q, false)
		x := r.QueueReads[len(r.QueueReads)-1]
		if x.Status == "OK" && x.Renewed {
			verified++
		}
		commit("queue/lease/ack/"+tc.id, queue, "ack", "received row", nil, []*spanner.Mutation{spanner.Ack(queue, spanner.Key{key})})
		receiver.Stop()
		receiverCancel()
	}
	if verified == 0 {
		return fmt.Errorf("no lease-query spelling returned a renewed token")
	}
	return nil
}

func redactLeaseError(err error, token string) string {
	s := redact(status.Convert(err).Message())
	if token != "" {
		s = strings.ReplaceAll(s, token, "<lease-token>")
	}
	return s
}
