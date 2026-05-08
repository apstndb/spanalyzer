package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/apstndb/go-googlesql-spanner-poc/internal/querygen"
)

const v1AlphaIdentifierPattern = `^[A-Za-z_][A-Za-z0-9_]*$`

func runConfigSchema(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("spanner-query-gen config-schema", flag.ContinueOnError)
	fs.SetOutput(flagOutput(args, stdout, stderr))
	output := fs.String("output", "json", "schema output format: json or yaml")
	outPath := fs.String("out", "", "write schema to file instead of stdout; use - for stdout")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	data, err := configSchemaBytes(*output)
	if err != nil {
		return err
	}
	return writeSchemaOutput(*outPath, data, stdout)
}

func runPlanReportSchema(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("spanner-query-gen plan-report-schema", flag.ContinueOnError)
	fs.SetOutput(flagOutput(args, stdout, stderr))
	output := fs.String("output", "json", "schema output format: json or yaml")
	outPath := fs.String("out", "", "write schema to file instead of stdout; use - for stdout")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	data, err := planReportSchemaBytes(*output)
	if err != nil {
		return err
	}
	return writeSchemaOutput(*outPath, data, stdout)
}

func runPlanContractSchema(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("spanner-query-gen plan-contract-schema", flag.ContinueOnError)
	fs.SetOutput(flagOutput(args, stdout, stderr))
	output := fs.String("output", "json", "schema output format: json or yaml")
	outPath := fs.String("out", "", "write schema to file instead of stdout; use - for stdout")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	data, err := planContractSchemaBytes(*output)
	if err != nil {
		return err
	}
	return writeSchemaOutput(*outPath, data, stdout)
}

func configSchemaBytes(output string) ([]byte, error) {
	schema := v1AlphaConfigJSONSchema()
	return schemaBytes(output, schema)
}

func planReportSchemaBytes(output string) ([]byte, error) {
	schema := planReportJSONSchema()
	return schemaBytes(output, schema)
}

func planContractSchemaBytes(output string) ([]byte, error) {
	schema := planContractJSONSchema()
	return schemaBytes(output, schema)
}

func schemaBytes(output string, schema map[string]interface{}) ([]byte, error) {
	switch output {
	case "json":
		return marshalIndentedJSON(schema)
	case "yaml":
		return marshalYAMLViaJSON(schema)
	default:
		return nil, fmt.Errorf("unsupported --output %q", output)
	}
}

func writeSchemaOutput(path string, data []byte, stdout io.Writer) error {
	if path == "" || path == "-" {
		_, err := stdout.Write(data)
		return err
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func planContractJSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  "https://github.com/apstndb/go-googlesql-spanner-poc/schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json",
		"title":                "spanner-query-gen plan contracts v1alpha",
		"type":                 "object",
		"additionalProperties": false,
		"required":             []interface{}{"version", "contracts"},
		"properties": map[string]interface{}{
			"version": map[string]interface{}{
				"const":       planContractFileVersionV1Alpha,
				"description": "Plan contract file version.",
			},
			"contracts": map[string]interface{}{
				"type":        "array",
				"minItems":    1,
				"uniqueItems": true,
				"items":       map[string]interface{}{"$ref": "#/$defs/contract"},
			},
		},
		"$defs": map[string]interface{}{
			"identifier":      identifierSchema("Identifier used in canonical target IDs."),
			"operator_family": enumSchema(interfaceSlice(planReportKnownOperatorFamilies()), "Normalized operator family."),
			"target":          patternStringSchema(planContractTargetIDPattern, "Canonical target ID, for example query/ListSingers or query/ExternalQuery#inner."),
			"contract": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []interface{}{"name", "target"},
				"oneOf": []interface{}{
					map[string]interface{}{"required": []interface{}{"use"}},
					map[string]interface{}{"required": []interface{}{"forbid"}},
					map[string]interface{}{"required": []interface{}{"cel"}},
				},
				"properties": map[string]interface{}{
					"name":   map[string]interface{}{"$ref": "#/$defs/identifier"},
					"target": map[string]interface{}{"$ref": "#/$defs/target"},
					"use": map[string]interface{}{
						"type":        "array",
						"minItems":    1,
						"uniqueItems": true,
						"items":       enumSchema(planContractPredefinedValues(), "Predefined plan contract name."),
					},
					"forbid": map[string]interface{}{
						"type":        "array",
						"minItems":    1,
						"uniqueItems": true,
						"items":       map[string]interface{}{"$ref": "#/$defs/forbid"},
					},
					"cel": stringSchema("CEL expression evaluated against the plan-report contract input."),
				},
			},
			"forbid": objectSchema([]interface{}{"operator_family"}, map[string]interface{}{
				"operator_family": map[string]interface{}{"$ref": "#/$defs/operator_family"},
				"max_count":       nonNegativeIntegerSchemaWithDefault("Maximum allowed count; omitted means 0.", 0),
			}),
		},
	}
}

func planReportJSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  "https://github.com/apstndb/go-googlesql-spanner-poc/schemas/spanner-query-gen.plan-report.v1alpha.schema.json",
		"title":                "spanner-query-gen plan-report output",
		"type":                 "object",
		"additionalProperties": false,
		"required": []interface{}{
			"report_version",
			"status",
			"backend",
			"input",
			"plan_source",
			"backend_identity",
			"normalization",
			"target_summary",
			"format",
			"render_mode",
			"stable",
			"queries",
			"contract_evaluation_mode",
			"optimizer",
			"warnings",
		},
		"allOf": []interface{}{map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{
					"contract_evaluation_mode": enumSchema([]interface{}{planContractEvaluationModeReportOnly, planContractEvaluationModeCheck}, "Contract evaluation modes that require contract result fields."),
				},
				"required": []interface{}{"contract_evaluation_mode"},
			},
			"then": map[string]interface{}{
				"required": []interface{}{"contract_file_version", "contract_evaluator_version", "contract_evaluations", "contract_summary"},
				"properties": map[string]interface{}{
					"input":                map[string]interface{}{"required": []interface{}{"contract_file_sha256"}},
					"contract_evaluations": map[string]interface{}{"minItems": 1},
				},
			},
			"else": map[string]interface{}{
				"properties": map[string]interface{}{
					"contract_evaluation_mode": map[string]interface{}{"const": planContractEvaluationModeNone},
				},
				"not": anyRequired("contract_file_version", "contract_evaluator_version", "contract_evaluations", "contract_summary"),
			},
		}},
		"properties": map[string]interface{}{
			"report_version":             map[string]interface{}{"const": planReportVersionV1Alpha, "description": "Plan-report output format version."},
			"status":                     enumSchema([]interface{}{planReportStatusOK, planReportStatusNoTargets}, "Artifact production status. This does not mean every query target was successfully planned; use target_summary and queries[].status for target-level success."),
			"backend":                    enumSchema([]interface{}{"omni"}, "Runtime backend used to obtain query plans."),
			"input":                      map[string]interface{}{"$ref": "#/$defs/input"},
			"plan_source":                map[string]interface{}{"$ref": "#/$defs/plan_source"},
			"backend_identity":           map[string]interface{}{"$ref": "#/$defs/backend_identity"},
			"normalization":              map[string]interface{}{"$ref": "#/$defs/normalization"},
			"target_summary":             map[string]interface{}{"$ref": "#/$defs/target_summary"},
			"format":                     enumSchema([]interface{}{"CURRENT", "TRADITIONAL", "COMPACT"}, "spannerplan tree format used for rendered plan text."),
			"render_mode":                enumSchema([]interface{}{"PLAN"}, "spannerplan render mode used for rendered plan text."),
			"stable":                     boolSchema("Whether stable-output mode was requested."),
			"queries":                    arraySchemaAllowEmpty(map[string]interface{}{"$ref": "#/$defs/query"}),
			"contract_evaluation_mode":   enumSchema([]interface{}{planContractEvaluationModeNone, planContractEvaluationModeReportOnly, planContractEvaluationModeCheck}, "Contract evaluation mode for this report."),
			"contract_file_version":      stringSchema("Plan contract file version used for contract evaluation."),
			"contract_evaluator_version": stringSchema("Plan contract evaluator version used for contract evaluation."),
			"contract_evaluations":       arraySchemaAllowEmpty(map[string]interface{}{"$ref": "#/$defs/contract_evaluation"}),
			"contract_summary":           map[string]interface{}{"$ref": "#/$defs/contract_summary"},
			"warnings":                   arraySchemaAllowEmpty(map[string]interface{}{"$ref": "#/$defs/diagnostic"}),
			"optimizer":                  map[string]interface{}{"$ref": "#/$defs/optimizer"},
		},
		"$defs": map[string]interface{}{
			"operator_family":          enumSchema(interfaceSlice(planReportKnownOperatorFamilies()), "Normalized operator family, including derived umbrella families usable in counts and contracts."),
			"concrete_operator_family": enumSchema(interfaceSlice(planReportConcreteOperatorFamilies()), "Concrete normalized operator family that can appear in normalized_operators[].family."),
			"input": objectSchema([]interface{}{"config_sha256"}, map[string]interface{}{
				"config_sha256":        sha256Schema("SHA-256 of the resolved spanner-query-gen config file bytes."),
				"contract_file_sha256": sha256Schema("SHA-256 of the resolved plan contract file bytes."),
				"contract_file_path":   stringSchema("Relative plan contract file path when stable output does not omit it."),
			}),
			"plan_source": objectSchema([]interface{}{"backend", "api", "render_tool"}, map[string]interface{}{
				"backend":     enumSchema([]interface{}{"omni"}, "Runtime backend used to obtain query plans."),
				"api":         enumSchema([]interface{}{"analyze_query"}, "Plan acquisition API."),
				"render_tool": enumSchema([]interface{}{"spannerplan"}, "Human-readable plan renderer."),
			}),
			"backend_identity": planReportBackendIdentityJSONSchema(),
			"normalization": objectSchema([]interface{}{"operator_tree_version", "operator_family_mapping_version", "cel_input_defaults"}, map[string]interface{}{
				"operator_tree_version":           stringSchema("Normalized operator tree digest input version."),
				"operator_family_mapping_version": stringSchema("Operator family mapping version."),
				"cel_input_defaults":              map[string]interface{}{"$ref": "#/$defs/cel_input_defaults"},
			}),
			"cel_input_defaults": objectSchema([]interface{}{"optional_string", "optional_boolean", "applies_to"}, map[string]interface{}{
				"optional_string": map[string]interface{}{
					"const":       "",
					"description": "Default value applied to absent optional string fields in the internal CEL input.",
				},
				"optional_boolean": constBoolSchema(false, "Default value applied to absent optional boolean fields in the internal CEL input."),
				"applies_to":       celInputDefaultAppliesToSchema(),
			}),
			"target_summary": objectSchema([]interface{}{"included_count", "planned", "errors", "skipped", "excluded"}, map[string]interface{}{
				"included_count": nonNegativeIntegerSchema("Number of query targets included in queries[]. Must equal planned + errors + skipped."),
				"planned":        nonNegativeIntegerSchema("Number of query targets with status ok."),
				"errors":         nonNegativeIntegerSchema("Number of query targets with status error."),
				"skipped":        nonNegativeIntegerSchema("Number of query targets with status skipped."),
				"excluded":       arraySchemaAllowEmpty(map[string]interface{}{"$ref": "#/$defs/excluded_target"}),
			}),
			"excluded_target": objectSchema([]interface{}{"id", "query", "scope", "reason"}, map[string]interface{}{
				"id":     targetIDSchema("Canonical target ID."),
				"query":  stringSchema("Configured query name."),
				"source": stringSchema("Configured or derived query source catalog."),
				"scope":  planReportScopeSchema("Plan-report target scope."),
				"reason": patternStringSchema(`^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$`, "Machine-readable namespaced exclusion reason."),
			}),
			"query": planReportQueryJSONSchema(),
			"operator": objectSchema([]interface{}{"index", "display_name", "family", "child_indexes", "descendant_indexes", "subtree_family_counts"}, map[string]interface{}{
				"index":                 nonNegativeIntegerSchema("PlanNode index."),
				"display_name":          stringSchema("PlanNode display name."),
				"family":                map[string]interface{}{"$ref": "#/$defs/concrete_operator_family"},
				"execution_method":      stringSchema("Normalized execution_method metadata."),
				"iterator_type":         stringSchema("Normalized iterator_type metadata."),
				"scan_method":           stringSchema("Normalized scan_method metadata."),
				"scan_format":           stringSchema("Normalized scan_format metadata."),
				"scan_type":             stringSchema("Normalized scan_type metadata, for example table_scan, index_scan, batch_scan, or search_index_scan."),
				"scan_target":           stringSchema("Raw scan target metadata."),
				"seekable_key_size":     stringSchema("Raw seekable key size metadata."),
				"join_type":             stringSchema("Normalized join_type metadata."),
				"join_configuration":    stringSchema("Normalized join_configuration metadata."),
				"call_type":             stringSchema("Normalized call_type metadata."),
				"distribution_table":    stringSchema("Raw distribution_table metadata."),
				"subquery_cluster_node": stringSchema("Raw subquery_cluster_node metadata."),
				"spool_name":            stringSchema("Raw spool_name metadata for SpoolBuild and SpoolScan operators."),
				"full_scan":             boolSchema("Whether the PlanNode metadata marks a full scan."),
				"child_indexes":         integerListSchema("Direct child PlanNode indexes from child_links."),
				"descendant_indexes":    integerListSchema("Transitive child PlanNode indexes reachable through child_links."),
				"subtree_family_counts": completeOperatorFamilyIntegerMapSchema("Complete operator family counts for this PlanNode subtree, including the operator itself. Contains every operator_family enum value; absent families are represented by 0."),
			}),
			"operator_edge": objectSchema([]interface{}{"parent_index", "child_index"}, map[string]interface{}{
				"parent_index": nonNegativeIntegerSchema("Parent PlanNode index."),
				"child_index":  nonNegativeIntegerSchema("Child PlanNode index."),
				"type":         stringSchema("PlanNode child-link type."),
				"variable":     stringSchema("PlanNode child-link variable."),
			}),
			"optimizer": objectSchema([]interface{}{"requested", "effective"}, map[string]interface{}{
				"requested": objectSchema([]interface{}{"version", "statistics_package"}, map[string]interface{}{
					"version":            sentinelOrPatternSchema("not_pinned", `^[0-9A-Za-z._:+/-]+$`, "Requested optimizer version passed to AnalyzeQuery, or not_pinned."),
					"statistics_package": sentinelOrPatternSchema("not_pinned", `^[0-9A-Za-z._:+/-]+$`, "Requested optimizer statistics package passed to AnalyzeQuery, or not_pinned."),
				}),
				"effective": objectSchema([]interface{}{"version", "statistics_package"}, map[string]interface{}{
					"version":            sentinelOrPatternSchema("not_recorded", `^[0-9A-Za-z._:+/-]+$`, "Effective optimizer version observed from the backend, or not_recorded when unavailable."),
					"statistics_package": sentinelOrPatternSchema("not_recorded", `^[0-9A-Za-z._:+/-]+$`, "Effective optimizer statistics package observed from the backend, or not_recorded when unavailable."),
				}),
			}),
			"contract_summary": objectSchema([]interface{}{"status", "contracts", "passed", "failed", "not_evaluated", "environment_warnings"}, map[string]interface{}{
				"status":               enumSchema([]interface{}{planContractStatusPass, planContractStatusFail}, "Overall contract evaluation status."),
				"contracts":            nonNegativeIntegerSchema("Number of configured contracts."),
				"passed":               nonNegativeIntegerSchema("Number of contracts that passed."),
				"failed":               nonNegativeIntegerSchema("Number of contracts that failed."),
				"not_evaluated":        nonNegativeIntegerSchema("Number of contracts that could not be evaluated because the target plan was unavailable."),
				"environment_warnings": stringListSchema("Environment warnings that affect contract interpretation."),
			}),
			"contract_evaluation":  planReportContractEvaluationJSONSchema(),
			"contract_stability":   planReportContractStabilityJSONSchema(),
			"contract_rule_result": planReportContractRuleResultJSONSchema(),
			"remediation": objectSchema([]interface{}{"kind", "auto_fix", "message"}, map[string]interface{}{
				"kind":       stringSchema("Suggestion kind."),
				"applies_to": enumSchema([]interface{}{"config", "ddl", "sql", "contract"}, "Surface the suggestion applies to."),
				"confidence": enumSchema([]interface{}{"low", "medium", "high"}, "Suggestion confidence."),
				"auto_fix":   constBoolSchema(false, "Remediation is advisory in v1alpha and is never applied automatically."),
				"message":    stringSchema("Human-readable suggestion text."),
			}),
			"diagnostic": objectSchema([]interface{}{"id", "message"}, map[string]interface{}{
				"id":      stringSchema("Diagnostic ID."),
				"message": stringSchema("Human-readable diagnostic text."),
			}),
		},
	}
}

