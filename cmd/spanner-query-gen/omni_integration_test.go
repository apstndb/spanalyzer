//go:build integration && omni

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanemuboost"
	"github.com/apstndb/spannerplan/plantree/reference"
)

func TestIntegrationQueryCodegenGeneratedSpannerQueriesRunOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	querygenIntegrationRequireContainerRuntime(t)

	config, plan, baseDir, ddl := querygenIntegrationBuildFixture(t)
	runtime := spanemuboost.NewLazyRuntime(spanemuboost.BackendOmni)
	t.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			t.Errorf("failed to close Spanner Omni runtime: %v", err)
		}
	})
	clients := spanemuboost.SetupClients(t, runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(querygenIntegrationDDLs(t, "schema.sql", ddl)),
	)
	querygenIntegrationExecutePlan(t, clients.Client, plan, "Spanner Omni")

	indexQuery := querygenOmniIntegrationPlanQuery(t, plan, "FindSingersByFirstName")
	t.Run("GeneratedIndexQueryPlanDoesNotSort", func(t *testing.T) {
		queryPlan := querygenOmniIntegrationAnalyzeQuery(t, clients.Client, indexQuery)
		if querygenOmniIntegrationPlanHasSort(queryPlan) {
			t.Fatalf("generated index query plan contains a Sort node:\nSQL: %s\nplan nodes:\n%s", indexQuery.SQL, querygenOmniIntegrationPlanNodeSummary(queryPlan))
		}
	})
	t.Run("DifferentOrderRequiresSort", func(t *testing.T) {
		sortQuery := indexQuery
		sortQuery.SQL = strings.Replace(sortQuery.SQL,
			"ORDER BY `FirstName`, `LastName`, `Rating`, `SingerId`",
			"ORDER BY `Rating`, `LastName`, `SingerId`",
			1,
		)
		if sortQuery.SQL == indexQuery.SQL {
			t.Fatalf("failed to build intentionally sorted query from SQL: %s", indexQuery.SQL)
		}
		queryPlan := querygenOmniIntegrationAnalyzeQuery(t, clients.Client, sortQuery)
		if !querygenOmniIntegrationPlanHasSort(queryPlan) {
			t.Fatalf("query with non-index order did not contain a Sort node:\nSQL: %s\nplan nodes:\n%s", sortQuery.SQL, querygenOmniIntegrationPlanNodeSummary(queryPlan))
		}
	})
	t.Run("MissingNullFilteredIndexPredicateFails", func(t *testing.T) {
		brokenQuery := indexQuery
		brokenQuery.SQL = strings.Replace(brokenQuery.SQL, " AND `LastName` IS NOT NULL", "", 1)
		if brokenQuery.SQL == indexQuery.SQL {
			t.Fatalf("failed to remove null-filtered predicate from SQL: %s", indexQuery.SQL)
		}
		stmt := spanner.NewStatement(brokenQuery.SQL)
		stmt.Params = querygenIntegrationParamValues(t, brokenQuery.Params)
		if _, err := clients.Client.Single().AnalyzeQuery(t.Context(), stmt); err == nil {
			t.Fatalf("AnalyzeQuery() succeeded for query missing a NULL_FILTERED index predicate:\nSQL: %s", brokenQuery.SQL)
		}
	})
	t.Run("AggregateGroupMethodContracts", func(t *testing.T) {
		hashQuery := querygen.QueryCodegenPlanQuery{
			Name: "HashAggregateSingersByRating",
			Kind: "sql",
			SQL:  "SELECT Rating, COUNT(*) AS singer_count FROM Singers GROUP@{GROUP_METHOD=HASH_GROUP} BY Rating",
		}
		streamQuery := querygen.QueryCodegenPlanQuery{
			Name: "StreamAggregateSingersByRating",
			Kind: "sql",
			SQL:  "SELECT Rating, COUNT(*) AS singer_count FROM Singers GROUP@{GROUP_METHOD=STREAM_GROUP} BY Rating",
		}
		hashPlan := querygenOmniIntegrationAnalyzeQuery(t, clients.Client, hashQuery)
		streamPlan := querygenOmniIntegrationAnalyzeQuery(t, clients.Client, streamQuery)
		if !querygenOmniIntegrationPlanHasFamily(hashPlan, "hash_aggregate") {
			t.Fatalf("HASH_GROUP query plan does not contain hash_aggregate:\nSQL: %s\nplan nodes:\n%s", hashQuery.SQL, querygenOmniIntegrationPlanNodeSummary(hashPlan))
		}
		if querygenOmniIntegrationPlanHasFamily(hashPlan, "stream_aggregate") {
			t.Fatalf("HASH_GROUP query plan unexpectedly contains stream_aggregate:\nSQL: %s\nplan nodes:\n%s", hashQuery.SQL, querygenOmniIntegrationPlanNodeSummary(hashPlan))
		}
		if !querygenOmniIntegrationPlanHasFamily(streamPlan, "stream_aggregate") {
			t.Fatalf("STREAM_GROUP query plan does not contain stream_aggregate:\nSQL: %s\nplan nodes:\n%s", streamQuery.SQL, querygenOmniIntegrationPlanNodeSummary(streamPlan))
		}
		if querygenOmniIntegrationPlanHasFamily(streamPlan, "hash_aggregate") {
			t.Fatalf("STREAM_GROUP query plan unexpectedly contains hash_aggregate:\nSQL: %s\nplan nodes:\n%s", streamQuery.SQL, querygenOmniIntegrationPlanNodeSummary(streamPlan))
		}

		report := planReport{
			Status:  planReportStatusOK,
			Backend: "omni",
			Queries: []planReportQuery{
				querygenOmniIntegrationPlanReportQuery(hashQuery, hashPlan),
				querygenOmniIntegrationPlanReportQuery(streamQuery, streamPlan),
			},
		}
		contracts := planContractsFile{
			Version: "v1alpha-plan-contracts",
			Contracts: []planContract{
				{
					Name:   "HashAggregateCEL",
					Target: "query/" + hashQuery.Name,
					CEL:    `operators.exists(o, o.family == "hash_aggregate") && operators.all(o, o.family != "stream_aggregate")`,
				},
				{Name: "NoHashAggregate", Target: "query/" + hashQuery.Name, Use: []string{"no_hash_aggregate"}},
				{Name: "NoStreamAggregate", Target: "query/" + streamQuery.Name, Use: []string{"no_stream_aggregate"}},
			},
		}
		if err := applyPlanContracts(&report, contracts); err != nil {
			t.Fatalf("applyPlanContracts() error = %v", err)
		}
		if got, want := report.ContractSummary.Status, planContractStatusFail; got != want {
			t.Fatalf("aggregate contract status = %q, want %q: %+v", got, want, report.ContractEvaluations)
		}
		if got, want := report.ContractSummary.Passed, 1; got != want {
			t.Fatalf("aggregate contract passed = %d, want %d: %+v", got, want, report.ContractEvaluations)
		}
		if got, want := planReportContractViolationCount(report), 2; got != want {
			t.Fatalf("aggregate contract violation count = %d, want %d: %+v", got, want, report.ContractEvaluations)
		}
	})
	t.Run("JoinMethodContracts", func(t *testing.T) {
		joinQueries := []struct {
			query querygen.QueryCodegenPlanQuery
			want  string
		}{
			{
				query: querygen.QueryCodegenPlanQuery{
					Name: "HashJoin",
					Kind: "sql",
					SQL:  "SELECT s.SingerId, a.AlbumId FROM Singers s JOIN@{JOIN_METHOD=HASH_JOIN} Albums a ON s.SingerId = a.SingerId",
				},
				want: "hash_join",
			},
			{
				query: querygen.QueryCodegenPlanQuery{
					Name: "ApplyJoin",
					Kind: "sql",
					SQL:  "SELECT s.SingerId, a.AlbumId FROM Singers s JOIN@{JOIN_METHOD=APPLY_JOIN} Albums a ON s.SingerId = a.SingerId",
				},
				want: "distributed_cross_apply",
			},
			{
				query: querygen.QueryCodegenPlanQuery{
					Name: "MergeJoin",
					Kind: "sql",
					SQL:  "SELECT s.SingerId, a.AlbumId FROM Singers s JOIN@{JOIN_METHOD=MERGE_JOIN} Albums a ON s.SingerId = a.SingerId",
				},
				want: "merge_join",
			},
			{
				query: querygen.QueryCodegenPlanQuery{
					Name: "PushBroadcastHashJoin",
					Kind: "sql",
					SQL:  "SELECT s.SingerId, a.AlbumId FROM Singers s JOIN@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} Albums a ON s.SingerId = a.SingerId",
				},
				want: "push_broadcast_hash_join",
			},
		}
		report := planReport{Status: planReportStatusOK, Backend: "omni"}
		for _, tc := range joinQueries {
			queryPlan := querygenOmniIntegrationAnalyzeQuery(t, clients.Client, tc.query)
			if !querygenOmniIntegrationPlanHasFamily(queryPlan, tc.want) {
				t.Fatalf("%s plan does not contain %s:\nSQL: %s\nplan nodes:\n%s", tc.query.Name, tc.want, tc.query.SQL, querygenOmniIntegrationPlanNodeSummary(queryPlan))
			}
			if tc.query.Name == "PushBroadcastHashJoin" {
				if querygenOmniIntegrationPlanHasFamily(queryPlan, "hash_join") {
					t.Fatalf("Push Broadcast Hash Join plan was classified as a plain hash_join:\nSQL: %s\nplan nodes:\n%s", tc.query.SQL, querygenOmniIntegrationPlanNodeSummary(queryPlan))
				}
				if !querygenOmniIntegrationPlanHasFamily(queryPlan, "push_broadcast_hash_join_internal_hash_join") {
					t.Fatalf("Push Broadcast Hash Join plan does not expose its internal Hash Join family:\nSQL: %s\nplan nodes:\n%s", tc.query.SQL, querygenOmniIntegrationPlanNodeSummary(queryPlan))
				}
			}
			if tc.query.Name == "ApplyJoin" {
				if querygenOmniIntegrationPlanHasFamily(queryPlan, "apply_join") {
					t.Fatalf("Distributed Cross Apply plan was classified as a plain apply_join:\nSQL: %s\nplan nodes:\n%s", tc.query.SQL, querygenOmniIntegrationPlanNodeSummary(queryPlan))
				}
				if !querygenOmniIntegrationPlanHasFamily(queryPlan, "distributed_cross_apply_internal_apply") {
					t.Fatalf("Distributed Cross Apply plan does not expose its internal Cross Apply family:\nSQL: %s\nplan nodes:\n%s", tc.query.SQL, querygenOmniIntegrationPlanNodeSummary(queryPlan))
				}
			}
			report.Queries = append(report.Queries, querygenOmniIntegrationPlanReportQuery(tc.query, queryPlan))
		}
		contracts := planContractsFile{
			Version: "v1alpha-plan-contracts",
			Contracts: []planContract{
				{Name: "NoHashJoin", Target: "query/HashJoin", Use: []string{"no_hash_join"}},
				{Name: "NoHashJoinRejectsPushBroadcast", Target: "query/PushBroadcastHashJoin", Use: []string{"no_hash_join"}},
				{Name: "NoDistributedCrossApply", Target: "query/ApplyJoin", Use: []string{"no_distributed_cross_apply"}},
				{Name: "NoMergeJoin", Target: "query/MergeJoin", Use: []string{"no_merge_join"}},
				{Name: "NoPushBroadcastHashJoin", Target: "query/PushBroadcastHashJoin", Use: []string{"no_push_broadcast_hash_join"}},
			},
		}
		if err := applyPlanContracts(&report, contracts); err != nil {
			t.Fatalf("applyPlanContracts() error = %v", err)
		}
		if got, want := planReportContractViolationCount(report), 5; got != want {
			t.Fatalf("join contract violation count = %d, want %d: %+v", got, want, report.ContractEvaluations)
		}
	})
	t.Run("ScanAndIndexHintMetadataContracts", func(t *testing.T) {
		queries := []querygen.QueryCodegenPlanQuery{
			{
				Name: "RowScan",
				Kind: "sql",
				SQL:  "SELECT SingerId FROM Singers@{SCAN_METHOD=ROW}",
			},
			{
				Name: "BatchScan",
				Kind: "sql",
				SQL:  "SELECT SingerId FROM Singers@{SCAN_METHOD=BATCH}",
			},
			{
				Name: "ForceBaseTable",
				Kind: "sql",
				SQL:  "SELECT SingerId FROM Singers@{FORCE_INDEX=_BASE_TABLE} WHERE FirstName = 'A'",
			},
			{
				Name: "IndexUnion",
				Kind: "sql",
				SQL:  "SELECT SingerId FROM Singers@{INDEX_STRATEGY=FORCE_INDEX_UNION} WHERE FirstName = 'A' OR LastName = 'B'",
			},
		}
		report := planReport{Status: planReportStatusOK, Backend: "omni"}
		for _, query := range queries {
			queryPlan := querygenOmniIntegrationAnalyzeQuery(t, clients.Client, query)
			report.Queries = append(report.Queries, querygenOmniIntegrationPlanReportQuery(query, queryPlan))
		}
		contracts := planContractsFile{
			Version: "v1alpha-plan-contracts",
			Contracts: []planContract{
				{Name: "RowScanMethod", Target: "query/RowScan", CEL: `operators.exists(o, o.scan_method == "row")`},
				{Name: "RowScanRawPlanMetadata", Target: "query/RowScan", CEL: `raw_nodes.exists(n, n.metadata["scan_method"] == "Row") && raw_nodes.exists(n, n.child_links.size() > 0)`},
				{Name: "BatchScanMethod", Target: "query/BatchScan", CEL: `operators.exists(o, o.scan_method == "batch")`},
				{Name: "BaseTableScan", Target: "query/ForceBaseTable", CEL: `operators.exists(o, o.scan_type == "table_scan" && o.scan_target == "Singers")`},
				{Name: "IndexUnionShape", Target: "query/IndexUnion", CEL: `operators.exists(o, o.family == "union_all") && operators.exists(o, o.family == "hash_aggregate")`},
			},
		}
		if err := applyPlanContracts(&report, contracts); err != nil {
			t.Fatalf("applyPlanContracts() error = %v", err)
		}
		if got, want := report.ContractSummary.Status, planContractStatusPass; got != want {
			t.Fatalf("scan/index contract status = %q, want %q: %+v", got, want, report.ContractEvaluations)
		}
	})
	t.Run("PlanReportUsesSharedOmniRuntime", func(t *testing.T) {
		report, err := buildPlanReportWithRuntime(t.Context(), config, plan, baseDir, planReportOptions{
			Backend:    "omni",
			Format:     reference.FormatCurrent,
			RenderMode: reference.RenderModePlan,
			WrapWidth:  100,
		}, runtime)
		if err != nil {
			t.Fatalf("buildPlanReportWithRuntime() error = %v", err)
		}
		if got, want := len(report.Queries), len(plan.Queries); got != want {
			t.Fatalf("plan-report query count = %d, want %d", got, want)
		}
		for _, query := range report.Queries {
			if query.Status != "ok" {
				t.Fatalf("plan-report query %s status = %q, error = %q", query.Name, query.Status, query.Error)
			}
			if query.Plan == "" {
				t.Fatalf("plan-report query %s has empty rendered plan", query.Name)
			}
			if len(query.OperatorFamilyCounts) == 0 {
				t.Fatalf("plan-report query %s has empty operator family counts", query.Name)
			}
		}
		contracts := planContractsFile{
			Version: "v1alpha-plan-contracts",
			Contracts: []planContract{{
				Name:   "FindSingersByFirstNamePlan",
				Target: "query/FindSingersByFirstName",
				Use:    []string{"no_explicit_sort"},
			}, {
				Name:   "FindSingersByFirstNameCounts",
				Target: "query/FindSingersByFirstName",
				CEL:    `operator_family_counts["scan"] > 0 && query.operator_family_counts["scan"] == operator_family_counts["scan"]`,
			}},
		}
		if err := applyPlanContracts(&report, contracts); err != nil {
			t.Fatalf("applyPlanContracts() error = %v", err)
		}
		if got, want := report.ContractSummary.Status, planContractStatusPass; got != want {
			t.Fatalf("plan-report contract status = %q, want %q: %+v", got, want, report.ContractEvaluations)
		}
	})
}

