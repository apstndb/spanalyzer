# Optional Query Parameters: Design Notes

Status: non-normative design notes for the v1alpha optional-parameter
implementation. The original `internal/optparam` PoC has been merged into
`internal/querygen`; this document still records the design rationale and
remaining follow-up work.

Scope: how `spanner-query-gen` could support optional query parameters that
drive dynamic SQL generation at codegen time, plus the design questions that
must be settled before the feature is promoted into the v1alpha config and
the plan contract.

## Problem Statement

A common need in application code is "a single named query whose WHERE
clause is partly conditional". For example, a customer-search query may
accept any subset of `{first_name, status, since, until}` and the SQL
emitted to Spanner should reflect only the predicates that the caller
actually provided. Today `spanner-query-gen` treats every declared
parameter as required, so users either fall back to hand-written SQL or
abuse Spanner runtime tricks like `(@p IS NULL OR col = @p)`.

The goal is to express the optional behavior in the YAML config and have
the generator emit:

1. One SQL constant per variant (so the Spanner query planner can pick the
   best execution plan for each shape).
2. Go code that lets the caller signal "this filter is absent" in a
   type-safe way.
3. A plan-contract entry that records the row type and per-variant SQL
   hashes so downstream `spanner-query-plan-shape` runs can produce one
   execution plan per variant.

## Core Axis: per-parameter `optional` mode

The v1alpha scope adds a new field on each parameter and a small marker grammar
inside the SQL template. The current implementation supports four kinds of
markers, each backed by a per-parameter mode:

```yaml
params:
  - name: first_name
    type: STRING
    optional: omit_when_null
  - name: ids
    type: ARRAY<INT64>
    optional: omit_when_empty
  - name: status
    type: STRING
    optional: null_is_null
  - name: sort
    optional: orderby_choice
    default: id_asc
    choices:
      id_asc:   "ORDER BY SingerId ASC"
      name_asc: "ORDER BY LastName, FirstName"
```

| Mode              | Marker                                       | Caller signals "absent" via             | SQL effect                                                                                |
|-------------------|----------------------------------------------|-----------------------------------------|-------------------------------------------------------------------------------------------|
| `required`        | none                                         | n/a                                     | None. `col = @p` is unchanged.                                                            |
| `null_is_null`    | `/*?null_is_null:NAME*/ ... = @NAME ... /*?end*/` | `spanner.NullString{Valid: false}`  | Generator rewrites `= @NAME` to `IS NOT DISTINCT FROM @NAME`. No SQL variants.            |
| `omit_when_null`  | `/*?optional:NAME*/ ... /*?end*/`            | `*T == nil`                             | Block is removed from the SQL. Multiplies variant count by 2.                             |
| `omit_when_empty` | `/*?empty:NAME*/ ... IN UNNEST(@NAME) ... /*?end*/` | `len([]T) == 0`                  | Block is removed from the SQL. Multiplies variant count by 2.                             |
| `orderby_choice`  | `/*?orderby:NAME*/ <default ORDER BY> /*?end*/` | string choice key (defaults to Default) | Block body is replaced wholesale by `Choices[key]`. Multiplies variant count by N choices. |

`null_is_null` is single-SQL: the rewritten predicate stays in place and
runtime relies on Spanner's `IS NOT DISTINCT FROM`.

`omit_when_null`, `omit_when_empty`, and `orderby_choice` are multi-SQL:
the generator enumerates the Cartesian product of their on/off bits and
choice keys, analyzes each one separately, and confirms every product
point yields the same row type. The runtime composer (`EmitGoBuilder`)
walks the segment list linearly and produces a SQL string byte-equal to
exactly one of those verified variants.

## Other directions worth exploring

These were brainstormed alongside the two-mode proposal. None are blockers
for v1, but they shape the data model so it is worth deciding which ones
the design admits and which it explicitly defers.

