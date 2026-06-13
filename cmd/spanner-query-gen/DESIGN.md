# spanner-query-gen Design

`spanner-query-gen` is an experimental code generator for Go applications that
need type-safe access to Cloud Spanner, BigQuery, or BigQuery `EXTERNAL_QUERY`
queries against Spanner. It is intentionally narrower than a general ORM: SQL
and schema remain the source of truth, and generated code should be small,
predictable, and easy to review.

The product boundary should stay narrow:

`spanner-query-gen` converts statically declared GoogleSQL queries and Spanner
writes into Go types and helpers without runtime magic. It is not an ORM, a
query builder, or a broad CRUD generator.

This document describes the intended shape, including future work. The current
implementation is only an early slice of this design.

## Goals

- Generate Go DTOs from analyzed Spanner GoogleSQL and BigQuery query results.
- Support one DTO shared by compatible Spanner and BigQuery result shapes.
- Support Spanner writes through both mutations and DML.
- Preserve Spanner-specific write ergonomics, especially explicit update masks.
- Support BigQuery `EXTERNAL_QUERY` use cases where each connection maps to a
  separate Spanner database schema.
- Make generated code suitable for CI, including stale generated file checks.
- Keep schema and query configuration composable enough for larger applications.
- Explain why generated DTO fields are nullable or required.
- Explain how table/index shorthands expand before code is generated.

## Non-Goals

- Do not build a general ORM or query builder.
- Do not hide SQL behind fluent APIs.
- Do not depend on a live Spanner or BigQuery database for the primary workflow.
- Do not generate broad CRUD code unless it follows from declared tables,
  indexes, or explicit write definitions.
- Do not fully emulate every sqlc or yo feature if it does not fit
  Spanner/BigQuery analysis.
- Do not make cross-dialect DTO sharing implicit once the runtime surface grows.
- Do not create BigQuery Spanner external datasets, connections, IAM bindings,
  or Terraform resources. The generator can model external dataset catalog
  shape, but infrastructure creation remains outside this tool.

## Inspirations

The closest reference points are sqlc and yo, but this tool should not copy
either one wholesale.

Relevant sqlc ideas:

- Query-driven generation from SQL files or inline SQL.
- Query annotations such as `:one` and `:many`.
- Generated query constants, parameter structs, result structs, and client
  methods.
- Type overrides, field renames, optional JSON tags, and DB tags.
- CI workflows such as stale-code checks and query validation.
- Plugin-oriented design as a possible long-term extension point.

Relevant yo ideas:

- Spanner table and index models derived from schema metadata.
- Mutation helpers on table structs.
- Explicit column-list update helpers.
- Index-derived read helpers.
- Ignore lists, build tags, custom type mappings, and template customization.

Features to treat carefully:

- sqlc macros should not be copied wholesale. GoogleSQL already has named
  parameters, and runtime SQL expansion such as slice macros weakens the
  "SQL is the contract" property.
- yo-style broad model generation should be opt-in and should not become the
  default product identity.
- Template customization should wait until the internal plan model is stable.

Useful upstream references:

- sqlc configuration: https://docs.sqlc.dev/en/latest/reference/config.html
- sqlc query annotations: https://docs.sqlc.dev/en/latest/reference/query-annotations.html
- sqlc macros: https://docs.sqlc.dev/en/latest/reference/macros.html
- yo documentation: https://pkg.go.dev/go.mercari.io/yo/v2

## Where Current Support Is Documented

This document deliberately does not maintain a claimed-support inventory.
The public YAML contract is defined by README and `config-schema`; the
implemented surface and its drift from this design are tracked in
[`IMPLEMENTATION_STATUS.md`](IMPLEMENTATION_STATUS.md). An earlier revision
duplicated that inventory here, which created a third place to keep in sync.

One framing note that belongs to the design: Roadmap Phase 4.5 tracks
hardening for the current external dataset surface (live verification,
richer vet policy, more generated method behavior). It is not a separate
second feature that changes external datasets into `EXTERNAL_QUERY`
shorthand.

## Config Version Lifecycle

The current public YAML version is `v1alpha`. It is intentionally mutable until
the first stable `v1` config is defined, because this repository has not shipped
a stable query-generator contract yet and breaking simplifications are still
allowed.

The intended lifecycle is:

1. Accept only `version: v1alpha` while the v1 shape is still being refined.
2. Add `version: v1` as the canonical stable spelling once the schema and
   normalizer are fixed.
3. Keep `v1alpha` as a deprecated alias to that same `v1` normalizer for a
   transition period.
4. Do not change the meaning of that post-v1 alias; use `v2alpha` or a later
   preview version for future breaking config experiments.

When the alias is introduced, golden tests should verify that `v1alpha` and
`v1` produce the same normalized plan.

## Design Principles

### SQL Is the Contract

Queries should remain visible and reviewable. Generated code should expose the
SQL as constants unless users explicitly opt out later. Generated methods should
execute declared SQL; they should not synthesize complex query behavior at
runtime except for narrowly scoped features such as optional slice expansion.

### Catalog Objects Are Named Once

Catalogs are named under `catalogs`. Queries, writes, and future model
generation refer to catalog names. This keeps the config ready for:

- one Spanner database,
- one BigQuery dataset,
- multiple Spanner databases behind BigQuery connection IDs,
- mixed BigQuery and Spanner query sets.

The canonical workflow is DDL-first and deterministic. Live introspection, if
added later, should be an import or debugging convenience, not the source of
truth for normal generation.

### Planning Precedes Rendering

Generation should be split into these internal steps:

1. Load config and resolve paths relative to the config file.
2. Build schema catalogs and analyzers.
3. Expand table and index shorthands into SQL.
4. Analyze SQL and write definitions into an intermediate plan.
5. Resolve Go names, field names, parameter names, imports, and conflicts.
6. Render Go files from the plan.

Rendering should not guess names that planning already knows. For example,
write helpers must use the actual generated field name, not recompute a field
name from a column string.

Name collision handling should also live in planning. Generated struct names,
field names, constants, methods, parameter names, and helper symbols should be
allocated from one deterministic namespace per output file. Collisions after Go
identifier normalization, such as `first-name` and `first_name`, should either
receive deterministic suffixes where unambiguous or fail with a plan diagnostic
that identifies the two source declarations. Rendering should never silently
shadow a symbol.

The plan should be dumpable as YAML or JSON. This is not just a debugging aid:
it is how users can review shorthand SQL expansion, DTO merge decisions,
nullability confidence, and generated write column choices.
`explain-plan --stable` is a snapshot aid, not a v1alpha compatibility promise
for every field. It keeps semantic fields and omits volatile audit fields from
both default and `--audit` output. Stable golden tests should focus on
`plan_version` and documented semantic fields while v1alpha remains mutable.

### DTOs Are Shared Only When Type-Compatible

Multiple query results and write inputs can share one Go struct when their
fields are compatible. The merge rules should stay conservative:

- Same field name, scalar kind, repeatedness, and nested shape are compatible.
- Missing fields become nullable in the shared DTO.
- Conflicting scalar kinds are errors.
- Requiredness can only become non-null when explicitly requested or proven.

This supports common table/index cases where a secondary index returns a subset
of the full table row.

Cross-dialect sharing should become explicit as the generator matures. A future
config should distinguish "emit Spanner and BigQuery adapters" from "merge these
Spanner and BigQuery shapes into one DTO". That avoids turning `client: both`
into a broad promise that every Spanner type can safely cross a BigQuery
boundary.

The plan should also preserve field provenance. For each generated field, it
should record which query, write, table, or index contributed that field and why
the final nullable/required decision was made.

For shared DTOs, read and write roles must be tracked separately. A field that
is acceptable as a decoded query result is not automatically safe as an encoded
write input. Plan fields should record roles such as:

- `decode_result`,
- `encode_key`,
- `encode_insert_value`,
- `encode_update_value`.

`vet` should reject write receivers that are missing key columns, missing
insert/update columns, or rely on nullable custom types that do not satisfy the
required Spanner encoder contracts.

The plan should make runtime contracts explicit:

| Role | Spanner runtime contract | BigQuery runtime contract |
| --- | --- | --- |
| Decode query result | `spanner.Row.ToStruct`, `spanner.Decoder`, or compatible field types | `bigquery.RowIterator.Next`, `bigquery.ValueLoader`, or compatible field types |
| Encode DML parameter | `spanner.Statement.Params` values accepted by the Spanner client | not applicable until BigQuery methods are generated |
| Encode mutation value | mutation value types accepted by `cloud.google.com/go/spanner` | not applicable |
| Emit schema metadata | Spanner row-type metadata for analysis/debugging | BigQuery REST `TableSchema` metadata |

A shared DTO can be generated only when every enabled runtime role has a
compatible encoding and decoding contract for that field.

### Spanner Writes Must Be Column-Explicit

For Spanner, updating fewer columns is not a cosmetic optimization. It affects
lock scope and mutation count. Generated write helpers should therefore prefer
explicit column lists derived from `insert.columns` or `update.columns` instead of
struct-wide mutation helpers.

Primary key columns are key inputs and are not updatable fields.

Write semantics should be modeled by branch, not only by a single operation
string. In particular, upsert-like operations need separate insert-column and
update-mask semantics. `INSERT OR UPDATE`, mutation `Replace`, and future
`ON CONFLICT` support must remain distinct because their unspecified-column and
conflict-target behavior differs.

For v1alpha `operation: upsert`, the branch split is constrained to real
Spanner `INSERT OR UPDATE` behavior:
`set(insert.columns)` must equal the table primary key set plus
`set(update.columns)`. Duplicates are rejected. The rendered statement uses a
deterministic key/update column order instead of treating YAML order as branch
semantics. Insert-only non-key columns are rejected because Spanner updates
every specified non-key column when the row already exists. Branch-different
upsert semantics belong to future `ON CONFLICT DO UPDATE SET ...` support.

For v1alpha writes, `key` is the table primary key set. If omitted, it is
inferred from the table DDL. Partial primary keys, unique-index keys, and future
conflict targets should be modeled by future conflict-target fields rather than
overloading `key`.

The resolved plan can record `conflict_strategy: insert_or_update` as normalized
semantics for `operation: upsert`. Public v1alpha YAML does not expose a
`conflict` block because `insert_or_update` is currently the only upsert
strategy.

Cloud Spanner's Go mutation API has `Replace`, but the documented GoogleSQL DML
grammar does not expose an equivalent top-level `INSERT OR REPLACE` form. Until
that syntax is separately verified, `replace` should be mutation-only and DML
helpers must reject it.

Implicitly updating every non-key column is too dangerous as a long-term
default. v1alpha config requires explicit `update.columns` or an explicit
`update.all_non_key_columns` opt-in.

Generated DML helpers and mutation helpers should also remain separate API
shapes. They are useful in different transaction patterns, and generated docs or
lint rules should discourage mixing DML and mutations in the same transaction.

Standard DML and Partitioned DML should also be separate surfaces. Standard DML
is transaction-scoped and can participate in read/write transactions. Partitioned
DML has different execution semantics, restrictions, row-count behavior, and
retry expectations. A future config should model it as a distinct method kind
or execution option, not as a boolean on every generated DML statement.

### Generated Queries Prefer Deterministic Order

Spanner query result order is undefined without `ORDER BY`. Generated Spanner
`table` and `index` shorthand queries therefore default to `order_by: key` so
repeated code generation yields SQL with reviewable deterministic ordering.
For table shorthand, `key` means the table primary key. For index shorthand, it
means the index key columns followed by the base table primary key columns with
duplicates removed. Users can specify `order_by: none` when that ordering would
block a desired query plan optimization. Explicit `sql` and `external_query`
entries own their SQL text and cannot use `order_by`.

For index shorthand over a Spanner `NULL_FILTERED` index, forced index reads
must exclude rows the index omits. The generator adds `IS NOT NULL` predicates
for nullable index key columns, including columns also constrained by generated
key-prefix equality predicates. It skips columns declared `NOT NULL`.

The Spanner-only generated query surface is verified by an integration test
that uses `github.com/apstndb/spanemuboost` to start Cloud Spanner Emulator,
apply the catalog DDL, build a v1alpha plan, and execute each planned Spanner
query. BigQuery `EXTERNAL_QUERY` and external dataset verification remain out
of scope for this emulator-backed test. Cloud Spanner Emulator rejects forced
indexes created with the `CREATE NULL_FILTERED INDEX` clause when source key
columns are nullable, because that validator checks base source column
nullability instead of query predicates. This is true even when the forced-index
query itself has `WHERE key IS NOT NULL` predicates. The test starts the
emulator gateway with `--disable_query_null_filtered_index_check`, which has
been available since Cloud Spanner Emulator v1.5.7, so generated production SQL
runs without emulator-specific query hints.

An optional Spanner Omni integration test runs the same generated SQL against
Omni without the emulator-specific startup flag. Omni also exposes query plans,
so the optional test verifies that the generated index query does not include a
Sort node, that an intentionally different `ORDER BY` does include one, and
that removing a required null-filtered-index predicate is rejected. This keeps
the default emulator test fast while giving a stronger opt-in check for
optimizer-sensitive generated SQL.

### Plan Reports And Contracts Extend Review, Not Generation

`plan-report` uses the same Omni plan source for review artifacts. It should
start at most one Omni runtime per command invocation and create a separate
Spanner database for each referenced Spanner catalog. Separate databases are
enough to isolate catalog DDL; separate Omni containers would only add startup
cost and make reports harder to reproduce. This workflow depends on Spanner
Omni execution-plan support and is for review, testing, and prototyping. The
operator-shape work done so far has not found an observable query optimizer
plan/operator vocabulary difference between Spanner Omni and the Cloud Spanner
DBaaS service. The practical remaining risk is release skew, where DBaaS
Spanner receives optimizer/operator changes before the next Omni release. It is
not part of the primary DDL-first generation path and does not provide
production performance guarantees.

Plan contracts are intentionally outside the main v1alpha config for now.
They are an optional extension of `plan-report`, not part of the primary
DDL-first `generate` / `check` / `vet` workflow. The first experimental shape is
an external file:

```yaml
version: v1alpha-plan-contracts

contracts:
- name: SingerIndexLookupPlan
  target: query/ScanSingerIDsFast
  use:
  - no_explicit_sort
```

`spanner-query-gen plan-report --contracts plan-contracts.yaml --check` can
evaluate those contracts and exit non-zero on violations. Passing `--contracts`
without `--check` still evaluates contracts and includes the results in the
report, but violations do not change the exit code. Unknown query references,
unknown contract fields, and contracts targeting non-Spanner query scopes should
not be mixed with operator violations in the artifact. Unknown or unavailable
targets are reported as `status: not_evaluated` with a machine-readable reason;
`--check` still exits non-zero for that state. The initial predefined contract
set should stay small.
The current experimental set covers `no_explicit_sort`, `no_full_sort`,
`no_minor_sort`, `no_hash_join`, `no_standalone_hash_join`,
`no_push_broadcast_hash_join`, `no_apply_join`, `no_standalone_apply_join`,
`no_distributed_cross_apply`, `no_merge_join`, `no_hash_aggregate`, and
`no_stream_aggregate`, plus metadata/child-link/topology rules `no_full_scan`,
`no_full_scan_without_timestamp_condition`, `require_timestamp_condition`, and
`no_blocking_operator_under_limit`.

