# Query Execution Operator Plan Shape Observations

Observed on 2026-05-06 with Spanner Omni through `spanemuboost`.

This document is observed vocabulary evidence for the plan normalizer. Golden
fixtures, generated schemas, and unit tests are the normative contract for
v1alpha `spanner-query-gen plan-report` behavior.

No observable query optimizer plan/operator vocabulary difference has been
found between these Spanner Omni observations and Cloud Spanner DBaaS plans used
for comparison. Keep treating this as environment-specific evidence, because
DBaaS Spanner can receive optimizer/operator changes before a corresponding
Spanner Omni release.

Environment:

- `spanner_omni_image`: `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`
- `spanner_omni_image_digest`: `sha256:e98a088fa66d4a87dbb560d729bf21d998bb843f6018bd8dc118fe320e671886`
- `spanemuboost`: `github.com/apstndb/spanemuboost v0.4.0`
- `spannerplan`: `github.com/apstndb/spannerplan v0.1.8`
- `go`: `go1.26.2 darwin/arm64`
- `optimizer_version`: `not_pinned`
- `optimizer_statistics_package`: `not_pinned`

Additional follow-up checks on 2026-05-07 used
`github.com/apstndb/spannerplan v0.1.9` and are recorded in
[`OPERATOR_VERIFICATION_FOLLOWUP.md`](OPERATOR_VERIFICATION_FOLLOWUP.md).

Source pages:

- <https://docs.cloud.google.com/spanner/docs/query-execution-operators>
- <https://docs.cloud.google.com/spanner/docs/query-execution-plans>
- <https://docs.cloud.google.com/spanner/docs/query-operators-leaf>
- <https://docs.cloud.google.com/spanner/docs/query-operators-unary>
- <https://docs.cloud.google.com/spanner/docs/query-operators-binary>
- <https://docs.cloud.google.com/spanner/docs/query-operators-n-ary>
- <https://docs.cloud.google.com/spanner/docs/query-operators-distributed>
- <https://docs.cloud.google.com/spanner/docs/query-operators-scalar-subqueries>
- <https://docs.cloud.google.com/spanner/docs/query-operators-array-subqueries>
- <https://docs.cloud.google.com/spanner/docs/query-operators-struct-constructor>
- <https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax>
- <https://docs.cloud.google.com/spanner/docs/reference/standard-sql/functions-reference#function_hints>
- <https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax>
- <https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-query-statements#graph_hints>
- <https://docs.cloud.google.com/spanner/docs/full-text-search>
- <https://docs.cloud.google.com/spanner/docs/full-text-search/tokenization>
- <https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes>
- <https://docs.cloud.google.com/spanner/docs/full-text-search/query-overview>
- <https://docs.cloud.google.com/spanner/docs/full-text-search/ranked-search>
- <https://docs.cloud.google.com/spanner/docs/full-text-search/numeric-indexes>
- <https://docs.cloud.google.com/spanner/docs/full-text-search/substring-search>
- <https://docs.cloud.google.com/spanner/docs/full-text-search/search-multiple-columns>
- <https://docs.cloud.google.com/spanner/docs/full-text-search/mix-full-text-and-non-text-queries>
- <https://docs.cloud.google.com/spanner/docs/sql-best-practices>
- <https://docs.cloud.google.com/spanner/docs/query-optimizer/versions>

The reusable probe is:

```sh
go run ./tools/spanner-query-plan-shape --case docs --output compact-tree-metadata
```

`compact-tree-metadata` is the canonical one-line observation format for this
research note. It follows visible PlanNode child links, renders structural links
as edge labels such as `-[Build]->`, `-[Probe]->`, `-[Input]->`, and `-[Map]->`,
and keeps metadata annotations. A child link is visible when the child PlanNode
kind is `RELATIONAL` or the link type is `Scalar`, matching
`spannerplan.QueryPlan.IsVisible`. Hidden scalar links are grouped by child
PlanNode `DisplayName`, such as `Function[Residual Condition]`,
`Reference[Output]`, or `Array Constructor`. Add `--compact-tree-indexes` only when
PlanNode indexes are needed for cross-checking with raw output.

