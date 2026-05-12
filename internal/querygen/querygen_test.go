package querygen

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestParseQueryCodegenConfigYAMLV1AlphaMinimal(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
emit:
  spanner:
    mutations: true
    dml: true
catalogs:
- name: app
  kind: spanner
  ddl: spanner.sql
queries:
- name: ListSingers
  catalog: app
  kind: table
  table: Singers
  result:
    cardinality: many
    struct: SingerRow
writes:
- name: UpdateSingerName
  catalog: app
  table: Singers
  operation: update
  input: SingerWrite
  key:
  - SingerId
  update:
    columns:
    - FirstName
    - LastName
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v", err)
	}
	if config.Package != "db" || config.Client != GoStructTargetSpanner {
		t.Fatalf("config package/client = %q/%q, want db/spanner", config.Package, config.Client)
	}
	if got, want := config.Schemas[0].Name, "app"; got != want {
		t.Fatalf("schema name = %q, want %q", got, want)
	}
	if got, want := config.Queries[0].Catalog, "app"; got != want {
		t.Fatalf("query catalog = %q, want %q", got, want)
	}
	if got, want := config.Queries[0].ResultStruct, "SingerRow"; got != want {
		t.Fatalf("query result struct = %q, want %q", got, want)
	}
	if got, want := strings.Join(config.Writes[0].Update.Columns, ","), "FirstName,LastName"; got != want {
		t.Fatalf("write update columns = %q, want %q", got, want)
	}
	code, err := GenerateQueryCode(config, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	for _, want := range []string{
		"type SingerRow struct",
		"type SingerWrite struct",
		"func (w *SingerWrite) UpdateSingerNameMutation() (*spanner.Mutation, error)",
		"func (w *SingerWrite) UpdateSingerNameDMLStatement() spanner.Statement",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaExternalQuery(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
emit:
  bigquery:
    table_schema: true
catalogs:
- name: app
  kind: spanner
  ddl: spanner.sql
- name: analytics
  kind: bigquery
  bindings:
    external_query_connections:
    - name: app_conn
      id: example-project.us.example-connection
      spanner_catalog: app
queries:
- name: ExternalQuerySingerIDs
  catalog: analytics
  kind: external_query
  binding: app_conn
  inner_sql: SELECT SingerId FROM Singers
  outer_sql: SELECT * FROM __external__
  result:
    cardinality: many
    struct: SingerIDRow
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v", err)
	}
	plan, err := BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	sql := plan.Queries[0].SQL
	if !strings.Contains(sql, `EXTERNAL_QUERY('example-project.us.example-connection'`) {
		t.Fatalf("planned SQL = %s, want actual connection ID", sql)
	}
	if got, want := plan.Queries[0].Kind, "external_query"; got != want {
		t.Fatalf("planned query kind = %q, want %q", got, want)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsConnectionIDAlias(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    external_query_connections:
    - name: app_conn
      connection_id: example-project.us.example-connection
      spanner_catalog: app
queries:
- name: ExternalSingerIDs
  catalog: analytics
  kind: external_query
  binding: app_conn
  inner_sql: SELECT 1 AS value
  result:
    struct: Row
`))
	if err == nil || !strings.Contains(err.Error(), `unsupported v1alpha field "connection_id"`) {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want connection_id alias rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsRawExternalQuery(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: analytics
  kind: bigquery
queries:
- name: RawExternalQuery
  catalog: analytics
  kind: sql
  sql: SELECT * FROM EXTERNAL_QUERY('conn', 'SELECT 1')
  result:
    struct: Row
`))
	if err == nil || !strings.Contains(err.Error(), "raw EXTERNAL_QUERY is not supported") {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want raw EXTERNAL_QUERY rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaSpannerExternalDataset(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
  ddl: spanner.sql
- name: analytics
  kind: bigquery
  project: example-project
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app
      spanner_database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
      location: US
      access:
        cloud_resource_connection_id: example-project.us.example-connection
        verification_evidence:
          status: verified
          verifier: terraform-plan
          checked_at: "2026-05-04T10:30:00Z"
      projection:
        unsupported_columns: omit
        named_schema_tables: warn_and_omit
queries:
- name: ExternalDatasetSingers
  catalog: analytics
  kind: sql
  sql: SELECT SingerId FROM ` + "`example-project.analytics_spanner.Singers`" + `
  result:
    struct: SingerRow
rules:
  suppressions:
  - scope: catalog-binding/analytics.app_dataset
    rule: external-dataset-access-unverified
    reason: checked outside generator
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v", err)
	}
	plan, err := BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if got, want := plan.CatalogBindings[0].DatabaseRole, "reader"; got != want {
		t.Fatalf("external dataset role = %q, want %q", got, want)
	}
	if got, want := plan.CatalogBindings[0].Access, "cloud_resource_connection"; got != want {
		t.Fatalf("external dataset access = %q, want %q", got, want)
	}
	verification := plan.CatalogBindings[0].AccessVerification
	if got, want := verification.Status, "verified"; got != want {
		t.Fatalf("external dataset verification status = %q, want %q", got, want)
	}
	if got, want := verification.Source, "external_evidence"; got != want {
		t.Fatalf("external dataset verification source = %q, want %q", got, want)
	}
	if verification.IndependentlyVerifiedByGenerator {
		t.Fatalf("external dataset verification IndependentlyVerifiedByGenerator = true, want false")
	}
	if got, want := len(plan.CatalogBindings[0].VetSuppressions), 1; got != want {
		t.Fatalf("vet suppressions = %d, want %d", got, want)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsExternalDatasetDatabaseRole(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app
      access:
        database_role: reader
`))
	if err == nil || !strings.Contains(err.Error(), `unsupported v1alpha field "database_role"`) {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want database_role rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsVerificationEvidenceSource(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app
      access:
        verification_evidence:
          status: verified
          source: external_evidence
          verifier: live-check
          checked_at: "2026-05-04T10:30:00Z"
`))
	if err == nil || !strings.Contains(err.Error(), `access.verification_evidence.source is normalized by the plan`) {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want verification source rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaGlobalEmitControlsWrites(t *testing.T) {
	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
emit:
  spanner:
    mutations: true
catalogs:
- name: app
  kind: spanner
writes:
- name: UpdateSingerName
  catalog: app
  table: Singers
  operation: update
  input: SingerWrite
  key:
  - SingerId
  update:
    columns:
    - FirstName
- name: UpdateSingerTitle
  catalog: app
  table: Singers
  operation: update
  input: SingerWrite
  key:
  - SingerId
  update:
    columns:
    - Title
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v", err)
	}
	for _, write := range config.Writes {
		if got, want := strings.Join(write.Methods, ","), "mutation"; got != want {
			t.Fatalf("write %s methods = %q, want %q", write.Name, got, want)
		}
		if !write.EmitExplicit {
			t.Fatalf("write %s EmitExplicit = false, want true", write.Name)
		}
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsWriteLevelEmit(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
writes:
- name: UpdateSingerName
  catalog: app
  table: Singers
  operation: update
  input: SingerWrite
  key:
  - SingerId
  update:
    columns:
    - FirstName
  emit:
    mutation: true
`))
	if err == nil || !strings.Contains(err.Error(), `unsupported v1alpha field "emit" at writes[0]; use "global emit.spanner"`) {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want write-level emit rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaPreservesQueryParams(t *testing.T) {
	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
queries:
- name: GetSinger
  catalog: app
  kind: sql
  sql: SELECT @SingerId AS SingerId
  params:
  - name: SingerId
    type: INT64
  result:
    struct: SingerRow
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v", err)
	}
	if got, want := len(config.Queries[0].Params), 1; got != want {
		t.Fatalf("len(config.Queries[0].Params) = %d, want %d", got, want)
	}
	if got, want := config.Queries[0].Params[0].Name, "SingerId"; got != want {
		t.Fatalf("param name = %q, want %q", got, want)
	}
	if got, want := config.Queries[0].Params[0].Type, "INT64"; got != want {
		t.Fatalf("param type = %q, want %q", got, want)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaPreservesQueryParamScope(t *testing.T) {
	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    external_query_connections:
    - name: app_conn
      id: example-project.us.example-connection
      spanner_catalog: app
queries:
- name: GetSinger
  catalog: analytics
  kind: external_query
  binding: app_conn
  inner_sql: SELECT @SingerId AS SingerId
  params:
  - name: SingerId
    type: INT64
    scope: inner
  result:
    struct: SingerRow
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v", err)
	}
	if got, want := config.Queries[0].Params[0].Scope, "inner"; got != want {
		t.Fatalf("param scope = %q, want %q", got, want)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaPreservesGeneratedQueryOrderBy(t *testing.T) {
	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
queries:
- name: ListSingers
  catalog: app
  kind: table
  table: Singers
  order_by: none
  result:
    struct: SingerRow
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v", err)
	}
	if got, want := config.Queries[0].OrderBy, "none"; got != want {
		t.Fatalf("query order_by = %q, want %q", got, want)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsOrderByOnExplicitSQL(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
queries:
- name: GetSinger
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  order_by: none
  result:
    struct: Row
`))
	if err == nil || !strings.Contains(err.Error(), "field order_by is not supported") {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want order_by rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsQueryKindFieldMismatch(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr string
	}{
		{
			name: "table missing table",
			query: `
- name: Bad
  catalog: app
  kind: table
  result:
    struct: Row
`,
			wantErr: "kind table: table is required",
		},
		{
			name: "sql with table",
			query: `
- name: Bad
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  table: Singers
  result:
    struct: Row
`,
			wantErr: "kind sql: field table is not supported",
		},
		{
			name: "external query missing inner sql",
			query: `
- name: Bad
  catalog: analytics
  kind: external_query
  binding: app_conn
  result:
    struct: Row
`,
			wantErr: "kind external_query: inner_sql is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    external_query_connections:
    - name: app_conn
      id: example-project.us.example-connection
      spanner_catalog: app
queries:
` + tt.query))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsStructuredInputAndKey(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
writes:
- name: UpdateSinger
  catalog: app
  table: Singers
  operation: update
  input:
    struct: SingerWrite
  key:
  - SingerId
  update:
    columns:
    - FirstName
`))
	if err == nil || !strings.Contains(err.Error(), "input must be a scalar struct name") {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want structured input rejection", err)
	}

	_, err = ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
writes:
- name: UpdateSinger
  catalog: app
  table: Singers
  operation: update
  input: SingerWrite
  key:
    columns:
    - SingerId
  update:
    columns:
    - FirstName
`))
	if err == nil || !strings.Contains(err.Error(), "column list must be a YAML sequence") {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want structured key rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsRulesSeverity(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
rules:
  severity:
    external-dataset-access-unverified: error
catalogs:
- name: app
  kind: spanner
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`))
	if err == nil || !strings.Contains(err.Error(), `unsupported v1alpha field "severity" at rules`) {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want rules.severity rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsValueAliases(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "dialect alias",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
  dialect: google-sql
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`,
			wantErr: `unsupported Spanner dialect "google-sql"`,
		},
		{
			name: "projection error alias",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app
      projection:
        unsupported_columns: error
queries:
- name: GetLiteral
  catalog: analytics
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`,
			wantErr: `unsupported unsupported_columns policy "error"`,
		},
		{
			name: "silent named schema omit",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app
      projection:
        named_schema_tables: omit
queries:
- name: GetLiteral
  catalog: analytics
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`,
			wantErr: `unsupported named_schema_tables policy "omit"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseQueryCodegenConfigYAML([]byte(tt.yaml))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaUpsertRequiresInsertUpdateColumnMatch(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
emit:
  spanner:
    mutations: true
catalogs:
- name: app
  kind: spanner
writes:
- name: UpsertSinger
  catalog: app
  table: Singers
  operation: upsert
  input: SingerWrite
  key:
  - SingerId
  insert:
    columns:
    - SingerId
    - FirstName
    - LastName
    - CreatedAt
  update:
    columns:
    - FirstName
    - LastName
`))
	if err == nil || !strings.Contains(err.Error(), "cannot represent insert-only non-key column CreatedAt") {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want insert-only column rejection", err)
	}

	_, err = ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
writes:
- name: UpsertSinger
  catalog: app
  table: Singers
  operation: upsert
  input: SingerWrite
  key:
  - SingerId
  insert:
    columns:
    - SingerId
    - FirstName
  update:
    columns:
    - FirstName
    - LastName
`))
	if err == nil || !strings.Contains(err.Error(), "missing LastName") {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want missing update column rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsWriteOperationFieldMismatch(t *testing.T) {
	tests := []struct {
		name    string
		write   string
		wantErr string
	}{
		{
			name: "insert without columns",
			write: `
- name: BadInsert
  catalog: app
  table: Singers
  operation: insert
  input: SingerWrite
`,
			wantErr: "operation insert: insert.columns is required",
		},
		{
			name: "delete with update",
			write: `
- name: BadDelete
  catalog: app
  table: Singers
  operation: delete
  input: SingerWrite
  update:
    columns:
    - FirstName
`,
			wantErr: "operation delete: update is not supported",
		},
		{
			name: "update with insert",
			write: `
- name: BadUpdate
  catalog: app
  table: Singers
  operation: update
  input: SingerWrite
  insert:
    columns:
    - SingerId
  update:
    columns:
    - FirstName
`,
			wantErr: "operation update: insert is not supported",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
writes:
` + tt.write))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsQueryMethods(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
emit:
  spanner:
    query_methods: true
catalogs:
- name: app
  kind: spanner
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`))
	if err == nil || !strings.Contains(err.Error(), "emit.spanner.query_methods is not supported in v1alpha yet") {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want query_methods rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsStableV1Spelling(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: "1"
go:
  package: db
catalogs:
- name: app
  kind: spanner
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`))
	if err == nil || !strings.Contains(err.Error(), `unsupported config version "1"; use version: v1alpha`) {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want stable v1 spelling rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsLegacyTopLevelKeys(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
package: db
schemas: []
`))
	if err == nil || !strings.Contains(err.Error(), `unsupported v1alpha field "package"; use "go.package"`) {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want legacy field rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsLegacyNestedKeys(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
queries:
- name: OldQuery
  source: app
  sql: SELECT 1
  result_struct: Row
`))
	if err == nil || !strings.Contains(err.Error(), `unsupported v1alpha field "result_struct" at queries[0]; use "result.struct"`) {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want nested legacy field rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsUnknownKeys(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "top level",
			yaml: `
version: v1alpha
go:
  package: db
unknown: true
`,
			wantErr: `unsupported v1alpha field "unknown"`,
		},
		{
			name: "query",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  unexpected: true
  result:
    struct: Row
`,
			wantErr: `unsupported v1alpha field "unexpected" at queries[0]`,
		},
		{
			name: "verification evidence source",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app
      access:
        verification_evidence:
          status: verified
          source: ""
          verifier: terraform-plan
          checked_at: "2026-05-04T10:30:00Z"
`,
			wantErr: `access.verification_evidence.source is normalized by the plan`,
		},
		{
			name: "verification evidence extra",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app
      access:
        verification_evidence:
          status: verified
          verifier: terraform-plan
          checked_at: "2026-05-04T10:30:00Z"
          evidence_digest: abc123
`,
			wantErr: `unsupported v1alpha field "evidence_digest" at catalogs[1].bindings.spanner_external_datasets[0].access.verification_evidence`,
		},
		{
			name: "write conflict",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
writes:
- name: UpsertSinger
  catalog: app
  table: Singers
  operation: upsert
  input: Singer
  key:
  - SingerId
  insert:
    columns:
    - SingerId
    - FirstName
  update:
    columns:
    - FirstName
  conflict:
    strategy: insert_or_update
`,
			wantErr: `unsupported v1alpha field "conflict" at writes[0]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseQueryCodegenConfigYAML([]byte(tt.yaml))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsInvalidQueryResultShape(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "exec cardinality",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
queries:
- name: RunDML
  catalog: app
  kind: sql
  sql: UPDATE Singers SET Active = TRUE WHERE TRUE
  result:
    cardinality: exec
    struct: Row
`,
			wantErr: `query RunDML result.cardinality "exec" is not supported; use one, maybe_one, or many`,
		},
		{
			name: "missing result struct",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    cardinality: one
`,
			wantErr: `query GetLiteral result.struct is required`,
		},
		{
			name: "invalid required policy",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
    required:
      policy: trust_me
`,
			wantErr: `query GetLiteral result.required.policy "trust_me" is not supported; use override or strict`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseQueryCodegenConfigYAML([]byte(tt.yaml))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsDuplicatePublicNames(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "catalog",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: app
  kind: spanner
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`,
			wantErr: `duplicate catalog name "app"`,
		},
		{
			name: "query",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 2 AS value
  result:
    struct: Row
`,
			wantErr: `duplicate query name "GetLiteral"`,
		},
		{
			name: "write",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
writes:
- name: UpdateSinger
  catalog: app
  table: Singers
  operation: update
  input: Singer
  update:
    columns:
    - FirstName
- name: UpdateSinger
  catalog: app
  table: Singers
  operation: update
  input: Singer
  update:
    columns:
    - LastName
`,
			wantErr: `duplicate write name "UpdateSinger"`,
		},
		{
			name: "external_query_connection",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    external_query_connections:
    - name: app_conn
      id: example-project.us.app-connection
      spanner_catalog: app
    - name: app_conn
      id: example-project.us.other-connection
      spanner_catalog: app
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`,
			wantErr: `duplicate catalog analytics external_query_connections name "app_conn"`,
		},
		{
			name: "spanner_external_dataset",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app
    - name: app_dataset
      dataset: analytics_spanner_copy
      spanner_catalog: app
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`,
			wantErr: `duplicate catalog analytics spanner_external_datasets name "app_dataset"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseQueryCodegenConfigYAML([]byte(tt.yaml))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsNonIdentifierPublicNames(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
queries:
- name: Get/Literal
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`))
	if err == nil || !strings.Contains(err.Error(), `query name "Get/Literal" must match ^[A-Za-z_][A-Za-z0-9_]*$`) {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want identifier rejection", err)
	}
}

func TestParseQueryCodegenConfigYAMLV1AlphaRejectsNonIdentifierReferences(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "query catalog",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
queries:
- name: GetLiteral
  catalog: app-prod
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`,
			wantErr: `query GetLiteral catalog reference "app-prod" must match ^[A-Za-z_][A-Za-z0-9_]*$`,
		},
		{
			name: "external query binding",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    external_query_connections:
    - name: app_conn
      id: example-project.us.app-connection
      spanner_catalog: app
queries:
- name: ExternalGetLiteral
  catalog: analytics
  kind: external_query
  binding: app-conn
  inner_sql: SELECT 1 AS value
  result:
    struct: Row
`,
			wantErr: `query ExternalGetLiteral binding reference "app-conn" must match ^[A-Za-z_][A-Za-z0-9_]*$`,
		},
		{
			name: "connection spanner catalog",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    external_query_connections:
    - name: app_conn
      id: example-project.us.app-connection
      spanner_catalog: app-prod
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`,
			wantErr: `catalog analytics external_query_connections app_conn spanner_catalog reference "app-prod" must match ^[A-Za-z_][A-Za-z0-9_]*$`,
		},
		{
			name: "write catalog",
			yaml: `
version: v1alpha
go:
  package: db
emit:
  spanner:
    mutations: true
catalogs:
- name: app
  kind: spanner
writes:
- name: UpdateSinger
  catalog: app-prod
  table: Singers
  operation: update
  input: Singer
  update:
    columns:
    - FirstName
`,
			wantErr: `write UpdateSinger catalog reference "app-prod" must match ^[A-Za-z_][A-Za-z0-9_]*$`,
		},
		{
			name: "external dataset spanner catalog",
			yaml: `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
- name: analytics
  kind: bigquery
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app-prod
queries:
- name: GetLiteral
  catalog: app
  kind: sql
  sql: SELECT 1 AS value
  result:
    struct: Row
`,
			wantErr: `catalog analytics spanner_external_datasets app_dataset spanner_catalog reference "app-prod" must match ^[A-Za-z_][A-Za-z0-9_]*$`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseQueryCodegenConfigYAML([]byte(tt.yaml))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ParseQueryCodegenConfigYAML() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateQueryCodeMergesCompatibleResultStructs(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{{
			Name:    "spanner",
			Dialect: "spanner",
			DDL:     "schema.sql",
		}},
		Queries: []QueryCodegenQuery{
			{
				Name:         "ListSingerIDs",
				Catalog:      "spanner",
				SQL:          "SELECT SingerId FROM Singers",
				ResultStruct: "SingerRow",
				Required:     []string{"SingerId"},
			},
			{
				Name:         "ListSingerNames",
				Catalog:      "spanner",
				SQL:          "SELECT SingerId, FirstName FROM Singers",
				ResultStruct: "SingerRow",
				Required:     []string{"SingerId"},
			},
		},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	for _, want := range []string{
		"package db",
		"ListSingerIDsSQL",
		`"SELECT SingerId FROM Singers"`,
		`ListSingerNamesSQL = "SELECT SingerId, FirstName FROM Singers"`,
		"type SingerRow struct",
		"SingerId  int64",
		"FirstName NullValue[string]",
		`bigquery:"SingerId" spanner:"SingerId"`,
		`bigquery:"FirstName" spanner:"FirstName"`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
}

func TestGenerateQueryCodeTableQuery(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  FullName STRING(MAX) AS (FirstName) STORED HIDDEN
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "ListSingers",
			Catalog:      "spanner",
			Table:        "Singers",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	if strings.Contains(code, "FullName") {
		t.Fatalf("generated table query included hidden column:\n%s", code)
	}
	for _, want := range []string{
		`ListSingersSQL = "SELECT ` + "`SingerId`" + `, ` + "`FirstName`" + ` FROM ` + "`Singers`" + ` ORDER BY ` + "`SingerId`" + `"`,
		"SingerId  NullValue[int64]",
		"FirstName NullValue[string]",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
}

func TestGenerateQueryCodeIndexQuery(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  AlbumTitle STRING(MAX),
  MarketingBudget INT64
) PRIMARY KEY (SingerId, AlbumId);

CREATE INDEX AlbumsByTitle ON Albums(AlbumTitle, SingerId) STORING (MarketingBudget);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "FindAlbumsByTitle",
			Catalog:      "spanner",
			Index:        "AlbumsByTitle",
			KeyPrefix:    []string{"AlbumTitle"},
			ResultStruct: "AlbumIndexRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	for _, want := range []string{
		"AlbumTitle      NullValue[string]",
		"SingerId        NullValue[int64]",
		"AlbumId         NullValue[int64]",
		"MarketingBudget NullValue[int64]",
		"SELECT `AlbumTitle`, `SingerId`, `AlbumId`, `MarketingBudget` FROM `Albums`@{FORCE_INDEX=`AlbumsByTitle`} WHERE `AlbumTitle` = @AlbumTitle ORDER BY `AlbumTitle`, `SingerId`, `AlbumId`",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
}

func TestGenerateQueryCodeNullFilteredIndexQueryAddsNullFilters(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX),
  Rating INT64 NOT NULL,
) PRIMARY KEY (SingerId);

CREATE NULL_FILTERED INDEX SingersByFirstLastRating ON Singers(FirstName, LastName, Rating);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "FindSingersByFirstName",
			Catalog:      "spanner",
			Index:        "SingersByFirstLastRating",
			KeyPrefix:    []string{"FirstName"},
			ResultStruct: "SingerIndexRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	for _, want := range []string{
		"SELECT `FirstName`, `LastName`, `Rating`, `SingerId` FROM `Singers`@{FORCE_INDEX=`SingersByFirstLastRating`} WHERE `FirstName` = @FirstName AND `FirstName` IS NOT NULL AND `LastName` IS NOT NULL ORDER BY `FirstName`, `LastName`, `Rating`, `SingerId`",
		`FindSingersByFirstNameSQL = "SELECT `,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
	if strings.Contains(code, "`Rating` IS NOT NULL") {
		t.Fatalf("generated code added redundant NOT NULL key filter:\n%s", code)
	}
}

func TestGenerateQueryCodeGeneratedQueryOrderByCanBeDisabled(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "ListSingers",
			Catalog:      "spanner",
			Table:        "Singers",
			OrderBy:      "none",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	if strings.Contains(code, "ORDER BY") {
		t.Fatalf("generated code contains ORDER BY despite order_by none:\n%s", code)
	}
}

func TestGenerateQueryCodeRejectsUnsupportedGeneratedQueryOrderBy(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	_, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "ListSingers",
			Catalog:      "spanner",
			Table:        "Singers",
			OrderBy:      "primary_key",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), `unsupported table query order_by "primary_key"`) {
		t.Fatalf("GenerateQueryCode() error = %v, want unsupported table order_by error", err)
	}
}

func TestGenerateQueryCodeIndexQueryParamOverride(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  AlbumTitle STRING(MAX)
) PRIMARY KEY (SingerId, AlbumId);

CREATE INDEX AlbumsByTitle ON Albums(AlbumTitle);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "FindAlbumsByTitle",
			Catalog:      "spanner",
			Index:        "AlbumsByTitle",
			KeyPrefix:    []string{"AlbumTitle"},
			ResultStruct: "AlbumIndexRow",
			Params:       []QueryCodegenParam{{Name: "AlbumTitle", Type: "STRING(MAX)"}},
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if got, want := len(plan.Queries[0].Params), 1; got != want {
		t.Fatalf("len(plan.Queries[0].Params) = %d, want %d: %+v", got, want, plan.Queries[0].Params)
	}
	if got, want := plan.Queries[0].Params[0].Type, "STRING(MAX)"; got != want {
		t.Fatalf("param type = %q, want %q", got, want)
	}
}

func TestGenerateQueryCodeMergesTableAndIndexResultsAsNullableUnion(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  AlbumTitle STRING(MAX),
  MarketingBudget INT64
) PRIMARY KEY (SingerId, AlbumId);

CREATE INDEX AlbumsByTitle ON Albums(AlbumTitle, SingerId) STORING (MarketingBudget);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{
			{
				Name:         "ListAlbums",
				Catalog:      "spanner",
				Table:        "Albums",
				ResultStruct: "AlbumRow",
				Required:     []string{"AlbumId"},
			},
			{
				Name:         "FindAlbumsByTitle",
				Catalog:      "spanner",
				Index:        "AlbumsByTitle",
				KeyPrefix:    []string{"AlbumTitle"},
				ResultStruct: "AlbumRow",
			},
		},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	for _, want := range []string{
		"type AlbumRow struct",
		"AlbumId         NullValue[int64]",
		"SingerId        NullValue[int64]",
		"AlbumTitle      NullValue[string]",
		"MarketingBudget NullValue[int64]",
		"ListAlbumsSQL",
		"FindAlbumsByTitleSQL",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
	if strings.Contains(code, "AlbumId int64") {
		t.Fatalf("field with unproven nullability must stay nullable in shared struct:\n%s", code)
	}
}

func TestGenerateQueryCodeSpannerWrites(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Writes: []QueryCodegenWrite{
			{
				Name:        "SaveSinger",
				Catalog:     "spanner",
				Table:       "Singers",
				Operation:   "insert_or_update",
				InputStruct: "SingerWrite",
				Update:      QueryCodegenWriteUpdate{Columns: []string{"FirstName"}},
			},
			{
				Name:        "UpdateSingerName",
				Catalog:     "spanner",
				Table:       "Singers",
				Operation:   "update",
				InputStruct: "SingerNameUpdate",
				Update:      QueryCodegenWriteUpdate{Columns: []string{"FirstName", "LastName"}},
				Methods:     []string{"mutation", "dml"},
			},
			{
				Name:        "DeleteSinger",
				Catalog:     "spanner",
				Table:       "Singers",
				Operation:   "delete",
				InputStruct: "SingerDelete",
			},
		},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	for _, want := range []string{
		`"cloud.google.com/go/spanner"`,
		"type SingerWrite struct",
		"SingerId  int64",
		"FirstName spanner.NullString",
		"func (w *SingerWrite) SaveSingerMutation() (*spanner.Mutation, error)",
		`return spanner.InsertOrUpdate("Singers", []string{"SingerId", "FirstName"}, []interface{}{w.SingerId, w.FirstName}), nil`,
		`const SaveSingerDML = "INSERT OR UPDATE INTO ` + "`Singers`" + ` (` + "`SingerId`" + `, ` + "`FirstName`" + `) VALUES (@SingerId, @FirstName)"`,
		"func (w *SingerWrite) SaveSingerDMLStatement() spanner.Statement",
		"type SingerNameUpdate struct",
		"func (w *SingerNameUpdate) UpdateSingerNameMutation() (*spanner.Mutation, error)",
		`const UpdateSingerNameDML = "UPDATE ` + "`Singers`" + ` SET ` + "`FirstName`" + ` = @FirstName, ` + "`LastName`" + ` = @LastName WHERE ` + "`SingerId`" + ` = @SingerId"`,
		"func (w *SingerNameUpdate) UpdateSingerNameDMLStatement() spanner.Statement",
		`"SingerId": w.SingerId`,
		"type SingerDelete struct",
		"func (w *SingerDelete) DeleteSingerMutation() *spanner.Mutation",
		`return spanner.Delete("Singers", spanner.Key{w.SingerId})`,
		`const DeleteSingerDML = "DELETE FROM ` + "`Singers`" + ` WHERE ` + "`SingerId`" + ` = @SingerId"`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
}

func TestGenerateQueryCodeWriteUpdateMaskIsRequired(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	_, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Writes: []QueryCodegenWrite{{
			Name:      "UpdateSinger",
			Catalog:   "spanner",
			Table:     "Singers",
			Operation: "update",
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "update.columns is required for operation update") {
		t.Fatalf("GenerateQueryCode() error = %v, want required update.columns error", err)
	}

	_, err = GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Writes: []QueryCodegenWrite{{
			Name:      "UpdateSinger",
			Catalog:   "spanner",
			Table:     "Singers",
			Operation: "update",
			Insert:    QueryCodegenWriteInsert{Columns: []string{"FirstName"}},
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "columns is only valid for insert and replace") {
		t.Fatalf("GenerateQueryCode() error = %v, want columns alias rejection", err)
	}
}

func TestGenerateQueryCodeWriteUpdateMaskExplicitAutoAll(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Writes: []QueryCodegenWrite{{
			Name:        "UpdateSinger",
			Catalog:     "spanner",
			Table:       "Singers",
			Operation:   "update",
			InputStruct: "SingerUpdate",
			Update:      QueryCodegenWriteUpdate{Columns: []string{autoAllNonKeyColumns}},
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	for _, want := range []string{
		`return spanner.Update("Singers", []string{"SingerId", "FirstName", "LastName"}, []interface{}{w.SingerId, w.FirstName, w.LastName}), nil`,
		`const UpdateSingerDML = "UPDATE ` + "`Singers`" + ` SET ` + "`FirstName`" + ` = @FirstName, ` + "`LastName`" + ` = @LastName WHERE ` + "`SingerId`" + ` = @SingerId"`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
}

func TestGenerateQueryCodeInsertOrUpdateRequiresInsertBranchColumns(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX) NOT NULL
) PRIMARY KEY (SingerId);
`)
	_, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Writes: []QueryCodegenWrite{{
			Name:        "SaveSinger",
			Catalog:     "spanner",
			Table:       "Singers",
			Operation:   "insert_or_update",
			InputStruct: "SingerWrite",
			Update:      QueryCodegenWriteUpdate{Columns: []string{"FirstName"}},
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "insert_or_update requires insert value for NOT NULL column LastName") {
		t.Fatalf("GenerateQueryCode() error = %v, want missing insert branch column error", err)
	}
}

func TestGenerateQueryCodeRejectsGeneratedUpdateColumn(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  FullName STRING(MAX) AS (FirstName) STORED
) PRIMARY KEY (SingerId);
`)
	_, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Writes: []QueryCodegenWrite{{
			Name:        "UpdateSinger",
			Catalog:     "spanner",
			Table:       "Singers",
			Operation:   "update",
			InputStruct: "SingerWrite",
			Update:      QueryCodegenWriteUpdate{Columns: []string{"FullName"}},
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "column FullName is not updatable") {
		t.Fatalf("GenerateQueryCode() error = %v, want generated column update rejection", err)
	}
}

func TestBuildQueryCodegenPlanColumnCapabilities(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  UpdatedAt TIMESTAMP OPTIONS (allow_commit_timestamp = TRUE),
  LastTouched TIMESTAMP DEFAULT (PENDING_COMMIT_TIMESTAMP()) ON UPDATE (PENDING_COMMIT_TIMESTAMP()),
  FullName STRING(MAX) AS (FirstName) STORED
) PRIMARY KEY (SingerId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Writes: []QueryCodegenWrite{{
			Name:        "UpdateSinger",
			Catalog:     "spanner",
			Table:       "Singers",
			Operation:   "update",
			InputStruct: "SingerWrite",
			Update:      QueryCodegenWriteUpdate{Columns: []string{"FirstName"}},
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	write := plan.Writes[0]
	caps := map[string]QueryCodegenPlanColumnCapability{}
	for _, cap := range write.ColumnCapabilities {
		caps[cap.Name] = cap
	}
	if !caps["UpdatedAt"].AllowCommitTimestamp {
		t.Fatalf("UpdatedAt capability = %+v, want allow_commit_timestamp", caps["UpdatedAt"])
	}
	if caps["FullName"].InsertValue || caps["FullName"].UpdateValue || caps["FullName"].GeneratedSQL == "" {
		t.Fatalf("FullName capability = %+v, want generated non-writable column", caps["FullName"])
	}
	if caps["LastTouched"].OnUpdateSQL == "" {
		t.Fatalf("LastTouched capability = %+v, want on_update_sql", caps["LastTouched"])
	}
	if got, want := len(write.ServerSideUpdateEffects), 1; got != want {
		t.Fatalf("len(write.ServerSideUpdateEffects) = %d, want %d: %+v", got, want, write.ServerSideUpdateEffects)
	}
	if write.ServerSideUpdateEffects[0].Column != "LastTouched" {
		t.Fatalf("server side update effect = %+v, want LastTouched", write.ServerSideUpdateEffects[0])
	}
}

func TestGenerateQueryCodeReplaceWriteIsMutationOnly(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Writes: []QueryCodegenWrite{{
			Name:        "ReplaceSinger",
			Catalog:     "spanner",
			Table:       "Singers",
			Operation:   "replace",
			InputStruct: "SingerWrite",
			Insert:      QueryCodegenWriteInsert{Columns: []string{"SingerId", "FirstName"}},
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	for _, want := range []string{
		"func (w *SingerWrite) ReplaceSingerMutation() (*spanner.Mutation, error)",
		`return spanner.Replace("Singers", []string{"SingerId", "FirstName"}, []interface{}{w.SingerId, w.FirstName}), nil`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
	for _, dontWant := range []string{"ReplaceSingerDML", "INSERT OR REPLACE"} {
		if strings.Contains(code, dontWant) {
			t.Fatalf("generated code included unsupported DML replace %q:\n%s", dontWant, code)
		}
	}

	_, err = GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Writes: []QueryCodegenWrite{{
			Name:        "ReplaceSinger",
			Catalog:     "spanner",
			Table:       "Singers",
			Operation:   "replace",
			InputStruct: "SingerWrite",
			Insert:      QueryCodegenWriteInsert{Columns: []string{"SingerId", "FirstName"}},
			Methods:     []string{"dml"},
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "operation replace does not support dml helpers") {
		t.Fatalf("GenerateQueryCode() error = %v, want DML replace rejection", err)
	}
}

func TestGenerateQueryCodeWriteCanReuseResultStruct(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "ListSingers",
			Catalog:      "spanner",
			Table:        "Singers",
			ResultStruct: "SingerRow",
		}},
		Writes: []QueryCodegenWrite{{
			Name:        "UpdateSingerName",
			Catalog:     "spanner",
			Table:       "Singers",
			Operation:   "update",
			InputStruct: "SingerRow",
			Update:      QueryCodegenWriteUpdate{Columns: []string{"FirstName"}},
			Methods:     []string{"mutation", "dml"},
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	if got := strings.Count(code, "type SingerRow struct"); got != 1 {
		t.Fatalf("generated %d SingerRow structs, want 1:\n%s", got, code)
	}
	for _, want := range []string{
		"type SingerRow struct",
		"SingerId  NullValue[int64]",
		"FirstName NullValue[string]",
		"LastName  NullValue[string]",
		"func (w *SingerRow) UpdateSingerNameMutation() (*spanner.Mutation, error)",
		`return spanner.Update("Singers", []string{"SingerId", "FirstName"}, []interface{}{w.SingerId, w.FirstName}), nil`,
		"func (w *SingerRow) UpdateSingerNameDMLStatement() spanner.Statement",
		"func (n NullValue[T]) EncodeSpanner() (interface{}, error)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
}

func TestGenerateQueryCodeWriteReuseResultStructRejectsMissingWriteColumn(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  Status STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	_, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "ListSingerNames",
			Catalog:      "spanner",
			SQL:          "SELECT SingerId, FirstName FROM Singers",
			ResultStruct: "SingerRow",
		}},
		Writes: []QueryCodegenWrite{{
			Name:        "UpdateSingerStatus",
			Catalog:     "spanner",
			Table:       "Singers",
			Operation:   "update",
			InputStruct: "SingerRow",
			Update:      QueryCodegenWriteUpdate{Columns: []string{"Status"}},
			Methods:     []string{"mutation"},
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "input_struct SingerRow is shared with query results but has no field for column Status") {
		t.Fatalf("GenerateQueryCode() error = %v, want shared write field error", err)
	}
}

func TestGenerateQueryCodeNormalizesSharedStructNames(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "ListSingers",
			Catalog:      "spanner",
			Table:        "Singers",
			ResultStruct: "singer-row",
		}},
		Writes: []QueryCodegenWrite{{
			Name:        "UpdateSingerName",
			Catalog:     "spanner",
			Table:       "Singers",
			Operation:   "update",
			InputStruct: "singer-row",
			Update:      QueryCodegenWriteUpdate{Columns: []string{"FirstName"}},
			Methods:     []string{"mutation"},
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	if got := strings.Count(code, "type SingerRow struct"); got != 1 {
		t.Fatalf("generated %d SingerRow structs, want 1:\n%s", got, code)
	}
	if !strings.Contains(code, "func (w *SingerRow) UpdateSingerNameMutation() (*spanner.Mutation, error)") {
		t.Fatalf("generated code does not reuse normalized result struct:\n%s", code)
	}
}

func TestGenerateQueryCodeWriteUpdateMaskRejectsPrimaryKey(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	_, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Writes: []QueryCodegenWrite{{
			Name:      "UpdateSinger",
			Catalog:   "spanner",
			Table:     "Singers",
			Operation: "update",
			Update:    QueryCodegenWriteUpdate{Columns: []string{"SingerId"}},
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "primary key column SingerId cannot be in update_columns") {
		t.Fatalf("GenerateQueryCode() error = %v, want primary key update_mask error", err)
	}
}

func TestGenerateQueryCodeBigQueryExternalQuerySource(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalQueryConnections: []QueryCodegenExternalSchema{{
					Connection:    "example-project.us.example-connection",
					SpannerSource: "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:         "ExternalSingerIDs",
			Catalog:      "bigquery",
			SQL:          "SELECT * FROM EXTERNAL_QUERY('example-project.us.example-connection', '''SELECT SingerId FROM Singers''')",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	if !strings.Contains(code, "SingerId NullValue[int64]") {
		t.Fatalf("generated code missing external query field:\n%s", code)
	}
}

func TestGenerateQueryCodeBigQueryFederatedQuerySource(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalSchemas: []QueryCodegenExternalSchema{{
					Connection: "example-project.us.example-connection",
					Schema:     "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalSingerIDs",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection:    "example-project.us.example-connection",
				SpannerSource: "spanner",
				InnerSQL:      "SELECT SingerId FROM Singers",
			},
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	for _, want := range []string{
		"ExternalSingerIDsSpannerSQL",
		`"SELECT SingerId FROM Singers"`,
		"ExternalSingerIDsBigQuerySQL",
		`"SELECT * FROM EXTERNAL_QUERY('example-project.us.example-connection', 'SELECT SingerId FROM Singers')"`,
		"SingerId NullValue[int64]",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
}

func TestBuildQueryCodegenPlanBigQueryFederatedQueryUsesExternalQueryKind(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalSchemas: []QueryCodegenExternalSchema{{
					Connection: "example-project.us.example-connection",
					Schema:     "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalSingerIDs",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection: "example-project.us.example-connection",
				InnerSQL:   "SELECT SingerId FROM Singers",
			},
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if got, want := plan.Queries[0].Kind, "external_query"; got != want {
		t.Fatalf("plan query kind = %q, want %q", got, want)
	}
}

func TestGenerateQueryCodeBigQuerySpannerExternalDataset(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  SearchName STRING(MAX) AS (FirstName) STORED HIDDEN
) PRIMARY KEY (SingerId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				Project: "example-project",
				SpannerExternalDatasets: []QueryCodegenSpannerExternalDataset{{
					Dataset:        "analytics_spanner",
					SpannerSource:  "spanner",
					ExternalSource: "google-cloudspanner://reader@/projects/example-project/instances/app/databases/app",
					Location:       "us",
					Access: QueryCodegenSpannerExternalDatasetAccess{
						Mode:                    "cloud_resource",
						CloudResourceConnection: "example-project.us.example-connection",
					},
					DatabaseRole:       "reader",
					UnsupportedColumns: "omit",
					NamedSchemaPolicy:  "warn_and_omit",
					Vet: QueryCodegenVetConfig{Disable: []QueryCodegenVetDisable{{
						Rule:    "external-dataset-access-unverified",
						Reason:  "Static CI does not have access to live BigQuery external dataset metadata.",
						Owner:   "analytics-platform",
						Expires: "2026-12-31",
					}}},
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:         "ExternalDatasetSingers",
			Catalog:      "bigquery",
			SQL:          "SELECT * FROM analytics_spanner.Singers",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if got, want := len(plan.CatalogBindings), 1; got != want {
		t.Fatalf("len(plan.CatalogBindings) = %d, want %d: %+v", got, want, plan.CatalogBindings)
	}
	binding := plan.CatalogBindings[0]
	if binding.Kind != "spanner_external_dataset" || binding.BigQueryDataset != "example-project.analytics_spanner" || binding.SpannerSource != "spanner" {
		t.Fatalf("unexpected catalog binding: %+v", binding)
	}
	if got, want := binding.BigQueryDatasetRef.Project, "example-project"; got != want {
		t.Fatalf("binding.BigQueryDatasetRef.Project = %q, want %q", got, want)
	}
	if binding.Access != "cloud_resource_connection" || binding.DatabaseRole != "reader" || binding.DatabaseRoleSource != "config_and_external_source" {
		t.Fatalf("binding access metadata = %q/%q/%q, want cloud_resource_connection/reader/config_and_external_source", binding.Access, binding.DatabaseRole, binding.DatabaseRoleSource)
	}
	if binding.ExternalSource == "" || binding.Location != "US" || binding.CloudResourceConnection != "example-project.us.example-connection" {
		t.Fatalf("binding external metadata = source %q location %q cloud_resource_connection %q", binding.ExternalSource, binding.Location, binding.CloudResourceConnection)
	}
	if binding.LocationMetadata == nil || binding.CloudResourceConnectionMetadata == nil ||
		binding.LocationMetadata.Canonical != "US" ||
		binding.CloudResourceConnectionMetadata.ParsedLocationCanonical != "US" ||
		!binding.CloudResourceConnectionMetadata.LocationMatch {
		t.Fatalf("binding location metadata = %+v connection metadata = %+v, want canonical match", binding.LocationMetadata, binding.CloudResourceConnectionMetadata)
	}
	if got, want := binding.AccessVerification.Status, "not_checked"; got != want {
		t.Fatalf("binding.AccessVerification.Status = %q, want %q", got, want)
	}
	if got, want := binding.AccessVerification.Source, ""; got != want {
		t.Fatalf("binding.AccessVerification.Source = %q, want %q", got, want)
	}
	if binding.AccessVerification.IndependentlyVerifiedByGenerator {
		t.Fatalf("binding.AccessVerification = %+v, want no independent generator verification", binding.AccessVerification)
	}
	if binding.ProjectionMatrix.Source == "" ||
		binding.ProjectionMatrix.SourceURL == "" ||
		binding.ProjectionMatrix.DocsLastUpdated == "" ||
		binding.ProjectionMatrix.DocsLastChecked == "" ||
		binding.ProjectionMatrix.GeneratorMatrixVersion == 0 ||
		len(binding.ProjectionMatrix.Rows) == 0 {
		t.Fatalf("binding.ProjectionMatrix = %+v, want source/version/docs date/rows", binding.ProjectionMatrix)
	}
	if got, want := len(binding.VetSuppressions), 1; got != want {
		t.Fatalf("len(binding.VetSuppressions) = %d, want %d", got, want)
	}
	if binding.VetSuppressions[0].Owner != "analytics-platform" || binding.VetSuppressions[0].Expires != "2026-12-31" {
		t.Fatalf("binding vet suppression = %+v, want owner/expires", binding.VetSuppressions[0])
	}
	if binding.ProjectionPolicy.UnsupportedColumns != "omit" || binding.ProjectionPolicy.NamedSchemas != "warn_and_omit" {
		t.Fatalf("binding.ProjectionPolicy = %+v, want omit/warn_and_omit", binding.ProjectionPolicy)
	}
	if got, want := binding.ProjectedTables[0].Name, "example-project.analytics_spanner.Singers"; got != want {
		t.Fatalf("external dataset table = %q, want %q", got, want)
	}
	if got, want := binding.ProjectedTables[0].BigQueryTable, "example-project.analytics_spanner.Singers"; got != want {
		t.Fatalf("external dataset BigQuery table = %q, want %q", got, want)
	}
	if got, want := binding.ProjectedTables[0].SpannerTable, "Singers"; got != want {
		t.Fatalf("external dataset Spanner table = %q, want %q", got, want)
	}
	if !binding.ProjectedTables[0].VisibleInBigQueryMetadata || binding.ProjectedTables[0].BigQueryKeyMetadataVisible {
		t.Fatalf("external dataset table metadata flags = %+v, want BigQuery-visible table without BigQuery key metadata", binding.ProjectedTables[0])
	}
	if !binding.ProjectedTables[0].Columns[0].UnderlyingSpannerPrimaryKey || binding.ProjectedTables[0].Columns[0].BigQueryKeyMetadataVisible {
		t.Fatalf("external dataset PK column metadata = %+v, want underlying Spanner PK without BigQuery key metadata", binding.ProjectedTables[0].Columns[0])
	}
	if binding.ProjectedTables[0].Columns[2].Visible || binding.ProjectedTables[0].Columns[2].Reason == "" {
		t.Fatalf("hidden column projection = %+v, want invisible reason", binding.ProjectedTables[0].Columns[2])
	}
	if len(binding.Warnings) < 2 {
		t.Fatalf("binding.Warnings = %+v, want Data Boost and omitted-column warnings", binding.Warnings)
	}
	if !queryCodegenWarningsContain(binding.Warnings, "external-dataset-role-visibility-not-verified") {
		t.Fatalf("binding.Warnings = %+v, want database role visibility warning", binding.Warnings)
	}
	if got, want := len(plan.Queries[0].Fields), 2; got != want {
		t.Fatalf("len(plan.Queries[0].Fields) = %d, want %d", got, want)
	}
	if plan.Queries[0].StarExpansion == nil || !plan.Queries[0].StarExpansion.ProjectionLoss {
		t.Fatalf("plan.Queries[0].StarExpansion = %+v, want projection loss", plan.Queries[0].StarExpansion)
	}
	if got, want := strings.Join(plan.Queries[0].StarExpansion.OmittedColumns, ","), "Singers.SearchName"; got != want {
		t.Fatalf("omitted star expansion columns = %q, want %q", got, want)
	}
	if got, want := len(plan.Queries[0].Relations), 1; got != want {
		t.Fatalf("len(plan.Queries[0].Relations) = %d, want %d: %+v", got, want, plan.Queries[0].Relations)
	}
	relation := plan.Queries[0].Relations[0]
	if relation.SQLPath != "example-project.analytics_spanner.Singers" ||
		relation.Catalog != "spanner_external_dataset_projection" ||
		relation.Role != "select_source" ||
		!relation.Allowed ||
		!relation.ProjectionLoss {
		t.Fatalf("relation provenance = %+v, want lossy external dataset projection", relation)
	}
	if got, want := strings.Join(relation.OmittedColumns, ","), "Singers.SearchName"; got != want {
		t.Fatalf("relation omitted columns = %q, want %q", got, want)
	}
}

func queryCodegenWarningsContain(warnings []QueryCodegenPlanWarning, rule string) bool {
	for _, warning := range warnings {
		if warning.Rule == rule {
			return true
		}
	}
	return false
}

func TestBuildQueryCodegenPlanExternalDatasetExplicitColumnRelationProvenance(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  SearchName STRING(MAX) AS (FirstName) STORED HIDDEN
) PRIMARY KEY (SingerId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				SpannerExternalDatasets: []QueryCodegenSpannerExternalDataset{{
					Dataset:        "analytics_spanner",
					SpannerSource:  "spanner",
					ExternalSource: "opaque-spanner-source",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:         "ExternalDatasetSingerNames",
			Catalog:      "bigquery",
			SQL:          "SELECT SingerId, FirstName FROM analytics_spanner.Singers",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	query := plan.Queries[0]
	if query.StarExpansion != nil {
		t.Fatalf("query.StarExpansion = %+v, want nil for explicit columns", query.StarExpansion)
	}
	if got, want := len(query.Relations), 1; got != want {
		t.Fatalf("len(query.Relations) = %d, want %d: %+v", got, want, query.Relations)
	}
	relation := query.Relations[0]
	if relation.SQLPath != "analytics_spanner.Singers" || relation.Role != "select_source" || !relation.Allowed || !relation.ProjectionLoss {
		t.Fatalf("relation = %+v, want lossy external dataset provenance", relation)
	}
	if got, want := strings.Join(relation.OmittedColumns, ","), "Singers.SearchName"; got != want {
		t.Fatalf("relation omitted columns = %q, want %q", got, want)
	}
	if !queryCodegenWarningsContain(plan.CatalogBindings[0].Warnings, "external-dataset-external-source-unrecognized") {
		t.Fatalf("binding warnings = %+v, want unrecognized external_source warning", plan.CatalogBindings[0].Warnings)
	}
}

func TestBuildQueryCodegenPlanExternalDatasetNoRoleExternalSource(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				SpannerExternalDatasets: []QueryCodegenSpannerExternalDataset{{
					Dataset:        "analytics_spanner",
					SpannerSource:  "spanner",
					ExternalSource: "google-cloudspanner:/projects/example-project/instances/app/databases/app",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:         "ExternalDatasetSingerNames",
			Catalog:      "bigquery",
			SQL:          "SELECT SingerId FROM analytics_spanner.Singers",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if plan.CatalogBindings[0].DatabaseRole != "" || plan.CatalogBindings[0].DatabaseRoleSource != "" {
		t.Fatalf("database role metadata = %q/%q, want no inferred role", plan.CatalogBindings[0].DatabaseRole, plan.CatalogBindings[0].DatabaseRoleSource)
	}
	if queryCodegenWarningsContain(plan.CatalogBindings[0].Warnings, "external-dataset-external-source-unrecognized") {
		t.Fatalf("binding warnings = %+v, want no unrecognized warning for official no-role external_source", plan.CatalogBindings[0].Warnings)
	}
}

func TestBuildQueryCodegenPlanExternalDatasetWarnsOnUnofficialExternalSourceSlashForm(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				SpannerExternalDatasets: []QueryCodegenSpannerExternalDataset{{
					Dataset:        "analytics_spanner",
					SpannerSource:  "spanner",
					ExternalSource: "google-cloudspanner:///projects/example-project/instances/app/databases/app",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:         "ExternalDatasetSingerNames",
			Catalog:      "bigquery",
			SQL:          "SELECT SingerId FROM analytics_spanner.Singers",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if !queryCodegenWarningsContain(plan.CatalogBindings[0].Warnings, "external-dataset-external-source-unrecognized") {
		t.Fatalf("binding warnings = %+v, want unrecognized warning for unofficial external_source slash form", plan.CatalogBindings[0].Warnings)
	}
}

func TestBuildQueryCodegenPlanExternalDatasetRelationProvenanceIgnoresLiteralsAndComments(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				SpannerExternalDatasets: []QueryCodegenSpannerExternalDataset{{
					Dataset:       "analytics_spanner",
					SpannerSource: "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalDatasetNameLiteral",
			Catalog: "bigquery",
			SQL: `SELECT 'analytics_spanner.Singers' AS literal_value
-- analytics_spanner.Singers in a comment is not a relation
`,
			ResultStruct: "LiteralRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if len(plan.Queries[0].Relations) != 0 {
		t.Fatalf("query relations = %+v, want none for table names in literals/comments", plan.Queries[0].Relations)
	}
}

func TestBuildQueryCodegenPlanRejectsExternalDatasetDMLTarget(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	_, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				SpannerExternalDatasets: []QueryCodegenSpannerExternalDataset{{
					Dataset:       "analytics_spanner",
					SpannerSource: "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:         "InsertExternalDatasetSinger",
			Catalog:      "bigquery",
			SQL:          "INSERT INTO analytics_spanner.Singers (SingerId) VALUES (1)",
			Result:       "many",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "external dataset table analytics_spanner.Singers is read-only and cannot be used as dml_target") {
		t.Fatalf("BuildQueryCodegenPlan() error = %v, want read-only external dataset target error", err)
	}
}

func TestBuildQueryCodegenPlanRejectsExternalDatasetMetadataTarget(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	_, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				SpannerExternalDatasets: []QueryCodegenSpannerExternalDataset{{
					Dataset:       "analytics_spanner",
					SpannerSource: "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:         "CreateExternalDatasetTable",
			Catalog:      "bigquery",
			SQL:          "CREATE TABLE analytics_spanner.NewTable (SingerId INT64)",
			Result:       "many",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "external dataset path analytics_spanner.newtable is read-only and cannot be used as metadata_target") {
		t.Fatalf("BuildQueryCodegenPlan() error = %v, want read-only external dataset metadata target error", err)
	}
}

func TestBuildQueryCodegenPlanRejectsInvalidSpannerExternalDatasetAccess(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	_, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				SpannerExternalDatasets: []QueryCodegenSpannerExternalDataset{{
					Dataset:       "analytics_spanner",
					SpannerSource: "spanner",
					Access:        QueryCodegenSpannerExternalDatasetAccess{Mode: "direct"},
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:         "ExternalDatasetSingers",
			Catalog:      "bigquery",
			SQL:          "SELECT SingerId FROM analytics_spanner.Singers",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), `unsupported access "direct"`) {
		t.Fatalf("BuildQueryCodegenPlan() error = %v, want access validation error", err)
	}
}

func TestBuildQueryCodegenPlanRejectsInvalidSpannerExternalDatasetPolicies(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	baseConfig := QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:         "ExternalDatasetSingers",
			Catalog:      "bigquery",
			SQL:          "SELECT SingerId FROM analytics_spanner.Singers",
			ResultStruct: "SingerRow",
		}},
	}
	for _, tc := range []struct {
		name    string
		dataset QueryCodegenSpannerExternalDataset
		want    string
	}{
		{
			name: "unsupported columns policy",
			dataset: QueryCodegenSpannerExternalDataset{
				Dataset:            "analytics_spanner",
				SpannerSource:      "spanner",
				UnsupportedColumns: "warn",
			},
			want: `unsupported unsupported_columns policy "warn"`,
		},
		{
			name: "named schema policy",
			dataset: QueryCodegenSpannerExternalDataset{
				Dataset:           "analytics_spanner",
				SpannerSource:     "spanner",
				NamedSchemaPolicy: "omit",
			},
			want: `unsupported named_schema_policy "omit"`,
		},
		{
			name: "access verification hint",
			dataset: QueryCodegenSpannerExternalDataset{
				Dataset:       "analytics_spanner",
				SpannerSource: "spanner",
				Access:        QueryCodegenSpannerExternalDatasetAccess{VerificationHint: "verified"},
			},
			want: `access.verification_hint must be not_checked`,
		},
		{
			name: "access verification status",
			dataset: QueryCodegenSpannerExternalDataset{
				Dataset:            "analytics_spanner",
				SpannerSource:      "spanner",
				AccessVerification: QueryCodegenSpannerExternalDatasetVerification{Status: "unknown"},
			},
			want: `unsupported access verification status "unknown"`,
		},
		{
			name: "verified access without evidence",
			dataset: QueryCodegenSpannerExternalDataset{
				Dataset:            "analytics_spanner",
				SpannerSource:      "spanner",
				AccessVerification: QueryCodegenSpannerExternalDatasetVerification{Status: "verified"},
			},
			want: `access_verification status "verified" requires verifier and checked_at`,
		},
		{
			name: "invalid access verification source",
			dataset: QueryCodegenSpannerExternalDataset{
				Dataset:       "analytics_spanner",
				SpannerSource: "spanner",
				Access: QueryCodegenSpannerExternalDatasetAccess{
					VerificationEvidence: QueryCodegenSpannerExternalDatasetVerification{
						Status: "verified",
						Source: "terraform_plan",
					},
				},
			},
			want: `unsupported access verification source "terraform_plan"`,
		},
		{
			name: "user config cannot be verified evidence",
			dataset: QueryCodegenSpannerExternalDataset{
				Dataset:       "analytics_spanner",
				SpannerSource: "spanner",
				Access: QueryCodegenSpannerExternalDatasetAccess{
					VerificationEvidence: QueryCodegenSpannerExternalDatasetVerification{
						Status:    "verified",
						Source:    "user_config",
						Verifier:  "terraform-plan",
						CheckedAt: "2026-05-04T10:30:00Z",
					},
				},
			},
			want: `access verification source user_config is only valid with not_checked status`,
		},
		{
			name: "external source database role mismatch",
			dataset: QueryCodegenSpannerExternalDataset{
				Dataset:        "analytics_spanner",
				SpannerSource:  "spanner",
				ExternalSource: "google-cloudspanner://reader@/projects/example/instances/app/databases/app",
				Access:         QueryCodegenSpannerExternalDatasetAccess{DatabaseRole: "writer"},
			},
			want: `external_source database role "reader" conflicts with access.database_role "writer"`,
		},
		{
			name: "connection location mismatch",
			dataset: QueryCodegenSpannerExternalDataset{
				Dataset:       "analytics_spanner",
				SpannerSource: "spanner",
				Location:      "us",
				Access: QueryCodegenSpannerExternalDatasetAccess{
					CloudResourceConnection: "example-project.eu.example-connection",
				},
			},
			want: `external dataset connection location "eu" conflicts with location "us"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := baseConfig
			config.Schemas[1].SpannerExternalDatasets = []QueryCodegenSpannerExternalDataset{tc.dataset}
			_, err := BuildQueryCodegenPlan(config, dir)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("BuildQueryCodegenPlan() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestBuildQueryCodegenPlanRejectsPostgreSQLSpannerExternalDatasetSource(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	_, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{
			{Name: "spanner_pg", Dialect: "spanner_postgresql", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				SpannerExternalDatasets: []QueryCodegenSpannerExternalDataset{{
					Dataset:        "analytics_spanner",
					SpannerSource:  "spanner_pg",
					ExternalSource: "google-cloudspanner://reader@/projects/example-project/instances/app/databases/app",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:         "ExternalDatasetSingerNames",
			Catalog:      "bigquery",
			SQL:          "SELECT SingerId FROM analytics_spanner.Singers",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), `uses PostgreSQL dialect, which is not supported by this generator`) {
		t.Fatalf("BuildQueryCodegenPlan() error = %v, want PostgreSQL external dataset source rejection", err)
	}
}

func TestQueryCodegenSpannerExternalDatasetAccessUnmarshal(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		yaml                 string
		wantMode             string
		wantRole             string
		wantVerificationHint string
	}{
		{
			name: "scalar access",
			yaml: `
schemas:
- name: bigquery
  dialect: bigquery
  spanner_external_datasets:
  - dataset: analytics_spanner
    spanner_source: spanner
    access: euc
`,
			wantMode: "euc",
		},
		{
			name: "structured access",
			yaml: `
schemas:
- name: bigquery
  dialect: bigquery
  spanner_external_datasets:
  - dataset: analytics_spanner
    spanner_source: spanner
    access:
      mode: cloud_resource
      cloud_resource_connection: example-project.us.example-connection
      database_role: reader
      verification_hint: not_checked
      verification_evidence:
        status: verified
        source: external_evidence
        verifier: terraform-plan
        checked_at: "2026-05-04T10:30:00Z"
`,
			wantMode:             "cloud_resource",
			wantRole:             "reader",
			wantVerificationHint: "not_checked",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var config QueryCodegenConfig
			if err := yaml.Unmarshal([]byte(tc.yaml), &config); err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}
			access := config.Schemas[0].SpannerExternalDatasets[0].Access
			if access.Mode != tc.wantMode || access.DatabaseRole != tc.wantRole || access.VerificationHint != tc.wantVerificationHint {
				t.Fatalf("access = %+v, want mode=%q role=%q verification_hint=%q", access, tc.wantMode, tc.wantRole, tc.wantVerificationHint)
			}
		})
	}
}

func TestBuildQueryCodegenPlanBigQueryFederatedTimestampWarning(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Events (
  EventId INT64 NOT NULL,
  EventTimestamp TIMESTAMP
) PRIMARY KEY (EventId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalSchemas: []QueryCodegenExternalSchema{{
					Connection: "example-project.us.example-connection",
					Schema:     "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalEventTimes",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection: "example-project.us.example-connection",
				InnerSQL:   "SELECT EventTimestamp FROM Events",
			},
			ResultStruct: "EventRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if got, want := len(plan.Queries[0].Warnings), 1; got != want {
		t.Fatalf("len(plan.Queries[0].Warnings) = %d, want %d: %+v", got, want, plan.Queries[0].Warnings)
	}
	if plan.Queries[0].Warnings[0].Rule != "cross-dialect-timestamp-truncation" ||
		!strings.Contains(plan.Queries[0].Warnings[0].Message, "truncate TIMESTAMP nanoseconds") ||
		plan.Queries[0].Warnings[0].Remediation == "" {
		t.Fatalf("warning = %+v, want timestamp truncation warning", plan.Queries[0].Warnings[0])
	}
}

func TestGenerateQueryCodeBigQueryFederatedQueryOuterSQL(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalSchemas: []QueryCodegenExternalSchema{{
					Connection: "example-project.us.example-connection",
					Schema:     "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalSingerNames",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection:    "example-project.us.example-connection",
				SpannerSource: "spanner",
				InnerSQL:      "SELECT SingerId, FirstName FROM Singers",
				OuterSQL:      "SELECT SingerId FROM __external__ WHERE FirstName IS NOT NULL",
			},
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	want := `"SELECT SingerId FROM EXTERNAL_QUERY('example-project.us.example-connection', 'SELECT SingerId, FirstName FROM Singers') WHERE FirstName IS NOT NULL"`
	if !strings.Contains(code, want) {
		t.Fatalf("generated code missing outer SQL %q:\n%s", want, code)
	}
}

func TestGenerateQueryCodeBigQueryFederatedQueryRequiresOneOuterPlaceholder(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	_, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalSchemas: []QueryCodegenExternalSchema{{
					Connection: "example-project.us.example-connection",
					Schema:     "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalSingerIDs",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection: "example-project.us.example-connection",
				InnerSQL:   "SELECT SingerId FROM Singers",
				OuterSQL:   "SELECT * FROM __external__ UNION ALL SELECT * FROM __external__",
			},
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "outer_sql must contain exactly one __external__ placeholder") {
		t.Fatalf("GenerateQueryCode() error = %v, want placeholder count error", err)
	}
	var diagnostic QueryCodegenDiagnosticError
	if !errors.As(err, &diagnostic) {
		t.Fatalf("GenerateQueryCode() error = %T, want QueryCodegenDiagnosticError", err)
	}
	if diagnostic.ID != "external-query-placeholder-duplicate" ||
		diagnostic.Stage != "external_query_analysis" ||
		diagnostic.Subject != "external_query.outer_sql" {
		t.Fatalf("diagnostic = %+v, want external_query placeholder duplicate diagnostic", diagnostic)
	}
}

func TestGenerateQueryCodeBigQueryFederatedQueryRequiresTableExpressionPlaceholder(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	_, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalQueryConnections: []QueryCodegenExternalSchema{{
					Connection:    "example-project.us.example-connection",
					SpannerSource: "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalSingerIDs",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection: "example-project.us.example-connection",
				InnerSQL:   "SELECT SingerId FROM Singers",
				OuterSQL:   "SELECT __external__",
			},
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "placeholder must appear where a table expression is valid") {
		t.Fatalf("GenerateQueryCode() error = %v, want placeholder position error", err)
	}
}

func TestGenerateQueryCodeBigQueryFederatedQueryPlaceholderIgnoresLiteralsAndComments(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalQueryConnections: []QueryCodegenExternalSchema{{
					Connection:    "example-project.us.example-connection",
					SpannerSource: "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalSingerIDs",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection: "example-project.us.example-connection",
				InnerSQL:   "SELECT SingerId FROM Singers",
				OuterSQL: `SELECT q.SingerId, '__external__' AS literal_value
FROM __external__ AS q
-- __external__ in a comment is not a placeholder
`,
			},
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	if !strings.Contains(code, "'__external__' AS literal_value") {
		t.Fatalf("generated code did not preserve literal placeholder text:\n%s", code)
	}
}

func TestBuildQueryCodegenPlanBigQueryFederatedInnerOrderWarning(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalSchemas: []QueryCodegenExternalSchema{{
					Connection: "example-project.us.example-connection",
					Schema:     "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalSingerIDs",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection: "example-project.us.example-connection",
				InnerSQL:   "SELECT SingerId FROM Singers ORDER BY SingerId",
			},
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if got, want := len(plan.Queries[0].Warnings), 1; got != want {
		t.Fatalf("len(plan.Queries[0].Warnings) = %d, want %d: %+v", got, want, plan.Queries[0].Warnings)
	}
	if plan.Queries[0].Warnings[0].Rule != "external-query-inner-order-by-ignored" ||
		!strings.Contains(plan.Queries[0].Warnings[0].Message, "does not preserve inner query result ordering") ||
		plan.Queries[0].Warnings[0].Remediation == "" {
		t.Fatalf("warning = %+v, want inner ordering warning", plan.Queries[0].Warnings[0])
	}
}

func TestBuildQueryCodegenPlanBigQueryFederatedInnerOrderWithLimitOneIsNotWarned(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalSchemas: []QueryCodegenExternalSchema{{
					Connection: "example-project.us.example-connection",
					Schema:     "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalSingerIDs",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection: "example-project.us.example-connection",
				InnerSQL:   "SELECT SingerId FROM Singers ORDER BY SingerId LIMIT 1",
			},
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if len(plan.Queries[0].Warnings) != 0 {
		t.Fatalf("warnings = %+v, want none for ORDER BY with LIMIT 1", plan.Queries[0].Warnings)
	}
}

func TestBuildQueryCodegenPlan(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "ListSingers",
			Catalog:      "spanner",
			Table:        "Singers",
			Result:       "many",
			ResultStruct: "SingerRow",
		}},
		Writes: []QueryCodegenWrite{{
			Name:        "UpdateSingerName",
			Catalog:     "spanner",
			Table:       "Singers",
			Operation:   "update",
			InputStruct: "SingerRow",
			Update:      QueryCodegenWriteUpdate{Columns: []string{"FirstName"}},
			Methods:     []string{"mutation"},
			Vet: QueryCodegenVetConfig{Disable: []QueryCodegenVetDisable{{
				Rule:    "single-transaction-write-surface",
				Reason:  "Method choice reviewed for this package.",
				Owner:   "app-platform",
				Expires: "2026-12-31",
			}}},
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if plan.PlanVersion != 1 || plan.Generator.Name != "spanner-query-gen" || len(plan.SchemaDigests) != 1 {
		t.Fatalf("unexpected plan metadata: %+v", plan)
	}
	if plan.SchemaDigests[0].Kind != "ddl" || plan.SchemaDigests[0].SHA256 == "" {
		t.Fatalf("unexpected schema digest: %+v", plan.SchemaDigests[0])
	}
	if got, want := len(plan.Queries), 1; got != want {
		t.Fatalf("len(plan.Queries) = %d, want %d", got, want)
	}
	query := plan.Queries[0]
	if query.Kind != "table" || query.Result != "many" || query.OrderBy != "key" || query.SQLSHA256 == "" || query.SQL != "SELECT `SingerId`, `FirstName`, `LastName` FROM `Singers` ORDER BY `SingerId`" {
		t.Fatalf("unexpected query plan: %+v", query)
	}
	if got, want := len(plan.Writes), 1; got != want {
		t.Fatalf("len(plan.Writes) = %d, want %d", got, want)
	}
	write := plan.Writes[0]
	if strings.Join(write.Keys, ",") != "SingerId" || strings.Join(write.InsertColumns, ",") != "SingerId" {
		t.Fatalf("unexpected write plan: %+v", write)
	}
	if got, want := len(write.VetSuppressions), 1; got != want {
		t.Fatalf("len(write.VetSuppressions) = %d, want %d", got, want)
	}
	if write.VetSuppressions[0].Owner != "app-platform" || write.VetSuppressions[0].Expires != "2026-12-31" {
		t.Fatalf("write vet suppression = %+v, want owner/expires", write.VetSuppressions[0])
	}
}

func TestBuildQueryCodegenPlanRejectsInvalidResultMode(t *testing.T) {
	_, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Queries: []QueryCodegenQuery{{
			Name:         "GetLiteral",
			SQL:          "SELECT 1 AS value",
			Result:       "first",
			ResultStruct: "LiteralRow",
		}},
	}, "")
	if err == nil || !strings.Contains(err.Error(), `unsupported result "first"`) {
		t.Fatalf("BuildQueryCodegenPlan() error = %v, want invalid result error", err)
	}
}

func TestBuildQueryCodegenPlanRequiresVetDisableReason(t *testing.T) {
	_, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Queries: []QueryCodegenQuery{{
			Name:         "GetLiteral",
			SQL:          "SELECT 1 AS value",
			ResultStruct: "LiteralRow",
			Vet: QueryCodegenVetConfig{Disable: []QueryCodegenVetDisable{{
				Rule: "cross-dialect-timestamp-truncation",
			}}},
		}},
	}, "")
	if err == nil || !strings.Contains(err.Error(), "reason is required") {
		t.Fatalf("BuildQueryCodegenPlan() error = %v, want vet reason error", err)
	}
}

func TestBuildQueryCodegenPlanRejectsInvalidVetDisableExpires(t *testing.T) {
	_, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Queries: []QueryCodegenQuery{{
			Name:         "GetLiteral",
			SQL:          "SELECT 1 AS value",
			ResultStruct: "LiteralRow",
			Vet: QueryCodegenVetConfig{Disable: []QueryCodegenVetDisable{{
				Rule:    "cross-dialect-timestamp-truncation",
				Reason:  "Reviewed.",
				Expires: "12/31/2026",
			}}},
		}},
	}, "")
	if err == nil || !strings.Contains(err.Error(), "expires must use YYYY-MM-DD format") {
		t.Fatalf("BuildQueryCodegenPlan() error = %v, want expires format error", err)
	}
}

func TestBuildQueryCodegenPlanIncludesVetSuppressions(t *testing.T) {
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Queries: []QueryCodegenQuery{{
			Name:         "GetLiteral",
			SQL:          "SELECT 1 AS value",
			ResultStruct: "LiteralRow",
			Vet: QueryCodegenVetConfig{Disable: []QueryCodegenVetDisable{{
				Rule:    "cross-dialect-timestamp-truncation",
				Reason:  "Reviewed; downstream uses microsecond precision only.",
				Owner:   "analytics-platform",
				Expires: "2026-12-31",
			}}},
		}},
	}, "")
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	suppressions := plan.Queries[0].VetSuppressions
	if got, want := len(suppressions), 1; got != want {
		t.Fatalf("len(suppressions) = %d, want %d", got, want)
	}
	if suppressions[0].Owner != "analytics-platform" || suppressions[0].Expires != "2026-12-31" {
		t.Fatalf("suppression = %+v, want owner/expires", suppressions[0])
	}
}

func TestBuildQueryCodegenPlanRejectsDuplicateParams(t *testing.T) {
	_, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Queries: []QueryCodegenQuery{{
			Name:         "GetLiteral",
			SQL:          "SELECT @Value AS value",
			ResultStruct: "LiteralRow",
			Params: []QueryCodegenParam{
				{Name: "Value", Type: "INT64"},
				{Name: "value", Type: "INT64"},
			},
		}},
	}, "")
	if err == nil || !strings.Contains(err.Error(), `duplicate param "value"`) {
		t.Fatalf("BuildQueryCodegenPlan() error = %v, want duplicate param error", err)
	}
}

func TestBuildQueryCodegenPlanExternalQueryInnerParam(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalQueryConnections: []QueryCodegenExternalSchema{{
					Connection:    "example-project.us.example-connection",
					SpannerSource: "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalSingerIDs",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection: "example-project.us.example-connection",
				InnerSQL:   "SELECT SingerId FROM Singers WHERE SingerId = @SingerId",
			},
			ResultStruct: "SingerRow",
			Params: []QueryCodegenParam{{
				Name:  "SingerId",
				Type:  "INT64",
				Scope: "inner",
			}},
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if got, want := plan.Queries[0].Params[0].Scope, "inner"; got != want {
		t.Fatalf("plan param scope = %q, want %q", got, want)
	}
}

func TestBuildQueryCodegenPlanBigQueryParam(t *testing.T) {
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{{
			Name:    "bigquery",
			Dialect: "bigquery",
		}},
		Queries: []QueryCodegenQuery{{
			Name:         "GetValue",
			Catalog:      "bigquery",
			SQL:          "SELECT @Value AS Value",
			ResultStruct: "ValueRow",
			Params: []QueryCodegenParam{{
				Name: "Value",
				Type: "INT64",
			}},
		}},
	}, "")
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if got, want := plan.Queries[0].Fields[0].Kind, "INTEGER"; got != want {
		t.Fatalf("field kind = %q, want %q", got, want)
	}
}

func TestBuildQueryCodegenPlanExternalQueryOuterParam(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	plan, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalQueryConnections: []QueryCodegenExternalSchema{{
					Connection:    "example-project.us.example-connection",
					SpannerSource: "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalSingerIDs",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection: "example-project.us.example-connection",
				InnerSQL:   "SELECT SingerId FROM Singers",
				OuterSQL:   "SELECT * FROM __external__ WHERE SingerId = @SingerId",
			},
			ResultStruct: "SingerRow",
			Params: []QueryCodegenParam{{
				Name:  "SingerId",
				Type:  "INT64",
				Scope: "outer",
			}},
		}},
	}, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	if got, want := plan.Queries[0].Params[0].Scope, "outer"; got != want {
		t.Fatalf("plan param scope = %q, want %q", got, want)
	}
}

func TestBuildQueryCodegenPlanExternalQueryRequiresAmbiguousParamScope(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	_, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalQueryConnections: []QueryCodegenExternalSchema{{
					Connection:    "example-project.us.example-connection",
					SpannerSource: "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalSingerIDs",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection: "example-project.us.example-connection",
				InnerSQL:   "SELECT SingerId FROM Singers WHERE SingerId = @SingerId",
				OuterSQL:   "SELECT * FROM __external__ WHERE SingerId = @SingerId",
			},
			ResultStruct: "SingerRow",
			Params: []QueryCodegenParam{{
				Name: "SingerId",
				Type: "INT64",
			}},
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "scope is required because both inner_sql and outer_sql reference the parameter") {
		t.Fatalf("BuildQueryCodegenPlan() error = %v, want ambiguous scope error", err)
	}
}

func TestBuildQueryCodegenPlanRejectsScopedParamOnRegularQuery(t *testing.T) {
	_, err := BuildQueryCodegenPlan(QueryCodegenConfig{
		Package: "db",
		Queries: []QueryCodegenQuery{{
			Name:         "GetValue",
			SQL:          "SELECT @Value AS Value",
			ResultStruct: "ValueRow",
			Params: []QueryCodegenParam{{
				Name:  "Value",
				Type:  "INT64",
				Scope: "inner",
			}},
		}},
	}, "")
	if err == nil || !strings.Contains(err.Error(), "scope is only valid for kind: external_query") {
		t.Fatalf("BuildQueryCodegenPlan() error = %v, want regular-query scope error", err)
	}
}

func TestGenerateQueryCodeRejectsNonPrimaryKeyWriteKey(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE SingerAlbums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  Title STRING(MAX)
) PRIMARY KEY (SingerId, AlbumId);
`)
	_, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Writes: []QueryCodegenWrite{{
			Name:      "UpdateAlbum",
			Catalog:   "spanner",
			Table:     "SingerAlbums",
			Operation: "update",
			Keys:      []string{"SingerId"},
			Update:    QueryCodegenWriteUpdate{Columns: []string{"Title"}},
		}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "key must equal the table primary key set") {
		t.Fatalf("GenerateQueryCode() error = %v, want primary-key-set error", err)
	}
}

func TestGenerateQueryCodeRejectsConflictingSharedFieldTypes(t *testing.T) {
	_, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Queries: []QueryCodegenQuery{
			{Name: "GetInt", SQL: "SELECT 1 AS value", ResultStruct: "ValueRow"},
			{Name: "GetString", SQL: `SELECT "x" AS value`, ResultStruct: "ValueRow"},
		},
	}, "")
	if err == nil || !strings.Contains(err.Error(), "conflicting types") {
		t.Fatalf("GenerateQueryCode() error = %v, want conflicting types", err)
	}
}

func TestGenerateQueryCodeStrictRequiredRejectsUnprovenNullability(t *testing.T) {
	_, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Queries: []QueryCodegenQuery{{
			Name:           "GetValue",
			SQL:            "SELECT 1 AS value",
			ResultStruct:   "ValueRow",
			Required:       []string{"value"},
			RequiredPolicy: "strict",
		}},
	}, "")
	if err == nil || !strings.Contains(err.Error(), "nullability is not proven") {
		t.Fatalf("GenerateQueryCode() error = %v, want strict nullability error", err)
	}
}

func writeTestFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
