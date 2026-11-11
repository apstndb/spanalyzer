package spanalyzer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
)

// MutationCountAnalyzer provides preview, non-executing write accounting.
// The declared schema determines the baseline interval. Caller assertions and
// measurements are retained as separate refinements, never as universal prices.
// An analyzer performs no database reads or writes and may be reused.
type MutationCountAnalyzer struct{ schema *mutationCountSchemaModel }

// MutationCountOptions supplies conditional facts for one analyzed input.
// Facts are parallel to normalized row operations, not original wire messages.
// Repeated updates to one key use the first old image as the pre-commit image.
// INSERT SELECT normalizes to one group accepting cardinality facts, not a
// singular Exists/Old image. Its source query is never analyzed or executed.
type MutationCountOptions struct {
	Facts       []MutationCountRowFacts
	Observation *MutationCountObservation
	// Calibration opts into automatic-column rules observed on the named
	// runtime fixture. Empty preserves uncertainty even when old rows are known.
	Calibration string
}

// MutationCountObservation is a measurement of this exact input and schema.
// An absent statistic is not zero. A single observed execution never supplies
// an upper bound for future executions or a proof of quota enforcement.
type MutationCountObservation struct {
	Runtime      string `json:"runtime" yaml:"runtime"`
	ObservedAt   string `json:"observed_at" yaml:"observed_at"`
	SchemaSHA256 string `json:"schema_sha256" yaml:"schema_sha256"`
	InputSHA256  string `json:"input_sha256" yaml:"input_sha256"`
	Scope        string `json:"scope" yaml:"scope"`
	Successful   bool   `json:"successful" yaml:"successful"`
	StatsPresent bool   `json:"stats_present" yaml:"stats_present"`
	Count        *int64 `json:"mutation_count" yaml:"mutation_count"`
}

// MutationCountReport is the mutable v0alpha1 write-accounting projection.
// Bounds always remains available, including when an observation disagrees.
// Effective is conditional on the separately identified facts/measurement.
type MutationCountReport struct {
	Version                  string                    `json:"version" yaml:"version"`
	SchemaSHA256             string                    `json:"schema_sha256" yaml:"schema_sha256"`
	InputSHA256              string                    `json:"input_sha256" yaml:"input_sha256"`
	Scope                    string                    `json:"scope" yaml:"scope"`
	Bounds                   MutationCountAnalysis     `json:"bounds" yaml:"bounds"`
	OldRows                  *MutationCountAnalysis    `json:"old_rows,omitempty" yaml:"old_rows,omitempty"`
	Observation              *MutationCountObservation `json:"observation,omitempty" yaml:"observation,omitempty"`
	Effective                MutationCountInterval     `json:"effective" yaml:"effective"`
	EffectiveBasis           string                    `json:"effective_basis" yaml:"effective_basis"`
	Disagreements            []string                  `json:"disagreements,omitempty" yaml:"disagreements,omitempty"`
	Indexes                  []MutationCountIndex      `json:"indexes,omitempty" yaml:"indexes,omitempty"`
	CommitSizeLimitChecked   bool                      `json:"commit_size_limit_checked" yaml:"commit_size_limit_checked"`
	QuotaEnforcementVerified bool                      `json:"quota_enforcement_verified" yaml:"quota_enforcement_verified"`
}

// MutationWriteShape describes a homogeneous, independent write scenario.
// Omitted values remain unknown; they are not NULL. A nil Rows means one row
// for insert/update/upsert/replace and zero or one matched row for a delete key.
// Values uses wire scalar encodings, including decimal strings for INT64.
// It does not validate application values or prove that the write succeeds.
type MutationWriteShape struct {
	Operation string                 `json:"operation" yaml:"operation"`
	Table     string                 `json:"table" yaml:"table"`
	Columns   []string               `json:"columns" yaml:"columns"`
	Values    map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
	Rows      *int64                 `json:"rows,omitempty" yaml:"rows,omitempty"`
	RowBounds *MutationCountInterval `json:"row_bounds,omitempty" yaml:"row_bounds,omitempty"`
	Channel   string                 `json:"channel,omitempty" yaml:"channel,omitempty"`
}

