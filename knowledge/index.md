---
okf_version: "0.2"
---

# spanalyzer Knowledge

This directory is an [Open Knowledge Format (OKF) v0.2](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/SPEC.md)
bundle embedded in the spanalyzer repository. It provides curated discovery
metadata and relationships without replacing the repository's canonical code,
schemas, tests, package documentation, or command documentation.

The specification link is commit-pinned for reproducibility. The former copy
under `GoogleCloudPlatform/knowledge-catalog/okf` is a frozen snapshot whose
own README directs consumers to the canonical `open-knowledge-format`
repository; do not use that copy as the current authority.

Each concept has one writable body. Repository indexes may link to a concept,
but must not maintain a second copy of its findings.

## Browse by purpose

- [Concepts](concepts/index.md) - cross-cutting explanations of the repository's
  architecture, documentation authority, query generation, and plan tooling.
- [Observations](observations/index.md) - curated, non-normative findings tied
  to identified sources and an explicit verification scope.
- [Research](research/index.md) - canonical authoring guidance for new
  environment-scoped research notes and migration from the legacy tree.
- [References](references/index.md) - role-based inventories of canonical
  Markdown documents and selected knowledge-bearing repository assets that
  remain outside this bundle.

## Concept types

- `Architecture`: the purpose and boundaries of a component or subsystem.
- `Documentation Guide`: how to select and interpret repository documentation.
- `Observation`: non-normative evidence tied to identified sources and a
  specific comparison or verification scope.
- `Research Note`: environment-scoped probe results, design background, or
  candidate ideas with explicit provenance and lifecycle metadata.
- `Reference`: discovery metadata for canonical material that remains at its
  repository-native path.

The bundle is intentionally a navigation and synthesis layer. Current CLI
behavior remains defined by code, schemas, tests, package documentation, and
command documentation; research notes remain environment-scoped evidence.
