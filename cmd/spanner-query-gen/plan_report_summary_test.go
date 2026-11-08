package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanalyzer/plancontract"
	"github.com/apstndb/spanemuboost"
	"github.com/goccy/go-yaml"
)

func TestPlanReportSummaryPreservesEvaluationWithoutRendering(t *testing.T) {
	config, dir, opts := feedbackReportFixture(t)
	model := &querygen.QueryCodegenPlan{Queries: []querygen.QueryCodegenPlanQuery{
		{Name: "List", Catalog: "good", Kind: "sql", SQL: "SELECT Id FROM T", Variants: []querygen.QueryCodegenPlanQueryVariant{{Label: "(none)", SQL: "SELECT Id FROM T", AbsentParams: []string{"id"}}}},
		{Name: "NoPlan", Catalog: "good", Kind: "sql", SQL: "no_plan"},
	}}
	open := func(context.Context, spanemuboost.RuntimeHandle, querygen.QueryCodegenSchema, string, *spanalyzer.ProtoDescriptorSet, planReportOptimizer) (planReportAnalyzer, error) {
		return &feedbackPlanAnalyzer{}, nil
	}
	full, err := buildPlanReportWithAnalyzer(t.Context(), config, model, dir, opts, nil, open)
	if err != nil {
		t.Fatal(err)
	}
	opts.Summary = true
	summary, err := buildPlanReportWithAnalyzer(t.Context(), config, model, dir, opts, nil, open)
	if err != nil {
		t.Fatal(err)
	}
	contracts := planContractsFile{Version: planContractFileVersionV1Alpha, Contracts: []planContract{
		{Name: "NoSort", Target: "query/List", Use: []string{"no_explicit_sort"}},
		{Name: "MissingPlan", Target: "query/NoPlan", Use: []string{"no_explicit_sort"}},
	}}
	for _, report := range []*planReport{&full, &summary} {
		if err := applyPlanContracts(report, contracts); err != nil {
			t.Fatal(err)
		}
		report.ContractEvalMode = planContractEvaluationModeCheck
	}
	if !reflect.DeepEqual(full.ContractEvaluations, summary.ContractEvaluations) || !reflect.DeepEqual(full.TargetSummary, summary.TargetSummary) || full.Queries[0].Plan == "" || summary.Queries[0].Plan != "" {
		t.Fatal("summary collection changed evaluation/target results or rendered a plan")
	}
	if summary.Queries[1].Status != "error" || !strings.Contains(summary.Queries[1].Error, "no plan nodes") {
		t.Fatalf("empty PLAN was accepted: %+v", summary.Queries[1])
	}
	if planReportContractCheckFailureCount(full) != planReportContractCheckFailureCount(summary) {
		t.Fatal("summary changed --check failure accounting")
	}
	before, err := json.Marshal(summary.ContractEvaluations)
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{"json", "yaml"} {
		t.Run(output, func(t *testing.T) {
			var buf bytes.Buffer
			if err := writePlanReport(&buf, output, summary); err != nil {
				t.Fatal(err)
			}
			data := buf.Bytes()
			if output == "yaml" {
				var err error
				data, err = yaml.YAMLToJSON(data)
				if err != nil {
					t.Fatal(err)
				}
			}
			var artifact map[string]interface{}
			if err := json.Unmarshal(data, &artifact); err != nil {
				t.Fatal(err)
			}
			if artifact["report_version"] != planReportSummaryVersionV1Alpha || artifact["format"] != nil || artifact["render_mode"] != nil || artifact["plan_source"].(map[string]interface{})["render_tool"] != nil {
				t.Fatalf("invalid summary identity: %s", data)
			}
			queries := artifact["queries"].([]interface{})
			for _, value := range queries {
				q := value.(map[string]interface{})
				for _, field := range []string{"plan", "sql", "normalized_operators", "operator_edges", "operator_families", "operator_family_counts"} {
					if _, ok := q[field]; ok {
						t.Fatalf("summary leaked %s", field)
					}
				}
			}
			okQuery := queries[0].(map[string]interface{})
			variant := okQuery["variant"].(map[string]interface{})
			if okQuery["target_id"] != full.Queries[0].TargetID || okQuery["sql_sha256"] != full.Queries[0].SQLSHA256 || variant["label"] != "(none)" || variant["sql"] != nil || variant["sql_sha256"] != nil {
				t.Fatal("summary lost identity or retained variant SQL")
			}
			if _, ok := okQuery["access_paths"].([]interface{}); !ok || queries[1].(map[string]interface{})["access_paths"] != nil {
				t.Fatal("success/error access-path presence is incorrect")
			}
			for _, value := range artifact["contract_evaluations"].([]interface{}) {
				if value.(map[string]interface{})["stability"].(map[string]interface{})["replayable_from_report"] != false {
					t.Fatal("summary claims offline contract replay")
				}
			}
		})
	}
	after, err := json.Marshal(summary.ContractEvaluations)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("summary serialization mutated full evaluation evidence")
	}
}

