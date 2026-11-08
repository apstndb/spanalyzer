---
type: Research Note
title: "Managed vector index rebuild verification, 2026-09-19"
description: "Verification of documentation commit 12599355: GoogleSQL REBUILD and PostgreSQL concurrent REINDEX, distinguishing syntax acceptance from operation completion."
tags: [spanner, managed, vector-index, ddl, postgresql, memefish]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: mirror-commit
    resource: https://github.com/apstndb/spanner-docs-mirror/commit/12599355b809e2995701ad1fa8ff4fdb0ed4fcdf
    title: "Inspected documentation update"
  - id: google-ddl
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_vector_index
    title: "GoogleSQL ALTER VECTOR INDEX, retrieved 2026-09-19"
  - id: pg-ddl
    resource: https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#reindex_index
    title: "PostgreSQL REINDEX INDEX, retrieved 2026-09-19"
  - id: practices
    resource: https://docs.cloud.google.com/spanner/docs/vector-index-best-practices
    title: "Vector index refresh options, retrieved 2026-09-19"
---

# Vector index rebuild verification

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

## Inspected update

All six files in commit `12599355b809e2995701ad1fa8ff4fdb0ed4fcdf`
(commit time 2026-09-17 18:02:00 UTC) were inspected:

| File | Change and interpretation |
| --- | --- |
| GoogleSQL DDL | Adds ALTER VECTOR INDEX REBUILD and describes background in-place refresh with unchanged options. |
| PostgreSQL DDL | Adds REINDEX INDEX CONCURRENTLY; CONCURRENTLY is required and only vector indexes are supported. |
| Vector index best practices | Separates same-options in-place refresh from create-new-index migration when changing index options. |
| Spanner Omni landing page | Reorganizes getting-started links, secure-deployment links, license, console, CLI and pricing navigation; no runtime capability change established. |
| logs/urls.txt | Fetch timestamps; not additional feature evidence. |
| metadata.yaml | Sync timestamp; file_count remains 918. |

## Method

On 2026-09-19 JST (2026-09-18 UTC), the existing managed verification profile
was used for GoogleSQL, and a temporary PostgreSQL database was created on the
same instance. Each probe created its own table, inserted 1,000 deterministic
2D FLOAT32 vectors, and created an EUCLIDEAN vector index with tree_depth=2,
num_leaves=2 and a stored marker column. No existing application data was changed.

The producer captures UpdateDatabaseDdl submission separately from operation
Wait completion. It attempts ANN reads and a marker update after submission,
then records the final operation result and subsequent ANN/DDL output.
No claim of active rebuild overlap follows merely from these request timings.

## Runtime results (initial run, 2026-09-19)

Both dialects **accept the syntax but fail operation completion with
Unimplemented** on these targets:

```text
REBUILD is not supported yet for vector index `<fixture-index>`.
```

| Probe | GoogleSQL | PostgreSQL |
| --- | --- | --- |
| Create populated vector index | OK | OK |
| Submit documented rebuild statement | OK, operation returned | OK, operation returned |
| Wait for rebuild operation | Unimplemented | Unimplemented |
| Missing vector index | NotFound | NotFound |
| Existing ordinary secondary index | NotFound (not a vector index target) | NotFound (not a vector index target) |
| REBUILD with trailing OPTIONS | Syntax error at OPTIONS | Not applicable |
| Omit CONCURRENTLY | Not applicable | InvalidArgument: must specify CONCURRENTLY |
| ANN before, after submission, and after failure | OK | OK |
| Marker write after submission | OK, visible in final ANN | OK, visible in final ANN |
| Exported vector index DDL before/after failure | Identical | Identical |

This establishes `accepted / explicitly_unsupported`, rather than syntax
rejection. The documentation describes a capability that is not executable on
these targets yet; the cause and broader rollout status are unknown. Success
of reads/writes around a failed request does **not** verify reads/writes during
an actual rebuild. Identical DDL after failure does not prove options survive a
successful rebuild. The initial index supports ANN and stored-column updates;
no tree refresh occurred that this probe can establish.

For memefish, a scoped REBUILD AST/parser addition is justified by the documented
syntax and managed syntax acceptance. Service availability remains a separate
constraint; neither implementation nor an upstream issue is created here.

## Recheck on 2026-09-21

The same probe cases were rerun with fresh fixture names on 2026-09-21 JST
(2026-09-20 UTC). Both dialects still complete with `Unimplemented` and
`REBUILD is not supported yet for vector index`. The GoogleSQL operation
completed at 20:05:09 UTC and the PostgreSQL operation at 20:05:04 UTC.
There is no observed availability change from the initial run.

ANN reads before submission, after submission and after failure succeeded;
the marker update was visible in the final ANN result. Exported vector index
DDL was unchanged. Missing-index, ordinary-index, trailing-OPTIONS (GoogleSQL)
and missing-CONCURRENTLY (PostgreSQL) controls reproduced the initial outcomes.
All fixture indexes and tables were removed, and the temporary PostgreSQL
database was dropped. These results retain the initial limits: no successful
rebuild or active-rebuild concurrency was demonstrated.

- [GoogleSQL recheck](evidence/vector-rebuild-20260921/managed.jsonl)
- [PostgreSQL recheck](evidence/vector-rebuild-20260921/pg-managed.jsonl)
- [Recheck receipt](evidence/vector-rebuild-20260921/receipt.json)
- [Executable assertions](evidence/vector-rebuild-20260921/verify.py)
- [Assertion results](evidence/vector-rebuild-20260921/verification.json)

## Local parser (2026-09-19 snapshot)

The repository's memefish v0.8.1 rejects `ALTER VECTOR INDEX V REBUILD`:
`expected token: SET, pseudo keyword: ADD, DROP, but: REBUILD`.
The existing SET OPTIONS control parses successfully. The inspected local
upstream checkout at `1691dd67c4c71ca545be852b7a66dc08767082a2` likewise has only
ADD/DROP/SET branches in parseVectorIndexAlteration. This is a compatibility
gap relative to documented GoogleSQL syntax, not an assertion about the latest
remote branch. PostgreSQL DDL is outside memefish's GoogleSQL scope.

## Limits

Runtime observations apply to these managed targets only. They do not establish
fleet rollout timing, Omni or Emulator support, improved recall, background tree
refresh, or no-downtime behavior under load. Parser syntax support and successful
execution must be reported separately.

## Retained evidence

- [GoogleSQL capture](evidence/vector-rebuild-20260919/managed.jsonl)
- [PostgreSQL capture](evidence/vector-rebuild-20260919/pg-managed.jsonl)
- [GoogleSQL producer](evidence/vector-rebuild-20260919/probe.go.txt)
- [PostgreSQL producer](evidence/vector-rebuild-20260919/pg-probe.go.txt)
- [Strict parser results](evidence/vector-rebuild-20260919/parser.jsonl)
- [Receipt](evidence/vector-rebuild-20260919/receipt.json)

The PostgreSQL producer's temporary database-name literal is replaced in the
retained copy; choose a fresh name before reproduction. Existing destination
coordinates are supplied only by the user-level profile. Fixture tables and
indexes are deleted, and the temporary PostgreSQL database is dropped.
