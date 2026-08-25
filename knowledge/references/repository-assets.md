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
  - id: plan-vocabulary
    paths:
      - plancontract/planvocab/catalog.json
      - plancontract/planvocab/catalog_source.json
      - plancontract/planvocab/testdata/fixture_plans.json
  - id: information-schema-catalog
    paths:
      - information_schema_manifest.json
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

## Plan vocabulary

The observational vocabulary consists of the reviewed
[`catalog_source.json`](../../plancontract/planvocab/catalog_source.json), its
generated [`catalog.json`](../../plancontract/planvocab/catalog.json), and the
generated [`fixture_plans.json`](../../plancontract/planvocab/testdata/fixture_plans.json)
mirror. The source catalog's `info.local_evidence` list remains the authority
for hashed catalog provenance; this inventory does not replace it.

## INFORMATION_SCHEMA catalog

The [`information_schema_manifest.json`](../../information_schema_manifest.json)
file is the analyzer's live-primary INFORMATION_SCHEMA projection. It records
the pinned survey commit and export hash, raw observed types, rollout status,
explicit frontend projection overrides, and documentation-only columns that
are intentionally excluded from analysis. The corresponding JSON Schema is
covered by the contract-schemas family.

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
