---
type: Research Note
title: Compact PLAN summaries on Spanner Omni LTS, November 8
description: Fixed-image comparison of full and summary reports, optimizer warning separation, and actual JSON/YAML schema validation.
tags: [spanner-omni, query-plan, plancontract, schemas, verification]
status: draft
sources:
  - id: receipt
    resource: evidence/plan-summary-lts-20261108/receipt.json
    title: Producer boundary, comparison assertions and retained transformations
  - id: runner
    resource: evidence/plan-summary-lts-20261108/runner.json
    title: Successful clean-source Omni integration suites and exact image pin
  - id: full
    resource: evidence/plan-summary-lts-20261108/full.json
    title: Full report from the public integration fixture
  - id: summary
    resource: evidence/plan-summary-lts-20261108/summary.json
    title: Compact summary from the same fixture and runtime
  - id: schema-validation
    resource: evidence/plan-summary-lts-20261108/schema-validation.json
    title: Artifact validation and projection checks with schema hashes
  - id: schema-defect
    resource: evidence/plan-summary-lts-20261108/schema-defect.json
    title: Reproduced rejection of missing range-key metadata by the earlier schema
  - id: validator
    resource: evidence/plan-summary-lts-20261108/validate.py
    title: Reusable artifact checks, including negative controls
  - id: design
    resource: ../../../cmd/spanner-query-gen/DESIGN.md
    title: Canonical summary design and alternatives
---

# Compact PLAN summaries on Omni LTS

Observed on November 8, 2026 JST, using the platform-specific Spanner Omni
`2026.r4-lts.2` image pin
`sha256:b607b10bee3c1903f0a7f88f99bece67cb0dd9d9a028aee64226fc311bd78dae`
for `linux/arm64`. This is the runner's configured pin, not an independent
server-version observation. The retained runner reports clean producer commit
`4fe4ff1ebff7c7f30e68c8f1ebb4d0936df3d17e` and successful plan-shape and
query-generator integration suites. The test source is
`cmd/spanner-query-gen/plan_report_feedback_omni_test.go` at that commit.

The first startup attempt failed before Docker provider creation; selecting
the existing Colima socket resolved it. Only the successful run's log is
retained here. Its host socket URL is replaced as described in the receipt;
all other bytes are preserved. The runner closed its runtime, and a subsequent
[cleanup check](evidence/plan-summary-lts-20261108/cleanup.json) found both
current-task container identities absent.

## Full and summary comparison

The public fixture expands into ten targets. Both modes produce nine PLAN
successes and one deliberate static error from a STRING LIMIT parameter.
Target accounting, canonical IDs, plan digests, access paths and live contract
evaluations match. The RangeSeek membership contract passes in both modes.
Summary collection produces no rendered plan strings. Optional variants,
structured parameters, DML acquisition and the generated DELETE helper remain
covered. A separate sentinel row retains its value after the complete
collection, proving that these DML PLAN requests do not execute the writes.

| JSON artifact | Bytes |
| --- | ---: |
| Full | 416,126 |
| Summary | 17,667 |

This fixture's summary is 95.75% smaller. These are output byte counts, not
runtime speed or memory measurements. The feedback author's 82-config,
153-target private corpus and approximately 112 MB / 150 KB figures were not
reproduced. Full normalized plans still exist in memory for contract evaluation.

The summary retains identity, digests, variant choices, access paths, errors,
environment evidence and contract results. It omits SQL text, rendered plans
and operator graphs. Its distinct `v1alpha-plan-summary-v1` version and schema
prevent confusion with a full replayable report. Serialized evaluations set
`stability.replayable_from_report: false`; their live truth and `--check`
accounting are unchanged. Expressions and errors can contain SQL fragments,
so this projection does not provide redaction.

## Pinning and column semantics

Optimizer version 9 is requested through the client query options. Effective
optimizer and statistics-package values are not separately observed. With no
statistics-package pin, the only pinning warning is
`statistics_package_not_pinned`; neither optimizer-version warning remains.
The borrowed test endpoint has no independent backend identity assertion, so
its report also retains the separate backend-identity warning. The outer
runner receipt supplies the configured runtime pin.

The strict `--require-optimizer-pinning` gate continues to require both fixed
values. Unit controls cover all four version/statistics pin combinations,
moving aliases, and writing a summary before a strict-gate failure.

Seek and residual columns remain lexicographically sorted, duplicate-free
sets. Declared key order remains in `key_columns`. Prefix checks use membership
against that ordered key list; equality between the two lists is unsuitable.

## Schema correction and evidence limits

Actual validation found an earlier schema defect in both full and summary
reports: an available Scan can omit range-key metadata, which serializes as
`seekable_key_size: ""`, but the schema incorrectly required a non-empty string.
The correction permits empty values for raw scan target/type, unresolved base
table, range-key metadata and condition expressions while retaining required
field presence. Missing evidence stays distinct from a known zero value.

The retained validator checks all four JSON/YAML artifacts against Draft
2020-12, JSON/YAML semantic equality, projection parity, unknown metadata and
negative controls for incompatible artifacts, missing access paths, rendered
plans and incorrect replay claims. Its 112 assertions pass. The schema hashes
identify the follow-up schema-only correction; runtime artifacts retain their
original producer commit. `config.json` and `contracts.json` are serialized
in-memory test fixtures rather than files read through the CLI. No new managed
Spanner, Emulator, PROFILE, latency or scan-row observations are claimed.
