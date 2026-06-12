package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/internal/querygen"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		exitErr(err)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("subcommand is required: generate, check, explain-plan, plan-report, vet, config-schema, plan-report-schema, or plan-contract-schema")
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		return writeRootHelp(stdout)
	}
	if args[0] == "generate" {
		return runGenerate(args[1:], stdout, stderr, false)
	}
	if args[0] == "check" {
		return runGenerate(args[1:], stdout, stderr, true)
	}
	if args[0] == "explain-plan" {
		return runExplainPlan(args[1:], stdout, stderr)
	}
	if args[0] == "plan-report" {
		return runPlanReport(args[1:], stdout, stderr)
	}
	if args[0] == "vet" {
		return runVet(args[1:], stdout, stderr)
	}
	if args[0] == "config-schema" {
		return runConfigSchema(args[1:], stdout, stderr)
	}
	if args[0] == "plan-report-schema" {
		return runPlanReportSchema(args[1:], stdout, stderr)
	}
	if args[0] == "plan-contract-schema" {
		return runPlanContractSchema(args[1:], stdout, stderr)
	}
	return fmt.Errorf("unsupported subcommand %q; use generate, check, explain-plan, plan-report, vet, config-schema, plan-report-schema, or plan-contract-schema", args[0])
}

func writeRootHelp(stdout io.Writer) error {
	_, err := io.WriteString(stdout, `Usage:
  spanner-query-gen <subcommand> [flags]

Subcommands:
  generate       generate Go code from a v1alpha config
  check          verify the configured generated file is up to date
  explain-plan   print the resolved generation plan
  plan-report    analyze configured Spanner queries on Omni and print query plans
  vet            validate the resolved generation plan
  config-schema  print the v1alpha config JSON Schema
  plan-report-schema
                 print the plan-report output JSON Schema
  plan-contract-schema
                 print the plan contracts JSON Schema

Use "spanner-query-gen <subcommand> --help" for subcommand flags.
`)
	return err
}

func flagOutput(args []string, stdout, stderr io.Writer) io.Writer {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return stdout
		}
	}
	return stderr
}

func runGenerate(args []string, stdout, stderr io.Writer, forceCheck bool) error {
	fs := flag.NewFlagSet("spanner-query-gen", flag.ContinueOnError)
	fs.SetOutput(flagOutput(args, stdout, stderr))
	configPath := fs.String("config", "spanner-query-gen.yaml", "query code generation config file")
	outOverride := fs.String("out", "", "override output Go file path; use - for stdout")
	check := fs.Bool("check", false, "verify the generated file is up to date without writing it")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := readConfig(*configPath)
	if err != nil {
		return err
	}
	outputPath := resolveOutputPath(filepath.Dir(*configPath), config.Out)
	if *outOverride != "" {
		outputPath = *outOverride
	}
	code, err := querygen.GenerateQueryCode(config, filepath.Dir(*configPath))
	if err != nil {
		return err
	}
	return writeOutput(outputPath, code, stdout, *check || forceCheck)
}

