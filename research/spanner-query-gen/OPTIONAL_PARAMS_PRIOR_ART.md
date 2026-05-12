# Optional Query Parameters: Prior Art Survey

Status: non-normative survey collected while sketching the `internal/optparam`
PoC. The goal is to position `spanner-query-gen`'s design against existing
OSS approaches for dynamic SQL generation and to flag the design choices we
should make consciously instead of by default.

This survey is written from working knowledge and is not a live web crawl.
Names, syntax details, and version-specific behavior should be verified
against each project's current documentation before being acted on.

## Reference Axes

To keep comparison concrete, every project below is rated against four
axes that matter for our use case.

1. **Template vs. builder vs. DSL**. Where does the SQL live?
   - *Template*: SQL is authored as text (sometimes with directives) and
     the library renders one final statement.
   - *Builder*: SQL is constructed via fluent method calls.
   - *DSL*: SQL is expressed in a host-language DSL with type-level
     guarantees.
2. **Compile-time analysis**. Does the tool parse the SQL against a real
   schema and surface type errors at build time? (`sqlc`, `pgtyped`,
   `spanner-query-gen` do; most builders do not.)
3. **Variant strategy**. How does it handle "predicate omitted"?
   - *Single SQL with NULL guard*: emit `(@p IS NULL OR col = @p)` and
     rely on the planner.
   - *Runtime concatenation*: glue strings at call time.
   - *Pre-expanded variants*: enumerate concrete SQL strings at build
     time (what our PoC does).
4. **Type-checked caller API**. Can the caller pass partial inputs in a
   way the type system understands?

## Closest conceptual matches

### TwoWaySQL family (Doma 2, uroboroSQL, S2Dao lineage)

The single most direct precedent. TwoWaySQL is a Java convention where
the dynamic-SQL directives live inside SQL comments so the source file
remains a *runnable* SQL statement when the directives are ignored.

Doma 2 syntax (Java/Kotlin):

```sql
SELECT id, name FROM employee
WHERE
  /*%if firstName != null */
  first_name = /* firstName */'sample'
  /*%end*/
  /*%if status != null */
  AND status = /* status */'active'
  /*%end*/
```

- `/*%if cond %*/ ... /*%end%*/` is a directive in a comment.
- `/* binding */ literal` is a binding marker plus a default literal so
  the same file is valid SQL outside the framework.

uroboroSQL extends the same approach with `IF`/`ELSE`/`BEGIN`/`END`
directives and treats the SQL file as the source of truth.

Relevance to us:

- Our `/*?optional:NAME*/ ... /*?end*/` marker is a stripped-down
  TwoWaySQL `if`. We chose a fixed predicate ("is the named param NULL")
  instead of a general expression so we can statically enumerate the
  variants.
- TwoWaySQL targets runtime template rendering and does not generally
  emit one prepared statement per branch. We want the opposite — emit
  every branch ahead of time so each gets its own execution plan.

### MyBatis Dynamic SQL

The XML-based ancestor in Java/Kotlin land. `<if test="...">`,
`<where>`, `<choose>`, `<foreach>` build SQL at runtime from a mapper
file. MyBatis Dynamic SQL (the newer programmatic API) replaces the XML
with builder code but keeps the runtime-rendering model.

Relevance: confirms that "predicate-block on/off" is the de-facto
dominant shape of dynamic SQL in mature ecosystems, but it is always
runtime-rendered and never pre-expanded.

### dbt / Jinja over SQL

`{% if var('status') %} AND status = '{{ var("status") }}' {% endif %}`.
Powerful and unconstrained — the rendered SQL is whatever the template
emits. Most analyses run after rendering, on the final SQL.

Relevance: highlights how unrestricted templating loses static
guarantees. Our PoC's hard rule "every variant must analyze to the same
row type" is precisely the constraint dbt-style templating gives up.

### HugSQL / YesQL (Clojure)

SQL files annotated with named queries; `:snip` and `--~` fragments
allow conditional inclusion at call time. The conditional logic is
authored in Clojure and the SQL is rendered per invocation. No
build-time variant expansion.

## Codegen tools with optional-param stories

### sqlc (Go)

Closest peer in spirit. SQL files are parsed against a schema; type-safe
Go code is generated.

- `sqlc.narg('x')` introduces a nullable parameter. Optional WHERE is
  typically expressed as `WHERE (sqlc.narg('x')::text IS NULL OR
  col = sqlc.narg('x'))` — i.e. our `null_is_null` mode rendered via the
  single-SQL-with-NULL-guard pattern.
- Plain `@x` is required (non-nullable).
- There is no "omit the predicate" mode; sqlc's static row-type and the
  pre-known plan rely on a single SQL string per query.

Relevance: validates `null_is_null` and `required` modes directly. The
gap that motivates our `omit_when_null` is what sqlc does *not* offer.

### pgtyped (TypeScript)

Reads tagged SQL queries, type-checks against the live database, and
emits typed TS clients. Supports parameter spreads for `IN` lists. Does
not emit pre-expanded variants for optional predicates; the typical
recommendation is also `col IS NULL OR col = $1`.

### Diesel (Rust)

A typed query builder with `.into_boxed()` enabling conditional `.filter`
chains. Builder-style, no SQL templates. Schema-aware, compile-time
typed.

