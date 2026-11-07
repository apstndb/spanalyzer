---
type: Research Note
title: Managed Spanner revalidation and documentation audit, November 8
description: Current managed metadata and behavior, all three new mirror commits, and validation after adopting spanemuboost v0.4.10.
tags: [managed-spanner, documentation-diff, search, queues, gql, runtime-verification]
status: draft
sources:
  - id: mirror
    resource: https://github.com/apstndb/spanner-docs-mirror/compare/293fe78b6da4b70bfe8f693161c5fc784f0364a8...fa8645afd32f27de73bd1fca7c29e89fbc027245
    title: Complete documentation-mirror comparison from the previous managed investigation
  - id: audit
    resource: evidence/managed-recheck-20261108/documentation-audit.json
    title: Commit-by-commit and document-by-document audit with source hashes
  - id: managed-readonly
    resource: evidence/managed-recheck-20261108/readonly.jsonl
    title: Current read-only SQL and both complete metadata surfaces
  - id: assertions
    resource: evidence/managed-recheck-20261108/assertions.json
    title: Bounded outcome assertions and fixture lifecycle checks
  - id: google-fixture
    resource: evidence/managed-recheck-20261108/google.jsonl
    title: GoogleSQL fixture outcomes
  - id: pg-fixture
    resource: evidence/managed-recheck-20261108/pg.jsonl
    title: PostgreSQL fixture outcomes
  - id: google-repair
    resource: evidence/managed-recheck-20261108/repair/google.jsonl
    title: Corrected GoogleSQL default-switch, queue and read-write sequence controls
  - id: pg-repair
    resource: evidence/managed-recheck-20261108/repair/pg.jsonl
    title: Full-signature PostgreSQL receivers and TEXT placement-key positive control
  - id: finalization
    resource: evidence/managed-recheck-20261108/request-id-redaction.json
    title: Explicit decoded-JSON error-field redaction and integrity mapping
  - id: pagination
    resource: https://docs.cloud.google.com/spanner/docs/full-text-search/paginate-search-results
    title: Ranked search pagination and scorer version state
  - id: queues
    resource: https://docs.cloud.google.com/spanner/docs/queues/queues-using
    title: Queue batch receive defaults
  - id: graph
    resource: https://docs.cloud.google.com/spanner/docs/graph/manage-schemaless-data
    title: Direct dynamic property comparisons, grouping and ordering
  - id: views
    resource: https://docs.cloud.google.com/spanner/docs/graph/graph-with-views-how-to
    title: Views calling remote functions and volatile column graph options
  - id: sequences
    resource: https://docs.cloud.google.com/spanner/docs/sequence-tasks
    title: Sequences with placement and interleaved tables
  - id: jdbc
    resource: https://docs.cloud.google.com/spanner/docs/pg-jdbc-connect
    title: PostgreSQL JDBC and PGAdapter dependency example
  - id: maven
    resource: https://repo.maven.apache.org/maven2/org/postgresql/postgresql/maven-metadata.xml
    title: Primary PostgreSQL JDBC artifact version metadata
  - id: dependency
    resource: https://github.com/apstndb/spanemuboost/releases/tag/v0.4.10
    title: Adopted spanemuboost release
  - id: local-validation
    resource: evidence/managed-recheck-20261108/validation/commands.json
    title: Five-module and pinned local runtime validation commands and exits
---

# Managed revalidation on November 8, 2026

This investigation compares the November 5 managed baseline with a new
observation on November 8 JST. Managed Spanner has no exposed backend build
identifier in these probes; the retained execution timestamps identify the
observation. The existing dedicated profile supplies the connection. Project,
instance, database and request identifiers are redacted from retained output.

The code baseline is `04d7019616ef4728dd72c5db9ac163fb4c56506a` with the
retained feature and runtime changes. Go is 1.26.10 and the Spanner SDK is
v1.95.1. The three container-consuming modules adopt spanemuboost `v0.4.10`.
The dependency now defaults to Omni r5, but this repository explicitly keeps
`2026.r4-lts.2`; `2026.r5.1` remains a selectable alternative. The analyzer's
public `v1alpha` and plan contract normalization versions remain previews.

