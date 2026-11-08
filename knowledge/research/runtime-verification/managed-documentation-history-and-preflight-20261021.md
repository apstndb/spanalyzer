---
type: Research Note
title: "Managed Spanner documentation history and incomplete preflight, 2026-10-21"
description: "Ten mirror commits and current primary sources reveal GQL, JSON search and HLL documentation changes; managed runtime revalidation stops at an unavailable instance."
tags: [spanner, managed, documentation, gql, json, hll, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: receipt
    resource: evidence/managed-recheck-20261021/receipt.json
    title: "Bounded execution status and preserved source identity"
  - id: history
    resource: evidence/managed-recheck-20261021/history-audit.json
    title: "Per-commit and per-document dispositions"
  - id: sources
    resource: evidence/managed-recheck-20261021/documentation.json
    title: "Full official retrieval receipts and live HTML/feed checks"
  - id: mirror-history
    resource: https://github.com/apstndb/spanner-docs-mirror/compare/27afee67642907f3b4537fa9a22d7a5726dc504e...9f3859b9ea131f481bca92225c4cd4c020d5885b
    title: "Pinned documentation history since the October 11 retrieval"
  - id: release
    resource: https://docs.cloud.google.com/spanner/docs/release-notes#October_20_2026
    title: "October 20 approximate aggregate announcement"
  - id: prior-runtime
    resource: managed-revalidation-20261011.md
    title: "Last successful retained managed runtime flow; historical"
---

# Managed Spanner documentation history and preflight, October 21

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Current documentation contains further changes, including the October 20
approximate aggregate announcement, but **managed runtime revalidation is
incomplete**. The configured dedicated target returns `NotFound: Instance not
found` through both the existing CLI and a direct Spanner SDK query. No current
catalog, integration or feature result can be inferred from that failure.

The last successful retained runtime flow remains
[October 11](managed-revalidation-20261011.md). Its catalog counts, feature
outcomes and known drift failure are historical observations, not results
reconfirmed on October 21. Omni and Emulator were not run in this investigation.

## Execution boundary

The CLI preflight ran October 21, 09:45 UTC / 18:45 JST. A separately built
direct SDK probe ran at 09:46 UTC / 18:46 JST. Both sent `SELECT 1 AS ready`.

| Check | Observed result | Interpretation |
| --- | --- | --- |
| Existing managed CLI | Exit 1; instance `NotFound` | Environment/resource failure before feature testing |
| Direct Go SDK client | Process exit 0; RPC code `NotFound`, no rows | The producer recorded an error; exit 0 is not SQL success |
| Catalog comparison and uncached integration | Not run | No current catalog or local integration assessment |
| SQL, PostgreSQL, queues, JSON, distinctness, graph and Change Streams | Not run | Existing runtime findings remain dated October 11 or earlier |
| Owned fixture creation and cleanup | No DDL or database creation started | No owned resources require cleanup |

[Command receipts](evidence/managed-recheck-20261021/commands.json), the
[redacted CLI log](evidence/managed-recheck-20261021/preflight.log), and the
[direct RPC record](evidence/managed-recheck-20261021/direct-connectivity.jsonl)
retain this distinction. The direct client uses Go 1.26.8 and Spanner SDK
v1.91.0 in the current workspace; the isolated survey integration configuration
uses its separate dependency resolution and was not executed today.

The local source is commit `c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c` plus
pre-existing dirty/untracked work, preserved in place. All 123 measured
implementation and dependency file hashes match the October 11 producer
snapshot; this is neither a clean-checkout claim nor a server identity.
See the [source comparison](evidence/managed-recheck-20261021/source-comparison.json).

An available managed target must be configured in the existing dedicated
verification profile before resuming. Credentials and target coordinates are
omitted from retained evidence. No instance was created and no alternative
target was selected.

## Commit history coverage

The review starts at `27afee67642907f3b4537fa9a22d7a5726dc504e`, the October 10
18:01:41 UTC snapshot available to the previous October 11 retrieval, and ends
at `9f3859b9ea131f481bca92225c4cd4c020d5885b`, October 20 18:02:13 UTC.
These are mirror commit times, not source publication or engine rollout dates.

All ten subsequent commits were inspected, including their complete file
lists. The net comparison has 52 files: 49 document paths, two logs and
metadata. The union of document paths across the individual commits is 50,
because the ML tutorial disappears and returns between the endpoints. Full
base/head retrieval retained 95 available Markdown bodies; all fetches
succeeded. See the [retrieval receipt](evidence/managed-recheck-20261021/history-retrieval.json)
and [commit details](evidence/managed-recheck-20261021/commit-details.json).

| Mirror commit date, UTC | Commit | Reviewed change or disposition |
| --- | --- | --- |
| October 11 | `c43d0e44bfdc` | Read-only replica availability/DR clarification and sample-code corrections |
| October 12 | `701dbac105ca` | Metadata only; no document path changed |
| October 13 | `9f61c5ebbf19` | Eight HTML-table formatting changes and cutover guidance |
| October 14 | `3130aa10f2e0` | JSON search equality examples and MySQL migration/cutover/fallback restructuring |
| October 15 | `65cf94c526d0` | IAM permission table and operational prose; temporary ML tutorial removal |
| October 16 | `202e22e47692` | Dynamic JSON graph properties, analytics guidance, hash prose, Omni policy; ML tutorial restored |
| October 17 | `15aa44be9ecc` | Geo-partitioning limitation bullets edited |
| October 18 | `b4e61272d205` | Metadata only; no document path changed |
| October 19 | `a1c98d849cc9` | Metadata only; no document path changed |
| October 20 | `9f3859b9ea13` | GQL set-operation column propagation modes |

The 50 path dispositions are: 35 substantive/editorial, four added, one
removed, two body-equal and eight HTML-table-format-only. Formatting-only
INFORMATION_SCHEMA changes do not establish catalog changes. The restored
ML tutorial is byte-identical at the pinned endpoints; its intermediate
absence does not establish product deprecation. Generated timestamps and
Markdown table padding are not new SQL features.

## Current official sources extend beyond the mirror

Full official Developer Knowledge retrieval covers 61 unique documents,
including all current changed mirror pages and earlier relevant references.
Three additional approximate-aggregate/HLL/sketch pages were recovered,
bringing the unique current official-document total to 64. Live release
HTML/feed, PostgreSQL function HTML and GoogleSQL operator HTML were also
fetched in full. Raw bodies were inspected directly.

Of 49 current changed mirror pages compared after removing mirror frontmatter
and trimming boundary whitespace, 46 agree with current official API bodies
and three differ. The separately checked release page has a further freshness
gap. See the [body comparison](evidence/managed-recheck-20261021/mirror-vs-current.json).

| Source | Current primary-source observation | Mirror/API boundary |
| --- | --- | --- |
| Release notes | Live HTML and Atom feed both contain October 20, 2026 approximate aggregates for GoogleSQL and PostgreSQL | Pinned mirror and current Developer Knowledge release body both stop at September 18; the API reports October 1 updateTime |
| PostgreSQL functions | Approximate count and HLL functions; official updateTime October 20 16:05:28 UTC; live HTML agrees | Approximate section absent from pinned mirror |
| GoogleSQL function index | Five added names: APPROX_COUNT_DISTINCT and HLL_COUNT.EXTRACT/INIT/MERGE/MERGE_PARTIAL; updateTime October 20 16:01:44 UTC | Names absent from pinned mirror; table re-padding is incidental |
| Geo-partitioning | Additional partition cannot move to a different instance configuration; updateTime October 20 13:45:52 UTC | Restriction absent from pinned mirror |

The mirror's failure log also lists approximate aggregate functions,
HyperLogLog++ functions and sketches despite HTTP 200 responses. All three
pages are retrievable now through the official API with October 20 updateTime.
These observations establish source divergence, without diagnosing its cause.
Checking only mirror commits or only the API release page would miss the
current announcement. Retained release entries are
[available separately](evidence/managed-recheck-20261021/release-entries-since-baseline.json).

## Changes requiring bounded runtime follow-up

| Area | Documented behavior now inspected | Required distinction during execution |
| --- | --- | --- |
| GQL set operations | FULL OUTER/FULL/OUTER, LEFT OUTER/LEFT and INNER column propagation before UNION/INTERSECT/EXCEPT ALL/DISTINCT; default exact-name matching remains | Verify acceptance, result names/types, NULL padding and mismatched-column rejection independently |
| JSON/JSONB search indexes | Equality on whole documents or subpaths, including =, !=, <>, IN and NOT IN; ranges are not accelerated; PostgreSQL ->> extraction is outside this acceleration | Successful SQL is not evidence of search-index use; inspect plans against forced/base-table controls |
| Dynamic graph properties | JSON comparison/filter/group/order/distinct examples; explicit conversion for boolean predicates, arithmetic and typed consumers | Compare values and row metadata, including JSON strings versus numbers and invalid bare JSON predicates |
| Approximate count and HLL | APPROX_COUNT_DISTINCT and sketch init/extract/merge/merge_partial for both dialects | Check input types, precision boundaries, NULL/empty cases and partial aggregation; do not infer speed or accuracy guarantees from a small fixture |

The [GQL source](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-query-statements),
[JSON index guide](https://docs.cloud.google.com/spanner/docs/full-text-search/json-indexes),
and [dynamic graph guide](https://docs.cloud.google.com/spanner/docs/graph/manage-schemaless-data)
describe the first three rows. Their changes remain documentation evidence
until the corresponding engine calls execute.

The [HLL reference](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/hll_functions)
documents GoogleSQL INIT inputs INT64, NUMERIC, STRING and BYTES, precision
10 through 24 with default 15, and matching underlying input types for merges.
Different precisions merge at the lowest precision. EXTRACT(NULL) is 0;
INIT of all NULL inputs and MERGE_PARTIAL of empty/all-NULL input are NULL;
MERGE of empty/all-NULL input is 0. The
[PostgreSQL reference](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions#approximate-aggregate)
documents the corresponding functions and says its numeric sketches cannot
merge with GoogleSQL NUMERIC sketches. These are source-backed contracts,
not observed October 21 results.

The September 29 managed baseline already executed a small
APPROX_COUNT_DISTINCT case. The October 20 announcement therefore does not
date its first runtime availability on the retained target. Performance
recommendations for columnar execution are not benchmark results here.

A [136-case draft plan](evidence/managed-recheck-20261021/doc-delta-probe-plan.json)
contains 32 approximate/HLL, 40 GQL propagation, 57 JSON search, six dynamic
property and one hash-control cases, with proposed fixtures. Every case is
explicitly unexecuted. Review the draft SQL and fixture ownership before
execution; additional mixed-type/precision and cross-dialect sketch controls
remain listed as future coverage.

## Source-only operational context

The complete path audit also covers changes that need distinct environments
or permissions rather than routine SQL fixtures:

- Read-only replicas are explicitly insufficient as standalone disaster
  recovery when the voting quorum/instance is unavailable, including stale reads.
- Graph analytics documents refine fallback-region jurisdiction, CALL versus
  CALL PER(), parameter constraints and centrality output names. No analytics
  job was provisioned or executed.
- MySQL migration, cutover, fallback/reverse replication and DLQ guidance is
  expanded/reorganized; no migration was performed.
- IAM predefined-role permission tables and geo-partitioning limitations were
  edited; neither permissions nor provisioning were exercised.
- Hash-function guidance adds use-case/security prose without new signatures.
  Sample corrections and compaction/launch guidance do not establish SQL changes.
- Omni LTS/STS support and upgrade policy documents changed. They are context
  only; there is no new Omni runtime result in this managed investigation.

GoogleSQL operators still have no IS DISTINCT FROM / IS NOT DISTINCT FROM
entry in either the full current API body or live HTML. The October 11
distinctness matrix remains the retained execution evidence; today's source
check neither reconfirms nor revokes that runtime behavior.

## Retention and resumption

Canonical evidence is under
[managed-recheck-20261021](evidence/managed-recheck-20261021/receipt.json).
Full raw official bodies, HTML/feed and pinned mirror base/head bodies are
retained in `<LOCAL_TASK_ROOT>/apstndb/spanalyzer/managed-recheck-20261021`,
with retrieval hashes in the canonical receipts. Keep these until explicitly
cleaned up. The shared artifact manifest records integrity, not execution.

After the dedicated target is updated, preserve the initial NotFound receipts
and use a separately named preflight attempt. Then run catalogs, uncached
integration, the earlier bounded feature/queue/JSON/distinctness/graph/Change
Streams corpus, reviewed new cases, and owned-fixture cleanup. Append an
execution receipt and history row rather than reinterpret this incomplete
attempt as a successful recheck. Catalog selectors are unchanged.

Local validation passes: 196 offline source/integrity and scope checks, all
four OKF gates, the Markdown language check and `git diff --check`. The offline
check verifies all 674 pre-existing retained artifact hashes are preserved.
These checks validate the research bundle, not managed SQL execution. The
[validation receipt](evidence/managed-recheck-20261021/validation.json) records
the commands and exits; the mirror main refresh still matches the reviewed head.
