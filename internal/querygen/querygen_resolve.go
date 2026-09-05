package querygen

import (
	"fmt"
	"strings"
)

// resolvedCodegen is the common input to Go emission and plan projection.
// Query fields and write-only fields stay separate: only query results require
// query decoder helpers, even when a write shares a result DTO.
type resolvedCodegen struct {
	options           GoStructOptions
	schemas           map[string]QueryCodegenSchema
	queries           []resolvedCodegenQuery
	querySpecs        []resolvedQuerySpec
	structs           map[string][]goResultField
	writeSpecs        []resolvedWriteSpec
	writeStructFields map[string][]goResultField
}

type resolvedCodegenQuery struct {
	query    QueryCodegenQuery
	fields   []goResultField
	variants []QueryCodegenPlanQueryVariant
	spec     resolvedQuerySpec
}

// resolveQueryCodegen owns input validation, SQL analysis, DTO merging, write
// planning and package namespace validation for both public entry points.
// Audit-only catalog metadata and warnings remain in the plan projection.
func resolveQueryCodegen(config QueryCodegenConfig, baseDir string) (*resolvedCodegen, error) {
	if len(config.Queries) == 0 && len(config.Writes) == 0 {
		return nil, fmt.Errorf("no queries or writes configured")
	}
	options := GoStructOptions{
		PackageName: config.Package,
		StructName:  "QueryRow",
		Target:      GoStructTarget(strings.ToLower(string(config.Client))),
	}
	if options.Target == "" {
		options.Target = GoStructTargetBoth
	}
	if err := validateGoStructOptions(options); err != nil {
		return nil, err
	}
	schemas, err := queryCodegenSchemas(config)
	if err != nil {
		return nil, err
	}
	structs := map[string][]goResultField{}
	var queries []resolvedCodegenQuery
	var querySpecs []resolvedQuerySpec
	for i, query := range config.Queries {
		if query.Name == "" {
			return nil, fmt.Errorf("query name is required")
		}
		if err := validateQueryCodegenQuery(query); err != nil {
			return nil, err
		}
		query, err := resolveCodegenQuerySQL(schemas, query, baseDir)
		if err != nil {
			return nil, err
		}
		if err := validateQueryCodegenParams(query); err != nil {
			return nil, err
		}
		fields, variants, err := analyzeCodegenQuery(schemas, query, baseDir)
		if err != nil {
			return nil, err
		}
		fields, err = applyRequiredFields(fields, query.Required, query.RequiredPolicy)
		if err != nil {
			return nil, fmt.Errorf("query %s: %w", query.Name, err)
		}
		if err := validateUniqueQueryResultFieldNames(fields); err != nil {
			return nil, fmt.Errorf("query %s: %w", query.Name, err)
		}
		structName := queryResultStructName(query)
		merged, err := mergeGoResultFields(structs[structName], fields)
		if err != nil {
			return nil, fmt.Errorf("query %s result_struct %s: %w", query.Name, structName, err)
		}
		structs[structName] = merged
		spec, err := newResolvedQuerySpec(schemas, query, structName, i)
		if err != nil {
			return nil, err
		}
		queries = append(queries, resolvedCodegenQuery{query: query, fields: fields, variants: variants, spec: spec})
		querySpecs = append(querySpecs, spec)
	}
	writeStructFields, writeSpecs, err := planWriteSpecs(schemas, config.Writes, baseDir, structs)
	if err != nil {
		return nil, err
	}
	if err := validateGeneratedPackageNamespace(querySpecs, writeSpecs, structs, writeStructFields, options.Target); err != nil {
		return nil, err
	}
	return &resolvedCodegen{
		options: options, schemas: schemas, queries: queries, querySpecs: querySpecs,
		structs: structs, writeSpecs: writeSpecs, writeStructFields: writeStructFields,
	}, nil
}
