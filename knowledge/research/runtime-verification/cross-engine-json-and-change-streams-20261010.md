---
type: Research Note
title: "JSON and Change Streams across managed Spanner, Omni r4, and Emulator 1.5.58"
description: "Matched October 10 JSON and conditional-expression probes, graph conversion differences, and 100 ms change-stream heartbeats in both dialects on exact arm64 container images."
tags: [spanner, managed, omni, emulator, json, change-streams, verification]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: managed
    resource: managed-json-comparison-and-change-streams-20261010.md
    title: "Earlier October 10 managed baseline and pinned documentation diff"
  - id: receipt
    resource: evidence/json-cross-engine-20261010/receipt.json
    title: "Runtime identities, dependencies, observation times, and cleanup"
  - id: comparison
    resource: evidence/json-cross-engine-20261010/comparison.json
    title: "Matched cases, error codes, values, and heartbeat timestamps"
  - id: omni
    resource: evidence/json-cross-engine-20261010/omni.jsonl
    title: "Complete Omni observations and cleanup records"
  - id: emulator
    resource: evidence/json-cross-engine-20261010/emulator.jsonl
    title: "Complete Emulator observations and cleanup records"
  - id: emulator-source
    resource: evidence/json-cross-engine-20261010/emulator-source-receipt.json
    title: "Release-tagged source for heartbeat validation and generation"
  - id: emulator-release
    resource: https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/releases/tag/v1.5.58
    title: "Official Emulator release selected for this investigation"
---

# JSON and Change Streams comparison, October 10

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Omni **2026.r4-lts** matches all 125 managed constant-case outcome codes,
including all 99 successful result sets. Emulator **1.5.58** does not implement
the new JSON comparisons, ordering, grouping, or ordinary casts to JSON.
Both container engines accept 100 ms Change Streams heartbeats and return
heartbeat records in GoogleSQL and PostgreSQL.

These are first observations of this corpus on the selected images, not
version-paired changes. The October 8 documentation update does not establish
a runtime introduction date: the tested r4 image and the Emulator release
were already published before that update.

## Identity and evidence

The managed baseline was collected October 10, 01:24-01:40 UTC / 10:24-10:40
JST. It was reused without another managed run. The new container observations
span 01:50:39-01:54:01 UTC / 10:50:39-10:54:01 JST:

| Target | Version / platform | Exact platform manifest |
| --- | --- | --- |
| Omni | 2026.r4-lts, linux/arm64 | `sha256:9f10586c5de366bec5c0ff6cf542a67c21c8363e8a12a3deb9c372a07c9aa8b6` |
| Emulator | 1.5.58, linux/arm64 | `sha256:5f42f55ad4129ea0d86cf60f0d385e8a6be60a7e8e0cc791adcdedc8982bcb55` |

Registry manifests were retrieved again. Both platform digests match the
[October 5 comparison](cross-engine-comparison-20261005.md); no image transition
was measured. An active Omni container inspection confirms its exact image
and `start-single-server --listen-addresses=0.0.0.0` command. Runtime platform
inspection and pinned local image descriptors confirm arm64 for both runs.

The repository's older runtime_targets.json pins are unchanged. These runs
use explicit image overrides. The scratch-only module uses Go 1.26.8, Cloud
Spanner Go SDK v1.91.0, and spanemuboost v0.4.10-alpha.1; build metadata is
retained. The SDK is the same version as the managed baseline. The two case
JSON files are byte-identical to that baseline. Go clients call the services
directly, without a local SQL parser.

Each runtime emits 230 records: bootstrap identity, 125 constant cases,
owned GoogleSQL/PostgreSQL fixtures, and cleanup. This is a bounded capability
probe, not a rerun of the repository's complete integration suites.

## Comparison

| Behavior | Managed baseline | Omni r4 LTS | Emulator 1.5.58 |
| --- | --- | --- | --- |
| JSON/JSON equality and ordering | Executes | Same | InvalidArgument; JSON is not comparable |
| JSON ORDER BY / GROUP BY / DISTINCT | Executes | Same, including stored columns | InvalidArgument; JSON is not orderable/groupable |
| Comparison-specific scalar/array coercion | Executes for documented types | Same | InvalidArgument signatures |
| Ordinary scalar, ARRAY, STRUCT, PROTO, ENUM CAST to JSON | Executes | Same | InvalidArgument casts |
| JSON identity and bare SQL NULL casts | Execute | Same | Execute; these are exceptions to the ordinary-cast gap |
| GRAPH_ELEMENT and GRAPH_PATH CAST to JSON | Unimplemented | Unimplemented | InvalidArgument |
| TO_JSON(node) | Executes | Executes | InvalidArgument signature |
| TO_JSON(path) | Executes | Executes | Executes, with JSON result metadata |
| Conditional evaluation-error suppression | Executes | Same | Same on the tested cases |
| Procedural IF ... THEN through SQL API | Syntax error | Syntax error | Syntax error |
| Heartbeat 100 / 999 / 1,000 / 300,000 ms | Accepts | Same | Same, both dialects |
| Heartbeat 99 / 300,001 ms | InvalidArgument | Same | OutOfRange |
| Omitted / NULL read_options | Accepts | Same | Same, both dialects |
| Empty or nonempty read_options array | InvalidArgument | Same | Unimplemented, with the same null-only requirement |
| Option-only ALTER CHANGE STREAM | Accepts; tracking remains unchanged | Same | Accepts; tracking remains unchanged, but option rows are absent |
| Old FOR ALL SET OPTIONS / SET examples | Syntax errors | Same | Same |

