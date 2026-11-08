---
type: Research Note
title: "October 31 documentation controls across engines and memefish gaps"
description: "Direct SDK probes test queue declaration case, pipe operators, scorer versions, search sort keys and timestamp buckets, with a current-head parser and public-PR audit."
tags: [spanner, managed, omni, emulator, memefish, syntax, queues, search, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: documentation
    resource: documentation-delta-review-20261031.md
    title: "Exact mirror interval and nine current official document bodies"
  - id: summary
    resource: evidence/docs-delta-runtime-20261031/summary.json
    title: "Completed runtime observations and explicit pending boundaries"
  - id: parser
    resource: evidence/docs-delta-runtime-20261031/parser-audit.json
    title: "Strict current-head parser results and refreshed public PR coverage"
  - id: gql-reference
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-query-statements#gql_set
    title: "Officially documented GQL column propagation syntax and semantics"
  - id: gql-documentation-addendum
    resource: evidence/docs-delta-runtime-20261031/gql-documentation-addendum.json
    title: "Current official body and October 20 mirror introduction"
  - id: provenance
    resource: evidence/docs-delta-runtime-20261031/sources/parser-source-provenance.json
    title: "Go module sources match three files fetched at the exact upstream SHA"
  - id: images
    resource: evidence/docs-delta-runtime-20261031/sources/selected-images.json
    title: "Exact native arm64 image manifests reused for the runtime comparison"
  - id: google-plan
    resource: evidence/docs-delta-runtime-20261031/google-cases.json
    title: "Core GoogleSQL controls, including all fifteen documented pipe operators"
  - id: final-plan
    resource: evidence/docs-delta-runtime-20261031/google-final-cases.json
    title: "Corrected SEARCH, sort-key and TVF controls, plus FROM/pipe isolation"
  - id: pg-plan
    resource: evidence/docs-delta-runtime-20261031/pg-final-cases.json
    title: "PostgreSQL scorer and timestamp-bucket boundary controls"
  - id: inspection
    resource: evidence/docs-delta-runtime-20261031/interrupted-managed-inspection.json
    title: "Read-only inspection of the managed fixture retained after a timeout"
  - id: receipt
    resource: evidence/docs-delta-runtime-20261031/receipt.json
    title: "Scope, preserved earlier artifacts and outstanding approval boundary"
---

# Documentation-driven runtime and parser follow-up

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The new queue declaration wording matches the managed and Omni samples:
`Payload` and `payload` work, while `PAYLOAD` and `PaYlOaD` fail. Emulator
1.5.59 accepts all four. Managed and both Omni images execute all fifteen
documented pipe-operator examples. Current memefish rejects ten operators;
this is broader than the recently clarified AGGREGATE grouping aliases.

The observations use direct Spanner SDK calls, so memefish cannot intercept
server probes. The parser is tested independently through `ParseStatements`,
requiring a nil error and successful reparse of unparsed accepted statements.
The exact target is main and v0.9.0 at
`da2ddfdd6277527f9dc2a54b87221579a9b18abe`.

## Runtime scope

| Target | Identity and interpretation |
| --- | --- |
| Managed | Dedicated user-level profile; connection destinations are not retained. The initial core run completed and removed both owned databases. A later corrected GoogleSQL fixture timed out and remains pending explicit deletion approval. Final scorer/index controls require explicit fixture approval. |
| Omni regular | `2026.r5`, native arm64 platform manifest `sha256:e506672206e0291cac28cbe19ad4cd470cec112c24601637bc896c65f7362396` |
| Omni LTS patch | `2026.r4-lts.1`, native arm64 platform manifest `sha256:5e3cf199f5abf3a09cd2348ab36c129b3268966eb4eb72fca278fe58252fcbdb` |
| Emulator | `1.5.59`, native arm64 platform manifest `sha256:969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8`; the refreshed official latest release remains v1.5.59. |

The [initial](evidence/docs-delta-runtime-20261031/local-commands.json),
[corrected](evidence/docs-delta-runtime-20261031/local-corrected-commands.json)
and [final](evidence/docs-delta-runtime-20261031/local-final-commands.json)
local receipts distinguish container lifecycles, image configuration, owned
cleanup and preservation of pre-existing containers. These are dated samples,
not a managed fleet rollout date or a benchmark.

## Queue names and grouping

| Control | Managed | Both Omni images | Emulator 1.5.59 |
| --- | --- | --- | --- |
| Declare `Payload` or `payload` | OK | OK | OK |
| Declare `PAYLOAD` or `PaYlOaD` | InvalidArgument: requires Payload or payload | Same rejection | Both declarations succeed |
| Fifteen pipe operators using a FROM-led input | All execute | All execute | All reject at the initial FROM token; this alone does not establish which operator caused rejection |
| AGGREGATE grouping item with explicit or bare alias | Executes with grouping columns before aggregate columns | Same | FROM-led fixture cannot reach grouping |
| GROUP BY ordinal inside AGGREGATE | Explicit ordinal-not-allowed error | Same | FROM-led fixture cannot reach grouping |
| GROUP BY an aggregate output alias absent from the input | Unrecognized-name error | Same | FROM-led fixture cannot reach grouping |

SELECT-led SELECT/WHERE/AGGREGATE controls separately return **Pipe query
syntax not supported** on Emulator. The FROM-only query instead has a syntax
error at FROM. Thus the pipe controls explicitly report a missing capability,
while the initial FROM-led failures cannot establish individual operator
behavior. Exact messages remain in the
[summary](evidence/docs-delta-runtime-20261031/summary.json).
The declaration-case discrepancy is separate from the earlier lease renewal
and stale-RECEIVE candidates; this note does not claim a fresh exhaustive
upstream duplicate search for that new discrepancy.

## Scorer, sort-key and bucket results

The first probes exposed fixture requirements. SCORE needs a generated
TOKENLIST column and a compatible SEARCH predicate. A search-index sort key
must be INT64 and NOT NULL: directly sorting a TIMESTAMP or a nullable
generated hour column fails. The successful corrected sort-key controls use
generated non-NULL epoch seconds and hour-start seconds. Earlier failed
attempts remain retained as fixture corrections rather than feature failures.

On the completed Omni final samples, scorer versions 1 and 2 produce distinct
values. Setting the GoogleSQL database default to 1 or 2 changes the unqualified
score accordingly, while explicit query options retain the selected version.
For one row, the scores are approximately 0.410131 for version 1 and 1.645082
for version 2. This tests current configuration precedence; it cannot observe
an automatic change on a future scorer release. Emulator's small fixture
produces equal term-count scores for both versions, so it does not demonstrate
equivalent scorer-version semantics.

The documented PostgreSQL statement setting `"spanner.score_version"` to
either 1 or 2 is rejected as an unsupported database option by the managed
sample, both Omni images and Emulator. Query-level version options are a
separate control: both Omni images execute explicit versions and produce distinct
values; Emulator executes them with equal scores on this fixture. This is a documentation/runtime
discrepancy for the statement as written, not a parser syntax failure.

The new PostgreSQL hourly-bucket example also has boundary errors in the
completed local samples. Casting epoch seconds to bigint before division
rounds fractional seconds; integer division truncates negative values:

| Timestamp | Documented expression | Floor-based hour index |
| --- | --- | --- |
| 1969-12-31 23:59:59Z | 0 | -1 |
| 2026-10-31 00:59:59.4Z | 498168 | 498168 |
| 2026-10-31 00:59:59.9Z | 498169 | 498168 |
| 2026-10-31 01:00:00Z | 498169 | 498169 |

The paired control uses
`FLOOR(EXTRACT(EPOCH FROM t - '1970-01-01Z'::timestamptz) / 3600)::bigint`.
The GoogleSQL example returns hour-start epoch seconds, whereas this PostgreSQL
expression returns an hour index. Those units should not be compared directly.
The managed boundary extension remains pending; the earlier managed
whole-second examples do not establish its fractional or pre-epoch behavior.
No write-overhead measurements are performed.

## memefish syntax without a matching public PR

The refreshed audit reads all 303 public PR records and six open PRs, and
inspects the old Graph draft #99 separately. The main SHA is unchanged.
This focused audit does not cover private trackers or every Spanner grammar
form. The [parser audit](evidence/docs-delta-runtime-20261031/parser-audit.json)
records 84 strict statement cases and the source/design boundaries.

| Unimplemented syntax | Runtime evidence | Existing work and disposition |
| --- | --- | --- |
| Pipe EXTEND, DROP, RENAME, AGGREGATE, JOIN, ORDER BY, UNION, INTERSECT, EXCEPT and TABLESAMPLE | All ten execute on this managed sample and both Omni images; current main returns a non-nil parse error for each. | [memefish #408](https://github.com/cloudspannerecosystem/memefish/issues/408) tracks complete pipe syntax. No matching open implementation PR was found. Prioritize AGGREGATE, including aliases and GROUP AND ORDER BY, then the other missing operators. |
| GQL set-operation column propagation: FULL OUTER, LEFT OUTER and INNER | FULL/INNER execute on managed and both Omni images; valid shared-column LEFT executes on both Omni images. Emulator recognizes the forms but rejects column propagation semantically. Current memefish rejects the prefixes. | The local `feat/gql-set-operation-propagation-mode` candidate exists at `60defb4942d0dde0431214591bb17cae0c311def`; no matching public PR was found. The broad historical draft #99 does not supply these modes. Reuse the candidate rather than start a duplicate implementation. |

### Official GQL documentation status

The official [Set operation reference](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-query-statements#gql_set)
explicitly documents the following grammar:

```text
column_propagation_mode:
  { FULL [ OUTER ] | OUTER | LEFT [ OUTER ] | INNER }
```

FULL OUTER/FULL/OUTER preserves both inputs' column names and fills missing
values with NULL. LEFT OUTER/LEFT preserves the left input's columns; INNER
keeps common columns. With no mode, inputs must have matching column-name
sets. The GQL CORRESPONDING keyword remains unsupported; the reference
explicitly permits column propagation modes and includes a FULL OUTER UNION
ALL example.

These additions appear in the [October 20 UTC mirror capture](https://github.com/apstndb/spanner-docs-mirror/commit/9f3859b9ea131f481bca92225c4cd4c020d5885b),
before the October 28 baseline used for the latest two-commit documentation
audit. The [addendum receipt](evidence/docs-delta-runtime-20261031/gql-documentation-addendum.json)
retains the refreshed official body, its update metadata and the actual
introduction patch. This is an **officially documented Spanner syntax gap in
the tested memefish main**, supported by the separately retained runtime cases.

These parser gaps are distinct from semantic restrictions. Queue declaration
names and scorer option values already fit ordinary identifier/argument
syntax. A parser should not be made responsible for runtime-only grouping
validity merely because the service rejects an ordinal.

Two other current-main gaps already have public PRs and are excluded from the
requested uncovered list: foreign-key SET NULL has
[PR #479](https://github.com/cloudspannerecosystem/memefish/pull/479), and TVF
aliases now have [PR #481](https://github.com/cloudspannerecosystem/memefish/pull/481).
The latter appeared during this audit. Corrected `SELECT * FROM RENEWLEASE_...`
controls execute with AS/bare aliases on the local engines; managed ML.PREDICT
alias controls reach model resolution. COUNT over the queue TVF is an unsupported
composition and is not counted as successful prediction or lease execution.

## Outstanding managed fixture and validation

The timed-out managed DDL is **inconclusive**. Subsequent read-only inspection
finds one READY database whose encoded name time falls inside the recorded
create window, with only the ScoreDocs DDL. Thus a timeout is not proof of no
schema side effect. Automatic approval review rejected deleting that candidate
because name/time inference might misidentify ownership, then also rejected
new managed fixture creation/deletion without explicit approval. Separate
approval questions are pending for the remaining final controls and this one
retained candidate. No rejected action is bypassed.

The [November 3 follow-up](managed-revalidation-and-omni-differences-20261103.md)
records later explicit authorization for two replacement disposable fixtures
and their scorer/search/PostgreSQL/GQL verification. That authorization excludes
the older retained candidate; its deletion approval remains separate. The
October 31 timeout and partial result are not rewritten as a completed run.

The [receipt](evidence/docs-delta-runtime-20261031/receipt.json) preserves all
1,917 earlier artifacts and 123 measured implementation/dependency files.
[Assertions](evidence/docs-delta-runtime-20261031/assertions.json) check completed
samples only; [validation](evidence/docs-delta-runtime-20261031/validation.json)
records repository documentation gates. No implementation change, commit,
push, public issue or PR is made by this task.
