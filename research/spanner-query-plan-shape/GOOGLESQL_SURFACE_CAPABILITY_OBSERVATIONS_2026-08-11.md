# GoogleSQL surface plan and capability observations (2026-08-11)

This note records accepted QueryPlan shapes and explicit runtime boundaries for
generic GoogleSQL grammar surfaces not otherwise distinguished by the
operator-oriented suites. Frontend acceptance alone is not Spanner PLAN
evidence.

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
- Upstream grammar, AST, resolver, and tests: `google/googlesql` commit
  `1f8aa333f4d6353cd3a64471fc83121df72df3f7` (2026-07-21).
- Emulator cross-check: `GoogleCloudPlatform/cloud-spanner-emulator` commit
  `2abe04c1ace40b8760133e6ca0800d257d127da4` (2026-08-03).
- Retained selector: `google_sql_surface`, 57 queries.
- Default result: 34 plans, 23 expected errors, 71 operator
  expectations, zero expectation failures, and zero vocabulary findings.
- Optimizer matrix: 456 submissions covering versions 1 through 8; every
  version returned the same 34 plans and 23 error classes. The matrix returned
  272 plans and 184 expected errors in total, with zero vocabulary findings.
- Descriptor-backed selector: `google_sql_proto_surface`, 18 queries. Its
  default result is 11 plans, 7 expected errors, and 28 operator expectations;
  its optimizer matrix is 144 submissions (88 plans and 56 errors). Both have
  zero expectation failures and zero vocabulary findings after admitting the
  observed `Proto Constructor` contract.

## Accepted surfaces

| Surface | Retained plan evidence |
| --- | --- |
| `HAVING` | Aggregate plus Filter condition |
| `SELECT ALL` | Scan; no ALL-specific operator |
| `SELECT expression.* EXCEPT (...)` | base-table Scan |
| `SELECT * REPLACE (...)` | scalar Function plus base-table Scan |
| `SELECT AS VALUE` inside a subquery | covering IndexScan |
| `FOR UPDATE` in a read-write transaction | Filter Scan and base-table Scan |
| aggregate-call `HAVING MAX` | Stream Aggregate with Key/Agg links |
| aggregate-call `HAVING MIN` | Stream Aggregate with Key/Agg links |
| `ARRAY_AGG(... ORDER BY ... LIMIT ...)` | Aggregate, Sort Limit, and Array Unnest |
| `ARRAY_TRANSFORM(array, e -> expression)` | scalar Array Subquery over Array Unnest, with the lambda body as a Function child |
| `ARRAY_FILTER(array, e -> predicate)` | scalar Array Subquery over Filter and Array Unnest; the predicate is the Filter's Condition child |
| `IN UNNEST(array)` | primary-key Filter Scan plus Array Constructor |
| scalar `WITH(...)` expression | scalar functions over a covering IndexScan |
| `ORDER BY ... COLLATE` | Compute/Function below Sort |
| aggregate-call `COUNT(DISTINCT ...)` | two Stream Aggregates; a Minor Sort appears from optimizer v5 |
| `ARRAY_AGG(DISTINCT ... IGNORE NULLS ORDER BY ...)` | three Stream Aggregates, Sort, Filter, and Array Unnest |
| `ARRAY_AGG(DISTINCT ... RESPECT NULLS ORDER BY ...)` | the same aggregate/sort/unnest family without the null-removal Filter |
| `NULL IS NOT DISTINCT FROM NULL` | Unit Relation plus a `true` Constant |
| `NULL IS DISTINCT FROM NULL` | Unit Relation plus a `false` Constant |
| explicit and implicit correlated `UNNEST` | byte-identical Cross Apply plus Array Unnest plans in optimizer v1-v8 |
| `LEFT JOIN UNNEST` | Outer Apply plus Array Unnest, preserving the left row when the array contributes no row |
| `IN (subquery)` and `IN UNNEST(ARRAY(subquery))` | byte-identical plans in every tested optimizer version; BUILD_SEMI Hash Join in v1-v4, Semi Apply in v5-v8 |
| `NOT IN (subquery)` and `NOT IN UNNEST(ARRAY(subquery))` | byte-identical plans in every tested optimizer version; BUILD_ANTI_SEMI Hash Join in v1-v4, Anti-Semi Apply in v5-v8 |
| `GROUP BY` ordinal and name control | byte-identical Stream Aggregate plans in optimizer v1-v8 |
| `TABLESAMPLE` on a subquery | Random Id Assign plus Filter, byte-identical in optimizer v1-v8 |
| sampled input joined to another table | Random Id Assign within a Distributed Cross Apply; the global Sort Limit becomes Limit at the v2/v3 boundary |
| `STRUCT` expression `.*` | Unit Relation with flattened scalar Constants |
| `UNNEST` of an array of structs | Array Constructor, two scalar Struct Constructors, and Array Unnest |
| regular-table-first set operation with a value-table input | executable top-level Union All plan |
| value-table-first set operation nested under an ordinary SELECT | executable Union All plan; the outer SELECT converts the value-table result to an ordinary result column |

