# GoogleSQL rewriter surface observations (2026-08-12)

This note maps every public rewriter registered by
[`RegisterBuiltinRewriters`](https://github.com/google/googlesql/blob/0e7d7073ed0360be587a5efa0fa78abeee00f17b/googlesql/analyzer/all_rewriters.cc)
at the pinned `google/googlesql` revision to retained spanalyzer evidence. It
does not assume that every registered rewriter is a Spanner syntax feature.
Some rewriters consume internal resolved nodes, and others belong to GoogleSQL
features that Spanner does not expose.

## Evidence scope

- Runtime: Spanner Omni image
  `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`.
- API: `AnalyzeQuery(QueryMode=PLAN)` through `spanner-query-plan-shape`.
- Optimizer versions: 1 through 8.
- Current Spanner language inventory:
  [GoogleSQL overview](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/overview),
  [query syntax](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax),
  and [data types](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-types).
- Retained selector: `rewriter_surface`, 32 queries.
- Results at every optimizer version: 18 plans and 14 expected errors.
- Manifest: 52 positive operator expectations, zero failures, and zero
  plan-vocabulary findings.
- Completeness gate: `TestRewriterSurfaceCatalogIsComplete` fixes the complete
  32-name upstream set and rejects missing, extra, or dangling evidence labels.

PLAN is evidence about the physical plan returned after frontend analysis. It
does not expose a trace of which frontend rewriters ran. The classifications
below therefore distinguish direct plan/error evidence from representative
Spanner syntax whose attribution to one particular frontend rewrite remains
indirect.

## Complete registered-rewriter map

| Registered rewriter | Retained status | Evidence or boundary |
| --- | --- | --- |
| `REWRITE_AGGREGATION_THRESHOLD` | direct error | `WITH AGGREGATION_THRESHOLD` is rejected at `WITH` |
| `REWRITE_ANONYMIZATION` | direct error | both deprecated `WITH ANONYMIZATION` and current `WITH DIFFERENTIAL_PRIVACY` are rejected at `WITH` |
| `REWRITE_BUILTIN_FUNCTION_INLINER` | direct plans | array first/last/min/max/slice, indexed filter/transform, four ARRAY_INCLUDES forms, and exact DOT_PRODUCT |
| `REWRITE_FLATTEN` | direct error | explicit `FLATTEN` reports `Function not found` |
| `REWRITE_GENERALIZED_QUERY_STMT` | existing error | the generic pipe query reports `Pipe query syntax not supported` |
| `REWRITE_GROUPING_SET` | existing errors | GROUPING SETS, ROLLUP, and CUBE each retain their explicit unsupported boundary |
| `REWRITE_INLINE_SQL_FUNCTIONS` | not exposed | Spanner has no SQL scalar-function definition surface |
| `REWRITE_INLINE_SQL_TVFS` | not exposed | Spanner has no SQL table-function definition surface |
| `REWRITE_INLINE_SQL_UDAS` | not exposed | Spanner has no SQL aggregate-function definition surface |
| `REWRITE_INLINE_SQL_VIEWS` | direct plans | single and nested invoker-rights views are compared with their inline controls |
| `REWRITE_INSERT_DML_VALUES` | direct plan | two-row INSERT VALUES exposes Union All, two Union Inputs, and two Unit Relations below Apply Mutations |
| `REWRITE_IS_FIRST_IS_LAST_FUNCTION` | existing plan | documented GQL IS_FIRST is retained; Spanner does not expose IS_LAST |
| `REWRITE_LIKE_ANY_ALL` | direct error | LIKE ANY reaches the explicit unsupported feature error |
| `REWRITE_MATCH_RECOGNIZE_FUNCTION` | existing error | MATCH_RECOGNIZE is rejected at runtime syntax |
| `REWRITE_MEASURE_TYPE` | not exposed | Spanner has no measure-definition or AGG-over-MEASURE surface |
| `REWRITE_MULTIWAY_UNNEST` | direct error | two-argument UNNEST reports that exactly one argument is supported |
| `REWRITE_NULLIFERROR_FUNCTION` | direct error | NULLIFERROR reports an unsupported built-in function |
| `REWRITE_ORDER_BY_AND_LIMIT_IN_AGGREGATE` | direct plans | ARRAY_CONCAT_AGG and ARRAY_AGG with ORDER BY/LIMIT are retained |
| `REWRITE_PIPE_ASSERT` | direct error | the exact pipe ASSERT spelling is rejected at the initial pipe query |
| `REWRITE_PIPE_DESCRIBE` | direct error | the exact pipe DESCRIBE spelling is rejected at the initial pipe query |
| `REWRITE_PIPE_IF` | direct error | the exact pipe IF/subpipeline spelling is rejected at the initial pipe query |
| `REWRITE_PIVOT` | existing error | PIVOT retains its explicit unsupported boundary |
| `REWRITE_PROTO_MAP_FNS` | not exposed | Spanner proto support does not expose the GoogleSQL MAP type or proto-map functions |
| `REWRITE_QUANTIFIED_COMPARISONS` | direct error | `= ANY (subquery)` reaches the explicit quantified-comparison error |
| `REWRITE_ROW_TYPE` | existing plans, indirect attribution | SELECT AS VALUE in a subquery and UNNEST of array-of-struct retain row/value-table surfaces |
| `REWRITE_SUBPIPELINE_STMT` | direct error | pipe IF retains a concrete subpipeline statement before the outer pipe rejection |
| `REWRITE_TUMBLE_FUNCTION` | direct error | TUMBLE reports `Table-valued function not found` |
| `REWRITE_TYPEOF_FUNCTION` | direct error | TYPEOF reports `Function not found` |
| `REWRITE_UNPIVOT` | existing error | UNPIVOT retains its explicit unsupported boundary |
| `REWRITE_UPDATE_CONSTRUCTOR` | direct error | nested UPDATE over an ARRAY column reports that nested array updates are unsupported |
| `REWRITE_VARIADIC_FUNCTION_SIGNATURE_EXPANDER` | not exposed | the pinned expander targets MAP functions, and Spanner does not expose MAP |
| `REWRITE_WITH_EXPR` | existing plan | scalar WITH expression is retained by the GoogleSQL surface selector |

## Plan-visible lowerings

Several array functions have physical plans that closely reflect their
upstream SQL definitions:

- `ARRAY_FIRST` becomes a null/empty check ending in `ARRAY_AT_OFFSET(..., 0)`.
- `ARRAY_LAST` ends in `ARRAY_AT_ORDINAL(..., ARRAY_LENGTH(...))`.
- `ARRAY_MIN` and `ARRAY_MAX` contain Array Unnest, null Filter, Sort Limit,
  and a scalar subquery; MAX reverses the value sort.
- `ARRAY_SLICE` contains Array Unnest, an offset-range Filter, and an Array
  Subquery.
- the indexed ARRAY_FILTER and ARRAY_TRANSFORM overloads expose the offset as
  a second lambda input; only FILTER adds a relational Filter.
- ARRAY_INCLUDES value, lambda, and ANY forms lower through Array Unnest,
  Filter, and Aggregate. ARRAY_INCLUDES_ALL nests a per-value aggregate below
  Cross Apply and a final aggregate.
- `ARRAY_CONCAT_AGG(... ORDER BY ... LIMIT ...)` introduces the expected
  aggregate, Sort Limit, Array Unnest, and Minor Sort pipeline.

These shapes were invariant in the asserted topology across optimizer v1-v8.
The exact `DOT_PRODUCT` query is accepted at every version, but its scalar
description remains `DOT_PRODUCT(...)`; PLAN alone does not prove whether the
upstream SQL-function body was inlined before Spanner's physical planner.

Both invoker-view queries have the same compact relational/metadata tree as
their explicit inline controls in every optimizer version. This is consistent
with view inlining, but remains post-analysis plan equivalence rather than a
frontend rewrite trace.

## Interpretation

The upstream rewriter list explains a meaningful subset of PLAN structure,
especially array-function subqueries, aggregate modifier pipelines, scalar
WITH elimination, view expansion, value-table handling, and INSERT VALUES
normalization. It is not a complete physical-plan specification. Spanner can
apply additional logical and physical optimization after rewriting, preserve
a function name in the final scalar description, or reject a GoogleSQL feature
before the corresponding upstream rewrite is applicable.

Consequently, spanalyzer retains three separate facts:

1. the complete pinned upstream registry;
2. Spanner-supported PLAN or stable capability-error evidence; and
3. an explicit `not-exposed` classification for internal or unavailable
   surfaces instead of inventing a runtime probe that cannot exercise them.

## Reproduction

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case rewriter_surface \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/rewriter_surface_expectations.json
```

The focused integration test runs the same 32 surfaces at optimizer versions
1 through 8 and adds topology, view/control equality, and error-class checks
that cannot be represented by a presence-only manifest.
