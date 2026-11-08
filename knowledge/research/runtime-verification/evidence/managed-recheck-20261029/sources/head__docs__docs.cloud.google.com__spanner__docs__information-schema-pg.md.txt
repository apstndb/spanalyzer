---
name: documents/docs.cloud.google.com/spanner/docs/information-schema-pg
uri: https://docs.cloud.google.com/spanner/docs/information-schema-pg
title: Information schema for PostgreSQL-dialect databases
description: Query the information_schema and spanner_sys schemas in Spanner PostgreSQL-dialect databases to retrieve database metadata.
data_source: docs.cloud.google.com
---

The information schema is a built-in schema that's common to every PostgreSQL database. You can run SQL queries against tables in the `information_schema` to fetch schema metadata for a database.

For example, the following query fetches the names of all user-defined tables in a database:

```
  SELECT
    table_schema,
    table_name
  FROM
    information_schema.tables
  WHERE
    table_schema NOT IN ('pg_catalog', 'information_schema', 'SPANNER_SYS')
    AND table_type = 'BASE TABLE'
```

## Usage

- `information_schema` tables are available only through SQL interfaces, for example:

  - The `executeQuery` API
  - The `gcloud spanner databases execute-sql` command
  - The **Query** page of a database in the Google Cloud console.

  Other single read methods don't support `information_schema` .

<!-- -->