`--case docs` uses the GoogleSQL examples from the operator pages and the
introductory linked execution-plan and SQL best-practice pages. `ML.PREDICT`,
PostgreSQL snippets, named-schema placeholders, and examples written only as
`SELECT ...` placeholders are not included because this probe uses a local
synthetic GoogleSQL schema.

The `leaf/empty-relation` probe intentionally uses the documented
`SELECT * FROM Albums LIMIT 0` example from the leaf operator page as the
primary fixture. The shorter `SELECT 1 LIMIT 0` shape is also known to produce
`Serialize Result > Empty Relation`, but it is kept as ad hoc evidence rather
than the documentation-derived case.

Spanner DML syntax examples are collected separately:

```sh
go run ./tools/spanner-query-plan-shape \
  --case dml \
  --output compact-tree-metadata \
  --continue-on-error
```

The DML case uses `ReadWriteTransaction.AnalyzeQuery` because DML planning may
need a read-write transaction even when the tool asks only for `PLAN`.

Change-stream table-valued function planning is collected separately because it
requires a `CREATE CHANGE STREAM` schema object:

```sh
go run ./tools/spanner-query-plan-shape \
  --case tvf \
  --output compact-tree-metadata \
  --continue-on-error
```

Function hint planning is collected separately because `DISABLE_INLINE` changes
scalar expression placement more than high-level operator families:

```sh
go run ./tools/spanner-query-plan-shape \
  --case function_hint \
  --output nodes
```

Use `--output yaml` or `--output json` for the same raw plan protobuf data when
machine-readable scalar operator inspection is useful.

Full Text Search planning is collected separately because it requires generated
`TOKENLIST` columns and `CREATE SEARCH INDEX` objects:

```sh
go run ./tools/spanner-query-plan-shape \
  --case full_text_search \
  --output compact-tree-metadata \
  --continue-on-error
```

The `full_text_search` case uses a dedicated `SearchAlbums` schema rather than
the broader documentation operator schema. It covers documented examples around
`SEARCH`, `SEARCH_SUBSTRING`, multi-column search, mixed text and non-text
predicates, explicit search-index hints, `SNIPPET`, ranked `SCORE` ordering,
`TOKENLIST_CONCAT`, partitioned ordered search indexes, and numeric array
predicates backed by search indexes.

## `subquery_cluster_node` Metadata

Observed SELECT plans support using `subquery_cluster_node` as a general
distributed-fragment signal, with one important filter: Local Distributed
Unions also have this metadata and should be ignored when applying Spanner's
root-partitionable rule.

This observation comes from raw PlanNode metadata and `compact-tree-metadata`
inspection. Human-readable `spannerplan` output modes such as `reference` hide
`subquery_cluster_node` by design, so those rendered plans do not show this
signal directly.

On 2026-05-06, the documentation-derived query set produced
`subquery_cluster_node` on these display names:

- `Distributed Union`, non-local: 49 nodes
- `Distributed Union`, `call_type: Local`: 63 nodes
- `Distributed Cross Apply`: 4 nodes
- `Push Broadcast Hash Join`: 1 node

The same query set produced no `Distributed ...` or `Push Broadcast ...` node
without `subquery_cluster_node`, and no non-candidate display name with
`subquery_cluster_node`. The join matrix showed the same pattern for
`Distributed Cross Apply`, `Distributed Outer Apply`, non-local and Local
`Distributed Union`, `Push Broadcast Hash Join`, and `Push Broadcast Hash Join
Outer Apply`.

DML plans are different: `Apply Mutations` did not expose
`subquery_cluster_node` in the observed DML matrix, while distributed apply and
union nodes inside those DML plans did. Therefore this metadata is suitable for
SELECT root-partitionable review, but not as the sole definition of every
documented distributed operator family across DML.

Optimizer-version sensitivity can be checked with statement hints for versions
1 through 8:

```sh
go run ./tools/spanner-query-plan-shape \
  --case docs \
  --optimizer-version-matrix \
  --allow-distributed-merge-matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

The matrix uses leading statement hints such as `@{OPTIMIZER_VERSION=5}` rather
than client query options. If a query already has a statement hint, the probe
replaces only `OPTIMIZER_VERSION` and preserves the other hint assignments.
The `ALLOW_DISTRIBUTED_MERGE` dimension repeats each query with the hint
unspecified, `TRUE`, and `FALSE`.
Detailed observed diffs are recorded in
[`OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md`](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md).

Adding only `CREATE UNIQUE INDEX UniqueIndex_SingerName ON Singers(FirstName,
LastName)` to the `docs` schema was compared against the baseline `docs` schema
with `--output compact-tree-metadata`, `--output reference`, and raw `--output nodes`.
All three diffs were empty on Spanner Omni 2026.r1-beta. The DML probes still
use DML-only schema additions so the documentation-query probe remains scoped to
the query-operator examples.

The join hint matrix is broader than the documentation examples:

```sh
go run ./tools/spanner-query-plan-shape \
  --case join_matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

It covers `INNER JOIN`, `LEFT JOIN`, and `RIGHT JOIN` against the currently
documented join hints, plus the legacy `JOIN_TYPE=APPLY_JOIN` spelling that
appears in older query-plan examples.

Other statement, table, group, and graph hints from the Spanner GoogleSQL query
syntax Hints section can be checked with:

```sh
go run ./tools/spanner-query-plan-shape \
  --case hint_matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

Subquery predicates can also become join-like plan shapes. This probe checks
`IN`, `EXISTS`, `NOT IN`, and `NOT EXISTS` with the same hints as statement
hints:

```sh
go run ./tools/spanner-query-plan-shape \
  --case subquery_join_hint_matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

The broad statement-hint query matrix applies documented hint variants in the
query-statement prefix position to every documentation-derived query case:

