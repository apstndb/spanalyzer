package main

import (
	"bytes"
	"encoding/json"
	"io"
	"slices"
	"strings"
	"testing"
)

func TestBuiltInCaseRegistry(t *testing.T) {
	seen := make(map[string]bool, len(builtInCaseSpecs))
	names := make([]string, 0, len(builtInCaseSpecs))
	for _, spec := range builtInCaseSpecs {
		if spec.Name == "" {
			t.Fatal("built-in case has an empty name")
		}
		if seen[spec.Name] {
			t.Fatalf("duplicate built-in case name %q", spec.Name)
		}
		seen[spec.Name] = true
		names = append(names, spec.Name)
		if strings.TrimSpace(spec.Description) == "" {
			t.Errorf("built-in case %q has no description", spec.Name)
		}
		if spec.Queries == nil {
			t.Errorf("built-in case %q has no query provider", spec.Name)
			continue
		}
		if queries := spec.Queries(); len(queries) == 0 {
			t.Errorf("built-in case %q has no queries", spec.Name)
		}
		if spec.DDLs == nil {
			t.Errorf("built-in case %q has no DDL provider", spec.Name)
			continue
		}
		if _, err := spec.DDLs(); err != nil {
			t.Errorf("built-in case %q DDL provider: %v", spec.Name, err)
		}
	}

	if got, want := builtInCaseNames(), strings.Join(names, ", "); got != want {
		t.Errorf("builtInCaseNames() = %q, want %q", got, want)
	}
}

func TestListBuiltInCasesText(t *testing.T) {
	var stdout bytes.Buffer
	if err := run([]string{"--list-cases"}, &stdout); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if got, want := len(lines), len(builtInCaseSpecs)+1; got != want {
		t.Fatalf("list line count = %d, want %d", got, want)
	}
	if got, want := lines[0], "NAME\tQUERIES\tDESCRIPTION"; got != want {
		t.Errorf("list heading = %q, want %q", got, want)
	}
	if got, want := lines[1], "all\t2\ttwo representative hash-join probes (not every built-in case)"; got != want {
		t.Errorf("first case row = %q, want %q", got, want)
	}
}

func TestListBuiltInCasesJSON(t *testing.T) {
	var stdout bytes.Buffer
	if err := run([]string{"--list-cases", "--list-cases-format", "json"}, &stdout); err != nil {
		t.Fatal(err)
	}
	var summaries []builtInCaseSummary
	if err := json.Unmarshal(stdout.Bytes(), &summaries); err != nil {
		t.Fatal(err)
	}
	if got, want := len(summaries), len(builtInCaseSpecs); got != want {
		t.Fatalf("JSON case count = %d, want %d", got, want)
	}
	proto := summaries[0]
	for _, summary := range summaries {
		if summary.Name == "google_sql_proto_surface" {
			proto = summary
			break
		}
	}
	if proto.Name != "google_sql_proto_surface" || !proto.RequiresProtoDescriptors {
		t.Errorf("proto case summary = %#v", proto)
	}
}

func TestListBuiltInCasesRejectsUnknownFormat(t *testing.T) {
	err := run([]string{"--list-cases", "--list-cases-format", "yaml"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "use text or json") {
		t.Fatalf("run() error = %v, want unsupported list format", err)
	}
}

func TestBuiltInCaseLookupIsNormalizedAndReturnsIndependentQuerySlices(t *testing.T) {
	spec, ok := lookupBuiltInCase("  DoCs  ")
	if !ok {
		t.Fatal("lookupBuiltInCase() did not normalize the case name")
	}
	if spec.Name != "docs" {
		t.Fatalf("lookupBuiltInCase() name = %q, want docs", spec.Name)
	}

	first := spec.Queries()
	second := spec.Queries()
	if len(first) == 0 || len(second) == 0 {
		t.Fatal("docs query provider returned no queries")
	}
	originalLabel := second[0].Label
	first[0].Label = "mutated"
	if got := spec.Queries()[0].Label; got != originalLabel {
		t.Fatalf("query provider reused a mutable result slice: label = %q, want %q", got, originalLabel)
	}
}

func TestAllBuiltInCaseRemainsTwoRepresentativeJoins(t *testing.T) {
	queries, err := loadQueries("all", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	labels := make([]string, 0, len(queries))
	for _, query := range queries {
		labels = append(labels, query.Label)
	}
	if want := []string{"PUSH_BROADCAST_HASH_JOIN", "HASH_JOIN"}; !slices.Equal(labels, want) {
		t.Fatalf("loadQueries(\"all\") labels = %v, want %v", labels, want)
	}
}

func TestCustomSQLPreservesDefaultDDLForUnknownCaseName(t *testing.T) {
	queries, err := loadQueries("custom", []string{"SELECT 1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 1 || queries[0].SQL != "SELECT 1" {
		t.Fatalf("custom queries = %#v", queries)
	}

	ddls, err := loadDDLs("custom", nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{singersDDL, albumsDDL}; !slices.Equal(ddls, want) {
		t.Fatalf("custom default DDLs = %#v, want built-in Singers/Albums DDL", ddls)
	}
}
