package spanalyzer

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// MutationCountAnalysis retains model totals and per-operation explanations.
// DML Statements retain their separate limit boundaries; Count is cumulative.
type MutationCountAnalysis struct {
	Strategy       string                       `json:"strategy" yaml:"strategy"`
	Count          MutationCountInterval        `json:"total_under_model" yaml:"total_under_model"`
	LimitScope     string                       `json:"limit_scope" yaml:"limit_scope"`
	LimitVerdict   string                       `json:"limit_verdict" yaml:"limit_verdict"`
	Results        []MutationCountEstimate      `json:"operations" yaml:"operations"`
	OperationCount int                          `json:"normalized_operation_count" yaml:"normalized_operation_count"`
	Reasons        []string                     `json:"batch_uncertainties,omitempty" yaml:"batch_uncertainties,omitempty"`
	GateEligible   bool                         `json:"gate_eligible" yaml:"gate_eligible"`
	Statements     []MutationCountStatement     `json:"dml_statements,omitempty" yaml:"dml_statements,omitempty"`
	BufferedCommit *MutationCountBufferedCommit `json:"buffered_commit,omitempty" yaml:"buffered_commit,omitempty"`
}

// MutationCountStatement aggregates normalized rows from one DML statement.
type MutationCountStatement struct {
	Ordinal int                   `json:"ordinal" yaml:"ordinal"`
	Count   MutationCountInterval `json:"count" yaml:"count"`
	Verdict string                `json:"limit_verdict" yaml:"limit_verdict"`
}

