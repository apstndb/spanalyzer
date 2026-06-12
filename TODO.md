# TODO

This repository is still experimental. This file tracks concrete follow-up work
that is known from the current implementation and documentation.

## Current Coordination Notes

- The 2026-05-09 other-agent progress has been committed and the build is back
  to green as of 2026-05-14. The historical "working tree" items below are kept
  as an audit trail, not as current blockers.
- The previously-noted `list_features.go` blocker is no longer present in the
  tree.

## Outstanding Issues In The Working Tree (2026-05-09)

These were the merge blockers found in the 2026-05-09 working tree. They are
now fixed, but the section is kept so future agents can understand why the
cleanup happened.

- [x] **Build is broken in `internal/querygen`.** Fixed by replacing the stale
  `allNonKeyColumnNames(table, keyColumns)` call with the existing
  `nonKeyColumnNames(selectableColumnNames(table), keyColumns)` helper. Verified
  with `go test ./...` on 2026-05-14.
- [x] **`gofmt` failure in `internal/querygen/querygen_test.go`.** Fixed by
  formatting the querygen package; `go test ./...` now builds the package.
- [x] **Stray files in the working tree.** Untracked artifacts to clean up
  before committing: `cmd/spanner-query-gen/main_test.go.bak2`,
  `internal/querygen/querygen_test.go.bak{,2,3,4}`, the `spanner-query-gen`
  built binary at the repo root, `all_features.txt`, and
  `testdata/protos/complex/` if it is not actually intended to be committed
  alongside the new `TestAnalyzerRowTypeForComplexProtoBundle` test (the test
  references `testdata/protos/complex/complex_descriptors.pb`, so the
  descriptor file should be tracked). The scratch files and binary were
  removed; `testdata/protos/complex/` is intentionally kept as a fixture.
- [x] **`cmd/spanner-query-gen/plan_report.go:390` keeps a local variable named
  `source` after renaming the field to `Catalog`.** Renamed the local variable
  to `catalog` for consistency with the public field name.

## Behavioural Concerns To Verify Before Merging

- [ ] **Property graph derived expressions still hardcode `JSON`.** The
  "Correction Needed" note further down already calls this out, but the new
  `TestAnalyzerRowTypeForPropertyGraphWithExpressions` test asserts the
  hardcoded JSON behaviour, which would lock in the bug as a regression
  oracle. Either fix the type extraction first, or mark the test with a TODO
  comment so it is not treated as the desired final behaviour.
- [ ] **`EXTERNAL_QUERY` third-argument options are now a hard error.** The new
  `TestBigQueryAnalyzerExternalQueryOptionsArgument` test pins that behaviour.
  The earlier TODO entry asked for "support or intentionally reject ... with
  documented behavior"; the README update only describes the connection/SQL
  arguments and does not mention the rejection. Add a one-line README note so
  users do not have to discover the rejection from a runtime error.
- [ ] **`SetExternalQueryAnalyzers` ordering.** The new map keyed by
  `connectionID + innerSQL` is robust against duplicate / nested calls, but
  identical inner-SQL strings under the same connection collapse to a single
  entry. Confirm with a test that two `EXTERNAL_QUERY` calls using the
  same connection and identical inner SQL still resolve correctly (they
  should, because the row type is identical, but the absence of an explicit
  test makes this an implicit invariant).
- [x] **`resultconv.go` swallows non-`THEN RETURN` DMLs as `ErrStatementHasNoRowType`,
  including the `errors` from `Returning()`.** The new code does
  `if err != nil || returning == nil { return nil, ErrStatementHasNoRowType }`,
  which masks any actual frontend error behind the sentinel. Surface the
  original error (wrapping is fine) so debugging an analyzer regression does
  not require re-running with extra logging. `Returning()` errors are now
  wrapped, while a missing returning clause still maps to `ErrStatementHasNoRowType`.

## Documentation Drift

- [x] Update the README `EXTERNAL_QUERY` limitation. BigQuery analysis now keeps the
  original `EXTERNAL_QUERY` TVF in the analyzed SQL and feeds the TVF callback
  with prepared row types. The older text still says the call is rewritten to an
  empty subquery before analysis.
