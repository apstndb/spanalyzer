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

func TestCatalogInputPathsIncludesEveryDeclaredLocalEvidence(t *testing.T) {
	source := filepath.Join("..", "..", "plancontract", "planvocab", "catalog_source.json")
	doc, err := readSource(source)
	if err != nil {
		t.Fatal(err)
	}
	inputs := catalogInputPaths("plancontract/planvocab/catalog_source.json", doc.Info.LocalEvidence)
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		seen[input] = struct{}{}
	}
	for _, evidence := range doc.Info.LocalEvidence {
		if _, ok := seen[evidence]; !ok {
			t.Errorf("declared local evidence %q is absent from catalog inputs", evidence)
		}
	}
}
