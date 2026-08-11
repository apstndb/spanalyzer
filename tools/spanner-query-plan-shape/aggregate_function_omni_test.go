//go:build integration && omni

package main

import (
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanemuboost"
	"google.golang.org/protobuf/proto"
)

func TestIntegrationAggregateFunctionAggTypesVersionMatrixOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	image := strings.TrimSpace(os.Getenv("SPANALYZER_OMNI_IMAGE"))
	if image == "" {
		t.Fatal("set SPANALYZER_OMNI_IMAGE to the pinned Spanner Omni image under test")
	}
	ddls, err := parseBuiltInDDLs("aggregate-function-schema.sql", docsDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	runtime := spanemuboost.NewLazyRuntime(
		spanemuboost.BackendOmni,
		spanemuboost.WithContainerImage(image),
	)
	t.Cleanup(func() { _ = runtime.Close() })
	clients, err := spanemuboost.OpenClients(t.Context(), runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(ddls),
	)
	if err != nil {
		t.Fatalf("OpenClients() error = %v", err)
	}
	t.Cleanup(func() { _ = clients.Close() })

	cases := queryCasesByLabel(t, aggregateFunctionQueries)
	wantAggTypes := map[string]map[string][]string{
		"aggregate-function/general/any-value":             {"Global": {"ANY"}, "Local": {"ANY"}},
		"aggregate-function/general/array-agg":             {"Global": {"ARRAY_CONCAT_AGG"}, "Local": {"ARRAY_AGG"}},
		"aggregate-function/general/array-concat-agg":      {"Global": {"ARRAY_CONCAT_AGG"}, "Local": {"ARRAY_CONCAT_AGG"}},
		"aggregate-function/general/avg":                   {"Global": {"AVG_FINAL"}, "Local": {"AVG_START"}},
		"aggregate-function/general/bit-and":               {"Global": {"BIT_AND"}, "Local": {"BIT_AND"}},
		"aggregate-function/general/bit-or":                {"Global": {"BIT_OR"}, "Local": {"BIT_OR"}},
		"aggregate-function/general/bit-xor":               {"Global": {"BIT_XOR"}, "Local": {"BIT_XOR"}},
		"aggregate-function/general/count":                 {"Global": {"COUNT_FINAL"}, "Local": {"COUNT"}},
		"aggregate-function/general/countif":               {"Global": {"COUNT_FINAL"}, "Local": {"COUNTIF"}},
		"aggregate-function/general/logical-and":           {"Global": {"LOGICAL_AND"}, "Local": {"LOGICAL_AND"}},
		"aggregate-function/general/logical-or":            {"Global": {"LOGICAL_OR"}, "Local": {"LOGICAL_OR"}},
		"aggregate-function/general/max":                   {"Global": {"MAX"}, "Local": {"MAX"}},
		"aggregate-function/general/min":                   {"Global": {"MIN"}, "Local": {"MIN"}},
		"aggregate-function/statistical/stddev":            {"Global": {"STDDEV_FINAL"}, "Local": {"STDDEV_START"}},
		"aggregate-function/statistical/stddev-samp":       {"Global": {"STDDEV_FINAL"}, "Local": {"STDDEV_START"}},
		"aggregate-function/general/string-agg":            {"Global": {"STRING_AGG"}, "Local": {"STRING_AGG"}},
		"aggregate-function/general/sum":                   {"Global": {"SUM"}, "Local": {"SUM"}},
		"aggregate-function/statistical/var-samp":          {"Global": {"STDDEV_FINAL"}, "Local": {"STDDEV_START"}},
		"aggregate-function/statistical/variance":          {"Global": {"STDDEV_FINAL"}, "Local": {"STDDEV_START"}},
		"aggregate-function/modifier/any-value-having-max": {"Global": {"HAVING_MAX"}, "Local": {"HAVING_MAX", "MAX"}},
		"aggregate-function/modifier/any-value-having-min": {"Global": {"HAVING_MIN"}, "Local": {"HAVING_MIN", "MIN"}},
		"aggregate-function/modifier/count-star":           {"Global": {"COUNT_FINAL"}, "Local": {"COUNT"}},
		"aggregate-function/modifier/count-distinct":       {"": {"COUNT"}},
		"aggregate-function/modifier/array-agg-distinct-ordered-limited": {
			"": {"NEST"}, "Global": {"ARRAY_CONCAT_AGG"}, "Local": {"ARRAY_AGG"},
		},
		"aggregate-function/modifier/avg-distinct": {"": {"AVG"}},
		"aggregate-function/modifier/string-agg-distinct-ordered-limited": {
			"": {"NEST"}, "Global": {"ARRAY_CONCAT_AGG"}, "Local": {"ARRAY_AGG"},
		},
		"aggregate-function/control/grouped-count-sum":           {"": {"COUNT", "SUM"}},
		"aggregate-function/control/group-by-without-agg":        {},
		"aggregate-function/control/select-distinct-without-agg": {},
	}

	for version := 1; version <= 8; version++ {
		t.Run("v"+strconv.Itoa(version), func(t *testing.T) {
			plans := make(map[string]*spannerpb.QueryPlan, len(wantAggTypes))
			for label, want := range wantAggTypes {
				query := cases[label]
				query.SQL = withOptimizerVersionStatementHint(query.SQL, version)
				plan, err := analyzePlan(t.Context(), clients.Client, query)
				if err != nil {
					t.Fatalf("AnalyzeQuery(%s, v%d) error = %v", label, version, err)
				}
				plans[label] = plan
				if got := aggregateAggTypes(plan); !reflect.DeepEqual(got, want) {
					t.Errorf("%s v%d Agg types = %#v, want %#v", label, version, got, want)
				}
			}

			if !proto.Equal(plans["aggregate-function/statistical/stddev"], plans["aggregate-function/statistical/stddev-samp"]) {
				t.Errorf("STDDEV and STDDEV_SAMP plans differ at v%d", version)
			}
			if !proto.Equal(plans["aggregate-function/statistical/var-samp"], plans["aggregate-function/statistical/variance"]) {
				t.Errorf("VAR_SAMP and VARIANCE plans differ at v%d", version)
			}
			for _, label := range []string{
				"aggregate-function/statistical/var-samp",
				"aggregate-function/statistical/variance",
			} {
				if !planHasScalarFunctionType(plans[label], "POW") {
					t.Errorf("%s v%d lacks the non-Agg POW finalization", label, version)
				}
			}
			if label := "aggregate-function/modifier/string-agg-distinct-ordered-limited"; !planHasScalarFunctionType(plans[label], "ARRAY_TO_STRING") {
				t.Errorf("%s v%d lacks the non-Agg ARRAY_TO_STRING finalization", label, version)
			}

			for label, wantError := range map[string]string{
				"aggregate-function/unsupported/approx-count-distinct": "Unsupported aggregate function: approx_count_distinct",
				"aggregate-function/unsupported/corr":                  "Unsupported aggregate function: corr",
			} {
				query := cases[label]
				query.SQL = withOptimizerVersionStatementHint(query.SQL, version)
				if _, err := analyzePlan(t.Context(), clients.Client, query); err == nil || !strings.Contains(err.Error(), wantError) {
					t.Errorf("AnalyzeQuery(%s, v%d) error = %v, want containing %q", label, version, err, wantError)
				}
			}
		})
	}
}

