# GQL surface plan observations (2026-08-11)

This note records environment-specific acceptance and query-plan evidence for
Spanner GQL and SQL/GQL bridge syntax. It is not a stable optimizer contract.

## Evidence environment

- Runtime: Spanner Omni
  `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`, local
  image digest
  `sha256:e98a088fa66d4a87dbb560d729bf21d998bb843f6018bd8dc118fe320e671886`.
- API: `AnalyzeQuery(QueryMode=PLAN)` through
  `spanner-query-plan-shape`.
- Schema: the built-in `MusicGraph` over `Singers` and `Collaborations`, plus
  `LabelGraph` over multi-labeled `Singers` and `Albums` nodes.
- Retained case: `gql_surface` with 117 queries and
  `testdata/gql_surface_expectations.json`.
- Revalidation: 94 plans, 23 expected query errors, 191 planvocab patterns,
  zero expectation failures, and zero vocabulary findings.
- Optimizer matrix: 936 submissions covering versions 1 through 8; every
  version returned 94 plans and the same 23 expected errors, with zero
  vocabulary findings.

## Retained accepted surface

The broad case deliberately includes accepted clauses that optimize to a
simpler physical shape. A plan need not expose a clause-specific operator to
serve as runtime acceptance evidence.

- SQL `GRAPH_TABLE`, including graph-element property access, `SAFE_TO_JSON`,
  filtering, and an eliminated self-join control;
- GQL `FILTER`, `FILTER WHERE`, `LET`, constant and correlated `FOR`, and
  `NEXT`;
- correlated `CALL`, `OPTIONAL CALL`, and uncorrelated `CALL ()`;
- correlated GQL `EXISTS`, `VALUE`, `ARRAY`, `IN`, and `NOT IN` subqueries,
  including the short MATCH-body and graph-pattern-body EXISTS forms and
  predicate-position graph-pattern/full-query controls;
- `WITH DISTINCT` and traversal `RETURN DISTINCT`;
- nontrivial GQL `UNION ALL`, `UNION DISTINCT`, `INTERSECT DISTINCT`,
  `EXCEPT DISTINCT`, `INTERSECT ALL`, and `EXCEPT ALL`, including reordered
  name-aligned columns and same-kind three-arm chains;
- `ANY`, `ANY SHORTEST`, `ALL`, `ANY CHEAPEST`, `WALK`, `ACYCLIC`, and
  `TRAIL` path selection/mode prefixes;
- fixed, constructed, concatenated, zero-hop, multi-quantifier, and recursive
  path values with `PATH_LENGTH`, `IS_ACYCLIC`, `IS_TRAIL`, `IS_SIMPLE`,
  `NODES`, `EDGES`, `PATH_FIRST`, and `PATH_LAST`;
- outgoing, reverse, any-direction, edge-only, wildcard, endpoint-predicate,
  and compound label OR/AND graph patterns, plus graph element metadata
  functions and predicates;
- GQL `IS_FIRST(...) OVER (...)`, including filtering between `NEXT` stages
  and edge-valued membership inside a quantified traversal;
- graph traversal `FACTORIZED_MODE`, both SQL/`GRAPH_TABLE` and GQL-native
  `TABLESAMPLE BERNOULLI`/`RESERVOIR`, ordered and unordered horizontal
  `ARRAY_AGG`, horizontal `COUNT(DISTINCT ...)`, and property-dependent
  `ANY CHEAPEST` cost;
- `SKIP`, `OFFSET`, primitive and RETURN-qualified `ORDER BY ... LIMIT`, GQL
  `COLLATE`, post-aggregate `FILTER`, implicit grouping, and nontrivial
  aggregate `WITH`; and
- inner `MATCH` and `OPTIONAL MATCH` controls; and
- read-write `UPDATE` whose predicate contains a recursive GQL `IN` subquery.

The baseline `GRAPH_TABLE` node projection used the covering
`SingersByFirstLastName` index rather than a base-table scan. `FILTER` and
`FILTER WHERE` returned the same filtered-scan shape. `NEXT` and
`WITH DISTINCT` over an already unique node key optimized to the same scan as
the simpler control.

## GQL-specific contrasts

### `CALL`, `OPTIONAL CALL`, and `OPTIONAL MATCH`

- correlated `CALL` exposed a Batch `Distributed Cross Apply`;
- correlated `OPTIONAL CALL` exposed `Distributed Outer Apply` and
  `Compute Struct`; and
- uncorrelated `CALL ()` exposed a row `Cross Apply` over a scalar Stream
  Aggregate; and
- `OPTIONAL MATCH` exposed a `Hash Join` with `join_type=BUILD_OUTER`,
  variable-bearing Build/Probe links, `BloomFilterBuild`, and
  `Compute Struct`.

The inner `MATCH` control used distributed cross-apply lowering and did not
contain that outer Hash Join. The `OPTIONAL MATCH` raw `plan_nodes` were
byte-equal across versions 1 through 8 in this probe.

