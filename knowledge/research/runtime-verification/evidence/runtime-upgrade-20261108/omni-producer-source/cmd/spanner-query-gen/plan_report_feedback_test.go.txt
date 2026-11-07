package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanalyzer/plancontract"
	"github.com/apstndb/spanemuboost"
	"github.com/apstndb/spannerplan/plantree/reference"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type feedbackPlanAnalyzer struct {
	statements []spanner.Statement
	dml        []bool
	closed     bool
}

func (a *feedbackPlanAnalyzer) AnalyzePlan(_ context.Context, stmt spanner.Statement, dml bool) (*spannerpb.QueryPlan, error) {
	a.statements = append(a.statements, stmt)
	a.dml = append(a.dml, dml)
	if strings.Contains(stmt.SQL, "no_plan") {
		return nil, nil
	}
	if strings.Contains(stmt.SQL, "bad_query") {
		return nil, errors.New("query rejected")
	}
	metadata, err := structpb.NewStruct(map[string]interface{}{"scan_target": "T", "scan_type": "TableScan", "Full scan": true})
	if err != nil {
		return nil, err
	}
	return &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{{Index: 0, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Scan", Metadata: metadata}}}, nil
}

func (a *feedbackPlanAnalyzer) Close() error { a.closed = true; return nil }

func feedbackReportFixture(t *testing.T) (querygen.QueryCodegenConfig, string, planReportOptions) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "schema.sql"), []byte("CREATE TABLE T (Id INT64 NOT NULL) PRIMARY KEY (Id)"), 0600); err != nil {
		t.Fatal(err)
	}
	config := querygen.QueryCodegenConfig{Schemas: []querygen.QueryCodegenSchema{{Name: "good", Dialect: "spanner", DDL: "schema.sql"}, {Name: "bad", Dialect: "spanner", DDL: "schema.sql"}}}
	return config, dir, planReportOptions{Backend: "omni", Format: reference.FormatCurrent, RenderMode: reference.RenderModePlan}
}

func TestPlanReportCollectorContinuesAfterCatalogAndQueryFailures(t *testing.T) {
	config, dir, opts := feedbackReportFixture(t)
	plan := &querygen.QueryCodegenPlan{Queries: []querygen.QueryCodegenPlanQuery{
		{Name: "Bad1", Catalog: "bad", Kind: "sql", SQL: "SELECT 1"},
		{Name: "Static", Catalog: "good", Kind: "sql", PlanningError: "invalid LIMIT parameter"},
		{Name: "BadQuery", Catalog: "good", Kind: "sql", SQL: "bad_query"},
		{Name: "NoPlan", Catalog: "good", Kind: "sql", SQL: "no_plan"},
		{Name: "Good", Catalog: "good", Kind: "sql", SQL: "SELECT Id FROM T"},
		{Name: "Remove", Catalog: "good", Kind: "write", ReportScope: "write", Result: "row_count", SQL: "DELETE FROM T WHERE Id=1"},
		{Name: "Bad2", Catalog: "bad", Kind: "sql", SQL: "SELECT 2"},
	}}
	opened := map[string]int{}
	analyzer := &feedbackPlanAnalyzer{}
	open := func(_ context.Context, _ spanemuboost.RuntimeHandle, schema querygen.QueryCodegenSchema, _ string, _ *spanalyzer.ProtoDescriptorSet, _ planReportOptimizer) (planReportAnalyzer, error) {
		opened[schema.Name]++
		if schema.Name == "bad" {
			return nil, errors.New("DDL rejected")
		}
		return analyzer, nil
	}
	report, err := buildPlanReportWithAnalyzer(t.Context(), config, plan, dir, opts, nil, open)
	if err != nil || report.TargetSummary.Planned != 2 || report.TargetSummary.Errors != 5 || len(report.Queries) != 7 || opened["bad"] != 1 || opened["good"] != 1 || !analyzer.closed {
		t.Fatalf("collector = %#v, opened %v, err %v", report, opened, err)
	}
	if report.Queries[5].TargetID != "write/Remove" || !slices.Equal(analyzer.dml, []bool{false, false, false, true}) {
		t.Fatalf("DML target = %#v, routing %v", report.Queries[5], analyzer.dml)
	}
	if len(report.Queries[3].NormalizedOperators) != 0 || len(report.Queries[3].AccessPaths) != 0 {
		t.Fatal("failed render leaked successful normalization fields")
	}
	if err := validatePlanReportInvariants(report); err != nil {
		t.Fatal(err)
	}
}

