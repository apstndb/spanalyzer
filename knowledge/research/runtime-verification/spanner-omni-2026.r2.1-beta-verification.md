---
type: Research Note
title: "Spanner Omni 2026.r2.1-beta retained runtime verification"
description: "Canonical record of retained August 2026 Omni metadata captures and clearly identified historical integration results, consolidated on 2026-09-05."
tags: [spanner, spanner-omni, runtime, metadata, provenance]
status: draft
sources:
  - id: infoschema
    resource: ../../../survey/infoschem/evidence/omni/ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741/linux-arm64-f8b0ea3092e7.json
    title: "Omni linux/arm64 INFORMATION_SCHEMA capture from 2026-08-25"
  - id: spannersys
    resource: ../../../survey/spannersys/evidence/omni-2026.r2.1-beta.json
    title: "Legacy Omni SPANNER_SYS capture from 2026-08-25"
  - id: survey-handoff
    resource: ../../../survey/HANDOFF.md
    title: "Dated survey validation and behavioral observations"
  - id: recovered
    resource: evidence/spanner-omni-2026.r2.1-beta/recovered-validation-summary.json
    title: "Historical integration summary recovered from coordination records"
---

# Spanner Omni 2026.r2.1-beta retained runtime verification

This note consolidates existing verification on 2026-09-05. It does not claim
that Omni was freshly rerun during the documentation migration or that this is
its latest release. Retained captures, historical run reports and executable
reproducers have different evidence strength and are identified below.

## Retained captures

| Observation | Result | Runtime identity and limits |
| --- | --- | --- |
| INFORMATION_SCHEMA, 2026-08-25 12:06:26.330524 UTC | 308 columns; `INDEXES.SEARCH_UNNEST` and `INDEX_COLUMNS.EXPRESSION` are queryable | `2026.r2.1-beta`, resolved `linux/arm64`, manifest `sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741` |
| SPANNER_SYS, 2026-08-25 | 539 column tuples across 50 tables | Legacy v0alpha1 capture records the tag and date; it does not record a platform manifest and cannot be retrospectively bound to the INFORMATION_SCHEMA image identity |

The INFORMATION_SCHEMA surface SHA-256 is
`f8b0ea3092e7601f260d6d9cb59002dcd0cf4fba6283df8a7c04c004e6a664ee`.
Its tuples match the explicitly retained managed capture at that time.[^infoschema]
This demonstrates those captured metadata tuples and the two bounded
queryability checks; it does not prove all managed SQL semantics on Omni.

The SPANNER_SYS content SHA-256 is
`2f1acd9eeff25dcccde68758984e35698f41338d3b4674b7fc7b25d36536fd44`.
The producer also records the exact metadata query.[^spannersys] The
[SPANNER_SYS observation](../../observations/spanner-sys-live-primary-catalog.md)
explains the legacy provenance boundary and the later v0alpha2 producer.
Neither this note nor migration changes the analyzer's selected projections.

## Historical integration results

The dated survey handoff reports an uncached full survey run and Omni drift
verification on 2026-08-25, including 50 SPANNER_SYS tables, 539 columns and
three decoded sample rows. It also records these observed behaviors:[^survey-handoff]

- UUID, `NEW_UUID()` and UUID nullability worked on the tested targets.
- Omni and managed Spanner generated a hidden identity `rowid` when the primary
  key clause was omitted; explicit `PRIMARY KEY ()` kept singleton semantics.
  The same handoff's Emulator 1.5.56 rejection is a historical boundary that
  was subsequently changed by [Emulator 1.5.57](cloud-spanner-emulator-1.5.57-verification.md).
- Custom dictionaries and table options were observed on managed Spanner and
  Omni. `columnar_policy` could occur in index DDL without a corresponding
  `INDEX_OPTIONS` observation; DDL acceptance and metadata visibility differed.
- Nullable locality-group options were omitted during schema reconstruction.
  Malformed emulator metadata was recorded separately from Omni results.

The later improvement-train handoff reports a passing `verify-containers`
baseline for Emulator 1.5.56 and Omni `2026.r2.1-beta`, followed by relevant
plan-shape and query-generator probes. Its recovered summary records the
Omni arm64 image digest and the same 308/539 metadata counts.[^recovered]
The handoff also reports two focused Omni outcomes from that train:

- Property Graph result metadata gave `STRING` for
  `CONCAT(FirstName, ' ', LastName)` and `INT64` for `LENGTH(FirstName)`.
  The [historical integration case](evidence/spanner-omni-2026.r2.1-beta/graph_derived_property_omni_test.go.txt)
  defines both properties and queries them through `GRAPH MyGraph`.
- The query-generator container gate passed with the pinned arm64 Omni
  identity. Its report distinguishes a command-owned pinned runtime from
  attached runtimes with manual identity assertions or unverified identity;
  those states are not interchangeable.

These are **reported historical outcomes**: the original full terminal logs
and individual timestamps were not retained in this canonical evidence set.
They must not be presented as a fresh exact-snapshot execution receipt.

## Query-plan comparison

The [2026-08-24 managed/Omni PLAN comparison](spanner-omni-2026.r2.1-beta-managed-plan-comparison-20260824.md)
retains the substantive SQL and optimizer findings: all 15 documented pipe
operators, optimizer version 9, distributed-apply batching, LIMIT-dependent
plan choices, and a v8/v9 `DELETE ... THEN RETURN` shape difference.
Its recorded digest starts `b0f5ca38`, and its platform was not recorded.
That observation remains separate from the later `ed31d9ee` arm64 capture.

## Reproduction and interpretation

Current producer entry points and target pins remain in
[survey/AGENTS.md](../../../survey/AGENTS.md),
[plan-shape tool guidance](../../../tools/spanner-query-plan-shape/README.md), and
[runtime_targets.json](../../../runtime_targets.json).
The retained query-plan cases and assertions are linked by the comparison note.
A future run should record its exact source snapshot, selected cases, image
digest/platform, exit outcomes and redacted captures here. Executable tests
alone are not evidence that a named run occurred.

No standalone server package, VM deployment, commercial license, backup
migration, or broader feature parity was verified by this consolidation.
Managed destination identifiers and local connection configuration are not
part of the retained research.

[^infoschema]: [Digest- and platform-bound Omni capture](../../../survey/infoschem/evidence/omni/ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741/linux-arm64-f8b0ea3092e7.json)
    and the [managed-primary observation](../../observations/information-schema-managed-primary-catalog.md).
[^spannersys]: [Legacy SPANNER_SYS capture](../../../survey/spannersys/evidence/omni-2026.r2.1-beta.json).
[^survey-handoff]: [Survey handoff](../../../survey/HANDOFF.md), dated
    2026-08-25; retained as the source of historical run statements.
[^recovered]: [Recovered validation summary](evidence/spanner-omni-2026.r2.1-beta/recovered-validation-summary.json)
    records source hashes and the limits of reconstructing prior run evidence.
