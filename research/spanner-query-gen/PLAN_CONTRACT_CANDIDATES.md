# Plan Contract Surface And Candidates

Sources:

- [`apstndb/zenn-contents/articles/spanner-query-optimizing-guide.md`](https://github.com/apstndb/zenn-contents/blob/main/articles/spanner-query-optimizing-guide.md)
- [`Cloud Spanner の Partition Query における root-partitionable の判定方法について`](https://zenn.dev/apstndb/articles/9f63227ac8a1da)
- [`Read data in parallel`](https://docs.cloud.google.com/spanner/docs/reads#read_data_in_parallel)
- [`Full-text search`](https://docs.cloud.google.com/spanner/docs/full-text-search)
- [`apstndb/spannerplan/cmd/lintplan`](https://github.com/apstndb/spannerplan/tree/main/cmd/lintplan)
- [`apstndb/spanneropttools`](https://github.com/apstndb/spanneropttools/)

This note translates Spanner query-plan review practices into possible
`spanner-query-gen plan-report` contracts. The v1alpha boundary is structural:
contracts target PLAN output only. They intentionally do not use PROFILE
execution statistics such as rows scanned, rows returned, latency, or CPU.

## Current Surface Summary

This section is a non-normative summary. The current v1alpha contract grammar
is defined by
[`cmd/spanner-query-gen/PLAN_CONTRACTS.md`](../../cmd/spanner-query-gen/PLAN_CONTRACTS.md)
and the checked-in
[`plan-contract` schema](../../schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json).
The machine-readable output shape is defined by the checked-in
[`plan-report` schema](../../schemas/spanner-query-gen.plan-report.v1alpha.schema.json).
Raw CEL variables are named `raw_plan` and `raw_nodes`; they are evaluator
inputs and are not replayable from the serialized YAML/JSON report.

### Predefined `use` Contracts

The following predefined contracts are implemented today. Each one is a small
alias for forbidding one or more normalized operator families.

- `no_explicit_sort`: forbids `explicit_sort`.
  `explicit_sort` is an umbrella count for `full_sort` and `minor_sort`.
- `no_full_sort`: forbids `full_sort`, which covers `Sort` and `Sort Limit`.
  This is useful when full/global sort should be rejected but local minor sort
  is acceptable.
- `no_minor_sort`: forbids `minor_sort`, which covers `Minor Sort` and
  `Minor Sort Limit`.
  The documented Distributed Merge Union behavior remains a separate
  `distributed_merge_union` family when observed as `Distributed Union` with
  `preserve_subquery_order: true`.
- `no_hash_join`: forbids standalone `hash_join` and the
  `push_broadcast_hash_join` wrapper. Internal hash joins under a `Push
  Broadcast Hash Join` wrapper are classified separately as
  `push_broadcast_hash_join_internal_hash_join`, so this contract can reject
  the hash-based wrapper without double-counting the implementation node.
- `no_standalone_hash_join`: forbids only standalone `hash_join`.
- `no_push_broadcast_hash_join`: forbids `push_broadcast_hash_join`.
  This targets the wrapper operator, not the internal implementation node.
- `no_apply_join`: forbids standalone apply-family operators and distributed
  apply-family wrappers. It does not reject `Push Broadcast Hash Join Semi
  Apply` or `Push Broadcast Hash Join Anti Semi Apply`; those wrappers are
  normalized as `push_broadcast_hash_join`.
- `no_standalone_apply_join`: forbids only standalone apply-family operators.
- `no_distributed_cross_apply`: forbids `distributed_cross_apply`.
  Internal apply nodes under a distributed wrapper are classified separately as
  `distributed_cross_apply_internal_apply`.
- Distributed semi/anti-semi wrappers are distinct families:
  `distributed_semi_apply` / `distributed_anti_semi_apply`. Their internal
  implementation apply nodes are reported as
  `distributed_semi_apply_internal_apply` /
  `distributed_anti_semi_apply_internal_apply`. v1alpha does not provide
  dedicated predefined contracts for only those wrappers; use direct
  `forbid.operator_family` rules for that narrower policy.
- `no_merge_join`: forbids `merge_join`.
- `no_hash_aggregate`: forbids `hash_aggregate`.
- `no_stream_aggregate`: forbids `stream_aggregate`.
  This is mostly useful for testing and debugging because the common production
  preference is usually the opposite.
- `no_full_scan`: rejects scan operators whose normalized metadata has
  `full_scan: true`. This is a PLAN-level early warning, not a proof of rows
  scanned.
- `no_full_scan_without_timestamp_condition`: rejects full-scan operators only
  when they do not have a `Timestamp Condition` child link. This is usually a
  better broad guardrail when recent-data commit timestamp scans are allowed.
- `require_timestamp_condition`: requires at least one scan operator with a
  `Timestamp Condition` child link. This is intended for recent-data commit
  timestamp reads where timestamp pruning is expected even if the scan still
  reports `full_scan: true`.
- `no_blocking_operator_under_limit`: rejects stream-blocking descendants below
  `Limit` or `Sort Limit`. This is topology-aware rather than a plain
  operator-family count; it is intended for cases where a limiting operator is
  expected to stream early rows, but a descendant full sort, hash aggregate,
  hash join, BloomFilterBuild, or SpoolBuild would materialize work first.

### Direct `forbid.operator_family` Contracts

`forbid.operator_family` is implemented for the normalized operator-family
enum in the plan contract and plan report schemas and supports `max_count`.
This candidate document intentionally does not duplicate the full enum because
the schema is the source of truth.

Broad sort policy:

```yaml
contracts:
- name: LookupSingerByNamePlan
  target: query/LookupSingerByName
  forbid:
  - operator_family: explicit_sort
```

Narrower sort policies can target either concrete sort family:

```yaml
contracts:
- name: LookupSingerByNamePlan
  target: query/LookupSingerByName
  forbid:
  - operator_family: full_sort
```

Use `operator_family: minor_sort` instead when a contract must reject only
`Minor Sort` / `Minor Sort Limit`.

Other direct operator-family policies can be mixed in the same way:

```yaml
contracts:
- name: LookupSingerByNamePlan
  target: query/LookupSingerByName
  forbid:
  - operator_family: apply_mutations
  - operator_family: change_stream_tvf
  - operator_family: mini_batch_assign
  - operator_family: mini_batch_key_order
  - operator_family: row_count
  - operator_family: hash_join
    max_count: 0
```

### CEL Contracts

CEL contracts are implemented for query-specific conditions that are too narrow
or too experimental to deserve predefined names. The current environment
exposes:

- `backend`
- `query`
- `operators`
- `operator_edges`
- `operator_families`
- `operator_family_counts`
- `optimizer`
- `raw_plan`
- `raw_nodes`

`execution_stats` is explicitly rejected because these are PLAN contracts, not
PROFILE contracts.
`raw_plan` and `raw_nodes` are evaluator inputs, not serialized report fields;
the serialized `queries[].plan` field is a rendered human-readable plan string.

`operator_family_counts` is zero-filled for every known operator family, so CEL
can safely use `operator_family_counts["family"]` even when that family is not
observed in the plan.
`operators[].subtree_family_counts` has the same complete-map semantics, and
`operators[].child_indexes` / `operators[].descendant_indexes` are always
present for analyzed queries.
Derived `blocking_operator` counts are also zero-filled, so CEL can express
broader stream-blocking checks without enumerating every concrete family.
CEL evaluation uses the tool's normalized input rather than the serialized
YAML/JSON object directly. Optional string fields in `operators[]` and
`operator_edges[]` default to `""`, and optional boolean fields default to
`false`; the plan-report artifact records this under
`normalization.cel_input_defaults`.

Useful examples:

```cel
operators.exists(o, o.scan_target == "SingersByLastName")
```

```cel
operators.all(o, !(o.family == "scan" && o.full_scan))
```

```cel
operator_family_counts["hash_join"] == 0 &&
operator_family_counts["push_broadcast_hash_join"] == 0
```

```cel
operators.exists(o, o.family == "hash_aggregate" && o.subtree_family_counts["scan"] == 1)
```

```cel
operators.exists(o, o.family == "limit" && o.subtree_family_counts["blocking_operator"] > 0)
```

Full Text Search contracts are better represented as CEL-positive contracts for
now. Search index access is a `scan` family node with `scan_type:
search_index_scan`, not a dedicated operator family:

```cel
operators.exists(o,
  o.family == "scan" &&
  o.scan_type == "search_index_scan" &&
  o.scan_target == "SearchAlbumsTitleIndex")
```

When a query should stay index-only over a search index, a CEL contract can
forbid a base table scan for the target table:

```cel
operators.all(o,
  !(o.family == "scan" &&
    o.scan_type == "table_scan" &&
    o.scan_target == "SearchAlbums"))
```

CEL can also inspect the raw `spannerpb.QueryPlan` and `spannerpb.PlanNode`
objects, but stable contracts should prefer normalized fields when available.

Approximate root-partitionable PLAN-shape review is possible as a normalized
CEL contract today. Keep it CEL-only for now because it is a positive
disjunction over the whole plan, not a simple `forbid.operator_family` alias.
Prefer `subquery_cluster_node` metadata over enumerating distributed operator
families: observed SELECT plans set it on distributed fragments, while Local
Distributed Unions can be ignored with `call_type != "local"`.

This uses raw PlanNode metadata that `plan-report` exposes in
`normalized_operators[]`. Human-readable `spannerplan` renderers, including
`reference`, intentionally hide `subquery_cluster_node`, so review the YAML/JSON
plan-report artifact when checking this contract. The resulting report keeps
`stability.tier: normalized_operator` but records a stability reason such as
`contract reads metadata-derived normalized fields: call_type, subquery_cluster_node`.

```cel
operator_family_counts["unknown"] == 0 &&
(
  (
    operators.filter(o, o.subquery_cluster_node != "" && o.call_type != "local").size() == 1 &&
    operators.exists(o,
      o.index == 0 &&
      o.family == "distributed_union" &&
      o.subquery_cluster_node != "" &&
      o.call_type != "local")
  ) ||
  (
    operators.filter(o, o.subquery_cluster_node != "" && o.call_type != "local").size() == 0
  )
)
```

This intentionally treats `distributed_merge_union` as not root-partitionable.
The raw display can be `Distributed Union` with `preserve_subquery_order: true`,
but the normalizer maps that case to `distributed_merge_union`; the CEL rule
therefore rejects it even though the node has `subquery_cluster_node`.
The `operator_family_counts["unknown"] == 0` guard is intentionally strict so a
future distributed operator that is not yet normalized cannot silently pass
through the zero-distributed-fragment branch.

Future spool consistency checks can use the normalized `spool_build`,
`spool_scan`, and `spool_name` fields. A repeated-CTE plan can contain a
`SpoolBuild` node and one or more `SpoolScan` nodes with matching
`metadata.spool_name`. v1alpha exposes that raw metadata as
`operators[].spool_name`, so a future contract can verify that every
`spool_scan` refers to an observed `spool_build` name while still remaining a
normalized-operator CEL contract.
Because absent optional string metadata defaults to `""` in the CEL input, the
check must explicitly reject empty spool names instead of silently accepting a
missing field:

```cel
operators
  .filter(o, o.family == "spool_scan")
  .all(scan,
    scan.spool_name != "" &&
    operators.exists(build,
      build.family == "spool_build" &&
      build.spool_name == scan.spool_name))
```

This contract validates the PLAN artifact that `plan-report` collected. It is
not identical to asking Spanner whether `PartitionQuery` accepts the SQL: the
official documentation says `PartitionQuery` runs in batch mode and may choose
a different plan from normal plan inspection. If this becomes a predefined
contract, use a conservative name such as
`approximate_root_partitionable_plan_shape`. A future authoritative check should
be a separate backend probe such as `partition_query_accepts`, which calls
`PartitionQuery` and treats successful partition creation as the source of
truth.

## Prior Implementations Checked

### `spannerplan/cmd/lintplan`

`spannerplan` has a small plan linter that reads PLAN output from stdin and
checks generic operator and metadata patterns:

- `Filter` operator: suggests using Filter Scan with a seek condition.
- `Sort` and `Minor Sort`: suggests using an index whose order matches the
  requested order.
- child link type `Residual Condition`: suggests moving the predicate into a
  scan condition.
- metadata `Full scan: true`: warns about full scans.
- metadata `iterator_type: Hash`: suggests stream aggregation.
- metadata `join_type: Hash`: suggests Cross Apply or Merge Join.

The current v1alpha contracts already cover the broad sort, hash join, hash
aggregate, and full-scan cases. The remaining directly reusable ideas are
`no_residual_condition` and a more positive `require_seekable_scan`.

### `spanneropttools`

`spanneropttools` contains an older, more schema-aware `lintplan` prototype. It
uses an INFORMATION_SCHEMA JSON input to build table/index maps and then
combines those maps with plan topology.

Reusable ideas:

- Build an index-to-base-table map before diagnosing back joins.
- Group scan nodes by owning table, not merely by scan target name.
- Use plan paths and lowest common ancestors to explain where table/index scans
  are joined.
- Resolve child-link variables to render sort keys and aggregate keys.
- Use child link types such as `Key`, `MajorKey`, `MinorKey`, and
  `Residual Condition` as structured signals.
- Reuse schema metadata when proposing or validating index-only plans, including
  key columns, stored columns, `UNIQUE`, and `NULL_FILTERED`.

The most important lesson for this repository is that `no_back_join` must be
schema-aware. A plan subtree that contains an index scan of one table and a
table scan of another table can be a normal join, not a back join. A correct
contract must confirm that the index scan target belongs to the same base table
as the table scan before reporting a back join.

## Strong Candidates For Predefined Contracts

### Promoted: `no_full_scan`

`no_full_scan` is now implemented as a predefined metadata rule. It catches
OLTP queries that scan a whole table or index when a selective seek was
expected.

Why it matters: `Full scan: true` on growing tables is a common early warning
sign. This is especially important in read-write transactions because broad
scans can widen the lock footprint.

Current CEL equivalent:

```cel
operators.all(o, !o.full_scan)
```

Remaining follow-up:

- Add allowlists because small/static master tables and intended full index
  scans are legitimate. Recent-data commit timestamp reads with
  `Timestamp Condition` are another legitimate case where a plan-level full
  scan can still have reduced storage I/O.
- Consider a direct scan predicate shape such as:

```yaml
forbid:
- scan:
    full_scan: true
    except_targets:
    - SmallMaster
```

### Promoted: `no_full_scan_without_timestamp_condition`

`no_full_scan_without_timestamp_condition` is now implemented as a predefined
metadata plus child-link rule. It fails only full-scan operators that do not
have a `Timestamp Condition` child link.

Why it matters: this captures the practical guardrail suggested by recent
commit timestamp optimization work. Plain `no_full_scan` is still useful for
strict lookup contracts, but it is too blunt when a recent-data path is allowed
to rely on storage-level timestamp pruning.

Current CEL equivalent:

```cel
operators.all(o,
  !(o.family == "scan" && o.full_scan) ||
  operator_edges.exists(e,
    e.parent_index == o.index &&
    e.type == "Timestamp Condition"))
```

Limitations:

- Like `require_timestamp_condition`, this is PLAN-only and does not prove
  rows-scanned reduction.
- It allows every full scan with `Timestamp Condition`; if only specific
  queries or tables should use this exception, use a narrower CEL contract
  until scan-target allowlists are added.

### Promoted: `require_timestamp_condition`

`require_timestamp_condition` is now implemented as a predefined child-link
rule. It checks `operator_edges[].type == "Timestamp Condition"` and reports
the parent scan operator indexes.

Why it matters: commit timestamp predicate pushdown can change the
index-design decision. A query that previously would have needed a secondary
timestamp index for recent-data filtering might be acceptable as a table scan
with a `Timestamp Condition`, avoiding index storage, write amplification, and
timestamp-index hotspot risk.

Current CEL equivalent:

```cel
operator_edges.exists(e, e.type == "Timestamp Condition")
```

Limitations:

- This is a structural PLAN signal. It confirms the optimizer produced the
  timestamp-pruning predicate, but it does not prove the actual rows-scanned
  reduction. That still needs PROFILE or Query Stats.
- It does not validate that the predicate is on the intended column. A future
  normalized scalar summary could expose the condition expression or referenced
  column more directly.
- It should normally be used instead of, not together with, `no_full_scan` for
  recent-data commit timestamp read paths.

### `require_scan_target`

Intent: assert that a query uses the expected table or index.

Why it matters: many review cases rely on selecting a specific secondary index,
often with `FORCE_INDEX`, and then verifying that the plan actually uses that
index.

Current CEL equivalent:

```cel
operators.exists(o, o.scan_target == "SingersByLastName")
```

Implementation notes:

- This is a good candidate for a direct rule because typo-safe scan target
  review is more readable than CEL.
- The rule should distinguish table scans, index scans, and batch scans using
  normalized `scan_type` plus a future `scan_target_kind`.

### `require_seekable_scan`

Intent: verify that filters are applied as seek conditions rather than merely
as residual filters after scanning.

Why it matters: useful seek predicates are materially different from residual
filtering. `seekable_key_size` is the stable metadata we currently expose that
approximates this.

Current CEL example:

```cel
operators.exists(o, o.scan_target == "SingersByLastName" && o.seekable_key_size == "1")
```

For sharded timestamp-order queries, the observed difference between
`shard_id BETWEEN 0 AND 9` and an equality-probe rewrite is exactly this kind
of signal: the range form can still use the index but only report
`seekable_key_size == "1"` with timestamp filtering left outside the full key
seek, while the per-shard equality probe can reach `seekable_key_size == "2"`.

Potential direct rule:

```yaml
require:
- scan:
    target: SingersByLastName
    min_seekable_key_size: 1
```

Limitations:

- `seekable_key_size` alone cannot prove the seek range is semantically narrow.
  A range such as `UserId BETWEEN min AND max` can still be too broad. That
  needs query-specific review or future SQL/parameter-aware checks.
- It also cannot replace PROFILE when the plan shape is identical but scan-row
  count changes. `GROUPBY_SCAN_OPTIMIZATION` is a known example where the plan
  can remain structurally similar while rows scanned differ.
- Numeric comparison should use a normalized `seekable_key_size_int` instead of
  string comparison.

### `no_residual_condition`

Intent: reject plans where an important predicate remains a residual condition
instead of becoming a scan condition.

Why it matters: residual filtering can read many rows and then discard them,
while scan conditions can narrow the scan itself.

Implementation notes:

- This needs normalized child links, not just flat operators.
- It should probably support target filters because residual conditions are not
  always harmful on tiny inputs.

### `no_back_join`

Intent: detect an index scan followed by a distributed fetch from the base table
when the query was expected to be index-only.

Why it matters: back join is a major source of avoidable distributed work.
Selecting fewer columns or adding `STORING` columns can turn the query into an
index-only scan.

Correctness requirement:

- This must not treat "any index scan plus any table scan in the same join
  subtree" as a back join.
- The scan target of the index must be resolved to its base table, and the table
  scan must target that same base table.
- Without the index-to-base-table mapping, a normal join between unrelated
  tables can be misdiagnosed as a back join.

Implementation notes:

- The useful pattern is structural and schema-aware:
  - a table/index ownership map from the parsed schema catalog,
  - a scan map grouped by owning table,
  - a topology check such as lowest common ancestor or relational-path
    containment,
  - a table scan of the same base table reachable from the index scan through
    the join/fetch shape.
- This should not simply forbid every `distributed_cross_apply`; distributed
  apply can be an intentional join method.
- We already separate wrapper/internal apply families, but `no_back_join` needs
  topology plus scan target ownership, not family counts alone.

### `index_only_scan`

Intent: require that a query can be satisfied by one or more index scans without
fetching the base table.

Why it matters: this is the positive version of `no_back_join`, matching the
recommendation to reduce selected columns or add `STORING` columns.

Implementation notes:

- This is likely easier to review as a positive contract than as a negative
  `no_back_join` for generated table/index shorthands.
- It should accept allowed scan targets and reject same-table table scans unless
  they are known synthetic implementation scans.
- It should use the schema catalog to confirm index ownership and coverage.

### `require_order_preserving`

Intent: for ordered pagination or top-N lookup, assert that the plan preserves
the intended order instead of sorting late.

Why it matters: `ORDER BY ... LIMIT` becomes efficient only when the index order
lets Spanner stop early.

For globally ordered access over sharded timestamp keys, a full/global
`Sort Limit` is usually the shape to reject. Per-shard local limits followed by
a small global merge/sort can be acceptable, so this candidate should compose
with `no_full_sort`, `no_blocking_operator_under_limit`, and future
seekability/index-only checks rather than replacing them.

Implementation notes:

- Expose `order_preserving` as normalized metadata when QueryPlan provides it.
- Consider extracting order keys from `Key`, `MajorKey`, and `MinorKey` child
  links, following the prior `spanneropttools` approach.
- This remains a structural PLAN contract. It does not prove that `LIMIT`
  reduced rows scanned; that is PROFILE-only.

### `require_stream_aggregate`

Intent: positively assert that aggregation uses stream aggregation.

Why it matters: matching scan order to `GROUP BY` keys can avoid hash
aggregation and can also avoid distributed back join when the index is covering.

Current workaround:

```yaml
contracts:
- name: AggregateBySingerPlan
  target: query/AggregateBySinger
  use:
  - no_hash_aggregate
```

Refinement candidates:

- Add positive `require_stream_aggregate` as a clearer alias for the common
  expectation.
- Add `no_aggregate_after_back_join` after topology support exists.
- Extract aggregate keys from child links for review output.

### `no_distributed_join`

Intent: provide a broad OLTP guardrail for queries that should be single-table
or single-index lookups and should not perform distributed joins.

Potential definition:

- forbid `hash_join`
- forbid `push_broadcast_hash_join`
- forbid `distributed_cross_apply`
- optionally forbid `merge_join` if observed plans show it as distributed in
  the relevant environment

Implementation notes:

- This should be a named preset, not an implicit expansion of `no_hash_join`,
  because it also covers non-hash distributed join shapes.
- It is too broad for analytical or intentional join queries, but useful for
  generated lookup shorthands where a join indicates a missing covering index or
  an accidental table fetch.

## Good CEL-Only Contracts For Now

These are useful but probably too query-specific for predefined v1alpha names.

### Require A Specific Index Scan And No Table Scan

```cel
operators.exists(o,
  o.scan_target == "SongsBySongNameStoring" &&
  o.scan_type == "index_scan") &&
operators.all(o, o.scan_type != "table_scan")
```

This is only a rough approximation of index-only behavior because it does not
know whether an index belongs to the same base table as a table scan.

### Require A Minimum Seekable Key Prefix

```cel
operators.exists(o, o.scan_target == "UserAccessLogByShardIdLastAccess" && o.seekable_key_size == "2")
```

### Forbid A Residual-Only Full Scan

```cel
operators.all(o, !(o.family == "scan" && o.full_scan && o.seekable_key_size == ""))
```

This is weaker than true predicate classification because residual predicate
labels are not yet normalized.

## Needs More Normalized Plan Shape

The current normalized `operators` view is intentionally flat. Several useful
contracts need parent/child, subtree, or schema relationships.

Implemented normalized topology additions:

- `operator_edges`: parent index, child index, child-link type, and child-link
  variable.
- `operators[].child_indexes`: direct child PlanNode indexes.
- `operators[].descendant_indexes`: transitive child PlanNode indexes.
- `operators[].subtree_family_counts`: operator-family counts for the subtree,
  including the operator itself.

Recommended next normalized additions:

- `relational_children`: child indexes filtered to relational operators.
- `order_preserving` metadata when present.
- extracted `order_keys` and `aggregate_keys` from `Key`, `MajorKey`, and
  `MinorKey` child links.
- normalized `has_residual_condition` or residual child-link summaries.
- normalized `has_timestamp_condition` or timestamp-condition child-link
  summaries, ideally with the referenced commit timestamp column when that can
  be extracted safely.
- `scan_target_kind`: `table`, `index`, `batch`, or `unknown`.
- `base_table`: for table scans this is the table itself; for index scans this
  is resolved from the schema catalog.
- index coverage metadata from the schema catalog: key columns, stored columns,
  and null-filtered/null-filtering attributes where available.
- numeric `seekable_key_size_int` in addition to the raw string.

Adding these keeps most advanced contracts in CEL while making them readable
and less dependent on raw `spannerpb.PlanNode` shape. It also avoids false
positives for contracts such as `no_back_join`, where plan topology alone is not
enough.

## Explicitly Out Of Scope For PLAN Contracts

The guide also recommends checks that require PROFILE execution stats or
production telemetry. These should not become v1alpha plan contracts:

- `rows_scanned` versus `rows_returned` ratio.
- Verifying that reducing `LIMIT` actually reduces scanned rows.
- Confirming commit timestamp scan optimization through runtime rows scanned.
- Query Stats thresholds such as total CPU, latency, bytes, execution count, or
  rows scanned.
- Detecting unused or redundant indexes across the whole workload.

Those are still valuable, but they belong in a future PROFILE/telemetry
workflow or a schema/workload analysis command, not in structural `plan-report`
contracts.
