# spanalyzer

[![Go Reference](https://pkg.go.dev/badge/github.com/apstndb/spanalyzer.svg)](https://pkg.go.dev/github.com/apstndb/spanalyzer)

`spanalyzer` is an experimental Go library and CLI toolkit for Cloud Spanner
GoogleSQL: it infers query result types from DDL, generates Go query code,
inspects query plans, and estimates mutation-count intervals. BigQuery analysis
is also available. APIs remain experimental. Query-generator configuration uses a mutable
`v1alpha` schema with no stable `v1` config.

Previously named `go-googlesql-spanner-poc`, the project uses
[memefish](https://github.com/cloudspannerecosystem/memefish) and the
[GoogleSQL frontend Go binding](https://github.com/goccy/go-googlesql) as
implementation details; experimental low-level APIs may expose their types
directly.

## Usage

Run commands from the repository root. To infer result field types from Spanner
DDL and compiled Protocol Buffer descriptors:

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

See the [analyzer command guide](cmd/spanner-analyzer/README.md) for query
parameters, standalone expressions, output formats, metadata catalogs, and
BigQuery federation.

You can also estimate write mutation counts statically (preview):

```sh
go run ./cmd/spanner-analyzer --mode mutation_count \
  --ddl testdata/mutation-count/schema.sql \
  --mutations-file testdata/mutation-count/update.json --output json
```

From the schema alone, this command reports `[3,6]`. Passing
`--mutation-facts-file testdata/mutation-count/old-row.json` narrows the
estimate to `3` only under those supplied old-row assertions, not as a universal
price for the write. Intervals count Spanner mutation units rather than commit
bytes, latency, or commit success. See the
[mutation-count feature map](knowledge/concepts/mutation-count-analysis.md) for
library, CLI, transaction, queue, and measurement workflows.

## Library components

To embed analysis in Go programs, see the
[package documentation](https://pkg.go.dev/github.com/apstndb/spanalyzer) for
convenience analyzers, composable catalogs, result converters, and the
mutation-count preview API. The Go `Example` tests provide runnable API examples.

## Documentation map

| Goal | Start here |
| --- | --- |
| Call the Go library and browse runnable examples | [Package documentation](https://pkg.go.dev/github.com/apstndb/spanalyzer) |
| Analyze SQL types or mutation counts from the CLI | [Analyzer command guide](cmd/spanner-analyzer/README.md) |
| Generate Go query code or collect plan reports | [spanner-query-gen guide](cmd/spanner-query-gen/README.md) |
| Author or evaluate structural plan contracts | [Plan contracts](cmd/spanner-query-gen/PLAN_CONTRACTS.md) and [planvocab](plancontract/planvocab/README.md) |
| Build, test, or work with pinned runtimes | [Development guide](DEVELOPMENT.md) |
| Read architecture notes, observations, and research | [Knowledge bundle](knowledge/index.md) |
| Track engine behavior across versions | [Runtime verification](knowledge/research/runtime-verification/index.md) |
| Check open work or repository agent guidelines | [TODO](TODO.md) and [AGENTS](AGENTS.md) |

Design documents describe intended architecture, whereas implementation-status
documents record current gaps; see the
[documentation authority guide](knowledge/concepts/documentation-authority.md)
for how they relate. Active research lives under `knowledge/research/`, while
the [legacy research tree](research/README.md) preserves earlier material.

## Modules

A checked-in `go.work` workspace links five Go modules so heavy runtime and
container dependencies stay isolated from core packages:

| Module | Responsibility and dependency boundary |
| --- | --- |
| Root `github.com/apstndb/spanalyzer` | Catalogs, SQL and mutation analysis, type conversion, lightweight CLIs, and codegen planning; owns no container dependencies |
| [`plancontract`](plancontract/) | Spanner plan normalization and structural contract evaluation without the GoogleSQL frontend or container dependencies |
| [`cmd/spanner-query-gen`](cmd/spanner-query-gen/) | Query-generation CLI and Omni-backed plan collection; owns container dependencies |
| [`survey`](survey/) | Schema discovery, DDL reconstruction, and cross-engine evidence producers; independently testable with `GOWORK=off` |
| [`tools`](tools/) | Developer probes and repository checks; may depend on container tooling |

## Why plan inspection

Plan tooling captures, normalizes, and checks Spanner execution plans against
explicit structural contracts so plan reviews are repeatable across changes.
Structural `PLAN` topology does not prove query latency, CPU consumption, or
production workload performance; see
[query-plan inspection](knowledge/concepts/query-plan-inspection.md) for the
workflow and its evidence boundaries.

## Limitations

The analyzer derives result types and structural metadata from DDL and catalogs
rather than executing database queries; opt-in measurement in the query
generator is a separate workflow. Protocol Buffer bundles require compiled
descriptor sets, Spanner Graph metadata support is partial, and dialect-specific
features have explicit support boundaries documented in the
[package documentation](https://pkg.go.dev/github.com/apstndb/spanalyzer) and
[analyzer command guide](cmd/spanner-analyzer/README.md).

## Repository verification

The [development guide](DEVELOPMENT.md) covers validation across all five
modules, the pre-push gate, container prerequisites, and opt-in managed
verification. [`runtime_targets.json`](runtime_targets.json) records the
checked-in runtime selections used by those checks; a pinned version records a
verification target, not a claim that it is the latest public release.

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
