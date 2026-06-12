# Plan Contracts

`spanner-query-gen plan-report` can evaluate optional plan contracts from a
separate `v1alpha-plan-contracts` YAML file. Contracts are intentionally outside
the main `spanner-query-gen.yaml` so the primary config stays focused on DDL,
SQL, DTOs, and write helpers.

Plan contracts target structural PLAN output only. They do not use PROFILE
runtime statistics such as rows scanned, rows returned, latency, or CPU. The
research note
[`Optimizer Decision Control And Plan Observability`](../../research/spanner-query-plan-shape/OPTIMIZER_DECISION_CONTROL_AND_OBSERVABILITY.md)
tracks which optimizer-version differences and hints are visible enough to be
PLAN-contract candidates.

The
Spanner Omni execution-plan feature is a Preview / Pre-GA feature in the
official Spanner Omni documentation, and is described as suitable for
development, testing, prototyping, and demonstration. In the operator work done
for this project, no observable query optimizer plan/operator vocabulary
difference has been found between Spanner Omni and the Cloud Spanner DBaaS
service. The remaining compatibility risk is release skew: DBaaS Spanner can
receive optimizer/operator changes before a corresponding Spanner Omni release.
Treat `plan-report` as a review workflow for a described plan environment, not
as a production performance guarantee.

The evaluator is implemented in `internal/plancontract` and only consumes an
already-built plan-report projection plus optional raw `spannerpb.QueryPlan`.
It does not parse DDL, invoke the GoogleSQL frontend, or start Spanner Omni.

