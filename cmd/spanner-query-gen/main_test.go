package main

import (
	"bytes"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/go-googlesql-spanner-poc/internal/querygen"
	"github.com/goccy/go-yaml"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRunRequiresSubcommand(t *testing.T) {
	err := run([]string{"--config", "querygen.yaml"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unsupported subcommand") {
		t.Fatalf("run() error = %v, want unsupported subcommand error", err)
	}
}

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"--help"}, &stdout, &stderr); err != nil {
		t.Fatalf("run --help error = %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("run --help wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		"Usage:",
		"spanner-query-gen <subcommand> [flags]",
		"generate",
		"config-schema",
		"plan-report-schema",
		"plan-contract-schema",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("run --help stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunConfigSchema(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"config-schema"}, &stdout, &stderr); err != nil {
		t.Fatalf("run config-schema error = %v, stderr = %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("config-schema wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		`"$schema": "https://json-schema.org/draft/2020-12/schema"`,
		`"const": "v1alpha"`,
		`"external_query_connections"`,
		`"spanner_external_datasets"`,
		`"params"`,
		`"query_param"`,
		`"scope"`,
		`"order_by"`,
		`"suppressions"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("config-schema stdout missing %q:\n%s", want, stdout.String())
		}
	}
	for _, unwanted := range []string{
		`"severity"`,
		`"query_methods"`,
		`"database_role"`,
		`"not_checked"`,
		`"user_hint"`,
		`"user_config"`,
		`"live_probe"`,
		`"external_evidence"`,
	} {
		if strings.Contains(stdout.String(), unwanted) {
			t.Fatalf("config-schema stdout contains unsupported v1alpha field %q:\n%s", unwanted, stdout.String())
		}
	}
}

func TestRunConfigSchemaRequiresVerificationEvidenceFields(t *testing.T) {
	data, err := configSchemaBytes("json")
	if err != nil {
		t.Fatalf("configSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	defs := schema["$defs"].(map[string]interface{})
	verification := defs["verification_evidence"].(map[string]interface{})
	required := verification["required"].([]interface{})
	if got, want := strings.Join(interfaceStrings(required), ","), "status,verifier,checked_at"; got != want {
		t.Fatalf("verification_evidence required = %q, want %q", got, want)
	}
	properties := verification["properties"].(map[string]interface{})
	checkedAt := properties["checked_at"].(map[string]interface{})
	if got, want := checkedAt["format"], "date-time"; got != want {
		t.Fatalf("verification_evidence checked_at format = %q, want %q", got, want)
	}
	for _, field := range []string{"source", "evidence_digest"} {
		if _, ok := properties[field]; ok {
			t.Fatalf("verification_evidence schema contains unsupported config field %q", field)
		}
	}
}

func TestRunConfigSchemaQueryParamScopeEnum(t *testing.T) {
	data, err := configSchemaBytes("json")
	if err != nil {
		t.Fatalf("configSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	defs := schema["$defs"].(map[string]interface{})
	queryParam := defs["query_param"].(map[string]interface{})
	properties := queryParam["properties"].(map[string]interface{})
	scope := properties["scope"].(map[string]interface{})
	values := scope["enum"].([]interface{})
	if got, want := strings.Join(interfaceStrings(values), ","), "inner,outer"; got != want {
		t.Fatalf("query_param scope enum = %q, want %q", got, want)
	}
}

func TestRunConfigSchemaQueryOrderByEnum(t *testing.T) {
	data, err := configSchemaBytes("json")
	if err != nil {
		t.Fatalf("configSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	defs := schema["$defs"].(map[string]interface{})
	query := defs["query"].(map[string]interface{})
	properties := query["properties"].(map[string]interface{})
	orderBy := properties["order_by"].(map[string]interface{})
	values := orderBy["enum"].([]interface{})
	if got, want := strings.Join(interfaceStrings(values), ","), "key,none"; got != want {
		t.Fatalf("query order_by enum = %q, want %q", got, want)
	}
}

func TestRunConfigSchemaKeepsExecAndConflictOutOfPublicV1Alpha(t *testing.T) {
	data, err := configSchemaBytes("json")
	if err != nil {
		t.Fatalf("configSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	defs := schema["$defs"].(map[string]interface{})
	queryResult := defs["query_result"].(map[string]interface{})
	resultProperties := queryResult["properties"].(map[string]interface{})
	cardinality := resultProperties["cardinality"].(map[string]interface{})
	if got, want := strings.Join(interfaceStrings(cardinality["enum"].([]interface{})), ","), "one,maybe_one,many"; got != want {
		t.Fatalf("query_result.cardinality enum = %q, want %q", got, want)
	}
	write := defs["write"].(map[string]interface{})
	writeProperties := write["properties"].(map[string]interface{})
	if _, ok := writeProperties["conflict"]; ok {
		t.Fatalf("write schema still exposes public conflict property")
	}
}

func TestRunConfigSchemaCatalogKindSpecificShape(t *testing.T) {
	data, err := configSchemaBytes("json")
	if err != nil {
		t.Fatalf("configSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	defs := schema["$defs"].(map[string]interface{})
	catalog := defs["catalog"].(map[string]interface{})
	oneOf := catalog["allOf"].([]interface{})[0].(map[string]interface{})["oneOf"].([]interface{})
	if got, want := len(oneOf), 2; got != want {
		t.Fatalf("catalog oneOf length = %d, want %d", got, want)
	}
	for _, clauseValue := range oneOf {
		clause := clauseValue.(map[string]interface{})
		properties := clause["properties"].(map[string]interface{})
		kind := properties["kind"].(map[string]interface{})["const"].(string)
		not := clause["not"].(map[string]interface{})
		forbidden := strings.Join(anyOfRequiredFields(not), ",")
		switch kind {
		case "spanner":
			if forbidden != "project,bindings" {
				t.Fatalf("spanner catalog forbidden fields = %q, want project,bindings", forbidden)
			}
		case "bigquery":
			if forbidden != "proto_descriptors" {
				t.Fatalf("bigquery catalog forbidden fields = %q, want proto_descriptors", forbidden)
			}
		default:
			t.Fatalf("unexpected catalog kind clause %q", kind)
		}
	}
}

func TestRunConfigSchemaRegularQueryParamsForbidScope(t *testing.T) {
	data, err := configSchemaBytes("json")
	if err != nil {
		t.Fatalf("configSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	defs := schema["$defs"].(map[string]interface{})
	query := defs["query"].(map[string]interface{})
	oneOf := query["allOf"].([]interface{})[0].(map[string]interface{})["oneOf"].([]interface{})
	for _, clauseValue := range oneOf {
		clause := clauseValue.(map[string]interface{})
		properties := clause["properties"].(map[string]interface{})
		kind := properties["kind"].(map[string]interface{})["const"].(string)
		if kind == "external_query" {
			continue
		}
		params := properties["params"].(map[string]interface{})
		items := params["items"].(map[string]interface{})
		allOf := items["allOf"].([]interface{})
		not := allOf[1].(map[string]interface{})["not"].(map[string]interface{})
		required := not["required"].([]interface{})
		if got, want := strings.Join(interfaceStrings(required), ","), "scope"; got != want {
			t.Fatalf("%s params forbidden required = %q, want %q", kind, got, want)
		}
	}
}

func TestRunConfigSchemaArraySyntaxConstraints(t *testing.T) {
	data, err := configSchemaBytes("json")
	if err != nil {
		t.Fatalf("configSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	properties := schema["properties"].(map[string]interface{})
	for _, name := range []string{"catalogs", "queries", "writes"} {
		array := properties[name].(map[string]interface{})
		if got, want := array["minItems"], float64(1); got != want {
			t.Fatalf("%s minItems = %v, want %v", name, got, want)
		}
		if got, want := array["uniqueItems"], true; got != want {
			t.Fatalf("%s uniqueItems = %v, want %v", name, got, want)
		}
	}
	defs := schema["$defs"].(map[string]interface{})
	query := defs["query"].(map[string]interface{})
	queryProperties := query["properties"].(map[string]interface{})
	params := queryProperties["params"].(map[string]interface{})
	if got, want := params["minItems"], float64(1); got != want {
		t.Fatalf("params minItems = %v, want %v", got, want)
	}
	if got, want := params["uniqueItems"], true; got != want {
		t.Fatalf("params uniqueItems = %v, want %v", got, want)
	}
}

func TestRunConfigSchemaIdentifierReferences(t *testing.T) {
	data, err := configSchemaBytes("json")
	if err != nil {
		t.Fatalf("configSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	defs := schema["$defs"].(map[string]interface{})
	for _, tt := range []struct {
		name     string
		def      string
		property string
	}{
		{name: "query catalog", def: "query", property: "catalog"},
		{name: "query binding", def: "query", property: "binding"},
		{name: "write catalog", def: "write", property: "catalog"},
		{name: "external query connection spanner catalog", def: "external_query_connection", property: "spanner_catalog"},
		{name: "spanner external dataset spanner catalog", def: "spanner_external_dataset", property: "spanner_catalog"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			def := defs[tt.def].(map[string]interface{})
			properties := def["properties"].(map[string]interface{})
			property := properties[tt.property].(map[string]interface{})
			if got, want := property["$ref"], "#/$defs/identifier"; got != want {
				t.Fatalf("%s.%s $ref = %v, want %v", tt.def, tt.property, got, want)
			}
		})
	}
}

func TestRunConfigSchemaHasV1AlphaDiscriminatedUnions(t *testing.T) {
	data, err := configSchemaBytes("json")
	if err != nil {
		t.Fatalf("configSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	required := schema["required"].([]interface{})
	if got, want := strings.Join(interfaceStrings(required), ","), "version,go,catalogs"; got != want {
		t.Fatalf("root required = %q, want %q", got, want)
	}
	if _, ok := schema["anyOf"].([]interface{}); !ok {
		t.Fatalf("root schema missing anyOf for queries/writes")
	}
	defs := schema["$defs"].(map[string]interface{})
	query := defs["query"].(map[string]interface{})
	queryAllOf := query["allOf"].([]interface{})
	queryOneOf := queryAllOf[0].(map[string]interface{})["oneOf"].([]interface{})
	if got, want := len(queryOneOf), 4; got != want {
		t.Fatalf("query kind oneOf length = %d, want %d", got, want)
	}
	write := defs["write"].(map[string]interface{})
	writeAllOf := write["allOf"].([]interface{})
	writeOneOf := writeAllOf[0].(map[string]interface{})["oneOf"].([]interface{})
	if got, want := len(writeOneOf), 5; got != want {
		t.Fatalf("write operation oneOf length = %d, want %d", got, want)
	}
}

func TestRunConfigSchemaYAML(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"config-schema", "--output", "yaml"}, &stdout, &stderr); err != nil {
		t.Fatalf("run config-schema --output yaml error = %v, stderr = %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("config-schema --output yaml wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		`$schema: https://json-schema.org/draft/2020-12/schema`,
		`const: v1alpha`,
		`external_query_connections:`,
		`spanner_external_datasets:`,
		`query_param:`,
		`scope:`,
		`order_by:`,
		`oneOf:`,
		`suppressions:`,
		`- status`,
		`- verifier`,
		`- checked_at`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("config-schema yaml stdout missing %q:\n%s", want, stdout.String())
		}
	}
	for _, unwanted := range []string{
		`severity:`,
		`query_methods:`,
		`database_role:`,
		`not_checked`,
		`user_hint`,
		`user_config`,
		`live_probe`,
		`external_evidence`,
	} {
		if strings.Contains(stdout.String(), unwanted) {
			t.Fatalf("config-schema yaml stdout contains unsupported v1alpha field %q:\n%s", unwanted, stdout.String())
		}
	}
}

func TestMarshalYAMLViaJSONUsesJSONShape(t *testing.T) {
	type sample struct {
		Value string `json:"json_name" yaml:"yaml_name"`
		Empty string `json:"empty,omitempty" yaml:"empty"`
	}

	data, err := marshalYAMLViaJSON(sample{Value: "value"})
	if err != nil {
		t.Fatalf("marshalYAMLViaJSON() error = %v", err)
	}
	out := string(data)
	for _, want := range []string{
		"json_name: value",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("marshalYAMLViaJSON() output missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{
		"yaml_name:",
		"empty:",
	} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("marshalYAMLViaJSON() output contains %q:\n%s", unwanted, out)
		}
	}

	var decoded map[string]interface{}
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v\n%s", err, out)
	}
	if got, want := decoded["json_name"], "value"; got != want {
		t.Fatalf("decoded json_name = %v, want %v", got, want)
	}
}

func interfaceStrings(values []interface{}) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.(string))
	}
	return out
}

func anyOfRequiredFields(not map[string]interface{}) []string {
	clauses := not["anyOf"].([]interface{})
	out := make([]string, 0, len(clauses))
	for _, clauseValue := range clauses {
		clause := clauseValue.(map[string]interface{})
		required := clause["required"].([]interface{})
		out = append(out, interfaceStrings(required)...)
	}
	return out
}

func TestRunConfigSchemaOutWritesFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "schema.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"config-schema", "--out", outPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run config-schema --out error = %v, stderr = %s", err, stderr.String())
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("config-schema --out wrote stdout/stderr:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", outPath, err)
	}
	want, err := configSchemaBytes("json")
	if err != nil {
		t.Fatalf("configSchemaBytes(json) error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("schema file mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestConfigSchemaFileIsCurrent(t *testing.T) {
	path := filepath.Join("..", "..", "schemas", "spanner-query-gen.v1alpha.schema.json")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", path, err)
	}
	want, err := configSchemaBytes("json")
	if err != nil {
		t.Fatalf("configSchemaBytes(json) error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s is stale; run `go run ./cmd/spanner-query-gen config-schema --out schemas/spanner-query-gen.v1alpha.schema.json`", path)
	}
}

func TestRunPlanReportSchema(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"plan-report-schema"}, &stdout, &stderr); err != nil {
		t.Fatalf("run plan-report-schema error = %v, stderr = %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("plan-report-schema wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		`"$schema": "https://json-schema.org/draft/2020-12/schema"`,
		`"title": "spanner-query-gen plan-report output"`,
		`"report_version"`,
		`"input"`,
		`"plan_source"`,
		`"backend_identity"`,
		`"cel_input_defaults"`,
		`"optional_string"`,
		`"target_summary"`,
		`"target_id"`,
		`"operator_family_counts"`,
		`"normalized_operators"`,
		`"contract_evaluation_mode"`,
		`"contract_evaluator_version"`,
		`"stability"`,
		`"failure_kind"`,
		`"source"`,
		`"predefined"`,
		`"contract_evaluations"`,
		`"environment_warnings"`,
		`"warnings"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("plan-report-schema stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunPlanReportSchemaYAML(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"plan-report-schema", "--output", "yaml"}, &stdout, &stderr); err != nil {
		t.Fatalf("run plan-report-schema --output yaml error = %v, stderr = %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("plan-report-schema --output yaml wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		`$schema: https://json-schema.org/draft/2020-12/schema`,
		`title: spanner-query-gen plan-report output`,
		`report_version:`,
		`input:`,
		`plan_source:`,
		`backend_identity:`,
		`cel_input_defaults:`,
		`target_summary:`,
		`target_id:`,
		`operator_family_counts:`,
		`normalized_operators:`,
		`contract_evaluation_mode:`,
		`contract_evaluator_version:`,
		`stability:`,
		`contract_evaluations:`,
		`warnings:`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("plan-report-schema yaml stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestPlanReportWarningsAreSerializedWhenEmpty(t *testing.T) {
	report := planReport{
		ReportVersion:    planReportVersionV1Alpha,
		Status:           planReportStatusOK,
		Backend:          "omni",
		Input:            planReportInput{ConfigSHA256: strings.Repeat("0", 64)},
		PlanSource:       planReportPlanSource{Backend: "omni", API: "analyze_query", RenderTool: "spannerplan"},
		BackendIdentity:  planReportBackendIdentity{Kind: "omni", Version: "not_recorded", ImageDigest: "not_recorded", Source: "spanemuboost"},
		Normalization:    defaultPlanReportNormalization(),
		TargetSummary:    planReportTargetSummary{Excluded: []planReportExcludedTarget{}},
		Format:           "CURRENT",
		RenderMode:       "PLAN",
		Queries:          []planReportQuery{},
		ContractEvalMode: planContractEvaluationModeNone,
		Warnings:         []planReportDiagnostic{},
		Optimizer:        planReportResolvedOptimizer(planReportOptimizerEnvironment{}),
	}
	jsonData, err := marshalIndentedJSON(report)
	if err != nil {
		t.Fatalf("marshalIndentedJSON(planReport) error = %v", err)
	}
	if !strings.Contains(string(jsonData), `"warnings": []`) {
		t.Fatalf("plan-report JSON missing empty warnings array:\n%s", jsonData)
	}
	yamlData, err := marshalYAMLViaJSON(report)
	if err != nil {
		t.Fatalf("marshalYAMLViaJSON(planReport) error = %v", err)
	}
	if !strings.Contains(string(yamlData), "warnings: []") {
		t.Fatalf("plan-report YAML missing empty warnings array:\n%s", yamlData)
	}
}

func TestRunPlanReportSchemaOutWritesFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "plan-report.schema.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"plan-report-schema", "--out", outPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run plan-report-schema --out error = %v, stderr = %s", err, stderr.String())
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("plan-report-schema --out wrote stdout/stderr:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", outPath, err)
	}
	want, err := planReportSchemaBytes("json")
	if err != nil {
		t.Fatalf("planReportSchemaBytes(json) error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("schema file mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRunPlanContractSchema(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"plan-contract-schema"}, &stdout, &stderr); err != nil {
		t.Fatalf("run plan-contract-schema error = %v, stderr = %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("plan-contract-schema wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		`"$schema": "https://json-schema.org/draft/2020-12/schema"`,
		`"title": "spanner-query-gen plan contracts v1alpha"`,
		`"const": "v1alpha-plan-contracts"`,
		`"target"`,
		`"forbid"`,
		`"no_explicit_sort"`,
		`"no_hash_aggregate"`,
		`"no_distributed_cross_apply"`,
		`"push_broadcast_hash_join_internal_hash_join"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("plan-contract-schema stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunPlanContractSchemaTargetPattern(t *testing.T) {
	data, err := planContractSchemaBytes("json")
	if err != nil {
		t.Fatalf("planContractSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	defs := schema["$defs"].(map[string]interface{})
	target := defs["target"].(map[string]interface{})
	if got, want := target["pattern"], planContractTargetIDPattern; got != want {
		t.Fatalf("target pattern = %v, want %v", got, want)
	}
}

func TestRunPlanContractSchemaPredefinedNamesAndForbidDefault(t *testing.T) {
	data, err := planContractSchemaBytes("json")
	if err != nil {
		t.Fatalf("planContractSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	defs := schema["$defs"].(map[string]interface{})
	contract := defs["contract"].(map[string]interface{})
	properties := contract["properties"].(map[string]interface{})
	use := properties["use"].(map[string]interface{})
	items := use["items"].(map[string]interface{})
	if got, want := items["enum"].([]interface{}), planContractPredefinedValues(); !reflect.DeepEqual(got, want) {
		t.Fatalf("use enum = %v, want %v", got, want)
	}
	forbid := defs["forbid"].(map[string]interface{})
	forbidProperties := forbid["properties"].(map[string]interface{})
	maxCount := forbidProperties["max_count"].(map[string]interface{})
	if got, want := maxCount["default"], float64(0); got != want {
		t.Fatalf("forbid.max_count default = %v, want %v", got, want)
	}
}

func TestRunPlanReportSchemaContractEvaluationResolvedTargetRequired(t *testing.T) {
	data, err := planReportSchemaBytes("json")
	if err != nil {
		t.Fatalf("planReportSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	defs := schema["$defs"].(map[string]interface{})
	evaluation := defs["contract_evaluation"].(map[string]interface{})
	allOf := evaluation["allOf"].([]interface{})
	for i, status := range []string{planContractStatusPass, planContractStatusFail} {
		clause := allOf[i].(map[string]interface{})
		ifClause := clause["if"].(map[string]interface{})
		properties := ifClause["properties"].(map[string]interface{})
		if got := properties["status"].(map[string]interface{})["const"]; got != status {
			t.Fatalf("contract_evaluation allOf[%d] status const = %v, want %s", i, got, status)
		}
		required := clause["then"].(map[string]interface{})["required"].([]interface{})
		if got, want := strings.Join(interfaceStrings(required), ","), "query,scope,results"; got != want {
			t.Fatalf("contract_evaluation %s required = %q, want %q", status, got, want)
		}
	}
	properties := evaluation["properties"].(map[string]interface{})
	targetID := properties["target_id"].(map[string]interface{})
	if got, want := targetID["pattern"], planContractTargetIDPattern; got != want {
		t.Fatalf("contract_evaluation.target_id pattern = %v, want %v", got, want)
	}
	scope := properties["scope"].(map[string]interface{})
	if got, want := scope["enum"].([]interface{}), []interface{}{"query", "external_query.inner"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("contract_evaluation.scope enum = %v, want %v", got, want)
	}
	if _, ok := properties["backend"]; ok {
		t.Fatal("contract_evaluation.backend must not be part of the v1alpha public report schema")
	}
}

func TestRunPlanContractSchemaYAML(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"plan-contract-schema", "--output", "yaml"}, &stdout, &stderr); err != nil {
		t.Fatalf("run plan-contract-schema --output yaml error = %v, stderr = %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("plan-contract-schema --output yaml wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		`$schema: https://json-schema.org/draft/2020-12/schema`,
		`title: spanner-query-gen plan contracts v1alpha`,
		`const: v1alpha-plan-contracts`,
		`target:`,
		`forbid:`,
		`no_explicit_sort`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("plan-contract-schema yaml stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunPlanContractSchemaOutWritesFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "plan-contracts.schema.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"plan-contract-schema", "--out", outPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run plan-contract-schema --out error = %v, stderr = %s", err, stderr.String())
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("plan-contract-schema --out wrote stdout/stderr:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", outPath, err)
	}
	want, err := planContractSchemaBytes("json")
	if err != nil {
		t.Fatalf("planContractSchemaBytes(json) error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("schema file mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestPlanContractSchemaFileIsCurrent(t *testing.T) {
	path := filepath.Join("..", "..", "schemas", "spanner-query-gen.plan-contracts.v1alpha.schema.json")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", path, err)
	}
	want, err := planContractSchemaBytes("json")
	if err != nil {
		t.Fatalf("planContractSchemaBytes(json) error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s is stale; run `go run ./cmd/spanner-query-gen plan-contract-schema --out schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json`", path)
	}
}

func TestPlanReportSchemaFileIsCurrent(t *testing.T) {
	path := filepath.Join("..", "..", "schemas", "spanner-query-gen.plan-report.v1alpha.schema.json")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", path, err)
	}
	want, err := planReportSchemaBytes("json")
	if err != nil {
		t.Fatalf("planReportSchemaBytes(json) error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s is stale; run `go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json`", path)
	}
}

func TestPlanReportSchemaContractConditionals(t *testing.T) {
	data, err := planReportSchemaBytes("json")
	if err != nil {
		t.Fatalf("planReportSchemaBytes(json) error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	required := interfaceStrings(schema["required"].([]interface{}))
	if !slices.Contains(required, "warnings") {
		t.Fatalf("plan-report root required fields = %v, want warnings", required)
	}
	defs := schema["$defs"].(map[string]interface{})
	contractModeClause := schema["allOf"].([]interface{})[0].(map[string]interface{})
	contractModeThen := contractModeClause["then"].(map[string]interface{})
	contractModeThenProperties := contractModeThen["properties"].(map[string]interface{})
	contractEvaluations := contractModeThenProperties["contract_evaluations"].(map[string]interface{})
	if got, want := contractEvaluations["minItems"], float64(1); got != want {
		t.Fatalf("contract evaluation mode contract_evaluations.minItems = %v, want %v", got, want)
	}
	backendIdentity := defs["backend_identity"].(map[string]interface{})
	if got, want := len(backendIdentity["allOf"].([]interface{})), 2; got != want {
		t.Fatalf("backend_identity allOf length = %d, want %d", got, want)
	}
	backendIdentityProperties := backendIdentity["properties"].(map[string]interface{})
	source := backendIdentityProperties["source"].(map[string]interface{})
	if got := source["description"].(string); !strings.Contains(got, "caller supplied the value as an assertion") {
		t.Fatalf("backend_identity.source description = %q, want caller assertion wording", got)
	}
	manualClause := backendIdentity["allOf"].([]interface{})[0].(map[string]interface{})
	manualThen := manualClause["then"].(map[string]interface{})
	manualNot := manualThen["not"].(map[string]interface{})
	manualNotProperties := manualNot["properties"].(map[string]interface{})
	if got := manualNotProperties["version"].(map[string]interface{})["const"]; got != "not_recorded" {
		t.Fatalf("backend_identity manual version forbidden const = %v, want not_recorded", got)
	}
	if got := manualNotProperties["image_digest"].(map[string]interface{})["const"]; got != "not_recorded" {
		t.Fatalf("backend_identity manual image_digest forbidden const = %v, want not_recorded", got)
	}
	notRecordedClause := backendIdentity["allOf"].([]interface{})[1].(map[string]interface{})
	notRecordedThen := notRecordedClause["then"].(map[string]interface{})
	notRecordedProperties := notRecordedThen["properties"].(map[string]interface{})
	if got := notRecordedProperties["version"].(map[string]interface{})["const"]; got != "not_recorded" {
		t.Fatalf("backend_identity not_recorded version const = %v, want not_recorded", got)
	}
	if got := notRecordedProperties["image_digest"].(map[string]interface{})["const"]; got != "not_recorded" {
		t.Fatalf("backend_identity not_recorded image_digest const = %v, want not_recorded", got)
	}
	stability := defs["contract_stability"].(map[string]interface{})
	if got, want := len(stability["allOf"].([]interface{})), 2; got != want {
		t.Fatalf("contract_stability allOf length = %d, want %d", got, want)
	}
	rawPlanClause := stability["allOf"].([]interface{})[0].(map[string]interface{})
	rawPlanThen := rawPlanClause["then"].(map[string]interface{})
	rawPlanProperties := rawPlanThen["properties"].(map[string]interface{})
	if got := rawPlanProperties["check_recommended"].(map[string]interface{})["const"]; got != false {
		t.Fatalf("raw_query_plan check_recommended const = %v, want false", got)
	}
	if got := rawPlanProperties["replayable_from_report"].(map[string]interface{})["const"]; got != false {
		t.Fatalf("raw_query_plan replayable_from_report const = %v, want false", got)
	}
	normalizedClause := stability["allOf"].([]interface{})[1].(map[string]interface{})
	normalizedThen := normalizedClause["then"].(map[string]interface{})
	normalizedProperties := normalizedThen["properties"].(map[string]interface{})
	if got := normalizedProperties["replayable_from_report"].(map[string]interface{})["const"]; got != true {
		t.Fatalf("normalized_operator replayable_from_report const = %v, want true", got)
	}
	evaluation := defs["contract_evaluation"].(map[string]interface{})
	if got, want := len(evaluation["allOf"].([]interface{})), 5; got != want {
		t.Fatalf("contract_evaluation allOf length = %d, want %d", got, want)
	}
	result := defs["contract_rule_result"].(map[string]interface{})
	if got, want := len(result["allOf"].([]interface{})), 13; got != want {
		t.Fatalf("contract_rule_result allOf length = %d, want %d", got, want)
	}
	resultProperties := result["properties"].(map[string]interface{})
	resultSource := resultProperties["source"].(map[string]interface{})
	if got, want := resultSource["pattern"], planContractRuleResultSourcePattern(); got != want {
		t.Fatalf("contract_rule_result.source pattern = %v, want %v", got, want)
	}
	diagnosticID := resultProperties["diagnostic_id"].(map[string]interface{})
	if got, want := diagnosticID["pattern"], `^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$`; got != want {
		t.Fatalf("contract_rule_result.diagnostic_id pattern = %v, want %v", got, want)
	}
	predefined := resultProperties["predefined"].(map[string]interface{})
	if got, want := predefined["enum"].([]interface{}), planContractPredefinedValues(); !reflect.DeepEqual(got, want) {
		t.Fatalf("contract_rule_result.predefined enum = %v, want %v", got, want)
	}
	if _, ok := resultProperties["matched_operator_indexes"]; !ok {
		t.Fatalf("contract_rule_result missing matched_operator_indexes property")
	}
	forbidClause := result["allOf"].([]interface{})[0].(map[string]interface{})
	forbidThen := forbidClause["then"].(map[string]interface{})
	forbidThenProperties := forbidThen["properties"].(map[string]interface{})
	forbidThenSource := forbidThenProperties["source"].(map[string]interface{})
	if got, want := forbidThenSource["pattern"], planContractRuleResultForbidSourcePattern(); got != want {
		t.Fatalf("forbid_operator_family source pattern = %v, want %v", got, want)
	}
	forbidRequired := interfaceStrings(forbidThen["required"].([]interface{}))
	if !slices.Contains(forbidRequired, "matched_operator_indexes") {
		t.Fatalf("forbid_operator_family required fields = %v, want matched_operator_indexes", forbidRequired)
	}
	forbidForbidden := anyOfRequiredFields(forbidThen["not"].(map[string]interface{}))
	if got, want := strings.Join(forbidForbidden, ","), "expression"; got != want {
		t.Fatalf("forbid_operator_family forbidden fields = %q, want %q", got, want)
	}
	celClause := result["allOf"].([]interface{})[1].(map[string]interface{})
	celThen := celClause["then"].(map[string]interface{})
	celThenProperties := celThen["properties"].(map[string]interface{})
	celThenSource := celThenProperties["source"].(map[string]interface{})
	if got, want := celThenSource["const"], "cel"; got != want {
		t.Fatalf("cel source const = %v, want %v", got, want)
	}
	celForbidden := anyOfRequiredFields(celThen["not"].(map[string]interface{}))
	if got, want := strings.Join(celForbidden, ","), "predefined,operator_family,observed_count,max_count,matched_operator_indexes"; got != want {
		t.Fatalf("cel rule forbidden fields = %q, want %q", got, want)
	}
	blockingClause := result["allOf"].([]interface{})[2].(map[string]interface{})
	blockingThen := blockingClause["then"].(map[string]interface{})
	blockingThenProperties := blockingThen["properties"].(map[string]interface{})
	blockingThenSource := blockingThenProperties["source"].(map[string]interface{})
	if got, want := blockingThenSource["const"], "use/no_blocking_operator_under_limit"; got != want {
		t.Fatalf("blocking-operator-under-limit source const = %v, want %v", got, want)
	}
	blockingRequired := interfaceStrings(blockingThen["required"].([]interface{}))
	for _, want := range []string{"predefined", "observed_count", "max_count", "matched_operator_indexes"} {
		if !slices.Contains(blockingRequired, want) {
			t.Fatalf("blocking-operator-under-limit required fields = %v, want %q", blockingRequired, want)
		}
	}
	blockingForbidden := anyOfRequiredFields(blockingThen["not"].(map[string]interface{}))
	if got, want := strings.Join(blockingForbidden, ","), "expression,operator_family,diagnostic_id"; got != want {
		t.Fatalf("blocking-operator-under-limit forbidden fields = %q, want %q", got, want)
	}
	fullScanClause := result["allOf"].([]interface{})[3].(map[string]interface{})
	fullScanThen := fullScanClause["then"].(map[string]interface{})
	fullScanThenProperties := fullScanThen["properties"].(map[string]interface{})
	fullScanThenSource := fullScanThenProperties["source"].(map[string]interface{})
	if got, want := fullScanThenSource["const"], "use/no_full_scan"; got != want {
		t.Fatalf("full-scan source const = %v, want %v", got, want)
	}
	fullScanRequired := interfaceStrings(fullScanThen["required"].([]interface{}))
	for _, want := range []string{"predefined", "observed_count", "max_count", "matched_operator_indexes"} {
		if !slices.Contains(fullScanRequired, want) {
			t.Fatalf("full-scan required fields = %v, want %q", fullScanRequired, want)
		}
	}
	fullScanForbidden := anyOfRequiredFields(fullScanThen["not"].(map[string]interface{}))
	if got, want := strings.Join(fullScanForbidden, ","), "expression,operator_family,diagnostic_id"; got != want {
		t.Fatalf("full-scan forbidden fields = %q, want %q", got, want)
	}
	fullScanWithoutTimestampClause := result["allOf"].([]interface{})[4].(map[string]interface{})
	fullScanWithoutTimestampThen := fullScanWithoutTimestampClause["then"].(map[string]interface{})
	fullScanWithoutTimestampThenProperties := fullScanWithoutTimestampThen["properties"].(map[string]interface{})
	fullScanWithoutTimestampThenSource := fullScanWithoutTimestampThenProperties["source"].(map[string]interface{})
	if got, want := fullScanWithoutTimestampThenSource["const"], "use/no_full_scan_without_timestamp_condition"; got != want {
		t.Fatalf("full-scan-without-timestamp source const = %v, want %v", got, want)
	}
	fullScanWithoutTimestampRequired := interfaceStrings(fullScanWithoutTimestampThen["required"].([]interface{}))
	for _, want := range []string{"predefined", "observed_count", "max_count", "matched_operator_indexes"} {
		if !slices.Contains(fullScanWithoutTimestampRequired, want) {
			t.Fatalf("full-scan-without-timestamp required fields = %v, want %q", fullScanWithoutTimestampRequired, want)
		}
	}
	fullScanWithoutTimestampForbidden := anyOfRequiredFields(fullScanWithoutTimestampThen["not"].(map[string]interface{}))
	if got, want := strings.Join(fullScanWithoutTimestampForbidden, ","), "expression,operator_family,diagnostic_id"; got != want {
		t.Fatalf("full-scan-without-timestamp forbidden fields = %q, want %q", got, want)
	}
	timestampConditionClause := result["allOf"].([]interface{})[5].(map[string]interface{})
	timestampConditionThen := timestampConditionClause["then"].(map[string]interface{})
	timestampConditionThenProperties := timestampConditionThen["properties"].(map[string]interface{})
	timestampConditionThenSource := timestampConditionThenProperties["source"].(map[string]interface{})
	if got, want := timestampConditionThenSource["const"], "use/require_timestamp_condition"; got != want {
		t.Fatalf("timestamp-condition source const = %v, want %v", got, want)
	}
	timestampConditionRequired := interfaceStrings(timestampConditionThen["required"].([]interface{}))
	for _, want := range []string{"predefined", "observed_count", "max_count", "matched_operator_indexes"} {
		if !slices.Contains(timestampConditionRequired, want) {
			t.Fatalf("timestamp-condition required fields = %v, want %q", timestampConditionRequired, want)
		}
	}
	timestampConditionForbidden := anyOfRequiredFields(timestampConditionThen["not"].(map[string]interface{}))
	if got, want := strings.Join(timestampConditionForbidden, ","), "expression,operator_family,diagnostic_id"; got != want {
		t.Fatalf("timestamp-condition forbidden fields = %q, want %q", got, want)
	}
	celFailClause := result["allOf"].([]interface{})[6].(map[string]interface{})
	celFailThen := celFailClause["then"].(map[string]interface{})
	celFailThenProperties := celFailThen["properties"].(map[string]interface{})
	celFailKind := celFailThenProperties["failure_kind"].(map[string]interface{})
	if got, want := celFailKind["const"], planContractFailureKindViolation; got != want {
		t.Fatalf("cel fail failure_kind const = %v, want %v", got, want)
	}
	celFailForbidden := anyOfRequiredFields(celFailThen["not"].(map[string]interface{}))
	if got, want := strings.Join(celFailForbidden, ","), "diagnostic_id"; got != want {
		t.Fatalf("cel fail forbidden fields = %q, want %q", got, want)
	}
}

func TestPlanReportSchemaSeparatesConcreteAndDerivedOperatorFamilies(t *testing.T) {
	schema := unmarshalSchemaForTest(t, planReportSchemaBytes)
	defs := schema["$defs"].(map[string]interface{})
	operatorFamily := defs["operator_family"].(map[string]interface{})
	if !slices.Contains(interfaceStrings(operatorFamily["enum"].([]interface{})), "explicit_sort") {
		t.Fatalf("operator_family enum must include derived explicit_sort")
	}
	if !slices.Contains(interfaceStrings(operatorFamily["enum"].([]interface{})), "blocking_operator") {
		t.Fatalf("operator_family enum must include derived blocking_operator")
	}
	concreteOperatorFamily := defs["concrete_operator_family"].(map[string]interface{})
	concreteValues := interfaceStrings(concreteOperatorFamily["enum"].([]interface{}))
	if slices.Contains(concreteValues, "explicit_sort") {
		t.Fatalf("concrete_operator_family enum must not include derived explicit_sort")
	}
	if slices.Contains(concreteValues, "blocking_operator") {
		t.Fatalf("concrete_operator_family enum must not include derived blocking_operator")
	}
	operator := defs["operator"].(map[string]interface{})
	operatorProperties := operator["properties"].(map[string]interface{})
	familyRef := operatorProperties["family"].(map[string]interface{})
	if got, want := familyRef["$ref"], "#/$defs/concrete_operator_family"; got != want {
		t.Fatalf("operator.family ref = %v, want %v", got, want)
	}
	if _, ok := operatorProperties["subquery_cluster_node"]; !ok {
		t.Fatalf("operator schema missing subquery_cluster_node metadata property")
	}
	if _, ok := operatorProperties["spool_name"]; !ok {
		t.Fatalf("operator schema missing spool_name metadata property")
	}
	query := defs["query"].(map[string]interface{})
	queryProperties := query["properties"].(map[string]interface{})
	operatorFamilies := queryProperties["operator_families"].(map[string]interface{})
	items := operatorFamilies["items"].(map[string]interface{})
	if got, want := items["$ref"], "#/$defs/concrete_operator_family"; got != want {
		t.Fatalf("query.operator_families item ref = %v, want %v", got, want)
	}
}

func TestPlanReportSchemaTargetIDsAndScopesAreCanonical(t *testing.T) {
	schema := unmarshalSchemaForTest(t, planReportSchemaBytes)
	defs := schema["$defs"].(map[string]interface{})
	query := defs["query"].(map[string]interface{})
	queryProperties := query["properties"].(map[string]interface{})
	queryTargetID := queryProperties["target_id"].(map[string]interface{})
	if got, want := queryTargetID["pattern"], planContractTargetIDPattern; got != want {
		t.Fatalf("query.target_id pattern = %v, want %v", got, want)
	}
	queryName := queryProperties["name"].(map[string]interface{})
	if got, want := queryName["pattern"], v1AlphaIdentifierPattern; got != want {
		t.Fatalf("query.name pattern = %v, want %v", got, want)
	}
	queryScope := queryProperties["scope"].(map[string]interface{})
	if got, want := queryScope["enum"].([]interface{}), []interface{}{"query", "external_query.inner"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("query.scope enum = %v, want %v", got, want)
	}
	contractEvaluation := defs["contract_evaluation"].(map[string]interface{})
	contractEvaluationProperties := contractEvaluation["properties"].(map[string]interface{})
	contractName := contractEvaluationProperties["name"].(map[string]interface{})
	if got, want := contractName["pattern"], v1AlphaIdentifierPattern; got != want {
		t.Fatalf("contract_evaluation.name pattern = %v, want %v", got, want)
	}
	contractQuery := contractEvaluationProperties["query"].(map[string]interface{})
	if got, want := contractQuery["pattern"], v1AlphaIdentifierPattern; got != want {
		t.Fatalf("contract_evaluation.query pattern = %v, want %v", got, want)
	}
	excluded := defs["excluded_target"].(map[string]interface{})
	excludedProperties := excluded["properties"].(map[string]interface{})
	excludedID := excludedProperties["id"].(map[string]interface{})
	if got, want := excludedID["pattern"], planContractTargetIDPattern; got != want {
		t.Fatalf("excluded_target.id pattern = %v, want %v", got, want)
	}
	excludedScope := excludedProperties["scope"].(map[string]interface{})
	if got, want := excludedScope["enum"].([]interface{}), []interface{}{"query", "external_query.inner"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("excluded_target.scope enum = %v, want %v", got, want)
	}
	excludedReason := excludedProperties["reason"].(map[string]interface{})
	if got, want := excludedReason["pattern"], `^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$`; got != want {
		t.Fatalf("excluded_target.reason pattern = %v, want %v", got, want)
	}
}

func TestPlanReportSchemaTargetSummaryCounts(t *testing.T) {
	schema := unmarshalSchemaForTest(t, planReportSchemaBytes)
	defs := schema["$defs"].(map[string]interface{})
	targetSummary := defs["target_summary"].(map[string]interface{})
	required := interfaceStrings(targetSummary["required"].([]interface{}))
	wantRequired := []string{"included_count", "planned", "errors", "skipped", "excluded"}
	if !reflect.DeepEqual(required, wantRequired) {
		t.Fatalf("target_summary required = %v, want %v", required, wantRequired)
	}
	properties := targetSummary["properties"].(map[string]interface{})
	for _, name := range wantRequired {
		if _, ok := properties[name]; !ok {
			t.Fatalf("target_summary properties missing %q", name)
		}
	}
	excluded := defs["excluded_target"].(map[string]interface{})
	if got, want := interfaceStrings(excluded["required"].([]interface{})), []string{"id", "query", "scope", "reason"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("excluded_target required = %v, want %v", got, want)
	}
}

func TestPlanReportSchemaErrorAndSkippedQueriesForbidPlanFields(t *testing.T) {
	schema := unmarshalSchemaForTest(t, planReportSchemaBytes)
	defs := schema["$defs"].(map[string]interface{})
	query := defs["query"].(map[string]interface{})
	allOf := query["allOf"].([]interface{})
	var okClause map[string]interface{}
	for _, value := range allOf {
		candidate := value.(map[string]interface{})
		ifBlock := candidate["if"].(map[string]interface{})
		properties := ifBlock["properties"].(map[string]interface{})
		statusSchema := properties["status"].(map[string]interface{})
		if statusSchema["const"] == "ok" {
			okClause = candidate
			break
		}
	}
	if okClause == nil {
		t.Fatalf("query schema missing allOf clause for status ok")
	}
	okThen := okClause["then"].(map[string]interface{})
	okRequired := interfaceStrings(okThen["required"].([]interface{}))
	if !slices.Contains(okRequired, "operator_edges") {
		t.Fatalf("status ok required fields = %v, want operator_edges", okRequired)
	}
	wantForbidden := []string{
		"operator_tree_sha256",
		"operator_families",
		"operator_family_counts",
		"normalized_operators",
		"operator_edges",
		"plan",
		"classification_warnings",
	}
	for _, status := range []string{"error", "skipped"} {
		t.Run(status, func(t *testing.T) {
			var clause map[string]interface{}
			for _, value := range allOf {
				candidate := value.(map[string]interface{})
				ifBlock := candidate["if"].(map[string]interface{})
				properties := ifBlock["properties"].(map[string]interface{})
				statusSchema := properties["status"].(map[string]interface{})
				if statusSchema["const"] == status {
					clause = candidate
					break
				}
			}
			if clause == nil {
				t.Fatalf("query schema missing allOf clause for status %q", status)
			}
			then := clause["then"].(map[string]interface{})
			forbidden := anyOfRequiredFields(then["not"].(map[string]interface{}))
			if !reflect.DeepEqual(forbidden, wantForbidden) {
				t.Fatalf("status %s forbidden fields = %v, want %v", status, forbidden, wantForbidden)
			}
		})
	}
}

func TestPlanReportSchemaOperatorFamilyCountsAreComplete(t *testing.T) {
	schema := unmarshalSchemaForTest(t, planReportSchemaBytes)
	defs := schema["$defs"].(map[string]interface{})
	query := defs["query"].(map[string]interface{})
	queryProperties := query["properties"].(map[string]interface{})
	counts := queryProperties["operator_family_counts"].(map[string]interface{})
	required := interfaceStrings(counts["required"].([]interface{}))
	if !reflect.DeepEqual(required, planReportKnownOperatorFamilies()) {
		t.Fatalf("operator_family_counts required = %v, want every known family %v", required, planReportKnownOperatorFamilies())
	}
	properties := counts["properties"].(map[string]interface{})
	for _, family := range planReportKnownOperatorFamilies() {
		if _, ok := properties[family]; !ok {
			t.Fatalf("operator_family_counts properties missing %q", family)
		}
	}
	operator := defs["operator"].(map[string]interface{})
	operatorRequired := interfaceStrings(operator["required"].([]interface{}))
	for _, field := range []string{"child_indexes", "descendant_indexes", "subtree_family_counts"} {
		if !slices.Contains(operatorRequired, field) {
			t.Fatalf("operator required fields = %v, want %s", operatorRequired, field)
		}
	}
	operatorProperties := operator["properties"].(map[string]interface{})
	subtreeCounts := operatorProperties["subtree_family_counts"].(map[string]interface{})
	subtreeRequired := interfaceStrings(subtreeCounts["required"].([]interface{}))
	if !reflect.DeepEqual(subtreeRequired, planReportKnownOperatorFamilies()) {
		t.Fatalf("operator subtree_family_counts required = %v, want every known family %v", subtreeRequired, planReportKnownOperatorFamilies())
	}
}

func TestPlanReportSchemaCELInputDefaultsAppliesToExactSet(t *testing.T) {
	schema := unmarshalSchemaForTest(t, planReportSchemaBytes)
	defs := schema["$defs"].(map[string]interface{})
	defaults := defs["cel_input_defaults"].(map[string]interface{})
	properties := defaults["properties"].(map[string]interface{})
	appliesTo := properties["applies_to"].(map[string]interface{})
	if got, want := appliesTo["minItems"], float64(2); got != want {
		t.Fatalf("cel_input_defaults.applies_to minItems = %v, want %v", got, want)
	}
	if got, want := appliesTo["maxItems"], float64(2); got != want {
		t.Fatalf("cel_input_defaults.applies_to maxItems = %v, want %v", got, want)
	}
	if got, want := appliesTo["uniqueItems"], true; got != want {
		t.Fatalf("cel_input_defaults.applies_to uniqueItems = %v, want %v", got, want)
	}
	items := appliesTo["items"].(map[string]interface{})
	if got, want := interfaceStrings(items["enum"].([]interface{})), []string{"operators[]", "operator_edges[]"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("cel_input_defaults.applies_to enum = %v, want %v", got, want)
	}
	prefixItems := appliesTo["prefixItems"].([]interface{})
	if got, want := prefixItems[0].(map[string]interface{})["const"], "operators[]"; got != want {
		t.Fatalf("cel_input_defaults.applies_to prefixItems[0] = %v, want %v", got, want)
	}
	if got, want := prefixItems[1].(map[string]interface{})["const"], "operator_edges[]"; got != want {
		t.Fatalf("cel_input_defaults.applies_to prefixItems[1] = %v, want %v", got, want)
	}
}

func TestPlanContractsMinimalReportOutcomeSnippetsStayCurrent(t *testing.T) {
	data, err := os.ReadFile("PLAN_CONTRACTS.md")
	if err != nil {
		t.Fatalf("os.ReadFile(PLAN_CONTRACTS.md) error = %v", err)
	}
	tests := []struct {
		name       string
		anchor     string
		wantMode   string
		wantStatus string
	}{
		{
			name:       "pass",
			anchor:     "Pass example:",
			wantMode:   planContractEvaluationModeCheck,
			wantStatus: planContractStatusPass,
		},
		{
			name:       "report_only fail",
			anchor:     "`report_only` fail example.",
			wantMode:   planContractEvaluationModeReportOnly,
			wantStatus: planContractStatusFail,
		},
		{
			name:       "not evaluated",
			anchor:     "Unavailable target example:",
			wantMode:   planContractEvaluationModeCheck,
			wantStatus: planContractStatusNotEvaluated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := markdownCodeBlockAfter(t, string(data), tt.anchor, "yaml")
			var snippet struct {
				ContractEvalMode    string                         `yaml:"contract_evaluation_mode"`
				ContractSummary     *planContractEvaluationSummary `yaml:"contract_summary"`
				ContractEvaluations []planContractEvaluation       `yaml:"contract_evaluations"`
			}
			if err := yaml.Unmarshal([]byte(block), &snippet); err != nil {
				t.Fatalf("yaml.Unmarshal(%s snippet) error = %v\n%s", tt.name, err, block)
			}
			if got := snippet.ContractEvalMode; got != tt.wantMode {
				t.Fatalf("contract_evaluation_mode = %q, want %q", got, tt.wantMode)
			}
			if snippet.ContractSummary == nil {
				t.Fatalf("contract_summary is nil")
			}
			if got := snippet.ContractSummary.Status; got != planContractSummaryStatusForSnippet(tt.wantStatus) {
				t.Fatalf("contract_summary.status = %q, want status matching %q", got, tt.wantStatus)
			}
			if len(snippet.ContractEvaluations) != 1 {
				t.Fatalf("contract_evaluations length = %d, want 1", len(snippet.ContractEvaluations))
			}
			evaluation := snippet.ContractEvaluations[0]
			if got := evaluation.Status; got != tt.wantStatus {
				t.Fatalf("evaluation.status = %q, want %q", got, tt.wantStatus)
			}
			if evaluation.TargetID == "" {
				t.Fatalf("evaluation.target_id is empty")
			}
			if evaluation.Stability.Tier != planContractStabilityNormalized || !evaluation.Stability.CheckRecommended {
				t.Fatalf("evaluation.stability = %+v, want normalized checkable stability", evaluation.Stability)
			}
			switch tt.wantStatus {
			case planContractStatusPass, planContractStatusFail:
				if evaluation.Query == "" || evaluation.Scope == "" {
					t.Fatalf("resolved evaluation missing query/scope: %+v", evaluation)
				}
				if len(evaluation.Results) == 0 {
					t.Fatalf("resolved evaluation has no results: %+v", evaluation)
				}
				for _, result := range evaluation.Results {
					if result.Rule == "" || result.Source == "" || result.Status == "" {
						t.Fatalf("result missing required rule/source/status fields: %+v", result)
					}
					if strings.HasPrefix(result.Source, "use/") && result.Predefined == "" {
						t.Fatalf("predefined result missing predefined field: %+v", result)
					}
				}
			case planContractStatusNotEvaluated:
				if evaluation.Reason == "" {
					t.Fatalf("not_evaluated evaluation missing reason: %+v", evaluation)
				}
				if len(evaluation.Results) != 0 {
					t.Fatalf("not_evaluated evaluation has results: %+v", evaluation)
				}
			default:
				t.Fatalf("unhandled status %q", tt.wantStatus)
			}
		})
	}
}

func TestPlanContractAndReportSchemaOperatorFamilyEnumsMatch(t *testing.T) {
	contractSchema := unmarshalSchemaForTest(t, planContractSchemaBytes)
	reportSchema := unmarshalSchemaForTest(t, planReportSchemaBytes)
	contractEnum := schemaOperatorFamilyEnum(t, contractSchema)
	reportEnum := schemaOperatorFamilyEnum(t, reportSchema)
	if !reflect.DeepEqual(contractEnum, reportEnum) {
		t.Fatalf("plan contract operator_family enum != plan report enum\ncontract=%v\nreport=%v", contractEnum, reportEnum)
	}
	if !reflect.DeepEqual(contractEnum, planReportKnownOperatorFamilies()) {
		t.Fatalf("schema operator_family enum = %v, want known families %v", contractEnum, planReportKnownOperatorFamilies())
	}
}

func TestPlanContractsDocumentEveryOperatorFamily(t *testing.T) {
	data, err := os.ReadFile("PLAN_CONTRACTS.md")
	if err != nil {
		t.Fatalf("os.ReadFile(PLAN_CONTRACTS.md) error = %v", err)
	}
	document := string(data)
	section := markdownSection(t, document, "## Operator Families", "## Predefined Contracts")
	for _, family := range planReportKnownOperatorFamilies() {
		rowPrefix := "| `" + family + "` |"
		if count := strings.Count(section, rowPrefix); count != 1 {
			t.Fatalf("PLAN_CONTRACTS.md documents operator family %q %d times, want exactly once", family, count)
		}
	}
}

func markdownSection(t testing.TB, document, start, end string) string {
	t.Helper()
	startIndex := strings.Index(document, start)
	if startIndex < 0 {
		t.Fatalf("section start %q not found", start)
	}
	rest := document[startIndex:]
	endIndex := strings.Index(rest, end)
	if endIndex < 0 {
		t.Fatalf("section end %q not found after %q", end, start)
	}
	return rest[:endIndex]
}

func markdownCodeBlockAfter(t testing.TB, document, anchor, lang string) string {
	t.Helper()
	anchorIndex := strings.Index(document, anchor)
	if anchorIndex < 0 {
		t.Fatalf("anchor %q not found", anchor)
	}
	fence := "```" + lang
	rest := document[anchorIndex:]
	fenceIndex := strings.Index(rest, fence)
	if fenceIndex < 0 {
		t.Fatalf("code fence %q not found after anchor %q", fence, anchor)
	}
	blockStart := fenceIndex + len(fence)
	blockRest := rest[blockStart:]
	fenceEnd := strings.Index(blockRest, "```")
	if fenceEnd < 0 {
		t.Fatalf("closing code fence not found after anchor %q", anchor)
	}
	return strings.TrimSpace(blockRest[:fenceEnd])
}

func planContractSummaryStatusForSnippet(evaluationStatus string) string {
	if evaluationStatus == planContractStatusPass {
		return planContractStatusPass
	}
	return planContractStatusFail
}

func TestObservedNormalizationImpactFamiliesAreKnown(t *testing.T) {
	path := filepath.Join("..", "..", "research", "spanner-query-plan-shape", "QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", path, err)
	}
	inImpact := false
	for _, line := range strings.Split(string(data), "\n") {
		switch strings.TrimSpace(line) {
		case "## Normalization Impact":
			inImpact = true
			continue
		case "":
			continue
		}
		if inImpact && strings.HasPrefix(line, "## ") {
			break
		}
		line = strings.TrimSpace(line)
		if !inImpact || !strings.HasPrefix(line, "- `") {
			continue
		}
		rest := strings.TrimPrefix(line, "- `")
		family, _, ok := strings.Cut(rest, "`")
		if !ok {
			t.Fatalf("malformed normalization impact line: %q", line)
		}
		if !planReportKnownOperatorFamily(family) {
			t.Fatalf("normalization impact family %q is not in planReportKnownOperatorFamilies", family)
		}
	}
}

func unmarshalSchemaForTest(t testing.TB, fn func(string) ([]byte, error)) map[string]interface{} {
	t.Helper()
	data, err := fn("json")
	if err != nil {
		t.Fatalf("schema bytes error = %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("json.Unmarshal(schema) error = %v", err)
	}
	return schema
}

func schemaOperatorFamilyEnum(t testing.TB, schema map[string]interface{}) []string {
	t.Helper()
	defs := schema["$defs"].(map[string]interface{})
	operatorFamily := defs["operator_family"].(map[string]interface{})
	raw := operatorFamily["enum"].([]interface{})
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		out = append(out, value.(string))
	}
	return out
}

func TestRunGenerateSupportsV1AlphaConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeTestFile(t, configPath, literalConfigYAML())

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"generate", "--config", configPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run generate error = %v, stderr = %s", err, stderr.String())
	}
	for _, want := range []string{
		"package querydemo",
		"type LiteralRow struct",
		"GetLiteralSQL",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunRejectsUnversionedConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeTestFile(t, configPath, `
package: querydemo
queries:
- name: GetLiteral
  sql: SELECT 1 AS value
  result_struct: LiteralRow
`)

	err := run([]string{"generate", "--config", configPath}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), `unsupported config version ""; use version: v1alpha`) {
		t.Fatalf("run generate error = %v, want v1 config error", err)
	}
}

func TestRunCheckSubcommandValidatesGeneratedFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "querydemo.sql.go")
	configPath := filepath.Join(dir, "querygen.yaml")
	writeTestFile(t, configPath, literalConfigYAMLWithOut())

	if err := run([]string{"generate", "--config", configPath}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("generate run() error = %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("generated file stat error = %v", err)
	}
	if err := run([]string{"check", "--config", configPath}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("check run() error = %v", err)
	}
	writeTestFile(t, outPath, "package stale\n")
	err := run([]string{"check", "--config", configPath}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "is not up to date") {
		t.Fatalf("check run() error = %v, want stale-file error", err)
	}
}

