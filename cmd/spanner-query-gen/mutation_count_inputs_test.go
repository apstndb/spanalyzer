package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanemuboost"
	"google.golang.org/protobuf/types/known/structpb"
)

type countFeedbackAnalyzer struct {
	feedbackPlanAnalyzer
	count        *int64
	measureCalls int
}

type indexFeedbackAnalyzer struct {
	feedbackPlanAnalyzer
	indexError error
}

func (a *indexFeedbackAnalyzer) ObserveIndexes(context.Context) ([]spanalyzer.MutationCountIndexObservation, error) {
	return nil, a.indexError
}

func TestIndexObservationFailureAndExplicitSnapshotRejectionRemainDistinct(t *testing.T) {
	config, dir, opts := feedbackReportFixture(t)
	config.Writes = []querygen.QueryCodegenWrite{{Name: "InsertT", Catalog: "good", Table: "T", Operation: "insert", InputStruct: "Input", Insert: querygen.QueryCodegenWriteInsert{Columns: []string{"Id"}}, Methods: []string{"mutation", "dml"}}}
	plan := querygen.BuildQueryCodegenReportPlan(config, dir)
	analyzer := &indexFeedbackAnalyzer{indexError: errors.New("catalog unavailable")}
	open := func(context.Context, spanemuboost.RuntimeHandle, querygen.QueryCodegenSchema, string, *spanalyzer.ProtoDescriptorSet, planReportOptimizer) (planReportAnalyzer, error) {
		return analyzer, nil
	}
	r, err := buildPlanReportWithAnalyzer(t.Context(), config, plan, dir, opts, nil, open)
	if err != nil || r.Queries[0].Status != "ok" || len(r.Queries[0].MutationCounts) != 2 || !strings.Contains(r.Queries[0].MutationCountError, "catalog unavailable") {
		t.Fatalf("optional metadata failure lost conservative counts or SQL PLAN: %+v %v", r, err)
	}
	analyzer.indexError = nil
	opts.MutationInputs.Catalogs = map[string]spanalyzer.MutationCountIndexSnapshot{"good": {SchemaSHA256: "wrong"}}
	r, err = buildPlanReportWithAnalyzer(t.Context(), config, plan, dir, opts, nil, open)
	if err != nil || len(r.Queries[0].MutationCounts) != 0 || !strings.Contains(r.Queries[0].MutationCountError, "supplied index snapshot differs") {
		t.Fatalf("explicit mismatched snapshot silently fell back: %+v %v", r, err)
	}
	for _, warning := range r.Warnings {
		if warning.ID == "mutation-index-snapshot-rejected" {
			return
		}
	}
	t.Fatal("snapshot rejection did not retain its failing diagnostic")
}

func TestMeasurementDeleteKeyUsesSDKScalarInsteadOfGenericValue(t *testing.T) {
	value := spanner.GenericColumnValue{Type: &spannerpb.Type{Code: spannerpb.TypeCode_INT64}, Value: structpb.NewStringValue("9007199254740993")}
	key, err := planReportMeasurementKeyPart(value)
	if err != nil || key != (spanner.NullInt64{Int64: 9007199254740993, Valid: true}) {
		t.Fatalf("delete key = %#v, %v", key, err)
	}
	value.Type.Code = spannerpb.TypeCode_JSON
	if _, err := planReportMeasurementKeyPart(value); err == nil {
		t.Fatal("unsupported key type reached the seed transaction")
	}
}

func (a *countFeedbackAnalyzer) MeasureWrite(context.Context, querygen.QueryCodegenPlanWrite, querygen.MutationCountWriteInput, spanalyzer.MutationWriteShape, string, map[string]string) (*int64, []spanalyzer.MutationCountRowFacts, error) {
	a.measureCalls++
	return a.count, nil, nil
}