| Category                                              | Example                                                                       | Notes                                                                              |
|-------------------------------------------------------|-------------------------------------------------------------------------------|------------------------------------------------------------------------------------|
| Three-valued NULL: any / IS NULL / no match           | `optional: any/is_null/none`                                                  | Common UI filter tristate. Strictly more expressive than today's `null_is_null`.   |
| Operator selection                                    | `=`, `IN @arr`, `LIKE`, prefix-`LIKE`, range (`>=`/`<=`/`BETWEEN`)            | Orthogonal to the optional axis.                                                   |
| Empty array handling                                  | empty `IN` → predicate dropped / always-false / always-true                   | Spanner `IN UNNEST(@arr)` empty-array semantics matter here.                       |
| Range pairs                                           | `since` / `until` independently optional                                      | The canonical cursor / time-window shape.                                          |
| ORDER BY / LIMIT / OFFSET                             | Whitelist-driven dynamic sort plus optional pagination                        | Allowlist the column set to prevent SQL injection.                                 |
| Keyset pagination                                     | Derive a typed cursor struct from ORDER BY                                    | Natural extension of the existing `key_prefix`.                                    |
| Column projection                                     | Runtime SELECT mask                                                           | Changes the result struct type; needs to be a separate feature.                    |
| STRUCT-level optional spread                          | One STRUCT param expands into per-field optionals                             | Clean API surface, but heavy analyzer cost.                                        |

## Largest design questions (decide first)

### 1. How is the analyzer driven?

Recommendation: **"analyze the fully-enabled SQL once, treat the resulting
row type as the contract, and require every variant to agree."**

That invariant — "predicate omission must not change the result row type"
— is what lets the generator reuse one Go result struct across every
variant. Features that violate the invariant (runtime projection, runtime
ORDER BY changing the result shape) belong to a separate RFC.

In the current `internal/optparam` implementation the verifier runs the analyzer per variant
(2^k times) and proves equality directly via `proto.Equal` on the
`spannerpb.StructType`. The cost is modest because:

- 2 optional params → 4 analyses → ~2 s wall (warm WASM compile cache).
- The analyzer can be reused across variants in future if needed; the current
  bridge rebuilds it per variant for isolation.

### 2. What does the Go-side API look like?

Recommendation: keep the existing `NullValue[T]` story consistent.

- For `required`: emit a plain `T` field on the generated params struct.
- For `null_is_null`: emit `spanner.NullString` / `spanner.NullInt64` /
  matching `Null*` types so the Valid flag carries the IS-NULL signal.
- For `omit_when_null`: emit `*T` (or `mo.Option[T]`) so a `nil` pointer
  means "this predicate is omitted".

The generated entry point picks the SQL variant from the set of non-nil
optional pointers and builds the Spanner params map accordingly.

### 3. What is the plan-contract key?

A single `sql_sha256` no longer uniquely identifies a query once the SQL
has multiple variants. Two options:

- **Per-variant entries**: `PlanQueryVariant{label, sql, sql_sha256, ...}`
  carried under `QueryCodegenPlanQuery.Variants`. Each variant gets its
  own plan in the plan-shape probe. This is what the current
  `BuildPlanVariants` already produces.
- **Composite hash**: keep one entry per query but hash
  `(template, sorted-set-of-present-predicates)`. Simpler but loses the
  per-variant plan rows downstream.

Recommended: per-variant entries, since the whole reason for emitting
multiple SQL strings is so each has its own plan.

## Hybrid: exhaustive verification, linear composition

The pre-expansion done at codegen time is for **verification and
plan-contract recording only**. The generated runtime code does not
embed all 2^k SQL strings; it embeds the template's segment list and
composes one SQL string per call by walking the segments linearly.

This is the hybrid invariant the implementation enforces:

```text
ComposeVariant(segments, presence_set) == EnumerateVariants(sql, params)[label_for(presence_set)].SQL
```

The codegen-time side records every variant (`label`, `sql`,
`sql_sha256`, present/absent param sets) into the plan contract, and the
runtime composer is generated from the same segment list so its output
text is guaranteed byte-identical to one of those verified variants.

Why this matters:

- **Binary size**. The variant table grows as 2^k. The segment list grows
  linearly in k. For k≥5 the difference dominates the generated file
  size.
