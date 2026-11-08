---
name: documents/docs.cloud.google.com/spanner/docs/free-trial-instance
uri: https://docs.cloud.google.com/spanner/docs/free-trial-instance
title: Spanner free trial instances overview
description: Create a Spanner free trial instance to explore features and develop applications without cost.
data_source: docs.cloud.google.com
---

> **Note:** A Spanner free trial instance supports Standard edition features, and Enterprise edition features, such as [KNN](https://docs.cloud.google.com/spanner/docs/find-k-nearest-neighbors) , [ANN vector distance functions](https://docs.cloud.google.com/spanner/docs/find-approximate-nearest-neighbors) , [full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) , and [Spanner Graph](https://docs.cloud.google.com/spanner/docs/graph/overview) . For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

This page provides an overview of Spanner free trial instances.

A Spanner free trial instance lets you learn and explore Spanner features and capabilities at no cost for up to 90 days. Using a Spanner free trial instance, you can create GoogleSQL or PostgreSQL-dialect databases and store up to 10 GB of data to test out key Spanner features, and learn key Spanner concepts.

You can [create a Spanner free trial instance](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart#create) using the Google Cloud console or Google Cloud CLI.

## Explore sample databases and applications

When you create the free trial instance using the Google Cloud console, Spanner creates and preloads a sample database for an ecommerce store in the free trial instance. The sample database schema defines entities for users, products, orders, shopping carts, order items, addresses, and payments. Spanner inserts sample data into the ecommerce database. You can view saved queries on the Spanner Studio page for the sample database that help showcase the following Spanner capabilities:

- The relational model
- Full-text search
- Vector search for product similarity
- Graph relationships between entities such as users, orders, and products

You can also create new databases, define schema, and load data in the free trial instance. To learn more, see [Create a Spanner free trial instance and sample application](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart) .

In addition to the ecommerce use case database, Spanner offers an open source [sample application](https://github.com/GoogleCloudPlatform/cloud-spanner-samples) to help you get started with the Spanner free trial instance. The sample application includes a backend gRPC service backed by a Spanner database and a workload generator that drives traffic to the service.

## Cost and eligibility

A Spanner free trial instance is available at no cost to both existing and new Google Cloud customers. If you're a new Google Cloud customer, you might also be eligible for the [Google Cloud 90-day, \$300 Free Trial](https://docs.cloud.google.com/free/docs/free-cloud-features#free-trial) that offers \$300 in free Cloud Billing credits to pay for any Google Cloud resources. The Spanner free trial instance is in addition to the \$300 Free Trial credits offered by the Google Cloud Free Trial, and you don't need to use any free Cloud Billing credits to create a free trial instance.

For more information about the Spanner free trial instance, see the following table:

<table>
<colgroup>
<col style="width: 50%" />
<col style="width: 50%" />
</colgroup>
<tbody>
<tr class="odd">
<th>Eligibility</th>
<td><p>You're eligible to create a Spanner free trial instance if you have a Cloud Billing account that's active, and Cloud Billing is enabled for your project.</p></td>
</tr>
<tr class="even">
<th>Spanner editions</th>
<td><p>A free trial instance supports Standard edition features, and Enterprise edition features, such as <a href="https://docs.cloud.google.com/spanner/docs/find-k-nearest-neighbors">KNN</a> , <a href="https://docs.cloud.google.com/spanner/docs/find-approximate-nearest-neighbors">ANN vector distance functions</a> , <a href="https://docs.cloud.google.com/spanner/docs/full-text-search">full-text search</a> , and <a href="https://docs.cloud.google.com/spanner/docs/graph/overview">Spanner Graph</a> . To experience other Enterprise edition features and more of Spanner's performance at scale, upgrade to a paid instance. Your free trial instance defaults to the Enterprise edition when you upgrade it to a paid instance. If you don't want to use the Enterprise edition, upgrade your free trial instance to the paid Enterprise edition first. Then, you can upgrade your instance to the Enterprise Plus edition, or contact support to downgrade to the Standard edition. For more information, see the <a href="https://docs.cloud.google.com/spanner/docs/editions-overview">Spanner editions overview</a> .</p></td>
</tr>
<tr class="odd">
<th>Initiation</th>
<td><p>The Spanner free trial instance period starts automatically when you create a free trial instance.</p>
<p>If you already have an active Cloud Billing account, you can create a Spanner free trial instance using the Google Cloud console or gcloud CLI.</p>
<p>If you are a new Google Cloud customer, sign in to the Google Cloud console with your Google Account and set up a Cloud Billing account with your <a href="https://docs.cloud.google.com/billing/docs/how-to/payment-methods#available_payment_methods">credit card or other payment method</a> first before creating a Spanner free trial instance. Google uses this payment information to verify your identity. We don't charge your Spanner instance unless you explicitly <a href="https://docs.cloud.google.com/free/docs/free-cloud-features#how-to-upgrade">upgrade your Cloud Billing account to a paid account</a> , and you <a href="https://docs.cloud.google.com/spanner/docs/free-trial-quickstart#upgrade">upgrade your Spanner free trial instance to a paid instance</a> .</p></td>
</tr>
<tr class="even">
<th>Duration</th>
<td><p>Your free trial instance ends when one of the following occurs:</p>
<ul>
<li>90 days have elapsed since you created your free trial instance.</li>
<li>You upgrade your free trial instance to a paid Enterprise edition instance.</li>
<li>You delete the free trial instance.</li>
</ul>
<p>If you don't upgrade your free trial instance after the 90-day trial period, the instance stops serving requests and enters a 30-day grace period. During the grace period, the data in the instance is retained and you can still upgrade your free trial instance. If you don't upgrade your free trial instance by the end of the 30-day grace period, the instance, along with the data in it, is deleted. Note that even if you enable <a href="https://docs.cloud.google.com/spanner/docs/prevent-database-deletion">database deletion protection</a> on a database in your free trial instance, your free trial instance will still be deleted after the 30-day grace period unless you upgrade it to a paid Enterprise edition instance.</p>
<p><strong>Note:</strong> An active Cloud Billing account is required to keep using your Spanner free trial instance. If your <a href="https://docs.cloud.google.com/free/docs/free-cloud-features#end">Google Cloud Free Trial ends</a> before the end of your Spanner free trial instance, then you need to <a href="https://docs.cloud.google.com/free/docs/free-cloud-features#how-to-upgrade">upgrade your Cloud Billing account to a paid account</a> to continue using your Spanner free trial instance.</p></td>
</tr>
<tr class="odd">
<th>Service level agreement (SLA)</th>
<td><p>SLAs don't apply to free trial instances. The free trial instance is intended to help you learn and explore Spanner. We don't recommend running production applications in your free trial instance.</p></td>
</tr>
<tr class="even">
<th>Available instance configurations</th>
<td><p>You can create a free trial instance in any of the Spanner <a href="https://docs.cloud.google.com/spanner/docs/instance-configurations#regional_configurations">regional instance configurations</a> .</p>
<p>You can't create a free trial instance in a <a href="https://docs.cloud.google.com/spanner/docs/instance-configurations">dual-region or multi-region instance configuration</a> . To create instances in a dual-region or multi-region configuration, use a paid Enterprise Plus edition instance.</p></td>
</tr>
</tbody>
</table>

## Limitations

The free trial instance is subject to the following limitations:

- One free trial instance allowed per project lifecycle.
- Maximum five free trial instances allowed per Cloud Billing account.
- Maximum five databases allowed per free trial instance.
- 90-day free trial instance period provided.
- No [SLA](https://cloud.google.com/spanner/sla) guarantees provided.
- All Spanner [quotas and limits](https://docs.cloud.google.com/spanner/quotas) apply.
- One free trial instance provides 10 GB of storage capacity and limited [compute capacity](https://docs.cloud.google.com/spanner/docs/compute-capacity) required for learning purposes. You can't edit a free trial instance to increase its storage or compute capacity limits. You can increase the storage and compute capacity of a free trial instance by [upgrading to a paid instance](https://docs.cloud.google.com/spanner/docs/free-trial-instance#upgrade) .
- Free trial instances don't support [backup and restore](https://docs.cloud.google.com/spanner/docs/backup) .
- Free trial instances don't support [customer-managed encryption keys (CMEK)](https://docs.cloud.google.com/spanner/docs/cmek) .
- Spanner free trial instances are available in all [regional configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) . They aren't available in [dual-region](https://docs.cloud.google.com/spanner/docs/instance-configurations#dual-region-configurations) or [multi-region configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations#multi-region-configurations) .
- [Data Boost](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) , and graph algorithms that rely on it, are not available.

The free trial instance is meant for evaluation purposes. It is not meant for the following use cases:

- Production related activities
- Performance evaluation and load testing of Spanner
- Ongoing testing and development

## Performance

> **Note:** These performance numbers are estimates only. Spanner performance is highly dependent on workload, schema design, and dataset characteristics.

A Spanner free trial instance can provide around 500 queries per second (QPS) of reads or 100 QPS of writes (writing single rows at 1 KB of data per row) when you follow these best practices:

- [Design a schema](https://docs.cloud.google.com/spanner/docs/schema-design) that prevents hotspots and other performance issues.
- [Write efficient queries](https://docs.cloud.google.com/spanner/docs/sql-best-practices) and follow other SQL best practices.
- Keep high priority [total CPU utilization under 65%](https://docs.cloud.google.com/spanner/docs/cpu-utilization#recommended-max) as recommended.
- Place compute resources within the same region as your Spanner instance, for optimal read and write latency.

You can monitor and optimize the performance of your free trial instance with the help of:

- [Monitoring charts and metrics](https://docs.cloud.google.com/spanner/docs/monitoring-console#charts-metrics)
- [Query Insights](https://docs.cloud.google.com/spanner/docs/using-query-insights)
- [Query plan visualizer](https://docs.cloud.google.com/spanner/docs/tune-query-with-visualizer)
- [Introspection tools](https://docs.cloud.google.com/spanner/docs/introspection)
- [Spanner Key Visualizer](https://docs.cloud.google.com/spanner/docs/key-visualizer)

For better performance, upgrade your free trial instance to a paid instance and scale up the [compute capacity](https://docs.cloud.google.com/spanner/docs/compute-capacity) of your instance.

## Upgrade from free trial instances

You can upgrade your Spanner free trial instance to a paid Enterprise edition instance anytime during the 90-day free trial instance period without any downtime. You aren't charged unless you explicitly upgrade your free trial instance to a paid instance.

Upgrading your Spanner free trial instance has the following benefits:

- Keeps your Spanner free trial instance running uninterrupted beyond the 90-day free trial period.
- Gives you access to Spanner features that aren't available in the free trial instance, such as backups and managed autoscaler. You can also upgrade to the Enterprise Plus edition for additional features, such as geo-partitioning and multi-regional configurations with 99.999% availability.
- Lets you increase the compute capacity of your instance and obtain higher performance (QPS, throughput) and storage capacity (greater than 10 GB).

As an option, you can opt in to automatically upgrade your free trial instance to a paid Enterprise edition instance after 90 days. For more information, see [Upgrade free trial instance](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart#upgrade) .

If don't want to use the Enterprise edition, upgrade your free trial instance to the paid Enterprise edition first. Then, you can upgrade your instance to the Enterprise Plus edition, or contact support to downgrade to the Standard edition. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

## Troubleshooting FAQs

Why can't I create a free trial instance in my project?  
You can create one free trial instance per project. If you've already created a free trial instance in your project, you can't create another one.

Why can't I create a free trial instance in instance configuration X?  
Spanner free trial instances are available in all [regional configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) . They aren't available in [dual-region](https://docs.cloud.google.com/spanner/docs/instance-configurations#dual-region-configurations) or [multi-region configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations#multi-region-configurations) .

How can I create more than five databases in a free trial instance?  
If you want to create more than five databases, upgrade your free trial instance to a paid instance.

My free trial instance has been disabled. How can I re-enable it?  
If you've completed the Spanner 90-day free trial period, the free trial instance is disabled and enters a 30-day grace period. You can start using your instance again by [upgrading your free trial instance to a paid instance](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart#upgrade) during the 90-day trial period or the 30-day grace period.

## Support

To get support for your Spanner instance, you can:

- Get a Google support package
- Ask a question on Stack Overflow
- Join online help communities
- File a bug or feature request

For more information, see [Get support](https://docs.cloud.google.com/spanner/docs/getting-support) .

## What's next

- Learn more about [Spanner editions](https://docs.cloud.google.com/spanner/docs/editions-overview) .
- Learn more about how to [get started with a free trial instance and sample application](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart) .
- Learn more about Spanner [quotas and limits](https://docs.cloud.google.com/spanner/quotas) .
- Learn more about Spanner [instances](https://docs.cloud.google.com/spanner/docs/instances) and [databases](https://docs.cloud.google.com/spanner/docs/databases) .
- For details on Spanner pricing after the free trial period, see the [Pricing page](https://cloud.google.com/spanner/pricing) .