- Queries against the `information_schema` can use strong, bounded staleness, or exact staleness [timestamp bounds](https://docs.cloud.google.com/spanner/docs/timestamp-bounds) .
- If you are using a GoogleSQL-dialect database, see [Information schema for GoogleSQL-dialect databases](https://docs.cloud.google.com/spanner/docs/information-schema) .

## Differences from `information_schema` for PostgreSQL

The tables in the `information_schema` for PostgreSQL-dialect databases include columns from the tables in the `information_schema` for open source PostgreSQL and in some cases also include columns from Spanner. In these tables, the open source PostgreSQL columns come first and in the same order as they do for a open source PostgreSQL database, and any distinct columns for Spanner are appended afterwards. Queries written for the open source PostgreSQL version of `information_schema` should work without modification when using PostgreSQL-dialect databases in Google Cloud CLI.

> **Note:** Queries against the `information_schema` for PostgreSQL-dialect databases that use `select *` and reference columns by offset might not work the same as they do against open source PostgreSQL databases. The GoogleSQL-specific columns can shift in position if PostgreSQL adds new columns to an `information_schema` table.

Other notable differences in the `information_schema` for PostgreSQL-dialect databases are:

- Some of the table columns for open source PostgreSQL are available, but not populated in PostgreSQL-dialect databases.
- PostgreSQL-dialect databases use `public` for the default schema name.
- Automatically generated constraint names use a different format than open source PostgreSQL databases.
- Tables related to open source PostgreSQL features that are not supported in PostgreSQL-dialect databases are not available.
- Some tables that are available with Spanner but not open source PostgreSQL, such as `database_options` , `index_columns` , `indexes` , and `spanner_statistics` are available.

## Row filtering in `information_schema` tables and views

Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` [system role](https://docs.cloud.google.com/spanner/docs/fgac-system-roles) (or to members of that role) can see all rows in all `information_schema` tables and views. For other principals, Spanner filters rows based on the current database role. The table and view descriptions in the following sections indicate how Spanner filters rows for each table and view.

## Tables in `information_schema` for PostgreSQL-dialect databases

The tables and views in the `information_schema` are compatible with the tables and views in the `information_schema` of open source PostgreSQL.

The following sections describe the tables and views in the `information_schema` for PostgreSQL-dialect databases.

### `applicable_roles`

This row-filtered view lists all role memberships that are explicitly granted to all database roles. Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only the role memberships that are granted to the current database role or to a role of which the current database role is a member.

Because all database roles are members of the [public role](https://docs.cloud.google.com/spanner/docs/fgac-system-roles#public) , the results omit records for implicit membership in the public role.

| **Column name** | **Type**            | **Description**                                                           |
|-----------------|---------------------|---------------------------------------------------------------------------|
| `grantee`       | `character varying` | The name of the database role to which membership is granted.             |
| `role_name`     | `character varying` | The name of the parent database role in which this membership is granted. |
| `is_grantable`  | `character varying` | Not used. Always `NO` .                                                   |

### `change_stream_columns`

This row-filtered view contains information about table columns and the change streams that watch them. Each row describes one change stream and one column. If a change stream tracks an entire table, then the columns in that table don't show in this view.

Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only rows for change streams on which the `SELECT` privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name             | Type                | Description                                                                                         |
|-------------------------|---------------------|-----------------------------------------------------------------------------------------------------|
| `change_stream_catalog` | `character varying` | The database name.                                                                                  |
| `change_stream_schema`  | `character varying` | The name of the change stream's schema. For PostgreSQL-dialect databases, the default is `public` . |
| `change_stream_name`    | `character varying` | The name of the change stream.                                                                      |
| `table_catalog`         | `character varying` | The database name.                                                                                  |
| `table_schema`          | `character varying` | The name of the table's schema. For PostgreSQL-dialect databases, the default is `public` .         |
| `table_name`            | `character varying` | The name of the table that this row refers to.                                                      |
| `column_name`           | `character varying` | The name of the column that this row refers to.                                                     |

### `change_stream_options`

This row-filtered view contains the configuration options for change streams. Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only options for change streams on which the `SELECT` privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name             | Type                | Description                                                                                         |
|-------------------------|---------------------|-----------------------------------------------------------------------------------------------------|
| `change_stream_catalog` | `character varying` | The database name.                                                                                  |
| `change_stream_schema`  | `character varying` | The name of the change stream's schema. For PostgreSQL-dialect databases, the default is `public` . |
| `change_stream_name`    | `character varying` | The name of the change stream.                                                                      |
| `option_name`           | `character varying` | The name of the change stream option.                                                               |
| `option_type`           | `character varying` | The data type of the change stream option.                                                          |
| `option_value`          | `character varying` | The value of the change stream option.                                                              |

### `change_stream_privileges`

This row-filtered view lists all fine-grained access control privileges granted on all change streams to any database role, including `public` . Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on change streams to the current database role, to roles of which the current database role is a member, or to `public` .

| **Column name**         | **Type**            | **Description**                                                                                                   |
|-------------------------|---------------------|-------------------------------------------------------------------------------------------------------------------|
| `grantor`               | `character varying` | Not used. Always `NULL` .                                                                                         |
| `grantee`               | `character varying` | The name of the database role to which this privilege is granted.                                                 |
| `change_stream_catalog` | `character varying` | The database name.                                                                                                |
| `change_stream_schema`  | `character varying` | The name of the schema that contains the change stream. The default is `public` for PostgreSQL-dialect databases. |
| `change_stream_name`    | `character varying` | The name of the change stream.                                                                                    |
| `privilege_type`        | `character varying` | The type of the privilege ( `SELECT` only).                                                                       |
| `is_grantable`          | `character varying` | Not used. Always `NO` .                                                                                           |

### `change_stream_tables`

This row-filtered view contains information about tables and the change streams that watch them. Each row describes one table and one change stream. Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only rows for change streams on which the `SELECT` privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

The data in `change_stream_tables` does not include the implicit relationships between tables and change streams that track the entire database.

| Column name             | Type                | Description                                                                                                                                                                                                     |
|-------------------------|---------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `change_stream_catalog` | `character varying` | The database name.                                                                                                                                                                                              |
| `change_stream_schema`  | `character varying` | The name of the change stream's schema. For PostgreSQL-dialect databases, the default is `public` .                                                                                                             |
| `change_stream_name`    | `character varying` | The name of the change stream that this row refers to.                                                                                                                                                          |
| `table_catalog`         | `character varying` | The database name.                                                                                                                                                                                              |
| `table_schema`          | `character varying` | The name of the table's schema. For PostgreSQL-dialect databases, the default is `public` .                                                                                                                     |
| `table_name`            | `character varying` | The name of the table that this row refers to.                                                                                                                                                                  |
| `all_columns`           | `character varying` | `YES` if this row's change stream tracks the entirety of the table this row refers to. Otherwise, `NO` . In accordance with the SQL standard, the string is either `YES` or `NO` , rather than a Boolean value. |

### `change_streams`

This row-filtered view lists all of a database's change streams, and notes which ones track the entire database versus specific tables or columns. Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only change streams on which the `SELECT` fine-grained access control privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name             | Type                | Description                                                                                                                                                                                                                  |
|-------------------------|---------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `change_stream_catalog` | `character varying` | The database name.                                                                                                                                                                                                           |
| `change_stream_schema`  | `character varying` | The name of this change stream's schema. For PostgreSQL-dialect databases, the default is `public` .                                                                                                                         |
| `change_stream_name`    | `character varying` | The name of the change stream.                                                                                                                                                                                               |
| `all`                   | `character varying` | `YES` if this change stream tracks the entire database. `NO` if this change stream tracks specific tables or columns. In accordance with the SQL standard, the string is either `YES` or `NO` , rather than a Boolean value. |

### `check_constraints`

The `check_constraints` view contains one row for each check constraint defined by either the `CHECK` or the `NOT NULL` keyword.

<table>
<colgroup>
<col style="width: 33%" />
<col style="width: 33%" />
<col style="width: 33%" />
</colgroup>
<tbody>
<tr class="header">
<th>Column name</th>
<th>Type</th>
<th>Description</th>
</tr>

<tr class="odd">
<td><code>constraint_catalog</code></td>
<td><code>character varying</code></td>
<td>The database name.</td>
</tr>
<tr class="even">
<td><code>constraint_schema</code></td>
<td><code>character varying</code></td>
<td>The name of the constraint's schema. The default is <code>public</code> for PostgreSQL-dialect databases.</td>
</tr>
<tr class="odd">
<td><code>constraint_name</code></td>
<td><code>character varying</code></td>
<td>The name of the constraint. If the name of the constraint is not explicitly specified in the schema, the auto-generated name is used.</td>
</tr>
<tr class="even">
<td><code>check_clause</code></td>
<td><code>character varying</code></td>
<td>The check constraint's expression.</td>
</tr>
<tr class="odd">
<td><code>spanner_state</code></td>
<td><code>character varying</code></td>
<td>The current state of the check constraint. The possible states are as follows:
<ul>
<li><code>VALIDATING</code> : The PostgreSQL-dialect database is validating the existing data for an <code>ALTER CONSTRAINT</code> or <code>ADD CONSTRAINT</code> command.</li>
<li><code>COMMITTED</code> : There is no active schema change for this constraint.</li>
</ul></td>
</tr>
</tbody>
</table>

### `column_column_usage`

This view lists all the generated columns that depend on another base column in the same table.

| Column name        | Type                | Description                                                                                                                                                                                            |
|--------------------|---------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `table_catalog`    | `character varying` | The database name.                                                                                                                                                                                     |
| `table_schema`     | `character varying` | The name of the schema that contains the table. The name is `public` for the default schema and non-empty for other schemas (for example, the `information_schema` itself). This column is never null. |
| `table_name`       | `character varying` | The name of the table that contains the generated columns.                                                                                                                                             |
| `column_name`      | `character varying` | The name of the base column that the generated column depends on.                                                                                                                                      |
| `dependent_column` | `character varying` | The name of the generated column.                                                                                                                                                                      |

### `column_options`

This view lists all the options defined for the referenced table columns of a foreign key constraint. The view contains only those columns in the reference table that the current user has access to (by way of being the owner or granted privileges).

| Column name     | Type                | Description                                                                                                                                                                                                    |
|-----------------|---------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `table_catalog` | `character varying` | The database name.                                                                                                                                                                                             |
| `table_schema`  | `character varying` | The name of the schema that contains the foreign table. The name is `public` for the default schema and non-empty for other schemas (for example, the `information_schema` itself). This column is never null. |
| `table_name`    | `character varying` | The name of the foreign table.                                                                                                                                                                                 |
| `column_name`   | `character varying` | The name of the column.                                                                                                                                                                                        |
| `option_name`   | `character varying` | A SQL identifier that uniquely identifies the option. This identifier is the key of the `OPTIONS` clause in DDL.                                                                                               |
| `option_value`  | `character varying` | A SQL literal describing the value of this option. The value of this column is parsable as part of a query.                                                                                                    |
| `option_type`   | `character varying` | A data type name that is the type of this option value.                                                                                                                                                        |

### `column_privileges`

This row-filtered view lists all fine-grained access control privileges granted on all columns to any database role, including `public` . Principals that have IAM database-level permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on columns to the current database role, to roles of which the current database role is a member, or to `public` .

The view includes the `SELECT` , `INSERT` , and `UPDATE` privileges that the column inherits from the table or view that contains the column.

| **Column name**  | **Type**            | **Description**                                                                                                   |
|------------------|---------------------|-------------------------------------------------------------------------------------------------------------------|
| `grantor`        | `character varying` | Not used. Always `NULL` .                                                                                         |
| `grantee`        | `character varying` | The name of the database role to which this privilege is granted.                                                 |
| `table_catalog`  | `character varying` | The database name.                                                                                                |
| `table_schema`   | `character varying` | The name of the schema that contains the table or view. The default is `public` for PostgreSQL-dialect databases. |
| `table_name`     | `character varying` | The name of the table or view that contains the column.                                                           |
| `column_name`    | `character varying` | The name of the column.                                                                                           |
| `privilege_type` | `character varying` | The type of the privilege ( `SELECT` , `INSERT` , or `UPDATE` ).                                                  |
| `is_grantable`   | `character varying` | Not used. Always `NO` .                                                                                           |

### `columns`

This row-filtered view provides information about all table columns and view columns in the database. Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only columns that have any fine-grained access control privileges granted on them (or the `SELECT` , `INSERT` or `UPDATE` privileges granted on their containing tables) to the current database role, to roles of which the current database role is a member, or to `public` .

<table>
<colgroup>
<col style="width: 33%" />
<col style="width: 33%" />
<col style="width: 33%" />
</colgroup>
<tbody>
<tr class="header">
<th>Column name</th>
<th>Type</th>
<th>Description</th>
</tr>

<tr class="odd">
<td><code>table_catalog</code></td>
<td><code>character varying</code></td>
<td>The database name.</td>
</tr>
<tr class="even">
<td><code>table_schema</code></td>
<td><code>character varying</code></td>
<td>The name of the schema that contains the table. The name is <code>public</code> for the default schema and non-empty for other schemas (for example, the <code>information_schema</code> itself). This column is never null.</td>
</tr>
<tr class="odd">
<td><code>table_name</code></td>
<td><code>character varying</code></td>
<td>The name of the table</td>
</tr>
<tr class="even">
<td><code>column_name</code></td>
<td><code>character varying</code></td>
<td>The name of the column</td>
</tr>
<tr class="odd">
<td><code>ordinal_position</code></td>
<td><code>BIGINT</code></td>
<td>The ordinal position of the column in the table, starting with a value of 1</td>
</tr>
<tr class="even">
<td><code>column_default</code></td>
<td><code>character varying</code></td>
<td>A string representation of the open source PostgreSQL expression of the default value of the column, for example, <code>'9'::bigint</code> .</td>
</tr>
<tr class="odd">
<td><code>is_nullable</code></td>
<td><code>character varying</code></td>
<td>A string that indicates whether the column is nullable. In accordance with the SQL standard, the string is either <code>YES</code> or <code>NO</code> , rather than a Boolean value.</td>
</tr>
<tr class="even">
<td><code>data_type</code></td>
<td><code>character varying</code></td>
<td>The data type of the column. The value is one of the following:
<ul>
<li>For built-in types, the name of the data type.</li>
<li>For arrays, the value ARRAY.</li>
</ul></td>
</tr>
<tr class="odd">
<td><code>character_maximum_length</code></td>
<td><code>BIGINT</code></td>
<td>The declared maximum length for character and bit string data types. If a maximum length was not specified, then the value is <code>NULL</code> . If the data type of the column is not a character or bit string, then the value is <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>character_octet_length</code></td>
<td><code>BIGINT</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>numeric_precision</code></td>
<td><code>BIGINT</code></td>
<td>The precision of the numeric data type of the current column. For <code>double precision</code> , the value is 53. For <code>bigint</code> , the value is 64. For all other data types, the value is <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>numeric_precision_radix</code></td>
<td><code>BIGINT</code></td>
<td>The base (unit) of the precision for numeric types. Only two values are supported:
<ul>
<li>2 for <code>double precision</code> <code>float8</code> , and <code>bigint</code></li>
<li>10 for <code>numeric</code></li>
</ul>
For all other data types the value is <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>numeric_scale</code></td>
<td><code>BIGINT</code></td>
<td>Contains the scale of the numeric column type, which is the number of precision base units after the radix point. For <code>bigint</code> , the value is 0. For all other data types, the value is <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>datetime_precision</code></td>
<td><code>BIGINT</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>interval_type</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>interval_precision</code></td>
<td><code>BIGINT</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>character_set_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>character_set_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>character_set_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>collation_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>collation_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>collation_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>domain_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>domain_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>domain_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>udt_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>udt_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>udt_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>scope_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>scope_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>scope_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>maximum_cardinality</code></td>
<td><code>BIGINT</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>dtd_identifier</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>is_self_referencing</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>is_identity</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>identity_generation</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>identity_start</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>identity_increment</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>identity_maximum</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>identity_minimum</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>identity_cycle</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>is_generated</code></td>
<td><code>character varying</code></td>
<td>A string that indicates whether the column is generated. The string is either <code>ALWAYS</code> for a generated column or <code>NEVER</code> for a non-generated column.</td>
</tr>
<tr class="odd">
<td><code>generation_expression</code></td>
<td><code>character varying</code></td>
<td>A string representing the SQL expression of a generated column, or <code>NULL</code> if the column is not a generated column.</td>
</tr>
<tr class="even">
<td><code>is_updatable</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>spanner_type</code></td>
<td><code>character varying</code></td>
<td>A string holding the DDL-compatible type of the column.</td>
</tr>
<tr class="even">
<td><code>is_stored</code></td>
<td><code>character varying</code></td>
<td>A string that indicates whether the generated column is stored. The string is always <code>YES</code> or <code>NO</code> for generated columns, and <code>NULL</code> for non-generated columns.</td>
</tr>
<tr class="odd">
<td><code>spanner_state</code></td>
<td><code>character varying</code></td>
<td>The current state of the column. A new stored generated column added to an existing table may go through multiple user-observable states before it is fully usable. Possible values are:
<ul>
<li><code>NO_WRITE</code> : No read or write is allowed to the columns. A stored generated column in this state does not cause any client effect.</li>
<li><code>WRITE_ONLY</code> : The column is being backfilled. No read is allowed.</li>
<li><code>COMMITTED</code> : The column is fully usable.</li>
<li><code>NULL</code> : Used for columns in system schemas.</li>
</ul></td>
</tr>
<tr class="even">
<td><code>on_update_expression</code></td>
<td><code>character varying</code></td>
<td>A string representation of the `ON UPDATE` expression for a column. If the column does not have an `ON UPDATE` expression, the value is `NULL`.</td>
</tr>
</tbody>
</table>

### `constraint_column_usage`

This view contains one row about each column used by a constraint.

- For `PRIMARY KEY` and `CHECK` constraints defined by the `NOT NULL` keyword, the view contains those columns.
- For `CHECK` constraints created with the `CHECK` keyword, the view includes the columns used by the check constraint expression.
- For foreign key constraints, the view contains the columns of the referenced table.
- For `UNIQUE` constraints, the view contains the columns from `KEY_COLUMN_USAGE` .

| Column name          | Type                | Description                                                                                             |
|----------------------|---------------------|---------------------------------------------------------------------------------------------------------|
| `table_catalog`      | `character varying` | The database name.                                                                                      |
| `table_schema`       | `character varying` | The name of the schema that contains the table that contains the column that is used by the constraint. |
| `table_name`         | `character varying` | The name of the table that contains the column that is used by the constraint.                          |
| `column_name`        | `character varying` | The name of the column that is used by the constraint.                                                  |
| `constraint_catalog` | `character varying` | The database name.                                                                                      |
| `constraint_schema`  | `character varying` | The name of the constraint's schema.                                                                    |
| `constraint_name`    | `character varying` | The name of the constraint.                                                                             |

### `constraint_table_usage`

This view contains one row for each table used by a constraint. For `FOREIGN KEY` constraints, the table information is for the tables in the `REFERENCES` clause. For a unique or primary key constraint, this view identifies the table the constraint belongs to. Check constraints and not-null constraints are not included in this view.

| Column name          | Type                | Description                                            |
|----------------------|---------------------|--------------------------------------------------------|
| `table_catalog`      | `character varying` | The database name.                                     |
| `table_schema`       | `character varying` | The name of the constrained table's schema.            |
| `table_name`         | `character varying` | The name of the table that is used by some constraint. |
| `constraint_catalog` | `character varying` | The database name.                                     |
| `constraint_schema`  | `character varying` | The name of the schema that contains the constraint.   |
| `constraint_name`    | `character varying` | The name of the constraint.                            |

### `database_options`

This table lists the options that are set on the database.

| Column name    | Type                | Description                                                                                 |
|----------------|---------------------|---------------------------------------------------------------------------------------------|
| `catalog_name` | `character varying` | The database name.                                                                          |
| `schema_name`  | `character varying` | The name of the schema. The default value is `public` for PostgreSQL-dialect databases.     |
| `option_name`  | `character varying` | The name of the database option. This is the value of `key` in the `OPTIONS` clause in DDL. |
| `option_type`  | `character varying` | The data type of the database option.                                                       |
| `option_value` | `character varying` | The value of the database option.                                                           |

### `enabled_roles`

This row-filtered view lists the defined database roles. Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all database roles. All other principals can see only database roles to which they have been granted access either directly or through inheritance. All system roles excluding `public` also appear in this view.

| **Column name**     | **Type**            | **Description**                                       |
|---------------------|---------------------|-------------------------------------------------------|
| `role_name`         | `character varying` | The name of the role.                                 |
| `spanner_is_system` | `character varying` | `YES` if the role is a system role. Otherwise, `NO` . |

### `index_columns`

This view lists the columns in an index.

| Column name        | Type                | Description                                                                                                                                                                                                                                                                                 |
|--------------------|---------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `table_catalog`    | `character varying` | The database name.                                                                                                                                                                                                                                                                          |
| `table_schema`     | `character varying` | The name of the schema that contains the index. The default value is `public` .                                                                                                                                                                                                             |
| `table_name`       | `character varying` | The name of the table associated with the index.                                                                                                                                                                                                                                            |
| `index_name`       | `character varying` | The name of the index. Tables that have a `PRIMARY KEY` specification have a pseudo-index entry generated with the name `PRIMARY_KEY` .                                                                                                                                                     |
| `index_type`       | `character varying` | The type of index. Possible values are `PRIMARY_KEY` , `LOCAL` , or `GLOBAL` .                                                                                                                                                                                                              |
| `column_name`      | `character varying` | The name of the column.                                                                                                                                                                                                                                                                     |
| `ordinal_position` | `BIGINT`            | The ordinal position of the column in the index (or primary key), starting with a value of 1. This value is `NULL` for non-key columns (for example, columns specified in the [`INCLUDE` clause](https://docs.cloud.google.com/spanner/docs/secondary-indexes#storing_clause) of an index). |
| `column_ordering`  | `character varying` | The sort order of the column. The value is `ASC` or `DESC` for key columns, and `NULL` for non-key columns (for example, columns specified in the `STORING` clause of an index).                                                                                                            |
| `is_nullable`      | `character varying` | A string that indicates whether the column is nullable. In accordance with the SQL standard, the string is either `YES` or `NO` , rather than a Boolean value.                                                                                                                              |
| `spanner_type`     | `character varying` | A string holding the DDL-compatible type of the column.                                                                                                                                                                                                                                     |

### `indexes`

This view lists the indexes in a schema.

<table>
<colgroup>
<col style="width: 33%" />
<col style="width: 33%" />
<col style="width: 33%" />
</colgroup>
<tbody>
<tr class="header">
<th>Column name</th>
<th>Type</th>
<th>Description</th>
</tr>

<tr class="odd">
<td><code>table_catalog</code></td>
<td><code>character varying</code></td>
<td>The database name.</td>
</tr>
<tr class="even">
<td><code>table_schema</code></td>
<td><code>character varying</code></td>
<td>The name of the schema. The default value is <code>public</code> .</td>
</tr>
<tr class="odd">
<td><code>table_name</code></td>
<td><code>character varying</code></td>
<td>The name of the table.</td>
</tr>
<tr class="even">
<td><code>index_name</code></td>
<td><code>character varying</code></td>
<td>The name of the index. Tables created with a <code>PRIMARY KEY</code> clause have a pseudo-index entry generated with the name <code>PRIMARY_KEY</code> , which allows the fields of the primary key to be identified.</td>
</tr>
<tr class="odd">
<td><code>index_type</code></td>
<td><code>character varying</code></td>
<td>The type of the index. The values include <code>PRIMARY_KEY</code> , <code>LOCAL</code> , or <code>GLOBAL</code> .</td>
</tr>
<tr class="even">
<td><code>parent_table_name</code></td>
<td><code>character varying</code></td>
<td>Secondary indexes can be interleaved in a parent table, as discussed in <a href="https://docs.cloud.google.com/spanner/docs/secondary-indexes#creating_a_secondary_index">Creating a secondary index</a> . This column holds the name of that parent table, or an empty string if the index is not interleaved.</td>
</tr>
<tr class="odd">
<td><code>is_unique</code></td>
<td><code>character varying</code></td>
<td>Whether the index keys must be unique. In accordance with the SQL standard, the string is either <code>YES</code> or <code>NO</code> , rather than a Boolean value.</td>
</tr>
<tr class="even">
<td><code>is_null_filtered</code></td>
<td><code>character varying</code></td>
<td>Whether the index includes entries with <code>NULL</code> values. In accordance with the SQL standard, the string is either <code>YES</code> or <code>NO</code> , rather than a Boolean value.</td>
</tr>
<tr class="odd">
<td><code>index_state</code></td>
<td><code>character varying</code></td>
<td>The current state of the index. Possible values and the states they represent are:
<ul>
<li><code>NULL</code> : the index type is <code>PRIMARY_KEY</code></li>
<li><code>PREPARE</code> : creating empty tables for a new index</li>
<li><code>WRITE_ONLY</code> : backfilling data for a new index</li>
<li><code>WRITE_ONLY_CLEANUP</code> : cleaning up a new index</li>
<li><code>WRITE_ONLY_VALIDATE_UNIQUE</code> : checking uniqueness of data in a new index</li>
<li><code>READ_WRITE</code> : normal index operation</li>
</ul></td>
</tr>
<tr class="even">
<td><code>spanner_is_managed</code></td>
<td><code>character varying</code></td>
<td>Whether the index is managed by Spanner. For example, secondary backing indexes for foreign keys are managed by Spanner. The string is either <code>YES</code> or <code>NO</code> , rather than a Boolean value, in accordance with the SQL standard.</td>
</tr>
</tbody>
</table>

### `information_schema_catalog_name`

This table contains one row and one column containing the database name.

| Column name    | Type                | Description        |
|----------------|---------------------|--------------------|
| `catalog_name` | `character varying` | The database name. |

### `key_column_usage`

This view identifies all columns in the current database that are referenced by a unique, primary key, or foreign key constraint. For information about `CHECK` constraint columns, see the `check_constraints` view.

| Column name                     | Type                | Description                                                                                                                                                                     |
|---------------------------------|---------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `constraint_catalog`            | `character varying` | The database name.                                                                                                                                                              |
| `constraint_schema`             | `character varying` | The name of the constraint's schema. The default value is `public` .                                                                                                            |
| `constraint_name`               | `character varying` | The name of the constraint.                                                                                                                                                     |
| `table_catalog`                 | `character varying` | The database name.                                                                                                                                                              |
| `table_schema`                  | `character varying` | The name of the schema that contains the table that contains the constrained column. The default value is `public` .                                                            |
| `table_name`                    | `character varying` | The name of the table that contains the column that is restricted by this constraint.                                                                                           |
| `column_name`                   | `character varying` | The name of the column that is constrained.                                                                                                                                     |
| `ordinal_position`              | `BIGINT`            | The ordinal position of the column within the constraint's key, starting with a value of `1` .                                                                                  |
| `position_in_unique_constraint` | `BIGINT`            | For `FOREIGN KEY` s, the ordinal position of the column within the unique constraint, starting with a value of `1` . This column has a `NULL` value for other constraint types. |

### `parameters`

This row-filtered view defines the arguments for each user defined and change stream read function. Each row describes one argument for one user defined or change stream read function.

Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only parameters for user defined and change stream read functions on which the `EXECUTE` fine-grained access control privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

<table>
<colgroup>
<col style="width: 33%" />
<col style="width: 33%" />
<col style="width: 33%" />
</colgroup>
<thead>
<tr class="header">
<th>Column name</th>
<th>Type</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><code>specific_catalog</code></td>
<td><code>character varying</code></td>
<td>The database name.</td>
</tr>
<tr class="even">
<td><code>specific_schema</code></td>
<td><code>character varying</code></td>
<td>The name of the routine's schema. For PostgreSQL-dialect databases, the default is <code>public</code> .</td>
</tr>
<tr class="odd">
<td><code>specific_name</code></td>
<td><code>character varying</code></td>
<td>The name of the routine. Uniquely identifies the routine even if its name is overloaded.</td>
</tr>
<tr class="even">
<td><code>ordinal_position</code></td>
<td><code>bigint</code></td>
<td>The ordinal position of the parameter in the argument list of the routine, starting with a value 1.</td>
</tr>
<tr class="odd">
<td><code>parameter_mode</code></td>
<td><code>character varying</code></td>
<td>Not used. Always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>is_result</code></td>
<td><code>character varying</code></td>
<td>Not used. Always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>as_locator</code></td>
<td><code>character varying</code></td>
<td>Not used. Always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>parameter_name</code></td>
<td><code>character varying</code></td>
<td>The name of the parameter.</td>
</tr>
<tr class="odd">
<td><code>data_type</code></td>
<td><code>character varying</code></td>
<td>The data type of the parameter. The value is one of the following:<br />

<ul>
<li>For built-in types, the name of the data type.</li>
</ul>
<ul>
<li>For arrays, the value <code>ARRAY</code> .</li>
</ul></td>
</tr>
<tr class="even">
<td><code>character_maximum_length</code></td>
<td><code>bigint</code></td>
<td>Not used. Always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>character_octet_length</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>character_set_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>character_set_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>character_set_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>collation_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>collation_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>collation_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>numeric_precision</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>numeric_precision_radix</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>numeric_scale</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>datetime_precision</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>interval_type</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>interval_precision</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>udt_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>udt_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>udt_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>scope_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>scope_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>scope_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>maximum_cardinality</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>dtd_identifier</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>parameter_default</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>spanner_type</code></td>
<td><code>character varying</code></td>
<td>The detailed return type of the routine. Includes the subtype if an <code>ARRAY</code> is returned.</td>
</tr>
</tbody>
</table>

### `placements`

This table lists the placements in the database.

| Column name      | Type                | Description                                                                                                                                                    |
|------------------|---------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `placement_name` | `character varying` | The name of the placement.                                                                                                                                     |
| `is_default`     | `character varying` | A string that indicates whether the column is nullable. In accordance with the SQL standard, the string is either `YES` or `NO` , rather than a Boolean value. |

### `placement-options`

For each placement, this table lists the options that are set on the placement in the `OPTIONS` clause of the [`CREATE PLACEMENT`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#create-placement) statement.

<table>
<colgroup>
<col style="width: 33%" />
<col style="width: 33%" />
<col style="width: 33%" />
</colgroup>
<tbody>
<tr class="header">
<th>Column name</th>
<th>Type</th>
<th>Description</th>
</tr>

<tr class="odd">
<td><code>placement_name</code></td>
<td><code>character varying</code></td>
<td>The name of the placement.</td>
</tr>
<tr class="even">
<td><code>option_name</code></td>
<td><code>character varying</code></td>
<td>The name of the placement option. The valid values for <code>option_name</code> include:
<ul>
<li><code>instance_partition</code></li>
<li><code>default_leader</code></li>
</ul></td>
</tr>
<tr class="odd">
<td><code>option_type</code></td>
<td><code>character varying</code></td>
<td>The data type of the placement option.</td>
</tr>
<tr class="even">
<td><code>option_value</code></td>
<td><code>character varying</code></td>
<td>The value of the placement option. For <code>instance_partition</code> , this is the name of the instance partition. For <code>default_leader</code> , it's the name of the default leader region.</td>
</tr>
</tbody>
</table>

### `locality-group-options`

For each locality group, this table lists the name and options that are set on the locality group in the `OPTIONS` clause of the [`CREATE LOCALITY GROUP`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#create-locality-group) statement.

<table>
<colgroup>
<col style="width: 33%" />
<col style="width: 33%" />
<col style="width: 33%" />
</colgroup>
<tbody>
<tr class="header">
<th>Column name</th>
<th>Type</th>
<th>Description</th>
</tr>

<tr class="odd">
<td><code>locality_group_name</code></td>
<td><code>character varying</code></td>
<td>The name of the locality group.</td>
</tr>
<tr class="even">
<td><code>option_name</code></td>
<td><code>character varying</code></td>
<td>The name of the locality group option. The valid options are:
<ul>
<li><code>storage</code> : defines the storage type for the locality group.</li>
<li><code>ssd_to_hdd_spill_timespan</code> : defines how long data is stored in SSD storage before it moves to HDD storage.</li>
</ul></td>
</tr>
<tr class="odd">
<td><code>option_value</code></td>
<td><code>character varying</code></td>
<td>The value of the locality group option. For <code>storage</code> , this is either <code>ssd</code> or <code>hdd</code> . For <code>ssd_to_hdd_spill_timespan</code> , this is the amount of time that data must be stored in SSD before it's moved to HDD storage. For example, <code>10d</code> is 10 days. The minimum amount of time you can set is one hour.</td>
</tr>
</tbody>
</table>

### `referential_constraints`

This view contains one row about each `FOREIGN KEY` constraint. You can see only those constraints for which you have write access to the referencing table. This view also identifies the `PRIMARY KEY` and `UNIQUE` constraints on the referenced tables that the foreign keys use for constraint enforcement and referential actions.

<table>
<colgroup>
<col style="width: 33%" />
<col style="width: 33%" />
<col style="width: 33%" />
</colgroup>
<tbody>
<tr class="header">
<th>Column name</th>
<th>Type</th>
<th>Description</th>
</tr>

<tr class="odd">
<td><code>constraint_catalog</code></td>
<td><code>character varying</code></td>
<td>The database name.</td>
</tr>
<tr class="even">
<td><code>constraint_schema</code></td>
<td><code>character varying</code></td>
<td>The name of the schema that contains the foreign key constraint. The default value is <code>public</code> .</td>
</tr>
<tr class="odd">
<td><code>constraint_name</code></td>
<td><code>character varying</code></td>
<td>The name of the foreign key constraint.</td>
</tr>
<tr class="even">
<td><code>unique_constraint_catalog</code></td>
<td><code>character varying</code></td>
<td>The database name.</td>
</tr>
<tr class="odd">
<td><code>unique_constraint_schema</code></td>
<td><code>character varying</code></td>
<td>The name of the schema that contains the unique or primary key constraint that the foreign key constraint references.</td>
</tr>
<tr class="even">
<td><code>unique_constraint_name</code></td>
<td><code>character varying</code></td>
<td>The name of the unique or primary key constraint that the foreign key constraint references.</td>
</tr>
<tr class="odd">
<td><code>match_option</code></td>
<td><code>character varying</code></td>
<td>The match method used by the foreign key constraint. The value is always <code>NONE</code> .</td>
</tr>
<tr class="even">
<td><code>update_rule</code></td>
<td><code>character varying</code></td>
<td>The update rule of the foreign key constraint. This value is always <code>NO ACTION</code> .</td>
</tr>
<tr class="odd">
<td><code>delete_rule</code></td>
<td><code>character varying</code></td>
<td>The delete rule of the foreign key constraint. This value is either <code>CASCADE</code> , <code>SET NULL</code> , or <code>NO ACTION</code> .</td>
</tr>
<tr class="even">
<td><code>spanner_state</code></td>
<td><code>character varying</code></td>
<td>The current state of the foreign key. Spanner does not begin enforcing the constraint until the foreign key's backing indexes are created and backfilled. Once the indexes are ready, Spanner begins enforcing the constraint for new transactions while it validates the existing data. The possible values and the states they represent are:
<ul>
<li><code>BACKFILLING_INDEXES</code> : Indexes are being backfilled.</li>
<li><code>VALIDATING_DATA</code> : Existing data and new writes are being validated.</li>
<li><code>WAITING_FOR_COMMIT</code> : The foreign key bulk operations have completed successfully, or none were needed, but the foreign key is still pending.</li>
<li><code>COMMITTED</code> : The schema change was committed.</li>
</ul></td>
</tr>
</tbody>
</table>

### `role_change_stream_grants`

This row-filtered view lists the `SELECT` privileges granted on all change streams to any database role, including `public` . Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on change streams to the current database role and to roles of which the current database role is a member, not including `public` .

| **Column name**         | **Type**            | **Description**                                                                                                   |
|-------------------------|---------------------|-------------------------------------------------------------------------------------------------------------------|
| `grantor`               | `character varying` | Not used. Always `NULL` .                                                                                         |
| `grantee`               | `character varying` | The name of the database role to which this privilege is granted.                                                 |
| `change_stream_catalog` | `character varying` | The database name.                                                                                                |
| `change_stream_schema`  | `character varying` | The name of the schema that contains the change stream. The default is `public` for PostgreSQL-dialect databases. |
| `change_stream_name`    | `character varying` | The name of the change stream.                                                                                    |
| `privilege_type`        | `character varying` | The type of the privilege ( `SELECT` only).                                                                       |
| `is_grantable`          | `character varying` | Not used. Always `NO` .                                                                                           |

### `role_column_grants`

This row-filtered view lists all fine-grained access control privileges granted on all columns to any database role, including `public` . Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on columns to the current database role and to roles of which the current database role is a member, not including `public` .

The view includes the `SELECT` , `INSERT` , and `UPDATE` privileges that the column inherits from the table or view that contains the column.

| **Column name**  | **Type**            | **Description**                                                                                                   |
|------------------|---------------------|-------------------------------------------------------------------------------------------------------------------|
| `grantor`        | `character varying` | Not used. Always `NULL` .                                                                                         |
| `grantee`        | `character varying` | The name of the database role to which this privilege is granted.                                                 |
| `table_catalog`  | `character varying` | The database name.                                                                                                |
| `table_schema`   | `character varying` | The name of the schema that contains the table or view. The default is `public` for PostgreSQL-dialect databases. |
| `table_name`     | `character varying` | The name of the table or view that contains the column.                                                           |
| `column_name`    | `character varying` | The name of the column.                                                                                           |
| `privilege_type` | `character varying` | The type of the privilege ( `SELECT` , `INSERT` , or `UPDATE` ).                                                  |
| `is_grantable`   | `character varying` | Not used. Always `NO` .                                                                                           |

### `role_routine_grants`

This row-filtered view lists the `EXECUTE` privileges granted on all change stream read functions to any database role, including `public` . Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on change stream read functions to the current database role and to roles of which the current database role is a member, not including `public` .

| Column name        | Type                | Description                                                                                   |
|--------------------|---------------------|-----------------------------------------------------------------------------------------------|
| `grantor`          | `character varying` | Not used. Always `NULL` .                                                                     |
| `grantee`          | `character varying` | The name of the role that the privilege was granted to.                                       |
| `specific_catalog` | `character varying` | The database name.                                                                            |
| `specific_schema`  | `character varying` | The name of the routine's schema. For PostgreSQL-dialect databases, the default is `public` . |
| `specific_name`    | `character varying` | The name of the routine. Uniquely identifies the routine even if its name is overloaded.      |
| `routine_catalog`  | `character varying` | The database name.                                                                            |
| `routine_schema`   | `character varying` | The name of the routine's schema. The default is `public` for PostgreSQL-dialect databases.   |
| `routine_name`     | `character varying` | The name of the routine. (Might be duplicated in case of overloading.)                        |
| `privilege_type`   | `character varying` | The type of the privilege granted. Always `EXECUTE` .                                         |
| `is_grantable`     | `character varying` | Not used. Always `NO` .                                                                       |

### `role_table_grants`

This row-filtered view lists all fine-grained access control privileges granted on all tables and views to any database role, including `public` . Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on tables and views to the current database role and to roles of which the current database role is a member, not including `public` .

| **Column name**  | **Type**            | **Description**                                                                                                   |
|------------------|---------------------|-------------------------------------------------------------------------------------------------------------------|
| `grantor`        | `character varying` | Not used. Always `NULL` .                                                                                         |
| `grantee`        | `character varying` | The name of the database role to which this privilege is granted.                                                 |
| `table_catalog`  | `character varying` | The database name.                                                                                                |
| `table_schema`   | `character varying` | The name of the schema that contains the table or view. The default is `public` for PostgreSQL-dialect databases. |
| `table_name`     | `character varying` | The name of the table or view.                                                                                    |
| `privilege_type` | `character varying` | The type of the privilege ( `SELECT` , `INSERT` , `UPDATE` , or `DELETE` ).                                       |
| `is_grantable`   | `character varying` | Not used. Always `NO` .                                                                                           |
| `with_hierarchy` | `character varying` | Not used. Always `NULL` .                                                                                         |

### `routine_options`

This row-filtered view contains one row for each option for each defined change stream read function.

Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only options for change stream read functions on which the `EXECUTE` fine-grained access control privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

<table>
<colgroup>
<col style="width: 33%" />
<col style="width: 33%" />
<col style="width: 33%" />
</colgroup>
<thead>
<tr class="header">
<th>Column name</th>
<th>Type</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><code>specific_catalog</code></td>
<td><code>character varying</code></td>
<td>The database name.</td>
</tr>
<tr class="even">
<td><code>specific_schema</code></td>
<td><code>character varying</code></td>
<td>The name of the routine's schema. The default is <code>public</code> for PostgreSQL-dialect databases.</td>
</tr>
<tr class="odd">
<td><code>specific_name</code></td>
<td><code>character varying</code></td>
<td>The name of the routine. Uniquely identifies the routine even if its name is overloaded.</td>
</tr>
<tr class="even">
<td><code>option_name</code></td>
<td><code>character varying</code></td>
<td>The name of the option.</td>
</tr>
<tr class="odd">
<td><code>option_type</code></td>
<td><code>character varying</code></td>
<td>The data type of the option. The value is one of the following:<br />

<ul>
<li>For built-in types, the name of the data type.</li>
</ul>
<ul>
<li>For arrays, the value <code>ARRAY</code> .</li>
</ul></td>
</tr>
<tr class="even">
<td><code>option_value</code></td>
<td><code>character varying</code></td>
<td>The value of the option.</td>
</tr>
</tbody>
</table>

### `routine_privileges`

This row-filtered view lists all fine-grained access control privileges granted on all change stream read functions to any database role, including `public` . Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on change stream read functions to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name        | Type                | Description                                                                                   |
|--------------------|---------------------|-----------------------------------------------------------------------------------------------|
| `grantor`          | `character varying` | Not used. Always `NULL` .                                                                     |
| `grantee`          | `character varying` | The name of the role that the privilege was granted to.                                       |
| `specific_catalog` | `character varying` | The database name.                                                                            |
| `specific_schema`  | `character varying` | The name of the routine's schema. For PostgreSQL-dialect databases, the default is `public` . |
| `specific_name`    | `character varying` | The name of the routine. Uniquely identifies the routine even if its name is overloaded.      |
| `routine_catalog`  | `character varying` | The database name.                                                                            |
| `routine_schema`   | `character varying` | The name of the routine's schema. The default is `public` .                                   |
| `routine_name`     | `character varying` | The name of the routine. (Might be duplicated if overloaded.)                                 |
| `privilege_type`   | `character varying` | The type of the privilege granted.                                                            |
| `is_grantable`     | `character varying` | Not used. Always `NO` .                                                                       |

### `routines`

This row-filtered view lists all of a database's user defined and change stream read functions. Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only user defined and change stream read functions on which the `EXECUTE` fine-grained access control privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

<table>
<colgroup>
<col style="width: 33%" />
<col style="width: 33%" />
<col style="width: 33%" />
</colgroup>
<thead>
<tr class="header">
<th>Column name</th>
<th>Type</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><code>specific_catalog</code></td>
<td><code>character varying</code></td>
<td>The database name.</td>
</tr>
<tr class="even">
<td><code>specific_schema</code></td>
<td><code>character varying</code></td>
<td>The name of the routine's schema. The default is <code>public</code> for PostgreSQL-dialect databases.</td>
</tr>
<tr class="odd">
<td><code>specific_name</code></td>
<td><code>character varying</code></td>
<td>The name of the routine. Uniquely identifies the routine even if its name is overloaded.</td>
</tr>
<tr class="even">
<td><code>routine_catalog</code></td>
<td><code>character varying</code></td>
<td>The database name.</td>
</tr>
<tr class="odd">
<td><code>routine_schema</code></td>
<td><code>character varying</code></td>
<td>The name of the routine's schema.</td>
</tr>
<tr class="even">
<td><code>routine_name</code></td>
<td><code>character varying</code></td>
<td>The name of the routine. (Might be duplicated in case of overloading.)</td>
</tr>
<tr class="odd">
<td><code>routine_type</code></td>
<td><code>character varying</code></td>
<td>The type of the routine ( <code>FUNCTION</code> or <code>PROCEDURE</code> ). Always <code>FUNCTION</code></td>
</tr>
<tr class="even">
<td><code>module_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>module_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>module_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>udt_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>udt_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>udt_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>data_type</code></td>
<td><code>character varying</code></td>
<td>The return type of the routine. The value is one of the following:<br />

<ul>
<li>For built-in types, the name of the data type.</li>
</ul>
<ul>
<li>For arrays, the value <code>ARRAY</code> .</li>
</ul></td>
</tr>
<tr class="odd">
<td><code>character_maximum_length</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>character_octet_length</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>character_set_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>character_set_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>character_set_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>collation_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>collation_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>collation_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>numeric_precision</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>numeric_precision_radix</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>numeric_scale</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>datetime_precision</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>interval_type</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>interval_precision</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>type_udt_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>type_udt_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>type_udt_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>scope_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>scope_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>scope_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>maximum_cardinality</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>dtd_identifier</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>routine_body</code></td>
<td><code>character varying</code></td>
<td>The type of the routine body ( <code>SQL</code> or <code>EXTERNAL</code> ).</td>
</tr>
<tr class="even">
<td><code>routine_definition</code></td>
<td><code>character varying</code></td>
<td>The definition for the <code>routine_body</code> SQL, empty otherwise.</td>
</tr>
<tr class="odd">
<td><code>external_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>external_language</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>parameter_style</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>is_deterministic</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>sql_data_access</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>is_null_call</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>sql_path</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>schema_level_routine</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>max_dynamic_result_sets</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>is_user_defined_cast</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>is_implicitly_invocable</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>security_type</code></td>
<td><code>character varying</code></td>
<td>The security type of the routine. Only <code>INVOKER</code> is supported.</td>
</tr>
<tr class="odd">
<td><code>to_sql_specific_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>to_sql_specific_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>to_sql_specific_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>as_locator</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>created</code></td>
<td><code>timestamp with time zone</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>last_altered</code></td>
<td><code>timestamp with time zone</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>new_savepoint_level</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>is_udt_dependent</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>result_cast_from_data_type</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>result_cast_as_locator</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>result_cast_char_max_length</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>result_cast_char_octet_length</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>result_cast_char_set_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>result_cast_char_set_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>result_cast_char_set_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>result_cast_collation_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>result_cast_collation_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>result_cast_collation_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>result_cast_numeric_precision</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>result_cast_numeric_precision_radix</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>result_cast_numeric_scale</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>result_cast_datetime_precision</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>result_cast_interval_type</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>result_cast_interval_precision</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>result_cast_type_udt_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>result_cast_type_udt_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>result_cast_type_udt_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>result_cast_scope_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>result_cast_scope_schema</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>result_cast_scope_name</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>result_cast_maximum_cardinality</code></td>
<td><code>bigint</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>result_cast_dtd_identifier</code></td>
<td><code>character varying</code></td>
<td>Not used. The value is always <code>NULL</code> .</td>
</tr>
<tr class="odd">
<td><code>spanner_type</code></td>
<td><code>character varying</code></td>
<td>The detailed return type of the routine. Includes the subtype if an <code>ARRAY</code> is returned.</td>
</tr>
<tr class="even">
<td><code>spanner_determinism</code></td>
<td><code>character varying</code></td>
<td>The user specified determinism of the function ( <code>DETERMINISTIC</code> , <code>NOT_DETERMINISTIC_STABLE</code> , or <code>NOT_DETERMINISTIC_VOLATILE</code> ). Further info can be found at the description for [provolatile](https://www.postgresql.org/docs/current/catalog-pg-proc.html#:~:text=provolatile).</td>
</tr>
</tbody>
</table>

### `schemata`

The `information_schema.schemata` view contains one row for each schema in the current database. The schemas include the information schema and a default schema named `public` .

| Column name                     | Type                      | Description                                                                                                    |
|---------------------------------|---------------------------|----------------------------------------------------------------------------------------------------------------|
| `catalog_name`                  | `character varying`       | The database name.                                                                                             |
| `schema_name`                   | `character varying`       | The name of the schema. This is set to `public` for the default schema and non-empty for named schemas.        |
| `schema_owner`                  | `character varying`       | The name of the owner of the schema.                                                                           |
| `default_character_set_catalog` | `character varying`       | Not used.                                                                                                      |
| `default_character_set_schema`  | `character varying`       | Not used.                                                                                                      |
| `default_character_set_name`    | `character varying`       | Not used.                                                                                                      |
| `sql_path`                      | `character varying`       | Not used.                                                                                                      |
| `effective_timestamp`           | `timestamp with timezone` | The timestamp at which all the data in this schema became effective. This is used only for the default schema. |

### `sequences`

The `information_schema.sequences` view contains the `sequences` metadata.

| Column name               | Type                | Description                                                                                   |
|---------------------------|---------------------|-----------------------------------------------------------------------------------------------|
| `sequence_catalog`        | `character varying` | The database name.                                                                            |
| `sequence_schema`         | `character varying` | The name of the sequence's schema. The default is `public` for a PostgreSQL-dialect database. |
| `sequence_name`           | `character varying` | The name of the sequence.                                                                     |
| `data_type`               | `character varying` | Sequence only supports `int8` .                                                               |
| `numeric_precision`       | `bigint`            | Not used. The value is always \`NULL\`.                                                       |
| `numeric_precision_radix` | `bigint`            | Not used. The value is always \`NULL\`.                                                       |
| `numeric_scale`           | `bigint`            | Not used. The value is always \`NULL\`.                                                       |
| `start_value`             | `bigint`            | Not used. The value is always \`NULL\`.                                                       |
| `minimum_value`           | `bigint`            | Not used. The value is always \`NULL\`.                                                       |
| `maximum_value`           | `bigint`            | Not used. The value is always \`NULL\`.                                                       |
| `increment`               | `bigint`            | Not used. The value is always \`NULL\`.                                                       |
| `cycle_option`            | `character varying` | The only option that `sequence` accepts is `no` .                                             |
| `sequence_kind`           | `character varying` | The kind of sequence. `bit_reversed_positive` is the only acceptable value.                   |
| `counter_start_value`     | `bigint`            | Starting value of the sequence counter.                                                       |
| `skip_range_min`          | `bigint`            | The minimum value in the skipped range. This value is `NULL` if not set.                      |
| `skip_range_max`          | `bigint`            | The maximum value in the skipped range. This value is `NULL` if not set.                      |

### `spanner_statistics`

This table lists the available query optimizer statistics packages.

| Column name    | Type                | Description                                                                                                                                                                                                                                                                                        |
|----------------|---------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `catalog_name` | `character varying` | The database name.                                                                                                                                                                                                                                                                                 |
| `schema_name`  | `character varying` | The name of the schema. The default schema value is `public` .                                                                                                                                                                                                                                     |
| `package_name` | `character varying` | The name of the statistics package.                                                                                                                                                                                                                                                                |
| `allow_gc`     | `character varying` | Whether the statistics package is exempted from garbage collection. In accordance with the SQL standard, the string is either `YES` or `NO` , rather than a Boolean value. This attribute must be set to `NO` before you can reference the statistics package in a hint or through the client API. |

### `table_constraints`

This view contains all constraints belonging to tables that the current user has access to (other than `SELECT` ).

<table>
<colgroup>
<col style="width: 33%" />
<col style="width: 33%" />
<col style="width: 33%" />
</colgroup>
<tbody>
<tr class="header">
<th>Column name</th>
<th>Type</th>
<th>Description</th>
</tr>

<tr class="odd">
<td><code>constraint_catalog</code></td>
<td><code>character varying</code></td>
<td>The database name.</td>
</tr>
<tr class="even">
<td><code>constraint_schema</code></td>
<td><code>character varying</code></td>
<td>The name of the schema that contains the constraint.</td>
</tr>
<tr class="odd">
<td><code>constraint_name</code></td>
<td><code>character varying</code></td>
<td>The name of the constraint.</td>
</tr>
<tr class="even">
<td><code>table_catalog</code></td>
<td><code>character varying</code></td>
<td>The database name.</td>
</tr>
<tr class="odd">
<td><code>table_schema</code></td>
<td><code>character varying</code></td>
<td>The name of schema that contains the table associated with the constraint.</td>
</tr>
<tr class="even">
<td><code>table_name</code></td>
<td><code>character varying</code></td>
<td>The name of the table.</td>
</tr>
<tr class="odd">
<td><code>constraint_type</code></td>
<td><code>character varying</code></td>
<td>The type of the constraint. Possible values are:
<ul>
<li><code>CHECK</code></li>
<li><code>FOREIGN KEY</code></li>
<li><code>PLACEMENT KEY</code></li>
<li><code>PRIMARY KEY</code></li>
<li><code>UNIQUE</code></li>
</ul></td>
</tr>
<tr class="even">
<td><code>is_deferrable</code></td>
<td><code>character varying</code></td>
<td>The value is always <code>NO</code> .</td>
</tr>
<tr class="odd">
<td><code>initially_deferred</code></td>
<td><code>character varying</code></td>
<td>The value is always <code>NO</code> .</td>
</tr>
<tr class="even">
<td><code>enforced</code></td>
<td><code>character varying</code></td>
<td>Whether the constraint is enforced. If a constraint is enforced, (after it reaches a certain state), it's validated both at write time and by a background integrity verifier. In accordance with the SQL standard, the string is either <code>YES</code> or <code>NO</code> , rather than a Boolean value.</td>
</tr>
</tbody>
</table>

### `table_privileges`

This row-filtered view lists all fine-grained access control privileges granted on all tables and views to any database role, including `public` . Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on tables and views to the current database role, to roles of which the current database role is a member, or to `public` .

| **Column name**       | **Type**            | **Description**                                                                                                   |
|-----------------------|---------------------|-------------------------------------------------------------------------------------------------------------------|
| `grantor`             | `character varying` | Not used. Always `NULL` .                                                                                         |
| `grantee`             | `character varying` | The name of the database role to which this privilege is granted.                                                 |
| `table_catalog`       | `character varying` | The database name.                                                                                                |
| `table_schema`        | `character varying` | The name of the schema that contains the table or view. The default is `public` for PostgreSQL-dialect databases. |
| `table_name`          | `character varying` | The name of the table or view.                                                                                    |
| `privilege_type`      | `character varying` | The type of the privilege ( `SELECT` , `INSERT` , `UPDATE` , or `DELETE` ).                                       |
| `is_grantable`        | `character varying` | Not used. Always `NO` .                                                                                           |
| `that have_hierarchy` | `character varying` | Not used. Always `NULL` .                                                                                         |

### `tables`

This row-filtered view lists all the tables and view that are in the current database. Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all tables and views. All other principals can see only tables that meet either of the following requirements:

- The `SELECT` , `INSERT` , `UPDATE` , or `DELETE` fine-grained access control privileges are granted on the table to the current database role, to roles of which the current database role is a member, or to `public` .
- The `SELECT` , `INSERT` , or `UPDATE` privileges are granted on any table column to the current database role, to roles of which the current database role is a member, or to `public` .

<table>
<colgroup>
<col style="width: 33%" />
<col style="width: 33%" />
<col style="width: 33%" />
</colgroup>
<tbody>
<tr class="header">
<th>Column name</th>
<th>Type</th>
<th>Description</th>
</tr>

<tr class="odd">
<td><code>table_catalog</code></td>
<td><code>character varying</code></td>
<td>The database name.</td>
</tr>
<tr class="even">
<td><code>table_schema</code></td>
<td><code>character varying</code></td>
<td>The name of the schema that contains the table or view.</td>
</tr>
<tr class="odd">
<td><code>table_name</code></td>
<td><code>character varying</code></td>
<td>The name of the table, view, or synonym.</td>
</tr>
<tr class="even">
<td><code>table_type</code></td>
<td><code>character varying</code></td>
<td>The table type. Possible values include 'BASE TABLE', 'VIEW', or 'SYNONYM'.</td>
</tr>
<tr class="odd">
<td><code>self_referencing_column_name</code></td>
<td><code>character varying</code></td>
<td>Not used.</td>
</tr>
<tr class="even">
<td><code>reference_generation</code></td>
<td><code>character varying</code></td>
<td>Not used.</td>
</tr>
<tr class="odd">
<td><code>user_defined_type_catalog</code></td>
<td><code>character varying</code></td>
<td>Not used.</td>
</tr>
<tr class="even">
<td><code>user_defined_type_schema</code></td>
<td><code>character varying</code></td>
<td>Not used.</td>
</tr>
<tr class="odd">
<td><code>user_defined_type_name</code></td>
<td><code>character varying</code></td>
<td>Not used.</td>
</tr>
<tr class="even">
<td><code>is_insertable_into</code></td>
<td><code>character varying</code></td>
<td>Not used.</td>
</tr>
<tr class="odd">
<td><code>is_typed</code></td>
<td><code>character varying</code></td>
<td>Not used.</td>
</tr>
<tr class="even">
<td><code>commit_action</code></td>
<td><code>character varying</code></td>
<td>Not used.</td>
</tr>
<tr class="odd">
<td><code>parent_table_name</code></td>
<td><code>character varying</code></td>
<td>The name of the parent table if this table is interleaved, or <code>NULL</code> .</td>
</tr>
<tr class="even">
<td><code>on_delete_action</code></td>
<td><code>character varying</code></td>
<td>This is set to <code>CASCADE</code> or <code>NO ACTION</code> for interleaved tables, and <code>NULL</code> otherwise. See <a href="https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#table_statements">TABLE statements</a> for more information.</td>
</tr>
<tr class="odd">
<td><code>spanner_state</code></td>
<td><code>character varying</code></td>
<td>The current creation state of the table.<br />
A table can go through multiple states during creation, if bulk operations are involved, for example, when the table is created with a foreign key that requires backfilling of its referenced index. Possible states are:
<ul>
<li><code>ADDING_FOREIGN_KEY</code> : Adding the table's foreign keys</li>
<li><code>WAITING_FOR_COMMIT</code> : Finalizing the schema change</li>
<li><code>COMMITTED</code> : The schema change to create the table has been committed. You cannot write to the table until the change is committed.</li>
<li><code>NULL</code> : Tables or views that are not base tables.</li>
</ul></td>
</tr>
<tr class="even">
<td><code>interleave_type</code></td>
<td><code>character varying</code></td>
<td>Whether there exists a parent-child relationship between this table and the table it is interleaved in. Possible values are:
<ul>
<li><code>IN</code> : An <code>INTERLEAVE IN</code> table that has no parent-child relationship. A row in this table can exist regardless of the existence of its parent table row.</li>
<li><code>IN PARENT</code> : An <code>INTERLEAVE IN PARENT</code> table that has a parent-child relationship. A row in this table requires the existence of its parent table row.</li>
</ul></td>
</tr>
<tr class="odd">
<td><code>row_deletion_policy_expression</code></td>
<td><code>character varying</code></td>
<td>An string that contains the expression text that defines the <code>ROW DELETION POLICY</code> .</td>
</tr>
</tbody>
</table>

### `table_synonyms`

This table lists synonym information for the table.

| Column name          | Type     | Description                               |
|----------------------|----------|-------------------------------------------|
| `CATALOG`            | `STRING` | Name of the catalog containing the table. |
| `SCHEMA`             | `STRING` | Name of the schema containing the table.  |
| `TABLE_NAME`         | `STRING` | Name of the table.                        |
| `SYNONYM_CATALOG`    | `STRING` | The name of the catalog for the synonym.  |
| `SYNONYM_SCHEMA`     | `STRING` | The name of the schema for the synonym.   |
| `SYNONYM_TABLE_NAME` | `STRING` | The name of the table for the synonym.    |

### `views`

This row-filtered view lists all views in the current database. Principals that have database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all views. All other principals can see only views that have the `SELECT` fine-grained access control privilege granted on them to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name                  | Type                | Description                                                                                                                                                |
|------------------------------|---------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `table_catalog`              | `character varying` | The database name.                                                                                                                                         |
| `table_schema`               | `character varying` | The name of the schema. The default value is `public` .                                                                                                    |
| `table_name`                 | `character varying` | The name of the view.                                                                                                                                      |
| `view_definition`            | `character varying` | The SQL text of the query that defines the view.                                                                                                           |
| `check_option`               | `character varying` | Not used.                                                                                                                                                  |
| `is_updatable`               | `character varying` | Not used.                                                                                                                                                  |
| `is_insertable_into`         | `character varying` | Not used.                                                                                                                                                  |
| `is_trigger_updatable`       | `character varying` | Not used.                                                                                                                                                  |
| `is_trigger_deletable`       | `character varying` | Not used.                                                                                                                                                  |
| `is_trigger_insertable_into` | `character varying` | Not used.                                                                                                                                                  |
| `security_type`              | `character varying` | The security type of the view. Either `INVOKER` or `DEFINER` . For more information, see [About views](https://docs.cloud.google.com/spanner/docs/views) . |

## Examples

Return information about each table in the user's schema:

```
SELECT
  t.table_schema,
  t.table_catalog,
  t.table_name,
  t.parent_table_name
FROM
  information_schema.tables AS t
WHERE
  t.table_schema NOT IN ('pg_catalog', 'information_schema', 'SPANNER_SYS')
  AND t.table_type = 'BASE TABLE'
ORDER BY
  t.table_catalog,
  t.table_schema,
  t.table_name
```

Return the name of all tables and views in the `information_schema` for PostgreSQL-dialect databases:

```
SELECT table_name
FROM information_schema.tables
WHERE table_schema = "information_schema"
```

Return information about columns in the user table `my_table` in the default schema:

```
SELECT
  t.ordinal_position,
  t.column_name,
  t.data_type,
  t.spanner_type,
  t.is_nullable
FROM
  information_schema.columns AS t
WHERE
  t.table_schema = 'public'
  AND
  t.table_name = 'my_table'
ORDER BY
  t.ordinal_position
```

Return information about each index in the default schema in the current database: \`\`\`postgresql SELECT t.table_name, t.index_name, t.parent_table_name FROM information_schema.indexes AS t WHERE t.table_schema = 'public' AND t.index_type != 'PRIMARY_KEY' ORDER BY t.table_schema, t.table_name, t.index_name

Return columns that use non-default options in the default schema:

```
SELECT
  t.table_name,
  t.column_name,
  t.option_type,
  t.option_value,
  t.option_name
FROM
  information_schema.column_options AS t
WHERE
  t.table_schema = 'public'
ORDER BY
  t.table_schema,
  t.table_name,
  t.column_name,
  t.option_name
```

Return the current optimizer-related database options:

```
SELECT
  s.option_name,
  s.option_value
FROM
  information_schema.database_options s
WHERE
  s.schema_name='public'
  AND s.option_name IN ('optimizer_version',
    'optimizer_statistics_package')
```

Return all available statistics packages:

```postgresql
SELECT *
FROM information_schema.spanner_statistics;
```## What's
next {: #whats-next}

+   Learn about available [Introspection tools](/spanner/docs/introspection) to
help you investigate database issues. 
  

  
    
  
```
