# Runtime verification

Environment-scoped managed Spanner, Cloud Spanner Emulator and Spanner Omni investigations live
here as canonical Research Notes. Coordination handoffs may link here, but do
not own the verification findings. Runtime identity, date, SQL or case scope,
source evidence, results and unverified boundaries travel with each note.

| Runtime and observation scope | Research Note |
| --- | --- |
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

Raw JSON is observation evidence. Probe source is a reproducer, and an
assertion summary is a derived check; neither is an execution attestation.
Historical reports recovered without full logs are identified as such. Each
source/image pair is independently pinned unless binary reproducibility was
actually established. Repeated runs without new findings update evidence
rather than create duplicate research bodies.
