package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type manifestCoverage string

const (
	manifestCoverageExact  manifestCoverage = "exact"
	manifestCoverageSubset manifestCoverage = "subset"
)

type expectationManifest struct {
	Version string `json:"version"`
	Queries []struct {
		Label    string            `json:"label"`
		Patterns []json.RawMessage `json:"patterns"`
	} `json:"queries"`
	ExpectedQueryErrors []struct {
		Label    string `json:"label"`
		Contains string `json:"contains"`
	} `json:"expected_query_errors"`
}

type expectationManifestContract struct {
	Selector     string
	Manifest     string
	Coverage     manifestCoverage
	Plans        int
	Errors       int
	PatternCount int
}

var expectationManifestContracts = []expectationManifestContract{
	{Selector: "aggregate_functions", Manifest: "aggregate_function_expectations.json", Coverage: manifestCoverageExact, Plans: 29, Errors: 2, PatternCount: 62},
	{Selector: "ai_plan", Manifest: "ai_plan_expectations.json", Coverage: manifestCoverageExact, Plans: 6, PatternCount: 16},
	{Selector: "condition_boundaries", Manifest: "condition_boundary_expectations.json", Coverage: manifestCoverageExact, Plans: 38, PatternCount: 62},
	{Selector: "dml", Manifest: "dml_expectations.json", Coverage: manifestCoverageExact, Plans: 28, Errors: 1, PatternCount: 28},
	{Selector: "full_text_search", Manifest: "facet_search_expectations.json", Coverage: manifestCoverageSubset, Plans: 4, PatternCount: 14},
	{Selector: "factorized_mode", Manifest: "factorized_mode_expectations.json", Coverage: manifestCoverageExact, Plans: 10, Errors: 5, PatternCount: 19},
	{Selector: "full_text_search", Manifest: "full_text_residual_expectations.json", Coverage: manifestCoverageSubset, Plans: 9, PatternCount: 26},
	{Selector: "google_sql_proto_surface", Manifest: "google_sql_proto_surface_expectations.json", Coverage: manifestCoverageExact, Plans: 11, Errors: 7, PatternCount: 28},
	{Selector: "google_sql_surface", Manifest: "google_sql_surface_expectations.json", Coverage: manifestCoverageExact, Plans: 35, Errors: 22, PatternCount: 74},
	{Selector: "gql_hint_surface", Manifest: "gql_hint_surface_expectations.json", Coverage: manifestCoverageExact, Plans: 47, Errors: 3, PatternCount: 93},
	{Selector: "gql_surface", Manifest: "gql_surface_expectations.json", Coverage: manifestCoverageExact, Plans: 100, Errors: 25, PatternCount: 208},
	{Selector: "hint_position_combinations", Manifest: "hint_position_combination_expectations.json", Coverage: manifestCoverageSubset, Plans: 9, PatternCount: 9},
	{Selector: "ngram_search", Manifest: "ngram_search_expectations.json", Coverage: manifestCoverageExact, Plans: 12, PatternCount: 24},
	{Selector: "planvocab_inference", Manifest: "planvocab_inference_expectations.json", Coverage: manifestCoverageExact, Plans: 17, PatternCount: 17},
	{Selector: "pipe_surface", Manifest: "pipe_surface_expectations.json", Coverage: manifestCoverageExact, Plans: 15, PatternCount: 26},
	{Selector: "optimizer_v9", Manifest: "optimizer_v9_expectations.json", Coverage: manifestCoverageExact, Plans: 6, Errors: 1, PatternCount: 9},
	{Selector: "remote_function", Manifest: "remote_function_expectations.json", Coverage: manifestCoverageExact, Plans: 2, PatternCount: 6},
	{Selector: "rewriter_surface", Manifest: "rewriter_surface_expectations.json", Coverage: manifestCoverageExact, Plans: 18, Errors: 15, PatternCount: 52},
	{Selector: "search_graph", Manifest: "search_graph_expectations.json", Coverage: manifestCoverageSubset, Plans: 3, PatternCount: 14},
	{Selector: "set_operation_distinct", Manifest: "set_operation_distinct_expectations.json", Coverage: manifestCoverageSubset, Plans: 47, Errors: 17, PatternCount: 77},
	{Selector: "statement_surface", Manifest: "statement_surface_expectations.json", Coverage: manifestCoverageExact, Plans: 3, Errors: 7, PatternCount: 9},
}