func planReportBackendIdentityJSONSchema() map[string]interface{} {
	schema := objectSchema([]interface{}{"kind", "version", "image_digest", "source"}, map[string]interface{}{
		"kind":         enumSchema([]interface{}{"omni"}, "Runtime backend kind."),
		"version":      sentinelOrPatternSchema("not_recorded", `^[0-9A-Za-z._:+/-]+$`, "Backend version, or not_recorded when unavailable."),
		"image_digest": sentinelOrPatternSchema("not_recorded", `^sha256:[a-f0-9]{64}$`, "Backend image digest, or not_recorded when unavailable."),
		"source": enumSchema(
			[]interface{}{"spanemuboost", "manual", "not_recorded"},
			"Backend identity provenance. manual means the caller supplied the value as an assertion, not observed evidence.",
		),
	})
	schema["allOf"] = []interface{}{
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"source": map[string]interface{}{"const": "manual"}},
				"required":   []interface{}{"source"},
			},
			"then": map[string]interface{}{
				"not": map[string]interface{}{
					"properties": map[string]interface{}{
						"version":      map[string]interface{}{"const": "not_recorded"},
						"image_digest": map[string]interface{}{"const": "not_recorded"},
					},
					"required": []interface{}{"version", "image_digest"},
				},
			},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"source": map[string]interface{}{"const": "not_recorded"}},
				"required":   []interface{}{"source"},
			},
			"then": map[string]interface{}{
				"properties": map[string]interface{}{
					"version":      map[string]interface{}{"const": "not_recorded"},
					"image_digest": map[string]interface{}{"const": "not_recorded"},
				},
			},
		},
	}
	return schema
}

