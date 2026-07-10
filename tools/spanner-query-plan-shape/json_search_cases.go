package main

const jsonSearchDDL = `
CREATE TABLE JSONSearchDocuments (
  TenantId INT64 NOT NULL,
  DocumentId INT64 NOT NULL,
  Title STRING(MAX),
  Body STRING(MAX),
  Metadata JSON,
  TitleTokens TOKENLIST AS (TOKENIZE_FULLTEXT(Title)) HIDDEN,
  MetadataTokens TOKENLIST AS (TOKENIZE_JSON(Metadata)) HIDDEN,
) PRIMARY KEY(TenantId, DocumentId);

CREATE SEARCH INDEX JSONSearchDocumentsByMetadata
ON JSONSearchDocuments(MetadataTokens)
STORING (Title);

CREATE SEARCH INDEX JSONSearchDocumentsByTitleMetadata
ON JSONSearchDocuments(TitleTokens, MetadataTokens)
STORING (Title);
`

var jsonSearchQueries = []queryCase{
	{
		Label: "json-search/containment-auto-index",
		SQL: `SELECT DocumentId FROM JSONSearchDocuments
WHERE JSON_CONTAINS(Metadata, JSON '{"labels":["large"]}')`,
	},
	{
		Label: "json-search/containment-force-index",
		SQL: `SELECT DocumentId, Title
FROM JSONSearchDocuments@{FORCE_INDEX=JSONSearchDocumentsByMetadata}
WHERE JSON_CONTAINS(
  Metadata,
  JSON '{"labels":["large"],"open":{"Friday":true}}')`,
	},
	{
		Label: "json-search/containment-base-table",
		SQL: `SELECT DocumentId
FROM JSONSearchDocuments@{FORCE_INDEX=_BASE_TABLE}
WHERE JSON_CONTAINS(Metadata, JSON '{"labels":["large"]}')`,
	},
	{
		Label: "json-search/nested-array-containment",
		SQL: `SELECT DocumentId
FROM JSONSearchDocuments@{FORCE_INDEX=JSONSearchDocumentsByMetadata}
WHERE JSON_CONTAINS(
  Metadata,
  JSON '{"RegionalReleases":[{"Region":"Japan"}]}')`,
	},
	{
		Label: "json-search/key-existence",
		SQL: `SELECT DocumentId
FROM JSONSearchDocuments@{FORCE_INDEX=JSONSearchDocumentsByMetadata}
WHERE Metadata.ReissueDate IS NOT NULL`,
	},
	{
		Label: "json-search/array-path-existence",
		SQL: `SELECT DocumentId
FROM JSONSearchDocuments@{FORCE_INDEX=JSONSearchDocumentsByMetadata}
WHERE Metadata.RegionalReleases[0].Region IS NOT NULL`,
	},
	{
		Label: "json-search/conjunction",
		SQL: `SELECT DocumentId
FROM JSONSearchDocuments@{FORCE_INDEX=JSONSearchDocumentsByMetadata}
WHERE JSON_CONTAINS(Metadata, JSON '{"labels":["large"]}')
  AND Metadata.ReissueDate IS NOT NULL`,
	},
	{
		Label: "json-search/disjunction-and-negation",
		SQL: `SELECT DocumentId
FROM JSONSearchDocuments@{FORCE_INDEX=JSONSearchDocumentsByMetadata}
WHERE JSON_CONTAINS(Metadata, JSON '{"labels":["large"]}')
  OR NOT JSON_CONTAINS(Metadata, JSON '{"open":{"Sunday":false}}')`,
	},
	{
		Label: "json-search/mixed-full-text-json",
		SQL: `SELECT DocumentId, Title
FROM JSONSearchDocuments@{FORCE_INDEX=JSONSearchDocumentsByTitleMetadata}
WHERE SEARCH(TitleTokens, "vector database")
  AND JSON_CONTAINS(Metadata, JSON '{"labels":["large"]}')`,
	},
	{
		Label: "json-search/stored-residual-filter",
		SQL: `SELECT DocumentId
FROM JSONSearchDocuments@{FORCE_INDEX=JSONSearchDocumentsByMetadata}
WHERE JSON_CONTAINS(Metadata, JSON '{"labels":["large"]}')
  AND Title = "Vector search"`,
	},
	{
		Label: "json-search/non-covering-back-join",
		SQL: `SELECT DocumentId, Body
FROM JSONSearchDocuments@{FORCE_INDEX=JSONSearchDocumentsByMetadata}
WHERE JSON_CONTAINS(Metadata, JSON '{"labels":["large"]}')`,
	},
}
