# Predicate and join condition boundaries (2026-08-11)

This note records where related expressions appear in raw Spanner QueryPlan:
`Distributed Union` `Split Range`, scan `Seek Condition`, scan
`Residual Condition`, and join `Condition` / `Residual Condition`. These are
plan-shape observations, not performance measurements or a promise that the
optimizer will make the same choice for another schema or release.

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
- Upstream grammar and analyzer: `google/googlesql` commit
  `1f8aa333f4d6353cd3a64471fc83121df72df3f7` (2026-07-21).
- Retained selector: `condition_boundaries`, 38 queries.
- Default result: 38 plans, no errors, and 62 operator expectations.
- Optimizer matrix: 304 plans covering versions 1 through 8, with no query
  errors. The focused integration test checks the condition expression text,
  not only operator presence.

The official scan documentation says a key-prefix predicate is generally
seekable. The official hash-join documentation separately distinguishes the
equality join condition from a residual condition and gives non-equality joins
as the residual example. The probes below make those boundaries concrete for
functions, parameters, conjunctions, disjunctions, and different join
families.

## Split Range is not the same classification as Seek Condition

Every forced `SongsBySongName` probe has a `Distributed Union` `Split Range`
expression, including predicates that the scan evaluates only as residuals.
For example, `LIKE '%A%'`, `STRPOS(SongName, 'A') > 0`, `ENDS_WITH`, `SUBSTR`,
and `LOWER` appear verbatim in `Split Range`, while the lower `Filter Scan`
retains the same expression as `Residual Condition` and the underlying index
scan has no `Seek Condition`.

Consequently, the presence of a `Split Range` link does not prove that a
predicate is seekable. In this plan family it is the distributed expression
used while dispatching work; the scan-side `Seek Condition`, `Residual
Condition`, and `Full scan` metadata provide the access classification.

A mixed conjunction shows the three layers directly:

```sql
WHERE STARTS_WITH(SongName, 'A') AND STRPOS(SongName, 'x') > 0
```

- `Split Range`: the complete conjunction;
- `Seek Condition`: `STARTS_WITH($SongName, 'A')`; and
- `Residual Condition`: `(STRPOS($SongName, 'x') > 0)`.

## String predicate boundary

| Predicate on the leading index key | v1 | v2-v8 |
| --- | --- | --- |
| equality, explicit range | Seek | Seek |
| `STARTS_WITH(key, 'A')` | Seek | Seek |
| `STARTS_WITH(key, @prefix)` | Seek | Seek |
| `key LIKE 'A%'` | rewritten to `STARTS_WITH`; Seek | same |
| `key LIKE @pattern`, with parameter value `A%` | Residual | Residual |
| `key LIKE '%A%'` | Residual | Residual |
| `REGEXP_CONTAINS(key, r'^A.*')` | Residual regexp | rewritten to `STARTS_WITH`; Seek |
| `REGEXP_CONTAINS(key, r'^A.B')` | Residual regexp | `STARTS_WITH('A')` Seek plus regexp Residual |
| unanchored regexp, `STRPOS`, `ENDS_WITH`, `SUBSTR`, or `LOWER(key)` | Residual | Residual |
| `key = UPPER('a')` | constant-folded equality Seek | same |
| two `STARTS_WITH` predicates joined by `OR` | combined Seek | combined Seek |

The parameter comparison is important: the runtime knows the value supplied
for `@pattern`, but the `LIKE` shape remains residual because the fixed-prefix
property is not part of the prepared SQL text. `STARTS_WITH` exposes its prefix
argument directly and remains seekable with a parameter. This agrees with the
official best-practices guidance to prefer parameterized `STARTS_WITH` over a
parameterized `LIKE` prefix.

Optimizer v2 performs partial extraction, not only all-or-nothing rewriting.
For `LIKE 'A_B%'` and the anchored regexp `^A.B`, v1 retains the whole
predicate as residual. Versions 2 through 8 extract `STARTS_WITH('A')` as the
seek bound and retain the original predicate as a residual correctness check.
For `^A.*`, the extracted prefix is sufficient and the regexp residual
disappears.

