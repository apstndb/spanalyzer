---
type: Research Note
title: "Managed SPANNER_SYS distribution-column rollout"
description: "48 new QUERY_STATS and TXN_STATS distribution columns on managed Cloud Spanner, the 539/587 surface flap, and the absent_both registry constraint."
tags: [spanner, spanner-sys, managed, runtime, metadata]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: v0alpha2-capture
    resource: ../../../survey/spannersys/evidence/managed/20260916T175741Z-58205fd318c3.json
    title: "Retained v0alpha2 managed SPANNER_SYS capture on a 587-column replica"
  - id: legacy-sidecar
    resource: ../../../survey/spannersys/evidence/managed.json
    title: "Legacy v0alpha1 managed sidecar (539-column baseline)"
  - id: observations
    resource: evidence/spanner-sys-distribution-rollout-20260916/observations.json
    title: "Coordinator-derived redacted observations from 2026-09-16"
  - id: added-columns
    resource: evidence/spanner-sys-distribution-rollout-20260916/added-columns.json
    title: "Per-column names, types, and ordinals for the 48 added columns"
  - id: query-stats-docs
    resource: https://docs.cloud.google.com/spanner/docs/introspection/query-statistics
    title: "Official query statistics introspection reference"
---

# Managed SPANNER_SYS distribution-column rollout

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Observed on 2026-09-16 UTC. This is a metadata-surface observation, not a
semantics or performance claim about the new distributions.

## Scope and runtime identity

The observation covers `SPANNER_SYS` column metadata only: table name, column
name, raw `SPANNER_TYPE`, and `ORDINAL_POSITION`. Captures and derived files
contain no project, instance, database, endpoint, or credential.[^observations]

- Managed Cloud Spanner: single-database observation scope. Consecutive strong
  reads of `INFORMATION_SCHEMA.COLUMNS` for `TABLE_SCHEMA='SPANNER_SYS'`
  advertised 587, then 539, then 539 columns. Destination coordinates are
  omitted. The service build identity is not exposed by these requests.
- Retained managed capture: v0alpha2 file
  `survey/spannersys/evidence/managed/20260916T175741Z-58205fd318c3.json`,
  observed at `2026-09-16T17:57:41.707458Z`, 50 tables / 587 columns, surface
  SHA-256 `58205fd318c3b38615f801290c44b78a3dddcbdd2e8916298c6c7a8ff4a2d727`,
  file SHA-256 `5941c929063184797d3be9f47a4828e24e8cde8c0ea0778dc7c588dfe8aa3a01`.[^v0alpha2-capture]
- Legacy managed sidecar: v0alpha1 `survey/spannersys/evidence/managed.json`,
  captured 2026-08-25, 539 columns, surface SHA-256
  `2f1acd9eeff25dcccde68758984e35698f41338d3b4674b7fc7b25d36536fd44`.[^legacy-sidecar]
- Pinned Omni `2026.r2.1-beta`: still 539 columns
  (`unchanged_legacy_baseline`); it does not advertise the new columns.

## Method

Coordinator producers, not re-run here:

- `survey/cmd/spanner-sys-capture check --target managed`
- `survey/cmd/spanner-sys-capture --target managed --write`
- `TestDrift_RealTableMetas` (`spannersys.Audit`)
- direct `INFORMATION_SCHEMA.COLUMNS` queries with
  `TABLE_SCHEMA='SPANNER_SYS'`

The added-column list was derived from the 587-column v0alpha2 capture against
the 539-column legacy sidecar. No column was removed or retyped; the 48 new
columns are appended after each table's last previously known column.[^added-columns]

## Results

Four columns appear on each of the twelve interval tables
(`QUERY_STATS_{TOP,TOTAL}_{MINUTE,10MINUTE,HOUR}` and
`TXN_STATS_{TOP,TOTAL}_{MINUTE,10MINUTE,HOUR}`).

| Family | Columns | Type | Shared-struct ordinals |
| --- | --- | --- | --- |
| QUERY_STATS_TOP_* | `CPU_SECONDS_DISTRIBUTION`, `CPU_SECONDS_DISTRIBUTION_JSON_STRING`, `ROW_SCANNED_DISTRIBUTION`, `ROW_SCANNED_DISTRIBUTION_JSON_STRING` | existing `LatencyDistribution` type; `STRING(MAX)` | 33–36 |
| QUERY_STATS_TOTAL_* | same four | same | 25–28 |
| TXN_STATS_TOP_* | `COMMIT_LATENCY_DISTRIBUTION`, `COMMIT_LATENCY_DISTRIBUTION_JSON_STRING`, `PARTICIPANTS_DISTRIBUTION`, `PARTICIPANTS_DISTRIBUTION_JSON_STRING` | existing `LatencyDistribution` type; `STRING(MAX)` | 24–27 |
| TXN_STATS_TOTAL_* | same four | same | 19–22 |

