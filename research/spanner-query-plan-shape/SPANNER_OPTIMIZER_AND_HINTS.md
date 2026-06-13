# Spanner Optimizer Versions And Hints

This note is a compact index of the official Cloud Spanner optimizer-version
history and Spanner GoogleSQL hints, with links to the local plan-shape evidence
that has been checked with `tools/spanner-query-plan-shape`.

It is non-normative research material. The official documentation and actual
Spanner plans remain the source of truth.

## Sources Checked

- [Spanner query optimizer versions](https://docs.cloud.google.com/spanner/docs/query-optimizer/versions)
  via `dkcli` on 2026-05-06. The retrieved document reported
  `updateTime: 2026-05-03T18:17:15Z`, current default optimizer version 8, and
  version 8 as latest.
- [Query syntax in GoogleSQL: Hints](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#hints)
  via `dkcli` on 2026-05-06. The retrieved document reported
  `updateTime: 2026-05-01T19:00:46Z`.
- [Graph hints](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-query-statements#graph_hints)
  via `dkcli` on 2026-05-06.
- [Function hints](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/functions-reference#function_hints)
  via `dkcli` on 2026-05-06. Function hints are not part of the query-syntax
  Hints section, but are included here because this repository has probe
  coverage for `DISABLE_INLINE`.

Related local evidence:

- [Optimizer decision control and observability](OPTIMIZER_DECISION_CONTROL_AND_OBSERVABILITY.md)
- [Query execution operator observations](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md)
- [Optimizer-version matrix observations](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md)
- [Optimizer-version rendered examples](OPTIMIZER_VERSION_RENDERED_EXAMPLES.md)
- Spanner Unofficial Hacks feedback drafts: delivered upstream and removed
  from this tree on 2026-06-12 (see git history before that date for the
  rendered-plan evidence; the upstream document is
  <https://spanner-hacks.apstn.dev/operators/>).

## Optimizer Version History

The following list mirrors every item from the official optimizer-version
history page as checked on 2026-05-06.

### Version 8: October 28, 2024, Default And Latest

- `WITH` clauses are considered when making cost-based plan choices.
- Improved performance of distributed cross apply and indexed lookup queries.
- Improved `JOIN` reordering.
- Improved performance of queries with large `IN (...)` clauses.
- Improved `GROUP BY` performance in certain cases.
- Other improvements, including more efficient handling of queries with
  `LIMIT`, foreign keys, and index selection.

Local verification summary:

- The broad docs matrix currently checks optimizer versions 1 through 8. The
  checked synthetic query set is not sufficient to prove every v8 improvement,
  but it did exercise CTEs, `IN` / `EXISTS`, joins, groups, indexed lookup, and
  sorted `LIMIT` shapes.
- The dedicated CTE probe found no v8-only boundary for simple single-reference
  and repeated-reference CTEs. Default and v1-v8 shapes matched for all
  successful CTE cases. This is only evidence for the simple synthetic cases,
  not evidence against v8's cost-based `WITH` handling.
- The optimizer gap probe added a combined `WITH` / large `IN` / join /
  `LIMIT` query and a standalone large `IN` query. Both planned successfully
  across v1-v8, but neither exposed a v8-only shape boundary in the empty
  synthetic schema.
- Informational foreign-key optimization did expose a v8 boundary:
  `USE_UNENFORCED_FOREIGN_KEY=TRUE` removed the referenced-table join in v8,
  while v1-v7 retained it. `USE_UNENFORCED_FOREIGN_KEY=FALSE` retained the join
  in every version.
- Details: [CTE Focused Check](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md#cte-focused-check)
  and [Optimizer Gap Probes](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md#optimizer-gap-probes).

### Version 7: May 22, 2024

- Added support for cost-based selection of index union plans.
- Added support for smart selection of seek versus scan plans based on
  statistics for queries that don't have seekable predicates for all key parts.
- Added support for cost-based selection of hash joins.

Local verification summary:

- `INDEX_STRATEGY=FORCE_INDEX_UNION` was checked in table-hint probes and
  produced `Union All` over index-scan branches in the accepted case.
- An unhinted disjunction over two indexed columns was added to the optimizer
  gap probe. v1-v6 used a single distributed scan/filter shape, while v7-v8
  used `Union All` over two index-scan branches plus an aggregate.
- `SEEKABLE_KEY_SIZE=0/1/2` was checked with `FORCE_INDEX=_BASE_TABLE` on a
  two-column primary key. `SEEKABLE_KEY_SIZE=3` was checked separately and
  returned `InvalidArgument`, as expected for a seekable key size larger than
  the key.
- A partial-key seek candidate planned with the same `seekable_key_size=1`
  shape in v1-v8. Cost-based hash join selection was not directly proved as a
  v7 boundary by the synthetic optimizer-version matrix.
- Details: [Other Hint Matrix](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#other-hint-matrix)
  [Broad Statement-Hint Query Matrix](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#broad-statement-hint-query-matrix),
  and [Optimizer Gap Probes](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md#optimizer-gap-probes).

### Version 6: September 11, 2023

- Improved limit pushing and predicate pushing through full outer joins.
- Improved cardinality estimation and cost model.
- Enabled cost-based optimization for DML queries.

Local verification summary:

- DML planning is checked with read-write transactions, including `INSERT`,
  `UPDATE`, `DELETE`, `INSERT IGNORE`, `INSERT OR IGNORE`, `INSERT OR UPDATE`,
  `ON CONFLICT`, and `THEN RETURN` forms.
- The optimizer gap probe adds a DML `INSERT ... SELECT ... WHERE` case to the
  v1-v8 matrix. It planned successfully in every version but did not isolate a
  v6 boundary in the empty synthetic schema.
- A full-outer-join predicate/limit probe did expose a v6 boundary: v1-v4 used
  `Hash Join`, v5 used `Outer Apply`, and v6-v8 used
  `Distributed Outer Apply`.
- Details: [DML Compact-Tree-Metadata Results](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#dml-compact-tree-metadata-results)
  and [Optimizer Gap Probes](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md#optimizer-gap-probes).

### Version 5: July 15, 2022

- Improved cost model for index selection, distribution management, sort
  placement, and `GROUP BY` selection.
- Added cost-based join algorithm selection that chooses between hash and apply
  join. Merge join still requires a query hint.
- Added cost-based join commutativity.

Local verification summary:

- The optimizer-version matrix observed several v5 boundary changes. The most
  visible one in this synthetic schema is `distributed/distributed-apply`,
  which changes from an apply-style distributed plan in v1-v4 to
  `Hash Join` with `BloomFilterBuild` in v5-v8.
- Sort placement also changes across optimizer versions, but the
  `ALLOW_DISTRIBUTED_MERGE` cross-check shows the v3+ distributed-merge sort
  behavior is the primary explanation for the checked sort-placement cases.
- Details: [Documentation Comparison](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md#documentation-comparison).

### Version 4: March 1, 2022

- Improvements to secondary index selection:
  - Improved secondary index usage under a join between interleaved tables.
  - Improved covering secondary index usage.
  - Improved index selection when optimizer statistics are outdated.
  - Prefer secondary indexes with predicates on leading indexed columns even if
    optimizer statistics are unavailable or report the base table is small.
- Introduced single-pass hash join, enabled by
  `HASH_JOIN_EXECUTION=ONE_PASS`.
- Improved selection of how many keys are used for seeking.

Local verification summary:

- `HASH_JOIN_EXECUTION=MULTI_PASS/ONE_PASS` is included in the join-hint matrix
  and subquery join-hint matrix. The rendered plans and a raw YAML recheck show
  identical `Hash Join` nodes for both values; the observed `PlanNode` metadata
  contains `execution_method` and `join_type`, but not the requested
  `HASH_JOIN_EXECUTION` value. The current probe is PLAN-only and therefore can
  verify hash join selection, but not one-pass versus multi-pass intent or
  runtime behavior such as disk spilling or probe-side scan count.
- `SEEKABLE_KEY_SIZE` is included in table-hint probes as noted under version 7.
- An interleaved secondary-index join probe planned successfully across v1-v8
  but did not isolate a v4 boundary in the empty synthetic schema.
- A separate DBaaS-observed example isolates a v3/v4 boundary for secondary
  index selection with a predicate on the leading indexed column:
  `WHERE FirstName = @firstName AND BirthDate BETWEEN @begin AND @end`.
  Version 3 scans the `Singers` base table and applies both predicates as a
  residual filter. Version 4 uses `SingersByFirstLastName` for the `FirstName`
  seek/split range, then performs a distributed apply/back join to the base
  `Singers` table for the `BirthDate` residual filter. This lines up with the
  documented v4 item that prefers secondary indexes with predicates on leading
  indexed columns even when statistics are unavailable or the base table is
  reported small.
- That DBaaS example was explicitly rechecked on Spanner Omni 2026.r1-beta with
  the minimal `Singers` / `SingersByFirstLastName` schema. Both a literal-value
  query and the original parameterized query planned as the same base-table full
  scan in optimizer versions 3 and 4. Treat the before/after as DBaaS evidence
  pending an Omni reproduction, and do not use the empty Omni schema as proof
  that the v4 secondary-index behavior is absent from DBaaS Spanner.
- Details: [Join Hint Matrix](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#join-hint-matrix)
  [Subquery Predicate Statement-Hint Matrix](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#subquery-predicate-statement-hint-matrix),
  and [Optimizer Gap Probes](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md#optimizer-gap-probes).

### Version 3: August 1, 2021

- Added the merge join algorithm, enabled by `JOIN_METHOD=MERGE_JOIN`.
- Added the push broadcast hash join algorithm, enabled by
  `JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN`.
- Introduced the distributed merge union operator, enabled by default when
  applicable.
- Improved scan performance under `GROUP BY` when no `MAX`, `MIN`, `HAVING MAX`,
  or `HAVING MIN` requires extra non-grouped columns.
- Improved performance of sorted `LIMIT` queries when a cross apply operator is
  introduced by joins.
- Improved query performance by pushing more computations through `JOIN`.

Local verification summary:

- `JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN` is rejected in optimizer versions 1
  and 2 and accepted from version 3 onward in the checked matrix. That matches
  the official v3 history.
- `JOIN_METHOD=MERGE_JOIN` is included in the join-hint matrix and direct docs
  examples.
- The documented distributed merge union behavior is observed as raw
  `Distributed Union` with `preserve_subquery_order: true`, not as a separate
  raw `PlanNode.displayName` named `Distributed Merge Union`.
- `ALLOW_DISTRIBUTED_MERGE=FALSE` reverts several v3+ sort-placement shapes to
  a v1/v2-style top-level sort shape in the checked cases.
- Additional optimizer gap probes for the official sorted-`LIMIT` cross-apply
  example and computation-pushdown-through-join example both showed visible
  v3 boundaries. The `GROUP BY` prefix-scan probe planned successfully in
  every version but did not isolate a v3 boundary.
- Details: [ALLOW_DISTRIBUTED_MERGE Cross-Check](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md#allow_distributed_merge-cross-check)
  [Distributed Merge Union](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#compact-tree-metadata-shapes),
  and [Optimizer Gap Probes](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md#optimizer-gap-probes).

### Version 2: March 1, 2020

- Added optimizations in index selection.
- Improved performance of `REGEXP_CONTAINS` and `LIKE` predicates under certain
  circumstances.
- Improved scan performance under `GROUP BY` in certain situations.

Local verification summary:

- Forced `SongsBySongName` probes isolate a visible v1/v2 boundary for
  `REGEXP_CONTAINS(SongName, "^A.*")` and `SongName LIKE "A%z"`. Version 1
  keeps the original predicate as a residual filter over a full index scan;
  version 2 extracts `STARTS_WITH($SongName, 'A')` as split/seek conditions.
- Group-by examples are present in the docs and optimizer gap query sets, but
  the current synthetic matrix does not isolate a version 2 boundary for
  `GROUP BY`.

### Version 1: June 18, 2019

- Includes many rule-based optimizations, such as predicate pushdown, limit
  pushdown, redundant join removal, and redundant expression removal.
- Uses statistics on user data to select which index to use to access each
  table.

Local verification summary:

- Version 1 is included as the baseline for all optimizer-version matrix probes.
  Several later features, such as push broadcast hash join, are absent or
  rejected in v1.

## Optimizer Version Matrix Coverage

Current local matrix summary:

- `51` documentation-derived query cases across optimizer versions 1 through 8:
  `408` matrix entries.
- `17` cases changed compact-tree-metadata operator shape, metadata, or error
  status across versions.
- `2` matrix entries returned errors, both from
  `JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN` on optimizer versions 1 and 2.
- `ALLOW_DISTRIBUTED_MERGE` cross-check: `1224` entries
  (`51` cases x `8` optimizer versions x `3` hint states). `FALSE` changed
  `6` sort or sorted-merge related cases. Default and `TRUE` matched in the
  observed matrix.
- Dedicated CTE check: default optimizer setting plus explicit versions 1
  through 8. Every successful CTE case matched across all settings.
- Optimizer gap probe: `14` targeted cases across optimizer versions 1 through
  8 (`112` matrix entries). Every entry planned successfully. Visible
  boundaries were observed for informational foreign-key optimization,
  unhinted index union, full outer join, sorted `LIMIT` cross apply, and
  computation pushdown through join.

Details:

- [Optimizer Version Matrix Observations](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md)
- [Optimizer-version sensitivity in operator observations](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#subquery_cluster_node-metadata)

## Hint Inventory

The following list covers every hint category in the official query-syntax
Hints section, plus graph hints and function hints as separate official pages.

### Statement Hints

| Hint | Official values | Local verification summary |
| --- | --- | --- |
| `USE_ADDITIONAL_PARALLELISM` | `TRUE`, `FALSE` default | Included in statement-hint and hint-matrix probes. Accepted in the synthetic scan shape. No distinct compact-tree-metadata shape was expected or observed. |
| `OPTIMIZER_VERSION` | `1` to `N`, `latest_version`, `default_version` | `1..8` are exercised by `--optimizer-version-matrix`; `default_version` and `latest_version` are included in statement-hint probes. |
| `OPTIMIZER_STATISTICS_PACKAGE` | `package_name`, `latest` | `latest` is included in statement-hint probes. Custom package names are not covered because the synthetic Omni database has no named statistics package setup. |
| `ALLOW_DISTRIBUTED_MERGE` | `TRUE` default, `FALSE` | Cross-checked across versions. Default and `TRUE` matched. `FALSE` changed only sort or sorted-merge cases in the observed matrix. |
| `LOCK_SCANNED_RANGES` | `exclusive`, `shared` default | Included in statement-hint probes. Read-only PLAN checks are effectively no-op for this hint, and a dedicated read-write `AnalyzeQuery` recheck still showed identical `Scan` metadata for `shared` and `exclusive` in the observed Spanner Omni build. This is natural if read-only and read-write executions share the same logical plan cache and locking is applied below the cached plan. A visible requested lock mode would be contractable if Spanner exposed it, but actual lock conflicts should still be validated with `SPANNER_SYS.LOCK_STATS*`. In principle, the scan condition determines the locked range, so a full scan puts the whole table or index range in scope. `exclusive` is useful mainly for write-after-read contention patterns where earlier waiting can be preferable to repeated abort/retry cycles. |
| `SCAN_METHOD` | `AUTO` default, `BATCH`, `ROW`, `COLUMNAR`, `NO_COLUMNAR` | `BATCH`, `ROW`, `COLUMNAR`, `NO_COLUMNAR` are included. `AUTO` is the absence of the hint and cannot be manually set according to docs. Scan/execution method interaction is summarized separately. |
| `EXECUTION_METHOD` | `DEFAULT`, `BATCH`, `ROW` | `BATCH` and `ROW` are included. `DEFAULT` is the absence of the hint according to docs. `EXECUTION_METHOD=ROW` can remove `RowToDataBlock` / `DataBlockToRow` in observed apply/back-join shapes where `SCAN_METHOD=ROW` alone did not. |
| `USE_UNENFORCED_FOREIGN_KEY` | `TRUE` default, `FALSE` | Included in statement-hint probes against a join shape. The synthetic schema does not model informational foreign-key consistency, so only acceptance/shape is checked. |
| `ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN` | `TRUE`, `FALSE` default | Verified with an `allow_commit_timestamp=true` column. `TRUE` adds a `Timestamp Condition` child link on the scan; `FALSE` leaves only the residual condition in the checked shape. This is a storage-pruning signal that can matter even when the relational plan still reports `Full scan: true`. |

Details:

- [Other Hint Matrix](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#other-hint-matrix)
- [Broad Statement-Hint Query Matrix](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#broad-statement-hint-query-matrix)
- [Scan and execution method notes](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#scan-and-execution-method-hints)
- [ALLOW_DISTRIBUTED_MERGE Cross-Check](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md#allow_distributed_merge-cross-check)

### Table Hints

| Hint | Official values | Local verification summary |
| --- | --- | --- |
| `FORCE_INDEX` | index name string, or `_BASE_TABLE` | Checked with `_BASE_TABLE` and secondary indexes. Back joins are observed when the forced index cannot provide all selected columns. |
| `GROUPBY_SCAN_OPTIMIZATION` | `TRUE`, `FALSE` | Checked in table-hint probes. `TRUE` changed placement of `Distributed Union` around `Aggregate` in the synthetic probe. |
| `SCAN_METHOD` | `AUTO` default, `BATCH`, `ROW`, `COLUMNAR`, `NO_COLUMNAR` | Checked in table-hint and statement-hint positions. Table `SCAN_METHOD=BATCH` can be rejected on the right side of apply joins and related shapes. |
| `INDEX_STRATEGY` | `FORCE_INDEX_UNION` | Accepted for the checked disjunction shape and produced `Union All` over index-scan branches. Rejected shapes returned `InvalidArgument` in broad statement-position probes. |
| `SEEKABLE_KEY_SIZE` | `0` to `16`; requires `FORCE_INDEX` | Checked with `FORCE_INDEX=_BASE_TABLE` and key sizes `0`, `1`, `2` on a two-part primary key. A larger value than the key length was separately checked and rejected. Statement-prefix use is rejected as unsupported; use table-hint position. |

Details:

- [Other Hint Matrix](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#other-hint-matrix)

### Join Hints

| Hint | Official values | Local verification summary |
| --- | --- | --- |
| `FORCE_JOIN_ORDER` | `TRUE`, `FALSE` default | Included in join-hint matrix and statement-position broad probes. Useful when optimizer-version changes come from join reordering or commutativity; it can constrain the join tree/input order, but not every distributed wrapper or back-join placement detail. |
| `JOIN_METHOD` | `HASH_JOIN`, `APPLY_JOIN`, `MERGE_JOIN`, `PUSH_BROADCAST_HASH_JOIN` | Checked across direct `INNER`, `LEFT`, `RIGHT` joins and subquery predicates (`IN`, `EXISTS`, `NOT IN`, `NOT EXISTS`) through statement hints. Push broadcast is rejected in optimizer versions 1 and 2, accepted from version 3. |
| `HASH_JOIN_BUILD_SIDE` | `BUILD_LEFT`, `BUILD_RIGHT`; only with `JOIN_METHOD=HASH_JOIN` | Included in join and subquery matrices. This can fix build/probe orientation for hash joins when `JOIN_METHOD=HASH_JOIN` is already selected. |
| `BATCH_MODE` | `TRUE` default, `FALSE`; only with `JOIN_METHOD=APPLY_JOIN` | `TRUE` produces distributed apply forms for several join/subquery shapes. `FALSE` keeps row-at-a-time apply forms in checked shapes. Some broad statement-position combinations returned `Hint batch_mode results in no plan.` |
| `HASH_JOIN_EXECUTION` | `MULTI_PASS` default, `ONE_PASS`; only with `JOIN_METHOD=HASH_JOIN` | Included in join and subquery matrices. The observed `PlanNode` metadata does not distinguish `MULTI_PASS` from `ONE_PASS`, so PLAN can verify only that a hash join was selected. It cannot verify the requested value or runtime spill/scan-count behavior. |
| `FACTORIZED_MODE` | `FACTORIZE_LEFT`, `FACTORIZE_RIGHT`, `FACTORIZE_BOTH`; INNER JOIN with equality conditions only | Documented join hint (query-syntax Hints section, "Factorized mode") that deduplicates join-key values on one or both sides before the join; most relevant to many-to-many graph traversals. Not yet locally verified on Omni; whether PLAN-level metadata distinguishes factorized execution is unconfirmed. |

Details:

- [Join Hint Matrix](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#join-hint-matrix)
- [Subquery Predicate Statement-Hint Matrix](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#subquery-predicate-statement-hint-matrix)

Join-related optimizer-version changes are often more controllable than a
single `JOIN_METHOD` hint suggests. In particular, v5-style cost-based join
selection can often be decomposed into:

- algorithm choice: `JOIN_METHOD`;
- input/join tree choice: `FORCE_JOIN_ORDER`;
- hash build/probe orientation: `HASH_JOIN_BUILD_SIDE`;
- apply batching and distributed apply shape: `BATCH_MODE`;
- hash runtime strategy intent: `HASH_JOIN_EXECUTION`;
- access path of each input: table hints such as `FORCE_INDEX` and
  `SEEKABLE_KEY_SIZE`.

The practical limit is that these hints still do not expose every physical
decision independently. Distributed wrapper placement, internal implementation
nodes, back-join placement, and some cost/statistics-sensitive choices can
still vary. See
[Optimizer Decision Control And Plan Observability](OPTIMIZER_DECISION_CONTROL_AND_OBSERVABILITY.md)
for the cross-cutting classification of which join decisions are controllable,
visible, and contractable.

### Group Hints

| Hint | Official values | Local verification summary |
| --- | --- | --- |
| `GROUP_METHOD` | `HASH_GROUP`, `STREAM_GROUP` | Checked in `GROUP@{...} BY` position. `HASH_GROUP` and `STREAM_GROUP` drive `Aggregate` metadata used by plan-contract normalization. Statement-prefix position is rejected as `Unsupported hint: GROUP_METHOD.` |

Details:

- [Other Hint Matrix](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#other-hint-matrix)
- `cmd/spanner-query-gen` Omni integration tests for
  `GROUP@{GROUP_METHOD=HASH_GROUP} BY` and
  `GROUP@{GROUP_METHOD=STREAM_GROUP} BY`.

### Graph Hints

Official graph hints reuse the group, join, and table hint categories in GQL
contexts. The graph hints page says only one hint key and value are allowed per
graph hint.

| Graph hint type | Official scope | Local verification summary |
| --- | --- | --- |
| Group | Applies a group hint to `GROUP BY` in a GQL `RETURN` statement. | Checked with `GROUP_METHOD=HASH_GROUP` and `GROUP_METHOD=STREAM_GROUP` in `GRAPH MusicGraph ... RETURN ... GROUP@{...} BY`. |
| Join, graph traversal | Applies join hints to graph traversal. Allowed between two path patterns, edge-to-node, node-to-edge, and graph-subpath-to-edge. Not allowed between two subpaths or subpath-to-node. | Checked with traversal `JOIN_METHOD=APPLY_JOIN` and `JOIN_METHOD=HASH_JOIN` in `GRAPH MusicGraph` probes. |
| Join, match statement | Applies join hints to a GQL `MATCH` statement. | Checked with `MATCH @{JOIN_METHOD=APPLY_JOIN}` and `MATCH @{JOIN_METHOD=HASH_JOIN}`. |
| Table, graph element | Applies table hints to a graph element at the beginning of a pattern filler. | Checked with an element-level `FORCE_INDEX=_BASE_TABLE` probe. |

Details:

- [Other Hint Matrix](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#other-hint-matrix)

### Function Hints

Function hints are documented on the functions reference page, not in the
query-syntax Hints section.

| Hint | Official values / syntax | Local verification summary |
| --- | --- | --- |
| `DISABLE_INLINE` | `function_name() @{DISABLE_INLINE = TRUE}` | Checked with `SHA512(...)`. Default and `DISABLE_INLINE=FALSE` produced the same scalar tree with duplicated function calls. `DISABLE_INLINE=TRUE` added `Compute`, materialized the function result once, and made outer expressions reference that result. |

Details:

- [Function Hint Shapes](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md#function-hint-shapes)

## Gaps And Caveats

- Most hint probes are PLAN-shape checks on an empty synthetic Spanner Omni
  database. They are not latency, CPU, lock-contention, spill, or rows-scanned
  benchmarks.
- `PROFILE` execution statistics are intentionally out of scope for
  `spanner-query-gen` plan contracts. Runtime behavior mentioned in official
  docs, such as lock contention or hash-join spill behavior, is not verified by
  the PLAN-only probes.
- Lack of a controlling hint and lack of plan observability are separate gaps.
  Even if a user cannot force a specific optimizer rewrite, any behavior fixed
  when the plan is produced should ideally be visible through QueryPlan
  operators, child links, scalar predicates, or metadata. Fully runtime-only
  effects such as actual spill, contention, latency, CPU, and rows scanned are
  expected to require PROFILE, Query Stats, or production telemetry.
- Cost-based optimizer improvements can depend on data distribution,
  statistics packages, schema size, and live service behavior. Stable shape in
  the synthetic matrix does not imply the optimizer version has no effect in
  production.
- The broad statement-position matrix intentionally tries some hints in places
  where the grammar accepts a statement hint but the semantic hint is better
  placed elsewhere. Rejections such as statement-prefix `GROUP_METHOD` or
  `SEEKABLE_KEY_SIZE` are recorded as placement evidence, not as unsupported
  hint keys overall.
