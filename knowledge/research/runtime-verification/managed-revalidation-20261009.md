---
type: Research Note
title: "Managed Spanner revalidation, 2026-10-09"
description: "First retained successful ACTIVE_BATCH_OPERATIONS scans after October 7 errors, with unchanged catalog tuples and remaining survey maintenance boundaries."
tags: [spanner, managed, runtime, metadata, queues, postgresql, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: receipt
    resource: evidence/managed-recheck-20261009/receipt.json
    title: "Current command exits, sources, dependencies and cleanup"
  - id: catalog
    resource: evidence/managed-recheck-20261009/catalog-comparison.json
    title: "Current catalog observations against October 7"
  - id: sys-capture
    resource: ../../../survey/spannersys/evidence/managed/20261009T025812Z-9bb70c640b69.json
    title: "New immutable managed SPANNER_SYS capture"
  - id: controls
    resource: evidence/managed-recheck-20261009/active-batch.jsonl
    title: "Star, count, individual-column and negative controls"
  - id: previous
    resource: managed-revalidation-20261007.md
    title: "Previous advertised-but-unqueryable surface and NULL evidence"
  - id: docs
    resource: evidence/managed-recheck-20261009/documentation.json
    title: "Full official document, live HTML and release-feed retrievals"
---

# Managed Spanner revalidation, October 9

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The configured managed target now successfully scans
`SPANNER_SYS.ACTIVE_BATCH_OPERATIONS`. October 7 advertised the same 11
columns, but star/count/individual-column reads returned Internal or table not
found. This new observation establishes empty-table queryability on the
sampled target; it does not exercise populated batch values or establish a
fleet-wide rollout completion date.

## Catalog and execution change

| Case | October 7 | October 9 |
| --- | --- | --- |
| INFORMATION_SCHEMA | 48 tables / 310 columns | Same; three observations agree |
| SPANNER_SYS advertisement | 51 tables / 598 columns | Same; three checks and a new immutable capture agree |
| ACTIVE_BATCH_OPERATIONS star scans | Internal | Original read-only check and three repeated controls return OK, zero rows |
| COUNT(*) | Internal | OK, returns 0 |
| Eleven individual-column scans | Five Internal, six table-not-found responses | All eleven return OK, zero rows |
| LIMIT 0 | OK | OK |
| Deliberately nonexistent column | InvalidArgument, unrecognized name | Same |

The SPANNER_SYS surface hash remains
`9bb70c640b69e4929e4b44ed834d69e1dfcd5fc17cc53e017f541b9f58f9ec8f`.
No column advertisement, type or ordinal change explains the execution
difference. The existing `ACTIVE_PARTITIONED_DMLS` control also succeeds.
Both old errors and the new successful reads remain in their dated captures.

The same SQL cases, SDK versions and measured implementation/dependency
sources were used. This records a change in the configured target's sampled
responses without identifying its deployed build or inferring a rollout cause.

The last retained failed identical star-scan control is
`2026-10-07T14:41:53.216715Z`; the first retained successful star scan is
`2026-10-09T03:03:58.665057Z` / 12:03:58 JST. Fifteen active-table controls
change from Internal/table-not-found errors to OK; LIMIT 0 and the positive
and negative controls retain their previous codes. The October 7 advertisement
and October 9 successful reads are separate observations, rather than a
single inferred feature-introduction date.

## Existing integration and NULL boundary

The uncached managed suite ran all four tests without skips. Canonical DDL,
expression-index metadata/reconstruction/cleanup, and UUID passed.
`TestDrift_RealTableMetas` failed with
`unknown_tables=[ACTIVE_BATCH_OPERATIONS]`; no unknown columns, type or ordinal
mismatches were reported. The decoder registry still needs this table.

October 7 failed earlier in row decoding, before reporting that registry gap.
This run's decoded sample reaches the drift report, but six focused NULL
queries still find a TOP_10MINUTE row and three TOP_HOUR rows with zero commit
attempts and NULL `AVG_PARTICIPANTS` / `AVG_TOTAL_LATENCY_SECONDS`.
The plain-float64 decoder boundary therefore remains relevant. Different first
rows or populated statistics can change which issue a drift run exposes;
the changed failure is not evidence that nullable decoding was repaired.

Canonical diagnostics remain 330 service statements and 330 reconstructed
statements, zero unexpected parse errors and one allowlisted statement.
Equal counts do not establish exact DDL or graph-option equivalence.

## Official source freshness

Full official release-note, queue-usage, PostgreSQL-function and DDL Markdown
was fetched again. All four bodies are byte-equal to the October 7 retrievals.
Queue/function source metadata has a later update timestamp, but no body
change follows from that timestamp alone. Live release HTML and the official
feed also retain September 18 as the latest entry.

The raw source bodies and source metadata are retained in the durable task
directory named by the documentation receipt. These document dates do not
date the observed active-table execution change.

## Repeated feature outcomes

All 53 GoogleSQL, 40 PostgreSQL and 32 queue observation codes match October
7. Both inspected graph metadata records still include
`allow_view_with_volatile_column=FALSE`, while explicit false/true remain
Unimplemented. Omission still permits the tested volatile-view graph.
Automatic-parameterization and nine approximate-count controls also match.

PostgreSQL ILIKE/NOT ILIKE and their operator spellings work. Direct
`pg.ilike`/`pg.not_ilike` and speculative `spanner.*` calls still report missing
functions. BERNOULLI and SPANNER.RESERVOIR controls succeed; SYSTEM remains
unimplemented. The public function reference still lists the two rejected
direct `pg.*` calls.

Queue DDL, catalog, enqueue/receive, rollback, ACK and bounded delayed-delivery
controls complete. Whole-array lease binding succeeds under default and
AUTO_PARAMETER OFF; scalar-inside-array bindings fail under both modes.
Receiver/renewal token-hash continuity is verified. The malformed token still
returns InvalidArgument, retaining the discrepancy with the published
invalid-token no-row description.

All 48 distribution-column queries return OK, with 40 populated and eight
empty results in this sample. Optimizer versions remain 1-9 with default
PROFILE version 9. The twelve quantified comparisons retain unsupported
responses. Populated/empty statistics and their first rows remain
workload-dependent observations rather than changes to catalog contracts.

## Source, cleanup and validation

The observation window is October 9, 2026, 02:57-03:11 UTC / 11:57-12:11 JST.
Source is `c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c` plus the existing dirty
working snapshot. Measured implementation and dependency file hashes match
October 7. Go is 1.26.8; survey probes use GOWORK off and Spanner v1.88.0;
feature probes use the existing workspace and Spanner v1.91.0. Only owned
fixture names changed to the `20261009r1` suffix. The runner redacts both JSON
error strings and plain diagnostics before persistence.

All created graphs, views, table, queue and temporary PostgreSQL database were
dropped. Final scans find no owned date-prefixed tables, graphs or generated
queue routines; the temporary PostgreSQL database returns NotFound.
Connection coordinates/request IDs are redacted and lease tokens are hashes.
Existing database objects, dependencies and projection selectors were not
modified.

The [receipt](evidence/managed-recheck-20261009/receipt.json) identifies
sources, commands and build hashes. Records comprise 86 read-only, 53
GoogleSQL, 40 PostgreSQL, 32 queue, 19 active-table-control and four cleanup
observations. Counts include setup and deliberate negative controls, not just
passing assertions. The only nonzero runtime suite command is the retained
managed drift failure. The initial unused-import preparation failure from
October 7 was not repeated.

The [offline verifier](evidence/managed-recheck-20261009/verify.py) checks the
queryability change, stable controls, NULL rows, registry failure, cleanup and
retained hashes. Documentation validation receipts are retained beside the
evidence. No commit, push, publication, implementation fix or selector
migration is included. Populated batch operations, performance and fleet
rollout remain outside this sample.
