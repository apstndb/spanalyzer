---
type: Architecture
title: Query Generation
description: The spanner-query-gen product boundary and its canonical documentation surfaces.
resource: ../../cmd/spanner-query-gen/README.md
tags: [spanner, bigquery, query-generation, plan-contracts]
status: draft
sources:
  - id: querygen-readme
    resource: ../../cmd/spanner-query-gen/README.md
    title: spanner-query-gen README
  - id: querygen-design
    resource: ../../cmd/spanner-query-gen/DESIGN.md
    title: spanner-query-gen design
  - id: querygen-status
    resource: ../../cmd/spanner-query-gen/IMPLEMENTATION_STATUS.md
    title: spanner-query-gen implementation status
  - id: plan-contracts
    resource: ../../cmd/spanner-query-gen/PLAN_CONTRACTS.md
    title: spanner-query-gen plan contracts
  - id: config-schema
    resource: ../../schemas/spanner-query-gen.v1alpha.schema.json
    title: spanner-query-gen v1alpha configuration schema
  - id: plan-report-schema
    resource: ../../schemas/spanner-query-gen.plan-report.v1alpha.schema.json
    title: spanner-query-gen v1alpha plan-report schema
  - id: plan-contract-schema
    resource: ../../schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json
    title: spanner-query-gen v1alpha plan-contract schema
---

# Query Generation

`spanner-query-gen` generates reviewable Go DTOs and helpers from declared
Spanner and BigQuery schemas and queries. It keeps SQL and schema as the source
of truth and deliberately avoids becoming a runtime ORM or general query
builder.

Use the canonical documents according to the question being asked:

- [README](../../cmd/spanner-query-gen/README.md) for current command UX and
  supported configuration.
- [DESIGN](../../cmd/spanner-query-gen/DESIGN.md) for intended architecture,
  including future work.
- [IMPLEMENTATION_STATUS](../../cmd/spanner-query-gen/IMPLEMENTATION_STATUS.md)
  for explicit drift between the mutable `v1alpha` implementation and design.
- [PLAN_CONTRACTS](../../cmd/spanner-query-gen/PLAN_CONTRACTS.md) for the
  external plan-contract format and evaluation semantics.
- [Query-generator research](../../research/spanner-query-gen/README.md) for
  non-normative design background and verification sessions.

The checked-in configuration, plan-report, and plan-contract JSON Schemas are
the machine-readable contracts for the mutable `v1alpha` surfaces described by
those documents.[^config-schema][^plan-report-schema][^plan-contract-schema]

See [Documentation authority](documentation-authority.md) before treating a
design or research claim as current behavior.

[^config-schema]: `spanner-query-gen` v1alpha configuration schema.
[^plan-report-schema]: `spanner-query-gen` v1alpha plan-report schema.
[^plan-contract-schema]: `spanner-query-gen` v1alpha plan-contract schema.
