# Optimizer Version Matrix Observations

Observed on 2026-05-06 with Spanner Omni through `spanemuboost`.

Environment:

- `spanner_omni_image`: `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`
- `spanner_omni_image_digest`: `sha256:e98a088fa66d4a87dbb560d729bf21d998bb843f6018bd8dc118fe320e671886`
- `spanemuboost`: `github.com/apstndb/spanemuboost v0.4.0`
- `spannerplan`: `github.com/apstndb/spannerplan v0.1.8`
- `go`: `go1.26.2 darwin/arm64`
- `optimizer_statistics_package`: `not_pinned`

Command:

```sh
go run ./tools/spanner-query-plan-shape \
  --case docs \
  --optimizer-version-matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

The probe injects statement hints `@{OPTIMIZER_VERSION=1}` through
`@{OPTIMIZER_VERSION=8}`. If an input query already has a leading statement
hint, only the `OPTIMIZER_VERSION` assignment is replaced; other statement
hint assignments are preserved.

The default optimizer behavior is checked separately by running the same query
set without `--optimizer-version-matrix`.

Additional `ALLOW_DISTRIBUTED_MERGE` cross-check:

```sh
go run ./tools/spanner-query-plan-shape \
  --case docs \
  --optimizer-version-matrix \
  --allow-distributed-merge-matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

This expands each query across `ALLOW_DISTRIBUTED_MERGE` unspecified, `TRUE`,
and `FALSE`. The unspecified and `TRUE` variants matched in the observed
matrix, consistent with the documented default.

Additional optimizer-version gap probes:

```sh
go run ./tools/spanner-query-plan-shape \
  --case optimizer_gaps \
  --optimizer-version-matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

This dedicated case targets optimizer-version history items that were not
isolated by the broad documentation-derived matrix. It still uses an empty
synthetic database, so cost-based choices that depend on data volume,
statistics, or latency remain only partially covered.

Source page:

- <https://docs.cloud.google.com/spanner/docs/query-optimizer/versions>
- <https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#statement-hints>

The optimizer-version page was checked with `dkcli` on 2026-05-06. The page
reported `updateTime: 2026-05-03T18:17:15Z`, current default version 8, and
version 8 as the latest version.

## Summary

- Checked `51` documentation-derived query cases across optimizer versions 1
  through 8 (`408` matrix entries).
- `17` cases changed compact-tree-metadata operator shape, metadata, or error
  status across versions.
- `2` matrix entries returned an error.
- The `ALLOW_DISTRIBUTED_MERGE` cross-check ran `1224` matrix entries (`51`
  cases x `8` optimizer versions x `3` hint states). `6` cases changed when
  `ALLOW_DISTRIBUTED_MERGE=FALSE`; all were sort or sorted-merge related.
- The optimizer gap probe ran `112` matrix entries (`14` cases x `8`
  optimizer versions) before the v2 prefix extraction probes were added. Every
  entry planned successfully.
- Additional v2 prefix extraction probes for forced `SongsBySongName` scans
  reproduce a clear v1/v2 boundary for both `REGEXP_CONTAINS(SongName,
  "^A.*")` and `SongName LIKE "A%z"`: v1 performs a full index scan with a
  residual predicate, while v2 extracts `STARTS_WITH($SongName, 'A')` as a
  split/seek condition.
- The observations are plan-shape evidence from an empty synthetic database, not a cost or latency benchmark.

## CTE Focused Check

The CTE-focused probe uses a dedicated case rather than the broad documentation
operator matrix:

```sh
go run ./tools/spanner-query-plan-shape \
  --case cte \
  --output compact-tree-metadata \
  --continue-on-error

go run ./tools/spanner-query-plan-shape \
  --case cte \
  --optimizer-version-matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

The default optimizer setting and explicit optimizer versions 1 through 8
produced the same shape for every successful CTE case in Spanner Omni
2026.r1-beta.

