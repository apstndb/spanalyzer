package main

// Keep these as raw statements because the current memefish parser accepts
// CREATE VECTOR INDEX but not its newer extra key column syntax yet.
var vectorSearchDDLs = []string{
	`CREATE TABLE VectorDocuments (
  TenantId INT64 NOT NULL,
  DocumentId INT64 NOT NULL,
  Category STRING(MAX),
  Body STRING(MAX),
  Embedding ARRAY<FLOAT32>(vector_length=>3) NOT NULL,
  TechOnly BOOL AS (IF(Category = "tech", TRUE, NULL)) HIDDEN,
) PRIMARY KEY(TenantId, DocumentId)`,
	`CREATE VECTOR INDEX VectorDocumentsByEmbedding
ON VectorDocuments(Embedding, TenantId)
STORING (Category)
OPTIONS (distance_type = 'COSINE')`,
	`CREATE VECTOR INDEX TechVectorDocumentsByEmbedding
ON VectorDocuments(Embedding, TenantId)
STORING (TechOnly, Category)
WHERE TechOnly IS NOT NULL
OPTIONS (distance_type = 'COSINE')`,
}

var vectorSearchQueries = []queryCase{
	{
		Label: "vector-search/exact-knn-base-table",
		SQL: `SELECT DocumentId,
  COSINE_DISTANCE(Embedding, ARRAY<FLOAT32>[1.0, 0.0, 0.0]) AS Distance
FROM VectorDocuments@{FORCE_INDEX=_BASE_TABLE}
WHERE TenantId = 7
ORDER BY Distance
LIMIT 10`,
	},
	{
		Label: "vector-search/ann-auto-index",
		SQL: `SELECT DocumentId,
  APPROX_COSINE_DISTANCE(
    ARRAY<FLOAT32>[1.0, 0.0, 0.0], Embedding,
    options => JSON '{"num_leaves_to_search": 1}') AS Distance
FROM VectorDocuments
ORDER BY Distance
LIMIT 10`,
	},
	{
		Label: "vector-search/ann-extra-key-filter",
		SQL: `SELECT DocumentId, Category,
  APPROX_COSINE_DISTANCE(
    ARRAY<FLOAT32>[1.0, 0.0, 0.0], Embedding,
    options => JSON '{"num_leaves_to_search": 1}') AS Distance
FROM VectorDocuments@{FORCE_INDEX=VectorDocumentsByEmbedding}
WHERE TenantId = 7
ORDER BY Distance
LIMIT 10`,
	},
	{
		Label: "vector-search/ann-stored-filter",
		SQL: `SELECT DocumentId,
  APPROX_COSINE_DISTANCE(
    ARRAY<FLOAT32>[1.0, 0.0, 0.0], Embedding,
    options => JSON '{"num_leaves_to_search": 1}') AS Distance
FROM VectorDocuments@{FORCE_INDEX=VectorDocumentsByEmbedding}
WHERE Category = "tech"
ORDER BY Distance
LIMIT 10`,
	},
	{
		Label: "vector-search/ann-back-join",
		SQL: `SELECT DocumentId, Body,
  APPROX_COSINE_DISTANCE(
    ARRAY<FLOAT32>[1.0, 0.0, 0.0], Embedding,
    options => JSON '{"num_leaves_to_search": 1}') AS Distance
FROM VectorDocuments@{FORCE_INDEX=VectorDocumentsByEmbedding}
WHERE TenantId = 7
ORDER BY Distance
LIMIT 10`,
	},
	{
		Label: "vector-search/ann-filtered-index",
		SQL: `SELECT DocumentId, Category,
  APPROX_COSINE_DISTANCE(
    ARRAY<FLOAT32>[1.0, 0.0, 0.0], Embedding,
    options => JSON '{"num_leaves_to_search": 1}') AS Distance
FROM VectorDocuments@{FORCE_INDEX=TechVectorDocumentsByEmbedding}
WHERE TenantId = 7 AND TechOnly IS NOT NULL
ORDER BY Distance
LIMIT 10`,
	},
}
