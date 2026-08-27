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
}

func TestRunCheckRejectsWrite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := run([]string{"check", "--target", "managed", "--write"}, &stdout, &stderr)
	if got != 2 || !strings.Contains(stderr.String(), "does not accept") {
		t.Fatalf("run() = %d, stderr=%q", got, stderr.String())
	}
}
