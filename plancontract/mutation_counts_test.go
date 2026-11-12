package plancontract

import "testing"

func TestCountContractsWorkWithoutSQLPlanAndKeepIntegers(t *testing.T) {
	large := int64(9007199254740993)
	q := Query{TargetID: "write/InsertEvent", Name: "InsertEvent", Scope: "write", Kind: "write", Status: "skipped", Error: "mutation-only write", MutationCounts: map[string]MutationCountEvidence{"mutation": {Effective: MutationCountInterval{Lower: large, Upper: &large}, EffectiveBasis: "schema_and_supplied_cardinality"}}}
	file := File{Version: FileVersionV1Alpha, Contracts: []Contract{
		{Name: "ExactInteger", Target: q.TargetID, CEL: `mutation_counts.mutation.effective.lower == 9007199254740993 && mutation_counts.mutation.effective.upper == 9007199254740993 && mutation_counts.mutation.disagreements.size() == 0`},
		{Name: "MissingPlan", Target: q.TargetID, Use: []string{"no_full_scan"}},
		{Name: "Mixed", Target: q.TargetID, CEL: `mutation_counts.mutation.effective.lower > 0 && operators.size() == 0`},
	}}
	r, err := Evaluate(Report{Queries: []Query{q}}, file)
	if err != nil || r.EvaluatorVersion != EvaluatorVersionV3 || r.Summary.Passed != 1 || r.Summary.NotEvaluated != 2 {
		t.Fatalf("Evaluate = %+v, %v", r, err)
	}
	q.Status = "ok"
	q.MutationCounts["mutation"] = MutationCountEvidence{Effective: MutationCountInterval{Lower: 1}}
	r, err = Evaluate(Report{Queries: []Query{q}}, File{Version: FileVersionV1Alpha, Contracts: []Contract{{Name: "Bounded", Target: q.TargetID, CEL: `mutation_counts.mutation.effective.upper != null && mutation_counts.mutation.effective.upper <= 80000`}}})
	if err != nil || r.Summary.Failed != 1 {
		t.Fatalf("unknown upper bound was treated as zero: %+v %v", r, err)
	}
}
