---
name: documents/docs.cloud.google.com/spanner/docs/databoost/databoost-applications
uri: https://docs.cloud.google.com/spanner/docs/databoost/databoost-applications
title: Use Data Boost in your applications
description: Explore various application use cases for Spanner Spanner Data Boost, such as data export, ETL, and federated queries.
data_source: docs.cloud.google.com
---

This page lists the Spanner APIs that support Spanner Data Boost and explains how to view sample code that uses Data Boost. With Data Boost, you can run large analytic queries with near-zero impact to existing workloads on the provisioned Spanner instance.

## Before you begin

Ensure that the principal (for example, the service account) that runs the application has the `spanner.databases.useDataBoost` Identity and Access Management (IAM) permission. For more information, see [Access control with IAM](https://docs.cloud.google.com/spanner/docs/iam) .

## APIs

For partitioned reads with Data Boost, the following Spanner APIs have an option to enable Data Boost:

- `ExecuteSql` [RPC](https://docs.cloud.google.com/spanner/docs/reference/rpc/google.spanner.v1#google.spanner.v1.Spanner.ExecuteSql) \| [REST](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases.sessions/executeSql)
- `ExecuteStreamingSql` [RPC](https://docs.cloud.google.com/spanner/docs/reference/rpc/google.spanner.v1#google.spanner.v1.Spanner.ExecuteStreamingSql) \| [REST](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases.sessions/executeStreamingSql)
- `read` [RPC](https://docs.cloud.google.com/spanner/docs/reference/rpc/google.spanner.v1#google.spanner.v1.Spanner.Read) \| [REST](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases.sessions/read)
- `streamingRead` [RPC](https://docs.cloud.google.com/spanner/docs/reference/rpc/google.spanner.v1#google.spanner.v1.Spanner.StreamingRead) \| [REST](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases.sessions/streamingRead)

We recommend that you use `ExecuteStreamingSql` and `streamingRead` in your applications, because `ExecuteSql` and `read` are limited to 10 MB of data in their responses.

## Sample code

For examples of using Data Boost in your application code, see [Read data in parallel](https://docs.cloud.google.com/spanner/docs/reads#read_data_in_parallel) .

## Apache Spark SQL Connect for Google Cloud Spanner

The Apache Spark SQL Connector for Google Cloud Spanner supports reading Google Cloud Spanner tables into Spark's DataFrames using the Spanner Java library. For more information about the Apacha Spark SQL Connector, see [Apache Spark SQL connector for Google Cloud Spanner](https://github.com/GoogleCloudDataproc/spark-spanner-connector) .

## What's next

- Learn about Data Boost in [Data Boost overview](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) .
- [Monitor Data Boost usage](https://docs.cloud.google.com/spanner/docs/databoost/databoost-monitor)
- [Monitor and manage Data Boost quota usage](https://docs.cloud.google.com/spanner/docs/databoost/databoost-quotas)
