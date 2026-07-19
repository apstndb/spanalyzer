package planvocab_test

import (
	"fmt"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanalyzer/plancontract/planvocab"
)

func ExampleInspect() {
	plan := &spannerpb.QueryPlan{PlanNodes: []*spannerpb.PlanNode{{
		Index:       0,
		Kind:        spannerpb.PlanNode_RELATIONAL,
		DisplayName: "Future Exchange",
	}}}

	for _, finding := range planvocab.Inspect(plan) {
		fmt.Println(finding.DisplayName, finding.Reasons)
	}
	// Output: Future Exchange [operator_family_unknown]
}
