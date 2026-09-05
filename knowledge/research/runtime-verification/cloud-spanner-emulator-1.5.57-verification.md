---
type: Research Note
title: "Cloud Spanner Emulator 1.5.57 verification"
description: "Release-note probes, survey compatibility and metadata comparison for the official linux/amd64 emulator image on 2026-09-05."
tags: [spanner, emulator, release, runtime, metadata]
status: draft
sources:
  - id: release
    resource: evidence/emulator-1.5.57-release/release.json
    title: "Official v1.5.57 release response"
  - id: runs
    resource: evidence/emulator-1.5.57-release/verification-summary.json
    title: "Selected execution outcomes from 2026-09-05"
  - id: capture
    resource: evidence/emulator-1.5.57-release/infoschema-1.5.57.json
    title: "Pinned linux/amd64 INFORMATION_SCHEMA capture"
---

# Cloud Spanner Emulator 1.5.57 verification

Observed on 2026-09-05; incorporated into OKF on the same date.

## Result

The official **linux/amd64** image passed the existing survey tests and the
release-specific functionality probes described below. Its INFORMATION_SCHEMA
surface is identical to the retained 1.5.56 observation: **198 columns**, with
both rolling index columns still absent.[^capture]

Two important boundaries remain:

1. The official `1.5.57` and `latest` tags resolved on 2026-09-05 to one amd64 image.
   The registry had no other tag containing `1.5.57` when rechecked at the end.
   Native arm64 execution is **unverified**. The normal runtime registry requires
   both amd64 and arm64 pins, so the repository pin remains at 1.5.56.
2. Cloud Queue DDL is listed in the release announcement, but the distributed
   image rejects both tested statement prefixes, and the tagged public parser
   has no queue production. Queue support was **not demonstrated**.

An additional change-stream case fails on both 1.5.56 and 1.5.57. It is recorded
as an existing behavior, not a newly established release regression.

## Release and runtime identity

- [Official release](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/releases/tag/v1.5.57)
- Published: `2026-09-04T18:57:52Z` / `2026-09-05 03:57:52 JST`.
- Tagged upstream commit: `fc811a1a93c7e4784db7f48fe15e72fe94f39d38`.
- [Source comparison](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/compare/v1.5.56...v1.5.57): two commits; the Copybara import and its merge.
- Image: `gcr.io/cloud-spanner-emulator/emulator:1.5.57`.
- Observed manifest: `sha256:4987860c9f8ecf1fffbbcdac115cb88cb9d1a42bd966c235a9ab843aea34fbd1`.
- Image configuration: `sha256:f2c7e3e51124a047fc7bfd4130284f64cfa6ab0f6f3bb286cf1f74b6ee629af7`.
- Actual container platform, inspected by the capture producer: `linux/amd64`.
- Host: macOS arm64; Colima Docker daemon: Linux arm64. The container therefore
  ran through architecture translation, not natively. Test binaries were built
  with `GOARCH=amd64` using Go 1.26.6.
- Read-only registry observation: [release-registry-observation.json](evidence/emulator-1.5.57-release/release-registry-observation.json).

The live GitHub API and official registry established v1.5.57 as the latest
release at the observation time.[^release] This note does not make a
continuously refreshed latest-release claim.

## Every release-note item and its evidence

| Announced change | Verification |
| --- | --- |
| Tables with an implicit primary key | GoogleSQL and PostgreSQL inserts passed; distinct generated row IDs and hidden `SELECT *` behavior passed. GoogleSQL metadata and schema reconstruction also passed. |
| NULL-safe distinctness operators | Both dialects passed equal/unequal values, one/both NULL, NaN equality, operand reversal, and negation checks. |
| PostgreSQL sampling | `BERNOULLI(0/100)` and `SPANNER.RESERVOIR(1)` passed. `SYSTEM` and `REPEATABLE` remain `Unimplemented`. |
| Database capacity expansion | Tagged source sets 5,000 tables and twice that number of indexes. These capacity boundaries were source-verified, not stress-tested. |
| Cloud Queue schema support | `CREATE QUEUE` and `CREATE CLOUD QUEUE` fail at the keyword after `CREATE`. The tagged grammar contains no queue production. Added queue catalog/builder/validator source does not establish exposed DDL support. |
| PostgreSQL parameterized partitioned updates | `$1`/`$2`, mapped to client parameters `p1`/`p2`, updated two rows and persisted the expected values. |
| PostgreSQL nullability alteration | Adding the constraint rejected NULL; removing it allowed a NULL insert. |
| Indexes storing unresolved commit times | Insert with `PENDING_COMMIT_TIMESTAMP()` followed by another update in the same transaction passed with a unique index storing that timestamp. Committed data and an index read passed. |
| Feature-flag synchronization fix | Tagged source returns the `Flags` value rather than a reference whose lock has expired. Source-verified; no concurrency stress test was run. |
| Change-stream schema crash fix | Column removal followed by the original no-op `SET FOR` sequence from issue 371 passed, as did column-list changes. An initially `FOR ALL` stream also accepted table removal followed by stream alterations. |

