---
type: Research Note
title: "Spanner Omni 2026.r4-lts verification"
description: "Digest-pinned arm64 verification of r4 LTS, including startup compatibility, catalog changes, SQL and queue probes, and Developer edition TLS authentication."
tags: [spanner, spanner-omni, runtime, metadata, query-plan, postgresql, queues, authentication]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: release
    resource: https://docs.cloud.google.com/spanner-omni/release-notes
    title: "Official release notes retrieved on 2026-10-04"
  - id: download
    resource: https://docs.cloud.google.com/spanner-omni/download
    title: "Official image and binary downloads"
  - id: security
    resource: https://docs.cloud.google.com/spanner-omni/deploy-encryption-vms
    title: "Official TLS and authentication deployment instructions"
  - id: go
    resource: https://docs.cloud.google.com/spanner-omni/go
    title: "Go v1.94.0 client configuration"
  - id: upgrade
    resource: https://docs.cloud.google.com/spanner-omni/upgrade-deployment
    title: "Phased upgrade and rollback boundaries"
  - id: infoschema
    resource: ../../../survey/infoschem/evidence/omni/9f10586c5de366bec5c0ff6cf542a67c21c8363e8a12a3deb9c372a07c9aa8b6/linux-arm64-89b4ab9e231d.json
    title: "Exact-image INFORMATION_SCHEMA capture"
  - id: spannersys
    resource: ../../../survey/spannersys/evidence/omni/9f10586c5de366bec5c0ff6cf542a67c21c8363e8a12a3deb9c372a07c9aa8b6/linux-arm64-9bb70c640b69.json
    title: "Exact-image SPANNER_SYS capture"
  - id: receipt
    resource: evidence/omni-r4-20261004/receipt.json
    title: "Runtime, source, commands, and retained evidence provenance"
---

# Spanner Omni 2026.r4-lts verification

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Observed on October 4, 2026 UTC using Developer edition in native
`linux/arm64` containers under Colima. The execution image is
`us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r4-lts@sha256:9f10586c5de366bec5c0ff6cf542a67c21c8363e8a12a3deb9c372a07c9aa8b6`.
The tag index also advertises amd64 manifest
`sha256:91055849d7a1f863a90959716ed3d46eaf14c5fd58aa31d82c13a0b5f951ed05`;
amd64 execution was not tested. The inspected arm64 image creation timestamp
is `2026-09-29T09:33:27.016913122Z`.

The isolated worktree starts at
`c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c` plus the existing dirty snapshot.
The main checkout's runtime pins and generated catalogs were not migrated.
The [r3 investigation](spanner-omni-2026.r3-beta-verification.md) supplies
retained comparison evidence; r3 was not comprehensively rerun in this session.

## Official release scope

The September 30 announcement identifies r4 as GA and LTS, with security
fixes for up to one year. It explicitly includes managed Spanner changes
through September 18, including queues. Other announced changes are:

- Commercial edition ANN search with dedicated stateless workers, up to one
  billion vectors.
- Developer edition TLS and mTLS encryption and authentication.
- Separate root and non-root StatefulSets in the Helm chart.
- Username/password authentication over TLS in Java 6.119.0+, Go v1.94.0+,
  and Python 3.72.0+.

The previously unavailable upgrade document now describes phased upgrades
from r3. Schema preparation is forward-only but backward-compatible; binary
rollback is possible before finalization begins. Finalization prevents
rollback. This investigation did not upgrade an existing deployment or test
the upgrade/rollback procedure, Helm deployment, Commercial ANN scale, Java,
Python, or multi-node behavior. The documented coverage date does not prove
that every later feature is absent or every advertised feature was exercised.

## Startup compatibility

Unmodified spanemuboost v0.4.7 invokes `spanner start-single-server` without
an explicit listen address. All four initial capture/test commands failed
during bootstrap with `Unavailable: error reading server preface: EOF`.
These failures occurred before SQL evaluation.

