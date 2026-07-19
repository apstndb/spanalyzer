# TODO

This repository is still experimental. This file tracks open follow-up work
only. Completed items and historical audit trails are archived in
[`research/archive/RESOLVED_TODO_2026-06-12.md`](research/archive/RESOLVED_TODO_2026-06-12.md)
and in git history.

## Long-Term Repository Hardening Backlog

These items came from the 2026-07-03 repository review. They are deliberately
kept as long-term planning notes rather than immediate fixes because most of
them change public APIs, module boundaries, diagnostics, or larger internal
architecture.

- [ ] **Normalize catalog object identity before any v1 freeze.** Spanner
  identifiers are case-insensitive, but several root catalog maps are keyed by
  original spelling. Introduce a single normalized object-key helper that keeps
  display spelling in values, then route table, index, view, sequence, model,
  proto, and graph lookups through it. Fold related DDL consistency issues into
  the same pass: table/view conflict checks, rename/drop index bookkeeping,
  synonym overwrite handling, and raw map lookups from analyzer-returned names.
- [ ] **Make row-type-irrelevant DDL lenient by design.** Common migration DDL
  such as `ALTER TABLE ADD/DROP CONSTRAINT`, row deletion policies, and
  unsupported `ALTER INDEX` alterations should not hard-fail row-type analysis
  when the schema shape used by query analysis is unchanged. Add explicit
  ignore paths for known irrelevant AST nodes and consider a catalog option
  that records warnings instead of failing on unknown but likely irrelevant
  DDL.
- [ ] **Consolidate SQL scanning on parser-backed or shared lexical logic.**
  The repo still contains multiple hand-rolled scanners for table-reference
  detection, star detection, optional-parameter markers, BigQuery
  `EXTERNAL_QUERY`, string literal unescaping, and developer probes. Replace
  substring/prefix heuristics with memefish, GoogleSQL resolved AST data, or a
  single quote/comment-aware scanner package with focused tests.
- [ ] **Trim and stabilize the root public API surface.** Before freezing any
  root API, reduce constructor proliferation around analyzer/options, document
  or avoid catalog-constructor mutation, and move generator/report policy types
  out of the root package. In particular, BigQuery Spanner external dataset
  audit/remediation policy belongs with querygen/reporting unless a smaller
  reusable projection API is intentionally designed.
- [ ] **Harden BigQuery and external-query analyzer internals.** Delete or test
  dead textual-rewrite paths, avoid mutable per-analyzer `EXTERNAL_QUERY`
  prepared state where concurrent analysis can race, dedupe duplicate
  argument validation, implement or reject the full GoogleSQL string escape
  set, apply external-dataset registration atomically so a rejected schema
  cannot leave a partially mutated live catalog, and make ambiguous
  `ML.PREDICT` model fallback fail loudly when multiple models are registered.
- [ ] **Finish unifying proto descriptor and type-resolution passes.**
  Descriptor sets loaded from multiple files now dedupe identical descriptor
  file names and reject conflicting definitions before building a
  `protodesc.Files` pool. PROTO/ENUM fix-up must still run over every `TypeSpec`
  carrier, including model inputs/outputs and view-derived types, not only
  table columns.
- [ ] **Strengthen `plancontract` as an independently reliable module.** Add
  table-driven tests for operator-family classification, topology,
  predefined-contract expansion, YAML validation, CEL evaluation, and digest
  behavior. Extract `Validate(File)`, call it from both `ReadFile` and
  `Evaluate`, report per-contract CEL errors instead of aborting the whole
  batch, prefer typed CEL inputs for normalized data, make digests deterministic
  by sorting by node index, include scan-target identity where appropriate, and
  emit diagnostics when a family comes from a heuristic fallback.
- [ ] **Refactor `plan-report` around testable boundaries.** Split the large
  command file into command, collection, rendering, and type files; replace
  historical one-line forwarding helpers with direct plancontract calls; and
  extract a small `planAnalyzer` interface so target iteration, skip/error
  handling, report assembly, annotation, and invariant validation can be tested
  without an Omni container. Catalog-open/DDL failures should become per-target
  errors when possible instead of discarding the entire partial report.
- [ ] **Finish querygen and optparam structural cleanup.** Share the
  GenerateQueryCode/BuildQueryCodegenPlan resolution pipeline, dedupe table
  and index key-prefix predicate assembly, decide nullable ARRAY element
  policy for query results, escape GoogleSQL string literals and identifiers
  according to dialect rules, escape generated struct-tag values, introduce
  one package-wide symbol
  allocator for top-level and nested generated declarations, compile complete
  generated packages in a collision/type matrix, make missing required-field
  diagnostics deterministic, dedupe generated optional-builder labels when one
  param appears in multiple segments, and make marker extraction reject
  strings/comments/orphaned residual markers intentionally.
- [ ] **Use tooling to prevent repeat classes of bugs.** Add targeted lints or
  tests for exhaustive `spannerpb.TypeCode` and AST-kind switches, align
  spannerplan versions across modules once the local replace is gone, add a
  focused GC-enabled subprocess or stress gate so suite-wide GC suppression
  does not permanently hide WASM-wrapper lifecycle regressions, and modernize
  developer probes such as semicolon-splitting DDL helpers so they reuse
  existing parsers instead of ad hoc string handling.

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
- [ ] **Add live proto-bundle plan-report coverage.** Plan acquisition now
  canonicalizes, merges, and dedupes configured descriptor sets, passes them to
  spanemuboost database creation, and records the deterministic merged
  descriptor digest. Add a tagged Omni case that creates a PROTO BUNDLE and
  successfully analyzes a proto-field query end to end.
