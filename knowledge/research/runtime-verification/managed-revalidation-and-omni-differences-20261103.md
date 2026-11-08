---
type: Research Note
title: Managed Spanner revalidation and Omni differences review, November 3
description: Current managed metadata and documentation-driven execution, completion of earlier scorer and search controls, and a source audit of Omni feature and operational differences.
tags: [spanner, managed, spanner-omni, documentation, runtime, search, gql, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: previous-runtime
    resource: documentation-runtime-and-memefish-audit-20261031.md
    title: October 31 runtime samples and unfinished managed controls
  - id: previous-managed-flow
    resource: managed-documentation-delta-revalidation-20261029.md
    title: Earlier complete managed flow and variable system catalog surfaces
  - id: differences
    resource: https://docs.cloud.google.com/spanner-omni/differences
    title: Current official differences between Spanner and Spanner Omni
  - id: omni-release
    resource: https://docs.cloud.google.com/spanner-omni/release-notes
    title: Official Omni r5 and r4 LTS patch announcements
  - id: managed-release
    resource: https://docs.cloud.google.com/spanner/docs/release-notes
    title: Managed JSON and JSONB announcement dated October 30
  - id: source-audit
    resource: evidence/managed-recheck-20261103/sources/document-audit.json
    title: Complete three-commit and seventeen-document disposition
  - id: runtime-receipt
    resource: evidence/managed-recheck-20261103/receipt.json
    title: Runtime scope, owned database cleanup and preserved evidence
  - id: assertions
    resource: evidence/managed-recheck-20261103/assertions.json
    title: Bounded value, metadata, error and lifecycle assertions
---

# Managed Spanner revalidation and Omni differences review

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The current managed sample retains the previously observed JSON/pipe behavior
and 310/624-column catalog surfaces. Previously incomplete scorer and index
controls now execute. Local catalog drift remains a failing gate. Current
official sources explicitly announce Omni r5, while the differences page needs
an edition/worker qualification for its high-scale ANN exclusion.

## Scope and target identity

The managed target is the existing dedicated user-level mise profile. Connection
destinations are not retained. Service queries use the direct Go SDK, bypassing
local parser classification. No managed build identifier is available; November
3 sampling times are observations, not a fleet rollout date.

Read-only preflight begins **01:34 UTC / 10:34 JST**. The two fixture runs cover
**01:37:14-01:47:19 UTC / 10:37-10:47 JST**. The scratch module uses Go 1.26.8
and Spanner SDK v1.91.0; binary build metadata and source hashes are retained.

The user explicitly authorized two disposable databases, one GoogleSQL and one
PostgreSQL, and deletion of those newly created databases after verification.
The database retained after the October 31 interruption is excluded. There is
no new instance, paid conversion, backup, Kubernetes deployment, Commercial
worker, public post, commit or push in this investigation.

This is a managed execution and official-documentation recheck. Omni and
Emulator results used for comparison retain their earlier observation dates
and exact image identities. Neither engine is executed again here. The full
October 29 heartbeat, capture-mode, queue-recreation and integration corpus is
not repeated; current coverage is enumerated in the retained case plans.

## Documentation interval

The prior source baseline is
`1e880188144a326e383778ebc165a63197efbf70`. Current inspected mirror main is
`ec91ce497a621dd3d318cad40a58944367415165`.

| Capture commit | Commit timestamp, UTC | Reviewed scope |
| --- | --- | --- |
| `5bd8ea59e865da65fa7fdb6cf9c2dbb4f81a7bba` | October 31, 18:01:38 | Thirteen documents: Omni r5/chart examples, LTS patch announcement, managed JSON announcement, TVF Monitoring metrics and scorer-pagination guidance |
| `6b454c1d25412f11ac9dc631ee9cb079587a450f` | November 1, 18:01:39 | Five documents: r5 announcement, asymmetric-autoscaling edition, Java dependencies and backup sample timeout |
| `ec91ce497a621dd3d318cad40a58944367415165` | November 2, 18:02:05 | Retrieval metadata only |

The net interval changes **17 document paths and two bookkeeping paths**.
All 34 before/after document bodies, both unchanged differences bodies and
three complete commit records are retained. Five current full official bodies
match the corresponding mirror target after frontmatter removal. The
[per-path audit](evidence/managed-recheck-20261103/sources/document-audit.json)
records a disposition for every changed document.

The latest managed release entry is **October 30: JSON/JSONB comparisons,
ordering/grouping, GoogleSQL casts/coercion and JSON search-index equality**.
Those capabilities had already been observed in earlier runtime samples;
this announcement does not date their backend introduction.

The search-function reference newly warns that scorer version changes can
change raw scores and relative rank, with a transient missing/repeated-result
risk when pagination crosses a default-version switch. Explicit version or
database pinning is the documented mitigation. This run tests present version
selection; it does not simulate a future automatic switch or pagination event.

The metrics page adds BETA `api/tvf_request_count` and
`api/tvf_request_latencies`, with 60-second sampling and up to 180-second
visibility delay. No Monitoring descriptor or emitted time series is tested.
Other changes are source-only: Helm 1.1.0 examples, Java BOM 26.90.0/Spanner
6.125.0, an explicit Enterprise Plus setting in asymmetric-autoscaling samples,
and Java backup wait examples changing from 1,200 to 3,600 seconds.

## Managed metadata and read-only controls

INFORMATION_SCHEMA remains **48 tables / 310 columns**. Three independent
SPANNER_SYS metadata responses all expose **52 tables / 624 columns**. This
matches one surface already observed on October 29, when 598/609/613/624-column
responses overlapped. Three equal samples do not establish fleet convergence.

`ACTIVE_BATCH_OPERATIONS` and `BATCH_OPERATIONS` both support empty counts and
explicit scalar projections in this run. No populated batch operation is
created. All twelve new lock-distribution columns and three split-correlation
columns can be referenced. Distribution lengths and JSON-string projections
execute, including populated TOTAL_10MINUTE/HOUR rows; split-score projections
are empty and do not establish populated score semantics.

`SELECT * FROM SPANNER_SYS.LOCK_STATS_TOTAL_10MINUTE LIMIT 0` returns an
unsupported result-shape error for a nested STRUCT construction. Scalar and
distribution-length projections execute. That star-query outcome is not
classified as an unavailable system table. The initial TOP scalar probes used
the TOTAL-only name `TOTAL_LOCK_WAIT_SECONDS`; corrected
`LOCK_WAIT_SECONDS` projections all execute. Both attempts are retained.

The existing read-only `TestDrift_RealTableMetas` **fails**: the local accepted
surface still excludes 52/624, with unknown tables `ACTIVE_BATCH_OPERATIONS`
and `BATCH_OPERATIONS`, twelve unknown lock columns and three unknown split
columns. Its ordinal/type mismatch sets are empty. This is a local registry
maintenance gap, not a new schema change first observed today. Catalog
selectors and production definitions are not changed by this research note.

The constant corpus reconfirms JSON equality/order/coercion/CAST, JSON grouping,
distinctness, HLL and approximate distinct count. All fifteen pipe operators
execute, with correct grouping aliases and ordered unique keys; pipe grouping
ordinals and references to absent input aliases reject semantically. RAND
remains explicitly unsupported. Unordered LIMIT asserts count only.

## Scorer, search, PostgreSQL and GQL

| Case family | Current managed observation |
| --- | --- |
| GoogleSQL scorer | Explicit v1/v2 produce different values. Unpinned default matches v2; database setting 1 or 2 changes the default accordingly. Query-level explicit versions remain unchanged across both database settings. |
| PostgreSQL scorer | Explicit query versions execute and differ; default matches v2. Both `ALTER DATABASE ... SET "spanner.score_version" = 1/2` statements reject as unsupported options; dependent pinned-default queries remain NOT_RUN. This is not evidence that the rejected setting takes effect. |
| Search sort keys | Generated non-NULL INT64 second and hour sort keys, and an index with no sort key, all create. The hour bucket stays constant within an hour and advances by 3,600 seconds at the next hour. No write-cost or performance measurement. |
| JSON/JSONB indexed equality | Both dialects run subpath equality, IN and whole-document equality through automatic, forced-search and forced-base plans. Results are compared across all three access paths. Plans are retained independently of result correctness. |
| Queue declaration and TVF aliases | `Payload` and `payload` create; `PAYLOAD` and `PaYlOaD` reject. Empty-token `RENEWLEASE` executes with no alias, AS alias and bare alias, returning no rows. No lease is renewed for a real message. |
| GQL propagation | FULL, FULL OUTER, OUTER, LEFT, LEFT OUTER and INNER return the expected column order, rows and NULL padding. The shared-column LEFT control missing from the October 31 managed completion now executes. |
| GQL ordering | RETURN ORDER BY yields 1,2,3; RETURN ORDER BY LIMIT 2 yields 1,2 for inserted keys 3,1,2. Two initially prepared NEXT controls ended without the required final RETURN and reject at end of statement. Those malformed controls do not establish NEXT ordering behavior; earlier separate evidence remains dated. |
| Enhanced full-text search | Managed `SEARCH(..., enhance_query=>true)` executes, including a `require_enhance_query=true` hint. Both return IDs 1,2 for this fixture. This is successful managed execution, not a test of Omni's documented exclusion or evidence of synonym-expansion quality. |
| PostgreSQL JSONB | Constant ordering returns JSON null, 1, 2 in that order; COUNT(jsonb) remains explicitly unsupported. |

For the first matching search row, explicit scorer v1 is approximately
**0.4101310372**, versus **1.6450823545** for v2. The same values were observed
in the earlier matched Omni fixtures. Raw score changes are demonstrated;
changed relative rank and in-flight pagination discontinuity are not reproduced
by this small corpus.

### PostgreSQL hourly bucket: documentation example boundary defect

The documented form casts fractional epoch seconds to bigint before integer
division:

```sql
EXTRACT(EPOCH FROM t - '1970-01-01Z'::timestamptz)::bigint / 3600
```

It produces these managed values, agreeing with the earlier local engines:

| Timestamp, UTC | Documented expression | Floor-based control |
| --- | --- | --- |
| 1969-12-31 23:59:59 | 0 | -1 |
| 2026-10-31 00:00:00 | 498168 | 498168 |
| 2026-10-31 00:59:59.4 | 498168 | 498168 |
| 2026-10-31 00:59:59.9 | 498169 | 498168 |
| 2026-10-31 01:00:00 | 498169 | 498169 |

The bigint cast rounds fractional seconds; integer division truncates negative
values toward zero. A successful paired control floors after division:

```sql
FLOOR(EXTRACT(EPOCH FROM t - '1970-01-01Z'::timestamptz) / 3600)::bigint
```

This is a bucket-expression problem in a documentation example, not a claim
that PostgreSQL casting or division violates its contract. These PostgreSQL
values are hour indices; the GoogleSQL generated-column example stores an
hour-start epoch-second value, so their units differ.

## Omni differences: current contract and qualifications

The current differences body is **unchanged across this three-commit interval**;
the latest path-history commit is October 3 and the official API update time is
October 1. Those timestamps do not imply that every statement remains complete
for every later edition or release.

| Area | Current official statement | Interpretation and verification boundary |
| --- | --- | --- |
| Deployment and operations | Managed runs in Google Cloud with automatic service maintenance; Omni is self-managed on user infrastructure, with manual updates and no vendor SLA | Operational contract, not established by local SQL success. No deployment, upgrade or availability experiment here. |
| Cloud integrations | Omni excludes BigQuery federation/external schemas/reverse ETL, Gemini database integration, Knowledge Catalog, Data Boost and geo-partitioning | Source-only exclusions; no cloud integration is attempted. Shared SQL behavior does not establish these integrations. |
| SQL/search/graph features | Omni excludes MODEL DDL and ML/AI functions, enhanced full-text search, high-scale ANN and scale-up graph algorithms | These are specific exclusions, not a prohibition on ordinary full-text search, JSON search, vector distance functions or GQL. The ANN wording needs the qualification below. |
| Storage, backups and topology | Omni excludes tiered storage, incremental backups, built-in CMEK integration and automatic dual-region-to-single-region reconfiguration | Source-only. No backup, key, storage or topology operation here. |
| Historical reads | Differences states seven-day managed retention versus thirty-day Omni retention | No retention setting is changed and no thirty-day read is tested. Its shared timestamp-bounds link describes managed one-week retention; that link alone is not independent confirmation of Omni's thirty-day claim. |
| Consistency | Hardware TrueTime for managed versus a software-defined TrueTime API for Omni | No clock uncertainty or external-consistency experiment here. |
| Client/API boundary | Omni lists Java, Go and Python support, with gRPC rather than managed gRPC and REST | A support list is not a claim that every other gRPC client fails. No REST or other-language compatibility probe here. |
| Authentication and authorization | Omni uses password/OPAQUE or client certificates and an internal IAM-like role system without custom roles; managed uses Google Cloud IAM | Keep authentication and authorization separate from earlier insecure local-development SQL probes. |
| Encryption and perimeter | Omni requires infrastructure security, disk/filesystem encryption, configured TLS 1.3 and server mTLS, plus user-managed networking/compliance | Source-only security responsibilities. The page does not imply that local developer-mode probes measured these properties. |

### ANN wording is incomplete across official pages

The differences page lists high-scale ANN as unsupported. However, the
[Omni vector-search guide](https://docs.cloud.google.com/spanner-omni/vector-search-overview#ann)
supports up to one million 128-dimensional vectors without dedicated workers
and larger tables when workers build the indexes. The
[September 30 release entry](https://docs.cloud.google.com/spanner-omni/release-notes#September_30_2026)
explicitly announces Commercial edition ANN with dedicated stateless workers
for up to **one billion vectors**. The editions guide says Developer edition
has no workers while Commercial edition does.

Thus the differences page's broad high-scale exclusion is insufficient as a
blanket statement for current Commercial worker deployments. Preserve this as
a **source inconsistency requiring edition/worker qualification**, not a
runtime-reproduced failure or a measured one-billion-vector capacity claim.

### Release documentation has caught up to the observed images

Current release notes explicitly identify **2026.r5, dated October 29**, and
state incorporation of managed updates through **October 27**. They also
identify **2026.r4-lts.1, dated October 28**, as a security/critical-fix patch.
The download page now lists r5 and Helm 1.1.0 examples.

The October 31 capture first adds the LTS patch announcement, and the November
1 capture adds r5. This supersedes the earlier *source freshness* observation
that r5 was public in the registry but absent from release/download bodies.
It does not change the earlier exact-image test dates, prove older image
removal, turn r5 into an LTS release, or make the managed October 27 cutoff a
claim that later-announced functionality must be absent. Earlier Omni r5/LTS
JSON results remain dated and linked in the
[cross-engine comparison](cross-engine-doc-delta-20261030.md).

## Retention and validation

All **173 planned steps** are recorded: 86 read-only controls, 61 GoogleSQL
fixture steps and 26 PostgreSQL fixture steps. Six additional lifecycle records
prove successful creation and deletion of exactly two owned databases, with
GetDatabase returning NotFound for each. The separate existing drift test is
retained as a failure. Known service rejections and two dependency-blocked
PostgreSQL option checks are not counted as successful feature executions.

All **157 bounded assertions pass**, including all eighteen JSON search plans:
the six automatic and six forced-index cases use SearchIndexScan, while six
forced-base controls use TableScan and return identical matching IDs. The
earlier 2,049 research artifacts are preserved. The prior interrupted database
is not targeted; its separate deletion approval remains pending.

The OKF publication inventory includes this note. Required OKF, Markdown-language
and whitespace checks are recorded in the
[validation receipt](evidence/managed-recheck-20261103/validation.json).
No implementation/dependency/catalog-selector update is included.
