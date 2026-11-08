---
type: Research Note
title: "Spanner Omni 2026.r3-beta verification"
description: "Digest-pinned arm64 verification of Omni r3 catalogs, existing integration expectations, and bounded GoogleSQL and PostgreSQL feature comparisons."
tags: [spanner, spanner-omni, runtime, metadata, query-plan, postgresql, queues]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: release
    resource: https://docs.cloud.google.com/spanner-omni/release-notes
    title: "Official Omni release notes, direct HTML retrieved 2026-09-26"
  - id: managed-release
    resource: https://docs.cloud.google.com/spanner/docs/release-notes
    title: "Managed release notes, full Markdown and direct HTML retrieved 2026-09-26 UTC"
  - id: queues
    resource: https://docs.cloud.google.com/spanner/docs/queues/queues-using
    title: "Official queue lifecycle and lease examples"
  - id: download
    resource: https://docs.cloud.google.com/spanner-omni/download
    title: "Official r3 download page"
  - id: mirror
    resource: https://github.com/apstndb/spanner-docs-mirror/commit/e033dcb6d7414e921cc57b8156d6e6411a51679d
    title: "User-provided documentation mirror commit"
  - id: infoschema
    resource: ../../../survey/infoschem/evidence/omni/8243ce3c748b10371a16f05bc7553465fe978d7349127ad331d02c1d3a9fcc99/linux-arm64-89b4ab9e231d.json
    title: "Exact-image INFORMATION_SCHEMA capture"
  - id: spannersys
    resource: ../../../survey/spannersys/evidence/omni/8243ce3c748b10371a16f05bc7553465fe978d7349127ad331d02c1d3a9fcc99/linux-arm64-58205fd318c3.json
    title: "Exact-image SPANNER_SYS capture"
  - id: receipt
    resource: evidence/omni-r3-20260926/receipt.json
    title: "Source, image, command, and document provenance"
---

# Spanner Omni 2026.r3-beta verification

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Observed on September 26, 2026 on native `linux/arm64` containers under Colima.
The exact image is
`us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r3-beta@sha256:8243ce3c748b10371a16f05bc7553465fe978d7349127ad331d02c1d3a9fcc99`.
The public index also advertises amd64 manifest
`sha256:0452b944e1ecd111d1e7c1d70dbd2371b302c6383b5c610222a382e8c894c5b3`;
amd64 execution was not tested.

The investigation worktree starts at
`c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c` and includes the existing dirty
working snapshot. Only that worktree switches runtime pins to r3. Its shared
integration runner adds JSON test output. The main checkout's runtime pins and
catalog projections remain unchanged. Source snapshots and exact commands are
identified by the receipt.

## Documentation

The direct official HTML announces r3 on September 17 and says it includes
Spanner changes through the September 3 release. This describes the documented coverage date;
it does not establish that later features are absent. It also says r3 is required
before upgrading to future releases. The linked `/spanner-omni/upgrade-deployment`
page returned HTTP 404 during this investigation; no deployment upgrade was
attempted.

The same-day `dkcli` download page includes r3, but its release-notes content
still begins with r2.1. Direct full HTML supplies the r3 announcement. The
user-provided mirror commit updates download/quickstart image tags and Helm
chart references but does not add that release-note entry. Full source documents
are retained in the durable directory identified by the receipt.

## Catalog changes

| Surface | Retained r2.1 | Fresh r3 | Delta |
| --- | --- | --- | --- |
| INFORMATION_SCHEMA | 308 columns | 310 columns | `COLUMN_OPTIONS.IS_INHERITED` and `LOCALITY_GROUP_OPTIONS.IS_INHERITED`, both BOOL |
| SPANNER_SYS | 50 tables / 539 columns | 50 tables / 587 columns | 48 distribution columns on twelve QUERY_STATS/TXN_STATS interval tables |

No retained column was removed, retyped, or reordered. Both r3 surface hashes
match the corresponding retained managed captures and the
[September 25 managed revalidation](managed-revalidation-20260925.md).
This is equality of advertised metadata tuples, not full service parity.

The INFORMATION_SCHEMA capture also confirms `INDEXES.SEARCH_UNNEST` and
`INDEX_COLUMNS.EXPRESSION` are queryable. The SPANNER_SYS drift audit reports
eight known-absent registry columns, no unknown/type/order drift, 50 checked
tables, three decoded rows, and zero advertised-but-unqueryable columns.

The existing Omni drift test fails its r2.1-specific 539-column count assertion.
It continues through the UUID, expression-index, reconstruction, and cleanup
checks without additional diagnostics. This is a failed test with an identified
outdated expectation, not a passing gate.

