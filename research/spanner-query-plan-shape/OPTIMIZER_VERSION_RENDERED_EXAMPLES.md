# Optimizer Version Rendered Examples

This note captures representative optimizer-version boundaries with rendered
`spannerplan` reference output. It is meant as reviewable evidence for later
Spanner Unofficial Hacks material, not as a stable benchmark.

Observed on 2026-05-08 with Spanner Omni through `spanemuboost`.

Environment:

- `spanemuboost`: `github.com/apstndb/spanemuboost v0.4.0`
- `spannerplan`: `github.com/apstndb/spannerplan v0.1.9`
- `go`: `go1.26.2 darwin/arm64`
- `optimizer_statistics_package`: not pinned

Official source:

- <https://docs.cloud.google.com/spanner/docs/query-optimizer/versions>
- The page was checked on 2026-05-08. It reported optimizer version 8 as the
  current default and latest version, and `Last updated 2026-05-06 UTC`.

Command:

```sh
go run ./tools/spanner-query-plan-shape \
  --case optimizer_gaps \
  --output reference \
  --continue-on-error \
  --timeout=5m \
  --sql '@{OPTIMIZER_VERSION=1}
SELECT SingerId, AlbumId, TrackId
FROM Songs
WHERE REGEXP_CONTAINS(SongName, "^A.*")' \
  --sql '@{OPTIMIZER_VERSION=2}
SELECT SingerId, AlbumId, TrackId
FROM Songs
WHERE REGEXP_CONTAINS(SongName, "^A.*")' \
  --sql '@{OPTIMIZER_VERSION=2} SELECT s.SongGenre FROM Songs AS s ORDER BY SongGenre' \
  --sql '@{OPTIMIZER_VERSION=3} SELECT s.SongGenre FROM Songs AS s ORDER BY SongGenre' \
  --sql '@{OPTIMIZER_VERSION=2}
SELECT
  t.ConcertDate,
  (
    SELECT COUNT(*)
    FROM UNNEST(t.TicketPrices) AS p
    WHERE p > 10
  ) AS expensive_tickets,
  u.VenueName
FROM Concerts AS t
JOIN Venues AS u ON t.VenueId = u.VenueId
ORDER BY expensive_tickets
LIMIT 2' \
  --sql '@{OPTIMIZER_VERSION=3}
SELECT
  t.ConcertDate,
  (
    SELECT COUNT(*)
    FROM UNNEST(t.TicketPrices) AS p
    WHERE p > 10
  ) AS expensive_tickets,
  u.VenueName
FROM Concerts AS t
JOIN Venues AS u ON t.VenueId = u.VenueId
ORDER BY expensive_tickets
LIMIT 2' \
  --sql '@{OPTIMIZER_VERSION=4} SELECT AlbumTitle FROM Songs JOIN Albums ON Albums.AlbumId = Songs.AlbumId' \
  --sql '@{OPTIMIZER_VERSION=5} SELECT AlbumTitle FROM Songs JOIN Albums ON Albums.AlbumId = Songs.AlbumId' \
  --sql '@{OPTIMIZER_VERSION=5}
SELECT a.SingerId, a.AlbumTitle, s.SongName
FROM Albums AS a
FULL OUTER JOIN Songs AS s
  ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
WHERE a.ReleaseDate >= DATE "2020-01-01" OR s.Duration > 180
LIMIT 10' \
  --sql '@{OPTIMIZER_VERSION=6}
SELECT a.SingerId, a.AlbumTitle, s.SongName
FROM Albums AS a
FULL OUTER JOIN Songs AS s
  ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
WHERE a.ReleaseDate >= DATE "2020-01-01" OR s.Duration > 180
LIMIT 10' \
  --sql '@{OPTIMIZER_VERSION=6} SELECT s.SingerId FROM Singers AS s WHERE s.FirstName = "Alice" OR s.LastName = "Smith"' \
  --sql '@{OPTIMIZER_VERSION=7} SELECT s.SingerId FROM Singers AS s WHERE s.FirstName = "Alice" OR s.LastName = "Smith"' \
  --sql '@{OPTIMIZER_VERSION=7, USE_UNENFORCED_FOREIGN_KEY=TRUE} SELECT o.CustomerId FROM FKOrders AS o JOIN FKCustomers AS c ON c.CustomerId = o.CustomerId' \
  --sql '@{OPTIMIZER_VERSION=8, USE_UNENFORCED_FOREIGN_KEY=TRUE} SELECT o.CustomerId FROM FKOrders AS o JOIN FKCustomers AS c ON c.CustomerId = o.CustomerId'
```

