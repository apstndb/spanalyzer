# go-googlesql-spanner-poc

[![Go Reference](https://pkg.go.dev/badge/github.com/apstndb/go-googlesql-spanner-poc.svg)](https://pkg.go.dev/github.com/apstndb/go-googlesql-spanner-poc)

`go-googlesql-spanner-poc` is an experimental Go library and CLI for deriving
Cloud Spanner GoogleSQL query result row types from Spanner DDL.

It parses Spanner DDL with `github.com/cloudspannerecosystem/memefish` and
analyzes queries with `github.com/goccy/go-googlesql`.

In this document, "GoogleSQL frontend" refers to the analyzer and catalog
library formerly named ZetaSQL. "Spanner GoogleSQL" refers to Cloud Spanner's
SQL dialect. Historical ZetaSQL names appear only when referring to upstream
API names or repositories that still use them.

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

The repository includes the Protocol Buffers example from the Cloud Spanner
protocol buffers reference under `testdata/protos/`, including
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
converting the `protojson` result with `github.com/goccy/go-yaml`. Use
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

`--mode` is inspired by GoogleSQL `execute_query` modes. The default
`--mode=spanner_type` returns the Cloud Spanner row type for query mode, or a
single Cloud Spanner type for expression mode. `--mode=parse` prints the parser
AST, `--mode=analyze` prints the resolved AST debug string like GoogleSQL
`execute_query` analyze mode, and `--mode=unparse` prints parser AST converted
back to SQL. Modes can be comma-separated, for example
`--mode=parse,analyze,spanner_type`.

GoogleSQL is initialized with wazero compiler mode and an on-disk compilation
cache. Set `SPANNER_ANALYZER_GOOGLESQL_CACHE_DIR` to override the cache
directory.

## Library components

The public API is intentionally split into composable steps:

- `BuildSchemaCatalog` parses Spanner DDL into this project's Spanner schema
  catalog.
- `BuildGoogleSQLCatalogFromSpannerCatalog` and `BuildGoogleSQLCatalogFromDDL`
  convert that schema into a GoogleSQL frontend catalog, analyzer options, and
  type factory.
- `GoogleSQLHelper` wraps parse, analyze, unparse, and resolved AST debug
  operations against that catalog.
- `RowTypeFromAnalyzerOutput`, `RowTypeFromResolvedQuery`, and
  `TypeFromAnalyzerOutput` convert GoogleSQL analyzer results into Cloud
  Spanner protobuf metadata.
- `Analyzer` remains a convenience wrapper that wires these components together
  for the CLI-style row type use case.

## License

This project is licensed under the Apache License 2.0.

The source distribution does not vendor `github.com/goccy/go-googlesql` or its
embedded `googlesql.wasm` artifact. Binary distributions built from this
project do include that dependency transitively, so distributors should include
the relevant third-party license notices for at least:

- `github.com/goccy/go-googlesql` and `github.com/goccy/googlesql-wasm`, which
  are MIT licensed.
- `github.com/google/googlesql`, the GoogleSQL frontend embedded in that WASM
  artifact, which is Apache-2.0 licensed.

If future releases vendor dependencies or attach compiled binaries, add the
corresponding third-party license and NOTICE material to those release
artifacts.

## Limitations

- `PROTO BUNDLE` support requires descriptor set files. DDL alone is not enough
  to analyze proto fields.
- Cloud Spanner and the Cloud Spanner emulator use the GoogleSQL frontend's
  native `MakeProtoType` and `MakeEnumType` APIs with descriptors from the
  active proto bundle. The current Go binding used by this project does not
  expose public equivalents, so proto values are represented internally as
  STRUCT shadows and enum values as INT64 shadows during query analysis.
- Direct top-level proto or enum column outputs are mapped back to Spanner row
  metadata when possible, but nested proto-derived expressions may reflect the
  internal shadow representation instead of native Spanner `PROTO` or `ENUM`
  types.
- Property graph DDL is parsed and graph names are registered in the GoogleSQL
  catalog, but GQL query analysis cannot yet expose node or edge properties.
  `github.com/goccy/go-googlesql` v0.1.0 has graph catalog types, but does not
  expose public constructors or usable callbacks for building graph node and
  edge table metadata from Go.
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
