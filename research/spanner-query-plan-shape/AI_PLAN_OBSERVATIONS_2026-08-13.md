# AI Function PLAN Observations (2026-08-13)

This note records destination-redacted, read-only managed-Spanner
`AnalyzeQuery(QueryMode=PLAN)` observations. No query was executed and no
Agent Platform inference result was requested. PLAN acceptance proves the
current service can analyze and plan the expressions, not that external model
execution is configured.

## Probe set

The `ai_plan` selector pairs each documented function with a scalar control:

- `AI.CLASSIFY(...)` projection versus `CASE`;
- `AI.IF(...)` predicate after the same primary-key predicate versus a scalar
  length predicate;
- `AI.SCORE(...)` top-k ranking versus scalar length ranking.

All six queries returned plans at optimizer versions 1 through 8 and at the
unhinted default. The full raw default plans are retained as destination-free
fixtures under `cmd/spanner-query-gen/testdata/plan_fixtures/`.

Pinned Spanner Omni `2026.r1-beta` differs: each AI candidate returned an
empty-detail gRPC `Internal` error at every optimizer version from 1 through
8, while every scalar control returned a plan. The focused Omni test retains
this as an environment capability boundary. It does not reinterpret the
managed positive plans as unsupported.

## Physical lowering

The plans do not expose the public AI function name as a scalar `Function`
description. Each AI call lowers to a `TVF` node with `Referenced`, `Output`,
and `PassThroughVars` links, and the result is carried as a `$udf` variable.
The controls contain no TVF.

- `AI.CLASSIFY` adds the TVF and a Compute to the common table-scan/seek/global
  Limit shell.
- `AI.IF` preserves the ordinary `SingerId = 1` `Seek Condition`, then applies
  the AI result through a separate `Filter` `Condition`. Its scalar control is
  instead a `Filter Scan` `Residual Condition`.
- `AI.SCORE` adds the TVF and `VerifyDeterminism` before the global
  `Sort Limit`. The scalar ranking control has the same top-k family but no
  TVF or determinism guard.

Default plans were byte-equal to explicit v7 and v8 plans. `AI.IF` and its
control shared one plan per query across v1-v6 and another across v7-v8. The
classification and scoring pairs used the v1-v4 shape again at v6, had a
distinct v5 shape, and used a third shape at v7-v8. These are optimizer-shape
partitions, not function-support boundaries.

The retained fixture SHA-256 values are:

- `managed_ai_classify_projection.json`:
  `f14b40adf7e2b435d7741f9d2da749b50e93a67d8195df3fcc2810bfff472a6e`;
- `managed_ai_classify_case_control.json`:
  `0d1c491e5962fe87a5ba8ac55c8226d69f90b8f0d7362a5dbfa60b98d9ac4088`;
- `managed_ai_if_filter.json`:
  `b72301a18641a3820fcfdea8cdbe1fc0f0a157c354557a3bfb8cfea06e7bbf58`;
- `managed_ai_if_scalar_filter_control.json`:
  `f4b6bc65141e877a96ae71722e087622d5302f87783897289d15bef1878b0b9e`;
- `managed_ai_score_order_limit.json`:
  `67c5fa7df158e23791e6f934a2b7bdc26a98ad1d85a1ad8cd6958f323433b7f9`;
- `managed_ai_score_scalar_order_limit_control.json`:
  `e90937e0babc94a2e28c08220856985536933a378d9946c23ab9528d256fb572`.
