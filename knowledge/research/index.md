# Research

New substantive research is authored here as OKF-native concepts. Existing
notes under the repository's [`research/`](../../research/README.md) tree are a
legacy evidence store and migrate incrementally; do not duplicate their bodies
here.

Place each research body at `knowledge/research/<area>/<note>.md` and declare
it as `type: Research Note`. Direct concept files under `knowledge/research/`
and Research Notes outside this subtree are rejected by the repository gate.

## Research Note policy

Use `type: Research Note` for environment-scoped probe results, design
background, and candidate ideas. These notes remain non-normative: code,
schemas, tests, package documentation, and command documentation define the
implemented spanalyzer surface, while official Spanner documentation defines
the public service contract.

Every Research Note must declare:

- a non-empty `title`, `description`, and `tags` list;
- an explicit `status` of `draft`, `stable`, or `deprecated`; and
- at least one `sources` entry naming the material or observation scope from
  which the note derives.

Use source IDs and matching Markdown footnotes when attributing individual
claims. Omit `generated`, `verified`, source credibility signals, and
`stale_after` unless the corresponding fact is known. A completed dated
session record may be `stable` as a record without implying that the observed
Spanner behavior is permanent.

```yaml
---
type: Research Note
title: Example title
description: One-sentence scope and result.
tags: [spanner, query-plan]
status: draft
sources:
  - id: probe
    resource: ../../../tools/spanner-query-plan-shape/example_cases.go
    title: Retained probe cases
---
```

## Migrating legacy notes

Migrate with `git mv` so the topic keeps one writable body and useful history.
In the same change:

1. add truthful OKF frontmatter and link the concept from the appropriate
   directory index;
2. remove the old path from
   [`legacy-research-markdown.txt`](../../tools/okf-check/legacy-research-markdown.txt);
3. update every inbound link; and
4. if the path appears in `catalog_source.info.local_evidence`, update it and
   regenerate the plan-vocabulary artifacts.

Do not use `Attested Computation` merely because a probe is executable. That
type requires the executor, receipt, and attester semantics defined by OKF;
ordinary probe code and expectation manifests belong in `sources`.
