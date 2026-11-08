package main

import "sort"

func planAccessPathJSONSchema() map[string]interface{} {
	properties := map[string]interface{}{
		"scan_index":              nonNegativeIntegerSchema("Owning Scan PlanNode index."),
		"filter_index":            map[string]interface{}{"type": []interface{}{"integer", "null"}, "minimum": 0},
		"scan_target":             stringSchema("Raw scan target identity."),
		"scan_type":               stringSchema("Normalized backend scan type."),
		"scan_target_kind":        stringSchema("Catalog target kind, explicit vector component kind, or unknown."),
		"base_table":              stringSchema("Catalog-resolved base table, or empty when unknown."),
		"seekable_key_size":       stringSchema("Raw range-key metadata; zero does not mean no seek."),
		"range_seekable_key_size": map[string]interface{}{"type": []interface{}{"integer", "null"}, "minimum": 1},
		"declared_key_count":      map[string]interface{}{"type": []interface{}{"integer", "null"}, "minimum": 0},
		"null_filtered":           map[string]interface{}{"type": []interface{}{"boolean", "null"}},
	}
	for _, field := range []string{"full_scan", "conditions_complete", "has_seek_condition", "has_residual_condition", "has_timestamp_condition", "has_search_condition"} {
		properties[field] = boolSchema("Observed scan signal; conditions_complete requires an unambiguous wrapper pairing.")
	}
	for _, field := range []string{"seek_columns", "residual_columns"} {
		properties[field] = stringListSchema("Sorted, duplicate-free set of proven column references, not declared key order. Use membership tests against key_columns for key prefixes; empty does not prove no columns when bindings are unavailable.")
	}
	properties["key_columns"] = stringListSchema("Catalog key columns in declared key order; empty when schema facts are unavailable.")
	properties["stored_columns"] = stringListSchema("Catalog stored columns; empty when schema facts are unavailable.")
	condition := objectSchema([]interface{}{"index", "expression", "columns"}, map[string]interface{}{
		"index":      nonNegativeIntegerSchema("Immediate condition child PlanNode index."),
		"expression": stringSchema("Verbatim backend short description; empty when unavailable."),
		"columns":    stringListSchema("Sorted, duplicate-free set of column references proven by this scan's Reference bindings."),
	})
	for _, field := range []string{"seek_conditions", "residual_conditions", "timestamp_conditions", "search_conditions"} {
		properties[field] = arraySchemaAllowEmpty(condition)
	}
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	required := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		required = append(required, key)
	}
	return objectSchema(required, properties)
}

func planReportVariantJSONSchema() map[string]interface{} {
	return objectSchema([]interface{}{"label", "sql", "sql_sha256"}, map[string]interface{}{
		"label":              stringSchema("Human-readable optional SQL shape."),
		"sql":                stringSchema("Resolved variant SQL."),
		"sql_sha256":         sha256Schema("SHA-256 of variant SQL."),
		"present_params":     stringListSchema("Present optional parameters."),
		"absent_params":      stringListSchema("Omitted optional parameters."),
		"choice_assignments": map[string]interface{}{"type": "object", "additionalProperties": map[string]interface{}{"type": "string"}},
	})
}

func queryResultJSONSchema() map[string]interface{} {
	schema := objectSchema(nil, map[string]interface{}{
		"cardinality": enumSchema([]interface{}{"one", "maybe_one", "many", "row_count", "row_set"}, "Read cardinality or DML result mode. Defaults to many."),
		"struct":      stringSchema("Generated or shared result struct; not required for row_count."),
		"required": objectSchema(nil, map[string]interface{}{
			"policy": enumSchema([]interface{}{"override", "strict"}, "Requiredness override policy."),
			"fields": stringArraySchema("Result fields required by user override."),
		}),
	})
	schema["allOf"] = []interface{}{map[string]interface{}{
		"if":   map[string]interface{}{"properties": map[string]interface{}{"cardinality": map[string]interface{}{"const": "row_count"}}, "required": []interface{}{"cardinality"}},
		"then": map[string]interface{}{},
		"else": map[string]interface{}{"required": []interface{}{"struct"}},
	}}
	return schema
}