`*_DISTRIBUTION` uses the same type as the existing `LATENCY_DISTRIBUTION`
columns:
`ARRAY<STRUCT<COUNT INT64, MEAN FLOAT64, SUM_OF_SQUARED_DEVIATION FLOAT64, NUM_FINITE_BUCKETS INT64, GROWTH_FACTOR FLOAT64, SCALE FLOAT64, BUCKET_COUNTS ARRAY<INT64>>>`.

Surface samples from consecutive strong reads, in order: 587, 539, 539. The
flap resembles the concurrent `INFORMATION_SCHEMA.IS_INHERITED` rollout in the
same window.[^observations]

`TestDrift_RealTableMetas` on a 587-column replica saw 50 tables / 587 columns
/ 8 known-absent registry entries, zero ordinal or type mismatches, and failed
closed on 48 unknown advertised columns. `spanner-sys-capture check` reported
`surface_changed` (587 versus the legacy 539 sidecar) because that run still
compared against the v0alpha1 sidecar.

## Interpretation

Managed replicas are mid-rollout. A 587-column replica advertises the 48 new
columns with stable types and ordinals; a 539-column replica does not advertise
them at all. Existing columns keep type and ordinal. Omni `2026.r2.1-beta` has
not joined the rollout, so the two required legacy targets still agree on the
pre-rollout 539-column surface.

## Repository consequences

`spannersys.ExportManifest` fails closed when the two required legacy targets
disagree on a column's presence. The 48 new columns can only be `absent_both`
until an Omni release exposes them. The projected analyzer surface therefore
stays 50 tables / 539 columns. The descriptor registry is now 51 tables / 595
columns; the audit reports registry-known columns missing from a target as
`known_absent`, which is not drift.

`verifySpannerSys` accepts:

- Omni: exactly 50 / 539 / 56 (8 previous absences plus the 48 new registry
  columns);
- managed (`real_spanner`): 50 / 587 / 8 or 50 / 539 / 56.

The 539 / 56 managed entry should be dropped once
`check-spanner-sys-managed` stabilizes on the 587-column surface.

`spanner-sys-capture check` for managed now uses the retained v0alpha2
capture under `survey/spannersys/evidence/managed/` (newest matching
`observed_at`) rather than the legacy sidecar. The only managed v0alpha2 file
present is `20260916T175741Z-58205fd318c3.json`. Omni still has no v0alpha2
capture in that tree, so Omni `check` continues to use the legacy
`omni-2026.r2.1-beta.json` sidecar.

## Unverified boundaries

- Steady state is unverified. The surface still flaps; it is not known when
  every serving replica will advertise 587 columns, or whether a later change
  will add, rename, or drop any of the 48 columns.
- Later Omni releases are unverified. Pinned `2026.r2.1-beta` does not expose
  the columns.
- Official documentation does not yet list the new columns. A 2026-09-17 fetch
  of the [query statistics](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics)
  page (last updated 2026-09-09 UTC) documents `LATENCY_DISTRIBUTION` and
  `LATENCY_DISTRIBUTION_JSON_STRING` only. A same-day fetch of the
  [transaction statistics](https://docs.cloud.google.com/spanner/docs/introspection/transaction-statistics)
  page (also last updated 2026-09-09 UTC) documents
  `TOTAL_LATENCY_DISTRIBUTION` and `TOTAL_LATENCY_DISTRIBUTION_JSON_STRING`
  only. Neither page names the 48 new columns.[^query-stats-docs]
- Semantics of the new distributions (units, population, relation to the
  existing average columns) were not probed.

[^v0alpha2-capture]: Retained capture
    [`survey/spannersys/evidence/managed/20260916T175741Z-58205fd318c3.json`](../../../survey/spannersys/evidence/managed/20260916T175741Z-58205fd318c3.json),
    file SHA-256 `5941c929063184797d3be9f47a4828e24e8cde8c0ea0778dc7c588dfe8aa3a01`.
[^legacy-sidecar]: Legacy sidecar
    [`survey/spannersys/evidence/managed.json`](../../../survey/spannersys/evidence/managed.json),
    539-column v0alpha1 baseline from 2026-08-25.
[^observations]: Derived counts and producer list in
    [observations.json](evidence/spanner-sys-distribution-rollout-20260916/observations.json).
[^added-columns]: Per-column ordinals and types in
    [added-columns.json](evidence/spanner-sys-distribution-rollout-20260916/added-columns.json).
[^query-stats-docs]: Official
    [query statistics](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics)
    page fetched 2026-09-17; last updated 2026-09-09 UTC. The transaction
    statistics page was fetched the same day for the TXN_STATS names.
