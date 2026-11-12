package spanalyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

type managedQueueReplaySummary struct {
	Status        string                       `json:"fixture_status"`
	Records       int                          `json:"transaction_attempts"`
	Measured      int                          `json:"successful_with_stats"`
	Modeled       int                          `json:"modeled_measurements"`
	Contains      int                          `json:"baseline_contains_observation"`
	Refined       int                          `json:"same_input_measured_refinements"`
	QueryReports  int                          `json:"unknown_queue_query_reports"`
	ScopeGaps     []string                     `json:"unmodeled_scopes"`
	Errors        []string                     `json:"analysis_errors"`
	Disagreements []string                     `json:"disagreements"`
	Cases         []managedMutationCaseSummary `json:"cases"`
}

func queueReplayInterval(q MutationCountInterval) *MutationCountInterval {
	// Store only the interval, not an interior pointer that would retain the
	// large per-operation reports from the 40,000-Ack boundary controls.
	copy := q
	return &copy
}

func replayManagedQueueCounts(f managedMutationFixture) managedQueueReplaySummary {
	s := managedQueueReplaySummary{Status: f.Status, Records: len(f.Observations)}
	var models []*MutationCountAnalyzer
	for _, p := range f.Phases {
		a, err := NewMutationCountAnalyzer(strings.Join(p.DDL, ";\n") + ";\n")
		if err == nil {
			a, err = a.WithObservedIndexes(a.SchemaSHA256(), p.Indexes)
		}
		if err != nil {
			s.Errors = append(s.Errors, "schema: "+err.Error())
		}
		models = append(models, a)
	}
	for _, o := range f.Observations {
		item := managedMutationCaseSummary{ID: o.ID, Status: o.Status, Observed: o.Count, Error: o.Error}
		if o.Status != "OK" {
			if o.StatsPresent || o.Count != nil {
				s.Errors = append(s.Errors, o.ID+": failed execution has numerical stats")
			}
			s.Cases = append(s.Cases, item)
			continue
		}
		if len(o.Queries) > 0 {
			if o.StatsPresent && o.Count != nil {
				s.Measured++
			}
			s.ScopeGaps = append(s.ScopeGaps, o.ID+": a queue query is not a caller mutation-count input")
			s.Cases = append(s.Cases, item)
			continue
		}
		if !o.StatsPresent || o.Count == nil {
			s.Errors = append(s.Errors, o.ID+": successful write lacks stats")
			continue
		}
		s.Measured++
		if o.Phase < 0 || o.Phase >= len(models) || models[o.Phase] == nil {
			s.Errors = append(s.Errors, o.ID+": missing preceding schema phase")
			continue
		}
		a := models[o.Phase]
		analyze := func(opts MutationCountOptions) (MutationCountReport, error) {
			if len(o.Statements) > 0 && len(o.Mutations) > 0 {
				b, err := json.Marshal(o.Mutations)
				if err != nil {
					return MutationCountReport{}, err
				}
				return a.AnalyzeTransaction(MutationCountTransaction{Statements: o.Statements, Mutations: b}, opts)
			}
			if len(o.Statements) > 0 {
				return a.AnalyzeDML(o.Statements, opts)
			}
			b, err := json.Marshal(o.Mutations)
			if err != nil {
				return MutationCountReport{}, err
			}
			return a.AnalyzeMutationsJSON(b, opts)
		}
		r, err := analyze(MutationCountOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "DML conflict clauses require a separate write model") {
				s.ScopeGaps = append(s.ScopeGaps, o.ID+": ON CONFLICT normalization is outside the current API")
				s.Cases = append(s.Cases, item)
				continue
			}
			if strings.HasPrefix(o.ID, "delete/assert-rows/") && strings.Contains(err.Error(), "but: ASSERT_ROWS_MODIFIED") {
				// The service accepted these DELETE controls, but the pinned
				// parser cannot represent their trailing assertion. Preserve the
				// exact measured SQL; stripping it would change the input identity.
				s.ScopeGaps = append(s.ScopeGaps, o.ID+": DELETE ASSERT_ROWS_MODIFIED is outside the pinned parser")
				s.Cases = append(s.Cases, item)
				continue
			}
			s.Errors = append(s.Errors, o.ID+": "+err.Error())
			continue
		}
		s.Modeled++
		item.Baseline = queueReplayInterval(r.Bounds.Count)
		if r.Bounds.Count.Lower <= *o.Count && (r.Bounds.Count.Upper == nil || *o.Count <= *r.Bounds.Count.Upper) {
			s.Contains++
		} else {
			s.Disagreements = append(s.Disagreements, o.ID+": measurement outside baseline")
		}
		observed := &MutationCountObservation{Runtime: "managed-queues-single-database", ObservedAt: f.Finished, SchemaSHA256: r.SchemaSHA256, InputSHA256: r.InputSHA256, Scope: r.Scope, Successful: true, StatsPresent: true, Count: o.Count}
		m, err := analyze(MutationCountOptions{Observation: observed})
		if err != nil {
			s.Errors = append(s.Errors, o.ID+": "+err.Error())
		} else {
			item.Effective = queueReplayInterval(m.Effective)
			if m.Effective.exact() && m.Effective.Lower == *o.Count {
				s.Refined++
			}
			for _, d := range m.Disagreements {
				s.Disagreements = append(s.Disagreements, o.ID+": "+d)
			}
		}
		s.Cases = append(s.Cases, item)
	}
	if len(models) > 0 && models[0] != nil {
		for _, q := range f.QueueReads {
			r, err := models[0].AnalyzeQueueRead(q.SQL)
			if err != nil || r.Scope != "queue_internal_queries" || r.Effective.Upper != nil {
				s.Errors = append(s.Errors, fmt.Sprintf("query %s: invalid accounting scope: %v", q.ID, err))
			} else {
				s.QueryReports++
			}
		}
	}
	return s
}

