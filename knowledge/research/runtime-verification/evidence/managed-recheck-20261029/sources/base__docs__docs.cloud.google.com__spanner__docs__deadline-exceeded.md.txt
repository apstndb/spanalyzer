---
name: documents/docs.cloud.google.com/spanner/docs/deadline-exceeded
uri: https://docs.cloud.google.com/spanner/docs/deadline-exceeded
title: Troubleshoot Spanner deadline exceeded errors
description: Troubleshoot errors in Spanner by analyzing query performance, transaction locks, and network latency.
data_source: docs.cloud.google.com
---

This page provides an overview of Spanner deadline exceeded errors: what they are, why they occur, and how to troubleshoot and resolve them.

When accessing Spanner APIs, requests might fail due to `DEADLINE_EXCEEDED` errors. This error indicates that a response has not been received within the configured timeout period.

A deadline exceeded error might occur for many different reasons, such as overloaded Spanner instances, unoptimized schemas, or unoptimized queries. This page describes common scenarios in which a deadline exceeded error happens, and provides a guide on how to investigate and resolve these issues.

## Spanner's deadline and retry philosophy

Spanner's deadline and retry philosophy differs from many other systems. In Spanner, you should specify a timeout deadline as the maximum amount of time in which a response is useful. Setting an artificially short deadline just to immediately retry the same operation again is not recommended, as this will lead to situations where operations never complete. In this context, the following strategies and operations are not recommended; they are counterproductive and defeat Spanner's internal retry behavior:

- Setting a deadline that is too short. This means that the operation is not resilient to occasional tail latency increases and can't complete before it times out. Instead, set a deadline that is the maximum amount of time in which a response is useful.

- Setting a deadline that is too long, and canceling the operation before the deadline exceeds. This leads to retries and wasted work on each try. In aggregate, this can create significant additional load on your instance.

## What is a deadline exceeded error?

When you use one of the [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) , the underlying gRPC layer takes care of communication, marshaling, unmarshalling, and deadline enforcement. Deadlines allow your application to specify how long it is willing to wait for a request to complete before the request is terminated with the deadline exceeded error.

