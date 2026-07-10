package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spannerplan/plantree/reference"
)

func TestParsePlanReportAnnotations(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    planReportAnnotateOptions
		wantErr bool
	}{
		{name: "empty", value: "", want: planReportAnnotateOptions{}},
		{name: "seekability", value: "seekability", want: planReportAnnotateOptions{Seekability: true}},
		{name: "families", value: "families", want: planReportAnnotateOptions{Families: true}},
		{name: "both with spaces and case", value: " Seekability , FAMILIES ", want: planReportAnnotateOptions{Seekability: true, Families: true}},
		{name: "trailing comma", value: "seekability,", want: planReportAnnotateOptions{Seekability: true}},
		{name: "unknown", value: "seekability,bogus", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parsePlanReportAnnotations(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatalf("parsePlanReportAnnotations(%q) error = nil, want error", test.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePlanReportAnnotations(%q) error = %v", test.value, err)
			}
			if got != test.want {
				t.Errorf("parsePlanReportAnnotations(%q) = %+v, want %+v", test.value, got, test.want)
			}
		})
	}
}

func TestPlanReportCatalogKeyCountsUsesDeclaredKeys(t *testing.T) {
	ddl := `
CREATE TABLE Foo (
  random_id STRING(22) NOT NULL,
  shard_id INT64 NOT NULL,
  timestamp_order TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
) PRIMARY KEY(random_id);
CREATE INDEX OrderIndexDesc ON Foo(shard_id, timestamp_order DESC);
`
	catalog, err := spanalyzer.BuildSchemaCatalog("test.sql", ddl)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	counts := planReportCatalogKeyCounts(catalog)
	if got, want := counts["Foo"], 1; got != want {
		t.Errorf("counts[Foo] = %d, want %d", got, want)
	}
	// Declared key parts only: the implicit base-table primary key suffix of
	// a secondary index is intentionally not counted.
	if got, want := counts["OrderIndexDesc"], 2; got != want {
		t.Errorf("counts[OrderIndexDesc] = %d, want %d", got, want)
	}
}

func TestPlanReportSchemaKeyCountsWithoutDDLIsEmpty(t *testing.T) {
	counts, err := planReportSchemaKeyCounts(querygen.QueryCodegenSchema{Name: "main"}, t.TempDir())
	if err != nil {
		t.Fatalf("planReportSchemaKeyCounts() error = %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("counts = %v, want empty", counts)
	}
}

func TestPlanReportSchemaKeyCountsReadsDDLFile(t *testing.T) {
	dir := t.TempDir()
	ddl := "CREATE TABLE Foo (id INT64 NOT NULL, shard INT64 NOT NULL) PRIMARY KEY(shard, id);"
	writeTestFile(t, filepath.Join(dir, "schema.sql"), ddl)
	counts, err := planReportSchemaKeyCounts(querygen.QueryCodegenSchema{Name: "main", DDL: "schema.sql"}, dir)
	if err != nil {
		t.Fatalf("planReportSchemaKeyCounts() error = %v", err)
	}
	if got, want := counts["Foo"], 2; got != want {
		t.Errorf("counts[Foo] = %d, want %d", got, want)
	}
}

// seekabilityTestPlan models the observed Filter Scan / Index Scan split:
// seekable_key_size sits on the Filter Scan wrapper while scan_target sits on
// the scan node below it.
func seekabilityTestPlan(t *testing.T) *spannerpb.QueryPlan {
	t.Helper()
	return &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       0,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_type":         "FilterScan",
				"seekable_key_size": "1",
			}),
			ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{ChildIndex: 1},
				{ChildIndex: 2, Type: "Residual Condition"},
			},
		},
		{
			Index:       1,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_type":   "IndexScan",
				"scan_target": "OrderIndexDesc",
			}),
		},
		{
			Index:       2,
			Kind:        spannerpb.PlanNode_SCALAR,
			DisplayName: "Function",
		},
	}}
}

