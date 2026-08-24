---
type: Architecture
title: Query-Plan Inspection
description: How plan probes, normalization, contracts, vocabulary drift detection, and recorded evidence fit together.
tags: [spanner, query-plan, plancontract, planvocab, research]
status: draft
sources:
  - id: planvocab-readme
    resource: ../../plancontract/planvocab/README.md
    title: planvocab README
  - id: probe-readme
    resource: ../../tools/spanner-query-plan-shape/README.md
    title: spanner-query-plan-shape README
  - id: evidence-index
    resource: ../../research/spanner-query-plan-shape/README.md
    title: legacy spanner-query-plan-shape research index
  - id: okf-research-index
    resource: ../research/index.md
    title: spanalyzer OKF research authoring guide
  - id: planvocab-source
    resource: ../../plancontract/planvocab/catalog_source.json
    title: reviewed plan-vocabulary source catalog
  - id: planvocab-catalog
    resource: ../../plancontract/planvocab/catalog.json
    title: generated plan-vocabulary catalog
  - id: planvocab-schema
    resource: ../../schemas/spanalyzer.planvocab.v0alpha1.schema.json
    title: plan-vocabulary catalog schema
  - id: planvocab-expectations-schema
    resource: ../../schemas/spanalyzer.planvocab-expectations.v0alpha1.schema.json
    title: plan-vocabulary expectation schema
  - id: query-matrix
    resource: ../../tools/spanner-query-plan-shape/query_matrix.go
    title: plan-shape query matrix
  - id: plan-expectations
    resource: ../../tools/spanner-query-plan-shape/testdata/
    title: plan-shape expectation manifests
---

# Query-Plan Inspection

The plan tooling turns one-off Spanner plan inspection into reproducible,
structural evidence. It deliberately separates observation from contract:

1. [`spanner-query-plan-shape`](../../tools/spanner-query-plan-shape/README.md)
   submits controlled queries and captures raw or rendered plans.
2. [`plancontract`](../../plancontract/) normalizes plans and evaluates the
   structural properties selected by a caller.
3. [`planvocab`](../../plancontract/planvocab/README.md) compares raw operator
   metadata and child links with a provenance-stamped observational catalog,
   surfacing unknown vocabulary instead of silently accepting it. Its reviewed
   source, generated catalog, and schemas are separate assets with explicit
   provenance.[^planvocab-source][^planvocab-catalog][^planvocab-schema][^planvocab-expectations-schema]
4. [OKF-native research notes](../research/index.md), together with the
   [legacy plan-shape research](../../research/spanner-query-plan-shape/README.md),
   retain environments, controls, negative results, and interpretation limits.
5. [OKF observations](../observations/index.md) curate cross-source conclusions
   that are useful beyond a single probe session.

PLAN output establishes optimizer topology, not runtime performance. Claims
about latency, CPU, bytes, or cardinality still require PROFILE or workload
evidence from a runtime that exposes those measurements.

The probe implementation, expectation manifests, and tests remain the
executable authority.[^query-matrix][^plan-expectations] Research and OKF
documents explain why those checks exist and which conclusions they support.

[^planvocab-source]: Reviewed source whose `info.local_evidence` list controls catalog provenance.
[^planvocab-catalog]: Generated observational vocabulary consumed by the checker.
[^planvocab-schema]: JSON Schema for the plan-vocabulary catalog.
[^planvocab-expectations-schema]: JSON Schema for structural expectation manifests.
[^query-matrix]: Query and selector registry for the plan-shape probe tool.
[^plan-expectations]: Checked-in structural expectation manifests used by probe tests.