func planReportQueryJSONSchema() map[string]interface{} {
	schema := objectSchema([]interface{}{"target_id", "name", "source", "kind", "status"}, map[string]interface{}{
		"target_id":               targetIDSchema("Canonical target ID, for example query/ListSingers or query/ExternalQuery#inner."),
		"name":                    identifierSchema("Configured query name."),
		"source":                  stringSchema("Spanner catalog used for analysis."),
		"scope":                   planReportScopeSchema("Plan-report target scope."),
		"kind":                    enumSchema([]interface{}{"sql", "table", "index", "external_query"}, "Configured query kind."),
		"status":                  enumSchema([]interface{}{"ok", "skipped", "error"}, "Per-query analysis status. Status ok means plan fields are valid. Status error or skipped means only the input identity fields and error field are reliable; plan normalization fields must not be interpreted as a successful plan."),
		"sql":                     stringSchema("Analyzed SQL text."),
		"sql_sha256":              sha256Schema("SHA-256 of analyzed SQL bytes."),
		"ddl_sha256":              sha256Schema("SHA-256 of resolved catalog DDL bytes."),
		"operator_tree_sha256":    sha256Schema("SHA-256 of normalized operator tree digest input."),
		"operator_families":       concreteOperatorFamilyListSchema("Concrete normalized operator families observed in normalized_operators[].family. Derived umbrella families such as explicit_sort are not listed here."),
		"operator_family_counts":  completeOperatorFamilyIntegerMapSchema("Normalized operator family counts observed in the query plan. Contains every operator_family enum value, including derived umbrella families; absent families are represented by 0."),
		"normalized_operators":    arraySchemaAllowEmpty(map[string]interface{}{"$ref": "#/$defs/operator"}),
		"operator_edges":          arraySchemaAllowEmpty(map[string]interface{}{"$ref": "#/$defs/operator_edge"}),
		"plan":                    stringSchema("Rendered human-readable query plan."),
		"error":                   stringSchema("Target acquisition error or skip message."),
		"optimizer_not_pinned":    boolSchema("Whether this query used unpinned optimizer settings."),
		"plan_environment_notes":  stringListSchema("Review notes about the plan acquisition environment."),
		"classification_warnings": arraySchemaAllowEmpty(map[string]interface{}{"$ref": "#/$defs/diagnostic"}),
	})
	schema["allOf"] = []interface{}{
		planReportQueryStatusRequiredClause("ok", []interface{}{"sql", "sql_sha256", "ddl_sha256", "operator_tree_sha256", "operator_families", "operator_family_counts", "normalized_operators", "operator_edges", "plan"}, nil),
		planReportQueryStatusRequiredClause("error", []interface{}{"error"}, planReportQuerySuccessOnlyFields()),
		planReportQueryStatusRequiredClause("skipped", []interface{}{"error"}, planReportQuerySuccessOnlyFields()),
	}
	return schema
}

func planReportQueryStatusRequiredClause(status string, required []interface{}, forbidden []string) map[string]interface{} {
	then := map[string]interface{}{"required": required}
	if len(forbidden) > 0 {
		then["not"] = anyRequired(forbidden...)
	}
	return map[string]interface{}{
		"if": map[string]interface{}{
			"properties": map[string]interface{}{
				"status": map[string]interface{}{"const": status},
			},
			"required": []interface{}{"status"},
		},
		"then": then,
	}
}

func planReportQuerySuccessOnlyFields() []string {
	return []string{
		"operator_tree_sha256",
		"operator_families",
		"operator_family_counts",
		"normalized_operators",
		"operator_edges",
		"plan",
		"classification_warnings",
	}
}

func planReportContractStabilityJSONSchema() map[string]interface{} {
	schema := objectSchema([]interface{}{"tier", "reasons", "check_recommended", "replayable_from_report"}, map[string]interface{}{
		"tier":                   enumSchema([]interface{}{planContractStabilityNormalized, planContractStabilityRawPlan}, "Contract stability tier."),
		"reasons":                nonEmptyStringListSchema("Why this stability tier was assigned."),
		"check_recommended":      boolSchema("Whether this contract tier is recommended for --check without additional review."),
		"replayable_from_report": boolSchema("Whether the contract can be re-evaluated from the serialized YAML/JSON report alone."),
	})
	schema["allOf"] = []interface{}{
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"tier": map[string]interface{}{"const": planContractStabilityRawPlan}},
				"required":   []interface{}{"tier"},
			},
			"then": map[string]interface{}{
				"properties": map[string]interface{}{
					"check_recommended":      map[string]interface{}{"const": false},
					"replayable_from_report": map[string]interface{}{"const": false},
				},
			},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"tier": map[string]interface{}{"const": planContractStabilityNormalized}},
				"required":   []interface{}{"tier"},
			},
			"then": map[string]interface{}{
				"properties": map[string]interface{}{
					"replayable_from_report": map[string]interface{}{"const": true},
				},
			},
		},
	}
	return schema
}

func planReportContractEvaluationJSONSchema() map[string]interface{} {
	schema := objectSchema([]interface{}{"name", "target_id", "status", "stability"}, map[string]interface{}{
		"name":      identifierSchema("Contract name."),
		"query":     identifierSchema("Target query name."),
		"target_id": targetIDSchema("Canonical target ID resolved for the contract."),
		"scope":     planReportScopeSchema("Resolved target scope."),
		"status":    enumSchema([]interface{}{planContractStatusPass, planContractStatusFail, planContractStatusNotEvaluated}, "Contract evaluation status."),
		"reason":    enumSchema([]interface{}{planContractReasonTargetNotFound, planContractReasonTargetError}, "Reason when status is not_evaluated."),
		"error":     stringSchema("Target availability error when status is not_evaluated because target acquisition failed."),
		"stability": map[string]interface{}{"$ref": "#/$defs/contract_stability"},
		"results":   arraySchema(map[string]interface{}{"$ref": "#/$defs/contract_rule_result"}),
	})
	schema["allOf"] = []interface{}{
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"status": map[string]interface{}{"const": planContractStatusPass}},
				"required":   []interface{}{"status"},
			},
			"then": map[string]interface{}{
				"required": []interface{}{"query", "scope", "results"},
				"not":      anyRequired("reason", "error"),
			},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"status": map[string]interface{}{"const": planContractStatusFail}},
				"required":   []interface{}{"status"},
			},
			"then": map[string]interface{}{
				"required": []interface{}{"query", "scope", "results"},
				"not":      anyRequired("reason", "error"),
			},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"status": map[string]interface{}{"const": planContractStatusNotEvaluated}},
				"required":   []interface{}{"status"},
			},
			"then": map[string]interface{}{
				"required": []interface{}{"reason"},
				"not":      anyRequired("results"),
			},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"reason": map[string]interface{}{"const": planContractReasonTargetError}},
				"required":   []interface{}{"reason"},
			},
			"then": map[string]interface{}{"required": []interface{}{"error"}},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"reason": map[string]interface{}{"const": planContractReasonTargetNotFound}},
				"required":   []interface{}{"reason"},
			},
			"then": map[string]interface{}{"not": anyRequired("error")},
		},
	}
	return schema
}

