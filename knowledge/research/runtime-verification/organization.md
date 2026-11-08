---
type: Research Note
title: Runtime Verification History Organization
description: Engine-specific chronological navigation, observed change boundaries, and a repeatable append workflow without duplicating research or moving evidence.
tags: [documentation, runtime, history, evidence, spanner, spanner-omni, emulator]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: lifecycle
    resource: ../../concepts/evidence-lifecycle.md
    title: Cross-environment evidence lifecycle
  - id: research-policy
    resource: ../index.md
    title: Research authoring and single-body policy
  - id: rolling-example
    resource: managed-information-schema-is-inherited-rollout-20260916.md
    title: Interleaved old and new managed surfaces within one observation window
  - id: paired-example
    resource: spanner-omni-2026.r3-beta-verification.md
    title: Version-paired probes and correction of a prior queue interpretation
  - id: historical-comparison
    resource: cross-engine-comparison-20261005.md
    title: Multiple engines and platforms in one retained comparison
---

# Runtime verification history organization

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Use one chronological index per engine and keep each research report and raw
capture at its existing canonical path. This navigation structure is applied
now; moving historical reports or introducing a machine-readable event format
is a separate future decision.

## Reading paths

| Question | Entry point | What it owns |
| --- | --- | --- |
| What differs in the latest selected samples? | [Runtime verification](index.md), selected comparison | One centrally maintained comparison link and its observation date |
| When did we observe a change on an engine? | [Managed](managed/index.md), [Omni](omni/index.md), [Emulator](emulator/index.md) | Oldest-to-newest observation rows with result kind and evidence links |
| What exactly was executed and concluded? | Linked Research Note | Case definitions, runtime scope, interpretation and limitations |
| What did the service return? | Linked producer capture or research evidence | Immutable observed values, errors, plans and execution receipts |
| What does the analyzer currently project? | Catalog Observations and selectors | Deliberately selected implementation inputs, independent of observation recency |

Current layout:

```text
runtime-verification/
  index.md                  # Entry point and selected comparison
  organization.md           # History conventions and append workflow
  managed/index.md          # Managed observation timeline
  omni/index.md             # Omni observation timeline
  emulator/index.md         # Emulator observation timeline
  <existing-report>.md      # One body, including reports about several engines
  evidence/<run>/           # Existing research evidence paths
survey/.../evidence/        # Producer-owned captures remain here
```

The engine indexes are navigation, not additional Research Notes. A shared run
appears in each relevant engine index with that engine's bounded result and a
link to the same report. Do not copy SQL matrices, release-note bodies, or raw
captures into the indexes. Existing report and evidence paths remain usable by
hash manifests, source references, and external links.

## Dates and change boundaries

Sort by **observation time**, oldest first. Prefer an exact UTC timestamp from
the capture or receipt; include JST as a reading aid when a report's filename
uses the local date. If only a day is recorded, say so and do not invent a time
or timezone. Same-day rows without timestamps have no claimed internal order.
Historical consolidation belongs at the original observation date, with the
later compilation date noted separately.

Correction rows are the exception: place them at the **correction date**, label
that date explicitly, and include the original observation date. This keeps the
history of interpretation visible without inventing another runtime execution.
Keep source-only publication/retrieval dates in a separate context section.

Keep these dates distinct:

- **Observed at:** when the identified case was run against that target.
- **Release published at:** the vendor's announcement or image publication;
  this is source context, not a measured introduction date.
- **Recorded or corrected at:** when a report was written, consolidated, or
  its interpretation changed. An interpretive correction is not a server change.

For managed Spanner, a service build is generally unavailable. Retain the
sample timestamp and redacted target scope from the original evidence without
adding database identifiers. Link both sides of a change comparison, and state
what is comparable. "First observed here" never means fleet rollout started
then. Keep both old and new observations when they overlap or flap.

For example, the managed `IS_INHERITED` investigation first records the new
surface at `2026-09-16T16:56:07.707925Z`, then records the old surface again at
`17:06:55.873321Z`. A single transition timestamp, or an interval whose end
precedes its start, would misrepresent the evidence. Later 310-column samples
are confirmations at their own times, not proof that all replicas converged.

For Omni and Emulator, retain the tag, platform manifest digest, and resolved
platform through the linked receipt. State both versions for a paired change.
Running an old immutable image today produces a new observation of that old
image; it does not move the feature's introduction date to today. A changed
architecture, producer, SQL case, optimizer hint, or fixture must remain visible
and may prevent attributing a difference to the engine version alone.

## Row kinds

| Kind | Use when | Required distinction |
| --- | --- | --- |
| Baseline | A retained observation supplies the comparison's earlier state | Earliest indexed baseline, not the engine's original behavior |
| Change | Comparable before/after observations differ | Name the changed case, both states, and both evidence references; qualify any platform/producer difference |
| First coverage | A case, platform, or metadata field was checked for the first time in this history | There is no earlier comparable result proving a change |
| Recheck | An already covered case is sampled again with the same material outcome | Name the scope and prior observation; do not imply untested cases stayed unchanged |
| Correction | A prior result was misread or a probe defect changed its interpretation | Preserve the prior raw result; record old/new interpretation and the correction date |
| Source only | A document, release announcement, or source change was inspected without a matching runtime probe | Do not promote it to observed runtime support |

