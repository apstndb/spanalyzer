# TODO

## Export a portable SPANNER_SYS manifest (S-C)

- ~~Persist the current managed-Spanner and Omni `2026.r2.1-beta` column
  observations as strict, redacted captures with exact query text, target
  provenance, and columns-only content hashes.~~
- ~~Add separate prerelease JSON Schemas for capture input and manifest output;
  keep dates, runtime versions, query text, and source commit outside the
  manifest content hash.~~
- ~~Add one public `spannersys.ExportManifest(sourceCommit)` byte API and a
  single-purpose CLI with an explicit 40-hex source commit. Do not perform a
  live probe or Git lookup and do not publish descriptor DTOs.~~
- ~~Fail closed on missing required targets, unknown or malformed capture data,
  target presence/type/ordinal disagreement, and canonical renderer mismatch.
  Project only the current 50-table / 539-column intersection; keep the eight
  absent-on-both registry columns default-denied.~~
- ~~Preserve the current official-document conflicts as non-projecting evidence
  and prove deterministic export, strict decoding, redaction, evidence counts,
  and hash boundaries in tests.~~
- ~~Run focused and repository validation, obtain exact-diff review through
  `agmsg`, and commit S-C separately. Do not check a full manifest into this
  repository; spanalyzer will export and pin it from the exact S-C commit.
  Focused tests, full Docker-backed tests, build, vet, lint, and short race pass;
  both teams returned FINAL READY on the exact staged implementation.~~ Commit
  remains the separate final action for this slice.

## Retain SPANNER_SYS live metadata (S-A)

- ~~Retain each advertised `SPANNER_TYPE` and `ORDINAL_POSITION` alongside the
  discovered SPANNER_SYS table and column name.~~
- ~~Report declaration-order drift and registry entries absent from a complete
  target observation without treating missing capture data as observed absence.~~
- ~~Change the active partitioned DML fingerprint and progress decoder fields to
  the live-observed string contract, with synthetic decode coverage.~~
- ~~Keep the structural reflect renderer, persisted target captures, and portable
  manifest exporter out of this change; those are the separately reviewed S-B
  and S-C slices.~~
- ~~Run focused and repository validation.~~ Managed and Omni each passed at
  50 tables / 539 columns with the same eight known-absent registry entries;
  lint, build, vet, full tests, and short race also passed with Go 1.26.6.
- ~~Request exact-diff review through `agmsg`, and commit only after the S-A
  boundary and findings converge.~~ Both teams returned READY on the final
  50/539/8 non-vacuity snapshot, committed as `f91b8fb`.

## Derive canonical SPANNER_SYS structural types (S-B)

- ~~Add private survey-local descriptor DTOs and a bounded registry extractor
  for all 51 tables / 547 columns without adding an exporter or schema file.~~
- ~~Record decoder pointer nullability separately from structural type identity;
  render canonical types through an explicit scalar/array/STRUCT mapping.~~
- ~~Fail closed on unsupported Go shapes, unsupported array nesting, missing or
  duplicate `spanner` tags, and malformed descriptor combinations.~~
- ~~Compare retained live raw types through a pure validator, attach general type
  mismatches to `AuditReport`, and remove the S-A-only two-column type map.~~
- ~~Prove 14 canonical shapes, the five nested STRUCT families, fully expanded
  interval tables, and managed/Omni equality over the advertised 50/539 surface.~~
  Both live targets passed the general comparator with zero type drift.
- ~~Run lint, build, vet, full uncached tests, and short race with Go 1.26.6.~~
- ~~Obtain exact-diff reviews before a separate S-B commit.~~ Both teams
  returned FINAL READY after converging on private DTOs and Audit-owned drift
  reporting with a separately testable pure comparator.

## Audit current Spanner and Spanner Omni release notes (2026-08-24)

- ~~Fetch and read the complete current managed-Spanner and Spanner Omni release
  notes from official sources, without relying on generated summaries.~~ Done:
  checked the full current documents and the live official pages (updated
  2026-08-11 and 2026-08-20 respectively).
- ~~Map every recent DDL, INFORMATION_SCHEMA, runtime, and compatibility change to
  existing repository evidence and identify unverified probe candidates.~~ Done:
  release-note-driven gaps and already-covered surfaces are classified below.
