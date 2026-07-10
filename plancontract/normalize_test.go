package plancontract

import (
	"maps"
	"reflect"
	"slices"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/types/known/structpb"
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

func TestScalarStreamAggregateIsMetadataAwareBlockingOperator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		scalarAggregate   bool
		wantBlockingCount int
		wantMatched       []int32
	}{
		{
			name:              "grouped stream aggregate remains streaming",
			wantBlockingCount: 0,
			wantMatched:       []int32{},
		},
		{
			name:              "scalar stream aggregate blocks until its single result is complete",
			scalarAggregate:   true,
			wantBlockingCount: 1,
			wantMatched:       []int32{1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
				{
					Index:       0,
					Kind:        spannerpb.PlanNode_RELATIONAL,
					DisplayName: "Limit",
					ChildLinks:  []*spannerpb.PlanNode_ChildLink{{ChildIndex: 1}},
				},
				{
					Index:       1,
					Kind:        spannerpb.PlanNode_RELATIONAL,
					DisplayName: "Aggregate",
					Metadata: mustMetadata(t, map[string]any{
						"iterator_type":    "Stream",
						"scalar_aggregate": tt.scalarAggregate,
					}),
					ChildLinks: []*spannerpb.PlanNode_ChildLink{{ChildIndex: 2}},
				},
				{Index: 2, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Scan"},
			}}

			operators := NormalizeOperators(plan)
			aggregate := operators[1]
			if got, want := aggregate.Family, "stream_aggregate"; got != want {
				t.Fatalf("aggregate family = %q, want %q", got, want)
			}
			if got := aggregate.ScalarAggregate; got != tt.scalarAggregate {
				t.Fatalf("scalar_aggregate = %t, want %t", got, tt.scalarAggregate)
			}
			counts := OperatorFamilyCounts(operators)
			if got := counts["blocking_operator"]; got != tt.wantBlockingCount {
				t.Fatalf("whole-plan blocking_operator count = %d, want %d", got, tt.wantBlockingCount)
			}
			if got := OperatorFamilyCountsOrEmpty(counts)["blocking_operator"]; got != tt.wantBlockingCount {
				t.Fatalf("round-tripped blocking_operator count = %d, want %d", got, tt.wantBlockingCount)
			}
			AddDerivedOperatorFamilyCounts(counts)
			if got := counts["blocking_operator"]; got != tt.wantBlockingCount {
				t.Fatalf("re-derived blocking_operator count = %d, want %d", got, tt.wantBlockingCount)
			}
			if got := operators[0].SubtreeFamilyCounts["blocking_operator"]; got != tt.wantBlockingCount {
				t.Fatalf("limit subtree blocking_operator count = %d, want %d", got, tt.wantBlockingCount)
			}
			if got := blockingOperatorUnderLimitIndexes(Query{NormalizedOperators: operators}); !slices.Equal(got, tt.wantMatched) {
				t.Fatalf("blocking operators under limit = %v, want %v", got, tt.wantMatched)
			}
			if got := predicateMatchedOperatorIndexes(Query{NormalizedOperators: operators}, "blocking_operator"); !slices.Equal(got, tt.wantMatched) {
				t.Fatalf("direct blocking_operator matches = %v, want %v", got, tt.wantMatched)
			}
			if got := slices.Contains(DerivedOperatorFamiliesForOperator(aggregate), "blocking_operator"); got != tt.scalarAggregate {
				t.Fatalf("metadata-aware derived blocking membership = %t, want %t", got, tt.scalarAggregate)
			}
		})
	}

	if got := DerivedOperatorFamilies("stream_aggregate"); len(got) != 0 {
		t.Fatalf("family-only DerivedOperatorFamilies(stream_aggregate) = %v, want empty without scalar metadata", got)
	}
}

