# Spanner Omni 2026.r2.1-beta and managed-Spanner PLAN differences (2026-08-24)

This note records a fresh, destination-redacted comparison between managed
Spanner and Spanner Omni. It is observational evidence, not a portability
contract.

## Environment and source baseline

- Spanner Omni image:
  `us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta`
- Pulled image digest:
  `sha256:b0f5ca3876391bf3aaa9302b0a6e872414874da379c17ab10b95f549029c1aad`
- Managed Spanner: read-only `QueryMode=PLAN` probes through the configured
  managed-service profile. Destination identifiers are intentionally omitted.
- Official release note baseline: the August 20, 2026 release announces
  `2026.r2.1-beta`, editions, a standalone server package, and flag
  deprecations. It does not enumerate SQL or optimizer changes.
- Official pipe syntax baseline: the current Spanner reference lists 15 pipe
  operators: `SELECT`, `EXTEND`, `SET`, `DROP`, `RENAME`, `AS`, `WHERE`,
  `AGGREGATE`, `JOIN`, `ORDER BY`, `LIMIT`, `UNION`, `INTERSECT`, `EXCEPT`,
  and `TABLESAMPLE`.

Sources:

- [Spanner Omni release notes](https://docs.cloud.google.com/spanner-omni/release-notes)
- [Download Spanner Omni](https://docs.cloud.google.com/spanner-omni/download)
- [Spanner pipe query syntax](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax)

## Newly closed Omni gaps

### Core pipe syntax

The earlier `2026.r1-beta` observation returned `Pipe query syntax not
supported`. On `2026.r2.1-beta`, all 15 documented operators returned plans
with optimizer versions 1 through 9. The same 15 SQL texts returned plans on
managed Spanner.

The dedicated `pipe_surface` selector retains one query per documented
operator. It uses `TABLESAMPLE BERNOULLI`, because Spanner rejects the
reference page's `SYSTEM (1 PERCENT)` example as an unsupported sampling
algorithm even though the pipe operator itself is supported.

The version matrix also exposes a physical boundary that syntax-only probes
would miss: the retained `INTERSECT DISTINCT` pipe query uses Hash Join at
versions 1 through 4 and an apply-based shape from version 5 onward in the
standalone probe. The expectation manifest intentionally pins the unhinted
shape, while the integration test requires a non-vacuous Aggregate across all
versions because empty-schema cost choices can select either join family.

`ASSERT`, `DESCRIBE`, `IF`, and `FINISH` are not among the 15 documented
operators. Both managed Spanner and `2026.r2.1-beta` now parse their positions
and return explicit `Pipe ... not supported` capability errors. This replaces
the older Omni parser-level `Unexpected FROM` evidence for the first three;
it does not make those operators supported. The pinned Emulator 1.5.55 still
rejects `FINISH` at parsing, so the hint-position test retains that result as
an environment-specific boundary rather than changing the shared expectation.

### Optimizer version 9

`2026.r2.1-beta` accepts `OPTIMIZER_VERSION=9` and rejects version 10 with
`Query optimizer version: 10 is not supported`, matching the fresh managed
boundary.

The no-LIMIT distributed-apply query from the managed v9 research note now
reproduces on Omni:

- v8: `Distributed Cross Apply` has `execution_method=Row`;
- v9: it has `execution_method=Batch`;
- the v9 `Create Batch` declares an `__row_id` field whose child is
  `Constant <typed null>`; and
- the v9 apply output contains a `restored_*` variable.

The LIMIT control remains deliberately non-contractual. In this refresh,
managed Spanner used the Batch DCA mechanism while Omni used a row-oriented
Cross Apply shape. This is consistent with prior evidence that LIMIT is not a
stable suppression rule and demonstrates a current managed/Omni plan-choice
difference.

The v9 index-union candidate also produced an `Aggregate > Union All` plan on
Omni. This is retained as a positive candidate, not attributed solely to v9
without a controlled before/after result.

### DML shape boundary

Extending the existing DML matrix to v9 found one new compact-tree partition:
`DELETE ... THEN RETURN` differs between v8 and v9. Both versions retain
`Apply Mutations(operation_type=DELETE, table=Singers)`, but v9 adds
`DataBlockToRow` above a Batch `Distributed Cross Apply` and replaces the
map-side `Cross Apply` input with `RowToDataBlock`. The `optimizer_v9` selector
retains v8 and v9 controls and requires their compact trees to differ.

## Retained probes

- `--case pipe_surface`: all 15 documented pipe operators.
- `--case optimizer_v9`: v8/v9 DCA pruning, LIMIT control, index-union
  candidate, v8/v9 `DELETE ... THEN RETURN`, and the v10 rejection boundary.
- The general optimizer-version matrix and focused Omni integration tests now
  use versions 1 through 9.
- The existing `google_sql_surface` core-pipe case is now a positive plan
  expectation.
- Rewriter and hint-position probes retain the explicit unsupported errors for
  `ASSERT`, `DESCRIBE`, `IF`, and `FINISH`.

## Broader refresh result

The existing Omni integration suites were rerun against `2026.r2.1-beta` with
their optimizer matrices extended through v9. Apart from the newly recorded
pipe, DCA, and DML partitions above, their asserted plan and capability
boundaries remained valid.
