# Repository Guidelines

This file provides shared guidance for coding agents working in this repository.
Keep it short and update it only with information that would cause real mistakes
if omitted.

## Project Overview

`spanalyzer` (formerly `go-googlesql-spanner-poc`) is a Spanner analyzer
framework: an experimental Go library and CLI toolkit for deriving Cloud
Spanner GoogleSQL query result row types from Spanner DDL. It parses DDL
with `github.com/cloudspannerecosystem/memefish` and analyzes queries with
`github.com/goccy/go-googlesql`; treat both as implementation details rather
than part of the public contract.

Use "GoogleSQL frontend" for the analyzer/catalog library formerly named
ZetaSQL, and "Spanner GoogleSQL" for Cloud Spanner's SQL dialect. Mention
ZetaSQL only for historical upstream API, repository, or symbol names.

The CLI entry point is `cmd/spanner-analyzer/main.go`. Core catalog and analyzer
code lives in the root package. `spanner-query-gen`-specific config, planning,
and DTO rendering code lives in `internal/querygen`; keep generator-only
dependencies out of the root package.

## Essential Commands

Use the Go toolchain declared in `go.mod`.

```sh
go test ./...
go build ./...
go run ./cmd/spanner-analyzer --ddl testdata/order-proto-schema.sql \
  --proto-descriptors-file testdata/protos/order_descriptors.pb \
  --sql 'SELECT OrderInfo.order_number FROM Orders'
```

Run `gofmt` on edited Go files. Run `go test ./...` before reporting a change as
complete.

## Implementation Notes

- `Catalog` is the source of truth for parsed schema objects. Add DDL support
  there before wiring objects into the GoogleSQL catalog.
- `GoogleSQLCatalog` registers tables, views, property graphs, functions,
  models, and type information into `go-googlesql` objects.
- `GoogleSQLHelper` owns parse/analyze/unparse/resolved-AST helper calls against
  a GoogleSQL catalog.
- Result conversion from GoogleSQL analyzer output to Spanner protobuf metadata
  lives in `resultconv.go`; keep it separate from catalog construction.
- Query generation and Go DTO rendering for `cmd/spanner-query-gen` live in
  `internal/querygen`; root package additions should stay focused on reusable
  analyzer/catalog/type-conversion APIs.
- Regular indexes, vector indexes, and search indexes are intentionally
  ignored for row-type analysis because they do not affect logical query
  result row types. Regular index metadata is still retained for query code
  generation (`kind: index`).
- Property graph support registers graph node and edge tables, labels, and
  direct column property definitions in GoogleSQL. More advanced graph
  metadata, including arbitrary property expressions and dynamic
  labels/properties, remains limited.
- Proto bundle support follows Spanner's input shape: DDL names active proto
  bundle types, while descriptor set files provide Protocol Buffers metadata.
  The supplied descriptor set is loaded into the GoogleSQL frontend descriptor
  pool so active proto and enum types can be registered as native analyzer
  types and converted back to Spanner row metadata.

## Testing Guidelines

Keep tests close to the code under test:

- DDL catalog behavior: `catalog_test.go`
- GoogleSQL analyzer behavior: `analyzer_googlesql_test.go`
- Query generator behavior: `internal/querygen/*_test.go`
- Query generator CLI and integration behavior: `cmd/spanner-query-gen/*_test.go`

Prefer focused regression tests using small inline DDL. For proto tests, use
the descriptor fixture at:

```text
testdata/protos/order_descriptors.pb
```

## Coding Style

Follow idiomatic Go and `gofmt`. Keep helpers small and explicit. Avoid broad
refactors unless they directly support the DDL or query-semantics behavior being
implemented. Saved files, comments, commit messages, and documentation in this
repository should be written in English.
