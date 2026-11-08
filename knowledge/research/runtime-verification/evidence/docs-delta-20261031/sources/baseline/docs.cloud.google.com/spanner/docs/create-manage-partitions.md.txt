---
name: documents/docs.cloud.google.com/spanner/docs/create-manage-partitions
uri: https://docs.cloud.google.com/spanner/docs/create-manage-partitions
title: Create and manage instance partitions
description: Manage Spanner instance partitions to store data and process queries in specific regions.
data_source: docs.cloud.google.com
---

> **Preview — [Geo-partitioning](https://docs.cloud.google.com/spanner/docs/geo-partitioning)**
>
> This feature is subject to the "Pre-GA Offerings Terms" in the General Service Terms section of the [Service Specific Terms](https://docs.cloud.google.com/terms/service-terms#1) . Pre-GA features are available "as is" and might have limited support. For more information, see the [launch stage descriptions](https://cloud.google.com/products/#product-launch-stages) .

> **Note:** This feature is available with the Spanner Enterprise Plus edition. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

This page describes how to create and manage Spanner [instance partitions](https://docs.cloud.google.com/spanner/docs/geo-partitioning) .

## Create an instance partition

### Console

1.  In the Google Cloud console, open the **Spanner** page.

2.  Select the instance in which you want to add instance partitions.

3.  In the navigation menu, select **Instance partitions** .

4.  Click **Create instance partition** .

5.  Enter an **Instance partition ID** to permanently identify your instance partition. The instance partition ID must also be unique within your instance. You can't change the instance partition ID later.

6.  In the **Choose a configuration** section, select **Regional** or **Multi-region** . Alternatively, if you want to compare the specifications between the regions, then click **Compare region configurations** .

7.  Select a configuration from the drop-down menu.

8.  In the **Configure compute capacity** section, under **Select unit** , click one of the following:

    - **Nodes** for large instances. A node is 1000 processing units.
    - **Processing units** for small instance partitions.

    For more information, see [Compute capacity, nodes, and processing units](https://docs.cloud.google.com/spanner/docs/compute-capacity) .

9.  Under **Choose a scaling mode** , click one of the following:

    - **Manual allocation** if you want to manually set compute capacity for fixed compute resources and costs.

      - **Quantity** indicates the number of processing units or nodes to use for this instance.

    - **Autoscaling** to let Spanner automatically add and remove compute capacity. Managed autoscaler is available in the [Spanner Enterprise edition and Enterprise Plus edition](https://docs.cloud.google.com/spanner/docs/editions-overview) . For more information about the managed autoscaler, see [Managed autoscaler for Spanner](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) . Configure the following managed autoscaler options:

      - **Minimum** indicates the minimum limit to scale down to, depending on the measurement unit that you choose for **Compute capacity** . For more information, see [Determine the minimum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-minimum) .
      - **Maximum** indicates the maximum limit to scale up to, depending on the measurement unit that you choose for **Compute capacity** . For more information, see [Determine the maximum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-maximum) .
      - **High priority CPU utilization target** indicates the target percentage of CPU to use for high priority tasks. For more information, see [Determine the CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-high-priority-cpu) .
      - **Total CPU utilization target** indicates the target percentage of CPU to use for all low, medium, and high priority tasks. For more information, see [Determine the total CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-total-cpu) .
      - **Storage utilization target** indicates the target percentage of storage to use. For more information, see [Determine the Storage Utilization Target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-storage) .

10. Click **Create** to create the instance partition.

### gcloud

To create an instance partition, use [`gcloud spanner instance-partitions create`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instance-partitions/create) .

```
gcloud spanner instance-partitions create INSTANCE_PARTITION_ID \
  --config=INSTANCE_PARTITION_CONFIG \
  --description="INSTANCE_PARTITION_DESCRIPTION" \
  --instance=INSTANCE_ID \
  [--nodes=NODE_COUNT | --processing-units=PROCESSING_UNIT_COUNT]
```

Replace the following:

- ` INSTANCE_PARTITION_ID ` : the permanent instance partition identifier that is unique within your instance. You can't change the instance partition ID later.
- ` INSTANCE_PARTITION_CONFIG ` : the permanent identifier of your instance partition configuration, which defines the geographic location of the instance partition and affects where data is stored.
- ` INSTANCE_PARTITION_DESCRIPTION ` : the name to display for the instance partition in the Google Cloud console. The instance partition name must be unique within your instance.
- ` INSTANCE_ID ` : the permanent identifier for your Spanner instance where this instance partition resides.
- ` NODE_COUNT ` : the compute capacity of the instance partition, expressed as a number of nodes. One node equals 1000 processing units.
- ` PROCESSING_UNIT_COUNT ` : the compute capacity of the instance, expressed as a number of processing units. Your instance partition must have at least 1000 processing units. Enter quantities in multiples of 1000 (1000, 2000, 3000 and so on).

For example, to create an instance partition `europe-partition` in `eur3` with 5 nodes, run the following:

```
  gcloud spanner instance-partitions create europe-partition --config=eur3 \
    --description="europe-partition" --instance=test-instance --nodes=5
```

### Use managed autoscaling

You can use managed autoscaling with the [`gcloud spanner instance-partitions create`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instance-partitions/create) command. For more information, see [Managed autoscaler](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) .

Use the following command to create an instance partition with managed autoscaler:

```
  gcloud 
  spanner instance-partitions create INSTANCE_PARTITION_ID \
    --config=INSTANCE_PARTITION_CONFIG \
    --description="INSTANCE_PARTITION_DESCRIPTION" \
    --instance=INSTANCE_ID \
    --autoscaling-min-processing-units=MINIMUM_PROCESSING_UNITS \
    --autoscaling-max-processing-units=MAXIMUM_PROCESSING_UNITS \
    --autoscaling-high-priority-cpu-target=HIGH_PRIORITY_CPU_PERCENTAGE \
    --autoscaling-total-cpu-target=TOTAL_CPU_PERCENTAGE \
    --autoscaling-storage-target=STORAGE_PERCENTAGE
```

or

```
  gcloud spanner instance-partitions create INSTANCE_PARTITION_ID \
    --config=INSTANCE_PARTITION_CONFIG \
    --description="INSTANCE_PARTITION_DESCRIPTION" \
    --instance=INSTANCE_ID \
    --autoscaling-min-nodes=MINIMUM_NODES \
    --autoscaling-max-nodes=MAXIMUM_NODES \
    --autoscaling-high-priority-cpu-target=HIGH_PRIORITY_CPU_PERCENTAGE \
    --autoscaling-total-cpu-target=TOTAL_CPU_PERCENTAGE
    --autoscaling-storage-target=STORAGE_PERCENTAGE
```

Replace the following:

- ` INSTANCE_PARTITION_ID ` : the permanent instance partition identifier that is unique within your instance. You can't change the instance partition ID later.
- ` INSTANCE_PARTITION_CONFIG ` : the permanent identifier of your instance partition configuration, which defines the geographic location of the instance partition and affects where data is stored.
- ` INSTANCE_PARTITION_DESCRIPTION ` : the name to display for the instance partition in the Google Cloud console. The instance partition name must be unique within your instance.
- ` INSTANCE_ID ` : the permanent identifier for your Spanner instance where this instance partition resides.
- ` MINIMUM_PROCESSING_UNITS ` or ` MINIMUM_NODES ` : the minimum number of processing units or nodes when scaling down. For more information, see [Determine the minimum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-minimum) .
- ` MAXIMUM_PROCESSING_UNITS ` or ` MAXIMUM_NODES ` : the maximum number of processing units or nodes when scaling up. For more information, see [Determine the maximum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-maximum) .
- ` HIGH_PRIORITY_CPU_PERCENTAGE ` : the target percentage of high priority CPU to use, based on the [priority of the task](https://docs.cloud.google.com/spanner/docs/cpu-utilization#task-priority) . The CPU percentage can range from 10 to 90%. For more information, see [Determine the high priority CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-high-priority-cpu) .
- ` TOTAL_CPU_PERCENTAGE ` : the target percentage of total priority CPU to use. The total CPU target has to be greater than the high priority CPU target. The CPU percentage can range from 10 to 90%. For more information, see [Determine the total CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-total-cpu) .
- ` STORAGE_PERCENTAGE ` : the target percentage of storage to use, from 10 to 99%. For more information, see [Determine the storage utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-storage) .

### Client libraries

### C++

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

```cpp
void CreateInstancePartition(
    google::cloud::spanner_admin::InstanceAdminClient client,
    std::string const& project_id, std::string const& instance_id,
    std::string const& instance_partition_id) {
  auto project = google::cloud::Project(project_id);
  auto in = google::cloud::spanner::Instance(project_id, instance_id);
  auto config = project.FullName() + "/instanceConfigs/nam3";

  google::spanner::admin::instance::v1::CreateInstancePartitionRequest request;
  request.set_parent(in.FullName());
  request.set_instance_partition_id(instance_partition_id);
  request.mutable_instance_partition()->set_display_name(
      "Test instance partition");
  request.mutable_instance_partition()->set_node_count(1);
  request.mutable_instance_partition()->set_config(config);

  auto instance_partition = client.CreateInstancePartition(request).get();
  if (!instance_partition) throw std::move(instance_partition).status();
  std::cout << "Created instance partition [" << instance_partition_id << "]:\n"
            << instance_partition->DebugString();
}
```

### C#

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

```csharp
using Google.Cloud.Spanner.Admin.Instance.V1;
using Google.Cloud.Spanner.Common.V1;
using Google.LongRunning;
using System;

public class CreateInstancePartitionSample
{
    public InstancePartition CreateInstancePartition(string projectId, string instanceId, string instancePartitionId)
    {
        // Create the InstanceAdminClient instance.
        InstanceAdminClient instanceAdminClient = InstanceAdminClient.Create();

        // Initialize request parameters.
        InstancePartition partition = new InstancePartition
        {
            DisplayName = "This is a display name.",
            NodeCount = 1,
            ConfigAsInstanceConfigName = InstanceConfigName.FromProjectInstanceConfig(projectId, "nam3"),
        };
        InstanceName instanceName = InstanceName.FromProjectInstance(projectId, instanceId);

        // Make the CreateInstancePartition request.
        Operation<InstancePartition, CreateInstancePartitionMetadata> response = instanceAdminClient.CreateInstancePartition(instanceName, partition, instancePartitionId);

        Console.WriteLine("Waiting for the operation to finish.");

        // Poll until the returned long-running operation is complete.
        Operation<InstancePartition, CreateInstancePartitionMetadata> completedResponse = response.PollUntilCompleted();

        if (completedResponse.IsFaulted)
        {
            Console.WriteLine($"Error while creating instance partition: {completedResponse.Exception}");
            throw completedResponse.Exception;
        }

        Console.WriteLine($"Instance created successfully.");

        return completedResponse.Result;
    }
}
```

### Go

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

```go
import (
    "context"
    "fmt"
    "io"

    instance "cloud.google.com/go/spanner/admin/instance/apiv1"
    "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
)

// Example of creating an instance partition with Go.
// projectID is the ID of the project that the new instance partition will be in.
// instanceID is the ID of the instance that the new instance partition will be in.
// instancePartitionID is the ID of the new instance partition to be created.
func createInstancePartition(w io.Writer, projectID, instanceID, instancePartitionID string) error {
    // projectID := "my-project-id"
    // instanceID := "my-instance"
    // instancePartitionID := "my-instance-partition"
    ctx := context.Background()
    instanceAdmin, err := instance.NewInstanceAdminClient(ctx)
    if err != nil {
        return err
    }
    defer instanceAdmin.Close()

    op, err := instanceAdmin.CreateInstancePartition(ctx, &instancepb.CreateInstancePartitionRequest{
        Parent:              fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID),
        InstancePartitionId: instancePartitionID,
        InstancePartition: &instancepb.InstancePartition{
            Config:          fmt.Sprintf("projects/%s/instanceConfigs/%s", projectID, "nam3"),
            DisplayName:     "my-instance-partition",
            ComputeCapacity: &instancepb.InstancePartition_NodeCount{NodeCount: 1},
        },
    })
    if err != nil {
        return fmt.Errorf("could not create instance partition %s: %w", fmt.Sprintf("projects/%s/instances/%s/instancePartitions/%s", projectID, instanceID, instancePartitionID), err)
    }
    // Wait for the instance partition creation to finish.
    i, err := op.Wait(ctx)
    if err != nil {
        return fmt.Errorf("waiting for instance partition creation to finish failed: %w", err)
    }
    // The instance partition may not be ready to serve yet.
    if i.State != instancepb.InstancePartition_READY {
        fmt.Fprintf(w, "instance partition state is not READY yet. Got state %v\n", i.State)
    }
    fmt.Fprintf(w, "Created instance partition [%s]\n", instancePartitionID)
    return nil
}
```

### Java

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

```java
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.admin.instance.v1.InstanceAdminClient;
import com.google.spanner.admin.instance.v1.CreateInstancePartitionRequest;
import com.google.spanner.admin.instance.v1.InstanceConfigName;
import com.google.spanner.admin.instance.v1.InstanceName;
import com.google.spanner.admin.instance.v1.InstancePartition;
import java.util.concurrent.ExecutionException;

class CreateInstancePartitionSample {

  static void createInstancePartition() {
    // TODO(developer): Replace these variables before running the sample.
    String projectId = "my-project";
    String instanceId = "my-instance";
    String instancePartitionId = "my-instance-partition";
    createInstancePartition(projectId, instanceId, instancePartitionId);
  }

  static void createInstancePartition(
      String projectId, String instanceId, String instancePartitionId) {
    // Set instance partition configuration.
    int nodeCount = 1;
    String displayName = "Descriptive name";

    // Create an InstancePartition object that will be used to create the instance partition.
    InstancePartition instancePartition =
        InstancePartition.newBuilder()
            .setDisplayName(displayName)
            .setNodeCount(nodeCount)
            .setConfig(InstanceConfigName.of(projectId, "nam3").toString())
            .build();

    try (Spanner spanner =
            SpannerOptions.newBuilder().setProjectId(projectId).build().getService();
        InstanceAdminClient instanceAdminClient = spanner.createInstanceAdminClient()) {

      // Wait for the createInstancePartition operation to finish.
      InstancePartition createdInstancePartition =
          instanceAdminClient
              .createInstancePartitionAsync(
                  CreateInstancePartitionRequest.newBuilder()
                      .setParent(InstanceName.of(projectId, instanceId).toString())
                      .setInstancePartitionId(instancePartitionId)
                      .setInstancePartition(instancePartition)
                      .build())
              .get();
      System.out.printf(
          "Instance partition %s was successfully created%n", createdInstancePartition.getName());
    } catch (ExecutionException e) {
      System.out.printf(
          "Error: Creating instance partition %s failed with error message %s%n",
          instancePartition.getName(), e.getMessage());
    } catch (InterruptedException e) {
      System.out.println(
          "Error: Waiting for createInstancePartition operation to finish was interrupted");
    }
  }
}
```

### Node.js

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

```javascript
// Imports the Google Cloud client library
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const instancePartitionId = 'my-instance-partition';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

// Get the instance admin client
const instanceAdminClient = spanner.getInstanceAdminClient();

// Creates a new instance partition
try {
  console.log(
    `Creating instance partition ${instanceAdminClient.instancePartitionPath(
      projectId,
      instanceId,
      instancePartitionId,
    )}.`,
  );
  const [operation] = await instanceAdminClient.createInstancePartition({
    instancePartitionId: instancePartitionId,
    parent: instanceAdminClient.instancePath(projectId, instanceId),
    instancePartition: {
      config: instanceAdminClient.instanceConfigPath(projectId, 'nam3'),
      nodeCount: 1,
      displayName: 'Test instance partition',
    },
  });

  console.log(
    `Waiting for operation on ${instancePartitionId} to complete...`,
  );
  await operation.promise();

  console.log(`Created instance partition ${instancePartitionId}.`);
} catch (err) {
  console.error('ERROR:', err);
}
```

### PHP

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

```php
use Google\Cloud\Spanner\Admin\Instance\V1\Client\InstanceAdminClient;
use Google\Cloud\Spanner\Admin\Instance\V1\CreateInstancePartitionRequest;
use Google\Cloud\Spanner\Admin\Instance\V1\InstancePartition;

/**
 * Creates an instance partition.
 * Example:
 * ```
 * create_instance_partition($projectId, $instanceId, $instancePartitionId);
 * ```
 *
 * @param string $projectId The Google Cloud project ID.
 * @param string $instanceId The Spanner instance ID.
 * @param string $instancePartitionId The instance partition ID.
 */
function create_instance_partition(string $projectId, string $instanceId, string $instancePartitionId): void
{
    $instanceAdminClient = new InstanceAdminClient();

    $instanceName = $instanceAdminClient->instanceName($projectId, $instanceId);
    $instancePartitionName = $instanceAdminClient->instancePartitionName($projectId, $instanceId, $instancePartitionId);
    $configName = $instanceAdminClient->instanceConfigName($projectId, 'nam3');

    $instancePartition = (new InstancePartition())
        ->setConfig($configName)
        ->setDisplayName('Test instance partition.')
        ->setNodeCount(1);

    $operation = $instanceAdminClient->createInstancePartition(
        (new CreateInstancePartitionRequest())
        ->setParent($instanceName)
        ->setInstancePartitionId($instancePartitionId)
        ->setInstancePartition($instancePartition)
    );

    print('Waiting for operation to complete...' . PHP_EOL);
    $operation->pollUntilComplete();

    printf('Created instance partition %s' . PHP_EOL, $instancePartitionId);
}
```

### Python

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

```python
def create_instance_partition(instance_id, instance_partition_id):
    """Creates an instance partition."""
    from google.cloud.spanner_admin_instance_v1.types import spanner_instance_admin

    spanner_client = spanner.Client()
    instance_admin_api = spanner_client.instance_admin_api

    config_name = "{}/instanceConfigs/nam3".format(spanner_client.project_name)

    operation = spanner_client.instance_admin_api.create_instance_partition(
        parent=instance_admin_api.instance_path(spanner_client.project, instance_id),
        instance_partition_id=instance_partition_id,
        instance_partition=spanner_instance_admin.InstancePartition(
            config=config_name,
            display_name="Test instance partition",
            node_count=1,
        ),
    )

    print("Waiting for operation to complete...")
    operation.result(OPERATION_TIMEOUT_SECONDS)

    print("Created instance partition {}".format(instance_partition_id))
```

## Describe an instance partition

### gcloud

To describe an instance partition, use [`gcloud spanner instance-partitions describe`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instance-partitions/describe) .

```
gcloud spanner instance-partitions describe PARTITION_ID \
  --instance=INSTANCE_ID
```

Replace the following:

- ` INSTANCE_PARTITION_ID ` : the permanent identifier for the instance partition.
- ` INSTANCE_ID ` : the permanent identifier for the instance.

For example, to describe the instance partition `europe-partition` , run the following:

```
  gcloud spanner instance-partitions describe europe-partition
    --instance=test-instance
```

## List instance partitions

### Console

1.  In the Google Cloud console, open the **Spanner** page.

2.  Select an instance from the list.

3.  In the navigation menu, select **Instance partitions** .

    A list of instance partitions associated with that instance is shown.

### gcloud

To list your instance partitions, use [`gcloud spanner instance-partitions list`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instance-partitions/list) .

```
gcloud spanner instance-partitions list --instance=INSTANCE_ID
```

The gcloud CLI prints a list of your Spanner instance partitions, along with each instance partition's ID, display name, configuration, and compute capacity.

## Edit an instance partition

The following section explains how to change the compute capacity of your instance partition. You can't change the instance partition ID, name, or configuration.

### Change the compute capacity

You must provision enough [compute capacity](https://docs.cloud.google.com/spanner/docs/compute-capacity) to keep [CPU utilization](https://docs.cloud.google.com/spanner/docs/cpu-utilization#recommended-max) and [storage utilization](https://docs.cloud.google.com/spanner/docs/storage-utilization) below the recommended maximums. For more information, see the [quotas and limits](https://docs.cloud.google.com/spanner/quotas) for Spanner.

If you want to increase the compute capacity of an instance partition, your Google Cloud project must have sufficient quota to add the compute capacity. The time it takes for the increase request to complete depends on the size of the request. In most cases, requests complete within a few minutes. On rare occasions, a scale up might take up to an hour to complete.

### Console

1.  In the Google Cloud console, open the **Spanner** page.

2.  Select an instance from the list.

3.  In the navigation menu, select **Instance partitions** .

4.  In the list of instance partitions, under the **Actions** column, click **More Actions** and select **Edit** .

5.  Change the compute capacity by choosing a measurement unit (processing units or nodes), and then entering a quantity. When using processing units, enter quantities in multiples of 1000 (1000, 2000, 3000 and so on). Each node equals 1000 processing units.

    Your instance partition must have at least one node (1000 processing units).

6.  Click **Save** .

    If you see a dialog that says you have insufficient quota to add compute capacity , follow the instructions to request a higher quota.

### gcloud

To change the compute capacity of your instance partition, use [`gcloud spanner instance-partitions update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instance-partitions/update) . When using this command, specify the [compute capacity](https://docs.cloud.google.com/spanner/docs/compute-capacity) as a number of nodes or processing units.

```
gcloud spanner instance-partitions update INSTANCE_PARTITION_ID \
  --instance=INSTANCE_ID \
  [--nodes=NODE_COUNT | --processing-units=PROCESSING_UNIT_COUNT]
  [--async]
```

Replace the following:

- ` INSTANCE_PARTITION_ID ` : the permanent identifier for the instance partition.
- ` INSTANCE_ID ` : the permanent identifier for the instance.
- ` NODE_COUNT ` : the new compute capacity of the instance partition, expressed as a number of nodes. One node equals 1000 processing units.
- ` PROCESSING_UNIT_COUNT ` : the new compute capacity of the instance partition, expressed as a number of processing units. Your instance partition must have at least 1000 processing units. Enter quantities in multiples of 1000 (1000, 2000, 3000 and so on).

Optional flags:

- `--async` : Use this flag if you want your request to return immediately, without waiting for the operation in progress to complete.

You can check the status of your request by running [`gcloud spanner operations describe`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/operations/describe) .

### Enable or modify the managed autoscaler on an instance partition

The following limitations apply when you enable or change the managed autoscaling feature on an existing instance partition:

- You can't [move an instance](https://docs.cloud.google.com/spanner/docs/move-instance) while the managed autoscaler is enabled.

### Console

1.  In the Google Cloud console, open the **Spanner** page.

2.  Select an instance from the list.

3.  In the navigation menu, select **Instance partitions** .

4.  In the list of instance partitions, under the **Actions** column, click **More Actions** and select **Edit** .

5.  Under **Configure compute capacity** , click **Autoscaling** .

6.  For **Minimum** , select the minimum limit to use when scaling down. For more information, see [Determine the minimum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-minimum) .

7.  For **Maximum** , select the maximum limit to use when scaling up. For more information, see [Determine the maximum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-maximum) .

8.  For **High priority CPU utilization target** , enter the percentage of CPU to use for high priority tasks. For more information, see [Determine the CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-cpu) .

9.  For **Total CPU utilization target** , enter the target CPU percentage to use for all [low, medium, and high priority tasks](https://docs.cloud.google.com/spanner/docs/cpu-utilization#task-priority) . The CPU percentage can range from 10 to 90%. For more information, see [Determine the total CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-total-cpu) .

10. For **Storage utilization target** , enter the percentage of storage to use. For more information, see [Determine the storage utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-storage) .

11. Click **Save** .

### gcloud

Use the [`gcloud spanner instance-partitions update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instance-partitions/update) command to enable the managed autoscaler on an instance partition. For more information and limitations, see [`Google Cloud CLI` flags and limitations](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#flags_and_limitations) .

You can add the managed autoscaler with the following command:

```
  gcloud spanner instance-partitions update INSTANCE_PARTITION_ID \
    --instance=INSTANCE_ID \
    --autoscaling-min-processing-units=MINIMUM_PROCESSING_UNITS \
    --autoscaling-max-processing-units=MAXIMUM_PROCESSING_UNITS \
    --autoscaling-high-priority-cpu-target=HIGH_PRIORITY_CPU_PERCENTAGE \
    --autoscaling-total-cpu-target=TOTAL_CPU_PERCENTAGE \
    --autoscaling-storage-target=STORAGE_PERCENTAGE
```

or

```
  gcloud spanner instance-partitions update INSTANCE_PARTITION_ID \
    --instance=INSTANCE_ID \
    --autoscaling-min-nodes=MINIMUM_NODES \
    --autoscaling-max-nodes=MAXIMUM_NODES \
    --autoscaling-high-priority-cpu-target=HIGH_PRIORITY_CPU_PERCENTAGE \
    --autoscaling-total-cpu-target=TOTAL_CPU_PERCENTAGE \
    --autoscaling-storage-target=STORAGE_PERCENTAGE
```

Replace the following:

- ` INSTANCE_PARTITION_ID ` : the permanent identifier for the instance partition.
- ` INSTANCE_ID ` : the permanent identifier for the instance.
- ` MINIMUM_PROCESSING_UNITS ` or ` MINIMUM_NODES ` : the minimum number of processing units or nodes to use when scaling down. For more information, see [Determine the minimum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-minimum) .
- ` MAXIMUM_PROCESSING_UNITS ` or ` MAXIMUM_NODES ` : the maximum number of processing units or nodes to use when scaling up. For more information, see [Determine the maximum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-maximum) .
- ` HIGH_PRIORITY_CPU_PERCENTAGE ` : the target percentage of high priority CPU to use, based on the [priority of the task](https://docs.cloud.google.com/spanner/docs/cpu-utilization#task-priority) . The CPU percentage can range from 10 to 90%. For more information, see [Determine the high priority CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-high-priority-cpu) .
- ` TOTAL_CPU_PERCENTAGE ` : the target percentage of total priority CPU to use. The total CPU target has to be greater than the high priority CPU target. The CPU percentage can range from 10 to 90%. For more information, see [Determine the total CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-total-cpu) .
- ` STORAGE_PERCENTAGE ` : the target percentage of storage to use, from 10% to 99%. For more information, see [Determine the Storage Utilization Target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-storage) .

After you enable the managed autoscaler on an instance partition, you can also modify the managed autoscaler settings. For example, if you want to increase the maximum number of processing units to 10000, run the following command:

```
gcloud spanner instance-partitions update test-instance-partition \
     --instance=test-instance
     --autoscaling-max-processing-units=10000
```

### Change an instance partition from using managed autoscaler to manual scaling

### Console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Select an instance from the list.

3.  In the navigation menu, select **Instance partitions** .

4.  In the list of instance partitions, under the **Actions** column, click **More Actions** and select **Edit** .

5.  Under **Choose a scaling mode** , check the **Manual allocation** box.

6.  Click **Save** .

### gcloud

Use the [`gcloud spanner instance-partitions update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instance-partitions/update) command to update the instance partition.

Use the following command to change an instance partition from using the managed autoscaler to manual scaling:

```
  gcloud spanner instance-partitions update INSTANCE_PARTITION_ID \
    --instance=INSTANCE_ID \
  --processing-units=PROCESSING_UNIT_COUNT
```

or

```
  gcloud spanner instance-partitions update INSTANCE_PARTITION_ID \
    --instance=INSTANCE_ID \
  --nodes=NODE_COUNT
```

Replace the following:

- ` INSTANCE_PARTITION_ID ` : the permanent identifier for the instance partition.
- ` INSTANCE_ID ` : the permanent identifier for the instance.
- ` NODE_COUNT ` : the compute capacity of the instance, expressed as a number of nodes. Each node equals 1000 processing units.
- ` PROCESSING_UNIT_COUNT ` : the compute capacity of the instance, expressed as a number of processing units. The minimum processing units for an instance partition is 1000.

## Delete an instance partition

You can't delete an instance partition while it's associated with any placements or data. You must first [move any data that's in the instance partition](https://docs.cloud.google.com/spanner/docs/create-manage-data-placements#move-row) or delete the placement tables that use the instance partition before you can delete the instance partition.

### Console

1.  In the Google Cloud console, open the **Spanner** page.

2.  Select an instance from the list.

3.  In the navigation menu, select **Instance partitions** .

4.  In the list of instance partitions, under the **Actions** column, click **More Actions** , and select **Delete** .

5.  Follow the instructions to confirm that you want to delete the Instance partition.

6.  Click **Delete** .

### gcloud

Use the [`gcloud spanner instance-partitions delete`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/delete) command.

```
gcloud spanner instance-partitions delete INSTANCE_PARTITION_ID
  --instance=INSTANCE_ID
```

## What's next

- Learn how to [create and manage placement table and keys](https://docs.cloud.google.com/spanner/docs/create-manage-data-placements) .
