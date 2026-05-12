package querygen

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

const QueryCodegenConfigVersionV1Alpha = "v1alpha"

const queryCodegenV1AlphaIdentifierPattern = `[A-Za-z_][A-Za-z0-9_]*`

var queryCodegenV1AlphaIdentifierRegexp = regexp.MustCompile(`^` + queryCodegenV1AlphaIdentifierPattern + `$`)

// ParseQueryCodegenConfigYAML parses the public spanner-query-gen YAML config.
// The command only accepts the v1alpha config shape. Tests and package-level
// helpers may still construct QueryCodegenConfig directly, but YAML input is the
// user-facing contract and must use version v1alpha.
func ParseQueryCodegenConfigYAML(data []byte) (QueryCodegenConfig, error) {
	var header struct {
		Version string `json:"version" yaml:"version"`
	}
	if err := yaml.Unmarshal(data, &header); err != nil {
		return QueryCodegenConfig{}, err
	}
	if strings.TrimSpace(header.Version) != QueryCodegenConfigVersionV1Alpha {
		return QueryCodegenConfig{}, fmt.Errorf("unsupported config version %q; use version: %s", header.Version, QueryCodegenConfigVersionV1Alpha)
	}
	if err := rejectV1AlphaLegacyTopLevelKeys(data); err != nil {
		return QueryCodegenConfig{}, err
	}
	var config QueryCodegenV1AlphaConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return QueryCodegenConfig{}, err
	}
	return config.Normalize()
}

type QueryCodegenV1AlphaConfig struct {
	Version  string                       `json:"version" yaml:"version"`
	Go       QueryCodegenV1AlphaGo        `json:"go" yaml:"go"`
	Emit     QueryCodegenV1AlphaEmit      `json:"emit" yaml:"emit"`
	Catalogs []QueryCodegenV1AlphaCatalog `json:"catalogs" yaml:"catalogs"`
	Queries  []QueryCodegenV1AlphaQuery   `json:"queries" yaml:"queries"`
	Writes   []QueryCodegenV1AlphaWrite   `json:"writes" yaml:"writes"`
	Rules    QueryCodegenV1AlphaRules     `json:"rules" yaml:"rules"`
}

type QueryCodegenV1AlphaGo struct {
	Package string `json:"package" yaml:"package"`
	Out     string `json:"out" yaml:"out"`
}

type QueryCodegenV1AlphaEmit struct {
	Spanner  QueryCodegenV1AlphaSpannerEmit  `json:"spanner" yaml:"spanner"`
	BigQuery QueryCodegenV1AlphaBigQueryEmit `json:"bigquery" yaml:"bigquery"`
}

type QueryCodegenV1AlphaSpannerEmit struct {
	QueryMethods bool `json:"query_methods" yaml:"query_methods"`
	Mutations    bool `json:"mutations" yaml:"mutations"`
	DML          bool `json:"dml" yaml:"dml"`
}

type QueryCodegenV1AlphaBigQueryEmit struct {
	RowLoader    bool `json:"row_loader" yaml:"row_loader"`
	TableSchema  bool `json:"table_schema" yaml:"table_schema"`
	QueryMethods bool `json:"query_methods" yaml:"query_methods"`
}

type QueryCodegenV1AlphaCatalog struct {
	Name                 string                            `json:"name" yaml:"name"`
	Kind                 string                            `json:"kind" yaml:"kind"`
	Dialect              string                            `json:"dialect,omitempty" yaml:"dialect,omitempty"`
	Project              string                            `json:"project,omitempty" yaml:"project,omitempty"`
	DDL                  string                            `json:"ddl,omitempty" yaml:"ddl,omitempty"`
	ProtoDescriptorFiles []string                          `json:"proto_descriptors,omitempty" yaml:"proto_descriptors,omitempty"`
	Bindings             QueryCodegenV1AlphaCatalogBinding `json:"bindings" yaml:"bindings"`
}

type QueryCodegenV1AlphaCatalogBinding struct {
	ExternalQueryConnections []QueryCodegenV1AlphaExternalQueryConnection `json:"external_query_connections" yaml:"external_query_connections"`
	SpannerExternalDatasets  []QueryCodegenV1AlphaSpannerExternalDataset  `json:"spanner_external_datasets" yaml:"spanner_external_datasets"`
}

type QueryCodegenV1AlphaExternalQueryConnection struct {
	Name           string `json:"name" yaml:"name"`
	ID             string `json:"id" yaml:"id"`
	SpannerCatalog string `json:"spanner_catalog" yaml:"spanner_catalog"`
}

type QueryCodegenV1AlphaSpannerExternalDataset struct {
	Name               string                                   `json:"name" yaml:"name"`
	Dataset            string                                   `json:"dataset" yaml:"dataset"`
	SpannerCatalog     string                                   `json:"spanner_catalog" yaml:"spanner_catalog"`
	SpannerDatabaseURI string                                   `json:"spanner_database_uri,omitempty" yaml:"spanner_database_uri,omitempty"`
	Location           string                                   `json:"location,omitempty" yaml:"location,omitempty"`
	Access             QueryCodegenV1AlphaExternalDatasetAccess `json:"access" yaml:"access"`
	Projection         QueryCodegenV1AlphaProjectionPolicy      `json:"projection" yaml:"projection"`
}

type QueryCodegenV1AlphaExternalDatasetAccess struct {
	CloudResourceConnectionID string                                         `json:"cloud_resource_connection_id,omitempty" yaml:"cloud_resource_connection_id,omitempty"`
	VerificationEvidence      QueryCodegenSpannerExternalDatasetVerification `json:"verification_evidence,omitempty" yaml:"verification_evidence,omitempty"`
}

type QueryCodegenV1AlphaProjectionPolicy struct {
	UnsupportedColumns string `json:"unsupported_columns,omitempty" yaml:"unsupported_columns,omitempty"`
	NamedSchemaTables  string `json:"named_schema_tables,omitempty" yaml:"named_schema_tables,omitempty"`
}

