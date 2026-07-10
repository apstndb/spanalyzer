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

Inspect the built-in Push Broadcast Hash Join and Hash Join examples:

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
creates only the table and search indexes needed for Full Text Search, then
checks `SEARCH`, `SEARCH_SUBSTRING`, multi-column search, mixed text and
non-text predicates, `SNIPPET`, `SCORE`, `TOKENLIST_CONCAT`, partitioned
ordered search indexes, and numeric array search-index predicates.

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
index selection, extra-key and stored-column filters, a non-covering back join,
and a filtered vector index. The vector index DDL is kept as an explicit raw
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
literals. Use `--optimizer-version-diff` to analyze versions 1 through 8 and
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
`--optimizer-version-diff` to print only queries whose v1-v8
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
