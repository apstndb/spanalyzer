package main

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
)

type quantity struct {
	Lower    int64  `json:"lower"`
	Upper    *int64 `json:"upper"`
	Overflow bool   `json:"overflow,omitempty"`
}

func fixed(n int64) quantity                 { return quantity{Lower: n, Upper: &n} }
func uncertain(lo int64, hi *int64) quantity { return quantity{Lower: lo, Upper: hi} }
func (q quantity) exact() bool               { return !q.Overflow && q.Upper != nil && q.Lower == *q.Upper }
func add(a, b quantity) quantity {
	q := quantity{Overflow: a.Overflow || b.Overflow}
	if a.Lower > math.MaxInt64-b.Lower {
		q.Lower = math.MaxInt64
		q.Overflow = true
	} else {
		q.Lower = a.Lower + b.Lower
	}
	if a.Upper != nil && b.Upper != nil && *a.Upper <= math.MaxInt64-*b.Upper {
		v := *a.Upper + *b.Upper
		q.Upper = &v
	} else if a.Upper != nil && b.Upper != nil {
		q.Overflow = true
	}
	return q
}
func multiply(a, b quantity) quantity {
	if a.Upper != nil && *a.Upper == 0 || b.Upper != nil && *b.Upper == 0 {
		return fixed(0)
	}
	q := quantity{Overflow: a.Overflow || b.Overflow}
	if a.Lower != 0 && b.Lower > math.MaxInt64/a.Lower {
		q.Lower = math.MaxInt64
		q.Overflow = true
	} else {
		q.Lower = a.Lower * b.Lower
	}
	if a.Upper != nil && b.Upper != nil && (*a.Upper == 0 || *b.Upper <= math.MaxInt64 / *a.Upper) {
		v := *a.Upper * *b.Upper
		q.Upper = &v
	} else if a.Upper != nil && b.Upper != nil {
		q.Overflow = true
	}
	return q
}
func union(a, b quantity) quantity {
	q := quantity{Lower: min(a.Lower, b.Lower), Overflow: a.Overflow || b.Overflow}
	if a.Upper != nil && b.Upper != nil {
		v := max(*a.Upper, *b.Upper)
		q.Upper = &v
	}
	return q
}

type rowFacts struct {
	Exists           *bool                  `json:"exists,omitempty"`
	Old              map[string]interface{} `json:"old,omitempty"`
	Rows             *int64                 `json:"rows,omitempty"`
	IndexRows        map[string]int64       `json:"index_rows,omitempty"`
	CascadeIndexRows map[string]int64       `json:"cascade_index_rows,omitempty"`
}
type writeOperation struct {
	Statement   int
	Kind        string
	Table       string
	Columns     []string
	Values      map[string]interface{}
	Rows        quantity
	DeleteUnits quantity
	Facts       rowFacts
	Channel     string
	Assumptions []string
}
type countResult struct {
	Strategy                 string   `json:"strategy"`
	Table                    string   `json:"table"`
	Operation                string   `json:"operation"`
	Channel                  string   `json:"channel"`
	Count                    quantity `json:"count"`
	Expression               string   `json:"expression"`
	Classification           string   `json:"classification"`
	Reasons                  []string `json:"uncertainties,omitempty"`
	Assumptions              []string `json:"assumptions,omitempty"`
	IndexSource              string   `json:"index_source"`
	LimitScope               string   `json:"limit_scope"`
	LimitVerdict             string   `json:"limit_verdict"`
	MutationLimitedBatchRows *int64   `json:"mutation_limited_batch_rows,omitempty"`
	GateEligible             bool     `json:"gate_eligible"`
	ByteLimitChecked         bool     `json:"commit_size_limit_checked"`
}

