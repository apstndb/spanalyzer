---
type: Research Note
title: Identifying the backing index of a Spanner foreign key
description: A two-dialect test of constraint names in FORCE_INDEX, a prospective physical-name prediction using Omni, and an Emulator catalog correction for reordered columns.
tags: [spanner, spanner-omni, emulator, foreign-keys, indexes, information-schema, query-plans]
status: draft
sources:
  - id: foreign-keys
    resource: https://docs.cloud.google.com/spanner/docs/foreign-keys/overview
    title: Backing-index ownership, sharing and use in FORCE_INDEX
  - id: google-hints
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax
    title: GoogleSQL table hints
  - id: pg-hints
    resource: https://docs.cloud.google.com/spanner/docs/reference/postgresql/query-syntax
    title: PostgreSQL table hints
  - id: emulator-catalog
    resource: https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/backend/query/information_schema_catalog.cc#L2505
    title: Emulator 1.5.59 KEY_COLUMN_USAGE producer
  - id: results
    resource: evidence/fk-index-resolution-20261118/receipt.json
    title: Dated run outcomes, physical indexes, plan scans and the pre-managed forecast
---

# Identifying the backing index of a Spanner foreign key

**The parent key has a documented catalog identity. The child index can also be observed on managed Spanner and Omni.** In both SQL dialects, using the FK constraint name in a table-level `FORCE_INDEX` on the child produced a plan naming the underlying index. The FK guide expressly permits using backing indexes in this directive and directs readers to their names in `INFORMATION_SCHEMA`. The GoogleSQL and PostgreSQL hint references describe an index name or `_BASE_TABLE`. None of these references specifies an FK constraint name as an alternative spelling. Treat constraint-name lookup as optional runtime evidence. [Foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#foreign-key-characteristics) · [GoogleSQL table hints](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#table_hints) · [PostgreSQL table hints](https://docs.cloud.google.com/spanner/docs/reference/postgresql/query-syntax#table-hints)

## Inspect an existing database

1. Resolve the parent through `INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS.UNIQUE_CONSTRAINT_{CATALOG,SCHEMA,NAME}`. In this fixture, non-primary references identified the parent's unique managed index. The primary-key reference identified constraint `PK_p`, with pseudo-index `PRIMARY_KEY`. PostgreSQL's `pg_constraint.conindid` was NULL for all five FKs on each engine. [Catalog queries](https://docs.cloud.google.com/spanner/docs/foreign-keys/how-to) · [PostgreSQL system catalog](https://docs.cloud.google.com/spanner/docs/reference/postgresql/pg-system-catalog-tables)
2. To inspect the child, send one of the following reads with `ExecuteSqlRequest.query_mode = PLAN`. Supply a predicate that excludes NULL on every nullable index key. Inspect the node with `metadata.scan_type = IndexScan` and match its `metadata.scan_target` to a managed index on the child table and schema. A successful request alone does not establish the physical target.

```sql
-- GoogleSQL
SELECT code FROM c@{FORCE_INDEX=fk_p} WHERE code = 'k';
-- PostgreSQL
SELECT code FROM c /*@ FORCE_INDEX=fk_p */ WHERE code = 'k';
```

| Result in both SQL dialects | Managed Spanner | Omni 2026.r4-lts.2 and 2026.r5.1 | Emulator 1.5.59 |
| --- | --- | --- | --- |
| Child FKs `fk_p` and `fk_q` on `c(code)` | Both selected `IDX_c_code_N_E985B7CD5F04A313` | Same full name for both | Both FK names returned `InvalidArgument` |
| Drop `fk_p`, then drop `fk_q` | Only the surviving name resolved after the first drop. The last drop removed the child backing index. | Same | Catalog showed the same index lifetime. The FK names were rejected throughout. |
| `fk_rev` and `fk_fwd` on the same child columns `cc(x,y)` | Distinct targets keyed `(y,x)` and `(x,y)`, respectively | Same | Both layouts existed. Separate drops identified each owner. |

The ordinary index `ix_c_code` survived both drops. On managed Spanner and Omni, the parent-table hint, an unknown FK name and the FK name on a child covered by its nonnullable primary key returned `InvalidArgument`. For `c(code)`, forcing `fk_p` with `WHERE id = 1` also failed. That predicate can select rows with NULL `code`, which the NULL-filtered index omits. The successful `WHERE code = 'k'` probe excludes those rows. A failed constraint-name hint cannot decide whether an index exists. In GoogleSQL the informational FK created only a parent index. PostgreSQL rejected its `NOT ENFORCED` declaration.

Emulator documents a separate rule for **physical names**. Both dialects accepted the managed-generated name above and rejected the Emulator's locally generated `IDX_c_code_N_D99AE4F461677C4A` as a hint. Omni LTS accepted the FK name and the managed-generated name and rejected the Emulator-generated name. Both engines accepted the user index. These were normal SQL reads. [Pinned Emulator README](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/README.md#L237)

## Predict a layout or name before applying DDL

Use the ordered FK declaration to pair each child column with its parent column. In the tested layouts the parent index starts with the participating leading parent primary-key columns, then takes the remaining referenced columns in table declaration order. Apply that permutation to the child. For an existing schema use the observed parent key order. Match the resulting child key, managed flag, uniqueness and null filtering against `INDEXES` and `INDEX_COLUMNS`, scoped by schema and table. Multiple FKs can share one physical index. An ordinary user index with matching keys does not replace it.

For example, `FOREIGN KEY (x,y) REFERENCES pa(b,a)` with parent index `pa(a,b)` yields child key `cc(y,x)`. A second FK on `(x,y)` referencing `pb(a,b)` yields `cc(x,y)`. Matching unordered column sets loses the distinction. The [offline predictor](evidence/fk-index-resolution-20261118/predict.py) reproduced all seven layouts in the primary fixture. It covers ascending scalar keys. A three-column permutation involving the child's PK can defeat a prefix-only coverage rule. [Earlier coverage and counterexamples](fk-backing-index-predictability-20261112.md)

For **complete managed names before deployment**, replay the identical DDL in a disposable, version-pinned Omni database. The seven full names and five plan targets observed on Omni LTS were recorded at 14:48:57 UTC, before either managed fixture was created. Managed GoogleSQL and PostgreSQL both matched all seven names and all five targets. Omni r5.1 matched too. The Emulator's seeded FarmHash recipe reproduced all seven Emulator names and zero of the seven managed names. The reviewed sources leave the managed suffix algorithm unspecified. A different engine build or schema needs its own comparison. [Forecast and results](evidence/fk-index-resolution-20261118/receipt.json)

## A catalog trap on Emulator

For `fk_rev`, joining the child's `KEY_COLUMN_USAGE.POSITION_IN_UNIQUE_CONSTRAINT` to the parent index's `ORDINAL_POSITION` produced `x → b, y → a` on managed and both Omni releases. Emulator returned `x → a, y → b` in both dialects. With two FKs sharing the same parent unique index, Emulator emitted each parent key row twice. A two-column FK then produced four joined rows. The same control on Omni LTS produced two. Removing duplicate rows cannot repair an incorrect pairing.

The pinned Emulator source writes the child's ordinal into both position fields and emits the referenced index's keys inside the loop over foreign keys. `GetDatabaseDdl` preserved the declared pairs in both dialects. For a CLI, use those declared pairs when the Emulator catalog disagrees. Show child indexes matched by layout as candidates. A plan identifies a physical target only when the target can be matched to the child's catalog row. Incomplete visibility under fine-grained access control leaves the association unresolved. [Emulator catalog producer](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/backend/query/information_schema_catalog.cc#L2505) · [Information schema](https://docs.cloud.google.com/spanner/docs/information-schema)

## Reproduction

The [case generator](evidence/fk-index-resolution-20261118/cases.py) prints exact SQL, request modes and dependencies. Use `python3 cases.py google` or `python3 cases.py pg` in the evidence directory. `--controls` selects the independent composite/drop fixture. `--physical-hints` compares three spellings and a user index. Apply DDL in order to an empty database. Use ordinary reads for `query`, PLAN for `plan`, and the admin API's `GetDatabaseDdl` for `ddl-export`.

Run `python3 -B verify.py` in the evidence directory to compare the exact input hashes, the pre-recorded names, the physical catalogs, and the run outcomes offline.

The primary managed run on November 18 UTC (November 19 JST) planned 36 GoogleSQL and 37 PostgreSQL steps. It obtained 69 RPC outcomes, including expected rejections. Four PostgreSQL reads depended on the unsupported informational FK. Both newly created databases were deleted, and both subsequent lookups returned `NotFound`. The shared-parent duplicate-row and primary-key permutation controls were additional local runs on Emulator and Omni LTS. [Compact observations](evidence/fk-index-resolution-20261118/receipt.json)
