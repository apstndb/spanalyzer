---
name: documents/docs.cloud.google.com/spanner/docs/full-text-search/ranked-search
uri: https://docs.cloud.google.com/spanner/docs/full-text-search/ranked-search
title: Rank search results
description: Perform ranked full-text searches in Spanner to retrieve the most relevant results based on scoring functions.
data_source: docs.cloud.google.com
---

> **Note:** This feature is available with the Spanner Enterprise edition and Enterprise Plus edition. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

This page describes how to rank search results for [full-text searches](https://docs.cloud.google.com/spanner/docs/full-text-search) in Spanner.

Spanner supports computing a topicality score, which provides a building block for creating sophisticated ranking functions. These scores calculate the relevance of a result to a query, based on the query term frequency and other customizable options.

The following example shows how to perform a ranked search using the [`SCORE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#score) function:

### GoogleSQL

```
SELECT AlbumId
FROM Albums
WHERE SEARCH(AlbumTitle_Tokens, "fifth symphony")
ORDER BY SCORE(AlbumTitle_Tokens, "fifth symphony") DESC
```

### PostgreSQL

This example uses [`spanner.search`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#search_functions) with [`spanner.score`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#search_functions) .

```
SELECT albumid
FROM albums
WHERE spanner.search(albumtitle_tokens, 'fifth symphony')
ORDER BY spanner.score(albumtitle_tokens, 'fifth symphony') DESC
```

## Score query terms with the `SCORE` function

The [`SCORE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#score) function computes a score for each query term and then combines the scores. The per-term score is roughly based on [term frequency–inverse document frequency (TF/IDF)](https://en.wikipedia.org/wiki/Tf%E2%80%93idf) . The score is one component of the final ordering for a record. The query combines it with other signals, such as the freshness modulating the topicality score.

In the current implementation, the IDF part of TF/IDF is only available when `enhance_query=>true` is used. It calculates the relative frequency of words based on the full web corpus used by Google Search, rather than a specific search index. If rquery enhancement isn't enabled, the scoring only uses the term frequency (TF) component (that is, the IDF term is set to 1).

The `SCORE` function returns values that serve as relevance scores that Spanner uses to establish a sort order. They have no standalone meaning. The higher the score, the better it matches the query.

Usually arguments like `query` and `enhance_query` are the same across both `SEARCH` and `SCORE` functions to ensure consistency in retrieval and ranking.

The recommended way to do this is to use these arguments with [query parameters](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/lexical#query_parameters) rather than string literals and specify the same query parameters in the `SEARCH` and `SCORE` functions.

## Score multiple columns

Spanner uses the `SCORE` function to score each field individually. The query then combines these individual scores together. A common way of doing this is to sum up the individual scores and then boost them according to user-provided field weights (which are provided using SQL query parameters).

For example, the following query combines the output of two `SCORE` functions:

### GoogleSQL

```
SELECT AlbumId
FROM Albums
WHERE SEARCH(Title_Tokens, @p1) OR SEARCH(Studio_Tokens, @p2)
ORDER BY SCORE(Title_Tokens, @p1) * @titleweight + SCORE(Studio_Tokens, @p2) * @studioweight
LIMIT 25
```

### PostgreSQL

This example uses query parameters `$1` and `$2` which are bound to 'fifth symphony' and 'blue note', respectively.

```
SELECT albumid
FROM albums
WHERE spanner.search(title_tokens, $1) OR spanner.search(studio_tokens, $2)
ORDER BY spanner.score(title_tokens, $1) * $titleweight
        + spanner.score(studio_tokens, $2) * $studioweight
LIMIT 25
```

The following example adds two boost parameters:

- Freshness ( `FreshnessBoost` ) increases the score with `(1 + @freshnessweight * GREATEST(0, 30 - DaysOld) / 30)`
- Popularity( `PopularityBoost` ) increases the score by multiplying it by factor `(1 + IF(HasGrammy, @grammyweight, 0)` .

For readability, the query uses the `WITH` operator.

### GoogleSQL

```
SELECT AlbumId
FROM Albums
WHERE SEARCH(Title_Tokens, @p1) OR SEARCH(Studio_Tokens, @p2)
ORDER BY WITH(
  TitleScore AS SCORE(Title_Tokens, @p1) * @titleweight,
  StudioScore AS SCORE(Studio_Tokens, @p2) * @studioweight,
  DaysOld AS (UNIX_MICROS(CURRENT_TIMESTAMP()) - ReleaseTimestamp) / 8.64e+10,
  FreshnessBoost AS (1 + @freshnessweight * GREATEST(0, 30 - DaysOld) / 30),
  PopularityBoost AS (1 + IF(HasGrammy, @grammyweight, 0)),
  (TitleScore + StudioScore) * FreshnessBoost * PopularityBoost)
LIMIT 25
```

### PostgreSQL

This example uses query parameters `$1` , `$2` , `$3` , `$4` , `$5` , and `$6` which are bound to values specified for `titlequery` , `studioquery` , `titleweight` , `studioweight` , `grammyweight` , and `freshnessweight` , respectively.

```
SELECT albumid
FROM
  (
    SELECT
      albumid,
      spanner.score(title_tokens, $1) * $3 AS titlescore,
      spanner.score(studio_tokens, $2) * $4 AS studioscore,
      (extract(epoch FROM current_timestamp) * 10e+6 - releasetimestamp) / 8.64e+10 AS daysold,
      (1 + CASE WHEN hasgrammy THEN $5 ELSE 0 END) AS popularityboost
    FROM albums
    WHERE spanner.search(title_tokens, $1) OR spanner.search(studio_tokens, $2)
  ) AS subquery
ORDER BY (subquery.TitleScore + subquery.studioscore)
  * (1 + $6 * greatest(0, 30 - subquery.daysold) / 30) * subquery.popularityboost
LIMIT 25
```

[`TOKENLIST_CONCAT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#tokenlist_concat) can also used in both searching and scoring to simplify queries when appropriate.

### GoogleSQL

```
SELECT AlbumId
FROM Albums
WHERE SEARCH(TOKENLIST_CONCAT([Title_Tokens, Studio_Tokens]), @p)
ORDER BY SCORE(TOKENLIST_CONCAT([Title_Tokens, Studio_Tokens]), @p)
LIMIT 25
```

### PostgreSQL

This example uses [`spanner.tokenlist_concat`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions-and-operators#search_functions) . The query parameter `$1` is bound to 'blue note'.

```
SELECT albumid
FROM albums
WHERE spanner.search(spanner.tokenlist_concat(ARRAY[title_tokens, studio_tokens]), $1)
ORDER BY spanner.score(spanner.tokenlist_concat(ARRAY[title_tokens, studio_tokens]), $1)
LIMIT 25
```

## Boost query order matches

Spanner applies a multiplicative boost to the output of the `SCORE` function for values that contain the query terms in the same order that they appear in the query. There are two versions of this boost: partial match and exact match. A partial match boost is applied when:

1.  The `TOKENLIST` contains all the original terms in the query.
2.  The tokens are adjacent to one another, and in the same order as they appear in the query.

There are certain special rules for conjunctions, negations, and phrases:

- A query with a negation can't receive a partial match boost.
- A query with a conjunction receives a boost if part of the conjunction appears in the appropriate locations.
- A query with a phrase receives a boost if the phrase appears in the `TOKENLIST` , and the term to the left of the phrase in the query appears to the left of the phrase in the `TOKENLIST` , and the same applies to the term to the right of the phrase.

Spanner applies an exact match boost when all of the previous rules are true, and the first and last tokens in the query are the first and last tokens in the document.

**Example document: Bridge Over Troubled Water**

| Query                                         | Boost Applied |
|-----------------------------------------------|---------------|
| Bridge Troubled                               | no boost      |
| Bridge Over - other water                     | no boost      |
| Bridge (Over OR Troubled) Water               | no boost      |
| Bridge Over                                   | partial boost |
| Bridge Over (Troubled OR Water)               | partial boost |
| Bridge Over Troubled Water                    | exact boost   |
| Bridge "Over Troubled" Water                  | exact boost   |
| Bridge ("Over Troubled" OR missingterm) Water | exact boost   |

## Scorer versions

The scorer algorithm is updated periodically. Each release bundles a set of scoring algorithm improvements. For a detailed list of differences between versions, see [Scorer Versions](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/search_functions#score) .

You can set the default scorer version for your database, or you can specify a version for a specific query.

### Set the database's default scorer version

You can set the default version of the scorer algorithm for your database. This option determines which version of the scoring algorithm Spanner uses when the `SCORE` function is called without an explicit `version` score option.

### GoogleSQL

Set the `score_version` database option:

```
ALTER DATABASE database_name SET OPTIONS (score_version = 2)
```

### PostgreSQL

Set the `spanner.score_version` database option:

```
ALTER DATABASE database_name SET "spanner.score_version" = 2
```

The valid values for the database option are `1` or `2` . If the `version` parameter is present in a request, it overrides the database's default version.

### Override the scorer version per query

You can override the default scorer version for a specific query using the `version` parameter in the `SCORE` function's `options` argument. If a query defines a scorer version, it overrides the database's default version.

The following example overrides the default scorer version by specifying `version` in the `options` parameter:

### GoogleSQL

```
SELECT AlbumId
FROM Albums
WHERE SEARCH(AlbumTitle_Tokens, @query)
ORDER BY SCORE(
  AlbumTitle_Tokens,
  @query,
  options=>JSON '{"version": 2}'
) DESC
```

### PostgreSQL

This example uses query parameter `$1` which is bound to the query string.

```
SELECT albumid
FROM albums
WHERE spanner.search(albumtitle_tokens, $1)
ORDER BY spanner.score(
  albumtitle_tokens,
  $1,
  options => '{"version": 2}'::jsonb
) DESC
```

## Limit retrieval depth

Search indexes often contain millions of documents. For queries where the predicates have low selectivity, it's impractical to rank all the results. Scoring queries usually have two limits:

1.  **Retrieval depth limit** : the maximum number of rows to score.
2.  **Result set size limit** : the maximum number of rows that the query should return (typically the page size).

Queries can limit retrieval depth with SQL subqueries:

### GoogleSQL

```
SELECT AlbumId
FROM
  (
    SELECT AlbumId, SCORE(Title_Tokens, @p1) AS score
    FROM Albums
    WHERE SEARCH(Title_Tokens, @p1)
    ORDER BY ReleaseTimestamp DESC
    LIMIT @retrieval_limit
  )
ORDER BY score DESC
LIMIT @page_size
```

### PostgreSQL

This example uses query parameters `$1` , `$2` , and `$3` which are bound to values specified for `title_query` , `retrieval_limit` , and `page_size` , respectively.

```
SELECT albumid
FROM
  (
    SELECT albumid, spanner.score(title_tokens, $1) AS score
    FROM albums
    WHERE spanner.search(title_tokens, $1)
    ORDER BY releasetimestamp DESC
    LIMIT $2
  ) AS subquery
ORDER BY score DESC
LIMIT $3
```

This works particularly well if Spanner uses the most important ranking signal to sort the index.

## What's next

- Learn about [full-text search queries](https://docs.cloud.google.com/spanner/docs/full-text-search/query-overview) .
- Learn how to [perform a substring search](https://docs.cloud.google.com/spanner/docs/full-text-search/substring-search) .
- Learn how to [paginate search results](https://docs.cloud.google.com/spanner/docs/full-text-search/paginate-search-results) .
- Learn how to [mix full-text and non-text queries](https://docs.cloud.google.com/spanner/docs/full-text-search/mix-full-text-and-non-text-queries) .
- Learn how to [search multiple columns](https://docs.cloud.google.com/spanner/docs/full-text-search/search-multiple-columns) .
