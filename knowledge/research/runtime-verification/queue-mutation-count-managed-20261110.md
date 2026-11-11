---
type: Research Note
title: Queue mutation counts on managed Spanner, November 10
description: Separately authorized 319-attempt and 43-attempt managed queue runs retain 302 CommitStats observations, including OR IGNORE, source inserts, assertion controls and read-write renewal rejection.
tags: [managed-spanner, queues, mutation-count, dml, runtime-verification]
status: draft
sources:
  - id: primary
    resource: ../../../testdata/mutation-count/managed-queues-20261110.json.gz
    title: Complete sanitized primary response with preceding schema snapshot
  - id: receipt
    resource: evidence/queue-mutation-count-20261110/receipt.json
    title: Authorized single-DB lifecycle, binary/source identity and verified cleanup
  - id: cases
    resource: evidence/queue-mutation-count-20261110/case-summary.json
    title: Derived case inventory and query/API outcomes
  - id: replay
    resource: evidence/queue-mutation-count-20261110/core-summary.json
    title: Core interval and same-input measurement replay with explicit scope gaps
  - id: followup
    resource: ../../../testdata/mutation-count/managed-queues-ignore-20261110.json.gz
    title: Complete sanitized 43-attempt follow-up primary response
  - id: followup-receipt
    resource: evidence/queue-mutation-count-20261110/followup-receipt.json
    title: Separate explicit one-DB authorization, execution identity and verified cleanup
  - id: followup-cases
    resource: evidence/queue-mutation-count-20261110/followup-case-summary.json
    title: Follow-up statuses, counts and DML affected rows
  - id: followup-replay
    resource: evidence/queue-mutation-count-20261110/followup-core-summary.json
    title: Thirty replayable measurements and two successful DELETE assertion parser gaps
  - id: producer
    resource: ../../../tools/mutation-count-lab/managed-probe/queue_matrix.go
    title: Payload, key, DML, composition and cascade case producer
  - id: runtime
    resource: ../../../tools/mutation-count-lab/managed-probe/queue_matrix_runtime.go
    title: Quota, receive, lease, PROFILE and non-Commit API controls
  - id: previous
    resource: mutation-count-managed-20261110.md
    title: Earlier ordinary-write investigation and partial lease boundary
  - id: documents
    resource: evidence/queue-mutation-count-20261110/document-sources.json
    title: Full dkcli document retrieval metadata and content digests
  - id: queues
    resource: https://docs.cloud.google.com/spanner/docs/queues/queues-using
    title: Queue operations and payload declarations
  - id: dml
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax
    title: OR IGNORE, source queries and affected-row assertions
  - id: limits
    resource: https://docs.cloud.google.com/spanner/quotas
    title: Distinct Mutation API, DML statement and byte-limit scopes
---

# Managed queue mutation-count matrix

The first matrix executes at 23:32:39-23:36:43 UTC on November 9, 2026
(08:32-08:36 JST on November 10). It completes **319 transaction attempts**:
270 successful commits with CommitStats and 49 failed controls. Seventeen
queue query calls and three other-API calls have separate records. These are
finite, named scenarios on one managed target, not an exhaustive claim about
all concurrent schedules or future backend versions.

A separately approved one-DB follow-up executes **43 attempts** at
06:35:23-06:36:48 UTC on November 10 (15:35-15:36 JST). It captures 32 successful
CommitStats and 11 failed controls, plus two successful receive streams used
to acquire fresh leases. Across these two independently owned runs there are
**362 attempts and 302 measured commits**. The follow-up database is also
dropped, with NotFound verified; it does not reuse or modify the first run's
primary response or authorization. Both approved lifecycles are consumed.

The user explicitly approved one additional temporary GoogleSQL database.
The producer creates that database once, installs four ordinary tables and
eight queues, captures the schema/index phase, and seeds synthetic data outside
each measured transaction. It uses Go SDK v1.95.1, requests CommitStats and
captures actual CommitRequest mutations through an interceptor. The owned DB
is dropped successfully and a subsequent GetDatabase returns NotFound.
Existing databases and the instance are not changed. The earlier partial
capture remains immutable and independently identified.

