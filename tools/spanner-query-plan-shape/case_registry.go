package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type queryProvider func() []queryCase

type ddlProvider func() ([]string, error)

type builtInCaseSpec struct {
	Name                     string
	Description              string
	Queries                  queryProvider
	DDLs                     ddlProvider
	RequiresProtoDescriptors bool
}

type builtInCaseSummary struct {
	Name                     string `json:"name"`
	Description              string `json:"description"`
	QueryCount               int    `json:"query_count"`
	RequiresProtoDescriptors bool   `json:"requires_proto_descriptors,omitempty"`
}

var (
	defaultDDLProvider = rawDDLProvider(singersDDL, albumsDDL)
	docsDDLProvider    = parsedDDLProvider("docs-schema.sql", docsDDL)

	builtInCaseSpecs = []builtInCaseSpec{
		{Name: "all", Description: "two representative hash-join probes (not every built-in case)", Queries: staticQueryProvider([]queryCase{
			{Label: "PUSH_BROADCAST_HASH_JOIN", SQL: pushBroadcastSQL},
			{Label: "HASH_JOIN", SQL: hashSQL},
		}), DDLs: defaultDDLProvider},
		{Name: "docs", Description: "operator examples documented by the plan probe", Queries: staticQueryProvider(docsQueries), DDLs: docsDDLProvider},
		{Name: "optimizer_gaps", Description: "optimizer-version and hint gap probes", Queries: staticQueryProvider(optimizerGapQueries), DDLs: parsedDDLProvider("optimizer-gaps-schema.sql", optimizerGapsDDL)},
		{Name: "optimizer_unhinted_candidates", Description: "unhinted optimizer-version candidate probes", Queries: staticQueryProvider(optimizerUnhintedCandidateQueries), DDLs: parsedDDLProvider("optimizer-gaps-schema.sql", optimizerGapsDDL)},
		{Name: "join_elimination", Description: "join-elimination hypotheses and controls", Queries: staticQueryProvider(joinEliminationQueries), DDLs: parsedDDLProvider("join-elimination-schema.sql", joinEliminationDDL)},
		{Name: "planvocab_inference", Description: "plan-vocabulary inference hypotheses and controls", Queries: staticQueryProvider(planVocabInferenceQueries), DDLs: parsedDDLProvider("planvocab-inference-schema.sql", joinEliminationDDL)},
		{Name: "cte", Description: "common-table-expression probes", Queries: staticQueryProvider(cteQueries), DDLs: docsDDLProvider},
		{Name: "dml", Description: "DML and top-level CALL statement surfaces", Queries: staticQueryProvider(dmlQueries), DDLs: parsedDDLProvider("dml-schema.sql", dmlDDL)},
		{Name: "tvf", Description: "table-valued function and change-stream probes", Queries: staticQueryProvider(tvfQueries), DDLs: parsedDDLProvider("tvf-schema.sql", changeStreamTVFDDL)},
		{Name: "lock_hints", Description: "lock-hint probes", Queries: staticQueryProvider(lockHintQueries), DDLs: parsedDDLProvider("lock-hints-schema.sql", docsDDL)},
		{Name: "full_text_search", Description: "full-text search and facet probes", Queries: staticQueryProvider(fullTextSearchQueries), DDLs: parsedDDLProvider("full-text-search-schema.sql", fullTextSearchDDL)},
		{Name: "ngram_search", Description: "n-gram fuzzy and pattern-search probes", Queries: staticQueryProvider(ngramSearchQueries), DDLs: parsedDDLProvider("ngram-search-schema.sql", ngramSearchDDL)},
		{Name: "json_search", Description: "JSON search-index probes", Queries: staticQueryProvider(jsonSearchQueries), DDLs: parsedDDLProvider("json-search-schema.sql", jsonSearchDDL)},
		{Name: "vector_search", Description: "vector-index and exact-neighbor probes", Queries: staticQueryProvider(vectorSearchQueries), DDLs: rawDDLProvider(vectorSearchDDLs...)},
		{Name: "search_graph", Description: "combined search and vector plan graph probes", Queries: combinedQueryProvider(fullTextSearchQueries, vectorSearchQueries), DDLs: combinedDDLProvider(
			parsedDDLProvider("full-text-search-schema.sql", fullTextSearchDDL),
			rawDDLProvider(vectorSearchDDLs...),
		)},
		{Name: "ai_plan", Description: "AI function plan-capability probes", Queries: staticQueryProvider(aiPlanQueries), DDLs: docsDDLProvider},
		{Name: "statement_surface", Description: "schema-independent top-level statement boundaries", Queries: staticQueryProvider(statementSurfaceQueries), DDLs: noDDLProvider},
		{Name: "function_hint", Description: "function-level hint probes", Queries: staticQueryProvider(functionHintQueries), DDLs: docsDDLProvider},
		{Name: "hint_matrix", Description: "cross-surface hint matrix", Queries: staticQueryProvider(hintMatrixQueries), DDLs: docsDDLProvider},
		{Name: "statement_hint_query_matrix", Description: "statement-hint query matrix", Queries: staticQueryProvider(statementHintQueryMatrixQueries), DDLs: docsDDLProvider},
		{Name: "hint_position_audit", Description: "hint grammar-position audit", Queries: hintPositionAuditQueries, DDLs: docsDDLProvider},
		{Name: "hint_position_combinations", Description: "composed hint-position probes and controls", Queries: staticQueryProvider(hintPositionCombinationQueries), DDLs: docsDDLProvider},
		{Name: "set_operation_distinct", Description: "set-operation semantics, hints, and rewrites", Queries: staticQueryProvider(setOperationDistinctQueries), DDLs: parsedDDLProvider("set-operation-distinct-schema.sql", setOperationDistinctDDL)},
		{Name: "factorized_mode", Description: "factorized-mode eligibility and effect probes", Queries: staticQueryProvider(factorizedModeQueries), DDLs: docsDDLProvider},
		{Name: "gql_surface", Description: "broad GQL surface including IS_FIRST probes", Queries: staticQueryProvider(gqlSurfaceQueries), DDLs: docsDDLProvider},
		{Name: "gql_set_propagation", Description: "GQL set-operation column-propagation probes", Queries: staticQueryProvider(gqlSetPropagationQueries), DDLs: docsDDLProvider},
		{Name: "gql_hint_surface", Description: "GQL hint names, values, placements, and boundaries", Queries: staticQueryProvider(gqlHintSurfaceQueries), DDLs: docsDDLProvider},
		{Name: "google_sql_surface", Description: "GoogleSQL syntax and runtime-capability boundaries", Queries: staticQueryProvider(googleSQLSurfaceQueries), DDLs: docsDDLProvider},
		{Name: "google_sql_proto_surface", Description: "protocol-buffer expression surfaces", Queries: staticQueryProvider(googleSQLProtoSurfaceQueries), DDLs: parsedDDLProvider("google-sql-proto-surface-schema.sql", googleSQLProtoSurfaceDDL), RequiresProtoDescriptors: true},
		{Name: "rewriter_surface", Description: "GoogleSQL rewriter-backed syntax surfaces", Queries: staticQueryProvider(rewriterSurfaceQueries), DDLs: parsedDDLProvider("rewriter-surface-schema.sql", rewriterSurfaceDDL)},
		{Name: "condition_boundaries", Description: "seek, residual, split-range, and join conditions", Queries: staticQueryProvider(conditionBoundaryQueries), DDLs: parsedDDLProvider("condition-boundary-schema.sql", conditionBoundaryDDL)},
		{Name: "aggregate_functions", Description: "aggregate function plan-shape probes", Queries: staticQueryProvider(aggregateFunctionQueries), DDLs: docsDDLProvider},
		{Name: "join_matrix", Description: "join-method and build-side matrix", Queries: staticQueryProvider(joinMatrixQueries), DDLs: docsDDLProvider},
		{Name: "subquery_join_hint_matrix", Description: "subquery join-hint matrix", Queries: staticQueryProvider(subqueryJoinHintMatrixQueries), DDLs: docsDDLProvider},
		{Name: "push_broadcast_hash_join", Description: "single push-broadcast hash-join probe", Queries: staticQueryProvider([]queryCase{{Label: "PUSH_BROADCAST_HASH_JOIN", SQL: pushBroadcastSQL}}), DDLs: defaultDDLProvider},
		{Name: "hash_join", Description: "single hash-join probe", Queries: staticQueryProvider([]queryCase{{Label: "HASH_JOIN", SQL: hashSQL}}), DDLs: defaultDDLProvider},
	}
)

