---
type: Research Note
title: "Emulator issue candidates after the GQL ordering report"
description: "A refreshed public issue audit separates two queue-validation candidates, metadata feature parity, existing reports and hypotheses needing narrower execution."
tags: [spanner, emulator, issues, queue, validation, metadata, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: audit
    resource: evidence/emulator-issue-candidates-20261031/summary.json
    title: "Prioritized candidates, existing reports and evidence boundaries"
  - id: upstream
    resource: evidence/emulator-issue-candidates-20261031/sources/issues-all.json
    title: "All accessible open and closed issue and pull-request bodies"
  - id: published
    resource: evidence/emulator-issue-candidates-20261031/sources/issue-408.json
    title: "Observed open GQL ordering report matching the saved draft"
  - id: searches
    resource: evidence/emulator-issue-candidates-20261031/sources/comment-search-summary.json
    title: "Targeted issue body and comment searches"
  - id: official
    resource: evidence/emulator-issue-candidates-20261031/sources/official-queue-sessions.json
    title: "Full official queue and session documentation"
  - id: implementation
    resource: evidence/emulator-issue-candidates-20261031/sources/queue-tvf.cc
    title: "Pinned queue TVF source at the current release and master"
  - id: runtime
    resource: evidence/cross-engine-doc-delta-20261029/emulator-queues.jsonl
    title: "Earlier pinned Emulator queue execution retained without rewriting"
  - id: checks
    resource: evidence/emulator-issue-candidates-20261031/assertions.json
    title: "31 checks of public status, contracts, existing execution and integrity"
  - id: validation
    resource: evidence/emulator-issue-candidates-20261031/validation.json
    title: "Local documentation and publication inventory checks"
---

# Emulator issue candidates, October 31, 2026

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The user-reported [GQL ordering issue #408](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/408)
is observed open, authored by `apstndb`, with title and body matching the saved
draft. This establishes publication; it does not authorize another issue or
comment. The [previous runtime survey](emulator-open-issue-runtime-20261030.md)
and [cross-engine comparison](cross-engine-doc-delta-20261030.md) supply the
execution evidence for the candidates below.

## Scope of this audit

The refreshed public API snapshot contains **405 objects: 293 issues
(71 open, 222 closed) and 112 pull requests**. It retains complete bodies,
nine targeted indexed body/comment searches and comments from six relevant
historical reports. No identical report was found for the two queue-validation
candidates. This is not proof about private trackers, discussions, unindexed
comments or every pull-request diff.

The latest official release is still **1.5.59**. Current master is
`d357ec1d2b08ae92ce1da4c5078de82fda0c07ab`, matching the inspected source.
**No new runtime is executed in this audit.** The runtime observations are the
October 29/30 captures of the pinned native arm64 1.5.59 image; current public
status, documentation and source are independently refreshed.

## Priority 1: successful renewal for an invalid lease token

The recorded Emulator query

```sql
SELECT * FROM RENEWLEASE_ManagedQueueProbe20261029r1(
  lease_tokens => ['not-a-token']);
```

returns OK and a row containing a non-NULL new token and expiration. The
[runtime record](evidence/cross-engine-doc-delta-20261029/emulator-queues.jsonl)
labels this `invalid-renew-token`; the queue was created by the retained
[producer](evidence/cross-engine-doc-delta-20261029/inputs/queues.go.txt).

The current [queue documentation](https://docs.cloud.google.com/spanner/docs/queues/queues-using)
says unparseable and expired tokens produce no row. Recorded Omni r5 and LTS
instead return InvalidArgument for this input. These sources differ on the
rejection form, but neither supports a successful renewal. A report should
focus on the fabricated success rather than require one exact error code.

The [input validation](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/backend/query/queue/queryable_queue_tvfs.cc#L491)
rejects only NULL or empty strings; the
[iterator](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/backend/query/queue/queryable_queue_tvfs.cc#L235)
constructs a new token by appending `-renewed` to any other input. The source
uses mock-lease terminology, so this is accurately described as a **lease
validation/conformance gap**, potentially an enhancement or a request to
document mock semantics rather than an accidental regression. Actual expiry,
acknowledged-token handling and renewal's effect on redelivery are separate
runtime scopes not established here.

The closed [#383](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/383)
requests CREATE QUEUE support; its two comments do not describe this renewal
behavior. RENEWLEASE and lease-token comment searches find no matching issue.

## Priority 2: RECEIVE accepts a stale read

The same retained producer uses
`spanner.ExactStaleness(time.Millisecond)` for `stale-receive`. Emulator
returns OK and the immediate message. Both recorded Omni images reject it
with a strong-concurrency requirement.

The [official receive contract](https://docs.cloud.google.com/spanner/docs/queues/queues-using)
requires a strong read and explicitly says stale reads are rejected. This is
a concrete validation discrepancy, distinct from ordinary historical reads.
The targeted queue/stale search finds no matching issue. The report should
include the SDK timestamp bound, since the SQL text alone does not reproduce
the condition.

## Other candidates and dispositions

| Observation | Appropriate disposition |
| --- | --- |
| INFORMATION_SCHEMA.TABLE_OPTIONS is absent | Feature-parity request. Recorded Emulator lookup fails while Omni lookups execute. No exact TABLE_OPTIONS report was found; #386 already covers the separate ROUTINES/ROUTINE_OPTIONS/PARAMETERS tables. Non-empty TABLE_OPTIONS contents and a complete public metadata contract need separate coverage. |
| HLL_COUNT and GQL column-propagation modes reject | Feature requests, not newly established wrong-result bugs. HLL_COUNT has no keyword report; generic CORRESPONDING matches are unrelated. |
| Long RECEIVE reaches the client deadline | Needs an isolated first-row timing reproducer before filing. Some short producers execute nested lease controls before recording elapsed time, so their elapsed_ms alone is not clean first-row latency. |
| NULL-filtered search-index query acceptance differs | Needs a narrow validator/flag/contract check. Successful execution without an Emulator plan does not establish physical use of the hinted index. |
| TO_JSON(graph node) rejects | The same signature failure is already in [#186's comments](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/186#issuecomment-3001440052). Prefer a follow-up or establish a distinct narrower case. |
| PostgreSQL JSONB operations reject | [#238](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/238) covers jsonb_set/jsonb_concat. It is related context, not proof that every unsupported comparison is an identical bug. |
| Partial Update mutations reset DEFAULT values | Already reported in [#395](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/395); the current-release reproduction is additional evidence. |
| Generated protobuf timestamps, FK metadata and Change Streams failures | Existing #332, #353, #210, #280 and #352 cover the observed scopes. |
| Multiplexed sessions absent from ListSessions | Expected enumeration semantics remain unsettled in this evidence; do not treat this as a confirmed new bug. |

## Retention and authority

The [31 evidence checks](evidence/emulator-issue-candidates-20261031/assertions.json)
verify publication status, contract text, actual retained outcomes, pinned
source paths and preservation of all 1,759 preceding artifacts and 123 measured
implementation files. The
[validation receipt](evidence/emulator-issue-candidates-20261031/validation.json)
records documentation checks. No new managed/Omni request, runtime fixture,
implementation edit, external post, comment, commit, push or merge is performed.
