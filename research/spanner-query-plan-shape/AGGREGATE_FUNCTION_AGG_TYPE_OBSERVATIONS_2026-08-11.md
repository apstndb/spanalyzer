# Aggregate function and physical Agg type observations (2026-08-11)

This note compares the aggregate functions documented for Spanner GoogleSQL
with the scalar expressions attached to an `Aggregate` operator by child links
whose `type` is `Agg`. An `Agg` link is a physical-plan role, not a public enum
of SQL aggregate functions. Its child scalar node can expose an internal
partial-aggregation function rather than the function spelling in the query.

## Evidence environment

- Runtime: Spanner Omni
  `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`, local
  image digest
  `sha256:e98a088fa66d4a87dbb560d729bf21d998bb843f6018bd8dc118fe320e671886`.
- API: `AnalyzeQuery(QueryMode=PLAN)` through
  `spanner-query-plan-shape`.
- Official Spanner documentation mirror:
  `apstndb/spanner-docs-mirror` commit
  `3ae52a8eb15d2d6952da6f387605ff2bc89d2720` (2026-08-10).
- Retained selector: `aggregate_functions`, 31 queries.
- Default result: 29 plans, 2 expected errors, and 62 operator expectations.
- Optimizer matrix: 248 submissions across versions 1 through 8, producing
  232 plans and 16 expected errors.

The documentation lists 19 general and statistical aggregate-function names.
All 19 are represented by positive probes. Additional probes cover `COUNT(*)`,
aggregate-call modifiers, grouping without an aggregate function, and two
unsupported names that are available in broader GoogleSQL environments but
are not listed as Spanner aggregate functions.

## Documented function to physical Agg mapping

An ungrouped scalar aggregate over `Songs` normally has a local and a global
`Aggregate`. The table records the top-level function name in each `Agg`
child's scalar description; operands and generated variable names are omitted.

| SQL function | Local `Agg` type | Global `Agg` type | Additional finalization |
| --- | --- | --- | --- |
| `ANY_VALUE` | `ANY` | `ANY` | none |
| `ARRAY_AGG` | `ARRAY_AGG` | `ARRAY_CONCAT_AGG` | none |
| `ARRAY_CONCAT_AGG` | `ARRAY_CONCAT_AGG` | `ARRAY_CONCAT_AGG` | none |
| `AVG` | `AVG_START` | `AVG_FINAL` | none |
| `BIT_AND` | `BIT_AND` | `BIT_AND` | none |
| `BIT_OR` | `BIT_OR` | `BIT_OR` | none |
| `BIT_XOR` | `BIT_XOR` | `BIT_XOR` | none |
| `COUNT(expression)` | `COUNT` | `COUNT_FINAL` | none |
| `COUNTIF` | `COUNTIF` | `COUNT_FINAL` | none |
| `LOGICAL_AND` | `LOGICAL_AND` | `LOGICAL_AND` | none |
| `LOGICAL_OR` | `LOGICAL_OR` | `LOGICAL_OR` | none |
| `MAX` | `MAX` | `MAX` | none |
| `MIN` | `MIN` | `MIN` | none |
| `STDDEV` | `STDDEV_START` | `STDDEV_FINAL` | none |
| `STDDEV_SAMP` | `STDDEV_START` | `STDDEV_FINAL` | none |
| `STRING_AGG` | `STRING_AGG` | `STRING_AGG` | none |
| `SUM` | `SUM` | `SUM` | none |
| `VAR_SAMP` | `STDDEV_START` | `STDDEV_FINAL` | non-`Agg` `POW(result, 2)` |
| `VARIANCE` | `STDDEV_START` | `STDDEV_FINAL` | non-`Agg` `POW(result, 2)` |

This gives several boundaries that cannot be inferred from the SQL function
list alone:

- `ANY_VALUE` is rendered as `ANY`.
- decomposable aggregates can expose explicit partial and final types:
  `AVG_START` / `AVG_FINAL`, `COUNT_FINAL`, and `STDDEV_START` /
  `STDDEV_FINAL`;
- the global combiner for `ARRAY_AGG` is `ARRAY_CONCAT_AGG`; and
- variance is implemented from the standard-deviation aggregate state and is
  squared by a scalar `POW` node outside the Aggregate operator.

`STDDEV` and `STDDEV_SAMP` produced byte-identical QueryPlan protos at every
optimizer version from 1 through 8. `VAR_SAMP` and `VARIANCE` did the same.
These observations agree with the documented alias relationships, while also
showing their physical implementation.

