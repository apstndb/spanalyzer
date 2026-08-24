---
type: Documentation Guide
title: Documentation Authority
description: How canonical, design, status, observational, and historical documents relate in spanalyzer.
tags: [documentation, authority, provenance, okf]
status: draft
sources:
  - id: repository-readme
    resource: ../../README.md
    title: spanalyzer README
  - id: querygen-status
    resource: ../../cmd/spanner-query-gen/IMPLEMENTATION_STATUS.md
    title: spanner-query-gen implementation status
  - id: research-index
    resource: ../../research/README.md
    title: spanalyzer legacy research index
  - id: okf-research-index
    resource: ../research/index.md
    title: spanalyzer OKF research authoring guide
---

# Documentation Authority

Repository documents serve different purposes. A nearby or newer-looking file
does not automatically override a canonical behavior surface.

## Evidence order

1. **Implemented behavior:** code, checked-in schemas, tests, package
   documentation, and command documentation are authoritative for what the
   current checkout does.
2. **Intended architecture:** design documents explain the target shape and
   explicitly include work that may not be implemented yet.
3. **Current drift and backlog:** implementation-status and TODO documents
   identify deliberate deferrals and open work; they are not behavior specs.
4. **Curated observations:** OKF observations synthesize claims across named
   sources while preserving their verification scope.
5. **Research evidence:** OKF-native `Research Note` concepts and legacy dated
   or environment-specific notes record how a conclusion was reached. They do
   not create a stable Spanner or spanalyzer contract.
6. **History:** archive documents and git history explain resolved work but do
   not describe the current surface.

When two layers disagree, inspect the implementation and its tests first, then
update stale guidance or record the unresolved discrepancy. The
[repository-document inventory](../references/repository-documents.md) maps
every tracked Markdown file to one of these roles.

## Single-body rule

An OKF concept may summarize and link canonical material, but it must not copy
large behavior descriptions or observation tables into a second writable body.
New research is itself canonical inside the bundle; a legacy research note
becomes canonical there only through `git mv`, never through a wrapper copy.
The `resource` and `sources` fields point to material that owns supporting
details. This keeps the bundle useful for discovery without creating a shadow
documentation tree.
