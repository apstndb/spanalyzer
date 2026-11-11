package spanalyzer

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
)

// MutationCountInterval is an inclusive, conditional interval of mutation units.
// A nil Upper means unknown, and Overflow prevents an exact classification.
type MutationCountInterval struct {
	Lower    int64  `json:"lower" yaml:"lower"`
	Upper    *int64 `json:"upper" yaml:"upper"`
	Overflow bool   `json:"overflow,omitempty" yaml:"overflow,omitempty"`
}

func mutationFixed(n int64) MutationCountInterval { return MutationCountInterval{Lower: n, Upper: &n} }
func mutationUncertain(lo int64, hi *int64) MutationCountInterval {
	return MutationCountInterval{Lower: lo, Upper: hi}
}
func (q MutationCountInterval) exact() bool {
	return !q.Overflow && q.Upper != nil && q.Lower == *q.Upper
}
func mutationAdd(a, b MutationCountInterval) MutationCountInterval {
	q := MutationCountInterval{Overflow: a.Overflow || b.Overflow}
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
func mutationMultiply(a, b MutationCountInterval) MutationCountInterval {
	if a.Upper != nil && *a.Upper == 0 || b.Upper != nil && *b.Upper == 0 {
		return mutationFixed(0)
	}
	q := MutationCountInterval{Overflow: a.Overflow || b.Overflow}
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
func mutationUnion(a, b MutationCountInterval) MutationCountInterval {
	q := MutationCountInterval{Lower: min(a.Lower, b.Lower), Overflow: a.Overflow || b.Overflow}
	if a.Upper != nil && b.Upper != nil {
		v := max(*a.Upper, *b.Upper)
		q.Upper = &v
	}
	return q
}

// MutationCountRowFacts contains caller assertions, not automatically read data.
// Old uses wire scalar encodings. IndexRows/CascadeIndexRows use physical index
// names; generated FK suffixes must come from an observed snapshot.
// IndexRows cannot override known cardinality or index membership. Unconditional
// indexes on a deleted table must agree on the number of matched rows.
type MutationCountRowFacts struct {
	Exists           *bool                  `json:"exists,omitempty" yaml:"exists,omitempty"`
	Old              map[string]interface{} `json:"old,omitempty" yaml:"old,omitempty"`
	Rows             *int64                 `json:"rows,omitempty" yaml:"rows,omitempty"`
	IndexRows        map[string]int64       `json:"index_rows,omitempty" yaml:"index_rows,omitempty"`
	CascadeIndexRows map[string]int64       `json:"cascade_index_rows,omitempty" yaml:"cascade_index_rows,omitempty"`
}
type mutationCountOperation struct {
	InsertSelect  bool
	Calibration   string
	WireOperation string
	Shape         bool
	DeleteKey     bool
	Statement     int
	Kind          string
	Table         string
	Columns       []string
	Values        map[string]interface{}
	Rows          MutationCountInterval
	DeleteUnits   MutationCountInterval
	Facts         MutationCountRowFacts
	Channel       string
	Assumptions   []string
}

// MutationCountEstimate explains one normalized operation. Its independent
// contribution need not sum to the batch total when same-key writes coalesce.
type MutationCountEstimate struct {
	IndexEffects             []MutationCountIndexEffect `json:"index_effects,omitempty" yaml:"index_effects,omitempty"`
	Strategy                 string                     `json:"strategy" yaml:"strategy"`
	Table                    string                     `json:"table" yaml:"table"`
	Operation                string                     `json:"operation" yaml:"operation"`
	Channel                  string                     `json:"channel" yaml:"channel"`
	Count                    MutationCountInterval      `json:"count" yaml:"count"`
	Expression               string                     `json:"expression" yaml:"expression"`
	Classification           string                     `json:"classification" yaml:"classification"`
	Reasons                  []string                   `json:"uncertainties,omitempty" yaml:"uncertainties,omitempty"`
	Assumptions              []string                   `json:"assumptions,omitempty" yaml:"assumptions,omitempty"`
	IndexSource              string                     `json:"index_source" yaml:"index_source"`
	LimitScope               string                     `json:"limit_scope" yaml:"limit_scope"`
	LimitVerdict             string                     `json:"limit_verdict" yaml:"limit_verdict"`
	MutationLimitedBatchRows *int64                     `json:"mutation_limited_batch_rows,omitempty" yaml:"mutation_limited_batch_rows,omitempty"`
	GateEligible             bool                       `json:"gate_eligible" yaml:"gate_eligible"`
	ByteLimitChecked         bool                       `json:"commit_size_limit_checked" yaml:"commit_size_limit_checked"`
}

// MutationCountIndexEffect isolates index maintenance from base-row cells.
// Zero is a known unaffected index; a nil upper bound is an unknown effect.
type MutationCountIndexEffect struct {
	Index string                `json:"index" yaml:"index"`
	Table string                `json:"table" yaml:"table"`
	Count MutationCountInterval `json:"count" yaml:"count"`
}

func estimateMutationCount(s *mutationCountSchemaModel, op mutationCountOperation, strategy string) (MutationCountEstimate, error) {
	t := s.Tables[strings.ToLower(op.Table)]
	if t == nil {
		return MutationCountEstimate{}, fmt.Errorf("unknown writable table %q", op.Table)
	}
	// Managed queue controls reject ordinary wire writes (Table not found) and
	// DML UPDATE/UPSERT. Keep these separate from successful queue operations
	// whose internal accounting has an unknown upper bound. See the retained
	// managed mutation-count note for the operation-specific negative controls.
	if t.Queue {
		if op.Channel == "mutation" && op.WireOperation != "send" && op.WireOperation != "ack" {
			return MutationCountEstimate{}, fmt.Errorf("queue %s requires Send/Ack mutations, not ordinary table mutations", op.Table)
		}
		if op.Channel == "dml" && op.Kind != "insert" && op.Kind != "delete" {
			return MutationCountEstimate{}, fmt.Errorf("queue %s supports only INSERT/DELETE DML in this preview", op.Table)
		}
	}
	if op.Facts.Rows != nil {
		if *op.Facts.Rows < 0 {
			return MutationCountEstimate{}, fmt.Errorf("negative row count")
		}
		op.Rows = mutationFixed(*op.Facts.Rows)
	}
	if strategy == "row-image" && op.WireOperation == "insert_or_ignore" && !op.InsertSelect && op.Facts.Exists != nil {
		rows := int64(1)
		if *op.Facts.Exists {
			rows = 0
		}
		if op.Facts.Rows != nil && *op.Facts.Rows != rows {
			return MutationCountEstimate{}, fmt.Errorf("existence contradicts INSERT OR IGNORE affected-row fact")
		}
		op.Rows = mutationFixed(rows)
	}
	if strategy == "row-image" && op.DeleteKey && op.Facts.Exists != nil {
		n := int64(0)
		if *op.Facts.Exists {
			n = 1
		}
		op.Rows = mutationFixed(n)
	}
	if strategy == "row-image" && op.Kind == "update" && op.Channel == "dml" && op.Facts.Exists != nil && !*op.Facts.Exists {
		op.Rows = mutationFixed(0)
	}
	if op.Rows.Lower < 0 || op.Rows.Upper != nil && *op.Rows.Upper < op.Rows.Lower {
		return MutationCountEstimate{}, fmt.Errorf("invalid row bounds")
	}
	r := MutationCountEstimate{Strategy: strategy, Table: t.Table.Name.String(), Operation: op.Kind, Channel: op.Channel, IndexSource: s.IndexSource, Assumptions: append([]string{"successful operation; schema and snapshot match the transaction; mutation quota only; experimental model, not a service guarantee"}, op.Assumptions...), LimitScope: "mutation_api_commit"}
	if op.WireOperation != "" {
		r.Operation = op.WireOperation
	}
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
		r.Count = mutationMultiply(mutationFixed(cost), op.Rows)
		r.Expression = fmt.Sprintf("rows * %d (shape baseline)", cost)
	case "bounds", "row-image":
		if strategy == "row-image" {
			r.Assumptions = append(r.Assumptions, "supplied old-row assertions; experimental index-delta model, not a database read")
		}
		if op.Kind == "delete" || op.Kind == "delete_range" {
			if op.Channel == "dml" {
				op.DeleteUnits = mutationUncertain(0, op.Rows.Upper)
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
					n = mutationFixed(v)
				} else if i.NullFiltered || i.PredicateUnknown {
					n = mutationUncertain(0, op.Rows.Upper)
					if strategy == "row-image" && op.DeleteKey && op.Rows.exact() {
						old := mutationLowerMap(op.Facts.Old)
						for key, v := range op.Values {
							old[key] = v
						}
						n = mutationMultiply(op.Rows, mutationIndexMembership(i, old))
					}
					r.Reasons = append(r.Reasons, "deleted_index_membership:"+i.Name)
				}
				r.Count = mutationAdd(r.Count, n)
				r.IndexEffects = append(r.IndexEffects, MutationCountIndexEffect{Index: i.Name, Table: i.Table, Count: n})
			}
			if potential := s.potentialManaged(op.Table); potential > 0 {
				r.Count = mutationAdd(r.Count, mutationMultiply(mutationUncertain(0, &potential), op.Rows))
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
						r.Count = mutationAdd(r.Count, mutationFixed(n))
					} else {
						r.Count = mutationAdd(r.Count, mutationUncertain(0, nil))
						r.Reasons = append(r.Reasons, "interleaved_child_index_entries:"+i.Name)
					}
				}
				if s.potentialManaged(child.Table.Name.String()) > 0 || len(child.Unknown) > 0 {
					r.Count = mutationAdd(r.Count, mutationUncertain(0, nil))
					r.Reasons = append(r.Reasons, "cascade_child_effects_not_reconciled")
				}
			}
			for _, f := range s.ForeignKeys {
				if deletedTables[strings.ToLower(f.Referenced)] && (strings.Contains(f.DeleteAction, "CASCADE") || strings.Contains(f.DeleteAction, "SET NULL")) {
					r.Count = mutationAdd(r.Count, mutationUncertain(0, nil))
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
				if op.Kind == "insert" && op.WireOperation != "insert_or_ignore" && *op.Facts.Exists || op.Kind == "update" && op.Channel != "dml" && !*op.Facts.Exists {
					return r, fmt.Errorf("existence fact contradicts successful %s", op.Kind)
				}
				branches = []bool{*op.Facts.Exists}
			}
			var perRow MutationCountInterval
			perIndex := map[string]MutationCountInterval{}
			for branch, present := range branches {
				touched := mutationTouchedColumns(t, op, present)
				q := mutationBaseWriteCost(t, op, strategy, present, touched, &r)
				old := map[string]interface{}{}
				if strategy == "row-image" {
					old = mutationLowerMap(op.Facts.Old)
				}
				for _, pk := range mutationPrimaryKeys(t.Table) {
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
							if col.GeneratedSQL == "" && !mutationContainsFold(op.Columns, col.Name) {
								after[key] = nil
							}
						}
					}
					for k, sql := range t.Defaults {
						if v, ok := mutationConstantValue(sql); ok && !mutationContainsFold(op.Columns, k) {
							after[k] = v
						}
					}
				}
				for k, v := range op.Values {
					after[k] = v
				}
				for _, c := range op.Columns {
					key := strings.ToLower(c)
					if _, known := op.Values[key]; !known {
						// SET expressions and configured shapes need not supply a
						// concrete new value. Never inherit the old written value.
						delete(after, key)
					}
				}
				for key := range t.Expressions {
					if touched[key] {
						delete(after, key)
					}
				}
				if present && (op.Kind == "update" || op.Kind == "upsert") {
					for key := range t.OnUpdate {
						if touched[key] && !mutationContainsFold(op.Columns, key) {
							delete(after, key)
						}
					}
				}
				for _, i := range s.Indexes[strings.ToLower(op.Table)] {
					effect := mutationIndexWriteCost(i, present, old, after, touched, strategy)
					if present && (op.Kind == "update" || op.Kind == "upsert") {
						for _, col := range i.Stored {
							key := strings.ToLower(col)
							if touched[key] && !mutationContainsFold(op.Columns, key) && (t.Expressions[key] != "" || t.OnUpdate[key]) {
								// Automatic STORING maintenance is not calibrated to
								// the explicit same-value stored-cell write rule.
								effect.Lower = 0
							}
						}
					}
					q = mutationAdd(q, effect)
					if branch == 0 {
						perIndex[i.Name] = effect
					} else {
						perIndex[i.Name] = mutationUnion(perIndex[i.Name], effect)
					}
					if i.PredicateUnknown {
						r.Reasons = append(r.Reasons, "index_predicate_not_evaluated:"+i.Name)
					}
				}
				if potential := s.potentialManaged(op.Table); potential > 0 {
					extra := potential
					if present {
						extra *= 2
					}
					q = mutationAdd(q, mutationUncertain(0, &extra))
					r.Reasons = append(r.Reasons, "managed_fk_indexes_not_reconciled")
				}
				if branch == 0 {
					perRow = q
				} else {
					perRow = mutationUnion(perRow, q)
				}
			}
			r.Count = mutationMultiply(perRow, op.Rows)
			for _, i := range s.Indexes[strings.ToLower(op.Table)] {
				r.IndexEffects = append(r.IndexEffects, MutationCountIndexEffect{Index: i.Name, Table: i.Table, Count: mutationMultiply(perIndex[i.Name], op.Rows)})
			}
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
			if op.Kind == "replace" && len(s.cascadeDescendants(op.Table)) > 0 {
				r.Count = mutationAdd(r.Count, mutationUncertain(0, nil))
				r.Reasons = append(r.Reasons, "replace_interleaved_cascade_accounting")
				r.MutationLimitedBatchRows = nil
			}
		}
		if len(t.Unknown) > 0 {
			r.Count = mutationAdd(r.Count, mutationUncertain(0, nil))
			r.Reasons = append(r.Reasons, t.Unknown...)
			r.MutationLimitedBatchRows = nil
		}
		if t.Queue {
			r.Count = mutationUncertain(0, nil)
		}
	default:
		return r, fmt.Errorf("strategy must be shape, bounds, or row-image")
	}
	if op.Channel == "dml" && op.Rows.exact() && op.Rows.Lower == 0 {
		// Managed empty/ignored queue DML commits expose a real zero. This is
		// conditional on affected-row facts; it does not price a missing Ack
		// or internal RECEIVE/RENEWLEASE writes from their result row count.
		r.Count = mutationFixed(0)
		r.Expression = "zero affected DML rows"
		r.MutationLimitedBatchRows = nil
		r.Reasons = append(r.Reasons, "zero_affected_dml_rows")
	}
	classifyMutationCount(&r)
	slices.Sort(r.Reasons)
	r.Reasons = slices.Compact(r.Reasons)
	return r, nil
}

