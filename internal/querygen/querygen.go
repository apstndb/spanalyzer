package querygen

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

type QueryCodegenConfig struct {
	Version      string               `json:"version" yaml:"version"`
	Package      string               `json:"package" yaml:"package"`
	Out          string               `json:"out" yaml:"out"`
	Client       GoStructTarget       `json:"client" yaml:"client"`
	Schemas      []QueryCodegenSchema `json:"schemas" yaml:"schemas"`
	Queries      []QueryCodegenQuery  `json:"queries" yaml:"queries"`
	Writes       []QueryCodegenWrite  `json:"writes" yaml:"writes"`
	RuleSeverity map[string]string    `json:"rule_severity,omitempty" yaml:"rule_severity,omitempty"`
}

type QueryCodegenSchema struct {
	Name                     string                               `json:"name" yaml:"name"`
	Dialect                  string                               `json:"dialect" yaml:"dialect"`
	Project                  string                               `json:"project,omitempty" yaml:"project,omitempty"`
	DDL                      string                               `json:"ddl" yaml:"ddl"`
	ProtoDescriptorFiles     []string                             `json:"proto_descriptors" yaml:"proto_descriptors"`
	ExternalSchemas          []QueryCodegenExternalSchema         `json:"external_schemas" yaml:"external_schemas"`
	ExternalQueryConnections []QueryCodegenExternalSchema         `json:"external_query_connections" yaml:"external_query_connections"`
	SpannerExternalDatasets  []QueryCodegenSpannerExternalDataset `json:"spanner_external_datasets" yaml:"spanner_external_datasets"`
}

type QueryCodegenExternalSchema struct {
	Connection    string `json:"connection" yaml:"connection"`
	Schema        string `json:"schema,omitempty" yaml:"schema,omitempty"`
	SpannerSource string `json:"spanner_source,omitempty" yaml:"spanner_source,omitempty"`
}

type QueryCodegenSpannerExternalDataset struct {
	Name                    string                                         `json:"name,omitempty" yaml:"name,omitempty"`
	Dataset                 string                                         `json:"dataset" yaml:"dataset"`
	Project                 string                                         `json:"project,omitempty" yaml:"project,omitempty"`
	SpannerSource           string                                         `json:"spanner_source" yaml:"spanner_source"`
	ExternalSource          string                                         `json:"external_source,omitempty" yaml:"external_source,omitempty"`
	Location                string                                         `json:"location,omitempty" yaml:"location,omitempty"`
	Connection              string                                         `json:"connection,omitempty" yaml:"connection,omitempty"`
	CloudResourceConnection string                                         `json:"cloud_resource_connection,omitempty" yaml:"cloud_resource_connection,omitempty"`
	Access                  QueryCodegenSpannerExternalDatasetAccess       `json:"access,omitempty" yaml:"access,omitempty"`
	DatabaseRole            string                                         `json:"database_role,omitempty" yaml:"database_role,omitempty"`
	AccessVerification      QueryCodegenSpannerExternalDatasetVerification `json:"access_verification,omitempty" yaml:"access_verification,omitempty"`
	UnsupportedColumns      string                                         `json:"unsupported_columns,omitempty" yaml:"unsupported_columns,omitempty"`
	NamedSchemaPolicy       string                                         `json:"named_schema_policy,omitempty" yaml:"named_schema_policy,omitempty"`
	Vet                     QueryCodegenVetConfig                          `json:"vet" yaml:"vet"`
}

type QueryCodegenSpannerExternalDatasetAccess struct {
	Mode                    string                                         `json:"mode,omitempty" yaml:"mode,omitempty"`
	CloudResourceConnection string                                         `json:"cloud_resource_connection,omitempty" yaml:"cloud_resource_connection,omitempty"`
	DatabaseRole            string                                         `json:"database_role,omitempty" yaml:"database_role,omitempty"`
	VerificationHint        string                                         `json:"verification_hint,omitempty" yaml:"verification_hint,omitempty"`
	VerificationEvidence    QueryCodegenSpannerExternalDatasetVerification `json:"verification_evidence,omitempty" yaml:"verification_evidence,omitempty"`
}

type QueryCodegenSpannerExternalDatasetVerification struct {
	Status         string `json:"status,omitempty" yaml:"status,omitempty"`
	Source         string `json:"source,omitempty" yaml:"source,omitempty"`
	CheckedAt      string `json:"checked_at,omitempty" yaml:"checked_at,omitempty"`
	Verifier       string `json:"verifier,omitempty" yaml:"verifier,omitempty"`
	EvidenceDigest string `json:"evidence_digest,omitempty" yaml:"evidence_digest,omitempty"`
}

func (a *QueryCodegenSpannerExternalDatasetAccess) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var mode string
	if err := unmarshal(&mode); err == nil {
		a.Mode = mode
		return nil
	}
	type rawAccess QueryCodegenSpannerExternalDatasetAccess
	var raw rawAccess
	if err := unmarshal(&raw); err != nil {
		return err
	}
	*a = QueryCodegenSpannerExternalDatasetAccess(raw)
	return nil
}

type QueryCodegenQuery struct {
	Name           string                     `json:"name" yaml:"name"`
	Kind           string                     `json:"kind,omitempty" yaml:"kind,omitempty"`
	Catalog        string                     `json:"catalog" yaml:"catalog"`
	SQL            string                     `json:"sql,omitempty" yaml:"sql,omitempty"`
	Table          string                     `json:"table,omitempty" yaml:"table,omitempty"`
	Index          string                     `json:"index,omitempty" yaml:"index,omitempty"`
	Federated      QueryCodegenFederatedQuery `json:"federated" yaml:"federated"`
	KeyPrefix      []string                   `json:"key_prefix" yaml:"key_prefix"`
	OrderBy        string                     `json:"order_by,omitempty" yaml:"order_by,omitempty"`
	Result         string                     `json:"result" yaml:"result"`
	ResultStruct   string                     `json:"result_struct" yaml:"result_struct"`
	Required       []string                   `json:"required" yaml:"required"`
	RequiredPolicy string                     `json:"required_policy" yaml:"required_policy"`
	Params         []QueryCodegenParam        `json:"params" yaml:"params"`
	Vet            QueryCodegenVetConfig      `json:"vet" yaml:"vet"`
}

type QueryCodegenFederatedQuery struct {
	Connection    string `json:"connection" yaml:"connection"`
	SpannerSource string `json:"spanner_source" yaml:"spanner_source"`
	InnerSQL      string `json:"inner_sql" yaml:"inner_sql"`
	OuterSQL      string `json:"outer_sql" yaml:"outer_sql"`
}

type QueryCodegenParam struct {
	Name  string `json:"name" yaml:"name"`
	Type  string `json:"type" yaml:"type"`
	Scope string `json:"scope,omitempty" yaml:"scope,omitempty"`
	// Optional selects how this parameter shapes the generated SQL. The
	// empty string is equivalent to "required" (current behavior). See
	// internal/optparam for the full data model.
	//
	// Supported values: "required", "null_is_null", "omit_when_null",
	// "omit_when_empty", "orderby_choice".
	Optional string `json:"optional,omitempty" yaml:"optional,omitempty"`
	// Choices is the allowlist of ORDER BY clauses for an
	// orderby_choice param, keyed by the runtime choice key.
	Choices map[string]string `json:"choices,omitempty" yaml:"choices,omitempty"`
	// Default is the choice key used when the runtime caller does not
	// specify one. Required for orderby_choice; must match a key in
	// Choices.
	Default string `json:"default,omitempty" yaml:"default,omitempty"`
}

type QueryCodegenVetConfig struct {
	Disable []QueryCodegenVetDisable `json:"disable" yaml:"disable"`
}

type QueryCodegenVetDisable struct {
	Rule    string `json:"rule" yaml:"rule"`
	Reason  string `json:"reason" yaml:"reason"`
	Owner   string `json:"owner,omitempty" yaml:"owner,omitempty"`
	Expires string `json:"expires,omitempty" yaml:"expires,omitempty"`
}

type QueryCodegenWrite struct {
	Name         string                    `json:"name" yaml:"name"`
	Catalog      string                    `json:"catalog" yaml:"catalog"`
	Table        string                    `json:"table" yaml:"table"`
	Operation    string                    `json:"operation" yaml:"operation"`
	InputStruct  string                    `json:"input_struct" yaml:"input_struct"`
	Keys         []string                  `json:"keys" yaml:"keys"`
	Insert       QueryCodegenWriteInsert   `json:"insert,omitempty" yaml:"insert,omitempty"`
	Update       QueryCodegenWriteUpdate   `json:"update,omitempty" yaml:"update,omitempty"`
	Conflict     QueryCodegenWriteConflict `json:"conflict,omitempty" yaml:"conflict,omitempty"`
	Methods      []string                  `json:"methods" yaml:"methods"`
	EmitExplicit bool                      `json:"-" yaml:"-"`
	Vet          QueryCodegenVetConfig     `json:"vet" yaml:"vet"`
}

type QueryCodegenWriteInsert struct {
	Columns []string `json:"columns,omitempty" yaml:"columns,omitempty"`
}

type QueryCodegenWriteUpdate struct {
	Columns []string `json:"columns,omitempty" yaml:"columns,omitempty"`
}

type QueryCodegenWriteConflict struct {
	Target   []string `json:"target,omitempty" yaml:"target,omitempty"`
	Strategy string   `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

const autoAllNonKeyColumns = "auto_all_non_key_columns"

type QueryCodegenPlan struct {
	PlanVersion     int                                     `json:"plan_version" yaml:"plan_version"`
	Generator       QueryCodegenPlanGenerator               `json:"generator" yaml:"generator"`
	Package         string                                  `json:"package,omitempty" yaml:"package,omitempty"`
	Out             string                                  `json:"out,omitempty" yaml:"out,omitempty"`
	Client          GoStructTarget                          `json:"client,omitempty" yaml:"client,omitempty"`
	SchemaDigests   []QueryCodegenPlanDigest                `json:"schema_digests,omitempty" yaml:"schema_digests,omitempty"`
	CatalogBindings []BigQuerySpannerExternalDatasetBinding `json:"catalog_bindings,omitempty" yaml:"catalog_bindings,omitempty"`
	Queries         []QueryCodegenPlanQuery                 `json:"queries,omitempty" yaml:"queries,omitempty"`
	Writes          []QueryCodegenPlanWrite                 `json:"writes,omitempty" yaml:"writes,omitempty"`
	Structs         []QueryCodegenPlanStruct                `json:"structs,omitempty" yaml:"structs,omitempty"`
}

type QueryCodegenPlanGenerator struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
}

type QueryCodegenPlanDigest struct {
	Catalog string `json:"catalog" yaml:"catalog"`
	Kind    string `json:"kind" yaml:"kind"`
	Path    string `json:"path,omitempty" yaml:"path,omitempty"`
	SHA256  string `json:"sha256" yaml:"sha256"`
}

type QueryCodegenPlanQuery struct {
	Name            string                           `json:"name" yaml:"name"`
	Catalog         string                           `json:"catalog" yaml:"catalog"`
	Kind            string                           `json:"kind" yaml:"kind"`
	Result          string                           `json:"result" yaml:"result"`
	OrderBy         string                           `json:"order_by,omitempty" yaml:"order_by,omitempty"`
	SQL             string                           `json:"sql" yaml:"sql"`
	SQLSHA256       string                           `json:"sql_sha256" yaml:"sql_sha256"`
	ResultStruct    string                           `json:"result_struct" yaml:"result_struct"`
	Constants       []QueryCodegenPlanConstant       `json:"constants,omitempty" yaml:"constants,omitempty"`
	Params          []QueryCodegenParam              `json:"params,omitempty" yaml:"params,omitempty"`
	StarExpansion   *QueryCodegenPlanStarExpansion   `json:"star_expansion,omitempty" yaml:"star_expansion,omitempty"`
	Relations       []QueryCodegenPlanRelation       `json:"relations,omitempty" yaml:"relations,omitempty"`
	VetSuppressions []QueryCodegenPlanVetSuppression `json:"vet_suppressions,omitempty" yaml:"vet_suppressions,omitempty"`
	Warnings        []QueryCodegenPlanWarning        `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	Fields          []QueryCodegenPlanField          `json:"fields,omitempty" yaml:"fields,omitempty"`
	// Variants is the per-shape plan-contract entry produced by
	// optparam.BuildPlanVariants. It is populated only for queries
	// that use one or more optional markers. The top-level SQL /
	// SQLSHA256 fields stay populated with the all-on variant so
	// existing consumers continue to work.
	Variants []QueryCodegenPlanQueryVariant `json:"variants,omitempty" yaml:"variants,omitempty"`
}

// QueryCodegenPlanQueryVariant is one row of the per-variant plan
// contract for a query with optional markers. The shape mirrors
// internal/optparam.PlanQueryVariant.
type QueryCodegenPlanQueryVariant struct {
	Label             string            `json:"label" yaml:"label"`
	SQL               string            `json:"sql" yaml:"sql"`
	SQLSHA256         string            `json:"sql_sha256" yaml:"sql_sha256"`
	PresentParams     []string          `json:"present_params,omitempty" yaml:"present_params,omitempty"`
	AbsentParams      []string          `json:"absent_params,omitempty" yaml:"absent_params,omitempty"`
	ChoiceAssignments map[string]string `json:"choice_assignments,omitempty" yaml:"choice_assignments,omitempty"`
}

type QueryCodegenPlanStarExpansion struct {
	Catalog        string                     `json:"catalog" yaml:"catalog"`
	ProjectionLoss bool                       `json:"projection_loss" yaml:"projection_loss"`
	OmittedColumns []string                   `json:"omitted_columns,omitempty" yaml:"omitted_columns,omitempty"`
	Projection     QueryCodegenPlanProjection `json:"projection" yaml:"projection"`
}

type QueryCodegenPlanRelation struct {
	SQLPath        string                     `json:"sql_path" yaml:"sql_path"`
	Catalog        string                     `json:"catalog" yaml:"catalog"`
	Role           string                     `json:"role" yaml:"role"`
	Allowed        bool                       `json:"allowed" yaml:"allowed"`
	WritableTarget bool                       `json:"writable_target" yaml:"writable_target"`
	Diagnostic     string                     `json:"diagnostic,omitempty" yaml:"diagnostic,omitempty"`
	ProjectionLoss bool                       `json:"projection_loss" yaml:"projection_loss"`
	OmittedColumns []string                   `json:"omitted_columns,omitempty" yaml:"omitted_columns,omitempty"`
	Projection     QueryCodegenPlanProjection `json:"projection" yaml:"projection"`
}

type QueryCodegenPlanProjection struct {
	RelationHasOmittedColumns bool     `json:"relation_has_omitted_columns" yaml:"relation_has_omitted_columns"`
	SelectedOutputAffected    bool     `json:"selected_output_affected" yaml:"selected_output_affected"`
	OmittedColumns            []string `json:"omitted_columns,omitempty" yaml:"omitted_columns,omitempty"`
}

type QueryCodegenPlanConstant struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

type QueryCodegenPlanWrite struct {
	Name                    string                             `json:"name" yaml:"name"`
	Catalog                 string                             `json:"catalog" yaml:"catalog"`
	Table                   string                             `json:"table" yaml:"table"`
	Operation               string                             `json:"operation" yaml:"operation"`
	InputStruct             string                             `json:"input_struct" yaml:"input_struct"`
	Keys                    []string                           `json:"keys,omitempty" yaml:"keys,omitempty"`
	InsertColumns           []string                           `json:"insert_columns,omitempty" yaml:"insert_columns,omitempty"`
	UpdateColumns           []string                           `json:"update_columns,omitempty" yaml:"update_columns,omitempty"`
	Methods                 []string                           `json:"methods,omitempty" yaml:"methods,omitempty"`
	ColumnCapabilities      []QueryCodegenPlanColumnCapability `json:"column_capabilities,omitempty" yaml:"column_capabilities,omitempty"`
	ServerSideUpdateEffects []QueryCodegenPlanServerSideEffect `json:"server_side_update_effects,omitempty" yaml:"server_side_update_effects,omitempty"`
	VetSuppressions         []QueryCodegenPlanVetSuppression   `json:"vet_suppressions,omitempty" yaml:"vet_suppressions,omitempty"`
}

type QueryCodegenPlanColumnCapability struct {
	Name                 string `json:"name" yaml:"name"`
	Nullable             bool   `json:"nullable" yaml:"nullable"`
	PrimaryKey           bool   `json:"primary_key,omitempty" yaml:"primary_key,omitempty"`
	Hidden               bool   `json:"hidden,omitempty" yaml:"hidden,omitempty"`
	InsertValue          bool   `json:"insert_value" yaml:"insert_value"`
	UpdateValue          bool   `json:"update_value" yaml:"update_value"`
	DefaultSQL           string `json:"default_sql,omitempty" yaml:"default_sql,omitempty"`
	GeneratedSQL         string `json:"generated_sql,omitempty" yaml:"generated_sql,omitempty"`
	OnUpdateSQL          string `json:"on_update_sql,omitempty" yaml:"on_update_sql,omitempty"`
	AllowCommitTimestamp bool   `json:"allow_commit_timestamp,omitempty" yaml:"allow_commit_timestamp,omitempty"`
}

