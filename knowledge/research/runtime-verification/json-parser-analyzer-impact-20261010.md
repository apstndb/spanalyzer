---
type: Research Note
title: "Parser and analyzer impact of the October 10 JSON verification"
description: "Strict memefish parsing and round trips, current spanalyzer row-type analysis, and a latest-binding control distinguish grammar gaps from Spanner-specific JSON casts and coercion."
tags: [spanner, memefish, analyzer, json, compatibility, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: runtime
    resource: cross-engine-json-and-change-streams-20261010.md
    title: "Managed, Omni r4, and Emulator runtime baseline"
  - id: receipt
    resource: evidence/json-parser-impact-20261010/receipt.json
    title: "Offline probe identities, source hashes, and command outcomes"
  - id: memefish
    resource: https://github.com/cloudspannerecosystem/memefish/commit/b2e4052e331584723633066850ee343fbcc6e095
    title: "Remote main verified October 10"
  - id: frontend
    resource: https://github.com/google/googlesql/blob/36dd14aa0657ea299725504bc0f938732f58f380/googlesql/public/cast.cc
    title: "Pinned frontend cast implementation"
  - id: binding
    resource: https://github.com/goccy/go-googlesql/commit/2b3726f947f7f809afb1daf534ff9ca06fd4da5c
    title: "Latest main resolved to v0.4.0 and tested separately"
  - id: known-range-bug
    resource: https://github.com/cloudspannerecosystem/memefish/issues/463
    title: "Existing ChangeStreamForAll source-range defect"
  - id: documentation-check
    resource: evidence/json-parser-impact-20261010/distinctness-documentation.json
    title: "Fresh documentation check distinguishing Spanner GoogleSQL, PostgreSQL, BigQuery, and the frontend"
---

# Parser and analyzer impact, October 10

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The new JSON behavior primarily requires **semantic analyzer compatibility**,
not new JSON grammar. Existing memefish nodes represent JSON literals, ordinary
comparison operators, CAST/SAFE_CAST, grouping, ordering, and change-stream
arguments. One separately identifiable parser gap was reproduced:
`IS DISTINCT FROM` executes on managed Spanner and Omni but fails strict
memefish parsing in both v0.8.1 and current main.

## Scope and identities

These are offline probes, using the earlier October 10 runtime evidence. No
managed or container rerun was performed. Source repositories were read only;
no parser implementation, dependency, or runtime target was changed.

- Spanalyzer: `c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c`, with existing
  uncommitted changes. Relevant source hashes are retained in the receipt.
- Workspace-selected dependencies: memefish v0.8.1, go-googlesql v0.3.0,
  googlesqlwasm2go v0.1.0. These are the effective workspace versions rather
  than the root go.mod's lower memefish requirement.
- Memefish main: `b2e4052e331584723633066850ee343fbcc6e095`, confirmed by a
  fresh remote read. Only AGENTS.md has a tracked local modification; parser.go
  matches the committed source.
- Latest binding main: `2b3726f947f7f809afb1daf534ff9ca06fd4da5c`, resolved by
  Go to v0.4.0. A separate module tests this version without changing the
  workspace. Both binding READMEs identify GoogleSQL frontend revision
  `36dd14aa0657ea299725504bc0f938732f58f380`.
- Go 1.26.8. Source, cases, module resolution, full probe outputs, and derived
  assertions are retained under `evidence/json-parser-impact-20261010/`.

## Memefish

The corpus contains the same 125 constant queries as the runtime baseline,
plus sixteen DDL, graph, stored-column, heartbeat, and malformed controls.
Each positive parse is followed by SQL(), a strict second parse, SQL-text
idempotence, and source-range checks. Returned errors are inspected directly;
recovered ASTs and a zero exit from a display tool are not acceptance evidence.

| Target | Successful parses | Expected negative rejections | Missing query forms | Additional range failure |
| --- | --- | --- | --- | --- |
| v0.8.1 | 129 | 5 | IS DISTINCT FROM, OVER, and five GRAPH_TABLE probes | SET FOR ALL |
| Main | 134 | 5 | IS DISTINCT FROM and OVER | SET FOR ALL |

All successful parses reparse and have idempotent SQL output. The strict probe
exits 1 because it exposes the listed gaps: eight assertion failures for
v0.8.1 and three for main. These are retained findings, not a passing parser
conformance suite.

- `SELECT JSON '1' IS DISTINCT FROM JSON '1.0'` is a demonstrated working
  runtime case. Grammar/AST/unparser support is a runtime-compatibility candidate,
  independent of JSON type semantics, rather than a required documentation-following
  change. A fresh October 10 retrieval finds no `IS DISTINCT FROM` or
  `IS NOT DISTINCT FROM` in the Spanner GoogleSQL operators reference (updated
  October 8, 19:54:18 UTC). The documented IS forms cover TRUE, FALSE, NULL,
  and UNKNOWN. Related official-documentation searches find no GoogleSQL claim;
  search absence is not proof of absence across every page. Spanner PostgreSQL,
  BigQuery, and the pinned frontend document the operator, but those are separate
  dialect/source evidence. The current runtime introduction date and the full
  NOT/NULL/NaN/type matrix are not established by this single executed case.
  No dedicated matching open issue
  or PR was found in the 43 current open records, including PRs. Closed records
  and comments were not searched exhaustively.
- Existing CAST, equality/order, GROUP BY, ORDER BY, DISTINCT, MIN/MAX, IN,
  BETWEEN, and conditional-expression forms need no JSON-specific AST change.
  Add representative round-trip fixtures for the newly verified uses.
- Five GRAPH_TABLE conversion/control queries fail in v0.8.1 but parse in
  main. This is existing graph parser progress, not a need for new CAST nodes.
- The OVER probe uses ROW_NUMBER, which managed and Omni explicitly report
  as unimplemented. It establishes service syntax acceptance, not executable
  window-function support or a new JSON feature. Track it separately.
- Option-only ALTER CHANGE STREAM and the 100 ms heartbeat call parse.
  The obsolete bare FOR ALL SET OPTIONS / SET examples are rejected, as are
  malformed casts/comparisons and procedural IF ... THEN through ParseQuery.
- SET FOR ALL parses and round trips but ends at byte 46 for a 43-byte input.
  This reproduces existing issue #463; it is not caused by the documentation
  update. The parser records the token after ALL as ALL's source position.

## Spanalyzer and GoogleSQL frontend

The default Spanner catalog already enables maximum development features and
does not disable FEATURE_JSON_TYPE_COMPARISON. The feature is enabled in the
probe. JSON/JSON comparison, grouping, ordering, DISTINCT, MIN/MAX, and equality
JOIN type analysis work, including stored-column controls. TO_JSON(node) and
TO_JSON(path) produce JSON row types.

However, ordinary source-to-JSON casts and comparison-specific non-JSON
coercion fail. Examples that execute on managed and Omni include:

```sql
SELECT CAST(1 AS JSON) AS result;
SELECT SAFE_CAST(1 AS JSON) AS result;
SELECT JSON '1' = 1 AS result;
SELECT JSON '2' IN (1, 2, 3) AS result;
SELECT JSON '2' BETWEEN 1 AND 3 AS result;
```

The current 136-query offline corpus yields 52 successful row-type analyses
and 84 analysis errors. Those totals include deliberately invalid queries,
evaluation-error cases, and an unregistered READ_JsonChanges TVF; they are not
a compatibility percentage. The retained comparison restricts its coverage
to the 125 original runtime cases and their successful managed results.
Of the 99 successful managed cases, 61 fail local analysis. The remaining
38 have matching row-type metadata; this comparison does not execute queries
in the local analyzer or compare evaluated values.

A separate v0.4.0 control reproduces the same cast and coercion errors with
maximum language features enabled. JSON/JSON comparison, grouping/ordering,
and IS DISTINCT FROM controls succeed. Merely upgrading from v0.3.0 to v0.4.0
or adding the already-enabled JSON comparison flag does not close these gaps.
The pinned frontend cast table lacks ordinary source-to-JSON conversions.

Recommended semantic work:

1. Add Spanner-compatible CAST and SAFE_CAST analysis through a frontend
   extension or a narrowly scoped compatibility layer. Keep result metadata
   and resolved expression types consistent; parser acceptance alone is not
   sufficient.
2. Implement the documented conversion rules specifically for comparison
   contexts, including supported IN-list and BETWEEN operand directions.
   Do not make JSON a general common supertype: mixed COALESCE remains invalid
   on managed and Omni, and mixed IN UNNEST/subquery forms also reject.
3. Retain SQL NULL, JSON null, non-finite values, and wide numeric controls.
   CAST cannot simply be replaced with TO_JSON: their NULL and non-finite-value
   behavior differs in the runtime baseline.
4. Keep graph-element/path CAST's analysis and execution status separate.
   Managed/Omni accept the syntax but do not implement execution; TO_JSON
   controls execute. This is not grounds to claim a working graph CAST.

No implementation approach was selected or patched in this investigation.

## Other downstream effects

Memefish-lsp's inspected diagnostics derive from memefish parse errors; no
independent JSON comparison type checker was found. The distinctness operator
needs a parser update and then dependency/diagnostic/formatting coverage.
Its existing v0.8.1 pin also predates the graph parsing controls verified in main.

Bounded source searches in gsqlutils, spanner-mycli, spanemuboost, and wrench
found no relevant local JSON comparability restriction or change-stream
heartbeat minimum of 1,000 ms. No production fix in those projects was
identified here. This is not a rerun of their complete downstream suites.
Conditional-expression clarification and PostgreSQL read_options do not
require a new GoogleSQL parser form; memefish is not a PostgreSQL parser.

New semantic regression coverage must preserve the engine matrix. Managed and
Omni are capable of these ordinary JSON casts/coercions; Emulator 1.5.58 is not.
An Emulator rejection must not redefine the managed-Spanner parser or analyzer
contract. Keep the unsupported Emulator cases explicit and exercise successful
semantics on a capable target.
