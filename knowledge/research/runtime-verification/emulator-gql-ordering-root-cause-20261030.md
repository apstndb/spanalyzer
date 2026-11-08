---
type: Research Note
title: "Emulator GQL ordering: verified loss of order metadata"
description: "Pinned source, disassembly and five controlled runtime conditions identify why Emulator 1.5.59 scrambles an explicitly sorted GQL result."
tags: [spanner, emulator, googlesql, gql, ordering, root-cause, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: results
    resource: evidence/gql-order-root-cause-20261030/summary.json
    title: "Five controlled runtime conditions and their scope"
  - id: interventions
    resource: evidence/gql-order-root-cause-20261030/captures/binary-interventions.json
    title: "Original binary hashes and exact diagnostic instruction changes"
  - id: source
    resource: evidence/gql-order-root-cause-20261030/sources/googlesql-googlesql__reference_impl__algebrizer.cc
    title: "Pinned GoogleSQL projection and ordering implementation"
  - id: permutation
    resource: evidence/gql-order-root-cause-20261030/sources/googlesql-tuple.cc
    title: "Deterministic undefined-order scrambling implementation"
  - id: assertions
    resource: evidence/gql-order-root-cause-20261030/assertions.json
    title: "Offline checks of actual values, controls, identity and cleanup"
  - id: validation
    resource: evidence/gql-order-root-cause-20261030/validation.json
    title: "Local module and documentation check results"
  - id: query-shapes
    resource: evidence/gql-order-root-cause-20261030/query-shapes/summary.json
    title: "FinGraph scalar, aggregation, pagination and cardinality controls"
  - id: membership
    resource: evidence/gql-order-root-cause-20261030/limit-membership/summary.json
    title: "Complete-key top-N, tied cutoff identities and downstream LIMIT"
---

# Emulator GQL ordering root cause, October 30, 2026

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The reproduced discrepancy is caused by **lost order metadata at the graph-table
boundary**, followed by intentional undefined-order scrambling. A GQL sort is
constructed correctly, but the enclosing `GraphTableScan` supplies
`is_ordered=false`. Algebrization writes that value onto the returned sort
operator. With Emulator's scrambling option enabled, the evaluator then
permutes the already sorted rows.

This conclusion is source-backed and reproduced by two independent binary
interventions against the official linux/arm64 image. The interventions are
diagnostic experiments; neither is a reviewed general source fix. This follows
the [original ordering discrepancy](cross-engine-doc-delta-20261030.md#reproduced-gql-ordering-discrepancy)
without changing that earlier observation or inferring the first affected version.

## Identity and evidence boundary

Runtime execution was at **2026-10-30 09:37:53–09:38:15 UTC / 18:37:53–18:38:15 JST**.

| Object | Pinned identity |
| --- | --- |
| Official runtime | `gcr.io/cloud-spanner-emulator/emulator:1.5.59` linux/arm64, manifest `sha256:969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8` |
| Extracted `/emulator_main` | 119,961,736 bytes; SHA-256 `de255862e81c429b606bf7eb97c542efdc4ed4cd32ff5eddd0f18e64d226a12b` |
| Emulator source tag | `v1.5.59` → `d357ec1d2b08ae92ce1da4c5078de82fda0c07ab` |
| Declared GoogleSQL dependency | `2026.7.2` → `0e7d7073ed0360be587a5efa0fa78abeee00f17b` |
| Client | Same retained Go probe, SDK v1.91.0 and spanemuboost v0.4.10-alpha.1 for all five conditions |

The [Emulator module declaration](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/MODULE.bazel#L234)
pins that GoogleSQL dependency. Original sources and the Emulator's dependency
patches are retained under [sources](evidence/gql-order-root-cause-20261030/sources/).
The source tag and image are independently pinned: matching disassembly and
causal interventions establish the inspected boundary, but no reproducible-build
attestation or full source-rebuilt fix is claimed.

## Verified execution path

1. Emulator's
   [CommonEvaluatorOptions](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/backend/query/query_engine.cc#L159)
   sets `scramble_undefined_orderings=true` for ordinary query evaluation.
2. [ResolveGqlQuery](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/analyzer/resolver_query.cc#L19303)
   routes standalone GQL through `ResolveGraphTableQuery`. The resolved GQL
   tree has an ordered `OrderByScan` inside graph-linear and
   graph-table wrappers whose order flag remains false. The graph-table
   [builder](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/analyzer/graph_query_resolver.cc#L765)
   does not set that flag; the graph scan
   [definitions](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/resolved_ast/gen_resolved_ast.py#L11803)
   do not propagate it from the terminal child. Retained local frontend AST
   dumps corroborate this shape, with `OrderByScan.is_ordered=TRUE` and no
   graph-wrapper order flag. These are **local analyzer dumps**, not AST output
   collected from Emulator's service.
3. [AlgebrizeGraphLinearScan](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/reference_impl/algebrizer_graph.cc#L509)
   returns the final child operator. Then
   [AlgebrizeGraphTableScan](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/reference_impl/algebrizer_graph.cc#L389)
   passes the graph-table flag to `AlgebrizeProjectScanInternal`. In this
   reproducer there are no graph-table shape expressions. The
   [empty-projection branch](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/reference_impl/algebrizer.cc#L6652)
   drops the projection but calls `input->set_is_order_preserving(is_ordered)`.
   Thus the existing sort's order guarantee is overwritten with false.
4. [SortOp::CreateIterator](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/reference_impl/relational_op.cc#L1605)
   wraps a non-order-preserving sort in `ReorderingTupleIterator` when
   scrambling is enabled. Its
   [permutation](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/reference_impl/tuple.cc#L253)
   emits odd indexes followed by even indexes within batches of up to 100 rows.

For five ascending values, sorted `[10,20,30,40,50]` becomes
`[20,40,10,30,50]`, exactly as observed. For two rows, the same permutation
looks like reversal. A three-row LIMIT result becomes `[20,10,30]`; the tested
top-three membership remains correct while output order is broken. This
explains why changing JSON values to INT64 or STRING did not repair the earlier
cases.

For the tested LIMIT shape, the reference evaluator
[enables ORDER BY/LIMIT consolidation](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/public/evaluator_base.cc#L638),
and [algebrization](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/reference_impl/algebrizer.cc#L4335)
folds the immediate ordered input and limit into the sort operator. The same
graph-table boundary therefore clears that operator's order guarantee after
the top-three selection.

## Controlled interventions

The [retained matrix](evidence/gql-order-root-cause-20261030/producers/matrix.json)
creates five nodes with `(id,val)` pairs `(1,50),(2,10),(3,40),(4,20),(5,30)`.
Each condition runs three fixture steps and 27 queries: eighteen ordered GQL
cases, four explicitly ordered SQL controls, four unspecified-order controls,
and one scalar control. The value ASC/DESC and LIMIT cases repeat three times;
additional cases sort IDs, arithmetic expressions and strings.

| Condition | Ordered GQL correct | Ordered SQL correct | Other result |
| --- | --- | --- | --- |
| Official image before | 0/18 | 4/4 | All queries execute; GQL outputs match the predicted permutation |
| Same binary copied into the same base | 0/18 | 4/4 | All query values/codes match the official image |
| Disable scrambling only | 18/18 | 4/4 | Unspecified SQL output changes, as expected |
| Force graph-table order flag true only | 18/18 | 4/4 | Unspecified SQL scrambling stays enabled; unordered GQL produces Internal |
| Official image after | 0/18 | 4/4 | Original values/codes reproduce after the interventions |

The same official ELF was copied without modification for the packaging
control. The other two variants use exactly bounded aarch64 instruction edits:

- **Disable scrambling:** at address/file offset `0x682d74`, change
  `strb w1,[x19,#0x20]` to `strb wzr,[x19,#0x20]` in `CommonEvaluatorOptions`.
  One byte differs; the option becomes false.
- **Preserve graph order:** at `0x94dd38`, change the load of
  `GraphTableScan::is_ordered` into argument `w5` to `mov w5,#1` before the
  `AlgebrizeProjectScanInternal` call. Four bytes differ; global scrambling
  remains true.

Exact original/patched hashes, offsets and byte differences are retained in
the [intervention record](evidence/gql-order-root-cause-20261030/captures/binary-interventions.json),
with disassembly and local image identities under
[captures](evidence/gql-order-root-cause-20261030/captures/). The same compiled
client and matrix serve all conditions. No server response sorting is added
to the client.

The graph-local intervention isolates the order-flag boundary while leaving
the general scrambling mechanism active. Its unordered GQL failure is also
material counterevidence: the evaluator rejects an unconditional true flag
when the underlying operator cannot preserve order. The diagnostic patch must
not be presented as a production fix.

## Repair boundary and remaining scope

The appropriate source repair needs to preserve the **actual terminal
ordering** through the graph wrappers into algebrization. It must retain false
for unordered operators and respect later operators that discard ordering.
Simply setting every graph-table flag true breaks the retained unordered
control; globally disabling scrambling masks the problem and changes all
undefined-order queries. A general patch still needs source-build regression
tests for unordered GQL, nested graph queries, projections, NEXT and set
operations. No repository implementation or parser change is made here.

An outer SQL `ORDER BY` over `GRAPH_TABLE` works in both tested directions on
the unmodified image. An inner GQL ORDER BY alone does not guarantee the outer
SQL query's order, so those two controls are checked only for row membership.
This is a bounded workaround observation, not a new ordering contract.

A source-only check of GoogleSQL HEAD
[`a09b0bd95dd55dba9478eb8f20a3971bc9e30b70`](https://github.com/google/googlesql/tree/a09b0bd95dd55dba9478eb8f20a3971bc9e30b70)
finds the same graph-wrapper construction and graph-table order-flag argument.
It does not establish execution behavior of that HEAD. The first affected
release, amd64 execution, other graph shapes and managed/Omni internals remain
unverified. Previous managed/Omni runtime observations are not rerun here.

## FinGraph query-shape follow-up

At **10:41–10:44 UTC / 19:41–19:44 JST**, two additional disposable runtimes
used the same unmodified image and the complete official FinGraph setup/data.
All **73 queries** executed. In **32 cases with a unique final sort order**,
the final `RETURN ... ORDER BY` still returned the wrong sequence. These cover
IDs, names, countries, TIMESTAMPs, arithmetic, STRING casts, DISTINCT,
aggregated transfer totals, NEXT, multiple sort keys, FOR-generated values,
LIMIT and OFFSET. Independent SQL controls returned the expected values/order.
The [matrix and comparisons](evidence/gql-order-root-cause-20261030/query-shapes/summary.json)
retain exact queries, values and their different contracts.

Ordering is nevertheless effective in several distinct senses:

- `RETURN p.id ORDER BY p.id DESC LIMIT 1` selects ID 3; ASC selects ID 1.
  Across the tested top-N controls, selected membership is correct even when
  the final sequence is permuted.
- `RETURN ARRAY_AGG(p.id ORDER BY p.id) AS ids` returns `[1,2,3]`; DESC
  returns `[3,2,1]`. This orders array elements within one result value,
  rather than ordering the outer result rows.
- Ordering by `p.id > 2` returns valid keys `[false,false,true]` even though
  IDs 1 and 2 exchange places. Their equal sort keys make this legal; adding
  ID as a tie-breaker reveals the discrepancy.
- An outer SQL ORDER BY over GRAPH_TABLE returns `[1,2,3]` correctly.

Moving ORDER BY before RETURN is **not a reliable row-order workaround**.
On the five official transfer rows, a primitive ORDER BY followed by LIMIT
looks correct for limits 2 and 3, but fails for 4 and 5. The observed sequence
is exactly the source permutation applied twice. For ascending amounts and a
complete `(amount,source,destination)` sort key, LIMIT 5 returns amounts
`[300,300,200,100,500]` instead of `[100,200,300,300,500]`.
The FOR control similarly turns `[1,2,3,4,5]` into `[4,3,2,1,5]`.

Local ASTs using an explicit reduced `Person(id)` catalog show a standalone
ordered scan, a separate unordered LimitOffsetScan over GraphRefScan, and a
final computed projection. The source
[LimitOp](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/reference_impl/relational_op.cc#L2155)
and [ComputeOp](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/reference_impl/relational_op.cc#L1815)
paths explain the two permutations. This is source-backed corroboration, not
a separate instruction-intervention test of each operator. The initial local
analysis with the full implicit-property graph failed because that local
catalog exposed no Person properties; both that failure and the explicit
catalog ASTs are retained separately from the successful Emulator executions.

The [official syntax](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-query-statements#gql_order_by)
also distinguishes the standalone ORDER BY statement from RETURN's ORDER BY
clause: without an immediately following LIMIT or OFFSET, the standalone
statement is documented as ignored. A passing primitive example must not be
treated as evidence that final RETURN ordering is reliable. The NEXT plus
ARRAY_AGG-without-ORDER control happened to collect `[1,2,3]`; it remains an
encounter-order observation, not an explicit aggregate ordering contract.

Both owned databases end in NotFound and both containers are absent. The
[230 additional evidence checks](evidence/gql-order-root-cause-20261030/query-shapes/assertions.json)
verify codes, exact single/double permutations, selection, SQL controls,
cleanup, preservation of all 1,594 preceding evidence files and the same 123
implementation files. Follow-up documentation checks have their own
[receipt](evidence/gql-order-root-cause-20261030/query-shapes/validation.json).

## LIMIT membership and tied boundaries

At **11:12:27–11:12:32 UTC / 20:12:27–20:12:32 JST**, another unmodified
runtime executed **35 queries / 37 planned steps**. For the tested single
RETURN clauses with complete ordering keys, LIMIT/OFFSET selected the expected
row multisets before the deterministic final permutation. This is bounded
runtime evidence, not a guarantee covering every query shape. Calling the
final permutation random obscures the distinction: with a uniquely sorted
input, its odd-index/even-index mapping is deterministic.

An incomplete sort key can leave **membership** unspecified at the cutoff.
The official FinGraph fixture has transfers of amounts 100, 200, 300, 300 and
500. Four executions of the exact same query with `ORDER BY amount LIMIT 3`
always include 100 and 200, but the selected 300 transfer changes between
source/destination `(16,20)` and `(7,16)`. Both identities were observed in the
same unchanged database. Adding `src,dst` to the ordering makes this fixture's
cutoff unique; all four controls choose the 300 transfer `(7,16)`, while the
final output sequence still has the known permutation. Literal-array controls
also demonstrate both legal choices among equal keys and fixed membership
after adding an ID tie-breaker. These are under-specified tie choices, not
evidence of corrupt membership under a complete sort key.

Do not extend the single-clause result to a later LIMIT without its own
ordering. A five-value FOR query sorts IDs 1–5 in the first RETURN, then uses
`NEXT RETURN id LIMIT 2`; it returns `[4,2]`. Repeating ORDER BY in that final
RETURN restores the selected set `{1,2}`, although output remains `[2,1]`.
The final stage without its own ORDER BY has no safe top-N ordering contract.
The [SQL syntax source](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#limit_and_offset_clause)
documents undefined ordering for LIMIT/OFFSET without ORDER BY; the observed
GQL pipeline result is retained independently from that source statement.

The [membership comparisons](evidence/gql-order-root-cause-20261030/limit-membership/summary.json)
and [91 evidence checks](evidence/gql-order-root-cause-20261030/limit-membership/assertions.json)
retain the four identical-query tie choices, unique-key controls, deterministic
sequences, cleanup and preservation of all 1,632 earlier artifact hashes and
123 measured implementation files. The owned database is NotFound and the
owned container is absent. Follow-up documentation checks have a separate
[receipt](evidence/gql-order-root-cause-20261030/limit-membership/validation.json).

## October 31 historical release follow-up

The [release-level bisection](emulator-gql-ordering-release-bisect-20261031.md)
finds the same discrepancy in 1.5.30, which announced PropertyGraph support,
and six sampled later releases. The predecessor 1.5.29 rejects GRAPH syntax.
This updates the earliest observed affected release; it does not identify a
last-known-good GQL version or a finer introducing source commit. The earlier
observations and intervention evidence above retain their original scope.

## October 31 upstream publication

The user submitted [upstream issue #408](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/408).
The [subsequent source audit](emulator-issue-candidates-20261031.md) observes it
open under `apstndb`, with title and body matching the saved draft. Publication
is now verified; the original investigation's execution and authority records
below remain historical. No additional upstream issue or comment is submitted
by this workflow.

## Retention and validation

The original five runs retain **150 planned steps / 135 queries / 180 root records**.
One Internal result is intentional diagnostic counterevidence, not a passing
query. All five owned database lifecycles end in confirmed NotFound; five
runtime containers and one binary-extraction container are absent. Three owned
diagnostic image tags are removed and the official base image is preserved.

The [offline assertions](evidence/gql-order-root-cause-20261030/assertions.json)
verify 319 evidence conditions, including predicted permutations, packaging
and reversal controls, intervention hashes and cleanup. All 1,491 pre-existing
evidence files and 123 measured implementation/dependency files retain their
original hashes. Commands, probe/build provenance, source snapshots and
reconstruction scripts are retained alongside the captures; large reproducible
ELF copies remain checkout scratch rather than research payloads.

Local module, OKF, language and whitespace check results are retained in the
[validation receipt](evidence/gql-order-root-cause-20261030/validation.json).
No upstream issue, source patch, commit, push or release is submitted.
