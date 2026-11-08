---
type: Research Note
title: Repository hardening verification, November 8
description: Verification of seven review fixes across generated Go, DDL reconstruction, runtime ownership and plan contracts.
tags: [verification, spanner-omni, emulator, codegen, plancontract]
status: draft
sources:
  - id: runner
    resource: evidence/repo-hardening-20261108/omni-runner.json
    title: Three successful suites against the pinned Omni LTS2 image
  - id: survey
    resource: evidence/repo-hardening-20261108/survey-full.log
    title: Independent survey tests including pinned Emulator reapplication
  - id: closure
    resource: evidence/repo-hardening-20261108/standalone-builds.log
    title: All five modules build with GOWORK disabled
  - id: schema
    resource: evidence/repo-hardening-20261108/historical-schema-validation.json
    title: Historical full and summary artifact compatibility checks
  - id: source
    resource: evidence/repo-hardening-20261108/source-boundary.json
    title: Dirty runtime producer boundary and post-run source hashes
  - id: evaluator-schema
    resource: evidence/repo-hardening-20261108/v2-schema-validation.json
    title: Positive and negative schema checks using local evaluator v2 output
  - id: emulator-gap
    resource: evidence/repo-hardening-20261108/emulator-routines-gap.json
    title: UDF execution versus missing live routine metadata in Emulator
  - id: design
    resource: ../../../cmd/spanner-query-gen/DESIGN.md
    title: Canonical design choices and rejected alternatives
---

# Repository hardening verification

The review baseline was `f266d62ebeb7fadb27ac1f5aaacd86cbea835cf1`.
The following fixes were implemented in priority order. Design choices and
alternatives are maintained in the linked native design document.

| Priority | Finding | Result and validation boundary |
| --- | --- | --- |
| P1 | Reconstructed DDL placed sequences after table defaults and functions after dependent views | Known AST dependencies now determine a stable order. Both AST and live-metadata exports were reapplied successfully to fresh Omni databases, including a sequence default, a named UDF/view and a view chain. Cycles, CTE scope, range-variable paths and partial exports have focused tests. |
| P2 | Optional parameters such as `user_id` and `userId` generated duplicate Go fields | Deterministic pre-generation rejection replaces uncompilable output; generation and report planning both diagnose the collision. Existing generated-builder compilation checks remain active. |
| P2 | Quotes, backslashes, controls and backticks corrupted generated struct tags | Generated packages compile and preserve exact tags. A local Spanner SDK row decoder and the generated BigQuery loader both decode the original column names. This is SDK/local execution evidence, not a new managed or BigQuery service observation. |
| P2 | Attached-runtime database retention was implicit | Retention remains the default to defer deletion cost. `--drop-created-databases` is explicit; an Omni regression verifies both policies and checks that the endpoint seed database survives. The command never evicts previously retained databases. Cleanup errors remain in the report. |
| P2 | Workspace replacements hid obsolete module dependencies | CLI and tools reference immutable root/library pseudo-versions from `ec5bb1879b1b`. All five modules build with `GOWORK=off`; CI and pre-push now include this gate. No stable release or tag was created. |
| P2 | External config locations prevented cold-start pin discovery | The CLI bundles a generated, byte-checked registry copy and supports `--runtime-targets`. Tests resolve exact arm64/amd64 pins from an external directory and reject malformed overrides before runtime construction. |
| P2 | One runtime CEL error discarded the entire artifact | Evaluator `v2` reports `not_evaluated/evaluation_error` per contract, preserves neighboring results and keeps `--check` failing. Runtime errors, compilation errors, full/summary serialization and historical schema compatibility are checked separately. |

The updated schemas passed 112 historical full/summary checks and 10 additional
positive/negative checks adapting local evaluator `v2` output. Schema validation
does not attest service execution.

## Runtime evidence

The shared runner used the configured `linux/arm64` Omni `2026.r4-lts.2`
platform pin `sha256:b607b10bee3c1903f0a7f88f99bece67cb0dd9d9a028aee64226fc311bd78dae`.
Plan-shape, query-generator and schema-reconstruction suites all passed, and
the owned runtime closed successfully. The source was a dirty implementation
tree on the library commit, as recorded by the runner. Post-run hashes identify
the remaining implementation files; this receipt does not claim execution of
a clean final merge commit or independent server-version discovery.

Emulator `1.5.59` used the configured arm64 pin
`sha256:969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8`.
The independent survey suite passed, including reapplication of sequence and
view dependencies from both AST and live metadata.

An earlier extended Emulator fixture executed a named SQL UDF and view, and its
AST export reapplied successfully. Live loading returned zero routine rows,
however, so that export omitted the function and failed to recreate its view.
This is a separate metadata gap: dependency sorting cannot recover an object
absent from the input. The permanent Emulator live regression therefore covers
recoverable sequence/view dependencies; Omni additionally covers live UDF
metadata. No upstream issue was posted by this task.

No new managed Spanner databases were created for these fixes. This work does
not extend the managed or Omni r5 observations in earlier notes.
