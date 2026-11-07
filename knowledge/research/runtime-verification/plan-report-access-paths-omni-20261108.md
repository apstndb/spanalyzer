---
type: Research Note
title: Scan-local PLAN contracts and report collection on Spanner Omni, November 8
description: Fixed-image verification of seek and residual evidence, optional targets, partial errors, structured parameters, annotations and non-executing DML PLAN acquisition.
tags: [spanner-omni, query-plan, plancontract, predicates, dml, verification]
status: draft
sources:
  - id: runtime-receipt
    resource: evidence/plan-report-access-paths-20261108/runtime-receipt.json
    title: Pinned local-runtime command and completed assertions
  - id: runtime-log
    resource: evidence/plan-report-access-paths-20261108/runtime.log
    title: Complete fixed-image integration output
  - id: producer
    resource: evidence/plan-report-access-paths-20261108/runtime-producer/snapshot.json
    title: Exact dirty producer snapshot and retained reconstruction inputs
  - id: cleanup
    resource: evidence/plan-report-access-paths-20261108/runtime-cleanup.json
    title: Only current task-created container identities checked absent
  - id: local-validation
    resource: evidence/plan-report-access-paths-20261108/local-validation.json
    title: Per-module local tests and required pre-push checks
  - id: runtime-pin
    resource: evidence/plan-report-access-paths-20261108/runtime-producer/runtime_targets.json
    title: Exact historical registry reconstructed against its recorded source hash
  - id: test
    resource: evidence/plan-report-access-paths-20261108/runtime-producer/new-source/cmd/spanner-query-gen/plan_report_feedback_omni_test.go.txt
    title: Exact PLAN-only data-invariance and complete collector reproducer
  - id: contract-guide
    resource: ../../../cmd/spanner-query-gen/PLAN_CONTRACTS.md
    title: Current access-path, target and error contracts
---

# Scan-local PLAN contracts on Spanner Omni

Observed on November 8, 2026 JST. This is a bounded local-runtime check of the
application-query feedback, not a new managed-Spanner or Emulator investigation.

## Runtime and producer boundary

The cold local container uses Spanner Omni `2026.r2.1-beta` on `linux/arm64`,
with platform-manifest digest
`sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741`.
This deliberately matches the feedback's runtime; it does not assert that this
is the latest Omni release. Optimizer version 9 is requested through query
options and SQL hints. Effective optimizer/statistics metadata is not separately
reported. No PROFILE, latency or scan-row performance claim is made.

The producer uses Go 1.26.10 and Spanner SDK v1.95.1. The runtime receipt pins
the dirty producer bytes against source base
`300c760dbeb6215643799312da53c382f6101d6f`; the retained patch plus new-source
files reconstruct that snapshot. The post-run reconciliation at that time
found only three later write-scope expectations in an unexecuted unit test.
The retained producer and registry identify this r2 execution; the current
checkout now uses the newer runtime defaults described in the separate upgrade
verification note.

The access-path tables are empty. A separate T table has one sentinel row to
prove that DML PLAN acquisition does not execute its writes. Current owned
runtime and reaper container identities are absent after cleanup; unrelated
existing containers are excluded.

## Observed predicate distinctions

The generalized Events key is `(Shard, CreatedAt, Id)`. Surrogates has only
`Id` as its primary key.

| Case | Seek evidence | Residual evidence | Full scan |
| --- | --- | --- | --- |
| Shard equality, CreatedAt range, Owner equality | Shard and CreatedAt | Owner | false |
| Shard equality and Id equality, skipping CreatedAt | Shard | Id | false |
| Surrogates Owner equality and CreatedAt range | none | Owner and CreatedAt | true |
| Events seek with a separate latest-Surrogates subquery | Events Shard and CreatedAt | none on that Events path | true on the separate Surrogates path |

The skipped-middle-key case reports raw `seekable_key_size=0` while retaining
a real Seek Condition on Shard. Positive range-key metadata cannot substitute
for the typed condition signal, and zero cannot establish "no seek".

Both condition-root expressions and scan-local column bindings are retained.
Column extraction is conservative: it uses Reference nodes bound by the owning
scan. It does not infer a local column from arbitrary expression text or a
foreign join variable. Missing binding evidence stays empty rather than being
treated as proof of absence.

Unit and retained-managed-plan replay controls additionally cover ambiguous
wrapper pairing, independent scan ownership, node-order invariance, unknown
schema identity, expression-aware digests and CEL evaluation. These are source
and replay checks; they are not fresh managed executions.

Local tests pass in all five modules, with `GOWORK=off` and `-short` for the
survey module as required by its local verification task. The pre-push gate
also passes module-tidy, lint and vulnerability checks. A separate full survey
attempt could not access the Docker socket in the sandbox; it created no
runtime and does not constitute a container-test result.

## Complete collection and DML controls

The final collection contains ten targets: four optional shapes, configured
DELETE and UPDATE THEN RETURN, a generated DELETE helper, a STRUCT-array query,
a range query and one deliberately invalid STRING LIMIT parameter.

Nine targets have successful PLAN output. The invalid parameter remains one
target error while the other targets continue. Every optional shape receives
its own identity; the all-on shape keeps the original query ID. Omitted bind
parameters are absent from their statements. The generated helper is identified
as `write/Remove`.

Standalone UPDATE, DELETE and INSERT plans and the collected DML targets use
read-write AnalyzeQuery with explicit rollback. The sentinel row still contains
its original value after each standalone plan and after the complete collection.
The INSERT also targets its existing key without a duplicate-key execution
error. These controls distinguish PLAN acquisition from mutation execution.

A nested `ARRAY<STRUCT<...>>` parameter is accepted for UNNEST/join PLAN
acquisition through a single typed GenericColumnValue. The full collector also
accepts its simple STRUCT-array target. The range report renders `2/3` from the
catalog's three declared keys while its normalized raw value remains `2`.

## Remaining boundaries

The current catalog enrichment covers tables and ordinary secondary indexes.
Logical ANN root/leaf grouping, search/vector catalog facts, reliable back-join
lineage and a separate PROFILE surface remain follow-up work. Raw backend
expression spelling is version-sensitive; the normalized report exposes that
text without pretending it is a canonical SQL AST.

The developer plan-shape probe has no querygen-config input to connect to the
variant model. The shared expansion helper is available for a future adapter;
this change does not add an unrelated SQL-source extraction interface.