func estimate(s *schemaModel, op writeOperation, strategy string) (countResult, error) {
	t := s.Tables[strings.ToLower(op.Table)]
	if t == nil {
		return countResult{}, fmt.Errorf("unknown writable table %q", op.Table)
	}
	if op.Facts.Rows != nil {
		if *op.Facts.Rows < 0 {
			return countResult{}, fmt.Errorf("negative row count")
		}
		op.Rows = fixed(*op.Facts.Rows)
	}
	if op.Rows.Lower < 0 || op.Rows.Upper != nil && *op.Rows.Upper < op.Rows.Lower {
		return countResult{}, fmt.Errorf("invalid row bounds")
	}
	r := countResult{Strategy: strategy, Table: t.Table.Name.String(), Operation: op.Kind, Channel: op.Channel, IndexSource: s.IndexSource, Assumptions: append([]string{"successful operation; schema and snapshot match the transaction; mutation quota only; experimental model, not a service guarantee"}, op.Assumptions...), LimitScope: "mutation_api_commit"}
	if op.Channel == "dml" {
		r.LimitScope = "dml_statement"
	}
	switch strategy {
	case "shape":
		// Deliberately retained as a falsifiable baseline. It must never be
		// accepted as a production quota gate merely because it is a scalar.
		cost := int64(len(op.Columns))
		if op.Kind == "delete" || op.Kind == "delete_range" {
			cost = 1
		}
		for _, i := range s.Indexes[strings.ToLower(op.Table)] {
			cost += 1 + int64(len(i.Stored))
		}
		r.Count = multiply(fixed(cost), op.Rows)
		r.Expression = fmt.Sprintf("rows * %d (shape baseline)", cost)
	case "bounds", "row-image":
		if strategy == "row-image" {
			r.Assumptions = append(r.Assumptions, "experimental Omni LTS.2 calibration; supplied old-row facts, not a database read")
		}
		if op.Kind == "delete" || op.Kind == "delete_range" {
			if op.Channel == "dml" {
				op.DeleteUnits = uncertain(0, op.Rows.Upper)
				r.Reasons = append(r.Reasons, "dml_delete_range_normalization")
			}
			r.Count = op.DeleteUnits
			r.Expression = "delete_units + sum(deleted_index_entries)"
			for _, i := range s.Indexes[strings.ToLower(op.Table)] {
				n := op.Rows
				if v, ok := op.Facts.IndexRows[i.Name]; ok {
					if v < 0 {
						return r, fmt.Errorf("negative index row count")
					}
					n = fixed(v)
				} else if i.NullFiltered || i.PredicateUnknown {
					n = uncertain(0, op.Rows.Upper)
					r.Reasons = append(r.Reasons, "deleted_index_membership:"+i.Name)
				}
				r.Count = add(r.Count, n)
			}
			if potential := s.potentialManaged(op.Table); potential > 0 {
				r.Count = add(r.Count, multiply(uncertain(0, &potential), op.Rows))
				r.Reasons = append(r.Reasons, "managed_fk_indexes_not_reconciled")
			}
			deletedTables := map[string]bool{strings.ToLower(op.Table): true}
			for _, child := range s.cascadeDescendants(op.Table) {
				deletedTables[strings.ToLower(child.Table.Name.String())] = true
				for _, i := range s.Indexes[strings.ToLower(child.Table.Name.String())] {
					if n, ok := op.Facts.CascadeIndexRows[i.Name]; ok {
						if n < 0 {
							return r, fmt.Errorf("negative cascade index count")
						}
						r.Count = add(r.Count, fixed(n))
					} else {
						r.Count = add(r.Count, uncertain(0, nil))
						r.Reasons = append(r.Reasons, "interleaved_child_index_entries:"+i.Name)
					}
				}
				if s.potentialManaged(child.Table.Name.String()) > 0 || len(child.Unknown) > 0 {
					r.Count = add(r.Count, uncertain(0, nil))
					r.Reasons = append(r.Reasons, "cascade_child_effects_not_reconciled")
				}
			}
			for _, f := range s.ForeignKeys {
				if deletedTables[strings.ToLower(f.Referenced)] && (strings.Contains(f.DeleteAction, "CASCADE") || strings.Contains(f.DeleteAction, "SET NULL")) {
					r.Count = add(r.Count, uncertain(0, nil))
					r.Reasons = append(r.Reasons, "foreign_key_delete_action:"+f.Name)
				}
			}
		} else {
			var branches []bool
			switch op.Kind {
			case "insert":
				branches = []bool{false}
			case "update":
				branches = []bool{true}
			case "upsert", "replace":
				branches = []bool{false, true}
			default:
				return r, fmt.Errorf("unsupported operation %q", op.Kind)
			}
			if strategy == "row-image" && op.Facts.Exists != nil {
				if op.Kind == "insert" && *op.Facts.Exists || op.Kind == "update" && !*op.Facts.Exists {
					return r, fmt.Errorf("existence fact contradicts successful %s", op.Kind)
				}
				branches = []bool{*op.Facts.Exists}
			}
			var perRow quantity
			for branch, present := range branches {
				q := baseWriteCost(t, op, strategy, present, &r)
				old := map[string]interface{}{}
				if strategy == "row-image" {
					old = lowerMap(op.Facts.Old)
				}
				for _, pk := range primaryKeys(t.Table) {
					k := strings.ToLower(pk)
					if v, ok := op.Values[k]; ok {
						old[k] = v
					}
				}
				after := map[string]interface{}{}
				if present && (op.Kind == "update" || op.Kind == "upsert") {
					for k, v := range old {
						after[k] = v
					}
				} else {
					for _, col := range t.Table.Columns {
						key := strings.ToLower(col.Name)
						if _, ok := t.Defaults[key]; !ok {
							if _, generated := t.Generated[key]; !generated {
								after[key] = nil
							}
						}
					}
					for k, sql := range t.Defaults {
						if v, ok := constantValue(sql); ok {
							after[k] = v
						}
					}
				}
				for k, v := range op.Values {
					after[k] = v
				}
				touched := map[string]bool{}
				for _, c := range op.Columns {
					if !containsFold(primaryKeys(t.Table), c) {
						touched[strings.ToLower(c)] = true
					}
				}
				if op.Kind == "replace" {
					for _, col := range t.Table.Columns {
						touched[strings.ToLower(col.Name)] = true
					}
				}
				for _, i := range s.Indexes[strings.ToLower(op.Table)] {
					q = add(q, indexWriteCost(i, present, old, after, touched, strategy))
					if i.PredicateUnknown {
						r.Reasons = append(r.Reasons, "index_predicate_not_evaluated:"+i.Name)
					}
				}
				if potential := s.potentialManaged(op.Table); potential > 0 {
					extra := potential
					if present {
						extra *= 2
					}
					q = add(q, uncertain(0, &extra))
					r.Reasons = append(r.Reasons, "managed_fk_indexes_not_reconciled")
				}
				if branch == 0 {
					perRow = q
				} else {
					perRow = union(perRow, q)
				}
			}
			r.Count = multiply(perRow, op.Rows)
			r.Expression = "rows * (written_primary_and_value_cells + automatic_cells + index_effects)"
			if op.Rows.Upper == nil || !op.Rows.exact() {
				r.Reasons = append(r.Reasons, "affected_rows_not_known")
			}
			if !perRow.exact() {
				r.Reasons = append(r.Reasons, "row_existence_or_old_index_values")
			}
			if perRow.Upper != nil && *perRow.Upper > 0 && strategy == "bounds" {
				n := int64(80000) / *perRow.Upper
				r.MutationLimitedBatchRows = &n
			}
		}
		if len(t.Unknown) > 0 {
			r.Count = add(r.Count, uncertain(0, nil))
			r.Reasons = append(r.Reasons, t.Unknown...)
			r.MutationLimitedBatchRows = nil
		}
	default:
		return r, fmt.Errorf("strategy must be shape, bounds, or row-image")
	}
	classify(&r)
	slices.Sort(r.Reasons)
	r.Reasons = slices.Compact(r.Reasons)
	return r, nil
}