func classifyMutationCount(r *MutationCountEstimate) {
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

// mutationTouchedColumns follows dependency edges without evaluating SQL. A
// virtual column has no base cell, but can be a key or stored index column. See
// knowledge/research/write-analysis/mutation-count-reference-audit-20261110.md
// for generated/ON UPDATE semantics and the source-backed validation boundary.
func mutationTouchedColumns(t *mutationCountTableModel, op mutationCountOperation, present bool) map[string]bool {
	touched := map[string]bool{}
	for _, c := range op.Columns {
		if !mutationContainsFold(mutationPrimaryKeys(t.Table), c) {
			touched[strings.ToLower(c)] = true
		}
	}
	if present && (op.Kind == "update" || op.Kind == "upsert") {
		if len(touched) > 0 {
			for key := range t.OnUpdate {
				if !mutationContainsFold(op.Columns, key) {
					touched[key] = true
				}
			}
		}
		for changed := true; changed; {
			changed = false
			for key, expr := range t.Expressions {
				if !touched[key] && mutationGeneratedDependency(expr, touched, nil) {
					touched[key], changed = true, true
				}
			}
		}
	} else {
		for key := range t.Expressions {
			touched[key] = true
		}
	}
	if op.Kind == "replace" {
		for _, col := range t.Table.Columns {
			touched[strings.ToLower(col.Name)] = true
		}
	}
	return touched
}

func mutationBaseWriteCost(t *mutationCountTableModel, op mutationCountOperation, strategy string, present bool, touched map[string]bool, r *MutationCountEstimate) MutationCountInterval {
	written := map[string]bool{}
	for _, c := range op.Columns {
		written[strings.ToLower(c)] = true
	}
	for _, k := range mutationPrimaryKeys(t.Table) {
		written[strings.ToLower(k)] = true
	}
	base := int64(len(written))
	if present && (op.Kind == "update" || op.Kind == "upsert") {
		// A primary-key value identifies the row; it does not change its key.
		// Arbitrary generated-expression dependency/evaluation remains explicit.
		for key := range t.Generated {
			if touched[key] {
				r.Reasons = append(r.Reasons, "generated_update_dependency_not_evaluated")
				return mutationUncertain(base, nil)
			}
		}
		for key := range t.OnUpdate {
			if touched[key] && !written[key] {
				r.Reasons = append(r.Reasons, "on_update_expression")
				return mutationUncertain(base, nil)
			}
		}
		return mutationFixed(base)
	}
	auto := int64(len(t.Generated))
	for k := range t.Defaults {
		if !written[k] {
			auto++
		}
	}
	if auto == 0 {
		return mutationFixed(base)
	}
	if strategy == "row-image" && op.Calibration != "" {
		if op.Channel == "dml" {
			return mutationFixed(base + auto)
		}
		// Local calibration: stored generated-column insertion rewrites the
		// automatic columns, including declared defaults, in a second PK write.
		if len(t.Generated) > 0 {
			auto = int64(len(t.Generated) + len(t.Defaults))
		}
		return mutationFixed(base + auto + int64(len(t.Table.PrimaryKey)))
	}
	r.Reasons = append(r.Reasons, "automatic_column_policy_requires_calibration")
	hi := base + int64(len(t.Generated)+len(t.Defaults)+len(t.Table.PrimaryKey))
	return mutationUncertain(base, &hi)
}

func mutationIndexWriteCost(i MutationCountIndex, present bool, old, after map[string]interface{}, touched map[string]bool, strategy string) MutationCountInterval {
	insert := int64(1 + len(i.Stored))
	newPresence := mutationIndexMembership(i, after)
	if !present {
		return mutationMultiply(newPresence, mutationFixed(insert))
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
		return mutationFixed(0)
	}
	if strategy != "row-image" {
		if !keyTouched {
			if !i.NullFiltered && !i.PredicateUnknown {
				return mutationFixed(1 + storedTouched)
			}
			hi := 1 + storedTouched
			return mutationUncertain(0, &hi)
		}
		hi := int64(1) + insert
		return mutationUncertain(0, &hi)
	}
	oldPresence := mutationIndexMembership(i, old)
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
	for before := oldPresence.Lower; before <= mutationValueOr(oldPresence.Upper, 1); before++ {
		for next := newPresence.Lower; next <= mutationValueOr(newPresence.Upper, 1); next++ {
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
	return mutationUncertain(lo, &hi)
}
func mutationIndexMembership(i MutationCountIndex, values map[string]interface{}) MutationCountInterval {
	if i.PredicateUnknown {
		hi := int64(1)
		return mutationUncertain(0, &hi)
	}
	if !i.NullFiltered {
		return mutationFixed(1)
	}
	known := true
	for _, k := range i.Keys {
		v, ok := values[strings.ToLower(k)]
		if ok && v == nil {
			return mutationFixed(0)
		}
		known = known && ok
	}
	if known {
		return mutationFixed(1)
	}
	hi := int64(1)
	return mutationUncertain(0, &hi)
}
func mutationLowerMap(in map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range in {
		out[strings.ToLower(k)] = v
	}
	return out
}
func mutationContainsFold(list []string, name string) bool {
	for _, s := range list {
		if strings.EqualFold(s, name) {
			return true
		}
	}
	return false
}
func mutationValueOr(p *int64, otherwise int64) int64 {
	if p == nil {
		return otherwise
	}
	return *p
}
