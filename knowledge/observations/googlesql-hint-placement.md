---
type: Observation
title: GoogleSQL Hint Placement Inventory
description: Normalized generic hint placements in a pinned upstream GoogleSQL grammar, separated from Spanner verification layers.
tags: [googlesql, hints, grammar, spanner]
status: draft
sources:
  - id: googlesql-grammar
    resource: https://github.com/google/googlesql/blob/fd972655db97deac02f0696ea652a390209b794b/googlesql/parser/googlesql.tm
    title: GoogleSQL parser grammar at fd972655
    last_modified: 2026-07-17
  - id: frontend-tests
    resource: ../../analyzer_hint_position_test.go
    title: spanalyzer GoogleSQL frontend hint-position tests
  - id: runtime-audit
    resource: ../../research/spanner-query-plan-shape/HINT_POSITION_AUDIT_2026-08-04.md
    title: spanalyzer frontend, Emulator, and Omni hint-position audit
  - id: runtime-cases
    resource: ../../tools/spanner-query-plan-shape/hint_position_cases.go
    title: spanalyzer runtime hint-position cases
---

# GoogleSQL Hint Placement Inventory

Observed: 2026-08-04

Status: **grammar inventory complete; runtime coverage incomplete**. This note
normalizes every generic `hint` placement in the reviewed upstream GoogleSQL
grammar into semantic families. It does not claim that every generic GoogleSQL
placement or hint key is supported by Cloud Spanner.

## Provenance and scope

