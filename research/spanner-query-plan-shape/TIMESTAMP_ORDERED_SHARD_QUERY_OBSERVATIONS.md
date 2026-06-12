# Timestamp-Ordered Shard Query Observations

This note checks whether the plan-shape claims from the Stack Overflow thread
"Is it possible to have efficient timestamp ordered queries in spanner?" are
observable in Spanner execution plans.

Source:

- <https://stackoverflow.com/questions/61999247/is-it-possible-to-have-efficient-timestamp-ordered-queries-in-spanner>
- <https://github.com/apstndb/zenn-contents/blob/main/articles/spanner-query-optimizing-guide.md>
- <https://github.com/gcpug/nouhau/pull/135>
- <https://raw.githubusercontent.com/gcpug/nouhau/spanner/shard/spanner/note/shard/README.md>
- <https://docs.cloud.google.com/spanner/docs/commit-timestamp#optimize>
- <https://medium.com/google-cloud/cloud-spanner-evaluating-commit-timestamp-optimization-for-recent-data-queries-9291cddd85d5>

Observed on 2026-05-08 with Spanner Omni through `spanemuboost` and rendered
with `spannerplan` reference output.

## Schema

The first schema intentionally matches the Stack Overflow post. The queries use
explicit projection instead of `SELECT *`, so the covered and non-covered cases
can be discussed separately.

```sql
CREATE TABLE Foo (
  random_id STRING(22) NOT NULL,
  shard_id INT64 NOT NULL,
  timestamp_order TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
) PRIMARY KEY(random_id);

CREATE INDEX OrderIndex ON Foo(shard_id, timestamp_order);
```

## Findings

- `WHERE shard_id = 0 ORDER BY timestamp_order LIMIT 1` plans as a seek on
  `OrderIndex` with `Local Limit` and no `Sort Limit`. This is the efficient
  single-shard shape.
- `WHERE shard_id < 1`, `WHERE shard_id BETWEEN 1 AND 1`, and
  `WHERE shard_id BETWEEN 0 AND 1` all introduce `Local Sort Limit`.
  The plan therefore still treats these as shard ranges for ordering purposes,
  even when `BETWEEN 1 AND 1` is logically a single value.
- The `HAVING MIN` rewrite plans as `Global Stream Aggregate` over
  `Local Stream Aggregate`, then a small global `Sort Limit`. This confirms the
  article's main workaround is visible in the plan shape.
- In this Spanner Omni `PLAN` check, `GROUPBY_SCAN_OPTIMIZATION=TRUE`, `FALSE`,
  and unspecified produced the same `Stream Aggregate` shape for the
  `HAVING MIN` query. The historical effect of this hint was scan-row-count
  behavior, not a visible plan-shape difference. Ideally that behavioral
  difference would still be observable in the plan metadata, but this check
  does not expose it.
- The arbitrary-limit workaround using `JOIN UNNEST(ARRAY(... ORDER BY ...
  LIMIT n))` plans as: first enumerate shards with stream aggregation, then run
  a per-shard local limit under `Cross Apply`, and finally apply a global
  `Sort Limit`. This matches the article's intended "top N per shard, then
  merge" shape.
- If a non-covering column is requested, the direct query introduces an internal
  back join with `Distributed Cross Apply`. Rewriting the query to first select
  only indexed columns / primary key and then join back to `Foo` in the
  outermost query moves the table lookup after the limit-producing subquery.
  That shape is better aligned with the intended optimization, because the base
  table lookup is driven by the already-limited key set.
- The unmerged `gcpug/nouhau` PR #135 describes the same progression as V1,
  V2/V2.1, and V3:
  - V1 scans too broadly when asking for the latest N rows across a shard
    range.
  - V2/V2.1 enumerates shard values and fetches top N rows per shard, then
    sorts and limits globally. This reduces the index-side work to
    `N * shard_count`.
  - V3 avoids unnecessary back joins by selecting only index-covered columns and
    the primary key in the per-shard top-N subquery, then joining the base table
    only after the global top N is known.
  The `payload` probe below reproduces that V3 shape difference in current
  Spanner Omni plan output.
