---
type: Reference
title: Repository Markdown Inventory
description: Every tracked Markdown document in spanalyzer, grouped by authority and purpose.
tags: [documentation, repository, inventory, okf]
status: draft
sources:
  - id: repository-readme
    resource: ../../README.md
    title: spanalyzer README
  - id: research-index
    resource: ../../research/README.md
    title: spanalyzer research index
---

# Repository Markdown Inventory

This inventory makes every tracked Markdown document discoverable through the
OKF bundle without moving or duplicating its canonical body. Ignored local
review notes and untracked work are intentionally outside the inventory.

See [Documentation authority](../concepts/documentation-authority.md) for how
the groups below relate when their contents appear to disagree.

## Repository entry points and maintenance

- [README](../../README.md) - project positioning, module map, analyzer usage,
  public components, and limitations.
- [AGENTS](../../AGENTS.md) - repository rules for coding agents, including
  module boundaries and validation commands.
- [TODO](../../TODO.md) - unresolved work only; not a current behavior spec.

## Query generator

- [README](../../cmd/spanner-query-gen/README.md) - current CLI and
  configuration UX.
- [DESIGN](../../cmd/spanner-query-gen/DESIGN.md) - intended architecture and
  roadmap, including unimplemented work.
- [IMPLEMENTATION_STATUS](../../cmd/spanner-query-gen/IMPLEMENTATION_STATUS.md)
  - explicit implementation/design drift and deferrals.
- [PLAN_CONTRACTS](../../cmd/spanner-query-gen/PLAN_CONTRACTS.md) - plan
  contract format, semantics, and examples.

## Plan libraries and developer tools

- [planvocab README](../../plancontract/planvocab/README.md) - observational
  catalog, drift checker, generation, and provenance rules.
- [spanner-query-plan-shape README](../../tools/spanner-query-plan-shape/README.md)
  - developer probe usage, case families, and runtime requirements.

## Legacy research indexes and history

- [Research index](../../research/README.md) - role and conventions for legacy
  non-normative research material.
- [Query-generator research index](../../research/spanner-query-gen/README.md)
  - design background and verification logs for `spanner-query-gen`.
- [Query-plan research index](../../research/spanner-query-plan-shape/README.md)
  - operator, optimizer, hint, and pattern evidence.
- [Resolved TODO snapshot](../../research/archive/RESOLVED_TODO_2026-06-12.md)
  - historical record of completed work as of 2026-06-12.

## Query-generator research

- [Optional-parameter design](../../research/spanner-query-gen/OPTIONAL_PARAMS_DESIGN.md)
  - design incorporated into the mutable `v1alpha` surface.
- [Optional-parameter prior art](../../research/spanner-query-gen/OPTIONAL_PARAMS_PRIOR_ART.md)
  - comparative background for that design.
- [Plan-contract candidates](../../research/spanner-query-gen/PLAN_CONTRACT_CANDIDATES.md)
  - candidate structural checks derived from optimization practice.
- [Plan-report operator coverage](../../research/spanner-query-gen/PLAN_REPORT_OPERATOR_COVERAGE_2026-06-12.md)
  - dated Omni verification session and classifier findings.

## Query-plan operator and metadata evidence

- [Aggregate function Agg-type observations](../../research/spanner-query-plan-shape/AGGREGATE_FUNCTION_AGG_TYPE_OBSERVATIONS_2026-08-11.md)
  - aggregate functions and their physical `Agg` expressions.
- [AI function PLAN observations](../../research/spanner-query-plan-shape/AI_PLAN_OBSERVATIONS_2026-08-13.md)
  - managed PLAN-only TVF lowering, controls, and optimizer-version partitions.
- [Query execution operator observations](../../research/spanner-query-plan-shape/QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md)
  - observed operator vocabulary, metadata, hints, and normalization impact.
- [Compact-tree metadata observations](../../research/spanner-query-plan-shape/COMPACT_TREE_METADATA_OBSERVATIONS.md)
  - rendered metadata tables for the built-in corpus.
- [Operator verification follow-up](../../research/spanner-query-plan-shape/OPERATOR_VERIFICATION_FOLLOWUP.md)
  - focused checks for previously uncertain operator families and children.
- [planvocab inference observations](../../research/spanner-query-plan-shape/PLANVOCAB_INFERENCE_OBSERVATIONS.md)
  - inferred catalog combinations and full-corpus gates.
