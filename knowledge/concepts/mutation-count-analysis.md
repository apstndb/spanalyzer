---
type: Architecture
title: Mutation-count analysis
description: Implemented Mutation API and DML accounting, explicit refinement and measurement, and environment-scoped verification evidence.
tags: [spanner, mutation-count, dml, queues, write-analysis, evidence]
status: draft
sources:
  - id: api
    resource: ../../doc.go
    title: Package documentation and runnable mutation-count examples
  - id: analyzer
    resource: ../../mutation_count.go
    title: Non-executing analyzer and separately qualified refinements
  - id: cli
    resource: ../../cmd/spanner-analyzer/mutation_count.go
    title: Standalone write-accounting CLI
  - id: generator
    resource: ../../cmd/spanner-query-gen/README.md
    title: Configured writes, scenarios, runtime reports and contracts
  - id: local
    resource: ../research/runtime-verification/mutation-count-20261110.md
    title: Local Omni and Emulator evidence
  - id: managed
    resource: ../research/runtime-verification/mutation-count-managed-20261110.md
    title: Managed ordinary-write verification
  - id: queues
    resource: ../research/runtime-verification/queue-mutation-count-managed-20261110.md
    title: Managed queue matrix and focused follow-up
  - id: integration
    resource: ../research/runtime-verification/mutation-count-feedback-20261112.md
    title: Shared observed indexes, scenario measurement and latest CLI revalidation
  - id: mixed-transactions
    resource: ../research/write-analysis/mixed-transactions-and-evidence-20261112.md
    title: Mixed-transaction integration, remaining uncertainties and evidence retention
  - id: fk-predictability
    resource: ../research/runtime-verification/fk-backing-index-predictability-20261112.md
    title: FK backing-index source and runtime investigation
---

# Mutation-count analysis

Mutation-count accounting is an implemented preview in the root library and
both CLIs. The lab retains historical strategy comparisons; it is not the
feature's execution engine. Current API behavior belongs in godoc and current
command behavior in the command documentation. This concept connects those
surfaces to the verification record without duplicating their instructions.

## Implemented surfaces

| Surface | Supported input and behavior | Execution boundary |
| --- | --- | --- |
| `MutationCountAnalyzer` | DDL, wire Mutation arrays, ordinary DML, mixed transactions, configured write shapes, explicit old rows/cardinality, complete observed-index snapshots and input-qualified observations | No database reads or writes |
| `spanner-analyzer --mode mutation_count` | Mutation arrays, `--sql` or `--mutation-transaction-file`, optional facts/index/observation files, JSON or YAML reports; unknown fields fail explicitly | No database connection |
| `spanner-query-gen explain-plan` | API-specific configured-write intervals and explicit scenario files | No database connection; `seed_dml` is inert |
| `spanner-query-gen plan-report` | Live regular-index metadata shared with access paths and write accounting | Ordinary SQL PLAN collection retains its rollback behavior |
| Explicit `--measure-mutation-counts` | Selected one-row Mutation/DML commits, separate fixture seeds, transactional old-row reads and CommitStats | Writes only to command-created fixture databases on Omni |
| Count contracts | `mutation_counts.<api>.effective`, basis and disagreement inputs | Independent of whether a Mutation-only target has a SQL PLAN |

The [command documentation](../../cmd/spanner-query-gen/README.md#runnable-mutation-count-workflow)
contains a checked-in configuration, scenario file and contracts that exercise
the feature from CLI input through explicit measurement and CEL evaluation.
The preview channels remain mutable; this is not a stable quota-gate promise.

## What narrows an interval

Schema-only analysis retains `bounds`. Explicit facts can add `old_rows`, and
a same-input/same-schema observation can add `observation`; `effective` states
which conditional result is currently available. A missing upper bound is
`null`, and missing statistics are unknown. The original bounds remain visible
when an observation disagrees.

An update depends on the indexes affected by its written columns and generated
dependencies. An unrelated column, a stored column and an index key have
different effects. Old/new membership also matters for NULL_FILTERED indexes.
The table's total index count is therefore not a fixed coefficient for every
update. DML additionally needs affected-row information; an old image alone
does not supply an explicit cardinality assertion in the current API.

`IndexCatalog()` exposes the same validated index facts used by accounting.
Observed physical names and managed flags remain distinct from inferred
constraint selectors. Offline snapshots assert completeness and freshness for
the exact DDL bytes; `plan-report` reconciles against the database it creates.
It does not invent generated index names.

## Verification map

| Concern | Retained evidence and present limitation |
| --- | --- |
| Insert, update, insert-or-update, replace, delete; partial updates | [Managed ordinary writes](../research/runtime-verification/mutation-count-managed-20261110.md), plus [Omni/API integration](../research/runtime-verification/mutation-count-feedback-20261112.md); observations identify their schema, input and runtime |
| Index keys, STORING, NULL_FILTERED, FK backing indexes, automatic columns and cascades | [Local comparison](../research/runtime-verification/mutation-count-20261110.md) and [reference audit](../research/write-analysis/mutation-count-reference-audit-20261110.md); unresolved values or fanout preserve intervals/unknown upper bounds |
| FK nullability, referenced-column order and generated physical names | [November 12 source/runtime investigation](../research/runtime-verification/fk-backing-index-predictability-20261112.md): nullable PKs and reordered composite keys retain an extra-index upper bound; observed shared physical indexes are counted once |
| Queue Send/Ack, missing messages, payload/key variants, same-key composition, DML and limit probes | [Managed queue matrix](../research/runtime-verification/queue-mutation-count-managed-20261110.md), with the completed 43-attempt follow-up retained separately |
| Mixed DML and buffered Mutation commits | [Mixed-transaction integration](../research/write-analysis/mixed-transactions-and-evidence-20261112.md) replays both retained managed observations; related stages remain uncertain and a whole-commit measurement refines only the total |
| Queue RECEIVE and RENEWLEASE | The queue API and CLI classify these separately; the [managed note](../research/runtime-verification/queue-mutation-count-managed-20261110.md) explains why their query responses do not provide caller CommitStats |
| Emulator | [Local controls](../research/runtime-verification/mutation-count-20261110.md) execute writes and inspect metadata; unavailable CommitStats do not provide numerical agreement or zero counts |
| Latest feedback and command integration | [November 12 follow-up](../research/runtime-verification/mutation-count-feedback-20261112.md#latest-feedback-and-cli-revalidation): foundation-versus-main disposition, strict input correction, runnable workflow, fresh runtime and contract results |

Same-key operations are not generally additive. Arbitrary application commit
assembly, source-code write discovery and queue internal bookkeeping need their
own evidence; the one-row measurement command does not infer mixed execution.
The offline mixed API accepts an explicit complete input instead. Mutation
counts also do not measure transaction bytes, latency,
contention, DDL backfill cost or future commit success.

See the [implemented strategy comparison](../research/write-analysis/mutation-count-alternatives-20261110.md)
for why conditional bounds were selected over a universal scalar, and the
[integration alternatives](../research/runtime-verification/mutation-count-feedback-20261112.md#alternatives-and-chosen-boundaries)
for observed-index reuse, explicit scenarios and opt-in measurement.