func TestRunExplainPlan(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeSingerSchema(t, dir)
	writeTestFile(t, configPath, `
version: v1alpha
go:
  package: querydemo
catalogs:
- name: app
  kind: spanner
  ddl: schema.sql
queries:
- name: ListSingers
  catalog: app
  kind: table
  table: Singers
  result:
    struct: SingerRow
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"explain-plan", "--config", configPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run explain-plan error = %v, stderr = %s", err, stderr.String())
	}
	for _, want := range []string{
		"queries:",
		"name: ListSingers",
		"kind: table",
		"order_by: key",
		"sql: SELECT `SingerId`, `FirstName` FROM `Singers` ORDER BY `SingerId`",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("explain-plan output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunExplainPlanJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeTestFile(t, configPath, literalConfigYAML())

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"explain-plan", "--config", configPath, "--output", "json"}, &stdout, &stderr); err != nil {
		t.Fatalf("run explain-plan json error = %v, stderr = %s", err, stderr.String())
	}
	for _, want := range []string{
		`"queries": [`,
		`"name": "GetLiteral"`,
		`"kind": "sql"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("explain-plan json output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunExplainPlanSummary(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeTestFile(t, configPath, `
version: v1alpha
go:
  package: querydemo
catalogs:
- name: app
  kind: spanner
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    cardinality: one
    struct: LiteralRow
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"explain-plan", "--config", configPath, "--output", "summary"}, &stdout, &stderr); err != nil {
		t.Fatalf("run explain-plan summary error = %v, stderr = %s", err, stderr.String())
	}
	for _, want := range []string{
		"Query: GetLiteral (sql, result: one, catalog: app)",
		"SQL: SELECT 1 AS value",
		"value: INT64 nullable",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("explain-plan summary output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunPlanReportHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"plan-report", "--help"}, &stdout, &stderr); err != nil {
		t.Fatalf("run plan-report --help error = %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("plan-report --help wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		"Usage of spanner-query-gen plan-report:",
		"-backend string",
		"-backend-version string",
		"-backend-image-digest string",
		"-check",
		"-contracts string",
		"-format string",
		"-optimizer-version string",
		"-optimizer-statistics-package string",
		"-require-targets",
		"-stable",
		"-wrap-width int",
		"-output string",
		"-require-optimizer-pinning",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("plan-report --help stdout missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "-render-mode") {
		t.Fatalf("plan-report --help stdout still exposes removed --render-mode flag:\n%s", stdout.String())
	}
}

func TestRunPlanReportRejectsRenderModeFlag(t *testing.T) {
	err := run([]string{"plan-report", "--render-mode", "profile"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("run plan-report --render-mode profile error = %v, want unknown flag rejection", err)
	}
}

func TestRunPlanReportNoTargetsRecordsInputDigest(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeTestFile(t, configPath, `
version: v1alpha
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
	configDigest, err := planReportFileDigest(configPath)
	if err != nil {
		t.Fatalf("planReportFileDigest() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"plan-report", "--config", configPath, "--output", "yaml", "--stable"}, &stdout, &stderr); err != nil {
		t.Fatalf("run plan-report no-targets error = %v, stderr = %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("plan-report no-targets wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		"status: no_targets",
		"config_sha256: " + configDigest,
		"contract_evaluation_mode: none",
		"skipped: 1",
		"excluded: []",
		"status: skipped",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("plan-report no-targets output missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "contract_file_path:") {
		t.Fatalf("stable plan-report output contains contract_file_path:\n%s", stdout.String())
	}
}

func TestPlanReportOptimizerPinningWarnings(t *testing.T) {
	report := planReport{
		Optimizer: planReportOptimizer{
			Requested: planReportOptimizerEnvironment{
				Version:           "not_pinned",
				StatisticsPackage: "not_pinned",
			},
		},
		Queries: []planReportQuery{{
			Name:               "ListSingers",
			OptimizerNotPinned: true,
		}},
	}
	if got, want := strings.Join(planReportOptimizerPinningWarnings(report), ","), "optimizer_not_pinned,statistics_package_not_pinned,query_optimizer_not_pinned"; got != want {
		t.Fatalf("planReportOptimizerPinningWarnings() = %q, want %q", got, want)
	}
	report.Optimizer = planReportOptimizer{Requested: planReportOptimizerEnvironment{Version: "8", StatisticsPackage: "latest"}}
	report.Queries[0].OptimizerNotPinned = false
	if got := planReportOptimizerPinningWarnings(report); len(got) != 0 {
		t.Fatalf("planReportOptimizerPinningWarnings() = %q, want no warnings", got)
	}
}

func TestDefaultPlanReportNormalizationCELInputDefaults(t *testing.T) {
	normalization := defaultPlanReportNormalization()
	if got, want := normalization.OperatorTreeVersion, "v1alpha"; got != want {
		t.Fatalf("operator tree version = %q, want %q", got, want)
	}
	if got, want := normalization.OperatorFamilyMappingVersion, "v1alpha"; got != want {
		t.Fatalf("operator family mapping version = %q, want %q", got, want)
	}
	if got, want := normalization.CELInputDefaults.OptionalString, ""; got != want {
		t.Fatalf("optional string default = %q, want %q", got, want)
	}
	if got, want := normalization.CELInputDefaults.OptionalBoolean, false; got != want {
		t.Fatalf("optional boolean default = %t, want %t", got, want)
	}
	if got, want := strings.Join(normalization.CELInputDefaults.AppliesTo, ","), "operators[],operator_edges[]"; got != want {
		t.Fatalf("CEL input default applies_to = %q, want %q", got, want)
	}
}

func TestWritePlanReportOutputs(t *testing.T) {
	report := planReport{
		ReportVersion: planReportVersionV1Alpha,
		Status:        "ok",
		Backend:       "omni",
		Input: planReportInput{
			ConfigSHA256: planReportDigest("config"),
		},
		Format:           "CURRENT",
		RenderMode:       "PLAN",
		BackendIdentity:  planReportBackendIdentity{Kind: "omni", Version: "not_recorded", ImageDigest: "not_recorded", Source: "spanemuboost"},
		ContractEvalMode: planContractEvaluationModeNone,
		Normalization:    defaultPlanReportNormalization(),
		TargetSummary: planReportTargetSummary{
			IncludedCount: 1,
			Planned:       1,
			Excluded:      []planReportExcludedTarget{},
		},
		Queries: []planReportQuery{
			{
				TargetID:           "query/ListSingers",
				Name:               "ListSingers",
				Catalog:            "app",
				Scope:              "query",
				Kind:               "table",
				Status:             "ok",
				SQL:                "SELECT SingerId FROM Singers ORDER BY SingerId",
				SQLSHA256:          planReportDigest("SELECT SingerId FROM Singers ORDER BY SingerId"),
				DDLSHA256:          planReportDigest("CREATE TABLE Singers"),
				OperatorTreeSHA256: planReportDigest("operators"),
				OperatorFamilies:   []string{"distributed_union", "scan"},
				OperatorFamilyCounts: map[string]int{
					"distributed_union": 1,
					"scan":              1,
				},
				NormalizedOperators: []planReportOperator{
					{
						Index:               0,
						DisplayName:         "Distributed Union",
						Family:              "distributed_union",
						ChildIndexes:        []int32{1},
						DescendantIndexes:   []int32{1},
						SubtreeFamilyCounts: planReportOperatorFamilyCounts([]planReportOperator{{Family: "distributed_union"}, {Family: "scan"}}),
					},
					{
						Index:               1,
						DisplayName:         "Scan",
						Family:              "scan",
						ChildIndexes:        []int32{},
						DescendantIndexes:   []int32{},
						SubtreeFamilyCounts: planReportOperatorFamilyCounts([]planReportOperator{{Family: "scan"}}),
					},
				},
				OperatorEdges: []planReportOperatorEdge{{ParentIndex: 0, ChildIndex: 1}},
				Plan:          "Distributed Union\n+- Local Distributed Union",
			},
		},
	}

	for _, tt := range []struct {
		name   string
		output string
		want   []string
	}{
		{
			name:   "markdown",
			output: "markdown",
			want: []string{
				"# Spanner Query Plan Report",
				"- Report version: `v1alpha-plan-report-v1`",
				"- Status: `ok`",
				"Config SHA-256:",
				"Contract evaluation mode: `none`",
				"## ListSingers",
				"- Target ID: `query/ListSingers`",
				"Operator tree SHA-256:",
				"Operator families:",
				"Operator family counts:",
				"```sql\nSELECT SingerId FROM Singers ORDER BY SingerId\n```",
				"Distributed Union",
			},
		},
		{
			name:   "yaml",
			output: "yaml",
			want: []string{
				"status: ok",
				"config_sha256:",
				"contract_evaluation_mode: none",
				"cel_input_defaults:",
				`optional_string: ""`,
				"applies_to:",
				"target_id: query/ListSingers",
				"backend: omni",
				"name: ListSingers",
				"sql_sha256:",
				"operator_tree_sha256:",
				"operator_families:",
				"operator_family_counts:",
				"operator_edges:",
				"parent_index: 0",
				"Distributed Union",
			},
		},
		{
			name:   "json",
			output: "json",
			want: []string{
				`"status": "ok"`,
				`"config_sha256":`,
				`"contract_evaluation_mode": "none"`,
				`"cel_input_defaults":`,
				`"optional_string": ""`,
				`"applies_to":`,
				`"target_id": "query/ListSingers"`,
				`"backend": "omni"`,
				`"name": "ListSingers"`,
				`"sql_sha256":`,
				`"operator_tree_sha256":`,
				`"operator_families":`,
				`"operator_family_counts":`,
				`"operator_edges": [`,
				`"parent_index": 0`,
				`"Distributed Union`,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			if err := writePlanReport(&stdout, tt.output, report); err != nil {
				t.Fatalf("writePlanReport(%s) error = %v", tt.output, err)
			}
			for _, want := range tt.want {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("writePlanReport(%s) output missing %q:\n%s", tt.output, want, stdout.String())
				}
			}
		})
	}
}

func TestWritePlanReportRejectsTargetSummaryInvariantViolation(t *testing.T) {
	report := planReport{
		ReportVersion: planReportVersionV1Alpha,
		Status:        planReportStatusOK,
		Backend:       "omni",
		TargetSummary: planReportTargetSummary{
			IncludedCount: 1,
			Planned:       0,
			Errors:        0,
			Skipped:       0,
		},
		ContractEvalMode: planContractEvaluationModeNone,
		Normalization:    defaultPlanReportNormalization(),
	}
	var stdout bytes.Buffer
	err := writePlanReport(&stdout, "yaml", report)
	if err == nil || !strings.Contains(err.Error(), "target_summary.planned + target_summary.errors + target_summary.skipped") {
		t.Fatalf("writePlanReport() error = %v, want target summary invariant violation", err)
	}
}

func TestWritePlanReportRejectsContractSummaryInvariantViolation(t *testing.T) {
	report := planReport{
		ReportVersion:    planReportVersionV1Alpha,
		Status:           planReportStatusOK,
		Backend:          "omni",
		BackendIdentity:  planReportBackendIdentity{Kind: "omni", Version: "not_recorded", ImageDigest: "not_recorded", Source: "spanemuboost"},
		ContractEvalMode: planContractEvaluationModeReportOnly,
		Normalization:    defaultPlanReportNormalization(),
		TargetSummary: planReportTargetSummary{
			Excluded: []planReportExcludedTarget{},
		},
		ContractEvaluations: []planContractEvaluation{{
			Name:     "Pass",
			TargetID: "query/ListSingers",
			Status:   planContractStatusPass,
		}},
		ContractSummary: &planContractEvaluationSummary{
			Status:    planContractStatusPass,
			Contracts: 1,
			Passed:    0,
		},
	}
	var stdout bytes.Buffer
	err := writePlanReport(&stdout, "yaml", report)
	if err == nil || !strings.Contains(err.Error(), "contract_summary.passed") {
		t.Fatalf("writePlanReport() error = %v, want contract summary invariant violation", err)
	}
}

func TestWritePlanReportRejectsContractModeWithoutEvaluations(t *testing.T) {
	report := newPlanReportInvariantTestReport(t)
	report.ContractEvalMode = planContractEvaluationModeReportOnly
	report.ContractSummary = &planContractEvaluationSummary{
		Status:              planContractStatusPass,
		EnvironmentWarnings: []string{},
	}
	var stdout bytes.Buffer
	err := writePlanReport(&stdout, "yaml", report)
	if err == nil || !strings.Contains(err.Error(), "requires at least one contract_evaluations entry") {
		t.Fatalf("writePlanReport() error = %v, want empty contract_evaluations invariant violation", err)
	}
}

func TestWritePlanReportRejectsBackendIdentityInvariantViolation(t *testing.T) {
	tests := []struct {
		name     string
		identity planReportBackendIdentity
		want     string
	}{
		{
			name:     "manual with no recorded field",
			identity: planReportBackendIdentity{Kind: "omni", Version: "not_recorded", ImageDigest: "not_recorded", Source: "manual"},
			want:     "backend_identity.source manual requires version or image_digest",
		},
		{
			name:     "not_recorded with recorded version",
			identity: planReportBackendIdentity{Kind: "omni", Version: "2026.r1-beta", ImageDigest: "not_recorded", Source: "not_recorded"},
			want:     "backend_identity.source not_recorded requires version and image_digest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := newPlanReportInvariantTestReport(t)
			report.BackendIdentity = tt.identity
			var stdout bytes.Buffer
			err := writePlanReport(&stdout, "yaml", report)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("writePlanReport() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestWritePlanReportRejectsTopologyInvariantViolation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*planReport)
		want   string
	}{
		{
			name: "duplicate operator index",
			mutate: func(report *planReport) {
				report.Queries[0].NormalizedOperators = append(report.Queries[0].NormalizedOperators, report.Queries[0].NormalizedOperators[0])
			},
			want: "normalized_operators index 0 is duplicated",
		},
		{
			name: "missing edge child",
			mutate: func(report *planReport) {
				report.Queries[0].OperatorEdges = []planReportOperatorEdge{{ParentIndex: 0, ChildIndex: 99}}
			},
			want: "operator_edges child_index 99 is not in normalized_operators",
		},
		{
			name: "stale subtree family counts",
			mutate: func(report *planReport) {
				report.Queries[0].NormalizedOperators[0].SubtreeFamilyCounts["scan"] = 0
			},
			want: "subtree_family_counts",
		},
		{
			name: "stale whole tree family counts",
			mutate: func(report *planReport) {
				report.Queries[0].OperatorFamilyCounts["scan"] = 0
			},
			want: "operator_family_counts",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := newPlanReportInvariantTestReport(t)
			tt.mutate(&report)
			var stdout bytes.Buffer
			err := writePlanReport(&stdout, "yaml", report)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("writePlanReport() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestWritePlanReportRejectsMatchedOperatorIndexInvariantViolation(t *testing.T) {
	report := newPlanReportInvariantTestReport(t)
	matched := []int32{99}
	report.ContractEvalMode = planContractEvaluationModeReportOnly
	report.ContractEvaluations = []planContractEvaluation{{
		Name:     "BadMatchedIndex",
		TargetID: "query/ListSingers",
		Status:   planContractStatusPass,
		Results: []planContractRuleResult{{
			Rule:                   "forbid_operator_family",
			Source:                 "forbid[0]",
			OperatorFamily:         "scan",
			Status:                 planContractStatusPass,
			MatchedOperatorIndexes: &matched,
		}},
	}}
	report.ContractSummary = &planContractEvaluationSummary{
		Status:              planContractStatusPass,
		Contracts:           1,
		Passed:              1,
		EnvironmentWarnings: []string{},
	}
	var stdout bytes.Buffer
	err := writePlanReport(&stdout, "yaml", report)
	if err == nil || !strings.Contains(err.Error(), "matched_operator_indexes contains missing index 99") {
		t.Fatalf("writePlanReport() error = %v, want matched operator index invariant violation", err)
	}
}

func newPlanReportInvariantTestReport(t testing.TB) planReport {
	t.Helper()
	rawPlan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       0,
			DisplayName: "Serialize Result",
			ChildLinks:  []*spannerpb.PlanNode_ChildLink{{ChildIndex: 1, Type: "Input"}},
		},
		{Index: 1, DisplayName: "Scan"},
	}}
	operators := planReportOperators(rawPlan)
	return planReport{
		ReportVersion:    planReportVersionV1Alpha,
		Status:           planReportStatusOK,
		Backend:          "omni",
		BackendIdentity:  planReportBackendIdentity{Kind: "omni", Version: "not_recorded", ImageDigest: "not_recorded", Source: "spanemuboost"},
		ContractEvalMode: planContractEvaluationModeNone,
		Normalization:    defaultPlanReportNormalization(),
		TargetSummary: planReportTargetSummary{
			IncludedCount: 1,
			Planned:       1,
			Excluded:      []planReportExcludedTarget{},
		},
		Queries: []planReportQuery{{
			TargetID:             "query/ListSingers",
			Name:                 "ListSingers",
			Catalog:              "app",
			Scope:                "query",
			Kind:                 "sql",
			Status:               "ok",
			OperatorFamilyCounts: planReportOperatorFamilyCounts(operators),
			NormalizedOperators:  operators,
			OperatorEdges:        planReportOperatorEdges(rawPlan),
			Plan:                 "Serialize Result\n+- Scan",
		}},
	}
}

func TestPlanReportContracts(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{
			{
				Name:  "IndexLookup",
				Scope: "query",
				NormalizedOperators: []planReportOperator{
					{Index: 0, DisplayName: "Scan", Family: "scan"},
				},
			},
			{
				Name:  "SortedLookup",
				Scope: "query",
				NormalizedOperators: []planReportOperator{
					{Index: 0, DisplayName: "Sort", Family: "full_sort"},
					{Index: 1, DisplayName: "Scan", Family: "scan"},
				},
			},
			{
				Name:  "HashAggregate",
				Scope: "query",
				NormalizedOperators: []planReportOperator{
					{Index: 0, DisplayName: "Aggregate", Family: "hash_aggregate"},
				},
			},
			{
				Name:  "StreamAggregate",
				Scope: "query",
				NormalizedOperators: []planReportOperator{
					{Index: 0, DisplayName: "Aggregate", Family: "stream_aggregate"},
				},
			},
			{
				Name:  "UnknownAggregate",
				Scope: "query",
				NormalizedOperators: []planReportOperator{
					{Index: 0, DisplayName: "Aggregate", Family: "aggregate"},
				},
				ClassificationNotes: []planReportDiagnostic{{
					ID:      "aggregate_iterator_type_unknown",
					Message: "Aggregate PlanNode 0 could not be classified.",
				}},
			},
		},
	}
	contracts := planContractsFile{
		Version: "v1alpha-plan-contracts",
		Contracts: []planContract{
			{Name: "IndexLookupPlan", Target: "query/IndexLookup", Use: []string{"no_explicit_sort"}},
			{Name: "SortedLookupPlan", Target: "query/SortedLookup", Use: []string{"no_explicit_sort"}},
			{Name: "NoHashAggregatePlan", Target: "query/HashAggregate", Use: []string{"no_hash_aggregate"}},
			{Name: "NoStreamAggregatePlan", Target: "query/StreamAggregate", Use: []string{"no_stream_aggregate"}},
			{Name: "UnknownHashAggregatePlan", Target: "query/UnknownAggregate", Use: []string{"no_hash_aggregate"}},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	if got, want := report.ContractSummary.Status, planContractStatusFail; got != want {
		t.Fatalf("contract summary status = %q, want %q", got, want)
	}
	if got, want := report.ContractSummary.Passed, 1; got != want {
		t.Fatalf("contract summary passed = %d, want %d", got, want)
	}
	if got, want := report.ContractSummary.Failed, 4; got != want {
		t.Fatalf("contract summary failed = %d, want %d", got, want)
	}
	if got, want := planReportContractViolationCount(report), 4; got != want {
		t.Fatalf("contract violation count = %d, want %d", got, want)
	}
	failed := report.ContractEvaluations[1].Results[0]
	if failed.Status != planContractStatusFail || failed.ObservedCount != 1 || len(failed.Remediation) == 0 {
		t.Fatalf("failed contract result = %+v, want explicit_sort violation with remediation", failed)
	}
	if got, want := planContractMatchedOperatorIndexes(failed), []int32{0}; !reflect.DeepEqual(got, want) {
		t.Fatalf("failed matched operator indexes = %v, want %v", got, want)
	}
	unknown := report.ContractEvaluations[4].Results[0]
	if unknown.Status != planContractStatusFail || unknown.FailureKind != planContractFailureKindClassificationUnknown || unknown.DiagnosticID != "plan.aggregate_classification_unknown" || unknown.ObservedCount != 0 || len(unknown.Remediation) == 0 || unknown.Remediation[0].Kind != "operator_classification_unknown" {
		t.Fatalf("unknown aggregate contract result = %+v, want classification warning failure", unknown)
	}
	if got, want := planContractMatchedOperatorIndexes(unknown), []int32{0}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unknown aggregate matched operator indexes = %v, want %v", got, want)
	}
	if got, want := strings.Join(report.ContractSummary.EnvironmentWarnings, ","), "operator_classification_unknown"; got != want {
		t.Fatalf("contract summary environment warnings = %q, want %q", got, want)
	}
}

func TestPlanReportDirectAggregateForbidFailsOnUnknownClassification(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{{
			Name:   "UnknownAggregate",
			Scope:  "query",
			Status: "ok",
			NormalizedOperators: []planReportOperator{{
				Index:       0,
				DisplayName: "Aggregate",
				Family:      "aggregate",
			}},
			ClassificationNotes: []planReportDiagnostic{{
				ID:      "aggregate_iterator_type_unknown",
				Message: "Aggregate PlanNode 0 could not be classified.",
			}},
		}},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{{
			Name:   "NoHashAggregate",
			Target: "query/UnknownAggregate",
			Forbid: []planContractPredicate{{
				OperatorFamily: "hash_aggregate",
				MaxCount:       0,
			}},
		}},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	result := report.ContractEvaluations[0].Results[0]
	if result.Status != planContractStatusFail || result.FailureKind != planContractFailureKindClassificationUnknown || result.DiagnosticID != "plan.aggregate_classification_unknown" {
		t.Fatalf("direct aggregate forbid result = %+v, want classification_unknown failure", result)
	}
	if got, want := planContractMatchedOperatorIndexes(result), []int32{0}; !reflect.DeepEqual(got, want) {
		t.Fatalf("direct aggregate forbid matched indexes = %v, want %v", got, want)
	}
}

func TestPlanReportJoinContracts(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{
			{Name: "HashJoin", Scope: "query", NormalizedOperators: []planReportOperator{{Index: 0, DisplayName: "Hash Join", Family: "hash_join"}}},
			{
				Name:  "PushBroadcastHashJoin",
				Scope: "query",
				NormalizedOperators: []planReportOperator{
					{Index: 0, DisplayName: "Push Broadcast Hash Join", Family: "push_broadcast_hash_join"},
					{Index: 9, DisplayName: "Hash Join", Family: "push_broadcast_hash_join_internal_hash_join"},
				},
			},
			{Name: "ApplyJoin", Scope: "query", NormalizedOperators: []planReportOperator{{Index: 0, DisplayName: "Cross Apply", Family: "apply_join"}}},
			{Name: "SemiApply", Scope: "query", NormalizedOperators: []planReportOperator{{Index: 0, DisplayName: "Semi Apply", Family: "semi_apply"}}},
			{Name: "AntiSemiApply", Scope: "query", NormalizedOperators: []planReportOperator{{Index: 0, DisplayName: "Anti-Semi Apply", Family: "anti_semi_apply"}}},
			{Name: "DistributedCrossApply", Scope: "query", NormalizedOperators: []planReportOperator{{Index: 0, DisplayName: "Distributed Cross Apply", Family: "distributed_cross_apply"}}},
			{Name: "DistributedSemiApply", Scope: "query", NormalizedOperators: []planReportOperator{{Index: 0, DisplayName: "Distributed Semi Apply", Family: "distributed_semi_apply"}}},
			{Name: "DistributedAntiSemiApply", Scope: "query", NormalizedOperators: []planReportOperator{{Index: 0, DisplayName: "Distributed Anti Semi Apply", Family: "distributed_anti_semi_apply"}}},
			{Name: "MergeJoin", Scope: "query", NormalizedOperators: []planReportOperator{{Index: 0, DisplayName: "Merge Join", Family: "merge_join"}}},
		},
	}
	contracts := planContractsFile{
		Version: "v1alpha-plan-contracts",
		Contracts: []planContract{
			{Name: "NoHashJoin", Target: "query/HashJoin", Use: []string{"no_hash_join"}},
			{Name: "NoStandaloneHashJoinAllowsPushBroadcast", Target: "query/PushBroadcastHashJoin", Use: []string{"no_standalone_hash_join"}},
			{Name: "NoHashJoinRejectsPushBroadcast", Target: "query/PushBroadcastHashJoin", Use: []string{"no_hash_join"}},
			{Name: "NoPushBroadcastHashJoin", Target: "query/PushBroadcastHashJoin", Use: []string{"no_push_broadcast_hash_join"}},
			{Name: "NoApplyJoin", Target: "query/ApplyJoin", Use: []string{"no_apply_join"}},
			{Name: "NoStandaloneApplyJoinAllowsDistributedCrossApply", Target: "query/DistributedCrossApply", Use: []string{"no_standalone_apply_join"}},
			{Name: "NoApplyJoinRejectsDistributedCrossApply", Target: "query/DistributedCrossApply", Use: []string{"no_apply_join"}},
			{Name: "NoDistributedCrossApply", Target: "query/DistributedCrossApply", Use: []string{"no_distributed_cross_apply"}},
			{Name: "NoApplyJoinRejectsSemiApply", Target: "query/SemiApply", Use: []string{"no_apply_join"}},
			{Name: "NoStandaloneApplyJoinRejectsSemiApply", Target: "query/SemiApply", Use: []string{"no_standalone_apply_join"}},
			{Name: "NoApplyJoinRejectsAntiSemiApply", Target: "query/AntiSemiApply", Use: []string{"no_apply_join"}},
			{Name: "NoStandaloneApplyJoinRejectsAntiSemiApply", Target: "query/AntiSemiApply", Use: []string{"no_standalone_apply_join"}},
			{Name: "NoApplyJoinRejectsDistributedSemiApply", Target: "query/DistributedSemiApply", Use: []string{"no_apply_join"}},
			{Name: "NoStandaloneApplyJoinAllowsDistributedSemiApply", Target: "query/DistributedSemiApply", Use: []string{"no_standalone_apply_join"}},
			{Name: "NoApplyJoinRejectsDistributedAntiSemiApply", Target: "query/DistributedAntiSemiApply", Use: []string{"no_apply_join"}},
			{Name: "NoStandaloneApplyJoinAllowsDistributedAntiSemiApply", Target: "query/DistributedAntiSemiApply", Use: []string{"no_standalone_apply_join"}},
			{Name: "NoMergeJoin", Target: "query/MergeJoin", Use: []string{"no_merge_join"}},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	if got, want := report.ContractSummary.Failed, 13; got != want {
		t.Fatalf("contract summary failed = %d, want %d", got, want)
	}
	if got, want := planReportContractViolationCount(report), 13; got != want {
		t.Fatalf("contract violation count = %d, want %d", got, want)
	}
	noHash := report.ContractEvaluations[0].Results[0]
	if got, want := noHash.Source, "use/no_hash_join"; got != want {
		t.Fatalf("predefined source = %q, want %q", got, want)
	}
	if got, want := noHash.Predefined, "no_hash_join"; got != want {
		t.Fatalf("predefined = %q, want %q", got, want)
	}
}

func TestPlanReportJoinContractsFailOnUnknownClassification(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{
			{
				Name:   "GenericJoin",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{{
					Index:       40,
					DisplayName: "Join",
					Family:      "join",
				}},
				ClassificationNotes: []planReportDiagnostic{{
					ID:      "join_family_unknown",
					Message: "Join-like PlanNode 40 could not be classified.",
				}},
			},
		},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{
			{Name: "NoHashJoin", Target: "query/GenericJoin", Use: []string{"no_hash_join"}},
			{Name: "NoMergeJoin", Target: "query/GenericJoin", Forbid: []planContractPredicate{{OperatorFamily: "merge_join"}}},
			{Name: "ForbidGenericJoin", Target: "query/GenericJoin", Forbid: []planContractPredicate{{OperatorFamily: "join"}}},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	for i, evaluation := range report.ContractEvaluations[:2] {
		if evaluation.Status != planContractStatusFail {
			t.Fatalf("join-specific contract %d status = %q, want fail", i, evaluation.Status)
		}
		result := evaluation.Results[0]
		if result.FailureKind != planContractFailureKindClassificationUnknown || result.DiagnosticID != "plan.join_classification_unknown" {
			t.Fatalf("join-specific result = %+v, want join classification_unknown", result)
		}
		if got, want := planContractMatchedOperatorIndexes(result), []int32{40}; !reflect.DeepEqual(got, want) {
			t.Fatalf("join-specific matched indexes = %v, want %v", got, want)
		}
	}
	generic := report.ContractEvaluations[2].Results[0]
	if generic.FailureKind != planContractFailureKindViolation || generic.DiagnosticID != "" {
		t.Fatalf("generic join forbid result = %+v, want regular violation", generic)
	}
	if got, want := strings.Join(report.ContractSummary.EnvironmentWarnings, ","), "operator_classification_unknown"; got != want {
		t.Fatalf("contract summary environment warnings = %q, want %q", got, want)
	}
}

func TestPlanReportSortContractsDistinguishFullAndMinorSort(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{
			{
				Name:   "FullSortedLookup",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{{
					Index:       10,
					DisplayName: "Sort",
					Family:      "full_sort",
				}},
			},
			{
				Name:   "MinorSortedLookup",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{{
					Index:       20,
					DisplayName: "Minor Sort",
					Family:      "minor_sort",
				}},
			},
		},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{
			{Name: "NoFullSortFailsOnFullSort", Target: "query/FullSortedLookup", Use: []string{"no_full_sort"}},
			{Name: "NoFullSortAllowsMinorSort", Target: "query/MinorSortedLookup", Use: []string{"no_full_sort"}},
			{Name: "NoMinorSortFailsOnMinorSort", Target: "query/MinorSortedLookup", Use: []string{"no_minor_sort"}},
			{Name: "NoExplicitSortFailsOnMinorSort", Target: "query/MinorSortedLookup", Use: []string{"no_explicit_sort"}},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	tests := []struct {
		index       int
		wantStatus  string
		wantFamily  string
		wantIndexes []int32
	}{
		{index: 0, wantStatus: planContractStatusFail, wantFamily: "full_sort", wantIndexes: []int32{10}},
		{index: 1, wantStatus: planContractStatusPass, wantFamily: "full_sort", wantIndexes: []int32{}},
		{index: 2, wantStatus: planContractStatusFail, wantFamily: "minor_sort", wantIndexes: []int32{20}},
		{index: 3, wantStatus: planContractStatusFail, wantFamily: "explicit_sort", wantIndexes: []int32{20}},
	}
	for _, tt := range tests {
		evaluation := report.ContractEvaluations[tt.index]
		if evaluation.Status != tt.wantStatus {
			t.Fatalf("evaluation[%d] status = %q, want %q", tt.index, evaluation.Status, tt.wantStatus)
		}
		result := evaluation.Results[0]
		if result.OperatorFamily != tt.wantFamily {
			t.Fatalf("evaluation[%d] operator_family = %q, want %q", tt.index, result.OperatorFamily, tt.wantFamily)
		}
		if got := planContractMatchedOperatorIndexes(result); !reflect.DeepEqual(got, tt.wantIndexes) {
			t.Fatalf("evaluation[%d] matched operator indexes = %v, want %v", tt.index, got, tt.wantIndexes)
		}
	}
}

func TestPlanReportBlockingOperatorUnderLimitContract(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{
			{
				Name:   "LimitOverHashAggregate",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{
					{Index: 0, DisplayName: "Global Limit", Family: "limit", DescendantIndexes: []int32{1, 2}},
					{Index: 1, DisplayName: "Hash Aggregate", Family: "hash_aggregate", DescendantIndexes: []int32{2}},
					{Index: 2, DisplayName: "Table Scan", Family: "scan"},
				},
			},
			{
				Name:   "LimitOverSortLimit",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{
					{Index: 10, DisplayName: "Global Limit", Family: "limit", DescendantIndexes: []int32{11, 12}},
					{Index: 11, DisplayName: "Sort Limit", Family: "full_sort", DescendantIndexes: []int32{12}},
					{Index: 12, DisplayName: "Table Scan", Family: "scan"},
				},
			},
			{
				Name:   "SortLimitOverHashAggregate",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{
					{Index: 20, DisplayName: "Sort Limit", Family: "full_sort", DescendantIndexes: []int32{21, 22}},
					{Index: 21, DisplayName: "Hash Aggregate", Family: "hash_aggregate", DescendantIndexes: []int32{22}},
					{Index: 22, DisplayName: "Table Scan", Family: "scan"},
				},
			},
			{
				Name:   "LimitOverMinorSortLimit",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{
					{Index: 30, DisplayName: "Global Limit", Family: "limit", DescendantIndexes: []int32{31, 32}},
					{Index: 31, DisplayName: "Minor Sort Limit", Family: "minor_sort", DescendantIndexes: []int32{32}},
					{Index: 32, DisplayName: "Table Scan", Family: "scan"},
				},
			},
			{
				Name:   "HashAggregateWithoutLimit",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{
					{Index: 40, DisplayName: "Hash Aggregate", Family: "hash_aggregate", DescendantIndexes: []int32{41}},
					{Index: 41, DisplayName: "Table Scan", Family: "scan"},
				},
			},
		},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{
			{Name: "LimitOverHashAggregate", Target: "query/LimitOverHashAggregate", Use: []string{"no_blocking_operator_under_limit"}},
			{Name: "LimitOverSortLimit", Target: "query/LimitOverSortLimit", Use: []string{"no_blocking_operator_under_limit"}},
			{Name: "SortLimitOverHashAggregate", Target: "query/SortLimitOverHashAggregate", Use: []string{"no_blocking_operator_under_limit"}},
			{Name: "LimitOverMinorSortLimit", Target: "query/LimitOverMinorSortLimit", Use: []string{"no_blocking_operator_under_limit"}},
			{Name: "HashAggregateWithoutLimit", Target: "query/HashAggregateWithoutLimit", Use: []string{"no_blocking_operator_under_limit"}},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	tests := []struct {
		index       int
		wantStatus  string
		wantIndexes []int32
	}{
		{index: 0, wantStatus: planContractStatusFail, wantIndexes: []int32{1}},
		{index: 1, wantStatus: planContractStatusFail, wantIndexes: []int32{11}},
		{index: 2, wantStatus: planContractStatusFail, wantIndexes: []int32{21}},
		{index: 3, wantStatus: planContractStatusPass, wantIndexes: []int32{}},
		{index: 4, wantStatus: planContractStatusPass, wantIndexes: []int32{}},
	}
	for _, tt := range tests {
		evaluation := report.ContractEvaluations[tt.index]
		if evaluation.Status != tt.wantStatus {
			t.Fatalf("evaluation[%d] status = %q, want %q", tt.index, evaluation.Status, tt.wantStatus)
		}
		result := evaluation.Results[0]
		if got, want := result.Rule, "forbid_blocking_operator_under_limit"; got != want {
			t.Fatalf("evaluation[%d] rule = %q, want %q", tt.index, got, want)
		}
		if got, want := result.Source, "use/no_blocking_operator_under_limit"; got != want {
			t.Fatalf("evaluation[%d] source = %q, want %q", tt.index, got, want)
		}
		if got, want := result.Predefined, "no_blocking_operator_under_limit"; got != want {
			t.Fatalf("evaluation[%d] predefined = %q, want %q", tt.index, got, want)
		}
		if result.OperatorFamily != "" {
			t.Fatalf("evaluation[%d] operator_family = %q, want empty for topology rule", tt.index, result.OperatorFamily)
		}
		if got := planContractMatchedOperatorIndexes(result); !reflect.DeepEqual(got, tt.wantIndexes) {
			t.Fatalf("evaluation[%d] matched operator indexes = %v, want %v", tt.index, got, tt.wantIndexes)
		}
	}
	if got, want := report.ContractSummary.Failed, 3; got != want {
		t.Fatalf("contract summary failed = %d, want %d", got, want)
	}
}

func TestPlanReportFullScanContract(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{
			{
				Name:   "FullScanLookup",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{
					{Index: 7, DisplayName: "Index Scan on FooByTimestamp", Family: "scan", ScanType: "index_scan", ScanTarget: "FooByTimestamp", FullScan: true},
				},
			},
			{
				Name:   "SeekLookup",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{
					{Index: 8, DisplayName: "Index Scan on FooByTimestamp", Family: "scan", ScanType: "index_scan", ScanTarget: "FooByTimestamp", SeekableKeySize: "2"},
				},
			},
		},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{
			{Name: "NoFullScanRejectsFullScan", Target: "query/FullScanLookup", Use: []string{"no_full_scan"}},
			{Name: "NoFullScanAllowsSeekScan", Target: "query/SeekLookup", Use: []string{"no_full_scan"}},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	failing := report.ContractEvaluations[0]
	if got, want := failing.Status, planContractStatusFail; got != want {
		t.Fatalf("full scan evaluation status = %q, want %q", got, want)
	}
	result := failing.Results[0]
	if got, want := result.Rule, "forbid_full_scan"; got != want {
		t.Fatalf("full scan rule = %q, want %q", got, want)
	}
	if got, want := result.Source, "use/no_full_scan"; got != want {
		t.Fatalf("full scan source = %q, want %q", got, want)
	}
	if got, want := result.Predefined, "no_full_scan"; got != want {
		t.Fatalf("full scan predefined = %q, want %q", got, want)
	}
	if result.OperatorFamily != "" {
		t.Fatalf("full scan operator_family = %q, want empty for metadata rule", result.OperatorFamily)
	}
	if got := planContractMatchedOperatorIndexes(result); !reflect.DeepEqual(got, []int32{7}) {
		t.Fatalf("full scan matched operator indexes = %v, want [7]", got)
	}
	passing := report.ContractEvaluations[1]
	if got, want := passing.Status, planContractStatusPass; got != want {
		t.Fatalf("seek scan evaluation status = %q, want %q", got, want)
	}
	if got := planContractMatchedOperatorIndexes(passing.Results[0]); !reflect.DeepEqual(got, []int32{}) {
		t.Fatalf("seek scan matched operator indexes = %v, want []", got)
	}
	if got, want := report.ContractSummary.Failed, 1; got != want {
		t.Fatalf("contract summary failed = %d, want %d", got, want)
	}
}

func TestPlanReportTimestampConditionContract(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{
			{
				Name:   "CommitTimestampRecentRead",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{
					{Index: 4, DisplayName: "Table Scan on Foo", Family: "scan", ScanType: "table_scan", ScanTarget: "Foo", FullScan: true},
				},
				OperatorEdges: []planReportOperatorEdge{
					{ParentIndex: 4, ChildIndex: 10, Type: "Timestamp Condition"},
				},
			},
			{
				Name:   "RegularTimestampRead",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{
					{Index: 5, DisplayName: "Table Scan on Foo", Family: "scan", ScanType: "table_scan", ScanTarget: "Foo", FullScan: true},
				},
				OperatorEdges: []planReportOperatorEdge{
					{ParentIndex: 3, ChildIndex: 11, Type: "Residual Condition"},
				},
			},
		},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{
			{Name: "RequiresTimestampConditionPass", Target: "query/CommitTimestampRecentRead", Use: []string{"require_timestamp_condition"}},
			{Name: "RequiresTimestampConditionFail", Target: "query/RegularTimestampRead", Use: []string{"require_timestamp_condition"}},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	passing := report.ContractEvaluations[0]
	if got, want := passing.Status, planContractStatusPass; got != want {
		t.Fatalf("timestamp condition pass status = %q, want %q", got, want)
	}
	passResult := passing.Results[0]
	if got, want := passResult.Rule, "require_timestamp_condition"; got != want {
		t.Fatalf("timestamp condition pass rule = %q, want %q", got, want)
	}
	if got, want := passResult.Source, "use/require_timestamp_condition"; got != want {
		t.Fatalf("timestamp condition pass source = %q, want %q", got, want)
	}
	if got, want := passResult.Predefined, "require_timestamp_condition"; got != want {
		t.Fatalf("timestamp condition pass predefined = %q, want %q", got, want)
	}
	if got := planContractMatchedOperatorIndexes(passResult); !reflect.DeepEqual(got, []int32{4}) {
		t.Fatalf("timestamp condition matched operator indexes = %v, want [4]", got)
	}
	failing := report.ContractEvaluations[1]
	if got, want := failing.Status, planContractStatusFail; got != want {
		t.Fatalf("timestamp condition fail status = %q, want %q", got, want)
	}
	failResult := failing.Results[0]
	if got := planContractMatchedOperatorIndexes(failResult); !reflect.DeepEqual(got, []int32{}) {
		t.Fatalf("timestamp condition fail matched operator indexes = %v, want []", got)
	}
	if got, want := report.ContractSummary.Failed, 1; got != want {
		t.Fatalf("contract summary failed = %d, want %d", got, want)
	}
}

func TestPlanReportFullScanWithoutTimestampConditionContract(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{
			{
				Name:   "UnprunedFullScan",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{
					{Index: 5, DisplayName: "Table Scan on Foo", Family: "scan", ScanType: "table_scan", ScanTarget: "Foo", FullScan: true},
				},
				OperatorEdges: []planReportOperatorEdge{
					{ParentIndex: 3, ChildIndex: 11, Type: "Residual Condition"},
				},
			},
			{
				Name:   "TimestampPrunedFullScan",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{
					{Index: 6, DisplayName: "Table Scan on Foo", Family: "scan", ScanType: "table_scan", ScanTarget: "Foo", FullScan: true},
				},
				OperatorEdges: []planReportOperatorEdge{
					{ParentIndex: 6, ChildIndex: 12, Type: "Timestamp Condition"},
				},
			},
			{
				Name:   "SeekScan",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{
					{Index: 7, DisplayName: "Index Scan on FooByUpdatedAt", Family: "scan", ScanType: "index_scan", ScanTarget: "FooByUpdatedAt", SeekableKeySize: "1"},
				},
			},
		},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{
			{Name: "RejectUnprunedFullScan", Target: "query/UnprunedFullScan", Use: []string{"no_full_scan_without_timestamp_condition"}},
			{Name: "AllowTimestampPrunedFullScan", Target: "query/TimestampPrunedFullScan", Use: []string{"no_full_scan_without_timestamp_condition"}},
			{Name: "AllowSeekScan", Target: "query/SeekScan", Use: []string{"no_full_scan_without_timestamp_condition"}},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	failing := report.ContractEvaluations[0]
	if got, want := failing.Status, planContractStatusFail; got != want {
		t.Fatalf("unpruned full scan status = %q, want %q", got, want)
	}
	result := failing.Results[0]
	if got, want := result.Rule, "forbid_full_scan_without_timestamp_condition"; got != want {
		t.Fatalf("unpruned full scan rule = %q, want %q", got, want)
	}
	if got, want := result.Source, "use/no_full_scan_without_timestamp_condition"; got != want {
		t.Fatalf("unpruned full scan source = %q, want %q", got, want)
	}
	if got, want := result.Predefined, "no_full_scan_without_timestamp_condition"; got != want {
		t.Fatalf("unpruned full scan predefined = %q, want %q", got, want)
	}
	if got := planContractMatchedOperatorIndexes(result); !reflect.DeepEqual(got, []int32{5}) {
		t.Fatalf("unpruned full scan matched operator indexes = %v, want [5]", got)
	}
	for i, wantName := range []string{"AllowTimestampPrunedFullScan", "AllowSeekScan"} {
		evaluation := report.ContractEvaluations[i+1]
		if got, want := evaluation.Name, wantName; got != want {
			t.Fatalf("evaluation[%d] name = %q, want %q", i+1, got, want)
		}
		if got, want := evaluation.Status, planContractStatusPass; got != want {
			t.Fatalf("%s status = %q, want %q", wantName, got, want)
		}
		if got := planContractMatchedOperatorIndexes(evaluation.Results[0]); !reflect.DeepEqual(got, []int32{}) {
			t.Fatalf("%s matched operator indexes = %v, want []", wantName, got)
		}
	}
	if got, want := report.ContractSummary.Failed, 1; got != want {
		t.Fatalf("contract summary failed = %d, want %d", got, want)
	}
}

func TestPlanContractPredefinedResultSourcesMatch(t *testing.T) {
	for _, predefined := range planContractPredefinedNames() {
		t.Run(predefined, func(t *testing.T) {
			report := planReport{
				Status:  "ok",
				Backend: "omni",
				Queries: []planReportQuery{{
					Name:   "CleanLookup",
					Scope:  "query",
					Status: "ok",
				}},
			}
			contracts := planContractsFile{
				Version: planContractFileVersionV1Alpha,
				Contracts: []planContract{{
					Name:   "Contract",
					Target: "query/CleanLookup",
					Use:    []string{predefined},
				}},
			}
			if err := applyPlanContracts(&report, contracts); err != nil {
				t.Fatalf("applyPlanContracts() error = %v", err)
			}
			results := report.ContractEvaluations[0].Results
			if len(results) == 0 {
				t.Fatalf("predefined %q produced no rule results", predefined)
			}
			for _, result := range results {
				if got, want := result.Source, "use/"+predefined; got != want {
					t.Fatalf("result source = %q, want %q", got, want)
				}
				if got, want := result.Predefined, predefined; got != want {
					t.Fatalf("result predefined = %q, want %q", got, want)
				}
			}
		})
	}
}

func TestPlanContractDirectResultSourcesDoNotSetPredefined(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{{
			Name:   "CleanLookup",
			Scope:  "query",
			Status: "ok",
		}},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{
			{
				Name:   "DirectForbid",
				Target: "query/CleanLookup",
				Forbid: []planContractPredicate{{OperatorFamily: "explicit_sort"}},
			},
			{
				Name:   "CELTrue",
				Target: "query/CleanLookup",
				CEL:    "true",
			},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	forbid := report.ContractEvaluations[0].Results[0]
	if got, want := forbid.Source, "forbid[0]"; got != want {
		t.Fatalf("forbid source = %q, want %q", got, want)
	}
	if forbid.Predefined != "" {
		t.Fatalf("forbid predefined = %q, want empty", forbid.Predefined)
	}
	if got, want := planContractMatchedOperatorIndexes(forbid), []int32{}; !reflect.DeepEqual(got, want) {
		t.Fatalf("forbid matched operator indexes = %v, want %v", got, want)
	}
	cel := report.ContractEvaluations[1].Results[0]
	if got, want := cel.Source, "cel"; got != want {
		t.Fatalf("cel source = %q, want %q", got, want)
	}
	if cel.Predefined != "" {
		t.Fatalf("cel predefined = %q, want empty", cel.Predefined)
	}
	if cel.MatchedOperatorIndexes != nil {
		t.Fatalf("cel matched operator indexes = %v, want absent", *cel.MatchedOperatorIndexes)
	}
}

func TestPlanContractCELFailureUsesViolationOnly(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{{
			Name:   "CleanLookup",
			Scope:  "query",
			Status: "ok",
		}},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{{
			Name:   "CELFalse",
			Target: "query/CleanLookup",
			CEL:    "false",
		}},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	result := report.ContractEvaluations[0].Results[0]
	if got, want := result.Rule, "cel"; got != want {
		t.Fatalf("CEL result rule = %q, want %q", got, want)
	}
	if got, want := result.Status, planContractStatusFail; got != want {
		t.Fatalf("CEL result status = %q, want %q", got, want)
	}
	if got, want := result.FailureKind, planContractFailureKindViolation; got != want {
		t.Fatalf("CEL result failure_kind = %q, want %q", got, want)
	}
	if result.DiagnosticID != "" {
		t.Fatalf("CEL result diagnostic_id = %q, want empty", result.DiagnosticID)
	}
}

func TestPlanContractDirectForbidSourceIndexesUseYAMLOrder(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{{
			Name:   "Lookup",
			Scope:  "query",
			Status: "ok",
		}},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{{
			Name:   "DirectForbid",
			Target: "query/Lookup",
			Forbid: []planContractPredicate{
				{OperatorFamily: "full_sort"},
				{OperatorFamily: "minor_sort"},
			},
		}},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	results := report.ContractEvaluations[0].Results
	if got, want := len(results), 2; got != want {
		t.Fatalf("result count = %d, want %d", got, want)
	}
	for i, result := range results {
		if got, want := result.Source, "forbid["+strconv.Itoa(i)+"]"; got != want {
			t.Fatalf("result[%d] source = %q, want %q", i, got, want)
		}
	}
}

func TestPlanContractStatusInvariants(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{
			{
				Name:   "CleanLookup",
				Scope:  "query",
				Status: "ok",
			},
			{
				Name:   "SortedLookup",
				Scope:  "query",
				Status: "ok",
				NormalizedOperators: []planReportOperator{{
					Index:       0,
					DisplayName: "Sort",
					Family:      "full_sort",
				}},
			},
		},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{
			{Name: "Pass", Target: "query/CleanLookup", Use: []string{"no_explicit_sort"}},
			{Name: "Fail", Target: "query/SortedLookup", Use: []string{"no_explicit_sort"}},
			{Name: "Missing", Target: "query/MissingLookup", Use: []string{"no_explicit_sort"}},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	if got, want := report.ContractSummary.Status, planContractStatusFail; got != want {
		t.Fatalf("summary status = %q, want %q", got, want)
	}
	if got, want := report.ContractSummary.Passed, 1; got != want {
		t.Fatalf("summary passed = %d, want %d", got, want)
	}
	if got, want := report.ContractSummary.Failed, 1; got != want {
		t.Fatalf("summary failed = %d, want %d", got, want)
	}
	if got, want := report.ContractSummary.NotEvaluated, 1; got != want {
		t.Fatalf("summary not_evaluated = %d, want %d", got, want)
	}
	for _, evaluation := range report.ContractEvaluations {
		hasFailedRule := false
		for _, result := range evaluation.Results {
			if result.Status == planContractStatusFail {
				hasFailedRule = true
			}
		}
		switch evaluation.Status {
		case planContractStatusPass:
			if hasFailedRule || len(evaluation.Results) == 0 {
				t.Fatalf("pass evaluation invariant failed: %+v", evaluation)
			}
		case planContractStatusFail:
			if !hasFailedRule {
				t.Fatalf("fail evaluation invariant failed: %+v", evaluation)
			}
		case planContractStatusNotEvaluated:
			if len(evaluation.Results) != 0 {
				t.Fatalf("not_evaluated evaluation invariant failed: %+v", evaluation)
			}
		default:
			t.Fatalf("unexpected evaluation status %q", evaluation.Status)
		}
	}
}

func TestPlanContractUnknownFamilyCanBeForbiddenDirectly(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{{
			Name:   "UnknownOperator",
			Scope:  "query",
			Status: "ok",
			NormalizedOperators: []planReportOperator{{
				Index:       0,
				DisplayName: "New Fancy Operator",
				Family:      "unknown",
			}},
			ClassificationNotes: []planReportDiagnostic{{
				ID:      "operator_family_unknown",
				Message: "PlanNode 0 could not be classified into a known operator family.",
			}},
		}},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{
			{
				Name:   "NoExplicitSortIgnoresUnknown",
				Target: "query/UnknownOperator",
				Use:    []string{"no_explicit_sort"},
			},
			{
				Name:   "NoUnknownOperator",
				Target: "query/UnknownOperator",
				Forbid: []planContractPredicate{{OperatorFamily: "unknown"}},
			},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	if got, want := report.ContractEvaluations[0].Status, planContractStatusPass; got != want {
		t.Fatalf("predefined contract status = %q, want %q", got, want)
	}
	direct := report.ContractEvaluations[1]
	if got, want := direct.Status, planContractStatusFail; got != want {
		t.Fatalf("direct unknown contract status = %q, want %q", got, want)
	}
	result := direct.Results[0]
	if result.OperatorFamily != "unknown" || result.ObservedCount != 1 || result.FailureKind != planContractFailureKindViolation {
		t.Fatalf("direct unknown result = %+v, want violation on one unknown operator", result)
	}
	if got, want := planContractMatchedOperatorIndexes(result), []int32{0}; !reflect.DeepEqual(got, want) {
		t.Fatalf("direct unknown matched operator indexes = %v, want %v", got, want)
	}
	if got, want := strings.Join(report.ContractSummary.EnvironmentWarnings, ","), "operator_classification_unknown"; got != want {
		t.Fatalf("contract summary environment warnings = %q, want %q", got, want)
	}
}

func TestPlanReportOperatorsClassifyPushBroadcastInternalHashJoin(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       1,
			DisplayName: "Push Broadcast Hash Join",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "Map", ChildIndex: 8},
				{Type: "Input", ChildIndex: 2},
			},
		},
		{
			Index:       2,
			DisplayName: "Hash Join",
		},
		{
			Index:       8,
			DisplayName: "Serialize Result",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "", ChildIndex: 9},
			},
		},
		{
			Index:       9,
			DisplayName: "Hash Join",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "Build", ChildIndex: 10},
				{Type: "Probe", ChildIndex: 30},
			},
		},
		{
			Index:       10,
			DisplayName: "DataBlockToRow",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "", ChildIndex: 11},
			},
		},
		{
			Index:       11,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_method": "Batch",
				"scan_target": "$v2",
				"scan_type":   "BatchScan",
			}),
		},
		{
			Index:       30,
			DisplayName: "Hash Join",
		},
	}}
	operators := planReportOperators(plan)
	byIndex := map[int32]string{}
	for _, operator := range operators {
		byIndex[operator.Index] = operator.Family
	}
	if got, want := byIndex[1], "push_broadcast_hash_join"; got != want {
		t.Fatalf("push broadcast family = %q, want %q", got, want)
	}
	if got, want := byIndex[2], "hash_join"; got != want {
		t.Fatalf("non-map hash join family = %q, want %q", got, want)
	}
	if got, want := byIndex[9], "push_broadcast_hash_join_internal_hash_join"; got != want {
		t.Fatalf("push broadcast internal hash join family = %q, want %q", got, want)
	}
	if got, want := byIndex[30], "hash_join"; got != want {
		t.Fatalf("nested regular hash join family = %q, want %q", got, want)
	}
}

func TestPlanReportOperatorTopology(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       0,
			DisplayName: "Serialize Result",
			ChildLinks:  []*spannerpb.PlanNode_ChildLink{{ChildIndex: 1, Type: "Input"}},
		},
		{
			Index:       1,
			DisplayName: "Hash Join",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{ChildIndex: 2, Type: "LeftInput", Variable: "left"},
				{ChildIndex: 3, Type: "RightInput", Variable: "right"},
			},
		},
		{Index: 2, DisplayName: "Scan"},
		{Index: 3, DisplayName: "Scan"},
	}}

	operators := planReportOperators(plan)
	if got, want := operators[0].ChildIndexes, []int32{1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("root child indexes = %v, want %v", got, want)
	}
	if got, want := operators[0].DescendantIndexes, []int32{1, 2, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("root descendant indexes = %v, want %v", got, want)
	}
	if got, want := operators[0].SubtreeFamilyCounts["serialize_result"], 1; got != want {
		t.Fatalf("root serialize_result subtree count = %d, want %d", got, want)
	}
	if got, want := operators[0].SubtreeFamilyCounts["hash_join"], 1; got != want {
		t.Fatalf("root hash_join subtree count = %d, want %d", got, want)
	}
	if got, want := operators[0].SubtreeFamilyCounts["scan"], 2; got != want {
		t.Fatalf("root scan subtree count = %d, want %d", got, want)
	}

	edges := planReportOperatorEdges(plan)
	if got, want := len(edges), 3; got != want {
		t.Fatalf("operator edge count = %d, want %d", got, want)
	}
	if got, want := edges[1].Variable, "left"; got != want {
		t.Fatalf("operator edge variable = %q, want %q", got, want)
	}
}

