# spanner-query-gen Implementation Status

This document is a review aid for the mutable `v1alpha` implementation. The
README describes the current command UX; `DESIGN.md` describes the intended
architecture and future direction. This file calls out current drift and
deliberate deferrals.

## Implemented v1alpha Surface

- Strict `version: v1alpha` config parsing with unknown-field rejection.
- `generate`, `check`, `explain-plan`, `vet`, `config-schema`,
  `plan-report-schema`, and `plan-contract-schema` subcommands.
- Query result DTO and SQL constant generation for Spanner, BigQuery, and
  reviewable `external_query` declarations.
- Basic runtime query free functions for Spanner and BigQuery query
  cardinalities (`one`, `maybe_one`, `many`) plus row-set DML
  `THEN RETURN`. Optional Spanner queries emit typed params structs and
  `Build<Name>SQL` helpers instead of a single SQL constant.
- Spanner `table` and `index` shorthand query generation with deterministic
  `order_by: key` by default and opt-out `order_by: none`.
- Spanner write helper generation for mutation and DML surfaces, including
  update column masks.
- External BigQuery bindings for `EXTERNAL_QUERY` and Spanner external dataset
  modeling as reviewable analyzer inputs.
- Analyzer parameter type declarations via `queries[].params`, including
  `inner` / `outer` scopes for external queries.
- JSON Schema generation and checked-in schema files for the main config,
  plan-report output, and plan contract files.
- Optional Spanner Emulator and Spanner Omni integration tests under
  `cmd/spanner-query-gen`.
- Optional `plan-report` workflow using Spanner Omni `AnalyzeQuery` and
  `spannerplan` to produce Markdown, YAML, or JSON review artifacts.
- Machine-readable YAML output is produced from the same JSON/protojson-shaped
  data used for JSON output where practical, using `goccy/go-yaml` JSON-to-YAML
  conversion instead of direct Go YAML marshaling.
- Experimental external plan contract files with canonical `target` IDs,
  exactly one rule mode per entry (`use`, `forbid`, or `cel`), normalized
  operator-family checks, normalized topology fields, raw QueryPlan CEL for
  advanced cases, and PLAN-only scope.
- Plan contract evaluation lives in `internal/plancontract`, which depends on
  normalized plan-report data and `spannerpb.QueryPlan`, but not on
  `go-googlesql`, `memefish`, or `spanemuboost`.

## Recent Design Drift Resolved

- `plan-report` no longer exposes `--render-mode`; v1alpha always analyzes
  structural PLAN output and records `render_mode: PLAN`.
- Plan contract entries no longer accept `query`, `scope`, or `backend`.
  Target selection is the canonical `target` field only.
- Contract entries require exactly one rule mode. Multiple predefined rules may
  still be placed in one `use` array.
- `forbid.operator_family` is validated against the known normalized family
  list and is also emitted as a schema enum.
- `queries[].status: ok` always serializes `operator_edges`, using `[]` for
  edgeless plans, and error/skipped targets do not serialize stale plan fields.
- `normalization.cel_input_defaults` records the replay defaults for optional
  `operators[]` and `operator_edges[]` fields used by CEL evaluation; the
  schema fixes `applies_to` to that exact target set and canonical order.
- Raw CEL variables are named `raw_plan` and `raw_nodes` to avoid confusion
  with the serialized human-readable `queries[].plan` report field. Raw CEL
  stability now records `replayable_from_report: false`.
- Plan-report schema conditionals couple `contract_rule_result.rule` to its
  `source` kind, so CEL results cannot claim `source: use/<predefined>` and
  forbid-operator-family results cannot claim `source: cel`.
- `no_full_scan` is implemented as a metadata rule rather than an operator
  family rule; its result uses `rule: forbid_full_scan` and reports matching
  scan operator indexes.
- `no_full_scan_without_timestamp_condition` is implemented as the practical
  recent-data variant: it fails only full-scan operators that do not have a
  `Timestamp Condition` child link.
- `require_timestamp_condition` is implemented as a child-link rule over
  `operator_edges[].type == "Timestamp Condition"`; it is intended for
  recent-data commit timestamp reads where storage-level timestamp pruning is
  expected even if the scan still reports `full_scan: true`.
- CEL rule failures use `failure_kind: violation`; `classification_unknown`
  remains reserved for predefined or direct `forbid_operator_family` rules.
- CEL contracts that read metadata-derived normalized fields, such as
  `call_type` or `subquery_cluster_node`, stay in the normalized tier but list
  those field names in `stability.reasons`.
- `writePlanReport` validates target summary, contract summary, topology, and
  matched-operator-index invariants before serializing artifacts, so mismatched
  report internals fail as generator errors.
- Push Broadcast Hash Join normalization distinguishes the wrapper from the
  implementation Hash Join reached through a non-branching relational path and
  consuming pushed batch input. Regular nested Hash Joins remain visible as
  `hash_join`.
- Sort normalization distinguishes `full_sort` (`Sort` / `Sort Limit`) from
  `minor_sort` (`Minor Sort` / `Minor Sort Limit`), while retaining
  `explicit_sort` as a derived umbrella count for broader contracts.
- Distributed Cross Apply normalization similarly distinguishes the wrapper
  from the pushed-batch implementation Cross Apply node.

## Design-Only Or Partial Work

- `emit.spanner.query_methods` and `emit.bigquery.query_methods` are still
  reserved and rejected as explicit config gates. Current query methods are the
  built-in v1alpha free-function surface and are not controlled by those flags.
- A richer generated client wrapper, stable query error taxonomy, and
  pointer/value result-policy configuration remain design work.
- Custom command entries remain future work. Current row-count DML and DML
  `THEN RETURN` support is modeled through `queries` result modes.
- BigQuery external dataset support is analyzer/config modeling only. Live
  BigQuery or Terraform evidence collection remains outside the generator.
- Plan reports separate `optimizer.requested` from `optimizer.effective`.
  Requested optimizer options and pinning absence are recorded; effective
  optimizer version/statistics package are currently `not_recorded` because the
  backend source split is not captured yet.
- Backend identity records `not_recorded` for Omni version and image digest by
  default with `source: spanemuboost`. `plan-report` can also record manually
  supplied `--backend-version` and `--backend-image-digest` values with
  `source: manual` as caller assertions, not observed backend evidence, until
  the backend runtime exposes stable automatic identity fields.
- Plan contracts are review contracts for a described plan environment, not
  production performance guarantees. They do not use PROFILE execution stats.
- The normalized operator-family catalog is bounded, registry-driven, and
  evidence-based. It grows only when observed plans, fixtures, or contract use
  cases require a new family.
- Query-plan remediation is advisory only. The generator never rewrites user SQL
  or config to satisfy a contract.
