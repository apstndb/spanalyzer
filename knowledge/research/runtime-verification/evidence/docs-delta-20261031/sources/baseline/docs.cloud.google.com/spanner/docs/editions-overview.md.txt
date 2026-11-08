---
name: documents/docs.cloud.google.com/spanner/docs/editions-overview
uri: https://docs.cloud.google.com/spanner/docs/editions-overview
title: Spanner editions overview
description: Explore Spanner editions (Standard, Enterprise, Enterprise Plus) to choose the right features, performance, and cost for your workloads.
data_source: docs.cloud.google.com
---

This page describes Spanner editions and its key features.

Spanner editions is a tier-based pricing model that provides different capabilities at different price points. Spanner offers the following editions to support your various business and application needs:

- **Standard edition** : provides a comprehensive suite of established capabilities that include all of the features that are Generally Available (GA) prior to September 24, 2024 along with selected additional capabilities, such as reverse ETL from BigQuery and scheduled backups, in single-region (regional) instance configurations.

- **Enterprise edition** : builds on the Standard edition and offers multi-model capabilities including Spanner Graph, full-text search, and Vector Search. It also offers enhanced operational simplicity and data protection using managed autoscaling and incremental backups.

- **Enterprise Plus edition** : designed for the most demanding workloads that require 99.999% availability with multi-region instance configurations and geo-partitioning support. This tier includes all Standard edition and Enterprise edition features.

## Editions features

The following table lists the features available for each edition.

