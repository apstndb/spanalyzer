---
name: documents/docs.cloud.google.com/spanner/docs/information-schema
uri: https://docs.cloud.google.com/spanner/docs/information-schema
title: Information schema for GoogleSQL-dialect databases
description: Query the INFORMATION_SCHEMA in Spanner using SQL to retrieve database schema metadata, including tables, columns, and constraints.
data_source: docs.cloud.google.com
---

The information schema is a built-in schema that's common to every Spanner database. You can run SQL queries against tables in the `INFORMATION_SCHEMA` to fetch schema metadata for a database.

For example, the following query fetches the names of all user-defined tables in a database:

```
  SELECT
    table_schema,
    table_name
  FROM
    information_schema.tables
  WHERE
    table_schema NOT IN ('information_schema', 'SPANNER_SYS')
    AND table_type = 'BASE TABLE'
```

Fine-grained access control users see filtered results for some `INFORMATION_SCHEMA` tables depending on their database role. For more information, see [About fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) .

## Usage

`INFORMATION_SCHEMA` tables are available only through SQL interfaces, for example:

- The `executeQuery` API
- The `gcloud spanner databases execute-sql` command
- The **Spanner Studio** page of a database in the Google Cloud console)

Other single read methods don't support `INFORMATION_SCHEMA` .

Some additional `INFORMATION_SCHEMA` usage notes:

