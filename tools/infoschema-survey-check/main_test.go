package main

import (
	"os"
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

func TestCompareManifestToSurveyFixture(t *testing.T) {
	survey := readSurveyFixture(t)
	doc := fixtureManifest(t, survey)
	if err := compareManifestToSurvey(&doc, survey); err != nil {
		t.Fatal(err)
	}
}

func TestCompareManifestToSurveyRejectsHiddenLiveColumn(t *testing.T) {
	survey := readSurveyFixture(t)
	doc := fixtureManifest(t, survey)
	column := &doc.Tables[0].Columns[0]
	column.EvidenceStatus = "docs_only_absent"
	column.Project = false
	column.Ordinal = 0
	doc.ContentSHA256 = mustHashJSON(t, doc.Tables)
	err := compareManifestToSurvey(&doc, survey)
	if err == nil || !strings.Contains(err.Error(), "docs_only_absent") {
		t.Fatalf("compareManifestToSurvey() error = %v, want hidden live-column failure", err)
	}
}

func TestRunRejectsExplicitInvalidSurveyRoot(t *testing.T) {
	err := run([]string{"--repo-root", "../..", "--survey-root", filepath.Join(t.TempDir(), "missing")})
	if err == nil || !strings.Contains(err.Error(), "invalid --survey-root") {
		t.Fatalf("run() error = %v, want invalid --survey-root", err)
	}
}

func readSurveyFixture(t *testing.T) []surveyTable {
	t.Helper()
	data, err := os.ReadFile("testdata/survey-export.json")
	if err != nil {
		t.Fatal(err)
	}
	survey, err := decodeSurveyExport(data)
	if err != nil {
		t.Fatal(err)
	}
	return survey
}

func fixtureManifest(t *testing.T, survey []surveyTable) manifest {
	t.Helper()
	doc := manifest{
		SchemaVersion: "v0alpha1",
		Source: source{
			Commit:       "0123456789012345678901234567890123456789",
			ExportSHA256: mustHashJSON(t, survey),
		},
		Tables: []table{{
			Name: "EXAMPLE_TABLE",
			Columns: []column{
				{Name: "LIVE_COLUMN", Ordinal: 1, RawType: "STRING(MAX)", EvidenceStatus: "live_observed", Project: true},
				{Name: "ROLLING_COLUMN", Ordinal: 2, RawType: "BOOL", EvidenceStatus: "rolling", Project: true},
				{Name: "DOCUMENTED_COLUMN", RawType: "STRING", EvidenceStatus: "docs_only_absent", Project: false},
			},
		}},
	}
	doc.ContentSHA256 = mustHashJSON(t, doc.Tables)
	return doc
}

func mustHashJSON(t *testing.T, value any) string {
	t.Helper()
	hash, err := hashJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}