The absence of a clause-specific operator does not weaken the syntax result:
these statements still passed the Spanner runtime and returned a standard
QueryPlan at every optimizer version. Top-level `SELECT AS VALUE` is a separate
result-shape restriction and was rejected.

The product-specific Spanner operators page retrieved on 2026-08-11 omitted
both `IS DISTINCT FROM` and `IS NOT DISTINCT FROM`. The
[BigQuery GoogleSQL operators page](https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/operators#is_distinct_from_operator)
documents both spellings and their NULL semantics, but it is not Spanner
support evidence. The pinned runtime evidence above independently establishes
acceptance in the tested Spanner environment. The strict memefish parser
snapshot `64f857b2c61e` rejects both forms, while the spanalyzer GoogleSQL
frontend accepts them; these parser layers must not be treated as
interchangeable service evidence.

## Descriptor-backed proto surfaces

The dedicated proto selector merges the repository's two serialized
`FileDescriptorSet` fixtures before creating a database. Spanner Omni requires
both a message and any referenced nested enum to be named in the proto bundle;
registering `examples.user.User` without `examples.user.User.UserType` fails
database setup even though the local GoogleSQL frontend accepts the parent
message registration.

The following forms return plans at optimizer versions 1 through 8:

- both `NEW proto { ... }` and `NEW proto(...)` constructors;
- `CAST(string AS proto)` and nested-path `REPLACE_FIELDS`;
- nested `SELECT AS proto` and `SELECT DISTINCT AS proto`;
- ordinary, nested, presence, enum, and repeated proto field access; and
- `UNNEST` over a repeated proto field.

The four construction forms that are not folded away expose a previously
uncataloged SCALAR operator named `Proto Constructor`. It has no metadata and
has repeated unnamed SCALAR child links with absent variables. The four
observed occurrences and their plans are byte-stable from optimizer v1
through v8. The
catalog now embeds that exact observed combination, so a future kind,
metadata, or child-link change is reported as unknown rather than silently
accepted.

The duplicate-capable `SELECT DISTINCT AS proto` plan contains one Local and
one Global Hash Aggregate, each with two `Key` links, followed by the proto
constructor. This is plan evidence that duplicate elimination operates on the
input fields before construction. Presence access has a narrower visibility
limit: `OrderInfo.has_order_number` is represented as a `Field` whose
`metadata.name` is `order_number`, so QueryPlan metadata alone does not
distinguish the presence accessor from ordinary field access.

Pinned upstream GoogleSQL also accepts `EXTRACT(FIELD/HAS/RAW/ONEOF_CASE)`,
`PROTO_DEFAULT_IF_NULL`, and `FILTER_FIELDS` in the local frontend. Pinned
Omni rejects them at every optimizer version: the EXTRACT forms report
`EXTRACT from PROTO is not supported`, while the two function forms report
`Function not found`. Top-level `SELECT AS proto` independently reaches the
same top-level value-table restriction as `SELECT AS VALUE` and
`SELECT AS STRUCT`.

## Invariant runtime boundaries

| Surface | Runtime result |
| --- | --- |
| core pipe syntax | `Pipe query syntax not supported` |
| `GROUP BY GROUPING SETS` | `GROUP BY GROUPING SETS is unsupported` |
| `GROUP BY ROLLUP` | `GROUP BY ROLLUP is unsupported` |
| `GROUP BY CUBE` | `GROUP BY CUBE is unsupported` |
| `PIVOT` | `PIVOT is not supported` |
| `UNPIVOT` | `UNPIVOT is not supported` |
| `FOR UPDATE` under PLAN's read-only transaction | transaction-type error |
| `LOCK_SCANNED_RANGES` with `FOR UPDATE` in read-write mode | explicit combination error |
| recursive `WITH` | `RECURSIVE is not supported in the WITH clause` |
| top-level `SELECT AS VALUE` | top-level value-table shape error |
| `WITH` inside a subquery | language-version restriction |
| `ROW_NUMBER() OVER (...)` | `Unsupported built-in function: ROW_NUMBER` |
| framed `SUM(...) OVER (...)` | `Unsupported built-in function: SUM` |
| `TABLESAMPLE ... REPEATABLE (...)` | `REPEATABLE TABLESAMPLE is not supported` |
| top-level `SELECT AS STRUCT` | top-level value-table shape error |
| `CROSS JOIN LATERAL` and `LEFT JOIN LATERAL` | `LATERAL join is not supported` |
| SQL set operation `BY NAME` | `BY NAME for set operations is not supported` |
| SQL set operation `CORRESPONDING` | `CORRESPONDING for set operations is not supported` |
| `GROUP BY ALL` | `GROUP BY ALL is not supported` |
| `MATCH_RECOGNIZE` | service syntax error at the clause |
| value-table-first top-level set operation | top-level value-table shape error |
| `SECURE_CONTEXT(key)` | `Unsupported built-in function: SECURE_CONTEXT is not supported` |

The same `FOR UPDATE` statement was analyzed in both transaction contexts. It
returned a plan in read-write mode and the documented transaction-type error
in read-only mode. Running the combined `LOCK_SCANNED_RANGES` plus `FOR UPDATE`
case in read-write mode reached the later, explicit combination error instead
of being masked by the read-only check. These are PLAN/capability results;
they do not prove that locks were acquired during query execution.

The two analytic statements are valid GoogleSQL analytic syntax but do not
produce a plan on this pinned Omni runtime. Keeping both a numbering function
and a framed aggregate distinguishes an Omni capability boundary from a
missing plan-vocabulary case. Aggregate-call `DISTINCT` is supported: its
tested plan gains a `Minor Sort` only at optimizer versions 5 through 8.
`IGNORE NULLS` adds one Filter at every version, while the explicit
`RESPECT NULLS` form retains the no-filter shape.

The emulator conformance suite executes `ARRAY_TRANSFORM`, `ARRAY_FILTER`,
`ARRAY_FIRST`, and `ARRAY_LAST`; the first and last functions were already
covered by spanalyzer's frontend tests. The retained PLAN pair adds the two
lambda forms and distinguishes transformation from filtering by the latter's
relational Filter. The same emulator suite also executes `SECURE_CONTEXT`
inside a `SQL SECURITY DEFINER` view, but pinned Omni rejects a direct
`SECURE_CONTEXT` query at every optimizer version. This is a runtime
capability divergence, not evidence that either implementation defines the
managed-service contract. The current official
[Spanner views documentation](https://docs.cloud.google.com/spanner/docs/views)
describes and gives examples of `SQL SECURITY DEFINER`, which the local
row-type analyzer now covers, but it does not document a `SECURE_CONTEXT`
function. Consequently spanalyzer does not register that emulator-only
function in its local catalog.

## Source and runtime divergences

The current Spanner query-syntax document defines explicit and implicit
`UNNEST`, correlated array paths, ordinal grouping, `TABLESAMPLE` on a
subquery, and value-table set operations. These documented surfaces are now
retained as plans rather than inferred from neighboring examples.

The pinned upstream GoogleSQL checkout additionally defines `LATERAL`, set
operation `BY NAME` and `CORRESPONDING`, `GROUP BY ALL`, and
`MATCH_RECOGNIZE` in grammar, AST, unparser, resolver, and parser/analyzer
tests. The spanalyzer frontend based on `go-googlesql` v0.3.0 analyzes all of
the retained statements and derives row types. Pinned Omni instead returns the
explicit unsupported errors above, except for `MATCH_RECOGNIZE`, which is
rejected at syntax. Frontend acceptance is therefore deliberately tested
separately and is not presented as Spanner runtime support.

The value-table set-operation wording has a narrower inconsistency. The
current Spanner page says that combining a one-column ordinary table with a
value table always produces a value table. Upstream
`googlesql/analyzer/resolver_query.cc` says the output is a value table only
when the first input is a value table. Pinned Omni follows the upstream rule:

- ordinary first, value table second executes as an ordinary top-level result;
- value table first, ordinary second reaches the top-level value-table error;
- wrapping the value-table-first operation in an ordinary outer SELECT makes
  it executable.

The integration test executes both directions for all six combinations of
`UNION`, `INTERSECT`, and `EXCEPT` with `ALL` and `DISTINCT` at optimizer
versions 1 through 8. This establishes a pinned-Omni/documentation divergence,
not a managed-service contract.

## Reproduction

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case google_sql_surface \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/google_sql_surface_expectations.json
```

Add `--optimizer-version-matrix` without the unprefixed expectation manifest
to repeat the 456-submission capability and vocabulary matrix. The
descriptor-backed selector uses the analogous command documented in
`tools/spanner-query-plan-shape/README.md` and requires both descriptor flags.
