---
name: documents/docs.cloud.google.com/spanner/docs/secondary-indexes
uri: https://docs.cloud.google.com/spanner/docs/secondary-indexes
title: Secondary indexes
description: Use secondary indexes in Spanner to improve query performance and efficiency.
data_source: docs.cloud.google.com
---

In a Spanner database, Spanner automatically creates an index for each table's primary key. For example, you don't need to do anything to index the primary key of `Singers` , because it's automatically indexed for you.

You can also create *secondary indexes* for other columns. Adding a secondary index on a column makes it more efficient to look up data in that column. For example, if you need to quickly look up an album by title, you should create a secondary index on `AlbumTitle` , so that Spanner does not need to scan the entire table.

If the lookup in the previous example is done within a read-write transaction, then the more efficient lookup also avoids holding locks on the entire table, which allows concurrent inserts and updates to the table for rows outside of the `AlbumTitle` lookup range.

In addition to the benefits they bring to lookups, secondary indexes can also help Spanner execute [scans](https://docs.cloud.google.com/spanner/docs/query-execution-operators#scan) more efficiently, enabling *index scans* rather than full table scans.

Spanner stores the following data in each secondary index:

- All key columns from the base table
- All columns that are included in the index
- All columns specified in the optional [`STORING` clause (GoogleSQL-dialect databases) or `INCLUDE` clause (PostgreSQL-dialect databases)](https://docs.cloud.google.com/spanner/docs/secondary-indexes#storing-clause) of the index definition.

Over time, Spanner analyzes your tables to ensure that your secondary indexes are used for the appropriate queries.

## Add a secondary index

The most efficient time to add a secondary index is when you create the table. To create a table and its indexes at the same time, send the DDL statements for the new table and the new indexes in a single request to Spanner.

In Spanner, you can also add a new secondary index to an existing table while the database continues to serve traffic. Like any other schema changes in Spanner, adding an index to an existing database does not require taking the database offline and does not lock entire columns or tables.

Whenever a new index is added to an existing table, Spanner automatically *backfills* , or populates, the index to reflect an up-to-date view of the data being indexed. Spanner manages this backfill process for you, and the process runs in the background using node resources at low priority. Index backfill speed adapts to changing node resources during index creation, and backfilling doesn't significantly affect the performance of the database.

Index creation can take from several minutes to many hours. Because index creation is a schema update, it is bound by the same [performance constraints](https://docs.cloud.google.com/spanner/docs/schema-updates#performance) as any other schema update. The time needed to create a secondary index depends on several factors:

- The size of the dataset
- The compute capacity of the instance
- The load on the instance

To view the progress made for an index backfill process, refer to the [progress section](https://docs.cloud.google.com/spanner/docs/secondary-indexes#index-progress) .

Be aware that using the [commit timestamp](https://docs.cloud.google.com/spanner/docs/commit-timestamp) column as the first part of the secondary index can [create hotspots](https://docs.cloud.google.com/spanner/docs/schema-design#primary-key-prevent-hotspots) and reduce write performance.

> **Note:** If you are adding many secondary indexes to a database, follow the [guidance for large schema updates](https://docs.cloud.google.com/spanner/docs/schema-updates#large-updates) when you create the indexes.

Use the `CREATE INDEX` statement to define a secondary index in your schema. Here are some examples:

To index all `Singers` in the database by their first and last name:

### GoogleSQL

```
CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName);
```

### PostgreSQL

```
CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName);
```

To create an index of all `Songs` in the database by the value of `SongName` :

### GoogleSQL

```
CREATE INDEX SongsBySongName ON Songs(SongName);
```

### PostgreSQL

```
CREATE INDEX SongsBySongName ON Songs(SongName);
```

To index only the songs for a particular singer, use the [`INTERLEAVE IN`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create-index-interleave) clause to interleave the index in the table `Singers` :

### GoogleSQL

```
CREATE INDEX SongsBySingerSongName ON Songs(SingerId, SongName),
    INTERLEAVE IN Singers;
```

### PostgreSQL

```
CREATE INDEX SongsBySingerSongName ON Songs(SingerId, SongName)
    INTERLEAVE IN Singers;
```

To index only the songs on a particular album:

### GoogleSQL

```
CREATE INDEX SongsBySingerAlbumSongName ON Songs(SingerId, AlbumId, SongName),
    INTERLEAVE IN Albums;
```

### PostgreSQL

```
CREATE INDEX SongsBySingerAlbumSongName ON Songs(SingerId, AlbumId, SongName)
    INTERLEAVE IN Albums;
```

To index by descending order of `SongName` :

### GoogleSQL

```
CREATE INDEX SongsBySingerAlbumSongNameDesc ON Songs(SingerId, AlbumId, SongName DESC),
    INTERLEAVE IN Albums;
```

### PostgreSQL

```
CREATE INDEX SongsBySingerAlbumSongNameDesc ON Songs(SingerId, AlbumId, SongName DESC)
    INTERLEAVE IN Albums;
```

Note that the previous `DESC` annotation applies only to `SongName` . To index by descending order of other index keys, annotate them with `DESC` as well: `SingerId DESC, AlbumId DESC` .

Also note that `PRIMARY_KEY` is a reserved word and cannot be used as the name of an index. It is the name given to the [pseudo-index](https://docs.cloud.google.com/spanner/docs/information-schema#indexes) that is created when a table with PRIMARY KEY specification is created

For more details and best practices for choosing non-interleaved indexes and interleaved indexes, see [Choose between interleaved and global indexes](https://docs.cloud.google.com/spanner/docs/secondary-indexes#choose-index-type) , [Index options](https://docs.cloud.google.com/spanner/docs/whitepapers/optimizing-schema-design#index-options) , and [Use an interleaved index on a column whose value monotonically increases or decreases](https://docs.cloud.google.com/spanner/docs/schema-design#creating-indexes) .

## Indexes and Interleaving

Spanner indexes can be interleaved with other tables in order to colocate index rows with those of another table. Similar to Spanner table interleaving, the primary key columns of the index's parent must be a prefix of the indexed columns, matching in type and sort order. Unlike interleaved tables, column name matching is not required. Each row of an interleaved index is physically stored together with the associated parent row.

For example, consider the following schema:

```
CREATE TABLE Singers (
  SingerId   INT64 NOT NULL,
  FirstName  STRING(1024),
  LastName   STRING(1024),
  SingerInfo PROTO<Singer>(MAX)
) PRIMARY KEY (SingerId);

CREATE TABLE Albums (
  SingerId     INT64 NOT NULL,
  AlbumId      INT64 NOT NULL,
  AlbumTitle   STRING(MAX),
  PublisherId  INT64 NOT NULL
) PRIMARY KEY (SingerId, AlbumId),
  INTERLEAVE IN PARENT Singers ON DELETE CASCADE;

CREATE TABLE Songs (
  SingerId     INT64 NOT NULL,
  AlbumId      INT64 NOT NULL,
  TrackId      INT64 NOT NULL,
  PublisherId  INT64 NOT NULL,
  SongName     STRING(MAX)
) PRIMARY KEY (SingerId, AlbumId, TrackId),
  INTERLEAVE IN PARENT Albums ON DELETE CASCADE;

CREATE TABLE Publishers (
  Id            INT64 NOT NULL,
  PublisherName STRING(MAX)
) PRIMARY KEY (Id);
```

To index all `Singers` in the database by their first and last name, you must create an index. Here's how to define the index `SingersByFirstLastName` :

```
CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName);
```

If you want to create an index of `Songs` on `(SingerId, AlbumId, SongName)` , you could do the following:

```
CREATE INDEX SongsBySingerAlbumSongName
    ON Songs(SingerId, AlbumId, SongName);
```

Or you could create an index that is interleaved with an ancestor of `Songs` , such as the following:

```
CREATE INDEX SongsBySingerAlbumSongName
    ON Songs(SingerId, AlbumId, SongName),
    INTERLEAVE IN Albums;
```

Further, you could also create an index of `Songs` on `(PublisherId, SingerId, AlbumId, SongName)` that's interleaved with a table that isn't an ancestor of `Songs` , like `Publishers` . Note that the primary key for the `Publishers` table ( `id` ), is not a prefix of the indexed columns in the following example. This is still allowed because `Publishers.Id` and `Songs.PublisherId` share the same type, sort order, and nullability.

```
CREATE INDEX SongsByPublisherSingerAlbumSongName
    ON Songs(PublisherId, SingerId, AlbumId, SongName),
    INTERLEAVE IN Publishers;
```

## Choose between interleaved and global indexes

When you create a secondary index in Spanner, you define it as either a *global index* (the default, non-interleaved index) or an *interleaved index* . Choosing the optimal index type is critical for read and write performance.

### Index comparison

| Feature              | Interleaved index                                                                                                                                                                                                                                                                 | Global index                                                                         |
|----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------|
| **Storage location** | Co-located with the parent row in the same split (row tree).                                                                                                                                                                                                                      | Stored in a separate root table, which can be split independently of the base table. |
| **Key requirements** | The primary key of the parent table must be a prefix of the index keys.                                                                                                                                                                                                           | No restrictions on the index keys.                                                   |
| **Write overhead**   | **Low.** When interleaved in an ancestor of the base table, updates are usually local to the split, avoiding two-phase commits across splits. If interleaved in a table that is not an ancestor, write overhead is higher because updates might require distributed transactions. | **Higher.** Updates often span different splits, requiring distributed transactions. |
| **Best for**         | Queries scoped to a single parent entity—for example, "find photos for user ` USER_NAME ` ."                                                                                                                                                                                      | Queries that search across the entire database—for example, "find user by email".    |

### Why interleaved indexes are usually preferred

In a distributed database like Spanner, data locality is key to performance. Interleaved indexes are usually preferred because they preserve data locality:

- **Efficient reads:** If your queries always filter by a parent entity's ID, an interleaved index allows Spanner to satisfy the query by scanning only the split that contains that parent entity, rather than consulting multiple splits.
- **Efficient writes:** When you update a row in a table, Spanner must also update its secondary indexes. If you use a global index, the index entry likely resides in a different split than the base row. This forces Spanner to perform a distributed transaction (two-phase commit) across splits, which increases latency and CPU usage. If an interleaved index is interleaved in an ancestor of the base table, the index entry is stored in the same split as the base row, making the write a local operation.

### Decision guide

Use the following guidelines to choose the optimal index type for your use case:

#### When to use an interleaved index

Use an interleaved index when one or more of the following are true:

- Your queries always include a filter on the parent table's primary key—for example, `WHERE UserId = @userId AND ...` .
- The parent table's primary key columns can naturally be used as a prefix for your index keys.
- You want to minimize the write latency and CPU overhead of index maintenance.

*Example:*

```
-- Indexing albums by title for a specific singer
CREATE INDEX AlbumsBySingerTitle ON Albums(SingerId, AlbumTitle),
  INTERLEAVE IN Singers;
```

#### When to use a global index

Use a global index when one or more of the following are true:

- You need to query data across the entire database without filtering by a parent entity—for example, a global search.
- The columns you want to index don't share a prefix with any parent table.

*Example:*

```
-- Indexing all albums globally by title. For example, for a global search catalog.
CREATE INDEX AlbumsByTitle ON Albums(AlbumTitle);
```

## Check index backfill progress

### Console

1.  In the Spanner navigation menu, click the **Operations** tab. The **Operations** page shows a list of running operations.

2.  Find the backfill operation in the list. If it's still running, the progress indicator in the **End time** column shows the percentage of the operation that is complete, as shown in the following image:

    ![Progress indicator showing progress is at 98%](https://docs.cloud.google.com/static/spanner/docs/images/ddl-operation-progress.png)

### gcloud

Use [`gcloud spanner operations describe`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/operations/describe) to check the progress of an operation.

1.  Get the operation ID:

    ```
    gcloud spanner operations list --instance=INSTANCE-NAME \
    --database=DATABASE-NAME --type=DATABASE_UPDATE_DDL
    ```

    Replace the following:

    - ` INSTANCE-NAME ` with the Spanner instance name.
    - ` DATABASE-NAME ` with the name of the database.

    Usage notes:

    - To limit the list, specify the `--filter` flag. For example:

      - `--filter="metadata.name:example-db"` only lists the operations on a specific database.
      - `--filter="error:*"` only lists the backup operations that failed.

      For information on filter syntax, see [gcloud topic filters](https://docs.cloud.google.com/sdk/gcloud/reference/topic/filters) . For information on filtering backup operations, see the `filter` field in [ListBackupOperationsRequest](https://docs.cloud.google.com/spanner/docs/reference/rpc/google.spanner.admin.database.v1#listbackupoperationsrequest) .

    - The `--type` flag is not case sensitive.

    The output looks similar to the following:

    ```
    OPERATION_ID     STATEMENTS                                                                                          DONE   @TYPE
    _auto_op_123456  CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName)                                 False  UpdateDatabaseDdlMetadata
                    CREATE INDEX SongsBySingerAlbumSongName ON Songs(SingerId, AlbumId, SongName), INTERLEAVE IN Albums
    _auto_op_234567                                                                                                      True   CreateDatabaseMetadata
    ```

2.  Run [`gcloud spanner operations describe`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/operations/describe) :

    ```
    gcloud spanner operations describe \
    --instance=INSTANCE-NAME \
    --database=DATABASE-NAME \
    projects/PROJECT-NAME/instances/INSTANCE-NAME/databases/DATABASE-NAME/operations/OPERATION_ID
    ```

    Replace the following:

    - ` INSTANCE-NAME ` : The Spanner instance name.
    - ` DATABASE-NAME ` : The Spanner database name.
    - ` PROJECT-NAME ` : The project name.
    - ` OPERATION-ID ` : The operation ID of the operation that you want to check.

    The `progress` section in the output shows the percentage of the operation that's complete. The output looks similar to the following::

    ```
    done: true
    ...
      progress:
      - endTime: '2021-01-22T21:58:42.912540Z'
        progressPercent: 100
        startTime: '2021-01-22T21:58:11.053996Z'
      - progressPercent: 67
        startTime: '2021-01-22T21:58:11.053996Z'
    ...
    ```

### REST v1

Get the operation ID:

```
gcloud spanner operations list --instance=INSTANCE-NAME 

  --database=DATABASE-NAME --type=DATABASE_UPDATE_DDL
```

Replace the following:

- ` INSTANCE-NAME ` with the Spanner instance name.
- ` DATABASE-NAME ` with the name of the database.

Before using any of the request data, make the following replacements:

- ` PROJECT-ID ` : the project ID.
- ` INSTANCE-ID ` : the instance ID.
- ` DATABASE-ID ` : the database ID.
- ` OPERATION-ID ` : the operation ID.

HTTP method and URL:

```
GET https://spanner.googleapis.com/v1/projects/PROJECT-ID/instances/INSTANCE-ID/databases/DATABASE-ID/operations/OPERATION-ID
```

To send your request, expand one of these options:

#### curl (Linux, macOS, or Cloud Shell)

> **Note:** The following command assumes that you have logged in to the `gcloud` CLI with your user account by running [`gcloud init`](https://docs.cloud.google.com/sdk/gcloud/reference/init) or [`gcloud auth login`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) , or by using [Cloud Shell](https://docs.cloud.google.com/shell/docs) , which automatically logs you into the `gcloud` CLI . You can check the currently active account by running [`gcloud auth list`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/list) .

Execute the following command:

```
curl -X GET \
     -H "Authorization: Bearer $(gcloud auth print-access-token)" \
     "https://spanner.googleapis.com/v1/projects/PROJECT-ID/instances/INSTANCE-ID/databases/DATABASE-ID/operations/OPERATION-ID"
```

#### PowerShell (Windows)

> **Note:** The following command assumes that you have logged in to the `gcloud` CLI with your user account by running [`gcloud init`](https://docs.cloud.google.com/sdk/gcloud/reference/init) or [`gcloud auth login`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) . You can check the currently active account by running [`gcloud auth list`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/list) .

Execute the following command:

```
$cred = gcloud auth print-access-token
$headers = @{ "Authorization" = "Bearer $cred" }

Invoke-WebRequest `
    -Method GET `
    -Headers $headers `
    -Uri "https://spanner.googleapis.com/v1/projects/PROJECT-ID/instances/INSTANCE-ID/databases/DATABASE-ID/operations/OPERATION-ID" | Select-Object -Expand Content
```

You should receive a JSON response similar to the following:

```
{
...
    "progress": [
      {
        "progressPercent": 100,
        "startTime": "2023-05-27T00:52:27.366688Z",
        "endTime": "2023-05-27T00:52:30.184845Z"
      },
      {
        "progressPercent": 100,
        "startTime": "2023-05-27T00:52:30.184845Z",
        "endTime": "2023-05-27T00:52:40.750959Z"
      }
    ],
...
  "done": true,
  "response": {
    "@type": "type.googleapis.com/google.protobuf.Empty"
  }
}
```

For `gcloud` and REST, you can find the progress of each index backfill statement in the `progress` section. For each statement in the statement array, there is a corresponding field in the progress array. This progress array order corresponds to the order of the statements array. Once available, the `startTime` , `progressPercent` , and `endTime` fields are populated accordingly. Note that the output doesn't show an estimated time for when the backfill progress will complete.

If the operation takes too long, you can cancel it. For more information, see [Cancel index creation](https://docs.cloud.google.com/spanner/docs/secondary-indexes#cancel-create) .

### Scenarios when viewing index backfill progress

There are different scenarios that you can encounter when trying to check the progress of an index backfill. Index creation statements that require an index backfill are part of schema update operations, and there can be several statements that are part of a schema update operation.

The first scenario is the simplest, which is when the index creation statement is the first statement in the schema update operation. Since the index creation statement is the first statement, it is the first one processed and executed due to the [order of execution](https://docs.cloud.google.com/spanner/docs/schema-updates#order_of_execution_of_statements_in_batches) . Immediately, the `startTime` field of the index creation statement will populate with the start time of the schema update operation. Next, the index creation statement's `progressPercent` field is populated when the progress of the index backfill is over 0%. Finally, the `endTime` field is populated once the statement is committed.

The second scenario is when the index creation statement is not the first statement in the schema update operation. No fields related to the index creation statement will populate until the previous statement(s) have been committed due to the [order of execution](https://docs.cloud.google.com/spanner/docs/schema-updates#order_of_execution_of_statements_in_batches) . Similar to the previous scenario, once the previous statements are committed, the `startTime` field of the index creation statement populates first, followed by the `progressPercent` field. Lastly, the `endTime` field populates once the statement finishes committing.

## Cancel index creation

You can use the Google Cloud CLI to cancel index creation. To retrieve a list of schema-update operations for a Spanner database, use the [`gcloud spanner operations list`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/operations/list) command, and include the `--filter` option:

```
gcloud spanner operations list \
    --instance=INSTANCE \
    --database=DATABASE \
    --filter="@TYPE:UpdateDatabaseDdlMetadata"
```

Find the `OPERATION_ID` for the operation you want to cancel, then use the [`gcloud spanner operations cancel`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/operations/cancel) command to cancel it:

```
gcloud spanner operations cancel OPERATION_ID \
    --instance=INSTANCE \
    --database=DATABASE
```

> **Note:** When you cancel an ongoing operation to create more than one index, only indexes that are yet to be created are canceled. The indexes already created are not dropped by the cancellation.

## View existing indexes

To view information about existing indexes in a database, you can use the Google Cloud console or the Google Cloud CLI:

### Console

1.  Go to the Spanner **Instances** page in the Google Cloud console.

2.  Click the name of the instance you want to view.

3.  In the left pane, click the database you want to view, then click the table you want to view.

4.  Click the **Indexes** tab. The Google Cloud console shows a list of indexes.

5.  Optional: To get details about an index, such as the columns that it includes, click the name of the index.

### gcloud

Use the `gcloud spanner databases ddl describe` command:

```
    gcloud spanner databases ddl describe DATABASE \
        --instance=INSTANCE
```

The gcloud CLI prints the [Data Definition Language (DDL)](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language) statements to create the database's tables and indexes. The [`CREATE INDEX`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_index) statements describe the existing indexes. For example:

```
    --- |-
  CREATE TABLE Singers (
    SingerId INT64 NOT NULL,
    FirstName STRING(1024),
    LastName STRING(1024),
    SingerInfo BYTES(MAX),
  ) PRIMARY KEY(SingerId)
---
  CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName)
```

## Query with a specific index

The following sections explain how to specify an index in a SQL statement and with the read interface for Spanner. The examples in these sections assume that you added a `MarketingBudget` column to the `Albums` table and created an index called `AlbumsByAlbumTitle` :

### GoogleSQL

```
CREATE TABLE Albums (
  SingerId         INT64 NOT NULL,
  AlbumId          INT64 NOT NULL,
  AlbumTitle       STRING(MAX),
  MarketingBudget  INT64,
) PRIMARY KEY (SingerId, AlbumId),
  INTERLEAVE IN PARENT Singers ON DELETE CASCADE;

CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle);
```

### PostgreSQL

```
CREATE TABLE Albums (
  SingerId         BIGINT NOT NULL,
  AlbumId          BIGINT NOT NULL,
  AlbumTitle       VARCHAR,
  MarketingBudget  BIGINT,
  PRIMARY KEY (SingerId, AlbumId)
) INTERLEAVE IN PARENT Singers ON DELETE CASCADE;

CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle);
```

### Specify an index in a SQL statement

When you use SQL to query a Spanner table, Spanner automatically uses any indexes that are likely to make the query more efficient. As a result, you don't need to specify an index for SQL queries. However, for queries that are critical for your workload, Google advises you to use `FORCE_INDEX` directives in your SQL statements for more consistent performance.

> **Note:** After you make significant changes to the data or schema of your database, [constructing a new statistics package](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview#construct-statistics-package) can improve the query optimizer's automatic index selection.

In a few cases, Spanner might choose an index that causes query latency to increase. If you've followed the [troubleshooting steps for performance regressions](https://docs.cloud.google.com/spanner/docs/troubleshooting-performance-regressions) and confirmed that it makes sense to try a different index for the query, you can specify the index as part of your query.

To specify an index in a SQL statement, use the [`FORCE_INDEX`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#table-hints) hint to provide an *index directive* . Index directives use the following syntax:

### GoogleSQL

```
FROM MyTable@{FORCE_INDEX=MyTableIndex}
```

### PostgreSQL

```
FROM MyTable /*@ FORCE_INDEX = MyTableIndex */
```

You can also use an index directive to tell Spanner to scan the base table instead of using an index:

### GoogleSQL

```
FROM MyTable@{FORCE_INDEX=_BASE_TABLE}
```

### PostgreSQL

```
FROM MyTable /*@ FORCE_INDEX = _BASE_TABLE */
```

You can use an index directive to tell Spanner to scan an index in a table with named schemas:

### GoogleSQL

```
FROM NAMED_SCHEMA_NAME.TABLE_NAME@{FORCE_INDEX="NAMED_SCHEMA_NAME.TABLE_INDEX_NAME"}
```

### PostgreSQL

```
FROM NAMED_SCHEMA_NAME.TABLE_NAME /*@ FORCE_INDEX = TABLE_INDEX_NAME */
```

The following example shows a SQL query that specifies an index:

### GoogleSQL

```
SELECT AlbumId, AlbumTitle, MarketingBudget
    FROM Albums@{FORCE_INDEX=AlbumsByAlbumTitle}
    WHERE AlbumTitle >= "Aardvark" AND AlbumTitle < "Goo";
```

### PostgreSQL

```
SELECT AlbumId, AlbumTitle, MarketingBudget
    FROM Albums /*@ FORCE_INDEX = AlbumsByAlbumTitle */
    WHERE AlbumTitle >= 'Aardvark' AND AlbumTitle < 'Goo';
```

An index directive might force Spanner's query processor to read additional columns that are required by the query but not stored in the index. The query processor retrieves these columns by joining the index and the base table. To avoid this extra join, use a [`STORING` clause (GoogleSQL-dialect databases) or `INCLUDE` clause (PostgreSQL-dialect databases)](https://docs.cloud.google.com/spanner/docs/secondary-indexes#storing-clause) to store the additional columns in the index.

In the previous example, the `MarketingBudget` column is not stored in the index, but the SQL query selects this column. As a result, Spanner must look up the `MarketingBudget` column in the base table, then join it with data from the index, to return the query results.

Spanner raises an error if the index directive has any of the following issues:

- The index does not exist.
- The index is on a different base table.
- The query is missing a [required `NULL` filtering expression](https://docs.cloud.google.com/spanner/docs/secondary-indexes#null-indexing-disable) for a [`NULL_FILTERED`](https://docs.cloud.google.com/spanner/docs/secondary-indexes#null-indexing) index.

The following examples show how to write and execute queries that fetch the values of `AlbumId` , `AlbumTitle` , and `MarketingBudget` using the index `AlbumsByAlbumTitle` :

### C++

```
void QueryUsingIndex(google::cloud::spanner::Client client) {
  namespace spanner = ::google::cloud::spanner;

  spanner::SqlStatement select(
      "SELECT AlbumId, AlbumTitle, MarketingBudget"
      " FROM Albums@{FORCE_INDEX=AlbumsByAlbumTitle}"
      " WHERE AlbumTitle >= @start_title AND AlbumTitle < @end_title",
      {{"start_title", spanner::Value("Aardvark")},
       {"end_title", spanner::Value("Goo")}});
  using RowType =
      std::tuple<std::int64_t, std::string, std::optional<std::int64_t>>;
  auto rows = client.ExecuteQuery(std::move(select));
  for (auto& row : spanner::StreamOf<RowType>(rows)) {
    if (!row) throw std::move(row).status();
    std::cout << "AlbumId: " << std::get<0>(*row) << "\t";
    std::cout << "AlbumTitle: " << std::get<1>(*row) << "\t";
    auto marketing_budget = std::get<2>(*row);
    if (marketing_budget) {
      std::cout << "MarketingBudget: " << *marketing_budget << "\n";
    } else {
      std::cout << "MarketingBudget: NULL\n";
    }
  }
  std::cout << "Read completed for [spanner_query_data_with_index]\n";
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System.Collections.Generic;
using System.Threading.Tasks;

public class QueryDataWithIndexAsyncSample
{
    public class Album
    {
        public int AlbumId { get; set; }
        public string AlbumTitle { get; set; }
        public long MarketingBudget { get; set; }
    }

    public async Task<List<Album>> QueryDataWithIndexAsync(string projectId, string instanceId, string databaseId,
        string startTitle, string endTitle)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";
        using var connection = new SpannerConnection(connectionString);
        using var cmd = connection.CreateSelectCommand(
            "SELECT AlbumId, AlbumTitle, MarketingBudget FROM Albums@ "
            + "{FORCE_INDEX=AlbumsByAlbumTitle} "
            + $"WHERE AlbumTitle >= @startTitle "
            + $"AND AlbumTitle < @endTitle",
            new SpannerParameterCollection
            {
                { "startTitle", SpannerDbType.String, startTitle },
                { "endTitle", SpannerDbType.String, endTitle }
            });

        var albums = new List<Album>();
        using var reader = await cmd.ExecuteReaderAsync();
        while (await reader.ReadAsync())
        {
            albums.Add(new Album
            {
                AlbumId = reader.GetFieldValue<int>("AlbumId"),
                AlbumTitle = reader.GetFieldValue<string>("AlbumTitle"),
                MarketingBudget = reader.IsDBNull(reader.GetOrdinal("MarketingBudget")) ? 0 : reader.GetFieldValue<long>("MarketingBudget")
            });
        }
        return albums;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"
    "strconv"

    "cloud.google.com/go/spanner"
    "google.golang.org/api/iterator"
)

func queryUsingIndex(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    stmt := spanner.Statement{
        SQL: `SELECT AlbumId, AlbumTitle, MarketingBudget
            FROM Albums@{FORCE_INDEX=AlbumsByAlbumTitle}
            WHERE AlbumTitle >= @start_title AND AlbumTitle < @end_title`,
        Params: map[string]interface{}{
            "start_title": "Aardvark",
            "end_title":   "Goo",
        },
    }
    iter := client.Single().Query(ctx, stmt)
    defer iter.Stop()
    for {
        row, err := iter.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return err
        }
        var albumID int64
        var marketingBudget spanner.NullInt64
        var albumTitle string
        if err := row.ColumnByName("AlbumId", &albumID); err != nil {
            return err
        }
        if err := row.ColumnByName("AlbumTitle", &albumTitle); err != nil {
            return err
        }
        if err := row.ColumnByName("MarketingBudget", &marketingBudget); err != nil {
            return err
        }
        budget := "NULL"
        if marketingBudget.Valid {
            budget = strconv.FormatInt(marketingBudget.Int64, 10)
        }
        fmt.Fprintf(w, "%d %s %s\n", albumID, albumTitle, budget)
    }
    return nil
}
```

### Java

```
static void queryUsingIndex(DatabaseClient dbClient) {
  Statement statement =
      Statement
          // We use FORCE_INDEX hint to specify which index to use. For more details see
          // https://cloud.google.com/spanner/docs/query-syntax#from-clause
          .newBuilder(
              "SELECT AlbumId, AlbumTitle, MarketingBudget "
                  + "FROM Albums@{FORCE_INDEX=AlbumsByAlbumTitle} "
                  + "WHERE AlbumTitle >= @StartTitle AND AlbumTitle < @EndTitle")
          // We use @BoundParameters to help speed up frequently executed queries.
          //  For more details see https://cloud.google.com/spanner/docs/sql-best-practices
          .bind("StartTitle")
          .to("Aardvark")
          .bind("EndTitle")
          .to("Goo")
          .build();
  try (ResultSet resultSet = dbClient.singleUse().executeQuery(statement)) {
    while (resultSet.next()) {
      System.out.printf(
          "%d %s %s\n",
          resultSet.getLong("AlbumId"),
          resultSet.getString("AlbumTitle"),
          resultSet.isNull("MarketingBudget") ? "NULL" : resultSet.getLong("MarketingBudget"));
    }
  }
}
```

### Node.js

```
/**
 * TODO(developer): Uncomment these variables before running the sample.
 */
// const instanceId = 'my-instance';
// const databaseId = 'my-database';
// const projectId = 'my-project-id';
// const startTitle = 'Ardvark';
// const endTitle = 'Goo';

// Imports the Google Cloud Spanner client library
const {Spanner} = require('@google-cloud/spanner');

// Instantiates a client
const spanner = new Spanner({
  projectId: projectId,
});

async function queryDataWithIndex() {
  // Gets a reference to a Cloud Spanner instance and database
  const instance = spanner.instance(instanceId);
  const database = instance.database(databaseId);

  const query = {
    sql: `SELECT AlbumId, AlbumTitle, MarketingBudget
                FROM Albums@{FORCE_INDEX=AlbumsByAlbumTitle}
                WHERE AlbumTitle >= @startTitle AND AlbumTitle <= @endTitle`,
    params: {
      startTitle: startTitle,
      endTitle: endTitle,
    },
  };

  // Queries rows from the Albums table
  try {
    const [rows] = await database.run(query);

    rows.forEach(row => {
      const json = row.toJSON();
      const marketingBudget = json.MarketingBudget
        ? json.MarketingBudget
        : null; // This value is nullable
      console.log(
        `AlbumId: ${json.AlbumId}, AlbumTitle: ${json.AlbumTitle}, MarketingBudget: ${marketingBudget}`,
      );
    });
  } catch (err) {
    console.error('ERROR:', err);
  } finally {
    // Close the database when finished.
    database.close();
  }
}
queryDataWithIndex();
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;

/**
 * Queries sample data from the database using SQL and an index.
 *
 * The index must exist before running this sample. You can add the index
 * by running the `add_index` sample or by running this DDL statement against
 * your database:
 *
 *     CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle)
 *
 * Example:
 * ```
 * query_data_with_index($instanceId, $databaseId);
 * ```
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 * @param string $startTitle The start of the title index.
 * @param string $endTitle   The end of the title index.
 */
function query_data_with_index(
    string $instanceId,
    string $databaseId,
    string $startTitle = 'Aardvark',
    string $endTitle = 'Goo'
): void {
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $parameters = [
        'startTitle' => $startTitle,
        'endTitle' => $endTitle
    ];

    $results = $database->execute(
        'SELECT AlbumId, AlbumTitle, MarketingBudget ' .
        'FROM Albums@{FORCE_INDEX=AlbumsByAlbumTitle} ' .
        'WHERE AlbumTitle >= @startTitle AND AlbumTitle < @endTitle',
        ['parameters' => $parameters]
    );

    foreach ($results as $row) {
        printf('AlbumId: %s, AlbumTitle: %s, MarketingBudget: %d' . PHP_EOL,
            $row['AlbumId'], $row['AlbumTitle'], $row['MarketingBudget']);
    }
}
```

### Python

```
def query_data_with_index(
    instance_id, database_id, start_title="Aardvark", end_title="Goo"
):
    """Queries sample data from the database using SQL and an index.

    The index must exist before running this sample. You can add the index
    by running the `add_index` sample or by running this DDL statement against
    your database:

        CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle)

    This sample also uses the `MarketingBudget` column. You can add the column
    by running the `add_column` sample or by running this DDL statement against
    your database:

        ALTER TABLE Albums ADD COLUMN MarketingBudget INT64

    """
    spanner_client = spanner.Client()
    instance = spanner_client.instance(instance_id)
    database = instance.database(database_id)

    params = {"start_title": start_title, "end_title": end_title}
    param_types = {
        "start_title": spanner.param_types.STRING,
        "end_title": spanner.param_types.STRING,
    }

    with database.snapshot() as snapshot:
        results = snapshot.execute_sql(
            "SELECT AlbumId, AlbumTitle, MarketingBudget "
            "FROM Albums@{FORCE_INDEX=AlbumsByAlbumTitle} "
            "WHERE AlbumTitle >= @start_title AND AlbumTitle < @end_title",
            params=params,
            param_types=param_types,
        )

        for row in results:
            print("AlbumId: {}, AlbumTitle: {}, " "MarketingBudget: {}".format(*row))
```

### Ruby

```
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"
# start_title = "An album title to start with such as 'Ardvark'"
# end_title   = "An album title to end with such as 'Goo'"

require "google/cloud/spanner"

spanner = Google::Cloud::Spanner.new project: project_id
client  = spanner.client instance_id, database_id

sql_query = "SELECT AlbumId, AlbumTitle, MarketingBudget
             FROM Albums@{FORCE_INDEX=AlbumsByAlbumTitle}
             WHERE AlbumTitle >= @start_title AND AlbumTitle < @end_title"

params      = { start_title: start_title, end_title: end_title }
param_types = { start_title: :STRING,     end_title: :STRING }

client.execute(sql_query, params: params, types: param_types).rows.each do |row|
  puts "#{row[:AlbumId]} #{row[:AlbumTitle]} #{row[:MarketingBudget]}"
end
```

### Specify an index in the read interface

When you use the read interface to Spanner, and you want Spanner to use an index, you must specify the index. The read interface does not select the index automatically.

In addition, your index must contain all of the data that appears in the query results, excluding columns that are part of the primary key. This restriction exists because the read interface does not support joins between the index and the base table. If you need to include other columns in the query results, you have a few options:

- Use a [`STORING` or `INCLUDE` clause](https://docs.cloud.google.com/spanner/docs/secondary-indexes#storing-clause) to store the additional columns in the index.
- Query without including the additional columns, then use the primary keys to send another query that reads the additional columns.

Spanner returns values from the index in ascending sort order by index key. To retrieve values in descending order, complete these steps:

- Annotate the index key with `DESC` . For example:

  ```
  CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle DESC);
  ```

  The `DESC` annotation applies to a single index key. If the index includes more than one key, and you want the query results to appear in descending order based on all keys, include a `DESC` annotation for each key.

- If the read specifies a key range, ensure that the key range is also in descending order. In other words, the value of the start key must be greater than the value of the end key.

The following example shows how to retrieve the values of `AlbumId` and `AlbumTitle` using the index `AlbumsByAlbumTitle` :

### C++

```
void ReadDataWithIndex(google::cloud::spanner::Client client) {
  namespace spanner = ::google::cloud::spanner;

  auto rows =
      client.Read("Albums", google::cloud::spanner::KeySet::All(),
                  {"AlbumId", "AlbumTitle"},
                  google::cloud::Options{}.set<spanner::ReadIndexNameOption>(
                      "AlbumsByAlbumTitle"));
  using RowType = std::tuple<std::int64_t, std::string>;
  for (auto& row : spanner::StreamOf<RowType>(rows)) {
    if (!row) throw std::move(row).status();
    std::cout << "AlbumId: " << std::get<0>(*row) << "\t";
    std::cout << "AlbumTitle: " << std::get<1>(*row) << "\n";
  }
  std::cout << "Read completed for [spanner_read_data_with_index]\n";
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System.Collections.Generic;
using System.Threading.Tasks;

public class QueryDataWithIndexAsyncSample
{
    public class Album
    {
        public int AlbumId { get; set; }
        public string AlbumTitle { get; set; }
        public long MarketingBudget { get; set; }
    }

    public async Task<List<Album>> QueryDataWithIndexAsync(string projectId, string instanceId, string databaseId,
        string startTitle, string endTitle)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";
        using var connection = new SpannerConnection(connectionString);
        using var cmd = connection.CreateSelectCommand(
            "SELECT AlbumId, AlbumTitle, MarketingBudget FROM Albums@ "
            + "{FORCE_INDEX=AlbumsByAlbumTitle} "
            + $"WHERE AlbumTitle >= @startTitle "
            + $"AND AlbumTitle < @endTitle",
            new SpannerParameterCollection
            {
                { "startTitle", SpannerDbType.String, startTitle },
                { "endTitle", SpannerDbType.String, endTitle }
            });

        var albums = new List<Album>();
        using var reader = await cmd.ExecuteReaderAsync();
        while (await reader.ReadAsync())
        {
            albums.Add(new Album
            {
                AlbumId = reader.GetFieldValue<int>("AlbumId"),
                AlbumTitle = reader.GetFieldValue<string>("AlbumTitle"),
                MarketingBudget = reader.IsDBNull(reader.GetOrdinal("MarketingBudget")) ? 0 : reader.GetFieldValue<long>("MarketingBudget")
            });
        }
        return albums;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
    "google.golang.org/api/iterator"
)

func readUsingIndex(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    iter := client.Single().ReadUsingIndex(ctx, "Albums", "AlbumsByAlbumTitle", spanner.AllKeys(),
        []string{"AlbumId", "AlbumTitle"})
    defer iter.Stop()
    for {
        row, err := iter.Next()
        if err == iterator.Done {
            return nil
        }
        if err != nil {
            return err
        }
        var albumID int64
        var albumTitle string
        if err := row.Columns(&albumID, &albumTitle); err != nil {
            return err
        }
        fmt.Fprintf(w, "%d %s\n", albumID, albumTitle)
    }
}
```

### Java

```
static void readUsingIndex(DatabaseClient dbClient) {
  try (ResultSet resultSet =
      dbClient
          .singleUse()
          .readUsingIndex(
              "Albums",
              "AlbumsByAlbumTitle",
              KeySet.all(),
              Arrays.asList("AlbumId", "AlbumTitle"))) {
    while (resultSet.next()) {
      System.out.printf("%d %s\n", resultSet.getLong(0), resultSet.getString(1));
    }
  }
}
```

### Node.js

```
/**
 * TODO(developer): Uncomment these variables before running the sample.
 */
// const instanceId = 'my-instance';
// const databaseId = 'my-database';
// const projectId = 'my-project-id';

// Imports the Google Cloud Spanner client library
const {Spanner} = require('@google-cloud/spanner');

// Instantiates a client
const spanner = new Spanner({
  projectId: projectId,
});

async function readDataWithIndex() {
  // Gets a reference to a Cloud Spanner instance and database
  const instance = spanner.instance(instanceId);
  const database = instance.database(databaseId);

  const albumsTable = database.table('Albums');

  const query = {
    columns: ['AlbumId', 'AlbumTitle'],
    keySet: {
      all: true,
    },
    index: 'AlbumsByAlbumTitle',
  };

  // Reads the Albums table using an index
  try {
    const [rows] = await albumsTable.read(query);

    rows.forEach(row => {
      const json = row.toJSON();
      console.log(`AlbumId: ${json.AlbumId}, AlbumTitle: ${json.AlbumTitle}`);
    });
  } catch (err) {
    console.error('ERROR:', err);
  } finally {
    // Close the database when finished.
    database.close();
  }
}
readDataWithIndex();
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;

/**
 * Reads sample data from the database using an index.
 *
 * The index must exist before running this sample. You can add the index
 * by running the `add_index` sample or by running this DDL statement against
 * your database:
 *
 *     CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle)
 *
 * Example:
 * ```
 * read_data_with_index($instanceId, $databaseId);
 * ```
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function read_data_with_index(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $keySet = $spanner->keySet(['all' => true]);
    $results = $database->read(
        'Albums',
        $keySet,
        ['AlbumId', 'AlbumTitle'],
        ['index' => 'AlbumsByAlbumTitle']
    );

    foreach ($results->rows() as $row) {
        printf('AlbumId: %s, AlbumTitle: %s' . PHP_EOL,
            $row['AlbumId'], $row['AlbumTitle']);
    }
}
```

### Python

```
def read_data_with_index(instance_id, database_id):
    """Reads sample data from the database using an index.

    The index must exist before running this sample. You can add the index
    by running the `add_index` sample or by running this DDL statement against
    your database:

        CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle)

    """
    spanner_client = spanner.Client()
    instance = spanner_client.instance(instance_id)
    database = instance.database(database_id)

    with database.snapshot() as snapshot:
        keyset = spanner.KeySet(all_=True)
        results = snapshot.read(
            table="Albums",
            columns=("AlbumId", "AlbumTitle"),
            keyset=keyset,
            index="AlbumsByAlbumTitle",
        )

        for row in results:
            print("AlbumId: {}, AlbumTitle: {}".format(*row))
```

### Ruby

```
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"

require "google/cloud/spanner"

spanner = Google::Cloud::Spanner.new project: project_id
client  = spanner.client instance_id, database_id

result = client.read "Albums", [:AlbumId, :AlbumTitle],
                     index: "AlbumsByAlbumTitle"

result.rows.each do |row|
  puts "#{row[:AlbumId]} #{row[:AlbumTitle]}"
end
```

## Create an index for index-only scans

Optionally, you can use the `STORING` clause (for GoogleSQL-dialect databases) or `INCLUDE` clause (for PostgreSQL-dialect databases) to store a copy of a column in the index. This type of index provides advantages for queries and read calls using the index, at the cost of using extra storage:

- SQL queries that use the index and select columns stored in the `STORING` or `INCLUDE` clause don't require an extra join to the base table.
- `read()` calls that use the index can read columns stored by the `STORING` / `INCLUDE` clause.

For example, suppose you created an alternate version of `AlbumsByAlbumTitle` that stores a copy of the `MarketingBudget` column in the index (note the `STORING` or `INCLUDE` clause in bold):

### GoogleSQL

```
CREATE INDEX AlbumsByAlbumTitle2 ON Albums(AlbumTitle) STORING (MarketingBudget);
```

### PostgreSQL

```
CREATE INDEX AlbumsByAlbumTitle2 ON Albums(AlbumTitle) INCLUDE (MarketingBudget);
```

With the old `AlbumsByAlbumTitle` index, Spanner must join the index with the base table, then retrieve the column from the base table. With the new `AlbumsByAlbumTitle2` index, Spanner reads the column directly from the index, which is more efficient.

If you use the read interface instead of SQL, the new `AlbumsByAlbumTitle2` index also lets you read the `MarketingBudget` column directly:

### C++

```
void ReadDataWithStoringIndex(google::cloud::spanner::Client client) {
  namespace spanner = ::google::cloud::spanner;

  auto rows =
      client.Read("Albums", google::cloud::spanner::KeySet::All(),
                  {"AlbumId", "AlbumTitle", "MarketingBudget"},
                  google::cloud::Options{}.set<spanner::ReadIndexNameOption>(
                      "AlbumsByAlbumTitle2"));
  using RowType =
      std::tuple<std::int64_t, std::string, std::optional<std::int64_t>>;
  for (auto& row : spanner::StreamOf<RowType>(rows)) {
    if (!row) throw std::move(row).status();
    std::cout << "AlbumId: " << std::get<0>(*row) << "\t";
    std::cout << "AlbumTitle: " << std::get<1>(*row) << "\t";
    auto marketing_budget = std::get<2>(*row);
    if (marketing_budget) {
      std::cout << "MarketingBudget: " << *marketing_budget << "\n";
    } else {
      std::cout << "MarketingBudget: NULL\n";
    }
  }
  std::cout << "Read completed for [spanner_read_data_with_storing_index]\n";
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System.Collections.Generic;
using System.Threading.Tasks;

public class QueryDataWithStoringIndexAsyncSample
{
    public class Album
    {
        public int AlbumId { get; set; }
        public string AlbumTitle { get; set; }
        public long? MarketingBudget { get; set; }
    }

    public async Task<List<Album>> QueryDataWithStoringIndexAsync(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        var cmd = connection.CreateSelectCommand(
            "SELECT AlbumId, AlbumTitle, MarketingBudget FROM Albums@ "
            + "{FORCE_INDEX=AlbumsByAlbumTitle2}");

        var albums = new List<Album>();
        using var reader = await cmd.ExecuteReaderAsync();
        while (await reader.ReadAsync())
        {
            albums.Add(new Album
            {
                AlbumId = reader.GetFieldValue<int>("AlbumId"),
                AlbumTitle = reader.GetFieldValue<string>("AlbumTitle"),
                MarketingBudget = reader.IsDBNull(reader.GetOrdinal("MarketingBudget")) ? 0 : reader.GetFieldValue<long>("MarketingBudget")
            });
        }
        return albums;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"
    "strconv"

    "cloud.google.com/go/spanner"
    "google.golang.org/api/iterator"
)

func readStoringIndex(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    iter := client.Single().ReadUsingIndex(ctx, "Albums", "AlbumsByAlbumTitle2", spanner.AllKeys(),
        []string{"AlbumId", "AlbumTitle", "MarketingBudget"})
    defer iter.Stop()
    for {
        row, err := iter.Next()
        if err == iterator.Done {
            return nil
        }
        if err != nil {
            return err
        }
        var albumID int64
        var marketingBudget spanner.NullInt64
        var albumTitle string
        if err := row.Columns(&albumID, &albumTitle, &marketingBudget); err != nil {
            return err
        }
        budget := "NULL"
        if marketingBudget.Valid {
            budget = strconv.FormatInt(marketingBudget.Int64, 10)
        }
        fmt.Fprintf(w, "%d %s %s\n", albumID, albumTitle, budget)
    }
}
```

### Java

```
static void readStoringIndex(DatabaseClient dbClient) {
  // We can read MarketingBudget also from the index since it stores a copy of MarketingBudget.
  try (ResultSet resultSet =
      dbClient
          .singleUse()
          .readUsingIndex(
              "Albums",
              "AlbumsByAlbumTitle2",
              KeySet.all(),
              Arrays.asList("AlbumId", "AlbumTitle", "MarketingBudget"))) {
    while (resultSet.next()) {
      System.out.printf(
          "%d %s %s\n",
          resultSet.getLong(0),
          resultSet.getString(1),
          resultSet.isNull("MarketingBudget") ? "NULL" : resultSet.getLong("MarketingBudget"));
    }
  }
}
```

### Node.js

```
/**
 * TODO(developer): Uncomment these variables before running the sample.
 */
// const instanceId = 'my-instance';
// const databaseId = 'my-database';
// const projectId = 'my-project-id';

// Imports the Google Cloud Spanner client library
const {Spanner} = require('@google-cloud/spanner');

// Instantiates a client
const spanner = new Spanner({
  projectId: projectId,
});

// "Storing" indexes store copies of the columns they index
// This speeds up queries, but takes more space compared to normal indexes
// See the link below for more information:
// https://cloud.google.com/spanner/docs/secondary-indexes#storing_clause
async function readDataWithStoringIndex() {
  // Gets a reference to a Cloud Spanner instance and database
  const instance = spanner.instance(instanceId);
  const database = instance.database(databaseId);

  const albumsTable = database.table('Albums');

  const query = {
    columns: ['AlbumId', 'AlbumTitle', 'MarketingBudget'],
    keySet: {
      all: true,
    },
    index: 'AlbumsByAlbumTitle2',
  };

  // Reads the Albums table using a storing index
  try {
    const [rows] = await albumsTable.read(query);

    rows.forEach(row => {
      const json = row.toJSON();
      let rowString = `AlbumId: ${json.AlbumId}`;
      rowString += `, AlbumTitle: ${json.AlbumTitle}`;
      if (json.MarketingBudget) {
        rowString += `, MarketingBudget: ${json.MarketingBudget}`;
      }
      console.log(rowString);
    });
  } catch (err) {
    console.error('ERROR:', err);
  } finally {
    // Close the database when finished.
    database.close();
  }
}
readDataWithStoringIndex();
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;

/**
 * Reads sample data from the database using an index with a storing
 * clause.
 *
 * The index must exist before running this sample. You can add the index
 * by running the `add_storing_index` sample or by running this DDL statement
 * against your database:
 *
 *     CREATE INDEX AlbumsByAlbumTitle2 ON Albums(AlbumTitle)
 *     STORING (MarketingBudget)
 *
 * Example:
 * ```
 * read_data_with_storing_index($instanceId, $databaseId);
 * ```
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function read_data_with_storing_index(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $keySet = $spanner->keySet(['all' => true]);
    $results = $database->read(
        'Albums',
        $keySet,
        ['AlbumId', 'AlbumTitle', 'MarketingBudget'],
        ['index' => 'AlbumsByAlbumTitle2']
    );

    foreach ($results->rows() as $row) {
        printf('AlbumId: %s, AlbumTitle: %s, MarketingBudget: %d' . PHP_EOL,
            $row['AlbumId'], $row['AlbumTitle'], $row['MarketingBudget']);
    }
}
```

### Python

```
def read_data_with_storing_index(instance_id, database_id):
    """Reads sample data from the database using an index with a storing
    clause.

    The index must exist before running this sample. You can add the index
    by running the `add_scoring_index` sample or by running this DDL statement
    against your database:

        CREATE INDEX AlbumsByAlbumTitle2 ON Albums(AlbumTitle)
        STORING (MarketingBudget)

    """
    spanner_client = spanner.Client()
    instance = spanner_client.instance(instance_id)
    database = instance.database(database_id)

    with database.snapshot() as snapshot:
        keyset = spanner.KeySet(all_=True)
        results = snapshot.read(
            table="Albums",
            columns=("AlbumId", "AlbumTitle", "MarketingBudget"),
            keyset=keyset,
            index="AlbumsByAlbumTitle2",
        )

        for row in results:
            print("AlbumId: {}, AlbumTitle: {}, " "MarketingBudget: {}".format(*row))
```

### Ruby

```
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"

require "google/cloud/spanner"

spanner = Google::Cloud::Spanner.new project: project_id
client  = spanner.client instance_id, database_id

result = client.read "Albums", [:AlbumId, :AlbumTitle, :MarketingBudget],
                     index: "AlbumsByAlbumTitle2"

result.rows.each do |row|
  puts "#{row[:AlbumId]} #{row[:AlbumTitle]} #{row[:MarketingBudget]}"
end
```

## Alter an index

You can use the `ALTER INDEX` statement to add additional columns into an existing index or drop columns. This can update the columns list defined by the `STORING` clause (GoogleSQL-dialect databases) or `INCLUDE` clause (PostgreSQL-dialect databases) when you create the index. You can't use this statement to add columns to or drop columns from the index key. For example, instead of creating a new index `AlbumsByAlbumTitle2` , you can use `ALTER INDEX` to add a column into `AlbumsByAlbumTitle` , as shown in the following example:

### GoogleSQL

```
ALTER INDEX AlbumsByAlbumTitle ADD STORED COLUMN MarketingBudget
```

### PostgreSQL

```
ALTER INDEX AlbumsByAlbumTitle ADD INCLUDE COLUMN MarketingBudget
```

When you add a new column into an existing index, Spanner uses a background backfilling process. While the backfill is ongoing, the column in the index is not readable, so you might not get the expected performance boost. You can use the `gcloud spanner operations` command to list the long-running operation and view its status. For more information, see [describe operation](https://docs.cloud.google.com/spanner/docs/secondary-indexes#index-progress) .

You can also use [cancel operation](https://docs.cloud.google.com/spanner/docs/secondary-indexes#cancel-create) to cancel a running operation.

After the backfill is done, Spanner adds the column into the index. As the Index grows bigger, this might slow down the queries that use the index.

The following example shows how to drop a column from an index:

### GoogleSQL

```
ALTER INDEX AlbumsByAlbumTitle DROP STORED COLUMN MarketingBudget
```

### PostgreSQL

```
ALTER INDEX AlbumsByAlbumTitle DROP INCLUDE COLUMN MarketingBudget
```

## Index of NULL values

By default, Spanner indexes `NULL` values. For example, recall the definition of the index `SingersByFirstLastName` on the table `Singers` :

```
CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName);
```

All rows of `Singers` are indexed even if either `FirstName` or `LastName` , or both, are `NULL` .

[![Shows rows that are omitted from a NULL filtered index.](https://docs.cloud.google.com/static/spanner/docs/images/indexing_nulls.svg)](https://docs.cloud.google.com/static/spanner/docs/images/indexing_nulls.svg)

When `NULL` values are indexed, you can perform efficient SQL queries and reads over data that includes `NULL` values. For example, use this SQL query statement to find all `Singers` with a `NULL` `FirstName` :

### GoogleSQL

```
SELECT s.SingerId, s.FirstName, s.LastName
    FROM Singers@{FORCE_INDEX=SingersByFirstLastName} AS s
    WHERE s.FirstName IS NULL;
```

### PostgreSQL

```
SELECT s.SingerId, s.FirstName, s.LastName
    FROM Singers /* @ FORCE_INDEX = SingersByFirstLastName */ AS s
    WHERE s.FirstName IS NULL;
```

### Sort order for NULL values

Spanner sorts `NULL` as the smallest value for any given type. For a column in ascending ( `ASC` ) order, `NULL` values sort first. For a column in descending ( `DESC` ) order, `NULL` values sort last.

### Disable indexing of NULL values

### GoogleSQL

To disable the indexing of nulls, add the `NULL_FILTERED` keyword to the index definition. `NULL_FILTERED` indexes are particularly useful for indexing sparse columns, where most rows contain a `NULL` value. In these cases, the `NULL_FILTERED` index can be considerably smaller and more efficient to maintain than a normal index that includes `NULL` values.

Here's an alternate definition of `SingersByFirstLastName` that does not index `NULL` values:

```
CREATE NULL_FILTERED INDEX SingersByFirstLastNameNoNulls
    ON Singers(FirstName, LastName);
```

The `NULL_FILTERED` keyword applies to all index key columns. You cannot specify `NULL` filtering on a per-column basis.

### PostgreSQL

To filter out rows with null values in one or more indexed columns, use the `WHERE `` COLUMN `` IS NOT NULL` predicate. Null-filtered indexes are particularly useful for indexing sparse columns, where most rows contain a `NULL` value. In these cases, the null-filtered index can be considerably smaller and more efficient to maintain than a normal index that includes `NULL` values.

Here's an alternate definition of `SingersByFirstLastName` that does not index `NULL` values:

```
CREATE INDEX SingersByFirstLastNameNoNulls
    ON Singers(FirstName, LastName)
    WHERE FirstName IS NOT NULL
    AND LastName IS NOT NULL;
```

Filtering out `NULL` values prevents Spanner from using it for some queries. For example, Spanner does not use the index for this query, because the index omits any `Singers` rows for which `LastName` is `NULL` ; as a result, using the index would prevent the query from returning the correct rows:

### GoogleSQL

```
FROM Singers@{FORCE_INDEX=SingersByFirstLastNameNoNulls}
    WHERE FirstName = "John";
```

### PostgreSQL

```
FROM Singers /*@ FORCE_INDEX = SingersByFirstLastNameNoNulls */
    WHERE FirstName = 'John';
```

To enable Spanner to use the index, you must rewrite the query so it excludes the rows that are also excluded from the index:

### GoogleSQL

```
SELECT FirstName, LastName
    FROM Singers@{FORCE_INDEX=SingersByFirstLastNameNoNulls}
    WHERE FirstName = 'John' AND LastName IS NOT NULL;
```

### PostgreSQL

```
SELECT FirstName, LastName
    FROM Singers /*@ FORCE_INDEX = SingersByFirstLastNameNoNulls */
    WHERE FirstName = 'John' AND LastName IS NOT NULL;
```

## Index proto fields

Use [generated columns](https://docs.cloud.google.com/spanner/docs/generated-column/how-to) to index fields in protocol buffers stored in `PROTO` columns, as long as the fields being indexed use the primitive or `ENUM` data types.

If you define an index on a protocol message field, you can't modify or remove that field from the proto schema. For more information, see [Updates to schemas that contain an index on proto fields](https://docs.cloud.google.com/spanner/docs/secondary-indexes#updates-schemas-index-proto) .

The following is an example of the `Singers` table with a `SingerInfo` proto message column. To define an index on the `nationality` field of the `PROTO` , you need to create a stored generated column:

### GoogleSQL

```
CREATE PROTO BUNDLE (googlesql.example.SingerInfo, googlesql.example.SingerInfo.Residence);

CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  ...
  SingerInfo googlesql.example.SingerInfo,
  SingerNationality STRING(MAX) AS (SingerInfo.nationality) STORED
) PRIMARY KEY (SingerId);
```

It has the following definition of the `googlesql.example.SingerInfo` proto type:

### GoogleSQL

```
package googlesql.example;

message SingerInfo {
optional string    nationality = 1;
repeated Residence residence   = 2;

  message Residence {
    required int64  start_year   = 1;
    optional int64  end_year     = 2;
    optional string city         = 3;
    optional string country      = 4;
  }
}
```

Then define an index on the `nationality` field of the proto:

### GoogleSQL

```
CREATE INDEX SingersByNationality ON Singers(SingerNationality);
```

The following SQL query reads data using the previous index:

### GoogleSQL

```
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE s.SingerNationality = "English";
```

Notes:

- Use an [index directive](https://docs.cloud.google.com/spanner/docs/secondary-indexes#index-directive) to access indexes on the fields of protocol buffer columns.
- You can't create an index on repeated protocol buffer fields.

### Updates to schemas that contain an index on proto fields

If you define an index on a protocol message field, you can't modify or remove that field from the proto schema. This is because after you define the index, type checking is performed every time the schema is updated. Spanner captures the type information for all fields in the path that are used in the index definition.

## Unique indexes

Indexes can be declared `UNIQUE` . `UNIQUE` indexes add a constraint to the data being indexed that prohibits duplicate entries for a given index key. This constraint is enforced by Spanner at transaction commit time. Specifically, any transaction that would cause multiple index entries for the same key to exist will fail to commit.

If a table contains non- `UNIQUE` data in it to begin with, attempting to create a `UNIQUE` index on it will fail.

### A note about UNIQUE NULL_FILTERED indexes

A `UNIQUE NULL_FILTERED` index does not enforce index key uniqueness when at least one of the index's key parts is NULL.

For example, suppose that you created the following table and index:

### GoogleSQL

```
CREATE TABLE ExampleTable (
  Key1 INT64 NOT NULL,
  Key2 INT64,
  Key3 INT64,
  Col1 INT64,
) PRIMARY KEY (Key1, Key2, Key3);

CREATE UNIQUE NULL_FILTERED INDEX ExampleIndex ON ExampleTable (Key1, Key2, Col1);
```

### PostgreSQL

```
CREATE TABLE ExampleTable (
  Key1 BIGINT NOT NULL,
  Key2 BIGINT,
  Key3 BIGINT,
  Col1 BIGINT,
  PRIMARY KEY (Key1, Key2, Key3)
);

CREATE UNIQUE INDEX ExampleIndex ON ExampleTable (Key1, Key2, Col1)
    WHERE Key1 IS NOT NULL
    AND Key2 IS NOT NULL
    AND Col1 IS NOT NULL;
```

The following two rows in `ExampleTable` have the same values for the secondary index keys `Key1` , `Key2` and `Col1` :

```
1, NULL, 1, 1
1, NULL, 2, 1
```

Because `Key2` is `NULL` and the index is null-filtered, the rows won't be present in the index `ExampleIndex` . Because they are not inserted into the index, the index won't reject them for violating uniqueness on `(Key1, Key2, Col1)` .

If you want the index to enforce the uniqueness of values of the tuple ( `Key1` , `Key2` , `Col1` ), then you must annotate `Key2` with `NOT NULL` in the table definition or create the index without filtering nulls.

## Create an index based on a scalar expression

An expression index lets you create an index based on a scalar expression that applies to one or more table columns. This is beneficial because it eliminates the need to create and maintain a dedicated generated column in your table just for indexing purposes.

Indexes on generated columns and expression indexes can be useful for indexing specific fields within a JSON column. You can create an index on a value extracted from a JSON object, which lets the query optimizer efficiently perform lookups based on that value.

In this example, a generated column named `VenueCity` , which is based on the `city` field within the JSON, is created solely for the purpose of placing it in an index:

### GoogleSQL

```
CREATE TABLE Venues (
  Id INT64 NOT NULL,
  VenueData JSON,
  VenueCity STRING AS (JSON_VALUE(VenueData.address.city))
) PRIMARY KEY (Id);

CREATE INDEX VenuesByCity ON Venues(VenueCity);
```

### PostgreSQL

```
CREATE TABLE Venues (
  Id BIGINT NOT NULL PRIMARY KEY,
  VenueData JSONB,
  VenueCity VARCHAR GENERATED ALWAYS AS (VenueData -> 'address' ->> 'city')
);

CREATE INDEX VenuesByCity ON Venues(VenueCity);
```

To achieve this more efficiently, you can define the index directly, and Spanner handles the underlying mechanism implicitly:

### GoogleSQL

```
CREATE TABLE Venues (
  Id INT64 NOT NULL,
  VenueData JSON
) PRIMARY KEY (Id);

CREATE INDEX VenuesByCity ON Venues((JSON_VALUE(VenueData.address.city)));
```

### PostgreSQL

```
CREATE TABLE Venues (
  Id BIGINT NOT NULL PRIMARY KEY,
  VenueData JSONB
);

CREATE INDEX VenuesByCity ON Venues((VenueData -> 'address' ->> 'city'));
```

In both cases, your query can use the `VenuesByCity` index for faster execution:

### GoogleSQL

```
SELECT Id
FROM Venues
WHERE JSON_VALUE(VenueData.address.city) = 'Seattle';
```

### PostgreSQL

```
SELECT Id
FROM Venues
WHERE VenueData -> 'address' ->> 'city' = 'Seattle';
```

Expression Indexes can have columns that are a mix of expressions and direct column references. This allows more queries to use the index for faster execution.

### GoogleSQL

```
CREATE TABLE Venues (
  Id INT64 NOT NULL,
  VenueName STRING(MAX),
  VenueData JSON
) PRIMARY KEY (Id);

CREATE INDEX VenuesByCity ON Venues(
  VenueName,
  (JSON_VALUE(VenueData.address.city)),
  (JSON_VALUE(VenueData.address.state))
);
```

### PostgreSQL

```
CREATE TABLE Venues (
  Id BIGINT NOT NULL PRIMARY KEY,
  VenueName VARCHAR,
  VenueData JSONB
);

CREATE INDEX VenuesIdx ON Venues(
  VenueName,
  (VenueData -> 'address' ->> 'city'),
  (VenueData -> 'address' ->> 'state'),
);
```

## Drop an index

Use the `DROP INDEX` statement to drop a secondary index from your schema.

To drop the index named `SingersByFirstLastName` :

```
DROP INDEX SingersByFirstLastName;
```

## Index for faster scanning

When Spanner needs to perform a table scan (rather than an indexed lookup) to fetch values from one or more columns, you can receive faster results if an index exists for those columns, and in the order specified by the query. If you frequently perform queries that require scans, consider creating secondary indexes to help these scans happen more efficiently.

In particular, if you need Spanner to frequently scan a table's primary key or other index in reverse order, then you can increase its efficiency through a secondary index that makes the chosen order explicit.

For example, the following query always returns a fast result, even though Spanner needs to scan `Songs` to find the lowest value of `SongId` :

```
SELECT SongId FROM Songs LIMIT 1;
```

`SongId` is the table's primary key, stored (as with all primary keys) in ascending order. Spanner can scan that key's index and find the first result rapidly.

However, without the help of a secondary index, the following query wouldn't return as quickly, especially if `Songs` holds a lot of data:

```
SELECT SongId FROM Songs ORDER BY SongId DESC LIMIT 1;
```

Even though `SongId` is the table's primary key, Spanner has no way to fetch the column's highest value without resorting to a full table scan.

Adding the following index would allow this query to return more quickly:

```
CREATE INDEX SongIdDesc On Songs(SongId DESC);
```

With this index in place, Spanner would use it to return a result for the second query much more quickly.

## What's next

- Learn about [SQL best practices for Spanner](https://docs.cloud.google.com/spanner/docs/sql-best-practices) .
- Understand [query execution plans for Spanner](https://docs.cloud.google.com/spanner/docs/query-execution-plans) .
- Find out how to [troubleshoot performance regressions in SQL queries](https://docs.cloud.google.com/spanner/docs/troubleshooting-performance-regressions) .