type QueryCodegenPlanServerSideEffect struct {
	Column string `json:"column" yaml:"column"`
	Reason string `json:"reason" yaml:"reason"`
}

type QueryCodegenPlanStruct struct {
	Name   string                  `json:"name" yaml:"name"`
	Fields []QueryCodegenPlanField `json:"fields,omitempty" yaml:"fields,omitempty"`
}

type QueryCodegenPlanField struct {
	Name     string                  `json:"name" yaml:"name"`
	Kind     string                  `json:"kind" yaml:"kind"`
	Repeated bool                    `json:"repeated,omitempty" yaml:"repeated,omitempty"`
	Nullable bool                    `json:"nullable" yaml:"nullable"`
	Fields   []QueryCodegenPlanField `json:"fields,omitempty" yaml:"fields,omitempty"`
}

func GenerateQueryCode(config QueryCodegenConfig, baseDir string) (string, error) {
	if len(config.Queries) == 0 && len(config.Writes) == 0 {
		return "", fmt.Errorf("no queries or writes configured")
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
		return "", err
	}
	schemas, err := queryCodegenSchemas(config)
	if err != nil {
		return "", err
	}
	structs := map[string][]goResultField{}
	constants := make([]generatedGoConst, 0, len(config.Queries))
	var builderCode bytes.Buffer
	var builderImports []string
	var querySpecs []resolvedQuerySpec
	for _, query := range config.Queries {
		if query.Name == "" {
			return "", fmt.Errorf("query name is required")
		}
		if err := validateQueryCodegenQuery(query); err != nil {
			return "", err
		}
		query, err := resolveCodegenQuerySQL(schemas, query, baseDir)
		if err != nil {
			return "", err
		}
		if err := validateQueryCodegenParams(query); err != nil {
			return "", err
		}
		fields, _, err := analyzeCodegenQuery(schemas, query, baseDir)
		if err != nil {
			return "", err
		}
		fields, err = applyRequiredFields(fields, query.Required, query.RequiredPolicy)
		if err != nil {
			return "", fmt.Errorf("query %s: %w", query.Name, err)
		}
		structName := queryResultStructName(query)
		merged, err := mergeGoResultFields(structs[structName], fields)
		if err != nil {
			return "", fmt.Errorf("query %s result_struct %s: %w", query.Name, structName, err)
		}
		structs[structName] = merged
		sourceName, _ := querySourceName(schemas, query)
		methodPrefix := exportedIdentifier(query.Name, "Query")
		var builderFunc, paramsType string
		if queryHasOptionalMarkers(query) {
			builderFunc = "Build" + methodPrefix + "SQL"
			paramsType = methodPrefix + "Params"
			code, imports, err := emitQueryGoBuilder(query, builderFunc, paramsType)
			if err != nil {
				return "", fmt.Errorf("query %s: %w", query.Name, err)
			}
			if builderCode.Len() > 0 {
				builderCode.WriteByte('\n')
			}
			builderCode.WriteString(code)
			builderImports = append(builderImports, imports...)
		} else {
			constants = append(constants, queryGoConstants(query)...)
		}
		querySpecs = append(querySpecs, resolvedQuerySpec{
			Name:         query.Name,
			MethodPrefix: methodPrefix,
			ResultStruct: structName,
			ResultMode:   queryResultMode(query),
			Params:       query.Params,
			Dialect:      emptyDefault(schemas[sourceName].Dialect, "spanner"),
			BuilderFunc:  builderFunc,
			ParamsType:   paramsType,
		})
	}
	writeImports, writeCode, err := generateWriteCode(schemas, config.Writes, baseDir, structs)
	if err != nil {
		return "", err
	}
	var queryMethods bytes.Buffer
	methodImports := map[string]struct{}{}
	for _, spec := range querySpecs {
		queryMethods.WriteByte('\n')
		writeQueryMethods(&queryMethods, spec, methodImports)
	}
	allImports := make([]string, 0, len(writeImports)+len(methodImports))
	allImports = append(allImports, writeImports...)
	for imp := range methodImports {
		allImports = append(allImports, imp)
	}
	allImports = append(allImports, builderImports...)

	names := make([]string, 0, len(structs))
	for name := range structs {
		names = append(names, name)
	}
	sort.Strings(names)
	namedStructs := make([]namedGoStruct, 0, len(names))
	for _, name := range names {
		namedStructs = append(namedStructs, namedGoStruct{Name: name, Fields: structs[name]})
	}
	return generateGoStructsWithExtra(namedStructs, options, constants, allImports, writeCode+builderCode.String()+queryMethods.String())
}

type resolvedQuerySpec struct {
	Name         string
	MethodPrefix string
	ResultStruct string
	ResultMode   string
	Params       []QueryCodegenParam
	Dialect      string
	BuilderFunc  string
	ParamsType   string
}

func writeQueryMethods(b *bytes.Buffer, spec resolvedQuerySpec, imports map[string]struct{}) {
	switch strings.ToLower(spec.Dialect) {
	case "bigquery":
		writeBigQueryMethods(b, spec, imports)
	default:
		writeSpannerMethods(b, spec, imports)
	}
}

func spannerMethodParams(spec resolvedQuerySpec) string {
	if spec.BuilderFunc != "" {
		return spec.ParamsType
	}
	return "map[string]interface{}"
}

func writeSpannerStatementSetup(b *bytes.Buffer, spec resolvedQuerySpec) {
	if spec.BuilderFunc != "" {
		fmt.Fprintf(b, "\tsql, args, _ := %s(params)\n", spec.BuilderFunc)
	}
}

func spannerStatementExpr(spec resolvedQuerySpec, constName string) string {
	if spec.BuilderFunc != "" {
		return "spanner.Statement{SQL: sql, Params: args}"
	}
	return "spanner.Statement{SQL: " + constName + ", Params: params}"
}

func writeSpannerMethods(b *bytes.Buffer, spec resolvedQuerySpec, imports map[string]struct{}) {
	imports["context"] = struct{}{}
	imports["cloud.google.com/go/spanner"] = struct{}{}
	constName := spec.MethodPrefix + "SQL"
	paramsType := spannerMethodParams(spec)
	switch spec.ResultMode {
	case "many":
		imports["google.golang.org/api/iterator"] = struct{}{}
		fmt.Fprintf(b, "// %s returns a Cloud Spanner row iterator.\n", spec.MethodPrefix)
		fmt.Fprintf(b, "func %s(ctx context.Context, tx spanner.ReadOnlyTransaction, params %s) *spanner.RowIterator {\n", spec.MethodPrefix, paramsType)
		writeSpannerStatementSetup(b, spec)
		fmt.Fprintf(b, "\treturn tx.Query(ctx, %s)\n", spannerStatementExpr(spec, constName))
		b.WriteString("}\n")
		fmt.Fprintf(b, "// %sAll returns all Cloud Spanner rows as a slice.\n", spec.MethodPrefix)
		fmt.Fprintf(b, "func %sAll(ctx context.Context, tx spanner.ReadOnlyTransaction, params %s) ([]*%s, error) {\n", spec.MethodPrefix, paramsType, spec.ResultStruct)
		fmt.Fprintf(b, "\tit := %s(ctx, tx, params)\n", spec.MethodPrefix)
		b.WriteString("\tdefer it.Stop()\n")
		fmt.Fprintf(b, "\tvar out []*%s\n", spec.ResultStruct)
		b.WriteString("\tfor {\n")
		b.WriteString("\t\trow, err := it.Next()\n")
		b.WriteString("\t\tif err == iterator.Done {\n")
		b.WriteString("\t\t\treturn out, nil\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tif err != nil {\n")
		b.WriteString("\t\t\treturn nil, err\n")
		b.WriteString("\t\t}\n")
		fmt.Fprintf(b, "\t\tvar r %s\n", spec.ResultStruct)
		b.WriteString("\t\tif err := row.ToStruct(&r); err != nil {\n")
		b.WriteString("\t\t\treturn nil, err\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tout = append(out, &r)\n")
		b.WriteString("\t}\n")
		b.WriteString("}\n")
	case "one", "maybe_one":
		imports["fmt"] = struct{}{}
		imports["google.golang.org/api/iterator"] = struct{}{}
		fmt.Fprintf(b, "// %s returns a single Cloud Spanner row.\n", spec.MethodPrefix)
		fmt.Fprintf(b, "func %s(ctx context.Context, tx spanner.ReadOnlyTransaction, params %s) (*%s, error) {\n", spec.MethodPrefix, paramsType, spec.ResultStruct)
		writeSpannerStatementSetup(b, spec)
		fmt.Fprintf(b, "\tit := tx.Query(ctx, %s)\n", spannerStatementExpr(spec, constName))
		b.WriteString("\tdefer it.Stop()\n")
		b.WriteString("\trow, err := it.Next()\n")
		b.WriteString("\tif err != nil {\n")
		if spec.ResultMode == "one" {
			b.WriteString("\t\treturn nil, err\n")
		} else {
			b.WriteString("\t\tif err == iterator.Done {\n")
			b.WriteString("\t\t\treturn nil, nil\n")
			b.WriteString("\t\t}\n")
			b.WriteString("\t\treturn nil, err\n")
		}
		b.WriteString("\t}\n")
		b.WriteString("\tif _, err := it.Next(); err != iterator.Done {\n")
		b.WriteString("\t\tif err == nil {\n")
		fmt.Fprintf(b, "\t\t\treturn nil, fmt.Errorf(\"query %%s: expected at most one row, but got multiple\", %q)\n", spec.Name)
		b.WriteString("\t\t}\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")
		fmt.Fprintf(b, "\tvar r %s\n", spec.ResultStruct)
		b.WriteString("\tif err := row.ToStruct(&r); err != nil {\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn &r, nil\n")
		b.WriteString("}\n")
	case "row_count":
		fmt.Fprintf(b, "// %s executes a Cloud Spanner DML statement and returns the row count.\n", spec.MethodPrefix)
		fmt.Fprintf(b, "func %s(ctx context.Context, tx spanner.ReadWriteTransaction, params %s) (int64, error) {\n", spec.MethodPrefix, paramsType)
		writeSpannerStatementSetup(b, spec)
		fmt.Fprintf(b, "\treturn tx.Update(ctx, %s)\n", spannerStatementExpr(spec, constName))
		b.WriteString("}\n")
	case "row_set":
		fmt.Fprintf(b, "// %s executes a Cloud Spanner DML statement with THEN RETURN and returns a row iterator.\n", spec.MethodPrefix)
		fmt.Fprintf(b, "func %s(ctx context.Context, tx spanner.ReadWriteTransaction, params %s) *spanner.RowIterator {\n", spec.MethodPrefix, paramsType)
		writeSpannerStatementSetup(b, spec)
		fmt.Fprintf(b, "\treturn tx.Query(ctx, %s)\n", spannerStatementExpr(spec, constName))
		b.WriteString("}\n")
	}
}

func writeBigQueryMethods(b *bytes.Buffer, spec resolvedQuerySpec, imports map[string]struct{}) {
	imports["context"] = struct{}{}
	imports["cloud.google.com/go/bigquery"] = struct{}{}
	constName := spec.MethodPrefix + "SQL"
	switch spec.ResultMode {
	case "many":
		imports["google.golang.org/api/iterator"] = struct{}{}
		fmt.Fprintf(b, "// %s returns a BigQuery row iterator.\n", spec.MethodPrefix)
		fmt.Fprintf(b, "func %s(ctx context.Context, client *bigquery.Client, params []bigquery.QueryParameter) (*bigquery.RowIterator, error) {\n", spec.MethodPrefix)
		fmt.Fprintf(b, "\tq := client.Query(%s)\n", constName)
		b.WriteString("\tq.Parameters = params\n")
		b.WriteString("\treturn q.Read(ctx)\n")
		b.WriteString("}\n")
		fmt.Fprintf(b, "// %sAll returns all BigQuery rows as a slice.\n", spec.MethodPrefix)
		fmt.Fprintf(b, "func %sAll(ctx context.Context, client *bigquery.Client, params []bigquery.QueryParameter) ([]*%s, error) {\n", spec.MethodPrefix, spec.ResultStruct)
		fmt.Fprintf(b, "\tit, err := %s(ctx, client, params)\n", spec.MethodPrefix)
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")
		fmt.Fprintf(b, "\tvar out []*%s\n", spec.ResultStruct)
		b.WriteString("\tfor {\n")
		fmt.Fprintf(b, "\t\tvar r %s\n", spec.ResultStruct)
		b.WriteString("\t\terr := it.Next(&r)\n")
		b.WriteString("\t\tif err == iterator.Done {\n")
		b.WriteString("\t\t\treturn out, nil\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tif err != nil {\n")
		b.WriteString("\t\t\treturn nil, err\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tout = append(out, &r)\n")
		b.WriteString("\t}\n")
		b.WriteString("}\n")
	case "one", "maybe_one":
		imports["fmt"] = struct{}{}
		imports["google.golang.org/api/iterator"] = struct{}{}
		fmt.Fprintf(b, "// %s returns a single BigQuery row.\n", spec.MethodPrefix)
		fmt.Fprintf(b, "func %s(ctx context.Context, client *bigquery.Client, params []bigquery.QueryParameter) (*%s, error) {\n", spec.MethodPrefix, spec.ResultStruct)
		fmt.Fprintf(b, "\tit, err := %s(ctx, client, params)\n", spec.MethodPrefix)
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")
		fmt.Fprintf(b, "\tvar r %s\n", spec.ResultStruct)
		b.WriteString("\terr = it.Next(&r)\n")
		b.WriteString("\tif err != nil {\n")
		if spec.ResultMode == "one" {
			b.WriteString("\t\treturn nil, err\n")
		} else {
			b.WriteString("\t\tif err == iterator.Done {\n")
			b.WriteString("\t\t\treturn nil, nil\n")
			b.WriteString("\t\t}\n")
			b.WriteString("\t\treturn nil, err\n")
		}
		b.WriteString("\t}\n")
		b.WriteString("\tif err := it.Next(&r); err != iterator.Done {\n")
		b.WriteString("\t\tif err == nil {\n")
		fmt.Fprintf(b, "\t\t\treturn nil, fmt.Errorf(\"query %%s: expected at most one row, but got multiple\", %q)\n", spec.Name)
		b.WriteString("\t\t}\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn &r, nil\n")
		b.WriteString("}\n")
	}
}

func queryCodegenSchemas(config QueryCodegenConfig) (map[string]QueryCodegenSchema, error) {
	schemas := map[string]QueryCodegenSchema{}
	for _, schema := range config.Schemas {
		if schema.Name == "" {
			return nil, fmt.Errorf("schema name is required")
		}
		if _, ok := schemas[schema.Name]; ok {
			return nil, fmt.Errorf("duplicate schema %q", schema.Name)
		}
		schemas[schema.Name] = schema
	}
	if len(schemas) == 0 {
		schemas["default"] = QueryCodegenSchema{Name: "default", Dialect: "spanner"}
	}
	return schemas, nil
}

func queryCodegenExternalQueryConnections(schema QueryCodegenSchema) []QueryCodegenExternalSchema {
	out := make([]QueryCodegenExternalSchema, 0, len(schema.ExternalSchemas)+len(schema.ExternalQueryConnections))
	for _, external := range schema.ExternalSchemas {
		if external.SpannerSource == "" {
			external.SpannerSource = external.Schema
		}
		if external.Schema == "" {
			external.Schema = external.SpannerSource
		}
		out = append(out, external)
	}
	for _, external := range schema.ExternalQueryConnections {
		if external.SpannerSource == "" {
			external.SpannerSource = external.Schema
		}
		if external.Schema == "" {
			external.Schema = external.SpannerSource
		}
		out = append(out, external)
	}
	return out
}

func queryCodegenSchemaDigests(schemas map[string]QueryCodegenSchema, baseDir string) ([]QueryCodegenPlanDigest, error) {
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]QueryCodegenPlanDigest, 0, len(names))
	for _, name := range names {
		schema := schemas[name]
		path, ddl, err := readCodegenDDL(schema, baseDir)
		if err != nil {
			return nil, fmt.Errorf("schema %s: %w", name, err)
		}
		out = append(out, QueryCodegenPlanDigest{
			Catalog: name,
			Kind:    "ddl",
			Path:    path,
			SHA256:  digestString(ddl),
		})
		for _, descriptorPath := range schema.ProtoDescriptorFiles {
			path := resolveCodegenPath(baseDir, descriptorPath)
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("schema %s proto descriptor %s: %w", name, descriptorPath, err)
			}
			out = append(out, QueryCodegenPlanDigest{
				Catalog: name,
				Kind:    "proto_descriptor",
				Path:    path,
				SHA256:  digestBytes(data),
			})
		}
	}
	return out, nil
}

