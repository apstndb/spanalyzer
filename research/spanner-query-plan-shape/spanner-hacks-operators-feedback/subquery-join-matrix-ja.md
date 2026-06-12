# spanner-hacks operators evidence: subquery predicate join hint matrix

Generated from `go run ./tools/spanner-query-plan-shape --case subquery_join_hint_matrix --output reference --continue-on-error` on 2026-05-06 with Spanner Omni.

```text
=== subquery-join-hint-matrix/in/force_join_order_true ===
@{FORCE_JOIN_ORDER=TRUE}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId IN (SELECT a.SingerId FROM Albums AS a)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Semi Apply <Row>                                                           |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/in/join_method_hash_join ===
@{JOIN_METHOD=HASH_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId IN (SELECT a.SingerId FROM Albums AS a)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_SEMI)                               |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId = $SingerId_1)


=== subquery-join-hint-matrix/in/join_method_apply_join ===
@{JOIN_METHOD=APPLY_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId IN (SELECT a.SingerId FROM Albums AS a)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Semi Apply <Row>                                                           |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/in/join_method_merge_join ===
@{JOIN_METHOD=MERGE_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId IN (SELECT a.SingerId FROM Albums AS a)
+----+--------------------------------------------------------------------------------+
| ID | Operator                                                                       |
+----+--------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                      |
|  1 | +- Serialize Result <Row>                                                      |
| *2 |    +- Merge Join <Row> (join_configuration: ONE_TO_MANY, join_type: LEFT_SEMI) |
|  3 |       +- [Left] Local Distributed Union <Row>                                  |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic)    |
|  7 |       +- [Right] Local Distributed Union <Row>                                 |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)     |
+----+--------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId = $SingerId_1)


=== subquery-join-hint-matrix/in/join_method_push_broadcast_hash_join ===
@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId IN (SELECT a.SingerId FROM Albums AS a)
+-----+--------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                         |
+-----+--------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SingersByFirstLastName <Row>                                                |
|   1 | +- Serialize Result <Row>                                                                        |
|  *2 |    +- Push Broadcast Hash Join Semi Apply <Row>                                                  |
|   3 |       +- [Input] Create Batch <Row>                                                              |
|   4 |       |  +- Local Distributed Union <Row>                                                        |
|   5 |       |     +- Compute Struct <Row>                                                              |
|   6 |       |        +- Index Scan on SingersByFirstLastName <Row> (Full scan, scan_method: Automatic) |
| *13 |       +- [Map] Hash Join <Row> (join_type: INNER)                                                |
|  14 |          +- [Build] Batch Scan on $v2 <Row> (scan_method: Row)                                   |
|  18 |          +- [Probe] Local Distributed Union <Row>                                                |
|  19 |             +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                    |
+-----+--------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($SingerId_1 = $SingerId)
 13: Condition: ($batched_SingerId = $SingerId_1)


=== subquery-join-hint-matrix/in/hash_join_build_left ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId IN (SELECT a.SingerId FROM Albums AS a)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_SEMI)                               |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId = $SingerId_1)


=== subquery-join-hint-matrix/in/hash_join_build_right ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId IN (SELECT a.SingerId FROM Albums AS a)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: PROBE_SEMI)                               |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
|  6 |       +- [Probe] Local Distributed Union <Row>                              |
|  7 |          +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId = $SingerId_1)


=== subquery-join-hint-matrix/in/hash_join_execution_multi_pass ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=MULTI_PASS}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId IN (SELECT a.SingerId FROM Albums AS a)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_SEMI)                               |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId = $SingerId_1)


=== subquery-join-hint-matrix/in/hash_join_execution_one_pass ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId IN (SELECT a.SingerId FROM Albums AS a)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_SEMI)                               |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId = $SingerId_1)


=== subquery-join-hint-matrix/in/apply_join_batch_true ===
@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId IN (SELECT a.SingerId FROM Albums AS a)
+-----+--------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                         |
+-----+--------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SingersByFirstLastName <Row>                                                |
|   1 | +- Serialize Result <Row>                                                                        |
|  *2 |    +- Distributed Semi Apply <Row>                                                               |
|   3 |       +- [Input] Create Batch <Row>                                                              |
|   4 |       |  +- Local Distributed Union <Row>                                                        |
|   5 |       |     +- Compute Struct <Row>                                                              |
|   6 |       |        +- Index Scan on SingersByFirstLastName <Row> (Full scan, scan_method: Automatic) |
|  13 |       +- [Map] Semi Apply <Row>                                                                  |
|  14 |          +- [Input] KeyRangeAccumulator <Row>                                                    |
|  15 |          |  +- Batch Scan on $v2 <Row> (scan_method: Row)                                        |
|  19 |          +- [Map] Local Distributed Union <Row>                                                  |
|  20 |             +- Filter Scan <Row> (seekable_key_size: 0)                                          |
| *21 |                +- Table Scan on Albums <Row> (scan_method: Row)                                  |
+-----+--------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($SingerId_1 = $SingerId)
 21: Seek Condition: ($SingerId_1 = $batched_SingerId)


=== subquery-join-hint-matrix/in/apply_join_batch_false ===
@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId IN (SELECT a.SingerId FROM Albums AS a)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Semi Apply <Row>                                                           |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/in/legacy_join_type_apply_join ===
@{JOIN_TYPE=APPLY_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId IN (SELECT a.SingerId FROM Albums AS a)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Semi Apply <Row>                                                           |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/exists/force_join_order_true ===
@{FORCE_JOIN_ORDER=TRUE}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Semi Apply <Row>                                                           |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/exists/join_method_hash_join ===
@{JOIN_METHOD=HASH_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_SEMI)                               |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/exists/join_method_apply_join ===
@{JOIN_METHOD=APPLY_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Semi Apply <Row>                                                           |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/exists/join_method_merge_join ===
@{JOIN_METHOD=MERGE_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+--------------------------------------------------------------------------------+
| ID | Operator                                                                       |
+----+--------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                      |
|  1 | +- Serialize Result <Row>                                                      |
| *2 |    +- Merge Join <Row> (join_configuration: ONE_TO_MANY, join_type: LEFT_SEMI) |
|  3 |       +- [Left] Local Distributed Union <Row>                                  |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic)    |
|  7 |       +- [Right] Local Distributed Union <Row>                                 |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)     |
+----+--------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/exists/join_method_push_broadcast_hash_join ===
@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+-----+--------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                         |
+-----+--------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SingersByFirstLastName <Row>                                                |
|   1 | +- Serialize Result <Row>                                                                        |
|  *2 |    +- Push Broadcast Hash Join Semi Apply <Row>                                                  |
|   3 |       +- [Input] Create Batch <Row>                                                              |
|   4 |       |  +- Local Distributed Union <Row>                                                        |
|   5 |       |     +- Compute Struct <Row>                                                              |
|   6 |       |        +- Index Scan on SingersByFirstLastName <Row> (Full scan, scan_method: Automatic) |
| *13 |       +- [Map] Hash Join <Row> (join_type: INNER)                                                |
|  14 |          +- [Build] Batch Scan on $v2 <Row> (scan_method: Row)                                   |
|  18 |          +- [Probe] Local Distributed Union <Row>                                                |
|  19 |             +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                    |
+-----+--------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($SingerId_1 = $SingerId)
 13: Condition: ($SingerId_1 = $batched_SingerId)


=== subquery-join-hint-matrix/exists/hash_join_build_left ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_SEMI)                               |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/exists/hash_join_build_right ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: PROBE_SEMI)                               |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
|  6 |       +- [Probe] Local Distributed Union <Row>                              |
|  7 |          +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/exists/hash_join_execution_multi_pass ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=MULTI_PASS}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_SEMI)                               |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/exists/hash_join_execution_one_pass ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_SEMI)                               |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/exists/apply_join_batch_true ===
@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+-----+--------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                         |
+-----+--------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SingersByFirstLastName <Row>                                                |
|   1 | +- Serialize Result <Row>                                                                        |
|  *2 |    +- Distributed Semi Apply <Row>                                                               |
|   3 |       +- [Input] Create Batch <Row>                                                              |
|   4 |       |  +- Local Distributed Union <Row>                                                        |
|   5 |       |     +- Compute Struct <Row>                                                              |
|   6 |       |        +- Index Scan on SingersByFirstLastName <Row> (Full scan, scan_method: Automatic) |
|  13 |       +- [Map] Semi Apply <Row>                                                                  |
|  14 |          +- [Input] KeyRangeAccumulator <Row>                                                    |
|  15 |          |  +- Batch Scan on $v2 <Row> (scan_method: Row)                                        |
|  19 |          +- [Map] Local Distributed Union <Row>                                                  |
|  20 |             +- Filter Scan <Row> (seekable_key_size: 0)                                          |
| *21 |                +- Table Scan on Albums <Row> (scan_method: Row)                                  |
+-----+--------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($SingerId_1 = $SingerId)
 21: Seek Condition: ($SingerId_1 = $batched_SingerId)


=== subquery-join-hint-matrix/exists/apply_join_batch_false ===
@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Semi Apply <Row>                                                           |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/exists/legacy_join_type_apply_join ===
@{JOIN_TYPE=APPLY_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Semi Apply <Row>                                                           |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_in/force_join_order_true ===
@{FORCE_JOIN_ORDER=TRUE}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Anti-Semi Apply <Row>                                                      |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_in/join_method_hash_join ===
@{JOIN_METHOD=HASH_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_ANTI_SEMI)                          |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId = $SingerId_1)


=== subquery-join-hint-matrix/not_in/join_method_apply_join ===
@{JOIN_METHOD=APPLY_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Anti-Semi Apply <Row>                                                      |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_in/join_method_merge_join ===
@{JOIN_METHOD=MERGE_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Serialize Result <Row>                                                           |
| *2 |    +- Merge Join <Row> (join_configuration: ONE_TO_MANY, join_type: LEFT_ANTI_SEMI) |
|  3 |       +- [Left] Local Distributed Union <Row>                                       |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic)         |
|  7 |       +- [Right] Local Distributed Union <Row>                                      |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)          |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId = $SingerId_1)


=== subquery-join-hint-matrix/not_in/join_method_push_broadcast_hash_join ===
@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)
+-----+--------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                         |
+-----+--------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SingersByFirstLastName <Row>                                                |
|   1 | +- Serialize Result <Row>                                                                        |
|  *2 |    +- Push Broadcast Hash Join Anti Semi Apply <Row>                                             |
|   3 |       +- [Input] Create Batch <Row>                                                              |
|   4 |       |  +- Local Distributed Union <Row>                                                        |
|   5 |       |     +- Compute Struct <Row>                                                              |
|   6 |       |        +- Index Scan on SingersByFirstLastName <Row> (Full scan, scan_method: Automatic) |
| *13 |       +- [Map] Hash Join <Row> (join_type: INNER)                                                |
|  14 |          +- [Build] Batch Scan on $v2 <Row> (scan_method: Row)                                   |
|  18 |          +- [Probe] Local Distributed Union <Row>                                                |
|  19 |             +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                    |
+-----+--------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($SingerId_1 = $SingerId)
 13: Condition: ($batched_SingerId = $SingerId_1)


=== subquery-join-hint-matrix/not_in/hash_join_build_left ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_ANTI_SEMI)                          |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId = $SingerId_1)


=== subquery-join-hint-matrix/not_in/hash_join_build_right ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: PROBE_ANTI_SEMI)                          |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
|  6 |       +- [Probe] Local Distributed Union <Row>                              |
|  7 |          +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId = $SingerId_1)


=== subquery-join-hint-matrix/not_in/hash_join_execution_multi_pass ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=MULTI_PASS}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_ANTI_SEMI)                          |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId = $SingerId_1)


=== subquery-join-hint-matrix/not_in/hash_join_execution_one_pass ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_ANTI_SEMI)                          |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId = $SingerId_1)


=== subquery-join-hint-matrix/not_in/apply_join_batch_true ===
@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)
+-----+--------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                         |
+-----+--------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SingersByFirstLastName <Row>                                                |
|   1 | +- Serialize Result <Row>                                                                        |
|  *2 |    +- Distributed Anti Semi Apply <Row>                                                          |
|   3 |       +- [Input] Create Batch <Row>                                                              |
|   4 |       |  +- Local Distributed Union <Row>                                                        |
|   5 |       |     +- Compute Struct <Row>                                                              |
|   6 |       |        +- Index Scan on SingersByFirstLastName <Row> (Full scan, scan_method: Automatic) |
|  13 |       +- [Map] Semi Apply <Row>                                                                  |
|  14 |          +- [Input] KeyRangeAccumulator <Row>                                                    |
|  15 |          |  +- Batch Scan on $v2 <Row> (scan_method: Row)                                        |
|  19 |          +- [Map] Local Distributed Union <Row>                                                  |
|  20 |             +- Filter Scan <Row> (seekable_key_size: 0)                                          |
| *21 |                +- Table Scan on Albums <Row> (scan_method: Row)                                  |
+-----+--------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($SingerId_1 = $SingerId)
 21: Seek Condition: ($SingerId_1 = $batched_SingerId)


=== subquery-join-hint-matrix/not_in/apply_join_batch_false ===
@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Anti-Semi Apply <Row>                                                      |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_in/legacy_join_type_apply_join ===
@{JOIN_TYPE=APPLY_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Anti-Semi Apply <Row>                                                      |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_exists/force_join_order_true ===
@{FORCE_JOIN_ORDER=TRUE}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Anti-Semi Apply <Row>                                                      |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_exists/join_method_hash_join ===
@{JOIN_METHOD=HASH_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_ANTI_SEMI)                          |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_exists/join_method_apply_join ===
@{JOIN_METHOD=APPLY_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Anti-Semi Apply <Row>                                                      |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_exists/join_method_merge_join ===
@{JOIN_METHOD=MERGE_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Serialize Result <Row>                                                           |
| *2 |    +- Merge Join <Row> (join_configuration: ONE_TO_MANY, join_type: LEFT_ANTI_SEMI) |
|  3 |       +- [Left] Local Distributed Union <Row>                                       |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic)         |
|  7 |       +- [Right] Local Distributed Union <Row>                                      |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)          |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_exists/join_method_push_broadcast_hash_join ===
@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+-----+--------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                         |
+-----+--------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SingersByFirstLastName <Row>                                                |
|   1 | +- Serialize Result <Row>                                                                        |
|  *2 |    +- Push Broadcast Hash Join Anti Semi Apply <Row>                                             |
|   3 |       +- [Input] Create Batch <Row>                                                              |
|   4 |       |  +- Local Distributed Union <Row>                                                        |
|   5 |       |     +- Compute Struct <Row>                                                              |
|   6 |       |        +- Index Scan on SingersByFirstLastName <Row> (Full scan, scan_method: Automatic) |
| *13 |       +- [Map] Hash Join <Row> (join_type: INNER)                                                |
|  14 |          +- [Build] Batch Scan on $v2 <Row> (scan_method: Row)                                   |
|  18 |          +- [Probe] Local Distributed Union <Row>                                                |
|  19 |             +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)                    |
+-----+--------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($SingerId_1 = $SingerId)
 13: Condition: ($SingerId_1 = $batched_SingerId)


=== subquery-join-hint-matrix/not_exists/hash_join_build_left ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_ANTI_SEMI)                          |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_exists/hash_join_build_right ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: PROBE_ANTI_SEMI)                          |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
|  6 |       +- [Probe] Local Distributed Union <Row>                              |
|  7 |          +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_exists/hash_join_execution_multi_pass ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=MULTI_PASS}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_ANTI_SEMI)                          |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_exists/hash_join_execution_one_pass ===
@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                   |
|  1 | +- Serialize Result <Row>                                                   |
| *2 |    +- Hash Join <Row> (join_type: BUILD_ANTI_SEMI)                          |
|  3 |       +- [Build] Local Distributed Union <Row>                              |
|  4 |       |  +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |       +- [Probe] Local Distributed Union <Row>                              |
|  8 |          +- Table Scan on Albums <Row> (Full scan, scan_method: Automatic)  |
+----+-----------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_exists/apply_join_batch_true ===
@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+-----+--------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                         |
+-----+--------------------------------------------------------------------------------------------------+
|   0 | Distributed Union on SingersByFirstLastName <Row>                                                |
|   1 | +- Serialize Result <Row>                                                                        |
|  *2 |    +- Distributed Anti Semi Apply <Row>                                                          |
|   3 |       +- [Input] Create Batch <Row>                                                              |
|   4 |       |  +- Local Distributed Union <Row>                                                        |
|   5 |       |     +- Compute Struct <Row>                                                              |
|   6 |       |        +- Index Scan on SingersByFirstLastName <Row> (Full scan, scan_method: Automatic) |
|  13 |       +- [Map] Semi Apply <Row>                                                                  |
|  14 |          +- [Input] KeyRangeAccumulator <Row>                                                    |
|  15 |          |  +- Batch Scan on $v2 <Row> (scan_method: Row)                                        |
|  19 |          +- [Map] Local Distributed Union <Row>                                                  |
|  20 |             +- Filter Scan <Row> (seekable_key_size: 0)                                          |
| *21 |                +- Table Scan on Albums <Row> (scan_method: Row)                                  |
+-----+--------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($SingerId_1 = $SingerId)
 21: Seek Condition: ($SingerId_1 = $batched_SingerId)


=== subquery-join-hint-matrix/not_exists/apply_join_batch_false ===
@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Anti-Semi Apply <Row>                                                      |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== subquery-join-hint-matrix/not_exists/legacy_join_type_apply_join ===
@{JOIN_TYPE=APPLY_JOIN}
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row> (split_ranges_aligned)                           |
|  1 | +- Local Distributed Union <Row>                                                    |
|  2 |    +- Serialize Result <Row>                                                        |
|  3 |       +- Anti-Semi Apply <Row>                                                      |
|  4 |          +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |          +- [Map] Local Distributed Union <Row>                                     |
|  8 |             +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+-------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)

```
