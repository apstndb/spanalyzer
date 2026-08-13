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
- Official docs (retrieved 2026-08-12 with `dkcli`):
  [`IS_FIRST`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-gql-functions#is_first)
  and [Graph query best practices](https://docs.cloud.google.com/spanner/docs/graph/best-practices-tuning-queries).
- Schema: the built-in `MusicGraph` over `Singers` and `Collaborations`, plus
  `LabelGraph` over multi-labeled `Singers` and `Albums` nodes.
- Retained case: `gql_surface` with 120 queries and
  `testdata/gql_surface_expectations.json`.
- Revalidation: 97 plans, 23 expected query errors, 197 planvocab patterns,
  zero expectation failures, and zero vocabulary findings.
- Optimizer matrix: 960 submissions covering versions 1 through 8; every
  version returned 97 plans and the same 23 expected errors, with zero
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
- GQL `IS_FIRST(...) OVER (...)` in `RETURN`, direct `FILTER`, one-hop and
  quantified edge-valued membership, and ordered or unordered filtering
  between `NEXT` stages;
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

#### Column propagation mode lowering

The memefish propagation-mode work exposed a stronger plan probe than the
same-name `FULL` acceptance control. The retained `gql_set_propagation` case
uses two constant arms whose column sets overlap only at `shared`:

```sql
GRAPH MusicGraph
RETURN 1 AS left_only, 10 AS shared
<mode> UNION ALL
RETURN 2 AS right_only, 20 AS shared
```

The graph name is semantically inert here; `MusicGraph` lets the repository
harness reuse its existing schema, while the managed recheck used the existing
`FinGraph` database from the parser verification. The query does not read a
node, edge, or property in either environment.

Evidence gathered on 2026-08-12:

- Spanner Omni image
  `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta.2`, local
  digest
  `sha256:115622065afefd267f9ef3ff1025e35d73a03f66a7335de8e051b393ebdcfacc`,
  through `AnalyzeQuery(QueryMode=PLAN)`;
- managed Spanner through the dedicated user-level profile, with connection
  identifiers not recorded, using read-only `PLAN` requests and
  `CLI_EXPLAIN_PRINT_SECTIONS=full`; and
- the official [GQL set-operation
  documentation](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-query-statements#gql_set),
  retrieved in full with `dkcli` on 2026-08-12. Its grammar does not advertise
  propagation modifiers and describes the default behavior as requiring
  identical column-name sets.

The Japanese operator notes in `apstndb/spanner-hacks` were also checked at
commit `ed0c72b22e563af4113d98c0aa98cc784240f7ec`. They describe `Union Input`
slot mappings but contained no prior GQL propagation-mode observation.

| Spelling | Result columns | `Union Input` slots per arm | Typed NULLs in the raw plan |
| --- | --- | ---: | ---: |
| `FULL`, `FULL OUTER`, `OUTER` | `left_only`, `shared`, `right_only` | 3 | 2 |
| `INNER` | `shared` | 1 | 0 |
| `LEFT`, `LEFT OUTER` | `left_only`, `shared` | 2 | 1 |
| default (`STRICT`) | no result | no plan | no plan |

All accepted forms lowered to two `Union Input` arms under one `Union All`.
`FULL` synthesized `right_only=<typed null>` in the left arm and
`left_only=<typed null>` in the right arm. `INNER` pruned both non-shared
columns before the union. `LEFT` retained the left schema and synthesized
`left_only=<typed null>` in the right arm. The focused Omni integration test
also executes every form and verifies the corresponding result schema and
rows.

The three FULL spellings produced byte-equal managed plan renderings at the
default setting, and the two LEFT spellings did the same. On Omni their raw
query-plan protos were equal within each alias group at the default setting and
every explicit optimizer version from 1 through 8. The FULL, INNER, and LEFT
managed renderings were each byte-equal across versions 1 through 8 and to the
default rendering. The STRICT control was rejected before plan construction
because its two inputs have different column-name sets.

These exact shapes are environment-bound regression evidence, not a stable
optimizer or performance contract. In particular, a future optimizer may move
or eliminate `Compute` while preserving the same result schema and NULL
semantics. The durable semantic checks and raw evidence should therefore stay
together; a normalized topology digest alone cannot prove which output slot
received a typed NULL.

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

The [`IS_FIRST` function reference](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-gql-functions#is_first)
defines a boolean window result. Its result is non-deterministic when `ORDER
BY` is omitted or values tie, so the retained unordered `NEXT` case is
acceptance/PLAN coverage only; a separately retained ordered case is used for
result assertions.

In a `RETURN` expression, GQL `IS_FIRST(1) OVER (...)` returned a row `Crowd`
operator above a Sort in every optimizer version from 1 through 8. The same
`IS_FIRST` expression in ordinary SQL returned `Unsupported built-in function:
IS_FIRST.` The observed `Crowd` shape has one relational input, three untyped
scalar inputs without variables, and one variable-bearing scalar result;
plancontract normalizes it as the concrete `crowd` family.

Placement is semantically and physically significant. The populated-graph
integration test inserts five nodes and six ordered edges. `RETURN IS_FIRST`
keeps all six rows and marks three as true; a direct `FILTER` and a one-hop
edge-valued `IN` subquery both return those same three selected edges. The
direct and one-hop forms contain two `Crowd` operators in v1-v5 and three in
v6-v8. The quantified edge-valued form uses the documented
[`FILTER`-inside-edge predicate pattern](https://docs.cloud.google.com/spanner/docs/graph/best-practices-tuning-queries#limit-traversed-edges-to-improve-query-performance): it retains `Recursive Union` and Limit-based selection, with no `Crowd` in this v1-v8 matrix. On the fixture it reduces the unfiltered `{1,3}` traversal from ten result rows to three.

When `IS_FIRST` filters the intermediate result before a second `NEXT`
traversal, the unordered plan contains exactly two `Crowd` operators at every
optimizer version; the no-filter two-stage control contains none. The ordered
form contains at least two `Crowd` operators and reduces the populated control
from four rows to two before the second traversal. Thus, `Crowd` is a
placement-specific lowering, not a syntax-wide marker for every `IS_FIRST`
use.

#### Managed Spanner revalidation and Top-k lowering hypothesis

On 2026-08-12, the same `MusicGraph` schema and six-edge fixture used by the
Omni integration test were created in a managed Spanner database reached via
the dedicated user-level managed-probe profile. Connection identifiers are not
recorded. Read-only execution and `EXPLAIN` covered the six retained placements
at the default optimizer setting and explicit versions 1 through 8. Every
deterministic result multiset was identical across the eight versions:
`RETURN` kept all six edges and marked three true; direct and one-hop filters
returned the same three edges; quantified filtering reduced the ten-row
control to three; and the ordered `NEXT` filter reduced its four-row control to
two.

The managed plans establish an implementation boundary that the pinned Omni
plans do not expose. At versions 1 through 4, direct and one-hop filtering used
two `Crowd` operators, while the quantified edge predicate used no `Crowd` and
placed `Limit`, `Sort Limit`, and `Local Sort Limit` under the recursive map.
The current recursive tail key appeared in the same subtree's scan seek
condition. This is plan evidence for a per-tail, per-recursion-step Top-k
pushdown, not a measured performance result.

At managed versions 5 through 8, direct and one-hop filtering used one `Crowd`.
The quantified form retained `Recursive Union` but changed to one `Crowd` over
a `Sort`: it first formed the selected edge relation and then joined it to the
recursive spool with an equality filter. The default plan used this latter
family, but plan similarity does not establish which optimizer version the
service selected. This precomputed/decorrelated family differs from the pinned
Omni matrix, where the quantified form kept the recursive Limit-based lowering
and no `Crowd` through v8.

An `ORDER BY` ablation separates `Crowd` from sorting. In both `RETURN` and
filter positions, removing `ORDER BY` removed `Sort` while retaining `Crowd`.
Changing `IS_FIRST(1)` to `IS_FIRST(2)` retained the non-recursive
`Crowd`/`Sort` family, although the managed v4 quantified plan changed the
distributed Top-k pipeline around its Limits. The evidence therefore supports
the narrower working model that `Crowd` evaluates or carries `IS_FIRST`'s
partition state, `Sort` supplies requested order, and a predicate-only
`IS_FIRST` may be rewritten to ordinary Top-k operators when that selection can
be pushed into traversal. It does not support treating `Crowd` as an alias for
`Sort`, `Limit`, or `Sort Limit`, nor treating operator counts as portable
across runtimes.

Native `TABLESAMPLE RESERVOIR` used Random Id Assign plus Sort Limit, while
BERNOULLI used Random Id Assign plus Filter. `WITH WEIGHT` was rejected at all
versions. Ordered and unordered horizontal `ARRAY_AGG` both contained a Sort;
their stable distinction was the Sort's scalar links (ordered: Key only;
unordered control: Key and Value), not the presence of Sort itself.

Horizontal `STRING_AGG` repeated that ordered-versus-positional Sort-link
distinction at every version and lowered through `ARRAY_AGG` plus
`ARRAY_TO_STRING`. Horizontal `ARRAY_CONCAT_AGG([e.AlbumTitle] ORDER BY ...)`
used the common three-Aggregate recursive shell but added an `Array Subquery`,
two more `Array Unnest` nodes, and `Minor Sort`; it was not plan-equivalent to
the semantically aligned horizontal `ARRAY_AGG` control. Moving either
`STRING_AGG(e.AlbumTitle, ...)` or `ARRAY_CONCAT_AGG(e.AlbumTitle)` directly to
`RETURN` did not become a vertical aggregate: `e` is an array-valued group
variable there, and all v1-v8 probes failed with the same field-access error.

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
