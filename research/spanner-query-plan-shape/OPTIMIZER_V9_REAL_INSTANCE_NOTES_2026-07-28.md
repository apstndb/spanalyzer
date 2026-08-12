# Optimizer version 9: real-instance observations and planned Omni probes (2026-07-28)

Purpose: preserve the first observed evidence of optimizer version 9 behavior
so the probe implementation is ready the moment a Spanner Omni release ships
v9. Everything in "Observations" is `observed` on real Cloud Spanner; the
"Planned probes" section is the implementation sketch.

## Environment

- Real Cloud Spanner (not Omni), reached through the user-level managed-probe
  profile. Destination identifiers are intentionally not recorded.
- Access: read-only query and PLAN probes through the
  `apstndb-spanner-syntax-verify` managed runner; no DDL or writes.
- Tools: `spanner-mycli` v0.33.1 and the Cloud Spanner API.
- Docs baseline: query-optimizer/versions page, "Version 9: July 21st, 2026
  (latest)"; default remains 8.

## Observations (real Spanner, 2026-07-28)

### 1. Version acceptance boundary

- `@{OPTIMIZER_VERSION=9}` is accepted.
- `@{OPTIMIZER_VERSION=10}` fails with
  `InvalidArgument ... Query optimizer version: 10 is not supported`.

### 2. Distributed-apply input column pruning (the headline v9 item)

Query (no LIMIT):

```sql
SELECT s.FirstName, c.ConcertDate
FROM Concerts AS c JOIN Singers AS s ON s.SingerId = c.SingerId
```

v8 shape: root `Distributed Cross Apply <Row>`; map side ends in
`Serialize Result` (final rows assembled remotely).

v9 shape: root `Serialize Result > DataBlockToRow >`
`Distributed Cross Apply <Batch>`; map side returns blocks via
`RowToDataBlock` (final rows assembled locally).

The pruning itself is visible in the Create Batch variable list — what is
actually shipped to the map side:

| | v8 `Create Batch` | v9 `Create Batch` |
| --- | --- | --- |
| batch fields | `v2.Batch.FirstName`, `v2.Batch.SingerId_1` | `v5.Batch.SingerId_1`, `v5.Batch.__row_id` |
| field defining exprs | `Reference $FirstName`, `Reference $SingerId_1` | `Reference $SingerId_1`, **`Constant <typed null>`** |

Mechanism, fully visible in the v9 raw plan:

- `__row_id` is declared as a batch field on Create Batch whose defining
  expression is `Constant <typed null>` — a schema-slot placeholder with no
  plan-level computation. The actual identifier is runtime-materialized
  (inferred: the block row ordinal); no `Function` or id-assigning operator
  appears.
- The map side reads back `batched_SingerId_1` + `batched___row_id` (Batch
  Scan), seeks `ConcertsBySingerId`, and returns `ConcertDate` together with
  the row id.
- The DCA re-associates map results with the retained input block by row id:
  its Batch-typed output `restored_FirstName'` is defined by
  `Reference $FirstName` against the retained input batch. Non-key input
  columns never cross the network.

### 3. LIMIT suppresses the new shape (documented boundary confirmed)

The same query with `LIMIT 5` produces byte-identical v8/v9 plans (row-mode
DCA, map-side Serialize Result). Matches the doc: prefer DA "if there is no
`LIMIT` above the apply".

### 4. Probes with no visible v8→v9 change on this schema/data

- Index-union aggressiveness: `WHERE s.FirstName = 'Alice' OR s.LastName =
  'Smith'` with both index directions available — identical plans.
- DCA column pruning on `Albums JOIN Songs` (both key columns projected) —
  identical plans (nothing prunable).

### 5. Default-version rollout is per-database and can lag

Unhinted `EXPLAIN ANALYZE` reported optimizer version **7** on one configured
database and **8** on another; neither database has an `optimizer_version`
database option set and the client sets no session default. "default = 8"
cannot be assumed per-database.

## Planned probes once an Omni release ships v9

Preconditions: bump the version-matrix upper bound (currently 8) and the
acceptance-error expectation (`10 is not supported` becomes the new upper
probe) wherever the harness pins them; then add an `optimizer-gaps/v9/`
family next to the existing per-version gap probes in
`tools/spanner-query-plan-shape/docs_cases.go`:

- `optimizer-gaps/v9/dca-input-column-pruning`: the section-2 query shape —
  input side must read at least one non-key, non-output-join column through a
  covering index, map side a secondary-index seek, no LIMIT. Expected v9
  signature: `Distributed Cross Apply` with `execution_method=Batch`, a
  Create Batch field defined by `Constant <typed null>` (`__row_id`), and
  `restored_*` variables on the DCA.
- `optimizer-gaps/v9/dca-input-column-pruning-limit-control`: same query
  `LIMIT 5`; expected to match the v8 shape (control).
- `optimizer-gaps/v9/index-union-aggressiveness`: the section-4 OR query; the
  real instance showed no shape change on this data — run it anyway on Omni
  and record whichever way it lands (empty-stats caveat applies).
- Optional acceptance pair mirroring section 1 if the harness gains an
  expected-error case kind.

planvocab impact to check when the fixtures land:

- `execution_method=Batch` on `Distributed Cross Apply` (verify the catalog's
  `execution_method` common-metadata enum already covers `Batch` for this
  operator's fingerprint path).
- The `Constant <typed null>` batch-field pattern and `__row_id`-style
  variables are scalar-side vocabulary; today's planvocab scope is
  relational-node metadata/links, so no catalog change is required, but the
  fixture will exercise the new shape end to end.
- Candidate raw fixture: the section-2 query against the docsDDL
  `Singers`/`Concerts` schema, captured at v8 and v9, added to
  `cmd/spanner-query-gen/testdata/plan_fixtures/` so the generator mirrors it
  into the planvocab gate.

Raw v8/v9 plans for section 2 were captured from the real instance on
2026-07-28; the variable tables above embed the load-bearing parts. Re-derive
with the managed runner and configured user-level profile; do not place the
destination identifiers in repository files or probe output.

## Revalidation status (2026-08-12)

The managed runner successfully executed read-only result and compact PLAN
probes earlier in the revalidation session. A later attempt to re-fetch this
query's raw v8/v9 nodes failed with endpoint-wide `DeadlineExceeded`, and the
same failure reproduced with `SELECT 1`. That failure is transport evidence,
not a query or optimizer rejection. The detailed DCA comparison above remains
the 2026-07-28 observation until a successful raw-node recheck is captured.