## Existing plan-shape integration suite

All 22 selected top-level tests ran: 20 passed and two failed. Nested results
are 200 passes and 18 failures; these are not additional independent top-level
tests. There were no skips.

- `TestIntegrationAggregateFunctionAggTypesVersionMatrixOnOmni`: all nine
  optimizer-version subtests expected `APPROX_COUNT_DISTINCT` to be rejected,
  but PLAN analysis succeeds.
- `TestIntegrationConditionBoundariesVersionMatrixOnOmni`: all nine
  optimizer-version subtests expected the literal `'Alpha'` in Split Range
  and Seek Condition expressions. r3 instead contains `_auto_p0_string_`.

Other inspected assertions in these two tests did not report failures. The
remaining 20 top-level suites passed. These results validate their existing
case sets, not an exhaustive inventory of SQL or GQL support. The runner stops
after the plan-shape package fails, so query-generator validation is executed
separately and recorded below.

The separate query-generator run passed all three top-level tests and 26
subtests, with no skips (52.881 seconds of package execution). Generated-query
execution, operator-family coverage, and metadata contracts passed against r3.
The overall original suite remains failed because the drift and plan-shape
expectations described above were not rewritten for this investigation.

## Paired feature probes

The same probe binary ran against both exact arm64 images in fresh isolated
GoogleSQL and PostgreSQL databases. The r2.1 image was pinned to
`sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741`.
Both runs retain 92 observation records and report successful cleanup of both
databases and the runtime. Sources, JSONL, raw PROFILE plans, and build hashes
are retained in [the evidence directory](evidence/omni-r3-20260926/receipt.json).

| Probe | Fresh r2.1 result | Fresh r3 result |
| --- | --- | --- |
| `APPROX_COUNT_DISTINCT(Name)` over two distinct names, optimizer v1 through v9 | `Unimplemented` in all nine executions | Returns 2 in all nine executions; PROFILE succeeds |
| Approximate distinct count over `[1,1,2,NULL]` | `Unimplemented` | Returns 2 |
| GoogleSQL primary-key lookup, default/AUTO/ON | No generated `auto_parameters` observed | `_auto_p0_int64_` reflects each supplied literal |
| Same lookup, OFF | No generated `auto_parameters` | No generated `auto_parameters` |
| Invalid AUTO_PARAMETER hint value | Accepted in these probes | `InvalidArgument` |
| Graph uniqueness validation, default/true/false over a join view | Reject / reject / accept | Same |
| Graph label/property description and synonyms | Accepted and retained in metadata/export | Same |
| PostgreSQL `ILIKE`, `NOT ILIKE`, `~~*`, `!~~*` | Expected true/false results | Same |
| PostgreSQL `pg.ilike(text,text)` and `pg.not_ilike(text,text)` | Function does not exist | Same |
| PostgreSQL BERNOULLI(0)/(100), two input rows | 0/2 rows | Same |
| PostgreSQL SPANNER.RESERVOIR(0)/(1)/(100), two input rows | 0/1/2 rows | Same |
| PostgreSQL SYSTEM sampling | `Unimplemented` | Same |
| Twelve quantified comparison and LIKE cases | Explicit unsupported-feature errors | Same |

The approximate-count plans contain `APPROX_COUNT_DISTINCT_START` and
`APPROX_COUNT_DISTINCT_FINAL`. These tiny samples establish execution and basic
duplicate/NULL behavior, not large-cardinality accuracy or a performance claim.
The parameterization comparison establishes this bounded primary-key SELECT
behavior; it does not benchmark plan caching or latency. Both images report
optimizer 9 for ordinary PROFILE requests, and r3 lists versions 1 through 9
with 9 as the default.

The graph tests and PostgreSQL matching/sampling already work in the freshly
tested r2.1 image. Their later managed release announcements therefore do not
establish that r3 introduced those features to Omni. The `pg.*` function failures
match the [September 22 managed observation](managed-release-notes-20260922.md),
which was not freshly rerun here. Speculative `spanner.*` function aliases also
fail on both Omni versions.

## Managed announcements after September 3

A follow-up retrieved the complete managed release notes through both `dkcli`
and direct HTML. Both begin with September 18; the entries after September 3
are September 11 (two Graph enhancements), September 17 (queues GA), and
September 18 (automatic parameterization). Every item has a bounded runtime
probe here. Managed announcement dates are not Omni introduction dates, and
managed GA status does not change Omni's beta status.