The phase's DDL is the submitted fixture text whose initialization operation
completed successfully; it is not a GetDatabaseDdl canonical-text response.
Index and queue-column tuples are observed INFORMATION_SCHEMA reads. The
replay binds those tuples to that submitted DDL under the producer's schema
consistency assertion.

## Coverage

| Axis | Cases |
| --- | --- |
| Logical queue keys | One, two and three columns; 1 KiB STRING key; standalone and one/two-level interleaving |
| Payloads | BYTES, STRING, JSON and PROTO; constructed bodies of length 0, 1 and 8,192; an additional 65,536-byte BYTES payload; JSON null |
| Delivery | Default, explicit past and explicit future timestamps |
| Existence and validation | New/duplicate Send, same/changed payload, missing/NULL keys, NULL/wrong payload and NULL delivery |
| Ack | Present/missing, ignore_not_found false/true, previously acknowledged keys, distinct and repeated keys |
| Batches | Send/Ack of 2 and 10; DML INSERT of 1/2/10 and DELETE matching 0/1/3/10 |
| Delete shapes | Disjoint key predicates, ranges, all rows and interleaved parent cascades/replacements |
| Composition | Same-key Send/Ack orders; ordinary indexed/unindexed table writes plus Send; multiple DML statements; DML plus buffered mutations |
| Quotas | Send 16,000/16,001 and Ack 40,000/40,001 |
| Lease state | Receive/renew, batches, expiration, redelivery, Ack after expiry and renewal after Ack |
| API boundary | Read-write RECEIVE attempts, empty PROFILE, Partitioned DML and per-group BatchWrite outcomes |

The length parameter is the constructed input body, not necessarily serialized
payload bytes: JSON adds object syntax and PROTO adds its wire envelope. Empty
BYTES/STRING and an empty PROTO message differ from SQL NULL. JSON null is a
valid non-NULL JSON payload. The descriptor set is the existing Order fixture;
the raw receipt records its digest.

## Send and acknowledgment counts

| Successful case | Observed mutation_count |
| --- | ---: |
| One new Send, any of the eight queue schemas | 5 |
| N distinct new Sends, N=2 or 10 | 5N |
| DML INSERT of N new messages, N=1/2/10 | 5N |
| Ack one existing message, either ignore_not_found value | 2 |
| Ack N distinct existing messages, N=2/10 | 2N |
| Ack one missing or previously acknowledged message, ignore_not_found=true | 1 |
| Ack one existing and one missing message, ignore_not_found=true | 3 |
| Ack the same existing key twice, second Ack ignoring not-found | 3 |

The tested key-column count, interleaving, delivery timestamp and payload
variants do not change the new-message Send count. This differs from counting
the visible queue columns as ordinary-table cells. INFORMATION_SCHEMA exposes
the logical keys, Payload and implicit NOT NULL TIMESTAMP DeliverTime, without
a complete internal queue write-price model. No public formula for those
internal effects was established, so the default core upper bound stays
unknown. The constants above remain scoped observations.

Duplicate Sends to existing keys return AlreadyExists. Two buffered Sends of
one new key return InvalidArgument. Missing Ack with ignore_not_found=false
returns NotFound; a repeated Ack without ignoring the now-missing row returns
InvalidArgument. Failed controls expose no CommitStats and are excluded from
numerical agreement scores.

## Composition and delete paths

| Successful case | Observed mutation_count |
| --- | ---: |
| Buffered Send then Ack of the same initially absent key | 3 |
| Buffered Ack then Send of the same initially existing key | 7 |
| DML INSERT then DELETE, initially absent key | 7 |
| DML DELETE then INSERT, initially existing key | 7 |
| DML INSERT followed by buffered Ack | 7 |
| New two-column ordinary row plus N Sends, N=1/3 | 2 + 5N |
| The same ordinary row with one secondary index, plus N Sends | 3 + 5N |
| Direct queue DML DELETE of N messages, N=0/1/3/10 | 2N |
| Parent key deletion with N queued children, N=0/1/3/10 | 1 + N |
| Parent replacement writing its two columns with N queued children | 2 + N |