Contract matching should use semantic predicates over a normalized operator
family list, not full rendered-plan snapshots. The report records the rendered
plan for humans plus stable review fields such as SQL digest, DDL digest,
normalized operator tree digest, operator families, backend, plan mode, and
optimizer pinning status. A contract validates that configured plan
environment; it is not a universal production performance guarantee. Hint
recommendations should be emitted as remediation text only and must not rewrite
user SQL or config automatically.
Reports should include the plan source (`api: analyze_query`,
`render_tool: spannerplan`), backend identity fields even when the version and
image digest are still `not_recorded`, `backend_identity.source` for the
identity acquisition path, target inclusion/exclusion details, and
operator-tree/operator-family normalization versions. They should also include
`report_version`, `contract_file_version`, and
`contract_evaluator_version` when contracts are evaluated. Each target should
have a canonical target ID such as `query/ScanSingerIDsFast` or
`query/ExternalQuerySingerIDs#inner`. Contract files require `target` because it
remains unambiguous if future report targets include outer SQL, writes, or DML.
v1alpha query, write, catalog, binding, suppression, and contract names are
identifiers (`[A-Za-z_][A-Za-z0-9_]*`) so target IDs do not need escaping.
When the backend runtime cannot expose stable version or image digest fields,
`plan-report` can record manually supplied `--backend-version` and
`--backend-image-digest` values with `backend_identity.source: manual`.
Digest definitions are:

- `sql_sha256`: rendered/generated SQL bytes for the target query.
- `ddl_sha256`: resolved catalog DDL bytes used to set up the target database.
- `operator_tree_sha256`: the normalized operator tree, excluding temporary
  database IDs, timestamps, runtime statistics, row counts, and rendering
  width. The report records `normalization.operator_tree_version` and
  `normalization.operator_family_mapping_version`; while the report format is
  pre-release these identifiers follow the same mutable `v1alpha` philosophy
  as the config and may change in place.

Operator-family normalization follows Spanner's documented query execution
operators. `Sort` and `Sort Limit` are `full_sort`; `Minor Sort` and
`Minor Sort Limit` are `minor_sort`. `explicit_sort` is retained as an umbrella
count for contracts that want to reject both full and minor sorts.
`explicit_sort` is a derived family, not a concrete PlanNode family:

- `normalized_operators[].family` and `operator_families` use concrete
  families only.
- `operator_family_counts["explicit_sort"]` is
  `operator_family_counts["full_sort"] + operator_family_counts["minor_sort"]`.
- Each operator's `subtree_family_counts["explicit_sort"]` follows the same
  invariant for that operator subtree.
- `matched_operator_indexes` for `operator_family: explicit_sort` reports the
  contributing `full_sort` and `minor_sort` operator indexes.

The documented `Distributed Merge Union` operator is observed in raw QueryPlan
as a `Distributed Union` PlanNode with `preserve_subquery_order: true`, not as a
separate `display_name`. The normalizer maps that metadata-bearing
`Distributed Union` to the `distributed_merge_union` family so it remains
reviewable as the documented operator and is not treated as an explicit sort
violation by `explicit_sort`.
Scalar-kind PlanNodes get display-name classification first, so subquery
operators that are kind `SCALAR` in raw plans (Array Subquery, Scalar
Subquery) keep their concrete families; scalar expression nodes such as
Reference, Function, Constant, and Parameter that match no concrete family
classify as `scalar` rather than polluting `unknown`. `unknown` stays
reserved for unclassified relational operators so a strict
`forbid: operator_family: unknown` policy is usable.
`Aggregate` nodes are split into
`hash_aggregate` and `stream_aggregate` when QueryPlan metadata exposes
`iterator_type: Hash` or `iterator_type: Stream`; integration tests can force
those shapes with `GROUP@{GROUP_METHOD=HASH_GROUP} BY` and
`GROUP@{GROUP_METHOD=STREAM_GROUP} BY`. If aggregate metadata is missing or
unknown, the node remains the generic fallback concrete family `aggregate` and
the report must emit `classification_warnings`. `aggregate` is not an umbrella
family for `hash_aggregate` and `stream_aggregate`; it means only the fallback
unclassified aggregate shape. The same vocabulary rule applies to generic
`join`, which is a fallback concrete family for join-like PlanNodes that do not
map to a more specific join family. A predefined or
direct aggregate-family contract that depends on that unknown classification
must fail with `failure_kind: classification_unknown` instead of silently
passing. The rule result should use
`diagnostic_id: plan.aggregate_classification_unknown` and
`matched_operator_indexes` should point at the ambiguous `Aggregate` nodes.
The same conservative policy applies to join-specific contracts when the plan
contains fallback `join` nodes: the result should use
`diagnostic_id: plan.join_classification_unknown` and point at the ambiguous
join node indexes.

`no_hash_join` is intentionally broad: it rejects standalone `hash_join` nodes
and the `push_broadcast_hash_join` wrapper. The narrower
`no_standalone_hash_join` rejects only standalone `hash_join`. In the Spanner
Omni plans observed while designing this feature, the Push Broadcast operator
contains an implementation Hash Join under its `Map` subtree. The normalizer
only treats that node as internal when it is reached through a non-branching
relational path from the `Map` child and consumes a pushed `BatchScan` input.
The implementation node is reported as
`push_broadcast_hash_join_internal_hash_join` rather than `hash_join`. A regular
multi-stage Hash Join nested inside the broadcast side input remains
`hash_join` unless it has that pushed-batch shape. This is more precise than
treating every `Map` descendant Hash Join as internal.

The plan-contract and plan-report JSON Schema `operator_family` enums are
generated from this same normalizer registry and should remain identical.

`no_apply_join` follows the same broad naming rule: it rejects standalone
apply-family operators (`apply_join`, `semi_apply`, and `anti_semi_apply`) and
distributed apply-family wrappers (`distributed_cross_apply` and
`distributed_semi_apply` / `distributed_anti_semi_apply`). Use
`no_standalone_apply_join` for the narrower standalone-only policy. v1alpha
does not define dedicated `no_distributed_semi_apply` or
`no_distributed_anti_semi_apply` predefined contracts; direct
`forbid.operator_family` rules cover those narrower policies. Distributed
wrapper pushed-batch implementation nodes are reported as
`distributed_cross_apply_internal_apply`,
`distributed_semi_apply_internal_apply`, or
`distributed_anti_semi_apply_internal_apply`.

`plan-report` targets structural PLAN output. PROFILE / execution statistics
are explicitly outside the v1alpha contract surface because they are runtime and
data dependent. There is no `--render-mode` flag in v1alpha; every report uses
the structural PLAN mode and still records `render_mode: PLAN` in the artifact.
CEL expressions that reference `execution_stats` should be rejected instead of
downgraded into a weaker contract tier.

`plan-report` should return a report with `status: no_targets` and exit zero
when no Spanner query target is available. `--require-targets` turns that state
into a command error for CI jobs that expect every config to have Spanner
targets. Top-level `status` describes artifact production, not per-query
planning success; `target_summary.planned`, `target_summary.errors`,
`target_summary.skipped`, and `queries[].status` are the target-level contract.
`target_summary.included_count` must equal `planned + errors + skipped`, and
`target_summary.excluded` is always present as either an empty list or records
for targets excluded from the report target set. Skipped targets remain in
`queries[]`, are counted by `target_summary.skipped`, and are not duplicated in
`target_summary.excluded`. For `queries[].status: error | skipped`, only
partial input/error fields are reliable: target identity, query identity, SQL
and SQL digest when rendering succeeded, DDL digest only after DDL resolution,
and the error message. Plan normalization fields are absent and are not
successful plan evidence. Target selection is
Spanner SQL only: generated `kind: sql`,
`kind: table`, and `kind: index` queries with a Spanner catalog are included;
`kind: external_query` `inner_sql` is included because it is Spanner SQL;
BigQuery outer SQL, BigQuery external dataset bindings, and writes are excluded
until future DML/mutation plan support exists. `--stable` is available for
snapshot-oriented output and should avoid host paths, temporary database IDs,
timestamps, and other non-semantic runtime metadata. Report artifacts should
record semantic input identity with `input.config_sha256` and, when contracts
are evaluated, `input.contract_file_sha256`; `contract_file_path` is omitted in
stable output.

