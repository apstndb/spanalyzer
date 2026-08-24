# Legacy Research

This directory is the legacy store for existing non-normative observation
logs, design background, and candidate ideas. Maintain or migrate the current
files, but do not add new substantive research here. Author new research as
[`Research Note` concepts](../knowledge/research/index.md) inside the OKF
bundle so provenance, lifecycle, and progressive discovery travel with the
canonical body.

Legacy files remain evidence tied to specific probe environments (usually
Spanner Omni via spanemuboost), not a stable contract. When they disagree with
generated schemas, tests, or command documentation, check the implementation
surface directly before changing behavior.

This directory supports the repository's positioning (see the root README):
careful execution plan inspection, recorded so it does not have to be redone.
Curated cross-area concepts and all new research live in the
[`knowledge/`](../knowledge/index.md) OKF bundle. Each topic has one writable
body. The bundle's
[repository-document inventory](../knowledge/references/repository-documents.md)
also classifies every tracked legacy note while it remains at its current path.

Migration is incremental. Move one note with `git mv`, remove its path from
`tools/okf-check/legacy-research-markdown.txt`, add truthful OKF frontmatter,
and update inbound links plus any `catalog_source.info.local_evidence` path and
generated plan-vocabulary artifacts in the same change. Do not leave a wrapper
or copied summary at the old path.

## Areas

- [`spanner-query-plan-shape/`](spanner-query-plan-shape/): Spanner query
  plan observations — operator vocabulary, optimizer-version behavior, hint
  effects, and pattern-specific studies, produced with
  `tools/spanner-query-plan-shape` and the `plancontract` module.
- [`spanner-query-gen/`](spanner-query-gen/): design notes and verification
  logs for `cmd/spanner-query-gen` (optional parameters, plan contract
  candidates, session verification notes).
- [`archive/`](archive/): resolved material kept for the record (for example
  the resolved-TODO snapshot). Content whose role has fully ended is deleted
  instead; git history preserves it. Removed on 2026-06-12: the
  `spanner-query-gen/reviews/` AI review-exchange archive (its input role
  had ended) and the delivered spanner-hacks feedback drafts.

## Legacy conventions

- One topic per file, named in `SCREAMING_SNAKE_CASE.md`; session-style
  verification logs carry a date suffix.
- Record the backend image, tool versions, and known caveats (empty database,
  PLAN-only) alongside observations.
- Keep retracted claims explicitly listed rather than silently rewritten, so
  they are not re-derived.
