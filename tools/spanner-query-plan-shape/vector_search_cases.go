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
	`CREATE VECTOR INDEX VectorDocumentsByDotProduct
ON VectorDocuments(Embedding, TenantId)
OPTIONS (distance_type = 'DOT_PRODUCT')`,
	`CREATE VECTOR INDEX VectorDocumentsByEuclidean
ON VectorDocuments(Embedding, TenantId)
OPTIONS (distance_type = 'EUCLIDEAN')`,
	`CREATE TABLE VectorRelated (
  SourceTenantId INT64 NOT NULL,
  SourceDocumentId INT64 NOT NULL,
  TargetTenantId INT64 NOT NULL,
  TargetDocumentId INT64 NOT NULL,
) PRIMARY KEY(SourceTenantId, SourceDocumentId, TargetTenantId, TargetDocumentId)`,
	`CREATE PROPERTY GRAPH VectorGraph
  NODE TABLES(
    VectorDocuments
      KEY(TenantId, DocumentId)
      LABEL Document PROPERTIES(TenantId, DocumentId, Category, Embedding)
  )
  EDGE TABLES(
    VectorRelated AS Related
      KEY(SourceTenantId, SourceDocumentId, TargetTenantId, TargetDocumentId)
      SOURCE KEY(SourceTenantId, SourceDocumentId)
        REFERENCES VectorDocuments(TenantId, DocumentId)
      DESTINATION KEY(TargetTenantId, TargetDocumentId)
        REFERENCES VectorDocuments(TenantId, DocumentId)
      LABEL Related PROPERTIES ALL COLUMNS
)`,
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
	{
		Label: "vector-search/ann-dot-product",
		SQL: `SELECT DocumentId,
  APPROX_DOT_PRODUCT(
    Embedding, ARRAY<FLOAT32>[1.0, 0.0, 0.0],
    options => JSON '{"num_leaves_to_search": 1}') AS Distance
FROM VectorDocuments@{FORCE_INDEX=VectorDocumentsByDotProduct}
ORDER BY Distance DESC
LIMIT 10`,
	},
	{
		Label: "vector-search/ann-euclidean-distance",
		SQL: `SELECT DocumentId,
  APPROX_EUCLIDEAN_DISTANCE(
    Embedding, ARRAY<FLOAT32>[1.0, 0.0, 0.0],
    options => JSON '{"num_leaves_to_search": 1}') AS Distance
FROM VectorDocuments@{FORCE_INDEX=VectorDocumentsByEuclidean}
ORDER BY Distance
LIMIT 10`,
	},
	{
		Label: "vector-search/ann-gql-next-traversal",
		SQL: `GRAPH VectorGraph
MATCH (@{FORCE_INDEX=VectorDocumentsByEmbedding} d:Document)
WHERE d.TenantId = 7
RETURN d,
  APPROX_COSINE_DISTANCE(
    ARRAY<FLOAT32>[1.0, 0.0, 0.0], d.Embedding,
    options => JSON '{"num_leaves_to_search": 1}') AS distance
ORDER BY distance
LIMIT 10
NEXT
MATCH (d)-[:Related]->(r:Document)
RETURN d.DocumentId AS document_id, r.DocumentId AS related_id, distance`,
	},
}
