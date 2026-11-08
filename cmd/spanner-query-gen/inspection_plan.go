package main

import (
	"errors"

	"github.com/apstndb/spanalyzer/internal/querygen"
)

type inspectionPlan struct {
	Plan   *querygen.QueryCodegenPlan
	Errors []vetDiagnostic
	cause  error
}

// Inspection is CLI orchestration over the existing strict and report planners,
// not another generation resolver. Valid inputs take the unchanged strict path.
// Failed inputs reuse report target isolation, then rebuild the valid subset to
// retain catalog warnings, suppressions, structs and write metadata where possible.
func buildInspectionPlan(config querygen.QueryCodegenConfig, baseDir string) inspectionPlan {
	plan, err := querygen.BuildQueryCodegenPlan(config, baseDir)
	if err == nil {
		return inspectionPlan{Plan: plan}
	}
	partial := querygen.BuildQueryCodegenReportPlan(config, baseDir)
	result := inspectionPlan{Plan: partial}
	valid := config
	valid.Queries, valid.Writes = nil, nil
	var causes []error
	for i, target := range partial.Queries {
		isQuery := i < len(config.Queries)
		if target.PlanningError == "" {
			if isQuery {
				valid.Queries = append(valid.Queries, config.Queries[i])
			} else {
				valid.Writes = append(valid.Writes, config.Writes[i-len(config.Queries)])
			}
			continue
		}
		// The report API preserves error text. Recover the typed diagnostic
		// through the strict API when its target error is the same; unrelated
		// catalog errors must not replace the isolated target's actual cause.
		one := config
		one.Queries, one.Writes = nil, nil
		scope := "query"
		if isQuery {
			one.Queries = []querygen.QueryCodegenQuery{config.Queries[i]}
		} else {
			scope = "write"
			one.Writes = []querygen.QueryCodegenWrite{config.Writes[i-len(config.Queries)]}
		}
		_, typedErr := querygen.BuildQueryCodegenPlan(one, baseDir)
		cause := errors.New(target.PlanningError)
		if typedErr != nil && typedErr.Error() == target.PlanningError {
			cause = typedErr
		}
		diagnostic := vetDiagnosticFromError(cause)
		diagnostic.Scope, diagnostic.Name = scope, target.Name
		result.Errors = append(result.Errors, diagnostic)
		causes = append(causes, cause)
	}
	// Report PLAN targets include synthetic DML/mutation entries. Explain-plan
	// keeps declared queries and the strict planner's separate Writes projection.
	queries := append([]querygen.QueryCodegenPlanQuery(nil), partial.Queries[:len(config.Queries)]...)
	partial.Queries = queries
	partial.Package, partial.Out, partial.Client = config.Package, config.Out, config.Client
	if len(valid.Queries)+len(valid.Writes) > 0 {
		resolved, resolveErr := querygen.BuildQueryCodegenPlan(valid, baseDir)
		if resolveErr == nil {
			j := 0
			for i := range queries {
				if queries[i].PlanningError == "" {
					queries[i] = resolved.Queries[j]
					j++
				}
			}
			resolved.Queries = queries
			result.Plan = resolved
		} else {
			// Cross-target namespace/DTO conflicts and independent catalog
			// failures remain unsuppressible plan errors. Partial target data
			// must never make a globally invalid generation plan look valid.
			result.Errors = append(result.Errors, vetDiagnosticFromError(resolveErr))
			causes = append(causes, resolveErr)
		}
	} else if len(result.Errors) == 0 {
		result.Errors = append(result.Errors, vetDiagnosticFromError(err))
		causes = append(causes, err)
	}
	result.cause = errors.Join(causes...)
	return result
}
