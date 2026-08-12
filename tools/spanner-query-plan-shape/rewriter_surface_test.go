package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRewriterSurfaceCatalogIsComplete(t *testing.T) {
	wantNames := []string{
		"REWRITE_AGGREGATION_THRESHOLD",
		"REWRITE_ANONYMIZATION",
		"REWRITE_BUILTIN_FUNCTION_INLINER",
		"REWRITE_FLATTEN",
		"REWRITE_GENERALIZED_QUERY_STMT",
		"REWRITE_GROUPING_SET",
		"REWRITE_INLINE_SQL_FUNCTIONS",
		"REWRITE_INLINE_SQL_TVFS",
		"REWRITE_INLINE_SQL_UDAS",
		"REWRITE_INLINE_SQL_VIEWS",
		"REWRITE_INSERT_DML_VALUES",
		"REWRITE_IS_FIRST_IS_LAST_FUNCTION",
		"REWRITE_LIKE_ANY_ALL",
		"REWRITE_MATCH_RECOGNIZE_FUNCTION",
		"REWRITE_MEASURE_TYPE",
		"REWRITE_MULTIWAY_UNNEST",
		"REWRITE_NULLIFERROR_FUNCTION",
		"REWRITE_ORDER_BY_AND_LIMIT_IN_AGGREGATE",
		"REWRITE_PIPE_ASSERT",
		"REWRITE_PIPE_DESCRIBE",
		"REWRITE_PIPE_IF",
		"REWRITE_PIVOT",
		"REWRITE_PROTO_MAP_FNS",
		"REWRITE_QUANTIFIED_COMPARISONS",
		"REWRITE_ROW_TYPE",
		"REWRITE_SUBPIPELINE_STMT",
		"REWRITE_TUMBLE_FUNCTION",
		"REWRITE_TYPEOF_FUNCTION",
		"REWRITE_UNPIVOT",
		"REWRITE_UPDATE_CONSTRUCTOR",
		"REWRITE_VARIADIC_FUNCTION_SIGNATURE_EXPANDER",
		"REWRITE_WITH_EXPR",
	}
	gotNames := make([]string, 0, len(registeredRewriterCoverage))
	allLabels := make(map[string]struct{})
	for _, queries := range [][]queryCase{
		rewriterSurfaceQueries,
		googleSQLSurfaceQueries,
		dmlQueries,
		gqlSurfaceQueries,
	} {
		for _, query := range queries {
			allLabels[query.Label] = struct{}{}
		}
	}
	for _, entry := range registeredRewriterCoverage {
		gotNames = append(gotNames, entry.Name)
		switch entry.Retention {
		case rewriterDirectPlan, rewriterDirectError, rewriterExistingPlan, rewriterExistingError:
			if len(entry.EvidenceLabels) == 0 {
				t.Errorf("%s has retention %q without evidence labels", entry.Name, entry.Retention)
			}
			for _, label := range entry.EvidenceLabels {
				if _, ok := allLabels[label]; !ok {
					t.Errorf("%s references unknown evidence label %q", entry.Name, label)
				}
			}
		case rewriterNotExposed:
			if len(entry.EvidenceLabels) != 0 {
				t.Errorf("%s is not exposed but has evidence labels %v", entry.Name, entry.EvidenceLabels)
			}
			if strings.TrimSpace(entry.Note) == "" {
				t.Errorf("%s is not exposed without a rationale", entry.Name)
			}
		default:
			t.Errorf("%s has unknown retention %q", entry.Name, entry.Retention)
		}
	}
	if !slices.Equal(gotNames, wantNames) {
		t.Errorf("registered rewriters = %v, want %v", gotNames, wantNames)
	}
}

func TestLoadQueriesRewriterSurfaceMatchesManifest(t *testing.T) {
	queries, err := loadQueries("rewriter_surface", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "rewriter_surface", err)
	}
	if got, want := len(queries), 32; got != want {
		t.Fatalf("rewriter surface query count = %d, want %d", got, want)
	}
	seen := make(map[string]struct{}, len(queries))
	for _, query := range queries {
		if !strings.HasPrefix(query.Label, "rewriter-surface/accepted/") &&
			!strings.HasPrefix(query.Label, "rewriter-surface/unsupported/") {
			t.Errorf("rewriter surface label %q has wrong prefix", query.Label)
		}
		if _, duplicate := seen[query.Label]; duplicate {
			t.Errorf("duplicate rewriter surface label %q", query.Label)
		}
		seen[query.Label] = struct{}{}
	}

	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "rewriter_surface_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
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
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Version, "v0alpha1"; got != want {
		t.Errorf("rewriter surface expectation version = %q, want %q", got, want)
	}
	if got, want := len(manifest.Queries), 18; got != want {
		t.Errorf("rewriter surface positive expectations = %d, want %d", got, want)
	}
	if got, want := len(manifest.ExpectedQueryErrors), 14; got != want {
		t.Errorf("rewriter surface error expectations = %d, want %d", got, want)
	}
	patternCount := 0
	manifestLabels := make(map[string]struct{}, len(queries))
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("rewriter surface expectation label %q is absent from the built-in case", expectation.Label)
		}
		if len(expectation.Patterns) == 0 {
			t.Errorf("rewriter surface expectation label %q has no operator patterns", expectation.Label)
		}
		if _, duplicate := manifestLabels[expectation.Label]; duplicate {
			t.Errorf("duplicate rewriter surface manifest label %q", expectation.Label)
		}
		manifestLabels[expectation.Label] = struct{}{}
		patternCount += len(expectation.Patterns)
	}
	if got, want := patternCount, 52; got != want {
		t.Errorf("rewriter surface operator patterns = %d, want %d", got, want)
	}
	for _, expectation := range manifest.ExpectedQueryErrors {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("rewriter surface expected-error label %q is absent from the built-in case", expectation.Label)
		}
		if expectation.Contains == "" {
			t.Errorf("rewriter surface expected-error label %q has no matching text", expectation.Label)
		}
		if _, duplicate := manifestLabels[expectation.Label]; duplicate {
			t.Errorf("duplicate rewriter surface manifest label %q", expectation.Label)
		}
		manifestLabels[expectation.Label] = struct{}{}
	}
	if len(manifestLabels) != len(seen) {
		t.Errorf("rewriter surface manifest labels = %d, query labels = %d", len(manifestLabels), len(seen))
	}

	ddls, err := loadDDLs("rewriter_surface", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "rewriter_surface", err)
	}
	if len(ddls) == 0 {
		t.Fatal("rewriter surface DDL is empty")
	}
}