- ~~Classify candidates by repository-local feasibility, required environment,
  and priority; do not change pins or product-facing code during the audit.~~
  Done; no pin or product-facing implementation was changed by the audit.

### Release-note-driven follow-up

- ~~Add the officially documented Enterprise-only
  `SPANNER_SYS.VECTOR_INDEX_STATS` table to the package superset, with an exact
  query helper and registry/type tests.~~ Done: the 51-table registry keeps this
  documentation-backed entry distinct from the tested managed/Omni 50-table
  advertised surface, where it is absent.

- ~~Support the managed canonical `ALTER TABLE ... ADD CONSTRAINT` form without
  broadening conversion to destructive or stateful ALTER operations.~~ Done in
  `dd06e8a`: check and foreign-key additions fold into table metadata; missing
  targets, primary-key additions, and all other `ALTER TABLE` variants fail
  closed.

- ~~**P0 — pre-filtered vector indexes:** preserve `CreateVectorIndex.Where` in
  `Indexes.Filter` in both directions and fail closed on additional key
  columns.~~ Done in `4fdd3a3`; the filter round-trips and the memefish v0.8.1
  one-key limitation cannot silently discard service-supported keys.
- ~~**P0 — property graphs backed by SQL views:** add AST and live metadata
  coverage for the official `GraphView KEY(id)` form.~~ Done in `ce82131`; view
  output columns now supply implicit graph properties.
- ~~**P1 — change-stream seven-day default:** probe an omitted
  `retention_period` on managed Spanner, Omni, and Emulator.~~ Done in `6f96672`;
  all three targets preserve omission, so reconstruction does not materialize
  the inherited default.
- ~~**P1 — generic option surfaces:** live-probe dictionary and
  `columnar_policy` options.~~ Done in `1032944`; managed Spanner and Omni expose
  dictionary/table options but omit index rows, while Emulator v1.5.56 has the
  narrower documented compatibility boundary. Shared database options were not
  mutated merely to probe SQL NULL behavior.
- ~~**P1 — SPANNER_SYS drift:** add a read-only live column/decode audit for
  managed Spanner and Omni.~~ Done in `3d0689a`; both targets matched the
  50-table / 539-column registry and all available sample rows decoded.
- ~~**P2 — UUID:** verify metadata and reconstruction on managed Spanner, Omni,
  and Emulator.~~ Done in `a13aeb2`; all three report SQL NULL `DATA_TYPE`,
  `SPANNER_TYPE = UUID`, and `COLUMN_DEFAULT = NEW_UUID()` and reconstruct the
  primary key.
- Omni `2026.r2.1-beta` commercial-edition licensing, standalone VM packaging,
  and backup migration are outside this repository's container-backed schema
  survey. `spanemuboost` does not use either deprecated `spanner start` flag;
  do not add those operational probes unless the repository scope expands.

## Verify primary-key omission on Cloud Spanner Emulator (2026-08-24)

- ~~Run the same omitted-key, zero-column-key, canonical DDL, metadata, and insert
  behavior probe against Cloud Spanner Emulator v1.5.56.~~ Done: omission is
  rejected; explicit zero-column keys are accepted and retain singleton behavior.
- ~~Compare the result with the already verified managed-Spanner and Omni behavior
  and record the exact compatibility boundary.~~ Done: the emulator additionally
  exposes an `INDEXES.PRIMARY_KEY` row for the zero-column key, but no
  `INDEX_COLUMNS`; conversion coverage accepts both shapes.

## Reconcile tables without explicit primary keys (2026-08-24)

- ~~Fetch and read the full official tables-without-primary-keys documentation,
  then identify the exact accepted DDL forms and documented generated-key
  contract.~~ Done: omission generates a hidden identity `rowid`; explicit
  `PRIMARY KEY ()` instead defines a singleton zero-column key.
- ~~Inspect memefish v0.8.1 and the repository conversion paths for whether a
  missing `PRIMARY KEY` can be parsed, represented, and regenerated without
  inventing or losing service-generated metadata.~~ Done: omission is now
  preserved without invented metadata, and the explicit empty-key form uses a
  service-equivalent `ast.TablePrimaryKey` representation.