func queryCodegenCatalogBindings(schemas map[string]QueryCodegenSchema, baseDir string) ([]BigQuerySpannerExternalDatasetBinding, error) {
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	var bindings []BigQuerySpannerExternalDatasetBinding
	for _, name := range names {
		schema := schemas[name]
		if strings.ToLower(emptyDefault(schema.Dialect, "spanner")) != "bigquery" || len(schema.SpannerExternalDatasets) == 0 {
			continue
		}
		analyzer, err := NewBigQueryAnalyzerFromDDL("catalog-bindings.sql", "")
		if err != nil {
			return nil, fmt.Errorf("schema %s: %w", name, err)
		}
		schemaBindings, err := addCodegenSpannerExternalDatasets(analyzer, schemas, schema, baseDir)
		if err != nil {
			return nil, fmt.Errorf("schema %s: %w", name, err)
		}
		bindings = append(bindings, schemaBindings...)
	}
	return bindings, nil
}

func digestString(value string) string {
	return digestBytes([]byte(value))
}

func digestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func BuildQueryCodegenPlan(config QueryCodegenConfig, baseDir string) (*QueryCodegenPlan, error) {
	if len(config.Queries) == 0 && len(config.Writes) == 0 {
		return nil, fmt.Errorf("no queries or writes configured")
	}
	schemas, err := queryCodegenSchemas(config)
	if err != nil {
		return nil, err
	}
	structs := map[string][]goResultField{}
	plan := &QueryCodegenPlan{
		PlanVersion: 1,
		Generator: QueryCodegenPlanGenerator{
			Name:    "spanner-query-gen",
			Version: "0.0.0-dev",
		},
		Package: config.Package,
		Out:     config.Out,
		Client:  config.Client,
	}
	schemaDigests, err := queryCodegenSchemaDigests(schemas, baseDir)
	if err != nil {
		return nil, err
	}
	plan.SchemaDigests = schemaDigests
	catalogBindings, err := queryCodegenCatalogBindings(schemas, baseDir)
	if err != nil {
		return nil, err
	}
	plan.CatalogBindings = catalogBindings
	applyCatalogBindingSeverityOverrides(plan.CatalogBindings, config.RuleSeverity)
	for _, query := range config.Queries {
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
		structName := queryResultStructName(query)
		merged, err := mergeGoResultFields(structs[structName], fields)
		if err != nil {
			return nil, fmt.Errorf("query %s result_struct %s: %w", query.Name, structName, err)
		}
		structs[structName] = merged
		sourceName, err := querySourceName(schemas, query)
		if err != nil {
			return nil, err
		}
		warnings, err := queryPlanWarnings(schemas, query, baseDir)
		if err != nil {
			return nil, err
		}
		applyWarningSeverityOverrides(warnings, config.RuleSeverity)
		topSQL, topSHA := query.SQL, digestString(query.SQL)
		planVariants := variants
		if len(variants) > 0 {
			// Use the all-on (most-fields-present) variant as the
			// canonical SQL so single-SQL consumers see the rewritten
			// shape (eg. IS NOT DISTINCT FROM in place of = @p).
			top := pickCanonicalVariant(variants)
			topSQL, topSHA = top.SQL, top.SQLSHA256
			// A single-variant slice carries no new information beyond
			// the top-level SQL; suppress it to keep the plan tidy.
			if len(variants) == 1 {
				planVariants = nil
			}
		}
		plan.Queries = append(plan.Queries, QueryCodegenPlanQuery{
			Name:            query.Name,
			Catalog:         sourceName,
			Kind:            queryKind(query),
			Result:          queryResultMode(query),
			OrderBy:         queryPlanOrderByMode(schemas[sourceName], query),
			SQL:             topSQL,
			SQLSHA256:       topSHA,
			ResultStruct:    structName,
			Constants:       planConstants(queryGoConstants(query)),
			Params:          append([]QueryCodegenParam(nil), query.Params...),
			Variants:        planVariants,
			StarExpansion:   queryPlanStarExpansion(query, plan.CatalogBindings),
			Relations:       queryPlanRelations(query, plan.CatalogBindings),
			VetSuppressions: planVetSuppressions(query.Vet),
			Warnings:        warnings,
			Fields:          planFields(fields),
		})
	}
	writeStructFields, writeSpecs, err := planWriteSpecs(schemas, config.Writes, baseDir, structs)
	if err != nil {
		return nil, err
	}
	for _, spec := range writeSpecs {
		plan.Writes = append(plan.Writes, QueryCodegenPlanWrite{
			Name:                    spec.Name,
			Catalog:                 spec.Catalog,
			Table:                   spec.Table,
			Operation:               spec.Operation,
			InputStruct:             spec.InputStruct,
			Keys:                    columnNamesFromColumns(spec.Keys),
			InsertColumns:           columnNamesFromColumns(spec.InsertColumns),
			UpdateColumns:           columnNamesFromColumns(spec.UpdateColumns),
			Methods:                 spec.Methods,
			ColumnCapabilities:      planColumnCapabilities(spec.AllColumns),
			ServerSideUpdateEffects: planServerSideUpdateEffects(spec),
			VetSuppressions:         spec.VetSuppressions,
		})
	}
	for name, fields := range structs {
		plan.Structs = append(plan.Structs, QueryCodegenPlanStruct{Name: name, Fields: planFields(fields)})
	}
	for name, fields := range writeStructFields {
		plan.Structs = append(plan.Structs, QueryCodegenPlanStruct{Name: name, Fields: planFields(fields)})
	}
	sort.Slice(plan.Structs, func(i, j int) bool {
		return plan.Structs[i].Name < plan.Structs[j].Name
	})
	return plan, nil
}

func applyCatalogBindingSeverityOverrides(bindings []BigQuerySpannerExternalDatasetBinding, overrides map[string]string) {
	if len(overrides) == 0 {
		return
	}
	for i := range bindings {
		applyWarningSeverityOverrides(bindings[i].Warnings, overrides)
	}
}

func applyWarningSeverityOverrides(warnings []QueryCodegenPlanWarning, overrides map[string]string) {
	if len(overrides) == 0 {
		return
	}
	for i := range warnings {
		if severity := overrides[warnings[i].Rule]; severity != "" {
			warnings[i].Severity = severity
		}
	}
}

func queryResultStructName(query QueryCodegenQuery) string {
	if query.ResultStruct != "" {
		return exportedIdentifier(query.ResultStruct, "QueryRow")
	}
	return exportedIdentifier(query.Name, "Query") + "Row"
}

func writeInputStructName(write QueryCodegenWrite) string {
	if write.InputStruct != "" {
		return exportedIdentifier(write.InputStruct, "Write")
	}
	return exportedIdentifier(write.Name, "Write") + "Write"
}

func queryGoConstants(query QueryCodegenQuery) []generatedGoConst {
	if !query.Federated.isZero() {
		return []generatedGoConst{
			{Name: query.Name + "SpannerSQL", Value: query.Federated.InnerSQL},
			{Name: query.Name + "BigQuerySQL", Value: query.SQL},
		}
	}
	return []generatedGoConst{{Name: query.Name, Value: query.SQL}}
}

func planConstants(constants []generatedGoConst) []QueryCodegenPlanConstant {
	out := make([]QueryCodegenPlanConstant, 0, len(constants))
	for _, constant := range constants {
		out = append(out, QueryCodegenPlanConstant(constant))
	}
	return out
}

func planVetSuppressions(config QueryCodegenVetConfig) []QueryCodegenPlanVetSuppression {
	out := make([]QueryCodegenPlanVetSuppression, 0, len(config.Disable))
	for _, disabled := range config.Disable {
		out = append(out, QueryCodegenPlanVetSuppression(disabled))
	}
	return out
}

func planFields(fields []goResultField) []QueryCodegenPlanField {
	out := make([]QueryCodegenPlanField, 0, len(fields))
	for _, field := range fields {
		out = append(out, QueryCodegenPlanField{
			Name:     field.Name,
			Kind:     field.Kind,
			Repeated: field.Repeated,
			Nullable: field.Nullable,
			Fields:   planFields(field.Fields),
		})
	}
	return out
}

func queryPlanStarExpansion(query QueryCodegenQuery, bindings []BigQuerySpannerExternalDatasetBinding) *QueryCodegenPlanStarExpansion {
	if !sqlHasStarToken(query.SQL) {
		return nil
	}
	expansion := &QueryCodegenPlanStarExpansion{Catalog: "sql"}
	var omitted []string
	matchedExternalDataset := false
	for _, binding := range bindings {
		for _, table := range binding.ProjectedTables {
			if !table.Visible || !sqlReferencesExternalDatasetTable(query.SQL, binding, table) {
				continue
			}
			matchedExternalDataset = true
			expansion.Catalog = "external_dataset_projection"
			for _, column := range table.Columns {
				if column.Visible {
					continue
				}
				omitted = append(omitted, table.SourceTable+"."+column.Name)
			}
		}
	}
	if !matchedExternalDataset {
		return nil
	}
	sort.Strings(omitted)
	expansion.OmittedColumns = omitted
	expansion.ProjectionLoss = len(omitted) > 0
	expansion.Projection = QueryCodegenPlanProjection{
		RelationHasOmittedColumns: len(omitted) > 0,
		SelectedOutputAffected:    len(omitted) > 0,
		OmittedColumns:            omitted,
	}
	return expansion
}

func queryPlanRelations(query QueryCodegenQuery, bindings []BigQuerySpannerExternalDatasetBinding) []QueryCodegenPlanRelation {
	return spannerExternalDatasetRelations(query.SQL, bindings)
}

func spannerExternalDatasetRelations(sql string, bindings []BigQuerySpannerExternalDatasetBinding) []QueryCodegenPlanRelation {
	var out []QueryCodegenPlanRelation
	seen := map[string]bool{}
	for _, binding := range bindings {
		for _, table := range binding.ProjectedTables {
			role, matched := spannerExternalDatasetTableReferenceRole(sql, binding, table)
			if !table.Visible || !matched {
				continue
			}
			relation := QueryCodegenPlanRelation{
				SQLPath:        table.BigQueryTable,
				Catalog:        "spanner_external_dataset_projection",
				Role:           role,
				Allowed:        role == "select_source",
				WritableTarget: false,
			}
			if !relation.Allowed {
				relation.Diagnostic = "external_dataset_table_is_not_writable"
			}
			for _, column := range table.Columns {
				if column.Visible {
					continue
				}
				relation.OmittedColumns = append(relation.OmittedColumns, table.SourceTable+"."+column.Name)
			}
			sort.Strings(relation.OmittedColumns)
			relation.ProjectionLoss = len(relation.OmittedColumns) > 0
			relation.Projection = QueryCodegenPlanProjection{
				RelationHasOmittedColumns: len(relation.OmittedColumns) > 0,
				SelectedOutputAffected:    sqlHasStarToken(sql) && len(relation.OmittedColumns) > 0,
				OmittedColumns:            relation.OmittedColumns,
			}
			key := strings.ToLower(relation.SQLPath)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, relation)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].SQLPath) < strings.ToLower(out[j].SQLPath)
	})
	return out
}

func spannerExternalDatasetTableReferenceRole(sql string, binding BigQuerySpannerExternalDatasetBinding, table BigQuerySpannerExternalDatasetTable) (string, bool) {
	if !sqlReferencesExternalDatasetTable(sql, binding, table) {
		return "", false
	}
	normalizedSQL := normalizeSQLForTableReference(sql)
	names := externalDatasetTableReferenceNames(binding, table)
	for _, name := range names {
		normalizedName := normalizeSQLForTableReference(name)
		if normalizedName == "" {
			continue
		}
		if sqlTargetReference(normalizedSQL, "insert into", normalizedName) ||
			sqlTargetReference(normalizedSQL, "insert", normalizedName) ||
			sqlTargetReference(normalizedSQL, "update", normalizedName) ||
			sqlTargetReference(normalizedSQL, "merge into", normalizedName) ||
			sqlTargetReference(normalizedSQL, "merge", normalizedName) ||
			sqlTargetReference(normalizedSQL, "delete from", normalizedName) {
			return "dml_target", true
		}
		if sqlTargetReference(normalizedSQL, "truncate table", normalizedName) ||
			sqlTargetReference(normalizedSQL, "alter table", normalizedName) ||
			sqlTargetReference(normalizedSQL, "drop table", normalizedName) ||
			sqlTargetReference(normalizedSQL, "create table", normalizedName) ||
			sqlTargetReference(normalizedSQL, "create or replace table", normalizedName) {
			return "metadata_target", true
		}
	}
	return "select_source", true
}

func sqlTargetReference(sql, prefix, table string) bool {
	if !strings.HasPrefix(sql, prefix+" ") {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(sql, prefix))
	return rest == table ||
		strings.HasPrefix(rest, table+" ") ||
		strings.HasPrefix(rest, table+"(") ||
		strings.HasPrefix(rest, table+"\n")
}

func sqlReferencesExternalDatasetTable(sql string, binding BigQuerySpannerExternalDatasetBinding, table BigQuerySpannerExternalDatasetTable) bool {
	sql = normalizeSQLForTableReference(sql)
	for _, name := range externalDatasetTableReferenceNames(binding, table) {
		name = normalizeSQLForTableReference(name)
		if name == "" {
			continue
		}
		if strings.Contains(sql, name) {
			return true
		}
	}
	return false
}

func externalDatasetTableReferenceNames(binding BigQuerySpannerExternalDatasetBinding, table BigQuerySpannerExternalDatasetTable) []string {
	return []string{
		table.BigQueryTable,
		table.Name,
		binding.BigQueryDatasetRef.Dataset + "." + table.SpannerTable,
		binding.BigQueryDatasetRef.Path + "." + table.SpannerTable,
	}
}