Omni's constant corpus has 99 OK, 12 InvalidArgument, 13 OutOfRange, and one
Unimplemented response. Codes and rows match managed for all 125 cases; the
successful result-set comparison covers 99 cases. The ROW_NUMBER control's
Unimplemented response concerns ROW_NUMBER, not JSON ordering.

Emulator has 15 OK, 102 InvalidArgument, and eight OutOfRange responses. It
matches 35 managed outcome codes and all fifteen mutually successful constant
result sets. Ninety codes differ. Empty rows accompanying errors are not
successful result-set equality; they are excluded from the successful-result
comparison.

Emulator's JSON comparison/cast failures are semantic analysis errors, not
SQL grammar failures. SAFE_CAST also rejects unsupported source-to-JSON type
conversions during analysis; it does not turn them into SQL NULL. Existing
TO_JSON(NULL), TO_JSON(infinity), JSON identity, and bare NULL casts still
execute, so the observations do not imply that all JSON functionality is
absent.

## Graph conversion controls

Each engine creates the same four-node, one-edge graph. Ordinary node/edge
queries return four and one rows respectively. Managed and Omni cast an
INT64 node property to JSON, while Emulator rejects that ordinary INT64 cast.

Managed and Omni return Unimplemented for node/path CAST even though TO_JSON
of the same values works. Emulator rejects TO_JSON(node), but TO_JSON(path)
returns a JSON array containing node, edge, and node objects. Its metadata is
JSON, rather than a JSON-formatted STRING.

The path payloads preserve the tested node properties and edge endpoint links,
but their encodings differ: Emulator uses different identifier strings and
omits element_definition_name. Successful conversion therefore does not
establish serialization-byte parity across engines. No general stability
promise is made for these graph encodings.

## Conditional results and physical execution

COALESCE, IFNULL, IF, simple/searched CASE, and scalar-subquery controls return
the selected value while suppressing unchosen division-by-zero errors.
Selected errors return OutOfRange. An unresolved identifier in an unchosen
branch still produces an analysis error on every engine.

The two table-backed Omni PROFILE cases return 10, scan one row, and retain
fallback Scan nodes without execution statistics, matching the earlier managed
observation. This does not reproduce physical execution of unchosen subqueries
or prove a universal lazy-execution guarantee. Emulator returns the same
values but supplies no plan nodes or rows_scanned statistic; its profiles
cannot establish which unchosen scans ran.

The procedural IF recommendation remains unavailable through this tested SQL
API on all three engines. No separate procedural execution interface was
examined.

## Heartbeat records and arrival times

The bounded child-partition reads use a requested 100 ms interval and a
two-second end timestamp:

| Target / dialect | Heartbeat count | Median adjacent record timestamp gap |
| --- | --- | --- |
| Managed / GoogleSQL, earlier baseline | 18 | 100.058 ms |
| Managed / PostgreSQL, earlier baseline | 17 | 100.058 ms |
| Omni / GoogleSQL | 19 | 104.246 ms |
| Omni / PostgreSQL | 18 | 101.782 ms |
| Emulator / GoogleSQL | 18 | 100.001 ms |
| Emulator / PostgreSQL | 18 | 100.001 ms |

The final record can be clipped to the requested end timestamp. Counts and
timing differences do not identify a feature change. Record timestamps
describe stream progress; client arrivals also include SDK buffering, network
transport, and independently running host/container clocks.

Emulator's first record arrives separately, followed by a gap of approximately
1.78 seconds and a burst of subsequent rows. This proves that the requested
interval generates heartbeat progress records, not that the Go caller receives
one row every 100 ms. No delivery-latency guarantee is inferred for any engine.

The release-tagged Emulator source at
`a081a7e71aba5f4d05fc8c6b21ef1a269dcbac28` defines the minimum as 100 and maximum
as 300,000 in common/limits.h. The query validator enforces those bounds and
null-only read_options; the streaming handler uses the requested interval to
emit heartbeat records. The tagged source predates the October documentation
update, but no earlier runtime or source-transition audit dates introduction.

## ALTER scope and cleanup

Four valid ALTER statements in each dialect set exclude_ttl_deletes and
allow_txn_exclusion to true and false. INFORMATION_SCHEMA retains one selected
table and its Value/value column on both engines. Omni reports the requested
option values; Emulator's CHANGE_STREAM_OPTIONS reads all succeed but return
zero rows, including after the true settings. This is an additional metadata
difference. Acceptance does not establish filter enforcement, and absent
metadata alone does not prove that settings are ignored internally. TTL
deletion and transaction-exclusion filter enforcement were not exercised.

Both GoogleSQL and PostgreSQL databases in each runtime are deleted; subsequent
GetDatabase calls return NotFound. Both runtime Close calls succeed, and
independent Docker ancestor inspections show no remaining containers for the
tested images. No managed destination or credential is used by the container
runner. The source checkout's existing dirty work is preserved.

Negative label-propagation seeds and export-based analytics jobs remain
unverified. Cloud instance configurations and monitoring metrics have no
equivalent container control in this corpus. These boundaries remain separate
from the executable JSON and Change Streams results.
