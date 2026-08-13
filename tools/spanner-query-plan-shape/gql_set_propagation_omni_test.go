//go:build integration && omni

package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/proto"
)

const gqlSetPropagationStrictError = "GQL set operation requires all input queries to have identical column names"

func TestIntegrationGQLSetPropagationOnOmni(t *testing.T) {
	ddls, err := parseBuiltInDDLs("gql-set-propagation-schema.sql", docsDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	clients := openOmniClients(t, ddls)

	cases := gqlSetPropagationCasesByLabel(t)
	tests := []struct {
		name       string
		label      string
		columns    []string
		rows       []string
		slots      int
		typedNulls int
	}{
		{
			name:       "full",
			label:      "gql-set-propagation/full",
			columns:    []string{"left_only", "shared", "right_only"},
			rows:       []string{"1,10,NULL", "NULL,20,2"},
			slots:      3,
			typedNulls: 2,
		},
		{
			name:       "full outer",
			label:      "gql-set-propagation/full-outer",
			columns:    []string{"left_only", "shared", "right_only"},
			rows:       []string{"1,10,NULL", "NULL,20,2"},
			slots:      3,
			typedNulls: 2,
		},
		{
			name:       "outer",
			label:      "gql-set-propagation/outer",
			columns:    []string{"left_only", "shared", "right_only"},
			rows:       []string{"1,10,NULL", "NULL,20,2"},
			slots:      3,
			typedNulls: 2,
		},
		{
			name:       "inner",
			label:      "gql-set-propagation/inner",
			columns:    []string{"shared"},
			rows:       []string{"10", "20"},
			slots:      1,
			typedNulls: 0,
		},
		{
			name:       "left",
			label:      "gql-set-propagation/left",
			columns:    []string{"left_only", "shared"},
			rows:       []string{"1,10", "NULL,20"},
			slots:      2,
			typedNulls: 1,
		},
		{
			name:       "left outer",
			label:      "gql-set-propagation/left-outer",
			columns:    []string{"left_only", "shared"},
			rows:       []string{"1,10", "NULL,20"},
			slots:      2,
			typedNulls: 1,
		},
	}

	for version := 0; version <= 8; version++ {
		versionName := "default"
		if version != 0 {
			versionName = fmt.Sprintf("v%d", version)
		}
		t.Run(versionName, func(t *testing.T) {
			plans := make(map[string]*spannerpb.QueryPlan, len(tests))
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					query := cases[tt.label]
					query.SQL = gqlSetPropagationWithOptimizerVersion(query.SQL, version)
					plan, err := analyzePlan(t.Context(), clients.Client, query)
					if err != nil {
						t.Fatalf("AnalyzeQuery(%s) error = %v", query.Label, err)
					}
					plans[tt.label] = plan
					assertGQLSetPropagationPlan(t, plan, tt.slots, tt.typedNulls)
					assertGQLSetPropagationRows(t, clients.Client, query.SQL, tt.columns, tt.rows)
				})
			}

			for _, aliases := range [][]string{
				{
					"gql-set-propagation/full",
					"gql-set-propagation/full-outer",
					"gql-set-propagation/outer",
				},
				{
					"gql-set-propagation/left",
					"gql-set-propagation/left-outer",
				},
			} {
				baseline := plans[aliases[0]]
				for _, alias := range aliases[1:] {
					if !proto.Equal(baseline, plans[alias]) {
						t.Errorf("%s and %s plans differ", aliases[0], alias)
					}
				}
			}

			strict := cases["gql-set-propagation/strict-control"]
			strict.SQL = gqlSetPropagationWithOptimizerVersion(strict.SQL, version)
			if _, err := analyzePlan(t.Context(), clients.Client, strict); err == nil ||
				!strings.Contains(err.Error(), gqlSetPropagationStrictError) {
				t.Errorf("AnalyzeQuery(%s) error = %v, want containing %q", strict.Label, err, gqlSetPropagationStrictError)
			}
		})
	}
}

func gqlSetPropagationCasesByLabel(t *testing.T) map[string]queryCase {
	t.Helper()
	result := make(map[string]queryCase, len(gqlSetPropagationQueries))
	for _, query := range gqlSetPropagationQueries {
		if _, duplicate := result[query.Label]; duplicate {
			t.Fatalf("duplicate query label %q", query.Label)
		}
		result[query.Label] = query
	}
	return result
}

func gqlSetPropagationWithOptimizerVersion(sql string, version int) string {
	if version == 0 {
		return sql
	}
	return withOptimizerVersionStatementHint(sql, version)
}