// NewMutationCountAnalyzer constructs a preview write-impact catalog from DDL.
// It supplements the ordinary row-type catalog with write-only schema facts.
func NewMutationCountAnalyzer(ddl string) (*MutationCountAnalyzer, error) {
	s, err := newMutationCountSchema(ddl)
	if err != nil {
		return nil, err
	}
	return &MutationCountAnalyzer{schema: s}, nil
}

// SchemaSHA256 identifies the exact DDL bytes, without normalizing histories.
func (a *MutationCountAnalyzer) SchemaSHA256() string { return a.schema.SourceSHA256 }

// WithObservedIndexes returns a new analyzer reconciled with a complete index
// snapshot of this DDL. The caller must ensure snapshot completeness/freshness.
// Physical indexes are deduplicated; constraint aliases remain inferred.
func (a *MutationCountAnalyzer) WithObservedIndexes(ddlSHA256 string, indexes []MutationCountIndexObservation) (*MutationCountAnalyzer, error) {
	if ddlSHA256 != a.schema.SourceSHA256 {
		return nil, fmt.Errorf("index snapshot schema digest mismatch")
	}
	s, err := a.schema.withObservedIndexes(indexes)
	if err != nil {
		return nil, err
	}
	return &MutationCountAnalyzer{schema: s}, nil
}

// AnalyzeMutationsJSON validates a JSON array of google.spanner.v1.Mutation
// messages and estimates one Mutation API commit. It supports the wire types
// documented by the input validator; unsupported encodings fail explicitly.
func (a *MutationCountAnalyzer) AnalyzeMutationsJSON(data []byte, options MutationCountOptions) (MutationCountReport, error) {
	ops, err := decodeMutationCountInput(a.schema, data)
	if err != nil {
		return MutationCountReport{}, err
	}
	var canonical interface{}
	if err := json.Unmarshal(data, &canonical); err != nil {
		return MutationCountReport{}, err
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return MutationCountReport{}, err
	}
	return a.analyze(ops, digestMutationInput("mutation", encoded), "mutation_api_commit", options)
}

// AnalyzeDML estimates ordinary GoogleSQL DML statements in one transaction.
// Statement limits are checked separately from cumulative CommitStats. UPDATE
// and DELETE cardinality stays unknown unless supplied in Facts. No SELECT is
// executed and SQL expression semantics remain a service/analyzer check.
func (a *MutationCountAnalyzer) AnalyzeDML(statements []string, options MutationCountOptions) (MutationCountReport, error) {
	if len(statements) == 0 {
		return MutationCountReport{}, fmt.Errorf("DML statements are required")
	}
	var ops []mutationCountOperation
	for i, sql := range statements {
		part, err := mutationDMLOperations(a.schema, sql)
		if err != nil {
			return MutationCountReport{}, fmt.Errorf("statement %d: %w", i, err)
		}
		for n := range part {
			part[n].Statement = i
		}
		ops = append(ops, part...)
	}
	b, err := json.Marshal(statements)
	if err != nil {
		return MutationCountReport{}, err
	}
	return a.analyze(ops, digestMutationInput("dml", b), "dml_transaction", options)
}