- ~~Run bounded create / `GetDatabaseDdl` / `INFORMATION_SCHEMA` / `LoadSchema`
  probes against Spanner Omni `2026.r2.1-beta` and the configured managed
  Spanner database, using collision-safe temporary names and guaranteed cleanup.~~
  Done: both environments agreed; managed temporary tables were removed and
  the Omni container was torn down.
- ~~Classify any managed/Omni divergence, document the exact metadata rows needed
  by `astconv`, and identify the smallest repository-local implementation and
  regression-test scope without committing or publishing it.~~ Done: no product
  divergence; conversion and regression coverage were committed in `86c8870`.

## Validate latest Spanner Omni release (2026-08-24)

- ~~Fetch and read the complete official Spanner Omni release notes, identify the
  newest available version/image and enumerate all changes since the version
  currently used by this repository's test harness.~~ Done: latest is
  `2026.r2.1-beta`; all three listed updates and the no-in-place-upgrade rule are
  recorded in `HANDOFF.md`.
- ~~Resolve the actual Omni image selected by `spanemuboost` and record image
  digest/platform provenance without changing repository dependencies.~~ Done:
  at audit time v0.4.6 still defaulted to `2026.r1-beta.2`; the module was later
  updated to spanemuboost v0.4.7. The retained INFORMATION_SCHEMA capture under
  `infoschem/evidence/omni/` now records the observed linux/arm64 OCI digest.
- ~~Run the repository Omni drift gate and a bounded loader/DDL fixture against
  the newest image, distinguishing product regressions from container-runtime
  or readiness failures.~~ Done: both old and new images start; latest accepts
  and returns the fixture. The locality NULL decode issue found here was later
  resolved in `3217001`.
- ~~Compare the observed Omni metadata surface with the current registry,
  emulator v1.5.56, and the latest real-Spanner observations, then record the
  verified outcome without committing or publishing it.~~ Done: latest Omni is
  48/308 and advertises both new columns; a retained same-transaction probe
  records both as queryable in Omni and in the selected managed observation.
  Emulator v1.5.56 remains 28/198 without them, while managed-service rollout
  remains observation-time dependent.
- ~~Before registering the two rolling columns, design the real-Spanner path for
  columns that are advertised but not queryable in the same session/snapshot.~~
  Done in `d9778d4`: rolling columns are probed before selection, and failure to
  query one column does not poison loading of the stable surface.

## Validate latest environment state (2026-08-23)

- ~~Verify the latest published Cloud Spanner Emulator release and container
  image from current primary sources.~~ Done: stable v1.5.56 remains latest;
  the retained capture under `infoschem/evidence/emulator/` records the tested
  linux/arm64 OCI digest.
- ~~Refresh `origin/main` and the latest public memefish module version without
  disturbing the existing dirty worktree.~~ memefish remains v0.8.1. The local
  checkout has no remotes, and the expected GitHub repository API returned 404,
  so no hosted `main` comparison is available.
- ~~Re-run cache-disabled real-Spanner drift and the expanded read-only metadata,
  DDL parsing, and loader probe against the configured instance.~~ Done: the
  required drift gate now fails on newly advertised `INDEXES.SEARCH_UNNEST`;
  `INDEX_COLUMNS.EXPRESSION` was also observed during the wider probe.
- ~~Run the same bounded metadata and locality-group fixture against the latest
  emulator release, compare it with the real instance and the 2026-08-07
  baseline, and record only current observed evidence.~~ Done: v1.5.56 remains
  28/198 and lacks both columns; the then-observed locality/NULL issues were
  subsequently resolved or made fail-closed in `3217001`.
- ~~Design drift/discovery handling for the rolling service surface and register
  nullable coverage for `INDEXES.SEARCH_UNNEST` and
  `INDEX_COLUMNS.EXPRESSION`.~~ Done in `d9778d4` with per-column queryability
  probes, nullable struct fields, and loader/registry regressions.

## Refresh real-Spanner investigation (2026-08-07)

- ~~Re-run the required real-Spanner drift gate without the Go test cache.~~
  Passed.
