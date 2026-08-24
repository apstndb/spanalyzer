---
type: Observation
title: Spanner Set Operations
description: SQL-standard multiplicity semantics and environment-scoped Spanner plan observations for UNION, INTERSECT, and EXCEPT with ALL and DISTINCT.
tags: [spanner, googlesql, set-operations, multiplicity, query-plan]
status: draft
sources:
  - id: spanner-query-syntax
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax
    title: Cloud Spanner GoogleSQL query syntax
    last_modified: 2026-08-06T00:00:00Z
  - id: spanner-operators
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/operators
    title: Cloud Spanner GoogleSQL operators
    last_modified: 2026-08-04T00:00:00Z
  - id: bigquery-operators
    resource: https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/operators#is_distinct_from_operator
    title: BigQuery GoogleSQL IS DISTINCT FROM operator
  - id: googlesql-set-operation-resolver
    resource: https://github.com/google/googlesql/blob/1f8aa333f4d6353cd3a64471fc83121df72df3f7/googlesql/analyzer/resolver_query.cc#L11711-L11714
    title: google/googlesql set-operation value-table resolver rule
  - id: sql86
    resource: https://www.govinfo.gov/content/pkg/GOVPUB-C13-34d692548be2e76a4af31ba0cf22c936/pdf/GOVPUB-C13-34d692548be2e76a4af31ba0cf22c936.pdf
    title: ANSI X3.135-1986 Database Language SQL
  - id: sql92
    resource: https://web.cecs.pdx.edu/~len/sql-92.pdf
    title: SQL-92 draft, Section 7.10 query expression
  - id: sql99-grammar
    resource: https://ronsavage.github.io/SQL/sql-99.bnf.html
    title: Public SQL:1999 grammar transcription
  - id: sql2023
    resource: https://www.iso.org/standard/76584.html
    title: ISO/IEC 9075-2:2023 SQL/Foundation
  - id: sql2023-feature-catalog
    resource: https://modern-sql.com/standard/2023
    title: Public SQL:2023 feature catalog and identifier migration notes
  - id: formal-sql-semantics
    resource: https://www.vldb.org/pvldb/vol11/p27-guagliardo.pdf
    title: A Formal Semantics of SQL Queries, Its Validation, and Applications
  - id: runtime-observation
    resource: ../../research/spanner-query-plan-shape/SET_OPERATION_DISTINCT_HINTS_2026-08-04.md
    title: spanalyzer set-operation and DISTINCT hint observations
  - id: runtime-probes
    resource: ../../tools/spanner-query-plan-shape/set_operation_distinct_cases.go
    title: spanalyzer set-operation runtime probe cases
  - id: plan-expectations
    resource: ../../tools/spanner-query-plan-shape/testdata/set_operation_distinct_expectations.json
    title: spanalyzer set-operation planvocab expectations
  - id: rewrite-equivalence-test
    resource: ../../tools/spanner-query-plan-shape/set_operation_equivalence_omni_test.go
    title: spanalyzer nullable rewrite result and plan-topology integration test
---

# Spanner Set Operations

Observed: 2026-08-11

Status: **logical semantics documented; physical-plan findings are
environment-scoped**. Spanner's public GoogleSQL reference defines all six
combinations of `UNION`, `INTERSECT`, and `EXCEPT` with `ALL` and `DISTINCT`,
including their duplicate-row multiplicities.[^spanner-query-syntax] The plan
shapes below were observed with `AnalyzeQuery(QueryMode=PLAN)` on a pinned
Spanner Omni runtime and are not a stable optimizer contract or evidence of
managed Cloud Spanner equivalence.[^runtime-observation]

## Evidence layers

This observation deliberately separates four layers that answer different
questions:

1. **SQL semantics** define which rows and multiplicities a correct
implementation must return.
2. **Spanner GoogleSQL syntax** defines which spellings, input types, and
   groupings Spanner accepts.
3. **Omni plan observations** show how one pinned optimizer version represented
   those semantics for one empty-schema fixture.
4. **Plan contracts** encode only selected, plan-visible observations for
   regression detection; they do not turn an optimizer choice into a public
   Spanner guarantee.

The word "set" is historical terminology. `ALL` variants operate on bags, or
multisets, because duplicate rows retain a count. `DISTINCT` variants reduce
each surviving row's count to one. A formal semantics of SQL can therefore
model the six operations using bag union, bag intersection, bag difference,
and duplicate elimination.[^formal-sql-semantics]

