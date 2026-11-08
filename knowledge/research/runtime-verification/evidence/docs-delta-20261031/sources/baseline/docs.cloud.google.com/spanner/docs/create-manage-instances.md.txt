---
name: documents/docs.cloud.google.com/spanner/docs/create-manage-instances
uri: https://docs.cloud.google.com/spanner/docs/create-manage-instances
title: Create and manage instances
description: Manage Spanner instances, including configuring compute capacity, regions, and other settings.
data_source: docs.cloud.google.com
---

This page describes how to create, list, edit, and delete Spanner [instances](https://docs.cloud.google.com/spanner/docs/instances) .

> **Note:** To explore Spanner using a 90-day free trial instance, see [Create a Spanner free trial instance](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart) .

## Create an instance

You can create an instance with the Google Cloud console, the [Google Cloud CLI](https://docs.cloud.google.com/spanner/docs/gcloud-spanner) , or client libraries. You can also create an instance with a [custom instance configuration](https://docs.cloud.google.com/spanner/docs/instance-configurations#configuration) by adding optional [read-only replicas](https://docs.cloud.google.com/spanner/docs/replication#read-only) .

> **Note:** You don't need to provision storage in Spanner, and you are billed only for the storage that you use. Learn more about [storage pricing](https://docs.cloud.google.com/spanner/pricing#storage) .

### Console

1.  In the Google Cloud console, go to the **Spanner Instances** page.

2.  Click **Create instance** .

3.  In the **Select an edition** section, select a Spanner edition.

    If you want to compare the specifications between the different editions, then click **Compare editions** . For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

4.  Click **Continue** .

5.  In the **Name your instance** section, enter an **Instance name** to display in the Google Cloud console. The instance name must be unique within your Google Cloud project.

6.  Enter an **Instance ID** to permanently identify your instance. The instance ID must also be unique within your Google Cloud project. You can't change the instance ID later.

7.  Click **Continue** .

8.  In the **Configure your instance** section, under **Choose a configuration** , select **Regional** , **Dual-region** , or **Multi-region** .

    > **Note:** Dual-region and multi-region instance configurations are only available in the Enterprise Plus edition. If you want to compare the specifications between the three configurations, then click **Compare region configurations** .

9.  Select a configuration location from the drop-down.

10. Optional: To add a read-only replica to a Spanner base configuration, first [create a custom instance configuration using the Google Cloud CLI](https://docs.cloud.google.com/spanner/docs/create-manage-configurations#gcloud_2) . Adding optional read-only replicas to a base configuration is available in the [Enterprise edition and Enterprise Plus edition](https://docs.cloud.google.com/spanner/docs/editions-overview#features) .

11. Click **Continue** .

12. In the **Allocate compute capacity** section, under **Select unit** , click one of the following:

    - **Nodes** for large instances. A node is 1000 processing units.
    - **Processing units** for small instances.

    For more information, see [Compute capacity, nodes and processing units](https://docs.cloud.google.com/spanner/docs/compute-capacity) .

13. Under **Choose a scaling mode** , click one of the following:

    - **Manual allocation** if you want to manually set compute capacity for fixed compute resources and costs.

      - **Quantity** indicates the number of processing units or nodes to use for this instance.

    - **Autoscaling** to let Spanner automatically add and remove compute capacity. Managed autoscaler is available in the [Spanner Enterprise edition and Enterprise Plus edition](https://docs.cloud.google.com/spanner/docs/editions-overview) . For more information about the managed autoscaler, see [Managed autoscaler for Spanner](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) . Configure the following managed autoscaler options:

      - **Minimum** indicates the minimum limit to scale down to, depending on the measurement unit that you choose for **Compute capacity** . For more information, see [Determine the minimum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-minimum) .
      - **Maximum** indicates the maximum limit to scale up to, depending on the measurement unit that you choose for **Compute capacity** . For more information, see [Determine the maximum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-maximum) .
      - **High priority CPU utilization target** indicates the target percentage of CPU to use for high priority tasks. For more information, see [Determine the CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-high-priority-cpu) .
      - **Total CPU utilization target** indicates the target percentage of CPU to use for all low, medium, and high priority tasks. For more information, see [Determine the total CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-total-cpu) .
      - **Storage utilization target** indicates the target percentage of storage to use. For more information, see [Determine the Storage Utilization Target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-storage) .

14. Optional: If you select **Autoscaling** as the scaling mode, you can click the **Show asymmetric autoscaling options** dropdown to autoscale your read-only replicas independently from other replicas. For more information, see [Asymmetric read-only autoscaling](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#asymmetric-read-only-autoscaling) .

    1.  Select the read-only replica you want to asymmetrically autoscale.

    2.  Configure the following asymmetric autoscaler options:

        - **Minimum** indicates the minimum limit to scale down to, depending on the measurement unit that you choose for **Compute capacity** . For more information, see [Determine the minimum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-minimum) .
        - **Maximum** indicates the maximum limit to scale up to, depending on the measurement unit that you choose for **Compute capacity** . For more information, see [Determine the maximum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-maximum) .
        - **High priority CPU utilization target** indicates the target percentage of CPU to use for [high priority tasks](https://docs.cloud.google.com/spanner/docs/cpu-utilization#task-priority) . For more information, see [Determine the high priority CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-high-priority-cpu) .
        - **Total CPU utilization target** indicates the target percentage of CPU to use for all [low, medium, and high priority tasks](https://docs.cloud.google.com/spanner/docs/cpu-utilization#task-priority) . The CPU percentage can range from 10 to 90%. The total CPU target has to be greater than the high priority CPU target. If you're optimizing for cost, then use a higher percentage. For more information, see [Determine the total CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-total-cpu) .
        - **Disable total CPU** indicates that the autoscaler should explicitly ignore the total CPU target.
        - **Disable high priority CPU** indicates that the autoscaler should explicitly ignore the CPU target for high priority tasks.

15. Under **Backups** , the **Enable default backup schedules** checkbox is checked by default. To disable default backup schedules, uncheck the checkbox. When enabled, all new databases in the instance have full backups created every 24 hours. These backups are retained for 7 days. You can edit or delete the default backup schedules at any time. For more information, see [Default backup schedules](https://docs.cloud.google.com/spanner/docs/backup#default-backup-schedules) .

16. Click **Create** to create the instance.

### gcloud

Use the [`gcloud spanner instances create`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/create) command to create an instance. Specify the [compute capacity](https://docs.cloud.google.com/spanner/docs/compute-capacity) as the number of nodes or processing units that you want on the instance.

```
gcloud spanner instances create INSTANCE_ID \
--edition=EDITION \
--config=INSTANCE_CONFIG \
--description=INSTANCE_DESCRIPTION \
--default-backup-schedule-type=DEFAULT_BACKUP_SCHEDULE_TYPE \
--nodes=NODE_COUNT
```

or

```
gcloud spanner instances create INSTANCE_ID \
--edition=EDITION \
--config=INSTANCE_CONFIG \
--description=INSTANCE_DESCRIPTION \
--default-backup-schedule-type=DEFAULT_BACKUP_SCHEDULE_TYPE \
--processing-units=PROCESSING_UNIT_COUNT
```

Replace the following:

- ` INSTANCE-ID ` : a permanent identifier that is unique within your Google Cloud project. You can't change the instance ID later.

- ` INSTANCE-CONFIG ` : a permanent identifier of your instance configuration, which defines the geographic location of the instance and affects how data is replicated. For custom instance configurations, it starts with `custom-` . For more information, see [instance configurations](https://docs.cloud.google.com/spanner/docs/instances) .

- ` INSTANCE_DESCRIPTION ` : the name to display for the instance in the Google Cloud console. The instance name must be unique within your Google Cloud project.

- ` DEFAULT_BACKUP_SCHEDULE_TYPE ` : the default backup schedule type that is used in the instance. Must be one of the following values:

  - `AUTOMATIC` : a default backup schedule is created automatically when a new database is created in the instance. The default backup schedule creates a full backup every 24 hours. These full backups are retained for 7 days. You can edit or delete the default backup schedule once it's created.
  - `NONE` : a default backup schedule isn't created automatically when a new database is created in the instance.

- ` NODE-COUNT ` : the compute capacity of the instance, expressed as a number of nodes. Each node equals 1000 processing units.

- ` PROCESSING_UNIT_COUNT ` : the compute capacity of the instance, expressed as a number of processing units. Enter quantities up to 1000 in multiples of 100 (100, 200, 300 and so on) and enter greater quantities in multiples of 1000 (1000, 2000, 3000 and so on). Note: Don't use this parameter if you're creating an instance that you intend to enable with the managed autoscaler later.

### Add managed autoscaling

You can also create [Enterprise edition and Enterprise Plus edition](https://docs.cloud.google.com/spanner/docs/editions-overview) instances to use managed autoscaling with the [`gcloud spanner instances create`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/create) command. For more information, see [Managed autoscaler for Spanner](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) .

Use the following command to create an instance with managed autoscaler.

```
  gcloud spanner instances create INSTANCE_ID \
    --edition=EDITION \
    --config=INSTANCE_CONFIG \
    --description=INSTANCE_DESCRIPTION \
    --autoscaling-min-processing-units=MINIMUM_PROCESSING_UNITS \
    --autoscaling-max-processing-units=MAXIMUM_PROCESSING_UNITS \
    --autoscaling-high-priority-cpu-target=HIGH_PRIORITY_CPU_PERCENTAGE \
    --autoscaling-total-cpu-target=TOTAL_CPU_PERCENTAGE \
    --autoscaling-storage-target=STORAGE_PERCENTAGE \
    [--disable-downscaling | --no-disable-downscaling] \
    [--asymmetric-autoscaling-option \
       location=ASYMMETRIC_AUTOSCALING_LOCATION,\
       min_nodes=ASYMMETRIC_AUTOSCALING_MIN,\
       max_nodes=ASYMMETRIC_AUTOSCALING_MAX,\
       high_priority_cpu_target=ASYMMETRIC_HIGH_PRIORITY_CPU_TARGET,\
       total_cpu_target=ASYMMETRIC_TOTAL_CPU_TARGET \
       [,disable_total_cpu_autoscaling=TRUE|FALSE \
       | disable_high_priority_cpu_autoscaling=TRUE|FALSE]
    ]
```

or

```
  gcloud spanner instances create INSTANCE_ID \
    --edition=EDITION \
    --config=INSTANCE_CONFIG \
    --description=INSTANCE_DESCRIPTION \
    --autoscaling-min-nodes=MINIMUM_NODES \
    --autoscaling-max-nodes=MAXIMUM_NODES \
    --autoscaling-high-priority-cpu-target=HIGH_PRIORITY_CPU_PERCENTAGE \
    --autoscaling-total-cpu-target=TOTAL_CPU_PERCENTAGE \
    --autoscaling-storage-target=STORAGE_PERCENTAGE \
    [--disable-downscaling | --no-disable-downscaling] \
    [--asymmetric-autoscaling-option \
       location=ASYMMETRIC_AUTOSCALING_LOCATION,\
       min_nodes=ASYMMETRIC_AUTOSCALING_MIN,\
       max_nodes=ASYMMETRIC_AUTOSCALING_MAX,\
       high_priority_cpu_target=ASYMMETRIC_HIGH_PRIORITY_CPU_TARGET, \
       total_cpu_target=ASYMMETRIC_TOTAL_CPU_TARGET, \
       [,disable_total_cpu_autoscaling=TRUE|FALSE \
       | disable_high_priority_cpu_autoscaling=TRUE|FALSE]
    ]
```

Replace the following:

- ` INSTANCE-ID ` : a permanent identifier that is unique within your Google Cloud project. You can't change the instance ID later.
- ` INSTANCE-CONFIG ` : a permanent identifier of your instance configuration, which defines the geographic location of the instance and affects how data is replicated. For custom instance configurations, it starts with `custom-` . For more information, see [instance configurations](https://docs.cloud.google.com/spanner/docs/instances) .
- ` INSTANCE-DESCRIPTION ` : the name to display for the instance in the Google Cloud console. The instance name must be unique within your Google Cloud project.
- ` MINIMUM_PROCESSING_UNITS ` , ` MINIMUM_NODES ` : the minimum number of processing units or nodes when scaling down. For more information, see [Determine the minimum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-minimum) .
- ` MAXIMUM_PROCESSING_UNITS ` , ` MAXIMUM_NODES ` : the maximum number of processing units or nodes when scaling up. For more information, see [Determine the maximum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-maximum) .
- ` HIGH_PRIORITY_CPU_PERCENTAGE ` : the target percentage of high priority CPU to use, based on the [priority of the task](https://docs.cloud.google.com/spanner/docs/cpu-utilization#task-priority) . The CPU percentage can range from 10 to 90%. If you're optimizing for cost, then use a higher percentage. For more information, see [Determine the high priority CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-high-priority-cpu) .
- ` TOTAL_CPU_PERCENTAGE ` : the target percentage of total priority CPU to use. The total CPU target has to be greater than the high priority CPU target. The CPU percentage can range from 10 to 90%. If you're optimizing for cost, then use a higher percentage. For more information, see [Determine the total CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-total-cpu) .
- ` STORAGE_PERCENTAGE ` : the target percentage of storage to use, from 10 to 99%. For more information, see [Determine the storage utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-storage) .

Optional flags:

- `--disable-downscaling` : use this boolean flag to disable downscaling. When set to `TRUE` , it prevents the autoscaler from reducing the number of nodes or processing units. Upscaling continues to function normally to meet increased demand. To enable downscaling after disabling it, use the `--no-disable-downscaling` flag.

- `--asymmetric-autoscaling-option` : use this flag to enable [asymmetric autoscaling](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#asymmetric-read-only-autoscaling) . Replace the following parameters:

  - ` ASYMMETRIC_AUTOSCALING_LOCATION ` : if the flag is used, then this parameter is required. The location of the read-only region that you want to scale asymmetrically.
  - ` ASYMMETRIC_AUTOSCALING_MIN ` : optional parameter. The minimum number of nodes when scaling down. If not specified, this value is inherited from the base instance configuration.
  - ` ASYMMETRIC_AUTOSCALING_MAX ` : optional parameter. The maximum number of nodes when scaling up. If not specified, this value is inherited from the base instance configuration.
  - ` ASYMMETRIC_HIGH_PRIORITY_CPU_TARGET ` : optional parameter. The target percentage of CPU to use, based on the [priority of the task](https://docs.cloud.google.com/spanner/docs/cpu-utilization#task-priority) . If not specified, this value is inherited from the base instance configuration. If you're optimizing for cost, then use a higher percentage. Replicas can have a different target percentage than the base instance and from other replicas, but for failover scenarios, we recommend that replicas use consistent targets across different replicas. For more information, see [Failover concerns](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#failover-concerns) .
  - ` ASYMMETRIC_TOTAL_CPU_TARGET ` : optional parameter. Set the target from 10 to 90% for the [total CPU](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-total-cpu) . If not specified, this value is inherited from the base instance configuration. If you're optimizing for cost, then use a higher percentage. Replicas can have a different target percentage than the base instance and from other replicas, but for failover scenarios, we recommend that replicas use consistent targets across different replicas. For more information, see [Asymmetric read-only autoscaling](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#asymmetric-read-only-autoscaling) .
  - `disable_total_cpu_autoscaling` : set this parameter to `TRUE` if you don't want the autoscaler to autoscale total CPU on replicas. Don't specify this parameter if you want the instance to inherit the `total_cpu_autoscaling` target from the base instance. You can't disable both total CPU and high priority CPU on the same replica.
  - `disable_high_priority_cpu_autoscaling` : set the `disable_high_priority_cpu_autoscaling` to `TRUE` if you don't want the autoscaler to autoscale the high priority CPU on replicas. Don't specify this parameter if you want the instance to inherit the `high_priority_cpu_autoscaling` target from the base instance. You can't disable both total CPU and high priority CPU on the same replica.

### Examples for using custom configurations

To create an instance `test-instance` in the base regional instance configuration `us-central1` , run:

```
gcloud spanner instances create test-instance --edition=STANDARD --config=regional-us-central1 \
  --description="Test Instance" --nodes=1
```

To create an instance `custom-eur6-instance` in the custom multi-region instance configuration `custom-eur6` , first [create a custom instance configuration](https://docs.cloud.google.com/spanner/docs/create-manage-configurations#create-configuration) .

Then, run:

```
  gcloud spanner instances create custom-eur6-instance --edition=ENTERPRISE_PLUS --config=custom-eur6 \
      --description="Instance with custom read-only" --nodes=1
```

You should see a message similar to the following example after running either one of the previous commands:

```
Creating instance...done.
```

### C++

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```cpp
void CreateInstance(google::cloud::spanner_admin::InstanceAdminClient client,
                    std::string const& project_id,
                    std::string const& instance_id,
                    std::string const& display_name,
                    std::string const& config_id) {
  namespace spanner = ::google::cloud::spanner;
  spanner::Instance in(project_id, instance_id);

  auto project = google::cloud::Project(project_id);
  std::string config_name =
      project.FullName() + "/instanceConfigs/" + config_id;
  auto instance =
      client
          .CreateInstance(spanner::CreateInstanceRequestBuilder(in, config_name)
                              .SetDisplayName(display_name)
                              .SetNodeCount(1)
                              .SetLabels({{"cloud_spanner_samples", "true"}})
                              .Build())
          .get();
  if (!instance) throw std::move(instance).status();
  std::cout << "Created instance [" << in << "]:\n" << instance->DebugString();
}
```

### C#

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```csharp
using Google.Api.Gax.ResourceNames;
using Google.Cloud.Spanner.Admin.Instance.V1;
using Google.Cloud.Spanner.Common.V1;
using Google.LongRunning;
using System;
using System.Threading.Tasks;

public class CreateInstanceAsyncSample
{
    public async Task<Instance> CreateInstanceAsync(
        string projectId,
        string instanceId,
        Instance.Types.Edition edition = Instance.Types.Edition.Standard)
    {
        // Create the InstanceAdminClient instance.
        InstanceAdminClient instanceAdminClient = await InstanceAdminClient.CreateAsync();

        // Initialize request parameters.
        Instance instance = new Instance
        {
            InstanceName = InstanceName.FromProjectInstance(projectId, instanceId),
            ConfigAsInstanceConfigName = InstanceConfigName.FromProjectInstanceConfig(projectId, "regional-us-central1"),
            DisplayName = "This is a display name.",
            NodeCount = 1,
            Labels =
            {
                { "cloud_spanner_samples", "true" },
            },
            Edition = edition,
        };
        ProjectName projectName = ProjectName.FromProject(projectId);

        // Make the CreateInstance request.
        Operation<Instance, CreateInstanceMetadata> response = await instanceAdminClient.CreateInstanceAsync(projectName, instanceId, instance);

        Console.WriteLine("Waiting for the operation to finish.");

        // Poll until the returned long-running operation is complete.
        Operation<Instance, CreateInstanceMetadata> completedResponse = await response.PollUntilCompletedAsync();

        if (completedResponse.IsFaulted)
        {
            Console.WriteLine($"Error while creating instance: {completedResponse.Exception}");
            throw completedResponse.Exception;
        }

        Console.WriteLine($"Instance created successfully.");

        return completedResponse.Result;
    }
}
```

### Create an instance without a default backup schedule

```csharp
using Google.Api.Gax.ResourceNames;
using Google.Cloud.Spanner.Admin.Instance.V1;
using Google.Cloud.Spanner.Common.V1;
using Google.LongRunning;
using System;
using System.Threading.Tasks;

public class CreateInstanceWithoutDefaultBackupSchedulesAsyncSample
{
    public async Task<Instance> CreateInstanceWithoutDefaultBackupSchedulesAsync(string projectId, string instanceId)
    {
        // Create the InstanceAdminClient instance.
        InstanceAdminClient instanceAdminClient = await InstanceAdminClient.CreateAsync();

        // Initialize request parameters.
        Instance instance = new Instance
        {
            InstanceName = InstanceName.FromProjectInstance(projectId, instanceId),
            ConfigAsInstanceConfigName =
                InstanceConfigName.FromProjectInstanceConfig(projectId, "regional-me-central2"),
            DisplayName = "This is a display name.",
            NodeCount = 1,
            Labels =
            {
                { "cloud_spanner_samples", "true" },
            },
            DefaultBackupScheduleType = Instance.Types.DefaultBackupScheduleType.None,
        };
        ProjectName projectName = ProjectName.FromProject(projectId);

        // Make the CreateInstance request.
        Operation<Instance, CreateInstanceMetadata> response =
            await instanceAdminClient.CreateInstanceAsync(projectName, instanceId, instance);

        Console.WriteLine("Waiting for the operation to finish.");

        // Poll until the returned long-running operation is complete.
        Operation<Instance, CreateInstanceMetadata> completedResponse =
            await response.PollUntilCompletedAsync();

        if (completedResponse.IsFaulted)
        {
            Console.WriteLine($"Error while creating instance: {completedResponse.Exception}");
            throw completedResponse.Exception;
        }

        Console.WriteLine($"Instance created successfully.");
        return completedResponse.Result;
    }
}
```

### Go

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```go
import (
    "context"
    "fmt"
    "io"

    instance "cloud.google.com/go/spanner/admin/instance/apiv1"
    "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
)

func createInstance(w io.Writer, projectID, instanceID string) error {
    // projectID := "my-project-id"
    // instanceID := "my-instance"
    ctx := context.Background()
    instanceAdmin, err := instance.NewInstanceAdminClient(ctx)
    if err != nil {
        return err
    }
    defer instanceAdmin.Close()

    op, err := instanceAdmin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
        Parent:     fmt.Sprintf("projects/%s", projectID),
        InstanceId: instanceID,
        Instance: &instancepb.Instance{
            Config:      fmt.Sprintf("projects/%s/instanceConfigs/%s", projectID, "regional-us-central1"),
            DisplayName: instanceID,
            NodeCount:   1,
            Labels:      map[string]string{"cloud_spanner_samples": "true"},
            Edition:     instancepb.Instance_STANDARD,
        },
    })
    if err != nil {
        return fmt.Errorf("could not create instance %s: %w", fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID), err)
    }
    // Wait for the instance creation to finish.
    i, err := op.Wait(ctx)
    if err != nil {
        return fmt.Errorf("waiting for instance creation to finish failed: %w", err)
    }
    // The instance may not be ready to serve yet.
    if i.State != instancepb.Instance_READY {
        fmt.Fprintf(w, "instance state is not READY yet. Got state %v\n", i.State)
    }
    fmt.Fprintf(w, "Created instance [%s]\n", instanceID)
    return nil
}
```

### Create an instance with managed autoscaling using Go

```go
import (
    "context"
    "fmt"
    "io"

    instance "cloud.google.com/go/spanner/admin/instance/apiv1"
    "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
    "google.golang.org/genproto/protobuf/field_mask"
)

// Example of creating an autoscaling instance with Go.
// projectID is the ID of the project that the new instance will be in.
// instanceID is the ID of the new instance to be created.
func createInstanceWithAutoscalingConfig(w io.Writer, projectID, instanceID string) error {
    // projectID := "my-project-id"
    // instanceID := "my-instance"
    ctx := context.Background()
    instanceAdmin, err := instance.NewInstanceAdminClient(ctx)
    if err != nil {
        return fmt.Errorf("could not create instance admin client for project %s: %w", projectID, err)
    }
    defer instanceAdmin.Close()

    instanceName := fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID)
    fmt.Fprintf(w, "Creating instance %s.", instanceName)

    op, err := instanceAdmin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
        Parent:     fmt.Sprintf("projects/%s", projectID),
        InstanceId: instanceID,
        Instance: &instancepb.Instance{
            Config:      fmt.Sprintf("projects/%s/instanceConfigs/%s", projectID, "regional-us-central1"),
            DisplayName: "Create instance example",
            AutoscalingConfig: &instancepb.AutoscalingConfig{
                AutoscalingLimits: &instancepb.AutoscalingConfig_AutoscalingLimits{
                    MinLimit: &instancepb.AutoscalingConfig_AutoscalingLimits_MinNodes{
                        MinNodes: 1,
                    },
                    MaxLimit: &instancepb.AutoscalingConfig_AutoscalingLimits_MaxNodes{
                        MaxNodes: 2,
                    },
                },
                AutoscalingTargets: &instancepb.AutoscalingConfig_AutoscalingTargets{
                    HighPriorityCpuUtilizationPercent: 65,
                    StorageUtilizationPercent:         95,
                },
            },
            Labels:  map[string]string{"cloud_spanner_samples": "true"},
            Edition: instancepb.Instance_ENTERPRISE_PLUS,
        },
    })
    if err != nil {
        return fmt.Errorf("could not create instance %s: %w", instanceName, err)
    }
    fmt.Fprintf(w, "Waiting for operation on %s to complete...", instanceID)
    // Wait for the instance creation to finish.
    i, err := op.Wait(ctx)
    if err != nil {
        return fmt.Errorf("waiting for instance creation to finish failed: %w", err)
    }
    // The instance may not be ready to serve yet.
    if i.State != instancepb.Instance_READY {
        fmt.Fprintf(w, "instance state is not READY yet. Got state %v\n", i.State)
    }
    fmt.Fprintf(w, "Created instance [%s].\n", instanceID)

    instance, err := instanceAdmin.GetInstance(ctx, &instancepb.GetInstanceRequest{
        Name: instanceName,
        // Get the autoscaling_config field from the newly created instance.
        FieldMask: &field_mask.FieldMask{Paths: []string{"autoscaling_config"}},
    })
    if err != nil {
        return fmt.Errorf("failed to get instance [%s]: %w", instanceName, err)
    }
    fmt.Fprintf(w, "Instance %s has autoscaling_config: %s.", instanceID, instance.AutoscalingConfig)
    return nil
}
```

### Create an instance with asymmetric read-only autoscaling using Go

```go
import (
    "context"
    "fmt"
    "io"

    instance "cloud.google.com/go/spanner/admin/instance/apiv1"
    "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
    "google.golang.org/genproto/protobuf/field_mask"
)

// createInstanceWithAsymmetricAutoscalingConfig is a code snippet to show
// an example of creating an asymmetric autoscaling enabled instance in Go.
//
// projectID is the ID of the project that the new instance will be in.
// instanceID is the ID of the new instance to be created.
func createInstanceWithAsymmetricAutoscalingConfig(w io.Writer, projectID, instanceID string) error {
    ctx := context.Background()
    instanceAdmin, err := instance.NewInstanceAdminClient(ctx)
    if err != nil {
        return fmt.Errorf("could not create instance admin client for project %s: %w", projectID, err)
    }
    defer instanceAdmin.Close()

    instanceName := fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID)
    fmt.Fprintf(w, "Creating instance %s.", instanceName)

    op, err := instanceAdmin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
        Parent:     fmt.Sprintf("projects/%s", projectID),
        InstanceId: instanceID,
        Instance: &instancepb.Instance{
            Config:      fmt.Sprintf("projects/%s/instanceConfigs/%s", projectID, "nam-eur-asia3"),
            DisplayName: "Create instance example",
            AutoscalingConfig: &instancepb.AutoscalingConfig{
                AutoscalingLimits: &instancepb.AutoscalingConfig_AutoscalingLimits{
                    MinLimit: &instancepb.AutoscalingConfig_AutoscalingLimits_MinNodes{
                        MinNodes: 1,
                    },
                    MaxLimit: &instancepb.AutoscalingConfig_AutoscalingLimits_MaxNodes{
                        MaxNodes: 10,
                    },
                },
                AutoscalingTargets: &instancepb.AutoscalingConfig_AutoscalingTargets{
                    HighPriorityCpuUtilizationPercent: 65,
                    StorageUtilizationPercent:         95,
                },
                // Read-only replicas in europe-west1, europe-west4, and asia-east1 are autoscaled
                // independly from other replicas based on the usage in the respective region.
                AsymmetricAutoscalingOptions: []*instancepb.AutoscalingConfig_AsymmetricAutoscalingOption{
                    &instancepb.AutoscalingConfig_AsymmetricAutoscalingOption{
                        ReplicaSelection: &instancepb.ReplicaSelection{
                            Location: "europe-west1",
                        },
                    },
                    &instancepb.AutoscalingConfig_AsymmetricAutoscalingOption{
                        ReplicaSelection: &instancepb.ReplicaSelection{
                            Location: "europe-west4",
                        },
                    },
                    &instancepb.AutoscalingConfig_AsymmetricAutoscalingOption{
                        ReplicaSelection: &instancepb.ReplicaSelection{
                            Location: "asia-east1",
                        },
                    },
                },
            },
            Labels:  map[string]string{"cloud_spanner_samples": "true"},
            Edition: instancepb.Instance_ENTERPRISE_PLUS,
        },
    })
    if err != nil {
        return fmt.Errorf("could not create instance %s: %w", instanceName, err)
    }
    fmt.Fprintf(w, "Waiting for operation on %s to complete...", instanceID)
    // Wait for the instance creation to finish.
    i, err := op.Wait(ctx)
    if err != nil {
        return fmt.Errorf("waiting for instance creation to finish failed: %w", err)
    }
    // The instance may not be ready to serve yet.
    if i.State != instancepb.Instance_READY {
        fmt.Fprintf(w, "instance state is not READY yet. Got state %v\n", i.State)
    }
    fmt.Fprintf(w, "Created instance [%s].\n", instanceID)

    instance, err := instanceAdmin.GetInstance(ctx, &instancepb.GetInstanceRequest{
        Name: instanceName,
        // Get the autoscaling_config field from the newly created instance.
        FieldMask: &field_mask.FieldMask{Paths: []string{"autoscaling_config"}},
    })
    if err != nil {
        return fmt.Errorf("failed to get instance [%s]: %w", instanceName, err)
    }
    fmt.Fprintf(w, "Instance %s has autoscaling_config: %s.", instanceID, instance.AutoscalingConfig)
    return nil
}
```

### Create an instance without a default backup schedule

```go
import (
    "context"
    "fmt"
    "io"

    instance "cloud.google.com/go/spanner/admin/instance/apiv1"
    "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
)

// createInstanceWithoutDefaultBackupSchedule creates instance with default backup schedule disabled.
func createInstanceWithoutDefaultBackupSchedule(w io.Writer, projectID, instanceID string) error {
    // projectID := "my-project-id"
    // instanceID := "my-instance"
    ctx := context.Background()
    instanceAdmin, err := instance.NewInstanceAdminClient(ctx)
    if err != nil {
        return err
    }
    defer instanceAdmin.Close()

    // Create an instance without default backup schedule, whicn means no default backup schedule will
    // be created automatically on creation of a database within the instance.
    req := &instancepb.CreateInstanceRequest{
        Parent:     fmt.Sprintf("projects/%s", projectID),
        InstanceId: instanceID,
        Instance: &instancepb.Instance{
            Config:                    fmt.Sprintf("projects/%s/instanceConfigs/%s", projectID, "regional-us-central1"),
            DisplayName:               instanceID,
            NodeCount:                 1,
            Labels:                    map[string]string{"cloud_spanner_samples": "true"},
            DefaultBackupScheduleType: instancepb.Instance_NONE,
        },
    }

    op, err := instanceAdmin.CreateInstance(ctx, req)
    if err != nil {
        return fmt.Errorf("could not create instance %s: %w", fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID), err)
    }
    // Wait for the instance creation to finish.  For more information about instances, see
    // https://cloud.google.com/spanner/docs/instances.
    instance, err := op.Wait(ctx)
    if err != nil {
        return fmt.Errorf("waiting for instance creation to finish failed: %w", err)
    }
    // The instance may not be ready to serve yet.
    if instance.State != instancepb.Instance_READY {
        fmt.Fprintf(w, "instance state is not READY yet. Got state %v\n", instance.State)
    }
    fmt.Fprintf(w, "Created instance [%s]\n", instanceID)
    return nil
}
```

### Java

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```java
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.admin.instance.v1.InstanceAdminClient;
import com.google.spanner.admin.instance.v1.CreateInstanceRequest;
import com.google.spanner.admin.instance.v1.Instance;
import com.google.spanner.admin.instance.v1.InstanceConfigName;
import com.google.spanner.admin.instance.v1.ProjectName;
import java.util.concurrent.ExecutionException;

class CreateInstanceExample {

  static void createInstance() {
    // TODO(developer): Replace these variables before running the sample.
    String projectId = "my-project";
    String instanceId = "my-instance";
    createInstance(projectId, instanceId);
  }

  static void createInstance(String projectId, String instanceId) {
    // Set Instance configuration.
    int nodeCount = 2;
    String displayName = "Descriptive name";

    // Create an Instance object that will be used to create the instance.
    Instance instance =
        Instance.newBuilder()
            .setDisplayName(displayName)
            .setEdition(Instance.Edition.STANDARD)
            .setNodeCount(nodeCount)
            .setConfig(InstanceConfigName.of(projectId, "regional-us-east4").toString())
            .build();

    try (Spanner spanner =
            SpannerOptions.newBuilder().setProjectId(projectId).build().getService();
        InstanceAdminClient instanceAdminClient = spanner.createInstanceAdminClient()) {

      // Wait for the createInstance operation to finish.
      Instance createdInstance =
          instanceAdminClient
              .createInstanceAsync(
                  CreateInstanceRequest.newBuilder()
                      .setParent(ProjectName.of(projectId).toString())
                      .setInstanceId(instanceId)
                      .setInstance(instance)
                      .build())
              .get();
      System.out.printf("Instance %s was successfully created%n", createdInstance.getName());
    } catch (ExecutionException e) {
      System.out.printf(
          "Error: Creating instance %s failed with error message %s%n",
          instance.getName(), e.getMessage());
    } catch (InterruptedException e) {
      System.out.println("Error: Waiting for createInstance operation to finish was interrupted");
    }
  }
}
```

### Create an instance with managed autoscaling using Java

```java
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.admin.instance.v1.InstanceAdminClient;
import com.google.spanner.admin.instance.v1.AutoscalingConfig;
import com.google.spanner.admin.instance.v1.CreateInstanceRequest;
import com.google.spanner.admin.instance.v1.Instance;
import com.google.spanner.admin.instance.v1.InstanceConfigName;
import com.google.spanner.admin.instance.v1.ProjectName;
import com.google.spanner.admin.instance.v1.ReplicaSelection;
import java.util.concurrent.ExecutionException;

class CreateInstanceWithAsymmetricAutoscalingConfigExample {

  static void createInstance() {
    // TODO(developer): Replace these variables before running the sample.
    String projectId = "my-project";
    String instanceId = "my-instance";
    createInstance(projectId, instanceId);
  }

  static void createInstance(String projectId, String instanceId) {
    try (Spanner spanner =
        SpannerOptions.newBuilder()
            .setProjectId(projectId)
            .build()
            .getService();
        InstanceAdminClient instanceAdminClient = spanner.createInstanceAdminClient()) {
      // Set Instance configuration.
      String configId = "nam-eur-asia3";
      String displayName = "Descriptive name";

      // Create an autoscaling config.
      // When autoscaling_config is enabled, node_count and processing_units fields
      // need not be specified.
      // The read-only replicas listed in the asymmetric autoscaling options scale independently
      // from other replicas.
      AutoscalingConfig autoscalingConfig =
          AutoscalingConfig.newBuilder()
              .setAutoscalingLimits(
                  AutoscalingConfig.AutoscalingLimits.newBuilder().setMinNodes(1).setMaxNodes(2))
              .setAutoscalingTargets(
                  AutoscalingConfig.AutoscalingTargets.newBuilder()
                      .setHighPriorityCpuUtilizationPercent(65)
                      .setStorageUtilizationPercent(95))
              .addAsymmetricAutoscalingOptions(
                  AutoscalingConfig.AsymmetricAutoscalingOption.newBuilder()
                  .setReplicaSelection(ReplicaSelection.newBuilder().setLocation("europe-west1")))
              .addAsymmetricAutoscalingOptions(
                  AutoscalingConfig.AsymmetricAutoscalingOption.newBuilder()
                  .setReplicaSelection(ReplicaSelection.newBuilder().setLocation("europe-west4")))
              .addAsymmetricAutoscalingOptions(
                  AutoscalingConfig.AsymmetricAutoscalingOption.newBuilder()
                  .setReplicaSelection(ReplicaSelection.newBuilder().setLocation("asia-east1")))
              .build();
      Instance instance =
          Instance.newBuilder()
              .setAutoscalingConfig(autoscalingConfig)
              .setDisplayName(displayName)
              .setConfig(
                  InstanceConfigName.of(projectId, configId).toString())
              .build();

      // Creates a new instance
      System.out.printf("Creating instance %s.%n", instanceId);
      try {
        // Wait for the createInstance operation to finish.
        Instance instanceResult = instanceAdminClient.createInstanceAsync(
            CreateInstanceRequest.newBuilder()
                .setParent(ProjectName.of(projectId).toString())
                .setInstanceId(instanceId)
                .setInstance(instance)
                .build()).get();
        System.out.printf("Asymmetric Autoscaling instance %s was successfully created%n",
            instanceResult.getName());
      } catch (ExecutionException e) {
        System.out.printf(
            "Error: Creating instance %s failed with error message %s%n",
            instance.getName(), e.getMessage());
      } catch (InterruptedException e) {
        System.out.println("Error: Waiting for createInstance operation to finish was interrupted");
      }
    }
  }
}
```

### Create an instance with asymmetric read-only autoscaling using Java

```java
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.admin.instance.v1.InstanceAdminClient;
import com.google.spanner.admin.instance.v1.AutoscalingConfig;
import com.google.spanner.admin.instance.v1.CreateInstanceRequest;
import com.google.spanner.admin.instance.v1.Instance;
import com.google.spanner.admin.instance.v1.InstanceConfigName;
import com.google.spanner.admin.instance.v1.ProjectName;
import com.google.spanner.admin.instance.v1.ReplicaSelection;
import java.util.concurrent.ExecutionException;

class CreateInstanceWithAsymmetricAutoscalingConfigExample {

  static void createInstance() {
    // TODO(developer): Replace these variables before running the sample.
    String projectId = "my-project";
    String instanceId = "my-instance";
    createInstance(projectId, instanceId);
  }

  static void createInstance(String projectId, String instanceId) {
    try (Spanner spanner =
        SpannerOptions.newBuilder()
            .setProjectId(projectId)
            .build()
            .getService();
        InstanceAdminClient instanceAdminClient = spanner.createInstanceAdminClient()) {
      // Set Instance configuration.
      String configId = "nam-eur-asia3";
      String displayName = "Descriptive name";

      // Create an autoscaling config.
      // When autoscaling_config is enabled, node_count and processing_units fields
      // need not be specified.
      // The read-only replicas listed in the asymmetric autoscaling options scale independently
      // from other replicas.
      AutoscalingConfig autoscalingConfig =
          AutoscalingConfig.newBuilder()
              .setAutoscalingLimits(
                  AutoscalingConfig.AutoscalingLimits.newBuilder().setMinNodes(1).setMaxNodes(2))
              .setAutoscalingTargets(
                  AutoscalingConfig.AutoscalingTargets.newBuilder()
                      .setHighPriorityCpuUtilizationPercent(65)
                      .setStorageUtilizationPercent(95))
              .addAsymmetricAutoscalingOptions(
                  AutoscalingConfig.AsymmetricAutoscalingOption.newBuilder()
                  .setReplicaSelection(ReplicaSelection.newBuilder().setLocation("europe-west1")))
              .addAsymmetricAutoscalingOptions(
                  AutoscalingConfig.AsymmetricAutoscalingOption.newBuilder()
                  .setReplicaSelection(ReplicaSelection.newBuilder().setLocation("europe-west4")))
              .addAsymmetricAutoscalingOptions(
                  AutoscalingConfig.AsymmetricAutoscalingOption.newBuilder()
                  .setReplicaSelection(ReplicaSelection.newBuilder().setLocation("asia-east1")))
              .build();
      Instance instance =
          Instance.newBuilder()
              .setAutoscalingConfig(autoscalingConfig)
              .setDisplayName(displayName)
              .setConfig(
                  InstanceConfigName.of(projectId, configId).toString())
              .build();

      // Creates a new instance
      System.out.printf("Creating instance %s.%n", instanceId);
      try {
        // Wait for the createInstance operation to finish.
        Instance instanceResult = instanceAdminClient.createInstanceAsync(
            CreateInstanceRequest.newBuilder()
                .setParent(ProjectName.of(projectId).toString())
                .setInstanceId(instanceId)
                .setInstance(instance)
                .build()).get();
        System.out.printf("Asymmetric Autoscaling instance %s was successfully created%n",
            instanceResult.getName());
      } catch (ExecutionException e) {
        System.out.printf(
            "Error: Creating instance %s failed with error message %s%n",
            instance.getName(), e.getMessage());
      } catch (InterruptedException e) {
        System.out.println("Error: Waiting for createInstance operation to finish was interrupted");
      }
    }
  }
}
```

### Create an instance without a default backup schedule

```java
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.admin.instance.v1.InstanceAdminClient;
import com.google.spanner.admin.instance.v1.CreateInstanceRequest;
import com.google.spanner.admin.instance.v1.Instance;
import com.google.spanner.admin.instance.v1.InstanceConfigName;
import com.google.spanner.admin.instance.v1.ProjectName;
import java.util.concurrent.ExecutionException;

class CreateInstanceWithoutDefaultBackupSchedulesExample {

  static void createInstanceWithoutDefaultBackupSchedules() {
    // TODO(developer): Replace these variables before running the sample.
    String projectId = "my-project";
    String instanceId = "my-instance";
    createInstanceWithoutDefaultBackupSchedules(projectId, instanceId);
  }

  static void createInstanceWithoutDefaultBackupSchedules(String projectId, String instanceId) {
    // Set Instance configuration.
    int nodeCount = 2;
    String displayName = "Descriptive name";

    // Create an Instance object that will be used to create the instance.
    Instance instance =
        Instance.newBuilder()
            .setDisplayName(displayName)
            .setDefaultBackupScheduleType(Instance.DefaultBackupScheduleType.NONE)
            .setNodeCount(nodeCount)
            .setConfig(InstanceConfigName.of(projectId, "regional-us-east4").toString())
            .build();

    try (Spanner spanner =
            SpannerOptions.newBuilder().setProjectId(projectId).build().getService();
        InstanceAdminClient instanceAdminClient = spanner.createInstanceAdminClient()) {

      // Wait for the createInstance operation to finish.
      Instance createdInstance =
          instanceAdminClient
              .createInstanceAsync(
                  CreateInstanceRequest.newBuilder()
                      .setParent(ProjectName.of(projectId).toString())
                      .setInstanceId(instanceId)
                      .setInstance(instance)
                      .build())
              .get();
      System.out.printf("Instance %s was successfully created%n", createdInstance.getName());
    } catch (ExecutionException e) {
      System.out.printf(
          "Error: Creating instance %s failed with error message %s%n",
          instance.getName(), e.getMessage());
    } catch (InterruptedException e) {
      System.out.println("Error: Waiting for createInstance operation to finish was interrupted");
    }
  }
}
```

### Node.js

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```javascript
// Imports the Google Cloud client library
const {Spanner, protos} = require('@google-cloud/spanner');

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

const instanceAdminClient = await spanner.getInstanceAdminClient();
/**
 * TODO(developer): Uncomment the following lines before running the sample.
 **/
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';

// Creates a new instance
try {
  console.log(
    `Creating instance ${instanceAdminClient.instancePath(
      projectId,
      instanceId,
    )}.`,
  );
  const [operation] = await instanceAdminClient.createInstance({
    instanceId: instanceId,
    parent: instanceAdminClient.projectPath(projectId),
    instance: {
      config: instanceAdminClient.instanceConfigPath(
        projectId,
        'regional-us-central1',
      ),
      nodeCount: 1,
      displayName: 'Display name for the instance.',
      labels: {
        cloud_spanner_samples: 'true',
        created: Math.round(Date.now() / 1000).toString(), // current time
      },
      edition:
        protos.google.spanner.admin.instance.v1.Instance.Edition.STANDARD, //optional
    },
  });

  console.log(`Waiting for operation on ${instanceId} to complete...`);
  await operation.promise();

  console.log(`Created instance ${instanceId}.`);
} catch (err) {
  console.error('ERROR:', err);
}
```

> **Note:** The old client library interface code samples for Node.js are archived in [GitHub](https://github.com/googleapis/nodejs-spanner/tree/main/samples/archived) .

### Create an instance with managed autoscaling using Node.js

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```javascript
// Imports the Google Cloud client library
const {Spanner, protos} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

// Get the instance admin client
const instanceAdminClient = spanner.getInstanceAdminClient();

const autoscalingConfig =
  protos.google.spanner.admin.instance.v1.AutoscalingConfig.create({
    // Only one of minNodes/maxNodes or minProcessingUnits/maxProcessingUnits can be set.
    autoscalingLimits:
      protos.google.spanner.admin.instance.v1.AutoscalingConfig.AutoscalingLimits.create(
        {
          minNodes: 1,
          maxNodes: 2,
        },
      ),
    // highPriorityCpuUtilizationPercent and storageUtilizationPercent are both
    // percentages and must lie between 0 and 100.
    autoscalingTargets:
      protos.google.spanner.admin.instance.v1.AutoscalingConfig.AutoscalingTargets.create(
        {
          highPriorityCpuUtilizationPercent: 65,
          storageUtilizationPercent: 95,
        },
      ),
  });

// Creates a new instance with autoscaling configuration
// When autoscalingConfig is enabled, nodeCount and processingUnits fields
// need not be specified.
try {
  console.log(
    `Creating instance ${instanceAdminClient.instancePath(
      projectId,
      instanceId,
    )}.`,
  );
  const [operation] = await instanceAdminClient.createInstance({
    instanceId: instanceId,
    parent: instanceAdminClient.projectPath(projectId),
    instance: {
      config: instanceAdminClient.instanceConfigPath(
        projectId,
        'regional-us-central1',
      ),
      displayName: 'Display name for the instance.',
      autoscalingConfig: autoscalingConfig,
      labels: {
        cloud_spanner_samples: 'true',
        created: Math.round(Date.now() / 1000).toString(), // current time
      },
      // Managed autoscaler is available only for ENTERPRISE edition
      edition:
        protos.google.spanner.admin.instance.v1.Instance.Edition.ENTERPRISE,
    },
  });

  console.log(`Waiting for operation on ${instanceId} to complete...`);
  await operation.promise();
  console.log(`Created instance ${instanceId}.`);

  // get instance metadata
  const [metadata] = await instanceAdminClient.getInstance({
    name: instanceAdminClient.instancePath(projectId, instanceId),
  });
  console.log(
    `Autoscaling configurations of ${instanceId} are:  ` +
      '\n' +
      `Min nodes: ${metadata.autoscalingConfig.autoscalingLimits.minNodes} ` +
      'nodes.' +
      '\n' +
      `Max nodes: ${metadata.autoscalingConfig.autoscalingLimits.maxNodes}` +
      ' nodes.' +
      '\n' +
      `High priority cpu utilization percent: ${metadata.autoscalingConfig.autoscalingTargets.highPriorityCpuUtilizationPercent}.` +
      '\n' +
      `Storage utilization percent: ${metadata.autoscalingConfig.autoscalingTargets.storageUtilizationPercent}.`,
  );
} catch (err) {
  console.error('ERROR:', err);
}
```

### Create an instance without a default backup schedule

```javascript
/**
 * TODO(developer): Uncomment the following lines before running the sample.
 **/
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';

// Imports the Google Cloud client library
const {Spanner, protos} = require('@google-cloud/spanner');

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

const instanceAdminClient = await spanner.getInstanceAdminClient();
// Creates a new instance
try {
  const [operation] = await instanceAdminClient.createInstance({
    instanceId: instanceId,
    parent: instanceAdminClient.projectPath(projectId),
    instance: {
      config: instanceAdminClient.instanceConfigPath(
        projectId,
        'regional-me-central2',
      ),
      nodeCount: 1,
      displayName: 'Display name for the instance.',
      labels: {
        cloud_spanner_samples: 'true',
        created: Math.round(Date.now() / 1000).toString(), // current time
      },
      defaultBackupScheduleType:
        protos.google.spanner.admin.instance.v1.Instance
          .DefaultBackupScheduleType.NONE,
    },
  });
  await operation.promise();

  console.log(
    `Created instance ${instanceId} without default backup schedules.`,
  );
} catch (err) {
  console.error('ERROR:', err);
}
```

### PHP

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```php
use Google\Cloud\Spanner\Admin\Instance\V1\Client\InstanceAdminClient;
use Google\Cloud\Spanner\Admin\Instance\V1\CreateInstanceRequest;
use Google\Cloud\Spanner\Admin\Instance\V1\Instance;

/**
 * Creates an instance.
 * Example:
 * ```
 * create_instance($projectId, $instanceId);
 * ```
 *
 * @param string $projectId  The Spanner project ID.
 * @param string $instanceId The Spanner instance ID.
 */
function create_instance(string $projectId, string $instanceId): void
{
    $instanceAdminClient = new InstanceAdminClient();
    $parent = InstanceAdminClient::projectName($projectId);
    $instanceName = InstanceAdminClient::instanceName($projectId, $instanceId);
    $configName = $instanceAdminClient->instanceConfigName($projectId, 'regional-us-central1');
    $instance = (new Instance())
        ->setName($instanceName)
        ->setConfig($configName)
        ->setDisplayName('dispName')
        ->setNodeCount(1);

    $operation = $instanceAdminClient->createInstance(
        (new CreateInstanceRequest())
        ->setParent($parent)
        ->setInstanceId($instanceId)
        ->setInstance($instance)
    );

    print('Waiting for operation to complete...' . PHP_EOL);
    $operation->pollUntilComplete();

    printf('Created instance %s' . PHP_EOL, $instanceId);
}
```

> **Note:** The old client library interface code samples for PHP are archived in [GitHub](https://github.com/GoogleCloudPlatform/php-docs-samples/tree/main/spanner/src/admin/archived) .

### Python

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```python
def create_instance(instance_id):
    """Creates an instance."""
    from google.cloud.spanner_admin_instance_v1.types import spanner_instance_admin

    spanner_client = spanner.Client()

    config_name = "{}/instanceConfigs/regional-us-central1".format(
        spanner_client.project_name
    )

    operation = spanner_client.instance_admin_api.create_instance(
        parent=spanner_client.project_name,
        instance_id=instance_id,
        instance=spanner_instance_admin.Instance(
            config=config_name,
            display_name="This is a display name.",
            node_count=1,
            labels={
                "cloud_spanner_samples": "true",
                "sample_name": "snippets-create_instance-explicit",
                "created": str(int(time.time())),
            },
            edition=spanner_instance_admin.Instance.Edition.STANDARD,  # Optional
        ),
    )

    print("Waiting for operation to complete...")
    operation.result(OPERATION_TIMEOUT_SECONDS)

    print("Created instance {}".format(instance_id))
```

> **Note:** The old client library interface code samples for Python are archived in [GitHub](https://github.com/googleapis/python-spanner/tree/main/samples/samples/archived) .

### Create an instance with managed autoscaling using Python

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```python
def create_instance_with_autoscaling_config(instance_id):
    """Creates a Cloud Spanner instance with an autoscaling configuration."""
    from google.cloud.spanner_admin_instance_v1.types import spanner_instance_admin

    spanner_client = spanner.Client()

    config_name = "{}/instanceConfigs/regional-us-central1".format(
        spanner_client.project_name
    )

    autoscaling_config = spanner_instance_admin.AutoscalingConfig(
        # Only one of minNodes/maxNodes or minProcessingUnits/maxProcessingUnits can be set.
        autoscaling_limits=spanner_instance_admin.AutoscalingConfig.AutoscalingLimits(
            min_nodes=1,
            max_nodes=2,
        ),
        # highPriorityCpuUtilizationPercent and storageUtilizationPercent are both
        # percentages and must lie between 0 and 100.
        autoscaling_targets=spanner_instance_admin.AutoscalingConfig.AutoscalingTargets(
            high_priority_cpu_utilization_percent=65,
            storage_utilization_percent=95,
        ),
    )

    #  Creates a new instance with autoscaling configuration
    #  When autoscalingConfig is enabled, nodeCount and processingUnits fields
    #  need not be specified.
    request = spanner_instance_admin.CreateInstanceRequest(
        parent=spanner_client.project_name,
        instance_id=instance_id,
        instance=spanner_instance_admin.Instance(
            config=config_name,
            display_name="This is a display name.",
            autoscaling_config=autoscaling_config,
            labels={
                "cloud_spanner_samples": "true",
                "sample_name": "snippets-create_instance_with_autoscaling_config",
                "created": str(int(time.time())),
            },
            edition=spanner_instance_admin.Instance.Edition.ENTERPRISE,  # Optional
        ),
    )

    operation = spanner_client.instance_admin_api.create_instance(request=request)

    print("Waiting for operation to complete...")
    instance = operation.result(OPERATION_TIMEOUT_SECONDS)

    print(
        "Created instance {} with {} autoscaling config".format(
            instance_id, instance.autoscaling_config
        )
    )
```

### Create an instance without a default backup schedule

```python
def create_instance_without_default_backup_schedules(instance_id):
    spanner_client = spanner.Client()
    config_name = "{}/instanceConfigs/regional-me-central2".format(
        spanner_client.project_name
    )

    operation = spanner_client.instance_admin_api.create_instance(
        parent=spanner_client.project_name,
        instance_id=instance_id,
        instance=spanner_instance_admin.Instance(
            config=config_name,
            display_name="This is a display name.",
            node_count=1,
            default_backup_schedule_type=spanner_instance_admin.Instance.DefaultBackupScheduleType.NONE,  # Optional
        ),
    )

    print("Waiting for operation to complete...")
    operation.result(OPERATION_TIMEOUT_SECONDS)

    print("Created instance {} without default backup schedules".format(instance_id))
```

### Ruby

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```ruby
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# instance_config_id = "Your Spanner InstanceConfig ID"

require "google/cloud/spanner"
require "google/cloud/spanner/admin/instance"

instance_admin_client = Google::Cloud::Spanner::Admin::Instance.instance_admin

project_path = instance_admin_client.project_path project: project_id
instance_path = instance_admin_client.instance_path project: project_id, instance: instance_id
instance_config_path = instance_admin_client.instance_config_path project: project_id, instance_config: instance_config_id

job = instance_admin_client.create_instance parent: project_path,
                                            instance_id: instance_id,
                                            instance: { name: instance_path,
                                                        config: instance_config_path,
                                                        display_name: instance_id,
                                                        node_count: 2,
                                                        labels: { cloud_spanner_samples: "true" } }

puts "Waiting for create instance operation to complete"

job.wait_until_done!

if job.error?
  puts job.error
else
  puts "Created instance #{instance_id}"
end
```

## List instances

You can show a list of your Spanner instances.

### Console

Go to the **Spanner Instances** page in the Google Cloud console.

The Google Cloud console shows a list of your Spanner instances, along with each instance's ID, display name, configuration, and [compute capacity](https://docs.cloud.google.com/spanner/docs/compute-capacity) expressed in both processing units and in nodes.

### gcloud

Use the [`gcloud spanner instances list`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/list) command:

```
gcloud spanner instances list
```

The gcloud CLI prints a list of your Spanner instances, along with each instance's ID, display name, configuration, and compute capacity.

## Edit an instance

The following sections explain how to upgrade the edition of your instance, and change an instance's display name, compute capacity, and default backup schedule type. You can't change the instance ID or instance configuration (however, you can [move your instance](https://docs.cloud.google.com/spanner/docs/move-instance) ).

### Upgrade the edition

You can upgrade your Standard edition instances to a higher-tier edition. Standard edition instances can be upgraded to the Enterprise edition or Enterprise Plus edition as long as they have not been prevented by an [organization policy constraint](https://docs.cloud.google.com/spanner/docs/spanner-custom-constraints) . Enterprise edition instances can be upgraded to the Enterprise Plus edition. The edition upgrade takes approximately 10 minutes to complete with zero downtime.

### Console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Click the name of the instance that you want to upgrade.

3.  Click **Upgrade** next to the edition type.

4.  In the **Edition instance** page, and under **Update edition** , select the new higher-tier edition for your instance.

5.  Click **Save** .

### gcloud

Use the [`gcloud spanner instances update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/update) command to upgrade your instance's edition:

```
gcloud spanner instances update INSTANCE_ID --edition=EDITION \
[--async]
```

Replace the following:

- ` INSTANCE_ID ` : the permanent identifier for the instance.
- ` EDITION ` : specify the new higher-tier edition for your instance. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

Optional flags:

- `--async` : Use this flag if you want your request to return immediately, without waiting for the operation in progress to complete. You can check the status of your request by running [`gcloud spanner operations describe`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/operations/describe) .

### Go

```go
import (
    "context"
    "fmt"
    "io"

    instance "cloud.google.com/go/spanner/admin/instance/apiv1"
    "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
    "google.golang.org/genproto/protobuf/field_mask"
)

func updateInstance(w io.Writer, projectID, instanceID string) error {
    // projectID := "my-project-id"
    // instanceID := "my-instance"
    ctx := context.Background()
    instanceAdmin, err := instance.NewInstanceAdminClient(ctx)
    if err != nil {
        return err
    }
    defer instanceAdmin.Close()

    req := &instancepb.UpdateInstanceRequest{
        Instance: &instancepb.Instance{
            Name: fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID),
            // The edition selected for this instance.
            // Different editions provide different capabilities at different price points.
            // For more information, see https://cloud.google.com/spanner/docs/editions-overview.
            Edition: instancepb.Instance_ENTERPRISE,
        },
        FieldMask: &field_mask.FieldMask{
            Paths: []string{"edition"},
        },
    }
    op, err := instanceAdmin.UpdateInstance(ctx, req)
    if err != nil {
        return fmt.Errorf("could not update instance %s: %w", fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID), err)
    }
    // Wait for the instance update to finish.
    _, err = op.Wait(ctx)
    if err != nil {
        return fmt.Errorf("waiting for instance update to finish failed: %w", err)
    }

    fmt.Fprintf(w, "Updated instance [%s]\n", instanceID)
    return nil
}
```

### Java

```java
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.admin.instance.v1.InstanceAdminClient;
import com.google.common.collect.Lists;
import com.google.protobuf.FieldMask;
import com.google.spanner.admin.instance.v1.Instance;
import com.google.spanner.admin.instance.v1.InstanceConfigName;
import com.google.spanner.admin.instance.v1.InstanceName;
import com.google.spanner.admin.instance.v1.UpdateInstanceRequest;
import java.util.concurrent.ExecutionException;

public class UpdateInstanceExample {

  static void updateInstance() {
    // TODO(developer): Replace these variables before running the sample.
    String projectId = "my-project";
    String instanceId = "my-instance";
    updateInstance(projectId, instanceId);
  }

  static void updateInstance(String projectId, String instanceId) {
    // Set Instance configuration.
    int nodeCount = 2;
    String displayName = "Updated name";

    // Update an Instance object that will be used to update the instance.
    Instance instance =
        Instance.newBuilder()
            .setName(InstanceName.of(projectId, instanceId).toString())
            .setDisplayName(displayName)
            .setNodeCount(nodeCount)
            .setEdition(Instance.Edition.ENTERPRISE)
            .setConfig(InstanceConfigName.of(projectId, "regional-us-east4").toString())
            .build();

    try (Spanner spanner =
            SpannerOptions.newBuilder().setProjectId(projectId).build().getService();
        InstanceAdminClient instanceAdminClient = spanner.createInstanceAdminClient()) {

      // Wait for the updatedInstance operation to finish.
      Instance updatedInstance =
          instanceAdminClient
              .updateInstanceAsync(
                  UpdateInstanceRequest.newBuilder()
                      .setFieldMask(
                          FieldMask.newBuilder().addAllPaths(Lists.newArrayList("edition")))
                      .setInstance(instance)
                      .build())
              .get();
      System.out.printf("Instance %s was successfully updated%n", updatedInstance.getName());
    } catch (ExecutionException e) {
      System.out.printf(
          "Error: Updating instance %s failed with error message %s%n",
          instance.getName(), e.getMessage());
    } catch (InterruptedException e) {
      System.out.println("Error: Waiting for updateInstance operation to finish was interrupted");
    }
  }
}
```

### Node.js

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```javascript
// Imports the Google Cloud client library
const {Spanner, protos} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

const instanceAdminClient = spanner.getInstanceAdminClient();

// Updates an instance
try {
  console.log(
    `Updating instance ${instanceAdminClient.instancePath(
      projectId,
      instanceId,
    )}.`,
  );
  const [operation] = await instanceAdminClient.updateInstance({
    instance: {
      name: instanceAdminClient.instancePath(projectId, instanceId),
      labels: {
        updated: 'true',
        created: Math.round(Date.now() / 1000).toString(), // current time
      },
      edition:
        protos.google.spanner.admin.instance.v1.Instance.Edition.ENTERPRISE, //optional
    },
    // Field mask specifying fields that should get updated in an Instance
    fieldMask: (protos.google.protobuf.FieldMask = {
      paths: ['labels', 'edition'],
    }),
  });

  console.log(`Waiting for operation on ${instanceId} to complete...`);
  await operation.promise();
  console.log(`Updated instance ${instanceId}.`);
  const [metadata] = await instanceAdminClient.getInstance({
    name: instanceAdminClient.instancePath(projectId, instanceId),
  });
  console.log(
    `Instance ${instanceId} has been updated with the ${metadata.edition} ` +
      'edition.',
  );
} catch (err) {
  console.error('ERROR:', err);
}
```

### Python

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```python
def update_instance(instance_id):
    """Updates an instance."""
    from google.cloud.spanner_admin_instance_v1.types import spanner_instance_admin

    spanner_client = spanner.Client()

    name = "{}/instances/{}".format(spanner_client.project_name, instance_id)

    operation = spanner_client.instance_admin_api.update_instance(
        instance=spanner_instance_admin.Instance(
            name=name,
            labels={
                "sample_name": "snippets-update_instance-explicit",
            },
            edition=spanner_instance_admin.Instance.Edition.ENTERPRISE,  # Optional
        ),
        field_mask=field_mask_pb2.FieldMask(paths=["labels", "edition"]),
    )

    print("Waiting for operation to complete...")
    operation.result(900)

    print("Updated instance {}".format(instance_id))
```

### Downgrade the edition

You can downgrade your Spanner instances to a lower-tier edition. You must stop using the higher-tier edition features in order to downgrade. Enterprise edition instances can be downgraded to the Standard edition. Enterprise Plus edition instances can be downgraded to the Enterprise edition or Standard edition. The edition downgrade takes approximately 10 minutes to complete with zero downtime.

To monitor the usage of Enterprise edition and Enterprise Plus edition edition features in your instance, use the [Feature usage](https://docs.cloud.google.com/monitoring/api/metrics_gcp_p_z#gcp-spanner) ( `instance/edition/feature_usage` ) monitoring metric. For more information, see [Monitor edition usage](https://docs.cloud.google.com/spanner/docs/editions-overview#monitor-edition-usage)

### Console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Click the name of the instance that you want to downgrade.

3.  In the **Instance overview** page, click **Edit instance** .

4.  On the **Edit instance** page under **Choose a Spanner edition** , select the lower-tier edition.

5.  Click **Update instance** .

### gcloud

Use the [`gcloud spanner instances update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/update) command to downgrade your instance's edition:

```
gcloud spanner instances update INSTANCE_ID --edition=EDITION
```

Replace the following:

- ` INSTANCE_ID ` : the permanent identifier for the instance.
- ` EDITION ` : specify the new lower-tier edition for your instance. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

### Change the display name

### Console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Click the name of the instance that you want to rename.

3.  Click **Edit instance** .

4.  Enter a new instance name. This name must be unique within the Google Cloud project.

5.  Click **Save** .

### gcloud

Use the [`gcloud spanner instances update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/update) command:

```
gcloud spanner instances update INSTANCE_ID --description=INSTANCE_NAME
```

Replace the following:

- ` INSTANCE_ID ` : the permanent identifier for the instance.
- ` INSTANCE_NAME ` : the name to display for the instance in the Google Cloud console. The instance name must be unique within your Google Cloud project.

### Change the compute capacity

You must provision enough [compute capacity](https://docs.cloud.google.com/spanner/docs/compute-capacity) to keep [CPU utilization](https://docs.cloud.google.com/spanner/docs/cpu-utilization#recommended-max) and [storage utilization](https://docs.cloud.google.com/spanner/docs/storage-utilization) below the recommended maximums. For more information, see the [quotas and limits](https://docs.cloud.google.com/spanner/quotas) for Spanner.

You can reduce the compute capacity of a Spanner instance except in the following scenarios:

- You can't store more than 10 TiB of data per node (1000 processing units).

- There are a large number of [splits](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#database-splits) for your instance's data. In this scenario, Spanner might not be able to manage the splits after you reduce compute capacity. You can try reducing the compute capacity by progressively smaller amounts until you find the minimum capacity that Spanner needs to manage all of the instance's splits.

  Spanner can create a large number of splits to accommodate your usage patterns. If your usage patterns change, then after one or two weeks, Spanner might merge some splits together and you can try to reduce the instance's compute capacity.

When removing compute capacity, monitor your CPU utilization and request latencies in [Cloud Monitoring](https://docs.cloud.google.com/spanner/docs/monitoring-cloud) to ensure CPU utilization stays under 65% for regional instances and 45% for each region in multi-region instances. You might experience a temporary increase in request latencies while removing compute capacity.

If you want to increase the compute capacity of an instance, your Google Cloud project must have sufficient quota to add the compute capacity. The time it takes for the increase request to complete depends on the size of the request. In most cases, requests complete within a few minutes. On rare occasions, a scale up might take up to an hour to complete.

### Console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Click the name of the instance that you want to change.

3.  Click **Edit Instance** .

4.  Change the compute capacity by choosing the measurement units (processing units or nodes) and then entering a quantity. When using processing units, enter quantities up to 1000 in multiples of 100 (100, 200, 300 and so on) and enter greater quantities in multiples of 1000 (1000, 2000, 3000 and so on). Each node equals 1000 processing units.

5.  Click **Save** .

If you see a dialog that says you have insufficient quota to add compute capacity in this location, follow the instructions to request a higher quota.

### gcloud

Use the [`gcloud spanner instances update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/update) command. When using this command, specify the [compute capacity](https://docs.cloud.google.com/spanner/docs/compute-capacity) as a number of nodes or processing units.

```
gcloud spanner instances update INSTANCE_ID --nodes=NODE_COUNT
[--async]
```

or

```
gcloud spanner instances update INSTANCE_ID
--processing-units=PROCESSING_UNIT_COUNT [--async]
```

Replace the following:

- ` INSTANCE_ID ` : the permanent identifier for the instance.
- ` NODE_COUNT ` : the compute capacity of the instance, expressed as a number of nodes. Each node equals 1000 processing units.
- ` PROCESSING_UNIT_COUNT ` : the compute capacity of the instance, expressed as a number of processing units. Enter quantities up to 1000 in multiples of 100 (100, 200, 300 and so on) and enter greater quantities in multiples of 1000 (1000, 2000, 3000 and so on).

Optional flags:

- `--async` : Use this flag if you want your request to return immediately, without waiting for the operation in progress to complete. You can check the status of your request by running [`gcloud spanner operations describe`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/operations/describe) .

### Enable or modify managed autoscaler on an instance

You can enable or modify autoscaling on a Spanner instance using the Google Cloud console, the [gcloud CLI](https://cloud.google.com/sdk/docs/install) , or the [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) . The following limitations apply when you add or change the managed autoscaling feature on an existing instance:

- Managed autoscaler is only available in the Enterprise edition or Enterprise Plus edition.

- You can't enable the managed autoscaler on an instance that you are moving.

- You can't [move an instance](https://docs.cloud.google.com/spanner/docs/create-manage-instances#move-instance) while the managed autoscaler is enabled.

### Console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Click the name of the instance that you want to enable the managed autoscaler on.

3.  Click **Edit instance** .

4.  Under **Configure compute capacity** , click **Autoscaling** .

5.  For **Minimum** , select the minimum limit to use when scaling down. For more information, see [Determine the minimum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-minimum) .

6.  For **Maximum** , select the maximum limit to use when scaling up. For more information, see [Determine the maximum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-maximum) .

7.  For **High priority CPU utilization target** , select the percentage of CPU to use for high priority tasks. For more information, see [Determine the CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-high-priority-cpu) .

8.  For **Total CPU utilization target** , select the target CPU percentage to use for all [low, medium, and high priority tasks](https://docs.cloud.google.com/spanner/docs/cpu-utilization#task-priority) . The CPU percentage can range from 10 to 90%. For more information, see [Determine the total CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-total-cpu) .

9.  For **Storage utilization target** , select the percentage of storage to use. For more information, see [Determine the storage utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-storage) .

10. Optional: If you select **Autoscaling** as the scaling mode, then you can click the **Show asymmetric autoscaling options** dropdown to autoscale your read-only replicas independently from other replicas.

    1.  Select the read-only replica you want to asymmetrically autoscale.

    2.  Configure the following autoscaler options:

        - **Minimum** indicates the minimum limit to scale down to, depending on the measurement unit that you choose for **Compute capacity** . For more information, see [Determine the minimum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-minimum) .
        - **Maximum** indicates the maximum limit to scale up to, depending on the measurement unit that you choose for **Compute capacity** . For more information, see [Determine the maximum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-maximum) .
        - **High priority CPU utilization target** indicates the target percentage of CPU to use for high priority tasks. For more information, see [Determine the high priority CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-high-priority-cpu) .
        - **Total CPU utilization target** indicates the target percentage of CPU to use for all [low, medium, and high priority tasks](https://docs.cloud.google.com/spanner/docs/cpu-utilization#task-priority) . The CPU percentage can range from 10 to 90%. For more information, see [Determine the total CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-total-cpu) .

11. Click **Save** .

### gcloud

Use the [`gcloud spanner instances update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/update) command to add the managed autoscaler to an instance. For more information and limitations, see [`Google Cloud CLI` flags and limitations](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#flags_and_limitations) .

You can add the managed autoscaler with the following command:

```
  gcloud spanner instances update INSTANCE_ID \
    --autoscaling-min-processing-units=MINIMUM_PROCESSING_UNITS \
    --autoscaling-max-processing-units=MAXIMUM_PROCESSING_UNITS \
    --autoscaling-high-priority-cpu-target=HIGH_PRIORITY_CPU_PERCENTAGE \
    --autoscaling-total-cpu-target=TOTAL_CPU_PERCENTAGE \
    --autoscaling-storage-target=STORAGE_PERCENTAGE \
    [--disable-downscaling | --no-disable-downscaling] \
    [--asymmetric-autoscaling-option \
       location=ASYMMETRIC_AUTOSCALING_LOCATION,\
       min_nodes=ASYMMETRIC_AUTOSCALING_MIN,\
       max_nodes=ASYMMETRIC_AUTOSCALING_MAX,\
       high_priority_cpu_target=ASYMMETRIC_HIGH_PRIORITY_CPU_TARGET,\
       total_cpu_target=ASYMMETRIC_TOTAL_CPU_TARGET,\
       disable_total_cpu_autoscaling=TRUE|FALSE
       | disable_high_priority_cpu_autoscaling=TRUE|FALSE]
```

or

```
  gcloud spanner instances update INSTANCE_ID \
    --autoscaling-min-processing-units=MINIMUM_NODES \
    --autoscaling-max-processing-units=MAXIMUM_NODES \
    --autoscaling-high-priority-cpu-target=HIGH_PRIORITY_CPU_PERCENTAGE \
    --autoscaling-total-cpu-target=TOTAL_CPU_PERCENTAGE \
    --autoscaling-storage-target=STORAGE_PERCENTAGE \
    [--disable-downscaling | --no-disable-downscaling] \
    [--asymmetric-autoscaling-option \
       location=ASYMMETRIC_AUTOSCALING_LOCATION,\
       min_nodes=ASYMMETRIC_AUTOSCALING_MIN,\
       max_nodes=ASYMMETRIC_AUTOSCALING_MAX,\
       high_priority_cpu_target=ASYMMETRIC_HIGH_PRIORITY_CPU_TARGET,\
       total_cpu_target=ASYMMETRIC_TOTAL_CPU_TARGET,\
       disable_total_cpu_autoscaling=TRUE|FALSE
       | disable_high_priority_cpu_autoscaling=TRUE|FALSE]
```

Replace the following:

- ` INSTANCE_ID ` : the permanent identifier for the instance.
- ` MINIMUM_PROCESSING_UNITS ` , ` MINIMUM_NODES ` : the minimum number of processing units or nodes to use when scaling down. For more information, see [Determine the minimum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-minimum) .
- ` MAXIMUM_PROCESSING_UNITS ` , ` MAXIMUM_NODES ` : the maximum number of processing units or nodes to use when scaling up. For more information, see [Determine the maximum limit](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-maximum) .
- ` HIGH_PRIORITY_CPU_PERCENTAGE ` : the target percentage of high priority CPU to use, based on the [priority of the task](https://docs.cloud.google.com/spanner/docs/cpu-utilization#task-priority) . The CPU percentage can range from 10 to 90%. If you're optimizing for cost, then use a higher percentage. For more information, see [Determine the high priority CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-high-priority-cpu) .
- ` TOTAL_CPU_PERCENTAGE ` : the target percentage of total priority CPU to use. The total CPU target has to be greater than the high priority CPU target. The CPU percentage can range from 10 to 90%. If you're optimizing for cost, then use a higher percentage. For more information, see [Determine the total CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-total-cpu) .
- ` STORAGE_PERCENTAGE ` : the target percentage of storage to use, from 10% to 99%. For more information, see [Determine the storage utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-storage) .

Optional flags:

- `--disable-downscaling` : use this boolean flag to disable downscaling. When set to `TRUE` , it prevents the autoscaler from reducing the number of nodes or processing units. Upscaling continues to function normally to meet increased demand. To enable downscaling after disabling it, use the `--no-disable-downscaling` flag.

- `--asymmetric-autoscaling-option` : Use this flag to enable [asymmetric autoscaling](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#asymmetric-read-only-autoscaling) .

  Replace the following parameters:

  - ` ASYMMETRIC_AUTOSCALING_LOCATION ` : if the asymmetric autoscaling flag is used, then this parameter is required. The location of the read-only region that you want to scale asymmetrically.
  - ` ASYMMETRIC_AUTOSCALING_MIN ` : optional parameter. The minimum number of nodes when scaling down. If not specified, this value is inherited from the base instance configuration.
  - ` ASYMMETRIC_AUTOSCALING_MAX ` : optional parameter. The maximum number of nodes when scaling up. If not specified, this value is inherited from the base instance configuration.
  - ` ASYMMETRIC_HIGH_PRIORITY_CPU_TARGET ` : optional parameter. The target percentage of CPU to use, based on the [priority of the task](https://docs.cloud.google.com/spanner/docs/cpu-utilization#task-priority) . If not specified, this value is inherited from the base instance configuration. If you're optimizing for cost, then use a higher percentage. Replicas can have a different target percentage than the base instance and from other replicas, but for failover scenarios, we recommend that replicas use consistent targets across different replicas. For more information, see [Failover concerns](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#failover-concerns) .
  - ` ASYMMETRIC_TOTAL_CPU_TARGET ` : optional parameter. Set the target from 10 to 90% for the [total CPU](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#determine-total-cpu) . If not specified, this value is inherited from the base instance configuration. If you're optimizing for cost, then use a higher percentage. Replicas can have a different target percentage than the base instance and from other replicas, but for failover scenarios, we recommend that replicas use consistent targets across different replicas. For more information, see [Asymmetric read-only autoscaling](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#asymmetric-read-only-autoscaling) .
  - `disable_total_cpu_autoscaling` : set this parameter to `TRUE` if you don't want the autoscaler to autoscale total CPU on replicas. Don't specify this parameter if you want the instance to inherit the `total_cpu_autoscaling` target from the base instance. You can't disable both total CPU and high priority CPU on the same replica.
  - `disable_high_priority_cpu_autoscaling` : set the `disable_high_priority_cpu_autoscaling` to `TRUE` if you don't want the autoscaler to autoscale the high priority CPU on replicas. Don't specify this parameter if you want the instance to inherit the `high_priority_cpu_autoscaling` target from the base instance. You can't disable both total CPU and high priority CPU on the same replica.

After you add the managed autoscaler to an instance, you can also modify the managed autoscaler settings. For example, if you want to increase the maximum number of processing units to 10000, run the following command:

```
gcloud spanner instances update test-instance \
     --autoscaling-max-processing-units=10000
```

### Change an instance from using managed autoscaler to manual scaling

You can change whether a Spanner instance uses manual or managed scaling by using the Google Cloud console, the [gcloud CLI](https://cloud.google.com/sdk/docs/install) , or the [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

### Console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Click the name of the instance that you want to disable managed autoscaler on.

3.  Under **Choose a scaling mode** , click **Manual allocation** .

4.  Click **Save** .

### gcloud

Use the [`gcloud spanner instances update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/update) command to update the instance.

Use the following command to change an instance from using the managed autoscaler to manual scaling:

```
  gcloud spanner instances update INSTANCE_ID \
  --processing-units=PROCESSING_UNIT_COUNT
```

or

```
  gcloud spanner instances update INSTANCE_ID \
  --nodes=NODE_COUNT
```

Replace the following:

- ` INSTANCE_ID ` : the permanent identifier for the instance.
- ` NODE_COUNT ` : the compute capacity of the instance, expressed as a number of nodes. Each node equals 1000 processing units.
- ` PROCESSING_UNIT_COUNT ` : the compute capacity of the instance, expressed as a number of processing units. Enter quantities up to 1000 in multiples of 100 (100, 200, 300 and so on) and enter greater quantities in multiples of 1000 (1000, 2000, 3000 and so on).

### Label an instance

Labels help organize your resources.

### Console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Select the checkbox for the instance. The **Info panel** appears on the right-hand side of the page.

3.  Click the **Labels** tab in the **Info panel** . You can then add, delete or update labels for the Spanner instance.

### Edit the default backup schedule type

Default backup schedules are automatically enabled for all new instances. You can enable or disable default backup schedules in an instance when creating the instance or by editing the instance later. For more information, see [Default backup schedules](https://docs.cloud.google.com/spanner/docs/backup#default-backup-schedules) .

### Console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

<!-- -->

1.  Click the name of the instance that you want to edit the default backup schedule.
2.  Click **Edit instance** .
3.  Under **Backups** , the **Enable default backup schedules** checkbox determines whether default backup schedules are enabled or not. When enabled, all new databases in this instance have a default backup schedule created.
4.  Click **Save** .

### gcloud

Use the [`gcloud spanner instances update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/update) command to edit the default backup schedules type.

You can edit the default backup schedule type by running the following command:

```
gcloud spanner instances update INSTANCE_ID \
    --default-backup-schedule-type=DEFAULT_BACKUP_SCHEDULE_TYPE
```

Replace the following:

- ` INSTANCE_ID ` : the permanent identifier for the instance.

- ` DEFAULT_BACKUP_SCHEDULE_TYPE ` : the default backup schedule type that is used in the instance. Must be one of the following values:

  - `AUTOMATIC` : a default backup schedule is created automatically when a new database is created in the instance. The default backup schedule creates a full backup every 24 hours. These full backups are retained for 7 days. You can edit or delete the default backup schedule once it's created.
  - `NONE` : a default backup schedule isn't created automatically when a new database is created in the instance.

### C#

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```csharp
using Google.Cloud.Spanner.Admin.Instance.V1;
using Google.Cloud.Spanner.Common.V1;
using Google.LongRunning;
using Google.Protobuf.WellKnownTypes;
using System;
using System.Threading.Tasks;

public class UpdateInstanceDefaultBackupScheduleTypeAsyncSample
{
    public async Task<Instance> UpdateInstanceDefaultBackupScheduleTypeAsync(string projectId, string instanceId)
    {
        // Create the InstanceAdminClient instance.
        InstanceAdminClient instanceAdminClient = await InstanceAdminClient.CreateAsync();

        // Initialize request parameters.
        Instance instance = new Instance
        {
            InstanceName = InstanceName.FromProjectInstance(projectId, instanceId),
            DefaultBackupScheduleType = Instance.Types.DefaultBackupScheduleType.Automatic,
        };
        FieldMask mask = new FieldMask 
        {
            Paths = { "default_backup_schedule_type" }
        };

        // Make the CreateInstance request.
        Operation<Instance, UpdateInstanceMetadata> response =
            await instanceAdminClient.UpdateInstanceAsync(instance, mask);

        Console.WriteLine("Waiting for the operation to finish.");

        // Poll until the returned long-running operation is complete.
        Operation<Instance, UpdateInstanceMetadata> completedResponse =
            await response.PollUntilCompletedAsync();

        if (completedResponse.IsFaulted)
        {
            Console.WriteLine($"Error while updating instance: {completedResponse.Exception}");
            throw completedResponse.Exception;
        }

        Console.WriteLine($"Instance updated successfully.");
        return completedResponse.Result;
    }
}
```

### Go

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```go
import (
    "context"
    "fmt"
    "io"

    instance "cloud.google.com/go/spanner/admin/instance/apiv1"
    "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
    "google.golang.org/genproto/protobuf/field_mask"
)

// updateInstanceDefaultBackupScheduleType updates instance default backup schedule type to AUTOMATIC.
// This means a default backup schedule will be created automatically on creation of a database within the instance.
func updateInstanceDefaultBackupScheduleType(w io.Writer, projectID, instanceID string) error {
    // projectID := "my-project-id"
    // instanceID := "my-instance"
    ctx := context.Background()
    instanceAdmin, err := instance.NewInstanceAdminClient(ctx)
    if err != nil {
        return err
    }
    defer instanceAdmin.Close()

    // Updates the default backup schedule type field of an instance.  The field mask is required to
    // indicate which field is being updated.
    req := &instancepb.UpdateInstanceRequest{
        Instance: &instancepb.Instance{
            Name: fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID),
            // Controls the default backup behavior for new databases within the instance.
            DefaultBackupScheduleType: instancepb.Instance_AUTOMATIC,
        },
        FieldMask: &field_mask.FieldMask{
            Paths: []string{"default_backup_schedule_type"},
        },
    }
    op, err := instanceAdmin.UpdateInstance(ctx, req)
    if err != nil {
        return fmt.Errorf("could not update instance %s: %w", fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID), err)
    }
    // Wait for the instance update to finish.
    _, err = op.Wait(ctx)
    if err != nil {
        return fmt.Errorf("waiting for instance update to finish failed: %w", err)
    }

    fmt.Fprintf(w, "Updated instance [%s]\n", instanceID)
    return nil
}
```

### Java

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```java
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.admin.instance.v1.InstanceAdminClient;
import com.google.common.collect.Lists;
import com.google.protobuf.FieldMask;
import com.google.spanner.admin.instance.v1.Instance;
import com.google.spanner.admin.instance.v1.InstanceConfigName;
import com.google.spanner.admin.instance.v1.InstanceName;
import com.google.spanner.admin.instance.v1.UpdateInstanceRequest;
import java.util.concurrent.ExecutionException;

public class UpdateInstanceDefaultBackupScheduleTypeExample {

  static void updateInstanceDefaultBackupScheduleType() {
    // TODO(developer): Replace these variables before running the sample.
    String projectId = "my-project";
    String instanceId = "my-instance";
    updateInstanceDefaultBackupScheduleType(projectId, instanceId);
  }

  static void updateInstanceDefaultBackupScheduleType(String projectId, String instanceId) {
    // Set Instance configuration.
    int nodeCount = 2;
    String displayName = "Updated name";

    // Update an Instance object that will be used to update the instance.
    Instance instance =
        Instance.newBuilder()
            .setName(InstanceName.of(projectId, instanceId).toString())
            .setDisplayName(displayName)
            .setNodeCount(nodeCount)
            .setDefaultBackupScheduleType(Instance.DefaultBackupScheduleType.AUTOMATIC)
            .setConfig(InstanceConfigName.of(projectId, "regional-us-east4").toString())
            .build();

    try (Spanner spanner =
            SpannerOptions.newBuilder().setProjectId(projectId).build().getService();
        InstanceAdminClient instanceAdminClient = spanner.createInstanceAdminClient()) {

      // Wait for the updatedInstance operation to finish.
      Instance updatedInstance =
          instanceAdminClient
              .updateInstanceAsync(
                  UpdateInstanceRequest.newBuilder()
                      .setFieldMask(
                          FieldMask.newBuilder()
                              .addAllPaths(Lists.newArrayList("default_backup_schedule_type")))
                      .setInstance(instance)
                      .build())
              .get();
      System.out.printf("Instance %s was successfully updated%n", updatedInstance.getName());
    } catch (ExecutionException e) {
      System.out.printf(
          "Error: Updating instance %s failed with error message %s%n",
          instance.getName(), e.getMessage());
    } catch (InterruptedException e) {
      System.out.println("Error: Waiting for updateInstance operation to finish was interrupted");
    }
  }
}
```

### Node.js

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```javascript
/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';

// Imports the Google Cloud client library
const {Spanner, protos} = require('@google-cloud/spanner');

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});
const instanceAdminClient = await spanner.getInstanceAdminClient();

// Updates an instance
try {
  const [operation] = await instanceAdminClient.updateInstance({
    instance: {
      name: instanceAdminClient.instancePath(projectId, instanceId),
      defaultBackupScheduleType:
        protos.google.spanner.admin.instance.v1.Instance
          .DefaultBackupScheduleType.AUTOMATIC, // optional
    },
    // Field mask specifying fields that should get updated in an Instance
    fieldMask: (protos.google.protobuf.FieldMask = {
      paths: ['default_backup_schedule_type'],
    }),
  });

  await operation.promise();
  const [metadata] = await instanceAdminClient.getInstance({
    name: instanceAdminClient.instancePath(projectId, instanceId),
  });
  console.log(
    `Instance ${instanceId} has been updated with the ${metadata.defaultBackupScheduleType}` +
      ' default backup schedule type.',
  );
} catch (err) {
  console.error('ERROR:', err);
}
```

### Python

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```python
def update_instance_default_backup_schedule_type(instance_id):
    spanner_client = spanner.Client()

    name = "{}/instances/{}".format(spanner_client.project_name, instance_id)

    operation = spanner_client.instance_admin_api.update_instance(
        instance=spanner_instance_admin.Instance(
            name=name,
            default_backup_schedule_type=spanner_instance_admin.Instance.DefaultBackupScheduleType.AUTOMATIC,  # Optional
        ),
        field_mask=field_mask_pb2.FieldMask(paths=["default_backup_schedule_type"]),
    )

    print("Waiting for operation to complete...")
    operation.result(OPERATION_TIMEOUT_SECONDS)

    print("Updated instance {} to have default backup schedules".format(instance_id))
```

## Move an instance

> **Note:** You can't move an instance that has managed autoscaler enabled.

For instructions on how to move your instance from any instance configuration to any other instance configuration, including between regional and multi-regional configurations, see [Move an instance](https://docs.cloud.google.com/spanner/docs/move-instance#how-instance-move) .

## Delete an instance

> **Warning:** Deleting an instance permanently removes the instance and all its databases. You can't undo this later. Also, you can't create another free trial instance once you've deleted your first free trial instance. You can create one free trial instance per project lifecycle.

You can delete an instance with the Google Cloud console or the [Google Cloud CLI](https://docs.cloud.google.com/spanner/docs/gcloud-spanner) .

If you want to delete an instance that has one or more databases with deletion protection enabled, you must first [disable the deletion protection](https://docs.cloud.google.com/spanner/docs/prevent-database-deletion#disable) on all databases in that instance before you can delete the instance.

### Console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Click the name of the instance that you want to delete.

3.  Click **Delete instance** .

4.  Follow the instructions to confirm that you want to delete the instance.

5.  Click **Delete** .

### gcloud

Use the [`gcloud spanner instances delete`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/list) command, replacing ` INSTANCE_ID ` with the instance ID:

```
gcloud spanner instances delete INSTANCE_ID
```

## Stop or restart an instance

Spanner is a fully managed database service which oversees its own underlying tasks and resources, including monitoring and restarting processes when necessary with zero downtime. As there is no need to manually stop or restart a given instance, Spanner does not offer a way to do so.

## What's next

- Learn how to insert, update, and delete data with [Data Manipulation Language (DML)](https://docs.cloud.google.com/spanner/docs/dml-tasks) or the [gcloud CLI](https://docs.cloud.google.com/spanner/docs/modify-gcloud) .
- Grant [Identity and Access Management roles](https://docs.cloud.google.com/spanner/docs/grant-permissions) for the instance and its databases.
- Understand how to [design a Spanner schema](https://docs.cloud.google.com/spanner/docs/schema-design) .
