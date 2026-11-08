---
name: documents/docs.cloud.google.com/spanner/docs/fgac-system-roles
uri: https://docs.cloud.google.com/spanner/docs/fgac-system-roles
title: Fine-grained access control system roles
description: Learn about the predefined system roles in Spanner's fine-grained access control, including public, spanner_info_reader, and spanner_sys_reader.
data_source: docs.cloud.google.com
---

This page defines the characteristics, constraints, and intended use of the three predefined system roles that fine-grained access control provides for each database. Each system role has a different set of privileges, which can't be revoked. This information applies to both GoogleSQL-dialect databases and PostgreSQL-dialect databases.

### `public` system role

- All fine-grained access control users have IAM membership in `public` by default.

- All database roles inherit privileges from this role.

- Initially, `public` has no privileges, but you can grant it privileges. If you grant a privilege to `public` , it's available to all database roles, including database roles created afterward.

### `spanner_info_reader` system role

- This role has the `SELECT` privilege on [`INFORMATION_SCHEMA`](https://docs.cloud.google.com/spanner/docs/information-schema) views for GoogleSQL-dialect databases and [`information_schema`](https://docs.cloud.google.com/spanner/docs/information-schema-pg) views for PostgreSQL-dialect databases.

- You can't grant any other privileges to `spanner_info_reader` .

- Grant membership in this role to any database role that needs to have unfiltered read access to the `INFORMATION_SCHEMA` views (GoogleSQL-dialect databases) or the `information_schema` views (PostgreSQL-dialect databases).

### `spanner_sys_reader` system role

- This role has the `SELECT` privilege on `SPANNER_SYS` tables.

- You can't grant any other privileges to `spanner_sys_reader` .

- Grant membership in this role to any database role that must have read access to the `SPANNER_SYS` schema.

## Restrictions on system roles

- You can't delete a system role by using a `DROP ROLE` statement.

- System roles can't be members of other database roles. That is, the following GoogleSQL statement is invalid:

  ```
  GRANT ROLE pii_access TO ROLE spanner_info_reader;
  ```

- You can't grant membership in the `public` role to your database roles. For example, the following GoogleSQL statement is also invalid:

  ```
  GRANT ROLE public TO ROLE pii_access;
  ```

  However, you can grant membership in the `spanner_info_reader` and `spanner_sys_reader` roles. For example, the following are valid statements.

  ### GoogleSQL

  ````
    GRANT ROLE spanner_info_reader TO ROLE pii_access;
    GRANT ROLE spanner_sys_reader TO ROLE pii_access;
    ```
  ````

  ### PostgreSQL

  ```
  GRANT spanner_info_reader TO pii_access;
  GRANT spanner_sys_reader TO pii_access;
  ```

## What's next

- Learn how to [Configure fine-grained access control](https://docs.cloud.google.com/spanner/docs/configure-fgac) .
- Learn [About fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) .
