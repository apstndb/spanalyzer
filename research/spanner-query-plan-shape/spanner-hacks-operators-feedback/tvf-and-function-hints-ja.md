# spanner-hacks operators evidence: TVF and function hints

Generated from `go run ./tools/spanner-query-plan-shape --case tvf --output reference --continue-on-error` and `go run ./tools/spanner-query-plan-shape --case function_hint --output reference --continue-on-error` on 2026-05-06 with Spanner Omni.

## Change-stream TVF

```text
=== tvf/change-stream ===
SELECT ChangeRecord FROM READ_EverythingStream (start_timestamp => TIMESTAMP "2026-05-06T00:00:00Z")
+----+---------------------------+
| ID | Operator                  |
+----+---------------------------+
|  0 | Serialize Result <Row>    |
|  1 | +- ChangeStream TVF <Row> |
+----+---------------------------+

```

## Function hints

```text
=== function-hint/default_inline ===
SELECT
  SUBSTRING(CAST(x AS STRING), 2, 5) AS w,
  SUBSTRING(CAST(x AS STRING), 3, 7) AS y
FROM (
  SELECT SHA512(s.SingerInfo) AS x
  FROM Singers AS s
)
+----+--------------------------------------------------------------------------+
| ID | Operator                                                                 |
+----+--------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row>                                       |
|  1 | +- Local Distributed Union <Row>                                         |
|  2 |    +- Serialize Result <Row>                                             |
|  3 |       +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
+----+--------------------------------------------------------------------------+


=== function-hint/disable_inline_false ===
SELECT
  SUBSTRING(CAST(x AS STRING), 2, 5) AS w,
  SUBSTRING(CAST(x AS STRING), 3, 7) AS y
FROM (
  SELECT SHA512(s.SingerInfo) @{DISABLE_INLINE=FALSE} AS x
  FROM Singers AS s
)
+----+--------------------------------------------------------------------------+
| ID | Operator                                                                 |
+----+--------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row>                                       |
|  1 | +- Local Distributed Union <Row>                                         |
|  2 |    +- Serialize Result <Row>                                             |
|  3 |       +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
+----+--------------------------------------------------------------------------+


=== function-hint/disable_inline_true ===
SELECT
  SUBSTRING(CAST(x AS STRING), 2, 5) AS w,
  SUBSTRING(CAST(x AS STRING), 3, 7) AS y
FROM (
  SELECT SHA512(s.SingerInfo) @{DISABLE_INLINE=TRUE} AS x
  FROM Singers AS s
)
+----+-----------------------------------------------------------------------------+
| ID | Operator                                                                    |
+----+-----------------------------------------------------------------------------+
|  0 | Distributed Union on Singers <Row>                                          |
|  1 | +- Local Distributed Union <Row>                                            |
|  2 |    +- Serialize Result <Row>                                                |
|  3 |       +- Compute <Row>                                                      |
|  4 |          +- Table Scan on Singers <Row> (Full scan, scan_method: Automatic) |
+----+-----------------------------------------------------------------------------+

```