## Spanner syntax and type rules

Spanner requires the qualifier to be written explicitly:

```sql
left_query
{ UNION | INTERSECT | EXCEPT } { ALL | DISTINCT }
right_query
```

The inputs are matched by column position. They must have the same number of
columns, and each position must have a common supertype. Except for `UNION
ALL`, every result column type must support equality comparison. Output column
names come from the first input.[^spanner-query-syntax]

The same set operation and qualifier can be chained without parentheses and
is evaluated incrementally from left to right. Parentheses are required when
mixing operations or qualifiers at the same level, and they are semantically
important for non-associative cases such as `EXCEPT`.[^spanner-query-syntax]

### Value-table input direction

The current Spanner query-syntax page says that combining a value table with
a one-column ordinary table always produces a value table. The pinned
`google/googlesql` resolver instead makes the result a value table only when
the first input is a value table.[^spanner-query-syntax][^googlesql-set-operation-resolver]

Pinned Omni followed the first-input rule for all six set-operation and
qualifier combinations in optimizer versions 1 through 8: ordinary-first
queries returned an ordinary top-level result, while value-table-first queries
reached Spanner's top-level value-table restriction. Wrapping a
value-table-first operation in an ordinary outer `SELECT` made it executable.
This is a documentation/runtime divergence for the recorded Omni image, not a
managed Cloud Spanner contract.[^runtime-observation]

## Multiplicity semantics

For a row value `t`, let `m` be its multiplicity in the left input and `n` its
multiplicity in the right input. Spanner documents the following rules, which
match the SQL bag semantics specified by SQL-92.[^spanner-query-syntax][^sql92]

| Operation | Multiplicity of `t` in the result |
|---|---:|
| `UNION ALL` | `m + n` |
| `UNION DISTINCT` | `1` when `m + n > 0`, otherwise `0` |
| `INTERSECT ALL` | `min(m, n)` |
| `INTERSECT DISTINCT` | `1` when `m > 0` and `n > 0`, otherwise `0` |
| `EXCEPT ALL` | `max(m - n, 0)` |
| `EXCEPT DISTINCT` | `1` when `m > 0` and `n = 0`, otherwise `0` |

This table is the semantic contract. A query plan is one possible strategy for
producing these counts, not part of the result definition.

## Standardization history

The semantics predate their current feature identifiers:

| Milestone | Set-operation surface |
|---|---|
| SQL-86 | Standardized `UNION` with optional `ALL`; it did not yet standardize `INTERSECT` or `EXCEPT`.[^sql86] |
| SQL-92 | Added `INTERSECT` and `EXCEPT` and explicitly defined all six multiplicity rules. Its grammar used optional `ALL`; omitting it selected distinct semantics.[^sql92] |
| SQL:1999 | Exposed `ALL` and `DISTINCT` as explicit alternatives in the set-operator grammar.[^sql99-grammar] |
| SQL:2023 | Retains the six semantic variants. The current feature catalog classifies the `UNION` variants and `EXCEPT DISTINCT` as Core-family E071 features, and `INTERSECT DISTINCT`, `EXCEPT ALL`, and `INTERSECT ALL` as optional F303, F304, and F305 features.[^sql2023][^sql2023-feature-catalog] |

The important distinction is that SQL:1999's explicit `DISTINCT` spelling did
not invent distinct set semantics. SQL-92 already specified those semantics
when `ALL` was omitted.

## Relational interpretation

Let `R` and `S` be bags, `R ⊎ S` their bag union, and `δ` duplicate
elimination. The distinct variants can be described as:

- `UNION DISTINCT`: `δ(R ⊎ S)`;
- `INTERSECT DISTINCT`: rows of `R` that have a match in `S` under the set
  operation's row-equivalence relation, followed by duplicate elimination;
- `EXCEPT DISTINCT`: rows of `R` that have no such match in `S`, followed by
  duplicate elimination.

The latter two descriptions explain why a physical optimizer can use a
semi-join and anti-semi-join family. They are equivalences at the logical
level, not requirements to expose a particular operator name.

`INTERSECT ALL` and `EXCEPT ALL` cannot be implemented by an ordinary
semi-join or anti-semi-join alone: the result must emit a row exactly
`min(m,n)` or `max(m-n,0)` times. An implementation therefore needs to retain
or reconstruct multiplicity information.