func TestPlanReportCollectorAcquiresEachOptionalShape(t *testing.T) {
	config, dir, opts := feedbackReportFixture(t)
	query := querygen.QueryCodegenPlanQuery{Name: "List", Catalog: "good", Kind: "sql", SQL: "SELECT Id FROM T WHERE Id=@id", Params: []querygen.QueryCodegenParam{{Name: "id", Type: "INT64", Optional: "omit_when_null"}}, Variants: []querygen.QueryCodegenPlanQueryVariant{
		{Label: "(none)", SQL: "SELECT Id FROM T", AbsentParams: []string{"id"}},
		{Label: "id", SQL: "SELECT Id FROM T WHERE Id=@id", PresentParams: []string{"id"}},
	}}
	analyzer := &feedbackPlanAnalyzer{}
	open := func(context.Context, spanemuboost.RuntimeHandle, querygen.QueryCodegenSchema, string, *spanalyzer.ProtoDescriptorSet, planReportOptimizer) (planReportAnalyzer, error) {
		return analyzer, nil
	}
	report, err := buildPlanReportWithAnalyzer(t.Context(), config, &querygen.QueryCodegenPlan{Queries: []querygen.QueryCodegenPlanQuery{query}}, dir, opts, nil, open)
	if err != nil || len(report.Queries) != 2 || report.TargetSummary.Planned != 2 || len(analyzer.statements[0].Params) != 0 || len(analyzer.statements[1].Params) != 1 {
		t.Fatalf("variants = %#v, %v", report, err)
	}
	if report.Queries[0].TargetID == report.Queries[1].TargetID || report.Queries[1].TargetID != "query/List" || report.Queries[0].Variant.Label != "(none)" {
		t.Fatalf("variant identities = %#v", report.Queries)
	}
}

func TestCapturedPlanAccessPathKeepsSeekAndResidualColumns(t *testing.T) {
	plan, _, _ := loadPlanReportOperatorFixture(t, "managed_ai_if_scalar_filter_control")
	paths := plancontract.NormalizeAccessPaths(plan, nil)
	if len(paths) != 1 || paths[0].ScanTarget != "Songs" || !slices.Equal(paths[0].SeekColumns, []string{"SingerId"}) || !slices.Equal(paths[0].ResidualColumns, []string{"SongName"}) || !paths[0].HasSeekCondition || paths[0].RangeSeekableKeySize != nil {
		t.Fatalf("captured access paths = %#v", paths)
	}
}

func TestStructuredPlanParametersPreserveTypeAndValue(t *testing.T) {
	values, err := planReportParamValues([]querygen.QueryCodegenParam{{Name: "items", Type: "ARRAY<STRUCT<id INT64, nested STRUCT<label STRING, tags ARRAY<STRING>>>>"}})
	if err != nil {
		t.Fatal(err)
	}
	value, ok := values["items"].(spanner.GenericColumnValue)
	if !ok || value.Type.GetCode() != spannerpb.TypeCode_ARRAY || value.Type.GetArrayElementType().GetStructType().GetFields()[0].GetName() != "id" {
		t.Fatalf("typed value = %#v", values)
	}
	want, err := structpb.NewValue([]interface{}{[]interface{}{"1", []interface{}{"value", []interface{}{"value"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(value.Value, want) {
		t.Fatalf("nested typed parameter = %v, want %v", value.Value, want)
	}
	simple, err := planReportParamValues([]querygen.QueryCodegenParam{{Name: "items", Type: "ARRAY<STRUCT<id INT64>>"}})
	if err != nil {
		t.Fatal(err)
	}
	var decoded []*struct {
		Id int64 `spanner:"id"`
	}
	if err := simple["items"].(spanner.GenericColumnValue).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1 || decoded[0].Id != 1 {
		t.Fatalf("decoded = %#v", decoded)
	}

}

func TestSeekabilityRenderingLeavesOriginalPlanUntouched(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{Index: 0, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Filter Scan", Metadata: mustStructPB(t, map[string]interface{}{"seekable_key_size": "1"}), ChildLinks: []*spannerpb.PlanNode_ChildLink{{ChildIndex: 1}}},
		{Index: 1, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Scan", Metadata: mustStructPB(t, map[string]interface{}{"scan_type": "TableScan", "scan_target": "T"})},
	}}
	before := proto.Clone(plan)
	paths := plancontract.NormalizeAccessPaths(plan, []plancontract.ScanTarget{{Name: "T", Kind: "table", BaseTable: "T", KeyColumns: []string{"Id", "SubId"}}})
	text, err := renderPlanReportTree(plan, paths, planReportOptions{AnnotateSeekability: true, Format: reference.FormatCurrent, RenderMode: reference.RenderModePlan})
	if err != nil || !strings.Contains(text, "1/2") || !proto.Equal(before, plan) {
		t.Fatalf("rendered %q, error %v, mutated %t", text, err, !proto.Equal(before, plan))
	}
}

func TestPlanReportRejectsUnsupportedProtoParameters(t *testing.T) {
	_, err := planReportParamValue(&spanalyzer.TypeSpec{Code: spannerpb.TypeCode_PROTO, ProtoTypeFQN: "example.Message"})
	if err == nil {
		t.Fatal("unsupported PROTO parameter must fail explicitly")
	}
}
