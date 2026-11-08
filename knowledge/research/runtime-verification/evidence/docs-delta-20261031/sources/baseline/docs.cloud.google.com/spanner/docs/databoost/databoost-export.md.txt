---
name: documents/docs.cloud.google.com/spanner/docs/databoost/databoost-export
uri: https://docs.cloud.google.com/spanner/docs/databoost/databoost-export
title: Export data with Data Boost
description: Export data from Spanner using Spanner Data Boost with Dataflow for high-performance, serverless offline access.
data_source: docs.cloud.google.com
---

This page explains how to use Spanner Data Boost to export Spanner data with near-zero impact to existing workloads on the provisioned Spanner instance.

To learn about Data Boost, see [Data Boost overview](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) .

You can export data by using the **Export data** page in the Spanner console, or by using a Dataflow template.

## Before you begin

Ensure that you have the `spanner.databases.useDataBoost` Identity and Access Management (IAM) permission. For more information, see [Access control with IAM](https://docs.cloud.google.com/spanner/docs/iam#databases) .

## Export data by using the Spanner console

To export data in Avro format by using the Spanner console:

1.  Follow the instructions in [Export databases from Spanner to Avro](https://docs.cloud.google.com/spanner/docs/export) .

2.  Select the **Use Spanner Data Boost** checkbox.

To export data in CSV format:

- Use the [Spanner to Cloud Storage Text](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided/cloud-spanner-to-cloud-storage) Dataflow template. See the next section.

## Export data by using Dataflow templates

To run Dataflow exports with Data Boost, select one of the following options:

### Console

1.  Go to one of the following pages:

    - [Spanner to Cloud Storage Avro](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided/cloud-spanner-to-avro)
    - [Spanner to Cloud Storage Text](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided/cloud-spanner-to-cloud-storage)

2.  Follow the **Console** instructions under **Run the template** .

3.  On the **Create job from template** page, under **Optional parameters** , enter `true` in the **Use Spanner Data Boost** field.

### gcloud CLI

1.  Go to one of the following pages:

    - [Spanner to Cloud Storage Avro](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided/cloud-spanner-to-avro)
    - [Spanner to Cloud Storage Text](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided/cloud-spanner-to-cloud-storage)

2.  Follow the **gloud** instructions under **Run the template** .

3.  Add the following parameter to the command:

    ```
    dataBoostEnabled=true
    ```

    The following example runs the Spanner to Cloud Storage Avro template and specifies the use of Data Boost.

    ```
    gcloud dataflow jobs run my_export_job \
    --gcs-location gs://dataflow-templates/latest/Cloud_Spanner_to_GCS_Avro \
    --region us-central1 \
    --staging-location gs://mybucket/temp \
    --parameters \
    instanceId=my_instance,\
    databaseId=my_database,\
    outputDir=gs://mybucket/export \
    dataBoostEnabled=true
    ```

## What's next

- Learn about Data Boost in [Data Boost overview](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) .
- [Use Data Boost in your applications](https://docs.cloud.google.com/spanner/docs/databoost/databoost-applications)
