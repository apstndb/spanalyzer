# Survey Module Guidance

Guidance for coding agents working in the nested
`github.com/apstndb/spanalyzer/survey` module.

## Task tracking

- Before proceeding with unfinished work, write the remaining actionable tasks
  to `TODO.md` at the module root and update that file as tasks are
  completed or superseded.

## Project purpose

This module round-trips Cloud Spanner DDL through `INFORMATION_SCHEMA`:
DDL string → memefish AST → `infoschem` structs (mirror of `INFORMATION_SCHEMA` rows) → DDL string.
The same `infoschem` structs can be populated by querying a live database (real or emulator)
and converted back to DDL via `astconv`.

Scope is bounded by what `GetDatabaseDdl` (`gcloud spanner databases ddl describe`) emits:
primarily `CREATE` statements, `GRANT`, `ALTER DATABASE`, and `ALTER STATISTICS`. Managed
Spanner can also retain `ALTER TABLE ... ADD CONSTRAINT`; AST-to-schema conversion folds its
check or foreign-key constraint into the target table metadata. Other `ALTER` operations and
all `DROP`/`RENAME` statements are intentionally out of scope. Items that cannot be expressed
(no memefish AST node, or INFORMATION_SCHEMA loses information) are tracked in
`UNSUPPORTED_DDL.md` with linked upstream issues — keep that file current when changing
capability.

## Build, test, run

We use `mise` to define and run project tasks. Run `mise tasks` to see all available tasks.

```sh
mise run build               # Build all packages
mise run test                # unit tests; emulator is started in-process
mise run test-drift-emulator # Run drift test against Cloud Spanner Emulator
mise run test-drift-omni     # Run drift test against Spanner Omni
mise run test-drift-real     # Required real-Spanner drift gate; fails unless
                             # TEST_REAL_SPANNER_DATABASE is set (e.g. in a
                             # gitignored `mise.local.toml` [env] block)
mise run test-uuid-real      # Managed-Spanner UUID metadata/reconstruction fixture;
                             # uses a collision-safe temporary table and verifies cleanup
mise run test-canonical-real # Read-only aggregate GetDatabaseDdl versus reconstructed DDL comparison
mise run run-roundtrip       # end-to-end demo: starts emulator via spanemuboost,
                             # creates sample DDL, queries INFORMATION_SCHEMA,
                             # rebuilds DDL, prints generated SQL
GOWORK=off GOFLAGS=-mod=readonly go run ./cmd/infoschema-capture \
  --target managed --write --repo-root ..
GOWORK=off GOFLAGS=-mod=readonly go run ./cmd/infoschema-capture \
  --target emulator \
  --image gcr.io/cloud-spanner-emulator/emulator:1.5.56@sha256:5b1e3607fe8574fb04144eeabfa54120559fb01968ffe3ffc0a9a8f6776fc454 \
  --write --repo-root ..
GOWORK=off GOFLAGS=-mod=readonly go run ./cmd/infoschema-capture \
  --target omni \
  --image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta@sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741 \
  --write --repo-root ..
```

`--write` creates but never replaces the canonical capture path. An equivalent
rerun preserves the first artifact only when its producer-source and invocation
hashes also match. If either producer identity changes, the command fails
closed and reports both identities; review or remove the retained file before
recapturing rather than treating stale provenance as a successful rerun.

`cmd/roundtrip` requires Docker (uses `spanemuboost` / testcontainers).

## Package structure

- `spannertype/` — parses `SPANNER_TYPE` strings (e.g. `STRING(MAX)`, `ARRAY<INT64>`) into
  `ast.SchemaType`. `ParseSchemaType` delegates to `memefish.ParseSchemaType`, which is
  exported since memefish v0.8.0. Reverse direction is just `ast.SchemaType.SQL()`.