- The PR review discussion also distinguishes "uses the index" from "uses the
  index efficiently". A shard range such as `BETWEEN 0 AND 9` can still scan
  `OrderIndexDesc`, but if the shard range is not discretized into per-shard
  equality probes, the timestamp predicate may not become a full seek predicate.
  In the observed plan, the direct range and `IN` forms use `seekable_key_size=1`
  and keep the timestamp range as a residual condition, while the per-shard
  `GENERATE_ARRAY` rewrite uses `seekable_key_size=2` and includes the timestamp
  range in the `Seek Condition`.
- The PR review discussion also notes that "full scan" should be reserved for
  the execution-plan-level full scan shape. A query may read many rows because
  an access predicate is weak or not fully seekable without literally being a
  full scan in the plan.
- Newer Spanner plans can expose `Timestamp Condition` on a scan. This changes
  the old decision tree for some recent-data queries: a plan-level full table
  scan on an `allow_commit_timestamp=true` column can still receive
  storage-level timestamp pruning, so a secondary timestamp index is no longer
  automatically required just because the timestamp predicate is not a
  key seek.

These are plan-shape observations only. They do not prove row-scan counts
because the local probe uses an empty synthetic database and `PLAN`, not
`PROFILE`.

## Timestamp Condition Update

The official commit timestamp documentation now describes a recent-data
optimization for `allow_commit_timestamp=true` columns. The qualifying shape is
a `>` or `>=` comparison against a constant expression; additional `AND`
predicates are allowed, while `OR` disqualifies the optimization.

The Medium probe is useful because it shows the visible plan signal. A
non-commit-timestamp column keeps the timestamp filter as only a
`Residual Condition`, while the commit timestamp column adds a
`Timestamp Condition` child link on the table scan. The relational operator
shape can still show `Table Scan (Full scan: true)`, but `PROFILE` rows scanned
and query stats can improve substantially for recent data. The same article
also shows that old data may still scan the full table even with
`Timestamp Condition`.

This means `Timestamp Condition` should be interpreted as a separate storage
pruning signal, not as a replacement for index seekability:

- If the access pattern is "recent rows by commit timestamp" and global
  ordering by timestamp is not required,
  `no_full_scan_without_timestamp_condition` is often a better broad guardrail
  than plain `no_full_scan`; it still rejects unpruned full scans while
  allowing full scans that carry the `Timestamp Condition` storage-pruning
  signal.
- If the access pattern needs ordered pagination or top-N by timestamp, the
  secondary index / sharded index reasoning in the rest of this note still
  applies because `Timestamp Condition` does not provide index order.
- If the query must be efficient for older data as well, the Medium result
  suggests `Timestamp Condition` alone is not enough; a secondary index,
  partitioning strategy, or workload-specific telemetry remains necessary.

## Contract Implications

The optimizing guide is useful as a contract taxonomy, but not every warning is
equally expressible from structural PLAN output.

Directly expressible today:

- `no_full_scan`: rejects plan-level full scans. This catches the guide's
  "Full Scan on OLTP data is an early warning" case, but it does not prove row
  count. A scan with `seekable_key_size=1` can still read too much if the
  timestamp predicate is residual, while a scan with `Timestamp Condition` can
  read fewer rows despite still reporting `full_scan: true`.
- `no_full_scan_without_timestamp_condition`: rejects full scans unless the
  scan has a `Timestamp Condition` child link. This is the practical default
  when recent-data commit timestamp reads are an allowed exception.
- `require_timestamp_condition`: requires a `Timestamp Condition` child link.
  This is the correct structural contract for recent-data commit timestamp
  reads where storage-level pruning is expected.
- `no_full_sort` / `no_explicit_sort`: rejects global/full sort shapes. For
  timestamp-ordered top-N lookups, this is useful when the desired shape should
  stream from index order.
