package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apstndb/spanalyzer/survey/infoschem"
)

func TestCompareRetainedCapture(t *testing.T) {
	repoRoot, err := resolveRepoRoot("")
	if err != nil {
		t.Fatal(err)
	}
	selected, err := selectedManagedObservationPath(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := readRetainedCapture(repoRoot, selected)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name         string
		mutate       func(*infoschem.CaptureDocument)
		wantStatus   string
		wantMaterial bool
		wantProducer bool
		wantDelta    bool
	}{
		{name: "unchanged", mutate: func(*infoschem.CaptureDocument) {}, wantStatus: "unchanged"},
		{
			name: "producer only",
			mutate: func(capture *infoschem.CaptureDocument) {
				capture.ProducerSourceSHA256 = strings.Repeat("f", 64)
			},
			wantStatus:   "producer_changed_only",
			wantProducer: true,
		},
		{
			name: "surface",
			mutate: func(capture *infoschem.CaptureDocument) {
				capture.SurfaceSHA256 = strings.Repeat("e", 64)
			},
			wantStatus:   "surface_changed",
			wantMaterial: true,
			wantDelta:    true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := cloneCapture(t, retained)
			test.mutate(current)
			report, err := compareRetainedCapture(repoRoot, current)
			if err != nil {
				t.Fatal(err)
			}
			if report.Status != test.wantStatus || report.MaterialChange != test.wantMaterial {
				t.Fatalf("comparison = status %q material %t, want %q/%t", report.Status, report.MaterialChange, test.wantStatus, test.wantMaterial)
			}
			if (report.ProducerChange != nil) != test.wantProducer {
				t.Fatalf("ProducerChange = %#v, want present %t", report.ProducerChange, test.wantProducer)
			}
			if (report.SurfaceDelta != nil) != test.wantDelta {
				t.Fatalf("SurfaceDelta = %#v, want present %t", report.SurfaceDelta, test.wantDelta)
			}
			if report.Observations != nil {
				t.Fatalf("Observations = %#v, want absent", report.Observations)
			}
		})
	}
}

func TestCompareRetainedCaptureDetectsTargetIdentityChange(t *testing.T) {
	repoRoot, err := resolveRepoRoot("")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(
		repoRoot,
		"survey/infoschem/evidence/omni/ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741/linux-arm64-f8b0ea3092e7.json",
	)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := infoschem.DecodeCapture(data)
	if err != nil {
		t.Fatal(err)
	}
	current := cloneCapture(t, retained)
	current.Target.Image.Tag = "experimental"
	report, err := compareRetainedCapture(repoRoot, current)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "target_identity_changed" || !report.MaterialChange {
		t.Fatalf("comparison = %#v, want target_identity_changed material change", report)
	}
	if report.SurfaceDelta != nil {
		t.Fatalf("SurfaceDelta = %#v, want absent", report.SurfaceDelta)
	}
	if report.Observations != nil {
		t.Fatalf("Observations = %#v, want absent", report.Observations)
	}
}

func TestWriteComparisonReportIsCompactAndMachineReadable(t *testing.T) {
	report := &comparisonReport{SchemaVersion: comparisonSchemaVersion, Status: "unchanged"}
	var output bytes.Buffer
	if err := writeComparisonReport(&output, report); err != nil {
		t.Fatal(err)
	}
	var decoded comparisonReport
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Status != "unchanged" {
		t.Fatalf("decoded status = %q", decoded.Status)
	}
	if output.Len() > 1000 {
		t.Fatalf("empty compact report size = %d, want <= 1000", output.Len())
	}
}