func analyzeMutationCount(s *mutationCountSchemaModel, ops []mutationCountOperation, strategy string) (MutationCountAnalysis, error) {
	if strategy != "shape" && strategy != "bounds" && strategy != "row-image" {
		return MutationCountAnalysis{}, fmt.Errorf("unknown strategy %q", strategy)
	}
	hasDML, hasBuffered := false, false
	for _, op := range ops {
		if op.Channel == "dml" {
			hasDML = true
		} else {
			hasBuffered = true
		}
	}
	if hasDML && hasBuffered {
		var dml, buffered []mutationCountOperation
		for _, op := range ops {
			if op.Channel == "dml" {
				dml = append(dml, op)
			} else {
				buffered = append(buffered, op)
			}
		}
		return analyzeMixedMutationCount(s, dml, buffered, strategy)
	}
	a := MutationCountAnalysis{Strategy: strategy, Count: mutationFixed(0), LimitScope: "mutation_api_commit", OperationCount: len(ops)}
	groups := map[string][]int{}
	for n, op := range ops {
		r, e := estimateMutationCount(s, op, strategy)
		if e != nil {
			return a, fmt.Errorf("operation %d: %w", n, e)
		}
		a.Results = append(a.Results, r)
		if op.Channel == "dml" {
			a.LimitScope = "dml_statements_separately"
		} else if key, ok := mutationRowIdentity(s, op); ok {
			groups[key] = append(groups[key], n)
		}
	}
	covered := map[int]bool{}
	for _, ns := range groups {
		if len(ns) < 2 || strategy == "shape" {
			continue
		}
		updates := true
		for _, n := range ns {
			updates = updates && ops[n].Kind == "update"
		}
		if !updates {
			q := mutationFixed(0)
			for _, n := range ns {
				q = mutationAdd(q, a.Results[n].Count)
				covered[n] = true
			}
			q.Lower = 0
			a.Count = mutationAdd(a.Count, q)
			a.Reasons = append(a.Reasons, "mixed_same_key_operations_not_calibrated")
			continue
		}
		// Omni LTS.2 and the retained managed control coalesce index maintenance
		// for repeated buffered updates, while charging each input's base cells.
		// DML statements are separate
		// boundaries and never enter this branch. This is a calibration, not a
		// claim about arbitrary mixed inserts/replaces/deletes or service-wide behavior.
		merged := ops[ns[0]]
		merged.Values = mutationLowerMap(merged.Values)
		merged.Columns = append([]string(nil), merged.Columns...)
		base := mutationFixed(0)
		t := s.Tables[strings.ToLower(merged.Table)]
		for _, n := range ns {
			covered[n] = true
			r := MutationCountEstimate{}
			base = mutationAdd(base, mutationBaseWriteCost(t, ops[n], strategy, true, mutationTouchedColumns(t, ops[n], true), &r))
			for _, c := range ops[n].Columns {
				if !mutationContainsFold(merged.Columns, c) {
					merged.Columns = append(merged.Columns, c)
				}
			}
			for k, v := range ops[n].Values {
				merged.Values[k] = v
			}
		}
		r, e := estimateMutationCount(s, merged, strategy)
		if e != nil {
			return a, e
		}
		singleBase := mutationBaseWriteCost(t, merged, strategy, true, mutationTouchedColumns(t, merged, true), &MutationCountEstimate{})
		if singleBase.exact() && r.Count.Upper != nil {
			r.Count.Lower = max(0, r.Count.Lower-singleBase.Lower)
			hi := max(0, *r.Count.Upper-singleBase.Lower)
			r.Count.Upper = &hi
			a.Count = mutationAdd(a.Count, mutationAdd(base, r.Count))
		} else {
			a.Count = mutationAdd(a.Count, mutationUncertain(base.Lower, nil))
		}
		a.Reasons = append(a.Reasons, "same_key_buffered_update_index_coalescing")
	}
	deleteTables := map[string]int{}
	hasDelete, hasWrite := false, false
	for n, op := range ops {
		if op.Channel == "mutation" {
			if op.Kind == "delete" {
				hasDelete = true
			} else {
				hasWrite = true
			}
		}
		if !covered[n] {
			q := a.Results[n].Count
			if strategy != "shape" && op.Channel == "mutation" && op.Kind == "delete" {
				deleteTables[strings.ToLower(op.Table)]++
				if deleteTables[strings.ToLower(op.Table)] > 1 {
					q.Lower = 0
					a.Reasons = append(a.Reasons, "delete_keyset_overlap_not_normalized")
				}
			}
			a.Count = mutationAdd(a.Count, q)
		}
	}
	if strategy != "shape" && hasDelete && hasWrite {
		a.Count = mutationUncertain(0, nil)
		a.Reasons = append(a.Reasons, "mixed_delete_write_interaction_not_calibrated")
	}
	if strategy != "shape" && mutationDeleteCascadesOverlap(s, deleteTables) {
		// Old-row assertions describe the pre-commit state. A preceding parent
		// cascade can remove entries before a later explicit child delete, so
		// independently counted removals are an upper bound, not additive facts.
		a.Count.Lower = 0
		a.Reasons = append(a.Reasons, "delete_cascade_overlap_not_normalized")
	}
	if a.LimitScope == "dml_statements_separately" {
		statements := map[int]MutationCountInterval{}
		for n, op := range ops {
			q, ok := statements[op.Statement]
			if !ok {
				q = mutationFixed(0)
			}
			statements[op.Statement] = mutationAdd(q, a.Results[n].Count)
		}
		var ordinals []int
		for id := range statements {
			ordinals = append(ordinals, id)
		}
		slices.Sort(ordinals)
		a.LimitVerdict = "within_each_model_statement_limit_only"
		for _, id := range ordinals {
			r := MutationCountEstimate{Strategy: strategy, Count: statements[id]}
			classifyMutationCount(&r)
			a.Statements = append(a.Statements, MutationCountStatement{Ordinal: id, Count: r.Count, Verdict: r.LimitVerdict})
			if r.Count.Lower > 80000 {
				a.LimitVerdict = "model_exceeds_dml_statement_limit"
			} else if (r.Count.Upper == nil || *r.Count.Upper > 80000) && a.LimitVerdict != "model_exceeds_dml_statement_limit" {
				a.LimitVerdict = "indeterminate"
			}
		}
		if strategy == "shape" {
			a.LimitVerdict = "comparison_only"
		}
	} else {
		r := MutationCountEstimate{Strategy: strategy, Count: a.Count}
		classifyMutationCount(&r)
		a.LimitVerdict = r.LimitVerdict
	}
	slices.Sort(a.Reasons)
	a.Reasons = slices.Compact(a.Reasons)
	return a, nil
}

func mutationDeleteCascadesOverlap(s *mutationCountSchemaModel, tables map[string]int) bool {
	for parent := range tables {
		for _, child := range s.cascadeDescendants(parent) {
			if tables[strings.ToLower(child.Table.Name.String())] > 0 {
				return true
			}
		}
	}
	for _, fk := range s.ForeignKeys {
		if tables[strings.ToLower(fk.Referenced)] > 0 && tables[strings.ToLower(fk.Table)] > 0 && (strings.Contains(fk.DeleteAction, "CASCADE") || strings.Contains(fk.DeleteAction, "SET NULL")) {
			return true
		}
	}
	return false
}

func mutationRowIdentity(s *mutationCountSchemaModel, op mutationCountOperation) (string, bool) {
	if op.Kind == "delete" || op.Channel == "dml" {
		return "", false
	}
	t := s.Tables[strings.ToLower(op.Table)]
	if t == nil {
		return "", false
	}
	var key []interface{}
	for _, pk := range mutationPrimaryKeys(t.Table) {
		v, ok := op.Values[strings.ToLower(pk)]
		if !ok {
			return "", false
		}
		if sv, ok := v.(string); ok && sv == "spanner.commit_timestamp()" {
			return "", false
		}
		key = append(key, v)
	}
	b, err := json.Marshal(key)
	return strings.ToLower(op.Table) + ":" + string(b), err == nil
}