func lookupBuiltInCase(name string) (builtInCaseSpec, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, spec := range builtInCaseSpecs {
		if spec.Name == name {
			return spec, true
		}
	}
	return builtInCaseSpec{}, false
}

func builtInCaseNames() string {
	names := make([]string, 0, len(builtInCaseSpecs))
	for _, spec := range builtInCaseSpecs {
		names = append(names, spec.Name)
	}
	return strings.Join(names, ", ")
}

func printBuiltInCases(stdout io.Writer, outputFormat string) error {
	summaries := make([]builtInCaseSummary, 0, len(builtInCaseSpecs))
	for _, spec := range builtInCaseSpecs {
		summaries = append(summaries, builtInCaseSummary{
			Name:                     spec.Name,
			Description:              spec.Description,
			QueryCount:               len(spec.Queries()),
			RequiresProtoDescriptors: spec.RequiresProtoDescriptors,
		})
	}

	switch strings.ToLower(strings.TrimSpace(outputFormat)) {
	case "text":
		if _, err := fmt.Fprintln(stdout, "NAME\tQUERIES\tDESCRIPTION"); err != nil {
			return err
		}
		for _, summary := range summaries {
			if _, err := fmt.Fprintf(stdout, "%s\t%d\t%s\n", summary.Name, summary.QueryCount, summary.Description); err != nil {
				return err
			}
		}
		return nil
	case "json":
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(summaries)
	default:
		return fmt.Errorf("unsupported --list-cases-format %q; use text or json", outputFormat)
	}
}

func staticQueryProvider(queries []queryCase) queryProvider {
	return func() []queryCase {
		return append([]queryCase(nil), queries...)
	}
}

func combinedQueryProvider(groups ...[]queryCase) queryProvider {
	return func() []queryCase {
		var queries []queryCase
		for _, group := range groups {
			queries = append(queries, group...)
		}
		return queries
	}
}

func parsedDDLProvider(path, ddlSQL string) ddlProvider {
	return func() ([]string, error) {
		return parseBuiltInDDLs(path, ddlSQL)
	}
}

func rawDDLProvider(ddls ...string) ddlProvider {
	return func() ([]string, error) {
		return append([]string(nil), ddls...), nil
	}
}

func combinedDDLProvider(providers ...ddlProvider) ddlProvider {
	return func() ([]string, error) {
		var ddls []string
		for _, provider := range providers {
			group, err := provider()
			if err != nil {
				return nil, err
			}
			ddls = append(ddls, group...)
		}
		return ddls, nil
	}
}

func noDDLProvider() ([]string, error) {
	return nil, nil
}
