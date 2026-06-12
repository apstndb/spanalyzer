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
- [ ] **Document the `EXTERNAL_QUERY` third-argument rejection in the
  README.** The options argument is intentionally a hard error
  (`TestBigQueryAnalyzerExternalQueryOptionsArgument` pins it), but the
  README only describes the connection/SQL arguments.
- [ ] **Test `SetExternalQueryAnalyzers` with duplicate identical inner
  SQL.** The map keyed by `connectionID + innerSQL` collapses identical
  entries; that should be fine because the row type is identical, but the
  invariant deserves an explicit test.
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
- [ ] **Let generated query free functions run inside read-write
  transactions.** They take `*spanner.ReadOnlyTransaction`; consider a small
  interface such as
  `interface { Query(context.Context, spanner.Statement) *spanner.RowIterator }`
  satisfied by both transaction types.
- [ ] **Drop the `params map[string]interface{}` argument for queries with no
  parameters.**
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

- [ ] **Validation approach for the spannerplan extension point:** prototype
  a per-node annotation callback (e.g. `RowAnnotator func(*spannerpb.PlanNode)
  string` on the plantree/reference render options) in a local spannerplan
  checkout consumed via a temporary `replace` in the cmd/spanner-query-gen
  module; only propose it upstream after the seekability and family
  annotations below prove useful against real Omni plans (the Order1M shard
  schema showing `seek 2/2` on optimizer v3-v6 vs `seek 1/2` on v7-v8 is the
  acceptance demo). Do not push spannerplan changes before that validation.
- [ ] **Seekability annotation in plan-report.** Render scan rows with
  "seek k/N keys" by combining the plan's Seek/Residual Conditions and
  `seekable_key_size` with the index/table key count from the catalog DDL
  that plan-report already loads. This is the visualization of the
  shard-range discretization finding (optimizer v7-v8 dropping from 2/2 to
  1/2 keys) and is schema-dependent, hence out of spannerplan's scope.
- [ ] **Operator-family annotations in rendered reports.** plan-report can
  annotate rendered rows with normalized families (for example marking
  blocking operators under Limit, the visual form of
  `no_blocking_operator_under_limit`) without adding a plancontract
  dependency to spannerplan.
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