- **Source readability**. The generated builder function reads top-to-
  bottom in the same order as the SQL template's marker blocks, so a
  reader can map "this predicate appears here when this field is set"
  one-for-one.
- **Plan contract integrity**. Per-variant entries remain the only
  artifact the plan-shape probe consumes, and the runtime composer
  cannot drift away from them as long as the segment list is the shared
  source of truth.

Constraints the implementation must hold:

1. **Segmentation runs once at codegen time.** Both
   `EnumerateVariants` and the emitted builder consume the same
   `[]Segment`. The runtime composer never reformats, retokenizes, or
   "tidies" the SQL.
2. **Runtime output equals one verified variant.** The builder returns
   the variant label alongside the SQL so callers can attribute logs
   and plan-shape probe results back to a specific contract entry.
3. **Only `omit_when_null` is segment-driven.** `null_is_null` does not
   change the SQL — it only changes how the caller passes the value
   (`spanner.NullString{Valid: false}` instead of a `*T` that is nil).
   `required` is unchanged.

## Implementation Results (`internal/optparam`)

The marker grammar and variant machinery live in `internal/optparam/` and are
intentionally decoupled from `internal/querygen` so the data model remains
independently testable.

Build-time pipeline:

- `SegmentTemplate(sql, params)` — single source of truth for the
  template's segmentation. Returns `[]Segment` (fixed text alternating
  with optional blocks).
- `ComposeVariant(segments, present)` — composes one SQL string from a
  presence set. Same logic the runtime composer is generated from.
- `EnumerateVariants(sql, params)` — calls `SegmentTemplate` and emits
  one `Variant` per 2^k presence combination by calling `ComposeVariant`.
- `VerifyVariants(ddl, sql, params)` — runs the GoogleSQL analyzer on
  each variant and rejects any divergence in the resulting row type via
  `proto.Equal`.
- `BuildPlanVariants(result)` — produces the plan-contract-shaped slice
  (`label`, `sql`, `sql_sha256`, `present_params`, `absent_params`).

Runtime pipeline:

- `EmitGoBuilder(segments, params, opts)` — generates a Go file
  containing a `<FuncName>Params` struct (one `*T` field per
  `omit_when_null` parameter) and a `<FuncName>(p) (sql, args, variant)`
  function. The body writes each segment to a `strings.Builder`, gating
  optional blocks on `if p.X != nil`, and returns the variant label
  built by sorting and joining the names that were present.
- `VerifyBuilderRoundTrip(segments, variants)` — asserts the runtime
  composer's output equals every enumerated variant byte-for-byte. The
  codegen pipeline must call this before writing the generated file to
  disk.

Test fixtures cover the four marker kinds plus two derived patterns
(20 tests, all green):

- **`omit_when_null`** — `Singers` table with two optional `STRING`
  predicates (`first_name`, `status`). Four variants, identical row
  type, runtime composer `go run`-compiled and byte-equal to each
  verified variant.
- **`null_is_null`** — single SQL after rewriting `= @status` to
  `IS NOT DISTINCT FROM @status`. The analyzer accepts the rewritten
  predicate. Markers that don't actually contain `= @NAME` are
  rejected at codegen time.
- **`omit_when_empty`** — `IN UNNEST(@ids)` with `ARRAY<INT64>` type.
  Two variants, same row type. Emitter gates on `len(p.Ids) > 0` and
  emits `[]int64` rather than `*int64`.
- **`orderby_choice`** — two ORDER BY allowlist entries plus a default.
  Two variants (one per choice), identical row type. Builder emits a
  `switch` over choice keys plus a panic on unknown keys. The choice
  param is not added to the analyzer because it never reaches the
  query as a bind variable.
- **Cross-product (`omit × orderby`)** — 2 × 2 = 4 variants with keys
  like `sort=name_asc+status`. Confirms the mixed-radix enumeration.