## Summary

| Boundary | Documented area | Visible plan-shape change |
| --- | --- | --- |
| v1 to v2 | `REGEXP_CONTAINS` / `LIKE` improvements | A prefix regexp on `SongName` becomes a `STARTS_WITH` split and seek condition instead of a full scan with residual regexp. |
| v2 to v3 | distributed merge union | Sorted results move from a top-level `Sort` above `Distributed Union` to `Distributed Union` with `preserve_subquery_order: true`. |
| v2 to v3 | computation pushdown through `JOIN` | The scalar aggregate over `UNNEST(t.TicketPrices)` is pushed under the distributed join side in v3. |
| v4 to v5 | cost-based join algorithm selection | The unhinted join changes from distributed apply to `Hash Join` with `BloomFilterBuild`. |
| v5 to v6 | full outer join predicate and limit pushdown | The full outer join changes from `Outer Apply` to `Distributed Outer Apply` and scans the release-date index. |
| v6 to v7 | cost-based index union | A disjunction over two indexed columns changes from one index scan to `Union All` over two index scans plus deduplication. |
| v7 to v8 | foreign-key handling | With `USE_UNENFORCED_FOREIGN_KEY=TRUE`, v8 removes the referenced-table join and scans only the referencing table. |

The v4 secondary-index-selection item has separate DBaaS evidence in
[`SPANNER_OPTIMIZER_AND_HINTS.md`](SPANNER_OPTIMIZER_AND_HINTS.md), but the
minimal empty Spanner Omni schema has not reproduced that v3/v4 boundary yet.

## v2: Prefix Predicate Extraction

Version 2 documents improved `REGEXP_CONTAINS` and `LIKE` performance under
certain circumstances. This example shows a prefix regexp becoming a seekable
range.

```text
@{OPTIMIZER_VERSION=1}
SELECT SingerId, AlbumId, TrackId
FROM Songs
WHERE REGEXP_CONTAINS(SongName, "^A.*")
+----+----------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                           |
+----+----------------------------------------------------------------------------------------------------+
| *0 | Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                                          |
|  1 | +- Local Distributed Union <Row>                                                                   |
|  2 |    +- Serialize Result <Row>                                                                       |
| *3 |       +- Filter Scan <Row> (seekable_key_size: 0)                                                  |
|  4 |          +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+----+----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 0: Split Range: REGEXP_CONTAINS($SongName, '^A.*')
 3: Residual Condition: REGEXP_CONTAINS($SongName, '^A.*')


@{OPTIMIZER_VERSION=2}
SELECT SingerId, AlbumId, TrackId
FROM Songs
WHERE REGEXP_CONTAINS(SongName, "^A.*")
+----+--------------------------------------------------------------------+
| ID | Operator                                                           |
+----+--------------------------------------------------------------------+
| *0 | Distributed Union on SongsBySongName <Row>                         |
|  1 | +- Local Distributed Union <Row>                                   |
|  2 |    +- Serialize Result <Row>                                       |
|  3 |       +- Filter Scan <Row> (seekable_key_size: 1)                  |
| *4 |          +- Index Scan on SongsBySongName <Row> (scan_method: Row) |
+----+--------------------------------------------------------------------+
Predicates(identified by ID):
 0: Split Range: STARTS_WITH($SongName, 'A')
 4: Seek Condition: STARTS_WITH($SongName, 'A')
```

## v3: Distributed Merge Union

Version 3 documents distributed merge union. In raw `QueryPlan`, the observed
shape is a `Distributed Union` with `preserve_subquery_order: true`, not a
separate `display_name` named `Distributed Merge Union`.