func TestPlanReportSeekableKeyValuesResolveTargetFromSubtree(t *testing.T) {
	plan := seekabilityTestPlan(t)
	got := planReportSeekableKeyValues(plan, map[string]int{"OrderIndexDesc": 2, "Foo": 1})
	want := map[int32]string{0: "1/2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("planReportSeekableKeyValues() = %v, want %v", got, want)
	}
}

func TestPlanReportSeekableKeyValuesSkipAmbiguousMultiScanSubtree(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       0,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Filter Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"seekable_key_size": "1",
			}),
			ChildLinks: []*spannerpb.PlanNode_ChildLink{{ChildIndex: 1}, {ChildIndex: 2}},
		},
		{
			Index:       1,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Scan",
			Metadata:    mustStructPB(t, map[string]interface{}{"scan_target": "Orders"}),
		},
		{
			Index:       2,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Scan",
			Metadata:    mustStructPB(t, map[string]interface{}{"scan_target": "OrdersByDate"}),
		},
	}}
	if got := planReportSeekableKeyValues(plan, map[string]int{"Orders": 1, "OrdersByDate": 2}); len(got) != 0 {
		t.Fatalf("planReportSeekableKeyValues() = %v, want no annotation for an ambiguous multi-scan subtree", got)
	}
}

// TestPlanReportSeekableKeyValuesSkipPointSeekZero pins that the ambiguous
// seekable_key_size value 0 is not annotated: it is reported both for plain
// full scans and for pure point seeks (all-equality key conditions), so
// "0/N" would misread a perfect point read as seeking nothing.
func TestPlanReportSeekableKeyValuesSkipPointSeekZero(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       0,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_type":         "FilterScan",
				"seekable_key_size": "0",
			}),
			ChildLinks: []*spannerpb.PlanNode_ChildLink{{ChildIndex: 1}},
		},
		{
			Index:       1,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_type":   "TableScan",
				"scan_target": "Songs",
			}),
		},
	}}
	if got := planReportSeekableKeyValues(plan, map[string]int{"Songs": 3}); len(got) != 0 {
		t.Errorf("planReportSeekableKeyValues() = %v, want empty for seekable_key_size 0", got)
	}
}

func TestPlanReportSeekableKeyValuesKeepRawWhenSizeCannotBeAnnotated(t *testing.T) {
	tests := []struct {
		name string
		size string
	}{
		{name: "not an integer", size: "unknown"},
		{name: "exceeds declared key count", size: "3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
				{
					Index:       0,
					Kind:        spannerpb.PlanNode_RELATIONAL,
					DisplayName: "Scan",
					Metadata: mustStructPB(t, map[string]interface{}{
						"scan_type":         "FilterScan",
						"seekable_key_size": test.size,
					}),
					ChildLinks: []*spannerpb.PlanNode_ChildLink{{ChildIndex: 1}},
				},
				{
					Index:       1,
					Kind:        spannerpb.PlanNode_RELATIONAL,
					DisplayName: "Scan",
					Metadata: mustStructPB(t, map[string]interface{}{
						"scan_type":   "IndexScan",
						"scan_target": "OrderIndexDesc",
					}),
				},
			}}
			if got := planReportSeekableKeyValues(plan, map[string]int{"OrderIndexDesc": 2}); len(got) != 0 {
				t.Errorf("planReportSeekableKeyValues() = %v, want empty so raw seekable_key_size is kept", got)
			}
		})
	}
}

func TestPlanReportSeekableKeyValuesSkipUnknownAndSyntheticTargets(t *testing.T) {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{
		{
			Index:       0,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_type":         "BatchScan",
				"scan_target":       "$BatchPartition",
				"seekable_key_size": "1",
			}),
		},
		{
			Index:       1,
			Kind:        spannerpb.PlanNode_RELATIONAL,
			DisplayName: "Scan",
			Metadata: mustStructPB(t, map[string]interface{}{
				"scan_type":         "TableScan",
				"scan_target":       "NotInCatalog",
				"seekable_key_size": "1",
			}),
		},
	}}
	if got := planReportSeekableKeyValues(plan, map[string]int{"Foo": 1}); len(got) != 0 {
		t.Errorf("planReportSeekableKeyValues() = %v, want empty", got)
	}
}

