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
  `apply_join` + `stream_aggregate`).
- Follow-up (same day): `Scalar Subquery` nodes are also kind `SCALAR` (see
  [spanner-hacks operators.md](https://github.com/apstndb/spanner-hacks/blob/master/content/ja/operators.md):
  the only scalar operators rendered in standard plan trees are the two
  subquery operators, Array Subquery and Scalar Subquery). The conditional
  pattern from that document,
  `IF(FirstName = 'Alice', (SELECT COUNT(*) FROM Songs WHERE Duration > 300), 0)`,
  prevents decorrelation; running it through plan-report on Omni observed
  `scalar_subquery` with `unknown == 0` and no classification warnings,
  confirming the display-name-first classifier covers both subquery
  operators.
- Families not observed by this sweep (future probe candidates):
  `minor_sort`, standalone `semi_apply` / `anti_semi_apply`,
  `bloom_filter_build` (observed separately in the `hash_join` plan-shape
  case), `change_stream_tvf`, `search_*` (needs search
  indexes), `recursive_*` (needs recursive CTE), `spool_*`,
  `mini_batch_*` / `row_count` (undocumented SELECT back-join operators seen
  only on Cloud Spanner optimizer v5; see
  `research/spanner-query-plan-shape/OPERATOR_VERIFICATION_FOLLOWUP.md`),
  `random_id_assign`, `apply_mutations` (DML targets are
  excluded from plan-report), `empty_relation`, `unit_relation`,
  `verify_determinism`, and the generic `join` / `aggregate` fallbacks.
  `scalar_subquery` was unobserved by the original 17 queries but was
  confirmed by the conditional-subquery follow-up above.
- Follow-up (same day, via the standalone `plancontract` module fed with raw
  `AnalyzeQuery` plans from Omni — the analyzer-independent path):
  - `change_stream_tvf` observed and classified warning-free from
    `SELECT ChangeRecord FROM READ_EverythingStream(start_timestamp => ...)`.
    This stays outside plan-report because the GoogleSQL frontend catalog
    does not register `READ_<stream>` TVFs.
  - `recursive_union` and `recursive_spool_scan` observed and classified
    warning-free from the spanner-hacks operators.md quantified-path graph
    repro (`MATCH ...-[c:CollabWith]->{1,2}...`). Recursive Union introduces
    `input_0` / `input_1` child-link types not seen elsewhere.
  - `Local Split Union` is confirmed unobservable on Omni 2026.r1-beta:
    `CREATE PLACEMENT` fails with `Unimplemented: Geo-partitioning is not
    supported for this environment`, and the operator is documented to
    appear for placement-table scans.
  - `Generate Relation` still has no known repro;
    `UNNEST(GENERATE_ARRAY(...))` classifies as `array_unnest`. The official
    query-operators-leaf page documents only generic properties for it, with
    no example query.
  - `random_id_assign` observed warning-free via the official
    query-operators-unary repro: `TABLESAMPLE BERNOULLI` pairs it with
    `filter`, and `TABLESAMPLE RESERVOIR` pairs it with `full_sort` +
    `limit`, exactly as documented. Both queries are now pinned by
    `TestIntegrationPlanReportOperatorFamilyCoverageOnOmni`.
  - DML plans verified via `ReadWriteTransaction.AnalyzeQuery` (PLAN mode,
    nothing executed): INSERT VALUES / INSERT ... THEN RETURN / UPDATE /
    DELETE / INSERT SELECT all classify warning-free with `apply_mutations`
    observed in every shape. Correction: an earlier draft implied
    `row_count` and `mini_batch_*` were DML operators; they are not. The
    undocumented `RowCount` / `MiniBatch*` PlanNodes appear in SELECT
    back-join shard-optimization shapes on Cloud Spanner optimizer v5 (see
    `OPERATOR_VERIFICATION_FOLLOWUP.md` and spanner-hacks operators.md), so
    their absence from DML plans says nothing about them. The naming
    collision with querygen's DML result mode `row_count` caused the
    mix-up.
  - `CREATE MODEL` fails on Omni with an empty-message `InvalidArgument`,
    so the documented `ML.PREDICT` TVF plan shape (query-operators-unary
    shows ML.PREDICT compiling to a `TVF` operator) is unobservable on this
    backend. A non-search TVF would currently classify as `unknown`, which
    remains the correct conservative fallback until a real plan is
    available.
- Omni behavior checks against docs.cloud.google.com/spanner-omni/develop:
  `version_retention_period` accepts up to `30d` and rejects `31d` with
  ``range [`1h`, `30d`]`` (DBaaS caps at 7 days);
  `@{ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=TRUE}` still produces a
  `Timestamp Condition` child link even though Omni has no tiered storage;
  and `SEARCH(..., enhance_query => TRUE)` is accepted at both PLAN and
  execution time even though the develop page documents enhanced query mode
  as unsupported on Omni (apparently a silent no-op rather than a
  rejection).

## Family-gap probes (same day): nine more families observed, two bugs found

A follow-up probe pass targeted families the 17-query sweep never produced.
Verified shapes on Omni, now pinned by the
`TestIntegrationPlanReportOperatorFamilyCoverageOnOmni` integration test
(`cmd/spanner-query-gen/omni_family_coverage_test.go`):

- `SELECT 1` → `unit_relation`; `WHERE FALSE` → `empty_relation`.
- `GROUP BY ... HAVING` → `filter` (HAVING is a Filter, not Filter Scan,
  matching spanner-hacks operators.md).
- `ORDER BY SingerId, Title` on `Albums(SingerId, AlbumId)` → `minor_sort`
  (prefix already key-ordered).
- A CTE referenced twice through `UNION ALL` → `spool_build` + `spool_scan`
  plus `union_all` / `union_input`.
- A hinted hash join against a non-interleaved table → `hash_join` with
  `bloom_filter_build` on the build side (the interleave-eliminated joins in
  the original sweep never built bloom filters).
- Colocated `EXISTS` / `NOT EXISTS` against an interleaved child keyed by the
  full parent key → standalone `semi_apply` / `anti_semi_apply` (the
  non-colocated forms in the original sweep were the distributed variants).
- Full-text search (`SEARCH` over a `TOKENLIST` column with a search index)
  → `search_predicate`, `search_query_conversion_tvf`, `verify_determinism`.

Bugs found by these probes (both fixed):

- The catalog rejected `CREATE SEARCH INDEX` (`unsupported DDL
  *ast.CreateSearchIndex`), so full-text search schemas could not flow
  through plan-report at all. Search indexes are now ignored like regular and
  vector indexes for row-type analysis.
- The Search Query Conversion TVF classifier read the lowercase `name`
  metadata key, but Omni emits capitalized `Name` (verified from raw
  PlanNodes), so the TVF always classified `unknown`. The classifier now
  accepts both spellings.

Correction (same day): an earlier draft of this note claimed the unhinted
`GROUP BY ... HAVING` aggregate method "flapped" between Stream and Hash
across runs. That was a schema-difference artifact: one run's schema had
`CREATE INDEX SingersByRating ON Singers(Rating)` and the other did not.
A controlled re-run (3x each) showed the choice is deterministic on this
Omni build: without an index on the GROUP BY key the plan uses Hash
Aggregate, and with `SingersByRating` providing Rating-ordered input it
uses Stream Aggregate, exactly matching the spanner-hacks operators.md
description. Contracts on the aggregate family should still pin
`GROUP@{GROUP_METHOD=...}` when the index design is expected to evolve,
because adding or dropping an index on the grouping key changes the
unhinted choice.

Families still unobservable through plan-report's public config:
`apply_mutations` (DML-only) because `queries[].result.cardinality` accepts
only row-returning modes; the plan-shape `--case dml` probe and
`TestIntegrationDMLOperatorFamilyCoverageOnOmni` cover those shapes.
(`random_id_assign` turned out to be reachable via `TABLESAMPLE`, and
`row_count` / `mini_batch_*` are SELECT back-join operators unobserved on
Omni regardless of statement kind.)
`change_stream_tvf` needs a `READ_<stream>` TVF that the GoogleSQL frontend
catalog does not register, and `recursive_*` shapes still need a reproducible
query (likely graph or quantified-path based).

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
- The `USE_UNENFORCED_FOREIGN_KEY` statement hint (default `TRUE`, overrides
  the `use_unenforced_foreign_key_for_query_optimization` database option per
  statement) controls only the informational-FK part: with
  `@{USE_UNENFORCED_FOREIGN_KEY=FALSE}` the NOT ENFORCED FK join is no longer
  eliminated, while enforced-FK and interleave elimination still happen. The
  hint is statement-scope only; the join-hint position is rejected with
  `Unsupported hint`.
- The corresponding database option could NOT be verified on Omni
  2026.r1-beta. `ALTER DATABASE <db> SET OPTIONS
  (use_unenforced_foreign_key_for_query_optimization = false)` fails with an
  empty-message `InvalidArgument` (the option appears unsupported), and the
  `SET DATABASE OPTIONS (...)` syntax shown in the official docs fails to
  parse (`Encountered 'SET' while parsing: ddl_statement`). The ALTER
  DATABASE mechanism itself works on Omni (`version_retention_period`
  applied successfully via spanemuboost `WithDatabaseID` + a separate
  `UpdateDatabaseDdl` call), so this is an option gap, not a framework gap.
  Library-level verification is possible today once Omni supports the
  option; the CLI probes (`spanner-query-plan-shape`, `plan-report`) would
  additionally need a way to apply database options because they pin
  `WithRandomDatabaseID()` and a static DDL file cannot name the random
  database.

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

## Shard-pattern seekability follow-up (gcpug/nouhau#135)

Re-verified the 2020 shard-note discussion (gcpug/nouhau PR #135) on Omni
2026.r1-beta with the plan-shape probe, using
`Order1M(ShardCreatedAt, CreatedAt DESC)`:

- `ShardCreatedAt BETWEEN 0 AND 9` + timestamp range: the interval
  discretization documented in the Spanner SQL paper (pub46103, 4.3 Filter
  tree) is optimizer-version dependent. Versions 1-6 discretize and seek
  both keys (`seekable_key_size=2`, no residual); versions 7-8 (current
  default) seek only the shard range and demote the timestamp to a
  Residual Condition. An explicit `IN (0, ..., 9)` enumeration does not
  help on v7-v8. Empty-database caveat applies (cost-based choice may
  differ with statistics).
- The per-shard rewrite (`UNNEST(GENERATE_ARRAY(0, 9))` driving a
  correlated `ARRAY(SELECT ... WHERE ShardCreatedAt = shard ...)`)
  produces an equality + timestamp-range two-key seek on every optimizer
  version, with per-shard `Local Limit` pushdown under the distributed
  union and only the final top-N hitting `Global Sort Limit`.
- Contract-selection nuance: the naive V1 shard query
  (`BETWEEN 0 AND 9 ORDER BY ... LIMIT`) does NOT set `Full scan: true`
  on Omni (the seek covers the shard range, which simply spans the whole
  index), so a `no_full_scan` contract would not flag it; the
  `Sort Limit` it contains is caught by `no_explicit_sort` instead.
  This matches the intent of the full-scan remediation text that
  recommends per-shard equality probes for sharded timestamp queries.

The same-day documentation update in spanner-hacks
`seek-residual-conditions.md` records the optimizer-version dependence
with the reproduction query.
