---
name: documents/docs.cloud.google.com/spanner/docs/monitor-columnar-engine
uri: https://docs.cloud.google.com/spanner/docs/monitor-columnar-engine
title: Monitor columnar engine performance
description: Monitor performance and memory usage of the Spanner columnar engine using system tables and metrics in Cloud Monitoring.
data_source: docs.cloud.google.com
---

> **Note:** This feature is available with the Spanner Enterprise edition and Enterprise Plus edition. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

This page describes how to monitor Spanner's columnar engine. You can monitor using:

- [The query execution plan](https://docs.cloud.google.com/spanner/docs/monitor-columnar-engine#columnar-data-query-plan)
- [Storage usage](https://docs.cloud.google.com/spanner/docs/monitor-columnar-engine#monitor-storage-usage) graphs in [system insights](https://docs.cloud.google.com/spanner/docs/monitor-columnar-engine#monitor-storage-usage)

## View columnar data in the query plan

1.  Run a query.

2.  View its [query plan](https://docs.cloud.google.com/spanner/docs/query-execution-plans) .

3.  In the **Query summary** section of the query execution, check for the **Columnar read share** information.

    The following image shows the **Columnar read share** percentage:

    ![The \*\*Query summary\*\* shows the columnar read share percentage.](https://docs.cloud.google.com/static/spanner/docs/images/columnar-read-share.svg "image_tooltip")

This metric shows the percentage of bytes read from columnar storage out of total bytes read from both row-based and columnar storage. A high percentage is optimal, while a low percentage suggests that much of the data remains unconverted to columnar format.

## Columnar data storage metrics

To view columnar data storage metrics:

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Select the instance you want to view metrics for.

3.  Click **System insights** .

4.  In the **Storage usage** section, you can see two new metrics:

    **columnar storage - hdd** and **columnar storage - ssd** . These metrics show the columnar representation usage of HDD and SSD. The total storage metrics include both PAX and columnar data. You can see these metrics on storage usage graphs on the **System insights** page. For example:

    ![The amount of columnar storage ssd.](https://docs.cloud.google.com/static/spanner/docs/images/columnar-storage.svg)

## Query and table operations statistics

`SPANNER_SYS.QUERY_STATS_TOP_*` tables have a column named `AVG_COLUMNAR_READ_SHARE` , which is an average percentage of bytes read from columnar storage out of total bytes read from both row-based and columnar storage. A high percentage is optimal, while a low percentage suggests that much of the data remains unconverted to columnar format.

## What's next

- Learn about [columnar engine](https://docs.cloud.google.com/spanner/docs/columnar-engine) .
- Learn how to [enable columnar engine](https://docs.cloud.google.com/spanner/docs/configure-columnar-engine) .
- Learn how to [query columnar data](https://docs.cloud.google.com/spanner/docs/query-columnar-data) .