func TestOperatorTreeDigestIncludesStableNormalizedMetadata(t *testing.T) {
	t.Parallel()

	base := map[string]any{
		"scan_format":           "Row",
		"scan_target":           "AlbumsByTitle",
		"seekable_key_size":     "1",
		"join_type":             "BUILD_SEMI",
		"join_configuration":    "ONE-TO-MANY",
		"call_type":             "Local",
		"distribution_table":    "Albums",
		"subquery_cluster_node": "4",
		"spool_name":            "CommonRows",
		"scalar_aggregate":      false,
	}
	tests := []struct {
		name  string
		key   string
		value any
	}{
		{name: "scan format", key: "scan_format", value: "Columnar"},
		{name: "scan target", key: "scan_target", value: "AlbumsByBudget"},
		{name: "seekable key size", key: "seekable_key_size", value: "2"},
		{name: "join type", key: "join_type", value: "PROBE_SEMI"},
		{name: "join configuration", key: "join_configuration", value: "MANY-TO-MANY"},
		{name: "call type", key: "call_type", value: "Global"},
		{name: "distribution table", key: "distribution_table", value: "Singers"},
		{name: "subquery cluster node", key: "subquery_cluster_node", value: "8"},
		{name: "spool name", key: "spool_name", value: "OtherRows"},
		{name: "scalar aggregate", key: "scalar_aggregate", value: true},
	}
	baseDigest := OperatorTreeDigest(singleNodePlan(t, base))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			changed := maps.Clone(base)
			changed[tt.key] = tt.value
			if got := OperatorTreeDigest(singleNodePlan(t, changed)); got == baseDigest {
				t.Fatalf("digest did not change when %s changed", tt.key)
			}
		})
	}
}

func TestOperatorTreeDigestStructuredEncodingAvoidsDelimiterCollision(t *testing.T) {
	t.Parallel()

	left := singleNodePlan(t, map[string]any{
		"scan_format": "a|b",
		"scan_target": "c",
	})
	right := singleNodePlan(t, map[string]any{
		"scan_format": "a",
		"scan_type":   "b",
		"scan_target": "|c",
	})
	if got, wantNot := OperatorTreeDigest(left), OperatorTreeDigest(right); got == wantNot {
		t.Fatalf("structured digest collided for delimiter-bearing metadata: %q", got)
	}
}

func TestOperatorTreeDigestIncludesChildLinkVariable(t *testing.T) {
	t.Parallel()

	plan := func(variable string) *spannerpb.QueryPlan {
		return &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
			{
				Index:       0,
				Kind:        spannerpb.PlanNode_RELATIONAL,
				DisplayName: "Filter Scan",
				ChildLinks:  []*spannerpb.PlanNode_ChildLink{{ChildIndex: 1, Type: "Scalar", Variable: variable}},
			},
			{Index: 1, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Function"},
		}}
	}
	if got, wantNot := OperatorTreeDigest(plan("Seek Condition")), OperatorTreeDigest(plan("Residual Condition")); got == wantNot {
		t.Fatalf("digest did not change with child-link variable: %q", got)
	}
}

func TestGenericTVFNameIsNormalizedAndDigested(t *testing.T) {
	t.Parallel()

	plan := func(name string) *spannerpb.QueryPlan {
		return &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{{
			Index:       0,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "TVF",
			Metadata:    mustMetadata(t, map[string]any{"Name": name}),
		}}}
	}
	operators := NormalizeOperators(plan("ML.PREDICT"))
	if got, want := operators[0].TVFName, "ML.PREDICT"; got != want {
		t.Fatalf("tvf_name = %q, want %q", got, want)
	}
	if got, wantNot := OperatorTreeDigest(plan("ML.PREDICT")), OperatorTreeDigest(plan("OtherTVF")); got == wantNot {
		t.Fatalf("digest did not change with TVF name: %q", got)
	}
}

func TestNodeMetadataNullIsAbsent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value *structpb.Value
	}{
		{name: "nil value"},
		{name: "protobuf null", value: structpb.NewNullValue()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &spannerpb.PlanNode{Metadata: &structpb.Struct{Fields: map[string]*structpb.Value{"scan_target": tt.value}}}
			if got := nodeMetadataRawString(node, "scan_target"); got != "" {
				t.Fatalf("nodeMetadataRawString(scan_target) = %q, want empty", got)
			}
		})
	}
}

func TestContextTraversalUsesPlanNodeKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		intermediateName string
		intermediateKind spannerpb.PlanNode_Kind
		wantFamily       string
	}{
		{
			name:             "explicit relational kind permits future adapter",
			intermediateName: "Future Relational Adapter",
			intermediateKind: spannerpb.PlanNode_RELATIONAL,
			wantFamily:       "push_broadcast_hash_join_internal_hash_join",
		},
		{
			name:             "explicit scalar kind stops traversal despite relational-looking name",
			intermediateName: "Compute",
			intermediateKind: spannerpb.PlanNode_SCALAR,
			wantFamily:       "hash_join",
		},
		{
			name:             "unspecified kind retains known display fallback",
			intermediateName: "Compute",
			intermediateKind: spannerpb.PlanNode_KIND_UNSPECIFIED,
			wantFamily:       "push_broadcast_hash_join_internal_hash_join",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
				{
					Index:       0,
					Kind:        spannerpb.PlanNode_RELATIONAL,
					DisplayName: "Push Broadcast Hash Join",
					ChildLinks:  []*spannerpb.PlanNode_ChildLink{{Type: "Map", ChildIndex: 1}},
				},
				{
					Index:       1,
					Kind:        tt.intermediateKind,
					DisplayName: tt.intermediateName,
					ChildLinks:  []*spannerpb.PlanNode_ChildLink{{ChildIndex: 2}},
				},
				{
					Index:       2,
					Kind:        spannerpb.PlanNode_RELATIONAL,
					DisplayName: "Hash Join",
					ChildLinks:  []*spannerpb.PlanNode_ChildLink{{ChildIndex: 3}},
				},
				{
					Index:       3,
					Kind:        spannerpb.PlanNode_RELATIONAL,
					DisplayName: "Scan",
					Metadata:    mustMetadata(t, map[string]any{"scan_type": "BatchScan"}),
				},
			}}
			if got := NormalizeOperators(plan)[2].Family; got != tt.wantFamily {
				t.Fatalf("internal hash join family = %q, want %q", got, tt.wantFamily)
			}
		})
	}
}

func TestOperatorFamilyDocumentedCoverage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		displayName string
		metadata    map[string]any
		want        string
	}{
		{name: "generate relation", displayName: "Generate Relation", want: "generate_relation"},
		{name: "local split union", displayName: "Local Split Union", want: "local_split_union"},
		{name: "generic tvf", displayName: "TVF", metadata: map[string]any{"Name": "ML.PREDICT"}, want: "tvf"},
		{name: "search query conversion tvf stays specialized", displayName: "TVF", metadata: map[string]any{"Name": "Search Query Conversion"}, want: "search_query_conversion_tvf"},
		{name: "data block adapter official spelling", displayName: "DataBlockToRowAdapter", want: "data_block_to_row"},
		{name: "row adapter official spelling", displayName: "RowToDataBlockAdapter", want: "row_to_data_block"},
		{name: "scan with filter scan metadata", displayName: "Scan", metadata: map[string]any{"scan_type": "FilterScan"}, want: "filter_scan"},
		{name: "historical unspaced filter scan", displayName: "FilterScan", want: "filter_scan"},
		{name: "ordinary scan remains scan", displayName: "Scan", metadata: map[string]any{"scan_type": "TableScan"}, want: "scan"},
		{
			name:        "vector index root scan remains scan",
			displayName: "Scan",
			metadata:    map[string]any{"scan_type": "VectorIndexRootScan"},
			want:        "scan",
		},
		{
			name:        "vector index leaf scan remains scan",
			displayName: "Scan",
			metadata:    map[string]any{"scan_type": "VectorIndexLeafScan"},
			want:        "scan",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &spannerpb.PlanNode{DisplayName: tt.displayName}
			if tt.metadata != nil {
				node.Metadata = mustMetadata(t, tt.metadata)
			}
			if got := OperatorFamily(node); got != tt.want {
				t.Fatalf("OperatorFamily(%q) = %q, want %q", tt.displayName, got, tt.want)
			}
			if !KnownOperatorFamily(tt.want) {
				t.Fatalf("classified family %q is absent from KnownOperatorFamilies", tt.want)
			}
		})
	}
}

func TestVectorIndexScanTypeMetadataNormalized(t *testing.T) {
	t.Parallel()

	for raw, want := range map[string]string{
		"VectorIndexRootScan": "vector_index_root_scan",
		"VectorIndexLeafScan": "vector_index_leaf_scan",
	} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			node := &spannerpb.PlanNode{Metadata: mustMetadata(t, map[string]any{"scan_type": raw})}
			if got := OperatorMetadataString(node, "scan_type"); got != want {
				t.Fatalf("OperatorMetadataString(scan_type) = %q, want %q", got, want)
			}
		})
	}
}

func singleNodePlan(t *testing.T, metadata map[string]any) *spannerpb.QueryPlan {
	t.Helper()
	return &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{{
		Index:       0,
		Kind:        spannerpb.PlanNode_RELATIONAL,
		DisplayName: "Scan",
		Metadata:    mustMetadata(t, metadata),
	}}}
}

func mustMetadata(t *testing.T, fields map[string]any) *structpb.Struct {
	t.Helper()
	metadata, err := structpb.NewStruct(fields)
	if err != nil {
		t.Fatalf("structpb.NewStruct() error = %v", err)
	}
	return metadata
}
