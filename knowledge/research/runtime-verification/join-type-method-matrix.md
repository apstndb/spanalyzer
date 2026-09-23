---
type: Research Note
title: "Join type and method matrix"
description: "Per-cell SQL, raw plans and result evidence for join types and methods, including distributed apply and full-outer decomposition."
tags: [spanner, omni, joins, optimizer, query-plan]
status: draft
sources:
  - id: current-cells
    resource: evidence/join-matrix-20260916/audit-summary.json
    title: "Per-cell SQL, raw-record locations, outcomes and operator metadata"
  - id: producer
    resource: evidence/join-matrix-20260916/probe.go.txt
    title: "Exact executed Go producer with DDL and deterministic seed data"
  - id: verification
    resource: evidence/join-matrix-20260916/verification.txt
    title: "Independent evidence completeness and result checks"
  - id: historical-plans
    resource: ../../../research/spanner-query-plan-shape/COMPACT_TREE_METADATA_OBSERVATIONS.md
    title: "Historical join and subquery join hint matrices"
  - id: historical-full-outer
    resource: ../../../research/spanner-query-plan-shape/UNOBSERVED_PLAN_PROBE_MATRIX_2026-07-10.md
    title: "Historical full outer join observations"
---

# Join type and method matrix

## Evidence scope

The historical table was audited and replaced with fresh evidence on
2026-09-16 JST (2026-09-15 UTC). This is a measured matrix for one explicitly
specified fixture, not a universal promise that every query supports every
hint or has the same physical plan.

Each runtime has **56 cells**: eight SQL forms times five methods at optimizer
v9, plus FULL OUTER Hash/Merge at v1-v8. Each cell has separate PLAN and
ordinary execution requests. The eight forms are INNER, LEFT, RIGHT, FULL,
IN, EXISTS, NOT IN and NOT EXISTS; equivalent-looking forms are tested
separately before grouping them in the display table.

- Omni: 2026.r2.1-beta, observed linux/arm64, pinned image
  `us-docker.pkg.dev/spanner-omni/images/spanner-omni@sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741`.
- Managed Spanner: dedicated verification profile, with destination coordinates
  omitted; service build identity is not available. Every query explicitly
  selects its optimizer version.
- Producer source and build identity: [receipt](evidence/join-matrix-20260916/receipt.json).
- Emulator is excluded from plan verification. Its role is schema, constraints,
  metadata and execution semantics; it does not return execution plans.

## Observed operators at optimizer v9

All 112 cells across Omni and managed Spanner succeeded in both PLAN and
ordinary execution. The displayed operator names and join-type metadata agree
between runtimes; this does not assert byte-identical full plans. The display
summarizes those fresh per-cell captures. Parentheses in Hash and
Merge cells contain physical `join_type` metadata. SQL LEFT/RIGHT and physical
BUILD/PROBE are different axes; these are the observed build orientations.
`PBH` below abbreviates the literal operator-name prefix
`Push Broadcast Hash Join`.

| SQL or logical join type | Hash | Apply | Distributed Apply | Merge | Push Broadcast Hash |
| --- | --- | --- | --- | --- | --- |
| INNER | Hash Join (`INNER`) | Cross Apply | Distributed Cross Apply | Merge Join (`INNER`) | PBH |
| LEFT OUTER | Hash Join (`BUILD_OUTER`) | Outer Apply | Distributed Outer Apply | Merge Join (`LEFT_OUTER`) | PBH Outer Apply |
| RIGHT OUTER [1] | Hash Join (`PROBE_OUTER`) | Outer Apply | Distributed Outer Apply | Merge Join (`LEFT_OUTER`) | PBH Outer Apply |
| FULL OUTER [2] | Hash Join (`BUILD_PROBE_OUTER`) | Union All of Outer Apply + Anti-Semi Apply | Union All of Distributed Outer Apply + Distributed Anti Semi Apply | Merge Join (`FULL_OUTER`) | Union All of PBH Outer Apply + PBH Anti Semi Apply |
| SEMI (`IN` / `EXISTS`) | Hash Join (`BUILD_SEMI`) | Semi Apply | Distributed Semi Apply | Merge Join (`LEFT_SEMI`) | PBH Semi Apply |
| ANTI-SEMI (`NOT IN` / `NOT EXISTS`) [3] | Hash Join (`BUILD_ANTI_SEMI`) | Anti-Semi Apply | Distributed Anti Semi Apply | Merge Join (`LEFT_ANTI_SEMI`) | PBH Anti Semi Apply |

1. RIGHT OUTER is preserved: both unmatched right rows appear in execution
   results. Merge reverses the physical inputs (SQL right table on physical
   Left) and uses `LEFT_OUTER`. Apply and PBH use outer forms too. This replaces
   the old interleaved fixture's misleading INNER-simplified RIGHT row.
2. FULL OUTER Apply-family cells are successful decompositions, not a single
   dedicated full-outer Apply operator. Both unmatched sides survive and the
   matching rows are not duplicated by the anti-semi branch.
3. Matching operator families do not mean matching NULL semantics. The right
   input contains NULL: NOT IN returns no rows, while NOT EXISTS returns the
   unmatched and NULL-key left rows. Both results are explicitly checked.
   The table summarizes join nodes, not all scalar conditions in each plan.

## Apply and distribution

Distributed Apply is a separate display column, not a separate `JOIN_METHOD`
value. This audit uses explicit controls:

- Apply: `JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE`.
- Distributed Apply: `JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE`.
- Other methods: `JOIN_METHOD=HASH_JOIN`, `MERGE_JOIN`, or
  `PUSH_BROADCAST_HASH_JOIN`.

