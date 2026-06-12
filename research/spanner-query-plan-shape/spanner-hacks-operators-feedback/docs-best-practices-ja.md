# spanner-hacks operators evidence: SQL best-practices shapes

Generated from `go run ./tools/spanner-query-plan-shape --case docs --output reference --continue-on-error` on 2026-05-06 with Spanner Omni.

```text
=== best-practices/index-auto ===
SELECT s.SingerId FROM Singers AS s WHERE s.LastName = 'Smith'
+----+----------------------------------------------------------------------------+
| ID | Operator                                                                   |
+----+----------------------------------------------------------------------------+
| *0 | Distributed Union on SingersByLastName <Row>                               |
|  1 | +- Local Distributed Union <Row>                                           |
|  2 |    +- Serialize Result <Row>                                               |
|  3 |       +- Filter Scan <Row> (seekable_key_size: 0)                          |
| *4 |          +- Index Scan on SingersByLastName <Row> (scan_method: Automatic) |
+----+----------------------------------------------------------------------------+
Predicates(identified by ID):
 0: Split Range: ($LastName = 'Smith')
 4: Seek Condition: IS_NOT_DISTINCT_FROM($LastName, 'Smith')


=== best-practices/force-index ===
SELECT s.SingerId FROM Singers@{FORCE_INDEX=SingersByLastName} AS s WHERE s.LastName = 'Smith'
+----+----------------------------------------------------------------------------+
| ID | Operator                                                                   |
+----+----------------------------------------------------------------------------+
| *0 | Distributed Union on SingersByLastName <Row>                               |
|  1 | +- Local Distributed Union <Row>                                           |
|  2 |    +- Serialize Result <Row>                                               |
|  3 |       +- Filter Scan <Row> (seekable_key_size: 0)                          |
| *4 |          +- Index Scan on SingersByLastName <Row> (scan_method: Automatic) |
+----+----------------------------------------------------------------------------+
Predicates(identified by ID):
 0: Split Range: ($LastName = 'Smith')
 4: Seek Condition: IS_NOT_DISTINCT_FROM($LastName, 'Smith')


=== best-practices/force-index-covering ===
SELECT s.SingerId, s.FirstName FROM Singers@{FORCE_INDEX=SingersByLastName} AS s WHERE s.LastName = 'Smith'
+----+----------------------------------------------------------------------------+
| ID | Operator                                                                   |
+----+----------------------------------------------------------------------------+
| *0 | Distributed Union on SingersByLastName <Row>                               |
|  1 | +- Local Distributed Union <Row>                                           |
|  2 |    +- Serialize Result <Row>                                               |
|  3 |       +- Filter Scan <Row> (seekable_key_size: 0)                          |
| *4 |          +- Index Scan on SingersByLastName <Row> (scan_method: Automatic) |
+----+----------------------------------------------------------------------------+
Predicates(identified by ID):
 0: Split Range: ($LastName = 'Smith')
 4: Seek Condition: IS_NOT_DISTINCT_FROM($LastName, 'Smith')


=== best-practices/order-by-auto-index ===
SELECT a.AlbumTitle, a.ReleaseDate FROM Albums AS a ORDER BY a.ReleaseDate, a.AlbumTitle DESC
+----+-----------------------------------------------------------------------------------------------+
| ID | Operator                                                                                      |
+----+-----------------------------------------------------------------------------------------------+
|  0 | Distributed Union on AlbumsByReleaseDateTitleDesc <Row>                                       |
|  1 | +- Local Distributed Union <Row>                                                              |
|  2 |    +- Serialize Result <Row>                                                                  |
|  3 |       +- Index Scan on AlbumsByReleaseDateTitleDesc <Row> (Full scan, scan_method: Automatic) |
+----+-----------------------------------------------------------------------------------------------+


=== best-practices/order-by-index ===
SELECT a.AlbumTitle, a.ReleaseDate FROM Albums@{FORCE_INDEX=AlbumsByReleaseDateTitleDesc} AS a ORDER BY a.ReleaseDate, a.AlbumTitle DESC
+----+-----------------------------------------------------------------------------------------------+
| ID | Operator                                                                                      |
+----+-----------------------------------------------------------------------------------------------+
|  0 | Distributed Union on AlbumsByReleaseDateTitleDesc <Row>                                       |
|  1 | +- Local Distributed Union <Row>                                                              |
|  2 |    +- Serialize Result <Row>                                                                  |
|  3 |       +- Index Scan on AlbumsByReleaseDateTitleDesc <Row> (Full scan, scan_method: Automatic) |
+----+-----------------------------------------------------------------------------------------------+


=== best-practices/order-by-desc-limit-back-join-optimizer-version-5 ===
@{OPTIMIZER_VERSION=5} SELECT * FROM Songs@{FORCE_INDEX=SongsBySongName} ORDER BY SongName DESC LIMIT 1
+-----+-------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                        |
+-----+-------------------------------------------------------------------------------------------------+
|   0 | Global Limit <Row>                                                                              |
|   1 | +- Distributed Union on SongsBySongName <Row> (preserve_subquery_order: true)                   |
|   2 |    +- Serialize Result <Row>                                                                    |
|  *3 |       +- Distributed Cross Apply <Row> (order_preserving: true)                                 |
|   4 |          +- [Input] Create Batch <Batch>                                                        |
|   5 |          |  +- RowToDataBlock                                                                   |
|   6 |          |     +- Local Sort Limit <Row>                                                        |
|   7 |          |        +- Local Distributed Union <Row>                                              |
|   8 |          |           +- Index Scan on SongsBySongName <Row> (Full scan, scan_method: Automatic) |
|  23 |          +- [Map] Cross Apply <Row>                                                             |
|  24 |             +- [Input] KeyRangeAccumulator <Row>                                                |
|  25 |             |  +- DataBlockToRow                                                                |
|  26 |             |     +- Batch Scan on $v2 <Batch> (scan_method: Batch)                             |
|  37 |             +- [Map] Local Distributed Union <Row>                                              |
|  38 |                +- Filter Scan <Row> (seekable_key_size: 0)                                      |
| *39 |                   +- Table Scan on Songs <Row> (scan_method: Row)                               |
+-----+-------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  3: Split Range: (($SingerId' = $sort_SingerId) AND ($AlbumId' = $sort_AlbumId) AND ($TrackId' = $sort_TrackId))
 39: Seek Condition: (($SingerId' = $batched_SingerId'2) AND ($AlbumId' = $batched_AlbumId'2) AND ($TrackId' = $batched_TrackId'2))


=== best-practices/order-by-primary-key ===
SELECT * FROM Singers ORDER BY SingerId
+----+--------------------------------------------------------------------------+
| ID | Operator                                                                 |
+----+--------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row>                                       |
|  1 | +- Local Distributed Union <Row>                                         |
|  2 |    +- Serialize Result <Row>                                             |
|  3 |       +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
+----+--------------------------------------------------------------------------+


=== best-practices/no-order-by ===
SELECT * FROM Singers
+----+--------------------------------------------------------------------------+
| ID | Operator                                                                 |
+----+--------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row>                                       |
|  1 | +- Local Distributed Union <Row>                                         |
|  2 |    +- Serialize Result <Row>                                             |
|  3 |       +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
+----+--------------------------------------------------------------------------+


=== best-practices/like ===
SELECT a.AlbumTitle FROM Albums a WHERE a.AlbumTitle LIKE 'Blue%'
+----+-----------------------------------------------------------------------+
| ID | Operator                                                              |
+----+-----------------------------------------------------------------------+
| *0 | Distributed Union on AlbumsByAlbumTitle <Row>                         |
|  1 | +- Local Distributed Union <Row>                                      |
|  2 |    +- Serialize Result <Row>                                          |
|  3 |       +- Filter Scan <Row> (seekable_key_size: 1)                     |
| *4 |          +- Index Scan on AlbumsByAlbumTitle <Row> (scan_method: Row) |
+----+-----------------------------------------------------------------------+
Predicates(identified by ID):
 0: Split Range: STARTS_WITH($AlbumTitle, 'Blue')
 4: Seek Condition: STARTS_WITH($AlbumTitle, 'Blue')


=== best-practices/starts-with ===
SELECT a.AlbumTitle FROM Albums a WHERE STARTS_WITH(a.AlbumTitle, 'Blue')
+----+-----------------------------------------------------------------------+
| ID | Operator                                                              |
+----+-----------------------------------------------------------------------+
| *0 | Distributed Union on AlbumsByAlbumTitle <Row>                         |
|  1 | +- Local Distributed Union <Row>                                      |
|  2 |    +- Serialize Result <Row>                                          |
|  3 |       +- Filter Scan <Row> (seekable_key_size: 1)                     |
| *4 |          +- Index Scan on AlbumsByAlbumTitle <Row> (scan_method: Row) |
+----+-----------------------------------------------------------------------+
Predicates(identified by ID):
 0: Split Range: STARTS_WITH($AlbumTitle, 'Blue')
 4: Seek Condition: STARTS_WITH($AlbumTitle, 'Blue')

```