func TestPlanReportOperatorsClassifyScalarKindNodesAsScalar(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       0,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Serialize Result",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{ChildIndex: 1, Type: "Input"},
				{ChildIndex: 2},
			},
		},
		{Index: 1, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "New Fancy Operator"},
		{Index: 2, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Reference"},
		{Index: 3, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Function"},
		// Array Subquery and Scalar Subquery nodes are kind SCALAR on
		// Spanner Omni but still carry concrete operator families.
		{Index: 4, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Array Subquery"},
		{Index: 5, Kind: spannerpb.PlanNode_SCALAR, DisplayName: "Scalar Subquery"},
	}}
	operators := planReportOperators(plan)
	byIndex := map[int32]string{}
	for _, operator := range operators {
		byIndex[operator.Index] = operator.Family
	}
	if got, want := byIndex[0], "serialize_result"; got != want {
		t.Fatalf("relational root family = %q, want %q", got, want)
	}
	if got, want := byIndex[1], "unknown"; got != want {
		t.Fatalf("unclassified relational family = %q, want %q", got, want)
	}
	if got, want := byIndex[2], "scalar"; got != want {
		t.Fatalf("scalar Reference family = %q, want %q", got, want)
	}
	if got, want := byIndex[3], "scalar"; got != want {
		t.Fatalf("scalar Function family = %q, want %q", got, want)
	}
	if got, want := byIndex[4], "array_subquery"; got != want {
		t.Fatalf("scalar-kind Array Subquery family = %q, want %q", got, want)
	}
	if got, want := byIndex[5], "scalar_subquery"; got != want {
		t.Fatalf("scalar-kind Scalar Subquery family = %q, want %q", got, want)
	}
	counts := planReportOperatorFamilyCounts(operators)
	if got, want := counts["scalar"], 2; got != want {
		t.Fatalf("scalar family count = %d, want %d", got, want)
	}
	if got, want := counts["unknown"], 1; got != want {
		t.Fatalf("unknown family count = %d, want %d", got, want)
	}
	warnings := planReportClassificationWarnings(operators)
	if got, want := len(warnings), 1; got != want {
		t.Fatalf("classification warning count = %d, want %d: %+v", got, want, warnings)
	}
	if warnings[0].ID != "operator_family_unknown" || !strings.Contains(warnings[0].Message, "PlanNode 1 ") {
		t.Fatalf("classification warning = %+v, want operator_family_unknown for PlanNode 1", warnings[0])
	}
}

