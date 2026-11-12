package querygen

import (
	spanalyzer "github.com/apstndb/spanalyzer"
	"slices"
)

// ExpandPlanReportQueries returns one target per optional SQL shape. The
// canonical all-on shape keeps the query's original target identity.
func ExpandPlanReportQueries(queries []QueryCodegenPlanQuery) []QueryCodegenPlanQuery {
	out := []QueryCodegenPlanQuery{}
	for _, query := range queries {
		if len(query.Variants) == 0 {
			out = append(out, query)
			continue
		}
		canonical := pickCanonicalVariant(query.Variants)
		for _, variant := range query.Variants {
			target := query
			target.Variants = nil
			target.SQL, target.SQLSHA256 = variant.SQL, variant.SQLSHA256
			target.ReportVariant = &variant
			target.ReportVariantCanonical = variant.Label == canonical.Label
			target.Params = nil
			for _, param := range query.Params {
				if !slices.Contains(variant.AbsentParams, param.Name) {
					target.Params = append(target.Params, param)
				}
			}
			out = append(out, target)
		}
	}
	return out
}

// BuildQueryCodegenReportPlan resolves each configured query independently.
// Unlike generation, reporting retains an invalid target so another query or
// catalog can still be inspected. Public config-shape validation remains strict.
func BuildQueryCodegenReportPlan(config QueryCodegenConfig, baseDir string) *QueryCodegenPlan {
	plan := &QueryCodegenPlan{PlanVersion: 1}
	for _, query := range config.Queries {
		one := config
		one.Queries, one.Writes = []QueryCodegenQuery{query}, nil
		one.Schemas = reportPlanSchemas(config.Schemas, query.Catalog, query.Federated.SpannerSource)
		resolved, err := BuildQueryCodegenPlan(one, baseDir)
		if err != nil {
			plan.Queries = append(plan.Queries, QueryCodegenPlanQuery{
				Name: query.Name, Catalog: query.Catalog, Kind: queryKind(query), Result: queryResultMode(query),
				SQL: query.SQL, Params: query.Params, PlanningError: err.Error(),
			})
			continue
		}
		plan.Queries = append(plan.Queries, resolved.Queries...)
	}
	for _, write := range config.Writes {
		one := config
		one.Queries, one.Writes = nil, []QueryCodegenWrite{write}
		one.Schemas = reportPlanSchemas(config.Schemas, write.Catalog)
		target := QueryCodegenPlanQuery{Name: write.Name, Catalog: write.Catalog, Kind: "write", ReportScope: "write", Result: "row_count"}
		resolved, err := resolveQueryCodegen(one, baseDir)
		if err != nil {
			target.PlanningError = err.Error()
			plan.Queries = append(plan.Queries, target)
			continue
		}
		spec := resolved.writeSpecs[0]
		var schema QueryCodegenSchema
		for _, candidate := range one.Schemas {
			if candidate.Name == spec.Catalog {
				schema = candidate
				break
			}
		}
		model, countErr := mutationCountCatalog(schema, baseDir)
		var counts map[string]spanalyzer.MutationCountReport
		if countErr == nil && model != nil {
			counts, countErr = writeMutationCounts(model, spec)
		}
		countError := ""
		if countErr != nil {
			countError = countErr.Error()
		}
		plan.Writes = append(plan.Writes, newMutationCountPlanWrite(spec, counts, countError))
		target.ReportWriteParamColumns = map[string]string{}
		for _, column := range spec.InputColumns {
			target.ReportWriteParamColumns[column.Name] = writeParamName(spec, column)
		}
		if !slices.Contains(spec.Methods, "dml") {
			target.ReportSkipReason = "mutation-only write has no SQL PLAN target"
			plan.Queries = append(plan.Queries, target)
			continue
		}
		target.SQL = writeDMLSQL(spec)
		target.SQLSHA256 = digestString(target.SQL)
		for _, column := range spec.InputColumns {
			typeSQL, err := typeSpecSQL(column.Type)
			if err != nil {
				target.PlanningError = err.Error()
				break
			}
			target.Params = append(target.Params, QueryCodegenParam{Name: writeParamName(spec, column), Type: typeSQL})
		}
		plan.Queries = append(plan.Queries, target)
	}
	return plan
}

// Keep only the target's catalog dependency closure. Digesting an unrelated
// missing DDL file must not turn every otherwise-valid target into an error.
func reportPlanSchemas(schemas []QueryCodegenSchema, names ...string) []QueryCodegenSchema {
	needed := map[string]bool{}
	for _, name := range names {
		needed[name] = true
	}
	for changed := true; changed; {
		changed = false
		for _, schema := range schemas {
			if !needed[schema.Name] {
				continue
			}
			var sources []string
			for _, external := range append(slices.Clone(schema.ExternalSchemas), schema.ExternalQueryConnections...) {
				sources = append(sources, external.SpannerSource)
			}
			for _, dataset := range schema.SpannerExternalDatasets {
				sources = append(sources, dataset.SpannerSource)
			}
			for _, source := range sources {
				if source != "" && !needed[source] {
					needed[source], changed = true, true
				}
			}
		}
	}
	out := []QueryCodegenSchema{}
	for _, schema := range schemas {
		if needed[schema.Name] {
			out = append(out, schema)
		}
	}
	return out
}
