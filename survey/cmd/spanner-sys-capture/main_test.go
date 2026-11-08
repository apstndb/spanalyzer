package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apstndb/spanalyzer/survey/internal/capturemeta"
	"github.com/apstndb/spanalyzer/survey/spannersys"
)

func TestRunRejectsUnsupportedTarget(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := run([]string{"--target", "emulator"}, &stdout, &stderr); got != 2 {
		t.Fatalf("run() = %d, want 2; stderr=%s", got, stderr.String())
	}
}

func TestCompareRetainedCaptureDetectsOmniIdentityOverride(t *testing.T) {
	document, err := spannersys.BuildSurfaceCapture([]spannersys.ColumnObservation{
		{TableName: "EXAMPLE", ColumnName: "ID", SpannerType: "INT64", OrdinalPosition: 1},
	}, time.Date(2026, 8, 27, 1, 2, 3, 0, time.UTC), capturemeta.Target{
		Kind: "omni",
		Image: &capturemeta.ImageIdentity{
			Family: "us-docker.pkg.dev/spanner-omni/images/spanner-omni",
			Tag:    "2026.r2.1-beta", Digest: "sha256:" + strings.Repeat("f", 64), Platform: "linux/arm64",
		},
	}, spannersys.SurfaceCaptureProducerIdentity{
		SourceSHA256: strings.Repeat("a", 64), InvocationSHA256: strings.Repeat("b", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	report, err := compareRetainedCapture(repoRoot, document)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "target_identity_changed" || !report.MaterialChange {
		t.Fatalf("comparison report = %#v", report)
	}
	if report.SurfaceDelta != nil {
		t.Fatalf("SurfaceDelta = %#v, want absent", report.SurfaceDelta)
	}
}

func TestClassifyV0Alpha2ComparisonSurfaceDelta(t *testing.T) {
	producer := spannersys.SurfaceCaptureProducerIdentity{
		SourceSHA256: strings.Repeat("a", 64), InvocationSHA256: strings.Repeat("b", 64),
	}
	target := capturemeta.Target{Kind: "managed", ObservationScope: "single_database"}
	retained, err := spannersys.BuildSurfaceCapture([]spannersys.ColumnObservation{
		{TableName: "APPLE", ColumnName: "COL_B", SpannerType: "INT64", OrdinalPosition: 1},
		{TableName: "APPLE", ColumnName: "COL_C", SpannerType: "STRING(MAX)", OrdinalPosition: 2},
		{TableName: "ZEBRA", ColumnName: "OLD", SpannerType: "BOOL", OrdinalPosition: 1},
	}, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), target, producer)
	if err != nil {
		t.Fatal(err)
	}
	current, err := spannersys.BuildSurfaceCapture([]spannersys.ColumnObservation{
		{TableName: "APPLE", ColumnName: "COL_A", SpannerType: "BOOL", OrdinalPosition: 1},
		{TableName: "APPLE", ColumnName: "COL_B", SpannerType: "STRING(MAX)", OrdinalPosition: 2},
		{TableName: "APPLE", ColumnName: "COL_C", SpannerType: "STRING(MAX)", OrdinalPosition: 3},
		{TableName: "MANGO", ColumnName: "NEW", SpannerType: "INT64", OrdinalPosition: 1},
	}, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), target, producer)
	if err != nil {
		t.Fatal(err)
	}
	report := &comparisonReport{}
	classifyV0Alpha2Comparison(report, current, retained, false)
	if report.Status != "surface_changed" || !report.MaterialChange || report.SurfaceDelta == nil {
		t.Fatalf("report = %#v", report)
	}
	wantAdded := []surfaceColumnRef{
		{TableName: "APPLE", ColumnName: "COL_A", SpannerType: "BOOL", OrdinalPosition: 1},
		{TableName: "MANGO", ColumnName: "NEW", SpannerType: "INT64", OrdinalPosition: 1},
	}
	if len(report.SurfaceDelta.AddedColumns) != len(wantAdded) {
		t.Fatalf("added = %#v, want %#v", report.SurfaceDelta.AddedColumns, wantAdded)
	}
	for i := range wantAdded {
		if report.SurfaceDelta.AddedColumns[i] != wantAdded[i] {
			t.Fatalf("added[%d] = %#v, want %#v", i, report.SurfaceDelta.AddedColumns[i], wantAdded[i])
		}
	}
	if len(report.SurfaceDelta.RemovedColumns) != 1 || report.SurfaceDelta.RemovedColumns[0].TableName != "ZEBRA" {
		t.Fatalf("removed = %#v", report.SurfaceDelta.RemovedColumns)
	}
	if len(report.SurfaceDelta.ChangedColumns) != 2 ||
		report.SurfaceDelta.ChangedColumns[0].ColumnName != "COL_B" ||
		report.SurfaceDelta.ChangedColumns[1].ColumnName != "COL_C" {
		t.Fatalf("changed = %#v", report.SurfaceDelta.ChangedColumns)
	}
	if report.SurfaceDelta.ChangedColumns[0].Retained.SpannerType != "INT64" ||
		report.SurfaceDelta.ChangedColumns[0].Current.SpannerType != "STRING(MAX)" {
		t.Fatalf("COL_B type change = %#v", report.SurfaceDelta.ChangedColumns[0])
	}
	if report.SurfaceDelta.ChangedColumns[1].Retained.OrdinalPosition != 2 ||
		report.SurfaceDelta.ChangedColumns[1].Current.OrdinalPosition != 3 {
		t.Fatalf("COL_C ordinal change = %#v", report.SurfaceDelta.ChangedColumns[1])
	}
}

func TestClassifyV0Alpha2ComparisonOmitsSurfaceDelta(t *testing.T) {
	producer := spannersys.SurfaceCaptureProducerIdentity{
		SourceSHA256: strings.Repeat("a", 64), InvocationSHA256: strings.Repeat("b", 64),
	}
	target := capturemeta.Target{Kind: "managed", ObservationScope: "single_database"}
	retained, err := spannersys.BuildSurfaceCapture([]spannersys.ColumnObservation{
		{TableName: "EXAMPLE", ColumnName: "ID", SpannerType: "INT64", OrdinalPosition: 1},
	}, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), target, producer)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := spannersys.BuildSurfaceCapture([]spannersys.ColumnObservation{
		{TableName: "EXAMPLE", ColumnName: "ID", SpannerType: "INT64", OrdinalPosition: 1},
	}, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), target, producer)
	if err != nil {
		t.Fatal(err)
	}
	producerOnly, err := spannersys.BuildSurfaceCapture([]spannersys.ColumnObservation{
		{TableName: "EXAMPLE", ColumnName: "ID", SpannerType: "INT64", OrdinalPosition: 1},
	}, time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), target, spannersys.SurfaceCaptureProducerIdentity{
		SourceSHA256: strings.Repeat("c", 64), InvocationSHA256: strings.Repeat("b", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name          string
		current       *spannersys.SurfaceCapture
		targetChanged bool
		wantStatus    string
	}{
		{name: "unchanged", current: unchanged, wantStatus: "unchanged"},
		{name: "producer only", current: producerOnly, wantStatus: "producer_changed_only"},
		{name: "target identity", current: unchanged, targetChanged: true, wantStatus: "target_identity_changed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := &comparisonReport{}
			classifyV0Alpha2Comparison(report, test.current, retained, test.targetChanged)
			if report.Status != test.wantStatus || report.SurfaceDelta != nil {
				t.Fatalf("report = %#v, want status %q and no surface_delta", report, test.wantStatus)
			}
		})
	}
}

func TestRunCheckRejectsWrite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := run([]string{"check", "--target", "managed", "--write"}, &stdout, &stderr)
	if got != 2 || !strings.Contains(stderr.String(), "does not accept") {
		t.Fatalf("run() = %d, stderr=%q", got, stderr.String())
	}
}
