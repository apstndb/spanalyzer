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
}
