# spanner-query-plan-shape Research

Long-form plan-shape evidence produced with
[`tools/spanner-query-plan-shape`](../../tools/spanner-query-plan-shape/README.md)
and, for newer notes, the `plancontract` module fed with raw `AnalyzeQuery`
plans. Observation logs, not a stable contract.

## Operator vocabulary

- [`AGGREGATE_FUNCTION_AGG_TYPE_OBSERVATIONS_2026-08-11.md`](AGGREGATE_FUNCTION_AGG_TYPE_OBSERVATIONS_2026-08-11.md):
  documented Spanner aggregate functions versus the physical expressions on
  Aggregate `Agg` child links, including partial/final lowering and modifiers.
- [`QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md`](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md):
  observed Spanner query-plan operator vocabulary, normalization impact, and
  probe environment notes.
- [`COMPACT_TREE_METADATA_OBSERVATIONS.md`](COMPACT_TREE_METADATA_OBSERVATIONS.md):
  regenerated `--output compact-tree-metadata` result tables for the built-in
  verification cases.
- [`OPERATOR_VERIFICATION_FOLLOWUP.md`](OPERATOR_VERIFICATION_FOLLOWUP.md):
  follow-up checks for remaining vocabulary uncertainty — normal `SpoolScan`,
  Search Predicate mapping, Generate Relation candidates, Local Split Union,
  MiniBatch/RowCount environment sensitivity, and `Create Batch` scalar
  children.

## Optimizer behavior

- [`GoogleSQL hint placement inventory`](../../knowledge/observations/googlesql-hint-placement.md):
  normalized inventory of all generic hint placements in the reviewed
  upstream GoogleSQL grammar, separated from frontend, runtime, and
  plan-effect evidence.
- [`HINT_POSITION_AUDIT_2026-08-04.md`](HINT_POSITION_AUDIT_2026-08-04.md):
  frontend, Emulator, and Omni verification for the currently implemented
  hint-position probe set.
- [`SPANNER_OPTIMIZER_AND_HINTS.md`](SPANNER_OPTIMIZER_AND_HINTS.md):
  official optimizer-version and hint inventory mapped to local verification.
- [`OPTIMIZER_DECISION_CONTROL_AND_OBSERVABILITY.md`](OPTIMIZER_DECISION_CONTROL_AND_OBSERVABILITY.md):
  which optimizer decisions are controllable and visible enough to become
  PLAN contracts.
- [`OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md`](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md):
  optimizer-version and `ALLOW_DISTRIBUTED_MERGE` matrix observations.
- [`OPTIMIZER_VERSION_RENDERED_EXAMPLES.md`](OPTIMIZER_VERSION_RENDERED_EXAMPLES.md):
  representative optimizer-version before/after rendered examples.
- [`FACTORIZED_MODE_OBSERVATIONS_2026-08-11.md`](FACTORIZED_MODE_OBSERVATIONS_2026-08-11.md):
  factorized join acceptance, plan-visible effect, eligibility, and the v4/v5
  optimizer boundary.
- [`GQL_SURFACE_OBSERVATIONS_2026-08-11.md`](GQL_SURFACE_OBSERVATIONS_2026-08-11.md):
  broad GQL/SQL bridge plan coverage, nontrivial graph set operations,
  OPTIONAL MATCH lowering, and QUALIFY capability boundaries.
- [`GQL_HINT_VERSION_OBSERVATIONS_2026-08-11.md`](GQL_HINT_VERSION_OBSERVATIONS_2026-08-11.md):
  graph-specific hint placements, plan-visible effects, accepted-no-effect
  controls, and exact optimizer-version boundaries.
- [`GOOGLESQL_SURFACE_CAPABILITY_OBSERVATIONS_2026-08-11.md`](GOOGLESQL_SURFACE_CAPABILITY_OBSERVATIONS_2026-08-11.md):
  accepted generic GoogleSQL PLAN surfaces plus explicit runtime and
  transaction capability boundaries.
- [`CONDITION_BOUNDARY_OBSERVATIONS_2026-08-11.md`](CONDITION_BOUNDARY_OBSERVATIONS_2026-08-11.md):
  expression boundaries between Split Range, scan Seek/Residual Condition,
  and Hash/Merge/Apply join conditions across optimizer v1-v8.

## Pattern studies

- [`TIMESTAMP_ORDERED_SHARD_QUERY_OBSERVATIONS.md`](TIMESTAMP_ORDERED_SHARD_QUERY_OBSERVATIONS.md):
  the timestamp-ordered sharded index pattern (Stack Overflow thread,
  gcpug/nouhau#135) verified against rendered plans, updated 2026-06-12 with
  the optimizer-version dependence of shard-range seekability.

## Feedback drafts

- [`SEEKABLE_KEY_SIZE_FEEDBACK_DRAFT.md`](SEEKABLE_KEY_SIZE_FEEDBACK_DRAFT.md):
  draft Google-channel feedback (not delivered) on the `seekable_key_size`
  plan-metadata field reporting `0` for all-equality point seeks, contrary to
  the documented definitions and the `SEEKABLE_KEY_SIZE` hint. Reproduced on
  both Cloud Spanner DBaaS and Omni 2026.r1-beta with a self-contained minimal
  table. Remove after delivery, as past drafts were.

Delivered feedback drafts for Spanner Unofficial Hacks were removed on
2026-06-12 after upstream incorporation; see git history before that date.