The r4 CLI exposes `--listen-addresses` with default `localhost`; the retained
r3 CLI help has no such option. Adding `--listen-addresses=0.0.0.0` to the
isolated container command permits Docker port forwarding and makes the
captures and SQL probes connect successfully. The argument changes binding
inside the container; host-side port exposure must still be configured
appropriately. The separate TLS test published its port only on host loopback.

The workaround is a local copy of spanemuboost v0.4.7 selected through
worktree-only `go.work` and survey `go.mod` replacements. The exact one-line
change is retained in [harness.patch](evidence/omni-r4-20261004/harness.patch).
The module cache and main checkout's dependency declarations are unchanged.
The existing integration runner also adds JSON test output in the worktree.
Initial failure logs remain under `startup-original/`; later results use the
explicit listen address. This verifies the workaround without claiming an
upstream spanemuboost fix has shipped.

## Catalog changes

| Surface | Retained r3 | Fresh r4 | Observed difference |
| --- | --- | --- | --- |
| INFORMATION_SCHEMA | 310 columns | 310 columns | Identical surface hash |
| SPANNER_SYS | 50 tables / 587 columns | 51 tables / 598 columns | Added ACTIVE_BATCH_OPERATIONS |

No existing column was removed, retyped, or reordered. The new table has
eleven columns in this order:

| Column | Type |
| --- | --- |
| TEXT | STRING(MAX) |
| QUERY_ID | STRING(MAX) |
| NUM_PARTITIONS_TOTAL | INT64 |
| NUM_PARTITIONS_COMPLETE | INT64 |
| NUM_TRIVIAL_PARTITIONS_COMPLETE | INT64 |
| PROGRESS | STRING(MAX) |
| ROWS_PROCESSED | INT64 |
| STATUS | STRING(MAX) |
| RESULT | STRING(MAX) |
| START_TIMESTAMP | TIMESTAMP |
| LAST_UPDATE_TIMESTAMP | TIMESTAMP |

`SELECT * FROM SPANNER_SYS.ACTIVE_BATCH_OPERATIONS LIMIT 1` succeeds and
returns no rows in the empty probe deployment. This establishes queryability,
not the behavior of active batch operations. No matching official explanation
was located in the two bounded Developer Knowledge searches.

The drift test fails its old 50-table/539-column expectation and reports the
new table as unknown to the registry. Its diagnostics check 50 registered
tables, decode three rows, and find zero advertised-but-unqueryable columns;
eight known-absent columns remain. These diagnostics exclude the new table,
which was queried independently above. INFORMATION_SCHEMA still exposes
`INDEXES.SEARCH_UNNEST` and `INDEX_COLUMNS.EXPRESSION`.

## Existing integration expectations

All 22 top-level plan-shape tests ran: 20 passed and two failed. Nested
results are 200 passes and 18 failures, with no skips. The failures match
the retained r3 failures:

- `TestIntegrationAggregateFunctionAggTypesVersionMatrixOnOmni` expects
  `APPROX_COUNT_DISTINCT` rejection, but analysis succeeds for versions 1-9.
- `TestIntegrationConditionBoundariesVersionMatrixOnOmni` expects literal
  `'Alpha'` in range/seek expressions, but all nine versions use the automatic
  parameter `_auto_p0_string_`.

These are failed gates with identified stale expectations, not passing tests.
The query-generator suite was run separately because the shared runner stops
after the plan-suite failure: all three top-level tests and 26 nested tests
passed, with no skips.

## Bounded SQL and queue comparison

The feature probe produced 93 records: the same 92 observations as the
retained r3 probe plus the new active-batch-table query. Matching labels have
the same result/error codes. The following behavior remains observed:

- `APPROX_COUNT_DISTINCT` returns 2 for all optimizer versions 1-9. The
  default optimizer remains version 9.
- Automatic parameterization is present for DEFAULT/AUTO/ON and absent for
  OFF. An invalid hint value is rejected.
- Graph label/property descriptions and synonyms survive metadata and DDL
  export. A join-view graph fails its default/true uniqueness validation and
  succeeds with `validate_element_key_uniqueness=false`, returning two nodes.
