---
type: Research Note
title: Emulator 1.5.59 and Omni r4 LTS.2 and r5.1 verification, November 8
description: Pinned runtime upgrades, shared-suite results and two reproduced metadata or PLAN failures.
tags: [spanner-omni, emulator, spanemuboost, runtime-upgrade, expression-index, remote-function]
status: draft
sources:
  - id: pins
    resource: evidence/runtime-upgrade-20261108/verified-image-pins.json
    title: Registry-retrieved index and per-platform manifest identities
  - id: lts-result
    resource: evidence/runtime-upgrade-20261108/omni-lts-final-report.json
    title: Completed default LTS shared-runtime verification
  - id: lts-log
    resource: evidence/runtime-upgrade-20261108/omni-lts-final.log
    title: LTS plan-probe and generated-code suite output
  - id: r5-result
    resource: evidence/runtime-upgrade-20261108/omni-r5-final-report.json
    title: Completed alternative r5 verification with a retained failure
  - id: r5-log
    resource: evidence/runtime-upgrade-20261108/omni-r5-final.log
    title: All eighteen Remote Function PLAN failures and later generator success
  - id: emulator-generator
    resource: evidence/runtime-upgrade-20261108/emulator-integration.log
    title: Emulator generated-code integration with race detection
  - id: emulator-expression
    resource: evidence/runtime-upgrade-20261108/expression-metadata-probe.log
    title: Initial minimal reproducer showing omitted index keys and incorrect reconstruction
  - id: reproducer
    resource: evidence/runtime-upgrade-20261108/expression-metadata-probe.go.txt
    title: Complete local Emulator metadata and reconstruction reproducer
  - id: emulator-canonical
    resource: evidence/runtime-upgrade-20261108/emulator-drift-fixed.log
    title: GetDatabaseDdl also loses expression-only index keys
  - id: dependency
    resource: https://github.com/apstndb/spanemuboost/releases/tag/v0.4.10-alpha.1
    title: Experimental Omni startup selection and image override release
  - id: source-reconciliation
    resource: evidence/runtime-upgrade-20261108/source-reconciliation.json
    title: Exact retained Omni producer bytes and later case-identifier-only changes
  - id: local-validation
    resource: evidence/runtime-upgrade-20261108/local-validation.json
    title: Module race tests, full survey, pre-push and generated-documentation gates
  - id: supplemental-probes
    resource: evidence/runtime-upgrade-20261108/supplemental-probes.json
    title: Roundtrip demo and complete optional-parameter developer probes
  - id: release-followup
    resource: evidence/managed-recheck-20261108/validation/commands.json
    title: Fresh five-module and local runtime validation with spanemuboost v0.4.10
---

# Runtime upgrades on November 8, 2026

The initial executions below used spanemuboost `v0.4.10-alpha.1` in the three
container-consuming modules. The later v0.4.10 adoption and fresh executions
are recorded in the follow-up section. The dependency's Omni support remains
experimental. The
repository default changes from Emulator `1.5.56` to `1.5.59`, and from Omni
`2026.r2.1-beta` to `2026.r4-lts.2`. Omni `2026.r5.1` is an explicit alternative,
not a change to the LTS default.

## Execution and selection boundary

Both `linux/amd64` and `linux/arm64` platform manifests were retrieved from the
public registries. Runtime executions in this note are on `linux/arm64` only;
amd64 manifest availability is not an amd64 test result. The retained pin
observation records all three image tags and their exact digests.

The producer is based on commit `300c760dbeb6215643799312da53c382f6101d6f` with
uncommitted feature and runtime changes. The retained source snapshot identifies
the Omni producer's changed inputs. Both final Omni executions completed with
those sources unchanged. Later survey-only reconstruction fixes do not change
the code exercised by these two suites. Go is 1.26.10 and the consumer Spanner
SDK is v1.95.1. No new managed-Spanner execution is claimed.

After those executions, the historical APPROX_COUNT_DISTINCT case identifier
was restored in its definition and lookup to preserve existing r2 expectation
manifests. The SQL and runtime assertions are unchanged. The reconciliation
receipt and retained producer files verify the original execution bytes.
Later changes route only the roundtrip and optional-parameter developer commands
through the registry; their complete executions are retained separately.

`runtime_targets.json` keeps the LTS tag and platform digests as the default and
stores r5.1 under the Omni target's `alternatives`. The strict module-local
loaders select only registered versions. Unknown version names fail before
startup; selecting an alternative leaves other callers' default unchanged.

```sh
mise run verify-omni-integration
mise run verify-omni-r5-integration
# From tools/:
go run ./omni-integration-runner --repo-root .. --omni-version 2026.r5.1
go run ./optparam-plan-probe --omni-version 2026.r5.1
```

The runner shares one cold container between the plan-probe and generator
suites. Each suite owns its databases. A failed suite is retained in the report
and does not discard the other suite's result. Cancellation stops further work,
and any suite failure still produces a nonzero exit status.

## Observed outcomes

| Runtime | Result |
| --- | --- |
| Emulator `1.5.59` | Generated-code integration passes; expression-index metadata and DDL serialization gaps are reproduced below. |
| Omni `2026.r4-lts.2` | Both complete plan-probe and generator suites pass. |
| Omni `2026.r5.1` | Generator suite passes. Plan-probe suite fails only the Remote Function test: both input shapes fail for optimizer versions 1 through 9. |

LTS accepts `APPROX_COUNT_DISTINCT`; the refreshed aggregate probe checks both
PLAN and the zero result for an empty table. Equality-literal PLAN descriptions
can contain `@_auto_p0_string_` instead of the literal. The condition test checks
both exact representations while preserving the parent operator, condition
type, and full predicate. Raw normalization does not rewrite these expressions.