func querygenOmniIntegrationPlanQuery(t testing.TB, plan *querygen.QueryCodegenPlan, name string) querygen.QueryCodegenPlanQuery {
	t.Helper()
	for _, query := range plan.Queries {
		if query.Name == name {
			return query
		}
	}
	t.Fatalf("query %q not found in plan", name)
	return querygen.QueryCodegenPlanQuery{}
}

func querygenOmniIntegrationAnalyzeQuery(t testing.TB, client *spanner.Client, query querygen.QueryCodegenPlanQuery) *spannerpb.QueryPlan {
	t.Helper()
	stmt := spanner.NewStatement(query.SQL)
	stmt.Params = querygenIntegrationParamValues(t, query.Params)
	plan, err := client.Single().AnalyzeQuery(t.Context(), stmt)
	if err != nil {
		t.Fatalf("AnalyzeQuery() error = %v\nSQL: %s\nparams: %#v", err, query.SQL, stmt.Params)
	}
	return plan
}

func querygenOmniIntegrationPlanHasSort(plan *spannerpb.QueryPlan) bool {
	for _, node := range plan.GetPlanNodes() {
		if strings.Contains(strings.ToLower(node.GetDisplayName()), "sort") ||
			strings.Contains(strings.ToLower(node.GetShortRepresentation().GetDescription()), "sort") {
			return true
		}
	}
	return false
}

