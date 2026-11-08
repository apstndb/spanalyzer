---
type: Research Note
title: "Emulator GQL ordering is already broken in the first announced graph release"
description: "Eight native arm64 release probes locate the availability boundary at 1.5.29/1.5.30 and reproduce the ordering discrepancy in seven graph-capable releases."
tags: [spanner, emulator, gql, ordering, release, bisection, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: results
    resource: evidence/gql-order-bisect-20261031/summary.json
    title: "Version classifications, result counts and evidence limits"
  - id: releases
    resource: evidence/gql-order-bisect-20261031/sources/releases.json
    title: "All 69 public GitHub release bodies retrieved for this investigation"
  - id: images
    resource: evidence/gql-order-bisect-20261031/sources/selected-images.json
    title: "Eight platform manifests selected from the official registry"
  - id: corpus
    resource: evidence/gql-order-bisect-20261031/sources/matrix.json
    title: "Identical FinGraph setup and eleven query cases"
  - id: historical-source
    resource: evidence/gql-order-bisect-20261031/sources/old-frontend-algebrizer-graph.cc
    title: "The graph-table ordering argument in the 1.5.30 frontend dependency"
  - id: assertions
    resource: evidence/gql-order-bisect-20261031/assertions.json
    title: "313 checks of values, platform identity, unavailable cases and cleanup"
  - id: validation
    resource: evidence/gql-order-bisect-20261031/validation.json
    title: "Documentation and evidence integrity checks"
---

# Emulator GQL ordering release bisection, October 31, 2026

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

The discrepancy **already reproduces in 1.5.30**, the release that announced
PropertyGraph support. It also reproduces in 1.5.31, 1.5.37, 1.5.44, 1.5.52,
1.5.58 and 1.5.59. The immediately preceding release, 1.5.29, rejects GRAPH
syntax while its ordinary SQL controls succeed. Consequently this search
reaches a **feature-availability boundary**, not a last-known-good GQL release.

This extends the [October 30 root-cause investigation](emulator-gql-ordering-root-cause-20261030.md).
It does not establish that every untested intervening tag has identical
behavior, or identify a finer source commit that introduced the bug.

## Method and identity

Execution was **October 30, 19:03–19:05 UTC / October 31, 04:03–04:05 JST**.
The refreshed [release catalogue](evidence/gql-order-bisect-20261031/sources/releases.json)
contains 69 complete release bodies. The
[1.5.30 release](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/releases/tag/v1.5.30),
published March 12, 2025, announces PropertyGraph support. Endpoints 1.5.30 and
1.5.59 and midpoint 1.5.44 all fail, so the investigation tests the predecessor
1.5.29, neighbor 1.5.31 and representative later releases instead of treating
unsupported GRAPH as a successful sorting result.

Every target is the official `gcr.io/cloud-spanner-emulator/emulator` image,
selected by **linux/arm64 platform digest**. Manifest indexes are retained and
hash-checked, and active container image IDs match the inspected local images.
The same unmodified compiled probe, Go client v1.91.0, fixture and query input
serve all eight releases. No architecture switch, source rebuild, or
diagnostic binary patch is involved in this comparison.

| Release | Published UTC date | Platform digest prefix | FinGraph ID ASC result |
| --- | --- | --- | --- |
| 1.5.29 | 2025-02-21 | `ef44e4116647` | InvalidArgument: unexpected GRAPH |
| 1.5.30 | 2025-03-12 | `e2582adc5efc` | `2,1,3` |
| 1.5.31 | 2025-04-04 | `d9650c946a2a` | `2,1,3` |
| 1.5.37 | 2025-06-25 | `6415f1d2d79f` | `2,1,3` |
| 1.5.44 | 2025-11-19 | `a63034eba579` | `2,1,3` |
| 1.5.52 | 2026-04-09 | `d33f3aab823d` | `2,1,3` |
| 1.5.58 | 2026-09-15 | `5f42f55ad412` | `2,1,3` |
| 1.5.59 | 2026-10-14 | `969af6a93499` | `2,1,3` |

Full digests, release URLs and platform manifests are in the
[identity record](evidence/gql-order-bisect-20261031/sources/selected-images.json).
No fleet, amd64, managed Spanner or Omni behavior is inferred.

## Identical reproduction and controls

The complete official FinGraph sample has Person IDs 1, 2 and 3. Every
graph-capable tested release executes:

```sql
GRAPH FinGraph
MATCH (p:Person)
RETURN p.id
ORDER BY p.id ASC;
```

Expected sequence: `1,2,3`. Actual sequence: `2,1,3`.
`SELECT id FROM Person ORDER BY id ASC` returns `1,2,3` on all eight targets.

Each of the seven graph-capable releases also reproduces the predicted
permutation for ID DESC, both two-row LIMIT directions, an output alias,
Person names, and a five-value FOR input. Thus **49 accepted, uniquely ordered
GQL cases** have incorrect output order, with the expected row membership.

Tables and seed data are independent of graph creation. On 1.5.29, the graph
DDL fails at PROPERTY and the independently attempted main GQL query fails
at GRAPH. Seven graph-dependent cases are explicitly NOT_RUN. This prevents
a fixture dependency from being mistaken for a passing query or an execution
result. Ordinary SQL remains healthy.

The outer SQL GRAPH_TABLE control uses a newer query form: 1.5.30 and 1.5.31
reject it at MATCH, while the five later targets execute and sort correctly.
This syntax difference is retained separately; it is not a failure of the
ordinary SQL control or the main standalone GQL reproducer.

## Historical source corroboration

The [1.5.30 WORKSPACE](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/0c1d49622079618b6d5f4b02765243fff2fa7b4c/WORKSPACE#L324)
pins GoogleSQL frontend revision `a516c6b26d183efc4f56293256bba92e243b7a61`
using its historical repository paths. Its
[graph-table algebrizer](https://github.com/google/googlesql/blob/a516c6b26d183efc4f56293256bba92e243b7a61/zetasql/reference_impl/algebrizer_graph.cc#L392)
already passes `graph_table_scan->is_ordered()` to projection algebrization.
The [empty-projection path](https://github.com/google/googlesql/blob/a516c6b26d183efc4f56293256bba92e243b7a61/zetasql/reference_impl/algebrizer.cc#L5411)
already overwrites the input operator's order-preserving flag, while
[Emulator enables scrambling](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/0c1d49622079618b6d5f4b02765243fff2fa7b4c/backend/query/query_engine.cc#L147).
The graph-table builder does not set an ordered flag. These source paths
corroborate the same mechanism identified in 1.5.59; only the latter received
the separate binary causal interventions in the earlier investigation.

The Emulator tag commits are `16643ae8d6308a54e546fcf7f02ddb3d24959240`
(1.5.29) and `0c1d49622079618b6d5f4b02765243fff2fa7b4c` (1.5.30).
Their pinned source snapshots and dependency patch are retained. This is a
release-level search, not a source-built git bisect: unsupported GQL would be
skipped rather than marked good in such a bisect.

## Retention and validation

The eight runs retain **112 planned steps / 88 planned query records / 160
root records**. Of the queries, 78 return OK, three are syntax rejections and
seven are blocked by the unsupported graph fixture. Passing process status
does not mean ordered-result conformance.

All eight owned databases end in confirmed NotFound and all eight runtime
containers are absent. Pre-existing containers are preserved. Unmodified
official images remain in the local reproducible cache; no prune is performed.

The [313 offline evidence checks](evidence/gql-order-bisect-20261031/assertions.json)
verify exact values, unavailable cases, matching architecture/image IDs,
cleanup, unchanged probe bytes, all 1,653 preceding research artifacts and
123 measured implementation/dependency files. The
[validation receipt](evidence/gql-order-bisect-20261031/validation.json)
records documentation gates. No repository implementation, dependency or
selector is changed; the local issue draft gains the explicitly tested version
list. No upstream post, commit, push or release is performed.
