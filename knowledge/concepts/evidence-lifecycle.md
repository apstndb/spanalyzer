---
type: Architecture
title: Cross-Environment Evidence Lifecycle
description: How spanalyzer retains producers and separates target observations from portable analyzer projections.
tags: [spanalyzer, evidence, spanner, spanner-omni, emulator, provenance]
status: draft
sources:
  - id: survey-import-provenance
    resource: ../../survey/import-provenance.json
    title: Survey import provenance
  - id: information-schema-manifest
    resource: ../../information_schema_manifest.json
    title: INFORMATION_SCHEMA projection manifest
  - id: spanner-sys-manifest
    resource: ../../spanner_sys_manifest.json
    title: SPANNER_SYS projection manifest
  - id: survey-agent-guide
    resource: ../../survey/AGENTS.md
    title: Survey producer guidance
---

# Cross-Environment Evidence Lifecycle

spanalyzer keeps schema reconstruction and metadata probes in the nested
[`survey`](../../survey/) module. This puts managed Spanner, Spanner Omni, and
Emulator producers beside the analyzer consumers without treating target
agreement as a universal service contract.

## Layers

1. Survey code discovers or reconstructs target metadata. Environment-specific
   tests retain which target, version, and date were actually observed.
2. Redacted captures retain only the metadata needed for a claim. They exclude
   project, instance, database, endpoint, credential, and local runner state.
3. Producer manifests combine structural declarations with explicit target
   observations and fail closed when required evidence is missing or disagrees.
4. Analyzer manifests project a pinned common surface. Their source identity
   remains immutable even after the producer code moves.
5. Consumer tests and commands reproduce committed bytes from the retained
   producer and independently validate the legacy import mapping.

These layers intentionally preserve target distinctions. Managed, Omni, and
Emulator results can agree, differ, or be unavailable without being collapsed
into a single `supported` flag.

## Legacy repository disposition

The survey began as an unpublished local Git repository. Commit
`91908d001349f844aac070cc6518119c0e3c36c0` was imported as one exact tree into
spanalyzer commit `cd1de13e5d60af39033fa8611b5c4df6b5a3f6ea`; both sides of
that mapping have tree ID
`34d63cf89aaf885cbfd8069e91c4ead707b048c8`.[^survey-import-provenance]

The 27-commit legacy history, ignored agent/runtime directories, local managed
database locator, and an untracked memefish feedback scratch file are not part
of the import. The scratch file is retired because its two actionable upstream
issues are closed. No standalone GitHub repository is created. The checked
mapping preserves reviewable origin without requiring the old checkout or
publishing local-only history.

## Deletion criterion

The old checkout is disposable only when all of these remain true in a clean
clone:

- the immutable import mapping passes `survey-import-check`;
- the nested survey module resolves and tests with `GOWORK=off`;
- INFORMATION_SCHEMA and SPANNER_SYS manifests reproduce from `survey/`;
- workspace, OKF, asset-inventory, and repository validation gates pass; and
- remote `main` contains the reviewed commits.

This is a reproducibility criterion, not authorization to delete the checkout.

[^survey-import-provenance]: [`survey/import-provenance.json`](../../survey/import-provenance.json)
    is strict machine-readable provenance. It validates the initial imported
    subtree rather than pretending later retained development has the same tree.
