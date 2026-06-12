# Operator Verification Follow-up

Observed on 2026-05-07 with Spanner Omni through `spanemuboost`.

Environment:

- `spanner_omni_image`: `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`
- `spanemuboost`: `github.com/apstndb/spanemuboost v0.4.0`
- `spannerplan`: `github.com/apstndb/spannerplan v0.1.9`
- `go`: `go1.26.2 darwin/arm64`
- `optimizer_statistics_package`: `not_pinned`

This follow-up checks remaining uncertainty after the latest
`apstndb/spanner-hacks` `content/ja/operators.md` update.

## SpoolScan Raw Plan

The repeated CTE query:

```sql
WITH CTE AS (
  SELECT 1 AS PK, "foo" AS col
)
SELECT *
FROM CTE c1
JOIN CTE c2 USING (PK);
```

was captured with:

```sh
go run ./tools/spanner-query-plan-shape \
  --case cte \
  --sql 'WITH CTE AS (SELECT 1 AS PK, "foo" AS col) SELECT * FROM CTE c1 JOIN CTE c2 USING (PK)' \
  --output yaml
```

Raw QueryPlan has `display_name: SpoolScan` directly:

```yaml
index: 11
kind: RELATIONAL
display_name: SpoolScan
child_links:
- child_index: 12
  variable: PK_1
- child_index: 13
  variable: col_1
metadata:
  execution_method: Row
  spool_name: CTE
```

The second scan is equivalent:

```text
index=15 display=SpoolScan scan_type= spool_name=CTE
```

Conclusion: for normal repeated-CTE plans observed here, `SpoolScan` is not
raw `display_name: Scan` with `metadata.scan_type: SpoolScan`; it is a distinct
raw `display_name: SpoolScan` with `metadata.spool_name`.

This is a concrete correction candidate for upstream `operators.md`.

## Search Predicate Raw-vs-Reference Mapping

The Full Text Search matrix was captured with both raw YAML and
`spannerplan/plantree/reference`:

```sh
go run ./tools/spanner-query-plan-shape \
  --case full_text_search \
  --output yaml \
  --continue-on-error

go run ./tools/spanner-query-plan-shape \
  --case full_text_search \
  --output reference \
  --continue-on-error
```

In all 15 Full Text Search cases, raw `Search Predicate` child links are now
visible in `spannerplan v0.1.9` reference output as `Search Predicate:` rows in
the `Predicates(identified by ID)` section.

| Case | Raw link target | Reference predicate form |
| --- | --- | --- |
| `full-text-search/search` | `Search Predicate` | `SQUERY(index_name:AlbumTitle_Tokens ...)` |
| `full-text-search/force-index` | `Search Predicate` | `SQUERY(index_name:AlbumTitle_Tokens ...)` |
| `full-text-search/snippet` | `Search Predicate` | `SQUERY(index_name:AlbumTitle_Tokens ...)` |
| `full-text-search/score-order` | `Search Predicate` | `SQUERY(index_name:AlbumTitle_Tokens ...)` |
| `full-text-search/substring` | `Search Predicate` | `SEARCH_SUBSTRING_QUERY(...)` |
| `full-text-search/multi-column-conjunction` | `Function` | `SQUERY(...) AND SQUERY(...)` |
| `full-text-search/multi-column-disjunction` | `Function` | `SQUERY(...) OR SQUERY(...)` |
| `full-text-search/multi-column-negation` | `Search Predicate` | `SQUERY(index_name:AlbumTitle_Tokens ...)` |
| `full-text-search/tokenlist-concat` | `Search Predicate` | `SPAN_MAKE_FIELDED_SQUERY(...)` |
| `full-text-search/partitioned-ordered-index` | `Search Predicate` | `SQUERY(...)` plus `Seek Condition` on the same scan |
| `full-text-search/numeric-array-any` | `Search Predicate` | `SPAN_NUMBER_QUERY_TO_SQUERY(...)` |
| `full-text-search/numeric-array-all` | `Search Predicate` | `SPAN_NUMBER_QUERY_TO_SQUERY(...)` |
| `full-text-search/mixed-accelerated` | `Function` | compound `SQUERY(...)` with residual `Filter Scan` |
| `full-text-search/mixed-stored-filter` | `Function` | compound `SQUERY(...)` with residual `Filter Scan` |
| `full-text-search/mixed-back-join` | `Function` | compound `SQUERY(...)`, residual `Filter Scan`, and back join |

Conclusion: the latest upstream `Search Predicate` description is consistent
with local raw and reference evidence. The useful additional detail is that the
raw child target is `Function` for composite cases and `Search Predicate` for
simple cases.

## Generate Relation Candidates

The following candidate shapes were checked:

```sql
SELECT 1 + 2 AS result;
SELECT * FROM UNNEST([1,2,3]) AS n;
SELECT * FROM UNNEST([STRUCT(1 AS n), STRUCT(2 AS n)]);
SELECT * FROM UNNEST([]) AS n;
SELECT * FROM UNNEST(GENERATE_ARRAY(1, 3)) AS n;
SELECT * FROM UNNEST(GENERATE_DATE_ARRAY(DATE "2026-01-01", DATE "2026-01-03")) AS d;
SELECT * FROM (SELECT 1 AS n UNION ALL SELECT 2 AS n);
WITH RECURSIVE seq AS (...) SELECT * FROM seq;
```

Observed shapes:

- Constant-only `SELECT` uses `Unit Relation`.
- Array literal and generated-array queries use `Array Unnest` plus either
  `Array Constructor` or scalar `Function`.
- `UNION ALL` uses `Union All`, `Union Input`, `Compute`, and `Unit Relation`.
- `WITH RECURSIVE` is rejected by Spanner GoogleSQL in this environment:
  `RECURSIVE is not supported in the WITH clause`.

Conclusion: no positive `Generate Relation` reproduction was found.

Update (2026-06-12): still no reproduction. The official query-operators-leaf
page documents only generic properties for it with no example query, and
`UNNEST(GENERATE_ARRAY(...))` classifies as `array_unnest`.

## Local Split Union

The current documentation-derived query set was regenerated with:

```sh
go run ./tools/spanner-query-plan-shape \
  --case docs \
  --output summary \
  --continue-on-error
```

No `Local Split Union` was observed. The official leaf operator page fetched on
2026-05-07 lists `Generate relation` but did not expose a `Local Split Union`
section in the fetched page. Current evidence still points to requiring
placement/locality-specific configuration or a non-synthetic environment.

Conclusion: keep `Local Split Union` as unreproduced in the sample schema.

Update (2026-06-12): now confirmed unreproducible on Omni 2026.r1-beta
regardless of schema. The official documentation describes the operator as
appearing for placement-table scans, and `CREATE PLACEMENT` fails on Omni
with `Unimplemented: Geo-partitioning is not supported for this
environment`.

## MiniBatchAssign, MiniBatchKeyOrder, and RowCount

The candidate back-join query was checked across optimizer versions 1 through 8:

```sql
SELECT *
FROM Songs@{FORCE_INDEX=SongsBySongName}
ORDER BY SongName DESC
LIMIT 1;
```

Command:

```sh
go run ./tools/spanner-query-plan-shape \
  --case docs \
  --sql 'SELECT * FROM Songs@{FORCE_INDEX=SongsBySongName} ORDER BY SongName DESC LIMIT 1' \
  --optimizer-version-matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

Observed boundary:

- v1-v2: global `Sort Limit` over `Distributed Union` and back join.
- v3-v8: `Distributed Union(preserve_subquery_order=true)` and
  `Distributed Cross Apply(order_preserving=true)`.
- No `MiniBatchAssign`, `MiniBatchKeyOrder`, or `RowCount` appeared in this
  synthetic empty Spanner Omni database, including the exact
  `@{OPTIMIZER_VERSION=5}` query.

Conclusion: the MiniBatch/RowCount plan appears environment-sensitive. It was
observed in DBaaS-style evidence with optimizer version 5, but it is not
reproduced in this empty Spanner Omni synthetic database.

## Create Batch Scalar Children

Raw `Create Batch` child links were inspected in three representative contexts.

Back join over `SongsBySongName`:

```text
CreateBatch#4
  type= variable= child#5:RowToDataBlock
  type= variable=v2.Batch.AlbumId child#18:Reference short=$sort_AlbumId
  type= variable=v2.Batch.SingerId child#19:Reference short=$sort_SingerId
  type= variable=v2.Batch.SongName child#20:Reference short=$sort_SongName
  type= variable=v2.Batch.TrackId child#21:Reference short=$sort_TrackId
  type= variable=v2.Batch.__row_id child#22:Constant short=<typed null>
```

Push Broadcast Hash Join:

```text
CreateBatch#2
  type= variable= child#3:RowToDataBlock
  type= variable=v2.Batch.SingerId child#7:Reference short=$SingerId
```

Graph recursive traversal:

```text
CreateBatch#20
  type= variable= child#21:Distributed Cross Apply
  type= variable=v28.Batch.FeaturingSingerId'6 child#53:Reference short=$FeaturingSingerId'6
  type= variable=v28.Batch.__row_id child#54:Constant short=<typed null>

CreateBatch#22
  type= variable= child#23:Recursive Spool Scan
  type= variable=v26.Batch.tail'_SingerId'16 child#26:Reference short=$tail'_SingerId'16
  type= variable=v26.Batch.__row_id child#27:Constant short=<typed null>
```

Conclusion: `Create Batch` has one relational input, followed by scalar child
links whose variables define fields in the generated batch relation. The fields
depend on the operation: broadcast keys, back-join keys/sort values, traversal
keys, and sometimes `__row_id`.