## Observed Spanner Omni plan shapes

The retained probe used empty tables from the documentation schema. Its `ALL`
hint cases project non-unique columns from unrelated `Albums` and `Songs`
inputs so the optimizer cannot discharge the multiplicity comparison from a
parent/child schema relationship. It submitted 116 cases: 99 returned plans and
17 intentionally returned label-bound hint-validation errors.[^runtime-observation]

| Query family | Observed physical shape |
|---|---|
| `UNION ALL` | `Union All` |
| `UNION DISTINCT` | local/global Hash `Aggregate` above `Union All` |
| `INTERSECT DISTINCT` | apply, semi-apply, semi-hash-join, or semi-merge-join family depending on hints |
| `EXCEPT DISTINCT` | anti-semi apply, anti-semi hash join, or anti-semi merge join depending on hints |
| `INTERSECT ALL` | inner hash/merge join or apply family, then repeat-count computation and `Generate Relation` |
| `EXCEPT ALL` | outer hash/merge join or apply family, then repeat-count computation and `Generate Relation` |

These shapes are consistent with the multiplicity formulas:

- aggregate over `UNION ALL` implements duplicate elimination for `UNION
  DISTINCT`;
- semi and anti-semi families implement existence and non-existence tests for
  the distinct intersection and difference;
- `Generate Relation` expands a computed count back into repeated rows for the
  `ALL` variants.

The last point is an inference from the observed topology and scalar child
link. It is not a public definition of the `Generate Relation` operator.

## Hint observations

Spanner places a generic hint after the set operator and before `ALL` or
`DISTINCT`, for example:

```sql
SELECT SingerId FROM Singers
INTERSECT @{JOIN_METHOD=HASH_JOIN} DISTINCT
SELECT SingerId FROM Albums
```

The pinned Omni probes found the following boundaries:[^runtime-observation]

- `JOIN_METHOD` selected hash, apply, or merge implementations for
  all tested `INTERSECT` and `EXCEPT` qualifiers. PUSH selected distributed
  push-broadcast families where accepted.
- `FORCE_JOIN_ORDER=TRUE` changed the observed `INTERSECT DISTINCT` apply
  topology, including a three-input case.
- `UNION ALL` and `UNION DISTINCT` did not acquire a visible join-family
  change from join-method controls.
- `INTERSECT ALL` and `EXCEPT ALL` changed comparison families under explicit
  HASH, APPLY, MERGE, and PUSH controls while retaining `Generate Relation`.
- `HASH_JOIN_BUILD_SIDE` was rejected for both sides of all four tested
  `INTERSECT`/`EXCEPT` qualifier combinations and must not be inferred to work
  merely because a hash join appeared.
- `FACTORIZED_MODE=FACTORIZE_BOTH` was accepted for the inner-join-shaped
  `INTERSECT ALL` probe, but the DISTINCT variants and `EXCEPT ALL` rejected
  it.
- `GROUP_METHOD` was rejected both directly on `UNION DISTINCT` and directly
  after `SELECT`. Rewriting duplicate elimination as an explicit `GROUP BY`
  made the documented group hint applicable, but that is a query rewrite, not
  direct control of `DISTINCT`.
- `GROUPBY_SCAN_OPTIMIZATION` was accepted on eligible `SELECT DISTINCT`
  scans, as the public documentation states, but the observed plan difference
  was too weak to claim a dedicated optimization effect.

## Equivalent rewrites and hintability

Result equivalence was checked separately with duplicate-bearing and
NULL-bearing inputs whose corresponding columns already had the same types.
`INTERSECT DISTINCT` matched `SELECT DISTINCT` plus a correlated `EXISTS`, and
`EXCEPT DISTINCT` matched `SELECT DISTINCT` plus a correlated `NOT EXISTS`,
when their correlation predicate treated two NULLs as equal:

```sql
right_key IS NOT DISTINCT FROM left_key
```

