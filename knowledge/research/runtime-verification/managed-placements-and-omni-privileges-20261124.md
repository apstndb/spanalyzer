---
type: Research Note
title: Managed placement metadata and function privileges on Omni r5.2 and Emulator
description: A November 24 managed GoogleSQL replay, a controlled explanation of its one changed response, and a role-based privilege comparison on two pinned local engines.
tags: [managed-spanner, spanner-omni, emulator, placements, runtime-verification, fine-grained-access-control]
status: draft
sources:
  - id: measurements
    resource: evidence/managed-recheck-20261124/receipt.json
    title: Read-only SQL, returned values, role controls and registry digests
  - id: previous
    resource: managed-queue-search-upsert-20261121.md
    title: November 21 managed baseline and database scope
  - id: placements
    resource: https://docs.cloud.google.com/spanner/docs/create-manage-data-placements
    title: Placement routing metadata and per-dialect DDL
  - id: replication
    resource: https://docs.cloud.google.com/spanner/docs/replication
    title: Replica selection for read-only requests
  - id: omni-image
    resource: https://us-docker.pkg.dev/v2/spanner-omni/images/spanner-omni/manifests/2026.r5.2
    title: Published Omni r5.2 image index
---

# Managed placement metadata and function privileges on Omni r5.2 and Emulator

On November 24, 2026, one managed GoogleSQL query changed gRPC status from the November 21 replay, depending on the contents of a statistics window. The newly published Omni `2026.r5.2` matched `2026.r5.1` on all 49 privilege fixture steps. Emulator `1.5.59` behaved differently on both bulk privilege syntax and scalar-function execution under a database role. [SQL, results and image identities](evidence/managed-recheck-20261124/receipt.json)

## Managed GoogleSQL

The replay used one existing GoogleSQL database in a regional Enterprise trial instance. It repeated 236 inputs on the same database and 36 statistics inputs previously run on a disposable GoogleSQL database. On the first pass, 271 of 272 had the same gRPC status and result fields as on November 21. Across the 272 queries, 35 returned different row values or counts, including live statistics, server time and the changed-status case. Eleven further queries covered placement metadata and histogram controls. PostgreSQL and write fixtures were outside this run. [November 21 baseline](managed-queue-search-upsert-20261121.md)

The exception computed the 99th percentile of query latency over the last hour. The service reported a null or empty histogram on the first pass. The same SQL later succeeded when a companion count found 294 non-null histograms. Forcing an empty window reproduced `InvalidArgument`. Guarding the aggregate returned SQL `NULL`, and a fixed nonempty histogram returned `1018.88`. The following query reproduces the empty-input error even on a busy database.

```sql
SELECT SPANNER_SYS.DISTRIBUTION_PERCENTILE(
  SPANNER_SYS.DISTRIBUTION_AGG(LATENCY_DISTRIBUTION_JSON_STRING), 99.0
) AS p99
FROM SPANNER_SYS.QUERY_STATS_TOP_MINUTE
WHERE FALSE
```

`INFORMATION_SCHEMA` exposed 310 columns. The empty-schema canonical DDL check passed with zero definitions in both the administrative DDL and the reconstruction. `SPANNER_SYS` exposed 630 columns. Both catalog hashes matched November 21. The repository drift check still flags the six columns first observed on November 20, the lock latency distribution and its JSON form on the minute, ten-minute and hour `TXN_STATS_TOP` tables.

## What the documentation change covers

Three mirror commits followed the November 21 baseline. The first expanded the GoogleSQL and PostgreSQL instructions for setting `per_placement_routing_metadata`, clarified how read-only requests are routed across placements, and restored the gcloud backup-delete reference removed in the preceding snapshot. The following two changed only retrieval metadata. [Commit comparison](https://github.com/apstndb/spanner-docs-mirror/compare/8ff17fc9c5e93332c20a5ce54f15cf7c8cb3ebaf...5f1eee77499cc8cffa5759c87ce72d575b3de319)

The database option controls where routing metadata resides. With multiple data placements, Spanner considers replicas across all placement configurations and selects the closest available serving replica to the client for a read-only request, independently of the data queried. Our database reported only the default placement on the default instance partition. Its database options included the dialect and had no row for the routing option. This deployment could not exercise cross-placement replica selection. Both documented Boolean settings round-tripped in the memefish v0.9.0 GoogleSQL parser. [Placement guide](https://docs.cloud.google.com/spanner/docs/create-manage-data-placements) · [Replication guide](https://docs.cloud.google.com/spanner/docs/replication)

## Function privileges on local engines

Omni's registry recorded the `2026.r5.2` image index upload on November 22 UTC. All 49 steps on its Linux arm64 image returned the same statuses and query rows as on `2026.r5.1`. The 24 bulk statements covered grants and revocations on tables, views, change streams, sequences, scalar functions and table functions, in both the default and a named schema. Omni accepted all 24. Emulator rejected all 24 during DDL parsing. [Published r5.2 image](https://us-docker.pkg.dev/v2/spanner-omni/images/spanner-omni/manifests/2026.r5.2)

For the individual-function check, an administrator created `udfscope.f(x)`, which returns `x + 1`, and executed `SELECT udfscope.f(41)` as a positive control. The same SQL was then issued through a client configured with the database role `udf_reader`.

| Observation | Omni 2026.r5.2 | Emulator 1.5.59 |
| --- | --- | --- |
| Administrator control | 42 | 42 |
| Role before individual grant | `PermissionDenied` | 42 |
| Role after grant | 42 | 42 |
| Role after revoke | `PermissionDenied` | 42 |
| Grant on a missing function | `NotFound` | `OK` |

Emulator rejected `GRANT USAGE ON SCHEMA` in the original setup batch. Issuing each setup statement separately confirmed the schema, role and function existed before repeating the role checks. The administrator and role both read an existing function in the split fixture. Google lists permissions and roles among the emulator limitations. The November 21 managed fixture tested a schema-wide function grant. Individual-function grants were exercised on the two local engines in this run. [Managed schema-wide grant](managed-queue-search-upsert-20261121.md) · [Emulator limitations](https://docs.cloud.google.com/spanner/docs/emulator) · [Setup statements and per-role results](evidence/managed-recheck-20261124/receipt.json)

As of this check, the managed Atom feed and BigQuery both reached November 18, when QueryData for Spanner Graph was announced. The managed HTML and Developer Knowledge copies displayed November 10. Omni's release notes displayed October 29 and did not mention the exact r5.2 tag. The newest published LTS tag was `2026.r4-lts.3`, and the latest Emulator release was `1.5.59`. [Managed feed](https://docs.cloud.google.com/feeds/spanner-release-notes.xml) · [Omni release notes](https://docs.cloud.google.com/spanner-omni/release-notes) · [Emulator release](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/releases/tag/v1.5.59)