Queue key-list, range and whole-queue DML deletions agree on 2N in these
controls. The parent cascade uses a different accounting path. For the
two-level fixture there is also one ordinary child with one secondary-index
entry; deleting/replacing its parent adds one unit to the corresponding
one-level result. Zero queued children still leaves an existing parent (and,
in the deep fixture, that indexed ordinary child).

Thus a successful operation's independent observed cost need not add to a
buffered same-key transaction total. DML statement boundaries also matter.
The default model preserves uncertainty for these interactions instead of
using a universal per-message multiplier.

## Mutation API limit boundaries

| Commit | Result | CommitStats |
| --- | --- | ---: |
| 16,000 new Sends | OK | 80,000 |
| 16,001 new Sends | InvalidArgument, maximum 80,000 mutations | Absent |
| 40,000 existing-message Acks | OK | 80,000 |
| 40,001 existing-message Acks | InvalidArgument, maximum 80,000 mutations | Absent |

Ack fixtures are seeded in separate 5,000-message commits. These controls
establish the tested queue mutation-limit boundaries, not transaction-byte
headroom, BatchWrite quotas or DML's separate per-statement enforcement.
Subsequent fixture cleanup uses bounded DML statements so the cleanup itself
does not accidentally exceed the mutation limit.

## Lease and statistics boundary

