# GQL hint and optimizer-version observations (2026-08-11)

This note records graph-specific hint placement and plan-effect evidence. It is
an environment-bound `AnalyzeQuery(QueryMode=PLAN)` observation, not a
performance result or a managed Cloud Spanner compatibility promise.

## Evidence environment

- Runtime: Spanner Omni
  `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`, local
  image digest
  `sha256:e98a088fa66d4a87dbb560d729bf21d998bb843f6018bd8dc118fe320e671886`.
- Schema: the built-in `MusicGraph` over `Singers` and `Collaborations`.
- Retained selector: `gql_hint_surface`, 50 base queries.
- Default result: 47 plans, three expected errors, 93 operator expectations,
  zero expectation failures, and zero vocabulary findings.
- Optimizer matrix: 400 submissions across versions 1 through 8, with 362
  plans, 38 expected errors, and zero vocabulary findings.
- The focused Omni integration test compares raw `spannerpb.QueryPlan` values
  with `proto.Equal`; it also checks exact operator counts where equality alone
  would not attribute a hint effect.

## Factorized graph traversal

Both the quantified and nonlinear `FACTORIZED_MODE=FACTORIZE_BOTH` pairs were
accepted at every optimizer version.

- v1-v4: each hinted plan was byte-identical to its exact unhinted control.
- v5-v8: each hinted plan differed from its control and added factorized
  Aggregate/Array-Unnest materialization.

The nonlinear case is the documented graph placement between path-list
branches. Ordinary relational `FACTORIZE_BOTH` evidence therefore does not
stand in for this graph-specific result.

## GQL set-operation controls

A hint immediately after the GQL set operator is a syntax error at every
version: the parser expects `ALL` or `DISTINCT` there. Two legal alternatives
were plan-visible:

- statement-level `JOIN_METHOD=HASH_JOIN` replaced both graph-arm traversal
  applies with exactly two inner Hash Joins at v1-v8;
- separate arm-local HASH and APPLY hints produced one inner Hash Join and two
  Batch `Distributed Cross Apply` operators at v1-v8;
- statement-level MERGE selected two inner Merge Joins in the graph arms at
  v1-v8, while statement-level APPLY retained two distributed applies; and
- statement-level PUSH was rejected at v1-v2, then produced exactly two Push
  Broadcast Hash Join operators at v3-v8.

The statement-level hint changed the arm traversals; it did not prove control
of the set-comparison operator itself. The arm-local case proves per-arm
flexibility, not a hidden direct set-operator hint slot.

## Correlated graph subqueries

`HASH_JOIN`, build-side, and `MULTI_PASS` attached to correlated GQL `EXISTS`
were accepted at v1-v8, but every raw plan was byte-identical to the unhinted
control and contained no Hash Join. `ONE_PASS` was rejected at v1-v3, accepted
from v4, and likewise byte-identical to both `MULTI_PASS` and unhinted plans
when accepted. The correct classification is accepted with no plan-visible
effect, not build-side control.

`BATCH_MODE=FALSE` on the tested correlated GQL `VALUE` placement was rejected
as an unsupported hint at v1-v8. Because validation stopped at `BATCH_MODE`,
that combined negative did not isolate APPLY. The APPLY-only probe does:
`JOIN_METHOD` itself is also rejected at that VALUE placement in v1-v8.

## Scan, index, and PUSH controls

- graph-element `FORCE_INDEX=_BASE_TABLE` selected `Singers` as a TableScan;
- `FORCE_INDEX=SingersByFirstLastName` selected that IndexScan;
- `SCAN_METHOD=ROW` and `BATCH` inside `GRAPH_TABLE` produced the requested
  method on the `SingersByFirstLastName` element scan; and
- graph traversal `PUSH_BROADCAST_HASH_JOIN` was rejected at v1-v2, accepted
  from v3, and produced one Batch `Push Broadcast Hash Join` plus its inner
  Hash Join.

These outcomes were enforced across v1-v8 where applicable, rather than
inferred from relational-query boundaries.

Direct graph traversal MERGE is accepted and plan-visible at v1-v8; it does
not have a v3 acceptance boundary in this pinned runtime. The same matrix also
shows that:

- `FORCE_INDEX=SingersByFirstLastName` composes with ROW/BATCH `SCAN_METHOD`
  on the same element and preserves both target/type and method metadata;
- the same forced element makes `COLUMNAR` and `NO_COLUMNAR` independently
  plan-visible at v1-v8: COLUMNAR records `scan_method=Batch,
  scan_format=Columnar`, NO_COLUMNAR records `scan_method=Automatic,
  scan_format=Row`, and the exact no-hint control has no `scan_format` key;
- graph-element `SEEKABLE_KEY_SIZE=0/1` composes with the forced index and is
  recorded on the matching Filter Scan;
- HASH build-left/build-right direct traversal plans differ at every version,
  represented by Build/Probe topology rather than a metadata value; and
- APPLY BATCH TRUE/FALSE is accepted throughout and yields two versus one
  Distributed Cross Apply operators. TRUE becomes identical to the unhinted
  choice at v6-v8, which is accepted with no additional visible effect.

The expanded element matrix adds three independently controlled boundaries:

- `INDEX_STRATEGY=FORCE_INDEX_UNION` forces two named index branches plus
  `Union All` and Aggregate at v1-v8; the unhinted control adopts the same
  operator families only at v7-v8;
- `GROUPBY_SCAN_OPTIMIZATION=TRUE` changes the forced index scan method to
  `Row`, while FALSE is byte-identical to the unhinted `Automatic` control at
  every version; and
- edge-element `SCAN_METHOD=ROW` succeeds at v1-v8, while BATCH is rejected on
  the right side of apply join at v1-v5 and succeeds with
  `scan_target=Collaborations, scan_method=Batch` at v6-v8.

## Traversal-hint placement evidence

The official graph-hint prose generally describes one key/value pair per hint.
Retained graph cases that combine method with build side, batching, scan method,
or factorization therefore document pinned-Omni composition behavior; they do
not broaden the portable graph-hint contract.

Official node-to-edge HASH and subpath-to-edge APPLY placements both changed
their exact controls at v1-v8. A hint between comma-separated path patterns
was accepted without effect at v1-v2, then selected one inner Hash Join from
v3 onward. A hint on the second MATCH selected a `MANY_TO_MANY` Merge Join at
v1-v8; PUSH kept the familiar v2/v3 acceptance boundary.

The pinned Omni runtime also accepts and honors HASH between a subpath and a
node and between two subpaths. In both cases the unhinted controls use apply
families and the hinted plans contain an inner Hash Join throughout v1-v8.
Current official graph-hint prose says these two placements are not allowed,
so the cases are labeled `runtime-extension` and are evidence about this
runtime only, not an official managed-Spanner compatibility claim. The shared
GoogleSQL grammar admits them because both node and parenthesized subpath are
`graph_path_factor` values.

## Reproduction

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case gql_hint_surface \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/gql_hint_surface_expectations.json
```

Use `--optimizer-version-matrix` without the unprefixed expectation manifest
for the 400-submission vocabulary matrix. Run the enforced boundary/equality
test with:

```sh
SPANEMUBOOST_ENABLE_OMNI_TESTS=1 \
SPANALYZER_OMNI_IMAGE=us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta \
go test -tags='integration omni' ./tools/spanner-query-plan-shape \
  -run '^TestIntegrationGQLHintSurfaceVersionBoundariesOnOmni$' -count=1
```
