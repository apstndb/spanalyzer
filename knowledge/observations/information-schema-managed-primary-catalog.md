---
type: Observation
title: INFORMATION_SCHEMA Managed-Primary Catalog Projection
description: Point-in-time managed selection, digest-pinned Omni and Emulator comparisons, and rolling-column queryability observed on 2026-08-25.
tags: [spanner, spanner-omni, emulator, information-schema, catalog]
status: draft
sources:
  - id: managed-capture
    resource: ../../survey/infoschem/evidence/managed/20260825T120538Z-f8b0ea3092e7.json
    title: Managed Spanner INFORMATION_SCHEMA capture
  - id: omni-capture
    resource: ../../survey/infoschem/evidence/omni/ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741/linux-arm64-f8b0ea3092e7.json
    title: Spanner Omni 2026.r2.1-beta INFORMATION_SCHEMA capture
  - id: emulator-capture
    resource: ../../survey/infoschem/evidence/emulator/5b1e3607fe8574fb04144eeabfa54120559fb01968ffe3ffc0a9a8f6776fc454/linux-arm64-74894aba5bf3.json
    title: Cloud Spanner Emulator 1.5.56 INFORMATION_SCHEMA capture
  - id: projection-source
    resource: ../../information_schema_projection_source.json
    title: Explicit managed-primary projection source
  - id: capture-definition
    resource: ../../survey/infoschem/capture-definition.v0alpha1.json
    title: Versioned INFORMATION_SCHEMA capture definition
  - id: analyzer-manifest
    resource: ../../information_schema_manifest.json
    title: Generated INFORMATION_SCHEMA analyzer manifest
  - id: capture-schema
    resource: ../../survey/schemas/information-schema-capture.v0alpha1.schema.json
    title: INFORMATION_SCHEMA capture contract
  - id: official-information-schema
    resource: https://docs.cloud.google.com/spanner/docs/information-schema
    title: Cloud Spanner INFORMATION_SCHEMA reference
---

# INFORMATION_SCHEMA Managed-Primary Catalog Projection

Observed: 2026-08-25

Status: **the analyzer projection selects one managed-database observation at
one read timestamp; it is not a fleet-wide or durable managed Spanner
version**.

## Retained observations

The capture producer queried the complete definition surface through
`INFORMATION_SCHEMA.COLUMNS` in one read-only transaction. It also used
bounded `LIMIT 0` probes for every registry column marked as rolling. Captures
retain system table and column names, raw types, ordinals, and bounded
queryability outcomes. The versioned capture definition freezes the rolling
probe identities and requires an isolated, read-only survey module build.
Captures contain no project, instance, database, endpoint, credential, raw
service error, or user-schema object identifier.

The selected managed database at `2026-08-25T12:05:38.437354Z` advertised 48
tables and 308 columns. Both rolling columns were queryable in the same
read-only transaction:

- `INDEXES.SEARCH_UNNEST`;
- `INDEX_COLUMNS.EXPRESSION`.

Spanner Omni `2026.r2.1-beta` on `linux/arm64`, with platform-manifest digest
`sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741`,
advertised the same 48 tables, 308 columns, and two queryability outcomes. Its
semantic surface SHA-256 therefore equals the managed capture:
`f8b0ea3092e7601f260d6d9cb59002dcd0cf4fba6283df8a7c04c004e6a664ee`.

Cloud Spanner Emulator `1.5.56` on `linux/arm64`, with platform-manifest digest
`sha256:5b1e3607fe8574fb04144eeabfa54120559fb01968ffe3ffc0a9a8f6776fc454`,
advertised 28 tables and 198 columns. Neither rolling column was advertised.
Its semantic surface SHA-256 is
`74894aba5bf3006fd3c85df5a1f1c2e14361bcb5c86424895623fb4ad80118a7`.

Container tags are descriptive. Digest and platform are the primary release
identity, so these observations can be rerun retrospectively even when a tag
later moves. Their timestamps remain provenance but do not define the release.

## Analyzer selection

[`information_schema_projection_source.json`](../../information_schema_projection_source.json)
names the managed capture by exact path and whole-file SHA-256. Selection never
uses filesystem modification time or a `latest` alias. Omni and Emulator are
comparison evidence; they do not implicitly intersect the managed-primary
projection.

The generated `v0alpha2`
[`information_schema_manifest.json`](../../information_schema_manifest.json)
projects 306 stable observed columns and two registry columns marked `rolling`.
The explicit `runtime_filtered` policy reflects that advertisement and
queryability can roll out separately. Nine documentation-only columns remain
`docs_only_absent` and default-denied, and `SCHEMATA.PROTO_BUNDLE` retains its
existing analyzer projection override to `STRING(MAX)`.

The capture's producer-source and invocation hashes identify the closed inputs
used for this run. They support review and comparison but are not independent
execution attestations.

The managed capture filename records the observation time to the second and
uses the surface hash to distinguish a surface change within that second. The
JSON `observed_at` field remains authoritative for the exact read timestamp;
padding the filename with fractional zeroes would add no evidence.

## Maintenance boundary

Routine reruns add or validate raw JSON observations. The projection changes
only when `information_schema_projection_source.json` explicitly selects a
different managed capture and the generated manifest is regenerated. This
Observation should change only when the selected projection or its material
cross-target interpretation changes; an identical rerun does not justify a new
Markdown report.

The repository gate validates the selected capture, registry, policy, and
exact generated bytes:

```sh
(cd tools && go run ./infoschema-survey-check)
```
