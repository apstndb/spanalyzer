//go:build integration && omni

package main

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/proto"
)

func TestIntegrationGQLHintSurfaceVersionBoundariesOnOmni(t *testing.T) {
	ddls, err := parseBuiltInDDLs("gql-hint-surface-schema.sql", docsDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	clients := openOmniClients(t, ddls)

	hintCases := queryCasesByLabel(t, gqlHintSurfaceQueries)
	gqlCases := queryCasesByLabel(t, gqlSurfaceQueries)
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

	t.Run("factorized-both-v4-v5", func(t *testing.T) {
		for _, pair := range [][2]string{
			{"gql-hint/factorized/quantified-both", "gql-hint/factorized/quantified-control"},
			{"gql-hint/factorized/nonlinear-both", "gql-hint/factorized/nonlinear-control"},
		} {
			hintedV4 := mustAnalyze(t, hintCases[pair[0]], 4)
			controlV4 := mustAnalyze(t, hintCases[pair[1]], 4)
			if !proto.Equal(hintedV4, controlV4) {
				t.Errorf("v4 %s plan differs from control %s", pair[0], pair[1])
			}

			hintedV5 := mustAnalyze(t, hintCases[pair[0]], 5)
			controlV5 := mustAnalyze(t, hintCases[pair[1]], 5)
			if proto.Equal(hintedV5, controlV5) {
				t.Errorf("v5 %s plan unexpectedly equals control %s", pair[0], pair[1])
			}
			if gotHinted, gotControl := countPlanNodes(hintedV5, "Array Unnest", ""), countPlanNodes(controlV5, "Array Unnest", ""); gotHinted <= gotControl {
				t.Errorf("v5 %s Array Unnest count = %d, control = %d; want hinted > control", pair[0], gotHinted, gotControl)
			}
		}
	})

	t.Run("exists-one-pass-boundary-and-no-effect", func(t *testing.T) {
		multi := hintCases["gql-hint/subquery/exists-hash-multi-pass-accepted-no-effect"]
		one := hintCases["gql-hint/subquery/exists-hash-one-pass-accepted-no-effect"]
		unhinted := gqlCases["gql-surface/subquery/correlated-exists"]
		for version := firstOptimizerVersion; version <= latestOptimizerVersion; version++ {
			unhintedPlan := mustAnalyze(t, unhinted, version)
			multiPlan := mustAnalyze(t, multi, version)
			if !proto.Equal(multiPlan, unhintedPlan) {
				t.Errorf("MULTI_PASS v%d plan differs from unhinted correlated EXISTS", version)
			}
			onePlan, err := analyze(t, one, version)
			if version <= 3 {
				if err == nil || !strings.Contains(err.Error(), "Invalid value for hash_join_execution hint: ONE_PASS") {
					t.Errorf("ONE_PASS v%d error = %v, want stable invalid-value error", version, err)
				}
				continue
			}
			if err != nil {
				t.Errorf("ONE_PASS v%d error = %v", version, err)
				continue
			}
			if !proto.Equal(onePlan, unhintedPlan) {
				t.Errorf("ONE_PASS v%d plan differs from unhinted correlated EXISTS", version)
			}
		}
	})

	t.Run("push-v2-v3", func(t *testing.T) {
		query := hintCases["gql-hint/version/traversal-push"]
		if _, err := analyze(t, query, 2); err == nil || !strings.Contains(err.Error(), "Invalid value for join_type/join_method hint: push_broadcast_hash_join") {
			t.Fatalf("PUSH v2 error = %v, want stable invalid-value error", err)
		}
		plan := mustAnalyze(t, query, 3)
		if got := countPlanNodes(plan, "Push Broadcast Hash Join", ""); got != 1 {
			t.Errorf("PUSH v3 operator count = %d, want 1", got)
		}
	})

	t.Run("extended-hint-value-and-placement-matrix-v1-v8", func(t *testing.T) {
		directSetHint := hintCases["gql-hint/set-operation/direct-hint-unsupported"]
		valueApplyBatch := hintCases["gql-hint/subquery/value-apply-batch-false-unsupported"]
		valueApply := hintCases["gql-hint/subquery/value-apply-unsupported"]
		traversalMerge := hintCases["gql-hint/element/traversal-merge"]
		statementMerge := hintCases["gql-hint/set-operation/statement-merge"]
		statementApply := hintCases["gql-hint/set-operation/statement-apply"]
		statementPush := hintCases["gql-hint/set-operation/statement-push"]
		composedRow := hintCases["gql-hint/element/node-force-secondary-index-scan-row"]
		composedBatch := hintCases["gql-hint/element/node-force-secondary-index-scan-batch"]
		composedColumnar := hintCases["gql-hint/element/node-force-secondary-index-scan-columnar"]
		composedNoColumnar := hintCases["gql-hint/element/node-force-secondary-index-scan-no-columnar"]
		composedControl := hintCases["gql-hint/element/node-force-secondary-index"]
		seekZero := hintCases["gql-hint/element/node-force-secondary-index-seekable-key-size-0"]
		seekOne := hintCases["gql-hint/element/node-force-secondary-index-seekable-key-size-1"]
		hashLeft := hintCases["gql-hint/element/traversal-hash-build-left"]
		hashRight := hintCases["gql-hint/element/traversal-hash-build-right"]
		applyTrue := hintCases["gql-hint/element/traversal-apply-batch-true"]
		applyFalse := hintCases["gql-hint/element/traversal-apply-batch-false"]

		for version := firstOptimizerVersion; version <= latestOptimizerVersion; version++ {
			if _, err := analyze(t, directSetHint, version); err == nil || !strings.Contains(err.Error(), "Expected keyword ALL or keyword DISTINCT but got") {
				t.Errorf("direct GQL set hint v%d error = %v, want stable syntax error", version, err)
			}
			if _, err := analyze(t, valueApplyBatch, version); err == nil || !strings.Contains(err.Error(), "Unsupported hint: batch_mode.") {
				t.Errorf("VALUE APPLY BATCH v%d error = %v, want stable unsupported-hint error", version, err)
			}
			if _, err := analyze(t, valueApply, version); err == nil || !strings.Contains(err.Error(), "Unsupported hint: join_method.") {
				t.Errorf("VALUE APPLY v%d error = %v, want stable unsupported-hint error", version, err)
			}

			traversalMergePlan := mustAnalyze(t, traversalMerge, version)
			if got := countPlanNodes(traversalMergePlan, "Merge Join", "INNER"); got != 1 {
				t.Errorf("traversal MERGE v%d INNER Merge Join count = %d, want 1", version, got)
			}

			statementMergePlan := mustAnalyze(t, statementMerge, version)
			if got := countPlanNodes(statementMergePlan, "Merge Join", "INNER"); got != 2 {
				t.Errorf("statement MERGE v%d INNER Merge Join count = %d, want 2", version, got)
			}
			if got := countPlanNodes(statementMergePlan, "Distributed Cross Apply", ""); got != 0 {
				t.Errorf("statement MERGE v%d Distributed Cross Apply count = %d, want 0", version, got)
			}

			statementApplyPlan := mustAnalyze(t, statementApply, version)
			if got := countPlanNodes(statementApplyPlan, "Distributed Cross Apply", ""); got != 2 {
				t.Errorf("statement APPLY v%d Distributed Cross Apply count = %d, want 2", version, got)
			}
			if got := countPlanNodes(statementApplyPlan, "Cross Apply", ""); got != 2 {
				t.Errorf("statement APPLY v%d Cross Apply count = %d, want 2", version, got)
			}

			statementPushPlan, err := analyze(t, statementPush, version)
			if version <= 2 {
				if err == nil || !strings.Contains(err.Error(), "Invalid value for join_type/join_method hint: push_broadcast_hash_join") {
					t.Errorf("statement PUSH v%d error = %v, want stable invalid-value error", version, err)
				}
			} else if err != nil {
				t.Errorf("statement PUSH v%d error = %v", version, err)
			} else if got := countPlanNodes(statementPushPlan, "Push Broadcast Hash Join", ""); got != 2 {
				t.Errorf("statement PUSH v%d operator count = %d, want 2", version, got)
			}

			rowPlan := mustAnalyze(t, composedRow, version)
			batchPlan := mustAnalyze(t, composedBatch, version)
			if proto.Equal(rowPlan, batchPlan) {
				t.Errorf("composed FORCE_INDEX ROW/BATCH v%d plans unexpectedly equal", version)
			}
			if got := countPlanNodesWithMetadata(rowPlan, "Scan", map[string]string{"scan_target": "SingersByFirstLastName", "scan_type": "IndexScan", "scan_method": "Row"}); got != 1 {
				t.Errorf("composed ROW v%d matching scan count = %d, want 1", version, got)
			}
			if got := countPlanNodesWithMetadata(batchPlan, "Scan", map[string]string{"scan_target": "SingersByFirstLastName", "scan_type": "IndexScan", "scan_method": "Batch"}); got != 1 {
				t.Errorf("composed BATCH v%d matching scan count = %d, want 1", version, got)
			}

			columnarPlan := mustAnalyze(t, composedColumnar, version)
			noColumnarPlan := mustAnalyze(t, composedNoColumnar, version)
			controlPlan := mustAnalyze(t, composedControl, version)
			if proto.Equal(columnarPlan, noColumnarPlan) || proto.Equal(columnarPlan, controlPlan) || proto.Equal(noColumnarPlan, controlPlan) {
				t.Errorf("COLUMNAR/NO_COLUMNAR/control v%d plans are not pairwise distinct", version)
			}
			if got := countPlanNodesWithMetadata(columnarPlan, "Scan", map[string]string{"scan_target": "SingersByFirstLastName", "scan_type": "IndexScan", "scan_method": "Batch", "scan_format": "Columnar"}); got != 1 {
				t.Errorf("COLUMNAR v%d matching scan count = %d, want 1", version, got)
			}
			if got := countPlanNodesWithMetadata(noColumnarPlan, "Scan", map[string]string{"scan_target": "SingersByFirstLastName", "scan_type": "IndexScan", "scan_method": "Automatic", "scan_format": "Row"}); got != 1 {
				t.Errorf("NO_COLUMNAR v%d matching scan count = %d, want 1", version, got)
			}
			if got := countPlanNodesWithMetadata(controlPlan, "Scan", map[string]string{"scan_target": "SingersByFirstLastName", "scan_type": "IndexScan", "scan_method": "Automatic"}); got != 1 {
				t.Errorf("COLUMNAR control v%d matching scan count = %d, want 1", version, got)
			}
			if got := countPlanNodesWithMetadataKey(controlPlan, "Scan", "scan_format"); got != 0 {
				t.Errorf("COLUMNAR control v%d scan_format-bearing Scan count = %d, want 0", version, got)
			}

			zeroPlan := mustAnalyze(t, seekZero, version)
			onePlan := mustAnalyze(t, seekOne, version)
			if proto.Equal(zeroPlan, onePlan) {
				t.Errorf("seek-size 0/1 v%d plans unexpectedly equal", version)
			}
			if got := countPlanNodesWithMetadata(zeroPlan, "Filter Scan", map[string]string{"seekable_key_size": "0"}); got != 1 {
				t.Errorf("seek-size 0 v%d Filter Scan count = %d, want 1", version, got)
			}
			if got := countPlanNodesWithMetadata(onePlan, "Filter Scan", map[string]string{"seekable_key_size": "1"}); got != 1 {
				t.Errorf("seek-size 1 v%d Filter Scan count = %d, want 1", version, got)
			}

			leftPlan := mustAnalyze(t, hashLeft, version)
			rightPlan := mustAnalyze(t, hashRight, version)
			if proto.Equal(leftPlan, rightPlan) {
				t.Errorf("HASH build LEFT/RIGHT v%d plans unexpectedly equal", version)
			}
			for label, plan := range map[string]*spannerpb.QueryPlan{"left": leftPlan, "right": rightPlan} {
				if got := countPlanNodes(plan, "Hash Join", "INNER"); got != 1 {
					t.Errorf("HASH build %s v%d INNER Hash Join count = %d, want 1", label, version, got)
				}
			}

			truePlan := mustAnalyze(t, applyTrue, version)
			falsePlan := mustAnalyze(t, applyFalse, version)
			if proto.Equal(truePlan, falsePlan) {
				t.Errorf("APPLY BATCH TRUE/FALSE v%d plans unexpectedly equal", version)
			}
			if got := countPlanNodes(truePlan, "Distributed Cross Apply", ""); got != 2 {
				t.Errorf("APPLY BATCH TRUE v%d Distributed Cross Apply count = %d, want 2", version, got)
			}
			if got := countPlanNodes(falsePlan, "Distributed Cross Apply", ""); got != 1 {
				t.Errorf("APPLY BATCH FALSE v%d Distributed Cross Apply count = %d, want 1", version, got)
			}
		}
	})

	t.Run("stable-plan-effects-v1-v8", func(t *testing.T) {
		statementHash := hintCases["gql-hint/set-operation/statement-hash"]
		unhintedSet := gqlCases["gql-surface/set-operation/intersect-distinct-nontrivial"]
		armMixed := hintCases["gql-hint/set-operation/arm-mixed-hash-apply"]
		base := hintCases["gql-hint/element/node-force-base-table"]
		secondary := hintCases["gql-hint/element/node-force-secondary-index"]
		row := hintCases["gql-hint/graph-table/node-scan-row"]
		batch := hintCases["gql-hint/graph-table/node-scan-batch"]
		for version := firstOptimizerVersion; version <= latestOptimizerVersion; version++ {
			statementPlan := mustAnalyze(t, statementHash, version)
			unhintedPlan := mustAnalyze(t, unhintedSet, version)
			if proto.Equal(statementPlan, unhintedPlan) {
				t.Errorf("statement HASH v%d plan unexpectedly equals unhinted GQL INTERSECT", version)
			}
			if got := countPlanNodes(statementPlan, "Hash Join", "INNER"); got != 2 {
				t.Errorf("statement HASH v%d INNER Hash Join count = %d, want 2", version, got)
			}
			if got := countPlanNodes(statementPlan, "Distributed Cross Apply", ""); got != 0 {
				t.Errorf("statement HASH v%d Distributed Cross Apply count = %d, want 0", version, got)
			}

			mixedPlan := mustAnalyze(t, armMixed, version)
			if got := countPlanNodes(mixedPlan, "Hash Join", "INNER"); got != 1 {
				t.Errorf("arm-mixed v%d INNER Hash Join count = %d, want 1", version, got)
			}
			if got := countPlanNodes(mixedPlan, "Distributed Cross Apply", ""); got != 2 {
				t.Errorf("arm-mixed v%d Distributed Cross Apply count = %d, want 2", version, got)
			}

			basePlan := mustAnalyze(t, base, version)
			secondaryPlan := mustAnalyze(t, secondary, version)
			if proto.Equal(basePlan, secondaryPlan) {
				t.Errorf("forced base/index v%d plans unexpectedly equal", version)
			}
			rowPlan := mustAnalyze(t, row, version)
			batchPlan := mustAnalyze(t, batch, version)
			if proto.Equal(rowPlan, batchPlan) {
				t.Errorf("ROW/BATCH graph scan v%d plans unexpectedly equal", version)
			}
		}
	})

	t.Run("placement-index-and-scan-matrix-v1-v8", func(t *testing.T) {
		indexUnion := hintCases["gql-hint/element/node-index-strategy-force-index-union"]
		indexControl := hintCases["gql-hint/element/node-index-strategy-control"]
		groupTrue := hintCases["gql-hint/element/node-force-secondary-index-groupby-scan-true"]
		groupFalse := hintCases["gql-hint/element/node-force-secondary-index-groupby-scan-false"]
		groupControl := hintCases["gql-hint/element/node-force-secondary-index-groupby-scan-control"]
		edgeRow := hintCases["gql-hint/element/edge-scan-row"]
		edgeBatch := hintCases["gql-hint/element/edge-scan-batch"]
		nodeEdgeHash := hintCases["gql-hint/element/node-to-edge-hash-build-right"]
		nodeEdgeControl := hintCases["gql-hint/element/node-to-edge-control"]
		subpathEdgeApply := hintCases["gql-hint/element/subpath-to-edge-apply-batch-false"]
		subpathEdgeControl := hintCases["gql-hint/element/subpath-to-edge-control"]
		betweenPathsHash := hintCases["gql-hint/element/between-path-patterns-hash"]
		betweenPathsControl := hintCases["gql-hint/element/between-path-patterns-control"]
		subpathNodeHash := hintCases["gql-hint/runtime-extension/subpath-to-node-hash"]
		subpathNodeControl := hintCases["gql-hint/runtime-extension/subpath-to-node-control"]
		subpathSubpathHash := hintCases["gql-hint/runtime-extension/subpath-to-subpath-hash"]
		subpathSubpathControl := hintCases["gql-hint/runtime-extension/subpath-to-subpath-control"]
		matchMerge := hintCases["gql-hint/match/second-merge"]
		matchPush := hintCases["gql-hint/match/second-push"]
		matchControl := hintCases["gql-hint/match/second-control"]

		checkCounts := func(t *testing.T, label string, version int, plan *spannerpb.QueryPlan, want map[string]int) {
			t.Helper()
			for displayName, count := range want {
				if got := countPlanNodes(plan, displayName, ""); got != count {
					t.Errorf("%s v%d %s count = %d, want %d", label, version, displayName, got, count)
				}
			}
		}

		for version := firstOptimizerVersion; version <= latestOptimizerVersion; version++ {
			indexUnionPlan := mustAnalyze(t, indexUnion, version)
			indexControlPlan := mustAnalyze(t, indexControl, version)
			if proto.Equal(indexUnionPlan, indexControlPlan) {
				t.Errorf("FORCE_INDEX_UNION/control v%d plans unexpectedly equal", version)
			}
			checkCounts(t, "FORCE_INDEX_UNION", version, indexUnionPlan, map[string]int{"Union All": 1, "Aggregate": 1})
			if got := countPlanNodesWithMetadata(indexUnionPlan, "Scan", map[string]string{"scan_target": "SingersByFirstLastName", "scan_type": "IndexScan"}); got != 1 {
				t.Errorf("FORCE_INDEX_UNION v%d FirstName index count = %d, want 1", version, got)
			}
			if got := countPlanNodesWithMetadata(indexUnionPlan, "Scan", map[string]string{"scan_target": "SingersByLastName", "scan_type": "IndexScan"}); got != 1 {
				t.Errorf("FORCE_INDEX_UNION v%d LastName index count = %d, want 1", version, got)
			}
			controlUnion := 0
			if version >= 7 {
				controlUnion = 1
			}
			checkCounts(t, "index-union control", version, indexControlPlan, map[string]int{"Union All": controlUnion, "Aggregate": controlUnion})

			groupTruePlan := mustAnalyze(t, groupTrue, version)
			groupFalsePlan := mustAnalyze(t, groupFalse, version)
			groupControlPlan := mustAnalyze(t, groupControl, version)
			if proto.Equal(groupTruePlan, groupFalsePlan) {
				t.Errorf("GROUPBY_SCAN TRUE/FALSE v%d plans unexpectedly equal", version)
			}
			if !proto.Equal(groupFalsePlan, groupControlPlan) {
				t.Errorf("GROUPBY_SCAN FALSE/control v%d plans differ", version)
			}
			if got := countPlanNodesWithMetadata(groupTruePlan, "Scan", map[string]string{"scan_target": "SingersByFirstLastName", "scan_method": "Row"}); got != 1 {
				t.Errorf("GROUPBY_SCAN TRUE v%d matching scan count = %d, want 1", version, got)
			}
			for label, plan := range map[string]*spannerpb.QueryPlan{"FALSE": groupFalsePlan, "control": groupControlPlan} {
				if got := countPlanNodesWithMetadata(plan, "Scan", map[string]string{"scan_target": "SingersByFirstLastName", "scan_method": "Automatic"}); got != 1 {
					t.Errorf("GROUPBY_SCAN %s v%d matching scan count = %d, want 1", label, version, got)
				}
			}

			edgeRowPlan := mustAnalyze(t, edgeRow, version)
			if got := countPlanNodesWithMetadata(edgeRowPlan, "Scan", map[string]string{"scan_target": "Collaborations", "scan_method": "Row"}); got != 1 {
				t.Errorf("edge ROW v%d matching scan count = %d, want 1", version, got)
			}
			edgeBatchPlan, err := analyze(t, edgeBatch, version)
			if version <= 5 {
				if err == nil || !strings.Contains(err.Error(), "scan_method=batch} is not supported on the right side of apply join") {
					t.Errorf("edge BATCH v%d error = %v, want stable right-side error", version, err)
				}
			} else if err != nil {
				t.Errorf("edge BATCH v%d error = %v", version, err)
			} else if got := countPlanNodesWithMetadata(edgeBatchPlan, "Scan", map[string]string{"scan_target": "Collaborations", "scan_method": "Batch"}); got != 1 {
				t.Errorf("edge BATCH v%d matching scan count = %d, want 1", version, got)
			}

			nodeEdgeHashPlan := mustAnalyze(t, nodeEdgeHash, version)
			nodeEdgeControlPlan := mustAnalyze(t, nodeEdgeControl, version)
			if proto.Equal(nodeEdgeHashPlan, nodeEdgeControlPlan) {
				t.Errorf("node-to-edge HASH/control v%d plans unexpectedly equal", version)
			}
			checkCounts(t, "node-to-edge HASH", version, nodeEdgeHashPlan, map[string]int{"Hash Join": 1, "Distributed Cross Apply": 1, "Cross Apply": 1, "Scan": 4})
			checkCounts(t, "node-to-edge control", version, nodeEdgeControlPlan, map[string]int{"Hash Join": 0, "Distributed Cross Apply": 2, "Cross Apply": 2, "Scan": 5})

			subpathEdgeApplyPlan := mustAnalyze(t, subpathEdgeApply, version)
			subpathEdgeControlPlan := mustAnalyze(t, subpathEdgeControl, version)
			if proto.Equal(subpathEdgeApplyPlan, subpathEdgeControlPlan) {
				t.Errorf("subpath-to-edge APPLY/control v%d plans unexpectedly equal", version)
			}
			checkCounts(t, "subpath-to-edge APPLY", version, subpathEdgeApplyPlan, map[string]int{"Distributed Cross Apply": 3, "Cross Apply": 4, "Scan": 8})
			checkCounts(t, "subpath-to-edge control", version, subpathEdgeControlPlan, map[string]int{"Distributed Cross Apply": 4, "Cross Apply": 4, "Scan": 9})

			betweenHashPlan := mustAnalyze(t, betweenPathsHash, version)
			betweenControlPlan := mustAnalyze(t, betweenPathsControl, version)
			if version <= 2 {
				if !proto.Equal(betweenHashPlan, betweenControlPlan) {
					t.Errorf("between-path HASH/control v%d plans differ before effect boundary", version)
				}
				if got := countPlanNodes(betweenHashPlan, "Hash Join", ""); got != 0 {
					t.Errorf("between-path HASH v%d Hash Join count = %d, want 0", version, got)
				}
			} else {
				if proto.Equal(betweenHashPlan, betweenControlPlan) {
					t.Errorf("between-path HASH/control v%d plans unexpectedly equal", version)
				}
				if got := countPlanNodes(betweenHashPlan, "Hash Join", "INNER"); got != 1 {
					t.Errorf("between-path HASH v%d INNER Hash Join count = %d, want 1", version, got)
				}
			}

			subpathNodeHashPlan := mustAnalyze(t, subpathNodeHash, version)
			subpathNodeControlPlan := mustAnalyze(t, subpathNodeControl, version)
			if proto.Equal(subpathNodeHashPlan, subpathNodeControlPlan) {
				t.Errorf("runtime-extension subpath-to-node v%d plans unexpectedly equal", version)
			}
			checkCounts(t, "subpath-to-node HASH", version, subpathNodeHashPlan, map[string]int{"Hash Join": 1, "Distributed Cross Apply": 1, "Cross Apply": 1, "Scan": 4})
			if got := countPlanNodes(subpathNodeControlPlan, "Hash Join", ""); got != 0 {
				t.Errorf("subpath-to-node control v%d Hash Join count = %d, want 0", version, got)
			}

			subpathSubpathHashPlan := mustAnalyze(t, subpathSubpathHash, version)
			subpathSubpathControlPlan := mustAnalyze(t, subpathSubpathControl, version)
			if proto.Equal(subpathSubpathHashPlan, subpathSubpathControlPlan) {
				t.Errorf("runtime-extension subpath-to-subpath v%d plans unexpectedly equal", version)
			}
			if got := countPlanNodes(subpathSubpathHashPlan, "Hash Join", ""); got < 1 {
				t.Errorf("subpath-to-subpath HASH v%d Hash Join count = %d, want positive", version, got)
			}
			if got := countPlanNodes(subpathSubpathControlPlan, "Hash Join", ""); got != 0 {
				t.Errorf("subpath-to-subpath control v%d Hash Join count = %d, want 0", version, got)
			}

			matchMergePlan := mustAnalyze(t, matchMerge, version)
			matchControlPlan := mustAnalyze(t, matchControl, version)
			if proto.Equal(matchMergePlan, matchControlPlan) {
				t.Errorf("second MATCH MERGE/control v%d plans unexpectedly equal", version)
			}
			checkCounts(t, "second MATCH MERGE", version, matchMergePlan, map[string]int{"Merge Join": 1, "Sort": 1, "Distributed Cross Apply": 3, "Cross Apply": 3, "Scan": 8})
			matchPushPlan, err := analyze(t, matchPush, version)
			if version <= 2 {
				if err == nil || !strings.Contains(err.Error(), "Invalid value for join_type/join_method hint: push_broadcast_hash_join") {
					t.Errorf("second MATCH PUSH v%d error = %v, want stable invalid-value error", version, err)
				}
			} else if err != nil {
				t.Errorf("second MATCH PUSH v%d error = %v", version, err)
			} else {
				if proto.Equal(matchPushPlan, matchControlPlan) {
					t.Errorf("second MATCH PUSH/control v%d plans unexpectedly equal", version)
				}
				checkCounts(t, "second MATCH PUSH", version, matchPushPlan, map[string]int{"Push Broadcast Hash Join": 1, "Hash Join": 1, "Distributed Cross Apply": 3, "Cross Apply": 3, "Scan": 9})
			}
		}
	})
}

func queryCasesByLabel(t *testing.T, queries []queryCase) map[string]queryCase {
	t.Helper()
	out := make(map[string]queryCase, len(queries))
	for _, query := range queries {
		if _, duplicate := out[query.Label]; duplicate {
			t.Fatalf("duplicate query label %q", query.Label)
		}
		out[query.Label] = query
	}
	return out
}

func countPlanNodes(plan *spannerpb.QueryPlan, displayName, joinType string) int {
	count := 0
	for _, node := range plan.GetPlanNodes() {
		if node.GetKind() != spannerpb.PlanNode_RELATIONAL || node.GetDisplayName() != displayName {
			continue
		}
		if joinType != "" {
			value := node.GetMetadata().GetFields()["join_type"]
			if value == nil || value.GetStringValue() != joinType {
				continue
			}
		}
		count++
	}
	return count
}

func countPlanNodesWithMetadata(plan *spannerpb.QueryPlan, displayName string, metadata map[string]string) int {
	count := 0
	for _, node := range plan.GetPlanNodes() {
		if node.GetKind() != spannerpb.PlanNode_RELATIONAL || node.GetDisplayName() != displayName {
			continue
		}
		matches := true
		for key, want := range metadata {
			value := node.GetMetadata().GetFields()[key]
			if value == nil || value.GetStringValue() != want {
				matches = false
				break
			}
		}
		if matches {
			count++
		}
	}
	return count
}

func countPlanNodesWithMetadataKey(plan *spannerpb.QueryPlan, displayName, key string) int {
	count := 0
	for _, node := range plan.GetPlanNodes() {
		if node.GetKind() != spannerpb.PlanNode_RELATIONAL || node.GetDisplayName() != displayName {
			continue
		}
		if _, ok := node.GetMetadata().GetFields()[key]; ok {
			count++
		}
	}
	return count
}
