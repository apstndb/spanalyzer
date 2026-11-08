---
type: Research Note
title: "Managed quantified comparison probes, 2026-09-21"
description: "ANY/SOME/ALL comparisons and LIKE quantifiers reach explicit unsupported errors in managed GoogleSQL; memefish rejects their syntax."
tags: [spanner, managed, quantified-comparison, memefish]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: operators
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/operators
    title: "Spanner operators, retrieved 2026-09-21"
  - id: lexical
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/lexical
    title: "Spanner reserved keywords, retrieved 2026-09-21"
  - id: bigquery-operators
    resource: https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/operators#like_operator_quantified
    title: "BigQuery quantified LIKE, a different product contract"
  - id: runtime
    resource: evidence/quantified-20260921/results.jsonl
    title: "Twelve managed SELECT probes and strict memefish ParseQuery results"
  - id: previous
    resource: ../../../research/spanner-query-plan-shape/HINT_POSITION_AUDIT_2026-08-04.md
    title: "Earlier Emulator and Omni hint-position audit"
---

# Quantified comparison status

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

On 2026-09-21 JST, twelve constant-only queries were submitted directly through
the Spanner Go client to the existing managed GoogleSQL verification target.
No schema or data mutations were performed. All reached explicit unsupported
feature errors (InvalidArgument), rather than transport failures or local
parser errors.

| Form (Q = ANY, SOME, ALL) | Managed result | memefish v0.8.1 |
| --- | --- | --- |
| SELECT 1 = Q (SELECT 1) | Quantified comparisons (= Q) are not supported | Unexpected token Q |
| SELECT 1 = Q (1,2) | Same unsupported feature error | Unexpected token Q |
| SELECT 1 = Q UNNEST([1,2]) | Same unsupported feature error | Unexpected token Q |
| SELECT 'abc' LIKE Q (SELECT 'a%') | LIKE Q is not supported | Unexpected token Q |

Each form was tested separately with all three quantifiers. These results show
recognized constructs with unavailable execution, not verified truth tables or
full operand validation. Other comparison operators, NULL/empty-set behavior,
correlation, hints and PostgreSQL were not tested in this run.

The earlier 2026-08-04 hint-position audit recorded corresponding unsupported
feature behavior on Emulator 1.5.55 and Omni 2026.r1-beta, with managed explicitly
unverified. This run closes that managed observation gap for the cases above.

GitHub main was refreshed to `1691dd67c4c71ca545be852b7a66dc08767082a2`, matching
the inspected local memefish checkout; it has no quantified comparison parser
support. Strict ParseQuery probes used the repository dependency v0.8.1.
Generic GoogleSQL frontend support in the earlier audit is not memefish support.
No parser implementation or GitHub issue/comment was created in this check.

[Producer](evidence/quantified-20260921/probe.go.txt) and
[build identity](evidence/quantified-20260921/build.txt) accompany the raw results.
Destination identifiers are omitted or redacted. Results apply to this managed
target and date; no fleet-wide availability claim is made.

## Official documentation check (2026-09-21)

Full Spanner GoogleSQL operators, query-syntax and lexical references were
retrieved with dkcli. The operators reference documents ordinary binary
comparisons, IN, and `expression [NOT] LIKE pattern`; it contains no ANY/SOME/ALL
quantified-comparison or quantified-LIKE section. The checked query-syntax
reference likewise supplies no support claim for those expressions. This is
absence of documentation in these references, not an explicit documentation
statement that the feature is unsupported. The direct managed errors above
provide the latter evidence for the tested cases.

The LIKE section instead demonstrates multiple-pattern matching with
`ARRAY_INCLUDES(patterns, pattern -> (value LIKE pattern))`. This example is
not a claim that every quantified SQL expression has identical NULL/empty-set
semantics to an array predicate.

ANY, SOME and ALL appear in Spanner's reserved-keyword list. Reserved words
are lexical restrictions, not proof that quantified operators are available.
BigQuery's separate GoogleSQL operators reference explicitly documents
`LIKE ANY/SOME/ALL` with a value list or UNNEST RHS. That BigQuery contract
must not be applied to Spanner, or used as evidence for general `= ANY` support.

The documentation check and runtime results are consistent: no documented
Spanner support was found, and the service explicitly rejects execution.
