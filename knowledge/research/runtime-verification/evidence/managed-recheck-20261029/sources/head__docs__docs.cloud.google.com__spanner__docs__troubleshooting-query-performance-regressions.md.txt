---
name: documents/docs.cloud.google.com/spanner/docs/troubleshooting-query-performance-regressions
uri: https://docs.cloud.google.com/spanner/docs/troubleshooting-query-performance-regressions
title: Troubleshoot query performance regressions
description: Troubleshoot performance regressions in your Spanner database by analyzing metrics, query plans, and lock statistics.
data_source: docs.cloud.google.com
---

This page explains how to detect changes in query execution speed, inspect query execution plans, and specify a different index for future queries if necessary.

## Get the detailed profile of a query in JSON format using the REST API

By default, when you execute a query, the API returns only the statement results. This is because [`queryMode`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/QueryMode) is set to `NORMAL` . To include detailed execution statistics with the query results, set `queryMode` to `PROFILE` .

### Create a session

Before you update your query mode, create a [session](https://docs.cloud.google.com/spanner/docs/sessions) , which represents a communication channel with the Spanner database service.

1.  Go to the [`projects.instances.databases.sessions.create`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases.sessions/create) page.

2.  Provide the project ID, instance ID, and database ID in the following form:

    `projects/ `` PROJECT_ID `` /instances/ `` INSTANCE_ID `` /databases/ `` DATABASE_ID`

3.  Click **Execute** . The response shows the session that you created in this form:

    `projects/ `` PROJECT_ID `` /instances/ `` INSTANCE_ID `` /databases/ `` DATABASE_ID `` /sessions/ `` SESSION_ID`

    Use this session to profile the query in the next step. A session remains active for up to one hour between consecutive uses before Spanner deletes it.

For more information about how to use sessions, see [Sessions](https://docs.cloud.google.com/spanner/docs/sessions) .

### Profile the query

Enable `PROFILE` mode for the query.

1.  Go to the [`projects.instances.databases.sessions.executeSql`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases.sessions/executeSql) page.

2.  For **session** , enter the session ID that you created in [Create a session](https://docs.cloud.google.com/spanner/docs/troubleshooting-query-performance-regressions#create-session) :

    `projects/ `` PROJECT_ID `` /instances/ `` INSTANCE_ID `` /databases/ `` DATABASE_ID `` /sessions/ `` SESSION_ID`

3.  For **Request body** , use the following:

    `{ "sql": " `` SQL_QUERY `` ", "queryMode": "PROFILE" }`

4.  Click **Execute** . The response includes the query results, [query plan](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/ResultSetStats#QueryPlan) , and the execution statistics for the query.

## Troubleshoot index usage

When you use SQL queries to look up data, Spanner automatically uses any [secondary indexes](https://docs.cloud.google.com/spanner/docs/secondary-indexes) that are likely to retrieve the data more efficiently. However, in a few cases, though, Spanner might choose an index that causes queries to be slower. As a result, you might notice that some queries run slower than expected.

### Detect changes in query execution speed

You are most likely to see a change in query execution speed after you make one of these changes:

- Significantly changing a large amount of existing data that has a secondary index.
- Adding, changing, or dropping a secondary index.

You can use several different tools to identify a specific query that Spanner is executing more slowly than usual:

- [Query insights](https://docs.cloud.google.com/spanner/docs/using-query-insights#view-query-details) and [Query statistics](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics) .

- [Latency metrics](https://docs.cloud.google.com/spanner/docs/latency-metrics) .

- Application-specific metrics that you capture and analyze with [Cloud Monitoring](https://docs.cloud.google.com/monitoring/docs) . For example, you can monitor the [Count of queries](https://docs.cloud.google.com/spanner/docs/query-optimizer/manage-query-optimizer#visualize_query_optimizer_version_in_metrics_explorer) metric to determine the number of queries in an instance over time and to find out what query optimizer version was used to run a query.

- Client-side monitoring tools that measure your application's performance.

#### Optimizer statistics for newly created databases

When you query newly created databases with freshly inserted or imported data, Spanner might not select the most appropriate indexes because the query optimizer takes up to three days to collect statistics automatically. To optimize a new Spanner database's index usage sooner than that, you can [manually construct a new statistics package](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview#construct-statistics-package) .

### Review the schema

After you find the query that slowed down, look at the SQL statement for the query, and identify the tables that the statement uses and the columns that it retrieves from those tables.

Next, [find the secondary indexes](https://docs.cloud.google.com/spanner/docs/secondary-indexes#view-indexes) that exist for those tables. Determine whether any of the indexes include the columns you're querying, which means that Spanner might use one of the indexes to process the query.

- **If there are applicable indexes** , the next step is to [find the index that Spanner used for the query](https://docs.cloud.google.com/spanner/docs/troubleshooting-query-performance-regressions#find-index) .

- **If there are no applicable indexes** , use the [`gcloud spanner operations list`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/operations/list) command to check whether you recently dropped an applicable index:

  ```
  gcloud spanner operations list \
      --instance=INSTANCE \
      --database=DATABASE \
      --filter="@TYPE:UpdateDatabaseDdlMetadata"
  ```

  If you dropped an applicable index, that change might have affected query performance. [Add the secondary index back to the table](https://docs.cloud.google.com/spanner/docs/secondary-indexes#add-index) . After Spanner adds the index, run the query again and look at its performance. If performance does not improve, the next step is to [find the index that Spanner used for the query](https://docs.cloud.google.com/spanner/docs/troubleshooting-query-performance-regressions#find-index) .

  If you did not drop an applicable index, then index selection did not cause query performance to regress. Look for other changes to your data or usage patterns that might have affected performance.

### Find the index used for a query

> **Note:** Spanner Studio (formerly labeled **Query** in the Google Cloud console) supports SQL, DML, and DDL operations in a single editor. For more information, see [Manage your data using the Google Cloud console](https://docs.cloud.google.com/spanner/docs/manage-data-using-console) .

To find out what index Spanner is using to process a query, view the [query execution plan](https://docs.cloud.google.com/spanner/docs/query-execution-plans) in the Google Cloud console:

1.  Go to the Spanner **Instances** page in the Google Cloud console.

2.  Click the name of the instance you want to query.

3.  In the navigation menu, select the database you want to query, and then click search **Spanner Studio** .

4.  Enter the query to test.

5.  In the **Run query** menu, select **Explanation only** . Spanner displays the query plan.

Look for at least one of the following operators in the query plan:

- *Table scan*
- *Index scan*
- *Cross apply* or *distributed cross apply*

The following sections explain the meaning of each operator.

### Table scan operator

The [*table scan*](https://docs.cloud.google.com/spanner/docs/query-execution-operators#scan) operator indicates that Spanner did not use a secondary index:

![Table scan operator in a query plan.](https://docs.cloud.google.com/static/spanner/docs/images/query-plan-table-scan.png)

For example, suppose that the `Albums` table does not have any secondary indexes, and you run the following query:

```
SELECT AlbumTitle FROM Albums WHERE STARTS_WITH(AlbumTitle, "Now");
```

Because there are no indexes to use, the query plan includes a table scan operator.

#### Index scan operator

The [*index scan*](https://docs.cloud.google.com/spanner/docs/query-execution-operators#scan) operator indicates that Spanner used a secondary index when it processed the query:

![Index scan operator in a query plan.](https://docs.cloud.google.com/static/spanner/docs/images/query-plan-index-scan.png)

For example, suppose you add an index to the `Albums` table:

```
CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle);
```

Then you run the following query:

```
SELECT AlbumTitle FROM Albums WHERE STARTS_WITH(AlbumTitle, "Now");
```

The `AlbumsByAlbumTitle` index contains `AlbumTitle` , which is the only column that the query selects. As a result, the query plan includes an index scan operator.

#### Cross apply operator

In some cases, Spanner uses an index that contains only some of the columns that the query selects. As a result, Spanner must join the index with the base table.

When this type of join occurs, the query plan includes a [*cross apply*](https://docs.cloud.google.com/spanner/docs/query-execution-operators#cross-apply) or [*distributed cross apply*](https://docs.cloud.google.com/spanner/docs/query-execution-operators#distributed-cross-apply) operator that has the following inputs:

- An index scan operator for a table's index
- A table scan operator for the table that owns the index

![Distributed cross apply in a query plan with an index scan and a table scan as inputs.](https://docs.cloud.google.com/static/spanner/docs/images/query-plan-cross-apply.png)

For example, suppose you add an index to the `Albums` table:

```
CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle);
```

Then you run the following query:

```
SELECT * FROM Albums WHERE STARTS_WITH(AlbumTitle, "Now");
```

The `AlbumsByAlbumTitle` index contains `AlbumTitle` , but the query selects all of the columns in the table, not just `AlbumTitle` . As a result, the query plan includes a distributed cross apply operator, with an index scan of `AlbumsByAlbumTitle` and a table scan of `Albums` as its inputs.

### Choose a different index

After you find the index that Spanner used for your query, try running your query with a different index, or by scanning the base table instead of using an index. To specify the index, [add a `FORCE_INDEX` directive to the query](https://docs.cloud.google.com/spanner/docs/secondary-indexes#index-directive) .

If you find a faster version of the query, update your application to use the faster version.

#### Guidelines for choosing an index

Use these guidelines to decide what index to test for the query:

- If your query meets any of these criteria, try using the base table instead of a secondary index:

  - The query checks for equality with a prefix of the base table's [primary key](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#primary_keys) (for example, `SELECT * FROM Albums WHERE SingerId = 1` ).
  - A large number of rows satisfy the query predicates (for example, `SELECT * FROM Albums WHERE AlbumTitle != "There Is No Album With This Title"` ).
  - The query uses a base table that contains only a few hundred rows.

- If the query contains a very selective predicate (for example, `REGEXP_CONTAINS` , `STARTS_WITH` , `<` , `<=` , `>` , `>=` , or `!=` ), try using an index that includes the same columns that you use in the predicate.

> **Important:** If the database doesn't have a secondary index that's relevant to the query, then don't create an index immediately. Adding this index might affect the performance of other queries.

#### Test the updated query

Use the Google Cloud console to test the updated query and find out how long it takes to process the query.

If your query includes [query parameters](https://docs.cloud.google.com/spanner/docs/sql-best-practices#query-parameters) , and a query parameter is bound to some values much more often than others, then bind the query parameter with one of those values in your tests. For example, if the query includes a predicate such as `WHERE country = @countryId` , and almost all of your queries bind `@countryId` to the value `US` , then bind `@countryId` to `US` for your performance tests. This approach helps you optimize for the queries you run most frequently.

To test the updated query in the Google Cloud console, follow these steps:

1.  Go to the Spanner **Instances** page in the Google Cloud console.

2.  Click the name of the instance you want to query.

3.  In the navigation menu, click the database you want to query, and then click search **Spanner Studio** .

4.  Enter the query to test, including the `FORCE_INDEX` directive, and click **Run query** .

    The Google Cloud console opens the **Results table** tab, then shows the query results, including how long it took for the Spanner service to process the query.

    This metric does not include other sources of latency, such as the time it took for the Google Cloud console to interpret and display the query results.

## Query throttling

During query execution, Spanner can throttle queries in several places.

**Client wait time** and **Server queue delay** are the two primary throttling metrics.

### Client wait time

**Client wait time** represents the amount of time the Spanner client waits for the application to consume results. This indicates that the client can't process results as fast as Spanner produces them. To resolve this issue, either increase the performance of the consuming application or buffer all results in the client application and process them after the query completes.

### Server queue delay

**Server queue delay** occurs when the processing server delays scheduling query execution. In the **Query summary** pane, check the **Server queue delay** metric; if the metric isn't present, then no server delay occurred.

This delay can indicate that the servers executing the query are overloaded. To verify whether a hotspot exists, see [Find hotspots in your database](https://docs.cloud.google.com/spanner/docs/find-hotspots-in-database) . To troubleshoot hotspots, see [Troubleshoot hotspots using hot split statistics](https://docs.cloud.google.com/spanner/docs/introspection/hot-split-statistics#troubleshoot_hotspots_using_hot_split_statistics) .

## Recently changed data

Changes to table data can affect query performance.

### Outdated statistics package

If the optimizer uses a stale statistics package, then it might choose a suboptimal plan. In the **Query summary** section, you can check the statistics package. If the package is stale or missing, then run [`ANALYZE`](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview#construct-statistics-package) to make sure that optimizer statistics are up to date.

![Statistics package in query summary](https://docs.cloud.google.com/static/spanner/docs/images/statistics_package.png)

For more information, see [Statistics packages](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview#statistics-packages) .

### Changes in table contents

Query performance depends on the data that the query processes. The best way to address a query performance issue depends on how the data changed:

- **Skew in value distribution** : the statistics package captures this skew, but the query optimizer can only use this information when you use literal values in the query.
- **Large change in row count** : evaluate whether the predicates sufficiently filter the rows as the table grows in size.
- **Skew in value size** : skew in the size of values is problematic in conjunction with expensive scalar predicates (for example, performing a `LIKE` predicate on a `STRING` value that contains large strings).

To get more insights into query execution, use the [REST API](https://docs.cloud.google.com/spanner/docs/getting-started/rest) .
