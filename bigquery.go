package spanalyzer

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	googlesql "github.com/goccy/go-googlesql"
)

// BigQueryGoogleSQLCatalog contains a GoogleSQL frontend catalog built from
// BigQuery DDL.
type BigQueryGoogleSQLCatalog struct {
	SimpleCatalog                  *googlesql.SimpleCatalog
	simpleCatalogs                 map[string]*googlesql.SimpleCatalog
	spannerExternalDatasetBindings []BigQuerySpannerExternalDatasetBinding
	externalQueryAnalyzers         map[string]*Analyzer
	externalQueryTVFRegistered     bool
	externalQueryRowTypes          map[string]map[string]*spannerpb.StructType
	AnalyzerOptions                *googlesql.AnalyzerOptions
	TypeFactory                    *googlesql.TypeFactory
}

// BigQueryAnalyzer analyzes BigQuery GoogleSQL statements and returns BigQuery
// REST TableSchema-shaped metadata.
type BigQueryAnalyzer struct {
	googleSQL              *BigQueryGoogleSQLCatalog
	helper                 *GoogleSQLHelper
	externalQueryAnalyzers map[string]*Analyzer
}

// BigQueryTableSchema mirrors the BigQuery REST TableSchema JSON shape.
type BigQueryTableSchema struct {
	Fields []*BigQueryTableFieldSchema `json:"fields,omitempty"`
}

// BigQueryTableFieldSchema mirrors the BigQuery REST TableFieldSchema JSON
// shape for fields this project can infer from GoogleSQL analyzer types.
type BigQueryTableFieldSchema struct {
	Name             string                      `json:"name,omitempty"`
	Type             string                      `json:"type,omitempty"`
	Mode             string                      `json:"mode,omitempty"`
	Fields           []*BigQueryTableFieldSchema `json:"fields,omitempty"`
	RangeElementType *BigQueryFieldElementType   `json:"rangeElementType,omitempty"`
	Description      string                      `json:"description,omitempty"`
	MaxLength        string                      `json:"maxLength,omitempty"`
	Precision        string                      `json:"precision,omitempty"`
	Scale            string                      `json:"scale,omitempty"`
	Collation        string                      `json:"collation,omitempty"`
	DefaultValueExpr string                      `json:"defaultValueExpression,omitempty"`
	PolicyTags       *BigQueryPolicyTags         `json:"policyTags,omitempty"`
	DataPolicies     []*BigQueryDataPolicyOption `json:"dataPolicies,omitempty"`
	RoundingMode     string                      `json:"roundingMode,omitempty"`
}

// BigQueryFieldElementType mirrors the REST fieldElementType object used by
// RANGE fields.
type BigQueryFieldElementType struct {
	Type string `json:"type,omitempty"`
}

// BigQueryPolicyTags mirrors the REST policyTags object.
type BigQueryPolicyTags struct {
	Names []string `json:"names,omitempty"`
}

// BigQueryDataPolicyOption mirrors the REST dataPolicies entry shape.
type BigQueryDataPolicyOption struct {
	Name string `json:"name,omitempty"`
}

// BuildBigQueryGoogleSQLCatalogFromDDL analyzes BigQuery DDL and registers its
// tables and views in a GoogleSQL frontend catalog.
func BuildBigQueryGoogleSQLCatalogFromDDL(path, ddlSQL string, options ...AnalyzerOption) (*BigQueryGoogleSQLCatalog, error) {
	config := defaultAnalyzerConfig()
	for _, option := range options {
		option(&config)
	}
	if err := InitGoogleSQL(); err != nil {
		return nil, err
	}
	tf, err := googlesql.NewTypeFactory()
	if err != nil {
		return nil, err
	}
	simpleCatalog, opts, err := newGoogleSQLAnalyzerObjects("bigquery", tf, config)
	if err != nil {
		return nil, err
	}
	catalog := &BigQueryGoogleSQLCatalog{
		SimpleCatalog:   simpleCatalog,
		simpleCatalogs:  map[string]*googlesql.SimpleCatalog{"": simpleCatalog},
		AnalyzerOptions: opts,
		TypeFactory:     tf,
	}
	if strings.TrimSpace(ddlSQL) == "" {
		return catalog, nil
	}
	if err := catalog.applyDDL(path, ddlSQL); err != nil {
		return nil, err
	}
	return catalog, nil
}

// NewBigQueryAnalyzerFromDDL builds a BigQuery analyzer from BigQuery DDL.
func NewBigQueryAnalyzerFromDDL(path, ddlSQL string, options ...AnalyzerOption) (*BigQueryAnalyzer, error) {
	catalog, err := BuildBigQueryGoogleSQLCatalogFromDDL(path, ddlSQL, options...)
	if err != nil {
		return nil, err
	}
	return NewBigQueryAnalyzerFromGoogleSQLCatalog(catalog)
}

// NewBigQueryAnalyzerFromGoogleSQLCatalog wraps an existing BigQuery
// GoogleSQL catalog.
func NewBigQueryAnalyzerFromGoogleSQLCatalog(catalog *BigQueryGoogleSQLCatalog) (*BigQueryAnalyzer, error) {
	if catalog == nil {
		return nil, fmt.Errorf("nil BigQuery GoogleSQL catalog")
	}
	analyzer := &BigQueryAnalyzer{
		googleSQL: catalog,
		helper:    catalog.Helper(),
	}
	if err := catalog.addExternalQueryTVF(); err != nil {
		return nil, err
	}
	return analyzer, nil
}

// SetExternalQueryAnalyzers sets Spanner analyzers keyed by BigQuery
// connection ID for EXTERNAL_QUERY result schema inference.
func (a *BigQueryAnalyzer) SetExternalQueryAnalyzers(analyzers map[string]*Analyzer) {
	a.externalQueryAnalyzers = analyzers
	if a != nil && a.googleSQL != nil {
		a.googleSQL.externalQueryAnalyzers = analyzers
	}
}

func (a *BigQueryAnalyzer) AddQueryParameter(name string, spec *TypeSpec) error {
	if a == nil || a.googleSQL == nil {
		return fmt.Errorf("nil BigQuery analyzer")
	}
	typ, err := typeSpecToGoogleSQLTypeWithProto(a.googleSQL.TypeFactory, spec, nil, nil)
	if err != nil {
		return err
	}
	if err := a.googleSQL.AnalyzerOptions.SetParameterMode(googlesql.ParameterModeParameterNamed); err != nil {
		return err
	}
	return a.googleSQL.AnalyzerOptions.AddQueryParameter(name, typ)
}

// AddSpannerExternalDataset registers a BigQuery-visible projection of Spanner
// tables under dataset. It models BigQuery Spanner external datasets for static
// analysis only; it does not create BigQuery datasets or Spanner connections.
func (a *BigQueryAnalyzer) AddSpannerExternalDataset(dataset, spannerSource string, catalog *Catalog) (*BigQuerySpannerExternalDatasetBinding, error) {
	if a == nil || a.googleSQL == nil {
		return nil, fmt.Errorf("nil BigQuery analyzer")
	}
	return a.googleSQL.addSpannerExternalDataset(dataset, spannerSource, catalog, BigQuerySpannerExternalDatasetOptions{})
}

// AddSpannerExternalDatasetWithOptions registers a BigQuery-visible projection
// of Spanner tables under dataset with explicit static-analysis metadata.
func (a *BigQueryAnalyzer) AddSpannerExternalDatasetWithOptions(dataset, spannerSource string, catalog *Catalog, options BigQuerySpannerExternalDatasetOptions) (*BigQuerySpannerExternalDatasetBinding, error) {
	if a == nil || a.googleSQL == nil {
		return nil, fmt.Errorf("nil BigQuery analyzer")
	}
	return a.googleSQL.addSpannerExternalDataset(dataset, spannerSource, catalog, options)
}

type BigQuerySpannerExternalDatasetOptions struct {
	Project                  string
	DefaultProject           string
	ExternalSource           string
	Location                 string
	Connection               string
	CloudResourceConnection  string
	Access                   string
	DatabaseRole             string
	DatabaseRoleSource       string
	AccessVerificationStatus string
	AccessVerificationSource string
	AccessVerificationHint   string
	AccessVerification       BigQuerySpannerExternalDatasetVerification
	UnsupportedColumns       string
	NamedSchemaPolicy        string
}

