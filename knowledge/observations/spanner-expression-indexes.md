---
type: Observation
title: Spanner Scalar-Expression Indexes
description: Cross-target syntax and metadata boundaries, fail-closed reconstruction, and observed query-plan shapes for scalar-expression indexes.
tags: [spanner, spanner-omni, emulator, expression-index, information-schema, query-plan]
status: draft
sources:
  - id: generated-column-guide
    resource: https://docs.cloud.google.com/spanner/docs/generated-column/how-to
    title: Create an index on an expression
  - id: documentation-change
    resource: https://github.com/apstndb/spanner-docs-mirror/commit/7673834e6449a2ec3f760b9121cb431aa580df2b
    title: Spanner documentation change adding expression-index guidance
  - id: runtime-targets
    resource: ../../runtime_targets.json
    title: Pinned Emulator and Spanner Omni runtime identities
  - id: survey-probe
    resource: ../../survey/infoschem/expression_index_integration_test.go
    title: Cross-target expression-index syntax and metadata probe
  - id: reconstruction-boundary
    resource: ../../survey/astconv/expression_index_test.go
    title: Fail-closed expression-index reconstruction test
  - id: plan-probes
    resource: ../../tools/spanner-query-plan-shape/expression_index_cases.go
    title: Expression-index query-plan cases
  - id: plan-expectations
    resource: ../../tools/spanner-query-plan-shape/testdata/expression_index_expectations.json
    title: Expression-index planvocab expectations
---

# Spanner Scalar-Expression Indexes

Observed: 2026-08-28

Status: **documented syntax with environment-scoped runtime observations**.
The Spanner generated-column guide now documents secondary indexes whose key
is a scalar expression. No release-note entry was used to infer a launch stage,
and the observations below are not evidence of fleet-wide availability or a
stable optimizer contract.

## Documented surface

The documented GoogleSQL form adds parentheses around an expression used in an
index key:

```sql
CREATE INDEX VenuesByCity
ON Venues((JSON_VALUE(VenueData.address.city)));
```

An expression key can be combined with ordinary column keys. The service still
applies its determinism rules to the expression; `CURRENT_TIMESTAMP()` was
rejected as a non-deterministic index expression in the retained probes.

## Runtime boundaries

The managed database observed on 2026-08-28 and the exact Spanner Omni image
pinned in [`runtime_targets.json`](../../runtime_targets.json) accepted the
documented form and a mixed ordinary/expression key. They also accepted a form
without the extra expression wrapper, but that spelling is an observation, not
a documented contract.

The pinned Cloud Spanner Emulator rejected the documented form with a syntax
error. Managed Spanner and pinned Omni rejected `DESC` after a GoogleSQL
expression key with a syntax error, although the generic index-key grammar
shows an ordering suffix. The probes retain these as current boundaries rather
than normalizing them into supported syntax.

## INFORMATION_SCHEMA and reconstruction

For an expression key, `INFORMATION_SCHEMA.INDEX_COLUMNS` exposes both:

- the scalar text in `EXPRESSION`; and
- an internal `_ExpressionIndex_<index>_<position>` value in `COLUMN_NAME`.

The internal key name is not exposed as a row in `INFORMATION_SCHEMA.COLUMNS`.
It is an implementation identifier, not a user-authored table column.

The survey loads this metadata without discarding it. Because memefish v0.8.1
cannot represent an expression as an index key, reconstruction deliberately
fails closed when `EXPRESSION` is non-NULL. Emitting the internal column name
would produce misleading DDL and is therefore prohibited. Parser support can
be added later without weakening the metadata contract established here.

## Query-plan observation

On the pinned Omni runtime, both automatic selection and `FORCE_INDEX` used a
normal `Scan` node with `scan_type: IndexScan` and the expression index name as
`scan_target`. The key equality appeared under `Seek Condition`, while the
distributed access range appeared as `Split Range` under `Distributed Union`.
A mixed column/expression key produced the same operator family, and an
explicit `_BASE_TABLE` control produced a `TableScan` instead.

No expression-index-specific physical operator was observed. The retained
contract therefore checks existing operator/link/metadata combinations and
does not add a new plan-vocabulary family.