- [x] Add a short BigQuery `EXTERNAL_QUERY` example that shows connection-specific
  external DDL and confirms that proto fields can be used inside the Spanner
  inner query while unsupported result values stay rejected.
- [x] Document the `ML.PREDICT` analyzer-only model more explicitly: it models
  schema only, returns model output columns followed by non-duplicated input
  relation columns, and does not execute prediction logic.

## BigQuery EXTERNAL_QUERY

- [x] Replace the prepared-row-type handoff with direct TVF callback argument value
  extraction once `go-googlesql` exposes a safe API for scalar literal values in
  `TVFInputArgumentType`.
- [x] Support or intentionally reject the third `EXTERNAL_QUERY` options argument
  with documented behavior. It is currently accepted for signature compatibility
  but not interpreted.
- [x] Add coverage for nested or repeated `EXTERNAL_QUERY` calls whose source order
  and analyzer callback order could diverge.
- [x] Keep PostgreSQL-dialect Spanner inner SQL explicitly unsupported until there
  is a parser/analyzer path for it.
- [x] Decide whether `ParseDebugString` and `Unparse` should validate
  `EXTERNAL_QUERY` literal arguments or remain parse-only helpers. (Decision: Keep them parse-only to match their design and avoid duplication).
- [x] Verify direct `TVFInputArgumentType.ScalarExpr()` literal extraction under the
  latest `go-googlesql` release with `go test ./...` after removing or moving
  the root-level `list_features.go` scratch file.
- [x] Confirm the implementation handles duplicate identical `EXTERNAL_QUERY`
  calls, nested scalar-subquery calls, and callback ordering without depending
  on source order. Prefer connection ID plus inner SQL lookup over sequential
  consumption.

## Proto And Enum Types

- Continue comparing native proto/enum analyzer behavior with Cloud Spanner and
  the Cloud Spanner emulator, especially for top-level proto outputs, nested
  proto fields, arrays, and enum values.
- [x] Expand descriptor-set tests beyond the order proto sample so native
  `MakeProtoType` / `MakeEnumType` paths cover multiple files, imports, and
  packages.
- [x] Decide how BigQuery external dataset and `EXTERNAL_QUERY` surfaces should
  report unsupported Spanner proto, enum, struct, and tokenlist result columns. (Decision: Reject them with descriptive error messages based on BigQuery documentation).

## Spanner DDL And Catalog Semantics

- [x] Improve property graph coverage beyond direct column-derived labels and
  properties, including arbitrary property expressions and dynamic
  labels/properties when the GoogleSQL frontend API can represent them. (Implemented arbitrary property expressions as JSON types and added dynamic label/property support).
  - **Correction Needed**: The current implementation incorrectly hardcodes arbitrary property expressions to fall back to the `JSON` type. As verified in the Cloud Spanner Emulator source (`PopulatePropertyGraph`), the property type should exactly match the analyzed type of the expression (e.g., `INT64` for `LENGTH(FirstName)`). We need to fix this to extract the correct type from the analyzer output without increasing direct dependencies on `go-zetasql`.
- [x] Audit remaining Spanner DDL forms that can affect query semantics and add
  targeted catalog support or explicit ignored-object tests. (Audit completed: added support for ignoring change streams, database options, and IAM objects).
- Keep regular indexes, vector indexes, and search indexes ignored for
  row-type analysis unless a future feature needs index metadata for generated
  queries or plan reports. (`CREATE SEARCH INDEX` was added to the ignored set
  on 2026-06-12 so full-text search schemas can flow through plan-report.)

## BigQuery Catalog Semantics

- [x] Expand BigQuery DDL support beyond ordinary tables and views when the object
  affects result type analysis. (Added support for ignoring/bypassing `CREATE TABLE FUNCTION`, `CREATE PROCEDURE`, and `EXPORT DATA`).
- [x] Track BigQuery query result nullability if the GoogleSQL analyzer exposes
  enough information to distinguish `REQUIRED` from `NULLABLE`. (Decision: Current analyzer does not expose this information for results; keeping them NULLABLE for now).
