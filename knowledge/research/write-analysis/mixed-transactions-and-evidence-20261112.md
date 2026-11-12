---
type: Research Note
title: Mixed mutation-count transactions and minimal retained evidence
description: Official-source review, combined DML and buffered Mutation accounting, replay of two managed observations, remaining uncertainties and retirement of redundant logs.
tags: [spanner, mutation-count, dml, queues, evidence, known-unknowns]
status: draft
sources:
  - id: comparison
    resource: https://docs.cloud.google.com/spanner/docs/dml-versus-mutations
    title: DML visibility, execution order and separate mutation limits
  - id: quotas
    resource: https://docs.cloud.google.com/spanner/quotas
    title: Mutation limits, automatic columns and separate commit-byte limit
  - id: stats
    resource: https://docs.cloud.google.com/spanner/docs/commit-statistics
    title: Cumulative transaction statistics and per-statement DML enforcement
  - id: practices
    resource: https://docs.cloud.google.com/spanner/docs/dml-best-practices
    title: DML batching and last_statement incompatibility with mutations
  - id: queue-api
    resource: https://docs.cloud.google.com/spanner/docs/queues/queues-using
    title: Queue renewal parameter example and documented token outcomes
  - id: foreign-keys
    resource: https://docs.cloud.google.com/spanner/docs/foreign-keys/overview
    title: Action-specific base, index and cascading effects
  - id: documents
    resource: ../runtime-verification/evidence/mutation-integration-20261112/document-sources.json
    title: Retrieved full-document identities, hashes and reviewed sections
  - id: implementation
    resource: ../../../mutation_count_transaction.go
    title: Explicit mixed input, component boundaries and conservative relationship closure
  - id: example
    resource: ../../../mutation_count_transaction_example_test.go
    title: Runnable non-executing mixed API example
  - id: earlier-managed
    resource: ../runtime-verification/queue-mutation-count-managed-20261110.md
    title: Original queue matrix, follow-up, lifecycle and measurement boundaries
  - id: replay
    resource: ../runtime-verification/evidence/mutation-integration-20261112/replay-summary.json
    title: Current offline replay, including the two previously unmodeled commits
  - id: retention
    resource: ../../concepts/evidence-lifecycle.md
    title: Minimum retained evidence and artifact retirement policy
  - id: retired
    resource: ../runtime-verification/evidence/mutation-integration-20261112/evidence-retirement.json
    title: Original identities and replacements for 31 retired archives
  - id: validation
    resource: ../runtime-verification/evidence/mutation-integration-20261112/validation.json
    title: Local gate results and source identities, with full logs kept outside tracked evidence
---

# Mixed transactions and evidence retention

The root library and analyzer CLI now accept one explicit transaction containing
ordered DML statements followed by buffered Mutation API writes. This closes
the **input-scope gap** for both retained managed mixed commits. It does not
establish a universal queue formula or resolve cross-stage index coalescing.
No new managed database or service measurement was needed for this integration.

## Documentation review

Full pages were retrieved with `dkcli` on November 12. The
[source record](../runtime-verification/evidence/mutation-integration-20261112/document-sources.json)
lists the canonical URLs, reported update times, content hashes and selected
sections reviewed directly. Search chunks were used to locate pages, not as
the authority. An initial guessed URL returning 404 is not evidence that a
feature is unavailable. Full downloaded pages and ordinary command logs stay
in task scratch; another compressed copy is not added to the bundle.

| Official source | Source-backed conclusion | Implementation consequence |
| --- | --- | --- |
| DML versus mutations | DML executes before buffered mutations, regardless of client buffer-call order; buffered writes are invisible to preceding DML/reads | Two ordered input lists; do not interleave SDK calls or propagate buffered values backward |
| Quotas and CommitStats | Each DML statement has an 80,000-mutation limit; Mutation API writes have a separate cumulative commit limit; transaction CommitStats can exceed 80,000 | Preserve `dml_statements` and `buffered_commit` boundaries; do not classify a combined total against one 80,000 threshold |
| Transactions and DML best practices | DML affects later statements; retries are separate attempts; `last_statement` cannot be combined with Mutation API writes | Facts describe each execution stage. No accumulation of aborted attempts, and no `last_statement` or arbitrary request-options input |
| Foreign-key actions | SET NULL writes the affected referencing columns, while CASCADE deletes rows/ranges and can affect every secondary index and descendants | Keep a conservative transitive relationship check between stages; unrelated target names alone do not establish independence |
| Generated columns, DDL and quota note 10 | Automatic values and their index dependencies can add cells; an old value does not evaluate a new generated/default value | Retain the existing bounds and explicit calibration boundary; no new automatic-column scalar is inferred from this review |
| Search/vector index pages | Indexes are maintained by the service; vector dimensions are constrained | Keep specialized accounting unknown; index maintenance semantics alone do not yield a CommitStats coefficient |

