package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunChecksSingleAndArrayEnvelopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single fixture envelope",
			input: `{"plan":{"planNodes":[{"index":1,"displayName":"Scan"}]}}`,
			want:  "plans: 1\nquery_errors: 0\nfindings: 0\n",
		},
		{
			name: "live envelope array",
			input: `[
  {"query_label":"known", "plan":{"plan_nodes":[{"index":1,"kind":"RELATIONAL","display_name":"Scan"}]}},
  {"query_label":"expected_failure", "error":"unsupported query"}
]`,
			want: "plans: 1\nquery_errors: 1\nfindings: 0\nexpectations: 0\nexpectation_failures: 0\nquery_error: stdin:expected_failure\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			args := []string{"--allow-query-errors"}
			if err := run(args, strings.NewReader(tt.input), &output); err != nil {
				t.Fatalf("run() error = %v\n%s", err, output.String())
			}
			if got := output.String(); !strings.Contains(got, tt.want) {
				t.Fatalf("run() output = %q, want substring %q", got, tt.want)
			}
		})
	}
}

func TestRunReportsUnknownVocabulary(t *testing.T) {
	t.Parallel()

	input := `{"query_label":"future","plan":{"planNodes":[{"index":1,"kind":"RELATIONAL","displayName":"Future Exchange"}]}}`
	var output bytes.Buffer
	err := run(nil, strings.NewReader(input), &output)
	if err == nil || !strings.Contains(err.Error(), "1 finding") {
		t.Fatalf("run() error = %v, want one-finding error", err)
	}
	if got := output.String(); !strings.Contains(got, `operator="Future Exchange" reasons=operator_family_unknown`) {
		t.Fatalf("run() output = %q, want redacted finding", got)
	}
}

func TestRunRejectsQueryErrorsByDefault(t *testing.T) {
	t.Parallel()

	input := `[{"query_label":"bad","error":"private backend detail"}]`
	var output bytes.Buffer
	err := run(nil, strings.NewReader(input), &output)
	if err == nil || !strings.Contains(err.Error(), "1 query error") {
		t.Fatalf("run() error = %v, want query-error failure", err)
	}
	if strings.Contains(output.String(), "private backend detail") {
		t.Fatalf("run() output exposed source error: %q", output.String())
	}
}

func TestRunJSONReport(t *testing.T) {
	t.Parallel()

	input := `{"plan":{"planNodes":[{"index":1,"displayName":"Scan"}]}}`
	var output bytes.Buffer
	if err := run([]string{"--format", "json"}, strings.NewReader(input), &output); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if got := output.String(); !strings.Contains(got, `"plans": 1`) || !strings.Contains(got, `"findings": []`) {
		t.Fatalf("run() JSON output = %q", got)
	}
}

func TestRunChecksPositiveExpectations(t *testing.T) {
	t.Parallel()

	manifest := `{
  "version": "v0alpha1",
  "queries": [{
    "label": "scan",
    "patterns": [{
      "family": "scan",
      "metadata": [{"key": "scan_method", "value": "Batch"}],
      "child_links": [{"kind": "SCALAR", "type": "", "variable": "present", "min_count": 2}]
    }]
  }]
}`
	path := filepath.Join(t.TempDir(), "expectations.json")
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	input := `[{"query_label":"scan","plan":{"planNodes":[
  {"index":1,"kind":"RELATIONAL","displayName":"Scan","metadata":{"scan_method":"Batch"},"childLinks":[{"childIndex":2,"variable":"a"},{"childIndex":3,"variable":"b"}]},
  {"index":2,"kind":"SCALAR","displayName":"Reference"},
  {"index":3,"kind":"SCALAR","displayName":"Reference"}
]}}]`
	var output bytes.Buffer
	if err := run([]string{"--expect", path}, strings.NewReader(input), &output); err != nil {
		t.Fatalf("run() error = %v\n%s", err, output.String())
	}
	if got := output.String(); !strings.Contains(got, "expectations: 1\nexpectation_failures: 0") {
		t.Fatalf("run() output = %q, want passing expectation count", got)
	}
}

func TestRunReportsPatternExpectationOrdinal(t *testing.T) {
	t.Parallel()

	manifest := `{"version":"v0alpha1","queries":[{"label":"scan","patterns":[{"display_name":"Scan","metadata":[{"key":"scan_method","value":"Batch"}]}]}]}`
	path := filepath.Join(t.TempDir(), "expectations.json")
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	input := `[{"query_label":"scan","plan":{"planNodes":[{"index":1,"kind":"RELATIONAL","displayName":"Scan","metadata":{"scan_method":"Row"}}]}}]`
	var output bytes.Buffer
	err := run([]string{"--expect", path, "--format", "json"}, strings.NewReader(input), &output)
	if err == nil || !strings.Contains(err.Error(), "1 expectation failure") {
		t.Fatalf("run() error = %v, want expectation failure", err)
	}
	if got := output.String(); !strings.Contains(got, `"pattern": 1`) || !strings.Contains(got, `metadata key \"scan_method\" did not match`) {
		t.Fatalf("run() output = %q, want 1-based pattern failure", got)
	}
}

func TestRunReportsMissingExpectationLabelWithoutValues(t *testing.T) {
	t.Parallel()

	manifest := `{"version":"v0alpha1","queries":[{"label":"private_query","patterns":[{"display_name":"Scan"}]}]}`
	path := filepath.Join(t.TempDir(), "expectations.json")
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	input := `[{"query_label":"other","plan":{"planNodes":[{"index":1,"displayName":"Scan"}]}}]`
	var output bytes.Buffer
	err := run([]string{"--expect", path}, strings.NewReader(input), &output)
	if err == nil || !strings.Contains(err.Error(), "1 expectation failure") {
		t.Fatalf("run() error = %v, want expectation failure", err)
	}
	if got := output.String(); strings.Contains(got, "private backend") || strings.Contains(got, "pattern=0") || !strings.Contains(got, "query label was not present") {
		t.Fatalf("run() output = %q, want safe missing-label diagnostic", got)
	}

	output.Reset()
	err = run([]string{"--expect", path, "--format", "json"}, strings.NewReader(input), &output)
	if err == nil || !strings.Contains(err.Error(), "1 expectation failure") {
		t.Fatalf("run(json) error = %v, want expectation failure", err)
	}
	if got := output.String(); strings.Contains(got, `"pattern"`) || !strings.Contains(got, `"reason": "query label was not present in the input"`) {
		t.Fatalf("run(json) output = %q, want label-level failure without a pattern ordinal", got)
	}
}