Whole-array STRING parameter binding (`lease_tokens=>@tokens`) renews one or
three fresh leases. Both projected and SELECT-star scalar-inside-array forms
(`lease_tokens=>[@token]`) return InvalidArgument with an invalid-lease-token
message while the receive stream stays open. This agrees with the
[earlier controlled observation](managed-spanner-queues-20260918.md#lease-renewal),
so it is not identified as a newly introduced regression.

Ack after successful receive/renew costs 2 per message. Ack after the captured
expiration, and Ack after redelivery of that same key with a new token, also
cost 2. Renewal immediately after Ack returns a non-NULL new token in this
sample; that observation does not prove the message can be delivered again.
A malformed token returns InvalidArgument.

RECEIVE and RENEWLEASE responses expose no caller CommitStats. Empty RECEIVE
with PROFILE completes with zero rows and query statistics, whose captured
keys include no mutation_count. RECEIVE inside a read-write transaction,
with or without a prospective ordinary write, is rejected because queue
receive queries must be single-use. None of these results establishes zero
internal delivery or lease mutations.

Partitioned DML DELETE succeeds with SDK-reported affected-row lower bounds
of 0 and 3; its API does not expose mutation_count. The retained raw tag
`affected_rows` denotes that estimate, not an exact general-purpose count.
BatchWrite's transport stream completes OK
while **all three groups return Unimplemented and are uncommitted**, with the
queue-not-supported message. Its response schema also has no CommitStats
field. A transport-level OK is not a successful mutation group or a measured
zero count.

## Integration and replay

The root decoder now accepts PROTO's base64 wire envelope as one opaque value;
descriptor-level payload validation stays a service/caller precondition.
The matrix also discovers a supported absent-target INSERT OR IGNORE, and the
core now represents its affected-row uncertainty. INSERT SELECT normalizes to
one cardinality group without analyzing or executing the source query.

Of 270 measured transactions, **268** are representable by the separate
Mutation API or DML entry points. Their schema intervals contain the counts,
and same-schema/input/scope observations narrow them to the measured value.
Most queue intervals intentionally have unknown upper bounds; containment is
not evidence of a precise static predictor. The two combined DML/buffered
mutation cases remain explicit scope gaps. Their cumulative total is not
attached to a partial input. Query effects remain in their own unknown scope.

The captured case labeled `send/invalid-key/2` succeeds because SDK int64(1)
encodes as a protobuf string scalar, which is the valid logical STRING key
"1". The label describes a rejected producer hypothesis, not a server type
validation defect. The other rejected hypothesis is the case labeled
`unsupported/dml-insert-or-ignore`, which succeeds with count 5. Retained
producer labels/expectations are corrected; primary case IDs and binary/source
digests are preserved.

The first receipt's prepared/unverified follow-up entries describe its earlier
checkpoint. The completed follow-up has its own receipt, raw response, case
summary and replay, linked above. Of its 32 measurements, **30** are represented
by the current core and refine to their same-input observed values. Two
successful `DELETE ... ASSERT_ROWS_MODIFIED` statements are rejected by pinned
memefish v0.8.1; their complete SQL and measurements are retained as explicit
parser gaps. The assertion is not stripped to attach stats to different SQL.

## Completed 43-attempt follow-up

| Control | Observed result |
| --- | --- |
| One OR IGNORE row, absent target, with/without DeliverTime | One inserted row; count 5 |
| One OR IGNORE row, existing target, same/changed payload, with/without DeliverTime | Zero inserted rows; count 0 |
| Three distinct OR IGNORE input rows, P existing targets, P=0/1/2/3 | Count `5 * (3-P)`; DML rows `3-P` |
| Repeated key within one OR IGNORE VALUES input, same/different payload | New key inserts once, count 5; existing key inserts none, count 0 |
| OR IGNORE with NULL payload | Absent target: FailedPrecondition and no stats. Existing target: ignored, count 0 |
| INSERT SELECT from 0/1/3 source rows into absent targets | Counts 0/5/15 |
| OR IGNORE SELECT with three source rows and 1/3 existing targets | Counts 10/0, corresponding to 2/0 actual inserts |
| OR IGNORE followed by DELETE, initially absent/present key | Counts 7/2; per-statement DML rows `[1,1]` / `[0,1]` |
| Two OR IGNORE statements for one initially absent key | Count 5; DML rows `[1,0]` |
| DELETE assertion agrees with actual 0/1 matched rows | Success; counts 0/2 |
| DELETE assertion disagrees with actual 0/1 matched rows | OutOfRange, no CommitStats |
| ON CONFLICT DO NOTHING, targeted DO NOTHING, targeted DO UPDATE; absent/present targets | All six InvalidArgument: conflict action is not supported on queues |
| RENEWLEASE inside RW, with/without subsequent buffered Ack | Both InvalidArgument: queue renewal queries must be single-use only |

There are 28 successful insertion-only transactions in this follow-up, including
the two-statement control. All agree with 5 times the actually inserted DML
rows; 12 insert no rows and expose an actual zero statistic. This remains a
finite BYTES-payload/one-key queue observation, not a new public pricing rule.

The new core refinement uses successful DML affected-row assertions: zero rows
produce zero units even on a queue. A singular old-row assertion can also prove
that OR IGNORE ignores an existing target. An aggregate INSERT SELECT accepts
cardinality facts without a singular old image. A missing Ack still has unknown
internal accounting by default (the earlier observed ignored-missing Ack costs
1), and zero-row RECEIVE/RENEWLEASE results do not prove zero internal writes.

For the renewal controls, a fresh receive stream stays open while a separate
RW transaction attempts the whole-array token binding. No malformed token is
used to infer the RW restriction. The API rejects the query before a caller
commit statistic can be obtained. Together with the earlier RW RECEIVE
controls, this closes the tested route for measuring queue TVF work through
caller CommitStats.

The [DDL/DML reference audit](../write-analysis/mutation-count-reference-audit-20261110.md)
separates these runtime results from source-backed automatic-column fixes and
remaining generated-key, expression-index, conflict and compound-update work.

PostgreSQL, other SDKs, concurrent races, TTL background deletion, placement
differences, bytes and latency remain outside this finite GoogleSQL matrix.
No enforcing CI contract, stable public counting formula or published release
is introduced by this investigation.

The full root race suite, workspace CLI consumer, standalone tools, five-module
pre-push gate, OKF bundle gates and Markdown language check pass after the
follow-up integration. The reference audit retains the local validation receipt.
