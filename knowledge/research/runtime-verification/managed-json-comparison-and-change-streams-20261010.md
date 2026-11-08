---
type: Research Note
title: "Managed Spanner JSON and documentation-diff verification, 2026-10-10"
description: "Runtime checks for JSON comparison and casts, 100 ms change-stream heartbeats, corrected ALTER examples, and conditional-expression semantics in the October 9 documentation snapshot."
tags: [spanner, managed, json, change-streams, conditional-expressions, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: requested-diff
    resource: https://github.com/apstndb/spanner-docs-mirror/compare/ea4e26a05fb66275a371157d99d7d9d70a19cf7a...622684f93418cd569c8c9204ba4ecc522494f631
    title: "Exact documentation comparison requested by the user"
  - id: receipt
    resource: evidence/docs-diff-20261010/receipt.json
    title: "Runtime scope, dependencies, cleanup, and retained source identities"
  - id: documentation
    resource: evidence/docs-diff-20261010/documentation.json
    title: "Full official document retrievals compared with the exact mirror head"
  - id: json-results
    resource: evidence/docs-diff-20261010/queries.jsonl
    title: "JSON conversion, comparison, grouping, and conditional-expression cases"
  - id: google-results
    resource: evidence/docs-diff-20261010/google-features.jsonl
    title: "Stored JSON, PROTO/ENUM, execution profiles, and heartbeat records"
  - id: pg-results
    resource: evidence/docs-diff-20261010/pg-features.jsonl
    title: "PostgreSQL heartbeat boundaries and read_options"
  - id: graph-results
    resource: evidence/docs-diff-20261010/graph-controls.jsonl
    title: "Graph execution versus graph-to-JSON conversion controls"
---

# Managed Spanner documentation-diff verification, October 10

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The sampled managed service executes the newly documented JSON comparison,
ordering, grouping, and most JSON casts. The retained cases also exercise the
100 ms change-stream heartbeat boundary. Treat this as first coverage of these
cases, not as a measured feature-introduction date: no identical earlier
runtime baseline is retained here.

## Source and runtime boundaries

The requested mirror comparison is pinned from
`ea4e26a05fb66275a371157d99d7d9d70a19cf7a` to
`622684f93418cd569c8c9204ba4ecc522494f631`. Its eleven changed documentation
pages were retrieved in full from the official Developer Knowledge API on
October 10. After removing the mirror's frontmatter, all eleven bodies match
the official bodies; this comparison ignores trailing whitespace only.
The changed JSON pages report an October 8 update time. Source update times
and the mirror's October 9 synchronization time do not date a runtime rollout.

The direct Go client bypasses local SQL parsers. The target comes from the
dedicated user-level verification profile; destination IDs and partition
tokens are redacted or hashed before retaining results. GoogleSQL and
PostgreSQL fixtures run in owned temporary databases, with successful deletion
and subsequent NotFound checks recorded. The local source checkout remains
dirty at `c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c`.
Probe builds use Go 1.26.8 and Cloud Spanner Go SDK v1.91.0.
No Omni or Emulator run is included in this observation.

## JSON comparison, grouping, and sorting

The [type reference](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-types#json_type)
now documents JSON comparison and removes JSON from the non-orderable,
non-groupable, and non-comparable lists. Observed cases agree:

| Case | Observed result |
| --- | --- |
| Different JSON types | JSON null < string < number < boolean < array < object |
| Numbers | `JSON '1' = JSON '1.0'`; 2 sorts before 10 |
| Booleans and strings | false < true; lexical string order |
| Objects | Key-order-independent equality; compare sorted keys before values, then prefixes |
| Arrays | Element-wise comparison and prefix ordering |
| GROUP BY | 1 and 1.0 share a group; reordered object keys share a group |
| DISTINCT, MIN/MAX, equality JOIN | Executed successfully on the bounded literal fixtures |
| Stored JSON column | Four rows sort by JSON object value; GROUP BY returns two groups of two |

SQL NULL and JSON null remain distinct. Ascending sorting places SQL NULL
before JSON null. JSON null equals JSON null, while comparison with SQL NULL
returns SQL NULL. The grouping fixture preserves separate SQL-NULL and
JSON-null groups.

Implicit comparison coercion succeeds in both operand orders for numeric,
BOOL, STRING, DATE, TIMESTAMP, UUID, BYTES equality, and supported arrays.
ENUM and ARRAY<ENUM> equality also succeed in an owned proto bundle.
BYTES/ENUM ordering coercion and STRUCT/INTERVAL comparison coercion return
InvalidArgument signature errors. Explicitly casting BYTES to JSON first
allows a JSON-string ordering comparison; that does not preserve byte order.

Mixed list-based IN and BETWEEN work with a JSON first operand. A non-JSON
first operand is rejected. Mixed JSON/non-JSON IN UNNEST and IN-subquery
forms are rejected, but the corresponding JSON/JSON forms execute. The
documentation's list-only rule appears under *implicit coercion*; it must not
be expanded into a ban on all JSON IN-subquery forms.

Comparison-specific coercion is not a general common-supertype rule:
`COALESCE(JSON '1', 1)` is rejected. The exploratory ROW_NUMBER case returns
Unimplemented for ROW_NUMBER itself and does not establish a JSON-ordering
failure.

## CAST AS JSON

The [new cast section](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/conversion_functions#cast_as_json)
agrees with the observed scalar, container, PROTO, and ENUM conversions:

| Input | Observed JSON result or error |
| --- | --- |
| INT64, FLOAT32, FLOAT64, NUMERIC, BOOL | JSON numbers or booleans |
| STRING | JSON string; JSON-looking text is not parsed |
| BYTES | Base64 JSON string |
| DATE, TIMESTAMP, UUID, INTERVAL | JSON strings; the interval control becomes `"P1D"` |
| ARRAY and STRUCT | JSON array/object; nested SQL NULL becomes JSON null |
| SQL NULL | SQL NULL; TO_JSON(NULL) instead returns JSON null |
| PROTO | Object with protobuf JSON mapping; the INT64 field is the string `"1"` |
| ENUM | JSON enum-name string `"ONE"` |
| JSON | Identity conversion |
| Infinity, -Infinity, NaN, nested infinity | OutOfRange; SAFE_CAST returns SQL NULL |
| Large NUMERIC not representable without precision loss | OutOfRange; SAFE_CAST returns SQL NULL |

The maximum signed INT64 control remains a JSON number. The tested large
NUMERIC fails in both CAST and default TO_JSON. TO_JSON(infinity) returns a
JSON string, whereas CAST rejects it; those differences must not be flattened
into interchangeable conversion functions.

The documentation additionally lists GRAPH_ELEMENT and GRAPH_PATH, but both
casts return **Unimplemented** in NORMAL and PROFILE modes on the sampled
target. Normal-mode node and edge controls execute, CAST of the node's INT64
property succeeds, and **TO_JSON of the same graph element and graph path
succeeds**. This isolates a graph-to-JSON CAST gap rather than a general graph
query or JSON-serialization failure. TO_JSON is an observed alternative for
these non-null fixture values; it does not make the functions interchangeable
for SQL NULL or non-finite numbers.

## Change Streams

The [read-function reference](https://docs.cloud.google.com/spanner/docs/change-streams/details)
changes the heartbeat minimum from 1,000 to 100 ms in both dialects and names
the PostgreSQL read_options parameter as optional text[]. Boundary and actual
partition-record probes are retained separately from plan acceptance.

GoogleSQL accepts 100, 999, 1,000, and 300,000 ms and rejects 99 and 300,001 ms
with an explicit min-100/max-300000 error. A two-second child-partition read
at 100 ms returns eighteen heartbeat records. Adjacent record timestamps
have a median gap of approximately 100.058 ms; the final record is clipped
to the requested end timestamp. Record timestamps and client arrival times
are both retained; a requested heartbeat period is not a network-latency SLA.

PostgreSQL repeats the same boundary results and returns seventeen actual
100 ms child-partition heartbeats. Both dialects permit omitted read_options
or NULL and reject both an empty array and a nonempty array with
`read_options must be null`. The PostgreSQL calls use positional `$1`–`$3`
parameters and corresponding p1–p3 bindings.

The [management-example correction](https://docs.cloud.google.com/spanner/docs/change-streams/manage)
removes `FOR ALL` before SET OPTIONS / SET. Runtime tests in both dialects
confirm that the old combined form is a syntax error. Four valid option-only
ALTER statements toggle exclude_ttl_deletes and allow_txn_exclusion;
INFORMATION_SCHEMA shows that the selected table and its single selected
column remain unchanged throughout. The sampled option metadata also tracks
the requested values.

The separate formal tracking action is `SET FOR ALL`, as specified in the
[DDL reference](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_change_stream).
The exploratory `FOR ALL`-only control also lacks SET and is rejected; it is
not evidence that the correctly spelled tracking action is unavailable.

## Conditional expressions

The [revised contract](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/conditional_expressions#short_circuiting)
guarantees results and evaluation-error behavior *as if* short-circuited. It
allows physical work in unchosen expressions, so it does not guarantee that
COALESCE, IF, or CASE saves all unchosen subquery work.

COALESCE, IFNULL, IF, simple/searched CASE, and scalar-subquery controls return
the chosen value while suppressing unchosen division-by-zero errors. Choosing
the erroneous expression returns OutOfRange. An unrecognized identifier in
an unchosen branch still fails name resolution: evaluation-error suppression
does not exempt branches from analysis.

Two table-backed PROFILE cases return 10 and scan one row. Fallback Scan nodes
remain in their plans but have no execution statistics; this fixture does
**not** reproduce physical execution of unchosen subqueries. Plan membership
alone does not prove execution, and this observation does not restore the
removed universal lazy-execution guarantee.

The revised page suggests procedural `IF ... THEN`. The direct Spanner SQL
API rejects `IF TRUE THEN SELECT 1; END IF` with `Unexpected keyword IF`.
The recommendation is not an executable replacement in this tested API.
No separate procedural execution service was tested.

## Other changed pages and limits

- nam15 now lists us-central2 as an optional read-only replica. The instance
  configuration API advertises us-central2, us-east1, and us-west3. No replica
  provisioning or directed data read was performed.
- The metrics page changes only the generated-table timestamp; the requested
  diff supplies no metric-definition change.
- The label-propagation reference changes seed labels to non-negative integers.
  Its invocation documentation uses a graph-analysis EXPORT DATA job. No such
  job was submitted; rejection of a negative seed remains unverified.

Probe diagnostics retain early termination after a deliberately invalid ALTER,
an initial PostgreSQL catalog-name case mismatch, a database-quota failure,
and a later PostgreSQL query using GoogleSQL parameter syntax. These are
producer/environment limitations, not heartbeat feature failures. Corrected
captures are selected explicitly; earlier raw outcomes remain intact.

The offline evidence checks and the repository's OKF/language gates validate
the report and retained artifacts. They do not identify a managed build,
prove fleet-wide rollout completion, or replace the runtime records.
