---
type: Research Note
title: Managed Spanner revalidation and query-statistics documentation, November 5
description: Completed managed metadata and fixture controls, newly documented query statistics, volatile-view contract reproduction, and a separately verified catalog repair.
tags: [spanner, managed, runtime, documentation, statistics, query, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: previous
    resource: managed-revalidation-and-omni-differences-20261103.md
    title: November 3 managed baseline and completed owned database lifecycles
  - id: source-audit
    resource: evidence/managed-recheck-20261105/sources/document-audit.json
    title: Complete two-commit and two-document audit
  - id: query-statistics
    resource: https://docs.cloud.google.com/spanner/docs/introspection/query-statistics
    title: Current query-statistics columns and percentile guidance
  - id: routing
    resource: https://docs.cloud.google.com/spanner/docs/leader-aware-routing
    title: Current geo-partitioning-aware routing description
  - id: receipt
    resource: evidence/managed-recheck-20261105/complete-receipt.json
    title: Completed runtime scope, authorizations and three owned database cleanups
  - id: assertions
    resource: evidence/managed-recheck-20261105/assertions.json
    title: Bounded observations, value comparisons and negative controls
  - id: catalog-update
    resource: evidence/managed-recheck-20261105/catalog-update/validation-receipt.json
    title: Final catalog branch source, projections and managed verification
  - id: catalog-push
    resource: evidence/managed-recheck-20261105/catalog-update/push-receipt.json
    title: Verified local and remote branch HEAD
  - id: validation
    resource: evidence/managed-recheck-20261105/complete-validation.json
    title: OKF, Markdown-language and whitespace validation
---

# Managed Spanner revalidation, November 5

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The November 5 managed revalidation is complete: **270 recorded steps and
422 bounded assertions**, including negative controls. All three authorized
owned databases have been deleted and independently return GetDatabase
NotFound. The initial two-database run retains its failed strict-name view
fixture; a separately authorized, corrected twelve-step fixture completes the
volatile-view controls.

IS remains 48 tables / 310 columns. Three initial SYS metadata samples remain
52 tables / 624 columns, matching the November 3 tuples. The initial source
checkout's local catalog drift test fails. The separately requested catalog
patch then registers the complete current superset, and its managed drift and
canonical DDL tests pass. These before/after code states are recorded separately.

## Target, producers and comparison boundary

The target is the existing dedicated user-level mise profile. Connection
destinations remain outside retained evidence. SELECT 1 succeeds through the
same direct SDK query producer used for the remaining cases; local parser
acceptance cannot intercept the service response. No managed backend build
identifier is available. Exact observation times and query producer hashes
are retained in the per-phase command receipts.

Initial metadata reads use the existing base database. The explicitly authorized
fixture lifecycles use fresh GoogleSQL/PostgreSQL databases and clean up only
current successfully created databases.
Omni, Emulator, local parser compatibility, authenticated Omni clients and
cross-region routing are not rerun. Scorer, indexed search, queue
TVF, SET NULL and volatile-view observations below are fresh managed executions.
The catalog implementation/selector update is handled in an isolated worktree
and branch; source and execution evidence are not conflated with the original
checkout. No public issue, merge, release or earlier retained database deletion
is performed.

## Documentation interval and release freshness

The pinned source interval is
`ec91ce497a621dd3d318cad40a58944367415165` through
`293fe78b6da4b70bfe8f693161c5fc784f0364a8`:

| Capture commit | Scope |
| --- | --- |
| `28532761a37ec33c986bc1c75a10f25fb22a0e9a` | November 3, 18:02:04 UTC; both document changes and retrieval bookkeeping |
| `293fe78b6da4b70bfe8f693161c5fc784f0364a8` | November 4, 18:01:46 UTC; retrieval metadata only |

Net changes cover **two document paths and two bookkeeping paths**. Both
documents' pinned before/after bodies and current full official API bodies
are retained. Both current bodies match mirror target after frontmatter
removal. The current release API body, live HTML and Atom feed all still
identify **October 30** as the latest managed announcement. The separate
graph-schema, ranked-search and search-index contracts also retain their
previous update metadata and wording in this retrieval.

The query-statistics guide adds five fields to both TOP and TOTAL descriptions:

- `AVG_COLUMNAR_READ_SHARE` as FLOAT64, a fraction between zero and one.
- `CPU_SECONDS_DISTRIBUTION` as a single-element distribution STRUCT array,
  and `CPU_SECONDS_DISTRIBUTION_JSON_STRING` as its string representation.
- `ROW_SCANNED_DISTRIBUTION` and its JSON-string counterpart.

Across TOP/TOTAL and MINUTE/10MINUTE/HOUR these are **30 column occurrences**.
All 30 were already advertised in the retained October 29 624-column capture;
this documentation update does not date a new service schema rollout.
The PostgreSQL guidance excludes the STRUCT-array columns and directs readers
to their JSON-string counterparts. The fresh PostgreSQL fixture confirms eighteen scalar/JSON columns across six
tables, executes JSON percentile calls, and rejects the STRUCT column reference.

The guide also fixes a PostgreSQL summary example so that a TOTAL table's
latest interval is selected from that same TOTAL table, rather than TOP.
The other document clarifies leader-aware routing with geo-partitioning:
routing considers all database placements, including placements unused by a
transaction; enabled routing considers placement leader regions, disabled
routing considers serving regions. This is **source-only** on the current
regional test target; no multi-placement network-routing experiment is claimed.

## Current managed observations

All fifteen bounded pipe-operator examples execute. Grouping aliases work;
ordinal grouping and a missing input alias reject with the same semantic
errors as before. Unique-key pipe ordering retains its expected sequence.
Unordered LIMIT compares counts only. JSON equality/order/grouping/coercion,
CAST, distinctness, HLL and approximate distinct count retain their earlier
values and result types. RAND remains explicitly unsupported.

The batch-operation tables support empty counts and scalar projections. The
newer lock-distribution and split-correlation columns remain referenceable;
populated TOTAL lock statistics and empty TOP/split samples remain distinct.
The nested-result `SELECT * ... LIMIT 0` lock control still returns the same
unsupported STRUCT-result-shape error; it does not establish table absence.
TOP scalar probes use the correct `LOCK_WAIT_SECONDS` name in this run.

The existing `TestDrift_RealTableMetas` fails again on its old accepted surface,
unknown `ACTIVE_BATCH_OPERATIONS`/`BATCH_OPERATIONS`, twelve lock columns and
three split columns. Ordinal/type mismatch sets are empty. This retained
failure is a local registry maintenance gap; passing evidence assertions do
not make the integration gate green.

## Newly covered statistics execution

All six query-statistics tables return a populated scalar/JSON distribution
sample. Observed columnar-read fractions are in the documented range; the
small test workload does not establish active columnar execution, performance
or a service-wide distribution of that fraction.

All twelve JSON-string percentile calls execute. Paired calls in each table
for both CPU and rows-scanned distributions execute with **identical** values
when using the first STRUCT element and the JSON string in the same query:

```sql
SELECT
  SPANNER_SYS.DISTRIBUTION_PERCENTILE(
    CPU_SECONDS_DISTRIBUTION[SAFE_OFFSET(0)], 99.0) AS p99_struct,
  SPANNER_SYS.DISTRIBUTION_PERCENTILE(
    CPU_SECONDS_DISTRIBUTION_JSON_STRING, 99.0) AS p99_json
FROM SPANNER_SYS.QUERY_STATS_TOTAL_MINUTE
LIMIT 1;
```

Passing the **whole ARRAY** directly yields a signature error in twelve
negative controls. The service advertises STRUCT and STRING overloads, not
an ARRAY overload. The element-extraction control resolves this input-shape
boundary; the generic documentation notation `distribution` alone is not a
complete SQL argument example.

Some observed histogram JSON has negative entries in `BUCKET_COUNTS`. An
exact observed JSON input successfully executes as a constant, whereas an
attempt to expand those entries into literal zero buckets is rejected. A
separate synthetic JSON construction is also rejected. Those exploratory
negative controls do **not** establish corrupt negative sample counts, a
portable expanded representation or a documented compression contract.
Use the service percentile function for these retained samples rather than
summing or expanding the encoded array based on an inferred representation.

## Completed fixture observations

The initial fixtures execute 77 GoogleSQL and 50 PostgreSQL steps. Each dialect
reconfirms three automatic/forced/base JSON search triplets and ON DELETE SET
NULL. Scorer v1/v2 values remain distinct; database defaults and explicit query
versions retain their earlier precedence. PostgreSQL database score_version
settings still reject while query-level versions execute. The PostgreSQL hourly
bucket's fractional/pre-epoch discrepancy and floor-based control agree with
November 3. GQL propagation and RETURN ordering retain their expected values;
the corrected standalone and NEXT/LIMIT shapes execute and select IDs 1,2.

The initial view declaration used an unqualified Id in strict view resolution,
so it fails before graph creation and leaves nine dependent cases NOT_RUN.
That is a producer setup error, not evidence about volatile-view enforcement.
After qualifying n.Id in the view definition, the separately authorized repair
database completes all twelve steps. Default, FALSE and TRUE
allow_view_with_volatile_column settings all create graphs and return three
valid UUID-valued node properties. Every UUID value changes between the paired
reads even with FALSE, reproducing the documented-prohibition discrepancy
beyond creation-time acceptance. Both the initial failure and corrected results
remain retained.

Exactly three database lifecycles are recorded: the originally approved two,
then one separately approved repair database. All three report successful DROP
and subsequent GetDatabase NotFound. The earlier interrupted database is excluded.

The local catalog patch is verified separately against the same managed profile:
53 registered tables / 632 known columns cover the 52-table / 624-column sample,
with eight known absent columns and no name, type or ordinal drift. The current
INFORMATION_SCHEMA projection selects a new 310-column capture. SYS's analyzer
common projection remains explicitly pinned to its matching historical
managed/Omni inputs; recognizing the live superset does not fabricate newer
cross-engine evidence. The catalog branch
[fix/managed-catalog-20261105](https://github.com/apstndb/spanalyzer/tree/fix/managed-catalog-20261105)
is pushed at `99ab0125949f842fcb98c816850140e1117746a4`; the local and remote heads match.
All five module tests (survey with -short), generation checks, OKF/Markdown
checks and the mandatory pre-push gate pass. The gate required Go 1.26.10,
x/net v0.60.0 and OpenTelemetry SDK v1.45.0 with their minimum coordinated
dependencies. Original failures remain retained. Catalog verification and push
receipts are copied alongside this report, with full logs in the durable
catalog-update task area. The original dirty research checkout is preserved.

All **2,234 artifacts from the completed read-only phase** are preserved,
including the earlier 2,159 artifacts. New official
sources, redacted runtime records, command receipts, producer sources and
assertions are retained under `evidence/managed-recheck-20261105/`. The final
validation receipt records the OKF, Markdown-language and whitespace checks.
