# spanner-query-plan-shape

`spanner-query-plan-shape` is a developer probe for inspecting Cloud Spanner
query plan shapes with Spanner Omni through
[`spanemuboost`](https://github.com/apstndb/spanemuboost).

It is intentionally under `tools/` rather than `cmd/`: the output is for
designing and reviewing `spanner-query-gen` plan normalization and contract
rules, not for the public CLI surface.

This tool depends on Spanner Omni execution-plan support, which is Preview /
Pre-GA in the official Spanner Omni documentation. Use it for design review,
testing, prototyping, and normalization experiments, not as a production
performance guarantee.

## Usage

List every built-in selector, its query count, and a short description without
starting Spanner Omni:

```sh
go run ./tools/spanner-query-plan-shape --list-cases
go run ./tools/spanner-query-plan-shape --list-cases --list-cases-format json
```

The default `all` selector is intentionally a small smoke test, not a union of
all selectors. It inspects the built-in Push Broadcast Hash Join and Hash Join
examples:

```sh
go run ./tools/spanner-query-plan-shape
```

Inspect one built-in example:

```sh
go run ./tools/spanner-query-plan-shape --case push_broadcast_hash_join
go run ./tools/spanner-query-plan-shape --case hash_join
```

Inspect Google Cloud documentation-derived query examples:

```sh
go run ./tools/spanner-query-plan-shape --case docs --output compact-tree-metadata
```

Inspect optimizer-version gap probes. These target documented optimizer-version
items that are hard to prove in an empty synthetic database, such as cost-based
choices, `WITH` plan choices, large `IN` lists, informational foreign keys,
full outer join pushdown, and version-specific join/group examples:

```sh
go run ./tools/spanner-query-plan-shape \
  --case optimizer_gaps \
  --optimizer-version-matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

Compare join elimination across interleaving, enforced foreign keys,
informational foreign keys, and an unconstrained control. The informational-FK
queries vary whether the referencing column leads the primary key, appears
later in it, or remains outside it, as well as the projection direction and the
`USE_UNENFORCED_FOREIGN_KEY` hint. Earlier observations disagreed on the
optimizer-version boundary, and primary-key placement was a plausible
confounding variable that this matrix tests explicitly:

```sh
go run ./tools/spanner-query-plan-shape \
  --case join_elimination \
  --optimizer-version-matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

Probe plausible catalog cross-products that are not established by the broader
documentation matrices, including outer/one-to-one merge joins, OFFSET links,
hash-join residuals, and simultaneous aggregate/minor-sort multiplicity:

```sh
go run ./tools/spanner-query-plan-shape \
  --case planvocab_inference \
  --output json \
  --continue-on-error
```

Pipe this raw output through `planvocab-check` as documented in
[`plancontract/planvocab/README.md`](../../plancontract/planvocab/README.md) to
check both unknown vocabulary and positive operator requirements.

Use `--omni-image` when reproducing an observation from a specific Spanner Omni
release instead of the current spanemuboost default. Record the immutable image
digest alongside retained raw plans:

```sh
go run ./tools/spanner-query-plan-shape \
  --case join_elimination \
  --optimizer-version-diff \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output compact-tree-metadata \
  --continue-on-error
```

Use `compact-tree-metadata` for the canonical one-line research output. It keeps
operator metadata and child-link topology, which is important for joins, apply
nodes, scalar links, and predicate links:

```sh
go run ./tools/spanner-query-plan-shape --case hint_matrix --output compact-tree-metadata
```

`compact-tree-metadata` keeps scalar expression nodes out of the main operator
chain and renders visible child links as edges, such as
`Hash Join(-[Build]-> Scan, -[Probe]-> Scan)`. A child link is visible when the
child PlanNode kind is `RELATIONAL` or the link type is `Scalar`, matching
`spannerplan.QueryPlan.IsVisible`. Hidden scalar links are grouped by their
child PlanNode `DisplayName`, such as `Function[Residual Condition]`,
`Reference[Output]`, or `Array Constructor`. Links already rendered as tree
edges are omitted from the annotation.

Use `compact-tree` when parent/child topology matters, such as joins with
`Build` / `Probe` children or apply nodes with `Input` / `Map` children:

```sh
go run ./tools/spanner-query-plan-shape --case join_matrix --output compact-tree
```

`compact-tree` is still a one-line summary, but it omits metadata. Add
`--compact-tree-indexes` when PlanNode indexes are needed for cross-checking
against `nodes`, `yaml`, or `json` output.

Inspect Spanner DML syntax examples. These are analyzed in read-write
transactions because DML planning may require a read-write transaction even when
the tool only asks for `PLAN`:

```sh
go run ./tools/spanner-query-plan-shape --case dml --output compact-tree-metadata --continue-on-error
```

The 29-case selector has a retained expectation manifest at
`testdata/dml_expectations.json`: 28 plans require the expected `Apply
Mutations` operation and table, while the documented `ASSERT_ROWS_MODIFIED`
form records the pinned-Omni capability error. The focused Omni test repeats
all cases at optimizer versions 1 through 9 and pins the observed
version-sensitive compact-tree partitions, including the v9-only
`DELETE ... THEN RETURN` shape.

Inspect top-level system-procedure `CALL` plans and representative Query API
routing boundaries:

```sh
go run ./tools/spanner-query-plan-shape --case statement_surface --output json --continue-on-error
```

The positive cases cover literal and cast arguments to `cancel_query`, plus
`compact_all`; the manifest requires `Serialize Result`, `TVF`, and `Unit
Relation`. The focused Omni test also sends each positive through both
`ExecuteSql(PLAN)` and `ExecuteStreamingSql(PLAN)` and sets optimizer versions
1 through 8 in RPC `query_options`. CALL does not accept the SQL
`OPTIMIZER_VERSION` statement hint, so `--optimizer-version-matrix` is not the
version test for this selector. The remaining cases retain procedure-argument,
unknown-procedure, DDL-routing, batch-command, and procedural-language
capability boundaries without treating them as operator vocabulary.

Inspect a change-stream table-valued function probe:

```sh
go run ./tools/spanner-query-plan-shape --case tvf --output compact-tree-metadata --continue-on-error
```

The built-in TVF case creates a minimal table and `CREATE CHANGE STREAM
EverythingStream FOR ALL`, then runs `READ_EverythingStream(...)`.

Inspect Full Text Search probes with a dedicated schema containing generated
`TOKENLIST` columns and search indexes:

```sh
go run ./tools/spanner-query-plan-shape \
  --case full_text_search \
  --output compact-tree-metadata \
  --continue-on-error
```

This case intentionally does not reuse the documentation operator schema. It
creates only the tables, search indexes, and small property graph needed for
Full Text Search, then
checks `SEARCH`, `SEARCH_SUBSTRING`, multi-column search, mixed text and
non-text predicates, `SNIPPET`, `SCORE`, `TOKENLIST_CONCAT`, partitioned
ordered search indexes, numeric array search-index predicates, and a graph
traversal whose destination predicate uses a search index. The facet cases
compare a covering single aggregation with a search-only control and show a
multi-facet CTE being materialized once with `SpoolBuild` and read three times
with `SpoolScan`. Additional pairs retain `enhance_query` with its default
control and both documented statement hints, the documented `SOUNDEX`
generated-column composition on a multi-column search index, and custom
dictionary lookup with its staleness hint and `enhance_query` composition.

The full-text raw stream has two focused manifests so aggregation composition
and query-enhancement/phonetic composition can evolve independently:

```sh
/tmp/spanner-query-plan-shape \
  --case full_text_search \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error > /tmp/full-text-search.json
/tmp/planvocab-check \
  --expect tools/spanner-query-plan-shape/testdata/facet_search_expectations.json \
  < /tmp/full-text-search.json
/tmp/planvocab-check \
  --expect tools/spanner-query-plan-shape/testdata/full_text_residual_expectations.json \
  < /tmp/full-text-search.json
```

Inspect n-gram fuzzy search and ordinary pattern-predicate acceleration with a
minimal two-token-column fixture:

```sh
go run ./tools/spanner-query-plan-shape \
  --case ngram_search \
  --optimizer-version-matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

The fuzzy column uses a direct source-column reference for `SCORE_NGRAMS`;
the separate pattern column uses `TOKENIZE_NGRAMS(LOWER(...))`. The matrix
includes forced search-index/base-table pairs for `LIKE`, `STARTS_WITH`,
`ENDS_WITH`, and `REGEXP_CONTAINS`, plus parameter and too-short-literal
eligibility boundaries.

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case ngram_search \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --expect tools/spanner-query-plan-shape/testdata/ngram_search_expectations.json
```

The `vector_search` selector also includes a reciprocal-rank-fusion candidate
that keeps ANN and full-text retrieval below one `Union All`, plus an exact-KNN
control that removes the vector-index access path while preserving the fusion
shell.

Use the combined `search_graph` selector to check the manifest that spans the
full-text GQL traversal, vector GQL traversal, and hybrid RRF shape in one raw
stream:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case search_graph \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --expect tools/spanner-query-plan-shape/testdata/search_graph_expectations.json
```

The `ai_plan` query set pairs `AI.CLASSIFY`, `AI.IF`, and `AI.SCORE` with scalar
controls. Its positive evidence comes from managed Spanner
`AnalyzeQuery(QueryMode=PLAN)`; the CLI's pinned Omni backend returns a stable
`Internal` capability error for the AI candidates while planning the controls.
No retained probe executes external inference. Destination-redacted managed
fixtures make the observed TVF lowering replayable without another service
call, and the focused Omni test preserves the environment divergence.

Inspect JSON search-index probes separately from Full Text Search:

```sh
go run ./tools/spanner-query-plan-shape \
  --case json_search \
  --output compact-tree-metadata \
  --continue-on-error
```

The dedicated JSON schema compares `JSON_CONTAINS`, key existence, logical
combinations, a mixed `SEARCH` plus JSON predicate, a stored projection, and a
non-covering base-table back join. This isolates direct JSON Search Predicate
plans from Full Text Search plans that also require the Search Query Conversion
TVF.

Inspect exact KNN and approximate nearest-neighbor vector-index probes:

```sh
go run ./tools/spanner-query-plan-shape \
  --case vector_search \
  --output compact-tree-metadata \
  --continue-on-error
```

The vector case compares an exact `_BASE_TABLE` KNN query with ANN automatic
index selection, explicit COSINE, DOT_PRODUCT, and EUCLIDEAN index families,
extra-key and stored-column filters, a non-covering back join, a filtered
vector index, and an ANN result passed through GQL `NEXT` into a relationship
traversal. The focused Omni test requires DOT_PRODUCT and EUCLIDEAN queries to
route to their matching VectorIndexRootScan and VectorIndexLeafScan targets in
optimizer versions 1 through 9. The vector index DDL is kept as an explicit raw
statement list because the current memefish parser does not yet accept the
documented extra key columns after the embedding column.

Inspect `DISABLE_INLINE` function hint probes:

```sh
go run ./tools/spanner-query-plan-shape --case function_hint --output nodes
go run ./tools/spanner-query-plan-shape --case function_hint --output yaml
```

The `nodes`, `yaml`, and `json` outputs keep scalar `Function`
nodes visible, which is useful when the compact one-line operator summary hides
expression-level changes.

Inspect common table expression probes:

```sh
go run ./tools/spanner-query-plan-shape --case cte --output compact-tree-metadata
go run ./tools/spanner-query-plan-shape --case cte --optimizer-version-matrix --output compact-tree-metadata
```

The built-in CTE case compares single and repeated references over constant
rows, deterministic functions, `CURRENT_TIMESTAMP()`, and base-table scans. Run
the first command without `--optimizer-version-matrix` to capture the default
optimizer behavior, then the second command to compare pinned versions 1
through 8.

Expand any built-in or custom query set across `OPTIMIZER_VERSION` statement
hints from 1 through 8:

```sh
go run ./tools/spanner-query-plan-shape \
  --case docs \
  --optimizer-version-matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

When a query already has a leading statement hint, this keeps the other hint
assignments and replaces only `OPTIMIZER_VERSION`.

For optimizer behavior probes, prefer starting from unhinted queries. The
`optimizer_unhinted_candidates` case is generated from the `docs` and
`optimizer_gaps` query sets with all `@{...}` hints stripped outside string
literals. Use `--optimizer-version-diff` to analyze versions 1 through 9 and
print only queries whose compact-tree-metadata shape, or planning error shape,
actually changes:

```sh
go run ./tools/spanner-query-plan-shape \
  --case optimizer_unhinted_candidates \
  --optimizer-version-diff
```

This keeps broad exploratory input separate from the smaller set of probes that
are worth preserving as optimizer-version evidence.

Expand a query set across `ALLOW_DISTRIBUTED_MERGE` default, `TRUE`, and
`FALSE`. This can be combined with `--optimizer-version-matrix`:

```sh
go run ./tools/spanner-query-plan-shape \
  --case docs \
  --optimizer-version-matrix \
  --allow-distributed-merge-matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

Inspect statement, table, group, and graph hints from the Spanner GoogleSQL
query syntax Hints section, excluding join hints that have dedicated matrices:

```sh
go run ./tools/spanner-query-plan-shape \
  --case hint_matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

Try documented query hints in statement position against all documentation-
derived query examples:

```sh
go run ./tools/spanner-query-plan-shape \
  --case statement_hint_query_matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

This matrix is intentionally broad. It includes statement hints plus documented
table, group, and join hints that the query grammar allows before `query_expr`.
Some combinations are expected to fail because a hint may require a matching
query shape, such as a join or `GROUP BY`.

Inspect an explicit join hint matrix that is broader than the documentation
examples:

```sh
go run ./tools/spanner-query-plan-shape \
  --case join_matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

Inspect subquery predicates with statement-level join hints:

```sh
go run ./tools/spanner-query-plan-shape \
  --case subquery_join_hint_matrix \
  --output compact-tree-metadata \
  --continue-on-error
```

Verify hint positions that are not covered by the statement, table, join,
group, graph, function, and DML matrices. This includes accepted positions with
no known usable key, accepted subquery positions, and paired positions that
must be rejected. Use `--continue-on-error`: an `Unsupported hint` or later
feature error proves parsing reached hint validation, while `Syntax error` and
the targeted GQL placement errors or `Hints on set operations must appear on
the first operation` are expected for labels under `hint-position/rejected/`.

```sh
go run ./tools/spanner-query-plan-shape \
  --case hint_position_audit \
  --output summary \
  --continue-on-error
```

The environment-gated integration assertion uses the same case table:

```sh
go test -tags=integration ./tools/spanner-query-plan-shape \
  -run TestIntegrationHintPositionAuditOnEmulator

SPANEMUBOOST_ENABLE_OMNI_TESTS=1 \
  SPANALYZER_OMNI_IMAGE=us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  go test -tags=integration,omni ./tools/spanner-query-plan-shape \
  -run TestIntegrationHintPositionAuditOnOmni
```

The Emulator tests above classify syntax with the execution API. The Emulator
does not return plans for `PLAN` or `PROFILE` query modes; use Omni for the
plan-shape checks below.

Verify combinations of multiple hint keys on SQL `WHERE EXISTS` and `WHERE IN`
predicates. The positive planvocab expectations check hash-join orientation,
Apply batch mode, and coexistence with statement hints. Syntax-accepted
set-operation, scalar-subquery, and GQL controls remain in the same raw stream
without claiming that every accepted key affected the selected plan:

```sh
go build -o /tmp/planvocab-check ./plancontract/cmd/planvocab-check
go build -o /tmp/spanner-query-plan-shape ./tools/spanner-query-plan-shape
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case hint_position_combinations \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --expect tools/spanner-query-plan-shape/testdata/hint_position_combination_expectations.json
```

Separate factorized-hint acceptance from plan-visible effect and eligibility
rejection:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case factorized_mode \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/factorized_mode_expectations.json
```

The explicit v4/v5 controls preserve the boundary between accepted hints with
no visible factorization and the Aggregate/Array Unnest signature. The
explicit v8 errors cover join-key-only projections, outer joins, and
non-equality conditions. Use `--optimizer-version-matrix` to reproduce the
v1-v9 acceptance boundary; prefixed matrix labels are a vocabulary gate and do
not match the unprefixed expectation manifest.

Probe the broad GQL and SQL/GQL bridge surface, including accepted clauses
that optimize to a simpler control plan:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case gql_surface \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/gql_surface_expectations.json
```

This case retains nontrivial graph DISTINCT and ALL set-operation arms,
correlated ARRAY/IN/NOT IN, inner/optional/uncorrelated CALL and MATCH controls,
recursive/subpath/path-construction variants, element functions and predicates,
meaningful compound-label OR/AND over a multi-label graph, graph-element SQL
bridging, correlated FOR, GQL-native BERNOULLI/RESERVOIR, horizontal
aggregates, `IS_FIRST` in `RETURN`, direct and edge `FILTER`, quantified
traversal, and `NEXT` placements, and explicit capability errors.
See
[`GQL_SURFACE_OBSERVATIONS_2026-08-11.md`](../../research/spanner-query-plan-shape/GQL_SURFACE_OBSERVATIONS_2026-08-11.md)
for the plan and optimizer-version evidence.

Inspect GQL set-operation column propagation independently from the broad
surface inventory:

```sh
go run ./tools/spanner-query-plan-shape \
  --case gql_set_propagation \
  --output compact-tree-metadata \
  --continue-on-error
```

The six accepted spellings compare `FULL`, `INNER`, and `LEFT` propagation
with their `OUTER` aliases over deliberately mismatched output names. A
seventh default/`STRICT` control must fail before producing a plan. The focused
Omni integration test verifies result columns and rows, `Union Input` slot
counts, typed-NULL synthesis, and alias plan equivalence at the default
optimizer setting and explicit versions 1 through 9. These modifiers are not
advertised by the current Spanner GQL set-operation documentation, so the
probe records runtime evidence rather than a portable syntax guarantee.

Probe graph-specific hint placements and their default plan shapes:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case gql_hint_surface \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/gql_hint_surface_expectations.json
```

The selector covers graph `FACTORIZE_BOTH`, GQL set-operation statement/arm
controls, correlated EXISTS/VALUE hint placement, traversal MERGE/HASH/APPLY,
official and pinned-runtime-extension traversal-hint positions, graph-element
FORCE_INDEX/INDEX_STRATEGY/SCAN_METHOD/GROUPBY_SCAN_OPTIMIZATION/
SEEKABLE_KEY_SIZE composition—including distinct COLUMNAR/NO_COLUMNAR scan
metadata—and graph PUSH boundaries. Its focused Omni
test enforces v4/v5 factorization equality changes, ONE_PASS v3/v4, PUSH
v2/v3, edge BATCH v5/v6, between-path HASH v2/v3, and the v1-v9 plan effects
that a presence-only manifest cannot express. See
[`GQL_HINT_VERSION_OBSERVATIONS_2026-08-11.md`](../../research/spanner-query-plan-shape/GQL_HINT_VERSION_OBSERVATIONS_2026-08-11.md).

Probe generic GoogleSQL grammar surfaces with accepted PLAN shapes or explicit
runtime/transaction boundaries:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case google_sql_surface \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/google_sql_surface_expectations.json
```

This case covers accepted HAVING, aggregate-call DISTINCT/null modifiers,
array lambda transform/filter, SELECT ALL, star
EXCEPT/REPLACE, subquery SELECT AS VALUE, scalar WITH, IN UNNEST, COLLATE, and
read-write FOR UPDATE. It also retains explicit errors for analytic functions
on the pinned Omni runtime, TABLESAMPLE REPEATABLE, grouping
extensions, PIVOT/UNPIVOT, recursive WITH, top-level value tables, subquery
WITH, read-only FOR UPDATE, `SECURE_CONTEXT`, and the lock-hint/FOR UPDATE
conflict.

Probe every pipe operator listed by the Spanner reference:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case pipe_surface \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  | /tmp/planvocab-check \
      --expect tools/spanner-query-plan-shape/testdata/pipe_surface_expectations.json
```

The selector covers `SELECT`, `EXTEND`, `SET`, `DROP`, `RENAME`, `AS`,
`WHERE`, `AGGREGATE`, `JOIN`, `ORDER BY`, `LIMIT`, `UNION`, `INTERSECT`,
`EXCEPT`, and `TABLESAMPLE`. Its focused Omni test repeats all 15 at optimizer
versions 1 through 9.

Probe optimizer version 9 and its managed-service controls:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case optimizer_v9 \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/optimizer_v9_expectations.json
```

This selector pins the v8 Row/v9 Batch DCA boundary and the v9 typed-null
`__row_id`/`restored_*` mechanism, retains a LIMIT control without treating it
as a stable suppression rule, checks an index-union candidate, compares the
v8/v9 `DELETE ... THEN RETURN` shapes, and requires the version 10 rejection.

Map every public rewriter registered by the pinned GoogleSQL frontend to
retained Spanner evidence:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case rewriter_surface \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/rewriter_surface_expectations.json
```

The selector retains 18 PLAN-producing surfaces and 15 exact runtime
capability errors. A separate completeness table records all 33 registered
rewriters, including internal or GoogleSQL-only features that Spanner does not
expose. The focused Omni test repeats all 33 direct probes for optimizer
versions 1 through 9 and verifies plan-visible array lowerings, aggregate
modifier topology, view/control equivalence, and stable error classes. See
[`REWRITER_SURFACE_OBSERVATIONS_2026-08-12.md`](../../research/spanner-query-plan-shape/REWRITER_SURFACE_OBSERVATIONS_2026-08-12.md).

Descriptor-backed GoogleSQL surfaces use a separate selector because Spanner
must receive the serialized `FileDescriptorSet` with `CREATE PROTO BUNDLE`:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case google_sql_proto_surface \
  --proto-descriptors-file testdata/protos/order_descriptors.pb \
  --proto-descriptors-file testdata/protos/complex/complex_descriptors.pb \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/google_sql_proto_surface_expectations.json
```

The selector covers both `NEW` constructor forms, `CAST AS PROTO`,
`REPLACE_FIELDS` with a nested field path, nested `SELECT AS typename`, the
documented rule that `DISTINCT` is applied before proto construction, scalar,
nested, presence, enum, and repeated proto field access, repeated-field
`UNNEST`, and the top-level value-table boundary. The descriptor flag may be
repeated; descriptor files are merged by source file name and conflicting
definitions are rejected before the runtime starts.

The same selector retains six upstream GoogleSQL proto surfaces as explicit
unsupported expectations: `EXTRACT(FIELD/HAS/RAW/ONEOF_CASE)`,
`PROTO_DEFAULT_IF_NULL`, and `FILTER_FIELDS`. On the pinned runtime each error
class is invariant from optimizer v1 through v8. Add
`--optimizer-version-matrix` without the unprefixed expectation manifest to
repeat the 144-submission vocabulary matrix.

Compare expression boundaries between distributed split selection, scan
seeking, residual filtering, and join-key evaluation:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case condition_boundaries \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --expect tools/spanner-query-plan-shape/testdata/condition_boundary_expectations.json
```

The selector contrasts literal and parameterized prefix predicates, partial
`LIKE` and regexp extraction, substring/suffix/transformed-key residuals,
conjunctions and disjunctions, computed Hash/Merge equality keys, non-equality
join residuals, one-side scan pushdown, Apply Join correlation, and a leading
commit-timestamp primary key with `ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN` enabled
or disabled. The enabled case requires `Seek Condition` and `Timestamp
Condition` on the same Scan. Add `--optimizer-version-matrix` without the
unprefixed expectation manifest to repeat the 304-plan matrix. Expression text
and version boundaries are checked by the focused Omni integration test and recorded in
[`CONDITION_BOUNDARY_OBSERVATIONS_2026-08-11.md`](../../research/spanner-query-plan-shape/CONDITION_BOUNDARY_OBSERVATIONS_2026-08-11.md).

Compare the 19 documented Spanner aggregate-function names with the physical
expressions attached to Aggregate `Agg` child links:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case aggregate_functions \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/aggregate_function_expectations.json
```

The selector retains all documented general/statistical functions plus
`DISTINCT`, ordering/limit, `HAVING MAX/MIN`, grouping-only, and unsupported
controls. Add `--optimizer-version-matrix` without the unprefixed expectation
manifest to repeat the 248-submission matrix. Exact `Agg` expression types,
alias plan equality, and finalization outside Aggregate are checked by the
focused Omni integration test and recorded in
[`AGGREGATE_FUNCTION_AGG_TYPE_OBSERVATIONS_2026-08-11.md`](../../research/spanner-query-plan-shape/AGGREGATE_FUNCTION_AGG_TYPE_OBSERVATIONS_2026-08-11.md).

Probe join-family selection for `INTERSECT` and `EXCEPT`, ALL multiplicity
restoration, and aggregate implementations of `UNION DISTINCT` and
`SELECT DISTINCT`:

```sh
set -o pipefail
/tmp/spanner-query-plan-shape \
  --case set_operation_distinct \
  --omni-image us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
  --output json \
  --continue-on-error \
  | /tmp/planvocab-check \
      --allow-query-errors \
      --expect tools/spanner-query-plan-shape/testdata/set_operation_distinct_expectations.json
```

The expected errors are negative controls for `GROUP_METHOD` directly on
DISTINCT and `HASH_JOIN_BUILD_SIDE` on set operations. The positive
expectations cover plan-visible HASH, MERGE, local APPLY, distributed APPLY,
aggregate iterator, and `Generate Relation` link shapes. The same case also
compares `INTERSECT DISTINCT` / `EXCEPT DISTINCT` with `EXISTS` / `NOT EXISTS`
rewrites on non-null keys: the rewrites cover hash build-side selection and
simultaneous join plus group-method controls that the direct set operators do
not expose. The retained three-input controls also show that separately hinted
predicates can select Hash and APPLY semi/anti comparisons in one plan, while
a hint on the second same-level set operator is rejected. Separate NULL-bearing
result probes use native `IS NOT DISTINCT FROM` comparisons. See
[`SET_OPERATION_DISTINCT_HINTS_2026-08-04.md`](../../research/spanner-query-plan-shape/SET_OPERATION_DISTINCT_HINTS_2026-08-04.md)
for the pinned-runtime evidence and rewrite caveats.

Run the populated multi-column and three-input result-equivalence test on the
pinned Omni image with:

```sh
SPANEMUBOOST_ENABLE_OMNI_TESTS=1 \
SPANALYZER_OMNI_IMAGE=us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta \
go test -tags='integration omni' ./tools/spanner-query-plan-shape \
  -run '^TestIntegrationSetOperationRewriteEquivalenceOnOmni$' -count=1
```

See
[`research/spanner-query-plan-shape/QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md`](../../research/spanner-query-plan-shape/QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md)
for the checked Spanner documentation examples, and
[`research/spanner-query-plan-shape/OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md`](../../research/spanner-query-plan-shape/OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md)
for optimizer-version compact-tree-metadata observations. The prioritized DDL,
data, query, and hint combinations for still-unobserved shapes are recorded in
[`research/spanner-query-plan-shape/UNOBSERVED_PLAN_PROBE_MATRIX_2026-07-10.md`](../../research/spanner-query-plan-shape/UNOBSERVED_PLAN_PROBE_MATRIX_2026-07-10.md).

Analyze custom DDL and SQL:

```sh
go run ./tools/spanner-query-plan-shape \
  --ddl testdata/querygen-schema.sql \
  --sql 'SELECT SingerId FROM Singers ORDER BY SingerId'
```

`--ddl`, `--sql`, and `--sql-file` may be repeated. When any custom SQL is
provided, the built-in `--case` examples are not run. Use
`--output compact-tree-metadata` for the canonical one-line child-link tree with
metadata and child-link annotations, `--output compact-tree` for the same tree
without metadata, `--output summary` for node names,
`--output reference` for `spannerplan/plantree/reference` rendering,
`--output yaml` or `--output json` for a query result array/list whose entries
wrap the raw query plan protobuf with the query label and SQL, or the default
`--output nodes` for node metadata. Use `--compact-tree-indexes` to include
PlanNode indexes in compact tree outputs, `--optimizer-version-matrix` to
repeat the selected queries with statement-level optimizer version hints,
`--optimizer-version-diff` to print only queries whose v1-v9
compact-tree-metadata/error shape changes, and
`--allow-distributed-merge-matrix` to repeat them with
`ALLOW_DISTRIBUTED_MERGE` default, `TRUE`, and `FALSE`.
The raw YAML output is converted from the same JSON envelope that contains the
`protojson` query plan payload.

Legacy flattened vocabulary modes remain available as `--output compact-dfs`
and `--output compact-dfs-metadata`; `compact` and `compact-metadata` remain
accepted as aliases for those modes.

## Requirements

The tool starts Spanner Omni through `spanemuboost`, so Docker must be running
and the Spanner Omni container image must be available to the local Docker
environment.
