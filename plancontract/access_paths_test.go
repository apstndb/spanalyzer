package plancontract

import (
	"reflect"
	"slices"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/proto"
)

func accessPathFixture(t *testing.T) *spannerpb.QueryPlan {
	t.Helper()
	ref := func(index int32, text string) *spannerpb.PlanNode {
		return &spannerpb.PlanNode{Index: index, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Reference", ShortRepresentation: &spannerpb.PlanNode_ShortRepresentation{Description: text}}
	}
	condition := func(index int32, text string, children ...int32) *spannerpb.PlanNode {
		node := &spannerpb.PlanNode{Index: index, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Function", ShortRepresentation: &spannerpb.PlanNode_ShortRepresentation{Description: text}}
		for _, child := range children {
			node.ChildLinks = append(node.ChildLinks, &spannerpb.PlanNode_ChildLink{ChildIndex: child})
		}
		return node
	}
	return &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{Index: 0, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Hash Join", ChildLinks: []*spannerpb.PlanNode_ChildLink{{ChildIndex: 1}, {ChildIndex: 10}}},
		{Index: 1, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Filter Scan", Metadata: mustMetadata(t, map[string]any{"seekable_key_size": "2"}), ChildLinks: []*spannerpb.PlanNode_ChildLink{{ChildIndex: 2}, {ChildIndex: 8, Type: "Residual Condition"}}},
		{Index: 2, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Scan", Metadata: mustMetadata(t, map[string]any{"scan_type": "IndexScan", "scan_target": "EventsByShard"}), ChildLinks: []*spannerpb.PlanNode_ChildLink{{ChildIndex: 3, Variable: "shard'2"}, {ChildIndex: 4, Variable: "created"}, {ChildIndex: 5, Variable: "owner"}, {ChildIndex: 7, Type: "Seek Condition"}}},
		ref(3, "Shard"), ref(4, "CreatedAt"), ref(5, "Owner"),
		condition(7, "(($shard'2 = @shard) AND ($created >= @start))", 20, 21),
		condition(8, "($owner = @owner)", 22),
		{Index: 10, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Filter Scan", Metadata: mustMetadata(t, map[string]any{"seekable_key_size": "0"}), ChildLinks: []*spannerpb.PlanNode_ChildLink{{ChildIndex: 11}, {ChildIndex: 14, Type: "Residual Condition"}}},
		{Index: 11, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Scan", Metadata: mustMetadata(t, map[string]any{"scan_type": "TableScan", "scan_target": "Latest", "Full scan": true}), ChildLinks: []*spannerpb.PlanNode_ChildLink{{ChildIndex: 12, Variable: "other_owner"}}},
		ref(12, "Owner"), condition(14, "($other_owner = @owner)", 23),
		ref(20, "$shard'2"), ref(21, "$created"), ref(22, "$owner"), ref(23, "$other_owner"),
	}}
}

func TestNormalizeAccessPathsKeepsScanPredicatesSeparate(t *testing.T) {
	plan := accessPathFixture(t)
	before := proto.Clone(plan)
	paths := NormalizeAccessPaths(plan, []ScanTarget{{Name: "EventsByShard", Kind: "secondary_index", BaseTable: "Events", KeyColumns: []string{"Shard", "CreatedAt", "Id"}, StoredColumns: []string{"Owner"}}})
	if len(paths) != 2 {
		t.Fatalf("paths = %#v", paths)
	}
	seek, full := paths[0], paths[1]
	if seek.ScanIndex != 2 || seek.FilterIndex == nil || *seek.FilterIndex != 1 || !seek.ConditionsComplete || !seek.HasSeekCondition || !seek.HasResidualCondition || seek.FullScan {
		t.Fatalf("seek path = %#v", seek)
	}
	if !slices.Equal(seek.SeekColumns, []string{"CreatedAt", "Shard"}) || !slices.Equal(seek.ResidualColumns, []string{"Owner"}) || seek.SeekConditions[0].Expression != "(($shard'2 = @shard) AND ($created >= @start))" {
		t.Fatalf("seek predicates = %#v", seek)
	}
	if seek.BaseTable != "Events" || seek.DeclaredKeyCount == nil || *seek.DeclaredKeyCount != 3 || seek.RangeSeekableKeySize == nil || *seek.RangeSeekableKeySize != 2 {
		t.Fatalf("schema and range facts = %#v", seek)
	}
	if !full.FullScan || full.HasSeekCondition || !full.HasResidualCondition || full.RangeSeekableKeySize != nil || full.ScanTarget != "Latest" || len(full.SeekColumns) != 0 {
		t.Fatalf("full path = %#v", full)
	}
	if !proto.Equal(plan, before) {
		t.Fatal("normalization mutated the input")
	}
	slices.Reverse(plan.PlanNodes)
	if got := NormalizeAccessPaths(plan, []ScanTarget{{Name: "EventsByShard", Kind: "secondary_index", BaseTable: "Events", KeyColumns: []string{"Shard", "CreatedAt", "Id"}, StoredColumns: []string{"Owner"}}}); !reflect.DeepEqual(got, paths) {
		t.Fatal("normalization depends on PlanNodes order")
	}
}

func TestAccessPathZeroRangeMetadataDoesNotRejectPointSeek(t *testing.T) {
	plan := accessPathFixture(t)
	plan.PlanNodes[1].Metadata = mustMetadata(t, map[string]any{"seekable_key_size": "0"})
	path := NormalizeAccessPaths(plan, nil)[0]
	if !path.HasSeekCondition || path.RangeSeekableKeySize != nil || path.SeekableKeySize != "0" {
		t.Fatalf("point seek = %#v", path)
	}
}

func TestAccessPathAmbiguousWrapperDoesNotAttachAnotherScansResidual(t *testing.T) {
	plan := accessPathFixture(t)
	plan.PlanNodes[1].ChildLinks = append(plan.PlanNodes[1].ChildLinks, &spannerpb.PlanNode_ChildLink{ChildIndex: 11})
	paths := NormalizeAccessPaths(plan, nil)
	if paths[0].ConditionsComplete || paths[1].ConditionsComplete || paths[0].HasResidualCondition || paths[0].FilterIndex != nil {
		t.Fatalf("ambiguous paths = %#v", paths)
	}
}

func TestAccessPathCELUsesPredicatesAndNullableRangeSize(t *testing.T) {
	plan := accessPathFixture(t)
	query := Query{Name: "Example", Status: "ok", AccessPaths: NormalizeAccessPaths(plan, nil)}
	file := File{Version: FileVersionV1Alpha, Contracts: []Contract{{Name: "Scoped", Target: "query/Example", CEL: `access_paths.exists(p, p.scan_target == "EventsByShard" && p.conditions_complete && p.has_seek_condition && p.seek_columns == ["CreatedAt", "Shard"] && p.residual_columns == ["Owner"] && p.range_seekable_key_size == 2) && access_paths.exists(p, p.scan_target == "Latest" && p.full_scan && !p.has_seek_condition && p.range_seekable_key_size == null && p.residual_conditions.exists(c, c.expression == "($other_owner = @owner)"))`}}}
	result, err := Evaluate(Report{Queries: []Query{query}}, file)
	if err != nil || result.Summary.Failed != 0 || result.Summary.Passed != 1 {
		t.Fatalf("Evaluate() = %#v, %v", result, err)
	}
}

func TestNormalizedExpressionsAndDigestRetainConditionChanges(t *testing.T) {
	plan := accessPathFixture(t)
	before := OperatorTreeDigest(plan)
	for _, node := range plan.PlanNodes {
		if node.GetIndex() == 8 {
			node.ShortRepresentation.Description = "($owner != @owner)"
		}
	}
	if OperatorTreeDigest(plan) == before {
		t.Fatal("predicate change did not affect the versioned digest")
	}
	for _, edge := range NormalizeOperatorEdges(plan) {
		if edge.ChildIndex == 8 && edge.Expression != "($owner != @owner)" {
			t.Fatalf("condition edge = %#v", edge)
		}
	}
}

func TestUnpairedFilterAboveIntermediateKeepsConditionsIncomplete(t *testing.T) {
	plan := accessPathFixture(t)
	plan.PlanNodes = append(plan.PlanNodes, &spannerpb.PlanNode{Index: 30, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Serialize Result", ChildLinks: []*spannerpb.PlanNode_ChildLink{{ChildIndex: 2}}})
	plan.PlanNodes[1].ChildLinks[0].ChildIndex = 30
	path := NormalizeAccessPaths(plan, nil)[0]
	if path.ConditionsComplete || path.HasResidualCondition || path.FilterIndex != nil {
		t.Fatalf("unpaired ancestor filter = %#v", path)
	}
}

func TestAccessPathColumnsDoNotGuessFromExpressionText(t *testing.T) {
	plan := accessPathFixture(t)
	for _, node := range plan.PlanNodes {
		if node.GetIndex() == 22 {
			node.ShortRepresentation.Description = "$other_owner"
		}
	}
	path := NormalizeAccessPaths(plan, nil)[0]
	if len(path.ResidualColumns) != 0 || path.ResidualConditions[0].Expression != "($owner = @owner)" {
		t.Fatalf("foreign reference was guessed from expression text: %#v", path)
	}
}

func TestAccessPathSchemaTargetKindDisambiguatesObjectNames(t *testing.T) {
	plan := accessPathFixture(t)
	targets := []ScanTarget{{Name: "EventsByShard", Kind: "table", BaseTable: "EventsByShard", KeyColumns: []string{"Wrong"}}, {Name: "EventsByShard", Kind: "secondary_index", BaseTable: "Events", KeyColumns: []string{"Shard", "CreatedAt"}}}
	path := NormalizeAccessPaths(plan, targets)[0]
	if path.BaseTable != "Events" || !slices.Equal(path.KeyColumns, []string{"Shard", "CreatedAt"}) {
		t.Fatalf("resolved target = %#v", path)
	}
	delete(plan.PlanNodes[2].Metadata.Fields, "scan_type")
	path = NormalizeAccessPaths(plan, targets)[0]
	if path.BaseTable != "" || path.DeclaredKeyCount != nil || path.ScanTargetKind != "unknown" {
		t.Fatalf("ambiguous target must retain unknown facts: %#v", path)
	}
}
