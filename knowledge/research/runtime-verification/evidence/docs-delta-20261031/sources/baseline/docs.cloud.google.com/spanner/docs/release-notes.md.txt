---
name: documents/docs.cloud.google.com/spanner/docs/release-notes
uri: https://docs.cloud.google.com/spanner/docs/release-notes
title: Spanner release notes
description: A managed, mission-critical, globally consistent and scalable relational database service.
data_source: docs.cloud.google.com
---

This page documents production updates to Spanner. You can periodically check this page for announcements about new or updated features, bug fixes, known issues, and deprecated functionality.

You can see the latest product updates for all of Google Cloud on the [Google Cloud](https://docs.cloud.google.com/release-notes) page, browse and filter all release notes in the [Google Cloud console](https://console.cloud.google.com/release-notes) , or programmatically access release notes in [BigQuery](https://console.cloud.google.com/bigquery?p=bigquery-public-data&d=google_cloud_release_notes&t=release_notes&page=table) .

To get the latest product updates delivered to you, add the URL of this page to your [feed reader](https://wikipedia.org/wiki/Comparison_of_feed_aggregators) , or add the [feed URL](https://docs.cloud.google.com/feeds/spanner-release-notes.xml) directly.

## October 20, 2026

Feature

Spanner supports approximate aggregate functions for both GoogleSQL and PostgreSQL dialects. These functions produce statistical estimates of distinct counts, allowing you to compute results faster and save resources on large datasets compared to exact aggregation.

For more information, see approximate aggregate functions ( [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/approximate_aggregate_functions) and [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions#approximate-aggregate) ), [HyperLogLog++ functions](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/hll_functions) , and [Sketches](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/sketches) .

## September 18, 2026

Feature

Spanner supports automatic parameterization of SQL query literals to improve query performance, reduce latency, and lower CPU costs. Spanner converts literal values hardcoded in CRUD-style queries, such as primary key lookups, index lookups, and primary key joins, into query parameters, allowing execution plans to be cached and reused to reduce latency and CPU costs.

For more information, see [Use query parameters](https://docs.cloud.google.com/spanner/docs/sql-best-practices#query-parameters) and statement hints ( [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#statement_hints) and [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/query-syntax#statement-hints) ).

## September 17, 2026

Feature

Spanner queues are [generally available (GA)](https://cloud.google.com/products/#product-launch-stages) . Spanner queues provide transactional messaging to help you manage asynchronous work. The feature pairs this capability with the scalability and reliability of Spanner, letting you build event-driven applications. For more information, see the [Spanner queues overview](https://docs.cloud.google.com/spanner/docs/queues/queues-overview) .

## September 11, 2026

Feature

Spanner Graph supports the following enhancements for `CREATE PROPERTY GRAPH` :

- **Semantic options for labels and properties** : You can configure `OPTIONS` ( `description` and `synonyms` ) on labels and properties to provide context and improve discoverability for natural language querying interfaces.
- **Element key uniqueness validation option** : You can configure the `validate_element_key_uniqueness` option in the property graph `OPTIONS` clause to control whether Spanner Graph validates element key uniqueness at schema creation time.

For more information, see [CREATE PROPERTY GRAPH statement](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#create_property_graph) and [Disabled key validation](https://docs.cloud.google.com/spanner/docs/graph/graph-with-views-overview#disabled-key-validation) .

## September 03, 2026

Feature

Spanner supports the following PostgreSQL-dialect database operators:

- `ILIKE` ( `~~*` )
- `NOT ILIKE` ( `!~~*` )

Spanner supports the following PostgreSQL-dialect database functions:

- `pg.ilike`
- `pg.not_ilike`

For more information, see [Pattern matching operators](https://docs.cloud.google.com/spanner/docs/reference/postgresql/operators#pattern-matching-operators) and [Functions](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions) .

## September 02, 2026

Feature

Spanner supports using the `TABLESAMPLE` operator in PostgreSQL-dialect databases to select a random sample of a dataset.

For more information, see [`TABLESAMPLE` operator](https://docs.cloud.google.com/spanner/docs/reference/postgresql/query-syntax#tablesample_operator) .

## August 28, 2026

Feature

Spanner supports creating secondary indexes based on scalar expressions that apply to one or more table columns. Expression indexes eliminate the need to create and maintain a dedicated generated column in your table for indexing purposes. For more information, see [Create an index based on a scalar expression](https://docs.cloud.google.com/spanner/docs/secondary-indexes#expression-index) .

## August 11, 2026

Feature

For DML statements, Spanner now enforces the 80,000 mutation limit (including indexes) per statement rather than cumulatively across the transaction. This allows a transaction to execute multiple DML statements that collectively exceed the limit, as long as each individual statement remains under it. For the Mutation API, the 80,000 limit applies to all mutations in the commit. All transactions are still subject to the 100 MiB commit size limit.

## August 10, 2026

Feature

Dynamic channel pooling (DCP) for gRPC channels in the Spanner Go and Java client libraries is generally available ( [GA](https://cloud.google.com/products#product-launch-stages) ).

DCP prevents performance issues from under-provisioned or over-provisioned channels and reduces configuration overhead. DCP is disabled by default.

For more information, see [Configure the number of sessions and gRPC channels in the pools](https://docs.cloud.google.com/spanner/docs/sessions#configure_the_number_of_sessions_and_grpc_channels_in_the_pools) .

## July 28, 2026

Feature

Spanner supports creating tables without defining primary keys. When you create a table without a primary key, Spanner creates a hidden `rowid` column that serves as the primary key. For more information, see [Create a table without defining a primary key](https://docs.cloud.google.com/spanner/docs/primary-key-default-value#tables-without-primary-keys) .

## July 06, 2026

Feature

You can use Gemini in Spanner Studio to fix errors in your SQL queries. When you run a query that contains an error, you can click **Fix** to view a line-by-line comparison of your query and a recommended correction, along with an explanation of the change. This feature is available in [Preview](https://docs.cloud.google.com/products#product-launch-stages) .

## June 23, 2026

Feature

Spanner supports [direct connectivity](https://docs.cloud.google.com/spanner/docs/latency-points#direct_connectivity) . When enabled, your application traffic is routed directly to Spanner servers, bypassing the Google Front End (GFE) servers. This can reduce your overall latency. Direct connectivity is [generally available (GA)](https://docs.cloud.google.com/products#product-launch-stages) .

## June 15, 2026

Feature

You can publish a [Spanner conversational analytics agent in Gemini Enterprise](https://docs.cloud.google.com/gemini/data-agents/conversational-analytics/spanner/create-data-agents#publish-agent-gemini-enterprise) . This feature is in [Preview](https://cloud.google.com/products/#product-launch-stages) .

## May 27, 2026

Feature

Spanner Graph supports a suite of [graph algorithms](https://docs.cloud.google.com/spanner/docs/graph/graph-algorithms-overview) covering use cases such as fraud detection, entity resolution, and recommendations. You can invoke graph algorithms as built-in function calls in Spanner Graph queries. You can save your output to Cloud Storage or Spanner. This feature is available in [Preview](https://docs.cloud.google.com/products#product-launch-stages) .

## May 18, 2026

Feature

You can optimize Spanner Graph query performance by using factorized execution. If a graph pattern traversal generates duplicate intermediate results, add the `@{factorize_mode}` hint to the pattern traversal or at the query level to optimize the query.

For more information, see [Use factorized execution to optimize queries](https://docs.cloud.google.com/spanner/docs/graph/best-practices-tuning-queries#use-factorized-execution) .

## May 15, 2026

Feature

Spanner Studio supports query parameters, which let you bind literal values to a templated SQL statement at query time in the Google Cloud console. For more information, see [query parameters](https://docs.cloud.google.com/spanner/docs/sql-best-practices#query-parameters) .

## May 14, 2026

Feature

Spanner supports the `FACTORIZED_MODE` join hint. You can control which sides of a join to factorize (deduplicate) before executing the join by using the `FACTORIZE_LEFT` , `FACTORIZE_RIGHT` , or `FACTORIZE_BOTH` hint values. For more information, see [Factorized mode](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#factorized_mode) .

Announcement

The Spanner change streams default [retention period](https://docs.cloud.google.com/spanner/docs/change-streams#data-retention) has been increased from 1 day to 7 days. This change affects both new and existing change streams that don't have a retention period explicitly set. You can always specify the retention period through [create change stream](https://docs.cloud.google.com/spanner/docs/change-streams/manage#create) or [alter change stream](https://docs.cloud.google.com/spanner/docs/change-streams/manage#modify) DDL statements to override the default.

Feature

You can populate new PostgreSQL dialect databases in an existing Spanner instance from sample datasets that help you explore Spanner capabilities.

For more information, see [Create and manage databases](https://docs.cloud.google.com/spanner/docs/create-manage-databases#use-datasets) .

## May 04, 2026

Feature

Spanner [vector index](https://docs.cloud.google.com/spanner/docs/vector-indexes) and [approximate nearest neighbor (ANN)](https://docs.cloud.google.com/spanner/docs/find-approximate-nearest-neighbors) distance functions are [generally available](https://cloud.google.com/products/#product-launch-stages) (GA) for PostgreSQL databases.

## April 21, 2026

Feature

Spanner full-text search supports [custom dictionaries](https://docs.cloud.google.com/spanner/docs/full-text-search/search-query-enhancement#custom-dictionaries) to create custom synonym mappings. You can use custom dictionaries with the `SEARCH` , `SCORE` , and `SNIPPET` functions.

## April 20, 2026

Feature

The [Spanner remote MCP server](https://docs.cloud.google.com/spanner/docs/use-spanner-mcp) is generally available ( [GA](https://cloud.google.com/products#product-launch-stages) ). The Spanner remote MCP server lets you interact with Spanner instances from LLMs, AI applications, and AI-enabled development platforms.

Feature

You can use the [Database Insights remote MCP server](https://docs.cloud.google.com/spanner/docs/reference/mcp/databaseinsights/mcp) to analyze Spanner's performance and system metrics. This feature is generally available ( [GA](https://cloud.google.com/products#product-launch-stages) ).

Feature

You can now use Spanner Studio to [visually create and manage a Spanner Graph schema](https://docs.cloud.google.com/spanner/docs/graph/create-update-drop-schema-visually) . Visual modeling simplifies graph design by enabling you to map nodes and edges through an intuitive interface instead of creating manual DDL statements. This feature is available in [Preview](https://docs.cloud.google.com/products#product-launch-stages) .

## April 17, 2026

Feature

Repeatable read isolation is [generally available](https://docs.cloud.google.com/products#product-launch-stages) . You can use it to reduce latency and transaction failure rates for workloads that have many reads contending with fewer writes. For more information, see [Repeatable read isolation](https://docs.cloud.google.com/spanner/docs/isolation-levels#repeatable-read) .

Feature

Spanner supports Gemini Cloud Assist investigation capabilities. You can create, run, and edit [Gemini Cloud Assist investigations](https://docs.cloud.google.com/cloud-assist/investigations) only if you have a [Premium support contract](https://cloud.google.com/support/premium) .

For more information, see [monitor and troubleshoot your Spanner instance with AI assistance](https://docs.cloud.google.com/spanner/docs/monitor-troubleshoot-with-ai) .

Feature

Columnar engine for Spanner is now [generally available (GA)](https://cloud.google.com/products/#product-launch-stages) . Columnar engine is a storage technique used with analytical queries to make scans up to 200 times faster on live operational data without affecting transaction workloads. This release enables support for Columnar Engine in databases that use the [Postgres interface](https://docs.cloud.google.com/spanner/docs/configure-columnar-engine#enable-columnar-engine-postgres) .

For more information, see the [Columnar engine for Spanner overview](https://docs.cloud.google.com/spanner/docs/configure-columnar-engine#enable-columnar-engine-postgres) .

## April 13, 2026

Feature

Spanner supports importing data from Cloud SQL for MySQL. This feature lets you migrate the schema and perform a one-time bulk data load to evaluate Spanner for your use case. For more information, see [Import from Cloud SQL to Spanner](https://docs.cloud.google.com/spanner/docs/import-database-cloudsql) .

This feature is [generally available](https://docs.cloud.google.com/products#product-launch-stages) .

## April 06, 2026

Feature

The QueryData tool lets you to query the data in your database using conversational language and build data agents. For more information, see [QueryData tool overview](https://docs.cloud.google.com/spanner/docs/data-agent-overview) . This feature is available in ( [Preview](https://cloud.google.com/products#product-launch-stages) ).

The preview release increases the accuracy of SQL generation with value search queries which match values and their context within a database. Value search queries trigger automatically.

## March 30, 2026

Feature

Spanner offers conversational analytics, which lets users query their operational data using natural language. This feature is powered by the [Conversational Analytics API](https://docs.cloud.google.com/gemini/data-agents/conversational-analytics-api/overview) , which can help you translate complex human dialog into precise database queries to provide actionable insights. This feature is in [Preview](https://docs.cloud.google.com/products#product-launch-stages) . For more information, see [Conversational analytics for Spanner overview](https://docs.cloud.google.com/gemini/data-agents/conversational-analytics/spanner) .

Feature

You can create BigQuery [non-incremental materialized views over Spanner data](https://docs.cloud.google.com/bigquery/docs/materialized-views-create#spanner) to improve query performance by periodically caching results. This feature is [generally available](https://cloud.google.com/products/#product-launch-stages) (GA).

Feature

You can use [Cloud resource connections with `EXPORT DATA` statements](https://docs.cloud.google.com/bigquery/docs/export-to-spanner#export_using_a_cloud_resource_connection) to reverse ETL (extract, transform, load) BigQuery data to Spanner. This feature is [generally available (GA)](https://cloud.google.com/products/#product-launch-stages) .

## March 25, 2026

Feature

The Spark Spanner connector supports writing a Spark Dataframe to a Spanner table using the Spark data source API. For more information, see [Use the Spark Spanner connector](https://docs.cloud.google.com/dataproc/docs/tutorials/spanner-connector-spark-example#write-spanner-tables) .

## March 19, 2026

Feature

Spanner now offers [AI functions](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/ml-functions) , as a part of machine learning functions, that help you perform semantic operations using Large Language Models (LLMs) in SQL to classify, evaluate, and rank your data:

- [`AI.CLASSIFY`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/ml-functions#aiclassify) : Classify a natural language input into user-defined categories.
- [`AI.IF`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/ml-functions#aiif) : Evaluate a condition described in natural language.
- [`AI.SCORE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/ml-functions#aiscore) : Rate natural language input and assign it a score.

## March 10, 2026

Feature

Spanner supports the optimistic concurrency control mode. Optimistic concurrency control is suitable for transactional workloads with low read-write contention. It assumes that conflicts in transactions are rare. Reads and queries within a read-write transaction proceed without acquiring locks. For more information, see [Concurrency control](https://docs.cloud.google.com/spanner/docs/concurrency-control) .

## March 04, 2026

Feature

You can use [continuous queries to stream BigQuery data to Spanner in real time](https://docs.cloud.google.com/bigquery/docs/export-to-spanner#export_continuously) . This feature is [generally available (GA)](https://cloud.google.com/products/#product-launch-stages) .

## March 03, 2026

Feature

Spanner supports the `ON CONFLICT` clause for GoogleSQL `INSERT` statements to handle unique constraint violations. For more information, see [`ON CONFLICT DO NOTHING`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-on-conflict-do-nothing) and [`ON CONFLICT DO UPDATE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-on-conflict-do-update) .

## February 27, 2026

Feature

Spanner supports using the pending commit timestamp function as a default value and an `ON UPDATE` expression for column definitions. For more information, see the GoogleSQL [`ON UPDATE` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#spanner-on-update) and the PostgreSQL [`ON UPDATE` clause](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#postgresql-on-update) .

## February 26, 2026

Feature

Spanner provides JSON string versions of the following statistics columns for PostgreSQL-dialect databases:

- `TOTAL_LATENCY_DISTRIBUTION_JSON_STRING` in [Transaction statistics](https://docs.cloud.google.com/spanner/docs/introspection/transaction-statistics)
- `OPERATIONS_BY_TABLE_JSON_STRING` in [Transaction statistics](https://docs.cloud.google.com/spanner/docs/introspection/transaction-statistics)
- `LATENCY_DISTRIBUTION_JSON_STRING` in [Query statistics](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics)
- `SAMPLE_LOCK_REQUESTS_JSON_STRING` in [Lock statistics](https://docs.cloud.google.com/spanner/docs/introspection/lock-statistics)

You can use these columns to retrieve statistics in JSON format from the `SPANNER_SYS` tables.

## February 25, 2026

Feature

Spanner supports the following:

- [Monitor the usage of Enterprise edition and Enterprise Plus edition features](https://docs.cloud.google.com/spanner/docs/editions-overview#monitor-edition-feature-usage) in your Spanner instance.
- [Downgrade the instance's edition using the Google Cloud console](https://docs.cloud.google.com/spanner/docs/create-manage-instances#downgrade-edition) .
- [Verify edition compatibility when restoring from a backup](https://docs.cloud.google.com/spanner/docs/backup/restore-backups) . If the database uses higher-tier features, you must restore the backup to a destination instance that uses the same or a higher-tier edition.
- [Set custom edition constraints within your organization policies](https://docs.cloud.google.com/spanner/docs/spanner-custom-constraints#google-managed_constraints) , limiting the edition of your instance that your users can create.

Feature

Spanner supports the following [managed autoscaler](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) features:

- Autoscaling based on [total CPU utilization target](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#how_managed_autoscaler_works) . When an instance's total CPU utilization exceeds or falls below the target that you have set, Spanner adds or removes compute capacity.
- Autoscaling for [instance partitions (in Preview)](https://docs.cloud.google.com/spanner/docs/geo-partitioning) .

For more information, see [Managed autoscaler](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) .

## February 24, 2026

Feature

Spanner provides `UNSPLITTABLE_REASONS` in hotspot insights and split statistics views to detect when splits can't be split further. Unsplittable reasons can help you identify schema anti-patterns that require intervention. For more information, see [Find hotspots in your database](https://docs.cloud.google.com/spanner/docs/find-hotspots-in-database) and [Split statistics](https://docs.cloud.google.com/spanner/docs/introspection/hot-split-statistics) .

This feature is [generally available (GA)](https://docs.cloud.google.com/products#product-launch-stages) .

## February 17, 2026

Deprecated

Control of MCP use with organization policies is deprecated. After March 17, 2026, organization policies that use the `gcp.managed.allowedMCPServices` constraint won't work, and you can control MCP use with IAM deny policies.

For more information about controlling MCP use, see [Control MCP use with IAM](https://docs.cloud.google.com/mcp/control-mcp-use-iam) .

Change

After March 17, 2026, when you enable Spanner, the Spanner MCP server is automatically enabled.

Announcement

New best practices are available for securing generative AI agents using Model Context Protocol (MCP) with Google Cloud databases. This guide covers key security measures like least privilege, native database controls, and secure agent design to help you build safer AI applications. For more information, see [Best practices for securing agent interactions with Model Context Protocol](https://docs.cloud.google.com/spanner/docs/secure-agent-interactions-mcp) .

This feature is in [Preview](https://cloud.google.com/products/#product-launch-stages) .

## February 10, 2026

Feature

You can use the [Spanner remote MCP server](https://docs.cloud.google.com/spanner/docs/use-spanner-mcp) to interact with Spanner instances and databases from agentic AI applications such as Gemini CLI, agent mode in Gemini Code Assist, or Claude.ai.

This feature is in [Preview](https://cloud.google.com/products/#product-launch-stages) .

## February 09, 2026

Feature

You can right-click a node in a Spanner Graph query visualization to access options like expanding or collapsing adjacent nodes, highlighting or hiding nodes, and viewing only a node's neighbors.

For more information, see [Work with visualizations](https://docs.cloud.google.com/spanner/docs/graph/work-with-visualizations#choose-nodes-to-display) .

## February 02, 2026

Feature

You can create and host [remote functions](https://docs.cloud.google.com/spanner/docs/cloud-run-remote-function) in Cloud Run and call them from Spanner queries using the GoogleSQL dialect. This feature is in [Preview](https://cloud.google.com/products/#product-launch-stages) .

## January 30, 2026

Feature

Spanner supports the `UUID` data type for both GoogleSQL and PostgreSQL-dialect databases. This data type stores universally unique identifiers (UUIDs) as 128-bit values.

You can use the GoogleSQL [`NEW_UUID()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/utility-functions#new_uuid) function or the PostgreSQL [`gen_random_uuid()`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions#utility) to function to create UUID values.

For more information, see [Use a universally unique identifier (UUID)](https://docs.cloud.google.com/spanner/docs/schema-design#uuid_primary_key) .

## January 28, 2026

Feature

Spanner supports the following compression functions:

- [`ZSTD_COMPRESS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/compression-functions#zstd_compress)
- [`ZSTD_DECOMPRESS_TO_BYTES`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/compression-functions#zstd_decompress_to_bytes)
- [`ZSTD_DECOMPRESS_TO_STRING`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/compression-functions#zstd_decompress_to_string)

These functions use the Zstandard (Zstd) lossless data compression algorithm to compress and decompress `STRING` or `BYTES` values. For more information, see [Compression functions](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/compression-functions) .

## January 26, 2026

Feature

Columnar engine for Spanner is now in [Public Preview](https://cloud.google.com/products#product-launch-stages) . Columnar engine is a storage technique used with analytical queries to speed up scans up to 200 times faster on live operational data without affecting transaction workloads. In databases or tables enabled with columnar engine, this release:

- Supports the ability to execute columnar queries automatically and perform faster columnar scans using vectorized execution.
- Provides a new [major compaction API](https://docs.cloud.google.com/spanner/docs/manual-data-compaction) to accelerate the conversion of non-columnar data into columnar data.

For more information, see the [Columnar engine for Spanner overview](https://docs.cloud.google.com/spanner/docs/columnar-engine) .

## January 20, 2026

Feature

You can create Spanner [regional instance configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) in Bangkok, Thailand ( `asia-southeast3` ). For more information, see [Google Cloud locations](https://cloud.google.com/about/locations) and [Spanner pricing](https://cloud.google.com/spanner/pricing) .

## January 13, 2026

Feature

Several updates have been made to [full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) :

- [Named schemas](https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes#create_and_query_a_search_index_for_a_named_schema) support full-text search.
- Spanner search indexes can [accelerate pattern matching expressions](https://docs.cloud.google.com/spanner/docs/full-text-search/pattern-matching-function-acceleration) such as `LIKE` , `STARTS_WITH` , and `ENDS_WITH` for pattern matching, and `REGEXP_CONTAINS` for regular expression matching.
- [`TOKENIZE_FULLTEXT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#tokenize_fulltext) has an argument for removing diacritics. `SEARCH` and `SCORE` use this if the data was tokenized with this option.
- [`TOKENIZE_SUBSTRING`](https://docs.cloud.google.com/spanner/docs/full-text-search/substring-search) supports emojis.

## January 05, 2026

Feature

You can use [SQL views](https://docs.cloud.google.com/spanner/docs/views) to create a graph. For requirements, considerations, and the benefits of using SQL views to create a graph, see [Overview of graphs created from SQL views](https://docs.cloud.google.com/spanner/docs/graph/graph-with-views-overview) . To learn how to create a graph from views, see [Create a property graph from SQL views](https://docs.cloud.google.com/spanner/docs/graph/graph-with-views-how-to) .

## December 18, 2025

Feature

The GoogleSQL function [`ELEMENT_DEFINITION_NAME`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-gql-functions#element_definition_name) is available. `ELEMENT_DEFINITION_NAME` returns the name of the graph element table underlying a graph element.

## December 17, 2025

Feature

You can build data agents that interact with the data in your database using conversational language. Use these data agents as tools to empower your applications. For more information, see [Data agents overview](https://docs.cloud.google.com/spanner/docs/data-agent-overview) . This feature is available in [Preview](https://cloud.google.com/products#product-launch-stages) , and access to it requires a [sign-up](https://forms.gle/pJByTWfenZAWbaXo7) .

## December 12, 2025

Feature

Spanner supports the PostgreSQL `generate_series()` function. You can use this function to create a sequence of numbers. For more information, see [Set returning functions](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions#set-returning-functions) .

## December 11, 2025

Feature

Spanner [Data Boost](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) includes a the [Data Boost concurrent requests in milli-operations per second per region](https://docs.cloud.google.com/spanner/docs/databoost/databoost-quotas) quota, which applies fine-grained control over how multiple concurrent requests for your project share Data Boost resources. Instead of counting 1 request against 1 unit of quota under the existing [concurrency quota regime](https://docs.cloud.google.com/spanner/docs/databoost/databoost-quotas) , Data Boost splits a request at a granularity of 1/1000, allowing for a greater number of concurrent requests to make progress. For more information, see [Quotas and limits](https://docs.cloud.google.com/spanner/quotas#data_boost_limits) .

Feature

Spanner Graph supports using the [`ANY CHEAPEST`](https://docs.cloud.google.com/spanner/docs/graph/work-with-paths#any-cheapest) path search prefix in a query to return the path with the lowest total compute cost. For more information, see [Path search prefix](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-patterns#search_prefix) in the [Spanner Graph Language](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-intro) reference.

## December 10, 2025

Feature

Spanner supports the following new columns in the `SPANNER_SYS` [oldest active queries](https://docs.cloud.google.com/spanner/docs/introspection/oldest-active-queries) table:

- `CLIENT_IP_ADDRESS`
- `API_CLIENT_HEADER`
- `USER_AGENT_HEADER`
- `SERVER_REGION`
- `PRIORITY`
- `TRANSACTION_TYPE`

You can also view these columns in the Spanner **query insights** page on the Google Cloud console. For more information, see [Monitor active queries](https://docs.cloud.google.com/spanner/docs/monitor-active-queries#view-longest-running-queries) .

## December 03, 2025

Change

String values in [Spanner Studio query results](https://docs.cloud.google.com/spanner/docs/manage-data-using-console#create-modify-query-data) are enclosed in double quotes, providing a visual cue to differentiate string values from other data types. This enhancement is for display purposes only and doesn't affect how data is exported or accessed.

## November 20, 2025

Feature

Query optimizer version 8 is the [default version](https://docs.cloud.google.com/spanner/docs/query-optimizer/versions) for Spanner.

## November 14, 2025

Feature

The GoogleSQL function [`IS_FIRST`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-sql-functions#is_first) is available for graph queries. `IS_FIRST` returns `true` if a row is in the first `k` rows (1-based) within a window. You can use `IS_FIRST` in graph queries to [limit traversed edges to improve query performance](https://docs.cloud.google.com/spanner/docs/graph/best-practices-tuning-queries#limit-traversed-edges) . You can also use `IS_FIRST` to [sample intermediate nodes to optimize multi-hop queries](https://docs.cloud.google.com/spanner/docs/graph/best-practices-tuning-queries#use-is-first) .

## November 11, 2025

Feature

[Spanner Studio](https://docs.cloud.google.com/spanner/docs/manage-data-using-console) added support for several export options for your query results. You can export to a CSV or JSON file, Google Sheets, or copy the results to a clipboard. For more information, see [Create, modify, and query your data](https://docs.cloud.google.com/spanner/docs/manage-data-using-console#create-modify-query-data) .

## November 10, 2025

Feature

Spanner automatically provides recommendations to apply [schema design best practices](https://docs.cloud.google.com/spanner/docs/schema-design) to your databases. You can view these schema issue recommendations on the Spanner Studio page for your database. For more information, see [View schema design best practice recommendations](https://docs.cloud.google.com/spanner/docs/manage-data-using-console#recommend-schema-best) . This feature is in [Preview](https://cloud.google.com/products/#product-launch-stages%22%3E) .

## October 31, 2025

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.102.1](https://github.com/googleapis/java-spanner/compare/v6.102.0...v6.102.1) (2025-10-23)

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.63.0 ( [c1a8238](https://github.com/googleapis/java-spanner/commit/c1a8238af33a083411f63cf6276eb683ee67ac6a) )
- Do a quick check if the application runs on Google Cloud ( [\#4163](https://github.com/googleapis/java-spanner/issues/4163) ) ( [b9d7daf](https://github.com/googleapis/java-spanner/commit/b9d7daf000c0fb8b67142c6161bb578cadf49b18) )
- Migrate away from GoogleCredentials.fromStream() usages ( [\#4151](https://github.com/googleapis/java-spanner/issues/4151) ) ( [94d0474](https://github.com/googleapis/java-spanner/commit/94d0474ace62ea1059e5b69243f0b6eef31ddd06) )

##### Dependencies

- Update actions/checkout action to v5 ( [\#4158](https://github.com/googleapis/java-spanner/issues/4158) ) ( [b32ebcf](https://github.com/googleapis/java-spanner/commit/b32ebcf96bbf696b1eb84204622463fac59be017) )
- Update actions/checkout action to v5 ( [\#4161](https://github.com/googleapis/java-spanner/issues/4161) ) ( [02a17c6](https://github.com/googleapis/java-spanner/commit/02a17c6e6253e026cb3c6360eb925a322143b518) )
- Update dependency com.google.cloud:sdk-platform-java-config to v3.53.0 ( [\#4178](https://github.com/googleapis/java-spanner/issues/4178) ) ( [24fe194](https://github.com/googleapis/java-spanner/commit/24fe194fa3595b2ab817b9fc4cd57840250fef1f) )
- Update dependency net.bytebuddy:byte-buddy to v1.17.8 ( [\#4154](https://github.com/googleapis/java-spanner/issues/4154) ) ( [c911381](https://github.com/googleapis/java-spanner/commit/c911381c2ca9cd46fbeb831c659aaf55f21437f2) )
- Update dependency net.bytebuddy:byte-buddy-agent to v1.17.8 ( [\#4155](https://github.com/googleapis/java-spanner/issues/4155) ) ( [3075df7](https://github.com/googleapis/java-spanner/commit/3075df714b1787512174b3f18cbc802359d442dc) )
- Update googleapis/sdk-platform-java action to v2.63.0 ( [\#4179](https://github.com/googleapis/java-spanner/issues/4179) ) ( [5f48191](https://github.com/googleapis/java-spanner/commit/5f481913d60372fccf399c5c0e168b7d0c553ba0) )

##### Documentation

- Add warning for encoded credential ( [\#4182](https://github.com/googleapis/java-spanner/issues/4182) ) ( [92620f9](https://github.com/googleapis/java-spanner/commit/92620f969908a8ba7fcf92d0b350a8c4d05398f8) )

#### [6.102.0](https://github.com/googleapis/java-spanner/compare/v6.101.1...v6.102.0) (2025-10-08)

##### Features

- Add connection property for gRPC interceptor provider ( [\#4149](https://github.com/googleapis/java-spanner/issues/4149) ) ( [deb8dff](https://github.com/googleapis/java-spanner/commit/deb8dff6c01c37a3158e8f4a28ef5e821d10092a) )
- Support statement_timeout in connection url ( [\#4103](https://github.com/googleapis/java-spanner/issues/4103) ) ( [542c6aa](https://github.com/googleapis/java-spanner/commit/542c6aa63bfdd526070f14cb76921dd34527c1f9) )

##### Bug Fixes

- Automatically set default_sequence_kind for CREATE SEQUENCE ( [\#4105](https://github.com/googleapis/java-spanner/issues/4105) ) ( [3beea6a](https://github.com/googleapis/java-spanner/commit/3beea6ac4eb53b70db34e0a2d2e33e56f450c88b) )
- **deps:** Update the Java code generator (gapic-generator-java) to 2.62.3 ( [7047a3a](https://github.com/googleapis/java-spanner/commit/7047a3ae31aae51e9e23758fe004b93855a0ee4b) )

##### Dependencies

- Update actions/checkout action to v5 ( [\#4069](https://github.com/googleapis/java-spanner/issues/4069) ) ( [4c88eb9](https://github.com/googleapis/java-spanner/commit/4c88eb91a321aa718f957296012f9e7501c7caec) )
- Update actions/checkout action to v5 ( [\#4106](https://github.com/googleapis/java-spanner/issues/4106) ) ( [14ebdb3](https://github.com/googleapis/java-spanner/commit/14ebdb35c33442c4e0f70d63dce3425edb730525) )
- Update actions/setup-java action to v5 ( [\#4071](https://github.com/googleapis/java-spanner/issues/4071) ) ( [e23134a](https://github.com/googleapis/java-spanner/commit/e23134a2f864e8abd2890ac3a81ff6b668afbe63) )
- Update all dependencies ( [\#4099](https://github.com/googleapis/java-spanner/issues/4099) ) ( [b262edc](https://github.com/googleapis/java-spanner/commit/b262edcfc4713bb64986bc4acd3f02b69d3367f8) )
- Update dependency com.google.api.grpc:grpc-google-cloud-monitoring-v3 to v3.77.0 ( [\#4117](https://github.com/googleapis/java-spanner/issues/4117) ) ( [2451ca2](https://github.com/googleapis/java-spanner/commit/2451ca2abe1dd2de3907b88e8d18beab1a15a634) )
- Update dependency com.google.api.grpc:proto-google-cloud-monitoring-v3 to v3.77.0 ( [\#4143](https://github.com/googleapis/java-spanner/issues/4143) ) ( [6c9dc26](https://github.com/googleapis/java-spanner/commit/6c9dc26330cf66f196adc2203323a482e08f0325) )
- Update dependency com.google.api.grpc:proto-google-cloud-trace-v1 to v2.76.0 ( [\#4144](https://github.com/googleapis/java-spanner/issues/4144) ) ( [d566a42](https://github.com/googleapis/java-spanner/commit/d566a4295be018070169ba082a018394a2e60b45) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.77.0 ( [\#4145](https://github.com/googleapis/java-spanner/issues/4145) ) ( [8917c05](https://github.com/googleapis/java-spanner/commit/8917c054410e4035d6d4e201e43599d5ddc1fadd) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.77.0 ( [\#4146](https://github.com/googleapis/java-spanner/issues/4146) ) ( [4ebea1a](https://github.com/googleapis/java-spanner/commit/4ebea1adf726069084087ce46900f3174658055c) )
- Update dependency com.google.cloud:google-cloud-trace to v2.76.0 ( [\#4147](https://github.com/googleapis/java-spanner/issues/4147) ) ( [4b1d4af](https://github.com/googleapis/java-spanner/commit/4b1d4af19336e493af38a1e58c95786da3892d34) )
- Update dependency com.google.cloud:google-cloud-trace to v2.76.0 ( [\#4148](https://github.com/googleapis/java-spanner/issues/4148) ) ( [8f91a89](https://github.com/googleapis/java-spanner/commit/8f91a894771653213b6fcded5795349ad7ea6724) )
- Update dependency com.google.cloud:sdk-platform-java-config to v3.52.3 ( [\#4107](https://github.com/googleapis/java-spanner/issues/4107) ) ( [8a8a042](https://github.com/googleapis/java-spanner/commit/8a8a042494b092b3dddd0c9606a63197d8a23555) )
- Update dependency org.json:json to v20250517 ( [\#3881](https://github.com/googleapis/java-spanner/issues/3881) ) ( [5658c83](https://github.com/googleapis/java-spanner/commit/5658c8378aa2e8028d4ef7dfaf94b647f33cd812) )
- Update googleapis/sdk-platform-java action to v2.62.3 ( [\#4108](https://github.com/googleapis/java-spanner/issues/4108) ) ( [65913ec](https://github.com/googleapis/java-spanner/commit/65913ec0638fec4ea536cf42f8fe25460133f68e) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [8.2.2](https://github.com/googleapis/nodejs-spanner/compare/v8.2.1...v8.2.2) (2025-10-07)

##### Bug Fixes

- Correctly determine project ID for metrics export ( [\#2427](https://github.com/googleapis/nodejs-spanner/issues/2427) ) ( [0d63312](https://github.com/googleapis/nodejs-spanner/commit/0d633126a87c1274abfd59550cb94052a819fcaa) )
- Metrics Export Error log ( [\#2425](https://github.com/googleapis/nodejs-spanner/issues/2425) ) ( [110923e](https://github.com/googleapis/nodejs-spanner/commit/110923ea1dc6f6c891e0f70406b3839224a25b9e) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.59.0](https://github.com/googleapis/python-spanner/compare/v3.58.0...v3.59.0) (2025-10-18)

##### Features

- **spanner:** Add lazy decode to partitioned query ( [\#1411](https://github.com/googleapis/python-spanner/issues/1411) ) ( [a09961b](https://github.com/googleapis/python-spanner/commit/a09961b381314e3f06f1ff4be7b672cd9da9c64b) )

##### Bug Fixes

- **spanner:** Resolve TypeError in metrics resource detection ( [\#1446](https://github.com/googleapis/python-spanner/issues/1446) ) ( [7266686](https://github.com/googleapis/python-spanner/commit/7266686d6773f39a30603061ae881e258421d927) )

##### Documentation

- Add snippet for Repeatable Read configuration at client and transaction ( [\#1326](https://github.com/googleapis/python-spanner/issues/1326) ) ( [58e2406](https://github.com/googleapis/python-spanner/commit/58e2406af3c8918e37e0daadefaf537073aed1a4) )

## October 29, 2025

Feature

[Named schemas](https://docs.cloud.google.com/spanner/docs/named-schemas) are supported in Spanner Graph. For more information, see [Spanner Graph schema overview](https://docs.cloud.google.com/spanner/docs/graph/schema-overview) .

## October 21, 2025

Feature

[Schema object drop protection](https://docs.cloud.google.com/spanner/docs/schema-drop-protection) is generally available. This feature protects schema objects such as tables, indexes, and columns from accidental deletion.

## October 20, 2025

Feature

You can use the GoogleSQL [`ML.PREDICT` function](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/ml-functions#mlpredict) to convert your natural language query text into an embedding and perform approximate nearest neighbors (ANN) vector search.

## October 02, 2025

Feature

You can use repeatable read isolation (in [Preview](https://cloud.google.com/products#product-launch-stages0) ) to reduce latency and transaction abort rates for workloads that have many reads contending with fewer writes. For more information, see [Repeatable read isolation](https://docs.cloud.google.com/spanner/docs/isolation-levels#repeatable-read) .

## October 01, 2025

Feature

The [Spanner CLI](https://docs.cloud.google.com/spanner/docs/spanner-cli) is generally available. Bundled with Google Cloud CLI, you can use the Spanner command-line interface to open an interactive session or automate SQL executions from the shell or an input file.

## September 30, 2025

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.86.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.85.1...spanner/v1.86.0) (2025-09-26)

##### Features

- **spanner:** Support "readOnly" column tag parsing for Go struct operations ( [\#12895](https://github.com/googleapis/google-cloud-go/issues/12895) ) ( [003abca](https://github.com/googleapis/google-cloud-go/commit/003abca9172082ad1f2fbcc9b37639f389ade8ee) )

##### Bug Fixes

- **spanner:** Use fresh context for rollback ( [\#12897](https://github.com/googleapis/google-cloud-go/issues/12897) ) ( [99c7eeb](https://github.com/googleapis/google-cloud-go/commit/99c7eeb6ff95af9c967c78764069f752d3c26d34) )

#### [1.85.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.84.1...spanner/v1.85.0) (2025-08-28)

##### Features

- **spanner:** Enable multiplex sessions by default for all operations ( [\#12734](https://github.com/googleapis/google-cloud-go/issues/12734) ) ( [0491ba6](https://github.com/googleapis/google-cloud-go/commit/0491ba6f258600cc518a5a2c274caa0fb6105c8a) )

##### Performance Improvements

- **spanner:** Improve mutationProto allocations and performance ( [\#12740](https://github.com/googleapis/google-cloud-go/issues/12740) ) ( [2a4add5](https://github.com/googleapis/google-cloud-go/commit/2a4add5405205fea315b17e4f70096d4be78b506) )

#### [1.85.1](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.85.0...spanner/v1.85.1) (2025-09-12)

##### Bug Fixes

- **spanner:** Disable afe_connectivity_error_count metric ( [\#12866](https://github.com/googleapis/google-cloud-go/issues/12866) ) ( [baab714](https://github.com/googleapis/google-cloud-go/commit/baab714b87311822a7c8c5202430a0cd7e11cc7d) )

##### Documentation

- **spanner:** A comment for enum Kind is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for enum Priority is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for enum value LOCK_HINT_EXCLUSIVE in enum LockHint is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for enum value LOCK_HINT_UNSPECIFIED in enum LockHint is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for enum value ORDER_BY_PRIMARY_KEY in enum OrderBy is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for enum value ORDER_BY_UNSPECIFIED in enum OrderBy is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for enum value PROFILE in enum QueryMode is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for enum value SERIALIZABLE in enum IsolationLevel is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field approximate_last_use_time in message .google.spanner.v1.Session is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field auto_failover_disabled in message .google.spanner.v1.DirectedReadOptions is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field columns in message .google.spanner.v1.Mutation is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field data_boost_enabled in message .google.spanner.v1.ExecuteSqlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field data_boost_enabled in message .google.spanner.v1.ReadRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field exclude_replicas in message .google.spanner.v1.DirectedReadOptions is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field exclude_txn_from_change_streams in message .google.spanner.v1.BatchWriteRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field include_replicas in message .google.spanner.v1.DirectedReadOptions is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field index in message .google.spanner.v1.PlanNode is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field insert_or_update in message .google.spanner.v1.Mutation is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field key_set in message .google.spanner.v1.Mutation is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field key_set in message .google.spanner.v1.PartitionReadRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field key_set in message .google.spanner.v1.ReadRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field kind in message .google.spanner.v1.PlanNode is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field last_statement in message .google.spanner.v1.ExecuteSqlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field last_statements in message .google.spanner.v1.ExecuteBatchDmlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field limit in message .google.spanner.v1.ReadRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field location in message .google.spanner.v1.DirectedReadOptions is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field max_commit_delay in message .google.spanner.v1.CommitRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field max_partitions in message .google.spanner.v1.PartitionOptions is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field multiplexed in message .google.spanner.v1.Session is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field mutation_key in message .google.spanner.v1.BeginTransactionRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field optimizer_statistics_package in message .google.spanner.v1.ExecuteSqlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field optimizer_version in message .google.spanner.v1.ExecuteSqlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field order_by in message .google.spanner.v1.ReadRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field param_types in message .google.spanner.v1.ExecuteBatchDmlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field param_types in message .google.spanner.v1.ExecuteSqlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field param_types in message .google.spanner.v1.PartitionQueryRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field params in message .google.spanner.v1.ExecuteBatchDmlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field params in message .google.spanner.v1.ExecuteSqlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field params in message .google.spanner.v1.PartitionQueryRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field partition_size_bytes in message .google.spanner.v1.PartitionOptions is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field partition_token in message .google.spanner.v1.ExecuteSqlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field partition_token in message .google.spanner.v1.Partition is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field partition_token in message .google.spanner.v1.ReadRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field plan_nodes in message .google.spanner.v1.QueryPlan is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field precommit_token in message .google.spanner.v1.CommitRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field precommit_token in message .google.spanner.v1.ExecuteBatchDmlResponse is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field ranges in message .google.spanner.v1.KeySet is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field replace in message .google.spanner.v1.Mutation is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field request_options in message .google.spanner.v1.BeginTransactionRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field request_tag in message .google.spanner.v1.RequestOptions is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field return_commit_stats in message .google.spanner.v1.CommitRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field seqno in message .google.spanner.v1.ExecuteBatchDmlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field seqno in message .google.spanner.v1.ExecuteSqlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field session_count in message .google.spanner.v1.BatchCreateSessionsRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field session_template in message .google.spanner.v1.BatchCreateSessionsRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field short_representation in message .google.spanner.v1.PlanNode is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field single_use_transaction in message .google.spanner.v1.CommitRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field sql in message .google.spanner.v1.PartitionQueryRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field transaction in message .google.spanner.v1.ExecuteSqlRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field transaction in message .google.spanner.v1.PartitionQueryRequest is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field transaction_tag in message .google.spanner.v1.RequestOptions is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field values in message .google.spanner.v1.Mutation is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for field variable in message .google.spanner.v1.PlanNode is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for message DirectedReadOptions is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for message DirectedReadOptions is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for message DirectedReadOptions is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for message Mutation is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for message PartitionOptions is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for message PlanNode is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for method BatchWrite in service Spanner is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for method Commit in service Spanner is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for method CreateSession in service Spanner is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for method DeleteSession in service Spanner is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for method ExecuteSql in service Spanner is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for method ExecuteStreamingSql in service Spanner is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for method GetSession in service Spanner is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for method PartitionQuery in service Spanner is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for method PartitionRead in service Spanner is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for method Read in service Spanner is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )
- **spanner:** A comment for method Rollback in service Spanner is changed ( [51583bd](https://github.com/googleapis/google-cloud-go/commit/51583bd5c9b886d22b45da092dc8311422b8b5ac) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.99.0](https://github.com/googleapis/java-spanner/compare/v6.98.1...v6.99.0) (2025-08-26)

##### Features

- Support read lock mode for R/W transactions ( [\#4010](https://github.com/googleapis/java-spanner/issues/4010) ) ( [7d752d6](https://github.com/googleapis/java-spanner/commit/7d752d686e638b6266aab3a5188c01641d2f9adc) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.62.0 ( [52c68db](https://github.com/googleapis/java-spanner/commit/52c68db5c75f24a066c2e828ed79917c824f699b) )
- GetCommitResponse() should return error if tx has not committed ( [\#4021](https://github.com/googleapis/java-spanner/issues/4021) ) ( [a2c179f](https://github.com/googleapis/java-spanner/commit/a2c179f2e7c19d295bdbf9cf1bbd1c5562dd9e21) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.52.0 ( [\#4024](https://github.com/googleapis/java-spanner/issues/4024) ) ( [7e3294f](https://github.com/googleapis/java-spanner/commit/7e3294f6d42bddb4cfff67334118f615c90c3bb7) )

#### [6.100.0](https://github.com/googleapis/java-spanner/compare/v6.99.0...v6.100.0) (2025-09-11)

##### Features

- Read_lock_mode support for connections ( [\#4031](https://github.com/googleapis/java-spanner/issues/4031) ) ( [261abb4](https://github.com/googleapis/java-spanner/commit/261abb4b9c5ff00fac2d816a31926b23264657c4) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.62.1 ( [e9773a7](https://github.com/googleapis/java-spanner/commit/e9773a7aa27a414d56093b4e09e0f197a07b5980) )
- Disable afe_connectivity_error_count metric ( [\#4041](https://github.com/googleapis/java-spanner/issues/4041) ) ( [f89c1c0](https://github.com/googleapis/java-spanner/commit/f89c1c0517ba6b895f405b0085b8df41aac952be) )
- Skip session delete in case of multiplexed sessions ( [\#4029](https://github.com/googleapis/java-spanner/issues/4029) ) ( [8bcb09d](https://github.com/googleapis/java-spanner/commit/8bcb09d141fe986c92ccacbaa9a45302c5c8e79d) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.52.1 ( [\#4034](https://github.com/googleapis/java-spanner/issues/4034) ) ( [13bfa7c](https://github.com/googleapis/java-spanner/commit/13bfa7c68c7ea887e679fb5504dceb85cbb43cb9) )

##### Documentation

- A comment for field `ranges` in message `.google.spanner.v1.KeySet` is changed ( [e9773a7](https://github.com/googleapis/java-spanner/commit/e9773a7aa27a414d56093b4e09e0f197a07b5980) )

#### [6.101.1](https://github.com/googleapis/java-spanner/compare/v6.101.0...v6.101.1) (2025-09-26)

##### Bug Fixes

- Potential NullPointerException in LocalConnectionChecker ( [\#4092](https://github.com/googleapis/java-spanner/issues/4092) ) ( [3b9f597](https://github.com/googleapis/java-spanner/commit/3b9f597ba60199a16556824568b24908ce938a69) )

#### [6.101.0](https://github.com/googleapis/java-spanner/compare/v6.100.0...v6.101.0) (2025-09-26)

##### Features

- Add transaction_timeout connection property ( [\#4056](https://github.com/googleapis/java-spanner/issues/4056) ) ( [cdc52d4](https://github.com/googleapis/java-spanner/commit/cdc52d49b39c57e7255f4e09fb33a41f4810397d) )
- TPC support ( [\#4055](https://github.com/googleapis/java-spanner/issues/4055) ) ( [7625cce](https://github.com/googleapis/java-spanner/commit/7625cce9ad48b14a1cff9c2ede86a066ea292bef) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.62.2 ( [8d6cbf6](https://github.com/googleapis/java-spanner/commit/8d6cbf6bea9cbd823b8f0070516e34b4d8428e87) )
- Potential NullPointerException in Value#hashCode ( [\#4046](https://github.com/googleapis/java-spanner/issues/4046) ) ( [74abb34](https://github.com/googleapis/java-spanner/commit/74abb341e2ea42bbf0a2de4ec3e3555335b5fd9f) )
- Recalculate remaining statement timeout after retry ( [\#4053](https://github.com/googleapis/java-spanner/issues/4053) ) ( [5e26596](https://github.com/googleapis/java-spanner/commit/5e26596f4f9c924260da0908920854d8ddfc626b) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.52.2 ( [\#4057](https://github.com/googleapis/java-spanner/issues/4057) ) ( [d782aff](https://github.com/googleapis/java-spanner/commit/d782aff63ff81e1b760690d4dee3e566028d522e) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [8.2.0](https://github.com/googleapis/nodejs-spanner/compare/v8.1.0...v8.2.0) (2025-08-26)

##### Features

- **spanner:** Add support for multiplexed session for r/w transactions ( [\#2351](https://github.com/googleapis/nodejs-spanner/issues/2351) ) ( [6a9f1a2](https://github.com/googleapis/nodejs-spanner/commit/6a9f1a2b2c0dad955593571c71e9d4b6c9e7eeee) )
- **spanner:** Support setting read lock mode ( [\#2388](https://github.com/googleapis/nodejs-spanner/issues/2388) ) ( [bd66f61](https://github.com/googleapis/nodejs-spanner/commit/bd66f61f3ecac65678d31cbc841c11cd0fb7c3da) )

##### Bug Fixes

- **deps:** Add uuid to dependencies ( [\#2376](https://github.com/googleapis/nodejs-spanner/issues/2376) ) ( [0b2060b](https://github.com/googleapis/nodejs-spanner/commit/0b2060b4ad7302ab23ac757e79fe760e34e81083) )
- **deps:** Update dependency @grpc/proto-loader to ^0.8.0 ( [\#2354](https://github.com/googleapis/nodejs-spanner/issues/2354) ) ( [75dc4da](https://github.com/googleapis/nodejs-spanner/commit/75dc4daf114cbc4eb4669ed6cb042af051cdce63) )
- **deps:** Update dependency google-gax to v5.0.1 ( [\#2362](https://github.com/googleapis/nodejs-spanner/issues/2362) ) ( [9223470](https://github.com/googleapis/nodejs-spanner/commit/922347014ac3966ec4a48116b61ba4850edf0b50) )
- Provide option to disable built in metrics ( [\#2380](https://github.com/googleapis/nodejs-spanner/issues/2380) ) ( [b378e2e](https://github.com/googleapis/nodejs-spanner/commit/b378e2ed6739acf76f3f3f27090311129dd83473) )
- Race condition among transactions when running parallely ( [\#2369](https://github.com/googleapis/nodejs-spanner/issues/2369) ) ( [f8b6f63](https://github.com/googleapis/nodejs-spanner/commit/f8b6f6340f4f04e04213fdf0a9665d643f474eeb) )

#### [8.2.1](https://github.com/googleapis/nodejs-spanner/compare/v8.2.0...v8.2.1) (2025-09-12)

##### Bug Fixes

- **deps:** Update dependency google-gax to v5.0.3 ( [\#2371](https://github.com/googleapis/nodejs-spanner/issues/2371) ) ( [8a175e2](https://github.com/googleapis/nodejs-spanner/commit/8a175e2e5cc8d0ed81faee7b24b59b5026758a59) )
- Disable afe_connectivity_error_count metric ( [af72d70](https://github.com/googleapis/nodejs-spanner/commit/af72d707c8857d5596bd2b93830e52c8e152967f) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.58.0](https://github.com/googleapis/python-spanner/compare/v3.57.0...v3.58.0) (2025-09-10)

##### Features

- **spanner:** Support setting read lock mode ( [\#1404](https://github.com/googleapis/python-spanner/issues/1404) ) ( [ee24c6e](https://github.com/googleapis/python-spanner/commit/ee24c6ee2643bc74d52e9f0a924b80a830fa2697) )

##### Dependencies

- Remove Python 3.7 and 3.8 as supported runtimes ( [\#1395](https://github.com/googleapis/python-spanner/issues/1395) ) ( [fc93792](https://github.com/googleapis/python-spanner/commit/fc9379232224f56d29d2e36559a756c05a5478ff) )

## September 29, 2025

Feature

You can create BigQuery [non-incremental materialized views over Spanner data](https://docs.cloud.google.com/bigquery/docs/materialized-views-create#spanner) to improve query performance by periodically caching results. This feature is in [Preview](https://cloud.google.com/products#product-launch-stages) .

## September 25, 2025

Feature

The Cassandra interface for Spanner is generally available. The Cassandra interface lets you take advantage of Spanner's fully managed, scalable, and highly available infrastructure using familiar Cassandra tools and syntax. For more information, see [Cassandra interface](https://docs.cloud.google.com/spanner/docs/non-relational/cassandra-overview) , [Migrate from Cassandra to Spanner](https://docs.cloud.google.com/spanner/docs/non-relational/migrate-from-cassandra-to-spanner) , and [Connect to Spanner using the Cassandra interface](https://docs.cloud.google.com/spanner/docs/non-relational/connect-cassandra-adapter) .

## September 23, 2025

Feature

You can use [read lease](https://docs.cloud.google.com/spanner/docs/read-lease) regions to reduce latency for strong reads in multi-region or dual-region instances. Read leases use designated non-leader, read-write or read-only regions to serve strong reads locally, eliminating the network round trip to the leader region that is typically required. This feature is [generally available](https://cloud.google.com/products/#product-launch-stages) (GA).

Feature

You can use the dedicated [Gemini CLI extension for Spanner](https://docs.cloud.google.com/spanner/docs/pre-built-tools-with-mcp-toolbox) to execute SQL statements and query your Spanner instance using natural language controls.

## September 22, 2025

Feature

You can run federated queries against [PostgreSQL dialect databases in Spanner](https://docs.cloud.google.com/spanner/docs/reference/postgresql/overview) using [BigQuery external datasets](https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets) using GoogleSQL; this includes [cross-region federated queries](https://cloud.google.com/bigquery/docs/spanner-federated-queries#cross_region_queries) . This feature is [generally available](https://cloud.google.com/products#product-launch-stages) (GA).

## September 17, 2025

Feature

Spanner Graph support of schemaless schemas is [generally available](https://cloud.google.com/products#product-launch-stages) (GA). For more information, see [Manage schemaless data with Spanner Graph](https://docs.cloud.google.com/spanner/docs/graph/manage-schemaless-data) .

## September 03, 2025

Feature

You can import your own data into a Spanner database by using a CSV file, a MySQL dump file, or a PostgreSQL dump file.

Additionally, you can populate new databases in an existing Spanner instance from sample datasets that help you explore Spanner capabilities such as its relational model, full-text search, vector search, or Spanner Graph.

For more information, see [Create and manage databases](https://docs.cloud.google.com/spanner/docs/create-manage-databases) .

## August 29, 2025

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.84.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.83.0...spanner/v1.84.0) (2025-08-05)

##### Features

- **spanner/adapter:** Add last field in AdaptMessageResponse for internal optimization usage ( [c574e28](https://github.com/googleapis/google-cloud-go/commit/c574e287f49cc1c3b069b35d95b98da2bc9b948f) )
- **spanner/admin/database:** Proto changes for an internal api ( [eeb4b1f](https://github.com/googleapis/google-cloud-go/commit/eeb4b1fe8eb83b73ec31b0bd46e3704bdc0212c3) )
- **spanner:** A new field `snapshot_timestamp` is added to message `.google.spanner.v1.CommitResponse` ( [ac4970b](https://github.com/googleapis/google-cloud-go/commit/ac4970b5a6318dbfcdca7da5ee256852ca49ea23) )
- **spanner:** Add Google Cloud standard otel attributes ( [\#11652](https://github.com/googleapis/google-cloud-go/issues/11652) ) ( [f59fcff](https://github.com/googleapis/google-cloud-go/commit/f59fcfffdfcd01ef5b436b76fa83351e2b695920) )

##### Bug Fixes

- **spanner:** Context cancel in traces in case of skipping trailers ( [\#12635](https://github.com/googleapis/google-cloud-go/issues/12635) ) ( [509dc90](https://github.com/googleapis/google-cloud-go/commit/509dc90cd13061f8302d20451af1d9f7e186641f) )
- **spanner:** Enforce only one resource header ( [\#12618](https://github.com/googleapis/google-cloud-go/issues/12618) ) ( [4e04b7e](https://github.com/googleapis/google-cloud-go/commit/4e04b7efd68a979837f78d94ac1dbc930c2e5efb) )
- **spanner:** Fix blind retry for ResourceExhausted ( [\#12523](https://github.com/googleapis/google-cloud-go/issues/12523) ) ( [f9b6e88](https://github.com/googleapis/google-cloud-go/commit/f9b6e88bd3fce735ea58f70e3a7634837886d393) )
- **spanner:** Remove stream wrapper for direct path check ( [\#12622](https://github.com/googleapis/google-cloud-go/issues/12622) ) ( [88a36cd](https://github.com/googleapis/google-cloud-go/commit/88a36cdfb7f7d1d265f45ed8795b6c08915fe183) )

##### Documentation

- **spanner:** A comment for enum value `OPTIMISTIC` in enum `ReadLockMode` is changed ( [ac4970b](https://github.com/googleapis/google-cloud-go/commit/ac4970b5a6318dbfcdca7da5ee256852ca49ea23) )
- **spanner:** A comment for enum value `PESSIMISTIC` in enum `ReadLockMode` is changed ( [ac4970b](https://github.com/googleapis/google-cloud-go/commit/ac4970b5a6318dbfcdca7da5ee256852ca49ea23) )
- **spanner:** A comment for enum value `READ_LOCK_MODE_UNSPECIFIED` in enum `ReadLockMode` is changed ( [ac4970b](https://github.com/googleapis/google-cloud-go/commit/ac4970b5a6318dbfcdca7da5ee256852ca49ea23) )
- **spanner:** A comment for field `commit_stats` in message `.google.spanner.v1.CommitResponse` is changed ( [ac4970b](https://github.com/googleapis/google-cloud-go/commit/ac4970b5a6318dbfcdca7da5ee256852ca49ea23) )
- **spanner:** A comment for field `exclude_txn_from_change_streams` in message `.google.spanner.v1.TransactionOptions` is changed ( [ac4970b](https://github.com/googleapis/google-cloud-go/commit/ac4970b5a6318dbfcdca7da5ee256852ca49ea23) )
- **spanner:** A comment for field `multiplexed_session_previous_transaction_id` in message `.google.spanner.v1.TransactionOptions` is changed ( [ac4970b](https://github.com/googleapis/google-cloud-go/commit/ac4970b5a6318dbfcdca7da5ee256852ca49ea23) )
- **spanner:** A comment for field `precommit_token` in message `.google.spanner.v1.CommitResponse` is changed ( [ac4970b](https://github.com/googleapis/google-cloud-go/commit/ac4970b5a6318dbfcdca7da5ee256852ca49ea23) )
- **spanner:** A comment for message `.google.spanner.v1.MultiplexedSessionPrecommitToken` is changed ( [ac4970b](https://github.com/googleapis/google-cloud-go/commit/ac4970b5a6318dbfcdca7da5ee256852ca49ea23) )
- **spanner:** A comment for message `.google.spanner.v1.TransactionOptions` is changed ( [ac4970b](https://github.com/googleapis/google-cloud-go/commit/ac4970b5a6318dbfcdca7da5ee256852ca49ea23) )

#### [1.84.1](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.84.0...spanner/v1.84.1) (2025-08-06)

##### Features

- **spanner:** Release 1.84.1 ( [\#12663](https://github.com/googleapis/google-cloud-go/issues/12663) ) ( [8b410ec](https://github.com/googleapis/google-cloud-go/commit/8b410ec689591a591aecb46831f2f50706cb973f) )

##### Miscellaneous Chores

- **spanner:** Release 1.84.1 ( [\#12665](https://github.com/googleapis/google-cloud-go/issues/12665) ) ( [a1ce8c2](https://github.com/googleapis/google-cloud-go/commit/a1ce8c26651e7a0ba4f1b20aba4c0fefbab0b972) )

**DO NOT USE** This version is retracted due to https://github.com/googleapis/google-cloud-go/issues/12659, use version \>=v1.84.1

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.98.0](https://github.com/googleapis/java-spanner/compare/v6.97.1...v6.98.0) (2025-07-31)

##### Features

- Proto changes for an internal api ( [675e90b](https://github.com/googleapis/java-spanner/commit/675e90b4582b4fc968118121e6c23ec98ee178e9) )
- **spanner:** A new field `snapshot_timestamp` is added to message `.google.spanner.v1.CommitResponse` ( [675e90b](https://github.com/googleapis/java-spanner/commit/675e90b4582b4fc968118121e6c23ec98ee178e9) )
- Support Exemplar ( [\#3997](https://github.com/googleapis/java-spanner/issues/3997) ) ( [fcf0a01](https://github.com/googleapis/java-spanner/commit/fcf0a0182a33f229e865e4593635efaed34d6dac) )
- Use multiplex sessions for RW and Partition Ops ( [\#3996](https://github.com/googleapis/java-spanner/issues/3996) ) ( [a882204](https://github.com/googleapis/java-spanner/commit/a882204e07a2084b228c14fb37ac53e4e33d0f59) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.60.2 ( [675e90b](https://github.com/googleapis/java-spanner/commit/675e90b4582b4fc968118121e6c23ec98ee178e9) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.50.2 ( [\#4004](https://github.com/googleapis/java-spanner/issues/4004) ) ( [986c0e0](https://github.com/googleapis/java-spanner/commit/986c0e07fddecd51cd310a9759ce1d41c1f5c657) )

#### [6.98.1](https://github.com/googleapis/java-spanner/compare/v6.98.0...v6.98.1) (2025-08-11)

##### Bug Fixes

- Add missing span.end calls for AsyncTransactionManager ( [\#4012](https://github.com/googleapis/java-spanner/issues/4012) ) ( [1a4adb4](https://github.com/googleapis/java-spanner/commit/1a4adb4d70c3a3822fa6bda93d689f2dae1835fa) )
- **deps:** Update the Java code generator (gapic-generator-java) to 2.61.0 ( [8156ef3](https://github.com/googleapis/java-spanner/commit/8156ef31d93932c14f9fdd13c8c5e5b7ce370ba5) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.51.0 ( [\#4013](https://github.com/googleapis/java-spanner/issues/4013) ) ( [4e90c29](https://github.com/googleapis/java-spanner/commit/4e90c29ce3447d14411368e45a39c7b0965cb40a) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [8.1.0](https://github.com/googleapis/nodejs-spanner/compare/v8.0.0...v8.1.0) (2025-07-28)

##### Features

- Add Custom OpenTelemetry Exporter in for Service Metrics ( [\#2272](https://github.com/googleapis/nodejs-spanner/issues/2272) ) ( [610d1b9](https://github.com/googleapis/nodejs-spanner/commit/610d1b989ba186c0758791343deaa7f683c4bd26) )
- Add methods from gax to cache proto root and process custom error details ( [\#2330](https://github.com/googleapis/nodejs-spanner/issues/2330) ) ( [1b3931a](https://github.com/googleapis/nodejs-spanner/commit/1b3931a799bdd052adc91703e59e1d0c83270065) )
- Add metrics tracers ( [\#2319](https://github.com/googleapis/nodejs-spanner/issues/2319) ) ( [192bf2b](https://github.com/googleapis/nodejs-spanner/commit/192bf2bb603bca4ac481fcfd1f04974173adc6a1) )
- Add support for AFE latency metrics ( [\#2348](https://github.com/googleapis/nodejs-spanner/issues/2348) ) ( [0666f05](https://github.com/googleapis/nodejs-spanner/commit/0666f05d589e2f229b44dffae8e9649220bccf8b) )
- Add throughput_mode to UpdateDatabaseDdlRequest to be used by Spanner Migration Tool. See https://github.com/GoogleCloudPlatform/spanner-migration-tool ( [\#2304](https://github.com/googleapis/nodejs-spanner/issues/2304) ) ( [a29af56](https://github.com/googleapis/nodejs-spanner/commit/a29af56ae3c31f07115cb938bcf3f0f77241b725) )
- Operation, Attempt, and GFE metrics ( [\#2328](https://github.com/googleapis/nodejs-spanner/issues/2328) ) ( [646e6ea](https://github.com/googleapis/nodejs-spanner/commit/646e6ea6f1dc5fa1937e512ae9e81ae4d2637ed0) )
- Proto changes for an internal api ( [\#2356](https://github.com/googleapis/nodejs-spanner/issues/2356) ) ( [380e770](https://github.com/googleapis/nodejs-spanner/commit/380e7705a23a692168db386ba5426c91bf1587b6) )
- **spanner:** A new field `snapshot_timestamp` is added to message `.google.spanner.v1.CommitResponse` ( [\#2350](https://github.com/googleapis/nodejs-spanner/issues/2350) ) ( [0875cd8](https://github.com/googleapis/nodejs-spanner/commit/0875cd82e99fa6c95ab38807e09c5921303775f8) )
- **spanner:** Add new change_stream.proto ( [\#2315](https://github.com/googleapis/nodejs-spanner/issues/2315) ) ( [57d67be](https://github.com/googleapis/nodejs-spanner/commit/57d67be2e3b6d6ac2a8a903acf8613b27a049c3b) )
- **spanner:** Add tpc support ( [\#2333](https://github.com/googleapis/nodejs-spanner/issues/2333) ) ( [a381cab](https://github.com/googleapis/nodejs-spanner/commit/a381cab92c31373a6a10edca0f8a8bdfc4415e4b) )
- Track precommit token in r/w apis(multiplexed session) ( [\#2312](https://github.com/googleapis/nodejs-spanner/issues/2312) ) ( [3676bfa](https://github.com/googleapis/nodejs-spanner/commit/3676bfa60725c43f85a04ead87943be92e4a99f0) )

##### Bug Fixes

- Docs-test ( [\#2297](https://github.com/googleapis/nodejs-spanner/issues/2297) ) ( [61c571c](https://github.com/googleapis/nodejs-spanner/commit/61c571c729c2a065df6ff166db784a6e6eaef74d) )
- Ensure context propagation works in Node.js 22 with async/await ( [\#2326](https://github.com/googleapis/nodejs-spanner/issues/2326) ) ( [e8cdbed](https://github.com/googleapis/nodejs-spanner/commit/e8cdbedd55f049b8c7766e97388ed045fedd1b4e) )
- Pass the Span correctly ( [\#2332](https://github.com/googleapis/nodejs-spanner/issues/2332) ) ( [edaee77](https://github.com/googleapis/nodejs-spanner/commit/edaee7791b2d814f749ed35119dd705924984a78) )
- System test against emulator ( [\#2339](https://github.com/googleapis/nodejs-spanner/issues/2339) ) ( [2a6af4c](https://github.com/googleapis/nodejs-spanner/commit/2a6af4c36484f44929a2fac80d8f225dad5d702c) )
- Unhandled exceptions from gax ( [\#2338](https://github.com/googleapis/nodejs-spanner/issues/2338) ) ( [6428bcd](https://github.com/googleapis/nodejs-spanner/commit/6428bcd2980852c1bdbc4c3d0ab210a139e5f193) )

##### Performance Improvements

- Skip gRPC trailers for StreamingRead & ExecuteStreamingSql ( [\#2313](https://github.com/googleapis/nodejs-spanner/issues/2313) ) ( [8bd0781](https://github.com/googleapis/nodejs-spanner/commit/8bd0781e8b434a421f0e0f3395439a5a86c7847c) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.57.0](https://github.com/googleapis/python-spanner/compare/v3.56.0...v3.57.0) (2025-08-14)

##### Features

- Support configuring logger in dbapi kwargs ( [\#1400](https://github.com/googleapis/python-spanner/issues/1400) ) ( [ffa5c9e](https://github.com/googleapis/python-spanner/commit/ffa5c9e627583ab0635dcaa5512b6e034d811d86) )

## August 25, 2025

Feature

You can terminate multiple active queries in your Spanner instance. Active queries are long-running queries that might affect the performance of your instance. Monitoring these queries can help you identify causes of instance latency and high CPU usage. Terminating queries might help free up resources and reduce the load on your instance.

For more information, see [Monitor active queries](https://docs.cloud.google.com/spanner/docs/monitor-active-queries#terminate-query) .

## August 14, 2025

Feature

You can use [cross region federated queries](https://cloud.google.com/bigquery/docs/spanner-federated-queries#cross_region_queries) to query Spanner tables from regions other than the source BigQuery region. These cross region queries incur additional [Spanner network egress charges](https://docs.cloud.google.com/spanner/pricing#network) . This feature is [generally available](https://cloud.google.com/products#product-launch-stages) (GA).

## August 13, 2025

Feature

Spanner offers a predefined library of over [80 MySQL functions](https://docs.cloud.google.com/spanner/docs/reference/mysql/user_defined_functions_all) that you can install in a database. These functions let you perform operations that are common in the MySQL environments directly with Spanner. They can help reduce the changes required when migrating workloads from MySQL to Spanner.

These functions are packaged as user-defined functions that can be installed from an open-source DDL script hosted on GitHub. For more information, see [Install MySQL functions in Spanner](https://docs.cloud.google.com/spanner/docs/install-mysql-functions) .

## August 05, 2025

Feature

Columnar engine for Spanner is in Preview. Columnar engine is a storage technique used with analytics queries to speed up scans. Spanner columnar engine accelerates analytical query performance on live operational data by up to 200 times without affecting transaction workloads. This eliminates the need for ETL into separate data warehouses while maintaining strong consistency. For more information, see the [Columnar engine for Spanner overview](https://docs.cloud.google.com/spanner/docs/columnar-engine) .

## August 01, 2025

Feature

When you create the free trial instance using the Google Cloud console, Spanner creates and preloads it with a sample database for an ecommerce store. You can use the free trial instance to explore the dataset and learn about Spanner capabilities with pre-loaded queries.

For more information, see [Spanner free trial instances](https://docs.cloud.google.com/spanner/docs/free-trial-instance) .

## July 31, 2025

Feature

You can use [continuous queries](https://docs.cloud.google.com/bigquery/docs/continuous-queries-introduction) to [export BigQuery data into Spanner in real time](https://docs.cloud.google.com/bigquery/docs/export-to-spanner) . This feature is in [preview](https://cloud.google.com/products#product-launch-stages) .

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.96.1](https://github.com/googleapis/java-spanner/compare/v6.96.0...v6.96.1) (2025-06-30)

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.59.0 ( [2836042](https://github.com/googleapis/java-spanner/commit/2836042217fe29bb967fe892bd6b492391ded95c) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.50.0 ( [\#3925](https://github.com/googleapis/java-spanner/issues/3925) ) ( [1372bbd](https://github.com/googleapis/java-spanner/commit/1372bbd82b7828629cbc407b78878469bc477977) )

#### [6.97.0](https://github.com/googleapis/java-spanner/compare/v6.96.1...v6.97.0) (2025-07-10)

##### Features

- Next release from main branch is 6.97.0 ( [\#3984](https://github.com/googleapis/java-spanner/issues/3984) ) ( [5651f61](https://github.com/googleapis/java-spanner/commit/5651f6160e1e655f118aa2e7f0203a47cd6914c0) )

##### Bug Fixes

- Drop max message size ( [\#3987](https://github.com/googleapis/java-spanner/issues/3987) ) ( [3eee899](https://github.com/googleapis/java-spanner/commit/3eee89965547dfa49b4282b470f625d43c92f4fd) )
- Return non-empty metadata for DataBoost queries ( [\#3936](https://github.com/googleapis/java-spanner/issues/3936) ) ( [79c0684](https://github.com/googleapis/java-spanner/commit/79c06848c0ac4eff8410dd3bd63db8675c202d94) )

#### [6.97.1](https://github.com/googleapis/java-spanner/compare/v6.97.0...v6.97.1) (2025-07-15)

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.50.1 ( [\#3992](https://github.com/googleapis/java-spanner/issues/3992) ) ( [69ffd72](https://github.com/googleapis/java-spanner/commit/69ffd7282220b8b12c6b9b64d8856ff88068ffa2) )
- Update googleapis/sdk-platform-java action to v2.60.1 ( [\#3926](https://github.com/googleapis/java-spanner/issues/3926) ) ( [7001b7f](https://github.com/googleapis/java-spanner/commit/7001b7faaff581e26ec81c4db2c99a1e8726d5eb) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.56.0](https://github.com/googleapis/python-spanner/compare/v3.55.0...v3.56.0) (2025-07-24)

##### Features

- Add support for multiplexed sessions - read/write ( [\#1389](https://github.com/googleapis/python-spanner/issues/1389) ) ( [ce3f230](https://github.com/googleapis/python-spanner/commit/ce3f2305cd5589e904daa18142fbfeb180f3656a) )
- Add support for multiplexed sessions ( [\#1383](https://github.com/googleapis/python-spanner/issues/1383) ) ( [21f5028](https://github.com/googleapis/python-spanner/commit/21f5028c3fdf8b8632c1564efbd973b96711d03b) )
- Default enable multiplex session for all operations unless explicitly set to false ( [\#1394](https://github.com/googleapis/python-spanner/issues/1394) ) ( [651ca9c](https://github.com/googleapis/python-spanner/commit/651ca9cd65c713ac59a7d8f55b52b9df5b4b6923) )
- **spanner:** Add new change_stream.proto ( [\#1382](https://github.com/googleapis/python-spanner/issues/1382) ) ( [ca6255e](https://github.com/googleapis/python-spanner/commit/ca6255e075944d863ab4be31a681fc7c27817e34) )

##### Performance Improvements

- Skip gRPC trailers for StreamingRead & ExecuteStreamingSql ( [\#1385](https://github.com/googleapis/python-spanner/issues/1385) ) ( [cb25de4](https://github.com/googleapis/python-spanner/commit/cb25de40b86baf83d0fb1b8ca015f798671319ee) )

## July 14, 2025

Feature

[Spanner Data Boost](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) supports data stored on hard disk drives (HDD). This feature is [generally available](https://cloud.google.com/products/#product-launch-stages) (GA).

## July 01, 2025

Change

The performance of the `ANY` and the [`ANY SHORTEST`](https://docs.cloud.google.com/spanner/docs/graph/work-with-paths#any-shortest) algorithms have been improved. These algorithms are used to find Spanner Graph paths. For more information, see [`ANY` and `ANY SHORTEST` paths](https://docs.cloud.google.com/spanner/docs/graph/queries-overview#any-and-any-shortest-paths) .

## June 30, 2025

Feature

Spanner supports the following new client-side metrics to the Cloud Spanner API frontend (AFE) and Google frontend (GFE) for Java and Go applications:

- AFE connectivity error count
- AFE latencies
- GFE connectivity error count
- GFE latencies

These metrics can be used with server-side metrics to enable faster troubleshooting of performance and latency issues. For more information, see [Client-side metrics descriptions](https://docs.cloud.google.com/spanner/docs/client-side-metrics-descriptions) .

Feature

To troubleshoot or understand your Spanner queries better, you can download and save your [query execution plan](https://docs.cloud.google.com/spanner/docs/query-execution-plans) as a JSON file. You can use the content of this file to see a visualization of the query execution plan in Spanner Studio. For more information, see [Take a tour of the query plan visualizer](https://docs.cloud.google.com/spanner/docs/tune-query-with-visualizer#visual-plan-tour) .

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.83.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.82.0...spanner/v1.83.0) (2025-06-27)

##### Features

- **spanner/spansql:** Add support for TOKENIZE_JSON. ( [\#12338](https://github.com/googleapis/google-cloud-go/issues/12338) ) ( [72225a5](https://github.com/googleapis/google-cloud-go/commit/72225a52c0f6bc6eac6d4e450dad0b566e79553f) )
- **spanner/spansql:** Support EXISTS in query parsing ( [\#12439](https://github.com/googleapis/google-cloud-go/issues/12439) ) ( [f5cb67b](https://github.com/googleapis/google-cloud-go/commit/f5cb67b104e4d99196064fb4474e0644e90c9a00) )
- **spanner:** Add new change_stream.proto ( [40b60a4](https://github.com/googleapis/google-cloud-go/commit/40b60a4b268040ca3debd71ebcbcd126b5d58eaa) )
- **spanner:** Add option for how to call BeginTransaction ( [\#12436](https://github.com/googleapis/google-cloud-go/issues/12436) ) ( [2cba13b](https://github.com/googleapis/google-cloud-go/commit/2cba13b8fef80b6cb5980e3b5b2bfc6dc796a03e) )
- **spanner:** Wrap proto mutation ( [\#12497](https://github.com/googleapis/google-cloud-go/issues/12497) ) ( [e655889](https://github.com/googleapis/google-cloud-go/commit/e655889e2fd6a55f901d1d8b146e7aa5efdca705) )

##### Bug Fixes

- **spanner:** Pointer type custom struct decoder ( [\#12496](https://github.com/googleapis/google-cloud-go/issues/12496) ) ( [ac3cafb](https://github.com/googleapis/google-cloud-go/commit/ac3cafbac435a3ac98fd4693bac84a3f4a260c5b) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.95.0](https://github.com/googleapis/java-spanner/compare/v6.94.0...v6.95.0) (2025-06-05)

##### Features

- Enable ALTS hard bound token in DirectPath ( [\#3904](https://github.com/googleapis/java-spanner/issues/3904) ) ( [2b0f2ff](https://github.com/googleapis/java-spanner/commit/2b0f2ff214f4b68dd5957bc4280edb713b77a763) )
- Enable grpc and afe metrics ( [\#3896](https://github.com/googleapis/java-spanner/issues/3896) ) ( [706f794](https://github.com/googleapis/java-spanner/commit/706f794f044c2cb1112cfdae6f379e5f2bc3f26f) )
- Last statement sample ( [\#3830](https://github.com/googleapis/java-spanner/issues/3830) ) ( [2f62816](https://github.com/googleapis/java-spanner/commit/2f62816b0af9aced1b73e25525f60f8e3e923454) )
- **spanner:** Add new change_stream.proto ( [f385698](https://github.com/googleapis/java-spanner/commit/f38569865de7465ae9a37b844a9dd983571d3688) )

##### Bug Fixes

- Directpath_enabled attribute ( [\#3897](https://github.com/googleapis/java-spanner/issues/3897) ) ( [53bc510](https://github.com/googleapis/java-spanner/commit/53bc510145921d00bc3df04aa4cf407179ed8d8e) )

##### Dependencies

- Update dependency io.opentelemetry:opentelemetry-bom to v1.50.0 ( [\#3887](https://github.com/googleapis/java-spanner/issues/3887) ) ( [94b879c](https://github.com/googleapis/java-spanner/commit/94b879c8c1848fa0b14dbe8cda8390cfe9e8fce6) )

#### [6.95.1](https://github.com/googleapis/java-spanner/compare/v6.95.0...v6.95.1) (2025-06-06)

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.49.0 ( [\#3909](https://github.com/googleapis/java-spanner/issues/3909) ) ( [3de8502](https://github.com/googleapis/java-spanner/commit/3de8502b98ebb90526fc2339e279f9b710816b3b) )
- Update googleapis/sdk-platform-java action to v2.59.0 ( [\#3910](https://github.com/googleapis/java-spanner/issues/3910) ) ( [aed8bd6](https://github.com/googleapis/java-spanner/commit/aed8bd6d5a0b1e0dfab345e0de68f285e8b8aedb) )

#### [6.96.0](https://github.com/googleapis/java-spanner/compare/v6.95.1...v6.96.0) (2025-06-27)

##### Features

- Allow JDBC to configure directpath for connection ( [\#3929](https://github.com/googleapis/java-spanner/issues/3929) ) ( [d754f1f](https://github.com/googleapis/java-spanner/commit/d754f1f99294d86ec881583f217fa09f291a3d7a) )
- Support getOrNull and getOrDefault in Struct ( [\#3914](https://github.com/googleapis/java-spanner/issues/3914) ) ( [1dc5a3e](https://github.com/googleapis/java-spanner/commit/1dc5a3ec0ca9ea530e8691df5c2734c0a1ece559) )
- Use multiplexed sessions for read-only transactions ( [\#3917](https://github.com/googleapis/java-spanner/issues/3917) ) ( [37fdc27](https://github.com/googleapis/java-spanner/commit/37fdc27aab4e71ac141c2a2c979f864e97395a97) )

##### Bug Fixes

- Allow zero durations to be set for connections ( [\#3916](https://github.com/googleapis/java-spanner/issues/3916) ) ( [43ea4fa](https://github.com/googleapis/java-spanner/commit/43ea4fa68eac00801beb8e58c1eb09e9f32e5ce5) )

##### Documentation

- Add snippet for Repeatable Read configuration at client and transaction ( [\#3908](https://github.com/googleapis/java-spanner/issues/3908) ) ( [ff3d212](https://github.com/googleapis/java-spanner/commit/ff3d212c98276c4084f44619916d0444c9652803) )
- Update SpannerSample.java to align with best practices ( [\#3625](https://github.com/googleapis/java-spanner/issues/3625) ) ( [7bfc62d](https://github.com/googleapis/java-spanner/commit/7bfc62d3d9e57242e0dfddea090208f8c65f0f8e) )

## June 24, 2025

Feature

You can directly connect and interact with your Spanner database using the Spanner CLI, an interactive shell for Spanner that is built into the Google Cloud CLI. You can use the Spanner CLI to start an interactive session and automate SQL executions from the shell or an input file. This feature is available in [Preview](https://cloud.google.com/products#product-launch-stages) . For more information, see [Spanner CLI quickstart](https://docs.cloud.google.com/spanner/docs/spanner-cli) .

## June 20, 2025

Feature

A free trial creation workflow is available that makes it easier to start your Spanner free trial. With a free trial instance, you can learn and explore Spanner for 90 days at no cost. You can create relational (GoogleSQL and PostgreSQL) databases and deploy NoSQL models (Spanner Graph, vector search, and full-text search) in a single, fully managed database. For more information, see [Spanner free trial instances overview](https://docs.cloud.google.com/spanner/docs/free-trial-instance) .

## June 11, 2025

Feature

Column operations statistics are [generally available](https://cloud.google.com/products#product-launch-stages) . They help you get insights into and monitor the usage of columns in your database. For more information, see [Column operations statistics](https://docs.cloud.google.com/spanner/docs/introspection/column-operations-statistics) .

## June 02, 2025

Feature

BigQuery supports using [Spanner external datasets](https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets) with [authorized views](https://docs.cloud.google.com/bigquery/docs/authorized-views) , [authorized routines](https://docs.cloud.google.com/bigquery/docs/authorized-routines) , and [cloud resource connections](https://docs.cloud.google.com/bigquery/docs/create-cloud-resource-connection) . This feature is [generally available](https://cloud.google.com/products#product-launch-stages) (GA).

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.81.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.80.0...spanner/v1.81.0) (2025-05-09)

##### Features

- **spanner/spansql:** Add support for DROP SEARCH INDEX and ALTER SEARCH INDEX ( [\#11961](https://github.com/googleapis/google-cloud-go/issues/11961) ) ( [952cd7f](https://github.com/googleapis/google-cloud-go/commit/952cd7fd419af9eb74f5d30a111ae936094b0645) )

##### Bug Fixes

- **spanner:** Row mismatch in SelectAll using custom type ( [\#12222](https://github.com/googleapis/google-cloud-go/issues/12222) ) ( [ce6a23a](https://github.com/googleapis/google-cloud-go/commit/ce6a23a45fe66cc12e1b5014d2d45f1968ddc067) )

#### [1.81.1](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.81.0...spanner/v1.81.1) (2025-05-15)

##### Features

- **spanner:** Add support of AFE and GRPC metrics in client-side metrics ( [\#12067](https://github.com/googleapis/google-cloud-go/issues/12067) ) ( [7b77038](https://github.com/googleapis/google-cloud-go/commit/7b77038eb4afe31b1a0d42f7c35aeabce0f48810) )

#### [1.82.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.81.1...spanner/v1.82.0) (2025-05-20)

##### Features

- **spanner/admin/database:** Add throughput_mode to UpdateDatabaseDdlRequest to be used by Spanner Migration Tool. See https ( [\#12287](https://github.com/googleapis/google-cloud-go/issues/12287) ) ( [2a9d8ee](https://github.com/googleapis/google-cloud-go/commit/2a9d8eec71a7e6803eb534287c8d2f64903dcddd) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.92.0](https://github.com/googleapis/java-spanner/compare/v6.91.1...v6.92.0) (2025-04-29)

- **spanner:** Do not export metrics during shutdown if prev export was less than 30 seconds ago ( [\#12266](https://github.com/googleapis/google-cloud-go/issues/12266) ) ( [8ad7511](https://github.com/googleapis/google-cloud-go/commit/8ad75111433be5424f9fff8aafd73463cb467734) )
- **spanner:** Fix invalid trace in case of skipping trailers ( [\#12235](https://github.com/googleapis/google-cloud-go/issues/12235) ) ( [e54c439](https://github.com/googleapis/google-cloud-go/commit/e54c4398831b5a1c2998f9e8d159f0118aee1d0b) ) \### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner) \#### [6.94.0](https://github.com/googleapis/java-spanner/compare/v6.93.0...v6.94.0) (2025-05-21)

##### Features

- \[Internal\] client-side metrics for afe latency and connectivity error ( [\#3819](https://github.com/googleapis/java-spanner/issues/3819) ) ( [a8dba0a](https://github.com/googleapis/java-spanner/commit/a8dba0a83939fdbbc324f0a7aa6c44180462fa3a) )
- Support begin with AbortedException for manager interface ( [\#3835](https://github.com/googleapis/java-spanner/issues/3835) ) ( [5783116](https://github.com/googleapis/java-spanner/commit/578311693bed836c8916f4b4ffa0782a468c1af3) )
- Add throughput_mode to UpdateDatabaseDdlRequest to be used by Spanner Migration Tool. See https://github.com/GoogleCloudPlatform/spanner-migration-tool ( [3070f1d](https://github.com/googleapis/java-spanner/commit/3070f1db97788c2a55c553ab8a4de3419d1ccf5c) )
- Enable AFE and gRPC metrics for DP ( [\#3852](https://github.com/googleapis/java-spanner/issues/3852) ) ( [203baae](https://github.com/googleapis/java-spanner/commit/203baae3996378435095cb90e3b2c7ee71a643cd) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.56.2 ( [11bfd90](https://github.com/googleapis/java-spanner/commit/11bfd90daa244dbd31a76bc5a1d2e694e43fa292) )
- **deps:** Update the Java code generator (gapic-generator-java) to 2.58.0 ( [3070f1d](https://github.com/googleapis/java-spanner/commit/3070f1db97788c2a55c553ab8a4de3419d1ccf5c) )
- Remove trailing semicolons in DDL ( [\#3879](https://github.com/googleapis/java-spanner/issues/3879) ) ( [ca3a67d](https://github.com/googleapis/java-spanner/commit/ca3a67db715f398943382df1f8a9979905811ff8) )
- Change server timing duration attribute to float as per w3c ( [\#3851](https://github.com/googleapis/java-spanner/issues/3851) ) ( [da8dd8d](https://github.com/googleapis/java-spanner/commit/da8dd8da3171a073d7b450d4413936351a4c1060) )
- **deps:** Update the Java code generator (gapic-generator-java) to 2.57.0 ( [23b985c](https://github.com/googleapis/java-spanner/commit/23b985c9a04837b0b38f2cfc5d96469e1d664d67) )
- Non-ASCII Unicode characters in code ( [\#3844](https://github.com/googleapis/java-spanner/issues/3844) ) ( [85a0820](https://github.com/googleapis/java-spanner/commit/85a0820505889ae6482a9e4f845cd53430dd6b44) )
- Only close and return sessions once ( [\#3846](https://github.com/googleapis/java-spanner/issues/3846) ) ( [32b2373](https://github.com/googleapis/java-spanner/commit/32b2373d62cac3047d9686c56af278c706d7c488) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.46.2 ( [\#3836](https://github.com/googleapis/java-spanner/issues/3836) ) ( [2ee7f97](https://github.com/googleapis/java-spanner/commit/2ee7f971f3374b01d22e5a7f8f2483cf60c3363d) )

#### [6.93.0](https://github.com/googleapis/java-spanner/compare/v6.92.0...v6.93.0) (2025-05-09)

- Update dependency com.google.cloud:sdk-platform-java-config to v3.48.0 ( [\#3869](https://github.com/googleapis/java-spanner/issues/3869) ) ( [afa17f7](https://github.com/googleapis/java-spanner/commit/afa17f73beab80639467916bc73b5c96305093aa) )
- Update dependency com.google.cloud:sdk-platform-java-config to v3.48.0 ( [\#3880](https://github.com/googleapis/java-spanner/issues/3880) ) ( [f3b00b6](https://github.com/googleapis/java-spanner/commit/f3b00b663aa897fda1bc21222d29726e6be630cb) )
- Update dependency com.google.cloud.opentelemetry:exporter-metrics to v0.34.0 ( [\#3861](https://github.com/googleapis/java-spanner/issues/3861) ) ( [676b14f](https://github.com/googleapis/java-spanner/commit/676b14f916dea783b40ddec4061bd7af157b5d98) )
- Update dependency commons-io:commons-io to v2.19.0 ( [\#3863](https://github.com/googleapis/java-spanner/issues/3863) ) ( [80a6af8](https://github.com/googleapis/java-spanner/commit/80a6af836ca29ec196a2f509831e1d36c557168f) )
- Update dependency io.opentelemetry:opentelemetry-bom to v1.50.0 ( [\#3865](https://github.com/googleapis/java-spanner/issues/3865) ) ( [ae63050](https://github.com/googleapis/java-spanner/commit/ae6305089b394be0c1eaf8ff7e188711288d87ad) )
- Update googleapis/sdk-platform-java action to v2.58.0 ( [\#3870](https://github.com/googleapis/java-spanner/issues/3870) ) ( [d1e45fa](https://github.com/googleapis/java-spanner/commit/d1e45fa88bb005529bcfb2a6ff2df44065be0fd2) )
- Update opentelemetry.version to v1.50.0 ( [\#3866](https://github.com/googleapis/java-spanner/issues/3866) ) ( [f7e09b8](https://github.com/googleapis/java-spanner/commit/f7e09b8148c0e51503255694bd3347c637724b34) )

##### Documentation

- Add samples for unnamed (positional) parameters ( [\#3849](https://github.com/googleapis/java-spanner/issues/3849) ) ( [035cadd](https://github.com/googleapis/java-spanner/commit/035cadd5bb77a8f9f6fb25ac8c8e5a3e186d9a22) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [8.0.0](https://github.com/googleapis/nodejs-spanner/compare/v7.21.0...v8.0.0) (2025-05-12)

##### ⚠ BREAKING CHANGES

- remove the arrify package ( [\#2292](https://github.com/googleapis/nodejs-spanner/issues/2292) )
- migrate to Node 18 ( [\#2271](https://github.com/googleapis/nodejs-spanner/issues/2271) )

##### Features

- Add promise based signatures for createQueryPartitions ( [\#2284](https://github.com/googleapis/nodejs-spanner/issues/2284) ) ( [255d8a6](https://github.com/googleapis/nodejs-spanner/commit/255d8a6a5749b6a05cd87dd7444cab7dd75d3e42) )
- Add promise based signatures on createReadPartitions ( [\#2300](https://github.com/googleapis/nodejs-spanner/issues/2300) ) ( [7b8a1f7](https://github.com/googleapis/nodejs-spanner/commit/7b8a1f70f0de3aa5886a2cde9325c9a36222a311) )
- Support promise based signatures for execute method ( [\#2301](https://github.com/googleapis/nodejs-spanner/issues/2301) ) ( [bb857e1](https://github.com/googleapis/nodejs-spanner/commit/bb857e18459f717d67b9b3d144c2b022178363cb) )

##### Bug Fixes

- **deps:** Update dependency @google-cloud/kms to v5 ( [\#2289](https://github.com/googleapis/nodejs-spanner/issues/2289) ) ( [1ccb505](https://github.com/googleapis/nodejs-spanner/commit/1ccb505935e70b6f576f06e566325146ee68f3ff) )
- **deps:** Update dependency @google-cloud/precise-date to v5 ( [\#2290](https://github.com/googleapis/nodejs-spanner/issues/2290) ) ( [44f7575](https://github.com/googleapis/nodejs-spanner/commit/44f7575efd3751d0595beef2ec4eb9f39bc426d7) )
- **deps:** Update dependency big.js to v7 ( [\#2286](https://github.com/googleapis/nodejs-spanner/issues/2286) ) ( [0911297](https://github.com/googleapis/nodejs-spanner/commit/0911297cc33aec93c09ef2be42413f20c75fc2bf) )

##### Miscellaneous Chores

- Migrate to Node 18 ( [\#2271](https://github.com/googleapis/nodejs-spanner/issues/2271) ) ( [cab3f22](https://github.com/googleapis/nodejs-spanner/commit/cab3f229ccb2189bd5af0c25a3006b553f8a5453) )
- Remove the arrify package ( [\#2292](https://github.com/googleapis/nodejs-spanner/issues/2292) ) ( [e8f5ca1](https://github.com/googleapis/nodejs-spanner/commit/e8f5ca15125d570949769e6e66f0d911cb21f58d) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.54.0](https://github.com/googleapis/python-spanner/compare/v3.53.0...v3.54.0) (2025-04-28)

##### Features

- Add interval type support ( [\#1340](https://github.com/googleapis/python-spanner/issues/1340) ) ( [6ca9b43](https://github.com/googleapis/python-spanner/commit/6ca9b43c3038eca1317c7c9b7e3543b5f1bc68ad) )
- Add sample for pre-split feature ( [\#1333](https://github.com/googleapis/python-spanner/issues/1333) ) ( [ca76108](https://github.com/googleapis/python-spanner/commit/ca76108809174e4f3eea38d7ac2463d9b4c73304) )
- Add SQL statement for begin transaction isolation level ( [\#1331](https://github.com/googleapis/python-spanner/issues/1331) ) ( [3ac0f91](https://github.com/googleapis/python-spanner/commit/3ac0f9131b38e5cfb2b574d3d73b03736b871712) )
- Support transaction isolation level in dbapi ( [\#1327](https://github.com/googleapis/python-spanner/issues/1327) ) ( [03400c4](https://github.com/googleapis/python-spanner/commit/03400c40f1c1cc73e51733f2a28910a8dd78e7d9) )

##### Bug Fixes

- Improve client-side regex statement parser ( [\#1328](https://github.com/googleapis/python-spanner/issues/1328) ) ( [b3c259d](https://github.com/googleapis/python-spanner/commit/b3c259deec817812fd8e4940faacf4a927d0d69c) )

#### [3.55.0](https://github.com/googleapis/python-spanner/compare/v3.54.0...v3.55.0) (2025-05-28)

##### Features

- Add a `last` field in the `PartialResultSet` ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- Add support for multiplexed sessions ( [\#1381](https://github.com/googleapis/python-spanner/issues/1381) ) ( [97d7268](https://github.com/googleapis/python-spanner/commit/97d7268ac12a57d9d116ee3d9475580e1e7e07ae) )
- Add throughput_mode to UpdateDatabaseDdlRequest to be used by Spanner Migration Tool. See https://github.com/GoogleCloudPlatform/spanner-migration-tool ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- Support fine-grained permissions database roles in connect ( [\#1338](https://github.com/googleapis/python-spanner/issues/1338) ) ( [064d9dc](https://github.com/googleapis/python-spanner/commit/064d9dc3441a617cbc80af6e16493bc42c89b3c9) )

##### Bug Fixes

- E2E tracing metadata append issue ( [\#1357](https://github.com/googleapis/python-spanner/issues/1357) ) ( [3943885](https://github.com/googleapis/python-spanner/commit/394388595a312f60b423dfbfd7aaf2724cc4454f) )
- Pass through kwargs in dbapi connect ( [\#1368](https://github.com/googleapis/python-spanner/issues/1368) ) ( [aae8d61](https://github.com/googleapis/python-spanner/commit/aae8d6161580c88354d813fe75a297c318f1c2c7) )
- Remove setup.cfg configuration for creating universal wheels ( [\#1324](https://github.com/googleapis/python-spanner/issues/1324) ) ( [e064474](https://github.com/googleapis/python-spanner/commit/e0644744d7f3fcea42b461996fc0ee22d4218599) )

##### Documentation

- A comment for field `chunked_value` in message `.google.spanner.v1.PartialResultSet` is changed ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- A comment for field `precommit_token` in message `.google.spanner.v1.PartialResultSet` is changed ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- A comment for field `precommit_token` in message `.google.spanner.v1.ResultSet` is changed ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- A comment for field `query_plan` in message `.google.spanner.v1.ResultSetStats` is changed ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- A comment for field `row_count_lower_bound` in message `.google.spanner.v1.ResultSetStats` is changed ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- A comment for field `row_type` in message `.google.spanner.v1.ResultSetMetadata` is changed ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- A comment for field `rows` in message `.google.spanner.v1.ResultSet` is changed ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- A comment for field `stats` in message `.google.spanner.v1.PartialResultSet` is changed ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- A comment for field `stats` in message `.google.spanner.v1.ResultSet` is changed ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- A comment for field `values` in message `.google.spanner.v1.PartialResultSet` is changed ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- A comment for message `ResultSetMetadata` is changed ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- A comment for message `ResultSetStats` is changed ( [d532d57](https://github.com/googleapis/python-spanner/commit/d532d57fd5908ecd7bc9dfff73695715cc4b1ebe) )
- Fix markdown formatting in transactions page ( [\#1377](https://github.com/googleapis/python-spanner/issues/1377) ) ( [de322f8](https://github.com/googleapis/python-spanner/commit/de322f89642a3c13b6b1d4b9b1a2cdf4c8f550fb) )

## May 27, 2025

Feature

Spanner supports [cross regional federated queries](https://cloud.google.com/bigquery/docs/spanner-federated-queries#cross_region_queries) from BigQuery, which lets BigQuery users query Spanner tables from regions other than their BigQuery region. Users don't incur any Spanner [network egress charges](https://cloud.google.com/spanner/pricing#network) during the preview period. This feature is in [Preview](https://cloud.google.com/products#product-launch-stages) .

## May 26, 2025

Feature

Spanner enables efficient backup copying for [incremental backups](https://docs.cloud.google.com/spanner/docs/backup#incremental-backups) . When you copy an incremental backup, Spanner also copies all older backups in the chain needed to restore the backup. If the destination instance already contains a backup chain that ends with an older backup from the same source chain, Spanner avoids creating redundant copies to save storage and network costs. Spanner copies only the incremental backup and any older backups not present in the destination chain, and appends them to the existing chain.

While Spanner attempts to avoid redundant copies, in rare situations, Spanner might need to copy all the older backups in the chain, even if previously copied backups already exist in the destination instance.

For more information, see [Incremental backups](https://docs.cloud.google.com/spanner/docs/backup#incremental-backups) .

## May 15, 2025

Feature

You can create a pre-filtered vector index that indexes only rows in your database that match a specific filter condition. Using a pre-filtered vector index improves both the performance and recall of approximate nearest neighbor (ANN) searchers by restricting the search to only apply to rows that satisfy the filtering condition. For more information, see [Filter a vector index](https://docs.cloud.google.com/spanner/docs/find-approximate-nearest-neighbors#filter-vector-index) .

## May 13, 2025

Feature

Spanner supports the `INTERVAL` data type in GoogleSQL and PostgreSQL, which represents a duration or an amount of time.

For more information, see [Interval functions in GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/interval_functions) and [PostgreSQL data types](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-types#interval-type) .

Feature

Spanner supports the [`SPLIT_SUBSTR()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#split_substr) GoogleSQL function, which splits an input string using a delimiter and returns a substring composed of a specific number of segments, starting from a given segment index.

Spanner also supports the following GoogleSQL aliases:

- [`ADDDATE()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/date_functions#adddate) : Alias for [`DATE_ADD()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/date_functions#date_add)
- [`SUBDATE()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/date_functions#subdate) : Alias for [`DATE_SUB()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/date_functions#date_sub)
- [`LCASE()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#lcase) : Alias for [`LOWER()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#lower)
- [`UCASE()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#ucase) : Alias for [`UPPER()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#upper)

## May 01, 2025

Feature

Spanner Graph lets you model schemaless data with a dynamic label and properties. For more information, see [Manage schemaless data with Spanner Graph](https://docs.cloud.google.com/spanner/docs/graph/manage-schemaless-data) .

## April 30, 2025

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.79.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.78.0...spanner/v1.79.0) (2025-04-08)

##### Features

- **spanner:** Allow string values for Scan functions ( [\#11898](https://github.com/googleapis/google-cloud-go/issues/11898) ) ( [9989dd0](https://github.com/googleapis/google-cloud-go/commit/9989dd063ba36f39c880b2e7423adde05c703504) )
- **spanner:** New client(s) ( [\#11946](https://github.com/googleapis/google-cloud-go/issues/11946) ) ( [c60f28d](https://github.com/googleapis/google-cloud-go/commit/c60f28d5fd99d5356f39276268a68b729130f152) )

#### 0.1.0 (2025-04-15)

##### Bug Fixes

- **spanner/benchmarks:** Update google.golang.org/api to 0.229.0 ( [3319672](https://github.com/googleapis/google-cloud-go/commit/3319672f3dba84a7150772ccb5433e02dab7e201) )

#### [1.80.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.79.0...spanner/v1.80.0) (2025-04-23)

##### Features

- **spanner:** Add interval type support ( [\#12009](https://github.com/googleapis/google-cloud-go/issues/12009) ) ( [5152488](https://github.com/googleapis/google-cloud-go/commit/5152488d454f332373800134be1bce0e9ecf3505) )

##### Bug Fixes

- **spanner/benchmarks:** Update google.golang.org/api to 0.229.0 ( [3319672](https://github.com/googleapis/google-cloud-go/commit/3319672f3dba84a7150772ccb5433e02dab7e201) )
- **spanner/test/opentelemetry/test:** Update google.golang.org/api to 0.229.0 ( [3319672](https://github.com/googleapis/google-cloud-go/commit/3319672f3dba84a7150772ccb5433e02dab7e201) )
- **spanner:** Retry INTERNAL retriable auth error ( [\#12034](https://github.com/googleapis/google-cloud-go/issues/12034) ) ( [65c7461](https://github.com/googleapis/google-cloud-go/commit/65c7461c06deed3f5b7723a6049e1607f72fcbd4) )
- **spanner:** Update google.golang.org/api to 0.229.0 ( [3319672](https://github.com/googleapis/google-cloud-go/commit/3319672f3dba84a7150772ccb5433e02dab7e201) )

##### Performance Improvements

- **spanner:** Skip gRPC trailers for StreamingRead & ExecuteStreamingSql ( [\#11854](https://github.com/googleapis/google-cloud-go/issues/11854) ) ( [10dc8b7](https://github.com/googleapis/google-cloud-go/commit/10dc8b7e376bdc2ee22378c9334dd6552c135b09) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.90.0](https://github.com/googleapis/java-spanner/compare/v6.89.0...v6.90.0) (2025-03-31)

##### Features

- Add default_isolation_level connection property ( [\#3702](https://github.com/googleapis/java-spanner/issues/3702) ) ( [9472d23](https://github.com/googleapis/java-spanner/commit/9472d23c2b233275e779815f89040323e073a7d1) )
- Adds support for Interval datatype in Java client ( [\#3416](https://github.com/googleapis/java-spanner/issues/3416) ) ( [8be8f5e](https://github.com/googleapis/java-spanner/commit/8be8f5e6b08c8cf3e5f062e4b985b3ec9c725064) )
- Integration test for End to End tracing ( [\#3691](https://github.com/googleapis/java-spanner/issues/3691) ) ( [bf1a07a](https://github.com/googleapis/java-spanner/commit/bf1a07a153b1eb899757260b8ac2bc12384e45af) )
- Specify isolation level per transaction ( [\#3704](https://github.com/googleapis/java-spanner/issues/3704) ) ( [868f30f](https://github.com/googleapis/java-spanner/commit/868f30fde95d07c3fc18feaca64b4d1c3ba6a27d) )
- Support PostgreSQL isolation level statements ( [\#3706](https://github.com/googleapis/java-spanner/issues/3706) ) ( [dda2e1d](https://github.com/googleapis/java-spanner/commit/dda2e1dec38febdad54b61f588590c7572017ba9) )

#### [6.91.0](https://github.com/googleapis/java-spanner/compare/v6.90.0...v6.91.0) (2025-04-17)

##### Features

- \[Internal\] open telemetry built in metrics for GRPC ( [\#3709](https://github.com/googleapis/java-spanner/issues/3709) ) ( [cd76c73](https://github.com/googleapis/java-spanner/commit/cd76c73d838a9ccde2c8c11fc63144a62d76886c) )
- Add java sample for the pre-splitting feature ( [\#3713](https://github.com/googleapis/java-spanner/issues/3713) ) ( [e97b92e](https://github.com/googleapis/java-spanner/commit/e97b92ea4728bc8f013ff73478de4af9eaa1793b) )
- Add TransactionMutationLimitExceededException as cause to SpannerBatchUpdateException ( [\#3723](https://github.com/googleapis/java-spanner/issues/3723) ) ( [4cf5261](https://github.com/googleapis/java-spanner/commit/4cf52613c6c8280fdb864f5b8d04f8fb6ea55e16) )
- Built in metrics for afe latency and connectivity error ( [\#3724](https://github.com/googleapis/java-spanner/issues/3724) ) ( [e13a2f9](https://github.com/googleapis/java-spanner/commit/e13a2f9c5cadd15ab5a565c7dd1c1eec64c09488) )
- Support unnamed parameters ( [\#3820](https://github.com/googleapis/java-spanner/issues/3820) ) ( [1afd815](https://github.com/googleapis/java-spanner/commit/1afd815869785588dfd03ffc12e381e32c4aa0fe) )

##### Bug Fixes

- Add default implementations for Interval methods in AbstractStructReader ( [\#3722](https://github.com/googleapis/java-spanner/issues/3722) ) ( [97f4544](https://github.com/googleapis/java-spanner/commit/97f45448ecb51bd20699d1f163f78b2a7736b21f) )
- Set transaction isolation level had no effect ( [\#3718](https://github.com/googleapis/java-spanner/issues/3718) ) ( [b382999](https://github.com/googleapis/java-spanner/commit/b382999f42d1b643472cf3f605f8c6dc839dec19) )

##### Performance Improvements

- Cache the key used for OTEL traces and metrics ( [\#3814](https://github.com/googleapis/java-spanner/issues/3814) ) ( [c5a2045](https://github.com/googleapis/java-spanner/commit/c5a20452ad2ed5a8f1ac12cca4072a86f4457b93) )
- Optimize parsing in Connection API ( [\#3800](https://github.com/googleapis/java-spanner/issues/3800) ) ( [a2780ed](https://github.com/googleapis/java-spanner/commit/a2780edb3d9d4972c78befd097692f626a6a4bea) )
- Qualify statements without removing comments ( [\#3810](https://github.com/googleapis/java-spanner/issues/3810) ) ( [d358cb9](https://github.com/googleapis/java-spanner/commit/d358cb96e33bdf6de6528d03c884aa702b40b802) )
- Remove all calls to getSqlWithoutComments ( [\#3822](https://github.com/googleapis/java-spanner/issues/3822) ) ( [0e1e14c](https://github.com/googleapis/java-spanner/commit/0e1e14c0e8c1f3726c4d3cfd836c580b3b4122d0) )

#### [6.91.1](https://github.com/googleapis/java-spanner/compare/v6.91.0...v6.91.1) (2025-04-21)

##### Bug Fixes

- SkipHint in the internal parser skipped too much ( [\#3827](https://github.com/googleapis/java-spanner/issues/3827) ) ( [fbf7b4c](https://github.com/googleapis/java-spanner/commit/fbf7b4c4324c4d565bfe3950ecf80de02c88f16e) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.20.0](https://github.com/googleapis/nodejs-spanner/compare/v7.19.1...v7.20.0) (2025-04-11)

##### Features

- Add support for Interval ( [\#2192](https://github.com/googleapis/nodejs-spanner/issues/2192) ) ( [8c886cb](https://github.com/googleapis/nodejs-spanner/commit/8c886cbc0d7523fb99e65cfc5d8f565b630e26f0) )
- **debugging:** Implement x-goog-spanner-request-id propagation per request ( [\#2205](https://github.com/googleapis/nodejs-spanner/issues/2205) ) ( [e42caea](https://github.com/googleapis/nodejs-spanner/commit/e42caeaaa656c395d240f4af412ddb947f29c59b) )
- **spanner:** Add support for snapshot isolation ( [\#2245](https://github.com/googleapis/nodejs-spanner/issues/2245) ) ( [b60a683](https://github.com/googleapis/nodejs-spanner/commit/b60a683c0e1ddbf704766eb99f102fed925a348c) )
- **spanner:** Support for Multiplexed Session Partitioned Ops ( [\#2252](https://github.com/googleapis/nodejs-spanner/issues/2252) ) ( [e7ce471](https://github.com/googleapis/nodejs-spanner/commit/e7ce471332f6e73614638b96ed54c87095d785a2) )

#### [7.21.0](https://github.com/googleapis/nodejs-spanner/compare/v7.20.0...v7.21.0) (2025-04-15)

##### Features

- Adding sample for pre-split feature ( [\#2274](https://github.com/googleapis/nodejs-spanner/issues/2274) ) ( [3d5f080](https://github.com/googleapis/nodejs-spanner/commit/3d5f08065fdf40a1c441d97a049d7dacf1a5be93) )

##### Bug Fixes

- Adding span attributes for request tag and transaction tag ( [\#2236](https://github.com/googleapis/nodejs-spanner/issues/2236) ) ( [3f69dad](https://github.com/googleapis/nodejs-spanner/commit/3f69dad36cfdeb4effd191e0d38079ead1bd6654) )

Feature

The `enhance_query` option on the [`SEARCH`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#search_fulltext) , [`SCORE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#score) , and [`SNIPPET`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#snippet) functions provides automatic synonym matching and spell correction of single words, by default. Previously, if you provided a single word as the search string it likely didn't return any matches and required a phrase with context to perform the enhanced search.

## April 28, 2025

Feature

Manually adding split points to your Spanner database is generally available. Spanner automatically splits, or partitions, data in response to traffic changes to spread load across all available resources in an instance. For large, anticipated traffic changes, such as for a product launch, you can pre-split the database with split boundaries that represent future traffic. This warmup can yield significant performance benefits for large scaling events.

For more information about configuring split points for your database, see [Pre-splitting overview](https://docs.cloud.google.com/spanner/docs/pre-splitting-overview) .

## April 24, 2025

Feature

Spanner lets you use the [`INTERLEAVE IN` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_table) to colocate child rows with parent rows without enforcing the parent-child relationship. When you use `INTERLEAVE IN` (without the `PARENT` option), you can insert child rows before inserting the parent row. You can also delete the parent row without affecting the child rows. For more information, see [Create interleaved tables](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#create-interleaved-tables) and [Indexes and interleaving](https://docs.cloud.google.com/spanner/docs/secondary-indexes#indexes_and_interleaving) .

## April 14, 2025

Feature

End-to-end tracing is generally available ( [GA](https://cloud.google.com/products#product-launch-stages) ). Spanner supports end-to-end tracing, along with client-side tracing in the Node.js and Python client libraries, in addition to Java and Go. For more information, see [Trace collection overview](https://docs.cloud.google.com/spanner/docs/tracing-overview) .

## April 09, 2025

Feature

You can use Gemini assistance to help you use system insights to optimize and troubleshoot Spanner resources. For more information, see [Optimize and troubleshoot with Gemini assistance](https://docs.cloud.google.com/spanner/docs/observe-troubleshoot-with-gemini) .

Feature

Spanner offers Cassandra compatibility with API support and migration tools allowing seamless lift-and-shift migrations of Cassandra applications. For more information, see [Migrate from Cassandra to Spanner](https://docs.cloud.google.com/spanner/docs/non-relational/migrate-from-cassandra-to-spanner) .

## April 04, 2025

Feature

Spanner includes the [`PARAMETER_DEFAULT` column](https://docs.cloud.google.com/spanner/docs/information-schema#parameters) in the `INFORMATION_SCHEMA.PARAMETERS` table. This column returns the default value of change stream read functions parameters.

## April 03, 2025

Feature

In Spanner Graph you can view a visualization of graph elements returned by a Spanner Graph query and of a Spanner Graph schema. A graph query visualization helps you understand the query results by revealing patterns, dependencies, and anomalies in the returned graph elements. A graph schema visualization helps you understand how the nodes and edges in a schema are related. For more information, see [Work with Spanner Graph visualizations](https://docs.cloud.google.com/spanner/docs/graph/work-with-visualizations) .

## March 31, 2025

Feature

Spanner supports the following GoogleSQL JSON mutator functions:

- `JSON_ARRAY_APPEND()`
- `JSON_ARRAY_INSERT()`
- `JSON_REMOVE()`
- `JSON_SET()`
- `JSON_STRIP_NULLS()`

Spanner supports the following PostgreSQL `JSONB` mutator functions:

- `jsonb_insert()`
- `jsonb_set()`
- `jsonb_set_lax()`
- `jsonb_strip_nulls()`

Spanner also supports the following PostgreSQL `JSONB` operators:

- concat: `jsonb || jsonb -> jsonb`
- delete: `jsonb - text -> jsonb`

For more information, see [JSON functions in GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/functions-all) and [Supported PostgreSQL functions](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators) .

Feature

The GoogleSQL [`JSON_KEYS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#json_keys) and PostgreSQL [`json_object_keys`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#jsonb_functions) functions, which extract unique JSON keys from a JSON expression, are generally available.

Feature

JSON search indexes are generally available in Spanner. This extension of Spanner's full-text index capabilities accelerates many JSON document queries, even without prior knowledge of the documents' structure. You can create search indexes over any JSON document stored in a `JSON` column. The [`JSON_CONTAINS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#json_contains) function in GoogleSQL and the [`@>` and `<@`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#jsonb_operators) operators in PostgreSQL can use search indexes to determine if one document structure is contained in another. Search indexing supports `JSON` types in GoogleSQL-dialect databases and `JSONB` in PostgreSQL-dialect databases. For more information, see [JSON search indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/json-indexes) .

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.77.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.76.1...spanner/v1.77.0) (2025-03-03)

##### Features

- **spanner:** A new enum `IsolationLevel` is added ( [\#11624](https://github.com/googleapis/google-cloud-go/issues/11624) ) ( [2c4fb44](https://github.com/googleapis/google-cloud-go/commit/2c4fb448a2207a6d9988ec3a7646ea6cbb6f65f9) )
- **spanner:** A new field `isolation_level` is added to message `.google.spanner.v1.TransactionOptions` ( [2c4fb44](https://github.com/googleapis/google-cloud-go/commit/2c4fb448a2207a6d9988ec3a7646ea6cbb6f65f9) )
- **spanner:** Add a last field in the PartialResultSet ( [\#11645](https://github.com/googleapis/google-cloud-go/issues/11645) ) ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )
- **spanner:** Add option for LastStatement in transaction ( [\#11638](https://github.com/googleapis/google-cloud-go/issues/11638) ) ( [d662a45](https://github.com/googleapis/google-cloud-go/commit/d662a4537c3883d13a612e335477ca875b5cf479) )

##### Bug Fixes

- **spanner:** Avoid desructive context augmentation that dropped all headers ( [\#11659](https://github.com/googleapis/google-cloud-go/issues/11659) ) ( [594732d](https://github.com/googleapis/google-cloud-go/commit/594732dac26341ec00fe20cd40a6cfab9bde6317) )

##### Documentation

- **spanner:** A comment for enum value `OPTIMISTIC` in enum `ReadLockMode` is changed ( [2c4fb44](https://github.com/googleapis/google-cloud-go/commit/2c4fb448a2207a6d9988ec3a7646ea6cbb6f65f9) )
- **spanner:** A comment for enum value `PESSIMISTIC` in enum `ReadLockMode` is changed ( [2c4fb44](https://github.com/googleapis/google-cloud-go/commit/2c4fb448a2207a6d9988ec3a7646ea6cbb6f65f9) )
- **spanner:** A comment for enum value `READ_LOCK_MODE_UNSPECIFIED` in enum `ReadLockMode` is changed ( [2c4fb44](https://github.com/googleapis/google-cloud-go/commit/2c4fb448a2207a6d9988ec3a7646ea6cbb6f65f9) )
- **spanner:** A comment for field `chunked_value` in message `.google.spanner.v1.PartialResultSet` is changed ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )
- **spanner:** A comment for field `precommit_token` in message `.google.spanner.v1.PartialResultSet` is changed ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )
- **spanner:** A comment for field `precommit_token` in message `.google.spanner.v1.ResultSet` is changed ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )
- **spanner:** A comment for field `query_plan` in message `.google.spanner.v1.ResultSetStats` is changed ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )
- **spanner:** A comment for field `row_count_lower_bound` in message `.google.spanner.v1.ResultSetStats` is changed ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )
- **spanner:** A comment for field `row_type` in message `.google.spanner.v1.ResultSetMetadata` is changed ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )
- **spanner:** A comment for field `rows` in message `.google.spanner.v1.ResultSet` is changed ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )
- **spanner:** A comment for field `stats` in message `.google.spanner.v1.PartialResultSet` is changed ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )
- **spanner:** A comment for field `stats` in message `.google.spanner.v1.ResultSet` is changed ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )
- **spanner:** A comment for field `values` in message `.google.spanner.v1.PartialResultSet` is changed ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )
- **spanner:** A comment for message `ResultSetMetadata` is changed ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )
- **spanner:** A comment for message `ResultSetStats` is changed ( [794ecf7](https://github.com/googleapis/google-cloud-go/commit/794ecf77993a83fcad01912fb066366ba16adc11) )

#### [1.78.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.77.0...spanner/v1.78.0) (2025-03-24)

##### Features

- **spanner/spansql:** Add support for tokenlist and create search index ( [\#11522](https://github.com/googleapis/google-cloud-go/issues/11522) ) ( [cd894f8](https://github.com/googleapis/google-cloud-go/commit/cd894f8aea935c43a6c5625c2fd66b132a9b2f8b) )
- **spanner:** Support multiplexed sessions for ReadWriteStmtBasedTransaction ( [\#11852](https://github.com/googleapis/google-cloud-go/issues/11852) ) ( [528d9dd](https://github.com/googleapis/google-cloud-go/commit/528d9ddc548c5f05237cf5a0cc4b762ca5f1dd31) )

##### Bug Fixes

- **spanner/test/opentelemetry/test:** Update golang.org/x/net to 0.37.0 ( [1144978](https://github.com/googleapis/google-cloud-go/commit/11449782c7fb4896bf8b8b9cde8e7441c84fb2fd) )
- **spanner:** Revert the ALTS bound token enablement ( [\#11799](https://github.com/googleapis/google-cloud-go/issues/11799) ) ( [68cfb38](https://github.com/googleapis/google-cloud-go/commit/68cfb385ef30c636ab491e6d010cb69e8e1bebf4) )
- **spanner:** Update golang.org/x/net to 0.37.0 ( [1144978](https://github.com/googleapis/google-cloud-go/commit/11449782c7fb4896bf8b8b9cde8e7441c84fb2fd) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.88.0](https://github.com/googleapis/java-spanner/compare/v6.87.0...v6.88.0) (2025-02-27)

##### Features

- Add a last field in the PartialResultSet ( [7c714be](https://github.com/googleapis/java-spanner/commit/7c714be10eb345f2d8f566d752f6de615061c4da) )
- Automatically set default sequence kind in JDBC and PGAdapter ( [\#3658](https://github.com/googleapis/java-spanner/issues/3658) ) ( [e8abf33](https://github.com/googleapis/java-spanner/commit/e8abf338b85e95f185ab2875a804134523f84de3) )
- Default authentication support for external hosts ( [\#3656](https://github.com/googleapis/java-spanner/issues/3656) ) ( [ace11d5](https://github.com/googleapis/java-spanner/commit/ace11d5d928fb567b16560263ae95aa9cd916e22) )
- **spanner:** A new enum `IsolationLevel` is added ( [3fd33ba](https://github.com/googleapis/java-spanner/commit/3fd33ba9c5fab43ed475ed3cff9d60c008843981) )
- **spanner:** Add instance partitions field in backup proto ( [3fd33ba](https://github.com/googleapis/java-spanner/commit/3fd33ba9c5fab43ed475ed3cff9d60c008843981) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.54.0 ( [57497ad](https://github.com/googleapis/java-spanner/commit/57497ad00c62f152f493645f382530cf0eedf19e) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.44.0 ( [\#3665](https://github.com/googleapis/java-spanner/issues/3665) ) ( [3543548](https://github.com/googleapis/java-spanner/commit/35435488f87ebd59179698e8f74578b41eb219da) )

#### [6.89.0](https://github.com/googleapis/java-spanner/compare/v6.88.0...v6.89.0) (2025-03-20)

##### Features

- Enable ALTS hard bound token in DirectPath ( [\#3645](https://github.com/googleapis/java-spanner/issues/3645) ) ( [42cc961](https://github.com/googleapis/java-spanner/commit/42cc9616fa74c765d5716fd948dc0823df0a07a6) )
- Next release from main branch is 6.89.0 ( [\#3669](https://github.com/googleapis/java-spanner/issues/3669) ) ( [7a8a29b](https://github.com/googleapis/java-spanner/commit/7a8a29be40258294cafd13b1df7df5ea349a675d) )
- Support isolation level REPEATABLE_READ for R/W transactions ( [\#3670](https://github.com/googleapis/java-spanner/issues/3670) ) ( [e62f5ab](https://github.com/googleapis/java-spanner/commit/e62f5ab46da8696a8ff0d213f924588612bb4025) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.55.1 ( [b959f4c](https://github.com/googleapis/java-spanner/commit/b959f4c8ebb3551796a894b659aa42ba16fb1c39) )
- Revert the ALTS bound token enablement ( [\#3679](https://github.com/googleapis/java-spanner/issues/3679) ) ( [183c1f0](https://github.com/googleapis/java-spanner/commit/183c1f0e228a927a575596a38a01d63bb8eb6943) )

##### Performance Improvements

- Get database dialect using multiplexed session ( [\#3684](https://github.com/googleapis/java-spanner/issues/3684) ) ( [f641a40](https://github.com/googleapis/java-spanner/commit/f641a40ed515a6559718c2fe2757c322f037d83b) )
- Skip gRPC trailers for StreamingRead & ExecuteStreamingSql ( [\#3661](https://github.com/googleapis/java-spanner/issues/3661) ) ( [bd4b1f5](https://github.com/googleapis/java-spanner/commit/bd4b1f5b9612f6a4dfd748d735c887f8e46ae106) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.45.1 ( [\#3689](https://github.com/googleapis/java-spanner/issues/3689) ) ( [67188df](https://github.com/googleapis/java-spanner/commit/67188df2be23eef88de8f4febc3ac7208ebdd937) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.19.0](https://github.com/googleapis/nodejs-spanner/compare/v7.18.1...v7.19.0) (2025-02-26)

##### Features

- Add AddSplitPoints API ( [e4d389a](https://github.com/googleapis/nodejs-spanner/commit/e4d389a23ff4b73b2d0774ad31a84c9a6c19e306) )
- Paging changes for bigquery ( [e4d389a](https://github.com/googleapis/nodejs-spanner/commit/e4d389a23ff4b73b2d0774ad31a84c9a6c19e306) )
- **spanner:** A new enum `IsolationLevel` is added ( [\#2225](https://github.com/googleapis/nodejs-spanner/issues/2225) ) ( [e4d389a](https://github.com/googleapis/nodejs-spanner/commit/e4d389a23ff4b73b2d0774ad31a84c9a6c19e306) )
- **spanner:** A new field `isolation_level` is added to message `.google.spanner.v1.TransactionOptions` ( [e4d389a](https://github.com/googleapis/nodejs-spanner/commit/e4d389a23ff4b73b2d0774ad31a84c9a6c19e306) )
- **spanner:** Add instance partitions field in backup proto ( [e4d389a](https://github.com/googleapis/nodejs-spanner/commit/e4d389a23ff4b73b2d0774ad31a84c9a6c19e306) )
- **spanner:** Add support for Multiplexed Session for Read Only Tran… ( [\#2214](https://github.com/googleapis/nodejs-spanner/issues/2214) ) ( [3a7a51b](https://github.com/googleapis/nodejs-spanner/commit/3a7a51bee00730c2daf1b9791b45f75531c14a2c) )
- **x-goog-spanner-request-id:** Add bases ( [\#2211](https://github.com/googleapis/nodejs-spanner/issues/2211) ) ( [0008038](https://github.com/googleapis/nodejs-spanner/commit/000803812e670ce0f4bac4a6460351f2b08ec660) )

##### Bug Fixes

- Add x-goog-request params to headers for LRO-polling methods ( [e4d389a](https://github.com/googleapis/nodejs-spanner/commit/e4d389a23ff4b73b2d0774ad31a84c9a6c19e306) )
- Error from fill method should not be emitted ( [\#2233](https://github.com/googleapis/nodejs-spanner/issues/2233) ) ( [2cc44cf](https://github.com/googleapis/nodejs-spanner/commit/2cc44cf238bd18f5a456c76ddb8280c2252c2e87) ), closes [\#2103](https://github.com/googleapis/nodejs-spanner/issues/2103)
- Finalize fixing typings for headers in generator ( [e4d389a](https://github.com/googleapis/nodejs-spanner/commit/e4d389a23ff4b73b2d0774ad31a84c9a6c19e306) )
- Fix typings for headers in generator ( [e4d389a](https://github.com/googleapis/nodejs-spanner/commit/e4d389a23ff4b73b2d0774ad31a84c9a6c19e306) )
- Remove extra protos in ESM & capture ESM in headers ( [e4d389a](https://github.com/googleapis/nodejs-spanner/commit/e4d389a23ff4b73b2d0774ad31a84c9a6c19e306) )
- Rollback with no id ( [\#2231](https://github.com/googleapis/nodejs-spanner/issues/2231) ) ( [a6919b1](https://github.com/googleapis/nodejs-spanner/commit/a6919b15bd01ed93c62d32533d78181cbd333f5e) ), closes [\#2103](https://github.com/googleapis/nodejs-spanner/issues/2103)

#### [7.19.1](https://github.com/googleapis/nodejs-spanner/compare/v7.19.0...v7.19.1) (2025-03-13)

##### Bug Fixes

- CreateQueryPartition with query params ( [91f5afd](https://github.com/googleapis/nodejs-spanner/commit/91f5afda53bd9c46fcd1a1fe33f579b6aed5223a) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.53.0](https://github.com/googleapis/python-spanner/compare/v3.52.0...v3.53.0) (2025-03-12)

##### Features

- Add AddSplitPoints API ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- Add Attempt, Operation and GFE Metrics ( [\#1302](https://github.com/googleapis/python-spanner/issues/1302) ) ( [fb21d9a](https://github.com/googleapis/python-spanner/commit/fb21d9acf2545cf7b8e9e21b65eabf21a7bf895f) )
- Add REST Interceptors which support reading metadata ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- Add support for opt-in debug logging ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- Add support for reading selective GAPIC generation methods from service YAML ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- Add the last statement option to ExecuteSqlRequest and ExecuteBatchDmlRequest ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- Add UUID in Spanner TypeCode enum ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- End to end tracing ( [\#1315](https://github.com/googleapis/python-spanner/issues/1315) ) ( [aa5d0e6](https://github.com/googleapis/python-spanner/commit/aa5d0e6c1d3e5b0e4b0578e80c21e7c523c30fb5) )
- Exposing FreeInstanceAvailability in InstanceConfig ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- Exposing FreeInstanceMetadata in Instance configuration (to define the metadata related to FREE instance type) ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- Exposing InstanceType in Instance configuration (to define PROVISIONED or FREE spanner instance) ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- Exposing QuorumType in InstanceConfig ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- Exposing storage_limit_per_processing_unit in InstanceConfig ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- Snapshot isolation ( [\#1318](https://github.com/googleapis/python-spanner/issues/1318) ) ( [992fcae](https://github.com/googleapis/python-spanner/commit/992fcae2d4fd2b47380d159a3416b8d6d6e1c937) )
- **spanner:** A new enum `IsolationLevel` is added ( [\#1224](https://github.com/googleapis/python-spanner/issues/1224) ) ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )

##### Bug Fixes

- Allow Protobuf 6.x ( [\#1320](https://github.com/googleapis/python-spanner/issues/1320) ) ( [1faab91](https://github.com/googleapis/python-spanner/commit/1faab91790ae3e2179fbab11b69bb02254ab048a) )
- Cleanup after metric integration test ( [\#1322](https://github.com/googleapis/python-spanner/issues/1322) ) ( [d7cf8b9](https://github.com/googleapis/python-spanner/commit/d7cf8b968dfc2b98d3b1d7ae8a025da55bec0767) )
- **deps:** Require grpc-google-iam-v1\>=0.14.0 ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- Fix typing issue with gRPC metadata when key ends in -bin ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )

##### Performance Improvements

- Add option for last_statement ( [\#1313](https://github.com/googleapis/python-spanner/issues/1313) ) ( [19ab6ef](https://github.com/googleapis/python-spanner/commit/19ab6ef0d58262ebb19183e700db6cf124f9b3c5) )

##### Documentation

- A comment for enum `DefaultBackupScheduleType` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for enum value `AUTOMATIC` in enum `DefaultBackupScheduleType` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for enum value `GOOGLE_MANAGED` in enum `Type` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for enum value `NONE` in enum `DefaultBackupScheduleType` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for enum value `USER_MANAGED` in enum `Type` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `base_config` in message `.google.spanner.admin.instance.v1.InstanceConfig` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `default_backup_schedule_type` in message `.google.spanner.admin.instance.v1.Instance` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `filter` in message `.google.spanner.admin.instance.v1.ListInstanceConfigOperationsRequest` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `filter` in message `.google.spanner.admin.instance.v1.ListInstancePartitionOperationsRequest` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `instance_config` in message `.google.spanner.admin.instance.v1.CreateInstanceConfigRequest` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `instance_partition_deadline` in message `.google.spanner.admin.instance.v1.ListInstancePartitionOperationsRequest` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `location` in message `.google.spanner.admin.instance.v1.ReplicaInfo` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `node_count` in message `.google.spanner.admin.instance.v1.Instance` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `node_count` in message `.google.spanner.admin.instance.v1.InstancePartition` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `operations` in message `.google.spanner.admin.instance.v1.ListInstanceConfigOperationsResponse` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `operations` in message `.google.spanner.admin.instance.v1.ListInstancePartitionOperationsResponse` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `optional_replicas` in message `.google.spanner.admin.instance.v1.InstanceConfig` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `parent` in message `.google.spanner.admin.instance.v1.ListInstancePartitionsRequest` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `processing_units` in message `.google.spanner.admin.instance.v1.Instance` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `processing_units` in message `.google.spanner.admin.instance.v1.InstancePartition` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `referencing_backups` in message `.google.spanner.admin.instance.v1.InstancePartition` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `replicas` in message `.google.spanner.admin.instance.v1.InstanceConfig` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `storage_utilization_percent` in message `.google.spanner.admin.instance.v1.AutoscalingConfig` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for field `unreachable` in message `.google.spanner.admin.instance.v1.ListInstancePartitionsResponse` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for message `CreateInstanceConfigRequest` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for message `DeleteInstanceConfigRequest` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for message `UpdateInstanceConfigRequest` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for method `CreateInstance` in service `InstanceAdmin` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for method `CreateInstanceConfig` in service `InstanceAdmin` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for method `CreateInstancePartition` in service `InstanceAdmin` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for method `ListInstanceConfigOperations` in service `InstanceAdmin` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for method `ListInstanceConfigs` in service `InstanceAdmin` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for method `ListInstancePartitionOperations` in service `InstanceAdmin` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for method `MoveInstance` in service `InstanceAdmin` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for method `UpdateInstance` in service `InstanceAdmin` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for method `UpdateInstanceConfig` in service `InstanceAdmin` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- A comment for method `UpdateInstancePartition` in service `InstanceAdmin` is changed ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )
- Fix typo timzeone -\> timezone ( [7a5afba](https://github.com/googleapis/python-spanner/commit/7a5afba28b20ac94f3eec799f4b572c95af60b94) )

## March 28, 2025

Feature

Spanner vector index and approximate nearest neighbor (ANN) distance functions in the GoogleSQL-dialect are Generally Available. If you have a table with a large amount of vector data, you can use a vector index to accelerate similarity searches and nearest neighbor queries. Spanner also supports the following:

- [`ALTER VECTOR INDEX`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter-vector-index) DDL syntax
- [Import and export](https://docs.cloud.google.com/spanner/docs/import-export-overview) databases that use ANN
- Use the [`STORING` clause](https://docs.cloud.google.com/spanner/docs/secondary-indexes#storing_clause) to store a copy of a column in the vector index to accelerate queries that filter by those columns
- Use ANN in instances smaller than one node or 1000 processing units

For more information, see [Find approximate nearest neighbors, create vector indexes, and query vector embeddings](https://docs.cloud.google.com/spanner/docs/find-approximate-nearest-neighbors) .

Feature

Spanner ANN indexes are supported in Langchain. For more information, see [LangChain Quickstart for Spanner](https://github.com/googleapis/langchain-google-spanner-python/blob/main/samples/langchain_quick_start.ipynb) .

## March 27, 2025

Feature

You can save and manage your SQL scripts in Spanner Studio. This feature is in [preview](https://cloud.google.com/products#product-launch-stages) . For more information, see [Saved queries overview](https://docs.cloud.google.com/spanner/docs/saved-queries) .

## March 18, 2025

Feature

The default time zone of your Spanner databases can be set. For more information, see [Set the default time zone of a database](https://docs.cloud.google.com/spanner/docs/set-default-time-zone) . This feature is [generally available](https://cloud.google.com/products#product-launch-stages) (GA).

## March 17, 2025

Feature

You can create an [external dataset](https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets) in BigQuery that links to an existing database in Spanner. This feature is [generally available](https://cloud.google.com/products#product-launch-stages) (GA).

Feature

You can use [`EXPORT DATA` statements](https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/other-statements) to [reverse ETL BigQuery data to Spanner](https://docs.cloud.google.com/bigquery/docs/export-to-spanner) . This feature is [generally available](https://cloud.google.com/products#product-launch-stages) (GA).

## March 10, 2025

Feature

Tiered storage is Generally Available in Spanner. Tiered storage is a fully-managed feature that lets you store your data across solid-state drives (SSD) or hard disk drives (HDD). Using tiered storage lets you take advantage of both SSD storage, which supports the high performance of active data, and HDD storage, which supports infrequent data access at a lower cost. For more information, see [Tiered storage](https://docs.cloud.google.com/spanner/docs/tiered-storage) .

Feature

The following LangChain solutions can be used with Spanner Graph:

- Graph store for Spanner: Use to retrieve and store nodes and edges from a graph database. For more information, see [Graph store for Spanner](https://docs.cloud.google.com/spanner/docs/langchain#graph-store) .
- Graph QA chain for Spanner: Uses a graph to answer questions. For more information, see [Graph QA chain for Spanner](https://docs.cloud.google.com/spanner/docs/langchain#graph-qa) .

## March 04, 2025

Feature

You can create Spanner [regional instance configurations](https://cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) in Stockholm, Sweden ( `europe-north2` ). For more information, see [Google Cloud locations](https://cloud.google.com/about/locations) and [Spanner pricing](https://docs.cloud.google.com/spanner/pricing) .

Feature

The following [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-multi-region) is available in Europe - `eur7` (Milan/Frankfurt/Turin).

## February 28, 2025

Feature

[Full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) is generally available for PostgreSQL-dialect databases.

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.74.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.73.0...spanner/v1.74.0) (2025-01-24)

##### Features

- **spanner/admin/instance:** Exposing FreeInstanceAvailability in InstanceConfig ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** Exposing FreeInstanceMetadata in Instance configuration (to define the metadata related to FREE instance type) ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** Exposing InstanceType in Instance configuration (to define PROVISIONED or FREE spanner instance) ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** Exposing QuorumType in InstanceConfig ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** Exposing storage_limit_per_processing_unit in InstanceConfig ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner:** Add the last statement option to ExecuteSqlRequest and ExecuteBatchDmlRequest ( [8dedb87](https://github.com/googleapis/google-cloud-go/commit/8dedb878c070cc1e92d62bb9b32358425e3ceffb) )
- **spanner:** Add UUID in Spanner TypeCode enum ( [46fc993](https://github.com/googleapis/google-cloud-go/commit/46fc993a3195203a230e2831bee456baaa9f7b1c) )
- **spanner:** Implement generation and propagation of "x-goog-spanner-request-id" Header ( [\#11048](https://github.com/googleapis/google-cloud-go/issues/11048) ) ( [10960c1](https://github.com/googleapis/google-cloud-go/commit/10960c162660d183c70837762c8f71aeb6c6e1eb) )

##### Bug Fixes

- **spanner/spansql:** PROTO BUNDLE and protobuf type parsing fixes ( [\#11279](https://github.com/googleapis/google-cloud-go/issues/11279) ) ( [b1ca714](https://github.com/googleapis/google-cloud-go/commit/b1ca71449eeeda9ff19a57a3a759e17005692d06) )
- **spanner/test/opentelemetry/test:** Update golang.org/x/net to v0.33.0 ( [e9b0b69](https://github.com/googleapis/google-cloud-go/commit/e9b0b69644ea5b276cacff0a707e8a5e87efafc9) )
- **spanner:** ReadWriteStmtBasedTransaction would not remember options for retries ( [\#11443](https://github.com/googleapis/google-cloud-go/issues/11443) ) ( [7d8f0c5](https://github.com/googleapis/google-cloud-go/commit/7d8f0c556b43265a8cf7715509892ac136d3da4b) )
- **spanner:** Support setting monitoring host via env and override any endpoint override from spanner options with default one ( [\#11141](https://github.com/googleapis/google-cloud-go/issues/11141) ) ( [3d61545](https://github.com/googleapis/google-cloud-go/commit/3d6154588da598092e5cffe6c1c5770998037fe1) )
- **spanner:** Update golang.org/x/net to v0.33.0 ( [e9b0b69](https://github.com/googleapis/google-cloud-go/commit/e9b0b69644ea5b276cacff0a707e8a5e87efafc9) )

##### Documentation

- **spanner/admin/database:** Fix typo timzeone -\> timezone ( [a694e11](https://github.com/googleapis/google-cloud-go/commit/a694e1152fc75307da6ca8dcfff26cae9189f29c) )
- **spanner/admin/instance:** A comment for enum `DefaultBackupScheduleType` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for enum value `AUTOMATIC` in enum `DefaultBackupScheduleType` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for enum value `GOOGLE_MANAGED` in enum `Type` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for enum value `NONE` in enum `DefaultBackupScheduleType` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for enum value `USER_MANAGED` in enum `Type` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `base_config` in message `.google.spanner.admin.instance.v1.InstanceConfig` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `default_backup_schedule_type` in message `.google.spanner.admin.instance.v1.Instance` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `filter` in message `.google.spanner.admin.instance.v1.ListInstanceConfigOperationsRequest` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `filter` in message `.google.spanner.admin.instance.v1.ListInstancePartitionOperationsRequest` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `instance_config` in message `.google.spanner.admin.instance.v1.CreateInstanceConfigRequest` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `instance_partition_deadline` in message `.google.spanner.admin.instance.v1.ListInstancePartitionOperationsRequest` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `location` in message `.google.spanner.admin.instance.v1.ReplicaInfo` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `node_count` in message `.google.spanner.admin.instance.v1.Instance` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `node_count` in message `.google.spanner.admin.instance.v1.InstancePartition` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `operations` in message `.google.spanner.admin.instance.v1.ListInstanceConfigOperationsResponse` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `operations` in message `.google.spanner.admin.instance.v1.ListInstancePartitionOperationsResponse` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `optional_replicas` in message `.google.spanner.admin.instance.v1.InstanceConfig` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `parent` in message `.google.spanner.admin.instance.v1.ListInstancePartitionsRequest` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `processing_units` in message `.google.spanner.admin.instance.v1.Instance` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `processing_units` in message `.google.spanner.admin.instance.v1.InstancePartition` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `referencing_backups` in message `.google.spanner.admin.instance.v1.InstancePartition` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `replicas` in message `.google.spanner.admin.instance.v1.InstanceConfig` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `storage_utilization_percent` in message `.google.spanner.admin.instance.v1.AutoscalingConfig` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for field `unreachable` in message `.google.spanner.admin.instance.v1.ListInstancePartitionsResponse` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for message `CreateInstanceConfigRequest` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for message `DeleteInstanceConfigRequest` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for message `UpdateInstanceConfigRequest` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for method `CreateInstance` in service `InstanceAdmin` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for method `CreateInstanceConfig` in service `InstanceAdmin` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for method `CreateInstancePartition` in service `InstanceAdmin` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for method `ListInstanceConfigOperations` in service `InstanceAdmin` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for method `ListInstanceConfigs` in service `InstanceAdmin` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for method `ListInstancePartitionOperations` in service `InstanceAdmin` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for method `MoveInstance` in service `InstanceAdmin` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for method `UpdateInstance` in service `InstanceAdmin` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for method `UpdateInstanceConfig` in service `InstanceAdmin` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )
- **spanner/admin/instance:** A comment for method `UpdateInstancePartition` in service `InstanceAdmin` is changed ( [4254053](https://github.com/googleapis/google-cloud-go/commit/42540530e44e5f331e66e0777c4aabf449f5fd90) )

#### [1.75.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.74.0...spanner/v1.75.0) (2025-02-02)

##### Features

- **spanner/admin/database:** Add AddSplitPoints API ( [59fe58a](https://github.com/googleapis/google-cloud-go/commit/59fe58aba61abf69bfb7549c0a03b21bdb4b8b2f) )

##### Bug Fixes

- **spanner:** Inject "x-goog-spanner-request-id" into outgoing client context ( [\#11544](https://github.com/googleapis/google-cloud-go/issues/11544) ) ( [a8f16ef](https://github.com/googleapis/google-cloud-go/commit/a8f16ef102068ad793b4aa8c6cf8c8f0ca0c2d03) ), refs [\#11543](https://github.com/googleapis/google-cloud-go/issues/11543)

#### [1.76.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.75.0...spanner/v1.76.0) (2025-02-20)

**DO NOT USE** This version is retracted due to https://github.com/googleapis/google-cloud-go/issues/11630, use version \>=v1.76.1

##### Features

- **spanner/admin/database:** Add instance partitions field in backup proto ( [c6a6dc7](https://github.com/googleapis/google-cloud-go/commit/c6a6dc7c6e63740ec25fc1eb34990f4550a6a1f3) )
- **spanner:** Support multiplexed session for read-write transactions & partition ops ( [\#11615](https://github.com/googleapis/google-cloud-go/issues/11615) ) ( [4b40201](https://github.com/googleapis/google-cloud-go/commit/4b40201c5cfa223f3e5c039e18a72d74168f7ae9) )

##### Performance Improvements

- **spanner:** Grab debug stack outside of lock ( [\#11587](https://github.com/googleapis/google-cloud-go/issues/11587) ) ( [0ee82ff](https://github.com/googleapis/google-cloud-go/commit/0ee82ff4ff385bc632b5cc9630e7e6c4e25a438c) )

#### [1.76.1](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.76.0...spanner/v1.76.1) (2025-02-21)

##### Bug Fixes

- **spanner:** Multiplexed_session_previous_transaction_id is not supported in the request for a non multiplexed session ( [\#11626](https://github.com/googleapis/google-cloud-go/issues/11626) ) ( [a940bef](https://github.com/googleapis/google-cloud-go/commit/a940befed09ba7ff0de12c720036bc93fee7e8c7) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.86.0](https://github.com/googleapis/java-spanner/compare/v6.85.0...v6.86.0) (2025-01-31)

##### Features

- Add sample for asymmetric autoscaling instances ( [\#3562](https://github.com/googleapis/java-spanner/issues/3562) ) ( [3584b81](https://github.com/googleapis/java-spanner/commit/3584b81a27bfcdd071fbf7e0d40dfa840ea88151) )
- Support graph and pipe queries in Connection API ( [\#3586](https://github.com/googleapis/java-spanner/issues/3586) ) ( [71c3063](https://github.com/googleapis/java-spanner/commit/71c306346d5b3805f55d5698cf8867d5f4ae519e) )

##### Bug Fixes

- Always add instance-id for built-in metrics ( [\#3612](https://github.com/googleapis/java-spanner/issues/3612) ) ( [705b627](https://github.com/googleapis/java-spanner/commit/705b627646f1679b7d1c4c1f86a853872cf8bfd5) )
- **deps:** Update the Java code generator (gapic-generator-java) to 2.51.1 ( [3e27251](https://github.com/googleapis/java-spanner/commit/3e272510970d1951b74c4ec9425f1a890790ddb3) )
- **deps:** Update the Java code generator (gapic-generator-java) to 2.52.0 ( [bf69673](https://github.com/googleapis/java-spanner/commit/bf69673886dbe040292214ed6e64997a230441f6) )
- **spanner:** Moved mTLSContext configurator from builder to construtor ( [\#3605](https://github.com/googleapis/java-spanner/issues/3605) ) ( [ac7c30b](https://github.com/googleapis/java-spanner/commit/ac7c30bfb14bdafc11675c2a120effde4a71c922) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.42.0 ( [\#3616](https://github.com/googleapis/java-spanner/issues/3616) ) ( [2ea59f0](https://github.com/googleapis/java-spanner/commit/2ea59f05225f2dba2effb503e6abddcfdb6fe6ee) )
- Update dependency io.opentelemetry:opentelemetry-bom to v1.46.0 ( [\#3530](https://github.com/googleapis/java-spanner/issues/3530) ) ( [d505850](https://github.com/googleapis/java-spanner/commit/d5058504b94501cabd75ad5e7030404b63c3f8b4) )

##### Documentation

- Clarify how async updates can overtake each other ( [\#3581](https://github.com/googleapis/java-spanner/issues/3581) ) ( [1be250f](https://github.com/googleapis/java-spanner/commit/1be250fea686f3a41739c9c8aa474ed956b130e4) )
- Fix typo timzeone -\> timezone ( [bf69673](https://github.com/googleapis/java-spanner/commit/bf69673886dbe040292214ed6e64997a230441f6) )
- Fixed parameter arguments for AbstractResultSet's Listener's on TransactionMetadata doc ( [\#3602](https://github.com/googleapis/java-spanner/issues/3602) ) ( [1f143a4](https://github.com/googleapis/java-spanner/commit/1f143a4b7b899aec8cf58546f7540a41d1c73731) )
- **samples:** Add samples and tests for change streams transaction exclusion ( [\#3098](https://github.com/googleapis/java-spanner/issues/3098) ) ( [1f81600](https://github.com/googleapis/java-spanner/commit/1f816009abdbfb32bb26686d8fdb2a771216004e) )

#### [6.87.0](https://github.com/googleapis/java-spanner/compare/v6.86.0...v6.87.0) (2025-02-20)

##### Features

- Add AddSplitPoints API ( [a5ebcd3](https://github.com/googleapis/java-spanner/commit/a5ebcd343a67c57d61362cfb0ccb4888f5503681) )
- Add option for multiplexed sessions with partitioned operations ( [\#3635](https://github.com/googleapis/java-spanner/issues/3635) ) ( [dc89b4d](https://github.com/googleapis/java-spanner/commit/dc89b4d7663f0e40a9169b21243f2d94f2fc5749) )
- Add option to indicate that a statement is the last in a transaction ( [\#3647](https://github.com/googleapis/java-spanner/issues/3647) ) ( [b04ea80](https://github.com/googleapis/java-spanner/commit/b04ea804cfa9551b4d7c49cd83f0ef1120942423) )
- Adding gfe_latencies metric to built-in metrics ( [\#3490](https://github.com/googleapis/java-spanner/issues/3490) ) ( [314dadc](https://github.com/googleapis/java-spanner/commit/314dadc31f4a5aa798d45886db7231c1bd8b7a91) )
- **spanner:** Support multiplexed session for read-write transactions ( [\#3608](https://github.com/googleapis/java-spanner/issues/3608) ) ( [bda78ed](https://github.com/googleapis/java-spanner/commit/bda78edaba827acf974c87c335868a6f8caa38f2) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.53.0 ( [20a3d0d](https://github.com/googleapis/java-spanner/commit/20a3d0da41509ffca66c77de6771fc8080930613) )
- **spanner:** End spans for read-write methods ( [\#3629](https://github.com/googleapis/java-spanner/issues/3629) ) ( [4a1f99c](https://github.com/googleapis/java-spanner/commit/4a1f99c6bb872ffc08e60d3843e4cdfc4efa2690) )
- **spanner:** Release resources in TransactionManager ( [\#3638](https://github.com/googleapis/java-spanner/issues/3638) ) ( [e0a3e5b](https://github.com/googleapis/java-spanner/commit/e0a3e5bd169e28e349a2dc92f86a2a9b5510f8f6) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.43.0 ( [\#3642](https://github.com/googleapis/java-spanner/issues/3642) ) ( [c12968a](https://github.com/googleapis/java-spanner/commit/c12968a5f6dad95017d9867d96d4f19a26643a07) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.18.0](https://github.com/googleapis/nodejs-spanner/compare/v7.17.1...v7.18.0) (2025-01-29)

##### Features

- Add gcp client attributes for Opentelemetry traces ( [\#2215](https://github.com/googleapis/nodejs-spanner/issues/2215) ) ( [d2ff046](https://github.com/googleapis/nodejs-spanner/commit/d2ff046854b4139af6e3a6f0d2122619cdf83131) )

#### [7.18.1](https://github.com/googleapis/nodejs-spanner/compare/v7.18.0...v7.18.1) (2025-02-05)

##### Bug Fixes

- Fix NodeJS release ( [\#2229](https://github.com/googleapis/nodejs-spanner/issues/2229) ) ( [f830fc8](https://github.com/googleapis/nodejs-spanner/commit/f830fc82ce666902db3cddc667326dc2731c14a1) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.52.0](https://github.com/googleapis/python-spanner/compare/v3.51.0...v3.52.0) (2025-02-19)

##### Features

- Add additional opentelemetry span events for session pool ( [a6811af](https://github.com/googleapis/python-spanner/commit/a6811afefa6739caa20203048635d94f9b85c4c8) )
- Add Google Cloud standard otel attributes for python client ( [\#1308](https://github.com/googleapis/python-spanner/issues/1308) ) ( [0839f98](https://github.com/googleapis/python-spanner/commit/0839f982a3e7f5142825d10c440005a39cdb39cb) )
- Add updated span events + trace more methods ( [\#1259](https://github.com/googleapis/python-spanner/issues/1259) ) ( [ad69c48](https://github.com/googleapis/python-spanner/commit/ad69c48f01b09cbc5270b9cefde23715d5ac54b6) )
- MetricsTracer implementation ( [\#1291](https://github.com/googleapis/python-spanner/issues/1291) ) ( [8fbde6b](https://github.com/googleapis/python-spanner/commit/8fbde6b84d11db12ee4d536f0d5b8064619bdaa9) )
- Support GRAPH and pipe syntax in dbapi ( [\#1285](https://github.com/googleapis/python-spanner/issues/1285) ) ( [959bb9c](https://github.com/googleapis/python-spanner/commit/959bb9cda953eead89ffc271cb2a472e7139f81c) )
- Support transaction and request tags in dbapi ( [\#1262](https://github.com/googleapis/python-spanner/issues/1262) ) ( [ee9662f](https://github.com/googleapis/python-spanner/commit/ee9662f57dbb730afb08b9b9829e4e19bda5e69a) )
- **x-goog-spanner-request-id:** Introduce AtomicCounter ( [\#1275](https://github.com/googleapis/python-spanner/issues/1275) ) ( [f2483e1](https://github.com/googleapis/python-spanner/commit/f2483e11ba94f8bd1e142d1a85347d90104d1a19) )

##### Bug Fixes

- Retry UNAVAILABLE errors for streaming RPCs ( [\#1278](https://github.com/googleapis/python-spanner/issues/1278) ) ( [ab31078](https://github.com/googleapis/python-spanner/commit/ab310786baf09033a28c76e843b654e98a21613d) ), closes [\#1150](https://github.com/googleapis/python-spanner/issues/1150)
- **tracing:** Ensure nesting of Transaction.begin under commit + fix suggestions from feature review ( [\#1287](https://github.com/googleapis/python-spanner/issues/1287) ) ( [d9ee75a](https://github.com/googleapis/python-spanner/commit/d9ee75ac9ecfbf37a95c95a56295bdd79da3006d) )
- **tracing:** Only set span.status=OK if UNSET ( [\#1248](https://github.com/googleapis/python-spanner/issues/1248) ) ( [1d393fe](https://github.com/googleapis/python-spanner/commit/1d393fedf3be8b36c91d0f52a5f23cfa5c05f835) ), closes [\#1246](https://github.com/googleapis/python-spanner/issues/1246)
- Update retry strategy for mutation calls to handle aborted transactions ( [\#1279](https://github.com/googleapis/python-spanner/issues/1279) ) ( [0887eb4](https://github.com/googleapis/python-spanner/commit/0887eb43b6ea8bd9076ca81977d1446011335853) )

## February 20, 2025

Feature

The Java and Go clients for Spanner implement multiplexed sessions. This allows all requests to be concurrently sent over a single session, thus eliminating the requirement that you define the minimum and maximum session count. Instead, you can use any number of requests to the configured gRPC channels. This approach eliminates the possibility of session leaks and reduces the occurrences of `Transaction outcome unknown` errors. You must set an environment variable in your client to opt in to this feature. For more information, see [Multiplexed sessions](https://docs.cloud.google.com/spanner/docs/sessions#multiplexed_sessions) .

## February 11, 2025

Feature

[Managed autoscaler](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) is Generally Available.

Managed autoscaler also supports the ability to scale read-only replicas independently from read-write replicas. By setting the compute capacity limits and CPU utilization targets, you can configure the managed autoscaler for all replicas of an instance or independently scale read-only replicas. For more information, see [Asymmetric read-only autoscaling](https://docs.cloud.google.com/spanner/docs/managed-autoscaler#asymmetric-read-only-autoscaling) .

## February 10, 2025

Feature

Custom organization policies are generally available for Spanner. For more information, see [Add a custom organization policy](https://docs.cloud.google.com/spanner/docs/spanner-custom-constraints) .

## February 05, 2025

Feature

Informational foreign keys are available in Spanner. Informational foreign keys don't enforce referential integrity and are used to declare the intended logical data model for query optimization. Enforced foreign keys, which enforce referential integrity, are also available.

Informational foreign keys are supported by GoogleSQL only. Enforced foreign keys are supported by GoogleSQL and PostgreSQL.

For more information, see the following:

- [Types of foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#types-of-foreign-keys)
- [Comparison of foreign key types](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#comparison-of-foreign-keys-types)
- [Choose which foreign key type to use](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#choose-foreign-key-type)
- [Use informational foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#use-informational-foreign-keys)

## January 31, 2025

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.85.0](https://github.com/googleapis/java-spanner/compare/v6.84.0...v6.85.0) (2025-01-10)

##### Features

- Add gcp client attributes in OpenTelemetry traces ( [\#3595](https://github.com/googleapis/java-spanner/issues/3595) ) ( [7893f24](https://github.com/googleapis/java-spanner/commit/7893f2499f6a43e4e80ec78a9f0da5beedb6967a) )
- Add LockHint feature ( [\#3588](https://github.com/googleapis/java-spanner/issues/3588) ) ( [326442b](https://github.com/googleapis/java-spanner/commit/326442bca41700debcbeb67b6bd11fc36bd4f26d) )
- **spanner:** MTLS setup for spanner external host clients ( [\#3574](https://github.com/googleapis/java-spanner/issues/3574) ) ( [f8dd152](https://github.com/googleapis/java-spanner/commit/f8dd15272f2a250c5b57c9f2527d03dbd557d717) )

##### Dependencies

- Update dependency com.google.api.grpc:proto-google-cloud-monitoring-v3 to v3.56.0 ( [\#3563](https://github.com/googleapis/java-spanner/issues/3563) ) ( [e4d0b0f](https://github.com/googleapis/java-spanner/commit/e4d0b0ffa2308c8d949630b52c67e3b79c4491fb) )
- Update dependency com.google.api.grpc:proto-google-cloud-monitoring-v3 to v3.57.0 ( [\#3592](https://github.com/googleapis/java-spanner/issues/3592) ) ( [a7542da](https://github.com/googleapis/java-spanner/commit/a7542daff466226221eeb9a885a2e67a99adb678) )
- Update dependency com.google.cloud:sdk-platform-java-config to v3.41.1 ( [\#3589](https://github.com/googleapis/java-spanner/issues/3589) ) ( [2cd4238](https://github.com/googleapis/java-spanner/commit/2cd42388370dac004bfd807f6aede3ba45456706) )
- Update dependency com.google.cloud.opentelemetry:exporter-trace to v0.33.0 ( [\#3455](https://github.com/googleapis/java-spanner/issues/3455) ) ( [70649dc](https://github.com/googleapis/java-spanner/commit/70649dc2f64aa06404893cc6a36716fc366c83e7) )
- Update dependency com.google.re2j:re2j to v1.8 ( [\#3594](https://github.com/googleapis/java-spanner/issues/3594) ) ( [0f2013d](https://github.com/googleapis/java-spanner/commit/0f2013d66d3fd14e6be019cda6745ddc32032091) )
- Update googleapis/sdk-platform-java action to v2.51.1 ( [\#3591](https://github.com/googleapis/java-spanner/issues/3591) ) ( [3daa1a0](https://github.com/googleapis/java-spanner/commit/3daa1a0c735000845558a1d3612257a7d0524350) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.17.0](https://github.com/googleapis/nodejs-spanner/compare/v7.16.0...v7.17.0) (2024-12-27)

##### Known Issues

This release inadvertently introduced an issue where OpenTelemetry Trace context Global Propagators are default set to W3CTraceContextPropagator. For more details, refer to issue \#2208

A fix for this issue has been included in version [7.17.1](https://github.com/googleapis/nodejs-spanner/releases/tag/v7.17.1)

##### Features

- Add the last statement option to ExecuteSqlRequest and ExecuteBatchDmlRequest ( [\#2196](https://github.com/googleapis/nodejs-spanner/issues/2196) ) ( [223f167](https://github.com/googleapis/nodejs-spanner/commit/223f167c1c9bc4da26155637eabbcabce5487ede) )
- Enable e2e tracing ( [\#2202](https://github.com/googleapis/nodejs-spanner/issues/2202) ) ( [3cc257e](https://github.com/googleapis/nodejs-spanner/commit/3cc257e99925594776b9a1886f0173ce2dfe904f) )

##### Bug Fixes

- Span events Issue 2166 ( [\#2184](https://github.com/googleapis/nodejs-spanner/issues/2184) ) ( [97ed577](https://github.com/googleapis/nodejs-spanner/commit/97ed5776dbdf5e90f8398fffea08e2a968045f9b) )

#### [7.17.1](https://github.com/googleapis/nodejs-spanner/compare/v7.17.0...v7.17.1) (2025-01-03)

##### Bug Fixes

- Remove default global trace context propagator ( [\#2209](https://github.com/googleapis/nodejs-spanner/issues/2209) ) ( [7898e0c](https://github.com/googleapis/nodejs-spanner/commit/7898e0ce0477e2d4327822ac26a2674203b47a64) ), closes [\#2208](https://github.com/googleapis/nodejs-spanner/issues/2208)

## January 30, 2025

Feature

The Spanner index advisor is Generally Available in GoogleSQL and PostgreSQL-dialect databases. The index advisor analyzes your queries to recommend new indexes or changes to existing indexes to improve the performance of your queries. For more information, see [Use the Spanner index advisor](https://docs.cloud.google.com/spanner/docs/index-advisor) .

Feature

Spanner supports [`SERIAL`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-types#serial-types) and [`AUTO_INCREMENT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#table_statements) DDL syntax. `SERIAL` is available in PostgreSQL-dialect databases and `AUTO_INCREMENT` is available in GoogleSQL. They streamline the ability to generate `IDENTITY` columns as primary keys. For more information, see [`SERIAL` and `AUTO_INCREMENT`](https://docs.cloud.google.com/spanner/docs/primary-key-default-value#serial-auto-increment) .

## January 28, 2025

Feature

You can downgrade your Spanner instance to a lower-tier [edition](https://docs.cloud.google.com/spanner/docs/editions-overview) . For more information, see [Downgrade the edition](https://docs.cloud.google.com/spanner/docs/create-manage-instances#downgrade-edition) .

## January 27, 2025

Feature

Spanner supports the `SELECT…FOR UPDATE` query syntax in GoogleSQL and PostgreSQL-dialect databases. When you use the `SELECT` query to scan a table, add a `FOR UPDATE` clause to enable exclusive locks on the scanned data in order to reduce aborts for workloads that operate on the same data concurrently. This is similar to the `LOCK_SCANNED_RANGES` hint ( [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#statement_hints) and [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/query-syntax#statement-hints) ). For more information, see [Use `SELECT…FOR UPDATE`](https://docs.cloud.google.com/spanner/docs/use-select-for-update) .

## January 15, 2025

Feature

Spanner supports query statistics for previously executed partitioned data manipulation language ( [partitioned DML](https://docs.cloud.google.com/spanner/docs/dml-partitioned) ) statements. For more information, see [Query statistics](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics) .

## January 10, 2025

Feature

Monitor and troubleshoot queries that are running in your Spanner instance. Active queries are long-running queries that might affect the performance of your instance. Monitoring these queries can help you identify causes of instance latency and high CPU usage. For more information, see [Monitor active queries](https://docs.cloud.google.com/spanner/docs/monitor-active-queries) .

## December 30, 2024

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.82.0](https://github.com/googleapis/java-spanner/compare/v6.81.2...v6.82.0) (2024-12-04)

##### Features

- Add option for retrying DML as PDML ( [\#3480](https://github.com/googleapis/java-spanner/issues/3480) ) ( [b545557](https://github.com/googleapis/java-spanner/commit/b545557b1a27868aeb5115b3947d42db015cc00e) )
- Add the last statement option to ExecuteSqlRequest and ExecuteBatchDmlRequest ( [76ab801](https://github.com/googleapis/java-spanner/commit/76ab8011b0aa03e5bb98e375595358732cde31b7) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.50.0 ( [76ab801](https://github.com/googleapis/java-spanner/commit/76ab8011b0aa03e5bb98e375595358732cde31b7) )
- Shutdown built in metrics meter provider ( [\#3518](https://github.com/googleapis/java-spanner/issues/3518) ) ( [c935e2e](https://github.com/googleapis/java-spanner/commit/c935e2eff780100273bc35c11458485c9bb05230) )
- **spanner:** GetEdition() is returning null for Instance ( [\#3496](https://github.com/googleapis/java-spanner/issues/3496) ) ( [77cb585](https://github.com/googleapis/java-spanner/commit/77cb585d57fd30f953b0ffb80be124e3cb1c6f39) )

##### Dependencies

- Update dependency commons-io:commons-io to v2.18.0 ( [\#3492](https://github.com/googleapis/java-spanner/issues/3492) ) ( [5c8b3ad](https://github.com/googleapis/java-spanner/commit/5c8b3ade163b4cdb81a53f5dcf777ebba48ef265) )

##### Documentation

- Add Multi Region Encryption samples ( [\#3524](https://github.com/googleapis/java-spanner/issues/3524) ) ( [316f971](https://github.com/googleapis/java-spanner/commit/316f97146a1fb9f120b642421ec1196be9abddf0) )

#### [6.83.0](https://github.com/googleapis/java-spanner/compare/v6.82.0...v6.83.0) (2024-12-13)

##### Features

- Add Metrics host for built in metrics ( [\#3519](https://github.com/googleapis/java-spanner/issues/3519) ) ( [4ed455a](https://github.com/googleapis/java-spanner/commit/4ed455a43edf7ff8d138ce4d40a52d3224383b14) )
- Add opt-in for using multiplexed sessions for blind writes ( [\#3540](https://github.com/googleapis/java-spanner/issues/3540) ) ( [216f53e](https://github.com/googleapis/java-spanner/commit/216f53e4cbc0150078ece7785da33b342a6ab082) )
- Add UUID in Spanner TypeCode enum ( [41f83dc](https://github.com/googleapis/java-spanner/commit/41f83dcf046f955ec289d4e976f40a03922054cb) )
- Introduce java.time variables and methods ( [\#3495](https://github.com/googleapis/java-spanner/issues/3495) ) ( [8a7d533](https://github.com/googleapis/java-spanner/commit/8a7d533ded21b9b94992b68c702c08bb84474e1b) )
- **spanner:** Support multiplexed session for Partitioned operations ( [\#3231](https://github.com/googleapis/java-spanner/issues/3231) ) ( [4501a3e](https://github.com/googleapis/java-spanner/commit/4501a3ea69a9346e8b95edf6f94ff839b509ec73) )
- Support 'set local' for retry_aborts_internally ( [\#3532](https://github.com/googleapis/java-spanner/issues/3532) ) ( [331942f](https://github.com/googleapis/java-spanner/commit/331942f51b11660b9de9c8fe8aacd6f60ac254b5) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.51.0 ( [41f83dc](https://github.com/googleapis/java-spanner/commit/41f83dcf046f955ec289d4e976f40a03922054cb) )

##### Dependencies

- Update sdk platform java dependencies ( [\#3549](https://github.com/googleapis/java-spanner/issues/3549) ) ( [6235f0f](https://github.com/googleapis/java-spanner/commit/6235f0f2c223718c537addc450fa5910d1500271) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.51.0](https://github.com/googleapis/python-spanner/compare/v3.50.1...v3.51.0) (2024-12-05)

##### Features

- Add connection variable for ignoring transaction warnings ( [\#1249](https://github.com/googleapis/python-spanner/issues/1249) ) ( [eeb7836](https://github.com/googleapis/python-spanner/commit/eeb7836b6350aa9626dfb733208e6827d38bb9c9) )
- **spanner:** Implement custom tracer_provider injection for opentelemetry traces ( [\#1229](https://github.com/googleapis/python-spanner/issues/1229) ) ( [6869ed6](https://github.com/googleapis/python-spanner/commit/6869ed651e41d7a8af046884bc6c792a4177f766) )
- Support float32 parameters in dbapi ( [\#1245](https://github.com/googleapis/python-spanner/issues/1245) ) ( [829b799](https://github.com/googleapis/python-spanner/commit/829b799e0c9c6da274bf95c272cda564cfdba928) )

##### Bug Fixes

- Allow setting connection.read_only to same value ( [\#1247](https://github.com/googleapis/python-spanner/issues/1247) ) ( [5e8ca94](https://github.com/googleapis/python-spanner/commit/5e8ca949b583fbcf0b92b42696545973aad8c78f) )
- Allow setting staleness to same value in tx ( [\#1253](https://github.com/googleapis/python-spanner/issues/1253) ) ( [a214885](https://github.com/googleapis/python-spanner/commit/a214885ed474f3d69875ef580d5f8cbbabe9199a) )
- Dbapi raised AttributeError with \[\] as arguments ( [\#1257](https://github.com/googleapis/python-spanner/issues/1257) ) ( [758bf48](https://github.com/googleapis/python-spanner/commit/758bf4889a7f3346bc8282a3eed47aee43be650c) )

##### Performance Improvements

- Optimize ResultSet decoding ( [\#1244](https://github.com/googleapis/python-spanner/issues/1244) ) ( [ccae6e0](https://github.com/googleapis/python-spanner/commit/ccae6e0287ba6cf3c14f15a907b2106b11ef1fdc) )
- Remove repeated GetSession calls for FixedSizePool ( [\#1252](https://github.com/googleapis/python-spanner/issues/1252) ) ( [c064815](https://github.com/googleapis/python-spanner/commit/c064815abaaa4b564edd6f0e365a37e7e839080c) )

##### Documentation

- **samples:** Add samples for Cloud Spanner Default Backup Schedules ( [\#1238](https://github.com/googleapis/python-spanner/issues/1238) ) ( [054a186](https://github.com/googleapis/python-spanner/commit/054a18658eedc5d4dbecb7508baa3f3d67f5b815) )

## December 12, 2024

Feature

Spanner supports `IDENTITY` columns. `IDENTITY` columns lets you generate unique integer values for key and non-key columns, and aligns with the ANSI standard. For more information, see [`IDENTITY` columns](https://docs.cloud.google.com/spanner/docs/primary-key-default-value#identity-columns) .

## December 02, 2024

Announcement

- [Spanner Graph](https://docs.cloud.google.com/spanner/docs/graph/overview) is Generally Available ( [GA](https://cloud.google.com/products#product-launch-stages) ). For more information, see [Spanner Graph overview](https://docs.cloud.google.com/spanner/docs/graph/overview) .

- [Spanner Graph](https://docs.cloud.google.com/spanner/docs/graph/overview) supports defining path variables and using path functions. For more information, see [Work with paths](https://docs.cloud.google.com/spanner/docs/graph/work-with-paths) .

- Information about how Spanner Graph supports the ISO international standard query language for graph databases is available. For more information, see [Spanner Graph and ISO standards](https://docs.cloud.google.com/spanner/docs/graph/iso-standards) .

- [Spanner Graph](https://docs.cloud.google.com/spanner/docs/graph/overview) supports vector similarity search to find K-nearest neighbors (KNN) and approximate nearest neighbors (ANN). For more information, see [Perform vector similarity search in Spanner Graph](https://docs.cloud.google.com/spanner/docs/graph/perform-vector-similarity-search) .

- [Full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) is available in Spanner Graph. For more information, see [Use full-text search with Spanner Graph](https://docs.cloud.google.com/spanner/docs/graph/full-text-search-and-graph) .

Feature

A predefined Identity and Access Management (IAM) role is available that lets you query a Spanner database using Data Boost. For more information, see details about the [Spanner Database Reader with Data Boost IAM role](https://docs.cloud.google.com/spanner/docs/iam#spanner.databaseReaderWithDataBoost) and [Run federated queries with Data Boost](https://docs.cloud.google.com/spanner/docs/databoost/databoost-run-queries) .

## November 29, 2024

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.71.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.70.0...spanner/v1.71.0) (2024-11-01)

##### Features

- **spanner/admin/instance:** Add support for Cloud Spanner Default Backup Schedules ( [706ecb2](https://github.com/googleapis/google-cloud-go/commit/706ecb2c813da3109035b986a642ca891a33847f) )
- **spanner:** Client built in metrics ( [\#10998](https://github.com/googleapis/google-cloud-go/issues/10998) ) ( [d81a1a7](https://github.com/googleapis/google-cloud-go/commit/d81a1a75b9efbf7104bb077300364f8d63da89b5) )

##### Bug Fixes

- **spanner/test/opentelemetry/test:** Update google.golang.org/api to v0.203.0 ( [8bb87d5](https://github.com/googleapis/google-cloud-go/commit/8bb87d56af1cba736e0fe243979723e747e5e11e) )
- **spanner/test/opentelemetry/test:** WARNING: On approximately Dec 1, 2024, an update to Protobuf will change service registration function signatures to use an interface instead of a concrete type in generated .pb.go files. This change is expected to affect very few if any users of this client library. For more information, see https://togithub.com/googleapis/google-cloud-go/issues/11020. ( [2b8ca4b](https://github.com/googleapis/google-cloud-go/commit/2b8ca4b4127ce3025c7a21cc7247510e07cc5625) )
- **spanner:** Attempt latency for streaming call should capture the total latency till decoding of protos ( [\#11039](https://github.com/googleapis/google-cloud-go/issues/11039) ) ( [255c6bf](https://github.com/googleapis/google-cloud-go/commit/255c6bfcdd3e844dcf602a829bfa2ce495bcd72e) )
- **spanner:** Decode PROTO to custom type variant of base type ( [\#11007](https://github.com/googleapis/google-cloud-go/issues/11007) ) ( [5e363a3](https://github.com/googleapis/google-cloud-go/commit/5e363a31cc9f2616832540ca82aa5cb998a3938c) )
- **spanner:** Update google.golang.org/api to v0.203.0 ( [8bb87d5](https://github.com/googleapis/google-cloud-go/commit/8bb87d56af1cba736e0fe243979723e747e5e11e) )
- **spanner:** WARNING: On approximately Dec 1, 2024, an update to Protobuf will change service registration function signatures to use an interface instead of a concrete type in generated .pb.go files. This change is expected to affect very few if any users of this client library. For more information, see https://togithub.com/googleapis/google-cloud-go/issues/11020. ( [2b8ca4b](https://github.com/googleapis/google-cloud-go/commit/2b8ca4b4127ce3025c7a21cc7247510e07cc5625) )

#### [1.72.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.71.0...spanner/v1.72.0) (2024-11-07)

##### Features

- **spanner/spansql:** Add support for protobuf column types & Proto bundles ( [\#10945](https://github.com/googleapis/google-cloud-go/issues/10945) ) ( [91c6f0f](https://github.com/googleapis/google-cloud-go/commit/91c6f0fcaadfb7bd983e070e6ceffc8aeba7d5a2) ), refs [\#10944](https://github.com/googleapis/google-cloud-go/issues/10944)

##### Bug Fixes

- **spanner:** Skip exporting metrics if attempt or operation is not captured. ( [\#11095](https://github.com/googleapis/google-cloud-go/issues/11095) ) ( [1d074b5](https://github.com/googleapis/google-cloud-go/commit/1d074b520c7a368fb8a7a27574ef56a120665c64) )

#### [1.73.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.72.0...spanner/v1.73.0) (2024-11-14)

##### Features

- **spanner:** Add ResetForRetry method for stmt-based transactions ( [\#10956](https://github.com/googleapis/google-cloud-go/issues/10956) ) ( [02c191c](https://github.com/googleapis/google-cloud-go/commit/02c191c5dc13023857812217f63be2395bfcb382) )

##### Bug Fixes

- **spanner:** Add safecheck to avoid deadlock when creating multiplex session ( [\#11131](https://github.com/googleapis/google-cloud-go/issues/11131) ) ( [8ee5d05](https://github.com/googleapis/google-cloud-go/commit/8ee5d05e288c7105ddb1722071d6719933effea4) )
- **spanner:** Allow non default service account only when direct path is enabled ( [\#11046](https://github.com/googleapis/google-cloud-go/issues/11046) ) ( [4250788](https://github.com/googleapis/google-cloud-go/commit/42507887523f41d0507ca8b1772235846947c3e0) )
- **spanner:** Use spanner options when initializing monitoring exporter ( [\#11109](https://github.com/googleapis/google-cloud-go/issues/11109) ) ( [81413f3](https://github.com/googleapis/google-cloud-go/commit/81413f3647a0ea406f25d4159db19b7ad9f0682b) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.80.1](https://github.com/googleapis/java-spanner/compare/v6.80.0...v6.80.1) (2024-10-28)

##### Dependencies

- Update googleapis/sdk-platform-java action to v2.49.0 ( [\#3430](https://github.com/googleapis/java-spanner/issues/3430) ) ( [beb788c](https://github.com/googleapis/java-spanner/commit/beb788c05d099a0c5edeabb7ed63f4a6a7a24c16) )
- Update sdk platform java dependencies ( [\#3431](https://github.com/googleapis/java-spanner/issues/3431) ) ( [eef03e9](https://github.com/googleapis/java-spanner/commit/eef03e9e5a5ce9d4fcf9728d6b14630bbb99afce) )

#### [6.81.0](https://github.com/googleapis/java-spanner/compare/v6.80.1...v6.81.0) (2024-11-01)

##### Features

- Client built in metrics ( [\#3408](https://github.com/googleapis/java-spanner/issues/3408) ) ( [6a36103](https://github.com/googleapis/java-spanner/commit/6a3610379d1d0eee741d5ef4b30e811ff5a67bc0) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.54.0 ( [\#3439](https://github.com/googleapis/java-spanner/issues/3439) ) ( [cdec63f](https://github.com/googleapis/java-spanner/commit/cdec63f84ef9b615adf19e4611b2dc223eec687b) )

#### [6.81.1](https://github.com/googleapis/java-spanner/compare/v6.81.0...v6.81.1) (2024-11-11)

##### Bug Fixes

- Client built in metrics. Skip export if instance id is null ( [\#3447](https://github.com/googleapis/java-spanner/issues/3447) ) ( [8b2e5ef](https://github.com/googleapis/java-spanner/commit/8b2e5ef5bb391e5a4d4df3cb45d6a3f722a8cfbe) )
- **spanner:** Avoid blocking thread in AsyncResultSet ( [\#3446](https://github.com/googleapis/java-spanner/issues/3446) ) ( [7c82f1c](https://github.com/googleapis/java-spanner/commit/7c82f1c7823d4d529a70c0da231d2593f00b638b) )

##### Dependencies

- Update dependency com.google.api.grpc:proto-google-cloud-monitoring-v3 to v3.54.0 ( [\#3437](https://github.com/googleapis/java-spanner/issues/3437) ) ( [7e28326](https://github.com/googleapis/java-spanner/commit/7e283261961d6435488ed668133dc3bdd238d402) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.54.0 ( [\#3438](https://github.com/googleapis/java-spanner/issues/3438) ) ( [fa18894](https://github.com/googleapis/java-spanner/commit/fa188942c506c85f4c628a8b442b0ee2e6cb845f) )
- Update dependency com.google.cloud:google-cloud-trace to v2.53.0 ( [\#3440](https://github.com/googleapis/java-spanner/issues/3440) ) ( [314eeb8](https://github.com/googleapis/java-spanner/commit/314eeb823e14c386ea6e65caae8c80e908e05600) )
- Update dependency io.opentelemetry:opentelemetry-bom to v1.44.1 ( [\#3452](https://github.com/googleapis/java-spanner/issues/3452) ) ( [6518eea](https://github.com/googleapis/java-spanner/commit/6518eea2921006f1aa431e02754118e3d3d3b620) )
- Update opentelemetry.version to v1.44.1 ( [\#3451](https://github.com/googleapis/java-spanner/issues/3451) ) ( [d9b0271](https://github.com/googleapis/java-spanner/commit/d9b0271603dd14c51954532054b134419150625a) )

##### Documentation

- Update samples' README.md to ensure given ( [\#3420](https://github.com/googleapis/java-spanner/issues/3420) ) ( [663a974](https://github.com/googleapis/java-spanner/commit/663a974dc2a52d773deb620b0bc65f0049f63693) )

#### [6.81.2](https://github.com/googleapis/java-spanner/compare/v6.81.1...v6.81.2) (2024-11-20)

##### Bug Fixes

- Directpath enabled attribute ( [\#3477](https://github.com/googleapis/java-spanner/issues/3477) ) ( [ea1ebad](https://github.com/googleapis/java-spanner/commit/ea1ebadd1ef5d2a343e7117828cae71a798c38eb) )

##### Dependencies

- Update dependency com.google.api.grpc:proto-google-cloud-monitoring-v3 to v3.55.0 ( [\#3482](https://github.com/googleapis/java-spanner/issues/3482) ) ( [bf350b0](https://github.com/googleapis/java-spanner/commit/bf350b024592312b0a00a04c2ab6d3d2312ea686) )
- Update dependency com.google.api.grpc:proto-google-cloud-trace-v1 to v2.53.0 ( [\#3454](https://github.com/googleapis/java-spanner/issues/3454) ) ( [8729b30](https://github.com/googleapis/java-spanner/commit/8729b30a1043a7e77b0277036c70c7c2616d0b47) )
- Update dependency com.google.cloud:google-cloud-trace to v2.53.0 ( [\#3464](https://github.com/googleapis/java-spanner/issues/3464) ) ( [a507e4c](https://github.com/googleapis/java-spanner/commit/a507e4c89bb59d154881812f10cab02d68325a08) )
- Update dependency com.google.cloud:google-cloud-trace to v2.54.0 ( [\#3488](https://github.com/googleapis/java-spanner/issues/3488) ) ( [1d1fecf](https://github.com/googleapis/java-spanner/commit/1d1fecf04a4e800c9b756324914cb1feed7c9866) )
- Update googleapis/sdk-platform-java action to v2.50.0 ( [\#3475](https://github.com/googleapis/java-spanner/issues/3475) ) ( [e992f18](https://github.com/googleapis/java-spanner/commit/e992f18a651ec034b89aa214cb87ec43f33f2f79) )
- Update sdk platform java dependencies ( [\#3476](https://github.com/googleapis/java-spanner/issues/3476) ) ( [acb6446](https://github.com/googleapis/java-spanner/commit/acb6446cb952bdbc54ca1b6c53dc466c72cb55b0) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.15.0](https://github.com/googleapis/nodejs-spanner/compare/v7.14.0...v7.15.0) (2024-10-30)

##### Features

- (observability, samples): add tracing end-to-end sample ( [\#2130](https://github.com/googleapis/nodejs-spanner/issues/2130) ) ( [66d99e8](https://github.com/googleapis/nodejs-spanner/commit/66d99e836cd2bfbb3b0f78980ec2b499f9e5e563) )
- (observability) add spans for BatchTransaction and Table ( [\#2115](https://github.com/googleapis/nodejs-spanner/issues/2115) ) ( [d51aae9](https://github.com/googleapis/nodejs-spanner/commit/d51aae9c9c3c0e6319d81c2809573ae54675acf3) ), closes [\#2114](https://github.com/googleapis/nodejs-spanner/issues/2114)
- (observability) Add support for OpenTelemetry traces and allow observability options to be passed. ( [\#2131](https://github.com/googleapis/nodejs-spanner/issues/2131) ) ( [5237e11](https://github.com/googleapis/nodejs-spanner/commit/5237e118befb4b7fe4aea76a80a91e822d7a22e4) ), closes [\#2079](https://github.com/googleapis/nodejs-spanner/issues/2079)
- (observability) propagate database name for every span generated to aid in quick debugging ( [\#2155](https://github.com/googleapis/nodejs-spanner/issues/2155) ) ( [0342e74](https://github.com/googleapis/nodejs-spanner/commit/0342e74721a0684d8195a6299c3a634eefc2b522) )
- (observability) trace Database.batchCreateSessions + SessionPool.createSessions ( [\#2145](https://github.com/googleapis/nodejs-spanner/issues/2145) ) ( [f489c94](https://github.com/googleapis/nodejs-spanner/commit/f489c9479fa5402f0c960cf896fd3be0e946f182) )
- (observability): trace Database.runPartitionedUpdate ( [\#2176](https://github.com/googleapis/nodejs-spanner/issues/2176) ) ( [701e226](https://github.com/googleapis/nodejs-spanner/commit/701e22660d5ac9f0b3e940ad656b9ca6c479251d) ), closes [\#2079](https://github.com/googleapis/nodejs-spanner/issues/2079)
- (observability): trace Database.runTransactionAsync ( [\#2167](https://github.com/googleapis/nodejs-spanner/issues/2167) ) ( [d0fe178](https://github.com/googleapis/nodejs-spanner/commit/d0fe178623c1c48245d11bcea97fcd340b6615af) ), closes [\#207](https://github.com/googleapis/nodejs-spanner/issues/207)
- Allow multiple KMS keys to create CMEK database/backup ( [\#2099](https://github.com/googleapis/nodejs-spanner/issues/2099) ) ( [51bc8a7](https://github.com/googleapis/nodejs-spanner/commit/51bc8a7445ab8b3d2239493b69d9c271c1086dde) )
- **observability:** Fix bugs found from product review + negative cases ( [\#2158](https://github.com/googleapis/nodejs-spanner/issues/2158) ) ( [cbc86fa](https://github.com/googleapis/nodejs-spanner/commit/cbc86fa80498af6bd745eebb9443612936e26d4e) )
- **observability:** Trace Database methods ( [\#2119](https://github.com/googleapis/nodejs-spanner/issues/2119) ) ( [1f06871](https://github.com/googleapis/nodejs-spanner/commit/1f06871f7aca386756e8691013602b069697bb87) ), closes [\#2114](https://github.com/googleapis/nodejs-spanner/issues/2114)
- **observability:** Trace Database.batchWriteAtLeastOnce ( [\#2157](https://github.com/googleapis/nodejs-spanner/issues/2157) ) ( [2a19ef1](https://github.com/googleapis/nodejs-spanner/commit/2a19ef1af4f6fd1b81d08afc15db76007859a0b9) ), closes [\#2079](https://github.com/googleapis/nodejs-spanner/issues/2079)
- **observability:** Trace Transaction ( [\#2122](https://github.com/googleapis/nodejs-spanner/issues/2122) ) ( [a464bdb](https://github.com/googleapis/nodejs-spanner/commit/a464bdb5cbb7856b7a08dac3ff48132948b65792) ), closes [\#2114](https://github.com/googleapis/nodejs-spanner/issues/2114)

##### Bug Fixes

- Exact staleness timebound ( [\#2143](https://github.com/googleapis/nodejs-spanner/issues/2143) ) ( [f01516e](https://github.com/googleapis/nodejs-spanner/commit/f01516ec6ba44730622cfb050c52cd93f30bba7a) ), closes [\#2129](https://github.com/googleapis/nodejs-spanner/issues/2129)
- GetMetadata for Session ( [\#2124](https://github.com/googleapis/nodejs-spanner/issues/2124) ) ( [2fd63ac](https://github.com/googleapis/nodejs-spanner/commit/2fd63acb87ce06a02d7fdfa78d836dbd7ad59a26) ), closes [\#2123](https://github.com/googleapis/nodejs-spanner/issues/2123)

#### [7.16.0](https://github.com/googleapis/nodejs-spanner/compare/v7.15.0...v7.16.0) (2024-11-09)

##### Features

- **spanner:** Add support for Cloud Spanner Default Backup Schedules ( [\#2135](https://github.com/googleapis/nodejs-spanner/issues/2135) ) ( [19f137c](https://github.com/googleapis/nodejs-spanner/commit/19f137c870796d60902be8d9d3a82f4abcfc693f) )

##### Bug Fixes

- **deps:** Update dependency google-gax to v4.4.1 ( [\#2100](https://github.com/googleapis/nodejs-spanner/issues/2100) ) ( [2e94bcd](https://github.com/googleapis/nodejs-spanner/commit/2e94bcd06d99a98c7767281e8035d000e186692b) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.50.0](https://github.com/googleapis/python-spanner/compare/v3.49.1...v3.50.0) (2024-11-11)

##### Features

- **spanner:** Add support for Cloud Spanner Default Backup Schedules ( [45d4517](https://github.com/googleapis/python-spanner/commit/45d4517789660a803849b829c8eae8b4ea227599) )

##### Bug Fixes

- Add PROTO in streaming chunks ( [\#1213](https://github.com/googleapis/python-spanner/issues/1213) ) ( [43c190b](https://github.com/googleapis/python-spanner/commit/43c190bc694d56e0c57d96dbaa7fc48117f3c971) )
- Pass through route-to-leader option in dbapi ( [\#1223](https://github.com/googleapis/python-spanner/issues/1223) ) ( [ec6c204](https://github.com/googleapis/python-spanner/commit/ec6c204f66e5c8419ea25c4b77f18a38a57acf81) )
- Pin `nox` version in `requirements.in` for devcontainer. ( [\#1215](https://github.com/googleapis/python-spanner/issues/1215) ) ( [41604fe](https://github.com/googleapis/python-spanner/commit/41604fe297d02f5cc2e5516ba24e0fdcceda8e26) )

##### Documentation

- Allow multiple KMS keys to create CMEK database/backup ( [68551c2](https://github.com/googleapis/python-spanner/commit/68551c20cd101045f3d3fe948d04b99388f28c26) )

#### [3.50.1](https://github.com/googleapis/python-spanner/compare/v3.50.0...v3.50.1) (2024-11-14)

##### Bug Fixes

- Json data type for non object values ( [\#1236](https://github.com/googleapis/python-spanner/issues/1236) ) ( [0007be3](https://github.com/googleapis/python-spanner/commit/0007be37a65ff0d4b6b5a1c9ee53d884957c4942) )
- **spanner:** Multi_scm issue in python release ( [\#1230](https://github.com/googleapis/python-spanner/issues/1230) ) ( [6d64e9f](https://github.com/googleapis/python-spanner/commit/6d64e9f5ccc811600b5b51a27c19e84ad5957e2a) )

## November 25, 2024

Feature

Default backup schedules are available and automatically enabled for all new instances. You can enable or disable default backup schedules in an instance when creating the instance or by editing the instance later. You can also let default backup schedules for new databases in existing instances. You can edit or delete the default backup schedule once it's created.

When enabled, Spanner creates a default backup schedule for every new database created in the instance. The default backup schedule creates a full backup every 24 hours. These backups have a retention period of 7 days.

For more information, see [Default backup schedules](https://docs.cloud.google.com/spanner/docs/backup#default-backup-schedules) .

## November 19, 2024

Feature

Spanner supports the [`ALL_DIFFERENT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-operators#all_different_predicate) graph predicate in GoogleSQL-dialect databases. You can use this predicate to see if the graph elements in a list are mutually distinct.

## November 18, 2024

Feature

You can create Spanner [regional instance configurations](https://cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) in Querétaro, Mexico ( `northamerica-south1` ). For more information, see [Google Cloud locations](https://cloud.google.com/about/locations) and [Spanner pricing](https://docs.cloud.google.com/spanner/pricing) .

## November 05, 2024

Feature

Spanner supports client-side metrics for Java and Go applications. These metrics can be used with server-side metrics for faster troubleshooting of performance and latency issues.

These metrics are included in the latest Spanner client libraries for the following languages:

- Java in version 6.81.0 and later
- Go in version 1.71.0 and later

For more information, see [View and manage client-side metrics](https://docs.cloud.google.com/spanner/docs/view-manage-client-side-metrics) .

## October 31, 2024

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.69.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.68.0...spanner/v1.69.0) (2024-10-03)

##### Features

- **spanner:** Add x-goog-spanner-end-to-end-tracing header for requests to Spanner ( [\#10241](https://github.com/googleapis/google-cloud-go/issues/10241) ) ( [7f61cd5](https://github.com/googleapis/google-cloud-go/commit/7f61cd579f7e4ed4f1ac161f2c2a28e931406f16) )

##### Bug Fixes

- **spanner:** Handle errors ( [\#10943](https://github.com/googleapis/google-cloud-go/issues/10943) ) ( [c67f964](https://github.com/googleapis/google-cloud-go/commit/c67f964de364808c02085dda61fa53e2b2fda850) )

##### Performance Improvements

- **spanner:** Use passthrough with emulator endpoint ( [\#10947](https://github.com/googleapis/google-cloud-go/issues/10947) ) ( [9e964dd](https://github.com/googleapis/google-cloud-go/commit/9e964ddc01a54819f25435cfcc9d5b37c91f5a1d) )

#### [1.70.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.69.0...spanner/v1.70.0) (2024-10-14)

##### Features

- **spanner/admin/instance:** Define ReplicaComputeCapacity and AsymmetricAutoscalingOption ( [78d8513](https://github.com/googleapis/google-cloud-go/commit/78d8513f7e31c6ef118bdfc784049b8c7f1e3249) )
- **spanner:** Add INTERVAL API ( [78d8513](https://github.com/googleapis/google-cloud-go/commit/78d8513f7e31c6ef118bdfc784049b8c7f1e3249) )
- **spanner:** Add new QueryMode enum values (WITH_STATS, WITH_PLAN_AND_STATS) ( [78d8513](https://github.com/googleapis/google-cloud-go/commit/78d8513f7e31c6ef118bdfc784049b8c7f1e3249) )

##### Documentation

- **spanner/admin/instance:** A comment for field `node_count` in message `spanner.admin.instance.v1.Instance` is changed ( [78d8513](https://github.com/googleapis/google-cloud-go/commit/78d8513f7e31c6ef118bdfc784049b8c7f1e3249) )
- **spanner/admin/instance:** A comment for field `processing_units` in message `spanner.admin.instance.v1.Instance` is changed ( [78d8513](https://github.com/googleapis/google-cloud-go/commit/78d8513f7e31c6ef118bdfc784049b8c7f1e3249) )
- **spanner:** Update comment for PROFILE QueryMode ( [78d8513](https://github.com/googleapis/google-cloud-go/commit/78d8513f7e31c6ef118bdfc784049b8c7f1e3249) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.77.0](https://github.com/googleapis/java-spanner/compare/v6.76.0...v6.77.0) (2024-10-02)

##### Features

- Add INTERVAL API ( [c078ac3](https://github.com/googleapis/java-spanner/commit/c078ac34c3d14b13bbd4a507de4f0013975dca4e) )

##### Dependencies

- Update dependency com.google.api.grpc:proto-google-cloud-monitoring-v3 to v3.52.0 ( [\#3291](https://github.com/googleapis/java-spanner/issues/3291) ) ( [9241063](https://github.com/googleapis/java-spanner/commit/92410638b0ba88f8e89e28bd12dd58830f7aaeb3) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.52.0 ( [\#3292](https://github.com/googleapis/java-spanner/issues/3292) ) ( [da27a19](https://github.com/googleapis/java-spanner/commit/da27a1992e40b1b4591f0232f687d8031387e749) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.52.0 ( [\#3293](https://github.com/googleapis/java-spanner/issues/3293) ) ( [c6dbdb2](https://github.com/googleapis/java-spanner/commit/c6dbdb255eb4cd231a2dc7cef94bf3353fa7e837) )
- Update dependency com.google.cloud:google-cloud-trace to v2.51.0 ( [\#3294](https://github.com/googleapis/java-spanner/issues/3294) ) ( [a269747](https://github.com/googleapis/java-spanner/commit/a269747889ea0b2380f07e1efef3b288a9c4fd04) )
- Update dependency com.google.cloud:sdk-platform-java-config to v3.36.1 ( [\#3355](https://github.com/googleapis/java-spanner/issues/3355) ) ( [5191e71](https://github.com/googleapis/java-spanner/commit/5191e71a83a316b41564ce2604980c8f33135f2f) )
- Update dependency com.google.cloud.opentelemetry:exporter-metrics to v0.32.0 ( [\#3371](https://github.com/googleapis/java-spanner/issues/3371) ) ( [d5b5ca0](https://github.com/googleapis/java-spanner/commit/d5b5ca0cccc6cf73d759245d2bd72f33c7d39830) )
- Update dependency com.google.cloud.opentelemetry:exporter-trace to v0.32.0 ( [\#3372](https://github.com/googleapis/java-spanner/issues/3372) ) ( [aa9a71d](https://github.com/googleapis/java-spanner/commit/aa9a71d38dabd8d1974bb553761e93735ade5c26) )
- Update dependency commons-io:commons-io to v2.17.0 ( [\#3349](https://github.com/googleapis/java-spanner/issues/3349) ) ( [7c21164](https://github.com/googleapis/java-spanner/commit/7c21164f2b8e75afab268f2fb8e132a372ac0d67) )
- Update dependency io.opentelemetry:opentelemetry-bom to v1.42.1 ( [\#3323](https://github.com/googleapis/java-spanner/issues/3323) ) ( [95dfc02](https://github.com/googleapis/java-spanner/commit/95dfc02ae2d65f99219dcced66cf4e74d1c4975b) )
- Update dependency ubuntu to v24 ( [\#3356](https://github.com/googleapis/java-spanner/issues/3356) ) ( [042c294](https://github.com/googleapis/java-spanner/commit/042c294cc5f83eebd2e3600cffb165e5b467d63e) )
- Update googleapis/sdk-platform-java action to v2.46.1 ( [\#3354](https://github.com/googleapis/java-spanner/issues/3354) ) ( [378f5cf](https://github.com/googleapis/java-spanner/commit/378f5cfb08d4e5ee80b21007bfc829de61bfbdbe) )
- Update junixsocket.version to v2.10.1 ( [\#3367](https://github.com/googleapis/java-spanner/issues/3367) ) ( [5f94915](https://github.com/googleapis/java-spanner/commit/5f94915941c4e4132f8460a04dde0643fa63ab99) )
- Update opentelemetry.version to v1.42.1 ( [\#3330](https://github.com/googleapis/java-spanner/issues/3330) ) ( [7b05e43](https://github.com/googleapis/java-spanner/commit/7b05e4301953364617691e8ae225cea823e3a323) )

##### Documentation

- Update comment for PROFILE QueryMode ( [c078ac3](https://github.com/googleapis/java-spanner/commit/c078ac34c3d14b13bbd4a507de4f0013975dca4e) )

#### [6.78.0](https://github.com/googleapis/java-spanner/compare/v6.77.0...v6.78.0) (2024-10-11)

##### Features

- Define ReplicaComputeCapacity and AsymmetricAutoscalingOption ( [f46a6b3](https://github.com/googleapis/java-spanner/commit/f46a6b34383fe45d63b2db912389b26067f3a853) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.47.0 ( [139a715](https://github.com/googleapis/java-spanner/commit/139a715d3f617b20a00b0cf4f5819e5a61a87c96) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-trace to v2.52.0 ( [\#3393](https://github.com/googleapis/java-spanner/issues/3393) ) ( [79453f9](https://github.com/googleapis/java-spanner/commit/79453f9985eda10631cd29ae58c0cedf234c2e18) )

#### [6.79.0](https://github.com/googleapis/java-spanner/compare/v6.78.0...v6.79.0) (2024-10-11)

##### Features

- Support DML auto-batching in Connection API ( [\#3386](https://github.com/googleapis/java-spanner/issues/3386) ) ( [a1ce267](https://github.com/googleapis/java-spanner/commit/a1ce267cbd4d4c5c638ab7fe0dd5dba24bcfab86) )

##### Dependencies

- Update dependency com.google.api.grpc:proto-google-cloud-monitoring-v3 to v3.53.0 ( [\#3390](https://github.com/googleapis/java-spanner/issues/3390) ) ( [a060e92](https://github.com/googleapis/java-spanner/commit/a060e92141d3dad0db1fc5175416e24a191fa326) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.53.0 ( [\#3391](https://github.com/googleapis/java-spanner/issues/3391) ) ( [7f0927d](https://github.com/googleapis/java-spanner/commit/7f0927d495966d7a2ef9023d65545bfe1fecc20b) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.53.0 ( [\#3392](https://github.com/googleapis/java-spanner/issues/3392) ) ( [fd3e92d](https://github.com/googleapis/java-spanner/commit/fd3e92da940419cd1aed14f770186381d59a2b47) )
- Update dependency com.google.cloud:sdk-platform-java-config to v3.37.0 ( [\#3395](https://github.com/googleapis/java-spanner/issues/3395) ) ( [8ecb1a9](https://github.com/googleapis/java-spanner/commit/8ecb1a901f94d9d49efb0278516428e379803088) )
- Update dependency com.google.cloud.opentelemetry:exporter-metrics to v0.33.0 ( [\#3388](https://github.com/googleapis/java-spanner/issues/3388) ) ( [26aa51d](https://github.com/googleapis/java-spanner/commit/26aa51d561c35295dfb7e2867c3b04b79ce6efc9) )
- Update dependency com.google.cloud.opentelemetry:exporter-trace to v0.33.0 ( [\#3389](https://github.com/googleapis/java-spanner/issues/3389) ) ( [6e34c5a](https://github.com/googleapis/java-spanner/commit/6e34c5a1c20c20a2e994d112f042a59c9b93e1e6) )
- Update googleapis/sdk-platform-java action to v2.47.0 ( [\#3383](https://github.com/googleapis/java-spanner/issues/3383) ) ( [4f0d693](https://github.com/googleapis/java-spanner/commit/4f0d69316a910c23abcb2a142e59bbaf550ca89c) )

#### [6.80.0](https://github.com/googleapis/java-spanner/compare/v6.79.0...v6.80.0) (2024-10-25)

##### Features

- Enabling endToEndTracing support in Connection API ( [\#3412](https://github.com/googleapis/java-spanner/issues/3412) ) ( [16cc6ee](https://github.com/googleapis/java-spanner/commit/16cc6eed58cf735026d7757a28f61f29821a14bf) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.38.0 ( [\#3424](https://github.com/googleapis/java-spanner/issues/3424) ) ( [b727453](https://github.com/googleapis/java-spanner/commit/b727453b93d1089f76e1b908255610cc2796da43) )
- Update dependency io.opentelemetry:opentelemetry-bom to v1.43.0 ( [\#3399](https://github.com/googleapis/java-spanner/issues/3399) ) ( [a755c6c](https://github.com/googleapis/java-spanner/commit/a755c6c2f44cc3eb0f5a54cd58244cebc62b7a4f) )
- Update dependency io.opentelemetry:opentelemetry-sdk-testing to v1.43.0 ( [\#3398](https://github.com/googleapis/java-spanner/issues/3398) ) ( [693243a](https://github.com/googleapis/java-spanner/commit/693243afae34610441345645f627bf199e8ddb8b) )
- Update googleapis/sdk-platform-java action to v2.48.0 ( [\#3422](https://github.com/googleapis/java-spanner/issues/3422) ) ( [d5d1f55](https://github.com/googleapis/java-spanner/commit/d5d1f55d7e8e8f9aa89b7ab9e5f5bd0464bf0e1a) )

##### Documentation

- Fix tracing sample to exit when completed, and use custom monitored resource for export ( [\#3287](https://github.com/googleapis/java-spanner/issues/3287) ) ( [ddb65b1](https://github.com/googleapis/java-spanner/commit/ddb65b197a6f311c2bb8ec9856ea968f3a31d62a) )

## October 28, 2024

Feature

[Query Optimizer version 8](https://docs.cloud.google.com/spanner/docs/query-optimizer/versions#version-8) is available. Version 7 remains the default optimizer version.

## October 18, 2024

Feature

Spanner Graph supports the following functions:

- [`DESTINATION_NODE_ID()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-gql-functions#destination_node_id) : gets a unique identifier for a graph edge's destination node.
- [`ELEMENT_ID()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-gql-functions#element_id) : gets a unique identifier for a graph element.
- [`SOURCE_NODE_ID()`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-gql-functions#source_node_id) : gets a unique identifier for a graph edge's source node.

Feature

Spanner supports customer-managed encryption keys (CMEK) to protect databases in custom, dual-region, and multi-region instance configurations. For more information, see [Customer-managed encryption keys (CMEK) overview](https://docs.cloud.google.com/spanner/docs/cmek) .

## October 17, 2024

Feature

Spanner offers [usage statistics](https://docs.cloud.google.com/spanner/docs/introspection/hot-split-statistics) for [database splits](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#database-splits) along with the associated [System insights](https://docs.cloud.google.com/spanner/docs/find-hotspots-in-database) dashboard to help you identify hotspots on affected rows in your database.

Feature

Directed reads are Generally Available. This feature provides the flexibility to route read-only transactions and single reads to a specific replica type or region in a multi-region instance configuration. For more information, see [Directed reads](https://docs.cloud.google.com/spanner/docs/directed-reads) .

## October 14, 2024

Feature

[Query Optimizer version 7](https://docs.cloud.google.com/spanner/docs/query-optimizer/versions#version-7) is generally available and is the default optimizer version.

## October 10, 2024

Feature

Spanner lets you create incremental backups through a backup schedule. You can specify when and how often backups are created, and how long they're retained.

An incremental backup contains only the data that has changed since the previous backup. Incremental backups typically consume less storage, and can help reduce your storage costs.

Incremental backups are available on the Enterprise and Enterprise Plus editions.

For more information about incremental backups, see [Backups overview](https://docs.cloud.google.com/spanner/docs/backup#incremental-backups) .

Feature

[Spanner](https://docs.cloud.google.com/spanner/docs) is available on [Database Center](https://docs.cloud.google.com/database-center/docs/overview) in [Preview](https://cloud.google.com/products#product-launch-stages) . You can track your Spanner resources in the fleet inventory section and the resource table in the Database Center. You can also use Database Center to monitor the following health issues for your Spanner resources:

- Short backup retention
- Last backup older than 24h
- Not replicating across regions

For more information about Database Center, see [Database Center overview](https://docs.cloud.google.com/database-center/docs/overview) . For more information about health issues supported for Spanner, see [Supported health issues](https://docs.cloud.google.com/database-center/docs/database-health-issues#supported-health-issues) .

Feature

An open source Cassandra to Spanner proxy adapter is available. You can use it to migrate workloads from Cassandra to Spanner without making any changes to your application logic. For more information, see [Cassandra to Spanner proxy adapter](https://github.com/cloudspannerecosystem/cassandra-to-spanner-proxy) .

## October 09, 2024

Feature

Spanner supports a subset of `pg_system_catalog` tables and views. For more information, see [pg_system_catalog tables](https://docs.cloud.google.com/spanner/docs/reference/postgresql/pg-system-catalog-tables) and [pg_system_catalog views](https://docs.cloud.google.com/spanner/docs/reference/postgresql/pg-system-catalog-views) .

## October 07, 2024

Feature

[Full-text search overview](https://docs.cloud.google.com/spanner/docs/full-text-search) is generally available.

Feature

Spanner lets you create and manage backup schedules. You can use backup schedules to meet your organization's data protection and compliance needs. You can specify the following when creating a backup schedule:

- When and how often your databases are backed up.
- The retention duration of the backups created.
- The encryption type of the backups created.

For more information about backup schedules, see [Backups overview](https://docs.cloud.google.com/spanner/docs/backup#backup-schedules) .

## October 04, 2024

Feature

Spanner supports the [`SAFE_TO_JSON`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#safe_to_json) function in GoogleSQL-dialect databases. You can use this function to convert SQL objects to JSON objects. Unlike [`TO_JSON`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#to_json) , this function converts invalid JSON types to JSON null values, rather than errors.

## October 03, 2024

Feature

You can create an [external dataset](https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets) in BigQuery that links to an existing database in Spanner. This feature is in [Preview](https://cloud.google.com/products/#product-launch-stages) .

## October 02, 2024

Feature

You can perform vector similarity search using the Generally Available K-nearest neighbors (KNN) vector distance functions:

- `COSINE_DISTANCE()`
- `EUCLIDEAN_DISTANCE()`
- `DOT_PRODUCT()`

For more information, see [Perform vector similarity search in Spanner by finding the K-nearest neighbors](https://docs.cloud.google.com/spanner/docs/find-k-nearest-neighbors) .

Feature

The `FLOAT32` ( [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-types#floating_point_types) ) and `float4/real` ( [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-types#supported) ) data types are Generally Available.

## October 01, 2024

Feature

Spanner supports end-to-end tracing in preview, along with client-side tracing in the Java and Go client libraries. You can opt-in for end-to-end traces to have more visibility into the application to Spanner latencies. For more information, see [Trace collection overview](https://docs.cloud.google.com/spanner/docs/tracing-overview) .

## September 30, 2024

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.68.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.67.0...spanner/v1.68.0) (2024-09-25)

##### Features

- **spanner:** Add support for Go 1.23 iterators ( [84461c0](https://github.com/googleapis/google-cloud-go/commit/84461c0ba464ec2f951987ba60030e37c8a8fc18) )

##### Bug Fixes

- **spanner/test:** Bump dependencies ( [2ddeb15](https://github.com/googleapis/google-cloud-go/commit/2ddeb1544a53188a7592046b98913982f1b0cf04) )
- **spanner:** Bump dependencies ( [2ddeb15](https://github.com/googleapis/google-cloud-go/commit/2ddeb1544a53188a7592046b98913982f1b0cf04) )
- **spanner:** Check errors in tests ( [\#10738](https://github.com/googleapis/google-cloud-go/issues/10738) ) ( [971bfb8](https://github.com/googleapis/google-cloud-go/commit/971bfb85ee7bf8c636117a6424280a4323b5fb3c) )
- **spanner:** Enable toStruct support for structs with proto message pointer fields ( [\#10704](https://github.com/googleapis/google-cloud-go/issues/10704) ) ( [42cdde6](https://github.com/googleapis/google-cloud-go/commit/42cdde6ee34fc9058dc47c9c9ab39ba91b6b9c58) )
- **spanner:** Ensure defers run at the right time in tests ( [\#9759](https://github.com/googleapis/google-cloud-go/issues/9759) ) ( [7ef0ded](https://github.com/googleapis/google-cloud-go/commit/7ef0ded2502dbb37f07bc93bc2e868e29f7121c4) )
- **spanner:** Increase spanner ping timeout to give backend more time to process executeSQL requests ( [\#10874](https://github.com/googleapis/google-cloud-go/issues/10874) ) ( [6997991](https://github.com/googleapis/google-cloud-go/commit/6997991e2325e7a66d3ffa60c27622a1a13041a8) )
- **spanner:** Json null handling ( [\#10660](https://github.com/googleapis/google-cloud-go/issues/10660) ) ( [4c519e3](https://github.com/googleapis/google-cloud-go/commit/4c519e37a124defc3451adfdbd0883a5e081eb2f) )
- **spanner:** Support custom encoding and decoding of protos ( [\#10799](https://github.com/googleapis/google-cloud-go/issues/10799) ) ( [d410907](https://github.com/googleapis/google-cloud-go/commit/d410907f3e52bcc64bd92e0a341777c1277a6418) )
- **spanner:** Unnecessary string formatting fixes ( [\#10736](https://github.com/googleapis/google-cloud-go/issues/10736) ) ( [1efe5c4](https://github.com/googleapis/google-cloud-go/commit/1efe5c4275dca6d739691e89b8d460b97160d953) )
- **spanner:** Wait for things to complete ( [\#10095](https://github.com/googleapis/google-cloud-go/issues/10095) ) ( [7785cad](https://github.com/googleapis/google-cloud-go/commit/7785cad89effbc8c4e67043368f96d4768cdb40f) )

##### Performance Improvements

- **spanner:** Better error handling ( [\#10734](https://github.com/googleapis/google-cloud-go/issues/10734) ) ( [c342f65](https://github.com/googleapis/google-cloud-go/commit/c342f6550c24e3a16e32d1cd61c6fcfeaed77c7b) ), refs [\#9749](https://github.com/googleapis/google-cloud-go/issues/9749)

##### Documentation

- **spanner:** Fix Key related document code to add package name ( [\#10711](https://github.com/googleapis/google-cloud-go/issues/10711) ) ( [bbe7b9c](https://github.com/googleapis/google-cloud-go/commit/bbe7b9ceed1deb85a4f40ea95572595ce63ff002) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.74.0](https://github.com/googleapis/java-spanner/compare/v6.73.0...v6.74.0) (2024-08-27)

##### Features

- **spanner:** Add edition field to the instance proto ( [6b7e6ca](https://github.com/googleapis/java-spanner/commit/6b7e6ca109ea9679b5e36598d3c343fa40bff724) )

##### Documentation

- Change the example timestamps in Spanner Graph java sample code ( [\#3295](https://github.com/googleapis/java-spanner/issues/3295) ) ( [b6490b6](https://github.com/googleapis/java-spanner/commit/b6490b6a6ee2b7399431881a5e87b5ef7b577c89) )

#### [6.74.1](https://github.com/googleapis/java-spanner/compare/v6.74.0...v6.74.1) (2024-09-16)

##### Bug Fixes

- Use core pool size 1 for maintainer ( [\#3314](https://github.com/googleapis/java-spanner/issues/3314) ) ( [cce008d](https://github.com/googleapis/java-spanner/commit/cce008d212535d32da990242973f7f517ca5d6dc) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.35.0 ( [\#3329](https://github.com/googleapis/java-spanner/issues/3329) ) ( [654835f](https://github.com/googleapis/java-spanner/commit/654835f2433b97665c74be9ec80c169ac905a720) )

#### [6.75.0](https://github.com/googleapis/java-spanner/compare/v6.74.1...v6.75.0) (2024-09-19)

##### Features

- Support multiplexed session for blind write with single use transaction ( [\#3229](https://github.com/googleapis/java-spanner/issues/3229) ) ( [b3e2b0f](https://github.com/googleapis/java-spanner/commit/b3e2b0f4892951867715cb7f354c089fca4f050f) )

#### [6.76.0](https://github.com/googleapis/java-spanner/compare/v6.75.0...v6.76.0) (2024-09-27)

##### Features

- Add opt-in flag and ClientInterceptor to propagate trace context for Spanner end to end tracing ( [\#3162](https://github.com/googleapis/java-spanner/issues/3162) ) ( [0b7fdaf](https://github.com/googleapis/java-spanner/commit/0b7fdaf1d25e81ca8dd35a0f8d8caa7b77a7e58c) )
- Add samples for backup schedule feature APIs. ( [\#3339](https://github.com/googleapis/java-spanner/issues/3339) ) ( [8cd5163](https://github.com/googleapis/java-spanner/commit/8cd516351e7859a81f00f17cb5071edbd804ea90) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.46.1 ( [1719f44](https://github.com/googleapis/java-spanner/commit/1719f4465841354db3253fd132868394e530a82d) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.49.0](https://github.com/googleapis/python-spanner/compare/v3.48.0...v3.49.0) (2024-08-27)

##### Features

- Create a few code snippets as examples for using Spanner Graph in Python ( [\#1186](https://github.com/googleapis/python-spanner/issues/1186) ) ( [f886ebd](https://github.com/googleapis/python-spanner/commit/f886ebd80a6422c2167cd440a2a646f52701b684) )
- **spanner:** Add resource reference annotation to backup schedules ( [\#1176](https://github.com/googleapis/python-spanner/issues/1176) ) ( [b503fc9](https://github.com/googleapis/python-spanner/commit/b503fc95d8abd47869a24f0e824a227a281282d6) )
- **spanner:** Add samples for instance partitions ( [\#1168](https://github.com/googleapis/python-spanner/issues/1168) ) ( [55f83dc](https://github.com/googleapis/python-spanner/commit/55f83dc5f776d436b30da6056a9cdcad3971ce39) )

##### Bug Fixes

- JsonObject init when called on JsonObject of list ( [\#1166](https://github.com/googleapis/python-spanner/issues/1166) ) ( [c4af6f0](https://github.com/googleapis/python-spanner/commit/c4af6f09a449f293768f70a84e805ffe08c6c2fb) )

#### [3.49.1](https://github.com/googleapis/python-spanner/compare/v3.49.0...v3.49.1) (2024-09-06)

##### Bug Fixes

- Revert "chore(spanner): Issue [\#1143](https://github.com/googleapis/python-spanner/issues/1143) - Update dependency" ( [92f05ed](https://github.com/googleapis/python-spanner/commit/92f05ed04e49adfe0ad68bfa52e855baf8b17643) )

## September 25, 2024

Feature

Spanner supports the [`spanner.farm_fingerprint()`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#hash-functions) hash function in PostgreSQL-dialect databases.

## September 24, 2024

Feature

Spanner offers editions, a tier-based pricing model that provides greater flexibility, better cost transparency, and opportunities for cost savings. You can choose between the *Standard* , *Enterprise* , and *Enterprise Plus* editions, letting you pick the right set of capabilities to fit your needs and budget. To learn more, see [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) and this [blog](https://cloud.google.com/blog/products/databases/announcing-spanner-editions?e=48754805) post.

Feature

Spanner is enabled for use with Cloud KMS Autokey.

Using keys generated by Autokey can help you consistently align with industry standards and recommended practices for data security, including the HSM protection level, separation of duties, key rotation, location, and key specificity. Keys requested using Autokey function identically to other Cloud HSM keys with the same settings.

For more information, see [Customer-managed encryption keys (CMEK) overview](https://docs.cloud.google.com/spanner/docs/cmek) . To learn more about Cloud KMS Autokey, see the [Autokey overview](https://docs.cloud.google.com/kms/docs/autokey-overview) .

## August 30, 2024

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.65.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.64.0...spanner/v1.65.0) (2024-07-29)

##### Features

- **spanner:** Add RESOURCE_EXHAUSTED to retryable transaction codes ( [\#10412](https://github.com/googleapis/google-cloud-go/issues/10412) ) ( [29b52dc](https://github.com/googleapis/google-cloud-go/commit/29b52dc40f3d1a6ffe7fa40e6142d8035c0d95ee) )

##### Bug Fixes

- **spanner/test:** Bump google.golang.org/api@v0.187.0 ( [8fa9e39](https://github.com/googleapis/google-cloud-go/commit/8fa9e398e512fd8533fd49060371e61b5725a85b) )
- **spanner/test:** Bump google.golang.org/grpc@v1.64.1 ( [8ecc4e9](https://github.com/googleapis/google-cloud-go/commit/8ecc4e9622e5bbe9b90384d5848ab816027226c5) )
- **spanner/test:** Update dependencies ( [257c40b](https://github.com/googleapis/google-cloud-go/commit/257c40bd6d7e59730017cf32bda8823d7a232758) )
- **spanner:** Bump google.golang.org/api@v0.187.0 ( [8fa9e39](https://github.com/googleapis/google-cloud-go/commit/8fa9e398e512fd8533fd49060371e61b5725a85b) )
- **spanner:** Bump google.golang.org/grpc@v1.64.1 ( [8ecc4e9](https://github.com/googleapis/google-cloud-go/commit/8ecc4e9622e5bbe9b90384d5848ab816027226c5) )
- **spanner:** Fix negative values for max_in_use_sessions metrics [\#10449](https://github.com/googleapis/google-cloud-go/issues/10449) ( [\#10508](https://github.com/googleapis/google-cloud-go/issues/10508) ) ( [4e180f4](https://github.com/googleapis/google-cloud-go/commit/4e180f4539012eb6e3d1d2788e68b291ef7230c3) )
- **spanner:** HealthCheck should not decrement num_in_use sessions ( [\#10480](https://github.com/googleapis/google-cloud-go/issues/10480) ) ( [9b2b47f](https://github.com/googleapis/google-cloud-go/commit/9b2b47f107153d624d56709d9a8e6a6b72c39447) )
- **spanner:** Update dependencies ( [257c40b](https://github.com/googleapis/google-cloud-go/commit/257c40bd6d7e59730017cf32bda8823d7a232758) )

#### [1.66.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.65.0...spanner/v1.66.0) (2024-08-07)

##### Features

- **spanner:** Add support of multiplexed session support in writeAtleastOnce mutations ( [\#10646](https://github.com/googleapis/google-cloud-go/issues/10646) ) ( [54009ea](https://github.com/googleapis/google-cloud-go/commit/54009eab1c3b11a28531ad9e621917d01c9e5339) )
- **spanner:** Add support of using multiplexed session with ReadOnlyTransactions ( [\#10269](https://github.com/googleapis/google-cloud-go/issues/10269) ) ( [7797022](https://github.com/googleapis/google-cloud-go/commit/7797022e51d1ac07b8d919c421a8bfdf34a1d53c) )

#### [1.67.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.66.0...spanner/v1.67.0) (2024-08-15)

##### Features

- **spanner/admin/database:** Add resource reference annotation to backup schedules ( [\#10677](https://github.com/googleapis/google-cloud-go/issues/10677) ) ( [6593c0d](https://github.com/googleapis/google-cloud-go/commit/6593c0d62d48751c857bce3d3f858127467a4489) )
- **spanner/admin/instance:** Add edition field to the instance proto ( [6593c0d](https://github.com/googleapis/google-cloud-go/commit/6593c0d62d48751c857bce3d3f858127467a4489) )
- **spanner:** Support commit options in mutation operations. ( [\#10668](https://github.com/googleapis/google-cloud-go/issues/10668) ) ( [62a56f9](https://github.com/googleapis/google-cloud-go/commit/62a56f953d3b8fe82083c42926831c2728312b9c) )

##### Bug Fixes

- **spanner/test/opentelemetry/test:** Update google.golang.org/api to v0.191.0 ( [5b32644](https://github.com/googleapis/google-cloud-go/commit/5b32644eb82eb6bd6021f80b4fad471c60fb9d73) )
- **spanner:** Update google.golang.org/api to v0.191.0 ( [5b32644](https://github.com/googleapis/google-cloud-go/commit/5b32644eb82eb6bd6021f80b4fad471c60fb9d73) )

##### Documentation

- **spanner/admin/database:** Add an example to filter backups based on schedule name ( [6593c0d](https://github.com/googleapis/google-cloud-go/commit/6593c0d62d48751c857bce3d3f858127467a4489) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.72.0](https://github.com/googleapis/java-spanner/compare/v6.71.0...v6.72.0) (2024-08-07)

##### Features

- Add `RESOURCE_EXHAUSTED` to the list of retryable error codes ( [e859b29](https://github.com/googleapis/java-spanner/commit/e859b29ccf4e68b1ab62cffdd4cf197011ba9878) )
- Add field order_by in spanner.proto ( [e859b29](https://github.com/googleapis/java-spanner/commit/e859b29ccf4e68b1ab62cffdd4cf197011ba9878) )
- Add QueryCancellationAction message in executor protos ( [e859b29](https://github.com/googleapis/java-spanner/commit/e859b29ccf4e68b1ab62cffdd4cf197011ba9878) )
- Add SessionPoolOptions, SpannerOptions protos in executor protos ( [e859b29](https://github.com/googleapis/java-spanner/commit/e859b29ccf4e68b1ab62cffdd4cf197011ba9878) )
- Add support for multi region encryption config ( [e859b29](https://github.com/googleapis/java-spanner/commit/e859b29ccf4e68b1ab62cffdd4cf197011ba9878) )
- Enable hermetic library generation ( [\#3129](https://github.com/googleapis/java-spanner/issues/3129) ) ( [94b2a86](https://github.com/googleapis/java-spanner/commit/94b2a8610ac02d2b4212c421f03b4e9561ec9949) )
- **spanner:** Add samples for instance partitions ( [\#3221](https://github.com/googleapis/java-spanner/issues/3221) ) ( [bc48bf2](https://github.com/googleapis/java-spanner/commit/bc48bf212e37441221b3b6c8742b07ff601f6c41) )
- **spanner:** Adding `EXPECTED_FULFILLMENT_PERIOD` to the indicate instance creation times (with `FULFILLMENT_PERIOD_NORMAL` or `FULFILLMENT_PERIOD_EXTENDED` ENUM) with the extended instance creation time triggered by On-Demand Capacity Feature ( [e859b29](https://github.com/googleapis/java-spanner/commit/e859b29ccf4e68b1ab62cffdd4cf197011ba9878) )
- **spanner:** Set manual affinity incase of gRPC-GCP extenstion ( [\#3215](https://github.com/googleapis/java-spanner/issues/3215) ) ( [86b306a](https://github.com/googleapis/java-spanner/commit/86b306a4189483a5fd2746052bed817443630567) )
- Support Read RPC OrderBy ( [\#3180](https://github.com/googleapis/java-spanner/issues/3180) ) ( [735bca5](https://github.com/googleapis/java-spanner/commit/735bca523e4ea53a24929fb2c27d282c41350e91) )

##### Bug Fixes

- Make sure commitAsync always finishes ( [\#3216](https://github.com/googleapis/java-spanner/issues/3216) ) ( [440c88b](https://github.com/googleapis/java-spanner/commit/440c88bd67e1c9d08445fe26b01bf243f7fd1ca4) )
- SessionPoolOptions.Builder#toBuilder() skipped useMultiplexedSessions ( [\#3197](https://github.com/googleapis/java-spanner/issues/3197) ) ( [027f92c](https://github.com/googleapis/java-spanner/commit/027f92cf32fee8217d2075db61fe0be58d43a40d) )

##### Dependencies

- Bump sdk-platform-java-config to 3.33.0 ( [\#3243](https://github.com/googleapis/java-spanner/issues/3243) ) ( [35907c6](https://github.com/googleapis/java-spanner/commit/35907c63ae981612ba24dd9605db493b5b864217) )
- Update dependencies to latest ( [\#3250](https://github.com/googleapis/java-spanner/issues/3250) ) ( [d1d566b](https://github.com/googleapis/java-spanner/commit/d1d566b096915a537e0978715c81bfca00e34ceb) )
- Update dependency com.google.auto.value:auto-value-annotations to v1.11.0 ( [\#3191](https://github.com/googleapis/java-spanner/issues/3191) ) ( [065cd48](https://github.com/googleapis/java-spanner/commit/065cd489964aaee42fffe1e71327906bde907205) )
- Update dependency com.google.cloud:google-cloud-trace to v2.47.0 ( [\#3067](https://github.com/googleapis/java-spanner/issues/3067) ) ( [e336ab8](https://github.com/googleapis/java-spanner/commit/e336ab81a1d392d56386f9302bf51bf14e385dad) )

#### [6.73.0](https://github.com/googleapis/java-spanner/compare/v6.72.0...v6.73.0) (2024-08-22)

##### Features

- Add option for cancelling queries when closing client ( [\#3276](https://github.com/googleapis/java-spanner/issues/3276) ) ( [95da1ed](https://github.com/googleapis/java-spanner/commit/95da1eddbc979f4ce78c9d1ac15bc4c1faba6dca) )

##### Bug Fixes

- GitHub workflow vulnerable to script injection ( [\#3232](https://github.com/googleapis/java-spanner/issues/3232) ) ( [599255c](https://github.com/googleapis/java-spanner/commit/599255c36d1fbe8317705a7eeb2a9e400c3efd15) )
- Make DecodeMode.DIRECT the deafult ( [\#3280](https://github.com/googleapis/java-spanner/issues/3280) ) ( [f31a95a](https://github.com/googleapis/java-spanner/commit/f31a95ab105407305e988e86c8f7b0d8654995e0) )
- Synchronize lazy ResultSet decoding ( [\#3267](https://github.com/googleapis/java-spanner/issues/3267) ) ( [4219cf8](https://github.com/googleapis/java-spanner/commit/4219cf86dba5e44d55f13ab118113f119c92b9e9) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.34.0 ( [\#3277](https://github.com/googleapis/java-spanner/issues/3277) ) ( [c449a91](https://github.com/googleapis/java-spanner/commit/c449a91628b005481996bce5ab449d62496a4d2d) )
- Update dependency commons-cli:commons-cli to v1.9.0 ( [\#3275](https://github.com/googleapis/java-spanner/issues/3275) ) ( [84790f7](https://github.com/googleapis/java-spanner/commit/84790f7d437e88739487b148bf963f0ac9dc3f96) )
- Update dependency io.opentelemetry:opentelemetry-bom to v1.41.0 ( [\#3269](https://github.com/googleapis/java-spanner/issues/3269) ) ( [a7458e9](https://github.com/googleapis/java-spanner/commit/a7458e970e4ca55ff3e312b2129e890576145db1) )
- Update dependency org.hamcrest:hamcrest to v3 ( [\#3271](https://github.com/googleapis/java-spanner/issues/3271) ) ( [fc2e343](https://github.com/googleapis/java-spanner/commit/fc2e343dc06f80617a2cd6f2bea59b0631e70678) )
- Update dependency org.junit.vintage:junit-vintage-engine to v5.11.0 ( [\#3272](https://github.com/googleapis/java-spanner/issues/3272) ) ( [1bc0c46](https://github.com/googleapis/java-spanner/commit/1bc0c469b99ebf3778592b04dbf175b00bf5b06e) )
- Update opentelemetry.version to v1.41.0 ( [\#3270](https://github.com/googleapis/java-spanner/issues/3270) ) ( [88f6b56](https://github.com/googleapis/java-spanner/commit/88f6b56fb243bb17b814a7ae150c8f38dced119a) )

##### Documentation

- Create a few code snippets as examples for using Spanner Graph using Java ( [\#3234](https://github.com/googleapis/java-spanner/issues/3234) ) ( [61f0ab7](https://github.com/googleapis/java-spanner/commit/61f0ab7a48bc3e51b830534b1cfa70e40166ec91) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.11.0](https://github.com/googleapis/nodejs-spanner/compare/v7.10.0...v7.11.0) (2024-07-29)

##### Features

- Add support for blind writes ( [\#2065](https://github.com/googleapis/nodejs-spanner/issues/2065) ) ( [62fc0a4](https://github.com/googleapis/nodejs-spanner/commit/62fc0a47327017c115466b9e89e53dbd778579af) )
- **spanner:** Add samples for instance partitions ( [\#2083](https://github.com/googleapis/nodejs-spanner/issues/2083) ) ( [b91e284](https://github.com/googleapis/nodejs-spanner/commit/b91e2849056df9894e0590cb71e21c13319e6d70) )

#### [7.12.0](https://github.com/googleapis/nodejs-spanner/compare/v7.11.0...v7.12.0) (2024-08-02)

##### Features

- Grpc keep alive settings ( [\#2086](https://github.com/googleapis/nodejs-spanner/issues/2086) ) ( [7712c35](https://github.com/googleapis/nodejs-spanner/commit/7712c35be21863015bb709f5f89d9ef0bb656024) )

#### [7.13.0](https://github.com/googleapis/nodejs-spanner/compare/v7.12.0...v7.13.0) (2024-08-09)

##### Bug Fixes

- Unhandled exception error catch ( [\#2091](https://github.com/googleapis/nodejs-spanner/issues/2091) ) ( [e277752](https://github.com/googleapis/nodejs-spanner/commit/e277752fad961908e37e37d88d7b6a61d61a078e) )

#### [7.14.0](https://github.com/googleapis/nodejs-spanner/compare/v7.13.0...v7.14.0) (2024-08-14)

##### Features

- **spanner:** Add resource reference annotation to backup schedules ( [\#2093](https://github.com/googleapis/nodejs-spanner/issues/2093) ) ( [df539e6](https://github.com/googleapis/nodejs-spanner/commit/df539e665fe5d8fe01084b8d8cf6094c89b13d48) )

##### Bug Fixes

- **deps:** Update dependency google-gax to v4.3.9 ( [\#2094](https://github.com/googleapis/nodejs-spanner/issues/2094) ) ( [487efc0](https://github.com/googleapis/nodejs-spanner/commit/487efc091e0e143d3c59ac63d66005133b1ef2e5) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.48.0](https://github.com/googleapis/python-spanner/compare/v3.47.0...v3.48.0) (2024-07-30)

##### Features

- Add field lock_hint in spanner.proto ( [9609ad9](https://github.com/googleapis/python-spanner/commit/9609ad96d062fbd8fa4d622bfe8da119329facc0) )
- Add field order_by in spanner.proto ( [9609ad9](https://github.com/googleapis/python-spanner/commit/9609ad96d062fbd8fa4d622bfe8da119329facc0) )
- **spanner:** Add support for txn changstream exclusion ( [\#1152](https://github.com/googleapis/python-spanner/issues/1152) ) ( [00ccb7a](https://github.com/googleapis/python-spanner/commit/00ccb7a5c1f246b5099265058a5e9875e6627024) )

##### Bug Fixes

- Allow protobuf 5.x ( [9609ad9](https://github.com/googleapis/python-spanner/commit/9609ad96d062fbd8fa4d622bfe8da119329facc0) )
- **spanner:** Unskip emulator tests for proto ( [\#1145](https://github.com/googleapis/python-spanner/issues/1145) ) ( [cb74679](https://github.com/googleapis/python-spanner/commit/cb74679a05960293dd03eb6b74bff0f68a46395c) )

## August 13, 2024

Feature

The following [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-multi-region) is available in North America - `nam16` (Iowa/Northern Virginia/Columbus).

## August 01, 2024

Feature

Spanner offers Spanner Graph in [Preview](https://cloud.google.com/products#product-launch-stages) , which unites purpose-built graph database capabilities with Spanner. Spanner Graph includes a graph query interface compatible with the ISO Spanner Graph Language (GQL) standards, and interoperability between relational and graph models. For more information, see the following:

- [Set up and query Spanner Graph using the Google Cloud console](https://docs.cloud.google.com/spanner/docs/graph/set-up)
- [Spanner Graph overview](https://docs.cloud.google.com/spanner/docs/graph/overview)
- [Spanner Graph Language reference](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-intro)

Feature

Spanner full-text search ( [Preview](https://cloud.google.com/products#product-launch-stages) ) lets you search a table to find words, phrases, or integers, instead of just searching for exact matches in structured fields. Spanner full-text search capabilities also include making spelling corrections, automating language detection of search input, and ranking search results. To learn more, see the [Full-text search overview](https://docs.cloud.google.com/spanner/docs/full-text-search) .

## July 31, 2024

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.64.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.63.0...spanner/v1.64.0) (2024-06-29)

##### Features

- **spanner:** Add field lock_hint in spanner.proto ( [3df3c04](https://github.com/googleapis/google-cloud-go/commit/3df3c04f0dffad3fa2fe272eb7b2c263801b9ada) )
- **spanner:** Add field order_by in spanner.proto ( [3df3c04](https://github.com/googleapis/google-cloud-go/commit/3df3c04f0dffad3fa2fe272eb7b2c263801b9ada) )
- **spanner:** Add LockHint feature ( [\#10382](https://github.com/googleapis/google-cloud-go/issues/10382) ) ( [64bdcb1](https://github.com/googleapis/google-cloud-go/commit/64bdcb1a6a462d41a62d3badea6814425e271f22) )
- **spanner:** Add OrderBy feature ( [\#10289](https://github.com/googleapis/google-cloud-go/issues/10289) ) ( [07b8bd2](https://github.com/googleapis/google-cloud-go/commit/07b8bd2f5dc738e0293305dfc459c13632d5ea65) )
- **spanner:** Add support of checking row not found errors from ReadRow and ReadRowUsingIndex ( [\#10405](https://github.com/googleapis/google-cloud-go/issues/10405) ) ( [5cb0c26](https://github.com/googleapis/google-cloud-go/commit/5cb0c26013eeb3bbe51174bee628a20c2ec775e0) )

##### Bug Fixes

- **spanner:** Fix data-race caused by TrackSessionHandle ( [\#10321](https://github.com/googleapis/google-cloud-go/issues/10321) ) ( [23c5fff](https://github.com/googleapis/google-cloud-go/commit/23c5fffd06bcde408db50a981c015921cd4ecf0e) ), refs [\#10320](https://github.com/googleapis/google-cloud-go/issues/10320)
- **spanner:** Fix negative values for max_in_use_sessions metrics ( [\#10449](https://github.com/googleapis/google-cloud-go/issues/10449) ) ( [a1e198a](https://github.com/googleapis/google-cloud-go/commit/a1e198a9b18bd2f92c3438e4f609412047f8ccf4) )
- **spanner:** Prevent possible panic for Session not found errors ( [\#10386](https://github.com/googleapis/google-cloud-go/issues/10386) ) ( [ba9711f](https://github.com/googleapis/google-cloud-go/commit/ba9711f87ec871153ae00cfd0827bce17c31ee9c) ), refs [\#10385](https://github.com/googleapis/google-cloud-go/issues/10385)

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.70.0](https://github.com/googleapis/java-spanner/compare/v6.69.0...v6.70.0) (2024-06-27)

##### Features

- Add field order_by in spanner.proto ( [\#3064](https://github.com/googleapis/java-spanner/issues/3064) ) ( [52ee196](https://github.com/googleapis/java-spanner/commit/52ee1967ee3a37fb0482ad8b51c6e77e28b79844) )

##### Bug Fixes

- Do not end transaction span when rolling back to savepoint ( [\#3167](https://github.com/googleapis/java-spanner/issues/3167) ) ( [8ec0cf2](https://github.com/googleapis/java-spanner/commit/8ec0cf2032dece545c9e4d8a794b80d06550b710) )
- Remove unused DmlBatch span ( [\#3147](https://github.com/googleapis/java-spanner/issues/3147) ) ( [f7891c1](https://github.com/googleapis/java-spanner/commit/f7891c1ca42727c775cdbe91bff8d55191a3d799) )

##### Dependencies

- Update dependencies ( [\#3181](https://github.com/googleapis/java-spanner/issues/3181) ) ( [0c787e6](https://github.com/googleapis/java-spanner/commit/0c787e6fa67d2a259a76bbd2d7f1cfa20a1dbee8) )
- Update dependency com.google.cloud:sdk-platform-java-config to v3.32.0 ( [\#3184](https://github.com/googleapis/java-spanner/issues/3184) ) ( [9c85a6f](https://github.com/googleapis/java-spanner/commit/9c85a6fabea527253ea40a8970cc9071804d94c4) )
- Update dependency commons-cli:commons-cli to v1.8.0 ( [\#3073](https://github.com/googleapis/java-spanner/issues/3073) ) ( [36b5340](https://github.com/googleapis/java-spanner/commit/36b5340ef8bf197fbc8ed882f76caff9a6fe84b6) )

#### [6.71.0](https://github.com/googleapis/java-spanner/compare/v6.70.0...v6.71.0) (2024-07-03)

##### Features

- Include thread name in traces ( [\#3173](https://github.com/googleapis/java-spanner/issues/3173) ) ( [92b1e07](https://github.com/googleapis/java-spanner/commit/92b1e079e6093bc4a2e7b458c1bbe0f62a0fada9) )
- Support multiplexed sessions for RO transactions ( [\#3141](https://github.com/googleapis/java-spanner/issues/3141) ) ( [2b8e9ed](https://github.com/googleapis/java-spanner/commit/2b8e9ededc1ea1a5e8d4f90083f2cf862fcc198a) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.9.0](https://github.com/googleapis/nodejs-spanner/compare/v7.8.0...v7.9.0) (2024-06-21)

##### Features

- **spanner:** Add support for batchWrite ( [\#2054](https://github.com/googleapis/nodejs-spanner/issues/2054) ) ( [06aab6e](https://github.com/googleapis/nodejs-spanner/commit/06aab6e39bbce9e3786f1ac631c80e8909197e92) )

##### Bug Fixes

- **deps:** Update dependency google-gax to v4.3.4 ( [\#2051](https://github.com/googleapis/nodejs-spanner/issues/2051) ) ( [80abf06](https://github.com/googleapis/nodejs-spanner/commit/80abf06ba8ef9497318ffc597b83fb63e4408f9c) )
- **deps:** Update dependency google-gax to v4.3.5 ( [\#2055](https://github.com/googleapis/nodejs-spanner/issues/2055) ) ( [702c9b0](https://github.com/googleapis/nodejs-spanner/commit/702c9b0f34e6cc34233c5aa52b97601b19f70980) )
- **deps:** Update dependency google-gax to v4.3.6 ( [\#2057](https://github.com/googleapis/nodejs-spanner/issues/2057) ) ( [74ebf1e](https://github.com/googleapis/nodejs-spanner/commit/74ebf1e45cddf614c180295f3a761a8f84c5cb32) )
- **deps:** Update dependency google-gax to v4.3.7 ( [\#2068](https://github.com/googleapis/nodejs-spanner/issues/2068) ) ( [28fec6c](https://github.com/googleapis/nodejs-spanner/commit/28fec6ca505d78d725efc123950be978e0c84ab7) )

#### [7.9.1](https://github.com/googleapis/nodejs-spanner/compare/v7.9.0...v7.9.1) (2024-06-26)

##### Bug Fixes

- Retry with timeout ( [\#2071](https://github.com/googleapis/nodejs-spanner/issues/2071) ) ( [a943257](https://github.com/googleapis/nodejs-spanner/commit/a943257a0402b26fd80196057a9724fd28fc5c1b) )

#### [7.10.0](https://github.com/googleapis/nodejs-spanner/compare/v7.9.1...v7.10.0) (2024-07-19)

##### Features

- Add field lock_hint in spanner.proto ( [47520e9](https://github.com/googleapis/nodejs-spanner/commit/47520e927b0fdcc60cb67378b8b49f44329f210b) )
- Add field order_by in spanner.proto ( [47520e9](https://github.com/googleapis/nodejs-spanner/commit/47520e927b0fdcc60cb67378b8b49f44329f210b) )
- Add QueryCancellationAction message in executor protos ( [47520e9](https://github.com/googleapis/nodejs-spanner/commit/47520e927b0fdcc60cb67378b8b49f44329f210b) )
- Add support for change streams transaction exclusion option for Batch Write ( [\#2070](https://github.com/googleapis/nodejs-spanner/issues/2070) ) ( [2a9e443](https://github.com/googleapis/nodejs-spanner/commit/2a9e44328acda310db2d0d65d32ad82d77a9fcb0) )
- Update Nodejs generator to send API versions in headers for GAPICs ( [47520e9](https://github.com/googleapis/nodejs-spanner/commit/47520e927b0fdcc60cb67378b8b49f44329f210b) )

##### Bug Fixes

- Callback in getDatabaseDialect ( [\#2078](https://github.com/googleapis/nodejs-spanner/issues/2078) ) ( [7e4a8e9](https://github.com/googleapis/nodejs-spanner/commit/7e4a8e9ad4f785b15b68aaa06b6480098d7995ba) )
- **deps:** Update dependency google-gax to v4.3.8 ( [\#2077](https://github.com/googleapis/nodejs-spanner/issues/2077) ) ( [e927880](https://github.com/googleapis/nodejs-spanner/commit/e927880ff786a2528a2bbb063a244af3c42ff69c) )

## July 18, 2024

Feature

Spanner includes the `JSON_ARRAY()` and `JSON_OBJECT()` functions for building JSON types in GoogleSQL. For more information, see [JSON functions in GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions) .

## July 16, 2024

Feature

Spanner supports the following GoogleSQL JSON functions:

- [`BOOL_ARRAY`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#bool_array_for_json) : Converts a JSON array of booleans to a SQL `ARRAY<BOOL>` value.
- [`FLOAT32`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#float_for_json) : Converts a JSON number to a SQL `FLOAT32` value.
- [`FLOAT32_ARRAY`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#float_array_for_json) : Converts a JSON array of numbers to a SQL `ARRAY<FLOAT32>` value.
- [`FLOAT64_ARRAY`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#double_array_for_json) : Converts a JSON array of numbers to a SQL `ARRAY<FLOAT64>` value.
- [`INT64_ARRAY`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#int64_array_for_json) : Converts a JSON array of numbers to a SQL `INT64_ARRAY` value.
- [`STRING_ARRAY`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#string_array_for_json) : Converts a JSON array of strings to a SQL `ARRAY<STRING>` value.

Feature

Spanner supports the following PostgreSQL JSONB functions:

- `jsonb_array_elements()`
- `spanner.bool_array()`
- `spanner.float32_array()`
- `spanner.float64_array()`
- `spanner.int64_array()`
- `spanner.string_array()`

For more information, see [JSONB functions](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#jsonb_functions) and [Spanner specific JSONB functions](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#spanner-jsonb-functions) .

Feature

Spanner supports the GoogleSQL `PDML_MAX_PARALLELISM` statement-level hint. For more information, see [Statement hints](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#statement_hints) .

Feature

The following are supported for the `INSERT` statement:

- [`INSERT OR UPDATE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-or-update) and [`INSERT OR IGNORE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-ignore) DML statements support the [`THEN RETURN`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-and-then-return) clause in GoogleSQL.
- [`INSERT…ON CONFLICT`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/dml-syntax#on_conflict_clause) DML statement supports the [`RETURNING`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/dml-syntax#insert-returning) clause in PostgreSQL.
- [`THEN RETURN`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-and-then-return) supports the `WITH ACTION` clause in GoogleSQL.

Feature

Spanner supports geo-partitioning (in Preview). You can use geo-partitioning to segment and store rows in your database table across different configurations. For more information, see the [Geo-partitioning overview](https://docs.cloud.google.com/spanner/docs/geo-partitioning) .

## July 12, 2024

Feature

Spanner supports dual-region instance configurations in Australia, Germany, India, and Japan. Dual-region configurations let you replicate data in multiple zones across two regions in a single country. This helps you meet your data residency requirements, while taking advantage of 99.999% availability. For more information, see [Dual-region configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations#dual-region-configurations) .

Feature

Spanner supports the approximate nearest neighbor (ANN) distance functions ( `APPROX_COSINE_DISTANCE()` , `APPROX_EUCLIDEAN_DISTANCE()` , and `APPROX_DOT_PRODUCT()` ) in the GoogleSQL dialect (in Preview). For tables with a lot of vector data, you can create a vector index using DDL statements. This accelerates similarity searches and nearest neighbor queries using these functions in standard SQL, without copying the data to a separate system. For more information, see [Find approximate nearest neighbors to index and query vector embeddings in Spanner](https://docs.cloud.google.com/spanner/docs/find-approximate-nearest-neighbors) .

## July 11, 2024

Feature

You can use [`EXPORT DATA`](https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/other-statements) statements to [reverse ETL BigQuery data to Spanner](https://docs.cloud.google.com/bigquery/docs/export-to-spanner) . This feature is in [Preview](https://cloud.google.com/products/#product-launch-stages) .

## July 03, 2024

Feature

Multiplexed sessions are generally available. Multiplexed session is a new session management model which simplifies the pool management in clients. For more information, see [Multiplexed sessions](https://docs.cloud.google.com/spanner/docs/sessions#multiplexed_sessions) .

Feature

Spanner allows privileged users to cancel long-running queries. For more information, see [GoogleSQL Query cancellation](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/stored-procedures#query-cancellation) or [PostgreSQL Query cancellation](https://docs.cloud.google.com/spanner/docs/reference/postgresql/stored-procedures-pg#query-cancellation) .

## June 28, 2024

Libraries

A monthly digest of client library updates from across the [Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.68.0](https://github.com/googleapis/java-spanner/compare/v6.67.0...v6.68.0) (2024-05-27)

##### Features

- Allow passing libraries_bom_version from env ( [\#1967](https://github.com/googleapis/java-spanner/issues/1967) ) ( [\#3112](https://github.com/googleapis/java-spanner/issues/3112) ) ( [7d5a52c](https://github.com/googleapis/java-spanner/commit/7d5a52c19a4b8028b78fc64a10f1ba6127fa6ffe) )
- Allow DML batches in transactions to execute analyzeUpdate ( [\#3114](https://github.com/googleapis/java-spanner/issues/3114) ) ( [dee7cda](https://github.com/googleapis/java-spanner/commit/dee7cdabe74058434e4d630846f066dc82fdf512) )
- **spanner:** Add support for Proto Columns in Connection API ( [\#3123](https://github.com/googleapis/java-spanner/issues/3123) ) ( [7e7c814](https://github.com/googleapis/java-spanner/commit/7e7c814045dc84aaa57e7c716b0221e6cb19bcd1) )

##### Bug Fixes

- Allow getMetadata() calls before calling next() ( [\#3111](https://github.com/googleapis/java-spanner/issues/3111) ) ( [39902c3](https://github.com/googleapis/java-spanner/commit/39902c384f3f7f9438252cbee287f2428faf1440) )

##### Dependencies

- Update dependency org.graalvm.buildtools:native-maven-plugin to v0.10.2 ( [\#3117](https://github.com/googleapis/java-spanner/issues/3117) ) ( [ddebbbb](https://github.com/googleapis/java-spanner/commit/ddebbbbeef976f61f23cdd66c5f7c1f412e2f9bd) )

#### [6.69.0](https://github.com/googleapis/java-spanner/compare/v6.68.1...v6.69.0) (2024-06-12)

##### Features

- Add option to enable ApiTracer ( [\#3095](https://github.com/googleapis/java-spanner/issues/3095) ) ( [a0a4bc5](https://github.com/googleapis/java-spanner/commit/a0a4bc58d4269a8c1e5e76d9a0469f649bb69148) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.31.0 ( [\#3159](https://github.com/googleapis/java-spanner/issues/3159) ) ( [1ee19d1](https://github.com/googleapis/java-spanner/commit/1ee19d19c2db30d79c8741cc5739de1c69fb95f9) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.47.0](https://github.com/googleapis/python-spanner/compare/v3.46.0...v3.47.0) (2024-05-22)

##### Features

- **spanner:** Add support for Proto Columns ( [\#1084](https://github.com/googleapis/python-spanner/issues/1084) ) ( [3ca2689](https://github.com/googleapis/python-spanner/commit/3ca2689324406e0bd9a6b872eda4a23999115f0f) )

## June 20, 2024

Feature

Named schemas is generally available. With named schemas, you can group database objects in a namespace to avoid naming conflicts and collectively manage their fine-grained access control permissions, see [Named schemas](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#named-schemas) .

## June 17, 2024

Feature

Generated columns don't require the `STORED` attribute. Without this attribute, the generated column is evaluated at query or index time and doesn't require additional storage or write overhead. For more information, see [Create and manage generated columns](https://docs.cloud.google.com/spanner/docs/generated-column/how-to) .

## June 03, 2024

Feature

[Query optimizer version 7](https://docs.cloud.google.com/spanner/docs/query-optimizer/versions#version-7) is generally available. Version 6 remains the default optimizer version.

## May 31, 2024

Feature

Spanner supports the protocol buffer data type in GoogleSQL. For more information, see [Work with protocol buffers in GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/protocol-buffers) .

Libraries

A monthly digest of client library updates from across the [Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.61.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.60.0...spanner/v1.61.0) (2024-04-30)

##### Features

- **spanner/admin/instance:** Adding `EXPECTED_FULFILLMENT_PERIOD` to the indicate instance creation times (with `FULFILLMENT_PERIOD_NORMAL` or `FULFILLMENT_PERIOD_EXTENDED` ENUM) with the extended instance creation time triggered by On-Demand Capacity... ( [\#9693](https://github.com/googleapis/google-cloud-go/issues/9693) ) ( [aa93790](https://github.com/googleapis/google-cloud-go/commit/aa93790132ba830b4c97d217ef02764e2fb1b8ea) )
- **spanner/executor:** Add SessionPoolOptions, SpannerOptions protos in executor protos ( [2cdc40a](https://github.com/googleapis/google-cloud-go/commit/2cdc40a0b4288f5ab5f2b2b8f5c1d6453a9c81ec) )
- **spanner:** Add support for change streams transaction exclusion option ( [\#9779](https://github.com/googleapis/google-cloud-go/issues/9779) ) ( [979ce94](https://github.com/googleapis/google-cloud-go/commit/979ce94758442b1224a78a4f3b1f5d592ab51660) )
- **spanner:** Support MultiEndpoint ( [\#9565](https://github.com/googleapis/google-cloud-go/issues/9565) ) ( [0ac0d26](https://github.com/googleapis/google-cloud-go/commit/0ac0d265abedf946b05294ef874a892b2c5d6067) )

##### Bug Fixes

- **spanner/test/opentelemetry/test:** Bump x/net to v0.24.0 ( [ba31ed5](https://github.com/googleapis/google-cloud-go/commit/ba31ed5fda2c9664f2e1cf972469295e63deb5b4) )
- **spanner:** Bump x/net to v0.24.0 ( [ba31ed5](https://github.com/googleapis/google-cloud-go/commit/ba31ed5fda2c9664f2e1cf972469295e63deb5b4) )
- **spanner:** Fix uint8 conversion ( [9221c7f](https://github.com/googleapis/google-cloud-go/commit/9221c7fa12cef9d5fb7ddc92f41f1d6204971c7b) )

#### [1.62.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.61.0...spanner/v1.62.0) (2024-05-15)

##### Features

- **spanner/admin/database:** Add support for multi region encryption config ( [3e25053](https://github.com/googleapis/google-cloud-go/commit/3e250530567ee81ed4f51a3856c5940dbec35289) )
- **spanner/executor:** Add QueryCancellationAction message in executor protos ( [292e812](https://github.com/googleapis/google-cloud-go/commit/292e81231b957ae7ac243b47b8926564cee35920) )
- **spanner:** Add `RESOURCE_EXHAUSTED` to the list of retryable error codes ( [1d757c6](https://github.com/googleapis/google-cloud-go/commit/1d757c66478963d6cbbef13fee939632c742759c) )
- **spanner:** Add support for Proto Columns ( [\#9315](https://github.com/googleapis/google-cloud-go/issues/9315) ) ( [3ffbbbe](https://github.com/googleapis/google-cloud-go/commit/3ffbbbe50225684f4211c6dbe3ca25acb3d02b8e) )

##### Bug Fixes

- **spanner:** Add ARRAY keywords to keywords ( [\#10079](https://github.com/googleapis/google-cloud-go/issues/10079) ) ( [8e675cd](https://github.com/googleapis/google-cloud-go/commit/8e675cd0ccf12c6912209aa5c56092db3716c40d) )
- **spanner:** Handle unused errors ( [\#10067](https://github.com/googleapis/google-cloud-go/issues/10067) ) ( [a0c097c](https://github.com/googleapis/google-cloud-go/commit/a0c097c724b609cfa428e69f89075f02a3782a7b) )
- **spanner:** Remove json-iterator dependency ( [\#10099](https://github.com/googleapis/google-cloud-go/issues/10099) ) ( [3917cca](https://github.com/googleapis/google-cloud-go/commit/3917ccac57c403b3b4d07514ac10a66a86e298c0) ), refs [\#9380](https://github.com/googleapis/google-cloud-go/issues/9380)
- **spanner:** Update staleness bound ( [\#10118](https://github.com/googleapis/google-cloud-go/issues/10118) ) ( [c07f1e4](https://github.com/googleapis/google-cloud-go/commit/c07f1e47c06387b696abb1edbfa339b391ec1fd5) )

#### [1.63.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.62.0...spanner/v1.63.0) (2024-05-24)

##### Features

- **spanner:** Fix schema naming ( [\#10194](https://github.com/googleapis/google-cloud-go/issues/10194) ) ( [215e0c8](https://github.com/googleapis/google-cloud-go/commit/215e0c8125ea05246c834984bde1ca698c7dde4c) )
- **spanner:** Update go mod to use latest grpc lib ( [\#10218](https://github.com/googleapis/google-cloud-go/issues/10218) ) ( [adf91f9](https://github.com/googleapis/google-cloud-go/commit/adf91f9fd37faa39ec7c6f9200273220f65d2a82) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.65.1](https://github.com/googleapis/java-spanner/compare/v6.65.0...v6.65.1) (2024-04-30)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.43.0 ( [\#3066](https://github.com/googleapis/java-spanner/issues/3066) ) ( [97b0a93](https://github.com/googleapis/java-spanner/commit/97b0a93469ea1b0f0c9a3413e2364951c2d667d1) )

##### Documentation

- Add a sample for max commit delays ( [\#2941](https://github.com/googleapis/java-spanner/issues/2941) ) ( [d3b5097](https://github.com/googleapis/java-spanner/commit/d3b50976f8a6687a6dac2f483ae133c026b81cac) )

#### [6.66.0](https://github.com/googleapis/java-spanner/compare/v6.65.1...v6.66.0) (2024-05-03)

##### Features

- Allow DDL with autocommit=false ( [\#3057](https://github.com/googleapis/java-spanner/issues/3057) ) ( [22833ac](https://github.com/googleapis/java-spanner/commit/22833acf9f073271ce0ee10f2b496f3a1d39566a) )
- Include stack trace of checked out sessions in exception ( [\#3092](https://github.com/googleapis/java-spanner/issues/3092) ) ( [ba6a0f6](https://github.com/googleapis/java-spanner/commit/ba6a0f644b6caa4d2f3aa130c6061341b70957dd) )

##### Bug Fixes

- Multiplexed session metrics were not included in refactor move ( [\#3088](https://github.com/googleapis/java-spanner/issues/3088) ) ( [f3589c4](https://github.com/googleapis/java-spanner/commit/f3589c430b0e84933a91008bb306c26089788357) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.30.0 ( [\#3082](https://github.com/googleapis/java-spanner/issues/3082) ) ( [ddfc98e](https://github.com/googleapis/java-spanner/commit/ddfc98e240fb47ef51075ba4461bf9a98aa25ce0) )

#### [6.67.0](https://github.com/googleapis/java-spanner/compare/v6.66.0...v6.67.0) (2024-05-22)

##### Features

- Add tracing for batchUpdate, executeUpdate, and connections ( [\#3097](https://github.com/googleapis/java-spanner/issues/3097) ) ( [45cdcfc](https://github.com/googleapis/java-spanner/commit/45cdcfcde02aa7976b017a90f81c2ccd28658c8f) )

##### Performance Improvements

- Minor optimizations to the standard query path ( [\#3101](https://github.com/googleapis/java-spanner/issues/3101) ) ( [ec820a1](https://github.com/googleapis/java-spanner/commit/ec820a16e2b3cb1a12a15231491b75cd73afaa13) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.44.0 ( [\#3099](https://github.com/googleapis/java-spanner/issues/3099) ) ( [da44e93](https://github.com/googleapis/java-spanner/commit/da44e932a39ac0124b63914f8ea926998c10ea2e) )
- Update dependency com.google.cloud:sdk-platform-java-config to v3.30.1 ( [\#3116](https://github.com/googleapis/java-spanner/issues/3116) ) ( [d205a73](https://github.com/googleapis/java-spanner/commit/d205a73714786a609673012b771e7a0722b3e1f2) ) ( [d205a73](https://github.com/googleapis/java-spanner/commit/d205a73714786a609673012b771e7a0722b3e1f2) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.8.0](https://github.com/googleapis/nodejs-spanner/compare/v7.7.0...v7.8.0) (2024-05-24)

##### Features

- Add `RESOURCE_EXHAUSTED` to the list of retryable error codes ( [\#2032](https://github.com/googleapis/nodejs-spanner/issues/2032) ) ( [a4623c5](https://github.com/googleapis/nodejs-spanner/commit/a4623c560c16fa1f37a06cb57a5e47a1d6759d27) )
- Add support for multi region encryption config ( [81fa610](https://github.com/googleapis/nodejs-spanner/commit/81fa610895fe709cbb7429896493a67407a6343c) )
- Add support for Proto columns ( [\#1991](https://github.com/googleapis/nodejs-spanner/issues/1991) ) ( [ae59c7f](https://github.com/googleapis/nodejs-spanner/commit/ae59c7f957660e08cd5965b5e67694fa1ccc0057) )
- **spanner:** Add support for change streams transaction exclusion option ( [\#2049](https://github.com/googleapis/nodejs-spanner/issues/2049) ) ( [d95cab5](https://github.com/googleapis/nodejs-spanner/commit/d95cab5abe50cdb56cbc1d6d935aee29526e1096) )

##### Bug Fixes

- **deps:** Update dependency google-gax to v4.3.3 ( [\#2038](https://github.com/googleapis/nodejs-spanner/issues/2038) ) ( [d86c1b0](https://github.com/googleapis/nodejs-spanner/commit/d86c1b0c21c7c95e3110221b3ca6ff9ff3b4a088) )
- Drop table statement ( [\#2036](https://github.com/googleapis/nodejs-spanner/issues/2036) ) ( [f31d7b2](https://github.com/googleapis/nodejs-spanner/commit/f31d7b205d74d4a783f0d5159dd5b62efe968fe6) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.46.0](https://github.com/googleapis/python-spanner/compare/v3.45.0...v3.46.0) (2024-05-02)

##### Features

- **spanner:** Adding EXPECTED_FULFILLMENT_PERIOD to the indicate instance creation times (with FULFILLMENT_PERIOD_NORMAL or FULFILLMENT_PERIOD_EXTENDED ENUM) with the extended instance creation time triggered by On-Demand Capacity Feature ( [293ecda](https://github.com/googleapis/python-spanner/commit/293ecdad78b51f248f8d5c023bdba3bac998ea5c) )

##### Documentation

- Remove duplicate paramter description ( [\#1052](https://github.com/googleapis/python-spanner/issues/1052) ) ( [1164743](https://github.com/googleapis/python-spanner/commit/116474318d42a6f1ea0f9c2f82707e5dde281159) )

## May 29, 2024

Feature

Spanner supports the following columns in the `SPANNER_SYS` [query statistics](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics) table:

- `AVG_MEMORY_PEAK_USAGE_BYTES`
- `AVG_MEMORY_USAGE_PERCENTAGE`
- `AVG_QUERY_PLAN_CREATION_TIME_SECS`
- `AVG_FILESYSTEM_DELAY_SECS`
- `AVG_REMOTE_SERVER_CALLS`
- `AVG_ROWS_SPOOLED`

## May 13, 2024

Feature

Vector length annotation is generally available. For more information, see the [PostgreSQL vector length parameter](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-types#array-extensions) or the [GoogleSQL `vector_length` parameter](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#arrays) .

Feature

Spanner supports the `read_request_latencies_by_change_stream` metric in Cloud Monitoring. Use this metric to view all read latencies and filter latencies by change stream or non-change stream reads. For more information, see [Available charts and metrics](https://docs.cloud.google.com/spanner/docs/monitoring-console#charts-metrics) .

## April 30, 2024

Feature

Spanner supports the following for PostgreSQL arrays:

- [`UNNEST` operator](https://docs.cloud.google.com/spanner/docs/reference/postgresql/query-syntax#unnest_operator)
- [Array slices](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-types#array_slices)
- [`ANY` , `SOME` , and `ALL` array comparison operators](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#array_comparisons)
- [`arrayoverlap` , `arraycontains` , and `arraycontained` functions and their operators](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#array_operators)

Feature

Through self-service and with zero downtime, you can add and remove read-only replicas in base instance configurations and move your Spanner instance to a different instance configuration. For more information, see [Move an instance](https://docs.cloud.google.com/spanner/docs/move-instance) .

Libraries

A monthly digest of client library updates from across the [Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.62.1](https://github.com/googleapis/java-spanner/compare/v6.62.0...v6.62.1) (2024-03-28)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.39.0 ( [\#2966](https://github.com/googleapis/java-spanner/issues/2966) ) ( [a5cb1dd](https://github.com/googleapis/java-spanner/commit/a5cb1ddd065100497d9215eff30d57361d7e84de) )
- Update dependency com.google.cloud:google-cloud-trace to v2.38.0 ( [\#2967](https://github.com/googleapis/java-spanner/issues/2967) ) ( [b2dc788](https://github.com/googleapis/java-spanner/commit/b2dc788d5a54244d83a192ecac894ff931f884c4) )

#### [6.63.0](https://github.com/googleapis/java-spanner/compare/v6.62.1...v6.63.0) (2024-03-30)

##### Features

- Add support for transaction-level exclusion from change streams ( [\#2959](https://github.com/googleapis/java-spanner/issues/2959) ) ( [7ae376a](https://github.com/googleapis/java-spanner/commit/7ae376acea4dce7a0bb4565d6c9bfdbbb75146c6) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.40.0 ( [\#2987](https://github.com/googleapis/java-spanner/issues/2987) ) ( [0a1ffcb](https://github.com/googleapis/java-spanner/commit/0a1ffcb371bdee6e478e3aa53b0a4591055134e3) )
- Update dependency com.google.cloud:google-cloud-trace to v2.39.0 ( [\#2988](https://github.com/googleapis/java-spanner/issues/2988) ) ( [cf11641](https://github.com/googleapis/java-spanner/commit/cf116412d46c5047167d4dd60ef9c88c3d9c754b) )
- Update dependency commons-io:commons-io to v2.16.0 ( [\#2986](https://github.com/googleapis/java-spanner/issues/2986) ) ( [4697261](https://github.com/googleapis/java-spanner/commit/46972619f88018bad1b4e05526a618d38e2e0897) )

#### [6.64.0](https://github.com/googleapis/java-spanner/compare/v6.63.0...v6.64.0) (2024-04-12)

##### Features

- Add endpoint connection URL property ( [\#2969](https://github.com/googleapis/java-spanner/issues/2969) ) ( [c9be29c](https://github.com/googleapis/java-spanner/commit/c9be29c717924d7f4c5acd8fe09ee371d0101642) )
- Add PG OID support ( [\#2736](https://github.com/googleapis/java-spanner/issues/2736) ) ( [ba2a4af](https://github.com/googleapis/java-spanner/commit/ba2a4afa5c1d64c932e9687d52b15c28d9dd7d91) )
- Add SessionPoolOptions, SpannerOptions protos in executor protos ( [\#2932](https://github.com/googleapis/java-spanner/issues/2932) ) ( [1673fd7](https://github.com/googleapis/java-spanner/commit/1673fd70df4ebfaa4b5fa07112d152119427699a) )
- Support max_commit_delay in Connection API ( [\#2954](https://github.com/googleapis/java-spanner/issues/2954) ) ( [a8f1852](https://github.com/googleapis/java-spanner/commit/a8f185261c812e7d6c92cb61ecc1f9c78ba3c4d9) )

##### Bug Fixes

- Executor framework changes skipped in clirr checks, and added exception for partition methods in admin class ( [\#3000](https://github.com/googleapis/java-spanner/issues/3000) ) ( [c2d8e95](https://github.com/googleapis/java-spanner/commit/c2d8e955abddb0117f1b3b94c2d9650d2cf4fdfd) )

##### Dependencies

- Update actions/checkout action to v4 ( [\#3006](https://github.com/googleapis/java-spanner/issues/3006) ) ( [368a9f3](https://github.com/googleapis/java-spanner/commit/368a9f33758961d8e3fd387ec94d380e7c6460cc) )
- Update actions/github-script action to v7 ( [\#3007](https://github.com/googleapis/java-spanner/issues/3007) ) ( [b0cfea6](https://github.com/googleapis/java-spanner/commit/b0cfea6e73b7293f564357e8d1c8c6bb2e0cf855) )
- Update actions/setup-java action to v4 ( [\#3008](https://github.com/googleapis/java-spanner/issues/3008) ) ( [d337080](https://github.com/googleapis/java-spanner/commit/d337080089dbd58cb4bf94f2cb5925f627435d39) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.42.0 ( [\#2997](https://github.com/googleapis/java-spanner/issues/2997) ) ( [0615beb](https://github.com/googleapis/java-spanner/commit/0615beb806ef62dbbfcc6bbffd082adc9c62372c) )
- Update dependency com.google.cloud:google-cloud-trace to v2.41.0 ( [\#2998](https://github.com/googleapis/java-spanner/issues/2998) ) ( [f50cd04](https://github.com/googleapis/java-spanner/commit/f50cd04660f480c62ddbd6c8a9e892cd95ec16b0) )
- Update dependency commons-io:commons-io to v2.16.1 ( [\#3020](https://github.com/googleapis/java-spanner/issues/3020) ) ( [aafd5b9](https://github.com/googleapis/java-spanner/commit/aafd5b9514c14a0dbfd0bf2616990f3c347ac0c6) )
- Update opentelemetry.version to v1.37.0 ( [\#3021](https://github.com/googleapis/java-spanner/issues/3021) ) ( [8f1ed2a](https://github.com/googleapis/java-spanner/commit/8f1ed2ac20896fb413749bb18652764096f1fb2d) )
- Update stcarolas/setup-maven action to v5 ( [\#3009](https://github.com/googleapis/java-spanner/issues/3009) ) ( [541acd2](https://github.com/googleapis/java-spanner/commit/541acd23aaf2c9336615406e30618fb65606e6c5) )

#### [6.65.0](https://github.com/googleapis/java-spanner/compare/v6.64.0...v6.65.0) (2024-04-20)

##### Features

- Remove grpclb ( [\#2760](https://github.com/googleapis/java-spanner/issues/2760) ) ( [1df09d9](https://github.com/googleapis/java-spanner/commit/1df09d9b9189c5527de91189a063ecc15779ac77) )
- Support client-side hints for tags and priority ( [\#3005](https://github.com/googleapis/java-spanner/issues/3005) ) ( [48828df](https://github.com/googleapis/java-spanner/commit/48828df3489465bb53a18be50808fbd435f3e896) ), closes [\#2978](https://github.com/googleapis/java-spanner/issues/2978)

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.39.0 ( [\#3001](https://github.com/googleapis/java-spanner/issues/3001) ) ( [6cec1bf](https://github.com/googleapis/java-spanner/commit/6cec1bf1bb44a52c62c2310447c6a068a88209ea) )
- NullPointerException on AbstractReadContext.span ( [\#3036](https://github.com/googleapis/java-spanner/issues/3036) ) ( [55732fd](https://github.com/googleapis/java-spanner/commit/55732fd107ac1d3b8c16eee198c904d54d98b2b4) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.29.0 ( [\#3045](https://github.com/googleapis/java-spanner/issues/3045) ) ( [67a6534](https://github.com/googleapis/java-spanner/commit/67a65346d5a01d118d5220230e3bed6db7e79a33) )
- Update dependency commons-cli:commons-cli to v1.7.0 ( [\#3043](https://github.com/googleapis/java-spanner/issues/3043) ) ( [9fea7a3](https://github.com/googleapis/java-spanner/commit/9fea7a30e90227e735ad3595f4ca58dfb1ca1b93) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.6.0](https://github.com/googleapis/nodejs-spanner/compare/v7.5.0...v7.6.0) (2024-03-26)

##### Features

- Add instance partition support to spanner instance proto ( [\#2001](https://github.com/googleapis/nodejs-spanner/issues/2001) ) ( [4381047](https://github.com/googleapis/nodejs-spanner/commit/43810478e81d3a234e7fa94af90fd49ca379dd98) )
- Managed Autoscaler ( [\#2015](https://github.com/googleapis/nodejs-spanner/issues/2015) ) ( [547ca1b](https://github.com/googleapis/nodejs-spanner/commit/547ca1b0da8c5c5e28f85fbd4ea16af21e20c980) )
- **spanner:** Add a sample for max commit delays ( [\#1993](https://github.com/googleapis/nodejs-spanner/issues/1993) ) ( [91c7204](https://github.com/googleapis/nodejs-spanner/commit/91c7204e2c8f62e229d7a2b2a0ff059d421dd984) )
- **spanner:** Add support for float32 ( [\#2020](https://github.com/googleapis/nodejs-spanner/issues/2020) ) ( [99e2c1d](https://github.com/googleapis/nodejs-spanner/commit/99e2c1d4791a5ca86fdccb3f600aa4592efe0a45) )

#### [7.7.0](https://github.com/googleapis/nodejs-spanner/compare/v7.6.0...v7.7.0) (2024-04-17)

##### Features

- OptimisticLock option for getTransaction method ( [\#2028](https://github.com/googleapis/nodejs-spanner/issues/2028) ) ( [dacf869](https://github.com/googleapis/nodejs-spanner/commit/dacf8697b20752041684710982035b4c97837d28) )
- **spanner:** Adding `EXPECTED_FULFILLMENT_PERIOD` to the indicate instance creation times (with `FULFILLMENT_PERIOD_NORMAL` or `FULFILLMENT_PERIOD_EXTENDED` ENUM) with the extended instance creation time triggered by On-Demand Capacity Feature ( [\#2024](https://github.com/googleapis/nodejs-spanner/issues/2024) ) ( [5292e03](https://github.com/googleapis/nodejs-spanner/commit/5292e035c5278ba6806f9e1eb84809ed893b1e37) )

##### Bug Fixes

- **deps:** Update dependency google-gax to v4.3.2 ( [\#2026](https://github.com/googleapis/nodejs-spanner/issues/2026) ) ( [0ee9831](https://github.com/googleapis/nodejs-spanner/commit/0ee98319f291f552a0afc52629d12af9969d1d10) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.45.0](https://github.com/googleapis/python-spanner/compare/v3.44.0...v3.45.0) (2024-04-17)

##### Features

- Add support for PG.OID in parameterized queries ( [\#1035](https://github.com/googleapis/python-spanner/issues/1035) ) ( [ea5efe4](https://github.com/googleapis/python-spanner/commit/ea5efe4d0bc2790b5172e43e1b66fa3997190adf) )

##### Bug Fixes

- Dates before 1000AD should use 4-digit years ( [\#1132](https://github.com/googleapis/python-spanner/issues/1132) ) ( [0ef6565](https://github.com/googleapis/python-spanner/commit/0ef65657de631d876636d11756237496b7713e22) ), closes [\#1131](https://github.com/googleapis/python-spanner/issues/1131)

## April 09, 2024

Feature

The following [Gemini in Databases](https://docs.cloud.google.com/gemini/docs/databases/overview) features are available in [Public Preview](https://cloud.google.com/products#product-launch-stages) :

- [Spanner Studio (GA)](https://docs.cloud.google.com/spanner/docs/manage-data-using-console) : lets users interact with the SQL database and run SQL queries from the Google Cloud console to access and manipulate data.
- Spanner supports the use of Gemini models with GoogleSQL and PostgreSQL machine learning prediction functions.

To learn how to enable and activate Gemini in Databases, see [Set up Gemini in Databases](https://docs.cloud.google.com/gemini/docs/databases/set-up-gemini) .

Feature

Spanner supports using LangChain with the vector store, document loader, and chat message history objects. For more information, see [Build LLM-powered applications using LangChain](https://docs.cloud.google.com/spanner/docs/langchain) .

Feature

Spanner supports the `dot_product()` function (in [Preview](https://cloud.google.com/products#product-launch-stages) ). For more information, see [Choose among vector distance functions to measure vector embeddings similarity](https://docs.cloud.google.com/spanner/docs/choose-vector-distance-function) .

Feature

Spanner GoogleSQL supports [`SAFE.ML.PREDICT()`](https://docs.cloud.google.com/spanner/docs/ml#ml-functions) , which lets you to return a `null` instead of an error in your predictions.

Feature

Spanner supports the use of [Gemini](https://docs.cloud.google.com/gemini/docs/databases/overview) models with GoogleSQL and PostgreSQL [machine learning prediction functions](https://docs.cloud.google.com/spanner/docs/ml#ml-functions) (in [Preview](https://cloud.google.com/products#product-launch-stages) ).

Feature

Spanner supports the `float32` ( [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-types#floating_point_types) ) and `float4/real` ( [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-types#supported) ) data type (in [Preview](https://cloud.google.com/products#product-launch-stages) ).

Feature

Spanner has extended the array data type with the `VECTOR LENGTH` parameter (in [Preview](https://cloud.google.com/products#product-launch-stages) ). This optional parameter sets an array to a fixed size for use in a vector search. For more information, see the [PostgreSQL `array` data type](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-types#array) or the [GoogleSQL `array` data type](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#arrays) .

Feature

You can [generate ML predictions using the Spanner emulator](https://docs.cloud.google.com/spanner/docs/ml-emulator) with GoogleSQL and PostgreSQL.

Feature

Spanner supports the `ML_PREDICT_ROW()` function for PostgreSQL. You can use this function to generate predictions using SQL. To learn more about this function and how to use it, see [Using Spanner Vertex AI integration functions](https://docs.cloud.google.com/spanner/docs/ml#ml-functions) .

Feature

You can generate and backfill vector embeddings for textual data ( `STRING` or `JSON` ) stored in Spanner using GoogleSQL partitioned DML and the Vertex AI `textembedding-gecko` model. For more information, see [Generate vector embeddings for textual data in bulk using partitioned DML](https://docs.cloud.google.com/spanner/docs/backfill-embeddings) .

Feature

Spanner supports the following PostgreSQL `JSONB` functions:

- `spanner.jsonb_query_array()`
- `jsonb_build_array()`
- `jsonb_build_object()`

The PostgreSQL `CONCAT()` function also supports more than 4 arguments.

For more information, see [Supported PostgreSQL functions](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#jsonb) .

## April 08, 2024

Feature

You can add a [time to live (TTL)-based deletes filter](https://docs.cloud.google.com/spanner/docs/change-streams#ttl-filter) to your Spanner [change streams](https://docs.cloud.google.com/spanner/docs/change-streams) using the `exclude_ttl_deletes` option.

Feature

Spanner [change streams](https://docs.cloud.google.com/spanner/docs/change-streams) support a [value capture type](https://docs.cloud.google.com/spanner/docs/change-streams#value-capture-type) called `NEW_ROW_AND_OLD_VALUES` . This type captures all new values for both modified and unmodified columns, and old values for modified columns.

Feature

You can add a [table modification type filter](https://docs.cloud.google.com/spanner/docs/change-streams#mod-type-filter) to your Spanner [change streams](https://docs.cloud.google.com/spanner/docs/change-streams) to exclude `INSERT` , `UPDATE` , or `DELETE` table modifications.

## March 29, 2024

Libraries

A monthly digest of client library updates from across the [Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.58.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.57.0...spanner/v1.58.0) (2024-03-06)

##### Features

- **spanner/admin/instance:** Add instance partition support to spanner instance proto ( [ae1f547](https://github.com/googleapis/google-cloud-go/commit/ae1f5472bff1b476c3fd58e590ec135185446daf) )
- **spanner:** Add field for multiplexed session in spanner.proto ( [a86aa8e](https://github.com/googleapis/google-cloud-go/commit/a86aa8e962b77d152ee6cdd433ad94967150ef21) )
- **spanner:** SelectAll struct spanner tag annotation match should be case-insensitive ( [\#9460](https://github.com/googleapis/google-cloud-go/issues/9460) ) ( [6cd6a73](https://github.com/googleapis/google-cloud-go/commit/6cd6a73be87a261729d3b6b45f3d28be93c3fdb3) )
- **spanner:** Update TransactionOptions to include new option exclude_txn_from_change_streams ( [0195fe9](https://github.com/googleapis/google-cloud-go/commit/0195fe9292274ff9d86c71079a8e96ed2e5f9331) )

#### [1.59.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.58.0...spanner/v1.59.0) (2024-03-13)

##### Features

- **spanner/spansql:** Support Table rename & Table synonym ( [\#9275](https://github.com/googleapis/google-cloud-go/issues/9275) ) ( [9b97ce7](https://github.com/googleapis/google-cloud-go/commit/9b97ce75d36980fdaa06f15b0398b7b65e0d6082) )
- **spanner:** Add support of float32 type ( [\#9525](https://github.com/googleapis/google-cloud-go/issues/9525) ) ( [87d7ea9](https://github.com/googleapis/google-cloud-go/commit/87d7ea97787a56b18506b53e9b26d037f92759ca) )

##### Bug Fixes

- **spanner:** Add JSON_PARSE_ARRAY to funcNames slice ( [\#9557](https://github.com/googleapis/google-cloud-go/issues/9557) ) ( [f799597](https://github.com/googleapis/google-cloud-go/commit/f79959722352ead48bfb3efb3001fddd3a56db65) )

#### [1.60.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.59.0...spanner/v1.60.0) (2024-03-19)

##### Features

- **spanner:** Allow attempt direct path xds via env var ( [e4b663c](https://github.com/googleapis/google-cloud-go/commit/e4b663cdcb6e010c5a8ac791e5624407aaa191b3) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.61.0](https://github.com/googleapis/java-spanner/compare/v6.60.1...v6.61.0) (2024-03-04)

##### Features

- Support float32 type ( [\#2894](https://github.com/googleapis/java-spanner/issues/2894) ) ( [19b7976](https://github.com/googleapis/java-spanner/commit/19b79764294e938ad85d02b7c0662db6ec3afeda) )

##### Bug Fixes

- Flaky test issue due to AbortedException. ( [\#2925](https://github.com/googleapis/java-spanner/issues/2925) ) ( [cd34c1d](https://github.com/googleapis/java-spanner/commit/cd34c1d3ae9a5a36f4d5516dcf7c3667a9cf015a) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.27.0 ( [\#2935](https://github.com/googleapis/java-spanner/issues/2935) ) ( [f8f835a](https://github.com/googleapis/java-spanner/commit/f8f835a9da705605c492e232a58276c39d1d7e6c) )
- Update dependency org.json:json to v20240303 ( [\#2936](https://github.com/googleapis/java-spanner/issues/2936) ) ( [1d7044e](https://github.com/googleapis/java-spanner/commit/1d7044e97d16f5296b7de020cd24b11cbe2a7df0) )

##### Documentation

- Samples and tests for backup Admin APIs and overall spanner Admin APIs. ( [\#2882](https://github.com/googleapis/java-spanner/issues/2882) ) ( [de13636](https://github.com/googleapis/java-spanner/commit/de1363645e03f46deed5be41f90ddfed72766751) )
- Update all public documents to use auto-generated admin clients. ( [\#2928](https://github.com/googleapis/java-spanner/issues/2928) ) ( [ccb110a](https://github.com/googleapis/java-spanner/commit/ccb110ad6835557870933c95cfd76580fd317a16) )

#### [6.62.0](https://github.com/googleapis/java-spanner/compare/v6.61.0...v6.62.0) (2024-03-19)

##### Features

- Allow attempt direct path xds via env var ( [\#2950](https://github.com/googleapis/java-spanner/issues/2950) ) ( [247a15f](https://github.com/googleapis/java-spanner/commit/247a15f2b8b858143bc906e0619f95a017ffe5c3) )
- Next release from main branch is 6.56.0 ( [\#2929](https://github.com/googleapis/java-spanner/issues/2929) ) ( [66374b1](https://github.com/googleapis/java-spanner/commit/66374b1c4ed88e01ff60fb8e1b7409e5dbbcb811) )

##### Bug Fixes

- Return type of max commit delay option. ( [\#2953](https://github.com/googleapis/java-spanner/issues/2953) ) ( [6e937ab](https://github.com/googleapis/java-spanner/commit/6e937ab16d130e72d633979c1a76bf7b3edbe7b6) )

##### Performance Improvements

- Keep comments when searching for params ( [\#2951](https://github.com/googleapis/java-spanner/issues/2951) ) ( [b782725](https://github.com/googleapis/java-spanner/commit/b782725b92a2662c42ad35647b23009ad95a99a5) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.38.0 ( [\#2942](https://github.com/googleapis/java-spanner/issues/2942) ) ( [ba665bd](https://github.com/googleapis/java-spanner/commit/ba665bd483ba70f09770d92028355ad499003fed) )
- Update dependency com.google.cloud:google-cloud-trace to v2.37.0 ( [\#2944](https://github.com/googleapis/java-spanner/issues/2944) ) ( [b5e608e](https://github.com/googleapis/java-spanner/commit/b5e608ef001473ab5575f1619804b351053c57f2) )
- Update dependency com.google.cloud:sdk-platform-java-config to v3.28.1 ( [\#2952](https://github.com/googleapis/java-spanner/issues/2952) ) ( [1e45237](https://github.com/googleapis/java-spanner/commit/1e45237dd235484a6a279f71ae7e126727382f9c) )
- Update opentelemetry.version to v1.36.0 ( [\#2945](https://github.com/googleapis/java-spanner/issues/2945) ) ( [e70b035](https://github.com/googleapis/java-spanner/commit/e70b0357543d38b6e9265e04444cec494ebd6885) )

##### Documentation

- **samples:** Add tag to statement timeout sample ( [\#2931](https://github.com/googleapis/java-spanner/issues/2931) ) ( [2392afe](https://github.com/googleapis/java-spanner/commit/2392afed0d25266294e0ce11c6ae32d7307e6830) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.5.0](https://github.com/googleapis/nodejs-spanner/compare/v7.4.0...v7.5.0) (2024-03-04)

##### Features

- **spanner:** Add emulator support for the admin client autogenerated API samples ( [\#1994](https://github.com/googleapis/nodejs-spanner/issues/1994) ) ( [e2fe5b7](https://github.com/googleapis/nodejs-spanner/commit/e2fe5b748c3077078fa43e4bfa427fef603656a9) )

##### Bug Fixes

- Revert untyped param type feature ( [\#2012](https://github.com/googleapis/nodejs-spanner/issues/2012) ) ( [49fa60d](https://github.com/googleapis/nodejs-spanner/commit/49fa60dd0735fe66db33f7b9137dba0821eb5184) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.43.0](https://github.com/googleapis/python-spanner/compare/v3.42.0...v3.43.0) (2024-03-06)

##### Features

- Add retry and timeout for batch dml ( [\#1107](https://github.com/googleapis/python-spanner/issues/1107) ) ( [4f6340b](https://github.com/googleapis/python-spanner/commit/4f6340b0930bb1b5430209c4a1ff196c42b834d0) )
- Add support for max commit delay ( [\#1050](https://github.com/googleapis/python-spanner/issues/1050) ) ( [d5acc26](https://github.com/googleapis/python-spanner/commit/d5acc263d86fcbde7d5f972930255119e2f60e76) )
- Exposing Spanner client in dbapi connection ( [\#1100](https://github.com/googleapis/python-spanner/issues/1100) ) ( [9299212](https://github.com/googleapis/python-spanner/commit/9299212fb8aa6ed27ca40367e8d5aaeeba80c675) )
- Include RENAME in DDL regex ( [\#1075](https://github.com/googleapis/python-spanner/issues/1075) ) ( [3669303](https://github.com/googleapis/python-spanner/commit/3669303fb50b4207975b380f356227aceaa1189a) )
- Support partitioned dml in dbapi ( [\#1103](https://github.com/googleapis/python-spanner/issues/1103) ) ( [3aab0ed](https://github.com/googleapis/python-spanner/commit/3aab0ed5ed3cd078835812dae183a333fe1d3a20) )
- Untyped param ( [\#1001](https://github.com/googleapis/python-spanner/issues/1001) ) ( [1750328](https://github.com/googleapis/python-spanner/commit/1750328bbc7f8a1125f8e0c38024ced8e195a1b9) )

##### Documentation

- Samples and tests for admin backup APIs ( [\#1105](https://github.com/googleapis/python-spanner/issues/1105) ) ( [5410c32](https://github.com/googleapis/python-spanner/commit/5410c32febbef48d4623d8023a6eb9f07a65c2f5) )
- Samples and tests for admin database APIs ( [\#1099](https://github.com/googleapis/python-spanner/issues/1099) ) ( [c25376c](https://github.com/googleapis/python-spanner/commit/c25376c8513af293c9db752ffc1970dbfca1c5b8) )
- Update all public documents to use auto-generated admin clients. ( [\#1109](https://github.com/googleapis/python-spanner/issues/1109) ) ( [d683a14](https://github.com/googleapis/python-spanner/commit/d683a14ccc574e49cefd4e2b2f8b6d9bfd3663ec) )
- Use autogenerated methods to get names from admin samples ( [\#1110](https://github.com/googleapis/python-spanner/issues/1110) ) ( [3ab74b2](https://github.com/googleapis/python-spanner/commit/3ab74b267b651b430e96712be22088e2859d7e79) )

#### [3.44.0](https://github.com/googleapis/python-spanner/compare/v3.43.0...v3.44.0) (2024-03-13)

##### Features

- Add support of float32 type ( [\#1113](https://github.com/googleapis/python-spanner/issues/1113) ) ( [7e0b46a](https://github.com/googleapis/python-spanner/commit/7e0b46aba7c48f7f944c0fca0cb394551b8d60c1) )
- Changes for float32 in dbapi ( [\#1115](https://github.com/googleapis/python-spanner/issues/1115) ) ( [c9f4fbf](https://github.com/googleapis/python-spanner/commit/c9f4fbf2a42054ed61916fb544c5aca947a50598) )

##### Bug Fixes

- Correcting name of variable from `table_schema` to `schema_name` ( [\#1114](https://github.com/googleapis/python-spanner/issues/1114) ) ( [a92c6d3](https://github.com/googleapis/python-spanner/commit/a92c6d347f2ae84779ec8662280ea894d558a887) )

##### Documentation

- Add sample for managed autoscaler ( [\#1111](https://github.com/googleapis/python-spanner/issues/1111) ) ( [e73c671](https://github.com/googleapis/python-spanner/commit/e73c6718b23bf78a8f264419b2ba378f95fa2554) )

## March 26, 2024

Announcement

Duet AI in Google Cloud is now Gemini. See our [blog post](https://blog.google/technology/ai/google-gemini-update-sundar-pichai-2024/) for more information.

Feature

You can optimize your writes by setting the maximum delay time of your Spanner write requests between 0 and 500 milliseconds. For more information, see [Throughput optimized writes](https://docs.cloud.google.com/spanner/docs/throughput-optimized-writes) .

## March 20, 2024

Feature

Leader-aware routing dynamically routes read-write transactions to the leader region in Spanner multi-region instances, reducing latency and improving performance. For more information, see [Leader-aware routing](https://docs.cloud.google.com/spanner/docs/leader-aware-routing) .

## March 19, 2024

Feature

Statistics for active partitioned data manipulation language (DML) queries are generally available. You can get insights on active partitioned DMLs queries and their progress from statistics tables in your Spanner database. For more information, see [Active partitioned DMLs statistics](https://docs.cloud.google.com/spanner/docs/introspection/active-partitioned-dmls) .

## March 11, 2024

Feature

Table renaming is generally available. This feature lets you rename tables in place or safely swap names using synonyms. For more information, see [Manage table names](https://docs.cloud.google.com/spanner/docs/table-name-synonym) .

## March 04, 2024

Feature

Spanner supports a client library interface. The interface leverages auto-generated admin clients instead of hand-written admin clients for improved efficiency and maintainability. While the older client library interface remains supported, all Spanner admin features released after March 1, 2024 are available only through the client library interface. All code samples in the [Spanner documentation](https://docs.cloud.google.com/spanner/docs/reference/libraries#use_the_client_library_for_administrator_operations) are updated to use the client library interface. The older client interface code samples are archived in GitHub for [Java](https://github.com/googleapis/java-spanner/tree/main/samples/snippets/src/main/java/com/example/spanner/admin/archived) , [Node.js](https://github.com/googleapis/nodejs-spanner/tree/main/samples/archivedhttps://github.com/googleapis/nodejs-spanner/tree/main/samples/archived) , [Python](https://github.com/googleapis/python-spanner/tree/main/samples/samples/archived) , and [PHP](https://github.com/GoogleCloudPlatform/php-docs-samples/tree/main/spanner/src/admin/archived) .

## February 29, 2024

Feature

Spanner regional endpoint is available in `me-central2` . You can use regional endpoints if your data location must be restricted and controlled to comply with regulatory requirements. For more information, see [Global and regional service endpoints](https://docs.cloud.google.com/spanner/docs/endpoints) .

Libraries

February 2024 Client libraries release note

A monthly digest of client library updates from across the [Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.56.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.55.0...spanner/v1.56.0) (2024-01-30)

##### Features

- **spanner/admin/database:** Add proto descriptors for proto and enum types in create/update/get database ddl requests ( [97d62c7](https://github.com/googleapis/google-cloud-go/commit/97d62c7a6a305c47670ea9c147edc444f4bf8620) )
- **spanner/spansql:** Add support for CREATE VIEW with SQL SECURITY DEFINER ( [\#8754](https://github.com/googleapis/google-cloud-go/issues/8754) ) ( [5f156e8](https://github.com/googleapis/google-cloud-go/commit/5f156e8c88f4729f569ee5b4ac9378dda3907997) )
- **spanner:** Add FLOAT32 enum to TypeCode ( [97d62c7](https://github.com/googleapis/google-cloud-go/commit/97d62c7a6a305c47670ea9c147edc444f4bf8620) )
- **spanner:** Add max_commit_delay API ( [af2f8b4](https://github.com/googleapis/google-cloud-go/commit/af2f8b4f3401c0b12dadb2c504aa0f902aee76de) )
- **spanner:** Add proto and enum types ( [00b9900](https://github.com/googleapis/google-cloud-go/commit/00b990061592a20a181e61faa6964b45205b76a7) )
- **spanner:** Add SelectAll method to decode from Spanner iterator.Rows to golang struct ( [\#9206](https://github.com/googleapis/google-cloud-go/issues/9206) ) ( [802088f](https://github.com/googleapis/google-cloud-go/commit/802088f1322752bb9ce9bab1315c3fed6b3a99aa) )

#### [1.57.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.56.0...spanner/v1.57.0) (2024-02-13)

##### Features

- **spanner:** Add OpenTelemetry implementation ( [\#9254](https://github.com/googleapis/google-cloud-go/issues/9254) ) ( [fc51cc2](https://github.com/googleapis/google-cloud-go/commit/fc51cc2ac71e8fb0b3e381379dc343630ed441e7) )
- **spanner:** Support max_commit_delay in Spanner transactions ( [\#9299](https://github.com/googleapis/google-cloud-go/issues/9299) ) ( [a8078f0](https://github.com/googleapis/google-cloud-go/commit/a8078f0b841281bd439c548db9d303f6b5ce54e6) )

##### Bug Fixes

- **spanner:** Enable universe domain resolution options ( [fd1d569](https://github.com/googleapis/google-cloud-go/commit/fd1d56930fa8a747be35a224611f4797b8aeb698) )
- **spanner:** Internal test package should import local version ( [\#9416](https://github.com/googleapis/google-cloud-go/issues/9416) ) ( [f377281](https://github.com/googleapis/google-cloud-go/commit/f377281a73553af9a9a2bee2181efe2e354e1c68) )
- **spanner:** SelectAll struct fields match should be case-insensitive ( [\#9417](https://github.com/googleapis/google-cloud-go/issues/9417) ) ( [7ff5356](https://github.com/googleapis/google-cloud-go/commit/7ff535672b868e6cba54abdf5dd92b9199e4d1d4) )
- **spanner:** Support time.Time and other custom types using SelectAll ( [\#9382](https://github.com/googleapis/google-cloud-go/issues/9382) ) ( [dc21234](https://github.com/googleapis/google-cloud-go/commit/dc21234268b08a4a21b2b3a1ed9ed74d65a289f0) )

##### Documentation

- **spanner:** Update the comment regarding eligible SQL shapes for PartitionQuery ( [e60a6ba](https://github.com/googleapis/google-cloud-go/commit/e60a6ba01acf2ef2e8d12e23ed5c6e876edeb1b7) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.57.0](https://github.com/googleapis/java-spanner/compare/v6.56.0...v6.57.0) (2024-01-29)

##### Features

- Add FLOAT32 enum to TypeCode ( [\#2800](https://github.com/googleapis/java-spanner/issues/2800) ) ( [383fea5](https://github.com/googleapis/java-spanner/commit/383fea5b5dc434621585a1b5cfd128a01780472a) )
- Add support for Proto Columns ( [\#2779](https://github.com/googleapis/java-spanner/issues/2779) ) ( [30d37dd](https://github.com/googleapis/java-spanner/commit/30d37dd80c91b2dffdfee732677607ce028fb8d2) )
- **spanner:** Add proto descriptors for proto and enum types in create/update/get database ddl requests ( [\#2774](https://github.com/googleapis/java-spanner/issues/2774) ) ( [4a906bf](https://github.com/googleapis/java-spanner/commit/4a906bf2719c30dcd7371f497a8a28c250db77be) )

##### Bug Fixes

- Remove google-cloud-spanner-executor from the BOM ( [\#2844](https://github.com/googleapis/java-spanner/issues/2844) ) ( [655000a](https://github.com/googleapis/java-spanner/commit/655000a3b0471b279cbcbe8a4a601337e7274ef8) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.22.0 ( [\#2785](https://github.com/googleapis/java-spanner/issues/2785) ) ( [f689f74](https://github.com/googleapis/java-spanner/commit/f689f742d8754134523ed0394b9c1b8256adcae2) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.23.0 ( [\#2801](https://github.com/googleapis/java-spanner/issues/2801) ) ( [95f064f](https://github.com/googleapis/java-spanner/commit/95f064f9f60a17de375e532ec6dd78dca0743e79) )

##### Documentation

- Samples and tests for instance APIs. ( [\#2768](https://github.com/googleapis/java-spanner/issues/2768) ) ( [88e24c7](https://github.com/googleapis/java-spanner/commit/88e24c7a7d046056605a2a824450e0153b339c86) )

#### [6.58.0](https://github.com/googleapis/java-spanner/compare/v6.57.0...v6.58.0) (2024-02-08)

##### Features

- Open telemetry implementation ( [\#2770](https://github.com/googleapis/java-spanner/issues/2770) ) ( [244d6a8](https://github.com/googleapis/java-spanner/commit/244d6a836795bf07dacd6b766436dbd6bf5fa912) )
- **spanner:** Support max_commit_delay in Spanner transactions ( [\#2854](https://github.com/googleapis/java-spanner/issues/2854) ) ( [e2b7ae6](https://github.com/googleapis/java-spanner/commit/e2b7ae66648ea775c18c71ab353edd6c0f50e7ac) )
- Support Directed Read in Connection API ( [\#2855](https://github.com/googleapis/java-spanner/issues/2855) ) ( [ee477c2](https://github.com/googleapis/java-spanner/commit/ee477c2e7c509ce4b7c43da3b68c1433c59e46fb) )

##### Bug Fixes

- Cast for Proto type ( [\#2862](https://github.com/googleapis/java-spanner/issues/2862) ) ( [0a95dba](https://github.com/googleapis/java-spanner/commit/0a95dba47681c9c4cc4e41ecfb5dadec6357bff6) )
- Ignore UnsupportedOperationException for virtual threads ( [\#2866](https://github.com/googleapis/java-spanner/issues/2866) ) ( [aa9ad7f](https://github.com/googleapis/java-spanner/commit/aa9ad7f5a5e2405e8082a542916c3d1fa7d0fa25) )
- Use default query options with statement cache ( [\#2860](https://github.com/googleapis/java-spanner/issues/2860) ) ( [741e4cf](https://github.com/googleapis/java-spanner/commit/741e4cf4eb51c4635078cfe2c52b7462bd4cbbd8) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.24.0 ( [\#2856](https://github.com/googleapis/java-spanner/issues/2856) ) ( [968877e](https://github.com/googleapis/java-spanner/commit/968877e4eff7da3ff27180c2a6129b04922d1af4) )

#### [6.59.0](https://github.com/googleapis/java-spanner/compare/v6.58.0...v6.59.0) (2024-02-15)

##### Features

- Support public methods to use autogenerated admin clients. ( [\#2878](https://github.com/googleapis/java-spanner/issues/2878) ) ( [53bcb3e](https://github.com/googleapis/java-spanner/commit/53bcb3eca2e814472c3def24e8e03d47652a8e42) )

##### Dependencies

- Update dependency com.google.cloud:sdk-platform-java-config to v3.25.0 ( [\#2888](https://github.com/googleapis/java-spanner/issues/2888) ) ( [8e2da51](https://github.com/googleapis/java-spanner/commit/8e2da5126263c7acd134fb7fcfeb590ca190ce8e) )

##### Documentation

- README for OpenTelemetry metrics and traces ( [\#2880](https://github.com/googleapis/java-spanner/issues/2880) ) ( [c8632f5](https://github.com/googleapis/java-spanner/commit/c8632f5b2f462420a8c2a1f4308a68a18a414472) )
- Samples and tests for database Admin APIs. ( [\#2775](https://github.com/googleapis/java-spanner/issues/2775) ) ( [14ae01c](https://github.com/googleapis/java-spanner/commit/14ae01cd82e455a0dc22d7e3bb8c362e541ede12) )

#### [6.60.0](https://github.com/googleapis/java-spanner/compare/v6.59.0...v6.60.0) (2024-02-21)

##### Features

- Add an API method for reordering firewall policies ( [62319f0](https://github.com/googleapis/java-spanner/commit/62319f032163c4ad3e8771dd5f92e7b8a086b5ee) )
- **spanner:** Add field for multiplexed session in spanner.proto ( [62319f0](https://github.com/googleapis/java-spanner/commit/62319f032163c4ad3e8771dd5f92e7b8a086b5ee) )
- Update TransactionOptions to include new option exclude_txn_from_change_streams ( [\#2853](https://github.com/googleapis/java-spanner/issues/2853) ) ( [62319f0](https://github.com/googleapis/java-spanner/commit/62319f032163c4ad3e8771dd5f92e7b8a086b5ee) )

##### Bug Fixes

- Add ensureDecoded to proto type ( [\#2897](https://github.com/googleapis/java-spanner/issues/2897) ) ( [e99b78c](https://github.com/googleapis/java-spanner/commit/e99b78c5d810195d368112eed2b185d2d99e62a9) )
- **spanner:** Fix write replace used by dataflow template and import export ( [\#2901](https://github.com/googleapis/java-spanner/issues/2901) ) ( [64b9042](https://github.com/googleapis/java-spanner/commit/64b90429d4fe53f8509a3923e046406b4bc5876a) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-trace to v2.36.0 ( [\#2749](https://github.com/googleapis/java-spanner/issues/2749) ) ( [51a348a](https://github.com/googleapis/java-spanner/commit/51a348a0c2b84106ea763721bed3420a0d07f30a) )

##### Documentation

- Update comments ( [62319f0](https://github.com/googleapis/java-spanner/commit/62319f032163c4ad3e8771dd5f92e7b8a086b5ee) )
- Update the comment regarding eligible SQL shapes for PartitionQuery ( [62319f0](https://github.com/googleapis/java-spanner/commit/62319f032163c4ad3e8771dd5f92e7b8a086b5ee) )

#### [6.60.1](https://github.com/googleapis/java-spanner/compare/v6.60.0...v6.60.1) (2024-02-23)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.37.0 ( [\#2920](https://github.com/googleapis/java-spanner/issues/2920) ) ( [a3441bb](https://github.com/googleapis/java-spanner/commit/a3441bbad546a1aac1349d6e142a4ac8d32d2a90) )
- Update dependency org.graalvm.buildtools:native-maven-plugin to v0.10.0 ( [\#2861](https://github.com/googleapis/java-spanner/issues/2861) ) ( [a652c3b](https://github.com/googleapis/java-spanner/commit/a652c3b6ef6d6ed87d581e73a26a5086acdc5f07) )
- Update dependency org.graalvm.buildtools:native-maven-plugin to v0.10.1 ( [\#2919](https://github.com/googleapis/java-spanner/issues/2919) ) ( [8800a28](https://github.com/googleapis/java-spanner/commit/8800a2894a1c17bde1a0da3ffcc868f10f7690d5) )
- Update dependency org.json:json to v20240205 ( [\#2913](https://github.com/googleapis/java-spanner/issues/2913) ) ( [277ed81](https://github.com/googleapis/java-spanner/commit/277ed81a0beb95ea57f95a9660a4a6b6adea645b) )
- Update dependency org.junit.vintage:junit-vintage-engine to v5.10.2 ( [\#2868](https://github.com/googleapis/java-spanner/issues/2868) ) ( [71a65ec](https://github.com/googleapis/java-spanner/commit/71a65ecee5af63996297f8692d569d2a9acfd8ac) )
- Update opentelemetry.version to v1.35.0 ( [\#2902](https://github.com/googleapis/java-spanner/issues/2902) ) ( [3286eae](https://github.com/googleapis/java-spanner/commit/3286eaea96a40c6ace8abed22040a637d291b09c) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.3.0](https://github.com/googleapis/nodejs-spanner/compare/v7.2.0...v7.3.0) (2024-02-08)

##### Features

- **spanner:** Add maxCommitDelay support ( [\#1992](https://github.com/googleapis/nodejs-spanner/issues/1992) ) ( [9f84408](https://github.com/googleapis/nodejs-spanner/commit/9f8440843fd8926a37ec300a318dad33b83b4f97) )

##### Bug Fixes

- **deps:** Update dependency google-gax to v4.1.0 ( [\#1981](https://github.com/googleapis/nodejs-spanner/issues/1981) ) ( [2a36150](https://github.com/googleapis/nodejs-spanner/commit/2a36150cb61e9abeef073724189cc651d29d8776) )
- **deps:** Update dependency google-gax to v4.2.0 ( [\#1988](https://github.com/googleapis/nodejs-spanner/issues/1988) ) ( [005589a](https://github.com/googleapis/nodejs-spanner/commit/005589a7727ee87948a55a6c7710f5150fc1c6a7) )
- **deps:** Update dependency google-gax to v4.2.1 ( [\#1989](https://github.com/googleapis/nodejs-spanner/issues/1989) ) ( [d2ae995](https://github.com/googleapis/nodejs-spanner/commit/d2ae9952e7449ce2321e69a6be36c9d50d863095) )
- **deps:** Update dependency google-gax to v4.3.0 ( [\#1990](https://github.com/googleapis/nodejs-spanner/issues/1990) ) ( [e625753](https://github.com/googleapis/nodejs-spanner/commit/e625753a37393f32d9e449aa7324763082f6c923) )

#### [7.4.0](https://github.com/googleapis/nodejs-spanner/compare/v7.3.0...v7.4.0) (2024-02-23)

##### Features

- **spanner:** Add PG.OID support ( [\#1948](https://github.com/googleapis/nodejs-spanner/issues/1948) ) ( [cf9df7a](https://github.com/googleapis/nodejs-spanner/commit/cf9df7a54c21ac995bbea9ad82c3544e4aff41b6) )
- Untyped param types ( [\#1869](https://github.com/googleapis/nodejs-spanner/issues/1869) ) ( [6ef44c3](https://github.com/googleapis/nodejs-spanner/commit/6ef44c383a90bf6ae95de531c83e21d2d58da159) )
- Update TransactionOptions to include new option exclude_txn_from_change_streams ( [\#1998](https://github.com/googleapis/nodejs-spanner/issues/1998) ) ( [937a7a1](https://github.com/googleapis/nodejs-spanner/commit/937a7a13f8c7660e21d34ebbaecad426b2bacd99) )

##### Bug Fixes

- **deps:** Update dependency google-gax to v4.3.1 ( [\#1995](https://github.com/googleapis/nodejs-spanner/issues/1995) ) ( [bed4832](https://github.com/googleapis/nodejs-spanner/commit/bed4832445e72c7116fe5495c79d989664220b38) )
- Only reset pending value with resume token ( [\#2000](https://github.com/googleapis/nodejs-spanner/issues/2000) ) ( [f337089](https://github.com/googleapis/nodejs-spanner/commit/f337089567d7d92c9467e311be7d72b0a7dc8047) ), closes [\#1959](https://github.com/googleapis/nodejs-spanner/issues/1959)

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.42.0](https://github.com/googleapis/python-spanner/compare/v3.41.0...v3.42.0) (2024-01-30)

##### Features

- Add FLOAT32 enum to TypeCode ( [5b94dac](https://github.com/googleapis/python-spanner/commit/5b94dac507cebde2025d412da0a82373afdbdaf5) )
- Add max_commit_delay API ( [\#1078](https://github.com/googleapis/python-spanner/issues/1078) ) ( [ec87c08](https://github.com/googleapis/python-spanner/commit/ec87c082570259d6e16834326859a73f6ee8286a) )
- Add proto descriptors for proto and enum types in create/update/get database ddl requests ( [5b94dac](https://github.com/googleapis/python-spanner/commit/5b94dac507cebde2025d412da0a82373afdbdaf5) )
- Fixing and refactoring transaction retry logic in dbapi. Also adding interceptors support for testing ( [\#1056](https://github.com/googleapis/python-spanner/issues/1056) ) ( [6640888](https://github.com/googleapis/python-spanner/commit/6640888b7845b7e273758ed9a6de3044e281f555) )
- Implementation of run partition query ( [\#1080](https://github.com/googleapis/python-spanner/issues/1080) ) ( [f3b23b2](https://github.com/googleapis/python-spanner/commit/f3b23b268766b6ff2704da18945a1b607a6c8909) )

##### Bug Fixes

- Few fixes in DBAPI ( [\#1085](https://github.com/googleapis/python-spanner/issues/1085) ) ( [1ed5a47](https://github.com/googleapis/python-spanner/commit/1ed5a47ce9cfe7be0805a2961b24d7b682cda2f3) )
- Small fix in description when metadata is not present in cursor's \_result_set ( [\#1088](https://github.com/googleapis/python-spanner/issues/1088) ) ( [57643e6](https://github.com/googleapis/python-spanner/commit/57643e66a64d9befeb27fbbad360613ff69bd48c) )
- **spanner:** Add SpannerAsyncClient import to spanner_v1 package ( [\#1086](https://github.com/googleapis/python-spanner/issues/1086) ) ( [2d98b54](https://github.com/googleapis/python-spanner/commit/2d98b5478ee201d9fbb2775975f836def2817e33) )

##### Documentation

- Samples and tests for auto-generated createDatabase and createInstance APIs. ( [\#1065](https://github.com/googleapis/python-spanner/issues/1065) ) ( [16c510e](https://github.com/googleapis/python-spanner/commit/16c510eeed947beb87a134c64ca83a37f90b03fb) )

## February 26, 2024

Feature

The following GoogleSQL JSON functions are [generally available](https://cloud.google.com/products#product-launch-stages) (GA):

- [`LAX_BOOL`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#lax_bool) : Attempts to convert a JSON value to a SQL `BOOL` value.
- [`LAX_FLOAT64`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#lax_double) : Attempts to convert a JSON value to a SQL `FLOAT64` value.
- [`LAX_INT64`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#lax_int64) : Attempts to convert a JSON value to a SQL `INT64` value.
- [`LAX_STRING`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#lax_string) : Attempts to convert a JSON value to a SQL `STRING` value.
- [`BOOL`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#bool_for_json) : Converts a JSON boolean to a SQL `BOOL` value.
- [`FLOAT64`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#double_for_json) : Converts a JSON number to a SQL `FLOAT64` value.
- [`INT64`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#int64_for_json) : Converts a JSON number to a SQL `INT64` value.
- [`STRING`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#string_for_json) : Converts a JSON string to a SQL `STRING` value.
- [`JSON_TYPE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/json_functions#json_type) : Gets the JSON type of the outermost JSON value and converts the name of this type to a SQL `STRING` value.

## February 21, 2024

Feature

[The OpenCensus libraries are archived](https://opentelemetry.io/blog/2023/sunsetting-opencensus/) . Spanner supports OpenTelemetry. We recommend all OpenCensus users to migrate to OpenTelemetry for your observability needs. For more information, see [Examine latency in a Spanner component with OpenTelemetry](https://docs.cloud.google.com/spanner/docs/capture-visualize-latency) .

## February 07, 2024

Fixed

Made changes to the information schema to improve the accuracy of data type reporting.

The `information_schema.columns.spanner_type` and `information_schema.index_columns.spanner_type` columns include a limit value for the `character varying(limit_value)` and `character varying(limit_value)[]` types.

## January 31, 2024

Libraries

A monthly digest of client library updates from across the [Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.55.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.54.0...spanner/v1.55.0) (2024-01-08)

##### Features

- **spanner:** Add directed reads feature ( [\#7668](https://github.com/googleapis/google-cloud-go/issues/7668) ) ( [a42604a](https://github.com/googleapis/google-cloud-go/commit/a42604a3a6ea90c38a2ff90d036a79fd070174fd) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.56.0](https://github.com/googleapis/java-spanner/compare/v6.55.0...v6.56.0) (2024-01-05)

##### Features

- Add autoscaling config in the instance to support autoscaling in systests ( [\#2756](https://github.com/googleapis/java-spanner/issues/2756) ) ( [99ae565](https://github.com/googleapis/java-spanner/commit/99ae565c5e90a2862b4f195fe64656ba8a05373d) )
- Add support for Directed Read options ( [\#2766](https://github.com/googleapis/java-spanner/issues/2766) ) ( [26c6c63](https://github.com/googleapis/java-spanner/commit/26c6c634b685bce66ce7caf05057a98e9cc6f5dc) )
- Update OwlBot.yaml file to pull autogenerated executor code ( [\#2754](https://github.com/googleapis/java-spanner/issues/2754) ) ( [20562d4](https://github.com/googleapis/java-spanner/commit/20562d4d7e62ab20bb1c4e78547b218a9a506f21) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.21.0 ( [\#2772](https://github.com/googleapis/java-spanner/issues/2772) ) ( [173f520](https://github.com/googleapis/java-spanner/commit/173f520f931073c4c6ddf3b3d98d255fb575914f) )

##### Documentation

- Samples and tests for auto-generated createDatabase and createInstance APIs. ( [\#2764](https://github.com/googleapis/java-spanner/issues/2764) ) ( [74a586f](https://github.com/googleapis/java-spanner/commit/74a586f8713ef742d65400da8f04a750316faf78) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.2.0](https://github.com/googleapis/nodejs-spanner/compare/v7.1.0...v7.2.0) (2024-01-11)

##### Features

- Support for Directed Reads ( [\#1966](https://github.com/googleapis/nodejs-spanner/issues/1966) ) ( [c0a4363](https://github.com/googleapis/nodejs-spanner/commit/c0a43638c81dd769cc55e021cc4cf1d93db8a72a) )

##### Bug Fixes

- **deps:** Update dependency @google-cloud/precise-date to v4 ( [\#1903](https://github.com/googleapis/nodejs-spanner/issues/1903) ) ( [7464c8b](https://github.com/googleapis/nodejs-spanner/commit/7464c8b2412a9b718cd8981363cb982aebbe3723) )
- **deps:** Update dependency @types/stack-trace to v0.0.33 ( [\#1952](https://github.com/googleapis/nodejs-spanner/issues/1952) ) ( [45ab751](https://github.com/googleapis/nodejs-spanner/commit/45ab751da1f0f73bc06c8b8e0007b457fa75518f) )
- **deps:** Update dependency retry-request to v7 ( [\#1934](https://github.com/googleapis/nodejs-spanner/issues/1934) ) ( [c575c80](https://github.com/googleapis/nodejs-spanner/commit/c575c80b17e5fdf2cbba24c806fa21f26c2010dc) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.41.0](https://github.com/googleapis/python-spanner/compare/v3.40.1...v3.41.0) (2024-01-10)

##### Features

- Add BatchWrite API ( [\#1011](https://github.com/googleapis/python-spanner/issues/1011) ) ( [d0e4ffc](https://github.com/googleapis/python-spanner/commit/d0e4ffccea071feaa2ca012a0e3f60a945ed1a13) )
- Add PG.OID type cod annotation ( [\#1023](https://github.com/googleapis/python-spanner/issues/1023) ) ( [2d59dd0](https://github.com/googleapis/python-spanner/commit/2d59dd09b8f14a37c780d8241a76e2f109ba88b0) )
- Add support for Directed Reads ( [\#1000](https://github.com/googleapis/python-spanner/issues/1000) ) ( [c4210b2](https://github.com/googleapis/python-spanner/commit/c4210b28466cfd88fffe546140a005a8e0a1af23) )
- Add support for Python 3.12 ( [\#1040](https://github.com/googleapis/python-spanner/issues/1040) ) ( [b28dc9b](https://github.com/googleapis/python-spanner/commit/b28dc9b0f97263d3926043fe5dfcb4cdc75ab35a) )
- Batch Write API implementation and samples ( [\#1027](https://github.com/googleapis/python-spanner/issues/1027) ) ( [aa36b07](https://github.com/googleapis/python-spanner/commit/aa36b075ebb13fa952045695a8f4eb6d21ae61ff) )
- Implementation for batch dml in dbapi ( [\#1055](https://github.com/googleapis/python-spanner/issues/1055) ) ( [7a92315](https://github.com/googleapis/python-spanner/commit/7a92315c8040dbf6f652974e19cd63abfd6cda2f) )
- Implementation for Begin and Rollback clientside statements ( [\#1041](https://github.com/googleapis/python-spanner/issues/1041) ) ( [15623cd](https://github.com/googleapis/python-spanner/commit/15623cda0ac1eb5dd71434c9064134cfa7800a79) )
- Implementation for partitioned query in dbapi ( [\#1067](https://github.com/googleapis/python-spanner/issues/1067) ) ( [63daa8a](https://github.com/googleapis/python-spanner/commit/63daa8a682824609b5a21699d95b0f41930635ef) )
- Implementation of client side statements that return ( [\#1046](https://github.com/googleapis/python-spanner/issues/1046) ) ( [bb5fa1f](https://github.com/googleapis/python-spanner/commit/bb5fa1fb75dba18965cddeacd77b6af0a05b4697) )
- Implementing client side statements in dbapi (starting with commit) ( [\#1037](https://github.com/googleapis/python-spanner/issues/1037) ) ( [eb41b0d](https://github.com/googleapis/python-spanner/commit/eb41b0da7c1e60561b46811d7307e879f071c6ce) )
- Introduce compatibility with native namespace packages ( [\#1036](https://github.com/googleapis/python-spanner/issues/1036) ) ( [5d80ab0](https://github.com/googleapis/python-spanner/commit/5d80ab0794216cd093a21989be0883b02eaa437a) )
- Return list of dictionaries for execute streaming sql ( [\#1003](https://github.com/googleapis/python-spanner/issues/1003) ) ( [b534a8a](https://github.com/googleapis/python-spanner/commit/b534a8aac116a824544d63a24e38f3d484e0d207) )
- **spanner:** Add autoscaling config to the instance proto ( [\#1022](https://github.com/googleapis/python-spanner/issues/1022) ) ( [4d490cf](https://github.com/googleapis/python-spanner/commit/4d490cf9de600b16a90a1420f8773b2ae927983d) )
- **spanner:** Add directed_read_option in spanner.proto ( [\#1030](https://github.com/googleapis/python-spanner/issues/1030) ) ( [84d662b](https://github.com/googleapis/python-spanner/commit/84d662b056ca4bd4177b3107ba463302b5362ff9) )

##### Bug Fixes

- Executing existing DDL statements on executemany statement execution ( [\#1032](https://github.com/googleapis/python-spanner/issues/1032) ) ( [07fbc45](https://github.com/googleapis/python-spanner/commit/07fbc45156a1b42a5e61c9c4b09923f239729aa8) )
- Fix for flaky test_read_timestamp_client_side_autocommit test ( [\#1071](https://github.com/googleapis/python-spanner/issues/1071) ) ( [0406ded](https://github.com/googleapis/python-spanner/commit/0406ded8b0abcdc93a7a2422247a14260f5c620c) )
- Require google-cloud-core \>= 1.4.4 ( [\#1015](https://github.com/googleapis/python-spanner/issues/1015) ) ( [a2f87b9](https://github.com/googleapis/python-spanner/commit/a2f87b9d9591562877696526634f0c7c4dd822dd) )
- Require proto-plus 1.22.2 for python 3.11 ( [\#880](https://github.com/googleapis/python-spanner/issues/880) ) ( [7debe71](https://github.com/googleapis/python-spanner/commit/7debe7194b9f56b14daeebb99f48787174a9471b) )
- Use `retry_async` instead of `retry` in async client ( [\#1044](https://github.com/googleapis/python-spanner/issues/1044) ) ( [1253ae4](https://github.com/googleapis/python-spanner/commit/1253ae46011daa3a0b939e22e957dd3ab5179210) )

##### Documentation

- Minor formatting ( [498dba2](https://github.com/googleapis/python-spanner/commit/498dba26a7c1a1cb710a92c0167272ff5c0eef27) )

Feature

You can create Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) in Johannesburg, South Africa ( `africa-south1` ).

## January 30, 2024

Feature

Spanner directed reads is available in Preview. Directed reads provides the flexibility to route read-only transactions and single reads to a specific replica type or region in a multi-region instance configuration. For more information, see [Directed reads](https://docs.cloud.google.com/spanner/docs/directed-reads) .

## January 23, 2024

Feature

Spanner supports the GoogleSQL [`INSERT OR IGNORE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-ignore) and [`INSERT OR UPDATE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-or-update) clauses.

Spanner supports the PostgreSQL [`ON CONFLICT DO NOTHING`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/dml-syntax#on_conflict_clause) and [`ON CONFLICT DO UPDATE SET`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/dml-syntax#on_conflict_clause) clauses.

## January 22, 2024

Feature

Spanner supports `COSINE_DISTANCE()` and `EUCLIDEAN_DISTANCE()` functions (in Preview). You can use these vector distance functions to perform similarity vector search. For more information, see [Perform similarity vector search in Spanner by finding the K-nearest neighbors](https://docs.cloud.google.com/spanner/docs/find-k-nearest-neighbors) .

## December 27, 2023

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.53.1](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.53.0...spanner/v1.53.1) (2023-12-01)

##### Bug Fixes

- **spanner:** Handle nil error when cleaning up long running session ( [\#9052](https://github.com/googleapis/google-cloud-go/issues/9052) ) ( [a93bc26](https://github.com/googleapis/google-cloud-go/commit/a93bc2696bf9ae60aae93af0e8c4911b58514d31) )
- **spanner:** MarshalJSON function caused errors for certain values ( [\#9063](https://github.com/googleapis/google-cloud-go/issues/9063) ) ( [afe7c98](https://github.com/googleapis/google-cloud-go/commit/afe7c98036c198995075530d4228f1f4ae3f1222) )

#### [1.54.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.53.1...spanner/v1.54.0) (2023-12-14)

##### Features

- **spanner/executor:** Add autoscaling config in the instance to support autoscaling in systests ( [29effe6](https://github.com/googleapis/google-cloud-go/commit/29effe600e16f24a127a1422ec04263c4f7a600a) )
- **spanner:** New clients ( [\#9127](https://github.com/googleapis/google-cloud-go/issues/9127) ) ( [2c97389](https://github.com/googleapis/google-cloud-go/commit/2c97389ddacdfc140a06f74498cc2753bb040a4d) )

##### Bug Fixes

- **spanner:** Use json.Number for decoding unknown values from spanner ( [\#9054](https://github.com/googleapis/google-cloud-go/issues/9054) ) ( [40d1392](https://github.com/googleapis/google-cloud-go/commit/40d139297bd484408c63c9d6ad1d7035d9673c1c) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.55.0](https://github.com/googleapis/java-spanner/compare/v6.54.0...v6.55.0) (2023-12-01)

##### Features

- Add java sample for managed autoscaler ( [\#2709](https://github.com/googleapis/java-spanner/issues/2709) ) ( [9ea4f4f](https://github.com/googleapis/java-spanner/commit/9ea4f4fe2925410b3defb4e53f3f0a328cc2e738) )

##### Bug Fixes

- **deps:** Update the Java code generator (gapic-generator-java) to 2.30.0 ( [\#2703](https://github.com/googleapis/java-spanner/issues/2703) ) ( [961aa78](https://github.com/googleapis/java-spanner/commit/961aa7894be41ff87f1b460aa374ee2ed75a163b) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.20.0 ( [\#2746](https://github.com/googleapis/java-spanner/issues/2746) ) ( [12bcabb](https://github.com/googleapis/java-spanner/commit/12bcabbf1ef82b19524400ebe280d9986bf70ea7) )
- Update dependency commons-io:commons-io to v2.15.1 ( [\#2745](https://github.com/googleapis/java-spanner/issues/2745) ) ( [b9d9571](https://github.com/googleapis/java-spanner/commit/b9d9571dcc2d1d004cd785d79e45754c0ce63a51) )

## December 19, 2023

Feature

Spanner supports partition queries whose query plans don't contain any distributed unions. To learn more about how to read data in parallel using partition queries, see [Read data in parallel](https://docs.cloud.google.com/spanner/docs/reads#read_data_in_parallel) .

## December 18, 2023

Change

The number of mutations per commit that Spanner supports has increased from 40,000 to 80,000. For more information, see [Quotas and limits](https://docs.cloud.google.com/spanner/quotas#limits-for) .

## December 14, 2023

Feature

Data Catalog support in Spanner is generally available. For more information, see [Manage resources using Data Catalog](https://docs.cloud.google.com/spanner/docs/dc-integration) .

## December 05, 2023

Feature

Spanner supports the following PostgreSQL functions:

- [`unnest`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/arrays)
- [`array_length`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#array-functions)
- [`array(subquery)`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#array-functions)
- [`date_trunc`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#date-time-functions)
- [`extract`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#date-time-functions)
- [`spanner.date_bin`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#spanner-date-time-functions)
- [`spanner.timestamptz_add`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#spanner-date-time-functions)
- [`spanner.timestamptz_subtract`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#spanner-date-time-functions)

For more information, see [working with arrays in PostgreSQL-dialect databases](https://docs.cloud.google.com/spanner/docs/reference/postgresql/arrays) .

## November 30, 2023

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.52.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.51.0...spanner/v1.52.0) (2023-11-14)

##### Features

- **spanner:** Add DML, DQL, Mutation, Txn Actions and Utility methods for executor framework ( [\#8976](https://github.com/googleapis/google-cloud-go/issues/8976) ) ( [ca76671](https://github.com/googleapis/google-cloud-go/commit/ca7667194007394bdcade8058fa84c1fe19c06b1) )
- **spanner:** Add lastUseTime property to session ( [\#8942](https://github.com/googleapis/google-cloud-go/issues/8942) ) ( [b560cfc](https://github.com/googleapis/google-cloud-go/commit/b560cfcf967ff6dec0cd6ac4b13045470945f30b) )
- **spanner:** Add method ( [\#8945](https://github.com/googleapis/google-cloud-go/issues/8945) ) ( [411a51e](https://github.com/googleapis/google-cloud-go/commit/411a51e320fe21ffe830cdaa6bb4e4d77f7a996b) )
- **spanner:** Add methods to return Row fields ( [\#8953](https://github.com/googleapis/google-cloud-go/issues/8953) ) ( [e22e70f](https://github.com/googleapis/google-cloud-go/commit/e22e70f44f83aab4f8b89af28fcd24216d2e740e) )
- **spanner:** Add PG.OID type cod annotation ( [\#8749](https://github.com/googleapis/google-cloud-go/issues/8749) ) ( [ffb0dda](https://github.com/googleapis/google-cloud-go/commit/ffb0ddabf3d9822ba8120cabaf25515fd32e9615) )
- **spanner:** Admin, Batch, Partition actions for executor framework ( [\#8932](https://github.com/googleapis/google-cloud-go/issues/8932) ) ( [b2db89e](https://github.com/googleapis/google-cloud-go/commit/b2db89e03a125cde31a7ea86eecc3fbb08ebd281) )
- **spanner:** Auto-generated executor framework proto changes ( [\#8713](https://github.com/googleapis/google-cloud-go/issues/8713) ) ( [2ca939c](https://github.com/googleapis/google-cloud-go/commit/2ca939cba4bc240f2bfca7d5683708fd3a94fd74) )
- **spanner:** BatchWrite ( [\#8652](https://github.com/googleapis/google-cloud-go/issues/8652) ) ( [507d232](https://github.com/googleapis/google-cloud-go/commit/507d232cdb09bd941ebfe800bdd4bfc020346f5d) )
- **spanner:** Executor framework server and worker proxy ( [\#8714](https://github.com/googleapis/google-cloud-go/issues/8714) ) ( [6b931ee](https://github.com/googleapis/google-cloud-go/commit/6b931eefb9aa4a18758788167bdcf9e2fad1d7b9) )
- **spanner:** Fix falkiness ( [\#8977](https://github.com/googleapis/google-cloud-go/issues/8977) ) ( [ca8d3cb](https://github.com/googleapis/google-cloud-go/commit/ca8d3cbf80f7fc2f47beb53b95138040c83097db) )
- **spanner:** Long running transaction clean up - disabled ( [\#8177](https://github.com/googleapis/google-cloud-go/issues/8177) ) ( [461d11e](https://github.com/googleapis/google-cloud-go/commit/461d11e913414e9de822e5f1acdf19c8f3f953d5) )
- **spanner:** Update code for session leaks cleanup ( [\#8978](https://github.com/googleapis/google-cloud-go/issues/8978) ) ( [cc83515](https://github.com/googleapis/google-cloud-go/commit/cc83515d0c837c8b1596a97b6f09d519a0f75f72) )

##### Bug Fixes

- **spanner:** Bump google.golang.org/api to v0.149.0 ( [8d2ab9f](https://github.com/googleapis/google-cloud-go/commit/8d2ab9f320a86c1c0fab90513fc05861561d0880) )
- **spanner:** Expose Mutations field in MutationGroup ( [\#8923](https://github.com/googleapis/google-cloud-go/issues/8923) ) ( [42180cf](https://github.com/googleapis/google-cloud-go/commit/42180cf1134885188270f75126a65fa71b03c033) )
- **spanner:** Update grpc-go to v1.56.3 ( [343cea8](https://github.com/googleapis/google-cloud-go/commit/343cea8c43b1e31ae21ad50ad31d3b0b60143f8c) )
- **spanner:** Update grpc-go to v1.59.0 ( [81a97b0](https://github.com/googleapis/google-cloud-go/commit/81a97b06cb28b25432e4ece595c55a9857e960b7) )

##### Documentation

- **spanner:** Updated comment formatting ( [24e410e](https://github.com/googleapis/google-cloud-go/commit/24e410efbb6add2d33ecfb6ad98b67dc8894e578) )

#### [1.53.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.52.0...spanner/v1.53.0) (2023-11-15)

##### Features

- **spanner:** Enable long running transaction clean up ( [\#8969](https://github.com/googleapis/google-cloud-go/issues/8969) ) ( [5d181bb](https://github.com/googleapis/google-cloud-go/commit/5d181bb3a6fea55b8d9d596213516129006bdae2) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.53.0](https://github.com/googleapis/java-spanner/compare/v6.52.1...v6.53.0) (2023-11-06)

##### Features

- Move session lastUseTime parameter from PooledSession to SessionImpl class. Fix updation of the parameter for chained RPCs within one transaction. ( [\#2704](https://github.com/googleapis/java-spanner/issues/2704) ) ( [e75a281](https://github.com/googleapis/java-spanner/commit/e75a2818124621a3ab837151a8e1094fa6c3b8f3) )
- Rely on graal-sdk version declaration from property in java-shared-config ( [\#2696](https://github.com/googleapis/java-spanner/issues/2696) ) ( [cfab83a](https://github.com/googleapis/java-spanner/commit/cfab83ad3bd1a026e0b3da5a4cc2154b0f8c3ddf) )

##### Bug Fixes

- Prevent illegal negative timeout values into thread sleep() method in ITTransactionManagerTest. ( [\#2715](https://github.com/googleapis/java-spanner/issues/2715) ) ( [1c26cf6](https://github.com/googleapis/java-spanner/commit/1c26cf60efa1b98203af9b21a47e37c8fb1e0e97) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.19.0 ( [\#2719](https://github.com/googleapis/java-spanner/issues/2719) ) ( [e320753](https://github.com/googleapis/java-spanner/commit/e320753b2bd125f94775db9c71a4b7803fa49c38) )
- Update dependency com.google.cloud:google-cloud-trace to v2.28.0 ( [\#2670](https://github.com/googleapis/java-spanner/issues/2670) ) ( [078b7ca](https://github.com/googleapis/java-spanner/commit/078b7ca95548ac984c79d29197032b3f813abbcf) )
- Update dependency com.google.cloud:google-cloud-trace to v2.29.0 ( [\#2714](https://github.com/googleapis/java-spanner/issues/2714) ) ( [b400eca](https://github.com/googleapis/java-spanner/commit/b400ecabb9fa6f262befa903163746fac2c7c15e) )
- Update dependency commons-cli:commons-cli to v1.6.0 ( [\#2710](https://github.com/googleapis/java-spanner/issues/2710) ) ( [e3e8f6a](https://github.com/googleapis/java-spanner/commit/e3e8f6ac82d827280299038d3962fe66b110e0c4) )
- Update dependency commons-io:commons-io to v2.15.0 ( [\#2712](https://github.com/googleapis/java-spanner/issues/2712) ) ( [a5f59aa](https://github.com/googleapis/java-spanner/commit/a5f59aa3e992d0594519983880a29f17301923e7) )
- Update dependency org.graalvm.buildtools:junit-platform-native to v0.9.28 ( [\#2692](https://github.com/googleapis/java-spanner/issues/2692) ) ( [d8a2b02](https://github.com/googleapis/java-spanner/commit/d8a2b02d43a68e04bebb2349af61cc8901ccd667) )
- Update dependency org.graalvm.buildtools:native-maven-plugin to v0.9.28 ( [\#2705](https://github.com/googleapis/java-spanner/issues/2705) ) ( [2b17f09](https://github.com/googleapis/java-spanner/commit/2b17f095a294defa5ea022c243fa750486b7d496) )
- Update dependency org.junit.vintage:junit-vintage-engine to v5.10.1 ( [\#2723](https://github.com/googleapis/java-spanner/issues/2723) ) ( [9cf6d0e](https://github.com/googleapis/java-spanner/commit/9cf6d0eae5d2a86c89de2d252d0f4a4dab0b54a4) )

#### [6.54.0](https://github.com/googleapis/java-spanner/compare/v6.53.0...v6.54.0) (2023-11-15)

##### Features

- Enable session leaks prevention by cleaning up long-running tra… ( [\#2655](https://github.com/googleapis/java-spanner/issues/2655) ) ( [faa7e5d](https://github.com/googleapis/java-spanner/commit/faa7e5dff17897b0432bc505b7ed24c33805f418) )

##### Bug Fixes

- Copy backup issue when backup is done across different instance IDs ( [\#2732](https://github.com/googleapis/java-spanner/issues/2732) ) ( [7f6b158](https://github.com/googleapis/java-spanner/commit/7f6b1582770d2270efc9501136afb17a2677eaeb) )
- Respect SPANNER_EMULATOR_HOST env var when autoConfigEmulator=true ( [\#2730](https://github.com/googleapis/java-spanner/issues/2730) ) ( [9c19934](https://github.com/googleapis/java-spanner/commit/9c19934a6170232f6ac2478ef9bfcdb2914d2562) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-trace to v2.30.0 ( [\#2725](https://github.com/googleapis/java-spanner/issues/2725) ) ( [8618042](https://github.com/googleapis/java-spanner/commit/8618042bb716d8a6626bacee59f9e6c6f0d50362) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.1.0](https://github.com/googleapis/nodejs-spanner/compare/v7.0.0...v7.1.0) (2023-11-16)

##### Features

- Add PG.OID type cod annotation ( [69192b5](https://github.com/googleapis/nodejs-spanner/commit/69192b50ead0bde98676cb647ba4bf8a3112bb02) )
- **spanner:** Add autoscaling config to the instance proto ( [\#1935](https://github.com/googleapis/nodejs-spanner/issues/1935) ) ( [fe285c6](https://github.com/googleapis/nodejs-spanner/commit/fe285c67074ba36aaf5b49ea867c0d5851d83717) )
- **spanner:** Add directed_read_option in spanner.proto ( [69192b5](https://github.com/googleapis/nodejs-spanner/commit/69192b50ead0bde98676cb647ba4bf8a3112bb02) )

##### Bug Fixes

- **deps:** Update dependency @types/stack-trace to v0.0.31 ( [\#1924](https://github.com/googleapis/nodejs-spanner/issues/1924) ) ( [96af405](https://github.com/googleapis/nodejs-spanner/commit/96af4051c6717dfcbbc6e117e3ecd7f8e9dd758a) )
- **deps:** Update dependency @types/stack-trace to v0.0.32 ( [\#1939](https://github.com/googleapis/nodejs-spanner/issues/1939) ) ( [cb66474](https://github.com/googleapis/nodejs-spanner/commit/cb66474e995a90c1288e70842f723c51f1ffd37d) )
- **deps:** Update dependency google-gax to v4.0.4 ( [\#1926](https://github.com/googleapis/nodejs-spanner/issues/1926) ) ( [361fe6a](https://github.com/googleapis/nodejs-spanner/commit/361fe6a812f56c6834f1f7c7db60fc1083243768) )
- **deps:** Update dependency google-gax to v4.0.5 ( [\#1937](https://github.com/googleapis/nodejs-spanner/issues/1937) ) ( [ab26075](https://github.com/googleapis/nodejs-spanner/commit/ab260759be2fcc9ff80342f710b4c807742da2c5) )

## November 21, 2023

Feature

Spanner emulator support for the PostgreSQL dialect is generally available. To learn more about the emulator, see [Emulate Spanner locally](https://docs.cloud.google.com/spanner/docs/emulator) .

## November 16, 2023

Feature

Spanner supports automatic cleanup of long running transactions (in Preview). To enhable this feature, use the Java or Go client library to automatically remove long running transactions that might cause session leaks and receive warning logs about problematic transactions. For more information, see [Automatic cleanup of session leaks](https://docs.cloud.google.com/spanner/docs/sessions#automatic_cleanup_of_session_leaks) .

Feature

Spanner supports Hibernate ORM 6.3 in GoogleSQL Hibernate dialect. For more information, see [Integrate Spanner with Hibernate ORM (GoogleSQL dialect)](https://docs.cloud.google.com/spanner/docs/use-hibernate) .

## November 15, 2023

Feature

Spanner provides an integration workflow with Vertex AI Vector Search to enable vector similarity search on data stored in Spanner. For more information, see [Export embeddings from Spanner to Vector Search](https://docs.cloud.google.com/spanner/docs/vector-search-embeddings) .

## November 13, 2023

Feature

Managed autoscaler for compute capacity on Spanner instances is in preview. With managed autoscaler, Spanner automatically increases or decreases compute capacity on the instance in response to changing workload or storage needs and user defined goals. For more information, see [Managed autoscaler](https://docs.cloud.google.com/spanner/docs/managed-autoscaler) .

## November 10, 2023

Feature

Spanner supports batch-oriented scans. For certain queries, Spanner chooses a batch-oriented processing mode to help improve scan throughput and performance. For more information, see [Optimize scans](https://docs.cloud.google.com/spanner/docs/sql-best-practices#optimize-scans) .

## November 07, 2023

Feature

Spanner supports the Go programming language ORM, GORM, with GoogleSQL-dialect databases. For more information, see [Integrate Spanner with GORM (GoogleSQL dialect)](https://docs.cloud.google.com/spanner/docs/use-gorm) .

## November 02, 2023

Feature

Table and index operations statistics are generally available. This feature helps you get insights and monitor usages of your tables and indexes in your database. For more information, see [Table operations statistics](https://docs.cloud.google.com/spanner/docs/introspection/table-operations-statistics) .

## October 31, 2023

Feature

The Spanner `ExecuteBatchDml` API applies optimizations to groups of statements within a batch to enable faster and more efficient data updates. For more information, see [Improve latency with batch DML](https://docs.cloud.google.com/spanner/docs/dml-best-practices#batch-dml) .

## October 30, 2023

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.50.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.49.0...spanner/v1.50.0) (2023-10-03)

##### Features

- **spanner/spansql:** Add support for aggregate functions ( [\#8498](https://github.com/googleapis/google-cloud-go/issues/8498) ) ( [d440d75](https://github.com/googleapis/google-cloud-go/commit/d440d75f19286653afe4bc81a5f2efcfc4fa152c) )
- **spanner/spansql:** Add support for bit functions, sequence functions and `GENERATE_UUID` ( [\#8482](https://github.com/googleapis/google-cloud-go/issues/8482) ) ( [3789882](https://github.com/googleapis/google-cloud-go/commit/3789882c8b30a6d3100a56c1dcc8844952605637) )
- **spanner/spansql:** Add support for SEQUENCE statements ( [\#8481](https://github.com/googleapis/google-cloud-go/issues/8481) ) ( [ccd0205](https://github.com/googleapis/google-cloud-go/commit/ccd020598921f1b5550587c95b4ceddf580705bb) )
- **spanner:** Add `BatchWrite` API ( [02a899c](https://github.com/googleapis/google-cloud-go/commit/02a899c95eb9660128506cf94525c5a75bedb308) )
- **spanner:** Allow non-default service accounts ( [\#8488](https://github.com/googleapis/google-cloud-go/issues/8488) ) ( [c90dd00](https://github.com/googleapis/google-cloud-go/commit/c90dd00350fa018dbc5f0af5aabce80e80be0b90) )

#### [1.51.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.50.0...spanner/v1.51.0) (2023-10-17)

##### Features

- **spanner/admin/instance:** Add autoscaling config to the instance proto ( [\#8701](https://github.com/googleapis/google-cloud-go/issues/8701) ) ( [56ce871](https://github.com/googleapis/google-cloud-go/commit/56ce87195320634b07ae0b012efcc5f2b3813fb0) )

##### Bug Fixes

- **spanner:** Update golang.org/x/net to v0.17.0 ( [174da47](https://github.com/googleapis/google-cloud-go/commit/174da47254fefb12921bbfc65b7829a453af6f5d) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.48.0](https://github.com/googleapis/java-spanner/compare/v6.47.0...v6.48.0) (2023-09-26)

##### Features

- Add support for `BatchWriteAtLeastOnce` ( [\#2520](https://github.com/googleapis/java-spanner/issues/2520) ) ( [8ea7bd1](https://github.com/googleapis/java-spanner/commit/8ea7bd18e92a7c5547d8a33bf46c1e322326447b) )

##### Bug Fixes

- Retry aborted errors for `writeAtLeastOnce` ( [\#2627](https://github.com/googleapis/java-spanner/issues/2627) ) ( [2addb19](https://github.com/googleapis/java-spanner/commit/2addb1930a7b9ada4a4304a44a36d8ff1397cf9e) )

##### Dependencies

- Update actions/checkout action to v4 ( [\#2608](https://github.com/googleapis/java-spanner/issues/2608) ) ( [59f3e70](https://github.com/googleapis/java-spanner/commit/59f3e7047a0a9578350b37b46395377d7e014763) )
- Update dependency org.graalvm.buildtools:junit-platform-native to v0.9.27 ( [\#2574](https://github.com/googleapis/java-spanner/issues/2574) ) ( [e804a4c](https://github.com/googleapis/java-spanner/commit/e804a4c60f369ca88b804fef182b5afae44bd05e) )
- Update dependency org.graalvm.buildtools:native-maven-plugin to v0.9.27 ( [\#2575](https://github.com/googleapis/java-spanner/issues/2575) ) ( [6fe132a](https://github.com/googleapis/java-spanner/commit/6fe132a7c1458da4fc28c950009d152643ced038) )

#### [6.49.0](https://github.com/googleapis/java-spanner/compare/v6.48.0...v6.49.0) (2023-09-28)

##### Features

- Add session pool option for modelling a timeout around session acquisition. ( [\#2641](https://github.com/googleapis/java-spanner/issues/2641) ) ( [428e294](https://github.com/googleapis/java-spanner/commit/428e294b94392e290921b5c0eda0139c57d3a185) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.16.1 ( [\#2637](https://github.com/googleapis/java-spanner/issues/2637) ) ( [3f48624](https://github.com/googleapis/java-spanner/commit/3f486245f574f3a6abf4d3b9146b51dc92cf5eea) )

##### Documentation

- Improve timeout and retry sample ( [\#2630](https://github.com/googleapis/java-spanner/issues/2630) ) ( [f03ce56](https://github.com/googleapis/java-spanner/commit/f03ce56119e2985286ede15352f19c3cb6f39979) )
- Remove reference to returning clauses for Batch DML ( [\#2644](https://github.com/googleapis/java-spanner/issues/2644) ) ( [038d8ca](https://github.com/googleapis/java-spanner/commit/038d8cac3fe06ca2dcf0b4e85f5e536b73ce9313) )

#### [6.50.0](https://github.com/googleapis/java-spanner/compare/v6.49.0...v6.50.0) (2023-10-09)

##### Features

- Support setting core pool size for async API in system property ( [\#2632](https://github.com/googleapis/java-spanner/issues/2632) ) ( [e51c55d](https://github.com/googleapis/java-spanner/commit/e51c55d332bacb9d174a24b0d842b2cba4762db8) ), closes [\#2631](https://github.com/googleapis/java-spanner/issues/2631)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-trace to v2.24.0 ( [\#2577](https://github.com/googleapis/java-spanner/issues/2577) ) ( [311c2ad](https://github.com/googleapis/java-spanner/commit/311c2ad97311490893f3abf4da5fe4d511c445dd) )

#### [6.50.1](https://github.com/googleapis/java-spanner/compare/v6.50.0...v6.50.1) (2023-10-11)

##### Bug Fixes

- Noop in case there is no change in `autocommit` value for `setAutocommit()` method ( [\#2662](https://github.com/googleapis/java-spanner/issues/2662) ) ( [9f51b64](https://github.com/googleapis/java-spanner/commit/9f51b6445f064439379af752372a3490a2fd5087) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.17.0 ( [\#2660](https://github.com/googleapis/java-spanner/issues/2660) ) ( [96b9dd6](https://github.com/googleapis/java-spanner/commit/96b9dd6b6a0ee7b1a0a1cc58a8880a10799665e6) )
- Update dependency commons-io:commons-io to v2.14.0 ( [\#2649](https://github.com/googleapis/java-spanner/issues/2649) ) ( [fa1b73c](https://github.com/googleapis/java-spanner/commit/fa1b73c1bf4700be5e8865211817e2bc7cc77119) )

#### [6.51.0](https://github.com/googleapis/java-spanner/compare/v6.50.1...v6.51.0) (2023-10-14)

##### Features

- **spanner:** Add autoscaling config to the instance proto ( [\#2674](https://github.com/googleapis/java-spanner/issues/2674) ) ( [8d38ca3](https://github.com/googleapis/java-spanner/commit/8d38ca393a6c0f9df18c9d02fa9392e11af01246) )

##### Bug Fixes

- Always include default client lib header ( [\#2676](https://github.com/googleapis/java-spanner/issues/2676) ) ( [74fd174](https://github.com/googleapis/java-spanner/commit/74fd174a84f6f97949b9caaadddf366aafd4a469) )

#### [6.52.0](https://github.com/googleapis/java-spanner/compare/v6.51.0...v6.52.0) (2023-10-19)

##### Features

- Add support for Managed Autoscaler ( [\#2624](https://github.com/googleapis/java-spanner/issues/2624) ) ( [e5e6923](https://github.com/googleapis/java-spanner/commit/e5e6923a351670ab237c411bb4a549533dac1b6b) )

#### [6.52.1](https://github.com/googleapis/java-spanner/compare/v6.52.0...v6.52.1) (2023-10-20)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.18.0 ( [\#2691](https://github.com/googleapis/java-spanner/issues/2691) ) ( [b425021](https://github.com/googleapis/java-spanner/commit/b4250218a500eb1540920ed0023454d06c54d621) )

## October 26, 2023

Feature

Spanner supports `FULL JOIN` with `USING` in PostgreSQL-dialect databases. For information about PostgreSQL queries in Spanner, see [PostgreSQL queries](https://docs.cloud.google.com/spanner/docs/reference/postgresql/query-syntax) .

## October 23, 2023

Feature

Spanner PostgreSQL supports the `SELECT DISTINCT` statement. For more information, see [`SELECT`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/query-syntax#select) .

## October 17, 2023

Feature

[Query Optimizer version 6](https://docs.cloud.google.com/spanner/docs/query-optimizer/versions) is generally available, and is the default optimizer version.

## October 11, 2023

Feature

Spanner has made improvements that provide higher throughput for instances located in select Spanner regional and multi-region instance configurations. These improvements are available without additional cost or any configuration changes. For more information, see [Performance improvements](https://docs.cloud.google.com/spanner/docs/performance#improved-performance) .

## October 09, 2023

Feature

Spanner batch write is available in Preview. You can use Spanner batch write to commit multiple mutations non-atomically in a single request with low latency. For more information, see [Modify data using batch write](https://docs.cloud.google.com/spanner/docs/batch-write) .

Feature

Spanner Vertex AI integration supports Vertex AI Generative AI text embeddings and the [`text-bison`](https://docs.cloud.google.com/vertex-ai/docs/generative-ai/model-reference/text) model. For more information, see [Get Vertex AI text embeddings](https://docs.cloud.google.com/spanner/docs/ml-tutorial-embeddings) .

## October 05, 2023

Feature

Spanner sampled query plans are available in GA. You can view samples of historic query plans and compare the performance of a query over time. For more information, see [Sampled query plans](https://docs.cloud.google.com/spanner/docs/query-execution-plans#sampled-plans) .

## September 25, 2023

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.46.0](https://github.com/googleapis/java-spanner/compare/v6.45.3...v6.46.0) (2023-09-06)

##### Features

- Add support for databoost ( [\#2505](https://github.com/googleapis/java-spanner/issues/2505) ) ( [dd3e9a0](https://github.com/googleapis/java-spanner/commit/dd3e9a0fe4846edcab9501b71c3d9e0fa24ed75b) )
- Support PostgreSQL for autoConfigEmulator ( [\#2601](https://github.com/googleapis/java-spanner/issues/2601) ) ( [fbf1df9](https://github.com/googleapis/java-spanner/commit/fbf1df9f3fb12faaead8634b88fd4843cbdedf5b) )

##### Bug Fixes

- Fix kokoro windows java8 ci ( [\#2573](https://github.com/googleapis/java-spanner/issues/2573) ) ( [465df7b](https://github.com/googleapis/java-spanner/commit/465df7bad12fbea7dbcf6dbabb1b29d088c42665) )

##### Documentation

- Add sample for transaction timeouts ( [\#2599](https://github.com/googleapis/java-spanner/issues/2599) ) ( [59cec9b](https://github.com/googleapis/java-spanner/commit/59cec9b9cdad169bd8de8ab7b264b04150dda7fb) )

#### [6.47.0](https://github.com/googleapis/java-spanner/compare/v6.46.0...v6.47.0) (2023-09-12)

##### Features

- Add devcontainers for enabling github codespaces usage. ( [\#2605](https://github.com/googleapis/java-spanner/issues/2605) ) ( [a7d60f1](https://github.com/googleapis/java-spanner/commit/a7d60f13781f87054a1631ca511492c5c8334751) )
- Disable dynamic code loading properties by default ( [\#2606](https://github.com/googleapis/java-spanner/issues/2606) ) ( [d855ebb](https://github.com/googleapis/java-spanner/commit/d855ebbd2dec11cdd6cdbe326de81115632598cd) )

##### Bug Fixes

- Add reflection configurations for com.google.rpc classes ( [\#2617](https://github.com/googleapis/java-spanner/issues/2617) ) ( [c42460a](https://github.com/googleapis/java-spanner/commit/c42460ae7b6bb5874cc18c7aecff34186dcbff2a) )
- Avoid unbalanced session pool creation ( [\#2442](https://github.com/googleapis/java-spanner/issues/2442) ) ( [db751ce](https://github.com/googleapis/java-spanner/commit/db751ceebc8b6981d00cd07ce4742196cc1dd50d) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.15.0 ( [\#2615](https://github.com/googleapis/java-spanner/issues/2615) ) ( [ac762fb](https://github.com/googleapis/java-spanner/commit/ac762fbf079db79eab5f2ebee971b850ac89eb11) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [7.0.0](https://github.com/googleapis/nodejs-spanner/compare/v6.16.0...v7.0.0) (2023-08-30)

##### ⚠ BREAKING CHANGES

- upgrade to Node 14 ( [\#1890](https://github.com/googleapis/nodejs-spanner/issues/1890) )

##### Bug Fixes

- Idwaiter with multiple requests ( [\#1910](https://github.com/googleapis/nodejs-spanner/issues/1910) ) ( [83dd1f8](https://github.com/googleapis/nodejs-spanner/commit/83dd1f8201d07898bd3ddff9e339dfbcef7d7ace) )

##### Miscellaneous Chores

- Upgrade to Node 14 ( [\#1890](https://github.com/googleapis/nodejs-spanner/issues/1890) ) ( [0024772](https://github.com/googleapis/nodejs-spanner/commit/0024772b750de404cd44771e320fe89cd430f064) )

## September 19, 2023

Feature

You can create Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) in Dammam, Saudi Arabia ( `me-central2` ).

## September 13, 2023

Feature

You can create definer's rights views in Spanner. A definer's rights view adds additional security functionality by providing different privileges on the view and the underlying schema objects. Users with access to a definer's rights view can see and query its contents even if they don't have access to the view's underlying schema objects. For more information, see [About views](https://docs.cloud.google.com/spanner/docs/views) .

## September 11, 2023

Feature

[Query optimizer version 6](https://docs.cloud.google.com/spanner/docs/query-optimizer/versions) is generally available. Version 5 remains the default optimizer version in production.

## September 06, 2023

Feature

A Spanner [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instance-configurations#multi-region-configurations) is available in Asia - `asia2` (Mumbai/Delhi/Singapore).

## August 31, 2023

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.48.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.47.0...spanner/v1.48.0) (2023-08-18)

##### Features

- **spanner/spansql:** Add complete set of math functions ( [\#8246](https://github.com/googleapis/google-cloud-go/issues/8246) ) ( [d7a238e](https://github.com/googleapis/google-cloud-go/commit/d7a238eca2a9b08e968cea57edc3708694673e22) )
- **spanner/spansql:** Add support for foreign key actions ( [\#8296](https://github.com/googleapis/google-cloud-go/issues/8296) ) ( [d78b851](https://github.com/googleapis/google-cloud-go/commit/d78b8513b13a9a2c04b8097f0d89f85dcfd73797) )
- **spanner/spansql:** Add support for IF NOT EXISTS and IF EXISTS clause ( [\#8245](https://github.com/googleapis/google-cloud-go/issues/8245) ) ( [96840ab](https://github.com/googleapis/google-cloud-go/commit/96840ab1232bbdb788e37f81cf113ee0f1b4e8e7) )
- **spanner:** Add integration tests for Bit Reversed Sequences ( [\#7924](https://github.com/googleapis/google-cloud-go/issues/7924) ) ( [9b6e7c6](https://github.com/googleapis/google-cloud-go/commit/9b6e7c6061dc69683d7f558faed7f4249da5b7cb) )

##### Bug Fixes

- **spanner:** Reset buffer after abort on first SQL statement ( [\#8440](https://github.com/googleapis/google-cloud-go/issues/8440) ) ( [d980b42](https://github.com/googleapis/google-cloud-go/commit/d980b42f33968ef25061be50e18038d73b0503b6) )
- **spanner:** REST query UpdateMask bug ( [df52820](https://github.com/googleapis/google-cloud-go/commit/df52820b0e7721954809a8aa8700b93c5662dc9b) )

#### [1.49.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.48.0...spanner/v1.49.0) (2023-08-24)

##### Features

- **spanner/spannertest:** Support INSERT DML ( [\#7820](https://github.com/googleapis/google-cloud-go/issues/7820) ) ( [3dda7b2](https://github.com/googleapis/google-cloud-go/commit/3dda7b27ec536637d8ebaa20937fc8019c930481) )

##### Bug Fixes

- **spanner:** Transaction was started in a different session ( [\#8467](https://github.com/googleapis/google-cloud-go/issues/8467) ) ( [6c21558](https://github.com/googleapis/google-cloud-go/commit/6c21558f75628908a70de79c62aff2851e756e7b) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.45.0](https://github.com/googleapis/java-spanner/compare/v6.44.0...v6.45.0) (2023-08-04)

##### Features

- Enable leader aware routing by default in Connection API. This enables its use in the JDBC driver and PGAdapter. The update contains performance optimisations that will reduce the latency of read/write transactions that originate from a region other than the default leader region. ( [2a85446](https://github.com/googleapis/java-spanner/commit/2a85446b162b006ce84a86285af1767c879b27ed) )
- Enable leader aware routing by default. This update contains performance optimisations that will reduce the latency of read/write transactions that originate from a region other than the default leader region. ( [441c1b0](https://github.com/googleapis/java-spanner/commit/441c1b03c3e976c6304a99fefd93b5c4291e5364) )
- Long running transaction clean up background task. Adding configuration options for closing inactive transactions. ( [\#2419](https://github.com/googleapis/java-spanner/issues/2419) ) ( [423e1a4](https://github.com/googleapis/java-spanner/commit/423e1a4b483798d9683ff9bd232b53d76e09beb0) )
- Support partitioned queries + data boost in Connection API ( [\#2540](https://github.com/googleapis/java-spanner/issues/2540) ) ( [4e31d04](https://github.com/googleapis/java-spanner/commit/4e31d046f5d80abe8876a729ddba045c70f3261d) )

##### Bug Fixes

- Apply stream wait timeout ( [\#2544](https://github.com/googleapis/java-spanner/issues/2544) ) ( [5a12cd2](https://github.com/googleapis/java-spanner/commit/5a12cd29601253423c5738be5471a036fd0334be) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.14.0 ( [\#2562](https://github.com/googleapis/java-spanner/issues/2562) ) ( [dbd5c75](https://github.com/googleapis/java-spanner/commit/dbd5c75be39262003092ff4a925ed470cc45f8be) )
- Update dependency org.openjdk.jmh:jmh-core to v1.37 ( [\#2565](https://github.com/googleapis/java-spanner/issues/2565) ) ( [d5c36bf](https://github.com/googleapis/java-spanner/commit/d5c36bfbb67ecb14854944779da6e4dbd93f3559) )
- Update dependency org.openjdk.jmh:jmh-generator-annprocess to v1.37 ( [\#2566](https://github.com/googleapis/java-spanner/issues/2566) ) ( [73e92d4](https://github.com/googleapis/java-spanner/commit/73e92d42fe6d334b6efa6485246dc67858adb0a9) )

#### [6.45.1](https://github.com/googleapis/java-spanner/compare/v6.45.0...v6.45.1) (2023-08-11)

##### Bug Fixes

- Always allow metadata queries ( [\#2580](https://github.com/googleapis/java-spanner/issues/2580) ) ( [ebb17fc](https://github.com/googleapis/java-spanner/commit/ebb17fc8aeac5fc75e4f135f33dba970f2480585) )

#### [6.45.2](https://github.com/googleapis/java-spanner/compare/v6.45.1...v6.45.2) (2023-08-14)

##### Bug Fixes

- GetColumnCount would fail for empty partititioned result sets ( [\#2588](https://github.com/googleapis/java-spanner/issues/2588) ) ( [9a2f3fc](https://github.com/googleapis/java-spanner/commit/9a2f3fc01748224fc8084fbf2b4a0223426b1603) )

#### [6.45.3](https://github.com/googleapis/java-spanner/compare/v6.45.2...v6.45.3) (2023-08-17)

##### Bug Fixes

- Use streaming read/query settings for stream retry ( [\#2579](https://github.com/googleapis/java-spanner/issues/2579) ) ( [f78b838](https://github.com/googleapis/java-spanner/commit/f78b838e294f9c29bfc34a5d964933657b70417f) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [6.15.0](https://github.com/googleapis/nodejs-spanner/compare/v6.14.0...v6.15.0) (2023-08-04)

##### Features

- Enable leader aware routing by default. This update contains performance optimisations that will reduce the latency of read/write transactions that originate from a region other than the default leader region. ( [6852d99](https://github.com/googleapis/nodejs-spanner/commit/6852d99b858eb323ac3fc5e61905b8bf59486062) )

#### [6.16.0](https://github.com/googleapis/nodejs-spanner/compare/v6.15.0...v6.16.0) (2023-08-07)

##### Features

- Bit reverse sequence ( [\#1846](https://github.com/googleapis/nodejs-spanner/issues/1846) ) ( [4154c02](https://github.com/googleapis/nodejs-spanner/commit/4154c02f4c5ac1aa23f4c7c61521ab6fbabadfb8) )

##### Bug Fixes

- Databoost tests ( [\#1870](https://github.com/googleapis/nodejs-spanner/issues/1870) ) ( [45e13c7](https://github.com/googleapis/nodejs-spanner/commit/45e13c70607abf717d533a8c5b1c58752a5439cb) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.39.0](https://github.com/googleapis/python-spanner/compare/v3.38.0...v3.39.0) (2023-08-02)

##### Features

- Foreign key on delete cascade action testing and samples ( [\#910](https://github.com/googleapis/python-spanner/issues/910) ) ( [681c8ee](https://github.com/googleapis/python-spanner/commit/681c8eead40582addf75e02c159ea1ff9d6de85e) )

##### Documentation

- Minor formatting ( [\#991](https://github.com/googleapis/python-spanner/issues/991) ) ( [60efc42](https://github.com/googleapis/python-spanner/commit/60efc426cf26c4863d81743a5545c5f296308815) )

#### [3.40.0](https://github.com/googleapis/python-spanner/compare/v3.39.0...v3.40.0) (2023-08-04)

##### Features

- Enable leader aware routing by default. This update contains performance optimisations that will reduce the latency of read/write transactions that originate from a region other than the default leader region. ( [e8dbfe7](https://github.com/googleapis/python-spanner/commit/e8dbfe709d72a04038e05166adbad275642f1f22) )

#### [3.40.1](https://github.com/googleapis/python-spanner/compare/v3.40.0...v3.40.1) (2023-08-17)

##### Bug Fixes

- Fix to reload table when checking if table exists ( [\#1002](https://github.com/googleapis/python-spanner/issues/1002) ) ( [53bda62](https://github.com/googleapis/python-spanner/commit/53bda62c4996d622b7a11e860841c16e4097bded) )

## August 29, 2023

Feature

Spanner Studio includes [Gemini](https://docs.cloud.google.com/duet-ai/docs/overview) (in Preview), an AI-powered collaborator in Google Cloud that accelerates SQL development by helping you write SQL statements. For more information, see [Write SQL with Gemini assistance](https://docs.cloud.google.com/spanner/docs/write-sql-duet-ai) .

## August 24, 2023

Feature

Spanner has added 13 PostgreSQL functions and operators:

- [`ARRAY_UPPER(anyarray, dimension)` function](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#array-functions)
- [`QUOTE_IDENT(string)` function](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#string_functions)
- [`SUBSTRING(string, pattern)` function](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#string_functions)
- [`DATE - DATE` operator](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#date-time-operators)
- [`DATE - INTEGER` operator](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#date-time-operators)
- [`DATE + INTEGER` operator](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#date-time-operators)
- [`REGEXP_MATCH(string, pattern [, flags])` function](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#pattern-matching)
- [`REGEXP_SPLIT_TO_ARRAY(string, pattern [, flags])` function](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#pattern-matching)
- [`STRING !~ PATTERN` operator](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#pattern-matching-operators)
- [`TO_CHAR(timestamptz, format)` , `TO_CHAR(double, format)` , `TO_CHAR(bigint, format)` , `TO_CHAR(numeric, format)` function](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#formatting-functions)
- [`TO_NUMBER(string, format)` function](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#formatting-functions)
- [`TO_DATE(string, format)` function](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#formatting-functions)
- [`TO_TIMESTAMP(string, format)` function](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#formatting-functions)

For more information, see [Supported PostgreSQL functions](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators) .

## August 23, 2023

Feature

Spanner supports integer sequences and bit reversal.

- The `SEQUENCE` DDL statement generates unique, uniformly distributed integers as part of a primary key `DEFAULT` expression. For more information, see `SEQUENCE` ( [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#sequence_statements) , [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#sequence_statements) ).

- The bit reverse function lets you map existing integer keys using the same logic as a bit-reversed sequence to avoid hotspotting. For more information, see `BIT_REVERSE` ( [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/bit_functions#bit_reverse) , [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#mathematical) ).

For overview information and scenarios for when to use these features, see [Primary key default values management](https://docs.cloud.google.com/spanner/docs/primary-key-default-value) .

Feature

Spanner supports generating a UUID (v4) as part of a table's primary key `DEFAULT` expression using the [`GENERATE_UUID`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/functions-and-operators#generate_uuid) function in GoogleSQL or [`generate_uuid()`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#utility) in PostgreSQL-dialect databases.

For overview information and scenarios for when to use this feature, see [Primary key default values management](https://docs.cloud.google.com/spanner/docs/primary-key-default-value) .

## August 22, 2023

Feature

You can create Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) in Berlin, Germany ( `europe-west10` ).

## August 21, 2023

Feature

Spanner Studio enhances the Spanner query editor in the Google Cloud console, with full support for SQL, DML, and DDL operations. The Spanner Studio also features an **Explorer** pane (in Preview) which lets you interactively browse, query, and modify your database. For more information, see [Manage your data using the Google Cloud console](https://docs.cloud.google.com/spanner/docs/manage-data-using-console) .

## August 17, 2023

Feature

Data Boost for Spanner is available in all regions. For information about Data Boost, see [Data Boost overview](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) .

## August 15, 2023

Feature

Spanner lets you check the progress on long-running operations, such as backups, restores, and schema updates. This feature is [generally available (GA)](https://cloud.google.com/products#product-launch-stages) . For more information, see [Check the progress of a long-running schema update operation](https://docs.cloud.google.com/spanner/docs/manage-and-observe-long-running-operations#check_the_progress_of_a_long-running_schema_update_operation) and [Check the progress of a long-running backup or restore operation](https://docs.cloud.google.com/spanner/docs/manage-and-observe-long-running-operations#check_the_progress_of_a_long-running_backup_or_restore_operation) .

## August 08, 2023

Feature

Spanner database deletion protection is generally available. You can enable database deletion protection to prevent the accidental deletion of databases. For more information, see [Prevent accidental database deletion](https://docs.cloud.google.com/spanner/docs/prevent-database-deletion) .

## July 31, 2023

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.43.1](https://github.com/googleapis/java-spanner/compare/v6.43.0...v6.43.1) (2023-06-26)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.20.0 ( [\#2492](https://github.com/googleapis/java-spanner/issues/2492) ) ( [faa6807](https://github.com/googleapis/java-spanner/commit/faa68073673e789e35b600dab72152591a647dc6) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.21.0 ( [\#2510](https://github.com/googleapis/java-spanner/issues/2510) ) ( [f10400b](https://github.com/googleapis/java-spanner/commit/f10400baf2d320991e75794250b9e1b2fb218718) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.12.0.with temp exclusions. ( [\#2512](https://github.com/googleapis/java-spanner/issues/2512) ) ( [ce04645](https://github.com/googleapis/java-spanner/commit/ce0464527ef489d351b9086f6bb8922f295f1897) )
- Update dependency com.google.cloud:google-cloud-trace to v2.19.0 ( [\#2493](https://github.com/googleapis/java-spanner/issues/2493) ) ( [1dc7cea](https://github.com/googleapis/java-spanner/commit/1dc7cea723658c43b8c8d2e085c964371fb72223) )
- Update dependency com.google.cloud:google-cloud-trace to v2.20.0 ( [\#2511](https://github.com/googleapis/java-spanner/issues/2511) ) ( [2ea52ec](https://github.com/googleapis/java-spanner/commit/2ea52ec1cef2468e6c36b76797a3878f270badaa) )
- Update dependency commons-io:commons-io to v2.13.0 ( [\#2490](https://github.com/googleapis/java-spanner/issues/2490) ) ( [b087b0e](https://github.com/googleapis/java-spanner/commit/b087b0e813cacb4f08d12815d9371fe9c004ca9e) )
- Update dependency org.graalvm.buildtools:junit-platform-native to v0.9.23 ( [\#2500](https://github.com/googleapis/java-spanner/issues/2500) ) ( [0b794a6](https://github.com/googleapis/java-spanner/commit/0b794a68d57eb990e013fdd05c72eaed868497b0) )
- Update dependency org.graalvm.buildtools:native-maven-plugin to v0.9.23 ( [\#2501](https://github.com/googleapis/java-spanner/issues/2501) ) ( [9db5c78](https://github.com/googleapis/java-spanner/commit/9db5c7850b53fa10d1856d88908d5e8e95467206) )
- Update dependency org.json:json to v20230618 ( [\#2504](https://github.com/googleapis/java-spanner/issues/2504) ) ( [8a87fee](https://github.com/googleapis/java-spanner/commit/8a87fee19bb2dd41495a15740893375c8778f71a) )

#### [6.43.2](https://github.com/googleapis/java-spanner/compare/v6.43.1...v6.43.2) (2023-07-09)

##### Bug Fixes

- Recognize ABORT statements for PostgreSQL ( [\#2479](https://github.com/googleapis/java-spanner/issues/2479) ) ( [da47b0a](https://github.com/googleapis/java-spanner/commit/da47b0aef7a2e03fc9b5e25cf036ef8d8d001672) )

##### Documentation

- Add background info for session pool ( [\#2498](https://github.com/googleapis/java-spanner/issues/2498) ) ( [0bbb1a1](https://github.com/googleapis/java-spanner/commit/0bbb1a1b5ac6b9d4ea061a2f2a4d26c3bd958d7e) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.13.0 ( [\#2521](https://github.com/googleapis/java-spanner/issues/2521) ) ( [bdb2461](https://github.com/googleapis/java-spanner/commit/bdb2461dfa90535241c333d1cfee33afc2b33eca) )

#### [6.44.0](https://github.com/googleapis/java-spanner/compare/v6.43.2...v6.44.0) (2023-07-27)

##### Features

- Foreign key on delete cascade ( [\#2340](https://github.com/googleapis/java-spanner/issues/2340) ) ( [f659105](https://github.com/googleapis/java-spanner/commit/f6591053db1c38f0e13e35cba2087a68d3ab1b01) )

##### Bug Fixes

- Add imports used in sample files. ( [\#2532](https://github.com/googleapis/java-spanner/issues/2532) ) ( [9a6d3fc](https://github.com/googleapis/java-spanner/commit/9a6d3fcbaa8d44f2e08407252a69beca1e4525b1) )

##### Documentation

- Fixing errors ( [\#2536](https://github.com/googleapis/java-spanner/issues/2536) ) ( [8aa407f](https://github.com/googleapis/java-spanner/commit/8aa407f3e1b4c6cf66b679e698992a6a5e3034c0) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.22.0 ( [\#2525](https://github.com/googleapis/java-spanner/issues/2525) ) ( [be0db6f](https://github.com/googleapis/java-spanner/commit/be0db6f10509fe3e5f74aa6ca6569552e65cb87a) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.23.0 ( [\#2542](https://github.com/googleapis/java-spanner/issues/2542) ) ( [67351dd](https://github.com/googleapis/java-spanner/commit/67351dd2cb557d461421c4a0321ae6d2d0fd9dcb) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.13.1 ( [\#2537](https://github.com/googleapis/java-spanner/issues/2537) ) ( [9396d8d](https://github.com/googleapis/java-spanner/commit/9396d8d8b5450dd545687af6c513b7f6c7a6c283) )
- Update dependency com.google.cloud:google-cloud-trace to v2.21.0 ( [\#2526](https://github.com/googleapis/java-spanner/issues/2526) ) ( [2d95234](https://github.com/googleapis/java-spanner/commit/2d952347e0eb7db42387d8abb91d4b11d51cef9c) )
- Update dependency com.google.cloud:google-cloud-trace to v2.22.0 ( [\#2543](https://github.com/googleapis/java-spanner/issues/2543) ) ( [47c6a43](https://github.com/googleapis/java-spanner/commit/47c6a430405ebf1c2fe392991e3f4554e9ac37aa) )
- Update dependency org.graalvm.sdk:graal-sdk to v22.3.3 ( [\#2533](https://github.com/googleapis/java-spanner/issues/2533) ) ( [0806b11](https://github.com/googleapis/java-spanner/commit/0806b116cc6650b353cee26c83929e7bcdcb1c34) )
- Update dependency org.junit.vintage:junit-vintage-engine to v5.10.0 ( [\#2539](https://github.com/googleapis/java-spanner/issues/2539) ) ( [8801b2b](https://github.com/googleapis/java-spanner/commit/8801b2bf639b7903958668a2274a6e5d457de00a) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [6.14.0](https://github.com/googleapis/nodejs-spanner/compare/v6.13.0...v6.14.0) (2023-07-21)

##### Features

- Foreign key delete cascade testing, samples ( [\#1825](https://github.com/googleapis/nodejs-spanner/issues/1825) ) ( [74a54b0](https://github.com/googleapis/nodejs-spanner/commit/74a54b03f0d73a62edd524fa8d0248aea7ddf344) )
- Set LAR as False ( [\#1883](https://github.com/googleapis/nodejs-spanner/issues/1883) ) ( [ed510e8](https://github.com/googleapis/nodejs-spanner/commit/ed510e8545876e188e7bd782b6db80e677c3063c) )

#### [6.13.0](https://github.com/googleapis/nodejs-spanner/compare/v6.12.0...v6.13.0) (2023-07-21)

##### Features

- Enable leader aware routing by default. This update contains performance optimisations that will reduce the latency of read/write transactions that originate from a region other than the default leader region. ( [87cd5e6](https://github.com/googleapis/nodejs-spanner/commit/87cd5e6ecdf6d888dd0e7fe712b7070c58b32d42) )

##### Bug Fixes

- **deps:** Update dependency yargs to v17 ( [\#1866](https://github.com/googleapis/nodejs-spanner/issues/1866) ) ( [24e321f](https://github.com/googleapis/nodejs-spanner/commit/24e321f6327cfdfc191a84bb47d80a156eff5be9) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.38.0](https://github.com/googleapis/python-spanner/compare/v3.37.0...v3.38.0) (2023-07-21)

##### Features

- Set LAR as False ( [\#980](https://github.com/googleapis/python-spanner/issues/980) ) ( [75e8a59](https://github.com/googleapis/python-spanner/commit/75e8a59ff5d7f15088b9c4ba5961345746e35bcc) )

#### [3.37.0](https://github.com/googleapis/python-spanner/compare/v3.36.0...v3.37.0) (2023-07-21)

##### Features

- Enable leader aware routing by default. This update contains performance optimisations that will reduce the latency of read/write transactions that originate from a region other than the default leader region. ( [402b101](https://github.com/googleapis/python-spanner/commit/402b1015a58f0982d5e3f9699297db82d3cdd7b2) )

##### Bug Fixes

- Add async context manager return types ( [\#967](https://github.com/googleapis/python-spanner/issues/967) ) ( [7e2e712](https://github.com/googleapis/python-spanner/commit/7e2e712f9ee1e8643c5c59dbd1d15b13b3c0f3ea) )

##### Documentation

- Fix documentation structure ( [\#949](https://github.com/googleapis/python-spanner/issues/949) ) ( [b73e47b](https://github.com/googleapis/python-spanner/commit/b73e47bb43f5767957685400c7876d6a8b7489a3) )

## July 18, 2023

Feature

Spanner supports cascading deletes for foreign keys. For more information, see [Foreign key actions](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#how-to-define-foreign-key-action) .

## June 26, 2023

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.47.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.46.0...spanner/v1.47.0) (2023-06-20)

##### Features

- **spanner/admin/database:** Add DdlStatementActionInfo and add actions to UpdateDatabaseDdlMetadata ( [01eff11](https://github.com/googleapis/google-cloud-go/commit/01eff11eedb3edde69cc33db23e26be6a7e42f10) )
- **spanner:** Add databoost property for batch transactions ( [\#8152](https://github.com/googleapis/google-cloud-go/issues/8152) ) ( [fc49c78](https://github.com/googleapis/google-cloud-go/commit/fc49c78c9503c6dd4cbcba8c15e887415a744136) )
- **spanner:** Add tests for database roles in PG dialect ( [\#7898](https://github.com/googleapis/google-cloud-go/issues/7898) ) ( [dc84649](https://github.com/googleapis/google-cloud-go/commit/dc84649c546fe09b0bab09991086c156bd78cb3f) )
- **spanner:** Enable client to server compression ( [\#7899](https://github.com/googleapis/google-cloud-go/issues/7899) ) ( [3a047d2](https://github.com/googleapis/google-cloud-go/commit/3a047d2a449b0316a9000539ec9797e47cdd5c91) )
- **spanner:** Update all direct dependencies ( [b340d03](https://github.com/googleapis/google-cloud-go/commit/b340d030f2b52a4ce48846ce63984b28583abde6) )

##### Bug Fixes

- **spanner:** Fix TestRetryInfoTransactionOutcomeUnknownError flaky behaviour ( [\#7959](https://github.com/googleapis/google-cloud-go/issues/7959) ) ( [f037795](https://github.com/googleapis/google-cloud-go/commit/f03779538f949fb4ad93d5247d3c6b3e5b21091a) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.43.0](https://github.com/googleapis/java-spanner/compare/v6.42.3...v6.43.0) (2023-06-07)

##### Features

- Delay transaction start option ( [\#2462](https://github.com/googleapis/java-spanner/issues/2462) ) ( [f1cbd16](https://github.com/googleapis/java-spanner/commit/f1cbd168a7e5f48206cdfc2d782835cf7ccb8b0d) )
- Make administrative request retries optional ( [\#2476](https://github.com/googleapis/java-spanner/issues/2476) ) ( [ee6548c](https://github.com/googleapis/java-spanner/commit/ee6548cfa511d6efc99f508290ed0b1ce025a4cc) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.11.0 ( [\#2486](https://github.com/googleapis/java-spanner/issues/2486) ) ( [82400d5](https://github.com/googleapis/java-spanner/commit/82400d5576c3ffe08ff6bb94d8b1a307e2f41662) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [6.11.0](https://github.com/googleapis/nodejs-spanner/compare/v6.10.1...v6.11.0) (2023-06-06)

##### Features

- **spanner:** Add DdlStatementActionInfo and add actions to UpdateDatabaseDdlMetadata ( [\#1860](https://github.com/googleapis/nodejs-spanner/issues/1860) ) ( [3e86f36](https://github.com/googleapis/nodejs-spanner/commit/3e86f369b927e3bf0a2046bd13d0b6a39a9bb076) )
- Testing for fgac in pg ( [\#1811](https://github.com/googleapis/nodejs-spanner/issues/1811) ) ( [c48945f](https://github.com/googleapis/nodejs-spanner/commit/c48945f536685d6e4ee4097cfac7d5f57853553e) )

#### [6.12.0](https://github.com/googleapis/nodejs-spanner/compare/v6.11.0...v6.12.0) (2023-06-19)

##### Features

- Databoostenabled for Query and Read partitions ( [\#1784](https://github.com/googleapis/nodejs-spanner/issues/1784) ) ( [66ff70c](https://github.com/googleapis/nodejs-spanner/commit/66ff70cd377d5e3f60a6796bc36bab3a39337f31) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.36.0](https://github.com/googleapis/python-spanner/compare/v3.35.1...v3.36.0) (2023-06-06)

##### Features

- Add DdlStatementActionInfo and add actions to UpdateDatabaseDdlMetadata ( [\#948](https://github.com/googleapis/python-spanner/issues/948) ) ( [1ca6874](https://github.com/googleapis/python-spanner/commit/1ca687464fe65a19370a460556acc0957d693399) )
- Testing for fgac-pg ( [\#902](https://github.com/googleapis/python-spanner/issues/902) ) ( [ad1f527](https://github.com/googleapis/python-spanner/commit/ad1f5277dfb3b6a6c7458ff2ace5f724e56360c1) )

## June 23, 2023

Feature

Spanner Data Boost lets you execute analytics queries and data exports with near-zero impact to existing workloads on your provisioned Spanner instance. This feature is generally available (GA) in the following regions:

- asia-northeast1 (Tokyo)
- asia-south1 (Mumbai)
- us-central1 (Iowa)
- nam3 (North America)
- southamerica-east1 (São Paulo)
- europe-west-1 (Belgium)
- europe-west2 (London)
- europe-west3 (Frankfurt)

For more information, see [Data Boost overview](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) .

## June 22, 2023

Feature

Spanner Agent Platform integration is generally available. You can use Vertex AI with GoogleSQL to enhance your Spanner applications with machine learning capabilities. For more information, see [About Spanner Agent Platform integration](https://docs.cloud.google.com/spanner/docs/ml) .

## June 09, 2023

Feature

In both the GoogleSQL and PostgreSQL dialects, adds support for the `IF NOT EXISTS` clause in `CREATE TABLE` , `CREATE INDEX` , and `ALTER TABLE ADD COLUMN` , along with `IF EXISTS` for `DROP TABLE` and `DROP INDEX` .

## June 07, 2023

Feature

Fine-grained access control is available for PostgreSQL-dialect databases. For more information, see [About fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) .

## June 05, 2023

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.46.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.45.1...spanner/v1.46.0) (2023-05-12)

##### Features

- **spanner/admin/database:** Add support for UpdateDatabase in Cloud Spanner ( [\#7917](https://github.com/googleapis/google-cloud-go/issues/7917) ) ( [83870f5](https://github.com/googleapis/google-cloud-go/commit/83870f55035d6692e22264b209e39e07fe2823b9) )
- **spanner:** Make leader aware routing default enabled for supported RPC requests. ( [\#7912](https://github.com/googleapis/google-cloud-go/issues/7912) ) ( [d0d3755](https://github.com/googleapis/google-cloud-go/commit/d0d37550911f37e09ea9204d0648fb64ff3204ff) )

##### Bug Fixes

- **spanner:** Update grpc to v1.55.0 ( [1147ce0](https://github.com/googleapis/google-cloud-go/commit/1147ce02a990276ca4f8ab7a1ab65c14da4450ef) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.41.0](https://github.com/googleapis/java-spanner/compare/v6.40.1...v6.41.0) (2023-04-28)

##### Features

- Add TransactionExecutionOptions support to executor. ( [\#2396](https://github.com/googleapis/java-spanner/issues/2396) ) ( [8327f21](https://github.com/googleapis/java-spanner/commit/8327f210df86bf681ffed6a78ccc9e8fd899c967) )
- Leader Aware Routing ( [\#2214](https://github.com/googleapis/java-spanner/issues/2214) ) ( [9695ace](https://github.com/googleapis/java-spanner/commit/9695acee9195b50e525d87700e86d701b1d9eed2) )
- Make leak detection configurable for connections ( [\#2405](https://github.com/googleapis/java-spanner/issues/2405) ) ( [85213c8](https://github.com/googleapis/java-spanner/commit/85213c8764fcb7fb12df49baaac9bd00e095f269) )

##### Dependencies

- Update dependency com.google.api.grpc:proto-google-cloud-spanner-executor-v1 to v1.4.0 ( [\#2395](https://github.com/googleapis/java-spanner/issues/2395) ) ( [02dc53c](https://github.com/googleapis/java-spanner/commit/02dc53c097bae3f20d7915fecc9c236c4a5f91f9) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.17.0 ( [\#2406](https://github.com/googleapis/java-spanner/issues/2406) ) ( [d46097f](https://github.com/googleapis/java-spanner/commit/d46097f9f17d9009d211c8c0f16b3e084f8fdbad) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.8.0 ( [\#2400](https://github.com/googleapis/java-spanner/issues/2400) ) ( [b815cb8](https://github.com/googleapis/java-spanner/commit/b815cb88ff29fb5b9a5d7998e765548244f287c1) )
- Update dependency com.google.cloud:google-cloud-trace to v2.16.0 ( [\#2407](https://github.com/googleapis/java-spanner/issues/2407) ) ( [7993be2](https://github.com/googleapis/java-spanner/commit/7993be25e9f380071cded2fa4c2bf630d760a53e) )
- Update dependency org.junit.vintage:junit-vintage-engine to v5.9.3 ( [\#2401](https://github.com/googleapis/java-spanner/issues/2401) ) ( [8aa7a1d](https://github.com/googleapis/java-spanner/commit/8aa7a1dbbf484446ae8eed3cb27d16fc65e6de83) )

#### [6.42.0](https://github.com/googleapis/java-spanner/compare/v6.41.0...v6.42.0) (2023-05-15)

##### Features

- Add support for UpdateDatabase in Cloud Spanner ( [\#2265](https://github.com/googleapis/java-spanner/issues/2265) ) ( [2ea06e7](https://github.com/googleapis/java-spanner/commit/2ea06e70a6f22635bcad7b7e4c79d0cf710dc6dc) )
- Add support for UpdateDatabase in Cloud Spanner ( [\#2429](https://github.com/googleapis/java-spanner/issues/2429) ) ( [09f20bd](https://github.com/googleapis/java-spanner/commit/09f20bd43913a7a01985fd290964d134612c14eb) )

##### Bug Fixes

- Add error details for INTERNAL error ( [\#2413](https://github.com/googleapis/java-spanner/issues/2413) ) ( [ed62aa6](https://github.com/googleapis/java-spanner/commit/ed62aa666ae34cf5e552e19b6b5dc2a8c6609e4e) )
- Use javax.annotation.Nonnull in executor framework ( [\#2414](https://github.com/googleapis/java-spanner/issues/2414) ) ( [afcc598](https://github.com/googleapis/java-spanner/commit/afcc598e05c75610db8d0adacd4da79b4c124122) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.18.0 ( [\#2426](https://github.com/googleapis/java-spanner/issues/2426) ) ( [05a45f8](https://github.com/googleapis/java-spanner/commit/05a45f81c2c71dd236fa36cc987e78a6aa31b594) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.9.0 ( [\#2427](https://github.com/googleapis/java-spanner/issues/2427) ) ( [42dbfe3](https://github.com/googleapis/java-spanner/commit/42dbfe3600b1d482d64c6c4f6865f88db399bae3) )
- Update dependency com.google.cloud:google-cloud-trace to v2.17.0 ( [\#2428](https://github.com/googleapis/java-spanner/issues/2428) ) ( [6f7fee8](https://github.com/googleapis/java-spanner/commit/6f7fee81233811f5bc002f212c8972ffc6afbe16) )
- Update dependency org.graalvm.buildtools:junit-platform-native to v0.9.22 ( [\#2423](https://github.com/googleapis/java-spanner/issues/2423) ) ( [679bb36](https://github.com/googleapis/java-spanner/commit/679bb366162575c28bab1df9b87d01517ea8d5aa) )
- Update dependency org.graalvm.buildtools:native-maven-plugin to v0.9.22 ( [\#2424](https://github.com/googleapis/java-spanner/issues/2424) ) ( [a72f4ff](https://github.com/googleapis/java-spanner/commit/a72f4ff64cce2e9c746e8f6a9e107cbd72afa67f) )
- Update dependency org.graalvm.sdk:graal-sdk to v22.3.2 ( [\#2391](https://github.com/googleapis/java-spanner/issues/2391) ) ( [c082a1f](https://github.com/googleapis/java-spanner/commit/c082a1fccb79cf4c001519eba4a75cef30150541) )

#### [6.42.1](https://github.com/googleapis/java-spanner/compare/v6.42.0...v6.42.1) (2023-05-22)

##### Dependencies

- Update dependency commons-io:commons-io to v2.12.0 ( [\#2439](https://github.com/googleapis/java-spanner/issues/2439) ) ( [d08b226](https://github.com/googleapis/java-spanner/commit/d08b226d5da6272b2de5f66ee1657d03268e396d) )

#### [6.42.2](https://github.com/googleapis/java-spanner/compare/v6.42.1...v6.42.2) (2023-05-30)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.19.0 ( [\#2466](https://github.com/googleapis/java-spanner/issues/2466) ) ( [6de2cf6](https://github.com/googleapis/java-spanner/commit/6de2cf6a2d075b4347d69b9af21ac0cf96413884) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.10.1 ( [\#2465](https://github.com/googleapis/java-spanner/issues/2465) ) ( [0a89f49](https://github.com/googleapis/java-spanner/commit/0a89f49cd55311f4cb84a501aa302eab88b46575) )
- Update dependency com.google.cloud:google-cloud-trace to v2.18.0 ( [\#2467](https://github.com/googleapis/java-spanner/issues/2467) ) ( [45609ed](https://github.com/googleapis/java-spanner/commit/45609ed65e49147077eaaf3eb90ab0c732eef80b) )

#### [6.42.3](https://github.com/googleapis/java-spanner/compare/v6.42.2...v6.42.3) (2023-05-31)

##### Performance Improvements

- Only capture the call stack if the call is actually async ( [\#2471](https://github.com/googleapis/java-spanner/issues/2471) ) ( [ae9c8ad](https://github.com/googleapis/java-spanner/commit/ae9c8add484bc0f7808571cbcffb7b352d6ed739) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [6.9.0](https://github.com/googleapis/nodejs-spanner/compare/v6.8.0...v6.9.0) (2023-04-26)

##### Features

- Leader aware routing ( [\#1783](https://github.com/googleapis/nodejs-spanner/issues/1783) ) ( [0703f41](https://github.com/googleapis/nodejs-spanner/commit/0703f4160c4a0b4c9f9f716174daca110ab8e50f) )

#### [6.10.0](https://github.com/googleapis/nodejs-spanner/compare/v6.9.0...v6.10.0) (2023-05-17)

##### Features

- Add support for UpdateDatabase ( [\#1802](https://github.com/googleapis/nodejs-spanner/issues/1802) ) ( [f4fbe71](https://github.com/googleapis/nodejs-spanner/commit/f4fbe71d819fde9a237f25b03af228b27cf58689) )
- Add support for UpdateDatabase in Cloud Spanner ( [\#1848](https://github.com/googleapis/nodejs-spanner/issues/1848) ) ( [dd9d505](https://github.com/googleapis/nodejs-spanner/commit/dd9d505e1480b9f45f0f4a09b0abca8282d5fceb) )

##### Bug Fixes

- Set grpc useragent ( [\#1847](https://github.com/googleapis/nodejs-spanner/issues/1847) ) ( [021e54e](https://github.com/googleapis/nodejs-spanner/commit/021e54ef469d7d95bae64c687b65489cbfc56cfa) )

#### [6.10.1](https://github.com/googleapis/nodejs-spanner/compare/v6.10.0...v6.10.1) (2023-05-30)

##### Bug Fixes

- Set database admin and instance as having handwritten layers (republish docs) ( [3e3e624](https://github.com/googleapis/nodejs-spanner/commit/3e3e624187013d62a5ff479386fb8961f279b5ca) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.32.0](https://github.com/googleapis/python-spanner/compare/v3.31.0...v3.32.0) (2023-04-25)

##### Features

- Enable instance-level connection ( [\#931](https://github.com/googleapis/python-spanner/issues/931) ) ( [d6963e2](https://github.com/googleapis/python-spanner/commit/d6963e2142d880e94c6f3e9eb27ed1ac310bd1d0) )

#### [3.33.0](https://github.com/googleapis/python-spanner/compare/v3.32.0...v3.33.0) (2023-04-27)

##### Features

- Enable leader aware routing ( [\#899](https://github.com/googleapis/python-spanner/issues/899) ) ( [f9fefad](https://github.com/googleapis/python-spanner/commit/f9fefad6ee2e16804d109d8bfbb613062f57ea65) )

#### [3.34.0](https://github.com/googleapis/python-spanner/compare/v3.33.0...v3.34.0) (2023-05-16)

##### Features

- Add support for UpdateDatabase in Cloud Spanner ( [\#941](https://github.com/googleapis/python-spanner/issues/941) ) ( [38fb890](https://github.com/googleapis/python-spanner/commit/38fb890e34762f104ca97e612e62d4f59e752133) )

##### Bug Fixes

- Upgrade version of sqlparse ( [\#943](https://github.com/googleapis/python-spanner/issues/943) ) ( [df57ce6](https://github.com/googleapis/python-spanner/commit/df57ce6f00b6a992024c9f1bd6948905ae1e5cf4) )

#### [3.35.0](https://github.com/googleapis/python-spanner/compare/v3.34.0...v3.35.0) (2023-05-16)

##### Features

- Add support for updateDatabase in Cloud Spanner ( [\#914](https://github.com/googleapis/python-spanner/issues/914) ) ( [6c7ad29](https://github.com/googleapis/python-spanner/commit/6c7ad2921d2bf886b538f7e24e86397c188620c8) )

#### [3.35.1](https://github.com/googleapis/python-spanner/compare/v3.35.0...v3.35.1) (2023-05-25)

##### Bug Fixes

- Catch rst stream error for all transactions ( [\#934](https://github.com/googleapis/python-spanner/issues/934) ) ( [d317d2e](https://github.com/googleapis/python-spanner/commit/d317d2e1b882d9cf576bfc6c195fa9df7c518c4e) )

## May 24, 2023

Feature

Spanner lets you use a [generated column in the primary key](https://docs.cloud.google.com/spanner/docs/generated-column/how-to#primary-key-generated-column) .

Feature

Spanner database deletion protection is available in Preview. You can enable database deletion protection to prevent the accidental deletion of databases. For more information, see [Prevent accidental database deletion](https://docs.cloud.google.com/spanner/docs/prevent-database-deletion) .

## May 22, 2023

Feature

Spanner automatically increases the degree of parallelism on a query when the instance size allows. For more information on parallel execution of queries, see [Life of a Spanner Query](https://docs.cloud.google.com/spanner/docs/whitepapers/life-of-query#execution) .

## May 09, 2023

Feature

Support for logging the processing duration of your Spanner read and write requests is available in Cloud Audit Logs. For more information, see [Processing duration](https://docs.cloud.google.com/spanner/docs/audit-logging#processing-duration) .

## May 02, 2023

Feature

Spanner supports query capabilities for PostgreSQL dialect databases:

- Set operations (such as `UNION` and `INTERSECT` ) with [`ORDER BY` , `LIMIT` , or `OFFSET`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/query-syntax#select) , or in [subqueries](https://docs.cloud.google.com/spanner/docs/reference/postgresql/subqueries)
- Parameterized `LIMIT` and `OFFSET` operations
- [Statement hints](https://docs.cloud.google.com/spanner/docs/query-optimizer/manage-query-optimizer#statement-hint) for configuring the query optimizer (such as `optimizer_version` and `optimizer_statistics_package` )

Feature

Spanner sampled query plans are available in Preview. You can view samples of historic query plans and compare the performance of a query over time. For more information, see [Sampled query plans](https://docs.cloud.google.com/spanner/docs/query-execution-plans#sampled-plans) .

## April 28, 2023

Feature

The number of indexes per table that Spanner supports increased from 32 to 128. For more information, see [Quotas & limits](https://docs.cloud.google.com/spanner/quotas#indexes) .

Feature

The following [multi-region instance configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-multi-region) are available in North America: `nam14` (Northern Virginia/Montréal/South Carolina) and `nam15` (Dallas/Northern Virginia/Iowa).

## April 24, 2023

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.45.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.44.0...spanner/v1.45.0) (2023-04-10)

##### Features

- **spanner/spansql:** Add support for missing DDL syntax for ALTER CHANGE STREAM ( [\#7429](https://github.com/googleapis/google-cloud-go/issues/7429) ) ( [d34fe02](https://github.com/googleapis/google-cloud-go/commit/d34fe02cfa31520f88dedbd41bbc887e8faa857f) )
- **spanner/spansql:** Support fine-grained access control DDL syntax ( [\#6691](https://github.com/googleapis/google-cloud-go/issues/6691) ) ( [a7edf6b](https://github.com/googleapis/google-cloud-go/commit/a7edf6b5c62d02b7d5199fc83d435f6a37a8eac5) )
- **spanner/spansql:** Support grant/revoke view, change stream, table function ( [\#7533](https://github.com/googleapis/google-cloud-go/issues/7533) ) ( [9c61215](https://github.com/googleapis/google-cloud-go/commit/9c612159647d540e694ec9e84cab5cdd1c94d2b8) )
- **spanner:** Add x-goog-spanner-route-to-leader header to Spanner RPC contexts for RW/PDML transactions. ( [\#7500](https://github.com/googleapis/google-cloud-go/issues/7500) ) ( [fcab05f](https://github.com/googleapis/google-cloud-go/commit/fcab05faa5026896af76b762eed5b7b6b2e7ee07) )
- **spanner:** Add new fields for Serverless analytics ( [69067f8](https://github.com/googleapis/google-cloud-go/commit/69067f8c0075099a84dd9d40e438711881710784) )
- **spanner:** Enable custom decoding for list value ( [\#7463](https://github.com/googleapis/google-cloud-go/issues/7463) ) ( [3aeadcd](https://github.com/googleapis/google-cloud-go/commit/3aeadcd97eaf2707c2f6e288c8b72ef29f49a185) )
- **spanner:** Update iam and longrunning deps ( [91a1f78](https://github.com/googleapis/google-cloud-go/commit/91a1f784a109da70f63b96414bba8a9b4254cddd) )

##### Bug Fixes

- **spanner/spansql:** Fix SQL for CREATE CHANGE STREAM TableName; case ( [\#7514](https://github.com/googleapis/google-cloud-go/issues/7514) ) ( [fc5fd86](https://github.com/googleapis/google-cloud-go/commit/fc5fd8652771aeca73e7a28ee68134155a5a9499) )
- **spanner:** Correcting the proto field Id for field data_boost_enabled ( [00fff3a](https://github.com/googleapis/google-cloud-go/commit/00fff3a58bed31274ab39af575876dab91d708c9) )

#### [1.45.1](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.45.0...spanner/v1.45.1) (2023-04-21)

##### Bug Fixes

- **spanner/spannertest:** Send transaction id in result metadata ( [\#7809](https://github.com/googleapis/google-cloud-go/issues/7809) ) ( [e3bbd5f](https://github.com/googleapis/google-cloud-go/commit/e3bbd5f10b3922ab2eb50cb39daccd7bc1891892) )
- **spanner:** Context timeout should be wrapped correctly ( [\#7744](https://github.com/googleapis/google-cloud-go/issues/7744) ) ( [f8e22f6](https://github.com/googleapis/google-cloud-go/commit/f8e22f6cbba10fc262e87b4d06d5c1289d877503) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.38.1](https://github.com/googleapis/java-spanner/compare/v6.38.0...v6.38.1) (2023-03-29)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.6.0 ( [\#2352](https://github.com/googleapis/java-spanner/issues/2352) ) ( [19175ce](https://github.com/googleapis/java-spanner/commit/19175ce22777ac68f8c825a438c0a2503234aa42) )

#### [6.38.2](https://github.com/googleapis/java-spanner/compare/v6.38.1...v6.38.2) (2023-04-01)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.15.0 ( [\#2356](https://github.com/googleapis/java-spanner/issues/2356) ) ( [e4c001a](https://github.com/googleapis/java-spanner/commit/e4c001a2a78af756213fb28e01c571721e105262) )
- Update dependency com.google.cloud:google-cloud-trace to v2.14.0 ( [\#2357](https://github.com/googleapis/java-spanner/issues/2357) ) ( [dbb8e66](https://github.com/googleapis/java-spanner/commit/dbb8e669d855c08f48c15c9eafec03a85fa08bca) )

#### [6.39.0](https://github.com/googleapis/java-spanner/compare/v6.38.2...v6.39.0) (2023-04-11)

##### Features

- Capture stack trace for session checkout is now optional ( [\#2350](https://github.com/googleapis/java-spanner/issues/2350) ) ( [6b6427a](https://github.com/googleapis/java-spanner/commit/6b6427a25af25fde944dfc1dd4bf6a6463682caf) )

#### [6.40.0](https://github.com/googleapis/java-spanner/compare/v6.39.0...v6.40.0) (2023-04-14)

##### Features

- Savepoints ( [\#2278](https://github.com/googleapis/java-spanner/issues/2278) ) ( [b02f584](https://github.com/googleapis/java-spanner/commit/b02f58435b97346cc8e08a96635affe8383981bb) )

##### Performance Improvements

- Remove custom transport executor ( [\#2366](https://github.com/googleapis/java-spanner/issues/2366) ) ( [e27dbe5](https://github.com/googleapis/java-spanner/commit/e27dbe5f58229dab208eeeed44d53e741700c814) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.7.0 ( [\#2377](https://github.com/googleapis/java-spanner/issues/2377) ) ( [40402af](https://github.com/googleapis/java-spanner/commit/40402af54f94f16619d018e252181db29ae6855e) )
- Update dependency org.graalvm.buildtools:junit-platform-native to v0.9.21 ( [\#2379](https://github.com/googleapis/java-spanner/issues/2379) ) ( [ae7262d](https://github.com/googleapis/java-spanner/commit/ae7262d37391c0ec2fee1dcbb24899e4fa16ae17) )
- Update dependency org.graalvm.buildtools:native-maven-plugin to v0.9.21 ( [\#2380](https://github.com/googleapis/java-spanner/issues/2380) ) ( [0cb159e](https://github.com/googleapis/java-spanner/commit/0cb159efc97f02b42f064244e3812a0fd3d82db6) )

#### [6.40.1](https://github.com/googleapis/java-spanner/compare/v6.40.0...v6.40.1) (2023-04-17)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.16.0 ( [\#2383](https://github.com/googleapis/java-spanner/issues/2383) ) ( [5d5c33a](https://github.com/googleapis/java-spanner/commit/5d5c33ae7c01e10112c72777f202187a50b55ac3) )
- Update dependency com.google.cloud:google-cloud-trace to v2.15.0 ( [\#2384](https://github.com/googleapis/java-spanner/issues/2384) ) ( [6b4ce1f](https://github.com/googleapis/java-spanner/commit/6b4ce1fc7ffd837fab6250e36269589d95f5b8c6) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [6.8.0](https://github.com/googleapis/nodejs-spanner/compare/v6.7.2...v6.8.0) (2023-04-06)

##### Features

- Add new fields for Serverless analytics ( [\#1816](https://github.com/googleapis/nodejs-spanner/issues/1816) ) ( [2a6ca6f](https://github.com/googleapis/nodejs-spanner/commit/2a6ca6f09215752f9451d625ac02837e9d70b66a) )

##### Bug Fixes

- Begin transaction foes not handle error ( [\#1833](https://github.com/googleapis/nodejs-spanner/issues/1833) ) ( [6ecd366](https://github.com/googleapis/nodejs-spanner/commit/6ecd366da7183d502c710cb5c879984c276b12db) )
- Correct the proto field Id for field data_boost_enabled ( [\#1827](https://github.com/googleapis/nodejs-spanner/issues/1827) ) ( [7f6d4cc](https://github.com/googleapis/nodejs-spanner/commit/7f6d4ccce9269197312f2d795ef854e1789e8fce) )
- Logic for retrying specifiied internal errors ( [\#1822](https://github.com/googleapis/nodejs-spanner/issues/1822) ) ( [f915bd1](https://github.com/googleapis/nodejs-spanner/commit/f915bd16cf7e817243e46a319b3e6f270b24bf68) ), closes [\#1808](https://github.com/googleapis/nodejs-spanner/issues/1808)

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.30.0](https://github.com/googleapis/python-spanner/compare/v3.29.0...v3.30.0) (2023-03-28)

##### Features

- Pass custom Client object to dbapi ( [\#911](https://github.com/googleapis/python-spanner/issues/911) ) ( [52b1a0a](https://github.com/googleapis/python-spanner/commit/52b1a0af0103a5b91aa5bf9ea1138319bdb90d79) )

#### [3.31.0](https://github.com/googleapis/python-spanner/compare/v3.30.0...v3.31.0) (2023-04-12)

##### Features

- Add databoost enabled property for batch transactions ( [\#892](https://github.com/googleapis/python-spanner/issues/892) ) ( [ffb3915](https://github.com/googleapis/python-spanner/commit/ffb39158be5a551b698739c003ee6125a11c1c7a) )

##### Bug Fixes

- Set databoost false ( [\#928](https://github.com/googleapis/python-spanner/issues/928) ) ( [c9ed9d2](https://github.com/googleapis/python-spanner/commit/c9ed9d24d19594dfff57c979fa3bf68d84bbc3b5) )

## April 10, 2023

Feature

Spanner integration with Data Catalog is available in Preview in the `europe-central2` region.

For more information, see [Manage resources using Data Catalog](https://docs.cloud.google.com/spanner/docs/dc-integration) .

## March 31, 2023

Feature

Spanner integration with Data Catalog is available in Preview. Data Catalog is a fully managed, scalable metadata management service within Knowledge Catalog. It catalogs metadata about Spanner instances, databases, tables, columns, and views. For Preview, integration with Data Catalog isn't available in the `europe-central2` region.

For more information, see [Manage resources using Data Catalog](https://docs.cloud.google.com/spanner/docs/dc-integration) .

## March 30, 2023

Feature

You can create Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) in Doha, Qatar ( `me-central1` ).

## March 27, 2023

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.37.0](https://github.com/googleapis/java-spanner/compare/v6.36.1...v6.37.0) (2023-03-03)

##### Features

- Add new fields for Serverless analytics ( [\#2315](https://github.com/googleapis/java-spanner/issues/2315) ) ( [ce9cd74](https://github.com/googleapis/java-spanner/commit/ce9cd7469e2fed15711a8dffe944934cdaa45ce8) )

##### Bug Fixes

- Update test certificate name. ( [\#2300](https://github.com/googleapis/java-spanner/issues/2300) ) ( [18e76d6](https://github.com/googleapis/java-spanner/commit/18e76d6636c530c9cfc0ac872d72e321e75c990e) )

##### Dependencies

- Update dependency com.google.api.grpc:proto-google-cloud-spanner-executor-v1 to v1.3.0 ( [\#2306](https://github.com/googleapis/java-spanner/issues/2306) ) ( [8372250](https://github.com/googleapis/java-spanner/commit/8372250e0aaae68b0d610d59c1ee88c4dc0d9e8b) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.13.0 ( [\#2311](https://github.com/googleapis/java-spanner/issues/2311) ) ( [6ba613b](https://github.com/googleapis/java-spanner/commit/6ba613b44598e48699aca320683e65572a730fc7) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.4.0 ( [\#2312](https://github.com/googleapis/java-spanner/issues/2312) ) ( [266c49c](https://github.com/googleapis/java-spanner/commit/266c49cc58beaa935a328599a3e75d3b1fb4988d) )
- Update dependency com.google.cloud:google-cloud-trace to v2.12.0 ( [\#2313](https://github.com/googleapis/java-spanner/issues/2313) ) ( [e5f76c6](https://github.com/googleapis/java-spanner/commit/e5f76c6598887b616d371b4d0b3551e236e080f8) )
- Update dependency org.json:json to v20230227 ( [\#2310](https://github.com/googleapis/java-spanner/issues/2310) ) ( [badcc14](https://github.com/googleapis/java-spanner/commit/badcc14182244929042412f97e5a7e05799eea22) )

#### [6.38.0](https://github.com/googleapis/java-spanner/compare/v6.37.0...v6.38.0) (2023-03-20)

##### Features

- Add option to wait on session pool creation ( [\#2329](https://github.com/googleapis/java-spanner/issues/2329) ) ( [ff17244](https://github.com/googleapis/java-spanner/commit/ff17244ee918fa17c96488a0f7081728cda7b342) )
- Add PartitionedUpdate support to executor ( [\#2228](https://github.com/googleapis/java-spanner/issues/2228) ) ( [2c8ecf6](https://github.com/googleapis/java-spanner/commit/2c8ecf6fee591df95ee4abfa230c3fcf0c34c589) )

##### Bug Fixes

- Correct the proto field Id for field data_boost_enabled ( [\#2328](https://github.com/googleapis/java-spanner/issues/2328) ) ( [6159d7e](https://github.com/googleapis/java-spanner/commit/6159d7ec49b17f6bc40e1b8c93d1e64198c59dcf) )
- Update executeCloudBatchDmlUpdates. ( [\#2326](https://github.com/googleapis/java-spanner/issues/2326) ) ( [27ef53c](https://github.com/googleapis/java-spanner/commit/27ef53c8447bd51a56fdfe6b2b206afe234fad80) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.14.0 ( [\#2333](https://github.com/googleapis/java-spanner/issues/2333) ) ( [9c81109](https://github.com/googleapis/java-spanner/commit/9c81109e452d6bae2598cf6cf541a09423a8ed6e) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.5.0 ( [\#2335](https://github.com/googleapis/java-spanner/issues/2335) ) ( [5eac2be](https://github.com/googleapis/java-spanner/commit/5eac2beb2ce5eebb61e70428e2ac2e11593fc986) )
- Update dependency com.google.cloud:google-cloud-trace to v2.13.0 ( [\#2334](https://github.com/googleapis/java-spanner/issues/2334) ) ( [c461ba0](https://github.com/googleapis/java-spanner/commit/c461ba0b1a145cc3e9bee805ec6ad827376e5168) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.28.0](https://github.com/googleapis/python-spanner/compare/v3.27.1...v3.28.0) (2023-02-28)

##### Features

- Enable "rest" transport in Python for services supporting numeric enums ( [\#897](https://github.com/googleapis/python-spanner/issues/897) ) ( [c21a0d5](https://github.com/googleapis/python-spanner/commit/c21a0d5f4600818ca79cd4e199a2245683c33467) )

#### [3.29.0](https://github.com/googleapis/python-spanner/compare/v3.28.0...v3.29.0) (2023-03-23)

##### Features

- Add new fields for Serverless analytics ( [\#906](https://github.com/googleapis/python-spanner/issues/906) ) ( [2a5a636](https://github.com/googleapis/python-spanner/commit/2a5a636fc296ad0a7f86ace6a5f361db1e2ee26d) )

##### Bug Fixes

- Correct the proto field ID for field data_boost_enabled ( [\#915](https://github.com/googleapis/python-spanner/issues/915) ) ( [428aa1e](https://github.com/googleapis/python-spanner/commit/428aa1e5e4458649033a5566dc3017d2fadbd2a0) )

##### Documentation

- Fix formatting of request arg in docstring ( [\#918](https://github.com/googleapis/python-spanner/issues/918) ) ( [c022bf8](https://github.com/googleapis/python-spanner/commit/c022bf859a3ace60c0a9ddb86896bc83f85e327f) )

## March 23, 2023

Feature

You can create Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) in Turin, Italy ( `europe-west12` ).

## March 21, 2023

Feature

The following functions and expressions have been added to the GoogleSQL dialect:

- [`ARRAY_FILTER` function](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/array_functions#array_filter)
- [`ARRAY_TRANSFORM` function](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/array_functions#array_transform)
- [Lambda expressions](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/functions-reference#lambdas)

## March 20, 2023

Feature

You can use Google Cloud tags to group and organize your Spanner instances, and to condition Identity and Access Management (IAM) policies based on whether an instance has a specific tag. For more information, see [Control access and organize instances with tags](https://docs.cloud.google.com/spanner/docs/tags) .

## March 17, 2023

Feature

The following functions have been added to the GoogleSQL dialect:

- [`ARRAY_INCLUDES_ALL` function](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/array_functions#array_includes_all)
- [`ARRAY_INCLUDES_ANY` function](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/array_functions#array_includes_any)
- [`ARRAY_MIN` function](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/array_functions#array_min)
- [`ARRAY_MAX` function](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/array_functions#array_max)

Feature

Support for the GoogleSQL-dialect `THEN RETURN` clause and the PostgreSQL-dialect `RETURNING` clause is generally available. For more information, see [`THEN RETURN`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-and-then-return) and [`RETURNING`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/dml-syntax#insert-returning) .

## March 09, 2023

Feature

Spanner fine-grained access control is generally available. Fine-grained access control combines the benefits of Identity and Access Management (IAM) with traditional SQL role-based access control. For more information, see [About fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) .

## March 03, 2023

Feature

Added support for the `JSONB` array data type in the PostgreSQL dialect. For more information, see [Work with JSONB data](https://docs.cloud.google.com/spanner/docs/working-with-jsonb) .

## March 01, 2023

Feature

Change streams are supported for PostgreSQL-dialect databases.

## February 27, 2023

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.36.0](https://github.com/googleapis/java-spanner/compare/v6.35.2...v6.36.0) (2023-02-08)

##### Features

- Support UNRECOGNIZED types + decode BYTES columns lazily ( [\#2219](https://github.com/googleapis/java-spanner/issues/2219) ) ( [fc721c4](https://github.com/googleapis/java-spanner/commit/fc721c4d30de6ed9e5bc4fbbe0e1e7b79a5c7490) )

##### Bug Fixes

- **java:** Skip fixing poms for special modules ( [\#1744](https://github.com/googleapis/java-spanner/issues/1744) ) ( [\#2244](https://github.com/googleapis/java-spanner/issues/2244) ) ( [e7f4b40](https://github.com/googleapis/java-spanner/commit/e7f4b4016f8c4c7e4fac0b822f5af2cffd181134) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.11.0 ( [\#2262](https://github.com/googleapis/java-spanner/issues/2262) ) ( [d566613](https://github.com/googleapis/java-spanner/commit/d566613442217bdfc69caea7242464fba2647519) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.2.0 ( [\#2264](https://github.com/googleapis/java-spanner/issues/2264) ) ( [b5fdbc0](https://github.com/googleapis/java-spanner/commit/b5fdbc0accdaaf1f63c62c1837d72bb378dc8f43) )
- Update dependency com.google.cloud:google-cloud-trace to v2.10.0 ( [\#2263](https://github.com/googleapis/java-spanner/issues/2263) ) ( [96f0c81](https://github.com/googleapis/java-spanner/commit/96f0c8181aeb8ca75647a783d8b163f371ad937e) )

#### [6.36.1](https://github.com/googleapis/java-spanner/compare/v6.36.0...v6.36.1) (2023-02-21)

##### Bug Fixes

- Prevent illegal negative timeout values into thread sleep() method while retrying exceptions in unit tests. ( [\#2268](https://github.com/googleapis/java-spanner/issues/2268) ) ( [ce66098](https://github.com/googleapis/java-spanner/commit/ce66098c7139ea13d5ea91cf6fbceb5c732b392d) )

##### Dependencies

- Update dependency com.google.api.grpc:proto-google-cloud-spanner-executor-v1 to v1.2.0 ( [\#2256](https://github.com/googleapis/java-spanner/issues/2256) ) ( [f0ca86a](https://github.com/googleapis/java-spanner/commit/f0ca86a0858bde84cc38f1ad8fae5f3c4f4f3395) )
- Update dependency com.google.cloud:google-cloud-monitoring to v3.12.0 ( [\#2284](https://github.com/googleapis/java-spanner/issues/2284) ) ( [0be701a](https://github.com/googleapis/java-spanner/commit/0be701a8b59277f2cfb990a88e4f1dafcbafdd97) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.3.0 ( [\#2285](https://github.com/googleapis/java-spanner/issues/2285) ) ( [bb5d5c6](https://github.com/googleapis/java-spanner/commit/bb5d5c66e78812b943a85e0fd888e7021c11bde1) )
- Update dependency com.google.cloud:google-cloud-trace to v2.11.0 ( [\#2286](https://github.com/googleapis/java-spanner/issues/2286) ) ( [3c80932](https://github.com/googleapis/java-spanner/commit/3c80932d577de0ea108e695d0a4e542fbfc01deb) )
- Update dependency org.graalvm.buildtools:junit-platform-native to v0.9.20 ( [\#2280](https://github.com/googleapis/java-spanner/issues/2280) ) ( [685d1ea](https://github.com/googleapis/java-spanner/commit/685d1ea1c3bf59cd71093a68c260276c605d835f) )
- Update dependency org.graalvm.buildtools:native-maven-plugin to v0.9.20 ( [\#2281](https://github.com/googleapis/java-spanner/issues/2281) ) ( [f2aabc2](https://github.com/googleapis/java-spanner/commit/f2aabc24770d1b9c505dfc96b39fe81c6a0ad5a5) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [6.7.1](https://github.com/googleapis/nodejs-spanner/compare/v6.7.0...v6.7.1) (2023-01-23)

##### Bug Fixes

- Change of tag for fgac ( [\#1780](https://github.com/googleapis/nodejs-spanner/issues/1780) ) ( [d75b6dd](https://github.com/googleapis/nodejs-spanner/commit/d75b6dd79ffc2442cbd7a14f1ea952edc6678a64) )
- **codec:** Use index to determine array struct member value ( [\#1775](https://github.com/googleapis/nodejs-spanner/issues/1775) ) ( [fc2b695](https://github.com/googleapis/nodejs-spanner/commit/fc2b695d9ea6b65df856b4b081a75165009413ee) ), closes [\#1774](https://github.com/googleapis/nodejs-spanner/issues/1774)

#### [6.7.2](https://github.com/googleapis/nodejs-spanner/compare/v6.7.1...v6.7.2) (2023-02-17)

##### Bug Fixes

- Tests emit empty metadata before emitting unspecified error ( [14ef031](https://github.com/googleapis/nodejs-spanner/commit/14ef0318db756e7debad8599b1e274b8877291e1) )

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.44.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.43.0...spanner/v1.44.0) (2023-02-01)

##### Features

- **spanner/spansql:** Add support for ALTER INDEX statement ( [\#7287](https://github.com/googleapis/google-cloud-go/issues/7287) ) ( [fbe1bd4](https://github.com/googleapis/google-cloud-go/commit/fbe1bd4d0806302a48ff4a5822867757893a5f2d) )
- **spanner/spansql:** Add support for managing the optimizer statistics package ( [\#7283](https://github.com/googleapis/google-cloud-go/issues/7283) ) ( [e528221](https://github.com/googleapis/google-cloud-go/commit/e52822139e2821a11873c2d6af85a5fea07700e8) )
- **spanner:** Add support for Optimistic Concurrency Control ( [\#7332](https://github.com/googleapis/google-cloud-go/issues/7332) ) ( [48ba16f](https://github.com/googleapis/google-cloud-go/commit/48ba16f3a09893a3527a22838ad1e9ff829da15b) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.27.1](https://github.com/googleapis/python-spanner/compare/v3.27.0...v3.27.1) (2023-01-30)

##### Bug Fixes

- Add context manager return types ( [830f325](https://github.com/googleapis/python-spanner/commit/830f325c4ab9ab1eb8d53edca723d000c23ee0d7) )
- Change fgac database role tags ( [\#888](https://github.com/googleapis/python-spanner/issues/888) ) ( [ae92f0d](https://github.com/googleapis/python-spanner/commit/ae92f0dd8a78f2397977354525b4be4b2b02aec3) )
- Fix for database name in batch create request ( [\#883](https://github.com/googleapis/python-spanner/issues/883) ) ( [5e50beb](https://github.com/googleapis/python-spanner/commit/5e50bebdd1d43994b3d83568641d1dff1c419cc8) )

##### Documentation

- Add documentation for enums ( [830f325](https://github.com/googleapis/python-spanner/commit/830f325c4ab9ab1eb8d53edca723d000c23ee0d7) )

Feature

The system insights dashboard displays metrics and scorecards for the resources that your instance or database uses and helps you get a high-level view of your system's performance. For more information, see [Monitor instances with system insights](https://docs.cloud.google.com/spanner/docs/monitoring-console) .

## February 16, 2023

Change

The Spanner regional endpoints feature has been moved to a future release. It is not currently available.

## February 13, 2023

Announcement

As of today, the list compute price for the following 9-replica Spanner multi-region configurations has been reduced: `nam-eur-asia1` and `nam-eur-asia3` . For more details, see [Spanner pricing](https://cloud.google.com/spanner/pricing) .

## February 09, 2023

Feature

The Google Cloud console for Spanner displays the status and progress of copy backup long-running operations that you have initiated in the Google Cloud console. The operation is visible for 7 days.

## February 07, 2023

Feature

Spanner autocompletes and validates the syntax of your DDL statements when you use the Google Cloud console to write DDL statements for your PostgreSQL-dialect databases.

## February 06, 2023

Feature

Spanner supports regional endpoints. You can use regional endpoints if your data location must be restricted and controlled to comply with regulatory requirements.

## January 31, 2023

Feature

Table sizes statistics are generally available. Table size statistics help you get insights into the size of individual tables in your database. For more information, see [Table sizes statistics](https://docs.cloud.google.com/spanner/docs/introspection/table-sizes-statistics) .

## January 30, 2023

Libraries

A monthly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [6.7.0](https://github.com/googleapis/nodejs-spanner/compare/v6.6.0...v6.7.0) (2023-01-17)

##### Features

- Added SuggestConversationSummary RPC ( [\#1744](https://github.com/googleapis/nodejs-spanner/issues/1744) ) ( [14346f3](https://github.com/googleapis/nodejs-spanner/commit/14346f3cf8ed0cb0a93c255dc520dc62887c0e1a) )

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.43.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.42.0...spanner/v1.43.0) (2023-01-19)

##### Features

- **spanner/spansql:** Add support for change stream value_capture_type option ( [\#7201](https://github.com/googleapis/google-cloud-go/issues/7201) ) ( [27b3398](https://github.com/googleapis/google-cloud-go/commit/27b33988f078779c2d641f776a11b2095a5ccc51) )
- **spanner/spansql:** Support `default_leader` database option ( [\#7187](https://github.com/googleapis/google-cloud-go/issues/7187) ) ( [88adaa2](https://github.com/googleapis/google-cloud-go/commit/88adaa216832467560c19e61528b5ce5f1e5ff76) )
- **spanner:** Add REST client ( [06a54a1](https://github.com/googleapis/google-cloud-go/commit/06a54a16a5866cce966547c51e203b9e09a25bc0) )
- **spanner:** Inline begin transaction for ReadWriteTransactions ( [\#7149](https://github.com/googleapis/google-cloud-go/issues/7149) ) ( [2ce3606](https://github.com/googleapis/google-cloud-go/commit/2ce360644439a386aeaad7df5f47541667bd621b) )

##### Bug Fixes

- **spanner:** Fix integration tests data race ( [\#7229](https://github.com/googleapis/google-cloud-go/issues/7229) ) ( [a741024](https://github.com/googleapis/google-cloud-go/commit/a741024abd6fb1f073831503c2717b2a44226a59) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.35.0](https://github.com/googleapis/java-spanner/compare/v6.34.1...v6.35.0) (2023-01-12)

##### Features

- Add support for new cloud client test framework in google-cloud-spanner-executor ( [\#2217](https://github.com/googleapis/java-spanner/issues/2217) ) ( [d75ebc1](https://github.com/googleapis/java-spanner/commit/d75ebc1387de7ba0e0a32dfcdd564392d43ff555) )
- **spanner:** Add samples for fine grained access control ( [\#2172](https://github.com/googleapis/java-spanner/issues/2172) ) ( [77969e3](https://github.com/googleapis/java-spanner/commit/77969e35feee4dee3460fcdc45227e9a9d924d74) )

##### Bug Fixes

- Retry on RST_STREAM internal error ( [\#2111](https://github.com/googleapis/java-spanner/issues/2111) ) ( [d5372e6](https://github.com/googleapis/java-spanner/commit/d5372e662624831abc694d81acecf797d32d86e3) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.8.0 ( [\#2192](https://github.com/googleapis/java-spanner/issues/2192) ) ( [fe7e755](https://github.com/googleapis/java-spanner/commit/fe7e755a798b584bf79d16d1f419b1ca7f957172) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.1.1 ( [\#2222](https://github.com/googleapis/java-spanner/issues/2222) ) ( [7d3bcca](https://github.com/googleapis/java-spanner/commit/7d3bcca4e5846d823106f724fef42d2ef3a1c822) )
- Update dependency com.google.cloud:google-cloud-trace to v2.7.0 ( [\#2193](https://github.com/googleapis/java-spanner/issues/2193) ) ( [da2b924](https://github.com/googleapis/java-spanner/commit/da2b924e037dd366d171c481c6db799de7cacc22) )
- Update dependency org.graalvm.buildtools:junit-platform-native to v0.9.19 ( [\#2180](https://github.com/googleapis/java-spanner/issues/2180) ) ( [43b54e9](https://github.com/googleapis/java-spanner/commit/43b54e92b4df3ec6474b8ba7fef61b5b613e6ab0) )
- Update dependency org.graalvm.buildtools:native-maven-plugin to v0.9.19 ( [\#2181](https://github.com/googleapis/java-spanner/issues/2181) ) ( [b42eb38](https://github.com/googleapis/java-spanner/commit/b42eb3866e1fd74f9a9ad2a9dc3d100ac0893f38) )

#### [6.35.1](https://github.com/googleapis/java-spanner/compare/v6.35.0...v6.35.1) (2023-01-18)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.9.0 ( [\#2230](https://github.com/googleapis/java-spanner/issues/2230) ) ( [717f70f](https://github.com/googleapis/java-spanner/commit/717f70f76f915e15a7283b32a83a6f4ac64fc931) )
- Update dependency com.google.cloud:google-cloud-trace to v2.8.0 ( [\#2231](https://github.com/googleapis/java-spanner/issues/2231) ) ( [557ea16](https://github.com/googleapis/java-spanner/commit/557ea164ebf948cd78f937c6996fd21e9618d3ae) )
- Update dependency org.graalvm.sdk:graal-sdk to v22.3.1 ( [\#2238](https://github.com/googleapis/java-spanner/issues/2238) ) ( [d5f5237](https://github.com/googleapis/java-spanner/commit/d5f52375394ef617f4fcb823937a374930f941e7) )
- Update dependency org.junit.vintage:junit-vintage-engine to v5.9.2 ( [\#2223](https://github.com/googleapis/java-spanner/issues/2223) ) ( [3278f91](https://github.com/googleapis/java-spanner/commit/3278f9167b1b2688ed090a7dfd5874e88b8945a5) )

#### [6.35.2](https://github.com/googleapis/java-spanner/compare/v6.35.1...v6.35.2) (2023-01-24)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.10.0 ( [\#2249](https://github.com/googleapis/java-spanner/issues/2249) ) ( [d18780e](https://github.com/googleapis/java-spanner/commit/d18780ec0278fc49495939647fe6a2f9e0b4f94e) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.1.2 ( [\#2246](https://github.com/googleapis/java-spanner/issues/2246) ) ( [1adaf7c](https://github.com/googleapis/java-spanner/commit/1adaf7cae629ba7b9903d6512adc7b13b6d1208e) )
- Update dependency com.google.cloud:google-cloud-trace to v2.9.0 ( [\#2250](https://github.com/googleapis/java-spanner/issues/2250) ) ( [3cd5ab0](https://github.com/googleapis/java-spanner/commit/3cd5ab05e1fd24090fd58c2320b6875135e49b69) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.27.0](https://github.com/googleapis/python-spanner/compare/v3.26.0...v3.27.0) (2023-01-10)

##### Features

- Add support for python 3.11 ( [\#879](https://github.com/googleapis/python-spanner/issues/879) ) ( [4b8c2cf](https://github.com/googleapis/python-spanner/commit/4b8c2cf6c30892ad977e3db6c3a147a93af649e6) )
- Add typing to proto.Message based class attributes ( [4683d10](https://github.com/googleapis/python-spanner/commit/4683d10c75e24aa222591d6001e07aacb6b4ee46) )

##### Bug Fixes

- Add dict typing for client_options ( [4683d10](https://github.com/googleapis/python-spanner/commit/4683d10c75e24aa222591d6001e07aacb6b4ee46) )
- **deps:** Require google-api-core \>=1.34.0, \>=2.11.0 ( [4683d10](https://github.com/googleapis/python-spanner/commit/4683d10c75e24aa222591d6001e07aacb6b4ee46) )
- Drop packaging dependency ( [4683d10](https://github.com/googleapis/python-spanner/commit/4683d10c75e24aa222591d6001e07aacb6b4ee46) )
- Drop usage of pkg_resources ( [4683d10](https://github.com/googleapis/python-spanner/commit/4683d10c75e24aa222591d6001e07aacb6b4ee46) )
- Fix timeout default values ( [4683d10](https://github.com/googleapis/python-spanner/commit/4683d10c75e24aa222591d6001e07aacb6b4ee46) )

##### Documentation

- **samples:** Snippetgen handling of repeated enum field ( [4683d10](https://github.com/googleapis/python-spanner/commit/4683d10c75e24aa222591d6001e07aacb6b4ee46) )
- **samples:** Snippetgen should call await on the operation coroutine before calling result ( [4683d10](https://github.com/googleapis/python-spanner/commit/4683d10c75e24aa222591d6001e07aacb6b4ee46) )

## December 22, 2022

Feature

The Spanner Kafka connector publishes change streams records to Kafka for application integration and event triggering. For more information, see [Build change streams connections to Kafka](https://docs.cloud.google.com/spanner/docs/change-streams/use-kafka) .

## December 19, 2022

Feature

You can use the `ALTER INDEX` statement to add columns into an index or drop non-key columns. For more information, see [Alter an index](https://docs.cloud.google.com/spanner/docs/secondary-indexes#alter_index) .

## December 14, 2022

Feature

Spanner offers the [Spanner change streams to Pub/Sub Dataflow template](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided-streaming#cloud-spanner-change-streams-to-pubsub) , which streams Spanner data change records and writes them into Pub/Sub topics.

Feature

You can create a custom instance configuration and add optional read-only replicas to your custom instance configurations to scale reads and support low latency stale reads. For more information, see [Regional and multi-region configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations#configuration) .

## December 12, 2022

Feature

Support for moving a Spanner instance is generally available. You can request to move your Spanner instance from any instance configuration to any other instance configuration, including between regional and multi-region configurations. For more information, see [Move an instance](https://docs.cloud.google.com/spanner/docs/move-instance) .

Feature

An update to Spanner change streams provides two new data capture types for change records:

- `NEW_VALUES` mode captures only new values in non-key columns, and no old values. Keys are always captured.
- `NEW_ROW` mode captures the full new row, including columns that aren't included in updates. No old values are captured.

Note that existing change streams remain set to `OLD_AND_NEW_VALUES` .

## December 06, 2022

Announcement

We identified an issue in how we calculate the Total Database Storage metric in multi-regional Spanner instances. This metric is used to calculate the charges for Spanner database storage.

Database storage is currently incorrectly reported lower than it actually is in multi-regional configurations, resulting in undercharging for database storage. We communicated a Service Announcement in October and started rolling out this change to pricing on December 1, 2022. Depending on your configuration, your Total Database Storage could increase by up to 25%.

For the majority of impacted customers, the impact on your total bill will be less than 0.5%. For those affected, you will notice an increase in database storage charges that reflect this corrected metric.

We waived the under-billed amount for all past billing cycles. Please note that this issue only affects multi-region configurations of Spanner. It doesn't affect regional configurations of Spanner. Additionally, the Total Backup Storage metric isn't affected by this issue, and has always been reported correctly.

For more information, see [Database storage prices](https://cloud.google.com/spanner/pricing#database_storage) .

## December 05, 2022

Feature

New SQL syntax, [`RETURNING`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/dml-syntax#insert-returning) in the PostgreSQL dialect and [`THEN RETURN`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-and-then-return) in Google Standard SQL, selects and returns data from rows that were just updated as part of a `DML` statement. This is especially useful for getting values from default or generated columns and can reduce latency over equivalent multi-statement transactions. The preview supports the Java, JDBC, Python, and Go Spanner clients as well as PostgreSQL drivers that connect through [PGAdapter](https://github.com/GoogleCloudPlatform/pgadapter#google-cloud-spanner-pgadapter) .

## December 02, 2022

Feature

The number of concurrent database restore operations per instance that Spanner supports has increased from five to ten. For more information, see [Backup and restore limits](https://docs.cloud.google.com/spanner/quotas#backup-limits) .

## November 15, 2022

Feature

Added support for the `JSONB` data type in the Spanner PostgreSQL dialect. For more information, see [Work with JSONB data](https://docs.cloud.google.com/spanner/docs/working-with-jsonb) .

Feature

[Time to live (TTL)](https://docs.cloud.google.com/spanner/docs/ttl) is supported in PostgreSQL-dialect databases. With TTL, you can reduce storage costs, improve query performance, and simplify data retention by automatically removing unneeded data based on user-defined policies.

## November 08, 2022

Feature

Spanner supports cross-region and cross-project backup use cases. You can [copy a backup](https://docs.cloud.google.com/spanner/docs/backup/copy-backup) of your database from one instance to another instance in a different region or project to provide additional data protection and compliance capabilities.

## November 07, 2022

Libraries

A weekly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.40.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.39.0...spanner/v1.40.0) (2022-11-03)

##### Features

- Expose logger in ClientConfig ( [\#6958](https://github.com/googleapis/google-cloud-go/issues/6958) ) ( [bd85442](https://github.com/googleapis/google-cloud-go/commit/bd85442bc6fb8c18d1a7c6d73850d220c3973c46) ), refs [\#6957](https://github.com/googleapis/google-cloud-go/issues/6957)
- Update `result_set.proto` to return undeclared parameters in ExecuteSql API ( [de4e16a](https://github.com/googleapis/google-cloud-go/commit/de4e16a498354ea7271f5b396f7cb2bb430052aa) )
- Update `transaction.proto` to include different lock modes ( [caf4afa](https://github.com/googleapis/google-cloud-go/commit/caf4afa139ad7b38b6df3e3b17b8357c81e1fd6c) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.4.4-sp.7](https://github.com/googleapis/java-spanner/compare/v6.4.4-sp.6...v6.4.4-sp.7) (2022-11-02)

##### Dependencies

- Regenerate with new protobuf (6.4.4-sp) ( [\#2135](https://github.com/googleapis/java-spanner/issues/2135) ) ( [ebeac9a](https://github.com/googleapis/java-spanner/commit/ebeac9ae14c8ea6321d7b0b497532b0895661581) )

## November 03, 2022

Feature

Support for the NHibernate ORM is generally available, letting you use Spanner as a backend database for the NHibernate framework. For more information, see [NHibernate Dialect for Spanner](https://cloud.google.com/blog/topics/developers-practitioners/nhibernate-dialect-cloud-spanner-now-generally-available) .

## October 31, 2022

Libraries

A weekly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.32.0](https://github.com/googleapis/java-spanner/compare/v6.31.2...v6.32.0) (2022-10-27)

##### Features

- Enable client to server compression ( [\#2117](https://github.com/googleapis/java-spanner/issues/2117) ) ( [50f8425](https://github.com/googleapis/java-spanner/commit/50f8425fe9e1db16ed060337d26feccc9a9813e2) )
- Increase default number of channels when gRPC channel pool is enabled ( [\#1997](https://github.com/googleapis/java-spanner/issues/1997) ) ( [44f27fc](https://github.com/googleapis/java-spanner/commit/44f27fc90fa3f9f4914574fb0476e971da4c02ff) )
- Update `result_set.proto` to return undeclared parameters in ExecuteSql API ( [\#2101](https://github.com/googleapis/java-spanner/issues/2101) ) ( [826eb93](https://github.com/googleapis/java-spanner/commit/826eb9305095db064f52a15dc502bc0e0df9a984) )

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.6.0 ( [\#2125](https://github.com/googleapis/java-spanner/issues/2125) ) ( [7d86fe4](https://github.com/googleapis/java-spanner/commit/7d86fe40de29311ad65bd382e55f75326d16c4e3) )
- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.0.5 ( [\#2122](https://github.com/googleapis/java-spanner/issues/2122) ) ( [308a65c](https://github.com/googleapis/java-spanner/commit/308a65c3e07e33f82b7ce474e0e95099192bb593) )
- Update dependency com.google.cloud:google-cloud-trace to v2.5.0 ( [\#2126](https://github.com/googleapis/java-spanner/issues/2126) ) ( [5167928](https://github.com/googleapis/java-spanner/commit/516792809cf976aeab10709ca62503b7f03bb333) )
- Update dependency org.graalvm.buildtools:junit-platform-native to v0.9.16 ( [\#2119](https://github.com/googleapis/java-spanner/issues/2119) ) ( [b2d27e8](https://github.com/googleapis/java-spanner/commit/b2d27e8f841cab096d5ccad64a250c7f0b35f670) )
- Update dependency org.graalvm.buildtools:native-maven-plugin to v0.9.16 ( [\#2120](https://github.com/googleapis/java-spanner/issues/2120) ) ( [151cf77](https://github.com/googleapis/java-spanner/commit/151cf778ff76edaee9e849181f72119ffa6cb897) )
- Update dependency org.graalvm.sdk:graal-sdk to v22.3.0 ( [\#2116](https://github.com/googleapis/java-spanner/issues/2116) ) ( [9d6930b](https://github.com/googleapis/java-spanner/commit/9d6930b77ec479e5f517236852244476c23dc5c8) )

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [6.4.0](https://github.com/googleapis/nodejs-spanner/compare/v6.3.0...v6.4.0) (2022-10-27)

##### Features

- Update `result_set.proto` to return undeclared parameters in ExecuteSql API ( [eaa445e](https://github.com/googleapis/nodejs-spanner/commit/eaa445ed314190abefc17e3672bb5e200142618b) )
- Update `transaction.proto` to include different lock modes ( [\#1723](https://github.com/googleapis/nodejs-spanner/issues/1723) ) ( [eaa445e](https://github.com/googleapis/nodejs-spanner/commit/eaa445ed314190abefc17e3672bb5e200142618b) )

## October 24, 2022

Libraries

A weekly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [6.3.0](https://github.com/googleapis/nodejs-spanner/compare/v6.2.0...v6.3.0) (2022-10-03)

##### Bug Fixes

- **deps:** Update dependency @google-cloud/precise-date to v3 ( [\#1676](https://github.com/googleapis/nodejs-spanner/issues/1676) ) ( [3f20ec4](https://github.com/googleapis/nodejs-spanner/commit/3f20ec47bbf89e1f72546a8ebf41a8b4ba93832f) )
- Don't import the whole google-gax from proto JS ( [\#1553](https://github.com/googleapis/nodejs-spanner/issues/1553) ) ( [\#1700](https://github.com/googleapis/nodejs-spanner/issues/1700) ) ( [f9c2640](https://github.com/googleapis/nodejs-spanner/commit/f9c2640e054659a2e8299b8f989fa7936d04b0d7) )
- Update google-gax to v3.3.0 ( [f9c2640](https://github.com/googleapis/nodejs-spanner/commit/f9c2640e054659a2e8299b8f989fa7936d04b0d7) )

## October 19, 2022

Feature

The number of concurrent database restore operations per instance that Spanner supports has increased from one to five. For more information, see [Backup and restore limits](https://docs.cloud.google.com/spanner/quotas#backup-limits) .

## October 18, 2022

Feature

The following generally available features help you identify and troubleshoot high latencies in specific databases:

- The [Lock insights dashboard](https://docs.cloud.google.com/spanner/docs/use-lock-and-transaction-insights#lock-insights) helps you identify latency spikes that are due to lock contentions.

- The [Transaction insights dashboard](https://docs.cloud.google.com/spanner/docs/use-lock-and-transaction-insights#txn-insights) helps you identify the transactions that cause lock contentions and, possibly, high latencies.

## October 17, 2022

Libraries

A weekly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.23.4](https://github.com/googleapis/java-spanner/compare/v6.23.3...v6.23.4) (2022-10-12)

##### Dependencies

- Regenerate with new protobuf 6.23.x ( [\#2103](https://github.com/googleapis/java-spanner/issues/2103) ) ( [22ee992](https://github.com/googleapis/java-spanner/commit/22ee992c00ce88db95972b003ec605c590efc5a7) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.22.2](https://github.com/googleapis/python-spanner/compare/v3.22.1...v3.22.2) (2022-10-10)

##### Bug Fixes

- **deps:** Allow protobuf 3.19.5 ( [\#839](https://github.com/googleapis/python-spanner/issues/839) ) ( [06725fc](https://github.com/googleapis/python-spanner/commit/06725fcf7fb216ad0cffb2cb568f8da38243c32e) )

## October 11, 2022

Feature

Spanner Vertex AI integration is available in public preview. You can enhance your Spanner applications with machine learning capabilities by using Google Standard SQL. For more information, see [About Spanner Vertex AI integration](https://docs.cloud.google.com/spanner/docs/ml) .

## October 10, 2022

Libraries

A weekly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.31.2](https://github.com/googleapis/java-spanner/compare/v6.31.1...v6.31.2) (2022-10-05)

##### Bug Fixes

- Update protobuf to v3.21.7 ( [ac71008](https://github.com/googleapis/java-spanner/commit/ac71008bf8b1244cb3c5cf4317a0d25d4ffc5bbd) )

#### [6.31.1](https://github.com/googleapis/java-spanner/compare/v6.31.0...v6.31.1) (2022-10-03)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-shared-dependencies to v3.0.4 ( [\#2090](https://github.com/googleapis/java-spanner/issues/2090) ) ( [8f46938](https://github.com/googleapis/java-spanner/commit/8f46938b67e44a7b739dc156dc8a0a89bcb33ef0) )
- Update dependency org.graalvm.buildtools:native-maven-plugin to v0.9.14 ( [\#2031](https://github.com/googleapis/java-spanner/issues/2031) ) ( [c5e9ba1](https://github.com/googleapis/java-spanner/commit/c5e9ba1c1a47faf89c47a9146a97cb6711dce242) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.22.1](https://github.com/googleapis/python-spanner/compare/v3.22.0...v3.22.1) (2022-10-04)

##### Bug Fixes

- **deps:** Require protobuf \>= 3.20.2 ( [\#830](https://github.com/googleapis/python-spanner/issues/830) ) ( [4d71563](https://github.com/googleapis/python-spanner/commit/4d7156376f4633de6c1a2bfd25ba97126386ebd0) )

##### Documentation

- **samples:** Add samples for CMMR phase 2 ( [4282340](https://github.com/googleapis/python-spanner/commit/4282340bc2c3a34496c59c33f5c64ff76dceda4c) )

## October 03, 2022

Libraries

A weekly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.31.0](https://github.com/googleapis/java-spanner/compare/v6.30.2...v6.31.0) (2022-09-29)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-trace to v2.3.4 ( [\#2027](https://github.com/googleapis/java-spanner/issues/2027) ) ( [14890ed](https://github.com/googleapis/java-spanner/commit/14890ed8e0df99eba7c2521a196132c78054b6ed) )
- Update dependency com.google.cloud:google-cloud-trace to v2.3.5 ( [\#2083](https://github.com/googleapis/java-spanner/issues/2083) ) ( [cef4e0a](https://github.com/googleapis/java-spanner/commit/cef4e0ada98ab65020f32836fc0c8ab1ee0c7eed) )
- Update dependency org.graalvm.buildtools:junit-platform-native to v0.9.14 ( [\#2030](https://github.com/googleapis/java-spanner/issues/2030) ) ( [04b59ff](https://github.com/googleapis/java-spanner/commit/04b59ff8a1efaa32082aa4e9567d90b5956810c6) )
- Update dependency org.json:json to v20220924 ( [\#2035](https://github.com/googleapis/java-spanner/issues/2035) ) ( [a26a14a](https://github.com/googleapis/java-spanner/commit/a26a14a94ac3ca6cd7eabce6826cce3dde27ea66) )

### Python

### Changes for [google-cloud-spanner](https://github.com/googleapis/python-spanner)

#### [3.22.0](https://github.com/googleapis/python-spanner/compare/v3.21.0...v3.22.0) (2022-09-26)

##### Features

- Add reason, domain, metadata & details fields in Custom Exceptions to provide additional error info ( [\#804](https://github.com/googleapis/python-spanner/issues/804) ) ( [2a74060](https://github.com/googleapis/python-spanner/commit/2a740607a00cb622ac9ce4005c12afd52114b4a5) )

## September 28, 2022

Feature

The following `SPANNER_SYS` statistical tables have been enhanced with new columns:

- [Transaction statistics](https://cloud.google.com/spanner/docs/introspection/transaction-statistics) : `TOTAL_LATENCY_DISTRIBUTION` , `OPERATIONS_BY_TABLE` , and `ATTEMPT_COUNT` .
- [Query statistics](https://cloud.google.com/spanner/docs/introspection/query-statistics) : `LATENCY_DISTRIBUTION` and `RUN_IN_RW_TRANSACTION_EXECUTION_COUNT` .
- [Read statistics](https://cloud.google.com/spanner/docs/introspection/read-statistics) : `RUN_IN_RW_TRANSACTION_EXECUTION_COUNT` .

## September 27, 2022

Change

The number of mutations per commit that Spanner supports has increased from 20,000 to 40,000. For more information, see [Quotas and limits](https://docs.cloud.google.com/spanner/quotas#limits_for_creating_reading_updating_and_deleting_data) .

Feature

The [`ARRAY_SLICE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/array_functions#array_slice) function is available to use in Google Standard SQL. This function returns an `ARRAY` containing zero or more consecutive elements from an input array.

## September 26, 2022

Libraries

A weekly digest of client library updates from across the [Google Cloud SDK](https://docs.cloud.google.com/sdk) .

### Node.js

### Changes for [@google-cloud/spanner](https://github.com/googleapis/nodejs-spanner)

#### [6.2.0](https://github.com/googleapis/nodejs-spanner/compare/v6.1.4...v6.2.0) (2022-09-16)

##### Bug Fixes

- Allow passing gax instance to client constructor ( [\#1698](https://github.com/googleapis/nodejs-spanner/issues/1698) ) ( [588c1a2](https://github.com/googleapis/nodejs-spanner/commit/588c1a2e0c449cfcb86cac73da32dd5794ee2baa) )
- **deps:** Use grpc-gcp v1.0.0 ( [\#1710](https://github.com/googleapis/nodejs-spanner/issues/1710) ) ( [12eab9d](https://github.com/googleapis/nodejs-spanner/commit/12eab9d628b72b5a7fc88f3d5e932b7a4d70dce2) )
- Move runtime dependencies from dev dependencies to dependencies ( [\#1704](https://github.com/googleapis/nodejs-spanner/issues/1704) ) ( [b2c1c0f](https://github.com/googleapis/nodejs-spanner/commit/b2c1c0f93653af6cc7bd9893ca14394f2a631b68) )
- Preserve default values in x-goog-request-params header ( [\#1711](https://github.com/googleapis/nodejs-spanner/issues/1711) ) ( [f1ae513](https://github.com/googleapis/nodejs-spanner/commit/f1ae51301d4ea9b0ed1ad4d4762c249fef9f8d08) )

### Go

### Changes for [spanner/admin/database/apiv1](https://github.com/googleapis/google-cloud-go/tree/main/spanner/admin/database/apiv1)

#### [1.39.0](https://github.com/googleapis/google-cloud-go/compare/spanner/v1.38.0...spanner/v1.39.0) (2022-09-21)

##### Features

- **spanner/spannersql:** Add backticks when name contains a hypen ( [\#6621](https://github.com/googleapis/google-cloud-go/issues/6621) ) ( [e88ca66](https://github.com/googleapis/google-cloud-go/commit/e88ca66ca950e15d9011322dbfca3c88ccceb0ec) )
- **spanner/spansql:** Add support for create, alter, and drop change streams ( [\#6669](https://github.com/googleapis/google-cloud-go/issues/6669) ) ( [cc4620a](https://github.com/googleapis/google-cloud-go/commit/cc4620a5ee3a9129a4cdd48d90d4060ba0bbcd58) )
- **spanner:** Retry spanner transactions and mutations when RST_STREAM error is returned ( [\#6699](https://github.com/googleapis/google-cloud-go/issues/6699) ) ( [1b56cd0](https://github.com/googleapis/google-cloud-go/commit/1b56cd0ec31bc32362259fc722907e092bae081a) )

##### Bug Fixes

- **spanner:** Destroy session when client is closing ( [\#6700](https://github.com/googleapis/google-cloud-go/issues/6700) ) ( [a1ce541](https://github.com/googleapis/google-cloud-go/commit/a1ce5410f1e0f4d68dae0ddc790518e9978faf0c) )
- **spanner:** Spanner sessions will be cleaned up from the backend ( [\#6679](https://github.com/googleapis/google-cloud-go/issues/6679) ) ( [c27097e](https://github.com/googleapis/google-cloud-go/commit/c27097e236abeb8439a67ad9b716d05c001aea2e) )

### Java

### Changes for [google-cloud-spanner](https://github.com/googleapis/java-spanner)

#### [6.30.2](https://github.com/googleapis/java-spanner/compare/v6.30.1...v6.30.2) (2022-09-21)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.4.5 ( [\#2022](https://github.com/googleapis/java-spanner/issues/2022) ) ( [0536962](https://github.com/googleapis/java-spanner/commit/0536962df9af3feed237f758a560c24fafd81d60) )
- Update dependency org.junit.vintage:junit-vintage-engine to v5.9.1 ( [\#2023](https://github.com/googleapis/java-spanner/issues/2023) ) ( [3fb4235](https://github.com/googleapis/java-spanner/commit/3fb423571c1128b7cafdc6596d5366268d74f0e4) )

#### [6.30.1](https://github.com/googleapis/java-spanner/compare/v6.30.0...v6.30.1) (2022-09-20)

##### Dependencies

- Update dependency com.google.cloud:google-cloud-monitoring to v3.4.4 ( [\#2014](https://github.com/googleapis/java-spanner/issues/2014) ) ( [9cebad4](https://github.com/googleapis/java-spanner/commit/9cebad485afc8b8d94bd4bc1673542a330451fbd) )
- Update dependency com.google.cloud:google-cloud-trace to v2.3.3 ( [\#2004](https://github.com/googleapis/java-spanner/issues/2004) ) ( [54f9095](https://github.com/googleapis/java-spanner/commit/54f90957544f0798d9872956dbe40ce822d5167d) )

## September 15, 2022

Feature

Fine-grained access control for Spanner is available in public preview. fine-grained access control control lets you secure your Spanner databases at the table and column level by using new RDBMS-style roles and `GRANT` / `REVOKE` SQL statements. With fine-grained access control, you can protect your transactional data and ensure that the right controls are in place when granting access to data. For more information, see [About fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) .

Feature

The Spanner Golang database/sql driver is generally available. Add the driver to your application to enable the use of the database/sql package with Spanner. For more information, see the [Spanner blog](https://cloud.google.com/blog/topics/developers-practitioners/golangs-databasesql-driver-support-cloud-spanner-now-generally-available) and the [package documentation](https://pkg.go.dev/github.com/googleapis/go-sql-spanner) .

## September 13, 2022

Feature

You can create Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) in Tel Aviv, Israel ( `me-west1` ).

## September 09, 2022

Feature

[Query Optimizer version 5](https://docs.cloud.google.com/spanner/docs/query-optimizer/versions) is generally available, and is the default optimizer version.

## September 08, 2022

Feature

Spanner free trial instances are generally available. With a free trial instance, you can learn and explore Spanner for 90 days at no cost. You can create Google Standard SQL or PostgreSQL-dialect databases and store up to 10 GB of data, with the option to upgrade at any time. For more information, see [About Spanner free trial instances](https://docs.cloud.google.com/spanner/docs/free-trial-instance) .

## August 17, 2022

Feature

The [`DISABLE_INLINE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/functions-reference#function_hints) hint is available to use in a Google Standard SQL function call. This allows a function to be computed once instead of each time another part of a query references it.

## July 25, 2022

Feature

[Query Optimizer version 5](https://docs.cloud.google.com/spanner/docs/query-optimizer/versions) is generally available. Version 4 remains the default optimizer version in production.

## July 14, 2022

Feature

You can view aggregated Spanner statistics related to [transactions](https://docs.cloud.google.com/spanner/docs/introspection/transaction-statistics#transaction-stats-total) , [reads](https://docs.cloud.google.com/spanner/docs/introspection/read-statistics#read-stats-total) , [queries](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics#query-stats-total) , and [lock contentions](https://docs.cloud.google.com/spanner/docs/introspection/lock-statistics#lock-stats-total) in GA in [Cloud Monitoring](https://docs.cloud.google.com/monitoring/docs) .

## June 30, 2022

Feature

The [`ANALYZE` DDL command](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#analyze-statistics) lets administrators to manually update the [query statistics package](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview#statistics-packages) that the optimizer uses to build query execution plans. This complements the existing automatic updates to provide faster feedback cycles when data, queries, or indexes change frequently.

## June 29, 2022

Feature

Query insights is [generally available](https://cloud.google.com/products#product-launch-stages) . Query insights helps you visually detect and identify query performance issues for Spanner databases. You can also dig deeper and analyse the query details to know the root cause of these issues.

To learn more, see [Detect query performance issues with Query insights](https://docs.cloud.google.com/spanner/docs/using-query-insights) .

## June 23, 2022

Feature

The PostgreSQL interface is generally available, making the capabilities of Spanner accessible from the PostgreSQL ecosystem. It includes a core subset of the PostgreSQL SQL dialect, support for the `psql` command-line tool, native language clients, and integration into existing Google tools. For more information, see [PostgreSQL interface](https://docs.cloud.google.com/spanner/docs/postgresql-interface) .

## June 10, 2022

Feature

Commit timestamps let [a Spanner optimization that can reduce query I/O](https://docs.cloud.google.com/spanner/docs/sql-best-practices#commit-timestamps) when retrieving data written after a particular time.

## June 07, 2022

Feature

You can create Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) in Dallas ( `us-south1` ).

## May 31, 2022

Feature

Granular instance sizing is generally available. You can create production instances of fewer than 1000 processing units. To learn more, see [Compute capacity, nodes and processing units](https://cloud.google.com/spanner/docs/compute-capacity) .

## May 27, 2022

Feature

Spanner [change streams](https://docs.cloud.google.com/spanner/docs/change-streams) capture and stream out inserts, updates, and deletes in near real-time—useful for analytics, archiving, and triggering downstream application workflows.

## May 24, 2022

Feature

You can create Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) in Columbus ( `us-east5` ).

## May 10, 2022

Feature

You can create Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) in Madrid ( `europe-southwest1` ).

## May 03, 2022

Feature

You can create Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-regional) in Paris ( `europe-west9` ).

Feature

[Query Optimizer version 4](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview#version-history) is generally available, and is the default optimizer version.

## April 20, 2022

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instance-configurations#regional_configurations) can be created in Milan ( `europe-west8` ).

## April 14, 2022

Feature

You can define a default value for a non-key table column when creating or altering a table. Using the `DEFAULT` keyword, a schema author can provide a fallback for a column when an insert statement or mutation doesn't explicitly specify a value.

Feature

A new three-continent, nine-replica [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instance-configurations#configs-multi-region) is available for Spanner: `nam-eur-asia3` (Iowa/South Carolina/Belgium/Netherlands/Taiwan/Oklahoma).

## April 05, 2022

Feature

Spanner allows you to [export a subset of your database tables](https://docs.cloud.google.com/spanner/docs/export#exporting_a_subset_of_tables) to Google Cloud Storage as Avro files.

## March 25, 2022

Feature

All instances with a compute capacity of at least one node (1,000 processing units) have [a data storage allotment of 4 TB per node](https://docs.cloud.google.com/spanner/docs/compute-capacity#data_storage_limits) , an increase from 2 TB per node. Relatedly, instances smaller than one node have a data storage allotment of 409.6 GB for every 100 processing units.

## March 22, 2022

Breaking

The data type of the `COLUMN_DEFAULT` column in [the information schema `COLUMNS` table](https://docs.cloud.google.com/spanner/docs/information-schema#information_schemacolumns) has changed from `BYTES` to `STRING` . This aligns better with industry standards and lets you make future improvements to Spanner.

## March 10, 2022

Feature

Spanner offers [committed use discounts](https://docs.cloud.google.com/spanner/docs/cuds) . You can get significantly discounted prices in exchange for your commitment to use Spanner compute resources continuously for a year or longer.

## March 08, 2022

Feature

You can see and manage the [views](https://docs.cloud.google.com/spanner/docs/views) of your Spanner databases from the Google Cloud console. To do so, visit a database's **Overview** page, and then click the **Views** tab.

## March 03, 2022

Feature

You can view aggregated Spanner statistics related to [transactions](https://docs.cloud.google.com/spanner/docs/introspection/transaction-statistics#transaction-stats-total) , [reads](https://docs.cloud.google.com/spanner/docs/introspection/read-statistics#read-stats-total) , [queries](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics#query-stats-total) , and [lock contentions](https://docs.cloud.google.com/spanner/docs/introspection/lock-statistics#lock-stats-total) in Preview in [Cloud Monitoring](https://docs.cloud.google.com/monitoring/docs) . Additionally, the retention period for these metrics at one-minute intervals has been increased from [six hours to six weeks](https://docs.cloud.google.com/monitoring/quotas#data_retention_policy) .

## March 01, 2022

Feature

Released [Query Optimizer version 4](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview#version-history) . Version 3 remains the default optimizer version in production.

Feature

The following [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instance-configurations#configs-multi-region) is available in North America - `nam13` (Iowa/Oklahoma/Salt Lake City).

## February 11, 2022

Feature

Spanner [optimizes the way it processes groups of similar statements in DML batches](https://docs.cloud.google.com/spanner/docs/dml-best-practices#batch-dml) , significantly improving the speed at which it performs batched data writes under certain conditions.

## February 08, 2022

Feature

[Query statistics](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics) cover DML statements, including inserts, updates, and deletes.

## February 07, 2022

Feature

[CPU Utilization metrics](https://docs.cloud.google.com/monitoring/api/metrics_gcp#gcp-spanner) in Spanner provide grouping by all task priorities: low, medium, and high.

Also, the monitoring console in Spanner lets you [view the CPU utilization of your instance by operation type](https://docs.cloud.google.com/spanner/docs/introspection/investigate-cpu-utilization) , filtered by task priority.

## January 25, 2022

Breaking

Starting no sooner than February 23, 2022, the data type of the `COLUMN_DEFAULT` column in [the information schema's `COLUMNS` table](https://docs.cloud.google.com/spanner/docs/information-schema#information_schemacolumns) will change from `BYTES` to `STRING` . This aligns better with industry standards, and lets you make future improvements to Spanner.

## November 16, 2021

Announcement

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instance-configurations#regional_configurations) can be created in Santiago ( `southamerica-west1` ).

## November 04, 2021

Feature

[Time to live (TTL)](https://docs.cloud.google.com/spanner/docs/ttl) is generally available. TTL reduces storage costs, improves query performance, and simplifies data retention by automatically removing unneeded data based on user-defined policies.

## October 29, 2021

Feature

The [`django-spanner` plugin](https://github.com/googleapis/python-spanner-django) is available, letting you use Spanner as a backend database for the Django Web framework. For more information, see [Django ORM with Spanner](https://docs.cloud.google.com/spanner/docs/django-orm) .

## October 27, 2021

Feature

When performing a CSV export, you can use [the `spannerSnapshotTime` option](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided-batch#cloud-spanner-to-cloud-storage-text) to export a specific past version of your data.

## October 13, 2021

Feature

You can assign [`request tags` and `transaction tags`](https://docs.cloud.google.com/spanner/docs/introspection/troubleshooting-with-tags) in your application code to easily troubleshoot query performance, transaction latency, and lock contentions by correlating introspection statistics to application code.

## October 12, 2021

Feature

The [PostgreSQL interface](https://docs.cloud.google.com/spanner/docs/postgresql-interface) is available in [Preview](https://cloud.google.com/products#preview) , making the capabilities of Spanner accessible from the PostgreSQL ecosystem. The release supports a subset of the PostgreSQL SQL dialect, including core data types, functions, and operators. Applications can connect using updated Spanner drivers for JDBC, Java, Go, and Python. Starting initially with `psql` , community tools can connect using `PGAdapter` , a sidecar proxy that implements the PostgreSQL wire protocol. [Sign up](https://goo.gle/PostgreSQL-interface) for the preview today.

## October 06, 2021

Feature

You can [specify the statistics package](https://cloud.google.com/spanner/docs/query-optimizer/manage-query-optimizer#list-statistics-packages) for the query optimizer to use, to ensure predictability in your query plans.

## September 24, 2021

Feature

[Query Optimizer version 3](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview#version-history) is the default optimizer version in production.

## August 31, 2021

Feature

The [R2DBC driver](https://docs.cloud.google.com/spanner/docs/use-oss-r2dbc) for Spanner is available in [Preview](https://cloud.google.com/products#product-launch-stages) . This driver lets you connect to Spanner from fully reactive applications.

## August 30, 2021

Feature

Added support for the JSON data type. For more information, see [Working with JSON data](https://docs.cloud.google.com/spanner/docs/working-with-json) .

Feature

In the Google Cloud console, a database's **Query** page supports multiple query tabs so you no longer have to clear one query to create and run another. Additionally, you can enter multiple query and DML statements in a single query tab. When you do so, the **Results** and **Explanation** subtabs let you choose which statement's results or query plan you want to view. See [A tour of the query editor](https://docs.cloud.google.com/spanner/docs/tune-query-with-visualizer#a-tour-of-the-query-editor) for details.

Feature

Added support for [changing the leader region location](https://docs.cloud.google.com/spanner/docs/modifying-leader-region) of a Spanner database.

## August 20, 2021

Feature

Views are supported in Spanner databases. Use views to provide logical data-modeling to applications, to centralize query definitions and simplify maintenance, and to ensure stability of query definitions across schema changes. [Learn more](https://docs.cloud.google.com/spanner/docs/views) .

Change

Spanner creates dedicated backup jobs to take backups instead of using an instance's server resources. As a result, backup time is reduced and backup operations do not affect instance performance.

## August 17, 2021

Feature

Released [Query Optimizer version 3](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview#version-history) . Query Optimizer v3 is currently set to **off** by default in production.

## August 04, 2021

Announcement

Spanner has an [end-to-end latency guide](https://docs.cloud.google.com/spanner/docs/latency-guide) . This guide describes the high-level components involved in a Cloud Spanner API request, and explains how to extract, capture, and visualize latencies associated with these components to know the source of the latencies.

## August 03, 2021

Feature

Added support for [changing instance configuration](https://docs.cloud.google.com/spanner/docs/instance-configurations#moving_an_instance_to_a_different_configuration) (Preview).

Announcement

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instance-configurations#regional_configurations) can be created in Toronto ( `northamerica-northeast2` ).

## July 21, 2021

Feature

[Time to live](https://docs.cloud.google.com/spanner/docs/ttl) (TTL) is available in public preview. This feature lets database administrators periodically delete unneeded data from Spanner tables, and so decrease storage and backup costs and potentially increase query performance. To use this feature, a database owner defines a [row deletion policy](https://docs.cloud.google.com/spanner/docs/ttl#defining_a_row_deletion_policy) on a table schema.

## July 20, 2021

Feature

Granular instance sizing is available in public preview. Historically, the most granular unit for provisioning compute capacity on Spanner has been the node. To provide more granular control, we are introducing Processing Units (PUs); one Spanner node is equal to 1,000 PUs. You can provision in batches of 100 PUs, and get a proportionate amount of compute and storage resources. [Learn more](https://docs.cloud.google.com/spanner/docs/compute-capacity) .

## July 19, 2021

Feature

[Key Visualizer for Spanner](https://docs.cloud.google.com/spanner/docs/key-visualizer) is available. Key Visualizer is an interactive monitoring tool to analyze usage patterns in Spanner databases. It reveals trends and outliers in important performance and resource metrics.

## July 08, 2021

Feature

The `NUMERIC` data type is [supported as a valid key column type](https://docs.cloud.google.com/spanner/docs/data-types) , so you can use `NUMERIC` type columns when specifying primary keys, foreign keys, and secondary indexes.

## June 30, 2021

Feature

Spanner supports Cloud External Key Manager (Cloud EKM) when using [customer-managed encryption keys](https://docs.cloud.google.com/spanner/docs/cmek) . Cloud EKM also provides [Key Access Justification](https://docs.cloud.google.com/assured-workloads/key-access-justifications/docs/overview) to give you more visibility into key access requests.

## June 29, 2021

Announcement

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Delhi ( `asia-south2` ).

## June 21, 2021

Announcement

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Melbourne ( `australia-southeast2` ).

## June 15, 2021

Feature

The SQL mathematical functions [`EXP`](https://docs.cloud.google.com/spanner/docs/mathematical_functions#exp) , [`LN`](https://docs.cloud.google.com/spanner/docs/mathematical_functions#ln) , [`LOG`](https://docs.cloud.google.com/spanner/docs/mathematical_functions#log) , [`LOG10`](https://docs.cloud.google.com/spanner/docs/mathematical_functions#log10) and [`SQRT`](https://docs.cloud.google.com/spanner/docs/mathematical_functions#sqrt) support `NUMERIC` data as input. You no longer need to cast `NUMERIC` data to `FLOAT64` data before passing it as input to these functions.

## June 11, 2021

Announcement

You can find common queries for monitoring and troubleshooting on the Query page in the Google Cloud console. This page has query templates to help you to access these introspection system tables: Query Stats, Read Stats, Transaction Stats, Lock Stats, and Oldest active queries.

## June 04, 2021

Change

We are replacing the **Insert a row** and **Edit a row** data forms in the Google Cloud console with pre-populated DML query templates on the **Query** page. These templates provide you more flexibility when adding and editing data. [Learn More](https://docs.cloud.google.com/spanner/docs/quickstart-console#insert_and_modify_data)

## May 27, 2021

Announcement

We've enhanced the experience for creating, updating, and deleting schemas in the Google Cloud console. On a database's Overview page you'll now find a Write DDL link to the DDL editor where you can perform all these activities.

## April 13, 2021

Feature

[Transaction statistics](https://docs.cloud.google.com/spanner/docs/introspection/transaction-statistics) includes information about commit retries to help users debug performance issues caused by transaction aborts.

## April 06, 2021

Feature

You can track the progress of long-running index backfill operations through the gcloud command line tool, [REST API](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/UpdateDatabaseDdlMetadata) , and [RPC API](https://docs.cloud.google.com/spanner/docs/reference/rpc/google.spanner.admin.database.v1#google.spanner.admin.database.v1.UpdateDatabaseDdlMetadata) . For more information, see [Checking the progress of a secondary index backfill](https://docs.cloud.google.com/spanner/docs/secondary-indexes#index-progress) .

## March 31, 2021

Feature

You can use Customer-Managed Encryption Keys (CMEK) to protect databases in Spanner. CMEK in Spanner is generally available. For more information, see [CMEK](https://docs.cloud.google.com/spanner/docs/cmek) .

Change

The maximum number of JOIN statements allowed in a query has increased from 15 to 20. [Query limits](https://docs.cloud.google.com/spanner/quotas#query_limits) .

Feature

You can optionally specify the priority of data requests. For more information, see [CPU utilization and task priority](https://docs.cloud.google.com/spanner/docs/cpu-utilization#task-priority) .

## March 24, 2021

Announcement

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Warsaw ( `europe-central2` ).

## March 17, 2021

Feature

The Google Cloud console query page has been updated with a revamped query editor, which offers improved autocomplete, prevalidation of your query, formatting options, and the ability to run a selection from your query. This update also includes a query plan visualizer. For a tour of these features and to learn more, see [Tuning a query using the query plan visualizer](https://docs.cloud.google.com/spanner/docs/tune-query-with-visualizer) .

## March 11, 2021

Feature

Spanner provides a metric, **CPU Utilization by operation types** , which breaks down CPU usage by user-initiated operations. For more information, see [CPU utilization metrics](https://docs.cloud.google.com/spanner/docs/cpu-utilization#metrics) .

## March 03, 2021

Feature

Spanner supports [point-in-time recovery (PITR)](https://docs.cloud.google.com/spanner/docs/pitr) , which lets you recover data from a specific point in time in the past.

## March 01, 2021

Feature

You can optionally receive the mutation count for a transaction in the commit response to optimize the transactions while staying within the mutation count limit. For more information, see [Retrieving commit statistics for a transaction](https://docs.cloud.google.com/spanner/docs/commit-statistics) .

## February 08, 2021

Feature

The Spanner Studio displays database storage utilization and warns you if you are approaching the recommended limit. For more information, see [storage utilization metrics](https://docs.cloud.google.com/spanner/docs/storage-utilization) .

## January 15, 2021

Feature

You can run SQL queries to retrieve [lock statistics](https://docs.cloud.google.com/spanner/docs/introspection/lock-statistics) to investigate lock conflicts in your database.

## January 14, 2021

Feature

[Query statistics](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics) includes information about queries that failed, queries that timed out, and queries that were canceled by the user.

## December 17, 2020

Feature

A [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-multi-region) is available in North America - `nam12` (Iowa/Northern Virginia/Oregon/Oklahoma).

Feature

A [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-multi-region) is available in Europe - `eur6` (Netherlands/Frankfurt/Zurich).

## December 07, 2020

Feature

Spanner supports the [LOCK_SCANNED_RANGES](https://docs.cloud.google.com/spanner/docs/query-syntax#statement-hints) statement hint which lets you request an exclusive lock on a set of ranges scanned by a transaction.

## November 09, 2020

Feature

A [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-multi-region) is available in North America - `nam8` (Los Angeles/Oregon/Salt Lake City).

## October 29, 2020

Feature

The following updates for Spanner SQL are available:

- Ability to convert between `BYTES` and base32-encoded strings using [FROM_BASE32](https://docs.cloud.google.com/spanner/docs/functions-and-operators#from_base32) and [TO_BASE32](https://docs.cloud.google.com/spanner/docs/functions-and-operators#to_base32) functions.

- Support for [ARRAY_IS_DISTINCT](https://docs.cloud.google.com/spanner/docs/array_functions#array_is_distinct) .

## October 23, 2020

Feature

The following [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-multi-region) is available in North America - `nam7` (Iowa/North Virginia/Oklahoma).

## October 15, 2020

Feature

A [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-multi-region) is available in North America - `nam9` (North Virginia/Iowa/South Carolina/Oregon).

## October 13, 2020

Feature

Generated columns support is [generally available](https://cloud.google.com/terms/launch-stages) , allowing you to define columns that are computed from other columns in a row. For more information, see [Creating and managing generated columns](https://docs.cloud.google.com/spanner/docs/generated-column/how-to) .

Feature

`CHECK` constraints are [generally available](https://cloud.google.com/terms/launch-stages) , allowing you to define a boolean expression on the columns of a table and require that all rows in the table satisfy the expression. For more information, see [Creating and managing check constraints](https://docs.cloud.google.com/spanner/docs/check-constraint/how-to) .

## October 08, 2020

Feature

The following updates to Spanner standard SQL are available :

- Support for [SELECT \* REPLACE](https://docs.cloud.google.com/spanner/docs/query-syntax#select_replace) and [SELECT \* EXCEPT](https://docs.cloud.google.com/spanner/docs/query-syntax#select_except) syntax.
- Documentation for [Net functions](https://docs.cloud.google.com/spanner/docs/net_functions) .

## September 28, 2020

Feature

The [`NUMERIC` data type](https://cloud.google.com/spanner/docs/data-types#numeric_type) is [generally available](https://cloud.google.com/products#product-launch-stages) .

## September 09, 2020

Feature

An introspection tool is available in Spanner that provides insights into queries that are currently running in your database. Use [oldest active queries](https://docs.cloud.google.com/spanner/docs/introspection/oldest-active-queries) to analyze what queries are running and how they are impacting database performance characteristics.

## August 20, 2020

Feature

The following [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-multi-region) is available in North America - `nam11` (Iowa/South Carolina).

## August 06, 2020

Feature

The following [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-multi-region) is available in North America - `nam10` (Iowa/Salt Lake).

## July 30, 2020

Feature

The Spanner emulator is [generally available](https://cloud.google.com/products/#product-launch-stages) , letting you to develop and test applications locally. For more information, see [Emulate Spanner locally](https://docs.cloud.google.com/spanner/docs/emulator) .

## July 15, 2020

Feature

You can run SQL queries to retrieve [read statistics](https://docs.cloud.google.com/spanner/docs/introspection/read-statistics) for your database over recent one-minute, 10-minute, and one-hour time periods.

## June 08, 2020

Feature

A [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-multi-region) is available in Asia - `asia1` (Tokyo/Osaka).

Feature

A second [multi-region instance configuration](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-multi-region) is available in Europe - `eur5` (London/Belgium).

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Jakarta (asia-southeast2).

## June 03, 2020

Change

Spanner SQL supports the following statistical aggregate functions - STDDEV, VARIANCE. For more information, see [Statistical Aggregate Functions](https://docs.cloud.google.com/spanner/docs/statistical_aggregate_functions) .

## May 18, 2020

Feature

You can run SQL queries to retrieve [transaction statistics](https://docs.cloud.google.com/spanner/docs/transaction-stats-tables) for your database over recent one-minute, 10-minute, and one-hour time periods.

## April 20, 2020

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Las Vegas (us-west4).

## April 17, 2020

Feature

Spanner Backup and Restore is [generally available](https://cloud.google.com/products#product-launch-stages) , enabling you to create backups of Spanner databases on demand, and restore them. For more information, see [Backup and Restore](https://docs.cloud.google.com/spanner/docs/backup) .

Feature

Query Optimizer Versioning is [generally available](https://cloud.google.com/products#product-launch-stages) , enabling you to select which version of the optimizer to use for your database, application or query. For more information, see [Query optimizer](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview) .

## April 01, 2020

Feature

A [beta](https://cloud.google.com/products/#product-launch-stages) version of the Spanner emulator is available, enabling you to develop and test Spanner applications locally. For more information, see [Using the Spanner Emulator](https://docs.cloud.google.com/spanner/docs/emulator) .

## March 19, 2020

Feature

The open-source [C++ client library for Spanner](https://github.com/googleapis/google-cloud-cpp-spanner) is available. To get started using C++ with Spanner, [see this tutorial](https://docs.cloud.google.com/spanner/docs/getting-started/cpp) .

## March 05, 2020

Feature

Foreign keys is [generally available](https://cloud.google.com/terms/launch-stages) . For more information, see [Foreign keys overview](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview) .

## February 24, 2020

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Salt Lake City (us-west3).

## January 24, 2020

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Seoul (asia-northeast3).

## December 18, 2019

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Frankfurt ( `europe-west3` ).

## November 25, 2019

Feature

SQL queries support the [`WITH` clause](https://docs.cloud.google.com/spanner/docs/query-syntax#with-clause) . This clause lets you bind the results of subqueries to temporary tables which makes it easier to structure complex queries and optimize them for a faster execution time.

## October 16, 2019

Feature

If you would like to use [Hibernate ORM](https://hibernate.org/orm/) with Spanner, we provide a guide to [help you connect Hibernate ORM to Spanner](https://docs.cloud.google.com/spanner/docs/use-hibernate) .

## September 25, 2019

Change

All Spanner instances, including 1-node and 2-node instances, are covered under the [Service Level Agreement (SLA)](https://cloud.google.com/spanner/sla) . Spanner supports 99.99% monthly uptime percentage for all regional instances and 99.999% monthly uptime percentage for all multi-region instances under the Spanner SLA, regardless of instance size.

## August 09, 2019

Change

The Google Cloud console no longer provides a chart that shows the stacked throughput, by region, for instances with multi-region configurations. Instead, you can use the chart that shows total throughput for all regions. [Learn more about monitoring with the Google Cloud console](https://docs.cloud.google.com/spanner/docs/monitoring-console) .

## August 07, 2019

Feature

An [open-source JDBC driver](https://docs.cloud.google.com/spanner/docs/open-source-jdbc) for Spanner is available. This open-source driver enables Java applications to access Spanner through the Java Database Connectivity (JDBC) API.

## July 31, 2019

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in São Paulo ( `southamerica-east1` ).

## June 26, 2019

Change

If you write your applications in Java with the [Spring Framework](https://spring.io/projects/spring-framework) , we provide a guide to help you [add Spring Data Spanner to your application](https://docs.cloud.google.com/spanner/docs/adding-spring) . Spring Data Spanner can make it easier and more efficient to work with Spanner.

Feature

You can [import and export Spanner data in CSV format](https://docs.cloud.google.com/spanner/docs/import-export-csv) . You can use this feature to copy CSV data between Spanner and traditional relational database management systems, in combination with tools such as the `mysqldump` tool for MySQL, the `COPY` statement for PostgreSQL, or the `bcp` tool for Microsoft SQL Server.

## June 21, 2019

Feature

For SQL queries, Spanner automatically uses any [secondary indexes](https://docs.cloud.google.com/spanner/docs/secondary-indexes) that are likely to make the query more efficient.

If you notice a performance regression in an existing query, [follow these troubleshooting steps](https://docs.cloud.google.com/spanner/docs/troubleshooting-performance-regressions) , and [get support](https://docs.cloud.google.com/spanner/docs/getting-support) if you cannot resolve the issue.

## May 15, 2019

Feature

The Google Cloud console and the Cloud Monitoring console provide [latency charts](https://docs.cloud.google.com/spanner/docs/identify-latency-point) for Spanner. Use these charts to help you troubleshoot performance issues.

## April 30, 2019

Feature

The Google Cloud console and the Cloud Monitoring console provide [CPU utilization charts](https://docs.cloud.google.com/spanner/docs/cpu-utilization) for additional CPU utilization metrics. You can also [set up alerts in Cloud Monitoring](https://docs.cloud.google.com/spanner/docs/monitoring-stackdriver#create-alert) to track these metrics.

## April 18, 2019

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Osaka ( `asia-northeast2` ).

## March 14, 2019

Feature

Spanner lets you to send multiple DML statements in one transaction using [batch DML](https://docs.cloud.google.com/spanner/docs/dml-tasks#use-batch) .

## March 11, 2019

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Zürich ( `europe-west6` ).

## February 01, 2019

Feature

Published new documentation for migrating data to Spanner:

- [Migrating from DynamoDB to Spanner](https://cloud.google.com/solutions/migrating-dynamodb-to-cloud-spanner)
- [Migrating from MySQL to Spanner](https://cloud.google.com/solutions/migrating-mysql-to-spanner)
- [Migrating from an Oracle OLTP system to Spanner](https://cloud.google.com/solutions/migrating-oracle-to-cloud-spanner)
- [Migrating from PostgreSQL to Spanner](https://docs.cloud.google.com/spanner/docs/migrating-postgres-spanner)

## January 17, 2019

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Sydney ( `australia-southeast1` ).

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in London ( `europe-west2` ).

## January 11, 2019

Feature

Spanner supports [importing data from other databases](https://docs.cloud.google.com/spanner/docs/import-non-spanner) using the [Avro to Spanner template](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided-batch#gcsavrotocloudspanner) .

## December 06, 2018

Feature

Announced general availability of the [Java client library](https://docs.cloud.google.com/spanner/docs/reference/libraries) for Spanner.

## November 26, 2018

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Hong Kong ( `asia-east2` ).

## November 13, 2018

Feature

The Google Cloud console displays [query statistics](https://docs.cloud.google.com/spanner/docs/query-statistics) , as measured by CPU usage, for Spanner queries over recent one-minute, 10-minute, one-hour, and one-day time periods. You can also run SQL queries over the [SPANNER_SYS](https://docs.cloud.google.com/spanner/docs/query-stats-tables) tables to retrieve recent query statistics.

## October 29, 2018

Feature

Spanner [multi-region instance configurations](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-multi-region) can be created in Europe ( `eur3` ).

## October 26, 2018

Feature

The Google Cloud CLI command-line tool includes beta support for [inserting, updating, and deleting table rows](https://docs.cloud.google.com/spanner/docs/modify-gcloud#modifying_data_using_dml) using Partitioned DML.

## October 22, 2018

Feature

Spanner [multi-region instance configurations](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-multi-region) can be created in a second multi-region instance configuration in North America ( `nam6` ).

## October 10, 2018

Feature

Spanner supports [Data Manipulation Language (DML)](https://docs.cloud.google.com/spanner/docs/dml-tasks) statements, including [Partitioned DML](https://docs.cloud.google.com/spanner/docs/dml-partitioned) .

## August 17, 2018

Feature

You can use the REST API or the `gcloud` command-line tool to [export](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided-batch#cloudspannertogcsavro) and [import](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided-batch#gcsavrotocloudspanner) Spanner databases.

## July 12, 2018

Feature

Spanner supports [exporting](https://docs.cloud.google.com/spanner/docs/export) and [importing](https://docs.cloud.google.com/spanner/docs/import) databases using the Google Cloud console.

## July 10, 2018

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Los Angeles ( `us-west2` ).

## June 11, 2018

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Finland ( `europe-north1` ).

## June 06, 2018

Change

Published new and updated documentation for designing and updating schemas:

- [Schema and data model](https://docs.cloud.google.com/spanner/docs/schema-and-data-model)
- [Schema design](https://docs.cloud.google.com/spanner/docs/schema-design)
- [Schema updates](https://docs.cloud.google.com/spanner/docs/schema-updates)

## May 10, 2018

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Singapore ( `asia-southeast1` ).

## April 26, 2018

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in South Carolina ( `us-east1` ).

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Oregon ( `us-west1` ).

## April 24, 2018

Feature

The gcloud command-line tool includes beta support for [inserting, updating, and deleting rows](https://docs.cloud.google.com/spanner/docs/modify-gcloud) in a table.

## March 28, 2018

Feature

Spanner supports the [commit timestamp column option](https://docs.cloud.google.com/spanner/docs/commit-timestamp) that you can use to automatically write the commit timestamp of a transaction into a column.

## March 15, 2018

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in the Netherlands ( `europe-west4` ).

## March 02, 2018

Feature

Spanner supports [reading and querying data in parallel](https://docs.cloud.google.com/spanner/docs/reads#read_data_in_parallel) with multiple workers using the client libraries for Node.js and Ruby.

## February 27, 2018

Feature

Spanner supports [reading and querying data in parallel](https://docs.cloud.google.com/spanner/docs/reads#read_data_in_parallel) with multiple workers using the client libraries for C#, Go, Java, and PHP.

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Montréal ( `northamerica-northeast1` ).

## February 20, 2018

Feature

Spanner supports both Admin Activity and Data Access [audit logs](https://docs.cloud.google.com/spanner/docs/audit-logging) as a part of Cloud Logging.

## January 31, 2018

Feature

Announced general availability of [IAM custom roles](https://docs.cloud.google.com/spanner/docs/iam#custom-roles) for Spanner.

## January 18, 2018

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Northern Virginia ( `us-east4` ) and Mumbai ( `asia-south1` ).

## November 14, 2017

Feature

Spanner [multi-region instance configurations](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-multi-region) are available. Multi-region instances can be created in one continent (nam3) or three continents (nam-eur-asia1).

## June 15, 2017

Feature

Spanner [regional instances](https://docs.cloud.google.com/spanner/docs/instances#available-configurations-regional) can be created in Tokyo ( `asia-northeast1` ).

## May 16, 2017

Feature

Announced general availability of the Cloud Spanner API.

Feature

The following documentation is available: [SQL best practices](https://docs.cloud.google.com/spanner/docs/sql-best-practices) , [Working with arrays in SQL](https://docs.cloud.google.com/spanner/docs/arrays) , [Using Spanner with Cloud Run functions](https://docs.cloud.google.com/spanner/docs/use-cloud-functions) , [Monitoring using Cloud Monitoring](https://docs.cloud.google.com/spanner/docs/monitoring) , [Applying IAM roles for databases, instances, and projects](https://docs.cloud.google.com/spanner/docs/grant-permissions) , [Integrating with other Google Cloud services](https://docs.cloud.google.com/spanner/docs/integrate-google-cloud-platform) .

## February 14, 2017

Issue

The Spanner Studio in the Google Cloud console is available.

Feature

Initial Beta release of Cloud Spanner API.

Issue

The version of the Google Cloud CLI command-line tool that supports Spanner is being rolled out. It is expected to be completely available by February 16, 2017.
