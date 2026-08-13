package main

const fullTextSearchDDL = `
CREATE TABLE SearchAlbums (
  SingerId INT64 NOT NULL,
  AlbumId STRING(MAX) NOT NULL,
  AlbumTitle STRING(MAX),
  AlbumStudio STRING(MAX),
  ArtistName STRING(MAX),
  Rating FLOAT64,
  ReleaseTimestamp INT64 NOT NULL,
  Likes INT64,
  Genres ARRAY<STRING(MAX)>,
  Cover BYTES(MAX),
  Ratings ARRAY<INT64>,
  AlbumTitle_Tokens TOKENLIST AS (TOKENIZE_FULLTEXT(AlbumTitle)) HIDDEN,
  AlbumTitle_SubstringTokens TOKENLIST AS (TOKENIZE_SUBSTRING(AlbumTitle)) HIDDEN,
  AlbumStudio_Tokens TOKENLIST AS (TOKENIZE_FULLTEXT(AlbumStudio)) HIDDEN,
  ArtistNameSoundex STRING(MAX) AS (LOWER(SOUNDEX(ArtistName))),
  ArtistNameSoundex_Tokens TOKENLIST AS (TOKEN(ArtistNameSoundex)) HIDDEN,
  Rating_Tokens TOKENLIST AS (TOKENIZE_NUMBER(Rating)) HIDDEN,
  Genres_Tokens TOKENLIST AS (TOKEN(Genres)) HIDDEN,
  Ratings_Tokens TOKENLIST AS (TOKENIZE_NUMBER(Ratings, comparison_type=>"equality")) HIDDEN,
) PRIMARY KEY(SingerId, AlbumId);

CREATE SEARCH INDEX SearchAlbumsTitleIndex
ON SearchAlbums(AlbumTitle_Tokens);

CREATE SEARCH INDEX SearchAlbumsTitleSubstringIndex
ON SearchAlbums(AlbumTitle_SubstringTokens);

CREATE SEARCH INDEX SearchAlbumsTitleStudioIndex
ON SearchAlbums(AlbumTitle_Tokens, AlbumStudio_Tokens);

CREATE SEARCH INDEX SearchAlbumsTitleRatingIndex
ON SearchAlbums(AlbumTitle_Tokens, Rating_Tokens)
PARTITION BY SingerId
ORDER BY ReleaseTimestamp DESC
OPTIONS (sort_order_sharding = true);

CREATE SEARCH INDEX SearchAlbumsRatingsIndex
ON SearchAlbums(Ratings_Tokens);

CREATE SEARCH INDEX SearchAlbumsMixedIndex
ON SearchAlbums(AlbumTitle_Tokens, Rating_Tokens, Genres_Tokens)
STORING (Likes);

CREATE SEARCH INDEX SearchAlbumsFacetIndex
ON SearchAlbums(AlbumTitle_Tokens)
STORING (AlbumTitle, Rating, Genres, Likes);

CREATE SEARCH INDEX SearchAlbumsPhoneticIndex
ON SearchAlbums(AlbumTitle_Tokens, ArtistNameSoundex_Tokens);

CREATE TABLE SearchSingers (
  SingerId INT64 NOT NULL,
  Bio STRING(MAX),
  BioTokens TOKENLIST AS (TOKENIZE_FULLTEXT(Bio)) HIDDEN,
) PRIMARY KEY(SingerId);

CREATE SEARCH INDEX SearchSingersByBio
ON SearchSingers(BioTokens);

CREATE TABLE SearchCollaborations (
  SingerId INT64 NOT NULL,
  FeaturingSingerId INT64 NOT NULL,
) PRIMARY KEY(SingerId, FeaturingSingerId);

CREATE PROPERTY GRAPH SearchMusicGraph
  NODE TABLES(
    SearchSingers
      KEY(SingerId)
      LABEL Singer PROPERTIES(SingerId, Bio, BioTokens)
  )
  EDGE TABLES(
    SearchCollaborations AS CollabWith
      KEY(SingerId, FeaturingSingerId)
      SOURCE KEY(SingerId) REFERENCES SearchSingers(SingerId)
      DESTINATION KEY(FeaturingSingerId) REFERENCES SearchSingers(SingerId)
      LABEL CollabWith PROPERTIES ALL COLUMNS
  );
`

