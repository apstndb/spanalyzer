package planvocab

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanalyzer/plancontract"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestInspectKnownShape(t *testing.T) {
	t.Parallel()

	plan := knownDistributedScanPlan(t)
	if findings := Inspect(plan); len(findings) != 0 {
		t.Fatalf("Inspect() findings = %+v, want none", findings)
	}
}

func TestInspectAggregatesUnknownVocabularyAndRedactsIdentifiers(t *testing.T) {
	t.Parallel()

	plan := unknownHashJoinPlan(t, "private_table")
	findings := Inspect(plan)
	if len(findings) != 1 {
		t.Fatalf("Inspect() returned %d findings, want 1: %+v", len(findings), findings)
	}
	finding := findings[0]
	if got, want := finding.DisplayName, "Hash Join"; got != want {
		t.Fatalf("finding display name = %q, want %q", got, want)
	}
	data, err := json.Marshal(finding)
	if err != nil {
		t.Fatalf("json.Marshal(finding) error = %v", err)
	}
	if got := string(data); !strings.Contains(got, `"operator":"Hash Join"`) || strings.Contains(got, `"display_name"`) {
		t.Fatalf("json.Marshal(finding) = %s, want stable operator field", got)
	}
	if got, want := finding.NodeIndex, int32(0); got != want {
		t.Fatalf("finding node index = %d, want %d", got, want)
	}
	wantReasons := []string{"unknown_child_link", "unknown_metadata_key", "unknown_metadata_value"}
	if !slices.Equal(finding.Reasons, wantReasons) {
		t.Fatalf("finding reasons = %v, want %v", finding.Reasons, wantReasons)
	}
	if got, want := finding.UnknownMetadataKeys, []string{"future_target"}; !slices.Equal(got, want) {
		t.Fatalf("unknown metadata keys = %v, want %v", got, want)
	}
	if got, want := finding.UnknownMetadataValues, []MetadataShape{{Key: "join_type", Value: "future_join"}}; !slices.Equal(got, want) {
		t.Fatalf("unknown metadata values = %+v, want %+v", got, want)
	}
	if strings.Contains(finding.Fingerprint, "private_table") {
		t.Fatalf("fingerprint contains identifier-like metadata: %q", finding.Fingerprint)
	}
	for _, metadata := range finding.Metadata {
		if metadata.Key == "future_target" && metadata.Value != "string" {
			t.Fatalf("future_target shape = %q, want redacted string kind", metadata.Value)
		}
		if metadata.Value == "private_table" {
			t.Fatalf("metadata contains raw identifier value: %+v", finding.Metadata)
		}
	}

	otherIdentifier := Inspect(unknownHashJoinPlan(t, "other_private_table"))[0]
	if got, want := otherIdentifier.Fingerprint, finding.Fingerprint; got != want {
		t.Fatalf("identifier-only change altered fingerprint: got %q, want %q", got, want)
	}
}

func TestInspectUsesPlancontractClassificationDiagnosticForUnknownOperator(t *testing.T) {
	t.Parallel()

	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{{
		Index:       4,
		Kind:        spannerpb.PlanNode_RELATIONAL,
		DisplayName: "Future Exchange",
		ChildLinks: []*spannerpb.PlanNode_ChildLink{{
			ChildIndex: 99,
		}},
	}}}
	findings := Inspect(plan)
	if len(findings) != 1 {
		t.Fatalf("Inspect() returned %d findings, want 1", len(findings))
	}
	wantReasons := []string{"operator_family_unknown"}
	if !slices.Equal(findings[0].Reasons, wantReasons) {
		t.Fatalf("finding reasons = %v, want %v", findings[0].Reasons, wantReasons)
	}
	if got, want := len(findings[0].Diagnostics), 1; got != want {
		t.Fatalf("diagnostic count = %d, want %d", got, want)
	}
	if got, want := findings[0].Diagnostics[0].ID, "operator_family_unknown"; got != want {
		t.Fatalf("diagnostic ID = %q, want %q", got, want)
	}
}

func TestObserverLogsUnknownFingerprintOnce(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	observer := NewObserver(logger)
	plan := unknownHashJoinPlan(t, "private_table")

	for range 2 {
		if findings := observer.Observe(context.Background(), plan); len(findings) != 1 {
			t.Fatalf("Observe() returned %d findings, want 1", len(findings))
		}
	}
	if got, want := strings.Count(output.String(), `"event":"`+logEvent+`"`), 1; got != want {
		t.Fatalf("logged event count = %d, want %d\n%s", got, want, output.String())
	}
	if strings.Contains(output.String(), "private_table") {
		t.Fatalf("log contains identifier-like metadata:\n%s", output.String())
	}
}

func TestObserverBoundsFingerprintSetAndReportsSuppression(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	observer := NewObserverWithOptions(ObserverOptions{
		Logger:          slog.New(slog.NewJSONHandler(&output, nil)),
		MaxFingerprints: 1,
	})
	plans := []*spannerpb.QueryPlan{
		unknownHashJoinPlanWithJoinType(t, "private_table", "FUTURE_JOIN"),
		unknownHashJoinPlanWithJoinType(t, "private_table", "ANOTHER_JOIN"),
		unknownHashJoinPlanWithJoinType(t, "private_table", "THIRD_JOIN"),
	}
	for _, plan := range plans {
		observer.Observe(context.Background(), plan)
	}
	if got, want := observer.SuppressedCount(), uint64(2); got != want {
		t.Fatalf("SuppressedCount() = %d, want %d", got, want)
	}
	if got, want := strings.Count(output.String(), `"event":"`+logEvent+`"`), 1; got != want {
		t.Fatalf("finding log count = %d, want %d\n%s", got, want, output.String())
	}
	if got, want := strings.Count(output.String(), `"event":"`+suppressionLogEvent+`"`), 1; got != want {
		t.Fatalf("suppression log count = %d, want %d\n%s", got, want, output.String())
	}
}

