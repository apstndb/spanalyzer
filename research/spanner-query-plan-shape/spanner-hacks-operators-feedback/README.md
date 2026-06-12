# spanner-hacks operators.md feedback evidence

This directory keeps SQL plus rendered plan examples intended as follow-up feedback for <https://github.com/apstndb/spanner-hacks/blob/master/content/ja/operators.md>.

The rendered plans are generated with `spannerplan/plantree/reference` through
`tools/spanner-query-plan-shape --output reference`.

Checked upstream on 2026-05-07:

- Repository: `apstndb/spanner-hacks`
- File: `content/ja/operators.md`
- Source: <https://raw.githubusercontent.com/apstndb/spanner-hacks/master/content/ja/operators.md>

The current upstream document already includes many reproduction SQL snippets. These files preserve the heavier rendered-plan evidence separately so the root feedback file can stay readable.

## Files

- [../OPERATOR_VERIFICATION_FOLLOWUP.md](../OPERATOR_VERIFICATION_FOLLOWUP.md): raw follow-up checks for remaining uncertainty, including normal `SpoolScan`, Search Predicate raw-vs-reference mapping, Generate Relation candidates, Local Split Union, MiniBatch/RowCount reproduction sensitivity, and `Create Batch` scalar children.
- [docs-basic-leaf-unary-ja.md](docs-basic-leaf-unary-ja.md): execution-plan, leaf, and unary operator examples.
- [docs-binary-distributed-scalar-ja.md](docs-binary-distributed-scalar-ja.md): binary, n-ary, distributed, scalar, CTE, and TVF-adjacent examples.
- [docs-best-practices-ja.md](docs-best-practices-ja.md): SQL best-practices examples that are relevant to operator shape.
- [join-matrix-ja.md](join-matrix-ja.md): rendered plans for `INNER`, `LEFT`, and `RIGHT` joins with documented join hints.
- [subquery-join-matrix-ja.md](subquery-join-matrix-ja.md): rendered plans for `IN`, `EXISTS`, `NOT IN`, and `NOT EXISTS` with statement-level join hints.
- [dml-ja.md](dml-ja.md): DML plan examples, including INSERT variants, UPDATE, and DELETE.
- [tvf-and-function-hints-ja.md](tvf-and-function-hints-ja.md): change-stream TVF and function hint examples.
- [full-text-search-ja.md](full-text-search-ja.md): Full Text Search examples generated with `spannerplan v0.1.9`, including `SEARCH`, `SEARCH_SUBSTRING`, multi-column search, ranked search, numeric array predicates, mixed text/non-text predicates, non-stored-column back joins, and `Search Predicate` annotations in the `Predicates(identified by ID)` section.

## Remaining upstream follow-ups

- Add rendered plans, or cross-references to rendered plans, for sections that currently have SQL but no plan.
- Full Text Search has separate rendered-plan evidence in [full-text-search-ja.md](full-text-search-ja.md). Upstream has already incorporated the main Search Predicate observations; keep this file as the latest shareable PLAN evidence.
- Normal repeated-CTE `SpoolScan` was observed as raw `PlanNode.display_name: SpoolScan` with `metadata.spool_name`, not as `Scan.scan_type=SpoolScan`; upstream should either correct that statement or add a counterexample raw PlanNode.
- Distributed Apply, Semi Apply, Anti-Semi Apply, and Push Broadcast variants can use the join/subquery matrix files rather than repeating all cases inline.
- Keep `Distributed Merge Union` as the documented operator name, but render the raw plan as `Distributed Union` with ordering metadata such as `preserve_subquery_order: true`.
- `Generate Relation` and `Local Split Union` still need confirmed minimal reproductions in the local probe environment.
- `MiniBatchAssign`, `MiniBatchKeyOrder`, and `RowCount` can cross-reference the shared Cloud Spanner DBaaS example; the same SQL did not reproduce those operators in the empty synthetic Spanner Omni 2026.r1-beta database, so keep the wording environment-sensitive.
