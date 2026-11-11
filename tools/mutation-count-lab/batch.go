package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type analysis struct {
	Strategy       string           `json:"strategy"`
	Count          quantity         `json:"total_under_model"`
	LimitScope     string           `json:"limit_scope"`
	LimitVerdict   string           `json:"limit_verdict"`
	Results        []countResult    `json:"operations"`
	OperationCount int              `json:"normalized_operation_count"`
	Reasons        []string         `json:"batch_uncertainties,omitempty"`
	GateEligible   bool             `json:"gate_eligible"`
	Statements     []statementCount `json:"dml_statements,omitempty"`
}

type statementCount struct {
	Ordinal int      `json:"ordinal"`
	Count   quantity `json:"count"`
	Verdict string   `json:"limit_verdict"`
}

func analyze(s *schemaModel, ops []writeOperation, strategy string) (analysis, error) {
	if strategy != "shape" && strategy != "bounds" && strategy != "row-image" {
		return analysis{}, fmt.Errorf("unknown strategy %q", strategy)
	}
	a := analysis{Strategy: strategy, Count: fixed(0), LimitScope: "mutation_api_commit", OperationCount: len(ops)}
	groups := map[string][]int{}
	for n, op := range ops {
		r, e := estimate(s, op, strategy)
		if e != nil {
			return a, fmt.Errorf("operation %d: %w", n, e)
		}
		a.Results = append(a.Results, r)
		if op.Channel == "dml" {
			a.LimitScope = "dml_statements_separately"
		} else if key, ok := rowIdentity(s, op); ok {
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
			q := fixed(0)
			for _, n := range ns {
				q = add(q, a.Results[n].Count)
				covered[n] = true
			}
			q.Lower = 0
			a.Count = add(a.Count, q)
			a.Reasons = append(a.Reasons, "mixed_same_key_operations_not_calibrated")
			continue
		}
		// Omni LTS.2 coalesces index maintenance for repeated buffered updates,
		// while charging each input's base cells. DML statements are separate
		// boundaries and never enter this branch. This is a calibration, not a
		// claim about arbitrary mixed inserts/replaces/deletes or managed Spanner.
		merged := ops[ns[0]]
		merged.Values = lowerMap(merged.Values)
		merged.Columns = append([]string(nil), merged.Columns...)
		base := fixed(0)
		t := s.Tables[strings.ToLower(merged.Table)]
		for _, n := range ns {
			covered[n] = true
			r := countResult{}
			base = add(base, baseWriteCost(t, ops[n], strategy, true, &r))
			for _, c := range ops[n].Columns {
				if !containsFold(merged.Columns, c) {
					merged.Columns = append(merged.Columns, c)
				}
			}
			for k, v := range ops[n].Values {
				merged.Values[k] = v
			}
		}
		r, e := estimate(s, merged, strategy)
		if e != nil {
			return a, e
		}
		singleBase := baseWriteCost(t, merged, strategy, true, &countResult{})
		if singleBase.exact() && r.Count.Upper != nil {
			r.Count.Lower = max(0, r.Count.Lower-singleBase.Lower)
			hi := max(0, *r.Count.Upper-singleBase.Lower)
			r.Count.Upper = &hi
			a.Count = add(a.Count, add(base, r.Count))
		} else {
			a.Count = add(a.Count, uncertain(base.Lower, nil))
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
			a.Count = add(a.Count, q)
		}
	}
	if strategy != "shape" && hasDelete && hasWrite {
		a.Count = uncertain(0, nil)
		a.Reasons = append(a.Reasons, "mixed_delete_write_interaction_not_calibrated")
	}
	if a.LimitScope == "dml_statements_separately" {
		statements := map[int]quantity{}
		for n, op := range ops {
			if op.Channel != "dml" {
				return a, fmt.Errorf("mixed DML and Mutation API commits need separate accounting")
			}
			q, ok := statements[op.Statement]
			if !ok {
				q = fixed(0)
			}
			statements[op.Statement] = add(q, a.Results[n].Count)
		}
		var ordinals []int
		for id := range statements {
			ordinals = append(ordinals, id)
		}
		slices.Sort(ordinals)
		a.LimitVerdict = "within_each_model_statement_limit_only"
		for _, id := range ordinals {
			r := countResult{Strategy: strategy, Count: statements[id]}
			classify(&r)
			a.Statements = append(a.Statements, statementCount{Ordinal: id, Count: r.Count, Verdict: r.LimitVerdict})
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
		r := countResult{Strategy: strategy, Count: a.Count}
		classify(&r)
		a.LimitVerdict = r.LimitVerdict
	}
	return a, nil
}
func rowIdentity(s *schemaModel, op writeOperation) (string, bool) {
	if op.Kind == "delete" || op.Channel == "dml" {
		return "", false
	}
	t := s.Tables[strings.ToLower(op.Table)]
	if t == nil {
		return "", false
	}
	var key []interface{}
	for _, pk := range primaryKeys(t.Table) {
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
