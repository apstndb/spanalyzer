package plancontract

import (
	"reflect"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestPlanNormalizationIsIndependentOfPlanNodeOrder(t *testing.T) {
	t.Parallel()

	nodes := []*spannerpb.PlanNode{
		{
			Index:       0,
			DisplayName: "Hash Join",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "Build", ChildIndex: 2},
				{Type: "Probe", ChildIndex: 1},
			},
		},
		{Index: 1, DisplayName: "Scan"},
		{
			Index:       2,
			DisplayName: "Filter",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "Input", ChildIndex: 1},
			},
		},
	}
	ordered := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{nodes[0], nodes[1], nodes[2]}}
	reordered := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{nodes[2], nodes[0], nodes[1]}}
	orderedInput := planNodeIndexes(ordered)
	reorderedInput := planNodeIndexes(reordered)

	orderedOperators := NormalizeOperators(ordered)
	reorderedOperators := NormalizeOperators(reordered)
	if !reflect.DeepEqual(reorderedOperators, orderedOperators) {
		t.Fatalf("NormalizeOperators() differs by input order:\nordered:  %#v\nreordered: %#v", orderedOperators, reorderedOperators)
	}
	if got, want := operatorIndexes(orderedOperators), []int32{0, 1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeOperators() indexes = %v, want %v", got, want)
	}
	if got, want := orderedOperators[0].ChildIndexes, []int32{2, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeOperators() child indexes = %v, want child-link order %v", got, want)
	}

	orderedEdges := NormalizeOperatorEdges(ordered)
	reorderedEdges := NormalizeOperatorEdges(reordered)
	if !reflect.DeepEqual(reorderedEdges, orderedEdges) {
		t.Fatalf("NormalizeOperatorEdges() differs by input order:\nordered:  %#v\nreordered: %#v", orderedEdges, reorderedEdges)
	}
	wantEdges := []OperatorEdge{
		{ParentIndex: 0, ChildIndex: 2, Type: "Build"},
		{ParentIndex: 0, ChildIndex: 1, Type: "Probe"},
		{ParentIndex: 2, ChildIndex: 1, Type: "Input"},
	}
	if !reflect.DeepEqual(orderedEdges, wantEdges) {
		t.Fatalf("NormalizeOperatorEdges() = %#v, want child-link order %#v", orderedEdges, wantEdges)
	}

	if got, want := OperatorTreeDigest(reordered), OperatorTreeDigest(ordered); got != want {
		t.Fatalf("OperatorTreeDigest() differs by input order: got %q, want %q", got, want)
	}
	if got := planNodeIndexes(ordered); !reflect.DeepEqual(got, orderedInput) {
		t.Fatalf("ordered input indexes mutated to %v, want %v", got, orderedInput)
	}
	if got := planNodeIndexes(reordered); !reflect.DeepEqual(got, reorderedInput) {
		t.Fatalf("reordered input indexes mutated to %v, want %v", got, reorderedInput)
	}
}

func planNodeIndexes(plan *spannerpb.QueryPlan) []int32 {
	indexes := make([]int32, 0, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		indexes = append(indexes, node.GetIndex())
	}
	return indexes
}

func operatorIndexes(operators []Operator) []int32 {
	indexes := make([]int32, 0, len(operators))
	for _, operator := range operators {
		indexes = append(indexes, operator.Index)
	}
	return indexes
}