Split a run into multiple rows if it contains both a change and unchanged or
newly covered cases. Failed preparation, skipped execution, syntax rejection,
later semantic rejection, and successful execution are different outcomes;
the row kind does not replace those outcome details.

Examples: r2.1-to-r3 automatic parameterization is a paired **Change**;
queues working on both pinned images in the same investigation is **First
coverage** of that pair, not "queues introduced in r3." Discovering permissive
graph options on Emulator 1.5.58 arm64 is **First coverage**, without an earlier
identical-case baseline proving when that behavior began.

## Minimum row and evidence information

The compact indexes use four columns: observation date/runtime, kind, bounded
finding or change, and report/evidence links. The linked evidence must supply:

| Information | Recording rule |
| --- | --- |
| Engine, dialect, case or feature | Name the case precisely enough to distinguish DDL, execution, metadata and PLAN checks |
| Observation time and runtime identity | Preserve available precision, timezone, digest/platform or managed sampling scope |
| Earlier observation | Link it for Change/Recheck; use "no comparable baseline" for First coverage |
| Result and delta | Before/after outcome; keep error layer, advertisement/queryability, and no-row/error distinctions |
| Producer and comparability | SQL/fixture, source/build identity, SDK or optimizer changes that affect interpretation |
| Evidence and qualification | Report plus raw capture/receipt; explain unknown rollout boundaries and incomplete coverage |

For stable case names, reuse existing probe labels such as
`active-batch-operations` or `control-array-default`. Similar feature titles
alone do not establish that two probes are comparable. Do not assign synthetic
managed deployment identities to make old runs appear more comparable.

## Appending the next verification

1. Retain the new raw capture/receipt with its real time and exact runtime scope.
   Preserve previous captures and hashes; update the artifact manifest for new
   research evidence. Producer-owned captures keep their producer format.
2. Compare against a named earlier observation of the same case. Classify each
   material result as Change, First coverage, Recheck, or Correction. Keep
   publication/source-only information separate from runtime results.
3. Add a short row to every engine index involved. An unchanged run still gets
   a dated **Recheck** row linked to its receipt, even if it needs no new report.
   Group probes from one run only when their comparison and outcome agree.
4. Write or extend a Research Note only when new findings or interpretation
   need explanation. Preserve dated historical conclusions; add a dated
   correction or follow-up with links instead of overwriting the previous
   observed state. One new run does not require one new Markdown report.
5. Update the root's selected-comparison link only when a comparison actually
   includes new evidence. An engine-only rerun does not refresh every cell of
   a cross-engine matrix. The comparison must retain per-engine evidence dates.
6. For new concept/index files, update the publication inventory and directory
   links. Run from `tools`:

   ```sh
   go run ./okf-check --repo-root .. --gate all
   go run ./markdown-language-check --repo-root ..
   ```

An index addition does not change authoritative runtime pins, parser support,
or analyzer projection selectors. Those are separate implementation decisions.

Use this row shape; replace every placeholder with a supported value:

```markdown
| Observed at / runtime | Kind | Bounded result or change | Report and evidence |
| --- | --- | --- | --- |
| <time and zone; image/platform if applicable> | <kind> | <case: earlier result → observed result; limitation> | <relative links to the earlier result and the new report or receipt> |
```

For First coverage, state that no comparable baseline is retained. For an
unchanged Recheck, link both the earlier scope and the new receipt; a newly
queried feature must not be hidden inside a general "unchanged" row.

## Alternatives and later scaling

| Option | Assessment |
| --- | --- |
| One mixed date-sorted report list | Simple inventory, but a reader must reconstruct each engine's state across unrelated reports |
| Move every report into an engine directory | Useful only for single-engine reports; shared reports need one owner and many inbound paths would change |
| Engine timelines linking existing reports | Selected now: low migration cost, readable history, shared evidence remains single-owned |
| Machine-readable event ledger with generated views | Possible later when manual timelines need filtering or become inconsistent; requires a versioned schema, validator and deterministic generation |

If an event ledger becomes necessary, make it the only writable source for
generated timeline rows. Candidate fields are `event_id`, `engine`, `case`,
`observed_at` with precision/timezone, `runtime`, `kind`, `previous_event`,
`outcome`, `evidence`, and `qualification`; release and correction dates remain
separate. These are design candidates, not an adopted public contract. Do not
start a parallel JSON ledger while maintaining the same events independently
in Markdown.

The initial indexes cover the Research Notes already under runtime verification
and the baseline captures they reference. They do not claim to have backfilled
every experiment in the frozen legacy research tree. Future backfill should
identify its source and original observation date explicitly.