```sh
go run ./tools/spanner-query-plan-shape \
  --case statement_hint_query_matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

This matrix includes statement hints plus documented table, group, and join
hints that are syntactically plausible before `query_expr`. Graph-specific
hints remain in the graph hint matrix because they have graph-specific
positions.

## Full Text Search

Observed Full Text Search plans add a small set of search-specific PlanNode
vocabulary:

- Text search queries include a `TVF` PlanNode with metadata
  `name=Search Query Conversion`, followed by `VerifyDeterminism`. These are
  normalized as `search_query_conversion_tvf` and `verify_determinism`.
- The observed `SEARCH_SUBSTRING` plan did not include `Search Query
  Conversion` or `VerifyDeterminism`; it directly used a substring search
  index scan and a `Search Predicate`.
- Search index access is represented as a normal `Scan` PlanNode with metadata
  `scan_type=SearchIndexScan` and `scan_target=<search index name>`. The
  normalized metadata value is `scan_type: search_index_scan`; it is not a
  separate operator family.
- The search expression appears as a `Search Predicate` PlanNode reached through
  a `Search Predicate` child link. This is normalized as `search_predicate`.
- `SNIPPET(...)` required a back join to the base table in the observed plan,
  because the displayed text column was not returned directly from the search
  index.
- `ORDER BY SCORE(...) DESC` introduced an explicit `Sort`; the
  `TOKENLIST_CONCAT(...)` ranked query with `LIMIT` used `Sort Limit`.
- The partitioned ordered search index case with `PARTITION BY SingerId` and
  `ORDER BY ReleaseTimestamp DESC` did not introduce an explicit sort for
  `WHERE SingerId = 1 ... ORDER BY ReleaseTimestamp DESC LIMIT 10`; it used the
  search index scan with both `Search Predicate` and `Seek Condition` links.
- Numeric array predicates `ARRAY_INCLUDES_ANY` and `ARRAY_INCLUDES_ALL` also
  used `SearchIndexScan` plus `Search Predicate` when backed by a
  `TOKENIZE_NUMBER(..., comparison_type=>"equality")` generated token column.
- Mixed full-text and non-text predicates over tokenized columns used
  `SearchIndexScan` plus a residual `Filter Scan` in the observed plans.
  Selecting a non-stored column such as `Cover` produced a back join to the base
  table.

Full regenerated Full Text Search results are recorded in [`COMPACT_TREE_METADATA_OBSERVATIONS.md`](COMPACT_TREE_METADATA_OBSERVATIONS.md#full-text-search).
Shareable `spannerplan/plantree/reference` output is recorded in
the full-text-search feedback evidence (delivered upstream; removed from this tree on 2026-06-12, see git history).
With `spannerplan v0.1.9`, the reference renderer marks `SearchIndex Scan`
rows and prints `Search Predicate:` annotations in the
`Predicates(identified by ID)` section.

Raw-vs-reference follow-up confirmed that all 15 Full Text Search cases with
raw `Search Predicate` child links are visible in the v0.1.9 reference output.
Simple cases link to a scalar `Search Predicate` node, while composite cases
can link to a scalar `Function` node whose descendants contain multiple
`Search Predicate` nodes.

## Compact-Tree-Metadata Shapes

The full regenerated `docs` case table is recorded in [`COMPACT_TREE_METADATA_OBSERVATIONS.md`](COMPACT_TREE_METADATA_OBSERVATIONS.md#docs).

The `best-practices/order-by-desc-limit-back-join-optimizer-version-5` query is
the same shape family as the Cloud Spanner example that can contain
`MiniBatchAssign`, `MiniBatchKeyOrder`, and `RowCount` with optimizer version 5.
Spanner Omni 2026.r1-beta produced a simpler back-join plan in an empty
synthetic database, including the exact `@{OPTIMIZER_VERSION=5}` query and the
optimizer-version matrix from 1 through 8. These operators should therefore be
treated as optimizer-version and environment-sensitive, and the DBaaS example
should remain the positive evidence until a stable synthetic reproduction is
found.

The `unary/minor-sort-order-by-partial-key`,
`unary/minor-sort-limit-order-by-partial-key`, and
`unary/minor-sort-stream-aggregate` rows are local synthetic probes added to
separate `Minor Sort` / `Minor Sort Limit` from full `Sort` / `Sort Limit`.
The `cte/spool-build` row is a repeated-CTE probe that observes `SpoolBuild`
and `SpoolScan`.

The dedicated `cte` case compares CTE content and reference count. In Spanner
Omni 2026.r1-beta, single-reference CTEs were inlined for constant rows,
`SHA256(...)`, `CURRENT_TIMESTAMP()`, and base-table scans. Repeated references
introduced `SpoolBuild` and `SpoolScan` in all of those successful cases. The
base-table repeated-reference case spooled a distributed scan subtree.

Raw QueryPlan follow-up showed normal repeated-CTE `SpoolScan` as raw
`PlanNode.display_name: SpoolScan` with `metadata.spool_name`, not as raw
`display_name: Scan` with `metadata.scan_type: SpoolScan`. Recursive graph
plans still use the separate `Recursive Spool Scan` display name.

The regenerated CTE result table is recorded in [`COMPACT_TREE_METADATA_OBSERVATIONS.md`](COMPACT_TREE_METADATA_OBSERVATIONS.md#cte).

`GENERATE_UUID()` was checked as an additional non-deterministic function. A
single-reference CTE planned as `Serialize Result > Compute > Unit Relation`,
but the repeated-reference variant returned an internal error for optimizer
versions 1 through 8 and for the default optimizer setting in this Omni build,
so it is not included in the no-error built-in `cte` case.

The documented `Distributed Merge Union` operator is represented in the raw
plan as `Distributed Union` with `preserve_subquery_order: true`. The
`distributed/distributed-merge-union` probe uses an `ORDER BY` query to observe
that metadata shape; it should not be read as evidence for a separate raw
`PlanNode.displayName` named `Distributed Merge Union`.

The current documentation-derived and candidate-query probes still have no
positive `Generate Relation` or `Local Split Union` reproduction. Constant-only
queries use `Unit Relation`; array generators use `Array Unnest` plus scalar
function or constructor nodes. Local Split Union likely needs placement or
locality-specific configuration that is absent from the synthetic schema.

The `tvf/change-stream` row confirms that Spanner Omni accepts a
`READ_<change_stream_name>` table-valued function after `CREATE CHANGE STREAM`
and emits `ChangeStream TVF`.

## Function Hint Shapes

The function hint probes are based on the Spanner GoogleSQL function-hints
documentation. `DISABLE_INLINE` works only on top-level scalar functions, so
the probe computes `SHA512(s.SingerInfo)` in a subquery and references the
result twice from outer `SUBSTRING(CAST(x AS STRING), ...)` expressions.

Regenerated operator shapes are recorded in [`COMPACT_TREE_METADATA_OBSERVATIONS.md`](COMPACT_TREE_METADATA_OBSERVATIONS.md#function-hint). Scalar-level observations from `nodes` / YAML / JSON remain:

- `function-hint/default_inline`: the scalar tree contains two separate `SHA512($SingerInfo)` function nodes, one under each `SUBSTR(CAST<STRING>(...))` reference.
- `function-hint/disable_inline_false`: same shape and scalar tree as the default-inline case.
- `function-hint/disable_inline_true`: a `Compute` operator materializes one `SHA512($SingerInfo)` function node as `$x`; both outer `SUBSTR(CAST<STRING>($x), ...)` expressions reference `$x`.

`compact-tree-metadata` and `reference` can show the extra `Compute` operator for
`DISABLE_INLINE=TRUE`, but the scalar-level distinction is clearest in
`--output nodes`, `--output yaml`, or `--output json` because those outputs keep
`Function`, `Reference`, and `Constant` nodes.

## DML Compact-Tree-Metadata Results

The DML probes cover the documented `INSERT`, `UPDATE`, and `DELETE` syntax,
including `INSERT IGNORE`, `INSERT OR IGNORE`, `INSERT OR UPDATE`, `ON CONFLICT
DO NOTHING`, `ON CONFLICT DO UPDATE`, `ASSERT_ROWS_MODIFIED`, and `THEN RETURN`
forms.

`ASSERT_ROWS_MODIFIED` is included because it is part of the documented syntax,
but Spanner Omni 2026.r1-beta returned `Unimplemented` for `PLAN`:
`ASSERT_ROWS_MODIFIED is not supported.`

Regenerated DML results are recorded in [`COMPACT_TREE_METADATA_OBSERVATIONS.md`](COMPACT_TREE_METADATA_OBSERVATIONS.md#dml).

## Schema-Sensitive Best-Practice Variants

Some SQL best-practice examples intentionally change the schema before rerunning
the same query. The `docs` case uses a superset schema, so a few optimizer
choices reflect the final schema state rather than the intermediate one.

These variants were checked separately with the pre-improvement schema:

```sh
go run ./tools/spanner-query-plan-shape \
  --ddl <tmp>/spanner-docs-best-practices-base.sql \
  --sql 'SELECT s.SingerId, s.FirstName FROM Singers@{FORCE_INDEX=SingersByLastName} AS s WHERE s.LastName = "Smith"' \
  --sql 'SELECT a.AlbumTitle, a.ReleaseDate FROM Albums AS a ORDER BY a.ReleaseDate, a.AlbumTitle DESC' \
  --output compact-tree-metadata