[Issue 371](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/371)
was still open when inspected. That issue state does not override the runtime
result for the tested sequence.

## Existing survey compatibility

The tracked-plus-untracked Spanalyzer source at the time of verification was
copied to an isolated directory. Its base was
`ed36db1e144340a160a53e1f6ab5c7e628d2cffe` and its accepted dirty snapshot
fingerprint was
`38472b315b8b7e8fae15931b22bc833550bf6da9610a10810f9a77f8c2cf6fbd`.
Only the copied `survey/internal/runtimepins/runtimepins.go` was adjusted to
return the explicit 1.5.57 digest for emulator startup. The original pin registry
and every existing test assertion were retained for the successful run.

- `GOWORK=off`, `GOFLAGS=-mod=readonly`, `GOARCH=amd64`.
- `go test -count=1 -json -timeout=15m -skip '^TestDrift_OmniTableMetas$' ./...`.
- **189 top-level tests passed**, **0 failed**, across **11 test packages**.
- Four optional managed-Spanner tests skipped because no managed target was
  supplied. The Omni integration test was explicitly excluded from this
  emulator-only run.
- Emulator metadata drift, UUID behavior, expression-index rejection, named
  schema roundtrip, proto descriptors, and repeated queries in one capture
  transaction all passed.
- Full log: [survey-tests-r2.jsonl](evidence/emulator-1.5.57-release/survey-tests-r2.jsonl).

An initial probe-only pin change removed the missing arm64 entry. That failed
the existing two-platform contract before containers started. It was abandoned
in the isolated copy; it was never applied to the actual checkout. Another
initial capture attempt needed an explicit Colima `DOCKER_HOST`. Neither setup
failure is counted as emulator execution evidence.

## INFORMATION_SCHEMA comparison

The unmodified repository capture command ran with an explicit `--image`,
producing the retained JSON linked below.

- New capture: [infoschema-1.5.57.json](evidence/emulator-1.5.57-release/infoschema-1.5.57.json).
- Baseline: the [retained arm64 1.5.56 capture](../../../survey/infoschem/evidence/emulator/5b1e3607fe8574fb04144eeabfa54120559fb01968ffe3ffc0a9a8f6776fc454/linux-arm64-74894aba5bf3.json).
- Both surface hashes:
  `74894aba5bf3006fd3c85df5a1f1c2e14361bcb5c86424895623fb4ad80118a7`.
- Entire `columns` and `rolling_queryability` values are equal.
- `INDEXES.SEARCH_UNNEST`: not advertised.
- `INDEX_COLUMNS.EXPRESSION`: not advertised.
- Only target identity, observation time, producer identity, and invocation
  identity differ. This is a cross-version and cross-architecture observation;
  it does not demonstrate native arm64 1.5.57 compatibility.
- Compact comparison: [infoschema-comparison.json](evidence/emulator-1.5.57-release/infoschema-comparison.json).

The implicit-key table reconstructed to:

```sql
CREATE TABLE NoPK (
  Value STRING(MAX),
  rowid INT64 NOT NULL GENERATED BY DEFAULT AS IDENTITY (BIT_REVERSED_POSITIVE) HIDDEN
) PRIMARY KEY (rowid ASC)
```

Recreating this DDL under a new table name and inserting two rows without rowid
produced two distinct IDs. Explicit `PRIMARY KEY ()` retained singleton
semantics: a second insert was rejected.

## Additional change-stream observation

Both 1.5.56 and 1.5.57 reject the final statement in this isolated sequence with
`FailedPrecondition`, claiming the table is still explicitly tracked:

