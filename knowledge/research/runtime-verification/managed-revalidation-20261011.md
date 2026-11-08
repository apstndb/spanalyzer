---
type: Research Note
title: "Managed Spanner revalidation, 2026-10-11"
description: "Unchanged bounded October 9/10 outcomes, expanded undocumented distinctness controls, and first retained NEW_ROW_AND_OLD_VALUES records in both dialects."
tags: [spanner, managed, runtime, json, change-streams, queues, postgresql, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: receipt
    resource: evidence/managed-recheck-20261011/receipt.json
    title: "Current command windows, identities, dependencies and cleanup scope"
  - id: previous-baseline
    resource: managed-revalidation-20261009.md
    title: "October 9 catalog, integration and bounded feature baseline"
  - id: previous-json
    resource: managed-json-comparison-and-change-streams-20261010.md
    title: "October 10 JSON, graph and Change Streams baseline"
  - id: distinctness
    resource: evidence/managed-recheck-20261011/distinctness.jsonl
    title: "64 direct managed cases, including NOT, NULL, NaN and type controls"
  - id: capture
    resource: evidence/managed-recheck-20261011/value-capture-summary.json
    title: "Derived insert, update and delete payloads from both dialects"
  - id: docs
    resource: evidence/managed-recheck-20261011/documentation.json
    title: "Full official document retrievals and live HTML/feed freshness checks"
  - id: capture-definition
    resource: https://docs.cloud.google.com/spanner/docs/change-streams#value-capture-type
    title: "Official definition of the four value capture types"
---

# Managed Spanner revalidation, October 11

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The repeated bounded managed cases agree with the October 9 and 10 results.
New coverage confirms both distinctness spellings and the documented
`NEW_ROW_AND_OLD_VALUES` capture payloads. This does not identify a server
transition: the latter mode was already described in the overview, and no
earlier identical capture-mode probe is retained here. The existing managed
drift gate still fails on the unregistered `ACTIVE_BATCH_OPERATIONS` table.

The observation window is October 11, 2026, 03:04-03:33 UTC / 12:04-12:33 JST.
Command wall times and catalog read timestamps remain in their own receipts;
the configured service build and fleet-wide rollout state are unavailable.

## Repeated outcomes

| Scope | Current bounded result | Comparison |
| --- | --- | --- |
| INFORMATION_SCHEMA | 48 tables / 310 columns; repeated reads agree | Same tuples as October 9 |
| SPANNER_SYS | 51 tables / 598 columns, unchanged surface hash | Three checks plus a new immutable capture agree with October 9 |
| ACTIVE_BATCH_OPERATIONS | Empty star/count/all 11 individual-column scans succeed; missing-column control is rejected | 19 controls match October 9; populated operations remain untested |
| Managed integration | Canonical DDL, expression indexes and UUID pass; drift fails with `unknown_tables=[ACTIVE_BATCH_OPERATIONS]` | All four tests run uncached, with no skips; same failure as October 9 |
| GoogleSQL / PostgreSQL / queues | 53 / 40 / 32 observation codes match | Setup and intentional negative controls are included |
| JSON and conditional constants | All 125 cases match in codes, values and result row types | October 10 corpus, direct API without a local SQL parser |
| Stored JSON, PROTO/ENUM and graph controls | 29 GoogleSQL, 15 PostgreSQL and 12 focused graph codes match | Graph-element/path CAST remains Unimplemented in NORMAL/PROFILE; TO_JSON controls work |
| Change Stream option-only ALTER | Both dialects accept the four true/false settings and preserve the tracked table/column rows | Both 30-record sequences match; the old bare FOR ALL combined examples are still syntax errors |

The catalog surface hash remains
`9bb70c640b69e4929e4b44ed834d69e1dfcd5fc17cc53e017f541b9f58f9ec8f`.
The [new producer capture](../../../survey/spannersys/evidence/managed/20261011T030456Z-9bb70c640b69.json)
is retained without changing an analyzer projection selector. The
INFORMATION_SCHEMA producer-only status against the September baseline is
not a managed surface change; the measured producer files also match October 9.

All 48 distribution-column queries succeed, with 40 populated and eight empty
results. Five sampled TOP_10MINUTE/TOP_HOUR rows have zero commit attempts and
NULL participant/total-latency averages. These workload-dependent rows keep
the nullable-decoder boundary relevant; reaching the registry error does not
show that decoding was repaired. Canonical diagnostics are 330 service and
330 reconstructed statements, zero unexpected parse errors and one allowlist
entry. Counts do not establish exact DDL equivalence.

Queue rollback, ACK and whole-array lease renewal controls pass; hashed token
continuity is checked. Scalar-inside-array and malformed lease tokens still
return InvalidArgument. Automatic parameters, optimizer versions 1-9,
approximate-count controls, explicit unsupported graph options, PostgreSQL
matching/sampling and quantified-comparison rejection retain their earlier
outcomes. Statistics populations, timing and plans are not equivalence claims.

Both dialects accept heartbeat values 100, 999, 1000 and 300000 ms and reject
99/300001 ms. Actual 100 ms child-partition heartbeat reads produce
GoogleSQL 19 records, median 100.075 ms and PostgreSQL
19 records, median 100.056 ms. These intervals use record timestamps,
not client receive latency. PostgreSQL read-options remains NULL-only.

## Expanded distinctness coverage

The [64-case corpus](evidence/managed-recheck-20261011/distinctness-cases.json)
produces 49 OK and 15 InvalidArgument observations. Two successful cases are
ordinary equality controls; successful distinctness queries evaluate both
`IS DISTINCT FROM` and `IS NOT DISTINCT FROM` and return complementary,
non-NULL BOOL values. Four successful PROFILE repeats and one rejected ARRAY
PROFILE repeat match their NORMAL controls.

- SQL NULL with SQL NULL is not distinct; SQL NULL with a value is distinct.
- NaN with NaN is not distinct, while ordinary NaN equality is false. NaN with
  a finite value or SQL NULL is distinct.
- JSON numeric/object/array equality works. SQL NULL of type JSON is distinct
  from JSON null. Object member order does not create distinct values.
- Tested JSON mixtures with INT64, STRING, BOOL, BYTES and INT64 arrays work
  in both operand orders, including the tested array containing NULL.
- Direct SQL ARRAY/STRUCT comparisons are rejected with
  `Grouping is not defined`; this is a later semantic rejection. INTERVAL
  equality works. Incompatible operand types produce signature errors.
- Three deliberately malformed spellings produce syntax errors. They are
  separate from the ten grouping and two type-signature rejections.

The fresh GoogleSQL operators body and live HTML article still contain no
distinctness entry. PostgreSQL's operators page contains it. The
[documentation receipt](evidence/managed-recheck-20261011/documentation.json)
retains the specific page scope and hashes; it does not prove absence across
every official page. Treat the managed GoogleSQL results as an undocumented
runtime compatibility candidate, separate from parser support or a public
documentation contract. No parser or analyzer implementation changed here.

## First retained value-capture payload coverage

The management-page diff adds `NEW_ROW_AND_OLD_VALUES` creation examples in
both dialects. The overview retrieved this run already defines that mode.
Each owned database creates the stream with it, then changes the option to
`NEW_ROW` and `OLD_AND_NEW_VALUES`. Each mode gets separate committed insert,
update and delete mutations. Root partition tokens are followed in memory and
only hashes enter the retained evidence. Nine data change records per dialect
cover all three operations under all three modes.

The UPDATE changes only `Value` from 10 to 20. `Other='stable'` and
`Nullable=NULL` are unchanged. The primary key is retained separately in keys.

| Capture mode | UPDATE new_values | UPDATE old_values |
| --- | --- | --- |
| NEW_ROW_AND_OLD_VALUES | Value=20, Other=stable, Nullable=NULL | Value=10 |
| NEW_ROW | Value=20, Other=stable, Nullable=NULL | Empty |
| OLD_AND_NEW_VALUES | Value=20 | Value=10 |

Both dialects match this distinction. Successful DDL alone is not the basis
of the claim: the [GoogleSQL records](evidence/managed-recheck-20261011/google-capture.jsonl)
and [PostgreSQL records](evidence/managed-recheck-20261011/pg-capture.jsonl)
retain the actual capture type, mutation kinds, column types and payloads.
The [summary](evidence/managed-recheck-20261011/value-capture-summary.json)
is a derived view of those records. This is first coverage in this history,
not first availability on managed Spanner.

## Sources, cleanup and validation

Sixteen full official documents were retrieved and compared with the relevant
prior snapshots; an additional full Change Streams overview was fetched.
Only the management-page body changes among the 15 pages with a retained prior
body comparison. The remaining PostgreSQL operators body and live GoogleSQL
HTML receive focused documentation checks. Official release Markdown/feed
retain September 18 as the latest entry; the live release HTML's first dated
heading agrees. A raw HTML hash change is not an added release-note item.

Source HEAD is `c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c` plus the existing
dirty checkout. All 123 measured implementation/dependency file hashes match
October 9. Go is 1.26.8; the isolated survey uses Spanner v1.88.0, and workspace
feature probes use v1.91.0. Dated copies, build information, source/binary
hashes, exact cases, redacted command results and raw documentation provenance
remain in the [receipt](evidence/managed-recheck-20261011/receipt.json).

Owned tables, graphs, views and queue objects are cleaned up. The baseline
PostgreSQL database and all seven serial follow-up databases return NotFound
after deletion. Earlier 601 retained artifact hashes are preserved. Coordinates
and request IDs are redacted before persistence; lease/partition tokens are
hashed in the producers. The durable task area retains full raw official bodies.

The [offline verifier](evidence/managed-recheck-20261011/verify.py) checks the
matched comparisons, catalog identities, expected values and errors, capture
payloads, cleanup and artifact hashes. OKF all gates and the Markdown language
check validate the documentation changes; their command exits are retained
beside the evidence. The failing managed drift test remains an explicit open
maintenance issue, separate from those passing checks. No implementation,
dependency, selector, commit, push, PR, release or external synchronization
is included. Omni/Emulator, populated batch operations, provisioning,
analytics jobs, performance and fleet rollout are outside this run.
