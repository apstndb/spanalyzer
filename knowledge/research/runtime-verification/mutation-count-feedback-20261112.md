---
type: Research Note
title: Observed indexes and explicit mutation-count scenarios in plan reports
description: Design alternatives, observed indexes, explicit scenarios and count contracts, with original 31-commit evidence and a separate 36-commit CLI revalidation.
tags: [runtime-verification, spanner-omni, mutation-count, foreign-keys, plan-contracts]
status: draft
sources:
  - id: comparison
    resource: cross-engine-managed-recheck-20261112.md
    title: Separate matched managed, Omni and Emulator controls
  - id: earlier-accounting
    resource: mutation-count-20261110.md
    title: Earlier API-specific automatic cells, old-row effects and missing Emulator statistics
  - id: receipt
    resource: evidence/mutation-count-feedback-20261112/receipt.json
    title: Fixture scope, source identities, image provenance and retained correction
  - id: report
    resource: evidence/mutation-count-feedback-20261112/report.json.gz
    title: Full collector report and the 31 measured one-row scenarios
  - id: counts
    resource: evidence/mutation-count-feedback-20261112/counts.json
    title: Bounds, old-row refinement and same-execution measurements per API
  - id: cli
    resource: ../../../cmd/spanner-query-gen/README.md
    title: Scenario files and explicit fixture-measurement flags
  - id: contracts
    resource: ../../../cmd/spanner-query-gen/PLAN_CONTRACTS.md
    title: Conditional count contracts and observed access-path metadata
  - id: model
    resource: ../../../mutation_count.go
    title: Shared index catalog and bounds-first accounting implementation
  - id: foundation
    resource: https://github.com/apstndb/spanalyzer/pull/77
    title: Merged root and contract foundation with input-integrity corrections
  - id: publication-validation
    resource: evidence/mutation-count-feedback-20261112/publication-validation.json
    title: Subsequent standalone consumer closure and local publication checks
  - id: latest-revalidation
    resource: evidence/mutation-count-workflow-20261112/receipt.json
    title: Latest-feedback disposition, source hashes, 36 fresh CommitStats and cleanup
  - id: workflow
    resource: ../../../testdata/mutation-count/querygen.yaml
    title: Executable CLI configuration with explicit scenario and contract companions
  - id: latest-validation
    resource: evidence/mutation-count-workflow-20261112/validation.json
    title: Five-module gates and compiled CLI validation for the latest working snapshot
---

# Mutation-count feedback follow-up on November 12, 2026

The feedback was written against `8ba3f95`; its two remaining requests concern
foreign-key backing-index metadata and scenario-specific mutation intervals.
This follow-up starts from `eaf2694b9e68281d087d8bee339d0c0acf76cb68`.
It uses synthetic fixture names and explicit inputs. The supplied document is
feedback evidence, not authorization to execute its examples or publish changes.

## Alternatives and chosen boundaries

| Decision | Alternatives considered | Implementation and reason |
| --- | --- | --- |
| Physical FK names | Infer a generated-name pattern; duplicate a plan-only catalog; share validated metadata | Read a complete regular-index snapshot and reuse the accounting model's `IndexCatalog()`. Physical names are observed; constraint selectors remain explicitly inferred. This avoids naming-pattern and duplicate-catalog drift. |
| Old-row inputs | Add permanent generator config; analyze source code; accept explicit scenario files | Add `--mutation-facts-file` and `--mutation-indexes-file` to both report commands. Per-write/per-API scenarios can vary without changing generated application code or pretending to infer runtime values. |
| Live index snapshots | Trust a supplied file in any new DB; always require live metadata; reconcile when available | `explain-plan` remains offline. `plan-report` reads its own DB metadata, validates exact DDL identity and rejects an explicitly supplied snapshot whose physical contents differ. An optional observation failure retains a warning and conservative schema counts. |
| Actual counts | Always commit writes; report only static intervals; make measurement explicit | `--measure-mutation-counts` commits only selected one-row scenarios in command-created random fixture DBs. The normal PLAN path retains its rollback behavior. |
| Automatic columns | Generalize one LTS price; evaluate SQL expressions locally; retain bounds and exact-execution observations | Preserve `bounds`, `old_rows`, `observation` and the conditional `effective` interval. One measured point does not price future values, engines or schemas. |
| Contract evaluation | Require a SQL plan for every contract; treat a missing plan as empty; separate count-only contracts | Count-only CEL can evaluate an available interval on a skipped Mutation-only target. Contracts using operators, access paths or other plan inputs remain unevaluated when the SQL plan is unavailable. |