CEL is available for user-defined plan contracts in the external
plan-contract file. The preferred contract input is the normalized report view,
not raw renderer output: `backend`, `query`, `operators`, `operator_edges`,
`operator_families`, `operator_family_counts`, and `optimizer`. CEL also
exposes the raw `spannerpb.QueryPlan` as `raw_plan` and its
`[]*spannerpb.PlanNode` as `raw_nodes` for advanced contracts that need all
structural metadata fields, child links, or short representations.
User-facing contracts should prefer the normalized `operators` view when it is
expressive enough, because raw QueryPlan metadata can vary by backend,
optimizer version, and plan mode. Contract evaluation results should include a
stability tier: normalized contracts are `normalized_operator`; `raw_plan`
or `raw_nodes` CEL expressions are `raw_query_plan` with
`check_recommended: false` and `replayable_from_report: false`.
Normalized contracts that reference raw PlanNode metadata copied into
`normalized_operators[]`, such as `subquery_cluster_node` or `call_type`, stay
in the `normalized_operator` tier but add a stability reason listing the
metadata-derived normalized fields read by the expression. This detection is
based on CEL identifiers and select fields, not raw substring matching, so
string literals that mention field names do not affect the stability tier. CEL
input uses normalized operator objects rather than the serialized YAML/JSON
object directly, so absent string metadata fields in `operators[]` and
`operator_edges[]` are exposed as `""`, and absent boolean metadata fields are
exposed as `false`. The serialized report records these replay defaults, and
their canonical `applies_to` order, under `normalization.cel_input_defaults`.
CEL support should remain outside the main v1alpha config until the expression
input schema is stable.

The normalized topology surface is data-oriented rather than macro-oriented:
`operator_edges`, `operators[].child_indexes`,
`operators[].descendant_indexes`, and `operators[].subtree_family_counts`.
For successfully planned targets, `operator_edges` is serialized as `[]` when
there are no edges, matching the CEL empty-list view. This lets users write
descendant/subtree checks in CEL without requiring recursive CEL expressions or
custom predicate macros. Schema-aware contracts
such as `no_back_join` still need catalog metadata such as index-to-base-table
ownership before they can be promoted from candidates to predefined rules.
When `raw_plan` or `raw_nodes` CEL is evaluated under `--check`, the report
should surface `raw_query_plan_contract_used_in_check` as a contract
environment warning so CI users do not rely only on the exit code.

Query plan hint coverage should stay evidence-driven:

- Optimizer pinning hints such as `OPTIMIZER_VERSION` and
  `OPTIMIZER_STATISTICS_PACKAGE` are environment controls. Record them in the
  report before turning them into operator contracts.
- Table scan and index hints such as `FORCE_INDEX`, `SCAN_METHOD`,
  `INDEX_STRATEGY=FORCE_INDEX_UNION`, and `SEEKABLE_KEY_SIZE` can be checked
  through normalized scan metadata or raw QueryPlan metadata.
- Join method hints map to the experimental join families when the resulting
  operator is visible in the plan.
- Group method hints map to `hash_aggregate` and `stream_aggregate` through
  `Aggregate` node metadata.
- Hints whose effects are not reliably visible in simple QueryPlan metadata,
  such as extra parallelism or timestamp predicate pushdown, should remain CEL
  or raw-plan experiments until stable evidence exists.

### EXTERNAL_QUERY Has Multiple Spanner Scopes

BigQuery `EXTERNAL_QUERY` can target different Spanner connections. Each
connection can map to a different Spanner database, DDL, and proto descriptor
set. Proto types are valid inside the corresponding Spanner SQL scope, but
cannot be projected as BigQuery output values.

The analyzer layer should preserve this boundary:

- BigQuery outer query is analyzed with BigQuery catalog rules.
- Each `EXTERNAL_QUERY(connection, sql)` body is analyzed by the Spanner analyzer
  configured for that connection.
- Output type conversion must reject Spanner-only values that BigQuery cannot
  return.

v1alpha config rejects raw BigQuery SQL containing literal `EXTERNAL_QUERY`
calls. The reviewable form separates the inner Spanner SQL from the outer
BigQuery SQL:

```yaml
queries:
- name: ExternalQuerySingerIDs
  catalog: analytics_bigquery
  kind: external_query
  binding: app_conn
  inner_sql: |
    SELECT SingerId
    FROM Singers
  outer_sql: |
    SELECT *
    FROM __external__
  result:
    struct: SingerRow
```

The generator can then analyze `inner_sql` with the named Spanner catalog,
validate the Spanner-to-BigQuery conversion, and render separate constants for
the original inner SQL and the escaped BigQuery SQL.

`outer_sql` placeholder semantics should stay deliberately narrow:

- If `outer_sql` is omitted, the generated BigQuery SQL is
  `SELECT * FROM EXTERNAL_QUERY(...)`.
- If `outer_sql` is supplied, it must contain exactly one `__external__`
  placeholder.
- `__external__` is a config-level table-expression placeholder, not a BigQuery
  identifier. The generator replaces it with the generated `EXTERNAL_QUERY(...)`
  table expression before analyzing the outer BigQuery SQL. The placeholder is
  detected as a lexical identifier token; occurrences inside string literals,
  quoted identifiers, or comments do not count. Users can wrap it in a CTE
  themselves if the outer query needs repeated references or complex aliasing.
- v1alpha public YAML rejects raw BigQuery SQL containing `EXTERNAL_QUERY(...)`.
  Literal-only extraction can be considered later as a legacy migration helper,
  but it is not part of the v1alpha public grammar.
- Inner `ORDER BY` does not guarantee the BigQuery output order. The plan should
  warn when `inner_sql` has `ORDER BY` without an outer `ORDER BY`; `LIMIT 1` is
  exempt because inner ordering can be semantically relevant there. The
  generator must not add `ORDER BY` to `inner_sql` because it can prevent
  root-partitionable Spanner execution. If the final BigQuery result order
  matters, ordering belongs in `outer_sql`.

The `EXTERNAL_QUERY` type-conversion compatibility matrix should be part of the
plan and `vet` rules:

| Spanner GoogleSQL type | BigQuery output | Severity | Remediation |
| --- | --- | --- | --- |
| `BOOL` | `BOOL` | ok | none |
| `INT64` | `INT64` | ok | none |
| `FLOAT64` | `FLOAT64` | ok | none |
| `NUMERIC` | `NUMERIC` | ok | Use explicit casts if range or precision assumptions matter. |
| `STRING` | `STRING` | ok | none |
| `BYTES` | `BYTES` | ok | none |
| `DATE` | `DATE` | ok | none |
| `TIMESTAMP` | `TIMESTAMP` | warning | Cast or format the value in inner SQL if nanosecond precision matters. |
| `JSON` | `JSON` | ok | Runtime loader behavior must be explicit. |
| `ARRAY<T>` | `ARRAY<T>` | recursive | Apply the element-type rule recursively. |
| `STRUCT` | unsupported | error | Project scalar fields or encode the value explicitly. |
| `PROTO` / `ENUM` | unsupported | error | Keep it inside the Spanner inner scope or cast/project supported fields. |

### Spanner External Datasets Are Catalog Bindings