- `no_blocking_operator_under_limit`: rejects blocking descendants under
  `Limit` or `Sort Limit`. This captures the guide's streaming concern more
  precisely than a whole-plan ban on every blocking operator.
- `no_hash_aggregate`: checks the `HAVING MIN` rewrite expectation that
  aggregation can stream from ordered index input.

Useful CEL for current reports:

```cel
operator_edges.exists(e, e.type == "Timestamp Condition")
```

This is equivalent to `require_timestamp_condition`. It confirms that the plan
has the storage-pruning signal, but it does not prove rows-scanned reduction.

```cel
operators.all(o,
  !(o.family == "scan" && o.full_scan) ||
  operator_edges.exists(e,
    e.parent_index == o.index &&
    e.type == "Timestamp Condition"))
```

This is equivalent to `no_full_scan_without_timestamp_condition`.

```cel
operators.exists(o,
  o.scan_target == "OrderIndexDesc" &&
  o.seekable_key_size == "2")
```

This checks the per-shard equality-probe shape where both `shard_id` and
`timestamp_order` participate in the seek. It is stricter than merely requiring
that `OrderIndexDesc` was scanned.

```cel
operators.all(o,
  !(o.scan_type == "table_scan" && o.scan_target == "Foo"))
```

This approximates an index-only expectation, but it is not a correct generic
`no_back_join` rule because it lacks the schema mapping from index target to
base table.

Future schema-aware contracts:

- `require_seekable_scan`: require a specific scan target with a minimum
  normalized seekable key size.
- `index_only_scan`: assert that selected columns are satisfied by an index
  without a base-table lookup.
- `no_back_join`: detect same-base-table index scan plus table scan after
  resolving index ownership. The observed V3 rewrite shows why this needs to
  distinguish "lookup before global limit" from "lookup after global limit".

## Original Query Shapes

```text
SELECT random_id, shard_id, timestamp_order
FROM Foo@{FORCE_INDEX=OrderIndex}
WHERE shard_id = 0
ORDER BY timestamp_order
LIMIT 1
+----+---------------------------------------------------------------------+
| ID | Operator                                                            |
+----+---------------------------------------------------------------------+
|  0 | Global Limit <Row>                                                  |
| *1 | +- Distributed Union on OrderIndex <Row>                            |
|  2 |    +- Serialize Result <Row>                                        |
|  3 |       +- Local Limit <Row>                                          |
|  4 |          +- Local Distributed Union <Row>                           |
|  5 |             +- Filter Scan <Row> (seekable_key_size: 0)             |
| *6 |                +- Index Scan on OrderIndex <Row> (scan_method: Row) |
+----+---------------------------------------------------------------------+
Predicates(identified by ID):
 1: Split Range: ($shard_id = 0)
 6: Seek Condition: ($shard_id = 0)


SELECT random_id, shard_id, timestamp_order
FROM Foo@{FORCE_INDEX=OrderIndex}
WHERE shard_id < 1
ORDER BY timestamp_order
LIMIT 1
+----+--------------------------------------------------------------------------+
| ID | Operator                                                                 |
+----+--------------------------------------------------------------------------+
|  0 | Global Limit <Row>                                                       |
| *1 | +- Distributed Union on OrderIndex <Row> (preserve_subquery_order: true) |
|  2 |    +- Serialize Result <Row>                                             |
|  3 |       +- Local Sort Limit <Row>                                          |
|  4 |          +- Local Distributed Union <Row>                                |
|  5 |             +- Filter Scan <Row> (seekable_key_size: 1)                  |
| *6 |                +- Index Scan on OrderIndex <Row> (scan_method: Row)      |
+----+--------------------------------------------------------------------------+
Predicates(identified by ID):
 1: Split Range: ($shard_id < 1)
 6: Seek Condition: ($shard_id < 1)


SELECT random_id, shard_id, timestamp_order
FROM Foo@{FORCE_INDEX=OrderIndex}
WHERE shard_id BETWEEN 1 AND 1
ORDER BY timestamp_order
LIMIT 1
+----+--------------------------------------------------------------------------+
| ID | Operator                                                                 |
+----+--------------------------------------------------------------------------+
|  0 | Global Limit <Row>                                                       |
| *1 | +- Distributed Union on OrderIndex <Row> (preserve_subquery_order: true) |
|  2 |    +- Serialize Result <Row>                                             |
|  3 |       +- Local Sort Limit <Row>                                          |
|  4 |          +- Local Distributed Union <Row>                                |
|  5 |             +- Filter Scan <Row> (seekable_key_size: 1)                  |
| *6 |                +- Index Scan on OrderIndex <Row> (scan_method: Row)      |
+----+--------------------------------------------------------------------------+
Predicates(identified by ID):
 1: Split Range: BETWEEN($shard_id, 1, 1)
 6: Seek Condition: BETWEEN($shard_id, 1, 1)


SELECT random_id, shard_id, timestamp_order
FROM Foo@{FORCE_INDEX=OrderIndex}
WHERE shard_id BETWEEN 0 AND 1
ORDER BY timestamp_order
LIMIT 1
+----+--------------------------------------------------------------------------+
| ID | Operator                                                                 |
+----+--------------------------------------------------------------------------+
|  0 | Global Limit <Row>                                                       |
| *1 | +- Distributed Union on OrderIndex <Row> (preserve_subquery_order: true) |
|  2 |    +- Serialize Result <Row>                                             |
|  3 |       +- Local Sort Limit <Row>                                          |
|  4 |          +- Local Distributed Union <Row>                                |
|  5 |             +- Filter Scan <Row> (seekable_key_size: 1)                  |
| *6 |                +- Index Scan on OrderIndex <Row> (scan_method: Row)      |
+----+--------------------------------------------------------------------------+
Predicates(identified by ID):
 1: Split Range: BETWEEN($shard_id, 0, 1)
 6: Seek Condition: BETWEEN($shard_id, 0, 1)
```

