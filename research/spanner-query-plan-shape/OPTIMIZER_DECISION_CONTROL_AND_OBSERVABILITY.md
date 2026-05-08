# Optimizer Decision Control And Plan Observability

This note organizes the relationship among three surfaces:

- optimizer versions, which can change optimizer decisions;
- hints or SQL rewrites, which can sometimes force a decision independently of
  the selected optimizer version;
- `QueryPlan`, which may or may not make the selected decision visible.

It is a higher-level interpretation layer over:

- [Spanner optimizer versions and hints](SPANNER_OPTIMIZER_AND_HINTS.md)
- [Optimizer version matrix observations](OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md)
- [Query execution operator observations](QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md)

## Review Questions

For every optimizer-version boundary or hint, ask four separate questions:

1. Does the optimizer version change a plan-time decision?
2. Can the old or new shape be forced while using the opposite optimizer
   version?
3. If a hint forces the decision, is that forced decision visible in
   `QueryPlan`?
4. If it is not visible in `QueryPlan`, is the remaining evidence a `PROFILE`
   or Query Stats concern rather than a PLAN contract concern?

These questions are intentionally separate. A hint can be accepted but not
visible. A version change can be visible but not controllable. A runtime effect
can be important but still outside structural PLAN contracts.

## Vocabulary

- **Plan-time decision**: a choice fixed before execution, such as join method,
  join build side, scan target, seek prefix, distributed merge placement,
  aggregate method, predicate placement, back-join placement, or CTE spooling.
- **Runtime effect**: behavior that depends on execution, such as actual rows
  scanned, latency, CPU, lock contention, hash-join spill, or number of hash
  join passes actually needed.
- **Controllable**: a hint or SQL rewrite can reproduce the desired shape even
  when the optimizer version would otherwise choose a different shape.
- **Observable**: the selected decision appears in `QueryPlan` as an operator,
  child-link type, scalar predicate, short representation, or metadata field.
- **Contractable**: a decision is both observable and stable enough to validate
  from PLAN output without relying on raw SQL text or runtime counters.

## Control And Observability Notes

### Predicate Prefix Extraction

- Version boundary or hint: v2 extracts `STARTS_WITH`-style split and seek
  conditions for checked forced-index `REGEXP_CONTAINS` / `LIKE` probes.
- Control: the newer seek shape can be reproduced portably by rewriting SQL to
  explicit `STARTS_WITH` or range predicates and forcing the target index when
  needed. There is no pure hint that makes v1 perform the newer rewrite.
- PLAN observability: visible as `Split Range`, `Seek Condition`,
  `Residual Condition`, and scan `seekable_key_size`.
- Contract implication: prefer contracts on seek predicates or absence of
  residual-only full scans, not on optimizer version alone.

### Distributed Merge And Sorted `LIMIT`

- Version boundary or hint: v3 introduces distributed merge behavior;
  `ALLOW_DISTRIBUTED_MERGE=FALSE` reverts affected v3+ cases to older top-level
  sort shapes.
- Control: mostly one-way across the observed boundary. Newer versions can
  force the older shape with `ALLOW_DISTRIBUTED_MERGE=FALSE`; v1/v2 cannot be
  hinted into the v3 distributed merge implementation.
- PLAN observability: visible as `Distributed Union` with
  `preserve_subquery_order: true`, local `Sort` / `Sort Limit`, and changed
  sort placement.
- Contract implication: contract the structural effect, such as no top-level
  full sort or required distributed merge metadata, rather than only the hint.

### Push Broadcast Hash Join

- Version boundary or hint: v3 accepts
  `JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN`; v1/v2 reject it.
- Control: not backportable to v1/v2. In v3+, the operator can be requested
  with `JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN`, subject to query eligibility.
- PLAN observability: visible as `Push Broadcast Hash Join` or related
  outer/semi/anti-semi variants, with map-side local `Hash Join` subtrees.
- Contract implication: contract the operator family and distinguish wrapper
  join from inner local hash joins.

### Join Algorithm Selection And Commutativity

- Version boundary or hint: v5 introduces cost-based hash/apply selection and
  join commutativity; later versions may adjust join order.
- Control: often partially controllable with `JOIN_METHOD`,
  `FORCE_JOIN_ORDER`, `HASH_JOIN_BUILD_SIDE`, `BATCH_MODE`, and per-input
  table hints such as `FORCE_INDEX` and `SEEKABLE_KEY_SIZE`. Exact distributed
  wrappers, internal nodes, and back-join placement may still vary.
