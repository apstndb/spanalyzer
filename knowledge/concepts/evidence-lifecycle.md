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
3. The SPANNER_SYS producer manifest combines structural declarations with
   explicit required-target observations and fails closed on disagreement.
   INFORMATION_SCHEMA instead retains independent target captures and selects
   one managed observation explicitly; it has no cross-target producer
   manifest.
4. Analyzer manifests project their reviewed policy. SPANNER_SYS uses a pinned
   common surface, while INFORMATION_SCHEMA uses the explicit managed-primary
   selector. Their source identity remains reviewable after producer code
   moves.
5. Consumer tests and commands reproduce committed bytes from the retained
   producer and independently validate the legacy import mapping.

These layers intentionally preserve target distinctions. Managed, Omni, and
Emulator results can agree, differ, or be unavailable without being collapsed
into a single `supported` flag.

## Observation identity and projection selection

Managed Spanner is a rolling service. A retained managed capture is evidence
for one database at its read-only transaction timestamp; it is not a fleet-wide
claim and does not remain current merely because its bytes are still tracked.
The database locator is never retained, and the public capture does not assign
a reusable pseudonymous database identifier.

Spanner Omni and the Emulator are container releases that can be rerun after
the original observation date. Their primary identity is the platform-specific
OCI manifest digest and the resolved `os/arch[/variant]`; the mutable tag is
retained for readability and the timestamp is secondary provenance. Two files
with the same tag but different digests are distinct observations.

INFORMATION_SCHEMA captures retain advertised column tuples and bounded
queryability outcomes for rolling columns. Their source and invocation hashes
identify the reviewed capture inputs but are not execution attestations. Raw
captures remain producer-owned JSON assets rather than OKF concepts.

The analyzer never selects a capture by modification time or a `latest` name.
[`information_schema_projection_source.json`](../../information_schema_projection_source.json)
names one exact managed capture and its whole-file hash. The generated
[`information_schema_manifest.json`](../../information_schema_manifest.json)
remains the sole analyzer projection; Omni and Emulator observations are
comparison evidence and do not implicitly intersect or demote the
managed-primary surface.

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