type BigQuerySpannerExternalDatasetBinding struct {
	Kind                            string                                                         `json:"kind" yaml:"kind"`
	Name                            string                                                         `json:"name,omitempty" yaml:"name,omitempty"`
	BigQueryDataset                 string                                                         `json:"bigquery_dataset" yaml:"bigquery_dataset"`
	BigQueryDatasetRef              BigQueryDatasetReference                                       `json:"bigquery_dataset_ref" yaml:"bigquery_dataset_ref"`
	SpannerSource                   string                                                         `json:"spanner_source" yaml:"spanner_source"`
	SourceSchema                    string                                                         `json:"source_schema" yaml:"source_schema"`
	ExternalSource                  string                                                         `json:"external_source,omitempty" yaml:"external_source,omitempty"`
	Location                        string                                                         `json:"location,omitempty" yaml:"location,omitempty"`
	LocationMetadata                *BigQuerySpannerExternalDatasetLocationMetadata                `json:"location_metadata,omitempty" yaml:"location_metadata,omitempty"`
	Connection                      string                                                         `json:"connection,omitempty" yaml:"connection,omitempty"`
	CloudResourceConnection         string                                                         `json:"cloud_resource_connection,omitempty" yaml:"cloud_resource_connection,omitempty"`
	CloudResourceConnectionMetadata *BigQuerySpannerExternalDatasetCloudResourceConnectionMetadata `json:"cloud_resource_connection_metadata,omitempty" yaml:"cloud_resource_connection_metadata,omitempty"`
	Access                          string                                                         `json:"access_mode,omitempty" yaml:"access_mode,omitempty"`
	DatabaseRole                    string                                                         `json:"database_role,omitempty" yaml:"database_role,omitempty"`
	DatabaseRoleSource              string                                                         `json:"database_role_source,omitempty" yaml:"database_role_source,omitempty"`
	AccessVerification              BigQuerySpannerExternalDatasetVerification                     `json:"access_verification" yaml:"access_verification"`
	RequiresDataBoostAccess         bool                                                           `json:"requires_databoost_access" yaml:"requires_databoost_access"`
	ProjectionPolicy                BigQuerySpannerExternalDatasetProjectionPolicy                 `json:"projection_policy" yaml:"projection_policy"`
	ProjectionMatrix                BigQuerySpannerExternalDatasetProjectionMatrix                 `json:"projection_matrix" yaml:"projection_matrix"`
	Execution                       BigQuerySpannerExternalDatasetExecution                        `json:"execution" yaml:"execution"`
	Limitations                     []string                                                       `json:"limitations,omitempty" yaml:"limitations,omitempty"`
	Warnings                        []QueryCodegenPlanWarning                                      `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	VetSuppressions                 []QueryCodegenPlanVetSuppression                               `json:"vet_suppressions,omitempty" yaml:"vet_suppressions,omitempty"`
	ProjectedTables                 []BigQuerySpannerExternalDatasetTable                          `json:"projected_tables,omitempty" yaml:"projected_tables,omitempty"`
}

type BigQuerySpannerExternalDatasetExecution struct {
	DataBoost     string `json:"data_boost" yaml:"data_boost"`
	QueryPriority string `json:"query_priority" yaml:"query_priority"`
	Writable      bool   `json:"writable" yaml:"writable"`
}

type BigQuerySpannerExternalDatasetVerification struct {
	Status                           string `json:"status" yaml:"status"`
	Source                           string `json:"source" yaml:"source"`
	ConfiguredHint                   string `json:"configured_hint,omitempty" yaml:"configured_hint,omitempty"`
	CheckedAt                        string `json:"checked_at,omitempty" yaml:"checked_at,omitempty"`
	Verifier                         string `json:"verifier,omitempty" yaml:"verifier,omitempty"`
	EvidenceDigest                   string `json:"evidence_digest,omitempty" yaml:"evidence_digest,omitempty"`
	IndependentlyVerifiedByGenerator bool   `json:"independently_verified_by_generator" yaml:"independently_verified_by_generator"`
	Volatile                         bool   `json:"volatile,omitempty" yaml:"volatile,omitempty"`
}

type BigQueryDatasetReference struct {
	Project string `json:"project,omitempty" yaml:"project,omitempty"`
	Dataset string `json:"dataset" yaml:"dataset"`
	Path    string `json:"path" yaml:"path"`
}

type BigQuerySpannerExternalDatasetProjectionPolicy struct {
	UnsupportedColumns string `json:"unsupported_columns" yaml:"unsupported_columns"`
	NamedSchemas       string `json:"named_schemas" yaml:"named_schemas"`
}

type BigQuerySpannerExternalDatasetProjectionMatrix struct {
	Source                 string                                              `json:"source" yaml:"source"`
	SourceURL              string                                              `json:"source_url,omitempty" yaml:"source_url,omitempty"`
	DocsLastUpdated        string                                              `json:"docs_last_updated,omitempty" yaml:"docs_last_updated,omitempty"`
	DocsLastChecked        string                                              `json:"docs_last_checked,omitempty" yaml:"docs_last_checked,omitempty"`
	GeneratorMatrixVersion int                                                 `json:"generator_matrix_version" yaml:"generator_matrix_version"`
	Rows                   []BigQuerySpannerExternalDatasetProjectionMatrixRow `json:"rows,omitempty" yaml:"rows,omitempty"`
}

type BigQuerySpannerExternalDatasetProjectionMatrixRow struct {
	Object          string `json:"object" yaml:"object"`
	Behavior        string `json:"behavior" yaml:"behavior"`
	DefaultSeverity string `json:"default_severity" yaml:"default_severity"`
	EvidenceKind    string `json:"evidence_kind" yaml:"evidence_kind"`
	SourceURL       string `json:"source_url,omitempty" yaml:"source_url,omitempty"`
}

type BigQuerySpannerExternalDatasetLocationMetadata struct {
	Configured string `json:"configured,omitempty" yaml:"configured,omitempty"`
	Canonical  string `json:"canonical,omitempty" yaml:"canonical,omitempty"`
}

type BigQuerySpannerExternalDatasetCloudResourceConnectionMetadata struct {
	ID                      string `json:"id,omitempty" yaml:"id,omitempty"`
	ParsedLocation          string `json:"parsed_location,omitempty" yaml:"parsed_location,omitempty"`
	ParsedLocationCanonical string `json:"parsed_location_canonical,omitempty" yaml:"parsed_location_canonical,omitempty"`
	LocationMatch           bool   `json:"location_match,omitempty" yaml:"location_match,omitempty"`
}

func spannerExternalDatasetProjectionMatrixRows() []BigQuerySpannerExternalDatasetProjectionMatrixRow {
	sourceURL := "https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets"
	return []BigQuerySpannerExternalDatasetProjectionMatrixRow{
		{Object: "default_schema_table", Behavior: "project", DefaultSeverity: "ok", EvidenceKind: "documented", SourceURL: sourceURL},
		{Object: "named_schema_table", Behavior: "omit", DefaultSeverity: "warning", EvidenceKind: "documented", SourceURL: sourceURL},
		{Object: "unsupported_visible_column", Behavior: "omit", DefaultSeverity: "warning", EvidenceKind: "documented", SourceURL: sourceURL},
		{Object: "hidden_column", Behavior: "omit", DefaultSeverity: "info", EvidenceKind: "generator_safety_policy"},
		{Object: "primary_or_foreign_key_metadata", Behavior: "preserve_as_underlying_provenance_only", DefaultSeverity: "info", EvidenceKind: "documented", SourceURL: sourceURL},
		{Object: "dml_or_metadata_target", Behavior: "reject", DefaultSeverity: "error", EvidenceKind: "documented", SourceURL: sourceURL},
		{Object: "case_folded_table_collision", Behavior: "reject", DefaultSeverity: "error", EvidenceKind: "generator_safety_policy"},
		{Object: "case_folded_column_collision", Behavior: "reject", DefaultSeverity: "error", EvidenceKind: "generator_safety_policy"},
		{Object: "timestamp_runtime_precision", Behavior: "project_with_caveat", DefaultSeverity: "warning", EvidenceKind: "currently_inferred"},
		{Object: "numeric_runtime_loader", Behavior: "project_with_caveat", DefaultSeverity: "warning", EvidenceKind: "currently_inferred"},
	}
}

type BigQuerySpannerExternalDatasetTable struct {
	Name                       string                                 `json:"name" yaml:"name"`
	BigQueryTable              string                                 `json:"bigquery_table" yaml:"bigquery_table"`
	SourceTable                string                                 `json:"source_table" yaml:"source_table"`
	SpannerTable               string                                 `json:"spanner_table" yaml:"spanner_table"`
	Visible                    bool                                   `json:"visible" yaml:"visible"`
	VisibleInBigQueryMetadata  bool                                   `json:"visible_in_bigquery_metadata" yaml:"visible_in_bigquery_metadata"`
	BigQueryKeyMetadataVisible bool                                   `json:"bigquery_key_metadata_visible" yaml:"bigquery_key_metadata_visible"`
	Reason                     string                                 `json:"reason,omitempty" yaml:"reason,omitempty"`
	NameMatching               string                                 `json:"name_matching,omitempty" yaml:"name_matching,omitempty"`
	Columns                    []BigQuerySpannerExternalDatasetColumn `json:"columns,omitempty" yaml:"columns,omitempty"`
}

type BigQuerySpannerExternalDatasetColumn struct {
	Name                        string `json:"name" yaml:"name"`
	SpannerType                 string `json:"spanner_type,omitempty" yaml:"spanner_type,omitempty"`
	BigQueryType                string `json:"bigquery_type,omitempty" yaml:"bigquery_type,omitempty"`
	Visible                     bool   `json:"visible" yaml:"visible"`
	VisibleInBigQueryMetadata   bool   `json:"visible_in_bigquery_metadata" yaml:"visible_in_bigquery_metadata"`
	UnderlyingSpannerPrimaryKey bool   `json:"underlying_spanner_primary_key,omitempty" yaml:"underlying_spanner_primary_key,omitempty"`
	BigQueryKeyMetadataVisible  bool   `json:"bigquery_key_metadata_visible" yaml:"bigquery_key_metadata_visible"`
	Reason                      string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// Helper returns a reusable GoogleSQL helper for this catalog.
func (c *BigQueryGoogleSQLCatalog) Helper() *GoogleSQLHelper {
	return &GoogleSQLHelper{
		Catalog:     c.SimpleCatalog,
		Options:     c.AnalyzerOptions,
		TypeFactory: c.TypeFactory,
	}
}

func (c *BigQueryGoogleSQLCatalog) applyDDL(path, ddlSQL string) error {
	location, err := googlesql.NewParseResumeLocationFromString2(path, ddlSQL)
	if err != nil {
		return err
	}
	for {
		if atEnd, err := parseResumeLocationAtEnd(location); err != nil {
			return err
		} else if atEnd {
			return nil
		}
		out, err := googlesql.AnalyzeNextStatement(location, c.AnalyzerOptions, c.SimpleCatalog, c.TypeFactory)
		if err != nil {
			return err
		}
		if out == nil {
			return nil
		}
		if err := c.applyDDLAnalyzerOutput(out); err != nil {
			return err
		}
	}
}

func parseResumeLocationAtEnd(location *googlesql.ParseResumeLocation) (bool, error) {
	input, err := location.Input()
	if err != nil {
		return false, err
	}
	position, err := location.BytePosition()
	if err != nil {
		return false, err
	}
	if position < 0 || int(position) > len(input) {
		return false, fmt.Errorf("invalid parse resume byte position %d", position)
	}
	remaining := strings.TrimSpace(input[position:])
	return remaining == "" || remaining == ";", nil
}

func (c *BigQueryGoogleSQLCatalog) applyDDLAnalyzerOutput(out *googlesql.AnalyzerOutput) error {
	stmt, err := out.ResolvedStatement()
	if err != nil {
		return err
	}
	switch stmt := stmt.(type) {
	case *googlesql.ResolvedCreateTableStmt:
		return c.addResolvedTable(stmt)
	case *googlesql.ResolvedCreateTableAsSelectStmt:
		return c.addResolvedTable(stmt)
	case *googlesql.ResolvedCreateExternalTableStmt:
		return c.addResolvedTable(stmt)
	case *googlesql.ResolvedCreateViewStmt:
		return c.addResolvedView(stmt)
	case *googlesql.ResolvedCreateSnapshotTableStmt:
		// Snapshot tables inherit schema from the source table. For now, we
		// can just ignore the DDL if it doesn't break analysis.
		return nil
	case *googlesql.ResolvedCreateTableFunctionStmt:
		// Table functions are not yet modeled as TVFs in the catalog, but we
		// can at least ignore the DDL if it doesn't break analysis.
		return nil
	case *googlesql.ResolvedExportDataStmt:
		// EXPORT DATA does not create a persistent table.
		return nil
	case *googlesql.ResolvedCreateProcedureStmt, *googlesql.ResolvedExecuteImmediateStmt:
		// Procedural statements are out of scope for schema analysis.
		return nil
	case *googlesql.ResolvedCreateSchemaStmt:
		return nil
	case *googlesql.ResolvedCreateIndexStmt:
		return nil
	case *googlesql.ResolvedDropStmt:
		return nil
	default:
		kind, _ := stmt.NodeKindString()
		if kind == "" {
			kind = fmt.Sprintf("%T", stmt)
		}
		return fmt.Errorf("unsupported BigQuery DDL statement %s", kind)
	}
}

type resolvedCreateTable interface {
	NamePath() ([]string, error)
	ColumnDefinitionList() ([]*googlesql.ResolvedColumnDefinition, error)
}

func (c *BigQueryGoogleSQLCatalog) addResolvedTable(stmt resolvedCreateTable) error {
	namePath, err := stmt.NamePath()
	if err != nil {
		return err
	}
	columns, err := stmt.ColumnDefinitionList()
	if err != nil {
		return err
	}
	return c.addTableFromColumnDefinitions(ObjectName{Parts: namePath}, columns)
}

func (c *BigQueryGoogleSQLCatalog) addResolvedView(stmt *googlesql.ResolvedCreateViewStmt) error {
	namePath, err := stmt.NamePath()
	if err != nil {
		return err
	}
	columns, err := stmt.ColumnDefinitionList()
	if err != nil {
		return err
	}
	return c.addTableFromColumnDefinitions(ObjectName{Parts: namePath}, columns)
}

func (c *BigQueryGoogleSQLCatalog) addTableFromColumnDefinitions(name ObjectName, columns []*googlesql.ResolvedColumnDefinition) error {
	tableName := name.String()
	parentCatalog, leafName, err := simpleCatalogForObjectName(c.SimpleCatalog, c.simpleCatalogs, name)
	if err != nil {
		return err
	}
	gsTable, err := googlesql.NewSimpleTable(leafName, 0)
	if err != nil {
		return err
	}
	if tableName != leafName {
		if err := gsTable.SetFullName(tableName); err != nil {
			return err
		}
	}
	if err := gsTable.SetAllowDuplicateColumnNames(true); err != nil {
		return err
	}
	if err := gsTable.SetAllowAnonymousColumnName(true); err != nil {
		return err
	}
	for _, column := range columns {
		columnName, err := column.Name()
		if err != nil {
			return err
		}
		columnType, err := column.Type()
		if err != nil {
			return err
		}
		hidden, err := column.IsHidden()
		if err != nil {
			return err
		}
		gsCol, err := googlesql.NewSimpleColumn(tableName, columnName, columnType, hidden, true)
		if err != nil {
			return err
		}
		if err := gsTable.AddColumn2(gsCol, true); err != nil {
			return err
		}
	}
	return parentCatalog.AddOwnedTable(gsTable)
}

func (c *BigQueryGoogleSQLCatalog) addSpannerExternalDataset(dataset, spannerSource string, catalog *Catalog, options BigQuerySpannerExternalDatasetOptions) (*BigQuerySpannerExternalDatasetBinding, error) {
	if strings.TrimSpace(dataset) == "" {
		return nil, fmt.Errorf("external dataset name is required")
	}
	if catalog == nil {
		return nil, fmt.Errorf("nil Spanner catalog")
	}
	datasetRef, err := normalizeBigQueryDatasetReference(dataset, options.Project, options.DefaultProject)
	if err != nil {
		return nil, err
	}
	access, err := normalizeSpannerExternalDatasetAccess(options.Access)
	if err != nil {
		return nil, err
	}
	unsupportedColumns, err := normalizeSpannerExternalDatasetUnsupportedColumnsPolicy(options.UnsupportedColumns)
	if err != nil {
		return nil, err
	}
	namedSchemas, err := normalizeSpannerExternalDatasetNamedSchemaPolicy(options.NamedSchemaPolicy)
	if err != nil {
		return nil, err
	}
	databaseRole, databaseRoleSource, err := externalDatasetDatabaseRole(options.ExternalSource, options.DatabaseRole, options.DatabaseRoleSource)
	if err != nil {
		return nil, err
	}
	accessVerification, err := normalizeSpannerExternalDatasetVerification(options)
	if err != nil {
		return nil, err
	}
	cloudResourceConnection := strings.TrimSpace(options.CloudResourceConnection)
	if cloudResourceConnection == "" {
		cloudResourceConnection = strings.TrimSpace(options.Connection)
	}
	location := canonicalBigQueryLocation(options.Location)
	var locationMetadata *BigQuerySpannerExternalDatasetLocationMetadata
	if strings.TrimSpace(options.Location) != "" {
		locationMetadata = &BigQuerySpannerExternalDatasetLocationMetadata{
			Configured: strings.TrimSpace(options.Location),
			Canonical:  location,
		}
	}
	binding := &BigQuerySpannerExternalDatasetBinding{
		Kind:                    "spanner_external_dataset",
		BigQueryDataset:         datasetRef.Path,
		BigQueryDatasetRef:      datasetRef,
		SpannerSource:           spannerSource,
		SourceSchema:            spannerSource,
		ExternalSource:          options.ExternalSource,
		Location:                location,
		LocationMetadata:        locationMetadata,
		Connection:              options.Connection,
		CloudResourceConnection: cloudResourceConnection,
		Access:                  access,
		DatabaseRole:            databaseRole,
		DatabaseRoleSource:      databaseRoleSource,
		AccessVerification:      accessVerification,
		RequiresDataBoostAccess: true,
		ProjectionPolicy: BigQuerySpannerExternalDatasetProjectionPolicy{
			UnsupportedColumns: unsupportedColumns,
			NamedSchemas:       namedSchemas,
		},
		ProjectionMatrix: BigQuerySpannerExternalDatasetProjectionMatrix{
			Source:                 "google_cloud_docs",
			SourceURL:              "https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets",
			DocsLastUpdated:        "2026-05-02",
			DocsLastChecked:        "2026-05-05",
			GeneratorMatrixVersion: 1,
			Rows:                   spannerExternalDatasetProjectionMatrixRows(),
		},
		Execution: BigQuerySpannerExternalDatasetExecution{
			DataBoost:     "forced",
			QueryPriority: "medium",
			Writable:      false,
		},
		Limitations: []string{
			"default_spanner_schema_only",
			"primary_and_foreign_keys_not_visible_to_bigquery",
			"unsupported_spanner_columns_not_visible",
			"no_dml",
			"no_read_api_or_write_api",
			"no_information_schema",
		},
		Warnings: []QueryCodegenPlanWarning{{
			Rule:        "external-dataset-data-boost-permission-note",
			Severity:    "warning",
			Message:     "BigQuery Spanner external dataset queries use Data Boost by default",
			Remediation: "Make sure the query principal or delegated connection service account has the required Spanner Data Boost read permissions.",
		}},
	}
	if accessVerification.Status == "not_checked" {
		binding.Warnings = append(binding.Warnings, QueryCodegenPlanWarning{
			Rule:        "external-dataset-access-unverified",
			Severity:    "warning",
			Message:     "BigQuery Spanner external dataset visibility is based on DDL and has not been verified against IAM, access mode, or database role permissions",
			Remediation: "Verify the BigQuery external dataset access mode, connection service account or end-user credentials, Spanner database role, and Data Boost permissions in infrastructure checks.",
		})
	}
	if accessVerification.Status == "mismatch" || accessVerification.Status == "failed" {
		binding.Warnings = append(binding.Warnings, QueryCodegenPlanWarning{
			Rule:        "external-dataset-access-verification-" + accessVerification.Status,
			Severity:    "warning",
			Message:     fmt.Sprintf("BigQuery Spanner external dataset access verification status is %s", accessVerification.Status),
			Remediation: "Review the live external dataset verification result before relying on this static catalog projection.",
		})
	}
	if databaseRole != "" && accessVerification.Status == "not_checked" {
		binding.Warnings = append(binding.Warnings, QueryCodegenPlanWarning{
			Rule:        "external-dataset-role-visibility-not-verified",
			Severity:    "warning",
			Message:     fmt.Sprintf("BigQuery Spanner external dataset projection uses database_role %q but static DDL projection is not role-filtered", databaseRole),
			Remediation: "Verify role-filtered visibility outside this generator or add a future role-based allowlist before relying on this projection for runtime permissions.",
		})
	}
	if externalSourceShouldWarnUnrecognized(options.ExternalSource) {
		binding.Warnings = append(binding.Warnings, QueryCodegenPlanWarning{
			Rule:        "external-dataset-external-source-unrecognized",
			Severity:    "warning",
			Message:     fmt.Sprintf("external_source %q is not recognized as google-cloudspanner://DATABASE_ROLE@/projects/... or google-cloudspanner:/projects/...", options.ExternalSource),
			Remediation: "The generator preserves unrecognized external_source values as opaque metadata and cannot infer database_role from them.",
		})
	}
	if err := addSpannerExternalDatasetAccessWarnings(binding, options.Location, cloudResourceConnection); err != nil {
		return nil, err
	}
	if err := validateSpannerExternalDatasetTableNames(catalog); err != nil {
		return nil, err
	}
	tableKeys := make([]string, 0, len(catalog.Tables))
	for key := range catalog.Tables {
		if isSpannerBuiltInTable(catalog.Tables[key]) {
			continue
		}
		tableKeys = append(tableKeys, key)
	}
	sort.Strings(tableKeys)
	for _, key := range tableKeys {
		table := catalog.Tables[key]
		if len(table.Name.Parts) != 1 {
			projected := skippedSpannerExternalDatasetTable(table, "named_spanner_schema_not_visible")
			if namedSchemas == "error" {
				return nil, queryCodegenDiagnosticError("external-dataset-named-schema-table-omitted", "external_dataset_projection", table.Name.String(), fmt.Sprintf("external dataset %s table %s: named Spanner schemas are not visible through BigQuery Spanner external datasets", datasetRef.Path, table.Name))
			}
			binding.Warnings = append(binding.Warnings, externalDatasetProjectionWarnings(projected)...)
			binding.ProjectedTables = append(binding.ProjectedTables, projected)
			continue
		}
		projected, err := c.addSpannerExternalDatasetTable(datasetRef, table)
		if err != nil {
			return nil, fmt.Errorf("external dataset %s table %s: %w", datasetRef.Path, table.Name, err)
		}
		if unsupportedColumns == "error" {
			if column := firstUnsupportedSpannerExternalDatasetColumn(projected); column != nil {
				return nil, queryCodegenDiagnosticError("external-dataset-unsupported-column-omitted", "external_dataset_projection", table.Name.String()+"."+column.Name, fmt.Sprintf("external dataset %s table %s column %s: unsupported Spanner column is not visible through BigQuery Spanner external datasets", datasetRef.Path, table.Name, column.Name))
			}
		}
		binding.Warnings = append(binding.Warnings, externalDatasetProjectionWarnings(projected)...)
		binding.ProjectedTables = append(binding.ProjectedTables, projected)
	}
	c.spannerExternalDatasetBindings = append(c.spannerExternalDatasetBindings, *binding)
	return binding, nil
}

func validateSpannerExternalDatasetTableNames(catalog *Catalog) error {
	seen := map[string]string{}
	for _, table := range catalog.Tables {
		if isSpannerBuiltInTable(table) {
			continue
		}
		if len(table.Name.Parts) != 1 {
			continue
		}
		key := strings.ToLower(table.Name.Parts[0])
		if previous := seen[key]; previous != "" {
			return fmt.Errorf("spanner external dataset table names are case-insensitive; tables %s and %s conflict", previous, table.Name.String())
		}
		seen[key] = table.Name.String()
	}
	return nil
}

func isSpannerBuiltInTable(table *Table) bool {
	if table == nil || len(table.Name.Parts) == 0 {
		return false
	}
	return strings.EqualFold(table.Name.Parts[0], informationSchemaName) || strings.EqualFold(table.Name.Parts[0], spannerSysName)
}

func externalDatasetDatabaseRole(externalSource, configuredRole, configuredSource string) (string, string, error) {
	sourceInfo := parseSpannerExternalSource(externalSource)
	if sourceInfo.Recognized && sourceInfo.DatabaseRole != "" && configuredRole != "" && sourceInfo.DatabaseRole != configuredRole {
		return "", "", queryCodegenDiagnosticError("external-dataset-database-role-conflict", "external_dataset_binding", "spanner_external_datasets.access.database_role", fmt.Sprintf("external_source database role %q conflicts with access.database_role %q", sourceInfo.DatabaseRole, configuredRole))
	}
	if sourceInfo.Recognized && sourceInfo.DatabaseRole != "" && configuredRole == "" {
		return sourceInfo.DatabaseRole, "external_source", nil
	}
	if configuredRole == "" {
		return "", "", nil
	}
	if configuredSource != "" {
		return configuredRole, configuredSource, nil
	}
	if sourceInfo.Recognized && sourceInfo.DatabaseRole != "" {
		return configuredRole, "config_and_external_source", nil
	}
	return configuredRole, "config", nil
}

type spannerExternalSourceInfo struct {
	Recognized   bool
	DatabaseRole string
}

func parseSpannerExternalSource(externalSource string) spannerExternalSourceInfo {
	externalSource = strings.TrimSpace(externalSource)
	if externalSource == "" {
		return spannerExternalSourceInfo{}
	}
	parsed, err := url.Parse(externalSource)
	if err != nil || !strings.EqualFold(parsed.Scheme, "google-cloudspanner") || !strings.HasPrefix(parsed.Path, "/projects/") {
		return spannerExternalSourceInfo{}
	}
	hasRole := parsed.User != nil && parsed.User.Username() != ""
	roleForm := hasRole && strings.HasPrefix(externalSource, "google-cloudspanner://") && parsed.Host == ""
	noRoleForm := !hasRole && strings.HasPrefix(externalSource, "google-cloudspanner:/projects/")
	if !roleForm && !noRoleForm {
		return spannerExternalSourceInfo{}
	}
	info := spannerExternalSourceInfo{Recognized: true}
	if hasRole {
		info.DatabaseRole = parsed.User.Username()
	}
	return info
}

func externalSourceShouldWarnUnrecognized(externalSource string) bool {
	return strings.TrimSpace(externalSource) != "" && !parseSpannerExternalSource(externalSource).Recognized
}

func normalizeSpannerExternalDatasetVerification(options BigQuerySpannerExternalDatasetOptions) (BigQuerySpannerExternalDatasetVerification, error) {
	verification := options.AccessVerification
	if verification.Status == "" {
		verification.Status = options.AccessVerificationStatus
	}
	if verification.Source == "" {
		verification.Source = options.AccessVerificationSource
	}
	if verification.ConfiguredHint == "" {
		verification.ConfiguredHint = options.AccessVerificationHint
	}
	status, err := normalizeSpannerExternalDatasetVerificationStatus(verification.Status)
	if err != nil {
		return BigQuerySpannerExternalDatasetVerification{}, err
	}
	verification.Status = status
	source, err := normalizeSpannerExternalDatasetVerificationSource(verification.Source, status)
	if err != nil {
		return BigQuerySpannerExternalDatasetVerification{}, err
	}
	if status == "verified" || status == "mismatch" || status == "failed" {
		if verification.Verifier == "" || verification.CheckedAt == "" {
			return BigQuerySpannerExternalDatasetVerification{}, fmt.Errorf("access verification status %q requires verifier and checked_at", status)
		}
	}
	verification.Source = source
	verification.IndependentlyVerifiedByGenerator = source == "live_probe"
	verification.Volatile = verification.CheckedAt != ""
	return verification, nil
}

func normalizeSpannerExternalDatasetVerificationStatus(status string) (string, error) {
	switch strings.ToLower(emptyDefault(status, "not_checked")) {
	case "not_checked":
		return "not_checked", nil
	case "verified":
		return "verified", nil
	case "mismatch":
		return "mismatch", nil
	case "failed":
		return "failed", nil
	default:
		return "", fmt.Errorf("unsupported access verification status %q; use not_checked, verified, mismatch, or failed", status)
	}
}

func normalizeSpannerExternalDatasetVerificationSource(source, status string) (string, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		if status == "" || status == "not_checked" {
			return "", nil
		}
		return "external_evidence", nil
	}
	switch source {
	case "user_config":
		if status != "" && status != "not_checked" {
			return "", fmt.Errorf("access verification source user_config is only valid with not_checked status")
		}
		return source, nil
	case "external_evidence", "live_probe":
		return source, nil
	default:
		return "", fmt.Errorf("unsupported access verification source %q; use user_config, external_evidence, or live_probe", source)
	}
}

func normalizeBigQueryDatasetReference(dataset, project, defaultProject string) (BigQueryDatasetReference, error) {
	parts := splitBigQueryPath(dataset)
	if len(parts) == 0 || len(parts) > 2 {
		return BigQueryDatasetReference{}, fmt.Errorf("external dataset name must be dataset or project.dataset, got %q", dataset)
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return BigQueryDatasetReference{}, fmt.Errorf("external dataset name must not contain empty path parts, got %q", dataset)
		}
	}
	project = strings.TrimSpace(project)
	defaultProject = strings.TrimSpace(defaultProject)
	ref := BigQueryDatasetReference{}
	if len(parts) == 2 {
		ref.Project = parts[0]
		ref.Dataset = parts[1]
		if project != "" && project != ref.Project {
			return BigQueryDatasetReference{}, fmt.Errorf("external dataset project %q conflicts with project-qualified dataset %q", project, dataset)
		}
	} else {
		ref.Project = emptyDefault(project, defaultProject)
		ref.Dataset = parts[0]
	}
	ref.Path = ref.Dataset
	if ref.Project != "" {
		ref.Path = ref.Project + "." + ref.Dataset
	}
	return ref, nil
}

func (r BigQueryDatasetReference) parts() []string {
	if r.Project == "" {
		return []string{r.Dataset}
	}
	return []string{r.Project, r.Dataset}
}

func normalizeSpannerExternalDatasetUnsupportedColumnsPolicy(policy string) (string, error) {
	switch strings.ToLower(emptyDefault(policy, "omit")) {
	case "omit":
		return "omit", nil
	case "error":
		return "error", nil
	default:
		return "", fmt.Errorf("unsupported unsupported_columns policy %q; use omit or error", policy)
	}
}

func normalizeSpannerExternalDatasetNamedSchemaPolicy(policy string) (string, error) {
	switch strings.ToLower(emptyDefault(policy, "warn_and_omit")) {
	case "warn_and_omit":
		return "warn_and_omit", nil
	case "error":
		return "error", nil
	default:
		return "", fmt.Errorf("unsupported named_schema_policy %q; use warn_and_omit or error", policy)
	}
}

func addSpannerExternalDatasetAccessWarnings(binding *BigQuerySpannerExternalDatasetBinding, location, connection string) error {
	connectionRef, ok, err := parseBigQueryConnectionReference(connection)
	if strings.TrimSpace(connection) != "" {
		binding.CloudResourceConnectionMetadata = &BigQuerySpannerExternalDatasetCloudResourceConnectionMetadata{ID: connection}
	}
	if err != nil {
		binding.Warnings = append(binding.Warnings, QueryCodegenPlanWarning{
			Rule:        "external-dataset-connection-id-unparsed",
			Severity:    "warning",
			Message:     fmt.Sprintf("BigQuery Spanner external dataset connection %q is not project.location.connection format", connection),
			Remediation: "Use project.location.connection format so the generator can validate location assumptions.",
		})
	}
	if ok {
		binding.CloudResourceConnectionMetadata.ParsedLocation = connectionRef.Location
		binding.CloudResourceConnectionMetadata.ParsedLocationCanonical = canonicalBigQueryLocation(connectionRef.Location)
	}
	if ok && location != "" && binding.CloudResourceConnectionMetadata.ParsedLocationCanonical != canonicalBigQueryLocation(location) {
		return queryCodegenDiagnosticError("external-dataset-connection-location-mismatch", "external_dataset_binding", "spanner_external_datasets.access.cloud_resource_connection", fmt.Sprintf("external dataset connection location %q conflicts with location %q", connectionRef.Location, location))
	}
	if ok && location != "" {
		binding.CloudResourceConnectionMetadata.LocationMatch = true
	}
	switch binding.Access {
	case "cloud_resource_connection":
		if connection == "" {
			binding.Warnings = append(binding.Warnings, QueryCodegenPlanWarning{
				Rule:        "external-dataset-cloud-resource-connection-missing",
				Severity:    "warning",
				Message:     "access.mode is cloud_resource_connection but connection metadata is not set",
				Remediation: "Set connection to project.location.connection when the external dataset uses a Cloud Resource connection.",
			})
		}
	case "end_user_credentials":
		if connection != "" {
			binding.Warnings = append(binding.Warnings, QueryCodegenPlanWarning{
				Rule:        "external-dataset-euc-connection-specified",
				Severity:    "warning",
				Message:     "access.mode is end_user_credentials but connection metadata is set",
				Remediation: "Remove connection metadata for end-user-credential access, or use cloud_resource_connection if a connection service account is intended.",
			})
		}
	}
	return nil
}

func canonicalBigQueryLocation(location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return ""
	}
	switch strings.ToLower(location) {
	case "us":
		return "US"
	case "eu":
		return "EU"
	default:
		return strings.ToLower(location)
	}
}

type bigQueryConnectionReference struct {
	Project    string
	Location   string
	Connection string
}

func parseBigQueryConnectionReference(connection string) (bigQueryConnectionReference, bool, error) {
	if strings.TrimSpace(connection) == "" {
		return bigQueryConnectionReference{}, false, nil
	}
	parts := splitBigQueryPath(connection)
	if len(parts) != 3 {
		return bigQueryConnectionReference{}, false, fmt.Errorf("invalid connection reference")
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return bigQueryConnectionReference{}, false, fmt.Errorf("invalid connection reference")
		}
	}
	return bigQueryConnectionReference{
		Project:    parts[0],
		Location:   parts[1],
		Connection: parts[2],
	}, true, nil
}

func skippedSpannerExternalDatasetTable(table *Table, reason string) BigQuerySpannerExternalDatasetTable {
	return BigQuerySpannerExternalDatasetTable{
		Name:         table.Name.String(),
		SourceTable:  table.Name.String(),
		SpannerTable: table.Name.String(),
		Visible:      false,
		Reason:       reason,
		NameMatching: "not_applicable",
	}
}

func firstUnsupportedSpannerExternalDatasetColumn(table BigQuerySpannerExternalDatasetTable) *BigQuerySpannerExternalDatasetColumn {
	for i := range table.Columns {
		column := &table.Columns[i]
		if column.Visible || column.Reason != "unsupported_bigquery_external_dataset_column" {
			continue
		}
		return column
	}
	return nil
}

func externalDatasetProjectionWarnings(table BigQuerySpannerExternalDatasetTable) []QueryCodegenPlanWarning {
	var warnings []QueryCodegenPlanWarning
	if !table.Visible {
		return []QueryCodegenPlanWarning{{
			Rule:        "external-dataset-named-schema-table-omitted",
			Severity:    "warning",
			Message:     fmt.Sprintf("table %s is not visible through the BigQuery Spanner external dataset projection: %s", table.SourceTable, table.Reason),
			Remediation: "External datasets expose only default Spanner schema tables. Move the table to the default schema or keep it out of BigQuery external dataset queries.",
		}}
	}
	for _, column := range table.Columns {
		if column.Visible {
			if warning, ok := externalDatasetVisibleColumnCaveat(table, column); ok {
				warnings = append(warnings, warning)
			}
			continue
		}
		rule := "external-dataset-column-omitted"
		severity := "warning"
		remediation := "Project only BigQuery-visible Spanner columns, or expose a supported scalar representation outside the external dataset."
		if column.Reason == "hidden_spanner_column_not_visible" {
			rule = "external-dataset-hidden-column"
			severity = "info"
			remediation = "Do not reference hidden Spanner columns through BigQuery Spanner external datasets."
		}
		if column.Reason == "unsupported_bigquery_external_dataset_column" {
			rule = "external-dataset-unsupported-column"
			remediation = "Project only BigQuery-supported Spanner columns, or expose a supported scalar representation outside the external dataset."
		}
		warnings = append(warnings, QueryCodegenPlanWarning{
			Rule:        rule,
			Severity:    severity,
			Message:     fmt.Sprintf("field %s.%s is not visible through the BigQuery Spanner external dataset projection: %s", table.SourceTable, column.Name, column.Reason),
			Remediation: remediation,
		})
	}
	return warnings
}

func externalDatasetVisibleColumnCaveat(table BigQuerySpannerExternalDatasetTable, column BigQuerySpannerExternalDatasetColumn) (QueryCodegenPlanWarning, bool) {
	switch column.SpannerType {
	case "TIMESTAMP":
		return QueryCodegenPlanWarning{
			Rule:        "external-dataset-timestamp-caveat",
			Severity:    "warning",
			Message:     fmt.Sprintf("field %s.%s is BigQuery-visible through an external dataset; verify TIMESTAMP precision behavior for generated consumers", table.SourceTable, column.Name),
			Remediation: "Keep TIMESTAMP precision assumptions explicit in downstream code until external-dataset-specific behavior is verified by docs or live tests.",
		}, true
	case "NUMERIC":
		return QueryCodegenPlanWarning{
			Rule:        "external-dataset-numeric-caveat",
			Severity:    "warning",
			Message:     fmt.Sprintf("field %s.%s is BigQuery-visible through an external dataset; verify NUMERIC range and loader behavior for generated consumers", table.SourceTable, column.Name),
			Remediation: "Keep NUMERIC range and loader assumptions explicit in downstream code until external-dataset-specific behavior is verified by docs or live tests.",
		}, true
	default:
		return QueryCodegenPlanWarning{}, false
	}
}

func splitBigQueryPath(path string) []string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(strings.TrimSuffix(path, "`"), "`")
	if path == "" {
		return nil
	}
	return strings.Split(path, ".")
}

func (c *BigQueryGoogleSQLCatalog) addSpannerExternalDatasetTable(datasetRef BigQueryDatasetReference, table *Table) (BigQuerySpannerExternalDatasetTable, error) {
	projected := BigQuerySpannerExternalDatasetTable{
		SourceTable:               table.Name.String(),
		SpannerTable:              table.Name.String(),
		Visible:                   true,
		VisibleInBigQueryMetadata: true,
		NameMatching:              "case_insensitive",
	}
	if len(table.Name.Parts) != 1 {
		projected.Name = table.Name.String()
		projected.Visible = false
		projected.VisibleInBigQueryMetadata = false
		projected.Reason = "named_spanner_schema_not_visible"
		return projected, nil
	}
	tableName := table.Name.Parts[0]
	projectedName := ObjectName{Parts: append(datasetRef.parts(), tableName)}
	projected.Name = projectedName.String()
	projected.BigQueryTable = projected.Name
	parentCatalog, leafName, err := simpleCatalogForObjectName(c.SimpleCatalog, c.simpleCatalogs, projectedName)
	if err != nil {
		return projected, err
	}
	gsTable, err := googlesql.NewSimpleTable(leafName, 0)
	if err != nil {
		return projected, err
	}
	if projected.Name != leafName {
		if err := gsTable.SetFullName(projected.Name); err != nil {
			return projected, err
		}
	}
	if err := ensureSimpleCatalogTableAbsent(parentCatalog, leafName, projected.Name); err != nil {
		return projected, err
	}
	if projected.Name != leafName {
		if err := ensureSimpleCatalogTableAbsent(c.SimpleCatalog, projected.Name, projected.Name); err != nil {
			return projected, err
		}
	}
	if err := gsTable.SetAllowDuplicateColumnNames(false); err != nil {
		return projected, err
	}
	if err := gsTable.SetAllowAnonymousColumnName(true); err != nil {
		return projected, err
	}
	if datasetRef.Project != "" {
		aliasName := ObjectName{Parts: []string{datasetRef.Dataset, tableName}}
		if aliasName.String() != projected.Name {
			aliasParentCatalog, aliasLeafName, err := simpleCatalogForObjectName(c.SimpleCatalog, c.simpleCatalogs, aliasName)
			if err != nil {
				return projected, err
			}
			if err := ensureSimpleCatalogTableAbsent(aliasParentCatalog, aliasLeafName, aliasName.String()); err != nil {
				return projected, err
			}
			if err := ensureSimpleCatalogTableAbsent(c.SimpleCatalog, aliasName.String(), aliasName.String()); err != nil {
				return projected, err
			}
		}
	}
	seenColumns := map[string]string{}
	for _, column := range table.Columns {
		projectedColumn, gsType, err := c.spannerExternalDatasetColumn(column)
		if err != nil {
			return projected, err
		}
		projectedColumn.UnderlyingSpannerPrimaryKey = column.PrimaryKey
		projected.Columns = append(projected.Columns, projectedColumn)
		if !projectedColumn.Visible {
			continue
		}
		columnKey := strings.ToLower(projectedColumn.Name)
		if previous := seenColumns[columnKey]; previous != "" {
			return projected, fmt.Errorf("external dataset projected column names are case-insensitive; table %s columns %s and %s conflict", table.Name.String(), previous, projectedColumn.Name)
		}
		seenColumns[columnKey] = projectedColumn.Name
		gsCol, err := googlesql.NewSimpleColumn(projected.Name, column.Name, gsType, false, true)
		if err != nil {
			return projected, err
		}
		if err := gsTable.AddColumn2(gsCol, true); err != nil {
			return projected, err
		}
	}
	hasAlias := projected.Name != leafName || datasetRef.Project != ""
	if hasAlias {
		if err := parentCatalog.AddTable(gsTable); err != nil {
			return projected, err
		}
		if projected.Name != leafName {
			if err := c.SimpleCatalog.AddTable2(projected.Name, gsTable); err != nil {
				return projected, err
			}
		}
		if datasetRef.Project != "" {
			aliasName := ObjectName{Parts: []string{datasetRef.Dataset, tableName}}
			if aliasName.String() != projected.Name {
				if err := c.addTableAlias(aliasName, gsTable); err != nil {
					return projected, err
				}
			}
		}
	} else {
		if err := parentCatalog.AddOwnedTable(gsTable); err != nil {
			return projected, err
		}
		if projected.Name != leafName {
			if err := c.SimpleCatalog.AddTable2(projected.Name, gsTable); err != nil {
				return projected, err
			}
		}
	}
	return projected, nil
}

func (c *BigQueryGoogleSQLCatalog) addTableAlias(name ObjectName, table *googlesql.SimpleTable) error {
	parentCatalog, _, err := simpleCatalogForObjectName(c.SimpleCatalog, c.simpleCatalogs, name)
	if err != nil {
		return err
	}
	if err := ensureSimpleCatalogTableAbsent(parentCatalog, name.Parts[len(name.Parts)-1], name.String()); err != nil {
		return err
	}
	if name.String() != name.Parts[len(name.Parts)-1] {
		if err := ensureSimpleCatalogTableAbsent(c.SimpleCatalog, name.String(), name.String()); err != nil {
			return err
		}
	}
	if err := parentCatalog.AddTable(table); err != nil {
		return err
	}
	return c.SimpleCatalog.AddTable2(name.String(), table)
}

func ensureSimpleCatalogTableAbsent(catalog *googlesql.SimpleCatalog, tableName, fullName string) error {
	exists, err := simpleCatalogHasTable(catalog, tableName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("bigquery catalog table collision for %s: existing source_kind=bigquery_ddl_or_catalog_table incoming source_kind=spanner_external_dataset_projection", fullName)
	}
	return nil
}

func simpleCatalogHasTable(catalog *googlesql.SimpleCatalog, tableName string) (bool, error) {
	names, err := catalog.TableNames()
	if err != nil {
		return false, err
	}
	for _, name := range names {
		if strings.EqualFold(name, tableName) {
			return true, nil
		}
	}
	return false, nil
}

func (c *BigQueryGoogleSQLCatalog) spannerExternalDatasetColumn(column *Column) (BigQuerySpannerExternalDatasetColumn, googlesql.Googlesql_TypeNode, error) {
	spannerType, err := typeSpecSQL(column.Type)
	if err != nil {
		spannerType = ""
	}
	projected := BigQuerySpannerExternalDatasetColumn{
		Name:                      column.Name,
		SpannerType:               spannerType,
		Visible:                   true,
		VisibleInBigQueryMetadata: true,
	}
	if column.Hidden {
		projected.Visible = false
		projected.VisibleInBigQueryMetadata = false
		projected.Reason = "hidden_spanner_column_not_visible"
		return projected, nil, nil
	}
	gsType, bigQueryType, reason, err := c.spannerExternalDatasetType(column.Type)
	if err != nil {
		return projected, nil, fmt.Errorf("column %s: %w", column.Name, err)
	}
	if reason != "" {
		projected.Visible = false
		projected.VisibleInBigQueryMetadata = false
		projected.Reason = reason
		return projected, nil, nil
	}
	projected.BigQueryType = bigQueryType
	return projected, gsType, nil
}

func (c *BigQueryGoogleSQLCatalog) spannerExternalDatasetType(spec *TypeSpec) (googlesql.Googlesql_TypeNode, string, string, error) {
	if spec == nil {
		return nil, "", "", fmt.Errorf("nil Spanner type")
	}
	if spec.Tokenlist {
		return nil, "", "unsupported_bigquery_external_dataset_column", nil
	}
	switch spec.Code {
	case spannerpb.TypeCode_BOOL:
		typ, err := c.TypeFactory.GetBool()
		return typ, "BOOL", "", err
	case spannerpb.TypeCode_INT64:
		typ, err := c.TypeFactory.GetInt64()
		return typ, "INT64", "", err
	case spannerpb.TypeCode_FLOAT32, spannerpb.TypeCode_FLOAT64:
		typ, err := c.TypeFactory.GetDouble()
		return typ, "FLOAT64", "", err
	case spannerpb.TypeCode_TIMESTAMP:
		typ, err := c.TypeFactory.GetTimestamp()
		return typ, "TIMESTAMP", "", err
	case spannerpb.TypeCode_DATE:
		typ, err := c.TypeFactory.GetDate()
		return typ, "DATE", "", err
	case spannerpb.TypeCode_STRING:
		typ, err := c.TypeFactory.GetString()
		return typ, "STRING", "", err
	case spannerpb.TypeCode_BYTES:
		typ, err := c.TypeFactory.GetBytes()
		return typ, "BYTES", "", err
	case spannerpb.TypeCode_NUMERIC:
		typ, err := c.TypeFactory.GetNumeric()
		return typ, "NUMERIC", "", err
	case spannerpb.TypeCode_JSON:
		typ, err := c.TypeFactory.GetJson()
		return typ, "JSON", "", err
	case spannerpb.TypeCode_ARRAY:
		elemType, elemSQL, reason, err := c.spannerExternalDatasetType(spec.ArrayElement)
		if err != nil || reason != "" {
			return nil, "", reason, err
		}
		typ, err := c.TypeFactory.MakeArrayType2(elemType)
		return typ, "ARRAY<" + elemSQL + ">", "", err
	case spannerpb.TypeCode_STRUCT, spannerpb.TypeCode_PROTO, spannerpb.TypeCode_ENUM, spannerpb.TypeCode_INTERVAL, spannerpb.TypeCode_UUID:
		return nil, "", "unsupported_bigquery_external_dataset_column", nil
	default:
		return nil, "", "unsupported_bigquery_external_dataset_column", nil
	}
}

// TableSchemaForStatement analyzes a query statement and returns a BigQuery
// REST TableSchema-shaped result schema.
func (a *BigQueryAnalyzer) TableSchemaForStatement(sql string) (*BigQueryTableSchema, error) {
	out, err := a.analyzeStatement(sql)
	if err != nil {
		return nil, err
	}
	return BigQueryTableSchemaFromAnalyzerOutput(out)
}

// TableSchemaForExpression analyzes a single expression and returns a
// one-field BigQuery TableSchema-shaped result schema.
func (a *BigQueryAnalyzer) TableSchemaForExpression(sql string) (*BigQueryTableSchema, error) {
	out, err := a.helper.AnalyzeExpression(sql)
	if err != nil {
		return nil, err
	}
	field, err := BigQueryTableFieldSchemaFromExpressionAnalyzerOutput("expression", out)
	if err != nil {
		return nil, err
	}
	return &BigQueryTableSchema{Fields: []*BigQueryTableFieldSchema{field}}, nil
}

// ParseDebugString returns the parser AST debug string.
func (a *BigQueryAnalyzer) ParseDebugString(sqlMode, sql string) (string, error) {
	return a.helper.ParseDebugString(sqlMode, sql)
}

// Unparse returns SQL generated from the parser AST.
func (a *BigQueryAnalyzer) Unparse(sqlMode, sql string) (string, error) {
	return a.helper.Unparse(sqlMode, sql)
}

// ResolvedASTDebugString returns the resolved AST debug string.
func (a *BigQueryAnalyzer) ResolvedASTDebugString(sqlMode, sql string) (string, error) {
	if sqlMode != "query" {
		return a.helper.ResolvedASTDebugString(sqlMode, sql)
	}
	out, err := a.analyzeStatement(sql)
	if err != nil {
		return "", err
	}
	node, err := out.ResolvedNode()
	if err != nil {
		return "", err
	}
	return node.DebugString()
}

func (a *BigQueryAnalyzer) analyzeStatement(sql string) (*googlesql.AnalyzerOutput, error) {
	if err := validateSpannerExternalDatasetRelationRoles(sql, a.googleSQL.spannerExternalDatasetBindings); err != nil {
		return nil, err
	}
	if a.googleSQL.externalQueryTVFRegistered {
		if err := a.prepareExternalQueryTVFCalls(sql); err != nil {
			return nil, err
		}
		defer a.googleSQL.clearExternalQueryTVFCalls()
		return a.helper.AnalyzeStatement(sql)
	}
	rewritten, err := a.rewriteExternalQueries(sql)
	if err != nil {
		return nil, err
	}
	return a.helper.AnalyzeStatement(rewritten)
}

func (a *BigQueryAnalyzer) rewriteExternalQueries(sql string) (string, error) {
	if len(a.externalQueryAnalyzers) == 0 {
		return sql, nil
	}
	calls, err := findExternalQueryCalls(sql)
	if err != nil {
		return "", err
	}
	if len(calls) == 0 {
		return sql, nil
	}
	var b strings.Builder
	last := 0
	for _, call := range calls {
		b.WriteString(sql[last:call.start])
		replacement, err := a.externalQueryReplacement(call.args)
		if err != nil {
			return "", err
		}
		b.WriteString(replacement)
		last = call.end
	}
	b.WriteString(sql[last:])
	return b.String(), nil
}

func (a *BigQueryAnalyzer) prepareExternalQueryTVFCalls(sql string) error {
	calls, err := findExternalQueryCalls(sql)
	if err != nil {
		return err
	}
	if a.googleSQL.externalQueryRowTypes == nil {
		a.googleSQL.externalQueryRowTypes = make(map[string]map[string]*spannerpb.StructType)
	}
	for _, call := range calls {
		rowType, err := a.externalQueryRowType(call.args)
		if err != nil {
			return err
		}
		connectionID, _ := decodeGoogleSQLStringLiteral(strings.TrimSpace(call.args[0]))
		spannerSQL, _ := decodeGoogleSQLStringLiteral(strings.TrimSpace(call.args[1]))
		if a.googleSQL.externalQueryRowTypes[connectionID] == nil {
			a.googleSQL.externalQueryRowTypes[connectionID] = make(map[string]*spannerpb.StructType)
		}
		a.googleSQL.externalQueryRowTypes[connectionID][spannerSQL] = rowType
	}
	return nil
}

func (c *BigQueryGoogleSQLCatalog) clearExternalQueryTVFCalls() {
	c.externalQueryRowTypes = nil
}

func (a *BigQueryAnalyzer) externalQueryRowType(args []string) (*spannerpb.StructType, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("EXTERNAL_QUERY requires 2 or 3 arguments, got %d", len(args))
	}
	if len(args) == 3 {
		return nil, fmt.Errorf("EXTERNAL_QUERY options argument is currently not supported for static analysis")
	}
	connectionID, err := decodeGoogleSQLStringLiteral(strings.TrimSpace(args[0]))
	if err != nil {
		return nil, fmt.Errorf("EXTERNAL_QUERY connection argument: %w", err)
	}
	spannerSQL, err := decodeGoogleSQLStringLiteral(strings.TrimSpace(args[1]))
	if err != nil {
		return nil, fmt.Errorf("EXTERNAL_QUERY SQL argument: %w", err)
	}
	analyzer, err := a.externalQueryAnalyzerForConnection(connectionID)
	if err != nil {
		return nil, err
	}
	rowType, err := analyzer.RowTypeForStatement(spannerSQL)
	if err != nil {
		return nil, fmt.Errorf("EXTERNAL_QUERY Spanner SQL: %w", err)
	}
	return rowType, nil
}

func (a *BigQueryAnalyzer) externalQueryReplacement(args []string) (string, error) {
	rowType, err := a.externalQueryRowType(args)
	if err != nil {
		return "", err
	}
	return bigQueryTypedEmptySubquery(rowType)
}

func (a *BigQueryAnalyzer) externalQueryAnalyzerForConnection(connectionID string) (*Analyzer, error) {
	if analyzer := a.externalQueryAnalyzers[connectionID]; analyzer != nil {
		return analyzer, nil
	}
	if a.googleSQL != nil {
		if analyzer := a.googleSQL.externalQueryAnalyzers[connectionID]; analyzer != nil {
			return analyzer, nil
		}
	}
	return nil, fmt.Errorf("no Spanner schema configured for EXTERNAL_QUERY connection %q", connectionID)
}

// BigQueryTableSchemaFromAnalyzerOutput converts a resolved query statement to
// a BigQuery REST TableSchema-shaped schema.
func BigQueryTableSchemaFromAnalyzerOutput(out *googlesql.AnalyzerOutput) (*BigQueryTableSchema, error) {
	stmt, err := out.ResolvedStatement()
	if err != nil {
		return nil, err
	}
	query, ok := stmt.(*googlesql.ResolvedQueryStmt)
	if !ok {
		return nil, ErrStatementHasNoRowType
	}
	return BigQueryTableSchemaFromResolvedQuery(query)
}

// BigQueryTableSchemaFromResolvedQuery converts a resolved query to a BigQuery
// REST TableSchema-shaped schema.
func BigQueryTableSchemaFromResolvedQuery(query *googlesql.ResolvedQueryStmt) (*BigQueryTableSchema, error) {
	n, err := query.OutputColumnListSize()
	if err != nil {
		return nil, err
	}
	fields := make([]*BigQueryTableFieldSchema, 0, n)
	for i := int32(0); i < n; i++ {
		outCol, err := query.OutputColumnList2(i)
		if err != nil {
			return nil, err
		}
		name, err := outCol.Name()
		if err != nil {
			return nil, err
		}
		resolvedCol, err := outCol.Column()
		if err != nil {
			return nil, err
		}
		typ, err := resolvedCol.Type()
		if err != nil {
			return nil, err
		}
		field, err := BigQueryTableFieldSchemaFromGoogleSQLType(name, typ)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	return &BigQueryTableSchema{Fields: fields}, nil
}

// BigQueryTableFieldSchemaFromExpressionAnalyzerOutput converts a resolved
// expression to a BigQuery REST TableFieldSchema-shaped field.
func BigQueryTableFieldSchemaFromExpressionAnalyzerOutput(name string, out *googlesql.AnalyzerOutput) (*BigQueryTableFieldSchema, error) {
	expr, err := out.ResolvedExpr()
	if err != nil {
		return nil, err
	}
	typ, err := expr.Type()
	if err != nil {
		return nil, err
	}
	return BigQueryTableFieldSchemaFromGoogleSQLType(name, typ)
}

// BigQueryTableFieldSchemaFromGoogleSQLType converts a GoogleSQL frontend type
// to a BigQuery REST TableFieldSchema-shaped field.
func BigQueryTableFieldSchemaFromGoogleSQLType(name string, typ googlesql.Googlesql_TypeNode) (*BigQueryTableFieldSchema, error) {
	field, err := bigQueryTableFieldSchemaFromGoogleSQLType(typ)
	if err != nil {
		return nil, err
	}
	field.Name = name
	return field, nil
}

func bigQueryTableFieldSchemaFromGoogleSQLType(typ googlesql.Googlesql_TypeNode) (*BigQueryTableFieldSchema, error) {
	if typ == nil {
		return nil, fmt.Errorf("nil GoogleSQL type")
	}
	kind, err := typ.Kind()
	if err != nil {
		return nil, err
	}
	switch kind {
	case googlesql.TypeKindTypeBool:
		return nullableBigQueryField("BOOLEAN"), nil
	case googlesql.TypeKindTypeInt64:
		return nullableBigQueryField("INTEGER"), nil
	case googlesql.TypeKindTypeFloat, googlesql.TypeKindTypeDouble:
		return nullableBigQueryField("FLOAT"), nil
	case googlesql.TypeKindTypeString:
		return nullableBigQueryField("STRING"), nil
	case googlesql.TypeKindTypeBytes:
		return nullableBigQueryField("BYTES"), nil
	case googlesql.TypeKindTypeTimestamp:
		return nullableBigQueryField("TIMESTAMP"), nil
	case googlesql.TypeKindTypeDate:
		return nullableBigQueryField("DATE"), nil
	case googlesql.TypeKindTypeTime:
		return nullableBigQueryField("TIME"), nil
	case googlesql.TypeKindTypeDatetime:
		return nullableBigQueryField("DATETIME"), nil
	case googlesql.TypeKindTypeGeography:
		return nullableBigQueryField("GEOGRAPHY"), nil
	case googlesql.TypeKindTypeNumeric:
		return nullableBigQueryField("NUMERIC"), nil
	case googlesql.TypeKindTypeBignumeric:
		return nullableBigQueryField("BIGNUMERIC"), nil
	case googlesql.TypeKindTypeJson:
		return nullableBigQueryField("JSON"), nil
	case googlesql.TypeKindTypeArray:
		arrayType, err := typ.AsArray()
		if err != nil {
			return nil, err
		}
		elemType, err := arrayType.ElementType()
		if err != nil {
			return nil, err
		}
		field, err := bigQueryTableFieldSchemaFromGoogleSQLType(elemType)
		if err != nil {
			return nil, err
		}
		field.Mode = "REPEATED"
		return field, nil
	case googlesql.TypeKindTypeStruct:
		structType, err := typ.AsStruct()
		if err != nil {
			return nil, err
		}
		structFields, err := structType.Fields()
		if err != nil {
			return nil, err
		}
		fields := make([]*BigQueryTableFieldSchema, 0, len(structFields))
		for _, structField := range structFields {
			field, err := BigQueryTableFieldSchemaFromGoogleSQLType(structField.Name, structField.Type_)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field)
		}
		return &BigQueryTableFieldSchema{Type: "RECORD", Mode: "NULLABLE", Fields: fields}, nil
	case googlesql.TypeKindTypeRange:
		rangeType, err := typ.AsRange()
		if err != nil {
			return nil, err
		}
		elemType, err := rangeType.ElementType()
		if err != nil {
			return nil, err
		}
		elemField, err := bigQueryTableFieldSchemaFromGoogleSQLType(elemType)
		if err != nil {
			return nil, err
		}
		if elemField.Type != "DATE" && elemField.Type != "DATETIME" && elemField.Type != "TIMESTAMP" {
			return nil, fmt.Errorf("unsupported BigQuery RANGE element type %s", elemField.Type)
		}
		return &BigQueryTableFieldSchema{
			Type: "RANGE",
			Mode: "NULLABLE",
			RangeElementType: &BigQueryFieldElementType{
				Type: elemField.Type,
			},
		}, nil
	default:
		debug, _ := typ.DebugString(false)
		return nil, fmt.Errorf("unsupported BigQuery TableSchema type kind %s (%s)", kind, debug)
	}
}

func nullableBigQueryField(typ string) *BigQueryTableFieldSchema {
	return &BigQueryTableFieldSchema{Type: typ, Mode: "NULLABLE"}
}

type externalQueryCall struct {
	start int
	end   int
	args  []string
}

func findExternalQueryCalls(sql string) ([]externalQueryCall, error) {
	var calls []externalQueryCall
	for i := 0; i < len(sql); {
		if next := skipGoogleSQLTrivia(sql, i); next != i {
			i = next
			continue
		}
		if sql[i] == '\'' || sql[i] == '"' {
			next, err := scanGoogleSQLString(sql, i)
			if err != nil {
				return nil, err
			}
			i = next
			continue
		}
		if !strings.HasPrefix(strings.ToUpper(sql[i:]), "EXTERNAL_QUERY") || !externalQueryBoundary(sql, i, i+len("EXTERNAL_QUERY")) {
			_, size := utf8.DecodeRuneInString(sql[i:])
			if size == 0 {
				size = 1
			}
			i += size
			continue
		}
		open := i + len("EXTERNAL_QUERY")
		for open < len(sql) && unicode.IsSpace(rune(sql[open])) {
			open++
		}
		if open >= len(sql) || sql[open] != '(' {
			i = open
			continue
		}
		close, args, err := parseExternalQueryArguments(sql, open)
		if err != nil {
			return nil, err
		}
		calls = append(calls, externalQueryCall{start: i, end: close + 1, args: args})
		i = close + 1
	}
	return calls, nil
}

func externalQueryBoundary(sql string, start, end int) bool {
	if start > 0 && isGoogleSQLIdentRune(rune(sql[start-1])) {
		return false
	}
	return end >= len(sql) || !isGoogleSQLIdentRune(rune(sql[end]))
}

func isGoogleSQLIdentRune(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func parseExternalQueryArguments(sql string, open int) (int, []string, error) {
	depth := 1
	argStart := open + 1
	var args []string
	for i := open + 1; i < len(sql); {
		if next := skipGoogleSQLTrivia(sql, i); next != i {
			i = next
			continue
		}
		switch sql[i] {
		case '\'', '"':
			next, err := scanGoogleSQLString(sql, i)
			if err != nil {
				return 0, nil, err
			}
			i = next
			continue
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				args = append(args, strings.TrimSpace(sql[argStart:i]))
				return i, args, nil
			}
		case ',':
			if depth == 1 {
				args = append(args, strings.TrimSpace(sql[argStart:i]))
				argStart = i + 1
			}
		}
		i++
	}
	return 0, nil, fmt.Errorf("unterminated EXTERNAL_QUERY call")
}

func skipGoogleSQLTrivia(sql string, i int) int {
	if strings.HasPrefix(sql[i:], "--") {
		if end := strings.IndexByte(sql[i:], '\n'); end >= 0 {
			return i + end + 1
		}
		return len(sql)
	}
	if strings.HasPrefix(sql[i:], "/*") {
		if end := strings.Index(sql[i+2:], "*/"); end >= 0 {
			return i + 2 + end + 2
		}
		return len(sql)
	}
	return i
}

func scanGoogleSQLString(sql string, start int) (int, error) {
	quote := sql[start]
	triple := strings.HasPrefix(sql[start:], strings.Repeat(string(quote), 3))
	if triple {
		endToken := strings.Repeat(string(quote), 3)
		if end := strings.Index(sql[start+3:], endToken); end >= 0 {
			return start + 3 + end + 3, nil
		}
		return 0, fmt.Errorf("unterminated triple-quoted string literal")
	}
	for i := start + 1; i < len(sql); i++ {
		if sql[i] == '\\' {
			i++
			continue
		}
		if sql[i] == quote {
			if i+1 < len(sql) && sql[i+1] == quote {
				i++
				continue
			}
			return i + 1, nil
		}
	}
	return 0, fmt.Errorf("unterminated string literal")
}

func decodeGoogleSQLStringLiteral(lit string) (string, error) {
	lit = strings.TrimSpace(lit)
	if len(lit) == 0 {
		return "", fmt.Errorf("empty string literal")
	}
	raw := false
	if len(lit) >= 2 && (lit[0] == 'r' || lit[0] == 'R') && (lit[1] == '\'' || lit[1] == '"') {
		raw = true
		lit = lit[1:]
	}
	if strings.HasPrefix(lit, "'''") || strings.HasPrefix(lit, `"""`) {
		quote := lit[:3]
		if !strings.HasSuffix(lit, quote) || len(lit) < 6 {
			return "", fmt.Errorf("invalid triple-quoted string literal")
		}
		body := lit[3 : len(lit)-3]
		if raw {
			return body, nil
		}
		return unescapeGoogleSQLString(body), nil
	}
	if lit[0] != '\'' && lit[0] != '"' {
		return "", fmt.Errorf("want string literal, got %q", lit)
	}
	if len(lit) < 2 || lit[len(lit)-1] != lit[0] {
		return "", fmt.Errorf("invalid string literal")
	}
	body := lit[1 : len(lit)-1]
	if raw {
		return body, nil
	}
	return unescapeGoogleSQLString(body), nil
}

var googleSQLSimpleEscapes = map[byte]byte{
	'a':  '\a',
	'b':  '\b',
	'f':  '\f',
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
	'v':  '\v',
	'\\': '\\',
	'?':  '?',
	'\'': '\'',
	'"':  '"',
	'`':  '`',
}

func unescapeGoogleSQLString(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' && i+1 < len(s) && s[i+1] == '\'' {
			b.WriteByte('\'')
			i++
			continue
		}
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		if repl, ok := googleSQLSimpleEscapes[s[i]]; ok {
			b.WriteByte(repl)
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func bigQueryTypedEmptySubquery(rowType *spannerpb.StructType) (string, error) {
	if rowType == nil {
		return "", fmt.Errorf("nil Spanner row type")
	}
	if len(rowType.Fields) == 0 {
		return "(SELECT 1 WHERE FALSE)", nil
	}
	cols := make([]string, 0, len(rowType.Fields))
	for _, field := range rowType.Fields {
		typ, err := bigQuerySQLTypeFromSpannerType(field.Type)
		if err != nil {
			return "", fmt.Errorf("field %s: %w", field.Name, err)
		}
		cols = append(cols, fmt.Sprintf("CAST(NULL AS %s) AS %s", typ, quoteBigQueryIdent(field.Name)))
	}
	return "(SELECT " + strings.Join(cols, ", ") + " LIMIT 0)", nil
}

var simpleBigQueryIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func quoteBigQueryIdent(name string) string {
	if simpleBigQueryIdentifier.MatchString(name) {
		return name
	}
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}