The [timeout configuration guide](https://docs.cloud.google.com/spanner/docs/custom-timeout-and-retry) demonstrates how you can specify deadlines (or timeouts) in each of the supported Spanner client libraries. The Spanner client libraries use default timeout and retry policy settings which are defined in the following configuration files:

- [spanner_grpc_service_config.json](https://github.com/googleapis/googleapis/blob/master/google/spanner/v1/spanner_grpc_service_config.json)
- [spanner_admin_instance_grpc_service_config.json](https://github.com/googleapis/googleapis/blob/master/google/spanner/admin/instance/v1/spanner_admin_instance_grpc_service_config.json)
- [spanner_admin_database_grpc_service_config.json](https://github.com/googleapis/googleapis/blob/master/google/spanner/admin/database/v1/spanner_admin_database_grpc_service_config.json)

To learn more about gRPC deadlines, see [gRPC and Deadlines](https://grpc.io/blog/deadlines/) .

## How to investigate and resolve common deadline exceeded errors

You might encounter `DEADLINE_EXCEEDED` errors for the following issue types:

- [Data access API issues](https://docs.cloud.google.com/spanner/docs/deadline-exceeded#data-access)
- [Data API issues](https://docs.cloud.google.com/spanner/docs/deadline-exceeded#data-api)
- [Admin API issues](https://docs.cloud.google.com/spanner/docs/deadline-exceeded#admin-api)
- [Google Cloud console issues](https://docs.cloud.google.com/spanner/docs/deadline-exceeded#console)
- [Dataflow issues](https://docs.cloud.google.com/spanner/docs/deadline-exceeded#dataflow)

### Data access API issues

A Spanner instance must be appropriately configured for your specific workloads to avoid data access API issues. The following sections describe how to investigate and resolve different data access API issues.

#### Check the Spanner instance CPU load

Request latency can significantly increase as CPU utilization crosses the recommended [healthy threshold](https://docs.cloud.google.com/spanner/docs/cpu-utilization#recommended-max) . You can check your Spanner CPU utilization in the [monitoring console](https://docs.cloud.google.com/spanner/docs/monitoring-console) provided in the Google Cloud console. You can also create [alerts](https://docs.cloud.google.com/spanner/docs/monitoring-cloud#create-alert) based on the instance's CPU utilization.

##### Resolution

For steps to reduce the instance's CPU utilization, see [reducing CPU utilization](https://docs.cloud.google.com/spanner/docs/cpu-utilization#reduce) .

#### Check the request's end-to-end latency breakdown

As a request travels from the client to Spanner servers and back, there are several network hops that need to be made: from the client library to Google Front End (GFE); from the GFE to the Spanner API frontend; and finally from the Spanner API frontend to the Spanner database. If there are network issues at any of these stages, you might see deadline exceeded errors.

It's possible to capture the latency at each stage. To learn more, see [Latency points in a Spanner request](https://docs.cloud.google.com/spanner/docs/latency-points) . To find where latency occurs in Spanner, see [identify where latency occurs in Spanner](https://docs.cloud.google.com/spanner/docs/identify-latency-point) .

##### Resolution

Once you obtain the latency breakdown, you can [use metrics to diagnose latency](https://docs.cloud.google.com/spanner/docs/latency-metrics) , understand why it's happening, and find solutions.

### Data API issues

Certain non-optimal usage patterns of Spanner's Data API might cause deadline exceeded errors. This section provides guidelines on how to check for these non-optimal usage patterns.

#### Check for expensive queries

Trying to run expensive queries that don't execute within the configured timeout deadline in the client libraries might result in a deadline exceeded error. Some examples of expensive queries include, but are not limited to, full scans of a large table, cross-joins over several large tables, or a query execution with a predicate over a non-key column (also a full table scan).

You can inspect expensive queries using the [query statistics table](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics) and the [transaction statistics table](https://docs.cloud.google.com/spanner/docs/introspection/transaction-statistics) . These tables show information about slow running queries and transactions, such as the average number of rows read, the average bytes read, the average number of rows scanned and more. Moreover, you can generate [query execution plans](https://docs.cloud.google.com/spanner/docs/query-execution-plans) to further inspect how your queries are being executed.

##### Resolution

To optimize your queries, use the [best practices for SQL queries guide](https://docs.cloud.google.com/spanner/docs/sql-best-practices) . You can also use the data obtained through the statistics tables mentioned previously and execution plans to optimize your queries and make schema changes to your databases. These best practices can help reduce the execution time of the statements, potentially helping to rid the deadline exceeded errors.

#### Check for lock contention

Spanner transactions need to acquire [locks](https://docs.cloud.google.com/spanner/docs/transactions#introduction) to commit. Applications running at high throughput may cause transactions to compete for the same resources, causing an increased wait to obtain the locks and impacting overall performance. This could result in exceeded deadlines for any read or write requests.

You can find the root cause for high latency read-write transactions by using the [lock statistics table](https://docs.cloud.google.com/spanner/docs/introspection/lock-statistics) and checking out the following [blog post](https://medium.com/google-cloud/lock-statistics-diagnose-performance-issues-in-cloud-spanner-266aab4ee3e2) . Within your lock statistics table, you can find the row keys with the highest lock wait times.

This [lock conflicts troubleshooting guide](https://docs.cloud.google.com/spanner/docs/introspection/lock-statistics#troubleshooting_lock_conflicts_in_your_database_using_lock_statistics) explains how to find the transactions that are accessing the columns involved in the lock conflict. You can also discover which transactions are involved in a lock conflict using the [troubleshooting with transaction tags guide](https://docs.cloud.google.com/spanner/docs/introspection/troubleshooting-with-tags#discovering_the_transactions_involved_in_lock_conflict) .

##### Resolution

Apply these [best practices](https://docs.cloud.google.com/spanner/docs/introspection/lock-statistics#apply_best_practices_to_reduce_lock_contention) to reduce lock contentions. In addition, use [read-only transactions](https://docs.cloud.google.com/spanner/docs/transactions#read-only_transactions) for plain reads use cases to avoid lock conflicts with the writes. Using read-write transactions should be reserved for writes or mixed read-write workflows. Following these steps should improve the overall latency of your transaction execution time and reduce deadline exceeded errors.

#### Check for unoptimized schemas

Prior to designing an optimal database schema for your Spanner database, you should consider the kinds of queries that are going be executed in your database. Sub-optimal schemas may cause performance issues when running some queries. These performance issues might prevent requests from completing within the configured deadline.

##### Resolution

The most optimal schema design will depend on the reads and writes being made to your database. The [schema design best practices](https://docs.cloud.google.com/spanner/docs/schema-design) and [SQL best practices](https://docs.cloud.google.com/spanner/docs/sql-best-practices) guides should be followed regardless of schema specifics. By following these guides, you avoid the most common schema design issues. Some other root causes for poor performance are attributed to your [choice of primary keys](https://medium.com/google-cloud/cloud-spanner-choosing-the-right-primary-keys-cd2a47c7b52d) , table layout (see [using interleaved tables for faster access](https://medium.com/google-cloud/cloud-spanners-table-interleaving-a-query-optimization-feature-b8a87059da16) ), schema design (see [optimizing schema for performance](https://docs.cloud.google.com/spanner/docs/whitepapers/optimizing-schema-design) ), and the performance of the node configured within your Spanner instance (see Spanner [Performance overview](https://docs.cloud.google.com/spanner/docs/performance) ).

#### Check for hotspots

Because Spanner is a distributed database, the [schema design](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#primary_keys) needs to account for preventing hotspots. For example, creating monotonically increasing columns will limit the number of splits that Spanner can work with to distribute the workload evenly. These bottlenecks might result in timeouts. Also, you can use the [Key Visualizer](https://docs.cloud.google.com/spanner/docs/key-visualizer) to troubleshoot performance issues caused by hotspots.

##### Resolution

Refer to the resolutions identified in the previous section [Check for unoptimized schemas](https://docs.cloud.google.com/spanner/docs/deadline-exceeded#check-schemas) as a first step to resolve this issue. Redesign your [database schema](https://docs.cloud.google.com/spanner/docs/schema-design) and use [interleaved indexes](https://docs.cloud.google.com/spanner/docs/schema-design#creating-indexes) to avoid indexes that might cause hotspotting. If following these steps don't mitigate the problem, refer to the [choose a primary key to prevent hotspots guide](https://docs.cloud.google.com/spanner/docs/schema-design#primary-key-prevent-hotspots) . Finally, avoid suboptimal traffic patterns such as large range reads which might prevent load based splitting.

#### Check for misconfigured timeouts

The client libraries provide reasonable timeout defaults for all requests in Spanner. However, these default configurations might need to be adjusted for your specific workload. It is worth observing the cost of your queries and adjusting the deadlines to be suitable to your specific use case.

##### Resolution

The default settings for timeouts are suitable for most use cases. Users can override these configurations (see the [custom timeout and retry guide](https://docs.cloud.google.com/spanner/docs/custom-timeout-and-retry) ), but it is not recommended to use more aggressive timeouts than the default ones. If you decide to change the timeout, set it to the actual amount of time the application is willing to wait for the result. You can experiment with longer configured timeouts, but never set a timeout shorter than the actual time the application is willing to wait, as this would cause the operation to be retried more frequently.

### Admin API issues

Admin API requests are expensive operations when compared to data API requests. Admin requests like `CreateInstance` , `CreateDatabase` or `CreateBackups` can take many seconds before returning a response. Spanner client libraries set 60 minutes long deadlines for both [instance](https://github.com/googleapis/googleapis/blob/master/google/spanner/admin/instance/v1/spanner_admin_instance_grpc_service_config.json) and [database](https://github.com/googleapis/googleapis/blob/master/google/spanner/admin/database/v1/spanner_admin_database_grpc_service_config.json) administrator requests. This is to ensure the server has the opportunity to complete the request before the client retries or fails.

##### Resolution

If you're using the Google [Spanner client library](https://docs.cloud.google.com/spanner/docs/reference/libraries) to access the administrator API, ensure the client library is updated and using the latest version. If you are accessing the Spanner API directly through a client library you created, ensure you don't have more aggressive deadline settings than the default settings (60 minutes) for your [instance](https://github.com/googleapis/googleapis/blob/master/google/spanner/admin/instance/v1/spanner_admin_instance_grpc_service_config.json) and [database](https://github.com/googleapis/googleapis/blob/master/google/spanner/admin/database/v1/spanner_admin_database_grpc_service_config.json) administrator requests.

### Google Cloud console issues

Queries issued from the Google Cloud console Spanner Studio page cannot exceed five minutes. If you create an expensive query that takes more than five minutes to run, you will see the following error message:

![Screenshot of Google Cloud console deadline exceeded error message](https://docs.cloud.google.com/static/spanner/docs/images/deadline_exceeded_error.png)

The backend will cancel the failed query, and the transaction might roll back if necessary.

##### Resolution

You can rewrite the query using the [best practices for SQL queries guide](https://docs.cloud.google.com/spanner/docs/sql-best-practices) .

### Dataflow issues

In Apache Beam, the default timeout configuration is [two hours](https://github.com/apache/beam/blob/master/sdks/java/io/google-cloud-platform/src/main/java/org/apache/beam/sdk/io/gcp/spanner/SpannerAccessor.java) for read operations and [15 seconds](https://github.com/apache/beam/blob/master/sdks/java/io/google-cloud-platform/src/main/java/org/apache/beam/sdk/io/gcp/spanner/SpannerConfig.java) for commit operations. These configurations allow for longer operations when compared to the standalone client library deadline timeouts. However, it is still possible to receive a timeout and deadline exceeded error when the work items are too large. If necessary, you can customize the Apache Beam commit timeout configuration.

##### Resolution

If a deadline exceeded error occurs in the steps `ReadFromSpanner / Execute query / Read from Spanner / Read from Partitions` , check the [query statistics table](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics#sql-scan-data) to find out which query scanned a large number of rows. Then, modify such queries to try and reduce the execution time.

Another example of a Dataflow deadline exceeded error is shown in the following exception message:

```
exception:
     org.apache.beam.sdk.util.UserCodeException:
     com.google.cloud.spanner.SpannerException: DEADLINE_EXCEEDED:
     io.grpc.StatusRuntimeException: DEADLINE_EXCEEDED: deadline exceeded after
     3599.999905380s.
     [remote_addr=batch-spanner.googleapis.com/172.217.5.234:443] at
 org.apache.beam.runners.dataflow.worker.GroupAlsoByWindowsParDoFn$1.output(GroupAlsoByWindowsParDoFn.java:184)
```

This timeout resulted because the work items are too large. In the previous example, the following two recommendations might help. Firstly, you can try enabling the [shuffle service](https://cloud.google.com/blog/products/gcp/introducing-cloud-dataflow-shuffle-for-up-to-5x-performance-improvement-in-data-analytic-pipelines) if it is not yet enabled. Secondly, you can try tweaking the configurations in your database's read, such as `maxPartitions` and `partitionSizeBytes` . For more information, see [`PartitionOptions`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/PartitionOptions) to try and reduce the work item size. An example of how to do this can be found in this [Dataflow template](https://github.com/GoogleCloudPlatform/DataflowTemplates/blob/main/v1/src/main/java/com/google/cloud/teleport/templates/common/SpannerConverters.java#L207) .

## Additional deadline exceeded troubleshooting resources

If you're still seeing a `DEADLINE_EXCEEDED` error after you've completed the troubleshooting steps, [open a support case](https://docs.cloud.google.com/spanner/docs/getting-support) if you experience the following scenarios:

- A high Google Front End latency, but low Spanner API request latency
- A high Spanner API request latency, but a low query latency

You can also refer to the following troubleshooting resources:

- [Examine latency in a Spanner component with OpenTelemetry](https://docs.cloud.google.com/spanner/docs/capture-visualize-latency)
- [Troubleshoot performance regressions](https://docs.cloud.google.com/spanner/docs/troubleshooting-performance-regressions)
- [Analyze running queries in Spanner to help diagnose performance issues](https://medium.com/@rghetia/analyze-running-queries-in-cloud-spanner-to-help-diagnose-performance-issues-4d8d85ccc21a)
