---
type: Research Note
title: "Managed Spanner revalidation, 2026-09-25"
description: "Fresh catalog comparisons, distribution-column queryability, optimizer identity, and quantified-syntax rechecks against the existing managed verification target."
tags: [spanner, managed, runtime, metadata, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: evidence
    resource: evidence/managed-recheck-20260925/receipt.json
    title: "Runtime and source provenance"
  - id: catalog
    resource: evidence/managed-recheck-20260925/infoschema-check.json
    title: "Three-observation INFORMATION_SCHEMA comparison"
  - id: sys
    resource: evidence/managed-recheck-20260925/spanner-sys-check.json
    title: "SPANNER_SYS comparison"
  - id: rows
    resource: evidence/managed-recheck-20260925/row-queryability.jsonl
    title: "Qualified LIMIT 1 distribution probes and optimizer identity"
  - id: release-notes
    resource: https://docs.cloud.google.com/spanner/docs/release-notes
    title: "Official release notes retrieved through dkcli and direct HTML"
---

# Managed Spanner revalidation

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Observed on September 25, 2026 JST (September 24 UTC), against the existing
single-database managed GoogleSQL verification target. PROFILE reports optimizer
version 9. `SUPPORTED_OPTIMIZER_VERSIONS` lists versions 1 through 9, with 9 as
the default. The service build is unavailable. Target coordinates are omitted.

The producer used the existing dirty checkout at
`c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c`, with Go 1.26.8 and the survey
module's independent dependency graph. This is a working-snapshot observation,
not an attestation about the clean commit. Source hashes are retained in
[survey-source-hashes.json](evidence/managed-recheck-20260925/survey-source-hashes.json).

## Catalog comparison

| Surface | Fresh observation | September 16 baseline comparison |
| --- | --- | --- |
| INFORMATION_SCHEMA | 310 columns in all three sequential observations | Same surface hash; producer source changed only |
| SPANNER_SYS | 587 columns | Same surface hash; producer source changed only |

The INFORMATION_SCHEMA check reports `producer_changed_only` only after its
three observations agree. Its compact successful report retains the first
observation, not all three individual timestamps. Both comparisons report
`material_change=false`; producer changes are not server-side schema changes.

`IS_INHERITED` resolves in `COLUMN_OPTIONS` and `LOCALITY_GROUP_OPTIONS` with
`LIMIT 0`. It remains an unrecognized name in `TABLE_OPTIONS`. These probes
establish column resolution, not inherited-option value semantics.

## Distribution-column queryability

All 48 added distribution columns on the twelve QUERY_STATS/TXN_STATS interval
tables accept the existing audit's statement shape:

```sql
SELECT t.`CPU_SECONDS_DISTRIBUTION`
FROM SPANNER_SYS.`QUERY_STATS_TOP_MINUTE` AS t LIMIT 1
```

Forty probes returned one row; eight returned no rows. Thus eight cases establish
accepted queries but do not exercise value decoding from a populated interval.
Distribution units, populations, and percentile accuracy were not investigated.
The [row probes](evidence/managed-recheck-20260925/row-queryability.jsonl) retain
the exact statements, types, values, and errors.

Two exploratory shapes are retained to avoid misclassifying their failures as
missing columns:

- Direct unqualified `LIMIT 0` succeeds for the 24 JSON-string columns but
  returns a STRUCT-result `Unimplemented` error for the 24 array distribution
  columns.
- Wrapping either kind in `TO_JSON_STRING` produces unsupported-function errors
  in these SPANNER_SYS queries. The qualified `LIMIT 1` probes above succeed.

These observations do not isolate whether qualification or LIMIT changes the
first failure; no causal claim is made. They also do not establish completion
of the rollout across other databases or all serving replicas.

## Quantified syntax

The same twelve cases as the
[September 21 investigation](managed-quantified-comparisons-20260921.md) still
return explicit unsupported-feature `InvalidArgument` errors: nine
`= ANY/SOME/ALL` cases and three quantified LIKE cases. The independent local
memefish v0.8.1 parser rejects all twelve. Service feature rejection and local
parse rejection are separate results; neither establishes NULL/empty-set
semantics. Raw results are in
[read-only-probes.jsonl](evidence/managed-recheck-20260925/read-only-probes.jsonl).

## Managed integration checks

All four selected tests passed, with `-count=1` and no skips. The suite took
333.421 seconds, including cleanup assertions for the temporary expression-index
and UUID fixtures. The
[test log](evidence/managed-recheck-20260925/managed-tests.log) records:

- Canonical DDL diagnostics: 330 service statements and 330 reconstructed
  statements, zero unexpected parse errors, and one allowlisted statement.
  This aggregate check is not exact DDL equality: an `AlterTable` statement is
  folded into table metadata, and reconstruction emits one `AlterStatistics`
  statement that the service export omits.
- Drift: 50 advertised SPANNER_SYS tables, 587 columns, eight known-absent
  registry columns, no drift, and zero advertised-but-unqueryable columns.
  Fifty tables were checked and 27 rows decoded.
- Expression indexes: accepted metadata/syntax cases, rejection boundaries,
  fail-closed reconstruction, and cleanup checks passed.
- UUID: metadata/reconstruction and cleanup checks passed.

The three verification commands mirror `verify-managed`, with three
INFORMATION_SCHEMA observations instead of one. A dedicated Go cache and direct
toolchain invocation avoided sandbox cache restrictions and an existing shared
cache version mismatch; no toolchain installation or configuration was changed.

## Documentation freshness and scope

The official release page still starts at September 18 in both directly fetched
HTML and `dkcli` content. The September 22 retrieval discrepancy is gone:
`dkcli` now includes automatic parameterization. The fetched query-statistics
and transaction-statistics pages still do not name the four new distribution
families. Full documents and hashes are retained as identified in the receipt.

This run does not repeat PostgreSQL matching/sampling, queue operations, graph
semantic options, vector-index rebuild, or performance comparisons. Their
previous notes remain dated evidence. No catalog projection or managed drift
allowlist was changed based on this single observation window.