type QueryCodegenV1AlphaQuery struct {
	Name      string                         `json:"name" yaml:"name"`
	Catalog   string                         `json:"catalog" yaml:"catalog"`
	Kind      string                         `json:"kind" yaml:"kind"`
	SQL       string                         `json:"sql,omitempty" yaml:"sql,omitempty"`
	Table     string                         `json:"table,omitempty" yaml:"table,omitempty"`
	Index     string                         `json:"index,omitempty" yaml:"index,omitempty"`
	Binding   string                         `json:"binding,omitempty" yaml:"binding,omitempty"`
	InnerSQL  string                         `json:"inner_sql,omitempty" yaml:"inner_sql,omitempty"`
	OuterSQL  string                         `json:"outer_sql,omitempty" yaml:"outer_sql,omitempty"`
	KeyPrefix []string                       `json:"key_prefix,omitempty" yaml:"key_prefix,omitempty"`
	OrderBy   string                         `json:"order_by,omitempty" yaml:"order_by,omitempty"`
	Result    QueryCodegenV1AlphaQueryResult `json:"result" yaml:"result"`
	Params    []QueryCodegenParam            `json:"params,omitempty" yaml:"params,omitempty"`
}

type QueryCodegenV1AlphaQueryResult struct {
	Cardinality string                            `json:"cardinality,omitempty" yaml:"cardinality,omitempty"`
	Struct      string                            `json:"struct,omitempty" yaml:"struct,omitempty"`
	Required    QueryCodegenV1AlphaRequiredFields `json:"required,omitempty" yaml:"required,omitempty"`
}

type QueryCodegenV1AlphaRequiredFields struct {
	Policy string   `json:"policy,omitempty" yaml:"policy,omitempty"`
	Fields []string `json:"fields,omitempty" yaml:"fields,omitempty"`
}

type QueryCodegenV1AlphaWrite struct {
	Name      string                           `json:"name" yaml:"name"`
	Catalog   string                           `json:"catalog" yaml:"catalog"`
	Table     string                           `json:"table" yaml:"table"`
	Operation string                           `json:"operation" yaml:"operation"`
	Input     QueryCodegenV1AlphaInputStruct   `json:"input" yaml:"input"`
	Key       QueryCodegenV1AlphaColumnList    `json:"key" yaml:"key"`
	Insert    QueryCodegenV1AlphaWriteInsert   `json:"insert" yaml:"insert"`
	Update    QueryCodegenV1AlphaWriteUpdate   `json:"update" yaml:"update"`
	Conflict  QueryCodegenV1AlphaWriteConflict `json:"conflict" yaml:"conflict"`
}

type QueryCodegenV1AlphaInputStruct struct {
	Struct string
}

func (i *QueryCodegenV1AlphaInputStruct) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err == nil {
		i.Struct = value
		return nil
	}
	var object map[string]interface{}
	if err := unmarshal(&object); err == nil {
		return fmt.Errorf("input must be a scalar struct name")
	}
	return fmt.Errorf("input must be a scalar struct name")
}

type QueryCodegenV1AlphaColumnList struct {
	Columns []string
}

func (l *QueryCodegenV1AlphaColumnList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var values []string
	if err := unmarshal(&values); err == nil {
		l.Columns = values
		return nil
	}
	var object map[string]interface{}
	if err := unmarshal(&object); err == nil {
		return fmt.Errorf("column list must be a YAML sequence")
	}
	return fmt.Errorf("column list must be a YAML sequence")
}

type QueryCodegenV1AlphaWriteInsert struct {
	Columns []string `json:"columns,omitempty" yaml:"columns,omitempty"`
}

type QueryCodegenV1AlphaWriteUpdate struct {
	Columns          []string `json:"columns,omitempty" yaml:"columns,omitempty"`
	AllNonKeyColumns bool     `json:"all_non_key_columns,omitempty" yaml:"all_non_key_columns,omitempty"`
}

