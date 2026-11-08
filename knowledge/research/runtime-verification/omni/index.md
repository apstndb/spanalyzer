# Spanner Omni observation history

[All runtimes](../index.md) · [History conventions](../organization.md)

Read oldest to newest by **observation time**, not release announcement or
report consolidation date. Image names below are navigation labels; linked
reports/captures retain full digests and platforms. `IS` means
INFORMATION_SCHEMA column tuples; `SYS` means advertised SPANNER_SYS column
tuples. A newly covered case on an older image does not date its introduction.
Correction rows use the labeled correction date and retain the original
observation date. Same-day rows without exact times have no claimed order.

## Observations

| Observed at / runtime | Kind | Bounded result or change | Report and evidence |
| --- | --- | --- | --- |
| 2026-08-24; timezone unrecorded; r2.1, platform unrecorded | Baseline | All 15 then-documented pipe operators planned on r2.1 and managed. The report mentions an earlier r1 pipe rejection but lacks comparable full runtime provenance; no exact introduction date follows. Its pulled digest is not the later arm64 pin. | [Historical PLAN comparison](../spanner-omni-2026.r2.1-beta-managed-plan-comparison-20260824.md) |
| 2026-08-25T12:06:26.330524Z; r2.1 arm64 for IS | Baseline | IS 308, including SEARCH_UNNEST and EXPRESSION; SYS 50 tables / 539 columns. The SYS sidecar lacks a platform manifest and must not inherit the IS capture identity. Other recovered integration results have weaker historical provenance. Consolidated September 5 without a new run. | [r2.1 verification](../spanner-omni-2026.r2.1-beta-verification.md), [IS capture](../../../../survey/infoschem/evidence/omni/ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741/linux-arm64-f8b0ea3092e7.json), [SYS sidecar](../../../../survey/spannersys/evidence/omni-2026.r2.1-beta.json) |
| 2026-09-06; time/zone unrecorded; r2.1 arm64 | First coverage | Catalog naming, lookup and synonym cases compared with Emulator 1.5.56 arm64. Omni rejected the tested ALTER through a synonym while Emulator accepted it. Cross-engine difference, not an Omni version transition. | [Catalog identity](../catalog-identity-20260906.md), [runtime log](../evidence/catalog-identity-20260906/runtime.log) |
| 2026-09-15 UTC/JST; r2.1 arm64 | First coverage | Fixture-specific automatic index choices and successful result hashes agreed with managed across v4/v5/v6/v8/v9. Forced search-index acceptance at v5 did not imply automatic selection. Bounded column-case and UNIQUE nullability DDL controls also ran. | [Index-choice comparison](../fts-index-choice-and-emulator-1.5.58-20260915.md), [summary](../evidence/fts-index-choice-20260915/summary.json) |
| 2026-09-15 UTC / September 16 JST; r2.1 arm64 rerun and correction | Correction | Fresh 56-cell join matrix agreed with managed on the tested operator families and result multisets. It corrected earlier fixture interpretation and grouping, not engine behavior. | [Join matrix](../join-type-method-matrix.md), [per-cell audit](../evidence/join-matrix-20260916/audit-summary.json) |
| 2026-09-26; r3 arm64 against retained r2.1 captures | **Change** | IS **308 → 310** adds two IS_INHERITED columns; SYS **539 → 587** adds 48 distribution columns. No retained tuple was removed, retyped or reordered. The historical SYS baseline's missing platform pin limits provenance. | [r3 catalog comparison](../spanner-omni-2026.r3-beta-verification.md), [r2.1 baseline](../spanner-omni-2026.r2.1-beta-verification.md), [r3 receipt](../evidence/omni-r3-20260926/receipt.json) |
| 2026-09-26; paired r2.1/r3 arm64 runs | **Change** | Same-binary probes: approximate count rejected on r2.1 but returned 2 on r3; automatic parameters absent on r2.1 but present on r3 for DEFAULT/AUTO/ON. Invalid AUTO_PARAMETER accepted on r2.1 but rejected on r3. These establish image-to-image differences, not their release-day introduction. | [Paired feature probes](../spanner-omni-2026.r3-beta-verification.md), [receipt](../evidence/omni-r3-20260926/receipt.json) |
| 2026-09-26; paired r2.1/r3 arm64 runs | First coverage | Graph semantic/key-uniqueness options, PostgreSQL operators/sampling and quantified rejection agreed on both images. Features announced for managed after September 3 already worked on r2.1; the r3 documentation cutoff is not an absence boundary. | [Post-cutoff comparison](../spanner-omni-2026.r3-beta-verification.md), [coverage](../evidence/omni-r3-20260926/post-cutoff-coverage.json) |
| 2026-09-26 UTC, crossing September 27 JST; paired r2.1/r3 arm64 runs | First coverage | Queue lifecycle and rollback controls worked on both images. Ordinary 1 ms stale read succeeded, while stale RECEIVE required strong concurrency. Scalar-array renewal errored; array-bound renewal succeeded, also with AUTO_PARAMETER OFF. This does not place queue introduction in r3. | [Queue follow-up](../spanner-omni-2026.r3-beta-verification.md), [paired receipt](../evidence/omni-r3-20260926/queue-receipt.json) |
| Correction recorded 2026-09-29; original managed observation September 18 UTC | Correction | The r3 report's managed comparison was corrected from successful zero-row lease behavior to `InvalidArgument`. Raw managed and Omni records were unchanged; no runtime change occurred on the correction date. | [Dated correction](../spanner-omni-2026.r3-beta-verification.md), [original managed controls](../evidence/queues-20260918/lease-r2.jsonl) |
| 2026-10-04 UTC; r4 LTS arm64 against retained r3 | **Change** | SYS **587 → 598**, adding the 11-column ACTIVE_BATCH_OPERATIONS table; its empty-deployment SELECT succeeded. Graph metadata newly emitted `allow_view_with_volatile_column=FALSE`. IS remained 310. Explicit graph-option values still returned `Unimplemented`; metadata is not usable option semantics. | [r4 catalog/graph comparison](../spanner-omni-2026.r4-lts-verification.md), [r3 baseline](../spanner-omni-2026.r3-beta-verification.md), [r4 receipt](../evidence/omni-r4-20261004/receipt.json) |
| 2026-10-04 UTC; r4 LTS arm64 | First coverage | Developer TLS/mTLS and password positive/negative controls ran using Go SDK v1.94.0. No prior identical runtime probe dates introduction. Container startup needed explicit `--listen-addresses=0.0.0.0`; initial failures preceded SQL. | [Authentication and startup scope](../spanner-omni-2026.r4-lts-verification.md), [receipt](../evidence/omni-r4-20261004/receipt.json) |
| 2026-10-04 UTC; r4 LTS arm64 | Recheck | Matched SQL/queue outcomes agreed with r3. The same two top-level stale plan expectations failed; the separately run generator suite passed. Failed expectations remain failed gates, not a passing integration suite. | [r4 bounded comparison](../spanner-omni-2026.r4-lts-verification.md), [retained r3 results](../spanner-omni-2026.r3-beta-verification.md) |
| 2026-10-05 UTC/JST; same r4 LTS arm64 digest | Recheck | IS 310 / SYS 598, bounded SQL/graph/queue and lease controls confirmed the October 4 r4 observations in a fresh cross-engine run. No new image or queue-support transition. | [Cross-engine comparison](../cross-engine-comparison-20261005.md), [receipt](../evidence/cross-engine-20261005/receipt.json), [command times](../evidence/cross-engine-20261005/container-commands.json) |
| 2026-10-10 01:50:39-01:53:55 UTC / 10:50:39-10:53:55 JST; same r4 LTS arm64 digest | First coverage | All 125 managed JSON/conditional constant-case codes and 99 successful result sets match. Stored JSON sorting/grouping and PROTO/ENUM casts execute; graph node/path CAST remains Unimplemented while TO_JSON succeeds. Both dialects produce 100 ms heartbeat records and reject 99 ms. No image or feature-introduction transition is inferred. | [Focused cross-engine comparison](../cross-engine-json-and-change-streams-20261010.md), [exact-image receipt](../evidence/json-cross-engine-20261010/receipt.json), [Omni records](../evidence/json-cross-engine-20261010/omni.jsonl) |
| 2026-10-29 14:53-15:23 UTC / October 29-30 JST | New image / first coverage | Public 2026.r5 arm64 digest e506672206e0 executes SET NULL and all tested available-view graph options, unlike the LTS patch. JSONB, bounded paths, search and HLL match the earlier managed cases; SYS remains 51/598. Official release/download pages still name r4 LTS, so no r5 announcement or support promise is inferred. | [Shared comparison](../cross-engine-doc-delta-20261030.md), [r5 records](../evidence/cross-engine-doc-delta-20261029/omni-r5.jsonl) |
| 2026-10-29 15:01-15:24 UTC / October 30 JST | New LTS patch / first coverage | Public 2026.r4-lts.1 arm64 digest 5e3cf199f5ab rejects SET NULL and explicit allow_view_with_volatile_column while omitted-option graphs, JSONB order, search, GQL propagation and HLL execute. Full constant codes/values match managed; this is new-case coverage, not a measured r4-to-patch feature introduction date. | [Shared comparison](../cross-engine-doc-delta-20261030.md), [LTS records](../evidence/cross-engine-doc-delta-20261029/omni-lts.jsonl) |
| 2026-10-31 JST; r5 and r4-lts.1 arm64, exact digests retained | First coverage / recheck | All fifteen pipe controls execute, payload declaration names match the managed restriction, and valid grouping aliases execute. Integer non-NULL search sort keys work. Scorer and PostgreSQL bucket-boundary controls distinguish value semantics from accepted options. | [Runtime and parser follow-up](../documentation-runtime-and-memefish-audit-20261031.md) |
| 2026-11-08 JST; r2.1-beta arm64 | First focused coverage | Scan-local seek/residual contracts, partial PLAN collection and DML non-execution controls run on the deliberately selected older image. This is new case coverage, not an older release becoming current. | [Access-path verification](../plan-report-access-paths-omni-20261108.md) |
| 2026-11-08 JST; r4-lts.2 / r5.1 arm64 | Recheck / first focused coverage | LTS.2 remains the default and r5.1 is selectable. Local compatibility fixtures execute on both; r5.1 Remote Function PLAN returns Internal in the retained control. This is a regression candidate, not a demonstrated release-wide root cause. | [Runtime upgrade comparison](../runtime-upgrades-20261108.md) |
| 2026-11-08 JST; r4-lts.2 arm64 | First focused coverage | Full and compact PLAN reports agree on target identities, digests and contract outcomes; output sizes are measured without a latency claim. Dependency-safe DDL also reapplies from AST and live metadata. | [PLAN summary](../plan-summary-omni-lts-20261108.md), [repository hardening](../repository-hardening-20261108.md) |
| 2026-11-09 JST; r4-lts.2 arm64 | First focused coverage | All six correlated-range / column-choice statements return expected row sets and nonempty PLANs; the collector produces six successful targets. This verifies a generator feature without an optimizer or performance claim. | [SQL-choice verification](../finite-sql-choices-20261109.md), [receipt](../evidence/finite-sql-choices-20261109/receipt.json) |

