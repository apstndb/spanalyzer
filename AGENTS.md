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

The local directory name `spanner-analyzer` intentionally differs from the
repository name `apstndb/spanalyzer`; do not propose renaming the directory.
A v1 config freeze is deliberately deferred; `v1alpha` stays the mutable
preview channel.

Use "GoogleSQL frontend" for the analyzer/catalog library formerly named
ZetaSQL, and "Spanner GoogleSQL" for Cloud Spanner's SQL dialect. Mention
ZetaSQL only for historical upstream API, repository, or symbol names.

The CLI entry point is `cmd/spanner-analyzer/main.go`. Core catalog and analyzer
code lives in the root package. `spanner-query-gen`-specific config, planning,
and DTO rendering code lives in `internal/querygen`; keep generator-only
dependencies out of the root package.

The repository is a four-module Go workspace (see `go.work`): the root
analyzer module, `plancontract`, `cmd/spanner-query-gen`, and `tools`.
`go test ./...` covers only the current module; run tests per module
directory. Dependency-weight rules to preserve:

- `plancontract` normalizes raw `spannerpb.QueryPlan` values and evaluates
  plan contracts. Never add dependencies on go-googlesql, memefish,
  spanemuboost, or container tooling there.
- The root module must not depend on spanemuboost, testcontainers, the
  Docker client, or spannerplan; those belong in `cmd/spanner-query-gen`
  and `tools`.

## Essential Commands

Use the Go toolchain declared in `go.mod`.

```sh
go test ./...
(cd plancontract && go test ./...)
(cd cmd/spanner-query-gen && go test ./...)
(cd tools && go test ./...)
go build ./...
go run ./cmd/spanner-analyzer --ddl testdata/order-proto-schema.sql \
  --proto-descriptors-file testdata/protos/order_descriptors.pb \
  --sql 'SELECT OrderInfo.order_number FROM Orders'
```

Run `gofmt` on edited Go files. Run the per-module tests above (at least the
modules you touched) before reporting a change as complete.

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

## Documentation Guidelines

The [`knowledge/`](knowledge/index.md) directory is the repository's embedded
OKF v0.2 bundle. Keep canonical command, design, and status documents at their
repository-native paths; represent them in OKF through concepts and links
instead of copying their bodies. Author new substantive research directly
under [`knowledge/research/<area>/`](knowledge/research/index.md) as `Research Note`
concepts. The existing [`research/`](research/README.md) tree is a frozen
legacy store: maintain or migrate its current notes, but do not add new notes
there. When adding or removing tracked Markdown outside the bundle, update
[`knowledge/references/repository-documents.md`](knowledge/references/repository-documents.md).
Every non-`index.md` concept inside the bundle must have parseable YAML
frontmatter with a non-empty `type`, and every concept must be reachable from
the root `knowledge/index.md` through directory indexes.

Repository policy requires every `Research Note` to declare a non-empty
`title`, `description`, `tags`, explicit `status`, and at least one `sources`
entry. Use `generated`, `verified`, credibility signals, and `stale_after` only
when the corresponding producer, verification event, source fact, or expiry is
actually known. Use `git mv` when one coherent legacy body remains one
`Research Note`. Split, merge, absorption, executable replacement, and
retirement instead require a completed, source-hashed entry in
[`legacy-research-migrations.md`](knowledge/references/legacy-research-migrations.md).
In the same change, remove the path from the active legacy allowlist, update
every inbound link, and disposition any hashed `planvocab` evidence. Never
leave a duplicate writable body.

Non-Markdown discovery is intentionally narrower than repository inventory.
[`knowledge/references/repository-assets.md`](knowledge/references/repository-assets.md)
classifies contract schemas, plan-vocabulary artifacts, executable plan
evidence, and reusable fixtures. Do not extend it into a census of general Go
sources, module files, CI configuration, IDE metadata, or other operational
files. Inventory membership is discovery metadata, not evidence that an asset
supports a claim; `planvocab` continues to derive its authoritative hashed
inputs from `catalog_source.info.local_evidence`.

Run the OKF checks from the `tools` module after changing the bundle or either
inventory:

```sh
go run ./okf-check --repo-root .. --gate all
```

## Coding Style

Follow idiomatic Go and `gofmt`. Keep helpers small and explicit. Avoid broad
refactors unless they directly support the DDL or query-semantics behavior being
implemented. Saved files, comments, commit messages, and documentation in this
repository should be written in English.
