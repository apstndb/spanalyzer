//go:build integration && omni

package main

import (
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanemuboost"
	"google.golang.org/protobuf/proto"
)

func TestIntegrationGoogleSQLSurfaceVersionMatrixOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	image := strings.TrimSpace(os.Getenv("SPANALYZER_OMNI_IMAGE"))
	if image == "" {
		t.Fatal("set SPANALYZER_OMNI_IMAGE to the pinned Spanner Omni image under test")
	}
	ddls, err := parseBuiltInDDLs("google-sql-surface-schema.sql", docsDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	runtime := spanemuboost.NewLazyRuntime(
		spanemuboost.BackendOmni,
		spanemuboost.WithContainerImage(image),
	)
	t.Cleanup(func() { _ = runtime.Close() })
	clients, err := spanemuboost.OpenClients(t.Context(), runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(ddls),
	)
	if err != nil {
		t.Fatalf("OpenClients() error = %v", err)
	}
	t.Cleanup(func() { _ = clients.Close() })
	if _, err := clients.Client.Apply(t.Context(), []*spanner.Mutation{
		spanner.Insert("Concerts", []string{"VenueId", "SingerId", "ConcertDate", "TicketPrices"}, []any{int64(1), int64(1), civil.Date{Year: 2026, Month: 8, Day: 11}, []int64{}}),
		spanner.Insert("Concerts", []string{"VenueId", "SingerId", "ConcertDate", "TicketPrices"}, []any{int64(2), int64(2), civil.Date{Year: 2026, Month: 8, Day: 11}, []int64{10, 20}}),
	}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	cases := queryCasesByLabel(t, googleSQLSurfaceQueries)
	analyze := func(t *testing.T, query queryCase, version int) (*spannerpb.QueryPlan, error) {
		t.Helper()
		query.SQL = withOptimizerVersionStatementHint(query.SQL, version)
		return analyzePlan(t.Context(), clients.Client, query)
	}
	mustAnalyze := func(t *testing.T, query queryCase, version int) *spannerpb.QueryPlan {
		t.Helper()
		plan, err := analyze(t, query, version)
		if err != nil {
			t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", query.Label, version, err)
		}
		return plan
	}

	readWrite := cases["google-sql-surface/accepted/for-update-read-write"]
	readOnly := cases["google-sql-surface/unsupported/for-update-read-only"]
	lockConflict := cases["google-sql-surface/unsupported/lock-hint-with-for-update"]
	havingMax := cases["google-sql-surface/accepted/aggregate-having-max"]
	havingMin := cases["google-sql-surface/accepted/aggregate-having-min"]
	arrayAgg := cases["google-sql-surface/accepted/array-agg-order-limit"]
	arrayTransform := cases["google-sql-surface/accepted/array-transform-lambda"]
	arrayFilter := cases["google-sql-surface/accepted/array-filter-lambda"]
	inUnnest := cases["google-sql-surface/accepted/in-unnest"]
	withExpression := cases["google-sql-surface/accepted/with-expression"]
	collate := cases["google-sql-surface/accepted/order-by-collate"]
	countDistinct := cases["google-sql-surface/accepted/aggregate-count-distinct"]
	arrayDistinctIgnore := cases["google-sql-surface/accepted/aggregate-array-distinct-ignore-nulls"]
	arrayDistinctRespect := cases["google-sql-surface/accepted/aggregate-array-distinct-respect-nulls"]
	notDistinct := cases["google-sql-surface/accepted/is-not-distinct-from"]
	distinctFrom := cases["google-sql-surface/accepted/is-distinct-from"]
	explicitUnnest := cases["google-sql-surface/accepted/correlated-unnest-explicit"]
	implicitUnnest := cases["google-sql-surface/accepted/correlated-unnest-implicit"]
	leftUnnest := cases["google-sql-surface/accepted/correlated-unnest-left"]
	inSubquery := cases["google-sql-surface/accepted/in-subquery-control"]
	inArraySubquery := cases["google-sql-surface/accepted/in-unnest-array-subquery"]
	notInSubquery := cases["google-sql-surface/accepted/not-in-subquery-control"]
	notInArraySubquery := cases["google-sql-surface/accepted/not-in-unnest-array-subquery"]
	groupOrdinal := cases["google-sql-surface/accepted/group-by-ordinal"]
	groupName := cases["google-sql-surface/accepted/group-by-name-control"]
	tablesampleSubquery := cases["google-sql-surface/accepted/tablesample-subquery"]
	tablesampleJoin := cases["google-sql-surface/accepted/tablesample-join"]
	structStar := cases["google-sql-surface/accepted/struct-expression-star"]
	arrayStructUnnest := cases["google-sql-surface/accepted/unnest-array-of-struct"]
	regularFirstSetOperation := cases["google-sql-surface/accepted/set-operation-regular-first-value-second"]
	valueFirstSetOperationNested := cases["google-sql-surface/accepted/set-operation-value-first-nested"]
	analyticRowNumber := cases["google-sql-surface/unsupported/analytic-row-number"]
	analyticFramedSum := cases["google-sql-surface/unsupported/analytic-framed-sum"]
	repeatable := cases["google-sql-surface/unsupported/tablesample-repeatable"]
	selectAsStruct := cases["google-sql-surface/unsupported/select-as-struct-top-level"]
	lateralCross := cases["google-sql-surface/unsupported/lateral-cross"]
	lateralLeft := cases["google-sql-surface/unsupported/lateral-left"]
	setOperationByName := cases["google-sql-surface/unsupported/set-operation-by-name"]
	setOperationCorresponding := cases["google-sql-surface/unsupported/set-operation-corresponding"]
	groupByAll := cases["google-sql-surface/unsupported/group-by-all"]
	matchRecognize := cases["google-sql-surface/unsupported/match-recognize"]
	valueFirstSetOperation := cases["google-sql-surface/unsupported/set-operation-value-first-top-level"]
	secureContext := cases["google-sql-surface/unsupported/secure-context"]

	for version := 1; version <= 8; version++ {
		readWritePlan := mustAnalyze(t, readWrite, version)
		if got := countPlanNodesWithMetadata(readWritePlan, "Filter Scan", map[string]string{"seekable_key_size": "1"}); got != 1 {
			t.Errorf("read-write FOR UPDATE v%d Filter Scan count = %d, want 1", version, got)
		}
		if _, err := analyze(t, readOnly, version); err == nil || !strings.Contains(err.Error(), "FOR UPDATE is not supported in this transaction type") {
			t.Errorf("read-only FOR UPDATE v%d error = %v, want stable transaction-type error", version, err)
		}
		if _, err := analyze(t, lockConflict, version); err == nil || !strings.Contains(err.Error(), "FOR UPDATE cannot be combined with statement-level lock hints") {
			t.Errorf("lock hint with FOR UPDATE v%d error = %v, want stable conflict error", version, err)
		}

		if got := countPlanNodes(mustAnalyze(t, havingMax, version), "Aggregate", ""); got != 1 {
			t.Errorf("HAVING MAX v%d Aggregate count = %d, want 1", version, got)
		}
		if got := countPlanNodes(mustAnalyze(t, havingMin, version), "Aggregate", ""); got != 1 {
			t.Errorf("HAVING MIN v%d Aggregate count = %d, want 1", version, got)
		}
		arrayAggPlan := mustAnalyze(t, arrayAgg, version)
		if got := countPlanNodes(arrayAggPlan, "Sort Limit", ""); got != 1 {
			t.Errorf("ARRAY_AGG ORDER BY LIMIT v%d Sort Limit count = %d, want 1", version, got)
		}
		if got := countPlanNodes(arrayAggPlan, "Array Unnest", ""); got != 1 {
			t.Errorf("ARRAY_AGG ORDER BY LIMIT v%d Array Unnest count = %d, want 1", version, got)
		}
		arrayTransformPlan := mustAnalyze(t, arrayTransform, version)
		if got := countPlanNodesByKind(arrayTransformPlan, spannerpb.PlanNode_SCALAR, "Array Subquery"); got != 1 {
			t.Errorf("ARRAY_TRANSFORM v%d Array Subquery count = %d, want 1", version, got)
		}
		if got := countPlanNodes(arrayTransformPlan, "Array Unnest", ""); got != 1 {
			t.Errorf("ARRAY_TRANSFORM v%d Array Unnest count = %d, want 1", version, got)
		}
		if got := countPlanNodes(arrayTransformPlan, "Filter", ""); got != 0 {
			t.Errorf("ARRAY_TRANSFORM v%d Filter count = %d, want 0", version, got)
		}
		arrayFilterPlan := mustAnalyze(t, arrayFilter, version)
		if got := countPlanNodesByKind(arrayFilterPlan, spannerpb.PlanNode_SCALAR, "Array Subquery"); got != 1 {
			t.Errorf("ARRAY_FILTER v%d Array Subquery count = %d, want 1", version, got)
		}
		if got := countPlanNodes(arrayFilterPlan, "Array Unnest", ""); got != 1 {
			t.Errorf("ARRAY_FILTER v%d Array Unnest count = %d, want 1", version, got)
		}
		if got := countPlanNodes(arrayFilterPlan, "Filter", ""); got != 1 {
			t.Errorf("ARRAY_FILTER v%d Filter count = %d, want 1", version, got)
		}
		if got := countPlanNodes(mustAnalyze(t, inUnnest, version), "Filter Scan", ""); got != 1 {
			t.Errorf("IN UNNEST v%d Filter Scan count = %d, want 1", version, got)
		}
		_ = mustAnalyze(t, withExpression, version)
		if got := countPlanNodes(mustAnalyze(t, collate, version), "Sort", ""); got != 1 {
			t.Errorf("ORDER BY COLLATE v%d Sort count = %d, want 1", version, got)
		}

		countDistinctPlan := mustAnalyze(t, countDistinct, version)
		if got := countPlanNodes(countDistinctPlan, "Aggregate", ""); got != 2 {
			t.Errorf("COUNT DISTINCT v%d Aggregate count = %d, want 2", version, got)
		}
		wantMinorSort := 0
		if version >= 5 {
			wantMinorSort = 1
		}
		if got := countPlanNodes(countDistinctPlan, "Minor Sort", ""); got != wantMinorSort {
			t.Errorf("COUNT DISTINCT v%d Minor Sort count = %d, want %d", version, got, wantMinorSort)
		}

		ignorePlan := mustAnalyze(t, arrayDistinctIgnore, version)
		respectPlan := mustAnalyze(t, arrayDistinctRespect, version)
		for label, plan := range map[string]*spannerpb.QueryPlan{"IGNORE NULLS": ignorePlan, "RESPECT NULLS": respectPlan} {
			if got := countPlanNodes(plan, "Aggregate", ""); got != 3 {
				t.Errorf("ARRAY_AGG DISTINCT %s v%d Aggregate count = %d, want 3", label, version, got)
			}
			if got := countPlanNodes(plan, "Array Unnest", ""); got != 1 {
				t.Errorf("ARRAY_AGG DISTINCT %s v%d Array Unnest count = %d, want 1", label, version, got)
			}
		}
		if got := countPlanNodes(ignorePlan, "Filter", ""); got != 1 {
			t.Errorf("ARRAY_AGG DISTINCT IGNORE NULLS v%d Filter count = %d, want 1", version, got)
		}
		if got := countPlanNodes(respectPlan, "Filter", ""); got != 0 {
			t.Errorf("ARRAY_AGG DISTINCT RESPECT NULLS v%d Filter count = %d, want 0", version, got)
		}

		notDistinctPlan := mustAnalyze(t, notDistinct, version)
		if got := countPlanNodesWithDescription(notDistinctPlan, "Constant", "true"); got != 1 {
			t.Errorf("IS NOT DISTINCT FROM v%d true Constant count = %d, want 1", version, got)
		}
		distinctFromPlan := mustAnalyze(t, distinctFrom, version)
		if got := countPlanNodesWithDescription(distinctFromPlan, "Constant", "false"); got != 1 {
			t.Errorf("IS DISTINCT FROM v%d false Constant count = %d, want 1", version, got)
		}

		explicitUnnestPlan := mustAnalyze(t, explicitUnnest, version)
		implicitUnnestPlan := mustAnalyze(t, implicitUnnest, version)
		if !proto.Equal(explicitUnnestPlan, implicitUnnestPlan) {
			t.Errorf("explicit and implicit correlated UNNEST plans differ at v%d", version)
		}
		if got := countPlanNodes(explicitUnnestPlan, "Cross Apply", ""); got != 1 {
			t.Errorf("correlated UNNEST v%d Cross Apply count = %d, want 1", version, got)
		}
		if got := countPlanNodes(explicitUnnestPlan, "Array Unnest", ""); got != 1 {
			t.Errorf("correlated UNNEST v%d Array Unnest count = %d, want 1", version, got)
		}
		leftUnnestPlan := mustAnalyze(t, leftUnnest, version)
		if got := countPlanNodes(leftUnnestPlan, "Outer Apply", ""); got != 1 {
			t.Errorf("LEFT correlated UNNEST v%d Outer Apply count = %d, want 1", version, got)
		}
		if got := countPlanNodes(leftUnnestPlan, "Array Unnest", ""); got != 1 {
			t.Errorf("LEFT correlated UNNEST v%d Array Unnest count = %d, want 1", version, got)
		}

		inSubqueryPlan := mustAnalyze(t, inSubquery, version)
		inArraySubqueryPlan := mustAnalyze(t, inArraySubquery, version)
		if !proto.Equal(inSubqueryPlan, inArraySubqueryPlan) {
			t.Errorf("IN subquery and IN UNNEST(ARRAY(subquery)) plans differ at v%d", version)
		}
		notInSubqueryPlan := mustAnalyze(t, notInSubquery, version)
		notInArraySubqueryPlan := mustAnalyze(t, notInArraySubquery, version)
		if !proto.Equal(notInSubqueryPlan, notInArraySubqueryPlan) {
			t.Errorf("NOT IN subquery and NOT IN UNNEST(ARRAY(subquery)) plans differ at v%d", version)
		}
		if version <= 4 {
			if got := countPlanNodesWithMetadata(inSubqueryPlan, "Hash Join", map[string]string{"join_type": "BUILD_SEMI"}); got != 1 {
				t.Errorf("IN subquery v%d BUILD_SEMI Hash Join count = %d, want 1", version, got)
			}
			if got := countPlanNodesWithMetadata(notInSubqueryPlan, "Hash Join", map[string]string{"join_type": "BUILD_ANTI_SEMI"}); got != 1 {
				t.Errorf("NOT IN subquery v%d BUILD_ANTI_SEMI Hash Join count = %d, want 1", version, got)
			}
		} else {
			if got := countPlanNodes(inSubqueryPlan, "Semi Apply", ""); got != 1 {
				t.Errorf("IN subquery v%d Semi Apply count = %d, want 1", version, got)
			}
			if got := countPlanNodes(notInSubqueryPlan, "Anti-Semi Apply", ""); got != 1 {
				t.Errorf("NOT IN subquery v%d Anti-Semi Apply count = %d, want 1", version, got)
			}
		}

		if !proto.Equal(mustAnalyze(t, groupOrdinal, version), mustAnalyze(t, groupName, version)) {
			t.Errorf("GROUP BY ordinal and name-control plans differ at v%d", version)
		}
		tablesampleSubqueryPlan := mustAnalyze(t, tablesampleSubquery, version)
		if got := countPlanNodes(tablesampleSubqueryPlan, "Random Id Assign", ""); got != 1 {
			t.Errorf("subquery TABLESAMPLE v%d Random Id Assign count = %d, want 1", version, got)
		}
		if got := countPlanNodes(tablesampleSubqueryPlan, "Filter", ""); got != 1 {
			t.Errorf("subquery TABLESAMPLE v%d Filter count = %d, want 1", version, got)
		}
		tablesampleJoinPlan := mustAnalyze(t, tablesampleJoin, version)
		if got := countPlanNodes(tablesampleJoinPlan, "Random Id Assign", ""); got != 1 {
			t.Errorf("joined TABLESAMPLE v%d Random Id Assign count = %d, want 1", version, got)
		}
		if got := countPlanNodes(tablesampleJoinPlan, "Distributed Cross Apply", ""); got != 1 {
			t.Errorf("joined TABLESAMPLE v%d Distributed Cross Apply count = %d, want 1", version, got)
		}
		wantGlobalLimit, wantGlobalSortLimit := 1, 1
		if version <= 2 {
			wantGlobalLimit, wantGlobalSortLimit = 0, 2
		}
		if got := countPlanNodes(tablesampleJoinPlan, "Limit", ""); got != wantGlobalLimit {
			t.Errorf("joined TABLESAMPLE v%d Limit count = %d, want %d", version, got, wantGlobalLimit)
		}
		if got := countPlanNodes(tablesampleJoinPlan, "Sort Limit", ""); got != wantGlobalSortLimit {
			t.Errorf("joined TABLESAMPLE v%d Sort Limit count = %d, want %d", version, got, wantGlobalSortLimit)
		}

		if got := countPlanNodes(mustAnalyze(t, structStar, version), "Unit Relation", ""); got != 1 {
			t.Errorf("STRUCT expression.* v%d Unit Relation count = %d, want 1", version, got)
		}
		arrayStructUnnestPlan := mustAnalyze(t, arrayStructUnnest, version)
		if got := countPlanNodes(arrayStructUnnestPlan, "Array Unnest", ""); got != 1 {
			t.Errorf("array-of-STRUCT UNNEST v%d Array Unnest count = %d, want 1", version, got)
		}
		if got := countPlanNodesByKind(arrayStructUnnestPlan, spannerpb.PlanNode_SCALAR, "Struct Constructor"); got != 2 {
			t.Errorf("array-of-STRUCT UNNEST v%d Struct Constructor count = %d, want 2", version, got)
		}
		for label, plan := range map[string]*spannerpb.QueryPlan{
			"regular-first top-level": mustAnalyze(t, regularFirstSetOperation, version),
			"value-first nested":      mustAnalyze(t, valueFirstSetOperationNested, version),
		} {
			if got := countPlanNodes(plan, "Union All", ""); got != 1 {
				t.Errorf("%s set operation v%d Union All count = %d, want 1", label, version, got)
			}
			if got := countPlanNodes(plan, "Union Input", ""); got != 2 {
				t.Errorf("%s set operation v%d Union Input count = %d, want 2", label, version, got)
			}
		}

		if _, err := analyze(t, analyticRowNumber, version); err == nil || !strings.Contains(err.Error(), "Unsupported built-in function: ROW_NUMBER") {
			t.Errorf("analytic ROW_NUMBER v%d error = %v, want stable unsupported-function error", version, err)
		}
		if _, err := analyze(t, analyticFramedSum, version); err == nil || !strings.Contains(err.Error(), "Unsupported built-in function: SUM") {
			t.Errorf("analytic framed SUM v%d error = %v, want stable unsupported-function error", version, err)
		}
		if _, err := analyze(t, repeatable, version); err == nil || !strings.Contains(err.Error(), "REPEATABLE TABLESAMPLE is not supported") {
			t.Errorf("TABLESAMPLE REPEATABLE v%d error = %v, want stable capability error", version, err)
		}
		for label, query := range map[string]queryCase{
			"top-level SELECT AS STRUCT":   selectAsStruct,
			"value-first top-level set op": valueFirstSetOperation,
		} {
			if _, err := analyze(t, query, version); err == nil || !strings.Contains(err.Error(), "Top level queries may not specify value tables") {
				t.Errorf("%s v%d error = %v, want stable value-table result-shape error", label, version, err)
			}
		}
		for label, query := range map[string]queryCase{
			"CROSS JOIN LATERAL": lateralCross,
			"LEFT JOIN LATERAL":  lateralLeft,
		} {
			if _, err := analyze(t, query, version); err == nil || !strings.Contains(err.Error(), "LATERAL join is not supported") {
				t.Errorf("%s v%d error = %v, want stable unsupported error", label, version, err)
			}
		}
		if _, err := analyze(t, setOperationByName, version); err == nil || !strings.Contains(err.Error(), "BY NAME for set operations is not supported") {
			t.Errorf("set-operation BY NAME v%d error = %v, want stable unsupported error", version, err)
		}
		if _, err := analyze(t, setOperationCorresponding, version); err == nil || !strings.Contains(err.Error(), "CORRESPONDING for set operations is not supported") {
			t.Errorf("set-operation CORRESPONDING v%d error = %v, want stable unsupported error", version, err)
		}
		if _, err := analyze(t, groupByAll, version); err == nil || !strings.Contains(err.Error(), "GROUP BY ALL is not supported") {
			t.Errorf("GROUP BY ALL v%d error = %v, want stable unsupported error", version, err)
		}
		if _, err := analyze(t, matchRecognize, version); err == nil || !strings.Contains(err.Error(), "Syntax error:") {
			t.Errorf("MATCH_RECOGNIZE v%d error = %v, want stable syntax error", version, err)
		}
		if _, err := analyze(t, secureContext, version); err == nil || !strings.Contains(err.Error(), "Unsupported built-in function: SECURE_CONTEXT") {
			t.Errorf("SECURE_CONTEXT v%d error = %v, want stable unsupported-function error", version, err)
		}
	}

	assertValueTableSetOperationDirectionOnOmni(t, clients.Client)
	assertCorrelatedUnnestResultsOnOmni(t, clients.Client)
}

func TestIntegrationGoogleSQLProtoSurfaceVersionMatrixOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	image := strings.TrimSpace(os.Getenv("SPANALYZER_OMNI_IMAGE"))
	if image == "" {
		t.Fatal("set SPANALYZER_OMNI_IMAGE to the pinned Spanner Omni image under test")
	}
	descriptors, err := loadFileDescriptorSet([]string{
		"../../testdata/protos/order_descriptors.pb",
		"../../testdata/protos/complex/complex_descriptors.pb",
	})
	if err != nil {
		t.Fatalf("loadFileDescriptorSet() error = %v", err)
	}
	ddls, err := loadDDLs("google_sql_proto_surface", nil)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	runtime := spanemuboost.NewLazyRuntime(
		spanemuboost.BackendOmni,
		spanemuboost.WithContainerImage(image),
	)
	t.Cleanup(func() { _ = runtime.Close() })
	clients, err := spanemuboost.OpenClients(t.Context(), runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(ddls),
		spanemuboost.WithSetupFileDescriptorSet(descriptors),
	)
	if err != nil {
		t.Fatalf("OpenClients() error = %v", err)
	}
	t.Cleanup(func() { _ = clients.Close() })

	unsupportedErrors := map[string]string{
		"google-sql-proto-surface/unsupported/extract-field":             "EXTRACT from PROTO is not supported",
		"google-sql-proto-surface/unsupported/extract-presence":          "EXTRACT from PROTO is not supported",
		"google-sql-proto-surface/unsupported/extract-raw-field":         "EXTRACT from PROTO is not supported",
		"google-sql-proto-surface/unsupported/proto-default-if-null":     "Function not found: PROTO_DEFAULT_IF_NULL",
		"google-sql-proto-surface/unsupported/filter-fields":             "Function not found: FILTER_FIELDS",
		"google-sql-proto-surface/unsupported/extract-oneof-case":        "EXTRACT from PROTO is not supported",
		"google-sql-proto-surface/unsupported/select-as-proto-top-level": "Top level queries may not specify value tables",
	}
	baselinePlans := make(map[string]*spannerpb.QueryPlan)

	for version := 1; version <= 8; version++ {
		for _, query := range googleSQLProtoSurfaceQueries {
			query.SQL = withOptimizerVersionStatementHint(query.SQL, version)
			plan, err := analyzePlan(t.Context(), clients.Client, query)
			if strings.HasPrefix(query.Label, "google-sql-proto-surface/unsupported/") {
				want, ok := unsupportedErrors[query.Label]
				if !ok {
					t.Errorf("%s has no expected error", query.Label)
					continue
				}
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Errorf("AnalyzeQuery(%s, v%d) error = %v, want error containing %q", query.Label, version, err, want)
				}
				continue
			}
			if err != nil {
				t.Errorf("AnalyzeQuery(%s, v%d) error = %v", query.Label, version, err)
				continue
			}
			if baseline := baselinePlans[query.Label]; baseline == nil {
				baselinePlans[query.Label] = proto.Clone(plan).(*spannerpb.QueryPlan)
			} else if !proto.Equal(baseline, plan) {
				t.Errorf("AnalyzeQuery(%s, v%d) plan differs from v1", query.Label, version)
			}
			switch query.Label {
			case "google-sql-proto-surface/accepted/select-as-proto-distinct-nested":
				if got := countPlanNodes(plan, "Aggregate", ""); got != 2 {
					t.Errorf("%s v%d Aggregate count = %d, want 2", query.Label, version, got)
				}
				if got := countPlanNodes(plan, "Compute", ""); got != 1 {
					t.Errorf("%s v%d Compute count = %d, want 1", query.Label, version, got)
				}
			case "google-sql-proto-surface/accepted/proto-repeated-field-unnest":
				if got := countPlanNodes(plan, "Cross Apply", ""); got != 1 {
					t.Errorf("%s v%d Cross Apply count = %d, want 1", query.Label, version, got)
				}
				if got := countPlanNodes(plan, "Array Unnest", ""); got != 1 {
					t.Errorf("%s v%d Array Unnest count = %d, want 1", query.Label, version, got)
				}
			}
			t.Logf("%s v%d: %s", query.Label, version, compactPlanTree(plan, true, false))
		}
	}
	if got, want := len(baselinePlans), 11; got != want {
		t.Errorf("accepted proto plan count = %d, want %d", got, want)
	}
}