- `infoschem/` — Go struct mirrors of all 48 `INFORMATION_SCHEMA` tables (`tables.go`),
  plus `TableMeta`/`ColumnMeta` registry (`meta.go`).
  `DiscoverColumns(client)` queries `INFORMATION_SCHEMA.COLUMNS` at runtime to
  identify which columns are actually present in the target database.
  `TableMeta.Query(discovered)` then emits a `SELECT` that lists only those
  columns, ensuring safety across Emulator, Omni, and Real Spanner without
  hardcoded target-specific logic.
  If a column exists in the database but is missing from our `TableMeta`
  registry, a `WARNING` is logged via `infoschem.WarnUnknownColumns`. Use
  `spanner.SelectAll` with `spanner.WithLenient()` for robustness. Structs hold
  the **superset** of all known columns. `AllTableMetas` and `TableMetaByName`
  return defensive copies; callers must not assume they can mutate the registry.
  `LoadSchemaFromDiscovered` always performs discovery again inside its own
  read-only transaction so a caller-supplied map cannot describe a stale
  snapshot. `PlacementKeyColumn`, `SequenceGrant`, `SchemaGrant`, and
  `AllSchemaGrant` preserve AST-only state that current INFORMATION_SCHEMA
  surfaces cannot recover.

- `spannersys/` — Go structs for 51 known `SPANNER_SYS` tables. The tested managed and
  Omni targets currently advertise 50; the superset also includes the documented,
  Enterprise-only `VECTOR_INDEX_STATS`. Tables that exist per interval
  (e.g. `QUERY_STATS_TOP_MINUTE` / `_10MINUTE` / `_HOUR`) share a single struct; pick the
  table via `Interval` constants and `*Query(interval)` helpers in `query.go`; these
  helpers reject any interval outside the documented `MINUTE`, `10MINUTE`, and `HOUR`
  set. Shared
  nested `STRUCT` types (e.g. `LatencyDistribution`, `OperationsByTable`) live in `types.go`.
  `Audit` retains the complete advertised table/column/raw-type/ordinal tuples,
  reports relative-order drift, and distinguishes registry-known columns absent
  from a successful target capture from unknown advertised metadata. Missing or
  failed capture data is not an observed absence.
  The private descriptor extractor derives the complete 51-table / 547-column
  structural type surface from the decoder structs. Its canonical renderer keeps pointer
  nullability as decoder evidence only and compares all advertised raw types
  without parsing nested `SPANNER_TYPE` strings.
  `ExportManifest(sourceCommit)` combines those private descriptors with the
  strict redacted managed and Omni captures embedded from `spannersys/evidence/`.
  It emits the prerelease JSON contract defined under `schemas/`, performs no
  Git lookup or live probe, and fails closed unless every required target has a
  complete agreeing observation. `cmd/spanner-sys-export` is the corresponding
  explicit-commit CLI. The parent analyzer pins its bytes without importing
  producer packages into the root module.

- `astconv/` — bidirectional conversion between `infoschem.Schema` (a bag holding every
  `infoschem` slice) and `[]ast.DDL`. Each DDL family has paired files `to_ast_<family>.go`
  / `from_ast_<family>.go` (table, index, view, change_stream, sequence, model, graph,
  placement, schema, role, database, statistics, function, vector_index, proto_bundle,
  locality_group). `schema.go` declares `Schema` and the entry points
  `ToDDLStatements()` / `FromDDLStatements()`; `from_ast.go` dispatches on AST node type;
  `helpers.go` holds `ident`/`path` and option/options builders.
  `ToDDLStatements` emits in the order Spanner uses (proto bundle → schemata → ALTER
  DATABASE → tables → indexes → vector indexes → views → change streams → sequences →
  models → graphs → placements → functions → locality groups → roles+grants → ALTER
  STATISTICS).

## Architectural notes that aren't obvious from the code

- **Dependency: memefish v0.8.1.** The module pins `v0.8.1`. This version adds
  `ColumnDef.PlacementKey`, `PrivilegeOnSequence`, and
  `PrivilegeOnAllSequencesInSchema`. Several earlier types changed shape from
  v0.6.2/v0.7.0:
  `CreateSearchIndex.Name`/`TableName`, `PrivilegeOnTable.Names`,
  `SelectPrivilegeOnChangeStream.Names` are now `*ast.Path` / `[]*ast.Path` (was
  `*ast.Ident` / `[]*ast.Ident`). Use `astconv/helpers.go` helpers: `ident`/`path`
  when building AST, and `leafName(p *ast.Path)` when extracting the leaf identifier.
  `InterleaveIn.TableName` remains an `*ast.Ident`. A 2026-07-26 raw-DDL probe
  confirmed that emulator v1.5.55 and real Spanner accept a qualified regular-
  index interleave target, while the unqualified form emitted through this AST
  resolves in the emulator's default schema. Named-schema interleaved indexes
  therefore fail closed until memefish can retain the target schema.

