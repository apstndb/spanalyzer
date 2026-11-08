---
type: Research Note
title: "Managed Spanner, Omni r4, and Emulator comparison, 2026-10-05"
description: "Current official references and bounded runtime probes, separating engine differences, documentation discrepancies, and selected analyzer projections."
tags: [spanner, spanner-omni, emulator, verification, queues, postgresql, catalog]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: execution
    resource: evidence/cross-engine-20261005/receipt.json
    title: "Runtime identities, producer hashes, execution and cleanup receipts"
  - id: results
    resource: evidence/cross-engine-20261005/summary.json
    title: "Derived comparison with links to raw probe records"
  - id: documentation
    resource: evidence/cross-engine-20261005/documentation.json
    title: "Full official document retrieval hashes and source URLs"
  - id: managed-release
    resource: https://docs.cloud.google.com/spanner/docs/release-notes
    title: "Managed Spanner release notes"
  - id: omni-release
    resource: https://docs.cloud.google.com/spanner-omni/release-notes
    title: "Spanner Omni release notes"
  - id: emulator-release
    resource: https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/releases/tag/v1.5.58
    title: "Latest published Emulator release in the fetched release list"
  - id: previous-managed
    resource: managed-revalidation-20261005.md
    title: "Earlier October 5 JST managed integration and catalog verification"
  - id: previous-omni
    resource: spanner-omni-2026.r4-lts-verification.md
    title: "Omni r4 integration, startup and TLS verification"
---

# Engine comparison, October 5

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

This is the entry point for the latest compared samples. The observations below
were rerun on October 5, 2026 UTC/JST. Managed Spanner and Omni r4 agree on the
tested September SQL and queue behavior, but differ in `SPANNER_SYS` and graph
metadata. Emulator is a separate compatibility target; its results cannot
establish managed or Omni planning behavior.

## Managed follow-up, October 7

The [new managed revalidation](managed-revalidation-20261007.md) adds a later
managed sample without rerunning the container engines. SPANNER_SYS now
advertises 51 tables / 598 columns, with tuples equal to the retained Omni r4
capture. However, ACTIVE_BATCH_OPERATIONS scans return Internal or table not
found; LIMIT 0 alone succeeds. The advertisement difference below has changed,
while successful execution parity remains unestablished.

Managed graph metadata now emits allow_view_with_volatile_column=FALSE, and
explicit values return Unimplemented instead of unknown-option errors, matching
the earlier r4 outcomes. PostgreSQL/queue controls retain their earlier codes.
The managed drift gate also fails on SQL NULL transaction averages; the other
three managed integration tests pass. The matrices below remain October 5
observations. Keep each engine's sample date when reading the follow-up.

## Managed follow-up, October 9

The [next managed sample](managed-revalidation-20261009.md) successfully runs
ACTIVE_BATCH_OPERATIONS star/count/individual-column scans, returning no batch
rows. Catalog tuples remain 51 tables / 598 columns. This changes the October
7 queryability observation without proving populated batch behavior or a fleet
rollout boundary. Graph, PostgreSQL and queue codes match October 7. The
managed drift gate now reports the table's unregistered decoder shape; focused
NULL-average queries show that the earlier nullable decoding boundary remains.
The Omni/Emulator samples in this comparison were not rerun.

## Runtime and source identity

| Target | Identity and scope |
| --- | --- |
| Managed Spanner | One configured GoogleSQL verification database and an owned temporary PostgreSQL database; rolling service build unavailable; optimizer 1-9 advertised, default 9 |
| Spanner Omni | `2026.r4-lts`, Developer container, `linux/arm64`, platform manifest `sha256:9f10586c5de366bec5c0ff6cf542a67c21c8363e8a12a3deb9c372a07c9aa8b6` |
| Emulator | `1.5.58`, `linux/arm64`, platform manifest `sha256:5f42f55ad4129ea0d86cf60f0d385e8a6be60a7e8e0cc791adcdedc8982bcb55` |

The official release and tag lists both identify Emulator v1.5.58, published
September 15. The registry index was fetched independently to select its native
arm64 manifest. The [September 15 investigation](fts-index-choice-and-emulator-1.5.58-20260915.md)
used the amd64 manifest; that older run is not silently relabeled as arm64.

