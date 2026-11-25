---
type: Research Note
title: GoogleSQL catalogs on managed Spanner, Omni and Emulator
description: A November 25 JST comparison of catalog columns and the optimizer-version table on four GoogleSQL targets, with a repeated managed query suite and histogram controls.
tags: [managed-spanner, spanner-omni, emulator, runtime-verification, information-schema, spanner-sys]
status: draft
sources:
  - id: measurements
    resource: evidence/managed-recheck-20261125/receipt.json
    title: Dated measurements, column differences, case counts and runtime identities
  - id: baseline
    resource: managed-placements-and-omni-privileges-20261124.md
    title: Previous managed replay and histogram controls
  - id: registry
    resource: https://us-docker.pkg.dev/v2/spanner-omni/images/spanner-omni/tags/list
    title: Published Omni tags and image index identities
  - id: emulator
    resource: https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/releases/tag/v1.5.59
    title: Emulator 1.5.59 release
---

# GoogleSQL catalogs on managed Spanner, Omni and Emulator

On November 25, 2026 JST (November 24, 17:16–17:24 UTC), managed Spanner and both current Omni images exposed the same 310 `INFORMATION_SCHEMA` columns. Managed exposed 32 additional `SPANNER_SYS` columns. On Emulator, the optimizer-version table could be queried even though its columns did not appear in the catalog. [Measured rows, image identities and comparison](evidence/managed-recheck-20261125/receipt.json)

| GoogleSQL target | `INFORMATION_SCHEMA` tables / columns | Advertised `SPANNER_SYS` tables / columns | Versions returned by `SUPPORTED_OPTIMIZER_VERSIONS` | Reported default |
| --- | --- | --- | --- | --- |
| Managed, one regional Enterprise trial database | 48 / 310 | 52 / 630 | 1 through 9 | 9 |
| Omni `2026.r4-lts.3`, Linux arm64 | 48 / 310 | 51 / 598 | 1 through 9 | 9 |
| Omni `2026.r5.2`, Linux arm64 | 48 / 310 | 51 / 598 | 1 through 9 | 9 |
| Emulator `1.5.59`, Linux arm64 | 30 / 204 | 0 / 0 | 42 | 42 |

Both Omni images reported the same names, types and ordinal positions for all 598 system columns. The 32 columns visible only on managed comprised 11 columns in `BATCH_OPERATIONS`, 12 lock-wait distribution columns in `LOCK_STATS_*`, six lock-latency distribution columns in `TXN_STATS_TOP_*`, and three latency-impact correlation scores in `SPLIT_STATS_TOP_*`. All shared types and ordinals matched. The repository's managed baseline still contains 624 columns. Its drift test flagged exactly the six `TXN_STATS_TOP_*` columns first observed on November 20. [November 20 evidence](evidence/managed-recheck-20261120/receipt.json)

## Managed replay

The first pass repeated all 272 routine GoogleSQL inputs on the November 24 database. Every gRPC status and result field matched the previous first pass. Twenty row multisets changed in clock or live statistics reads. The six placement checks and the administrative partition listing again found only the default placement and partition and no explicit routing option. The administrative DDL and its reconstruction both contained zero definitions. [November 24 baseline](managed-placements-and-omni-privileges-20261124.md)

Five histogram controls ran with the placement checks. The second pass used the exact executable from the November 24 baseline. On the first pass, the rolling query-latency percentile returned `InvalidArgument` and its companion counted zero input histograms. Two minutes later, the identical SQL succeeded and the companion counted 299 non-null histograms. Both passes returned SQL `NULL` for the guarded empty input and `1018.88` for the fixed sample. This recheck used read-only GoogleSQL. [Replay inputs and compact results](evidence/managed-recheck-20261125/receipt.json)

## Publication check

At 02:31 JST the mirror remained at `5f1eee7`. By the evening recheck it had advanced one commit to `53b1cf9`. The only changed file was `metadata.yaml`. The managed Atom feed and BigQuery both led with the November 18 QueryData preview. The managed HTML and Developer Knowledge pages led with November 10, and the Omni release notes led with October 29. The three image tags in the table and their registry digests matched November 24. [Mirror comparison](https://github.com/apstndb/spanner-docs-mirror/compare/5f1eee77499cc8cffa5759c87ce72d575b3de319...53b1cf924b4243d286afaeb9ae6d0a76b3d12fd2) · [Managed feed](https://docs.cloud.google.com/feeds/spanner-release-notes.xml) · [Omni release notes](https://docs.cloud.google.com/spanner-omni/release-notes)

## Reproduction

Use Python 3 to run `replay_inputs.py primary` and `replay_inputs.py controls`, then save each emitted JSON array to a file. The receipt identifies the two measured binaries by SHA-256. To build an independent replay executable, compile the linked Go source as a `.go` file from the `survey` module with `GOWORK=off GOFLAGS=-mod=readonly`, set `TEST_REAL_SPANNER_DATABASE` to the GoogleSQL database resource, and pass one input file per invocation. The receipt records both file hashes, the four common catalog queries, and the image digests for repeating the local measurements. [Input generator](evidence/managed-recheck-20261125/replay_inputs.py) · [Go producer](evidence/managed-recheck-20261112/producer/query.go.txt) · [Measured results](evidence/managed-recheck-20261125/receipt.json)