- ~~Re-capture the current INFORMATION_SCHEMA table/column/type surface and
  registry coverage without retaining database identifiers.~~ Done: 48 tables /
  306 columns, exact name/type coverage; six registry ordinal mismatches were
  newly detected in `PARAMETERS` and `ROUTINES`.
- ~~Re-check row-level locality-group option nullability, view/change-stream
  option shapes, `GetDatabaseDdl` parsing, and the `LoadSchema` boundary.~~ Done:
  all previously measured counts and the locality NULL decode failure persist.
- ~~Compare the fresh observations with the 2026-08-05 snapshot and record only
  verified changes or confirmed persistence in the handoff.~~ Done; no verified
  real-instance change was found in the previously measured surface.
- ~~Correct the six `ColumnMeta.OrdinalPosition` values and extend drift coverage
  to compare registered types and ordinals.~~ Done in `fa0db9a`.

## Validate emulator v1.5.56 against real Spanner (2026-08-05)

- ~~Compare v1.5.56 and the configured real database INFORMATION_SCHEMA table
  and column surfaces, and verify registry coverage for both.~~ Done: the
  emulator exposed 28 tables / 198 columns and real Spanner exposed 48 / 306;
  all common table column sets matched and neither target had registry-unknown
  tables or columns.
- ~~Exercise row-level loading for locality-group options and view security on
  v1.5.56, then compare the result with the observed real-database shape.~~
  Done: v1.5.56 preserved `DEFINER` view security and a non-NULL
  `partition_mode` row, but reported requested `storage = 'hdd'` as
  `inflash = 'BOOL'`; real Spanner reported `storage = 'ssd'` plus nullable
  `ssd_to_hdd_spill_timespan` rows.
- ~~Record environment-specific results without persisting database identifiers;
  do not update the repository's emulator pin until validation is complete.~~
  Done in `HANDOFF.md` and a sanitized report under `/private/tmp`; current
  emulator drift and loader fixtures explicitly use v1.5.56.
- ~~Classify emulator `inflash = 'BOOL'` and make locality-group option values
  nullable for live loading.~~ Done in `3217001`: SQL NULL rows are omitted,
  while the malformed emulator sentinel fails closed instead of being emitted
  as DDL.

## Refresh real-Spanner comparison (2026-07-28)

- ~~Re-run the required real-Spanner drift gate against the configured
  database.~~ Passed with Go test caching disabled.
- ~~Capture a current, read-only registry-versus-instance comparison that
  distinguishes unknown service tables/columns from expected superset
  entries.~~ Exact match: 48 tables and 306 columns on each side; no unknown
  or absent tables or columns.
- ~~Record the result in the handoff and keep database identifiers out of
  artifacts and user-facing output.~~ Done.
- ~~Compare the canonical `GetDatabaseDdl` statement set with a fresh
  `LoadSchema` → `ToDDLStatements` reconstruction, reporting only aggregate
  family/count differences.~~ Done in `bb7a38b`: the current managed snapshot
  has 330 canonical statements (one unparsed `CREATE TABLE`) and 341 generated
  statements; all successfully parsed families are reported without object
  identifiers. The single canonical `ALTER TABLE` is an added table constraint.
- ~~Define the locality-group SQL NULL contract and add a real-shaped loader
  regression.~~ Done in `3217001`; the canonical comparison now completes.

## Address named-schema pre-commit review (2026-07-26)

- ~~Fix the synthesized referenced-unique-constraint identity so same-leaf child
  tables in different schemas cannot merge their referenced key-column rows.~~
  Done by including the child schema in the internal synthesized identity.
- ~~Add a regression with same-leaf cross-schema children referencing different
  columns of one shared parent.~~ Done.
- ~~Evaluate and, if supported by the current emulator, add a focused
  named-schema `LoadSchema` round-trip fixture for parent/interleave/index
  metadata shape.~~ Done for named parent/child table interleave plus a
  non-interleaved regular index, view, and sequence.
- ~~Probe raw schema-qualified `INTERLEAVE IN` index DDL on emulator v1.5.55 and
  the configured real Spanner database to distinguish a service restriction
  from memefish v0.8.1's unqualified AST field.~~ Done: both accepted qualified
  regular-index interleave, so the remaining gap is the upstream AST field.