// validatePlanReportQueryTopology must accept a report whose subtree contains
// a stream-blocking operator. The validator previously recomputed only the
// derived explicit_sort family and rejected plans containing, for example, a
// hash aggregate because the produced subtree_family_counts also derive
// blocking_operator.
func TestValidatePlanReportQueryTopologyAcceptsBlockingOperatorSubtree(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       0,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Serialize Result",
			ChildLinks:  []*spannerpb.PlanNode_ChildLink{{ChildIndex: 1}},
		},
		{
			Index:       1,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Aggregate",
			Metadata: mustStructPB(t, map[string]interface{}{
				"iterator_type": "Hash",
			}),
			ChildLinks: []*spannerpb.PlanNode_ChildLink{{ChildIndex: 2}},
		},
		{Index: 2, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Table Scan"},
	}}
	operators := planReportOperators(plan)
	query := planReportQuery{
		Name:                 "HashAggregateQuery",
		Scope:                "query",
		Status:               "ok",
		NormalizedOperators:  operators,
		OperatorEdges:        planReportOperatorEdges(plan),
		OperatorFamilyCounts: planReportOperatorFamilyCounts(operators),
	}
	if got, want := query.OperatorFamilyCounts["hash_aggregate"], 1; got != want {
		t.Fatalf("hash_aggregate count = %d, want %d", got, want)
	}
	if got, want := query.OperatorFamilyCounts["blocking_operator"], 1; got != want {
		t.Fatalf("blocking_operator derived count = %d, want %d", got, want)
	}
	if err := validatePlanReportQueryTopology(query); err != nil {
		t.Fatalf("validatePlanReportQueryTopology() error = %v", err)
	}
}