func normalizeSQLForTableReference(sql string) string {
	var b strings.Builder
	for i := 0; i < len(sql); {
		switch {
		case sql[i] == '\'' || sql[i] == '"':
			b.WriteByte(' ')
			i = skipSQLQuotedString(sql, i, sql[i])
		case sql[i] == '`':
			i++
			for i < len(sql) {
				if sql[i] == '`' {
					i++
					break
				}
				b.WriteByte(sql[i])
				i++
			}
		case strings.HasPrefix(sql[i:], "--"):
			b.WriteByte(' ')
			i = skipSQLLineComment(sql, i)
		case strings.HasPrefix(sql[i:], "/*"):
			b.WriteByte(' ')
			i = skipSQLBlockComment(sql, i)
		default:
			b.WriteByte(sql[i])
			i++
		}
	}
	sql = strings.ToLower(b.String())
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func planColumnCapabilities(columns []*Column) []QueryCodegenPlanColumnCapability {
	out := make([]QueryCodegenPlanColumnCapability, 0, len(columns))
	for _, column := range columns {
		out = append(out, QueryCodegenPlanColumnCapability{
			Name:                 column.Name,
			Nullable:             !column.NotNull && !column.PrimaryKey,
			PrimaryKey:           column.PrimaryKey,
			Hidden:               column.Hidden,
			InsertValue:          columnInsertable(column),
			UpdateValue:          columnUpdatable(column),
			DefaultSQL:           column.DefaultSQL,
			GeneratedSQL:         column.GeneratedSQL,
			OnUpdateSQL:          column.OnUpdateSQL,
			AllowCommitTimestamp: column.AllowCommitTimestamp,
		})
	}
	return out
}

func planServerSideUpdateEffects(spec resolvedWriteSpec) []QueryCodegenPlanServerSideEffect {
	if spec.Operation != "update" && spec.Operation != "insert_or_update" {
		return nil
	}
	if len(spec.UpdateColumns) == 0 {
		return nil
	}
	out := []QueryCodegenPlanServerSideEffect{}
	for _, column := range spec.AllColumns {
		if column.OnUpdateSQL == "" || containsColumn(spec.UpdateColumns, column.Name) {
			continue
		}
		out = append(out, QueryCodegenPlanServerSideEffect{
			Column: column.Name,
			Reason: "on_update_expression_triggered_by_non_key_update",
		})
	}
	return out
}

func queryKind(query QueryCodegenQuery) string {
	if strings.TrimSpace(query.Kind) != "" {
		return strings.ToLower(strings.TrimSpace(query.Kind))
	}
	switch {
	case query.Table != "":
		return "table"
	case query.Index != "":
		return "index"
	case !query.Federated.isZero():
		return "external_query"
	case strings.TrimSpace(query.SQL) != "":
		return "sql"
	default:
		return ""
	}
}

func queryResultMode(query QueryCodegenQuery) string {
	return emptyDefault(strings.ToLower(query.Result), "many")
}

func validateQueryCodegenQuery(query QueryCodegenQuery) error {
	switch queryResultMode(query) {
	case "one", "maybe_one", "many", "row_count", "row_set":
	default:
		return fmt.Errorf("query %s: unsupported result %q", query.Name, query.Result)
	}
	if err := validateVetConfig("query "+query.Name, query.Vet); err != nil {
		return err
	}
	return nil
}

func validateVetConfig(scope string, config QueryCodegenVetConfig) error {
	for i, disabled := range config.Disable {
		if disabled.Rule == "" {
			return fmt.Errorf("%s vet.disable[%d]: rule is required", scope, i)
		}
		if strings.TrimSpace(disabled.Reason) == "" {
			return fmt.Errorf("%s vet.disable[%d] rule %q: reason is required", scope, i, disabled.Rule)
		}
		if disabled.Expires != "" {
			if _, err := time.Parse("2006-01-02", disabled.Expires); err != nil {
				return fmt.Errorf("%s vet.disable[%d] rule %q: expires must use YYYY-MM-DD format", scope, i, disabled.Rule)
			}
		}
	}
	return nil
}

func mergeQueryCodegenParams(defaults, overrides []QueryCodegenParam) ([]QueryCodegenParam, error) {
	out := append([]QueryCodegenParam(nil), defaults...)
	index := map[string]int{}
	for i, param := range out {
		key := strings.ToLower(param.Name)
		if key == "" {
			return nil, fmt.Errorf("param name is required")
		}
		index[key] = i
	}
	seenOverrides := map[string]bool{}
	for _, param := range overrides {
		key := strings.ToLower(param.Name)
		if key == "" {
			return nil, fmt.Errorf("param name is required")
		}
		if seenOverrides[key] {
			return nil, fmt.Errorf("duplicate param %q", param.Name)
		}
		seenOverrides[key] = true
		if i, ok := index[key]; ok {
			// Override keeps user-specified fields but falls back to
			// the default-generated entry for unset fields. This lets
			// a user attach optional / choices / default to an
			// auto-generated kind: index / kind: table param without
			// also having to repeat the type.
			merged := param
			if strings.TrimSpace(merged.Type) == "" {
				merged.Type = out[i].Type
			}
			if strings.TrimSpace(merged.Scope) == "" {
				merged.Scope = out[i].Scope
			}
			out[i] = merged
			continue
		}
		index[key] = len(out)
		out = append(out, param)
	}
	return out, nil
}

func validateQueryCodegenParams(query QueryCodegenQuery) error {
	seen := map[string]bool{}
	seenUnscoped := map[string]bool{}
	seenScoped := map[string]bool{}
	for _, param := range query.Params {
		if param.Name == "" {
			return fmt.Errorf("query %s: param name is required", query.Name)
		}
		scope := strings.ToLower(strings.TrimSpace(param.Scope))
		switch scope {
		case "", "inner", "outer":
		default:
			return fmt.Errorf("query %s param %s: unsupported scope %q; use inner or outer", query.Name, param.Name, param.Scope)
		}
		if scope != "" && query.Federated.isZero() {
			return fmt.Errorf("query %s param %s: scope is only valid for kind: external_query", query.Name, param.Name)
		}
		nameKey := strings.ToLower(param.Name)
		key := scope + ":" + nameKey
		if seen[key] {
			return fmt.Errorf("query %s: duplicate param %q in scope %q", query.Name, param.Name, emptyDefault(scope, "default"))
		}
		seen[key] = true
		if scope == "" {
			if seenScoped[nameKey] {
				return fmt.Errorf("query %s param %s: scope is required because scoped and unscoped entries share the same name", query.Name, param.Name)
			}
			if seenUnscoped[nameKey] {
				return fmt.Errorf("query %s: duplicate unscoped param %q", query.Name, param.Name)
			}
			seenUnscoped[nameKey] = true
		} else {
			if seenUnscoped[nameKey] {
				return fmt.Errorf("query %s param %s: scope is required because scoped and unscoped entries share the same name", query.Name, param.Name)
			}
			seenScoped[nameKey] = true
		}
		if err := validateQueryCodegenOptionalParam(query.Name, param); err != nil {
			return err
		}
	}
	return nil
}

func validateQueryCodegenOptionalParam(queryName string, param QueryCodegenParam) error {
	mode := strings.ToLower(strings.TrimSpace(param.Optional))
	switch mode {
	case "", "required":
		if len(param.Choices) > 0 {
			return fmt.Errorf("query %s param %s: choices is only valid for optional: orderby_choice", queryName, param.Name)
		}
		if param.Default != "" {
			return fmt.Errorf("query %s param %s: default is only valid for optional: orderby_choice", queryName, param.Name)
		}
		return nil
	case "null_is_null", "omit_when_null", "omit_when_empty":
		if len(param.Choices) > 0 {
			return fmt.Errorf("query %s param %s: choices is only valid for optional: orderby_choice", queryName, param.Name)
		}
		if param.Default != "" {
			return fmt.Errorf("query %s param %s: default is only valid for optional: orderby_choice", queryName, param.Name)
		}
		return nil
	case "orderby_choice":
		if len(param.Choices) == 0 {
			return fmt.Errorf("query %s param %s: optional: orderby_choice requires choices", queryName, param.Name)
		}
		if param.Default == "" {
			return fmt.Errorf("query %s param %s: optional: orderby_choice requires default", queryName, param.Name)
		}
		for key := range param.Choices {
			if !optionalChoiceKeyPattern.MatchString(key) {
				return fmt.Errorf("query %s param %s: choice key %q must match ^[A-Za-z_][A-Za-z0-9_]*$", queryName, param.Name, key)
			}
		}
		if !optionalChoiceKeyPattern.MatchString(param.Default) {
			return fmt.Errorf("query %s param %s: default %q must match ^[A-Za-z_][A-Za-z0-9_]*$", queryName, param.Name, param.Default)
		}
		if _, ok := param.Choices[param.Default]; !ok {
			return fmt.Errorf("query %s param %s: default %q must be one of choices", queryName, param.Name, param.Default)
		}
		return nil
	default:
		return fmt.Errorf("query %s param %s: unsupported optional %q; use required, null_is_null, omit_when_null, omit_when_empty, or orderby_choice", queryName, param.Name, param.Optional)
	}
}

func addSpannerQueryParameter(analyzer *Analyzer, queryName string, param QueryCodegenParam) error {
	if param.Name == "" {
		return fmt.Errorf("query %s: param name is required", queryName)
	}
	spec, err := ParseTypeSpec("param", param.Type)
	if err != nil {
		return fmt.Errorf("query %s param %s: %w", queryName, param.Name, err)
	}
	if err := analyzer.AddQueryParameter(param.Name, spec); err != nil {
		return fmt.Errorf("query %s param %s: %w", queryName, param.Name, err)
	}
	return nil
}

func addBigQueryQueryParameter(analyzer *BigQueryAnalyzer, queryName string, param QueryCodegenParam) error {
	if param.Name == "" {
		return fmt.Errorf("query %s: param name is required", queryName)
	}
	spec, err := ParseTypeSpec("param", param.Type)
	if err != nil {
		return fmt.Errorf("query %s param %s: %w", queryName, param.Name, err)
	}
	if err := analyzer.AddQueryParameter(param.Name, spec); err != nil {
		return fmt.Errorf("query %s param %s: %w", queryName, param.Name, err)
	}
	return nil
}

func addBigQueryScopedQueryParams(analyzer *BigQueryAnalyzer, externalAnalyzers map[string]*Analyzer, query QueryCodegenQuery) error {
	for _, param := range query.Params {
		scope, err := queryParamEffectiveScope(query, param)
		if err != nil {
			return err
		}
		switch scope {
		case "inner":
			innerAnalyzer := externalAnalyzers[query.Federated.Connection]
			if innerAnalyzer == nil {
				return fmt.Errorf("query %s param %s: no Spanner analyzer configured for external_query connection %q", query.Name, param.Name, query.Federated.Connection)
			}
			if err := addSpannerQueryParameter(innerAnalyzer, query.Name, param); err != nil {
				return err
			}
		case "outer":
			if err := addBigQueryQueryParameter(analyzer, query.Name, param); err != nil {
				return err
			}
		default:
			return fmt.Errorf("query %s param %s: unsupported effective scope %q", query.Name, param.Name, scope)
		}
	}
	return nil
}

func queryParamEffectiveScope(query QueryCodegenQuery, param QueryCodegenParam) (string, error) {
	scope := strings.ToLower(strings.TrimSpace(param.Scope))
	if query.Federated.isZero() {
		return "outer", nil
	}
	if scope != "" {
		return scope, nil
	}
	innerUses := sqlUsesNamedParameter(query.Federated.InnerSQL, param.Name)
	outerUses := sqlUsesNamedParameter(query.Federated.OuterSQL, param.Name)
	if innerUses && outerUses {
		return "", fmt.Errorf("query %s param %s: scope is required because both inner_sql and outer_sql reference the parameter", query.Name, param.Name)
	}
	if outerUses {
		return "outer", nil
	}
	return "inner", nil
}

func queryPlanWarnings(schemas map[string]QueryCodegenSchema, query QueryCodegenQuery, baseDir string) ([]QueryCodegenPlanWarning, error) {
	if query.Federated.isZero() {
		return nil, nil
	}
	sourceName, err := querySourceName(schemas, query)
	if err != nil {
		return nil, err
	}
	bigQuerySchema := schemas[sourceName]
	spannerSchemaName, err := resolveFederatedSpannerSchemaName(bigQuerySchema, query.Federated)
	if err != nil {
		return nil, fmt.Errorf("query %s external_query: %w", query.Name, err)
	}
	spannerSchema := schemas[spannerSchemaName]
	ddlPath, ddl, err := readCodegenDDL(spannerSchema, baseDir)
	if err != nil {
		return nil, fmt.Errorf("query %s external_query schema %s: %w", query.Name, spannerSchemaName, err)
	}
	protoPaths := resolveCodegenPaths(baseDir, spannerSchema.ProtoDescriptorFiles)
	analyzer, err := NewAnalyzerFromDDLWithProtoDescriptorFiles(ddlPath, ddl, protoPaths)
	if err != nil {
		return nil, fmt.Errorf("query %s external_query schema %s: %w", query.Name, spannerSchemaName, err)
	}
	for _, param := range query.Params {
		scope, err := queryParamEffectiveScope(query, param)
		if err != nil {
			return nil, err
		}
		if scope != "inner" {
			continue
		}
		if err := addSpannerQueryParameter(analyzer, query.Name, param); err != nil {
			return nil, err
		}
	}
	rowType, err := analyzer.RowTypeForStatement(query.Federated.InnerSQL)
	if err != nil {
		return nil, fmt.Errorf("query %s external_query inner_sql: %w", query.Name, err)
	}
	var warnings []QueryCodegenPlanWarning
	for _, field := range rowType.Fields {
		warnings = append(warnings, federatedTypeWarnings(field.Name, field.Type)...)
	}
	if federatedInnerOrderByIgnored(query.Federated) {
		warnings = append(warnings, QueryCodegenPlanWarning{
			Rule:        "external-query-inner-order-by-ignored",
			Severity:    "warning",
			Message:     "EXTERNAL_QUERY does not preserve inner query result ordering",
			Remediation: "Move ORDER BY to outer_sql if BigQuery result order matters. Inner ORDER BY can still be useful with LIMIT 1.",
		})
	}
	return warnings, nil
}

var (
	sqlOrderByPattern        = regexp.MustCompile(`(?is)\border\s+by\b`)
	sqlLimitOnePattern       = regexp.MustCompile(`(?is)\blimit\s+1\b`)
	optionalChoiceKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

func federatedInnerOrderByIgnored(query QueryCodegenFederatedQuery) bool {
	if !sqlOrderByPattern.MatchString(query.InnerSQL) {
		return false
	}
	if sqlLimitOnePattern.MatchString(query.InnerSQL) {
		return false
	}
	return !sqlOrderByPattern.MatchString(query.OuterSQL)
}

func federatedTypeWarnings(path string, typ *spannerpb.Type) []QueryCodegenPlanWarning {
	if typ == nil {
		return nil
	}
	switch typ.Code {
	case spannerpb.TypeCode_TIMESTAMP:
		return []QueryCodegenPlanWarning{{
			Rule:        "cross-dialect-timestamp-truncation",
			Severity:    "warning",
			Message:     fmt.Sprintf("field %s: BigQuery EXTERNAL_QUERY over Spanner can truncate TIMESTAMP nanoseconds", path),
			Remediation: "Cast or format the TIMESTAMP in inner_sql if nanosecond precision matters to generated consumers.",
		}}
	case spannerpb.TypeCode_ARRAY:
		return federatedTypeWarnings(path+"[]", typ.ArrayElementType)
	default:
		return nil
	}
}

func analyzeCodegenQuery(schemas map[string]QueryCodegenSchema, query QueryCodegenQuery, baseDir string) ([]goResultField, []QueryCodegenPlanQueryVariant, error) {
	if query.Name == "" {
		return nil, nil, fmt.Errorf("query name is required")
	}
	schemaName, err := querySourceName(schemas, query)
	if err != nil {
		return nil, nil, err
	}
	schema, ok := schemas[schemaName]
	if !ok {
		return nil, nil, fmt.Errorf("query %s: unknown schema %q", query.Name, schemaName)
	}
	ddlPath, ddl, err := readCodegenDDL(schema, baseDir)
	if err != nil {
		return nil, nil, fmt.Errorf("query %s schema %s: %w", query.Name, schemaName, err)
	}
	switch strings.ToLower(emptyDefault(schema.Dialect, "spanner")) {
	case "spanner":
		if queryHasOptionalMarkers(query) {
			if queryResultMode(query) == "row_count" {
				return nil, nil, fmt.Errorf("query %s: optional markers are not supported on DML row_count queries yet", query.Name)
			}
			fields, variants, err := analyzeCodegenQuerySpannerVariants(ddlPath, ddl, schema, query, baseDir)
			if err != nil {
				return nil, nil, fmt.Errorf("query %s: %w", query.Name, err)
			}
			return fields, variants, nil
		}
		protoPaths := resolveCodegenPaths(baseDir, schema.ProtoDescriptorFiles)
		analyzer, err := NewAnalyzerFromDDLWithProtoDescriptorFiles(ddlPath, ddl, protoPaths)
		if err != nil {
			return nil, nil, fmt.Errorf("query %s: %w", query.Name, err)
		}
		for _, param := range query.Params {
			if err := addSpannerQueryParameter(analyzer, query.Name, param); err != nil {
				return nil, nil, err
			}
		}
		if queryResultMode(query) == "row_count" {
			return nil, nil, nil
		}
		rowType, err := analyzer.RowTypeForStatement(query.SQL)
		if err != nil {
			return nil, nil, fmt.Errorf("query %s: %w", query.Name, err)
		}
		fields := make([]goResultField, 0, len(rowType.Fields))
		for _, field := range rowType.Fields {
			fields = append(fields, goResultFieldFromSpanner(field.Name, field.Type))
		}
		return fields, nil, nil
	case "bigquery":
		if queryHasOptionalMarkers(query) {
			return nil, nil, fmt.Errorf("query %s: optional markers are not supported on bigquery dialect yet", query.Name)
		}
		analyzer, err := NewBigQueryAnalyzerFromDDL(ddlPath, ddl)
		if err != nil {
			return nil, nil, fmt.Errorf("query %s: %w", query.Name, err)
		}
		if _, err := addCodegenSpannerExternalDatasets(analyzer, schemas, schema, baseDir); err != nil {
			return nil, nil, fmt.Errorf("query %s: %w", query.Name, err)
		}
		externalAnalyzers, err := buildCodegenExternalAnalyzers(schemas, schema, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("query %s: %w", query.Name, err)
		}
		if err := addBigQueryScopedQueryParams(analyzer, externalAnalyzers, query); err != nil {
			return nil, nil, err
		}
		analyzer.SetExternalQueryAnalyzers(externalAnalyzers)
		tableSchema, err := analyzer.TableSchemaForStatement(query.SQL)
		if err != nil {
			return nil, nil, fmt.Errorf("query %s: %w", query.Name, err)
		}
		fields := make([]goResultField, 0, len(tableSchema.Fields))
		for _, field := range tableSchema.Fields {
			fields = append(fields, goResultFieldFromBigQuery(field))
		}
		return fields, nil, nil
	default:
		return nil, nil, fmt.Errorf("query %s: unsupported schema dialect %q", query.Name, schema.Dialect)
	}
}

// findOrderByChoiceParam returns the first param with mode
// orderby_choice, or a zero-value param if none. Generated SQL for
// kind: index uses it to emit an /*?orderby:NAME*/ marker in place of
// the default-key ORDER BY clause.
func findOrderByChoiceParam(params []QueryCodegenParam) QueryCodegenParam {
	for _, p := range params {
		if strings.EqualFold(strings.TrimSpace(p.Optional), "orderby_choice") {
			return p
		}
	}
	return QueryCodegenParam{}
}

// queryOptionalModes returns a name -> optional-mode map for the params
// in a query. Used by codegenIndexQuerySQL to swap predicate generation
// when a key-prefix column carries a non-default optional mode.
func queryOptionalModes(params []QueryCodegenParam) map[string]string {
	if len(params) == 0 {
		return nil
	}
	out := make(map[string]string, len(params))
	for _, p := range params {
		mode := strings.ToLower(strings.TrimSpace(p.Optional))
		if mode == "" || mode == "required" {
			continue
		}
		out[p.Name] = mode
	}
	return out
}

// queryHasOptionalMarkers reports whether any param triggers the
// per-variant analyzer path.
func queryHasOptionalMarkers(query QueryCodegenQuery) bool {
	for _, p := range query.Params {
		switch strings.ToLower(strings.TrimSpace(p.Optional)) {
		case "null_is_null", "omit_when_null", "omit_when_empty", "orderby_choice":
			return true
		}
	}
	return false
}

func buildCodegenExternalAnalyzers(schemas map[string]QueryCodegenSchema, schema QueryCodegenSchema, baseDir string) (map[string]*Analyzer, error) {
	analyzers := map[string]*Analyzer{}
	for _, external := range queryCodegenExternalQueryConnections(schema) {
		if external.Connection == "" || external.SpannerSource == "" {
			return nil, fmt.Errorf("external query connection entries require connection and spanner_source")
		}
		spannerSchema, ok := schemas[external.SpannerSource]
		if !ok {
			return nil, fmt.Errorf("external schema %q does not exist", external.SpannerSource)
		}
		if strings.ToLower(emptyDefault(spannerSchema.Dialect, "spanner")) != "spanner" {
			return nil, fmt.Errorf("external schema %q must use spanner dialect", external.SpannerSource)
		}
		ddlPath, ddl, err := readCodegenDDL(spannerSchema, baseDir)
		if err != nil {
			return nil, fmt.Errorf("external schema %q: %w", external.SpannerSource, err)
		}
		protoPaths := resolveCodegenPaths(baseDir, spannerSchema.ProtoDescriptorFiles)
		analyzer, err := NewAnalyzerFromDDLWithProtoDescriptorFiles(ddlPath, ddl, protoPaths)
		if err != nil {
			return nil, fmt.Errorf("external schema %q: %w", external.SpannerSource, err)
		}
		analyzers[external.Connection] = analyzer
	}
	return analyzers, nil
}

func addCodegenSpannerExternalDatasets(analyzer *BigQueryAnalyzer, schemas map[string]QueryCodegenSchema, schema QueryCodegenSchema, baseDir string) ([]BigQuerySpannerExternalDatasetBinding, error) {
	bindings := make([]BigQuerySpannerExternalDatasetBinding, 0, len(schema.SpannerExternalDatasets))
	for _, external := range schema.SpannerExternalDatasets {
		if external.Dataset == "" || external.SpannerSource == "" {
			return nil, fmt.Errorf("spanner_external_datasets entries require dataset and spanner_source")
		}
		if err := validateVetConfig("external dataset "+external.Dataset, external.Vet); err != nil {
			return nil, err
		}
		if err := validateCodegenExternalDatasetVerification(external); err != nil {
			return nil, err
		}
		spannerSchema, ok := schemas[external.SpannerSource]
		if !ok {
			return nil, fmt.Errorf("external dataset Spanner source %q does not exist", external.SpannerSource)
		}
		if err := validateExternalDatasetSpannerSourceDialect(external.SpannerSource, spannerSchema); err != nil {
			return nil, err
		}
		catalog, err := codegenSpannerCatalog(spannerSchema, baseDir)
		if err != nil {
			return nil, fmt.Errorf("external dataset %s: %w", external.Dataset, err)
		}
		access := queryCodegenSpannerExternalDatasetAccess(external)
		verification := queryCodegenSpannerExternalDatasetVerification(external)
		cloudResourceConnection := queryCodegenSpannerExternalDatasetCloudResourceConnection(external)
		binding, err := analyzer.AddSpannerExternalDatasetWithOptions(external.Dataset, external.SpannerSource, catalog, BigQuerySpannerExternalDatasetOptions{
			Project:                 external.Project,
			DefaultProject:          schema.Project,
			ExternalSource:          external.ExternalSource,
			Location:                external.Location,
			Connection:              external.Connection,
			CloudResourceConnection: cloudResourceConnection,
			Access:                  access.Mode,
			DatabaseRole:            access.DatabaseRole,
			AccessVerificationHint:  access.VerificationHint,
			AccessVerification:      verification,
			UnsupportedColumns:      external.UnsupportedColumns,
			NamedSchemaPolicy:       external.NamedSchemaPolicy,
		})
		if err != nil {
			return nil, fmt.Errorf("external dataset %s: %w", external.Dataset, err)
		}
		binding.Name = schema.Name + "." + emptyDefault(external.Name, external.Dataset)
		binding.VetSuppressions = planVetSuppressions(external.Vet)
		bindings = append(bindings, *binding)
	}
	return bindings, nil
}

func validateExternalDatasetSpannerSourceDialect(sourceName string, schema QueryCodegenSchema) error {
	dialect := strings.ToLower(emptyDefault(schema.Dialect, "spanner"))
	switch dialect {
	case "spanner", "spanner_google_sql", "spanner-google-sql":
		return nil
	case "postgresql", "spanner_postgresql", "spanner-postgresql":
		return queryCodegenDiagnosticError("external-dataset-postgresql-dialect-unsupported", "external_dataset_projection", "schemas."+sourceName, fmt.Sprintf("external dataset Spanner source %q uses PostgreSQL dialect, which is not supported by this generator; external_source metadata cannot override schema dialect", sourceName))
	default:
		return fmt.Errorf("external dataset Spanner source %q must use Spanner GoogleSQL dialect, got %q", sourceName, schema.Dialect)
	}
}

func queryCodegenSpannerExternalDatasetAccess(external QueryCodegenSpannerExternalDataset) QueryCodegenSpannerExternalDatasetAccess {
	access := external.Access
	if access.DatabaseRole == "" {
		access.DatabaseRole = external.DatabaseRole
	}
	return access
}

func queryCodegenSpannerExternalDatasetCloudResourceConnection(external QueryCodegenSpannerExternalDataset) string {
	if external.Access.CloudResourceConnection != "" {
		return external.Access.CloudResourceConnection
	}
	if external.CloudResourceConnection != "" {
		return external.CloudResourceConnection
	}
	return external.Connection
}

func queryCodegenSpannerExternalDatasetVerification(external QueryCodegenSpannerExternalDataset) BigQuerySpannerExternalDatasetVerification {
	verification := external.Access.VerificationEvidence
	if isZeroQueryCodegenVerification(verification) {
		verification = external.AccessVerification
	}
	return BigQuerySpannerExternalDatasetVerification{
		Status:         verification.Status,
		Source:         verification.Source,
		CheckedAt:      verification.CheckedAt,
		Verifier:       verification.Verifier,
		EvidenceDigest: verification.EvidenceDigest,
	}
}

func validateCodegenExternalDatasetVerification(external QueryCodegenSpannerExternalDataset) error {
	hint := external.Access.VerificationHint
	if hint != "" && hint != "not_checked" {
		return queryCodegenDiagnosticError("external-dataset-access-verification-conflict", "external_dataset_binding", "spanner_external_datasets.access.verification_hint", fmt.Sprintf("external dataset %s access.verification_hint must be not_checked; use access.verification_evidence for verification evidence", external.Dataset))
	}
	if !isZeroQueryCodegenVerification(external.Access.VerificationEvidence) && !isZeroQueryCodegenVerification(external.AccessVerification) {
		return queryCodegenDiagnosticError("external-dataset-access-verification-conflict", "external_dataset_binding", "spanner_external_datasets.access.verification_evidence", fmt.Sprintf("external dataset %s: use either access.verification_evidence or legacy access_verification, not both", external.Dataset))
	}
	verification := external.Access.VerificationEvidence
	scope := "access.verification_evidence"
	if isZeroQueryCodegenVerification(verification) {
		verification = external.AccessVerification
		scope = "access_verification"
	}
	status := verification.Status
	if status == "" {
		return nil
	}
	if _, err := normalizeSpannerExternalDatasetVerificationStatus(status); err != nil {
		return err
	}
	if verification.Source != "" {
		if _, err := normalizeSpannerExternalDatasetVerificationSource(verification.Source, status); err != nil {
			return err
		}
	}
	if status == "verified" || status == "mismatch" || status == "failed" {
		if verification.Verifier == "" || verification.CheckedAt == "" {
			return fmt.Errorf("external dataset %s %s status %q requires verifier and checked_at", external.Dataset, scope, status)
		}
		if _, err := time.Parse(time.RFC3339, verification.CheckedAt); err != nil {
			return fmt.Errorf("external dataset %s %s checked_at must be RFC3339: %w", external.Dataset, scope, err)
		}
	}
	return nil
}

func isZeroQueryCodegenVerification(verification QueryCodegenSpannerExternalDatasetVerification) bool {
	return verification.Status == "" &&
		verification.Source == "" &&
		verification.CheckedAt == "" &&
		verification.Verifier == "" &&
		verification.EvidenceDigest == ""
}

func querySourceName(schemas map[string]QueryCodegenSchema, query QueryCodegenQuery) (string, error) {
	schemaName := query.Catalog
	if schemaName == "" {
		if _, ok := schemas["default"]; ok {
			schemaName = "default"
		} else if len(schemas) == 1 {
			for name := range schemas {
				schemaName = name
			}
		} else {
			return "", fmt.Errorf("query %s: catalog is required when multiple catalogs are configured", query.Name)
		}
	}
	if _, ok := schemas[schemaName]; !ok {
		return "", fmt.Errorf("query %s: unknown catalog %q", query.Name, schemaName)
	}
	return schemaName, nil
}

func resolveCodegenQuerySQL(schemas map[string]QueryCodegenSchema, query QueryCodegenQuery, baseDir string) (QueryCodegenQuery, error) {
	count := 0
	if strings.TrimSpace(query.SQL) != "" {
		count++
	}
	if query.Table != "" {
		count++
	}
	if query.Index != "" {
		count++
	}
	if !query.Federated.isZero() {
		count++
	}
	if count != 1 {
		return query, fmt.Errorf("query %s: exactly one of sql, table, index, or external_query is required", query.Name)
	}
	if strings.TrimSpace(query.SQL) != "" {
		if strings.TrimSpace(query.OrderBy) != "" {
			return query, fmt.Errorf("query %s: order_by is only supported for generated table and index queries", query.Name)
		}
		return query, nil
	}
	source, err := querySourceName(schemas, query)
	if err != nil {
		return query, err
	}
	schema, ok := schemas[source]
	if !ok {
		return query, fmt.Errorf("query %s: unknown source %q", query.Name, source)
	}
	switch {
	case query.Table != "":
		sql, params, err := codegenTableQuerySQL(schema, query.Table, query.KeyPrefix, query.OrderBy, baseDir, query.Params)
		if err != nil {
			return query, fmt.Errorf("query %s table %s: %w", query.Name, query.Table, err)
		}
		query.SQL = sql
		query.Params, err = mergeQueryCodegenParams(params, query.Params)
		if err != nil {
			return query, fmt.Errorf("query %s params: %w", query.Name, err)
		}
		return query, nil
	case query.Index != "":
		if strings.ToLower(emptyDefault(schema.Dialect, "spanner")) != "spanner" {
			return query, fmt.Errorf("query %s: index queries are only supported for Spanner schemas", query.Name)
		}
		sql, params, err := codegenIndexQuerySQL(schema, query.Index, query.KeyPrefix, query.OrderBy, baseDir, query.Params)
		if err != nil {
			return query, fmt.Errorf("query %s index %s: %w", query.Name, query.Index, err)
		}
		query.SQL = sql
		query.Params, err = mergeQueryCodegenParams(params, query.Params)
		if err != nil {
			return query, fmt.Errorf("query %s params: %w", query.Name, err)
		}
		return query, nil
	case !query.Federated.isZero():
		if strings.TrimSpace(query.OrderBy) != "" {
			return query, fmt.Errorf("query %s: order_by is only supported for generated table and index queries", query.Name)
		}
		sql, err := codegenFederatedQuerySQL(schemas, schema, query.Federated)
		if err != nil {
			return query, fmt.Errorf("query %s external_query: %w", query.Name, err)
		}
		query.SQL = sql
		return query, nil
	default:
		return query, nil
	}
}

func (q QueryCodegenFederatedQuery) isZero() bool {
	return q.Connection == "" && q.SpannerSource == "" && strings.TrimSpace(q.InnerSQL) == "" && strings.TrimSpace(q.OuterSQL) == ""
}

func queryPlanOrderByMode(schema QueryCodegenSchema, query QueryCodegenQuery) string {
	switch queryKind(query) {
	case "table":
		mode, err := normalizeTableQueryOrderBy(schema, query.OrderBy)
		if err != nil || mode == "none" {
			return ""
		}
		return mode
	case "index":
		mode, err := normalizeIndexQueryOrderBy(query.OrderBy)
		if err != nil || mode == "none" {
			return ""
		}
		return mode
	default:
		return ""
	}
}

func codegenFederatedQuerySQL(schemas map[string]QueryCodegenSchema, bigQuerySchema QueryCodegenSchema, query QueryCodegenFederatedQuery) (string, error) {
	if strings.ToLower(emptyDefault(bigQuerySchema.Dialect, "spanner")) != "bigquery" {
		return "", fmt.Errorf("external_query requires a BigQuery source")
	}
	if query.Connection == "" {
		return "", fmt.Errorf("connection is required")
	}
	if strings.TrimSpace(query.InnerSQL) == "" {
		return "", fmt.Errorf("inner_sql is required")
	}
	externalSchemaName, err := resolveFederatedSpannerSchemaName(bigQuerySchema, query)
	if err != nil {
		return "", err
	}
	spannerSchema, ok := schemas[externalSchemaName]
	if !ok {
		return "", fmt.Errorf("external schema %q does not exist", externalSchemaName)
	}
	if strings.ToLower(emptyDefault(spannerSchema.Dialect, "spanner")) != "spanner" {
		return "", fmt.Errorf("external schema %q must use spanner dialect", externalSchemaName)
	}
	externalQuerySQL := "EXTERNAL_QUERY(" + googleSQLStringLiteral(query.Connection) + ", " + googleSQLStringLiteral(query.InnerSQL) + ")"
	outerSQL := strings.TrimSpace(query.OuterSQL)
	if outerSQL == "" {
		return "SELECT * FROM " + externalQuerySQL, nil
	}
	occurrences := sqlIdentifierTokenOccurrences(outerSQL, "__external__")
	if count := len(occurrences); count != 1 {
		id := "external-query-placeholder-duplicate"
		if count == 0 {
			id = "external-query-placeholder-missing"
		}
		return "", queryCodegenDiagnosticError(id, "external_query_analysis", "external_query.outer_sql", fmt.Sprintf("outer_sql must contain exactly one __external__ placeholder token, got %d", count))
	}
	occurrence := occurrences[0]
	if !sqlExternalPlaceholderIsTableExpression(outerSQL, occurrence.start) {
		return "", queryCodegenDiagnosticError("external-query-placeholder-invalid-position", "external_query_analysis", "external_query.outer_sql", "external_query.outer_sql placeholder must appear where a table expression is valid; use FROM __external__ or JOIN __external__")
	}
	return outerSQL[:occurrence.start] + externalQuerySQL + outerSQL[occurrence.end:], nil
}

func resolveFederatedSpannerSchemaName(bigQuerySchema QueryCodegenSchema, query QueryCodegenFederatedQuery) (string, error) {
	externalSchemaName := ""
	for _, external := range queryCodegenExternalQueryConnections(bigQuerySchema) {
		if external.Connection == query.Connection {
			externalSchemaName = external.SpannerSource
			break
		}
	}
	if externalSchemaName == "" {
		return "", fmt.Errorf("no external query connection entry for connection %q", query.Connection)
	}
	if query.SpannerSource != "" && query.SpannerSource != externalSchemaName {
		return "", fmt.Errorf("spanner_source %q does not match external query connection mapping %q for connection %q", query.SpannerSource, externalSchemaName, query.Connection)
	}
	return externalSchemaName, nil
}

type sqlTokenOccurrence struct {
	start int
	end   int
}

func sqlIdentifierTokenOccurrences(sql, token string) []sqlTokenOccurrence {
	var out []sqlTokenOccurrence
	for i := 0; i < len(sql); {
		switch {
		case sql[i] == '\'' || sql[i] == '"':
			i = skipSQLQuotedString(sql, i, sql[i])
		case sql[i] == '`':
			i = skipSQLQuotedIdentifier(sql, i)
		case strings.HasPrefix(sql[i:], "--"):
			i = skipSQLLineComment(sql, i)
		case strings.HasPrefix(sql[i:], "/*"):
			i = skipSQLBlockComment(sql, i)
		case isSQLIdentifierStart(sql[i]):
			start := i
			i++
			for i < len(sql) && isSQLIdentifierPart(sql[i]) {
				i++
			}
			if strings.EqualFold(sql[start:i], token) {
				out = append(out, sqlTokenOccurrence{start: start, end: i})
			}
		default:
			i++
		}
	}
	return out
}

func sqlUsesNamedParameter(sql, name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	for i := 0; i < len(sql); {
		switch {
		case sql[i] == '\'' || sql[i] == '"':
			i = skipSQLQuotedString(sql, i, sql[i])
		case sql[i] == '`':
			i = skipSQLQuotedIdentifier(sql, i)
		case strings.HasPrefix(sql[i:], "--"):
			i = skipSQLLineComment(sql, i)
		case strings.HasPrefix(sql[i:], "/*"):
			i = skipSQLBlockComment(sql, i)
		case sql[i] == '@':
			start := i + 1
			if start < len(sql) && isSQLIdentifierStart(sql[start]) {
				i = start + 1
				for i < len(sql) && isSQLIdentifierPart(sql[i]) {
					i++
				}
				if strings.EqualFold(sql[start:i], name) {
					return true
				}
				continue
			}
			i++
		default:
			i++
		}
	}
	return false
}

func sqlExternalPlaceholderIsTableExpression(sql string, placeholderStart int) bool {
	previous := sqlPreviousIdentifierToken(sql, placeholderStart)
	return strings.EqualFold(previous, "from") || strings.EqualFold(previous, "join")
}

func sqlPreviousIdentifierToken(sql string, before int) string {
	previous := ""
	for i := 0; i < len(sql) && i < before; {
		switch {
		case sql[i] == '\'' || sql[i] == '"':
			i = skipSQLQuotedString(sql, i, sql[i])
		case sql[i] == '`':
			i = skipSQLQuotedIdentifier(sql, i)
		case strings.HasPrefix(sql[i:], "--"):
			i = skipSQLLineComment(sql, i)
		case strings.HasPrefix(sql[i:], "/*"):
			i = skipSQLBlockComment(sql, i)
		case isSQLIdentifierStart(sql[i]):
			start := i
			i++
			for i < len(sql) && isSQLIdentifierPart(sql[i]) {
				i++
			}
			if i <= before {
				previous = sql[start:i]
			}
		default:
			i++
		}
	}
	return previous
}

func sqlHasStarToken(sql string) bool {
	for i := 0; i < len(sql); {
		switch {
		case sql[i] == '\'' || sql[i] == '"':
			i = skipSQLQuotedString(sql, i, sql[i])
		case sql[i] == '`':
			i = skipSQLQuotedIdentifier(sql, i)
		case strings.HasPrefix(sql[i:], "--"):
			i = skipSQLLineComment(sql, i)
		case strings.HasPrefix(sql[i:], "/*"):
			i = skipSQLBlockComment(sql, i)
		case sql[i] == '*':
			return true
		default:
			i++
		}
	}
	return false
}

func skipSQLQuotedString(sql string, start int, quote byte) int {
	i := start + 1
	for i < len(sql) {
		if sql[i] == quote {
			if i+1 < len(sql) && sql[i+1] == quote {
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return len(sql)
}

func skipSQLQuotedIdentifier(sql string, start int) int {
	i := start + 1
	for i < len(sql) {
		if sql[i] == '`' {
			return i + 1
		}
		i++
	}
	return len(sql)
}

func skipSQLLineComment(sql string, start int) int {
	i := start + 2
	for i < len(sql) && sql[i] != '\n' {
		i++
	}
	return i
}

func skipSQLBlockComment(sql string, start int) int {
	i := start + 2
	for i+1 < len(sql) {
		if sql[i] == '*' && sql[i+1] == '/' {
			return i + 2
		}
		i++
	}
	return len(sql)
}

func isSQLIdentifierStart(ch byte) bool {
	return ch == '_' || ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z')
}

func isSQLIdentifierPart(ch byte) bool {
	return isSQLIdentifierStart(ch) || ('0' <= ch && ch <= '9')
}

func readCodegenDDL(schema QueryCodegenSchema, baseDir string) (string, string, error) {
	if schema.DDL == "" {
		return "schema.sql", "", nil
	}
	path := resolveCodegenPath(baseDir, schema.DDL)
	ddl, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	return path, string(ddl), nil
}

func generateWriteCode(schemas map[string]QueryCodegenSchema, writes []QueryCodegenWrite, baseDir string, sharedStructFields map[string][]goResultField) ([]string, string, error) {
	if len(writes) == 0 {
		return nil, "", nil
	}
	writeGen := &goStructGenerator{
		target:  GoStructTargetSpanner,
		imports: map[string]string{},
		used:    map[string]bool{},
	}
	writeStructFields, writeSpecs, err := planWriteSpecs(schemas, writes, baseDir, sharedStructFields)
	if err != nil {
		return nil, "", err
	}
	structNames := make([]string, 0, len(writeStructFields))
	for name := range writeStructFields {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)
	var b bytes.Buffer
	for i, name := range structNames {
		if i > 0 {
			b.WriteByte('\n')
		}
		writeGeneratedStruct(&b, writeGen.buildStruct(name, writeStructFields[name]))
	}
	for _, nested := range writeGen.structs {
		b.WriteByte('\n')
		writeGeneratedStruct(&b, nested)
	}
	for _, spec := range writeSpecs {
		b.WriteByte('\n')
		writeWriteMethods(&b, spec)
	}
	imports := []string{"cloud.google.com/go/spanner"}
	for path := range writeGen.imports {
		imports = append(imports, path)
	}
	sort.Strings(imports)
	return imports, b.String(), nil
}

func planWriteSpecs(schemas map[string]QueryCodegenSchema, writes []QueryCodegenWrite, baseDir string, sharedStructFields map[string][]goResultField) (map[string][]goResultField, []resolvedWriteSpec, error) {
	writeStructFields := map[string][]goResultField{}
	writeSpecs := make([]resolvedWriteSpec, 0, len(writes))
	for _, write := range writes {
		spec, err := resolveWriteSpec(schemas, write, baseDir)
		if err != nil {
			return nil, nil, err
		}
		structFields := writeStructFields
		if existing, ok := sharedStructFields[spec.InputStruct]; ok {
			if err := validateSharedWriteInputFields(spec, existing); err != nil {
				return nil, nil, err
			}
			structFields = sharedStructFields
		}
		merged, err := mergeGoResultFields(structFields[spec.InputStruct], spec.Fields)
		if err != nil {
			return nil, nil, fmt.Errorf("write %s input_struct %s: %w", write.Name, spec.InputStruct, err)
		}
		structFields[spec.InputStruct] = merged
		writeSpecs = append(writeSpecs, spec)
	}
	writeSpecs, err := attachWriteSpecNames(writeSpecs, writeStructFields, sharedStructFields)
	if err != nil {
		return nil, nil, err
	}
	return writeStructFields, writeSpecs, nil
}

func validateSharedWriteInputFields(spec resolvedWriteSpec, fields []goResultField) error {
	fieldNames := map[string]bool{}
	for _, field := range fields {
		fieldNames[columnKey(field.Name)] = true
	}
	for _, column := range spec.InputColumns {
		if !fieldNames[columnKey(column.Name)] {
			return fmt.Errorf("write %s input_struct %s is shared with query results but has no field for column %s", spec.Name, spec.InputStruct, column.Name)
		}
	}
	return nil
}

type resolvedWriteSpec struct {
	Name            string
	MethodPrefix    string
	Catalog         string
	Table           string
	Operation       string
	InputStruct     string
	AllColumns      []*Column
	InsertColumns   []*Column
	UpdateColumns   []*Column
	InputColumns    []*Column
	Keys            []*Column
	Fields          []goResultField
	Methods         []string
	VetSuppressions []QueryCodegenPlanVetSuppression
	FieldNames      map[string]string
	ParamNames      map[string]string
}

func resolveWriteSpec(schemas map[string]QueryCodegenSchema, write QueryCodegenWrite, baseDir string) (resolvedWriteSpec, error) {
	if write.Name == "" {
		return resolvedWriteSpec{}, fmt.Errorf("write name is required")
	}
	if err := validateVetConfig("write "+write.Name, write.Vet); err != nil {
		return resolvedWriteSpec{}, err
	}
	if write.Table == "" {
		return resolvedWriteSpec{}, fmt.Errorf("write %s: table is required", write.Name)
	}
	operation := strings.ToLower(emptyDefault(write.Operation, "insert_or_update"))
	if err := validateWriteOperation(operation); err != nil {
		return resolvedWriteSpec{}, fmt.Errorf("write %s: %w", write.Name, err)
	}
	sourceName := write.Catalog
	if sourceName == "" && len(schemas) == 1 {
		for name := range schemas {
			sourceName = name
		}
	}
	schema, ok := schemas[sourceName]
	if !ok {
		return resolvedWriteSpec{}, fmt.Errorf("write %s: unknown source %q", write.Name, sourceName)
	}
	if strings.ToLower(emptyDefault(schema.Dialect, "spanner")) != "spanner" {
		return resolvedWriteSpec{}, fmt.Errorf("write %s: writes are only supported for Spanner schemas", write.Name)
	}
	catalog, err := codegenSpannerCatalog(schema, baseDir)
	if err != nil {
		return resolvedWriteSpec{}, fmt.Errorf("write %s: %w", write.Name, err)
	}
	table, err := catalogTable(catalog, write.Table)
	if err != nil {
		return resolvedWriteSpec{}, fmt.Errorf("write %s: %w", write.Name, err)
	}
	primaryKeyNames := primaryKeyColumnNames(table)
	keyColumns, err := resolveWriteColumns(table, write.Keys, primaryKeyNames)
	if err != nil {
		return resolvedWriteSpec{}, fmt.Errorf("write %s keys: %w", write.Name, err)
	}
	if err := validateWriteKeysArePrimaryKeySet(write, keyColumns, primaryKeyNames); err != nil {
		return resolvedWriteSpec{}, fmt.Errorf("write %s keys: %w", write.Name, err)
	}

	if (operation == "update" || operation == "delete") && len(write.Insert.Columns) > 0 {
		return resolvedWriteSpec{}, fmt.Errorf("write %s: columns is only valid for insert and replace", write.Name)
	}

	updateColumnsRaw := write.Update.Columns
	if operation == "update" && len(updateColumnsRaw) == 0 {
		return resolvedWriteSpec{}, fmt.Errorf("write %s: update.columns is required for operation update", write.Name)
	}
	if len(updateColumnsRaw) == 1 && updateColumnsRaw[0] == autoAllNonKeyColumns {
		updateColumnsRaw = nonKeyColumnNames(selectableColumnNames(table), keyColumns)
	}

	insertColumns, err := resolveWriteColumns(table, write.Insert.Columns, primaryKeyNames)
	if err != nil {
		return resolvedWriteSpec{}, fmt.Errorf("write %s insert columns: %w", write.Name, err)
	}
	updateColumns, err := resolveWriteColumns(table, updateColumnsRaw, nil)
	if err != nil {
		return resolvedWriteSpec{}, fmt.Errorf("write %s update columns: %w", write.Name, err)
	}
	if operation == "insert_or_update" && len(write.Insert.Columns) == 0 {
		insertColumns = appendMissingColumns(keyColumns, updateColumns)
	}
	if err := validateResolvedWriteColumns(table, operation, insertColumns, updateColumns, keyColumns); err != nil {
		return resolvedWriteSpec{}, fmt.Errorf("write %s columns: %w", write.Name, err)
	}
	inputColumns := insertColumns
	if operation == "update" {
		inputColumns = updateColumns
	}
	if operation == "update" || operation == "insert_or_update" || operation == "delete" {
		inputColumns = appendMissingColumns(inputColumns, keyColumns)
	}
	fields := make([]goResultField, 0, len(inputColumns))
	for _, column := range inputColumns {
		field, err := goResultFieldFromColumn(column)
		if err != nil {
			return resolvedWriteSpec{}, fmt.Errorf("write %s column %s: %w", write.Name, column.Name, err)
		}
		fields = append(fields, field)
	}
	methods, err := resolveWriteMethods(operation, write.Methods, write.EmitExplicit)
	if err != nil {
		return resolvedWriteSpec{}, fmt.Errorf("write %s: %w", write.Name, err)
	}
	return resolvedWriteSpec{
		Name:            write.Name,
		MethodPrefix:    exportedIdentifier(write.Name, "Write"),
		Catalog:         sourceName,
		Table:           table.Name.String(),
		Operation:       operation,
		InputStruct:     writeInputStructName(write),
		AllColumns:      append([]*Column(nil), table.Columns...),
		InsertColumns:   insertColumns,
		UpdateColumns:   updateColumns,
		InputColumns:    inputColumns,
		Keys:            keyColumns,
		Fields:          fields,
		Methods:         methods,
		VetSuppressions: planVetSuppressions(write.Vet),
	}, nil
}

func validateWriteOperation(operation string) error {
	switch operation {
	case "insert", "update", "insert_or_update", "replace", "delete":
		return nil
	default:
		return fmt.Errorf("unsupported operation %q", operation)
	}
}

func validateWriteKeysArePrimaryKeySet(write QueryCodegenWrite, keys []*Column, primaryKeyNames []string) error {
	if len(write.Keys) == 0 {
		return nil
	}
	keyNames := columnNamesFromColumns(keys)
	if !sameNameSet(keyNames, primaryKeyNames) {
		return fmt.Errorf("key must equal the table primary key set; got [%s], want [%s]", strings.Join(keyNames, ", "), strings.Join(primaryKeyNames, ", "))
	}
	return nil
}

func resolveWriteMethods(operation string, methods []string, explicit bool) ([]string, error) {
	if len(methods) == 0 {
		if explicit {
			return nil, nil
		}
		if operation == "replace" {
			return []string{"mutation"}, nil
		}
		return []string{"mutation", "dml"}, nil
	}
	out := make([]string, 0, len(methods))
	seen := map[string]bool{}
	for _, method := range methods {
		method = strings.ToLower(method)
		switch method {
		case "mutation":
		case "dml":
		default:
			return nil, fmt.Errorf("unsupported write method %q", method)
		}
		if operation == "replace" && method == "dml" {
			return nil, fmt.Errorf("operation replace does not support dml helpers; use mutation")
		}
		if seen[method] {
			return nil, fmt.Errorf("duplicate write method %q", method)
		}
		seen[method] = true
		out = append(out, method)
	}
	return out, nil
}

func attachWriteSpecNames(specs []resolvedWriteSpec, writeStructFields, sharedStructFields map[string][]goResultField) ([]resolvedWriteSpec, error) {
	fieldNamesByStruct := map[string]map[string]string{}
	for name, fields := range writeStructFields {
		fieldNamesByStruct[name] = goFieldNameMap(fields)
	}
	for _, spec := range specs {
		if _, ok := fieldNamesByStruct[spec.InputStruct]; ok {
			continue
		}
		fields, ok := sharedStructFields[spec.InputStruct]
		if !ok {
			return nil, fmt.Errorf("write %s input_struct %s was not planned", spec.Name, spec.InputStruct)
		}
		fieldNamesByStruct[spec.InputStruct] = goFieldNameMap(fields)
	}
	usedSymbols := map[string]string{}
	out := make([]resolvedWriteSpec, len(specs))
	copy(out, specs)
	for i := range out {
		spec := &out[i]
		spec.FieldNames = fieldNamesByStruct[spec.InputStruct]
		for _, column := range spec.InputColumns {
			if spec.FieldNames[columnKey(column.Name)] == "" {
				return nil, fmt.Errorf("write %s input_struct %s has no field for column %s", spec.Name, spec.InputStruct, column.Name)
			}
		}
		spec.ParamNames = writeParamNameMap(spec.InputColumns)
		for _, symbol := range writeSymbols(*spec) {
			if previous := usedSymbols[symbol]; previous != "" {
				return nil, fmt.Errorf("write %s generates duplicate symbol %s already used by write %s", spec.Name, symbol, previous)
			}
			usedSymbols[symbol] = spec.Name
		}
	}
	return out, nil
}

func writeSymbols(spec resolvedWriteSpec) []string {
	out := make([]string, 0, len(spec.Methods)*2)
	for _, method := range spec.Methods {
		switch method {
		case "mutation":
			out = append(out, spec.MethodPrefix+"Mutation")
		case "dml":
			out = append(out, spec.MethodPrefix+"DML", spec.MethodPrefix+"DMLStatement")
		}
	}
	return out
}

func writeParamNameMap(columns []*Column) map[string]string {
	used := map[string]bool{}
	out := map[string]string{}
	for _, column := range columns {
		out[columnKey(column.Name)] = uniqueIdentifier(safeQueryParamName(column.Name), used)
	}
	return out
}

func goFieldNameMap(fields []goResultField) map[string]string {
	used := map[string]bool{}
	out := map[string]string{}
	for i, field := range fields {
		name := exportedIdentifier(field.Name, fmt.Sprintf("Field%d", i+1))
		out[columnKey(field.Name)] = uniqueIdentifier(name, used)
	}
	return out
}

func goResultFieldFromColumn(column *Column) (goResultField, error) {
	typ, err := column.Type.SpannerPB()
	if err != nil {
		return goResultField{}, err
	}
	field := goResultFieldFromSpanner(column.Name, typ)
	field.Nullable = !column.NotNull && !column.PrimaryKey
	return field, nil
}

func columnInsertable(column *Column) bool {
	return !column.Hidden && column.GeneratedSQL == ""
}

func columnUpdatable(column *Column) bool {
	return columnInsertable(column) && !column.PrimaryKey
}

func writeWriteMethods(b *bytes.Buffer, spec resolvedWriteSpec) {
	for _, method := range spec.Methods {
		switch method {
		case "mutation":
			writeMutationMethod(b, spec)
		case "dml":
			writeDMLMethod(b, spec)
		}
	}
}

func writeMutationMethod(b *bytes.Buffer, spec resolvedWriteSpec) {
	switch spec.Operation {
	case "delete":
		writeMutationMethodComment(b, spec)
		fmt.Fprintf(b, "func (w *%s) %sMutation() *spanner.Mutation {\n", spec.InputStruct, spec.MethodPrefix)
		fmt.Fprintf(b, "\treturn spanner.Delete(%q, spanner.Key{%s})\n", spec.Table, writeFieldAccessList(spec, spec.Keys))
		b.WriteString("}\n")
	case "insert":
		writeStructMutationMethod(b, spec, "Insert", spec.InsertColumns)
	case "update":
		writeStructMutationMethod(b, spec, "Update", appendMissingColumns(spec.Keys, spec.UpdateColumns))
	case "insert_or_update":
		writeStructMutationMethod(b, spec, "InsertOrUpdate", spec.InsertColumns)
	case "replace":
		writeStructMutationMethod(b, spec, "Replace", spec.InsertColumns)
	}
}

func writeStructMutationMethod(b *bytes.Buffer, spec resolvedWriteSpec, spannerFunc string, columns []*Column) {
	writeMutationMethodComment(b, spec)
	fmt.Fprintf(b, "func (w *%s) %sMutation() (*spanner.Mutation, error) {\n", spec.InputStruct, spec.MethodPrefix)
	fmt.Fprintf(b, "\treturn spanner.%s(%q, []string{%s}, []interface{}{%s}), nil\n", spannerFunc, spec.Table, writeStringList(columnNamesFromColumns(columns)), writeFieldAccessList(spec, columns))
	b.WriteString("}\n")
}

func writeMutationMethodComment(b *bytes.Buffer, spec resolvedWriteSpec) {
	fmt.Fprintf(b, "// %sMutation returns a Cloud Spanner mutation.\n", spec.MethodPrefix)
	b.WriteString("// Prefer either mutations or DML statements within one transaction unless you intentionally account for their execution order.\n")
}

func writeDMLMethod(b *bytes.Buffer, spec resolvedWriteSpec) {
	constName := spec.MethodPrefix + "DML"
	fmt.Fprintf(b, "const %s = %s\n\n", constName, strconv.Quote(writeDMLSQL(spec)))
	fmt.Fprintf(b, "// %sDMLStatement returns a Cloud Spanner DML statement.\n", spec.MethodPrefix)
	b.WriteString("// Prefer either DML statements or mutations within one transaction unless you intentionally account for their execution order.\n")
	fmt.Fprintf(b, "func (w *%s) %sDMLStatement() spanner.Statement {\n", spec.InputStruct, spec.MethodPrefix)
	b.WriteString("\treturn spanner.Statement{\n")
	fmt.Fprintf(b, "\t\tSQL: %s,\n", constName)
	b.WriteString("\t\tParams: map[string]interface{}{\n")
	for _, column := range spec.InputColumns {
		fmt.Fprintf(b, "\t\t\t%q: w.%s,\n", writeParamName(spec, column), writeFieldName(spec, column))
	}
	b.WriteString("\t\t},\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")
}

func writeDMLSQL(spec resolvedWriteSpec) string {
	switch spec.Operation {
	case "insert", "insert_or_update":
		verb := "INSERT INTO"
		if spec.Operation == "insert_or_update" {
			verb = "INSERT OR UPDATE INTO"
		}
		columns := spec.InsertColumns
		columnSQL := quoteColumnNames(columns)
		params := make([]string, 0, len(columns))
		for _, column := range columns {
			params = append(params, "@"+writeParamName(spec, column))
		}
		return verb + " " + quoteGoogleSQLPath(spec.Table) + " (" + strings.Join(columnSQL, ", ") + ") VALUES (" + strings.Join(params, ", ") + ")"
	case "update":
		setSQL := make([]string, 0, len(spec.UpdateColumns))
		for _, column := range spec.UpdateColumns {
			setSQL = append(setSQL, quoteGoogleSQLIdent(column.Name)+" = @"+writeParamName(spec, column))
		}
		return "UPDATE " + quoteGoogleSQLPath(spec.Table) + " SET " + strings.Join(setSQL, ", ") + " WHERE " + writeWhereSQL(spec, spec.Keys)
	case "delete":
		return "DELETE FROM " + quoteGoogleSQLPath(spec.Table) + " WHERE " + writeWhereSQL(spec, spec.Keys)
	default:
		return ""
	}
}

func writeWhereSQL(spec resolvedWriteSpec, keys []*Column) string {
	parts := make([]string, 0, len(keys))
	for _, column := range keys {
		parts = append(parts, quoteGoogleSQLIdent(column.Name)+" = @"+writeParamName(spec, column))
	}
	return strings.Join(parts, " AND ")
}

func quoteColumnNames(columns []*Column) []string {
	out := make([]string, 0, len(columns))
	for _, column := range columns {
		out = append(out, quoteGoogleSQLIdent(column.Name))
	}
	return out
}

func writeFieldAccessList(spec resolvedWriteSpec, columns []*Column) string {
	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, "w."+writeFieldName(spec, column))
	}
	return strings.Join(parts, ", ")
}

func writeFieldName(spec resolvedWriteSpec, column *Column) string {
	if name := spec.FieldNames[columnKey(column.Name)]; name != "" {
		return name
	}
	return exportedIdentifier(column.Name, "Field")
}

func writeParamName(spec resolvedWriteSpec, column *Column) string {
	if name := spec.ParamNames[columnKey(column.Name)]; name != "" {
		return name
	}
	return safeQueryParamName(column.Name)
}

func writeStringList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return strings.Join(quoted, ", ")
}

func googleSQLStringLiteral(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`'`, `\'`,
		"\n", `\n`,
		"\r", `\r`,
		"\t", `\t`,
	)
	return "'" + replacer.Replace(value) + "'"
}

func resolveWriteColumns(table *Table, names []string, defaultNames []string) ([]*Column, error) {
	if len(names) == 0 {
		names = defaultNames
	}
	columns := make([]*Column, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		column, _ := table.Column(name)
		if column == nil {
			return nil, fmt.Errorf("column %s does not exist on table %s", name, table.Name)
		}
		key := columnKey(column.Name)
		if seen[key] {
			return nil, fmt.Errorf("column %s is specified more than once", column.Name)
		}
		seen[key] = true
		columns = append(columns, column)
	}
	return columns, nil
}

func validateResolvedWriteColumns(table *Table, operation string, insertColumns, updateColumns, keyColumns []*Column) error {
	switch operation {
	case "insert", "replace":
		for _, column := range insertColumns {
			if !columnInsertable(column) {
				return fmt.Errorf("column %s is not insertable", column.Name)
			}
		}
	case "update":
		for _, column := range updateColumns {
			if column.PrimaryKey {
				return fmt.Errorf("primary key column %s cannot be in update_columns", column.Name)
			}
			if !columnUpdatable(column) {
				return fmt.Errorf("column %s is not updatable", column.Name)
			}
		}
	case "insert_or_update":
		for _, column := range insertColumns {
			if !columnInsertable(column) {
				return fmt.Errorf("column %s is not insertable", column.Name)
			}
		}
		for _, column := range updateColumns {
			if column.PrimaryKey {
				return fmt.Errorf("primary key column %s cannot be in update_columns", column.Name)
			}
			if !columnUpdatable(column) {
				return fmt.Errorf("column %s is not updatable", column.Name)
			}
		}
		for _, column := range requiredInsertColumns(table, keyColumns) {
			if !containsColumn(insertColumns, column.Name) {
				return fmt.Errorf("insert_or_update requires insert value for NOT NULL column %s", column.Name)
			}
		}
	}
	return nil
}

func requiredInsertColumns(table *Table, keyColumns []*Column) []*Column {
	out := []*Column{}
	for _, column := range table.Columns {
		if !column.NotNull || containsColumn(keyColumns, column.Name) {
			continue
		}
		if column.DefaultSQL != "" || column.GeneratedSQL != "" || column.Hidden {
			continue
		}
		out = append(out, column)
	}
	return out
}

func resolveUpdateMaskNames(table *Table, operation string, updateMask []string, keys []*Column) ([]string, error) {
	if len(updateMask) == 0 {
		return nil, fmt.Errorf("update_mask is required for operation %s; use update_mask: [%s] to update every non-key column", operation, autoAllNonKeyColumns)
	}
	maskNames := updateMask
	if len(maskNames) == 1 && strings.EqualFold(maskNames[0], autoAllNonKeyColumns) {
		return nonKeyColumnNames(selectableColumnNames(table), keys), nil
	}
	for _, name := range maskNames {
		if strings.EqualFold(name, autoAllNonKeyColumns) {
			return nil, fmt.Errorf("%s cannot be combined with explicit update_mask columns", autoAllNonKeyColumns)
		}
	}
	return maskNames, nil
}

func selectableColumnNames(table *Table) []string {
	names := make([]string, 0, len(table.Columns))
	for _, column := range table.Columns {
		if !column.Hidden {
			names = append(names, column.Name)
		}
	}
	return names
}

func nonKeyColumnNames(names []string, keys []*Column) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if !containsColumn(keys, name) {
			out = append(out, name)
		}
	}
	return out
}