func aggregateAggTypes(plan *spannerpb.QueryPlan) map[string][]string {
	nodes := make(map[int32]*spannerpb.PlanNode, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		nodes[node.GetIndex()] = node
	}
	types := make(map[string][]string)
	for _, aggregate := range plan.GetPlanNodes() {
		if aggregate.GetDisplayName() != "Aggregate" {
			continue
		}
		callType := aggregate.GetMetadata().GetFields()["call_type"].GetStringValue()
		for _, link := range aggregate.GetChildLinks() {
			if link.GetType() != "Agg" {
				continue
			}
			child := nodes[link.GetChildIndex()]
			if child == nil {
				continue
			}
			types[callType] = append(types[callType], scalarFunctionType(child.GetShortRepresentation().GetDescription()))
		}
	}
	for callType := range types {
		slices.Sort(types[callType])
	}
	return types
}

func scalarFunctionType(description string) string {
	if end := strings.IndexByte(description, '('); end >= 0 {
		return description[:end]
	}
	return description
}

func planHasScalarFunctionType(plan *spannerpb.QueryPlan, want string) bool {
	for _, node := range plan.GetPlanNodes() {
		if node.GetKind() == spannerpb.PlanNode_SCALAR && scalarFunctionType(node.GetShortRepresentation().GetDescription()) == want {
			return true
		}
	}
	return false
}