func TestPlanReportOperatorsDoNotClassifyBranchedPushBroadcastMapPath(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       1,
			DisplayName: "Push Broadcast Hash Join",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "Map", ChildIndex: 8},
			},
		},
		{
			Index:       8,
			DisplayName: "Serialize Result",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "", ChildIndex: 9},
				{Type: "", ChildIndex: 30},
			},
		},
		{
			Index:       9,
			DisplayName: "Hash Join",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "Build", ChildIndex: 10},
			},
		},
		{
			Index:       10,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_method": "Batch",
				"scan_target": "$v2",
				"scan_type":   "BatchScan",
			}),
		},
		{
			Index:       30,
			DisplayName: "Hash Join",
		},
	}}
	operators := planReportOperators(plan)
	for _, operator := range operators {
		if operator.Family == "push_broadcast_hash_join_internal_hash_join" {
			t.Fatalf("branched Push Broadcast Map path classified internal Hash Join: %+v", operators)
		}
	}
}

func TestPlanReportOperatorsClassifyPushBroadcastOuterApplyInternalHashJoin(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       1,
			DisplayName: "Push Broadcast Hash Join Outer Apply",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "Map", ChildIndex: 2},
			},
		},
		{
			Index:       2,
			DisplayName: "Hash Join",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "Build", ChildIndex: 3},
			},
		},
		{
			Index:       3,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_method": "Row",
				"scan_target": "$v2",
				"scan_type":   "BatchScan",
			}),
		},
	}}
	operators := planReportOperators(plan)
	if got, want := operators[0].Family, "push_broadcast_hash_join"; got != want {
		t.Fatalf("wrapper family = %q, want %q", got, want)
	}
	if got, want := operators[1].Family, "push_broadcast_hash_join_internal_hash_join"; got != want {
		t.Fatalf("internal hash join family = %q, want %q", got, want)
	}
}

func TestPlanReportOperatorsClassifySubqueryJoinHintOperators(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{Index: 1, DisplayName: "Semi Apply"},
		{Index: 2, DisplayName: "Anti-Semi Apply"},
		{
			Index:       3,
			DisplayName: "Push Broadcast Hash Join Semi Apply",
			ChildLinks:  []*spannerpb.PlanNode_ChildLink{{Type: "Map", ChildIndex: 4}},
		},
		{
			Index:       4,
			DisplayName: "Hash Join",
			ChildLinks:  []*spannerpb.PlanNode_ChildLink{{Type: "Build", ChildIndex: 5}},
		},
		{
			Index:       5,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_method": "Row",
				"scan_target": "$v2",
				"scan_type":   "BatchScan",
			}),
		},
		{
			Index:       6,
			DisplayName: "Distributed Semi Apply",
			ChildLinks:  []*spannerpb.PlanNode_ChildLink{{Type: "Map", ChildIndex: 7}},
		},
		{
			Index:       7,
			DisplayName: "Semi Apply",
			ChildLinks:  []*spannerpb.PlanNode_ChildLink{{Type: "", ChildIndex: 8}},
		},
		{
			Index:       8,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_method": "Row",
				"scan_target": "$v2",
				"scan_type":   "BatchScan",
			}),
		},
		{Index: 9, DisplayName: "Push Broadcast Hash Join Anti Semi Apply"},
		{
			Index:       10,
			DisplayName: "Distributed Anti Semi Apply",
			ChildLinks:  []*spannerpb.PlanNode_ChildLink{{Type: "Map", ChildIndex: 11}},
		},
		{
			Index:       11,
			DisplayName: "Semi Apply",
			ChildLinks:  []*spannerpb.PlanNode_ChildLink{{Type: "", ChildIndex: 12}},
		},
		{
			Index:       12,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_method": "Row",
				"scan_target": "$v3",
				"scan_type":   "BatchScan",
			}),
		},
	}}
	operators := planReportOperators(plan)
	byIndex := map[int32]string{}
	for _, operator := range operators {
		byIndex[operator.Index] = operator.Family
	}
	assertions := map[int32]string{
		1:  "semi_apply",
		2:  "anti_semi_apply",
		3:  "push_broadcast_hash_join",
		4:  "push_broadcast_hash_join_internal_hash_join",
		6:  "distributed_semi_apply",
		7:  "distributed_semi_apply_internal_apply",
		9:  "push_broadcast_hash_join",
		10: "distributed_anti_semi_apply",
		11: "distributed_anti_semi_apply_internal_apply",
	}
	for index, want := range assertions {
		if got := byIndex[index]; got != want {
			t.Fatalf("operator %d family = %q, want %q", index, got, want)
		}
	}
}

type planReportOperatorFixture struct {
	Plan         json.RawMessage   `json:"plan"`
	WantFamilies map[string]string `json:"want_families"`
}

func TestPlanReportOperatorFamilyFixtures(t *testing.T) {
	for _, name := range []string{
		"standalone_hash_join",
		"push_broadcast_internal_hash_join",
		"push_broadcast_branched_regular_hash_join",
		"standalone_semi_apply",
		"distributed_semi_apply_internal",
		"distributed_anti_semi_apply_internal",
	} {
		t.Run(name, func(t *testing.T) {
			_, operators, fixture := loadPlanReportOperatorFixture(t, name)
			byIndex := map[int32]string{}
			for _, operator := range operators {
				byIndex[operator.Index] = operator.Family
			}
			for rawIndex, want := range fixture.WantFamilies {
				index64, err := strconv.ParseInt(rawIndex, 10, 32)
				if err != nil {
					t.Fatalf("invalid fixture index %q: %v", rawIndex, err)
				}
				if got := byIndex[int32(index64)]; got != want {
					t.Fatalf("operator %s family = %q, want %q; operators=%+v", rawIndex, got, want, operators)
				}
			}
		})
	}
}

func TestPlanReportContractFixtureSemantics(t *testing.T) {
	tests := []struct {
		fixture        string
		predefined     string
		wantStatus     string
		wantViolations int
	}{
		{fixture: "standalone_hash_join", predefined: "no_hash_join", wantStatus: planContractStatusFail, wantViolations: 1},
		{fixture: "standalone_hash_join", predefined: "no_standalone_hash_join", wantStatus: planContractStatusFail, wantViolations: 1},
		{fixture: "push_broadcast_internal_hash_join", predefined: "no_hash_join", wantStatus: planContractStatusFail, wantViolations: 1},
		{fixture: "push_broadcast_internal_hash_join", predefined: "no_standalone_hash_join", wantStatus: planContractStatusPass},
		{fixture: "push_broadcast_branched_regular_hash_join", predefined: "no_standalone_hash_join", wantStatus: planContractStatusFail, wantViolations: 1},
		{fixture: "standalone_semi_apply", predefined: "no_apply_join", wantStatus: planContractStatusFail, wantViolations: 2},
		{fixture: "standalone_semi_apply", predefined: "no_standalone_apply_join", wantStatus: planContractStatusFail, wantViolations: 2},
		{fixture: "distributed_semi_apply_internal", predefined: "no_apply_join", wantStatus: planContractStatusFail, wantViolations: 1},
		{fixture: "distributed_semi_apply_internal", predefined: "no_standalone_apply_join", wantStatus: planContractStatusPass},
		{fixture: "distributed_anti_semi_apply_internal", predefined: "no_apply_join", wantStatus: planContractStatusFail, wantViolations: 1},
		{fixture: "distributed_anti_semi_apply_internal", predefined: "no_standalone_apply_join", wantStatus: planContractStatusPass},
	}
	for _, tt := range tests {
		t.Run(tt.fixture+"/"+tt.predefined, func(t *testing.T) {
			_, operators, _ := loadPlanReportOperatorFixture(t, tt.fixture)
			report := planReport{
				Status:  "ok",
				Backend: "omni",
				Queries: []planReportQuery{{
					Name:                 tt.fixture,
					Scope:                "query",
					Status:               "ok",
					NormalizedOperators:  operators,
					OperatorFamilies:     planReportOperatorFamilies(operators),
					OperatorFamilyCounts: planReportOperatorFamilyCounts(operators),
				}},
			}
			contracts := planContractsFile{
				Version: planContractFileVersionV1Alpha,
				Contracts: []planContract{{
					Name:   "FixtureContract",
					Target: "query/" + tt.fixture,
					Use:    []string{tt.predefined},
				}},
			}
			if err := applyPlanContracts(&report, contracts); err != nil {
				t.Fatalf("applyPlanContracts() error = %v", err)
			}
			got := report.ContractEvaluations[0]
			if got.Status != tt.wantStatus {
				t.Fatalf("contract status = %q, want %q; evaluation=%+v", got.Status, tt.wantStatus, got)
			}
			if got := planReportContractViolationCount(report); got != tt.wantViolations {
				t.Fatalf("contract violation count = %d, want %d; evaluation=%+v", got, tt.wantViolations, report.ContractEvaluations[0])
			}
		})
	}
}

func loadPlanReportOperatorFixture(t testing.TB, name string) (*spannerpb.QueryPlan, []planReportOperator, planReportOperatorFixture) {
	t.Helper()
	path := filepath.Join("testdata", "plan_fixtures", name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", path, err)
	}
	var fixture planReportOperatorFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", path, err)
	}
	plan := &spannerpb.QueryPlan{}
	if err := protojson.Unmarshal(fixture.Plan, plan); err != nil {
		t.Fatalf("protojson.Unmarshal(%s plan) error = %v", path, err)
	}
	return plan, planReportOperators(plan), fixture
}

func TestPlanReportOperatorsClassifyDistributedCrossApplyInternalApply(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       1,
			DisplayName: "Distributed Cross Apply",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "Map", ChildIndex: 8},
			},
		},
		{
			Index:       8,
			DisplayName: "Serialize Result",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "", ChildIndex: 9},
			},
		},
		{
			Index:       9,
			DisplayName: "Cross Apply",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "", ChildIndex: 10},
				{Type: "Map", ChildIndex: 20},
			},
		},
		{
			Index:       10,
			DisplayName: "KeyRangeAccumulator",
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "", ChildIndex: 11},
			},
		},
		{
			Index:       11,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_method": "Batch",
				"scan_target": "$v2",
				"scan_type":   "BatchScan",
			}),
		},
		{
			Index:       20,
			DisplayName: "Scan",
		},
	}}
	operators := planReportOperators(plan)
	byIndex := map[int32]string{}
	for _, operator := range operators {
		byIndex[operator.Index] = operator.Family
	}
	if got, want := byIndex[1], "distributed_cross_apply"; got != want {
		t.Fatalf("distributed cross apply family = %q, want %q", got, want)
	}
	if got, want := byIndex[9], "distributed_cross_apply_internal_apply"; got != want {
		t.Fatalf("distributed cross apply internal family = %q, want %q", got, want)
	}
}

func TestPlanReportCELContract(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Optimizer: planReportOptimizer{
			Requested: planReportOptimizerEnvironment{
				Version:           "not_pinned",
				StatisticsPackage: "not_pinned",
			},
		},
		Queries: []planReportQuery{
			{
				Name:             "HashAggregate",
				Scope:            "query",
				Status:           "ok",
				OperatorFamilies: []string{"hash_aggregate", "scan"},
				OperatorFamilyCounts: map[string]int{
					"hash_aggregate": 1,
					"scan":           1,
				},
				RawPlan: &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
					{
						Index:       0,
						DisplayName: "Aggregate",
						ChildLinks: []*spannerpb.PlanNode_ChildLink{
							{ChildIndex: 1, Type: "Input"},
						},
						Metadata: &structpb.Struct{Fields: map[string]*structpb.Value{
							"iterator_type": structpb.NewStringValue("Hash"),
						}},
					},
					{
						Index:       1,
						DisplayName: "Scan",
						Metadata: &structpb.Struct{Fields: map[string]*structpb.Value{
							"Full scan": structpb.NewBoolValue(true),
						}},
					},
				}},
			},
			{
				Name:   "RowScan",
				Scope:  "query",
				Status: "ok",
				OperatorFamilyCounts: map[string]int{
					"scan": 1,
				},
				NormalizedOperators: []planReportOperator{
					{Index: 0, DisplayName: "Scan", Family: "scan", ScanMethod: "row", ScanType: "index_scan", FullScan: true},
				},
			},
		},
	}
	report.Queries[0].NormalizedOperators = planReportOperators(report.Queries[0].RawPlan)
	report.Queries[0].OperatorEdges = planReportOperatorEdges(report.Queries[0].RawPlan)
	contracts := planContractsFile{
		Version: "v1alpha-plan-contracts",
		Contracts: []planContract{
			{
				Name:   "HasHashAggregate",
				Target: "query/HashAggregate",
				CEL:    `operators.exists(o, o.family == "hash_aggregate") && !operator_families.exists(f, f == "stream_aggregate")`,
			},
			{
				Name:   "NoHashAggregate",
				Target: "query/HashAggregate",
				CEL:    `operators.all(o, o.family != "hash_aggregate")`,
			},
			{
				Name:   "RawPlanShape",
				Target: "query/HashAggregate",
				CEL:    `raw_plan.plan_nodes.size() == raw_nodes.size() && raw_nodes.exists(n, n.display_name == "Aggregate" && n.child_links.exists(c, c.child_index == 1 && c.type == "Input") && n.metadata["iterator_type"] == "Hash")`,
			},
			{
				Name:   "RowFullScan",
				Target: "query/RowScan",
				CEL:    `operators.exists(o, o.scan_method == "row" && o.scan_type == "index_scan" && o.full_scan)`,
			},
			{
				Name:   "OperatorFamilyCounts",
				Target: "query/HashAggregate",
				CEL:    `operator_family_counts["hash_aggregate"] == 1 && query.operator_family_counts["scan"] == 1`,
			},
			{
				Name:   "ZeroFilledOperatorFamilyCounts",
				Target: "query/HashAggregate",
				CEL:    `operator_family_counts["stream_aggregate"] == 0 && query.operator_family_counts["merge_join"] == 0`,
			},
			{
				Name:   "NormalizedTopology",
				Target: "query/HashAggregate",
				CEL:    `operator_edges.exists(e, e.parent_index == 0 && e.child_index == 1 && e.type == "Input") && operators.exists(o, o.index == 0 && o.descendant_indexes.exists(i, i == 1) && o.subtree_family_counts["scan"] == 1)`,
			},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	if got, want := report.ContractSummary.Passed, 6; got != want {
		t.Fatalf("contract summary passed = %d, want %d", got, want)
	}
	if got, want := report.ContractSummary.Failed, 1; got != want {
		t.Fatalf("contract summary failed = %d, want %d", got, want)
	}
	if got, want := strings.Join(report.ContractSummary.EnvironmentWarnings, ","), "optimizer_not_pinned,statistics_package_not_pinned"; got != want {
		t.Fatalf("contract summary environment warnings = %q, want %q", got, want)
	}
	if got, want := report.ContractEvaluations[0].Results[0].Rule, "cel"; got != want {
		t.Fatalf("first contract rule = %q, want %q", got, want)
	}
	if got, want := report.ContractEvaluations[1].Status, planContractStatusFail; got != want {
		t.Fatalf("second contract status = %q, want %q", got, want)
	}
	if got, want := report.ContractEvaluations[2].Status, planContractStatusPass; got != want {
		t.Fatalf("raw plan contract status = %q, want %q", got, want)
	}
	if got, want := report.ContractEvaluations[0].Stability.Tier, planContractStabilityNormalized; got != want {
		t.Fatalf("normalized contract stability = %q, want %q", got, want)
	}
	if got, want := report.ContractEvaluations[2].Stability.Tier, planContractStabilityRawPlan; got != want {
		t.Fatalf("raw plan contract stability = %q, want %q", got, want)
	}
	if report.ContractEvaluations[2].Stability.CheckRecommended {
		t.Fatalf("raw plan contract stability check_recommended = true, want false")
	}
	if report.ContractEvaluations[2].Stability.ReplayableFromReport {
		t.Fatalf("raw plan contract replayable_from_report = true, want false")
	}
	if got, want := report.ContractEvaluations[2].TargetID, "query/HashAggregate"; got != want {
		t.Fatalf("raw plan contract target_id = %q, want %q", got, want)
	}
	addPlanReportContractCheckWarnings(&report)
	if got, want := strings.Join(report.ContractSummary.EnvironmentWarnings, ","), "optimizer_not_pinned,statistics_package_not_pinned,raw_query_plan_contract_used_in_check"; got != want {
		t.Fatalf("contract summary environment warnings after check = %q, want %q", got, want)
	}
}

func TestPlanReportRootPartitionableCELContractExample(t *testing.T) {
	rootPartitionableCEL := `
operator_family_counts["unknown"] == 0 &&
(
  operators.filter(o, o.subquery_cluster_node != "" && o.call_type != "local").size() == 1 &&
  operators.exists(o,
    o.index == 0 &&
    o.family == "distributed_union" &&
    o.subquery_cluster_node != "" &&
    o.call_type != "local")
) ||
(
  operators.filter(o, o.subquery_cluster_node != "" && o.call_type != "local").size() == 0
)`
	query := func(name string, operators []planReportOperator) planReportQuery {
		return planReportQuery{
			Name:                 name,
			Scope:                "query",
			Status:               "ok",
			OperatorFamilies:     planReportOperatorFamilies(operators),
			OperatorFamilyCounts: planReportOperatorFamilyCounts(operators),
			NormalizedOperators:  operators,
		}
	}
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{
			query("RootDistributedUnion", []planReportOperator{
				{Index: 0, DisplayName: "Distributed Union", Family: "distributed_union", SubqueryClusterNode: "1"},
				{Index: 1, DisplayName: "Distributed Union", Family: "distributed_union", CallType: "local", SubqueryClusterNode: "2"},
				{Index: 2, DisplayName: "Serialize Result", Family: "serialize_result"},
				{Index: 3, DisplayName: "Scan", Family: "scan"},
			}),
			query("RootDistributedUnionWithLocalDU", []planReportOperator{
				{Index: 0, DisplayName: "Distributed Union", Family: "distributed_union", SubqueryClusterNode: "1"},
				{Index: 1, DisplayName: "Distributed Union", Family: "distributed_union", CallType: "local", SubqueryClusterNode: "2"},
				{Index: 2, DisplayName: "Serialize Result", Family: "serialize_result"},
				{Index: 3, DisplayName: "Scan", Family: "scan"},
			}),
			query("NoDistributedOperators", []planReportOperator{
				{Index: 0, DisplayName: "Serialize Result", Family: "serialize_result"},
				{Index: 1, DisplayName: "Unit Relation", Family: "unit_relation"},
			}),
			query("SortAboveDistributedUnion", []planReportOperator{
				{Index: 0, DisplayName: "Serialize Result", Family: "serialize_result"},
				{Index: 1, DisplayName: "Sort", Family: "full_sort"},
				{Index: 2, DisplayName: "Distributed Union", Family: "distributed_union", SubqueryClusterNode: "3"},
				{Index: 3, DisplayName: "Scan", Family: "scan"},
			}),
			query("DistributedMergeUnion", []planReportOperator{
				{Index: 0, DisplayName: "Distributed Union", Family: "distributed_merge_union", SubqueryClusterNode: "1"},
				{Index: 1, DisplayName: "Serialize Result", Family: "serialize_result"},
				{Index: 2, DisplayName: "Sort", Family: "full_sort"},
				{Index: 3, DisplayName: "Scan", Family: "scan"},
			}),
			query("DistributedCrossApply", []planReportOperator{
				{Index: 0, DisplayName: "Distributed Union", Family: "distributed_union", SubqueryClusterNode: "1"},
				{Index: 1, DisplayName: "Distributed Cross Apply", Family: "distributed_cross_apply", SubqueryClusterNode: "2"},
				{Index: 2, DisplayName: "Scan", Family: "scan"},
			}),
			query("FutureDistributedOperator", []planReportOperator{
				{Index: 0, DisplayName: "Distributed Union", Family: "distributed_union", SubqueryClusterNode: "1"},
				{Index: 1, DisplayName: "Future Distributed Operator", Family: "unknown", SubqueryClusterNode: "2"},
				{Index: 2, DisplayName: "Scan", Family: "scan"},
			}),
		},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{
			{Name: "RootDU", Target: "query/RootDistributedUnion", CEL: rootPartitionableCEL},
			{Name: "NoDistributed", Target: "query/NoDistributedOperators", CEL: rootPartitionableCEL},
			{Name: "RootDULocal", Target: "query/RootDistributedUnionWithLocalDU", CEL: rootPartitionableCEL},
			{Name: "SortAboveDU", Target: "query/SortAboveDistributedUnion", CEL: rootPartitionableCEL},
			{Name: "DistributedMerge", Target: "query/DistributedMergeUnion", CEL: rootPartitionableCEL},
			{Name: "DistributedApply", Target: "query/DistributedCrossApply", CEL: rootPartitionableCEL},
			{Name: "FutureDistributed", Target: "query/FutureDistributedOperator", CEL: rootPartitionableCEL},
		},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	wantStatuses := []string{
		planContractStatusPass,
		planContractStatusPass,
		planContractStatusPass,
		planContractStatusFail,
		planContractStatusFail,
		planContractStatusFail,
		planContractStatusFail,
	}
	for i, want := range wantStatuses {
		if got := report.ContractEvaluations[i].Status; got != want {
			t.Fatalf("contract %d status = %q, want %q; evaluation=%+v", i, got, want, report.ContractEvaluations[i])
		}
	}
	if got, want := report.ContractSummary.Passed, 3; got != want {
		t.Fatalf("contract summary passed = %d, want %d", got, want)
	}
	if got, want := report.ContractSummary.Failed, 4; got != want {
		t.Fatalf("contract summary failed = %d, want %d", got, want)
	}
	if got := report.ContractEvaluations[0].Stability.Reasons; !slices.Contains(got, "contract reads metadata-derived normalized fields: call_type, subquery_cluster_node") {
		t.Fatalf("root-partitionable stability reasons = %v, want metadata-derived reason", got)
	}
}

func TestPlanReportCELRejectsExecutionStats(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{{
			Name:   "ProfileOnly",
			Scope:  "query",
			Status: "ok",
		}},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{{
			Name:   "ExecutionStats",
			Target: "query/ProfileOnly",
			CEL:    `raw_nodes.exists(n, n.execution_stats != null)`,
		}},
	}
	err := applyPlanContracts(&report, contracts)
	if err == nil || !strings.Contains(err.Error(), "execution_stats is not supported") {
		t.Fatalf("applyPlanContracts() error = %v, want execution_stats rejection", err)
	}
}

