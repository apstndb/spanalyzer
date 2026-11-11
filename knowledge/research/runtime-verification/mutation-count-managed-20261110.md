---
type: Research Note
title: Managed mutation-count integration, November 10
description: 197 managed CommitStats observations, index-sensitive partial updates and explicit queue-flow failures inform the interval-first preview API.
tags: [managed-spanner, mutation-count, dml, queues, runtime-verification]
status: draft
sources:
  - id: primary
    resource: ../../../testdata/mutation-count/managed-20261110.json.gz
    title: Sanitized primary managed response and per-phase schema snapshots
  - id: receipt
    resource: evidence/mutation-count-managed-20261110/receipt.json
    title: One-database authorization, producer identity, recovery and cleanup
  - id: replay
    resource: evidence/mutation-count-managed-20261110/summary.json
    title: Derived core-model comparison with the measured cases
  - id: producer
    resource: ../../../tools/mutation-count-lab/managed-probe/
    title: Explicit managed lifecycle and existing-owned-database recovery producer
  - id: core
    resource: ../../../mutation_count.go
    title: Preview interval, old-row and input-qualified observation API
  - id: local
    resource: mutation-count-20261110.md
    title: Earlier Omni LTS.2 and Emulator observations kept independently
  - id: queues
    resource: https://docs.cloud.google.com/spanner/docs/queues/queues-using
    title: Send, acknowledgment and lease operations
  - id: limits
    resource: https://docs.cloud.google.com/spanner/quotas
    title: Mutation API, DML statement and byte-limit boundaries
---

# Managed write-accounting integration

This is a single-database observation on November 10, 2026 JST. Managed
Spanner exposes no backend build identifier here. Execution windows, schema
phases and captured input identify the observation; this does not establish a
fleet-wide rollout or a feature introduction date.

## Method and recovery boundary

The user explicitly authorized one temporary GoogleSQL database on the
existing dedicated instance. A private exclusive journal binds ownership to
that one database. The producer never creates or changes an instance.
Synthetic seed writes are outside the measured transactions, which request
CommitStats and capture only their wire Mutation objects.

The initial attempt did not finalize a result. While it was running, the
owned database was READY with 16 base tables, and a later metadata read reached
its deadline. No definitive server-side cause was established. The producer
also lacked per-case deadlines and incremental result persistence.

That process was stopped. Recovery reopened the same owned database, reset
only fixture data in foreign-key dependency order and added per-case deadlines
and an append-only case stream. It made no second CreateDatabase call. Two
large-literal DML stress controls were explicitly retained as NOT_RUN in the
recovery; no runtime conclusion is drawn from them. The finalized recovery
capture is the primary numerical evidence. An interrupted process is not
treated as completed verification.

Schema/index snapshots precede their operation phases. Queue creation occurs
after the ordinary-table controls, so later queue metadata is not attributed
to earlier commits. Receive/renew queries retain status and row counts without
lease tokens or a fabricated zero CommitStats.

The recovery capture is **partial**: 211 records contain 209 attempted writes
and two NOT_RUN stress controls. Of those attempts, 197 succeed with
CommitStats and 12 return errors. RECEIVE returns one row. The subsequent
RENEWLEASE query returns InvalidArgument, whose detailed message was not
captured by that producer, so the final received-message ACK is not attempted.
Cleanup still completes: DropDatabase is OK and GetDatabase returns NotFound.
The retained producer adds a minimal lease-query comparison and token-redacted
error capture; preparing that reproducer is not evidence of another execution.

## Managed observations

Ordinary fixtures use `(Shard, Id, Active, Note)` with primary key
`(Shard, Id)`. A stored index is on Active with `STORING(Note)`.

| Measured operation | Managed mutation_count |
| --- | ---: |
| Full insert, no / one / two ordinary indexes | 4 / 5 / 6 |
| Change a non-indexed Note, Active-only index | 3 |
| Change Active, ordinary / stored Active index | 5 / 6 |
| Write unchanged Active, ordinary / stored Active index | 3 / 3 |
| Write changed / unchanged stored Note | 5 / 5 |
| Partial upsert of unchanged Active / stored Note | 3 / 5 |
| Existing full upsert / replace, two indexes and changed values | 8 / 8 |
| Delete one three-row range, no / two ordinary / one stored index | 1 / 7 / 4 |
| Replace a parent with three indexed interleaved children | 4 |
| Two buffered updates of one indexed key, a to b to a | 6 |
| The same transitions in two DML statements | 10 |