func classify(r *countResult) {
	r.Classification = "bounded"
	if r.Count.Upper == nil {
		r.Classification = "unknown_upper"
	}
	if r.Count.exact() {
		r.Classification = "exact_under_model"
	}
	if r.Count.Overflow {
		r.Reasons = append(r.Reasons, "count_arithmetic_overflow")
	}
	r.LimitVerdict = "indeterminate"
	if r.Strategy == "shape" {
		r.LimitVerdict = "comparison_only"
		return
	}
	if r.Count.Lower > 80000 {
		r.LimitVerdict = "model_exceeds_mutation_limit"
	} else if r.Count.Upper != nil && *r.Count.Upper <= 80000 {
		r.LimitVerdict = "within_model_mutation_limit_only"
	}
}

func baseWriteCost(t *tableModel, op writeOperation, strategy string, present bool, r *countResult) quantity {
	written := map[string]bool{}
	for _, c := range op.Columns {
		written[strings.ToLower(c)] = true
	}
	for _, k := range primaryKeys(t.Table) {
		written[strings.ToLower(k)] = true
	}
	base := int64(len(written))
	if present && (op.Kind == "update" || op.Kind == "upsert") {
		// A primary-key value identifies the row; it does not change its key.
		// Arbitrary generated-expression dependency/evaluation remains explicit.
		for _, expr := range t.Generated {
			if generatedDependsOnWritten(expr, written, primaryKeys(t.Table)) {
				r.Reasons = append(r.Reasons, "generated_update_dependency_not_evaluated")
				return uncertain(base, nil)
			}
		}
		return fixed(base)
	}
	auto := int64(len(t.Generated))
	for k := range t.Defaults {
		if !written[k] {
			auto++
		}
	}
	if auto == 0 {
		return fixed(base)
	}
	if strategy == "row-image" {
		if op.Channel == "dml" {
			return fixed(base + auto)
		}
		// Local calibration: stored generated-column insertion rewrites the
		// automatic columns, including declared defaults, in a second PK write.
		if len(t.Generated) > 0 {
			auto = int64(len(t.Generated) + len(t.Defaults))
		}
		return fixed(base + auto + int64(len(t.Table.PrimaryKey)))
	}
	r.Reasons = append(r.Reasons, "automatic_column_policy_requires_calibration")
	hi := base + int64(len(t.Generated)+len(t.Defaults)+len(t.Table.PrimaryKey))
	return uncertain(base, &hi)
}

