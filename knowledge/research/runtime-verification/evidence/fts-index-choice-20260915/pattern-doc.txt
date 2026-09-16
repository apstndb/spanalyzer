> **Note:** This feature is available with the Spanner Enterprise edition and Enterprise Plus edition. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

Spanner [search indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes) can accelerate pattern matching expressions such as [`LIKE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/operators#like_operator) , [`STARTS_WITH`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#starts_with) , [`ENDS_WITH`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#ends_with) , and regular expression matching predicate [`REGEXP_CONTAINS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#regexp_contains) . This page describes how to create and configure a search index using [`TOKENIZE_NGRAMS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#tokenize_ngrams) to accelerate pattern matching predicates.

## Configure an n-gram `TOKENLIST` for pattern-matching acceleration

To enable pattern-matching expressions acceleration, tokenize a lower-cased `STRING` column with [`TOKENIZE_NGRAMS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#tokenize_ngrams) and store the `STRING` column using the `STORING` clause in GoogleSQL, or `INCLUDE` clause in PostgreSQL.

### GoogleSQL

    CREATE TABLE Albums (
    AlbumId INT64 NOT NULL,
    AlbumTitle STRING(MAX),
    AlbumTitle_Ngram_Tokens TOKENLIST AS (
      TOKENIZE_NGRAMS(LOWER(AlbumTitle), ngram_size_min=>3, ngram_size_max=>4)) HIDDEN,
    ) PRIMARY KEY(AlbumId);
    
    CREATE SEARCH INDEX AlbumsIndex
    ON Albums(AlbumTitle_Ngram_Tokens) STORING (AlbumTitle);

### PostgreSQL

    CREATE TABLE albums (
    albumid bigint NOT NULL,
    album_title varchar,
    album_title_ngrams_tokens spanner.tokenlist GENERATED ALWAYS AS (
      spanner.tokenize_ngrams(
        lower(album_title),
        ngram_size_min => 3,
        ngram_size_max => 4
      )
    ) VIRTUAL HIDDEN,
    PRIMARY KEY(albumid));
    
    CREATE SEARCH INDEX albumsidx ON
    albums(album_title_ngrams_tokens) INCLUDE (album_title);

## Automatic acceleration of queries with pattern-matching predicates

The query optimizer might choose to accelerate the following queries using `AlbumsIndex` with `AlbumTitle_Ngram_Tokens` . Optionally, the query can provide `@{force_index = AlbumsIndex}` to force the optimizer to use `AlbumsIndex` .

### GoogleSQL

In GoogleSQL, we accelerate [`LIKE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/operators#like_operator) , [`STARTS_WITH`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#starts_with) , [`ENDS_WITH`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#ends_with) , and [`REGEXP_CONTAINS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#regexp_contains) .

  - `LIKE` predicate:
    
        SELECT AlbumId
        FROM Albums @{FORCE_INDEX=AlbumsIndex}
        WHERE AlbumTitle LIKE "%999%";

  - `STARTS_WITH` predicate:
    
        SELECT AlbumId
        FROM Albums @{FORCE_INDEX=AlbumsIndex}
        WHERE STARTS_WITH(AlbumTitle, "apple")

  - `ENDS_WITH` predicate:
    
        SELECT AlbumId
        FROM Albums @{FORCE_INDEX=AlbumsIndex}
        WHERE ENDS_WITH(AlbumTitle, "apple")

  - `REGEXP_CONTAINS` predicate:
    
        SELECT AlbumId
        FROM Albums @{FORCE_INDEX=AlbumsIndex}
        WHERE REGEXP_CONTAINS(AlbumTitle, r"(good|great)[ ]+morning")

### PostgreSQL

In PostgreSQL, we accelerate [`LIKE`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions#pattern-matching) and [`STARTS_WITH`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions#string_functions) .

  - `LIKE` predicate:
    
        SELECT albumid
        FROM albums /*@ FORCE_INDEX = albumsidx */
        WHERE album_title like '%999%';

  - `STARTS_WITH` predicate:
    
        SELECT albumid
        FROM albums /*@ FORCE_INDEX = albumsidx */
        WHERE starts_with(album_title, 'apple')

## Prerequisites on acceleration

For Spanner to enable this acceleration, the following rules must be met:

  - The index must store the `STRING` column using the `STORING` clause in GoogleSQL, or `INCLUDE` clause in PostgreSQL. This prevents costly back-joins to the base table during post-filtering, which is critical for performance when the search over-retrieves documents.
  - The `STRING` column must be tokenized using [`TOKENIZE_NGRAMS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#tokenize_ngrams) .
  - The tokenization must apply to `LOWER(column_name)` rather than `column_name` .
  - The [`LIKE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/operators#like_operator) pattern, [`STARTS_WITH`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#starts_with) prefix, [`ENDS_WITH`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#ends_with) suffix, or [`REGEXP_CONTAINS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#regexp_contains) regular expression must be specified as a constant literal. [Query parameters](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/lexical#query_parameters) are not supported to avoid acceleration on patterns that are too short.
  - The [`LIKE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/operators#like_operator) pattern, [`STARTS_WITH`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#starts_with) prefix, [`ENDS_WITH`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#ends_with) suffix, or [`REGEXP_CONTAINS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/string_functions#regexp_contains) regular expression must contain enough text for at least one n-gram. For example `r".*"` doesn't qualify because there's no sequence of characters to match. Similarly, if the ngram minimum size is set to 3, the [`LIKE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/operators#like_operator) predicate `"%ab%"` doesn't qualify because `"ab"` (size 2) is too short.

## What's next

  - Learn about [finding approximate matches with fuzzy search](https://docs.cloud.google.com/spanner/docs/full-text-search/fuzzy-search) .