var fullTextSearchQueries = []queryCase{
	{
		Label: "full-text-search/search",
		SQL:   `SELECT AlbumId FROM SearchAlbums WHERE SEARCH(AlbumTitle_Tokens, "friday OR monday")`,
	},
	{
		Label: "full-text-search/force-index",
		SQL:   `SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsTitleIndex} WHERE SEARCH(AlbumTitle_Tokens, "fifth symphony")`,
	},
	{
		Label: "full-text-search/snippet",
		SQL:   `SELECT AlbumId, SNIPPET(AlbumTitle, "Fast Car") FROM SearchAlbums WHERE SEARCH(AlbumTitle_Tokens, "Fast Car")`,
	},
	{
		Label: "full-text-search/score-order",
		SQL:   `SELECT AlbumId FROM SearchAlbums WHERE SEARCH(AlbumTitle_Tokens, "fifth symphony") ORDER BY SCORE(AlbumTitle_Tokens, "fifth symphony") DESC`,
	},
	{
		Label: "full-text-search/enhanced-query",
		SQL:   `SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsTitleIndex} WHERE SEARCH(AlbumTitle_Tokens, "hotl cal", enhance_query => TRUE)`,
	},
	{
		Label: "full-text-search/enhanced-query-control",
		SQL:   `SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsTitleIndex} WHERE SEARCH(AlbumTitle_Tokens, "hotl cal")`,
	},
	{
		Label: "full-text-search/enhanced-query-required-hint",
		SQL:   `@{require_enhance_query=true} SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsTitleIndex} WHERE SEARCH(AlbumTitle_Tokens, "hotl cal", enhance_query => TRUE)`,
	},
	{
		Label: "full-text-search/enhanced-query-timeout-hint",
		SQL:   `@{enhance_query_timeout_ms=200} SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsTitleIndex} WHERE SEARCH(AlbumTitle_Tokens, "hotl cal", enhance_query => TRUE)`,
	},
	{
		Label: "full-text-search/phonetic-composition",
		SQL:   `SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsPhoneticIndex} WHERE ArtistNameSoundex = LOWER(SOUNDEX("stefan")) AND SEARCH(AlbumTitle_Tokens, "cat")`,
	},
	{
		Label: "full-text-search/phonetic-search-only-control",
		SQL:   `SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsPhoneticIndex} WHERE SEARCH(AlbumTitle_Tokens, "cat")`,
	},
	{
		Label: "full-text-search/substring",
		SQL:   `SELECT AlbumId FROM SearchAlbums WHERE SEARCH_SUBSTRING(AlbumTitle_SubstringTokens, "happ")`,
	},
	{
		Label: "full-text-search/multi-column-conjunction",
		SQL:   `SELECT AlbumId FROM SearchAlbums WHERE SEARCH(AlbumTitle_Tokens, "car") AND SEARCH(AlbumStudio_Tokens, "sun")`,
	},
	{
		Label: "full-text-search/multi-column-disjunction",
		SQL:   `SELECT AlbumId FROM SearchAlbums WHERE SEARCH(AlbumTitle_Tokens, "car") OR SEARCH(AlbumStudio_Tokens, "sun")`,
	},
	{
		Label: "full-text-search/multi-column-negation",
		SQL:   `SELECT AlbumId FROM SearchAlbums WHERE NOT SEARCH(AlbumTitle_Tokens, "car")`,
	},
	{
		Label: "full-text-search/tokenlist-concat",
		SQL:   `SELECT AlbumId FROM SearchAlbums WHERE SEARCH(TOKENLIST_CONCAT([AlbumTitle_Tokens, AlbumStudio_Tokens]), "blue note") ORDER BY SCORE(TOKENLIST_CONCAT([AlbumTitle_Tokens, AlbumStudio_Tokens]), "blue note") LIMIT 25`,
	},
	{
		Label: "full-text-search/partitioned-ordered-index",
		SQL:   `SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsTitleRatingIndex} WHERE SingerId = 1 AND SEARCH(AlbumTitle_Tokens, "fifth symphony") ORDER BY ReleaseTimestamp DESC LIMIT 10`,
	},
	{
		Label: "full-text-search/numeric-array-any",
		SQL:   `SELECT AlbumId FROM SearchAlbums WHERE ARRAY_INCLUDES_ANY(Ratings, [1, 2])`,
	},
	{
		Label: "full-text-search/numeric-array-all",
		SQL:   `SELECT AlbumId FROM SearchAlbums WHERE ARRAY_INCLUDES_ALL(Ratings, [1, 5])`,
	},
	{
		Label: "full-text-search/mixed-accelerated",
		SQL:   `SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsMixedIndex} WHERE (SEARCH(AlbumTitle_Tokens, "car") OR Rating > 4) AND NOT ARRAY_INCLUDES_ANY(Genres, ["jazz"])`,
	},
	{
		Label: "full-text-search/mixed-stored-filter",
		SQL:   `SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsMixedIndex} WHERE SEARCH(AlbumTitle_Tokens, "car") AND Rating > 4 AND Likes >= 1000`,
	},
	{
		Label: "full-text-search/mixed-back-join",
		SQL:   `SELECT AlbumId, Cover FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsMixedIndex} WHERE SEARCH(AlbumTitle_Tokens, "car") AND Rating > 4`,
	},
	{
		Label: "full-text-search/gql-search-traversal",
		SQL: `GRAPH SearchMusicGraph
MATCH (src:Singer)-[:CollabWith]->(dst:Singer)
WHERE SEARCH(dst.BioTokens, "jazz")
RETURN src.SingerId AS src_id, dst.SingerId AS dst_id
ORDER BY dst.SingerId`,
	},
	{
		Label: "full-text-search/facet-single-count",
		SQL: `SELECT Rating, COUNT(*) AS result_count
FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsFacetIndex}
WHERE SEARCH(AlbumTitle_Tokens, 'foo')
GROUP BY Rating
ORDER BY Rating DESC`,
	},
	{
		Label: "full-text-search/facet-search-only-control",
		SQL: `SELECT Rating
FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsFacetIndex}
WHERE SEARCH(AlbumTitle_Tokens, 'foo')
ORDER BY Rating DESC`,
	},
	{
		Label: "full-text-search/facet-multiple",
		SQL: `WITH search_results AS (
  SELECT SingerId, AlbumId, AlbumTitle, Genres, Rating, Likes
  FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsFacetIndex}
  WHERE SEARCH(AlbumTitle_Tokens, 'foo')
  ORDER BY Likes DESC, SingerId, AlbumId
  LIMIT 10000
)
SELECT
  ARRAY(SELECT AS STRUCT * FROM search_results ORDER BY Likes DESC, SingerId, AlbumId LIMIT 50) AS result_page,
  ARRAY(SELECT AS STRUCT Rating, COUNT(*) AS result_count FROM search_results GROUP BY Rating ORDER BY result_count DESC, Rating DESC) AS rating_counts,
  ARRAY(SELECT AS STRUCT genre, COUNT(*) AS result_count FROM search_results JOIN UNNEST(Genres) AS genre GROUP BY genre ORDER BY result_count DESC, genre LIMIT 5) AS genres_counts`,
	},
	{
		Label: "full-text-search/facet-result-page-control",
		SQL: `WITH search_results AS (
  SELECT SingerId, AlbumId, AlbumTitle, Genres, Rating, Likes
  FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsFacetIndex}
  WHERE SEARCH(AlbumTitle_Tokens, 'foo')
  ORDER BY Likes DESC, SingerId, AlbumId
  LIMIT 10000
)
SELECT ARRAY(SELECT AS STRUCT * FROM search_results ORDER BY Likes DESC, SingerId, AlbumId LIMIT 50) AS result_page`,
	},
}
