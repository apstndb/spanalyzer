# Set-operation and DISTINCT hint observations (2026-08-04)

This note records environment-specific query-plan observations. It is not a
stable Spanner optimizer contract and does not claim managed Cloud Spanner
equivalence.

## Evidence

- Official query syntax source: `documents/docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax`
  from the Google Developer Knowledge API, updated 2026-07-22 and retrieved
  2026-08-04. The corresponding public page is
  <https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax>.
- Runtime: Spanner Omni
  `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta.2`, local
  image digest
  `sha256:115622065afefd267f9ef3ff1025e35d73a03f66a7335de8e051b393ebdcfacc`.
- API: `AnalyzeQuery(QueryMode=PLAN)` through `spanner-query-plan-shape`.
- Data: empty tables created from the built-in documentation schema.
- Exploratory scope: 59 submitted probes, 51 returned plans and 8 reached hint
  validation before returning an unsupported-hint error. The retained built-in
  matrix has 48 cases, including the same 8 negative controls.

The official source says that `UNION DISTINCT` applies duplicate elimination
after the union. It documents `JOIN_METHOD`, `BATCH_MODE`, and
`FORCE_JOIN_ORDER` as join hints. `GROUP_METHOD` is documented for `GROUP BY`,
not DISTINCT. The table hint `GROUPBY_SCAN_OPTIMIZATION` explicitly applies to
eligible `GROUP BY` and `SELECT DISTINCT` queries.

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
| `INTERSECT DISTINCT` | `FORCE_JOIN_ORDER=TRUE` | changed the two-input default from `Cross Apply` to `Semi Apply` |
| three-input `INTERSECT DISTINCT` | default versus statement `FORCE_JOIN_ORDER=TRUE` | default contained `Semi Apply` plus `Cross Apply`; forced order contained two `Semi Apply` nodes |
| `EXCEPT DISTINCT` | default | `Anti-Semi Apply` |
| `EXCEPT DISTINCT` | HASH / APPLY batch true / APPLY batch false / MERGE | respectively `Hash Join BUILD_ANTI_SEMI`, `Distributed Anti Semi Apply`, `Anti-Semi Apply`, and `Merge Join LEFT_ANTI_SEMI` |

`HASH_JOIN_BUILD_SIDE` on `INTERSECT DISTINCT` and `EXCEPT DISTINCT` reached
hint validation but returned `Unsupported hint: HASH_JOIN_BUILD_SIDE` for both
`BUILD_LEFT` and `BUILD_RIGHT`. It must not be treated as a supported
set-operation control merely because the general hash-join hint exists.

`INTERSECT ALL` used `Generate Relation` after computing a repeat count; HASH,
MERGE, and APPLY controls did not change the visible join family in this
fixture. `EXCEPT ALL` also used `Generate Relation`, while the comparison part
did change: default/APPLY used `Outer Apply`, HASH used `Hash Join BUILD_OUTER`,
and MERGE used `Merge Join LEFT_OUTER`. ALL and DISTINCT therefore need
separate plan contracts even when they share a set-operator spelling.

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
- `Generate Relation` admits its observed untyped scalar repeat-count link.
- `Distributed Anti Semi Apply` admits multiple variable-bearing Batch links.
- SQL-file loading uses lexical statement splitting. A strict parser must not
  prevent Omni from validating set-operation hint syntax that the pinned
  memefish version does not yet parse.
