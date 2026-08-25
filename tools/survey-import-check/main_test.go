package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidatesRepositoryMapping(t *testing.T) {
	if err := run([]string{"--repo-root", "../.."}); err != nil {
		t.Fatal(err)
	}
}

func TestReadProvenanceRejectsUnknownField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "provenance.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"v0alpha1","unknown":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := readProvenance(path)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("readProvenance() error = %v, want unknown-field failure", err)
	}
}

func TestValidateProvenanceRejectsPublishedHistory(t *testing.T) {
	document, err := readProvenance("../../survey/import-provenance.json")
	if err != nil {
		t.Fatal(err)
	}
	document.Source.HistoryPublished = true
	if err := validateProvenance(document); err == nil || !strings.Contains(err.Error(), "unpublished") {
		t.Fatalf("validateProvenance() error = %v, want published-history failure", err)
	}
}
