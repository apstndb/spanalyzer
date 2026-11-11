package spanalyzer

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
)

type managedMutationFixture struct {
	Status               string            `json:"status"`
	FatalError           string            `json:"fatal_error"`
	Started              string            `json:"started_at"`
	Finished             string            `json:"finished_at"`
	Lifecycle            map[string]string `json:"lifecycle"`
	CleanupErrors        []string          `json:"cleanup_errors"`
	Mode                 string            `json:"fixture_mode"`
	HypothesisMismatches []string          `json:"hypothesis_mismatches"`
	Phases               []struct {
		DDL     []string                        `json:"ddl"`
		Indexes []MutationCountIndexObservation `json:"indexes"`
	} `json:"schema_phases"`
	Observations []managedMutationCase `json:"observations"`
	QueueReads   []struct {
		ID, SQL, Status string
		Rows            int
		StatsScope      string
		Renewed         bool
	} `json:"queue_reads"`
	OtherWrites []struct {
		ID     string `json:"id"`
		API    string `json:"api"`
		Status string `json:"status"`
		Count  *int64 `json:"mutation_count"`
		Rows   *int64 `json:"affected_rows"`
		Groups []struct {
			Indexes   []int32 `json:"indexes"`
			Status    string  `json:"status"`
			Committed bool    `json:"committed"`
		} `json:"groups"`
	} `json:"other_writes"`
}
type managedMutationCase struct {
	ID             string            `json:"id"`
	Phase          int               `json:"schema_phase"`
	Kind           string            `json:"kind"`
	Table          string            `json:"table"`
	State          string            `json:"before_state"`
	Status         string            `json:"status"`
	Error          string            `json:"error"`
	StatsPresent   bool              `json:"stats_present"`
	Count          *int64            `json:"mutation_count"`
	Rows           []int64           `json:"dml_rows"`
	Mutations      []json.RawMessage `json:"mutations"`
	Statements     []string          `json:"statements"`
	Queries        []string          `json:"queries"`
	CommitObserved bool              `json:"commit_rpc_observed"`
}
type managedMutationSummary struct {
	Status           string                       `json:"fixture_status"`
	Records          int                          `json:"case_records"`
	NotRun           []string                     `json:"not_run"`
	Attempted        int                          `json:"attempted"`
	Measured         int                          `json:"successful_with_stats"`
	FiniteBaseline   int                          `json:"finite_baseline_upper"`
	BaselineContains int                          `json:"baseline_contains_observation"`
	OldExact         int                          `json:"old_row_exact_matches"`
	ObservedExact    int                          `json:"same_input_measured_refinements"`
	Failures         []string                     `json:"failed_cases"`
	Unsupported      []string                     `json:"analysis_errors"`
	Disagreements    []string                     `json:"disagreements"`
	Cases            []managedMutationCaseSummary `json:"cases"`
}
type managedMutationCaseSummary struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	Error     string                 `json:"error,omitempty"`
	Observed  *int64                 `json:"observed"`
	Baseline  *MutationCountInterval `json:"baseline,omitempty"`
	OldRows   *MutationCountInterval `json:"old_rows,omitempty"`
	Effective *MutationCountInterval `json:"effective,omitempty"`
}

var managedFixtureKeyEquality = regexp.MustCompile(`(?i)\b(Shard|Id|ParentId)\s*=\s*([0-9]+)\b`)