Relevance: the most ergonomic builder API for optional WHERE we have in
mainstream OSS. Builders sidestep template variant management entirely —
the cost is paid in giving up SQL-as-source-of-truth.

### Ent (Go), GORM, Squirrel, goqu, Jet

All builder-style Go libraries. Optional predicates are added by `if
opts.X != nil { q = q.Where(...) }`. Some (Jet, ent's predicate API)
provide compile-time-checked column references; none materialize one
prepared statement per call shape.

### jOOQ (Java)

Heavy-weight DSL. Optional filters compose via `.and()` calls and the
final SQL is rendered per execution. Schema-aware. No build-time variant
enumeration.

### Quill (Scala), Slick (Scala)

DSL with macro-driven query compilation. Optional predicates are
expressed as Scala collection operations. Quill in particular can fold
constant booleans away at compile time, which is conceptually close to
"the predicate is gone".

### Prisma (TypeScript/Node)

Schema-first. Filters are JavaScript objects passed at runtime; absent
keys are omitted. The generated client gives up SQL-as-source-of-truth.

## How our PoC positions

| Project              | SQL is source | Schema-checked at build | Variant strategy           | Caller API typing               |
|----------------------|---------------|-------------------------|----------------------------|---------------------------------|
| Doma TwoWaySQL       | Yes           | No (runtime)            | Runtime rendering          | Java method-level types         |
| uroboroSQL           | Yes           | No (runtime)            | Runtime rendering          | Java method-level types         |
| MyBatis Dynamic SQL  | Yes (XML)     | No                      | Runtime rendering          | Java/Kotlin types               |
| sqlc                 | Yes           | Yes                     | Single SQL + NULL guard    | Go generated types              |
| pgtyped              | Yes           | Yes (live DB)           | Single SQL + NULL guard    | TS generated types              |
| Diesel / Ent / GORM  | No (builder)  | Yes / partial           | Runtime concatenation      | Rust / Go generated types       |
| jOOQ / Slick / Quill | No (DSL)      | Yes                     | Runtime concatenation      | JVM types                       |
| **spanner-query-gen PoC** | Yes      | Yes                     | **Pre-expanded variants**  | Go generated types (planned)    |

The combination "SQL is source" + "schema-checked at build" + "variants
pre-expanded" is, as far as this survey found, not occupied. The closest
neighbor is sqlc with its single-SQL strategy; the closest design
inspiration is TwoWaySQL with its comment-embedded directives.

## What this implies for the design

1. **The marker convention is well-trodden.** TwoWaySQL has proven that
   directives-in-SQL-comments is an ergonomic authoring model. Our
   `/*?optional:NAME*/ ... /*?end*/` is a minimal subset of that idea
   purpose-built for predicate-block on/off.

2. **Pre-expansion is the differentiator.** The reason to do build-time
   variant enumeration — instead of a single `(@p IS NULL OR ...)` SQL —
   is the Spanner execution plan. Spanner's planner cannot always
   optimize the NULL-guard form to the same plan as the simpler predicate
   form, and "one prepared SQL per shape" is what makes per-variant plan
   contracts meaningful.

3. **Builder-style alternatives are off-the-shelf.** If we judge that
   pre-expansion is not worth the analyzer cost, the alternative is to
   give up SQL-as-source for the optional cases and route them through a
   small builder API. That would put us next to Diesel / Ent and we
   would lose the SQL audit story. The PoC's empirical 2 s analyzer
   wall-clock on a tiny case is the data point that supports keeping
   pre-expansion.

4. **Three-valued behavior maps onto known idioms.**
   - `required` ≈ sqlc's plain `@p`.
   - `null_is_null` ≈ sqlc's `sqlc.narg` + the `IS NULL OR =` idiom.
   - `omit_when_null` ≈ Doma's `/*%if%*/` block but evaluated at
     codegen time.

## Open questions worth verifying against the OSS landscape

- Does any TwoWaySQL implementation pre-render variants and cache them?
  (Not seen, but worth a closer look at uroboroSQL's batch processing
  modes.)
- Does sqlc have an open issue tracking "optional predicate omission" as
  a mode? Confirming that would give us a place to publish the
  Spanner-side write-up.
- Is there prior art for variant-aware plan contracts (one execution
  plan per SQL shape, hashed and checked into CI) outside the Spanner
  ecosystem? `pg_qualstats` and PostgreSQL's `auto_explain` operate per
  plan but are observability tools, not contract tools.

## References (verify before citing)

The following links capture where the named projects' documentation
typically lives. They were not fetched during this write-up and should
be re-checked before being included in normative documentation.

- Doma 2: <https://doma.readthedocs.io/>
- uroboroSQL: <https://future-architect.github.io/uroborosql/>
- MyBatis Dynamic SQL: <https://mybatis.org/mybatis-dynamic-sql/>
- sqlc: <https://docs.sqlc.dev/> (look for `sqlc.narg` and nullable
  arguments)
- pgtyped: <https://pgtyped.dev/>
- Diesel: <https://diesel.rs/>
- Ent: <https://entgo.io/>
- jOOQ: <https://www.jooq.org/>
- HugSQL: <https://www.hugsql.org/>
- dbt Jinja: <https://docs.getdbt.com/docs/build/jinja-macros>
