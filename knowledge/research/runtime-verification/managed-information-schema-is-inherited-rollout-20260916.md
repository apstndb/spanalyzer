---
type: Research Note
title: "Managed INFORMATION_SCHEMA IS_INHERITED rollout, 2026-09-16"
description: "Single-database managed Cloud Spanner observation of flapping COLUMN_OPTIONS and LOCALITY_GROUP_OPTIONS IS_INHERITED advertisement and inherited locality_group row emission."
tags: [spanner, information-schema, managed, emulator, omni, runtime]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: observations
    resource: evidence/is-inherited-rollout-20260916/observations.json
    title: "Redacted derived counts from managed-cycle-20260917"
  - id: retained-capture
    resource: ../../../survey/infoschem/evidence/managed/20260916T171439Z-89b4ab9e231d.json
    title: "Producer-owned managed INFORMATION_SCHEMA capture at 2026-09-16T17:14:39Z"
  - id: retained-baseline
    resource: ../../../survey/infoschem/evidence/managed/20260825T120538Z-f8b0ea3092e7.json
    title: "Retained managed INFORMATION_SCHEMA baseline at 2026-08-25T12:05:38Z"
  - id: official-docs
    resource: https://docs.cloud.google.com/spanner/docs/information-schema
    title: "Official Cloud Spanner INFORMATION_SCHEMA reference, retrieved 2026-09-17"
  - id: reconstruction-columns
    resource: ../../../survey/astconv/to_ast_table.go
    title: "Schema reconstruction emits every COLUMN_OPTIONS row as column OPTIONS"
  - id: reconstruction-locality
    resource: ../../../survey/astconv/to_ast_locality_group.go
    title: "Locality-group reconstruction skips the built-in default group by name"
---

# Managed INFORMATION_SCHEMA IS_INHERITED rollout, 2026-09-16

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Observed on 2026-09-16 between 16:56 and 17:15 UTC on one managed Cloud
Spanner database. Database identifiers were never recorded. This is a dated
single-database observation of a flapping INFORMATION_SCHEMA surface, not a
fleet-wide or durable service contract.

## Scope and runtime identity

- Managed Cloud Spanner: one database; service build identity is not exposed
  by these requests. Target coordinates are omitted.
- Producer: `survey/cmd/infoschema-capture check --target managed`, plus
  direct `INFORMATION_SCHEMA` queries through spanner-mycli.[^observations]
- Retained baseline: managed capture at `2026-08-25T12:05:38.437354Z`, 308
  columns, surface SHA-256
  `f8b0ea3092e7601f260d6d9cb59002dcd0cf4fba6283df8a7c04c004e6a664ee`.[^retained-baseline]
- New retained capture: `2026-09-16T17:14:39.178882Z`, 310 columns, whole-file
  SHA-256 `e4ee194e6cf4f10a3c2e9139e6708d4e046b145a6bde9e59086edd674830613c`,
  surface SHA-256
  `89b4ab9e231d7d61bf417669b6f5d29d1715c3999058b905f60698bc1047c6b0`.[^retained-capture]
- Pinned containers in the same cycle: Emulator
  `gcr.io/cloud-spanner-emulator/emulator:1.5.56` (198 columns) and Omni
  `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta` (308
  columns). Neither advertised `IS_INHERITED`.[^observations]

## Method

The producer compared the managed INFORMATION_SCHEMA column surface against
the retained 308-column baseline. Four consecutive `check` runs between
16:56:07Z and 17:06:59Z recorded either the new 310-column surface or the
retained 308-column surface. Eight additional strong reads selected
`TABLE_NAME`, `COLUMN_NAME`, `SPANNER_TYPE`, and `ORDINAL_POSITION` from
`INFORMATION_SCHEMA.COLUMNS` where `TABLE_SCHEMA = 'INFORMATION_SCHEMA'`.
Eight combined single-use read-only queries each returned four scalar
subqueries: information-schema column count, `COLUMN_OPTIONS` rows with
`locality_group`, all `COLUMN_OPTIONS` rows, and `LOCALITY_GROUP_OPTIONS`
rows. Three samples of the `IS_INHERITED` distribution were identical.

No database locator, project, instance, or user-schema object identifier is
retained.

## Results

Two columns were advertised on the 310-column surface and were absent from
the retained 308-column baseline. No other columns were added, removed, or
changed in that comparison.[^observations]

| Table | Column | Type | Ordinal |
| --- | --- | --- | --- |
| `COLUMN_OPTIONS` | `IS_INHERITED` | `BOOL` | 8 |
| `LOCALITY_GROUP_OPTIONS` | `IS_INHERITED` | `BOOL` | 4 |

Producer `check` runs:

| Observed at (UTC) | Status | Columns | Surface SHA-256 prefix |
| --- | --- | --- | --- |
| 2026-09-16T16:56:07.707925Z | `surface_changed` | 310 | `89b4ab9e231d` |
| 2026-09-16T17:06:52.155787Z | `surface_changed` | 310 | `89b4ab9e231d` |
| 2026-09-16T17:06:55.873321Z | `producer_changed_only` | 308 | `f8b0ea3092e7` |
| 2026-09-16T17:06:59.465014Z | `surface_changed` | 310 | `89b4ab9e231d` |

Direct column-count samples, in order: 310, 308, 310, 310, 310, 310, 310,
310. Seven of eight returned 310. All 310-column samples shared one surface.

Combined single-use reads (column count, `COLUMN_OPTIONS` `locality_group`
rows, all `COLUMN_OPTIONS` rows, `LOCALITY_GROUP_OPTIONS` rows):