func planReportContractRuleResultJSONSchema() map[string]interface{} {
	schema := objectSchema([]interface{}{"rule", "source", "status"}, map[string]interface{}{
		"rule":                     enumSchema([]interface{}{"forbid_operator_family", "forbid_blocking_operator_under_limit", "forbid_full_scan", "forbid_full_scan_without_timestamp_condition", "require_timestamp_condition", "cel"}, "Rule kind."),
		"source":                   patternStringSchema(planContractRuleResultSourcePattern(), "Original contract source that produced this result, such as use/no_hash_join, forbid[0], or cel."),
		"predefined":               enumSchema(planContractPredefinedValues(), "Predefined contract name when this result was expanded from use."),
		"expression":               stringSchema("CEL expression for cel rules."),
		"operator_family":          map[string]interface{}{"$ref": "#/$defs/operator_family"},
		"status":                   enumSchema([]interface{}{planContractStatusPass, planContractStatusFail}, "Rule evaluation status."),
		"failure_kind":             enumSchema([]interface{}{planContractFailureKindViolation, planContractFailureKindClassificationUnknown}, "Reason class when status is fail."),
		"diagnostic_id":            patternStringSchema(`^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$`, "Diagnostic ID associated with a non-violation failure kind."),
		"observed_count":           nonNegativeIntegerSchema("Observed normalized operator count."),
		"max_count":                nonNegativeIntegerSchema("Allowed normalized operator count."),
		"matched_operator_indexes": integerListSchema("PlanNode indexes from normalized_operators[].index that matched this operator-family rule."),
		"remediation":              arraySchemaAllowEmpty(map[string]interface{}{"$ref": "#/$defs/remediation"}),
	})
	schema["allOf"] = []interface{}{
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"rule": map[string]interface{}{"const": "forbid_operator_family"}},
				"required":   []interface{}{"rule"},
			},
			"then": map[string]interface{}{
				"properties": map[string]interface{}{
					"source": patternStringSchema(planContractRuleResultForbidSourcePattern(), "Predefined or direct forbid contract rule source."),
				},
				"required": []interface{}{"operator_family", "observed_count", "max_count", "matched_operator_indexes"},
				"not":      anyRequired("expression"),
			},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"rule": map[string]interface{}{"const": "cel"}},
				"required":   []interface{}{"rule"},
			},
			"then": map[string]interface{}{
				"properties": map[string]interface{}{
					"source": map[string]interface{}{"const": "cel"},
				},
				"required": []interface{}{"expression"},
				"not":      anyRequired("predefined", "operator_family", "observed_count", "max_count", "matched_operator_indexes"),
			},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"rule": map[string]interface{}{"const": "forbid_blocking_operator_under_limit"}},
				"required":   []interface{}{"rule"},
			},
			"then": map[string]interface{}{
				"properties": map[string]interface{}{
					"source": map[string]interface{}{"const": "use/no_blocking_operator_under_limit"},
				},
				"required": []interface{}{"predefined", "observed_count", "max_count", "matched_operator_indexes"},
				"not":      anyRequired("expression", "operator_family", "diagnostic_id"),
			},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"rule": map[string]interface{}{"const": "forbid_full_scan"}},
				"required":   []interface{}{"rule"},
			},
			"then": map[string]interface{}{
				"properties": map[string]interface{}{
					"source": map[string]interface{}{"const": "use/no_full_scan"},
				},
				"required": []interface{}{"predefined", "observed_count", "max_count", "matched_operator_indexes"},
				"not":      anyRequired("expression", "operator_family", "diagnostic_id"),
			},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"rule": map[string]interface{}{"const": "forbid_full_scan_without_timestamp_condition"}},
				"required":   []interface{}{"rule"},
			},
			"then": map[string]interface{}{
				"properties": map[string]interface{}{
					"source": map[string]interface{}{"const": "use/no_full_scan_without_timestamp_condition"},
				},
				"required": []interface{}{"predefined", "observed_count", "max_count", "matched_operator_indexes"},
				"not":      anyRequired("expression", "operator_family", "diagnostic_id"),
			},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"rule": map[string]interface{}{"const": "require_timestamp_condition"}},
				"required":   []interface{}{"rule"},
			},
			"then": map[string]interface{}{
				"properties": map[string]interface{}{
					"source": map[string]interface{}{"const": "use/require_timestamp_condition"},
				},
				"required": []interface{}{"predefined", "observed_count", "max_count", "matched_operator_indexes"},
				"not":      anyRequired("expression", "operator_family", "diagnostic_id"),
			},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{
					"rule":   map[string]interface{}{"const": "cel"},
					"status": map[string]interface{}{"const": planContractStatusFail},
				},
				"required": []interface{}{"rule", "status"},
			},
			"then": map[string]interface{}{
				"properties": map[string]interface{}{
					"failure_kind": map[string]interface{}{"const": planContractFailureKindViolation},
				},
				"not": anyRequired("diagnostic_id"),
			},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"status": map[string]interface{}{"const": planContractStatusFail}},
				"required":   []interface{}{"status"},
			},
			"then": map[string]interface{}{"required": []interface{}{"failure_kind"}},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"status": map[string]interface{}{"const": planContractStatusPass}},
				"required":   []interface{}{"status"},
			},
			"then": map[string]interface{}{"not": anyRequired("failure_kind", "diagnostic_id", "remediation")},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"failure_kind": map[string]interface{}{"const": planContractFailureKindClassificationUnknown}},
				"required":   []interface{}{"failure_kind"},
			},
			"then": map[string]interface{}{"required": []interface{}{"diagnostic_id"}},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"failure_kind": map[string]interface{}{"const": planContractFailureKindViolation}},
				"required":   []interface{}{"failure_kind"},
			},
			"then": map[string]interface{}{"not": anyRequired("diagnostic_id")},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"source": patternStringSchema(`^use/`, "Predefined contract expansion source.")},
				"required":   []interface{}{"source"},
			},
			"then": map[string]interface{}{"required": []interface{}{"predefined"}},
		},
		map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{"source": patternStringSchema(`^(forbid\[[0-9]+\]|cel)$`, "Direct contract rule source.")},
				"required":   []interface{}{"source"},
			},
			"then": map[string]interface{}{"not": anyRequired("predefined")},
		},
	}
	return schema
}

