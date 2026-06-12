# plan-report Operator Coverage Sweep (2026-06-12)

Non-normative observations from running `spanner-query-gen plan-report`
against Spanner Omni (`spanemuboost` default image,
`us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`) with a
17-query config designed to exercise diverse operators: hinted joins
(`HASH_JOIN`, `APPLY_JOIN`, `MERGE_JOIN`, `PUSH_BROADCAST_HASH_JOIN`),
hinted aggregates (`HASH_GROUP`, `STREAM_GROUP`), sort+limit, `UNION ALL`,
`UNNEST`, scalar/array subqueries, `EXISTS` / `NOT EXISTS`, `DISTINCT`,
`WITH`, timestamp range predicates, and interleaved multi-table joins.

This sweep was run after scalar-kind PlanNodes were reclassified into the
`scalar` family and the invariant validator was fixed for
`blocking_operator` subtrees.

## Classification coverage

- All 17 plans produced `operator_family_counts["unknown"] == 0` and zero
  `classification_warnings`. Every relational PlanNode observed on this Omni
  build mapped to a concrete family.
- Contextual classification worked on live plans:
  `PUSH_BROADCAST_HASH_JOIN` produced both `push_broadcast_hash_join` and
  `push_broadcast_hash_join_internal_hash_join`; `EXISTS` / `NOT EXISTS`
  produced `distributed_semi_apply` / `distributed_anti_semi_apply` plus
  their `_internal_apply` families.
- A strict `forbid: operator_family: unknown, max_count: 0` contract now
  passes on real plans. Before the scalar reclassification it failed on every
  plan because scalar expression nodes (Reference, Function, Constant,
  Parameter) all fell into `unknown`.
- Follow-up (same day): the first scalar reclassification checked
  `PlanNode.Kind` before display-name mapping, which shadowed concrete
  families carried by scalar-kind nodes. `Array Subquery` is kind `SCALAR`
  on this Omni build (verified from raw `nodes` output), so the `ARRAY(...)`
  sweep query stopped reporting `array_subquery`. The classifier now applies
  display-name families first and uses `scalar` only as the fallback for
  otherwise-unclassified scalar-kind nodes; the sweep query reports
  `array_subquery` again. The decorrelated `(SELECT COUNT(*) ...)` query
  never contains a `Scalar Subquery` node (it compiles to
  `apply_join` + `stream_aggregate`), so `scalar_subquery` remains unobserved
  in this sweep.
- Families not observed by this sweep (future probe candidates):
  `minor_sort`, standalone `semi_apply` / `anti_semi_apply`,
  `bloom_filter_build` (observed separately in the `hash_join` plan-shape
  case), `scalar_subquery`, `change_stream_tvf`, `search_*` (needs search
  indexes), `recursive_*` (needs recursive CTE), `spool_*`, `mini_batch_*`,
  `random_id_assign`, `apply_mutations`, `row_count` (DML targets are
  excluded from plan-report), `empty_relation`, `unit_relation`,
  `verify_determinism`, and the generic `join` / `aggregate` fallbacks.

## Join elimination on interleaved tables silently ignores JOIN_METHOD hints

With `Albums` interleaved in `Singers`, the query

```sql
SELECT s.SingerId, a.Title
FROM Singers AS s JOIN@{JOIN_METHOD=HASH_JOIN} Albums AS a
  ON s.SingerId = a.SingerId
```

compiled to a single `Table Scan on Albums` with no join operator at all:
parent-row existence is guaranteed by interleaving and only the join key is
projected from the parent, so the optimizer eliminated the join and the
`JOIN_METHOD` hint had nothing to bind to (no error, no warning). The same
happened for a three-table interleaved `Singers JOIN Albums JOIN Songs`
projecting only keys. Projecting a non-key parent column
(`s.FirstName`) restored real join operators for every join method.

Implication for plan contracts: a contract asserting a specific join family
can fail (or trivially pass) because the join was eliminated, not because the
hint changed. Operator-presence contracts such as `no_full_scan` or explicit
family counts describe the actual executed shape and are more robust than
assumptions about hint effects.

Follow-up (same day, plan-shape probe over schema variants): the elimination
is driven by declared referential integrity, not by interleaving
specifically.

- `INTERLEAVE IN PARENT`: join eliminated (single `Table Scan` on the child).
- Plain table with no constraint: join survives as
  `Distributed Cross Apply`.
- Enforced `FOREIGN KEY`: join eliminated.
- `FOREIGN KEY ... NOT ENFORCED` (informational): join **also** eliminated —
  the optimizer trusts the declared constraint, matching the documented
  caveat that nonconforming data under informational constraints can change
  query results.
- The eliminated interleaved shape is stable across optimizer versions 1-8
  (`--optimizer-version-diff` reported no shape change).

## Optimizer version changes plans for 7 of 17 queries

Comparing `--optimizer-version 1` against `--optimizer-version 8`
(operator tree digests): `ApplyJoinSingersAlbums`, `PushBroadcastHashJoin`,
`SortLimit`, `SemiJoinExists`, `AntiSemiJoinNotExists`, `WithClause`, and
`InterleavedJoin` produced different operator trees; the other 10 were
identical. `--require-optimizer-pinning` failed with
`optimizer_not_pinned, statistics_package_not_pinned,
query_optimizer_not_pinned` when no pin flags were supplied, and recorded
`optimizer.requested` while `optimizer.effective` stayed `not_recorded` as
documented.

## Limitations

- All plans were taken with `AnalyzeQuery` against empty tables and default
  statistics. Unhinted operator choices (join method, distribution) may
  differ under real data volumes and refreshed optimizer statistics; hinted
  shapes and the classification results themselves are not data-dependent in
  the observed cases.
- Backend identity is `not_recorded` (spanemuboost does not yet expose the
  resolved image digest), so the image tag above is the spanemuboost default
  at the time of the run, not observed evidence.
- The sweep config and schema are not checked in; the query list in the
  header is the authoritative description. Re-running requires only the
  17 queries described there against the three-table interleaved
  Singers/Albums/Songs schema plus a non-interleaved Concerts table.

## Contract failure-path behavior on live plans

Intentional violations produced the expected results with `--check`
(exit code 1): `no_full_scan` failed with `matched_operator_indexes` and
index-design/query-shape remediations; `no_explicit_sort` failed on the
`ORDER BY ... LIMIT` plan; a direct `forbid: operator_family: hash_join`
failed with the JOIN_METHOD hint remediation; `require_timestamp_condition`
failed for a non-commit-timestamp `TIMESTAMP` predicate with an explanatory
remediation; a CEL contract reading `operator_family_counts["scalar"]`
evaluated as replayable_from_report=true.
