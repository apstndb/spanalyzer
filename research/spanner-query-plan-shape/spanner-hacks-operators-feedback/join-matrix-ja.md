# spanner-hacks operators evidence: join hint matrix

Generated from `go run ./tools/spanner-query-plan-shape --case join_matrix --output reference --continue-on-error` on 2026-05-06 with Spanner Omni.

```text
=== join-matrix/inner/force_join_order_true ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a INNER JOIN@{FORCE_JOIN_ORDER=TRUE} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+--------------------------------------------------------------------------------------+
| ID  | Operator                                                                             |
+-----+--------------------------------------------------------------------------------------+
|   0 | Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                            |
|   1 | +- Serialize Result <Row>                                                            |
|   2 |    +- Cross Apply <Row>                                                              |
|   3 |       +- [Input] Local Distributed Union on Albums <Row>                             |
|   4 |       |  +- Local Distributed Union <Row>                                            |
|   5 |       |     +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)        |
|  10 |       +- [Map] Local Distributed Union <Row>                                         |
|  11 |          +- Filter Scan <Row> (seekable_key_size: 0)                                 |
| *12 |             +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (scan_method: Row) |
+-----+--------------------------------------------------------------------------------------+
Predicates(identified by ID):
 12: Seek Condition: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))


=== join-matrix/inner/join_method_hash_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a INNER JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/inner/join_method_apply_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a INNER JOIN@{JOIN_METHOD=APPLY_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+--------------------------------------------------------------------------------------+
| ID  | Operator                                                                             |
+-----+--------------------------------------------------------------------------------------+
|   0 | Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                            |
|   1 | +- Serialize Result <Row>                                                            |
|   2 |    +- Cross Apply <Row>                                                              |
|   3 |       +- [Input] Local Distributed Union on Albums <Row>                             |
|   4 |       |  +- Local Distributed Union <Row>                                            |
|   5 |       |     +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)        |
|  10 |       +- [Map] Local Distributed Union <Row>                                         |
|  11 |          +- Filter Scan <Row> (seekable_key_size: 0)                                 |
| *12 |             +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (scan_method: Row) |
+-----+--------------------------------------------------------------------------------------+
Predicates(identified by ID):
 12: Seek Condition: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))


=== join-matrix/inner/join_method_merge_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a INNER JOIN@{JOIN_METHOD=MERGE_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/inner/join_method_push_broadcast_hash_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a INNER JOIN@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/inner/hash_join_build_left ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a INNER JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/inner/hash_join_build_right ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a INNER JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+----+----------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                           |
+----+----------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                           |
|  1 | +- Serialize Result <Row>                                                                          |
| *2 |    +- Hash Join <Row> (join_type: INNER)                                                           |
|  3 |       +- [Build] Local Distributed Union <Row>                                                     |
|  4 |       |  +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
|  8 |       +- [Probe] Local Distributed Union <Row>                                                     |
|  9 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                         |
+----+----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))


=== join-matrix/inner/hash_join_execution_multi_pass ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a INNER JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=MULTI_PASS} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/inner/hash_join_execution_one_pass ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a INNER JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/inner/apply_join_batch_true ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a INNER JOIN@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+-------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                  |
+-----+-------------------------------------------------------------------------------------------+
|   0 | Distributed Union on AlbumsByAlbumTitle <Row>                                             |
|  *1 | +- Distributed Cross Apply <Row>                                                          |
|   2 |    +- [Input] Create Batch <Batch>                                                        |
|   3 |    |  +- RowToDataBlock                                                                   |
|   4 |    |     +- Local Distributed Union <Row>                                                 |
|   5 |    |        +- Index Scan on AlbumsByAlbumTitle <Row> (Full scan, scan_method: Automatic) |
|  12 |    +- [Map] Serialize Result <Row>                                                        |
|  13 |       +- Cross Apply <Row>                                                                |
|  14 |          +- [Input] KeyRangeAccumulator <Row>                                             |
|  15 |          |  +- DataBlockToRow                                                             |
|  16 |          |     +- Batch Scan on $v2 <Batch> (scan_method: Batch)                          |
|  23 |          +- [Map] Local Distributed Union <Row>                                           |
|  24 |             +- Filter Scan <Row> (seekable_key_size: 0)                                   |
| *25 |                +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (scan_method: Row)   |
+-----+-------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  1: Split Range: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))
 25: Seek Condition: (($SingerId_1 = $batched_SingerId') AND ($AlbumId_1 = $batched_AlbumId'))


=== join-matrix/inner/apply_join_batch_false ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a INNER JOIN@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+--------------------------------------------------------------------------------------+
| ID  | Operator                                                                             |
+-----+--------------------------------------------------------------------------------------+
|   0 | Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                            |
|   1 | +- Serialize Result <Row>                                                            |
|   2 |    +- Cross Apply <Row>                                                              |
|   3 |       +- [Input] Local Distributed Union on Albums <Row>                             |
|   4 |       |  +- Local Distributed Union <Row>                                            |
|   5 |       |     +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)        |
|  10 |       +- [Map] Local Distributed Union <Row>                                         |
|  11 |          +- Filter Scan <Row> (seekable_key_size: 0)                                 |
| *12 |             +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (scan_method: Row) |
+-----+--------------------------------------------------------------------------------------+
Predicates(identified by ID):
 12: Seek Condition: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))


=== join-matrix/inner/legacy_join_type_apply_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a INNER JOIN@{JOIN_TYPE=APPLY_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+--------------------------------------------------------------------------------------+
| ID  | Operator                                                                             |
+-----+--------------------------------------------------------------------------------------+
|   0 | Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                            |
|   1 | +- Serialize Result <Row>                                                            |
|   2 |    +- Cross Apply <Row>                                                              |
|   3 |       +- [Input] Local Distributed Union on Albums <Row>                             |
|   4 |       |  +- Local Distributed Union <Row>                                            |
|   5 |       |     +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)        |
|  10 |       +- [Map] Local Distributed Union <Row>                                         |
|  11 |          +- Filter Scan <Row> (seekable_key_size: 0)                                 |
| *12 |             +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (scan_method: Row) |
+-----+--------------------------------------------------------------------------------------+
Predicates(identified by ID):
 12: Seek Condition: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))


=== join-matrix/left/force_join_order_true ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a LEFT JOIN@{FORCE_JOIN_ORDER=TRUE} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+-----------------------------------------------------------------------------------------+
| ID  | Operator                                                                                |
+-----+-----------------------------------------------------------------------------------------+
|   0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                |
|   1 | +- Local Distributed Union <Row>                                                        |
|   2 |    +- Serialize Result <Row>                                                            |
|   3 |       +- Outer Apply <Row>                                                              |
|   4 |          +- [Input] Table Scan on Albums <Row> (Full scan, scan_method: Automatic)      |
|   8 |          +- [Map] Local Distributed Union <Row>                                         |
|   9 |             +- Filter Scan <Row> (seekable_key_size: 0)                                 |
| *10 |                +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (scan_method: Row) |
+-----+-----------------------------------------------------------------------------------------+
Predicates(identified by ID):
 10: Seek Condition: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))


=== join-matrix/left/join_method_hash_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a LEFT JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+----+----------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                           |
+----+----------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                           |
|  1 | +- Serialize Result <Row>                                                                          |
| *2 |    +- Hash Join <Row> (join_type: BUILD_OUTER)                                                     |
|  3 |       +- [Build] Local Distributed Union <Row>                                                     |
|  4 |       |  +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                         |
|  8 |       +- [Probe] Local Distributed Union <Row>                                                     |
|  9 |          +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+----+----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))


=== join-matrix/left/join_method_apply_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a LEFT JOIN@{JOIN_METHOD=APPLY_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+-----------------------------------------------------------------------------------------+
| ID  | Operator                                                                                |
+-----+-----------------------------------------------------------------------------------------+
|   0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                |
|   1 | +- Local Distributed Union <Row>                                                        |
|   2 |    +- Serialize Result <Row>                                                            |
|   3 |       +- Outer Apply <Row>                                                              |
|   4 |          +- [Input] Table Scan on Albums <Row> (Full scan, scan_method: Automatic)      |
|   8 |          +- [Map] Local Distributed Union <Row>                                         |
|   9 |             +- Filter Scan <Row> (seekable_key_size: 0)                                 |
| *10 |                +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (scan_method: Row) |
+-----+-----------------------------------------------------------------------------------------+
Predicates(identified by ID):
 10: Seek Condition: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))


=== join-matrix/left/join_method_merge_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a LEFT JOIN@{JOIN_METHOD=MERGE_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+----+----------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                           |
+----+----------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                           |
|  1 | +- Serialize Result <Row>                                                                          |
| *2 |    +- Merge Join <Row> (join_configuration: ONE_TO_MANY, join_type: LEFT_OUTER)                    |
|  3 |       +- [Left] Local Distributed Union <Row>                                                      |
|  4 |       |  +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                         |
|  8 |       +- [Right] Local Distributed Union <Row>                                                     |
|  9 |          +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+----+----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))


=== join-matrix/left/join_method_push_broadcast_hash_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a LEFT JOIN@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+-------------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                              |
+-----+-------------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on AlbumsByAlbumTitle <Row>                                                         |
|   1 | +- Serialize Result <Row>                                                                             |
|  *2 |    +- Push Broadcast Hash Join Outer Apply <Row>                                                      |
|   3 |       +- [Input] Create Batch <Row>                                                                   |
|   4 |       |  +- Local Distributed Union <Row>                                                             |
|   5 |       |     +- Compute Struct <Row>                                                                   |
|   6 |       |        +- Index Scan on AlbumsByAlbumTitle <Row> (Full scan, scan_method: Automatic)          |
| *15 |       +- [Map] Hash Join <Row> (join_type: INNER)                                                     |
|  16 |          +- [Build] Batch Scan on $v2 <Row> (scan_method: Row)                                        |
|  21 |          +- [Probe] Local Distributed Union <Row>                                                     |
|  22 |             +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+-----+-------------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))
 15: Condition: (($batched_SingerId = $SingerId_1) AND ($batched_AlbumId = $AlbumId_1))


=== join-matrix/left/hash_join_build_left ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a LEFT JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+----+----------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                           |
+----+----------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                           |
|  1 | +- Serialize Result <Row>                                                                          |
| *2 |    +- Hash Join <Row> (join_type: BUILD_OUTER)                                                     |
|  3 |       +- [Build] Local Distributed Union <Row>                                                     |
|  4 |       |  +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                         |
|  8 |       +- [Probe] Local Distributed Union <Row>                                                     |
|  9 |          +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+----+----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))


=== join-matrix/left/hash_join_build_right ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a LEFT JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+----+----------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                           |
+----+----------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                           |
|  1 | +- Serialize Result <Row>                                                                          |
| *2 |    +- Hash Join <Row> (join_type: PROBE_OUTER)                                                     |
|  3 |       +- [Build] Local Distributed Union <Row>                                                     |
|  4 |       |  +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
|  8 |       +- [Probe] Local Distributed Union <Row>                                                     |
|  9 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                         |
+----+----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))


=== join-matrix/left/hash_join_execution_multi_pass ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a LEFT JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=MULTI_PASS} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+----+----------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                           |
+----+----------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                           |
|  1 | +- Serialize Result <Row>                                                                          |
| *2 |    +- Hash Join <Row> (join_type: BUILD_OUTER)                                                     |
|  3 |       +- [Build] Local Distributed Union <Row>                                                     |
|  4 |       |  +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                         |
|  8 |       +- [Probe] Local Distributed Union <Row>                                                     |
|  9 |          +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+----+----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))


=== join-matrix/left/hash_join_execution_one_pass ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a LEFT JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+----+----------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                           |
+----+----------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                           |
|  1 | +- Serialize Result <Row>                                                                          |
| *2 |    +- Hash Join <Row> (join_type: BUILD_OUTER)                                                     |
|  3 |       +- [Build] Local Distributed Union <Row>                                                     |
|  4 |       |  +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                         |
|  8 |       +- [Probe] Local Distributed Union <Row>                                                     |
|  9 |          +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
+----+----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))


=== join-matrix/left/apply_join_batch_true ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a LEFT JOIN@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+----------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                     |
+-----+----------------------------------------------------------------------------------------------+
|   0 | Distributed Union on AlbumsByAlbumTitle <Row>                                                |
|   1 | +- Serialize Result <Row>                                                                    |
|  *2 |    +- Distributed Outer Apply <Row>                                                          |
|   3 |       +- [Input] Create Batch <Row>                                                          |
|   4 |       |  +- Local Distributed Union <Row>                                                    |
|   5 |       |     +- Compute Struct <Row>                                                          |
|   6 |       |        +- Index Scan on AlbumsByAlbumTitle <Row> (Full scan, scan_method: Automatic) |
|  15 |       +- [Map] Cross Apply <Row>                                                             |
|  16 |          +- [Input] KeyRangeAccumulator <Row>                                                |
|  17 |          |  +- Batch Scan on $v2 <Row> (scan_method: Row)                                    |
|  22 |          +- [Map] Local Distributed Union <Row>                                              |
|  23 |             +- Filter Scan <Row> (seekable_key_size: 0)                                      |
| *24 |                +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (scan_method: Row)      |
+-----+----------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))
 24: Seek Condition: (($SingerId_1 = $batched_SingerId) AND ($AlbumId_1 = $batched_AlbumId))


=== join-matrix/left/apply_join_batch_false ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a LEFT JOIN@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+-----------------------------------------------------------------------------------------+
| ID  | Operator                                                                                |
+-----+-----------------------------------------------------------------------------------------+
|   0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                |
|   1 | +- Local Distributed Union <Row>                                                        |
|   2 |    +- Serialize Result <Row>                                                            |
|   3 |       +- Outer Apply <Row>                                                              |
|   4 |          +- [Input] Table Scan on Albums <Row> (Full scan, scan_method: Automatic)      |
|   8 |          +- [Map] Local Distributed Union <Row>                                         |
|   9 |             +- Filter Scan <Row> (seekable_key_size: 0)                                 |
| *10 |                +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (scan_method: Row) |
+-----+-----------------------------------------------------------------------------------------+
Predicates(identified by ID):
 10: Seek Condition: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))


=== join-matrix/left/legacy_join_type_apply_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a LEFT JOIN@{JOIN_TYPE=APPLY_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+-----------------------------------------------------------------------------------------+
| ID  | Operator                                                                                |
+-----+-----------------------------------------------------------------------------------------+
|   0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                |
|   1 | +- Local Distributed Union <Row>                                                        |
|   2 |    +- Serialize Result <Row>                                                            |
|   3 |       +- Outer Apply <Row>                                                              |
|   4 |          +- [Input] Table Scan on Albums <Row> (Full scan, scan_method: Automatic)      |
|   8 |          +- [Map] Local Distributed Union <Row>                                         |
|   9 |             +- Filter Scan <Row> (seekable_key_size: 0)                                 |
| *10 |                +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (scan_method: Row) |
+-----+-----------------------------------------------------------------------------------------+
Predicates(identified by ID):
 10: Seek Condition: (($SingerId_1 = $SingerId) AND ($AlbumId_1 = $AlbumId))


=== join-matrix/right/force_join_order_true ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a RIGHT JOIN@{FORCE_JOIN_ORDER=TRUE} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/right/join_method_hash_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a RIGHT JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/right/join_method_apply_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a RIGHT JOIN@{JOIN_METHOD=APPLY_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/right/join_method_merge_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a RIGHT JOIN@{JOIN_METHOD=MERGE_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+----+----------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                           |
+----+----------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                           |
|  1 | +- Serialize Result <Row>                                                                          |
| *2 |    +- Merge Join <Row> (join_configuration: MANY_TO_ONE, join_type: INNER)                         |
|  3 |       +- [Left] Local Distributed Union <Row>                                                      |
|  4 |       |  +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
|  8 |       +- [Right] Local Distributed Union <Row>                                                     |
|  9 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                         |
+----+----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))


=== join-matrix/right/join_method_push_broadcast_hash_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a RIGHT JOIN@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+-------------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                              |
+-----+-------------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                                             |
|  *1 | +- Push Broadcast Hash Join <Row>                                                                     |
|   2 |    +- Create Batch <Batch>                                                                            |
|   3 |    |  +- RowToDataBlock                                                                               |
|   4 |    |     +- Local Distributed Union <Row>                                                             |
|   5 |    |        +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
|  12 |    +- [Map] Serialize Result <Row>                                                                    |
| *13 |       +- Hash Join <Row> (join_type: INNER)                                                           |
|  14 |          +- [Build] Local Distributed Union <Row>                                                     |
|  15 |          |  +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                         |
|  19 |          +- [Probe] DataBlockToRow                                                                    |
|  20 |             +- Batch Scan on $v2 <Batch> (scan_method: Batch)                                         |
+-----+-------------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  1: Split Range: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))
 13: Condition: (($SingerId = $batched_SingerId_1') AND ($AlbumId = $batched_AlbumId_1'))


=== join-matrix/right/hash_join_build_left ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a RIGHT JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/right/hash_join_build_right ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a RIGHT JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+----+----------------------------------------------------------------------------------------------------+
| ID | Operator                                                                                           |
+----+----------------------------------------------------------------------------------------------------+
|  0 | Distributed Union on Albums <Row> (split_ranges_aligned)                                           |
|  1 | +- Serialize Result <Row>                                                                          |
| *2 |    +- Hash Join <Row> (join_type: INNER)                                                           |
|  3 |       +- [Build] Local Distributed Union <Row>                                                     |
|  4 |       |  +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
|  8 |       +- [Probe] Local Distributed Union <Row>                                                     |
|  9 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                         |
+----+----------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))


=== join-matrix/right/hash_join_execution_multi_pass ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a RIGHT JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=MULTI_PASS} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/right/hash_join_execution_one_pass ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a RIGHT JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/right/apply_join_batch_true ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a RIGHT JOIN@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
+-----+-------------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                              |
+-----+-------------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SongsBySingerAlbumSongNameDesc <Row>                                             |
|  *1 | +- Distributed Cross Apply <Row>                                                                      |
|   2 |    +- [Input] Create Batch <Batch>                                                                    |
|   3 |    |  +- RowToDataBlock                                                                               |
|   4 |    |     +- Local Distributed Union <Row>                                                             |
|   5 |    |        +- Index Scan on SongsBySingerAlbumSongNameDesc <Row> (Full scan, scan_method: Automatic) |
|  12 |    +- [Map] Serialize Result <Row>                                                                    |
|  13 |       +- Cross Apply <Row>                                                                            |
|  14 |          +- [Input] KeyRangeAccumulator <Row>                                                         |
|  15 |          |  +- DataBlockToRow                                                                         |
|  16 |          |     +- Batch Scan on $v2 <Batch> (scan_method: Batch)                                      |
|  23 |          +- [Map] Local Distributed Union <Row>                                                       |
|  24 |             +- Filter Scan <Row> (seekable_key_size: 0)                                               |
| *25 |                +- Table Scan on Albums <Row> (scan_method: Row)                                       |
+-----+-------------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  1: Split Range: (($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))
 25: Seek Condition: (($SingerId = $batched_SingerId_1') AND ($AlbumId = $batched_AlbumId_1'))


=== join-matrix/right/apply_join_batch_false ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a RIGHT JOIN@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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


=== join-matrix/right/legacy_join_type_apply_join ===
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a RIGHT JOIN@{JOIN_TYPE=APPLY_JOIN} Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
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

```