func TestPlanReportFamilyAnnotations(t *testing.T) {
	plan := seekabilityTestPlan(t)
	operators := planReportOperators(plan)
	annotations := planReportFamilyAnnotations(operators)
	if got, want := annotations[0], "{filter_scan}"; got != want {
		t.Errorf("annotations[0] = %q, want %q", got, want)
	}
	if got, want := annotations[1], "{scan}"; got != want {
		t.Errorf("annotations[1] = %q, want %q", got, want)
	}
}

// TestPlanReportFamilyAnnotationsIncludeUmbrellaFamilies pins the DAG-shaped
// labels: the single-valued concrete family on the left of the colon, the
// derived umbrella families it contributes to on the right, so the two sorts
// are distinguishable without knowing a position convention.
func TestPlanReportFamilyAnnotationsIncludeUmbrellaFamilies(t *testing.T) {
	annotations := planReportFamilyAnnotations([]planReportOperator{
		{Index: 0, Family: "full_sort"},
		{Index: 1, Family: "minor_sort"},
		{Index: 2, Family: "hash_join"},
		{Index: 3, Family: "stream_aggregate"},
		{Index: 4, Family: "stream_aggregate", ScalarAggregate: true},
	})
	want := map[int32]string{
		0: "{full_sort: blocking_operator, explicit_sort}",
		1: "{minor_sort: explicit_sort}",
		2: "{hash_join: blocking_operator}",
		3: "{stream_aggregate}",
		4: "{stream_aggregate: blocking_operator}",
	}
	if !reflect.DeepEqual(annotations, want) {
		t.Errorf("planReportFamilyAnnotations() = %v, want %v", annotations, want)
	}
}

func TestPlanReportRenderAnnotationOptionsDisabledAreEmpty(t *testing.T) {
	plan := seekabilityTestPlan(t)
	operators := planReportOperators(plan)
	if got := planReportRenderAnnotationOptions(plan, operators, map[string]int{"OrderIndexDesc": 2}, planReportAnnotateOptions{}); len(got) != 0 {
		t.Errorf("planReportRenderAnnotationOptions() = %v, want empty", got)
	}
}

// TestPlanReportRenderAnnotationOptionsRenderedRow pins the rendered row text
// end to end: seekability replaces the seekable_key_size metadata value in
// place ("1" -> "1/2") instead of duplicating it as a row suffix, while the
// family annotation appends because it has no metadata counterpart.
func TestPlanReportRenderAnnotationOptionsRenderedRow(t *testing.T) {
	plan := seekabilityTestPlan(t)
	operators := planReportOperators(plan)
	renderOptions := append(
		[]reference.Option{reference.WithWrapWidth(0)},
		planReportRenderAnnotationOptions(plan, operators, map[string]int{"OrderIndexDesc": 2}, planReportAnnotateOptions{Seekability: true, Families: true})...,
	)
	rendered, err := reference.RenderTreeTableWithOptions(plan.GetPlanNodes(), reference.RenderModePlan, reference.FormatCurrent, renderOptions...)
	if err != nil {
		t.Fatalf("RenderTreeTableWithOptions() error = %v", err)
	}
	if want := "Filter Scan (seekable_key_size: 1/2) {filter_scan}"; !strings.Contains(rendered, want) {
		t.Errorf("rendered output does not contain %q:\n%s", want, rendered)
	}
	if strings.Contains(rendered, "seekable_key_size: 1)") {
		t.Errorf("rendered output still contains the unreplaced seekable_key_size value:\n%s", rendered)
	}
}
