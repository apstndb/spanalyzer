---
name: documents/docs.cloud.google.com/spanner/docs/fgac-about
uri: https://docs.cloud.google.com/spanner/docs/fgac-about
title: Fine-grained access control overview
description: Use FGAC in Spanner to define table and column-level permissions using roles.
data_source: docs.cloud.google.com
---

Spanner fine-grained access control combines the benefits of [Identity and Access Management](https://docs.cloud.google.com/iam/docs) (IAM) with SQL role-based access control. With fine-grained access control, you define database roles, grant privileges to the roles, and create IAM policies to grant permissions on database roles to IAM principals. This page describes how to use fine-grained access control with Spanner for GoogleSQL-dialect databases and PostgreSQL-dialect databases.

As an administrator, you must enable fine-grained access control for individual IAM principals. Principals for whom fine-grained access control is enabled ("fine-grained access control users") must assume a database role to access Spanner resources.

Resource access for users who are are not fine-grained access control users is governed by IAM database-level roles. Fine-grained access control is fully compatible and can co-exist with existing IAM database-level access control. You can use it to access individual database objects. To control access to the entire database, use [IAM roles](https://docs.cloud.google.com/spanner/docs/iam) .

> **Note:** Spanner relies on IAM for identity management rather than managing user credentials within the database.

With fine-grained access control, you can control access to tables, columns, views, and change streams.

To manage fine-grained access control, you use the following DDL statements:

- `CREATE` and `DROP` statements for creating and dropping database roles. Database roles are collections of privileges. You can create up to 100 roles for a database.

- `GRANT` and `REVOKE` statements to grant and revoke privileges to and from database roles. Privileges include `SELECT` , `INSERT` , `UPDATE` , `DELETE` , and `EXECUTE` . Privilege names correspond to the like-named SQL statements. For example, a role with the `INSERT` privilege can execute the `INSERT` SQL statement on the tables that are specified in the `GRANT` statement.

  The following DDL statements grant `SELECT` on table `employees` to the `hr_rep` database role.

  ### GoogleSQL

  ```
  CREATE ROLE hr_rep;
  GRANT SELECT ON TABLE employees TO ROLE hr_rep;
  ```

  ### PostgreSQL

  ```
  CREATE ROLE hr_rep;
  GRANT SELECT ON TABLE employees TO hr_rep;
  ```

  For more information on privileges, see [Fine-grained access control privileges reference](https://docs.cloud.google.com/spanner/docs/fgac-privileges) .

- `GRANT` statements for granting roles to other roles to create hierarchies of roles, with privilege inheritance.

## Use cases

The following are sample use cases for fine-grained access control:

- An HR information system that has roles for sales compensation analyst, sales management, and HR analyst, each with different access levels on the data. For example, compensation analysts and sales management shouldn't see social security numbers.
- A ride-sharing application with different service accounts and privileges for riders and drivers.
- A ledger that permits `SELECT` and `INSERT` operations but not `UPDATE` and `DELETE` operations.

## Spanner resources and their privileges

The following is a list of Spanner resources and the [fine-grained access control privileges](https://docs.cloud.google.com/spanner/docs/fgac-privileges) that you can grant for them.

Schemas  
You can grant the `USAGE` privilege on schemas to specific database roles. For a non-default schema, database roles must have the `USAGE` privilege to access the database objects. The privilege check looks like the following:

Do you have `USAGE` on the schema?

No: Reject access.

Yes: Do you also have the appropriate rights on the table?

No: Reject access.

Yes: You can access the table.

> **Note:** You can use `ALL` to bulk grant privileges on all objects that use the same type in the schema. If you later add objects, you must grant privileges to the new objects as you add them.

Tables  
You can grant the `SELECT` , `INSERT` , `UPDATE` , and `DELETE` privileges on tables to database roles. For interleaved tables, a privilege granted on the parent table doesn't propagate to the child table.

Columns  
You can grant `SELECT` , `INSERT` , and `UPDATE` on a subset of columns in a table. The privilege is then valid only for those columns. `DELETE` is not permitted at the column level.

Views  
You can grant `SELECT` privilege on a view. Only `SELECT` is supported for views. Spanner supports both invoker's rights views and definer's rights views. If you create a view with invoker's rights, to query the view, the database role or user needs the `SELECT` privilege on the view, and also the `SELECT` privilege on the underlying objects referenced in the view. If you create a view with definer's rights, to query the view, the database role or user only needs the `SELECT` privilege on the view. For more information, see [Views overview](https://docs.cloud.google.com/spanner/docs/views) .

Change streams  
You can grant `SELECT` on change streams. You must also grant `EXECUTE` on the read function associated with a change stream. For information, see [Fine-grained access control for change streams](https://docs.cloud.google.com/spanner/docs/fgac-change-streams) .

Sequences  
You can grant `SELECT` and `UPDATE` on sequences. For information, see [Fine-grained access control for sequences](https://docs.cloud.google.com/spanner/docs/fgac-sequences) .

Models  
You can grant `EXECUTE` on models. For information, see [Fine-grained access control for models](https://docs.cloud.google.com/spanner/docs/fgac-models) .

## Fine-grained access control system roles

Fine-grained access control has predefined *system roles* for each database. Like user-defined database roles, system roles can control access to Spanner resources.

For example, a fine-grained access control user needs to be granted the `spanner_sys_reader` system role to access Key Visualizer, and needs the `spanner_info_reader` system role to be able to see unfiltered results when querying the `INFORMATION_SCHEMA` tables.

For more information, see [Fine-grained access control system roles](https://docs.cloud.google.com/spanner/docs/fgac-system-roles) .

## Database role hierarchies and inheritance

You can create hierarchies of database roles, where child roles inherit the privileges of parent roles. Child roles are known as *members* of the parent role.

For example, consider the following `GRANT` statements:

### GoogleSQL

```
GRANT SELECT ON TABLE employees TO ROLE pii_access;
GRANT ROLE pii_access TO ROLE hr_manager, hr_director;
```

### PostgreSQL

```
GRANT SELECT ON TABLE employees TO pii_access;
GRANT pii_access TO hr_manager, hr_director;
```

`hr_manager` and `hr_director` are members of role `pii_access` , and inherit the `SELECT` privilege on table `employees` .

![Privilege inheritance](https://docs.cloud.google.com/static/spanner/docs/images/role_hierarchy.png)

`hr_manager` and `hr_director` can also have members, and those members would inherit the `SELECT` privilege on `employees` .

There are no limits on the depth of role hierarchies, but query performance might degrade with deep and wide role hierarchy structures.

## Backup and restore

Spanner [backups](https://docs.cloud.google.com/spanner/docs/backup) include database role definitions. When a database is restored from backup, database roles are re-created with their granted privileges. However, IAM policies are not a part of database backups, so you must re-grant access to database roles to principals in the restored database.

## Overview of setting up fine-grained access control

The following are the high-level steps that you take to begin securing data with fine-grained access control. For details, see [Configure fine-grained access control](https://docs.cloud.google.com/spanner/docs/configure-fgac) .

You must be granted the `roles/spanner.admin` or `roles/spanner.databaseAdmin` IAM roles to perform these tasks.

1.  Create database roles and grant privileges to the roles.
2.  Optional: Create role hierarchies with inheritance by granting roles to other roles.
3.  Perform these steps for each principal who is to be a fine-grained access control user:
    1.  Enable fine-grained access control for the principal. The principal is then automatically granted the `public` database role, which has no privileges by default. This is a one-time operation for each principal.
    2.  Grant IAM permissions on one or more database roles to the principal.
    3.  After the principal is granted all required database roles, if the principal has database-level IAM roles, consider revoking the database-level roles so that the principal's access control is managed by only one method.

## Limitations

- Export operations don't export database roles and privileges, and import operations can't import them. You must manually set up roles and privileges after your import is complete.
- The **Data** tab on the **TABLE** page in the Google Cloud console is not available for fine-grained access control users.

## What's next

- [Access a database with fine-grained access control](https://docs.cloud.google.com/spanner/docs/access-with-fgac)
- [Fine-grained access control for change streams](https://docs.cloud.google.com/spanner/docs/fgac-change-streams)
- [Configure fine-grained access control](https://docs.cloud.google.com/spanner/docs/configure-fgac)
- [Fine-grained access control privileges reference](https://docs.cloud.google.com/spanner/docs/fgac-privileges)
- [Fine-grained access control system roles](https://docs.cloud.google.com/spanner/docs/fgac-system-roles)
- [GoogleSQL `GRANT` and `REVOKE` statements](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#grant_and_revoke_statements)
- [PostgreSQL `GRANT` and `REVOKE` statements](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#grant_and_revoke_statements)
- [Fine-grained access control for sequences](https://docs.cloud.google.com/spanner/docs/fgac-sequences)
- [Fine-grained access control for models](https://docs.cloud.google.com/spanner/docs/fgac-models)