The documented recommendation is to avoid mixing DML and Mutation API in one
transaction. Accepting an existing mixed input for analysis does not recommend
that application pattern. Commit-byte limits, constraint success, locks and
latency remain independent of the mutation-count model.

## Alternatives and chosen model

| Choice | Alternatives considered | Result |
| --- | --- | --- |
| Public input | Continue rejecting mixtures; attach the count to one subset; accept an explicit combined input | Add `MutationCountTransaction` and `AnalyzeTransaction`; preserve the complete identity |
| Static total | Always add both independent reports; always return unknown; add only disconnected table families | Add disconnected stages under the existing conditional model; same-table or transitively related stages retain `[0, unknown]` |
| Old images | Treat all old rows as the transaction's initial snapshot; execute DML locally; accept stage-specific assertions | Facts follow normalized DML rows, then normalized buffered rows. The caller supplies the state immediately before each statement or buffered commit; no expression evaluation or state propagation is invented |
| Quota result | Compare the full total with 80,000; check only DML; retain every documented boundary | Classify each DML statement and the buffered subset independently. Unknown interactions make the combined verdict indeterminate; an observation never refines component counts |
| Evidence | Keep all gz outputs; keep only reproduction code; retain minimal dated observations | Retain inputs, source/runtime identity, counts, relevant failures and check statuses. Retire routine logs and duplicate full reports |

Relationship closure treats foreign keys and interleaving as undirected edges,
including transitive shared families. This intentionally over-approximates
possible interactions. Even distinct keys in the same table remain uncertain;
the analyzer does not solve predicates or child fanout. Per-operation estimates
remain diagnostic standalone estimates when the combined interval is unknown.
None of these preview reports is eligible as a verified quota gate.

The mixed digest includes exact SQL text and both sequence orders. JSON object
layout is canonicalized. A DML-only or Mutation-only observation cannot be
attached to the mixed scope. A successful whole-commit measurement narrows only
the overall `effective` interval; it supplies neither a statement measurement
nor a buffered-commit measurement.

## CLI reproduction

From the repository root:

```sh
go run ./cmd/spanner-analyzer --mode mutation_count \
  --ddl testdata/mutation-count/transaction.sql \
  --mutation-transaction-file testdata/mutation-count/transaction.json \
  --mutation-facts-file testdata/mutation-count/transaction-facts.json \
  --output json
```

The checked-in example supplies one affected DML row followed by a queue Send.
It retains separate component boundaries and an unknown queue upper bound.
All three input sources (`--sql`, `--mutations-file`, and the transaction file)
are mutually exclusive. Unknown transaction fields and unsupported SQL fail
explicitly. Existing facts, index-snapshot and observation files apply unchanged.

## Retained managed replay

| Original commit | Observed whole-commit count | Current treatment |
| --- | ---: | --- |
| DML INSERT into a queue, then buffered Ack of that same message | 7 | Same-table interaction remains unknown; matching whole-transaction observation refines the total to 7 |
| DML INSERT into PlainRows, then buffered Send to Messages | 7 | Disconnected table/queue components retain conditional bounds; matching observation refines the total to 7 |

The original main queue matrix now replays **270 of 270** measured commits,
instead of 268. The separate 43-attempt follow-up still replays **30 of 32**
measurements: its two DELETE assertions remain a pinned-parser gap. The older
summary files remain historical checkpoints. This replay is offline analysis
of the November 10 captures, not a fresh managed-service verification.

