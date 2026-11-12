# Write analysis

Experimental analysis of Spanner Mutation API and DML effects, using DDL as
schema input. Mutation counts, transaction bytes, lock costs and query plans
are separate measurements.

- [Implemented mutation-count analysis](../../concepts/mutation-count-analysis.md):
  current API/CLI entry points and a consolidated verification map.
- [Scenario integration and latest feedback](../runtime-verification/mutation-count-feedback-20261112.md):
  observed FK indexes, explicit old-row and cardinality inputs, measured counts
  and count contracts, including a reproducible command workflow.
- [DDL/DML reference audit, November 10](mutation-count-reference-audit-20261110.md):
  every GoogleSQL reference family, source-backed dependency corrections,
  implementation alternatives and prioritized adapter/accounting gaps.
- [Mutation-count alternatives, November 10](mutation-count-alternatives-20261110.md):
  three executable models, external wire input, schema reconciliation, feedback
  dispositions and a staged integration proposal.
- [Local runtime observations](../runtime-verification/mutation-count-20261110.md):
  exact image identities, retained CommitStats, unavailable Emulator statistics,
  shared foreign-key indexes and API-specific counting behavior.
- [Managed integration](../runtime-verification/mutation-count-managed-20261110.md):
  retained partial-flow boundary, 197 CommitStats replays, partial updates,
  queue operation restrictions and the mutation-limit enforcement difference.
- [Queue mutation-count matrix](../runtime-verification/queue-mutation-count-managed-20261110.md):
  payload/key variation, missing-message Ack, same-key composition, cascades,
  exact mutation-limit boundaries and the completed 43-attempt OR IGNORE,
  source-insert, assertion and read-write renewal follow-up.
