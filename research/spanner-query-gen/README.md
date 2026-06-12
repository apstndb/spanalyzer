# spanner-query-gen Research

Non-normative design notes and verification logs for
`cmd/spanner-query-gen`. The source of truth for current v1alpha behavior is
[`cmd/spanner-query-gen/`](../../cmd/spanner-query-gen/) documentation,
[`schemas/`](../../schemas/), and the test suite.

## Files

- [`PLAN_CONTRACT_CANDIDATES.md`](PLAN_CONTRACT_CANDIDATES.md): candidate
  query-plan contracts derived from query-optimization practice. The
  implemented contract surface is documented in
  [`cmd/spanner-query-gen/PLAN_CONTRACTS.md`](../../cmd/spanner-query-gen/PLAN_CONTRACTS.md).
- [`OPTIONAL_PARAMS_DESIGN.md`](OPTIONAL_PARAMS_DESIGN.md): design for the
  optional query parameter support integrated into v1alpha.
- [`OPTIONAL_PARAMS_PRIOR_ART.md`](OPTIONAL_PARAMS_PRIOR_ART.md): prior-art
  survey behind that design.
- [`PLAN_REPORT_OPERATOR_COVERAGE_2026-06-12.md`](PLAN_REPORT_OPERATOR_COVERAGE_2026-06-12.md):
  the 2026-06-12 Omni verification session — operator family coverage,
  classifier defects found and fixed, optimizer behavior observations
  (join elimination, interval discretization, aggregate determinism), and
  Omni platform checks.