The official optimizer-version history says version 8 considers `WITH` clauses
when making cost-based plan choices. The synthetic CTE probes below did not
expose a v8-only boundary: every successful case matched from v1 through v8 and
also matched the default optimizer setting. Treat this as a narrow plan-shape
observation for simple CTEs, not as evidence that v8's `WITH`-clause
optimization is irrelevant in cost-sensitive or statistics-sensitive workloads.

| Case | Default and v1-v8 operator-family summary |
| --- | --- |
| `cte/constant-single-reference` | `Serialize Result > Unit Relation` |
| `cte/constant-repeated-reference` | `Serialize Result > Cross Apply(SpoolBuild > Compute > Unit Relation, Map: Cross Apply(SpoolScan, Map: Filter > SpoolScan))` |
| `cte/deterministic-function-single-reference` | `Serialize Result > Unit Relation` |
| `cte/deterministic-function-repeated-reference` | `Serialize Result > Cross Apply(SpoolBuild > Compute > Unit Relation, Map: Cross Apply(SpoolScan, Map: Filter > SpoolScan))` |
| `cte/current-timestamp-single-reference` | `Serialize Result > Unit Relation` |
| `cte/current-timestamp-repeated-reference` | `Serialize Result > Cross Apply(SpoolBuild > Compute > Unit Relation, Map: Cross Apply(SpoolScan, Map: Filter > SpoolScan))` |
| `cte/table-single-reference` | `Distributed Union > Distributed Union > Serialize Result > Filter Scan > Scan` |
| `cte/table-repeated-reference` | `Serialize Result > Cross Apply(SpoolBuild > Distributed Union > Distributed Union > Scan, Map: Cross Apply(SpoolScan, Map: Filter > SpoolScan))` |

This suggests the observed CTE spool decision is dominated by repeated
reference count in these probes, not by constant vs deterministic function vs
statement-stable `CURRENT_TIMESTAMP()` vs table-reference content. The
repeated-reference cases materialize the CTE once with `SpoolBuild` and read it
twice with `SpoolScan`.

Additional non-deterministic probes:

- `RAND()` is not usable in this Omni build: `Unsupported built-in function:
  rand.`
- `GENERATE_UUID()` in a single-reference CTE planned as
  `Serialize Result > Compute > Unit Relation` for versions 1 through 8 and
  for the default optimizer setting.
- `GENERATE_UUID()` in a repeated-reference CTE returned an internal error for
  versions 1 through 8 and for the default optimizer setting, so it is not
  included in the stable built-in `cte` case.

## Optimizer Gap Probes

The dedicated `optimizer_gaps` case adds probes for optimizer-version history
items that are not covered well by the general operator examples. These probes
are useful acceptance and shape checks, but they should not be read as complete
performance validation for cost-based features.