- [Unobserved plan probe matrix](../../research/spanner-query-plan-shape/UNOBSERVED_PLAN_PROBE_MATRIX_2026-07-10.md)
  - remaining and newly closed operator-coverage hypotheses.

## Optimizer, hints, and language-surface evidence

- [Spanner optimizer versions and hints](../../research/spanner-query-plan-shape/SPANNER_OPTIMIZER_AND_HINTS.md)
  - official inventory mapped to local checks.
- [Optimizer decision control and observability](../../research/spanner-query-plan-shape/OPTIMIZER_DECISION_CONTROL_AND_OBSERVABILITY.md)
  - decisions controllable and visible enough for PLAN contracts.
- [Optimizer-version matrix observations](../../research/spanner-query-plan-shape/OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md)
  - optimizer versions 1 through 8 and distributed-merge controls.
- [Optimizer-version rendered examples](../../research/spanner-query-plan-shape/OPTIMIZER_VERSION_RENDERED_EXAMPLES.md)
  - representative version-dependent rendered plans.
- [Optimizer version 9 real-instance notes](../../research/spanner-query-plan-shape/OPTIMIZER_V9_REAL_INSTANCE_NOTES_2026-07-28.md)
  - managed-Spanner optimizer-v9 observations and the implemented Omni probe design.
- [Spanner Omni r2.1 and managed comparison](../../research/spanner-query-plan-shape/OMNI_R2_1_MANAGED_DIFF_2026-08-24.md)
  - pipe syntax, optimizer v9, DCA pruning, LIMIT sensitivity, and DML differences.
- [Hint-position audit](../../research/spanner-query-plan-shape/HINT_POSITION_AUDIT_2026-08-04.md)
  - frontend, Emulator, and Omni placement verification.
- [Factorized-mode observations](../../research/spanner-query-plan-shape/FACTORIZED_MODE_OBSERVATIONS_2026-08-11.md)
  - eligibility, effects, controls, and optimizer boundary.
- [GoogleSQL surface capability observations](../../research/spanner-query-plan-shape/GOOGLESQL_SURFACE_CAPABILITY_OBSERVATIONS_2026-08-11.md)
  - generic GoogleSQL PLAN surfaces and capability errors.
- [GoogleSQL rewriter surface observations](../../research/spanner-query-plan-shape/REWRITER_SURFACE_OBSERVATIONS_2026-08-12.md)
  - pinned upstream rewriters mapped to Spanner plans and capability boundaries.
- [GQL surface observations](../../research/spanner-query-plan-shape/GQL_SURFACE_OBSERVATIONS_2026-08-11.md)
  - graph-plan topology and SQL/GQL bridge coverage.
- [GQL hint-version observations](../../research/spanner-query-plan-shape/GQL_HINT_VERSION_OBSERVATIONS_2026-08-11.md)
  - graph hint placements, effects, controls, and version boundaries.
- [Condition-boundary observations](../../research/spanner-query-plan-shape/CONDITION_BOUNDARY_OBSERVATIONS_2026-08-11.md)
  - Split Range, seek/residual, and join-condition expression boundaries.

## Query-pattern studies and feedback

- [N-gram search observations](../../research/spanner-query-plan-shape/NGRAM_SEARCH_OBSERVATIONS_2026-08-13.md)
  - fuzzy search, pattern acceleration, controls, and eligibility boundaries.
- [Set-operation and DISTINCT hints](../../research/spanner-query-plan-shape/SET_OPERATION_DISTINCT_HINTS_2026-08-04.md)
  - set semantics, plan families, hints, and equivalent rewrites.
- [Timestamp-ordered shard queries](../../research/spanner-query-plan-shape/TIMESTAMP_ORDERED_SHARD_QUERY_OBSERVATIONS.md)
  - shard, timestamp, ordering, limit, and back-join plan shapes.
- [seekable_key_size feedback draft](../../research/spanner-query-plan-shape/SEEKABLE_KEY_SIZE_FEEDBACK_DRAFT.md)
  - undelivered feedback with a self-contained reproduction.

## OKF bundle

- [Bundle index](../index.md) - progressive-disclosure root.
- [Concepts](../concepts/index.md) - architecture and documentation guides.
- [Observations](../observations/index.md) - curated cross-source findings.
- [Research](../research/index.md) - authoring policy for new OKF-native
  research and migration from the legacy tree.
- [References](index.md) - canonical-document discovery.
