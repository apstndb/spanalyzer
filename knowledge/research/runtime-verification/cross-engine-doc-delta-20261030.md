---
type: Research Note
title: "Managed documentation cases on Omni r5, r4 LTS patch and Emulator 1.5.59"
description: "Exact-image replays distinguish Omni release tracks, Emulator JSON improvements, reversed GQL ordering and queue/search differences against the October 29 managed sample."
tags: [spanner, managed, omni, emulator, runtime, json, graph, queues, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: managed
    resource: managed-documentation-delta-revalidation-20261029.md
    title: "Earlier October 29 managed reference and documentation audit"
  - id: receipt
    resource: evidence/cross-engine-doc-delta-20261029/receipt.json
    title: "Images, client versions, observation window and cleanup"
  - id: comparison
    resource: evidence/cross-engine-doc-delta-20261029/comparison.json
    title: "Matched codes, deterministic values, row types and preserved attempts"
  - id: features
    resource: evidence/cross-engine-doc-delta-20261029/feature-summary.json
    title: "Feature cases, historical JSON comparison, heartbeat and queue observations"
  - id: cleanup
    resource: evidence/cross-engine-doc-delta-20261029/cleanup-summary.json
    title: "Thirty-seven local database lifecycles and fifteen runtime containers"
  - id: releases
    resource: evidence/cross-engine-doc-delta-20261029/sources/release-refresh.json
    title: "Refreshed public version tags and latest Emulator GitHub release"
  - id: official-documents
    resource: evidence/cross-engine-doc-delta-20261029/sources/official-runtime-documents.json
    title: "Full official Omni release notes and Emulator guide"
  - id: emulator-release
    resource: https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/releases/tag/v1.5.59
    title: "Official Emulator 1.5.59 release"
  - id: previous-containers
    resource: cross-engine-json-and-change-streams-20261010.md
    title: "Historical exact-image r4 and Emulator 1.5.58 comparison"
  - id: emulator-open-issues
    resource: evidence/emulator-open-issues-20261030/audit.json
    title: "October 30 open-issue snapshot and bounded runtime relationships"
---

# Documentation-driven comparison, October 29 UTC / October 29-30 JST

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Omni **2026.r5** matches the managed sample's new SET NULL, JSONB, graph,
search and tuning outcomes within the tested scope. **2026.r4-lts.1** rejects
SET NULL and the explicit volatile-view option while its JSONB ordering,
search, GQL propagation and HLL cases execute. **Emulator 1.5.59** materially
improves GoogleSQL JSON support, but still differs in PostgreSQL JSONB,
GQL column propagation, approximate/HLL functions, search validation and queues.

Emulator also returns the opposite of the requested GQL sort direction on the
small ordered fixtures. Independent SQL controls sort correctly. This is a
reproduced successful-query result discrepancy, separate from unsupported SQL
and from fixture capacity failures.

The container execution window is **October 29, 14:53:13-15:40:32 UTC** /
**October 29, 23:53:13-October 30, 00:40:32 JST**. The managed reference remains
its earlier **00:30:18-01:29:28 UTC** sample; no managed endpoint or credentials
were injected into these container probes. Managed fleet state and rollout
completion are not inferred from these local images.

## Latest release and image checks

| Item | Current primary-source observation |
| --- | --- |
| Emulator | GitHub's latest release and the highest public version tag are **1.5.59**, published October 14, 04:17:58 UTC. The selected image was built October 14. |
| Omni regular version tag | The public registry contains **2026.r5**, with a selected image built October 24. |
| Omni LTS patch tag | The public registry contains **2026.r4-lts.1**, with a selected image built October 22. |
| Omni announcement/download pages | Fetched full API bodies, live release HTML and the download table still identify **2026.r4-lts**, announced September 30. The official GitHub releases endpoint returns an empty list. No r5 or LTS-patch announcement/support promise is inferred from a public tag. |
| Existing r4 tag | `2026.r4-lts` still resolves to the historical arm64 digest `9f10586c5de3`; it is distinct from the patch tag. |

Both registries and the latest Emulator release were refreshed after execution;
the selected highest version tags remain unchanged. Full release bodies,
registry tag lists, manifests, image descriptors and source hashes are retained.
Image creation, publication, announcement and observation dates remain distinct.

**Documentation follow-up, October 31 JST:** the newer mirror capture and
current official download body now identify **2026.r4-lts.1**, dated October 28,
and update installation examples. This supersedes the earlier download-page
observation above, without changing the dated image/runtime findings. The
download page does not list r5. The [complete source audit](documentation-delta-review-20261031.md)
retains the exact interval and official bodies.

| Tested target | Platform | Exact platform manifest |
| --- | --- | --- |
| Omni 2026.r5 | linux/arm64 | `sha256:e506672206e0291cac28cbe19ad4cd470cec112c24601637bc896c65f7362396` |
| Omni 2026.r4-lts.1 | linux/arm64 | `sha256:5e3cf199f5abf3a09cd2348ab36c129b3268966eb4eb72fca278fe58252fcbdb` |
| Emulator 1.5.59 | linux/arm64 | `sha256:969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8` |

Active Omni inspections retain the exact image and
`start-single-server --listen-addresses=0.0.0.0` command. Two very fast Emulator
runs completed before active inspection; bounded daemon create/start/destroy
events recover their exact image/container identities. Runtime platform
inspection and image descriptors identify native arm64. Other architectures,
commercial deployments, TLS/mTLS, upgrades and external replication were not
exercised. Repository runtime pins and catalog selectors remain unchanged.

The Emulator release notes additionally list queue DDL/DML/TVFs, expression
indexes, placement metadata, multi-key vector indexes, placement routing,
JSON subfield DML, PostgreSQL UDF defaults/bodies, and size/clock/memory fixes.
This task probes the cases below; it does not establish every release-note
item, capacity limit or memory/clock fix through execution.

## Behavior matrix

The managed column is the retained October 29 sample. Executes means a
successful response within the stated fixture; it is not a complete feature
or performance guarantee.

| Behavior | Managed reference | Omni r5 | Omni r4-lts.1 | Emulator 1.5.59 |
| --- | --- | --- | --- | --- |
| SET NULL, both dialects | CREATE/ALTER, DML/mutation delete and rollback execute | Same bounded results | Explicitly unsupported | GoogleSQL syntax rejection; PostgreSQL unsupported action |
| GoogleSQL JSON comparison/cast corpus | 125 cases, 99 successful | All codes and successful values match | Same | 124 codes match; all 99 successful values match; analytic-window rejection uses a different code |
| Distinctness corpus | 64 cases: 49 OK / 15 rejected | Same codes/values/types | Same | Same |
| Ordinary ENUM CAST to JSON | Executes | Executes | Executes | Executes |
| Ordinary PROTO CAST to JSON | Executes | Executes | Executes | Unimplemented |
| GRAPH_ELEMENT / GRAPH_PATH CAST to JSON | Both Unimplemented | Same | Same | Element rejects; path executes and returns JSON |
| TO_JSON(node) / TO_JSON(path) | Both execute | Same | Same | Node rejects; path executes |
| PostgreSQL JSONB comparisons, ORDER BY, GROUP BY and equality join | Execute | Match bounded rows | Match bounded rows | Relevant functions/operations Unimplemented |
| COUNT(jsonb), including DISTINCT | Unimplemented | Same | Same | Unimplemented |
| Volatile-view option omitted / FALSE / TRUE | Available views create even with default/FALSE, contrary to documented prohibition | Same 24 definitions and reads | Eight omitted-option definitions create; 16 explicit-option definitions return Unimplemented | Same 24 definitions after quota isolation |
| RAND | Unsupported directly and in views | Same | Same | Same |
| Finite path quantifiers | Six execute | Same ordered rows | Same ordered rows | Same endpoint multisets; returned ordering differs |
| Unbounded / reversed path bounds | Ten unbounded and one invalid-bound case reject | Same | Same | Same after quota isolation |
| GQL column propagation | 36 positive set-operation cases execute | Match columns/rows | Match columns/rows | Column propagation modes rejected semantically |
| Dynamic-property GQL ORDER BY | Correct fixture ordering | Correct | Correct | Ascending/descending order reversed on the retained controls |
| Numeric prefixtree/logtree/base comparisons | Six predicates, four access variants agree | Same rows; forced plans show search access | Same | All 24 execute with matching rows; no physical plan supplied |
| NULL-filtered search requirements | Explicit or implied non-NULL predicate needed | Same | Same | Also accepts omitted/NULL-admitting predicates, including forced-index forms; returns base-equivalent rows |
| Approximate/HLL controls | 32 original plus 22 focused cases execute/reject as expected | Same scoped results | Same | All 54 calls reject; no positive HLL execution |
| Columnar statistics fields | Advertised/queryable; no active coverage established | Queryable | Queryable | No SPANNER_SYS tables advertised; value queries reject |
| ANY_VALUE HAVING MAX/MIN | Expected selected rows | Same | Same | Same |
| PostgreSQL generate_series | 32 values, sum 496 | Same | Same | Unimplemented |
| Change Streams 100 ms | Actual child heartbeat records in both dialects | Same capability | Same capability | Same capability |
| Same-name queue recreation | Two correct generations | Both correct | Both correct | Both correct |

Original rejections and dependency-blocked cases are retained. Unsupported
SET NULL blocks dependent child-data checks on LTS and Emulator; it does not
establish that ordinary transactions or CASCADE are unavailable. A successful
forced-index query without a plan does not prove that the index was used.

## Emulator changes against the retained 1.5.58 sample

The 125-case JSON input files are byte-identical to the October 10 corpus, and
the client stack remains Go 1.26.8, Spanner SDK v1.91.0 and spanemuboost
v0.4.10-alpha.1. Historical 1.5.58 had fifteen successful cases; current 1.5.59
has ninety-nine: **84 formerly failing cases now execute**. The full code
comparison improves from 35/125 matches to 124/125. The remaining code
difference concerns unsupported analytic functions, not JSON comparability.
This is a comparison of two pinned image versions observed on different dates.

ENUM and ordinary scalar/array/struct controls now execute, while the separate
PROTO CAST still returns Unimplemented. The separate GRAPH_PATH CAST changes
from historical rejection to execution, unlike managed/Omni. Its metadata is
JSON and the returned path is an array. Neither code acceptance nor this
array shape establishes cross-engine graph serialization compatibility.

## Reproduced GQL ordering discrepancy

The two dynamic property values are inserted as Id 1 -> value 2 and
Id 2 -> value 1. Managed, both Omni images and the SQL control return 1,2
for ascending order. Emulator returns 2,1 for the original query, three
repetitions, a renamed alias, an explicit ASC, and a version without WHERE.
DESC returns 1,2. Converting the projected value to INT64 does not repair it.
A static graph Id DESC control returns 1,2 while SQL Id DESC returns 2,1.

All three images run the identical twelve-query
[ordering matrix](evidence/cross-engine-doc-delta-20261029/inputs/order-matrix.json).
The [Omni r5](evidence/cross-engine-doc-delta-20261029/omni-r5-order.jsonl),
[LTS](evidence/cross-engine-doc-delta-20261029/omni-lts-order.jsonl) and
[Emulator](evidence/cross-engine-doc-delta-20261029/emulator-order.jsonl)
records preserve the results. Six bounded-path ordered results also differ
from managed while retaining the same endpoint multisets. The internal cause
and the first affected Emulator version remain unverified.

**Later October 30 follow-up:** the
[root-cause verification](emulator-gql-ordering-root-cause-20261030.md)
locates lost order metadata in graph-table algebrization and intentional
scrambling of the sorted output. Five controlled runtime conditions confirm
that boundary; the first affected release remains unverified. The original
two-row observation above is preserved, while a five-row fixture establishes
permutation rather than a general reversal rule.

## Search, catalog and queue boundaries

Both Omni images advertise exactly the managed reference's 310 IS tuples and
its historical 51-table / 598-column SYS shape. They do not advertise the new
BATCH_OPERATIONS/statistics additions in the managed 609/613/624 responses.
Emulator advertises 30 IS tables / 204 columns and no SYS columns in the
same discovery query. Research tuple-array hashes are independent from the
producer-owned capture contract; no new analyzer selector is implied.

Both Omni images reproduce the 19 JSON/JSONB search triplets and 24 numeric
access variants. Actual scan_type metadata distinguishes search and base
access. Emulator supplies no plan nodes, accepts all eighteen NULL-filtered
search queries and returns the corresponding base rows, including NULL rows
that the forced index excludes. This does not demonstrate physical use of
that filtered index or production-equivalent optimizer validation.

All engines create queues and complete two same-name recreation generations.
Emulator exposes TABLE_TYPE=QUEUE and three queue-table columns, but its
INFORMATION_SCHEMA lacks ROUTINES and TABLE_OPTIONS. Registering execution
TVFs does not imply those metadata views exist.

Both Omni images reject stale receive. Emulator accepts it and returns the
immediate message. Its subsequent 20-minute receive reaches the 35-second
client deadline. A separate fresh database omitting stale receive still
reaches that deadline. These are recorded as timeout/inconclusive execution,
not syntax rejection or proof that receive is unavailable. Three-second
recreation and single-message reads return data near the requested three
seconds; corresponding Omni first-row reads complete much sooner in this
fixture. Internal buffering/streaming causes were not established.

The [fresh single-message lease control](evidence/cross-engine-doc-delta-20261029/emulator-queue-single.jsonl)
exercises the previously unreached parameter forms. Emulator accepts default
scalar-inside-array and whole-array forms; explicit AUTO_PARAMETER=OFF rejects
as an unsupported hint. Managed/Omni reject the former form and accept the
whole-array form. Emulator also returns a renewal row for an intentionally
invalid token. Successful rows alone do not establish equivalent lease-state
validation. Its rollback checks preserve rows as expected; intentional
client rollback sentinels with code Unknown are not transport failures.

Actual child heartbeat counts are 20/20 for r5, 19/19 for LTS and 18/18 for
Emulator, GoogleSQL/PostgreSQL respectively. Median record gaps are about
100-107 ms. These bounded samples do not promise timing, delivery guarantees,
replication RPO/RTO or queue reset delay.

## Attempts, provenance and cleanup

Each image records the same 700 planned core steps, including setup and negative
controls. Later runs repeat the retained JSON/stream fixtures, queue lifecycle
and generation cases, and identical ordering controls. These are not 2,100
successful SQL statements: source-level unsupported cases and dependencies
remain explicit.

One generic adapter initially omitted choose=true from a parameterized
conditional. The unchanged parameter-aware producer repeats the original
125-case corpus; that corrected result supplies the comparison for this case.
Both attempts remain visible. Emulator's 16-graph limit caused nine initial
DDL capacity failures. A fresh database with fewer graphs executes the blocked
cases; quota errors are not counted as grammar or feature rejection.

All **37 owned temporary database lifecycles** have successful close/drop
and GetDatabase NotFound records. All **15 task runtime containers** are
confirmed absent, using active inspections or recovered daemon events.
Per-command broad daemon subset flags include helper-container churn and are
not used as a preservation attestation. Cleanup targets only task-owned
resources. The managed trial/database were not touched.

All 123 measured implementation/dependency hashes still match the managed
reference. The original 1,193 research evidence hashes are preserved. Direct
SDK calls bypass local memefish/GoogleSQL frontend analysis. No implementation,
repository dependency, runtime pin, catalog selection, commit, push, PR,
release or external synchronization changes were made.

The [validation receipt](evidence/cross-engine-doc-delta-20261029/validation.json)
records offline evidence/privacy assertions, module tests and OKF/language/
whitespace gates. These local checks do not convert unsupported features,
ordered-result discrepancies or the two retained receive timeouts into passes.

## Emulator open-issue follow-up, October 30 JST

A read-only GitHub audit retrieved all **70 open issues and 112 comments**,
excluding nine pull requests. REST pagination and the GraphQL open-issue total
agree. The [audit and complete inventory](evidence/emulator-open-issues-20261030/audit.json)
link to the retained full issue bodies/comments and refreshed release sources.
This is a source audit against the earlier runtime evidence; no new runtime
execution or upstream issue/comment was performed.

| Open issue | Relationship to this comparison |
| --- | --- |
| [#386: routine metadata](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/386) | Directly consistent with the 1.5.59 catalog: ROUTINES, ROUTINE_OPTIONS and PARAMETERS are absent. The queue ROUTINES query also rejects. TABLE_OPTIONS is a separate observed absence outside this issue's title. |
| [#300: query hints](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/300), [#349: startup bypass](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/349) | Related to unsupported AUTO_PARAMETER validation. The #349 comment records a per-query spanner_emulator.ignore_unknown_hints workaround while still requesting a startup flag. That workaround was not exercised here. |
| [#143: SQL ordering](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/143) | Similar symptom, unverified identity. The report concerns SQL ORDER BY plus LIMIT on 1.5.11; a later commenter could not reproduce it. Our discrepancy is GQL without LIMIT, with correct SQL controls. No duplicate or common root cause is established. |
| [#265: graph labels and zero-length paths](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/265) | Related graph scope. Our bounded omitted-lower `{,3}` cases already execute on 1.5.59. The issue's full original query and node/edge label-reuse case were not rerun, so the issue is not treated as resolved or as an explanation of the ordering discrepancy. |
| [#341: search-index metadata columns](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/341) | SEARCH_PARTITION_BY and SEARCH_ORDER_BY are both present in the retained 1.5.59 catalog. Their values and the complete historical report were not revalidated. Open status alone does not establish that those columns are still absent. |
| [#395: mutation defaults](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/395) | Separate high-impact report: partial Update mutations reset omitted DEFAULT columns when a key has a default or generated expression. The report targets 1.5.58; this exact trigger is outside the executed comparison. |

No identical report was confirmed in the current open bodies/comments for the
GQL sort inversion, queue long receive deadlines, stale receive/invalid-token
acceptance, PostgreSQL JSONB comparisons, HLL rejection or GQL column propagation.
This does not establish absence from closed issues, discussions or private
trackers, or identify a new root cause. The retained failed receive attempts
remain inconclusive.

Performance, memory and concurrency reports remain active in
[#389](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/389),
[#396](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/396)-
[#398](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/398),
[#400](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/400),
[#403](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/403)-
[#407](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/407).
The clock-drift part of #389 is explicitly listed as fixed in the official
1.5.59 notes; its remaining costs and proposed fork flags are separate claims.
No new performance or sustained-memory measurement was made, and these reports
are not assigned as the cause of queue timeouts.

A #403 comment claims fixes in **v1.5.60**, but the refreshed official latest
release remains **v1.5.59**, and the public registry has no 1.5.60 tag in this
snapshot. The comment author has author_association NONE in the API response;
the comment is not used as release, support-role or runtime-fix confirmation.

The later [October 30 runtime follow-up](emulator-open-issue-runtime-20261030.md)
executes many of the previously source-only cases. It supersedes their untested
status within its bounded fixtures while preserving this issue snapshot and the
earlier managed/Omni comparison.