func TestPlanReportCountOnlyTargetDoesNotWriteByDefault(t *testing.T) {
	config, dir, opts := feedbackReportFixture(t)
	config.Writes = []querygen.QueryCodegenWrite{{Name: "InsertT", Catalog: "good", Table: "T", Operation: "insert", InputStruct: "Input", Insert: querygen.QueryCodegenWriteInsert{Columns: []string{"Id"}}, Methods: []string{"mutation"}}}
	plan := querygen.BuildQueryCodegenReportPlan(config, dir)
	input := querygen.MutationCountWriteInput{Values: map[string]interface{}{"Id": "1"}}
	opts.MutationInputs = querygen.MutationCountInputs{Writes: map[string]map[string]querygen.MutationCountWriteInput{"InsertT": {"mutation": input}}}
	if err := querygen.RefinePlanMutationCounts(plan, config, dir, opts.MutationInputs); err != nil {
		t.Fatal(err)
	}
	count := int64(99)
	analyzer := &countFeedbackAnalyzer{count: &count}
	open := func(context.Context, spanemuboost.RuntimeHandle, querygen.QueryCodegenSchema, string, *spanalyzer.ProtoDescriptorSet, planReportOptimizer) (planReportAnalyzer, error) {
		return analyzer, nil
	}
	for _, measure := range []bool{false, true} {
		opts.MeasureMutationCounts = measure
		r, err := buildPlanReportWithAnalyzer(t.Context(), config, plan, dir, opts, nil, open)
		if err != nil {
			t.Fatal(err)
		}
		q := r.Queries[0]
		if q.Status != "skipped" || len(q.MutationCounts) != 1 {
			t.Fatalf("count-only target lost accounting: %+v", q)
		}
		if !measure && analyzer.measureCalls != 0 {
			t.Fatal("default PLAN path committed a write")
		}
		if measure {
			if analyzer.measureCalls != 1 || len(q.MutationCounts["mutation"].Disagreements) == 0 || q.MutationCounts["mutation"].Effective.Lower == 99 || q.MutationCounts["mutation"].Observation.Count == nil || len(r.Warnings) == 0 {
				t.Fatalf("disagreement was hidden: %+v", r)
			}
		}
	}
	analyzer.count = nil
	opts.MeasureMutationCounts = true
	r, err := buildPlanReportWithAnalyzer(t.Context(), config, plan, dir, opts, nil, open)
	if err != nil || !strings.Contains(r.Queries[0].MutationCountError, "no CommitStats") || r.Queries[0].MutationCounts["mutation"].Observation.StatsPresent {
		t.Fatalf("missing statistics became zero: %+v %v", r, err)
	}
	failedOpen := func(context.Context, spanemuboost.RuntimeHandle, querygen.QueryCodegenSchema, string, *spanalyzer.ProtoDescriptorSet, planReportOptimizer) (planReportAnalyzer, error) {
		return nil, errors.New("fixture unavailable")
	}
	r, err = buildPlanReportWithAnalyzer(t.Context(), config, plan, dir, opts, nil, failedOpen)
	if err != nil || r.Queries[0].Status != "error" || !strings.Contains(r.Queries[0].MutationCountError, "measurement was not executed") {
		t.Fatalf("unattempted measurement disappeared: %+v %v", r, err)
	}
	for _, warning := range r.Warnings {
		if warning.ID == "mutation-count-measurement-failed" {
			return
		}
	}
	t.Fatal("unattempted measurement did not retain its failing diagnostic")
}

func TestMutationInputFilesRejectUnknownFieldsAndTrailingDocuments(t *testing.T) {
	dir := t.TempDir()
	for _, data := range []string{`{"writez":{}}`, `{"writes":{}} {"writes":{}}`, `{"writes":{"Insert":{"mutation":{"valuess":{}}}}}`} {
		path := filepath.Join(dir, "facts.json")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := readQuerygenMutationInputs(dir, "facts.json", ""); err == nil {
			t.Fatal("bad input accepted", data)
		}
	}
}

func TestMutationIndexFileKeepsNullFilteringAndRejectsUnknownAttributes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "indexes.json")
	data := `{"catalogs":{"app":{"schema_sha256":"hash","indexes":[{"table":"Events","name":"physical","managed":true,"null_filtered":true,"columns":["Owner"]}]}}}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	inputs, _, err := readQuerygenMutationInputs(dir, "", "indexes.json")
	if err != nil || !inputs.Catalogs["app"].Indexes[0].NullFiltered {
		t.Fatalf("index file lost null filtering: %+v %v", inputs, err)
	}
	data = strings.Replace(data, "null_filtered", "null_filterd", 1)
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readQuerygenMutationInputs(dir, "", "indexes.json"); err == nil {
		t.Fatal("custom index decoder bypassed unknown-field validation")
	}
}
