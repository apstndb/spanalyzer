//go:build integration && omni

package main

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestIntegrationConditionBoundariesVersionMatrixOnOmni(t *testing.T) {
	ddls, err := parseBuiltInDDLs("condition-boundary-schema.sql", conditionBoundaryDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	clients := openOmniClients(t, ddls)

	cases := queryCasesByLabel(t, conditionBoundaryQueries)
	analyze := func(t *testing.T, label string, version int) *spannerpb.QueryPlan {
		t.Helper()
		query, ok := cases[label]
		if !ok {
			t.Fatalf("condition boundary query %q is missing", label)
		}
		query.SQL = withOptimizerVersionStatementHint(query.SQL, version)
		plan, err := analyzePlan(t.Context(), clients.Client, query)
		if err != nil {
			t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", label, version, err)
		}
		return plan
	}

	for version := firstOptimizerVersion; version <= latestOptimizerVersion; version++ {
		t.Run("v"+strconv.Itoa(version), func(t *testing.T) {
			plans := make(map[string]*spannerpb.QueryPlan, len(cases))
			for label := range cases {
				plans[label] = analyze(t, label, version)
			}

			assertCondition(t, plans["condition-boundary/scan/equality-literal"], "Distributed Union", "Split Range", "($SongName = 'Alpha')")
			assertCondition(t, plans["condition-boundary/scan/equality-literal"], "Scan", "Seek Condition", "IS_NOT_DISTINCT_FROM($SongName, 'Alpha')")
			assertCondition(t, plans["condition-boundary/scan/range-literal"], "Scan", "Seek Condition", "(($SongName >= 'A') AND ($SongName < 'B'))")
			assertCondition(t, plans["condition-boundary/scan/starts-with-literal"], "Scan", "Seek Condition", "STARTS_WITH($SongName, 'A')")
			assertCondition(t, plans["condition-boundary/scan/starts-with-parameter"], "Scan", "Seek Condition", "STARTS_WITH($SongName, @prefix)")
			assertCondition(t, plans["condition-boundary/scan/like-prefix-literal"], "Scan", "Seek Condition", "STARTS_WITH($SongName, 'A')")
			assertCondition(t, plans["condition-boundary/scan/like-prefix-parameter"], "Filter Scan", "Residual Condition", "($SongName LIKE @pattern)")
			assertCondition(t, plans["condition-boundary/scan/like-contains-literal"], "Filter Scan", "Residual Condition", "($SongName LIKE '%A%')")
			assertCondition(t, plans["condition-boundary/scan/regexp-contains"], "Filter Scan", "Residual Condition", "REGEXP_CONTAINS($SongName, 'A')")
			assertCondition(t, plans["condition-boundary/scan/strpos-contains"], "Filter Scan", "Residual Condition", "(STRPOS($SongName, 'A') > 0)")
			assertCondition(t, plans["condition-boundary/scan/ends-with"], "Filter Scan", "Residual Condition", "ENDS_WITH($SongName, 'A')")
			assertCondition(t, plans["condition-boundary/scan/substr-prefix"], "Filter Scan", "Residual Condition", "(SUBSTR($SongName, 1, 1) = 'A')")
			assertCondition(t, plans["condition-boundary/scan/lower-key"], "Filter Scan", "Residual Condition", "(LOWER($SongName) = 'a')")
			assertCondition(t, plans["condition-boundary/scan/foldable-constant"], "Distributed Union", "Split Range", "($SongName = 'A')")
			assertCondition(t, plans["condition-boundary/scan/foldable-constant"], "Scan", "Seek Condition", "IS_NOT_DISTINCT_FROM($SongName, 'A')")
			assertCondition(t, plans["condition-boundary/scan/or-seekable-prefixes"], "Scan", "Seek Condition", "(STARTS_WITH($SongName, 'A') OR STARTS_WITH($SongName, 'B'))")
			assertCondition(t, plans["condition-boundary/scan/seek-plus-residual"], "Distributed Union", "Split Range", "(STARTS_WITH($SongName, 'A') AND (STRPOS($SongName, 'x') > 0))")
			assertCondition(t, plans["condition-boundary/scan/seek-plus-residual"], "Scan", "Seek Condition", "STARTS_WITH($SongName, 'A')")
			assertCondition(t, plans["condition-boundary/scan/seek-plus-residual"], "Filter Scan", "Residual Condition", "(STRPOS($SongName, 'x') > 0)")

			regexpPrefix := plans["condition-boundary/scan/regexp-prefix"]
			regexpResidual := plans["condition-boundary/scan/regexp-prefix-with-residual"]
			likeResidual := plans["condition-boundary/scan/like-prefix-with-residual"]
			if version == 1 {
				assertCondition(t, regexpPrefix, "Filter Scan", "Residual Condition", "REGEXP_CONTAINS($SongName, '^A.*')")
				assertNoCondition(t, regexpPrefix, "Scan", "Seek Condition")
				assertCondition(t, regexpResidual, "Filter Scan", "Residual Condition", "REGEXP_CONTAINS($SongName, '^A.B')")
				assertNoCondition(t, regexpResidual, "Scan", "Seek Condition")
				assertCondition(t, likeResidual, "Filter Scan", "Residual Condition", "($SongName LIKE 'A_B%')")
				assertNoCondition(t, likeResidual, "Scan", "Seek Condition")
			} else {
				assertCondition(t, regexpPrefix, "Scan", "Seek Condition", "STARTS_WITH($SongName, 'A')")
				assertNoCondition(t, regexpPrefix, "Filter Scan", "Residual Condition")
				assertCondition(t, regexpResidual, "Scan", "Seek Condition", "STARTS_WITH($SongName, 'A')")
				assertCondition(t, regexpResidual, "Filter Scan", "Residual Condition", "REGEXP_CONTAINS($SongName, '^A.B')")
				assertCondition(t, likeResidual, "Scan", "Seek Condition", "STARTS_WITH($SongName, 'A')")
				assertCondition(t, likeResidual, "Filter Scan", "Residual Condition", "($SongName LIKE 'A_B%')")
			}

			mixedOR := plans["condition-boundary/scan/or-seekable-and-residual"]
			mixedORExpression := "(STARTS_WITH($SongName, 'A') OR ENDS_WITH($SongName, 'B'))"
			if version <= 6 {
				assertCondition(t, mixedOR, "Scan", "Seek Condition", mixedORExpression)
				assertNoCondition(t, mixedOR, "Filter Scan", "Residual Condition")
			} else {
				assertCondition(t, mixedOR, "Filter Scan", "Residual Condition", mixedORExpression)
				assertNoCondition(t, mixedOR, "Scan", "Seek Condition")
			}

			assertScanHasConditions(t, plans["condition-boundary/timestamp-key/pushdown-true"], "CommitTimestampKeys", "Seek Condition", "Timestamp Condition")
			assertNoCondition(t, plans["condition-boundary/timestamp-key/pushdown-true"], "Filter Scan", "Timestamp Condition")
			assertScanHasConditions(t, plans["condition-boundary/timestamp-key/pushdown-false"], "CommitTimestampKeys", "Seek Condition")
			assertNoCondition(t, plans["condition-boundary/timestamp-key/pushdown-false"], "Scan", "Timestamp Condition")

			assertCondition(t, plans["condition-boundary/join/hash-equality"], "Hash Join", "Condition", "($SingerId = $SingerId_1)")
			assertCondition(t, plans["condition-boundary/join/hash-two-equalities"], "Hash Join", "Condition", "(($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))")
			assertConditionContains(t, plans["condition-boundary/join/hash-cast-equality"], "Hash Join", "Condition", "CAST<STRING>")
			assertCondition(t, plans["condition-boundary/join/hash-arithmetic-equality"], "Hash Join", "Condition", "(($SingerId + 1) = $SingerId_1)")
			assertCondition(t, plans["condition-boundary/join/hash-lower-equality"], "Hash Join", "Condition", "(LOWER($AlbumTitle) = LOWER($SongName))")
			assertCondition(t, plans["condition-boundary/join/hash-null-safe-equality"], "Hash Join", "Condition", "IS_NOT_DISTINCT_FROM($AlbumTitle, $SongName)")
			for _, label := range []string{
				"condition-boundary/join/hash-equality",
				"condition-boundary/join/hash-two-equalities",
				"condition-boundary/join/hash-cast-equality",
				"condition-boundary/join/hash-arithmetic-equality",
				"condition-boundary/join/hash-lower-equality",
				"condition-boundary/join/hash-null-safe-equality",
			} {
				assertNoCondition(t, plans[label], "Hash Join", "Residual Condition")
			}
			assertCondition(t, plans["condition-boundary/join/hash-key-plus-inequality"], "Hash Join", "Condition", "($SingerId = $SingerId_1)")
			assertCondition(t, plans["condition-boundary/join/hash-key-plus-inequality"], "Hash Join", "Residual Condition", "($AlbumId < $AlbumId_1)")
			assertCondition(t, plans["condition-boundary/join/hash-key-plus-starts-with"], "Hash Join", "Residual Condition", "STARTS_WITH($SongName, $AlbumTitle)")
			assertCondition(t, plans["condition-boundary/join/hash-key-plus-strpos"], "Hash Join", "Residual Condition", "(STRPOS($SongName, $AlbumTitle) > 0)")
			assertCondition(t, plans["condition-boundary/join/hash-key-plus-like-contains"], "Hash Join", "Residual Condition", "($SongName LIKE CONCAT('%', $AlbumTitle, '%'))")
			assertCondition(t, plans["condition-boundary/join/hash-inequality-only"], "Hash Join", "Condition", "(true = true)")
			assertCondition(t, plans["condition-boundary/join/hash-inequality-only"], "Hash Join", "Residual Condition", "($AlbumId < $AlbumId_1)")
			assertCondition(t, plans["condition-boundary/join/hash-or-equalities"], "Hash Join", "Condition", "(true = true)")
			assertCondition(t, plans["condition-boundary/join/hash-or-equalities"], "Hash Join", "Residual Condition", "(($SingerId = $SingerId_1) OR ($AlbumId = $AlbumId_1))")
			assertCondition(t, plans["condition-boundary/join/hash-key-plus-one-side-seek"], "Hash Join", "Condition", "($SingerId = $SingerId_1)")
			assertCondition(t, plans["condition-boundary/join/hash-key-plus-one-side-seek"], "Distributed Union", "Split Range", "STARTS_WITH($SongName, 'A')")
			assertCondition(t, plans["condition-boundary/join/hash-key-plus-one-side-seek"], "Scan", "Seek Condition", "STARTS_WITH($SongName, 'A')")
			assertConditionContains(t, plans["condition-boundary/join/merge-lower-equality"], "Merge Join", "Condition", "$sort_AlbumTitle")
			assertCondition(t, plans["condition-boundary/join/merge-key-plus-inequality"], "Merge Join", "Condition", "($SingerId = $SingerId_1)")
			assertCondition(t, plans["condition-boundary/join/merge-key-plus-inequality"], "Merge Join", "Residual Condition", "($AlbumId < $AlbumId_1)")
			assertCondition(t, plans["condition-boundary/join/apply-equality"], "Scan", "Seek Condition", "($SingerId_1 = $SingerId)")
			assertCondition(t, plans["condition-boundary/join/apply-key-plus-range"], "Scan", "Seek Condition", "($SingerId_1 = $SingerId) AND ($AlbumId_1 > $AlbumId)")
		})
	}
}

func conditionExpressions(plan *spannerpb.QueryPlan, parentName, linkType string) []string {
	nodes := make(map[int32]*spannerpb.PlanNode, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		nodes[node.GetIndex()] = node
	}
	var expressions []string
	for _, parent := range plan.GetPlanNodes() {
		if parent.GetDisplayName() != parentName {
			continue
		}
		for _, link := range parent.GetChildLinks() {
			if link.GetType() != linkType {
				continue
			}
			if child := nodes[link.GetChildIndex()]; child != nil {
				expressions = append(expressions, child.GetShortRepresentation().GetDescription())
			}
		}
	}
	slices.Sort(expressions)
	return expressions
}

func assertScanHasConditions(t *testing.T, plan *spannerpb.QueryPlan, target string, linkTypes ...string) {
	t.Helper()
	nodes := make(map[int32]*spannerpb.PlanNode, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		nodes[node.GetIndex()] = node
	}
	for _, node := range plan.GetPlanNodes() {
		if node.GetDisplayName() != "Scan" || node.GetMetadata().GetFields()["scan_target"].GetStringValue() != target {
			continue
		}
		found := make(map[string]bool, len(node.GetChildLinks()))
		for _, link := range node.GetChildLinks() {
			if nodes[link.GetChildIndex()] != nil {
				found[link.GetType()] = true
			}
		}
		if slices.ContainsFunc(linkTypes, func(linkType string) bool { return !found[linkType] }) {
			continue
		}
		return
	}
	t.Errorf("Scan on %s does not have all condition links %q", target, linkTypes)
}

func assertCondition(t *testing.T, plan *spannerpb.QueryPlan, parentName, linkType, want string) {
	t.Helper()
	got := conditionExpressions(plan, parentName, linkType)
	if !slices.Contains(got, want) {
		t.Errorf("%s %s expressions = %q, want %q", parentName, linkType, got, want)
	}
}

func assertConditionContains(t *testing.T, plan *spannerpb.QueryPlan, parentName, linkType, wantSubstring string) {
	t.Helper()
	got := conditionExpressions(plan, parentName, linkType)
	for _, expression := range got {
		if strings.Contains(expression, wantSubstring) {
			return
		}
	}
	t.Errorf("%s %s expressions = %q, want substring %q", parentName, linkType, got, wantSubstring)
}

func assertNoCondition(t *testing.T, plan *spannerpb.QueryPlan, parentName, linkType string) {
	t.Helper()
	if got := conditionExpressions(plan, parentName, linkType); len(got) != 0 {
		t.Errorf("%s %s expressions = %q, want none", parentName, linkType, got)
	}
}