Probe source starts from `c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c` plus the
existing dirty working snapshot. Go is 1.26.8. Feature probes use the existing
Spanner Go v1.91.0 dependency; managed read-only/catalog checks use the independent
survey module with Spanner v1.88.0. Container probes use the retained isolated
spanemuboost v0.4.7 copy with the r4 `--listen-addresses=0.0.0.0` startup fix.
Source, binary, and build-information hashes are retained in the receipt.

The initial Emulator attempt failed before SQL because the reused Omni setup
enabled only database creation. Emulator-specific copies enable instance and
database creation; their subsequent probes and cleanup completed successfully.
The failed bootstrap receipt is retained separately from feature outcomes.

## Documentation checked

Full official Markdown was retrieved with `dkcli`, with raw GitHub release/tag
API bodies retained separately. These are direct source readings, not search
summaries. The latest managed release entry remains September 18. Omni's
September 30 r4 entry states coverage through September 18, including queues.
Neither date is used to infer an implementation boundary.

| Official reference | Claim checked against runtime |
| --- | --- |
| [Managed release notes](https://docs.cloud.google.com/spanner/docs/release-notes) | September 18 automatic parameterization; September 17 queues; September 11 graph options; September 3 PostgreSQL matching; September 2 sampling |
| [PostgreSQL operators](https://docs.cloud.google.com/spanner/docs/reference/postgresql/operators) and [functions](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions) | Operator spellings and separately listed `pg.ilike` / `pg.not_ilike` functions |
| [PostgreSQL query syntax](https://docs.cloud.google.com/spanner/docs/reference/postgresql/query-syntax) | Lists `BERNOULLI` and `SPANNER.RESERVOIR` as the supported sampling methods; `SYSTEM` is a runtime negative control |
| [Queue usage](https://docs.cloud.google.com/spanner/docs/queues/queues-using) | Go scalar-inside-array lease argument and the invalid-token no-row description |
| [Graph schema](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements) and [graph views](https://docs.cloud.google.com/spanner/docs/graph/graph-with-views-overview) | Documented key validation; neither fetched page names `allow_view_with_volatile_column` |
| [Emulator limitations](https://docs.cloud.google.com/spanner/docs/emulator) | Empty query plans, unsupported auth/TLS, ignored ANALYZE, and different transaction/error behavior |

The full fetched pages and their retrieval audit are retained in the durable
task directory recorded in [documentation.json](evidence/cross-engine-20261005/documentation.json).
Two guessed graph URLs returned 404 and were excluded; the graph references
linked above were fetched successfully.

## Catalogs and selected projections

Fresh catalog probes retain all advertised table/column/type/ordinal tuples.
These small comparison probes are not replacements for the versioned survey
capture contracts or their selected manifests.

| Surface | Managed sample | Omni r4 | Emulator 1.5.58 |
| --- | --- | --- | --- |
| INFORMATION_SCHEMA | 48 tables / 310 columns | 48 / 310; identical tuples | 28 / 198 |
| SPANNER_SYS advertisement | 50 / 587 | 51 / 598; the extra 11 columns belong to `ACTIVE_BATCH_OPERATIONS` | Zero column tuples advertised; see queryability caveat below |
| `ACTIVE_BATCH_OPERATIONS` | Not advertised; SELECT returns `InvalidArgument`, table not found | Advertised; SELECT succeeds with zero rows | Not advertised; SELECT returns `InvalidArgument`, table not found |
| `COLUMN_OPTIONS` and `LOCALITY_GROUP_OPTIONS`.`IS_INHERITED` | Advertised; LIMIT 0 probes succeed | Advertised | Neither column advertised |

Emulator advertises no SPANNER_SYS columns through the tested discovery query,
yet `SELECT * FROM SPANNER_SYS.SUPPORTED_OPTIMIZER_VERSIONS` succeeds and returns
version 42 as the default. This does not identify a managed optimizer version:
Emulator rejects `OPTIMIZER_VERSION` hints and returns no plan nodes. Absence
from discovery is therefore not a blanket statement that every SPANNER_SYS
query fails.

The analyzer's selected INFORMATION_SCHEMA projection still uses the September
16 managed capture. Its selected SPANNER_SYS projection still uses the August
25 managed/r2.1 common surface, 50 tables / 539 columns. The 48 distribution
columns remain `absent_both` in that selected manifest even though newer
managed and Omni captures advertise them. This is a selection fact, not current
runtime absence. `ACTIVE_BATCH_OPERATIONS` is not yet in the 51-table survey
decoder registry; its union with the r4 capture contains 52 table names.

See the [INFORMATION_SCHEMA projection](../../observations/information-schema-managed-primary-catalog.md)
and [SPANNER_SYS projection](../../observations/spanner-sys-live-primary-catalog.md)
for the deliberate update boundary. This comparison does not change runtime
pins, decoder structs, dependencies, or selected manifests.

## SQL and graph behavior

Results below distinguish an executable query from a missing function, an
unsupported operation, and metadata that merely exposes a name.

| Bounded case | Managed sample | Omni r4 | Emulator 1.5.58 |
| --- | --- | --- | --- |
| GoogleSQL automatic parameterization, literal primary-key lookup | DEFAULT/AUTO/ON expose PROFILE `auto_parameters`; OFF does not; invalid value rejected | Same | Unhinted query returns rows; all explicit `AUTO_PARAMETER` values rejected as unsupported hints; no plan nodes or `auto_parameters` |
| `APPROX_COUNT_DISTINCT` on two distinct names, optimizer 1-9 | Returns 2 at each requested version | Same | All nine `OPTIMIZER_VERSION` hints rejected before aggregate execution |
| Unhinted approximate aggregate, `[1,1,2,NULL]` | Returns 2 | Returns 2 | `Unimplemented`, unsupported built-in function |
| Graph over a join view | Default/true uniqueness validation fails; false creates and reads the graph | Same | Default/true/false all create; MATCH returns a count of 2; validation differs |
| Graph label/property `description` and `synonyms` | DDL, metadata, and export succeed | Same | Combined DDL rejected at the label OPTIONS clause; property-only case not isolated |
| Explicit `allow_view_with_volatile_column=false/true` | `InvalidArgument`, unknown graph option | `Unimplemented`, unsupported graph option | Both accepted; permissive-option controls below prevent interpreting this as implemented semantics |
| Graph over a `GENERATE_UUID()` view, volatile option omitted and uniqueness validation false | Creates and returns rows | Same | Creates and returns a row |
| `= ANY/SOME/ALL` with subquery/list/UNNEST, plus quantified LIKE subquery: 12 forms | Explicit unsupported rejection | Explicit unsupported rejection | Explicit unsupported rejection |
| PostgreSQL `ILIKE`, `NOT ILIKE`, `~~*`, `!~~*` | `true,false,true,false` | Same | Same |
| PostgreSQL direct `pg.ilike`, `pg.not_ilike`, `spanner.ilike`, `spanner.not_ilike` calls | `InvalidArgument`, function not found | Same | Same |
| PostgreSQL BERNOULLI(0/100), RESERVOIR(0/1/100) over two rows | 0/2 and 0/1/2 rows | Same | Same |
| PostgreSQL SYSTEM(0/100) | `Unimplemented` | Same | Same |

The PostgreSQL function reference still lists the two `pg.*` functions that
all three samples reject. Operator support must not be described
as absent merely because direct function calls fail.

Omni r4 emits `allow_view_with_volatile_column=FALSE` in graph metadata;
managed does not. Both reject explicit values, while the tested volatile-view
graph works with the option omitted. Its intended semantics remain unverified.
The omitted-option managed fixture has two rows and the container fixture one;
the comparison establishes successful execution, not byte-identical output.

Emulator also accepts `deliberately_unknown_probe_option=true` and
`allow_view_with_volatile_column='not-bool'`. Both graphs execute, but their
exported DDL and graph metadata omit all graph-level options, including
`validate_element_key_uniqueness`. These
[negative controls](evidence/cross-engine-20261005/emulator-option-control.jsonl)
show permissive acceptance and discarded option metadata on this image; they
do not establish either option's intended enforcement.

## Queues

Managed and Omni completed the GoogleSQL lifecycle: CREATE, canonical DDL and
catalog inspection, enqueue rollback, immediate receive, ACK rollback/commit,
and future delivery. The future message stayed stored and was not received by
the bounded early call. Ordinary queue reads with a one-millisecond exact-staleness request
succeeded; RECEIVE with the same bound was rejected as requiring strong
concurrency. This does not establish historical snapshot semantics for queue
contents.

Two separate discrepancies remain against the [official usage example](https://docs.cloud.google.com/spanner/docs/queues/queues-using):

1. A receiver-issued token was first supplied as `lease_tokens=>[@token]`,
   then as `lease_tokens=>@tokens` with an array binding. The scalar-inside-array
   form returned `InvalidArgument`; the array binding renewed the lease and
   returned a new token. Both results persisted with `AUTO_PARAMETER='OFF'`.
   Token hashes link the receiver output to the successful renewal input.
   The service's "invalid lease token" error does not prove the receiver-issued
   token itself was invalid.
2. The separate malformed literal `['not-a-token']` returned `InvalidArgument`,
   while the documentation describes unparseable tokens returning no row.
   This case does not test expired, otherwise well-formed tokens.

Emulator rejects `CREATE QUEUE` with a syntax error at `QUEUE`. Receive, ACK,
lease renewal, and queue catalog behavior consequently remain untested there;
the dedicated lease-control run was skipped with that reason in its receipt.

## Retained evidence

| Target | Fresh records |
| --- | --- |
| Managed | [Catalog (2)](evidence/cross-engine-20261005/managed-catalog.jsonl), [read-only controls (80)](evidence/cross-engine-20261005/managed-readonly.jsonl), [GoogleSQL (53)](evidence/cross-engine-20261005/managed-google-features.jsonl), [PostgreSQL (40)](evidence/cross-engine-20261005/managed-pg-features.jsonl), [queues (32)](evidence/cross-engine-20261005/managed-queue.jsonl), [cleanup (4)](evidence/cross-engine-20261005/managed-cleanup.jsonl) |
| Omni r4 | [Features/catalogs (95)](evidence/cross-engine-20261005/omni-feature-probe.jsonl), [queues (32)](evidence/cross-engine-20261005/omni-queue-probe.jsonl), [lease controls (14)](evidence/cross-engine-20261005/omni-queue-control.jsonl), [volatile graph (13)](evidence/cross-engine-20261005/omni-volatile-probe.jsonl) |
| Emulator | [Features/catalogs (98)](evidence/cross-engine-20261005/emulator-feature-probe.jsonl), [queue DDL/cleanup (5)](evidence/cross-engine-20261005/emulator-queue-probe.jsonl), [volatile graph (17)](evidence/cross-engine-20261005/emulator-volatile-probe.jsonl), [permissive-option controls (18)](evidence/cross-engine-20261005/emulator-option-control.jsonl) |

Counts are raw records, including setup, negative cases, and cleanup, rather
than a number of passing tests. The
[derived summary](evidence/cross-engine-20261005/summary.json) preserves target
differences and checks receiver/renewal token-hash continuity.

## Validation and limits

Fresh probes exercise isolated fixtures and await DDL operation completion.
Expected negative results remain in the raw records; a process exit of zero
alone does not establish feature support. Connection coordinates and request
IDs are excluded from retained records, and lease tokens are hashed.

Managed fixture scans returned no remaining tables, graphs, or generated queue
routines; the owned temporary PostgreSQL database returned NotFound. Container
database/runtime cleanup records are OK, and no containers remain for either
tested exact image. This is not a claim about unrelated Docker resources.

The retained-record assertions, OKF `--gate all`, and Markdown language check
passed. [Validation receipts](evidence/cross-engine-20261005/documentation-checks.json)
record exact commands and exit codes; the artifact manifest covers the retained
results, probe sources, and derived summary.

The earlier [October 5 managed run](managed-revalidation-20261005.md) passed all
four managed integration checks. The [r4 investigation](spanner-omni-2026.r4-lts-verification.md)
records stale catalog/plan assertions that failed and a separately passing
query-generator suite. Those larger suites were not rerun for this documentation
comparison, and this note does not claim that the whole Omni suite passes.

TLS/password/mTLS results are the earlier r4 probe linked above; commercial
billion-vector ANN, Kubernetes Helm behavior, performance, fleet rollout,
PostgreSQL queues, and untested SQL forms are outside this comparison. Emulator
success does not establish a managed plan, IAM, TLS, or concurrency contract.

Dated investigations remain historical evidence. Current guides now qualify
old r2.1 catalog claims, and the runtime index separates this comparison from
the dated archive. Future reruns should update this entry point or add a new
comparison only when changed evidence needs interpretation.