One non-monotonic observation should not be generalized: the mixed
disjunction `STARTS_WITH(key, 'A') OR ENDS_WITH(key, 'B')` is shown wholly as
a `Seek Condition` in v1-v6, but wholly as a `Residual Condition` in v7-v8.
`ENDS_WITH` alone is residual in every version. The test retains this exact
boundary as observed behavior; it does not infer that arbitrary disjunctions
containing a non-seekable term are semantically seekable.

## Commit timestamp key boundary

`CommitTimestampKeys` has an `allow_commit_timestamp = true` `CommitTs`
column as the leading primary-key part. With a range predicate on `CommitTs`
and `ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=TRUE`, the same `Scan` has both a
`Seek Condition` and a `Timestamp Condition` in every optimizer version from
v1 through v8. The two links therefore coexist: the former is the key-range
access signal and the latter is the commit-timestamp storage-pruning signal.

With the same SQL and `ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=FALSE`, the
`Seek Condition` remains on that `Scan`, while `Timestamp Condition` is
absent. This is a plan-shape observation only; it does not quantify the
additional pruning or prove a runtime-performance effect.

## Join condition boundary

For a forced Hash Join, `Condition` is broader than bare column equality.
Each of the following equality forms is used as the hash condition in v1-v8:

- two columns, including a conjunction of two equality keys;
- `CAST(left AS STRING) = CAST(right AS STRING)`;
- arithmetic such as `left + 1 = right`;
- `LOWER(left) = LOWER(right)`; and
- pinned-runtime null-safe equality, rendered as
  `IS_NOT_DISTINCT_FROM(left, right)`.

The last spelling is pinned-Omni evidence rather than current product-reference
evidence; the Spanner GoogleSQL operators page in the pinned documentation
mirror does not list it.

A cross-input predicate that is not an equality remains on the same Hash Join
as `Residual Condition`. This includes `<`, `STARTS_WITH(right, left)`,
`STRPOS(right, left) > 0`, and a dynamically constructed partial `LIKE`.
When no conjunctive equality is available, a forced Hash Join still appears,
but its `Condition` is `(true = true)` and the real inequality or disjunction
is entirely residual. In particular, an `OR` of two cross-input equalities is
not decomposed into hash keys.

A one-side predicate is independent of the join condition. With
`SongsBySongName` forced, `STARTS_WITH(s.SongName, 'A')` becomes that input's
`Split Range` and `Seek Condition`, while `a.SingerId = s.SingerId` remains the
Hash Join `Condition`. Without the matching forced index, optimizer versions
can instead leave the one-side predicate residual on the chosen scan.

Merge Join follows the same logical split: computed `LOWER` equality is a
`Condition` after explicit sorts, while an additional cross-input inequality
is a `Residual Condition`. This probe exposed a previously uncataloged
combination: `Merge Join` with a scalar `Residual Condition` and multiple
variable-bearing scalar `Right` links. The default-deny catalog now admits
that exact v1-v8 observation.

Apply Join expresses the boundary differently. There is no join-level
`Condition` child in these cases; correlation is pushed into the map-side
scan. Equality becomes its `Seek Condition`, and equality plus range becomes a
combined correlated seek over two key columns.

Bloom-filter predicates produced by Hash Join may separately appear as a scan
`Residual Condition`. They are an implementation filter on a join input, not
the logical join residual described above; consumers must consider the parent
operator when interpreting the link type.

## Upstream-only function boundary

An initial control used `CONTAINS_SUBSTR`. The local GoogleSQL frontend knows
that upstream function, but pinned Omni returned `Function not found:
CONTAINS_SUBSTR`. The retained Spanner probes therefore use supported
`STRPOS(...) > 0` for the comparable substring predicate. Frontend acceptance
must not be treated as runtime support.

## Reproduction

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case condition_boundaries \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --expect tools/spanner-query-plan-shape/testdata/condition_boundary_expectations.json
```

Add `--optimizer-version-matrix` without the unprefixed expectation manifest
to repeat the 304-plan vocabulary matrix. The focused integration test is the
authority for the expression strings and optimizer-version partitions because
the plan-vocabulary manifest intentionally matches operator/link structure,
not free-form `short_representation.description` text.
