---
type: Research Note
title: "Managed Spanner documentation-delta revalidation, 2026-10-29"
description: "Six mirror commits drive managed foreign-key, JSONB, graph, search and storage probes; independent catalog responses vary across four surfaces and volatile-view enforcement differs from the documented rule."
tags: [spanner, managed, runtime, documentation, foreign-keys, jsonb, graph, search, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: receipt
    resource: evidence/managed-recheck-20261029/receipt.json
    title: "Runtime identity, scope, dependencies and command ledger"
  - id: new-results
    resource: evidence/managed-recheck-20261029/weekly-summary.json
    title: "Current documentation cases, preserved attempts and semantic assertions"
  - id: catalog
    resource: evidence/managed-recheck-20261029/catalog-volatility-summary.json
    title: "Complete per-response tuples and independent queryability observations"
  - id: history
    resource: evidence/managed-recheck-20261029/sources/history-audit.json
    title: "Every intervening commit and document disposition"
  - id: constants
    resource: evidence/managed-recheck-20261029/json-comparison.json
    title: "Comparison with October 23 constant corpora"
  - id: cleanup
    resource: evidence/managed-recheck-20261029/cleanup-summary.json
    title: "Sixteen owned temporary database lifecycles and final private inventory"
  - id: previous
    resource: managed-trial-revalidation-20261023.md
    title: "Previous successful managed verification and source baseline"
  - id: mirror
    resource: https://github.com/apstndb/spanner-docs-mirror/compare/3dcf9df46b1a7bdb261991f0b4197e9b4dda6acf...6d7397601456dc5602c0926d254424f64643a64f
    title: "Pinned mirror interval"
  - id: graph-contract
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements
    title: "Current volatile-view graph option contract"
  - id: foreign-keys
    resource: https://docs.cloud.google.com/spanner/docs/foreign-keys/how-to
    title: "SET NULL referential action and restrictions"
  - id: jsonb
    resource: https://docs.cloud.google.com/spanner/docs/working-with-jsonb
    title: "JSONB comparison and sorting rules"
---

# Managed Spanner documentation-delta revalidation, October 29

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The October 23 flow was repeated on the same dedicated trial. The new cases
confirm `ON DELETE SET NULL` in both dialects and the documented PostgreSQL
JSONB ordering rules. Unbounded graph quantifiers are rejected, while bounded
controls succeed. Volatile-view graphs still accept the default and explicit
FALSE option, contrary to the newly explicit prohibition in the documentation.

Catalog advertisement varies between **598, 609, 613 and 624 SYS columns**.
The additions are BATCH_OPERATIONS and fifteen lock/split statistics columns;
advertisement and queryability do not consistently agree. Three managed
integration tests pass, and the local drift test fails on two unregistered
tables. This is a completed investigation with an explicit failed gate.

The runtime window is **00:30:18-01:29:28 UTC / 09:30:18-10:29:28 JST** on
October 29. The instance remains FREE_INSTANCE / Enterprise / READY in
asia-northeast1, created October 23 with expiry January 21, 2027 and
REMOVE_AFTER_GRACE_PERIOD. No new instance, paid conversion or instance move
was performed. A managed build identifier and fleet transition date are
unavailable. Omni, Emulator and local parser/analyzer compatibility were not
rerun.

## Source interval and coverage

The previous pinned mirror head was `3dcf9df46b1a7bdb261991f0b4197e9b4dda6acf`.
Current main remains `6d7397601456dc5602c0926d254424f64643a64f`, independently
refreshed during this investigation. The six intervening captures are:

| Commit | Commit time, UTC | Changed files |
| --- | --- | --- |
| `09ce8f56bc5d` | October 23, 18:01:59 | 12 |
| `7f3624d74658` | October 24, 18:01:51 | 4 |
| `e626247a85f8` | October 25, 18:01:31 | 15 |
| `46f7b510c2e6` | October 26, 18:02:27 | 4 |
| `bf8fecd08a43` | October 27, 18:02:01 | 13 |
| `6d7397601456` | October 28, 18:01:51 | 15 |

The net comparison has **48 files: 45 document paths and three retrieval
bookkeeping files**. Per-commit inspection covers 48 document paths including
the retired troubleshooting path and two paths whose endpoint bodies are equal
(create-manage-databases and use-pitr). A rename is not a product removal.
All **91 available pinned before/after bodies** and **47 current official
bodies** are retained. All 47 current bodies equal mirror head after removing
mirror frontmatter. Five supplemental official documents and eight live
HTML/feed responses provide additional source context; live response hashes
and extracted code are retained, with original HTML in the durable task area.

Current release HTML, official API and Atom feed still have October 20 as
the latest dated entry. Later documentation edits therefore required direct
inspection. A mirror capture date does not date backend introduction.
The [per-document audit](evidence/managed-recheck-20261029/sources/history-audit.json)
maps every path to execution or an explicit source-only disposition.

## Catalog variation and existing flow

IS remains 48 tables / 310 columns, surface `89b4ab9e231d`, across three
observations. SYS's three initial checks all return 609 columns. The separate
capture immediately afterwards contains 624. Ten later complete responses
contain all four shapes below, without removing or retyping any October 23
tuple:

| Tables / columns | Surface prefix | Additions over October 23 | Later complete-response count |
| --- | --- | --- | --- |
| 51 / 598 | `9bb70c640b69` | None | 2 |
| 52 / 609 | `2a09be7a4a72` | BATCH_OPERATIONS: 11 columns | 5 |
| 51 / 613 | `199d44cedfc1` | Statistics: 15 columns | 2 |
| 52 / 624 | `016061fbe58e` | Both sets: 26 columns | 1 |

BATCH_OPERATIONS has the eleven-column shape already seen for
ACTIVE_BATCH_OPERATIONS. Star, LIMIT 0, count and individual-column probes
alternate between empty success and table-not-found rejection. No populated
batch operation was exercised. Two ten-sample probes reuse one read-only
transaction per sample and retain the same client `Timestamp()` value after
its queries. Metadata counts, BATCH advertisement and queryability can still
disagree within a sample. This does not identify serving binaries, establish
a metadata snapshot contract or prove a rollout cause.

The fifteen added statistics columns are:

- LOCK_WAIT_DISTRIBUTION and LOCK_WAIT_DISTRIBUTION_JSON_STRING on the six
  LOCK_STATS_TOP/TOTAL MINUTE/10MINUTE/HOUR tables. Their advertised types are
  a distribution struct array and STRING(MAX), respectively.
- LATENCY_IMPACT_CORRELATION_SCORE on the three SPLIT_STATS_TOP interval tables,
  advertised as **STRING(MAX)**. A numeric score type is not inferred from its name.

Thirty qualified queries across these fifteen columns produce 26 OK and
four InvalidArgument responses naming an unrecognized column. The TOTAL_HOUR
JSON-string column fails both attempts; TOP_MINUTE distribution and MINUTE
split-score queries mix success and failure. Some lock distribution queries
return rows; split-score queries provide no populated
score interpretation. These exact names are absent from the fetched versions
of the lock/split statistics guides. The new Monitoring lock-latency metric
text is separate source evidence, not proof of metric emission.

The retained producer captures are
[IS 310](../../../survey/infoschem/evidence/managed/20261029T003028Z-89b4ab9e231d.json)
and [SYS 624](../../../survey/spannersys/evidence/managed/20261029T003029Z-016061fbe58e.json).
They do not advance analyzer selectors. Complete alternate shapes and query
results remain in the [catalog summary](evidence/managed-recheck-20261029/catalog-volatility-summary.json).

| Repeated area | Current bounded result |
| --- | --- |
| Uncached integration | Canonical DDL, expression index and UUID pass; drift fails on unknown ACTIVE_BATCH_OPERATIONS and BATCH_OPERATIONS, with empty ordinal/type mismatch sets in that sample |
| Existing feature cases | All 57 GoogleSQL records, 40 PostgreSQL records and 31 labeled queue events retain October 23 outcome codes |
| JSON/distinctness | All 125 JSON and 64 distinctness cases retain codes, values and row types; distinctness remains 49 OK / 15 InvalidArgument |
| GQL column propagation | All 36 positive variants again match expected columns, rows and NULL padding |
| JSON/JSONB search | All 19 automatic/search/base triplets agree on rows; automatic plans use ten search scans and nine base scans; forced range/text cases retain full-scan boundaries |
| HLL | The 32 original cases and 22 corrected empty/NULL/type/precision controls agree with their retained expectations |
| Change Streams | Both dialects emit actual 100 ms child heartbeats; medians are about 100.06 ms, without a timing guarantee |
| Capture modes | Nine INSERT/UPDATE/DELETE data records per dialect; three modes' normalized payloads exactly match October 23 |
| Queues | Both same-name recreation generations return only their new message IDs; owned queue fixtures are removed |

The [integration log](evidence/managed-recheck-20261029/managed-tests.log),
[constant comparison](evidence/managed-recheck-20261029/json-comparison.json),
[feature comparison](evidence/managed-recheck-20261029/feature-comparison.json)
and [repeated documentation cases](evidence/managed-recheck-20261029/doc-delta-summary.json)
retain these scopes independently. Sampled statistics, tokens, costs and
volatile values are not deterministic equality claims.

## New runtime coverage

### Foreign-key SET NULL

Both GoogleSQL and PostgreSQL accept CREATE and ALTER forms, expose SET NULL
in DELETE_RULE rows and preserve the action in exported DDL. Deleting a
parent with DML or an SDK mutation preserves all eight fixture child rows,
nulls their corresponding references and retains other payload columns.
Within a rolled-back transaction the references become NULL; after rollback
the original references remain. CASCADE and NO ACTION controls behave
differently, and an insert with a missing parent still fails enforcement.

For composite references, deletion nulls both referencing fields of a full
match. A partially NULL reference remains unchanged. NOT NULL columns are
rejected with InvalidArgument, key columns with Unimplemented, and a corrected
generated-column control with InvalidArgument. GoogleSQL also rejects an
ON DELETE action combined with NOT ENFORCED. These are first retained cases,
not measured feature introduction dates.

### PostgreSQL JSONB

All 36 type-pair comparisons and seven edge controls match the documented
order. Nonempty containers follow object > array > boolean > number > string >
JSON null; empty array is a special value below JSON null. Arrays compare
length first and objects compare pair count first. The three corresponding
GoogleSQL JSON controls return different results.

Stored ORDER BY matches all 22 expected row IDs. GROUP BY produces twenty
groups, merging numeric 1/1.0 and normalized duplicate object keys. Equality
join count is 25. SELECT DISTINCT followed by COUNT(*) yields nineteen
non-NULL values; there are twenty-one non-NULL rows. Direct COUNT(jsonb),
including COUNT(DISTINCT jsonb), returns Unimplemented. Exploratory MIN/MAX
returns InvalidArgument. Comparison/sorting support does not establish those
aggregate signatures, and these limitations have no measured introduction date.

### Volatile views and path quantifiers

Independent view submissions avoid hiding valid cases behind an unavailable
function. All **24 available-view graph definitions and their reads succeed**.
The matrix covers stable/volatile projection, volatile generated key, unused
volatile projection, UUID expressions in WHERE/JOIN, omitted/FALSE/TRUE
allow flags, and omitted/FALSE/TRUE uniqueness flags for the projection case.
All tested available views produce three graph nodes. The UUID body/join
predicates are always true for these rows; nondeterministic row membership
was not measured. This does not prove
that key validation is performed merely because its option is accepted.

Default/FALSE acceptance for volatile projections and UUID expressions in the
view body contradicts the current documented prohibition. Explicit TRUE works
as well. The simpler explicit FALSE/TRUE outcome agrees with October 23;
the broader matrix is first coverage, not an observed new service transition.
RAND views reject with InvalidArgument, and direct SELECT RAND() returns
Unimplemented. Six dependent RAND-view graphs and their reads remain NOT_RUN;
UUID body/join controls exercise the available volatile-function path.

On a finite three-node cycle, TRAIL, ACYCLIC and ANY SHORTEST each accept
{1,3} and {,3}: six successful cases. ACYCLIC {1,3} returns six endpoint pairs;
the other bounded variants return nine. All nine {1,}/star/plus cases and a
grouped-star case reject with an explicit unbounded-quantifier error.
Reversed bounds reject separately. These outcomes support the documentation's
removal of unbounded examples, without dating a backend removal.

### Numeric and NULL-filtered search

Six numeric predicates return identical rows with automatic, forced prefixtree,
forced logtree and base-table access. All twelve forced index plans actually
use SearchIndexScan. Native NUMERIC tokenization and BIGNUMERIC type lookup
reject. FLOAT64 works with correctly typed bounds and granularity; missing
granularity rejects independently. This is functional/plan evidence, without
measured throughput, index-size or storage savings.

For a NULL-filtered search index, explicit IS NOT NULL, equality and IN
predicates use the index and agree with base controls. Omitting the predicate,
admitting NULL, or selecting only NULL rejects SEARCH, including forced-index
access. Thus an implied non-NULL condition can suffice; dropping the condition
entirely is not supported by this sample. This distinguishes the two adjacent
sentences in the updated guide.

### Columnar storage fields

All six documented columnar fields are advertised and query successfully in
both dialects. They were already in the October 23 598-column SYS capture;
the documentation addition is not a new observed catalog addition. GoogleSQL
returns a zero-valued row named unknown_table_name; PostgreSQL returns no
statistics rows. No active columnar coverage, physical accounting accuracy or
storage saving was established.

### Additional query-tuning examples

The new guide's ANY_VALUE(... HAVING MAX(...)) example executes with an
inline fixture and selects the row associated with the maximum end time;
the HAVING MIN control selects the other row. PostgreSQL generate_series(0,31)
returns all thirty-two integers, with count 32 and sum 496. Additional-parallelism
hint controls execute in both dialects. These six read-only cases add functional
coverage, without a parallelism or performance claim. PostgreSQL uses one
additional owned temporary database, removed after the cases. See the
[GoogleSQL results](evidence/managed-recheck-20261029/tuning-google.jsonl) and
[PostgreSQL results](evidence/managed-recheck-20261029/tuning-pg.jsonl).

## Source-only changes and execution limits

The complete source audit also covers Knowledge Catalog aspects and Dataplex
migration guidance, instance moves/configurations, throughput guidance,
BigQuery/Data Boost Java connection setup, console visualizer/tuning/saved
queries, links and wording corrections. These workflows were not deployed or
benchmarked. Scan/execution hints and parallelism guidance already occur in
inspected base documents; relocation into query-tuning is not a new-feature
signal. Four queue Monitoring metric labels catch up from BETA to GA; queue
product GA was already announced September 17.

Omni backup interoperability, external storage/import and asynchronous
managed-to-Omni DR guidance are source-only. No Omni server, pipeline,
external bucket, networking or cutover was tested; local Change Streams
success supplies no replication RPO/RTO guarantee. The current search/Data
Boost snippets repair the earlier API/mirror extraction corruption; full
live code and body hashes are retained separately from runtime evidence.

## Attempts, cleanup and validation

The [attempt summary](evidence/managed-recheck-20261029/weekly-summary.json)
retains 240 initial steps, 83 focused repair steps and 88 independent graph
steps. Across these five runs, 265 steps were submitted and 146 records are
explicitly dependency-blocked. A DDL step can contain several statements;
these are not 411 successful SQL tests. Original unqualified view definitions,
a duplicate constraint name and an incorrectly typed FLOAT64 control remain
visible. Corrected focused results supersede only those intended cases.
RAND rejection is a product result. An initial final-state CLI PATH failure
occurred before a cloud request; the explicit SDK path completed the final
read-only check.

All seventeen accepted temporary database lifecycles have successful Drop and
Get-after-drop NotFound evidence. Final counts for user tables, views, graphs,
streams, routines and models are zero, and the instance lists only the
original dedicated READY GoogleSQL database. No pre-existing resources were
deleted. The trial remains for subsequent verification.

All 123 measured implementation/dependency files and the fifteen reused Go
producers remain unchanged, allowing only fixture dates in producer comparison.
Catalog/integration use Go 1.26.8 with isolated survey Spanner v1.88.0; direct
feature clients use the explicit workspace with v1.91.0. Pre-existing dirty
work is preserved. Source bodies, raw redacted outcomes, reproducer inputs,
command exits, cleanup and derived checks are retained together.

The [validation receipt](evidence/managed-recheck-20261029/validation.json)
records offline evidence/privacy checks, OKF all, Markdown language and
whitespace gates. The [module test receipt](evidence/managed-recheck-20261029/module-validation.json)
also records successful survey `GOWORK=off go test -short ./...` and tools
`go test ./...`; container integration is excluded from these local checks. Those local checks do not turn the failed managed drift
gate into a pass. Existing evidence hashes remain unchanged. No implementation,
selector, commit, push, PR, release or external Knowledge Catalog synchronization
was performed.


## Later container comparison

The [October 29 UTC / October 30 JST follow-up](cross-engine-doc-delta-20261030.md)
reuses this managed snapshot and executes its cases on exact Omni r5,
r4-lts.1 and Emulator 1.5.59 images. It records LTS gaps, Emulator JSON
improvements and ordered-result/search/queue differences independently of
this report's managed observation window. No additional managed request is
part of that comparison.