## Every new mirror commit

The previous source boundary is `293fe78b6da4b70bfe8f693161c5fc784f0364a8`.
The refreshed head is `fa8645afd32f27de73bd1fca7c29e89fbc027245`.
All three intervening commits were inspected individually, together with the
full before/after bodies of all ten changed documentation paths. The current
official bodies were fetched separately rather than relying on search summaries.

| Commit and UTC date | Changes | Verification disposition |
| --- | --- | --- |
| `d3a69819bdad28dd976c4fb2bf64b7b01f98ab59`, November 5 | `metadata.yaml` only | Synchronization bookkeeping, no product behavior claim. |
| `a14b9ad57306e22b369848c4839a430239dc02ce`, November 6 | Eight documentation paths plus URL/synchronization metadata | Queues, graph release-note additions, placement sequences, metrics and client dependency examples are accounted for below. |
| `fa8645afd32f27de73bd1fca7c29e89fbc027245`, November 7 | Ranked-search pagination and ranked-search warning, plus bookkeeping | Parameterized scorer pinning and offset pages are new fixture cases. |

The two graph bullets were added under **October 30**, not a new November 6
release date. They describe remote functions in graph input views and direct
JSON dynamic-property comparisons, grouping and ordering. A later mirror
commit does not establish a later service rollout.

The metrics document changes four queue metric launch labels from BETA to GA:
`active_receive_tvf_queries`, `buffered_in_flight_messages`,
`buffered_ready_messages` and `message_payload_sizes`. It also changes the
documented visibility delay from 150 to 210 seconds for
`query_stat/total/cpu_time` and `read_stat/total/execution_count`. These are
documentation observations; metric ingestion latency was not measured.

The Java examples move from client 6.125.0 to 6.126.0, and PGAdapter examples
move from 0.56.1 to 0.56.2. The JDBC connection page also assigns `0.56.2` to
`org.postgresql:postgresql`. Maven Central has no such driver version. The
earlier example already used PGAdapter-style numbering for that driver, so
this is a persistent documentation defect, not a new SQL engine regression.
No upstream report is filed by this investigation.

## Managed metadata and existing behavior

The 135 read-only steps satisfy 198 bounded assertions. Query status codes
match the prior observation. INFORMATION_SCHEMA remains 48 tables / 310
columns; SPANNER_SYS remains 52 / 624. Names, raw types and ordinals agree
with the committed November 5 captures, and all three SYS metadata reads agree.
The current local drift and canonical DDL tests pass. Canonical comparison is
against the existing empty user schema, not proof that every DDL family
round-trips.

The read-only coverage includes pipe operators, grouping aliases, JSON
comparison/grouping, distinctness, HLL/approximate counts, the two batch
operation tables, and lock/split/query statistics. `RAND()` still returns
`Unimplemented`. The zero-row `SELECT *` probe against
`LOCK_STATS_TOTAL_10MINUTE` still returns `Unimplemented`; successful bounded
projections are retained separately. An advertised table is not a promise
that every projection or execution path succeeds.

Distribution-percentile checks retain both positive and negative controls.
The function accepts individual distribution STRUCT values and the service's
encoded JSON strings; passing an entire distribution ARRAY is rejected.
The synthetic/expanded histogram negative controls remain rejected. These
outcomes are not counted as unexpected integration successes or failures.

## Owned fixture results

The initial GoogleSQL and PostgreSQL fixtures record all 201 planned steps:
186 are attempted and 15 are blocked by a prerequisite. Both owned databases are
deleted and independently return `NotFound`. A separately authorized repair
pair then attempts all 60 corrected/control steps; those two databases also
return `NotFound` after deletion. Across all phases, 471 bounded classification
assertions pass. There are 396 case records, of which 381 are attempted and the
original 15 remain blocked. This is not a claim that every probe succeeds.

