---
name: documents/docs.cloud.google.com/spanner/docs/performance
uri: https://docs.cloud.google.com/spanner/docs/performance
title: Performance overview
description: Optimize Spanner performance, covering schema design, query optimization, indexing, and monitoring best practices.
data_source: docs.cloud.google.com
---

This page describes the approximate performance that Spanner can provide under optimal conditions, factors that can affect performance, and tips for testing and troubleshooting Spanner performance issues.

The information on this page applies to both GoogleSQL and PostgreSQL databases.

## Performance and storage improvements

Performance and storage improvements have rolled out to all [Spanner regional, dual-region, and multi-region instance configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations) . You don't need to make any changes to your application or manually configure anything in your Spanner instances to take advantage of these improvements, which are offered at no additional cost. These performance improvements result in higher throughput and better latency in Spanner nodes in all instance configurations.

### Increased performance throughput

All Spanner instance configurations have improved performance and offer increased throughput. The following table provides the approximate throughput (queries per second) for Spanner instance configurations:

| Instance configuration type      | Peak reads (QPS per region) |        | Peak writes (QPS total) |     | Peak writes using [throughput optimized writes](https://docs.cloud.google.com/spanner/docs/throughput-optimized-writes) (QPS total) |
|----------------------------------|-----------------------------|--------|-------------------------|-----|-------------------------------------------------------------------------------------------------------------------------------------|
| **Regional**                     | SSD: 22,500 HDD: 1,500      | **or** | SSD: 3,500 HDD: 3,500   |     | SSD: 22,500 HDD: 22,500                                                                                                             |
| **Dual-region and multi-region** | SSD: 15,000 HDD: 1,000      | **or** | SSD: 2,700 HDD: 2,700   |     | SSD: 15,000 HDD: 15,000                                                                                                             |

For information about the performance throughput of these instance configurations, see [Performance for typical workloads](https://docs.cloud.google.com/spanner/docs/performance#typical-workloads) . For more information about using solid-state drives (SSD) and hard disk drives (HDD) to store your data, see [Tiered storage overview](https://docs.cloud.google.com/spanner/docs/tiered-storage) .

Read guidance is given per region (because reads can be served from any read-write or read-only region), while write guidance is for the entire configuration. Read guidance assumes you're reading single rows of 1KB. Write guidance assumes that you're writing single rows at 1KB of data per row.

Peak write performance using [throughput optimized writes](https://docs.cloud.google.com/spanner/docs/throughput-optimized-writes) is achieved using a batching delay of 100ms.

In general, both the read and write throughputs of a Spanner instance scale linearly as you add more compute capacity (nodes or processing units) to the instance. For example, if a single-region Spanner instance with 2 nodes can provide up to 45,000 reads per second, then a single-region Spanner instance with 4 nodes can provide up to 90,000 reads per second.

If you aren't seeing the expected performance for your workload from Spanner, see [troubleshooting performance regressions](https://docs.cloud.google.com/spanner/docs/troubleshooting-performance-regressions) for information about common causes.

### Increased storage

For all Spanner regional, dual-region, and multi-region instance configurations, each node (1,000 processing units) of compute capacity in the instance has an increased storage capacity of 10 TiB.

## Performance for typical workloads

All Spanner instance configurations have improved performance and offer increased throughput.

### Performance for regional configurations

Each 1,000 processing units (1 node) of compute capacity can provide the following peak performance (at 100% CPU) in a regional instance configuration:

| Peak reads (QPS per region) |        | Peak writes (QPS total) |     | Peak writes using [throughput optimized writes](https://docs.cloud.google.com/spanner/docs/throughput-optimized-writes) (QPS total) |
|-----------------------------|--------|-------------------------|-----|-------------------------------------------------------------------------------------------------------------------------------------|
| SSD: 22,500 HDD: 1,500      | **or** | SSD: 3,500 HDD: 3,500   |     | SSD: 22,500 HDD: 22,500                                                                                                             |

For regional instance configurations that allow optional read-only replicas, the optional read-only replica can support an additional 7,500 reads per second for SSD storage and 500 reads per second for HDD storage.

> **Note:** These throughput numbers are **estimates only** , and they reflect a read-only or write-only workload. Spanner throughput is highly dependent on workload, schema design, and dataset characteristics. These throughput numbers can help as a starting point when you are estimating the approximate [compute capacity](https://docs.cloud.google.com/spanner/docs/compute-capacity) (nodes or processing units) required for your Spanner instance. But these numbers can't be used for exact sizing and cost estimates.

### Performance for dual-region configurations

Each 1,000 processing units (1 node) of compute capacity can provide the following peak performance (at 100% CPU) in a dual-region instance configuration. Use [throughput optimized writes](https://docs.cloud.google.com/spanner/docs/throughput-optimized-writes) to increase write throughput beyond the numbers in the table.

| Base configuration name  | Approximate peak reads (QPS per region) | Approximate peak writes (QPS total) |
|--------------------------|-----------------------------------------|-------------------------------------|
| `dual-region-australia1` | SSD: 15,000 HDD: 1,000                  | SSD: 2,700 HDD: 2,700               |
| `dual-region-germany1`   | SSD: 15,000 HDD: 1,000                  | SSD: 2,700 HDD: 2,700               |
| `dual-region-india1`     | SSD: 15,000 HDD: 1,000                  | SSD: 2,700 HDD: 2,700               |
| `dual-region-japan1`     | SSD: 15,000 HDD: 1,000                  | SSD: 2,700 HDD: 2,700               |

Read guidance is given per region (because reads can be served from anywhere), while write guidance is for the entire configuration. Read and write guidance assume that you're reading and writing single rows at 1 KB of data per row.

### Performance for multi-region configurations

Each Spanner multi-region instance configuration has slightly different performance characteristics based on the replication topology. Use [throughput optimized writes](https://docs.cloud.google.com/spanner/docs/throughput-optimized-writes) to increase write throughput beyond the numbers in the table.

Each 1,000 processing units (1 node) of compute capacity can provide the following peak performance (at 100% CPU):

| Base configuration name | Approximate peak reads (QPS per region)                                                                                                                                                                                          | Approximate peak writes (QPS total) |
|-------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------|
| `asia1`                 | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `asia2`                 | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `eur3`                  | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `eur5`                  | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `eur6`                  | SSD: 15,000, 7,500 for each optional read-only replica HDD: 1,000, 500 for each optional read-only replica                                                                                                                       | SSD: 2,700 HDD: 2,700               |
| `eur7`                  | SSD: 15,000, 7,500 for each optional read-only replica HDD: 1,000, 500 for each optional read-only replica                                                                                                                       | SSD: 2,700 HDD: 2,700               |
| `nam3`                  | SSD: 15,000, 7,500 for each optional read-only replica HDD: 1,000, 500 for each optional read-only replica                                                                                                                       | SSD: 2,700 HDD: 2,700               |
| `nam6`                  | SSD: 15,000 in `us-central1` and `us-east1` 7,500 in `us-west1` and `us-west2` HDD: 1,000 in `us-central1` and `us-east1` 500 in `us-west1` and `us-west2` [*\[1\]*](https://docs.cloud.google.com/spanner/docs/performance#QPS) | SSD: 2,700 HDD: 2,700               |
| `nam7`                  | SSD: 15,000, 7,500 for each optional read-only replica HDD: 1,000, 500 for each optional read-only replica                                                                                                                       | SSD: 2,700 HDD: 2,700               |
| `nam8`                  | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `nam9`                  | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `nam10`                 | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `nam11`                 | SSD: 15,000, 7,500 for each optional read-only replica HDD: 1,000, 500 for each optional read-only replica                                                                                                                       | SSD: 2,700 HDD: 2,700               |
| `nam12`                 | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `nam13`                 | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `nam14`                 | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `nam15`                 | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `nam16`                 | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `nam-eur-asia1`         | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |
| `nam-eur-asia3`         | SSD: 15,000 HDD: 1,000                                                                                                                                                                                                           | SSD: 2,700 HDD: 2,700               |

- *\[1\]* : `us-west1` and `us-west2` provide only half of the QPS performance because they contain one replica per region instead of two.

Read guidance is given per region (because reads can be served from anywhere), while write guidance is for the entire configuration. Read and write guidance assume that you're reading and writing single rows at 1 KB of data per row.

## Run your typical workloads against Spanner

Always run your own typical workloads against a Spanner instance when doing capacity planning, so you can figure out the best resource allocation for your applications. Google's PerfKit Benchmarker uses [YCSB](https://ycsb.site/) to benchmark cloud services. You can follow the [PerfKitBenchmarker tutorial for Spanner](https://github.com/cloudspannerecosystem/spanner-benchmarks-tutorial) to create tests for your own workloads. When doing so, you should tune the parameters in the benchmarking configuration `yaml` files to make sure that the generated benchmark reflects the following characteristics in your production environment:

- Total size of your database
- Schema (For example: [row key size, number of columns, row data sizes](https://docs.cloud.google.com/spanner/docs/schema-design) )
- Data access pattern (row key distribution)
- Mixture of reads versus writes
- Type and complexity of queries

### Reproduce benchmark numbers

To reproduce the benchmark numbers, follow the [Benchmarking Spanner with PerfKit Benchmarker tutorial](https://github.com/cloudspannerecosystem/spanner-benchmarks-tutorial) using the corresponding `yaml` files in the [`throughput_benchmark`](https://github.com/cloudspannerecosystem/spanner-benchmarks-tutorial/tree/main/data/throughput_benchmarks) folder.

To benchmark instances in an instance configuration that has undergone [performance improvements](https://docs.cloud.google.com/spanner/docs/performance#improved-performance) , ensure that your tests are running in one of these improved instance configurations.

## Zonal and regional failure protection

When running your workloads in production, it is important to provision enough compute capacity to continue to serve your traffic in the event of the loss of an entire zone (for regional instances) or an entire region (for dual-region and multi-region instances). For more information about the recommended maximum CPU, see [alerts for high CPU utilization](https://docs.cloud.google.com/spanner/docs/cpu-utilization#recommended-max) .

## What's next

- Learn how to [design a Spanner schema](https://docs.cloud.google.com/spanner/docs/schema-design) .
- Find out how to [monitor Spanner performance](https://docs.cloud.google.com/spanner/docs/monitoring-console) .
- Learn how to [troubleshoot issues with Key Visualizer](https://docs.cloud.google.com/spanner/docs/key-visualizer) .
- Learn about [Spanner pricing](https://cloud.google.com/spanner/pricing) .
