---
type: Observation
title: SPANNER_SYS Live-Primary Catalog Projection
description: Pinned managed and Omni SPANNER_SYS metadata, its fail-closed projection policy, and the replacement of spanalyzer's handwritten registry.
tags: [spanner, spanner-omni, spanner-sys, information-schema, catalog]
status: draft
sources:
  - id: portable-manifest
    resource: ../../spanner_sys_manifest.json
    title: Pinned SPANNER_SYS structural and observational manifest
  - id: current-managed-capture
    resource: ../../survey/spannersys/evidence/managed/20261105T030547Z-016061fbe58e.json
    title: November 5 managed SPANNER_SYS capture
  - id: previous-registry
    resource: https://github.com/apstndb/spanalyzer/blob/c1041c8aaa1436b1c1178e1ae063d561393df33c/spanner_sys.go
    title: Previous handwritten SPANNER_SYS registry
  - id: active-partitioned-dml-docs
    resource: https://docs.cloud.google.com/spanner/docs/introspection/active-partitioned-dmls
    title: Active partitioned DML introspection reference
  - id: lock-statistics-docs
    resource: https://docs.cloud.google.com/spanner/docs/introspection/lock-statistics
    title: Lock statistics introspection reference
  - id: transaction-statistics-docs
    resource: https://docs.cloud.google.com/spanner/docs/introspection/transaction-statistics
    title: Transaction statistics introspection reference
---

# SPANNER_SYS Live-Primary Catalog Projection

Observed: August 25 common projection; November 5, 2026 current managed registry revalidation

Status: **the analyzer projection is pinned to matching managed and Omni
metadata; it is not a promise about every deployment or future rollout**.

## Evidence boundary

The committed
[`spanner_sys_manifest.json`](../../spanner_sys_manifest.json) uses the explicit
`v0alpha2` in-repository exporter identity at spanalyzer commit
`a9e779d0236a9d4a53c757cc74bd63b5175d67f0` (`survey/spannersys`). Its whole-file SHA-256 is
`42c5a900fd4b30b711e2a453945c0d2dde9ffcaf2e13fd14dd77380d4c410aa8`;
its content SHA-256 is
`430fc11a13992c9e24022e37d9d1ff5e8ab8838707e50c099e5574488a4176e0`. The v0alpha1 schema and August capture
sidecars remain immutable historical contracts; their identity is not silently
reinterpreted as the new exporter.

The producer used complete, redacted `INFORMATION_SCHEMA.COLUMNS` captures for
two required capable targets:

- managed Cloud Spanner on 2026-08-25;
- Spanner Omni `2026.r2.1-beta` on 2026-08-25.

These are grandfathered legacy captures. The Omni capture records a tag and
date but not an OCI manifest digest or resolved platform, so it cannot prove
the stronger retrospective container identity required for new captures. No
digest or platform should be inferred from the tag after the fact.

The captures contain only SPANNER_SYS table names, column names, raw
`SPANNER_TYPE` values, and ordinals. They do not contain a project, instance,
database, endpoint, or credential. Both selected August targets produced the same 539
column tuples. The separate November 5 managed capture advertises 52 tables /
624 columns, and its read/decode audit succeeds with no name, type or ordinal
drift. The current registry contains 53 tables / 632 columns, including eight
known entries absent from this managed sample. This does not establish universal
availability or update the older common projection.

## Projection rule

The survey registry knows 53 tables and 632 columns. Spanalyzer projects only
the 50 tables and 539 columns observed with identical type and ordinal on both
required targets. A projected column therefore has two matching observations.
A missing or disagreeing observation is not silently unioned into the catalog.

Ninety-three known entries are explicitly `absent_both` relative to the selected
August captures and remain default-denied. This status does not say that the
current managed service lacks them. Eight are the historical exclusions:

- `TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR.USED_BYTES`;
- all seven columns of `VECTOR_INDEX_STATS`.

The other 85 known entries are 48 query/transaction distribution columns,
22 columns in ACTIVE_BATCH_OPERATIONS/BATCH_OPERATIONS, and fifteen lock/split
statistics columns. Registering the managed superset does not invent matching
Omni observations. Selecting replacement common captures is a separate policy
change; this update deliberately keeps the analyzer's 50-table / 539-column
common projection.

The manifest also retains 11 documentation conflicts as provenance. The live
observations control this projection; documentation does not override them.
Those conflicts cover two `ACTIVE_PARTITIONED_DMLS` scalar types, three lock
request STRUCT spellings, and six transaction operation STRUCT spellings.

`decoder_nullable` describes the Go capture decoder. It is evidence about the
producer representation, not SQL nullability, and does not set catalog
`Column.NotNull`.

## Replacement of the handwritten registry

The previous registry at spanalyzer commit
`c1041c8aaa1436b1c1178e1ae063d561393df33c` contained 40 tables and 445
columns. A name/type/order comparison with the pinned manifest found:

- 11 live-only tables and 104 live-only columns admitted by the manifest;
- one old-only table and 10 old-only columns removed or denied;
- 12 exact type differences repaired;
- 22 shared tables whose common-column relative order differed.

For the last metric, `22` is a table count, not a column count. Filtering each
shared table to common columns leaves 224 displaced column positions; comparing
absolute live ordinals finds 229 differences because new and removed columns
also shift later positions.

The 10 old-only columns consist of the eight known-absent entries above plus
`VECTOR_INDEX_METRICS_HISTORY.NUM_CLUSTERS_SAMPLED` and
`VECTOR_INDEX_METRICS_HISTORY.NUM_ZERO_SIZE_CLUSTERS_SAMPLED`, which are not in
the producer's historical 51/547 known registry.

Notable live-primary type repairs include:

- `ACTIVE_PARTITIONED_DMLS.TEXT_FINGERPRINT` and `PROGRESS` are
  `STRING(MAX)`;
- lock request STRUCT field names are `COLUMN`, `LOCK_MODE`, and
  `TRANSACTION_TAG`, each `STRING(MAX)`;
- transaction operation STRUCTs use `TABLE_NAME`, not `TABLE`;
- vector cluster-size percentile values are `INT64`, not `FLOAT64`.

## Consumer boundary

The analyzer maps the manifest's bounded scalar, ARRAY, and STRUCT descriptors
directly to `TypeSpec`. It does not parse nested raw `SPANNER_TYPE` strings and
does not depend on the survey module. Unknown descriptor kinds or scalar codes,
target disagreement, malformed evidence, stale hashes, and projection-policy
violations fail catalog construction.

The producer's JSON Schema is retained once under the nested
[`survey`](../../survey/) module. The internal consumer validator and catalog
tests check the pinned `v0alpha2` contract, complete projection, recursive
types, ordinals, and denied entries.

The repository gate validates the manifest and reproduces its exact bytes from
the retained producer without relying on a sibling checkout:

```sh
(cd tools && go run ./spannersys-survey-check)
```

The exporter receives the pinned legacy source commit explicitly and compares
the complete file byte-for-byte, including its trailing newline. The separate
`survey-import-check` command validates the source-tree to initial imported-
subtree mapping without requiring the unpublished source history.

## Interpretation

This catalog is suitable for type-checking queries against the pinned common
surface. It is not a substitute for checking `INFORMATION_SCHEMA.COLUMNS` on a
specific deployment when rollout timing matters, and it does not guarantee
that a future managed or Omni release preserves the same surface.