| Columns | `locality_group` rows | All `COLUMN_OPTIONS` | `LOCALITY_GROUP_OPTIONS` |
| --- | --- | --- | --- |
| 310 | 0 | 28 | 4 |
| 310 | 1651 | 1679 | 4 |
| 310 | 1651 | 1679 | 4 |
| 310 | 1651 | 1679 | 4 |
| 310 | 0 | 28 | 4 |
| 308 | 1651 | 1679 | 4 |
| 310 | 1651 | 1679 | 4 |
| 310 | 0 | 28 | 4 |

`IS_INHERITED` distribution, identical across three samples:

| Table | Option | `IS_INHERITED` | Count |
| --- | --- | --- | --- |
| `COLUMN_OPTIONS` | `allow_commit_timestamp` | `NULL` | 11 |
| `COLUMN_OPTIONS` | `cassandra_type` | `NULL` | 17 |
| `COLUMN_OPTIONS` | `locality_group` | `true` | 1651 |
| `LOCALITY_GROUP_OPTIONS` | `ssd_to_hdd_spill_timespan` | `NULL` | 2 |
| `LOCALITY_GROUP_OPTIONS` | `storage` | `false` | 1 |
| `LOCALITY_GROUP_OPTIONS` | `storage` | `true` | 1 |

The 1651 inherited `COLUMN_OPTIONS.locality_group` rows have type `STRING`
and value `default`, across 206 tables.

## Interpretation

The advertised column surface flaps between the retained 308-column hash and
the new 310-column hash across consecutive strong reads. The 310-column
surface is the majority observation in this window (seven of eight direct
samples). One producer run in the same minute as two 310-column runs still
saw the 308-column baseline.[^observations]

Row emission rolls independently of column advertisement. The 28 non-locality
`COLUMN_OPTIONS` rows and four `LOCALITY_GROUP_OPTIONS` rows are present in
every combined sample. The 1651 `locality_group = 'default'` rows with
`IS_INHERITED = true` appear and disappear independently of whether the
surface has 308 or 310 columns: they are present on a 308-column sample and
absent on some 310-column samples.

When those inherited rows are present, `LOCALITY_GROUP_OPTIONS.storage` is
`false` for the user group and `true` for the built-in `default` group.
Non-locality column options and `ssd_to_hdd_spill_timespan` have
`IS_INHERITED = NULL`.

Pinned Emulator 1.5.56 and Omni 2026.r2.1-beta in the same cycle do not
advertise `IS_INHERITED`. Omni's 308-column count matches the retained
managed baseline.

## Repository consequences

`buildColumnDef` in `survey/astconv/to_ast_table.go` emits every
`COLUMN_OPTIONS` row as a column `OPTIONS` clause. It does not inspect
inheritance. If the 1651 inherited `locality_group = 'default'` rows are
fed to reconstruction, every affected column would gain a spurious
`OPTIONS (locality_group = 'default')` clause.[^reconstruction-columns]

`toLocalityGroupsDDL` skips the built-in `default` locality group by name
only. That skip prevents a `CREATE LOCALITY GROUP` for `default`; it does
not filter inherited column options.[^reconstruction-locality]

This note records that reconstruction consequence. It does not record a
filter implementation or its status.

## Unverified boundaries

- The steady state after this rollout completes is not known. This window
  shows both surfaces and both inherited-row states.
- Whether `TABLE_OPTIONS` will gain `IS_INHERITED` is not known. These
  samples did not add that column.
- Official documentation status: the
  [INFORMATION_SCHEMA reference](https://docs.cloud.google.com/spanner/docs/information-schema)
  was fetched on 2026-09-17. The retrieved page does not document
  `IS_INHERITED` on `COLUMN_OPTIONS` or `LOCALITY_GROUP_OPTIONS`. Whether
  that page will later describe the column is not known.[^official-docs]
- Emulator and Omni behavior in later releases is not known. Only the pinned
  1.5.56 and 2026.r2.1-beta images were compared in this cycle.

## Evidence

- Derived counts:
  [observations.json](evidence/is-inherited-rollout-20260916/observations.json)
- New producer-owned capture:
  [20260916T171439Z-89b4ab9e231d.json](../../../survey/infoschem/evidence/managed/20260916T171439Z-89b4ab9e231d.json)
- Retained managed baseline:
  [20260825T120538Z-f8b0ea3092e7.json](../../../survey/infoschem/evidence/managed/20260825T120538Z-f8b0ea3092e7.json)
- Artifact hashes:
  [artifact-sha256.json](evidence/artifact-sha256.json)

[^observations]: [Redacted derived counts](evidence/is-inherited-rollout-20260916/observations.json)
    from the producer `check` runs, direct column reads, combined
    single-use reads, and `IS_INHERITED` distribution samples.
[^retained-capture]: [Producer-owned managed capture](../../../survey/infoschem/evidence/managed/20260916T171439Z-89b4ab9e231d.json)
    at `2026-09-16T17:14:39.178882Z`.
[^retained-baseline]: [Retained managed baseline](../../../survey/infoschem/evidence/managed/20260825T120538Z-f8b0ea3092e7.json)
    at `2026-08-25T12:05:38.437354Z`.
[^official-docs]: [Cloud Spanner INFORMATION_SCHEMA reference](https://docs.cloud.google.com/spanner/docs/information-schema),
    retrieved 2026-09-17.
[^reconstruction-columns]: [`survey/astconv/to_ast_table.go`](../../../survey/astconv/to_ast_table.go),
    `buildColumnDef` column `OPTIONS` emission.
[^reconstruction-locality]: [`survey/astconv/to_ast_locality_group.go`](../../../survey/astconv/to_ast_locality_group.go),
    `toLocalityGroupsDDL` built-in `default` group handling.
