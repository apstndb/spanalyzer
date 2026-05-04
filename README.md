# go-googlesql-spanner-poc

[![Go Reference](https://pkg.go.dev/badge/github.com/apstndb/go-googlesql-spanner-poc.svg)](https://pkg.go.dev/github.com/apstndb/go-googlesql-spanner-poc)

`go-googlesql-spanner-poc` is an experimental Go library and CLI for deriving
Cloud Spanner GoogleSQL query result row types from Spanner DDL.

It parses Spanner DDL with
[`github.com/cloudspannerecosystem/memefish`](https://github.com/cloudspannerecosystem/memefish)
and analyzes queries with
[`github.com/goccy/go-googlesql`](https://github.com/goccy/go-googlesql).

In this document, "GoogleSQL frontend" refers to the analyzer and catalog
library formerly named ZetaSQL. "Spanner GoogleSQL" refers to
[Cloud Spanner's SQL dialect](https://cloud.google.com/spanner/docs/reference/standard-sql/query-syntax).
Historical ZetaSQL names appear only when referring to upstream API names or
repositories that still use them.

## Usage

```sh
go run ./cmd/spanner-analyzer \
  --ddl testdata/order-proto-schema.sql \
  --proto-descriptors-file testdata/protos/order_descriptors.pb \
  --sql 'SELECT OrderInfo.order_number FROM Orders'
```

Output:

```yaml
fields:
- name: order_number
  type:
    code: STRING
```

`--proto-descriptors-file` accepts a Protocol Buffers `FileDescriptorSet` used
to resolve types named by `CREATE PROTO BUNDLE` or `ALTER PROTO BUNDLE`. The
flag is repeatable.

`--ddl` is optional. Queries that only use built-in functions, parameters,
`INFORMATION_SCHEMA`, or `SPANNER_SYS` can be analyzed without a schema file:

```sh
go run ./cmd/spanner-analyzer \
  --sql 'SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES'
```

Registered GoogleSQL frontend and Spanner function signatures can be dumped
with the dedicated function catalog command:

```sh
go run ./cmd/spanner-function-catalog
```

Use `--verbose=false` to print only function names. `--ddl` and
`--proto-descriptors-file` are also accepted when the catalog depends on schema
objects or proto descriptors.

The repository includes the Protocol Buffers example from the
[Cloud Spanner protocol buffers reference](https://cloud.google.com/spanner/docs/reference/standard-sql/protocol-buffers)
under `testdata/protos/`, including
`order_protos.proto` and its compiled `order_descriptors.pb` descriptor set.

Named query parameters can be declared with `--param name=TYPE`. Positional
parameters can be declared with repeatable `--positional-param TYPE`.

```sh
go run ./cmd/spanner-analyzer \
  --ddl schema.sql \
  --sql 'SELECT @id AS id' \
  --param id=INT64
```

More complex queries can mix aggregate functions, conditional expressions, and
proto field access. The output is still the Cloud Spanner result row type, not
query data.

```sh
go run ./cmd/spanner-analyzer \
  --ddl testdata/order-proto-schema.sql \
  --proto-descriptors-file testdata/protos/order_descriptors.pb \
  --sql '
    SELECT
      COUNT(*) AS order_count,
      SUM(Id) AS id_sum,
      AVG(Id) AS avg_id,
      IF(COUNT(*) > 0, "nonempty", "empty") AS status,
      CASE WHEN MAX(Id) >= 100 THEN "large" ELSE "small" END AS id_bucket,
      COALESCE(MIN(OrderInfo.order_number), "none") AS first_order_number
    FROM Orders'
```

Output:

```yaml
fields:
- name: order_count
  type:
    code: INT64
- name: id_sum
  type:
    code: INT64
- name: avg_id
  type:
    code: FLOAT64
- name: status
  type:
    code: STRING
- name: id_bucket
  type:
    code: STRING
- name: first_order_number
  type:
    code: STRING
```

`--sql-mode expression` analyzes a single GoogleSQL expression and returns a
single Spanner `Type` instead of a query result row type.

```sh
go run ./cmd/spanner-analyzer \
  --ddl testdata/order-proto-schema.sql \
  --proto-descriptors-file testdata/protos/order_descriptors.pb \
  --sql-mode expression \
  --sql 'AI.SCORE(@prompt)' \
  --param 'prompt=STRING(MAX)'
```

Output:

```yaml
code: FLOAT64
```

Polymorphic functions resolve their return type from the argument type.

```sh
go run ./cmd/spanner-analyzer \
  --ddl testdata/order-proto-schema.sql \
  --proto-descriptors-file testdata/protos/order_descriptors.pb \
  --sql-mode expression \
  --sql 'ARRAY_FIRST([1, 2, 3])'
```

Output:

```yaml
code: INT64
```

```sh
go run ./cmd/spanner-analyzer \
  --ddl testdata/order-proto-schema.sql \
  --proto-descriptors-file testdata/protos/order_descriptors.pb \
  --sql-mode expression \
  --sql 'ARRAY_FIRST(["a", "b"])'
```

Output:

```yaml
code: STRING
```

Cloud Spanner `INFORMATION_SCHEMA` tables are registered as built-in catalog
tables for analysis. They provide names and column types only; no row data is
materialized.

```sh
go run ./cmd/spanner-analyzer \
  --ddl testdata/order-proto-schema.sql \
  --proto-descriptors-file testdata/protos/order_descriptors.pb \
  --sql 'SELECT TABLE_NAME, COLUMN_NAME, ORDINAL_POSITION, SPANNER_TYPE
         FROM INFORMATION_SCHEMA.COLUMNS'
```

Cloud Spanner `SPANNER_SYS` introspection tables are also registered as built-in
catalog tables. They are useful for type-checking monitoring queries and
statistics helpers such as `SPANNER_SYS.DISTRIBUTION_PERCENTILE`.

```sh
go run ./cmd/spanner-analyzer \
  --sql 'SELECT
           INTERVAL_END,
           TABLE_NAME,
           READ_QUERY_COUNT
         FROM SPANNER_SYS.TABLE_OPERATIONS_STATS_MINUTE'
```

```sh
go run ./cmd/spanner-analyzer \
  --sql 'SELECT
           SPANNER_SYS.DISTRIBUTION_PERCENTILE(LATENCY_DISTRIBUTION[OFFSET(0)], 99.0) AS p99
         FROM SPANNER_SYS.QUERY_STATS_TOTAL_10MINUTE'
```

The Spanner lock statistics documentation uses a join between transaction and
lock statistics tables. The same shape can be analyzed without DDL:

```sh
go run ./cmd/spanner-analyzer \
  --sql 'SELECT
           t.INTERVAL_END,
           t.AVG_COMMIT_LATENCY_SECONDS,
           l.TOTAL_LOCK_WAIT_SECONDS
         FROM SPANNER_SYS.TXN_STATS_TOTAL_10MINUTE AS t
         LEFT JOIN SPANNER_SYS.LOCK_STATS_TOTAL_10MINUTE AS l
           ON t.INTERVAL_END = l.INTERVAL_END
         ORDER BY t.INTERVAL_END'
```

The CLI also exposes selected GoogleSQL analyzer options from
`execute_query_tool`, including `--product-mode`, `--strict-name-resolution`,
`--fold-literal-cast`, `--prune-unused-columns`, and
`--parse-location-record-type`. The default `--product-mode` is `external`,
matching Cloud Spanner's public GoogleSQL dialect. `--mode=spanner_type` emits
the Cloud Spanner type protobuf as YAML by default. YAML output is produced by
converting the `protojson` result with
[`github.com/goccy/go-yaml`](https://github.com/goccy/go-yaml). Use
`--output json` or `--output textproto` to emit another protobuf format.

```sh
go run ./cmd/spanner-analyzer \
  --sql-mode expression \
  --sql '1'
```

Output:

```yaml
code: INT64
```

JSON output is still available:

```sh
go run ./cmd/spanner-analyzer \
  --sql-mode expression \
  --sql '1' \
  --output json
```

Output:

```json
{
  "code": "INT64"
}
```

BigQuery mode is also available. It analyzes BigQuery GoogleSQL queries against
[BigQuery DDL](https://cloud.google.com/bigquery/docs/reference/standard-sql/data-definition-language)
and emits a BigQuery REST
[`TableSchema`](https://cloud.google.com/bigquery/docs/reference/rest/v2/tables#TableSchema)
shaped result. The default mode for `--dialect=bigquery` is
`--mode=bigquery_type`.

```sh
go run ./cmd/spanner-analyzer \
  --dialect bigquery \
  --sql 'SELECT 1 AS n, ["a", "b"] AS tags'
```

Output:

```yaml
fields:
- name: "n"
  type: INTEGER
  mode: NULLABLE
- name: tags
  type: STRING
  mode: REPEATED
```

BigQuery DDL is analyzed by the GoogleSQL frontend, so BigQuery table schemas
can use nested `STRUCT`, repeated `ARRAY`, `JSON`, `BIGNUMERIC`, and
`RANGE<DATE|DATETIME|TIMESTAMP>` types.

```sh
go run ./cmd/spanner-analyzer \
  --dialect bigquery \
  --ddl bigquery-schema.sql \
  --sql 'SELECT customer_id, profile, events FROM mydataset.customers'
```

For
[BigQuery-to-Spanner federated queries](https://cloud.google.com/bigquery/docs/spanner-federated-queries)
that use `EXTERNAL_QUERY`, pass BigQuery DDL and Spanner DDL for each BigQuery
connection ID. The analyzer rewrites literal `EXTERNAL_QUERY` calls to
equivalent empty BigQuery subqueries using the inner Spanner query's analyzed
row type.

```sh
go run ./cmd/spanner-analyzer \
  --dialect bigquery \
  --ddl bigquery-schema.sql \
  --external-ddl my-project.us.example-db=spanner-schema.sql \
  --sql "SELECT c.customer_id, rq.first_order_date
         FROM mydataset.customers AS c
         LEFT JOIN EXTERNAL_QUERY(
           'my-project.us.example-db',
           '''SELECT CustomerId AS customer_id, MIN(OrderDate) AS first_order_date
              FROM Orders
              GROUP BY CustomerId''') AS rq
           ON rq.customer_id = c.customer_id"
```

If the Spanner schema uses `PROTO BUNDLE`, provide descriptor sets for the same
connection. Proto fields are available while analyzing that connection's inner
Spanner SQL, but top-level `PROTO` values still cannot be returned through
BigQuery `EXTERNAL_QUERY`.

```sh
go run ./cmd/spanner-analyzer \
  --dialect bigquery \
  --external-ddl example-project.asia-northeast1.example-connection=testdata/order-proto-schema.sql \
  --external-proto-descriptors-file example-project.asia-northeast1.example-connection=testdata/protos/order_descriptors.pb \
  --sql "SELECT * FROM EXTERNAL_QUERY(
           'example-project.asia-northeast1.example-connection',
           '''SELECT OrderInfo.order_number, OrderInfo.shipping_address.city FROM Orders''')"
```

Output:

```yaml
fields:
- name: order_number
  type: STRING
  mode: NULLABLE
- name: city
  type: STRING
  mode: NULLABLE
```

`--mode` is inspired by GoogleSQL
[`execute_query`](https://github.com/google/googlesql/blob/master/execute_query.md)
modes. The default `--mode=spanner_type` returns the Cloud Spanner row type for
query mode, or a single Cloud Spanner type for expression mode. With
`--dialect=bigquery`, the default `--mode=bigquery_type` returns a BigQuery
`TableSchema` shaped schema. `--mode=parse` prints the parser AST,
`--mode=analyze` prints the resolved AST debug string like GoogleSQL
`execute_query` analyze mode, and `--mode=unparse` prints parser AST converted
back to SQL. Modes can be comma-separated, for example
`--mode=parse,analyze,spanner_type`.

GoogleSQL is initialized with
[wazero](https://github.com/tetratelabs/wazero) compiler mode and an on-disk
compilation cache. Set `SPANNER_ANALYZER_GOOGLESQL_CACHE_DIR` to override the
cache directory.

## Library components

The public API is intentionally split into composable steps:

- `BuildSchemaCatalog` parses Spanner DDL into this project's Spanner schema
  catalog.
- `BuildGoogleSQLCatalogFromSpannerCatalog` and `BuildGoogleSQLCatalogFromDDL`
  convert that schema into a GoogleSQL frontend catalog, analyzer options, and
  type factory.
- `BuildBigQueryGoogleSQLCatalogFromDDL` converts BigQuery DDL into a
  GoogleSQL frontend catalog.
- `GoogleSQLHelper` wraps parse, analyze, unparse, and resolved AST debug
  operations against that catalog.
- `RowTypeFromAnalyzerOutput`, `RowTypeFromResolvedQuery`, and
  `TypeFromAnalyzerOutput` convert GoogleSQL analyzer results into Cloud
  Spanner protobuf metadata.
- `BigQueryTableSchemaFromAnalyzerOutput`,
  `BigQueryTableSchemaFromResolvedQuery`, and
  `BigQueryTableFieldSchemaFromGoogleSQLType` convert GoogleSQL analyzer types
  into BigQuery REST `TableSchema` shaped metadata.
- `Analyzer` remains a convenience wrapper that wires these components together
  for the CLI-style row type use case.
- `BigQueryAnalyzer` does the same for BigQuery `TableSchema` output.
  Connection-specific Spanner analyzers can be attached with
  `SetExternalQueryAnalyzers` to infer `EXTERNAL_QUERY` result schemas.

## License

This project is licensed under the Apache License 2.0.

The source distribution does not vendor
[`github.com/goccy/go-googlesql`](https://github.com/goccy/go-googlesql) or its
embedded `googlesql.wasm` artifact. Binary distributions built from this
project do include that dependency transitively, so distributors should include
the relevant third-party license notices for at least:

- [`github.com/goccy/go-googlesql`](https://github.com/goccy/go-googlesql) and
  [`github.com/goccy/googlesql-wasm`](https://github.com/goccy/googlesql-wasm),
  which are MIT licensed.
- [`github.com/google/googlesql`](https://github.com/google/googlesql), the
  GoogleSQL frontend embedded in that WASM artifact, which is Apache-2.0
  licensed.

If future releases vendor dependencies or attach compiled binaries, add the
corresponding third-party license and NOTICE material to those release
artifacts.

## Limitations

- `PROTO BUNDLE` support requires descriptor set files. DDL alone is not enough
  to analyze proto fields.
- Cloud Spanner and the
  [Cloud Spanner emulator](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator)
  use the GoogleSQL frontend's native `MakeProtoType` and `MakeEnumType` APIs
  with descriptors from the active proto bundle. The current Go binding used by
  this project does not expose public equivalents, so proto values are
  represented internally as STRUCT shadows and enum values as INT64 shadows
  during query analysis.
- Direct top-level proto or enum column outputs are mapped back to Spanner row
  metadata when possible, but nested proto-derived expressions may reflect the
  internal shadow representation instead of native Spanner `PROTO` or `ENUM`
  types.
- Property graph DDL is parsed and graph names are registered in the GoogleSQL
  catalog, but GQL query analysis cannot yet expose node or edge properties.
  [`github.com/goccy/go-googlesql`](https://github.com/goccy/go-googlesql)
  v0.1.0 has graph catalog types, but does not expose public constructors or
  usable callbacks for building graph node and edge table metadata from Go.
- Some Spanner-specific functions are registered locally because they are not
  included in the default `go-googlesql` v0.1.0 builtin function set. This
  includes commit timestamp, sequence, search, TOKENLIST, and AI helper
  functions needed for query analysis.
- `ML.PREDICT` is not supported yet. It is a table-valued function whose output
  schema depends on the referenced model and input relation; `go-googlesql`
  v0.1.0 exposes table-valued function catalog types, but not a Go callback
  path for implementing Spanner's dynamic `Resolve` behavior.
- `TOKENLIST` is supported as an internal analysis type for search expressions,
  but Cloud Spanner result sets cannot return `TOKENLIST`, so the Cloud Spanner
  protobuf API has no `TypeCode` for it.
- Named arguments for locally registered Spanner functions are normalized before
  analysis because the current Go binding does not expose the GoogleSQL
  frontend's argument name setters.
- BigQuery mode currently registers ordinary BigQuery tables and views from DDL
  but does not model every DDL side effect. Dataset DDL, indexes, and drops are
  ignored because they do not change query result types in the current catalog
  model.
- BigQuery `bigquery_type` output is derived from resolved GoogleSQL types. It
  can represent repeated and nested fields, but query result nullability is not
  tracked, so non-repeated query output fields are emitted as `NULLABLE`.
- BigQuery-to-Spanner federated queries use the `EXTERNAL_QUERY` table-valued
  function. The current implementation supports `EXTERNAL_QUERY` only when the
  connection and Spanner SQL arguments are string literals and the connection
  has a matching `--external-ddl`. It rewrites the call before analysis to an
  equivalent empty BigQuery subquery with the same inferred columns. It does not
  evaluate connection options, permissions, PostgreSQL-dialect Spanner SQL, or
  non-literal dynamic SQL expressions.