The correlated `EXISTS` probe used a scalar Stream Aggregate rather than a
semi-join display name. The correlated `VALUE` probe used Cross Apply plus
global/local Limits. An inner graph element must use a different variable and
an explicit equality filter; redeclaring the outer element name is rejected.

The MATCH-body and graph-pattern-body short EXISTS spellings were byte-equal
to each other at every optimizer version. Their Aggregate count changed from
three in v1-v4 to one in v5 and two in v6-v8, so the focused test records the
optimizer boundary even though the two spellings normalize identically.

The predicate-position graph-pattern `EXISTS` shorthand and its full
`EXISTS { MATCH ... RETURN ... }` control were also byte-equal at every
optimizer version. Both exposed one row `Distributed Semi Apply` and one
`Compute Struct`, so the retained pair proves normalization rather than a
shorthand-specific physical family.

The correlated `ARRAY` form exposed an `Array Subquery` plus Batch
`Distributed Cross Apply`. Predicate-position `IN` exposed `Distributed Semi
Apply`, while `NOT IN` changed the outer family to `Distributed Anti Semi
Apply`; both retained an inner `Semi Apply`. These shapes and their link kinds
were stable at optimizer versions 1 through 8.

Passing a graph element through `GRAPH_TABLE` was accepted when outer SQL read
its properties or converted it with `SAFE_TO_JSON`. Directly selecting the
element remained rejected. Property-only access used the covering index;
JSON conversion constructed the full element and read the base table. A
correlated `FOR ... IN GENERATE_ARRAY(1, p.SingerId)` exposed a row Cross
Apply and Array Unnest, unlike the retained constant-array spelling.

### Set operations

The retained arms are intentionally non-identical: the left arm projects
possibly duplicated collaboration destinations, while the right arm projects
positive singer IDs. Both logical inputs remained visible in every plan.

| Operation | Observed current plan boundary |
| --- | --- |
| `UNION ALL` | `Union All` and both scan families |
| `UNION DISTINCT` | `Union All` below a Hash `Aggregate` |
| `INTERSECT DISTINCT` | Hash Aggregates plus Batch distributed apply; both scans retained |
| `EXCEPT DISTINCT` | `Hash Join PROBE_ANTI_SEMI`; both scans retained |
| `INTERSECT ALL` | count Aggregates, inner Hash Join, and `Generate Relation` |
| `EXCEPT ALL` | count Aggregates, outer Hash Join, and `Generate Relation` |

The GQL ALL forms use the same count-and-expand idea as the ordinary SQL
forms. `INTERSECT ALL` kept `join_type=INNER` throughout versions 1 through 8.
`EXCEPT ALL` changed from `BUILD_OUTER` in versions 1 through 4 to
`PROBE_OUTER` in versions 5 through 8, while retaining five Aggregates and one
`Generate Relation` in this probe.

This avoids the earlier identical-arm controls, where intersection collapsed
to one scan and difference collapsed to `Empty Relation`.

GQL aligns set-operation columns by name. A two-column INTERSECT with reversed
RETURN order and a same-kind three-arm UNION ALL both returned plans. The
grammar-only `FULL UNION ALL` spelling also returned a stable `Union All` plan
in the pinned Omni runtime at versions 1 through 8, although the normal Spanner
GQL documentation does not advertise `FULL`; treat it as an Omni observation,
not a portable syntax promise. Mixed UNION/EXCEPT chains were rejected with
the documented same-operation-type rule.

### Recursive and path forms

`ANY SHORTEST` and constant-cost `ANY CHEAPEST` exposed `Recursive Union`,
`Recursive Spool Scan`, and `VerifyDeterminism`. `WALK`, `ACYCLIC`, and
`TRAIL` were all accepted, but their tested fixed-length paths lowered to
generic apply/filter machinery without a mode-specific operator or metadata
key. The broad suite retains them as acceptance evidence without claiming
that planvocab can distinguish all three modes.

The recursive path-materialization case forced node and edge arrays through
`NODES`, `EDGES`, `PATH_FIRST`, and `PATH_LAST`; it exposed additional
`Compute` scalar links and is now part of the vocabulary gate. Bounded `ANY`
and `ALL`, and a property-dependent `ANY CHEAPEST` cost, all returned recursive
plans throughout versions 1 through 8.

The expanded matrix also retained a quantified subpath-local `WHERE`, `{0,3}`
zero-hop traversal, two quantified segments, and the valid `ANY SHORTEST
(TRAIL subpath)` combination. The unbounded `+` form parsed but was rejected as
unsupported in every tested optimizer version.

### Compound label expressions

`LabelGraph` gives the `Singers` node definition both `Singer` and `Person`
labels and gives `Albums` the `Album` label. `MATCH (n:(Singer|Album))`
therefore remains semantically non-vacuous and exposes one `Union All` over
the `Singers` and `Albums` TableScans at every optimizer version. The
conjunction `MATCH (n:(Singer&Person))` is byte-identical to its exact
single-label `MATCH (n:Singer)` control at v1-v8, showing static label
simplification to the same `SingersByFirstLastName` covering IndexScan rather
than a distinct physical family. The malformed
`(Singer|)` control reaches a stable syntax error.

