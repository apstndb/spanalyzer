---
type: Research Note
title: "Managed Spanner revalidation, 2026-10-05"
description: "Fresh managed catalog, integration, SQL and queue observations, with direct checks of the differences found in Omni r4 LTS."
tags: [spanner, managed, runtime, metadata, queues, postgresql, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: receipt
    resource: evidence/managed-recheck-20261005/receipt.json
    title: "Execution, dependency, source, and document provenance"
  - id: catalog
    resource: evidence/managed-recheck-20261005/catalog-comparison.json
    title: "Current catalog comparison against September 29"
  - id: release
    resource: https://docs.cloud.google.com/spanner/docs/release-notes
    title: "Full official Markdown and direct HTML retrieved October 5 JST"
  - id: queues
    resource: https://docs.cloud.google.com/spanner/docs/queues/queues-using
    title: "Fresh official queue usage and lease parameter examples"
  - id: pg-functions
    resource: https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions
    title: "Fresh PostgreSQL function reference"
  - id: previous
    resource: managed-revalidation-20260929.md
    title: "Previous managed verification"
  - id: omni
    resource: spanner-omni-2026.r4-lts-verification.md
    title: "Digest-pinned Omni r4 LTS verification"
---

# Managed Spanner revalidation, October 5

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Observed on October 5, 2026 Asia/Tokyo (October 4 UTC) against the configured
managed GoogleSQL verification database and a temporary PostgreSQL database.
This is a sample of that target, not a globally identified managed service
build or evidence of rollout completion across all regions. Target coordinates
and credentials are omitted from retained evidence.

The existing source snapshot remains at
`c2d7792dd7f83d5dd086e2d32bc22e5d253cf06c` plus pre-existing dirty work. The
survey producer source hashes match September 29. The independent survey
module uses Go 1.26.8, Spanner v1.88.0, and memefish v0.8.1; feature probes use
the existing workspace with Spanner v1.91.0. No dependency or canonical
projection migration is included.

## Current catalogs and official documentation

Both fresh official Markdown and direct HTML still show September 18 as the
latest managed release entry. The fetched documents differ in rendering from
the earlier retained Markdown, so byte changes are not described as new
product releases. The PostgreSQL function reference still lists `pg.ilike`
and `pg.not_ilike`; the queue usage page still shows a scalar parameter inside
an array for lease renewal.

| Surface | Fresh managed observation | September 29 managed | Omni r4 |
| --- | --- | --- | --- |
| INFORMATION_SCHEMA | 310 columns; three agreeing observations | Same surface hash | Same surface hash |
| SPANNER_SYS | 587 columns | Same surface hash | 598 columns |
| ACTIVE_BATCH_OPERATIONS | Not advertised; SELECT returns InvalidArgument / Table not found | Not advertised | Advertised; SELECT succeeds |
| Optimizer | Versions 1-9; default and PROFILE version 9 | Same | Same |

Both catalog commands report `producer_changed_only` against their selected
September 16 baseline. Against September 29, their producer and surface hashes
both match. This is not a service-side catalog change.

All 48 qualified distribution-column queries succeeded. Sixteen returned a
row and 32 were empty, versus 32 populated and 16 empty on September 29.
Population depends on workload and aggregation windows; successful empty
queries establish queryability but do not exercise populated-value decoding.

`COLUMN_OPTIONS.IS_INHERITED` and `LOCALITY_GROUP_OPTIONS.IS_INHERITED` resolve;
`TABLE_OPTIONS.IS_INHERITED` remains unrecognized. The twelve bounded
ANY/SOME/ALL and LIKE-quantifier forms remain rejected by both managed Spanner
and memefish. The approximate distinct-count control over `[1,1,2,NULL]`
returns 2. These checks do not establish quantified NULL semantics or
large-cardinality accuracy.

## Existing managed integration suite

All four tests passed with `-count=1`, no skips, and 321.97 seconds including
invocation: canonical DDL, catalog drift, expression indexes, and UUID.
Canonical DDL diagnostics again report 330 service statements and 330
reconstructed statements, zero unexpected parse errors, and one allowlisted
statement. Equal totals are not exact DDL equivalence: an `AlterTable` is
folded into metadata, while reconstruction emits an extra `AlterStatistics`.

The drift audit reports 50 advertised tables, 587 columns, eight known-absent
registry columns, and no drift. Its diagnostics checked 50 tables, decoded
19 rows, and found zero advertised-but-unqueryable columns. Expression-index
and UUID fixtures passed their metadata, reconstruction, and cleanup checks.

## GoogleSQL and the Omni r4 graph difference

The GoogleSQL feature probe produced 53 records. The existing automatic
parameterization and graph semantic/uniqueness controls match September 29:
DEFAULT/AUTO/ON generate `_auto_p0_int64_`, OFF does not, and invalid mode
values are rejected. Graph descriptions and synonyms survive metadata and
DDL export. The join-view graph fails default/true uniqueness validation and
succeeds with false, returning two nodes.

Nine new approximate-count queries, one per optimizer version 1-9, all return
2 for the two distinct fixture names. No precision or performance claim is
inferred from this small input.

The new graph comparison uses a view selecting `GENERATE_UUID()` as a property
and disables key uniqueness validation, matching the final Omni probe:

| Observation | Fresh managed | Omni r4 |
| --- | --- | --- |
| Existing graph metadata options | Only validate_element_key_uniqueness | Also emits allow_view_with_volatile_column=FALSE |
| New option omitted | Graph creation and MATCH succeed | Graph creation and MATCH succeed |
| Explicit false or true | InvalidArgument: option is unknown for graph statements | Unimplemented: option is not supported in graph statement |

Thus the runtime error and metadata surface differ even though neither target
accepts explicit use of that option. Omitting it permits the tested generated
string property; this does not establish what the option's eventual semantics
will be. Both managed test graphs inspected for metadata omit the new option.
All created graphs, views, and the feature table were dropped successfully.

## PostgreSQL

The 40-record PostgreSQL probe repeats the September 29 parameterization,
matching, and sampling checks in a newly created temporary database.
`ILIKE`, `NOT ILIKE`, `~~*`, and `!~~*` succeed. The four direct `pg.*` and
speculative `spanner.*` matching function calls still return `InvalidArgument`
because the functions do not exist. The fresh official function reference
continues to list the two `pg.*` functions, so that documentation/runtime
discrepancy remains on this target.

BERNOULLI 0/100 returns 0/2 rows, SPANNER.RESERVOIR 0/1/100 returns 0/1/2,
and SYSTEM sampling returns `Unimplemented`. Sample membership and ordering
are not stability assertions. The temporary PostgreSQL database drop succeeded.

## Queues and final cleanup

The 32-record queue probe completed DDL/catalog checks, enqueue/receive,
transaction rollback, ACK, delayed delivery, lease renewal, and cleanup.
Ordinary stale reads succeed, while stale RECEIVE returns `InvalidArgument`
because strong concurrency is required. Enqueue rollback leaves no row; ACK
rollback preserves the message; committed ACK removes it; duplicate asserted
ACK returns `OutOfRange`. A future message remains stored and is not received
in the bounded observation window.

Both default and `AUTO_PARAMETER='OFF'` scalar-inside-array renewal forms
(`lease_tokens=>[@token]`) return `InvalidArgument` for an invalid lease token.
The whole-array form (`lease_tokens=>@tokens`, bound as `[]string`) succeeds
under both modes. The deliberately malformed literal token also returns
`InvalidArgument`. This repeats the September 29 managed and Omni r4 results.
The fresh official Go example and the statement that unparseable tokens return
no rows still differ from these bounded runtime observations. An empty rows
array accompanied by an error is not a successful empty query result.

Final read-only checks find no date-prefixed tables/views, property graphs, or
generated queue routines. GetDatabase for the temporary PostgreSQL database
returns `NotFound`. Lease tokens are retained only as hashes; target coordinates
and request identifiers are redacted.

## Evidence and limits

The [receipt](evidence/managed-recheck-20261005/receipt.json) identifies sources,
dependencies, documents, commands, and the durable task directory. Retained
records comprise 80 read-only observations, 53 GoogleSQL feature observations,
40 PostgreSQL observations, 32 queue observations, and four final cleanup
observations. Deliberate negative controls are included in these counts.
All 37 previously comparable GoogleSQL records, 40 PostgreSQL records, and
32 queue records have the same outcome codes as September 29; new graph and
optimizer checks are separately identified.

The existing integration suite finished before feature DDL began, preventing
temporary graph/queue objects from affecting canonical-DDL reconstruction.
Temporary fixtures were cleaned up; pre-existing database objects were not
changed. No commit, push, publication, dependency upgrade, or canonical
projection migration was performed.

This does not rerun vector-index rebuild, ordinary-column semantic options,
full-text index competition, the full optimizer PLAN matrix, PostgreSQL queues,
lease-expiry redelivery, distributed durability, or performance measurements.
Neither unchanged sampled catalogs nor these bounded SQL results establish
complete managed/Omni parity or a global rollout boundary.

Run `python3 knowledge/research/runtime-verification/evidence/managed-recheck-20261005/verify.py`
to check the retained observations, cleanup, integration results, and hashes
offline. Repository OKF and Markdown-language check receipts are retained
alongside the evidence.
