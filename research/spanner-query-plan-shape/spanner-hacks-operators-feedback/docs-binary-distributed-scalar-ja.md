# spanner-hacks operators evidence: binary, distributed, scalar, and CTE operators

Generated from `go run ./tools/spanner-query-plan-shape --case docs --output reference --continue-on-error` on 2026-05-06 with Spanner Omni.

```text
=== binary/cross-apply ===
SELECT si.FirstName, (SELECT so.SongName FROM Songs AS so WHERE so.SingerId = si.SingerId LIMIT 1) FROM Singers AS si
+-----+-----------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                      |
+-----+-----------------------------------------------------------------------------------------------+
|   0 | Distributed Union on Singers <Row> (split_ranges_aligned)                                     |
|   1 | +- Local Distributed Union <Row>                                                              |
|   2 |    +- Serialize Result <Row>                                                                  |
|   3 |       +- Cross Apply <Row>                                                                    |
|   4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic)           |
|   7 |          +- [Map] Stream Aggregate <Row> (scalar_aggregate: true)                             |
|   8 |             +- Global Limit <Row>                                                             |
|   9 |                +- Local Distributed Union <Row>                                               |
|  10 |                   +- Filter Scan <Row> (seekable_key_size: 0)                                 |
| *11 |                      +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (scan_method: Row) |
+-----+-----------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 11: Seek Condition: ($SingerId_1 = $SingerId)


=== binary/in-subquery ===
SELECT FirstName, LastName FROM Singers WHERE SingerId IN (SELECT SingerId FROM Albums)
+-----+-------------------------------------------------------------------------------+
| ID  | Operator                                                                      |
+-----+-------------------------------------------------------------------------------+
|   0 | Distributed Union on Singers <Row> (split_ranges_aligned)                     |
|   1 | +- Serialize Result <Row>                                                     |
|   2 |    +- Cross Apply <Row>                                                       |
|   3 |       +- [Input] Stream Aggregate <Row>                                       |
|   4 |       |  +- Local Distributed Union <Row>                                     |
|   5 |       |     +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic) |
|   8 |       +- [Map] Local Distributed Union <Row>                                  |
|   9 |          +- Filter Scan <Row> (seekable_key_size: 0)                          |
| *10 |             +- Table Scan on Singers <Row> (scan_method: Row)                 |
+-----+-------------------------------------------------------------------------------+
Predicates(identified by ID):
 10: Seek Condition: ($SingerId = $group_SingerId_1)


=== binary/not-in-subquery ===
SELECT FirstName, LastName FROM Singers WHERE SingerId NOT IN (SELECT SingerId FROM Albums)
+-----+-------------------------------------------------------------------------------------+
| ID  | Operator                                                                            |
+-----+-------------------------------------------------------------------------------------+
|   0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|   1 | +- Local Distributed Union <Row>                                                    |
|   2 |    +- Serialize Result <Row>                                                        |
|   3 |       +- Anti-Semi Apply <Row>                                                      |
|   4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|   8 |          +- [Map] Local Distributed Union <Row>                                     |
|   9 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *10 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+-----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 10: Seek Condition: ($SingerId_1 = $SingerId)


=== binary/hash-join ===
SELECT a.AlbumTitle, s.SongName FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+----+----------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                           |
+----+----------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                           |
|  1 | +- Serialize Result <Row>                                                                          |
| *2 |    +- Hash Join <Row> (join_type: INNER)                                                           |
|  3 |       +- [Build] Local Distributed Union <Row>                                                     |
|  4 |       |  +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                         |
|  8 |       +- [Probe] Local Distributed Union <Row>                                                     |
|  9 |          +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+----+----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))


=== binary/merge-join ===
SELECT a.AlbumTitle, s.SongName FROM Albums AS a JOIN@{JOIN_METHOD=MERGE_JOIN} Songs AS s ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+----+----------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                           |
+----+----------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                           |
|  1 | +- Serialize Result <Row>                                                                          |
| *2 |    +- Merge Join <Row> (join_configuration: ONE_TO_MANY, join_type: INNER)                         |
|  3 |       +- [Left] Local Distributed Union <Row>                                                      |
|  4 |       |  +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                         |
|  8 |       +- [Right] Local Distributed Union <Row>                                                     |
|  9 |          +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+----+----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))


=== binary/merge-join-with-sort ===
SELECT a.AlbumTitle, s.SongName FROM Albums AS a JOIN@{JOIN_METHOD=MERGE_JOIN} Songs AS s ON a.AlbumId = s.AlbumId
+----+---------------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                                |
+----+---------------------------------------------------------------------------------------------------------+
|  0 | Serialize Result <Row>                                                                                  |
| *1 | +- Merge Join <Row> (join_configuration: MANY_TO_MANY, join_type: INNER)                                |
|  2 |    +- [Left] Distributed Union on AlbumsByAlbumTitle <Row> (preserve_subquery_order: true)              |
|  3 |    |  +- Sort <Row>                                                                                     |
|  4 |    |     +- Local Distributed Union <Row>                                                               |
|  5 |    |        +- Index Scan on AlbumsByAlbumTitle <Row> (Full scan, scan_method: Automatic)               |
| 12 |    +- [Right] Distributed Union on SongsBySingerAlbumSongNameDesc <Row> (preserve_subquery_order: true) |
| 13 |       +- Sort <Row>                                                                                     |
| 14 |          +- Local Distributed Union <Row>                                                               |
| 15 |             +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic)   |
+----+---------------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  1: Condition: ($sort_AlbumId = $sort_AlbumId_1)


=== binary/recursive-union-graph ===
GRAPH MusicGraph MATCH (singer:Singers {singerId:42})-[c:CollabWith]->{1,2}(featured:Singers) RETURN singer.SingerId AS singer, featured.SingerId AS featured
+-----+-------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                  |
+-----+-------------------------------------------------------------------------------------------+
|  *0 | Distributed Union on Singers <Row>                                                        |
|   1 | +- Serialize Result <Row>                                                                 |
|   2 |    +- DataBlockToRow                                                                      |
|   3 |       +- Recursive Union <Batch>                                                          |
|   4 |          +- Union Input                                                                   |
|   5 |          |  +- RowToDataBlock                                                             |
|   6 |          |     +- Local Distributed Union <Row>                                           |
|   7 |          |        +- Compute <Row>                                                        |
|   8 |          |           +- Filter Scan <Row> (seekable_key_size: 0)                          |
|  *9 |          |              +- Table Scan on Singers <Row> (scan_method: Row)                 |
|  18 |          +- Union Input                                                                   |
| *19 |             +- Distributed Cross Apply <Batch>                                            |
|  20 |                +- [Input] Create Batch <Batch>                                            |
| *21 |                |  +- Distributed Cross Apply <Batch>                                      |
|  22 |                |     +- [Input] Create Batch <Batch>                                      |
|  23 |                |     |  +- Recursive Spool Scan <Batch>                                   |
|  28 |                |     +- [Map] RowToDataBlock                                              |
|  29 |                |        +- Cross Apply <Row>                                              |
|  30 |                |           +- [Input] KeyRangeAccumulator <Row>                           |
|  31 |                |           |  +- DataBlockToRow                                           |
|  32 |                |           |     +- Batch Scan on $v26 <Batch> (scan_method: Batch)       |
|  37 |                |           +- [Map] Local Distributed Union <Row>                         |
|  38 |                |              +- Filter Scan <Row> (seekable_key_size: 0)                 |
| *39 |                |                 +- Table Scan on Collaborations <Row> (scan_method: Row) |
|  55 |                +- [Map] RowToDataBlock                                                    |
|  56 |                   +- Cross Apply <Row>                                                    |
|  57 |                      +- [Input] KeyRangeAccumulator <Row>                                 |
|  58 |                      |  +- DataBlockToRow                                                 |
|  59 |                      |     +- Batch Scan on $v28 <Batch> (scan_method: Batch)             |
|  64 |                      +- [Map] Local Distributed Union <Row>                               |
|  65 |                         +- Filter Scan <Row> (seekable_key_size: 0)                       |
| *66 |                            +- Table Scan on Singers <Row> (scan_method: Row)              |
+-----+-------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  0: Split Range: ($SingerId'5 = 42)
  9: Seek Condition: ($SingerId'5 = 42)
 19: Split Range: ($SingerId_4'5 = $FeaturingSingerId'6)
 21: Split Range: ($SingerId_3'5 = $tail'_SingerId'16)
 39: Seek Condition: ($SingerId_3'5 = $batched_tail'_SingerId'17)
 66: Seek Condition: ($SingerId_4'5 = $batched_FeaturingSingerId'7)


=== n-ary/union-all ===
SELECT 1 a, 2 b UNION ALL SELECT 3 a, 4 b UNION ALL SELECT 5 a, 6 b
+----+---------------------------------+
| ID | Operator                        |
+----+---------------------------------+
|  0 | Serialize Result <Row>          |
|  1 | +- Union All <Row>              |
|  2 |    +- Union Input               |
|  3 |    |  +- Compute <Row>          |
|  4 |    |     +- Unit Relation <Row> |
| 10 |    +- Union Input               |
| 11 |    |  +- Compute <Row>          |
| 12 |    |     +- Unit Relation <Row> |
| 18 |    +- Union Input               |
| 19 |       +- Compute <Row>          |
| 20 |          +- Unit Relation <Row> |
+----+---------------------------------+


=== n-ary/union-all-different-names ===
SELECT 1 a, 2 b UNION ALL SELECT 3 c, 4 e
+----+---------------------------------+
| ID | Operator                        |
+----+---------------------------------+
|  0 | Serialize Result <Row>          |
|  1 | +- Union All <Row>              |
|  2 |    +- Union Input               |
|  3 |    |  +- Compute <Row>          |
|  4 |    |     +- Unit Relation <Row> |
| 10 |    +- Union Input               |
| 11 |       +- Compute <Row>          |
| 12 |          +- Unit Relation <Row> |
+----+---------------------------------+


=== distributed/distributed-union ===
SELECT s.SongName, s.SongGenre FROM Songs AS s WHERE s.SingerId = 2 AND s.SongGenre = 'ROCK'
+----+----------------------------------------------------------------+
| ID | Operator                                                       |
+----+----------------------------------------------------------------+
| *0 | Distributed Union on Songs <Row>                               |
|  1 | +- Local Distributed Union <Row>                               |
|  2 |    +- Serialize Result <Row>                                   |
| *3 |       +- Filter Scan <Row> (seekable_key_size: 0)              |
| *4 |          +- Table Scan on Songs <Row> (scan_method: Automatic) |
+----+----------------------------------------------------------------+
Predicates(identified by ID):
 0: Split Range: ($SingerId = 2)
 3: Residual Condition: ($SongGenre = 'ROCK')
 4: Seek Condition: ($SingerId = 2)


=== distributed/distributed-apply ===
SELECT AlbumTitle FROM Songs JOIN Albums ON Albums.AlbumId = Songs.AlbumId
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


=== distributed/distributed-merge-union ===
SELECT LastName, ConcertDate FROM Singers LEFT OUTER JOIN@{JOIN_TYPE=APPLY_JOIN} Concerts ON Singers.SingerId = Concerts.SingerId
+-----+--------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                         |
+-----+--------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SingersByFirstLastName <Row>                                                |
|   1 | +- Serialize Result <Row>                                                                        |
|  *2 |    +- Distributed Outer Apply <Row>                                                              |
|   3 |       +- [Input] Create Batch <Row>                                                              |
|   4 |       |  +- Local Distributed Union <Row>                                                        |
|   5 |       |     +- Compute Struct <Row>                                                              |
|   6 |       |        +- Index Scan on SingersByFirstLastName <Row> (Full scan, scan_method: Automatic) |
|  13 |       +- [Map] Cross Apply <Row>                                                                 |
|  14 |          +- [Input] Batch Scan on $v2 <Row> (scan_method: Row)                                   |
|  18 |          +- [Map] Local Distributed Union <Row>                                                  |
| *19 |             +- Filter Scan <Row> (seekable_key_size: 0)                                          |
|  20 |                +- Table Scan on Concerts <Row> (Full scan, scan_method: Row)                     |
+-----+--------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($SingerId_1 = $SingerId)
 19: Residual Condition: ($SingerId_1 = $batched_SingerId)


=== distributed/push-broadcast-hash-join ===
SELECT a.AlbumTitle, s.SongName FROM Albums AS a JOIN@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} Songs AS s ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+-------------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                              |
+-----+-------------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on AlbumsByAlbumTitle <Row>                                                         |
|  *1 | +- Push Broadcast Hash Join <Row>                                                                     |
|   2 |    +- Create Batch <Batch>                                                                            |
|   3 |    |  +- RowToDataBlock                                                                               |
|   4 |    |     +- Local Distributed Union <Row>                                                             |
|   5 |    |        +- Index Scan on AlbumsByAlbumTitle <Row> (Full scan, scan_method: Automatic)             |
|  12 |    +- [Map] Serialize Result <Row>                                                                    |
| *13 |       +- Hash Join <Row> (join_type: INNER)                                                           |
|  14 |          +- [Build] DataBlockToRow                                                                    |
|  15 |          |  +- Batch Scan on $v2 <Batch> (scan_method: Batch)                                         |
|  22 |          +- [Probe] Local Distributed Union <Row>                                                     |
|  23 |             +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+-----+-------------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  1: Split Range: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))
 13: Condition: (($batched_SingerId' = $SingerId_1) AND ($batched_AlbumId' = $AlbumId_1))


=== scalar-subquery/conditional ===
SELECT FirstName, IF(FirstName = 'Alice', (SELECT COUNT(*) FROM Songs WHERE Duration > 300), 0) FROM Singers
+-----+------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                 |
+-----+------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SingersByFirstLastName <Row>                                        |
|   1 | +- Local Distributed Union <Row>                                                         |
|   2 |    +- Serialize Result <Row>                                                             |
|   3 |       +- Index Scan on SingersByFirstLastName <Row> (Full scan, scan_method: Automatic)  |
|  10 |       +- [Scalar] Scalar Subquery                                                        |
|  11 |          +- Global Stream Aggregate <Row> (scalar_aggregate: true)                       |
|  12 |             +- Distributed Union on Songs <Row>                                          |
|  13 |                +- Local Stream Aggregate <Row> (scalar_aggregate: true)                  |
|  14 |                   +- Local Distributed Union <Row>                                       |
| *15 |                      +- Filter Scan <Row> (seekable_key_size: 0)                         |
|  16 |                         +- Table Scan on Songs <Row> (Full scan, scan_method: Automatic) |
+-----+------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 15: Residual Condition: ($Duration > 300)


=== scalar-subquery/filter ===
SELECT * FROM Songs WHERE Duration = (SELECT MAX(Duration) FROM Songs)
+-----+---------------------------------------------------------------------------------+
| ID  | Operator                                                                        |
+-----+---------------------------------------------------------------------------------+
|   0 | Serialize Result <Row>                                                          |
|   1 | +- Cross Apply <Row>                                                            |
|   2 |    +- [Input] Global Stream Aggregate <Row> (scalar_aggregate: true)            |
|   3 |    |  +- Distributed Union on Songs <Row>                                       |
|   4 |    |     +- Local Stream Aggregate <Row> (scalar_aggregate: true)               |
|   5 |    |        +- Local Distributed Union <Row>                                    |
|   6 |    |           +- Table Scan on Songs <Row> (Full scan, scan_method: Automatic) |
|  13 |    +- [Map] Distributed Union on Songs <Row>                                    |
|  14 |       +- Local Distributed Union <Row>                                          |
| *15 |          +- Filter Scan <Row> (seekable_key_size: 0)                            |
|  16 |             +- Table Scan on Songs <Row> (Full scan, scan_method: Row)          |
+-----+---------------------------------------------------------------------------------+
Predicates(identified by ID):
 15: Residual Condition: ($Duration = $agg1)


=== array-subquery ===
SELECT a.AlbumId, ARRAY(SELECT ConcertDate FROM Concerts WHERE Concerts.SingerId = a.SingerId) FROM Albums AS a
+-----+-------------------------------------------------------------------------------------+
| ID  | Operator                                                                            |
+-----+-------------------------------------------------------------------------------------+
|   0 | Distributed Union on AlbumsByAlbumTitle <Row>                                       |
|   1 | +- Local Distributed Union <Row>                                                    |
|   2 |    +- Serialize Result <Row>                                                        |
|   3 |       +- Index Scan on AlbumsByAlbumTitle <Row> (Full scan, scan_method: Automatic) |
|   7 |       +- [Scalar] Array Subquery                                                    |
|  *8 |          +- Distributed Union on Concerts <Row>                                     |
|   9 |             +- Local Distributed Union <Row>                                        |
| *10 |                +- Filter Scan <Row> (seekable_key_size: 0)                          |
|  11 |                   +- Table Scan on Concerts <Row> (Full scan, scan_method: Row)     |
+-----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
  8: Split Range: ($SingerId_1 = $SingerId)
 10: Residual Condition: ($SingerId_1 = $SingerId)


=== struct-constructor ===
SELECT IF(TRUE, STRUCT(1 AS A, 1 AS B), STRUCT(2 AS A, 2 AS B)).A
+----+------------------------+
| ID | Operator               |
+----+------------------------+
|  0 | Serialize Result <Row> |
|  1 | +- Unit Relation <Row> |
+----+------------------------+


=== cte/spool-build ===
WITH CTE AS (SELECT 1 AS PK, "foo" AS col) SELECT * FROM CTE c1 JOIN CTE c2 USING (PK)
+-----+----------------------------------------------------+
| ID  | Operator                                           |
+-----+----------------------------------------------------+
|   0 | Serialize Result <Row>                             |
|   1 | +- Cross Apply <Row>                               |
|   2 |    +- [Input] SpoolBuild <Row> (spool_name: CTE)   |
|   3 |    |  +- Compute <Row>                             |
|   4 |    |     +- Unit Relation <Row>                    |
|  10 |    +- [Map] Cross Apply <Row>                      |
|  11 |       +- [Input] SpoolScan <Row> (spool_name: CTE) |
| *14 |       +- [Map] Filter <Row>                        |
|  15 |          +- SpoolScan <Row> (spool_name: CTE)      |
+-----+----------------------------------------------------+
Predicates(identified by ID):
 14: Condition: ($PK_2 = $PK_1)


```