func TestAggregateCheckObservationsSurfaceDelta(t *testing.T) {
	retained := syntheticCapture(t, infoschem.DiscoveredColumnMetadata{
		"APPLE": {
			"COL_B": {Name: "COL_B", SpannerType: "INT64", OrdinalPosition: 1},
			"COL_C": {Name: "COL_C", SpannerType: "STRING(MAX)", OrdinalPosition: 2},
		},
		"ZEBRA": {
			"OLD": {Name: "OLD", SpannerType: "BOOL", OrdinalPosition: 1},
		},
	}, nil, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	current := syntheticCapture(t, infoschem.DiscoveredColumnMetadata{
		"APPLE": {
			"COL_A": {Name: "COL_A", SpannerType: "BOOL", OrdinalPosition: 1},
			"COL_B": {Name: "COL_B", SpannerType: "STRING(MAX)", OrdinalPosition: 2},
			"COL_C": {Name: "COL_C", SpannerType: "STRING(MAX)", OrdinalPosition: 3},
		},
		"MANGO": {
			"NEW": {Name: "NEW", SpannerType: "INT64", OrdinalPosition: 1},
		},
	}, nil, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	report := aggregateCheckObservations("retained.json", retained, false, []*infoschem.CaptureDocument{current})
	if report.Status != "surface_changed" || !report.MaterialChange || report.SurfaceDelta == nil {
		t.Fatalf("report = %#v", report)
	}
	assertColumnRefs(t, report.SurfaceDelta.AddedColumns, []surfaceColumnRef{
		{TableName: "APPLE", ColumnName: "COL_A", SpannerType: "BOOL", OrdinalPosition: 1},
		{TableName: "MANGO", ColumnName: "NEW", SpannerType: "INT64", OrdinalPosition: 1},
	})
	assertColumnRefs(t, report.SurfaceDelta.RemovedColumns, []surfaceColumnRef{
		{TableName: "ZEBRA", ColumnName: "OLD", SpannerType: "BOOL", OrdinalPosition: 1},
	})
	if len(report.SurfaceDelta.ChangedColumns) != 2 {
		t.Fatalf("changed columns = %#v", report.SurfaceDelta.ChangedColumns)
	}
	if report.SurfaceDelta.ChangedColumns[0].TableName != "APPLE" || report.SurfaceDelta.ChangedColumns[0].ColumnName != "COL_B" {
		t.Fatalf("first changed column = %#v", report.SurfaceDelta.ChangedColumns[0])
	}
	if report.SurfaceDelta.ChangedColumns[0].Retained != (surfaceColumnFields{SpannerType: "INT64", OrdinalPosition: 1}) ||
		report.SurfaceDelta.ChangedColumns[0].Current != (surfaceColumnFields{SpannerType: "STRING(MAX)", OrdinalPosition: 2}) {
		t.Fatalf("COL_B change = %#v", report.SurfaceDelta.ChangedColumns[0])
	}
	if report.SurfaceDelta.ChangedColumns[1].TableName != "APPLE" || report.SurfaceDelta.ChangedColumns[1].ColumnName != "COL_C" {
		t.Fatalf("second changed column = %#v", report.SurfaceDelta.ChangedColumns[1])
	}
	if report.SurfaceDelta.ChangedColumns[1].Retained.OrdinalPosition != 2 || report.SurfaceDelta.ChangedColumns[1].Current.OrdinalPosition != 3 {
		t.Fatalf("COL_C ordinal change = %#v", report.SurfaceDelta.ChangedColumns[1])
	}
}

func TestAggregateCheckObservationsRollingQueryabilityChange(t *testing.T) {
	metadata := infoschem.DiscoveredColumnMetadata{
		"INDEXES": {
			"SEARCH_UNNEST": {Name: "SEARCH_UNNEST", SpannerType: "BOOL", OrdinalPosition: 1},
		},
	}
	retainedRolling := []infoschem.RollingQueryability{
		{TableName: "INDEXES", ColumnName: "SEARCH_UNNEST", Status: "queryable"},
		{TableName: "INDEX_COLUMNS", ColumnName: "EXPRESSION", Status: "not_advertised"},
	}
	currentRolling := []infoschem.RollingQueryability{
		{TableName: "INDEXES", ColumnName: "SEARCH_UNNEST", Status: "advertised_not_queryable", StatusCode: "UNIMPLEMENTED"},
		{TableName: "INDEX_COLUMNS", ColumnName: "EXPRESSION", Status: "not_advertised"},
	}
	retained := syntheticCapture(t, metadata, retainedRolling, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	current := syntheticCapture(t, metadata, currentRolling, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	report := aggregateCheckObservations("retained.json", retained, false, []*infoschem.CaptureDocument{current})
	if report.Status != "surface_changed" || report.SurfaceDelta == nil {
		t.Fatalf("report = %#v", report)
	}
	want := []rollingQueryabilityChange{{
		TableName: "INDEXES", ColumnName: "SEARCH_UNNEST",
		RetainedStatus: "queryable", CurrentStatus: "advertised_not_queryable",
		CurrentStatusCode: "UNIMPLEMENTED",
	}}
	if len(report.SurfaceDelta.RollingQueryabilityChanges) != 1 || report.SurfaceDelta.RollingQueryabilityChanges[0] != want[0] {
		t.Fatalf("rolling changes = %#v, want %#v", report.SurfaceDelta.RollingQueryabilityChanges, want)
	}
}

func TestAggregateCheckObservationsAgreeUsesFirstObservation(t *testing.T) {
	retained := syntheticCapture(t, infoschem.DiscoveredColumnMetadata{
		"EXAMPLE": {"ID": {Name: "ID", SpannerType: "INT64", OrdinalPosition: 1}},
	}, nil, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	first := cloneCapture(t, retained)
	first.ObservedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	second := cloneCapture(t, first)
	second.ObservedAt = time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	report := aggregateCheckObservations("retained.json", retained, false, []*infoschem.CaptureDocument{first, second})
	if report.Status != "unchanged" || report.SurfaceDelta != nil || report.Observations != nil {
		t.Fatalf("agreeing observations = %#v", report)
	}
	if report.Current.ObservedAt != first.ObservedAt {
		t.Fatalf("current observed_at = %q, want first observation", report.Current.ObservedAt)
	}
}

func TestAggregateCheckObservationsDisagree(t *testing.T) {
	retained := syntheticCapture(t, infoschem.DiscoveredColumnMetadata{
		"EXAMPLE": {"ID": {Name: "ID", SpannerType: "INT64", OrdinalPosition: 1}},
	}, nil, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	matching := cloneCapture(t, retained)
	matching.ObservedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	differing := syntheticCapture(t, infoschem.DiscoveredColumnMetadata{
		"EXAMPLE": {
			"ID":  {Name: "ID", SpannerType: "INT64", OrdinalPosition: 1},
			"NEW": {Name: "NEW", SpannerType: "BOOL", OrdinalPosition: 2},
		},
	}, nil, time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC))
	report := aggregateCheckObservations("retained.json", retained, false, []*infoschem.CaptureDocument{matching, differing})
	if report.Status != "surface_inconsistent" || !report.MaterialChange {
		t.Fatalf("disagreeing observations = %#v", report)
	}
	if report.Current.SurfaceSHA256 != matching.SurfaceSHA256 {
		t.Fatalf("current hash = %q, want first observation", report.Current.SurfaceSHA256)
	}
	if len(report.Observations) != 2 {
		t.Fatalf("observations = %#v", report.Observations)
	}
	if report.Observations[0].SurfaceSHA256 != matching.SurfaceSHA256 || report.Observations[1].SurfaceSHA256 != differing.SurfaceSHA256 {
		t.Fatalf("observation hashes = %#v", report.Observations)
	}
	if report.Observations[0].ColumnCount != 1 || report.Observations[1].ColumnCount != 2 {
		t.Fatalf("observation column counts = %#v", report.Observations)
	}
	if report.SurfaceDelta == nil || len(report.SurfaceDelta.AddedColumns) != 1 ||
		report.SurfaceDelta.AddedColumns[0].TableName != "EXAMPLE" || report.SurfaceDelta.AddedColumns[0].ColumnName != "NEW" {
		t.Fatalf("inconsistent surface_delta = %#v", report.SurfaceDelta)
	}
}

func TestAggregateCheckObservationsTargetIdentityOmitsDelta(t *testing.T) {
	retained := syntheticCapture(t, infoschem.DiscoveredColumnMetadata{
		"EXAMPLE": {"ID": {Name: "ID", SpannerType: "INT64", OrdinalPosition: 1}},
	}, nil, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	current := cloneCapture(t, retained)
	report := aggregateCheckObservations("retained.json", retained, true, []*infoschem.CaptureDocument{current})
	if report.Status != "target_identity_changed" || report.SurfaceDelta != nil {
		t.Fatalf("target identity report = %#v", report)
	}
}

func TestValidateObservationsFlag(t *testing.T) {
	tests := []struct {
		name            string
		checkMode       bool
		observations    int
		observationsSet bool
		wantErr         bool
	}{
		{name: "check default", checkMode: true, observations: 1},
		{name: "check ten", checkMode: true, observations: 10, observationsSet: true},
		{name: "check zero", checkMode: true, observations: 0, observationsSet: true, wantErr: true},
		{name: "check eleven", checkMode: true, observations: 11, observationsSet: true, wantErr: true},
		{name: "non-check set", checkMode: false, observations: 1, observationsSet: true, wantErr: true},
		{name: "non-check unset", checkMode: false, observations: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateObservationsFlag(test.checkMode, test.observations, test.observationsSet)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateObservationsFlag() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

func TestRunCheckRejectsInvalidObservations(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "zero", args: []string{"check", "--target", "managed", "--observations", "0"}},
		{name: "eleven", args: []string{"check", "--target", "managed", "--observations", "11"}},
		{name: "non-check", args: []string{"--target", "managed", "--observations", "1"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := run(test.args, &stdout, &stderr)
			if got != 2 || !strings.Contains(stderr.String(), "--observations") {
				t.Fatalf("run() = %d, stderr=%q", got, stderr.String())
			}
		})
	}
}

func TestWriteComparisonReportOmitsSurfaceDeltaWhenAbsent(t *testing.T) {
	var output bytes.Buffer
	if err := writeComparisonReport(&output, &comparisonReport{SchemaVersion: comparisonSchemaVersion, Status: "unchanged"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "surface_delta") || strings.Contains(output.String(), "observations") {
		t.Fatalf("unchanged report included optional fields: %s", output.String())
	}
}

func syntheticCapture(t *testing.T, metadata infoschem.DiscoveredColumnMetadata, rolling []infoschem.RollingQueryability, observed time.Time) *infoschem.CaptureDocument {
	t.Helper()
	if rolling == nil {
		rolling = []infoschem.RollingQueryability{
			{TableName: "INDEXES", ColumnName: "SEARCH_UNNEST", Status: "not_advertised"},
			{TableName: "INDEX_COLUMNS", ColumnName: "EXPRESSION", Status: "not_advertised"},
		}
	}
	document, err := infoschem.BuildCapture(metadata, rolling, observed, infoschem.CaptureTarget{
		Kind: "managed", ObservationScope: "single_database",
	}, infoschem.ProducerIdentity{
		SourceSHA256: strings.Repeat("a", 64), InvocationSHA256: strings.Repeat("b", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func assertColumnRefs(t *testing.T, got, want []surfaceColumnRef) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("column refs = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("column refs[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func cloneCapture(t *testing.T, capture *infoschem.CaptureDocument) *infoschem.CaptureDocument {
	t.Helper()
	data, err := infoschem.EncodeCapture(capture)
	if err != nil {
		t.Fatal(err)
	}
	clone, err := infoschem.DecodeCapture(data)
	if err != nil {
		t.Fatal(err)
	}
	return clone
}

func TestComparisonIncludesQueryabilityErrorCodeChange(t *testing.T) {
	metadata := infoschem.DiscoveredColumnMetadata{
		"INDEXES": {"SEARCH_UNNEST": {Name: "SEARCH_UNNEST", SpannerType: "BOOL", OrdinalPosition: 1}},
	}
	build := func(code string) *infoschem.CaptureDocument {
		return syntheticCapture(t, metadata, []infoschem.RollingQueryability{
			{TableName: "INDEXES", ColumnName: "SEARCH_UNNEST", Status: "advertised_not_queryable", StatusCode: code},
			{TableName: "INDEX_COLUMNS", ColumnName: "EXPRESSION", Status: "not_advertised"},
		}, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	}
	report := aggregateCheckObservations("retained.json", build("UNIMPLEMENTED"), false, []*infoschem.CaptureDocument{build("INVALID_ARGUMENT")})
	if report.SchemaVersion != "v0alpha2" || report.Status != "surface_changed" || report.SurfaceDelta == nil {
		t.Fatalf("comparison = %#v", report)
	}
	changes := report.SurfaceDelta.RollingQueryabilityChanges
	if len(changes) != 1 || changes[0].RetainedStatusCode != "UNIMPLEMENTED" || changes[0].CurrentStatusCode != "INVALID_ARGUMENT" {
		t.Fatalf("error-code-only change was hidden: %#v", changes)
	}
}
