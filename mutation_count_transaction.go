package spanalyzer

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// MutationCountTransaction describes DML executed in statement order followed
// by buffered wire mutations in commit order. Both inputs must be nonempty.
// It does not describe SDK call order: buffered mutations execute after DML,
// even when the application buffered them first. No statement is executed.
type MutationCountTransaction struct {
	Statements []string        `json:"statements" yaml:"statements"`
	Mutations  json.RawMessage `json:"mutations" yaml:"mutations"`
}

// AnalyzeTransaction estimates one mixed DML and Mutation API transaction.
// Each DML statement and the buffered commit retain separate quota boundaries;
// the cumulative total is not compared to a single 80,000-mutation limit.
// Same-table or transitively related writes keep an unknown combined interval
// instead of assuming independent index or cascade effects. A matching whole-
// transaction observation can refine the total, never its individual parts.
func (a *MutationCountAnalyzer) AnalyzeTransaction(input MutationCountTransaction, options MutationCountOptions) (MutationCountReport, error) {
	ops, err := a.dmlOperations(input.Statements)
	if err != nil {
		return MutationCountReport{}, err
	}
	buffered, err := decodeMutationCountInput(a.schema, input.Mutations)
	if err != nil {
		return MutationCountReport{}, err
	}
	if len(buffered) == 0 {
		return MutationCountReport{}, fmt.Errorf("mixed transaction requires buffered mutations")
	}
	// Canonical JSON ignores object layout while retaining SQL text and both
	// sequence orders. A subset observation cannot acquire this input identity.
	var mutations interface{}
	if err := json.Unmarshal(input.Mutations, &mutations); err != nil {
		return MutationCountReport{}, err
	}
	b, err := json.Marshal(struct {
		Statements []string    `json:"statements"`
		Mutations  interface{} `json:"mutations"`
	}{input.Statements, mutations})
	if err != nil {
		return MutationCountReport{}, err
	}
	return a.analyze(append(ops, buffered...), digestMutationInput("mixed_transaction", b), "mixed_dml_mutation_transaction", options)
}

// MutationCountBufferedCommit retains only the buffered subset's modeled
// boundary. A whole-transaction CommitStats observation does not refine it.
type MutationCountBufferedCommit struct {
	Count   MutationCountInterval `json:"count" yaml:"count"`
	Verdict string                `json:"limit_verdict" yaml:"limit_verdict"`
}

func analyzeMixedMutationCount(s *mutationCountSchemaModel, dml, buffered []mutationCountOperation, strategy string) (MutationCountAnalysis, error) {
	statements, err := analyzeMutationCount(s, dml, strategy)
	if err != nil {
		return MutationCountAnalysis{}, err
	}
	commit, err := analyzeMutationCount(s, buffered, strategy)
	if err != nil {
		return MutationCountAnalysis{}, err
	}
	a := MutationCountAnalysis{
		Strategy: strategy, Count: mutationAdd(statements.Count, commit.Count),
		LimitScope:   "dml_statements_and_buffered_commit_separately",
		LimitVerdict: "within_each_model_boundary_only",
		Results:      append(statements.Results, commit.Results...), OperationCount: len(dml) + len(buffered),
		Statements:     statements.Statements,
		BufferedCommit: &MutationCountBufferedCommit{Count: commit.Count, Verdict: commit.LimitVerdict},
		Reasons:        append(statements.Reasons, commit.Reasons...),
	}
	// The documented execution order is not an additivity guarantee. In
	// particular DML can create the row later Ack'ed by the buffered commit.
	// Do not reuse an initial old image or infer cross-stage coalescing. See
	// knowledge/research/write-analysis/mixed-transactions-and-evidence-20261112.md.
	if mutationStagesRelated(s, dml, buffered) {
		a.Count = mutationUncertain(0, nil)
		a.BufferedCommit.Count = mutationUncertain(0, nil)
		a.BufferedCommit.Verdict = "indeterminate"
		a.Reasons = append(a.Reasons, "mixed_stage_related_writes_not_calibrated")
	}
	verdicts := []string{statements.LimitVerdict, a.BufferedCommit.Verdict}
	for _, v := range verdicts {
		if strings.HasPrefix(v, "model_exceeds_") {
			a.LimitVerdict = "model_exceeds_transaction_component_limit"
			break
		}
		if v == "indeterminate" {
			a.LimitVerdict = "indeterminate"
		}
	}
	if strategy == "shape" {
		a.LimitVerdict = "comparison_only"
	}
	slices.Sort(a.Reasons)
	a.Reasons = slices.Compact(a.Reasons)
	return a, nil
}

func mutationStagesRelated(s *mutationCountSchemaModel, dml, buffered []mutationCountOperation) bool {
	// An undirected closure is deliberately conservative: shared ancestors or
	// FK targets can connect indirect effects even when target names differ.
	related := map[string]bool{}
	for _, op := range dml {
		related[strings.ToLower(op.Table)] = true
	}
	connect := func(left, right string) bool {
		left, right = strings.ToLower(left), strings.ToLower(right)
		if left == "" || right == "" || related[left] == related[right] {
			return false
		}
		related[left], related[right] = true, true
		return true
	}
	for changed := true; changed; {
		changed = false
		for name, t := range s.Tables {
			changed = connect(name, t.Parent) || changed
		}
		for _, fk := range s.ForeignKeys {
			changed = connect(fk.Table, fk.Referenced) || changed
		}
	}
	for _, op := range buffered {
		if related[strings.ToLower(op.Table)] {
			return true
		}
	}
	return false
}
