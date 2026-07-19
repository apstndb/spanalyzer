package planvocab

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestFindMatchingOperators(t *testing.T) {
	t.Parallel()

	plan := matcherTestPlan(t)
	tests := []struct {
		name    string
		pattern OperatorPattern
		want    []int32
	}{
		{
			name:    "family selector",
			pattern: OperatorPattern{Family: "scan"},
			want:    []int32{2, 3},
		},
		{
			name: "typed metadata and child link on same node",
			pattern: OperatorPattern{
				DisplayName: "Scan",
				Family:      "scan",
				Metadata: []MetadataRequirement{
					{Key: "scan_method", Value: structpb.NewStringValue("Batch")},
					{Key: "scan_target"},
				},
				ChildLinks: []ChildLinkRequirement{{
					Kind:     spannerpb.PlanNode_SCALAR,
					Type:     "",
					Variable: VariablePresent,
					MinCount: 2,
				}},
			},
			want: []int32{2},
		},
		{
			name: "exact display name casing",
			pattern: OperatorPattern{
				DisplayName: "scan",
			},
			want: []int32{},
		},
		{
			name: "typed metadata does not coerce",
			pattern: OperatorPattern{
				DisplayName: "Scan",
				Metadata: []MetadataRequirement{{
					Key:   "Full scan",
					Value: structpb.NewBoolValue(true),
				}},
			},
			want: []int32{},
		},
		{
			name: "metadata and child cannot match different nodes",
			pattern: OperatorPattern{
				DisplayName: "Scan",
				Metadata: []MetadataRequirement{{
					Key:   "scan_method",
					Value: structpb.NewStringValue("Row"),
				}},
				ChildLinks: []ChildLinkRequirement{{
					Kind:     spannerpb.PlanNode_SCALAR,
					Type:     "",
					Variable: VariablePresent,
					MinCount: 2,
				}},
			},
			want: []int32{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := FindMatchingOperators(plan, tt.pattern)
			if err != nil {
				t.Fatalf("FindMatchingOperators() error = %v", err)
			}
			if !slices.Equal(result.NodeIndexes, tt.want) {
				t.Fatalf("FindMatchingOperators() nodes = %v, want %v; result=%s", result.NodeIndexes, tt.want, result)
			}
		})
	}
}

func TestFindMatchingOperatorsChildLinkSemantics(t *testing.T) {
	t.Parallel()

	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       1,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Compute",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{ChildIndex: 2, Variable: "private_a"},
				{ChildIndex: 2, Variable: "private_b"},
				{ChildIndex: 99, Type: "Dangling"},
			},
		},
		{Index: 2, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Reference"},
	}}
	tests := []struct {
		name        string
		requirement ChildLinkRequirement
		want        bool
	}{
		{
			name: "repeated concrete child",
			requirement: ChildLinkRequirement{
				Kind:     spannerpb.PlanNode_SCALAR,
				Variable: VariablePresent,
				MinCount: 2,
			},
			want: true,
		},
		{
			name: "dangling any kind",
			requirement: ChildLinkRequirement{
				Type:     "Dangling",
				Variable: VariableAbsent,
			},
			want: true,
		},
		{
			name: "dangling concrete kind",
			requirement: ChildLinkRequirement{
				Kind:     spannerpb.PlanNode_SCALAR,
				Type:     "Dangling",
				Variable: VariableAbsent,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := FindMatchingOperators(plan, OperatorPattern{
				DisplayName: "Compute",
				ChildLinks:  []ChildLinkRequirement{tt.requirement},
			})
			if err != nil {
				t.Fatalf("FindMatchingOperators() error = %v", err)
			}
			if got := result.HasMatches(); got != tt.want {
				t.Fatalf("FindMatchingOperators().HasMatches() = %v, want %v; result=%s", got, tt.want, result)
			}
		})
	}
}

