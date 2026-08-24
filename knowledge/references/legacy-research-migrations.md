---
type: Reference
title: Legacy Research Migration Ledger
description: Machine-checked disposition and reverse-provenance index for the frozen legacy research corpus.
tags: [documentation, research, migration, provenance, okf]
status: draft
sources:
  - id: baseline
    resource: ../../tools/okf-check/legacy-research-baseline.txt
    title: Immutable legacy research baseline
  - id: active
    resource: ../../tools/okf-check/legacy-research-markdown.txt
    title: Active legacy research allowlist
  - id: planvocab-baseline
    resource: ../../tools/okf-check/legacy-research-planvocab.txt
    title: Immutable hashed planvocab subset
legacy_research_migrations:
  - { legacy_path: research/archive/RESOLVED_TODO_2026-06-12.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-gen/OPTIONAL_PARAMS_DESIGN.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-gen/OPTIONAL_PARAMS_PRIOR_ART.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-gen/PLAN_CONTRACT_CANDIDATES.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-gen/PLAN_REPORT_OPERATOR_COVERAGE_2026-06-12.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-gen/README.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-plan-shape/AGGREGATE_FUNCTION_AGG_TYPE_OBSERVATIONS_2026-08-11.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/AI_PLAN_OBSERVATIONS_2026-08-13.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/COMPACT_TREE_METADATA_OBSERVATIONS.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/CONDITION_BOUNDARY_OBSERVATIONS_2026-08-11.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/FACTORIZED_MODE_OBSERVATIONS_2026-08-11.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/GOOGLESQL_SURFACE_CAPABILITY_OBSERVATIONS_2026-08-11.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/GQL_HINT_VERSION_OBSERVATIONS_2026-08-11.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/GQL_SURFACE_OBSERVATIONS_2026-08-11.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/HINT_POSITION_AUDIT_2026-08-04.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/NGRAM_SEARCH_OBSERVATIONS_2026-08-13.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/OMNI_R2_1_MANAGED_DIFF_2026-08-24.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/OPERATOR_VERIFICATION_FOLLOWUP.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-plan-shape/OPTIMIZER_DECISION_CONTROL_AND_OBSERVABILITY.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-plan-shape/OPTIMIZER_V9_REAL_INSTANCE_NOTES_2026-07-28.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-plan-shape/OPTIMIZER_VERSION_RENDERED_EXAMPLES.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-plan-shape/PLANVOCAB_INFERENCE_OBSERVATIONS.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/README.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-plan-shape/REWRITER_SURFACE_OBSERVATIONS_2026-08-12.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/SEEKABLE_KEY_SIZE_FEEDBACK_DRAFT.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-plan-shape/SET_OPERATION_DISTINCT_HINTS_2026-08-04.md, state: pending, planvocab_action: pending }
  - { legacy_path: research/spanner-query-plan-shape/SPANNER_OPTIMIZER_AND_HINTS.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-plan-shape/TIMESTAMP_ORDERED_SHARD_QUERY_OBSERVATIONS.md, state: pending, planvocab_action: not-applicable }
  - { legacy_path: research/spanner-query-plan-shape/UNOBSERVED_PLAN_PROBE_MATRIX_2026-07-10.md, state: pending, planvocab_action: not-applicable }
---

# Legacy Research Migration Ledger

This Reference accounts for every body in the immutable legacy research
baseline. It is a routing and reverse-provenance index, not a second research
body. A `pending` entry remains canonical at its `legacy_path`.

A completed entry records the exact source commit and blob, then gives every
retained claim group or retired section a disposition. Non-retirement
dispositions point to current repository paths. Retirement uses a controlled
reason rather than a copied summary. The ledger remains stable after the
migration so split, merge, absorption, and retirement remain discoverable.

The repository gate enforces these invariants:

```text
immutable baseline = active legacy paths union completed ledger paths
active legacy paths intersect completed ledger paths = empty
```

For a completed entry, `dispositions` uses `retain-move`, `split`, `merge`,
`absorb`, `executable-replacement`, or `retire`. A removed planvocab input must
also map its current hashed evidence and affected catalog selectors. Probe code
is a reproducer and an expectation is an assertion; neither is by itself a
receipt that a named backend and version produced an observed result.