## `HAVING MIN` Rewrite

```text
SELECT random_id, shard_id, timestamp_order
FROM (
  SELECT
    shard_id,
    ANY_VALUE(random_id HAVING MIN timestamp_order) AS random_id,
    MIN(timestamp_order) AS timestamp_order
  FROM Foo@{FORCE_INDEX=OrderIndex, GROUPBY_SCAN_OPTIMIZATION=TRUE}
  WHERE shard_id < 1
  GROUP BY shard_id
)
ORDER BY timestamp_order
LIMIT 1
+----+------------------------------------------------------------------------+
| ID | Operator                                                               |
+----+------------------------------------------------------------------------+
|  0 | Serialize Result <Row>                                                 |
|  1 | +- Global Sort Limit <Row>                                             |
|  2 |    +- Global Stream Aggregate <Row>                                    |
| *3 |       +- Distributed Union on OrderIndex <Row>                         |
|  4 |          +- Local Stream Aggregate <Row>                               |
|  5 |             +- Local Distributed Union <Row>                           |
|  6 |                +- Filter Scan <Row> (seekable_key_size: 1)             |
| *7 |                   +- Index Scan on OrderIndex <Row> (scan_method: Row) |
+----+------------------------------------------------------------------------+
Predicates(identified by ID):
 3: Split Range: ($shard_id < 1)
 7: Seek Condition: ($shard_id < 1)
```

The same `PLAN` shape was observed with no `GROUPBY_SCAN_OPTIMIZATION` hint and
with `GROUPBY_SCAN_OPTIMIZATION=FALSE`. That does not disprove the historical
effect: the Stack Overflow comment was about rows scanned, while this document
records plan shape only. A follow-up `PROFILE` check with representative data
would be needed to validate the runtime scan-row-count effect.

## Arbitrary-Limit Rewrite

