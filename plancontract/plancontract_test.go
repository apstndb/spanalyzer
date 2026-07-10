package plancontract

import (
	"slices"
	"testing"
)

func TestStabilityForUsesCELIdentifiersNotStringLiterals(t *testing.T) {
	tests := []struct {
		name                 string
		expression           string
		wantTier             string
		wantCheckRecommended bool
		wantReason           string
		wantReasons          []string
		wantOnlyBaseReason   bool
	}{
		{
			name:                 "metadata field select",
			expression:           `operators.exists(o, o.call_type != "" && o.subquery_cluster_node != "" && o.spool_name != "")`,
			wantTier:             StabilityNormalized,
			wantCheckRecommended: true,
			wantReason:           "contract reads metadata-derived normalized fields: call_type, spool_name, subquery_cluster_node",
		},
		{
			name:                 "spool name only",
			expression:           `operators.exists(o, o.family == "spool_scan" && o.spool_name != "")`,
			wantTier:             StabilityNormalized,
			wantCheckRecommended: true,
			wantReasons: []string{
				"contract uses the normalized plan-report view",
				"contract reads metadata-derived normalized fields: spool_name",
			},
		},
		{
			name:                 "TVF name",
			expression:           `operators.exists(o, o.family == "tvf" && o.tvf_name == "ML.PREDICT")`,
			wantTier:             StabilityNormalized,
			wantCheckRecommended: true,
			wantReason:           "contract reads metadata-derived normalized fields: tvf_name",
		},
		{
			name:                 "raw nodes variable",
			expression:           `raw_nodes.exists(n, n.display_name == "Serialize Result")`,
			wantTier:             StabilityRawPlan,
			wantCheckRecommended: false,
			wantReason:           "CEL expression references raw QueryPlan or PlanNode inputs",
		},
		{
			name:                 "string literals do not trigger metadata or raw detection",
			expression:           `operators.exists(o, o.display_name == "nodes scan_target call_type")`,
			wantTier:             StabilityNormalized,
			wantCheckRecommended: true,
			wantOnlyBaseReason:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stability := stabilityFor(Contract{CEL: tt.expression})
			if got := stability.Tier; got != tt.wantTier {
				t.Fatalf("tier = %q, want %q; stability=%+v", got, tt.wantTier, stability)
			}
			if got := stability.CheckRecommended; got != tt.wantCheckRecommended {
				t.Fatalf("check recommended = %t, want %t; stability=%+v", got, tt.wantCheckRecommended, stability)
			}
			if got, want := stability.ReplayableFromReport, tt.wantTier == StabilityNormalized; got != want {
				t.Fatalf("replayable from report = %t, want %t; stability=%+v", got, want, stability)
			}
			if tt.wantReason != "" && !slices.Contains(stability.Reasons, tt.wantReason) {
				t.Fatalf("reasons = %v, want %q", stability.Reasons, tt.wantReason)
			}
			if tt.wantReasons != nil && !slices.Equal(stability.Reasons, tt.wantReasons) {
				t.Fatalf("reasons = %v, want %v", stability.Reasons, tt.wantReasons)
			}
			if tt.wantOnlyBaseReason {
				if got, want := stability.Reasons, []string{"contract uses the normalized plan-report view"}; !slices.Equal(got, want) {
					t.Fatalf("reasons = %v, want %v", got, want)
				}
			}
		})
	}
}

func TestValidateCELExpressionRejectsExecutionStatsIdentifiersOnly(t *testing.T) {
	if err := validateCELExpression(`operators.exists(o, o.display_name == "execution_stats")`); err != nil {
		t.Fatalf("validateCELExpression() rejected string literal only reference: %v", err)
	}
	if err := validateCELExpression(`raw_nodes.exists(n, n.execution_stats != null)`); err == nil {
		t.Fatalf("validateCELExpression() succeeded for execution_stats field reference")
	}
}