- PostgreSQL `ILIKE`, `NOT ILIKE`, `~~*`, and `!~~*` work. Direct
  `pg.ilike`, `pg.not_ilike`, `spanner.ilike`, and `spanner.not_ilike` calls
  are rejected. BERNOULLI and SPANNER.RESERVOIR sampling work; SYSTEM remains
  unimplemented.
- The twelve bounded ANY/SOME/ALL and LIKE-quantifier forms remain rejected
  with `InvalidArgument`.

The result differences are a new graph metadata option and expected sampling
variation (sample choice/order); this comparison does not assert byte-identical
plans, performance, or exhaustive SQL parity. Both inspected graph metadata
objects now include `allow_view_with_volatile_column` as BOOL/FALSE, in
addition to `validate_element_key_uniqueness`.

A focused follow-up distinguishes metadata emission from DDL support.
A view containing `GENERATE_UUID()` can back a graph when the new option is
omitted and uniqueness validation is disabled; MATCH returns its generated
string property. Explicit `allow_view_with_volatile_column=false` and `true`
both fail with `Unimplemented: The allow_view_with_volatile_column option is
not supported in graph statement.` Therefore the emitted metadata does not
establish a usable public DDL option or its intended semantics. Earlier probe
attempts failed on an unqualified view column and then on unsupported `RAND()`;
both attempts and the corrected final probe are retained separately.

The queue lifecycle probe produced 32 records. It confirms queue DDL/catalog
exposure, enqueue and ACK transactions with rollback controls, immediate
delivery, exclusion of future messages, lease renewal using an array
parameter, the generated TVF plan, duplicate-ACK rejection, and cleanup.
These are runtime observations beyond the official coverage statement; queues
also worked in retained r3 and r2.1 probes and are not newly discovered runtime
support in r4.

The separate 14-record lease control probe matches r3: an ordinary stale read
works, but stale RECEIVE is rejected because strong concurrency is required.
`lease_tokens=>[@token]` rejects the scalar parameter as an invalid lease
token, while `lease_tokens=>@tokens` with an array parameter succeeds.
Explicit `AUTO_PARAMETER='OFF'` does not change either outcome. Tokens are
retained only as SHA-256 hashes, and all queue/database/runtime cleanup
records report success.

## Developer edition TLS authentication

A fresh unlicensed Developer container uses an ephemeral CA, separate server
and API certificates, an `admin` client certificate, and the documented
`password,client-certificate` authentication methods with the OPAQUE protocol.
The standalone probe module uses Go SDK v1.94.0 without upgrading repository
dependencies. It does not disable TLS certificate verification.

| Probe | Result |
| --- | --- |
| Create a database using mTLS | OK; operation completion awaited |
| SELECT 1 using mTLS | OK, returns 1 |
| SELECT 1 using TLS and the correct admin password | OK, returns 1 |
| SELECT 1 using TLS and a wrong password | Unauthenticated, auth tag mismatch |
| SELECT 1 using TLS without authentication credentials | Unauthenticated |
| Drop the database using mTLS | OK |

The container and its anonymous volume were removed, and host-side temporary
password and client-key files were deleted. Only probe source, dependency
versions, non-secret startup logs, outcomes, and cleanup receipts are retained.
This is a bounded connection/authentication check, not certificate rotation,
authorization-role coverage, or a security audit.

## Evidence and completion

The [receipt](evidence/omni-r4-20261004/receipt.json) identifies the worktree,
dirty source snapshot, binaries, document hashes, and durable task directory.
Exact commands, raw observations, and initial failed attempts are retained
under [the evidence directory](evidence/omni-r4-20261004/). The offline
[verifier](evidence/omni-r4-20261004/verify.py) checks catalog differences,
feature/queue outcomes, authentication controls, test counts, cleanup, and the
artifact hash manifest. It passes. The repository OKF `all` gate and Markdown
language check both pass; their receipts are retained.

No containers remain for the exact r4 image ancestor or this task's explicit
container label. Unrelated stopped containers were preserved. Per-probe
database and runtime cleanup also completed. The investigation worktree and
durable source/evidence checkpoint remain available until explicitly cleaned
up. No source commit, main runtime migration, external publication, or existing
deployment upgrade was performed.