```text
SELECT ff.random_id, ff.shard_id, ff.timestamp_order
FROM (
  SELECT shard_id
  FROM Foo@{FORCE_INDEX=OrderIndex, GROUPBY_SCAN_OPTIMIZATION=TRUE}
  WHERE shard_id < 2
  GROUP BY shard_id
) AS shards
JOIN UNNEST(ARRAY(
  SELECT AS STRUCT f.random_id, f.shard_id, f.timestamp_order
  FROM Foo@{FORCE_INDEX=OrderIndex} AS f
  WHERE f.shard_id = shards.shard_id
  ORDER BY timestamp_order
  LIMIT 2
)) AS ff
ORDER BY ff.timestamp_order
LIMIT 2
+-----+---------------------------------------------------------------------------+
| ID  | Operator                                                                  |
+-----+---------------------------------------------------------------------------+
|   0 | Serialize Result <Row>                                                    |
|   1 | +- Global Sort Limit <Row>                                                |
|   2 |    +- Cross Apply <Row>                                                   |
|   3 |       +- [Input] Global Stream Aggregate <Row>                            |
|  *4 |       |  +- Distributed Union on OrderIndex <Row>                         |
|   5 |       |     +- Local Stream Aggregate <Row>                               |
|   6 |       |        +- Local Distributed Union <Row>                           |
|   7 |       |           +- Filter Scan <Row> (seekable_key_size: 1)             |
|  *8 |       |              +- Index Scan on OrderIndex <Row> (scan_method: Row) |
|  19 |       +- [Map] Local Limit <Row>                                          |
| *20 |          +- Distributed Union on OrderIndex <Row>                         |
|  21 |             +- Local Limit <Row>                                          |
|  22 |                +- Local Distributed Union <Row>                           |
|  23 |                   +- Filter Scan <Row> (seekable_key_size: 0)             |
| *24 |                      +- Index Scan on OrderIndex <Row> (scan_method: Row) |
+-----+---------------------------------------------------------------------------+
Predicates(identified by ID):
  4: Split Range: ($shard_id < 2)
  8: Seek Condition: ($shard_id < 2)
 20: Split Range: ($shard_id_2 = $group_shard_id')
 24: Seek Condition: ($shard_id_2 = $group_shard_id')
```

The `gcpug/nouhau` PR #135 calls this family of rewrites V2/V2.1. In that
article, V2 and V2.1 have the same execution plan but different SQL spelling:
one nests `SELECT ARRAY(...)` in an intermediate row, while the other writes the
`UNNEST(GENERATE_ARRAY(...)), UNNEST(ARRAY(...))` form directly. The important
observable shape is the same: a shard-enumerating input side and a per-shard
limited index scan on the map side.

## Shard Range And Timestamp Seekability

Update (2026-06-12): the seekability of the shard-range form turned out to be
optimizer-version dependent. On Omni 2026.r1-beta, `OPTIMIZER_VERSION` 1-6
discretize the small shard range and seek both keys (`seekable_key_size=2`,
no residual), while versions 7-8 (the current default, observed below) seek
only the shard range and leave the timestamp residual. Plan-level seeks are
not a runtime performance guarantee either way; see
[`../spanner-query-gen/PLAN_REPORT_OPERATOR_COVERAGE_2026-06-12.md`](../spanner-query-gen/PLAN_REPORT_OPERATOR_COVERAGE_2026-06-12.md)
and the spanner-hacks seek-residual notes.

The PR review discussion asked whether a query over
`ShardCreatedAt BETWEEN 0 AND 9` and a timestamp range really "uses the index".
The answer depends on what is meant by "uses": it can scan the selected index
without doing a plan-level full scan, but still fail to make the timestamp range
a fully seekable key prefix.

For this check, use a descending timestamp index:

```sql
CREATE TABLE Foo (
  random_id STRING(22) NOT NULL,
  shard_id INT64 NOT NULL,
  timestamp_order TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
) PRIMARY KEY(random_id);

CREATE INDEX OrderIndexDesc ON Foo(shard_id, timestamp_order DESC);
```

The direct shard-range query scans `OrderIndexDesc`, but only `shard_id` is in
the seek condition. The timestamp range appears as a residual condition, with
only a lower timestamp condition attached to the scan:

```text
SELECT random_id, shard_id, timestamp_order
FROM Foo@{FORCE_INDEX=OrderIndexDesc}
WHERE shard_id BETWEEN 0 AND 9
  AND timestamp_order BETWEEN TIMESTAMP "2018-09-05T09:00:00Z"
                          AND TIMESTAMP "2018-09-05T10:00:00Z"
ORDER BY timestamp_order DESC
LIMIT 100
+----+------------------------------------------------------------------------------+
| ID | Operator                                                                     |
+----+------------------------------------------------------------------------------+
|  0 | Global Limit <Row>                                                           |
| *1 | +- Distributed Union on OrderIndexDesc <Row> (preserve_subquery_order: true) |
|  2 |    +- Serialize Result <Row>                                                 |
|  3 |       +- Local Sort Limit <Row>                                              |
|  4 |          +- Local Distributed Union <Row>                                    |
| *5 |             +- Filter Scan <Row> (seekable_key_size: 1)                      |
| *6 |                +- Index Scan on OrderIndexDesc <Row> (scan_method: Row)      |
+----+------------------------------------------------------------------------------+
Predicates(identified by ID):
 1: Split Range: (($shard_id >= 0) AND ($shard_id <= 9) AND ($timestamp_order >= timestamp (2018-09-05 02:00:00-07:00)) AND ($timestamp_order <= timestamp (2018-09-05 03:00:00-07:00)))
 5: Residual Condition: (($timestamp_order >= timestamp (2018-09-05 02:00:00-07:00)) AND ($timestamp_order <= timestamp (2018-09-05 03:00:00-07:00)))
 6: Seek Condition: (($shard_id >= 0) AND ($shard_id <= 9))
    Timestamp Condition: ($timestamp_order >= timestamp (2018-09-05 02:00:00-07:00))
```

Using `IN (0, ..., 9)` produced the same important shape: `seekable_key_size=1`
with the timestamp range still present as a residual condition.

The per-shard rewrite makes each map-side index probe use shard equality. In
that shape, both `shard_id` and `timestamp_order` are in the seek condition and
`seekable_key_size=2`:

```text
SELECT c.random_id, c.shard_id, c.timestamp_order
FROM UNNEST(GENERATE_ARRAY(0, 9)) AS OneShard,
UNNEST(ARRAY(
  SELECT AS STRUCT f.random_id, f.shard_id, f.timestamp_order
  FROM Foo@{FORCE_INDEX=OrderIndexDesc} AS f
  WHERE f.shard_id = OneShard
    AND f.timestamp_order BETWEEN TIMESTAMP "2018-09-05T09:00:00Z"
                              AND TIMESTAMP "2018-09-05T10:00:00Z"
  ORDER BY timestamp_order DESC
  LIMIT 100
)) AS c
ORDER BY c.timestamp_order DESC
LIMIT 100
+-----+-------------------------------------------------------------------------------+
| ID  | Operator                                                                      |
+-----+-------------------------------------------------------------------------------+
|   0 | Serialize Result <Row>                                                        |
|   1 | +- Global Sort Limit <Row>                                                    |
|   2 |    +- Cross Apply <Row>                                                       |
|   3 |       +- [Input] Array Unnest <Row>                                           |
|   8 |       +- [Map] Local Limit <Row>                                              |
|  *9 |          +- Distributed Union on OrderIndexDesc <Row>                         |
|  10 |             +- Local Limit <Row>                                              |
|  11 |                +- Local Distributed Union <Row>                               |
|  12 |                   +- Filter Scan <Row> (seekable_key_size: 2)                 |
| *13 |                      +- Index Scan on OrderIndexDesc <Row> (scan_method: Row) |
+-----+-------------------------------------------------------------------------------+
Predicates(identified by ID):
  9: Split Range: (($shard_id = $OneShard) AND ($timestamp_order >= timestamp (2018-09-05 02:00:00-07:00)) AND ($timestamp_order <= timestamp (2018-09-05 03:00:00-07:00)))
 13: Seek Condition: ($shard_id = $OneShard) AND (($timestamp_order >= timestamp (2018-09-05 02:00:00-07:00)) AND ($timestamp_order <= timestamp (2018-09-05 03:00:00-07:00)))
     Timestamp Condition: ($timestamp_order >= timestamp (2018-09-05 02:00:00-07:00))
```