func TestExpectationManifestContracts(t *testing.T) {
	for _, contract := range expectationManifestContracts {
		t.Run(contract.Manifest, func(t *testing.T) {
			assertExpectationManifestContract(t, contract)
		})
	}
}

func TestExpectationManifestContractInventory(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "*_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	contractNames := make(map[string]bool, len(expectationManifestContracts))
	for _, contract := range expectationManifestContracts {
		if contractNames[contract.Manifest] {
			t.Errorf("duplicate manifest contract %q", contract.Manifest)
		}
		contractNames[contract.Manifest] = true
	}
	for _, path := range paths {
		name := filepath.Base(path)
		if !contractNames[name] {
			t.Errorf("expectation manifest %q has no contract", name)
		}
		delete(contractNames, name)
	}
	for name := range contractNames {
		t.Errorf("manifest contract %q has no testdata file", name)
	}
}

func assertExpectationManifestContract(t *testing.T, contract expectationManifestContract) {
	t.Helper()
	queries, err := loadQueries(contract.Selector, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	selectorLabels := make(map[string]bool, len(queries))
	for _, query := range queries {
		if selectorLabels[query.Label] {
			t.Errorf("selector %q has duplicate label %q", contract.Selector, query.Label)
		}
		selectorLabels[query.Label] = true
	}

	manifest := loadExpectationManifest(t, contract.Manifest)
	if manifest.Version != "v0alpha1" {
		t.Errorf("manifest version = %q, want v0alpha1", manifest.Version)
	}
	if got := len(manifest.Queries); got != contract.Plans {
		t.Errorf("plan expectations = %d, want %d", got, contract.Plans)
	}
	if got := len(manifest.ExpectedQueryErrors); got != contract.Errors {
		t.Errorf("error expectations = %d, want %d", got, contract.Errors)
	}

	manifestLabels := make(map[string]bool, contract.Plans+contract.Errors)
	patternCount := 0
	for _, expectation := range manifest.Queries {
		assertManifestLabel(t, contract, selectorLabels, manifestLabels, expectation.Label)
		if len(expectation.Patterns) == 0 {
			t.Errorf("plan expectation %q has no patterns", expectation.Label)
		}
		patternCount += len(expectation.Patterns)
	}
	for _, expectation := range manifest.ExpectedQueryErrors {
		assertManifestLabel(t, contract, selectorLabels, manifestLabels, expectation.Label)
		if strings.TrimSpace(expectation.Contains) == "" {
			t.Errorf("error expectation %q has no matching text", expectation.Label)
		}
	}
	if patternCount != contract.PatternCount {
		t.Errorf("operator patterns = %d, want %d", patternCount, contract.PatternCount)
	}

	switch contract.Coverage {
	case manifestCoverageExact:
		for label := range selectorLabels {
			if !manifestLabels[label] {
				t.Errorf("selector label %q is absent from exact manifest", label)
			}
		}
	case manifestCoverageSubset:
		if len(manifestLabels) == 0 {
			t.Error("subset manifest is empty")
		}
		if len(manifestLabels) >= len(selectorLabels) {
			t.Errorf("subset manifest covers %d labels from a %d-label selector; use exact coverage when all labels are retained", len(manifestLabels), len(selectorLabels))
		}
	default:
		t.Fatalf("unknown manifest coverage policy %q", contract.Coverage)
	}
}

func assertManifestLabel(t *testing.T, contract expectationManifestContract, selectorLabels, manifestLabels map[string]bool, label string) {
	t.Helper()
	if !selectorLabels[label] {
		t.Errorf("manifest label %q is absent from selector %q", label, contract.Selector)
	}
	if manifestLabels[label] {
		t.Errorf("manifest label %q is duplicated", label)
	}
	manifestLabels[label] = true
}

func loadExpectationManifest(t *testing.T, name string) expectationManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var manifest expectationManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}
