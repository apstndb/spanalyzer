package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidatesRepositoryManifest(t *testing.T) {
	if err := run([]string{"--repo-root", "../.."}); err != nil {
		t.Fatal(err)
	}
}

func TestFindRepoRoot(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, start := range []string{
		root,
		filepath.Join(root, "tools"),
		filepath.Join(root, "tools", "infoschema-survey-check"),
	} {
		got, err := findRepoRoot(start)
		if err != nil {
			t.Fatalf("findRepoRoot(%q) error = %v", start, err)
		}
		if got != root {
			t.Fatalf("findRepoRoot(%q) = %q, want %q", start, got, root)
		}
	}
}

func TestBuildManifestBindsObservation(t *testing.T) {
	policy := fixtureProjectionSource()
	exported := fixtureSurveyExport()
	document, err := buildManifest(&policy, strings.Repeat("9", 64), &exported)
	if err != nil {
		t.Fatal(err)
	}
	if document.SchemaVersion != "v0alpha2" {
		t.Fatalf("SchemaVersion = %q", document.SchemaVersion)
	}
	if document.Source.SelectedObservationPath != policy.SelectedObservation.Path || document.Source.ObservedAt != exported.Capture.ObservedAt {
		t.Fatalf("manifest source = %#v", document.Source)
	}
	if got, want := document.Tables[0].Columns[0].EvidenceStatus, "live_observed"; got != want {
		t.Fatalf("evidence status = %q, want %q", got, want)
	}
	if got, want := document.Tables[0].Columns[1].EvidenceStatus, "rolling"; got != want {
		t.Fatalf("rolling evidence status = %q, want %q", got, want)
	}
	if got, want := document.Tables[0].Columns[1].ProjectedType, "STRING(MAX)"; got != want {
		t.Fatalf("projected type = %q, want %q", got, want)
	}
	if got, want := document.Tables[0].Columns[2].EvidenceStatus, "docs_only_absent"; got != want {
		t.Fatalf("documentation-only status = %q, want %q", got, want)
	}
	if err := validateManifest(document); err != nil {
		t.Fatal(err)
	}
}

func TestBuildManifestRejectsMissingStableColumn(t *testing.T) {
	policy := fixtureProjectionSource()
	exported := fixtureSurveyExport()
	exported.Capture.Columns = nil
	_, err := buildManifest(&policy, strings.Repeat("9", 64), &exported)
	if err == nil || !strings.Contains(err.Error(), "missing stable registry column") {
		t.Fatalf("buildManifest() error = %v, want missing stable column", err)
	}
}

func TestValidateProjectionSourceRejectsImplicitLatest(t *testing.T) {
	policy := fixtureProjectionSource()
	policy.SelectedObservation.Path = "survey/infoschem/evidence/managed/latest.json"
	if err := validateProjectionSource(&policy); err == nil {
		t.Fatal("validateProjectionSource() error = nil")
	}
}

func TestDecodeProjectionSourceRejectsDuplicateKeys(t *testing.T) {
	data, err := json.Marshal(fixtureProjectionSource())
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string][]byte{
		"top level": bytes.Replace(data, []byte(`"mode":"managed_live_primary"`), []byte(`"mode":"managed_live_primary","mode":"managed_live_primary"`), 1),
		"nested":    bytes.Replace(data, []byte(`"file_sha256":"`), []byte(`"file_sha256":"`+strings.Repeat("1", 64)+`","file_sha256":"`), 1),
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := decodeProjectionSource(input)
			if err == nil || !strings.Contains(err.Error(), "duplicate key") {
				t.Fatalf("decodeProjectionSource() error = %v, want duplicate key", err)
			}
		})
	}
}

func TestSafeRepositoryPath(t *testing.T) {
	root := t.TempDir()
	if _, err := safeRepositoryPath(root, "survey/infoschem/evidence/managed/capture.json"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../capture.json", "/tmp/capture.json", `survey\capture.json`} {
		if _, err := safeRepositoryPath(root, path); err == nil {
			t.Fatalf("safeRepositoryPath(%q) error = nil", path)
		}
	}
}

func fixtureProjectionSource() projectionSource {
	return projectionSource{
		SchemaVersion: "v0alpha1",
		Mode:          "managed_live_primary",
		SelectedObservation: selectedObservation{
			Path:       "survey/infoschem/evidence/managed/20260825T083012Z-0123456789ab.json",
			FileSHA256: strings.Repeat("1", 64),
		},
		RollingAdvertisementPolicy: "runtime_filtered",
		Documentation: documentation{
			URL:         "https://cloud.google.com/spanner/docs/information-schema",
			LastUpdated: "2026-08-15",
		},
		DocumentationOnlyAbsent: []documentationOnlyColumn{{
			TableName:  "EXAMPLE",
			ColumnName: "DOCUMENTED_ONLY",
			RawType:    "STRING",
		}},
		ProjectionOverrides: []projectionOverride{{
			TableName:     "EXAMPLE",
			ColumnName:    "ROLLING_PROTO",
			ProjectedType: "STRING(MAX)",
		}},
	}
}

func fixtureSurveyExport() surveyExport {
	return surveyExport{
		Registry: []surveyTable{{
			Schema: "INFORMATION_SCHEMA",
			Name:   "EXAMPLE",
			Columns: []surveyColumn{
				{Name: "STABLE", SpannerType: "STRING(MAX)", OrdinalPosition: 1},
				{Name: "ROLLING_PROTO", SpannerType: "PROTO<example.Type>", OrdinalPosition: 2, Rolling: true},
			},
		}},
		Capture: captureDocument{
			SchemaVersion:        "v0alpha1",
			Catalog:              "INFORMATION_SCHEMA",
			Dialect:              "googlesql",
			Target:               captureTarget{Kind: "managed", ObservationScope: "single_database"},
			ObservedAt:           "2026-08-25T08:30:12Z",
			ProducerSourceSHA256: strings.Repeat("2", 64),
			InvocationSHA256:     strings.Repeat("3", 64),
			SurfaceSHA256:        strings.Repeat("4", 64),
			Columns: []captureColumn{
				{TableName: "EXAMPLE", ColumnName: "STABLE", SpannerType: "STRING(MAX)", OrdinalPosition: 1},
			},
		},
	}
}