func TestPlanReportSummaryEmptyAccessPathsAndRejectedMarkdown(t *testing.T) {
	report := newPlanReportInvariantTestReport(t)
	report.Summary = true
	projected := summarizePlanReport(report)
	if paths, ok := projected.Queries[0].AccessPaths.([]plancontract.AccessPath); !ok || paths == nil {
		t.Fatalf("successful empty access_paths must be []: %#v", projected.Queries[0].AccessPaths)
	}
	if err := writePlanReport(&bytes.Buffer{}, "markdown", report); err == nil {
		t.Fatal("summary Markdown should fail")
	}
	err := run([]string{"plan-report", "--summary", "--config", "missing-config.yaml"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--summary requires --output") {
		t.Fatalf("summary format should fail before reading config or starting Omni: %v", err)
	}
}

func TestPlanReportCollectorSeparatesVersionAndStatisticsPinning(t *testing.T) {
	for _, tc := range []struct {
		version, stats, warnings string
		queryNotPinned           bool
	}{
		{"", "", "optimizer_not_pinned,statistics_package_not_pinned,query_optimizer_not_pinned", true},
		{"9", "", "statistics_package_not_pinned", false},
		{"", "auto_20260701_12_00_00UTC", "optimizer_not_pinned,query_optimizer_not_pinned", true},
		{"9", "auto_20260701_12_00_00UTC", "", false},
		{"latest_version", "latest", "optimizer_not_pinned,statistics_package_not_pinned,query_optimizer_not_pinned", true},
	} {
		t.Run(tc.version+"/"+tc.stats, func(t *testing.T) {
			config, dir, opts := feedbackReportFixture(t)
			opts.Optimizer = planReportOptimizerEnvironment{Version: tc.version, StatisticsPackage: tc.stats}
			open := func(context.Context, spanemuboost.RuntimeHandle, querygen.QueryCodegenSchema, string, *spanalyzer.ProtoDescriptorSet, planReportOptimizer) (planReportAnalyzer, error) {
				return &feedbackPlanAnalyzer{}, nil
			}
			model := &querygen.QueryCodegenPlan{Queries: []querygen.QueryCodegenPlanQuery{{Name: "List", Catalog: "good", Kind: "sql", SQL: "SELECT Id FROM T"}}}
			report, err := buildPlanReportWithAnalyzer(t.Context(), config, model, dir, opts, nil, open)
			if err != nil || report.Queries[0].OptimizerNotPinned != tc.queryNotPinned {
				t.Fatalf("collector pinning = %+v, %v", report, err)
			}
			if got := strings.Join(planReportOptimizerPinningWarnings(report), ","); got != tc.warnings {
				t.Fatalf("warnings = %s, want %s", got, tc.warnings)
			}
			if got := planReportOptimizerEnvironmentPinned(report.Optimizer.Requested); got != (tc.warnings == "") {
				t.Fatalf("strict gate = %t for %s", got, tc.warnings)
			}
		})
	}
}

func TestRunPlanReportSummaryWritesBeforeStrictPinningFailure(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestFile(t, configPath, `version: v1alpha
go:
  package: querydemo
catalogs:
- name: analytics
  kind: bigquery
queries:
- name: BigQueryOnly
  catalog: analytics
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: ValueRow
`)
	var output bytes.Buffer
	err := run([]string{"plan-report", "--summary", "--output", "json", "--config", configPath, "--optimizer-version", "9", "--require-optimizer-pinning"}, &output, &bytes.Buffer{})
	if err == nil || err.Error() != "optimizer pinning is required but not satisfied: statistics_package_not_pinned" {
		t.Fatalf("version-only strict pinning error = %v", err)
	}
	var artifact map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &artifact); err != nil {
		t.Fatal(err)
	}
	if artifact["report_version"] != planReportSummaryVersionV1Alpha || artifact["status"] != planReportStatusNoTargets {
		t.Fatalf("strict failure did not retain a summary artifact: %s", output.Bytes())
	}
}

func TestPlanReportSummarySchemaFileAndProjection(t *testing.T) {
	var output bytes.Buffer
	if err := runPlanReportSchema([]string{"--summary"}, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("..", "..", "schemas", "spanner-query-gen.plan-summary.v1alpha.schema.json")
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, output.Bytes()) {
		t.Fatalf("summary schema is missing or stale; regenerate with plan-report-schema --summary --out: %v", err)
	}
	schema := planReportSummaryJSONSchema()
	properties := schema["$defs"].(map[string]interface{})["query"].(map[string]interface{})["properties"].(map[string]interface{})
	data, err := json.Marshal(summarizePlanReport(newPlanReportInvariantTestReport(t)).Queries[0])
	if err != nil {
		t.Fatal(err)
	}
	var query map[string]interface{}
	if err := json.Unmarshal(data, &query); err != nil {
		t.Fatal(err)
	}
	for key := range query {
		if _, ok := properties[key]; !ok {
			t.Fatalf("summary schema is missing query field %s", key)
		}
	}
	if slices.Contains(interfaceStrings(schema["required"].([]interface{})), "format") {
		t.Fatal("summary requires rendered plan format")
	}
}
