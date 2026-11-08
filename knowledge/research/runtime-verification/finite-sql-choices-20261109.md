---
type: Research Note
title: Closed SQL choices on Emulator and Omni LTS, November 9
description: Correlated range and column-choice enumeration, selected bindings, local row membership and Omni PLAN collection for the mutable v1alpha generator.
tags: [querygen, sql-choice, emulator, spanner-omni, runtime-verification]
status: draft
sources:
  - id: receipt
    resource: evidence/finite-sql-choices-20261109/receipt.json
    title: Runtime window, producer hashes, exact dependency and cleanup verification
  - id: runtime
    resource: evidence/finite-sql-choices-20261109/runtime.jsonl
    title: Redacted primary integration test events
  - id: patch
    resource: evidence/finite-sql-choices-20261109/producer.patch
    title: Tracked CLI changes against the recorded producer base
  - id: fixture
    resource: evidence/finite-sql-choices-20261109/sql_choice_integration_test.go.txt
    title: Exact membership and PLAN collector controls
  - id: config
    resource: evidence/finite-sql-choices-20261109/sql_choice_test.go.txt
    title: Synthetic configuration and SDK-facing generation checks
  - id: command-guide
    resource: ../../../cmd/spanner-query-gen/README.md
    title: Closed SQL choices and compatibility limits
  - id: alternatives
    resource: ../../../cmd/spanner-query-gen/DESIGN.md
    title: Design alternatives and finite verification boundary
---

# Closed SQL choices on local runtimes

Observed November 9, 2026 JST (November 8 UTC). This verifies the generator's
new finite alternatives, not a change in Spanner syntax or an engine release.
The retained receipt gives exact timestamps and the dirty CLI producer hashes.
The CLI resolves root implementation commit
`beb4f5761002ee5a6d826a0a03cceab34e8a97df` through the immutable pseudo-version
`v0.2.2-0.20261108145506-beb4f5761002`, with `GOWORK=off`.

## Selected design

`sql_choice` replaces one marked location with one declared fragment. A single
range selector models both bounds, start only, or end only; it does not invent
a fourth unbounded case. A second independent selector chooses the Active or
Pending predicate, producing six same-result-type statements. Correlated
decisions belong in one selector. Generation lexes real binding references,
omits values not used by the selected SQL and never binds the selectors.

Separate named queries remain suitable when row types differ, at the cost of
duplicated common SQL and caller dispatch. Presence constraints would filter
Cartesian combinations but would not naturally express column selection.
Changing `orderby_choice` would obscure its existing meaning. Open runtime
identifiers and clauses need a closed allowlist or a different contract. These
alternatives and the chosen preview boundary are recorded in the design guide.
`null_is_null` retains NULL matching; omission modes remove predicates.

## Runtime and results

The local Docker run started these configured `linux/arm64` images:

| Runtime | Platform manifest |
| --- | --- |
| Emulator 1.5.59 | `sha256:969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8` |
| Omni 2026.r4-lts.2 | `sha256:b607b10bee3c1903f0a7f88f99bece67cb0dd9d9a028aee64226fc311bd78dae` |

The primary startup log records the selected image references. These are not
independent server-version responses. SQL-choice controls do not pin or
independently observe optimizer or statistics versions. Other integration
tests retain their own settings; no latency or scan-cost measurement is made.

The fixture contains four events in shard 1 and a fifth in another shard.
Times are Unix seconds 1 through 4, and the requested range is inclusive
seconds 2 through 3. Result order is unspecified, so membership is compared
after sorting the returned identifiers.

| Bounds / selected column | Expected IDs | Emulator | Omni |
| --- | --- | --- | --- |
| both / Active | 2 | matched | matched |
| both / Pending | 2, 3 | matched | matched |
| start / Active | 2, 4 | matched | matched |
| start / Pending | 2, 3 | matched | matched |
| end / Active | 1, 2 | matched | matched |
| end / Pending | 2, 3 | matched | matched |

Omni additionally returns a nonempty PLAN for every candidate, and the actual
collector produces six successful targets with no errors. Emulator query
execution is not evidence about Omni or managed optimizer plans. Eight
top-level CLI integration tests pass in the same run, including the existing
operator-family, DML, generated-query, partial-report and retention controls.
All eight container identities emitted by this process are absent from
`docker ps -a` after test exit; unrelated containers are not removed.

Local unit checks also cover exactly three/six/twelve-way enumeration, stable
labels and SQL digests, default and unknown selectors, quoted binding-like
text, undeclared bindings, row-type mismatch and the 4,096-variant cap. A
compiled standalone builder reproduces selected SQL and argument sets, and
the complete generated SDK-facing Go file compiles. These local checks are
distinct from the retained runtime events.

## Limits

This adds a mutable `v1alpha` feature, with no stable release or contract
freeze. Choice fragments require required value parameters; independent
nullable omission blocks remain separate. Required TIMESTAMP values use
`time.Time`; this does not add every GoogleSQL type to optional builders.
BigQuery and row-count DML optional variants remain unsupported. No managed
Spanner or Omni r5 execution is performed here. Earlier observations retain
their dates and are not promoted by this local run.
