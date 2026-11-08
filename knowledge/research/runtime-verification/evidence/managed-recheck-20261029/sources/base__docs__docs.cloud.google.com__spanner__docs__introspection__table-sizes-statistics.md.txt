---
name: documents/docs.cloud.google.com/spanner/docs/introspection/table-sizes-statistics
uri: https://docs.cloud.google.com/spanner/docs/introspection/table-sizes-statistics
title: Table sizes statistics
description: Monitor table and index sizes in Spanner using introspection tools to track storage growth and plan capacity.
data_source: docs.cloud.google.com
---

Spanner provides the following built-in tables:

- `SPANNER_SYS.TABLE_SIZES_STATS_1HOUR` : lists the sizes of your tables and indexes within your databases.
- `SPANNER_SYS.TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR` : lists the sizes of your tables and indexes within your databases for each locality group.

The table size is in bytes. Table sizes include data versions. You can use theses built-in tables to monitor your table and index sizes over time. You can also monitor the sizes of your indexes as you create, delete, and you modify them (as you insert more rows into the index or when you add new columns to it). Additionally, you can also look at the sizes of your change stream tables.

Database storage can be monitored with the [Total database storage metric](https://docs.cloud.google.com/spanner/docs/storage-utilization) . You can see the breakdown of the database storage with `SPANNER_SYS.TABLE_SIZES_STATS_1HOUR` and `SPANNER_SYS.TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR` .

> **Note:** These built-in tables provide a historical perspective of the sizes of your tables and indexes. It is not for real-time monitoring.

## Access table sizes statistics

> **Note:** Spanner Studio (formerly labeled **Query** in the Google Cloud console) supports SQL, DML, and DDL operations in a single editor. For more information, see [Manage your data using the Google Cloud console](https://docs.cloud.google.com/spanner/docs/manage-data-using-console) .

Spanner provides the table sizes statistics in the `SPANNER_SYS` schema. You can use the following ways to access `SPANNER_SYS` data:

- A database's Spanner Studio page in the Google Cloud console.

- The `gcloud spanner databases execute-sql` command.

- The [`executeSql`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases.sessions/executeSql) or the [`executeStreamingSql`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases.sessions/executeStreamingSql) method.

The following single read methods that Spanner provides don't support `SPANNER_SYS` :

- Performing a strong read from a single row or multiple rows in a table.
- Performing a stale read from a single row or multiple rows in a table.
- Reading from a single row or multiple rows in a secondary index.

## TABLE_SIZES_STATS_1HOUR

`SPANNER_SYS.TABLE_SIZES_STATS_1HOUR` contains the sizes of all the tables in your database, sorted by `interval_end` . The intervals are based on clock times, ending on the hour. Internally, every 5 minutes, Spanner collects data from all servers and then makes the data available in the `TABLE_SIZES_STATS_1HOUR` table shortly thereafter. The data is then averaged per every clock hour. For example, at 11:59:30 AM, `TABLE_SIZES_STATS_1HOUR` shows the average table sizes from the interval of 10:00:00 AM - 10:59:59 AM.

### Table schema

| Column name      | Type        | Description                                                   |
|------------------|-------------|---------------------------------------------------------------|
| `INTERVAL_END`   | `TIMESTAMP` | End of time interval in which the table sizes were collected. |
| `TABLE_NAME`     | `STRING`    | Name of the table or the index.                               |
| `USED_BYTES`     | `FLOAT64`   | Table size in bytes.                                          |
| `USED_SSD_BYTES` | `FLOAT64`   | SSD storage used by table in bytes.                           |
| `USED_HDD_BYTES` | `FLOAT64`   | HDD storage used by table in bytes.                           |

### Example queries

This section includes several example SQL statements that retrieve aggregate table sizes statistics. You can run these SQL statements using the [client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) , the [gcloud spanner](https://docs.cloud.google.com/spanner/docs/gcloud-spanner#execute_sql_statements) , or the [Google Cloud console](https://docs.cloud.google.com/spanner/docs/create-query-database-console#run_a_query) .

#### Query 4 largest tables and indexes for the most recent interval

The following query returns the 4 largest tables and indexes for the most recent interval:

```
    SELECT interval_end,
          table_name,
          used_bytes
    FROM spanner_sys.table_sizes_stats_1hour
    WHERE interval_end = (
          SELECT MAX(interval_end)
          FROM spanner_sys.table_sizes_stats_1hour)
    ORDER BY used_bytes DESC
    LIMIT 4;
  
```

##### Query output

| interval_end                | table_name       | used_bytes |
|-----------------------------|------------------|------------|
| `2022-11-15 13:00:00-07:00` | `order_item`     | `60495552` |
| `2022-11-15 13:00:00-07:00` | `orders`         | `13350000` |
| `2022-11-15 13:00:00-07:00` | `item_inventory` | `2094549`  |
| `2022-11-15 13:00:00-07:00` | `customer`       | `870000`   |

#### Query size trend for a specific table or index for the last 24 hours

The following query returns the size of the table over the last 24 hours:

### GoogleSQL

```
SELECT interval_end, used_bytes
  FROM spanner_sys.table_sizes_stats_1hour
WHERE interval_end > TIMESTAMP_ADD(CURRENT_TIMESTAMP(), INTERVAL -24 HOUR)
  AND table_name = "table_name"
ORDER BY interval_end DESC;
```

Where:

- ` ``table_name`` ` must be an existing table or index in the database.

### PostgreSQL

```
SELECT interval_end, used_bytes
  FROM spanner_sys.table_sizes_stats_1hour
WHERE interval_end > spanner.timestamptz_subtract(now(), '24 HOUR')
  AND table_name = 'table_name'
ORDER BY interval_end DESC;
```

Where:

- ` ``table_name`` ` must be an existing table or index in the database.

##### Query output

| interval_end                | used_bytes |
|-----------------------------|------------|
| `2022-11-15 13:00:00-07:00` | `13350000` |
| `2022-11-15 12:00:00-07:00` | `13350000` |
| `2022-11-15 11:00:00-07:00` | `13350000` |
| `2022-11-15 10:00:00-07:00` | `13350000` |
| `2022-11-15 09:00:00-07:00` | `13350000` |
| `2022-11-15 08:00:00-07:00` | `12350000` |
| `2022-11-15 07:00:00-07:00` | `12350000` |
| `2022-11-15 06:00:00-07:00` | `12350000` |
| `2022-11-15 05:00:00-07:00` | `11350000` |
| `2022-11-15 04:00:00-07:00` | `11350000` |
| `2022-11-15 03:00:00-07:00` | `11350000` |
| `2022-11-15 02:00:00-07:00` | `11350000` |
| `2022-11-15 01:00:00-07:00` | `11350000` |
| `2022-11-15 00:00:00-07:00` | `10350000` |
| `2022-11-14 23:00:00-07:00` | `10350000` |
| `2022-11-14 22:00:00-07:00` | `10350000` |
| `2022-11-14 21:00:00-07:00` | `10350000` |
| `2022-11-14 20:00:00-07:00` | `10350000` |
| `2022-11-14 19:00:00-07:00` | `10350000` |
| `2022-11-14 18:00:00-07:00` | `10350000` |
| `2022-11-14 17:00:00-07:00` | `10350000` |
| `2022-11-14 16:00:00-07:00` | `10350000` |
| `2022-11-14 15:00:00-07:00` | `10350000` |
| `2022-11-14 14:00:00-07:00` | `10350000` |
| `2022-11-14 13:00:00-07:00` | `10350000` |

## TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR

`SPANNER_SYS.TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR` contains the sizes of all the tables in your database, sorted by `interval_end` , for each locality group. The intervals are based on clock times, ending on the hour. Internally, every 5 minutes, Spanner collects data from all servers and then makes the data available in the `TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR` table shortly thereafter. The data is then averaged per every clock hour. For example, at 11:59:30 AM, `TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR` shows the average table sizes for each locality group from the interval of 10:00:00 AM - 10:59:59 AM.

### Table schema

| Column name      | Type        | Description                                                   |
|------------------|-------------|---------------------------------------------------------------|
| `INTERVAL_END`   | `TIMESTAMP` | End of time interval in which the table sizes were collected. |
| `TABLE_NAME`     | `STRING`    | Name of the table or the index.                               |
| `LOCALITY_GROUP` | `STRING`    | Name of the locality group.                                   |
| `USED_BYTES`     | `FLOAT64`   | Table size in bytes.                                          |
| `USED_SSD_BYTES` | `FLOAT64`   | SSD storage used by table in bytes.                           |
| `USED_HDD_BYTES` | `FLOAT64`   | HDD storage used by table in bytes.                           |

## Data retention

At a minimum, Spanner keeps data for `SPANNER_SYS.TABLE_SIZES_STATS_1HOUR` for intervals covering the previous 30 days.

> **Note:** You can't prevent Spanner from collecting table size statistics. To delete the data in the statistics table, you must delete the database associated with the table or wait until Spanner removes the data automatically. The retention period for the table is fixed. If you want to keep statistics for longer periods of time, we recommend that you periodically copy data out of this table.

## What's next

- Learn about other [Introspection tools](https://docs.cloud.google.com/spanner/docs/introspection) .
- Learn about other information Spanner stores for each database in the database's [information schema](https://docs.cloud.google.com/spanner/docs/information-schema) tables.
- Learn more about [SQL best practices](https://docs.cloud.google.com/spanner/docs/sql-best-practices) for Spanner.
- Learn more about [Investigating high CPU utilization](https://docs.cloud.google.com/spanner/docs/introspection/investigate-cpu-utilization) .