func TestPlanReportContractTargetID(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{
			{
				Name:                "ExternalQuerySingerIDs",
				TargetID:            "query/ExternalQuerySingerIDs#inner",
				Scope:               "external_query.inner",
				Status:              "ok",
				NormalizedOperators: []planReportOperator{{Index: 0, DisplayName: "Scan", Family: "scan"}},
			},
		},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{{
			Name:   "ExternalInnerPlan",
			Target: "query/ExternalQuerySingerIDs#inner",
			Use:    []string{"no_explicit_sort"},
		}},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	evaluation := report.ContractEvaluations[0]
	if got, want := evaluation.Query, "ExternalQuerySingerIDs"; got != want {
		t.Fatalf("contract evaluation query = %q, want %q", got, want)
	}
	if got, want := evaluation.TargetID, "query/ExternalQuerySingerIDs#inner"; got != want {
		t.Fatalf("contract evaluation target_id = %q, want %q", got, want)
	}
}

func TestPlanReportContractTargetIDRejectsMissingTarget(t *testing.T) {
	report := planReport{
		Status:  "ok",
		Backend: "omni",
		Queries: []planReportQuery{{
			Name:     "Actual",
			TargetID: "query/Actual",
			Status:   "ok",
		}},
	}
	contracts := planContractsFile{
		Version: planContractFileVersionV1Alpha,
		Contracts: []planContract{{
			Name:   "Missing",
			Target: "query/Missing",
			Use:    []string{"no_explicit_sort"},
		}},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	evaluation := report.ContractEvaluations[0]
	if evaluation.Status != planContractStatusNotEvaluated || evaluation.Reason != planContractReasonTargetNotFound {
		t.Fatalf("contract evaluation = %+v, want target_not_found not_evaluated", evaluation)
	}
	if evaluation.Error != "" {
		t.Fatalf("contract evaluation error = %q, want empty error for target_not_found", evaluation.Error)
	}
}

func TestReadPlanContractsRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan-contracts.yaml")
	if err := os.WriteFile(path, []byte(`
version: v1alpha-plan-contracts
contracts:
- name: UnexpectedField
  query: ListSingers
  backend: omni
  use:
  - no_explicit_sort
  suppressions: []
`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := readPlanContracts(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("readPlanContracts() error = %v, want unknown field error", err)
	}
}

func TestReadPlanContractsRejectsInvalidTargetID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan-contracts.yaml")
	if err := os.WriteFile(path, []byte(`
version: v1alpha-plan-contracts
contracts:
- name: InvalidTarget
  target: LookupSingerByName
  use:
  - no_explicit_sort
`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := readPlanContracts(path); err == nil || !strings.Contains(err.Error(), "target \"LookupSingerByName\" must match ^query/") {
		t.Fatalf("readPlanContracts() error = %v, want target pattern error", err)
	}
}

func TestReadPlanContractsRejectsDuplicateNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan-contracts.yaml")
	if err := os.WriteFile(path, []byte(`
version: v1alpha-plan-contracts
contracts:
- name: NoBadPlan
  target: query/A
  use:
  - no_explicit_sort
- name: NoBadPlan
  target: query/B
  use:
  - no_hash_join
`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := readPlanContracts(path); err == nil || !strings.Contains(err.Error(), "plan_contract.duplicate_contract_name") {
		t.Fatalf("readPlanContracts() error = %v, want duplicate contract name diagnostic", err)
	}
}

func TestReadPlanContractsRejectsDuplicateForbidOperatorFamily(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan-contracts.yaml")
	if err := os.WriteFile(path, []byte(`
version: v1alpha-plan-contracts
contracts:
- name: NoBadPlan
  target: query/A
  forbid:
  - operator_family: hash_join
    max_count: 0
  - operator_family: hash_join
    max_count: 1
`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := readPlanContracts(path); err == nil || !strings.Contains(err.Error(), "plan_contract.duplicate_forbid_operator_family") {
		t.Fatalf("readPlanContracts() error = %v, want duplicate forbid operator family diagnostic", err)
	}
}

func TestPlanReportContractRejectsSkippedTarget(t *testing.T) {
	report := planReport{
		Status:  "no_targets",
		Backend: "omni",
		Queries: []planReportQuery{{
			Name:    "BigQueryOnly",
			Catalog: "analytics",
			Scope:   "query",
			Kind:    "sql",
			Status:  "skipped",
			Error:   "source catalog \"analytics\" has dialect \"bigquery\"; plan-report currently supports Spanner GoogleSQL catalogs",
		}},
	}
	contracts := planContractsFile{
		Version: "v1alpha-plan-contracts",
		Contracts: []planContract{{
			Name:   "BigQueryOuterPlan",
			Target: "query/BigQueryOnly",
			Use:    []string{"no_explicit_sort"},
		}},
	}
	if err := applyPlanContracts(&report, contracts); err != nil {
		t.Fatalf("applyPlanContracts() error = %v", err)
	}
	evaluation := report.ContractEvaluations[0]
	if evaluation.Status != planContractStatusNotEvaluated || evaluation.Reason != planContractReasonTargetError {
		t.Fatalf("contract evaluation = %+v, want target_error not_evaluated", evaluation)
	}
}

func TestPlanReportOperatorFamilyUsesDisplayName(t *testing.T) {
	for _, tt := range []struct {
		name string
		node *spannerpb.PlanNode
		want string
	}{
		{
			name: "sort",
			node: &spannerpb.PlanNode{DisplayName: "Sort"},
			want: "full_sort",
		},
		{
			name: "sort limit",
			node: &spannerpb.PlanNode{DisplayName: "Sort Limit"},
			want: "full_sort",
		},
		{
			name: "local sort limit",
			node: &spannerpb.PlanNode{DisplayName: "Local Sort Limit"},
			want: "full_sort",
		},
		{
			name: "minor sort",
			node: &spannerpb.PlanNode{DisplayName: "Minor Sort"},
			want: "minor_sort",
		},
		{
			name: "minor sort limit",
			node: &spannerpb.PlanNode{DisplayName: "Minor Sort Limit"},
			want: "minor_sort",
		},
		{
			name: "local minor sort",
			node: &spannerpb.PlanNode{DisplayName: "Local Minor Sort"},
			want: "minor_sort",
		},
		{
			name: "global limit",
			node: &spannerpb.PlanNode{DisplayName: "Global Limit"},
			want: "limit",
		},
		{
			name: "distributed merge union",
			node: &spannerpb.PlanNode{
				DisplayName: "Distributed Union",
				Metadata:    mustStructPB(t, map[string]interface{}{"preserve_subquery_order": true}),
			},
			want: "distributed_merge_union",
		},
		{
			name: "distributed union",
			node: &spannerpb.PlanNode{
				DisplayName: "Distributed Union",
			},
			want: "distributed_union",
		},
		{
			name: "push broadcast hash join",
			node: &spannerpb.PlanNode{DisplayName: "Push Broadcast Hash Join"},
			want: "push_broadcast_hash_join",
		},
		{
			name: "cross apply",
			node: &spannerpb.PlanNode{DisplayName: "Cross Apply"},
			want: "apply_join",
		},
		{
			name: "distributed cross apply",
			node: &spannerpb.PlanNode{DisplayName: "Distributed Cross Apply"},
			want: "distributed_cross_apply",
		},
		{
			name: "merge join",
			node: &spannerpb.PlanNode{DisplayName: "Merge Join"},
			want: "merge_join",
		},
		{
			name: "hash aggregate",
			node: &spannerpb.PlanNode{
				DisplayName: "Aggregate",
				Metadata:    mustStructPB(t, map[string]interface{}{"iterator_type": "Hash"}),
			},
			want: "hash_aggregate",
		},
		{
			name: "stream aggregate",
			node: &spannerpb.PlanNode{
				DisplayName: "Aggregate",
				Metadata:    mustStructPB(t, map[string]interface{}{"iterator_type": "Stream"}),
			},
			want: "stream_aggregate",
		},
		{
			name: "unknown",
			node: &spannerpb.PlanNode{DisplayName: "New Fancy Operator"},
			want: "unknown",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := planReportOperatorFamily(tt.node); got != tt.want {
				t.Fatalf("planReportOperatorFamily(%q) = %q, want %q", tt.node.GetDisplayName(), got, tt.want)
			}
		})
	}
}

func TestPlanReportClassificationWarningsIncludeUnknownFamily(t *testing.T) {
	tests := []struct {
		name     string
		operator planReportOperator
		wantID   string
	}{
		{
			name: "unknown operator family",
			operator: planReportOperator{
				Index:       7,
				DisplayName: "New Fancy Operator",
				Family:      "unknown",
			},
			wantID: "operator_family_unknown",
		},
		{
			name: "generic join family",
			operator: planReportOperator{
				Index:       8,
				DisplayName: "Join",
				Family:      "join",
			},
			wantID: "join_family_unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := planReportClassificationWarnings([]planReportOperator{tt.operator})
			if len(warnings) != 1 {
				t.Fatalf("warnings length = %d, want 1: %+v", len(warnings), warnings)
			}
			if got := warnings[0].ID; got != tt.wantID {
				t.Fatalf("warning ID = %q, want %q", got, tt.wantID)
			}
		})
	}
}

func TestPlanReportObservedDocsOperatorFamiliesAreKnown(t *testing.T) {
	tests := []struct {
		displayName string
		want        string
	}{
		{displayName: "Anti-Semi Apply", want: "anti_semi_apply"},
		{displayName: "Apply Mutations", want: "apply_mutations"},
		{displayName: "Array Subquery", want: "array_subquery"},
		{displayName: "Array Unnest", want: "array_unnest"},
		{displayName: "BloomFilterBuild", want: "bloom_filter_build"},
		{displayName: "ChangeStream TVF", want: "change_stream_tvf"},
		{displayName: "Compute", want: "compute"},
		{displayName: "Compute Struct", want: "compute_struct"},
		{displayName: "Create Batch", want: "create_batch"},
		{displayName: "DataBlockToRow", want: "data_block_to_row"},
		{displayName: "Distributed Anti Semi Apply", want: "distributed_anti_semi_apply"},
		{displayName: "Distributed Semi Apply", want: "distributed_semi_apply"},
		{displayName: "Empty Relation", want: "empty_relation"},
		{displayName: "Filter", want: "filter"},
		{displayName: "KeyRangeAccumulator", want: "key_range_accumulator"},
		{displayName: "Global Limit", want: "limit"},
		{displayName: "Limit", want: "limit"},
		{displayName: "Local Limit", want: "limit"},
		{displayName: "Local Sort Limit", want: "full_sort"},
		{displayName: "Local Minor Sort", want: "minor_sort"},
		{displayName: "MiniBatchAssign", want: "mini_batch_assign"},
		{displayName: "MiniBatchKeyOrder", want: "mini_batch_key_order"},
		{displayName: "Push Broadcast Hash Join Outer Apply", want: "push_broadcast_hash_join"},
		{displayName: "Push Broadcast Hash Join Semi Apply", want: "push_broadcast_hash_join"},
		{displayName: "Push Broadcast Hash Join Anti Semi Apply", want: "push_broadcast_hash_join"},
		{displayName: "Random Id Assign", want: "random_id_assign"},
		{displayName: "Recursive Spool Scan", want: "recursive_spool_scan"},
		{displayName: "RowCount", want: "row_count"},
		{displayName: "RowToDataBlock", want: "row_to_data_block"},
		{displayName: "Scalar Subquery", want: "scalar_subquery"},
		{displayName: "Search Predicate", want: "search_predicate"},
		{displayName: "Semi Apply", want: "semi_apply"},
		{displayName: "SpoolBuild", want: "spool_build"},
		{displayName: "SpoolScan", want: "spool_scan"},
		{displayName: "Unit Relation", want: "unit_relation"},
		{displayName: "VerifyDeterminism", want: "verify_determinism"},
		{displayName: "Union Input", want: "union_input"},
	}
	for _, tt := range tests {
		t.Run(tt.displayName, func(t *testing.T) {
			got := planReportOperatorFamily(&spannerpb.PlanNode{DisplayName: tt.displayName})
			if got != tt.want {
				t.Fatalf("planReportOperatorFamily(%q) = %q, want %q", tt.displayName, got, tt.want)
			}
			if !planReportKnownOperatorFamily(got) {
				t.Fatalf("planReportOperatorFamily(%q) = %q, but it is not in planReportKnownOperatorFamilies", tt.displayName, got)
			}
		})
	}
}

func TestPlanReportSpoolScanDisplayNameWithSpoolNameMetadata(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{{
		Index:       15,
		DisplayName: "SpoolScan",
		Metadata: &structpb.Struct{Fields: map[string]*structpb.Value{
			"spool_name": structpb.NewStringValue("CTE"),
		}},
	}}}
	operators := planReportOperators(plan)
	if got, want := len(operators), 1; got != want {
		t.Fatalf("operator count = %d, want %d", got, want)
	}
	if got, want := operators[0].Family, "spool_scan"; got != want {
		t.Fatalf("SpoolScan family = %q, want %q", got, want)
	}
	if got, want := operators[0].SpoolName, "CTE"; got != want {
		t.Fatalf("SpoolScan spool_name = %q, want %q", got, want)
	}
	if got := operators[0].ScanType; got != "" {
		t.Fatalf("SpoolScan scan_type = %q, want empty because this raw representation is not Scan.scan_type=SpoolScan", got)
	}
}

func TestPlanReportObservedFullTextSearchTVFFamily(t *testing.T) {
	// Spanner Omni emits the capitalized "Name" metadata key; the lowercase
	// spelling is kept as a defensive fallback.
	for _, key := range []string{"Name", "name"} {
		node := &spannerpb.PlanNode{
			DisplayName: "TVF",
			Metadata: mustStructPB(t, map[string]interface{}{
				key: "Search Query Conversion",
			}),
		}
		if got, want := planReportOperatorFamily(node), "search_query_conversion_tvf"; got != want {
			t.Fatalf("planReportOperatorFamily(Search Query Conversion TVF, metadata key %q) = %q, want %q", key, got, want)
		}
	}
	if !planReportKnownOperatorFamily("search_query_conversion_tvf") {
		t.Fatal("search_query_conversion_tvf is not in planReportKnownOperatorFamilies")
	}
}

func TestPlanReportSearchIndexScanTypeMetadataNormalized(t *testing.T) {
	node := &spannerpb.PlanNode{
		DisplayName: "Scan",
		Metadata: mustStructPB(t, map[string]interface{}{
			"scan_type": "SearchIndexScan",
		}),
	}
	if got, want := planReportOperatorMetadataString(node, "scan_type"), "search_index_scan"; got != want {
		t.Fatalf("planReportOperatorMetadataString(scan_type) = %q, want %q", got, want)
	}
}

func mustStructPB(t testing.TB, fields map[string]interface{}) *structpb.Struct {
	t.Helper()
	value, err := structpb.NewStruct(fields)
	if err != nil {
		t.Fatalf("structpb.NewStruct() error = %v", err)
	}
	return value
}

func planContractMatchedOperatorIndexes(result planContractRuleResult) []int32 {
	if result.MatchedOperatorIndexes == nil {
		return nil
	}
	return *result.MatchedOperatorIndexes
}

func TestPlanReportOperatorsIncludeStableMetadata(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       0,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"execution_method":  "Row",
				"scan_method":       "Row",
				"scan_format":       "Row",
				"scan_type":         "IndexScan",
				"scan_target":       "SingersByFirstName",
				"seekable_key_size": "0",
				"Full scan":         "true",
			}),
		},
	}}
	operators := planReportOperators(plan)
	if got, want := len(operators), 1; got != want {
		t.Fatalf("operator count = %d, want %d", got, want)
	}
	operator := operators[0]
	if operator.ExecutionMethod != "row" || operator.ScanMethod != "row" || operator.ScanFormat != "row" {
		t.Fatalf("operator scan methods = %+v, want normalized row metadata", operator)
	}
	if operator.ScanType != "index_scan" || operator.ScanTarget != "SingersByFirstName" || operator.SeekableKeySize != "0" || !operator.FullScan {
		t.Fatalf("operator scan metadata = %+v, want scan type/target/seekable/full scan metadata", operator)
	}
}