- Queries against the `INFORMATION_SCHEMA` can be used in a [read-only transaction](https://docs.cloud.google.com/spanner/docs/transactions#read-only_transactions) , but not in a [read-write transaction](https://docs.cloud.google.com/spanner/docs/transactions#read-write_transactions) .
- Queries against the `INFORMATION_SCHEMA` can use strong, bounded staleness, or exact staleness [timestamp bounds](https://docs.cloud.google.com/spanner/docs/timestamp-bounds) .
- If you are using a PostgreSQL-dialect database, see [Information schema for PostgreSQL-dialect databases](https://docs.cloud.google.com/spanner/docs/information-schema-pg) .
- If you are a [fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) user, `INFORMATION_SCHEMA` tables are filtered to only show schema elements that you have access to.

## Row filtering in information_schema tables

Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` [system role](https://docs.cloud.google.com/spanner/docs/fgac-system-roles) (or to members of that role) can see all rows in all information_schema tables. For other principals, for some tables, rows are filtered based on the current database role. The table and view descriptions in the following sections indicate how row filtering is applied for each table and view.

## Tables in the INFORMATION_SCHEMA

The following sections describe the tables in the `INFORMATION_SCHEMA` for GoogleSQL-dialect databases.

### `SCHEMATA`

The `INFORMATION_SCHEMA.SCHEMATA` table lists the schemas in the database. These include the information schema and the named schemas, which contain the tables you define.

| Column name    | Type     | Description                                                                                                                                                                                  |
|----------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `CATALOG_NAME` | `STRING` | The name of the catalog. This column exists for compatibility with SQL-standard information schema tables. This column is always an empty string.                                            |
| `SCHEMA_NAME`  | `STRING` | The name of the schema. This is a name for named schemas or \`\` for the default schema.                                                                                                     |
| `PROTO_BUNDLE` | `STRING` | If the database contains proto bundle statements, this column provides information about the proto bundle used in the schema. This column is NULL if no proto bundle exists in the database. |

### `DATABASE_OPTIONS`

This table lists the options that are set on the database.

| Column name    | Type     | Description                                         |
|----------------|----------|-----------------------------------------------------|
| `CATALOG_NAME` | `STRING` | The name of the catalog. Always an empty string.    |
| `SCHEMA_NAME`  | `STRING` | The name of the schema. An empty string if unnamed. |
| `OPTION_NAME`  | `STRING` | The name of the database option.                    |
| `OPTION_TYPE`  | `STRING` | The data type of the database option.               |
| `OPTION_VALUE` | `STRING` | The database option value.                          |

### `PLACEMENTS`

This table lists the placements in the database.

| Column name      | Type     | Description                                                              |
|------------------|----------|--------------------------------------------------------------------------|
| `PLACEMENT_NAME` | `STRING` | The name of the placement.                                               |
| `IS_DEFAULT`     | `BOOL`   | A boolean that indicates whether the placement is the default placement. |

### `PLACEMENT_OPTIONS`

For each placement, this table lists the options that are set on the placement in the `OPTIONS` clause of the [`CREATE PLACEMENT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create-placement) statement.

The valid values for `OPTION_NAME` include:

- `instance_partition`
- `default_leader`

| Column name      | Type     | Description                                                                                                                                                                   |
|------------------|----------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `PLACEMENT_NAME` | `STRING` | The name of the placement.                                                                                                                                                    |
| `OPTION_NAME`    | `STRING` | The name of the placement option.                                                                                                                                             |
| `OPTION_TYPE`    | `STRING` | The data type of the placement option. For both options, this is `STRING(MAX)` .                                                                                              |
| `OPTION_VALUE`   | `STRING` | The value of the placement option. For `instance_partition` , this is the name of the instance partition. For `default_leader` , it is the name of the default leader region. |

### `LOCALITY_GROUP_OPTIONS`

For each locality group, this table lists the name and options that are set on the locality group in the `OPTIONS` clause of the [`CREATE LOCALITY GROUP`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create-locality-group) statement.

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
<td><code>LOCALITY_GROUP_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the locality group.</td>
</tr>
<tr class="even">
<td><code>OPTION_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the locality group option. The valid options are:
<ul>
<li><code>STORAGE</code> : defines the storage type for the locality group.</li>
<li><code>SSD_TO_HDD_SPILL_TIMESPAN</code> : defines how long data is stored in SSD storage before it moves to HDD storage.</li>
</ul></td>
</tr>
<tr class="odd">
<td><code>OPTION_VALUE</code></td>
<td><code>STRING</code></td>
<td>The value of the locality group option. For <code>STORAGE</code> , this is either <code>ssd</code> or <code>hdd</code> . For <code>SSD_TO_HDD_SPILL_TIMESPAN</code> , this is the amount of time that data must be stored in SSD before it's moved to HDD storage. For example, <code>10d</code> is 10 days. The minimum amount of time you can set is one hour.</td>
</tr>
</tbody>
</table>

### `TABLES`

This row-filtered table lists the tables and views in the database. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only tables that meet either of the following requirements:

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
<td><code>TABLE_CATALOG</code></td>
<td><code>STRING</code></td>
<td>Not used. Always an empty string.</td>
</tr>
<tr class="even">
<td><code>TABLE_SCHEMA</code></td>
<td><code>STRING</code></td>
<td>The schema name of the table. The name is empty for the default schema, and it contains a value for other schemas. This column always contains a value.</td>
</tr>
<tr class="odd">
<td><code>TABLE_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the table, view, or synonym.</td>
</tr>
<tr class="even">
<td><code>TABLE_TYPE</code></td>
<td><code>STRING</code></td>
<td>The type of the table. For tables it has the value <code>BASE TABLE</code> ; for views it has the value <code>VIEW</code> ; for synonyms, it has the value <code>SYNONYM</code> .</td>
</tr>
<tr class="odd">
<td><code>PARENT_TABLE_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the parent table if this table is interleaved, and <code>NULL</code> otherwise.</td>
</tr>
<tr class="even">
<td><code>ON_DELETE_ACTION</code></td>
<td><code>STRING</code></td>
<td>This is set to <code>CASCADE</code> or <code>NO ACTION</code> for interleaved tables, and <code>NULL</code> otherwise. See <a href="https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#table_statements">TABLE statements</a> for more information.</td>
</tr>
<tr class="odd">
<td><code>SPANNER_STATE</code></td>
<td><code>STRING</code></td>
<td>A table can go through multiple states during creation, if bulk operations are involved. For example, when the table is created with a foreign key that requires backfilling of its indexes. Possible states are:
<ul>
<li><code>ADDING_FOREIGN_KEY</code> : Adding the table's foreign keys.</li>
<li><code>WAITING_FOR_COMMIT</code> : Finalizing the schema change.</li>
<li><code>COMMITTED</code> : The schema change to create the table has been committed. You can't write to the table until the change is committed.</li>
</ul></td>
</tr>
<tr class="even">
<td><code>INTERLEAVE_TYPE</code></td>
<td><code>STRING</code></td>
<td>The expression text that indicates whether there is a parent-child relationship between this table and the table it is interleaved in. Possible values are:
<ul>
<li><code>IN</code> : The table doesn't have a parent-child relationship. A row in this table can exist regardless of the existence of its parent table row.</li>
<li><code>IN PARENT</code> : The table has a parent-child relationship. A row in this table requires the existence of its parent table row.</li>
<li>An empty string indicates that this table has no interleaving relationships.</li>
</ul></td>
</tr>
<tr class="odd">
<td><code>ROW_DELETION_POLICY_EXPRESSION</code></td>
<td><code>STRING</code></td>
<td>The expression text that defines the <a href="https://docs.cloud.google.com/spanner/docs/ttl">row deletion policy</a> of the table. For example, <code>OLDER_THAN(CreatedAt, INTERVAL 1 DAY)</code> or <code>OLDER_THAN(ExpiredDate, INTERVAL 0 DAY)</code> .</td>
</tr>
</tbody>
</table>

### `COLUMNS`

This row-filtered table lists the columns in a table. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only columns that meet either of the following requirements:

- Any of the `SELECT` , `INSERT` , or `UPDATE` fine-grained access control privileges are directly granted on the column to the current database role, to roles of which the current database role is a member, or to `public` .
- Any of the `SELECT` , `INSERT` , or `UPDATE` fine-grained access control privileges are granted on the table that contains the column to the current database role, to roles of which the current database role is a member, or to `public` .

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
<td><code>TABLE_CATALOG</code></td>
<td><code>STRING</code></td>
<td>Not used. Always an empty string.</td>
</tr>
<tr class="even">
<td><code>TABLE_SCHEMA</code></td>
<td><code>STRING</code></td>
<td>The schema name of the column's table. The name is empty for the default schema, and it contains a value for other schemas. This column always contains a value.</td>
</tr>
<tr class="odd">
<td><code>TABLE_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the table.</td>
</tr>
<tr class="even">
<td><code>COLUMN_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the column.</td>
</tr>
<tr class="odd">
<td><code>ORDINAL_POSITION</code></td>
<td><code>INT64</code></td>
<td>The ordinal position of the column in the table, starting with a value of 1.</td>
</tr>
<tr class="even">
<td><code>COLUMN_DEFAULT</code></td>
<td><code>STRING</code></td>
<td><p>A string representation of the SQL expression for the default value of the column. <code>NULL</code> if the column has no default value.</p>
<p><em>Note:</em> Prior to March 2022, <code>COLUMN_DEFAULT</code> used type <code>BYTES</code> .</p></td>
</tr>
<tr class="odd">
<td><code>DATA_TYPE</code></td>
<td><code>STRING</code></td>
<td>Included to satisfy the SQL standard. Always <code>NULL</code> . See the column <code>SPANNER_TYPE</code></td>
</tr>
<tr class="even">
<td><code>IS_NULLABLE</code></td>
<td><code>STRING</code></td>
<td>A string that indicates whether the column is nullable. In accordance with the SQL standard, the string is either <code>YES</code> or <code>NO</code> , rather than a Boolean value.</td>
</tr>
<tr class="odd">
<td><code>SPANNER_TYPE</code></td>
<td><code>STRING</code></td>
<td>The <a href="https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-types#allowable-types">data type</a> of the column.</td>
</tr>
<tr class="even">
<td><code>IS_GENERATED</code></td>
<td><code>STRING</code></td>
<td>A string that indicates whether the column is generated. The string is either <code>ALWAYS</code> for a generated column or <code>NEVER</code> for a non-generated column.</td>
</tr>
<tr class="odd">
<td><code>GENERATION_EXPRESSION</code></td>
<td><code>STRING</code></td>
<td>A string representing the SQL expression of a generated column. <code>NULL</code> if the column is not a generated column.</td>
</tr>
<tr class="even">
<td><code>IS_STORED</code></td>
<td><code>STRING</code></td>
<td>A string that indicates whether the generated column is stored. The string is always <code>YES</code> for generated columns, and <code>NULL</code> for non-generated columns.</td>
</tr>
<tr class="odd">
<td><code>IS_HIDDEN</code></td>
<td><code>STRING</code></td>
<td>A string that is set to <code>TRUE</code> if the column doesn't appear in a <code>SELECT *</code> query, and is set to <code>FALSE</code> otherwise. If the column is hidden, you can still select it using its name (for example, <code>SELECT Id, Name, ColHidden FROM TableWithHiddenColumn</code> ).</td>
</tr>
<tr class="even">
<td><code>SPANNER_STATE</code></td>
<td><code>STRING</code></td>
<td>The current state of the column. A new stored generated column added to an existing table may go through multiple user-observable states before it is fully usable. Possible values are:
<ul>
<li><code>WRITE_ONLY</code> : The column is being backfilled. No read is allowed.</li>
<li><code>COMMITTED</code> : The column is fully usable.</li>
</ul></td>
</tr>
<tr class="odd">
<td><code>IS_IDENTITY</code></td>
<td><code>STRING</code></td>
<td>A string that is set to <code>YES</code> if the generated column is an identity column, and <code>NO</code> otherwise.</td>
</tr>
<tr class="even">
<td><code>IDENTITY_GENERATION</code></td>
<td><code>STRING</code></td>
<td>A string that specifies whether the column permits only generated, and not custom user-inserted, values.
<ul>
<li><code>BY DEFAULT</code> : The default value. <code>BY DEFAULT</code> specifies that the column uses generated values if user-inserted values aren't provided.</li>
<li><code>ALWAYS</code> : The column permits only generated, and not custom user-inserted, values.</li>
</ul></td>
</tr>
<tr class="odd">
<td><code>IDENTITY_KIND</code></td>
<td><code>STRING</code></td>
<td>Always <code>BIT_REVERSED_POSITITVE_SEQUENCE</code> . Only bit-reversed positive sequences are supported.</td>
</tr>
<tr class="even">
<td><code>IDENTITY_START_WITH_COUNTER</code></td>
<td><code>STRING</code></td>
<td>The start value of the internal counter before transforming. For example, the start value before bit-reversing.</td>
</tr>
<tr class="odd">
<td><code>IDENTITY_SKIP_RANGE_MIN</code></td>
<td><code>STRING</code></td>
<td>The minimum value of a skipped range after transforming.</td>
</tr>
<tr class="even">
<td><code>IDENTITY_SKIP_RANGE_MAX</code></td>
<td><code>STRING</code></td>
<td>The maximum value of a skipped range after transforming.</td>
</tr>
<tr class="odd">
<td><code>ON_UPDATE_EXPRESSION</code></td>
<td><code>STRING</code></td>
<td>A STRING with the SQL representation of the `ON UPDATE` expression for a column. If the column does not have an `ON UPDATE` value, the value is `NULL`.</td>
</tr>
</tbody>
</table>

### `COLUMN_PRIVILEGES`

This row-filtered table lists all the privileges granted at the column-level to any [database role](https://docs.cloud.google.com/spanner/docs/information-schema#roles) , including `public` . Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see privileges only for columns that meet either of the following requirements:

- Any of the `SELECT` , `INSERT` , or `UPDATE` fine-grained access control privileges are directly granted on the column to the current database role, to roles of which the current database role is a member, or to `public` .
- Any of the `SELECT` , `INSERT` , or `UPDATE` fine-grained access control privileges are granted on the table that contains the column to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name      | Type     | Description                                                                                                                                                      |
|------------------|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `TABLE_CATALOG`  | `STRING` | Not used. Always an empty string.                                                                                                                                |
| `TABLE_SCHEMA`   | `STRING` | The schema name of the column's table. The name is empty for the default schema, and it contains a value for other schemas. This column always contains a value. |
| `TABLE_NAME`     | `STRING` | The name of the table that contains the privileged column.                                                                                                       |
| `COLUMN_NAME`    | `STRING` | The name of the privileged column.                                                                                                                               |
| `PRIVILEGE_TYPE` | `STRING` | `SELECT` , `INSERT` , `UPDATE`                                                                                                                                   |
| `GRANTEE`        | `STRING` | The name of the database role to which this privilege is granted.                                                                                                |

### `TABLE_PRIVILEGES`

This row-filtered table lists all the privileges granted at the table-level to [database roles](https://docs.cloud.google.com/spanner/docs/information-schema#roles) , including `public` . Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see privileges only for tables on which any of the `SELECT` , `INSERT` , `UPDATE` , or `DELETE` fine-grained access control privileges are granted to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name      | Type     | Description                                                                                                                                             |
|------------------|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| `TABLE_CATALOG`  | `STRING` | Not used. Always an empty string.                                                                                                                       |
| `TABLE_SCHEMA`   | `STRING` | The schema name of the table. The name is empty for the default schema, and it contains a value for other schemas. This column always contains a value. |
| `TABLE_NAME`     | `STRING` | The name of the table on which fine-grained access control privileges are granted.                                                                      |
| `PRIVILEGE_TYPE` | `STRING` | One of `SELECT` , `INSERT` , `UPDATE` , and `DELETE`                                                                                                    |
| `GRANTEE`        | `STRING` | The name of the database role to which this privilege is granted.                                                                                       |

### `TABLE_CONSTRAINTS`

This table contains one row for each constraint defined for the tables in the database.

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
<td><code>CONSTRAINT_CATALOG</code></td>
<td><code>STRING</code></td>
<td>Always an emptry string.</td>
</tr>
<tr class="even">
<td><code>CONSTRAINT_SCHEMA</code></td>
<td><code>STRING</code></td>
<td>The name of the constraint's schema. An empty string if unnamed.</td>
</tr>
<tr class="odd">
<td><code>CONSTRAINT_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the constraint.</td>
</tr>
<tr class="even">
<td><code>TABLE_CATALOG</code></td>
<td><code>STRING</code></td>
<td>The name of constrained table's catalog. Always an empty string.</td>
</tr>
<tr class="odd">
<td><code>TABLE_SCHEMA</code></td>
<td><code>STRING</code></td>
<td>The schema name of the constrained table. The name is empty for the default schema, and it contains a value for other schemas. This column always contains a value.</td>
</tr>
<tr class="even">
<td><code>TABLE_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the constrained table.</td>
</tr>
<tr class="odd">
<td><code>CONSTRAINT_TYPE</code></td>
<td><code>STRING</code></td>
<td>The type of the constraint. Possible values are:
<ul>
<li><code>PRIMARY KEY</code></li>
<li><code>FOREIGN KEY</code></li>
<li><code>PLACEMENT KEY</code></li>
<li><code>CHECK</code></li>
<li><code>UNIQUE</code></li>
</ul></td>
</tr>
<tr class="even">
<td><code>IS_DEFERRABLE</code></td>
<td><code>STRING</code></td>
<td>Always <code>NO</code> .</td>
</tr>
<tr class="odd">
<td><code>INITIALLY_DEFERRED</code></td>
<td><code>STRING</code></td>
<td>Always <code>NO</code> .</td>
</tr>
<tr class="even">
<td><code>ENFORCED</code></td>
<td><code>STRING</code></td>
<td><code>NO</code> if the constraint is an <a href="https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#informational-foreign-keys">informational ( <code>NOT ENFORCED</code> ) foreign key</a> . <code>YES</code> for enforced foreign keys or any other constraint type.</td>
</tr>
</tbody>
</table>

### `CONSTRAINT_TABLE_USAGE`

This table lists tables that define or are used by constraints. Includes tables that define `PRIMARY KEY` and `UNIQUE` constraints. Also includes the referenced tables of `FOREIGN KEY` definitions.

| Column name          | Type     | Description                                                                                                                                                         |
|----------------------|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `TABLE_CATALOG`      | `STRING` | The name of the constrained table's catalog. Always an empty string.                                                                                                |
| `TABLE_SCHEMA`       | `STRING` | The schema name of the constrained table. The name is empty for the default schema, and it contains a value for other schemas. This column always contains a value. |
| `TABLE_NAME`         | `STRING` | The name of the constrained table.                                                                                                                                  |
| `CONSTRAINT_CATALOG` | `STRING` | The name of the constraint's catalog. Always an empty string.                                                                                                       |
| `CONSTRAINT_SCHEMA`  | `STRING` | The name of the constraint's schema. An empty string if unnamed.                                                                                                    |
| `CONSTRAINT_NAME`    | `STRING` | The name of the constraint.                                                                                                                                         |

### `REFERENTIAL_CONSTRAINTS`

This table contains one row about each `FOREIGN KEY` constraint.

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
<td><code>CONSTRAINT_CATALOG</code></td>
<td><code>STRING</code></td>
<td>The name of the FOREIGN KEY's catalog. Always an empty string.</td>
</tr>
<tr class="even">
<td><code>CONSTRAINT_SCHEMA</code></td>
<td><code>STRING</code></td>
<td>The name of the FOREIGN KEY's schema. An empty string if unnamed.</td>
</tr>
<tr class="odd">
<td><code>CONSTRAINT_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the FOREIGN KEY.</td>
</tr>
<tr class="even">
<td><code>UNIQUE_CONSTRAINT_CATALOG</code></td>
<td><code>STRING</code></td>
<td>The catalog name of the PRIMARY KEY or UNIQUE constraint the FOREIGN KEY references. Always an empty string.</td>
</tr>
<tr class="odd">
<td><code>UNIQUE_CONSTRAINT_SCHEMA</code></td>
<td><code>STRING</code></td>
<td>The schema name of the PRIMARY KEY or UNIQUE constraint the FOREIGN KEY references. An empty string if unnamed.</td>
</tr>
<tr class="even">
<td><code>UNIQUE_CONSTRAINT_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the PRIMARY KEY or UNIQUE constraint the FOREIGN KEY references.</td>
</tr>
<tr class="odd">
<td><code>MATCH_OPTION</code></td>
<td><code>STRING</code></td>
<td>Always <code>SIMPLE</code> .</td>
</tr>
<tr class="even">
<td><code>UPDATE_RULE</code></td>
<td><code>STRING</code></td>
<td>Always <code>NO ACTION</code> .</td>
</tr>
<tr class="odd">
<td><code>DELETE_RULE</code></td>
<td><code>STRING</code></td>
<td>Either <code>CASCADE</code> or <code>NO ACTION</code> .</td>
</tr>
<tr class="even">
<td><code>SPANNER_STATE</code></td>
<td><code>STRING</code></td>
<td>The current state of the foreign key. Spanner does not begin enforcing the constraint until the foreign key's backing indexes are created and backfilled. Once the indexes are ready, Spanner begins enforcing the constraint for new transactions while it validates the existing data. Possible values and the states they represent are:
<ul>
<li><code>BACKFILLING_INDEXES</code> : indexes are being backfilled.</li>
<li><code>VALIDATING_DATA</code> : existing data and new writes are being validated.</li>
<li><code>WAITING_FOR_COMMIT</code> : the foreign key bulk operations have completed successfully, or none were needed, but the foreign key is still pending.</li>
<li><code>COMMITTED</code> : the schema change was committed.</li>
</ul></td>
</tr>
</tbody>
</table>

### `CHECK_CONSTRAINTS`

The `information_schema.CHECK_CONSTRAINTS` table contains one row about each `CHECK` constraint defined by either the `CHECK` or the `NOT NULL` keyword.

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
<td><code>CONSTRAINT_CATALOG</code></td>
<td><code>STRING</code></td>
<td>The name of the constraint's catalog. This column is never null, but always an empty string.</td>
</tr>
<tr class="even">
<td><code>CONSTRAINT_SCHEMA</code></td>
<td><code>STRING</code></td>
<td>The name of the constraint's schema. An empty string if unnamed.</td>
</tr>
<tr class="odd">
<td><code>CONSTRAINT_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the constraint. This column is never null. If not explicitly specified in the schema definition, a system-defined name is assigned.</td>
</tr>
<tr class="even">
<td><code>CHECK_CLAUSE</code></td>
<td><code>STRING</code></td>
<td>The expressions of the <code>CHECK</code> constraint. This column is never null.</td>
</tr>
<tr class="odd">
<td><code>SPANNER_STATE</code></td>
<td><code>STRING</code></td>
<td>The current state of the <code>CHECK</code> constraint. This column is never null. The possible states are as follows:
<ul>
<li><code>VALIDATING</code> : Spanner is validating the existing data.</li>
<li><code>COMMITTED</code> : There is no active schema change for this constraint.</li>
</ul></td>
</tr>
</tbody>
</table>

### `KEY_COLUMN_USAGE`

This row-filtered table contains one row about each column of the tables from `TABLE_CONSTRAINTS` that are constrained as keys by a `PRIMARY KEY` , `FOREIGN KEY` or `UNIQUE` constraint. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only columns that meet the following criteria:

- Any of the `SELECT` , `INSERT` , or `UPDATE` fine-grained access control privileges are directly granted on the column to the current database role, to roles of which the current database role is a member, or to `public` .
- Any of the `SELECT` , `INSERT` , or `UPDATE` fine-grained access control privileges are granted on the table that contains the column to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name                     | Type     | Description                                                                                                                                                                  |
|---------------------------------|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `CONSTRAINT_CATALOG`            | `STRING` | The name of the constraint's catalog. Always an empty string.                                                                                                                |
| `CONSTRAINT_SCHEMA`             | `STRING` | The name of the constraint's schema. This column is never null. An empty string if unnamed.                                                                                  |
| `CONSTRAINT_NAME`               | `STRING` | The name of the constraint.                                                                                                                                                  |
| `TABLE_CATALOG`                 | `STRING` | The name of the constrained column's catalog. Always an empty string.                                                                                                        |
| `TABLE_SCHEMA`                  | `STRING` | The schema name of the constrained column's table. The name is empty for the default schema, and it contains a value for other schemas. This column always contains a value. |
| `TABLE_NAME`                    | `STRING` | The name of the constrained column's table.                                                                                                                                  |
| `COLUMN_NAME`                   | `STRING` | The name of the column.                                                                                                                                                      |
| `ORDINAL_POSITION`              | `INT64`  | The ordinal position of the column within the constraint's key, starting with a value of `1` .                                                                               |
| `POSITION_IN_UNIQUE_CONSTRAINT` | `INT64`  | For `FOREIGN KEY` s, the ordinal position of the column within the unique constraint, starting with a value of `1` . This column is null for other constraint types.         |

### `CONSTRAINT_COLUMN_USAGE`

This table contains one row about each column used by a constraint. Includes the `PRIMARY KEY` and `UNIQUE` columns, plus the referenced columns of `FOREIGN KEY` constraints.

| Column name          | Type     | Description                                                                                                                                                      |
|----------------------|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `TABLE_CATALOG`      | `STRING` | The name of the column table's catalog. Always an empty string.                                                                                                  |
| `TABLE_SCHEMA`       | `STRING` | The schema name of the column's table. The name is empty for the default schema, and it contains a value for other schemas. This column always contains a value. |
| `TABLE_NAME`         | `STRING` | The name of the column's table.                                                                                                                                  |
| `COLUMN_NAME`        | `STRING` | The name of the column that is used by the constraint.                                                                                                           |
| `CONSTRAINT_CATALOG` | `STRING` | The name of the constraint's catalog. Always an empty string.                                                                                                    |
| `CONSTRAINT_SCHEMA`  | `STRING` | The name of the constraint's schema. An empty string if unnamed.                                                                                                 |
| `CONSTRAINT_NAME`    | `STRING` | The name of the constraint.                                                                                                                                      |

### `TABLE_SYNONYMS`

This table lists lists synonym information for the table.

| Column name          | Type     | Description                                                                                                                                             |
|----------------------|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| `TABLE_CATALOG`      | `STRING` | Not used. Always an empty string.                                                                                                                       |
| `TABLE_SCHEMA`       | `STRING` | The schema name of the table. The name is empty for the default schema, and it contains a value for other schemas. This column always contains a value. |
| `TABLE_NAME`         | `STRING` | The name of the table.                                                                                                                                  |
| `SYNONYM_CATALOG`    | `STRING` | The name of the catalog for the synonym.                                                                                                                |
| `SYNONYM_SCHEMA`     | `STRING` | The name of the schema for the synonym.                                                                                                                 |
| `SYNONYM_TABLE_NAME` | `STRING` | The name of the table for the synonym.                                                                                                                  |

### `INDEXES`

This row-filtered table lists the indexes in the database. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only indexes that meet either of the following requirements:

- Any of the `SELECT` , `INSERT` , or `UPDATE` fine-grained access control privileges are granted at the column level on *all* columns in the index to the current database role, to roles of which the current database role is a member, or to `public` .
- Any of the `SELECT` , `INSERT` , `UPDATE` , or `DELETE` fine-grained access control privileges are granted on the table that has the index to the current database role, to roles of which the current database role is a member, or to `public` .

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
<td><code>TABLE_CATALOG</code></td>
<td><code>STRING</code></td>
<td>The name of the catalog. Always an empty string.</td>
</tr>
<tr class="even">
<td><code>TABLE_SCHEMA</code></td>
<td><code>STRING</code></td>
<td>The schema name of the index table. The name is empty for the default schema, and it contains a value for other schemas. This column always contains a value.</td>
</tr>
<tr class="odd">
<td><code>TABLE_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the table.</td>
</tr>
<tr class="even">
<td><code>INDEX_NAME</code></td>
<td><code>STRING</code></td>
<td>The name of the index. Tables with a <code>PRIMARY KEY</code> specification have a pseudo-index entry generated with the name <code>PRIMARY_KEY</code> , which allows the fields of the primary key to be determined.</td>
</tr>
<tr class="odd">
<td><code>INDEX_TYPE</code></td>
<td><code>STRING</code></td>
<td>The type of the index. The type is <code>INDEX</code> or <code>PRIMARY_KEY</code> .</td>
</tr>
<tr class="even">
<td><code>PARENT_TABLE_NAME</code></td>
<td><code>STRING</code></td>
<td>Secondary indexes can be interleaved in a parent table, as discussed in <a href="https://docs.cloud.google.com/spanner/docs/secondary-indexes#creating_a_secondary_index">Creating a secondary index</a> . This column holds the name of that parent table, or an empty string if the index is not interleaved.</td>
</tr>
<tr class="odd">
<td><code>IS_UNIQUE</code></td>
<td><code>BOOL</code></td>
<td>Whether the index keys must be unique.</td>
</tr>
<tr class="even">
<td><code>IS_NULL_FILTERED</code></td>
<td><code>BOOL</code></td>
<td>Whether the index includes entries with <code>NULL</code> values.</td>
</tr>
<tr class="odd">
<td><code>INDEX_STATE</code></td>
<td><code>STRING</code></td>
<td>The current state of the index. Possible values and the states they represent are:
<ul>
<li><code>PREPARE</code> : creating empty tables for a new index.</li>
<li><code>WRITE_ONLY</code> : backfilling data for a new index.</li>
<li><code>WRITE_ONLY_CLEANUP</code> : cleaning up a new index.</li>
<li><code>WRITE_ONLY_VALIDATE_UNIQUE</code> : checking uniqueness of data in a new index.</li>
<li><code>READ_WRITE</code> : normal index operation.</li>
</ul></td>
</tr>
<tr class="even">
<td><code>SPANNER_IS_MANAGED</code></td>
<td><code>BOOL</code></td>
<td><code>TRUE</code> if the index is managed by Spanner; Otherwise, <code>FALSE</code> . Secondary backing indexes for foreign keys are managed by Spanner.</td>
</tr>
</tbody>
</table>

### `INDEX_COLUMNS`

This row-filtered table lists the columns in an index. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only indexes that meet either of the following requirements:

- Any of the `SELECT` , `INSERT` , or `UPDATE` fine-grained access control privileges are granted at the column level on *all* columns in the index to the current database role, to roles of which the current database role is a member, or to `public` .
- Any of the `SELECT` , `INSERT` , or `UPDATE` fine-grained access control privileges are granted on the table that has index to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name        | Type     | Description                                                                                                                                                                                                                                                                                 |
|--------------------|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `TABLE_CATALOG`    | `STRING` | The name of the catalog. Always an empty string.                                                                                                                                                                                                                                            |
| `TABLE_SCHEMA`     | `STRING` | The schema name of the index table. The name is empty for the default schema, and it contains a value for other schemas. This column always contains a value.                                                                                                                               |
| `TABLE_NAME`       | `STRING` | The name of the table.                                                                                                                                                                                                                                                                      |
| `INDEX_NAME`       | `STRING` | The name of the index.                                                                                                                                                                                                                                                                      |
| `COLUMN_NAME`      | `STRING` | The name of the column.                                                                                                                                                                                                                                                                     |
| `ORDINAL_POSITION` | `INT64`  | The ordinal position of the column in the index (or primary key), starting with a value of 1. This value is `NULL` for non-key columns (for example, columns specified in the [`STORING` clause](https://docs.cloud.google.com/spanner/docs/secondary-indexes#storing_clause) of an index). |
| `COLUMN_ORDERING`  | `STRING` | The ordering of the column. The value is `ASC` or `DESC` for key columns, and `NULL` for non-key columns (for example, columns specified in the `STORING` clause of an index).                                                                                                              |
| `IS_NULLABLE`      | `STRING` | A string that indicates whether the column is nullable. In accordance with the SQL standard, the string is either `YES` or `NO` , rather than a Boolean value.                                                                                                                              |
| `SPANNER_TYPE`     | `STRING` | The [data type](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-types#allowable-types) of the column.                                                                                                                                                                |

### `COLUMN_OPTIONS`

This row-filtered table lists lists the column options in a table. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see options only for columns that meet either of the following requirements:

- Any of the `SELECT` , `INSERT` , or `UPDATE` fine-grained access control privileges are directly granted on the column to the current database role, to roles of which the current database role is a member, or to `public` .
- Any of the `SELECT` , `INSERT` , or `UPDATE` fine-grained access control privileges are granted on the table that contains the column to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name     | Type     | Description                                                                                                                                                                                                                     |
|-----------------|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `TABLE_CATALOG` | `STRING` | The name of the catalog. Always an empty string.                                                                                                                                                                                |
| `TABLE_SCHEMA`  | `STRING` | The name of the schema. The name is empty for the default schema and non-empty for other schemas (for example, the `INFORMATION_SCHEMA` itself). This column is never null.                                                     |
| `TABLE_NAME`    | `STRING` | The name of the table.                                                                                                                                                                                                          |
| `COLUMN_NAME`   | `STRING` | The name of the column.                                                                                                                                                                                                         |
| `OPTION_NAME`   | `STRING` | A SQL identifier that uniquely identifies the option. This identifier is the key of the `OPTIONS` clause in DDL.                                                                                                                |
| `OPTION_TYPE`   | `STRING` | A data type name that is the type of this option value.                                                                                                                                                                         |
| `OPTION_VALUE`  | `STRING` | A SQL literal describing the value of this option. The value of this column must be parsable as part of a query. The expression resulting from parsing the value must be castable to `OPTION_TYPE` . This column is never null. |

### `SEQUENCES`

This table lists the sequences metadata. `SEQUENCES` is row-filtered based on fine-grained access privileges, if a user with fine-grained access privileges is querying it.

| Column name | Type     | Description                                                     |
|-------------|----------|-----------------------------------------------------------------|
| `CATALOG`   | `STRING` | The name of the catalog containing the sequence.                |
| `SCHEMA`    | `STRING` | The name of the schema containing the sequence.                 |
| `NAME`      | `STRING` | The name of the sequence.                                       |
| `DATA_TYPE` | `STRING` | The type of the sequence values. It uses the `INT64` data type. |

### `SEQUENCE_OPTIONS`

This table contains the configuration options for sequences. `SEQUENCE_OPTIONS` is row-filtered based on fine-grained access privileges, if a user with fine-grained access privileges is querying it.

| Column name    | Type     | Description                                                                                                          |
|----------------|----------|----------------------------------------------------------------------------------------------------------------------|
| `CATALOG`      | `STRING` | The name of the catalog containing the sequence.                                                                     |
| `SCHEMA`       | `STRING` | The name of the schema containing the sequence.                                                                      |
| `NAME`         | `STRING` | The name of the sequence.                                                                                            |
| `OPTION_NAME`  | `STRING` | The name of the sequence option.                                                                                     |
| `OPTION_TYPE`  | `STRING` | A data type name that is the type of this option value.                                                              |
| `OPTION_VALUE` | `STRING` | The sequence option value. The expression that results from parsing the value must permit casting to `OPTION_TYPE` . |

### `SPANNER_STATISTICS`

This table lists the available query optimizer statistics packages.

| Column name    | Type     | Description                                                                                                                                                                                                    |
|----------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `CATALOG_NAME` | `STRING` | The name of the catalog. Always an empty string.                                                                                                                                                               |
| `SCHEMA_NAME`  | `STRING` | The name of the schema. The name is empty for the default schema and non-empty for other schemas (for example, the `INFORMATION_SCHEMA` itself). This column is never null.                                    |
| `PACKAGE_NAME` | `STRING` | The name of the statistics package.                                                                                                                                                                            |
| `ALLOW_GC`     | `BOOL`   | `FALSE` if the statistics package is exempted from garbage collection; Otherwise, `TRUE` . This attribute must be set to `FALSE` in order to reference the statistics package in a hint or through client API. |

### `VIEWS`

This row-filtered table lists the views in the database. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only views on which the `SELECT` fine-grained access control privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name       | Type     | Description                                                                                                                                                |
|-------------------|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `TABLE_CATALOG`   | `STRING` | The name of the catalog. Always an empty string.                                                                                                           |
| `TABLE_SCHEMA`    | `STRING` | The name of the schema. An empty string if unnamed.                                                                                                        |
| `TABLE_NAME`      | `STRING` | The name of the view.                                                                                                                                      |
| `VIEW_DEFINITION` | `STRING` | The SQL text of the query that defines the view.                                                                                                           |
| `SECURITY_TYPE`   | `STRING` | The security type of the view. Either `INVOKER` or `DEFINER` . For more information, see [About views](https://docs.cloud.google.com/spanner/docs/views) . |

### `ROLES`

This row-filtered table lists the defined database roles for [fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) , including system roles. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all database roles. All other principals can see only database roles to which they have been granted access either directly or through inheritance.

| Column name | Type     | Description                                                                                                                       |
|-------------|----------|-----------------------------------------------------------------------------------------------------------------------------------|
| `ROLE_NAME` | `STRING` | The name of the database role.                                                                                                    |
| `IS_SYSTEM` | `BOOL`   | `TRUE` if the database role is a [system role](https://docs.cloud.google.com/spanner/docs/fgac-system-roles) ; `FALSE` otherwise. |

### `ROLE_GRANTEES`

This row-filtered table lists all role memberships explicitly granted to all database roles. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only the role memberships granted to the current database role or to a role of which the current database role is a member.

Because all database roles are members of the [public role](https://docs.cloud.google.com/spanner/docs/fgac-system-roles#public) , the results omit records for implicit membership in the public role.

| Column name | Type     | Description                                                        |
|-------------|----------|--------------------------------------------------------------------|
| `ROLE_NAME` | `STRING` | The name of the database role in which this membership is granted. |
| `GRANTEE`   | `STRING` | The name of the database role to which this membership is granted. |

### `CHANGE_STREAMS`

This row-filtered table lists all of a database's change streams, and notes which ones track the entire database versus specific tables or columns. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only change streams on which the `SELECT` fine-grained access control privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name             | Type     | Description                                                                                                               |
|-------------------------|----------|---------------------------------------------------------------------------------------------------------------------------|
| `CHANGE_STREAM_CATALOG` | `STRING` | The name of the change stream's catalog. Always an empty string.                                                          |
| `CHANGE_STREAM_SCHEMA`  | `STRING` | The name of this change stream's schema. Always an empty string.                                                          |
| `CHANGE_STREAM_NAME`    | `STRING` | The name of the change stream.                                                                                            |
| `ALL`                   | `BOOL`   | `TRUE` if this change stream tracks the entire database. `FALSE` if this change stream tracks specific tables or columns. |

### `CHANGE_STREAM_TABLES`

This row-filtered table contains information about tables and the change streams that watch them. Each row describes one table and one change stream. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only rows for change streams on which the `SELECT` privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

The data in `CHANGE_STREAM_TABLES` does not include the implicit relationships between tables and change streams that track the entire database.

| Column name             | Type     | Description                                                                                                  |
|-------------------------|----------|--------------------------------------------------------------------------------------------------------------|
| `CHANGE_STREAM_CATALOG` | `STRING` | The name of the change stream's catalog. Always an empty string.                                             |
| `CHANGE_STREAM_SCHEMA`  | `STRING` | The name of the change stream's schema. Always an empty string.                                              |
| `CHANGE_STREAM_NAME`    | `STRING` | The name of the change stream that this row refers to.                                                       |
| `TABLE_CATALOG`         | `STRING` | The name of the table's catalog. Always an empty string.                                                     |
| `TABLE_SCHEMA`          | `STRING` | The name of the table's schema. Always an empty string.                                                      |
| `TABLE_NAME`            | `STRING` | The name of the table that this row refers to.                                                               |
| `ALL_COLUMNS`           | `BOOL`   | `TRUE` if this row's change stream tracks the entirety of the table this row refers to. Otherwise, `FALSE` . |

### `CHANGE_STREAM_COLUMNS`

This row-filtered table contains information about table columns and the change streams that watch them. Each row describes one change stream and one column. If a change stream tracks an entire table, then the columns in that table don't show in this view.

Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only rows for change streams on which the `SELECT` privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name             | Type     | Description                                                      |
|-------------------------|----------|------------------------------------------------------------------|
| `CHANGE_STREAM_CATALOG` | `STRING` | The name of the change stream's catalog. Always an empty string. |
| `CHANGE_STREAM_SCHEMA`  | `STRING` | The name of the change stream's schema. Always an empty string.  |
| `CHANGE_STREAM_NAME`    | `STRING` | The name of the change stream.                                   |
| `TABLE_CATALOG`         | `STRING` | The name of the table's catalog. Always an empty string.         |
| `TABLE_SCHEMA`          | `STRING` | The name of the table's schema. Always an empty string.          |
| `TABLE_NAME`            | `STRING` | The name of the table that this row refers to.                   |
| `COLUMN_NAME`           | `STRING` | The name of the column that this row refers to.                  |

### `CHANGE_STREAM_OPTIONS`

This row-filtered table contains the configuration options for change streams. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only options for change streams on which the `SELECT` privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name             | Type     | Description                                                      |
|-------------------------|----------|------------------------------------------------------------------|
| `CHANGE_STREAM_CATALOG` | `STRING` | The name of the change stream's catalog. Always an empty string. |
| `CHANGE_STREAM_SCHEMA`  | `STRING` | The name of the change stream's schema. Always an empty string.  |
| `CHANGE_STREAM_NAME`    | `STRING` | The name of the change stream.                                   |
| `OPTION_NAME`           | `STRING` | The name of the change stream option.                            |
| `OPTION_TYPE`           | `STRING` | The data type of the change stream option.                       |
| `OPTION_VALUE`          | `STRING` | The change stream option value.                                  |

### `CHANGE_STREAM_PRIVILEGES`

This row-filtered table lists all fine-grained access control privileges granted on all change streams to any database role, including `public` . Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on change streams to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name             | Type     | Description                                                             |
|-------------------------|----------|-------------------------------------------------------------------------|
| `CHANGE_STREAM_CATALOG` | `STRING` | The name of the catalog containing the change stream (an empty string). |
| `CHANGE_STREAM_SCHEMA`  | `STRING` | The name of the schema containing the change stream (an empty string).  |
| `CHANGE_STREAM_NAME`    | `STRING` | The name of the change stream.                                          |
| `PRIVILEGE_TYPE`        | `STRING` | `SELECT` (the only privilege allowed for change streams).               |
| `GRANTEE`               | `STRING` | The name of database role to which this privilege is granted.           |

### `ROUTINES`

This row-filtered table lists all of a database's change stream read functions. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only change stream read functions on which the `EXECUTE` fine-grained access control privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name          | Type     | Description                                                                              |
|----------------------|----------|------------------------------------------------------------------------------------------|
| `SPECIFIC_CATALOG`   | `STRING` | The name of the routine's catalog. Always an empty string.                               |
| `SPECIFIC_SCHEMA`    | `STRING` | The name of the routine's schema. Always an empty string.                                |
| `SPECIFIC_NAME`      | `STRING` | The name of the routine. Uniquely identifies the routine even if its name is overloaded. |
| `ROUTINE_CATALOG`    | `STRING` | The name of the routine's catalog. Always an empty string.                               |
| `ROUTINE_SCHEMA`     | `STRING` | The name of the routine's schema. Always an empty string.                                |
| `ROUTINE_NAME`       | `STRING` | The name of the routine. (Might be duplicated in case of overloading.)                   |
| `ROUTINE_TYPE`       | `STRING` | The type of the routine ( `FUNCTION` or `PROCEDURE` ). Always `FUNCTION`                 |
| `DATA_TYPE`          | `STRING` | The data type that the routine returns                                                   |
| `ROUTINE_BODY`       | `STRING` | The type of the routine body ( `SQL` or `EXTERNAL` ).                                    |
| `ROUTINE_DEFINITION` | `STRING` | The definition for the `ROUTINE_BODY` .                                                  |
| `SECURITY_TYPE`      | `STRING` | The security type of the routine. Always `INVOKER` .                                     |

### `ROUTINE_OPTIONS`

This row-filtered table contains one row for each option for each defined change stream read function.

Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only options for change stream read functions on which the `EXECUTE` fine-grained access control privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name        | Type     | Description                                                                                                      |
|--------------------|----------|------------------------------------------------------------------------------------------------------------------|
| `SPECIFIC_CATALOG` | `STRING` | The name of the routine's catalog. Always an empty string.                                                       |
| `SPECIFIC_SCHEMA`  | `STRING` | The name of the routine's schema. Always an empty string.                                                        |
| `SPECIFIC_NAME`    | `STRING` | The name of the routine. Uniquely identifies the routine in case of name overloading.                            |
| `OPTION_NAME`      | `STRING` | A SQL identifier that uniquely identifies the option.                                                            |
| `OPTION_TYPE`      | `STRING` | The data type of `OPTION_VALUE` .                                                                                |
| `OPTION_VALUE`     | `STRING` | A SQL literal describing the value of this option. The value of this column must be parsable as part of a query. |

### `PARAMETERS`

This row-filtered table defines the arguments for each change stream read function. Each row describes one argument for one change stream read function.

Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only parameters for change stream read functions on which the `EXECUTE` fine-grained access control privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name         | Type     | Description                                                                                  |
|---------------------|----------|----------------------------------------------------------------------------------------------|
| `SPECIFIC_CATALOG`  | `STRING` | The name of the routine's catalog. Always an empty string.                                   |
| `SPECIFIC_SCHEMA`   | `STRING` | The name of the routine's schema. Always an empty string.                                    |
| `SPECIFIC_NAME`     | `STRING` | The name of the routine. Uniquely identifies the routine in case of name overloading.        |
| `ORDINAL_POSITION`  | `INT64`  | The ordinal position of the parameter in the routine, starting with a value of 1.            |
| `PARAMETER_NAME`    | `STRING` | The name of the parameter.                                                                   |
| `DATA_TYPE`         | `STRING` | The data type of the parameter.                                                              |
| `PARAMETER_DEFAULT` | `STRING` | The default value of the parameter or `NULL` for parameters that don't have a default value. |

### `ROUTINE_PRIVILEGES`

This row-filtered table lists all fine-grained access control privileges granted on all change stream read functions to any database role, including `public` . Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on change stream read functions to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name        | Type     | Description                                                                           |
|--------------------|----------|---------------------------------------------------------------------------------------|
| `SPECIFIC_CATALOG` | `STRING` | The name of the routine's catalog. Always an empty string.                            |
| `SPECIFIC_SCHEMA`  | `STRING` | The name of the routine's schema. Always an empty string.                             |
| `SPECIFIC_NAME`    | `STRING` | The name of the routine. Uniquely identifies the routine in case of name overloading. |
| `PRIVILEGE_TYPE`   | `STRING` | Always `EXECUTE` .                                                                    |
| `GRANTEE`          | `STRING` | The name of the database role to which this privilege is granted.                     |

### `ROLE_TABLE_GRANTS`

This row-filtered table lists all fine-grained access control privileges granted on all tables and views to any database role, including `public` . Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on tables and views to the current database role and to roles of which the current database role is a member, not including `public` .

| **Column name**  | **Type** | **Description**                                                             |
|------------------|----------|-----------------------------------------------------------------------------|
| `GRANTOR`        | `STRING` | Not used. Always `NULL` .                                                   |
| `GRANTEE`        | `STRING` | The name of the database role to which this privilege is granted.           |
| `TABLE_CATALOG`  | `STRING` | Not used. Always an empty string.                                           |
| `TABLE_SCHEMA`   | `STRING` | Not used. Always an empty string.                                           |
| `TABLE_NAME`     | `STRING` | The name of the table or view.                                              |
| `PRIVILEGE_TYPE` | `STRING` | The type of the privilege ( `SELECT` , `INSERT` , `UPDATE` , or `DELETE` ). |
| `IS_GRANTABLE`   | `STRING` | Not used. Always `NO` .                                                     |

### `ROLE_COLUMN_GRANTS`

This row-filtered table lists all fine-grained access control privileges granted on all columns to any database role, including `public` . Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on columns to the current database role and to roles of which the current database role is a member, not including `public` .

The view includes the `SELECT` , `INSERT` , and `UPDATE` privileges that the column inherits from the table or view that contains the column.

| **Column name**  | **Type** | **Description**                                                   |
|------------------|----------|-------------------------------------------------------------------|
| `GRANTOR`        | `STRING` | Not used. Always `NULL` .                                         |
| `GRANTEE`        | `STRING` | The name of the database role to which this privilege is granted. |
| `TABLE_CATALOG`  | `STRING` | Not used. Always an empty string.                                 |
| `TABLE_SCHEMA`   | `STRING` | Not used. Always an empty string.                                 |
| `TABLE_NAME`     | `STRING` | The name of the table or view that contains the column.           |
| `COLUMN_NAME`    | `STRING` | The name of the column on which the privilege is granted.         |
| `PRIVILEGE_TYPE` | `STRING` | The type of the privilege ( `SELECT` , `INSERT` , or `UPDATE` ).  |
| `IS_GRANTABLE`   | `STRING` | Not used. Always `NO` .                                           |

### `ROLE_CHANGE_STREAM_GRANTS`

This row-filtered table lists the `SELECT` privileges granted on all change streams to any database role, including `public` . Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on change streams to the current database role and to roles of which the current database role is a member, not including `public` .

| **Column name**         | **Type** | **Description**                                                   |
|-------------------------|----------|-------------------------------------------------------------------|
| `CHANGE_STREAM_CATALOG` | `STRING` | Not used. Always an empty string.                                 |
| `CHANGE_STREAM_SCHEMA`  | `STRING` | The name of the schema that contains the change stream.           |
| `CHANGE_STREAM_NAME`    | `STRING` | The name of the change stream.                                    |
| `PRIVILEGE_TYPE`        | `STRING` | The type of the privilege ( `SELECT` only).                       |
| `GRANTEE`               | `STRING` | The name of the database role to which this privilege is granted. |

### `ROLE_MODEL_GRANTS`

This row-filtered table lists all fine-grained access control privileges granted on all models to any database role, including `public` . Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on tables and views to the current database role and to roles of which the current database role is a member, not including `public` .

| **Column name**  | **Type** | **Description**                                                   |
|------------------|----------|-------------------------------------------------------------------|
| `GRANTOR`        | `STRING` | Not used. Always `NULL` .                                         |
| `GRANTEE`        | `STRING` | The name of the database role to which this privilege is granted. |
| `MODEL_CATALOG`  | `STRING` | Not used. Always an empty string.                                 |
| `MODEL_SCHEMA`   | `STRING` | Not used. Always an empty string.                                 |
| `MODEL_NAME`     | `STRING` | The name of the model.                                            |
| `PRIVILEGE_TYPE` | `STRING` | The type of the privilege ( `EXECUTE` ).                          |
| `IS_GRANTABLE`   | `STRING` | Not used. Always `NO` .                                           |

### `ROLE_ROUTINE_GRANTS`

This row-filtered table lists the `EXECUTE` privileges granted on all change stream read functions to any database role, including `public` . Principals with IAM database-level permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see only privileges granted on change stream read functions to the current database role and to roles of which the current database role is a member, not including `public` .

| Column name        | Type     | Description                                                                           |
|--------------------|----------|---------------------------------------------------------------------------------------|
| `GRANTOR`          | `STRING` | Not used. Always `NULL` .                                                             |
| `GRANTEE`          | `STRING` | The name of the role that the privilege is granted to.                                |
| `SPECIFIC_CATALOG` | `STRING` | The name of the routine catalog.                                                      |
| `SPECIFIC_SCHEMA`  | `STRING` | The name of the routine schema.                                                       |
| `SPECIFIC_NAME`    | `STRING` | The name of the routine. Uniquely identifies the routine in case of name overloading. |
| `PRIVILEGE_TYPE`   | `STRING` | The type of the privilege granted. Always `EXECUTE` .                                 |
| `IS_GRANTABLE`     | `STRING` | Not used. Always `NO` .                                                               |

### `MODELS`

This table lists all of a database's [models](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_model) .

| Column name     | Type     | Description                                                           |
|-----------------|----------|-----------------------------------------------------------------------|
| `MODEL_CATALOG` | `STRING` | The name of the catalog. Always an empty string.                      |
| `MODEL_SCHEMA`  | `STRING` | The name of this model's schema. Always an empty string.              |
| `MODEL_NAME`    | `STRING` | The name of the model.                                                |
| `IS_REMOTE`     | `BOOL`   | `TRUE` if this is a remote model. `FALSE` if this is a managed model. |

### `MODEL_OPTIONS`

This table contains the configuration options for models.

| Column name     | Type     | Description                                              |
|-----------------|----------|----------------------------------------------------------|
| `MODEL_CATALOG` | `STRING` | The name of the catalog. Always an empty string.         |
| `MODEL_SCHEMA`  | `STRING` | The name of this model's schema. Always an empty string. |
| `MODEL_NAME`    | `STRING` | The name of the model.                                   |
| `OPTION_NAME`   | `STRING` | The name of the model option.                            |
| `OPTION_TYPE`   | `STRING` | The data type of the model option.                       |
| `OPTION_VALUE`  | `STRING` | The model option value.                                  |

### `MODEL_COLUMNS`

This table lists the columns in a model.

| Column name        | Type     | Description                                                                                                       |
|--------------------|----------|-------------------------------------------------------------------------------------------------------------------|
| `MODEL_CATALOG`    | `STRING` | The name of the catalog. Always an empty string.                                                                  |
| `MODEL_SCHEMA`     | `STRING` | The name of this model's schema. Always an empty string.                                                          |
| `MODEL_NAME`       | `STRING` | The name of the model.                                                                                            |
| `COLUMN_KIND`      | `STRING` | Model column kind. One of: `"INPUT"` or `"OUTPUT"` .                                                              |
| `COLUMN_NAME`      | `STRING` | The name of the column.                                                                                           |
| `DATA_TYPE`        | `STRING` | The column's standard SQL data type.                                                                              |
| `ORDINAL_POSITION` | `INT64`  | Ordinal position of the column, starting with value of 1, to preserve the order of declared columns.              |
| `IS_EXPLICIT`      | `BOOL`   | `TRUE` if the column was specified explicitly in the DDL, `FALSE` if the column was discovered from the endpoint. |

### `MODEL_COLUMN_OPTIONS`

This table contains the configuration options for model columns.

| Column name     | Type     | Description                                              |
|-----------------|----------|----------------------------------------------------------|
| `MODEL_CATALOG` | `STRING` | The name of the catalog. Always an empty string.         |
| `MODEL_SCHEMA`  | `STRING` | The name of this model's schema. Always an empty string. |
| `MODEL_NAME`    | `STRING` | The name of the model.                                   |
| `COLUMN_KIND`   | `STRING` | Model column kind. One of: `"INPUT"` or `"OUTPUT"` .     |
| `COLUMN_NAME`   | `STRING` | The name of the column.                                  |
| `OPTION_NAME`   | `STRING` | The name of the model column option.                     |
| `OPTION_TYPE`   | `STRING` | The data type of the model column option.                |
| `OPTION_VALUE`  | `STRING` | The model column option value.                           |

### `MODEL_PRIVILEGES`

This row-filtered table lists all the privileges granted at the model-level to [database roles](https://docs.cloud.google.com/spanner/docs/information-schema#roles) , including `public` . Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can see privileges only for models on which `EXECUTE` fine-grained access control privilege is granted to the current database role, to roles of which the current database role is a member, or to `public` .

| Column name      | Type     | Description                                                                        |
|------------------|----------|------------------------------------------------------------------------------------|
| `MODEL_CATALOG`  | `STRING` | Not used. Always an empty string.                                                  |
| `MODEL_SCHEMA`   | `STRING` | Not used. Always an empty string.                                                  |
| `MODEL_NAME`     | `STRING` | The name of the model on which fine-grained access control privileges are granted. |
| `PRIVILEGE_TYPE` | `STRING` | `EXECUTE` \>                                                                       |
| `GRANTEE`        | `STRING` | The name of the database role to which this privilege is granted.                  |

### `PROPERTY_GRAPHS`

This row-filtered table lists the [property graphs](https://docs.cloud.google.com/spanner/docs/graph/schema-overview#property-graph-data-model) in the database. Principals with database-level IAM permissions and principals who have been granted access to the `spanner_info_reader` system role or to members of that role can see all rows in this view. All other principals can only see property graphs if they meet the requirements to see all the tables used to define those graphs.

| Column name                    | Type     | Description                                          |
|--------------------------------|----------|------------------------------------------------------|
| `PROPERTY_GRAPH_CATALOG`       | `STRING` | The name of the catalog. Always an empty string.     |
| `PROPERTY_GRAPH_SCHEMA`        | `STRING` | The name of the schema. An empty string if unnamed.  |
| `PROPERTY_GRAPH_NAME`          | `STRING` | The name of the property graph.                      |
| `PROPERTY_GRAPH_METADATA_JSON` | `JSON`   | The definition of the property graph in JSON format. |

The `PROPERTY_GRAPH_METADATA_JSON` column contains a `PropertyGraph` JSON object defined as the following:

| JSON object name           | Field name                | JSON type                                                                                                                                                               | Description                                      |
|----------------------------|---------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------|
| `PropertyGraph`            | `catalog`                 | `string`                                                                                                                                                                | The name of the catalog. Always an empty string. |
| `schema`                   | `string`                  | The name of the schema. An empty string if unnamed.                                                                                                                     |                                                  |
| `name`                     | `string`                  | The name of the property graph.                                                                                                                                         |                                                  |
| `nodeTables`               | `array<object>`           | A list of `GraphElementTable` objects for nodes.                                                                                                                        |                                                  |
| `edgeTables`               | `array<object>`           | A list of `GraphElementTable` objects for edges.                                                                                                                        |                                                  |
| `labels`                   | `array<object>`           | A list of `GraphElementLabel` objects.                                                                                                                                  |                                                  |
| `propertyDeclarations`     | `array<object>`           | A list of `GraphPropertyDeclaration` objects.                                                                                                                           |                                                  |
| `GraphElementTable`        | `name`                    | `string`                                                                                                                                                                | The name of the graph element table.             |
| `kind`                     | `string`                  | Either `NODE` or `EDGE` .                                                                                                                                               |                                                  |
| `baseCatalogName`          | `string`                  | The name of the catalog containing the base table.                                                                                                                      |                                                  |
| `baseSchemaName`           | `string`                  | The name of the schema containing the base table.                                                                                                                       |                                                  |
| `baseTableName`            | `string`                  | The name of the input table from which elements are created.                                                                                                            |                                                  |
| `keyColumns`               | `array<string>`           | The column names that constitute the element key.                                                                                                                       |                                                  |
| `labelNames`               | `array<string>`           | The label names attached to this element table.                                                                                                                         |                                                  |
| `propertyDefinitions`      | `array<object>`           | A list of `GraphPropertyDefinition` objects.                                                                                                                            |                                                  |
| `dynamicLabelExpr`         | `string`                  | The name of the column that contains the [`DYNAMIC LABEL`](https://docs.cloud.google.com/spanner/docs/graph/manage-schemaless-data#dynamic-label) definition.           |                                                  |
| `dynamicPropertyExpr`      | `string`                  | The name of the column that contains the [`DYNAMIC PROPERTIES`](https://docs.cloud.google.com/spanner/docs/graph/manage-schemaless-data#dynamic-properties) definition. |                                                  |
| `sourceNodeTable`          | `object`                  | A `GraphNodeTableReference` object. Only exist when the `kind` is `EDGE` .                                                                                              |                                                  |
| `destinationNodeTable`     | `object`                  | A `GraphNodeTableReference` object. Only exist when the `kind` is `EDGE` .                                                                                              |                                                  |
| `GraphNodeTableReference`  | `nodeTableName`           | `string`                                                                                                                                                                | The name of the graph element table.             |
| `edgeTableColumns`         | `array<string>`           | The name of the columns that are associated with the source and destination keys for the edges.                                                                         |                                                  |
| `nodeTableColumns`         | `array<string>`           | The name of the columns that are associated with the source and destination keys for the nodes.                                                                         |                                                  |
| `GraphElementLabel`        | `name`                    | `string`                                                                                                                                                                | The name of the label.                           |
| `propertyDeclarationNames` | `array<string>`           | The names of the properties associated with this label.                                                                                                                 |                                                  |
| `GraphPropertyDeclaration` | `name`                    | `string`                                                                                                                                                                | The name of the property.                        |
| `type`                     | `string`                  | The type of the property.                                                                                                                                               |                                                  |
| `GraphPropertyDefinition`  | `propertyDeclarationName` | `string`                                                                                                                                                                | The name of the property.                        |
| `valueExpressionSql`       | `string`                  | The expression that defines the property.                                                                                                                               |                                                  |

## Examples

Return information about each table in the user's schema:

```
SELECT
  t.table_schema,
  t.table_name,
  t.parent_table_name
FROM
  information_schema.tables AS t
WHERE
  t.table_catalog = ''
  AND
  t.table_schema NOT IN ('information_schema', 'SPANNER_SYS')
  AND t.table_type = 'BASE TABLE'
ORDER BY
  t.table_catalog,
  t.table_schema,
  t.table_name
```

Return the name of all tables in the INFORMATION_SCHEMA:

```
SELECT
  t.table_name
FROM
  information_schema.tables AS t
WHERE
  t.table_schema = "SPANNER_SYS"
```

Return information about the columns in the user table `MyTable` in default schema:

```
SELECT
  t.column_name,
  t.spanner_type,
  t.is_nullable
FROM
  information_schema.columns AS t
WHERE
  t.table_catalog = ''
  AND
  t.table_schema = ''
  AND
  t.table_name = 'MyTable'
ORDER BY
  t.table_catalog,
  t.table_schema,
  t.table_name,
  t.ordinal_position
```

Return information on what the default leader region for the database is. Returns empty if the default leader is not set:

```
SELECT
  s.option_name,
  s.option_value
FROM
  information_schema.database_options s
WHERE
  s.option_name = 'default_leader'
```

Return information about each index in the user's schema:

```
SELECT
  t.table_schema,
  t.table_name,
  t.index_name,
  t.parent_table_name
FROM
  information_schema.indexes AS t
WHERE
  t.table_catalog = ''
  AND
  t.table_schema NOT IN ('information_schema', 'SPANNER_SYS')
  AND
  t.index_type != 'PRIMARY_KEY'
ORDER BY
  t.table_catalog,
  t.table_schema,
  t.table_name,
  t.index_name
```

Returns all the columns that use options other than the default:

```
SELECT
  t.table_schema,
  t.table_name,
  t.column_name,
  t.option_type,
  t.option_value,
  t.option_name
FROM
  information_schema.column_options AS t
WHERE
  t.table_catalog = ''
AND
  t.table_schema NOT IN ('information_schema', 'SPANNER_SYS')
```

Returns the current optimizer related database options:

```
SELECT
  s.option_name,
  s.option_value
FROM
  information_schema.database_options s
WHERE
  s.schema_name=''
  AND s.option_name IN ('optimizer_version',
    'optimizer_statistics_package')
```

Returns all available statistics packages:

```
SELECT
  *
FROM
  information_schema.spanner_statistics;
```

Return all sequences:

```
SELECT
  *
FROM
  information_schema.sequences;
```

Return all sequence options for the sequence named "MySequence"

```
SELECT
  *
FROM
  information_schema.sequence_options WHERE name="MySequence";
```

Return the names of all property graphs and their definitions:

```
SELECT
  property_graph_name,
  property_graph_metadata_json
FROM
  information_schema.property_graphs
```

Return the names of all property graphs together with their labels and properties:

```
SELECT
  property_graph_name,
  property_graph_metadata_json.labels,
  property_graph_metadata_json.propertyDeclarations
FROM
  information_schema.property_graphs
```

## What's next

- Learn about available [Introspection tools](https://docs.cloud.google.com/spanner/docs/introspection) to help you investigate database issues.