No generated-name regular expression is used. A referencing-side constraint
alias is attached only when the existing schema reconciliation finds a unique
managed-key-prefix match. The runtime forced-index control independently checks
the tested association. Ambiguous aliases remain unresolved. The observed
index's table, keys, stored columns, origin and managed flag travel to both
accounting and `access_paths`; a physical name and a logical constraint selector
are separate fields.

## Input and execution behavior

Facts are keyed by write name and API (`mutation` or `dml`); complete index
snapshots are keyed by catalog and exact DDL digest. Unknown names, invalid
values, extra JSON fields, trailing JSON documents and mismatched snapshots
fail explicitly. Missing values stay unknown. INT64 uses decimal wire strings,
so values beyond JavaScript's exact-number range retain their identity.
Default Mutation-only reports with no eligible SQL target open no database;
their index snapshots remain caller freshness assertions. When another target
opens the same catalog, its live reconciliation also governs skipped writes.
A rejected snapshot removes those counts before any contract evaluation.

Each measured scenario must supply every written value and the complete primary
key. Caller-authored `seed_dml` resets its starting state in a separate
transaction. It is inert during analysis and excluded from the measured commit.
The measured transaction reads the old row, executes one generated DML statement
or buffers one Mutation, and requests `ReturnCommitStats`. Retries replace the
old-row evidence rather than accumulating aborted attempts. Delete keys are
decoded to supported native SDK scalar types before any seed executes; unsupported
key types fail explicitly. This fixture path does not accept arbitrary DML or
multirow application commits as a measured target.

The command never selects an existing attached database as its measurement
target. An attached endpoint still receives a new random fixture database;
its default retention policy and `--drop-created-databases` remain explicit.
Owned runtime shutdown removes its containers and contained fixture data.
Seed failure or target failure does not undo a previously successful seed commit.

Missing CommitStats are unknown, never zero. A failed measurement, rejected
explicit snapshot, or count outside the conditional interval is retained in the
artifact and produces a nonzero command result after output is written.
The model bounds remain available even on disagreement. Observation timestamps
remain present under `--stable`; event identity takes precedence over a
deterministic measurement artifact.

## LTS measurement scope and result

The fixture uses Omni `2026.r4-lts.2` on `linux/arm64`, with the pinned platform
manifest `sha256:b607b10bee3c1903f0a7f88f99bece67cb0dd9d9a028aee64226fc311bd78dae`,
SDK v1.95.1, spanemuboost v0.4.10 and Go 1.26.11. It measures 16 scenarios:
both APIs for 15 scenarios and Mutation API only for `replace`, for 31 target
commits. Seeds are additional commits and are not included in that total.
All 31 successful target commits provide CommitStats and lie within their
conditional intervals. The named FK seek and count report share one physical
index identity with key `Owner`, base table `ForeignEvents` and declared key
count 1.

| One-row scenario | Mutation API count | DML count |
| --- | --- | --- |
| Four-column insert, zero / one / two regular indexes | 4 / 5 / 6 | 4 / 5 / 6 |
| Primary-key-only insert, two indexes | 4 | 4 |
| Primary-key-only insert, NULL_FILTERED index | 2 | 2 |
| Four-column insert, index storing `Note` | 6 | 6 |
| Update unrelated `Note` | 3 | 3 |
| Update stored `Note` | 5 | 5 |
| Change indexed `Active` | 5 | 5 |
| Change indexed `Active` with stored `Note` | 6 | 6 |
| Insert-or-update existing / missing key | 5 / 4 | 5 / 4 |
| Partial replace of existing row | 5 | No generated DML method |
| Delete existing full key with one index | 2 | 2 |
| Four-column FK insert, observed backing index | 5 | 5 |
| Primary-key-only insert, generated and DEFAULT columns | 6 | 4 |