## Modifiers and Aggregate operators without Agg expressions

| SQL form | Observed physical boundary |
| --- | --- |
| `ANY_VALUE(value HAVING MAX key)` | local `HAVING_MAX` plus `MAX`; global `HAVING_MAX`; the nested expression contains `ANY` |
| `ANY_VALUE(value HAVING MIN key)` | local `HAVING_MIN` plus `MIN`; global `HAVING_MIN`; the nested expression contains `ANY` |
| `COUNT(*)` | local `COUNT`, global `COUNT_FINAL` |
| `COUNT(DISTINCT value)` | local/global key-only dedup Aggregates, then a scalar Aggregate with `COUNT` |
| `AVG(DISTINCT value)` | local/global hash dedup Aggregates, then a scalar Aggregate with `AVG`, not `AVG_START` / `AVG_FINAL` |
| `ARRAY_AGG(DISTINCT ... IGNORE NULLS ORDER BY ... LIMIT ...)` | local `ARRAY_AGG`, global `ARRAY_CONCAT_AGG`, key-only dedup Aggregate, then `NEST` |
| `STRING_AGG(DISTINCT ... ORDER BY ... LIMIT ...)` | the same `ARRAY_AGG` / `ARRAY_CONCAT_AGG` / `NEST` pipeline, then non-`Agg` `ARRAY_TO_STRING` |
| grouped `COUNT(*)`, `SUM(value)` | one Stream Aggregate with `COUNT` and `SUM` Agg links in this primary-key grouping probe |
| `GROUP BY` without an aggregate function | Aggregate with `Key` but no `Agg` child |
| `SELECT DISTINCT` | local/global Aggregates with `Key` but no `Agg` child |

Consequently, neither direction is one-to-one: a SQL aggregate function can
produce several physical `Agg` types, and an Aggregate operator can have zero
`Agg` links because it is performing grouping or duplicate elimination.
Modifier lowering can also move essential finalization outside the Aggregate
operator, as shown by variance and distinct ordered `STRING_AGG`.

The top-level `Agg` type multisets above were stable across optimizer versions
1 through 8. The distinct ordered array/string pipelines changed an auxiliary
dedup Aggregate from Stream to Hash at version 6, and `ANY_VALUE` changed a
generated variable name at version 6, but neither change altered the `Agg`
type mapping.

## Managed Spanner `ANY_VALUE` row versus STRUCT recheck

A read-only managed Spanner probe on 2026-08-12 compared three independent
calls, `ANY_VALUE(a HAVING MAX d)`, `ANY_VALUE(b HAVING MAX d)`, and
`ANY_VALUE(c HAVING MAX d)`, with `ANY_VALUE(STRUCT(a, b, c) HAVING MAX d).*`.
On the populated table used for the probe, both returned the same row. This
data had no observed tie at the maximum key, so the result comparison does not
prove that independent calls select a coherent source row when ties exist; the
STRUCT form expresses that requirement directly.

The compact PLAN topology was identical for the two spellings at optimizer
versions 8 and 9: `Serialize Result`, Global Stream `Aggregate`, `Distributed
Union`, Local Stream `Aggregate`, Local Distributed Union, and an index scan.
Thus there was no managed-plan evidence of an operator-count or distribution
advantage for either spelling. A later attempt to re-fetch the raw Aggregate
nodes encountered an endpoint-wide `DeadlineExceeded`, including for
`SELECT 1`; detailed `Agg` expression cardinality therefore remains
unverified on managed Spanner rather than being inferred from the compact
tree.

## Unsupported controls

Pinned Omni returned stable `Unimplemented` errors at every optimizer version
from 1 through 8 for:

- `APPROX_COUNT_DISTINCT`: `Unsupported aggregate function:
  approx_count_distinct`; and
- `CORR`: `Unsupported aggregate function: corr`.

Neither name appears in the pinned Spanner aggregate-function reference.
Their availability in other GoogleSQL products must not be treated as Spanner
support.

## Reproduction

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case aggregate_functions \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/aggregate_function_expectations.json
```

Add `--optimizer-version-matrix` without the unprefixed expectation manifest
to repeat the 248-submission vocabulary matrix. The focused integration test
is the authority for the `Agg` expression names, alias proto equality, and
version stability because the plan-vocabulary manifest intentionally checks
operator/link structure rather than free-form scalar descriptions.
