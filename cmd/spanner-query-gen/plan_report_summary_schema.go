package main

func planReportSummaryJSONSchema() map[string]interface{} {
	// Reuse the evidence definitions without weakening the full-report schema's
	// required plan fields or normalized-contract replay guarantee.
	schema := planReportJSONSchema()
	schema["$id"] = "https://github.com/apstndb/spanalyzer/schemas/spanner-query-gen.plan-summary.v1alpha.schema.json"
	schema["title"] = "spanner-query-gen plan-report summary output"
	schema["description"] = "Lightweight structured evidence. Full operator graphs and SQL are omitted; contract results cannot generally be replayed from this artifact."
	properties := schema["properties"].(map[string]interface{})
	properties["report_version"] = map[string]interface{}{"const": planReportSummaryVersionV1Alpha}
	delete(properties, "format")
	delete(properties, "render_mode")
	var required []interface{}
	for _, field := range schema["required"].([]interface{}) {
		if field != "format" && field != "render_mode" {
			required = append(required, field)
		}
	}
	schema["required"] = required
	defs := schema["$defs"].(map[string]interface{})
	defs["plan_source"] = objectSchema([]interface{}{"backend", "api"}, map[string]interface{}{
		"backend": enumSchema([]interface{}{"omni"}, "Runtime backend used to obtain query plans."),
		"api":     enumSchema([]interface{}{"analyze_query"}, "Plan acquisition API."),
	})
	fullQueryProperties := planReportQueryJSONSchema()["properties"].(map[string]interface{})
	queryProperties := map[string]interface{}{}
	for _, field := range []string{"target_id", "name", "catalog", "scope", "kind", "status", "sql_sha256", "ddl_sha256", "proto_descriptor_sha256", "operator_tree_sha256", "access_paths", "variant", "error", "optimizer_not_pinned", "plan_environment_notes", "classification_warnings"} {
		queryProperties[field] = fullQueryProperties[field]
	}
	query := objectSchema([]interface{}{"target_id", "name", "catalog", "kind", "status"}, queryProperties)
	successOnly := []string{"operator_tree_sha256", "access_paths", "classification_warnings"}
	query["allOf"] = []interface{}{
		planReportQueryStatusRequiredClause("ok", []interface{}{"sql_sha256", "ddl_sha256", "operator_tree_sha256", "access_paths"}, nil),
		planReportQueryStatusRequiredClause("error", []interface{}{"error"}, successOnly),
		planReportQueryStatusRequiredClause("skipped", []interface{}{"error"}, successOnly),
	}
	defs["query"] = query
	variant := planReportVariantJSONSchema()
	variant["required"] = []interface{}{"label"}
	variantProperties := variant["properties"].(map[string]interface{})
	delete(variantProperties, "sql")
	delete(variantProperties, "sql_sha256")
	defs["variant"] = variant
	defs["contract_stability"] = planReportContractStabilitySchema(false)
	return schema
}
