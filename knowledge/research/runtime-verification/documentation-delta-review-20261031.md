---
type: Research Note
title: "Spanner documentation delta through October 30, reviewed October 31"
description: "A complete two-commit audit separates scorer and search-index guidance, queue and pipe grouping contracts, Omni LTS download updates, and configuration corrections from runtime observations."
tags: [spanner, documentation, managed, omni, emulator, queues, search, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: target
    resource: https://github.com/apstndb/spanner-docs-mirror/commit/1e880188144a326e383778ebc165a63197efbf70
    title: "User-designated October 30 mirror capture"
  - id: interval
    resource: https://github.com/apstndb/spanner-docs-mirror/compare/6d7397601456dc5602c0926d254424f64643a64f...1e880188144a326e383778ebc165a63197efbf70
    title: "Complete interval from the previous managed documentation baseline"
  - id: files
    resource: evidence/docs-delta-20261031/file-audit.json
    title: "Every changed file and its disposition"
  - id: official
    resource: evidence/docs-delta-20261031/official-comparison.json
    title: "Nine complete current official API bodies match the pinned mirror"
  - id: configurations
    resource: evidence/docs-delta-20261031/configuration-table-comparison.json
    title: "All 69 configuration rows and explicit presentation normalization"
  - id: managed
    resource: managed-documentation-delta-revalidation-20261029.md
    title: "Earlier managed runtime coverage and its source baseline"
  - id: containers
    resource: cross-engine-doc-delta-20261030.md
    title: "Earlier exact-image Omni and Emulator runtime comparison"
  - id: candidates
    resource: emulator-issue-candidates-20261031.md
    title: "Earlier queue-validation and upstream issue triage"
  - id: receipt
    resource: evidence/docs-delta-20261031/receipt.json
    title: "Source identities, scope, observation date and retained raw sources"
---

# Documentation review, October 31 JST

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The designated capture adds explicit guidance about automatic scorer-version
updates and the write overhead of changing search-index sort keys. The wider
interval also changes queue payload-name wording, clarifies pipe aggregation,
and updates Omni download examples to **2026.r4-lts.1**. These are documentation
observations; this review executes no managed, Omni or Emulator queries.

## Exact source interval

The previous managed report reviewed mirror commit
`6d7397601456dc5602c0926d254424f64643a64f`. Two further captures are present:

| Capture commit | Mirror message date | Scope |
| --- | --- | --- |
| `51280d38ddf737240e6ee86b0da9138f6f790c1b` | October 29 | Omni LTS patch examples, queue/pipe contracts, release announcement, configuration prose and editorial changes |
| `1e880188144a326e383778ebc165a63197efbf70` | October 30 | Five document paths and two bookkeeping files |

Remote main equals the designated commit at retrieval. The combined interval
contains **45 files: 43 document paths and two bookkeeping files**. Thirty
document paths contain only naming, spelling, link-title or whitespace changes;
the retained editorial checker explains their complete before/after content.
The other thirteen paths are covered below, including small editorial fixes
in the designated commit.

Complete raw diffs and 91 revision-pinned document snapshots are retained in
[the source directory](evidence/docs-delta-20261031/sources/). Nine current
Developer Knowledge API bodies match the corresponding target snapshots after
removing mirror frontmatter and outer whitespace. This verifies those API
bodies, without implying a live HTML comparison or a runtime recheck.

## Designated commit: all five document changes

| Document | Change | Consequence and boundary |
| --- | --- | --- |
| [Rank search results](https://docs.cloud.google.com/spanner/docs/full-text-search/ranked-search#scorer_versions) | When neither database nor query pins a scorer version, the default advances automatically on a later scorer release. | Pin `score_version` or the query's `SCORE` options `version` for comparisons across observations. The edit does not announce a new scorer version or prove that a rollout occurred. |
| [Search indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes) | A sort-order column participates in the underlying `docid` key; changing it rewrites the secondary-index row and associated token rows. The guidance suggests omitting frequently changing sort keys or using a coarse generated timestamp bucket. | This concerns search-index DDL `ORDER BY` and write cost. It does not explain GQL result randomization in issue #408. No benchmark is performed. |
| [Instance configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations) | Clarify regional optional-replica and dual-region replica footnotes/anchors, table introductions, and headings. Default-leader changes are consistently described as metadata operations using a long-running operation. | All 69 named configuration rows retain their region and replica data. Formatting, footnote destinations, Zurich spelling and parentheses are accounted for explicitly. |
| [Emulator](https://docs.cloud.google.com/spanner/docs/emulator) | Correct the heading from C to C#. | The existing C# connection-string instructions remain. This is not a new emulator capability or an ordering-contract change. |
| [Custom constraints](https://docs.cloud.google.com/spanner/docs/spanner-custom-constraints) | Change a generic resource example from `container.googleapis.com/NodePool` to `container.googleapis.com/Cluster`. | This corrects a GKE example; it adds no Spanner SQL or constraint capability. |

## Earlier capture in the interval

| Document group | Change | Existing evidence or follow-up |
| --- | --- | --- |
| Omni download, quickstart, setup, upgrade and console | Document **2026.r4-lts.1**, with **October 28, 2026** in the download release-date table; update container/archive examples. | The October 29 container comparison already tested this exact patch image. Documentation catching up is separate from image creation, registry observation and runtime execution. The download page does not list r5. |
| GoogleSQL DDL, CREATE QUEUE | Replace the statement that the payload column name is case-insensitive with explicit names **`Payload` or `payload`**. | The retained managed/container queue producers declare `Payload`. They do not establish acceptance of every capitalization. Add an isolated declaration-case matrix before claiming an engine discrepancy. |
| Pipe syntax, AGGREGATE | Explicitly disallow grouping by column ordinals or aliases from the AGGREGATE expression list. Grouping expressions use the input table; standalone grouping items can receive aliases. Clarify implicit aliases and output-column order. | Add ordinal, name-resolution and valid grouping-alias controls. This does not by itself require a parser AST change: syntax acceptance and semantic validity must stay separate. |
| Managed release notes | Add the **October 27** announcement of `ON DELETE SET NULL` for both SQL dialects. | The managed and paired-container notes already cover the behavior, including the older Omni LTS rejection. This is announcement coverage catching up to existing tests. |

The earlier configuration rewrite also corrects prose to mention Canada,
already present in the unchanged dual-region table, and the **asia2** exception:
five read-write replicas across three regions, with no witness. It adds
dual-region compute-placement guidance, describes multi-region automatic leader
failover, includes dual-region instance moves, and limits eligible default
leader regions to read-write regions. The final capture removes the remaining
schema-change wording for leader updates. The region inventory remains
**43 regional, five dual-region and 21 multi-region configurations** throughout
the interval; this audit does not date the introduction of those configurations
or test topology, failover, migration, latency or metadata-operation behavior.

## Runtime and issue implications

The [follow-up plan](evidence/docs-delta-20261031/follow-up-plan.json) marks every
new control **NOT_RUN**. The first priorities are the four payload declaration
spellings (`Payload`, `payload`, `PAYLOAD`, `PaYlOaD`) and pipe grouping controls
across the same managed/Omni/Emulator targets. Scorer pinning follows; comparing
write overhead requires a separate measurement design. A single execution
cannot demonstrate an automatic change on a future scorer release.

No newly reproduced upstream defect follows from this source audit. The
[invalid-renewal and stale-RECEIVE candidates](emulator-issue-candidates-20261031.md)
retain their earlier runtime evidence and contract boundaries. The queue DDL
wording concerns declaration names, not their lease or timestamp-bound
semantics. The GQL ordering issue [#408](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/408)
remains separate from physical search-index ordering and scorer revisions.

The earlier managed and container observations remain dated October 29. The
audit does not change catalog selectors, engine defaults, parser behavior,
dependencies, image pins or the public issue. It preserves all 1,798 preceding
retained artifacts and all 123 measured implementation/dependency files; the
[112 assertions](evidence/docs-delta-20261031/assertions.json) and
[validation receipt](evidence/docs-delta-20261031/validation.json) record the
source checks and repository documentation gates.

## Subsequent runtime follow-up

The [October 31 direct-SDK follow-up](documentation-runtime-and-memefish-audit-20261031.md) executes the previously
planned controls and audits current memefish plus public PRs. Its completed
samples supersede NOT_RUN only for their exact covered forms; a timed-out
managed fixture and final managed controls remain explicitly pending approval.
The source-only observation and its earlier evidence remain unchanged.

GQL column propagation was already added in the October 20 UTC mirror capture,
before this interval. The [official documentation status](documentation-runtime-and-memefish-audit-20261031.md#official-gql-documentation-status)
records the complete mode grammar, semantics and official reference.
