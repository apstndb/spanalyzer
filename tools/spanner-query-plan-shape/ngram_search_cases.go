package main

const ngramSearchDDL = `
CREATE TABLE NgramAlbums (
  AlbumId INT64 NOT NULL,
  AlbumTitle STRING(MAX),
  AlbumTitle_FuzzyTokens TOKENLIST AS (
    TOKENIZE_SUBSTRING(
      AlbumTitle,
      ngram_size_min=>2,
      ngram_size_max=>3,
      relative_search_types=>["word_prefix", "word_suffix"]
    )
  ) HIDDEN,
  AlbumTitle_PatternTokens TOKENLIST AS (
    TOKENIZE_NGRAMS(
      LOWER(AlbumTitle),
      ngram_size_min=>3,
      ngram_size_max=>4
    )
  ) HIDDEN,
) PRIMARY KEY(AlbumId);

CREATE SEARCH INDEX NgramAlbumsFuzzyIndex
ON NgramAlbums(AlbumTitle_FuzzyTokens)
STORING (AlbumTitle);

CREATE SEARCH INDEX NgramAlbumsPatternIndex
ON NgramAlbums(AlbumTitle_PatternTokens)
STORING (AlbumTitle);
`

var ngramSearchQueries = append([]queryCase{
	{
		Label: "ngram-search/fuzzy/search-only",
		SQL:   `SELECT AlbumId FROM NgramAlbums@{FORCE_INDEX=NgramAlbumsFuzzyIndex} WHERE SEARCH_NGRAMS(AlbumTitle_FuzzyTokens, "Hatel Kaliphorn")`,
	},
	{
		Label: "ngram-search/fuzzy/score-limit",
		SQL: `SELECT AlbumId
FROM NgramAlbums@{FORCE_INDEX=NgramAlbumsFuzzyIndex}
WHERE SEARCH_NGRAMS(AlbumTitle_FuzzyTokens, "Hatel Kaliphorn")
ORDER BY SCORE_NGRAMS(AlbumTitle_FuzzyTokens, "Hatel Kaliphorn") DESC
LIMIT 10`,
	},
}, append(buildQueryMatrixCases(
	"ngram-search/pattern",
	`SELECT AlbumId
FROM NgramAlbums@{FORCE_INDEX={{.access.index}}}
WHERE {{.predicate.expression}}`,
	queryMatrixAxis{Name: "predicate", Values: []queryMatrixAxisValue{
		{Label: "like-contains-literal", Fields: map[string]string{"expression": `AlbumTitle LIKE "%999%"`}},
		{Label: "starts-with-literal", Fields: map[string]string{"expression": `STARTS_WITH(AlbumTitle, "apple")`}},
		{Label: "ends-with-literal", Fields: map[string]string{"expression": `ENDS_WITH(AlbumTitle, "apple")`}},
		{Label: "regexp-contains-literal", Fields: map[string]string{"expression": `REGEXP_CONTAINS(AlbumTitle, r"(good|great)[ ]+morning")`}},
	}},
	queryMatrixAxis{Name: "access", Values: []queryMatrixAxisValue{
		{Label: "search-index", Fields: map[string]string{"index": "NgramAlbumsPatternIndex"}},
		{Label: "base-table", Fields: map[string]string{"index": "_BASE_TABLE"}},
	}},
), []queryCase{
	{
		Label:  "ngram-search/pattern/like-parameter/search-index",
		SQL:    `SELECT AlbumId FROM NgramAlbums@{FORCE_INDEX=NgramAlbumsPatternIndex} WHERE AlbumTitle LIKE @pattern`,
		Params: map[string]interface{}{"pattern": "%999%"},
	},
	{
		Label: "ngram-search/pattern/like-too-short/search-index",
		SQL:   `SELECT AlbumId FROM NgramAlbums@{FORCE_INDEX=NgramAlbumsPatternIndex} WHERE AlbumTitle LIKE "%ab%"`,
	},
}...)...)
