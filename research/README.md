# Research

Non-normative research notes: observation logs, design background, and
candidate ideas. Files here are evidence tied to specific probe environments
(usually Spanner Omni via spanemuboost), not a stable contract. When they
disagree with generated schemas, tests, or command documentation, check the
implementation surface directly before changing behavior.

This directory supports the repository's positioning (see the root README):
careful execution plan inspection, recorded so it does not have to be redone.

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
  `spanner-query-gen/reviews/trash/` AI review-exchange archive and the
  delivered spanner-hacks feedback drafts.

## Conventions

- One topic per file, named in `SCREAMING_SNAKE_CASE.md`; session-style
  verification logs carry a date suffix.
- Record the backend image, tool versions, and known caveats (empty database,
  PLAN-only) alongside observations.
- Keep retracted claims explicitly listed rather than silently rewritten, so
  they are not re-derived.
