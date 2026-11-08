---
name: documents/docs.cloud.google.com/spanner/docs/full-text-search/numeric-indexes
uri: https://docs.cloud.google.com/spanner/docs/full-text-search/numeric-indexes
title: Numeric search indexes
description: Create search indexes on numeric data in Spanner to perform efficient range queries and other numeric comparisons.
data_source: docs.cloud.google.com
---

> **Note:** This feature is available with the Spanner Enterprise edition and Enterprise Plus edition. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

In addition to indexing text, the [search index](https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes) provides an efficient way to index numbers. It's primarily used to augment [full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) queries with conditions on numeric fields. This page describes indexing numbers for equality and inequality queries, and indexing an array of numbers.

## Tokenize numbers

Number tokenizers are used to generate a set of tokens that are used to accelerate numeric comparison searches.

Use the [`TOKENIZE_NUMBER`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#tokenize_number) function to create a numeric index. `TOKENIZE_NUMBER` supports `INT64` , `FLOAT32` , `FLOAT64` or `ARRAY` of these types.

For PostgreSQL, use the [`spanner.tokenize_number`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#search_functions) function to create a numeric index. `spanner.tokenize_number` only supports the `bigint` type.

## Index numbers for equality and inequality queries

Spanner supports indexing numbers for *equality* and *inequality* . Equality searches match a number. Range and inequality searches match a number within a specific range. You set this value in the [`TOKENIZE_NUMBER`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#tokenize_number) `comparison_type` parameter:

- **Equality** : `comparison_type=>"equality"`
- **Inequality and equality** : `comparison_type=>"all"`

In both cases, the original number (either integer or floating point) undergoes a process of tokenization, which is conceptually similar to full-text tokenization. It produces a set of tokens that the query can then use to locate documents matching the number condition.

Equality indexing only produces one token, which represents the number. This mode is recommended if queries only have conditions in the form of `field = @p` in the `WHERE` clause.

Inequality and equality indexing can accelerate a wider range of conditions in the `WHERE` clause of the query. This includes `field < @p` , `field <= @p` , `field > @p` , `field >= @p` , `field BETWEEN @p1 and @p2` and `field <> @p` in addition to equality conditions. To implement this type of indexing, Spanner produces tokens in the underlying search index. Spanner can produce many tokens for each indexed number, depending upon tuning parameters. The number of tokens depends on the parameters that are set for `TOKENIZE_NUMBER` , such as `algorithm` , `min` , `max` and `granularity` . It's therefore important to evaluate the tuning parameters carefully to ensure an appropriate balance between disk storage and lookup time.

### Array tokenization

> **Note:** The examples in this section are intended for GoogleSQL-dialect databases. PostgreSQL doesn't support numeric array acceleration with search indexes.

In addition to scalar values, [`TOKENIZE_NUMBER`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#tokenize_number) supports tokenization of an array of numbers.

When `TOKENIZE_NUMBER` is used with the `ARRAY` column, you must specify `comparison_type=>"equality"` . Range queries aren't supported with an array of numbers.

```
  CREATE TABLE Albums (
    AlbumId STRING(MAX) NOT NULL,
    Ratings ARRAY<INT64>,
    Ratings_Tokens TOKENLIST
      AS (TOKENIZE_NUMBER(Ratings, comparison_type=>"equality")) HIDDEN
  ) PRIMARY KEY(AlbumId);

  CREATE SEARCH INDEX AlbumsIndex ON Albums(Ratings_Tokens);
```

The following query finds all albums that have a rating of 1 or 2:

```
  SELECT AlbumId
  FROM Albums
  WHERE ARRAY_INCLUDES_ANY(Ratings, [1, 2])
```

The following query finds all albums that were rated as 1 and as 5:

```
SELECT AlbumId
FROM Albums
WHERE ARRAY_INCLUDES_ALL(Ratings, [1, 5])
```

## What's next

- Learn about [tokenization and Spanner tokenizers](https://docs.cloud.google.com/spanner/docs/full-text-search/tokenization) .
- Learn about [search indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes) .
- Learn about [index partitioning](https://docs.cloud.google.com/spanner/docs/full-text-search/partition-search-index) .
