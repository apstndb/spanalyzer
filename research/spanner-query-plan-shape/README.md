# spanner-query-plan-shape Research

Long-form plan-shape evidence produced with
[`tools/spanner-query-plan-shape`](../../tools/spanner-query-plan-shape/README.md)
and, for newer notes, the `plancontract` module fed with raw `AnalyzeQuery`
plans. Observation logs, not a stable contract.

## Operator vocabulary

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

- [`SPANNER_OPTIMIZER_AND_HINTS.md`](SPANNER_OPTIMIZER_AND_HINTS.md):
  official optimizer-version and hint inventory mapped to local verification.
- [`OPTIMIZER_DECISION_CONTROL_AND_OBSERVABILITY.md`](OPTIMIZER_DECISION_CONTROL_AND_OBSERVABILITY.md):
  which optimizer decisions are controllable and visible enough to become
  PLAN contracts.
- [`OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md`](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md):
  optimizer-version and `ALLOW_DISTRIBUTED_MERGE` matrix observations.
- [`OPTIMIZER_VERSION_RENDERED_EXAMPLES.md`](OPTIMIZER_VERSION_RENDERED_EXAMPLES.md):
  representative optimizer-version before/after rendered examples.

## Pattern studies

- [`TIMESTAMP_ORDERED_SHARD_QUERY_OBSERVATIONS.md`](TIMESTAMP_ORDERED_SHARD_QUERY_OBSERVATIONS.md):
  the timestamp-ordered sharded index pattern (Stack Overflow thread,
  gcpug/nouhau#135) verified against rendered plans, updated 2026-06-12 with
  the optimizer-version dependence of shard-range seekability.

Delivered feedback drafts for Spanner Unofficial Hacks were removed on
2026-06-12 after upstream incorporation; see git history before that date.
