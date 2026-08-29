---
type: Architecture
title: Knowledge Catalog Publication Boundary
description: How the repository prepares its OKF bundle for governed discovery without moving or overwriting the canonical source.
tags: [okf, knowledge-catalog, publication, provenance, governance]
status: draft
sources:
  - id: knowledge-catalog-okf-blog
    resource: https://cloud.google.com/blog/products/data-analytics/scale-okf-bundles-across-an-organization-with-knowledge-catalog
    title: Using OKF with Knowledge Catalog to serve context for agents
  - id: knowledge-catalog-okf-adapter
    resource: https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/74079e74a41890dc469f9a1ecf4719e8bcb27806/toolbox/mdcode/demo/okf/okf.ts
    title: Pinned Knowledge Catalog OKF staging adapter
  - id: publication-gate
    resource: ../../tools/okf-check/main.go
    title: spanalyzer OKF publication gate
---

# Knowledge Catalog Publication Boundary

The repository's [`knowledge`](../) tree is the single writable OKF source.
Knowledge Catalog can provide organization-wide search, IAM, hierarchy, and
context retrieval for that source, but a catalog entry is a publication
projection rather than a second authoring surface.[^knowledge-catalog-okf-blog]
Pulling catalog content over the repository or committing a generated
`catalog/` staging tree would violate that boundary.

## Publication candidate

The `okf-check` publication gate derives a candidate manifest from every
Markdown document in the bundle. It records:

- a path-derived Entry ID and its parent Entry ID;
- the source commit and whether the bundle matches that committed tree;
- per-document and whole-bundle SHA-256 digests; and
- for a committed tree, Git-tree-verified, commit-pinned GitHub URLs for the
  document, local resources, and local Markdown links.

The checked
[`publication inventory`](../../tools/okf-check/publication-inventory.json)
pins Entry identities and hierarchy separately from content hashes. A move or
deletion fails the normal gate until the inventory is deliberately updated.
Removed IDs remain in `retired_entries` with their former path, removal commit,
reason, and optional successor Entry ID so a sync can issue the required remote
delete without guessing from Git history.
This matters because a Knowledge Catalog push is an idempotent upsert of all
current entries, while removing a source document does not delete its former
Entry automatically.[^knowledge-catalog-okf-blog]

A dirty worktree may run the gate as a publication candidate. Such a candidate
retains original references and omits document source URLs because no remote
commit represents its bytes yet. Only a clean, committed tree can produce a
release manifest with commit-pinned URLs: the gate compares every document
and the publication inventory with their committed blobs, resolves each local
target from the Git tree rather than following the worktree filesystem, and
treats a repository-root reference as the source commit's root tree. All Git
reads used for publication disable local replacement objects so a `refs/replace`
mapping cannot make one commit ID resolve as another tree. CI uses
`--publication-require-clean`, and an operator can additionally write the
ephemeral manifest outside the repository with `--publication-manifest`.
Neither form performs a cloud write.

## Signal preservation

The current bundle uses OKF v0.2 lifecycle fields and producer-defined
frontmatter in addition to the core type, resource, provenance, and source
fields. The Knowledge Catalog article describes a 13-field `okf` Aspect,
including `status`, `verified`, freshness, attested-computation fields, and a
lossless `extra` field.[^knowledge-catalog-okf-blog] The pinned sample adapter,
however, currently maps only `type`, `resource`, `generated`, and `sources` and
does not preserve the repository's full signal surface or custom inventory
fields.[^knowledge-catalog-okf-adapter]

Consequently, this repository does not treat that sample adapter as a
lossless publisher yet. A future cloud sync must first pin a converter and
Aspect schema that round-trip every used frontmatter field, including unknown
producer fields, without inventing `verified`, `generated`, `stable`, or
freshness claims.

## Operational policy

The first catalog synchronization is manual and follows a reviewed manifest.
Automation may be added later only if it:

1. uses a repository-to-catalog direction of authority;
2. scopes the writer identity to the bundle's EntryGroup;
3. compares the prior and candidate Entry inventories before writing;
4. requires an explicit disposition for removed Entry IDs; and
5. reads back Entry count, hierarchy, body hash, and OKF signal fields into a
   publication receipt.

Search or LookupContext success is a discovery smoke test, not evidence that
the underlying Spanner observations were regenerated or verified. Their trust
continues to come from the environment-specific producers and evidence
lifecycle described elsewhere in this bundle.

[^knowledge-catalog-okf-blog]: [Using OKF with Knowledge Catalog to serve context for agents](https://cloud.google.com/blog/products/data-analytics/scale-okf-bundles-across-an-organization-with-knowledge-catalog)
[^knowledge-catalog-okf-adapter]: [Pinned Knowledge Catalog OKF staging adapter](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/74079e74a41890dc469f9a1ecf4719e8bcb27806/toolbox/mdcode/demo/okf/okf.ts)