The [local comparison](evidence/mutation-count-managed-20261110/local-comparison.json)
finds 85 shared case IDs with the earlier Omni corpus. Status/count pairs match
for 84; the difference is the mutation-limit control. The managed Mutation API
commit containing 20,001 four-column inserts is rejected with InvalidArgument
and the maximum 80,000 message. Omni reports 80,004 and succeeds on that same
control; Emulator also succeeds but exposes no CommitStats. This establishes
the tested enforcement difference, without inferring DML statement limits or
byte-limit behavior from it.

Automatic-column observations also match those shared Omni controls, including
PK-only Mutation API inserts costing 5 with a default, 5 with a stored generated
column and 6 with both. The default model keeps these costs conditional rather
than turning this agreement into a universal service calibration. The two
large-literal DML boundary controls remain unverified.

## Queue accounting and supported operations

| Queue case | Managed result |
| --- | --- |
| Send with BYTES / STRING / JSON payload, default or explicit DeliverTime | OK, 5 in all six cases |
| Ack existing message | OK, 2 |
| Ack missing message | NotFound, no CommitStats |
| DML INSERT with explicit or omitted DeliverTime | OK, 5 |
| DML DELETE | OK, 2 |
| Ordinary wire insert/update/upsert/replace/delete | NotFound: Table not found |
| DML UPDATE / INSERT OR UPDATE | InvalidArgument |
| DML INSERT OR REPLACE | Unimplemented |
| Parent delete with one interleaved queued message | OK, 2 |

The five queue schemas expose implicit NOT NULL TIMESTAMP DeliverTime columns.
No ordinary queue index appears in the captured index snapshots. That absence
does not identify internal bookkeeping or supply a general price formula.
The root model now rejects the observed unsupported wire and DML operation
kinds explicitly. Successful Send/Ack and INSERT/DELETE retain an unknown
baseline upper bound; 5 and 2 remain observations of these inputs.

RECEIVE succeeds, while the projected scalar-parameter RENEWLEASE query fails.
The [earlier queue investigation](managed-spanner-queues-20260918.md#lease-renewal)
also records scalar-array renewal errors;
this capture alone does not identify a new regression or its cause. No failed
query is interpreted as a successful zero-row response or zero internal writes.

## Replay and validation scope

`go test -run '^TestMutationCountManagedReplay$' .` reads the compressed primary
capture without cloud access. It checks the exact partial-completion boundary
and verified cleanup, then analyzes each successful commit with its preceding
DDL/index phase. Synthetic seed code supplies old images; DML affected-row
counts and range/index cardinalities are supplied facts, not schema-only
discoveries. They never derive from the measured mutation_count.

All 197 cardinality-qualified baseline intervals contain their measurements;
185 have finite upper bounds. Old-row refinements match an exact count in 161
cases. All 197 input/schema/scope-qualified measurements intersect the model
and narrow the effective result to the observed count. There are no model
disagreements in this corpus. This is a replay of these cases, not held-out
accuracy, automatic old-row fetching or proof about future executions.

## Integration boundary

The [later queue matrix](queue-mutation-count-managed-20261110.md) completes
319 transaction attempts in a separately authorized database. It confirms
whole-array lease renewal and Ack afterward, adds payload/key/batch/cascade
controls and tests exact Send/Ack mutation-limit boundaries. The earlier
capture above remains partial; the later evidence does not relabel it.

The root API and analyzer CLI preserve schema bounds, old-row assertions and
input-qualified measurements separately. The generator exposes per-method
mutation-count projections in structured explain output. Ordinary query
analysis, SDK generation and PLAN collection do not gain write side effects.

Queue bookkeeping is not advertised as a complete ordinary index-price model.
The baseline keeps its upper bound unknown, and a supplied measurement can
identify the count of that input. RECEIVE/RENEWLEASE expose no caller Commit
response. Count units, bytes, latency, lock costs and quota enforcement remain
separate properties. The observation does not add an enforcing CI contract or
freeze the mutable preview API.