- PLAN observability: broad join family, build/probe side, apply batching, scan
  target, and back joins are visible. Some internal placement details are only
  indirectly visible.
- Contract implication: use semantic contracts such as no forbidden join
  family, required build/probe orientation, no back join, or no blocking
  operator under `LIMIT`; avoid exact rendered-plan snapshots.

### Hash Join One-Pass Versus Multi-Pass Intent

- Version boundary or hint: v4 introduces single-pass hash join via
  `HASH_JOIN_EXECUTION=ONE_PASS`.
- Control: the hint is syntactically accepted with `JOIN_METHOD=HASH_JOIN`, but
  the observed PLAN does not distinguish `MULTI_PASS` and `ONE_PASS`.
- PLAN observability: not visible in observed `PlanNode` metadata. Raw YAML
  shows `Hash Join` with `execution_method` and `join_type`, but no
  `HASH_JOIN_EXECUTION` field. Actual spill/pass behavior is a runtime concern.
- Contract implication: do not create a PLAN-only contract for one-pass versus
  multi-pass. At most, preserve the requested hint text or use PROFILE / Query
  Stats evidence.

### Secondary-Index Choice For Leading Predicates

- Version boundary or hint: a v4 DBaaS example uses a secondary index and back
  join where v3 scans the base table. The same minimal example did not
  reproduce on Spanner Omni 2026.r1-beta.
- Control: the target access path can usually be forced with `FORCE_INDEX` or
  `_BASE_TABLE`; seek prefix can be constrained with `SEEKABLE_KEY_SIZE` where
  applicable. The cost-based choice itself can remain environment-sensitive.
- PLAN observability: visible through scan target, `Seek Condition`,
  `Residual Condition`, `seekable_key_size`, and back-join operators.
- Contract implication: contract the desired access path and seek/residual
  split when it matters; keep DBaaS/Omni differences documented as evidence
  scope.

### Aggregate Method

- Version boundary or hint: `GROUP_METHOD=HASH_GROUP` / `STREAM_GROUP` controls
  aggregate method.
- Control: highly controllable for eligible `GROUP BY` shapes. Input ordering
  and scan path may still affect whether stream aggregation is efficient.
- PLAN observability: visible as aggregate operator metadata/family used by
  plan-contract normalization.
- Contract implication: good PLAN-contract target for `no_hash_aggregate` or
  requiring stream aggregate.

### `GROUPBY_SCAN_OPTIMIZATION`

- Version boundary or hint: v2/v3 history includes `GROUP BY` scan
  improvements; table hint exists.
- Control: hint can be requested in table position, but the historically
  important effect may be rows scanned rather than a structural shape change.
- PLAN observability: weak in PLAN. Some probes show placement changes, but the
  intended row-scan reduction can be PROFILE / Query Stats evidence without a
  clear operator change.
- Contract implication: avoid relying on PLAN-only contracts for the
  performance effect; record rows-scanned evidence when validating it.

### Full Outer Join Predicate/Limit Pushdown

- Version boundary or hint: v6 matrix probes show a visible change around full
  outer joins.
- Control: no direct hint is known for the pushdown transformation. Surrounding
  choices can sometimes be constrained with join and access-path hints or SQL
  rewrites.
- PLAN observability: visible as operator placement changes, such as
  `Outer Apply` versus `Distributed Outer Apply`, and changed predicate/limit
  placement.
- Contract implication: contract the undesirable structural outcome if needed,
  but expect lower portability than direct hint-controlled features.

### Index Union

- Version boundary or hint: v7 introduces cost-based index union;
  `INDEX_STRATEGY=FORCE_INDEX_UNION` can request it for eligible disjunctions.
- Control: the union shape can often be forced for eligible queries. Disabling
  the cost-based transformation may require SQL rewrite or forcing a specific
  single index/base table.
- PLAN observability: visible as `Union All` over index-scan branches plus
  dedup/aggregate shapes.
- Contract implication: good PLAN-contract target when index union is intended;
  also useful as a guard against unexpected broad scans.

### Smart Seek Versus Scan

- Version boundary or hint: v7 improves cost-based seek/scan selection;
  `SEEKABLE_KEY_SIZE` can constrain seek prefix with `FORCE_INDEX`.