|                                                                                      | Standard                                                                                                                                                                                                                                                                                      | Enterprise                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | Enterprise Plus                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
|--------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [Availability SLA](https://cloud.google.com/spanner/sla)                             | 99.99% availability SLA                                                                                                                                                                                                                                                                       | 99.99% availability SLA                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | Up to 99.999% availability SLA                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| [Configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations) | Regional                                                                                                                                                                                                                                                                                      | Regional Optional custom read-only replicas                                                                                                                                                                                                                                                                                                                                                                                                                                                  | Regional, dual-region, and multi-region Optional custom read-only replicas                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| Multi-model capabilities                                                             | Relational ( [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/overview) , [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/overview) ) [Key-value](https://docs.cloud.google.com/spanner/docs/non-relational/overview)               | Relational ( [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/overview) , [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/overview) ) [Key-value](https://docs.cloud.google.com/spanner/docs/non-relational/overview) [Spanner Graph](https://docs.cloud.google.com/spanner/docs/graph/overview)                                                                                                                                   | Relational ( [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/overview) , [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/overview) ) [Key-value](https://docs.cloud.google.com/spanner/docs/non-relational/overview) [Spanner Graph](https://docs.cloud.google.com/spanner/docs/graph/overview)                                                                                                                                                                                                                   |
| Search capabilities                                                                  | —                                                                                                                                                                                                                                                                                             | [Full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) Vector search ( [KNN](https://docs.cloud.google.com/spanner/docs/find-k-nearest-neighbors) , [ANN](https://docs.cloud.google.com/spanner/docs/find-approximate-nearest-neighbors) )                                                                                                                                                                                                                          | [Full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) Vector search ( [KNN](https://docs.cloud.google.com/spanner/docs/find-k-nearest-neighbors) , [ANN](https://docs.cloud.google.com/spanner/docs/find-approximate-nearest-neighbors) )                                                                                                                                                                                                                                                                                                          |
| Resource management                                                                  | [Open source Autoscaler](https://docs.cloud.google.com/spanner/docs/autoscaler-tool-overview)                                                                                                                                                                                                 | [Open source Autoscaler](https://docs.cloud.google.com/spanner/docs/autoscaler-tool-overview) [Managed autoscaler](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) [Asymmetric read-only autoscaling](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#asymmetric-read-only-autoscaling) [Locality groups](https://docs.cloud.google.com/spanner/docs/create-manage-locality-groups) [Tiered storage](https://docs.cloud.google.com/spanner/docs/tiered-storage) | [Open source Autoscaler](https://docs.cloud.google.com/spanner/docs/autoscaler-tool-overview) [Managed autoscaler](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) [Asymmetric read-only autoscaling](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#asymmetric-read-only-autoscaling) [Locality groups](https://docs.cloud.google.com/spanner/docs/create-manage-locality-groups) [Tiered storage](https://docs.cloud.google.com/spanner/docs/tiered-storage) [Geo-partitioning](https://docs.cloud.google.com/spanner/docs/geo-partitioning) |
| Analytics                                                                            | [BigQuery federation](https://docs.cloud.google.com/bigquery/docs/spanner-federated-queries) [Spanner Data Boost](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) [Reverse ETL (BigQuery to Spanner)](https://docs.cloud.google.com/bigquery/docs/export-to-spanner) | [Columnar engine](https://docs.cloud.google.com/spanner/docs/columnar-engine) [BigQuery federation](https://docs.cloud.google.com/bigquery/docs/spanner-federated-queries) [Spanner Data Boost](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) [Reverse ETL (BigQuery to Spanner)](https://docs.cloud.google.com/bigquery/docs/export-to-spanner)                                                                                                                  | [Columnar engine](https://docs.cloud.google.com/spanner/docs/columnar-engine) [BigQuery federation](https://docs.cloud.google.com/bigquery/docs/spanner-federated-queries) [Spanner Data Boost](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) [Reverse ETL (BigQuery to Spanner)](https://docs.cloud.google.com/bigquery/docs/export-to-spanner)                                                                                                                                                                                                  |
| Data protection                                                                      | [Standard backups](https://docs.cloud.google.com/spanner/docs/backup) [7-day PITR](https://docs.cloud.google.com/spanner/docs/pitr) [Scheduled backups](https://docs.cloud.google.com/spanner/docs/backup#backup-schedules)                                                                   | [Standard backups](https://docs.cloud.google.com/spanner/docs/backup) [7-day PITR](https://docs.cloud.google.com/spanner/docs/pitr) [Scheduled backups](https://docs.cloud.google.com/spanner/docs/backup#backup-schedules) [Incremental backups](https://docs.cloud.google.com/spanner/docs/backup#incremental-backups)                                                                                                                                                                     | [Standard backups](https://docs.cloud.google.com/spanner/docs/backup) [7-day PITR](https://docs.cloud.google.com/spanner/docs/pitr) [Scheduled backups](https://docs.cloud.google.com/spanner/docs/backup#backup-schedules) [Incremental backups](https://docs.cloud.google.com/spanner/docs/backup#incremental-backups)                                                                                                                                                                                                                                                     |
| [CUDs](https://docs.cloud.google.com/spanner/docs/cuds)                              | 20% for 1 year 40% for 3 years                                                                                                                                                                                                                                                                | 20% for 1 year 40% for 3 years                                                                                                                                                                                                                                                                                                                                                                                                                                                               | 20% for 1 year 40% for 3 years                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |

## What you need to do

If you haven't selected an edition for your legacy Spanner instance, it has automatically upgraded to the lowest edition that matches your usage pattern to avoid workload disruption.

In general:

- Regional instances upgraded to the Standard edition.
- Regional instances with additional configurable read-only replicas or those using Enterprise edition features upgraded to the Enterprise edition.
- Multi-region instances or those using Enterprise Plus edition features upgraded to the Enterprise Plus edition.

If you are a Spanner customer under Google Cloud commitments with discounts on legacy SKUs, **no action is required on your part.** You can continue to use legacy SKUs until the expiration of your contract. You also have the option to upgrade to Spanner editions. We recommend contacting your sales team to understand your existing contractual obligation and renew your contracts to include the new editions SKUs, so you can optimize your total cost of ownership and get access to new capabilities that are only offered in editions.

## Monitor edition feature usage

You can monitor the usage of Enterprise edition and Enterprise Plus edition edition features in your instance. To do so, use the [Feature usage](https://docs.cloud.google.com/monitoring/api/metrics_gcp_p_z#gcp-spanner) ( `instance/edition/feature_usage` ) monitoring metric. The following features are shown in this metric when you use them in your instance.

- Asymmetric autoscaling
- Columnar engine
- Full-text search
- Geo-partitioning
- Incremental backups
- KNN vector search: includes use of the KNN vector distance functions
- Managed autoscaler
- Scheduled backups
- Spanner Graph
- Tiered storage
- Vector search: includes use of the ANN vector distance functions and vector index

> **Note:** The Feature usage metric is sampled every 60 seconds, and might take up to 120 seconds to become visible.

To view the edition feature usage metric in the Google Cloud console, follow these steps:

1.  In the Google Cloud console, go to **Monitoring** :

2.  In the navigation menu, select **Metrics explorer** .

3.  In the **Metric** field, click the **Select a metric** drop-down.

4.  In the **Filter by resource or metric name** field, select **Cloud Spanner Instance \> Instance \> Feature usage** , and then click **Apply** .

5.  In the **Aggregation** field, select **Unaggregated** .

6.  Select **Table** or **Both** as the table type instead of Chart.

    The table lists each higher-tier edition feature that is being used by your instance and database.

    Optionally, you can click view_column **Column display options** to display or hide columns to display in the table. The `name` column should generally be ignored in favor of `instance_id` , `instance_config` , `database` and `feature` .

To see a full list of Google Cloud metrics, see [Google Cloud metrics](https://docs.cloud.google.com/monitoring/api/metrics_gcp) .

## Pricing

For information about Spanner editions pricing, see [Spanner pricing](https://cloud.google.com/spanner/pricing) . To help control cost, it is possible to prevent specific Spanner editions from being created by using an [organization policy constraint](https://docs.cloud.google.com/spanner/docs/spanner-custom-constraints) .

## Frequently asked questions

**What are the changes if I'm using a Spanner free trial instance?**  
If you're using a free trial instance, it will default to the Enterprise edition when you [upgrade it to a paid instance](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart#upgrade) . Once in Enterprise edition, you can upgrade the instance to the Enterprise Plus edition, or contact support to downgrade to the Standard edition.

<!-- -->

**Can I upgrade my instance?**  
Yes, you can [upgrade your instance](https://docs.cloud.google.com/spanner/docs/create-manage-instances#upgrade-edition) to the Enterprise edition or Enterprise Plus edition. There is no data migration involved when you change the edition of your instance. The edition upgrade takes approximately 10 minutes to complete with zero downtime.

**Can I downgrade my instance?**  
Yes, you can [downgrade your instance](https://docs.cloud.google.com/spanner/docs/create-manage-instances#downgrade-edition) to a lower-tier edition. You must stop using the higher-tier edition features in order to downgrade. There is no data migration involved when you change the edition of your instance. The edition downgrade takes approximately 10 minutes to complete with zero downtime.

**Will granular instances continue to be supported with editions?**  
Yes, granular instances is supported in all Spanner editions. The minimum compute capacity you can set is 100 processing units or one node (1000 processing units).

**Will my instance undergo data migration when it upgrades to editions?**  
No, your instance doesn't undergo data migration when it upgrades to editions. This is a configuration change.

**How do I stop using Enterprise edition or Enterprise Plus edition features in order to downgrade my instance's edition?**

- Enterprise edition and Enterprise Plus edition features:

  - Custom read-only replicas: [Move your instance](https://docs.cloud.google.com/spanner/docs/move-instance) to a regional instance configuration or [delete your instance](https://docs.cloud.google.com/spanner/docs/create-manage-instances#delete-instance) .
  - Spanner Graph: [Delete all property graph schemas](https://docs.cloud.google.com/spanner/docs/graph/create-update-drop-schema#drop-property-graph-schema) in your instance.
  - Full-text search: [Delete all search indexes](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#drop-search-index) in your instance.
  - Vector search: Stop using all [KNN](https://docs.cloud.google.com/spanner/docs/find-k-nearest-neighbors) and [ANN](https://docs.cloud.google.com/spanner/docs/find-approximate-nearest-neighbors) distance functions, and [delete all vector indexes](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#drop-vector-index) in your instance.
  - Managed autoscaler: Change your instance from using the managed autoscaler to [use manual scaling](https://docs.cloud.google.com/spanner/docs/create-manage-instances#remove-managed-autoscaler) .

- Enterprise Plus edition only features:

  - Dual-region and multi-region instance configurations: [Move your instance](https://docs.cloud.google.com/spanner/docs/move-instance) to a regional instance configuration or [delete your instance](https://docs.cloud.google.com/spanner/docs/create-manage-instances#delete-instance) .
  - Geo-partitioning: [Delete all partitions](https://docs.cloud.google.com/spanner/docs/create-manage-partitions#delete-partition) in your instance.

## What's next

- Learn how to [create and manage your instances](https://docs.cloud.google.com/spanner/docs/create-manage-instances) .