// TestDerivedOperatorFamiliesMatchesAddDerivedOperatorFamilyCounts pins the
// per-family umbrella membership to the count derivation so the two cannot
// drift: a family contributes to an umbrella count exactly when
// DerivedOperatorFamilies reports that membership.
func TestDerivedOperatorFamiliesMatchesAddDerivedOperatorFamilyCounts(t *testing.T) {
	umbrellas := []string{"explicit_sort", "blocking_operator"}
	for _, family := range ConcreteOperatorFamilies() {
		counts := ZeroOperatorFamilyCounts()
		counts[family] = 1
		AddDerivedOperatorFamilyCounts(counts)
		derived := DerivedOperatorFamilies(family)
		for _, umbrella := range umbrellas {
			gotMember := slices.Contains(derived, umbrella)
			wantMember := counts[umbrella] == 1
			if gotMember != wantMember {
				t.Errorf("DerivedOperatorFamilies(%q) membership in %q = %t, want %t", family, umbrella, gotMember, wantMember)
			}
		}
	}
	for _, umbrella := range umbrellas {
		if got := DerivedOperatorFamilies(umbrella); len(got) != 0 {
			t.Errorf("DerivedOperatorFamilies(%q) = %v, want empty for umbrella families", umbrella, got)
		}
	}
	// The umbrella order is defined as lexicographic, the same canonical
	// order ObservedOperatorFamilies uses.
	for _, family := range ConcreteOperatorFamilies() {
		if derived := DerivedOperatorFamilies(family); !slices.IsSorted(derived) {
			t.Errorf("DerivedOperatorFamilies(%q) = %v, want lexicographic order", family, derived)
		}
	}
}

func TestOptimizerPinningTreatsMovingAliasesAsUnpinned(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "fixed numeric version", value: "8", want: true},
		{name: "fixed statistics package", value: "auto_20260709_12_34_UTC", want: true},
		{name: "empty"},
		{name: "not pinned", value: "not_pinned"},
		{name: "not recorded", value: "not_recorded"},
		{name: "latest", value: "latest"},
		{name: "latest version", value: "latest_version"},
		{name: "default version", value: "default_version"},
		{name: "moving alias case and whitespace", value: "  LATEST_VERSION  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := OptimizerVersionPinned(tt.value); got != tt.want {
				t.Errorf("OptimizerVersionPinned(%q) = %t, want %t", tt.value, got, tt.want)
			}
			if got := OptimizerStatisticsPackagePinned(tt.value); got != tt.want {
				t.Errorf("OptimizerStatisticsPackagePinned(%q) = %t, want %t", tt.value, got, tt.want)
			}
		})
	}

	if OptimizerEnvironmentPinned(OptimizerEnvironment{Version: "latest", StatisticsPackage: "auto_20260709_12_34_UTC"}) {
		t.Fatal("OptimizerEnvironmentPinned() = true with a moving optimizer version")
	}
	if !OptimizerEnvironmentPinned(OptimizerEnvironment{Version: "8", StatisticsPackage: "auto_20260709_12_34_UTC"}) {
		t.Fatal("OptimizerEnvironmentPinned() = false with fixed optimizer values")
	}

	warnings := EnvironmentWarnings(Report{Optimizer: Optimizer{Requested: OptimizerEnvironment{
		Version:           "latest_version",
		StatisticsPackage: "latest",
	}}})
	if want := []string{"optimizer_not_pinned", "statistics_package_not_pinned"}; !slices.Equal(warnings, want) {
		t.Fatalf("EnvironmentWarnings() = %v, want %v", warnings, want)
	}
}

func TestTimestampConditionAssociatesFilterScanWithItsScan(t *testing.T) {
	t.Parallel()

	query := Query{
		NormalizedOperators: []Operator{
			{Index: 0, Family: "filter_scan", ChildIndexes: []int32{1, 2}},
			{Index: 1, Family: "scan", FullScan: true},
			{Index: 2, Family: "scalar"},
			{Index: 3, Family: "scan", FullScan: true},
			{Index: 4, Family: "limit", ChildIndexes: []int32{5}},
			{Index: 5, Family: "scalar"},
		},
		OperatorEdges: []OperatorEdge{
			{ParentIndex: 0, ChildIndex: 2, Type: "Timestamp Condition"},
			// A same-named edge on a non-scan operator must not satisfy the
			// predefined timestamp-condition contracts.
			{ParentIndex: 4, ChildIndex: 5, Type: "Timestamp Condition"},
		},
	}

	if got, want := timestampConditionOperatorIndexes(query), []int32{0}; !slices.Equal(got, want) {
		t.Fatalf("timestamp-condition owners = %v, want %v", got, want)
	}
	if got, want := fullScanWithoutTimestampConditionOperatorIndexes(query), []int32{3}; !slices.Equal(got, want) {
		t.Fatalf("unprotected full scans = %v, want %v", got, want)
	}
}

