# spanner-analyzer command guide

`spanner-analyzer` analyzes SQL locally against a catalog derived from DDL. It
returns result types or generated Go structs, and exposes parser/analyzer debug
modes. The separate `mutation_count` mode estimates write-impact intervals.
These modes do not execute database queries or mutations.

Run all commands below from the repository root. For reusable Go APIs and
analysis boundaries, use the [package documentation](https://pkg.go.dev/github.com/apstndb/spanalyzer).
For query generation and opt-in runtime collection, see
[`spanner-query-gen`](../spanner-query-gen/README.md). Build and verification
tasks live in the [development guide](../../DEVELOPMENT.md).

- [Queries and proto bundles](#queries-and-proto-bundles)
- [Parameters and result schemas](#parameters-and-result-schemas)
- [Expressions](#expressions)
- [Built-in metadata catalogs](#built-in-metadata-catalogs)
- [Analyzer controls and output formats](#analyzer-controls-and-output-formats)
- [Go result structs](#go-result-structs)
- [BigQuery and EXTERNAL_QUERY](#bigquery-and-external_query)
- [Modes and language features](#modes-and-language-features)
- [Mutation counts](#mutation-counts)
- [Companion tools and fixtures](#companion-tools-and-fixtures)

## Queries and proto bundles

```sh
go run ./cmd/spanner-analyzer \
  --ddl testdata/order-proto-schema.sql \
  --proto-descriptors-file testdata/protos/order_descriptors.pb \
  --sql 'SELECT OrderInfo.order_number FROM Orders'
```

Output excerpt:

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

## Parameters and result schemas

Named query parameters can be declared with `--param name=TYPE`. Positional
parameters can be declared with repeatable `--positional-param TYPE`.

```sh
go run ./cmd/spanner-analyzer \
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

## Expressions

`--sql-mode expression` analyzes a single GoogleSQL expression and returns a
single Spanner `Type` instead of a query result row type.

```sh
go run ./cmd/spanner-analyzer \
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
  --sql-mode expression \
  --sql 'ARRAY_FIRST([1, 2, 3])'
```

Output:

```yaml
code: INT64
```

```sh
go run ./cmd/spanner-analyzer \
  --sql-mode expression \
  --sql 'ARRAY_FIRST(["a", "b"])'
```

Output:

```yaml
code: STRING
```

## Built-in metadata catalogs

Cloud Spanner `INFORMATION_SCHEMA` tables are registered as built-in catalog
tables for analysis. They provide names and column types only; no row data is
materialized. The built-in surface comes from the generated
[`information_schema_manifest.json`](../../information_schema_manifest.json), whose
[`projection source`](../../information_schema_projection_source.json) selects one
exact point-in-time managed observation. Observed and rolling columns are
projected, while documented columns absent from that observation remain
evidence metadata only. The observation is scoped to one database and does not
claim that managed Spanner has a fleet-wide or durable metadata version.

Validate the selected capture, producer registry, projection source, and exact
generated manifest bytes:

```sh
(cd tools && go run ./infoschema-survey-check)
```

```sh
go run ./cmd/spanner-analyzer \
  --sql 'SELECT TABLE_NAME, COLUMN_NAME, ORDINAL_POSITION, SPANNER_TYPE
         FROM INFORMATION_SCHEMA.COLUMNS'
```

Cloud Spanner `SPANNER_SYS` introspection tables are also registered as built-in
catalog tables. The built-in surface comes from the pinned, live-primary
[`spanner_sys_manifest.json`](../../spanner_sys_manifest.json): only columns observed
with identical type and ordinal on both the recorded managed and Omni targets
are projected, while known-absent entries and documentation conflicts remain
evidence metadata. This is a pinned common surface, not a guarantee about every
deployment or future rollout. The tables are useful for type-checking monitoring
queries and statistics helpers such as `SPANNER_SYS.DISTRIBUTION_PERCENTILE`.

Validate the manifest and reproduce its exact bytes from the retained
in-repository survey exporter:

```sh
(cd tools && go run ./spannersys-survey-check)
```

The immutable mapping from the unpublished legacy source snapshot to the
initial imported subtree is checked separately:

```sh
(cd tools && go run ./survey-import-check)
```

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

## Analyzer controls and output formats

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

## Go result structs

`--mode=go_struct` emits Go code for a struct that can receive query result
rows. Use `--go-client=bigquery`, `--go-client=spanner`, or
`--go-client=both` to choose struct tags and field types. The default is
`both`, which emits both `bigquery` and `spanner` tags. In `both` mode the
generator keeps one field per result column and emits a small `NullValue[T]`
helper so the same DTO can be loaded from BigQuery with `bigquery.ValueLoader`
and from Spanner with `spanner.Decoder`.

```sh
go run ./cmd/spanner-analyzer \
  --mode go_struct \
  --sql 'SELECT 1 AS n'
```

Output excerpt:

```go
package main

import (
	"cloud.google.com/go/bigquery"
	"fmt"
)

type QueryRow struct {
	N NullValue[int64] `bigquery:"n" spanner:"n"`
}

func (r *QueryRow) Load(values []bigquery.Value, schema bigquery.Schema) error {
	if len(values) != len(schema) {
		return fmt.Errorf("bigquery row has %d values for %d schema fields", len(values), len(schema))
	}
	for i, field := range schema {
		switch field.Name {
		case "n":
			if err := r.N.LoadBigQuery(values[i]); err != nil {
				return fmt.Errorf("n: %w", err)
			}
		}
	}
	return nil
}

type NullValue[T any] struct {
	Value T
	Valid bool
}

func (n NullValue[T]) IsNull() bool {
	return !n.Valid
}

func (n *NullValue[T]) LoadBigQuery(value bigquery.Value) error {
	return n.set(value)
}

func (n *NullValue[T]) DecodeSpanner(input interface{}) error {
	return n.set(input)
}

func (n *NullValue[T]) set(value interface{}) error {
	if value == nil {
		var zero T
		n.Value = zero
		n.Valid = false
		return nil
	}
	typed, ok := value.(T)
	if !ok {
		return fmt.Errorf("cannot decode %T", value)
	}
	n.Value = typed
	n.Valid = true
	return nil
}
```

## BigQuery and EXTERNAL_QUERY

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

BigQuery and Spanner use different GoogleSQL feature presets. The current
analyzer accepts pipe query syntax in both modes. This example selects the
BigQuery preset and emits BigQuery schema metadata; the Spanner preset emits
Spanner protobuf types. Analyzer acceptance does not establish support on a
particular Spanner runtime.

```sh
go run ./cmd/spanner-analyzer \
  --dialect bigquery \
  --sql 'FROM UNNEST([STRUCT("apples" AS item, 2 AS sales, "fruit" AS category),
                      STRUCT("carrots", 8, "vegetable"),
                      STRUCT("apples", 7, "fruit")]) AS produce
         |> WHERE category = "fruit"
         |> AGGREGATE COUNT(*) AS num_items, SUM(sales) AS total_sales
            GROUP BY item
         |> ORDER BY item'
```

Output:

```yaml
fields:
- name: item
  type: STRING
  mode: NULLABLE
- name: num_items
  type: INTEGER
  mode: NULLABLE
- name: total_sales
  type: INTEGER
  mode: NULLABLE
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
connection ID. The analyzer provides a `EXTERNAL_QUERY` table-valued function
that returns the row type inferred from the inner Spanner query using the
connection-specific Spanner catalog.

Only the two-argument form `EXTERNAL_QUERY(connection, sql)` is analyzed. The
optional third options argument is a hard error
(`EXTERNAL_QUERY options argument is currently not supported for static
analysis`), because the analyzer cannot evaluate connection options
statically; remove it from the SQL under analysis.

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

## Modes and language features

`--mode` is inspired by GoogleSQL
[`execute_query`](https://github.com/google/googlesql/blob/master/execute_query.md)
modes. The default `--mode=spanner_type` returns the Cloud Spanner row type for
query mode, or a single Cloud Spanner type for expression mode. With
`--dialect=bigquery`, the default `--mode=bigquery_type` returns a BigQuery
`TableSchema` shaped schema. `--mode=parse` prints the parser AST,
`--mode=analyze` prints the resolved AST debug string like GoogleSQL
`execute_query` analyze mode, `--mode=unparse` prints parser AST converted
back to SQL, and `--mode=go_struct` prints Go result struct code. Modes can be
comma-separated, for example `--mode=parse,analyze,spanner_type`.

GoogleSQL is initialized once per process through `go-googlesql`.

Dialect feature presets start from GoogleSQL
`EnableMaximumLanguageFeaturesForDevelopment()` and then disable features that
are not available in the selected dialect. Use
`--enable-maximum-development-language-features` to skip that blacklist and try
raw development features, for example when validating a newly released Spanner
feature before this project has updated its preset.

## Mutation counts

This preview estimates mutation units from DDL and explicitly supplied facts;
it does not infer runtime cardinality or perform database reads/writes:

```sh
go run ./cmd/spanner-analyzer --mode mutation_count \
  --ddl testdata/mutation-count/schema.sql \
  --mutations-file testdata/mutation-count/update.json --output json
```

Add `--mutation-facts-file testdata/mutation-count/old-row.json` to refine the
schema-only `[3,6]` interval to 3 under that old-row assertion. Unknown fields in
refinement files fail explicitly. For ordered DML and buffered mutations,
use `--mutation-transaction-file testdata/mutation-count/transaction.json`
with `--ddl testdata/mutation-count/transaction.sql` instead of `--mutations-file`.

The [feature map](../../knowledge/concepts/mutation-count-analysis.md) links
bounds, old-row/index refinements, same-execution observations, queue limits,
and the query-generator's explicit measurement workflow. See the
[transaction API](https://pkg.go.dev/github.com/apstndb/spanalyzer#MutationCountAnalyzer.AnalyzeTransaction)
for ordering and separate quota boundaries. An observed execution is not a
universal price; mutation units do not establish byte limits or commit success.

## Companion tools and fixtures

Registered GoogleSQL frontend and Spanner function signatures can be dumped
with the dedicated function catalog command:

```sh
go run ./cmd/spanner-function-catalog
```

Use `--verbose=false` to print only function names. `--ddl` and
`--proto-descriptors-file` are also accepted when the catalog depends on schema
objects or proto descriptors.

Developer-only probes live under `tools/`. For example,
[`tools/spanner-query-plan-shape`](../../tools/spanner-query-plan-shape) starts
Spanner Omni through [`spanemuboost`](https://github.com/apstndb/spanemuboost)
and prints raw query plan node shapes for plan normalization work.
Existing legacy investigation notes and review archives live under
[`research/`](../../research/); new substantive research is authored under
[`knowledge/research/`](../../knowledge/research/index.md). Both are non-normative
supporting material rather than the public CLI contract.

The repository includes the Protocol Buffers example from the
[Cloud Spanner protocol buffers reference](https://cloud.google.com/spanner/docs/reference/standard-sql/protocol-buffers)
under `testdata/protos/`, including
`order_protos.proto` and its compiled `order_descriptors.pb` descriptor set.