The last row is API-specific. With no automatic-column calibration, the
schema interval remains `2..6`; the actual measurements refine only these
executions to 6 and 4. Changing an index key has schema intervals `3..5` or
`3..6`; old-row information narrows the tested changes before an observation
is applied. FK metadata removes the unresolved-backing-index uncertainty in
this fixture. None of these examples establishes commit-byte limits, latency,
quota enforcement or an engine-independent coefficient.

The first measurement attempt exposed an implementation error: the SDK rejects
`GenericColumnValue` inside a delete `Key`, although it accepts that type for
query parameters and Mutation values. That failed target has no successful
statistic. The failed report is retained separately; after native key decoding,
all 31 targets succeed. This is a local collector correction, not an Omni
behavior change or a missing statistic interpreted as zero.

## Compatibility and verification

`mutation_counts.mutation.effective.lower` and `.upper` are available to CEL,
with an unknown upper bound represented by `null`. Counts remain signed INT64
through activation without a JSON floating-point round trip. Contracts should
also inspect `effective_basis` and `disagreements` when the source of a point
interval matters. SQL-plan status and count availability remain independent.
Full JSON/YAML reports, compact summaries and Markdown include the counts.

The access-path normalization marker advances to `v1alpha.2`; the CEL evaluator
marker advances to `v3`. The report/config channels and mutation-count model
remain mutable previews. The leaf `plancontract` module retains its lightweight
dependency boundary and receives only count DTOs, not the analyzer or container
dependencies.

The initial candidate passed workspace tests, lint, vulnerability checks and
the shared LTS integration runner. Its first standalone build selected older
root/leaf revisions and failed; those staging logs retain their original status.
The foundation subsequently merged through PR #77 as
`0c08dadcc78ddf0ff14f1072da610ce4112e23f7`. The CLI now pins root
`v0.2.2-0.20261112055308-0c08dadcc78d` and plancontract
`v0.1.1-0.20261112055308-0c08dadcc78d` to that same immutable commit.
The separate publication-validation record tracks consumer closure and later
checks without changing the initial engine observations or producer identities.

The two large report JSON payloads are retained as deterministic gzip archives.
Their decoded bytes match the captured originals; the receipt records both
payload and archive hashes. The compact count table remains ordinary JSON.

## Latest feedback and CLI revalidation

The latest supplied feedback inspected `0c08dad`, the foundation merged through
PR #77. Main subsequently included the CLI integration through PR #78 as
`638b0261bb4da996647764729b4c9e6b488bbbf7`. The following disposition was checked
against that main snapshot and the explicitly hashed working-tree changes in
the [new execution receipt](evidence/mutation-count-workflow-20261112/receipt.json).
It does not change the earlier producer identities or re-date managed evidence.

| Feedback item | Current implementation and verification |
| --- | --- |
| Read the created DB's index list for scans and write counts | `ObserveIndexes` uses a consistent regular-index snapshot; the existing 31-commit integration control again verifies that the forced FK scan and count report share the physical name, managed flag, key and inferred selector |
| Accept old-row scenarios in both inspection commands | Both commands accept facts and index files; the checked-in scenario additionally states one-row cardinality explicitly for each API |
| Execute fixture seeds and capture CommitStats | Only explicit measurement runs seeds and target commits; the new dispatcher-level command test uses the documented flags and compares five successful target counts |
| Supply write counts to the contract evaluator | Three count contracts pass, including a Mutation-only replace whose SQL PLAN status remains `skipped` |
| DDL-only intervals retain automatic/index/cascade uncertainty | Original bounds remain in every report. This change does not hard-code the feedback's LTS points or invent child fanout |
| Source extraction and open SQL fragments | Remain outside this write-accounting feature; no source-discovery or `ORDER BY` rewriting was added |