BigQuery Spanner external datasets are dataset-level external sources, not a
variant of `EXTERNAL_QUERY`. BigQuery SQL sees the Spanner tables as ordinary
BigQuery tables under an external dataset; there is no inner Spanner SQL string
to analyze. The generator should therefore model them as BigQuery catalog
extensions:

- Build a Spanner catalog from DDL.
- Normalize the configured dataset into a canonical BigQuery dataset reference
  with `project`, `dataset`, and `path` fields in the plan.
- Project default-schema Spanner tables into the BigQuery catalog under the
  canonical external dataset name.
- Register an unqualified dataset alias when a BigQuery source default project
  is known, so both `dataset.table` and `project.dataset.table` analyze against
  the same projected catalog table.
- Omit or reject columns that BigQuery external datasets cannot expose according
  to `unsupported_columns`.
- Omit or reject named-schema Spanner tables according to
  `projection.named_schema_tables`.
- Treat projected tables as read-only DML and metadata targets.
- Report the projection in `explain-plan` as catalog bindings.
- Reject Spanner tables whose names collide after case-folding, because
  external dataset table names are case-insensitive.
- Reject projected columns whose names collide after case-folding, because the
  BigQuery catalog cannot safely expose both `Foo` and `foo`.
- Reject BigQuery DDL tables and external dataset projections that define the
  same canonical table path. Silent replacement is never allowed; diagnostics
  should include the existing native/catalog table source kind and incoming
  external dataset projection source kind.
- Preserve access mode as canonical plan values `unknown`,
  `end_user_credentials`, or `cloud_resource_connection`, along with optional
  database role, role source, `spanner_database_uri`, location metadata,
  Cloud Resource connection metadata, access verification status, and Data
  Boost access requirement. v1alpha config uses
  `access.cloud_resource_connection_id` for external dataset Cloud Resource
  connections so it does not collide with
  `bindings.external_query_connections[].id`.
- Do not require users to write `not_checked` in config. Verification evidence
  supplied by external tooling is represented by
  `access.verification_evidence` in config without a `source` field; the plan
  normalizes it as external evidence. DDL-only planning records
  `status: not_checked` without a verification source. Plan sources are
  `external_evidence` and `live_probe`; only `live_probe` means the generator
  independently verified the fact.
- Validate audit metadata that can be checked statically: database role encoded
  in recognized `google-cloudspanner://DATABASE_ROLE@/projects/...`
  `spanner_database_uri` values must match any separately configured database
  role. The official
  no-role `google-cloudspanner:/projects/...` form is also recognized and does
  not infer a role. Other schemes or slash forms are preserved as opaque
  metadata and emit a warning instead of a parse-derived error. Also validate
  that `project.location.connection` location matches configured `location`
  after BigQuery location canonicalization.
- Preserve BigQuery-visible metadata and underlying Spanner metadata separately
  in the plan. For example, a projected primary-key column can record
  `underlying_spanner_primary_key: true` while keeping
  `bigquery_key_metadata_visible: false`.
- Use `projected_tables` in catalog binding output because the list can include
  omitted named-schema tables as well as BigQuery-visible tables.

The external dataset projection follows the documented BigQuery limits:

- only default Spanner schema tables are visible,
- primary and foreign key metadata is not visible to BigQuery,
- unsupported Spanner columns are not visible in BigQuery,
- DML and metadata changes against external dataset tables are unsupported,
- `INFORMATION_SCHEMA` is unsupported,
- Read API and Write API are unsupported,
- Data Boost is used by default for queries.

The projection policy is context-specific. `EXTERNAL_QUERY` output conversion
uses error semantics for values BigQuery cannot return. External dataset
projection defaults to `unsupported_columns: reject` and
`named_schema_tables: reject` in v1alpha so lossy catalog projection is
explicit. Users can opt into `unsupported_columns: omit` and
`named_schema_tables: warn_and_omit` to model BigQuery's lossy projection.

External dataset projection has its own compatibility matrix:

| Spanner GoogleSQL object | BigQuery external dataset projection | Default severity |
| --- | --- | --- |
| Default-schema table | Project as `dataset.table` and, when a source project is known, `project.dataset.table`. | ok |
| Named-schema table | Omit from the BigQuery catalog. Explicit references fail as table-not-found. | planning error by default; warning/provenance with `projection.named_schema_tables: warn_and_omit` |
| Supported scalar column | Project as a BigQuery-visible column. | ok |
| Unsupported column such as `PROTO`, `ENUM`, `STRUCT`, or `TOKENLIST` | Omit from the projected table. `SELECT *` excludes it and explicit references fail as column-not-found. | planning error by default; warning/provenance with `unsupported_columns: omit` |
| Hidden column | Omit from the projected table. | info |
| Primary or foreign key metadata | Preserve as underlying Spanner provenance only; do not expose as BigQuery key metadata. | info |
| `INFORMATION_SCHEMA` / `SPANNER_SYS` | Do not project. | unsupported |
| PostgreSQL-dialect Spanner database | Do not project in this generator yet. | unsupported |

Supported scalar columns are BigQuery-visible, but runtime precision and loader
behavior should remain reviewable where relevant. In particular, `TIMESTAMP`
and `NUMERIC` should keep conversion caveats in plan diagnostics until
external-dataset-specific behavior is verified by documentation or live tests.

Access verification is also explicit plan metadata. The status values are
`not_checked`, `verified`, `mismatch`, and `failed`; DDL-only planning emits
`status: not_checked` without a source unless config records verification
evidence from an external tool. Config-supplied evidence is emitted as
`source: external_evidence` and
`independently_verified_by_generator: false`; future live checks can emit
`source: live_probe` and set that boolean to true. `checked_at` is useful audit
metadata but volatile for snapshot output, so the plan marks it with
`volatile: true`. If `database_role` is set while verification is
`not_checked`, the plan emits a warning because static DDL projection is not
role-filtered.

Underlying Spanner metadata and BigQuery-visible metadata must stay separate.
The generator may use Spanner primary keys as provenance for DTO or write
planning, but primary and foreign key metadata is not exposed as BigQuery
external dataset catalog metadata.

This feature should remain DDL-first. Live BigQuery introspection can be added
later as optional verification, but it should not be required for normal CI.
Read-only means external dataset tables cannot be DML or metadata targets. A
future BigQuery execution surface may still allow using an external dataset
table as a source in an `INSERT INTO native_table SELECT ...` query.
Warning suppression affects `vet` output policy only. Projection facts such as
omitted columns and omitted named-schema tables must remain in `projected_tables`
so `explain-plan` continues to show lossy catalog projection.
Lossy projection must also be visible at query level. `SELECT *` records
`star_expansion.projection`, where `selected_output_affected` is true when the
star expansion excluded omitted columns. Explicit-column queries record
relation provenance so reviewers can see that a referenced relation is a
`spanner_external_dataset_projection` even when no star expansion occurred; in
that case `relation_has_omitted_columns` can be true while
`selected_output_affected` remains false. Relation provenance should also
record table role and `writable_target: false`. External dataset relations with
`role: select_source` are allowed; relations with `role: dml_target` or
`role: metadata_target` are rejected with
`external-dataset-dml-target-unsupported` or
`external-dataset-metadata-target-unsupported`.

### Emitted Surfaces Are Explicit

The old `client: both` setting mixed several concepts:

- DTO type choices,
- Spanner row decoding,
- Spanner mutation encoding,
- Spanner DML parameter generation,
- BigQuery row loading,
- future query method generation.

v1alpha uses `emit` to declare which generated API surfaces should exist.
This gives users a way to generate shared DTOs without promising BigQuery query
methods, or to generate Spanner mutations without BigQuery tags.
`emit.*.query_methods` is reserved for a future client-wrapper method surface
and is rejected in v1alpha. Basic query free functions are currently emitted
from `queries[]` without using those flags.