- Control: partially controllable. You can force a scan target and seekable
  prefix, but optimizer-specific predicate extraction still affects what can
  become a seek condition.
- PLAN observability: visible as scan target, `seekable_key_size`,
  `Seek Condition`, and residual predicates.
- Contract implication: contract seekability and residual/full-scan conditions
  rather than the optimizer version.

### Informational Foreign-Key Join Elimination

- Version boundary or hint: v8 can remove joins when
  `USE_UNENFORCED_FOREIGN_KEY=TRUE`; `FALSE` retains joins in checked probes.
- Control: newer versions can disable the optimization with
  `USE_UNENFORCED_FOREIGN_KEY=FALSE`. Older versions did not eliminate the join
  with `TRUE` in the observed matrix.
- PLAN observability: visible as the presence or absence of the referenced-table
  scan/join.
- Contract implication: good PLAN-contract target when a referenced-table join
  must not appear, but it is version-gated.

### CTE Materialization And Spooling

- Version boundary or hint: v8 considers `WITH` clauses in cost-based choices.
  Simple local probes were stable across versions.
- Control: no direct CTE materialization hint is known. SQL rewrite or external
  materialization is the main control.
- PLAN observability: visible when it happens as `SpoolBuild`, `SpoolScan`, and
  repeated-reference apply structure.
- Contract implication: contract spool presence/absence only for known stable
  cases; avoid broad claims from simple synthetic probes.

### Commit Timestamp Predicate Pushdown

- Version boundary or hint: `ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=TRUE` adds a
  storage-pruning predicate for eligible commit timestamp filters.
- Control: controllable with the statement hint and qualifying SQL predicate.
  It can coexist with a relational `Full scan: true` signal.
- PLAN observability: visible as a `Timestamp Condition` child link on the
  scan. Rows-scanned reduction remains runtime/profile evidence.
- Contract implication: contract `require_timestamp_condition` or
  `no_full_scan_without_timestamp_condition` for recent-data scan reviews.

### Function Inlining

- Version boundary or hint: function hint `DISABLE_INLINE=TRUE/FALSE` changes
  scalar computation placement in checked probes.
- Control: controllable with function hint syntax on eligible functions.
- PLAN observability: visible through scalar function duplication versus a
  `Compute` operator and references to the computed value.
- Contract implication: contractable if scalar-level plan shape matters, but
  keep it separate from relational operator contracts.

### Scan And Execution Method

- Version boundary or hint: `SCAN_METHOD` and `EXECUTION_METHOD` can affect
  row/batch conversion and scan metadata.
- Control: partially controllable; some combinations are rejected depending on
  query shape.
- PLAN observability: visible as scan `scan_method`, relational
  `execution_method`, and `RowToDataBlock` / `DataBlockToRow` nodes.
- Contract implication: useful for diagnostics and targeted contracts, but
  validate rejected combinations separately.

### Lock Acquisition Mode

- Version boundary or hint: `LOCK_SCANNED_RANGES=exclusive/shared` changes how
  scanned ranges are locked in read-write transactions.
- Control: the hint controls lock mode, but the lock footprint comes from the
  scan condition range. A full scan means the whole table or index range is in
  scope. `exclusive` is mainly a contention-control tool for write-after-read
  patterns that would otherwise repeatedly abort and retry.
- PLAN observability: read-only PLAN probes do not expose a distinct lock-mode
  field, which is reasonable because `LOCK_SCANNED_RANGES` is a no-op outside
  read-write transactions. A read-write `AnalyzeQuery` recheck also produced
  identical `Scan` metadata for `shared` and `exclusive` in the observed
  Spanner Omni build. This is also natural if read-only and read-write
  executions share the same logical plan cache and apply locking below the
  cached plan.
- Contract implication: keep observed lock conflicts outside PLAN-only
  contracts. Revisit contract support only if read-write plan capture exposes
  requested lock mode as scan metadata or a child-link signal.

## Practical Classification

### Strong PLAN Contract Candidates

These decisions are both controllable and visible in the observed plan output:

- join family selected by `JOIN_METHOD`;
- hash build/probe orientation selected by `HASH_JOIN_BUILD_SIDE`;
- apply batching selected by `BATCH_MODE` where accepted;
- distributed merge versus top-level sort for affected v3+ sorted queries;
- aggregate method selected by `GROUP_METHOD`;
- scan target and seek prefix selected by `FORCE_INDEX` / `SEEKABLE_KEY_SIZE`;
- index union selected by `INDEX_STRATEGY=FORCE_INDEX_UNION`;
- timestamp predicate pushdown selected by `ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN`;
- informational foreign-key join elimination in versions where it is supported.