func runExplainPlan(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("spanner-query-gen explain-plan", flag.ContinueOnError)
	fs.SetOutput(flagOutput(args, stdout, stderr))
	configPath := fs.String("config", "spanner-query-gen.yaml", "query code generation config file")
	output := fs.String("output", "yaml", "plan output format: yaml, json, or summary")
	stable := fs.Bool("stable", false, "omit volatile evidence fields from machine-readable plan output")
	audit := fs.Bool("audit", false, "include audit-only projection matrix details in machine-readable plan output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := readConfig(*configPath)
	if err != nil {
		return err
	}
	plan, err := querygen.BuildQueryCodegenPlan(config, filepath.Dir(*configPath))
	if err != nil {
		return err
	}
	if !*audit {
		makePlanConcise(plan)
	}
	if *stable {
		makePlanStable(plan)
	}
	switch *output {
	case "yaml":
		data, err := marshalYAMLViaJSON(newExplainPlan(plan))
		if err != nil {
			return err
		}
		_, err = stdout.Write(data)
		return err
	case "json":
		data, err := marshalIndentedJSON(newExplainPlan(plan))
		if err != nil {
			return err
		}
		_, err = stdout.Write(data)
		return err
	case "summary":
		return writePlanSummary(stdout, plan)
	default:
		return fmt.Errorf("unsupported --output %q", *output)
	}
}

func makePlanConcise(plan *querygen.QueryCodegenPlan) {
	if plan == nil {
		return
	}
	for i := range plan.CatalogBindings {
		plan.CatalogBindings[i].ProjectionMatrix.SourceURL = ""
		plan.CatalogBindings[i].ProjectionMatrix.DocsLastChecked = ""
		plan.CatalogBindings[i].ProjectionMatrix.Rows = nil
	}
}

func makePlanStable(plan *querygen.QueryCodegenPlan) {
	if plan == nil {
		return
	}
	for i := range plan.CatalogBindings {
		verification := &plan.CatalogBindings[i].AccessVerification
		if !verification.Volatile {
			plan.CatalogBindings[i].ProjectionMatrix.DocsLastChecked = ""
			continue
		}
		verification.CheckedAt = ""
		verification.Volatile = false
		plan.CatalogBindings[i].ProjectionMatrix.DocsLastChecked = ""
	}
}

type explainPlan struct {
	PlanVersion     int                                `json:"plan_version" yaml:"plan_version"`
	Generator       querygen.QueryCodegenPlanGenerator `json:"generator" yaml:"generator"`
	Package         string                             `json:"package,omitempty" yaml:"package,omitempty"`
	Out             string                             `json:"out,omitempty" yaml:"out,omitempty"`
	Client          querygen.GoStructTarget            `json:"client,omitempty" yaml:"client,omitempty"`
	SchemaDigests   []querygen.QueryCodegenPlanDigest  `json:"schema_digests,omitempty" yaml:"schema_digests,omitempty"`
	CatalogBindings []explainCatalogBinding            `json:"catalog_bindings,omitempty" yaml:"catalog_bindings,omitempty"`
	Queries         []explainPlanQuery                 `json:"queries,omitempty" yaml:"queries,omitempty"`
	Writes          []querygen.QueryCodegenPlanWrite   `json:"writes,omitempty" yaml:"writes,omitempty"`
	Structs         []querygen.QueryCodegenPlanStruct  `json:"structs,omitempty" yaml:"structs,omitempty"`
}

type explainCatalogBinding struct {
	Kind            string                                             `json:"kind" yaml:"kind"`
	Name            string                                             `json:"name" yaml:"name"`
	BigQuery        explainCatalogBindingBigQuery                      `json:"bigquery" yaml:"bigquery"`
	Spanner         explainCatalogBindingSpanner                       `json:"spanner" yaml:"spanner"`
	Access          explainCatalogBindingAccess                        `json:"access" yaml:"access"`
	Projection      explainCatalogBindingProjection                    `json:"projection" yaml:"projection"`
	Execution       spanalyzer.BigQuerySpannerExternalDatasetExecution `json:"execution" yaml:"execution"`
	Limitations     []string                                           `json:"limitations,omitempty" yaml:"limitations,omitempty"`
	Warnings        []querygen.QueryCodegenPlanWarning                 `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	VetSuppressions []querygen.QueryCodegenPlanVetSuppression          `json:"vet_suppressions,omitempty" yaml:"vet_suppressions,omitempty"`
}

type explainCatalogBindingBigQuery struct {
	Dataset          spanalyzer.BigQueryDatasetReference                        `json:"dataset" yaml:"dataset"`
	Location         string                                                     `json:"location,omitempty" yaml:"location,omitempty"`
	LocationMetadata *spanalyzer.BigQuerySpannerExternalDatasetLocationMetadata `json:"location_metadata,omitempty" yaml:"location_metadata,omitempty"`
}

type explainCatalogBindingSpanner struct {
	Catalog            string `json:"catalog" yaml:"catalog"`
	DatabaseURI        string `json:"database_uri,omitempty" yaml:"database_uri,omitempty"`
	DatabaseRole       string `json:"database_role,omitempty" yaml:"database_role,omitempty"`
	DatabaseRoleSource string `json:"database_role_source,omitempty" yaml:"database_role_source,omitempty"`
}

type explainCatalogBindingAccess struct {
	Mode                            string                                                                    `json:"mode" yaml:"mode"`
	ConnectionID                    string                                                                    `json:"connection_id,omitempty" yaml:"connection_id,omitempty"`
	CloudResourceConnectionID       string                                                                    `json:"cloud_resource_connection_id,omitempty" yaml:"cloud_resource_connection_id,omitempty"`
	CloudResourceConnectionMetadata *spanalyzer.BigQuerySpannerExternalDatasetCloudResourceConnectionMetadata `json:"cloud_resource_connection_metadata,omitempty" yaml:"cloud_resource_connection_metadata,omitempty"`
	Verification                    spanalyzer.BigQuerySpannerExternalDatasetVerification                     `json:"verification" yaml:"verification"`
	RequiresDataBoostAccess         bool                                                                      `json:"requires_databoost_access" yaml:"requires_databoost_access"`
}

type explainCatalogBindingProjection struct {
	UnsupportedColumns string                                                     `json:"unsupported_columns" yaml:"unsupported_columns"`
	NamedSchemaTables  string                                                     `json:"named_schema_tables" yaml:"named_schema_tables"`
	Matrix             *spanalyzer.BigQuerySpannerExternalDatasetProjectionMatrix `json:"matrix,omitempty" yaml:"matrix,omitempty"`
	ProjectedTables    []spanalyzer.BigQuerySpannerExternalDatasetTable           `json:"projected_tables,omitempty" yaml:"projected_tables,omitempty"`
}

type explainPlanQuery struct {
	Name            string                                    `json:"name" yaml:"name"`
	Catalog         string                                    `json:"catalog" yaml:"catalog"`
	Kind            string                                    `json:"kind" yaml:"kind"`
	Result          string                                    `json:"result" yaml:"result"`
	OrderBy         string                                    `json:"order_by,omitempty" yaml:"order_by,omitempty"`
	SQL             string                                    `json:"sql" yaml:"sql"`
	SQLSHA256       string                                    `json:"sql_sha256" yaml:"sql_sha256"`
	ResultStruct    string                                    `json:"result_struct" yaml:"result_struct"`
	Constants       []querygen.QueryCodegenPlanConstant       `json:"constants,omitempty" yaml:"constants,omitempty"`
	Params          []querygen.QueryCodegenParam              `json:"params,omitempty" yaml:"params,omitempty"`
	StarExpansion   *explainPlanStarExpansion                 `json:"star_expansion,omitempty" yaml:"star_expansion,omitempty"`
	Relations       []explainPlanRelation                     `json:"relations,omitempty" yaml:"relations,omitempty"`
	VetSuppressions []querygen.QueryCodegenPlanVetSuppression `json:"vet_suppressions,omitempty" yaml:"vet_suppressions,omitempty"`
	Warnings        []querygen.QueryCodegenPlanWarning        `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	Fields          []querygen.QueryCodegenPlanField          `json:"fields,omitempty" yaml:"fields,omitempty"`
}

type explainPlanStarExpansion struct {
	Catalog    string                              `json:"catalog" yaml:"catalog"`
	Projection querygen.QueryCodegenPlanProjection `json:"projection" yaml:"projection"`
}

type explainPlanRelation struct {
	SQLPath        string                              `json:"sql_path" yaml:"sql_path"`
	Catalog        string                              `json:"catalog" yaml:"catalog"`
	Role           string                              `json:"role" yaml:"role"`
	Allowed        bool                                `json:"allowed" yaml:"allowed"`
	WritableTarget bool                                `json:"writable_target" yaml:"writable_target"`
	Diagnostic     string                              `json:"diagnostic,omitempty" yaml:"diagnostic,omitempty"`
	Projection     querygen.QueryCodegenPlanProjection `json:"projection" yaml:"projection"`
}

func newExplainPlan(plan *querygen.QueryCodegenPlan) explainPlan {
	if plan == nil {
		return explainPlan{}
	}
	out := explainPlan{
		PlanVersion:   plan.PlanVersion,
		Generator:     plan.Generator,
		Package:       plan.Package,
		Out:           plan.Out,
		Client:        plan.Client,
		SchemaDigests: append([]querygen.QueryCodegenPlanDigest(nil), plan.SchemaDigests...),
		Writes:        append([]querygen.QueryCodegenPlanWrite(nil), plan.Writes...),
		Structs:       append([]querygen.QueryCodegenPlanStruct(nil), plan.Structs...),
	}
	for _, binding := range plan.CatalogBindings {
		out.CatalogBindings = append(out.CatalogBindings, newExplainCatalogBinding(binding))
	}
	for _, query := range plan.Queries {
		out.Queries = append(out.Queries, newExplainPlanQuery(query))
	}
	return out
}

func newExplainCatalogBinding(binding spanalyzer.BigQuerySpannerExternalDatasetBinding) explainCatalogBinding {
	name := binding.Name
	if name == "" {
		name = binding.SpannerSource + "." + binding.BigQueryDataset
	}
	var matrix *spanalyzer.BigQuerySpannerExternalDatasetProjectionMatrix
	if binding.ProjectionMatrix.Source != "" ||
		binding.ProjectionMatrix.SourceURL != "" ||
		binding.ProjectionMatrix.DocsLastUpdated != "" ||
		binding.ProjectionMatrix.DocsLastChecked != "" ||
		binding.ProjectionMatrix.GeneratorMatrixVersion != 0 ||
		len(binding.ProjectionMatrix.Rows) > 0 {
		value := binding.ProjectionMatrix
		matrix = &value
	}
	return explainCatalogBinding{
		Kind: binding.Kind,
		Name: name,
		BigQuery: explainCatalogBindingBigQuery{
			Dataset:          binding.BigQueryDatasetRef,
			Location:         binding.Location,
			LocationMetadata: binding.LocationMetadata,
		},
		Spanner: explainCatalogBindingSpanner{
			Catalog:            binding.SpannerSource,
			DatabaseURI:        binding.ExternalSource,
			DatabaseRole:       binding.DatabaseRole,
			DatabaseRoleSource: binding.DatabaseRoleSource,
		},
		Access: explainCatalogBindingAccess{
			Mode:                            binding.Access,
			ConnectionID:                    binding.Connection,
			CloudResourceConnectionID:       binding.CloudResourceConnection,
			CloudResourceConnectionMetadata: binding.CloudResourceConnectionMetadata,
			Verification:                    binding.AccessVerification,
			RequiresDataBoostAccess:         binding.RequiresDataBoostAccess,
		},
		Projection: explainCatalogBindingProjection{
			UnsupportedColumns: binding.ProjectionPolicy.UnsupportedColumns,
			NamedSchemaTables:  binding.ProjectionPolicy.NamedSchemas,
			Matrix:             matrix,
			ProjectedTables:    append([]spanalyzer.BigQuerySpannerExternalDatasetTable(nil), binding.ProjectedTables...),
		},
		Execution:       binding.Execution,
		Limitations:     append([]string(nil), binding.Limitations...),
		Warnings:        append([]querygen.QueryCodegenPlanWarning(nil), binding.Warnings...),
		VetSuppressions: append([]querygen.QueryCodegenPlanVetSuppression(nil), binding.VetSuppressions...),
	}
}

func newExplainPlanQuery(query querygen.QueryCodegenPlanQuery) explainPlanQuery {
	out := explainPlanQuery{
		Name:            query.Name,
		Catalog:         query.Catalog,
		Kind:            query.Kind,
		Result:          query.Result,
		OrderBy:         query.OrderBy,
		SQL:             query.SQL,
		SQLSHA256:       query.SQLSHA256,
		ResultStruct:    query.ResultStruct,
		Constants:       append([]querygen.QueryCodegenPlanConstant(nil), query.Constants...),
		Params:          append([]querygen.QueryCodegenParam(nil), query.Params...),
		VetSuppressions: append([]querygen.QueryCodegenPlanVetSuppression(nil), query.VetSuppressions...),
		Warnings:        append([]querygen.QueryCodegenPlanWarning(nil), query.Warnings...),
		Fields:          append([]querygen.QueryCodegenPlanField(nil), query.Fields...),
	}
	if query.StarExpansion != nil {
		out.StarExpansion = &explainPlanStarExpansion{
			Catalog:    query.StarExpansion.Catalog,
			Projection: query.StarExpansion.Projection,
		}
	}
	for _, relation := range query.Relations {
		out.Relations = append(out.Relations, explainPlanRelation{
			SQLPath:        relation.SQLPath,
			Catalog:        relation.Catalog,
			Role:           relation.Role,
			Allowed:        relation.Allowed,
			WritableTarget: relation.WritableTarget,
			Diagnostic:     relation.Diagnostic,
			Projection:     relation.Projection,
		})
	}
	return out
}

func writePlanSummary(w io.Writer, plan *querygen.QueryCodegenPlan) error {
	var b strings.Builder
	for _, binding := range plan.CatalogBindings {
		appendSummaryf(&b, "Catalog binding: %s (%s, source: %s)\n", binding.BigQueryDataset, binding.Kind, binding.SpannerSource)
		appendSummaryf(&b, "  Dataset: %s\n", binding.BigQueryDatasetRef.Path)
		if binding.BigQueryDatasetRef.Project != "" {
			appendSummaryf(&b, "  Project: %s\n", binding.BigQueryDatasetRef.Project)
		}
		if binding.ExternalSource != "" {
			appendSummaryf(&b, "  External source: %s\n", binding.ExternalSource)
		}
		if binding.Location != "" {
			appendSummaryf(&b, "  Location: %s", binding.Location)
			if binding.LocationMetadata != nil && binding.LocationMetadata.Configured != "" && binding.LocationMetadata.Configured != binding.Location {
				appendSummaryf(&b, " (configured: %s)", binding.LocationMetadata.Configured)
			}
			b.WriteByte('\n')
		}
		if binding.Connection != "" {
			appendSummaryf(&b, "  Connection: %s\n", binding.Connection)
		}
		if binding.CloudResourceConnection != "" {
			appendSummaryf(&b, "  Cloud resource connection: %s\n", binding.CloudResourceConnection)
			if binding.CloudResourceConnectionMetadata != nil && binding.CloudResourceConnectionMetadata.ParsedLocation != "" {
				appendSummaryf(&b, "  Cloud resource connection location: %s", binding.CloudResourceConnectionMetadata.ParsedLocation)
				if binding.CloudResourceConnectionMetadata.ParsedLocationCanonical != "" && binding.CloudResourceConnectionMetadata.ParsedLocationCanonical != binding.CloudResourceConnectionMetadata.ParsedLocation {
					appendSummaryf(&b, " (canonical: %s)", binding.CloudResourceConnectionMetadata.ParsedLocationCanonical)
				}
				if binding.CloudResourceConnectionMetadata.LocationMatch {
					b.WriteString(" matched")
				}
				b.WriteByte('\n')
			}
		}
		appendSummaryf(&b, "  Projection policy: unsupported_columns=%s, named_schema_tables=%s\n", binding.ProjectionPolicy.UnsupportedColumns, binding.ProjectionPolicy.NamedSchemas)
		appendSummaryf(&b, "  Access: %s", binding.Access)
		if binding.DatabaseRole != "" {
			appendSummaryf(&b, ", database_role=%s", binding.DatabaseRole)
		}
		appendSummaryf(&b, ", verification=%s\n", binding.AccessVerification.Status)
		appendSummaryf(&b, "  Execution: data_boost=%s, writable=%t\n", binding.Execution.DataBoost, binding.Execution.Writable)
		for _, table := range binding.ProjectedTables {
			if table.Visible {
				appendSummaryf(&b, "  - %s from %s", table.BigQueryTable, table.SpannerTable)
				if !table.BigQueryKeyMetadataVisible {
					b.WriteString(" (BigQuery key metadata hidden)")
				}
				b.WriteByte('\n')
			} else {
				appendSummaryf(&b, "  - %s from %s skipped: %s\n", table.Name, table.SourceTable, table.Reason)
			}
		}
		if len(binding.Warnings) > 0 {
			b.WriteString("  Warnings:\n")
			for _, warning := range binding.Warnings {
				appendSummaryf(&b, "  - [%s] %s: %s\n", warning.Severity, warning.Rule, warning.Message)
				if warning.Remediation != "" {
					appendSummaryf(&b, "    Remediation: %s\n", warning.Remediation)
				}
			}
		}
		if len(plan.Queries) > 0 || len(plan.Writes) > 0 {
			b.WriteByte('\n')
		}
	}
	for i, query := range plan.Queries {
		if i > 0 {
			b.WriteByte('\n')
		}
		appendSummaryf(&b, "Query: %s (%s, result: %s, catalog: %s)\n", query.Name, query.Kind, query.Result, query.Catalog)
		appendSummaryf(&b, "  SQL: %s\n", query.SQL)
		if len(query.Params) > 0 {
			b.WriteString("  Params:\n")
			for _, param := range query.Params {
				appendSummaryf(&b, "  - %s: %s\n", param.Name, param.Type)
			}
		}
		if query.StarExpansion != nil {
			appendSummaryf(&b, "  Star expansion: catalog=%s, projection_loss=%t\n", query.StarExpansion.Catalog, query.StarExpansion.ProjectionLoss)
			for _, column := range query.StarExpansion.OmittedColumns {
				appendSummaryf(&b, "  - omitted: %s\n", column)
			}
		}
		if len(query.Relations) > 0 {
			b.WriteString("  Relations:\n")
			for _, relation := range query.Relations {
				appendSummaryf(&b, "  - %s: catalog=%s, role=%s, allowed=%t, projection_loss=%t\n", relation.SQLPath, relation.Catalog, relation.Role, relation.Allowed, relation.ProjectionLoss)
				if relation.Diagnostic != "" {
					appendSummaryf(&b, "    diagnostic: %s\n", relation.Diagnostic)
				}
				for _, column := range relation.OmittedColumns {
					appendSummaryf(&b, "    omitted: %s\n", column)
				}
			}
		}
		if len(query.Fields) > 0 {
			b.WriteString("  Fields:\n")
			for _, field := range query.Fields {
				appendSummaryf(&b, "  - %s: %s%s\n", field.Name, field.Kind, nullableSummary(field.Nullable))
			}
		}
		if len(query.Warnings) > 0 {
			b.WriteString("  Warnings:\n")
			for _, warning := range query.Warnings {
				appendSummaryf(&b, "  - [%s] %s: %s\n", warning.Severity, warning.Rule, warning.Message)
				if warning.Remediation != "" {
					appendSummaryf(&b, "    Remediation: %s\n", warning.Remediation)
				}
			}
		}
	}
	for _, write := range plan.Writes {
		if len(plan.Queries) > 0 {
			b.WriteByte('\n')
		}
		appendSummaryf(&b, "Write: %s (%s, catalog: %s)\n", write.Name, write.Operation, write.Catalog)
		appendSummaryf(&b, "  Table: %s\n", write.Table)
		if len(write.Keys) > 0 {
			appendSummaryf(&b, "  Keys: %s\n", strings.Join(write.Keys, ", "))
		}
		if len(write.InsertColumns) > 0 {
			appendSummaryf(&b, "  Insert Columns: %s\n", strings.Join(write.InsertColumns, ", "))
		}
		if len(write.UpdateColumns) > 0 {
			appendSummaryf(&b, "  Update Columns: %s\n", strings.Join(write.UpdateColumns, ", "))
		}
		if len(write.Methods) > 0 {
			appendSummaryf(&b, "  Methods: %s\n", strings.Join(write.Methods, ", "))
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func appendSummaryf(b *strings.Builder, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(b, format, args...)
}

func nullableSummary(nullable bool) string {
	if nullable {
		return " nullable"
	}
	return " required"
}

func runVet(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("spanner-query-gen vet", flag.ContinueOnError)
	fs.SetOutput(flagOutput(args, stdout, stderr))
	configPath := fs.String("config", "spanner-query-gen.yaml", "query code generation config file")
	output := fs.String("output", "text", "vet output format: text, json, or yaml")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	config, err := readConfig(*configPath)
	if err != nil {
		var parseErr configParseError
		if isMachineVetOutput(*output) && errors.As(err, &parseErr) {
			return writeVetFailureReport(stdout, *output, vetDiagnostic{
				ID:           "config-parse-error",
				Stage:        "config_parse",
				Scope:        "config",
				Name:         parseErr.path,
				Rule:         "config-parse-error",
				Severity:     "error",
				Message:      err.Error(),
				Suppressible: false,
				Suppressed:   false,
			}, err)
		}
		return err
	}
	plan, err := querygen.BuildQueryCodegenPlan(config, filepath.Dir(*configPath))
	if err != nil {
		if isMachineVetOutput(*output) {
			return writeVetFailureReport(stdout, *output, vetDiagnosticFromError(err), err)
		}
		return err
	}
	report := vetReportFromPlan(plan)
	return writeVetReport(stdout, stderr, *output, report)
}

func isMachineVetOutput(output string) bool {
	return output == "json" || output == "yaml"
}

func writeVetFailureReport(stdout io.Writer, output string, diagnostic vetDiagnostic, cause error) error {
	if err := writeVetReportOutput(stdout, io.Discard, output, vetReport{Diagnostics: []vetDiagnostic{diagnostic}}); err != nil {
		return err
	}
	return silentExitError{err: cause}
}

func writeVetReport(stdout, stderr io.Writer, output string, report vetReport) error {
	if err := writeVetReportOutput(stdout, stderr, output, report); err != nil {
		return err
	}
	if reportHasUnsuppressedErrors(report) {
		return silentExitError{err: fmt.Errorf("vet found error diagnostics")}
	}
	return nil
}

func writeVetReportOutput(stdout, stderr io.Writer, output string, report vetReport) error {
	var writeErr error
	switch output {
	case "text", "summary":
		writeErr = writeVetReportText(stderr, report)
	case "json":
		data, err := marshalIndentedJSON(report)
		if err != nil {
			return err
		}
		_, writeErr = stdout.Write(data)
	case "yaml":
		data, err := marshalYAMLViaJSON(report)
		if err != nil {
			return err
		}
		_, writeErr = stdout.Write(data)
	default:
		return fmt.Errorf("unsupported --output %q", output)
	}
	if writeErr != nil {
		return writeErr
	}
	return nil
}

type vetReport struct {
	Diagnostics  []vetDiagnostic  `json:"diagnostics,omitempty" yaml:"diagnostics,omitempty"`
	Warnings     []vetDiagnostic  `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	Suppressions []vetSuppression `json:"suppressions,omitempty" yaml:"suppressions,omitempty"`
}

type vetDiagnostic struct {
	ID           string          `json:"id,omitempty" yaml:"id,omitempty"`
	Stage        string          `json:"stage,omitempty" yaml:"stage,omitempty"`
	Subject      string          `json:"subject,omitempty" yaml:"subject,omitempty"`
	Scope        string          `json:"scope" yaml:"scope"`
	Name         string          `json:"name" yaml:"name"`
	Rule         string          `json:"rule" yaml:"rule"`
	Severity     string          `json:"severity" yaml:"severity"`
	Message      string          `json:"message" yaml:"message"`
	Remediation  string          `json:"remediation,omitempty" yaml:"remediation,omitempty"`
	Suppressible bool            `json:"suppressible" yaml:"suppressible"`
	Suppressed   bool            `json:"suppressed" yaml:"suppressed"`
	Suppression  *vetSuppression `json:"suppression,omitempty" yaml:"suppression,omitempty"`
}

type vetSuppression struct {
	Scope   string `json:"scope" yaml:"scope"`
	Name    string `json:"name" yaml:"name"`
	Rule    string `json:"rule" yaml:"rule"`
	Reason  string `json:"reason" yaml:"reason"`
	Owner   string `json:"owner,omitempty" yaml:"owner,omitempty"`
	Expires string `json:"expires,omitempty" yaml:"expires,omitempty"`
}

func vetReportFromPlan(plan *querygen.QueryCodegenPlan) vetReport {
	var report vetReport
	if plan == nil {
		return report
	}
	for _, binding := range plan.CatalogBindings {
		suppressions := binding.VetSuppressions
		for _, warning := range binding.Warnings {
			suppression, suppressed := warningSuppression(warning.Rule, suppressions)
			diagnostic := vetDiagnosticFromWarning("catalog_binding", binding.BigQueryDataset, warning, suppressed)
			diagnostic.Suppression = vetSuppressionPtr("catalog_binding", binding.BigQueryDataset, suppression, suppressed)
			report.Diagnostics = append(report.Diagnostics, diagnostic)
			if suppressed {
				continue
			}
			report.Warnings = append(report.Warnings, diagnostic)
		}
		for _, suppression := range suppressions {
			report.Suppressions = append(report.Suppressions, vetSuppression{
				Scope:   "catalog_binding",
				Name:    binding.BigQueryDataset,
				Rule:    suppression.Rule,
				Reason:  suppression.Reason,
				Owner:   suppression.Owner,
				Expires: suppression.Expires,
			})
		}
	}
	for _, query := range plan.Queries {
		suppressions := query.VetSuppressions
		for _, warning := range query.Warnings {
			suppression, suppressed := warningSuppression(warning.Rule, suppressions)
			diagnostic := vetDiagnosticFromWarning("query", query.Name, warning, suppressed)
			diagnostic.Suppression = vetSuppressionPtr("query", query.Name, suppression, suppressed)
			report.Diagnostics = append(report.Diagnostics, diagnostic)
			if suppressed {
				continue
			}
			report.Warnings = append(report.Warnings, diagnostic)
		}
		for _, suppression := range suppressions {
			report.Suppressions = append(report.Suppressions, vetSuppression{
				Scope:   "query",
				Name:    query.Name,
				Rule:    suppression.Rule,
				Reason:  suppression.Reason,
				Owner:   suppression.Owner,
				Expires: suppression.Expires,
			})
		}
	}
	for _, write := range plan.Writes {
		for _, suppression := range write.VetSuppressions {
			report.Suppressions = append(report.Suppressions, vetSuppression{
				Scope:   "write",
				Name:    write.Name,
				Rule:    suppression.Rule,
				Reason:  suppression.Reason,
				Owner:   suppression.Owner,
				Expires: suppression.Expires,
			})
		}
	}
	return report
}

func vetDiagnosticFromError(err error) vetDiagnostic {
	diagnostic := vetDiagnostic{
		ID:           "planning-error",
		Stage:        "planning",
		Scope:        "plan",
		Name:         "spanner-query-gen",
		Rule:         "planning-error",
		Severity:     "error",
		Message:      err.Error(),
		Suppressible: false,
		Suppressed:   false,
	}
	var diagnosticErr querygen.QueryCodegenDiagnosticError
	if errors.As(err, &diagnosticErr) {
		diagnostic.ID = diagnosticErr.ID
		diagnostic.Rule = diagnosticErr.ID
		diagnostic.Stage = diagnosticErr.Stage
		diagnostic.Subject = diagnosticErr.Subject
	}
	return diagnostic
}

func vetDiagnosticFromWarning(scope, name string, warning querygen.QueryCodegenPlanWarning, suppressed bool) vetDiagnostic {
	return vetDiagnostic{
		ID:           warning.Rule,
		Stage:        vetDiagnosticStage(scope, warning.Rule),
		Scope:        scope,
		Name:         name,
		Rule:         warning.Rule,
		Severity:     warning.Severity,
		Message:      warning.Message,
		Remediation:  warning.Remediation,
		Suppressible: true,
		Suppressed:   suppressed,
	}
}

func vetDiagnosticStage(scope, rule string) string {
	if scope == "catalog_binding" && strings.HasPrefix(rule, "external-dataset-") {
		return "external_dataset_projection"
	}
	if scope == "query" && strings.HasPrefix(rule, "external-query-") {
		return "external_query_analysis"
	}
	if scope == "query" && strings.HasPrefix(rule, "cross-dialect-") {
		return "external_query_type_conversion"
	}
	if scope == "write" {
		return "write_planning"
	}
	return "planning"
}

func warningSuppression(rule string, suppressions []querygen.QueryCodegenPlanVetSuppression) (querygen.QueryCodegenPlanVetSuppression, bool) {
	for _, suppression := range suppressions {
		if suppression.Rule == rule {
			return suppression, true
		}
	}
	return querygen.QueryCodegenPlanVetSuppression{}, false
}

func vetSuppressionPtr(scope, name string, suppression querygen.QueryCodegenPlanVetSuppression, ok bool) *vetSuppression {
	if !ok {
		return nil
	}
	return &vetSuppression{
		Scope:   scope,
		Name:    name,
		Rule:    suppression.Rule,
		Reason:  suppression.Reason,
		Owner:   suppression.Owner,
		Expires: suppression.Expires,
	}
}

func reportHasUnsuppressedErrors(report vetReport) bool {
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Suppressed {
			continue
		}
		if diagnostic.Severity == "error" {
			return true
		}
	}
	return false
}

func writeVetReportText(w io.Writer, report vetReport) error {
	for _, warning := range report.Warnings {
		severity := warning.Severity
		if severity == "" {
			severity = "warning"
		}
		if _, err := fmt.Fprintf(w, "%s: %s %s: %s: %s\n", severity, warning.Scope, warning.Name, warning.Rule, warning.Message); err != nil {
			return err
		}
		if warning.Remediation != "" {
			if _, err := fmt.Fprintf(w, "  remediation: %s\n", warning.Remediation); err != nil {
				return err
			}
		}
	}
	for _, suppression := range report.Suppressions {
		if _, err := fmt.Fprintf(w, "suppression: %s %s: %s: %s\n", suppression.Scope, suppression.Name, suppression.Rule, suppression.Reason); err != nil {
			return err
		}
	}
	return nil
}

func readConfig(path string) (querygen.QueryCodegenConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return querygen.QueryCodegenConfig{}, err
	}
	config, err := querygen.ParseQueryCodegenConfigYAML(data)
	if err != nil {
		return querygen.QueryCodegenConfig{}, configParseError{path: path, err: err}
	}
	return config, nil
}

type configParseError struct {
	path string
	err  error
}

func (e configParseError) Error() string {
	return fmt.Sprintf("%s: %v", e.path, e.err)
}

func (e configParseError) Unwrap() error {
	return e.err
}

type silentExitError struct {
	err error
}

func (e silentExitError) Error() string {
	return e.err.Error()
}

func (e silentExitError) Unwrap() error {
	return e.err
}

func writeOutput(path, code string, stdout io.Writer, check bool) error {
	if path == "" || path == "-" {
		if check {
			return fmt.Errorf("--check requires file output")
		}
		_, err := io.WriteString(stdout, code)
		return err
	}
	if check {
		current, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(current) != code {
			return fmt.Errorf("%s is not up to date", path)
		}
		return nil
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(code), 0o644)
}

func resolveOutputPath(baseDir, path string) string {
	if path == "" || path == "-" || filepath.IsAbs(path) || baseDir == "" {
		return path
	}
	return filepath.Join(baseDir, path)
}

func exitErr(err error) {
	var silent silentExitError
	if errors.As(err, &silent) {
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