func firstMatchingColumn(columns, targets []*Column) (*Column, bool) {
	for _, column := range columns {
		if containsColumn(targets, column.Name) {
			return column, true
		}
	}
	return nil, false
}

func primaryKeyColumnNames(table *Table) []string {
	names := make([]string, 0, len(table.PrimaryKey))
	for _, key := range table.PrimaryKey {
		names = append(names, key.Name)
	}
	return names
}

func columnNamesFromColumns(columns []*Column) []string {
	names := make([]string, 0, len(columns))
	for _, column := range columns {
		names = append(names, column.Name)
	}
	return names
}

func appendMissingColumns(columns, extra []*Column) []*Column {
	out := append([]*Column(nil), columns...)
	for _, column := range extra {
		if !containsColumn(out, column.Name) {
			out = append(out, column)
		}
	}
	return out
}

func containsColumn(columns []*Column, name string) bool {
	for _, column := range columns {
		if strings.EqualFold(column.Name, name) {
			return true
		}
	}
	return false
}

func codegenTableQuerySQL(schema QueryCodegenSchema, tableName string, keyPrefix []string, orderBy, baseDir string, userParams []QueryCodegenParam) (string, []QueryCodegenParam, error) {
	orderByChoice := findOrderByChoiceParam(userParams)
	if strings.ToLower(emptyDefault(schema.Dialect, "spanner")) != "spanner" {
		if len(keyPrefix) > 0 {
			return "", nil, fmt.Errorf("key_prefix is only supported for Spanner table queries")
		}
		if strings.TrimSpace(orderBy) != "" && !strings.EqualFold(strings.TrimSpace(orderBy), "none") {
			return "", nil, fmt.Errorf("order_by is only supported for Spanner table queries")
		}
		return "SELECT * FROM " + quoteGoogleSQLPath(tableName), nil, nil
	}
	orderMode, err := normalizeTableQueryOrderBy(schema, orderBy)
	if err != nil {
		return "", nil, err
	}
	catalog, err := codegenSpannerCatalog(schema, baseDir)
	if err != nil {
		return "", nil, err
	}
	table, err := catalogTable(catalog, tableName)
	if err != nil {
		return "", nil, err
	}
	columns := make([]string, 0, len(table.Columns))
	for _, column := range table.Columns {
		if !column.Hidden {
			columns = append(columns, quoteGoogleSQLIdent(column.Name))
		}
	}
	if len(columns) == 0 {
		return "", nil, fmt.Errorf("table %s has no selectable columns", tableName)
	}
	sql := "SELECT " + strings.Join(columns, ", ") + " FROM " + quoteGoogleSQLPath(table.Name.String())

	// Optional WHERE-by-key filter. key_prefix must be a prefix of the
	// table's primary-key column list, mirroring kind: index.
	var params []QueryCodegenParam
	if len(keyPrefix) > 0 {
		pkNames := primaryKeyColumnNames(table)
		if err := validateKeyPrefix(pkNames, keyPrefix); err != nil {
			return "", nil, err
		}
		optionalModes := queryOptionalModes(userParams)
		paramNames := queryParamNameMap(keyPrefix)
		// Same contiguous-prefix rule as kind: index.
		keyModes := make([]string, len(keyPrefix))
		anyOmittable := false
		for i, key := range keyPrefix {
			keyModes[i] = strings.ToLower(strings.TrimSpace(optionalModes[paramNames[columnKey(key)]]))
			switch keyModes[i] {
			case "omit_when_null", "omit_when_empty":
				anyOmittable = true
			}
		}
		for i := range keyModes {
			switch keyModes[i] {
			case "omit_when_null", "omit_when_empty":
				for j := i + 1; j < len(keyModes); j++ {
					switch keyModes[j] {
					case "omit_when_null", "omit_when_empty":
					default:
						return "", nil, fmt.Errorf("key column %s is %s but column %s (later in the key prefix) is required; primary-key seeks need contiguous predicates, so trailing key columns must also be optional",
							keyPrefix[i], keyModes[i], keyPrefix[j])
					}
				}
				break
			}
		}
		whereSQL := make([]string, 0, len(keyPrefix))
		params = make([]QueryCodegenParam, 0, len(keyPrefix))
		for i, key := range keyPrefix {
			column, _ := table.Column(key)
			if column == nil {
				return "", nil, fmt.Errorf("key column %s does not exist on table %s", key, table.Name)
			}
			typeSQL, err := typeSpecSQL(column.Type)
			if err != nil {
				return "", nil, fmt.Errorf("key column %s: %w", key, err)
			}
			paramName := paramNames[columnKey(key)]
			predicate := quoteGoogleSQLIdent(key) + " = @" + paramName
			switch keyModes[i] {
			case "", "required":
			case "null_is_null":
				predicate = quoteGoogleSQLIdent(key) + " IS NOT DISTINCT FROM @" + paramName
			case "omit_when_null":
				predicate = "/*?optional:" + paramName + "*/ AND " + quoteGoogleSQLIdent(key) + " = @" + paramName + " /*?end*/"
			case "omit_when_empty":
				return "", nil, fmt.Errorf("key column %s param %s: optional: omit_when_empty is not supported on kind: table (key columns are scalar; use omit_when_null instead)", key, paramName)
			default:
				return "", nil, fmt.Errorf("key column %s param %s: optional: %s is not supported on kind: table", key, paramName, keyModes[i])
			}
			whereSQL = append(whereSQL, predicate)
			params = append(params, QueryCodegenParam{Name: paramName, Type: typeSQL})
		}
		if anyOmittable {
			sql += " WHERE TRUE"
			for _, predicate := range whereSQL {
				if strings.HasPrefix(predicate, "/*?optional:") {
					sql += " " + predicate
				} else {
					sql += " AND " + predicate
				}
			}
		} else {
			sql += " WHERE " + strings.Join(whereSQL, " AND ")
		}
	}

	if orderByChoice.Name != "" {
		if orderMode == "none" {
			return "", nil, fmt.Errorf("param %s: optional: orderby_choice cannot be combined with order_by: none", orderByChoice.Name)
		}
		defaultSQL, ok := orderByChoice.Choices[orderByChoice.Default]
		if !ok {
			return "", nil, fmt.Errorf("param %s: default %q is not in choices", orderByChoice.Name, orderByChoice.Default)
		}
		sql += " /*?orderby:" + orderByChoice.Name + "*/ " + defaultSQL + " /*?end*/"
	} else if orderMode != "none" {
		sql = appendOrderBy(sql, primaryKeyColumnNames(table))
	}
	return sql, params, nil
}

