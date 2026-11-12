# Runtime verification

Environment-scoped managed Spanner, Cloud Spanner Emulator and Spanner Omni investigations live
here as canonical Research Notes. Coordination handoffs may link here, but do
not own the verification findings. Runtime identity, date, SQL or case scope,
source evidence, results and unverified boundaries travel with each note.

| Runtime and observation scope | Research Note |
| --- | --- |
| Managed, Omni r4 LTS.2 / r5.1 and Emulator 1.5.59, 2026-11-12 JST | [FK backing-index predictability](fk-backing-index-predictability-20261112.md): 504 controlled steps, schema/source/name boundaries, nullable PK and reordered-key count bounds, shared physical indexes, three-cycle FK failures and verified owned-DB cleanup |
| Omni r4 LTS.2 / r5.1 and Emulator 1.5.59, 2026-11-12 JST | [Managed-control comparison](cross-engine-managed-recheck-20261112.md): 1,165 recorded steps, secure-context and privilege boundaries, distribution/compression, typed vectors, metadata tuples and current public releases |
| Omni r4 LTS.2, 2026-11-12 JST | [Mutation-count feedback integration](mutation-count-feedback-20261112.md): shared observed FK indexes, explicit scenarios and count contracts; original 31 commits plus a separate 36-commit revalidation including the runnable CLI workflow |
| Managed Spanner, 2026-11-12 JST | [Managed revalidation and parameterized views](managed-revalidation-20261112.md): four mirror commits, secure request context and FGAC, distribution aggregation, compression frames, scorer cohorts and explicit fixture corrections |
| Managed Spanner, 2026-11-10 JST, later runs | [Queue mutation-count matrix and follow-up](queue-mutation-count-managed-20261110.md): 319 + 43 attempts, 270 + 32 CommitStats, OR IGNORE/source inserts, RW renewal rejection, cascades, quota boundaries and explicit parser/API gaps |
| Managed Spanner, 2026-11-10 JST | [Mutation-count integration](mutation-count-managed-20261110.md): 197 CommitStats observations, index-sensitive partial writes, queue restrictions, lease failure and verified owned-DB cleanup |
| Omni r4 LTS.2 and Emulator 1.5.59, 2026-11-10 JST | [Mutation-count verification](mutation-count-20261110.md): 85 attempts each, API-specific automatic cells, same-key composition, shared FK indexes and unavailable Emulator statistics |
| Emulator 1.5.59 and Omni r4 LTS.2, 2026-11-09 JST | [Closed SQL choices](finite-sql-choices-20261109.md): correlated ranges, column selection, selected bindings and six-target PLAN collection |
| Local frontend and Emulator 1.5.59, 2026-11-08 | [Inspection continuation and row deletion policy folding](inspection-feedback-20261108.md): current-baseline reproduction, partial CLI artifacts and AST/live TTL reapplication |
| Emulator 1.5.59 and Omni r4 LTS.2, 2026-11-08 | [Repository hardening verification](repository-hardening-20261108.md): seven review fixes, dependency-safe DDL, explicit DB retention, standalone builds and isolated CEL errors |
| Omni r4 LTS.2, 2026-11-08 | [Compact PLAN summaries](plan-summary-omni-lts-20261108.md): full/summary parity, separate pinning warnings, output size and actual JSON/YAML schema checks |
| Managed Spanner, 2026-11-08 | [Managed revalidation and documentation audit](managed-revalidation-20261108.md): all three new mirror commits, search pagination, queue batch defaults and unchanged catalog surfaces |
| Emulator 1.5.59 and Omni r4 LTS.2 / r5.1, 2026-11-08 | [Runtime upgrades and compatibility verification](runtime-upgrades-20261108.md): new defaults, explicit r5 selection, expression-index metadata loss and a Remote Function PLAN regression candidate |
| Omni 2026.r2.1-beta, 2026-11-08 | [Scan-local PLAN contracts and partial collection](plan-report-access-paths-omni-20261108.md): seek/residual evidence, optional targets, structured parameters, annotations and non-executing DML |
| Omni and managed Spanner, 2026-09-16 | [Join type and method matrix](join-type-method-matrix.md): per-cell raw plans and result checks for Hash, Apply, Distributed Apply, Merge, push broadcast, semi and anti-semi forms |
| Emulator 1.5.58, Omni and managed Spanner, 2026-09-15 | [Index choice and release checks](fts-index-choice-and-emulator-1.5.58-20260915.md): competing secondary/n-gram indexes, version boundaries, result equivalence and DDL fixes |
| Emulator 1.5.56 and Omni 2026.r2.1-beta, 2026-09-06 | [Catalog identity](catalog-identity-20260906.md): creation uniqueness, exact DDL references, query lookup, and synonym divergence |
| Emulator 1.5.57, 2026-09-05 | [Release verification](cloud-spanner-emulator-1.5.57-verification.md): announced changes, survey results and metadata |
| Emulator 1.5.56 to 1.5.57, 2026-09-05 | [Source audit](cloud-spanner-emulator-1.5.57-source-audit.md): all 105 changed files and unannounced behavior |
| Omni 2026.r2.1-beta, August 2026 observations consolidated on 2026-09-05 | [Retained runtime verification](spanner-omni-2026.r2.1-beta-verification.md): direct metadata captures and identified historical run reports |
| Omni 2026.r2.1-beta and managed Spanner, 2026-08-24 | [PLAN comparison](spanner-omni-2026.r2.1-beta-managed-plan-comparison-20260824.md): pipe syntax, optimizer v9 and DML differences |

## Evidence ownership

