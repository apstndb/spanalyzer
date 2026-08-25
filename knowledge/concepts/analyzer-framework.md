---
type: Architecture
title: spanalyzer Analyzer Framework
description: The five-module architecture and dependency boundaries of spanalyzer.
resource: ../../README.md
tags: [spanalyzer, architecture, go, spanner]
status: draft
sources:
  - id: repository-readme
    resource: ../../README.md
    title: spanalyzer README
  - id: agent-guide
    resource: ../../AGENTS.md
    title: spanalyzer repository guidance
  - id: go-workspace
    resource: ../../go.work
    title: spanalyzer Go workspace definition
---

# spanalyzer Analyzer Framework

spanalyzer separates query analysis, plan normalization, query generation,
cross-environment schema surveys, and developer probes into five Go modules so
container and generator dependencies do not leak into the reusable analyzer or
plan-contract libraries.[^go-workspace]

## Module boundaries

- The [root module](../../README.md) owns DDL catalogs, the GoogleSQL frontend,
  result-type conversion, and lightweight analysis commands.
- [`plancontract`](../../plancontract/) normalizes raw Spanner query plans and
  evaluates structural contracts without depending on the GoogleSQL frontend
  or container tooling.
- [`cmd/spanner-query-gen`](../../cmd/spanner-query-gen/README.md) owns query
  generation, plan-report orchestration, and optional Spanner Omni integration.
- [`survey`](../../survey/) retains schema discovery, DDL reconstruction, and
  managed, Omni, and Emulator evidence producers. It remains independently
  testable with `GOWORK=off`.
- [`tools`](../../tools/spanner-query-plan-shape/README.md) contains
  developer-only probes and may depend on container tooling.

The exact public and experimental surfaces remain documented in the
[repository README](../../README.md). Dependency-weight rules and validation
commands are maintained in [AGENTS.md](../../AGENTS.md); this concept does not
duplicate them.

## Related concepts

- [Query generation](query-generation.md)
- [Query-plan inspection](query-plan-inspection.md)
- [Documentation authority](documentation-authority.md)
- [Cross-environment evidence lifecycle](evidence-lifecycle.md)

[^go-workspace]: The committed Go workspace defines the five local modules.
