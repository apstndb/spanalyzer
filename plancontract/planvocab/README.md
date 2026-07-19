# planvocab

`planvocab` inspects raw Spanner `QueryPlan` values for operator metadata and
child-link vocabulary that is not covered by the embedded observational
catalog. See the package documentation and `ExampleInspect` for the API.

The catalog is a `v0alpha1` observation snapshot, not a stable Spanner wire
contract. It records its `spanner-hacks` revision/blob and SHA-256 digests for
the checked-in evidence, schema, and raw plan fixtures used by the drift gates.
Identifier-like metadata is redacted before fingerprinting or logging.
The generator also mirrors only the PLAN payloads from querygen's raw fixtures
into `testdata/fixture_plans.json`, so the lightweight module can run its gate
without depending on an unpublished sibling-module version.
The `v0alpha1` token intentionally identifies this catalog-specific pre-v1
format; it is independent of the query generator's `v1alpha` report schemas.

Edit `catalog_source.json`, then regenerate and verify from the `tools` module:

```sh
go run ./planvocab-gen
go run ./planvocab-gen --check
```

The generated format is described by
[`schemas/spanalyzer.planvocab.v0alpha1.schema.json`](../../schemas/spanalyzer.planvocab.v0alpha1.schema.json).

To check raw plan envelopes without adding a `planvocab` dependency to the
producer module, stream the JSON output into the checker from the repository
root:

```sh
go build -o /tmp/planvocab-check ./plancontract/cmd/planvocab-check
go build -o /tmp/spanner-query-plan-shape ./tools/spanner-query-plan-shape
set -o pipefail
/tmp/spanner-query-plan-shape --case docs --output json --continue-on-error \
  | /tmp/planvocab-check --allow-query-errors
```

The checker accepts either the single-envelope files under
`cmd/spanner-query-gen/testdata/plan_fixtures` or envelope arrays emitted by
`spanner-query-plan-shape`. Query-analysis errors are counted separately from
vocabulary findings, and their backend error text is not copied into the
report. When multiple input files are passed, expectation labels share one
namespace; every envelope carrying a matching label must satisfy that label's
patterns.

The query generator's `integration && omni` tests also have an opt-in bridge
for plans produced inside that module. Build the checker, then set
`SPANALYZER_PLANVOCAB_CHECK_BIN` to its absolute path when running those tests.
This keeps the standalone querygen module independent of the unpublished local
package while making every successful live plan pass the same vocabulary gate.

Positive expectations use `FindMatchingOperators` semantics and are checked on
the same operator node. The repository's inference matrix can be reproduced
with:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case planvocab_inference \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --expect tools/spanner-query-plan-shape/testdata/planvocab_inference_expectations.json
```

The expectation format is described by
[`schemas/spanalyzer.planvocab-expectations.v0alpha1.schema.json`](../../schemas/spanalyzer.planvocab-expectations.v0alpha1.schema.json).