The [evidence lifecycle](../../concepts/evidence-lifecycle.md) governs target
identity, redaction and interpretation. Producer-owned captures already under
`survey/` remain there and are linked, without copying their bodies. Selected
release observations, logs and historical probe sources are retained beside
these notes in [evidence](evidence/). The [migration provenance](evidence/migration-provenance.json)
records original and retained hashes, transformations and historical sources.
The [artifact hash manifest](evidence/artifact-sha256.json) covers the retained
research evidence, including that provenance record.
The [November 12 retention review](research-evidence-retention-20261112.md)
maps compact historical API/test outputs and retired duplicate captures to
their pinned originals. Original acquisition/import hashes and current
retained-file hashes describe different events; the disposition records both.

Raw JSON is observation evidence. Probe source is a reproducer, and an
assertion summary is a derived check; neither is an execution attestation.
Historical reports recovered without full logs are identified as such. Each
source/image pair is independently pinned unless binary reproducibility was
actually established. Repeated runs without new findings update evidence
rather than create duplicate research bodies.

## Engine histories

Read the [managed](managed/index.md), [Omni](omni/index.md) and
[Emulator](emulator/index.md) timelines oldest to newest. Their rows distinguish
changes, first coverage, rechecks and corrections. Same-day rows without exact
times have no claimed internal order. [History conventions](organization.md)
explain how to append observations without rewriting earlier conclusions.
The November 12 FK investigation, managed revalidation and its cross-engine
follow-up are the newest selected investigations. The mutation-count feedback fixture, November
10 write-accounting notes and older comparisons retain their own observation
dates and scopes.

## Retained historical research

These notes were imported on November 8 without rerunning their experiments.
They preserve dated findings, including failed attempts and later corrections,
without replacing current runtime pins, catalog manifests or projection
selectors. The [import receipt](evidence/history-import-20261108/receipt.json)
records original and retained hashes at the pinned import revision and its
publication transformations. Later timeline edits keep their own history. Raw
producer hashes within older receipts continue to identify their original
inputs; transformed publication copies are not byte-identical reproductions.

- [Managed Spanner, Omni r4, and Emulator comparison, 2026-10-05](cross-engine-comparison-20261005.md)
- [Managed documentation cases on Omni r5, r4 LTS patch and Emulator 1.5.59](cross-engine-doc-delta-20261030.md)
- [JSON and Change Streams across managed Spanner, Omni r4, and Emulator 1.5.58](cross-engine-json-and-change-streams-20261010.md)
- [Spanner documentation delta through October 30, reviewed October 31](documentation-delta-review-20261031.md)
- [October 31 documentation controls across engines and memefish gaps](documentation-runtime-and-memefish-audit-20261031.md)
- [Emulator GQL ordering is already broken in the first announced graph release](emulator-gql-ordering-release-bisect-20261031.md)
- [Emulator GQL ordering: verified loss of order metadata](emulator-gql-ordering-root-cause-20261030.md)
- [Emulator issue candidates after the GQL ordering report](emulator-issue-candidates-20261031.md)
- [Emulator 1.5.59 open-issue runtime follow-up](emulator-open-issue-runtime-20261030.md)
- [Parser and analyzer impact of the October 10 JSON verification](json-parser-analyzer-impact-20261010.md)
- [Managed Spanner ordinary column semantic options, 2026-09-18](managed-column-semantic-options-20260918.md)
- [Managed Spanner documentation-delta revalidation, 2026-10-29](managed-documentation-delta-revalidation-20261029.md)
- [Managed Spanner documentation history and incomplete preflight, 2026-10-21](managed-documentation-history-and-preflight-20261021.md)
- [Managed INFORMATION_SCHEMA IS_INHERITED rollout, 2026-09-16](managed-information-schema-is-inherited-rollout-20260916.md)
- [Managed Spanner JSON and documentation-diff verification, 2026-10-10](managed-json-comparison-and-change-streams-20261010.md)
- [Managed quantified comparison probes, 2026-09-21](managed-quantified-comparisons-20260921.md)
- [Managed Spanner September release verification, 2026-09-22](managed-release-notes-20260922.md)
- [Managed Spanner revalidation, 2026-09-25](managed-revalidation-20260925.md)
- [Managed Spanner revalidation, 2026-09-29](managed-revalidation-20260929.md)
- [Managed Spanner revalidation, 2026-10-05](managed-revalidation-20261005.md)
- [Managed Spanner revalidation, 2026-10-07](managed-revalidation-20261007.md)
- [Managed Spanner revalidation, 2026-10-09](managed-revalidation-20261009.md)
- [Managed Spanner revalidation, 2026-10-11](managed-revalidation-20261011.md)
- [Managed Spanner revalidation and query-statistics documentation, November 5](managed-revalidation-20261105.md)
- [Managed Spanner revalidation and Omni differences review, November 3](managed-revalidation-and-omni-differences-20261103.md)
- [Managed Spanner queues verification, 2026-09-18](managed-spanner-queues-20260918.md)
- [Managed SPANNER_SYS distribution-column rollout](managed-spanner-sys-distribution-columns-rollout-20260916.md)
- [Managed Spanner trial revalidation, 2026-10-23](managed-trial-revalidation-20261023.md)
- [Managed vector index rebuild verification, 2026-09-19](managed-vector-rebuild-20260919.md)
- [Memefish origin audit, 2026-09-23](memefish-origin-audit-20260923.md)
- [Runtime Verification History Organization](organization.md)
- [Spanner Omni 2026.r3-beta verification](spanner-omni-2026.r3-beta-verification.md)
- [Spanner Omni 2026.r4-lts verification](spanner-omni-2026.r4-lts-verification.md)
