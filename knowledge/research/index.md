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

Use `git mv` when one coherent legacy body remains one `Research Note`. When
the useful material has different owners or lifecycles, use semantic
disposition instead: split it, merge it with a compatible evidence unit,
absorb current behavior into canonical documentation or executable evidence,
or retire material whose role has ended.

Every body-changing migration completes its source-hashed entry in the
[legacy research migration ledger](../references/legacy-research-migrations.md).
The entry identifies each retained claim group or retired section and its
successor or controlled retirement reason. In the same change:

1. add truthful OKF frontmatter to new concepts and link them from the
   appropriate directory index;
2. remove the old path from
   [`legacy-research-markdown.txt`](../../tools/okf-check/legacy-research-markdown.txt);
3. update every inbound link; and
4. if the path was a hashed planvocab input, map the catalog selectors it
   supported to retained evidence and regenerate the plan-vocabulary
   artifacts.

Executable probe code is a reproducer, an expectation manifest is an
assertion, and a raw result or scoped run record is observation evidence.
Preserve the datum, backend and version scope, case identity, outcome class,
derivation, and immutable evidence before replacing a hashed narrative with
non-narrative inputs.

Do not use `Attested Computation` merely because a probe is executable. That
type requires the executor, receipt, and attester semantics defined by OKF;
ordinary probe code and expectation manifests belong in `sources`.
