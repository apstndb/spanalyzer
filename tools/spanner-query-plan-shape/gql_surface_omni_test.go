//go:build integration && omni

package main

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/proto"
)

func TestIntegrationGQLCompositionalSurfaceOnOmni(t *testing.T) {
	ddls, err := parseBuiltInDDLs("gql-surface-schema.sql", docsDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	clients := openOmniClients(t, ddls)

	cases := queryCasesByLabel(t, gqlSurfaceQueries)
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

	isFirstReturn := cases[gqlSurfaceISFirstReturnLabel]
	isFirstFilter := cases[gqlSurfaceISFirstFilterLabel]
	isFirstEdgeOneHop := cases[gqlSurfaceISFirstEdgeOneHopLabel]
	edgeIn := cases[gqlSurfaceISFirstQuantifiedLabel]
	edgeControl := cases["gql-surface/search/all-bounded"]
	isFirstNext := cases[gqlSurfaceISFirstBeforeNextLabel]
	isFirstNextOrdered := cases[gqlSurfaceISFirstBeforeNextOrderedLabel]
	nextControl := cases["gql-surface/linear/next-two-stage-traversal"]
	reservoir := cases["gql-surface/linear/tablesample-reservoir"]
	bernoulli := cases["gql-surface/linear/tablesample-bernoulli"]
	withWeight := cases["gql-surface/unsupported/tablesample-with-weight"]
	ordered := cases["gql-surface/aggregate/horizontal-array-agg-ordered"]
	unordered := cases["gql-surface/aggregate/horizontal-array-agg-unordered-control"]
	stringOrdered := cases["gql-surface/aggregate/horizontal-string-agg-ordered"]
	stringUnordered := cases["gql-surface/aggregate/horizontal-string-agg-unordered-control"]
	stringReturn := cases["gql-surface/unsupported/horizontal-string-agg-return"]
	arrayConcatOrdered := cases["gql-surface/aggregate/horizontal-array-concat-agg-ordered"]
	arrayConcatReturn := cases["gql-surface/unsupported/vertical-array-concat-agg-group-variable"]
	existsMatch := cases["gql-surface/subquery/exists-match-body"]
	existsPattern := cases["gql-surface/subquery/exists-pattern-body"]
	existsPatternFilter := cases["gql-surface/subquery/exists-pattern-filter"]
	existsFullFilter := cases["gql-surface/subquery/exists-full-filter-control"]
	dmlRecursive := cases["gql-surface/bridge/dml-update-recursive-in"]
	labelOR := cases["gql-surface/pattern/label-or"]
	labelAND := cases["gql-surface/pattern/label-and"]
	labelANDControl := cases["gql-surface/pattern/label-and-single-control"]
	malformedLabelOR := cases["gql-surface/unsupported/malformed-label-or"]
	primitiveOrder := cases["gql-surface/pagination/primitive-order-by-limit"]
	primitiveCollate := cases["gql-surface/pagination/primitive-order-by-collate-limit"]
	groupFilter := cases["gql-surface/with/group-filter"]
	implicitGrouping := cases["gql-surface/with/implicit-grouping"]
	horizontalDistinct := cases["gql-surface/aggregate/horizontal-count-distinct"]
	horizontalCount := cases["gql-surface/aggregate/horizontal-count-control"]
	repeatable := cases["gql-surface/unsupported/tablesample-repeatable"]

	for version := 1; version <= 8; version++ {
		returnPlan := mustAnalyze(t, isFirstReturn, version)
		if got := countPlanNodes(returnPlan, "Crowd", ""); got != 1 {
			t.Errorf("IS_FIRST RETURN v%d Crowd count = %d, want 1", version, got)
		}
		for _, test := range []struct {
			label string
			query queryCase
		}{
			{label: "IS_FIRST FILTER", query: isFirstFilter},
			{label: "one-hop edge IN/IS_FIRST", query: isFirstEdgeOneHop},
		} {
			plan := mustAnalyze(t, test.query, version)
			// v6 adds an outer Crowd while preserving the result rows; retain
			// the observed version boundary instead of flattening it to >= 2.
			wantCrowds := 2
			if version >= 6 {
				wantCrowds = 3
			}
			if got := countPlanNodes(plan, "Crowd", ""); got != wantCrowds {
				t.Errorf("%s v%d Crowd count = %d, want %d", test.label, version, got, wantCrowds)
			}
		}

		edgePlan := mustAnalyze(t, edgeIn, version)
		edgeControlPlan := mustAnalyze(t, edgeControl, version)
		if got := countPlanNodes(edgePlan, "Recursive Union", ""); got != 1 {
			t.Errorf("edge IN/IS_FIRST v%d Recursive Union count = %d, want 1", version, got)
		}
		if got := countPlanNodes(edgePlan, "Limit", ""); got == 0 {
			t.Errorf("edge IN/IS_FIRST v%d Limit count = 0, want positive", version)
		}
		if got := countPlanNodes(edgePlan, "Crowd", ""); got != 0 {
			t.Errorf("quantified edge IN/IS_FIRST v%d Crowd count = %d, want 0", version, got)
		}
		if got := countPlanNodes(edgeControlPlan, "Limit", ""); got != 0 {
			t.Errorf("quantified control v%d Limit count = %d, want 0", version, got)
		}

		isFirstPlan := mustAnalyze(t, isFirstNext, version)
		nextControlPlan := mustAnalyze(t, nextControl, version)
		if got := countPlanNodes(isFirstPlan, "Crowd", ""); got != 2 {
			t.Errorf("IS_FIRST/NEXT v%d Crowd count = %d, want 2", version, got)
		}
		if got := countPlanNodes(nextControlPlan, "Crowd", ""); got != 0 {
			t.Errorf("NEXT control v%d Crowd count = %d, want 0", version, got)
		}
		orderedNextPlan := mustAnalyze(t, isFirstNextOrdered, version)
		if got := countPlanNodes(orderedNextPlan, "Crowd", ""); got < 2 {
			t.Errorf("ordered IS_FIRST/NEXT v%d Crowd count = %d, want at least 2", version, got)
		}

		reservoirPlan := mustAnalyze(t, reservoir, version)
		bernoulliPlan := mustAnalyze(t, bernoulli, version)
		for label, plan := range map[string]*spannerpb.QueryPlan{"RESERVOIR": reservoirPlan, "BERNOULLI": bernoulliPlan} {
			if got := countPlanNodes(plan, "Random Id Assign", ""); got != 1 {
				t.Errorf("%s v%d Random Id Assign count = %d, want 1", label, version, got)
			}
		}
		if got := countPlanNodes(reservoirPlan, "Sort Limit", ""); got == 0 {
			t.Errorf("RESERVOIR v%d Sort Limit count = 0, want positive", version)
		}
		if got := countPlanNodes(reservoirPlan, "Filter", ""); got != 0 {
			t.Errorf("RESERVOIR v%d Filter count = %d, want 0", version, got)
		}
		if got := countPlanNodes(bernoulliPlan, "Sort Limit", ""); got != 0 {
			t.Errorf("BERNOULLI v%d Sort Limit count = %d, want 0", version, got)
		}
		if got := countPlanNodes(bernoulliPlan, "Filter", ""); got != 1 {
			t.Errorf("BERNOULLI v%d Filter count = %d, want 1", version, got)
		}

		if _, err := analyze(t, withWeight, version); err == nil || !strings.Contains(err.Error(), "TABLESAMPLE WITH WEIGHT is not supported.") {
			t.Errorf("TABLESAMPLE WITH WEIGHT v%d error = %v, want stable capability error", version, err)
		}

		orderedPlan := mustAnalyze(t, ordered, version)
		unorderedPlan := mustAnalyze(t, unordered, version)
		for label, plan := range map[string]*spannerpb.QueryPlan{"ordered": orderedPlan, "unordered": unorderedPlan} {
			if got := countPlanNodes(plan, "Aggregate", ""); got != 3 {
				t.Errorf("horizontal ARRAY_AGG %s v%d Aggregate count = %d, want 3", label, version, got)
			}
			if got := countPlanNodes(plan, "Array Unnest", ""); got != 2 {
				t.Errorf("horizontal ARRAY_AGG %s v%d Array Unnest count = %d, want 2", label, version, got)
			}
		}
		orderedSort := singleRelationalNode(t, orderedPlan, "Sort")
		unorderedSort := singleRelationalNode(t, unorderedPlan, "Sort")
		if got := countScalarChildLinks(orderedPlan, orderedSort, "Key"); got != 1 {
			t.Errorf("ordered horizontal ARRAY_AGG v%d Sort Key links = %d, want 1", version, got)
		}
		if got := countScalarChildLinks(orderedPlan, orderedSort, "Value"); got != 0 {
			t.Errorf("ordered horizontal ARRAY_AGG v%d Sort Value links = %d, want 0", version, got)
		}
		if got := countScalarChildLinks(unorderedPlan, unorderedSort, "Key"); got != 1 {
			t.Errorf("unordered horizontal ARRAY_AGG v%d Sort Key links = %d, want 1", version, got)
		}
		if got := countScalarChildLinks(unorderedPlan, unorderedSort, "Value"); got != 1 {
			t.Errorf("unordered horizontal ARRAY_AGG v%d Sort Value links = %d, want 1", version, got)
		}

		stringOrderedPlan := mustAnalyze(t, stringOrdered, version)
		stringUnorderedPlan := mustAnalyze(t, stringUnordered, version)
		for label, plan := range map[string]*spannerpb.QueryPlan{"ordered": stringOrderedPlan, "unordered": stringUnorderedPlan} {
			if got := countPlanNodes(plan, "Aggregate", ""); got != 3 {
				t.Errorf("horizontal STRING_AGG %s v%d Aggregate count = %d, want 3", label, version, got)
			}
			if got := countPlanNodes(plan, "Array Unnest", ""); got != 2 {
				t.Errorf("horizontal STRING_AGG %s v%d Array Unnest count = %d, want 2", label, version, got)
			}
			if !planHasScalarFunctionType(plan, "ARRAY_TO_STRING") {
				t.Errorf("horizontal STRING_AGG %s v%d lacks ARRAY_TO_STRING lowering", label, version)
			}
		}
		stringOrderedSort := singleRelationalNode(t, stringOrderedPlan, "Sort")
		stringUnorderedSort := singleRelationalNode(t, stringUnorderedPlan, "Sort")
		if got := countScalarChildLinks(stringOrderedPlan, stringOrderedSort, "Value"); got != 0 {
			t.Errorf("ordered horizontal STRING_AGG v%d Sort Value links = %d, want 0", version, got)
		}
		if got := countScalarChildLinks(stringUnorderedPlan, stringUnorderedSort, "Value"); got != 1 {
			t.Errorf("unordered horizontal STRING_AGG v%d Sort Value links = %d, want 1", version, got)
		}
		for _, query := range []queryCase{stringReturn, arrayConcatReturn} {
			if _, err := analyze(t, query, version); err == nil || !strings.Contains(err.Error(), "Cannot access field AlbumTitle on a value with type ARRAY<GRAPH_EDGE") {
				t.Errorf("AnalyzeQuery(%s, v%d) error = %v, want group-variable field-access error", query.Label, version, err)
			}
		}

		arrayConcatPlan := mustAnalyze(t, arrayConcatOrdered, version)
		for displayName, want := range map[string]int{"Aggregate": 3, "Array Unnest": 4, "Minor Sort": 1} {
			if got := countPlanNodes(arrayConcatPlan, displayName, ""); got != want {
				t.Errorf("horizontal ARRAY_CONCAT_AGG v%d %s count = %d, want %d", version, displayName, got, want)
			}
		}
		if got := countPlanNodesAnyKind(arrayConcatPlan, "Array Subquery"); got != 1 {
			t.Errorf("horizontal ARRAY_CONCAT_AGG v%d Array Subquery count = %d, want 1", version, got)
		}
		if !planHasScalarFunctionType(arrayConcatPlan, "ARRAY_CONCAT") {
			t.Errorf("horizontal ARRAY_CONCAT_AGG v%d lacks ARRAY_CONCAT lowering", version)
		}
		if proto.Equal(arrayConcatPlan, orderedPlan) {
			t.Errorf("horizontal ARRAY_CONCAT_AGG/ARRAY_AGG v%d plans unexpectedly equal", version)
		}

		existsMatchPlan := mustAnalyze(t, existsMatch, version)
		existsPatternPlan := mustAnalyze(t, existsPattern, version)
		if !proto.Equal(existsMatchPlan, existsPatternPlan) {
			t.Errorf("short EXISTS MATCH/pattern v%d plans differ", version)
		}
		wantExistsAggregates := 2
		if version <= 4 {
			wantExistsAggregates = 3
		} else if version == 5 {
			wantExistsAggregates = 1
		}
		if got := countPlanNodes(existsMatchPlan, "Aggregate", ""); got != wantExistsAggregates {
			t.Errorf("short EXISTS v%d Aggregate count = %d, want %d", version, got, wantExistsAggregates)
		}
		existsPatternFilterPlan := mustAnalyze(t, existsPatternFilter, version)
		existsFullFilterPlan := mustAnalyze(t, existsFullFilter, version)
		if !proto.Equal(existsPatternFilterPlan, existsFullFilterPlan) {
			t.Errorf("FILTER EXISTS pattern/full v%d plans differ", version)
		}
		if got := countPlanNodes(existsPatternFilterPlan, "Distributed Semi Apply", ""); got != 1 {
			t.Errorf("FILTER EXISTS v%d Distributed Semi Apply count = %d, want 1", version, got)
		}

		dmlPlan := mustAnalyze(t, dmlRecursive, version)
		for displayName, want := range map[string]int{"Apply Mutations": 1, "Recursive Union": 1, "Recursive Spool Scan": 1} {
			if got := countPlanNodes(dmlPlan, displayName, ""); got != want {
				t.Errorf("GQL-in-DML v%d %s count = %d, want %d", version, displayName, got, want)
			}
		}

		labelORPlan := mustAnalyze(t, labelOR, version)
		if got := countPlanNodes(labelORPlan, "Union All", ""); got != 1 {
			t.Errorf("label OR v%d Union All count = %d, want 1", version, got)
		}
		for target, want := range map[string]int{"Singers": 1, "Albums": 1} {
			if got := countPlanNodesWithMetadata(labelORPlan, "Scan", map[string]string{"scan_target": target, "scan_type": "TableScan"}); got != want {
				t.Errorf("label OR v%d %s TableScan count = %d, want %d", version, target, got, want)
			}
		}
		labelANDPlan := mustAnalyze(t, labelAND, version)
		labelANDControlPlan := mustAnalyze(t, labelANDControl, version)
		if !proto.Equal(labelANDPlan, labelANDControlPlan) {
			t.Errorf("label AND/single-label control v%d plans differ", version)
		}
		if got := countPlanNodesWithMetadata(labelANDPlan, "Scan", map[string]string{"scan_target": "SingersByFirstLastName", "scan_type": "IndexScan"}); got != 1 {
			t.Errorf("label AND v%d Singers covering IndexScan count = %d, want 1", version, got)
		}
		if _, err := analyze(t, malformedLabelOR, version); err == nil || !strings.Contains(err.Error(), "Unexpected") {
			t.Errorf("malformed label OR v%d error = %v, want stable syntax error", version, err)
		}

		for label, query := range map[string]queryCase{"primitive": primitiveOrder, "collate": primitiveCollate} {
			plan := mustAnalyze(t, query, version)
			wantSortLimit, wantLimit := 1, 1
			if version <= 2 {
				wantSortLimit, wantLimit = 2, 0
			}
			if got := countPlanNodes(plan, "Sort Limit", ""); got != wantSortLimit {
				t.Errorf("%s primitive ORDER BY v%d Sort Limit count = %d, want %d", label, version, got, wantSortLimit)
			}
			if got := countPlanNodes(plan, "Limit", ""); got != wantLimit {
				t.Errorf("%s primitive ORDER BY v%d Limit count = %d, want %d", label, version, got, wantLimit)
			}
		}

		groupFilterPlan := mustAnalyze(t, groupFilter, version)
		implicitPlan := mustAnalyze(t, implicitGrouping, version)
		if got := countPlanNodes(groupFilterPlan, "Filter", ""); got != 1 {
			t.Errorf("post-group FILTER v%d Filter count = %d, want 1", version, got)
		}
		if got := countPlanNodes(implicitPlan, "Filter", ""); got != 0 {
			t.Errorf("implicit grouping v%d Filter count = %d, want 0", version, got)
		}
		for label, plan := range map[string]*spannerpb.QueryPlan{"post-filter": groupFilterPlan, "implicit": implicitPlan} {
			if got := countPlanNodes(plan, "Aggregate", ""); got != 2 {
				t.Errorf("%s grouping v%d Aggregate count = %d, want 2", label, version, got)
			}
		}

		distinctPlan := mustAnalyze(t, horizontalDistinct, version)
		countPlan := mustAnalyze(t, horizontalCount, version)
		if got := countPlanNodes(distinctPlan, "Aggregate", ""); got != 3 {
			t.Errorf("horizontal COUNT DISTINCT v%d Aggregate count = %d, want 3", version, got)
		}
		if got := countPlanNodes(countPlan, "Aggregate", ""); got != 2 {
			t.Errorf("horizontal COUNT control v%d Aggregate count = %d, want 2", version, got)
		}
		if proto.Equal(distinctPlan, countPlan) {
			t.Errorf("horizontal COUNT DISTINCT/control v%d plans unexpectedly equal", version)
		}

		if _, err := analyze(t, repeatable, version); err == nil || !strings.Contains(err.Error(), "REPEATABLE TABLESAMPLE is not supported") {
			t.Errorf("GQL TABLESAMPLE REPEATABLE v%d error = %v, want stable capability error", version, err)
		}
	}
}

func singleRelationalNode(t *testing.T, plan *spannerpb.QueryPlan, displayName string) *spannerpb.PlanNode {
	t.Helper()
	var found *spannerpb.PlanNode
	for _, node := range plan.GetPlanNodes() {
		if node.GetKind() != spannerpb.PlanNode_RELATIONAL || node.GetDisplayName() != displayName {
			continue
		}
		if found != nil {
			t.Fatalf("plan contains more than one %s node", displayName)
		}
		found = node
	}
	if found == nil {
		t.Fatalf("plan contains no %s node", displayName)
	}
	return found
}

func countScalarChildLinks(plan *spannerpb.QueryPlan, parent *spannerpb.PlanNode, linkType string) int {
	nodes := make(map[int32]*spannerpb.PlanNode, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		nodes[node.GetIndex()] = node
	}
	count := 0
	for _, link := range parent.GetChildLinks() {
		child := nodes[link.GetChildIndex()]
		if child != nil && child.GetKind() == spannerpb.PlanNode_SCALAR && link.GetType() == linkType {
			count++
		}
	}
	return count
}
