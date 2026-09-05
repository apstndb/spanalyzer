# Runtime verification

Environment-scoped Cloud Spanner Emulator and Spanner Omni investigations live
here as canonical Research Notes. Coordination handoffs may link here, but do
not own the verification findings. Runtime identity, date, SQL or case scope,
source evidence, results and unverified boundaries travel with each note.

| Runtime and observation scope | Research Note |
| --- | --- |
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
