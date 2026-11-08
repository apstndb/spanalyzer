---
name: documents/docs.cloud.google.com/spanner/docs/fgac-change-streams
uri: https://docs.cloud.google.com/spanner/docs/fgac-change-streams
title: Fine-grained access control for change streams
description: Control access to Spanner change streams using FGAC and grant EXECUTE privilege on table-valued functions.
data_source: docs.cloud.google.com
---

This page explains how [fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) works with Spanner change streams for GoogleSQL-dialect databases and PostgreSQL-dialect databases.

For fine-grained access control users, you allow read access to change streams data by using the following grants. Both grants are required.

- Grant `SELECT` on the change stream.

  ### GoogleSQL

  ```
  GRANT SELECT ON CHANGE STREAM CHANGE_STREAM_NAME TO ROLE ROLE_NAME;
  ```

  ### PostgreSQL

  ```
  GRANT SELECT ON CHANGE STREAM CHANGE_STREAM_NAME TO ROLE_NAME;
  ```

- Grant `EXECUTE` on the read function that is automatically created for the change stream. You use the read function to read change stream records.

  ### GoogleSQL

  ```
  GRANT EXECUTE ON TABLE FUNCTION READ_FUNCTION_NAME TO ROLE ROLE_NAME;
  ```

  ### PostgreSQL

  ```
  GRANT EXECUTE ON FUNCTION READ_FUNCTION_NAME TO ROLE_NAME;
  ```

  For information about naming conventions for change stream read functions and formatting for the information that they return, see the following topics:

  - [Change stream read functions and query syntax](https://docs.cloud.google.com/spanner/docs/change-streams/details#change_stream_query_syntax)
  - [Change streams record format](https://docs.cloud.google.com/spanner/docs/change-streams/details#change_streams_record_format)

## `INFORMATION_SCHEMA` views for change streams

The following views show database roles and privileges information for change streams:

- GoogleSQL-dialect databases: [`INFORMATION_SCHEMA.CHANGE_STREAM_PRIVILEGES`](https://docs.cloud.google.com/spanner/docs/information-schema#change-stream-privileges)
- PostgreSQL-dialect databases: [`information_schema.change_stream_privileges`](https://docs.cloud.google.com/spanner/docs/information-schema-pg#change-stream-privileges)

The rows in these views are filtered based on the current database role privileges on change streams. This ensures that principals can view only the roles, privileges, and change streams that they have access to.

Row filtering also applies to following change streams-related views:

### GoogleSQL

- [`INFORMATION_SCHEMA.CHANGE_STREAMS`](https://docs.cloud.google.com/spanner/docs/information-schema#change-streams)
- [`INFORMATION_SCHEMA.CHANGE_STREAM_TABLES`](https://docs.cloud.google.com/spanner/docs/information-schema#change-stream-tables)
- [`INFORMATION_SCHEMA.CHANGE_STREAM_COLUMNS`](https://docs.cloud.google.com/spanner/docs/information-schema#change-stream-columns)
- [`INFORMATION_SCHEMA.CHANGE_STREAM_OPTIONS`](https://docs.cloud.google.com/spanner/docs/information-schema#change-stream-options)

The system role `spanner_info_reader` and its members always see an unfiltered `INFORMATION_SCHEMA` .

### PostgreSQL

- [`information_schema.change_streams`](https://docs.cloud.google.com/spanner/docs/information-schema-pg#change-streams)
- [`information_schema.change_stream_tables`](https://docs.cloud.google.com/spanner/docs/information-schema-pg#change-stream-tables)
- [`information_schema.change_stream_columns`](https://docs.cloud.google.com/spanner/docs/information-schema-pg#change-stream-columns)
- [`information_schema.change_stream_options`](https://docs.cloud.google.com/spanner/docs/information-schema-pg#change-stream-options)

The system role `spanner_info_reader` and its members see an unfiltered `information_schema` .

Row filtering also applies to the following metadata views for change stream read functions:

### GoogleSQL

- [`INFORMATION_SCHEMA.ROUTINES`](https://docs.cloud.google.com/spanner/docs/information-schema#routines)
- [`INFORMATION_SCHEMA.ROUTINE_OPTIONS`](https://docs.cloud.google.com/spanner/docs/information-schema#routine_options)
- [`INFORMATION_SCHEMA.ROUTINE_PRIVILEGES`](https://docs.cloud.google.com/spanner/docs/information-schema#routine_privileges)
- [`INFORMATION_SCHEMA.PARAMETERS`](https://docs.cloud.google.com/spanner/docs/information-schema#parameters)

### PostgreSQL

- [`information_schema.routines`](https://docs.cloud.google.com/spanner/docs/information-schema-pg#routines)
- [`information_schema.routine_options`](https://docs.cloud.google.com/spanner/docs/information-schema-pg#routine_options)
- [`information_schema.routine_privileges`](https://docs.cloud.google.com/spanner/docs/information-schema-pg#routine_privileges)
- [`information_schema.parameters`](https://docs.cloud.google.com/spanner/docs/information-schema-pg#parameters)

## Caveats

- Change streams use a metadata database to maintain internal state. The metadata database can be the same as or different from the application database. We recommend that you use a different database. However, for fine-grained access control users, the metadata database can't be the same as the application database. This is because the IAM principal that runs the Dataflow job needs read or write access at the database level for the metadata database. This would override the fine-grained access control privileges that were configured for the application database.

  For more information, see [Consider a separate metadata database](https://docs.cloud.google.com/spanner/docs/change-streams/manage#why-metadata) .

- Because a change stream contains a separate copy of the data from the tracked tables and columns, be careful when granting users access to the change stream. The readers of the change stream can view data changes from the tracked tables and columns, even when they don't have `SELECT` privileges on the tables and columns. Although it's more flexible to set up separate controls on change streams and their tracked tables and columns, there's a potential risk, so ensure that you structure database roles and privileges accordingly. For example, when revoking the `SELECT` privilege on a table from a role, consider whether to also revoke `SELECT` on the change stream and revoke `EXECUTE` on the associated read function.

- If you grant `SELECT` on a change stream that tracks all tables, the grantee can see data changes for any tables added in the future.

## What's next

- [Change streams overview](https://docs.cloud.google.com/spanner/docs/change-streams)
- [Create and manage change streams](https://docs.cloud.google.com/spanner/docs/change-streams/manage)
- [Fine-grained access control overview](https://docs.cloud.google.com/spanner/docs/fgac-about)