| Official area | Probe | Observed result |
| --- | --- | --- |
| Version 8 `WITH`, large `IN`, join, and `LIMIT` interaction | `optimizer-gaps/v8/with-large-in-join-order-limit` | Planned in every version. Shapes split into v1-v2, v3-v4, v5, and v6-v8 groups. The visible boundaries line up more closely with version 3 distributed merge / sorted `LIMIT`, version 5 cost model, and version 6 cardinality or limit-related changes than with a v8-only feature. No v8-only boundary was observed in the empty synthetic schema. |
| Version 8 large `IN (...)` | `optimizer-gaps/v8/large-in-list` | Planned in every version with the same compact-tree-metadata shape. This verifies acceptance only; it does not verify the documented performance improvement. |
| Version 8 foreign-key handling / `USE_UNENFORCED_FOREIGN_KEY` | `optimizer-gaps/v8/use-unenforced-foreign-key-true` and `...-false` | With `TRUE`, v1-v7 kept the join to `FKCustomers`, while v8 scanned only `FKOrders`. With `FALSE`, all versions kept the join. The most relevant official item is version 8's "Other improvements" entry that explicitly mentions foreign keys. |
| Version 7 cost-based index union | `optimizer-gaps/v7/unhinted-index-union-candidate` | v1-v6 used a single distributed scan/filter shape; v7-v8 used `Union All` over two index-scan branches plus an aggregate. This directly exercises the unhinted index-union optimizer behavior in the synthetic schema. |
| Version 7 smart seek versus scan | `optimizer-gaps/v7/partial-key-seek-candidate` | Planned in every version with the same shape and `seekable_key_size=1`. This confirms a stable accepted shape, not a statistics-sensitive seek/scan decision. |
| Version 6 cost-based DML optimization | `optimizer-gaps/v6/dml-insert-select-filter` | Planned in a read-write transaction for every version with the same `Apply Mutations > ... > Filter Scan` shape. No v6 boundary was isolated. |
| Version 6 full outer join pushdown | `optimizer-gaps/v6/full-outer-join-predicate-limit` | v1-v4 used `Hash Join`, v5 used `Outer Apply`, and v6-v8 used `Distributed Outer Apply`. The most relevant official item is version 6's limit and predicate pushing through full outer joins. |
| Version 4 secondary index usage under interleaved-table join | `optimizer-gaps/v4/interleaved-secondary-index-join` | Planned in every version with the same index-scan shape. The empty schema did not isolate a v4 boundary. |
| Version 4 leading-index predicate selection | DBaaS-observed `Singers` / `SingersByFirstLastName` example | v3 used a full base-table scan with residual predicates. v4 used the secondary index for the leading `FirstName` predicate and then a distributed apply/back join to the base table for the `BirthDate` residual. This is the strongest current DBaaS v4 before/after example and lines up with the documented secondary-index selection improvement. A minimal Spanner Omni 2026.r1-beta recheck with the same schema did not reproduce the boundary: both literal-value and parameterized forms used the v3-style base-table full scan in v3 and v4. |
| Version 3 `GROUP BY` scan improvement | `optimizer-gaps/v3/group-by-prefix-no-extra-column` | Planned in every version with the same stream aggregate over `SongsBySingerAlbumSongNameDesc`. This verifies the shape is legal but does not isolate a v3 boundary. |
| Version 3 sorted `LIMIT` across cross apply | `optimizer-gaps/v3/sorted-limit-cross-apply` | v1-v2 used a top-level `Sort Limit`; v3-v4 and v6-v8 used the distributed/local limit placement expected from the v3 history; v5 had a distinct distributed-cross-apply shape. The most relevant official items are version 3's sorted `LIMIT` across cross apply and distributed merge union, with a later version 5 cost-model influence. |
| Version 3 computation pushdown through join | `optimizer-gaps/v3/push-computation-through-join` | Shapes split into v1-v2, v3-v4, v5, and v6-v8 groups, showing visible optimizer-version boundaries for this official example pattern. The most relevant official item is version 3's computation pushdown through `JOIN`; later v5/v6 shape changes likely reflect cost-model and cardinality updates. |
| Version 2 `REGEXP_CONTAINS` prefix predicate | `optimizer-gaps/v2/regexp-contains-prefix-forced-index` | v1 performed a full `SongsBySongName` index scan with residual `REGEXP_CONTAINS`; v2 extracted `STARTS_WITH($SongName, 'A')` as split and seek conditions. |
| Version 2 `LIKE` prefix predicate | `optimizer-gaps/v2/like-prefix-forced-index` | v1 performed a full `SongsBySongName` index scan with residual `LIKE`; v2 extracted `STARTS_WITH($SongName, 'A')` as split and seek conditions while keeping the full `LIKE` residual filter. |
| Version 2 scan under `GROUP BY` | `optimizer-gaps/v2/group-by-scan-prefix` | Planned in every version with the same stream aggregate over an index scan. No v2 boundary was isolated. |

