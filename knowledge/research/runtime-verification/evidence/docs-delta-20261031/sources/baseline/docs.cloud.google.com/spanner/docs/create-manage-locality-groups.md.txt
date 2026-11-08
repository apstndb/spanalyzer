---
name: documents/docs.cloud.google.com/spanner/docs/create-manage-locality-groups
uri: https://docs.cloud.google.com/spanner/docs/create-manage-locality-groups
title: Create and manage locality groups
description: Manage locality groups in Spanner to control the physical storage layout of your data for performance optimization.
data_source: docs.cloud.google.com
---

> **Note:** This feature is available with the Spanner Enterprise edition and Enterprise Plus edition. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

This page describes how to create and manage Spanner [locality groups](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#locality-groups) . You can use locality groups to define the tiered storage policy for data in your database schema. For information about how tiered storage works, see [Tiered storage](https://docs.cloud.google.com/spanner/docs/tiered-storage) .

## Create a locality group

You can create a locality group without any tiered storage policy, or you can create a locality group to define the storage policy for data in your database schema.

If you create a locality group without a tiered storage policy, the locality group inherits the tiered storage policy of the `default` locality group. If you haven't manually set the storage policy of the `default` locality group, then the storage policy is set to SSD-only.

### Console

1.  Go to the Spanner **Instances** page in the Google Cloud console.

2.  Select the instance in which you want to use tiered storage.

3.  Select the database in which you want to use tiered storage.

4.  In the navigation menu, click **Spanner Studio** .

5.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.

6.  Enter the `CREATE LOCALITY GROUP` DDL statement using [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_locality_group) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#create_locality_group) .

    For example, you can run the following to create a locality group, `separate_storage` , that stores columns in a separate file than the data for the rest of the columns:

    ### GoogleSQL

    ```
    CREATE LOCALITY GROUP separate_storage;
    ```

    ### PostgreSQL

    ```
    CREATE LOCALITY GROUP separate_storage;
    ```

    For example, you can run the following to create a locality group, `ssd_only` , that stores data on SSD storage:

    ### GoogleSQL

    ```
    CREATE LOCALITY GROUP ssd_only OPTIONS (storage='ssd');
    ```

    ### PostgreSQL

    ```
    CREATE LOCALITY GROUP ssd_only STORAGE 'ssd';
    ```

    For example, you can run the following to create a locality group, `hdd_only` , that stores data on HDD storage:

    ### GoogleSQL

    ```
    CREATE LOCALITY GROUP hdd_only OPTIONS (storage='hdd');
    ```

    ### PostgreSQL

    ```
    CREATE LOCALITY GROUP hdd_only STORAGE 'hdd';
    ```

7.  Click **Run** .

### gcloud

To create a locality group with the gcloud CLI command, use [`gcloud spanner databases ddl update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) .

For example, you can run the following to create a locality group, `separate_storage` , that stores columns in a separate file than the data for the rest of the columns:

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE LOCALITY GROUP separate_storage"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE LOCALITY GROUP separate_storage"
```

For example, you can run the following to create a locality group, `ssd_only` , that stores data on SSD:

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE LOCALITY GROUP ssd_only OPTIONS (storage='ssd')"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE LOCALITY GROUP ssd_only STORAGE 'ssd'"
```

For example, you can run the following to create a locality group, `hdd_only` , that stores data on HDD storage:

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE LOCALITY GROUP hdd_only OPTIONS (storage='hdd')"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE LOCALITY GROUP hdd_only STORAGE 'hdd'"
```

### Create an age-based policy for a locality group

A locality group with an age-based policy stores newer data in SSD storage for a specified time. This time is relative to the data's commit timestamp. After the specified time passes, Spanner migrates the data to HDD storage during its normal compaction cycle, which typically occurs over the course of seven days from the specified time. When using an age-based tiered storage policy, the minimum amount of time that data must be stored in SSD before it's moved to HDD storage is one hour.

To create an age-based locality group, use the `CREATE LOCALITY GROUP` DDL statement.

### Console

1.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.

2.  Enter the `CREATE LOCALITY GROUP` DDL statement using [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_locality_group) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#create_locality_group) .

    For example, the following DDL statement creates a locality group, `spill_to_hdd` , that stores data on SSD storage for the first 10 days, and then migrates older data to HDD storage over the normal compaction cycle:

    ### GoogleSQL

    ```
    CREATE LOCALITY GROUP spill_to_hdd
    OPTIONS (storage = 'ssd', ssd_to_hdd_spill_timespan = '10d');
    ```

    ### PostgreSQL

    ```
    CREATE LOCALITY GROUP spill_to_hdd
    STORAGE 'ssd' SSD_TO_HDD_SPILL_TIMESPAN '10d';
    ```

3.  Click **Run** .

### gcloud

To create an age-based locality group with the gcloud CLI command, use [`gcloud spanner databases ddl update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) .

For example, the following DDL statement creates a locality group `spill_to_hdd` that stores data in SSD for the first 10 days, and then migrates older data to HDD over the normal compaction cycle.

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE LOCALITY GROUP spill_to_hdd OPTIONS (storage='ssd', ssd_to_hdd_spill_timespan='10d')"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE LOCALITY GROUP spill_to_hdd STORAGE 'ssd' SSD_TO_HDD_SPILL_TIMESPAN '10d'"
```

## Set a tiered storage policy for your data

After you create your locality group, you can set the tiered storage policy for your data. The tiered storage policy determines the locality group that the data uses. You can set the tiered storage policy at the database, table, column, or secondary index-level. Each database object inherits its tiered storage policy from its parent, unless it's explicitly overridden.

If you create a locality group without a tiered storage policy, the locality group inherits the tiered storage policy of the `default` locality group. If you haven't manually set the storage policy of the `default` locality group, then the storage policy is set to SSD-only.

### Set a database-level locality group

The default tiered storage policy is that all data is stored on SSD storage. You can change the database-level tiered storage policy by altering the `default` locality group. For GoogleSQL-dialect databases, your `ALTER LOCALITY GROUP` DDL statement must have `default` within backticks ( `` `default` `` ). You only need to include the backticks for the `default` locality group.

### Console

1.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.

2.  Enter the `ALTER LOCALITY GROUP` DDL statement using [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_locality_group) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#alter_locality_group) .

    For example, the following DDL statements alter the `default` locality group to use an age-based tiered storage policy. All data in the database is moved to HDD storage after 10 days.

    ### GoogleSQL

    ```
    ALTER LOCALITY GROUP `default` SET OPTIONS (storage = 'ssd', ssd_to_hdd_spill_timespan = '10d');
    ```

    ### PostgreSQL

    ```
    ALTER LOCALITY GROUP "default" STORAGE 'ssd' SSD_TO_HDD_SPILL_TIMESPAN '10d';
    ```

3.  Click **Run** .

### gcloud

To alter the tiered storage policy of the `default` locality group with the gcloud CLI command, use [`gcloud spanner databases ddl update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) .

For example, the following DDL statements alter the `default` locality group to use an age-based tiered storage policy. All data in the database is moved to HDD storage after 10 days.

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="ALTER LOCALITY GROUP \`default\` SET OPTIONS (storage = 'ssd', ssd_to_hdd_spill_timespan = '10d');"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="ALTER LOCALITY GROUP \"default\" STORAGE 'ssd' SSD_TO_HDD_SPILL_TIMESPAN '10d';"
```

### Set a table-level locality group

You can set a table-level tiered storage policy for your data that overrides the database-level tiered storage policy. The table-level tiered storage policy is also applicable to all columns in the table, unless you have set a [column-level override tiered storage policy](https://docs.cloud.google.com/spanner/docs/create-manage-locality-groups#set-storage-policy-column) .

### Console

1.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.

2.  Enter the `CREATE TABLE` DDL statement using [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_table) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#create_table) .

    For example, the following DDL statements create a table, `Singers` , that uses the locality group `ssd_only` :

    ### GoogleSQL

    ```
    CREATE TABLE Singers (
      SingerId   INT64 NOT NULL,
      FirstName  STRING(1024),
      LastName   STRING(1024),
      SingerInfo BYTES(MAX)
    ) PRIMARY KEY (SingerId), OPTIONS (locality_group = 'ssd_only');
    ```

    ### PostgreSQL

    ```
    CREATE TABLE Singers (
      SingerId   bigint PRIMARY KEY,
      FirstName  varchar(1024),
      LastName   varchar(1024),
      SingerInfo bytea
    ) LOCALITY GROUP ssd_only;
    ```

3.  Click **Run** .

### gcloud

To set a table-level tiered storage policy with the gcloud CLI command, use [`gcloud spanner databases ddl update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) .

For example, the following DDL statements create a table, `Singers` , that uses the locality group `ssd_only` .

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE TABLE Singers ( SingerId   INT64 NOT NULL, \
        FirstName  STRING(1024), \
        LastName   STRING(1024), \
        SingerInfo BYTES(MAX) \
        ) PRIMARY KEY (SingerId), OPTIONS (locality_group = 'ssd_only');"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE TABLE Singers ( \
        SingerId bigint PRIMARY KEY, \
        FirstName  varchar(1024), \
        LastName   varchar(1024), \
        SingerInfo bytea \
        ) LOCALITY GROUP ssd_only;"
```

### Set a column-level override tiered storage policy

You can set a column-level override tiered storage policy for your data.

### Console

1.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.

2.  Enter the `CREATE TABLE` DDL statement with a column-level override tiered storage policy using [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_table) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#create_table) .

    For example, the following DDL statements create a `Singers` table that uses the locality group `ssd_only` . However, the `Awards` column overrides this table-level locality group and uses the `spill_to_hdd` locality group as its tiered storage policy:

    ### GoogleSQL

    ```
    CREATE TABLE Singers (
      SingerId   INT64 NOT NULL,
      FirstName  STRING(1024),
      LastName   STRING(1024),
      Awards     ARRAY<STRING(MAX)> OPTIONS (locality_group = 'spill_to_hdd')
    ) PRIMARY KEY (SingerId), OPTIONS (locality_group = 'ssd_only');
    ```

    ### PostgreSQL

    ```
    CREATE TABLE Singers (
      SingerId   bigint PRIMARY KEY,
      FirstName  varchar(1024),
      LastName   varchar(1024),
      Awards     varchar[] LOCALITY GROUP spill_to_hdd
    ) LOCALITY GROUP ssd_only;
    ```

3.  Click **Run** .

### gcloud

To set a column-level override tiered storage policy with the gcloud CLI command, use [`gcloud spanner databases ddl update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) .

For example, the following DDL statements create a `Singers` table that uses the locality group `ssd_only` . However, the `Awards` column overrides this table-level tiered storage policy and uses the `spill_to_hdd` locality group as its tiered storage policy:

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE TABLE Singers ( \
      SingerId   INT64 NOT NULL, \
      FirstName  STRING(1024), \
      LastName   STRING(1024), \
      Awards     ARRAY<STRING(MAX)> OPTIONS (locality_group = 'spill_to_hdd') \
    ) PRIMARY KEY (SingerId), OPTIONS (locality_group = 'ssd_only');" \
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE TABLE Singers ( \
      SingerId   bigint PRIMARY KEY, \
      FirstName  varchar(1024), \
      LastName   varchar(1024), \
      Awards     varchar[] LOCALITY GROUP spill_to_hdd \
    ) LOCALITY GROUP ssd_only;"
```

### Set a secondary index-level override tiered storage policy

You can set a secondary index-level override tiered storage policy for your data.

### Console

1.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.

2.  Enter the `CREATE INDEX` DDL statement with a secondary index-level override tiered storage policy using [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create-index) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#create-index) .

    For example, the following DDL statements create a `Singers` table that uses the locality group `ssd_only` . The database also has a secondary index on all `Singers` in the database by their first and last name. The `SingersByFirstLastName` index overrides the table-level tiered storage policy and uses the `spill_to_hdd` locality group as its tiered storage policy:

    ### GoogleSQL

    ```
    CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName)
    OPTIONS (locality_group = 'spill_to_hdd');
    ```

    ### PostgreSQL

    ```
    CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName)
    LOCALITY GROUP spill_to_hdd;
    ```

3.  Click **Run** .

### gcloud

To set a secondary index-level override tiered storage policy with the gcloud CLI command, use [`gcloud spanner databases ddl update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) .

For example, the following DDL statements create a `Singers` table that uses the locality group `ssd_only` . The database also creates a secondary index on all `Singers` in the database by their first and last name. The `SingersByFirstLastName` index overrides the table-level tiered storage policy and uses the `spill_to_hdd` locality group as its tiered storage policy:

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName) \
    OPTIONS (locality_group = 'spill_to_hdd');"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName) \
    LOCALITY GROUP spill_to_hdd;"
```

## Set a column-level locality group

You can set a column-level locality group for your data even if the locality group doesn't have a tiered storage policy. Reading data from this column is faster than reading data that is grouped with other columns.

### Console

1.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.

2.  Enter the `CREATE TABLE` DDL statement that assigns the column to a locality group using [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_table) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#create_table) .

    For example, the following DDL statements create a `Songs` table with a `LyricsCompressed` column that is stored separately in the `hdd_only` locality group:

    ### GoogleSQL

    ```
    CREATE TABLE Songs (
      SingerId INT64 NOT NULL,
      SongId INT64 NOT NULL,
      Title STRING(MAX),
      Description STRING(MAX),
      LyricsCompressed BYTES(MAX) OPTIONS (locality_group = 'hdd_only')
    ) PRIMARY KEY (SingerId, SongId),
      INTERLEAVE IN PARENT Singers ON DELETE CASCADE,
      OPTIONS (locality_group = 'ssd_only');
    ```

    ### PostgreSQL

    ```
    CREATE TABLE Songs (
      SingerId BIGINT NOT NULL,
      SongId BIGINT NOT NULL,
      Title VARCHAR,
      Description TEXT,
      LyricsCompressed BYTEA LOCALITY GROUP hdd_only,
      PRIMARY KEY (SingerId, SongId)
    ) LOCALITY GROUP ssd_only INTERLEAVE IN PARENT Singers ON DELETE CASCADE;
    ```

3.  Click **Run** .

### gcloud

To set a column-level locality group for your data with the gcloud CLI command, use [`gcloud spanner databases ddl update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) .

For example, the following DDL statements create a `Songs` table with a `LyricsCompressed` column that is stored separately in the `hdd_only` locality group:

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE TABLE Songs ( \
      SingerId INT64 NOT NULL, \
      SongId INT64 NOT NULL, \
      Title STRING(MAX), \
      Description STRING(MAX),
      LyricsCompressed BYTES(MAX) OPTIONS (locality_group = 'hdd_only') \
    ) PRIMARY KEY (SingerId, SongId), \
      INTERLEAVE IN PARENT Singers ON DELETE CASCADE, \
      OPTIONS (locality_group = 'ssd_only');"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="CREATE TABLE Songs ( \
      SingerId BIGINT NOT NULL, \
      SongId BIGINT NOT NULL, \
      Title VARCHAR, \
      Description TEXT, \
      LyricsCompressed BYTEA LOCALITY GROUP hdd_only, \
      PRIMARY KEY (SingerId, SongId) \
    ) LOCALITY GROUP ssd_only INTERLEAVE IN PARENT Singers ON DELETE CASCADE;
```

## Move data between storage options

You can move data between SSD and HDD storage. Moving data can take up to seven days. To migrate data between SSD and HDD faster, you can [manually trigger a major compaction](https://docs.cloud.google.com/spanner/docs/manual-data-compaction) . You can monitor the progress of the move by querying the built-in `SPANNER_SYS.TABLE_SIZES_STATS_1HOUR` table to check HDD and SSD storage usage for each table in your database. You can also monitor your storage usage by using the Cloud Monitoring `spanner.googleapis.com/instance/storage/used_bytes` metric to show the SSD and HDD breakdown for your database or instance. For more information, see [tiered storage observability](https://docs.cloud.google.com/spanner/docs/tiered-storage#observability) .

### Move data from SSD to HDD storage

To move data from SSD to HDD storage, you can create a new locality group with an aged-based tiered storage policy, or change the tiered storage policy of an existing locality group. Spanner moves data during its normal compaction cycle, which typically occurs over the course of seven days. If you're loading data and updating your locality group using one of the following `ALTER` DDL statements, then you must wait up to seven days for the change to the locality group to complete:

### GoogleSQL

- `ALTER TABLE ... SET OPTIONS (locality_group = '')`
- `ALTER TABLE ... ALTER COLUMN ... SET OPTIONS (locality_group = '')`
- `ALTER INDEX ... SET OPTIONS (locality_group = '')`

### PostgreSQL

- `ALTER TABLE ... SET LOCALITY GROUP ...`
- `ALTER TABLE ... ALTER COLUMN ... SET LOCALITY GROUP ...`
- `ALTER INDEX ... SET LOCALITY GROUP ...`

The waiting period isn't applicable if you're creating a new table, column, or index, and adding the locality group as part of the `CREATE` or `ADD COLUMN` syntax.

For more information, see [Create an age-based policy for a locality group](https://docs.cloud.google.com/spanner/docs/create-manage-locality-groups#create-age-based-policy) or [Change the storage option](https://docs.cloud.google.com/spanner/docs/create-manage-locality-groups#change-storage-option) .

### Move data from HDD to SSD storage

To move data from HDD to SSD storage, you can [change the storage option](https://docs.cloud.google.com/spanner/docs/create-manage-locality-groups#change-storage-option) of an existing locality group or [alter the locality group used by the table](https://docs.cloud.google.com/spanner/docs/create-manage-locality-groups#alter-table-locality-group) . Spanner moves data during its normal compaction cycle, which typically occurs over the course of seven days. If you're loading data and updating your locality group using one of the following `ALTER` DDL statements, then you must wait up to seven days for the change to the locality group to complete:

### GoogleSQL

- `ALTER TABLE ... SET OPTIONS (locality_group = '')`
- `ALTER TABLE ... ALTER COLUMN ... SET OPTIONS (locality_group = '')`
- `ALTER INDEX ... SET OPTIONS (locality_group = '')`

### PostgreSQL

- `ALTER TABLE ... SET LOCALITY GROUP ...`
- `ALTER TABLE ... ALTER COLUMN ... SET LOCALITY GROUP ...`
- `ALTER INDEX ... SET LOCALITY GROUP ...`

The waiting period isn't applicable if you're creating a new table, column, or index, and adding the locality group as part of the `CREATE` or `ADD COLUMN` syntax.

#### Alter the locality group used by a table

You can alter the locality group used by a table by setting a new or different locality group in the table options.

### Console

1.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.

2.  Enter the `ALTER TABLE` DDL statement that changes the locality group used by the table using [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_table) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#alter_table) .

    For example, the following DDL statements alter the locality group used by the table, `Singers` , to `spill_to_hdd` :

    ### GoogleSQL

    ```
    ALTER TABLE Singers SET OPTIONS (locality_group = 'spill_to_hdd');
    ```

    ### PostgreSQL

    ```
    ALTER TABLE Singers SET LOCALITY GROUP spill_to_hdd;
    ```

3.  Click **Run** .

### gcloud

To alter the locality group used by a table with the gcloud CLI command, use [`gcloud spanner databases ddl update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) .

For example, the following DDL statements alter the locality group used by the table, `Singers` , to `spill_to_hdd` :

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="ALTER TABLE Singers SET OPTIONS(locality_group = 'spill_to_hdd');"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="ALTER TABLE Singers SET LOCALITY GROUP spill_to_hdd;"
```

#### Alter the locality group used by a table's column

You can alter the locality group used by a table's column by setting the locality group in the column options.

### Console

1.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.

2.  Enter the `ALTER TABLE` DDL statement that changes the locality group used by the table using [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_table) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#alter_table) .

    For example, the following DDL statements alter the locality group used by the table's column, `LastName` , to `spill_to_hdd` :

    ### GoogleSQL

    ```
    ALTER TABLE Singers
    ALTER COLUMN LastName SET OPTIONS(locality_group = 'spill_to_hdd');
    ```

    ### PostgreSQL

    ```
    ALTER TABLE Singers
    ALTER COLUMN LastName SET LOCALITY GROUP spill_to_hdd;
    ```

3.  Click **Run** .

### gcloud

To alter the locality group used by a table with the gcloud CLI command, use [`gcloud spanner databases ddl update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) .

For example, the following DDL statements alter the locality group used by the table's column, `LastName` , to `spill_to_hdd` :

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="ALTER TABLE Singers ALTER COLUMN LastName SET OPTIONS(locality_group = 'spill_to_hdd');"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="ALTER TABLE Singers ALTER COLUMN SET LOCALITY GROUP spill_to_hdd;"
```

## Alter a locality group

You can alter a locality group by [changing its storage option](https://docs.cloud.google.com/spanner/docs/create-manage-locality-groups#change-storage-option) or [changing its age-based policy](https://docs.cloud.google.com/spanner/docs/create-manage-locality-groups#change-age-based-policy) . Spanner moves data during its normal compaction cycle, which typically occurs over the course of seven days. If you're loading data and updating your locality group using `ALTER LOCALITY GROUP` , then you must wait up to seven days for existing data to migrate. The updated setting applies to all new data written to that locality group immediately.

### Change the storage option

You can change the storage option of a locality group from SSD to HDD or HDD to SSD.

### Console

1.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.

2.  Enter the `ALTER LOCALITY GROUP` DDL statement using [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_locality_group) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#alter_locality_group) .

    For example, the following DDL statements change the storage option of the locality group, `separate_storage` , to HDD:

    ### GoogleSQL

    ```
    ALTER LOCALITY GROUP separate_storage SET OPTIONS (storage='hdd');
    ```

    ### PostgreSQL

    ```
    ALTER LOCALITY GROUP separate_storage STORAGE 'hdd';
    ```

3.  Click **Run** .

### gcloud

To change the storage option of a locality group with the gcloud CLI command, use [`gcloud spanner databases ddl update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) .

For example, the following DDL statements change the storage option of the locality group, `separate_storage` , to HDD:

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="ALTER LOCALITY GROUP separate_storage SET OPTIONS (storage = 'hdd');"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="ALTER LOCALITY GROUP separate_storage STORAGE 'hdd';"
```

### Change the age-based policy

You can change the age-based policy of a locality group by extending or shortening the time that data is stored in SSD before it's moved to HDD storage.

### Console

1.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.

2.  Enter the `ALTER LOCALITY GROUP` DDL statement using [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_locality_group) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#alter_locality_group) .

    For example, the following DDL statements change the age-based policy of the locality group, `spill_to_hdd` , by extending the amount of time that data is stored in SSD to 20 days:

    ### GoogleSQL

    ```
    ALTER LOCALITY GROUP spill_to_hdd SET OPTIONS (ssd_to_hdd_spill_timespan = '20d');
    ```

    ### PostgreSQL

    ```
    ALTER LOCALITY GROUP spill_to_hdd SSD_TO_HDD_SPILL_TIMESPAN '20d';
    ```

3.  Click **Run** .

### gcloud

To change the age-based policy of a locality group with the gcloud CLI command, use [`gcloud spanner databases ddl update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) .

For example, the following DDL statements change the age-based policy of the locality group, `spill_to_hdd` , by extending the amount of time that data is stored in SSD to 20 days:

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="ALTER LOCALITY GROUP spill_to_hdd SET OPTIONS (ssd_to_hdd_spill_timespan = '20d');"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="ALTER LOCALITY GROUP spill_to_hdd SSD_TO_HDD_SPILL_TIMESPAN '20d';"
```

## Delete a locality group

You can't delete a locality group if it contains data. You must first move all data that's in the locality group to another locality group. For more information, see [Alter the locality group used by the table](https://docs.cloud.google.com/spanner/docs/create-manage-locality-groups#alter-table-locality-group) .

### Console

1.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.

2.  Enter the `DROP LOCALITY GROUP` DDL statement using [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#drop_locality_group) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#drop_locality_group) .

    For example, you can run the following to drop a locality group `ssd_only` :

    ### GoogleSQL

    ```
    DROP LOCALITY GROUP ssd_only;
    ```

    ### PostgreSQL

    ```
    DROP LOCALITY GROUP ssd_only;
    ```

3.  Click **Run** .

### gcloud

To drop a locality group with the gcloud CLI command, use [`gcloud spanner databases ddl update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) .

For example, to drop the locality group `ssd_only` , run:

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="DROP LOCALITY GROUP ssd_only"
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
  --instance=test-instance \
  --ddl="DROP LOCALITY GROUP ssd_only"
```

## What's next

- Learn more about [tiered storage](https://docs.cloud.google.com/spanner/docs/tiered-storage) .
- Learn more about [locality groups](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#locality-groups) .
- Learn more about [optimizing queries with timestamp predicate pushdown](https://docs.cloud.google.com/spanner/docs/sql-best-practices#optimize-timestamp-predicate-pushdown) .
