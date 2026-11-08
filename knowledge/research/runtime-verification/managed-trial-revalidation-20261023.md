---
type: Research Note
title: "Managed Spanner trial revalidation, 2026-10-23"
description: "A new authorized trial completes the blocked managed flow, changes explicit volatile-view graph-option outcomes, and executes GQL, JSON search, HLL and queue recreation controls."
tags: [spanner, managed, trial, runtime, gql, json, hll, queues, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: receipt
    resource: evidence/managed-recheck-20261023/receipt.json
    title: "New trial identity, execution scope, dependencies and cleanup"
  - id: comparison
    resource: evidence/managed-recheck-20261023/feature-comparison.json
    title: "Bounded outcome comparison against October 11"
  - id: new-cases
    resource: evidence/managed-recheck-20261023/doc-delta-summary.json
    title: "136 GQL, JSON search, HLL and dynamic-property results"
  - id: history
    resource: evidence/managed-recheck-20261023/documentation-history-audit.json
    title: "Two subsequent mirror commits and current full official sources"
  - id: prior-attempt
    resource: managed-documentation-history-and-preflight-20261021.md
    title: "Historical unavailable target and source freshness gaps"
  - id: previous-runtime
    resource: managed-revalidation-20261011.md
    title: "Last successful flow on the previous target"
  - id: trial-policy
    resource: https://docs.cloud.google.com/spanner/docs/free-trial-instance
    title: "Official free trial conditions and limitations"
---

# Managed Spanner trial revalidation, October 23

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The previously blocked managed flow now completes on a newly created free
trial. The current sample accepts explicit `allow_view_with_volatile_column`
values that returned `Unimplemented` on the October 11 target. Existing JSON,
distinctness, PostgreSQL and queue controls otherwise agree within their
retained scope. New GQL propagation, JSON search, HLL and queue recreation
cases execute. The existing local registry drift failure remains open.

The SQL observation window is October 23, 14:27:00-14:56:29 UTC /
23:27:00-23:56:29 JST. This is a **new regional trial**, not a return to the
old unavailable target. Neither a fleet rollout time nor a managed engine
build identifier is available. Omni, Emulator and local parser/analyzer
compatibility were not rerun.

## Provisioning and identity boundary

The first user-designated project refused trial creation with
`FAILED_PRECONDITION: Free instances are limited to 1 per project lifecycle`.
It had no current instances; deleting an earlier trial does not restore trial
eligibility. The user then explicitly designated another project. That
project was ACTIVE with Spanner API disabled; enabling the API and creating
the requested trial succeeded. An initial display-name validation error and
the lifecycle refusal are retained separately from successful provisioning.

The service reports FREE_INSTANCE, Enterprise edition features, READY,
`asia-northeast1`, creation at 14:24:35 UTC and expiry at
2027-01-21T14:24:35 UTC. Expiry behavior is REMOVE_AFTER_GRACE_PERIOD;
automatic paid conversion is disabled and the default backup schedule is NONE.
The [redacted description](evidence/managed-recheck-20261023/target2-trial-description.json)
is the resource-state evidence. The dedicated user-level profile was updated
with permissions 0600; credentials were not changed and coordinates remain
outside the repository.

The free trial's capacity, regional/edition restrictions and workload differ
from the historical target. These are small functional fixtures, not capacity,
latency, approximation-accuracy or performance benchmarks. Data Boost/graph
analytics jobs and Enterprise Plus geo-partitioning were not provisioned.

Local HEAD remains `c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c` plus preserved
dirty/untracked work. All 123 measured implementation/dependency files are
unchanged from October 21. All twelve existing probe sources match October 11
after fixture-date substitution. Catalogs/integration use Go 1.26.8 with the
isolated survey dependency graph, Spanner v1.88.0; feature clients use the
explicit workspace and Spanner v1.91.0. Queries go directly to Spanner, without
local memefish or GoogleSQL frontend classification.

## Existing flow and changed graph outcome

| Area | Current observation | Comparison boundary |
| --- | --- | --- |
| INFORMATION_SCHEMA | Three agreeing observations; 48 tables / 310 columns, surface `89b4ab9e231d` | Matches retained column tuples; producer/invocation diagnostics do not identify server drift |
| SPANNER_SYS | Three agreeing observations; 51 tables / 598 columns, surface `9bb70c640b69` | Matches October 11; new captures do not advance analyzer selectors |
| Uncached managed integration | Canonical DDL, expression index and UUID pass; drift fails on unregistered ACTIVE_BATCH_OPERATIONS | Three successes and one known local failure, not a passing aggregate gate; canonical coverage is scoped to this fresh database |
| ACTIVE_BATCH_OPERATIONS | Star scans, all eleven columns and count succeed with no active rows; unknown-column control fails | Empty queryability agrees with October 11; no populated batch operation exercised |
| Volatile-view graph options | Both explicit FALSE and TRUE create a graph and return two rows from the volatile view | Changed codes in the new sample; previously Unimplemented |
| Existing PostgreSQL and queue cases | All 40 PostgreSQL records and 31 labeled queue events retain their outcome codes | Sampled rows, workload statistics and plans are not treated as deterministic equality |
| Constant JSON and distinctness | All 125 JSON cases and 64 distinctness cases match codes, values and row types | Transaction timestamps excluded; distinctness remains 49 OK / 15 InvalidArgument |
| Graph conversions | Element/path CAST to JSON remains Unimplemented; corresponding TO_JSON controls succeed | Matches earlier runtime evidence |
| Change Streams | Both dialects emit actual 100 ms heartbeat records and retain tracked columns on option-only ALTER | Twenty child heartbeat records each; medians 100.053 / 100.050 ms are observations, not timing guarantees |
| Value capture | Nine actual data records per dialect: INSERT/UPDATE/DELETE for each of NEW_ROW_AND_OLD_VALUES, NEW_ROW, OLD_AND_NEW_VALUES | Payloads retained independently of heartbeat records |

The graph difference is at 14:31:14 and 14:31:34 UTC: both definitions also
set `validate_element_key_uniqueness=false`. The fixture's volatile non-key
column uses GENERATE_UUID(). Explicit FALSE succeeding does **not** establish
that the option enforces a prohibition; retain the precise definitions and
results instead. The old target cannot be queried concurrently, so this is a
historical-sample difference without a fleet transition date.

Fresh transaction-statistics NULL controls returned no matching rows. That
does not demonstrate repair of the earlier nullable-average decoder failure;
this cold trial lacks a comparable populated NULL-row sample. See
[raw read-only controls](evidence/managed-recheck-20261023/readonly.jsonl),
[integration log](evidence/managed-recheck-20261023/managed-tests.log), and
[constant comparisons](evidence/managed-recheck-20261023/json-comparison.json).

## New documentation-driven execution

All 136 prepared cases were submitted, followed by 22 focused HLL controls.
Expected negative results are retained; a process exit zero alone is not a
successful SQL outcome. See the [new-case summary](evidence/managed-recheck-20261023/doc-delta-summary.json)
and the focused HLL inputs for
[GoogleSQL](evidence/managed-recheck-20261023/hll-controls-google-cases.json) and
[PostgreSQL](evidence/managed-recheck-20261023/hll-controls-pg-cases.json).

GQL FULL OUTER/FULL/OUTER, LEFT OUTER/LEFT and INNER each execute with six
UNION/INTERSECT/EXCEPT ALL/DISTINCT variants. All 36 match expected column
order, result rows and NULL padding. Default mismatched names and INNER with
no common columns fail semantically; two malformed right sides fail syntax.
This is first retained coverage, not an inferred introduction date.

The 57 JSON/JSONB search cases cover nineteen predicates with automatic,
forced-search and forced-base plans. All variants return the same rows.
Automatic plans choose the search index for equality, IN and existence in ten
cases; inequality/NOT IN, ranges and text extraction choose the base table in
the other nine. These decisions use scan_type metadata, because table and
search access can both have the display name Scan. Forced
equality, inequality, IN/NOT IN and existence cases use SearchIndexScan without
its Full scan marker. Forced range conditions in both dialects and PostgreSQL
`->>` text extraction instead show Full scan plus base-table filtering.
Their SQL succeeds: forced-index acceptance alone would miss this boundary.
There is no measured acceleration or large-data optimizer-selection claim.

Both dialects execute approximate distinct count and HLL INIT/EXTRACT/MERGE/
MERGE_PARTIAL. The small duplicate/NULL fixture estimates two. Precision
10/15/24 succeeds; 9/25 returns OutOfRange. INT64/bigint, string/text, bytes
and numeric inputs work; BOOL and INTERVAL INIT inputs fail signature matching.
EXTRACT(NULL) is zero; INIT of typed NULL is NULL; empty/all-NULL MERGE is zero
and MERGE_PARTIAL is NULL. Mixed 10/24 precision sketches merge; mixed input
types and invalid sketches return OutOfRange. Internal merged precision and
cross-dialect NUMERIC sketch interchange were not inspected.

The original GoogleSQL INIT(NULL) case omitted FROM and failed aggregate
placement, while PostgreSQL accepted its original case. A corrected GoogleSQL
case with a one-row FROM input succeeds with NULL. Both results are retained;
the first error is not a missing HLL feature. The extra controls also confirm
approximate distinct count is zero on empty/all-NULL input.

Dynamic JSON graph filtering, grouping, ordering and distinctness execute.
The fixture groups two JSON string values together; a numeric comparison with
JSON numeric strings returns no rows. A bare JSON boolean in WHERE returns
InvalidArgument because the predicate must be BOOL. These results agree with
the inspected guide. The hash result is a control, not evidence of a new hash
signature.

The [queue recreation control](evidence/managed-recheck-20261023/queue-recreate.jsonl)
creates, receives from, drops and recreates the same owned queue name. Both
generations return only their respective new message IDs. No deliberate
reset wait is inserted, but the administrative RPCs themselves take time.
Two successful generations do not prove zero reset delay or universal behavior.

## Documentation history since October 21 retrieval

The pinned interval is `9f3859b9ea131f481bca92225c4cd4c020d5885b` to
`3dcf9df46b1a7bdb261991f0b4197e9b4dda6acf`: commits `13fdf35fcd2e` at
October 21 18:01:49 UTC and `3dcf9df46b1a` at October 22 18:01:41 UTC.
All per-commit file lists were inspected: nineteen net files, sixteen unique
document paths, twenty-nine available pinned before/after bodies and sixteen
full current official documents. All sixteen mirror head bodies match the
current official API after mirror frontmatter removal. Mirror timestamps
remain capture dates, not runtime transition dates.

| Group | Disposition |
| --- | --- |
| Approximate/HLL/release references | Mirror and official API now include the October 20 release and the three formerly failed pages. Live feed also remains latest October 20. The October 21 freshness gap has caught up; function-table padding is incidental. PostgreSQL numeric sketch incompatibility also gains a known-issues section. |
| Transaction statistics | Four distribution fields added to both top/aggregate documentation. These fields were already present in retained runtime observations; this is not a new catalog rollout. |
| Queues | Same-name drop/recreate limitation warning removed; the bounded two-generation control above executes independently of that edit. |
| Geo-partitioning | Documented maximum changes ten to twenty; additional-partition movement wording changes and a partitioned-data Change Streams warning is removed. This regional trial does not verify provisioning or partitioned Change Streams. |
| Omni encrypted deployments | Wrapper keyset generation and distribution to all servers/pods added. Source context only; no Omni rerun. |
| Search functions / Data Boost examples | Mirror and API contain misplaced angle brackets, altered arrows and malformed snippets. Current live HTML has normal named-argument arrows and timestamp predicates. Preserve this source divergence; it does not establish changed SQL grammar. |
| MCP | Client-name prose changes from Gemini CLI to Antigravity; no managed SQL consequence established. |

Full bodies, per-path dispositions, retrieval times and hashes are retained in
the [history audit](evidence/managed-recheck-20261023/documentation-history-audit.json)
and its linked task directory. The new failure log names Omni replicate-data;
a document retrieval failure does not establish runtime support or lifecycle.

## Cleanup, retention and remaining work

All eleven owned temporary databases were dropped. Fixture-specific GetDatabase
checks return NotFound; the final administrative listing contains only the
dedicated verification database. Queue/table fixtures are removed. The free
trial and dedicated base database remain available through the user-level
profile for follow-up. See the [cleanup summary](evidence/managed-recheck-20261023/cleanup-summary.json)
and [final database listing](evidence/managed-recheck-20261023/target2-final-databases.json).

The existing ACTIVE_BATCH_OPERATIONS registry gap and historical nullable
decoder issue remain implementation work. No analyzer selector, dependency or
implementation was changed, and no commit, push, PR, merge, release or external
documentation synchronization was performed. Follow-up claims must retain the
new-trial boundary and distinguish accepted SQL from index filtering, actual
records, internal sketch representation and performance.

Canonical evidence is under
[managed-recheck-20261023](evidence/managed-recheck-20261023/receipt.json).
The durable task area is
`<LOCAL_TASK_ROOT>/apstndb/spanalyzer/managed-recheck-20261023`; keep its full
primary bodies and evidence until explicitly cleaned up. Local validation is
recorded separately in [validation.json](evidence/managed-recheck-20261023/validation.json):
109 offline source/result/cleanup/privacy assertions, all OKF gates, the
Markdown language check and whitespace checks pass. All 716 earlier retained
artifact hashes remain unchanged. These research gates do not turn the known
managed integration drift failure into a passing gate.