func indexWriteCost(i indexModel, present bool, old, after map[string]interface{}, touched map[string]bool, strategy string) quantity {
	insert := int64(1 + len(i.Stored))
	newPresence := membership(i, after)
	if !present {
		return multiply(newPresence, fixed(insert))
	}
	keyTouched := i.PredicateUnknown
	storedTouched := int64(0)
	for _, k := range i.Keys {
		keyTouched = keyTouched || touched[strings.ToLower(k)]
	}
	for _, k := range i.Stored {
		if touched[strings.ToLower(k)] {
			storedTouched++
		}
	}
	if !keyTouched && storedTouched == 0 {
		return fixed(0)
	}
	if strategy != "row-image" {
		if !keyTouched {
			if !i.NullFiltered && !i.PredicateUnknown {
				return fixed(1 + storedTouched)
			}
			hi := 1 + storedTouched
			return uncertain(0, &hi)
		}
		hi := int64(1) + insert
		return uncertain(0, &hi)
	}
	oldPresence := membership(i, old)
	keySame := !keyTouched
	keyKnown := !keyTouched
	if keyTouched {
		keySame = true
		keyKnown = true
		for _, k := range i.Keys {
			a, oa := old[strings.ToLower(k)]
			b, ob := after[strings.ToLower(k)]
			if !oa || !ob {
				keyKnown = false
				continue
			}
			if !reflect.DeepEqual(a, b) {
				keySame = false
				keyKnown = true
				break
			}
		}
	}
	var outcomes []int64
	for before := oldPresence.Lower; before <= valueOr(oldPresence.Upper, 1); before++ {
		for next := newPresence.Lower; next <= valueOr(newPresence.Upper, 1); next++ {
			switch {
			case before == 0 && next == 0:
				outcomes = append(outcomes, 0)
			case before == 0:
				outcomes = append(outcomes, insert)
			case next == 0:
				outcomes = append(outcomes, 1)
			default:
				if !keyKnown || keySame {
					v := int64(0)
					if storedTouched > 0 {
						v = 1 + storedTouched
					}
					outcomes = append(outcomes, v)
				}
				if !keyKnown || !keySame {
					outcomes = append(outcomes, 1+insert)
				}
			}
		}
	}
	lo, hi := outcomes[0], outcomes[0]
	for _, v := range outcomes {
		lo = min(lo, v)
		hi = max(hi, v)
	}
	return uncertain(lo, &hi)
}
func membership(i indexModel, values map[string]interface{}) quantity {
	if i.PredicateUnknown {
		hi := int64(1)
		return uncertain(0, &hi)
	}
	if !i.NullFiltered {
		return fixed(1)
	}
	known := true
	for _, k := range i.Keys {
		v, ok := values[strings.ToLower(k)]
		if ok && v == nil {
			return fixed(0)
		}
		known = known && ok
	}
	if known {
		return fixed(1)
	}
	hi := int64(1)
	return uncertain(0, &hi)
}
func lowerMap(in map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range in {
		out[strings.ToLower(k)] = v
	}
	return out
}
func containsFold(list []string, name string) bool {
	for _, s := range list {
		if strings.EqualFold(s, name) {
			return true
		}
	}
	return false
}
func valueOr(p *int64, otherwise int64) int64 {
	if p == nil {
		return otherwise
	}
	return *p
}
