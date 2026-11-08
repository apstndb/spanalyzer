---
type: Research Note
title: Inspection continuation and row deletion policy folding, November 8
description: Current-baseline reproduction and verification of partial static inspection and CREATE plus ALTER policy inputs.
tags: [verification, emulator, codegen, ddl, diagnostics]
status: draft
sources:
  - id: receipt
    resource: evidence/inspection-feedback-20261108/receipt.json
    title: Baseline, local source hashes and Emulator execution boundary
  - id: cli
    resource: evidence/inspection-feedback-20261108/cli-process-validation.json
    title: Actual standalone command exit codes and output routing
  - id: explain
    resource: evidence/inspection-feedback-20261108/explain.json
    title: Partial explanation preserving a later valid result
  - id: vet
    resource: evidence/inspection-feedback-20261108/vet.json
    title: Scoped unsuppressible query diagnostic
  - id: fixture
    resource: evidence/inspection-feedback-20261108/querygen.yaml
    title: Public minimal mixed-query input
  - id: design
    resource: ../../../cmd/spanner-query-gen/DESIGN.md
    title: Canonical inspection behavior and compared alternatives
---

# Inspection continuation and policy folding

The two reported gaps were reproduced against main commit
`55262a224d9f40ec5d1882452e1aac6e804e5926`. The retained fixtures use minimal
public examples and contain no downstream schema identifiers or source SQL.

## Local static inspection

`explain-plan` now keeps a failed target as `status: error`, emits its error,
and continues to later valid result fields. The structured artifact is
`status: partial` and includes scoped `planning_errors`. JSON, YAML and text
summary regressions cover two errors interleaved with two valid targets.

`vet` preserves later warnings and suppressions while reporting unsuppressible
query/write errors. Independent catalog failures and cross-target DTO conflicts
remain plan-wide errors. Tests also check write/query separation, typed
diagnostic IDs and output-error precedence. `generate` and `check` still reject
the mixed input without emitting partial Go code.

The retained standalone-process example produces nonempty explanation/vet
artifacts and exit code 1, with no duplicate stderr diagnostics for structured
inspection. The valid query retains its `Id` result field. Generation returns
exit code 1 and empty stdout. This is local GoogleSQL frontend and command
execution evidence, not managed Spanner or BigQuery service execution. The
CLI uses already-published planner APIs, so standalone behavior does not rely
on workspace replacements or a new dependency publication.
The retained explanation replaces its absolute local fixture path with
`schema.sql`; the receipt records the original artifact hash and transformation.

## Row deletion policy reconstruction

`ALTER TABLE ... ADD ROW DELETION POLICY` now populates the same table metadata
as inline `CREATE TABLE` policy syntax. Unit tests compare reconstructed DDL
for both default and named-schema tables, preserve schema identity, and reject
missing or non-table targets, duplicate policies and malformed ASTs. Policy REPLACE/DROP
remain explicit unsupported operations; arbitrary ALTERs are not skipped.

The pinned arm64 Emulator `1.5.59` accepted the CREATE-plus-ALTER fixture.
Exports from both AST and live metadata retained the policy as inline CREATE
syntax and reapplied successfully to fresh databases. The owned runtime closed
successfully. Its configured platform digest is
`sha256:969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8`;
this is not independent server-version discovery. The receipt records the
dirty source boundary and post-run hashes. No new managed or Omni execution
was performed for this follow-up.
