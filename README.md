# go-googlesql-spanner-poc

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
  --proto-descriptors-file /path/to/descriptors.pb \
  --sql 'SELECT OrderInfo.order_number FROM Orders'
```

`--proto-descriptors-file` accepts a Protocol Buffers `FileDescriptorSet` used
to resolve types named by `CREATE PROTO BUNDLE` or `ALTER PROTO BUNDLE`. The
flag is repeatable.

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

`--sql-mode expression` analyzes a single GoogleSQL expression and returns a
single Spanner `Type` instead of a query result row type.

```sh
go run ./cmd/spanner-analyzer \
  --ddl schema.sql \
  --sql-mode expression \
  --sql 'AI.SCORE(@prompt)' \
  --param 'prompt=STRING(MAX)'
```

The CLI also exposes selected GoogleSQL analyzer options from
`execute_query_tool`, including `--product-mode`, `--strict-name-resolution`,
`--fold-literal-cast`, `--prune-unused-columns`, and
`--parse-location-record-type`. Use `--output textproto` to emit protobuf text
format instead of JSON.

`--mode` is inspired by GoogleSQL `execute_query` modes. The default
`--mode=analyze` returns the Spanner row type. `--mode=parse` prints the parser
AST, `--mode=unparse` prints parser AST converted back to SQL, and
`--mode=resolved_ast` prints the resolved AST debug string. Modes can be
comma-separated, for example `--mode=parse,resolved_ast`.

GoogleSQL is initialized with wazero compiler mode and an on-disk compilation
cache. Set `SPANNER_ANALYZER_GOOGLESQL_CACHE_DIR` to override the cache
directory.

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