```

Observed:

- Non-covering `SingersByLastName`: `Distributed Union > Distributed Cross Apply > Create Batch > RowToDataBlock > Filter Scan > Scan > Serialize Result > Cross Apply > KeyRangeAccumulator > DataBlockToRow`.
- No release-date index for `ORDER BY`: `Distributed Union > Serialize Result > Sort > Scan`.

## Row/DataBlock Conversion and Method Hints

The Spanner SQL best-practices documentation distinguishes scan method from
execution method:

- `SCAN_METHOD` controls scan processing (`AUTO`, `BATCH`, `ROW`, `COLUMNAR`,
  `NO_COLUMNAR`).
- `EXECUTION_METHOD` controls row-oriented versus batch-oriented query
  execution at the statement level, but it is still a hint and the optimizer
  decides the method for each individual operator.

Method-hint observations are now represented by the `hint_matrix` regenerated results in [`COMPACT_TREE_METADATA_OBSERVATIONS.md`](COMPACT_TREE_METADATA_OBSERVATIONS.md#hint-matrix), especially the `SCAN_METHOD`, `EXECUTION_METHOD`, and `ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN` rows. The older ad hoc non-covering-index variants should be regenerated as built-in cases before being used as normative plan-shape evidence.

The same pattern was observed for `JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN`:
`EXECUTION_METHOD=ROW` removed `RowToDataBlock` and `DataBlockToRow`, while
`SCAN_METHOD=ROW` did not.

This suggests `RowToDataBlock` and `DataBlockToRow` are best modeled as
row/batch boundary conversion operators. They can be affected by
`EXECUTION_METHOD`, but should not be treated as directly controlled by
`SCAN_METHOD` alone.

## Join Hint Matrix

All 36 join matrix cases completed successfully on Spanner Omni. Regenerated results are recorded in [`COMPACT_TREE_METADATA_OBSERVATIONS.md`](COMPACT_TREE_METADATA_OBSERVATIONS.md#join-matrix).

Notable observations:

- `LEFT JOIN@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN}` exposes
  `Push Broadcast Hash Join Outer Apply`, which is not listed as a standalone
  operator page title. The plan-report normalizer treats it as
  `push_broadcast_hash_join`.
- `BATCH_MODE=TRUE` changes apply joins into distributed apply forms. The
  internal `Cross Apply` or `Outer Apply` below the distributed wrapper is
  classified separately so `no_standalone_apply_join` can ignore wrapper
  internals without hiding a real standalone apply join.
- The matrix only proves this synthetic schema and optimizer environment; it is
  intended to catch naming and topology classes, not to exhaustively prove every
  physical plan choice Spanner can produce.

## Other Hint Matrix

The `hint_matrix` case covers statement, table, group, and graph hints from the
Spanner GoogleSQL query syntax Hints section, excluding join hints that have
dedicated matrices above. On the observed Spanner Omni environment, 39 of 40
cases completed successfully. Regenerated results are recorded in [`COMPACT_TREE_METADATA_OBSERVATIONS.md`](COMPACT_TREE_METADATA_OBSERVATIONS.md#hint-matrix).

Notable observations:

- Most statement hints are accepted by `AnalyzeQuery` but do not necessarily
  change the compact-tree-metadata operator shape in this synthetic schema.
- `GROUPBY_SCAN_OPTIMIZATION=TRUE` changed the placement of `Distributed Union`
  around `Aggregate` in this probe.
- `ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN` cannot be validated from the compact-tree-metadata
  operator shape alone because `TRUE` and `FALSE` have the same compact-tree-metadata
  shape.
  The built-in schema now marks `ModificationTime` with
  `allow_commit_timestamp = true`; Spanner Omni produced the same plan-level
  signal without a locality group. With
  `ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=TRUE`, the reference plan contains a
  `Timestamp Condition` on the table scan; with `FALSE`, only the residual
  condition remains. `--output compact-tree-metadata` exposes that difference in a
  one-line shape by annotating the `Scan` node with
  `links: Timestamp Condition`. A locality group with age-based tiered storage
  remains relevant for the documented SSD/HDD performance goal, but it is not
  required to observe the `Timestamp Condition` plan predicate in this probe.
  The same `Timestamp Condition` was also observed with an inferred timestamp
  query parameter, for example `WHERE s.ModificationTime > @updatedAt`.
  This can change index-design decisions for recent-data reads: a plan may
  still display `Full scan: true`, but `Timestamp Condition` is a separate
  storage-pruning signal. Use PROFILE or Query Stats to validate rows-scanned
  reduction when that distinction matters.
- `INDEX_STRATEGY=FORCE_INDEX_UNION` produced a `Union All` over index-scan
  branches.
- `SEEKABLE_KEY_SIZE=0/1/2` is paired with `FORCE_INDEX` as required by
  Spanner. The probe uses `Albums@{FORCE_INDEX=_BASE_TABLE}` because the base
  table primary key is exactly `(SingerId, AlbumId)`. `SEEKABLE_KEY_SIZE=3`
  was checked separately and returned `InvalidArgument`, as expected for a
  value larger than the base table key length.
- `LOCK_SCANNED_RANGES=shared/exclusive` was first covered by read-only
  statement-hint probes, where it is effectively a no-op. A dedicated
  `lock_hints` case now also captures read-write `AnalyzeQuery` output for the
  same point-read query. Raw YAML for both read-only and read-write probes
  showed identical `Scan` metadata and no visible requested lock-mode field.
  This is consistent with an execution model where read-only and read-write
  executions share the same logical plan cache and locking is applied below the
  cached plan. The requested lock mode is not currently a PLAN-contract surface
  in the observed Spanner Omni build, even though actual lock conflicts remain
  verifiable through `SPANNER_SYS.LOCK_STATS*`.
- Graph hints are accepted on match, traversal, element, and group positions.
  A graph traversal `JOIN_METHOD=HASH_JOIN` produced a regular `Hash Join`,
  while several other graph join-hint placements retained apply-style shapes.
- A previous probe using `FORCE_INDEX=SingersByFirstLastName,
  SEEKABLE_KEY_SIZE=2` returned an empty-description `InvalidArgument` even
  though `FORCE_INDEX` was present. That points to seekable-prefix eligibility,
  not to the documented `FORCE_INDEX` requirement.

Reference output for the timestamp predicate pushdown probe:

```text
@{allow_timestamp_predicate_pushdown=TRUE}
SELECT s.SingerInfo
FROM Singers s
WHERE s.ModificationTime > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 12 HOUR)
Predicates(identified by ID):
 3: Residual Condition: ($ModificationTime > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), 12, HOUR))
 4: Timestamp Condition: ($ModificationTime >= IF((TIMESTAMP_SUB(CURRENT_TIMESTAMP(), 12, HOUR) < timestamp (0-12-31 16:07:02-07:52)), timestamp (0-12-31 16:07:02-07:52), TIMESTAMP_ADD(TIMESTAMP_SUB(CURRENT_TIMESTAMP(), 12, HOUR), 1, NANOSECOND)))

