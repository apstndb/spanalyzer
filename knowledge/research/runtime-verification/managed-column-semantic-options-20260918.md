---
type: Research Note
title: "Managed Spanner ordinary column semantic options, 2026-09-18"
description: "description and synonyms are rejected as unknown ordinary column options in CREATE TABLE and ALTER COLUMN on the managed target."
tags: [spanner, managed, column-options, property-graph, ddl]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: ddl-reference
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language
    title: "Spanner DDL reference, retrieved 2026-09-18"
  - id: runtime
    resource: evidence/column-semantic-options-20260918/managed.jsonl
    title: "Managed column option probes and cleanup"
---

# Ordinary column semantic options

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

On 2026-09-18, the managed GoogleSQL target rejected `description` and
`synonyms` as options on an ordinary STRING column. Property graph metadata
option support must not be generalized to ordinary table columns.

| DDL context | description only | synonyms only | both |
| --- | --- | --- | --- |
| CREATE TABLE column OPTIONS | InvalidArgument: unknown option | InvalidArgument: unknown option | Both options unknown |
| ALTER TABLE ALTER COLUMN SET OPTIONS | InvalidArgument: unknown option | InvalidArgument: unknown option | Both options unknown |

Representative statements (actual unique fixture names are in the capture):

```sql
CREATE TABLE T (
  Id INT64 NOT NULL,
  Name STRING(MAX) OPTIONS (description='Ordinary column')
) PRIMARY KEY(Id);

ALTER TABLE T ALTER COLUMN Name
SET OPTIONS (synonyms=['display name','full name']);
```

The service returned `Option: description is unknown.` and
`Option: synonyms is unknown.` This is service-side rejection of unsupported
option keys, not a local memefish failure or proof that generic OPTIONS syntax
cannot be parsed. The probe calls UpdateDatabaseDdl directly.

A control table with `Updated TIMESTAMP OPTIONS(allow_commit_timestamp=true)`
was successfully created. After each rejected ALTER, COLUMN_OPTIONS and
GetDatabaseDdl retained the control option and contained neither semantic
option. The probe deletes its own table and checks for remaining fixture names.

The retrieved CREATE TABLE reference lists allow_commit_timestamp and
locality_group under column_options_def, consistent with this observation.
Table-name SYNONYM syntax is a separate feature from the synonyms option here.

## Evidence and scope

- [Raw service responses](evidence/column-semantic-options-20260918/managed.jsonl)
- [Exact probe source](evidence/column-semantic-options-20260918/probe.go.txt)
- [Build identity](evidence/column-semantic-options-20260918/build.txt)

The producer uses the existing user-level managed verification profile, rejects
an emulator override, and redacts destination identifiers in errors. Source and
binary hashes are retained in the receipt. These are results on one managed
target, not a fleet-wide availability claim. Emulator, Omni, PostgreSQL, other
column types, ALTER ADD COLUMN and property graph behavior were not tested in
this probe. No performance or natural-language query claim is made.