- ~~Re-run focused tests and repository gates, then request a same-reviewer
  remaining-blockers verdict over `agmsg`.~~ The same reviewer returned READY
  on 2026-07-26 with no remaining blockers; Omni remained environment-blocked.

## Implement all repository-local DDL gaps (2026-07-26)

- ~~Implement named-schema views and sequences in both AST directions, including
  schema-qualified exact grants and qualified-identity collision coverage.~~
  Done with AST/metadata round-trip and collision tests.
- ~~Implement named-schema regular/search indexes while enforcing the
  table/index same-schema invariant and preserving fail-closed interleave
  behavior where metadata cannot be represented.~~ Done with invariant and
  qualified-identity tests.
- ~~Implement named-schema tables, foreign-key targets, interleave parents,
  synonyms, constraints, options, and dependency ordering without leaf-name
  collisions.~~ Done, including cross-schema foreign keys and same-schema
  interleave/synonym validation.
- ~~Implement named-schema targets for every path-backed grant/revoke family
  whose service object supports named schemas.~~ Done for tables, views,
  sequences, change streams, and table functions, with qualified revoke tests.
- ~~Validate named-schema function syntax against the configured real Spanner
  database; implement it only if the service accepts the syntax, and preserve
  exact routine identity in parameters, options, and grants.~~ Done: the
  2026-07-26 probe accepted the function and preserved routine/specific schemas;
  all temporary objects were removed.
- ~~Validate `SCHEMATA.PROTO_BUNDLE` behavior with multiple schemas and remove
  the non-default-row rejection when union semantics are confirmed.~~ Done:
  the service currently populated only the default-schema row; local conversion
  now safely unions, deduplicates, and sorts any non-empty rows.
- ~~Update unsupported-surface documentation and handoff state to retain only
  upstream AST/parser blockers and live metadata information loss.~~ Done.
- ~~Run targeted AST/schema regressions, `mise run test-all`, the required
  real-Spanner drift gate, short race tests, vet, module verification,
  formatting, and diff hygiene.~~ Done: all passed on 2026-07-26.

## Commit remediation and reassess remaining gaps (2026-07-25)

- ~~Audit the current remediation diff, exclude temporary artifacts, and confirm
  that every tracked and new source/test/document file belongs to the intended
  commit.~~ Done: `MEMEFISH_FEEDBACK.md` is the sole temporary artifact and is
  excluded.
- ~~Re-run the repository gates, stage files explicitly, and commit the verified
  remediation without pushing or opening a pull request.~~ Done in this commit:
  default gate, required real drift, short race, vet, module verification,
  formatting, and diff hygiene all passed.
- ~~Reassess each remaining unsupported or partial surface against memefish
  v0.8.1, current repository models, and observed real-Spanner metadata.~~ Done:
  the current upstream main is still v0.8.1, and a read-only real metadata probe
  confirmed the live-recovery gaps recorded in `UNSUPPORTED_DDL.md`.
- ~~Record which gaps are implementable locally and which remain blocked by an
  upstream AST or INFORMATION_SCHEMA contract.~~ Done: the named-schema matrix
  now distinguishes local path-backed work, upstream `Ident`/missing-node
  blockers, and live metadata loss.

## Remaining external boundaries after local implementation

- Do not treat vector indexes, change-stream creation, models, property graphs,
  placements, locality groups, or statistics packages as named-schema parser
  work until current Spanner service support is established. Their current
  memefish AST object names cannot represent qualification.
- Model grants require a new memefish privilege node.
- Revisit live recovery only when Spanner exposes schema/sequence grant,
  placement-key, optionless-locality-group, or exact remote-function metadata.

## memefish v0.8.1 and real-Spanner probe (2026-07-23)

- ~~Inspect the memefish v0.8.0..v0.8.1 parser and AST changes and define bounded
  real-Spanner probe cases for any newly representable DDL.~~ Done: v0.8.1 adds
  placement-key and sequence-grant AST nodes.
- ~~Upgrade the module to memefish v0.8.1 and adapt conversion code, tests, and
  unsupported-surface documentation where support changed.~~ Done: AST-only
  placement keys and sequence grants/revokes are wired with regression tests.
