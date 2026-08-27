# Hint Position Audit

Created: 2026-08-04

Status: **COMPLETE** for the spanalyzer GoogleSQL frontend, Cloud Spanner
Emulator, and Spanner Omni. Managed Spanner remains unverified because the
three connection environment variables were unset; no connection destination
is recorded here.

## Provenance

- Input audit: Cloud Spanner hint-position audit dated 2026-08-03.
- Cloud Spanner documentation snapshots in that audit were updated on
  2026-07-22 and retrieved on 2026-08-03.
- Generic GoogleSQL source reviewed by that audit:
  `google/googlesql@1f8aa333f4d6353cd3a64471fc83121df72df3f7`.
- spanalyzer frontend tested here: `github.com/goccy/go-googlesql@v0.3.0`.
- Emulator: `gcr.io/cloud-spanner-emulator/emulator:1.5.55`.
- Omni: `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta`,
  local image digest
  `sha256:e98a088fa66d4a87dbb560d729bf21d998bb843f6018bd8dc118fe320e671886`,
  revalidated 2026-08-11.

The integration probes submit statements directly through the Spanner Go
client. Memefish is used only to parse the fixture DDL before database setup;
it is not in the candidate-statement path.

### Emulator 1.5.56 revalidation (2026-08-27)

The repository-wide current-target run revalidated the audit on Emulator
`1.5.56`, `linux/arm64`, platform-manifest digest
`sha256:5b1e3607fe8574fb04144eeabfa54120559fb01968ffe3ffc0a9a8f6776fc454`.
Both `|> LOG @{a=1}` and `|> FINISH @{a=1}` now return `Pipe query syntax not
supported` at the pipe operator, before the hint. That result is classified as
`feature_unavailable`: it proves neither acceptance nor a position-specific
rejection of the hint. The executable audit keeps this state separate from
`accepted`, `rejected`, and transport-level `inconclusive` results. No managed
Spanner or Omni claim is changed by this Emulator-only grammar boundary.

## Classification

- **accepted**: execution succeeded, or a non-syntax error proved that parsing
  reached name resolution, hint validation, feature validation, or another
  later stage.
- **rejected**: the service returned `Syntax error`, one of the two targeted
  GQL path-placement errors, or the same-level set-operation restriction
  `Hints on set operations must appear on the first operation`.
- **inconclusive**: submission did not reach a capable service, or returned an
  `InvalidArgument` diagnostic that was neither a known position rejection nor
  evidence of later-stage validation. No final probe was inconclusive.