- **Range pair (`since` / `until`)** — two independent
  `omit_when_null TIMESTAMP` predicates. Four variants, all identical
  row type. (Go emitter not exercised: TIMESTAMP needs a wrapper type
  the emitter has not mapped yet.)
- **Optional `LIMIT`** — `INT64` with `omit_when_null` mode. Two
  variants, same row type. Emitter produces `Limit *int64` and the
  conditional `LIMIT @limit` segment.
- **Divergent SELECT list** — negative test placing markers across the
  SELECT list. The cross-check catches the divergence.

### Marker Convention

```sql
SELECT SingerId, FirstName FROM Singers
WHERE TRUE
  /*?optional:first_name*/ AND FirstName = @first_name /*?end*/
  /*?optional:status*/    AND Status    = @status     /*?end*/
```

- The markers are SQL comments, so the same template is still a valid
  SQL statement (with all predicates) if the markers are ignored.
- Each block is removed wholesale when its parameter is absent. The
  author owns the connector keyword (`AND`/`OR`) inside the block, so
  the omission is syntactic, not AST-level.
- `WHERE TRUE` is the standard idiom for keeping the WHERE clause valid
  when every optional block is dropped.

This convention is deliberately ASCII-marker-based, not AST-driven, so the
runtime builder and plan-contract machinery can stay independent from memefish
or go-googlesql predicate rewriting. An AST-based detector can replace it later
without changing the data model.

## Minimum-viable v1alpha scope

The v1alpha implementation currently supports:

1. `params[].optional: required | null_is_null | omit_when_null |
   omit_when_empty | orderby_choice` (default `required`).
2. Marker syntax for hand-written SQL:
   - `/*?null_is_null:NAME*/ ... = @NAME ... /*?end*/`
   - `/*?optional:NAME*/ ... /*?end*/`
   - `/*?empty:NAME*/ ... /*?end*/`
   - `/*?orderby:NAME*/ <default ORDER BY> /*?end*/`
3. Generated `kind: table` and `kind: index` query support for key-prefix
   `null_is_null` / `omit_when_null`, plus `orderby_choice` ORDER BY
   allowlists.
4. Per-variant analysis with a hard row-type-equality check.
5. Per-variant plan-contract entries under `QueryCodegenPlanQuery.Variants`.
6. Generated Go params structs and runtime SQL builders. Optional queries no
   longer emit a single SQL constant; generated methods accept typed
   `<QueryName>Params` values and call a `Build<QueryName>SQL` helper that
   returns the selected SQL, bind args, and variant label.

The remaining MVP gap is downstream consumption: `tools/spanner-query-plan-shape`
does not yet expand plan-contract variants automatically.

## Implementation status

The integrated implementation covers, in addition to the original three
modes, the following items from the "other directions" table:

- Empty-array handling for `IN UNNEST(@arr)` (`omit_when_empty`) in
  hand-written SQL.
- Range pairs as two independent `omit_when_null` params.
- Optional `LIMIT` / `OFFSET` via `omit_when_null INT64`.
- ORDER BY allowlist (`orderby_choice`) with N choices folded into the
  variant Cartesian product.
- `kind: index` and `kind: table` generated SQL hooks for key-prefix optional
  predicates and ORDER BY choice markers.

## Explicitly deferred

- Three-valued NULL beyond `null_is_null` / `omit_when_null`. The
  runtime tristate (any / IS NULL / no match) collapses to the same
  SQL pre-expansion as `omit_when_null`; only the caller-side API
  surface needs to grow.
- Operator selection (`LIKE`, prefix-`LIKE`, range without paired
  params) as first-class config.
- Always-false / always-true policies for empty arrays. The current
  `omit_when_empty` covers the "drop the predicate" policy; the other
  two would need a different segment kind because the SQL is unchanged
  but the predicate truth value flips at runtime.
- Keyset pagination cursor structs derived from ORDER BY.
- Runtime column projection (changes the result row type, breaks the
  invariant the verification step depends on).
- STRUCT-level optional spread (one STRUCT param expanding into per-
  field optionals).

Each of these would benefit from its own RFC once the basic optional
mechanism is stable.