func codegenIndexQuerySQL(schema QueryCodegenSchema, indexName string, keyPrefix []string, orderBy string, baseDir string, userParams []QueryCodegenParam) (string, []QueryCodegenParam, error) {
	optionalModes := queryOptionalModes(userParams)
	orderByChoice := findOrderByChoiceParam(userParams)
	orderMode, err := normalizeIndexQueryOrderBy(orderBy)
	if err != nil {
		return "", nil, err
	}
	catalog, err := codegenSpannerCatalog(schema, baseDir)
	if err != nil {
		return "", nil, err
	}
	index, err := catalogIndex(catalog, indexName)
	if err != nil {
		return "", nil, err
	}
	table, err := catalogTable(catalog, index.TableName.String())
	if err != nil {
		return "", nil, err
	}
	keyNames := make([]string, 0, len(index.Keys))
	for _, key := range index.Keys {
		keyNames = append(keyNames, key.Name)
	}
	if keyPrefix == nil {
		keyPrefix = keyNames
	}
	if err := validateKeyPrefix(keyNames, keyPrefix); err != nil {
		return "", nil, err
	}
	selectColumns := append([]string(nil), keyNames...)
	for _, column := range primaryKeyColumnNames(table) {
		if !containsName(selectColumns, column) {
			selectColumns = append(selectColumns, column)
		}
	}
	orderColumns := append([]string(nil), selectColumns...)
	for _, column := range index.StoredColumns {
		if !containsName(selectColumns, column) {
			selectColumns = append(selectColumns, column)
		}
	}
	selectSQL := make([]string, 0, len(selectColumns))
	for _, column := range selectColumns {
		if col, _ := table.Column(column); col == nil {
			return "", nil, fmt.Errorf("column %s does not exist on table %s", column, table.Name)
		}
		selectSQL = append(selectSQL, quoteGoogleSQLIdent(column))
	}
	whereSQL := make([]string, 0, len(keyPrefix)+len(index.Keys))
	params := make([]QueryCodegenParam, 0, len(keyPrefix))
	paramNames := queryParamNameMap(keyPrefix)
	// Walk the key prefix once to learn which columns are omittable.
	// Spanner index seeks need a contiguous prefix, so a column may
	// only be omittable if every column to its right is also omittable
	// (or absent).
	keyModes := make([]string, len(keyPrefix))
	anyOmittable := false
	for i, key := range keyPrefix {
		paramName := paramNames[columnKey(key)]
		keyModes[i] = strings.ToLower(strings.TrimSpace(optionalModes[paramName]))
		switch keyModes[i] {
		case "omit_when_null", "omit_when_empty":
			anyOmittable = true
		}
	}
	for i := range keyModes {
		switch keyModes[i] {
		case "omit_when_null", "omit_when_empty":
			for j := i + 1; j < len(keyModes); j++ {
				switch keyModes[j] {
				case "omit_when_null", "omit_when_empty":
				default:
					return "", nil, fmt.Errorf("key column %s is %s but column %s (later in the key prefix) is required; key-prefix seeks need contiguous predicates, so trailing key columns must also be optional",
						keyPrefix[i], keyModes[i], keyPrefix[j])
				}
			}
			break
		}
	}
	for i, key := range keyPrefix {
		column, _ := table.Column(key)
		if column == nil {
			return "", nil, fmt.Errorf("key column %s does not exist on table %s", key, table.Name)
		}
		typeSQL, err := typeSpecSQL(column.Type)
		if err != nil {
			return "", nil, fmt.Errorf("key column %s: %w", key, err)
		}
		paramName := paramNames[columnKey(key)]
		predicate := quoteGoogleSQLIdent(key) + " = @" + paramName
		switch keyModes[i] {
		case "", "required":
			// Default: standard equality.
		case "null_is_null":
			predicate = quoteGoogleSQLIdent(key) + " IS NOT DISTINCT FROM @" + paramName
		case "omit_when_null":
			predicate = "/*?optional:" + paramName + "*/ AND " + quoteGoogleSQLIdent(key) + " = @" + paramName + " /*?end*/"
		case "omit_when_empty":
			return "", nil, fmt.Errorf("key column %s param %s: optional: omit_when_empty is not supported on kind: index (key columns are scalar; use omit_when_null instead)", key, paramName)
		default:
			return "", nil, fmt.Errorf("key column %s param %s: optional: %s is not supported on kind: index", key, paramName, keyModes[i])
		}
		whereSQL = append(whereSQL, predicate)
		params = append(params, QueryCodegenParam{Name: paramName, Type: typeSQL})
	}
	whereSQL, err = appendNullFilteredIndexPredicates(whereSQL, table, index)
	if err != nil {
		return "", nil, err
	}
	sql := "SELECT " + strings.Join(selectSQL, ", ") +
		" FROM " + quoteGoogleSQLPath(table.Name.String()) +
		"@{FORCE_INDEX=" + quoteGoogleSQLIdent(index.Name.String()) + "}"
	if len(whereSQL) > 0 {
		if anyOmittable {
			// Use `WHERE TRUE` so the SQL stays valid when every
			// omittable predicate is dropped at runtime. Each
			// predicate carries its own conjunctor (either `AND` for
			// required, or `/*?optional:...*/ AND ... /*?end*/` for
			// omit_when_null).
			sql += " WHERE TRUE"
			for _, predicate := range whereSQL {
				if strings.HasPrefix(predicate, "/*?optional:") {
					sql += " " + predicate
				} else {
					sql += " AND " + predicate
				}
			}
		} else {
			sql += " WHERE " + strings.Join(whereSQL, " AND ")
		}
	}
	if orderByChoice.Name != "" {
		if orderMode == "none" {
			return "", nil, fmt.Errorf("param %s: optional: orderby_choice cannot be combined with order_by: none", orderByChoice.Name)
		}
		defaultSQL, ok := orderByChoice.Choices[orderByChoice.Default]
		if !ok {
			return "", nil, fmt.Errorf("param %s: default %q is not in choices", orderByChoice.Name, orderByChoice.Default)
		}
		sql += " /*?orderby:" + orderByChoice.Name + "*/ " + defaultSQL + " /*?end*/"
	} else if orderMode != "none" {
		sql = appendOrderBy(sql, orderColumns)
	}
	return sql, params, nil
}

