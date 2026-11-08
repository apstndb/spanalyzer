---
name: documents/docs.cloud.google.com/spanner/docs/full-text-search/search-indexes
uri: https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes
title: Search indexes
description: Create and manage search indexes in Spanner to enable efficient full-text search capabilities on your text data.
data_source: docs.cloud.google.com
---

> **Note:** This feature is available with the Spanner Enterprise edition and Enterprise Plus edition. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

This page describes how to add and use search indexes. The [full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) is run against entries in the search index.

## How to use search indexes

You can create a *search index* on any columns that you want to make available for full-text searches. To create a search index, use the [`CREATE SEARCH INDEX`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create-search-index) DDL statement. To update an index use the [`ALTER SEARCH INDEX`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter-search-index) DDL statement. Spanner automatically builds and maintains the search index, including adding and updating data in the search index as soon as it changes in the database.

## Search index partitions

A search index can be *partitioned* or *unpartitioned* , depending upon the type of queries that you want to accelerate.

- An example for when a partitioned index is the best choice is when the application queries an email mailbox. Each query is restricted to a specific mailbox.

- An example for when an unpartitioned query is the best choice is when there's a query across all product categories in a product catalog.

## Search index use cases

Besides full-text search, Spanner search indexes support the following:

- [JSON searches](https://docs.cloud.google.com/spanner/docs/full-text-search/json-indexes) , which is an efficient way to index and query JSON and JSONB documents.
- [Substring searches](https://docs.cloud.google.com/spanner/docs/full-text-search/substring-search) , which is a type of query that looks for a shorter string (the substring) within a larger body of text.
- Combining conditions on any subset of indexed data, including exact-match and numeric, into a single index scan.

For more information about use cases, see [Search versus secondary indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/search-vs-secondary-index) .

## Search index example

To show the capabilities of search indexes, suppose that there's a table that stores information about music albums:

### GoogleSQL

```
CREATE TABLE Albums (
  AlbumId STRING(MAX) NOT NULL,
  AlbumTitle STRING(MAX)
) PRIMARY KEY(AlbumId);
```

### PostgreSQL

```
CREATE TABLE albums (
  albumid character varying NOT NULL,
  albumtitle character varying,
PRIMARY KEY(albumid));
```

Spanner has several tokenize functions that create tokens. To modify the previous table to let users run a full-text search to find album titles, use the `TOKENIZE_FULLTEXT` function to create tokens from album titles. Then create a column that uses the `TOKENLIST` data type to hold the tokenization output from `TOKENIZE_FULLTEXT` . For this example, we create the `AlbumTitle_Tokens` column.

### GoogleSQL

```
ALTER TABLE Albums
  ADD COLUMN AlbumTitle_Tokens TOKENLIST
  AS (TOKENIZE_FULLTEXT(AlbumTitle)) HIDDEN;
```

### PostgreSQL

```
ALTER TABLE albums
  ADD COLUMN albumtitle_tokens spanner.tokenlist
    GENERATED ALWAYS AS (spanner.tokenize_fulltext(albumtitle)) VIRTUAL HIDDEN;
```

The following example uses the [`CREATE SEARCH INDEX`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create-search-index) DDL to create a search index ( `AlbumsIndex` ) on the `AlbumTitle` tokens ( `AlbumTitle_Tokens` ):

### GoogleSQL

```
CREATE SEARCH INDEX AlbumsIndex
  ON Albums(AlbumTitle_Tokens);
```

### PostgreSQL

This example uses [`CREATE SEARCH INDEX`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#create-search-index) .

```
CREATE SEARCH INDEX albumsindex ON albums(albumtitle_tokens);
```

After adding the search index, use SQL queries to find albums matching the search criteria. For example:

### GoogleSQL

```
SELECT AlbumId
FROM Albums
WHERE SEARCH(AlbumTitle_Tokens, "fifth symphony")
```

### PostgreSQL

```
SELECT albumid
FROM albums
WHERE spanner.search(albumtitle_tokens, 'fifth symphony')
```

### Create and query a search index for a named schema

You can create a search index on a [named schema](https://docs.cloud.google.com/spanner/docs/named-schemas) , as shown in the following examples.

### GoogleSQL

```
CREATE SCHEMA AlbumSchema;
CREATE TABLE AlbumSchema.Albums (
  AlbumId STRING(MAX) NOT NULL,
  AlbumTitle STRING(MAX),
  AlbumTitle_Tokens TOKENLIST AS (TOKENIZE_FULLTEXT(AlbumTitle)) STORED HIDDEN,
) PRIMARY KEY(AlbumId);

CREATE SEARCH INDEX AlbumSchema.AlbumsIndex ON AlbumSchema.Albums(AlbumTitle_Tokens);
```

### PostgreSQL

In PostgreSQL, you can't use fully qualified names to create indexes on named schemas.

```
CREATE SCHEMA AlbumSchema;
CREATE TABLE AlbumSchema.albums (
  albumid character varying NOT NULL,
  albumtitle character varying,
  albumtitle_tokens spanner.tokenlist GENERATED ALWAYS AS (spanner.tokenize_fulltext(albumtitle)) STORED HIDDEN,
  PRIMARY KEY(albumid)
);
CREATE SEARCH INDEX albumsindex ON AlbumSchema.albums(albumtitle_tokens);
```

The following examples show how to query the search index on the named schema:

### GoogleSQL

```
SELECT AlbumId
FROM AlbumSchema.Albums
WHERE SEARCH(AlbumTitle_Tokens, "fifth symphony")
```

The following example uses a query hint to tell the query optimizer to use the index named `AlbumsIndex` when executing the query.

```
SELECT AlbumId
FROM AlbumSchema.Albums@{FORCE_INDEX="AlbumSchema.AlbumsIndex"}
WHERE SEARCH(AlbumTitle_Tokens, "fifth symphony")
```

### PostgreSQL

In PostgreSQL, you can't use fully qualified names to create indexes on named schemas.

```
SELECT albumid
FROM AlbumSchema.albums
WHERE spanner.search(albumtitle_tokens, 'fifth symphony')
```

The following example uses a query hint to tell the query optimizer to use the index named `albumsindex` when executing the query.

```
SELECT albumid
FROM AlbumSchema.albums /*@ FORCE_INDEX = albumsindex */
WHERE spanner.search(albumtitle_tokens, 'fifth symphony')
```

## Data consistency

When an index is created, Spanner uses automated processes to backfill the data to ensure consistency. When writes are committed, the indexes are updated in the same transaction. Spanner automatically performs data consistency checks.

## Search index schema definitions

Search indexes are defined on one or more `TOKENLIST` columns of a table. Search indexes have the following components:

- **Base table** : the Spanner table that needs indexing.
- **`TOKENLIST` column** : a collection of columns that define the tokens that need indexing. The order of these columns is unimportant.

For example, in the following statement, the base table is Albums. `TOKENLIST` columns are created on `AlbumTitle` ( `AlbumTitle_Tokens` ) and `Rating` ( `Rating_Tokens` ).

### GoogleSQL

```
CREATE TABLE Albums (
  AlbumId STRING(MAX) NOT NULL,
  SingerId INT64 NOT NULL,
  ReleaseTimestamp INT64 NOT NULL,
  AlbumTitle STRING(MAX),
  Rating FLOAT64,
  AlbumTitle_Tokens TOKENLIST AS (TOKENIZE_FULLTEXT(AlbumTitle)) HIDDEN,
  Rating_Tokens TOKENLIST AS (TOKENIZE_NUMBER(Rating)) HIDDEN
) PRIMARY KEY(AlbumId);
```

### PostgreSQL

```
CREATE TABLE albums (
  albumid character varying NOT NULL,
  singerid bigint NOT NULL,
  releasetimestamp bigint NOT NULL,
  albumtitle character varying,
  rating double precision,
  albumtitle_tokens spanner.tokenlist GENERATED ALWAYS AS (spanner.tokenize_fulltext(albumtitle)) VIRTUAL HIDDEN,
  rating_tokens spanner.tokenlist GENERATED ALWAYS AS (spanner.tokenize_fulltext(rating)) VIRTUAL HIDDEN,
PRIMARY KEY(AlbumId));
```

Use the following [`CREATE SEARCH INDEX`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create-search-index) statement to create a search index using the tokens for `AlbumTitle` and `Rating` :

### GoogleSQL

```
CREATE SEARCH INDEX AlbumsIndex
ON Albums(AlbumTitle_Tokens, Rating_Tokens)
PARTITION BY SingerId
ORDER BY ReleaseTimestamp DESC
```

### PostgreSQL

```
CREATE SEARCH INDEX albumsindex
ON albums(albumtitle_tokens, rating_tokens)
PARTITION BY singerid
ORDER BY releasetimestamp DESC
```

Search indexes have the following options:

- **Partitions** : an optional group of columns that divide the search index. Querying a partitioned index is often significantly more efficient than querying an unpartitioned index. For more information, see [Partition search indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/partition-search-index) .
- **Sort order column** : an optional `INT64` column that establishes the order of retrieval from the search index. For more information, see [Search index sort order](https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes#search-index-sort-order) .
- **Interleaving** : like secondary indexes, you can interleave search indexes. Interleaved search indexes use fewer resources to write and join with the base table. For more information, see [Interleaved search indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes#interleaved_search_indexes) .
- **Options clause** : a list of key value pairs that overrides the default settings of the search index.

## Internal layout of search indexes

An important element of the internal representation of search indexes is a *docid* , which serves as a storage-efficient representation of the primary key of the base table which can be arbitrarily long. It's also what creates the order for the internal data layout according to the user-provided `ORDER BY` columns of the `CREATE SEARCH INDEX` statement. It's represented as one or two 64-bit integers.

Search indexes are implemented internally as a two-level mapping:

1.  Tokens to docids
2.  Docids to base table primary keys

This scheme results in significant storage savings as Spanner doesn't need to store the full base table primary key for each `<token, document>` pair.

There are two types of physical indexes that implement the two levels of mapping:

1.  **A secondary index** that maps partition keys and a docid to the base table primary key. In the example in the previous section, this maps `{SingerId, ReleaseTimestamp, uid}` to `{AlbumId}` . The secondary index also stores all columns specified in the `STORING` clause of `CREATE SEARCH INDEX` .
2.  **A token index** that maps tokens to docids, similar to inverted indexes in information retrieval literature. Spanner maintains a separate token index for each `TOKENLIST` of the search index. Logically, token indexes maintain lists of docids for each token within each partition (known in information retrieval as *postings lists* ). The lists are ordered by tokens for fast retrieval, and within lists, docid is used for ordering. Individual token indexes are an implementation detail not exposed through Spanner APIs.

Spanner supports the following options for docid.

| Search index                                      | Docid           | Behavior                                                                                                   |
|---------------------------------------------------|-----------------|------------------------------------------------------------------------------------------------------------|
| `ORDER BY` clause is omitted for the search index | `{uid}`         | Spanner adds a hidden unique value (UID) to identify each row.                                             |
| `ORDER BY column`                                 | `{column, uid}` | Spanner adds the UID column as a tiebreaker between rows with the same `column` values within a partition. |

Usage notes:

- The internal UID column isn't exposed through the Spanner API.
- In indexes where the UID isn't added, transactions that add a row with an already existing (partition,sort order) fails.

For example, consider the following data:

| AlbumId | SingerId | ReleaseTimestamp | SongTitle      |
|---------|----------|------------------|----------------|
| a1      | 1        | 997              | Beautiful days |
| a2      | 1        | 743              | Beautiful eyes |

Assuming the presort column is in ascending order, the content of the token index partitioned by `SingerId` partitions the content of the token index in the following way:

| SingerId | \_token   | ReleaseTimestamp | uid  |
|----------|-----------|------------------|------|
| 1        | beautiful | 743              | uid1 |
| 1        | beautiful | 997              | uid2 |
| 1        | days      | 743              | uid1 |
| 1        | eyes      | 997              | uid2 |

## Search index sharding

When Spanner [splits a table](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#database-splits) it distributes search index data so that all tokens in a particular base table row are in the same split. In other words, the search index is document-sharded. This sharding strategy has significant performance implications:

1.  The number of servers that each transaction communicates with remains constant, regardless of the number of tokens or the number of indexed `TOKENLIST` columns.
2.  Search queries involving multiple conditional expressions are executed independently on each split, avoiding the performance overhead associated with a distributed join.

Search indexes have two distribution modes:

- *Uniform sharding* (default). In uniform sharding, indexed data for each base table row is assigned randomly to an index split of a partition.
- *Sort-order sharding* . In sort-order sharding, data for each base table row is assigned to an index split of a partition based on the `ORDER BY` columns (that is, presort columns). For example, in the case of a descending sort order, all the rows with the largest sort order values appear on the first index split of a partition, and the next-largest group of sort order values on the next split.

These sharding modes come with a tradeoff between hotspot risks and the query cost:

- Uniform sharded search indexes are recommended when read or write patterns to the search index could lead to [hotspots](https://docs.cloud.google.com/spanner/docs/schema-design#primary-key-prevent-hotspots) . Uniform sharding mitigates hotspots by distributing read and write load evenly across splits, but this might increase resource usage during query executions as a tradeoff. In uniform sharded search indexes, queries must read all splits within a partition, due to the randomly distributed data. When accessing uniformly sharded indexes, Spanner reads all splits in parallel to reduce overall query latency.

- Sort-order sharded search indexes are preferable when read or write patterns are unlikely to cause hotspots. This approach can reduce the cost of queries whose `ORDER BY` matches the `ORDER BY` of the index, and specifies a relatively low `LIMIT` . When executing such queries, Spanner reads starting from the first splits of a partition incrementally, and the query can complete without reading all splits when `LIMIT` can be satisfied early.

- The sharding mode of a search index is configured using the [`OPTIONS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create-search-index) clause.

### GoogleSQL

```
CREATE SEARCH INDEX AlbumsIndex
ON Albums(AlbumTitle_Tokens, Rating_Tokens)
PARTITION BY SingerId
ORDER BY ReleaseTimestamp DESC
OPTIONS (sort_order_sharding = true);
```

### PostgreSQL

The sharding mode of a search index is configured using the [`WITH`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#create_search_index) clause.

```
CREATE SEARCH INDEX albumsindex
ON albums(albumtitle_tokens, rating_tokens)
PARTITION BY singerid
ORDER BY releasetimestamp DESC
WITH (sort_order_sharding = true);
```

When `sort_order_sharding=false` is set or left unspecified, the search index is created using uniform sharding.

## Interleaved search indexes

Like secondary indexes, you can interleave search indexes in a parent table of the base table. The primary reason to use interleaved search indexes is to colocate base table data with index data for small partitions. This opportunistic colocation has the following advantages:

- Writes don't need to do a [two-phase commit](https://docs.cloud.google.com/spanner/docs/transactions#introduction) .
- Back-joins of the search index with the base table aren't distributed.

Interleaved search indexes have the following restrictions:

1.  Only [sort-order sharded indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes#search_index_sharding) can be interleaved.
2.  Search indexes can only be interleaved in top-level tables (not in child tables).
3.  Like interleaved tables and secondary indexes, make the key of the parent table a prefix of the `PARTITION BY` columns in the interleaved search index.

### Define an interleaved search index

The following example demonstrates how to define an interleaved search index:

### GoogleSQL

```
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY(SingerId);

CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId STRING(MAX) NOT NULL,
  AlbumTitle STRING(MAX),
  AlbumTitle_Tokens TOKENLIST AS (TOKENIZE_FULLTEXT(AlbumTitle)) HIDDEN
) PRIMARY KEY(SingerId, AlbumId),
INTERLEAVE IN PARENT Singers ON DELETE CASCADE;

CREATE SEARCH INDEX AlbumsIndex
ON Albums(AlbumTitle_Tokens)
PARTITION BY SingerId,
INTERLEAVE IN Singers
OPTIONS (sort_order_sharding = true);
```

### PostgreSQL

```
CREATE TABLE singers(
  singerid bigint NOT NULL
PRIMARY KEY(singerid));

CREATE TABLE albums(
  singerid bigint NOT NULL,
  albumid character varying NOT NULL,
  albumtitle character varying,
  albumtitle_tokens spanner.tokenlist
  GENERATED ALWAYS
AS (
  spanner.tokenize_fulltext(albumtitle)
) VIRTUAL HIDDEN,
  PRIMARY KEY(singerid, albumid)),
INTERLEAVE IN PARENT singers ON DELETE CASCADE;

CREATE
  SEARCH INDEX albumsindex
ON
  albums(albumtitle_tokens)
  PARTITION BY singerid INTERLEAVE IN singers WITH(sort_order_sharding = true);
```

## Search index sort order

Requirements for the search index sort order definition are different from [secondary indexes](https://docs.cloud.google.com/spanner/docs/secondary-indexes) .

For example, consider the following table:

### GoogleSQL

```
CREATE TABLE Albums (
  AlbumId STRING(MAX) NOT NULL,
  ReleaseTimestamp INT64 NOT NULL,
  AlbumName STRING(MAX),
  AlbumName_Token TOKENLIST AS (TOKEN(AlbumName)) HIDDEN
) PRIMARY KEY(AlbumId);
```

### PostgreSQL

```
CREATE TABLE albums (
  albumid character varying NOT NULL,
  releasetimestamp bigint NOT NULL,
  albumname character varying,
  albumname_token spanner.tokenlist
      GENERATED ALWAYS AS(spanner.token(albumname)) VIRTUAL HIDDEN,
PRIMARY KEY(albumid));
```

The application might define a secondary index to look up information using the `AlbumName` sorted by `ReleaseTimestamp` :

```
CREATE INDEX AlbumsSecondaryIndex ON Albums(AlbumName, ReleaseTimestamp DESC);
```

The equivalent search index looks as following (this uses exact-match tokenization, since secondary indexes don't support full-text searches):

```
CREATE SEARCH INDEX AlbumsSearchIndex
ON Albums(AlbumName_Token)
ORDER BY ReleaseTimestamp DESC;
```

Search index sort order must conform to the following requirements:

1.  Only use `INT64` columns for the sort order of a search index. Columns that have arbitrary sizes use too many resources in the search index because Spanner needs to store a [docid](https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes#internal_layout_of_search_indexes) next to every token. Specifically, the sort order column can't use the `TIMESTAMP` type because `TIMESTAMP` uses nanosecond precision which doesn't fit in a 64-bit integer.

2.  Sort order columns must not be `NULL` . There are two ways to meet this requirement:

    1.  Declare the sort order column as `NOT NULL` .
    2.  Configure the index to [exclude NULL values](https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes#null-filtered-indexes) .

A timestamp is often used to determine the sort order. A common practice is to use [microseconds since the Unix epoch](https://en.wikipedia.org/wiki/Unix_time) for such timestamps.

Applications usually retrieve the newest data first using a search index that's sorted in descending order.

## NULL-filtered search indexes

Search indexes can use the `WHERE `` column_name `` IS NOT NULL` syntax to exclude base table rows. NULL filtering can apply to partitioning keys, sort order columns, and stored columns. NULL filtering on stored array columns isn't allowed.

**Example**

### GoogleSQL

```
CREATE SEARCH INDEX AlbumsIndex
ON Albums(AlbumTitle_Tokens)
STORING (Genre)
WHERE Genre IS NOT NULL
```

### PostgreSQL

```
CREATE SEARCH INDEX albumsindex
ON albums(albumtitle_tokens)
INCLUDE (genre)
WHERE genre IS NOT NULL
```

The query must specify the NULL filtering condition ( `Genre IS NOT NULL` for this example) in the `WHERE` clause. Otherwise, the Query Optimizer isn't able to use the search index. For more information, see [SQL query requirements](https://docs.cloud.google.com/spanner/docs/full-text-search/query-overview#sql-query-requirements) .

Use NULL filtering on a generated column to exclude rows based on any arbitrary criteria. For more information, see [Create a partial index using a generated column](https://docs.cloud.google.com/spanner/docs/generated-column/how-to#create_a_partial_index_using_a_generated_column) .

## What's next

- Learn about [tokenization and Spanner tokenizers](https://docs.cloud.google.com/spanner/docs/full-text-search/tokenization) .
- Learn about [numeric indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/numeric-indexes) .
- Learn about [JSON indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/json-indexes) .
- Learn about [index partitioning](https://docs.cloud.google.com/spanner/docs/full-text-search/partition-search-index) .