- [ ] **Normalize logical scan access paths.** Join each Filter Scan wrapper
  with its unique Scan child into `access_paths[]`: target and base-table
  identity, scan kind, full-scan state, Seek/Residual/Timestamp/Search signals,
  nullable range `seekable_key_size`, declared keys, and catalog-aware index
  coverage. Do not infer point-seek depth from `seekable_key_size=0`, which is
  also reported for full scans. Coalesce a `VectorIndexRootScan` plus its
  batch-driven `VectorIndexLeafScan` into one logical ANN access path; do not
  interpret the root centroid table's `Full scan: true` as a full base-row
  scan. Represent any later base-table back join separately.
- [ ] **Add non-vacuous positive plan predicates.** Start with
  `require.operator_family` plus `min_count` and matched indexes. Forbidding
  `hash_aggregate` does not prove a Stream Aggregate exists, and join
  elimination can make negative join-family contracts pass vacuously.
- [ ] **Add captured QueryPlan replay and a DBaaS fixture corpus.** Accept the
  protojson/protoyaml envelopes produced by `spanner-query-plan-shape`, retain
  backend/optimizer/statistics/capture and catalog provenance, and cover
  Local Split Union, MiniBatch/RowCount, generic TVFs, join elimination,
  historical spellings, and other operators Omni cannot produce. Use the
  exact DDL/data/query/hint recipes in
  `research/spanner-query-plan-shape/UNOBSERVED_PLAN_PROBE_MATRIX_2026-07-10.md`
  rather than empty-schema variants for the data-dependent cases.
- [ ] **Canonicalize operator-tree digests independently of PlanNode numbers.**
  Concrete normalization IDs now change when digest inputs or family semantics
  change; the remaining step is a canonical topology encoding that does not
  make an otherwise equivalent renumbered plan compare different.
- [ ] **Design a separate read-only profile-report surface.** Normalize query
  and node runtime statistics and ratios with dataset/setup/warmup/repetition
  provenance. Keep DML out of PROFILE and keep runtime thresholds out of the
  structural PLAN contract surface.
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
  `row-annotator` branch of a spannerplan checkout: value-replacing
  `queryplan.WithMetadataValueFunc` (plus a
  `reference.WithQueryPlanOptions` passthrough) for enriching fields the
  plan already renders, and additive
  `plantree.WithRowAnnotator` / `reference.WithRowAnnotator` for
  information with no metadata counterpart. The acceptance demo passed on
  Omni 2026.r1-beta (`TestIntegrationSeekabilityAnnotationOnOmni`:
  shard-range query renders `seekable_key_size: 2/2` under optimizer
  version 6 and `1/2` under version 8; the per-shard rewrite stays `2/2`
  on both). The downstream prototype and its tests remain available in commit
  `7b5d032`; they were removed from the active CLI on 2026-07-12 because the
  external workspace replacement made every GitHub Actions job fail on a
  fresh runner.
- [ ] **Upstream the spannerplan RowAnnotator hook and restore standalone
  module closure before restoring annotations.** First merge and tag the two
  spannerplan hook changes. Then restore the `--annotate` prototype from
  commit `7b5d032`; it also uses
  `plancontract.DerivedOperatorFamiliesForOperator` and
  `ProtoDescriptorSet.FileDescriptorSet`, which are absent from the root and
  plancontract versions pinned by the command module.
  Tag the root and plancontract changes, bump all command requirements, and
  add `GOWORK=off` CI for every nested module before restoring the CLI surface.
- [ ] **Restore seekability annotation in plan-report after the dependency
  gate above.** The validated `plan-report --annotate seekability` prototype
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
- [ ] **Restore operator-family annotations in rendered reports after the
  dependency gate above.** The validated `plan-report --annotate families`
  prototype renders
  `{<family>[: <umbrella>...]}`
  labels per relational row from plancontract normalization (for example
  `{full_sort: blocking_operator, explicit_sort}` — the single-valued
  concrete family left of the colon, derived umbrella attributes right of it
  in lexicographic order), without adding a
  plancontract dependency to spannerplan. Braces are reserved for these
  labels by convention. The umbrella suffixes come from the metadata-aware
  `plancontract.DerivedOperatorFamiliesForOperator`; the family-only
  `DerivedOperatorFamilies` remains pinned to `AddDerivedOperatorFamilyCounts`,
  while the operator-aware helper additionally marks scalar Stream Aggregates
  as blocking.
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

- [ ] **Migrate the tools module to spanemuboost v0.4.4 and testcontainers
  v0.42.0.** `cmd/spanner-query-gen` already uses that pair and the updated
  `github.com/moby/moby` `WithConfigModifier` callback type. `tools` still pins
  the known-good spanemuboost v0.4.0 / testcontainers v0.40.0 pair; coordinate
  its migration with the developer probes that depend on it.
- [ ] **Propose a runtime image/digest API in spanemuboost.** v0.4.0 exposes
  `RuntimePlatform` but no resolved image or digest, so plan-report backend
  identity stays `not_recorded` unless supplied manually.
- [ ] **Consider surfacing testcontainers Docker-host discovery failures as
  errors instead of panics** (spanemuboost propagates the
  `rootless Docker not found` panic from `MustExtractDockerHost`; a wrapped
  error with remediation hints would be friendlier for CLI users).