func appendNullFilteredIndexPredicates(whereSQL []string, table *Table, index *Index) ([]string, error) {
	if !index.NullFiltered {
		return whereSQL, nil
	}
	// Equality against a parameter is not a useful review-time proof that a
	// nullable NULL_FILTERED index key is non-NULL, so emit explicit filters
	// even for generated key-prefix predicates.
	for _, key := range index.Keys {
		column, _ := table.Column(key.Name)
		if column == nil {
			return nil, fmt.Errorf("index key column %s does not exist on table %s", key.Name, table.Name)
		}
		if column.NotNull {
			continue
		}
		whereSQL = append(whereSQL, quoteGoogleSQLIdent(column.Name)+" IS NOT NULL")
	}
	return whereSQL, nil
}

func normalizeTableQueryOrderBy(schema QueryCodegenSchema, orderBy string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(orderBy)) {
	case "", "key":
		if strings.ToLower(emptyDefault(schema.Dialect, "spanner")) != "spanner" {
			return "none", nil
		}
		return "key", nil
	case "none":
		return "none", nil
	default:
		return "", fmt.Errorf("unsupported table query order_by %q; use key or none", orderBy)
	}
}

func normalizeIndexQueryOrderBy(orderBy string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(orderBy)) {
	case "", "key":
		return "key", nil
	case "none":
		return "none", nil
	default:
		return "", fmt.Errorf("unsupported index query order_by %q; use key or none", orderBy)
	}
}