## Publication and source context

- **September 5 consolidation:** the r2.1 report recovered August observations
  and historical integration summaries. It did not execute Omni again, and
  historical results without exact times remain identified as such.
- **September 17 announcement, inspected September 26:** r3 stated managed
  coverage through September 3. Direct HTML contained the announcement while
  the fetched Developer Knowledge release page lagged. The [r3 report](../spanner-omni-2026.r3-beta-verification.md)
  preserves this source discrepancy separately from paired runtime results.
- **September 30 announcement, inspected October 4:** the vendor identified r4
  as GA/LTS with coverage through September 18. Image creation on September 29,
  release on September 30, and probes on October 4 are distinct dates. The
  [r4 report](../spanner-omni-2026.r4-lts-verification.md) separates tested
  Developer behavior from untested Commercial scale, Helm and upgrade claims.

- **October 31 documentation review:** the October 29 mirror capture and current
  official download body now identify **2026.r4-lts.1**, dated October 28, and
  update five installation/upgrade pages. The earlier exact-image test remains
  dated October 29. The download page does not list r5; registry observation and
  release documentation remain distinct. See the [source audit](../documentation-delta-review-20261031.md).

- **November 3 documentation recheck:** release notes now explicitly identify
  **r5, dated October 29**, with managed incorporation through October 27, and
  **r4-lts.1, dated October 28**. The download page advances to r5/Helm 1.1.0.
  The LTS patch notice first appears in the October 31 capture, and the r5
  announcement in the November 1 capture;
  earlier absence observations retain their original dates. The unchanged
  differences page's high-scale ANN exclusion requires qualification against
  Commercial worker documentation. No Omni runtime is executed in this recheck.
  See the [managed and differences review](../managed-revalidation-and-omni-differences-20261103.md).

Full image identities stay in the linked evidence. Repository runtime pin and
catalog-selection changes are separate from adding an observation to this index.
