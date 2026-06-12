# spanner-hacks operators evidence: execution, leaf, and unary operators

Generated from `go run ./tools/spanner-query-plan-shape --case docs --output reference --continue-on-error` on 2026-05-06 with Spanner Omni.

```text
=== execution-plans/simple-scan ===
SELECT s.SongName FROM Songs AS s
+----+-------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                        |
+----+-------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                                       |
|  1 | +- Local Distributed Union <Row>                                                                |
|  2 |    +- Serialize Result <Row>                                                                    |
|  3 |       +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+----+-------------------------------------------------------------------------------------------------+


=== execution-plans/aggregate ===
SELECT s.SingerId, COUNT(*) AS SongCount FROM Songs AS s WHERE s.SingerId < 100 GROUP BY s.SingerId
+----+--------------------------------------------------------------------------------------+
| ID | Operator                                                                             |
+----+--------------------------------------------------------------------------------------+
| *0 | Distributed Union on Singers <Row> (split_ranges_aligned)                            |
|  1 | +- Serialize Result <Row>                                                            |
|  2 |    +- Stream Aggregate <Row>                                                         |
|  3 |       +- Local Distributed Union <Row>                                               |
|  4 |          +- Filter Scan <Row> (seekable_key_size: 1)                                 |
| *5 |             +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (scan_method: Row) |
+----+--------------------------------------------------------------------------------------+
Predicates(identified by ID):
 0: Split Range: ($SingerId < 100)
 5: Seek Condition: ($SingerId < 100)


=== execution-plans/join ===
SELECT al.AlbumTitle, so.SongName FROM Albums AS al, Songs AS so WHERE al.SingerId = so.SingerId AND al.AlbumId = so.AlbumId
+-----+-------------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                              |
+-----+-------------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on Albums <Row>                                                                     |
|   1 | +- Serialize Result <Row>                                                                             |
|   2 |    +- Cross Apply <Row>                                                                               |
|   3 |       +- [Input] Local Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                      |
|   4 |       |  +- Local Distributed Union <Row>                                                             |
|   5 |       |     +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
|  10 |       +- [Map] Local Distributed Union <Row>                                                          |
|  11 |          +- Filter Scan <Row> (seekable_key_size: 0)                                                  |
| *12 |             +- Table Scan on Albums <Row> (scan_method: Row)                                          |
+-----+-------------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 12: Seek Condition: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))


=== execution-plans/index-with-back-join ===
SELECT s.SongName, s.Duration FROM Songs@{FORCE_INDEX=SongsBySongName} AS s WHERE STARTS_WITH(s.SongName, "B")
+-----+--------------------------------------------------------------------------+
| ID  | Operator                                                                 |
+-----+--------------------------------------------------------------------------+
|  *0 | Distributed Union on SongsBySongName <Row>                               |
|  *1 | +- Distributed Cross Apply <Row>                                         |
|   2 |    +- [Input] Create Batch <Batch>                                       |
|   3 |    |  +- RowToDataBlock                                                  |
|   4 |    |     +- Local Distributed Union <Row>                                |
|   5 |    |        +- Filter Scan <Row> (seekable_key_size: 1)                  |
|  *6 |    |           +- Index Scan on SongsBySongName <Row> (scan_method: Row) |
|  19 |    +- [Map] Serialize Result <Row>                                       |
|  20 |       +- Cross Apply <Row>                                               |
|  21 |          +- [Input] KeyRangeAccumulator <Row>                            |
|  22 |          |  +- DataBlockToRow                                            |
|  23 |          |     +- Batch Scan on $v2 <Batch> (scan_method: Batch)         |
|  32 |          +- [Map] Local Distributed Union <Row>                          |
|  33 |             +- Filter Scan <Row> (seekable_key_size: 0)                  |
| *34 |                +- Table Scan on Songs <Row> (scan_method: Row)           |
+-----+--------------------------------------------------------------------------+
Predicates(identified by ID):
  0: Split Range: STARTS_WITH($SongName, 'B')
  1: Split Range: (($Songs_key_SingerId' = $Songs_key_SingerId) AND ($Songs_key_AlbumId' = $Songs_key_AlbumId) AND ($Songs_key_TrackId' = $Songs_key_TrackId))
  6: Seek Condition: STARTS_WITH($SongName, 'B')
 34: Seek Condition: (($Songs_key_SingerId' = $batched_Songs_key_SingerId') AND ($Songs_key_AlbumId' = $batched_Songs_key_AlbumId') AND ($Songs_key_TrackId' = $batched_Songs_key_TrackId'))


=== execution-plans/index-only ===
SELECT s.SongName FROM Songs@{FORCE_INDEX=SongsBySongName} AS s WHERE STARTS_WITH(s.SongName, "B")
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
 0: Split Range: STARTS_WITH($SongName, 'B')
 4: Seek Condition: STARTS_WITH($SongName, 'B')


=== leaf/array-unnest ===
SELECT a, b FROM UNNEST([1,2,3]) a WITH OFFSET b
+----+------------------------+
| ID | Operator               |
+----+------------------------+
|  0 | Serialize Result <Row> |
|  1 | +- Array Unnest <Row>  |
+----+------------------------+


=== leaf/generate-relation ===
SELECT 1 + 2 AS Result
+----+------------------------+
| ID | Operator               |
+----+------------------------+
|  0 | Serialize Result <Row> |
|  1 | +- Unit Relation <Row> |
+----+------------------------+


=== leaf/empty-relation ===
SELECT * FROM Albums LIMIT 0
+----+-------------------------+
| ID | Operator                |
+----+-------------------------+
|  0 | Serialize Result <Row>  |
|  1 | +- Empty Relation <Row> |
+----+-------------------------+


=== leaf/filter-scan-index ===
SELECT s.LastName FROM Singers@{FORCE_INDEX=SingersByFirstLastName} AS s WHERE s.FirstName = 'Catalina'
+----+---------------------------------------------------------------------------------+
| ID | Operator                                                                        |
+----+---------------------------------------------------------------------------------+
| *0 | Distributed Union on SingersByFirstLastName <Row>                               |
|  1 | +- Local Distributed Union <Row>                                                |
|  2 |    +- Serialize Result <Row>                                                    |
|  3 |       +- Filter Scan <Row> (seekable_key_size: 0)                               |
| *4 |          +- Index Scan on SingersByFirstLastName <Row> (scan_method: Automatic) |
+----+---------------------------------------------------------------------------------+
Predicates(identified by ID):
 0: Split Range: ($FirstName = 'Catalina')
 4: Seek Condition: IS_NOT_DISTINCT_FROM($FirstName, 'Catalina')


=== leaf/filter-scan-table ===
SELECT LastName FROM Singers WHERE SingerId = 1
+----+------------------------------------------------------------+
| ID | Operator                                                   |
+----+------------------------------------------------------------+
| *0 | Distributed Union on Singers <Row>                         |
|  1 | +- Local Distributed Union <Row>                           |
|  2 |    +- Serialize Result <Row>                               |
|  3 |       +- Filter Scan <Row> (seekable_key_size: 0)          |
| *4 |          +- Table Scan on Singers <Row> (scan_method: Row) |
+----+------------------------------------------------------------+
Predicates(identified by ID):
 0: Split Range: ($SingerId = 1)
 4: Seek Condition: ($SingerId = 1)


=== unary/aggregate ===
SELECT s.SingerId, AVG(s.Duration) AS average, COUNT(*) AS count FROM Songs AS s GROUP BY SingerId
+----+---------------------------------------------------------------------------+
| ID | Operator                                                                  |
+----+---------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                 |
|  1 | +- Serialize Result <Row>                                                 |
|  2 |    +- Stream Aggregate <Row>                                              |
|  3 |       +- Local Distributed Union <Row>                                    |
|  4 |          +- Table Scan on Songs <Row> (Full scan, scan_method: Automatic) |
+----+---------------------------------------------------------------------------+


=== unary/compute-struct ===
SELECT FirstName, ARRAY(SELECT AS STRUCT song.SongName, song.SongGenre FROM Songs AS song WHERE song.SingerId = singer.SingerId) FROM Singers AS singer WHERE singer.SingerId = 1
+-----+-------------------------------------------------------------------+
| ID  | Operator                                                          |
+-----+-------------------------------------------------------------------+
|  *0 | Distributed Union on Singers <Row> (split_ranges_aligned)         |
|   1 | +- Local Distributed Union <Row>                                  |
|   2 |    +- Serialize Result <Row>                                      |
|   3 |       +- Filter Scan <Row> (seekable_key_size: 0)                 |
|  *4 |       |  +- Table Scan on Singers <Row> (scan_method: Row)        |
|  12 |       +- [Scalar] Array Subquery                                  |
|  13 |          +- Local Distributed Union <Row>                         |
|  14 |             +- Compute Struct <Row>                               |
|  15 |                +- Filter Scan <Row> (seekable_key_size: 0)        |
| *16 |                   +- Table Scan on Songs <Row> (scan_method: Row) |
+-----+-------------------------------------------------------------------+
Predicates(identified by ID):
  0: Split Range: ($SingerId = 1)
  4: Seek Condition: ($SingerId = 1)
 16: Seek Condition: ($SingerId_1 = 1)


=== unary/filter ===
SELECT s.LastName FROM (SELECT s.LastName FROM Singers AS s LIMIT 3) s WHERE s.LastName LIKE 'Rich%'
+----+--------------------------------------------------------------------------------------------+
| ID | Operator                                                                                   |
+----+--------------------------------------------------------------------------------------------+
|  0 | Serialize Result <Row>                                                                     |
| *1 | +- Filter <Row>                                                                            |
|  2 |    +- Global Limit <Row>                                                                   |
|  3 |       +- Distributed Union on SingersByFirstLastName <Row>                                 |
|  4 |          +- Local Limit <Row>                                                              |
|  5 |             +- Local Distributed Union <Row>                                               |
|  6 |                +- Index Scan on SingersByFirstLastName <Row> (Full scan, scan_method: Row) |
+----+--------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 1: Condition: STARTS_WITH($LastName, 'Rich')


=== unary/limit ===
SELECT s.SongName FROM Songs AS s LIMIT 3
+----+-------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                        |
+----+-------------------------------------------------------------------------------------------------+
|  0 | Global Limit <Row>                                                                              |
|  1 | +- Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                                    |
|  2 |    +- Serialize Result <Row>                                                                    |
|  3 |       +- Local Limit <Row>                                                                      |
|  4 |          +- Local Distributed Union <Row>                                                       |
|  5 |             +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Row) |
+----+-------------------------------------------------------------------------------------------------+


=== unary/tablesample-bernoulli ===
SELECT s.SongName FROM Songs AS s TABLESAMPLE BERNOULLI (10 PERCENT)
+----+-------------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                              |
+----+-------------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                                             |
|  1 | +- Serialize Result <Row>                                                                             |
| *2 |    +- Filter <Row>                                                                                    |
|  3 |       +- Random Id Assign <Row>                                                                       |
|  4 |          +- Local Distributed Union <Row>                                                             |
|  5 |             +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+----+-------------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($v1 < 900719925474099U)


=== unary/tablesample-reservoir ===
SELECT s.SongName FROM Songs AS s TABLESAMPLE RESERVOIR (2 ROWS)
+----+----------------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                                 |
+----+----------------------------------------------------------------------------------------------------------+
|  0 | Global Limit <Row>                                                                                       |
|  1 | +- Distributed Union on SongsBySingerAlbumSongNameDesc <Row> (preserve_subquery_order: true)             |
|  2 |    +- Serialize Result <Row>                                                                             |
|  3 |       +- Local Sort Limit <Row>                                                                          |
|  4 |          +- Random Id Assign <Row>                                                                       |
|  5 |             +- Local Distributed Union <Row>                                                             |
|  6 |                +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+----+----------------------------------------------------------------------------------------------------------+


=== unary/row-to-datablock ===
SELECT BirthDate FROM Singers
+----+--------------------------------------------------------------------------+
| ID | Operator                                                                 |
+----+--------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row>                                       |
|  1 | +- Local Distributed Union <Row>                                         |
|  2 |    +- Serialize Result <Row>                                             |
|  3 |       +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
+----+--------------------------------------------------------------------------+


=== unary/serialize-result-array ===
SELECT ARRAY(SELECT AS STRUCT so.SongName, so.SongGenre FROM Songs AS so WHERE so.SingerId = s.SingerId) FROM Singers AS s
+----+--------------------------------------------------------------------------+
| ID | Operator                                                                 |
+----+--------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                |
|  1 | +- Local Distributed Union <Row>                                         |
|  2 |    +- Serialize Result <Row>                                             |
|  3 |       +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  5 |       +- [Scalar] Array Subquery                                         |
|  6 |          +- Local Distributed Union <Row>                                |
|  7 |             +- Compute Struct <Row>                                      |
|  8 |                +- Filter Scan <Row> (seekable_key_size: 0)               |
| *9 |                   +- Table Scan on Songs <Row> (scan_method: Row)        |
+----+--------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== unary/sort ===
SELECT s.SongGenre FROM Songs AS s ORDER BY SongGenre
+----+---------------------------------------------------------------------------+
| ID | Operator                                                                  |
+----+---------------------------------------------------------------------------+
|  0 | Distributed Union on Songs <Row> (preserve_subquery_order: true)          |
|  1 | +- Serialize Result <Row>                                                 |
|  2 |    +- Sort <Row>                                                          |
|  3 |       +- Local Distributed Union <Row>                                    |
|  4 |          +- Table Scan on Songs <Row> (Full scan, scan_method: Automatic) |
+----+---------------------------------------------------------------------------+


=== unary/sort-limit ===
SELECT s.SongGenre FROM Songs AS s ORDER BY SongGenre LIMIT 3
+----+------------------------------------------------------------------------------+
| ID | Operator                                                                     |
+----+------------------------------------------------------------------------------+
|  0 | Global Limit <Row>                                                           |
|  1 | +- Distributed Union on Songs <Row> (preserve_subquery_order: true)          |
|  2 |    +- Serialize Result <Row>                                                 |
|  3 |       +- Local Sort Limit <Row>                                              |
|  4 |          +- Local Distributed Union <Row>                                    |
|  5 |             +- Table Scan on Songs <Row> (Full scan, scan_method: Automatic) |
+----+------------------------------------------------------------------------------+


=== unary/minor-sort-order-by-partial-key ===
SELECT SingerId, AlbumTitle FROM Albums ORDER BY SingerId, AlbumTitle
+----+----------------------------------------------------------------------------+
| ID | Operator                                                                   |
+----+----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                  |
|  1 | +- Serialize Result <Row>                                                  |
|  2 |    +- Minor Sort <Row>                                                     |
|  3 |       +- Local Distributed Union <Row>                                     |
|  4 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic) |
+----+----------------------------------------------------------------------------+


=== unary/minor-sort-limit-order-by-partial-key ===
SELECT SingerId, AlbumTitle FROM Albums WHERE SingerId > 0 ORDER BY SingerId, AlbumTitle LIMIT 3
+----+-----------------------------------------------------------------+
| ID | Operator                                                        |
+----+-----------------------------------------------------------------+
|  0 | Global Limit <Row>                                              |
| *1 | +- Distributed Union on Singers <Row> (split_ranges_aligned)    |
|  2 |    +- Serialize Result <Row>                                    |
|  3 |       +- Local Minor Sort Limit <Row>                           |
|  4 |          +- Local Distributed Union <Row>                       |
|  5 |             +- Filter Scan <Row> (seekable_key_size: 1)         |
| *6 |                +- Table Scan on Albums <Row> (scan_method: Row) |
+----+-----------------------------------------------------------------+
Predicates(identified by ID):
 1: Split Range: ($SingerId > 0)
 6: Seek Condition: ($SingerId > 0)


=== unary/minor-sort-stream-aggregate ===
SELECT SingerId, SongGenre FROM Songs GROUP@{GROUP_METHOD=STREAM_GROUP} BY SingerId, SongGenre
+----+------------------------------------------------------------------------------+
| ID | Operator                                                                     |
+----+------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                    |
|  1 | +- Serialize Result <Row>                                                    |
|  2 |    +- Stream Aggregate <Row>                                                 |
|  3 |       +- Minor Sort <Row>                                                    |
|  4 |          +- Local Distributed Union <Row>                                    |
|  5 |             +- Table Scan on Songs <Row> (Full scan, scan_method: Automatic) |
+----+------------------------------------------------------------------------------+


```
