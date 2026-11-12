---
type: Research Note
title: November 12 managed controls on Omni LTS, Omni r5 and Emulator
description: Current release checks and 1165 bounded cross-engine steps covering secure context, views, statistics, compression, scorer defaults and typed vectors.
tags: [runtime-verification, managed-spanner, spanner-omni, emulator, secure-context, parameterized-views]
status: draft
sources:
  - id: managed
    resource: managed-revalidation-20261112.md
    title: Earlier November 12 managed observations and owned-fixture corrections
  - id: receipt
    resource: evidence/cross-engine-recheck-20261112/receipt.json
    title: Exact images, observed platforms, scope, times and bootstrap corrections
  - id: matrix
    resource: evidence/cross-engine-recheck-20261112/comparison-matrix.json
    title: Every portable GoogleSQL step and matched managed read-only classifications
  - id: metadata
    resource: evidence/cross-engine-recheck-20261112/metadata-comparison.json
    title: Complete metadata tuple additions, removals and equality checks
  - id: postgresql
    resource: evidence/cross-engine-recheck-20261112/postgresql-comparison.json
    title: PostgreSQL control classifications against the retained managed outcomes
  - id: lts
    resource: evidence/cross-engine-recheck-20261112/omni-lts.jsonl
    title: Omni 2026.r4-lts.2 primary outcomes
  - id: r5
    resource: evidence/cross-engine-recheck-20261112/omni-r5.jsonl
    title: Omni 2026.r5.1 primary outcomes
  - id: emulator-google
    resource: evidence/cross-engine-recheck-20261112/emulator-google.jsonl
    title: Emulator GoogleSQL outcomes and separately identified PostgreSQL setup failure
  - id: emulator-pg
    resource: evidence/cross-engine-recheck-20261112/emulator-pg.jsonl
    title: Completed isolated Emulator PostgreSQL outcomes
  - id: omni-releases
    resource: https://docs.cloud.google.com/spanner-omni/release-notes
    title: Official Omni announcements, distinct from published patch tags
  - id: omni-download
    resource: https://docs.cloud.google.com/spanner-omni/download
    title: Official download page and advertised release series
  - id: emulator-release
    resource: https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/releases/tag/v1.5.59
    title: Current Emulator release from the GitHub API
  - id: emulator-limitations
    resource: https://docs.cloud.google.com/spanner/docs/emulator
    title: Official absence of Emulator authentication, permissions and roles
---

# Cross-engine observations on November 12, 2026

The starting repository revision is `eaf2694b9e68281d087d8bee339d0c0acf76cb68`.
The managed column below uses the earlier [November 12 investigation](managed-revalidation-20261112.md),
including its corrected GoogleSQL controls and additional native PostgreSQL
fixture. The host probes use Go 1.26.11, SDK v1.95.1 and spanemuboost
v0.4.10 on `darwin/arm64`; the engine containers use `linux/arm64`.
The retained binary build information confirms the probe dependencies.
Runtime timestamps are retained in UTC; the local
observations are November 12 JST. These are independently timed observations,
not simultaneous transactions or feature-introduction dates.

## Current public releases and tested identities

The complete public registry tag responses still identify `2026.r4-lts.2` as
the newest LTS patch, `2026.r5.1` as the newest Omni series/patch, and `1.5.59`
as the newest numbered Emulator release. Emulator's `latest` tag resolves to
the same image index as `1.5.59`. No newer numbered public tag was found.
The GitHub API's newest Emulator release is also v1.5.59, published October 14.

Omni's release notes announce r5 on October 29 and LTS.1 on October 28; the
download page lists the r5 series. Neither page provides a dated announcement
for each of the two patch tags actually tested here. Registry availability,
announcement dates and runtime observations retain their separate meanings.
The current main defaults and r5 alternative already select these versions;
this investigation does not change runtime pins.

| Runtime | Observed arm64 platform manifest |
| --- | --- |
| Omni `2026.r4-lts.2` | `sha256:b607b10bee3c1903f0a7f88f99bece67cb0dd9d9a028aee64226fc311bd78dae` |
| Omni `2026.r5.1` | `sha256:b918acd3c785bd13733d7dca6e6c029ef4ab8d7b63a0cd28800210a9bf976516` |
| Emulator `1.5.59` | `sha256:969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8` |

Each primary stream contains container-inspected image provenance and an OK
runtime-close outcome. The registry's multi-platform index digest is a
separate object from the platform manifest above. No amd64 run is claimed.

## Coverage and corrections

Each engine receives the same 355-step GoogleSQL plan: the 184 managed
read-only cases, corrected owned-fixture controls, UDF execution follow-ups
and six foreign-key/index controls. Emulator also receives the 100-step
PostgreSQL plan, including the separately corrected 17 native controls.
All 1,165 planned semantic steps have result records. An expected rejection
or dependency-blocked `NOT_RUN` remains such a record, not a successful API
call. No final semantic stream ends in an inconclusive transport error.

The portable GoogleSQL plan uses the managed corrections: qualified view
predicates, typed FLOAT32 literals, named-schema UDF prerequisites and vector
writes with the annotated index already present. Initial malformed managed
SQL is preserved in its original investigation rather than repeated as an
unsupported-feature claim. The PostgreSQL plan retains the two original
wrong-dialect index syntax controls; support conclusions use the later native
`CREATE INDEX ... USING scann ... WITH ...` controls.