Source:
[`View Spanner Omni execution plans`](https://docs.cloud.google.com/spanner-omni/execution-plans)

## Minimal Contract File

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

Passing `--contracts` evaluates contracts and includes results in the report
while keeping exit code 0 on violations or unavailable targets. Adding
`--check` makes violations and `not_evaluated` contract targets exit non-zero.

`contract_summary.status` describes contract truth, not process exit status:
it is `pass` only when every configured contract was evaluated and passed, and
it is `fail` when any contract failed or was `not_evaluated`. Process exit is
controlled by `contract_evaluation_mode`: `report_only` keeps exit code 0 even
when `contract_summary.status` is `fail`, while `check` exits non-zero when the
summary status is `fail`.

## Target IDs

`target` is the required stable selector.

- `query/ScanSingerIDsFast`: a regular configured query target.
- `query/ExternalQuerySingerIDs#inner`: the Spanner inner SQL of a configured
  `kind: external_query` target.

Configured query and contract names are v1alpha identifiers
(`[A-Za-z_][A-Za-z0-9_]*`) so target IDs do not need escaping.

Contract names must be unique within one contract file. Runtime validation
rejects duplicate names because report consumers may use
`contract_evaluations[].name` as a stable key.

Unknown query references and skipped/error targets are represented in the
artifact as `status: not_evaluated` with `reason: target_not_found` or
`reason: target_error`.

## Rule Modes

Each contract entry must use exactly one rule mode:

- `use`: predefined contract names.
- `forbid`: direct normalized operator-family counts.
- `cel`: CEL expression over the normalized report view and raw plan inputs.

Direct `forbid` rules count normalized operator nodes after operator-family
normalization. If `max_count` is omitted, the tool treats it as `0` and the
report always emits the resolved `max_count`.

Most operator families are concrete: one `normalized_operators[]` entry has one
concrete `family`. `explicit_sort` and `blocking_operator` are different. They
are derived umbrella families used only for counts and contracts:

```text
operator_family_counts["explicit_sort"]
  == operator_family_counts["full_sort"]
   + operator_family_counts["minor_sort"]

operators[i].subtree_family_counts["explicit_sort"]
  == operators[i].subtree_family_counts["full_sort"]
   + operators[i].subtree_family_counts["minor_sort"]

operator_family_counts["blocking_operator"]
  == count(full_sort, hash_aggregate, hash_join, push_broadcast_hash_join,
           aggregate, join, bloom_filter_build, spool_build)
```

`operator_families` lists concrete families observed in
`normalized_operators[].family`. Derived families such as `explicit_sort` and
`blocking_operator` may appear in `operator_family_counts` and
`subtree_family_counts`, but they do not appear as a PlanNode's concrete
family.

`aggregate` and `join` are generic fallback concrete families, not derived
umbrella families. `aggregate` means an `Aggregate` PlanNode whose hash/stream
classification could not be resolved from metadata. It is not a shorthand for
`hash_aggregate + stream_aggregate`, and direct `forbid.operator_family:
aggregate` catches only generic fallback aggregate nodes. `join` similarly
means a join-like PlanNode that could not be mapped to a more specific join
family such as `hash_join`, `merge_join`, `apply_join`, or
`push_broadcast_hash_join`.

## Operator Families

The authoritative machine-readable list is the `operator_family` enum generated
in `schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json` and
`schemas/spanner-query-gen.plan-report.v1alpha.schema.json`. The implementation
source is `internal/plancontract.KnownOperatorFamilies()`.

Every family below is valid in `forbid.operator_family` and
`operator_family_counts`. All families except `explicit_sort` and
`blocking_operator` can appear as a concrete `normalized_operators[].family`;
`explicit_sort` is derived from `full_sort + minor_sort`, and
`blocking_operator` is derived from stream-blocking families for count and
contract convenience.

| Family | Kind | Meaning |
| --- | --- | --- |
| `aggregate` | Concrete fallback | `Aggregate` PlanNode whose hash/stream classification could not be resolved. |
| `anti_semi_apply` | Concrete | Standalone Anti Semi Apply operator. |
| `apply_join` | Concrete | Standalone Apply Join, Cross Apply, or Outer Apply operator. |
| `apply_mutations` | Concrete | DML `Apply Mutations` write operator. |
| `array_subquery` | Concrete | Array Subquery operator. |
| `array_unnest` | Concrete | Array Unnest operator. |
| `bloom_filter_build` | Concrete | BloomFilterBuild operator. |
| `blocking_operator` | Derived | Count-only umbrella for stream-blocking operators; currently `full_sort`, `hash_aggregate`, `hash_join`, `push_broadcast_hash_join`, fallback `aggregate` / `join`, `bloom_filter_build`, and `spool_build`. |
| `change_stream_tvf` | Concrete | ChangeStream TVF operator. |
| `compute` | Concrete | Compute operator. |
| `compute_struct` | Concrete | Compute Struct operator. |
| `create_batch` | Concrete | Create Batch operator. |
| `data_block_to_row` | Concrete | DataBlockToRow operator. |
| `distributed_anti_semi_apply` | Concrete | Distributed Anti Semi Apply wrapper. |
| `distributed_anti_semi_apply_internal_apply` | Concrete contextual | Semi/Anti Semi Apply inside a Distributed Anti Semi Apply wrapper when it consumes a Batch Scan. |
| `distributed_cross_apply` | Concrete | Distributed Cross Apply, Distributed Outer Apply, or Distributed Apply wrapper. |
| `distributed_cross_apply_internal_apply` | Concrete contextual | Cross/Outer Apply inside a Distributed Cross/Outer Apply wrapper when it consumes a Batch Scan. |
| `distributed_merge_union` | Concrete | Documented Distributed Merge Union behavior, normalized from `Distributed Union` with `preserve_subquery_order: true` or an explicit Distributed Merge Union display name. |
| `distributed_semi_apply` | Concrete | Distributed Semi Apply wrapper. |
| `distributed_semi_apply_internal_apply` | Concrete contextual | Semi/Anti Semi Apply inside a Distributed Semi Apply wrapper when it consumes a Batch Scan. |
| `distributed_union` | Concrete | Distributed Union without Distributed Merge Union semantics. Local Distributed Union still maps here; inspect `call_type` when locality matters. |
| `empty_relation` | Concrete | Empty Relation operator. |
| `explicit_sort` | Derived | Count-only umbrella for `full_sort + minor_sort`; it never appears as `normalized_operators[].family`. |
| `filter` | Concrete | Filter operator. |
| `filter_scan` | Concrete | Filter Scan operator. |
| `full_sort` | Concrete | Sort, Sort Limit, Local Sort, or Local Sort Limit. |
| `hash_aggregate` | Concrete | Aggregate classified as hash aggregation from metadata or display/description text. |
| `hash_join` | Concrete | Hash Join not classified as the internal hash join of Push Broadcast Hash Join. |
| `join` | Concrete fallback | Join-like PlanNode that could not be mapped to a specific join family. |
| `key_range_accumulator` | Concrete | KeyRangeAccumulator operator. |
| `limit` | Concrete | Global Limit, Limit, or Local Limit operator. |
| `merge_join` | Concrete | Merge Join operator. |
| `mini_batch_assign` | Concrete | MiniBatchAssign operator. |
| `mini_batch_key_order` | Concrete | MiniBatchKeyOrder operator. |
| `minor_sort` | Concrete | Minor Sort, Minor Sort Limit, Local Minor Sort, or Local Minor Sort Limit. |
| `push_broadcast_hash_join` | Concrete | Push Broadcast Hash Join wrapper, including outer/semi/anti-semi variants. |
| `push_broadcast_hash_join_internal_hash_join` | Concrete contextual | Hash Join inside a Push Broadcast Hash Join wrapper when it consumes a Batch Scan. |
| `random_id_assign` | Concrete | Random ID Assign operator. |
| `recursive_spool_scan` | Concrete | Recursive Spool Scan operator. |
| `recursive_union` | Concrete | Recursive Union operator. |
| `row_count` | Concrete | RowCount operator. |
| `row_to_data_block` | Concrete | RowToDataBlock operator. |
| `scalar` | Concrete | Scalar-kind PlanNode such as Reference, Function, Constant, or Parameter. These are expression nodes, not relational operators, so they never fall back to `unknown`. |
| `scalar_subquery` | Concrete | Scalar Subquery operator. |
| `search_predicate` | Concrete | Full Text Search predicate operator reached from a `Search Predicate` child link. |
| `search_query_conversion_tvf` | Concrete | Full Text Search query-conversion TVF operator, displayed as `TVF` with metadata `name=Search Query Conversion`. |
| `scan` | Concrete | Scan, Table Scan, or Index Scan operator. |
| `semi_apply` | Concrete | Standalone Semi Apply operator. |
| `serialize_result` | Concrete | Serialize Result operator. |
| `spool_build` | Concrete | SpoolBuild operator. |
| `spool_scan` | Concrete | SpoolScan operator. |
| `stream_aggregate` | Concrete | Aggregate classified as stream aggregation from metadata or display/description text. |
| `union_all` | Concrete | Union All operator. |
| `union_input` | Concrete | Union Input operator. |
| `unit_relation` | Concrete | Unit Relation operator. |
| `verify_determinism` | Concrete | Full Text Search determinism check operator displayed as `VerifyDeterminism`. |
| `unknown` | Concrete fallback | Relational PlanNode that did not match any known normalization rule. Scalar-kind PlanNodes map to `scalar` instead. Use this in strict contracts when unknown operators should fail review. |

```yaml
contracts:
- name: NoExplicitSort
  target: query/ScanSingerIDsFast
  forbid:
  - operator_family: explicit_sort
    max_count: 0
```

## Predefined Contracts

The v1alpha predefined set is intentionally small:

| Name | Equivalent forbidden families | Intent |
| --- | --- | --- |
| `no_explicit_sort` | `explicit_sort` | Reject both full and minor explicit sort operators. |
| `no_full_sort` | `full_sort` | Reject full `Sort` / `Sort Limit` operators while allowing minor sorts. |
| `no_minor_sort` | `minor_sort` | Reject `Minor Sort` / `Minor Sort Limit` operators. |
| `no_hash_join` | `hash_join`, `push_broadcast_hash_join` | Reject standalone hash joins and Push Broadcast Hash Join wrappers. |
| `no_standalone_hash_join` | `hash_join` | Reject only standalone hash joins. |
| `no_push_broadcast_hash_join` | `push_broadcast_hash_join` | Reject only Push Broadcast Hash Join wrappers. |
| `no_apply_join` | `apply_join`, `semi_apply`, `anti_semi_apply`, `distributed_cross_apply`, `distributed_semi_apply`, `distributed_anti_semi_apply` | Reject standalone apply-family operators and distributed apply-family wrappers. |
| `no_standalone_apply_join` | `apply_join`, `semi_apply`, `anti_semi_apply` | Reject only standalone apply-family operators. |
| `no_distributed_cross_apply` | `distributed_cross_apply` | Reject Distributed Cross/Outer Apply wrappers. |
| `no_merge_join` | `merge_join` | Reject Merge Join operators. |
| `no_hash_aggregate` | `hash_aggregate` | Reject hash aggregate operators. |
| `no_stream_aggregate` | `stream_aggregate` | Reject stream aggregate operators. |
| `no_full_scan` | metadata rule | Reject scan operators whose metadata says `Full scan: true`. |
| `no_full_scan_without_timestamp_condition` | metadata plus child-link rule | Reject full scan operators unless they have a `Timestamp Condition` child link. |
| `require_timestamp_condition` | child-link rule | Require at least one scan operator with a `Timestamp Condition` child link. |
| `no_blocking_operator_under_limit` | topology rule | Reject stream-blocking descendants below `Limit` or `Sort Limit`. `Minor Sort Limit` is not treated as blocking by this rule. |

The table above is the public predefined vocabulary. Conceptually, each
operator-family predefined contract expands to direct `forbid.operator_family`
rules like the following. `no_full_scan`,
`no_full_scan_without_timestamp_condition`, `require_timestamp_condition`, and
`no_blocking_operator_under_limit` are different: `no_full_scan` reads
normalized scan metadata, `no_full_scan_without_timestamp_condition` combines
scan metadata with child links, `require_timestamp_condition` reads normalized
child links, and `no_blocking_operator_under_limit` uses a topology-aware rule
because it depends on ancestor/descendant relationships, not only whole-plan
counts.

```yaml
contracts:
- name: NoExplicitSort
  target: query/Target
  forbid:
  - operator_family: explicit_sort

- name: NoFullSort
  target: query/Target
  forbid:
  - operator_family: full_sort

- name: NoMinorSort
  target: query/Target
  forbid:
  - operator_family: minor_sort

- name: NoHashJoin
  target: query/Target
  forbid:
  - operator_family: hash_join
  - operator_family: push_broadcast_hash_join

- name: NoStandaloneHashJoin
  target: query/Target
  forbid:
  - operator_family: hash_join

- name: NoPushBroadcastHashJoin
  target: query/Target
  forbid:
  - operator_family: push_broadcast_hash_join

- name: NoApplyJoin
  target: query/Target
  forbid:
  - operator_family: apply_join
  - operator_family: semi_apply
  - operator_family: anti_semi_apply
  - operator_family: distributed_cross_apply
  - operator_family: distributed_semi_apply
  - operator_family: distributed_anti_semi_apply

- name: NoStandaloneApplyJoin
  target: query/Target
  forbid:
  - operator_family: apply_join
  - operator_family: semi_apply
  - operator_family: anti_semi_apply

- name: NoDistributedCrossApply
  target: query/Target
  forbid:
  - operator_family: distributed_cross_apply

- name: NoMergeJoin
  target: query/Target
  forbid:
  - operator_family: merge_join

- name: NoHashAggregate
  target: query/Target
  forbid:
  - operator_family: hash_aggregate

- name: NoStreamAggregate
  target: query/Target
  forbid:
  - operator_family: stream_aggregate

- name: NoFullScan
  target: query/Target
  use:
  - no_full_scan

- name: NoFullScanWithoutTimestampCondition
  target: query/Target
  use:
  - no_full_scan_without_timestamp_condition

- name: RequireTimestampCondition
  target: query/Target
  use:
  - require_timestamp_condition

- name: NoBlockingOperatorUnderLimit
  target: query/Target
  use:
  - no_blocking_operator_under_limit
```

The same pass/fail intent can also be written with CEL over
`operator_family_counts`. These CEL examples are useful when composing a custom
condition, but they are not byte-for-byte equivalent to predefined/direct
`forbid` results: CEL rules do not emit `matched_operator_indexes`,
predefined `source` / `predefined` fields, built-in remediation, or the special
aggregate/join `classification_unknown` failure used by aggregate- and
join-family `forbid` rules.

CEL contracts are evaluated literally. They do not automatically apply the
`classification_unknown` safeguards used by predefined and direct `forbid`
rules. If a CEL contract forbids a specific join or aggregate family and needs
the same conservative behavior, also check `operator_family_counts["join"]` or
`operator_family_counts["aggregate"]`, or use predefined/direct `forbid` rules.
When a CEL rule fails, the report uses `failure_kind: violation` and does not
emit `diagnostic_id`; `classification_unknown` is reserved for
predefined/direct `forbid_operator_family` rules.

```yaml
contracts:
- name: NoExplicitSortCEL
  target: query/Target
  cel: operator_family_counts["explicit_sort"] == 0

- name: NoFullSortCEL
  target: query/Target
  cel: operator_family_counts["full_sort"] == 0

- name: NoMinorSortCEL
  target: query/Target
  cel: operator_family_counts["minor_sort"] == 0

- name: NoHashJoinCEL
  target: query/Target
  cel: |
    operator_family_counts["hash_join"] == 0 &&
    operator_family_counts["push_broadcast_hash_join"] == 0

- name: NoStandaloneHashJoinCEL
  target: query/Target
  cel: operator_family_counts["hash_join"] == 0

- name: NoPushBroadcastHashJoinCEL
  target: query/Target
  cel: operator_family_counts["push_broadcast_hash_join"] == 0

- name: NoApplyJoinCEL
  target: query/Target
  cel: |
    operator_family_counts["apply_join"] == 0 &&
    operator_family_counts["semi_apply"] == 0 &&
    operator_family_counts["anti_semi_apply"] == 0 &&
    operator_family_counts["distributed_cross_apply"] == 0 &&
    operator_family_counts["distributed_semi_apply"] == 0 &&
    operator_family_counts["distributed_anti_semi_apply"] == 0

- name: NoStandaloneApplyJoinCEL
  target: query/Target
  cel: |
    operator_family_counts["apply_join"] == 0 &&
    operator_family_counts["semi_apply"] == 0 &&
    operator_family_counts["anti_semi_apply"] == 0

- name: NoDistributedCrossApplyCEL
  target: query/Target
  cel: operator_family_counts["distributed_cross_apply"] == 0

- name: NoMergeJoinCEL
  target: query/Target
  cel: operator_family_counts["merge_join"] == 0

- name: NoHashAggregateCEL
  target: query/Target
  cel: operator_family_counts["hash_aggregate"] == 0

- name: NoStreamAggregateCEL
  target: query/Target
  cel: operator_family_counts["stream_aggregate"] == 0

- name: NoFullScanCEL
  target: query/Target
  cel: operators.all(o, !(o.family == "scan" && o.full_scan))

- name: NoFullScanWithoutTimestampConditionCEL
  target: query/Target
  cel: |
    operators.all(o,
      !(o.family == "scan" && o.full_scan) ||
      operator_edges.exists(e,
        e.parent_index == o.index &&
        e.type == "Timestamp Condition"))

- name: RequireTimestampConditionCEL
  target: query/Target
  cel: operator_edges.exists(e, e.type == "Timestamp Condition")

- name: NoBlockingOperatorUnderLimitCEL
  target: query/Target
  cel: |
    operators.all(limit,
      !(limit.family == "limit" || limit.display_name.endsWith("Sort Limit")) ||
      operators.all(descendant,
        !limit.descendant_indexes.exists(index, index == descendant.index) ||
        descendant.family != "full_sort" &&
        descendant.family != "hash_aggregate" &&
        descendant.family != "hash_join" &&
        descendant.family != "push_broadcast_hash_join" &&
        descendant.family != "aggregate" &&
        descendant.family != "join" &&
        descendant.family != "bloom_filter_build" &&
        descendant.family != "spool_build"))
```

`no_hash_join` does not count the internal implementation Hash Join reached
through a non-branching relational path from the Push Broadcast `Map` child and
consuming a pushed batch scan; that node is reported separately as
`push_broadcast_hash_join_internal_hash_join`.

`no_apply_join` similarly ignores internal apply implementation nodes under
distributed wrappers, which are reported separately as
`distributed_cross_apply_internal_apply` or
`distributed_semi_apply_internal_apply` /
`distributed_anti_semi_apply_internal_apply`.

`Push Broadcast Hash Join Semi Apply` and `Push Broadcast Hash Join Anti Semi
Apply` are normalized as the `push_broadcast_hash_join` wrapper family. They
are therefore rejected by `no_hash_join` or `no_push_broadcast_hash_join`, not
by `no_apply_join`.

Join-specific contracts fail conservatively when the plan contains generic
fallback `join` nodes. In that case the evaluator cannot prove the node is not
the specific join family being forbidden, so it emits
`failure_kind: classification_unknown` with
`diagnostic_id: plan.join_classification_unknown` and the ambiguous join node
indexes in `matched_operator_indexes`. If a contract intentionally wants to
reject only fallback join nodes, use direct `forbid.operator_family: join`.

There are no dedicated `no_distributed_semi_apply` or
`no_distributed_anti_semi_apply` predefined contracts in v1alpha. Use direct
`forbid.operator_family` rules for those narrower policies:

```yaml
contracts:
- name: NoDistributedSemiApplyOnly
  target: query/Foo
  forbid:
  - operator_family: distributed_semi_apply
  - operator_family: distributed_anti_semi_apply
```

Each expanded predefined rule result records its origin in the report. For
example, `use: [no_hash_join]` expands into `forbid_operator_family` results
with `source: use/no_hash_join` and `predefined: no_hash_join`. Direct
`forbid` entries use source strings such as `forbid[0]`, where `0` is the
0-based YAML order index in that contract's original `forbid` array.

When `source` has the form `use/<name>`, `predefined` is present and equals
`<name>`. That equality is a runtime invariant tested by the implementation;
the schema restricts the value domains. When `source` is `forbid[n]` or `cel`,
`predefined` is absent.
The plan-report schema also enforces the rule/source kind coupling:
`rule: cel` requires `source: cel`, while `rule: forbid_operator_family`
requires `source: use/<operator-family-predefined-name>` or
`source: forbid[n]`, and `rule: forbid_blocking_operator_under_limit` requires
`source: use/no_blocking_operator_under_limit`. `rule: forbid_full_scan`
requires `source: use/no_full_scan`.
`rule: forbid_full_scan_without_timestamp_condition` requires
`source: use/no_full_scan_without_timestamp_condition`.
`rule: require_timestamp_condition` requires
`source: use/require_timestamp_condition`.
Direct `forbid` entries must not repeat the same `operator_family` within one
contract. Runtime validation rejects duplicates with
`plan_contract.duplicate_forbid_operator_family` rather than merging ambiguous
rules.
Contract names must also be unique within one contract file; duplicate names
are rejected with `plan_contract.duplicate_contract_name`.

For `forbid_operator_family`, `forbid_full_scan`,
`forbid_full_scan_without_timestamp_condition`,
`require_timestamp_condition`, and `forbid_blocking_operator_under_limit`
rules, `matched_operator_indexes` is always present: it is `[]` when
`observed_count == 0`, otherwise it contains every matching
`normalized_operators[].index` in ascending order. CEL rules do not emit
`matched_operator_indexes` in v1alpha.

For concrete families, matching means `operator.family == operator_family`.
For derived umbrella families, `matched_operator_indexes` contains indexes of
the concrete operators that contribute to the derived count. For
`explicit_sort`, that means `full_sort` and `minor_sort` operator indexes.
For `blocking_operator`, that means stream-blocking concrete operator indexes.

For `forbid_blocking_operator_under_limit` rules,
`matched_operator_indexes` contains blocking descendant operator indexes, not
the `Limit` or `Sort Limit` ancestor indexes. This is useful for queries where
a limiting operator is expected to stream early rows but a descendant full sort,
hash aggregate, hash join, BloomFilterBuild, or SpoolBuild can force upstream
materialization first.

For `forbid_full_scan` rules, `matched_operator_indexes` contains scan
operators whose normalized metadata has `full_scan: true`. This rule is a
PLAN-level early warning only: it does not prove how many rows are scanned, and
a non-full scan can still read too many rows if important predicates remain as
residual conditions.

For `forbid_full_scan_without_timestamp_condition` rules,
`matched_operator_indexes` contains full-scan operator indexes that do not have
a `Timestamp Condition` child link. This is the preferred broad OLTP guardrail
when recent-data commit timestamp reads are allowed to rely on storage-level
timestamp pruning.

For `require_timestamp_condition` rules, `matched_operator_indexes` contains
the parent scan operator indexes that have a `Timestamp Condition` child link.
This rule is for recent-data commit timestamp reads where storage-level
timestamp pruning is expected. It intentionally does not require the absence of
`full_scan: true`: current Spanner plans can still report a full table scan
while a `Timestamp Condition` reduces I/O below the scan operator.

The plan-contract and plan-report `operator_family` schema enums are generated
from the same normalizer registry used by `plan-report`; the two schema enums
are expected to match exactly.
The plan-report schema additionally exposes `concrete_operator_family` for
fields that cannot contain derived umbrella families, namely
`normalized_operators[].family` and `operator_families[]`.

## Unknown Classification

`unknown` is reserved for relational plan nodes that the normalizer cannot
classify into a known operator family. Scalar-kind plan nodes such as
Reference, Function, Constant, and Parameter are expression nodes, not
relational operators; they always classify as `scalar` and never as `unknown`.
Unknown operators appear in `operator_families`,
`operator_family_counts`, and `normalized_operators` like any other family, so
direct `forbid` contracts can target `operator_family: unknown` when a strict
"no unclassified operator" policy is useful.

Predefined contracts do not fail merely because an unrelated `unknown` family
is present. They fail on unknown classification only when the specific rule
depends on that classification. The same protection applies to direct
`forbid.operator_family: hash_aggregate` and
`forbid.operator_family: stream_aggregate` rules: aggregate-family contracts
fail with `failure_kind: classification_unknown` and
`diagnostic_id: plan.aggregate_classification_unknown` when an `Aggregate` node
has no decisive `iterator_type` metadata. In that case
`matched_operator_indexes` contains the ambiguous `Aggregate` node indexes.
Join-specific predefined and direct contracts use the same conservative
classification-unknown policy for fallback `join` nodes and report
`diagnostic_id: plan.join_classification_unknown`.

Strict CI can reject unknown operators with a direct `forbid` rule:

```yaml
contracts:
- name: NoUnknownOperators
  target: query/ImportantQuery
  forbid:
  - operator_family: unknown
    max_count: 0
```

## Status Invariants

`plan-report` validates these invariants before writing an artifact. A
violation is treated as a generator bug rather than a consumer responsibility.

- Top-level `status` reports whether a plan-report artifact was produced. It
  does not mean every query target was successfully planned.
- Use `target_summary.planned`, `target_summary.errors`,
  `target_summary.skipped`, and `queries[].status` for target-level success.
- `target_summary.included_count == target_summary.planned +
  target_summary.errors + target_summary.skipped`.
- `target_summary.excluded` is always present. It is `[]` when no target was
  excluded from the report target set. Skipped targets are included in
  `queries[]`, counted in `target_summary.skipped`, and not duplicated in
  `target_summary.excluded`.
- `queries[].status: ok` means plan fields such as `operator_tree_sha256`,
  `operator_family_counts`, `normalized_operators`, `operator_edges`, and
  `plan` are valid. `operator_edges` is always present for `ok` targets and is
  `[]` when the normalized edge list is empty.
- For each `queries[].status: ok` target, `normalized_operators[].index` is
  unique; every `operator_edges[].parent_index`,
  `operator_edges[].child_index`, `operators[].child_indexes[]`, and
  `operators[].descendant_indexes[]` entry refers to an existing
  `normalized_operators[].index`; every
  `operators[].subtree_family_counts` matches that operator's normalized
  subtree; and `queries[].operator_family_counts` matches the whole normalized
  operator list.
- `queries[].status: error | skipped` means only partial input/error fields are
  reliable. Target identity and `error` are reliable; for `skipped`, `error` is
  a skip message rather than necessarily a failure. `sql`, `sql_sha256`, and
  `ddl_sha256` are reliable only when present. Plan normalization fields such as
  `operator_tree_sha256`, `operator_families`, `operator_family_counts`,
  `normalized_operators`, `operator_edges`, `plan`, and
  `classification_warnings` are absent and must not be interpreted as successful
  plan evidence.
- A contract evaluation is `pass` iff every rule result is `pass`.
- A contract evaluation is `fail` iff at least one rule result is `fail`.
- A contract evaluation is `not_evaluated` when the target is missing or could
  not be analyzed, and then it has no `results`.
- `contract_summary.status` is `pass` iff every evaluation is `pass`.
- `contract_summary.status` is `fail` if any evaluation is `fail` or
  `not_evaluated`.
- `contract_summary.contracts == len(contract_evaluations)`.
- `contract_summary.passed == count(contract_evaluations[].status == pass)`.
- `contract_summary.failed == count(contract_evaluations[].status == fail)`.
- `contract_summary.not_evaluated == count(contract_evaluations[].status == not_evaluated)`.
- `contract_summary.environment_warnings` is always present. It is `[]` when
  no warning applies.
- Top-level `warnings` is also always present. It is `[]` when the artifact has
  no general production warnings.
- Top-level `warnings` records artifact-wide production and completeness
  warnings. `contract_summary.environment_warnings` records warnings that
  affect contract interpretation under the selected backend, optimizer, and
  contract mode.
- `contract_rule_result.matched_operator_indexes`, when present, refers only to
  indexes in the target's `normalized_operators`.

`target_summary.excluded[].reason` uses the same namespaced style as diagnostic
IDs. Current v1alpha plan-report keeps unavailable configured targets in
`queries[]` as `status: skipped`; therefore `excluded` is normally `[]`.
Reserved examples for future target-set pruning include:

- `target.filtered_by_selector`
- `target.non_spanner_catalog`
- `target.missing_catalog`
- `target.external_dataset`

An abridged no-target excerpt has no query entries and no contract evaluation:

```yaml
status: no_targets
warnings: []
target_summary:
  included_count: 0
  planned: 0
  errors: 0
  skipped: 0
  excluded: []
queries: []
contract_evaluation_mode: none
```

If `--contracts` is supplied and a contract targets a missing, error, or
skipped target, top-level artifact production can still be `ok` or
`no_targets`; the contract entry itself is `status: not_evaluated` and
`contract_summary.status` is `fail`.

## Minimal Report Outcomes

The following snippets are contract-related excerpts from a full
`plan-report` artifact. They omit unrelated top-level report fields, but each
`contract_evaluations` entry includes the fields required by the plan-report
schema for the shown status.

Pass example:

```yaml
contract_evaluation_mode: check
contract_summary:
  status: pass
  contracts: 1
  passed: 1
  failed: 0
  not_evaluated: 0
  environment_warnings: []
contract_evaluations:
- name: NoExplicitSort
  query: ScanSingerIDsFast
  target_id: query/ScanSingerIDsFast
  scope: query
  status: pass
  stability:
    tier: normalized_operator
    reasons:
    - contract uses the normalized plan-report view
    check_recommended: true
    replayable_from_report: true
  results:
  - rule: forbid_operator_family
    source: use/no_explicit_sort
    predefined: no_explicit_sort
    operator_family: explicit_sort
    status: pass
    observed_count: 0
    max_count: 0
    matched_operator_indexes: []
```

`report_only` fail example. The report records `status: fail`, but the process
exit code remains 0 because `--check` was not requested:

```yaml
contract_evaluation_mode: report_only
contract_summary:
  status: fail
  contracts: 1
  passed: 0
  failed: 1
  not_evaluated: 0
  environment_warnings: []
contract_evaluations:
- name: NoHashJoin
  query: ListAlbums
  target_id: query/ListAlbums
  scope: query
  status: fail
  stability:
    tier: normalized_operator
    reasons:
    - contract uses the normalized plan-report view
    check_recommended: true
    replayable_from_report: true
  results:
  - rule: forbid_operator_family
    source: use/no_hash_join
    predefined: no_hash_join
    operator_family: hash_join
    status: fail
    failure_kind: violation
    observed_count: 1
    max_count: 0
    matched_operator_indexes:
    - 3
```

Unavailable target example:

```yaml
contract_evaluation_mode: check
contract_summary:
  status: fail
  contracts: 1
  passed: 0
  failed: 0
  not_evaluated: 1
  environment_warnings: []
contract_evaluations:
- name: MissingTarget
  target_id: query/MissingTarget
  status: not_evaluated
  reason: target_not_found
  stability:
    tier: normalized_operator
    reasons:
    - contract uses the normalized plan-report view
    check_recommended: true
    replayable_from_report: true
```

## CEL Inputs

CEL contracts are useful for query-specific checks that are too narrow or too
experimental for predefined names. Prefer normalized inputs for CI. Raw plan
inputs remain available for exploratory checks, but are marked with
`stability.tier: raw_query_plan` and `check_recommended: false`.

Current CEL variables:

- `backend`
- `query`
- `operators`
- `operator_edges`
- `operator_families`
- `operator_family_counts`
- `optimizer`
- `raw_plan`
- `raw_nodes`

`raw_plan` is the in-memory `spannerpb.QueryPlan` supplied to the evaluator;
`raw_nodes` is `raw_plan.plan_nodes`. These are evaluator inputs, not
serialized report fields. A CEL contract that reads either variable is marked
`stability.tier: raw_query_plan`, `check_recommended: false`, and
`replayable_from_report: false`, because it cannot be replayed from the
YAML/JSON report alone unless the evaluator is also supplied with the original
raw QueryPlan.

`operator_family_counts` is a complete map over every plan-report
`operator_family` enum value, including derived umbrella families such as
`explicit_sort`. Families absent from the plan are represented as `0`, never as
a missing map key. `operator_families` remains the compact list of concrete
families that were actually observed in `normalized_operators[].family`.

Each `operators[]` entry includes stable topology fields:
`child_indexes`, `descendant_indexes`, and `subtree_family_counts`.
`subtree_family_counts` is also a complete map over every `operator_family`
enum value, so CEL may use direct indexing such as
`o.subtree_family_counts["scan"]`.

CEL evaluation is performed against the tool's normalized CEL input, not by
running CEL directly over the serialized report YAML/JSON. Optional string
fields in both `operators[]` and `operator_edges[]` default to `""` in CEL;
optional boolean fields default to `false`. External evaluators that replay CEL
from the report must apply the same defaults. The report records this contract
under `normalization.cel_input_defaults`.
Stability classification for CEL expressions is based on parsed CEL identifiers
and select fields; mentioning `raw_nodes`, `execution_stats`, or a metadata
field inside a string literal does not by itself change the stability tier.

For `operators[]`, optional string metadata includes `execution_method`,
`iterator_type`, `scan_method`, `scan_format`, `scan_type`, `scan_target`,
`seekable_key_size`, `join_type`, `join_configuration`, `call_type`,
`distribution_table`, `subquery_cluster_node`, and `spool_name`. Optional
boolean metadata fields such as `full_scan` default to `false`. For
`operator_edges[]`, optional string fields include `type` and `variable`.

`scan_type` uses normalized spelling such as `table_scan`, `index_scan`,
`batch_scan`, and `search_index_scan`, not raw PlanNode metadata values such as
`TableScan` or `IndexScan`.

`execution_stats` is explicitly rejected.

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
operator_edges.exists(e, e.parent_index == 0 && e.type == "Input")
```

```cel
operators.exists(o,
  o.family == "hash_aggregate" &&
  o.subtree_family_counts["scan"] == 1)
```

Approximate root-partitionable PLAN-shape review can also be written as a
structural CEL contract. This mirrors the documented PartitionQuery rule at the
PLAN shape level, but it is not a `PartitionQuery` API acceptance probe. The
normalized operator view exposes `subquery_cluster_node` metadata. Observed
Spanner Omni plans use that metadata on distributed plan fragments; Local
Distributed Unions also have it, but they are excluded by `call_type: local`.
Using that metadata keeps the rule resilient if Spanner adds new distributed
operator names later.

`subquery_cluster_node` is raw PlanNode metadata copied into the plan-report
normalized operator view. It is intentionally hidden by human-readable
`spannerplan` renderers such as `reference`, so this contract cannot be audited
from the rendered plan alone. Use YAML/JSON plan-report output when reviewing
the metadata that drives the CEL expression.

```yaml
contracts:
- name: ApproximateRootPartitionablePlanShape
  target: query/ExportSingers
  cel: |
    operator_family_counts["unknown"] == 0 &&
    (
      (
        operators.exists(o,
          o.index == 0 &&
          o.family == "distributed_union" &&
          o.subquery_cluster_node != "" &&
          o.call_type != "local") &&
        operators.filter(o,
          o.subquery_cluster_node != "" &&
          o.call_type != "local").size() == 1
      ) ||
      (
        operators.filter(o,
          o.subquery_cluster_node != "" &&
          o.call_type != "local").size() == 0
      )
    )
```

`distributed_merge_union` is intentionally rejected here. It includes the
documented Distributed Merge Union behavior that appears as `Distributed Union`
with `preserve_subquery_order: true` in raw plans. The root node still has
`subquery_cluster_node`, but its normalized family is `distributed_merge_union`
rather than `distributed_union`, so the first branch rejects it.

Because this CEL reads metadata-derived normalized fields, the report keeps the
stability tier as `normalized_operator` but records the dependency in
`stability.reasons`:

```yaml
stability:
  tier: normalized_operator
  reasons:
  - contract uses the normalized plan-report view
  - "contract reads metadata-derived normalized fields: call_type, subquery_cluster_node"
  check_recommended: true
  replayable_from_report: true
```

This is still a PLAN-shape review contract, not an authoritative
`PartitionQuery` probe. The official Spanner docs note that `PartitionQuery`
runs queries in batch mode and can choose a different plan from the plan shown
by normal analysis tools. If exact API acceptance matters, add a separate
backend probe that calls `PartitionQuery` and checks whether partition creation
succeeds.

The example rejects `operator_family_counts["unknown"] > 0` so future
distributed operators that are not yet normalized cannot silently pass through
the zero-distributed-fragment branch. If a looser exploratory check is desired,
remove that guard and review `classification_warnings` together with the
result.

The normalized topology fields are intentionally shallow data, not CEL macros:

- `operator_edges`: parent/child child-link records for the whole target.
- `operators[].child_indexes`: direct child PlanNode indexes.
- `operators[].descendant_indexes`: transitive child PlanNode indexes.
- `operators[].subtree_family_counts`: operator-family counts for this
  PlanNode subtree, including the operator itself.

This keeps user CEL simple without requiring recursive CEL expressions or
custom predicate macros.

## Optimizer Fields

The report separates requested and effective optimizer settings:

- `optimizer.requested.version`
- `optimizer.requested.statistics_package`
- `optimizer.effective.version`
- `optimizer.effective.statistics_package`

`--optimizer-version` and `--optimizer-statistics-package` populate requested
settings passed to `AnalyzeQuery`. `--require-optimizer-pinning` checks the
requested pinning state. v1alpha records effective optimizer values as
`not_recorded`.
This check does not infer optimizer pinning from SQL statement hints such as
`@{OPTIMIZER_VERSION=...}` or `@{OPTIMIZER_STATISTICS_PACKAGE=...}`. If a query
relies on statement hints for pinning, review the rendered SQL and the report
environment together.

Sort-related plan contracts can also be sensitive to statement hints such as
`ALLOW_DISTRIBUTED_MERGE`. v1alpha does not model that hint as optimizer
pinning; when a contract depends on it, make the hint visible in the rendered
SQL and review the SQL digest / optimizer matrix evidence together.

For example, a query using a sort contract might intentionally include:

```sql
@{ALLOW_DISTRIBUTED_MERGE=FALSE}
SELECT SingerId, SongGenre FROM Songs ORDER BY SongGenre
```

`--require-optimizer-pinning` still only checks requested optimizer version and
statistics package; it does not treat statement-level plan-shape hints as
optimizer pinning.
