---
type: Research Note
title: "Managed Spanner revalidation, 2026-09-29"
description: "Fresh managed catalog and September feature comparisons, including transactional queues and lease argument controls."
tags: [spanner, managed, runtime, metadata, queues, postgresql, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: receipt
    resource: evidence/managed-recheck-20260929/receipt.json
    title: "Execution, dependency, and source provenance"
  - id: catalog
    resource: evidence/managed-recheck-20260929/infoschema-check.json
    title: "Three-observation INFORMATION_SCHEMA comparison"
  - id: sys
    resource: evidence/managed-recheck-20260929/spanner-sys-check.json
    title: "SPANNER_SYS comparison"
  - id: release
    resource: https://docs.cloud.google.com/spanner/docs/release-notes
    title: "Full official Markdown and direct HTML retrieved September 29"
  - id: previous
    resource: managed-revalidation-20260925.md
    title: "Previous managed catalog and integration observation"
---

# Managed Spanner revalidation, September 29

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Observed on September 29, 2026 UTC/JST against the existing managed GoogleSQL
verification database and a temporary managed PostgreSQL database. This samples
the configured instance; it does not identify a global service build or prove
that every region or serving replica has the same rollout state. Target
coordinates and credentials are omitted from retained evidence.

The source is the existing dirty working snapshot at
`c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c`. Catalog checks and the read-only probe
use the independent survey module (Go 1.26.8, Spanner client v1.88.0 and memefish
v0.8.1); feature probes use the existing workspace dependencies and Spanner
client v1.91.0. No dependency upgrade, canonical projection change, or change
to pre-existing database objects is part of this investigation.

## Current documentation and catalog

Both full `dkcli` content and direct HTML still begin with September 18 automatic
parameterization. No subsequent managed release entry was observed. The
September 11 Graph options, September 17 queues, September 18 parameterization,
and September 2/3 PostgreSQL features are covered by the feature probes.

| Surface | September 29 | Comparison with September 25 |
| --- | --- | --- |
| INFORMATION_SCHEMA | 310 columns; three agreeing observations | Same surface and producer hashes |
| SPANNER_SYS | 587 columns | Same surface and producer hashes |
| Optimizer | Versions 1 through 9; default and PROFILE version 9 | Same |
| Qualified distribution-column queries | 48 accepted, 32 populated, 16 empty | All 48 also accepted previously; interval population differs |

Both catalog commands report `producer_changed_only` relative to the selected
September 16 baseline. This is not a server-side change: relative to September
25, even the producer hashes agree. The compact INFORMATION_SCHEMA report
retains the first observation after all three agree.

`COLUMN_OPTIONS.IS_INHERITED` and `LOCALITY_GROUP_OPTIONS.IS_INHERITED` resolve;
`TABLE_OPTIONS.IS_INHERITED` remains an unrecognized name. These `LIMIT 0`
queries establish name resolution, not inherited-value semantics. Distribution
queries use the qualified `SELECT t.<column> ... AS t LIMIT 1` form. Sixteen
empty results do not exercise populated-value decoding. Their population is
workload/window dependent, not evidence of a catalog regression.

The twelve `ANY`/`SOME`/`ALL` comparison and quantified LIKE queries still return
`InvalidArgument`; memefish independently rejects all twelve. The approximate
distinct-count control over `[1,1,2,NULL]` returns 2. Neither observation claims
large-cardinality accuracy, performance, or quantified NULL semantics.

## Feature and integration results

All four existing managed integration tests passed with `-count=1`, no skips,
and 326.065 seconds of package execution (328.4 seconds including invocation).
The canonical-DDL diagnostic again reports 330 service and 330 reconstructed
statements, no unexpected parse errors, and one allowlisted statement. As in
the previous run, equal totals are not exact DDL equality: `AlterTable` is
folded into metadata and reconstruction emits an extra `AlterStatistics`.

The drift audit reports 50 advertised tables, 587 columns, eight known-absent
registry columns, no drift, 50 checked tables, 23 decoded rows, and zero
advertised-but-unqueryable columns. The expression-index and UUID fixture tests
pass their metadata, reconstruction, and cleanup assertions.

| Feature | September 29 observation | Previous managed evidence |
| --- | --- | --- |
| Automatic parameterization, both dialects | Primary-key SELECT default/AUTO/ON generate `_auto_p0_int64_`; OFF does not; invalid mode rejected | Same as September 22 |
| Graph semantic options | Description/synonyms accepted and retained by metadata and DDL export | Same as September 22 |
| Graph key uniqueness | Default/true reject the join-view fixture; false accepts and MATCH returns 2 | Same as September 22 |
| PostgreSQL matching operators | ILIKE / NOT ILIKE / `~~*` / `!~~*` return expected booleans | Same as September 22 |
| PostgreSQL matching functions | `pg.ilike(text,text)` / `pg.not_ilike(text,text)` still do not exist | Same documentation/runtime mismatch |
| PostgreSQL sampling | BERNOULLI 0/100 returns 0/2 rows; RESERVOIR 0/1/100 returns 0/1/2; SYSTEM rejected | Same as September 22 |

The freshly retrieved PostgreSQL function reference still lists both `pg.*`
functions. Speculative `spanner.*` aliases also remain absent. The probes retain
raw PROFILE plans and generated parameter values; no plan-cache hit or
performance guarantee is inferred. All Graph/table/view drops succeeded and
the fixture-prefix TABLES query returned no rows. The temporary PostgreSQL
DropDatabase also succeeded.

## Queues and lease controls

The 32-record GoogleSQL queue lifecycle completed create/drop, enqueue/receive,
rollback, acknowledgment, delayed-delivery, default-delivery, metadata, and
receive-PLAN controls. Enqueue rollback left no row. Ack rollback retained the
message; committed ack removed it, and duplicate asserted ack returned
`OutOfRange`. A one-hour-future message was not received in the bounded window
but remained stored. A normal one-millisecond-stale query read both stored
rows; the receive TVF rejected the same staleness with a strong-concurrency
requirement.

While the strong receive stream remained open, both default and
`AUTO_PARAMETER='OFF'` scalar-inside-array lease calls (`lease_tokens=>[@token]`)
returned `InvalidArgument` with an invalid-token error. The same token bound
as a whole array (`lease_tokens=>@tokens`, `[]string{token}`) renewed successfully,
and the returned token renewed again with automatic parameterization OFF.
These calls completed before the initial lease expired. The invalid literal
`['not-a-token']` also returned `InvalidArgument`.

This matches the September 18 **raw** managed evidence, including
`lease-r2.jsonl` and `managed-r2.jsonl`, and the September 26 Omni probes. There
is no observed lease behavior change. The Omni narrative had incorrectly
called the old managed result zero rows, omitting its error status; that
paragraph was corrected on September 29. An empty `rows` array accompanying
`InvalidArgument` is not a successful empty result. The fresh official usage
page still shows the scalar-inside-array Go example and says malformed tokens
return no rows, so the documented examples do not establish current runtime
success for these forms.

Final read-only cleanup checks found no matching tables/views, property graphs,
or generated queue routines. GetDatabase for the temporary PostgreSQL database
returned `NotFound`. The complete feature probes and final cleanup commands all
exited zero; deliberate negative feature results are recorded separately in
their JSONL observations.

## Scope and reproduction

The existing integration suite uses temporary expression-index and UUID
fixtures. Feature DDL runs only after that suite finishes, so temporary graphs
or queues do not interfere with canonical-DDL reconstruction. PostgreSQL uses
an isolated temporary database; it does not modify the existing GoogleSQL
verification schema. Queue lease tokens are retained only as SHA-256 hashes.

This investigation does not rerun vector-index rebuild, ordinary-column
semantic options, full-text index competition, every optimizer-version matrix,
queue redelivery after expiry, PG queue execution, distributed durability, or
performance measurements. Previous dated notes remain the evidence for those
areas. Acceptance of these bounded cases is not full managed/Omni parity.

Run `python3 knowledge/research/runtime-verification/evidence/managed-recheck-20260929/verify.py`
for offline checks of the retained observations, cleanup, and hashes.