func assertValueTableSetOperationDirectionOnOmni(t *testing.T, client *spanner.Client) {
	t.Helper()
	tests := []struct {
		name         string
		regularFirst string
		valueFirst   string
		want         []int64
	}{
		{
			name:         "union-all",
			regularFirst: `SELECT 1 AS x UNION ALL SELECT AS VALUE 2 ORDER BY x`,
			valueFirst:   `SELECT AS VALUE 1 UNION ALL SELECT 2 AS x`,
			want:         []int64{1, 2},
		},
		{
			name:         "union-distinct",
			regularFirst: `SELECT 1 AS x UNION DISTINCT SELECT AS VALUE 1 ORDER BY x`,
			valueFirst:   `SELECT AS VALUE 1 UNION DISTINCT SELECT 1 AS x`,
			want:         []int64{1},
		},
		{
			name:         "intersect-all",
			regularFirst: `SELECT 1 AS x INTERSECT ALL SELECT AS VALUE 1 ORDER BY x`,
			valueFirst:   `SELECT AS VALUE 1 INTERSECT ALL SELECT 1 AS x`,
			want:         []int64{1},
		},
		{
			name:         "intersect-distinct",
			regularFirst: `SELECT 1 AS x INTERSECT DISTINCT SELECT AS VALUE 1 ORDER BY x`,
			valueFirst:   `SELECT AS VALUE 1 INTERSECT DISTINCT SELECT 1 AS x`,
			want:         []int64{1},
		},
		{
			name:         "except-all",
			regularFirst: `SELECT 1 AS x EXCEPT ALL SELECT AS VALUE 2 ORDER BY x`,
			valueFirst:   `SELECT AS VALUE 1 EXCEPT ALL SELECT 2 AS x`,
			want:         []int64{1},
		},
		{
			name:         "except-distinct",
			regularFirst: `SELECT 1 AS x EXCEPT DISTINCT SELECT AS VALUE 2 ORDER BY x`,
			valueFirst:   `SELECT AS VALUE 1 EXCEPT DISTINCT SELECT 2 AS x`,
			want:         []int64{1},
		},
	}

	for version := 1; version <= 8; version++ {
		for _, tt := range tests {
			regularSQL := withOptimizerVersionStatementHint(tt.regularFirst, version)
			if got := readInt64s(t, client, regularSQL); !slices.Equal(got, tt.want) {
				t.Errorf("%s regular-first v%d result = %v, want %v", tt.name, version, got, tt.want)
			}

			valueSQL := withOptimizerVersionStatementHint(tt.valueFirst, version)
			err := client.Single().Query(t.Context(), spanner.Statement{SQL: valueSQL}).Do(func(*spanner.Row) error { return nil })
			if err == nil || !strings.Contains(err.Error(), "Top level queries may not specify value tables") {
				t.Errorf("%s value-first v%d error = %v, want top-level value-table error", tt.name, version, err)
			}
		}

		nestedSQL := withOptimizerVersionStatementHint(`SELECT value_column FROM (SELECT AS VALUE 1 UNION ALL SELECT 2 AS x) AS value_column ORDER BY value_column`, version)
		if got, want := readInt64s(t, client, nestedSQL), []int64{1, 2}; !slices.Equal(got, want) {
			t.Errorf("nested value-first UNION ALL v%d result = %v, want %v", version, got, want)
		}
	}
}