func TestFindMatchingOperatorsDiagnosticsRedactValuesAndVariables(t *testing.T) {
	t.Parallel()

	plan := matcherTestPlan(t)
	result, err := FindMatchingOperators(plan, OperatorPattern{
		DisplayName: "Scan",
		Metadata: []MetadataRequirement{{
			Key:   "scan_target",
			Value: structpb.NewStringValue("other_private_table"),
		}},
		ChildLinks: []ChildLinkRequirement{{
			Kind:     spannerpb.PlanNode_SCALAR,
			Type:     "Secret Link",
			Variable: VariablePresent,
		}},
	})
	if err != nil {
		t.Fatalf("FindMatchingOperators() error = %v", err)
	}
	diagnostic := result.String()
	for _, secret := range []string{"private_table", "other_private_table", "private_column_a", "private_column_b"} {
		if strings.Contains(diagnostic, secret) {
			t.Fatalf("FindMatchingOperators() diagnostic contains %q: %s", secret, diagnostic)
		}
	}
	if !strings.Contains(diagnostic, `metadata key "scan_target" did not match (observed string)`) {
		t.Fatalf("FindMatchingOperators() diagnostic = %q, want redacted metadata kind", diagnostic)
	}
}

func TestMatchResultJSONNamesMismatches(t *testing.T) {
	t.Parallel()

	result, err := FindMatchingOperators(matcherTestPlan(t), OperatorPattern{
		DisplayName: "Scan",
		Metadata: []MetadataRequirement{{
			Key:   "scan_method",
			Value: structpb.NewStringValue("Future"),
		}},
	})
	if err != nil {
		t.Fatalf("FindMatchingOperators() error = %v", err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if got := string(data); !strings.Contains(got, `"mismatches"`) || strings.Contains(got, `"candidates"`) {
		t.Fatalf("json.Marshal() = %s, want mismatches field only", got)
	}
}

func TestFindMatchingOperatorsValidationAndNilPlan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern OperatorPattern
	}{
		{name: "empty pattern", pattern: OperatorPattern{}},
		{name: "unknown family", pattern: OperatorPattern{Family: "future_scan"}},
		{name: "empty metadata key", pattern: OperatorPattern{DisplayName: "Scan", Metadata: []MetadataRequirement{{}}}},
		{name: "null metadata value", pattern: OperatorPattern{DisplayName: "Scan", Metadata: []MetadataRequirement{{Key: "x", Value: structpb.NewNullValue()}}}},
		{name: "unset metadata value", pattern: OperatorPattern{DisplayName: "Scan", Metadata: []MetadataRequirement{{Key: "x", Value: &structpb.Value{}}}}},
		{name: "negative child count", pattern: OperatorPattern{DisplayName: "Scan", ChildLinks: []ChildLinkRequirement{{MinCount: -1}}}},
		{name: "unknown child kind", pattern: OperatorPattern{DisplayName: "Scan", ChildLinks: []ChildLinkRequirement{{Kind: spannerpb.PlanNode_Kind(99)}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := FindMatchingOperators(nil, tt.pattern); err == nil {
				t.Fatal("FindMatchingOperators() error = nil, want validation error")
			}
		})
	}
	result, err := FindMatchingOperators(nil, OperatorPattern{DisplayName: "Scan"})
	if err != nil {
		t.Fatalf("FindMatchingOperators(nil) error = %v", err)
	}
	if result.NodeIndexes == nil || result.Mismatches == nil || result.HasMatches() {
		t.Fatalf("FindMatchingOperators(nil) = %+v, want initialized unmatched result", result)
	}
}

func matcherTestPlan(t *testing.T) *spannerpb.QueryPlan {
	t.Helper()
	return &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       2,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Scan",
			Metadata: metadata(t, map[string]any{
				"Full scan":   "true",
				"scan_method": "Batch",
				"scan_target": "private_table",
				"scan_type":   "TableScan",
			}),
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{ChildIndex: 4, Variable: "private_column_a"},
				{ChildIndex: 5, Variable: "private_column_b"},
			},
		},
		{
			Index:       3,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Scan",
			Metadata: metadata(t, map[string]any{
				"scan_method": "Row",
				"scan_target": "other_private_table",
				"scan_type":   "IndexScan",
			}),
		},
		{Index: 4, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Reference"},
		{Index: 5, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Reference"},
	}}
}