- **Placement keys and sequence grants are AST-only metadata.** Real Spanner
  accepts both surfaces, and `GetDatabaseDdl` returns an explicit sequence
  grant, but the current production INFORMATION_SCHEMA has neither a placement-
  key attribute on `COLUMNS` nor a role-sequence-grant table. Preserve them in
  `Schema.PlacementKeyColumns`, `Schema.SequenceGrants`, and
  `Schema.AllSchemaGrants`; do not claim live recovery from `LoadSchema`.
  `GetDatabaseDdl` expands `ON ALL SEQUENCES IN SCHEMA` to exact grants for
  existing sequences and emits nothing for an empty schema, so it cannot
  recover the original wildcard intent either.

- **`token.Pos` zero value is valid.** memefish treats `token.Pos(0)` as a real position;
  the sentinel for "no position" is `token.InvalidPos = -1`. Optional positional fields
  (e.g. `IdentityColumn.Rparen`, `Hidden`, `Stored`) must be set to `token.InvalidPos`
  when absent. Setting them to `token.Pos(1)` is how you opt into emitting the optional
  syntax (e.g. parentheses around an empty `IDENTITY()`).

- **`ast.SchemaType` ≠ `ast.Type`.** `memefish.ParseType` returns `ast.Type`, which is
  not assignable to fields like `ColumnDef.Type`. Always go through
  `spannertype.ParseSchemaType` for column types.

- **GRANT views vs. tables.** `ROLE_TABLE_GRANTS` mixes table and view grants. Round-trip
  must build a view-name set from `Schema.Views` and emit `SelectPrivilegeOnView` for
  view names; otherwise `gcloud spanner databases ddl describe` output won't match.

- **PROPERTY GRAPH metadata is JSON-backed.** `INFORMATION_SCHEMA` stores the graph
  definition as JSON in `PROPERTY_GRAPHS.PROPERTY_GRAPH_METADATA_JSON`, and
  `astconv` now reconstructs default-schema `CREATE PROPERTY GRAPH` statements
  by rebuilding canonical SQL from that JSON and reparsing it with memefish.
  memefish still only accepts unqualified graph / element names here, so named
  schema graphs remain partially blocked upstream. By contrast,
  `SCHEMATA.PROTO_BUNDLE` is deserialized from its `FileDescriptorSet` payload to
  recover `CREATE PROTO BUNDLE` type names automatically. See `UNSUPPORTED_DDL.md`
  for the remaining partial / unsupported features and the upstream memefish issues
  blocking them.

- **Locality groups have two representations.** `LocalityGroupOptions` is the live
  INFORMATION_SCHEMA representation and can identify only groups with options;
  `Schema.LocalityGroups` preserves every `CREATE LOCALITY GROUP` seen in an AST,
  including optionless groups. Option values retain their expression text rather
  than being quoted a second time on regeneration.

- **Unsupported surfaces fail closed.** Model grants lack a memefish AST node.
  Named-schema tables, non-interleaved regular/search indexes, views, sequences,
  scalar functions, and path-backed grant targets are supported; DDL families
  or clauses whose AST still exposes only an unqualified identifier remain
  explicit errors.
  `ToDDLStatements` never silently drops a grant or dequalifies an object.
  Consult `UNSUPPORTED_DDL.md` before adding a new family.

- **No `CREATE PROCEDURE` in Spanner.** Stored procedures don't exist;
  `INFORMATION_SCHEMA.ROUTINES` only stores SQL UDFs and remote UDFs, both modeled as
  `ast.CreateFunction`.

- **Target evidence.** Redacted retained observations live under
  `infoschem/evidence/`: managed captures are timestamp-primary, while Omni and
  Emulator captures are keyed by OCI manifest digest and platform. The root
  `information_schema_projection_source.json` explicitly selects one managed
  capture; no `/tmp` output or modification-time-based `latest` file is an
  authority. Canonical capture commands require `GOWORK=off` and
  `GOFLAGS=-mod=readonly`, so the producer hash closes over the survey module's
  pinned dependency graph. The dated reports in `docs/` remain historical
  summaries.