- [x] Add more BigQuery dialect feature preset tests, especially where BigQuery and
  Spanner GoogleSQL diverge.

## spanner-query-gen

- [x] Generate runtime query methods for configured query cardinality (`one`,
  `maybe_one`, `many`) after the DTO and SQL constant surfaces stabilize.
- [x] Review generated runtime query methods before treating them as complete:
  ensure generated imports include `context`, `iterator`, `spanner`, and
  `bigquery` only when needed; ensure generated code compiles for Spanner-only,
  BigQuery-only, and `both` targets; and add golden tests for each cardinality. (Imports are now surgically added; validated Spanner/BQ/Both targets with tests).
- [x] Decide whether the first-pass runtime query methods should return iterators,
  loaded `[]*Struct` slices, or both. (Decision: Implemented both; added `...All` methods returning slices).
- [x] For `one` and `maybe_one`, decide whether generated methods must detect
  multiple rows and return a cardinality error instead of just reading the first
  row. (Implemented cardinality enforcement by checking for a second row).
- [x] Model row-count-only DML execution, DML `THEN RETURN`, and custom command
  entries separately from `queries`. (Supported row_count and row_set result modes).
- [x] Decide the long-term config shape for explicit upsert conflict targets,
  insert columns, and update masks. (Structured `QueryCodegenWrite` to use `InsertColumns` and `UpdateColumns`).
- Add live verification hooks for BigQuery external dataset access when a stable
  evidence source is available. The current config support is static modeling.

## Plan Reports And Contracts

- Capture effective optimizer version and statistics package when the backend
  source can expose them reliably.
- Capture Spanner Omni backend identity automatically when spanemuboost or the
  backend runtime exposes stable version/image digest evidence.
- Keep PLAN-only contract evaluation explicit. PROFILE execution stats are out
  of scope unless a separate profile-contract surface is designed.
- Grow predefined operator families only from observed plans, fixtures, or
  concrete contract use cases.

## Optional Query Parameters (optparam integration)

Added 2026-05-14. `internal/optparam` is integrated into `internal/querygen`;
see `research/spanner-query-gen/OPTIONAL_PARAMS_DESIGN.md` for the full design.

### Required — needed to make the feature complete

- [x] **Integrate EmitGoBuilder into GenerateQueryCode (gostruct.go).**
  Optional queries now emit a typed `<Name>Params` struct and a
  `Build<Name>SQL` helper instead of a single SQL constant. Required-only
  queries keep the existing constant-emit path.

- [x] **Strict v1alpha validation for `choices` / `default` key names.**
  `validateQueryCodegenOptionalParam` now rejects `orderby_choice` keys and
  defaults that do not match `^[A-Za-z_][A-Za-z0-9_]*$`.

- [x] **orderby_choice support on kind:table.**
  `kind: table` now mirrors `kind: index` by emitting
  `/*?orderby:NAME*/ <default> /*?end*/` when an `orderby_choice` param is
  present.

### Documentation

- [x] **Update OPTIONAL_PARAMS_DESIGN.md prototype/scope sections.**
  The "Prototype status" and "Minimum-viable scope" sections now reflect the
  integrated v1alpha state, including `kind:index`, `kind:table`, and generated
  SQL builders.

### Cleanup

- [x] **Remove the PoC worktree.**
  `worktree-optional-params-poc` is fully merged into main. Verified on
  2026-06-12 that neither the worktree nor the branch exists anymore.

- [x] **Remove stray .bak files.**
  `internal/querygen/querygen_test.go.bak{1-4}` and
  `cmd/spanner-query-gen/main_test.go.bak2` should be deleted.

### Future / deferrable

- [ ] **Wire plan contract Variants into spanner-query-plan-shape.**
  `tools/optparam-plan-probe` is a standalone probe. The main
  `tools/spanner-query-plan-shape` does not yet consume the `Variants` slice
  from a plan-contract YAML to produce one plan per variant automatically.

