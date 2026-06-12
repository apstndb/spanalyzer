# Spanner Unofficial Hacks operators.md consistency review

Target: <https://github.com/apstndb/spanner-hacks/blob/master/content/ja/operators.md>

Reviewed on 2026-05-07 against the upstream `master` version fetched from:
<https://raw.githubusercontent.com/apstndb/spanner-hacks/master/content/ja/operators.md>

This review compares upstream `operators.md` with the local plan-shape research
under `research/spanner-query-plan-shape/`.

## Summary

One concrete mismatch was found between the current upstream document and the
local raw QueryPlan observations: normal repeated-CTE `SpoolScan` appeared as
raw `PlanNode.display_name: SpoolScan`, not as `Scan` with
`metadata.scan_type: SpoolScan`.

Most earlier feedback has already been reflected upstream. In particular,
upstream now covers or explicitly qualifies the following points:

- Scalar `PlanNode`s are raw QueryPlan vocabulary and are not normally rendered
  as rows in the relational operator tree.
- `subquery_cluster_node` is useful for distributed operators, but `call_type:
  Local` Distributed Union can also carry it, so root-partitionability checks
  must inspect more than the existence of that metadata.
- `Distributed Merge Union` is a documented behavior, not a raw
  `PlanNode.displayName`; it is represented as `Distributed Union` with
  ordering metadata such as `preserve_subquery_order: true`.
- `Empty Relation`, `Minor Sort`, `Minor Sort Limit`, `SpoolBuild`,
  `SpoolScan`, `MiniBatchAssign`, `MiniBatchKeyOrder`, `RowCount`, DML
  `Apply Mutations`, `ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN`, function hint
  `DISABLE_INLINE`, `DataBlockToRow`, and `RowToDataBlock` are now represented
  with either examples or cautionary notes.
- Full Text Search basics and the previously missing details are covered:
  `SearchIndexScan` appears as `Scan.scan_type`, `Search Predicate` is a scalar
  child link, `SEARCH(...)` may involve `Search Query Conversion` and
  `VerifyDeterminism`, `SEARCH_SUBSTRING(...)` can avoid that TVF, composite
  search predicates can use a scalar `Function`, numeric array predicates can
  use search indexes, and mixed text/non-text predicates can leave residual
  `Filter Scan` conditions.

Apart from the `SpoolScan` raw-representation mismatch, the remaining
differences are small evidence-strength notes and upstream statements that
should remain tentative because local reproduction is still missing or limited.

## Latest Search Predicate PLAN Evidence

`github.com/apstndb/spannerplan v0.1.9` changed the useful human-readable
surface for Full Text Search plans: `spannerplan/plantree/reference` now marks
the relevant `SearchIndex Scan` row and prints the hidden `Search Predicate`
scalar expression in the `Predicates(identified by ID)` section.

The latest generated evidence is:

- `research/spanner-query-plan-shape/spanner-hacks-operators-feedback/full-text-search-ja.md`

It was regenerated on 2026-05-07 with Spanner Omni and
`github.com/apstndb/spannerplan v0.1.9`.

Important examples from that file:

- `full-text-search/search`: simple `SEARCH(...)` now renders
  `Search Predicate: SQUERY(index_name:AlbumTitle_Tokens in SearchAlbumsTitleIndex
  predicate:$oo_tvf_0)`.
- `full-text-search/substring`: `SEARCH_SUBSTRING(...)` renders
  `Search Predicate: SQUERY(... predicate:SEARCH_SUBSTRING_QUERY(...))`.
- `full-text-search/multi-column-conjunction` and
  `full-text-search/multi-column-disjunction`: composite search predicates are
  visible as `AND` / `OR` combinations of multiple `SQUERY(...)` terms.
- `full-text-search/numeric-array-any` and
  `full-text-search/numeric-array-all`: numeric array predicates are visible as
  `SPAN_NUMBER_QUERY_TO_SQUERY(...)`.
- `full-text-search/mixed-accelerated`,
  `full-text-search/mixed-stored-filter`, and
  `full-text-search/mixed-back-join`: reference output now shows both the
  residual `Filter Scan` condition and the `Search Predicate` attached to
  `SearchIndex Scan`.

This supersedes the previous note that `plantree/reference` could not show the
hidden scalar-child shape for Search Predicate. It still does not render
non-subquery scalar nodes as tree rows, but it now exposes these search
predicates as predicate annotations on the relevant plan node.

## Local Observations Worth Adding Upstream

### `SpoolScan` raw representation

Upstream says normal `SpoolScan` is represented as `Scan.scan_type=SpoolScan`,
while recursive plans use the display name `Recursive Spool Scan`.

The repeated-CTE raw QueryPlan captured locally does not match that statement.
It has raw `display_name: SpoolScan` and `metadata.spool_name: CTE` directly:

```yaml
index: 11
kind: RELATIONAL
display_name: SpoolScan
metadata:
  execution_method: Row
  spool_name: CTE
```

The second scan in the same plan has the same representation:
`index=15 display=SpoolScan scan_type= spool_name=CTE`.

This is a concrete correction candidate for upstream. A safe wording would be:
normal repeated-CTE spool scans can appear as raw `PlanNode.display_name:
SpoolScan` with `metadata.spool_name`, while graph recursive plans can appear
as `Recursive Spool Scan`.

## Upstream Statements Still Tentative Locally

### `Generate Relation`

Upstream explicitly says a stable positive reproduction has not been confirmed
on the local Spanner Omni environment, and that `SELECT 1 + 2` currently renders
as `Unit Relation`.

This matches local observations. No correction is needed, but the section
should remain tentative until a positive case is found.

### `Local Split Union`

Upstream explicitly says the sample schema has not produced a stable
reproduction and that placement or locality configuration may be required.

This matches local observations. No correction is needed, but the section
should remain tentative.

### `MiniBatchAssign`, `MiniBatchKeyOrder`, and `RowCount`

Upstream cross-references the shared `RowCount` example and only states the
relative position of these operators.

This still matches local evidence, but the reproduction should stay explicitly
environment-sensitive. The documented SQL:

```sql
@{OPTIMIZER_VERSION=5}
SELECT *
FROM Songs@{FORCE_INDEX=SongsBySongName}
ORDER BY SongName DESC
LIMIT 1;
```

reproduced `MiniBatchAssign`, `MiniBatchKeyOrder`, and `RowCount` in Cloud
Spanner DBaaS evidence, but the same query in the empty synthetic Spanner Omni
2026.r1-beta database produced a simpler back-join plan across optimizer
versions 1 through 8.

Local work still has not established deeper semantics such as the assignment
and key-ordering contract.

### `Create Batch` scalar children

`Create Batch` itself is well observed in distributed apply and push broadcast
plans. Raw child-link inspection shows one relational input followed by scalar
child links whose variables define fields in the generated batch relation, for
example `v2.Batch.SingerId`, `v2.Batch.TrackId`, or `v2.Batch.__row_id`.

The field set changes by context: broadcast keys in Push Broadcast Hash Join,
back-join keys and sort values in non-covering index plans, and traversal keys
in graph recursion. The exact engine-level meaning of each generated field is
still an interpretation, so detailed wording should stay conservative unless
raw plans are included as evidence.

### Distributed execution location

Upstream describes distributed operators as executing the subtree pointed to by
`subquery_cluster_node` on remote servers satisfying the `Split Range`.

This is consistent with the documented operator family and observed plan shape,
but local research did not directly observe physical execution location beyond
the QueryPlan metadata. This is acceptable as explanatory prose, but it should
be understood as documentation-backed interpretation, not something proven by
the local matrix alone.

## Things That Are No Longer Gaps

The following were previous feedback items, but the current upstream document
already reflects them well enough:

- `Distributed Merge Union` raw representation.
- `Empty Relation` reproduction using the official documented pattern.
- Scalar operator visibility and the distinction between raw PlanNode vocabulary
  and rendered relational operator trees.
- `Function` hint `DISABLE_INLINE`.
- `ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN` and `Timestamp Condition`.
- `subquery_cluster_node` caveat for `call_type: Local`.
- DML variants around `Apply Mutations`, including the observed
  `ASSERT_ROWS_MODIFIED` limitation in `PLAN`.
- `DataBlockToRow` / `RowToDataBlock` and the fact that `SCAN_METHOD=ROW` alone
  is not a complete control knob; `EXECUTION_METHOD=ROW` can also matter.
- Push Broadcast Hash Join wrapper variants and the warning not to classify a
  descendant `Hash Join` naively as a standalone regular Hash Join.
- Full Text Search numeric array predicates, mixed text/non-text predicates,
  and composite `Search Predicate` child-link behavior.

## Suggested Upstream Feedback

The remaining actionable feedback is small:

1. Correct the normal `SpoolScan` raw representation statement, or add a
   counterexample raw PlanNode if `Scan.scan_type=SpoolScan` exists in another
   plan family.
2. Keep `Generate Relation`, `Local Split Union`, and MiniBatch semantics
   explicitly tentative until positive reproductions or stronger raw evidence
   are available.
3. Consider adding a raw `Create Batch` child-link excerpt if describing the
   generated batch fields in more detail.

## Local Maintenance Note

Older local feedback in this file claimed several items were not reflected
upstream. That is now stale. This version replaces the old one with the current
bidirectional review:

- upstream content contradicted by local raw observations;
- local observations that are not yet fully represented upstream;
- upstream statements that remain intentionally tentative from the local
  evidence point of view.