Ordinary SQL equality is not sufficient for this rewrite because it evaluates
to unknown when either operand is NULL. `IS [NOT] DISTINCT FROM` is documented
on the BigQuery GoogleSQL operators page, including its NULL semantics, but the
current product-specific Spanner GoogleSQL operators page omits it. The
BigQuery page is syntax and semantics evidence for that product, not evidence
that Spanner accepts the spelling. Separately, the spanalyzer GoogleSQL
frontend and the pinned Omni runtime accepted both spellings, and Omni
constant-folded the two NULL comparisons to the expected Boolean values in
optimizer versions 1 through 8. A read-only managed Spanner probe on
2026-08-12 also returned `true` for `NULL IS NOT DISTINCT FROM NULL`, `false`
for `NULL IS DISTINCT FROM NULL`, `true` for two NaN values, and `true` for
positive zero versus negative zero. The retained nullable rewrite therefore
uses a native Spanner operator rather than a sibling-product spelling inferred
from documentation.[^bigquery-operators][^spanner-operators][^runtime-observation]

The retained Omni integration test extends this to two-column rows whose
columns can be independently NULL, duplicate input rows, and three-input
intersection and difference. The direct and rewritten results matched in all
four cases. For multiple columns, the rewrite must apply the NULL-safe
comparison to every output column and deduplicate the complete row.

The common-supertype conversion is part of the equivalence, not merely a type
checker detail. A managed Spanner counterexample used the exactly represented
`INT64` value `9007199254740993` on the left and its `FLOAT64` conversion on
the right. Direct `INTERSECT DISTINCT` returned a `FLOAT64` value rounded to
`9007199254740992`. A naive `EXISTS` rewrite compared the coercible operands
but projected the original left `INT64`, returning `9007199254740993` instead.
Casting both correlation operands and the projected left value to the set
operation's resolved common supertype restores equivalence. Rewrites must
therefore reproduce the resolved type of every output position as well as its
NULL-safe row comparison.

The rewrites had four observed control advantages over direct `INTERSECT
DISTINCT` and `EXCEPT DISTINCT`:

1. `HASH_JOIN_BUILD_SIDE=BUILD_LEFT|BUILD_RIGHT` was accepted on the correlated
   subquery hint and produced BUILD or PROBE semi/anti hash joins. The same
   hint was rejected directly on the set operations.
2. Replacing `SELECT DISTINCT` with an equivalent explicit `GROUP BY` allowed
   `GROUP_METHOD` to be combined with the join hint. Hash and Stream selected
   the requested aggregate iterator alongside hash or merge semi/anti joins.
3. On the duplicate-capable fixture, the direct plans had two deduplication
   pipelines, one for each input; the rewrites had one output deduplication
   pipeline and did not separately deduplicate the right input. The current
   plans represent each pipeline with two physical `Aggregate` nodes, so the
   retained integration test asserts direct count 4 and rewrite count 2 for
   both intersection and difference across optimizer versions 1 through 8.
4. A three-input rewrite attached `HASH_JOIN` plus `BUILD_LEFT` to one
   correlated predicate and non-batched `APPLY_JOIN` to the other. Both
   requested families appeared in one plan. The direct same-level chain
   rejected a hint on its second set operator because set-operation hints must
   appear on the first operation.

The checked Hash/Hash and Merge/Stream join-plus-group examples do not prove a
fully independent cross-product of the two hint axes. A minimal Hash/Stream
probe failed to return from `AnalyzeQuery` within two minutes and was
interrupted, so it is neither an acceptance result nor a performance result.

The unhinted rewrites kept the same semi/anti operator families from optimizer
versions 1 through 8, while the corresponding direct set-operation shapes
changed join family at versions 5 and 6. The rewrite plans were not byte-equal
throughout the matrix because versions 7 and 8 changed lookup/access metadata.
This is evidence of family stability and hint reach, not evidence that the
rewrites are faster. No execution metrics were measured.

`INTERSECT ALL` and `EXCEPT ALL` require count-aware rewrites rather than
`EXISTS` or `NOT EXISTS` alone. Grouping both inputs, joining their counts, and
expanding `min(m,n)` or `max(m-n,0)` rows reproduced the duplicate and NULL
results in the probe. These forms also exposed aggregate and hash build-side
hints, but their complexity and expansion risk make them an escape hatch, not
a default substitute for the direct set operators.

The same 116 cases were also run at each optimizer version from 1 through 8.
Versions 1-2 returned 89 plans and 27 expected errors each, version 3 returned
95/21, and versions 4-8 returned 99/17. The mixed-method multi-predicate
rewrite and the direct second-hint rejection were stable across all eight
versions. PUSH became accepted for these
join-exposing set operations at version 3 in this runtime, while
`HASH_JOIN_EXECUTION=ONE_PASS` became accepted at version 4. At version 5,
unhinted DISTINCT intersection/difference and unhinted `EXCEPT ALL` changed
from hash-join families to apply families. Version 9 was not tested because it
is unavailable in the pinned Spanner Omni runtime.[^runtime-observation]

