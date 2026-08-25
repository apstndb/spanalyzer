# Handoff — Spanner schema survey

## Retained in spanalyzer (2026-08-25)

The final tracked source tree is retained as the independently testable
`github.com/apstndb/spanalyzer/survey` nested module. The unpublished legacy
repository's 27-commit history is intentionally not published. The strict
[`import-provenance.json`](import-provenance.json) maps legacy commit
`91908d001349f844aac070cc6518119c0e3c36c0` and tree
`34d63cf89aaf885cbfd8069e91c4ead707b048c8` to the exact initial spanalyzer
import commit and subtree.

Ignored agent/runtime directories, the local managed-database locator, and the
untracked memefish feedback scratch file were excluded. The scratch file's two
actionable upstream issues are closed. The former local checkout is not
required for builds, tests, manifest regeneration, provenance checks, or
retained knowledge and can be deleted separately with explicit authority.

Last reviewed: 2026-08-25. All repository-local release-note and live-parity
follow-ups are implemented and committed. The latest work covers named-schema
DDL, omitted and zero-column primary keys, rolling metadata columns, metadata
type/ordinal validation, nullable locality options, vector-index filters,
view-backed property graphs, omitted change-stream retention, generic options,
live `SPANNER_SYS` audit, UUID metadata, live proto/enum and function types,
search-index type normalization, empty graph-label properties, and an aggregate
managed canonical-DDL comparison.

The managed canonical set also proves that `GetDatabaseDdl` can retain an
`ALTER TABLE ... ADD CONSTRAINT` statement. AST-to-schema conversion accepts
check and foreign-key additions and folds them into reconstructed `CREATE TABLE`;
all other `ALTER TABLE` variants remain fail-closed.

Managed Spanner and Omni `2026.r2.1-beta` both passed the current
`SPANNER_SYS` audit at 50 advertised tables and 539 columns against the
51-table package superset. The extra table is the officially documented
Enterprise-only `VECTOR_INDEX_STATS`, which neither tested target advertised.
Managed Spanner, Omni, and Emulator v1.5.56 agree on the UUID metadata contract:
`DATA_TYPE` is SQL `NULL`, `SPANNER_TYPE = UUID`, and
`COLUMN_DEFAULT = NEW_UUID()`. Managed and Omni expose custom dictionary and
table option rows; index `columnar_policy` remains in canonical DDL without an
`INDEX_OPTIONS` row. Emulator v1.5.56 has the narrower option surface recorded
in the regression tests.

The latest managed canonical comparison completes successfully: 330 canonical
statements, one memefish parse error in the `CREATE TABLE` family, and 341
reconstructed statements. Parsed family counts match for change streams,
functions, regular and search indexes, property graphs, proto bundles, roles,
schemas, sequences, views, and grants. At the family-count layer, canonical
`ALTER DATABASE` and `ALTER TABLE ... ADD CONSTRAINT` have no generated `ALTER`
counterpart, while reconstruction adds eleven `ALTER STATISTICS` statements and
one locality group, placement, and table. Commit `dd06e8a` preserves the added
constraint semantically by folding it into the generated `CREATE TABLE`.
Object identifiers are intentionally not retained or printed.

Primary-key omission remains environment-specific. Managed Spanner and Omni
create the documented hidden identity `rowid`; Emulator v1.5.56 rejects
omission. All three accept explicit `PRIMARY KEY ()` as a singleton table, with
a small metadata-shape difference that the converter handles. Nullable
locality-group option values are omitted during reconstruction, and malformed
emulator `inflash = 'BOOL'` metadata fails closed.

## Remaining boundaries

The remaining work is external or ongoing maintenance:

- memefish lacks AST shapes for additional vector-index keys, model
  privileges, named-schema property graphs, and qualified targets for some
  identifier-only families.
- Current `INFORMATION_SCHEMA` cannot recover sequence and schema grants,
  placement keys, optionless locality groups, or exact remote-function clauses.
- Omni commercial licensing, standalone VM packaging, and backup migration
  remain outside this container-backed schema-survey scope.

See [`UNSUPPORTED_DDL.md`](UNSUPPORTED_DDL.md) for the precise conversion and
live-recovery boundaries. Add new actionable work to [`TODO.md`](TODO.md)
rather than extending this dated handoff.

## Verification

Final verification on 2026-08-25 passed the uncached `mise run test-all` gate,
including Emulator v1.5.56 and Omni integration tests. Uncached managed drift
passed with 50/539 advertised `SPANNER_SYS` metadata against the 51-table
superset and 28 decoded sample rows. The explicit Omni drift gate passed with
the same 50/539 surface and three decoded rows. The managed UUID fixture passed
with cleanup, the read-only canonical comparison reproduced the 330/1/341
aggregate above, and `mise run lint` reported zero issues.

Common maintenance commands are:

```sh
mise run test-all
mise run run-roundtrip
mise run test-drift-real
mise run test-uuid-real
mise run test-canonical-real
```

The managed-Spanner commands require the configured, gitignored connection
environment. See [`AGENTS.md`](AGENTS.md) for package architecture, exact gate
semantics, runtime prerequisites, and the pitfalls that must remain visible to
implementers.

## Current INFORMATION_SCHEMA evidence

The current machine-readable target observations, explicit managed-primary
selection, and analyzer projection are described in the OKF
[`INFORMATION_SCHEMA managed-primary catalog`](../knowledge/observations/information-schema-managed-primary-catalog.md)
Observation. Raw captures live under [`infoschem/evidence`](infoschem/evidence/),
and the root `information_schema_projection_source.json` names the selected
managed capture by path and whole-file hash. Routine reruns update raw evidence
or the selector; they do not create another dated Markdown report unless a
material disagreement requires interpretation.

## Historical evidence

The dated Emulator-versus-managed metadata observations are retained in:

- [`docs/spanner-schema-diff-report-20260309.md`](docs/spanner-schema-diff-report-20260309.md)
- [`docs/spanner-schema-diff-report-20260429.md`](docs/spanner-schema-diff-report-20260429.md)
- [`docs/spanner-schema-diff-report-20260509.md`](docs/spanner-schema-diff-report-20260509.md)

These reports are historical observations, not current compatibility claims.
Current code behavior is defined by code, executable tests, and manifests;
current target evidence is always timestamp- or digest-scoped as described
above. Conversion boundaries remain explicit in `UNSUPPORTED_DDL.md`.
