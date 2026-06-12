package spanalyzer

import (
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestBigQueryAnalyzerTableSchemaForStatement(t *testing.T) {
	const ddl = `
CREATE SCHEMA mydataset;

CREATE TABLE mydataset.orders (
  order_id INT64 NOT NULL,
  customer_name STRING,
  tags ARRAY<STRING>,
  profile STRUCT<age INT64, active BOOL>,
  events ARRAY<STRUCT<event_ts TIMESTAMP, event_name STRING>>,
  metadata JSON,
  valid_range RANGE<DATE>
);
`
	analyzer, err := NewBigQueryAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	schema, err := analyzer.TableSchemaForStatement(`
SELECT
  order_id,
  customer_name AS name,
  tags,
  profile,
  events,
  metadata,
  valid_range
FROM mydataset.orders`)
	if err != nil {
		t.Fatalf("TableSchemaForStatement() error = %v", err)
	}
	if got, want := len(schema.Fields), 7; got != want {
		t.Fatalf("len(schema.Fields) = %d, want %d", got, want)
	}
	assertBigQueryField(t, schema.Fields[0], "order_id", "INTEGER", "NULLABLE")
	assertBigQueryField(t, schema.Fields[1], "name", "STRING", "NULLABLE")
	assertBigQueryField(t, schema.Fields[2], "tags", "STRING", "REPEATED")
	assertBigQueryField(t, schema.Fields[3], "profile", "RECORD", "NULLABLE")
	if got, want := len(schema.Fields[3].Fields), 2; got != want {
		t.Fatalf("len(profile.Fields) = %d, want %d", got, want)
	}
	assertBigQueryField(t, schema.Fields[3].Fields[0], "age", "INTEGER", "NULLABLE")
	assertBigQueryField(t, schema.Fields[3].Fields[1], "active", "BOOLEAN", "NULLABLE")
	assertBigQueryField(t, schema.Fields[4], "events", "RECORD", "REPEATED")
	if got, want := len(schema.Fields[4].Fields), 2; got != want {
		t.Fatalf("len(events.Fields) = %d, want %d", got, want)
	}
	assertBigQueryField(t, schema.Fields[4].Fields[0], "event_ts", "TIMESTAMP", "NULLABLE")
	assertBigQueryField(t, schema.Fields[4].Fields[1], "event_name", "STRING", "NULLABLE")
	assertBigQueryField(t, schema.Fields[5], "metadata", "JSON", "NULLABLE")
	assertBigQueryField(t, schema.Fields[6], "valid_range", "RANGE", "NULLABLE")
	if schema.Fields[6].RangeElementType == nil {
		t.Fatalf("valid_range.RangeElementType is nil")
	}
	if got, want := schema.Fields[6].RangeElementType.Type, "DATE"; got != want {
		t.Fatalf("valid_range.RangeElementType.Type = %q, want %q", got, want)
	}
}

func TestBigQueryAnalyzerTableSchemaForExpression(t *testing.T) {
	analyzer, err := NewBigQueryAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	schema, err := analyzer.TableSchemaForExpression("IF(TRUE, 1, 2)")
	if err != nil {
		t.Fatalf("TableSchemaForExpression() error = %v", err)
	}
	if got, want := len(schema.Fields), 1; got != want {
		t.Fatalf("len(schema.Fields) = %d, want %d", got, want)
	}
	assertBigQueryField(t, schema.Fields[0], "expression", "INTEGER", "NULLABLE")
}

func TestBigQueryAnalyzerTableSchemaForPipeSyntax(t *testing.T) {
	const ddl = `
CREATE SCHEMA mydataset;

CREATE TABLE mydataset.produce (
  item STRING,
  sales INT64,
  category STRING
);
`
	analyzer, err := NewBigQueryAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	schema, err := analyzer.TableSchemaForStatement(`
FROM mydataset.produce
|> WHERE category = 'fruit'
|> AGGREGATE COUNT(*) AS num_items, SUM(sales) AS total_sales
   GROUP BY item
|> ORDER BY item`)
	if err != nil {
		t.Fatalf("TableSchemaForStatement() error = %v", err)
	}
	if got, want := len(schema.Fields), 3; got != want {
		t.Fatalf("len(schema.Fields) = %d, want %d", got, want)
	}
	assertBigQueryField(t, schema.Fields[0], "item", "STRING", "NULLABLE")
	assertBigQueryField(t, schema.Fields[1], "num_items", "INTEGER", "NULLABLE")
	assertBigQueryField(t, schema.Fields[2], "total_sales", "INTEGER", "NULLABLE")
}

func TestBigQueryAnalyzerSpannerExternalDataset(t *testing.T) {
	spannerCatalog, err := BuildSchemaCatalog("spanner.sql", `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  UpdatedAt TIMESTAMP,
  SearchTokens TOKENLIST AS (TOKENIZE_FULLTEXT(FirstName)) HIDDEN
) PRIMARY KEY (SingerId);
`)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	analyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", `
CREATE TABLE analytics.Customers (
  CustomerId INT64,
  FavoriteSingerId INT64
);
`)
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	binding, err := analyzer.AddSpannerExternalDataset("analytics_spanner", "app_spanner", spannerCatalog)
	if err != nil {
		t.Fatalf("AddSpannerExternalDataset() error = %v", err)
	}
	if binding.BigQueryDatasetRef.Dataset != "analytics_spanner" || binding.BigQueryDatasetRef.Path != "analytics_spanner" {
		t.Fatalf("binding.BigQueryDatasetRef = %+v, want analytics_spanner canonical ref", binding.BigQueryDatasetRef)
	}
	if binding.Access != "unknown" || binding.AccessVerification.Status != "not_checked" {
		t.Fatalf("binding access metadata = access %q status %q, want unknown/not_checked", binding.Access, binding.AccessVerification.Status)
	}
	if !binding.RequiresDataBoostAccess {
		t.Fatalf("binding.RequiresDataBoostAccess = false, want true")
	}
	if got, want := len(binding.ProjectedTables), 1; got != want {
		t.Fatalf("len(binding.ProjectedTables) = %d, want %d: %+v", got, want, binding.ProjectedTables)
	}
	if got, want := binding.ProjectedTables[0].Name, "analytics_spanner.Singers"; got != want {
		t.Fatalf("binding table name = %q, want %q", got, want)
	}
	if got, want := binding.ProjectedTables[0].BigQueryTable, "analytics_spanner.Singers"; got != want {
		t.Fatalf("binding.BigQueryTable = %q, want %q", got, want)
	}
	if got, want := binding.ProjectedTables[0].SpannerTable, "Singers"; got != want {
		t.Fatalf("binding.SpannerTable = %q, want %q", got, want)
	}
	if !binding.ProjectedTables[0].VisibleInBigQueryMetadata || binding.ProjectedTables[0].BigQueryKeyMetadataVisible {
		t.Fatalf("binding table metadata flags = visible %v key-visible %v, want true/false", binding.ProjectedTables[0].VisibleInBigQueryMetadata, binding.ProjectedTables[0].BigQueryKeyMetadataVisible)
	}
	if !binding.ProjectedTables[0].Columns[0].UnderlyingSpannerPrimaryKey || binding.ProjectedTables[0].Columns[0].BigQueryKeyMetadataVisible {
		t.Fatalf("primary key metadata flags = %+v, want underlying Spanner PK without BigQuery key metadata", binding.ProjectedTables[0].Columns[0])
	}
	if binding.ProjectedTables[0].Columns[3].Visible || binding.ProjectedTables[0].Columns[3].Reason == "" {
		t.Fatalf("hidden/tokenlist column projection = %+v, want invisible reason", binding.ProjectedTables[0].Columns[3])
	}
	if binding.ProjectedTables[0].Columns[3].VisibleInBigQueryMetadata {
		t.Fatalf("hidden/tokenlist column VisibleInBigQueryMetadata = true, want false: %+v", binding.ProjectedTables[0].Columns[3])
	}
	if len(binding.Warnings) < 2 {
		t.Fatalf("binding.Warnings = %+v, want Data Boost and omitted-column warnings", binding.Warnings)
	}
	if !warningsContainRule(binding.Warnings, "external-dataset-timestamp-caveat") {
		t.Fatalf("binding.Warnings = %+v, want TIMESTAMP caveat warning", binding.Warnings)
	}

	schema, err := analyzer.TableSchemaForStatement(`
SELECT c.CustomerId, s.SingerId, s.FirstName, s.UpdatedAt
FROM analytics.Customers AS c
JOIN analytics_spanner.Singers AS s
  ON s.SingerId = c.FavoriteSingerId`)
	if err != nil {
		t.Fatalf("TableSchemaForStatement() error = %v", err)
	}
	if got, want := len(schema.Fields), 4; got != want {
		t.Fatalf("len(schema.Fields) = %d, want %d", got, want)
	}
	assertBigQueryField(t, schema.Fields[0], "CustomerId", "INTEGER", "NULLABLE")
	assertBigQueryField(t, schema.Fields[1], "SingerId", "INTEGER", "NULLABLE")
	assertBigQueryField(t, schema.Fields[2], "FirstName", "STRING", "NULLABLE")
	assertBigQueryField(t, schema.Fields[3], "UpdatedAt", "TIMESTAMP", "NULLABLE")
}

func TestBigQueryAnalyzerSpannerExternalDatasetCanonicalProjectAndAlias(t *testing.T) {
	spannerCatalog, err := BuildSchemaCatalog("spanner.sql", `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	analyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	binding, err := analyzer.AddSpannerExternalDatasetWithOptions("analytics_spanner", "app_spanner", spannerCatalog, BigQuerySpannerExternalDatasetOptions{
		DefaultProject: "example-project",
		Access:         "cloud_resource",
		DatabaseRole:   "reader",
		AccessVerification: BigQuerySpannerExternalDatasetVerification{
			Status:    "verified",
			Source:    "external_evidence",
			Verifier:  "terraform-plan",
			CheckedAt: "2026-05-04T10:30:00Z",
		},
	})
	if err != nil {
		t.Fatalf("AddSpannerExternalDatasetWithOptions() error = %v", err)
	}
	if got, want := binding.BigQueryDataset, "example-project.analytics_spanner"; got != want {
		t.Fatalf("binding.BigQueryDataset = %q, want %q", got, want)
	}
	if got, want := binding.BigQueryDatasetRef.Project, "example-project"; got != want {
		t.Fatalf("binding.BigQueryDatasetRef.Project = %q, want %q", got, want)
	}
	if got, want := binding.Access, "cloud_resource_connection"; got != want {
		t.Fatalf("binding.Access = %q, want %q", got, want)
	}
	if binding.AccessVerification.Status != "verified" || binding.DatabaseRole != "reader" {
		t.Fatalf("binding access metadata = status %q role %q, want verified reader", binding.AccessVerification.Status, binding.DatabaseRole)
	}
	if got, want := binding.AccessVerification.Status, "verified"; got != want {
		t.Fatalf("binding.AccessVerification.Status = %q, want %q", got, want)
	}
	for _, sql := range []string{
		"SELECT SingerId FROM `example-project.analytics_spanner.Singers`",
		"SELECT SingerId FROM `analytics_spanner.Singers`",
		"SELECT SingerId FROM analytics_spanner.Singers",
		"SELECT SingerId FROM analytics_spanner.singers",
	} {
		schema, err := analyzer.TableSchemaForStatement(sql)
		if err != nil {
			t.Fatalf("TableSchemaForStatement(%q) error = %v", sql, err)
		}
		if got, want := len(schema.Fields), 1; got != want {
			t.Fatalf("len(schema.Fields) = %d, want %d for %q", got, want, sql)
		}
		assertBigQueryField(t, schema.Fields[0], "SingerId", "INTEGER", "NULLABLE")
	}
}

func TestBigQueryAnalyzerSpannerExternalDatasetRejectsVerificationWithoutEvidence(t *testing.T) {
	spannerCatalog, err := BuildSchemaCatalog("spanner.sql", `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	analyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	_, err = analyzer.AddSpannerExternalDatasetWithOptions("analytics_spanner", "app_spanner", spannerCatalog, BigQuerySpannerExternalDatasetOptions{
		AccessVerificationStatus: "verified",
	})
	if err == nil || !strings.Contains(err.Error(), `access verification status "verified" requires verifier and checked_at`) {
		t.Fatalf("AddSpannerExternalDatasetWithOptions() error = %v, want evidence validation error", err)
	}
}

func TestBigQueryAnalyzerSpannerExternalDatasetOmitPoliciesStillRejectExplicitReferences(t *testing.T) {
	spannerCatalog := &Catalog{
		Tables: map[string]*Table{
			"Singers": {
				Name: ObjectName{Parts: []string{"Singers"}},
				Columns: []*Column{
					{Name: "SingerId", Type: &TypeSpec{Code: spannerpb.TypeCode_INT64}},
					{Name: "Profile", Type: &TypeSpec{Code: spannerpb.TypeCode_STRUCT}},
				},
			},
			"analytics.Events": {
				Name: ObjectName{Parts: []string{"analytics", "Events"}},
				Columns: []*Column{{
					Name: "EventId",
					Type: &TypeSpec{Code: spannerpb.TypeCode_INT64},
				}},
			},
		},
	}
	analyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	if _, err := analyzer.AddSpannerExternalDatasetWithOptions("analytics_spanner", "app_spanner", spannerCatalog, BigQuerySpannerExternalDatasetOptions{
		UnsupportedColumns: "omit",
		NamedSchemaPolicy:  "warn_and_omit",
	}); err != nil {
		t.Fatalf("AddSpannerExternalDatasetWithOptions() error = %v", err)
	}
	starSchema, err := analyzer.TableSchemaForStatement("SELECT * FROM analytics_spanner.Singers")
	if err != nil {
		t.Fatalf("TableSchemaForStatement(SELECT *) error = %v", err)
	}
	if got, want := len(starSchema.Fields), 1; got != want {
		t.Fatalf("SELECT * field count = %d, want %d", got, want)
	}
	assertBigQueryField(t, starSchema.Fields[0], "SingerId", "INTEGER", "NULLABLE")

	for _, tc := range []struct {
		name string
		sql  string
		want string
	}{
		{
			name: "unsupported omitted column",
			sql:  "SELECT Profile FROM analytics_spanner.Singers",
			want: "Unrecognized name: Profile",
		},
		{
			name: "omitted named schema table",
			sql:  "SELECT EventId FROM analytics_spanner.analytics.Events",
			want: "Table not found",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := analyzer.TableSchemaForStatement(tc.sql)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("TableSchemaForStatement() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestBigQueryAnalyzerSpannerExternalDatasetRejectsInformationSchemaAndDML(t *testing.T) {
	spannerCatalog, err := BuildSchemaCatalog("spanner.sql", `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	analyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	if _, err := analyzer.AddSpannerExternalDataset("analytics_spanner", "app_spanner", spannerCatalog); err != nil {
		t.Fatalf("AddSpannerExternalDataset() error = %v", err)
	}
	for _, tc := range []struct {
		name string
		sql  string
		want string
	}{
		{
			name: "information schema",
			sql:  "SELECT * FROM analytics_spanner.INFORMATION_SCHEMA.TABLES",
			want: "Table not found",
		},
		{
			name: "dml has no table schema row type",
			sql:  "INSERT INTO analytics_spanner.Singers (SingerId) VALUES (1)",
			want: "external dataset table analytics_spanner.Singers is read-only and cannot be used as dml_target",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := analyzer.TableSchemaForStatement(tc.sql)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("TableSchemaForStatement() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestBigQueryAnalyzerSpannerExternalDatasetRejectsCaseInsensitiveTableNameConflict(t *testing.T) {
	spannerCatalog := &Catalog{
		Tables: map[string]*Table{
			"Singers": {
				Name: ObjectName{Parts: []string{"Singers"}},
				Columns: []*Column{{
					Name: "SingerId",
					Type: &TypeSpec{Code: spannerpb.TypeCode_INT64},
				}},
			},
			"singers": {
				Name: ObjectName{Parts: []string{"singers"}},
				Columns: []*Column{{
					Name: "SingerId",
					Type: &TypeSpec{Code: spannerpb.TypeCode_INT64},
				}},
			},
		},
	}
	analyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	_, err = analyzer.AddSpannerExternalDataset("analytics_spanner", "app_spanner", spannerCatalog)
	if err == nil || !strings.Contains(err.Error(), "case-insensitive") {
		t.Fatalf("AddSpannerExternalDataset() error = %v, want case-insensitive conflict", err)
	}
}

func TestBigQueryAnalyzerSpannerExternalDatasetRejectsCaseInsensitiveColumnNameConflict(t *testing.T) {
	spannerCatalog := &Catalog{
		Tables: map[string]*Table{
			"Singers": {
				Name: ObjectName{Parts: []string{"Singers"}},
				Columns: []*Column{
					{Name: "SingerId", Type: &TypeSpec{Code: spannerpb.TypeCode_INT64}},
					{Name: "Foo", Type: &TypeSpec{Code: spannerpb.TypeCode_STRING}},
					{Name: "foo", Type: &TypeSpec{Code: spannerpb.TypeCode_STRING}},
				},
			},
		},
	}
	analyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	_, err = analyzer.AddSpannerExternalDataset("analytics_spanner", "app_spanner", spannerCatalog)
	if err == nil || !strings.Contains(err.Error(), "projected column names are case-insensitive") {
		t.Fatalf("AddSpannerExternalDataset() error = %v, want case-insensitive column conflict", err)
	}
}

func TestBigQueryAnalyzerSpannerExternalDatasetRejectsCatalogTableCollision(t *testing.T) {
	spannerCatalog, err := BuildSchemaCatalog("spanner.sql", `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	for _, tc := range []struct {
		name    string
		ddl     string
		options BigQuerySpannerExternalDatasetOptions
	}{
		{
			name: "direct dataset path",
			ddl: `
CREATE SCHEMA analytics_spanner;
CREATE TABLE analytics_spanner.Singers (SingerId INT64);
`,
		},
		{
			name: "unqualified alias from default project",
			ddl: `
CREATE SCHEMA analytics_spanner;
CREATE TABLE analytics_spanner.Singers (SingerId INT64);
`,
			options: BigQuerySpannerExternalDatasetOptions{DefaultProject: "example-project"},
		},
		{
			name: "project qualified path",
			ddl: `
CREATE SCHEMA ` + "`example-project`" + `.analytics_spanner;
CREATE TABLE ` + "`example-project`" + `.analytics_spanner.Singers (SingerId INT64);
`,
			options: BigQuerySpannerExternalDatasetOptions{DefaultProject: "example-project"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			analyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", tc.ddl)
			if err != nil {
				t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
			}
			_, err = analyzer.AddSpannerExternalDatasetWithOptions("analytics_spanner", "app_spanner", spannerCatalog, tc.options)
			if err == nil || !strings.Contains(err.Error(), "bigquery catalog table collision") {
				t.Fatalf("AddSpannerExternalDatasetWithOptions() error = %v, want catalog table collision", err)
			}
		})
	}
}

func TestBigQueryAnalyzerSpannerExternalDatasetProjectionPolicies(t *testing.T) {
	spannerCatalog := &Catalog{
		Tables: map[string]*Table{
			"Singers": {
				Name: ObjectName{Parts: []string{"Singers"}},
				Columns: []*Column{
					{Name: "SingerId", Type: &TypeSpec{Code: spannerpb.TypeCode_INT64}},
					{Name: "Profile", Type: &TypeSpec{Code: spannerpb.TypeCode_STRUCT}},
				},
			},
			"analytics.Events": {
				Name: ObjectName{Parts: []string{"analytics", "Events"}},
				Columns: []*Column{{
					Name: "EventId",
					Type: &TypeSpec{Code: spannerpb.TypeCode_INT64},
				}},
			},
		},
	}
	analyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	binding, err := analyzer.AddSpannerExternalDatasetWithOptions("analytics_spanner", "app_spanner", spannerCatalog, BigQuerySpannerExternalDatasetOptions{
		UnsupportedColumns: "omit",
		NamedSchemaPolicy:  "warn_and_omit",
	})
	if err != nil {
		t.Fatalf("AddSpannerExternalDatasetWithOptions() error = %v", err)
	}
	if got, want := binding.ProjectionPolicy.UnsupportedColumns, "omit"; got != want {
		t.Fatalf("UnsupportedColumns policy = %q, want %q", got, want)
	}
	if got, want := len(binding.ProjectedTables), 2; got != want {
		t.Fatalf("len(binding.ProjectedTables) = %d, want %d: %+v", got, want, binding.ProjectedTables)
	}
	if binding.ProjectedTables[0].Visible && binding.ProjectedTables[1].Visible {
		t.Fatalf("binding.ProjectedTables = %+v, want one omitted named-schema table", binding.ProjectedTables)
	}
	if len(binding.Warnings) < 4 {
		t.Fatalf("binding.Warnings = %+v, want access, Data Boost, unsupported-column, and named-schema warnings", binding.Warnings)
	}

	analyzer, err = NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	_, err = analyzer.AddSpannerExternalDatasetWithOptions("analytics_spanner", "app_spanner", spannerCatalog, BigQuerySpannerExternalDatasetOptions{
		UnsupportedColumns: "error",
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported Spanner column") {
		t.Fatalf("AddSpannerExternalDatasetWithOptions() error = %v, want unsupported-column policy error", err)
	}

	analyzer, err = NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	_, err = analyzer.AddSpannerExternalDatasetWithOptions("analytics_spanner", "app_spanner", spannerCatalog, BigQuerySpannerExternalDatasetOptions{
		NamedSchemaPolicy: "error",
	})
	if err == nil || !strings.Contains(err.Error(), "named Spanner schemas are not visible") {
		t.Fatalf("AddSpannerExternalDatasetWithOptions() error = %v, want named-schema policy error", err)
	}
}

func TestBigQueryAnalyzerExternalQueryRewrite(t *testing.T) {
	const bigQueryDDL = `
CREATE TABLE mydataset.customers (
  customer_id INT64,
  name STRING
);
`
	const spannerDDL = `
CREATE TABLE Orders (
  CustomerId INT64 NOT NULL,
  OrderDate DATE
) PRIMARY KEY (CustomerId);
`
	bigQueryAnalyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", bigQueryDDL)
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	spannerAnalyzer, err := NewAnalyzerFromDDL("spanner.sql", spannerDDL)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	bigQueryAnalyzer.SetExternalQueryAnalyzers(map[string]*Analyzer{
		"my-project.us.example-db": spannerAnalyzer,
	})

	schema, err := bigQueryAnalyzer.TableSchemaForStatement(`
SELECT c.customer_id, c.name, rq.first_order_date
FROM mydataset.customers AS c
LEFT JOIN EXTERNAL_QUERY(
  'my-project.us.example-db',
  '''SELECT CustomerId AS customer_id, MIN(OrderDate) AS first_order_date
     FROM Orders
     GROUP BY CustomerId''') AS rq
  ON rq.customer_id = c.customer_id`)
	if err != nil {
		t.Fatalf("TableSchemaForStatement() error = %v", err)
	}
	if got, want := len(schema.Fields), 3; got != want {
		t.Fatalf("len(schema.Fields) = %d, want %d", got, want)
	}
	assertBigQueryField(t, schema.Fields[0], "customer_id", "INTEGER", "NULLABLE")
	assertBigQueryField(t, schema.Fields[1], "name", "STRING", "NULLABLE")
	assertBigQueryField(t, schema.Fields[2], "first_order_date", "DATE", "NULLABLE")
}

func TestBigQueryAnalyzerExternalQueryTVFDirectAnalyze(t *testing.T) {
	spannerAnalyzer, err := NewAnalyzerFromDDL("spanner.sql", `
CREATE TABLE Orders (
  CustomerId INT64 NOT NULL,
  OrderDate DATE
) PRIMARY KEY (CustomerId);
`)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	bigQueryAnalyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	if !bigQueryAnalyzer.googleSQL.externalQueryTVFRegistered {
		t.Fatalf("EXTERNAL_QUERY TVF is not registered")
	}
	bigQueryAnalyzer.SetExternalQueryAnalyzers(map[string]*Analyzer{
		"my-project.us.example-db": spannerAnalyzer,
	})

	sql := `
SELECT customer_id, first_order_date
FROM EXTERNAL_QUERY(
  'my-project.us.example-db',
  '''SELECT CustomerId AS customer_id, MIN(OrderDate) AS first_order_date
     FROM Orders
     GROUP BY CustomerId''')`
	schema, err := bigQueryAnalyzer.TableSchemaForStatement(sql)
	if err != nil {
		t.Fatalf("TableSchemaForStatement() error = %v", err)
	}
	if got, want := len(schema.Fields), 2; got != want {
		t.Fatalf("len(schema.Fields) = %d, want %d", got, want)
	}
	assertBigQueryField(t, schema.Fields[0], "customer_id", "INTEGER", "NULLABLE")
	assertBigQueryField(t, schema.Fields[1], "first_order_date", "DATE", "NULLABLE")

	debug, err := bigQueryAnalyzer.ResolvedASTDebugString("query", sql)
	if err != nil {
		t.Fatalf("ResolvedASTDebugString() error = %v", err)
	}
	if !strings.Contains(debug, "EXTERNAL_QUERY") {
		t.Fatalf("ResolvedASTDebugString() = %s, want EXTERNAL_QUERY TVF in resolved AST", debug)
	}
}

func TestBigQueryAnalyzerExternalQueryConnectionsUseDifferentSpannerSchemas(t *testing.T) {
	spannerAnalyzerA, err := NewAnalyzerFromDDL("a.sql", `
CREATE TABLE Orders (
  Id INT64 NOT NULL,
  Value STRING(MAX)
) PRIMARY KEY (Id);
`)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL(a) error = %v", err)
	}
	spannerAnalyzerB, err := NewAnalyzerFromDDL("b.sql", `
CREATE TABLE Orders (
  Id INT64 NOT NULL,
  Value DATE
) PRIMARY KEY (Id);
`)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL(b) error = %v", err)
	}
	bigQueryAnalyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	bigQueryAnalyzer.SetExternalQueryAnalyzers(map[string]*Analyzer{
		"conn-a": spannerAnalyzerA,
		"conn-b": spannerAnalyzerB,
	})

	schema, err := bigQueryAnalyzer.TableSchemaForStatement(`
SELECT a.value AS string_value, b.value AS date_value
FROM EXTERNAL_QUERY('conn-a', '''SELECT Value AS value FROM Orders''') AS a
CROSS JOIN EXTERNAL_QUERY('conn-b', '''SELECT Value AS value FROM Orders''') AS b`)
	if err != nil {
		t.Fatalf("TableSchemaForStatement() error = %v", err)
	}
	if got, want := len(schema.Fields), 2; got != want {
		t.Fatalf("len(schema.Fields) = %d, want %d", got, want)
	}
	assertBigQueryField(t, schema.Fields[0], "string_value", "STRING", "NULLABLE")
	assertBigQueryField(t, schema.Fields[1], "date_value", "DATE", "NULLABLE")
}

func TestBigQueryAnalyzerExternalQueryProtoFieldsAreValidInConnectionScope(t *testing.T) {
	ddl, err := os.ReadFile("testdata/order-proto-schema.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	spannerAnalyzer, err := NewAnalyzerFromDDLWithProtoDescriptorFiles(
		"testdata/order-proto-schema.sql",
		string(ddl),
		[]string{"testdata/protos/order_descriptors.pb"},
	)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDLWithProtoDescriptorFiles() error = %v", err)
	}
	bigQueryAnalyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	bigQueryAnalyzer.SetExternalQueryAnalyzers(map[string]*Analyzer{
		"proto-conn": spannerAnalyzer,
	})

	schema, err := bigQueryAnalyzer.TableSchemaForStatement(`
SELECT q.order_number
FROM EXTERNAL_QUERY(
  'proto-conn',
  '''SELECT OrderInfo.order_number AS order_number FROM Orders''') AS q`)
	if err != nil {
		t.Fatalf("TableSchemaForStatement() error = %v", err)
	}
	if got, want := len(schema.Fields), 1; got != want {
		t.Fatalf("len(schema.Fields) = %d, want %d", got, want)
	}
	assertBigQueryField(t, schema.Fields[0], "order_number", "STRING", "NULLABLE")
}

func TestBigQueryAnalyzerExternalQueryRejectsStructOutput(t *testing.T) {
	spannerAnalyzer, err := NewAnalyzerFromDDL("spanner.sql", `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	bigQueryAnalyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	bigQueryAnalyzer.SetExternalQueryAnalyzers(map[string]*Analyzer{
		"spanner-conn": spannerAnalyzer,
	})

	_, err = bigQueryAnalyzer.TableSchemaForStatement(`
SELECT q.singer
FROM EXTERNAL_QUERY(
  'spanner-conn',
  '''SELECT STRUCT(SingerId AS SingerId, FirstName AS FirstName) AS singer FROM Singers''') AS q`)
	if err == nil || !strings.Contains(err.Error(), "STRUCT is unsupported for Spanner federated query output") {
		t.Fatalf("TableSchemaForStatement() error = %v, want STRUCT rejection", err)
	}
}

func assertBigQueryField(t *testing.T, field *BigQueryTableFieldSchema, name, typ, mode string) {
	t.Helper()
	if field == nil {
		t.Fatalf("field %s is nil", name)
	}
	if field.Name != name {
		t.Fatalf("field.Name = %q, want %q", field.Name, name)
	}
	if field.Type != typ {
		t.Fatalf("field.Type = %q, want %q", field.Type, typ)
	}
	if field.Mode != mode {
		t.Fatalf("field.Mode = %q, want %q", field.Mode, mode)
	}
}

func warningsContainRule(warnings []QueryCodegenPlanWarning, rule string) bool {
	for _, warning := range warnings {
		if warning.Rule == rule {
			return true
		}
	}
	return false
}

func TestBigQueryAnalyzerExternalQueryOptionsArgument(t *testing.T) {
	analyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}

	_, err = analyzer.TableSchemaForStatement(`
SELECT * FROM EXTERNAL_QUERY('conn', 'SELECT 1 AS x', '{"options": true}')`)
	if err == nil || !strings.Contains(err.Error(), "EXTERNAL_QUERY options argument is currently not supported for static analysis") {
		t.Fatalf("TableSchemaForStatement() error = %v, want options argument error", err)
	}
}

func TestBigQueryAnalyzerExternalQueryNestedCalls(t *testing.T) {
	spannerAnalyzerOuter, err := NewAnalyzerFromDDL("a.sql", `
CREATE TABLE Orders (
  Id INT64 NOT NULL,
  Value STRING(MAX)
) PRIMARY KEY (Id);
`)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL(a) error = %v", err)
	}
	spannerAnalyzerInner, err := NewAnalyzerFromDDL("b.sql", `
CREATE TABLE Customers (
  Id INT64 NOT NULL,
  Name STRING(MAX)
) PRIMARY KEY (Id);
`)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL(b) error = %v", err)
	}
	bigQueryAnalyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	bigQueryAnalyzer.SetExternalQueryAnalyzers(map[string]*Analyzer{
		"outer-conn": spannerAnalyzerOuter,
		"inner-conn": spannerAnalyzerInner,
	})

	// The inner call is in a scalar subquery.
	schema, err := bigQueryAnalyzer.TableSchemaForStatement(`
SELECT o.value, (SELECT c.Name FROM EXTERNAL_QUERY('inner-conn', '''SELECT Name FROM Customers''') AS c LIMIT 1) AS inner_val
FROM EXTERNAL_QUERY('outer-conn', '''SELECT Value AS value FROM Orders''') AS o`)
	if err != nil {
		t.Fatalf("TableSchemaForStatement() error = %v", err)
	}
	if got, want := len(schema.Fields), 2; got != want {
		t.Fatalf("len(schema.Fields) = %d, want %d", got, want)
	}
	if schema.Fields[0].Name != "value" || schema.Fields[1].Name != "inner_val" {
		t.Fatalf("fields: %+v", schema.Fields)
	}
}

func TestBigQueryAnalyzerParseDebugStringExternalQuery(t *testing.T) {
	analyzer, err := NewBigQueryAnalyzerFromDDL("bq.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}

	// Non-literal connection ID. Parser should accept it, but analysis would fail.
	sql := "SELECT * FROM EXTERNAL_QUERY(CONCAT('a', 'b'), 'SELECT 1')"
	debug, err := analyzer.ParseDebugString("query", sql)
	if err != nil {
		t.Errorf("ParseDebugString() error = %v, want nil", err)
	}
	if !strings.Contains(debug, "TVF") {
		t.Errorf("ParseDebugString() output does not contain TVF: %s", debug)
	}

	// Unparse should also work.
	unparsed, err := analyzer.Unparse("query", sql)
	if err != nil {
		t.Errorf("Unparse() error = %v, want nil", err)
	}
	if !strings.Contains(unparsed, "EXTERNAL_QUERY") {
		t.Errorf("Unparse() output does not contain EXTERNAL_QUERY: %s", unparsed)
	}
}

func TestBigQueryAnalyzerDialectFeatures(t *testing.T) {
	analyzer, err := NewBigQueryAnalyzerFromDDL("bq.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}

	tests := []struct {
		name    string
		sql     string
		wantErr string
	}{
		{
			name: "Pipe syntax is supported in BigQuery",
			sql:  "SELECT 1 AS x |> SELECT x + 1",
		},
		{
			name: "QUALIFY with analytic function is supported in BigQuery",
			sql:  "SELECT 1 AS x FROM (SELECT 1) QUALIFY ROW_NUMBER() OVER() > 0",
		},
		{
			name: "JSON subscript/path navigation is supported in BigQuery",
			sql:  "SELECT JSON '{\"a\": 1}'.a",
		},
		{
			name: "BigQuery specific functions (e.g. GENERATE_UUID) are supported",
			sql:  "SELECT GENERATE_UUID()",
		},
		{
			name: "BIGNUMERIC is supported in BigQuery",
			sql:  "SELECT BIGNUMERIC '123'",
		},
		{
			name: "GEOGRAPHY is supported in BigQuery",
			sql:  "SELECT ST_GEOGPOINT(1, 2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := analyzer.TableSchemaForStatement(tt.sql)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("TableSchemaForStatement() error = %v, want nil", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("TableSchemaForStatement() error = %v, want error containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestSpannerAnalyzerDialectFeatures(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("spanner.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}

	tests := []struct {
		name    string
		sql     string
		wantErr string
	}{
		{
			name: "Pipe syntax is supported in Spanner",
			sql:  "SELECT 1 AS x |> SELECT x + 1",
		},
		{
			name: "QUALIFY is supported in Spanner",
			sql:  "SELECT 1 AS x FROM (SELECT 1) QUALIFY ROW_NUMBER() OVER() > 0",
		},
		{
			name: "JSON subscript/path navigation IS supported in Spanner",
			sql:  "SELECT JSON '{\"a\": 1}'.a",
		},
		{
			name:    "BIGNUMERIC is NOT supported in Spanner",
			sql:     "SELECT BIGNUMERIC '123'",
			wantErr: "BIGNUMERIC literals are not supported",
		},
		{
			name:    "GEOGRAPHY is NOT supported in Spanner",
			sql:     "SELECT ST_GEOGPOINT(1, 2)",
			wantErr: "Function not found: ST_GEOGPOINT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := analyzer.RowTypeForStatement(tt.sql)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("RowTypeForStatement() error = %v, want nil", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("RowTypeForStatement() error = %v, want error containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestBuildBigQueryGoogleSQLCatalogFromDDL_ExpandedSupport(t *testing.T) {
	const ddl = `
CREATE SCHEMA mydataset;
CREATE TABLE mydataset.mytable (x INT64);
CREATE VIEW mydataset.myview AS SELECT * FROM mydataset.mytable;
CREATE INDEX myindex ON mydataset.mytable(x);
DROP TABLE mydataset.mytable;
CREATE TABLE mydataset.mytable2 (y STRING);
CREATE TABLE FUNCTION mydataset.myfunc(p INT64) AS SELECT p AS val;
CREATE PROCEDURE mydataset.myproc() BEGIN SELECT 1; END;
EXPORT DATA OPTIONS(uri='gs://bucket/file') AS SELECT 1;
`
	_, err := BuildBigQueryGoogleSQLCatalogFromDDL("bigquery.sql", ddl)
	if err != nil {
		t.Fatalf("BuildBigQueryGoogleSQLCatalogFromDDL() error = %v", err)
	}
}

// TestBigQueryAnalyzerExternalQueryDuplicateIdenticalInnerSQL pins the
// row-type map invariant: two EXTERNAL_QUERY calls with the same connection
// and the same inner SQL collapse to a single connectionID+innerSQL map
// entry, which is correct because their row types are identical, and both
// call sites resolve.
func TestBigQueryAnalyzerExternalQueryDuplicateIdenticalInnerSQL(t *testing.T) {
	spannerAnalyzer, err := NewAnalyzerFromDDL("spanner.sql", `
CREATE TABLE Orders (
  CustomerId INT64 NOT NULL,
  OrderDate DATE
) PRIMARY KEY (CustomerId);
`)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	analyzer, err := NewBigQueryAnalyzerFromDDL("bigquery.sql", "")
	if err != nil {
		t.Fatalf("NewBigQueryAnalyzerFromDDL() error = %v", err)
	}
	analyzer.SetExternalQueryAnalyzers(map[string]*Analyzer{
		"my-project.us.example-db": spannerAnalyzer,
	})

	schema, err := analyzer.TableSchemaForStatement(`
SELECT a.customer_id
FROM EXTERNAL_QUERY(
  'my-project.us.example-db',
  '''SELECT CustomerId AS customer_id FROM Orders''') AS a
JOIN EXTERNAL_QUERY(
  'my-project.us.example-db',
  '''SELECT CustomerId AS customer_id FROM Orders''') AS b
  ON a.customer_id = b.customer_id`)
	if err != nil {
		t.Fatalf("TableSchemaForStatement() error = %v", err)
	}
	if got, want := len(schema.Fields), 1; got != want {
		t.Fatalf("len(schema.Fields) = %d, want %d", got, want)
	}
	assertBigQueryField(t, schema.Fields[0], "customer_id", "INTEGER", "NULLABLE")

	// The prepared map is keyed by connectionID then inner SQL, so the two
	// identical calls above must have produced exactly one entry.
	if err := analyzer.prepareExternalQueryTVFCalls(`
SELECT *
FROM EXTERNAL_QUERY('my-project.us.example-db', 'SELECT CustomerId FROM Orders') AS a,
     EXTERNAL_QUERY('my-project.us.example-db', 'SELECT CustomerId FROM Orders') AS b`); err != nil {
		t.Fatalf("prepareExternalQueryTVFCalls() error = %v", err)
	}
	defer analyzer.googleSQL.clearExternalQueryTVFCalls()
	if got, want := len(analyzer.googleSQL.externalQueryRowTypes["my-project.us.example-db"]), 1; got != want {
		t.Fatalf("externalQueryRowTypes entries = %d, want %d (identical inner SQL must collapse)", got, want)
	}
}
