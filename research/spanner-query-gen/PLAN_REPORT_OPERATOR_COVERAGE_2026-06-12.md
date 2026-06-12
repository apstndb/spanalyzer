# Spanner Omni Plan Verification (2026-06-12)

Non-normative observations from a verification session against Spanner Omni
(`spanemuboost` default image,
`us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`).
Originally written as a chronological log of one day's probes; restructured
topically the same day. Corrections made during the session are summarized in
[Retracted claims](#retracted-claims) instead of being threaded through the
text.

Plan evidence was gathered through four paths:

- `spanner-query-gen plan-report` over v1alpha configs (the analyzer path),
- the `tools/spanner-query-plan-shape` probe (raw SQL, no analyzer),
- the standalone `plancontract` module fed with raw `AnalyzeQuery` plans
  (the analyzer-independent classification path),
- `ReadWriteTransaction.AnalyzeQuery` in PLAN mode for DML (nothing is
  executed).

Durable results are pinned by two Omni-gated integration tests in
`cmd/spanner-query-gen/omni_family_coverage_test.go`:
`TestIntegrationPlanReportOperatorFamilyCoverageOnOmni` (query shapes through
plan-report) and `TestIntegrationDMLOperatorFamilyCoverageOnOmni` (DML shapes
through the plancontract path).

## Operator family coverage status

### Observed and classified warning-free

The initial 17-query plan-report sweep (hinted joins and aggregates,
sort+limit, `UNION ALL`, `UNNEST`, scalar/array subqueries,
`EXISTS` / `NOT EXISTS`, `DISTINCT`, `WITH`, timestamp ranges, interleaved
multi-table joins) plus targeted follow-up probes produced
`operator_family_counts["unknown"] == 0` and zero classification warnings on
every plan. Repro conditions worth recording:

- Contextual classification works on live plans:
  `PUSH_BROADCAST_HASH_JOIN` produces both `push_broadcast_hash_join` and
  `push_broadcast_hash_join_internal_hash_join`; non-colocated
  `EXISTS` / `NOT EXISTS` produce `distributed_semi_apply` /
  `distributed_anti_semi_apply` plus their `_internal_apply` families.
- Colocated `EXISTS` / `NOT EXISTS` against an interleaved child keyed by
  the full parent key → standalone `semi_apply` / `anti_semi_apply`.
- `SELECT 1` → `unit_relation`; `WHERE FALSE` → `empty_relation`.
- `GROUP BY ... HAVING` → `filter` (HAVING is a Filter, not Filter Scan,
  matching spanner-hacks operators.md).
- `ORDER BY SingerId, Title` on `Albums(SingerId, AlbumId)` → `minor_sort`
  (prefix already key-ordered).
- A CTE referenced twice through `UNION ALL` → `spool_build` + `spool_scan`
  plus `union_all` / `union_input`.
- A hinted hash join against a non-interleaved table → `hash_join` with
  `bloom_filter_build` on the build side (interleave-eliminated joins never
  build bloom filters; see join elimination below).
- Full-text search (`SEARCH` over a `TOKENLIST` column with a search index)
  → `search_predicate`, `search_query_conversion_tvf`, `verify_determinism`.
- `TABLESAMPLE BERNOULLI` → `random_id_assign` + `filter`;
  `TABLESAMPLE RESERVOIR` → `random_id_assign` + `full_sort` + `limit`
  (exactly as documented in query-operators-unary).
- `ARRAY(SELECT ...)` → `array_subquery`; the non-decorrelatable conditional
  `IF(..., (SELECT COUNT(*) ...), 0)` from spanner-hacks operators.md →
  `scalar_subquery`. A plain correlated `(SELECT COUNT(*) ...)` decorrelates
  to `apply_join` + `stream_aggregate` and contains no Scalar Subquery node.
- `SELECT ChangeRecord FROM READ_EverythingStream(start_timestamp => ...)`
  → `change_stream_tvf` (via the plancontract path; see below for why
  plan-report cannot reach it).
- The spanner-hacks quantified-path graph repro
  (`MATCH ...-[c:CollabWith]->{1,2}...`) → `recursive_union` +
  `recursive_spool_scan`. Recursive Union introduces `input_0` / `input_1`
  child-link types not seen elsewhere.
- DML INSERT VALUES / INSERT ... THEN RETURN / UPDATE / DELETE /
  INSERT SELECT → `apply_mutations` in every shape (plus `union_all` /
  `union_input` / `unit_relation` for VALUES and `serialize_result` for
  THEN RETURN).

With scalar-kind expression nodes classified into the `scalar` family (see
classifier defects below), a strict
`forbid: operator_family: unknown, max_count: 0` contract passes on real
plans and is now a usable policy.

### Observable only outside the plan-report config path

- `apply_mutations`: DML targets are excluded from plan-report because public
  `queries[].result.cardinality` accepts only row-returning modes. The
  plan-shape `--case dml` probe and
  `TestIntegrationDMLOperatorFamilyCoverageOnOmni` cover these shapes via
  `ReadWriteTransaction.AnalyzeQuery`.
- `change_stream_tvf`: the GoogleSQL frontend catalog does not register
  `READ_<stream>` TVFs, so the analyzer path rejects the query; raw
  `AnalyzeQuery` plans classify fine.

### Confirmed unobservable on Omni 2026.r1-beta

- `local_split_union`: requires placement tables, and `CREATE PLACEMENT`
  fails with `Unimplemented: Geo-partitioning is not supported for this
  environment`.
- The `ML.PREDICT` TVF shape (query-operators-unary shows ML.PREDICT
  compiling to a `TVF` operator): `CREATE MODEL` fails with an empty-message
  `InvalidArgument`. A non-search TVF would classify as `unknown`, which
  remains the correct conservative fallback until a real plan is available.
- `mini_batch_assign` / `mini_batch_key_order` / `row_count`: undocumented
  operators observed only in SELECT back-join shard-optimization shapes on
  Cloud Spanner optimizer v5, never on Omni (see
  `../spanner-query-plan-shape/OPERATOR_VERIFICATION_FOLLOWUP.md`). Note the
  naming collision with querygen's unrelated DML result mode `row_count`.

### Remaining gaps

- `generate_relation`: no known repro anywhere; the official
  query-operators-leaf page documents only generic properties with no
  example, and `UNNEST(GENERATE_ARRAY(...))` classifies as `array_unnest`.
- The generic `join` / `aggregate` fallbacks: only reachable when
  classification metadata is missing, which has not been observed live.

## Classifier and catalog defects found and fixed

All found by the probes above, all fixed the same day with regression tests:

1. Scalar-kind expression nodes (Reference, Function, Constant, Parameter)
   all classified as `unknown`, flooding every report with
   `operator_family_unknown` warnings and making the strict no-unknown
   policy unusable. Fixed by introducing the `scalar` family.
2. The first version of that fix checked `PlanNode.Kind` before display-name
   mapping, which shadowed concrete families on scalar-kind operator nodes:
   `Array Subquery` and `Scalar Subquery` are kind `SCALAR` (the only scalar
   operators rendered in standard plan trees, per spanner-hacks operators.md)
   and stopped classifying. Fixed by applying display-name families first
   with `scalar` as the fallback for otherwise-unknown scalar-kind nodes.
3. The Search Query Conversion TVF classifier read the lowercase `name`
   metadata key, but Omni emits capitalized `Name`, so the TVF always
   classified `unknown`. Both spellings are accepted now.
4. The plan-report invariant validator recomputed only the derived
   `explicit_sort` family and rejected any report whose subtree contained a
   stream-blocking operator (for example `GROUP_METHOD=HASH_GROUP`); the
   validator now shares the producer's derived-count helper.
5. The catalog rejected `CREATE SEARCH INDEX` (`unsupported DDL
   *ast.CreateSearchIndex`), so full-text search schemas could not flow
   through plan-report at all. Search indexes are now ignored like regular
   and vector indexes for row-type analysis.

## Optimizer behavior observations

### Join elimination follows declared referential integrity

`SELECT s.SingerId, a.Title FROM Singers AS s JOIN@{JOIN_METHOD=HASH_JOIN}
Albums AS a ON s.SingerId = a.SingerId` over interleaved tables compiles to a
single `Table Scan on Albums` with no join operator: parent-row existence is
guaranteed and only the join key is projected from the parent. Probing schema
variants showed the driver is declared referential integrity in general:

- `INTERLEAVE IN PARENT`: eliminated.
- Enforced `FOREIGN KEY`: eliminated.
- `FOREIGN KEY ... NOT ENFORCED` (informational): also eliminated — the
  optimizer trusts the declared constraint, matching the documented caveat
  that nonconforming data can change query results.
- Plain table with no constraint: join survives (`Distributed Cross Apply`).
- Projecting a non-key parent column (`s.FirstName`) restores real join
  operators for every join method.
- Stable across optimizer versions 1-8.

`JOIN_METHOD` hints on an eliminated join are silently ignored (no error, no
warning). Contract implication: a contract asserting a specific join family
can fail or trivially pass because the join was eliminated, not because the
hint changed; operator-presence contracts describe the actual shape and are
more robust than assumptions about hint effects.

The `USE_UNENFORCED_FOREIGN_KEY` statement hint (default `TRUE`, overriding
the `use_unenforced_foreign_key_for_query_optimization` database option per
statement) controls only the informational-FK part: with `FALSE` the NOT
ENFORCED FK join is no longer eliminated, while enforced-FK and interleave
elimination still happen. It is statement-scope only; the join-hint position
is rejected with `Unsupported hint`. The corresponding database option could
not be verified: Omni rejects it via `ALTER DATABASE` with an empty-message
`InvalidArgument` (while `version_retention_period` applies fine, so the
mechanism works) and does not parse the documented `SET DATABASE OPTIONS`
syntax. Once Omni supports the option, library-level verification works
today; the CLI probes would additionally need a database-option entry point
because they pin `WithRandomDatabaseID()` (tracked in TODO.md).

### Aggregate method deterministically follows available orderings

The unhinted `GROUP BY ... HAVING` aggregate method is deterministic on this
build (verified 3x per schema): no index on the grouping key → Hash
Aggregate; an index providing grouping-key order → Stream Aggregate. This
matches the spanner-hacks operators.md description. Contracts on the
aggregate family should still pin `GROUP@{GROUP_METHOD=...}` when index
design is expected to evolve, because adding or dropping an index on the
grouping key changes the unhinted choice.

### Shard-range seekability and interval discretization

Follow-up to gcpug/nouhau PR #135 and the Stack Overflow thread it relates
to. See
[`../spanner-query-plan-shape/TIMESTAMP_ORDERED_SHARD_QUERY_OBSERVATIONS.md`](../spanner-query-plan-shape/TIMESTAMP_ORDERED_SHARD_QUERY_OBSERVATIONS.md)
(2026-05-08), which already verified that thread end to end (the
`HAVING MIN` rewrite shape, `GROUPBY_SCAN_OPTIMIZATION` producing no
PLAN-shape difference, the arbitrary-limit rewrite, back-join placement, and
the default-optimizer `seekable_key_size=1` shard-range observation). The new
information here, probed with `Order1M(ShardCreatedAt, CreatedAt DESC)`:

- The interval discretization documented in the Spanner SQL paper (pub46103,
  4.2/4.3: small integer intervals are discretized) is optimizer-version
  dependent. Versions 1-6 discretize `ShardCreatedAt BETWEEN 0 AND 9` and
  seek both keys (`seekable_key_size=2`, no residual); versions 7-8 (current
  default) seek only the shard range and demote the timestamp to a Residual
  Condition. An explicit `IN (0, ..., 9)` enumeration does not help on
  v7-v8.
- The per-shard rewrite (`UNNEST(GENERATE_ARRAY(0, 9))` driving a correlated
  `ARRAY(SELECT ... WHERE ShardCreatedAt = shard ...)`) produces an
  equality + timestamp-range two-key seek on every optimizer version, with
  per-shard `Local Limit` pushdown and only the final top-N hitting
  `Global Sort Limit`.
- Caveat (per the paper and spanner-hacks `seek-residual-conditions.md`): a
  displayed Seek Condition, including `seekable_key_size=2`, is a
  plan-representation fact, not a runtime performance guarantee. Range
  extraction is a runtime filter-tree process, fragmented many-range seeks
  can degrade and be processed like residual filtering, and the paper notes
  the seeks-vs-scans tradeoff can make extraction cost exceed its benefit.
  The v7-v8 change away from discretized seeks is not necessarily a
  regression; judging performance impact needs PROFILE statistics over real
  data, which PLAN-only contracts intentionally do not cover.
- Contract-selection nuance: the naive shard query
  (`BETWEEN 0 AND 9 ORDER BY ... LIMIT`) does NOT set `Full scan: true` on
  Omni (the seek covers the shard range, which spans the whole index), so
  `no_full_scan` would not flag it; its `Sort Limit` is caught by
  `no_explicit_sort` instead. This matches the full-scan remediation text
  that recommends per-shard equality probes for sharded timestamp queries.

The optimizer-version dependence is also recorded in spanner-hacks
`seek-residual-conditions.md` with the reproduction query.

### Plan differences across optimizer versions

Comparing `--optimizer-version 1` against `8` over the 17-query sweep by
operator tree digest: 7 queries changed (`ApplyJoinSingersAlbums`,
`PushBroadcastHashJoin`, `SortLimit`, `SemiJoinExists`,
`AntiSemiJoinNotExists`, `WithClause`, `InterleavedJoin`); 10 were identical.
`--require-optimizer-pinning` failed with `optimizer_not_pinned,
statistics_package_not_pinned, query_optimizer_not_pinned` when no pin flags
were supplied, and `optimizer.effective` stayed `not_recorded` as documented.

## Omni platform checks (docs.cloud.google.com/spanner-omni/develop)

- `version_retention_period` accepts up to `30d` and rejects `31d` with
  ``range [`1h`, `30d`]`` (DBaaS caps at 7 days), confirming the documented
  difference.
- `@{ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=TRUE}` still produces a
  `Timestamp Condition` child link even though Omni has no tiered storage;
  the develop page's "might not fully apply" reads as a performance note,
  not a plan-shape difference.
- `SEARCH(..., enhance_query => TRUE)` is accepted at both PLAN and
  execution time even though the develop page documents enhanced query mode
  as unsupported on Omni — apparently a silent no-op rather than a
  rejection.
- Change streams work as documented (DDL plus `READ_<stream>` plan
  acquisition).
- Geo-partitioning (placement tables) is unimplemented; see
  `local_split_union` above.

## Contract behavior verified on live plans

Intentional violations produced the expected results with `--check`
(exit code 1): `no_full_scan` failed with `matched_operator_indexes` and
index-design/query-shape remediations; `no_explicit_sort` failed on the
`ORDER BY ... LIMIT` plan; a direct `forbid: operator_family: hash_join`
failed with the JOIN_METHOD hint remediation; `require_timestamp_condition`
failed for a non-commit-timestamp `TIMESTAMP` predicate with an explanatory
remediation; a CEL contract reading `operator_family_counts["scalar"]`
evaluated with `replayable_from_report=true`.

## Limitations

- All plans were taken with `AnalyzeQuery` against empty tables and default
  statistics. Unhinted operator choices may differ under real data and
  refreshed statistics; hinted shapes and the classification results were
  not data-dependent in the observed cases. Performance claims are out of
  scope by design (see the repository README positioning).
- Backend identity is `not_recorded` (spanemuboost does not yet expose the
  resolved image digest), so the image tag above is the spanemuboost default
  at the time of the run, not observed evidence.
- The probe configs and schemas are not all checked in; the durable subset
  is pinned by the two integration tests named above.

## Retracted claims

Two claims from earlier drafts of this note were wrong and are kept here so
they are not re-derived:

- "The unhinted aggregate method flaps between Stream and Hash across runs."
  Wrong: the two runs used different schemas (one had an index on the
  grouping key). The choice is deterministic; see the aggregate section.
- "`row_count` and `mini_batch_*` are DML operators." Wrong: they are
  undocumented SELECT back-join operators (optimizer v5, DBaaS-only
  observations). The naming collision with querygen's DML result mode
  `row_count` caused the mix-up.

## Related notes

- `../spanner-query-plan-shape/TIMESTAMP_ORDERED_SHARD_QUERY_OBSERVATIONS.md`
  — the 2026-05-08 end-to-end verification of the timestamp-ordered shard
  query thread.
- `../spanner-query-plan-shape/OPERATOR_VERIFICATION_FOLLOWUP.md` — operator
  vocabulary verification including the MiniBatch/RowCount environment
  sensitivity.
- spanner-hacks `operators.md` and `seek-residual-conditions.md` — the
  knowledge-side documentation updated from the same observations.