```text
@{OPTIMIZER_VERSION=2}
SELECT s.SongGenre
FROM Songs AS s
ORDER BY SongGenre
+----+---------------------------------------------------------------------------+
| ID | Operator                                                                  |
+----+---------------------------------------------------------------------------+
|  0 | Serialize Result <Row>                                                    |
|  1 | +- Sort <Row>                                                             |
|  2 |    +- Distributed Union on Songs <Row>                                    |
|  3 |       +- Local Distributed Union <Row>                                    |
|  4 |          +- Table Scan on Songs <Row> (Full scan, scan_method: Automatic) |
+----+---------------------------------------------------------------------------+


@{OPTIMIZER_VERSION=3}
SELECT s.SongGenre
FROM Songs AS s
ORDER BY SongGenre
+----+---------------------------------------------------------------------------+
| ID | Operator                                                                  |
+----+---------------------------------------------------------------------------+
|  0 | Distributed Union on Songs <Row> (preserve_subquery_order: true)          |
|  1 | +- Serialize Result <Row>                                                 |
|  2 |    +- Sort <Row>                                                          |
|  3 |       +- Local Distributed Union <Row>                                    |
|  4 |          +- Table Scan on Songs <Row> (Full scan, scan_method: Automatic) |
+----+---------------------------------------------------------------------------+
```

## v3: Computation Pushdown Through Join

Version 3 documents pushing more computations through joins. This example is
based on the official query-optimizer-version page's `Concerts` / `Venues`
pattern. The local fixture keeps `Venues` minimal; the JSON-oriented columns in
the linked Spanner JSON examples are irrelevant to this plan boundary.

```text
@{OPTIMIZER_VERSION=2}
SELECT
  t.ConcertDate,
  (
    SELECT COUNT(*)
    FROM UNNEST(t.TicketPrices) AS p
    WHERE p > 10
  ) AS expensive_tickets,
  u.VenueName
FROM Concerts AS t
JOIN Venues AS u ON t.VenueId = u.VenueId
ORDER BY expensive_tickets
LIMIT 2
+-----+---------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                    |
+-----+---------------------------------------------------------------------------------------------+
|   0 | Serialize Result <Row>                                                                      |
|   1 | +- Global Sort Limit <Row>                                                                  |
|   2 |    +- Distributed Union on Concerts <Row>                                                   |
|   3 |       +- Local Sort Limit <Row>                                                             |
|   4 |          +- Cross Apply <Row>                                                               |
|  *5 |             +- [Input] Distributed Cross Apply <Row>                                        |
|   6 |             |  +- [Input] Create Batch <Row>                                                |
|   7 |             |  |  +- Local Distributed Union <Row>                                          |
|   8 |             |  |     +- Compute Struct <Row>                                                |
|   9 |             |  |        +- Table Scan on Concerts <Row> (Full scan, scan_method: Automatic) |
|  17 |             |  +- [Map] Cross Apply <Row>                                                   |
|  18 |             |     +- [Input] KeyRangeAccumulator <Row>                                      |
|  19 |             |     |  +- Batch Scan on $v4 <Row> (scan_method: Row)                          |
|  23 |             |     +- [Map] Local Distributed Union <Row>                                    |
|  24 |             |        +- Filter Scan <Row> (seekable_key_size: 0)                            |
| *25 |             |           +- Table Scan on Venues <Row> (scan_method: Row)                    |
|  35 |             +- [Map] Stream Aggregate <Row> (scalar_aggregate: true)                        |
| *36 |                +- Filter <Row>                                                              |
|  37 |                   +- Array Unnest <Row>                                                     |
+-----+---------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  5: Split Range: ($VenueId_1 = $VenueId)
 25: Seek Condition: ($VenueId_1 = $batched_VenueId)
 36: Condition: ($p > 10)


@{OPTIMIZER_VERSION=3}
SELECT
  t.ConcertDate,
  (
    SELECT COUNT(*)
    FROM UNNEST(t.TicketPrices) AS p
    WHERE p > 10
  ) AS expensive_tickets,
  u.VenueName
FROM Concerts AS t
JOIN Venues AS u ON t.VenueId = u.VenueId
ORDER BY expensive_tickets
LIMIT 2
+-----+-----------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                            |
+-----+-----------------------------------------------------------------------------------------------------+
|   0 | Global Limit <Row>                                                                                  |
|   1 | +- Distributed Union on Concerts <Row> (preserve_subquery_order: true)                              |
|   2 |    +- Serialize Result <Row>                                                                        |
|   3 |       +- Local Sort Limit <Row>                                                                     |
|  *4 |          +- Distributed Cross Apply <Row>                                                           |
|   5 |             +- [Input] Create Batch <Batch>                                                         |
|   6 |             |  +- RowToDataBlock                                                                    |
|   7 |             |     +- Local Distributed Union <Row>                                                  |
|   8 |             |        +- Cross Apply <Row>                                                           |
|   9 |             |           +- [Input] Table Scan on Concerts <Row> (Full scan, scan_method: Automatic) |
|  13 |             |           +- [Map] Stream Aggregate <Row> (scalar_aggregate: true)                    |
| *14 |             |              +- Filter <Row>                                                          |
|  15 |             |                 +- Array Unnest <Row>                                                 |
|  25 |             +- [Map] Local Sort Limit <Row>                                                         |
|  26 |                +- Cross Apply <Row>                                                                 |
|  27 |                   +- [Input] KeyRangeAccumulator <Row>                                              |
|  28 |                   |  +- DataBlockToRow                                                              |
|  29 |                   |     +- Batch Scan on $v4 <Batch> (scan_method: Batch)                           |
|  36 |                   +- [Map] Local Distributed Union <Row>                                            |
|  37 |                      +- Filter Scan <Row> (seekable_key_size: 0)                                    |
| *38 |                         +- Table Scan on Venues <Row> (scan_method: Row)                            |
+-----+-----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  4: Split Range: ($VenueId_1 = $VenueId)
 14: Condition: ($p > 10)
 38: Seek Condition: ($VenueId_1 = $batched_VenueId')
```