func assertGQLSetPropagationPlan(t *testing.T, plan *spannerpb.QueryPlan, slots, typedNulls int) {
	t.Helper()
	for displayName, want := range map[string]int{
		"Serialize Result": 1,
		"Union All":        1,
		"Union Input":      2,
		"Compute":          2,
		"Unit Relation":    2,
	} {
		if got := gqlSetPropagationRelationalCount(plan, displayName); got != want {
			t.Errorf("%s count = %d, want %d", displayName, got, want)
		}
	}

	nodes := make(map[int32]*spannerpb.PlanNode, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		nodes[node.GetIndex()] = node
	}
	inputSlots := make([]int, 0, 2)
	for _, node := range plan.GetPlanNodes() {
		if node.GetKind() != spannerpb.PlanNode_RELATIONAL || node.GetDisplayName() != "Union Input" {
			continue
		}
		count := 0
		for _, link := range node.GetChildLinks() {
			child := nodes[link.GetChildIndex()]
			if child != nil && child.GetKind() == spannerpb.PlanNode_SCALAR && strings.HasPrefix(link.GetType(), "input_") {
				count++
			}
		}
		inputSlots = append(inputSlots, count)
	}
	slices.Sort(inputSlots)
	wantInputSlots := []int{slots, slots}
	if !slices.Equal(inputSlots, wantInputSlots) {
		t.Errorf("Union Input slot counts = %v, want %v", inputSlots, wantInputSlots)
	}

	unionAll := gqlSetPropagationSingleRelationalNode(t, plan, "Union All")
	if got := gqlSetPropagationScalarChildCount(nodes, unionAll); got != slots {
		t.Errorf("Union All output slot count = %d, want %d", got, slots)
	}
	if got := gqlSetPropagationTypedNullCount(plan); got != typedNulls {
		t.Errorf("typed NULL count = %d, want %d", got, typedNulls)
	}
}

func assertGQLSetPropagationRows(
	t *testing.T,
	client *spanner.Client,
	sql string,
	wantColumns []string,
	wantRows []string,
) {
	t.Helper()
	columns := []string{}
	rows := []string{}
	err := client.Single().Query(t.Context(), spanner.Statement{SQL: sql}).Do(func(row *spanner.Row) error {
		if len(columns) == 0 {
			columns = row.ColumnNames()
		}
		values := make([]string, row.Size())
		for index := range row.Size() {
			var value spanner.NullInt64
			if err := row.Column(index, &value); err != nil {
				return err
			}
			if !value.Valid {
				values[index] = "NULL"
				continue
			}
			values[index] = strconv.FormatInt(value.Int64, 10)
		}
		rows = append(rows, strings.Join(values, ","))
		return nil
	})
	if err != nil {
		t.Fatalf("Query() error = %v\nSQL:\n%s", err, sql)
	}
	slices.Sort(rows)
	if !slices.Equal(columns, wantColumns) {
		t.Errorf("result columns = %v, want %v", columns, wantColumns)
	}
	if !slices.Equal(rows, wantRows) {
		t.Errorf("result rows = %v, want %v", rows, wantRows)
	}
}

func gqlSetPropagationRelationalCount(plan *spannerpb.QueryPlan, displayName string) int {
	count := 0
	for _, node := range plan.GetPlanNodes() {
		if node.GetKind() == spannerpb.PlanNode_RELATIONAL && node.GetDisplayName() == displayName {
			count++
		}
	}
	return count
}

func gqlSetPropagationSingleRelationalNode(
	t *testing.T,
	plan *spannerpb.QueryPlan,
	displayName string,
) *spannerpb.PlanNode {
	t.Helper()
	var result *spannerpb.PlanNode
	for _, node := range plan.GetPlanNodes() {
		if node.GetKind() != spannerpb.PlanNode_RELATIONAL || node.GetDisplayName() != displayName {
			continue
		}
		if result != nil {
			t.Fatalf("plan contains more than one %s node", displayName)
		}
		result = node
	}
	if result == nil {
		t.Fatalf("plan contains no %s node", displayName)
	}
	return result
}

func gqlSetPropagationScalarChildCount(nodes map[int32]*spannerpb.PlanNode, parent *spannerpb.PlanNode) int {
	count := 0
	for _, link := range parent.GetChildLinks() {
		if child := nodes[link.GetChildIndex()]; child != nil && child.GetKind() == spannerpb.PlanNode_SCALAR {
			count++
		}
	}
	return count
}

func gqlSetPropagationTypedNullCount(plan *spannerpb.QueryPlan) int {
	count := 0
	for _, node := range plan.GetPlanNodes() {
		if node.GetKind() == spannerpb.PlanNode_SCALAR &&
			node.GetShortRepresentation().GetDescription() == "<typed null>" {
			count++
		}
	}
	return count
}
