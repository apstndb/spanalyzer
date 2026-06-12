# Resolved TODO Archive (snapshot 2026-06-12)

Completed items and historical coordination notes moved out of the root
`TODO.md` during the 2026-06-12 documentation reorganization. Resolutions are
preserved verbatim; git history before this date has the original layout.

## Coordination Notes (May 2026)

- The 2026-05-09 other-agent progress has been committed and the build is back
  to green as of 2026-05-14.
- The previously-noted `list_features.go` blocker is no longer present in the
  tree.

## Outstanding Issues In The Working Tree (2026-05-09)

These were the merge blockers found in the 2026-05-09 working tree, all fixed
by 2026-05-14:

- [x] **Build is broken in `internal/querygen`.** Fixed by replacing the stale
  `allNonKeyColumnNames(table, keyColumns)` call with the existing
  `nonKeyColumnNames(selectableColumnNames(table), keyColumns)` helper.
- [x] **`gofmt` failure in `internal/querygen/querygen_test.go`.** Fixed by
  formatting the querygen package.
- [x] **Stray files in the working tree.** Scratch `.bak` files, a built
  binary, and `all_features.txt` were removed; `testdata/protos/complex/` was
  intentionally kept as a fixture for
  `TestAnalyzerRowTypeForComplexProtoBundle`.
- [x] **`plan_report.go` kept a local variable named `source` after renaming
  the field to `Catalog`.** Renamed to `catalog`.

## Analyzer Behaviour

- [x] **`resultconv.go` swallowed non-`THEN RETURN` DML errors as
  `ErrStatementHasNoRowType`.** `Returning()` errors are now wrapped, while a
  missing returning clause still maps to `ErrStatementHasNoRowType`.

## Documentation Drift (resolved)

- [x] Updated the README `EXTERNAL_QUERY` limitation (the TVF is kept in the
  analyzed SQL and fed prepared row types; it is no longer rewritten to an
  empty subquery).
- [x] Added a BigQuery `EXTERNAL_QUERY` example with connection-specific
  external DDL, proto fields inside the Spanner inner query, and rejected
  unsupported result values.
- [x] Documented the `ML.PREDICT` analyzer-only model (schema only, model
  output columns followed by non-duplicated input relation columns, no
  prediction execution).

## BigQuery EXTERNAL_QUERY (resolved)

- [x] Replaced the prepared-row-type handoff with direct
  `TVFInputArgumentType.ScalarExpr()` literal extraction.
- [x] The third `EXTERNAL_QUERY` options argument is intentionally rejected
  (README note tracked separately as an open item).
- [x] Added coverage for nested / repeated `EXTERNAL_QUERY` calls; lookup is
  by connection ID plus inner SQL rather than sequential consumption.
- [x] PostgreSQL-dialect Spanner inner SQL stays explicitly unsupported.
- [x] `ParseDebugString` and `Unparse` remain parse-only helpers (decision).

## Proto And Enum Types (resolved)

- [x] Expanded descriptor-set tests beyond the order proto sample (multiple
  files, imports, packages).
- [x] BigQuery surfaces reject unsupported Spanner proto/enum/struct/tokenlist
  result columns with descriptive errors (decision).

## Spanner DDL And Catalog Semantics (resolved)

- [x] Property graph coverage extended to arbitrary property expressions and
  dynamic labels/properties. The known incorrect `JSON` fallback for
  arbitrary property expressions remains an open item in `TODO.md`.
- [x] Audited remaining DDL forms; change streams, database options, and IAM
  objects are ignored with tests. (`CREATE SEARCH INDEX` joined the ignored
  set on 2026-06-12.)

## BigQuery Catalog Semantics (resolved)

- [x] `CREATE TABLE FUNCTION`, `CREATE PROCEDURE`, and `EXPORT DATA` are
  ignored/bypassed.
- [x] BigQuery result nullability stays `NULLABLE` (analyzer does not expose
  it; decision).
- [x] Added BigQuery dialect feature preset tests.

## spanner-query-gen (resolved)

- [x] Generated runtime query methods for `one` / `maybe_one` / `many`.
- [x] Reviewed generated methods: surgical imports, Spanner/BQ/Both targets
  compile, golden tests per cardinality.
- [x] Both iterators and `...All` slice loaders are generated (decision).
- [x] `one` / `maybe_one` enforce cardinality by checking for a second row.
- [x] `row_count` and `row_set` result modes model DML execution separately.
- [x] `QueryCodegenWrite` uses `InsertColumns` and `UpdateColumns` (decision).

## Optional Query Parameters (resolved subset)

- [x] `EmitGoBuilder` integrated into `GenerateQueryCode`; optional queries
  emit `<Name>Params` structs and `Build<Name>SQL` helpers.
- [x] Strict v1alpha validation for `choices` / `default` key names.
- [x] `orderby_choice` support on `kind: table`.
- [x] `OPTIONAL_PARAMS_DESIGN.md` prototype/scope sections updated.
- [x] PoC worktree and branch removed (verified 2026-06-12).
- [x] Stray `.bak` files removed.
