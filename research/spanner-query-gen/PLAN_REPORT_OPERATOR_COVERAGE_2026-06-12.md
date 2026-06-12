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

## Contract failure-path behavior on live plans

Intentional violations produced the expected results with `--check`
(exit code 1): `no_full_scan` failed with `matched_operator_indexes` and
index-design/query-shape remediations; `no_explicit_sort` failed on the
`ORDER BY ... LIMIT` plan; a direct `forbid: operator_family: hash_join`
failed with the JOIN_METHOD hint remediation; `require_timestamp_condition`
failed for a non-commit-timestamp `TIMESTAMP` predicate with an explanatory
remediation; a CEL contract reading `operator_family_counts["scalar"]`
evaluated as replayable_from_report=true.
