---
type: Research Note
title: GoogleSQL frontend releases and managed Spanner behavior
description: Seven upstream release comparisons from July to November 2026, paired with 71 read-only managed Spanner probes and controls.
tags: [managed-spanner, googlesql, runtime-verification, json, gql]
status: draft
sources:
  - id: managed
    resource: evidence/managed-recheck-20261125/googlesql-release-probes.json
    title: SQL, query options, result types, rows and diagnostics for all 71 probes
  - id: upstream
    resource: https://github.com/google/googlesql/releases/tag/2026.11.3
    title: Latest GoogleSQL frontend release at the time of observation
  - id: conversion
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/conversion_functions
    title: Spanner GoogleSQL conversion rules for arrays, structs and JSON
  - id: emulator
    resource: https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/MODULE.bazel#L234-L244
    title: GoogleSQL source and patch used by Emulator 1.5.59
---

# GoogleSQL frontend releases and managed Spanner behavior

On November 25, 2026, a managed GoogleSQL database accepted conversion of individual struct fields to JSON and rejected conversion of an integer array to an array of JSON values. These results agree with Spanner's published container conversion rules. Upstream `2026.11.3` provides a useful comparison because its analyzer tests cover both operations with the JSON cast feature enabled and disabled. The service's frontend revision and feature flags are unknown. [Spanner conversion rules](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/conversion_functions) · [Upstream array tests](https://github.com/google/googlesql/blob/762ca7a16f2b555c85b6f9250c713a82bd1052d6/googlesql/analyzer/testdata/cast_as_json.test#L440-L488) · [Upstream struct tests](https://github.com/google/googlesql/blob/762ca7a16f2b555c85b6f9250c713a82bd1052d6/googlesql/analyzer/testdata/cast_as_json.test#L867-L889)

## Managed results

The 71 queries ran from 22:56 to 23:02 JST on one existing regional Enterprise trial database. Each ran in a single-use read-only transaction. Thirty succeeded, 33 returned `InvalidArgument`, six returned `Unimplemented` and two returned `OutOfRange`. The set includes positive controls, intentional errors and a graph inventory. [Inputs, results and types](evidence/managed-recheck-20261125/googlesql-release-probes.json)

| Candidate | Result on managed GoogleSQL |
| --- | --- |
| Nested JSON casts | Casting `STRUCT(7 AS original_name)` to `STRUCT<new_name JSON>` returned JSON number `7` in the renamed field. Integer and string fields also worked, including values supplied by a subquery. Casting `ARRAY<INT64>` to `ARRAY<JSON>` failed for a literal and a column expression. Casting the entire array to one JSON value returned `[1,2]`. Applying `TO_JSON` to each element returned an actual `ARRAY<JSON>`. |
| Secure context | With a synthetic `user_id` in the request, `SECURE_CONTEXT('user_id')` returned the supplied string. Omitting the key raised a missing-parameter error. The routine suite had already exercised this function. |
| SQL cast shorthand and quantified comparisons | Both postfix cast operators, `::` and `?::`, and the tested `= ANY` and `!= ALL` forms were rejected. `CAST` and `SAFE_CAST` returned 123 and SQL NULL. The equivalent `IN` and `NOT IN` controls returned TRUE, NULL and TRUE. |
| Aggregate filters and window frames | `SUM(1 / x WHERE x != 0)` rejected the in-call modifier. Moving the predicate to the query returned 1.5, and removing it raised division by zero. A computed frame bound required a literal or parameter. Literal and type-matched parameter controls reached an unimplemented analytic aggregate, leaving the upstream RANGE arithmetic unmeasured. |
| JSON presence and wildcard paths | `JSON_EXISTS` was unavailable, including for a plain path to a JSON null. `JSON_QUERY` preserved that null. `JSON_QUERY_ARRAY` returned the two elements at `$.a` and rejected the wildcard in `$.a[*]` with `OutOfRange`. |
| Additional functions and types | `ARRAY_REDUCE`, `JAROWINKLER_SIMILARITY`, the tested bit-cast function names and the `VARIANT` type were unavailable. `NULLIF` rejected a lambda argument and accepted the ordinary two-value call. `EDIT_DISTANCE` reached an unimplemented built-in. |
| Name resolution and table expressions | A select-list alias was invisible to a later expression in the same list, including the grouped positive control. A subquery made the alias visible. Top-level `TABLE` and a string literal table reference were rejected. Ordinary table references worked. |
| Column lists | Standalone `**COLUMNS` reported a disabled feature. In a predicate it produced an expression-resolution error. Both the explicit and unpacked `UNION ALL BY NAME ON` lists rejected the surrounding set-operation syntax. Positional `UNION ALL` succeeded. |

To replay the evening cases, extract `[.cases[].input]` with jq and pass the resulting JSON array to the managed producer described in the [catalog note](managed-omni-emulator-catalogs-20261125.md#reproduction). Returned field types distinguish a single JSON array from an SQL array of JSON values.

## What changed upstream

The Emulator 1.5.59 source pins the July tag and applies a Spanner-specific patch. Separately, spanalyzer's `go-googlesql v0.4.0` build recipe points to `googlesql-wasm v0.3.1`, whose upstream gitlink matches `2026.01.1`. The following seven comparisons use adjacent published tags. The path counts come from `git diff --no-renames --name-only OLD NEW` on complete Git trees. July to September spans eight commits. Each later pair is a single source export. [Emulator source pin](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/MODULE.bazel#L234-L244) · [Go wrapper build](https://github.com/goccy/go-googlesql/blob/2b3726f947f7f809afb1daf534ff9ca06fd4da5c/Makefile#L1-L9) · [WASM source pin](https://github.com/goccy/googlesql-wasm/tree/f64227ee7f6ded155495269edb0344c77c78e12e)

| Release pair | Changed paths | Selected upstream changes relevant to Spanner |
| --- | ---: | --- |
| [2026.7.2 → 2026.9.1](https://github.com/google/googlesql/compare/2026.7.2...2026.9.1) | 593 | Postfix cast operators, bit casts, lambda overload of NULLIF and query-constant window-frame offsets. |
| [2026.9.1 → 2026.9.2](https://github.com/google/googlesql/compare/2026.9.1...2026.9.2) | 178 | Top-level TABLE and Jaro-Winkler leave development status. GQL label resolution changes. |
| [2026.9.2 → 2026.10.1](https://github.com/google/googlesql/compare/2026.9.2...2026.10.1) | 285 | JSON existence functions gain reference evaluation. JSONPath wildcards gain a feature flag. General quantified comparisons gain documentation. Index stored-column changes gain grammar and AST support. |
| [2026.10.1 → 2026.10.3](https://github.com/google/googlesql/compare/2026.10.1...2026.10.3) | 231 | SQL-standard JSONPath wildcards reach reference evaluation. GQL endpoint predicates on NULL operands return NULL in reference tests. Graph-element JSON casts and stored-column index resolution gain coverage. |
| [2026.10.3 → 2026.11.1](https://github.com/google/googlesql/compare/2026.10.3...2026.11.1) | 337 | SECURE_CONTEXT leaves development status. NUMERIC frame bounds and GQL analytic WITH followed by ANY k MATCH gain coverage. |
| [2026.11.1 → 2026.11.2](https://github.com/google/googlesql/compare/2026.11.1...2026.11.2) | 225 | Filtered aggregates evaluate arguments after filtering in reference cases. GQL checks duplicate WITH aliases earlier. Experimental ARRAY_REDUCE and catalog-backed string literal table references appear. |
| [2026.11.2 → 2026.11.3](https://github.com/google/googlesql/compare/2026.11.2...2026.11.3) | 39 | Experimental VARIANT casts expand. New nested JSON cast tests exercise an existing gate. Diagnostics change for misplaced column unpacking, and system-time resolution checks its feature gate. |

## Remaining candidates

The managed test database exposed zero property graphs. The following tests require a graph with a node, a directed edge and scalar IDs. Past measurements and the upstream release dates establish independent reference points.

- **NULL graph endpoints.** Force an unmatched OPTIONAL MATCH, verify that its edge is NULL, then apply `IS SOURCE OF` and `IS DESTINATION OF`. Compare with a matched edge. Upstream changed from an error to NULL, and Spanner documents three-valued results. [Reference cases](https://github.com/google/googlesql/blob/a09b0bd95dd55dba9478eb8f20a3971bc9e30b70/googlesql/compliance/testdata/graph_table.test#L1175-L1218) · [Spanner graph predicates](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/operators#is_source_predicate)
- **Graph JSON casts.** Reuse the October fixture to compare `CAST` and `TO_JSON` on the same node, edge and path. Managed returned `Unimplemented` for the graph casts in October. November's scalar and nested casts measured different input types. [October observations](managed-json-comparison-and-change-streams-20261010.md#cast-as-json) · [Spanner casts](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/conversion_functions#cast_as_json)
- **GQL name resolution and path selection.** Compare a duplicate intermediate `WITH` alias with a distinct alias while returning an unrelated constant. Separately test analytic `WITH` and `MATCH ANY 2` on their own before combining them. [November 11.2 source changes](https://github.com/google/googlesql/compare/2026.11.1...2026.11.2)
- **Index alteration.** Spanner documents `ALTER INDEX … ADD/DROP STORED COLUMN`. Column-level `IF [NOT] EXISTS` in the upstream grammar is an additional DDL candidate for a disposable index. [Upstream resolver](https://github.com/google/googlesql/blob/a09b0bd95dd55dba9478eb8f20a3971bc9e30b70/googlesql/analyzer/resolver_alter_stmt.cc#L915) · [Spanner secondary-index DDL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language)