func TestBuildPlanReportNoTargets(t *testing.T) {
	config := querygen.QueryCodegenConfig{
		Schemas: []querygen.QueryCodegenSchema{
			{Name: "analytics", Dialect: "bigquery"},
		},
	}
	plan := &querygen.QueryCodegenPlan{
		Queries: []querygen.QueryCodegenPlanQuery{
			{
				Name:    "BigQueryOnly",
				Catalog: "analytics",
				Kind:    "sql",
				SQL:     "SELECT 1 AS value",
			},
		},
	}
	report, err := buildPlanReportWithRuntime(t.Context(), config, plan, t.TempDir(), planReportOptions{
		Backend: "omni",
		Stable:  true,
		Optimizer: planReportOptimizerEnvironment{
			Version:           "8",
			StatisticsPackage: "latest",
		},
	}, nil)
	if err != nil {
		t.Fatalf("buildPlanReportWithRuntime() error = %v", err)
	}
	if got, want := report.Status, planReportStatusNoTargets; got != want {
		t.Fatalf("plan-report status = %q, want %q", got, want)
	}
	if !report.Stable {
		t.Fatalf("plan-report stable flag was not recorded")
	}
	if got, want := report.PlanSource.API, "analyze_query"; got != want {
		t.Fatalf("plan-report plan source api = %q, want %q", got, want)
	}
	if got, want := report.BackendIdentity.Version, "not_recorded"; got != want {
		t.Fatalf("plan-report backend version = %q, want %q", got, want)
	}
	if got, want := report.Normalization.OperatorTreeVersion, "v1alpha"; got != want {
		t.Fatalf("plan-report operator tree version = %q, want %q", got, want)
	}
	if got, want := report.Optimizer.Requested.Version, "8"; got != want {
		t.Fatalf("plan-report optimizer version = %q, want %q", got, want)
	}
	if got, want := report.Optimizer.Requested.StatisticsPackage, "latest"; got != want {
		t.Fatalf("plan-report optimizer statistics package = %q, want %q", got, want)
	}
	if got, want := report.Optimizer.Effective.Version, "not_recorded"; got != want {
		t.Fatalf("plan-report effective optimizer version = %q, want %q", got, want)
	}
	if got, want := report.TargetSummary.IncludedCount, 1; got != want {
		t.Fatalf("plan-report included target count = %d, want %d", got, want)
	}
	if got, want := report.TargetSummary.Planned, 0; got != want {
		t.Fatalf("plan-report planned target count = %d, want %d", got, want)
	}
	if got, want := report.TargetSummary.Errors, 0; got != want {
		t.Fatalf("plan-report error target count = %d, want %d", got, want)
	}
	if got, want := report.TargetSummary.Skipped, 1; got != want {
		t.Fatalf("plan-report skipped target count = %d, want %d", got, want)
	}
	if got, want := report.TargetSummary.Planned+report.TargetSummary.Errors+report.TargetSummary.Skipped, report.TargetSummary.IncludedCount; got != want {
		t.Fatalf("plan-report target summary invariant planned+errors+skipped = %d, want included_count %d", got, want)
	}
	if got, want := len(report.TargetSummary.Excluded), 0; got != want {
		t.Fatalf("plan-report excluded target count = %d, want %d", got, want)
	}
	if got, want := len(report.Queries), 1; got != want {
		t.Fatalf("plan-report query count = %d, want %d", got, want)
	}
	if got, want := report.Queries[0].Status, "skipped"; got != want {
		t.Fatalf("plan-report query status = %q, want %q", got, want)
	}
	if report.Queries[0].OperatorTreeSHA256 != "" || len(report.Queries[0].OperatorFamilies) != 0 || len(report.Queries[0].OperatorFamilyCounts) != 0 || len(report.Queries[0].NormalizedOperators) != 0 || len(report.Queries[0].OperatorEdges) != 0 || report.Queries[0].Plan != "" {
		t.Fatalf("skipped query leaked successful plan evidence: %+v", report.Queries[0])
	}
	if got, want := report.ContractEvalMode, planContractEvaluationModeNone; got != want {
		t.Fatalf("plan-report contract evaluation mode = %q, want %q", got, want)
	}
	if len(report.Warnings) != 1 || report.Warnings[0].ID != "plan-report-no-targets" {
		t.Fatalf("plan-report warnings = %+v, want no-target warning", report.Warnings)
	}
}

func TestPlanReportBackendIdentityFromFlags(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	identity, err := planReportBackendIdentityFromFlags("omni", "2026.r1-beta", digest)
	if err != nil {
		t.Fatalf("planReportBackendIdentityFromFlags() error = %v", err)
	}
	if got, want := identity, (planReportBackendIdentity{
		Kind:        "omni",
		Version:     "2026.r1-beta",
		ImageDigest: digest,
		Source:      "manual",
	}); got != want {
		t.Fatalf("identity = %+v, want %+v", got, want)
	}

	identity, err = planReportBackendIdentityFromFlags("omni", "", "")
	if err != nil {
		t.Fatalf("planReportBackendIdentityFromFlags(empty) error = %v", err)
	}
	if got, want := identity.Source, "spanemuboost"; got != want {
		t.Fatalf("empty source = %q, want %q", got, want)
	}
	if got, want := identity.Version, "not_recorded"; got != want {
		t.Fatalf("empty version = %q, want %q", got, want)
	}

	if _, err := planReportBackendIdentityFromFlags("omni", "", "sha256:"+strings.Repeat("A", 64)); err == nil {
		t.Fatalf("planReportBackendIdentityFromFlags() succeeded for uppercase digest")
	}
}

func TestBuildPlanReportTargetSummaryCountsPlanErrors(t *testing.T) {
	config := querygen.QueryCodegenConfig{
		Schemas: []querygen.QueryCodegenSchema{
			{Name: "app_spanner", Dialect: "spanner", DDL: "missing-schema.sql"},
		},
	}
	plan := &querygen.QueryCodegenPlan{
		Queries: []querygen.QueryCodegenPlanQuery{
			{
				Name:    "ListSingers",
				Catalog: "app_spanner",
				Kind:    "sql",
				SQL:     "SELECT SingerId FROM Singers",
			},
		},
	}
	report, err := buildPlanReportWithRuntime(t.Context(), config, plan, t.TempDir(), planReportOptions{
		Backend: "omni",
	}, nil)
	if err != nil {
		t.Fatalf("buildPlanReportWithRuntime() error = %v", err)
	}
	if got, want := report.Status, planReportStatusOK; got != want {
		t.Fatalf("plan-report status = %q, want %q", got, want)
	}
	if got, want := report.TargetSummary.IncludedCount, 1; got != want {
		t.Fatalf("plan-report included target count = %d, want %d", got, want)
	}
	if got, want := report.TargetSummary.Planned, 0; got != want {
		t.Fatalf("plan-report planned target count = %d, want %d", got, want)
	}
	if got, want := report.TargetSummary.Errors, 1; got != want {
		t.Fatalf("plan-report error target count = %d, want %d", got, want)
	}
	if got, want := report.TargetSummary.Skipped, 0; got != want {
		t.Fatalf("plan-report skipped target count = %d, want %d", got, want)
	}
	if got, want := report.TargetSummary.Planned+report.TargetSummary.Errors+report.TargetSummary.Skipped, report.TargetSummary.IncludedCount; got != want {
		t.Fatalf("plan-report target summary invariant planned+errors+skipped = %d, want included_count %d", got, want)
	}
	if report.Queries[0].OperatorTreeSHA256 != "" || len(report.Queries[0].OperatorFamilies) != 0 || len(report.Queries[0].OperatorFamilyCounts) != 0 || len(report.Queries[0].NormalizedOperators) != 0 || len(report.Queries[0].OperatorEdges) != 0 || report.Queries[0].Plan != "" {
		t.Fatalf("error query leaked successful plan evidence: %+v", report.Queries[0])
	}
	if got, want := report.Queries[0].Status, "error"; got != want {
		t.Fatalf("plan-report query status = %q, want %q", got, want)
	}
}

func TestPlanReportOperatorFamilyCounts(t *testing.T) {
	families := planReportOperatorFamilies([]planReportOperator{
		{Family: "scan"},
		{Family: "full_sort"},
		{Family: "minor_sort"},
		{},
	})
	if got, want := strings.Join(families, ","), "full_sort,minor_sort,scan"; got != want {
		t.Fatalf("operator families = %q, want concrete families only %q", got, want)
	}
	if slices.Contains(families, "explicit_sort") {
		t.Fatalf("operator families must not include derived explicit_sort: %v", families)
	}
	if slices.Contains(families, "blocking_operator") {
		t.Fatalf("operator families must not include derived blocking_operator: %v", families)
	}

	counts := planReportOperatorFamilyCounts([]planReportOperator{
		{Family: "scan"},
		{Family: "scan"},
		{Family: "full_sort"},
		{Family: "minor_sort"},
		{Family: "hash_join"},
		{},
	})
	if got, want := counts["scan"], 2; got != want {
		t.Fatalf("scan count = %d, want %d", got, want)
	}
	if got, want := counts["full_sort"], 1; got != want {
		t.Fatalf("full_sort count = %d, want %d", got, want)
	}
	if got, want := counts["minor_sort"], 1; got != want {
		t.Fatalf("minor_sort count = %d, want %d", got, want)
	}
	if got, want := counts["explicit_sort"], 2; got != want {
		t.Fatalf("explicit_sort count = %d, want %d", got, want)
	}
	if got, want := counts["hash_join"], 1; got != want {
		t.Fatalf("hash_join count = %d, want %d", got, want)
	}
	if got, want := counts["blocking_operator"], 2; got != want {
		t.Fatalf("blocking_operator count = %d, want %d", got, want)
	}
	if got, want := len(counts), len(planReportKnownOperatorFamilies()); got != want {
		t.Fatalf("operator family counts length = %d, want every known family %d", got, want)
	}
	for _, family := range planReportKnownOperatorFamilies() {
		if _, ok := counts[family]; !ok {
			t.Fatalf("operator family counts missing known family %q", family)
		}
	}
}

func TestPlanReportStableInputPath(t *testing.T) {
	baseDir := filepath.Join("tmp", "project")
	resolved := filepath.Join(baseDir, "contracts", "plan-contracts.yaml")
	if got, want := planReportStableInputPath(baseDir, resolved, "contracts/plan-contracts.yaml", false), "contracts/plan-contracts.yaml"; got != want {
		t.Fatalf("relative contract path = %q, want %q", got, want)
	}
	if got := planReportStableInputPath(baseDir, resolved, resolved, true); got != "" {
		t.Fatalf("stable contract path = %q, want empty", got)
	}
}

func TestPlanReportParamValues(t *testing.T) {
	values, err := planReportParamValues([]querygen.QueryCodegenParam{
		{Name: "active", Type: "BOOL"},
		{Name: "id", Type: "INT64"},
		{Name: "amount", Type: "NUMERIC"},
		{Name: "names", Type: "ARRAY<STRING>"},
	})
	if err != nil {
		t.Fatalf("planReportParamValues() error = %v", err)
	}
	if got, want := values["active"], true; got != want {
		t.Fatalf("active param = %#v, want %#v", got, want)
	}
	if got, want := values["id"], int64(1); got != want {
		t.Fatalf("id param = %#v, want %#v", got, want)
	}
	if got, want := values["amount"].(*big.Rat).String(), "1/1"; got != want {
		t.Fatalf("amount param = %q, want %q", got, want)
	}
	names, ok := values["names"].([]string)
	if !ok {
		t.Fatalf("names param type = %T, want []string", values["names"])
	}
	if got, want := strings.Join(names, ","), "value"; got != want {
		t.Fatalf("names param = %q, want %q", got, want)
	}
}

func TestPlanReportParamValuesRejectUnsupportedType(t *testing.T) {
	_, err := planReportParamValues([]querygen.QueryCodegenParam{
		{Name: "payload", Type: "STRUCT<id INT64>"},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported plan-report parameter type STRUCT") {
		t.Fatalf("planReportParamValues() error = %v, want unsupported STRUCT error", err)
	}
}

func TestRunExplainPlanDefaultOmitsAuditMatrix(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeExternalDatasetFixture(t, dir, configPath, false)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"explain-plan", "--config", configPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run explain-plan error = %v, stderr = %s", err, stderr.String())
	}
	for _, unwanted := range []string{
		"source_url:",
		"docs_last_checked:",
		"rows:",
	} {
		if strings.Contains(stdout.String(), unwanted) {
			t.Fatalf("default explain-plan output contains audit field %q:\n%s", unwanted, stdout.String())
		}
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"explain-plan", "--config", configPath, "--audit"}, &stdout, &stderr); err != nil {
		t.Fatalf("run explain-plan --audit error = %v, stderr = %s", err, stderr.String())
	}
	for _, want := range []string{
		"source_url:",
		"docs_last_checked:",
		"rows:",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("audit explain-plan output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunExplainPlanUsesNormalizedCatalogBindingShape(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeExternalDatasetFixture(t, dir, configPath, false)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"explain-plan", "--config", configPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run explain-plan error = %v, stderr = %s", err, stderr.String())
	}
	for _, want := range []string{
		"catalog_bindings:",
		"bigquery:",
		"dataset:",
		"path: analytics_spanner",
		"spanner:",
		"catalog: app",
		"database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app",
		"access:",
		"cloud_resource_connection_id: example-project.us.example-connection",
		"projection:",
		"unsupported_columns: omit",
		"named_schema_tables: warn_and_omit",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("explain-plan output missing %q:\n%s", want, stdout.String())
		}
	}
	for _, unwanted := range []string{
		"bigquery_dataset_ref:",
		"spanner_source:",
		"external_source:",
		"projection_policy:",
		"access_verification:",
		"projection_matrix:",
	} {
		if strings.Contains(stdout.String(), unwanted) {
			t.Fatalf("explain-plan output contains old binding field %q:\n%s", unwanted, stdout.String())
		}
	}
}

func TestRunExplainPlanStableOmitsVolatileEvidence(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeExternalDatasetFixture(t, dir, configPath, true)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"explain-plan", "--config", configPath, "--stable", "--audit"}, &stdout, &stderr); err != nil {
		t.Fatalf("run explain-plan --stable error = %v, stderr = %s", err, stderr.String())
	}
	if strings.Contains(stdout.String(), "checked_at") ||
		strings.Contains(stdout.String(), "docs_last_checked") ||
		strings.Contains(stdout.String(), "volatile: true") {
		t.Fatalf("stable explain-plan output contains volatile evidence fields:\n%s", stdout.String())
	}
	for _, want := range []string{
		"status: verified",
		"source: external_evidence",
		"verifier: terraform-plan",
		"independently_verified_by_generator: false",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stable explain-plan output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunExplainPlanSummaryIncludesCatalogBindingWarnings(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeExternalDatasetFixture(t, dir, configPath, true)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"explain-plan", "--config", configPath, "--output", "summary"}, &stdout, &stderr); err != nil {
		t.Fatalf("run explain-plan summary error = %v, stderr = %s", err, stderr.String())
	}
	for _, want := range []string{
		"Catalog binding: analytics_spanner (spanner_external_dataset, source: app)",
		"External source: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app",
		"Location: US (configured: us)",
		"Cloud resource connection: example-project.us.example-connection",
		"Cloud resource connection location: us (canonical: US) matched",
		"Projection policy: unsupported_columns=omit, named_schema_tables=warn_and_omit",
		"Access: cloud_resource_connection, database_role=reader, verification=verified",
		"Execution: data_boost=forced, writable=false",
		"analytics_spanner.Singers from Singers (BigQuery key metadata hidden)",
		"Relations:",
		"analytics_spanner.Singers: catalog=spanner_external_dataset_projection, role=select_source, allowed=true, projection_loss=true",
		"[warning] external-dataset-data-boost-permission-note",
		"[info] external-dataset-hidden-column",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("explain-plan summary output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunVet(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeTestFile(t, configPath, literalConfigYAML())

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"vet", "--config", configPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run vet error = %v, stderr = %s", err, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("vet wrote stdout:\n%s", stdout.String())
	}
}

func TestRunVetReportsWarningsAndCentralSuppressions(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeExternalDatasetFixture(t, dir, configPath, false)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"vet", "--config", configPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run vet error = %v, stderr = %s", err, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("vet text wrote stdout:\n%s", stdout.String())
	}
	for _, want := range []string{
		"warning: catalog_binding analytics_spanner: external-dataset-data-boost-permission-note",
		"info: catalog_binding analytics_spanner: external-dataset-hidden-column",
		"suppression: catalog_binding analytics_spanner: external-dataset-access-unverified",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("vet stderr missing %q:\n%s", want, stderr.String())
		}
	}
	if strings.Contains(stderr.String(), "warning: catalog_binding analytics_spanner: external-dataset-access-unverified") {
		t.Fatalf("vet stderr contains suppressed access warning:\n%s", stderr.String())
	}
}

func TestRunVetRejectsRulesSeverity(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeExternalDatasetFixture(t, dir, configPath, false)
	appendTestFile(t, configPath, `
  severity:
    external-dataset-data-boost-permission-note: error
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"vet", "--config", configPath}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), `unsupported v1alpha field "severity" at rules`) {
		t.Fatalf("run vet error = %v, want rules.severity rejection", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("run vet wrote stdout/stderr for config error:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunVetJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeExternalQueryFixture(t, dir, configPath, true)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"vet", "--config", configPath, "--output", "json"}, &stdout, &stderr); err != nil {
		t.Fatalf("run vet json error = %v, stderr = %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("vet json wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		`"diagnostics": [`,
		`"id": "cross-dialect-timestamp-truncation"`,
		`"stage": "external_query_type_conversion"`,
		`"warnings": [`,
		`"rule": "cross-dialect-timestamp-truncation"`,
		`"suppressible": true`,
		`"suppressed": false`,
		`"suppressions": [`,
		`"rule": "reviewed-warning"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("vet json stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunVetYAML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeExternalQueryFixture(t, dir, configPath, false)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"vet", "--config", configPath, "--output", "yaml"}, &stdout, &stderr); err != nil {
		t.Fatalf("run vet yaml error = %v, stderr = %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("vet yaml wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		"diagnostics:",
		"id: cross-dialect-timestamp-truncation",
		"stage: external_query_type_conversion",
		"warnings:",
		"rule: cross-dialect-timestamp-truncation",
		"suppressible: true",
		"suppressed: false",
		"scope: query",
		"name: ExternalEvents",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("vet yaml stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunVetReportsPlanErrors(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeSingerSchema(t, dir)
	writeTestFile(t, configPath, `
version: v1alpha
go:
  package: querydemo
catalogs:
- name: app
  kind: spanner
  ddl: schema.sql
writes:
- name: UpdateSinger
  catalog: app
  table: Singers
  operation: update
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"vet", "--config", configPath}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "update.columns is required") {
		t.Fatalf("run vet error = %v, want update.columns error", err)
	}
}

func TestRunVetJSONReportsPlanErrors(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeSingerSchema(t, dir)
	writeTestFile(t, configPath, `
version: v1alpha
go:
  package: querydemo
catalogs:
- name: app
  kind: spanner
  ddl: schema.sql
writes:
- name: UpdateSinger
  catalog: app
  table: Singers
  operation: update
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"vet", "--config", configPath, "--output", "json"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "update.columns is required") {
		t.Fatalf("run vet json error = %v, want update.columns error", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("vet json planning error wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		`"diagnostics": [`,
		`"id": "config-parse-error"`,
		`"stage": "config_parse"`,
		`"severity": "error"`,
		`"suppressible": false`,
		`"suppressed": false`,
		`update.columns is required`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("vet json planning error stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunVetJSONReportsSpecificPlanningDiagnosticIDs(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeSingerSchema(t, dir)
	writeTestFile(t, configPath, `
version: v1alpha
go:
  package: querydemo
catalogs:
- name: app
  kind: spanner
  ddl: schema.sql
- name: analytics
  kind: bigquery
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app
      location: us
      access:
        cloud_resource_connection_id: example-project.eu.example-connection
queries:
- name: ListSingers
  catalog: analytics
  kind: sql
  sql: SELECT SingerId FROM analytics_spanner.Singers
  result:
    struct: SingerRow
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"vet", "--config", configPath, "--output", "json"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "connection location") {
		t.Fatalf("run vet json error = %v, want connection location error", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("vet json planning error wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		`"id": "external-dataset-connection-location-mismatch"`,
		`"stage": "external_dataset_binding"`,
		`"subject": "spanner_external_datasets.access.cloud_resource_connection"`,
		`"rule": "external-dataset-connection-location-mismatch"`,
		`"suppressible": false`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("vet json planning error stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunVetYAMLReportsConfigParseErrors(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "querygen.yaml")
	writeTestFile(t, configPath, `
package: [
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"vet", "--config", configPath, "--output", "yaml"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "querygen.yaml") {
		t.Fatalf("run vet yaml error = %v, want config parse error", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("vet yaml config parse error wrote stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		"diagnostics:",
		"id: config-parse-error",
		"stage: config_parse",
		"severity: error",
		"suppressible: false",
		"suppressed: false",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("vet yaml config parse error stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func literalConfigYAML() string {
	return `
version: v1alpha
go:
  package: querydemo
catalogs:
- name: app
  kind: spanner
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: LiteralRow
`
}

func literalConfigYAMLWithOut() string {
	return `
version: v1alpha
go:
  package: querydemo
  out: querydemo.sql.go
catalogs:
- name: app
  kind: spanner
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: LiteralRow
`
}

func writeSingerSchema(t *testing.T, dir string) {
	t.Helper()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  SearchName STRING(MAX) AS (CAST(SingerId AS STRING)) STORED HIDDEN
) PRIMARY KEY (SingerId);
`)
}

func writeExternalDatasetFixture(t *testing.T, dir, configPath string, verified bool) {
	t.Helper()
	writeSingerSchema(t, dir)
	evidence := ""
	if verified {
		evidence = `
        verification_evidence:
          status: verified
          verifier: terraform-plan
          checked_at: "2026-05-04T10:30:00Z"`
	}
	writeTestFile(t, configPath, `
version: v1alpha
go:
  package: querydemo
catalogs:
- name: app
  kind: spanner
  ddl: schema.sql
- name: analytics
  kind: bigquery
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app
      spanner_database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
      location: us
      access:
        cloud_resource_connection_id: example-project.us.example-connection
`+evidence+`
      projection:
        unsupported_columns: omit
        named_schema_tables: warn_and_omit
queries:
- name: ListSingers
  catalog: analytics
  kind: sql
  sql: SELECT SingerId FROM analytics_spanner.Singers
  result:
    struct: SingerRow
rules:
  suppressions:
  - scope: catalog-binding/analytics.app_dataset
    rule: external-dataset-access-unverified
    reason: Static CI verifies external dataset access elsewhere.
`)
}

func writeExternalQueryFixture(t *testing.T, dir, configPath string, includeSuppression bool) {
	t.Helper()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Events (
  EventId INT64 NOT NULL,
  EventTimestamp TIMESTAMP
) PRIMARY KEY (EventId);
`)
	rules := ""
	if includeSuppression {
		rules = `
rules:
  suppressions:
  - scope: query/ExternalEvents
    rule: reviewed-warning
    reason: Reviewed test suppression.
`
	}
	writeTestFile(t, configPath, `
version: v1alpha
go:
  package: querydemo
catalogs:
- name: app
  kind: spanner
  ddl: spanner.sql
- name: analytics
  kind: bigquery
  bindings:
    external_query_connections:
    - name: app_conn
      id: example-project.us.example-connection
      spanner_catalog: app
queries:
- name: ExternalEvents
  catalog: analytics
  kind: external_query
  binding: app_conn
  inner_sql: SELECT EventTimestamp FROM Events
  result:
    struct: EventRow
`+rules)
}

func writeTestFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimLeft(data, "\n")), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%s) error = %v", path, err)
	}
}

func appendTestFile(t *testing.T, path, data string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("os.OpenFile(%s) error = %v", path, err)
	}
	if _, err := file.WriteString(data); err != nil {
		t.Fatalf("file.WriteString(%s) error = %v", path, err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("file.Close(%s) error = %v", path, err)
	}
}
