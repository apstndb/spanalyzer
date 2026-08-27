package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