### Visible But Weakly Controllable

These are visible in PLAN but have no complete direct hint surface:

- computation pushdown through joins;
- full outer join predicate/limit pushdown;
- CTE spooling and materialization;
- some back-join placement decisions;
- cost-based join distribution wrappers beyond the main join method and input
  orientation.

For these, prefer contracts that prohibit or require the resulting structural
property rather than contracts that assume a specific optimizer transformation.

### Accepted But Not PLAN-Observable

These hints or effects are currently not reliable PLAN-contract surfaces:

- `HASH_JOIN_EXECUTION=MULTI_PASS/ONE_PASS`: accepted in checked hash-join
  shapes but not visible in observed `PlanNode` metadata.
- `GROUPBY_SCAN_OPTIMIZATION`: the important effect can be rows scanned rather
  than a stable structural difference.
- lock conflicts from `LOCK_SCANNED_RANGES`; validate them separately with
  `SPANNER_SYS.LOCK_STATS*`, because conflicts are reported by row key range,
  column, lock mode, and transaction tag rather than as plan operators. The
  requested lock mode would be contractable only if read-write PLAN output made
  it visible.
- actual hash-join spill, pass count, CPU, latency, lock wait, and rows scanned.

These require raw SQL review, PROFILE output, Query Stats, or workload
measurement rather than structural PLAN contracts.

## Lock Statistics Follow-Up

`LOCK_SCANNED_RANGES` deserves a separate validation track. It can change the
requested lock mode for scanned ranges in read-write transactions, but the
observable effect is normally contention behavior, not relational plan shape.
In a single read-only transaction, `LOCK_SCANNED_RANGES` is a no-op, so a
read-only PLAN has no reason to expose it. A read-write `AnalyzeQuery` probe was
also checked with `LOCK_SCANNED_RANGES=shared` and `exclusive` on the same point
read. The raw YAML output had identical `Scan` metadata and no requested
lock-mode field in the observed Spanner Omni build. If read-only and read-write
queries share the same logical plan cache, this identical plan shape is the
expected outcome: locking mode can be applied by the transaction execution layer
without changing the cached relational plan.

There is one important boundary: if read-write PLAN capture exposed the
requested `exclusive` / `shared` lock mode as scan metadata or another
structural signal, that request would be a legitimate PLAN-contract candidate.
The actual lock conflicts would still require lock statistics.

Useful evidence sources:

- Official lock statistics tables:
  <https://docs.cloud.google.com/spanner/docs/introspection/lock-statistics>
- Background and example validation approach:
  <https://zenn.dev/apstndb/articles/a62ac78b3b91bb>

The validation should intentionally create a small number of read-write
transaction conflicts, set transaction tags, and inspect
`SPANNER_SYS.LOCK_STATS_TOP_*` / `SPANNER_SYS.LOCK_STATS_TOTAL_*`. The expected
correlation is:

- `QueryPlan` explains the scan condition range, such as a point seek, prefix
  range, or full table/index scan.
- Lock statistics report the conflicted row key range, column, lock mode, and
  transaction tag.
- `LOCK_SCANNED_RANGES=exclusive` should move some write-after-read contention
  from commit-time abort/retry behavior toward earlier waiting on the scanned
  range. This can reduce wasted retry work for high-contention read-modify-write
  patterns, while `shared` remains preferable when shared reads should not
  block one another.

Because the footprint follows the scan condition, plan-quality contracts still
matter indirectly: a full scan with `LOCK_SCANNED_RANGES=exclusive` can put the
whole table or index range in the lock-conflict scope.

## Implications For `spanner-query-gen`

- Plan contracts should prefer stable semantic properties over exact rendered
  plan snapshots.
- A contract should document whether it relies on operator names, child-link
  types, scalar predicates, metadata, or runtime evidence.
- `--require-optimizer-pinning` remains useful, but optimizer pinning alone is
  weaker than a semantic plan contract plus a known hint strategy.
- For every new predefined contract, add a note about whether the relevant
  decision is controllable, visible, and replayable from a saved plan report.
- Do not create PLAN-only contracts for decisions that are not visible in
  `QueryPlan`, even if the corresponding hint exists.