This matches the review-comment hypothesis: the risk is not necessarily a
literal full scan, but a shard range that is not decomposed into fully seekable
per-shard probes. The discussion also pointed at F1's filter-tree description:
small integer intervals can be discretized, while large intervals may remain
non-enumerable ranges. A planner hint to force this discretization would make
this behavior easier to control and verify.

## Back-Join Placement

For non-covered projections, add a non-indexed column:

```sql
CREATE TABLE Foo (
  random_id STRING(22) NOT NULL,
  shard_id INT64 NOT NULL,
  timestamp_order TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
  payload STRING(MAX),
) PRIMARY KEY(random_id);

CREATE INDEX OrderIndex ON Foo(shard_id, timestamp_order);
```

Directly projecting `payload` introduces the base-table lookup inside the
timestamp-order plan:

```text
SELECT random_id, shard_id, timestamp_order, payload
FROM Foo@{FORCE_INDEX=OrderIndex}
WHERE shard_id < 1
ORDER BY timestamp_order
LIMIT 1
+-----+------------------------------------------------------------------------------+
| ID  | Operator                                                                     |
+-----+------------------------------------------------------------------------------+
|   0 | Global Limit <Row>                                                           |
|  *1 | +- Distributed Union on OrderIndex <Row> (preserve_subquery_order: true)     |
|   2 |    +- Serialize Result <Row>                                                 |
|  *3 |       +- Distributed Cross Apply <Row> (order_preserving: true)              |
|   4 |          +- [Input] Create Batch <Batch>                                     |
|   5 |          |  +- RowToDataBlock                                                |
|   6 |          |     +- Local Sort Limit <Row>                                     |
|   7 |          |        +- Local Distributed Union <Row>                           |
|   8 |          |           +- Filter Scan <Row> (seekable_key_size: 1)             |
|  *9 |          |              +- Index Scan on OrderIndex <Row> (scan_method: Row) |
|  25 |          +- [Map] Cross Apply <Row>                                          |
|  26 |             +- [Input] KeyRangeAccumulator <Row>                             |
|  27 |             |  +- DataBlockToRow                                             |
|  28 |             |     +- Batch Scan on $v2 <Batch> (scan_method: Batch)          |
|  37 |             +- [Map] Local Distributed Union <Row>                           |
|  38 |                +- Filter Scan <Row> (seekable_key_size: 0)                   |
| *39 |                   +- Table Scan on Foo <Row> (scan_method: Row)              |
+-----+------------------------------------------------------------------------------+
Predicates(identified by ID):
  1: Split Range: ($shard_id < 1)
  3: Split Range: ($random_id' = $sort_random_id)
  9: Seek Condition: ($shard_id < 1)
 39: Seek Condition: ($random_id' = $batched_random_id'2)
```

Rewriting the query to select indexed columns first and join back by primary key
places the base-table lookup after the limiting subquery:

```text
SELECT f.random_id, f.shard_id, f.timestamp_order, f.payload
FROM (
  SELECT random_id, shard_id, timestamp_order
  FROM Foo@{FORCE_INDEX=OrderIndex}
  WHERE shard_id < 1
  ORDER BY timestamp_order
  LIMIT 1
) AS top
JOIN Foo AS f USING (random_id)
ORDER BY top.timestamp_order
LIMIT 1
+-----+-----------------------------------------------------------------------------------+
| ID  | Operator                                                                          |
+-----+-----------------------------------------------------------------------------------+
|   0 | Serialize Result <Row>                                                            |
|   1 | +- Global Limit <Row>                                                             |
|   2 |    +- Cross Apply <Row>                                                           |
|   3 |       +- [Input] Global Limit <Row>                                               |
|  *4 |       |  +- Distributed Union on OrderIndex <Row> (preserve_subquery_order: true) |
|   5 |       |     +- Local Sort Limit <Row>                                             |
|   6 |       |        +- Local Distributed Union <Row>                                   |
|   7 |       |           +- Filter Scan <Row> (seekable_key_size: 1)                     |
|  *8 |       |              +- Index Scan on OrderIndex <Row> (scan_method: Row)         |
| *24 |       +- [Map] Distributed Union on Foo <Row>                                     |
|  25 |          +- Local Limit <Row>                                                     |
|  26 |             +- Local Distributed Union <Row>                                      |
|  27 |                +- Filter Scan <Row> (seekable_key_size: 0)                        |
| *28 |                   +- Table Scan on Foo <Row> (scan_method: Row)                   |
+-----+-----------------------------------------------------------------------------------+
Predicates(identified by ID):
  4: Split Range: ($shard_id < 1)
  8: Seek Condition: ($shard_id < 1)
 24: Split Range: ($random_id_2 = $sort_random_id)
 28: Seek Condition: ($random_id_2 = $sort_random_id)
```

Combining `HAVING MIN` with an outermost back join keeps the stream-aggregate
optimization on the index side and performs the base-table lookup after the
limited key selection:

```text
SELECT f.random_id, f.shard_id, f.timestamp_order, f.payload
FROM (
  SELECT
    shard_id,
    ANY_VALUE(random_id HAVING MIN timestamp_order) AS random_id,
    MIN(timestamp_order) AS timestamp_order
  FROM Foo@{FORCE_INDEX=OrderIndex, GROUPBY_SCAN_OPTIMIZATION=TRUE}
  WHERE shard_id < 1
  GROUP BY shard_id
  ORDER BY timestamp_order
  LIMIT 1
) AS top
JOIN Foo AS f USING (random_id)
ORDER BY top.timestamp_order
LIMIT 1
+-----+------------------------------------------------------------------------------+
| ID  | Operator                                                                     |
+-----+------------------------------------------------------------------------------+
|   0 | Serialize Result <Row>                                                       |
|   1 | +- Global Limit <Row>                                                        |
|   2 |    +- Cross Apply <Row>                                                      |
|   3 |       +- [Input] Global Sort Limit <Row>                                     |
|   4 |       |  +- Global Stream Aggregate <Row>                                    |
|  *5 |       |     +- Distributed Union on OrderIndex <Row>                         |
|   6 |       |        +- Local Stream Aggregate <Row>                               |
|   7 |       |           +- Local Distributed Union <Row>                           |
|   8 |       |              +- Filter Scan <Row> (seekable_key_size: 1)             |
|  *9 |       |                 +- Index Scan on OrderIndex <Row> (scan_method: Row) |
| *39 |       +- [Map] Distributed Union on Foo <Row>                                |
|  40 |          +- Local Limit <Row>                                                |
|  41 |             +- Local Distributed Union <Row>                                 |
|  42 |                +- Filter Scan <Row> (seekable_key_size: 0)                   |
| *43 |                   +- Table Scan on Foo <Row> (scan_method: Row)              |
+-----+------------------------------------------------------------------------------+
Predicates(identified by ID):
  5: Split Range: ($shard_id < 1)
  9: Seek Condition: ($shard_id < 1)
 39: Split Range: ($random_id_3 = $sort_random_id_1)
 43: Seek Condition: ($random_id_3 = $sort_random_id_1)
```

This corresponds to the V3 optimization described in `gcpug/nouhau` PR #135:
when the secondary index does not cover all requested columns, reading full rows
inside each per-shard top-N subquery does a table lookup for every candidate row
(`N * shard_count`). Selecting only index-covered columns plus the primary key,
then joining the base table after the global top N has been selected, limits the
base-table lookup to the final N rows.
