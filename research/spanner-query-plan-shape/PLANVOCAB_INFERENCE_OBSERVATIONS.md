# Plan Vocabulary Inference Observations

Observed on 2026-07-18 and revalidated on 2026-08-11 with Spanner Omni through
`spanemuboost`.

This note records the live evidence used to extend the `planvocab` catalog and
to turn plausible operator-shape hypotheses into executable positive
expectations. It is an environment-specific observation, not a stable Spanner
contract.

Environment:

- `spanner_omni_image`: `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`
- `spanner_omni_image_digest`: `sha256:e98a088fa66d4a87dbb560d729bf21d998bb843f6018bd8dc118fe320e671886`
- `spanemuboost`: `github.com/apstndb/spanemuboost v0.4.0`
- `optimizer_version`: `not_pinned`
- `optimizer_statistics_package`: `not_pinned`

The inference cases and their positive expectations are checked with:

```sh
go build -o /tmp/planvocab-check ./plancontract/cmd/planvocab-check
(cd tools && go build -o /tmp/spanner-query-plan-shape ./spanner-query-plan-shape)
set -o pipefail
DOCKER_HOST=unix://$HOME/.colima/default/docker.sock \
TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock \
/tmp/spanner-query-plan-shape \
  --case planvocab_inference \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --expect tools/spanner-query-plan-shape/testdata/planvocab_inference_expectations.json
```

The run produced 17 plans, no query-analysis errors, no unknown-vocabulary
findings, and no expectation failures.

## Hypotheses and observations

| Probe | Hypothesis | Observation |
| --- | --- | --- |
| Distributed Cross Apply ordering | `ALLOW_DISTRIBUTED_MERGE=FALSE` might emit `order_preserving=false` | The default emitted `order_preserving=true`; the disabled case omitted the key rather than emitting `false`. |
| Right outer merge join | A `RIGHT OUTER JOIN` might retain `join_type=RIGHT_OUTER` | The optimizer normalized it to `join_type=LEFT_OUTER` and used a scalar `Right` link. |
| Unique-to-unique merge join | Unique inputs might emit `ONE_TO_ONE` | The observed shape still emitted `ONE_TO_MANY`. |
| Offset placement | Optimizer versions might place `Offset` on different nodes | Version 1 placed it on global `Sort Limit`; versions 3 and 8 placed it on global `Limit`. Local `Sort Limit` retained only `Limit`. |
| Hash outer join residual predicate | A non-key predicate should be separately observable | Both build-side choices emitted `Condition` and `Residual Condition`; the equality-only control did not emit `Residual Condition`. |
| Minor sort values | Multiple projected values might produce repeated `Value` links | The values were represented as one tuple-valued `Value` link. |
| Aggregate keys and aggregates | Multiple keys or aggregates might produce repeated `Key` and `Agg` links | Hash and stream aggregates each used one tuple-valued `Key` link and one tuple-valued `Agg` link. |

These negative results matter for the catalog: plausible enum values or repeated
links are not admitted merely by analogy. Only combinations observed in raw
plans or explicitly sourced from the pinned `apstndb/spanner-hacks`
operator notes are embedded.

## Full built-in corpus gate

All 20 selectors currently accepted by `spanner-query-plan-shape --case` were
run individually and streamed through `planvocab-check`. This includes the two
compatibility aliases, so their plans are intentionally counted twice.

| Selector | Successful plans | Query errors | Unknown findings |
| --- | ---: | ---: | ---: |
| `all` | 2 | 0 | 0 |
| `docs` | 53 | 0 | 0 |
| `optimizer_gaps` | 16 | 0 | 0 |
| `optimizer_unhinted_candidates` | 61 | 0 | 0 |
| `join_elimination` | 13 | 0 | 0 |
| `planvocab_inference` | 17 | 0 | 0 |
| `cte` | 8 | 0 | 0 |
| `dml` | 28 | 1 | 0 |
| `tvf` | 1 | 0 | 0 |
| `lock_hints` | 5 | 0 | 0 |
| `full_text_search` | 15 | 0 | 0 |
| `json_search` | 11 | 0 | 0 |
| `vector_search` | 6 | 0 | 0 |
| `function_hint` | 3 | 0 | 0 |
| `hint_matrix` | 40 | 0 | 0 |
| `statement_hint_query_matrix` | 1,718 | 402 | 0 |
| `join_matrix` | 36 | 0 | 0 |
| `subquery_join_hint_matrix` | 48 | 0 | 0 |
| `push_broadcast_hash_join` alias | 1 | 0 | 0 |
| `hash_join` alias | 1 | 0 | 0 |
| **Total** | **2,083** | **403** | **0** |

The 403 query errors are kept distinct from vocabulary findings. They are
invalid hint/query combinations in the matrix plus one DML analysis error; no
plan exists for `planvocab` to inspect in those cases.

## Query generator integration gate

The `cmd/spanner-query-gen` module has additional live Omni plan tests that do
not flow through `spanner-query-plan-shape`, including parameterized generated
queries and DML. Its `integration && omni` tests expose an opt-in,
test-only bridge through `SPANALYZER_PLANVOCAB_CHECK_BIN`. The bridge serializes
each raw plan to the external checker, avoiding a dependency from the querygen
module on the unpublished local `planvocab` package.

All three Omni integration suites passed with the bridge enabled. They made 31
plan checks across 30 distinct named cases; every checker invocation passed.
These tests used:

- `spanner_omni_image`: `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`
- `spanner_omni_image_digest`: `sha256:e98a088fa66d4a87dbb560d729bf21d998bb843f6018bd8dc118fe320e671886`
- `spanemuboost`: `github.com/apstndb/spanemuboost v0.4.6`

```sh
(cd cmd/spanner-query-gen && \
  SPANEMUBOOST_ENABLE_OMNI_TESTS=1 \
  SPANALYZER_OMNI_IMAGE=us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta \
  SPANALYZER_PLANVOCAB_CHECK_BIN=/tmp/planvocab-check \
  go test -tags='integration omni' \
    -run 'TestIntegration(QueryCodegenGeneratedSpannerQueriesRunOnOmni|PlanReportOperatorFamilyCoverageOnOmni|DMLOperatorFamilyCoverageOnOmni)$' \
    -count=1 ./...)
```