### GQL analytic and graph factorization

GQL `IS_FIRST(1) OVER (...)` returned a row `Crowd` operator above a Sort in
every optimizer version from 1 through 8. The same `IS_FIRST` expression in
ordinary SQL returned `Unsupported built-in function: IS_FIRST.` The observed
`Crowd` shape has one relational input, three untyped scalar inputs without
variables, and one variable-bearing scalar result; plancontract normalizes it
as the concrete `crowd` family.

When `IS_FIRST` filtered the intermediate result before a second `NEXT`
traversal, the plan contained exactly two Crowd operators at every optimizer
version; the no-filter two-stage control contained none. An edge-valued `IN`
subquery containing `IS_FIRST` inside a quantified traversal retained the
Recursive Union and added Limit-based selection beneath it.

Native `TABLESAMPLE RESERVOIR` used Random Id Assign plus Sort Limit, while
BERNOULLI used Random Id Assign plus Filter. `WITH WEIGHT` was rejected at all
versions. Ordered and unordered horizontal `ARRAY_AGG` both contained a Sort;
their stable distinction was the Sort's scalar links (ordered: Key only;
unordered control: Key and Value), not the presence of Sort itself.

Primitive `ORDER BY ... LIMIT` used two Sort Limit operators and no separate
Limit in v1-v2, then one Sort Limit plus one Limit in v3-v8. Adding GQL
`COLLATE` preserved that boundary. Post-aggregate `FILTER` adds one Filter to
the two-Aggregate grouping plan; implicit grouping retains the same two
Aggregate families without an explicit `GROUP BY`. Horizontal
`COUNT(DISTINCT e.AlbumTitle)` adds a third Aggregate relative to the plain
COUNT control while keeping the recursive Array Unnest machinery.

The read-write bridge case places a recursive graph `IN` subquery inside an
ordinary SQL `UPDATE` predicate. It retained `Apply Mutations` metadata
`operation_type=UPDATE, table=Singers` together with `Recursive Union`,
`Recursive Spool Scan`, and Batch `Distributed Cross Apply` at every optimizer
version. This is direct cross-family topology evidence rather than an
assumption from separately tested DML and graph queries.

The quantified traversal with `FACTORIZED_MODE=FACTORIZE_LEFT` was accepted in
all eight versions. Versions 1 through 4 had the same Aggregate/Array-Unnest
counts as the unhinted control; versions 5 through 8 added one Aggregate and
one Array Unnest. This repeats the ordinary-join v4/v5 plan-visibility boundary
on a graph traversal rather than merely assuming that the SQL result carries
over to GQL.

## Capability boundaries

The following results were invariant at optimizer versions 1 through 8:

- GQL `ROW_NUMBER()` returned `Unsupported built-in function: ROW_NUMBER.`;
- ordinary SQL `IS_FIRST()` returned `Unsupported built-in function: IS_FIRST.`;
- `CALL PageRank()` returned that graph algorithm TVFs must be used inside
  `EXPORT DATA` in scale-up execution mode;
- GQL-native `QUALIFY` was rejected at the grammar position after `RETURN`;
- ordinary SQL parsed `QUALIFY` but returned `QUALIFY is not supported`;
- counted `ANY` and `SHORTEST` search prefixes parsed but returned a
  path-count capability error;
- `ALL SHORTEST` and `SIMPLE PATH` returned explicit unimplemented errors;
  and
- combining a path search prefix with a path mode parsed but returned an
  invalid-combination error;
- GQL-native `TABLESAMPLE SYSTEM`, unbounded quantifiers, `ALL CHEAPEST`, and
  counted `CHEAPEST` returned stable capability errors;
- GQL `TABLESAMPLE ... REPEATABLE (...)` returned the stable
  `REPEATABLE TABLESAMPLE is not supported` boundary;
- `TABLESAMPLE ... WITH WEIGHT` returned a stable unsupported error;
- `CALL PER () PageRank()` still required `EXPORT DATA`, while `OPTIONAL CALL
  PER ()` was rejected by its documented restriction;
- returning a final `GRAPH_ELEMENT`, including through outer SQL over
  `GRAPH_TABLE`, and mixing set-operation kinds were rejected; and
- `AnalyzeQuery(PLAN)` recognized the `EXPORT DATA` graph-algorithm form but
  rejected the fixture's non-interleaved edge-table shape rather than
  returning a standard QueryPlan.

The SQL control shows that the absence of a QUALIFY plan is not specific to
`GRAPH_TABLE`. GQL analytic-function support and GQL-native QUALIFY placement
are separate boundaries, so the suite retains separate expected errors.

## Reproduction

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case gql_surface \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/gql_surface_expectations.json
```

The expectation manifest intentionally pins only the default-plan snapshot.
Add `--optimizer-version-matrix` without the unprefixed expectation manifest
for the 936-submission vocabulary matrix. A positive operator pattern proves
presence, not uniqueness or absence. PLAN evidence alone does not establish
execution cost or result-equivalence for the accepted clauses.
