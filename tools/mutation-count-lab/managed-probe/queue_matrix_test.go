package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/cloudspannerecosystem/memefish"
	"google.golang.org/protobuf/proto"
)

func TestQueueMatrixPreparedCoverage(t *testing.T) {
	b, err := os.ReadFile("../../../testdata/protos/order_descriptors.pb")
	if err != nil {
		t.Fatal(err)
	}
	// This producer-owned schema uses only the shared CREATE TABLE grammar
	// after replacing its known queue declaration tokens; no external SQL is
	// adapted by this test.
	ddl := strings.ReplaceAll(strings.Join(queueMatrixDDL(), ";\n"), "CREATE QUEUE ", "CREATE TABLE ")
	if _, err := memefish.ParseDDLs("queue-fixture.sql", ddl); err != nil {
		t.Fatal(err)
	}
	q := newQueueMatrixRunner(context.Background(), nil, &result{}, nil, nil, b)
	if err := q.writes(); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, c := range q.plan {
		if seen[c.ID] {
			t.Fatal("duplicate evidence ID", c.ID)
		}
		seen[c.ID] = true
	}
	for _, id := range []string{"send/Messages/bytes=0/delivery=default", "send/ProtoMessages/bytes=8192/delivery=future", "batch/ack/ThreeKeyMessages/n=10", "dml/delete/Messages/n=0", "composition/dml-insert-buffered-ack", "cascade/DeepMessages/replace/n=10", "quota/send/n=16000", "quota/send/n=16001", "quota/ack/n=40000", "quota/ack/n=40001"} {
		if !seen[id] {
			t.Error("missing scenario", id)
		}
	}
	t.Logf("prepared %d independent write cases", len(q.plan))
}

func TestQueueProtoPayloadUsesFixtureDescriptor(t *testing.T) {
	b, err := os.ReadFile("../../../testdata/protos/order_descriptors.pb")
	if err != nil {
		t.Fatal(err)
	}
	m, err := queueProtoPayload(b, "x")
	if err != nil {
		t.Fatal(err)
	}
	wire, err := proto.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if string(wire) != "\x0a\x01x" {
		t.Fatalf("unexpected Order payload: %x", wire)
	}
	if _, err := queueProtoPayload([]byte("not a descriptor set"), "x"); err == nil {
		t.Fatal("invalid descriptors accepted")
	}
}

func TestQueueIgnoreFollowupPlan(t *testing.T) {
	q := newQueueMatrixRunner(context.Background(), nil, &result{}, nil, nil, nil)
	if err := q.ignoreCases(); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, c := range q.plan {
		if seen[c.ID] {
			t.Fatal("duplicate evidence ID", c.ID)
		}
		seen[c.ID] = true
	}
	for _, id := range []string{"ignore/one/state=same/delivery=false", "ignore/mixed3/present=3/delivery=true", "select/ignore/n=3/present=3", "ignore/two-statements", "delete/assert-rows/present=false/assert=1"} {
		if !seen[id] {
			t.Error("missing follow-up", id)
		}
	}
	t.Logf("prepared %d follow-up cases", len(q.plan))
}