`AnalyzeQuery` is not used as the Emulator syntax oracle. By documented
design, the Cloud Spanner Emulator accepts the `PLAN` and `PROFILE` query modes
but does not return a query plan in either mode. The integration test therefore
uses the execution API for Emulator syntax classification. Omni supports query
plans, so the separate Omni probes below use `AnalyzeQuery` with `PLAN` for
plan-shape evidence. See the
[Cloud Spanner Emulator README](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/master/README.md#limitations-and-restrictions).

## Result matrix

| Position or restriction | Frontend | Emulator 1.5.55 | Omni 2026.r1-beta |
|---|---|---|---|
| SELECT hint | parse + Hint AST + unparse | accepted; unsupported synthetic key | accepted; unsupported synthetic key |
| ORDER BY hint | parse + Hint AST + unparse | accepted; unsupported synthetic key | accepted; unsupported synthetic key |
| First SQL set-operation hint | parse + Hint AST + unparse | executed | executed |
| Later hint in one same-level SQL set-operation chain | parser retains Hint AST; analyzer rejects | rejected | rejected |
| SQL EXISTS hint | parse + Hint AST + unparse | executed | executed |
| Subquery-IN hint | parse + Hint AST + unparse | executed | executed |
| IN value-list or UNNEST RHS hint | rejected | rejected | rejected |
| LIKE ANY/SOME/ALL subquery hints | parse + Hint AST + unparse | accepted; feature unsupported | accepted; feature unsupported |
| LIKE ANY/SOME/ALL value-list RHS hints | rejected | rejected | rejected |
| ANY/SOME/ALL quantified-comparison subquery hints | parse + Hint AST + unparse | accepted; feature unsupported | accepted; feature unsupported |
| ANY/SOME/ALL quantified-comparison value-list RHS hints | rejected | rejected | rejected |
| Window PARTITION BY hint | parse + Hint AST + unparse | accepted; analytics unsupported | accepted; built-in unsupported |
| TVF call hint | parse + Hint AST + unparse | accepted; TVF not found | accepted; TVF not found |
| GQL RETURN, WITH, ORDER BY, and VALUE hints | parse + Hint AST + unparse | accepted; unsupported synthetic key | accepted; unsupported synthetic key |
| GQL EXISTS hint | parse + Hint AST + unparse | executed | executed |
| GQL set-operation hint | rejected | rejected | rejected |
| Hint at the beginning of a GQL parenthesized path | rejected | rejected | rejected |
| Hint between two GQL edge patterns | rejected | rejected | rejected |
| GQL node-to-edge and subpath-to-edge traversal hints | parse + Hint AST + unparse | not separately re-run | accepted; plan-visible |
| GQL hint between comma-separated path patterns | parse + Hint AST + unparse | not separately re-run | accepted v1-v8; plan-visible from v3 |
| GQL hint on a later MATCH | parse + Hint AST + unparse | not separately re-run | MERGE accepted v1-v8; PUSH accepted v3-v8 |
| GQL subpath-to-node or subpath-to-subpath hint | grammar accepts | not separately re-run | accepted and plan-visible v1-v8 despite current documentation saying not allowed |
| Pipe LOG hint | parse + Hint AST + unparse | accepted; pipe syntax unsupported later | accepted; pipe syntax unsupported later |
| Pipe FINISH hint | rejected | rejected | rejected |
| INSERT target hint | parse + Hint AST + unparse | accepted; unsupported synthetic key | executed |
| PDML statement hint | parse + Hint AST + unparse | executed | executed |

The frontend also retains and round-trips generic hints on function calls and
DELETE/UPDATE targets. Their runtime positions were already covered by the
existing `function_hint` and DML plan cases, so the new runtime matrix focuses
on previously uncovered positions.

## Multi-hint combinations on Omni

The following observations come from `AnalyzeQuery(QueryMode=PLAN)` on the
pinned Omni image. They distinguish a plan-visible effect from syntax
acceptance: a successful plan does not by itself prove that every accepted
hint key affected optimizer behavior.

| Predicate and hint combination | Observed plan evidence | Grade |
|---|---|---|
| SQL `WHERE EXISTS` or `WHERE IN`; `JOIN_METHOD=HASH_JOIN`, `HASH_JOIN_BUILD_SIDE=BUILD_LEFT`, `HASH_JOIN_EXECUTION=ONE_PASS` | `Hash Join`, `join_type=BUILD_SEMI`; relational `Build`/`Probe`, scalar `Condition`, and variable-bearing scalar `Build` links | plan-visible |
| Same predicates with `HASH_JOIN_BUILD_SIDE=BUILD_RIGHT` | `Hash Join`, `join_type=PROBE_SEMI`; the variable-bearing scalar link is `Probe` | plan-visible |
| Same predicates with `JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE` | `Distributed Semi Apply` containing relational `Map`, scalar `Split Range`, and two variable-bearing scalar `Batch` links | plan-visible |
| Same predicates with `JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE` | row-at-a-time `Semi Apply` with relational `Map`; the top-level distributed apply transformation is absent from the observed plan | plan-visible pairwise contrast |
| Statement `OPTIMIZER_VERSION=8, FORCE_JOIN_ORDER=TRUE` plus the `WHERE EXISTS` hash/build-left/one-pass hints | the same `Hash Join` `BUILD_SEMI` shape | plan-visible coexistence |
| SQL set operation with `JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS` | successful ordinary `Distributed Union` / `Union All` plan | syntax accepted; no join effect visible |
| Scalar SQL `EXISTS` or `IN`, or GQL `RETURN EXISTS`, with hash/build-left/one-pass | successful plans retaining `Cross Apply`-family shapes rather than `Hash Join` | syntax accepted; requested hash effect not observed |
| `LIKE SOME/ALL` and quantified `SOME/ALL` with hash/build-left/one-pass | hint syntax reaches later feature validation, but the base feature is unsupported on the tested runtimes | syntax accepted; effect inconclusive |

The hash expectations prove the visible join family and build/probe
orientation selected for the complete hint combination. The observed plans do
not expose separate metadata that proves `HASH_JOIN_EXECUTION=ONE_PASS` itself
changed execution, so no per-key `ONE_PASS` effectiveness claim is made.

`HASH_JOIN_BUILD_SIDE` was also tried on the set-operation hint position and
was rejected as an unsupported hint. Consequently the durable set-operation
control uses only `JOIN_METHOD` plus `HASH_JOIN_EXECUTION`; it does not claim
that build-side orientation is available at that position.

The positive expectation manifest deliberately covers only the nine
plan-visible rows: both predicates across hash left/right and apply batch
true/false, plus statement/predicate coexistence. The four syntax-only controls
remain in the raw-plan stream and are still checked for unknown plan
vocabulary. The batch-true expectations require both observed `Batch` links,
which also records that multiplicity in the embedded catalog.

The later graph-specific matrix found a documentation/runtime discrepancy that
must not be collapsed into the generic rejected rows above. The shared grammar
permits a hint between successive `graph_path_factor` values, and the pinned
Omni runtime accepts and honors HASH between a subpath and node and between two
subpaths. Current Spanner graph-hint prose says those two relationships are not
allowed. The retained cases are therefore labeled `runtime-extension`; they
record pinned-Omni behavior without claiming official managed-service support.

## Reproduction

Frontend and case-table tests:

```sh
go test ./... -run 'TestGoogleSQLHintPositions|TestGoogleSQLSetOperationHint'
(cd tools && go test ./spanner-query-plan-shape -run TestLoadQueriesHintPositionAudit)
```

Runtime integration tests:

```sh
(cd tools && go test -tags=integration ./spanner-query-plan-shape \
  -run TestIntegrationHintPositionAuditOnEmulator -v)

(cd tools && \
  SPANEMUBOOST_ENABLE_OMNI_TESTS=1 \
  SPANALYZER_OMNI_IMAGE=us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta \
  go test -tags=integration,omni ./spanner-query-plan-shape \
  -run TestIntegrationHintPositionAuditOnOmni -v)
```

For exploratory PLAN output:

```sh
(cd tools && go run ./spanner-query-plan-shape \
  --case hint_position_audit \
  --output summary \
  --continue-on-error)
```

Reproduce the plan-visible multi-hint contracts on the pinned Omni image:

```sh
go build -o /tmp/planvocab-check ./plancontract/cmd/planvocab-check
go build -o /tmp/spanner-query-plan-shape ./tools/spanner-query-plan-shape
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case hint_position_combinations \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --expect tools/spanner-query-plan-shape/testdata/hint_position_combination_expectations.json
```

The later `gql_hint_surface` selector adds graph-specific plan-effect and
optimizer-version evidence that this placement inventory did not originally
enforce: nonlinear and quantified `FACTORIZE_BOTH`, statement/arm-local GQL
set-operation controls, correlated EXISTS acceptance-without-effect,
graph-element index/scan/seek composition, traversal MERGE/HASH/APPLY value
axes, official and runtime-extension traversal placements, index-union and
groupby-scan controls, edge BATCH v5/v6, and direct/statement GQL PUSH v2/v3
boundaries. See
[`GQL_HINT_VERSION_OBSERVATIONS_2026-08-11.md`](GQL_HINT_VERSION_OBSERVATIONS_2026-08-11.md).