| Managed announcement | Bounded r3 observation | Fresh r2.1 comparison |
| --- | --- | --- |
| September 11: label/property semantic options | DDL, metadata, and exported description/synonyms agree | Same |
| September 11: element-key uniqueness option | Default/true reject the join-view fixture; false accepts and MATCH returns 2 | Same |
| September 17: queues | Create, send, receive, renew, acknowledgment, and rollback controls execute | Same |
| September 18: automatic parameterization | Default/AUTO/ON generate parameters for primary-key SELECT; OFF does not | No generated parameters observed |

The Graph and automatic-parameterization results above already disprove a
September 3 runtime ceiling. The additional GoogleSQL queue probe on r3 also
completed `CREATE QUEUE` and `DROP QUEUE`, transactional send and receive, and
acknowledgment using `DELETE ... ASSERT_ROWS_MODIFIED 1`. The same binary also
completed this lifecycle against r2.1: queues were already executable there.
Both lifecycle runs contain 32 records and report successful queue, database,
and container cleanup.

Queue controls established the following:

- Rolling back an enqueue left no stored message; committed immediate and
  one-hour-future messages were stored, and only the immediate message was
  received in the bounded receive window.
- Rolling back an acknowledgment kept both messages. Committing it removed the
  immediate message. Repeating it returned `OutOfRange` because zero rows
  matched the expected one.
- Omitting `DeliverTime` assigned a timestamp. Catalog metadata identified
  `TABLE_TYPE=QUEUE`, the implicit non-null timestamp column, and the generated
  `RECEIVE_` and `RENEWLEASE_` table functions. No queue-specific catalog names
  were returned by the bounded name search; this is not a full catalog recapture.
- PLAN analysis of the receive query succeeded with Serialize Result and
  TVF nodes. The returned lease token was
  renewed successfully with the whole-array binding `lease_tokens=>@tokens`.
  Actual tokens remained in process memory; retained rows contain SHA-256 hashes.

The initial exact-staleness control used a timestamp one minute before the
query and returned a missing-TVF error. Because the queue was newly created,
that result alone does not establish the queue stale-read restriction.

A separate fresh-database control used exact staleness of one millisecond.
An ordinary query successfully read the just-enqueued row, establishing that
the schema and data existed at the selected timestamp; `RECEIVE` then failed
with `InvalidArgument: Queue receive queries must use strong concurrency.`
This result was reproduced on both r3 and r2.1.

Both images also reproduced a lease-argument discrepancy while the receive
stream remained open: `lease_tokens=>[@token]` returned `InvalidArgument`
(invalid lease token), while binding `[]string{token}` to
`lease_tokens=>@tokens` returned one row with a new token and expiration.
Disabling automatic parameterization did not change either result. Each
control completed in far less than the ten-second lease period, so these
observations are not explained by waiting for the lease to expire. The cause
was not traced into the server. This agrees with the retained
[managed September 18 result](managed-spanner-queues-20260918.md), whose controlled
raw capture also reports `InvalidArgument` for the scalar-array form; managed
was not rerun for this follow-up. Correction, September 29: the earlier version
of this paragraph mistakenly treated an empty rows array accompanying an error
as successful zero-row behavior. The raw evidence is unchanged.
The official Go example uses the scalar-array form, so its behavior should not
be assumed from syntax acceptance alone.

Each additional control run contains 14 records and successful cleanup.
[Coverage and documentation hashes](evidence/omni-r3-20260926/post-cutoff-coverage.json)
and the [follow-up receipt](evidence/omni-r3-20260926/queue-receipt.json)
identify the exact paired sources, binaries, and results. The follow-up ran
September 26 UTC, crossing into September 27 in Japan.

This bounded queue verification uses GoogleSQL and BYTES payloads. PostgreSQL
queues, other payload types, mutation Send/Ack, FGAC, options, TTL, interleaving,
redelivery after expiry, distributed durability, throughput, and full managed
feature parity were not tested. Tiny lifecycle fixtures do not establish those
properties or a service support guarantee.

## Adoption boundary

This investigation records behavior; it does not migrate the repository's
authoritative runtime targets or projections. A subsequent migration needs to
update the Omni drift expectation, intentionally select the new producer-owned
captures, regenerate dependent catalog contracts, and revise the two stale
plan expectations with appropriate r2/r3 coverage. In particular, the previous
Omni absence of the 48 distribution columns is no longer a reason to omit them
from a newly selected r3 projection.

No existing deployment data, in-place upgrade path, Kubernetes/Helm deployment,
commercial license behavior, amd64 execution, performance benchmark, or broad
managed/Omni equivalence was tested.

Run `python3 knowledge/research/runtime-verification/evidence/omni-r3-20260926/verify.py`
to check the retained evidence offline. This does not make new service requests.
