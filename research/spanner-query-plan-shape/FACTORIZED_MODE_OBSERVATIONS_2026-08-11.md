# Factorized-mode plan observations (2026-08-11)

This note records environment-specific query-plan observations. It is not a
stable optimizer contract and contains no performance claim.

## Evidence environment

- Runtime: Spanner Omni
  `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`, local
  image digest
  `sha256:e98a088fa66d4a87dbb560d729bf21d998bb843f6018bd8dc118fe320e671886`.
- API: `AnalyzeQuery(QueryMode=PLAN)` through
  `spanner-query-plan-shape`.
- Schema: the built-in documentation `Albums` and `Songs` tables.
- Retained case: `factorized_mode` with 15 queries and
  `testdata/factorized_mode_expectations.json`.
- Revalidation: 10 plans, 5 expected query errors, 19 planvocab patterns,
  zero expectation failures, and zero vocabulary findings.
- Optimizer matrix: 120 submissions, covering every retained query at
  optimizer versions 1 through 8; 88 plans, 32 expected errors, and zero
  vocabulary findings.

## Plan-visible signature

The retained positive queries project non-join payload from each factorized
side. A visible factorized side contains:

- a Local Stream `Aggregate` with variable-bearing `Key` and `Agg` links; and
- an `Array Unnest` that restores the payload rows.

`FACTORIZE_BOTH` contains two such aggregate/unnest pairs. With an explicit
Hash Join, its `Hash Join INNER` has variable-bearing `Build` and `Probe`
links. There is no dedicated factorized operator name or metadata key, so the
suite contracts this composite signature instead.

## Optimizer-version boundary

All three values were accepted on a projection-eligible inner equijoin in
versions 1 through 8. Their visible effect was version-dependent:

| Query family | v1-v4 | v5-v8 |
| --- | --- | --- |
| unforced `FACTORIZE_LEFT` | plan returned; no Aggregate or Array Unnest | one factorized-side signature |
| unforced `FACTORIZE_RIGHT` | plan returned; no Aggregate or Array Unnest | one factorized-side signature |
| unforced `FACTORIZE_BOTH` | plan returned; no Aggregate or Array Unnest | two factorized-side signatures |
| `HASH_JOIN` plus `FACTORIZE_BOTH` | visible | visible |

The explicit v1 and v8 `HASH_JOIN` plus `FACTORIZE_BOTH` controls each had 43
nodes and byte-equal raw `plan_nodes` in the retained run. This is a property
of that query, not a general cross-version stability promise.

The v5 boundary also changes eligibility enforcement:

| Query shape | v1-v4 | v5-v8 |
| --- | --- | --- |
| factorized side projects only the join key | accepted with no factorization signature | factorized-execution eligibility error |
| non-equality join with `FACTORIZE_BOTH` | accepted with no factorization signature | factorized-execution eligibility error |
| left outer join with `FACTORIZE_BOTH` | eligibility error | eligibility error |

Therefore an accepted plan in v1-v4 is not evidence that factorization was
executed. The retained suite uses explicit v4 accepted-no-visible-effect
controls and explicit v8 error controls to keep acceptance, effect, and
eligibility rejection separate.

## Graph traversal confirmation

The separate `gql_surface` case applies
`FACTORIZED_MODE=FACTORIZE_LEFT` to the destination of a quantified graph
edge and compares it with an otherwise identical unhinted query. Both forms
returned plans at optimizer versions 1 through 8. The hinted and control
forms had the same two Aggregates and one Array Unnest in versions 1 through
4; in versions 5 through 8 the hinted form had three Aggregates and two Array
Unnests while the control remained unchanged. This independently confirms the
same v4/v5 plan-visibility boundary for graph traversal placement.

## Reproduction

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case factorized_mode \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/factorized_mode_expectations.json
```

Add `--optimizer-version-matrix` to submit all retained queries at versions 1
through 8. Version-prefixed labels do not match the unprefixed expectation
manifest, so that matrix is used as a vocabulary and acceptance-boundary gate;
the explicit v4/v5 and v1/v8 labels retain the core effect assertions.

These PLAN observations do not establish a latency, CPU, memory, network, or
bytes-processed benefit.
