# Set-operation and DISTINCT hint observations (2026-08-04)

This note records environment-specific query-plan observations. It is not a
stable Spanner optimizer contract and does not claim managed Cloud Spanner
equivalence.

## Evidence

- Official query syntax source: `documents/docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax`
  from the Google Developer Knowledge API, updated 2026-08-06 and retrieved
  2026-08-11. The corresponding public page is
  <https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax>.
- Official operators source:
  `documents/docs.cloud.google.com/spanner/docs/reference/standard-sql/operators`,
  updated 2026-08-04 and retrieved 2026-08-11. The current document does not
  list `IS [NOT] DISTINCT FROM`. The
  [BigQuery GoogleSQL operators page](https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/operators#is_distinct_from_operator)
  does document the operator and its NULL semantics, but that sibling product
  page is not Spanner support evidence. Pinned Omni runtime evidence below,
  rather than the BigQuery documentation, establishes acceptance in the
  tested Spanner environment.
- Syntax evidence: the mirrored `google/googlesql` grammar (local SHA-256
  `7996a02316feab9109e30d1991f44fa66cec2c8f95ae83c889ec7558bdfb6309`)
  includes the feature-gated `distinct_operator` production. The spanalyzer
  GoogleSQL frontend using `go-googlesql` v0.3.0 and pinned Omni accepted both
  spellings. Omni constant-folded NULL/NULL to `true` for `IS NOT DISTINCT
  FROM` and `false` for `IS DISTINCT FROM` in optimizer versions 1 through 8.
  Memefish `64f857b2c61e` still rejects both forms, so its recovered AST is not
  syntax-acceptance evidence for these probes.
- Runtime: Spanner Omni
  `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`, local
  image digest
  `sha256:e98a088fa66d4a87dbb560d729bf21d998bb843f6018bd8dc118fe320e671886`.
- API: `AnalyzeQuery(QueryMode=PLAN)` through `spanner-query-plan-shape`.
- Plan data: empty tables created from the built-in documentation schema and
  three independent set-operation input tables.
- Result data: populated nullable, duplicate-bearing single-column and
  multi-column fixtures in
  `set_operation_equivalence_omni_test.go`.
- Retained scope, revalidated on 2026-08-11: 116 submitted probes, 99 returned
  plans and 17 returned an expected hint-validation error. The optimizer
  version matrix submitted all 116 probes at each version from 1 through 8
  (928 total): 768 plans and 160 expected errors. The
  `allow_distributed_merge` default/true/false matrix submitted 348 probes:
  297 plans and 51 expected errors.

The official source says that `UNION DISTINCT` applies duplicate elimination
after the union. It documents `JOIN_METHOD`, `BATCH_MODE`, and
`FORCE_JOIN_ORDER` as join hints. `GROUP_METHOD` is documented for `GROUP BY`,
not DISTINCT. The table hint `GROUPBY_SCAN_OPTIMIZATION` explicitly applies to
eligible `GROUP BY` and `SELECT DISTINCT` queries.

The same query-syntax source says that combining a value table with a
one-column ordinary table always yields a value table. The pinned
`google/googlesql` resolver at commit
`1f8aa333f4d6353cd3a64471fc83121df72df3f7` instead uses the first input to
determine value-table output. Pinned Omni v1-v8 followed the resolver rule for
all six set-operation/qualifier combinations: ordinary-first forms executed
at top level, while value-table-first forms reached the documented top-level
value-table restriction. The retained GoogleSQL-surface integration test also
executes the nested outer-SELECT workaround. This is environment-scoped
evidence of a documentation/runtime divergence.

## Set-operation results

| Query family | Hint | Observed plan distinction |
| --- | --- | --- |
| `UNION ALL` | default, HASH, APPLY | `Union All`; join-method hints did not produce a visible join-family change |
| `UNION DISTINCT` | default, HASH, APPLY | global/local Hash `Aggregate` above `Union All`; the join-method hints did not change that shape |
| `INTERSECT DISTINCT` | default | `Cross Apply` |
| `INTERSECT DISTINCT` | `JOIN_METHOD=HASH_JOIN` | `Hash Join`, `join_type=BUILD_SEMI` |
| `INTERSECT DISTINCT` | `JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE` | `Distributed Semi Apply` with two variable-bearing `Batch` links, containing `Semi Apply` |
| `INTERSECT DISTINCT` | `JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE` | `Semi Apply` |
| `INTERSECT DISTINCT` | `JOIN_METHOD=MERGE_JOIN` | `Merge Join`, `join_type=LEFT_SEMI`, `join_configuration=ONE_TO_MANY` |
| `INTERSECT DISTINCT` | `JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN` | `Push Broadcast Hash Join Semi Apply` |
| `INTERSECT DISTINCT` | `FORCE_JOIN_ORDER=TRUE` | changed the two-input default from `Cross Apply` to `Semi Apply` |
| three-input `INTERSECT DISTINCT` | default versus statement `FORCE_JOIN_ORDER=TRUE` | default contained `Semi Apply` plus `Cross Apply`; forced order contained two `Semi Apply` nodes |
| `EXCEPT DISTINCT` | default | `Anti-Semi Apply` |
| `EXCEPT DISTINCT` | HASH / APPLY batch true / APPLY batch false / MERGE / PUSH | respectively `Hash Join BUILD_ANTI_SEMI`, `Distributed Anti Semi Apply`, `Anti-Semi Apply`, `Merge Join LEFT_ANTI_SEMI`, and `Push Broadcast Hash Join Anti Semi Apply` |
| `INTERSECT ALL` | default / HASH / MERGE / APPLY batch true / APPLY batch false / PUSH | respectively `Hash Join INNER`, `Hash Join INNER`, `Merge Join INNER`, `Distributed Cross Apply`, `Cross Apply`, and `Push Broadcast Hash Join`; all retained `Generate Relation` |
| `EXCEPT ALL` | default / HASH / MERGE / APPLY batch true / APPLY batch false / PUSH | respectively `Outer Apply`, `Hash Join BUILD_OUTER`, `Merge Join LEFT_OUTER`, `Distributed Outer Apply`, `Outer Apply`, and `Push Broadcast Hash Join Outer Apply`; all retained `Generate Relation` |

`HASH_JOIN_BUILD_SIDE` on `INTERSECT` and `EXCEPT`, for both `ALL` and
`DISTINCT`, reached hint validation but returned `Unsupported hint:
HASH_JOIN_BUILD_SIDE` for both `BUILD_LEFT` and `BUILD_RIGHT`. It must not be
treated as a supported set-operation control merely because the general
hash-join hint exists.

The original `INTERSECT ALL` controls used `Singers` against `Albums`. That
fixture allowed the optimizer to eliminate the comparison by using schema
relationships, making accepted join hints vacuous. The retained controls use
`Albums` against `Songs`, whose projected `SingerId` is non-unique on both
sides and has no direct containment relationship. This exposes the join
families listed above before repeat-count restoration.
`FACTORIZED_MODE=FACTORIZE_BOTH` was accepted for this inner-join-shaped
`INTERSECT ALL` probe,
and variable-bearing Build and Probe links remained visible on its `Hash Join`.
The corresponding DISTINCT probes and `EXCEPT ALL` rejected factorized mode.
ALL and DISTINCT therefore need separate plan contracts even when they share a
set-operator spelling.

## Equivalent-query comparison

Separate result probes populated nullable, duplicate-bearing left and right
inputs. The following rewrites returned exactly the same ordered values as the
corresponding set operations in that fixture:

- `INTERSECT DISTINCT` used `SELECT DISTINCT` from the left input plus a
  correlated `EXISTS`;
- `EXCEPT DISTINCT` used `SELECT DISTINCT` from the left input plus a
  correlated `NOT EXISTS`;
- `INTERSECT ALL` grouped both inputs, joined their counts, and expanded
  `LEAST(left_count, right_count)` rows with `UNNEST(GENERATE_ARRAY(...))`;
- `EXCEPT ALL` grouped both inputs, left-joined their counts, and expanded
  `GREATEST(left_count - IFNULL(right_count, 0), 0)` rows.

The retained nullable correlation condition is
`right_key IS NOT DISTINCT FROM left_key`. Ordinary SQL equality alone is not
equivalent to set-operation row equality for NULLs. The earlier exploratory
equality-or-both-NULL expression matched the tested INT64/STRING cases, but it
is not used as a universal substitute for the native operator.
The observed results were:

| Pair | Result |
| --- | --- |
| `INTERSECT DISTINCT` and `DISTINCT ... EXISTS` | `NULL, 1` |
| `EXCEPT DISTINCT` and `DISTINCT ... NOT EXISTS` | `2, 3` |
| `INTERSECT ALL` and grouped-count expansion | `NULL, 1, 1` |
| `EXCEPT ALL` and grouped-count expansion | `NULL, 2, 3` |
| multi-column `INTERSECT DISTINCT` and NULL-safe `EXISTS` | `NULL:x, 1:NULL` |
| multi-column `EXCEPT DISTINCT` and NULL-safe `NOT EXISTS` | `1:a, 2:b, 3:NULL` |
| three-input `INTERSECT DISTINCT` and two separately hinted `EXISTS` predicates | `1` |
| three-input `EXCEPT DISTINCT` and two separately hinted `NOT EXISTS` predicates | `3` |

The retained Omni integration test covers the multi-column DISTINCT and
three-input mixed-method rows. The single-column and count-based ALL results
above came from the same pinned-runtime exploratory run but are not yet
retained as integration cases.

The multi-column rewrite applies `IS NOT DISTINCT FROM` independently to every
output column and deduplicates the entire output row. Omitting any output
column from the correlation is not equivalent to the direct set operation. It
must also reproduce the direct operation's resolved common supertype in both
the comparison and projection.

A read-only managed Spanner probe on 2026-08-12 demonstrated why the type step
is load-bearing. Direct `INTERSECT DISTINCT` between
`INT64(9007199254740993)` and the corresponding `FLOAT64` value returned the
common-supertype result `FLOAT64(9007199254740992)`. The naive correlated
rewrite compared the coercible values but projected the left `INT64`, so it
returned the exact integer `9007199254740993`. Explicitly casting both
correlation operands and the projected value to `FLOAT64` restored the direct
result. The integration test retains this counterexample so future rewrite
guidance cannot silently omit output coercion.

The DISTINCT rewrites exposed more hint controls than the direct set
operators. On the same duplicate-capable `Albums`/`Songs` inputs:

| Control | Direct set operation | `EXISTS` / `NOT EXISTS` rewrite |
| --- | --- | --- |
| `HASH_JOIN_BUILD_SIDE=BUILD_LEFT` | rejected | accepted; `BUILD_SEMI` / `BUILD_ANTI_SEMI` |
| `HASH_JOIN_BUILD_SIDE=BUILD_RIGHT` | rejected | accepted; `PROBE_SEMI` / `PROBE_ANTI_SEMI` |
| join hint plus `GROUP_METHOD=HASH_GROUP` | no direct GROUP hint on DISTINCT | accepted; Hash Aggregate plus the selected hash semi/anti join |
| join hint plus `GROUP_METHOD=STREAM_GROUP` | no direct GROUP hint on DISTINCT | accepted; Stream Aggregate plus the selected merge semi/anti join |
| batched APPLY | distributed semi/anti apply | also accepted; the rewrite exposed `order_preserving=true` in addition to the Batch links |
| different methods for two comparisons in a three-input expression | a hint on the second same-level set operator was rejected | accepted; one predicate produced a `BUILD_(ANTI_)SEMI` Hash Join and the other a row `(Anti-)Semi Apply` in the same plan |
| factorized semi/anti join | rejected | rejected |

The group/join examples establish that two hint families can be combined, but
not that their full Cartesian product is practical. An exploratory
`HASH_JOIN` plus `STREAM_GROUP` rewrite did not return from `AnalyzeQuery`
within two minutes on a minimal two-table schema and was interrupted. It is
not retained as a positive or negative contract. No independent-method claim
is made for that combination.

The direct DISTINCT plans had two deduplication pipelines, one for each
duplicate-capable input, before the semi/anti comparison. The rewritten plans
had one output deduplication pipeline and did not separately deduplicate the
right input; semi/anti semantics already need only existence. The current
plans represent each pipeline with two physical `Aggregate` nodes. The
nullable multi-column integration test therefore asserts direct count 4 and
rewrite count 2 for both intersection and difference across optimizer
versions 1 through 8. This is a concrete topology reduction, but it is not a
performance result: the probes used `PLAN` and did not measure latency, CPU,
memory, or bytes processed.

The multi-predicate advantage is structural, not merely a chosen default.
Spanner requires hints in a same-level set-operation chain to appear on the
first operation, so the direct three-input negative controls could not assign
different methods to the two comparisons. Two separately hinted correlated
predicates did so successfully. The mixed Hash/APPLY shape and the direct
syntax rejection were stable across optimizer versions 1 through 8.

The unhinted rewrites kept the same families across optimizer versions 1
through 8: Stream Aggregate over `Semi Apply` or `Anti-Semi Apply`. They were
not byte-equal throughout the matrix because versions 7 and 8 changed
lookup/access metadata. The corresponding direct duplicate-capable set
operations changed from hash joins in versions 1-4 to apply-family plans in
version 5, with `INTERSECT DISTINCT` changing again from `Cross Apply` to
`Semi Apply` in version 6. PUSH had the same version boundary for both
spellings: rejected in versions 1-2 and accepted from version 3.

Exploratory count-based ALL rewrites also exposed aggregate method and hash
build-side controls. The inner join in the `INTERSECT ALL` rewrite accepted
factorized execution, while the outer join in the `EXCEPT ALL` rewrite did
not. These probes are not retained in the 116-case selector or its expectation
manifest. The additional control comes with substantially more SQL and more
opportunities for NULL, count, overflow, and expansion mistakes. `EXISTS` or
`NOT EXISTS` alone cannot represent ALL multiplicities.

## Optimizer-version and environment matrices

The optimizer-version matrix was limited to versions 1 through 8 because the
pinned Spanner Omni runtime does not provide optimizer version 9.

| Version | Plans | Expected errors | Version-specific boundary |
| --- | ---: | ---: | --- |
| 1-2 | 89 each | 27 each | PUSH was rejected for direct and rewritten join-exposing operations; `HASH_JOIN_EXECUTION=ONE_PASS` was also rejected |
| 3 | 95 | 21 | PUSH was accepted; ONE_PASS remained rejected |
| 4 | 99 | 17 | ONE_PASS was accepted; only the invariant negative controls remained |
| 5-6 | 99 each | 17 each | same acceptance boundary as version 4; version 5 changed several default direct-operation join families |
| 7-8 | 99 each | 17 each | same acceptance boundary; several apply lookup plans changed `seekable_key_size` metadata |

Default physical choices changed at version 5. Unhinted `INTERSECT DISTINCT`
and `EXCEPT DISTINCT` moved from `BUILD_SEMI`/`BUILD_ANTI_SEMI` hash joins to
apply-family plans. Unhinted `EXCEPT ALL` moved from `Hash Join BUILD_OUTER` to
`Outer Apply`. The retained unhinted `INTERSECT ALL` stayed `Hash Join INNER`
through versions 1-8. Explicit join-method controls continued to select their
requested families wherever the hint value was accepted.

The compact-tree-metadata comparison found version-dependent shapes in 22
successful query labels. Most operator-family changes occurred at version 5.
One input-sensitive exception was the duplicate-capable `Albums INTERSECT
DISTINCT Songs` probe: version 5 used `Cross Apply`, version 6 selected `Semi
Apply`, and versions 7-8 retained that family while changing the Songs index
lookup's `seekable_key_size` from 2 to 0. Similar metadata-only version-7
changes appeared in three-input and ALL/apply probes. These are observed
optimizer choices, not semantic changes.

`allow_distributed_merge` default, true, and false produced byte-identical
plans for every successful probe and identical error classes for every
negative control. No set-operation effect was observed in this fixture.

Statement `SCAN_METHOD=ROW` and `SCAN_METHOD=BATCH` produced the corresponding
plan metadata on the controlled `INTERSECT DISTINCT` scan. By contrast,
statement `EXECUTION_METHOD=ROW` and `EXECUTION_METHOD=BATCH` around an
explicit batched APPLY probe produced byte-identical plans, as did
`HASH_JOIN_EXECUTION=ONE_PASS` and `MULTI_PASS` after the values became
accepted. Those pairs remain acceptance controls rather than plan-effect
contracts.

## DISTINCT results

`UNION DISTINCT` is aggregate-like in these plans: it is implemented by Hash
`Aggregate` nodes over `Union All`. `SELECT DISTINCT` is also aggregate-like
when duplicate elimination is needed. A forced ordered index produced Stream
`Aggregate` for `DISTINCT FirstName`; a forced base-table scan of non-key
`BirthDate` produced Hash `Aggregate`. A distinct primary-key projection was
optimized to a scan without an aggregate.

`GROUP_METHOD=HASH_GROUP` and `GROUP_METHOD=STREAM_GROUP` were rejected as
unsupported both at a set operation and directly after `SELECT`. There is no
evidence for a direct GROUP_METHOD control on DISTINCT.

`GROUPBY_SCAN_OPTIMIZATION=TRUE|FALSE` was accepted on `SELECT DISTINCT` as the
documentation predicts. In the ordered-index probes TRUE changed the scan
method metadata from `Automatic` to `Row`, but both values retained the Stream
Aggregate. That difference is not strong enough to encode a dedicated
optimization-effect contract: the hint remains an acceptance/control probe.

An equivalent semantic rewrite can expose the documented group hint:

```sql
SELECT FirstName
FROM Singers@{FORCE_INDEX=SingersByFirstLastName}
GROUP@{GROUP_METHOD=STREAM_GROUP} BY FirstName;
```

Likewise, `UNION DISTINCT` can be expressed as `UNION ALL` in a subquery plus
an outer hinted `GROUP BY`. HASH and STREAM selected the corresponding
aggregate iterators; STREAM also introduced Sort operators. This is an
actionable rewrite only when its duplicate and NULL semantics are confirmed
for the caller and the added sort cost is acceptable. It is not evidence that
GROUP_METHOD directly controls DISTINCT.

## Repository consequences

- The built-in `set_operation_distinct` case retains effects and negative
  controls.
- Positive planvocab expectations cover only plan-visible effects.
- Expected query errors are label-bound and substring-checked, so
  `--continue-on-error` cannot turn an unexpected rejection into a passing
  corpus run.
- `Generate Relation` admits its observed untyped scalar repeat-count link.
- Distributed semi/anti and push-broadcast semi/anti apply operators admit the
  observed `order_preserving=true` metadata; their two variable-bearing Batch
  links are covered by positive rewrite expectations.
- SQL-file loading uses lexical statement splitting. A strict parser must not
  prevent Omni from validating set-operation hint syntax that the pinned
  memefish version does not yet parse.
- The Omni integration test retains populated result equivalence for nullable
  multi-column DISTINCT rows and three-input mixed-method rewrites.