func TestInspectDetectsUnexpectedRepeatedChildLink(t *testing.T) {
	t.Parallel()

	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       0,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Hash Join",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{ChildIndex: 1, Type: "Build"},
				{ChildIndex: 2, Type: "Build"},
			},
		},
		{Index: 1, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Scan"},
		{Index: 2, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Scan"},
	}}
	findings := Inspect(plan)
	if len(findings) != 1 {
		t.Fatalf("Inspect() returned %d findings, want 1: %+v", len(findings), findings)
	}
	if got, want := findings[0].Reasons, []string{"unexpected_child_link_multiplicity"}; !slices.Equal(got, want) {
		t.Fatalf("finding reasons = %v, want %v", got, want)
	}
	if got, want := findings[0].UnknownChildLinks, []ChildLinkShape{{
		Kind:     "RELATIONAL",
		Type:     "Build",
		Repeated: true,
	}}; !slices.Equal(got, want) {
		t.Fatalf("unknown child links = %+v, want %+v", got, want)
	}
}

func TestInspectNilPlanReturnsInitializedEmptySlice(t *testing.T) {
	t.Parallel()

	findings := Inspect(nil)
	if findings == nil || len(findings) != 0 {
		t.Fatalf("Inspect(nil) = %#v, want initialized empty slice", findings)
	}
}

func TestEmbeddedCatalogInfoReturnsIndependentLocalEvidence(t *testing.T) {
	t.Parallel()

	first := EmbeddedCatalogInfo()
	if first.Version != "v0alpha1" || first.Revision == "" || first.Blob == "" {
		t.Fatalf("EmbeddedCatalogInfo() = %+v, want versioned source provenance", first)
	}
	first.LocalEvidence[0] = "changed"
	first.Inputs[0].Path = "changed"
	second := EmbeddedCatalogInfo()
	if second.LocalEvidence[0] == "changed" {
		t.Fatal("EmbeddedCatalogInfo() returned aliased LocalEvidence")
	}
	if second.Inputs[0].Path == "changed" {
		t.Fatal("EmbeddedCatalogInfo() returned aliased Inputs")
	}
}

func TestCataloguedOperatorsAreClassifiable(t *testing.T) {
	t.Parallel()

	for _, rule := range knownCatalog.Operators {
		for _, name := range rule.Names {
			rule := rule
			name := name
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				node := &spannerpb.PlanNode{
					Index:       0,
					Kind:        catalogKind(t, rule.Kind),
					DisplayName: name,
				}
				if strings.EqualFold(name, "Aggregate") {
					node.Metadata = metadata(t, map[string]any{"iterator_type": "Hash"})
				}
				operators := plancontract.NormalizeOperators(&spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{node}})
				if warnings := plancontract.ClassificationWarnings(operators); len(warnings) != 0 {
					t.Fatalf("catalogued operator is not classifiable: operators=%+v warnings=%+v", operators, warnings)
				}
			})
		}
	}
}

func knownDistributedScanPlan(t *testing.T) *spannerpb.QueryPlan {
	t.Helper()
	return &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       0,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Distributed Union",
			Metadata: metadata(t, map[string]any{
				"distribution_table":    "Singers",
				"execution_method":      "Row",
				"split_ranges_aligned":  false,
				"subquery_cluster_node": "1",
			}),
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{ChildIndex: 1},
				{ChildIndex: 2, Type: "Split Range"},
			},
		},
		{
			Index:       1,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Scan",
			Metadata: metadata(t, map[string]any{
				"Full scan":        "true",
				"execution_method": "Row",
				"scan_method":      "Automatic",
				"scan_target":      "Singers",
				"scan_type":        "TableScan",
			}),
			ChildLinks: []*spannerpb.PlanNode_ChildLink{{
				ChildIndex: 3,
				Variable:   "SingerId",
			}},
		},
		{Index: 2, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Function"},
		{Index: 3, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Reference"},
	}}
}

func unknownHashJoinPlan(t *testing.T, target string) *spannerpb.QueryPlan {
	t.Helper()
	return unknownHashJoinPlanWithJoinType(t, target, "FUTURE_JOIN")
}

func unknownHashJoinPlanWithJoinType(t *testing.T, target, joinType string) *spannerpb.QueryPlan {
	t.Helper()
	return &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       0,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Hash Join",
			Metadata: metadata(t, map[string]any{
				"future_target": target,
				"join_type":     joinType,
			}),
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{ChildIndex: 1, Type: "Build"},
				{ChildIndex: 2, Type: "Future Link"},
			},
		},
		{Index: 1, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Scan"},
		{Index: 2, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Function"},
	}}
}

func catalogKind(t *testing.T, kind string) spannerpb.PlanNode_Kind {
	t.Helper()
	switch kind {
	case "RELATIONAL":
		return spannerpb.PlanNode_RELATIONAL
	case "SCALAR":
		return spannerpb.PlanNode_SCALAR
	default:
		t.Fatalf("unsupported catalog kind %q", kind)
		return spannerpb.PlanNode_KIND_UNSPECIFIED
	}
}

func metadata(t *testing.T, values map[string]any) *structpb.Struct {
	t.Helper()
	metadata, err := structpb.NewStruct(values)
	if err != nil {
		t.Fatalf("structpb.NewStruct() error = %v", err)
	}
	return metadata
}
