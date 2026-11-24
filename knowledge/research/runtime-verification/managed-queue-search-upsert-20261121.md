---
type: Research Note
title: Managed queue privileges, indexed NaN predicates and ordinary-table upserts
description: Two-dialect results for queue execution before and after bulk grants, floating-point search against base scans, upserts on new and existing keys, and the November 21 managed compatibility replay.
tags: [managed-spanner, queues, fine-grained-access-control, search-indexes, floating-point, upserts, postgresql, runtime-verification]
status: draft
sources:
  - id: measurements
    resource: evidence/managed-recheck-20261121/receipt.json
    title: Fixture SQL, ordered row IDs, privilege outcomes, plans, commit statistics and cleanup
  - id: queue-grants
    resource: https://docs.cloud.google.com/spanner/docs/fgac-queues
    title: Queue TVF names and fine-grained privileges
  - id: number-search
    resource: https://docs.cloud.google.com/spanner/docs/full-text-search/numeric-indexes
    title: Numeric tokens and floating-point negation
  - id: pg-dml
    resource: https://docs.cloud.google.com/spanner/docs/reference/postgresql/dml-syntax
    title: PostgreSQL conflict-clause restrictions
  - id: google-dml
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax
    title: GoogleSQL conflict targets and update expressions
  - id: storage
    resource: https://docs.cloud.google.com/spanner/docs/tiered-storage
    title: Primary-key existence checks for cold data
---

# Managed queue privileges, indexed NaN predicates and ordinary-table upserts

- **Queue privileges.** A bulk `EXECUTE` grant covered queues in existence when the grant ran. Reading queue rows required a separate `SELECT` grant.
- **NaN and search.** All 108 GoogleSQL and 36 PostgreSQL comparisons matched their own dialect's base scan. For NaN, `x != 42` was true in both dialects. `NOT (x > 42)` was true in GoogleSQL and false in PostgreSQL.
- **Ordinary-table upserts.** Omitting `spare` from `INSERT` supplied its default on insertion and preserved its stored value on conflict (three mutations). Sending all columns and assigning only `val` on conflict also preserved `spare` (two mutations). Both dialects agreed on this fixture.

We ran the checks on November 21, 2026, in one fresh managed Spanner database per dialect. Both database lookups returned `NotFound` after deletion. [Fixture SQL, ordered row IDs, query plans and commit statistics](evidence/managed-recheck-20261121/receipt.json)

## Queue privileges

The table shows whether a grantee could call the two queue table-valued functions (TVFs), `RECEIVE` and `RENEWLEASE`. The second queue was created after the initial grant.

| Bulk `EXECUTE` target | Queue present at grant | Queue created afterward | Follow-up |
| --- | --- | --- | --- |
| GoogleSQL `ALL TABLE FUNCTIONS IN SCHEMA DEFAULT` | Both allowed | Both denied | Repeating the grant allowed both on the later queue. Revoking it denied both. |
| GoogleSQL `ALL FUNCTIONS IN SCHEMA DEFAULT` | Both denied | Both denied | Both remained denied on the original queue after revocation. |
| PostgreSQL `ALL FUNCTIONS IN SCHEMA public` | Both allowed | Both denied | Repeating the grant allowed both on the later queue. Revoking it denied both. |
| PostgreSQL `ALL ROUTINES IN SCHEMA public` | Both allowed | Both denied | Revocation denied both on the original queue. |

A role with only queue `SELECT` could read rows and received `PermissionDenied` on both TVFs. An execute-only role could call its granted TVFs and received `PermissionDenied` on direct reads. Granting receive or renewal individually enabled just that operation. Revoking another role's bulk privileges preserved those individual grants.