## v5: Cost-Based Join Algorithm Selection

Version 5 documents cost-based join algorithm selection between hash and apply
join. This empty-schema probe is still useful because the unhinted shape changes
from distributed apply to hash join at the v4/v5 boundary.

```text
@{OPTIMIZER_VERSION=4}
SELECT AlbumTitle
FROM Songs
JOIN Albums ON Albums.AlbumId = Songs.AlbumId
+-----+-------------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                              |
+-----+-------------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                                             |
|  *1 | +- Distributed Cross Apply <Row>                                                                      |
|   2 |    +- [Input] Create Batch <Batch>                                                                    |
|   3 |    |  +- RowToDataBlock                                                                               |
|   4 |    |     +- Local Distributed Union <Row>                                                             |
|   5 |    |        +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
|   8 |    +- [Map] Serialize Result <Row>                                                                    |
|   9 |       +- Cross Apply <Row>                                                                            |
|  10 |          +- [Input] DataBlockToRow                                                                    |
|  11 |          |  +- Batch Scan on $v2 <Batch> (scan_method: Batch)                                         |
|  14 |          +- [Map] Local Distributed Union <Row>                                                       |
| *15 |             +- Filter Scan <Row> (seekable_key_size: 0)                                               |
|  16 |                +- Index Scan on AlbumsByAlbumTitle <Row> (Full scan, scan_method: Row)                |
+-----+-------------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  1: Split Range: ($AlbumId_1 = $AlbumId)
 15: Residual Condition: ($AlbumId_1 = $batched_AlbumId')


@{OPTIMIZER_VERSION=5}
SELECT AlbumTitle
FROM Songs
JOIN Albums ON Albums.AlbumId = Songs.AlbumId
+-----+-------------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                              |
+-----+-------------------------------------------------------------------------------------------------------+
|   0 | Serialize Result <Row>                                                                                |
|  *1 | +- Hash Join <Row> (join_type: INNER)                                                                 |
|   2 |    +- [Build] BloomFilterBuild <Row>                                                                  |
|   3 |    |  +- Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                                    |
|   4 |    |     +- Local Distributed Union <Row>                                                             |
|   5 |    |        +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
|   9 |    +- [Probe] Distributed Union on AlbumsByAlbumTitle <Row>                                           |
|  10 |       +- Local Distributed Union <Row>                                                                |
| *11 |          +- Filter Scan <Row> (seekable_key_size: 0)                                                  |
|  12 |             +- Index Scan on AlbumsByAlbumTitle <Row> (Full scan, scan_method: Automatic)             |
+-----+-------------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  1: Condition: ($AlbumId_1 = $AlbumId)
 11: Residual Condition: BLOOM_FILTER_MATCH($existence_filter, $AlbumId_1)
```

## v6: Full Outer Join Pushdown

Version 6 documents improved limit and predicate pushdown through full outer
joins. The visible change here is from local `Outer Apply` over the base table
to `Distributed Outer Apply` over `AlbumsByReleaseDateTitleDesc`.