func v1AlphaConfigJSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  "https://github.com/apstndb/go-googlesql-spanner-poc/schemas/spanner-query-gen.v1alpha.schema.json",
		"title":                "spanner-query-gen v1alpha config",
		"type":                 "object",
		"additionalProperties": false,
		"required":             []interface{}{"version", "go", "catalogs"},
		"anyOf": []interface{}{
			map[string]interface{}{"required": []interface{}{"queries"}},
			map[string]interface{}{"required": []interface{}{"writes"}},
		},
		"properties": map[string]interface{}{
			"version": map[string]interface{}{
				"const":       querygen.QueryCodegenConfigVersionV1Alpha,
				"description": "Mutable pre-v1 preview config version.",
			},
			"go":   map[string]interface{}{"$ref": "#/$defs/go"},
			"emit": map[string]interface{}{"$ref": "#/$defs/emit"},
			"catalogs": map[string]interface{}{
				"type":        "array",
				"minItems":    1,
				"uniqueItems": true,
				"items":       map[string]interface{}{"$ref": "#/$defs/catalog"},
			},
			"queries": map[string]interface{}{
				"type":        "array",
				"minItems":    1,
				"uniqueItems": true,
				"items":       map[string]interface{}{"$ref": "#/$defs/query"},
			},
			"writes": map[string]interface{}{
				"type":        "array",
				"minItems":    1,
				"uniqueItems": true,
				"items":       map[string]interface{}{"$ref": "#/$defs/write"},
			},
			"rules": map[string]interface{}{"$ref": "#/$defs/rules"},
		},
		"$defs": map[string]interface{}{
			"identifier": identifierSchema("v1alpha public identifier."),
			"go": objectSchema([]interface{}{"package"}, map[string]interface{}{
				"package": stringSchema("Generated Go package name."),
				"out":     stringSchema("Generated Go output path, resolved relative to the config file."),
			}),
			"emit": objectSchema(nil, map[string]interface{}{
				"spanner": objectSchema(nil, map[string]interface{}{
					"mutations": boolSchema("Emit Cloud Spanner mutation helpers for compatible writes."),
					"dml":       boolSchema("Emit Cloud Spanner DML statement helpers for compatible writes."),
				}),
				"bigquery": objectSchema(nil, map[string]interface{}{
					"row_loader":   boolSchema("Emit BigQuery ValueLoader implementations."),
					"table_schema": boolSchema("Emit BigQuery TableSchema metadata helpers."),
				}),
			}),
			"catalog": catalogSchema(),
			"external_query_connection": objectSchema([]interface{}{"name", "id", "spanner_catalog"}, map[string]interface{}{
				"name":            map[string]interface{}{"$ref": "#/$defs/identifier"},
				"id":              stringSchema("BigQuery connection ID, for example project.location.connection."),
				"spanner_catalog": map[string]interface{}{"$ref": "#/$defs/identifier"},
			}),
			"spanner_external_dataset": objectSchema([]interface{}{"name", "dataset", "spanner_catalog"}, map[string]interface{}{
				"name":                 map[string]interface{}{"$ref": "#/$defs/identifier"},
				"dataset":              stringSchema("BigQuery dataset name exposed by the external dataset."),
				"spanner_catalog":      map[string]interface{}{"$ref": "#/$defs/identifier"},
				"spanner_database_uri": stringSchema("Optional Spanner external source URI; database roles are inferred into the plan."),
				"location":             stringSchema("Optional BigQuery dataset location."),
				"access": objectSchema(nil, map[string]interface{}{
					"cloud_resource_connection_id": stringSchema("Cloud resource connection ID used by the external dataset."),
					"verification_evidence":        map[string]interface{}{"$ref": "#/$defs/verification_evidence"},
				}),
				"projection": objectSchema(nil, map[string]interface{}{
					"unsupported_columns": enumSchema([]interface{}{"reject", "omit"}, "How unsupported Spanner columns are handled."),
					"named_schema_tables": enumSchema([]interface{}{"reject", "warn_and_omit"}, "How named-schema Spanner tables are handled."),
				}),
			}),
			"verification_evidence": objectSchema([]interface{}{"status", "verifier", "checked_at"}, map[string]interface{}{
				"status":     enumSchema([]interface{}{"verified", "mismatch", "failed"}, "External verification status."),
				"verifier":   stringSchema("Verifier name, such as terraform-plan."),
				"checked_at": dateTimeStringSchema("RFC3339 timestamp for real external evidence."),
			}),
			"query": querySchema(),
			"query_param": objectSchema([]interface{}{"name", "type"}, map[string]interface{}{
				"name":  stringSchema("GoogleSQL parameter name without the leading @."),
				"type":  stringSchema("GoogleSQL parameter type used when analyzer inference needs explicit input."),
				"scope": enumSchema([]interface{}{"inner", "outer"}, "Optional external_query parameter scope. Omit for regular queries."),
			}),
			"query_result": objectSchema([]interface{}{"struct"}, map[string]interface{}{
				"cardinality": enumSchema([]interface{}{"one", "maybe_one", "many"}, "Row-returning query result cardinality. Defaults to many."),
				"struct":      stringSchema("Generated or shared result struct name."),
				"required": objectSchema(nil, map[string]interface{}{
					"policy": enumSchema([]interface{}{"override", "strict"}, "Requiredness override policy."),
					"fields": stringArraySchema("Result fields required by user override."),
				}),
			}),
			"write": writeSchema(),
			"rules": objectSchema(nil, map[string]interface{}{
				"suppressions": arraySchema(map[string]interface{}{"$ref": "#/$defs/rule_suppression"}),
			}),
			"rule_suppression": objectSchema([]interface{}{"scope", "rule", "reason"}, map[string]interface{}{
				"scope":   patternStringSchema(`^(query|write|catalog-binding)/.+`, "Suppression scope, such as query/Name or catalog-binding/catalog.binding."),
				"rule":    stringSchema("Diagnostic rule ID."),
				"reason":  stringSchema("Required reviewable suppression reason."),
				"owner":   stringSchema("Optional suppression owner."),
				"expires": dateStringSchema("Optional YYYY-MM-DD suppression expiry date."),
			}),
		},
	}
}