func assertCorrelatedUnnestResultsOnOmni(t *testing.T, client *spanner.Client) {
	t.Helper()
	tests := []struct {
		name string
		sql  string
		want []string
	}{
		{
			name: "explicit",
			sql:  `SELECT c.VenueId, price FROM Concerts AS c CROSS JOIN UNNEST(c.TicketPrices) AS price ORDER BY c.VenueId, price`,
			want: []string{"2:10", "2:20"},
		},
		{
			name: "implicit",
			sql:  `SELECT c.VenueId, price FROM Concerts AS c, c.TicketPrices AS price ORDER BY c.VenueId, price`,
			want: []string{"2:10", "2:20"},
		},
		{
			name: "left",
			sql:  `SELECT c.VenueId, price FROM Concerts AS c LEFT JOIN UNNEST(c.TicketPrices) AS price ON TRUE ORDER BY c.VenueId, price`,
			want: []string{"1:NULL", "2:10", "2:20"},
		},
	}

	for version := 1; version <= 8; version++ {
		var explicit, implicit []string
		for _, tt := range tests {
			got := readVenuePrices(t, client, withOptimizerVersionStatementHint(tt.sql, version))
			if !slices.Equal(got, tt.want) {
				t.Errorf("correlated UNNEST %s v%d result = %v, want %v", tt.name, version, got, tt.want)
			}
			switch tt.name {
			case "explicit":
				explicit = got
			case "implicit":
				implicit = got
			}
		}
		if !slices.Equal(explicit, implicit) {
			t.Errorf("explicit and implicit correlated UNNEST results differ at v%d: explicit=%v implicit=%v", version, explicit, implicit)
		}
	}
}