// Facts derive from the retained synthetic seed code, never CommitStats.
func managedMutationOptions(a *MutationCountAnalyzer, o managedMutationCase) (MutationCountOptions, error) {
	var ops []mutationCountOperation
	if len(o.Statements) > 0 {
		for i, sql := range o.Statements {
			part, err := mutationDMLOperations(a.schema, sql)
			if err != nil {
				return MutationCountOptions{}, err
			}
			for n := range part {
				part[n].Statement = i
			}
			ops = append(ops, part...)
		}
	} else {
		b, err := json.Marshal(o.Mutations)
		if err != nil {
			return MutationCountOptions{}, err
		}
		ops, err = decodeMutationCountInput(a.schema, b)
		if err != nil {
			return MutationCountOptions{}, err
		}
	}
	options := MutationCountOptions{}
	active := "a"
	for _, op := range ops {
		f := MutationCountRowFacts{}
		if op.Channel == "dml" && op.Kind != "insert" && op.Kind != "upsert" && op.Kind != "replace" {
			if op.Statement >= len(o.Rows) {
				return options, fmt.Errorf("missing affected-row count")
			}
			n := o.Rows[op.Statement]
			f.Rows = &n
		}
		queue := a.schema.Tables[strings.ToLower(op.Table)].Queue
		switch {
		case queue:
			f.Exists = mutationTestBool(op.Kind != "insert")
		case o.State == "present:a,n":
			f.Exists = mutationTestBool(true)
			f.Old = map[string]interface{}{"active": active, "note": "n"}
		case o.State == "present=true":
			f.Exists = mutationTestBool(true)
			f.Old = map[string]interface{}{"active": "old", "note": "old"}
		case strings.HasPrefix(o.State, "partial-present=true:"):
			f.Exists = mutationTestBool(true)
			f.Old = map[string]interface{}{"active": "a", "note": "n"}
		case o.State == "absent" || o.State == "present=false" || strings.HasPrefix(o.State, "partial-present=false:") || strings.Contains(o.State, "new rows") || o.State == "referenced owner exists":
			f.Exists = mutationTestBool(false)
			if op.Kind == "delete" {
				f.Rows = mutationTestInt(0)
			}
		case o.State == "active=NULL":
			f.Exists = mutationTestBool(true)
			f.Old = map[string]interface{}{"active": nil, "note": "n"}
		case o.State == "active=a":
			f.Exists = mutationTestBool(true)
			f.Old = map[string]interface{}{"active": "a", "note": "n"}
		case o.State == "3 existing indexable rows" || o.State == "1 existing indexable row":
			rows := int64(3)
			if o.State == "1 existing indexable row" {
				rows = 1
			}
			f.Rows = &rows
			f.IndexRows = map[string]int64{}
			for _, i := range a.schema.Indexes[strings.ToLower(op.Table)] {
				f.IndexRows[i.Name] = rows
			}
		case o.State == "parent +3 indexed children":
			f.Exists = mutationTestBool(true)
			f.Rows = mutationTestInt(1)
			f.CascadeIndexRows = map[string]int64{"ChildNote": 3}
		case o.State == "parent +1 queued child":
			f.Exists = mutationTestBool(true)
			f.Rows = mutationTestInt(1)
		default:
			return options, fmt.Errorf("unknown seed state %q", o.State)
		}
		if op.Channel == "dml" && len(f.Old) > 0 {
			for _, m := range managedFixtureKeyEquality.FindAllStringSubmatch(o.Statements[op.Statement], -1) {
				f.Old[strings.ToLower(m[1])] = m[2]
			}
			if v, ok := op.Values["active"].(string); ok {
				active = v
			}
		}
		options.Facts = append(options.Facts, f)
	}
	return options, nil
}

