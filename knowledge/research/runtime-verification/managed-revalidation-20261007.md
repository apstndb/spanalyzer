---
type: Research Note
title: "Managed Spanner revalidation, 2026-10-07"
description: "New ACTIVE_BATCH_OPERATIONS advertisement with inconsistent execution, changed graph option recognition, and a data-dependent transaction-statistics decoder failure."
tags: [spanner, managed, runtime, metadata, queues, postgresql, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: receipt
    resource: evidence/managed-recheck-20261007/receipt.json
    title: "Sources, dependencies, executed commands and owned cleanup"
  - id: catalog
    resource: evidence/managed-recheck-20261007/catalog-comparison.json
    title: "Three current SPANNER_SYS observations and added-column tuples"
  - id: retained-sys
    resource: ../../../survey/spannersys/evidence/managed/20261007T142932Z-9bb70c640b69.json
    title: "Immutable provenance-complete managed SPANNER_SYS capture"
  - id: active-controls
    resource: evidence/managed-recheck-20261007/active-batch.jsonl
    title: "Repeated scans, projection controls, LIMIT 0 and existing-table control"
  - id: previous
    resource: managed-revalidation-20261005.md
    title: "Earlier October 5 JST managed revalidation"
  - id: last-comparison
    resource: cross-engine-comparison-20261005.md
    title: "Later October 5 UTC managed and container samples"
  - id: docs
    resource: evidence/managed-recheck-20261007/documentation.json
    title: "Full official document and release-feed retrieval provenance"
---

# Managed Spanner revalidation, October 7

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Observed on October 7, 2026, 14:26-14:44 UTC / 23:26-23:44 Asia/Tokyo,
against the configured managed verification database and an owned temporary
PostgreSQL database. A managed build identity and fleet rollout boundary are
unavailable. Connection coordinates and request IDs are redacted; receiver
and renewal tokens are retained only as hashes.

Two previously measured managed/Omni differences changed in this sample:
SPANNER_SYS now advertises `ACTIVE_BATCH_OPERATIONS`, and graph metadata emits
`allow_view_with_volatile_column=FALSE`. Advertisement is not successful
execution: the new system table's scans failed, and explicit graph-option
values remain unimplemented. The managed integration gate also exposed SQL
NULL values that the existing transaction-statistics decoder cannot handle.

The source is `c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c` plus the pre-existing
working snapshot. Measured implementation and dependency hashes match October
5. Go is 1.26.8; survey checks use `GOWORK=off`, Spanner v1.88.0 and memefish
v0.8.1, while feature probes use the existing workspace's Spanner v1.91.0.
Owned fixture names changed to the `20261007r1` suffix. Six focused NULL
queries and a separate active-table control producer were added.

## Catalog change and queryability

| Surface | Previous managed sample | October 7 managed observation |
| --- | --- | --- |
| INFORMATION_SCHEMA | 48 tables / 310 columns | Same surface; three agreeing observations |
| SPANNER_SYS advertisement | 50 tables / 587 columns | 51 / 598; three checks agree, followed by a matching immutable capture |
| ACTIVE_BATCH_OPERATIONS | Not advertised; SELECT returned table not found | Advertised with 11 columns; scans returned Internal or table not found |
| Graph-level volatile option | Absent from graph metadata; explicit values unknown | Default FALSE appears in metadata; explicit false/true return Unimplemented |

The new SPANNER_SYS surface hash is
`9bb70c640b69e4929e4b44ed834d69e1dfcd5fc17cc53e017f541b9f58f9ec8f`.
All 11 additions belong to `ACTIVE_BATCH_OPERATIONS`; no old column was
removed, retyped, or reordered. The complete tuples equal the retained Omni
r4 arm64 [capture](../../../survey/spannersys/evidence/omni/9f10586c5de366bec5c0ff6cf542a67c21c8363e8a12a3deb9c372a07c9aa8b6/linux-arm64-9bb70c640b69.json).
This is advertisement parity, not execution parity.

The last retained managed `SELECT * ... LIMIT 1` table-not-found observation
was `2026-10-05T00:20:00.850782Z`. New advertisement was first sampled here
at `2026-10-07T14:26:37.415349Z`, again at 14:29:26 and 14:29:29, and in the
new capture at 14:29:32. These observations bracket this investigation's
change; they do not identify an engine deployment timestamp or global rollout.

| Ordinal | Advertised column | Type |
| --- | --- | --- |
| 1 | TEXT | STRING(MAX) |
| 2 | QUERY_ID | STRING(MAX) |
| 3 | NUM_PARTITIONS_TOTAL | INT64 |
| 4 | NUM_PARTITIONS_COMPLETE | INT64 |
| 5 | NUM_TRIVIAL_PARTITIONS_COMPLETE | INT64 |
| 6 | PROGRESS | STRING(MAX) |
| 7 | ROWS_PROCESSED | INT64 |
| 8 | STATUS | STRING(MAX) |
| 9 | RESULT | STRING(MAX) |
| 10 | START_TIMESTAMP | TIMESTAMP |
| 11 | LAST_UPDATE_TIMESTAMP | TIMESTAMP |

The [queryability controls](evidence/managed-recheck-20261007/active-batch.jsonl)
contain 19 records. The original read-only probe and three repeated star scans
returned `Internal`. `COUNT(*)` also returned `Internal`. Eleven individual
column scans returned five `Internal` and six `InvalidArgument` responses;
the latter messages say **table not found**, not column not found.
`LIMIT 0` returned OK, while an intentionally nonexistent column was rejected
as unrecognized. The existing `ACTIVE_PARTITIONED_DMLS` control returned OK.

No successful scan or populated active-batch row was observed. These mixed
results establish an advertisement/queryability mismatch on this target.
Partial rollout is a possible explanation, not a proven cause. The retained
Omni r4 star scan returned OK with zero rows, so the newly equal catalog does
not eliminate this execution difference. The control producer's initial
attempt failed on unused imports before SQL; its corrected run is separate
from that preparation failure in the receipts.

All 48 existing distribution-column queries succeeded: 40 had a row and eight
were empty. October 5's earlier revalidation had 16 populated and 32 empty.
Population varies with workloads and aggregation windows; this is not a
column-surface change or proof that every populated value decodes correctly.
Optimizer versions remain 1-9, with default/PROFILE version 9. The twelve
bounded quantified comparisons still return unsupported-feature errors; the
approximate-count controls return 2 on their small inputs.

## Integration failure and SQL NULL evidence

The required, uncached managed suite ran all four tests without skips.
Canonical DDL, expression-index metadata/reconstruction/cleanup, and UUID
passed. `TestDrift_RealTableMetas` failed while decoding
`SPANNER_SYS.TXN_STATS_TOP_10MINUTE`: column 8 could not be decoded into a
non-nullable `float64`. The complete [log](evidence/managed-recheck-20261007/managed-tests.log)
and nonzero exit remain recorded; the run is not reported as fully passing.

The column selection follows the advertised order. Column 8 is zero-based
and maps to ordinal 9, `AVG_PARTICIPANTS`. The current `TxnStatsTop` fields
`AvgParticipants` and `AvgTotalLatencySeconds` are plain `float64`.
Six additional [read-only queries](evidence/managed-recheck-20261007/readonly.jsonl)
searched TOP and TOTAL minute/10-minute/hour tables for NULL participants.
Two TOP_10MINUTE rows and three TOP_HOUR rows had:

- `COMMIT_ATTEMPT_COUNT=0`;
- `AVG_PARTICIPANTS=NULL` and `AVG_TOTAL_LATENCY_SECONDS=NULL`;
- `AVG_COMMIT_LATENCY_SECONDS=0` and `AVG_BYTES=0`.

Other tested interval/table pairs returned no matching rows. This is a
reproduced, data-dependent decoder boundary, not a measured new schema type
or nullability rollout. Row interval ends include September 29 and October
4/5; their timestamps do not identify when the NULL representation began.

The independent capture also proves that `ACTIVE_BATCH_OPERATIONS` is absent
from the existing decoder registry. The drift test fails on NULL decoding
before reporting that table mismatch. Both nullable decoding and registry
coverage need maintenance; no implementation change is included in this run.

Canonical diagnostics remain 330 service and 330 reconstructed statements,
zero unexpected parse errors and one allowlisted statement. Equal counts are
not exact DDL equivalence or proof of complete graph-option preservation.

## Changed graph metadata and recognition

The 53-record GoogleSQL probe repeats the previous SQL/fixture shapes.
Graph metadata now adds `allow_view_with_volatile_column=FALSE` beside
`validate_element_key_uniqueness`, in both inspected new graphs. Earlier
managed records contained only the key-uniqueness option.

Explicit false and true values previously returned `InvalidArgument` for an
unknown option. Both now return `Unimplemented`: the option is not supported
in a graph statement. Omitting it still permits the tested `GENERATE_UUID()`
view graph when uniqueness validation is false, and MATCH returns two rows.
These metadata and error outcomes agree with the earlier Omni r4 sample;
the option's intended enforcement remains unverified.

Automatic parameterization, uniqueness controls, graph descriptions/synonyms,
and the nine approximate-count queries retain their previous outcome codes.
The only changed GoogleSQL codes are the two explicit volatile-option cases.

## PostgreSQL, queues and documents

All 40 PostgreSQL and 32 queue observation codes match the October 5 producers.
PostgreSQL matching operators work; direct `pg.ilike`/`pg.not_ilike` and the
speculative `spanner.*` calls still report missing functions. BERNOULLI and
SPANNER.RESERVOIR controls work; SYSTEM remains unimplemented.

Queue lifecycle, receive, transaction rollback, ACK and delayed-delivery
controls completed. Whole-array lease bindings succeed under default and
AUTO_PARAMETER OFF; scalar parameters inside arrays fail under both modes.
Renewal/receiver token-hash continuity is verified. The malformed token still
returns InvalidArgument. These observations continue to differ from the Go
example and invalid-token description in the queue usage reference.

Full official Markdown, direct release HTML and the release feed were fetched
again. All three release sources have September 18 as the latest entry.
Release Markdown, queue usage and PostgreSQL function Markdown are byte-equal
to the October 5 retrievals; direct HTML differs in bytes without a new release
entry. DDL reference content was also retained. Source publication dates are
separate from the new runtime observations above. An exact-name web search
and Developer Knowledge search did not find an applicable active-table
reference; that search outcome does not prove documentation absence.

## Cleanup, evidence and follow-up

Feature producers successfully dropped their created graphs, views, table,
queue and temporary PostgreSQL database. Final scans found no owned
date-prefixed tables, graphs or generated queue routines; GetDatabase for the
temporary PostgreSQL database returned NotFound. Existing database objects,
dependencies and projection selectors were not changed.

The [receipt](evidence/managed-recheck-20261007/receipt.json) identifies the
sources, command exits, build identities and durable full-document directory.
Records comprise 86 read-only, 53 GoogleSQL, 40 PostgreSQL, 32 queue, 19
active-table-control and four cleanup observations, including deliberate
negative controls. New immutable producer evidence stays in `survey/`;
research captures and verification sources stay beside this note.

Run the retained [offline verifier](evidence/managed-recheck-20261007/verify.py)
to check changes, stable controls, retained failures, cleanup and artifact
hashes. Documentation validation receipts are retained in the same directory.
No commit, push, publication, dependency upgrade or selector migration was
performed. Vector rebuild, full plan matrices, performance and fleet-wide
rollout are outside this recheck.