- ~~Run the bounded probes against the configured real Spanner database without
  recording resource identifiers, and clean up all temporary objects.~~ Done:
  all temporary role, sequence, and schema objects were removed; the placement-
  key table was rejected before creation by semantic/configuration validation.
- ~~Run the relevant unit, drift, race, lint, vet, and build gates and record the
  verified result.~~ Done: targeted v0.8.1 tests, `mise run test-all`, required
  real-Spanner drift, short race, vet, module verification, and diff checks all
  passed.

## Active repository review (`f1568ba`)

- ~~Review the full tracked repository snapshot for correctness, data loss,
  panic/error paths, and cross-package contract mismatches.~~ Done: blocking
  correctness and decode-contract findings confirmed.
- ~~Independently review AST conversion, schema loading/metadata, and
  tests/integration/docs with GPT-5.6 Sol/high subagents, without a Fast or
  priority service-tier override.~~ Done: three read-only lanes completed;
  effective child routing remains unverified without a harness receipt.
- ~~Run the repository validation gates that are available locally and record
  environment-limited checks explicitly.~~ Done: lint, build, vet, uncached
  full tests, short race tests, real-Spanner drift, and targeted reproductions
  passed or reproduced as recorded in the report.
- ~~Save a consolidated review report under `/private/tmp` and close this section
  with the verified result.~~ Done: report at
  `/private/tmp/spanner-emulator-survey-review-f1568ba.md`; the subsequent
  remediation is tracked in the next section.

## Active review remediation (`f1568ba`)

- ~~Preserve column-scoped table privileges without widening them, reconstruct
  live grants from `ROLE_COLUMN_GRANTS`, and implement supported `REVOKE`
  semantics.~~ Implemented with focused grant/revoke regression coverage.
- ~~Preserve remote-function language and determinism metadata and prefer exact
  `SPANNER_TYPE` parameter and return types.~~ Implemented for AST-originated
  functions; live metadata that cannot encode required remote clauses remains
  an explicit unsupported error rather than guessed DDL.
- ~~Preserve anonymous constraints without empty identifiers or map collisions,
  reconstruct composite foreign-key column order correctly, include foreign-key
  dependencies in table ordering, and use schema-qualified table identities.~~
  Implemented with constraint and ordering regression coverage; the later
  2026-07-26 work extended the same qualified identity through named-schema
  table reconstruction.
- ~~Propagate row-deletion-policy parse errors and restore identity-column kind.~~
  Implemented with focused table-conversion coverage.
- ~~Preserve locality-group option values and empty locality groups.~~ Implemented
  for AST round trips and option-bearing INFORMATION_SCHEMA rows; a live source
  still cannot enumerate an optionless locality group.
- ~~Correct `SPANNER_SYS` decode contracts for active partitioned DML, operations
  by table, and locality-group table sizes; validate interval inputs and prevent
  callers from mutating metadata registries through accessors.~~ Implemented
  with row-decoding, query-validation, and registry-isolation tests.
- ~~Make `LoadSchemaFromDiscovered` safe against stale or incomplete discovery
  maps.~~ Implemented: it rediscovers columns inside the read transaction
  instead of trusting the caller's map.
- ~~Fail explicitly, with current unsupported-surface documentation, where
  memefish cannot represent model grants or named-schema objects rather than
  silently losing or dequalifying information.~~ Implemented; locally
  representable named-schema families were subsequently enabled, while the
  remaining AST-blocked families stay documented as fail-closed.
- ~~Add focused regression/decode tests, strengthen loader and drift assertions,
  pin the linter toolchain, and make real-Spanner test skipping explicit.~~
  Implemented; `mise run test-drift-real` is now a separate required gate.
- ~~Run targeted tests, full uncached tests, short race tests, lint, vet, build,
  and a final audit against every confirmed review finding.~~ Done: pinned lint,
  build, vet, uncached `go test -count=1 ./...`, short race, emulator drift,
  Omni drift, and the required real-Spanner drift gate all passed.

## Active review

- ~~Review the current `main` snapshot (`364995f`) for correctness, data-loss,
  and missing-test risks.~~ Done: four findings recorded, including three
  source-verified incorrect-output paths.
