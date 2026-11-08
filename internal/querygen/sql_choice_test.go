package querygen

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func sqlChoiceConfig(t *testing.T) (QueryCodegenConfig, string) {
	t.Helper()
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `CREATE TABLE Events (
Id INT64 NOT NULL, Shard INT64, CreatedAt TIMESTAMP,
Active STRING(MAX), Pending STRING(MAX)
) PRIMARY KEY (Id)`)
	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
emit:
  spanner:
    mutations: true
catalogs:
- name: app
  kind: spanner
  ddl: schema.sql
queries:
- name: FindEvents
  catalog: app
  kind: sql
  sql: |
    SELECT Id FROM Events WHERE Shard = @shard
    /*?choice:bounds*/ AND CreatedAt BETWEEN @start AND @end /*?end*/
    /*?choice:flag*/ AND Active = "1" /*?end*/
  params:
  - {name: shard, type: INT64}
  - {name: start, type: TIMESTAMP}
  - {name: end, type: TIMESTAMP}
  - name: bounds
    type: STRING
    optional: sql_choice
    default: both
    choices:
      both: " AND CreatedAt BETWEEN @start AND @end"
      start: " AND CreatedAt >= @start"
      end: " AND CreatedAt <= @end"
  - name: flag
    type: STRING
    optional: sql_choice
    default: active
    choices:
      active: ' AND Active = "1"'
      pending: ' AND Pending = "1"'
  result:
    cardinality: many
    struct: EventRow
`))
	if err != nil {
		t.Fatal(err)
	}
	return config, dir
}

func TestSQLChoicePlanGenerationAndReportBindings(t *testing.T) {
	config, dir := sqlChoiceConfig(t)
	plan, err := BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatal(err)
	}
	q := plan.Queries[0]
	if len(q.Variants) != 6 || len(q.Fields) != 1 || q.Fields[0].Name != "Id" {
		t.Fatalf("plan = %+v", q)
	}
	expanded := ExpandPlanReportQueries(plan.Queries)
	if len(expanded) != 6 {
		t.Fatalf("expanded targets = %d", len(expanded))
	}
	for _, target := range expanded {
		var names []string
		for _, p := range target.Params {
			names = append(names, p.Name)
		}
		choice := target.ReportVariant.ChoiceAssignments["bounds"]
		if !slices.Contains(names, "shard") || (slices.Contains(names, "start") != (choice != "end")) || (slices.Contains(names, "end") != (choice != "start")) {
			t.Fatalf("choice %s report bindings = %v", choice, names)
		}
	}
	code, err := GenerateQueryCode(config, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"time"`, "Start", "time.Time", "Bounds", "func BuildFindEventsSQL(", `delete(args, "start")`, `delete(args, "end")`} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q", want)
		}
	}
	config.Queries[0].Params[3].Choices["both"] = " AND CreatedAt >= @missing"
	report := BuildQueryCodegenReportPlan(config, dir)
	if len(report.Queries) != 1 || !strings.Contains(report.Queries[0].PlanningError, "undeclared bind parameter @missing") {
		t.Fatalf("invalid closed branch was not retained as report error: %+v", report.Queries)
	}
}

func TestSQLChoiceConfigBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, kind, want string
		change           func(*QueryCodegenParam)
	}{
		{name: "empty choices", want: "requires choices", change: func(p *QueryCodegenParam) { p.Choices = nil }},
		{name: "empty default", want: "requires default", change: func(p *QueryCodegenParam) { p.Default = "" }},
		{name: "unknown default", want: "must be one of choices", change: func(p *QueryCodegenParam) { p.Default = "missing" }},
		{name: "unsafe key", want: "choice key", change: func(p *QueryCodegenParam) { p.Choices["bad+key"] = " AND TRUE" }},
		{name: "table shorthand", kind: "table", want: "sql_choice requires kind: sql"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, dir := sqlChoiceConfig(t)
			if tc.change != nil {
				tc.change(&config.Queries[0].Params[3])
			}
			if tc.kind != "" {
				q := &config.Queries[0]
				q.Kind, q.Table, q.SQL = tc.kind, "Events", ""
				q.Params = []QueryCodegenParam{q.Params[3]}
			}
			if _, err := BuildQueryCodegenPlan(config, dir); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}
