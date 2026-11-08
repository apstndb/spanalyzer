---
name: documents/docs.cloud.google.com/spanner/docs/index-advisor
uri: https://docs.cloud.google.com/spanner/docs/index-advisor
title: Use the Spanner index advisor
description: Optimize query performance in Spanner by using the index advisor to create beneficial secondary indexes based on query patterns.
data_source: docs.cloud.google.com
---

This page describes the Spanner index advisor and how you can view and apply its index recommendations. The index advisor is available for GoogleSQL-dialect databases and PostgreSQL-dialect databases.

The Spanner index advisor analyzes your queries to recommend new or altered indexes that can improve your query performance. You can view the index advisor's recommendations in the Google Cloud console using either of the following approaches:

- Run a query and view its [query execution plan](https://docs.cloud.google.com/spanner/docs/query-execution-plans)
- Use the Spanner [Query insights dashboard](https://docs.cloud.google.com/spanner/docs/using-query-insights)

To view the `CREATE INDEX` and `ALTER INDEX` recommendations, you can use the Google Cloud console.

For more information about Spanner indexes, see [Secondary indexes](https://docs.cloud.google.com/spanner/docs/secondary-indexes) .

## Limitations

Spanner index advisor has the following limitations:

- Only provides `CREATE INDEX` and `ALTER INDEX` recommendations. Doesn't provide `DROP INDEX` recommendations for existing indexes.

- An index recommendation is only shown if it provides a noticeable performance benefit.

- If you are a [fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) user or if you don't have DDL access, you can't execute index recommendation DDL statements. You can copy and save the recommendation.

## Use the query execution plan

To view and apply index advisor recommendations from a query's execution plan, follow these steps.

### View recommendations

To view the query execution plan, run a query in the Google Cloud console:

1.  Go to the Spanner **Instances** page in Google Cloud console.

2.  Select the instance that contains the database that you want to query.

3.  Select the name of the database you want to query.

4.  In the navigation menu, click **Spanner Studio** .

5.  Open a new SQL editor tab.

6.  In the editor pane, enter your SQL query.

7.  Click **Run** .

8.  After the query has finished running, to see the query execution plan, click the **Explanation** tab.

    The information panel shows detailed information about the query. If Spanner determines that a new or altered index can improve your query performance, then an index recommendations card is displayed.

9.  To view the index recommendation DDL statement, in the **Index recommendation** card, click **View details** to view the index.

### Apply recommendations

The Spanner index recommendation provides complete `CREATE INDEX` and `ALTER INDEX` DDL statements for recommended indexes.

To apply the index advisor's recommendation, copy and run the index advisor's DDL statement into the Spanner Studio editor exactly as presented.

1.  In the **Index recommendation** pane, select the checkbox next to the DDL statements that you want to copy.

2.  Click **Copy to new tab** .

3.  In the new Spanner Studio editor tab, run the copied DDL statement.

## Use the Query insights dashboard

To view and apply index advisor recommendations from the Query insights dashboard, follow these steps.

### View recommendations

1.  Go to the Spanner **Instances** page in Google Cloud console.

2.  Select the name of the instance containing the database you want to query.

3.  Select the name of the database you want to query.

4.  In the navigation menu, click **Query insights** .

5.  View the **TopN queries and tags** table.

    The table shows a **Recommendation** column. If Spanner determines that a new or altered index can improve your query performance, then an index recommendation is displayed. To view what this looks like in the Google Cloud console, see [Identify a potentially problematic query or request tag](https://docs.cloud.google.com/spanner/docs/using-query-insights#filter-db-load) .

6.  To view the index recommendation DDL statement, click **Index recommendation** .

### Apply recommendations

The Spanner index recommendation provides complete `CREATE INDEX` and `ALTER INDEX` DDL statements for recommended indexes.

To apply the index advisor's recommendation, copy and run the index advisor's DDL statement into the Spanner Studio editor exactly as presented.

1.  In the **Index recommendation** pane, select the DDL statements you want to copy.

    Spanner generates recommendations for TopN queries every 12 hours for the TopN queries executed in the preceding 12 hours. The **Index recommendation** pane displays a record of how long ago it was last refreshed.

    ![View of an index recommendation.](https://docs.cloud.google.com/static/spanner/docs/images/index-recommendation.png)

2.  Click **Copy to Spanner Studio** . Google Cloud console displays the **Spanner Studio** editor.

    If you are a fine-grained access control user, you don't see the **Copy to Spanner Studio** button, and you can't run the DDL statement.

3.  In the Spanner Studio editor, run the copied DDL statement.

## What's next

- Learn more about Spanner [secondary indexes](https://docs.cloud.google.com/spanner/docs/secondary-indexes) .

- Learn more [SQL best practices](https://docs.cloud.google.com/spanner/docs/sql-best-practices) .

- Learn how to [Troubleshoot performance regressions](https://docs.cloud.google.com/spanner/docs/troubleshooting-performance-regressions) .
