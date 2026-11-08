---
name: documents/docs.cloud.google.com/spanner/docs/fgac-privileges
uri: https://docs.cloud.google.com/spanner/docs/fgac-privileges
title: Fine-grained access control privileges
description: Understand the privileges used in Spanner FGAC, such as SELECT, INSERT, UPDATE, DELETE, and EXECUTE.
data_source: docs.cloud.google.com
---

This page describes the privileges that you can grant to a database role for fine-grained access control. This information applies to both GoogleSQL-dialect databases and PostgreSQL-dialect databases.

To learn about database roles and fine-grained access control, see [Fine-grained access control overview](https://docs.cloud.google.com/spanner/docs/fgac-about) .

The following table shows the fine-grained access control privileges and the database objects that they can be granted on.

|                                                                                                                             | SELECT | INSERT | UPDATE | DELETE | EXECUTE | USAGE |
|-----------------------------------------------------------------------------------------------------------------------------|--------|--------|--------|--------|---------|-------|
| Schema                                                                                                                      |        |        |        |        |         | ✓     |
| Table                                                                                                                       | ✓      | ✓      | ✓      | ✓      |         |       |
| Column                                                                                                                      | ✓      | ✓      | ✓      |        | ✓       |       |
| View                                                                                                                        | ✓      |        |        |        |         |       |
| Change stream                                                                                                               | ✓      |        |        |        |         |       |
| Change stream [read function](https://docs.cloud.google.com/spanner/docs/change-streams/details#change_stream_query_syntax) |        |        |        |        | ✓       |       |
| Sequence                                                                                                                    | ✓      |        | ✓      |        |         |       |
| Model                                                                                                                       |        |        |        |        | ✓       |       |

The following sections provide details about each privilege.

## `SELECT`

Allows the role to read or query from a table, view, change stream, sequence, or model.

- If a column list is specified for a table, the privilege is valid on only those columns. If no column list is specified, then the privilege is valid on all columns in the table, including columns added afterward. A column list isn't allowed for a view.

- Spanner supports both invoker's rights views and definer's rights views. For more information, see [Views overview](https://docs.cloud.google.com/spanner/docs/views) .

  If you create a view with invoker's rights, to query the view, the database role or user needs the `SELECT` privilege on the view, and also the `SELECT` privilege on the underlying objects referenced in the view. For example, suppose the view `SingerNames` is created on the `Singers` table.

  ```
  CREATE VIEW SingerNames SQL SECURITY INVOKER AS
  SELECT Singers.SingerId, Singers.FirstName, Singers.LastName FROM Singers;
  ```

  Suppose that the database role `myRole` performs the query `SELECT * FROM SingerNames` . The role must have `SELECT` privilege on the view and must have `SELECT` privilege on the three referenced columns or on the entire `Singers` table.

  If you create a view with definer's rights, to query the view, the database role or user only needs the `SELECT` privilege on the view. For example, suppose the view `AlbumsBudget` is created on the `Albums` table.

  ```
  CREATE VIEW AlbumsBudget SQL SECURITY DEFINER AS
  SELECT Albums.Id, Albums.AlbumTitle, MarketingBudget FROM Albums;
  ```

  Suppose that the database role `Analyst` performs the query `SELECT * FROM AlbumsBudget` . The role only needs `SELECT` privilege on the view. It doesn't need the `SELECT` privilege on the three referenced columns or on the `Albums` table.

- After granting `SELECT` on a subset of columns for a table, the FGAC user can no longer use `SELECT *` on that table. Queries on that table must name all columns to be included.

- `SELECT` granted on a generated column doesn't grant `SELECT` on the underlying base columns.

- For interleaved tables, `SELECT` granted on the parent table doesn't propagate to the child table.

- When you grant `SELECT` on a change stream, you must also grant `EXECUTE` on the table-valued function for the change stream. For more information, see [EXECUTE](https://docs.cloud.google.com/spanner/docs/fgac-privileges#execute-privilege) .

- When `SELECT` is used with an aggregate function on specific columns, for example `SUM(col_a)` , the role must have the `SELECT` privilege on those columns. If the aggregate function doesn't specify any columns, for example `COUNT(*)` , the role must have the `SELECT` privilege on at least one column in the table.

- When you use `SELECT` with a sequence, you can only view sequences that you have privileges to view.

#### Examples for using `GRANT SELECT`

### GoogleSQL

```
GRANT SELECT ON TABLE employees TO ROLE hr_director;

GRANT SELECT ON TABLE customers, orders, items TO ROLE account_mgr;

GRANT SELECT(name, level, cost_center, location, manager) ON TABLE employees TO ROLE hr_manager;

GRANT SELECT(name, address, phone) ON TABLE employees, contractors TO ROLE hr_rep;

GRANT SELECT ON VIEW orders_view TO ROLE hr_manager;

GRANT SELECT ON CHANGE STREAM ordersChangeStream TO ROLE hr_analyst;

GRANT SELECT ON SEQUENCE sequence_name TO ROLE role_name;
```

### PostgreSQL

```
GRANT SELECT ON TABLE employees TO hr_director;

GRANT SELECT ON TABLE customers, orders, items TO account_mgr;

GRANT SELECT(name, level, cost_center, location, manager) ON TABLE employees TO hr_manager;

GRANT SELECT(name, address, phone) ON TABLE employees, contractors TO hr_rep;

GRANT SELECT ON TABLE orders_view TO hr_manager; // orders_view is an invoker rights view

GRANT SELECT ON CHANGE STREAM orders_change_stream TO hr_analyst;

GRANT SELECT ON SEQUENCE sequence_name TO hr_package;
```

## `INSERT`

Allows the role to insert rows into the specified tables. If a column list is specified, the permission is valid on only those columns. If no column list is specified, then the privilege is valid on all columns in the table.

- If column names are specified, any column not included gets its default value upon insert.

- `INSERT` can't be granted on generated columns.

#### Examples for using `GRANT INSERT`

### GoogleSQL

```
GRANT INSERT ON TABLE employees, contractors TO ROLE hr_manager;

GRANT INSERT(name, address, phone) ON TABLE employees TO ROLE hr_rep;
```

### PostgreSQL

```
GRANT INSERT ON TABLE employees, contractors TO hr_manager;

GRANT INSERT(name, address, phone) ON TABLE employees TO hr_rep;
```

## `UPDATE`

Allows the role to update rows in the specified tables. Updates can be restricted to a subset of table columns. When you use this with sequences, it allows the role to call the `get-next-sequence-value` function on the sequence.

In addition to the `UPDATE` privilege, the role needs the `SELECT` privilege on all queried columns. Queried columns include columns in the `WHERE` clause.

`UPDATE` can't be granted on generated columns.

#### Examples for using `GRANT UPDATE`

### GoogleSQL

```
GRANT UPDATE ON TABLE employees, contractors TO ROLE hr_manager;

GRANT UPDATE(name, address, phone) ON TABLE employees TO ROLE hr_rep;
```

### PostgreSQL

```
GRANT UPDATE ON TABLE employees, contractors TO hr_manager;

GRANT UPDATE(name, address, phone) ON TABLE employees TO hr_rep;
```

## `DELETE`

Allows the role to delete rows from the specified tables.

- `DELETE` can't be granted at the column level.

- The role also needs `SELECT` on any columns that might be included in the query's `WHERE` clauses.

- For interleaved tables in GoogleSQL-dialect databases, the `DELETE` privilege is required only on the parent table. If a child table specifies `ON DELETE CASCADE` , rows from the child table are deleted even without the `DELETE` privilege on the child table.

#### Example for using `GRANT DELETE`

### GoogleSQL

```
GRANT DELETE ON TABLE employees, contractors TO ROLE hr_admin;
```

### PostgreSQL

```
GRANT DELETE ON TABLE employees, contractors TO hr_admin;
```

## `EXECUTE`

When you grant `SELECT` on a change stream, you must also grant `EXECUTE` on the read function for the change stream. For more information, see [Change stream read functions and query syntax](https://docs.cloud.google.com/spanner/docs/change-streams/details#change_stream_query_syntax) .

When you use this with models, it allows the role to use the model in [machine learning functions](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/ml-functions) .

#### Example for using `GRANT EXECUTE`

The following example shows how to grant `EXECUTE` on the read function for the change stream named `my_change_stream` .

### GoogleSQL

```
GRANT EXECUTE ON TABLE FUNCTION READ_my_change_stream TO ROLE hr_analyst;
```

### PostgreSQL

```
GRANT EXECUTE ON FUNCTION spanner.read_json_my_change_stream TO hr_analyst;
```

## USAGE

When you grant `USAGE` to a named schema, it provides privileges to access objects contained in the named schema. The `USAGE` privilege is granted, by default, to the default schema.

## What's next

- [Configure fine-grained access control](https://docs.cloud.google.com/spanner/docs/configure-fgac)
- [Fine-grained access control overview](https://docs.cloud.google.com/spanner/docs/fgac-about)
- [GRANT and REVOKE statements](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#grant_and_revoke_statements) (GoogleSQL-dialect databases)
- [GRANT and REVOKE statements](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#grant_and_revoke_statements) (PostgreSQL-dialect databases)