- Upstream grammar:
  [`google/googlesql` `googlesql.tm`](https://github.com/google/googlesql/blob/fd972655db97deac02f0696ea652a390209b794b/googlesql/parser/googlesql.tm),
  commit `fd972655db97deac02f0696ea652a390209b794b`, retrieved
  2026-08-04. The file SHA-256 was
  `d328af7413f4a0574d6560aa2182824711004095cfc2463cc17a79598c865cb9`.
- Local parser/frontend under test: `github.com/goccy/go-googlesql@v0.3.0`.
- spanalyzer comparison snapshot:
  `653a014e379daa2941f9eb7164d8df0317bdf6fb`.
- Runtime evidence is summarized separately in the
  [hint-position audit](../../research/spanner-query-plan-shape/HINT_POSITION_AUDIT_2026-08-04.md).

The inventory distinguishes four evidence layers:

1. an upstream grammar placement;
2. local frontend parse, AST retention, and round-trip behavior;
3. runtime syntax acceptance;
4. a plan-visible optimizer effect.

In this note, "runtime" refers to the pinned Emulator and Omni runs in the
cited audit; managed Cloud Spanner was not exercised.

Evidence at one layer does not imply support at a later layer. In particular,
the Cloud Spanner Emulator is exercised through statement execution rather
than `AnalyzeQuery`: its `PLAN` and `PROFILE` modes do not return plans.

The keyword forms `HASH JOIN` and `LOOKUP JOIN` are parsed by the separate
`join_hint` production. They are not generic `@{...}` hint placements and are
therefore outside the 32 families below.

## Normalized placement families

Multiple grammar alternatives that produce the same semantic placement are
one family. For example, the three forms of GQL braced `EXISTS` and duplicated
parser alternatives for `SELECT`, `JOIN`, and `INSERT` do not inflate the
count.

| # | Domain | Placement | Local evidence |
|---:|---|---|---|
| 1 | Statement | Before the statement body | Frontend and runtime covered; `DEFINE MACRO` is a targeted rejection |
| 2 | Query | Immediately after `SELECT` | Frontend and runtime covered |
| 3 | Query | After a table path, `UNNEST`, or CTE reference and before its alias | Table case covered; `UNNEST` and CTE contexts not isolated |
| 4 | Query | After a TVF call | Frontend and runtime position covered |
| 5 | Query | Immediately after `JOIN` | General matrix covered |
| 6 | Query | After `UNION`, `INTERSECT`, or `EXCEPT`, before `ALL` or `DISTINCT` | Frontend and runtime covered, including the same-level-chain restriction |
| 7 | Query | Between `GROUP` and `BY` | General matrix covered; function-internal and pipe contexts not isolated |
| 8 | Query | Between `ORDER` and `BY` | Frontend and runtime covered; window, `MATCH_RECOGNIZE`, and pipe contexts not isolated |
| 9 | Query | Between `PARTITION` and `BY` | Window context covered; `MATCH_RECOGNIZE` context not isolated |
| 10 | Expression | After a function call and before optional `OVER` | Frontend and general matrix covered |
| 11 | Predicate | Between SQL `EXISTS` and its parenthesized query | Frontend, runtime, and plan-effect combinations covered |
| 12 | Predicate | Between SQL `IN` and its parenthesized query | Frontend, runtime, and plan-effect combinations covered |
| 13 | Predicate | After `LIKE ANY`, `LIKE SOME`, or `LIKE ALL` and before a query RHS | Frontend and runtime position covered; base runtime feature unsupported |
| 14 | Predicate | After a quantified comparison `ANY`, `SOME`, or `ALL` and before a query RHS | Frontend and runtime position covered; base runtime feature unsupported |
| 15 | GQL | Immediately after `MATCH` | General graph matrix covered |
| 16 | GQL | Immediately after `OPTIONAL MATCH` | Not isolated |
| 17 | GQL | Immediately after `RETURN` | Frontend and runtime covered |
| 18 | GQL | In `WITH`, after optional `ALL` or `DISTINCT` | Frontend and runtime covered |
| 19 | GQL | Between `ORDER` and `BY` | Frontend and runtime covered |
| 20 | GQL | Between braced `EXISTS` and its graph subquery | Frontend and runtime covered; not every grammar alternative is isolated |
| 21 | GQL | Between `VALUE` and its braced graph subquery | Frontend and runtime covered |
| 22 | GQL path | After a path-list comma and before the next graph path pattern | Not isolated |
| 23 | GQL path | Between graph path factors | Edge-to-node covered; node-to-edge and node-to-node contexts not isolated |
| 24 | GQL element | At the start of a node or edge pattern filler | Edge filler covered; node filler not isolated |
| 25 | DML | After an `INSERT` target | Frontend and runtime covered |
| 26 | DML | After a `DELETE` target | Frontend and existing DML matrix covered |
| 27 | DML | After an `UPDATE` target | Frontend and existing DML matrix covered |
| 28 | Pipe | After `FINISH` | Present upstream; rejected by the pinned frontend and tested runtimes |
| 29 | Pipe | After `LOG` | Frontend and runtime position covered |
| 30 | Pipe | After `IF` | Not isolated |
| 31 | Pipe | After `FORK` | Not isolated |
| 32 | Pipe | After `TEE` | Not isolated |

Pipe operators can also reuse ordinary placements through their existing
`SELECT`, `JOIN`, `GROUP BY`, `ORDER BY`, set-operation, TVF, and `INSERT`
productions. Those are contexts of the families above, not additional syntax
positions.

## Deliberate non-placements and restrictions

The grammar contains targeted errors or hint-free productions that are as
important as the positive placements:

- Statement hints are rejected on `DEFINE MACRO`.
- A hint cannot lead a parenthesized GQL subpath.
- A hint between two GQL edge patterns is rejected.
- `IN`, `LIKE ANY/SOME/ALL`, and quantified-comparison hints are rejected for
  `UNNEST` and value-list right-hand sides; the parenthesized RHS must be a
  query.
- GQL set operations have no generic hint slot.
- DDL and other restricted contexts deliberately reuse `*_no_hint`
  productions.
- For a same-level SQL set-operation chain, the parser can retain a hint on a
  later operation, but analysis requires the hint to appear on the first
  operation.

## Remaining verification work

The highest-value gaps are context variants, not newly discovered grammar
families:

1. add frontend fixtures for pipe `IF`, `FORK`, and `TEE`, while recording the
   pinned frontend's divergence for `FINISH`;
2. cover `OPTIONAL MATCH`, path-list comma hints, node-to-edge and node-to-node
   path-factor hints, and node fillers;
3. isolate function-internal `GROUP BY` and `ORDER BY`, window `ORDER BY`, and
   `MATCH_RECOGNIZE` partition/order contexts;
4. isolate table-path variants for `UNNEST` and CTE references;
5. exercise ordinary hint productions when reached through pipe operators;
6. rerun supported candidates on Omni before classifying any hint key as
   plan-effective.

These gaps should be tracked separately as frontend grammar, runtime syntax,
and plan-effect checks. A generic grammar placement should not be admitted to
a Spanner-facing contract solely because upstream GoogleSQL parses it.
