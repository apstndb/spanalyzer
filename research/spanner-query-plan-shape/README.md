# spanner-query-plan-shape Research

This directory keeps long-form evidence produced by
`tools/spanner-query-plan-shape`.

The tool's command usage remains in
[`tools/spanner-query-plan-shape/README.md`](../../tools/spanner-query-plan-shape/README.md).
The files here are observation logs and upstream feedback material, not a
stable CLI contract.

## Files

- [`SPANNER_OPTIMIZER_AND_HINTS.md`](SPANNER_OPTIMIZER_AND_HINTS.md):
  official optimizer-version and hint inventory, mapped to local verification
  summaries and detailed evidence links.
- [`OPTIMIZER_DECISION_CONTROL_AND_OBSERVABILITY.md`](OPTIMIZER_DECISION_CONTROL_AND_OBSERVABILITY.md):
  interpretation layer for optimizer-version boundaries, hint controllability,
  and whether each decision is visible enough to become a PLAN contract.
- [`QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md`](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md):
  observed Spanner query-plan operator vocabulary, normalization impact, and
  probe environment notes.
- [`COMPACT_TREE_METADATA_OBSERVATIONS.md`](COMPACT_TREE_METADATA_OBSERVATIONS.md):
  regenerated `--output compact-tree-metadata` result tables for the built-in
  verification cases referenced by the operator observations.
- [`OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md`](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md):
  optimizer-version and `ALLOW_DISTRIBUTED_MERGE` matrix observations, including
  dedicated CTE reference-count checks.
- [`OPTIMIZER_VERSION_RENDERED_EXAMPLES.md`](OPTIMIZER_VERSION_RENDERED_EXAMPLES.md):
  representative optimizer-version before/after examples rendered with
  `spannerplan` reference output.
- [`TIMESTAMP_ORDERED_SHARD_QUERY_OBSERVATIONS.md`](TIMESTAMP_ORDERED_SHARD_QUERY_OBSERVATIONS.md):
  Stack Overflow timestamp-ordered sharded index query examples checked against
  rendered Spanner plan output.
- [`spanner-hacks-optimizer-version-feedback-ja.md`](spanner-hacks-optimizer-version-feedback-ja.md):
  candidate optimizer-version before/after examples for Spanner Unofficial
  Hacks, with `spannerplan` reference output.