// AnalyzeWriteShape estimates a configured write without runtime values. This
// is suitable for explain-plan; value validation belongs to AnalyzeMutationsJSON.
func (a *MutationCountAnalyzer) AnalyzeWriteShape(shape MutationWriteShape, options MutationCountOptions) (MutationCountReport, error) {
	op := mutationCountOperation{Kind: strings.ToLower(strings.TrimSpace(shape.Operation)), Table: shape.Table, Columns: slices.Clone(shape.Columns), Values: mutationLowerMap(shape.Values), Rows: mutationFixed(1), Channel: shape.Channel, Shape: true}
	if op.Kind == "insert_or_update" {
		op.Kind = "upsert"
	}
	switch op.Kind {
	case "insert", "update", "upsert", "replace", "delete":
	default:
		return MutationCountReport{}, fmt.Errorf("unsupported write shape operation %q", shape.Operation)
	}
	if op.Channel == "" {
		op.Channel = "mutation"
	}
	if op.Channel != "mutation" && op.Channel != "dml" {
		return MutationCountReport{}, fmt.Errorf("channel must be mutation or dml")
	}
	t := a.schema.Tables[strings.ToLower(shape.Table)]
	if t == nil {
		return MutationCountReport{}, fmt.Errorf("unknown writable table %q", shape.Table)
	}
	seen := map[string]bool{}
	for _, name := range shape.Columns {
		key := strings.ToLower(name)
		if seen[key] {
			return MutationCountReport{}, fmt.Errorf("duplicate write column %q", name)
		}
		seen[key] = true
		c, _ := t.Table.Column(name)
		if c == nil {
			return MutationCountReport{}, fmt.Errorf("unknown write column %q", name)
		}
		if t.Generated[key] != "" || t.Virtual[key] {
			return MutationCountReport{}, fmt.Errorf("generated column %q is not writable", name)
		}
	}
	for name, value := range shape.Values {
		key := strings.ToLower(name)
		if !seen[key] {
			return MutationCountReport{}, fmt.Errorf("shape value %q has no written column", name)
		}
		c, _ := t.Table.Column(name)
		p, err := structpb.NewValue(value)
		if err != nil {
			return MutationCountReport{}, err
		}
		v, err := mutationProtoValue(c.Type, p, c.NotNull, c.AllowCommitTimestamp)
		if err != nil {
			return MutationCountReport{}, fmt.Errorf("shape value %s: %w", name, err)
		}
		op.Values[key] = v
	}
	if op.Kind == "delete" {
		one := int64(1)
		op.Rows = mutationUncertain(0, &one)
		op.DeleteUnits = mutationFixed(1)
		op.DeleteKey = true
	}
	if shape.Rows != nil {
		if *shape.Rows < 0 || op.Kind == "delete" && *shape.Rows > 1 {
			return MutationCountReport{}, fmt.Errorf("invalid row count for write shape; delete shape is one full key")
		}
		op.Rows = mutationFixed(*shape.Rows)
	}
	if shape.RowBounds != nil {
		if shape.Rows != nil || shape.RowBounds.Lower < 0 || shape.RowBounds.Upper != nil && *shape.RowBounds.Upper < shape.RowBounds.Lower {
			return MutationCountReport{}, fmt.Errorf("invalid or conflicting shape row bounds")
		}
		if op.Kind == "delete" && (shape.RowBounds.Upper == nil || *shape.RowBounds.Upper > 1) {
			return MutationCountReport{}, fmt.Errorf("delete shape is one full key; row bounds cannot exceed one")
		}
		op.Rows = *shape.RowBounds
	}
	b, err := json.Marshal(shape)
	if err != nil {
		return MutationCountReport{}, err
	}
	scope := "mutation_api_commit"
	if op.Channel == "dml" {
		scope = "dml_transaction"
	}
	return a.analyze([]mutationCountOperation{op}, digestMutationInput("shape", b), scope, options)
}