The core CLI had a separate input-integrity gap: `old_row` was silently ignored
instead of rejecting a misspelling of `old`. The successful wider report was
reproduced before the fix. Facts and observation files now reject unknown
fields, with a file-qualified diagnostic; index-file strictness remains covered
too. Exactly-one-JSON-document validation is retained. The before/fix regression
logs and producer snapshot are part of the new evidence directory.

### Alternatives for this follow-up

| Choice | Considered alternatives | Applied result |
| --- | --- | --- |
| Functional integration | Duplicate accounting in a new tool; extend only the historical lab; exercise the existing library and CLIs | Keep the integrated implementation and add checked-in CLI scenarios/contracts that unit and live tests execute |
| Refinement JSON | Ignore unknown fields; warn and continue; reject them; introduce a new shared parsing package | Reject unknown fields using the existing command-local decoder pattern, preserving explicit schema and observation validation |
| DML cardinality | Infer one row from an old image; require a separate assertion | State `rows: 1` explicitly in the example; old rows and affected-row assertions remain separate inputs |
| Knowledge organization | Copy all historical findings into another mutable note; move or discard old runs; add a navigable feature concept | Add the [implemented-surface and verification map](../../concepts/mutation-count-analysis.md), retain the dated notes and append this source-scoped revalidation |

### New execution scope

The dedicated LTS container used the same platform manifest
`sha256:b607b10bee3c1903f0a7f88f99bece67cb0dd9d9a028aee64226fc311bd78dae`
for Omni `2026.r4-lts.2`, observed through container inspection on
`2026-11-12T07:56:12.063185Z`. The existing matrix completed 31 target commits;
the documented CLI workflow completed five more. All 36 counts agree with their
conditional intervals. The new workflow separately resets each API's initial
state; seed commits are excluded from target statistics.

| New scenario | Mutation API | DML |
| --- | --- | --- |
| Change the indexed key, with a stored column | 6 | 6 |
| Update an unrelated column | 3 | 3 |
| Partial replace of an existing row | 6 | No generated DML method |

The exact [scenarios](../../../testdata/mutation-count/scenarios.json) and
[contracts](../../../testdata/mutation-count/contracts.yaml) are linked from
the command documentation. The three contracts pass through the CLI dispatcher,
including the skipped-SQL-plan replace. An independent compiled CLI invocation
also parses the offline scenario file. These are concrete command paths in
addition to library tests; count availability is not inferred from exit zero.

The [count projection](evidence/mutation-count-workflow-20261112/counts.json)
retains baseline, old-row and observation results per API. Compressed full
reports and logs retain their hashes in the receipt. The first new test attempt
failed compilation because `EndpointFromRuntime` requires a started Runtime;
it created no container. After retrieving that value through `LazyRuntime.Get`,
the live run passed. Its owned container was stopped and terminated, and a
post-run inspection confirmed that it was absent.

The latest working snapshot passed `mise run pre-push` (all five modules' tidy,
standalone builds, lint and vulnerability checks) and `mise run verify-local`
(all five module test suites, generated catalogs, OKF and Markdown policy).
The [validation record](evidence/mutation-count-workflow-20261112/validation.json)
and retained logs identify their scope. Compiled CLI checks independently cover
the `[3,6]` schema interval, `[3,3]` old-row refinement, an explicit error for
`old_row`, queue TVF analysis with an unknown upper bound and the five configured
API scenarios. These are local checks of the hashed dirty source, not hosted CI
or a claim that these new changes have merged.

No new managed database was created, no new managed count was observed, and
Emulator's missing-statistic boundary is unchanged. The one-row measurement
path still does not execute arbitrary multirow/source-discovered application
commits or queue internal TVFs. The earlier managed ordinary-write and queue
notes remain the authority for those independently recorded experiments.