func TestTimestampConditionSupportsDirectScanAndRejectsAmbiguousWrapper(t *testing.T) {
	t.Parallel()

	t.Run("legacy direct scan edge", func(t *testing.T) {
		query := Query{
			NormalizedOperators: []Operator{
				{Index: 0, Family: "scan", FullScan: true, ChildIndexes: []int32{1}},
				{Index: 1, Family: "scalar"},
			},
			OperatorEdges: []OperatorEdge{{ParentIndex: 0, ChildIndex: 1, Type: " timestamp condition "}},
		}
		if got := fullScanWithoutTimestampConditionOperatorIndexes(query); len(got) != 0 {
			t.Fatalf("unprotected full scans = %v, want none", got)
		}
	})

	t.Run("two direct scans are ambiguous", func(t *testing.T) {
		query := Query{
			NormalizedOperators: []Operator{
				{Index: 0, Family: "filter_scan", ChildIndexes: []int32{1, 2, 3}},
				{Index: 1, Family: "scan", FullScan: true},
				{Index: 2, Family: "scan", FullScan: true},
				{Index: 3, Family: "scalar"},
			},
			OperatorEdges: []OperatorEdge{{ParentIndex: 0, ChildIndex: 3, Type: "Timestamp Condition"}},
		}
		if got, want := fullScanWithoutTimestampConditionOperatorIndexes(query), []int32{1, 2}; !slices.Equal(got, want) {
			t.Fatalf("unprotected full scans = %v, want %v", got, want)
		}
	})

	t.Run("filter scan can carry full scan metadata", func(t *testing.T) {
		query := Query{NormalizedOperators: []Operator{{Index: 0, Family: "filter_scan", FullScan: true}}}
		if got, want := fullScanOperatorIndexes(query), []int32{0}; !slices.Equal(got, want) {
			t.Fatalf("full scans = %v, want %v", got, want)
		}
		if got, want := fullScanWithoutTimestampConditionOperatorIndexes(query), []int32{0}; !slices.Equal(got, want) {
			t.Fatalf("unprotected full scans = %v, want %v", got, want)
		}
		query.OperatorEdges = []OperatorEdge{{ParentIndex: 0, Type: "Timestamp Condition"}}
		if got := fullScanWithoutTimestampConditionOperatorIndexes(query); len(got) != 0 {
			t.Fatalf("unprotected full scans = %v, want none", got)
		}
	})
}

func TestDocumentedTopologyCELRecipesCompile(t *testing.T) {
	t.Parallel()

	expressions := map[string]string{
		"full scan without timestamp condition": `
operators.all(o,
  !((o.family == "scan" || o.family == "filter_scan") && o.full_scan) ||
  operator_edges.exists(e,
    e.type == "Timestamp Condition" &&
    (e.parent_index == o.index ||
     operators.exists(f,
       f.family == "filter_scan" &&
       e.parent_index == f.index &&
       f.child_indexes.exists(i, i == o.index) &&
       operators.filter(s,
         s.family == "scan" &&
         f.child_indexes.exists(i, i == s.index)).size() == 1))))`,
		"require timestamp condition": `
operator_edges.exists(e,
  e.type == "Timestamp Condition" &&
  operators.exists(o,
    o.index == e.parent_index &&
    (o.family == "scan" || o.family == "filter_scan")))`,
		"blocking operator under limit": `
operators.all(limit,
  !(limit.family == "limit" || limit.display_name.endsWith("Sort Limit")) ||
  operators.all(descendant,
    !limit.descendant_indexes.exists(index, index == descendant.index) ||
    descendant.family != "full_sort" &&
    descendant.family != "hash_aggregate" &&
    descendant.family != "hash_join" &&
    descendant.family != "push_broadcast_hash_join" &&
    descendant.family != "aggregate" &&
    descendant.family != "join" &&
    descendant.family != "bloom_filter_build" &&
    descendant.family != "spool_build" &&
    !(descendant.family == "stream_aggregate" &&
      descendant.scalar_aggregate)))`,
	}
	for name, expression := range expressions {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := validateCELExpression(expression); err != nil {
				t.Fatalf("documented CEL recipe does not compile: %v", err)
			}
		})
	}
}
