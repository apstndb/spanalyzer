# TODO

This repository is still experimental. This file tracks open follow-up work
only. Completed items and historical audit trails are archived in
[`research/archive/RESOLVED_TODO_2026-06-12.md`](research/archive/RESOLVED_TODO_2026-06-12.md)
and in git history.

## Analyzer And Catalog

- [ ] **Property graph derived expressions still hardcode `JSON`.** As
  verified in the Cloud Spanner Emulator source (`PopulatePropertyGraph`),
  the property type should match the analyzed type of the expression (for
  example `INT64` for `LENGTH(FirstName)`). Extract the correct type from the
  analyzer output without increasing direct dependencies on `go-zetasql`.
  Note that `TestAnalyzerRowTypeForPropertyGraphWithExpressions` currently
  asserts the hardcoded `JSON` behaviour; fix the extraction first or mark
  the test as pinning known-wrong behaviour.
- Continue comparing native proto/enum analyzer behavior with Cloud Spanner
  and the Cloud Spanner emulator (top-level proto outputs, nested fields,
  arrays, enum values).
- Keep regular indexes, vector indexes, and search indexes ignored for
  row-type analysis unless a future feature needs index metadata for
  generated queries or plan reports.
- Add live verification hooks for BigQuery external dataset access when a
  stable evidence source is available. The current config support is static
  modeling.

## Generated Code Surface

These change the generated API, so they should be decided together before any
config/output freeze (v1 freeze is deliberately deferred).

- [ ] **Derive query result struct nullability from DDL for `kind: table` and
  `kind: index`.** Write input structs already use DDL `NOT NULL` to choose
  `int64` vs `spanner.NullInt64`; result structs always emit `spanner.Null*`.
  Shorthand kinds project bare table columns, so nullability is derivable.
  Decide whether `kind: sql` stays conservative and whether that split is
  acceptable for DTO reuse.
- [ ] **omit_when_empty on kind:index.** Currently rejected (key columns are
  scalar; ARRAY keys are rare). Lift if needed.
- [ ] **Tristate API.** Combine `null_is_null` + `omit_when_null` into a
  single `tristate` Go field expressing match-all / match-NULL / match-value.

## Plan Reports And Contracts

- Capture effective optimizer version and statistics package when the backend
  source can expose them reliably.
- Capture Spanner Omni backend identity automatically when spanemuboost or
  the backend runtime exposes stable version/image digest evidence (see the
  spanemuboost item below).
- Keep PLAN-only contract evaluation explicit. PROFILE execution stats are
  out of scope unless a separate profile-contract surface is designed.
- Grow predefined operator families only from observed plans, fixtures, or
  concrete contract use cases.
- [ ] **Decide blocking_operator attribution for wrapper/implementation
  pairs.** Found by reading family-annotated rendered output: the
  push_broadcast_hash_join wrapper is in `streamBlockingOperatorFamily` but
  `push_broadcast_hash_join_internal_hash_join` is not, so the node that
  performs the actual hash build renders without the blocking_operator
  attribute and contributes nothing to subtree blocking counts. Plan-wide
  counts avoid double counting this way, but a subtree rooted strictly
  between the wrapper and the implementation (for example a pushed Limit on
  the Map side) would see zero blocking operators above a hash build. A
  more principled attribution may be the inverse: the implementation node
  is blocking and the distribution wrapper is not, keeping plan-wide counts
  at one while fixing subtree scoping. Changes observable
  operator_family_counts, so decide deliberately (v1alpha allows it).
- [ ] **Extend plan-report to DML targets.** Mechanics proven:
  `ReadWriteTransaction.AnalyzeQuery` returns DML plans in PLAN mode without
  executing, and plancontract classifies them warning-free
  (`TestIntegrationDMLOperatorFamilyCoverageOnOmni`). Remaining design work
  is the target surface: whether `writes[]` helpers and/or DML `queries[]`
  (row_count / row_set modes are rejected by the public v1alpha config)
  become targets with `write/<name>`-style IDs, plus README and schema
  updates. The RowCount / MiniBatch* operators are unrelated to DML despite
  the `row_count` config-mode naming (they are undocumented SELECT back-join
  operators seen only on Cloud Spanner optimizer v5; see
  `research/spanner-query-plan-shape/OPERATOR_VERIFICATION_FOLLOWUP.md`).
- [ ] **Wire optional-parameter Variants into the plan probes.**
  `tools/spanner-query-plan-shape` does not consume the `Variants` slice, and
  `plan-report` analyzes only the canonical all-on variant (verified live:
  two `omit_when_null` params produced one report target instead of four).
  `tools/optparam-plan-probe` already contains the per-variant acquisition
  loop that could be ported.
- [ ] **Support applying database options in the Omni plan probes once Omni
  accepts them.** Needed to verify
  `use_unenforced_foreign_key_for_query_optimization` (for example a
  `--database-option key=value` flag that pins the database ID and issues
  `ALTER DATABASE`). Blocked: Omni 2026.r1-beta rejects that option via
  `ALTER DATABASE` (empty-message InvalidArgument) and does not parse the
  documented `SET DATABASE OPTIONS` syntax, while `version_retention_period`
  applies fine — the gap is the option, not the harness. See
  `research/spanner-query-gen/PLAN_REPORT_OPERATOR_COVERAGE_2026-06-12.md`.

## Schema-Aware Plan Rendering (this repo, not spannerplan)