- **Individual TVF names.** In GoogleSQL, `GRANT` and `REVOKE` required the uppercase TVF prefix and the queue's declared case. Either mismatch returned `NotFound`. After an exact grant, a lowercase SQL call succeeded. In PostgreSQL, the individual grant used `spanner.receive_ordersqueue`. An individual grant with `public.` returned `InvalidArgument`, and a bulk grant or revocation on schema `spanner` returned `NotFound`. [Queue privilege guide](https://docs.cloud.google.com/spanner/docs/fgac-queues#caveats-and-considerations)
- **GoogleSQL scalar functions.** Creating a scalar UDF in the default schema returned `Unimplemented`. In a named schema, a role with schema usage obtained the scalar result 42 after that schema's `ALL FUNCTIONS` grant and lost execution after revocation.
- **Parser coverage.** Managed GoogleSQL accepted grants and revocations for both bulk forms. The same-day strict parser check rejected both at `ALL` in memefish v0.8.1 and upstream `da2ddfd`. Both revisions accepted the individual queue TVF grant and revocation. [Parser source at the tested revision](https://github.com/cloudspannerecosystem/memefish/blob/da2ddfdd6277527f9dc2a54b87221579a9b18abe/parser.go#L5433-L5464)

To reproduce, seed `OrdersQueue` (PostgreSQL `ordersqueue`), grant privileges, then create and seed a second queue. Set `spanner.ClientConfig.DatabaseRole` to the grantee. Take the first `RECEIVE` row in a single-use strong read. For `RENEWLEASE`, hold an independent admin receive stream open and pass its lease token to the grantee in a bound string array. Successful renewals returned a replacement token. Eleven initial calls reached their 12-second client deadline. All eleven yielded authorization outcomes when repeated with eight fresh messages before each stream. [Queue usage](https://docs.cloud.google.com/spanner/docs/queues/queues-using)

## NaN and numeric search indexes

All 144 comparisons returned the same ordered IDs as the corresponding base scan. The comparisons included forced search indexes and automatic plans. GoogleSQL exercised float-token indexes directly. PostgreSQL used an independent integer-token index to test filtering on a float.

Each float width used seven rows. The integer filter `gateval = 1` kept IDs 1 through 5, whose respective `x` values were NaN, 41, 42, 43 and NULL. It excluded ID 6 (NaN) and ID 7 (41). The table shows which of the five rows also passed each float predicate.

| Predicate after `gateval = 1` | GoogleSQL IDs | PostgreSQL IDs |
| --- | --- | --- |
| `x != 42` | 1, 2, 4 | 1, 2, 4 |
| `NOT (x > 42)` | 1, 2, 3 | 2, 3 |
| `x < 42 OR x > 42` | 2, 4 | 1, 2, 4 |
| `x <= 42` | 2, 3 | 2, 3 |

**GoogleSQL.** The 132 profiles included 24 base controls and 108 alternatives across both float widths, equality and all-mode tokens, and float-only and composite indexes. For `gateval = 1 AND x != 42`, the forced plan searched the integer token and kept the float comparison as a `Residual Condition`. With only a float-token index, standalone `x != 42` scanned the index with a residual and no search predicate. Equality against 42 used a float search token. `DEBUG_TOKENLIST` showed no value token for NaN, an equality token for 42 and NULL for NULL. [Numeric index guide](https://docs.cloud.google.com/spanner/docs/full-text-search/numeric-indexes)

**PostgreSQL.** The 60 profiles comprised 24 base controls and 36 alternatives. Direct `spanner.tokenize_number` calls returned `Unimplemented` for both `real` and `double precision` in default, equality and all modes. All three `bigint` controls succeeded. The NaN differences shown above were also present on base scans. [PostgreSQL function reference](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions)

## Partial and full ordinary-table upserts

Both dialects produced these results on `plain_up`, a three-column table with primary key `id`, integer `val`, and `spare` defaulting to `'default-tag'`. Each statement sent `val = 11`. Each existing-key case used a separate row seeded with `val = 7` and `spare = 'kept'`.

| Columns in `INSERT` | Columns assigned from `excluded` in `SET` | New key (`spare` / mutations) | Existing key (`spare` / mutations) |
| --- | --- | --- | --- |
| `id, val` | `id, val` | `'default-tag'` / 3 | `'kept'` / 3 |
| `id, val, spare` | `id, val, spare` | `'sent'` / 3 | `'sent'` / 3 |
| `id, val, spare` | `val` | `'sent'` / 3 | `'kept'` / 2 |

Each statement affected one row. GoogleSQL `INSERT OR UPDATE` matched the first two rows for both key states. These counts measure committed mutations in this fixture. Physical existence reads on cold HDD data remain unmeasured. [Tiered-storage behavior](https://docs.cloud.google.com/spanner/docs/tiered-storage)

- **`id` omitted from `SET`.** Both dialects accepted arithmetic, a `WHERE` condition, and a `SET` list that omitted an inserted column. The PostgreSQL DML reference lists restrictions on all three. [PostgreSQL DML](https://docs.cloud.google.com/spanner/docs/reference/postgresql/dml-syntax#on_conflict_clause) · [GoogleSQL DML](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-on-conflict-do-update)
- **`id=excluded.id` included in `SET`.** Direct assignment of every **inserted** column to its matching `excluded` value succeeded, even when the insert omitted `spare`. Keeping the key assignment while adding a condition, changing an expression or omitting another inserted column from `SET` returned `InvalidArgument` with `Key column id cannot be updated`.

The following statement stored 11 on insertion and 12 on conflict in both dialects. In separate tests on an existing row with `val = 7`, `WHERE plain_up.val < 10` allowed the update. With `WHERE plain_up.val > 10`, the row kept 7 and the commit reported zero affected rows and zero mutations.

```sql
INSERT INTO plain_up (id, val, spare) VALUES (1021, 11, 'sent')
ON CONFLICT (id) DO UPDATE
SET val = excluded.val + 1, spare = excluded.spare;
```

With a unique secondary index on non-null `code`, GoogleSQL accepted `ON CONFLICT(code) DO UPDATE SET val=excluded.val` and kept the original row's primary key. PostgreSQL returned `Unimplemented` and cited a null-filtering index. Its catalog reported `is_unique = YES`, `is_null_filtered = NO` and `code.is_nullable = NO`. The reason for that diagnostic remains open.

## Routine recheck

Of 337 SQL cases replayed from November 20, 335 retained their response code. Every case successful on both dates retained its result fields. The two changed responses depended on the available statistics. GoogleSQL returned a p99 for populated query statistics and rejected an explicitly empty aggregate. PostgreSQL returned zero rows on an empty transaction-statistics window. After the workload populated that window, the same query reproduced the error for a NULL bucket array. Eight lock-mode reads succeeded and rolled back. Both tagged transactions appeared in the minute window with NULL histogram counts. [November 20 measurements](evidence/managed-recheck-20261120/receipt.json)

The catalog hashes matched November 20 at 310 information-schema columns and 630 `SPANNER_SYS` columns. The empty-database canonical DDL check passed. The survey drift test and system capture still flagged the six `TXN_STATS_TOP` columns missing from the repository baseline.

The mirror remained at `8ff17fc`. The official Atom feed and the public BigQuery release-note table showed the November 18 QueryData announcement. BigQuery had shown November 16 in the preceding check. The release-note HTML and Developer Knowledge copy still led with November 10. [Mirror commit](https://github.com/apstndb/spanner-docs-mirror/commit/8ff17fc9c5e93332c20a5ce54f15cf7c8cb3ebaf) · [Spanner Atom feed](https://docs.cloud.google.com/feeds/spanner-release-notes.xml)

The public registry served **Omni 2026.r4-lts.3** for amd64 and arm64. The Omni release notes still led with October 29 and did not list that tag. The newest numbered regular Omni tag was 2026.r5.1, and the Emulator tag was 1.5.59. [Omni image manifest](https://us-docker.pkg.dev/v2/spanner-omni/images/spanner-omni/manifests/2026.r4-lts.3) · [Omni release notes](https://docs.cloud.google.com/spanner-omni/release-notes)
