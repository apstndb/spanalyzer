---
name: documents/docs.cloud.google.com/spanner/docs/move-instance
uri: https://docs.cloud.google.com/spanner/docs/move-instance
title: Move an instance
description: Move your Spanner instance to a different instance configuration (e.g., from regional to multi-region) with minimal downtime.
data_source: docs.cloud.google.com
---

This page describes moving an instance in Spanner.

You can move your Spanner instance from any instance configuration to any other [instance configuration](https://docs.cloud.google.com/spanner/docs/instance-configurations) , including between regional, dual-region, and multi-region configurations. Moving your instance doesn't cause downtime, and Spanner continues to provide the usual [transaction guarantees](https://docs.cloud.google.com/spanner/docs/transactions#rw_transaction_properties) , including strong consistency, during the move.

You can also move your instance from its source instance configuration to a custom instance configuration (for example, a `nam3` base configuration with a `us-west2` read-only replica). Because you can't update the topology of existing instance configurations, you need to create a new custom instance configuration with the topology you want first. After creating the new custom instance configuration, you can move your instance from the source instance configuration to the new custom instance configuration.

## Why move your Spanner instance?

Benefits of moving your instance include:

- **Increase availability** : Obtain 99.999% availability with zero downtime after performing a regional to dual-region or multi-region move.
- **Reduce latency** : Reduce latency and increase geographic coverage with additional read-only replicas through a regional to dual-region or multi-region or multi-region to multi-region move.
- **Reduce cost** : Reduce hourly costs by moving from a dual-region or multi-region configuration to a regional configuration.
- **Colocate database** : Colocate the Spanner database with the client application by moving the instance to a more optimized location.

## Pricing

When moving an instance, both the source and destination instance configurations are subject to hourly [compute and storage charges](https://cloud.google.com/spanner/pricing) . Once the move is complete, you are billed for the instance storage at the destination configuration.

If you're moving your instance to a new regional, dual-region, or multi-region instance configuration, you might be subject to outbound data transfer charges. For more information, see [Spanner Pricing](https://cloud.google.com/spanner/pricing) .

## Limitations

- To move your instance, it must have a minimum of [1 node (1000 processing units)](https://docs.cloud.google.com/spanner/docs/compute-capacity) .
- You can't move your instance across projects or Google Cloud accounts.
- You can't move an instance that is using the Standard edition directly from a regional instance configuration to a dual-region or multi-region instance configuration. You must [upgrade the edition](https://docs.cloud.google.com/spanner/docs/create-manage-instances#update-edition) of your instance to the Enterprise Plus edition first, and then move the instance.
- If you have active requests using a [regional service endpoint](https://docs.cloud.google.com/spanner/docs/endpoints) on any of the instance resources, the instance move impacts all the requests that are using the regional endpoint because regional enforcement blocks access to cross region instances. Requests that use a global endpoint are unaffected.
- Spanner [backups](https://docs.cloud.google.com/spanner/docs/backup) and [backup schedules](https://docs.cloud.google.com/spanner/docs/backup/create-manage-backup-schedules) are specific to an instance configuration and aren't included when moving an instance. You must delete all existing backups and backup schedules on the instance before you start the move. For more information, see [Backups and backup schedules](https://docs.cloud.google.com/spanner/docs/move-instance#move-backups) .
- The following APIs are disabled during an instance move:
  - `InstanceAdmin.DeleteInstance`
  - `InstanceAdmin.UpdateInstance`
  - `InstanceAdmin.CreateInstancePartition`
  - `InstanceAdmin.UpdateInstancePartition`
  - `InstanceAdmin.DeleteInstancePartition`
  - `DatabaseAdmin.CreateDatabase`
  - `DatabaseAdmin.UpdateDatabaseDdl` (Disabled if `default_leader` is specified in the request.)
  - `DatabaseAdmin.RestoreDatabase`
  - `DatabaseAdmin.CreateBackup`
  - `DatabaseAdmin.CreateBackupSchedule`
  - `DatabaseAdmin.CopyBackup`
- If a database has a [modified default leader](https://docs.cloud.google.com/spanner/docs/instance-configurations#configure-leader-region) , the selection is preserved if it names a read-write region in the destination instance configuration, and that configuration is multi-region. If the destination configuration is regional, or doesn't include the named read-write region, the default leader selection is cleared.
- Moving an instance changes the instance configuration attribute of your instance. If you manage your Spanner resources through automation, make sure to prepare and address any inconsistencies that might arise.
  - For example, if you use [Terraform](https://docs.cloud.google.com/spanner/docs/use-terraform) to manage your Spanner instances and databases, and you enable `terraform apply --auto-approve` to keep your resources in sync, all instances and child resources are deleted when we move the instance. Update the configuration accordingly to avoid deletion and data loss. See [Terraform Apply Options](https://www.terraform.io/cli/commands/apply#apply-options) for more information about the `apply` command.
- While the instance is being moved, the Spanner monitoring metrics and charts might show data in both the source and destination instance configurations, or it might only reflect performance in one instance configuration.
- If you've configured the open source Autoscaler tool, then you don't need to disable it. It fails because `InstanceAdmin.UpdateInstance` (used for node and processing unit changes) is disabled.
- You can't move an instance if the [Spanner managed autoscaler](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) feature is enabled on it. Before you move the instance, switch the instance from managed autoscaling to [manual allocation](https://docs.cloud.google.com/spanner/docs/create-manage-instances#change-compute-capacity) with enough nodes or processing units to support your peak CPU usage (for example, by running `gcloud spanner instances update `` INSTANCE_ID `` --nodes= `` PEAK_NODES` ). After the move completes, you can re-enable the managed autoscaler.
- You can't move a [Spanner free trial instance](https://docs.cloud.google.com/spanner/docs/free-trial-instance) . You can move the instance after [upgrading to a paid instance](https://docs.cloud.google.com/spanner/docs/free-trial-instance#upgrade) .
- If the instance uses [geo-partitioning](https://docs.cloud.google.com/spanner/docs/geo-partitioning) , then moving the instance only moves the default instance partition, not any additional instance partitions.

## Performance considerations

When an instance is being moved, it experiences higher read-write latencies and a higher transaction abort rate. The CPU utilization during the move might go up to 100% because the instance move is performed using spare CPU that you have provisioned and yields to production traffic. However, moving an instance doesn't cause any downtime. The time it takes to move an instance depends on various factors including the size of the databases, the number of nodes, and the kind of move (for example, regional to multi-region).

For example, for a move to complete within 24 hours, your instance typically needs to meet these requirements before and during the move:

- **High-priority CPU utilization** : Less than 40% in the source instance configuration. For more information, see [CPU utilization](https://docs.cloud.google.com/spanner/docs/cpu-utilization) .
- **Storage per node** : Less than 1 Tebibyte (TiB) per node.

After moving an instance, the performance of the instance varies depending on the details of the instance configuration. For example, [dual-region](https://docs.cloud.google.com/spanner/docs/instance-configurations#dual-region-performance) and [multi-region configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations#multi-region-performance) generally have higher write latency and lower read latency than [regional configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations#regional-performance) .

## Backups and backup schedules

When you move an instance, the backups and [backup schedules](https://docs.cloud.google.com/spanner/docs/backup/create-manage-backup-schedules) in the instance's original configuration aren't moved to the new destination configuration automatically. The instance move fails to start if any backups or backup schedules (including [default backup schedules](https://docs.cloud.google.com/spanner/docs/backup#default-backup-schedules) ) exist on the instance when you start the move. Copy your backups and consider your [data recovery plan](https://docs.cloud.google.com/spanner/docs/backup#recommended) before moving your instance.

If there are backups in the instance's original configuration that you need to keep, we recommend that you [copy your backups](https://docs.cloud.google.com/spanner/docs/backup/copy-backup) to two small (100 PU) temporary instances, `placeholder-source` and `placeholder-dest` :

- **`placeholder-source`** : an instance with the same instance configuration as the moving instance's original configuration. This lets you restore your backups to the original configuration if you need to cancel the move.

- **`placeholder-dest`** : an instance with the same instance configuration as the destination instance configuration. This ensures that you have a backup readily available in the new configuration immediately after the move completes.

The restore feature does not support cross-configuration restores, so these placeholder instances are essential for a quick rollback or recovery in the new configuration if needed, providing a safety net in case of any issues with the moved instance.

After you copy your backups to `placeholder-source` and `placeholder-dest` , you must delete all existing backups and backup schedules on the moving instance before you can start the move (see [Copy backups to placeholder instances and delete existing backups and backup schedules](https://docs.cloud.google.com/spanner/docs/move-instance#copy-and-delete-backups) for the exact commands). Then, after the instance move is complete, you already have a copy of the backup in the destination configuration, and you can recreate your backup schedules and [create a new backup](https://docs.cloud.google.com/spanner/docs/backup/create-backup) .

By following this approach, you maintain business continuity and minimize potential downtime or data loss during the instance move process.

## How to move an instance

You can move an instance using the Google Cloud console or the gcloud CLI. Follow these four phases in order to prepare, run, monitor, and finalize your instance move:

- [Phase 1: Prepare your instance for the move](https://docs.cloud.google.com/spanner/docs/move-instance#move-prerequisites)
- [Phase 2: Start the instance move](https://docs.cloud.google.com/spanner/docs/move-instance#move-instance)
- [Phase 3: Monitor the instance move](https://docs.cloud.google.com/spanner/docs/move-instance#monitor-progress)
- [Phase 4: Complete post-move tasks](https://docs.cloud.google.com/spanner/docs/move-instance#post-move-tasks)

### Phase 1: Prepare your instance for the move

Before moving your instance, read the [Limitations](https://docs.cloud.google.com/spanner/docs/move-instance#move-limitations) and [Performance considerations](https://docs.cloud.google.com/spanner/docs/move-instance#move-performance) sections, and then complete each step in this pre-move checklist:

1.  **Verify permissions, edition, and destination quota** :

    - Confirm that the operator running the move has the `spanner.instances.update` [IAM permission](https://docs.cloud.google.com/spanner/docs/grant-permissions) on the moving instance.
    - Verify that your project has sufficient [node quota](https://docs.cloud.google.com/spanner/quotas#node_limits) in the destination instance configuration to support peak capacity. For more information, see [Spanner Quotas & Limits](https://docs.cloud.google.com/spanner/quotas) .
    - If you are moving a Standard edition instance from a regional configuration to a dual-region or multi-region configuration, [upgrade the edition](https://docs.cloud.google.com/spanner/docs/create-manage-instances#update-edition) to Enterprise Plus edition first.

2.  **Validate workload health, plan client placement, and test in a non-production environment** :

    - Move a non-production instance (such as test or staging) that mirrors your production instance first to estimate the move duration and observe the performance effect on your workload.
    - Check that there are no hotspots in your databases using [Key Visualizer](https://docs.cloud.google.com/spanner/docs/key-visualizer) . Resolving hotspots before the move helps prevent elevated latency and transaction aborts while data is being copied.
    - If your destination configuration has a different [default leader region](https://docs.cloud.google.com/spanner/docs/instance-configurations#multi-region-best-practices) than the source, your write latency may increase after the move. To prevent this, plan to deploy or route your write-heavy client applications in or near the destination leader region.

3.  **Switch from managed autoscaling to manual allocation and size for the move** :

    - To help the move complete within 24 hours and minimize latency impact, provision enough compute capacity so that peak high-priority [CPU utilization](https://docs.cloud.google.com/spanner/docs/cpu-utilization) is less than 40% and storage per node is less than 1 TiB during the move. Check the **CPU utilization** and **Total storage** charts in [System insights](https://docs.cloud.google.com/spanner/docs/monitoring-console#charts-metrics) to determine the number of nodes needed for your peak workload. Because Spanner uses spare CPU to move data, increasing compute capacity increases the speed of the instance move. You can confirm the ideal compute capacity for your workload by testing the move in a pre-production or staging environment.

    - If the [managed autoscaler](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) is enabled on the instance, you must switch the instance to [manual allocation](https://docs.cloud.google.com/spanner/docs/create-manage-instances#change-compute-capacity) before starting the move, using the number of nodes you determined is needed to keep peak high-priority CPU utilization under 40%:

      ```
      gcloud spanner instances update INSTANCE_ID \
        --nodes=PEAK_NODES
      ```

4.  **Copy backups to placeholder instances and delete existing backups and backup schedules** :

    - Create the temporary `placeholder-source` and `placeholder-dest` instances discussed in the [Backups and backup schedules](https://docs.cloud.google.com/spanner/docs/move-instance#move-backups) section, and copy your critical backups to them using the [`gcloud spanner backups copy`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/backups/copy) command:

      ```
      gcloud spanner backups copy \
        --source-instance=INSTANCE_ID \
        --source-backup=SOURCE_BACKUP_ID \
        --destination-instance=PLACEHOLDER_INSTANCE_ID \
        --destination-backup=DESTINATION_BACKUP_ID \
        --expiration-date=EXPIRATION_DATE
      ```

    - Verify that all backup copy operations on the placeholder instances have completed before deleting backups on the moving instance:

      ```
      gcloud spanner operations list \
        --instance=PLACEHOLDER_INSTANCE_ID \
        --type=BACKUP
      ```

    - [Delete all existing backups](https://docs.cloud.google.com/spanner/docs/backup/manage-backups#delete_a_backup) and [delete all backup schedules](https://docs.cloud.google.com/spanner/docs/backup/create-manage-backup-schedules#delete-backup-schedule) on every database in the moving instance:

      ```
      gcloud spanner backups delete BACKUP_ID \
        --instance=INSTANCE_ID
      ```

      ```
      gcloud spanner backup-schedules delete SCHEDULE_ID \
        --instance=INSTANCE_ID \
        --database=DATABASE_ID
      ```

5.  **Export Data Catalog tags, pause automation, and freeze instance and schema changes** :

    - Moving an instance deletes any instance tags [that you created in Data Catalog](https://docs.cloud.google.com/spanner/docs/dc-integration) . Export your tags before the move so you can import them afterward. For more information, see [Export and import tags](https://docs.cloud.google.com/spanner/docs/dc-integration#import-export-tags) .
    - Check your CI/CD workflows, scheduled jobs, or GitOps repositories for automated infrastructure pipelines (such as scheduled `terraform apply --auto-approve` runs) that manage your Spanner resources, and pause them so they don't conflict with the in-progress move.
    - Communicate to your organization that instance and schema changes (including node count updates, DDL updates, database creation or deletion, and backup operations) should be paused until the move completes.

6.  **CMEK-enabled databases only: Verify destination regions and Cloud Key Management Service (Cloud KMS) keys** :

    - Inspect the destination instance configuration to identify all constituent read-write, read-only, and witness regions:

      ```
      gcloud spanner instance-configs describe TARGET_CONFIG
      ```

    - Ensure that you have provisioned Cloud KMS keys in every constituent region of the destination configuration and granted the `roles/cloudkms.cryptoKeyEncrypterDecrypter` Identity and Access Management (IAM) role on each target key to the Spanner service agent.

If your instance meets these pre-move guidelines, the move typically completes within 24 hours. Depending on database size and application workload, completion time might be longer or shorter.

### Phase 2: Start the instance move

### Google Cloud console

> **Note:** You can't move instances that contain [CMEK](https://docs.cloud.google.com/spanner/docs/cmek) -enabled databases using the Google Cloud console. Move the instance using the gcloud CLI.

1.  Go to the **Instances** page in the Google Cloud console.

2.  Select the instance that you want to move.

3.  On the Instance overview page, next to **Configuration** , click edit **Move instance to a new configuration** .

4.  On the **Move database to new configuration** pane, select the new instance configuration for your instance.

5.  Click **Save** .

### gcloud CLI

Use the [`gcloud spanner instances move`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/move) command to move the instance.

```
gcloud spanner instances move INSTANCE_ID \
--target-config=TARGET_CONFIG
```

Replace the following:

- ` INSTANCE_ID ` : the permanent identifier for the instance that you want to move.
- ` TARGET_CONFIG ` : a permanent identifier of the instance configuration where you want to move your instance. The new geographic location of your instance. This could be a regional, dual-region, or multi-region instance configuration (for example, `nam3` , `regional-us-central1` , or `custom-nam3-us-west2` ).

For example, to move your instance `test-instance` from its current instance configuration to `nam3` , run the following:

```
  gcloud spanner instances move test-instance --target-config=nam3
```

Optional: If you want to add a read-only replica in the `us-west2` region to the base instance configuration in `nam3` , do the following:

1.  Clone the base configuration and add the read-only replica to the new custom instance configuration `custom-nam3-us-west2` :

    ```
    gcloud spanner instance-configs create custom-nam3-us-west2 \
    --clone-config=nam3 --add-replicas=location=us-west2, type=READ_ONLY
    ```

2.  Move your instance `test-instance` from its current instance configuration to this new `custom-nam3-us-west2` instance configuration:

    ```
    gcloud spanner instances move test-instance --target-config=custom-nam3-us-west2
    ```

### Optional: Move an instance with CMEK-enabled databases

Before moving an instance with [CMEK](https://docs.cloud.google.com/spanner/docs/cmek) -enabled databases, review the following usage notes:

- If you have multiple CMEK-enabled databases in the instance that you want to move, you must specify `--target-database-move-configs` for each of them. You can use the same keys for every database, but you must specify the keys for each CMEK-enabled database.
- Your keys must cover all the regions in the destination instance configuration, and the Spanner service agent must have the `roles/cloudkms.cryptoKeyEncrypterDecrypter` IAM role on each target key. For example, if your destination instance configuration is in `nam3` , then you must set keys in `us-east4` , `us-east1` , and `us-central1` .
- You can't set Cloud KMS keys for databases that aren't CMEK-enabled while moving the instance.
- Don't disable, destroy, rotate out, or revoke IAM permissions on CMEK keys in either the source or destination instance configuration while moving the instance. Doing so stalls the move and can lock out access to the database.

Use the [`gcloud spanner instances move`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/move) command to move an instance with CMEK-enabled databases. You must include the `--target-database-move-configs` flag and Cloud KMS key values in the command or configure a JSON or YAML file with the necessary Cloud KMS keys:

```
gcloud spanner instances move INSTANCE_ID \
  --target-config=TARGET_CONFIG \
  --target-database-move-configs=^:^database-id=DATABASE_ID_1:kms-key-names=KMS_KEY_1[, KMS_KEY_2 ... ] \
  [--target-database-move-configs=^:^database-id=DATABASE_ID_2:kms-key-names=KMS_KEY_1 ... ]
```

or

```
gcloud spanner instances move INSTANCE_ID \
  --target-config=TARGET_CONFIG \
  --target-database-move-configs=CONFIG_FILE_PATH
```

Configure the ` CONFIG_FILE_PATH ` file with your database IDs and Cloud KMS keys. The following configuration file example contains the Cloud KMS keys for two databases, `database-1` and `database-2` , with the same keys in `us-east4` , `us-east1` , and `us-central1` to cover all the regions in `nam3` :

```
[
  {
    "database-id": "database-1",
    "kms-key-names": "projects/PROJECT_ID/locations/us-east4/keyRings/KEY_RING/cryptoKeys/KEY_NAME,projects/PROJECT_ID/locations/us-east1/keyRings/KEY_RING/cryptoKeys/KEY_NAME,projects/PROJECT_ID/locations/us-central1/keyRings/KEY_RING/cryptoKeys/KEY_NAME"
  },
  {
    "database-id": "database-2",
    "kms-key-names": "projects/PROJECT_ID/locations/us-east4/keyRings/KEY_RING/cryptoKeys/KEY_NAME,projects/PROJECT_ID/locations/us-east1/keyRings/KEY_RING/cryptoKeys/KEY_NAME,projects/PROJECT_ID/locations/us-central1/keyRings/KEY_RING/cryptoKeys/KEY_NAME"
  }
]
```

### Phase 3: Monitor the instance move

You can use `gcloud spanner operations describe` and a custom Cloud Monitoring dashboard to monitor the progress of an instance move or cancellation.

#### View move and cancellation operation progress

To track the progress of an instance move or instance move cancellation operation, use the [`gcloud spanner operations describe`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/operations/describe) command. This command requires the operation ID of the in-progress instance move operation.

1.  Get the operation ID for your instance move operation by running:

    ```
    gcloud spanner operations list --instance="INSTANCE_ID"
    ```

    Replace the following:

    - ` INSTANCE_ID ` : the permanent identifier for the instance that you want to move.

    The output shows a list of long-running operations, including the instance move operation.

2.  Run the `gcloud spanner operations describe` command to view progress percentage and status:

    ```
    gcloud spanner operations describe OPERATION_ID --instance=INSTANCE_ID
    ```

    Replace the following:

    - ` OPERATION_ID ` : the operation ID of the instance move operation that you want to check.
    - ` INSTANCE_ID ` : the instance ID for the instance you want to check.

#### Monitor an instance move operation

You can create a custom Cloud Monitoring dashboard to display and monitor metrics during the instance move, a long-running operation with potential service implications.

The **Total storage** and **Total database storage by databases** graphs in the dashboard are helpful to monitor the progress of the move. You can see the storage in the source configuration gradually drop while the storage in the destination configuration increases.

### Google Cloud console

1.  Download the [`move-instance-dashboard.json`](https://docs.cloud.google.com/static/spanner/docs/move-instance-dashboard.json) file. This file has the information needed to populate a custom dashboard in Monitoring.

2.  In the Google Cloud console, go to the dashboard **Dashboards** page:

    If you use the search bar to find this page, then select the result whose subheading is **Monitoring** .

3.  In the **Dashboards Overview** page, click **Create dashboard** .

4.  In the dashboard toolbar, click the **Dashboard settings** drop-down. Then select **JSON** , followed by **JSON Editor** .

5.  In the **JSON Editor** pane, copy the contents of the `move-instance-dashboard.json` file you downloaded and paste it in the editor.

6.  To apply your changes to the dashboard, click **Apply changes** . If you don't want to use this dashboard, navigate back to the Dashboards Overview page.

7.  After the dashboard is created, click **Add Filter** . Then select either `project_id` or `instance_id` to monitor the progress of your instance move.

### gcloud CLI

1.  Download the [`move-instance-dashboard.json`](https://docs.cloud.google.com/static/spanner/docs/move-instance-dashboard.json) file. This file has the information needed to populate a custom dashboard in Monitoring.

2.  To create a dashboard in a project, use the `gcloud monitoring dashboards create` command:

    ```
    gcloud monitoring dashboards create --config-from-file=move-instance-dashboard.json
    ```

    For more information, see the [`gcloud monitoring dashboards create`](https://docs.cloud.google.com/sdk/gcloud/reference/monitoring/dashboards/create) reference.

### Phase 4: Complete post-move tasks

After the long-running move operation completes ( `done: true` ), complete the following post-move checklist to restore your operational configuration:

1.  **Re-enable the managed autoscaler** : If you switched from managed autoscaling to manual allocation before the move, [re-enable the managed autoscaler](https://docs.cloud.google.com/spanner/docs/create-manage-instances#change-compute-capacity) with your minimum and maximum compute capacity bounds (for example, by running `gcloud spanner instances update `` INSTANCE_ID` with your autoscaling flags or in the Google Cloud console).
2.  **Recreate backup schedules and create a backup** : [Recreate your backup schedules](https://docs.cloud.google.com/spanner/docs/backup/create-manage-backup-schedules#create-backup-schedule) and [create a new backup](https://docs.cloud.google.com/spanner/docs/backup/create-backup) for your databases in the destination instance configuration.
3.  **Import Data Catalog tags** : If you exported Data Catalog tags before the move, [import them back](https://docs.cloud.google.com/spanner/docs/dc-integration#import-export-tags) to your moved instance.
4.  **Update automation configurations, client placement, and downstream workloads** :
    - If you manage your Spanner resources using automation tools such as [Terraform](https://docs.cloud.google.com/spanner/docs/use-terraform) , update your configuration files to reflect the new destination instance configuration before re-enabling automated deployments.
    - If your leader region changed after the move, deploy or route your client applications within or close to the new leader region (or [modify the database default leader region](https://docs.cloud.google.com/spanner/docs/modifying-leader-region) if using a dual-region or multi-region configuration) to minimize write latency, and verify connectivity and latency across downstream applications and pipelines.
5.  **Delete temporary placeholder instances** : After you verify your moved instance and create new backups in the destination configuration, delete the temporary `placeholder-source` and `placeholder-dest` instances if you no longer need the copied backups.

## How to cancel an instance move

You can only cancel an instance move that is still in progress. If you want to revert an already completed instance move, you must start a new move.

You can use [`gcloud spanner operations cancel`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/operations/cancel) to cancel instance move operations. The cancellation is not instantaneous and takes roughly the same amount of time as the time that has elapsed since the start of the move. This is because data has to be moved back to source instance configuration.

This command requires the operation ID of the in progress instance move operation.

1.  Get the operation ID by running:

    ```
    gcloud spanner operations list --type=INSTANCE --instance="INSTANCE_ID"
    --filter="done:False AND metadata.@type:MoveInstanceMetadata"
    ```

    Replace the following:

    - ` INSTANCE_ID ` : the permanent identifier for the instance that you want to move.

    The output shows a list of in progress instance move operations.

2.  Run the `gcloud spanner operations cancel` command to cancel the instance move:

    ```
    gcloud spanner operations cancel OPERATION_ID
    ```

    Replace the following:

    - ` OPERATION_ID ` : the operation ID of the instance move operation that you want to cancel.

## What's next

- Learn more about Spanner [Regional, dual-region and multi-region configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations) .
- Learn more about [Google Cloud regions and zones](https://docs.cloud.google.com/docs/geography-and-regions) .
