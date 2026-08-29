---
type: Reference
title: Repository Asset Inventory
description: Selected non-Markdown contracts, observational catalogs, executable evidence, and fixtures grouped by role.
resource: ../../
tags: [repository, schemas, fixtures, evidence, okf]
status: draft
asset_families:
  - id: contract-schemas
    paths:
      - schemas/*.schema.json
  - id: runtime-target-contract
    paths:
      - runtime_targets.json
  - id: okf-publication-contract
    paths:
      - tools/okf-check/publication-inventory.json
  - id: plan-vocabulary
    paths:
      - plancontract/planvocab/catalog.json
      - plancontract/planvocab/catalog_source.json
      - plancontract/planvocab/testdata/fixture_plans.json
  - id: information-schema-catalog
    paths:
      - information_schema_manifest.json
      - information_schema_projection_source.json
  - id: spanner-sys-catalog
    paths:
      - spanner_sys_manifest.json
  - id: survey-provenance-and-contracts
    paths:
      - survey/import-provenance.json
      - survey/infoschem/capture-definition.v0alpha1.json
      - survey/spannersys/capture-definition.v0alpha2.json
      - survey/schemas/*.schema.json
  - id: survey-observational-evidence
    paths:
      - survey/infoschem/evidence/managed/*.json
      - survey/infoschem/evidence/omni/*/*.json
      - survey/infoschem/evidence/emulator/*/*.json
      - survey/spannersys/evidence/*.json
  - id: executable-plan-evidence
    paths:
      - tools/spanner-query-plan-shape/*_cases.go
      - tools/spanner-query-plan-shape/*_test.go
      - tools/spanner-query-plan-shape/query_matrix.go
      - tools/spanner-query-plan-shape/testdata/*_expectations.json
  - id: query-generator-fixtures
    paths:
      - cmd/spanner-query-gen/testdata/plan_fixtures/*.json
      - testdata/querygen.yaml
  - id: analyzer-fixtures
    paths:
      - testdata/*.sql
      - testdata/protos/*.pb
      - testdata/protos/*.proto
      - testdata/protos/*/*.pb
      - testdata/protos/*/*.proto
---

# Repository Asset Inventory

This reference makes selected knowledge-bearing assets discoverable without
turning the OKF bundle into a second listing of the Git tree. The path rules in
frontmatter are checked against tracked files. They classify discovery scope;
they do not assert that every matching file supports every related claim.

## Contract schemas

The [`schemas`](../../schemas/) directory contains the machine-readable
contracts for the mutable `v0alpha1` plan-vocabulary formats and the `v1alpha`
query-generator configuration, plan-report, and plan-contract formats. The
schema files remain the canonical definitions.

The [`runtime_targets.json`](../../runtime_targets.json) contract centralizes
the descriptive release tags and platform-specific OCI manifest digests used
by every Emulator and Omni test or capture path. A tag remains human-readable;
the digest is the execution identity.

## Plan vocabulary

The observational vocabulary consists of the reviewed
[`catalog_source.json`](../../plancontract/planvocab/catalog_source.json), its
generated [`catalog.json`](../../plancontract/planvocab/catalog.json), and the
generated [`fixture_plans.json`](../../plancontract/planvocab/testdata/fixture_plans.json)
mirror. The source catalog's `info.local_evidence` list remains the authority
for hashed catalog provenance; this inventory does not replace it.

## OKF publication contract

The checked
[`publication-inventory.json`](../../tools/okf-check/publication-inventory.json)
pins the path-derived Entry IDs and parent hierarchy for a future Knowledge
Catalog projection. It deliberately excludes content hashes and deployment
coordinates: the publication gate derives hashes and commit-pinned source URLs
for each candidate, while cloud project, location, and EntryGroup remain
operator configuration.

## INFORMATION_SCHEMA catalog

The [`information_schema_projection_source.json`](../../information_schema_projection_source.json)
file explicitly selects one point-in-time managed observation and owns
analyzer-only exceptions. The generated
[`information_schema_manifest.json`](../../information_schema_manifest.json)
is the analyzer projection. It binds the selected capture, producer and
invocation hashes, registry export, projection-source hash, raw observed types,
rollout status, frontend projection overrides, and documentation-only columns
that are intentionally excluded from analysis. The corresponding JSON Schemas
are covered by the contract-schemas family.

## SPANNER_SYS catalog

The [`spanner_sys_manifest.json`](../../spanner_sys_manifest.json) file is the
analyzer's pinned live-primary SPANNER_SYS projection. It combines a structural
type registry with matching managed and Omni observations, explicit
default-deny entries, capture provenance, and documentation-conflict sidecars.
Its producer-owned JSON Schema and redacted managed/Omni captures are retained
in the nested [`survey`](../../survey/) module. Spanalyzer validates the
consumed prerelease contract without creating a second schema authority.

## Survey provenance and evidence

The strict [`import-provenance.json`](../../survey/import-provenance.json)
records the exact unpublished source commit/tree and initial spanalyzer import
commit/subtree, plus exclusions and dispositions. The survey schemas define
the producer's prerelease capture and manifest contracts. The versioned
[`capture definition`](../../survey/infoschem/capture-definition.v0alpha1.json)
freezes the INFORMATION_SCHEMA query, rolling probes, execution policy, and
hashed producer inputs. The independent
[`SPANNER_SYS v0alpha2 definition`](../../survey/spannersys/capture-definition.v0alpha2.json)
closes the same provenance gap without reinterpreting the immutable v0alpha1
manifest sidecars. INFORMATION_SCHEMA captures retain redacted names, raw
types, ordinals, and bounded rolling-column queryability for managed, Omni, and
Emulator targets. SPANNER_SYS v0alpha2 captures additionally bind those
redacted names, raw types, and ordinals to the producer source, invocation,
point-in-time managed timestamp, or exact Omni platform manifest.

## Executable plan evidence

The [`spanner-query-plan-shape`](../../tools/spanner-query-plan-shape/)
developer tool retains case selectors, deterministic tests, and structural
expectation manifests. Together they provide executable evidence for the
environment-scoped notes authored under [`knowledge/research`](../research/)
or retained in the legacy
[`research/spanner-query-plan-shape`](../../research/spanner-query-plan-shape/)
tree.

## Query-generator fixtures

The query-generator fixture family includes the representative
[`plan_fixtures`](../../cmd/spanner-query-gen/testdata/plan_fixtures/) used by
plan-report and classification tests, plus the repository's
[`querygen.yaml`](../../testdata/querygen.yaml) example configuration.

## Analyzer fixtures

The [`testdata`](../../testdata/) tree contains reusable Spanner DDL, Protocol
Buffers source and descriptor sets, and analyzer inputs. These fixtures support
tests but do not define the analyzer's public contract independently of the
code and schemas that consume them.

## Explicit exclusions

General implementation sources, module and checksum files, CI and linter
configuration, IDE metadata, license and ignore files, ignored local notes,
and temporary artifacts are intentionally outside this inventory. A concept
may still cite one of those files when a specific claim actually derives from
it.