func readVenuePrices(t *testing.T, client *spanner.Client, sql string) []string {
	t.Helper()
	var got []string
	err := client.Single().Query(t.Context(), spanner.Statement{SQL: sql}).Do(func(row *spanner.Row) error {
		var venueID int64
		var price spanner.NullInt64
		if err := row.Columns(&venueID, &price); err != nil {
			return err
		}
		priceText := "NULL"
		if price.Valid {
			priceText = strconv.FormatInt(price.Int64, 10)
		}
		got = append(got, strconv.FormatInt(venueID, 10)+":"+priceText)
		return nil
	})
	if err != nil {
		t.Fatalf("query failed: %v\nSQL: %s", err, sql)
	}
	return got
}

func countPlanNodesWithDescription(plan *spannerpb.QueryPlan, displayName, description string) int {
	count := 0
	for _, node := range plan.GetPlanNodes() {
		if node.GetDisplayName() == displayName && node.GetShortRepresentation().GetDescription() == description {
			count++
		}
	}
	return count
}

func countPlanNodesByKind(plan *spannerpb.QueryPlan, kind spannerpb.PlanNode_Kind, displayName string) int {
	count := 0
	for _, node := range plan.GetPlanNodes() {
		if node.GetKind() == kind && node.GetDisplayName() == displayName {
			count++
		}
	}
	return count
}