The first Emulator runtime had no auto-created instance and executed zero
semantic steps. Corrected setup completed GoogleSQL; attempting PostgreSQL
with instance auto-creation again returned `AlreadyExists`. PostgreSQL then
completed all 100 steps in a separately owned runtime. Two failed provider
starts involved Colima socket discovery/mounting and occurred before SQL.
Those setup failures, corrected producers and close outcomes remain retained.
All local containers are command-owned. The managed comparison is retained
observation evidence; this follow-up creates no managed resource.

## GoogleSQL comparison

| Bounded behavior | Managed | Omni LTS.2 | Omni r5.1 | Emulator 1.5.59 |
| --- | --- | --- | --- | --- |
| String secure context supplied in request options | Returns supplied value | `Unimplemented`: secure parameters unsupported | Returns supplied value | Returns supplied value |
| Definer view with tenant context | Correct tenant rows; restricted role enforced | Corrected context view rejected; dependent role controls blocked | Correct tenant rows; restricted role enforced | Correct tenant rows; database-role restrictions absent |
| Direct base-table/function read by view-only role | `PermissionDenied` | Context-bearing requests rejected before useful privilege comparison | `PermissionDenied` | Succeeds; permissions/roles are unsupported |
| Invoker-view context with view-only caller | Denied | Dependent controls blocked | Denied | Executes; cannot establish a privilege boundary |
| Context call inside named-schema UDF | DDL accepted; execution binding error | Context-bearing request unsupported | Same execution binding error | Executes |
| Pass context result as ordinary UDF argument | Executes | Context-bearing request unsupported | Executes | Executes |
| Context in generated/default DDL | Rejected | Rejected | Rejected | DDL accepted; execution of those automatic expressions is not established here |
| Correct CAST with `AS`; comma-style documentation typo | Correct form executes; comma form rejected | Comma rejected; supplied-context positive request unsupported | Correct form executes; comma rejected | Correct form executes; comma rejected |
| Compression level limits, NULL and named-only size limit | Tested documented behavior | Same tested behavior | Same tested behavior | Compression calls unimplemented |
| Valid Zstd frame without content-size header | Rejected | Rejected | Rejected | Decompression unimplemented |
| Nonempty distribution aggregation, NULL guard | Counts 2/1 and same p99 | Same controlled results | Same controlled results | Aggregate fails resolution |
| New DB's unpinned SCORE | Matches v2, differs from v1 | Matches v2, differs from v1 | Matches v2, differs from v1 | Mock scores equal for default/v1/v2; no cohort conclusion |
| Typed FLOAT32 storage/KNN; short annotated vector with index present | Positive controls execute; short write rejected | Same controls | Same controls | Same bounded storage/length controls; no ANN quality claim |

Omni r5.1 already implements these parameterized-view controls despite the
later managed GA announcement. This is observed feature availability on this
image, not a claim that its release announcement lists the feature or that it
was introduced on November 10. The earlier LTS-to-r5 difference is established
by the paired pinned runs, not by applying a documentation cutoff as an
absence boundary.

Emulator context propagation does not provide managed-style access control.
The official Emulator limitations explicitly exclude authentication,
permissions and roles. Its successful base-table and direct-function reads
are consistent with that boundary. SQL-level role checks on the tested Omni
Developer endpoint do not establish IAM, TLS or caller-identity behavior.

## Metadata and PostgreSQL boundaries

| GoogleSQL advertised metadata | Managed | LTS.2 | r5.1 | Emulator |
| --- | --- | --- | --- | --- |
| INFORMATION_SCHEMA tables / columns | 48 / 310 | 48 / 310 | 48 / 310 | 30 / 204 |
| SPANNER_SYS tables / columns | 52 / 624 | 51 / 598 | 51 / 598 | 0 / 0 |

The complete INFORMATION_SCHEMA tuples match managed on both Omni images.
Their SPANNER_SYS tuples are exact subsets with 26 managed-only tuples.
Emulator has 119 managed-only and 13 Emulator-only INFORMATION_SCHEMA tuples;
equal column counts alone would not prove compatibility. Successful bounded
statistics calls are distinct from metadata advertisement and from populated
service statistics. The controlled histogram comes from retained managed
encoding, not invented Emulator/Omni telemetry.

Omni does not support the PostgreSQL interface, so no PostgreSQL DB is
attempted there. Emulator's annotated real-array short-write control rejects
as managed does, but native ScaNN index creation fails because setting an
access method is unsupported. Its PostgreSQL role DDL fails, and
`spanner.secure_context` does not resolve in the definer-view control. Dependent
role queries remain `NOT_RUN`; the GoogleSQL result is not projected onto this
dialect. Managed's successful native index and restricted-role view controls
retain their separate primary evidence.

## Foreign-key backing-index feedback

Both Omni versions expose a managed, null-filtered physical index whose leading
key is `Owner`. The seeded natural predicate and `FORCE_INDEX=EventOwner`
control resolve to that same physical index. Its generated suffix is retained
as observation evidence; no name pattern is used to synthesize catalog facts.
The implementation follow-up shares validated observed index metadata between
mutation accounting and access-path normalization. Constraint selectors remain
explicitly inferred from a unique managed-key prefix; the force-index runtime
control separately verifies the tested referring-side association.

The [implementation and measurement follow-up](mutation-count-feedback-20261112.md)
records the alternatives considered, the explicit measurement boundary and
the new 31-commit LTS fixture. It does not replace this cross-engine run.