| Scope | Observed result |
| --- | --- |
| Ranked search, both dialects | Explicit versions 1 and 2 and parameterized two-row offset pages work. Concatenating the three pages reproduces the complete six-row ranking exactly once, including primary-key ordering of equal scores. |
| Scorer defaults | GoogleSQL accepts `score_version=1` and `2`. Under each default, unpinned six-row results match that version, while query-pinned versions 1 and 2 independently match their original full results. PostgreSQL still rejects the documented `spanner.score_version` database option. |
| Queue batches, GoogleSQL | Omitted and NULL `max_batch_size` both produce batches of at most 30, including an observed 30-row batch; explicit 20 produces two 20-row batches. All 40 messages arrive once. A batch may be smaller than its maximum. |
| Queue batches, PostgreSQL | With the complete three-argument signature, NULL produces 30/10 batches and explicit 20 produces 20/20; all 40 messages arrive once. Omitting `max_batch_size` still fails function resolution even when `priority` and `max_duration` are supplied. |
| Dynamic graph properties | Numeric and boolean literal filters, pattern filters, grouping, ordering, DISTINCT and property-to-property equality succeed. A numeric property does not match a string literal. A bare JSON boolean WHERE condition is rejected. |
| GQL and views | Column propagation modes and ordered top-two controls agree with the prior behavior. All three volatile-view option modes accept `GENERATE_UUID()` properties and return new UUIDs on later reads. The explicit false option still permits this local volatile function. |
| Search and statistics | JSON equality/IN/whole-document predicates return the same rows using automatic, forced search-index and forced base-table plans. All six PostgreSQL statistics projections execute, while the STRUCT-column negative control is rejected. |
| Referential action | Deleting the parent sets the child foreign key to SQL NULL in both dialects. |
| Placement sequences | GoogleSQL placement-key parent and interleaved sequence-default child DDL succeeds. PostgreSQL rejects the new example's bounded `varchar(1024)`, while the TEXT parent and interleaved child positive control succeeds. Separate ordinary tables generate positive sequence-default keys through read-write INSERTs in both dialects. |

The PostgreSQL placement example is a new documentation defect candidate.
It is distinct from support for sequences in ordinary PostgreSQL tables. The
generic queue omission guidance also needs a dialect qualification: the
observed PostgreSQL receiver requires explicit arguments, with NULL selecting
the default batch size. The complete working receiver shape is:

```sql
SELECT id, spanner_last_batch_message
FROM spanner.receive_batchnull(
  max_batch_size => NULL, priority => NULL, max_duration => '5s');
```

Three initial probe mistakes are retained explicitly. The added pagination
default-switch cases mistakenly use `default_search_scorer_version`, rather
than the documented `score_version`, so their six dependent reads per dialect
do not execute. The initial PostgreSQL queue calls omit `priority`, leaving
the full-signature positive control untested. A standalone sequence-allocation
SELECT uses a read-only transaction and is rejected because allocation
requires a read-write transaction. These are harness limitations, not evidence
of unsupported scorer pinning, queues or sequence allocation. The corrected
minimal fixtures now supply the missing positive controls above. PostgreSQL's
remaining omitted-argument rejection is isolated from the original missing
`priority` error. The fixed search corpus changes scores but retains the same
ranking across versions; no actual scorer rollout or changed-ranking event is
reproduced.

## Runtime and evidence boundaries

The user first authorized two fresh database lifecycles, one per dialect, and
then separately authorized the additional two-database repair pair. All four
created databases are confirmed absent.
Only databases whose creation completes successfully in this producer may be
deleted. Exclusive retained start markers prevent accidental extra lifecycles
on retry. The base database and any older interrupted fixtures are
outside the mutation scope.

The pagination checks use a fixed data set. They do not simulate a managed
automatic scorer rollout or concurrent writes. Offset pages can still skip or
repeat rows under concurrent writes. No real remote endpoint is invoked or
deployed; local volatile-column views check the graph option boundary only.
The placement-sequence cases check DDL and positive sequence generation, not
multi-partition routing, insertion latency or a UUID performance comparison.

Local container executions use native `linux/arm64` manifests. The earlier
[runtime upgrade investigation](runtime-upgrades-20261108.md) retains the
initial alpha.1 executions and exact image identities. Its results are not
silently relabeled as executions of the later dependency release.