The compact plan comparison found 22 successful labels with at least one
version-dependent shape. Besides the broad version-5 family change, the
duplicate-capable `Albums INTERSECT DISTINCT Songs` case changed from `Cross
Apply` in version 5 to `Semi Apply` in version 6. Several apply-based probes
then changed only `seekable_key_size` metadata at version 7. These are physical
plan observations, not changes to set-operation semantics.

An additional 348-probe matrix found no plan or error-class difference among
the default, true, and false values of `allow_distributed_merge` for this
fixture.[^runtime-observation]

`SCAN_METHOD=ROW|BATCH` was plan-visible on a controlled `INTERSECT DISTINCT`
scan and is covered by metadata expectations. The statement
`EXECUTION_METHOD` pair around batched APPLY and the accepted
`HASH_JOIN_EXECUTION=ONE_PASS|MULTI_PASS` pairs were byte-identical in this
fixture, so they remain acceptance controls rather than asserted effects.

## Testing implications

Result semantics and plan structure should be tested separately:

1. Result tests should use duplicate-bearing and `NULL`-bearing inputs and
   assert the six multiplicity formulas.
2. Plan tests should use [`planvocab`](../../plancontract/planvocab) or plan
   contracts to assert only the operator metadata and child links that matter
   to the test.
3. `ALL` and `DISTINCT` need separate plan contracts even when they share the
   same set-operator name.
4. Hint-effect tests need an unhinted baseline and negative controls; syntax
   acceptance alone is not evidence of plan effect.
5. Unknown operator, metadata, or child-link vocabulary should remain visible
   as drift rather than being silently admitted to the catalog.
6. Expected failures must be label-bound and message-checked; a general
   continue-on-error switch is not a sufficient test assertion.

The executable case matrix and current plan expectations remain canonical in
the [runtime probes](../../tools/spanner-query-plan-shape/set_operation_distinct_cases.go)
and [planvocab expectation manifest](../../tools/spanner-query-plan-shape/testdata/set_operation_distinct_expectations.json).[^runtime-probes][^plan-expectations]

## Verification limits

- The physical-plan evidence is from the pinned Omni environment recorded in
  the source note. Managed Spanner was used only for the explicitly identified
  result-semantics checks above, not to generalize the Omni plan shapes.
- The plan probes used `PLAN` against empty tables. They establish optimizer
  topology, not runtime cardinalities, costs, or performance.
- A separate duplicate-bearing result fixture exercised the representative
  integer/NULL cases above. It does not establish equivalence for every
  Spanner type, collation, or composite-row edge case.
- Operator names, metadata values, link types, and default plan choices may
  change across optimizer or runtime versions.
- Equality behavior for every Spanner type, collation, and `NULL` combination
  was not exhaustively exercised by this probe family.

[^spanner-query-syntax]: Cloud Spanner GoogleSQL query syntax.
[^spanner-operators]: Cloud Spanner GoogleSQL operators; retrieved 2026-08-11. The page omitted `IS [NOT] DISTINCT FROM` at that time.
[^bigquery-operators]: BigQuery GoogleSQL `IS DISTINCT FROM` operator; retrieved 2026-08-11. This sibling product reference is not Spanner support evidence.
[^googlesql-set-operation-resolver]: `google/googlesql` set-operation resolver at commit `1f8aa333f4d6353cd3a64471fc83121df72df3f7`; the source comment defines the first-input value-table rule.
[^sql86]: ANSI X3.135-1986 Database Language SQL.
[^sql92]: SQL-92 draft, Section 7.10 query expression.
[^sql99-grammar]: Public SQL:1999 grammar transcription.
[^sql2023]: ISO/IEC 9075-2:2023 SQL/Foundation.
[^sql2023-feature-catalog]: Public SQL:2023 feature catalog and identifier migration notes.
[^formal-sql-semantics]: Guagliardo and Libkin, “A Formal Semantics of SQL Queries, Its Validation, and Applications,” PVLDB 11(1), 2017.
[^runtime-observation]: spanalyzer set-operation and DISTINCT hint observations.
[^runtime-probes]: spanalyzer set-operation runtime probe cases.
[^plan-expectations]: spanalyzer set-operation planvocab expectations.
