---
name: documents/docs.cloud.google.com/spanner/docs/databoost/databoost-quotas
uri: https://docs.cloud.google.com/spanner/docs/databoost/databoost-quotas
title: Monitor and manage Data Boost quota usage
description: Understand the quotas and limits associated with using Spanner Spanner Data Boost for your data processing workloads.
data_source: docs.cloud.google.com
---

Spanner Data Boost is subject to quotas that limit the number of concurrent `ExecuteStreamingSQL` and `StreamingRead` calls executing per project and per region. When the number of concurrent requests exceeds the quota, requests fail with `RESOURCE_EXHAUSTED` errors.

## Check your quota

To check the concurrency quota in your project, follow these steps:

1.  Go to the Quotas page in the Google Cloud console.

2.  In the **Filter** field, enter `databoost` . From the list, select one of the following:

    - **DataBoostQuotaPerProjectPerRegion** to view the Data Boost concurrent requests quota.
    - **DataBoostMilliQuotaPerProjectPerRegion** to view the Data Boost concurrent requests quota in milli-operations.

3.  In the resulting table, locate your region and view the limit for that region.

> **Note:** To check the default quota for regions, select **DataBoost operations per region (default)** or **DataBoost concurrent requests in milli-operations per region (default)** instead.

## Monitor quota usage

To monitor your quota usage, follow these steps:

1.  In the Google Cloud console, go to **Monitoring** .  

2.  If **Metrics Explorer** is shown in the navigation menu, select it. Otherwise, click **Resources** , and then select **Metrics Explorer** .

3.  At the top of the page, select a time interval.

4.  In the **Metric** drop-down list, in the **Filter by resource or metric name** field, enter `consumer` and press `Enter` to narrow the search.

5.  To view the Data Boost concurrent requests quota:
    1.  In the list, select **Consumer Quota \> Quota \> Concurrent Quota usage** , and then click **Apply** .
    2.  Click **+ ADD FILTER** to create a filter.
    3.  In the **Label 1** drop-down list, select **quota_metric** .
    4.  In the **Value 1** text field, enter or select **spanner.googleapis.com/data_boost_quota** .

6.  To view the Data Boost concurrent requests quota in milli-operations:

    1.  In the list, select **Consumer Quota \> Quota \> Rate Quota usage** , and then click **Apply** .
    2.  Click **+ ADD FILTER** to create a filter.
    3.  In the **Label 1** drop-down list, select **quota_metric** .
    4.  In the **Value 1** text field, enter or select **spanner.googleapis.com/data_boost_milli_quota** .

    Metrics explorer shows a line chart of quota usage by region.

7.  Optional: Under **Display** , for **Widget type** , select **Stacked bar chart** .

## Monitor quota errors and limits

To monitor quota errors due to Data Boost concurrent requests quota, follow these steps:

1.  In Metrics explorer, select the metric **Consumer Quota \> Quota \> Quota exceeded error** .

2.  Add a filter for **quota_metric** equals **spanner.googleapis.com/data_boost_quota** .

To monitor the limit for the Data Boost concurrent requests milli-operations quota, follow these steps:

1.  In Metrics explorer, select the metric **Consumer Quota \> Quota \> Quota limit** .

2.  Add a filter for **quota_metric** equals **spanner.googleapis.com/data_boost_milli_quota** .

## Set an alert for Data Boost usage

You can create an alert policy that notifies you when the number of concurrent [partitioned queries](https://docs.cloud.google.com/spanner/docs/reads#read_data_in_parallel) that request Data Boost (concurrent `ExecuteStreamingSQL` and `StreamingRead` calls) per project and per region exceeds a particular threshold. To do so, follow these steps:

1.  Follow the instructions in [Create metric-threshold alerting policies](https://cloud.google.com/monitoring/alerts/using-alerting-ui) .

2.  In the **Select a metric** drop-down list, in the **Filter by resource or metric name** field, enter `consumer` and press `Enter` to narrow the search.

3.  To set an alert on the Data Boost concurrent requests quota usage:

    1.  In the **Select a metric** list, select **Consumer Quota \> Quota \> Concurrent Quota usage** , and then click **Apply** .
    2.  In the **Add filters** section, click **Add a filter** to create a filter.
    3.  In the **Filter** drop-down list, select **quota_metric** .
    4.  In the **Value** text field, enter or select **spanner.googleapis.com/data_boost_quota** and click **Done** .
    5.  Continue with creating an alert policy and set the threshold to some percentage of the quota.

    For example if the default quota is 200 and you want to be notified when the number of concurrent requests reaches 80% of maximum, enter 160 in the **Threshold value** field.

4.  To set an alert on the Data Boost concurrent requests milli-operations quota usage:

    1.  In the **Select a metric** list, select **Consumer Quota \> Quota \> Rate Quota usage** , and then click **Apply** .
    2.  In the **Add filters** section, click **Add a filter** to create a filter.
    3.  In the **Filter** drop-down list, select **quota_metric** .
    4.  In the **Value** text field, enter or select **spanner.googleapis.com/data_boost_milli_quota** and click **Done** .
    5.  Continue with creating an alert policy and set the threshold to some percentage of the quota.

    For example if the default quota is 1000000 and you want to be notified when the request milli-operations reach 80% of the limit, enter 800000 in the **Threshold value** field.

## Handle quota errors

If the rate of quota-exceeded errors is high, when using Dataflow with Data Boost, we recommend that you adjust the maximum number of workers in your Dataflow job to avoid exceeding the Data Boost quota. You can also apply for a higher quota. Your workload might be limited either by the Data Boost concurrent requests quota or the milli-operations quota. You might need to increase your milli-operations quota after increasing the concurrent requests quota.

## Block a principal from consuming Data Boost resources

If one principal is consistently exceeding the amount of Data Boost resources that they can be reasonably expected to consume, you can block the principal from using Data Boost resources by revoking the `spanner.databases.useDataBoost` Identity and Access Management (IAM) permission from the principal. You can automate revoking the permission by configuring an alert's notification channel as a webhook that invokes a Cloud Function. For more information, see the following topics:

- [Create a notification channel](https://cloud.google.com/monitoring/support/notification-options#creating_channels)

- [Cloud Functions](https://cloud.google.com/functions)

- [Remove database-level permissions](https://cloud.google.com/spanner/docs/grant-permissions#remove-iam-db-role)

- [Modify the allow policy](https://cloud.google.com/iam/docs/granting-changing-revoking-access#modifying-policy) for sample code that revokes a role.

  > **Note:** A best practice is to add the `spanner.databases.useDataBoost` permission to a custom IAM role.

## What's next

- Learn about Data Boost in [Data Boost overview](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) .