func querygenOmniIntegrationPlanHasFamily(plan *spannerpb.QueryPlan, family string) bool {
	for _, operator := range planReportOperators(plan) {
		if operator.Family == family {
			return true
		}
	}
	return false
}

func querygenOmniIntegrationPlanReportQuery(query querygen.QueryCodegenPlanQuery, plan *spannerpb.QueryPlan) planReportQuery {
	operators := planReportOperators(plan)
	return planReportQuery{
		Name:                 query.Name,
		TargetID:             planReportTargetID(query.Name, "query"),
		Scope:                "query",
		Kind:                 query.Kind,
		Status:               "ok",
		SQL:                  query.SQL,
		SQLSHA256:            planReportDigest(query.SQL),
		OperatorTreeSHA256:   planReportOperatorTreeDigest(plan),
		OperatorFamilies:     planReportOperatorFamilies(operators),
		OperatorFamilyCounts: planReportOperatorFamilyCounts(operators),
		NormalizedOperators:  operators,
		ClassificationNotes:  planReportClassificationWarnings(operators),
		RawPlan:              plan,
	}
}

func querygenOmniIntegrationPlanNodeSummary(plan *spannerpb.QueryPlan) string {
	var b strings.Builder
	for _, node := range plan.GetPlanNodes() {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%d %s", node.GetIndex(), node.GetDisplayName())
		if short := node.GetShortRepresentation().GetDescription(); short != "" {
			fmt.Fprintf(&b, " -- %s", short)
		}
	}
	return b.String()
}