The cost-based limitations here are expected: the probe database is empty and
does not have representative optimizer statistics. For those features, the
useful signal is whether the query and hint combination is accepted and whether
any coarse operator-family boundary is visible.

## Documentation Comparison

- The official page says version 8 is the current default and latest optimizer version.
- Version 3 introduced `MERGE_JOIN`, `PUSH_BROADCAST_HASH_JOIN`, and the
  documented Distributed Merge Union behavior. In raw QueryPlan, that behavior
  is observed as `Distributed Union` with `preserve_subquery_order: true`. The
  matrix matches the join-hint history: `JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN`
  is rejected in v1 and v2, then accepted from v3 onward.
- The initial optimizer-version-only matrix was not sufficient to explain the
  sort placement changes. The `ALLOW_DISTRIBUTED_MERGE` cross-check shows that
  the v3+ sort-placement changes are primarily tied to the distributed merge
  sort algorithm: default and `TRUE` match, while `FALSE` reverts to the v1/v2
  top-level sort shape for the affected cases.
- Version 3 also documents better placement for sorted `LIMIT` across cross apply. The cross-check shows that this placement depends on distributed merge being allowed for the observed sorted `LIMIT` back-join probe.
- Version 5 documents cost-model changes for distribution management, sort placement, `GROUP BY`, join algorithm selection, and join commutativity. The observed matrix has several v5 boundary changes, including `distributed/distributed-apply` changing from an apply-style distributed plan to a hash join with `BloomFilterBuild`.
- Later versions document additional cost-model and index-selection improvements, but most synthetic cases here were stable from v5 through v8. This likely reflects the empty schema/statistics setup more than an absence of optimizer changes.

## `ALLOW_DISTRIBUTED_MERGE` Cross-Check

The statement hint documentation says `ALLOW_DISTRIBUTED_MERGE=TRUE` is the
default and favors a distributed merge sort algorithm for certain `ORDER BY`
queries. When applicable, global sorts are changed to local sorts and locally
sorted data is merged into global order.

The observed matrix supports that interpretation:

- `default` and `TRUE` were identical for every checked case.
- `FALSE` changed only the sort-related cases below.
- In the representative v3 reference plan, `ALLOW_DISTRIBUTED_MERGE=FALSE`
  moved `Sort` / `Sort Limit` above `Distributed Union`; the default plan kept
  `Distributed Union` above local `Sort` / `Local Sort Limit` and included
  `preserve_subquery_order: true` metadata.

| Case | Versions where `FALSE` differs | Default / `TRUE` shape | `FALSE` shape |
| --- | --- | --- | --- |
| `unary/tablesample-reservoir` | `v3-v8` | `Limit > Distributed Union > Serialize Result > Sort Limit > Random Id Assign > Scan` | `Serialize Result > Sort Limit > Distributed Union > Random Id Assign > Scan` |
| `unary/sort` | `v3-v8` | `Distributed Union > Serialize Result > Sort > Scan` | `Serialize Result > Sort > Distributed Union > Scan` |
| `unary/sort-limit` | `v3-v8` | `Limit > Distributed Union > Serialize Result > Sort Limit > Scan` | `Serialize Result > Sort Limit > Distributed Union > Scan` |
| `unary/minor-sort-order-by-partial-key` | `v3-v4` | `Distributed Union > Serialize Result > Sort > Scan` | `Distributed Union > Serialize Result > Minor Sort > Scan` |
| `binary/merge-join-with-sort` | `v3-v8` | `Serialize Result > Merge Join > Distributed Union > Sort > Scan` | `Serialize Result > Merge Join > Sort > Distributed Union > Scan` |
| `best-practices/order-by-desc-limit-back-join-optimizer-version-5` | `v3-v8` | `Limit > Distributed Union > Serialize Result > Distributed Cross Apply > Create Batch > RowToDataBlock > Sort Limit > Scan > Cross Apply > KeyRangeAccumulator > DataBlockToRow > Filter Scan` | `Serialize Result > Sort Limit > Distributed Union > Distributed Cross Apply > Create Batch > RowToDataBlock > Scan > Cross Apply > KeyRangeAccumulator > DataBlockToRow > Filter Scan` |

