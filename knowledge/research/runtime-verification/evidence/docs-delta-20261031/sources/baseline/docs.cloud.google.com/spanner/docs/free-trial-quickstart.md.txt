---
name: documents/docs.cloud.google.com/spanner/docs/free-trial-quickstart
uri: https://docs.cloud.google.com/spanner/docs/free-trial-quickstart
title: Create a Spanner free trial instance and sample application
description: Manage a Spanner free trial instance including, create a database, schema, adding data, and running queries using the {{dynamic_data.site_values.cloud_name_short}} console.
data_source: docs.cloud.google.com
---

> **Note:** A Spanner free trial instance supports Standard edition features, and Enterprise edition features, such as [KNN vector distance functions](https://docs.cloud.google.com/spanner/docs/find-k-nearest-neighbors) , [full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) , and [Spanner Graph](https://docs.cloud.google.com/spanner/docs/graph/overview) . For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

This quickstart shows you how to use a Spanner free trial instance and sample application. For more information, see the [Spanner free trial instances overview](https://docs.cloud.google.com/spanner/docs/free-trial-instance) .

## Get started with a free trial instance

A Spanner 90-day free trial instance is available to anyone with a Google Account who has Cloud Billing enabled in their project. You aren't charged unless you choose to [upgrade](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart#upgrade) your free trial instance to a paid instance.

You can create a free trial instance using the Google Cloud console or Google Cloud CLI.

> **Note:** You can only create one free trial instance per project lifecycle, and a maximum of five free trial instances per Cloud Billing account.

## Before you begin

### Google Cloud console

1.  In the Google Cloud console, go to the **Spanner** page.

2.  Select or create a Google Cloud project if you haven't done so already.

3.  If Cloud Billing is already enabled for your project, proceed to [Create free trial instance](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart#create) in the next section.

    If Cloud Billing is not enabled for your project, link an existing Cloud Billing account or create a new Cloud Billing account. Google uses this payment information to verify your identity. We don't charge your Spanner instance unless you explicitly [upgrade your Cloud Billing account to a paid account](https://docs.cloud.google.com/free/docs/free-cloud-features#steps-to-upgrade-your-account) , and you [upgrade your Spanner free trial instance to a paid instance](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart#upgrade) .

    a\. Click **Go to billing** .

    b\. Then, click **Link a billing account** .

    > **Note:** If you're a new Google Cloud customer, you might also be eligible for the [Google Cloud 90-day, \$300 Free Trial](https://docs.cloud.google.com/free/docs/free-cloud-features#free-trial) that offers \$300 in free Cloud Billing credits to pay for any Google Cloud resources. The Spanner free trial instance is in addition to the \$300 Free Trial credits offered by the Google Cloud Free Trial, and you don't need to use any free Cloud Billing credits to create a free trial instance.

    c\. Follow the steps to **Create billing account** , and then link it to your project.

    d\. After you enable Cloud Billing for your project, go to the **Spanner** page.

    e\. Click **Create free instance** , and proceed to [Create free trial instance](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart#create) in the next section.

4.  Optional: If you have created a Spanner instance in the project before, you see the following **Spanner Instances** page.

    Click **Create free instance** .

### gcloud

## Create a free trial instance

### Google Cloud console

The following steps explain how to create a free trial instance using the Google Cloud console.

On the **Create your free trial instance** page, do the following:

1.  Enter the **Instance name** to display in the Google Cloud console. The instance must be unique within your Google Cloud project.

2.  Enter the **Instance ID** to permanantly identify your instance. The instance ID must also be unique within your Google Cloud project. You can't change the instance ID later.

3.  Select a configuration location from the **Region** drop-down menu.

    Your instance configuration determines the geographic location where your instance is stored and replicated. You can create a free trial instance in any of the Spanner [regional instance configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations#regional_configurations) . For a full list of all available instance configurations, see [Regional, dual-region, and multi-region configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations) .

4.  Click **Create free instance** .

    After you create your free trial instance, Spanner creates a sample database for you to explore and familiarize yourself with Spanner features. For more information, see [Free trial instances overview](https://docs.cloud.google.com/spanner/docs/free-trial-instance) .

### gcloud

To create a free trial instance, use the [`gcloud spanner instances create`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/create) command.

```
gcloud spanner instances create INSTANCE_ID \
   --instance-type=free-instance --config=INSTANCE_CONFIG \
   --description=INSTANCE_DESCRIPTION
```

Replace the following:

- ` INSTANCE_ID ` : a permanent identifier that is unique within your Google Cloud project. You can't change the instance ID later.

- ` INSTANCE_CONFIG ` : a permanent identifier of your instance configuration, which defines the geographic location of the instance. You can create a free trial instance in any of the Spanner [regional instance configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations#regional_configurations) :

- ` INSTANCE_DESCRIPTION ` : the name to display for the instance in the Google Cloud console. The instance name must be unique within your Google Cloud project.

For example, to create a free trial instance named `trial-instance` with the display name `Trial Instance` using the regional instance configuration `regional-us-east5` , run the following:

```
gcloud spanner instances create trial-instance --config=regional-us-east5 \
  --instance-type=free-instance --description="Trial Instance"
```

> **Note:** Use the instance ID, not the display name, when referring to an instance in `gcloud CLI` commands.

After you create your free trial instance, you are prompted to continue learning and exploring Spanner by launching a step-by-step tutorial that teaches you how to create a database using a sample application. For more information, see [Get started with a sample application](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart#sample-app) .

## Get started with a sample database

After you create your free trial instance using the Google Cloud console, Spanner creates a sample database for you to explore and familiarize yourself with Spanner features.

You can access the sample database by using the Google Cloud console.

### Google Cloud console

1.  In the Google Cloud console, go to the **Spanner Instances** page.

2.  Click the name of the free trial instance.

3.  Click the name of the sample database.

You can view the tables and data in the sample database. You can also access and view saved queries on the Spanner Studio page. The saved queries showcase different Spanner features and capabilities.

## Get started with a sample application

Spanner also offers an open source [sample application](https://github.com/GoogleCloudPlatform/cloud-spanner-samples) to help you get started with the Spanner free trial instance. The sample application consists of a backend gRPC service backed by a Spanner database and a workload generator that drives traffic to the service.

You can access the sample application by using the Google Cloud CLI.

### gcloud

1.  Complete the steps described in the [gcloud set up](https://docs.cloud.google.com/spanner/docs/getting-started/set-up) , which covers creating and setting a default Google Cloud project, enabling the Cloud Spanner API, and setting up OAuth 2.0 to get authentication credentials to use the Cloud Spanner API.

    Run the [`gcloud auth application-default login`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/application-default/login) command to set up your local development environment with authentication credentials.

2.  Run the [`gcloud spanner samples run`](https://docs.cloud.google.com/sdk/gcloud/reference/alpha/spanner/samples/run) command to download the sample application and start the backend gRPC service and workload generator for the given sample application:

    ```
    gcloud spanner samples run APPNAME --instance-id INSTANCE_ID
    ```

3.  For more information and a list of other available commands for the sample application, see the [gcloud CLI documentation](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/samples) .

## Import your own data

You can import your own data into a Spanner database by using a CSV file, a MySQL dump file, or a PostgreSQL dump file. You can upload a local file using Cloud Storage or from a Cloud Storage bucket directly. Uploading a local file using Cloud Storage might incur charges.

If you choose to use a CSV file, you also need to upload a separate JSON file that contains the database schema.

### Google Cloud console

1.  In the Google Cloud console, go to the **Spanner Instances** page.

2.  Select the instance to create the database in.

3.  Click **Import my own data** .

4.  Enter the following values:

    - Select the **File type** .

    - Upload the file from your computer or select a Cloud Storage bucket path to the file.

    - (Optional) If you choose to use a CSV file, you also need to upload a separate JSON file that contains the database schema. The JSON file must use the following structure to define the schema:

      ```
      {
        "name": "COLUMN_NAME",
        "type": "TYPE",
        "notNull": NOT_NULL_VALUE,
        "primaryKeyOrder": PRIMARY_KEY_ORDER
      }
      ```

      Replace the following:

      - ` COLUMN_NAME ` : the name of the column in the table.

      - ` TYPE ` : the data type of the column.

      - (Optional) ` NOT_NULL_VALUE ` : whether the column can store null values or not. Valid inputs are `true` or `false` . Defaults to `false` .

      - (Optional): ` PRIMARY_KEY_ORDER ` : determines the primary key order. Set the value is set to `0` for a non-primary key column. Set the value to an integer, for example, `1` for a primary key column. Lower numbered columns appear earlier in a compound primary key.

      The CSV file expects a comma for the field delimiter and a new line for the line delimiter by default. For more information on using custom delimiters, see the [`gcloud alpha spanner databases import`](https://docs.cloud.google.com/sdk/gcloud/reference/alpha/spanner/databases/import) reference.

    - Select a new or existing database as the destination.

5.  Click **Import** .

6.  Spanner opens the Cloud Shell and populates a command that installs the [Spanner migration tool](https://googlecloudplatform.github.io/spanner-migration-tool/) and runs the [`gcloud alpha spanner databases import`](https://docs.cloud.google.com/sdk/gcloud/reference/alpha/spanner/databases/import) command. Press the `ENTER` key to import data into your database.x

## Upgrade a free trial instance

The following steps explain how to upgrade your free trial instance.

### Google Cloud console

1.  In the Google Cloud console, go to the **Spanner Instances** page.

2.  Click the name of the free trial instance.

3.  On the **Instance Overview** page, click **Edit instance** or **Edit to upgrade** .

4.  In the **Update instance name** field, enter a more applicable name for your paid instance if applicable.

5.  Select your **Upgrade option** . You can select one of the following:

    - **Upgrade now**
    - **Automatically upgrade to the full version of the [Enterprise edition](https://docs.cloud.google.com/spanner/docs/editions-overview) after my trial expires**
    - **Remind me later**

    The **Summary** section provides a description of compute and storage costs for the upgraded paid instance that you selected.

6.  Click **Save** to upgrade your free trial instance.

### gcloud

To upgrade your free trial instance to a paid Enterprise edition instance with the same instance configuration, run the following [`gcloud spanner instances update`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/update) command:

```
gcloud spanner instances update INSTANCE_ID --instance-type=provisioned
```

## Delete the instance

> **Warning:** Deleting an instance permanently removes the instance and all its databases. You cannot undo this later. Also, you cannot create another free trial instance after you've deleted your first free trial instance. You can create one free trial instance per project lifecycle.

### Google Cloud console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Click the name of the instance that you want to delete.

3.  On the **Instance Overview** page, click **Delete instance** .

4.  Follow the instructions to confirm that you want to delete the instance.

5.  Click **Delete** .

### gcloud

To delete your free trial instance, use the following [`gcloud spanner instances delete`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/instances/delete) command:

```
gcloud spanner instances delete INSTANCE_ID
```

## What's next

- Learn more about [Spanner free trial instances](https://docs.cloud.google.com/spanner/docs/free-trial-instance) .
- Learn more about Spanner [instances](https://docs.cloud.google.com/spanner/docs/instances) and [databases](https://docs.cloud.google.com/spanner/docs/databases) .
- For details on Spanner pricing after the free trial period, see the [Pricing page](https://cloud.google.com/spanner/pricing) .
