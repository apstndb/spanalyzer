---
type: Research Note
title: "Emulator 1.5.59 open-issue runtime follow-up"
description: "Pinned arm64 probes reproduce mutation/default, foreign-key, Change Streams and session discrepancies while separating working historical cases, fixture repairs and bounded performance samples."
tags: [spanner, emulator, runtime, issues, mutation, graph, change-streams, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: issue-audit
    resource: evidence/emulator-open-issues-20261030/audit.json
    title: "Seventy open issues and their complete source snapshot"
  - id: summary
    resource: evidence/emulator-issue-runtime-20261030/summary.json
    title: "Runtime identity, attempts, timings and scope limits"
  - id: initial
    resource: evidence/emulator-issue-runtime-20261030/emulator-issues.jsonl
    title: "Initial isolated issue cases and preserved unsuccessful attempts"
  - id: focused
    resource: evidence/emulator-issue-runtime-20261030/emulator-focused.jsonl
    title: "Corrected fixtures, binary protobuf, Change Streams and memory observations"
  - id: api
    resource: evidence/emulator-issue-runtime-20261030/emulator-extra.jsonl
    title: "Roles, sessions, schema concurrency and partitioned DML controls"
  - id: final
    resource: evidence/emulator-issue-runtime-20261030/emulator-last.jsonl
    title: "Corrected latency and explicit multiplexed-session controls"
  - id: coverage
    resource: evidence/emulator-issue-runtime-20261030/issue-coverage.json
    title: "Complete issue inventory with attempted and unverified scope"
  - id: cleanup
    resource: evidence/emulator-issue-runtime-20261030/cleanup-summary.json
    title: "Twenty-eight owned database lifecycles and sixteen containers"
  - id: release
    resource: evidence/emulator-issue-runtime-20261030/sources/latest-release.json
    title: "Refreshed official latest release"
---

# Emulator open-issue runtime follow-up, October 30

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The latest public release remains **1.5.59**. The pinned native linux/arm64
image is `sha256:969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8`.
The four SDK producer runs span **08:01:03-08:11:46 UTC / 17:01:03-17:11:46 JST**.
They use Go 1.26.8, Spanner SDK v1.91.0 and spanemuboost v0.4.10-alpha.1 in
an isolated GOWORK=off module. Direct SDK requests bypass the repository's
parser/analyzer. No new managed or Omni request was made.

There are **282 planned step records**, including setup, negatives, fixture
repairs and repeated controls. Forty-three issue families have directly
associated runtime attempts; eight further reports have related bounded
controls, and nineteen original scopes remain unexecuted. These counts do not
mean 282 successful statements or 43 fully reproduced/resolved issues. Every
one of the seventy open issues has a row in the linked coverage inventory.

## Reproduced discrepancies

| Issue | Observed result and control |
| --- | --- |
| [#395](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/395): partial mutation and defaults | Key-default, generated-key and composite-default variants reset omitted status/n from sent/5 to draft/0 after an Update mutation. Their DML controls retain sent/5. Ordinary keys and non-key-only defaults retain values under the tested mutation. |
| [#316](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/316): generated key and commit timestamps | The original generated-JSON-key schema accepts DML insertion. Mutation insertion fails with FailedPrecondition and a future timestamp sentinel. The successful DML row contains resolved commit timestamps. This is a partial reproduction, not the historical claim that both paths fail. |
| [#332](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/332): protobuf-generated timestamp | With the same valid binary protobuf value, the nanos-addition table returns Unknown/Unexpected error in RPC handling; the seconds-only table stores the expected row. Textproto also reaches a Type not found: INT32 error in the nanos expression. Following reads remain healthy; this is not grammar rejection or a demonstrated process crash. |
| [#353](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/353): cross-schema foreign-key metadata | Two referencing tables produce duplicated UNIQUE/KCU entries, empty constraint schema and inconsistent qualified names. The normal RC/TC join returns zero rows. |
| [#352](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/352): column stream untracking | ALTER succeeds, but DROP of the formerly column-tracked table returns FailedPrecondition naming the old stream. |
| [#210](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/210): Change Streams timestamps | The base row has a resolved commit timestamp, while the stream's new_values contains 9999-12-31T23:59:59.999999Z. A non-commit timestamp keeps its fractional precision in this fixture. |
| [#280](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/280): transaction tag | A transaction submitted with issue-runtime-tag produces a stream data record with an empty transaction_tag. |
| [#30](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/30): deleted active session | After deleting a session with an active read-write transaction, its commit returns NotFound and another session's read aborts due to active transaction 0. The owned database is still dropped successfully. |
| [#251](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/251): rename with foreign key | The bounded rename returns Internal with a foreign-key validator RET_CHECK, rather than hanging. Subsequent commands execute. This is a different failure form from the historical hang. |
| [#355](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/355), [#356](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/356): SNIPPET | Results retain start_position/end_position keys, omit source_begin/source_end and preserve repeated spaces/newlines. These are current Emulator observations consistent with the issue reports; no fresh managed SNIPPET comparison was run. |
| [#197](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/197): interleaved search index | The tokens-only interleaved fixture is rejected with parent-key type TOKENLIST versus STRING, a related failure form distinct from the original missing-key message. |

The successful data records for the stream cases come from an actual child
partition. Tokens and server transaction identities are hashed. The initial
root start preceded the emulator's earliest timestamp and returned OutOfRange;
the corrected run starts at the recorded commit timestamp. That first attempt
is a boundary/fixture result, not evidence that Change Streams is unavailable.

## Working historical cases and remaining limitations

| Issue or behavior | Bounded current result |
| --- | --- |
| [#265](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/265): zero lower bounds | TRAIL, ACYCLIC and ANY SHORTEST execute with explicit {0,1}/{0,3} and omitted-lower {,3}; expected zero-length endpoints are present. An upper bound of zero ({0}) rejects separately. Requested GQL ordering remains incorrect; endpoint sets and ordered lists are checked separately. |
| #265: shared node/edge label | Creates and MATCH executes when the label's property names/types agree. The initial incompatible property declarations correctly failed semantic validation and are not counted as the old label-reuse bug. |
| [#143](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/143): SQL ORDER BY/LIMIT | All six ASC/DESC and LIMIT 1/2/3 controls return the expected order. This does not explain or resolve the separate GQL discrepancy. |
| [#312](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/312): schema UPSERT | First insert, conflict update and DO NOTHING execute with the expected RETURNING and stored name values. |
| [#294](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/294): PostgreSQL FK conflict update | Both variants, including/excluding the FK key in SET, execute with the bytea literal. The initial unrelated decode() rejection is retained separately. No current managed legality claim is inferred. |
| [#233](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/233): view security | Both INVOKER and DEFINER create with properly qualified view columns. Initial strict-name errors were fixture failures, not missing security syntax. |
| [#304](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/304): SEARCH dialect | dialect=>'words_phrase' executes on populated indexed rows. |
| [#298](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/298): repeated stream ALTER | All five changes, including adding the fourth table, execute without the historical false quota error. |
| [#236](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/236): subquery UPDATE | The minimal original query and two controls complete on the bounded fixture. No historical hang occurs in this run. |
| [#347](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/347): timestamp keys | The stream preserves 1970-01-21T14:09:51.23456789Z, matching the base key. This is separate from the unresolved commit-timestamp sentinel above. |
| [#181](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/181): commit timestamp in IF | Insertion executes and stores a current resolved timestamp. |
| [#189](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/189): historical read | Both timestamp reads, measured after the actual commit timestamp, return the stored row. This does not cover every stale-read workload. |
| [#40](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/40): generated-column PDML | Ten read/update pairs on an interleaved generated-column table execute. No intermittent timeout occurs in this small pattern control; the external original project was not rerun. |
| [#292](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/292): nullable array of structs | Both original and COALESCE queries execute; the missing group's value is NULL versus an empty array respectively. The latter no longer fails in this fixture. No fresh managed nullable-array capability was tested. |

These are observations of current fixtures against historical reports. No
paired older-version rerun dates the introduction of any improvement, and
none of these observations closes the upstream issue.

Routine metadata (#386), role metadata/API (#350), ALTER SEARCH INDEX (#324),
empty-target GoogleSQL ON CONFLICT (#358), nested WITH (#276) and
FORCE_INDEX=PRIMARY_KEY (#214) retain rejection or missing metadata/API outcomes.
The documented _BASE_TABLE control executes. UDF execution (#261) works.
Search metadata columns (#341) exist; the unpartitioned/unordered search index
returns NULL option values, which does not validate non-empty option metadata.

## Hints, sessions and proposal flags

The per-query spanner_emulator.ignore_unknown_hints=TRUE bypass executes seven
otherwise unsupported name cases, including AUTO_PARAMETER, while normal
USE_ADDITIONAL_PARALLELISM already executes. It does **not** bypass a rejected
value of the recognized SCAN_METHOD hint. This distinguishes #300/#349 name
compatibility from value validation and from a startup-wide setting.

Explicit gateway_main --help and flag submissions reject the proposed
key-range locking, SQL pushdown, schema retention, diagnostic SQL and
startup ignore_unknown_hints flags as undefined. EMULATOR_RECLAIM and
EMULATOR_LOCK_STATS are absent. These are tests of the official binary's
current interface; linked fork proposals are not executed or promoted to
supported options. The existing version_retention_period='1s' database option
is accepted separately.

In three disjoint-key read-write controls, two second-session reads abort;
the third can proceed after the emulator's conflict decision. This does not
prove concurrent commit capability. The attempted schema-change control
rejects the DDL while a transaction is active, and the old transaction then
commits. It does not reach #406's reported schema-change abort message.

Ten newly opened/closed SDK clients complete their reads, and ListSessions
returns zero before/after. An additional explicit multiplexed CreateSession
also leaves ListSessions at zero while GetSession succeeds; DeleteSession
returns InvalidArgument because multiplexed sessions may not be deleted.
Consequently those zero counts do **not** prove all sessions were reclaimed.
ListDatabaseOperations is Unimplemented, so it cannot validate operation
retention. Commit with ReturnCommitStats succeeds but supplies no stats (#184).

## Bounded timing and memory observations

On the same 2,000-row fixture, five IN-subquery and JOIN samples each return
count 200. Median times are **1,544.116 ms versus 3.664 ms**. This reproduces a
large shape-dependent cost relevant to #209; it is not a production comparison,
a scaling proof or evidence for the individual profiling hypotheses #396-#398.

The duplicate-UNIQUE rejection takes **4.669 ms** in the corrected run. Five
omitted-default/explicit-default inserts have medians **1.785/2.190 ms**.
Neither the historical seven-second failure delay (#291) nor a default-value
slowdown (#187) appears in these samples. The original latency attempt hit
legitimate UNIQUE(NULL) conflicts; those failures are excluded from this
comparison and remain in the evidence.

The memory fixture writes 10,000 rows with 2 KiB payloads, sets a one-second
retention period and creates/drops two indexes. Container working-set samples
are approximately **28.08 MiB baseline, 68.73 MiB after seed, 79.39/87.85 MiB
after the two index creations, and 110.2 MiB after row/table deletion**. Short
post-drop samples do not return to baseline. Container working set is not
internal object reachability, allocator free space or proof of a storage leak;
this run does not settle all #400/#403 retention and reclamation claims.

## Attempts, remaining scope and cleanup

All initial inputs and producer versions are retained with corrected inputs.
The extra producer was reconstructed by removing only its later unused action;
rebuilding it produced exactly the retained original binary hash. Six initial
CLI attempts replaced the image CMD with a flag and never entered gateway_main;
the corrected explicit command invocations are separate. These Docker startup
errors are not treated as unknown-flag results. One short final runtime finished
before active inspection; bounded create/start/destroy daemon events recover
its exact image/container identity without fabricating active Cmd inspection.

The complete coverage inventory preserves nineteen original scopes that were
not executed: native/standalone build and distribution conditions, historical
image publication, conformance-harness shutdown, persistence, missing crash
inputs and original performance workloads among them. Eight related controls
also remain narrower than their reports. This research does not claim an
exhaustive reproduction or resolution of all seventy issues.

All **28 owned temporary database lifecycles** have successful drop and
GetDatabase NotFound results. Four SDK runtime containers and twelve CLI
attempt containers are confirmed absent. Earlier evidence and measured
implementation/dependency bytes remain unchanged. Repository runtime pins,
selectors and dependencies were not changed, and no upstream post, commit,
push or release was made.

The [validation receipt](evidence/emulator-issue-runtime-20261030/validation.json)
records offline value/provenance/privacy checks, module tests and documentation
gates. A successful producer exit or local gate does not convert the retained
engine errors into passing feature results.