```sql
CREATE TABLE SubsetTable (ID INT64 NOT NULL, Status STRING(MAX)) PRIMARY KEY (ID);
CREATE CHANGE STREAM SubsetStream FOR SubsetTable(Status);
ALTER CHANGE STREAM SubsetStream SET FOR ALL;
DROP TABLE SubsetTable;
```

Creating the stream for the whole table and then changing it to an explicit
column list before `SET FOR ALL` reproduces the same outcome on both versions.
The longer sequence involving an earlier column drop also fails on 1.5.57,
whether sent in one batch or as separate DDL operations. An ordinary whole-table
tracking to `FOR ALL` transition without the column-subset step passed.

This was reduced independently from issue 371's process-crash sequence. The
server remains available and returns a normal error. Managed Spanner and Omni
were not probed for this additional case. No upstream issue was created.

- Latest minimal repro: [subset-probe-1.5.57.jsonl](evidence/emulator-1.5.57-release/subset-probe-1.5.57.jsonl) (expected recorded failure).
- Same repro on old amd64 pin: [subset-probe-1.5.56.jsonl](evidence/emulator-1.5.57-release/subset-probe-1.5.56.jsonl) (same failure).
- Old manifest: `sha256:24c921c60e277e1ecfe50188169bf1aa818c8603d96842c827d4d1920206811c`.
- Longer latest sequence: [release-probes-r3.jsonl](evidence/emulator-1.5.57-release/release-probes-r3.jsonl).

## Reproduction and retained evidence

The [probe source](evidence/emulator-1.5.57-release/release_probe_test.go.txt)
is preserved as text so intentionally failing diagnostic cases are not picked
up by normal Go package tests. From the repository root, copy it to a temporary
Go test file and run it from the survey module:

```sh
runtime_probe_dir=$(mktemp -d)
cp knowledge/research/runtime-verification/evidence/emulator-1.5.57-release/release_probe_test.go.txt \
  "$runtime_probe_dir/release_probe_test.go"
cd survey
GOWORK=off GOFLAGS=-mod=readonly GOARCH=amd64 go test \
  -tags=integration -count=1 -v -timeout=10m \
  -run '^Test(GoogleSQLRelease|PostgreSQLRelease)$' \
  "$runtime_probe_dir/release_probe_test.go"
```

Use Go 1.26.6, an accessible Docker daemon and, on an arm64 host, support for
executing amd64 containers and test binaries. Set `DOCKER_HOST` and
`TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE` as required by the local Docker setup.
The probe creates dedicated containers and closes its fixtures on exit.

The release probe command passed. Its Cloud Queue subtest records rejection
and continued server availability; a passing collector is not successful Queue
DDL execution. To reproduce the existing change-stream failure, select
`^TestChangeStreamSubsetToAllTableDrop$`. Set `RELEASE_PROBE_IMAGE` to the old
fully pinned image above for the baseline comparison. A nonzero exit is the
recorded outcome for that diagnostic reproduction.

The [verification summary](evidence/emulator-1.5.57-release/verification-summary.json)
names the retained run outcomes and exit codes.[^runs] JSONL logs and the
capture are stored in [release evidence](evidence/emulator-1.5.57-release/).
Local workspace and scratch paths in selected logs were replaced with portable
markers; the [migration provenance](evidence/migration-provenance.json)
records original and retained SHA-256 values. The evidence is an observation
record, not an execution attestation.

The [source audit](cloud-spanner-emulator-1.5.57-source-audit.md) follows all
105 changed files and verifies material changes absent from the release notes.
The documentation migration itself did not execute the runtime again.

[^release]: [Full official release body](evidence/emulator-1.5.57-release/release.json)
    and [registry observation](evidence/emulator-1.5.57-release/release-registry-observation.json),
    fetched during the 2026-09-05 verification.
[^runs]: [Run summary](evidence/emulator-1.5.57-release/verification-summary.json)
    with [survey JSONL](evidence/emulator-1.5.57-release/survey-tests-r2.jsonl),
    [release probes](evidence/emulator-1.5.57-release/release-probes-final.jsonl),
    and version-paired subset-to-all reproductions.
[^capture]: [v1.5.57 INFORMATION_SCHEMA capture](evidence/emulator-1.5.57-release/infoschema-1.5.57.json)
    and [tuple comparison](evidence/emulator-1.5.57-release/infoschema-comparison.json).
