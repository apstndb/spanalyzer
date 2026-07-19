package main

import (
	"path/filepath"
	"testing"
)

func TestGeneratedCatalogIsCurrent(t *testing.T) {
	if err := run([]string{"--repo-root", "../..", "--check"}); err != nil {
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
		filepath.Join(root, "tools", "planvocab-gen"),
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