func TestMutationCountManagedQueueMatrix(t *testing.T) {
	f, err := readManagedMutationFixture("testdata/mutation-count/managed-queues-20261110.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != "complete" || f.FatalError != "" || f.Mode != "queue-matrix" || len(f.CleanupErrors) > 0 || f.Lifecycle["drop"] != "OK" || f.Lifecycle["absence"] != "NotFound" {
		t.Fatal("queue matrix did not finish its owned lifecycle")
	}
	s := replayManagedQueueCounts(f)
	if path := os.Getenv("SPANALYZER_QUEUE_SUMMARY_PATH"); path != "" {
		b, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if s.Records != 319 || s.Measured != 270 || s.Modeled != 270 || len(s.ScopeGaps) != 0 || s.Contains != s.Modeled || s.Refined != s.Modeled {
		t.Errorf("incomplete accounting coverage: records=%d measured=%d modeled=%d contains=%d refined=%d gaps=%v", s.Records, s.Measured, s.Modeled, s.Contains, s.Refined, s.ScopeGaps)
	}
	if len(s.Errors) > 0 || len(s.Disagreements) > 0 {
		t.Errorf("replay errors=%v disagreements=%v", s.Errors, s.Disagreements)
	}
	var renewed bool
	for _, r := range f.QueueReads {
		if r.ID == "renew/star-array" {
			renewed = r.Status == "OK" && r.Rows == 1 && r.Renewed
		}
	}
	if !renewed || len(f.QueueReads) != 17 || s.QueryReports != 17 {
		t.Fatal("missing positive renewal or query accounting controls")
	}
	for _, o := range f.OtherWrites {
		if o.Count != nil {
			t.Fatal("an API without CommitStats was assigned a count", o.ID)
		}
	}
	if len(f.OtherWrites) != 3 {
		t.Fatal("missing PDML/BatchWrite coverage")
	}
	if len(f.OtherWrites[2].Groups) != 3 {
		t.Fatal("missing per-group BatchWrite outcomes")
	}
	for _, g := range f.OtherWrites[2].Groups {
		if g.Status != "Unimplemented" || g.Committed {
			t.Fatal("queue BatchWrite unexpectedly represented as successful")
		}
	}
}

func TestMutationCountManagedQueueIgnoreFollowup(t *testing.T) {
	f, err := readManagedMutationFixture("testdata/mutation-count/managed-queues-ignore-20261110.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != "complete" || f.FatalError != "" || f.Mode != "queue-insert-ignore-followup" || len(f.CleanupErrors) > 0 || f.Lifecycle["drop"] != "OK" || f.Lifecycle["absence"] != "NotFound" {
		t.Fatal("follow-up did not finish its separately authorized one-DB lifecycle")
	}
	s := replayManagedQueueCounts(f)
	if path := os.Getenv("SPANALYZER_QUEUE_FOLLOWUP_SUMMARY_PATH"); path != "" {
		b, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if s.Records != 43 || s.Measured != 32 || s.Modeled != 30 || s.Contains != 30 || s.Refined != 30 || s.QueryReports != 2 || len(s.ScopeGaps) != 2 || len(s.Errors) != 0 || len(s.Disagreements) != 0 {
		t.Fatalf("incomplete follow-up replay: records=%d measured=%d modeled=%d contains=%d refined=%d queries=%d gaps=%v errors=%v disagreements=%v", s.Records, s.Measured, s.Modeled, s.Contains, s.Refined, s.QueryReports, s.ScopeGaps, s.Errors, s.Disagreements)
	}
	for _, gap := range s.ScopeGaps {
		if !strings.Contains(gap, "DELETE ASSERT_ROWS_MODIFIED") {
			t.Fatal("unexpected follow-up scope gap", gap)
		}
	}
	conflicts, rwRenewals, insertionChecks, zeroInserts := 0, 0, 0, 0
	for _, o := range f.Observations {
		if strings.HasPrefix(o.ID, "conflict/") {
			if o.Status != "InvalidArgument" || !strings.Contains(o.Error, "not supported on queues") {
				t.Fatal("queue ON CONFLICT was not rejected", o.ID)
			}
			conflicts++
		}
		if strings.HasPrefix(o.ID, "lease/renew-in-rw/") {
			if o.Status != "InvalidArgument" || !strings.Contains(o.Error, "must be single-use only") {
				t.Fatal("RW renewal boundary changed", o.ID)
			}
			rwRenewals++
		}
		if o.Status == "OK" && (o.Kind == "dml_insert_ignore" || o.Kind == "dml_insert_select") {
			if len(o.Rows) == 0 {
				t.Fatal("successful insertion lacks affected-row evidence", o.ID)
			}
			var inserted int64
			for _, rows := range o.Rows {
				inserted += rows
			}
			if o.Count == nil || *o.Count != 5*inserted {
				t.Fatal("captured insertion differs from 5 per actually inserted row", o.ID)
			}
			insertionChecks++
			if inserted == 0 {
				zeroInserts++
			}
		}
	}
	if conflicts != 6 || rwRenewals != 2 || insertionChecks != 28 || zeroInserts != 12 {
		t.Fatalf("missing controls: conflicts=%d renewals=%d insertions=%d zero=%d", conflicts, rwRenewals, insertionChecks, zeroInserts)
	}
}