Regression tests additionally cover a combined model total above 80,000 while
each component remains below its limit, over-limit DML and buffered components,
same-table/FK/interleave/transitive uncertainty, input identity and subset
observation rejection. Synthetic test observations are not service evidence.
All five module tests, generated-evidence/OKF checks, standalone builds, lint
and vulnerability checks passed for the local implementation; the
[validation record](../runtime-verification/evidence/mutation-integration-20261112/validation.json)
identifies its source boundary. No hosted CI, publication or fresh engine run
is claimed.

## Remaining known unknowns after document review

| Item | What documentation establishes | Still unverified or unimplemented |
| --- | --- | --- |
| Queue scalar-inside-array renewal parameter | The official Go example uses `lease_tokens => [@token1]` | Retained managed calls fail while whole-array binding succeeds. The cause remains unresolved; no new runtime reproduction in this task |
| Renewal immediately after Ack | The queue guide describes a NULL replacement token for unrenewable leases, including already-acknowledged messages | The retained non-NULL result differs in the tested sample. Whether that message can actually be redelivered remains untested; token presence alone is insufficient |
| Malformed lease tokens | The queue guide describes omission of an unparseable token's row | The retained malformed-token call returned InvalidArgument. The document/result mismatch remains, without an inferred cause |
| RECEIVE/RENEWLEASE mutation counts | Query/lease operations are distinct from caller commits | Caller CommitStats remain unavailable; the tested RW route is rejected. Missing statistics are not zero |
| Automatic keys and ON UPDATE | Generated/default values and dependency-triggered writes are described | Detailed runtime prices and supported input normalization still need separate controls. Explicit old rows alone cannot determine automatic new values |
| SET NULL and multilevel cascades | The FK guide explains target-column writes, index removal and downstream effects | Data-dependent fanout, index membership and overlapping actions still need dedicated assertions/measurements before finite general bounds |
| Predicate, expression, search and vector indexes | Storage/maintenance behavior and schema restrictions are described | Predicate/expression adapters and specialized mutation accounting are incomplete; no public constant coefficient was established |
| Vector longer than vector_length | DDL defines a fixed dimension and requires matching array dimensions | The expected rejection is source-backed; the earlier runtime check covered a short vector, not a long one |
| Historical scorer defaults | Ranked-search docs distinguish before/during/after April 2026 cohorts and future migration | New databases cannot establish an old cohort's current migration state or an exact rollout boundary |
| Omni r5.1 Remote Function PLAN | Managed Remote Functions are documented; the Omni differences page does not explain the retained r5-only PLAN Internal failure | Last runtime reproduction remains November 8. No new root cause, fix, or fresh execution is claimed |
| DELETE ASSERT_ROWS_MODIFIED and analyzer catalog functions | Service behavior was already measured; these are known local support gaps | The pinned parser still rejects the two retained DELETE assertions. SECURE_CONTEXT and SPANNER_SYS.DISTRIBUTION_AGG remain separate catalog work |

The vector dimension conclusion is inferred from the explicit fixed-dimension
contract, not a new runtime result. The scorer source's `version 2 (2025-08)`
algorithm label is not the April 2026 default-rollout date. This review keeps
those dates and evidence boundaries separate.

## Evidence retirement

Two November 12 mutation-count evidence directories previously contained full
reports, repeated successful logs and intermediate failed-attempt outputs.
The [retirement manifest](../runtime-verification/evidence/mutation-integration-20261112/evidence-retirement.json)
records 31 removed archives: **333,928 compressed bytes / 6,534,492 decoded
bytes**. Git history is unchanged. Each original is recoverable from its
recorded commit and path.

Per-scenario counts, image/platform identities, observation timestamps,
producer hashes, cleanup receipts, check exit statuses, the failed-attempt
explanation, physical FK access-path metadata and contract outcomes remain.
`report-summary.json` replaces complete graph-heavy reports; `counts.json`
already retains the measured accounting projections. Existing receipts now
point to the retention disposition rather than absent log paths.

Compressed managed/local wire fixtures remain because offline regression tests
consume their precise inputs and observed outcomes. Older official-page
snapshots and other investigations are not bulk-deleted. Their consumers and
dated claims must be reviewed before separate compaction. Reproduction commands
do not by themselves preserve a past managed response, but routine successful
test logs add little once the inputs, command, status and relevant result are
retained.