func appendOrderBy(sql string, columns []string) string {
	if len(columns) == 0 {
		return sql
	}
	orderSQL := make([]string, 0, len(columns))
	for _, column := range columns {
		orderSQL = append(orderSQL, quoteGoogleSQLIdent(column))
	}
	return sql + " ORDER BY " + strings.Join(orderSQL, ", ")
}

func codegenSpannerCatalog(schema QueryCodegenSchema, baseDir string) (*Catalog, error) {
	ddlPath, ddl, err := readCodegenDDL(schema, baseDir)
	if err != nil {
		return nil, err
	}
	return BuildSchemaCatalog(ddlPath, ddl)
}

func catalogTable(catalog *Catalog, name string) (*Table, error) {
	if table, ok := catalog.Tables[name]; ok {
		return table, nil
	}
	for _, table := range catalog.Tables {
		if strings.EqualFold(table.Name.String(), name) {
			return table, nil
		}
		for _, synonym := range table.Synonyms {
			if strings.EqualFold(synonym, name) {
				return table, nil
			}
		}
	}
	return nil, fmt.Errorf("table %s does not exist", name)
}

func catalogIndex(catalog *Catalog, name string) (*Index, error) {
	if index, ok := catalog.Indexes[name]; ok {
		return index, nil
	}
	for _, index := range catalog.Indexes {
		if strings.EqualFold(index.Name.String(), name) {
			return index, nil
		}
	}
	return nil, fmt.Errorf("index %s does not exist", name)
}

func validateKeyPrefix(indexKeys, keyPrefix []string) error {
	if len(keyPrefix) > len(indexKeys) {
		return fmt.Errorf("key_prefix has %d columns but index has %d keys", len(keyPrefix), len(indexKeys))
	}
	for i, key := range keyPrefix {
		if !strings.EqualFold(key, indexKeys[i]) {
			return fmt.Errorf("key_prefix must match index keys from the first key: got %s at position %d, want %s", key, i+1, indexKeys[i])
		}
	}
	return nil
}

var safeParamNonIdent = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func safeQueryParamName(name string) string {
	name = safeParamNonIdent.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		return "param"
	}
	r := rune(name[0])
	if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && r != '_' {
		return "_" + name
	}
	return name
}

func queryParamNameMap(names []string) map[string]string {
	used := map[string]bool{}
	out := map[string]string{}
	for _, name := range names {
		out[columnKey(name)] = uniqueIdentifier(safeQueryParamName(name), used)
	}
	return out
}

func columnKey(name string) string {
	return strings.ToLower(name)
}

func sameNameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, name := range a {
		seen[columnKey(name)]++
	}
	for _, name := range b {
		key := columnKey(name)
		if seen[key] == 0 {
			return false
		}
		seen[key]--
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}

func quoteGoogleSQLPath(name string) string {
	parts := strings.Split(name, ".")
	for i, part := range parts {
		parts[i] = quoteGoogleSQLIdent(part)
	}
	return strings.Join(parts, ".")
}

func quoteGoogleSQLIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func typeSpecSQL(spec *TypeSpec) (string, error) {
	if spec == nil {
		return "", fmt.Errorf("nil type")
	}
	if spec.Tokenlist {
		return "TOKENLIST", nil
	}
	switch spec.Code {
	case spannerpb.TypeCode_BOOL:
		return "BOOL", nil
	case spannerpb.TypeCode_INT64:
		return "INT64", nil
	case spannerpb.TypeCode_FLOAT64:
		return "FLOAT64", nil
	case spannerpb.TypeCode_FLOAT32:
		return "FLOAT32", nil
	case spannerpb.TypeCode_TIMESTAMP:
		return "TIMESTAMP", nil
	case spannerpb.TypeCode_DATE:
		return "DATE", nil
	case spannerpb.TypeCode_STRING:
		return "STRING", nil
	case spannerpb.TypeCode_BYTES:
		return "BYTES", nil
	case spannerpb.TypeCode_NUMERIC:
		return "NUMERIC", nil
	case spannerpb.TypeCode_JSON:
		return "JSON", nil
	case spannerpb.TypeCode_UUID:
		return "UUID", nil
	case spannerpb.TypeCode_ARRAY:
		elem, err := typeSpecSQL(spec.ArrayElement)
		if err != nil {
			return "", err
		}
		return "ARRAY<" + elem + ">", nil
	case spannerpb.TypeCode_STRUCT:
		fields := make([]string, 0, len(spec.StructFields))
		for _, field := range spec.StructFields {
			fieldType, err := typeSpecSQL(field.Type)
			if err != nil {
				return "", err
			}
			if field.Name != "" {
				fieldType = quoteGoogleSQLIdent(field.Name) + " " + fieldType
			}
			fields = append(fields, fieldType)
		}
		return "STRUCT<" + strings.Join(fields, ", ") + ">", nil
	case spannerpb.TypeCode_PROTO:
		if spec.ProtoTypeFQN == "" {
			return "", fmt.Errorf("unnamed proto type")
		}
		return quoteGoogleSQLIdent(spec.ProtoTypeFQN), nil
	case spannerpb.TypeCode_ENUM:
		if spec.ProtoTypeFQN == "" {
			return "", fmt.Errorf("unnamed enum type")
		}
		return quoteGoogleSQLIdent(spec.ProtoTypeFQN), nil
	default:
		return "", fmt.Errorf("unsupported type code %s", spec.Code)
	}
}

func applyRequiredFields(fields []goResultField, required []string, policy string) ([]goResultField, error) {
	if len(required) == 0 {
		return fields, nil
	}
	policy = strings.ToLower(emptyDefault(policy, "override"))
	requiredSet := map[string]bool{}
	for _, name := range required {
		requiredSet[strings.ToLower(name)] = true
	}
	out := make([]goResultField, len(fields))
	copy(out, fields)
	for i := range out {
		key := strings.ToLower(out[i].Name)
		if !requiredSet[key] {
			continue
		}
		if policy == "strict" && out[i].Nullable {
			return nil, fmt.Errorf("field %s is required but nullability is not proven", out[i].Name)
		}
		if policy != "override" && policy != "strict" {
			return nil, fmt.Errorf("unsupported required_policy %q", policy)
		}
		out[i].Nullable = false
		delete(requiredSet, key)
	}
	for name := range requiredSet {
		return nil, fmt.Errorf("required field %q is not present in query result", name)
	}
	return out, nil
}

func mergeGoResultFields(existing, next []goResultField) ([]goResultField, error) {
	if len(existing) == 0 {
		out := make([]goResultField, len(next))
		copy(out, next)
		return out, nil
	}
	out := make([]goResultField, len(existing))
	copy(out, existing)
	index := map[string]int{}
	for i, field := range out {
		index[strings.ToLower(field.Name)] = i
	}
	seen := map[string]bool{}
	for _, field := range next {
		key := strings.ToLower(field.Name)
		seen[key] = true
		if i, ok := index[key]; ok {
			merged, err := mergeGoResultField(out[i], field)
			if err != nil {
				return nil, err
			}
			out[i] = merged
			continue
		}
		index[key] = len(out)
		out = append(out, forceNullableGoResultField(field))
	}
	for i := range out {
		if !seen[strings.ToLower(out[i].Name)] {
			out[i] = forceNullableGoResultField(out[i])
		}
	}
	return out, nil
}

func mergeGoResultField(existing, next goResultField) (goResultField, error) {
	if !sameGoResultShape(existing, next) {
		return goResultField{}, fmt.Errorf("field %s has conflicting types %s and %s", next.Name, fieldCompatibilityKey(existing), fieldCompatibilityKey(next))
	}
	merged := existing
	merged.Nullable = existing.Nullable || next.Nullable
	if len(existing.Fields) > 0 || len(next.Fields) > 0 {
		fields, err := mergeGoResultFields(existing.Fields, next.Fields)
		if err != nil {
			return goResultField{}, fmt.Errorf("field %s: %w", next.Name, err)
		}
		merged.Fields = fields
	}
	return merged, nil
}

func sameGoResultShape(a, b goResultField) bool {
	return canonicalGoResultKind(a.Kind) == canonicalGoResultKind(b.Kind) && a.Repeated == b.Repeated
}

func forceNullableGoResultField(field goResultField) goResultField {
	field.Nullable = true
	return field
}

func fieldCompatibilityKey(field goResultField) string {
	kind := canonicalGoResultKind(field.Kind)
	parts := []string{kind}
	if field.Repeated {
		parts = append(parts, "repeated")
	}
	if field.Nullable {
		parts = append(parts, "nullable")
	} else {
		parts = append(parts, "required")
	}
	if len(field.Fields) > 0 {
		nested := make([]string, 0, len(field.Fields))
		for _, nestedField := range field.Fields {
			nested = append(nested, strings.ToLower(nestedField.Name)+":"+fieldCompatibilityKey(nestedField))
		}
		parts = append(parts, strings.Join(nested, ","))
	}
	return strings.Join(parts, "/")
}

func canonicalGoResultKind(kind string) string {
	switch strings.ToUpper(kind) {
	case "INTEGER":
		return "INT64"
	case "FLOAT", "DOUBLE":
		return "FLOAT64"
	case "BOOLEAN":
		return "BOOL"
	case "RECORD":
		return "STRUCT"
	default:
		return strings.ToUpper(kind)
	}
}

func resolveCodegenPaths(baseDir string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, resolveCodegenPath(baseDir, path))
	}
	return out
}

func resolveCodegenPath(baseDir, path string) string {
	if filepath.IsAbs(path) || baseDir == "" {
		return path
	}
	return filepath.Join(baseDir, path)
}

func emptyDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