Representative reference output:

```text
@{OPTIMIZER_VERSION=3} SELECT s.SongGenre FROM Songs AS s ORDER BY SongGenre
+----+---------------------------------------------------------------------------+
| ID | Operator                                                                  |
+----+---------------------------------------------------------------------------+
|  0 | Distributed Union on Songs <Row> (preserve_subquery_order: true)          |
|  1 | +- Serialize Result <Row>                                                 |
|  2 |    +- Sort <Row>                                                          |
|  3 |       +- Local Distributed Union <Row>                                    |
|  4 |          +- Table Scan on Songs <Row> (Full scan, scan_method: Automatic) |
+----+---------------------------------------------------------------------------+

@{OPTIMIZER_VERSION=3, ALLOW_DISTRIBUTED_MERGE=FALSE} SELECT s.SongGenre FROM Songs AS s ORDER BY SongGenre
+----+---------------------------------------------------------------------------+
| ID | Operator                                                                  |
+----+---------------------------------------------------------------------------+
|  0 | Serialize Result <Row>                                                    |
|  1 | +- Sort <Row>                                                             |
|  2 |    +- Distributed Union on Songs <Row>                                    |
|  3 |       +- Local Distributed Union <Row>                                    |
|  4 |          +- Table Scan on Songs <Row> (Full scan, scan_method: Automatic) |
+----+---------------------------------------------------------------------------+
```

## Control And Observability

Optimizer-version differences are not all equal. Some are plan-time decisions
that can be reproduced with hints or SQL rewrites; some are visible in
`QueryPlan` but have no complete hint surface; others affect runtime behavior
without a stable structural PLAN signal.

The maintained interpretation layer is now:

- [Optimizer Decision Control And Plan Observability](OPTIMIZER_DECISION_CONTROL_AND_OBSERVABILITY.md)

That note asks two additional questions for every version boundary listed here:

- Can the earlier or later shape be forced while using the opposite optimizer
  version?
- Is the optimizer-version or hint-induced decision visible in `QueryPlan`
  strongly enough to support a PLAN contract?

## Changed Cases

The table below groups compact-tree-metadata changes and notes the most likely
related optimizer-version history item. Some links are necessarily inferential:
the official history is feature-level, while the observed plan is a specific
empty-schema shape.

