---
name: documents/docs.cloud.google.com/spanner/docs/find-hotspots-in-database
uri: https://docs.cloud.google.com/spanner/docs/find-hotspots-in-database
title: Find hotspots in your database
description: Use the Hotspot insights dashboard to find and manage database hotspots, reduce latency and high CPU usage, and improve overall database performance.
data_source: docs.cloud.google.com
---

This document describes how to use the Hotspot insights dashboard to detect hotspots in your Spanner database.

## Hotspot insights overview

Hotspots cause latency in your Spanner database. The Hotspot insights dashboard helps you detect the splits affected by hotspots. Use the following steps to determine if hotspots are causing latency and if so, how to resolve the issue:

1.  [Open the dashboard.](https://docs.cloud.google.com/spanner/docs/find-hotspots-in-database#dashboard)
2.  [Determine whether hotspots need your intervention.](https://docs.cloud.google.com/spanner/docs/find-hotspots-in-database#determine-intervention)
3.  [Identify problematic hot splits.](https://docs.cloud.google.com/spanner/docs/find-hotspots-in-database#identify-hot-splits)

Hotspot insights is available in single-region, multi-region, and dual-region configurations.

## Pricing

There is no additional cost for Hotspot insights.

## Data retention

Data retention policies for the Hotspot insights charts and the TopN splits table are based on the underlying `SPANNER_SYS.SPLIT_STATS_TOP_*` tables. For specific retention policies, see [Hot split statistics data retention](https://docs.cloud.google.com/spanner/docs/introspection/hot-split-statistics#data_retention) .

## Required roles

You might need different IAM roles and permissions, depending on whether you are an IAM user or a fine-grained access control user.

### Identity and Access Management (IAM) user

To get the permissions that you need to view the **Hotspot insights** page, ask your administrator to grant you the following IAM roles on the instance:

- All:
  - Cloud Spanner Viewer ( `roles/spanner.viewer` )
  - [Cloud Spanner Database Reader](https://docs.cloud.google.com/iam/docs/roles-permissions/spanner#spanner.databaseReader) ( `roles/spanner.databaseReader` )

The following permissions in the [Cloud Spanner Database Reader](https://docs.cloud.google.com/iam/docs/roles-permissions/spanner#spanner.databaseReader) ( `roles/spanner.databaseReader` ) role are required to view the **Hotspot insights** page:

- `spanner.databases.beginReadOnlyTransaction`
- `spanner.databases.select`
- `spanner.sessions.create`

### Fine-grained access control user

If you are a fine-grained access control user, ensure that you:

- Have the [Cloud Spanner Viewer](https://docs.cloud.google.com/iam/docs/understanding-roles#spanner.viewer) ( `roles/spanner.viewer` )
- Have fine-grained access control privileges and are granted the `spanner_sys_reader` system role or one of its member roles.
- Select the `spanner_sys_reader` or a member role as your current system role on the database **Overview** page.

> **Note:** If you already have an IAM database-level permission, such as `spanner.databases.select` , the Google Cloud console assumes that you are an IAM user. You can't select the `spanner_sys_reader` or a member role on the database overview page as an IAM user.

For more information, see [fine-grained access control overview](https://docs.cloud.google.com/spanner/docs/fgac-about) and [Fine-grained access control system roles](https://docs.cloud.google.com/spanner/docs/fgac-system-roles) .

## Open the Hotspot insights dashboard

The **Hotspot insights** dashboard shows the peak split CPU usage percentage. This metric is an abstract percentage of 0 to 100 that reflects the amount of CPU used when rows within a split are accessed.

To view the **Hotspot insights** dashboard for a database, do the following:

1.  In the Google Cloud console, open the **Spanner** page.

2.  Select an instance from the list.

3.  In the navigation menu, click the **Hotspot insights** tab.

4.  In the **database** field, select a database from the list. The dashboard shows the peak split CPU usage score for the database.

The dashboard includes the following elements:

- **Peak split CPU usage score** graph: a higher CPU usage score (such as near 100) indicates that the split is hot and very likely causing a hotspot on the server compared to lower scores.
- **Database field** : filters the hot splits information on a specific database or all databases.
- **Time range filter** : filters the peak splits CPU usage by 1-minute increments up to a total of 6 hours.
- **TopN splits table** : displays the list of top splits sorted by split CPU usage scores.

![The Hotspot Insights dashboard in Spanner, showing a graph of peak split CPU usage, a database selector, a time range filter, and a table of the top N hot splits.](https://docs.cloud.google.com/static/spanner/docs/images/hotspot-insights.png)

**Understanding data in the TopN Splits Table:** The **TopN splits** table populates data from the underlying `SPANNER_SYS.SPLIT_STATS_TOP_*` tables based on the time range you select. For more information, see [Hot split statistics data retention](https://docs.cloud.google.com/spanner/docs/introspection/hot-split-statistics#data-retention) .

**Interpreting Rows from `10MINUTE` or `HOUR` tables:** Rows sourced from `SPANNER_SYS.SPLIT_STATS_TOP_10MINUTE` or `SPANNER_SYS.SPLIT_STATS_TOP_HOUR` represent aggregated data over their respective intervals. As described in [Table event aggregation](https://docs.cloud.google.com/spanner/docs/introspection/hot-split-statistics#view-event-aggregation) , the `CPU_USAGE_SCORE` in these rows is the *maximum* score seen in any underlying 1-minute sub-interval, and `UNSPLITTABLE_REASONS` is a *union* of reasons.

## Determine if hotspots need intervention

If you see a spike or an elevation in the graph that corresponds to the overall latency and a persistent high peak split CPU usage score, then you might need to investigate further.

Review the graph to explore these questions:

- **Which database is experiencing the latency degradation?** Select different databases from the **Databases** list to find the databases with the highest latency. To find out which database has the highest load, you can also review the [**Latency** chart](https://docs.cloud.google.com/spanner/docs/latency-metrics#overview) for databases in the Google Cloud console.

  ![A line graph titled 'Peak split CPU usage score', showing the peak split CPU usage score over time. The graph displays a spike, which can indicate a hotspot and potential latency issues.](https://docs.cloud.google.com/static/spanner/docs/images/peak-split-graph.png)

- **Is the latency high?** Is the latency high as compared to the expected latency for the workload? Is the graph spiking or becoming more elevated over time? If you don't see high latency, then hotspots aren't a problem.

- **Is the high peak split CPU usage score at 100%?** Is the graph spiking or becoming more elevated over time? If you don't see persistent 100% peak split CPU usage percentages for at least 10 minutes, then hotspots might not be a problem. If the peak split CPU usage percentage is high for longer than 10 minutes, then you might want to investigate further to see if the database has higher than expected latency levels.

If you see 100% peak split CPU usage percentages for longer than 10 minutes, then hotspots might need your intervention. Next, you can continue the debugging journey by identifying the hot splits in your database.

## Identify problematic hot splits

To identify a potentially problematic split that has hotspots, see the **TopN splits** section in the Google Cloud console, as shown in the following.

![A table of the 'TopN splits' in Spanner, which lists potentially problematic splits. The table includes columns for 'Interval end', 'Split start', 'Split limit', 'Split CPU usage score', 'Affected tables', and 'Unsplittable reasons'.](https://docs.cloud.google.com/static/spanner/docs/images/topn-splits-table.png)

The **TopN splits** table provides an overview of the splits that might be hot during the chosen time window, sorted from latest to earliest. The number of TopN splits is limited to 100.

For the graphs, Spanner fetches data from the TopN splits statistics table, with a granularity of one minute. The value for each data point in the graphs represents the average value over an interval of one minute.

The table shows the following properties:

- **Interval end** : the date and time when the high peak CPU usage ends.
- **Split start** : the starting key of the range of rows in the split. If the split start is \<begin\>, it indicates the beginning of the key range of the database.
- **Split limit** : the limit key of the range of rows in the split. If the limit key is \<end\>, it indicates the end of the key range of the database.
- **Split CPU usage score** : an abstract score of between 0 and 100 that reflects the amount of CPU used by accesses to the rows within the split on a single server. Use the CPU usage score to help evaluate whether you have hotspots.
- **Affected tables** : the tables whose rows might be in the split.
- **Unsplittable reasons** : An array of reasons why Spanner cannot split a hot split further. The presence of values here indicates that load-based splitting is unable to mitigate the hotspot for the reasons listed. For more information, see [`UNSPLITTABLE_REASONS` types](https://docs.cloud.google.com/spanner/docs/introspection/hot-split-statistics#unsplit-reason-types) .

### Analyze unsplittable reasons

The **TopN splits** table lets you drill down into *which specific splits* are affected by these reasons at particular times, as shown in the **Unsplittable reasons** column.

### Example diagnosis workflow

Here's a typical workflow for debugging hotspots using the dashboard:

1.  **Observe the performance issue:** Notice increased latency or errors in your application.
2.  **Open Hotspot insights:** Navigate to the Hotspot insights dashboard in the Google Cloud console for the relevant Spanner database. Select the time range corresponding to the issue.
3.  **Examine graph:**
    - Check the **Peak split CPU usage score** graph for sustained high values, for example, \>50%, especially approaching 100% lasting for at least 10 minutes.
4.  **Identify affected splits and correlate findings:** If CPU usage is high, go to the **TopN splits** table. Filter or sort to find the splits with the highest **Split CPU usage score** during the impact time. Examine the `UNSPLITTABLE_REASONS` column for these top splits:
    - **High split CPU usage score and unsplittable reasons:** This is a strong signal that the performance issue is related to hotspots that Spanner cannot automatically resolve. The type of reason, such as `HOT_ROW` or `MOVING_HOT_SPOT` , provides a crucial clue.
    - **High split CPU usage score and no unsplittable reasons:** The hotspot might be new, and Spanner might still be in the process of splitting. Alternatively, the issue might be responding to changes in the workload, which requires no action from you.
5.  **Understand the reasons:** Note the specific codes in the `UNSPLITTABLE_REASONS` array.
6.  **Mitigate:** Based on the identified reasons, refer to [`UNSPLITTABLE_REASONS` types](https://docs.cloud.google.com/spanner/docs/introspection/hot-split-statistics#unsplit-reason-types) for detailed explanations and recommended mitigation strategies, which usually involve schema design changes or workload adjustments.

## What's next

- Learn about [Split hotspot statistics](https://docs.cloud.google.com/spanner/docs/introspection/hot-split-statistics)
