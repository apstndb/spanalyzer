---
type: Research Note
title: "Secondary versus n-gram index choice and Emulator 1.5.58"
description: "Versioned query plans, result-equivalence probes, and release-specific DDL checks on isolated fixtures."
tags: [spanner, search, optimizer, emulator, omni, runtime]
status: draft
sources:
  - id: observations
    resource: evidence/fts-index-choice-20260915/summary.json
    title: "Derived per-runtime observations and result hashes"
  - id: producer
    resource: evidence/fts-index-choice-20260915/probe.go.txt
    title: "Go probe with complete DDL, seed data and query matrix"
  - id: release
    resource: https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/releases/tag/v1.5.58
    title: "Official Emulator 1.5.58 release"
  - id: pattern-doc
    resource: https://docs.cloud.google.com/spanner/docs/full-text-search/pattern-matching-function-acceleration
    title: "Official pattern matching acceleration requirements"
---

# Secondary versus n-gram index choice and Emulator 1.5.58

Observed on 2026-09-15 UTC / JST. This is a synthetic
fixture observation, not a performance benchmark or optimizer contract.

## Scope and fixture

The [producer](evidence/fts-index-choice-20260915/probe.go.txt) creates a
uniquely named table with `Id`, `Title`, and a hidden `Tokens` column using
`TOKENIZE_NGRAMS(LOWER(Title), ngram_size_min=>3, ngram_size_max=>4)`.
It creates both an ordinary index on `Title` and a search index on `Tokens`
with `STORING(Title)`. These are ordinary string predicates, not `SEARCH`
function calls, so both access paths can return the same logical result.

The 10,000-row deterministic dataset contains 1,000 titles starting with
`apple`, of which 10 start with `apple rarexyz`. The other 9,000 start with
`banana`. Every title except those 10 rare titles contains `common`.
The producer submits mutations in batches of 1,000.

| Case | Predicate | Expected rows |
| --- | --- | --- |
| p0 | `STARTS_WITH(Title, 'apple')` | 1,000 |
| p1 | `STARTS_WITH(Title, 'apple rarexyz')` | 10 |
| p2 | `Title LIKE '%rarexyz%'` | 10 |
| p3 | `Title LIKE 'apple%'` | 1,000 |
| p4 | `Title LIKE '%common%'` | 9,990 |

Each predicate is tested without an access hint, with the secondary index
forced, with the search index forced, and with `_BASE_TABLE` forced.
Optimizer versions 4, 5, 6, 8 and 9 are explicit statement hints. Each cell
has separate PLAN and ordinary query requests. Sorted result IDs are hashed
and compared across successful executions. No latency ranking is inferred
from PLAN or the collector wall time. No database-wide `ANALYZE` is issued;
fresh fixture statistics are not deliberately generated.

## Runtime identities

- Emulator 1.5.58: `gcr.io/cloud-spanner-emulator/emulator@sha256:e5bb3535cb3c1e821802531efd9a1d29bbf9804db57a4ae36731764926e3bcb8`,
  inspected as linux/amd64, running with architecture translation on an arm64
  Colima host. GitHub's latest-release endpoint returned v1.5.58, published
  `2026-09-15T06:17:12Z`. The [full response](evidence/fts-index-choice-20260915/release.json)
  is retained. Native arm64 1.5.58 is not verified.
- Omni 2026.r2.1-beta: repository-pinned linux/arm64 image
  `us-docker.pkg.dev/spanner-omni/images/spanner-omni@sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741`.
- Managed Spanner: existing dedicated local verification profile; target
  coordinates are intentionally omitted. The service build is not exposed by
  these requests. This is a dated observation, not an immutable runtime pin.
- Probe source base: `5be6f475fecb82b3e891722a0fea3f224cbab037`; isolated worktree.

## Observed automatic index choice

The completed Omni and managed Spanner runs produced the same following
no-access-hint access choices. Both indexes existed throughout each matrix.
This compares scan targets and predicate roles, not byte-identical full plans.

| Predicate family | v4 | v5 | v6, v8, v9 |
| --- | --- | --- | --- |
| Prefix: p0, p1, p3 | Secondary index, seek | Secondary index, seek | Secondary index, seek |
| Contains: p2, p4 | Secondary index, full scan | Base table, full scan | Search index, search predicate |

Prefix plans contain a `Seek Condition` child and `seekable_key_size=1`.
Contains search plans contain a `Search Predicate` and a residual condition
for exact string matching. Merely seeing `IndexScan` is insufficient to infer
a seek: v4 contains plans scan the ordinary index in full.