Direct joins use join-level hints; IN/EXISTS/NOT IN/NOT EXISTS use statement
hints. The exact placement is retained in every cell's SQL.

| Logical family | Ordinary operator | Distributed operator | Apply inside the observed distributed branch |
| --- | --- | --- | --- |
| Inner | Cross Apply | Distributed Cross Apply | Cross Apply |
| Left/right outer | Outer Apply | Distributed Outer Apply | Cross Apply |
| Semi | Semi Apply | Distributed Semi Apply | Semi Apply |
| Anti-semi | Anti-Semi Apply | Distributed Anti Semi Apply | Semi Apply |

Read the wrapper and its internal operators together. Internal Cross Apply
in Distributed Outer Apply does not indicate outer-join elimination; internal
Semi Apply in Distributed Anti Semi Apply does not change the overall
anti-semi semantics. Names preserve the actual hyphenation differences.
A non-Distributed Apply can itself have distributed input or map subplans;
its name does not imply that the whole query runs on one server. Batch-mode
behavior here is fixture-scoped, not a guarantee for arbitrary SQL.

## FULL OUTER optimizer-version boundary

| Method | v1-v6 | v7-v9 |
| --- | --- | --- |
| Hash | Union All of Hash Join (`BUILD_OUTER`) and Hash Join (`BUILD_ANTI_SEMI`) | Hash Join (`BUILD_PROBE_OUTER`) |
| Merge | Merge Join (`FULL_OUTER`) | Merge Join (`FULL_OUTER`) |

Every version returns the same nine-row multiset. Apply, Distributed Apply
and PBH FULL OUTER are tested only at v9 in this audit.

## Fixture and independent result oracle

Two independent tables have `Id INT64 NOT NULL` as primary key and nullable
`K INT64`, plus an ordinary index on `K`. There are no foreign keys or
interleaving relationships. Inputs are:

| Side | `(Id, K)` rows |
| --- | --- |
| Left | `(1,10), (2,20), (3,20), (4,30), (5,NULL)` |
| Right | `(101,10), (102,20), (103,20), (104,40), (105,NULL)` |

The independent Python verifier enumerates matching pairs under SQL equality,
adds unmatched rows as appropriate, and compares complete result **multisets**.
It does not derive expected results from another SQL plan. Expected counts:
INNER 5; LEFT 7; RIGHT 7; FULL 9; IN and EXISTS 3 each; NOT IN 0;
NOT EXISTS 2. Duplicate matching keys exercise the four combinations at K=20.
SQL returns plain ID columns; nullable values are serialized in the client.

## Completeness and provenance

[Cell inventory](evidence/join-matrix-20260916/audit-summary.json) provides
exact SQL, requested version/method, outcomes, expected and actual row counts,
operator metadata, and one-based PLAN/execution record locations in:

- [Omni JSONL](evidence/join-matrix-20260916/omni.jsonl)
- [Managed JSONL](evidence/join-matrix-20260916/managed.jsonl)

The [verifier](evidence/join-matrix-20260916/verify.py) requires the exact
56-label set for each runtime, both request modes per label, successful
requests, expected operator families/metadata, complete result multisets,
and successful cleanup DDL. It also checks the relational Map branch inside each
Distributed Apply and the reversed physical inputs of RIGHT Merge. Missing or failed cells fail verification.
It also checks the FULL Hash version boundary rather than assuming historical
results transfer to the current image. Full raw child links are retained to
inspect wrapper placement and input orientation.

The [receipt](evidence/join-matrix-20260916/receipt.json) records collector
exit codes and producer hashes. Evidence SHA-256 values are indexed by the
parent [artifact manifest](evidence/artifact-sha256.json). These are reproducible
observations and integrity checks, not an execution attestation.
Managed destination values are redacted by the producer, and request IDs are
removed during retention. A separate [cleanup check](evidence/join-matrix-20260916/managed-cleanup.txt)
checks that probe tables and indexes from both attempts are absent.

An initial producer used `TO_JSON_STRING(STRUCT(...))`, which was rejected
before JOIN planning on both runtimes. Those attempts are excluded from the
matrix; their [setup-failure record](evidence/join-matrix-20260916/initial-attempt.json)
retains the error and cleanup status. The corrected producer returns plain
columns. No JOIN support conclusion is drawn from that unrelated rejection.

## Reproduction

Copy `probe.go.txt` to checkout scratch as `probe.go`, then build it from
`tools` with `go build -o <binary> <source.go>` using the repository toolchain.
For Omni, set `PROBE_IMAGE` to the pinned digest above, configure the Docker
socket and `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock`, then
run `<binary> omni`. For managed, use the dedicated existing profile variables
`SPANNER_PROJECT_ID`, `SPANNER_INSTANCE_ID`, and `SPANNER_DATABASE_ID`, and run
`<binary> managed` only against the authorized verification database. The
producer creates uniquely named fixture objects and drops them with fresh
bounded cleanup contexts. It does not change database-wide optimizer options.

The collector records SQL errors and can exit zero with rejected queries;
collector status alone is not a passing test. Capture each runtime as `omni.jsonl` / `managed.jsonl` beside the retained
`summarize.py`, run `python3 summarize.py` to rebuild the per-cell inventory,
then run `python3 verify.py`. The final independent cleanup check returned
zero remaining tables and zero remaining indexes for both managed attempts. Historical records remain linked for
provenance, but no longer supply otherwise-unverified current matrix cells.
