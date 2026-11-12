package plancontract

// MutationCountInterval preserves integer precision and an explicitly unknown
// upper bound without depending on an analyzer or a database client.
type MutationCountInterval struct {
	Lower    int64
	Upper    *int64
	Overflow bool
}

// MutationCountEvidence is the conditional preview interval exposed to CEL.
// The producer owns schema validation and observation identity checks. A point
// interval from one execution never establishes a future execution's price.
type MutationCountEvidence struct {
	Effective      MutationCountInterval
	EffectiveBasis string
	Disagreements  []string
}

func celMutationCounts(counts map[string]MutationCountEvidence) map[string]interface{} {
	out := make(map[string]interface{}, len(counts))
	for method, count := range counts {
		var upper interface{}
		if count.Effective.Upper != nil {
			upper = *count.Effective.Upper
		}
		disagreements := count.Disagreements
		if disagreements == nil {
			disagreements = []string{}
		}
		out[method] = map[string]interface{}{
			"effective":       map[string]interface{}{"lower": count.Effective.Lower, "upper": upper, "overflow": count.Effective.Overflow},
			"effective_basis": count.EffectiveBasis,
			"disagreements":   disagreements,
		}
	}
	return out
}

// A skipped Mutation-only target has no SQL PLAN, but its conditional count
// analysis is still usable. Never let that exception turn missing operators or
// mixed plan/count contracts into an empty-plan success.
func mutationCountOnlyContract(contract compiledContract) bool {
	if len(contract.predicates) != 0 || contract.program == nil {
		return false
	}
	identifiers, ok := referencedCELIdentifiers(contract.CEL)
	if !ok || !identifiers["mutation_counts"] {
		return false
	}
	for _, name := range []string{"query", "optimizer", "operators", "operator_edges", "access_paths", "operator_families", "operator_family_counts", "raw_plan", "raw_nodes"} {
		if identifiers[name] {
			return false
		}
	}
	return true
}