```text
@{OPTIMIZER_VERSION=5}
SELECT a.SingerId, a.AlbumTitle, s.SongName
FROM Albums AS a
FULL OUTER JOIN Songs AS s
  ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
WHERE a.ReleaseDate >= DATE "2020-01-01" OR s.Duration > 180
LIMIT 10
+-----+---------------------------------------------------------------------------------------+
| ID  | Operator                                                                              |
+-----+---------------------------------------------------------------------------------------+
|   0 | Global Limit <Row>                                                                    |
|   1 | +- Distributed Union on Albums <Row> (split_ranges_aligned)                           |
|   2 |    +- Serialize Result <Row>                                                          |
|   3 |       +- Local Limit <Row>                                                            |
|   4 |          +- Local Distributed Union <Row>                                             |
|  *5 |             +- Filter <Row>                                                           |
|   6 |                +- Outer Apply <Row>                                                   |
|   7 |                   +- [Input] Table Scan on Albums <Row> (Full scan, scan_method: Row) |
|  12 |                   +- [Map] Local Distributed Union <Row>                              |
|  13 |                      +- Filter Scan <Row> (seekable_key_size: 0)                      |
| *14 |                         +- Table Scan on Songs <Row> (scan_method: Row)               |
+-----+---------------------------------------------------------------------------------------+
Predicates(identified by ID):
  5: Condition: (($ReleaseDate >= 18262 unix days (2020-01-01)) OR ($Duration_1 > 180))
 14: Seek Condition: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))


@{OPTIMIZER_VERSION=6}
SELECT a.SingerId, a.AlbumTitle, s.SongName
FROM Albums AS a
FULL OUTER JOIN Songs AS s
  ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
WHERE a.ReleaseDate >= DATE "2020-01-01" OR s.Duration > 180
LIMIT 10
+-----+-----------------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                                  |
+-----+-----------------------------------------------------------------------------------------------------------+
|   0 | Global Limit <Row>                                                                                        |
|   1 | +- Distributed Union on AlbumsByReleaseDateTitleDesc <Row>                                                |
|   2 |    +- Serialize Result <Row>                                                                              |
|   3 |       +- Local Limit <Row>                                                                                |
|  *4 |          +- Filter <Row>                                                                                  |
|  *5 |             +- Distributed Outer Apply <Row>                                                              |
|   6 |                +- [Input] Create Batch <Row>                                                              |
|   7 |                |  +- Local Distributed Union <Row>                                                        |
|   8 |                |     +- Compute Struct <Row>                                                              |
|   9 |                |        +- Index Scan on AlbumsByReleaseDateTitleDesc <Row> (Full scan, scan_method: Row) |
|  20 |                +- [Map] Cross Apply <Row>                                                                 |
|  21 |                   +- [Input] KeyRangeAccumulator <Row>                                                    |
|  22 |                   |  +- Batch Scan on $v2 <Row> (scan_method: Row)                                        |
|  28 |                   +- [Map] Local Distributed Union <Row>                                                  |
|  29 |                      +- Filter Scan <Row> (seekable_key_size: 0)                                          |
| *30 |                         +- Table Scan on Songs <Row> (scan_method: Row)                                   |
+-----+-----------------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  4: Condition: (($ReleaseDate' >= 18262 unix days (2020-01-01)) OR ($Duration_1 > 180))
  5: Split Range: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))
 30: Seek Condition: (($SingerId_1 = $batched_SingerId) AND ($AlbumId_1 = $batched_AlbumId))
```

## v7: Cost-Based Index Union

Version 7 documents cost-based selection of index union plans. This disjunction
changes from a single index-scan shape in v6 to a union of two index-scan
branches in v7.

