# spanner-query-gen

`spanner-query-gen` is an experimental query code generation command. It is
intended to grow in the direction of
[`sqlc`](https://github.com/sqlc-dev/sqlc)-style query-driven code generation
and [`yo`](https://github.com/cloudspannerecosystem/yo)-style Cloud Spanner Go
models.

The v1alpha configuration shape is intentionally small: config declares user
intent, `explain-plan` shows normalized semantics, and generated Go is only the
rendered result. v1alpha is pre-release and may make breaking changes before
the first stable v1 configuration.

## Config Versioning

Current YAML input must use `version: v1alpha`.

`v1alpha` is a mutable preview channel until the first stable `v1` config is
defined. Once `v1` is fixed, `v1alpha` should become a deprecated alias for
that same normalized schema for a transition period. Future breaking previews
should use a new version such as `v2alpha` rather than changing the meaning of
the post-v1 `v1alpha` alias.

The command rejects unsupported or unknown v1alpha YAML fields instead of
silently ignoring them. Use `config-schema` as the reviewable contract for
editor integration and CI validation.

## Documentation Map

- This README describes the current command UX and examples.
- [`DESIGN.md`](DESIGN.md) records the intended architecture and future design
  direction.
- [`IMPLEMENTATION_STATUS.md`](IMPLEMENTATION_STATUS.md) summarizes where the
  current implementation intentionally differs from or lags behind the design.
- [`PLAN_CONTRACTS.md`](PLAN_CONTRACTS.md) documents the current experimental
  `plan-report --contracts` surface.
- [`../../research/spanner-query-gen/PLAN_CONTRACT_CANDIDATES.md`](../../research/spanner-query-gen/PLAN_CONTRACT_CANDIDATES.md)
  collects possible future plan contracts from query-optimization practices.
- [`../../tools/spanner-query-plan-shape`](../../tools/spanner-query-plan-shape)
  is a developer-only Spanner Omni probe for inspecting raw query plan shapes
  before changing operator normalization or plan contracts.

## Usage

```sh
go run ./cmd/spanner-query-gen generate \
  --config testdata/querygen.yaml \
  --out /tmp/querydemo.sql.go

go run ./cmd/spanner-query-gen generate \
  --config testdata/querygen.yaml \
  --check

go run ./cmd/spanner-query-gen check \
  --config testdata/querygen.yaml

go run ./cmd/spanner-query-gen explain-plan \
  --config testdata/querygen.yaml \
  --output yaml

go run ./cmd/spanner-query-gen plan-report \
  --config testdata/querygen.yaml \
  --output markdown

go run ./cmd/spanner-query-gen vet \
  --config testdata/querygen.yaml \
  --output yaml

go run ./cmd/spanner-query-gen config-schema \
  --output yaml

go run ./cmd/spanner-query-gen config-schema \
  --out schemas/spanner-query-gen.v1alpha.schema.json

go run ./cmd/spanner-query-gen plan-report-schema \
  --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json

go run ./cmd/spanner-query-gen plan-contract-schema \
  --out schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json
```

If neither config `go.out` nor `--out` is set, generated Go code is written to
stdout. Use `--out -` to force stdout. `check` is equivalent to
`generate --check`.

Query DTOs, SQL constants, and basic runtime query free functions are the
baseline generated output for declared queries. Optional Spanner queries emit a
typed `<Name>Params` struct plus `Build<Name>SQL` helper instead of a single SQL
constant. `emit` controls additional runtime helper surfaces such as Cloud
Spanner mutations and DML statements. The explicit
`emit.*.query_methods` config flags are reserved for a future richer method
surface and are rejected in v1alpha.

`explain-plan` prints the resolved generation plan as YAML, JSON, or a human
summary. The default machine-readable plan omits heavy audit-only projection
matrix rows. Use `--audit` to include those rows, and `--stable` to omit
volatile audit fields such as verification timestamps from snapshot-oriented
output. In v1alpha, `--stable` keeps semantic fields and removes volatile audit
fields from both default and audit output; it does not freeze every plan field
across v1alpha releases.
Machine-readable YAML output is generated from the same JSON shape with
[`goccy/go-yaml`](https://github.com/goccy/go-yaml)'s `JSONToYAML`, so YAML and
JSON use the same field names, custom JSON marshalers, and `omitempty` behavior.

`plan-report` starts Spanner Omni, applies each referenced Spanner catalog DDL
to a separate database in the same Omni instance, analyzes configured Spanner
queries with `AnalyzeQuery`, and renders human-readable plan trees with
[`spannerplan`](https://github.com/apstndb/spannerplan). It writes Markdown by
default and also supports YAML/JSON for review artifacts. `plan-report` is an
optional Omni-backed workflow, not the primary DDL-first generation path.
It depends on Spanner Omni execution-plan support and is intended for review,
testing, and prototyping. Do not treat plan contracts as production performance
guarantees; they are review contracts for a described plan environment.
Use `--stable` for snapshot-oriented output. The current report intentionally
omits host paths, temporary database IDs, and acquisition timestamps; `--stable`
records that stable-output mode was requested while keeping semantic fields such
as backend, config digest, SQL digest, DDL digest, contract file digest,
operator tree digest, and optimizer pinning status.
In v1alpha, `--require-optimizer-pinning` checks optimizer options requested by
`plan-report` flags. It does not infer pinning from SQL statement hints, and
`optimizer.effective` is reported as `not_recorded`.
`--backend-version` and `--backend-image-digest` can record manually pinned
backend identity evidence when the runtime cannot expose it directly; when they
are omitted, version and image digest remain `not_recorded`.
Top-level `plan-report` status describes artifact production. It does not mean
every query target was successfully planned. Use `target_summary.planned`,
`target_summary.errors`, `target_summary.skipped`, and `queries[].status` for
target-level success. When a query is `error` or `skipped`, only input/error
fields are reliable when present; plan normalization fields are absent and are
not successful plan evidence.

`vet` validates the same plan without rendering code. Text output writes
warning and suppression summaries to stderr and keeps stdout empty.
Machine-readable JSON/YAML output writes reports to stdout and keeps stderr
empty unless there is a fatal CLI error.

`config-schema` prints a JSON Schema for the public v1alpha YAML config. It is
intended for review, editor integration, and CI checks outside this command.
The schema is canonical JSON Schema; `--output yaml` renders that same schema
document as YAML for readability through the same JSON-to-YAML conversion path.
The schema validates the public field set plus kind/operation syntax.
DDL-dependent semantics such as column existence, key inference, unsupported
type projection, and upsert column-set invariants are validated by `generate`,
`check`, `vet`, and `explain-plan` planning.
JSON Schema `uniqueItems` only rejects duplicate whole objects, so semantic
name uniqueness such as `catalogs[].name`, `queries[].name`, `writes[].name`,
binding names, and duplicate scoped parameters is diagnosed by planning.
The latest JSON schema is checked in at
[`schemas/spanner-query-gen.v1alpha.schema.json`](../../schemas/spanner-query-gen.v1alpha.schema.json);
regenerate it with `config-schema --out` when the config shape changes.
The latest `plan-report` output schema is checked in at
[`schemas/spanner-query-gen.plan-report.v1alpha.schema.json`](../../schemas/spanner-query-gen.plan-report.v1alpha.schema.json);
regenerate it with `plan-report-schema --out` when the report YAML/JSON shape
changes.
The latest plan contract file schema is checked in at
[`schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json`](../../schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json);
regenerate it with `plan-contract-schema --out` when the contract file shape
changes.

Current root help:

```text
Usage:
  spanner-query-gen <subcommand> [flags]

Subcommands:
  generate       generate Go code from a v1alpha config
  check          verify the configured generated file is up to date
  explain-plan   print the resolved generation plan
  plan-report    analyze configured Spanner queries on Omni and print query plans
  vet            validate the resolved generation plan
  config-schema  print the v1alpha config JSON Schema
  plan-report-schema
                 print the plan-report output JSON Schema
  plan-contract-schema
                 print the plan contracts JSON Schema

Use "spanner-query-gen <subcommand> --help" for subcommand flags.
```

Current `config-schema --help` output:

```text
Usage of spanner-query-gen config-schema:
  -out string
        write schema to file instead of stdout; use - for stdout
  -output string
        schema output format: json or yaml (default "json")
```

Current `plan-report-schema --help` output:

```text
Usage of spanner-query-gen plan-report-schema:
  -out string
        write schema to file instead of stdout; use - for stdout
  -output string
        schema output format: json or yaml (default "json")
```

Current `plan-contract-schema --help` output:

```text
Usage of spanner-query-gen plan-contract-schema:
  -out string
        write schema to file instead of stdout; use - for stdout
  -output string
        schema output format: json or yaml (default "json")
```

Current `plan-report --help` output:

```text
Usage of spanner-query-gen plan-report:
  -backend string
        runtime backend: omni (default "omni")
  -backend-image-digest string
        backend container image digest to record in backend_identity, for example sha256:<64 hex chars>
  -backend-version string
        backend version to record in backend_identity
  -check
        evaluate --contracts and fail when a contract is violated
  -config string
        query code generation config file (default "spanner-query-gen.yaml")
  -contracts string
        experimental plan contracts YAML file
  -format string
        spannerplan tree format: current, traditional, or compact (default "current")
  -optimizer-statistics-package string
        Spanner optimizer statistics package to pass to AnalyzeQuery
  -optimizer-version string
        Spanner optimizer version to pass to AnalyzeQuery
  -output string
        report output format: markdown, yaml, or json (default "markdown")
  -require-optimizer-pinning
        fail when optimizer version or statistics package is not pinned
  -require-targets
        fail when no Spanner query targets are available
  -stable
        omit volatile metadata from report output
  -wrap-width int
        maximum rendered plan width; 0 disables wrapping
```

## Minimal Config

The first example is Spanner-only on purpose. BigQuery, `EXTERNAL_QUERY`,
external datasets, DTO sharing, and suppressions are separate features shown
later.

```yaml
version: v1alpha

go:
  package: querydemo
  out: querydemo.sql.go

emit:
  spanner:
    mutations: true
    dml: true

catalogs:
- name: app
  kind: spanner
  ddl: schema/spanner.sql

queries:
- name: ListSingers
  catalog: app
  kind: table
  table: Singers
  result:
    struct: SingerRow

writes:
- name: UpdateSingerName
  catalog: app
  table: Singers
  operation: update
  input: SingerWrite
  key:
  - SingerId
  update:
    columns:
    - FirstName
    - LastName
```

`catalogs` are analyzer inputs. `kind: spanner` and `kind: bigquery` default to
GoogleSQL; only non-default dialects need `dialect`.

`emit.spanner.query_methods` and `emit.bigquery.query_methods` are reserved for
future generated client-wrapper method surfaces and are rejected in v1alpha.
Basic query free functions are generated from `queries[]` without those flags.

Query entries use `kind` as a discriminated union. Supported query kinds are
`sql`, `table`, `index`, and `external_query`. Raw BigQuery SQL containing
`EXTERNAL_QUERY(...)` is rejected in v1alpha config; use
`kind: external_query` so the inner SQL and connection mapping remain
reviewable. `result.cardinality` defaults to `many` and drives generated query
free functions. v1alpha query cardinality supports row-returning `one`,
`maybe_one`, and `many`, plus DML-oriented row count and row-set modes used by
the current generated runtime surface. A future `commands` surface may replace
or refine the DML shape, but it is not required for the current v1alpha
implementation.

Use `queries[].params` when the analyzer needs explicit parameter types, such
as ambiguous parameters, ARRAY/STRUCT parameters, or separated BigQuery outer
and Spanner inner SQL scopes. `params` is analyzer type input, not generated
method signature customization. Regular `sql`, `table`, and `index` queries
omit `scope`; `external_query` params may use `scope: inner` or `scope: outer`
when the same parameter name could appear in both SQL scopes.
`params[].type` is parsed as a GoogleSQL type expression such as `INT64`,
`ARRAY<INT64>`, or `STRUCT<...>`. Invalid or analyzer-unsupported type
expressions are planning errors.

Generated Spanner `table` and `index` shorthand queries include deterministic
`ORDER BY` clauses by default because Spanner does not guarantee result order
without one. Omitted `order_by` means `key`: `table` queries order by the
table primary key, and `index` queries order by the index key columns plus the
base table primary key columns with duplicates removed. Use `order_by: none`
on a generated query when that ordering would block a desired query plan
optimization. Explicit `sql` and `external_query` entries own their SQL text
and cannot use `order_by`.

For Spanner `index` shorthand over a `NULL_FILTERED` index, generated SQL adds
`IS NOT NULL` predicates for every nullable index key column, including columns
also constrained by generated key-prefix equality predicates. This matches
Spanner's `FORCE_INDEX` requirement that the query exclude rows omitted from
the null-filtered index.

Generated Spanner queries are covered by a command-package emulator integration
test using [`spanemuboost`](https://github.com/apstndb/spanemuboost). Keeping
this test under `cmd/spanner-query-gen` keeps generator-specific runtime
dependencies out of the root analyzer package test graph. Cloud Spanner
Emulator rejects forced indexes created with the `CREATE NULL_FILTERED INDEX`
clause when source key columns are nullable, without analyzing query predicates.
This is true even when the forced-index query itself has `WHERE key IS NOT NULL`
predicates. The test starts the emulator gateway with
`--disable_query_null_filtered_index_check`, which has been available since
Cloud Spanner Emulator v1.5.7, so generated production SQL can run without
emulator-specific query hints.

An optional Spanner Omni check uses the same generated queries and original
`CREATE NULL_FILTERED INDEX` DDL without the emulator-specific startup flag.
It also uses Omni query plans to verify that the generated index query avoids
a Sort node, while an intentionally different ordering does require Sort.
This check is split behind an extra build tag and environment variable because
Omni is experimental and has heavier host/container requirements.

Run it locally with:

```sh
go test -tags=integration -run '^TestIntegration' ./cmd/spanner-query-gen
```

Run the optional Omni check with:

```sh
SPANEMUBOOST_ENABLE_OMNI_TESTS=1 go test -tags='integration omni' -run '^TestIntegration.*Omni' ./cmd/spanner-query-gen
```

`plan-report` uses the same Omni plan source for review artifacts:

```sh
go run ./cmd/spanner-query-gen plan-report \
  --config testdata/querygen.yaml \
  --output markdown
```

````markdown
# Spanner Query Plan Report

- Status: `ok`
- Backend: `omni`
- Format: `current`
- Render mode: `plan`

## GetLiteral

- Source: `app`
- Scope: `query`
- Kind: `sql`
- Status: `ok`
- SQL SHA-256: `b8ccac726e885e4bae2a0596e87cc88c5113c16e87c3bdec2ffb0e09e49cb29e`
- DDL SHA-256: `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
- Operator tree SHA-256: `...`
- Operator families: `serialize_result`, `unit_relation`

```sql
SELECT 1 AS value
```

```text
+----+------------------------+
| ID | Operator               |
+----+------------------------+
|  0 | Serialize Result <Row> |
|  1 | +- Unit Relation <Row> |
+----+------------------------+
```
````

`plan-report` also has an experimental contract check mode. Plan contracts live
in a separate `v1alpha-plan-contracts` YAML file so the main
`spanner-query-gen.yaml` stays focused on DDL, SQL, DTOs, and write helpers.

```yaml
version: v1alpha-plan-contracts

contracts:
- name: SingerIndexLookupPlan
  target: query/ScanSingerIDsFast
  use:
  - no_explicit_sort
```

Run it with:

```sh
go run ./cmd/spanner-query-gen plan-report \
  --config spanner-query-gen.yaml \
  --contracts plan-contracts.yaml \
  --check \
  --output yaml
```

Plan contracts target structural PLAN output only. Spanner Omni execution-plan
support is documented as a Preview / Pre-GA feature, so `plan-report` is
intended for development, testing, prototyping, and review workflows. The
operator work in this repository has not found an observable query optimizer
plan/operator vocabulary difference between Spanner Omni and the Cloud Spanner
DBaaS service, but DBaaS Spanner can receive optimizer/operator changes before a
matching Omni release. Do not treat plan contracts as production performance
guarantees.

See [`PLAN_CONTRACTS.md`](PLAN_CONTRACTS.md) for predefined contract names,
direct `forbid.operator_family` rules, CEL inputs, normalized topology fields,
`not_evaluated` target handling, and optimizer request/effective fields.

`plan-report` targets Spanner SQL only:

- included: configured queries with a Spanner catalog and generated `kind: sql`,
  `kind: table`, or `kind: index` SQL
- included: `kind: external_query` `inner_sql`, because the inner query is
  Spanner SQL
- excluded: BigQuery outer SQL, BigQuery external dataset bindings, and writes
  until future DML/mutation plan support exists

Machine-readable reports include `plan_source.api: analyze_query`,
`plan_source.render_tool: spannerplan`, backend identity fields with
`source: spanemuboost` when collected from the runtime defaults or
`source: manual` when `--backend-version` / `--backend-image-digest` supplied
caller assertions, canonical target IDs such as `query/ScanSingerIDsFast` and
`query/ExternalQuerySingerIDs#inner`, target inclusion/exclusion details, and
normalization versions for the operator tree digest and operator-family mapping.
`source: manual` is recorded as self-reported provenance, not as observed
backend evidence; `source: not_recorded` means both backend identity fields are
`not_recorded`.
They also include `normalization.cel_input_defaults`, which records the replay
defaults used by CEL evaluation for absent optional fields in `operators[]` and
`operator_edges[]` in canonical order. CEL expressions that use raw
`raw_plan`/`raw_nodes` evaluator inputs are marked
`stability.replayable_from_report: false` because those raw protobuf inputs are
not serialized into the report.
The digest inputs are semantic bytes: rendered/generated SQL, resolved catalog
DDL, and a normalized operator tree that excludes temporary database IDs,
timestamps, runtime stats, row counts, and rendering width.
`target_summary.included_count` is the sum of `planned`, `errors`, and
`skipped`; `target_summary.excluded` is always present and is `[]` when there
are no targets excluded from the report target set. Skipped targets appear in
`queries[]`, are counted by `target_summary.skipped`, and are not duplicated in
`target_summary.excluded`. For `queries[].status: error | skipped`, only
target/input identity fields and the error message are reliable; SQL and DDL
digests are reliable only when present, and plan normalization fields are
absent.

```yaml
queries:
- name: ScanSingerIDsFast
  catalog: app
  kind: table
  table: Singers
  order_by: none
  result:
    struct: SingerIDRow
```

## EXTERNAL_QUERY

`EXTERNAL_QUERY` connections are BigQuery catalog bindings. Each binding points
to a Spanner catalog.

```yaml
catalogs:
- name: app
  kind: spanner
  ddl: schema/spanner.sql

- name: analytics
  kind: bigquery
  project: example-project
  ddl: schema/bigquery.sql
  bindings:
    external_query_connections:
    - name: app_conn
      id: example-project.us.example-connection
      spanner_catalog: app

queries:
- name: ExternalQuerySingerIDs
  catalog: analytics
  kind: external_query
  binding: app_conn
  inner_sql: |
    SELECT SingerId
    FROM Singers
  outer_sql: |
    SELECT *
    FROM __external__
  result:
    cardinality: many
    struct: ExternalQuerySingerIDRow
```

If `outer_sql` is omitted, the generated BigQuery SQL is
`SELECT * FROM EXTERNAL_QUERY(...)`. If `outer_sql` is supplied, it must contain
exactly one `__external__` token where a table expression is valid, such as
`FROM __external__` or `JOIN __external__`.
The generator never adds `ORDER BY` to `inner_sql`; adding it manually can
prevent root-partitionable Spanner execution and still does not guarantee
BigQuery output order. Put ordering in `outer_sql` when the final BigQuery
result order matters.

## Spanner External Datasets

BigQuery Spanner external datasets are modeled as BigQuery catalog bindings,
not as `EXTERNAL_QUERY` shorthand. The generator models static catalog shape
only; it does not create BigQuery datasets, connections, IAM bindings, or
Terraform resources.

```yaml
catalogs:
- name: app
  kind: spanner
  ddl: schema/spanner.sql

- name: analytics
  kind: bigquery
  project: example-project
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app
      spanner_database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
      location: US
      access:
        cloud_resource_connection_id: example-project.us.example-connection

queries:
- name: ExternalDatasetSingerNames
  catalog: analytics
  kind: sql
  sql: |
    SELECT SingerId, FirstName
    FROM `example-project.analytics_spanner.Singers`
  result:
    cardinality: many
    struct: ExternalSingerRow
```

The default external dataset projection policy is conservative:

- unsupported Spanner columns are rejected,
- named-schema Spanner tables are rejected,
- external dataset tables are read-only targets,
- DML or metadata mutation against an external dataset table is a planning
  error.

To match BigQuery's lossy external dataset projection behavior explicitly:

```yaml
projection:
  unsupported_columns: omit
  named_schema_tables: warn_and_omit
```

The canonical strict spelling is `reject`, which is also the default when a
projection policy is omitted. `error` and silent `named_schema_tables: omit`
are not v1alpha config values.

`spanner_database_uri` may include a database role, as in
`google-cloudspanner://reader@/projects/...`. The role is inferred into the
plan. If access verification evidence exists outside the generator, attach it
only when it is real evidence:

```yaml
access:
  cloud_resource_connection_id: example-project.us.example-connection
  verification_evidence:
    status: verified
    verifier: terraform-plan
    checked_at: "2026-05-04T10:30:00Z"
```

Config-supplied verification evidence is normalized in the plan as
`source: external_evidence`. `source: live_probe` is reserved for future
generator-owned live verification and is not a v1alpha config field.
When no evidence is supplied, the plan records `status: not_checked` without a
verification source.

## Writes

Writes declare branch semantics directly.

- `insert` uses `insert.columns`.
- `update` uses `update.columns` as the update mask.
- `upsert` maps to Spanner `INSERT OR UPDATE` for now; `insert.columns` must be
  the same column set as key plus `update.columns`.
- `replace` is mutation-only.
- `delete` uses the table primary key, inferred when `key` is omitted.

For v1alpha writes, `key` is the table primary key set. If omitted, it is
inferred from the table DDL. Partial primary keys, unique-index keys, and future
conflict targets are not modeled by `key`.

Global `emit.spanner` gates generated write helpers. Operation-specific
incompatibility is applied after the global setting; for example, `replace`
never emits a DML helper.

Example:

```yaml
writes:
- name: UpsertSingerName
  catalog: app
  table: Singers
  operation: upsert
  input: SingerWrite
  key:
  - SingerId
  insert:
    columns:
    - SingerId
    - FirstName
    - LastName
  update:
    columns:
    - FirstName
    - LastName
```

For `operation: upsert`, v1alpha emits Spanner `INSERT OR UPDATE` semantics.
Spanner updates every non-key column listed in the statement when the row
already exists. v1alpha therefore rejects insert-only non-key columns. Use
separate writes, or a future `ON CONFLICT` strategy, when insert and update
branches need different value columns. The resolved plan records
`conflict_strategy: insert_or_update` as normalized semantics; public v1alpha
YAML does not expose a `conflict` block.
The `insert.columns == key + update.columns` check is set equality and rejects
duplicates; generated statements render columns in deterministic key/update
order rather than treating YAML order as branch semantics.
When mutation helpers are enabled, `operation: upsert` emits a Spanner
`InsertOrUpdate` mutation helper. This does not allow branch-different column
sets: `InsertOrUpdate` uses one written column set, preserves unspecified
columns on existing rows, and still requires all insert-time `NOT NULL` values
to be supplied.

`input` names a write input struct. It may reuse a query result struct only
when that struct already contains every key and value field required by the
write and each field is valid for the required Spanner encode role. The
generator does not silently add write-only fields to a query DTO.

Spanner updates should normally declare touched columns because lock scope and
mutation count are cell-oriented. To intentionally update every non-key column:

```yaml
update:
  all_non_key_columns: true
```

## Rules

Suppressions are centralized under `rules.suppressions`. A suppression changes
diagnostic presentation only; it never removes plan facts such as omitted
columns, relation provenance, or catalog binding metadata. Severity overrides
are intentionally not part of v1alpha; they belong with a future stricter vet
policy layer.

```yaml
rules:
  suppressions:
  - scope: catalog-binding/analytics.app_dataset
    rule: external-dataset-access-unverified
    reason: access is validated by Terraform plan in CI
    owner: platform
    expires: "2026-12-31"
```

Supported scope prefixes are `query/`, `write/`, and `catalog-binding/`.
`expires`, when present, must use `YYYY-MM-DD` format.

## Design Direction

See [DESIGN.md](DESIGN.md) for the broader design, sqlc/yo comparison,
architecture direction, and roadmap.