Visualization enrichment that needs inputs spannerplan deliberately does not
have belongs here: this repository uniquely holds the DDL catalog
(root module), the operator-family normalization (`plancontract`), and the
plan acquisition workflows. spannerplan already covers variable resolution
(`rendertree --resolve-vars --resolve-vars-recursive`) and spannerplanviz
already draws distribution boundaries (dashed, SVG / mermaid.js), so those
are out of scope.

- [x] **Validation approach for the spannerplan extension point:** validated
  on 2026-06-12. Two complementary hooks are prototyped on the local
  `row-annotator` branch of a spannerplan checkout, consumed via an
  uncommitted `replace` in `go.work`: value-replacing
  `queryplan.WithMetadataValueFunc` (plus a
  `reference.WithQueryPlanOptions` passthrough) for enriching fields the
  plan already renders, and additive
  `plantree.WithRowAnnotator` / `reference.WithRowAnnotator` for
  information with no metadata counterpart. The acceptance demo passed on
  Omni 2026.r1-beta (`TestIntegrationSeekabilityAnnotationOnOmni`:
  shard-range query renders `seekable_key_size: 2/2` under optimizer
  version 6 and `1/2` under version 8; the per-shard rewrite stays `2/2`
  on both).
- [ ] **Upstream the spannerplan RowAnnotator hook and commit the dependent
  code.** The `--annotate` implementation in cmd/spanner-query-gen compiles
  only against the local spannerplan branch, so it stays uncommitted until
  the hook is merged and tagged upstream; then drop the `go.work` replace,
  bump the spannerplan requirement, and commit the feature plus this TODO
  update together.
- [x] **Seekability annotation in plan-report** (uncommitted until the
  spannerplan hooks ship, see above). `plan-report --annotate seekability`
  replaces the rendered `seekable_key_size` value in place with `k/N`,
  where N is the declared key column count of the scanned table or index
  from the catalog DDL, avoiding a duplicate row suffix. Declared keys
  only: the implicit base-table primary key suffix of secondary indexes is
  not counted, so key-joining probes can in principle render k greater
  than N. The ambiguous value 0 is intentionally left unannotated:
  verified on Omni 2026.r1-beta, `seekable_key_size` counts the key prefix
  of a range-bounded seek, so both full scans and perfect point seeks
  (all-equality key conditions, literal or parameter) report 0 (see
  `research/spanner-query-plan-shape/QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md`).
- [x] **Operator-family annotations in rendered reports** (uncommitted, same
  gate). `plan-report --annotate families` renders `{<family>[: <umbrella>...]}`
  labels per relational row from plancontract normalization (for example
  `{full_sort: blocking_operator, explicit_sort}` — the single-valued
  concrete family left of the colon, derived umbrella attributes right of it
  in lexicographic order), without adding a
  plancontract dependency to spannerplan. Braces are reserved for these
  labels by convention. The umbrella suffixes come from the new
  `plancontract.DerivedOperatorFamilies`, which is pinned to
  `AddDerivedOperatorFamilyCounts` by a consistency test and is committable
  independently of the spannerplan gate.
- [ ] **Consider a families annotation mode that skips trivial labels.**
  Observed on real and fixture plans: most rows render labels that merely
  restate the display name (`Global Limit {limit}`, `Batch Scan ... {scan}`,
  `DataBlockToRow {data_block_to_row}`), while the value concentrates where
  the family diverges from the title (`Local Sort Limit {full_sort:
  blocking_operator, explicit_sort}`, `Distributed Union on idx
  {distributed_merge_union}`, `Hash Join
  {push_broadcast_hash_join_internal_hash_join}`). A mode that labels only
  rows whose family differs from the trivially normalized display name or
  carries umbrella attributes would cut the noise; keep the current
  exhaustive mode for normalization debugging.
- [ ] **Normalized plan diff for optimizer comparisons.** An aligned tree
  diff over plancontract-normalized operators (shared prefix folded,
  differing operators expanded) to replace eyeballing
  `--optimizer-version-diff` compact trees. The operator tree digest
  normalization already provides the alignment vocabulary.
- [ ] **Hidden-scalar summary in rendered trees.** Rendered IDs jump over
  scalar-kind nodes, and classification warnings can reference nodes that
  are invisible in the tree; a per-row summary of folded scalar children
  (counts by display name) would make those references resolvable. Note for
  edge labels: Recursive Union inputs (`input_0` / `input_1`) should not be
  hard-labeled Base/Recursive because recursive CTE support could make the
  branch count variable.

## Dependencies And Infrastructure

- [ ] **Migrate the cmd/spanner-query-gen and tools modules to testcontainers
  v0.42+.** They pin the known-good spanemuboost v0.4.0 / testcontainers
  v0.40.0 pair because v0.42 moved Docker types to github.com/moby/moby and
  breaks the `WithConfigModifier` callback signature in
  `integration_test.go`. Coordinate with a spanemuboost upgrade.
- [ ] **Propose a runtime image/digest API in spanemuboost.** v0.4.0 exposes
  `RuntimePlatform` but no resolved image or digest, so plan-report backend
  identity stays `not_recorded` unless supplied manually.
- [ ] **Consider surfacing testcontainers Docker-host discovery failures as
  errors instead of panics** (spanemuboost propagates the
  `rootless Docker not found` panic from `MustExtractDockerHost`; a wrapped
  error with remediation hints would be friendlier for CLI users).