func readManagedMutationFixture(path string) (managedMutationFixture, error) {
	f, err := os.Open(path)
	if err != nil {
		return managedMutationFixture{}, err
	}
	defer func() { _ = f.Close() }()
	z, err := gzip.NewReader(f)
	if err != nil {
		return managedMutationFixture{}, err
	}
	defer func() { _ = z.Close() }()
	b, err := io.ReadAll(z)
	if err != nil {
		return managedMutationFixture{}, err
	}
	var fixture managedMutationFixture
	err = json.Unmarshal(b, &fixture)
	return fixture, err
}
func replayManagedMutations(f managedMutationFixture) managedMutationSummary {
	summary := managedMutationSummary{Status: f.Status}
	var models []*MutationCountAnalyzer
	for _, p := range f.Phases {
		a, err := NewMutationCountAnalyzer(strings.Join(p.DDL, ";\n") + ";\n")
		if err == nil {
			a, err = a.WithObservedIndexes(a.SchemaSHA256(), p.Indexes)
		}
		if err != nil {
			summary.Unsupported = append(summary.Unsupported, "phase: "+err.Error())
		}
		models = append(models, a)
	}
	for _, o := range f.Observations {
		summary.Records++
		item := managedMutationCaseSummary{ID: o.ID, Status: o.Status, Error: o.Error, Observed: o.Count}
		if o.Status == "NOT_RUN" {
			summary.NotRun = append(summary.NotRun, o.ID)
			summary.Cases = append(summary.Cases, item)
			continue
		}
		summary.Attempted++
		if o.Status != "OK" {
			summary.Failures = append(summary.Failures, o.ID+": "+o.Status)
			summary.Cases = append(summary.Cases, item)
			continue
		}
		if !o.StatsPresent || o.Count == nil {
			summary.Unsupported = append(summary.Unsupported, o.ID+": missing CommitStats")
			summary.Cases = append(summary.Cases, item)
			continue
		}
		summary.Measured++
		if o.Phase < 0 || o.Phase >= len(models) || models[o.Phase] == nil {
			summary.Unsupported = append(summary.Unsupported, o.ID+": invalid phase")
			continue
		}
		a := models[o.Phase]
		options, err := managedMutationOptions(a, o)
		if err != nil {
			summary.Unsupported = append(summary.Unsupported, o.ID+": "+err.Error())
			continue
		}
		analyze := func(opts MutationCountOptions) (MutationCountReport, error) {
			if len(o.Statements) > 0 {
				return a.AnalyzeDML(o.Statements, opts)
			}
			b, err := json.Marshal(o.Mutations)
			if err != nil {
				return MutationCountReport{}, err
			}
			return a.AnalyzeMutationsJSON(b, opts)
		}
		r, err := analyze(options)
		if err != nil {
			summary.Unsupported = append(summary.Unsupported, o.ID+": "+err.Error())
			continue
		}
		item.Baseline = &r.Bounds.Count
		item.Effective = &r.Effective
		if r.Bounds.Count.Upper != nil {
			summary.FiniteBaseline++
		}
		if r.Bounds.Count.Lower <= *o.Count && (r.Bounds.Count.Upper == nil || *o.Count <= *r.Bounds.Count.Upper) {
			summary.BaselineContains++
		} else {
			summary.Disagreements = append(summary.Disagreements, o.ID+": baseline excludes measurement")
		}
		if r.OldRows != nil {
			item.OldRows = &r.OldRows.Count
			if r.OldRows.Count.exact() && r.OldRows.Count.Lower == *o.Count {
				summary.OldExact++
			}
		}
		options.Observation = &MutationCountObservation{Runtime: "managed-single-database", ObservedAt: f.Finished, SchemaSHA256: r.SchemaSHA256, InputSHA256: r.InputSHA256, Scope: r.Scope, Successful: true, StatsPresent: true, Count: o.Count}
		measured, err := analyze(options)
		if err != nil {
			summary.Unsupported = append(summary.Unsupported, o.ID+": "+err.Error())
		} else {
			item.Effective = &measured.Effective
			if measured.Effective.exact() && measured.Effective.Lower == *o.Count {
				summary.ObservedExact++
			}
			for _, d := range measured.Disagreements {
				summary.Disagreements = append(summary.Disagreements, o.ID+": "+d)
			}
		}
		summary.Cases = append(summary.Cases, item)
	}
	return summary
}

func TestMutationCountManagedReplay(t *testing.T) {
	f, err := readManagedMutationFixture("testdata/mutation-count/managed-20261110.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	summary := replayManagedMutations(f)
	if path := os.Getenv("SPANALYZER_MUTATION_SUMMARY_PATH"); path != "" {
		b, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// The retained capture ended at a failed lease-renewal query. Its earlier
	// finalized commits remain primary observations, but the capture must never
	// be silently promoted to a complete queue-flow verification.
	if f.Status != "partial" || f.FatalError != "queue renew: InvalidArgument" || len(f.QueueReads) != 2 || f.QueueReads[1].Status != "InvalidArgument" {
		t.Fatal("unexpected primary capture completion boundary")
	}
	if len(f.CleanupErrors) != 0 || f.Lifecycle["drop"] != "OK" || f.Lifecycle["absence"] != "NotFound" {
		t.Fatal("unverified fixture cleanup")
	}
	if len(summary.Unsupported) > 0 {
		t.Errorf("analysis errors: %v", summary.Unsupported)
	}
	if len(summary.Disagreements) > 0 {
		t.Errorf("model disagreements: %v", summary.Disagreements)
	}
	if summary.Measured == 0 || summary.BaselineContains != summary.Measured || summary.ObservedExact != summary.Measured {
		t.Errorf("unaccounted cases: measured=%d contains=%d refined=%d", summary.Measured, summary.BaselineContains, summary.ObservedExact)
	}
}
