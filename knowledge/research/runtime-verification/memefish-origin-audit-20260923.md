---
type: Research Note
title: "Memefish origin audit, 2026-09-23"
description: "Current upstream source and issue/PR coverage for recent managed Spanner verification findings."
tags: [memefish, parser, spanner, compatibility, issue-tracking]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: origin
    resource: https://github.com/cloudspannerecosystem/memefish/commit/1691dd67c4c71ca545be852b7a66dc08767082a2
    title: "Fetched origin HEAD and matching GitHub HEAD"
  - id: issues
    resource: https://github.com/cloudspannerecosystem/memefish/issues
    title: "All 458 historical issue/PR records fetched on 2026-09-23"
  - id: runtime
    resource: managed-release-notes-20260922.md
    title: "Recent managed release verification"
---

# Memefish origin audit

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Fetched origin HEAD and GitHub HEAD both identify
`1691dd67c4c71ca545be852b7a66dc08767082a2`. There are 39 open issues and 16
open PRs. The classification below covers this thread's recent syntax findings
and closely related parser fixes, not reproduction of every historical issue.
An open implementation PR is not a fix available in origin HEAD.

## Already supported or fixed in origin HEAD

- Expression indexes: merged PR #412; strict parse and SQL rendering succeed.
- Omitted primary keys and explicit `PRIMARY KEY ()`: both parse and remain
  distinct when rendered; the empty-key fix is merged PR #427.
- `AUTO_PARAMETER`: existing generic statement-hint syntax accepts and renders
  the ON value. No dedicated AST or parser change is needed for the option key.
- Queue receive TVF call shape: generic `RECEIVE_Q()` parses and renders.
  This does not establish support for every queue parameter form or semantics.
- Ordinary column OPTIONS accept arbitrary option keys syntactically; the
  managed rejection of description/synonyms is a service semantic constraint,
  not evidence that memefish needs a syntax fix.

## Unfixed in origin HEAD, with an open PR

| Finding | Issue | Open PR |
| --- | --- | --- |
| Queue DDL | #443 | #446 |
| Queue privileges | #444 | #447 |
| DELETE ASSERT_ROWS_MODIFIED | #445 | #448 |
| ALTER VECTOR INDEX REBUILD | #449 | #450 |
| Graph-level OPTIONS | #441 | #453 |
| TVF TABLESAMPLE lost by SQL() | #432 | #451 |
| HASH join method lost by SQL() | #433 | #458 |
| SELECT trailing comma before semicolon | #456 | #457 |
| ALTER TABLE IF EXISTS | #402 | #452 |
| SET pipe operator | #408 (umbrella) | #455 |

The first eight rows were reproduced with strict ParseStatements against an
archive of origin HEAD. The final two are tracked from current PR bodies and
diffs, not fresh runtime tests. The TVF sample and HASH join cases parse
successfully but lose syntax when rendered; acceptance alone is insufficient.

PR #446's current diff includes trailing-comma fixtures and Path-based queue
names, confirming that the reported follow-ups are present in the proposed
implementation. It remains open. PR #453 adds options to CreatePropertyGraph,
not graph labels/properties. Its empty-options parser helper is enabled for
graph options; it does not settle the separate CREATE FUNCTION / EXPORT DATA
request in #454. No PR branch was certified by a full test suite in this audit.

## Issues without a matching open implementation PR

- #442: description/synonyms on graph labels and properties. Both positions
  fail strict parsing in origin HEAD, despite managed execution succeeding.
- #454: empty OPTIONS for selected statement contexts. The graph-specific
  relaxation in #453 is not general completion of this issue.
- #431: string-valued single-part INTERVAL literals; still open. No fresh
  parser probe for this item was performed here.

Other open issues remain visible in the retained full inventory; these three
are the relevant nearby gaps, not an exhaustive list of unresolved bugs.

## No dedicated issue found / outside memefish scope

ANY/SOME/ALL comparisons and quantified LIKE still fail strict parsing. No
dedicated issue or PR was found in all 458 historical titles and bodies,
including closed records (comments were not exhaustively searched). The
[managed quantifier evidence](managed-quantified-comparisons-20260921.md) reports
explicit unsupported-feature errors, so this is an untracked compatibility
question rather than a demonstrated working managed feature needing repair.

No AUTO_PARAMETER-specific issue was found, but its syntax already works.
PostgreSQL ILIKE/function and sampling findings are outside memefish's
GoogleSQL dialect. INFORMATION_SCHEMA/SPANNER_SYS content, ROUTINES history,
and the survey graph-metadata reconstruction gap belong to service/catalog
or downstream code, not memefish syntax tracking.

## Evidence and limits

Persistent local evidence lives at
`<LOCAL_TASK_ROOT>/cloudspannerecosystem/memefish/origin-audit-20260923/`:
`head.json`, paginated `issues-all.json`, `prs-open.json` with exact PR heads,
ten PR diffs, `parser.jsonl`, `probe.go.txt`, `open-inventory.md`, and integrity
hashes. The source archive is checkout-scoped scratch. Run used Go 1.26.6 with
GOWORK=off and GOTOOLCHAIN=local; upstream requests 1.27.1, so this is a bounded
parser observation, not upstream CI validation. The EXPORT DATA sample fails
at the statement entry point; it cannot isolate empty-options handling.
No implementation, external issue, PR, comment, or merge was created.