```text
@{OPTIMIZER_VERSION=6}
SELECT s.SingerId
FROM Singers AS s
WHERE s.FirstName = "Alice" OR s.LastName = "Smith"
+----+---------------------------------------------------------------------------+
| ID | Operator                                                                  |
+----+---------------------------------------------------------------------------+
| *0 | Distributed Union on SingersByFirstLastName <Row>                         |
|  1 | +- Local Distributed Union <Row>                                          |
|  2 |    +- Serialize Result <Row>                                              |
|  3 |       +- Filter Scan <Row> (seekable_key_size: 2)                         |
| *4 |          +- Index Scan on SingersByFirstLastName <Row> (scan_method: Row) |
+----+---------------------------------------------------------------------------+
Predicates(identified by ID):
 0: Split Range: (($FirstName = 'Alice') OR ($LastName = 'Smith'))
 4: Seek Condition: (($FirstName = 'Alice') OR ($LastName = 'Smith'))


@{OPTIMIZER_VERSION=7}
SELECT s.SingerId
FROM Singers AS s
WHERE s.FirstName = "Alice" OR s.LastName = "Smith"
+-----+------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                 |
+-----+------------------------------------------------------------------------------------------+
|   0 | Serialize Result <Row>                                                                   |
|   1 | +- Hash Aggregate <Row>                                                                  |
|   2 |    +- Union All <Row>                                                                    |
|   3 |       +- Union Input                                                                     |
|  *4 |       |  +- Distributed Union on SingersByFirstLastName <Row>                            |
|   5 |       |     +- Local Distributed Union <Row>                                             |
|   6 |       |        +- Filter Scan <Row> (seekable_key_size: 0)                               |
|  *7 |       |           +- Index Scan on SingersByFirstLastName <Row> (scan_method: Automatic) |
|  18 |       +- Union Input                                                                     |
| *19 |          +- Distributed Union on SingersByLastName <Row>                                 |
|  20 |             +- Local Distributed Union <Row>                                             |
|  21 |                +- Filter Scan <Row> (seekable_key_size: 0)                               |
| *22 |                   +- Index Scan on SingersByLastName <Row> (scan_method: Automatic)      |
+-----+------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  4: Split Range: ($FirstName' = 'Alice')
  7: Seek Condition: IS_NOT_DISTINCT_FROM($FirstName', 'Alice')
 19: Split Range: ($LastName'2 = 'Smith')
 22: Seek Condition: IS_NOT_DISTINCT_FROM($LastName'2, 'Smith')
```

## v8: Foreign-Key Handling

Version 8 documents other improvements including foreign-key handling. With
`USE_UNENFORCED_FOREIGN_KEY=TRUE`, v7 still joins the referenced table, while
v8 removes that join and scans only the referencing table.

```text
@{OPTIMIZER_VERSION=7, USE_UNENFORCED_FOREIGN_KEY=TRUE}
SELECT o.CustomerId
FROM FKOrders AS o
JOIN FKCustomers AS c ON c.CustomerId = o.CustomerId
+-----+---------------------------------------------------------------------------------+
| ID  | Operator                                                                        |
+-----+---------------------------------------------------------------------------------+
|   0 | Distributed Union on FKOrders <Row>                                             |
|  *1 | +- Distributed Cross Apply <Row>                                                |
|   2 |    +- [Input] Create Batch <Batch>                                              |
|   3 |    |  +- RowToDataBlock                                                         |
|   4 |    |     +- Local Distributed Union <Row>                                       |
|   5 |    |        +- Table Scan on FKOrders <Row> (Full scan, scan_method: Automatic) |
|   8 |    +- [Map] Serialize Result <Row>                                              |
|   9 |       +- Cross Apply <Row>                                                      |
|  10 |          +- [Input] KeyRangeAccumulator <Row>                                   |
|  11 |          |  +- DataBlockToRow                                                   |
|  12 |          |     +- Batch Scan on $v2 <Batch> (scan_method: Batch)                |
|  15 |          +- [Map] Local Distributed Union <Row>                                 |
|  16 |             +- Filter Scan <Row> (seekable_key_size: 0)                         |
| *17 |                +- Table Scan on FKCustomers <Row> (scan_method: Row)            |
+-----+---------------------------------------------------------------------------------+
Predicates(identified by ID):
  1: Split Range: ($CustomerId_1 = $CustomerId)
 17: Seek Condition: ($CustomerId_1 = $batched_CustomerId')


@{OPTIMIZER_VERSION=8, USE_UNENFORCED_FOREIGN_KEY=TRUE}
SELECT o.CustomerId
FROM FKOrders AS o
JOIN FKCustomers AS c ON c.CustomerId = o.CustomerId
+----+---------------------------------------------------------------------------+
| ID | Operator                                                                  |
+----+---------------------------------------------------------------------------+
|  0 | Distributed Union on FKOrders <Row>                                       |
|  1 | +- Local Distributed Union <Row>                                          |
|  2 |    +- Serialize Result <Row>                                              |
|  3 |       +- Table Scan on FKOrders <Row> (Full scan, scan_method: Automatic) |
+----+---------------------------------------------------------------------------+
```