- ~~Save the full review report under `/tmp` and send the verdict and findings to
  the `spanner-emulator-survey` team through `agmsg`.~~ Done: sent to
  `claude-code`; report at `/private/tmp/spanner-emulator-survey-review-364995f.md`.

## Repository work

- ~~Implement reverse mapping from `INFORMATION_SCHEMA.TABLE_OPTIONS` to
  `CREATE TABLE ... OPTIONS(locality_group=...)`.~~ Done: `TABLE_OPTIONS` added to
  `AllTableMetas()` and `cmd/roundtrip` reads it. Recovery works on real Spanner;
  the emulator does not expose `TABLE_OPTIONS` yet.
- ~~Add live Production Spanner integration coverage (behind a build tag or env var)
  to verify `infoschem.TableMeta` registry against true rolling release schema.~~
  Done: Added `TestDrift_RealTableMetas` which runs against a real database if
  `TEST_REAL_SPANNER_DATABASE` is set (can be configured in `mise.local.toml`).
- ~~Automate schema diff regeneration or add drift tests against
  `infoschem.AllTableMetas()`.~~ Done: `TestDrift_*` tests in
  `infoschem/drift_test.go` now use dynamic column discovery to verify that our
  local `TableMeta` registry covers all columns in the target database
  (Emulator/Omni/Real).
- Maintain `infoschem.AllTableMetas()` by responding to warnings from `mise run test-all`
  or `mise run run-roundtrip` when unknown columns are detected in the wild.
- ~~Refactor `cmd/roundtrip` to use `astconv.LoadSchema` instead of inline
  `INFORMATION_SCHEMA` query code.~~ Done.
- ~~Add an E2E round-trip test using `astconv.LoadSchema` (emulator).~~ Done:
  `astconv/loader_test.go`.
- ~~Document named-schema support status per DDL family.~~ Done:
  `UNSUPPORTED_DDL.md`.
- ~~Upgrade memefish to v0.8.0.~~ Done: `go.mod` updated, build and tests pass.
- ~~Implement anonymous table constraint style `PRIMARY KEY` in `astconv`
  (`ast.TablePrimaryKey`).~~ Done.
- ~~Implement `GRANT ... ON ALL ... IN SCHEMA` in `astconv`
  (`PrivilegeOnAllTablesInSchema`, `SelectPrivilegeOnAllChangeStreamsInSchema`,
  `SelectPrivilegeOnAllViewsInSchema`).~~ Done.
- ~~Simplify `spannertype.ParseSchemaType` to use the newly exported
  `memefish.ParseSchemaType`.~~ Done.
- ~~Fix model column `DataType` panic and per-column OPTIONS loss
  (`to_ast_model.go`).~~ Done.
- ~~Fix function `ParseExpr` error swallowing (`to_ast_function.go`).~~ Done.
- ~~Fix self-referential FK reference table resolution.~~ Done.
- ~~Validate invalid `AllSchemaGrant` combinations (`to_ast_role.go`).~~ Done.
- ~~Make `LoadSchema` / `LoadSchemaFromDiscovered` snapshot-consistent with a
  single `ReadOnlyTransaction`.~~ Done.
- ~~Remove dead `fromCreateTablePK` code.~~ Done.
- ~~Sync `AGENTS.md` versions and add regression tests.~~ Done.
- ~~Implement `REVOKE` handling for `ON ALL ... IN SCHEMA` (currently `fromRevoke`
  is a no-op).~~ Done as part of the `f1568ba` remediation; supported revoke
  forms now remove only matching grants.

## Upstream follow-up

- Track memefish support for named-schema property graphs so qualified graph and
  element names can round-trip.
- ~~Track memefish support for `GRANT USAGE ON SCHEMA`.~~ Done: AST round-trip
  implemented via `SchemaGrant` and `Schema.SchemaGrants`. Recovery from
  `INFORMATION_SCHEMA` alone is still blocked because no corresponding table is
  exposed by the emulator.
- ~~Track memefish support for `GRANT ... ON SEQUENCE` /
  `ON ALL SEQUENCES IN SCHEMA` and `PLACEMENT KEY` (`#193`).~~ Done in memefish
  v0.8.1; track the remaining INFORMATION_SCHEMA live-recovery gaps instead.