func (a *MutationCountAnalyzer) analyze(ops []mutationCountOperation, inputSHA, scope string, options MutationCountOptions) (MutationCountReport, error) {
	r := MutationCountReport{Version: "spanalyzer.mutation-count/v0alpha1", SchemaSHA256: a.schema.SourceSHA256, InputSHA256: inputSHA, Scope: scope, EffectiveBasis: "schema_and_supplied_cardinality"}
	if options.Calibration != "" && options.Calibration != "omni-2026.r4-lts.2" {
		return r, fmt.Errorf("unknown automatic-column calibration %q", options.Calibration)
	}
	if len(options.Facts) > 0 && len(options.Facts) != len(ops) {
		return r, fmt.Errorf("facts count %d differs from normalized operations %d", len(options.Facts), len(ops))
	}
	oldRows := false
	for n := range ops {
		if len(options.Facts) > 0 {
			f := options.Facts[n]
			if err := validateMutationRowFacts(a.schema, ops[n], &f); err != nil {
				return r, fmt.Errorf("facts %d: %w", n, err)
			}
			if !ops[n].Shape && ops[n].Channel == "mutation" && ops[n].Kind != "delete" && f.Rows != nil && *f.Rows != 1 {
				return r, fmt.Errorf("wire row cardinality is already known; row count fact must be one")
			}
			ops[n].Facts = f
			oldRows = oldRows || f.Exists != nil || len(f.Old) > 0
		}
		ops[n].Calibration = options.Calibration
	}
	var err error
	r.Bounds, err = analyzeMutationCount(a.schema, ops, "bounds")
	if err != nil {
		return r, err
	}
	r.Effective = r.Bounds.Count
	if oldRows {
		refined, e := analyzeMutationCount(a.schema, ops, "row-image")
		if e != nil {
			return r, e
		}
		r.OldRows = &refined
		if q, ok := intersectMutationCounts(r.Effective, refined.Count); ok {
			r.Effective = q
			r.EffectiveBasis = "schema_and_old_row_assertions"
		} else {
			r.Disagreements = append(r.Disagreements, "old-row model does not intersect the schema interval")
		}
	}
	if o := options.Observation; o != nil {
		copy := *o
		r.Observation = &copy
		if o.SchemaSHA256 != r.SchemaSHA256 || o.InputSHA256 != inputSHA || o.Scope != scope {
			return r, fmt.Errorf("observation must identify the same schema, input and accounting scope")
		}
		if o.Runtime == "" {
			return r, fmt.Errorf("observation runtime is required")
		}
		if _, err := time.Parse(time.RFC3339Nano, o.ObservedAt); err != nil {
			return r, fmt.Errorf("observation timestamp: %w", err)
		}
		if o.Successful && o.StatsPresent && o.Count != nil {
			if *o.Count < 0 {
				return r, fmt.Errorf("negative observed mutation count")
			}
			if q, ok := intersectMutationCounts(r.Effective, mutationFixed(*o.Count)); ok {
				r.Effective = q
				r.EffectiveBasis = "same_execution_observation"
			} else {
				r.Disagreements = append(r.Disagreements, "measured count lies outside the conditional model; observation retained without replacing the model")
			}
		}
	}
	var names []string
	for name := range a.schema.Indexes {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		for _, i := range a.schema.Indexes[name] {
			i.Keys = slices.Clone(i.Keys)
			i.Stored = slices.Clone(i.Stored)
			i.Constraints = slices.Clone(i.Constraints)
			if i.ObservedUnique != nil {
				v := *i.ObservedUnique
				i.ObservedUnique = &v
			}
			r.Indexes = append(r.Indexes, i)
		}
	}
	return r, nil
}

func digestMutationInput(kind string, data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(append([]byte(kind+"\n"), data...)))
}
func intersectMutationCounts(a, b MutationCountInterval) (MutationCountInterval, bool) {
	q := MutationCountInterval{Lower: max(a.Lower, b.Lower), Overflow: a.Overflow || b.Overflow}
	switch {
	case a.Upper == nil && b.Upper == nil:
	case a.Upper == nil:
		v := *b.Upper
		q.Upper = &v
	case b.Upper == nil:
		v := *a.Upper
		q.Upper = &v
	default:
		v := min(*a.Upper, *b.Upper)
		q.Upper = &v
	}
	return q, q.Upper == nil || q.Lower <= *q.Upper
}