The r5 profile recognizes `SECURE_CONTEXT` and reports `Missing secure
parameter: key`; LTS reports the unsupported-function boundary. For the tested
short GQL EXISTS query, r5 has three Aggregate nodes at optimizer versions 6–9,
where LTS has two. The existing equivalence and ownership assertions remain in
place; these profile-specific physical-plan differences are explicit.

New default-runtime catalog captures are retained under the survey producer:

- [Emulator INFORMATION_SCHEMA: 204 columns](../../../survey/infoschem/evidence/emulator/969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8/linux-arm64-e082c284708c.json).
- [LTS INFORMATION_SCHEMA: 310 columns](../../../survey/infoschem/evidence/omni/b607b10bee3c1903f0a7f88f99bece67cb0dd9d9a028aee64226fc311bd78dae/linux-arm64-89b4ab9e231d.json).
- [LTS SPANNER_SYS: 598 columns](../../../survey/spannersys/evidence/omni/b607b10bee3c1903f0a7f88f99bece67cb0dd9d9a028aee64226fc311bd78dae/linux-arm64-9bb70c640b69.json).

Fresh read-only comparisons against all three captures pass. Historical captures
and the managed-selected common projection are preserved.

Local race tests pass in all five modules, including the full independent
survey suite with real Emulator and LTS containers. Required tidy, lint and
vulnerability checks pass. The r5 Remote Function failure remains a separate
live-suite failure, rather than being skipped or converted into success.
The roundtrip demo also completes on Emulator 1.5.59, and the complete
optional-parameter developer probe completes on both Omni versions. These
commands explicitly select the registry's default or requested alternative
instead of inheriting a different image from the dependency.

## Emulator expression-index metadata loss

This reproduces the gap in a fresh Emulator database:

```sql
CREATE TABLE ExpressionProbe (Id INT64 NOT NULL, Data JSON) PRIMARY KEY (Id);
CREATE INDEX ExpressionProbeByValue ON ExpressionProbe((JSON_VALUE(Data.value)));
SELECT INDEX_NAME, COLUMN_NAME, ORDINAL_POSITION
FROM INFORMATION_SCHEMA.INDEX_COLUMNS
WHERE TABLE_NAME = 'ExpressionProbe';
```

The DDL succeeds. The query contains only the primary-key row, omitting the
expression index key. `INDEX_COLUMNS.EXPRESSION` itself is unavailable. The
generated hidden expression column instead appears in `COLUMNS`.
`GetDatabaseDdl` also returns `CREATE INDEX ExpressionProbeByValue ON
ExpressionProbe()` with no expression key.

Before the compatibility fix, reconstruction returned no error and emitted a
generated hidden table column plus an empty index. A mixed ordinary/expression
index can similarly lose its expression key. The converter now rejects a
generated hidden expression-index column whose corresponding key metadata is
missing. Complete ordinary indexes on explicitly described hidden generated
columns remain representable. Regression controls cover both empty and
truncated key metadata, plus that ordinary-column counterexample.

The shared live fixture explicitly records the Emulator's known incomplete
`GetDatabaseDdl` output while requiring reconstruction to fail closed. This is
an observed upstream reporting gap, not successful expression-index recovery.

## r5 Remote Function PLAN regression candidate

Create the existing fixture's remote function:

```sql
CREATE SCHEMA spanalyzer_remote;
CREATE FUNCTION spanalyzer_remote.remote_add(x INT64, y INT64) RETURNS INT64
NOT DETERMINISTIC LANGUAGE REMOTE
OPTIONS (endpoint = "https://spanalyzer-remote-test-uc.a.run.app", max_batching_rows = 10);
```

Request PLAN with `AnalyzeQuery`, for example:

```sql
@{OPTIMIZER_VERSION=9}
SELECT spanalyzer_remote.remote_add(1, 2) AS total;
```

On r5.1 this returns gRPC `Internal` with an empty description. Both literal
input and the table-column input in `remoteFunctionQueries` fail for all nine
optimizer versions, producing eighteen retained failures. A separate cold r5
run also failed the first literal-input case. The same full test passes on the
fixed LTS image with the same SDK and producer source. This is a runtime-
reproduced regression candidate; no upstream root cause or report is claimed.
The failing test remains enabled, so the r5 verification command exits nonzero.

## Follow-up with spanemuboost v0.4.10

The three consuming modules were subsequently updated to the published
`v0.4.10` module. The complete comparison with alpha.1 contains one commit,
`62a784a09d51c85c10c648e49a4b5bd99cb2aba0`: upstream default images and their
documentation/tests changed. This repository continues to select its explicit
LTS.2 default and r5.1 alternative rather than adopting the dependency's r5
default implicitly.

Fresh race tests pass in all five modules, including the full Docker-backed
survey suite. Emulator generated-code integration passes. Both LTS integration
suites pass. The r5 generator suite passes and its plan suite still fails only
`TestIntegrationRemoteFunctionPlansOnOmni`, with the same eighteen `Internal`
errors. The nonzero r5 status is retained. Read-only comparisons with all
three default-runtime catalog captures also pass. Tidy, lint and vulnerability
checks pass; no reachable vulnerability is reported.

The [new command receipts](evidence/managed-recheck-20261108/validation/commands.json)
and [producer source hashes](evidence/managed-recheck-20261108/validation/producer-sources.json)
identify these later executions. The original alpha.1 evidence above remains
historical. The separate [managed revalidation](managed-revalidation-20261108.md)
records the new service observation and documentation audit.