func catalogSchema() map[string]interface{} {
	schema := objectSchema([]interface{}{"name", "kind"}, map[string]interface{}{
		"name":              map[string]interface{}{"$ref": "#/$defs/identifier"},
		"kind":              enumSchema([]interface{}{"spanner", "bigquery"}, "Catalog resource kind."),
		"dialect":           enumSchema([]interface{}{"googlesql", "postgresql"}, "Optional non-default SQL dialect spelling."),
		"project":           stringSchema("Default BigQuery project for this catalog."),
		"ddl":               stringSchema("DDL file path, resolved relative to the config file."),
		"proto_descriptors": stringArraySchema("Protocol Buffers descriptor set paths for Spanner proto bundle analysis."),
		"bindings": objectSchema(nil, map[string]interface{}{
			"external_query_connections": arraySchema(map[string]interface{}{"$ref": "#/$defs/external_query_connection"}),
			"spanner_external_datasets":  arraySchema(map[string]interface{}{"$ref": "#/$defs/spanner_external_dataset"}),
		}),
	})
	schema["allOf"] = []interface{}{map[string]interface{}{
		"oneOf": []interface{}{
			catalogKindClause("spanner", nil, []string{"project", "bindings"}),
			catalogKindClause("bigquery", nil, []string{"proto_descriptors"}),
		},
	}}
	return schema
}

func catalogKindClause(kind string, required []interface{}, forbidden []string) map[string]interface{} {
	clause := map[string]interface{}{
		"properties": map[string]interface{}{
			"kind": map[string]interface{}{"const": kind},
		},
	}
	if len(required) > 0 {
		clause["required"] = required
	}
	if len(forbidden) > 0 {
		clause["not"] = anyRequired(forbidden...)
	}
	return clause
}

func querySchema() map[string]interface{} {
	schema := objectSchema([]interface{}{"name", "catalog", "kind", "result"}, map[string]interface{}{
		"name":       map[string]interface{}{"$ref": "#/$defs/identifier"},
		"catalog":    map[string]interface{}{"$ref": "#/$defs/identifier"},
		"kind":       enumSchema([]interface{}{"sql", "table", "index", "external_query"}, "Query declaration kind."),
		"sql":        stringSchema("SQL text for kind: sql."),
		"table":      stringSchema("Table shorthand for kind: table."),
		"index":      stringSchema("Spanner index shorthand for kind: index."),
		"binding":    map[string]interface{}{"$ref": "#/$defs/identifier"},
		"inner_sql":  stringSchema("Spanner SQL analyzed inside EXTERNAL_QUERY."),
		"outer_sql":  stringSchema("Optional BigQuery wrapper SQL containing exactly one __external__ placeholder."),
		"key_prefix": stringArraySchema("Index key columns used as query parameters from the leading key prefix."),
		"order_by":   enumSchema([]interface{}{"key", "none"}, "Ordering for generated table/index SQL. Omit for deterministic key ordering; use none to avoid ORDER BY."),
		"result":     map[string]interface{}{"$ref": "#/$defs/query_result"},
		"params":     arraySchema(map[string]interface{}{"$ref": "#/$defs/query_param"}),
	})
	schema["allOf"] = []interface{}{map[string]interface{}{
		"oneOf": []interface{}{
			queryKindClause("sql", []interface{}{"sql"}, []string{"table", "index", "binding", "inner_sql", "outer_sql", "key_prefix", "order_by"}, regularQueryParamsSchema()),
			queryKindClause("table", []interface{}{"table"}, []string{"sql", "index", "binding", "inner_sql", "outer_sql", "key_prefix"}, regularQueryParamsSchema()),
			queryKindClause("index", []interface{}{"index"}, []string{"sql", "table", "binding", "inner_sql", "outer_sql"}, regularQueryParamsSchema()),
			queryKindClause("external_query", []interface{}{"binding", "inner_sql"}, []string{"sql", "table", "index", "key_prefix", "order_by"}),
		},
	}}
	return schema
}

func writeSchema() map[string]interface{} {
	schema := objectSchema([]interface{}{"name", "catalog", "table", "operation", "input"}, map[string]interface{}{
		"name":      map[string]interface{}{"$ref": "#/$defs/identifier"},
		"catalog":   map[string]interface{}{"$ref": "#/$defs/identifier"},
		"table":     stringSchema("Spanner table name."),
		"operation": enumSchema([]interface{}{"insert", "update", "upsert", "replace", "delete"}, "Write operation."),
		"input":     stringSchema("Input struct name."),
		"key":       stringArraySchema("Table primary key columns used by update/delete/upsert helpers; omit to infer from DDL."),
		"insert": objectSchema(nil, map[string]interface{}{
			"columns": stringArraySchema("Insert branch columns."),
		}),
		"update": objectSchema(nil, map[string]interface{}{
			"columns":             stringArraySchema("Update mask columns."),
			"all_non_key_columns": constBoolSchema(true, "Explicit opt-in to update every non-key column."),
		}),
	})
	schema["allOf"] = []interface{}{map[string]interface{}{
		"oneOf": []interface{}{
			writeOperationClause("insert", []interface{}{"insert"}, map[string][]interface{}{"insert": {"columns"}}, []string{"update"}),
			writeOperationClause("update", []interface{}{"update"}, nil, []string{"insert"}, map[string]interface{}{
				"update": map[string]interface{}{
					"oneOf": []interface{}{
						map[string]interface{}{"required": []interface{}{"columns"}},
						map[string]interface{}{
							"required": []interface{}{"all_non_key_columns"},
							"properties": map[string]interface{}{
								"all_non_key_columns": map[string]interface{}{"const": true},
							},
						},
					},
				},
			}),
			writeOperationClause("upsert", []interface{}{"insert", "update"}, map[string][]interface{}{"insert": {"columns"}, "update": {"columns"}}, nil, map[string]interface{}{
				"update": map[string]interface{}{
					"not": map[string]interface{}{"required": []interface{}{"all_non_key_columns"}},
				},
			}),
			writeOperationClause("replace", []interface{}{"insert"}, map[string][]interface{}{"insert": {"columns"}}, []string{"update"}),
			writeOperationClause("delete", nil, nil, []string{"insert", "update"}),
		},
	}}
	return schema
}

func regularQueryParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"params": arraySchema(map[string]interface{}{
			"allOf": []interface{}{
				map[string]interface{}{"$ref": "#/$defs/query_param"},
				map[string]interface{}{"not": map[string]interface{}{"required": []interface{}{"scope"}}},
			},
		}),
	}
}

func queryKindClause(kind string, required []interface{}, forbidden []string, extraProperties ...map[string]interface{}) map[string]interface{} {
	properties := map[string]interface{}{
		"kind": map[string]interface{}{"const": kind},
	}
	for _, extra := range extraProperties {
		for property, value := range extra {
			properties[property] = value
		}
	}
	clause := map[string]interface{}{"properties": properties}
	if len(required) > 0 {
		clause["required"] = required
	}
	if len(forbidden) > 0 {
		clause["not"] = anyRequired(forbidden...)
	}
	return clause
}

