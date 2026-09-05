---
type: Research Note
title: "Cloud Spanner Emulator 1.5.57 source audit and unannounced changes"
description: "Complete changed-file audit and version-paired endpoint probes for v1.5.56 to v1.5.57, observed on 2026-09-05."
tags: [spanner, emulator, source-audit, postgresql, compatibility]
status: draft
sources:
  - id: release
    resource: evidence/emulator-1.5.57-release/release.json
    title: "Full official v1.5.57 release response"
  - id: scope
    resource: evidence/emulator-1.5.57-source/source-identity.json
    title: "Exact source pair and 105 changed files"
  - id: observations
    resource: evidence/emulator-1.5.57-source/verification-summary.json
    title: "Version-paired endpoint observations and assertions"
---

# Cloud Spanner Emulator 1.5.57 source audit

The v1.5.56 to v1.5.57 source and runtime comparison on 2026-09-05 found
material changes absent from the release notes: a PostgreSQL NULL ESCAPE
crash fix, `score_version` acceptance with broken DDL serialization, removal
of empty-array hints, clearer user errors, and changed PostgreSQL view DDL.
Several other edits advance unsupported inputs to a later rejection stage.
Cloud Queue DDL remains unreachable in the audited public source and rejected
by the tested release image.

The full [official release response](evidence/emulator-1.5.57-release/release.json)
was read during the investigation. v1.5.57 was published at
2026-09-04 18:57:52 UTC and was the latest release when checked.[^release]
The associated [release verification](cloud-spanner-emulator-1.5.57-verification.md)
records survey compatibility and every announced item.

## Exact scope and evidence

- Official source repository: [GoogleCloudPlatform/cloud-spanner-emulator](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator).
- Baseline: v1.5.56, `2abe04c1ace40b8760133e6ca0800d257d127da4`.
- Candidate: v1.5.57, `fc811a1a93c7e4784db7f48fe15e72fe94f39d38`.
- [Source comparison](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/compare/v1.5.56...v1.5.57): **105 changed files, 2963 insertions, 322 deletions**; one imported change commit and its merge commit. The imported commit is `3cdfbf3d013e470f84519c90fbc726711e6f801d` (GitOrigin-RevId `f7dabe2224da529500730d96bd4476bfc3031744`).
- All 105 paths were assigned exactly once: schema 30, PostgreSQL 45, query/transaction 26, primary build/gateway 4. [coverage-all.json](evidence/emulator-1.5.57-source/coverage-all.json) is the corrected, consolidated per-file ledger.[^scope]
- Runtime comparison used the official **linux/amd64** images pinned by digest. They ran through architecture translation on this arm64 Colima host. This audit does not establish an arm64 release image or reproduce the upstream Bazel build.

| Runtime | Pinned manifest digest |
| --- | --- |
| `gcr.io/cloud-spanner-emulator/emulator:1.5.56` | `sha256:24c921c60e277e1ecfe50188169bf1aa818c8603d96842c827d4d1920206811c` |
| `gcr.io/cloud-spanner-emulator/emulator:1.5.57` | `sha256:4987860c9f8ecf1fffbbcdac115cb88cb9d1a42bd966c235a9ab843aea34fbd1` |

There are **108 selected endpoint observations** and **44 passing evidence assertions**[^observations] in [verification-summary.json](evidence/emulator-1.5.57-source/verification-summary.json). The Go probes collect both success and error outcomes; their successful test exit does not mean every SQL statement is supported. The first baseline run ended with a Go test timeout after its emulator process crashed on NULL ESCAPE. Seven other probe runs exited zero, including a dedicated R4 old/new NULL ESCAPE reproduction with bounded client deadlines. Unsupported input and expected error outcomes are identified below.

## Unannounced changes confirmed through the public emulator

### U1. `score_version` is accepted but cannot round-trip through DDL

GoogleSQL now accepts all probed integer/NULL values: `1`, `-1`, `9223372036854775807`, and `NULL`. The old release rejects the option as unknown. A quoted string `'1'` is rejected in the new release. Source imposes only signed-int64/null typing, with no additional range check and no feature gate. The stored optional has no downstream readers in this source tree; no query/planning meaning is established.