Query DTOs, SQL constants, and basic query free functions are the baseline
output for declared queries. Optional Spanner queries emit typed params structs
and SQL builder helpers instead of a single SQL constant. `emit` controls
additional runtime helper surfaces such as Spanner mutations and DML statements;
it does not gate whether query DTOs, SQL constants, or basic query free
functions are rendered.

Global `emit` gates generated write helpers. v1alpha has no write-level `emit`;
operation-specific incompatibility is applied after the global setting. For
example, `replace` can emit a mutation helper but never emits a DML helper.

### Validation Starts Early

Validation should not be delayed until after broad code generation. The same
plan model needed for rendering can power early `vet` checks such as:

- require update masks for Spanner updates,
- reject mixed DML/mutation transaction patterns when they become expressible,
- flag unknown cross-dialect types,
- flag implicit cross-dialect DTO merges,
- require strict nullability policy in selected packages,
- show table/index shorthand expansion.

## v1alpha Config Shape

The public YAML uses the v1alpha shape when `version: v1alpha` is set. Because
the command has not had a stable release, this config version does not promise
backward compatibility. Unsafe aliases and compact fields can be removed instead
of carried forward while the public surface is still alpha.

```yaml
version: v1alpha

go:
  package: db
  out: db/query_gen.go

emit:
  spanner:
    mutations: true
    dml: true
  bigquery:
    row_loader: true
    table_schema: true

catalogs:
- name: app
  kind: spanner
  ddl: schema/spanner.sql
  proto_descriptors:
  - schema/order_descriptors.pb

- name: analytics
  kind: bigquery
  project: example-project
  ddl: schema/bigquery.sql
  bindings:
    external_query_connections:
    - name: app_conn
      id: example-project.us.example-connection
      spanner_catalog: app
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app
      spanner_database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
      location: US
      access:
        cloud_resource_connection_id: example-project.us.example-connection
      projection:
        unsupported_columns: omit
        named_schema_tables: warn_and_omit

queries:
- name: GetSinger
  catalog: app
  kind: sql
  sql: |
    SELECT SingerId, FirstName, LastName
    FROM Singers
    WHERE SingerId = @SingerId
  result:
    struct: Singer
  params:
  - name: SingerId
    type: INT64

- name: ListSingers
  catalog: app
  kind: table
  table: Singers
  # Omitted order_by defaults to key ordering for Spanner table shorthand.
  result:
    struct: Singer

- name: FindSingerByName
  catalog: app
  kind: index
  index: SingersByName
  key_prefix:
  - LastName
  # Omitted order_by defaults to key ordering for Spanner index shorthand.
  result:
    struct: Singer

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
    struct: Singer

- name: ExternalDatasetSingerIDs
  catalog: analytics
  kind: sql
  sql: |
    SELECT SingerId
    FROM `example-project.analytics_spanner.Singers`
  result:
    struct: Singer

writes:
- name: UpsertSingerName
  catalog: app
  table: Singers
  operation: upsert
  input: Singer
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

- name: ReplaceSinger
  catalog: app
  table: Singers
  operation: replace
  input: Singer
  insert:
    columns:
    - SingerId
    - FirstName
    - LastName
```

Future additions should be grouped rather than added as unrelated top-level
flags:

- `go`: Go rendering options.
- `emit`: which generated adapters and helpers to emit.
- `dto`: DTO sharing and nullable wrapper policy.
- `overrides`: type overrides and field renames.
- `models`: table/index model generation options.
- `rules`: lint or vet-style rules.
- `queries_file`: SQL files with sqlc-like annotations.

Migration from the compact experimental config is explicit:

| Current field | Future field | Notes |
| --- | --- | --- |
| `package` | `go.package` | Top-level rendering fields are not part of v1. |
| `out` | `go.out` | Paths remain config-relative. |
| `client` | `emit` plus future `dto` policy | Split generated surfaces from DTO sharing. |
| `schemas` | `catalogs` | Catalog graph, not only DDL schemas. |
| `source` / `schema` | `catalog` | Queries and writes target catalogs. |
| `spanner_source` | `spanner_catalog` | External bindings target a Spanner catalog. |
| `external_source` | `spanner_database_uri` | URI meaning is explicit. |
| `queries[].federated` | `queries[].kind: external_query` | Keeps `EXTERNAL_QUERY` separate from external datasets. |
| `queries[].required` | `queries[].result.required.fields` | Requiredness belongs to result declaration. |
| `queries[].params` | `queries[].params` | Kept as analyzer input when type inference needs help. |
| `writes[].columns` for insert/replace | `writes[].insert.columns` | Remove as a generic top-level write field. |
| `writes[].update_mask` | `writes[].update.columns` | Required for update-style writes. |
| `writes[].operation: insert_or_update` | `operation: upsert` | Insert columns must equal key plus update columns. The resolved plan records `conflict_strategy: insert_or_update`; public v1alpha YAML has no `conflict` block. |
| `writes[].methods` | global `emit.spanner` | v1alpha has no write-level method selection. |
| `vet.disable` | `rules.suppressions` | Keep required reason and optional owner/expires metadata. |

For Spanner updates, `update.columns` should be required unless users explicitly
choose the broad behavior:

```yaml
writes:
- name: UpdateAllSingerColumns
  catalog: app
  table: Singers
  operation: update
  update:
    all_non_key_columns: true
```

`update.all_non_key_columns: true` should normalize to an explicit broad-update
opt-in in the plan and should be a `vet` warning by default.

Structured config should avoid mixing sentinel strings with column names. An
explicit mask should use a column list shape:

```yaml
update:
  columns:
  - FirstName
  - LastName
```

Compact aliases have no v1alpha YAML support:

- remove `external_schemas` in favor of `bindings.external_query_connections`,
- remove top-level `columns` for update-style writes,
- replace `update_mask: [auto_all_non_key_columns]` with
  `update.all_non_key_columns`,
- keep external dataset projection policy under `projection`.

## Generated Go Shape

The long-term generated package should have three layers.

### Models and DTOs

Models are generated from declared table/index/query result shapes. DTOs may be:

- table model structs,
- query result structs,
- parameter structs,
- write input structs,
- shared structs that cover multiple compatible query/write shapes.

When both Spanner and BigQuery surfaces are emitted for the same DTO, nullable
scalar fields use generated wrapper types such as `NullValue[T]` so one struct
can participate in both BigQuery loading and Spanner decoding or encoding.

### Query Client Methods

The current v1alpha implementation emits basic package-level query free
functions. A richer generated client wrapper remains design work:

- `result: one` means exactly one row; zero rows and more than one row are
  errors.
- `result: maybe_one` allows zero rows but still errors on more than one row.
- `result: many` returns an iterator plus an `All` helper that loads `[]*T`.
- Future method receivers should use a generated client wrapper that can hold a
  Spanner client or a transaction-aware Spanner interface.

The generated error taxonomy should be stable enough for callers to branch on:

- `ErrNoRows` for `result: one` when no row is returned.
- `ErrTooManyRows` for `result: one` or `result: maybe_one` when more than one
  row is returned.
- Wrapped analyzer/client errors for transport, permission, invalid SQL, and
  row decoding failures.

Machine-readable `vet` diagnostics should also use specific IDs rather than a
generic planning-error ID whenever the generator can classify the failure:

- `external-dataset-connection-location-mismatch`
- `external-dataset-database-role-conflict`
- `external-dataset-external-source-unrecognized`
- `external-dataset-access-unverified`
- `external-dataset-unsupported-column-omitted`
- `external-dataset-named-schema-table-omitted`
- `external-dataset-dml-target-unsupported`
- `external-dataset-metadata-target-unsupported`
- `external-dataset-postgresql-dialect-unsupported`
- `external-query-placeholder-invalid-position`
- `external-query-placeholder-missing`
- `external-query-placeholder-duplicate`