| Case | Boundary | Most related optimizer-version history item | Observed plan-shape signal |
| --- | --- | --- | --- |
| `execution-plans/join` | v1-v4 / v5-v7 / v8 | v5 cost-based join selection and commutativity; v8 improved `JOIN` reordering | Join input order and `scan_target` switch in v5, then v8 adds a more distributed local index-scan branch. |
| `unary/compute-struct` | v1-v6 / v7-v8 | v7 smart seek versus scan when not all key parts are seekable | Nested array subquery `Filter Scan.seekable_key_size` changes from `2` to `0`. |
| `unary/tablesample-reservoir` | v1-v2 / v3-v8 | v3 distributed merge union | v3+ uses `Limit > Distributed Union(preserve_subquery_order=true) > Sort Limit`; v1-v2 keeps a global `Sort Limit` above `Distributed Union`. |
| `unary/serialize-result-array` | v1-v6 / v7-v8 | v7 smart seek versus scan when not all key parts are seekable | Nested array subquery `Filter Scan.seekable_key_size` changes from `2` to `0`. |
| `unary/sort` | v1-v2 / v3-v8 | v3 distributed merge union | v3+ moves `Distributed Union(preserve_subquery_order=true)` above local `Sort`. |
| `unary/sort-limit` | v1-v2 / v3-v8 | v3 distributed merge union and sorted `LIMIT` placement | v3+ uses `Limit > Distributed Union(preserve_subquery_order=true) > Sort Limit`; v1-v2 keeps global `Sort Limit` above `Distributed Union`. |
| `unary/minor-sort-order-by-partial-key` | v1-v2 / v3-v4 / v5-v8 | v3 distributed merge union; v5 sort-placement cost model | v3-v4 choose full `Sort` on `AlbumsByAlbumTitle`; v5-v8 return to `Minor Sort` on the base table. |
| `binary/cross-apply` | v1-v6 / v7-v8 | v7 smart seek versus scan when not all key parts are seekable | Map-side `Filter Scan.seekable_key_size` changes from `2` to `0`. |
| `binary/in-subquery` | v1-v4 / v5 / v6-v8 | v5 cost-based join selection; v6 cardinality and cost model | v1-v4 use `Hash Join(BUILD_SEMI)`, v5 uses `Semi Apply`, and v6-v8 use `Cross Apply` with a pre-aggregate branch. |
| `binary/not-in-subquery` | v1-v4 / v5-v8 | v5 cost-based join selection | v1-v4 use `Hash Join(BUILD_ANTI_SEMI)`, v5-v8 use `Anti-Semi Apply`. |
| `binary/merge-join-with-sort` | v1-v2 / v3-v8 | v3 distributed merge union with merge join | v3+ moves `Distributed Union(preserve_subquery_order=true)` below each merge-join side's local `Sort`. |
| `binary/recursive-union-graph` | v1-v4 / v5 / v6-v8 | v5 distribution-management cost model; v6 cardinality and cost model | Recursive graph traversal moves distributed boundaries and batch links across versions. |
| `distributed/distributed-union` | v1-v6 / v7-v8 | v7 smart seek versus scan when not all key parts are seekable | `Filter Scan.seekable_key_size` changes from `2` to `0`, and scan method metadata changes. |
| `distributed/distributed-apply` | v1-v4 / v5-v8 | v5 cost-based join algorithm selection and commutativity | v1-v4 use `Distributed Cross Apply`; v5-v8 use `Hash Join` with `BloomFilterBuild`. |
| `distributed/distributed-merge-union` | v1-v2 / v3-v8 | v3 distributed merge union | v3+ uses `Distributed Union(preserve_subquery_order=true)` above local `Sort`; v1-v2 keep global `Sort` above `Distributed Union`. |
| `distributed/push-broadcast-hash-join` | v1-v2 / v3-v8 | v3 push broadcast hash join introduction | v1-v2 reject `PUSH_BROADCAST_HASH_JOIN`; v3-v8 plan `Push Broadcast Hash Join` with a local `Hash Join` map subtree. |
| `best-practices/order-by-desc-limit-back-join-optimizer-version-5` | v1-v2 / v3-v8 | v3 distributed merge union and sorted `LIMIT` across cross apply | v3+ adds `Distributed Union(preserve_subquery_order=true)` and `Distributed Cross Apply(order_preserving=true)` around local `Sort Limit`. |

## Errors

| Case | Version | Error |
| --- | --- | --- |
| `distributed/push-broadcast-hash-join` | `v1` | `ERROR: spanner: code = "InvalidArgument", desc = "Invalid value for join_type/join_method hint: push_broadcast_hash_join."` |
| `distributed/push-broadcast-hash-join` | `v2` | `ERROR: spanner: code = "InvalidArgument", desc = "Invalid value for join_type/join_method hint: push_broadcast_hash_join."` |

## Compact-Tree-Metadata Results

The canonical full result set is intentionally generated from the tool instead
of hand-maintained here:

```sh
go run ./tools/spanner-query-plan-shape \
  --case docs \
  --optimizer-version-matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

The changed-case table above is the maintained summary. It keeps the
operator-tree topology, metadata keys, and child-link annotations that matter
for interpreting optimizer-version boundaries.
