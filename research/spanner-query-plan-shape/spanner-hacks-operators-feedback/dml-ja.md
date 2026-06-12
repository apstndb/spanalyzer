# spanner-hacks operators evidence: DML

Generated from `go run ./tools/spanner-query-plan-shape --case dml --output reference --continue-on-error` on 2026-05-06 with Spanner Omni.

```text
=== dml/insert-values ===
INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (1, 'Marc', 'Richards')
+----+-----------------------------------------------------------+
| ID | Operator                                                  |
+----+-----------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: INSERT) |
|  1 | +- Serialize Result <Row>                                 |
|  2 |    +- Union All <Row>                                     |
|  3 |       +- Union Input                                      |
|  4 |          +- Compute <Row>                                 |
|  5 |             +- Unit Relation <Row>                        |
+----+-----------------------------------------------------------+


=== dml/insert-default ===
INSERT INTO Singers (SingerId, Status) VALUES (2, DEFAULT)
+----+-----------------------------------------------------------+
| ID | Operator                                                  |
+----+-----------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: INSERT) |
|  1 | +- Serialize Result <Row>                                 |
|  2 |    +- Union All <Row>                                     |
|  3 |       +- Union Input                                      |
|  4 |          +- Compute <Row>                                 |
|  5 |             +- Unit Relation <Row>                        |
+----+-----------------------------------------------------------+


=== dml/insert-select ===
INSERT INTO Singers (SingerId, FirstName, LastName) SELECT SingerId, FirstName, LastName FROM AckworthSingers
+----+-------------------------------------------------------------------------------------+
| ID | Operator                                                                            |
+----+-------------------------------------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: INSERT)                           |
|  1 | +- Serialize Result <Row>                                                           |
|  2 |    +- Distributed Union on AckworthSingers <Row>                                    |
|  3 |       +- Local Distributed Union <Row>                                              |
|  4 |          +- Table Scan on AckworthSingers <Row> (Full scan, scan_method: Automatic) |
+----+-------------------------------------------------------------------------------------+


=== dml/insert-select-unnest ===
INSERT INTO Singers (SingerId, FirstName, LastName) SELECT * FROM UNNEST([(4, 'Lea', 'Martin'), (5, 'David', 'Lomond')])
+----+-----------------------------------------------------------+
| ID | Operator                                                  |
+----+-----------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: INSERT) |
|  1 | +- Serialize Result <Row>                                 |
|  2 |    +- Array Unnest <Row>                                  |
+----+-----------------------------------------------------------+


=== dml/insert-subquery ===
INSERT INTO Singers (SingerId, FirstName) VALUES (6, (SELECT FirstName FROM AckworthSingers WHERE SingerId = 6))
+-----+--------------------------------------------------------------------------------------+
| ID  | Operator                                                                             |
+-----+--------------------------------------------------------------------------------------+
|   0 | Apply Mutations on Singers <Row> (operation_type: INSERT)                            |
|   1 | +- Serialize Result <Row>                                                            |
|   2 |    +- Union All <Row>                                                                |
|   3 |       +- Union Input                                                                 |
|   4 |          +- Compute <Row>                                                            |
|   5 |             +- Global Stream Aggregate <Row> (scalar_aggregate: true)                |
|  *6 |                +- Distributed Union on AckworthSingers <Row>                         |
|   7 |                   +- Local Stream Aggregate <Row> (scalar_aggregate: true)           |
|   8 |                      +- Local Distributed Union <Row>                                |
|   9 |                         +- Filter Scan <Row> (seekable_key_size: 0)                  |
| *10 |                            +- Table Scan on AckworthSingers <Row> (scan_method: Row) |
+-----+--------------------------------------------------------------------------------------+
Predicates(identified by ID):
  6: Split Range: ($SingerId = 6)
 10: Seek Condition: ($SingerId = 6)


=== dml/insert-assert-rows-modified ===
INSERT INTO Singers (SingerId, FirstName) VALUES (7, 'Asserted') ASSERT_ROWS_MODIFIED 1
error: spanner: code = "Unimplemented", desc = "ASSERT_ROWS_MODIFIED is not supported.", requestID = "1.d01cd9602e6a1911.1.3.17.1"

=== dml/insert-ignore ===
INSERT IGNORE INTO Singers (SingerId, FirstName) VALUES (8, 'Ignored')
+-----+------------------------------------------------------------------+
| ID  | Operator                                                         |
+-----+------------------------------------------------------------------+
|   0 | Apply Mutations on Singers <Row> (operation_type: INSERT)        |
|   1 | +- Serialize Result <Row>                                        |
|  *2 |    +- Distributed Anti Semi Apply <Row>                          |
|   3 |       +- [Input] Create Batch <Row>                              |
|   4 |       |  +- Compute Struct <Row>                                 |
|   5 |       |     +- Union All <Row>                                   |
|   6 |       |        +- Union Input                                    |
|   7 |       |           +- Compute <Row>                               |
|   8 |       |              +- Unit Relation <Row>                      |
|  20 |       +- [Map] Semi Apply <Row>                                  |
|  21 |          +- [Input] KeyRangeAccumulator <Row>                    |
|  22 |          |  +- Batch Scan on $v7 <Row> (scan_method: Row)        |
|  26 |          +- [Map] Local Distributed Union <Row>                  |
|  27 |             +- Filter Scan <Row> (seekable_key_size: 0)          |
| *28 |                +- Table Scan on Singers <Row> (scan_method: Row) |
+-----+------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($scan_SingerId = $v1)
 28: Seek Condition: ($scan_SingerId = $batched_v1)


=== dml/insert-or-ignore ===
INSERT OR IGNORE INTO Singers (SingerId, FirstName) VALUES (9, 'Ignored')
+-----+------------------------------------------------------------------+
| ID  | Operator                                                         |
+-----+------------------------------------------------------------------+
|   0 | Apply Mutations on Singers <Row> (operation_type: INSERT)        |
|   1 | +- Serialize Result <Row>                                        |
|  *2 |    +- Distributed Anti Semi Apply <Row>                          |
|   3 |       +- [Input] Create Batch <Row>                              |
|   4 |       |  +- Compute Struct <Row>                                 |
|   5 |       |     +- Union All <Row>                                   |
|   6 |       |        +- Union Input                                    |
|   7 |       |           +- Compute <Row>                               |
|   8 |       |              +- Unit Relation <Row>                      |
|  20 |       +- [Map] Semi Apply <Row>                                  |
|  21 |          +- [Input] KeyRangeAccumulator <Row>                    |
|  22 |          |  +- Batch Scan on $v7 <Row> (scan_method: Row)        |
|  26 |          +- [Map] Local Distributed Union <Row>                  |
|  27 |             +- Filter Scan <Row> (seekable_key_size: 0)          |
| *28 |                +- Table Scan on Singers <Row> (scan_method: Row) |
+-----+------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($scan_SingerId = $v1)
 28: Seek Condition: ($scan_SingerId = $batched_v1)


=== dml/insert-or-update ===
INSERT OR UPDATE INTO Singers (SingerId, Status) VALUES (10, 'inactive')
+-----+---------------------------------------------------------------------+
| ID  | Operator                                                            |
+-----+---------------------------------------------------------------------+
|   0 | Apply Mutations on Singers <Row> (operation_type: INSERT)           |
|   1 | +- Serialize Result <Row>                                           |
|  *2 |    +- Distributed Outer Apply <Row>                                 |
|   3 |       +- [Input] Create Batch <Row>                                 |
|   4 |       |  +- Compute Struct <Row>                                    |
|   5 |       |     +- Union All <Row>                                      |
|   6 |       |        +- Union Input                                       |
|   7 |       |           +- Compute <Row>                                  |
|   8 |       |              +- Unit Relation <Row>                         |
|  20 |       +- [Map] Compute <Row>                                        |
|  21 |          +- Cross Apply <Row>                                       |
|  22 |             +- [Input] KeyRangeAccumulator <Row>                    |
|  23 |             |  +- Batch Scan on $v7 <Row> (scan_method: Row)        |
|  27 |             +- [Map] Local Distributed Union <Row>                  |
|  28 |                +- Filter Scan <Row> (seekable_key_size: 0)          |
| *29 |                   +- Table Scan on Singers <Row> (scan_method: Row) |
+-----+---------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($scan_SingerId = $v3)
 29: Seek Condition: ($scan_SingerId = $batched_v3)


=== dml/insert-or-update-then-return-with-action ===
INSERT OR UPDATE Singers (SingerId, FirstName, LastName) VALUES (11, 'Melissa', 'Gartner') THEN RETURN WITH ACTION SingerId, FirstName || ' ' || LastName AS FullName
+-----+---------------------------------------------------------------------+
| ID  | Operator                                                            |
+-----+---------------------------------------------------------------------+
|   0 | Apply Mutations on Singers <Row> (operation_type: INSERT)           |
|   1 | +- Serialize Result <Row>                                           |
|  *2 |    +- Distributed Outer Apply <Row>                                 |
|   3 |       +- [Input] Create Batch <Row>                                 |
|   4 |       |  +- Compute Struct <Row>                                    |
|   5 |       |     +- Union All <Row>                                      |
|   6 |       |        +- Union Input                                       |
|   7 |       |           +- Compute <Row>                                  |
|   8 |       |              +- Unit Relation <Row>                         |
|  24 |       +- [Map] Compute <Row>                                        |
|  25 |          +- Cross Apply <Row>                                       |
|  26 |             +- [Input] KeyRangeAccumulator <Row>                    |
|  27 |             |  +- Batch Scan on $v9 <Row> (scan_method: Row)        |
|  32 |             +- [Map] Local Distributed Union <Row>                  |
|  33 |                +- Filter Scan <Row> (seekable_key_size: 0)          |
| *34 |                   +- Table Scan on Singers <Row> (scan_method: Row) |
+-----+---------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($scan_SingerId = $v4)
 34: Seek Condition: ($scan_SingerId = $batched_v4)


=== dml/insert-on-conflict-do-nothing ===
INSERT INTO Singers (SingerId, FirstName) VALUES (12, 'John') ON CONFLICT DO NOTHING
+-----+------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                 |
+-----+------------------------------------------------------------------------------------------+
|   0 | Apply Mutations on Singers <Row> (operation_type: INSERT)                                |
|   1 | +- Serialize Result <Row>                                                                |
|   2 |    +- Anti-Semi Apply <Row>                                                              |
|   3 |       +- [Input] Compute Struct <Row>                                                    |
|   4 |       |  +- Union All <Row>                                                              |
|   5 |       |     +- Union Input                                                               |
|   6 |       |        +- Compute <Row>                                                          |
|   7 |       |           +- Unit Relation <Row>                                                 |
|  20 |       +- [Map] Hash Aggregate <Row>                                                      |
|  21 |          +- Union All <Row>                                                              |
|  22 |             +- Union Input                                                               |
| *23 |             |  +- Distributed Union on Singers <Row>                                     |
|  24 |             |     +- Local Distributed Union <Row>                                       |
|  25 |             |        +- Filter Scan <Row> (seekable_key_size: 0)                         |
| *26 |             |           +- Table Scan on Singers <Row> (scan_method: Row)                |
|  36 |             +- Union Input                                                               |
| *37 |                +- Distributed Union on SingersByFirstLastName <Row>                      |
|  38 |                   +- Local Distributed Union <Row>                                       |
|  39 |                      +- Filter Scan <Row> (seekable_key_size: 0)                         |
| *40 |                         +- Index Scan on SingersByFirstLastName <Row> (scan_method: Row) |
+-----+------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 23: Split Range: ($scan_SingerId' = $v1)
 26: Seek Condition: ($scan_SingerId' = $v1)
 37: Split Range: (IS_NOT_DISTINCT_FROM($scan_LastName'2, <typed null>) AND IS_NOT_DISTINCT_FROM($scan_FirstName'2, $v2))
 40: Seek Condition: (IS_NOT_DISTINCT_FROM($scan_FirstName'2, $v2) AND IS_NOT_DISTINCT_FROM($scan_LastName'2, <typed null>))


=== dml/insert-on-conflict-target-do-nothing ===
INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (13, 'John', 'Smith') ON CONFLICT(SingerId) DO NOTHING
+-----+------------------------------------------------------------------+
| ID  | Operator                                                         |
+-----+------------------------------------------------------------------+
|   0 | Apply Mutations on Singers <Row> (operation_type: INSERT)        |
|   1 | +- Serialize Result <Row>                                        |
|  *2 |    +- Distributed Anti Semi Apply <Row>                          |
|   3 |       +- [Input] Create Batch <Row>                              |
|   4 |       |  +- Compute Struct <Row>                                 |
|   5 |       |     +- Compute Struct <Row>                              |
|   6 |       |        +- Union All <Row>                                |
|   7 |       |           +- Union Input                                 |
|   8 |       |              +- Compute <Row>                            |
|   9 |       |                 +- Unit Relation <Row>                   |
|  30 |       +- [Map] Semi Apply <Row>                                  |
|  31 |          +- [Input] KeyRangeAccumulator <Row>                    |
|  32 |          |  +- Batch Scan on $v11 <Row> (scan_method: Row)       |
|  36 |          +- [Map] Local Distributed Union <Row>                  |
|  37 |             +- Filter Scan <Row> (seekable_key_size: 0)          |
| *38 |                +- Table Scan on Singers <Row> (scan_method: Row) |
+-----+------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($scan_SingerId = $v1)
 38: Seek Condition: ($scan_SingerId = $batched_v1)


=== dml/insert-on-conflict-unique-do-nothing ===
INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (14, 'Jane', 'Smith') ON CONFLICT ON UNIQUE CONSTRAINT UniqueIndex_SingerName DO NOTHING
+-----+---------------------------------------------------------------------------------+
| ID  | Operator                                                                        |
+-----+---------------------------------------------------------------------------------+
|   0 | Apply Mutations on Singers <Row> (operation_type: INSERT)                       |
|   1 | +- Serialize Result <Row>                                                       |
|  *2 |    +- Distributed Anti Semi Apply <Row>                                         |
|   3 |       +- [Input] Create Batch <Row>                                             |
|   4 |       |  +- Compute Struct <Row>                                                |
|   5 |       |     +- Compute Struct <Row>                                             |
|   6 |       |        +- Union All <Row>                                               |
|   7 |       |           +- Union Input                                                |
|   8 |       |              +- Compute <Row>                                           |
|   9 |       |                 +- Unit Relation <Row>                                  |
|  31 |       +- [Map] Semi Apply <Row>                                                 |
|  32 |          +- [Input] KeyRangeAccumulator <Row>                                   |
|  33 |          |  +- Batch Scan on $v11 <Row> (scan_method: Row)                      |
|  38 |          +- [Map] Local Distributed Union <Row>                                 |
|  39 |             +- Filter Scan <Row> (seekable_key_size: 0)                         |
| *40 |                +- Index Scan on SingersByFirstLastName <Row> (scan_method: Row) |
+-----+---------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: (IS_NOT_DISTINCT_FROM($scan_FirstName, $v2) AND IS_NOT_DISTINCT_FROM($scan_LastName, $v3))
 40: Seek Condition: (IS_NOT_DISTINCT_FROM($scan_FirstName, $batched_v2) AND IS_NOT_DISTINCT_FROM($scan_LastName, $batched_v3))


=== dml/insert-on-conflict-do-update ===
INSERT INTO Singers (SingerId, FirstName, LastName, Status) VALUES (15, 'Adele', 'Adkins', 'active') ON CONFLICT(SingerId) DO UPDATE SET Status = EXCLUDED.Status
+-----+------------------------------------------------------------------------+
| ID  | Operator                                                               |
+-----+------------------------------------------------------------------------+
|   0 | Apply Mutations on Singers <Row> (operation_type: INSERT)              |
|   1 | +- Serialize Result <Row>                                              |
|   2 |    +- Compute Struct <Row>                                             |
|  *3 |       +- Distributed Outer Apply <Row>                                 |
|   4 |          +- [Input] Create Batch <Row>                                 |
|   5 |          |  +- Compute Struct <Row>                                    |
|   6 |          |     +- Union All <Row>                                      |
|   7 |          |        +- Union Input                                       |
|   8 |          |           +- Compute <Row>                                  |
|   9 |          |              +- Unit Relation <Row>                         |
|  29 |          +- [Map] Compute <Row>                                        |
|  30 |             +- Cross Apply <Row>                                       |
|  31 |                +- [Input] KeyRangeAccumulator <Row>                    |
|  32 |                |  +- Batch Scan on $v13 <Row> (scan_method: Row)       |
|  38 |                +- [Map] Local Distributed Union <Row>                  |
|  39 |                   +- Filter Scan <Row> (seekable_key_size: 0)          |
| *40 |                      +- Table Scan on Singers <Row> (scan_method: Row) |
+-----+------------------------------------------------------------------------+
Predicates(identified by ID):
  3: Split Range: ($scan_SingerId = $v5)
 40: Seek Condition: ($scan_SingerId = $batched_v5)


=== dml/insert-on-conflict-do-update-where ===
INSERT INTO Singers (SingerId, FirstName, LastName, Status) VALUES (16, 'Adele', 'Adkins', 'active') ON CONFLICT ON UNIQUE CONSTRAINT UniqueIndex_SingerName DO UPDATE SET Status = EXCLUDED.Status WHERE Singers.Status IS DISTINCT FROM EXCLUDED.Status
+-----+------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                       |
+-----+------------------------------------------------------------------------------------------------+
|   0 | Apply Mutations on Singers <Row> (operation_type: INSERT)                                      |
|   1 | +- Serialize Result <Row>                                                                      |
|   2 |    +- Compute Struct <Row>                                                                     |
|  *3 |       +- Filter <Row>                                                                          |
|  *4 |          +- Distributed Outer Apply <Row>                                                      |
|   5 |             +- [Input] Create Batch <Row>                                                      |
|   6 |             |  +- Compute Struct <Row>                                                         |
|   7 |             |     +- Union All <Row>                                                           |
|   8 |             |        +- Union Input                                                            |
|   9 |             |           +- Compute <Row>                                                       |
|  10 |             |              +- Unit Relation <Row>                                              |
| *30 |             +- [Map] Distributed Cross Apply <Row>                                             |
|  31 |                +- [Input] Create Batch <Batch>                                                 |
|  32 |                |  +- RowToDataBlock                                                            |
|  33 |                |     +- Cross Apply <Row>                                                      |
|  34 |                |        +- [Input] KeyRangeAccumulator <Row>                                   |
|  35 |                |        |  +- Batch Scan on $v13 <Row> (scan_method: Row)                      |
|  41 |                |        +- [Map] Local Distributed Union <Row>                                 |
|  42 |                |           +- Filter Scan <Row> (seekable_key_size: 0)                         |
| *43 |                |              +- Index Scan on SingersByFirstLastName <Row> (scan_method: Row) |
|  61 |                +- [Map] Compute <Row>                                                          |
|  62 |                   +- Cross Apply <Row>                                                         |
|  63 |                      +- [Input] KeyRangeAccumulator <Row>                                      |
|  64 |                      |  +- DataBlockToRow                                                      |
|  65 |                      |     +- Batch Scan on $v19 <Batch> (scan_method: Batch)                  |
|  78 |                      +- [Map] Local Distributed Union <Row>                                    |
|  79 |                         +- Filter Scan <Row> (seekable_key_size: 0)                            |
| *80 |                            +- Table Scan on Singers <Row> (scan_method: Row)                   |
+-----+------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  3: Condition: (ISFALSE(IFNULL($row_exists_or_null, false)) OR ISTRUE(NOT(IS_NOT_DISTINCT_FROM($original_Status, $v8'))))
  4: Split Range: (IS_NOT_DISTINCT_FROM($scan_FirstName, $v6) AND IS_NOT_DISTINCT_FROM($scan_LastName, $v7))
 30: Split Range: ($scan_SingerId' = $scan_SingerId)
 43: Seek Condition: (IS_NOT_DISTINCT_FROM($scan_FirstName, $batched_v6) AND IS_NOT_DISTINCT_FROM($scan_LastName, $batched_v7))
 80: Seek Condition: ($scan_SingerId' = $batched_scan_SingerId')


=== dml/insert-on-conflict-select ===
INSERT INTO Singers (SingerId, FirstName, LastName) (SELECT SingerId, FirstName, LastName FROM AckworthSingers) ON CONFLICT(SingerId) DO NOTHING
+-----+-------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                        |
+-----+-------------------------------------------------------------------------------------------------+
|   0 | Apply Mutations on Singers <Row> (operation_type: INSERT)                                       |
|   1 | +- Serialize Result <Row>                                                                       |
|   2 |    +- Distributed Union on AckworthSingers <Row>                                                |
|  *3 |       +- Distributed Anti Semi Apply <Row>                                                      |
|   4 |          +- [Input] Create Batch <Row>                                                          |
|   5 |          |  +- Local Distributed Union <Row>                                                    |
|   6 |          |     +- Compute Struct <Row>                                                          |
|   7 |          |        +- Compute Struct <Row>                                                       |
|   8 |          |           +- Table Scan on AckworthSingers <Row> (Full scan, scan_method: Automatic) |
|  22 |          +- [Map] Semi Apply <Row>                                                              |
|  23 |             +- [Input] KeyRangeAccumulator <Row>                                                |
|  24 |             |  +- Batch Scan on $v10 <Row> (scan_method: Row)                                   |
|  28 |             +- [Map] Local Distributed Union <Row>                                              |
|  29 |                +- Filter Scan <Row> (seekable_key_size: 0)                                      |
| *30 |                   +- Table Scan on Singers <Row> (scan_method: Row)                             |
+-----+-------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  3: Split Range: ($scan_SingerId = $SingerId)
 30: Seek Condition: ($scan_SingerId = $batched_SingerId)


=== dml/insert-then-return ===
INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (17, 'Russell', 'Morales') THEN RETURN SingerId, FirstName || ' ' || LastName AS FullName
+----+-----------------------------------------------------------+
| ID | Operator                                                  |
+----+-----------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: INSERT) |
|  1 | +- Serialize Result <Row>                                 |
|  2 |    +- Union All <Row>                                     |
|  3 |       +- Union Input                                      |
|  4 |          +- Compute <Row>                                 |
|  5 |             +- Unit Relation <Row>                        |
+----+-----------------------------------------------------------+


=== dml/insert-fans-default-key-then-return ===
INSERT INTO Fans (FirstName, LastName) VALUES ('Melissa', 'Garcia') THEN RETURN FanId
+----+--------------------------------------------------------+
| ID | Operator                                               |
+----+--------------------------------------------------------+
|  0 | Apply Mutations on Fans <Row> (operation_type: INSERT) |
|  1 | +- Serialize Result <Row>                              |
|  2 |    +- Compute <Row>                                    |
|  3 |       +- Union All <Row>                               |
|  4 |          +- Union Input                                |
|  5 |             +- Compute <Row>                           |
|  6 |                +- Unit Relation <Row>                  |
+----+--------------------------------------------------------+


=== dml/update-literal ===
UPDATE Singers SET BirthDate = DATE '1990-10-10', SingerInfo = b'nationality:USA' WHERE FirstName = 'Marc' AND LastName = 'Richards'
+----+------------------------------------------------------------------------------------+
| ID | Operator                                                                           |
+----+------------------------------------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: UPDATE)                          |
|  1 | +- Serialize Result <Row>                                                          |
| *2 |    +- Distributed Union on SingersByFirstLastName <Row>                            |
|  3 |       +- Local Distributed Union <Row>                                             |
|  4 |          +- Filter Scan <Row> (seekable_key_size: 0)                               |
| *5 |             +- Index Scan on SingersByFirstLastName <Row> (scan_method: Automatic) |
+----+------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Split Range: (($FirstName = 'Marc') AND ($LastName = 'Richards'))
 5: Seek Condition: (IS_NOT_DISTINCT_FROM($FirstName, 'Marc') AND IS_NOT_DISTINCT_FROM($LastName, 'Richards'))


=== dml/update-default ===
UPDATE Singers SET Status = DEFAULT WHERE SingerId = 1
+----+---------------------------------------------------------------+
| ID | Operator                                                      |
+----+---------------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: UPDATE)     |
|  1 | +- Serialize Result <Row>                                     |
| *2 |    +- Distributed Union on Singers <Row>                      |
|  3 |       +- Local Distributed Union <Row>                        |
|  4 |          +- Filter Scan <Row> (seekable_key_size: 0)          |
| *5 |             +- Table Scan on Singers <Row> (scan_method: Row) |
+----+---------------------------------------------------------------+
Predicates(identified by ID):
 2: Split Range: ($SingerId = 1)
 5: Seek Condition: ($SingerId = 1)


=== dml/update-on-update-column ===
UPDATE Singers SET Status = 'inactive' WHERE SingerId = 10
+----+---------------------------------------------------------------+
| ID | Operator                                                      |
+----+---------------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: UPDATE)     |
|  1 | +- Serialize Result <Row>                                     |
| *2 |    +- Distributed Union on Singers <Row>                      |
|  3 |       +- Local Distributed Union <Row>                        |
|  4 |          +- Filter Scan <Row> (seekable_key_size: 0)          |
| *5 |             +- Table Scan on Singers <Row> (scan_method: Row) |
+----+---------------------------------------------------------------+
Predicates(identified by ID):
 2: Split Range: ($SingerId = 10)
 5: Seek Condition: ($SingerId = 10)


=== dml/update-array ===
UPDATE Concerts SET TicketPrices = [25, 50, 100] WHERE VenueId = 1
+----+----------------------------------------------------------------------+
| ID | Operator                                                             |
+----+----------------------------------------------------------------------+
|  0 | Apply Mutations on Concerts <Row> (operation_type: UPDATE)           |
|  1 | +- Serialize Result <Row>                                            |
| *2 |    +- Distributed Union on Concerts <Row>                            |
|  3 |       +- Local Distributed Union <Row>                               |
|  4 |          +- Filter Scan <Row> (seekable_key_size: 0)                 |
| *5 |             +- Table Scan on Concerts <Row> (scan_method: Automatic) |
+----+----------------------------------------------------------------------+
Predicates(identified by ID):
 2: Split Range: ($VenueId = 1)
 5: Seek Condition: ($VenueId = 1)


=== dml/update-subquery ===
UPDATE Singers SET Status = 'active' WHERE SingerId IN (SELECT SingerId FROM Albums)
+----+----------------------------------------------------------------------------------------+
| ID | Operator                                                                               |
+----+----------------------------------------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: UPDATE)                              |
|  1 | +- Serialize Result <Row>                                                              |
|  2 |    +- Distributed Union on Singers <Row> (split_ranges_aligned)                        |
|  3 |       +- Local Distributed Union <Row>                                                 |
|  4 |          +- Semi Apply <Row>                                                           |
|  5 |             +- [Input] Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
|  7 |             +- [Map] Local Distributed Union <Row>                                     |
|  8 |                +- Filter Scan <Row> (seekable_key_size: 0)                             |
| *9 |                   +- Table Scan on Albums <Row> (scan_method: Row)                     |
+----+----------------------------------------------------------------------------------------+
Predicates(identified by ID):
 9: Seek Condition: ($SingerId_1 = $SingerId)


=== dml/update-force-index ===
UPDATE Singers@{FORCE_INDEX=SingersByFirstLastName} SET Status = 'inactive' WHERE FirstName = 'Marc' AND LastName = 'Richards'
+----+------------------------------------------------------------------------------------+
| ID | Operator                                                                           |
+----+------------------------------------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: UPDATE)                          |
|  1 | +- Serialize Result <Row>                                                          |
| *2 |    +- Distributed Union on SingersByFirstLastName <Row>                            |
|  3 |       +- Local Distributed Union <Row>                                             |
|  4 |          +- Filter Scan <Row> (seekable_key_size: 0)                               |
| *5 |             +- Index Scan on SingersByFirstLastName <Row> (scan_method: Automatic) |
+----+------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Split Range: (($FirstName = 'Marc') AND ($LastName = 'Richards'))
 5: Seek Condition: (IS_NOT_DISTINCT_FROM($FirstName, 'Marc') AND IS_NOT_DISTINCT_FROM($LastName, 'Richards'))


=== dml/update-then-return ===
UPDATE Singers SET BirthDate = DATE '1990-10-10' WHERE FirstName = 'Russell' THEN RETURN SingerId, EXTRACT(YEAR FROM BirthDate) AS year
+----+------------------------------------------------------------------------------------+
| ID | Operator                                                                           |
+----+------------------------------------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: UPDATE)                          |
|  1 | +- Serialize Result <Row>                                                          |
| *2 |    +- Distributed Union on SingersByFirstLastName <Row>                            |
|  3 |       +- Local Distributed Union <Row>                                             |
|  4 |          +- Filter Scan <Row> (seekable_key_size: 0)                               |
| *5 |             +- Index Scan on SingersByFirstLastName <Row> (scan_method: Automatic) |
+----+------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Split Range: ($FirstName = 'Russell')
 5: Seek Condition: IS_NOT_DISTINCT_FROM($FirstName, 'Russell')


=== dml/delete-where ===
DELETE FROM Singers WHERE FirstName = 'Alice'
+----+------------------------------------------------------------------------------------+
| ID | Operator                                                                           |
+----+------------------------------------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: DELETE)                          |
|  1 | +- Serialize Result <Row>                                                          |
| *2 |    +- Distributed Union on SingersByFirstLastName <Row>                            |
|  3 |       +- Local Distributed Union <Row>                                             |
|  4 |          +- Filter Scan <Row> (seekable_key_size: 0)                               |
| *5 |             +- Index Scan on SingersByFirstLastName <Row> (scan_method: Automatic) |
+----+------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Split Range: ($FirstName = 'Alice')
 5: Seek Condition: IS_NOT_DISTINCT_FROM($FirstName, 'Alice')


=== dml/delete-subquery ===
DELETE FROM Singers WHERE FirstName NOT IN (SELECT FirstName FROM AckworthSingers)
+----+-----------------------------------------------------------------------------------------------+
| ID | Operator                                                                                      |
+----+-----------------------------------------------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: DELETE)                                     |
|  1 | +- Serialize Result <Row>                                                                     |
| *2 |    +- Hash Join <Row> (join_type: PROBE_ANTI_SEMI)                                            |
|  3 |       +- [Build] Distributed Union on AckworthSingers <Row>                                   |
|  4 |       |  +- Local Distributed Union <Row>                                                     |
|  5 |       |     +- Table Scan on AckworthSingers <Row> (Full scan, scan_method: Automatic)        |
|  8 |       +- [Probe] Distributed Union on SingersByFirstLastName <Row>                            |
|  9 |          +- Local Distributed Union <Row>                                                     |
| 10 |             +- Index Scan on SingersByFirstLastName <Row> (Full scan, scan_method: Automatic) |
+----+-----------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Condition: (true = true)
     Residual Condition: (ISNULL($FirstName) OR ISNULL($FirstName_1) OR ($FirstName = $FirstName_1))


=== dml/delete-force-index ===
DELETE FROM Singers@{FORCE_INDEX=SingersByFirstLastName} WHERE FirstName = 'Alice'
+----+------------------------------------------------------------------------------------+
| ID | Operator                                                                           |
+----+------------------------------------------------------------------------------------+
|  0 | Apply Mutations on Singers <Row> (operation_type: DELETE)                          |
|  1 | +- Serialize Result <Row>                                                          |
| *2 |    +- Distributed Union on SingersByFirstLastName <Row>                            |
|  3 |       +- Local Distributed Union <Row>                                             |
|  4 |          +- Filter Scan <Row> (seekable_key_size: 0)                               |
| *5 |             +- Index Scan on SingersByFirstLastName <Row> (scan_method: Automatic) |
+----+------------------------------------------------------------------------------------+
Predicates(identified by ID):
 2: Split Range: ($FirstName = 'Alice')
 5: Seek Condition: IS_NOT_DISTINCT_FROM($FirstName, 'Alice')


=== dml/delete-then-return ===
DELETE FROM Singers WHERE FirstName = 'Melissa' THEN RETURN * EXCEPT (LastUpdated)
+-----+---------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                    |
+-----+---------------------------------------------------------------------------------------------+
|   0 | Apply Mutations on Singers <Row> (operation_type: DELETE)                                   |
|   1 | +- Serialize Result <Row>                                                                   |
|  *2 |    +- Distributed Union on SingersByFirstLastName <Row>                                     |
|  *3 |       +- Distributed Cross Apply <Row>                                                      |
|   4 |          +- [Input] Create Batch <Batch>                                                    |
|   5 |          |  +- RowToDataBlock                                                               |
|   6 |          |     +- Local Distributed Union <Row>                                             |
|   7 |          |        +- Filter Scan <Row> (seekable_key_size: 0)                               |
|  *8 |          |           +- Index Scan on SingersByFirstLastName <Row> (scan_method: Automatic) |
|  18 |          +- [Map] Cross Apply <Row>                                                         |
|  19 |             +- [Input] KeyRangeAccumulator <Row>                                            |
|  20 |             |  +- DataBlockToRow                                                            |
|  21 |             |     +- Batch Scan on $v2 <Batch> (scan_method: Batch)                         |
|  26 |             +- [Map] Local Distributed Union <Row>                                          |
|  27 |                +- Filter Scan <Row> (seekable_key_size: 0)                                  |
| *28 |                   +- Table Scan on Singers <Row> (scan_method: Row)                         |
+-----+---------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  2: Split Range: ($FirstName = 'Melissa')
  3: Split Range: ($SingerId' = $SingerId)
  8: Seek Condition: IS_NOT_DISTINCT_FROM($FirstName, 'Melissa')
 28: Seek Condition: ($SingerId' = $batched_SingerId')

```