type QueryCodegenV1AlphaWriteConflict struct {
	Strategy string `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

type QueryCodegenV1AlphaRules struct {
	Suppressions []QueryCodegenV1AlphaRuleSuppression `json:"suppressions,omitempty" yaml:"suppressions,omitempty"`
}

type QueryCodegenV1AlphaRuleSuppression struct {
	Scope   string `json:"scope" yaml:"scope"`
	Rule    string `json:"rule" yaml:"rule"`
	Reason  string `json:"reason" yaml:"reason"`
	Owner   string `json:"owner,omitempty" yaml:"owner,omitempty"`
	Expires string `json:"expires,omitempty" yaml:"expires,omitempty"`
}

func (c QueryCodegenV1AlphaConfig) Normalize() (QueryCodegenConfig, error) {
	if strings.TrimSpace(c.Version) != QueryCodegenConfigVersionV1Alpha {
		return QueryCodegenConfig{}, fmt.Errorf("unsupported config version %q", c.Version)
	}
	if len(c.Catalogs) == 0 {
		return QueryCodegenConfig{}, fmt.Errorf("catalogs is required")
	}
	if len(c.Queries) == 0 && len(c.Writes) == 0 {
		return QueryCodegenConfig{}, fmt.Errorf("at least one query or write is required")
	}
	if c.Emit.Spanner.QueryMethods {
		return QueryCodegenConfig{}, fmt.Errorf("emit.spanner.query_methods is not supported in v1alpha yet")
	}
	if c.Emit.BigQuery.QueryMethods {
		return QueryCodegenConfig{}, fmt.Errorf("emit.bigquery.query_methods is not supported in v1alpha yet")
	}
	config := QueryCodegenConfig{
		Version: c.Version,
		Package: c.Go.Package,
		Out:     c.Go.Out,
		Client:  c.emitTarget(),
	}
	catalogNames := map[string]bool{}
	queryNames := map[string]bool{}
	writeNames := map[string]bool{}
	connectionBindings := map[string]QueryCodegenV1AlphaExternalQueryConnection{}
	externalDatasetKeys := map[string]int{}
	for _, catalog := range c.Catalogs {
		if err := requireUniqueV1AlphaName("catalog", catalog.Name, catalogNames); err != nil {
			return QueryCodegenConfig{}, err
		}
		schema, err := catalog.normalize()
		if err != nil {
			return QueryCodegenConfig{}, err
		}
		connectionNames := map[string]bool{}
		for _, connection := range catalog.Bindings.ExternalQueryConnections {
			if err := requireUniqueV1AlphaName("catalog "+catalog.Name+" external_query_connections", connection.Name, connectionNames); err != nil {
				return QueryCodegenConfig{}, err
			}
			if connection.Name == "" {
				return QueryCodegenConfig{}, fmt.Errorf("catalog %s external_query_connections entry: name is required", catalog.Name)
			}
			if connection.ID == "" {
				return QueryCodegenConfig{}, fmt.Errorf("catalog %s external_query_connections %s: id is required", catalog.Name, connection.Name)
			}
			if connection.SpannerCatalog == "" {
				return QueryCodegenConfig{}, fmt.Errorf("catalog %s external_query_connections %s: spanner_catalog is required", catalog.Name, connection.Name)
			}
			if err := requireV1AlphaIdentifierReference("catalog "+catalog.Name+" external_query_connections "+connection.Name+" spanner_catalog", connection.SpannerCatalog); err != nil {
				return QueryCodegenConfig{}, err
			}
			key := catalog.Name + "." + connection.Name
			connectionBindings[key] = QueryCodegenV1AlphaExternalQueryConnection{
				Name:           connection.Name,
				ID:             connection.ID,
				SpannerCatalog: connection.SpannerCatalog,
			}
			schema.ExternalQueryConnections = append(schema.ExternalQueryConnections, QueryCodegenExternalSchema{
				Connection:    connection.ID,
				SpannerSource: connection.SpannerCatalog,
				Schema:        connection.SpannerCatalog,
			})
		}
		datasetNames := map[string]bool{}
		for _, dataset := range catalog.Bindings.SpannerExternalDatasets {
			if err := requireUniqueV1AlphaName("catalog "+catalog.Name+" spanner_external_datasets", dataset.Name, datasetNames); err != nil {
				return QueryCodegenConfig{}, err
			}
			normalized, err := dataset.normalize(catalog.Name)
			if err != nil {
				return QueryCodegenConfig{}, err
			}
			externalDatasetKeys[catalog.Name+"."+dataset.Name] = len(schema.SpannerExternalDatasets)
			schema.SpannerExternalDatasets = append(schema.SpannerExternalDatasets, normalized)
		}
		config.Schemas = append(config.Schemas, schema)
	}
	for _, query := range c.Queries {
		if err := requireUniqueV1AlphaName("query", query.Name, queryNames); err != nil {
			return QueryCodegenConfig{}, err
		}
		normalized, err := query.normalize(connectionBindings)
		if err != nil {
			return QueryCodegenConfig{}, err
		}
		config.Queries = append(config.Queries, normalized)
	}
	for _, write := range c.Writes {
		if err := requireUniqueV1AlphaName("write", write.Name, writeNames); err != nil {
			return QueryCodegenConfig{}, err
		}
		normalized, err := write.normalize(c.Emit.Spanner)
		if err != nil {
			return QueryCodegenConfig{}, err
		}
		config.Writes = append(config.Writes, normalized)
	}
	if err := c.applyRuleSuppressions(&config, externalDatasetKeys); err != nil {
		return QueryCodegenConfig{}, err
	}
	return config, nil
}

func requireUniqueV1AlphaName(kind, name string, seen map[string]bool) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil
	}
	if !queryCodegenV1AlphaIdentifierRegexp.MatchString(trimmed) {
		return fmt.Errorf("%s name %q must match ^%s$", kind, trimmed, queryCodegenV1AlphaIdentifierPattern)
	}
	if seen[trimmed] {
		return fmt.Errorf("duplicate %s name %q", kind, trimmed)
	}
	seen[trimmed] = true
	return nil
}

func requireV1AlphaIdentifierReference(kind, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil
	}
	if !queryCodegenV1AlphaIdentifierRegexp.MatchString(trimmed) {
		return fmt.Errorf("%s reference %q must match ^%s$", kind, trimmed, queryCodegenV1AlphaIdentifierPattern)
	}
	return nil
}

func (c QueryCodegenV1AlphaConfig) emitTarget() GoStructTarget {
	spanner := c.Emit.Spanner.QueryMethods || c.Emit.Spanner.Mutations || c.Emit.Spanner.DML
	bigquery := c.Emit.BigQuery.RowLoader || c.Emit.BigQuery.TableSchema || c.Emit.BigQuery.QueryMethods
	switch {
	case spanner && bigquery:
		return GoStructTargetBoth
	case spanner:
		return GoStructTargetSpanner
	case bigquery:
		return GoStructTargetBigQuery
	default:
		return GoStructTargetBoth
	}
}

func (c QueryCodegenV1AlphaCatalog) normalize() (QueryCodegenSchema, error) {
	if c.Name == "" {
		return QueryCodegenSchema{}, fmt.Errorf("catalog name is required")
	}
	if err := validateV1AlphaCatalogShape(c); err != nil {
		return QueryCodegenSchema{}, err
	}
	dialect, err := normalizeV1AlphaCatalogDialect(c.Kind, c.Dialect)
	if err != nil {
		return QueryCodegenSchema{}, fmt.Errorf("catalog %s: %w", c.Name, err)
	}
	return QueryCodegenSchema{
		Name:                 c.Name,
		Dialect:              dialect,
		Project:              c.Project,
		DDL:                  c.DDL,
		ProtoDescriptorFiles: c.ProtoDescriptorFiles,
	}, nil
}

func validateV1AlphaCatalogShape(c QueryCodegenV1AlphaCatalog) error {
	kind := strings.ToLower(strings.TrimSpace(c.Kind))
	switch kind {
	case "spanner":
		if strings.TrimSpace(c.Project) != "" {
			return fmt.Errorf("catalog %s kind spanner: project is not supported", c.Name)
		}
		if len(c.Bindings.ExternalQueryConnections) > 0 || len(c.Bindings.SpannerExternalDatasets) > 0 {
			return fmt.Errorf("catalog %s kind spanner: bindings are only supported for kind: bigquery", c.Name)
		}
	case "bigquery":
		if len(c.ProtoDescriptorFiles) > 0 {
			return fmt.Errorf("catalog %s kind bigquery: proto_descriptors is only supported for kind: spanner", c.Name)
		}
	default:
		return fmt.Errorf("unsupported catalog kind %q; use spanner or bigquery", c.Kind)
	}
	return nil
}

func normalizeV1AlphaCatalogDialect(kind, dialect string) (string, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	dialect = strings.ToLower(strings.TrimSpace(dialect))
	switch kind {
	case "spanner":
		switch firstNonEmpty(dialect, "googlesql") {
		case "googlesql":
			return "spanner", nil
		case "postgresql":
			return "spanner_postgresql", nil
		default:
			return "", fmt.Errorf("unsupported Spanner dialect %q", dialect)
		}
	case "bigquery":
		switch firstNonEmpty(dialect, "googlesql") {
		case "googlesql":
			return "bigquery", nil
		default:
			return "", fmt.Errorf("unsupported BigQuery dialect %q", dialect)
		}
	default:
		return "", fmt.Errorf("unsupported catalog kind %q; use spanner or bigquery", kind)
	}
}

func (d QueryCodegenV1AlphaSpannerExternalDataset) normalize(catalogName string) (QueryCodegenSpannerExternalDataset, error) {
	if d.Name == "" {
		return QueryCodegenSpannerExternalDataset{}, fmt.Errorf("catalog %s spanner_external_datasets entry: name is required", catalogName)
	}
	if d.Dataset == "" {
		return QueryCodegenSpannerExternalDataset{}, fmt.Errorf("catalog %s spanner_external_datasets %s: dataset is required", catalogName, d.Name)
	}
	if d.SpannerCatalog == "" {
		return QueryCodegenSpannerExternalDataset{}, fmt.Errorf("catalog %s spanner_external_datasets %s: spanner_catalog is required", catalogName, d.Name)
	}
	if err := requireV1AlphaIdentifierReference("catalog "+catalogName+" spanner_external_datasets "+d.Name+" spanner_catalog", d.SpannerCatalog); err != nil {
		return QueryCodegenSpannerExternalDataset{}, err
	}
	if err := validateV1AlphaVerificationEvidenceSource(catalogName, d.Name, d.Access.VerificationEvidence.Source); err != nil {
		return QueryCodegenSpannerExternalDataset{}, err
	}
	unsupportedColumns, err := normalizeV1AlphaUnsupportedColumnPolicy(d.Projection.UnsupportedColumns)
	if err != nil {
		return QueryCodegenSpannerExternalDataset{}, err
	}
	namedSchemaPolicy, err := normalizeV1AlphaNamedSchemaPolicy(d.Projection.NamedSchemaTables)
	if err != nil {
		return QueryCodegenSpannerExternalDataset{}, err
	}
	access := QueryCodegenSpannerExternalDatasetAccess{
		CloudResourceConnection: d.Access.CloudResourceConnectionID,
		VerificationEvidence:    d.Access.VerificationEvidence,
	}
	if d.Access.CloudResourceConnectionID != "" {
		access.Mode = "cloud_resource_connection"
	}
	return QueryCodegenSpannerExternalDataset{
		Name:               d.Name,
		Dataset:            d.Dataset,
		SpannerSource:      d.SpannerCatalog,
		ExternalSource:     d.SpannerDatabaseURI,
		Location:           d.Location,
		Access:             access,
		UnsupportedColumns: unsupportedColumns,
		NamedSchemaPolicy:  namedSchemaPolicy,
	}, nil
}

func validateV1AlphaVerificationEvidenceSource(catalogName, datasetName, source string) error {
	if strings.TrimSpace(source) == "" {
		return nil
	}
	return fmt.Errorf("catalog %s spanner_external_datasets %s: access.verification_evidence.source is normalized by the plan; omit it from v1alpha config", catalogName, datasetName)
}

func normalizeV1AlphaUnsupportedColumnPolicy(policy string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case "", "reject":
		return "error", nil
	case "omit":
		return "omit", nil
	default:
		return "", fmt.Errorf("unsupported unsupported_columns policy %q; use reject or omit", policy)
	}
}

func normalizeV1AlphaNamedSchemaPolicy(policy string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case "", "reject":
		return "error", nil
	case "warn_and_omit":
		return "warn_and_omit", nil
	default:
		return "", fmt.Errorf("unsupported named_schema_tables policy %q; use reject or warn_and_omit", policy)
	}
}

var rawExternalQueryPattern = regexp.MustCompile(`(?is)\bexternal_query\s*\(`)

func (q QueryCodegenV1AlphaQuery) normalize(connections map[string]QueryCodegenV1AlphaExternalQueryConnection) (QueryCodegenQuery, error) {
	if err := validateV1AlphaQueryShape(q); err != nil {
		return QueryCodegenQuery{}, err
	}
	query := QueryCodegenQuery{
		Name:           q.Name,
		Kind:           strings.ToLower(strings.TrimSpace(q.Kind)),
		Catalog:        q.Catalog,
		Result:         q.Result.Cardinality,
		ResultStruct:   q.Result.Struct,
		Required:       append([]string(nil), q.Result.Required.Fields...),
		RequiredPolicy: q.Result.Required.Policy,
		Params:         append([]QueryCodegenParam(nil), q.Params...),
		KeyPrefix:      append([]string(nil), q.KeyPrefix...),
		OrderBy:        q.OrderBy,
	}
	switch strings.ToLower(strings.TrimSpace(q.Kind)) {
	case "sql":
		if rawExternalQueryPattern.MatchString(q.SQL) {
			return QueryCodegenQuery{}, fmt.Errorf("query %s: raw EXTERNAL_QUERY is not supported in v1alpha config; use kind: external_query", q.Name)
		}
		query.SQL = q.SQL
	case "table":
		query.Table = q.Table
	case "index":
		query.Index = q.Index
	case "external_query":
		key := q.Catalog + "." + q.Binding
		connection, ok := connections[key]
		if !ok {
			return QueryCodegenQuery{}, fmt.Errorf("query %s: unknown external_query binding %q for catalog %q", q.Name, q.Binding, q.Catalog)
		}
		query.Federated = QueryCodegenFederatedQuery{
			Connection:    connection.ID,
			SpannerSource: connection.SpannerCatalog,
			InnerSQL:      q.InnerSQL,
			OuterSQL:      q.OuterSQL,
		}
	default:
		return QueryCodegenQuery{}, fmt.Errorf("query %s: unsupported kind %q", q.Name, q.Kind)
	}
	return query, nil
}

func validateV1AlphaQueryShape(q QueryCodegenV1AlphaQuery) error {
	if strings.TrimSpace(q.Name) == "" {
		return fmt.Errorf("query name is required")
	}
	if strings.TrimSpace(q.Catalog) == "" {
		return fmt.Errorf("query %s: catalog is required", q.Name)
	}
	if err := requireV1AlphaIdentifierReference("query "+q.Name+" catalog", q.Catalog); err != nil {
		return err
	}
	if err := requireV1AlphaIdentifierReference("query "+q.Name+" binding", q.Binding); err != nil {
		return err
	}
	if strings.TrimSpace(q.Result.Struct) == "" {
		return fmt.Errorf("query %s result.struct is required", q.Name)
	}
	switch strings.ToLower(strings.TrimSpace(q.Result.Cardinality)) {
	case "", "one", "maybe_one", "many":
	default:
		return fmt.Errorf("query %s result.cardinality %q is not supported; use one, maybe_one, or many", q.Name, q.Result.Cardinality)
	}
	switch strings.ToLower(strings.TrimSpace(q.Result.Required.Policy)) {
	case "", "override", "strict":
	default:
		return fmt.Errorf("query %s result.required.policy %q is not supported; use override or strict", q.Name, q.Result.Required.Policy)
	}
	kind := strings.ToLower(strings.TrimSpace(q.Kind))
	checkForbidden := func(fields map[string]bool) error {
		present := map[string]bool{
			"sql":        strings.TrimSpace(q.SQL) != "",
			"table":      strings.TrimSpace(q.Table) != "",
			"index":      strings.TrimSpace(q.Index) != "",
			"binding":    strings.TrimSpace(q.Binding) != "",
			"inner_sql":  strings.TrimSpace(q.InnerSQL) != "",
			"outer_sql":  strings.TrimSpace(q.OuterSQL) != "",
			"key_prefix": len(q.KeyPrefix) > 0,
			"order_by":   strings.TrimSpace(q.OrderBy) != "",
		}
		for field := range fields {
			if present[field] {
				return fmt.Errorf("query %s kind %s: field %s is not supported", q.Name, kind, field)
			}
		}
		return nil
	}
	switch kind {
	case "sql":
		if strings.TrimSpace(q.SQL) == "" {
			return fmt.Errorf("query %s kind sql: sql is required", q.Name)
		}
		if err := checkForbidden(stringSet("table", "index", "binding", "inner_sql", "outer_sql", "key_prefix", "order_by")); err != nil {
			return err
		}
	case "table":
		if strings.TrimSpace(q.Table) == "" {
			return fmt.Errorf("query %s kind table: table is required", q.Name)
		}
		if err := checkForbidden(stringSet("sql", "index", "binding", "inner_sql", "outer_sql", "key_prefix")); err != nil {
			return err
		}
	case "index":
		if strings.TrimSpace(q.Index) == "" {
			return fmt.Errorf("query %s kind index: index is required", q.Name)
		}
		if err := checkForbidden(stringSet("sql", "table", "binding", "inner_sql", "outer_sql")); err != nil {
			return err
		}
	case "external_query":
		if strings.TrimSpace(q.Binding) == "" {
			return fmt.Errorf("query %s kind external_query: binding is required", q.Name)
		}
		if strings.TrimSpace(q.InnerSQL) == "" {
			return fmt.Errorf("query %s kind external_query: inner_sql is required", q.Name)
		}
		if err := checkForbidden(stringSet("sql", "table", "index", "key_prefix", "order_by")); err != nil {
			return err
		}
	default:
		return fmt.Errorf("query %s: unsupported kind %q", q.Name, q.Kind)
	}
	return nil
}

func (w QueryCodegenV1AlphaWrite) normalize(globalEmit QueryCodegenV1AlphaSpannerEmit) (QueryCodegenWrite, error) {
	operation := strings.ToLower(strings.TrimSpace(w.Operation))
	if err := validateV1AlphaWriteShape(w, operation); err != nil {
		return QueryCodegenWrite{}, err
	}
	if operation == "upsert" {
		operation = "insert_or_update"
	}
	if strings.TrimSpace(w.Conflict.Strategy) != "" {
		return QueryCodegenWrite{}, fmt.Errorf("write %s: conflict is not supported in v1alpha public YAML; operation: upsert implies insert_or_update semantics", w.Name)
	}
	updateColumns := append([]string(nil), w.Update.Columns...)
	if w.Update.AllNonKeyColumns {
		if len(updateColumns) > 0 {
			return QueryCodegenWrite{}, fmt.Errorf("write %s: update.all_non_key_columns cannot be combined with update.columns", w.Name)
		}
		updateColumns = []string{autoAllNonKeyColumns}
	}
	if (operation == "update" || operation == "insert_or_update") && len(updateColumns) == 0 {
		return QueryCodegenWrite{}, fmt.Errorf("write %s: update.columns is required; use update.all_non_key_columns: true to update every non-key column", w.Name)
	}
	methods := v1AlphaSpannerWriteMethods(globalEmit)
	if operation == "replace" {
		methods = v1AlphaMutationOnly(methods)
	}
	write := QueryCodegenWrite{
		Name:         w.Name,
		Catalog:      w.Catalog,
		Table:        w.Table,
		Operation:    operation,
		InputStruct:  w.Input.Struct,
		Keys:         append([]string(nil), w.Key.Columns...),
		Methods:      append([]string(nil), methods...),
		EmitExplicit: true,
	}
	switch operation {
	case "insert", "replace":
		write.Insert.Columns = append([]string(nil), w.Insert.Columns...)
	case "update", "insert_or_update":
		write.Update.Columns = updateColumns
		if operation == "insert_or_update" {
			write.Insert.Columns = append([]string(nil), w.Insert.Columns...)
			if err := validateV1AlphaUpsertInsertColumns(w); err != nil {
				return QueryCodegenWrite{}, err
			}
		}
	case "delete":
	default:
		return QueryCodegenWrite{}, fmt.Errorf("write %s: unsupported operation %q", w.Name, w.Operation)
	}
	return write, nil
}

func validateV1AlphaWriteShape(w QueryCodegenV1AlphaWrite, operation string) error {
	if strings.TrimSpace(w.Catalog) == "" {
		return fmt.Errorf("write %s: catalog is required", w.Name)
	}
	if err := requireV1AlphaIdentifierReference("write "+w.Name+" catalog", w.Catalog); err != nil {
		return err
	}
	hasInsert := len(w.Insert.Columns) > 0
	hasUpdateColumns := len(w.Update.Columns) > 0
	hasAllNonKeyUpdate := w.Update.AllNonKeyColumns
	hasUpdate := hasUpdateColumns || hasAllNonKeyUpdate
	hasConflict := strings.TrimSpace(w.Conflict.Strategy) != ""
	if hasConflict {
		return fmt.Errorf("write %s: conflict is not supported in v1alpha public YAML; operation: upsert implies insert_or_update semantics", w.Name)
	}
	switch operation {
	case "insert":
		if !hasInsert {
			return fmt.Errorf("write %s operation insert: insert.columns is required", w.Name)
		}
		if hasUpdate {
			return fmt.Errorf("write %s operation insert: update is not supported", w.Name)
		}
	case "update":
		if hasInsert {
			return fmt.Errorf("write %s operation update: insert is not supported", w.Name)
		}
		if hasUpdateColumns && hasAllNonKeyUpdate {
			return fmt.Errorf("write %s operation update: update.columns and update.all_non_key_columns are mutually exclusive", w.Name)
		}
		if !hasUpdate {
			return fmt.Errorf("write %s: update.columns is required; use update.all_non_key_columns: true to update every non-key column", w.Name)
		}
	case "upsert":
		if !hasInsert {
			return fmt.Errorf("write %s operation upsert: insert.columns is required", w.Name)
		}
		if !hasUpdateColumns {
			return fmt.Errorf("write %s operation upsert: update.columns is required", w.Name)
		}
		if hasAllNonKeyUpdate {
			return fmt.Errorf("write %s operation upsert: update.all_non_key_columns is not supported", w.Name)
		}
	case "replace":
		if !hasInsert {
			return fmt.Errorf("write %s operation replace: insert.columns is required", w.Name)
		}
		if hasUpdate {
			return fmt.Errorf("write %s operation replace: update is not supported", w.Name)
		}
	case "delete":
		if hasInsert {
			return fmt.Errorf("write %s operation delete: insert is not supported", w.Name)
		}
		if hasUpdate {
			return fmt.Errorf("write %s operation delete: update is not supported", w.Name)
		}
	default:
		return fmt.Errorf("write %s: unsupported operation %q", w.Name, w.Operation)
	}
	return nil
}

func v1AlphaSpannerWriteMethods(emit QueryCodegenV1AlphaSpannerEmit) []string {
	var methods []string
	if emit.Mutations {
		methods = append(methods, "mutation")
	}
	if emit.DML {
		methods = append(methods, "dml")
	}
	return methods
}

func v1AlphaMutationOnly(methods []string) []string {
	for _, method := range methods {
		if method == "mutation" {
			return []string{"mutation"}
		}
	}
	return nil
}

func validateV1AlphaUpsertInsertColumns(w QueryCodegenV1AlphaWrite) error {
	if len(w.Insert.Columns) == 0 {
		return fmt.Errorf("write %s: upsert insert.columns is required", w.Name)
	}
	required := map[string]string{}
	for _, name := range w.Key.Columns {
		required[strings.ToLower(name)] = name
	}
	for _, name := range w.Update.Columns {
		required[strings.ToLower(name)] = name
	}
	actual := map[string]string{}
	for _, name := range w.Insert.Columns {
		key := strings.ToLower(name)
		if actual[key] != "" {
			return fmt.Errorf("write %s: duplicate upsert insert.columns entry %s", w.Name, name)
		}
		actual[key] = name
		if required[key] == "" {
			return fmt.Errorf("write %s: operation upsert cannot represent insert-only non-key column %s; insert.columns must equal key plus update.columns", w.Name, name)
		}
	}
	missing := make([]string, 0)
	for key, name := range required {
		if actual[key] == "" {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("write %s: operation upsert requires insert.columns to include key plus update.columns; missing %s", w.Name, strings.Join(missing, ", "))
	}
	return nil
}

func (c QueryCodegenV1AlphaConfig) applyRuleSuppressions(config *QueryCodegenConfig, externalDatasetKeys map[string]int) error {
	queryByName := map[string]int{}
	for i, query := range config.Queries {
		queryByName[query.Name] = i
	}
	writeByName := map[string]int{}
	for i, write := range config.Writes {
		writeByName[write.Name] = i
	}
	schemaByName := map[string]int{}
	for i, schema := range config.Schemas {
		schemaByName[schema.Name] = i
	}
	for _, suppression := range c.Rules.Suppressions {
		disabled := QueryCodegenVetDisable{
			Rule:    suppression.Rule,
			Reason:  suppression.Reason,
			Owner:   suppression.Owner,
			Expires: suppression.Expires,
		}
		scope, name, ok := strings.Cut(suppression.Scope, "/")
		if !ok {
			return fmt.Errorf("rules.suppressions scope %q must be kind/name", suppression.Scope)
		}
		switch scope {
		case "query":
			index, ok := queryByName[name]
			if !ok {
				return fmt.Errorf("rules.suppressions scope %q references unknown query", suppression.Scope)
			}
			config.Queries[index].Vet.Disable = append(config.Queries[index].Vet.Disable, disabled)
		case "write":
			index, ok := writeByName[name]
			if !ok {
				return fmt.Errorf("rules.suppressions scope %q references unknown write", suppression.Scope)
			}
			config.Writes[index].Vet.Disable = append(config.Writes[index].Vet.Disable, disabled)
		case "catalog-binding":
			catalogName, bindingName, ok := strings.Cut(name, ".")
			if !ok {
				return fmt.Errorf("rules.suppressions catalog-binding scope %q must be catalog.binding", suppression.Scope)
			}
			schemaIndex, ok := schemaByName[catalogName]
			if !ok {
				return fmt.Errorf("rules.suppressions scope %q references unknown catalog", suppression.Scope)
			}
			bindingIndex, ok := externalDatasetKeys[catalogName+"."+bindingName]
			if !ok {
				return fmt.Errorf("rules.suppressions scope %q references unknown catalog binding", suppression.Scope)
			}
			config.Schemas[schemaIndex].SpannerExternalDatasets[bindingIndex].Vet.Disable = append(config.Schemas[schemaIndex].SpannerExternalDatasets[bindingIndex].Vet.Disable, disabled)
		default:
			return fmt.Errorf("rules.suppressions scope %q has unsupported kind %q", suppression.Scope, scope)
		}
	}
	return nil
}

func rejectV1AlphaLegacyTopLevelKeys(data []byte) error {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	legacy := map[string]string{
		"package": "go.package",
		"out":     "go.out",
		"client":  "emit",
		"schemas": "catalogs",
		"source":  "catalog",
	}
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if replacement, ok := legacy[key]; ok {
			return fmt.Errorf("unsupported v1alpha field %q; use %q", key, replacement)
		}
	}
	if err := rejectV1AlphaLegacyObjectKeys(raw["queries"], "queries", map[string]string{
		"source":          "catalog",
		"schema":          "catalog",
		"federated":       "kind: external_query",
		"result_struct":   "result.struct",
		"required":        "result.required.fields",
		"required_policy": "result.required.policy",
		"vet":             "rules.suppressions",
	}); err != nil {
		return err
	}
	if err := rejectV1AlphaLegacyObjectKeys(raw["writes"], "writes", map[string]string{
		"source":       "catalog",
		"schema":       "catalog",
		"input_struct": "input",
		"columns":      "insert.columns or update.columns",
		"update_mask":  "update.columns",
		"methods":      "global emit.spanner",
		"emit":         "global emit.spanner",
		"vet":          "rules.suppressions",
	}); err != nil {
		return err
	}
	if err := rejectV1AlphaLegacyCatalogBindingKeys(raw["catalogs"]); err != nil {
		return err
	}
	if err := rejectV1AlphaLegacyQueryResultScalars(raw["queries"]); err != nil {
		return err
	}
	if err := rejectV1AlphaRulesKeys(raw["rules"]); err != nil {
		return err
	}
	if err := rejectV1AlphaUnknownKeys(raw); err != nil {
		return err
	}
	return nil
}

func rejectV1AlphaLegacyCatalogBindingKeys(value interface{}) error {
	catalogs, ok := value.([]interface{})
	if !ok {
		return nil
	}
	for i, item := range catalogs {
		catalog, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		bindings, ok := catalog["bindings"].(map[string]interface{})
		if !ok {
			continue
		}
		connections, ok := bindings["external_query_connections"].([]interface{})
		if ok {
			for j, connectionItem := range connections {
				connection, ok := connectionItem.(map[string]interface{})
				if !ok {
					continue
				}
				if _, ok := connection["connection_id"]; ok {
					return fmt.Errorf("unsupported v1alpha field %q at catalogs[%d].bindings.external_query_connections[%d]; use %q", "connection_id", i, j, "id")
				}
			}
		}
		datasets, ok := bindings["spanner_external_datasets"].([]interface{})
		if !ok {
			continue
		}
		for j, datasetItem := range datasets {
			dataset, ok := datasetItem.(map[string]interface{})
			if !ok {
				continue
			}
			legacy := map[string]string{
				"spanner_source":      "spanner_catalog",
				"external_source":     "spanner_database_uri",
				"connection":          "access.cloud_resource_connection_id",
				"cloud_resource":      "access.cloud_resource_connection_id",
				"access_verification": "access.verification_evidence",
				"verification_hint":   "access.verification_evidence",
				"vet":                 "rules.suppressions",
			}
			for key, replacement := range legacy {
				if _, ok := dataset[key]; ok {
					return fmt.Errorf("unsupported v1alpha field %q at catalogs[%d].bindings.spanner_external_datasets[%d]; use %q", key, i, j, replacement)
				}
			}
			access, ok := dataset["access"].(map[string]interface{})
			if !ok {
				continue
			}
			if evidence, ok := access["verification_evidence"].(map[string]interface{}); ok {
				if _, ok := evidence["source"]; ok {
					datasetName, _ := dataset["name"].(string)
					return fmt.Errorf("catalog %s spanner_external_datasets %s: access.verification_evidence.source is normalized by the plan; omit it from v1alpha config", catalogName(catalog), datasetName)
				}
			}
			accessLegacy := map[string]string{
				"mode":              "access.cloud_resource_connection_id",
				"database_role":     "spanner_database_uri role component",
				"verification_hint": "access.verification_evidence",
			}
			for key, replacement := range accessLegacy {
				if _, ok := access[key]; ok {
					return fmt.Errorf("unsupported v1alpha field %q at catalogs[%d].bindings.spanner_external_datasets[%d].access; use %q", key, i, j, replacement)
				}
			}
		}
	}
	return nil
}

func rejectV1AlphaUnknownKeys(raw map[string]interface{}) error {
	if err := rejectV1AlphaUnknownObjectKeys(raw, "", stringSet("version", "go", "emit", "catalogs", "queries", "writes", "rules")); err != nil {
		return err
	}
	if err := rejectV1AlphaUnknownObjectKeys(raw["go"], "go", stringSet("package", "out")); err != nil {
		return err
	}
	if err := rejectV1AlphaUnknownObjectKeys(raw["emit"], "emit", stringSet("spanner", "bigquery")); err != nil {
		return err
	}
	emit, _ := raw["emit"].(map[string]interface{})
	if err := rejectV1AlphaUnknownObjectKeys(emit["spanner"], "emit.spanner", stringSet("mutations", "dml", "query_methods")); err != nil {
		return err
	}
	if err := rejectV1AlphaUnknownObjectKeys(emit["bigquery"], "emit.bigquery", stringSet("row_loader", "table_schema", "query_methods")); err != nil {
		return err
	}
	if err := rejectV1AlphaUnknownCatalogKeys(raw["catalogs"]); err != nil {
		return err
	}
	if err := rejectV1AlphaUnknownQueryKeys(raw["queries"]); err != nil {
		return err
	}
	if err := rejectV1AlphaUnknownWriteKeys(raw["writes"]); err != nil {
		return err
	}
	if err := rejectV1AlphaUnknownRulesKeys(raw["rules"]); err != nil {
		return err
	}
	return nil
}

func rejectV1AlphaUnknownCatalogKeys(value interface{}) error {
	catalogs, ok := value.([]interface{})
	if !ok {
		return nil
	}
	for i, item := range catalogs {
		scope := fmt.Sprintf("catalogs[%d]", i)
		catalog, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if err := rejectV1AlphaUnknownObjectKeys(catalog, scope, stringSet("name", "kind", "dialect", "project", "ddl", "proto_descriptors", "bindings")); err != nil {
			return err
		}
		if err := rejectV1AlphaUnknownObjectKeys(catalog["bindings"], scope+".bindings", stringSet("external_query_connections", "spanner_external_datasets")); err != nil {
			return err
		}
		bindings, _ := catalog["bindings"].(map[string]interface{})
		if err := rejectV1AlphaUnknownArrayObjectKeys(bindings["external_query_connections"], scope+".bindings.external_query_connections", stringSet("name", "id", "spanner_catalog")); err != nil {
			return err
		}
		datasets, _ := bindings["spanner_external_datasets"].([]interface{})
		for j, datasetItem := range datasets {
			datasetScope := fmt.Sprintf("%s.bindings.spanner_external_datasets[%d]", scope, j)
			dataset, ok := datasetItem.(map[string]interface{})
			if !ok {
				continue
			}
			if err := rejectV1AlphaUnknownObjectKeys(dataset, datasetScope, stringSet("name", "dataset", "spanner_catalog", "spanner_database_uri", "location", "access", "projection")); err != nil {
				return err
			}
			if err := rejectV1AlphaUnknownObjectKeys(dataset["access"], datasetScope+".access", stringSet("cloud_resource_connection_id", "verification_evidence")); err != nil {
				return err
			}
			access, _ := dataset["access"].(map[string]interface{})
			if err := rejectV1AlphaUnknownObjectKeys(access["verification_evidence"], datasetScope+".access.verification_evidence", stringSet("status", "verifier", "checked_at")); err != nil {
				return err
			}
			if err := rejectV1AlphaUnknownObjectKeys(dataset["projection"], datasetScope+".projection", stringSet("unsupported_columns", "named_schema_tables")); err != nil {
				return err
			}
		}
	}
	return nil
}

func rejectV1AlphaUnknownQueryKeys(value interface{}) error {
	queries, ok := value.([]interface{})
	if !ok {
		return nil
	}
	for i, item := range queries {
		scope := fmt.Sprintf("queries[%d]", i)
		query, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if err := rejectV1AlphaUnknownObjectKeys(query, scope, stringSet("name", "catalog", "kind", "sql", "table", "index", "binding", "inner_sql", "outer_sql", "key_prefix", "order_by", "result", "params")); err != nil {
			return err
		}
		if result, ok := query["result"].(map[string]interface{}); ok {
			if err := rejectV1AlphaUnknownObjectKeys(result, scope+".result", stringSet("cardinality", "struct", "required")); err != nil {
				return err
			}
			if err := rejectV1AlphaUnknownObjectKeys(result["required"], scope+".result.required", stringSet("policy", "fields")); err != nil {
				return err
			}
		}
		if err := rejectV1AlphaUnknownArrayObjectKeys(query["params"], scope+".params", stringSet("name", "type", "scope", "optional", "choices", "default")); err != nil {
			return err
		}
	}
	return nil
}

func rejectV1AlphaUnknownWriteKeys(value interface{}) error {
	writes, ok := value.([]interface{})
	if !ok {
		return nil
	}
	for i, item := range writes {
		scope := fmt.Sprintf("writes[%d]", i)
		write, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if err := rejectV1AlphaUnknownObjectKeys(write, scope, stringSet("name", "catalog", "table", "operation", "input", "key", "insert", "update")); err != nil {
			return err
		}
		if err := rejectV1AlphaUnknownObjectKeys(write["insert"], scope+".insert", stringSet("columns")); err != nil {
			return err
		}
		if err := rejectV1AlphaUnknownObjectKeys(write["update"], scope+".update", stringSet("columns", "all_non_key_columns")); err != nil {
			return err
		}
	}
	return nil
}

func rejectV1AlphaUnknownRulesKeys(value interface{}) error {
	if err := rejectV1AlphaUnknownObjectKeys(value, "rules", stringSet("suppressions")); err != nil {
		return err
	}
	rules, _ := value.(map[string]interface{})
	return rejectV1AlphaUnknownArrayObjectKeys(rules["suppressions"], "rules.suppressions", stringSet("scope", "rule", "reason", "owner", "expires"))
}

func rejectV1AlphaUnknownArrayObjectKeys(value interface{}, scope string, allowed map[string]bool) error {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	for i, item := range items {
		if err := rejectV1AlphaUnknownObjectKeys(item, fmt.Sprintf("%s[%d]", scope, i), allowed); err != nil {
			return err
		}
	}
	return nil
}

func rejectV1AlphaUnknownObjectKeys(value interface{}, scope string, allowed map[string]bool) error {
	object, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !allowed[key] {
			if scope == "" {
				return fmt.Errorf("unsupported v1alpha field %q", key)
			}
			return fmt.Errorf("unsupported v1alpha field %q at %s", key, scope)
		}
	}
	return nil
}

func stringSet(values ...string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func catalogName(catalog map[string]interface{}) string {
	name, _ := catalog["name"].(string)
	return name
}

func rejectV1AlphaLegacyObjectKeys(value interface{}, scope string, legacy map[string]string) error {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	for i, item := range items {
		object, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if replacement, ok := legacy[key]; ok {
				return fmt.Errorf("unsupported v1alpha field %q at %s[%d]; use %q", key, scope, i, replacement)
			}
		}
	}
	return nil
}

func rejectV1AlphaLegacyQueryResultScalars(value interface{}) error {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	for i, item := range items {
		object, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		result, ok := object["result"]
		if !ok || result == nil {
			continue
		}
		if _, ok := result.(map[string]interface{}); !ok {
			return fmt.Errorf("unsupported v1alpha scalar field %q at queries[%d]; use %q", "result", i, "result.cardinality")
		}
	}
	return nil
}

func rejectV1AlphaRulesKeys(value interface{}) error {
	rules, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	if _, ok := rules["severity"]; ok {
		return fmt.Errorf("unsupported v1alpha field %q at rules; severity overrides are future vet policy", "severity")
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
