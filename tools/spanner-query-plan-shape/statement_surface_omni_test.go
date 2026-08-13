//go:build integration && omni

package main

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	spannerapi "cloud.google.com/go/spanner/apiv1"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/proto"
)

func TestIntegrationStatementSurfaceOnOmni(t *testing.T) {
	clients := openSearchGraphOmniClients(t, nil)
	cases := queryCasesByLabel(t, statementSurfaceQueries)

	wantErrors := map[string]string{
		"statement-surface/call/cancel-parameter-error":     "Invalid arguments to procedure: cancel_query",
		"statement-surface/call/cancel-expression-error":    "Invalid arguments to procedure: cancel_query",
		"statement-surface/call/unknown-procedure-error":    "missing_spanalyzer_procedure is not supported",
		"statement-surface/call/sql-optimizer-hint-error":   "Unsupported hint: OPTIMIZER_VERSION",
		"statement-surface/routing/analyze-ddl-error":       "DDL statements cannot be processed by the Query API",
		"statement-surface/routing/start-batch-ddl-error":   "Statement not supported: StartBatchStatement",
		"statement-surface/routing/execute-immediate-error": "Statement not supported: ExecuteImmediateStatement",
	}
	for _, query := range statementSurfaceQueries {
		t.Run("default/"+query.Label, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			plan, err := analyzePlan(ctx, clients.Client, query)
			if want, isError := wantErrors[query.Label]; isError {
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("AnalyzeQuery() error = %v, want substring %q", err, want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertTopLevelCallPlan(t, plan)
		})
	}

	raw, err := spannerapi.NewClient(t.Context(), clients.ClientOptions()...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	session, err := raw.CreateSession(t.Context(), &spannerpb.CreateSessionRequest{
		Database: clients.DatabasePath(),
		Session:  &spannerpb.Session{Multiplexed: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Multiplexed sessions cannot be deleted; closing the client is the only
	// cleanup needed for this raw-RPC view of the same database.

	for _, label := range []string{
		"statement-surface/call/cancel-literal",
		"statement-surface/call/cancel-cast",
		"statement-surface/call/compact-all",
	} {
		query := cases[label]
		var first *spannerpb.QueryPlan
		for version := 1; version <= 8; version++ {
			t.Run(label+"/v"+strconv.Itoa(version), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()
				unary, err := raw.ExecuteSql(ctx, callPlanRequest(session.GetName(), query.SQL, version))
				if err != nil {
					t.Fatal(err)
				}
				unaryPlan := unary.GetStats().GetQueryPlan()
				streamPlan, err := executeStreamingCallPlan(ctx, raw, callPlanRequest(session.GetName(), query.SQL, version))
				if err != nil {
					t.Fatal(err)
				}
				assertTopLevelCallPlan(t, unaryPlan)
				if !proto.Equal(unaryPlan, streamPlan) {
					t.Error("ExecuteSql and ExecuteStreamingSql plans differ")
				}
				if first == nil {
					first = proto.Clone(unaryPlan).(*spannerpb.QueryPlan)
				} else if !proto.Equal(first, unaryPlan) {
					t.Errorf("v%d plan differs from v1", version)
				}
			})
		}
	}
}

func callPlanRequest(session, sql string, optimizerVersion int) *spannerpb.ExecuteSqlRequest {
	return &spannerpb.ExecuteSqlRequest{
		Session:      session,
		Sql:          sql,
		QueryMode:    spannerpb.ExecuteSqlRequest_PLAN,
		QueryOptions: &spannerpb.ExecuteSqlRequest_QueryOptions{OptimizerVersion: strconv.Itoa(optimizerVersion)},
		Transaction: &spannerpb.TransactionSelector{Selector: &spannerpb.TransactionSelector_SingleUse{
			SingleUse: &spannerpb.TransactionOptions{Mode: &spannerpb.TransactionOptions_ReadOnly_{
				ReadOnly: &spannerpb.TransactionOptions_ReadOnly{TimestampBound: &spannerpb.TransactionOptions_ReadOnly_Strong{Strong: true}},
			}},
		}},
	}
}

func executeStreamingCallPlan(ctx context.Context, client *spannerapi.Client, request *spannerpb.ExecuteSqlRequest) (*spannerpb.QueryPlan, error) {
	stream, err := client.ExecuteStreamingSql(ctx, request)
	if err != nil {
		return nil, err
	}
	var plan *spannerpb.QueryPlan
	for {
		partial, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if current := partial.GetStats().GetQueryPlan(); current != nil {
			plan = current
		}
	}
	if plan == nil {
		return nil, fmt.Errorf("stream returned no query plan")
	}
	return plan, nil
}

func assertTopLevelCallPlan(t *testing.T, plan *spannerpb.QueryPlan) {
	t.Helper()
	if plan == nil {
		t.Fatal("query plan is nil")
	}
	if got, want := len(plan.GetPlanNodes()), 6; got != want {
		t.Errorf("plan node count = %d, want %d", got, want)
	}
	for displayName, want := range map[string]int{
		"Serialize Result": 1,
		"TVF":              1,
		"Unit Relation":    1,
		"Constant":         1,
		"Reference":        2,
	} {
		got := 0
		for _, node := range plan.GetPlanNodes() {
			if node.GetDisplayName() == displayName {
				got++
			}
		}
		if got != want {
			t.Errorf("%s count = %d, want %d", displayName, got, want)
		}
	}
	nodes := compactTreeNodesByIndex(plan)
	for _, node := range plan.GetPlanNodes() {
		if node.GetDisplayName() != "TVF" {
			continue
		}
		for _, link := range node.GetChildLinks() {
			child := nodes[link.GetChildIndex()]
			if link.GetType() == "Output" && link.GetVariable() != "" && child.GetKind() == spannerpb.PlanNode_SCALAR {
				return
			}
		}
	}
	t.Error("TVF has no scalar Output link with a variable")
}
