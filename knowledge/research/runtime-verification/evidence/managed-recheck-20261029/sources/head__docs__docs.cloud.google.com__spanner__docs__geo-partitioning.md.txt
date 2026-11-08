---
name: documents/docs.cloud.google.com/spanner/docs/geo-partitioning
uri: https://docs.cloud.google.com/spanner/docs/geo-partitioning
title: Geo-partitioning overview
description: Implement geo-partitioning in Spanner to control data placement and improve performance for globally distributed applications.
data_source: docs.cloud.google.com
---

> **Preview — [Geo-partitioning](https://docs.cloud.google.com/spanner/docs/geo-partitioning)**
>
> This feature is subject to the "Pre-GA Offerings Terms" in the General Service Terms section of the [Service Specific Terms](https://docs.cloud.google.com/terms/service-terms#1) . Pre-GA features are available "as is" and might have limited support. For more information, see the [launch stage descriptions](https://cloud.google.com/products/#product-launch-stages) .

> **Note:** This feature is available with the Spanner Enterprise Plus edition. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

This page introduces geo-partitioning and explains how it works in Spanner.

Spanner offers [regional and multi-regional instance configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations) , which let you replicate your data across geographic locations. Geo-partitioning lets you further segment and store rows in your database table across different instance configurations.

## Benefits and use cases

Geo-partitioning lets you partition rows in your database, providing the following benefits:

- **Regional latency in a global database** : By using geo-partitioning, Spanner manages your data in a single, unified database across geographically distributed locations, while ensuring low latency for regional access. Using geo-partitioning simplifies operations and reduces complexity compared to managing multiple sharded databases.
- **Global database capabilities** : Geo-partitioning offers database features such as global transactions, data movement between regions, and uniqueness enforcement across geographical regions.
- **Data residency compliance** : Spanner provides data residency commitments at the placement level. For information, see [data residency compatibility for databases that use geo-partitioning](https://docs.cloud.google.com/spanner/docs/data-residency#databases-that-use-geo-partitioning) .

The following are common use cases:

- **User-related data** : Geo-partitioning user-related data to process and store data in a region that is closest to the user.

- **Localized data** : Location-specific information like traffic and special events.

## How geo-partitioning works

All Spanner instances have a main instance partition that is called the `default` instance partition. If you don't create additional instance partitions, all database objects are stored in the default partition, which is in the same location as your instance configuration. If you want to partition the data in a database, you must create additional instance partitions in your instance.

To use geo-partitioning in a database:

1.  [Create additional instance partitions](https://docs.cloud.google.com/spanner/docs/create-manage-partitions#create-partition) in your instance. These user-created instance partitions have their own configuration (either regional or multi-region) and node count.

2.  [Create your database](https://docs.cloud.google.com/spanner/docs/create-manage-databases#create-database) like you would normally. The database has a default placement that is associated with the default instance partition of the instance.

3.  [Create placements](https://docs.cloud.google.com/spanner/docs/create-manage-data-placements#create-placement) in your database that are associated with the additional instance partitions. Your database can interact with the additional instance partitions that were created in the same instance.

4.  [Create placement tables](https://docs.cloud.google.com/spanner/docs/create-manage-data-placements#create-table) that have a placement key attribute. You must use the placement key in your DML statements to specify which instance partition the row data resides in. If you create non-placement tables in your database, Spanner stores that data in the default instance partition.

The placement key for each row in a placement table must be assigned to one of the following:

- A value which matches the name of one of the user-created placements defined for that database; or

- The placement key value, `default` , which stores the data in the default placement.

For instructions on how to use instance partitions, see [create and manage instance partitions](https://docs.cloud.google.com/spanner/docs/create-manage-partitions) .

## Important considerations

Consider the following before creating your instance partitions, placements, and placement tables:

- **Instance partition location** : Carefully select the instance partition regions that provide the most benefits for your application.

  - You must create instance partitions in an instance with a multi-region instance configuration.

  - For better latency, we recommend that you select a multi-region default instance partition location that has read-write and read-only regions that cover all jurisdictions required by your application. Then, create additional instance partitions (which can be regional) with leader regions that match the regions in the multi-region default instance partition.

- **Number of instance partitions** : Too many instance partitions can lead to overhead, while too few might not offer enough benefits. You can create a maximum of 20 instance partitions per instance. For more information about geo-partitioning limits, see [Geo-partitioning limits](https://docs.cloud.google.com/spanner/quotas#geo-partitioning_limits) .

## Limitations

The following limitations apply during the [Preview](https://cloud.google.com/products#product-launch-stages) release and are subject to change or removal upon the GA release or after:

- Your default instance partition must be a multi-region configuration.
- Your additional instance partition must be a multi-region configuration.
- You can't create an instance partition using a [dual-region configuration](https://docs.cloud.google.com/spanner/docs/geo-partitioning#limitations) .
- For each instance partition, the compute capacity must be at least one node (1000 processing units).
- For a given instance, you can't create more than one instance partition that uses the same base instance configuration. For example, within `test-instance` , you can't create two partitions, `partition-1` and `partition-2` that both use `us-central1` as the instance partition configuration.
- For every node in your instance partition, you can place a maximum of 100 million placement rows. You can view the number of placement rows that have been placed in each of your instance partitions on the Instance partitions page of the Google Cloud console.
- For every node in your destination instance partition, Spanner can [move](https://docs.cloud.google.com/spanner/docs/create-manage-data-placements#move-row) around 10 placement rows per second.
- You can't create [incremental backups](https://docs.cloud.google.com/spanner/docs/backup#incremental-backups) or [copy the backup](https://docs.cloud.google.com/spanner/docs/backup#how-backup-copy-works) .
- You can't enable [asymmetric read-only autoscaling](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#asymmetric-read-only-autoscaling) on non-default instance partitions.
- You can't move the additional partition to a different instance configuration.
- You can't move an instance that has databases using additional placements pointing to the default instance partition.
- If you use an `INSERT` or `DELETE` DML statement for a placement table, that statement must be the only statement in the [transaction](https://docs.cloud.google.com/spanner/docs/transactions) .
- The read-write transaction mode lets you reference only the primary keys of a placement table in the `WHERE` clause. If you need to reference a non-primary key column of a placement table in the `WHERE` clause, then you can use one of the following alternatives:
  - If you only need read-only access, switch to the read-only transaction mode.
  - If you need to make updates, either use the partitioned DML transaction mode, or find the primary keys in a read-only transaction query, and then, in a separate read-write transaction, reference the returned primary keys in the `WHERE` clause.
- You can't use [named schemas](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#named-schemas) .
- You can't create instance partitions in [free trial instances](https://docs.cloud.google.com/spanner/docs/free-trial-instance) or granular-sized instances smaller than one node (1000 processing units).
- You can't alter a placement. Instead, you can [create a new placement](https://docs.cloud.google.com/spanner/docs/create-manage-data-placements#create-placement) , use [partitioned DML](https://docs.cloud.google.com/spanner/docs/dml-partitioned) to update the placement of your data to a new placement, and then [drop the original placement](https://docs.cloud.google.com/spanner/docs/create-manage-data-placements#drop-placement) .

## Access control with IAM

You need to have the `spanner.instancePartitions.create` , `spanner.instancePartitions.update` , and `spanner.instancePartitions.delete` permissions to create and manage instance partitions. If you only need to view the instance partitions, you need to have the `spanner.instancePartitions.list` or `spanner.instancePartitions.get` permission. For more information, see [Identity and Access Management (IAM) overview](https://docs.cloud.google.com/spanner/docs/iam) .

For information on how to grant Spanner IAM permissions, see [Apply IAM permissions](https://docs.cloud.google.com/spanner/docs/grant-permissions) .

## Monitoring

Spanner provides several metrics to help you monitor your instance partitions. After you've created an additional instance partition, you see an additional drop-down filter for *Instance partitions* on the System insights page in the Google Cloud console. The default selection is to show metrics for *All* instance partitions. You can use the drop-down to filter the metrics for a specific instance partition.

To learn more about monitoring your Spanner resources, see [Monitor instances with Cloud Monitoring](https://docs.cloud.google.com/spanner/docs/monitoring-cloud) .

## Backups

You can create [full backups](https://docs.cloud.google.com/spanner/docs/backup) for databases that use geo-partitioning. You can't delete the instance partition if you use it in a backup. For more information, see [Create backups](https://docs.cloud.google.com/spanner/docs/backup/create-backups) .

Usage notes:

- Spanner stores backups of geo-partitioned data in the same placement location as the original data at the backup's version time.

- To [restore backups](https://docs.cloud.google.com/spanner/docs/backup/restore-backup-overview) that contain geo-partitioned data, your destination instance must meet the following conditions:

  - The destination instance must use the same instance partition names as the original backup.
  - Each instance partition in the destination instance must use the same instance configuration as the original backup.

## Managed autoscaler

You can enable the [managed autoscaler](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) on instance partitions. When you enable the managed autoscaler, Spanner automatically adjusts the size of your instance partition for you. The managed autoscaler reacts to changes in your instance partition's workload or storage needs as your load increases or decreases. The managed autoscaler either scales up, adding compute capacity to the instance partition, or it scales down, removing compute capacity from the instance partition.

## Pricing

There is no additional charge for using geo-partitioning. You are charged the standard Spanner pricing for the amount of compute capacity that your instance uses and the amount of storage that your database uses.

For more information, see [Spanner pricing](https://docs.cloud.google.com/spanner/pricing) .

## What's next

- Learn how to [Create and manage instance partitions](https://docs.cloud.google.com/spanner/docs/create-manage-partitions) .
- Learn how to [Create and manage data placements](https://docs.cloud.google.com/spanner/docs/create-manage-data-placements) .