@{allow_timestamp_predicate_pushdown=FALSE}
SELECT s.SingerInfo
FROM Singers s
WHERE s.ModificationTime > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 12 HOUR)
Predicates(identified by ID):
 3: Residual Condition: ($ModificationTime > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), 12, HOUR))
```

## Subquery Predicate Statement-Hint Matrix

All 48 subquery statement-hint matrix cases completed successfully on Spanner
Omni. Regenerated results are recorded in [`COMPACT_TREE_METADATA_OBSERVATIONS.md`](COMPACT_TREE_METADATA_OBSERVATIONS.md#subquery-join-hint-matrix).

Notable observations:

- `IN` and `EXISTS` use `Semi Apply` for apply-style plans and can be forced
  to `Hash Join`, `Merge Join`, or `Push Broadcast Hash Join Semi Apply` with
  statement hints.
- `NOT IN` and `NOT EXISTS` use `Anti-Semi Apply` for apply-style plans and can
  be forced to `Hash Join`, `Merge Join`, or
  `Push Broadcast Hash Join Anti Semi Apply` with statement hints.
- `BATCH_MODE=TRUE` produces `Distributed Semi Apply` or
  `Distributed Anti Semi Apply`. Distributed semi and anti-semi wrappers are
  distinct normalized families. The internal `Semi Apply` below each
  distributed wrapper is classified separately so standalone-apply contracts do
  not double-count wrapper internals.

## Broad Statement-Hint Query Matrix

The `statement_hint_query_matrix` case checked 40 documented hint variants
against 51 documentation-derived queries on Spanner Omni 2026.r1-beta. It ran
2040 entries total: 1654 completed successfully and 386 returned errors.

This matrix is intentionally broader than `hint_matrix`: it answers whether a
hint can be written in query-statement prefix position for a broad set of query
shapes, not whether that hint is useful for each shape.

Error summary:

| Hint variant | Successes | Errors | Main observation |
| --- | ---: | ---: | --- |
| `GROUPBY_SCAN_OPTIMIZATION=TRUE` | 0 | 51 | `Unsupported hint: GROUPBY_SCAN_OPTIMIZATION.` in statement prefix position. Use table hint position. |
| `GROUPBY_SCAN_OPTIMIZATION=FALSE` | 0 | 51 | Same as above. |
| `GROUP_METHOD=HASH_GROUP` | 0 | 51 | `Unsupported hint: GROUP_METHOD.` in statement prefix position. Use `GROUP@{...} BY`. |
| `GROUP_METHOD=STREAM_GROUP` | 0 | 51 | Same as above. |
| `FORCE_INDEX=_BASE_TABLE, SEEKABLE_KEY_SIZE=0` | 0 | 51 | `Unsupported hint: SEEKABLE_KEY_SIZE.` in statement prefix position. Use table hint position with `FORCE_INDEX`. |
| `FORCE_INDEX=_BASE_TABLE, SEEKABLE_KEY_SIZE=1` | 0 | 51 | Same as above. |
| `FORCE_INDEX=_BASE_TABLE, SEEKABLE_KEY_SIZE=2` | 0 | 51 | Same as above. |
| `INDEX_STRATEGY=FORCE_INDEX_UNION` | 38 | 13 | Accepted only for some query shapes; rejected cases returned an empty-description `InvalidArgument`. |
| `JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE` | 43 | 8 | Some subquery/sort/CTE shapes returned `Hint batch_mode results in no plan.` |
| `JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN` | 43 | 8 | Same no-plan error set as above in this synthetic schema. |

All other checked hint variants completed for all 51 query cases:

- `USE_ADDITIONAL_PARALLELISM=TRUE/FALSE`
- `OPTIMIZER_VERSION=default_version/latest_version`
- `OPTIMIZER_STATISTICS_PACKAGE=latest`
- `ALLOW_DISTRIBUTED_MERGE=TRUE/FALSE`
- `LOCK_SCANNED_RANGES=exclusive/shared`
- statement `SCAN_METHOD=BATCH/ROW/COLUMNAR/NO_COLUMNAR`
- `EXECUTION_METHOD=BATCH/ROW`
- `USE_UNENFORCED_FOREIGN_KEY=TRUE/FALSE`
- `ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=TRUE/FALSE`
- `FORCE_INDEX=_BASE_TABLE`
- `FORCE_JOIN_ORDER=TRUE/FALSE`
- `JOIN_METHOD=HASH_JOIN/APPLY_JOIN/MERGE_JOIN`
- `JOIN_METHOD=HASH_JOIN` with `HASH_JOIN_BUILD_SIDE=BUILD_LEFT/BUILD_RIGHT`
- `JOIN_METHOD=HASH_JOIN` with `HASH_JOIN_EXECUTION=MULTI_PASS/ONE_PASS`
- `JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE`

The important practical result is that the query-statement grammar is useful
for broad probing, but not every documented table/group hint is accepted in
statement prefix position by Spanner Omni. The broad matrix records those
differences explicitly instead of assuming compatibility from the grammar.

## Normalization Impact

The documentation examples and additional hint matrices exposed operator names
that were not previously in the plan-report operator-family enum. The
normalization now includes these families:

- `anti_semi_apply`
- `array_subquery`
- `array_unnest`
- `bloom_filter_build`
- `change_stream_tvf`
- `compute`
- `compute_struct`
- `create_batch`
- `data_block_to_row`
- `distributed_anti_semi_apply`
- `distributed_anti_semi_apply_internal_apply`
- `distributed_semi_apply`
- `distributed_semi_apply_internal_apply`
- `empty_relation`
- `filter`
- `key_range_accumulator`
- `limit`
- `push_broadcast_hash_join` also covers the observed
  `Push Broadcast Hash Join Outer Apply`,
  `Push Broadcast Hash Join Semi Apply`, and
  `Push Broadcast Hash Join Anti Semi Apply` spellings.
- `random_id_assign`
- `recursive_spool_scan`
- `row_to_data_block`
- `scalar_subquery`
- `search_predicate`
- `search_query_conversion_tvf`
- `semi_apply`
- `union_input`
- `unit_relation`
- `verify_determinism`

Full Text Search also exposed `SearchIndexScan` as `Scan.scan_type` metadata;
the normalized metadata spelling is `search_index_scan`.