`planning-error` remains only as a fallback category for unclassified planning
failures.

`maybe_one` currently returns `(*T, error)`, using `nil, nil` for no rows.
A future pointer/value result policy could switch this to `(T, bool, error)`;
that choice should be global config, not inferred per query.

The current implementation includes both Spanner and BigQuery free functions.
The future generated client-wrapper method surface should still keep BigQuery
and Spanner runtime abstractions separate.

Custom command entries remain future work. Current row-count DML and DML
`THEN RETURN` support is modeled through `queries` result modes; a future
`commands` surface may replace or refine that shape after the query runtime API
settles.

Parameter structs should be inferred from analyzer output when possible. Config
`params` entries are overrides or disambiguation, not the primary source of
truth. When the analyzer cannot infer a parameter type, when ARRAY/STRUCT
parameters need extra shape information, or when `external_query` separates outer
BigQuery parameters from inner Spanner parameters, explicit parameter config is
required.

v1alpha `params` remains narrow analyzer input:

- regular `sql`, `table`, and `index` queries omit `scope`,
- `external_query` params may specify `scope: inner` or `scope: outer`,
- omitted `external_query` scope is inferred only when the parameter appears in
  one SQL scope,
- if both `inner_sql` and `outer_sql` can reference the same unscoped parameter,
  planning fails and asks for explicit scope,
- `params` does not customize generated method signatures yet.

### Write Helpers

Write helpers should remain independent from query methods because mutations
and DML serve different application needs.

For Spanner:

- Mutation helpers return `*spanner.Mutation`.
- DML helpers return `spanner.Statement`.
- Update helpers always use explicit column lists.
- Insert, update, upsert, replace, and delete should keep their branch semantics
  visible.
- Upsert should model insert columns and update masks explicitly; explicit
  conflict targets should be introduced only when generator support for
  Spanner's `ON CONFLICT` DML is added. (Spanner already documents
  `ON CONFLICT DO UPDATE` and `ON CONFLICT DO NOTHING`; the gap here is the
  generator emitting them, not the SQL feature existing.)
- Replace is mutation-only unless a supported DML syntax is verified later.
- Future `ON CONFLICT` support should be modeled with explicit conflict target
  and conflict action fields, not by overloading `operation: upsert`.

## Type System Strategy

The analyzer result should be normalized into an internal type model before Go
rendering. That model should capture:

- source dialect,
- scalar kind,
- nullability,
- repeatedness,
- nested fields,
- proto or enum identity,
- whether a value can cross BigQuery boundaries,
- nullability confidence,
- provenance explaining which query/write/table/index contributed a field.

Nullability confidence should be explicit in the plan. Useful categories
include:

- `non_null_by_schema`: table/index shorthand selected a NOT NULL column,
- `non_null_by_expression`: expression is known non-null, such as a literal or
  `COUNT(*)`,
- `nullable_by_schema`: source column is nullable,
- `nullable_by_join`: outer join or similar construct can introduce nulls,
- `unknown`: analyzer cannot prove nullability,
- `user_required_override`: config forced the field to be required.

Type overrides should apply after analysis and before rendering. Overrides
should support:

- type by dialect type name,
- type by table and column,
- nullable and non-nullable variants,
- import path and package alias,
- custom encoder/decoder expectation,
- JSON tag behavior.

Override resolution should be deterministic. Later, broader rules should not
silently override earlier, narrower choices. The intended order is:

1. query result field override,
2. write input field override,
3. table column override,
4. named proto or enum override,
5. dialect scalar type override,
6. default scalar mapping.

Nullable custom type handling should be explicit:

- non-null fields use `CustomType`,
- nullable fields default to `NullValue[CustomType]`,
- users may specify `CustomNullType`,
- custom nullable types must satisfy the required BigQuery loader and Spanner
  encoder/decoder contracts for the enabled runtime surfaces.

Nullability should remain conservative. If a field is not proven non-null, the
generator should default to nullable. Users can override individual fields, but
`required_policy: strict` should keep rejecting unproven non-null assumptions.
The `explain-plan` output should show these decisions so users can refine the
query or config instead of guessing why wrappers were generated.

## File Layout

Generator-specific code should stay behind an internal package boundary so the
root package remains focused on reusable analyzer, catalog, and type conversion
APIs:

```text
cmd/spanner-query-gen/        CLI and report assembly (nested Go module so
                              spanemuboost/testcontainers/Docker stay out of
                              the root module); CLI and integration tests
internal/querygen             YAML config, schema/query/write planning,
                              generated SQL shorthands, and Go rendering
                              (root module; imported across the module
                              boundary by the cmd module via the path-prefix
                              internal rule)
plancontract/                 nested Go module: PLAN normalization (operator
                              families, topology, digests) and contract
                              evaluation over spannerpb.QueryPlan from any
                              source
tools/                        nested Go module for the Omni developer probes
```

Public root-package helpers can remain thin convenience APIs if they are useful
outside the CLI. Generator-specific dependencies, especially emulator and
container-based integration test dependencies, should stay under
`cmd/spanner-query-gen` or `internal/querygen` instead of being attached to the
root analyzer package. `plancontract` is intentionally kept independent
of `go-googlesql`, `memefish`, and `spanemuboost`; callers pass an
already-collected `spannerpb.QueryPlan` plus normalized operator metadata.

## CLI UX

The CLI should feel like a normal generator with explicit subcommands:

- `spanner-query-gen generate --config path.yaml`
- `spanner-query-gen generate --config path.yaml --out generated.go`
- `spanner-query-gen generate --config path.yaml --out -`
- `spanner-query-gen check --config path.yaml`
- `spanner-query-gen explain-plan --config path.yaml --output yaml`
- `spanner-query-gen explain-plan --config path.yaml --output yaml --audit`
- `spanner-query-gen plan-report --config path.yaml --output markdown`
- `spanner-query-gen plan-report --config path.yaml --output yaml`
- `spanner-query-gen plan-report --config path.yaml --contracts plan-contracts.yaml --check --output yaml`
- `spanner-query-gen vet --config path.yaml --output yaml`
- `spanner-query-gen config-schema --output yaml`
- `spanner-query-gen config-schema --out schemas/spanner-query-gen.v1alpha.schema.json`
- `spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json`
- `spanner-query-gen plan-contract-schema --out schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json`

Expected behavior:

- stdout is generated code only when output is stdout.
- diagnostics go to stderr.
- `check` exits non-zero when the configured generated file is stale.
- config-relative paths are resolved from the config file directory.
- no live service credentials are required for DDL-based generation.
- `vet --output text` keeps stdout empty and writes diagnostics to stderr.
- `vet --output yaml/json` writes the report to stdout and keeps stderr empty
  unless there is a fatal CLI error.
- `plan-report --check` writes the report first, then exits non-zero when a
  configured plan contract is violated.
- `plan-report --require-optimizer-pinning` writes the report first, then exits
  non-zero when optimizer version or statistics package pinning is not recorded.
- `plan-report --optimizer-version` and `--optimizer-statistics-package` pass
  Spanner query options through to `AnalyzeQuery`, making optimizer-pinning
  policy satisfiable without editing user SQL hints.
- `config-schema` writes the reviewable v1alpha JSON Schema to stdout as JSON
  or YAML, or writes JSON/YAML to `--out`, and does not require a config file.
  The checked-in schema file should stay byte-for-byte identical to the command
  output so config-shape changes are reviewable.

`explain-plan` should separate human and machine output:

- `--output summary` for concise review text,
- `--output yaml` for stable CI or golden-test output,
- `--output json` for machine consumers that prefer JSON,
- `--audit` for heavy projection matrix rows and other audit-only metadata.
- `--stable` to omit volatile audit fields such as live or external
  verification timestamps, verification volatility markers, and
  projection matrix docs-last-checked timestamps from snapshot-oriented output.

`rules.suppressions` entries require a reason and use stable scopes:

```yaml
rules:
  suppressions:
  - scope: query/ExternalQueryReviewedQuery
    rule: cross-dialect-timestamp-truncation
    reason: "Reviewed; downstream uses microsecond precision only."
    owner: analytics-platform
    expires: "2026-12-31"
  - scope: catalog-binding/analytics_bigquery.app_dataset
    rule: external-dataset-access-unverified
    reason: Static CI verifies external dataset access outside this generator.
```

`owner` and `expires` are optional metadata, but the plan should preserve them
so teams can audit suppressions in CI.
If present, `expires` must use `YYYY-MM-DD` format. Expired suppression behavior
belongs to the future rule policy layer.

Current `vet` is plan validation, not a full policy engine. It should:

- build the same resolved plan as code generation,
- fail on planning errors,
- print implemented unsuppressed warning summaries and suppression summaries to
  stderr for the default text output while keeping stdout empty,
- expose implemented diagnostics, unsuppressed warnings, and suppressions in
  machine-readable `vet` output on stdout while keeping stderr empty,
- report config parse errors and planning errors as machine-readable
  diagnostics for JSON/YAML output while still returning a non-zero status,
- preserve scoped suppressions in the resolved plan and apply matching
  same-scope rule suppressions in `vet` output.

Diagnostic records include `stage`, `suppressible`, and `suppressed`.
Warning diagnostics are suppressible. Planning and config-parse errors are not
suppressible.

Future `vet` work can add per-rule severity configuration, `--strict`,
warnings-as-errors, source spans, expired suppression behavior, and custom
rules. v1alpha keeps `rules` suppressions-only until that policy layer is
defined.

## Roadmap

Phase headings carry a coarse status as of 2026-06-12. "Shipped" means the
surface exists with tests; open follow-ups stay in `TODO.md` rather than
being re-listed here.

### Phase 1: Stabilize the Current Generator and Plan Model (shipped)

- Keep one-file output.
- Keep YAML config.
- Keep table/index query expansion, including base table primary key columns in
  Spanner index shorthand output.
- Keep Spanner write helpers.
- Keep DML `replace` unsupported and mutation-only unless a supported syntax is
  verified by an integration test.
- Finish the planning layer so rendering has no hidden naming decisions.
- Add field provenance and nullability confidence to the internal plan.
- Add read/write field roles to the internal plan.
- Add BigQuery `EXTERNAL_QUERY` type-conversion decisions to the internal plan.
- Add `explain-plan` output for expanded SQL, result fields, DTO merges, and
  write column choices.
- Provide both human summary and stable YAML/JSON `explain-plan` formats.
- Keep explicit subcommands, stdout output, config schema output, and robust CLI
  tests.
- Document the design and known limitations.

### Phase 2: Validation and Write Semantics (shipped; rule policy layer remains)

- Keep expanding the `vet` command backed by the plan model.
- Add built-in rules for update masks, cross-dialect unknown types, duplicate
  generated names, and risky DTO sharing.
- Keep central `rules.suppressions` with required reasons.
- Split write config into insert columns and update masks. Future `ON CONFLICT`
  support can add explicit conflict target and conflict action fields.
- Require explicit update masks, with `update.all_non_key_columns: true` as an
  explicit broad-update opt-in.
- Keep unsafe compact aliases out of the v1alpha YAML loader.
- Document generated-code guidance for not mixing DML and mutations in one
  transaction.

### Phase 2.5: Minimal Type and Field Overrides (not started)

- Add the minimal override path needed before query methods make DTO contracts
  hard to change.
- Support field rename overrides and scalar type overrides for common blockers
  such as NUMERIC, JSON, BYTES, and TIMESTAMP.
- Validate custom nullable types against the enabled Spanner and BigQuery
  runtime contracts.
- Keep broad template customization deferred until the plan model is stable.

### Phase 3: Spanner Query Methods (mostly shipped as free functions)

- Activate `result.cardinality` semantics for generated query methods.
- Define cardinality behavior for zero and multiple rows.
- Generate parameter structs from analyzer inference plus explicit overrides.
- Generate Spanner query methods.
- Generate Spanner DML execution helpers.
- Keep Partitioned DML as a distinct opt-in execution helper after standard DML
  semantics are covered.
- Keep BigQuery generation at DTO/TableSchema metadata level for now.
- Decide pointer vs value result defaults.

### Phase 4: External Query UX and SQL Files (external_query shipped; SQL files not started)

- Harden `kind: external_query` config that separates BigQuery outer SQL from
  Spanner inner SQL.
- Generate separate constants for inner Spanner SQL and escaped outer BigQuery
  SQL.
- Define parameter rules for outer BigQuery parameters and inner Spanner
  parameters.
- Add `queries_file` or `queries` path entries.
- Support sqlc-like minimal comments:

  ```sql
  -- name: GetSinger :one
  SELECT SingerId, FirstName
  FROM Singers
  WHERE SingerId = @SingerId;
  ```

- Keep YAML entries for generated table/index/write helpers.
- Do not promise sqlc macro compatibility. Consider only GoogleSQL-native
  conveniences that preserve reviewable SQL.

### Phase 4.5: BigQuery Spanner External Dataset Catalog Integration (binding shipped; hardening open)

The basic catalog binding is in current scope. This phase is the hardening
work that makes that binding safer for production review.

- Keep `EXTERNAL_QUERY` support as a query-level external source.
- Keep external datasets as separate BigQuery catalog bindings.
- Add optional live verification for projected table/column visibility,
  database role filtering, connection metadata, and Data Boost permissions.
- Use plan `verification.source` values to distinguish config defaults, external
  evidence, and future live probes while keeping DDL-only default
  `not_checked`.
- Add stricter `vet` policy controls for unsupported columns, named Spanner
  schemas, writes, `INFORMATION_SCHEMA`, and access/execution caveats.
- Add generated-method safeguards for BigQuery execution surfaces while keeping
  external dataset tables read-only DML and metadata targets.
- Do not generate BigQuery execution methods yet.

### Phase 5: Broader Type and Field Overrides (not started)

- Extend type override support to proto, enum, BigQuery BIGNUMERIC, and nested
  field-specific cases.
- Add JSON tag options.
- Implement the documented type override resolution order.
- Keep template customization out of scope until the plan model is stable.

### Phase 6: Opt-In Spanner Model Generation (not started)

- Generate table model structs from Spanner DDL for explicitly included tables.
- Generate index read helpers.
- Add ignore tables and ignore fields.
- Add exact-name or inflection controls.
- Add build tags.

### Phase 7: Later Runtime and Customization Work (not started)

- Add BigQuery query method generation after DTO loading is robust.
- Add schema-change verification against generated query definitions.
- Add examples for CI.
- Consider template customization or plugin interfaces.

## Open Questions

- Should generated query methods return pointers by default?
- Should BigQuery method generation use `cloud.google.com/go/bigquery` directly
  or only generate DTO/schema metadata at first?
- How much sqlc annotation compatibility is useful when GoogleSQL parameter
  syntax is already named?
- (Answered) DML `THEN RETURN` shipped through `queries` row-set result
  modes; a future `commands` surface may still refine the shape.
- Should table models and query DTOs be generated into separate files once the
  generator supports multi-file output?
- Should proto field access remain STRUCT-shadow based until go-googlesql
  exposes public proto and enum type constructors?
- What minimum machine-readable plan schema should be stable enough for CI
  golden tests?
