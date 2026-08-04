---
okf_version: "0.2"
---

# spanalyzer Knowledge

This directory is an [Open Knowledge Format (OKF) v0.2](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)
bundle embedded in the spanalyzer repository. It provides curated discovery
metadata and relationships without replacing the repository's canonical code,
schemas, tests, package documentation, or command documentation.

Each concept has one writable body. Repository indexes may link to a concept,
but must not maintain a second copy of its findings.

## Producer types

- `Observation`: non-normative evidence tied to identified sources and a
  specific comparison or verification scope.

Other concept types are deferred until a concrete routing need is established.

## Observations

- [GoogleSQL hint placement inventory](observations/googlesql-hint-placement.md)
  - Normalized generic hint placements in a pinned upstream GoogleSQL grammar,
    separated from frontend, runtime-syntax, and plan-effect evidence.
