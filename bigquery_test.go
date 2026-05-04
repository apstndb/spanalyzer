package spanalyzer

import (
	"os"
	"testing"
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