Forcing the search index fails at v4 with the error text "query optimizer
version 6 or above", but succeeds at v5, v6, v8 and v9. In particular, v5
forced access and v5 automatic selection are different observations. This
fixture does not establish that v5 never automatically selects a search index.
Nor does it prove a universal secondary-index preference whenever prefix seek
is possible: costs, statistics, projections, ordering and other predicates
were not exhaustively varied.

Each of Omni and managed Spanner produced 95 successful plans and 95
successful ordinary queries out of 100 cells. The five expected failures in each mode were v4 forced-search
queries. Each predicate returned identical row counts and sorted-ID hashes
across successful versions and access paths, also matching the emulator.

## Emulator boundaries

All 100 versioned query cells fail because the emulator rejects the
`OPTIMIZER_VERSION` statement hint. This cannot establish a v4/v5 search
capability boundary on the emulator.

The [default-version producer](evidence/fts-index-choice-20260915/default-probe.go.txt)
removes that statement hint. All 20 ordinary queries succeed, with the same
result IDs for every access choice. Table and both index DDLs succeed.
All 20 `AnalyzeQuery` calls return the Go client's `query plan unavailable`
error. Thus forced search-index query acceptance is verified, but actual
physical index selection is not observable from these emulator captures.

## Emulator 1.5.58 release checks

| Release item | Latest-emulator observation |
| --- | --- |
| Issue 360: column-name case in ALTER/DROP | Wrong-case `MYCOLUMN` is rejected with `NotFound`; original-case `MyColumn` ALTER and DROP succeed. |
| Issue 372: null-filter column absent from index | PostgreSQL index on `baz WHERE bar IS NOT NULL` is rejected with `Unimplemented`; `baz WHERE baz IS NOT NULL` succeeds. |
| Issue 373: UNIQUE index and nullability | Dropping NOT NULL from an indexed column is rejected with `FailedPrecondition`, in GoogleSQL and PostgreSQL. |

Managed Spanner and Omni also reproduced the GoogleSQL wrong-case rejection,
original-case success, and UNIQUE-index nullability rejection. PostgreSQL
release probes were run only on the latest emulator.

These are direct API DDL probes. The PostgreSQL probe does not use PGAdapter,
and tests `ALTER COLUMN bar DROP NOT NULL` separately from the issue's
combined ALTER command. It verifies the changed nullability rule, not
PGAdapter translation. The old emulator image is not rerun; no newly measured
before/after regression claim is made. This is targeted release validation,
not a full survey suite or an exhaustive source audit. Repository runtime pins
remain unchanged.

## Reproduction and evidence

Copy either retained `.go.txt` producer to a `.go` file in checkout scratch,
then build it from the `tools` module with `go build -o <probe> <source.go>`.
For local runs, set `DOCKER_HOST` to the active Docker context socket and
`TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock`. Set `PROBE_IMAGE`
to the digest above and invoke `<probe> emulator`, `<probe> postgres`, or
`<probe> omni`. Container and temporary-database teardown is automatic. Collector exit zero
means the collector finished; expected and unexpected SQL errors are recorded
per request. Use the retained `verify.py` after capture to validate the matrix.

For managed execution, use the existing dedicated profile providing
`SPANNER_PROJECT_ID`, `SPANNER_INSTANCE_ID`, and `SPANNER_DATABASE_ID`, confirm
its target with a read-only preflight, then invoke `<probe> managed`. It
creates uniquely named probe objects and drops only those objects using fresh
bounded cleanup contexts. Never run against an unspecified database.

Retained JSONL files preserve exact generated fixture names, SQL, errors,
plans, row counts and sorted-ID hashes. Managed target coordinates are redacted
by the producer; request IDs are removed during retention. The summary is
mechanically derived, not an execution attestation. The official
[pattern-acceleration text](evidence/fts-index-choice-20260915/pattern-doc.txt)
is also retained. Data generation and predicates are fully defined by the
producer, so raw rows do not need to be retained.

The completed collector processes all exited zero. The retained
`verify.py` validates 440 query records across default emulator, Omni and
managed runs: expected failure classes, expected row counts, cross-runtime
result hashes, and successful cleanup DDL. The version-hinted emulator's
200 unsupported-hint records are retained separately. A final independent
[managed metadata query](evidence/fts-index-choice-20260915/managed-cleanup.txt)
returned zero remaining probe tables and zero remaining probe indexes.