func writeOperationClause(operation string, required []interface{}, nestedRequired map[string][]interface{}, forbidden []string, extraProperties ...map[string]interface{}) map[string]interface{} {
	properties := map[string]interface{}{
		"operation": map[string]interface{}{"const": operation},
	}
	for property, nested := range nestedRequired {
		properties[property] = map[string]interface{}{"required": nested}
	}
	for _, extra := range extraProperties {
		for property, value := range extra {
			if existing, ok := properties[property].(map[string]interface{}); ok {
				if extraValue, ok := value.(map[string]interface{}); ok {
					for key, nestedValue := range extraValue {
						existing[key] = nestedValue
					}
					continue
				}
			}
			properties[property] = value
		}
	}
	clause := map[string]interface{}{"properties": properties}
	if len(required) > 0 {
		clause["required"] = required
	}
	if len(forbidden) > 0 {
		clause["not"] = anyRequired(forbidden...)
	}
	return clause
}

func anyRequired(fields ...string) map[string]interface{} {
	clauses := make([]interface{}, 0, len(fields))
	for _, field := range fields {
		clauses = append(clauses, map[string]interface{}{"required": []interface{}{field}})
	}
	return map[string]interface{}{"anyOf": clauses}
}

func objectSchema(required []interface{}, properties map[string]interface{}) map[string]interface{} {
	schema := map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func arraySchema(item map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"minItems":    1,
		"uniqueItems": true,
		"items":       item,
	}
}

func arraySchemaAllowEmpty(item map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"uniqueItems": true,
		"items":       item,
	}
}

func stringArraySchema(description string) map[string]interface{} {
	schema := arraySchema(map[string]interface{}{"type": "string", "minLength": 1})
	schema["description"] = description
	schema["minItems"] = 1
	schema["uniqueItems"] = true
	return schema
}

func stringListSchema(description string) map[string]interface{} {
	schema := arraySchemaAllowEmpty(map[string]interface{}{"type": "string", "minLength": 1})
	schema["description"] = description
	return schema
}

func nonEmptyStringListSchema(description string) map[string]interface{} {
	schema := stringListSchema(description)
	schema["minItems"] = 1
	return schema
}

func integerListSchema(description string) map[string]interface{} {
	schema := arraySchemaAllowEmpty(nonNegativeIntegerSchema("PlanNode index."))
	schema["description"] = description
	return schema
}

func celInputDefaultAppliesToSchema() map[string]interface{} {
	schema := arraySchema(enumSchema([]interface{}{"operators[]", "operator_edges[]"}, "Normalized report arrays whose optional fields use these CEL defaults."))
	schema["minItems"] = 2
	schema["maxItems"] = 2
	schema["prefixItems"] = []interface{}{
		map[string]interface{}{"const": "operators[]"},
		map[string]interface{}{"const": "operator_edges[]"},
	}
	return schema
}

func concreteOperatorFamilyListSchema(description string) map[string]interface{} {
	schema := arraySchemaAllowEmpty(map[string]interface{}{"$ref": "#/$defs/concrete_operator_family"})
	schema["description"] = description
	return schema
}

func completeOperatorFamilyIntegerMapSchema(description string) map[string]interface{} {
	properties := make(map[string]interface{}, len(planReportKnownOperatorFamilies()))
	required := make([]interface{}, 0, len(planReportKnownOperatorFamilies()))
	for _, family := range planReportKnownOperatorFamilies() {
		properties[family] = nonNegativeIntegerSchema("Count for normalized operator family " + family + ".")
		required = append(required, family)
	}
	schema := objectSchema(required, properties)
	schema["description"] = description
	return schema
}

func interfaceSlice(values []string) []interface{} {
	out := make([]interface{}, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func stringSchema(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"minLength":   1,
		"description": description,
	}
}

func identifierSchema(description string) map[string]interface{} {
	return patternStringSchema(v1AlphaIdentifierPattern, description)
}

func targetIDSchema(description string) map[string]interface{} {
	return patternStringSchema(planContractTargetIDPattern, description)
}

func planReportScopeSchema(description string) map[string]interface{} {
	return enumSchema([]interface{}{"query", "external_query.inner"}, description)
}

func patternStringSchema(pattern, description string) map[string]interface{} {
	schema := stringSchema(description)
	schema["pattern"] = pattern
	return schema
}

func dateStringSchema(description string) map[string]interface{} {
	schema := stringSchema(description)
	schema["format"] = "date"
	schema["pattern"] = `^\d{4}-\d{2}-\d{2}$`
	return schema
}

func dateTimeStringSchema(description string) map[string]interface{} {
	schema := stringSchema(description)
	schema["format"] = "date-time"
	return schema
}

func boolSchema(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "boolean",
		"description": description,
	}
}

func nonNegativeIntegerSchema(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"minimum":     0,
		"description": description,
	}
}

func nonNegativeIntegerSchemaWithDefault(description string, defaultValue int) map[string]interface{} {
	schema := nonNegativeIntegerSchema(description)
	schema["default"] = defaultValue
	return schema
}

func planContractPredefinedValues() []interface{} {
	return interfaceSlice(planContractPredefinedNames())
}

func planContractRuleResultSourcePattern() string {
	return `^(use/(` + strings.Join(planContractPredefinedNames(), "|") + `)|forbid\[[0-9]+\]|cel)$`
}

func planContractRuleResultForbidSourcePattern() string {
	return `^(use/(` + strings.Join(planContractForbidOperatorFamilyPredefinedNames(), "|") + `)|forbid\[[0-9]+\])$`
}

func planContractForbidOperatorFamilyPredefinedNames() []string {
	var names []string
	for _, name := range planContractPredefinedNames() {
		if name == "no_blocking_operator_under_limit" || name == "no_full_scan" || name == "no_full_scan_without_timestamp_condition" || name == "require_timestamp_condition" {
			continue
		}
		names = append(names, name)
	}
	return names
}

func sha256Schema(description string) map[string]interface{} {
	return patternStringSchema(`^[a-f0-9]{64}$`, description)
}

func sentinelOrPatternSchema(sentinel, pattern, description string) map[string]interface{} {
	return map[string]interface{}{
		"anyOf": []interface{}{
			map[string]interface{}{"const": sentinel},
			patternStringSchema(pattern, description),
		},
		"description": description,
	}
}

func constBoolSchema(value bool, description string) map[string]interface{} {
	return map[string]interface{}{
		"const":       value,
		"description": description,
	}
}

func enumSchema(values []interface{}, description string) map[string]interface{} {
	return map[string]interface{}{
		"enum":        values,
		"description": description,
	}
}