- [ ] **omit_when_empty on kind:index.**
  Currently rejected (key columns are scalar; ARRAY keys are rare). Lift if
  needed.

- [ ] **Tristate API.**
  `null_is_null` + `omit_when_null` combined into a single `tristate` Go
  field so the caller can express match-all / match-NULL / match-value with
  one field instead of two.

## Improvement Proposals From 2026-06-12 Omni Verification Session

These were found while exercising the CLIs end-to-end against Spanner Omni.
They need a design decision before implementation, so they are recorded here
instead of being fixed inline.

- [ ] **Derive query result struct nullability from DDL for `kind: table` and
  `kind: index`.** Write input structs already use DDL `NOT NULL` to choose
  `int64` vs `spanner.NullInt64`, but query result structs always emit the
  `spanner.Null*` wrappers even for `NOT NULL` columns selected directly from
  one table. Table/index shorthand queries project bare table columns, so
  nullability is derivable from the catalog. Decide whether `kind: sql`
  queries should stay conservative (analyzer output does not expose
  nullability) while shorthand kinds become precise, and whether that split is
  acceptable for DTO reuse across queries.
- [ ] **Let generated query free functions run inside read-write
  transactions.** They currently take `*spanner.ReadOnlyTransaction`, so the
  same query cannot be issued from a `ReadWriteTransaction`. Consider
  accepting a small interface such as
  `interface { Query(context.Context, spanner.Statement) *spanner.RowIterator }`
  that both transaction types satisfy.
- [ ] **Drop the `params map[string]interface{}` argument for queries with no
  parameters.** Generated functions like `ListSingers` require callers to pass
  `nil` for a query that declares no parameters.
- [ ] **Fan out optional-parameter Variants in `plan-report`.**
  `plan-report` analyzes only the canonical all-on variant of an optional
  query (verified live: a query with two `omit_when_null` params produced one
  report target instead of four variants). `tools/optparam-plan-probe` already
  contains the per-variant plan acquisition loop that could be ported. This
  complements the existing "wire Variants into spanner-query-plan-shape" item
  above.
- [ ] **Propose a runtime image/digest API in spanemuboost for backend
  identity.** spanemuboost v0.4.0 exposes `RuntimePlatform` but no way to read
  the resolved container image or digest, so plan-report backend identity
  stays `not_recorded` unless supplied manually. The fix belongs in
  spanemuboost (for example `RuntimeImage`/`RuntimeImageDigest`), then
  plan-report can record `source: spanemuboost` evidence automatically.
- [ ] **Support applying database options in the Omni plan probes once Omni
  accepts them.** Verifying `use_unenforced_foreign_key_for_query_optimization`
  needs a database-option entry point in `tools/spanner-query-plan-shape` and
  `plan-report` (for example a `--database-option key=value` flag that pins
  the database ID and issues `ALTER DATABASE`, or rewriting the database name
  of an `ALTER DATABASE` statement in `--ddl` input). Blocked for now: Omni
  2026.r1-beta rejects this option via `ALTER DATABASE` (empty-message
  InvalidArgument) and does not parse the documented `SET DATABASE OPTIONS`
  syntax, while `version_retention_period` applies fine, so the gap is the
  option itself, not the harness. See
  `research/spanner-query-gen/PLAN_REPORT_OPERATOR_COVERAGE_2026-06-12.md`.

- [ ] **Migrate cmd/spanner-query-gen and tools modules to testcontainers
  v0.42+.** They pin the known-good spanemuboost v0.4.0 / testcontainers
  v0.40.0 pair because testcontainers v0.42 moved Docker types to
  github.com/moby/moby and breaks the WithConfigModifier callback signature
  in integration_test.go. Coordinate with a spanemuboost upgrade.

- [ ] **Consider surfacing testcontainers Docker-host discovery failures as
  errors instead of panics.** spanemuboost propagates the
  `rootless Docker not found` panic from testcontainers
  `MustExtractDockerHost`. A wrapped error with remediation hints (check
  `DOCKER_HOST`, docker context, properties file) would be friendlier for CLI
  users. This also belongs in spanemuboost.