```sql
ALTER DATABASE `emulator-database` SET OPTIONS (score_version = 1);
```

`GetDatabaseDdl` returns this for every accepted value, including NULL:

```sql
ALTER DATABASE emulator-database SET OPTIONS (score_version = '')
```

This loses the typed value. The raw statement also has a separate unquoted hyphenated-database-name problem. Replaying it after correcting only the database-name quoting still fails because `''` is not an integer or NULL. Thus the option's serialization failure is independently demonstrated. After a successful setting, `INFORMATION_SCHEMA.DATABASE_OPTIONS` contains **zero** `score_version` rows.

Source: [parser branch](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/backend/schema/parser/ddl_parser.cc#L2289), [typed storage](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/backend/schema/updater/schema_updater.cc#L1396), [validation](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/backend/schema/updater/schema_updater.cc#L5154), [GoogleSQL DDL printer](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/backend/schema/printer/print_ddl.cc#L963), [schema dump](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/backend/schema/catalog/schema.cc#L547), [metadata population](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/backend/query/information_schema_catalog.cc#L749). Evidence: [observations-r2-1.5.57-TestSourceAuditGoogleSQL.json](evidence/emulator-1.5.57-source/observations-r2-1.5.57-TestSourceAuditGoogleSQL.json), baseline [observations-1.5.56-TestSourceAuditGoogleSQL.json](evidence/emulator-1.5.57-source/observations-1.5.56-TestSourceAuditGoogleSQL.json).

Do not advertise PostgreSQL option support or a score-related execution effect. The public acceptance probe was GoogleSQL only.

### U2. `LIKE ... ESCAPE NULL` no longer terminates the PostgreSQL emulator

```sql
SELECT ('a' LIKE 'a' ESCAPE NULL)::text;
```

- v1.5.56: the dedicated emulator terminates. Its log records `Check failed: !metadata_.is_null() Null value`, followed by the gateway shutting down because its gRPC server terminated. Container exit code is 255 and `OOMKilled` is false. The original client retried until its eight-minute test timeout. A fresh R4 container reproduced the failure with only a successful SELECT 1 control followed by the exact NULL ESCAPE request, logged before sending; both the failing request and subsequent control then reached their 15-second client deadlines.
- v1.5.57: ordinary `InvalidArgument` explaining that NULL is not an accepted escape; a subsequent `SELECT 1::text` succeeds. The ordinary backslash-escape control succeeds in both releases.

The new guard is immediately before reading the nullable value's string payload: [forward_function.cc](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/transformer/forward_function.cc#L455). Evidence: the dedicated `probes-r4-*.jsonl`, `observations-r4-*-TestSourceAuditNullEscape.json`, [r4-baseline-null-escape-container.log](evidence/emulator-1.5.57-source/r4-baseline-null-escape-container.log), and [r4-baseline-null-escape-container-state.json](evidence/emulator-1.5.57-source/r4-baseline-null-escape-container-state.json). In R4 the exact pre-send SQL log is at 07:01:34.366 UTC, and the container fatal null-value check is at 07:01:34.436 UTC. No other SQL was sent between the successful control and that request. This is a confirmed unannounced reliability fix, separate from the release's advertised Change Stream crash fix.

### U3. Empty-array PostgreSQL hints are silently removed

These now execute successfully; v1.5.56 rejects them during syntax/type analysis:

```sql
/*@ scan_method=[] */ SELECT 1::text;
/*@ scan_method=ARRAY[] */ SELECT 1::text;
/*@ unknown_hint=[] */ SELECT 1::text;
SELECT id::text FROM hint_table /*@ force_index=[] */;
```

The implementation returns no hint for an empty array and callers omit it. This even removes an otherwise unknown hint name. Nonempty arrays do not thereby become valid values: `scan_method=['row']` is rejected for its value type, and `unknown_hint=['x']` is rejected as unsupported. A successful empty array does not request a particular plan.

Source: [array hint grammar](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/src/backend/parser/gram.y#L19272), [empty-array removal](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/src/backend/parser/parse_relation.c#L1743), [statement caller](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/src/backend/parser/analyze.c#L257). Evidence: the two `observations-r2-*-TestSourceAuditPostgreSQLHints.json` files.

### U4. Unsupported COMPRESSION and JSONB IN-subqueries get normal user errors

| Input | v1.5.56 | v1.5.57 |
| --- | --- | --- |
| `CREATE TABLE ... (v text COMPRESSION pglz)` | `Internal`, parse-tree RET_CHECK | `InvalidArgument`, column compression unsupported |
| JSONB left operand of `IN (SELECT ...)` | `Internal`, resolved AST equality validation failure | `InvalidArgument`, non-equality type `PG.JSONB` unsupported |

These are error-handling changes; neither feature becomes supported. The JSONB example was `SELECT ('{"a":1}'::jsonb IN (SELECT '{"a":1}'::jsonb))::text`.

Source: [COMPRESSION validation](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/ddl/pg_parse_tree_validator.cc#L216), [IN-subquery guard](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/transformer/forward_expr.cc#L829). Evidence: PostgreSQL general observations and both hint-observation files.

### U5. Stored PostgreSQL view DDL reflects the rewritten query

For this accepted view, both versions return rows `1, 2`:

```sql
CREATE VIEW unnest_view SQL SECURITY INVOKER AS
SELECT unnest(ARRAY[1,2]) AS value;
```

The relevant `GetDatabaseDdl` body changes from:

```sql
SELECT unnest(ARRAY['1'::bigint, '2'::bigint]) AS value
```

to:

```sql
SELECT value FROM unnest(ARRAY['1'::bigint, '2'::bigint]) "$array"(value)
```

The deparser callback moved after forward transformation, including the existing simple set-returning-function rewrite. This is a concrete serialization change for schema-diff/reconstruction consumers, not new UNNEST query support. The same callback also serves UDF translation, but no UDF output comparison was established here.

Source: [deparser ordering](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/interface/spangres_translator.cc#L523). Evidence: both `observations-r3-*-TestSourceAuditPostgreSQLViewDDL.json` files.

## Unannounced changes that do not establish new usable SQL features

### U6. JSONB numeric brackets advance to a later unsupported layer

`jsonbsubs.c` changes INT4 handling to INT8. For `(jsonb_value)[1]` and larger/negative indexes, baseline returns `InvalidArgument` during PostgreSQL analysis; candidate reaches the forward transformer and returns `Unimplemented` for `(jsonb, bigint)` SubscriptingRef. Its lookup still maps bracket syntax to the generic array function, which has no JSONB signature. **JSONB bracket support is not delivered by this patch.**

The separate `->>` evaluator narrows INT64 to int32 in both releases. Both return `'a'` for `'["a","b"]'::jsonb ->> 4294967296` and `'b'` for index 4294967297. This wraparound is a pre-existing issue, not a v1.5.57 regression. The unchanged bootstrap procedure overrides explain why the large inputs were already accepted.

Source: [INT8 parser change](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/src/backend/utils/adt/jsonbsubs.c#L83), [generic expression mapping](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/catalog/builtin_expression_functions.cc#L52), [unchanged text evaluator](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/catalog/emulator_functions.cc#L2104). The per-file ledger retains the corrected source classification.

### U7. Remote-UDF JSON arrays only change rejection layers

All-string arrays in the PostgreSQL REMOTE function AS JSON object pass the new translator branch. The public schema validator still only accepts scalar `endpoint` and `max_batching_rows` options. A tested `endpoints` array moves from “unsupported array type” to “invalid option endpoints”; a mixed string/number array receives a new element-type error. No usable remote-UDF array option was found. The scalar endpoint control creates successfully in both versions; its later DDL dump failure is also pre-existing.

Source: [new JSON array handling](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/ddl/pg_to_spanner_ddl_translator.cc#L3091), [public option validation](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/backend/schema/updater/sql_expression_validators.cc#L476). Evidence: both R2 PostgreSQL hint-observation files. No remote function was executed or called over the network.

### U8. PG catalog full names improve serialization but still cannot support a persisted catalog view

Direct `pg_catalog.pg_type` and `pg_catalog.pg_tables` queries work in both versions. New full-name qualification fixes an intermediate SQLBuilder name: baseline catalog-view DDL fails with unqualified `pg_tables` not found, while candidate reaches the dependency collector and fails with `Dependency not found: pg_tables`. The view remains unsupported. Unqualified `pg_type` remains unresolved in both tested environments.

Source: [catalog names](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/backend/query/pg_catalog.cc#L386), [dependency validator](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/backend/schema/updater/sql_expression_validators.cc#L205). Evidence: general PostgreSQL observations.

### U9. Library, dormant, build, and mechanical changes

- A new view-statement-hint rewriter exists, but public PostgreSQL view grammar admits no statement-hint prefix inside its `SelectStmt`. Parentheses do not change that grammar; an outer DDL hint is separately rejected. The tested spelling fails in both releases. Do not advertise hinted-view support. See the [rewriter](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/interface/spangres_translator.cc#L190).
- A named-catalog expression-wrapping parameter defaults to false; no production emulator caller enables it. This is opt-in library scaffolding, not confirmed named-schema expression support. [Wrapper](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/interface/spangres_translator.cc#L764).
- The direct PostgreSQL DROP SEQUENCE printer preserves `IF EXISTS`; public `GetDatabaseDdl` emits current sequence creations, not historical DROP statements. Ordinary DROP execution already worked. This is printer completeness, without a demonstrated new public endpoint result. [Printer](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/ddl/spangres_direct_schema_printer_impl.cc#L1469).
- PostgreSQL timezone code adds pthread cleanup of per-thread GMT state. This is source-confirmed resource cleanup; a memory/concurrency benchmark was not run and SQL semantics were not shown to change. [Cleanup](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/third_party/spanner_pg/src/timezone/localtime.c#L102).
- Go build SDK pin changes **1.25.11 to 1.25.13**. The development container is built from the repository Dockerfile instead of using the published devcontainer image. No particular Go security effect is asserted. [SDK](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/MODULE.bazel#L106), [devcontainer](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/fc811a1a93c7e4784db7f48fe15e72fe94f39d38/.devcontainer/devcontainer.json#L9).
- Both gateway Go files change only blank lines; ignoring whitespace/blank lines leaves no gateway diff. Removing one duplicate `REMOTE` keyword does not change membership. Queue-related ownership tolerance, optional-value substitutions, build wiring, fixtures, and test refactors are individually covered in the ledger.

## Reconciliation with all ten release-note bullets

| Announced item | Source/runtime assessment |
| --- | --- |
| PostgreSQL parameterized partitioned DML | Real dialect-dispatch repair in the validation route; prior runtime probe passed. Ordinary parameter support was pre-existing. |
| PostgreSQL SET/DROP NOT NULL | Generic updater now executes the translated operations; prior runtime probe passed. |
| Indexes storing pending commit timestamps | Internal index maintenance bypasses a client read guard. Index-key PCT update/delete restrictions remain. Prior stored-value update probe passed. |
| Feature-flag reference lifetime | Returns an owning snapshot under the lock; source-confirmed concurrency repair. |
| Change Stream stale-object nullptr | Skips missing tracked tables/columns while reconfiguring a stream after DROP. Prior exact ALTER-after-DROP probe passed. |
| Cloud Queue DDL | New Queue files are unbuilt and unreachable: no grammar, DDL proto, updater/schema ownership, printer path, or build registration. Queue-specific error helpers are missing. Prior image rejected CREATE QUEUE and CREATE CLOUD QUEUE at the keyword. |
| Tables without explicit primary keys | Default-enabled hidden identity rowid; both dialects exercised previously. PostgreSQL schema dump now preserves HIDDEN. |
| PostgreSQL TABLESAMPLE | BERNOULLI and SPANNER.RESERVOIR wired; SYSTEM and REPEATABLE remain unsupported. Prior positive/negative probes passed. |
| 5000 tables / 10000 indexes | Configured enforcement constants confirmed; 5000/10000-object stress test not executed. |
| IS [NOT] DISTINCT FROM | Both dialects enabled; previous NULL/NaN/scalar probes passed. ARRAY/STRUCT limits remain. |

The newly added Change Stream historical record-shape test does **not** call the fixed unregister path. It is added coverage, not evidence of a newly implemented schema-versioning feature. ANN changes are test-fixture refactoring; no new ANN distance validator behavior was found. The per-file ledger records their test-only scope.

## Implications for Spanalyzer

The release verification established the existing survey suite against v1.5.57
and the unchanged 198-column INFORMATION_SCHEMA baseline. This audit identifies
additional compatibility traps: metadata reconstruction cannot discover
`score_version`; DDL dump/replay cannot retain its typed value; and PostgreSQL
view DDL text can change after the UNNEST rewrite. Queue, JSONB brackets and
remote-UDF array options must not be promoted based on newly added source alone.

## Evidence and reproduction

The [scope ledger](evidence/emulator-1.5.57-source/coverage-all.json),
[source identity](evidence/emulator-1.5.57-source/source-identity.json) and
[changed-source hashes](evidence/emulator-1.5.57-source/changed-source-hashes.json)
bind the audit to the exact source pair. The [verification summary](evidence/emulator-1.5.57-source/verification-summary.json)
selects ten observation files and 44 assertions. SQL inputs, return values and
error classes are retained in those files. Their filenames preserve probe
revisions because the first PostgreSQL baseline process crashed.

The dedicated R4 crash reproduction includes the exact pre-send statement,
pre/post controls, bounded client deadlines, container fatal log and non-OOM
exit state. It resolves the initial uncertainty about the request causing
termination. Earlier provisional peer reports are not canonical evidence.

All four probe revisions are retained in
[source evidence](evidence/emulator-1.5.57-source/), together with the release
probe helper. The `.go.txt` files are historical producer source, including
their original temporary-file comments; they are not automatically discovered
Go tests. From the repository root:

```sh
runtime_audit_dir=$(mktemp -d)
cp knowledge/research/runtime-verification/evidence/emulator-1.5.57-release/release_probe_test.go.txt \
  "$runtime_audit_dir/release_probe_test.go"
cp knowledge/research/runtime-verification/evidence/emulator-1.5.57-source/source_audit_probe_test.go.txt \
  "$runtime_audit_dir/source_audit_probe_test.go"
cd survey
GOWORK=off GOFLAGS=-mod=readonly GOARCH=amd64 \
  AUDIT_OBSERVATION_PREFIX="$runtime_audit_dir/observations-1.5.57" \
  go test -tags=integration -count=1 -json -timeout=10m \
  -run '^TestSourceAudit' "$runtime_audit_dir/release_probe_test.go" \
  "$runtime_audit_dir/source_audit_probe_test.go"
```

Use Go 1.26.6 and an accessible Docker daemon, with amd64 execution support on
an arm64 host. Set `RELEASE_PROBE_IMAGE` to the baseline's fully pinned image
for the old-version comparison. Tests create isolated emulator containers and
close them on exit. The NULL ESCAPE probe intentionally terminates its baseline
container; its observation collector can still pass after recording the
expected deadlines. Passing collectors do not imply supported SQL features.

This covers every changed file for the specified pair and selected runtime
behaviors. It is not a full upstream Bazel build/test run or exhaustive SQL
verification. Managed Spanner, Omni, an arm64 v1.5.57 image, quota stress and
concurrency stress were not tested in this source-audit phase. Source and
released images were independently pinned; binary reproducibility was not
established. Migration into OKF did not rerun the runtime.

[^release]: [Full official release response](evidence/emulator-1.5.57-release/release.json),
    retained from the 2026-09-05 investigation.
[^scope]: [Source identity](evidence/emulator-1.5.57-source/source-identity.json),
    [105-file ledger](evidence/emulator-1.5.57-source/coverage-all.json) and
    [source hashes](evidence/emulator-1.5.57-source/changed-source-hashes.json).
[^observations]: [Verification summary](evidence/emulator-1.5.57-source/verification-summary.json)
    names all ten retained observation files. [Migration provenance](evidence/migration-provenance.json)
    records original and retained artifact hashes, including portable path
    replacements and removal of references to provisional peer reports.
