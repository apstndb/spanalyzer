package main

const conditionBoundaryDDL = docsDDL + `
CREATE TABLE CommitTimestampKeys (
  CommitTs TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
  RowId INT64 NOT NULL,
  Payload STRING(MAX),
) PRIMARY KEY(CommitTs, RowId);
`

// conditionBoundaryQueries compares predicates that are logically similar but
// cross a physical-plan boundary between split extraction, scan seeking,
// residual filtering, and join-key evaluation.
var conditionBoundaryQueries = []queryCase{
	{Label: "condition-boundary/scan/equality-literal", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE SongName = 'Alpha'`},
	{Label: "condition-boundary/scan/range-literal", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE SongName >= 'A' AND SongName < 'B'`},
	{Label: "condition-boundary/scan/starts-with-literal", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE STARTS_WITH(SongName, 'A')`},
	{Label: "condition-boundary/scan/starts-with-parameter", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE STARTS_WITH(SongName, @prefix)`, Params: map[string]interface{}{"prefix": "A"}},
	{Label: "condition-boundary/scan/like-prefix-literal", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE SongName LIKE 'A%'`},
	{Label: "condition-boundary/scan/like-prefix-parameter", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE SongName LIKE @pattern`, Params: map[string]interface{}{"pattern": "A%"}},
	{Label: "condition-boundary/scan/like-contains-literal", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE SongName LIKE '%A%'`},
	{Label: "condition-boundary/scan/regexp-prefix", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE REGEXP_CONTAINS(SongName, r'^A.*')`},
	{Label: "condition-boundary/scan/regexp-prefix-with-residual", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE REGEXP_CONTAINS(SongName, r'^A.B')`},
	{Label: "condition-boundary/scan/regexp-contains", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE REGEXP_CONTAINS(SongName, r'A')`},
	{Label: "condition-boundary/scan/strpos-contains", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE STRPOS(SongName, 'A') > 0`},
	{Label: "condition-boundary/scan/ends-with", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE ENDS_WITH(SongName, 'A')`},
	{Label: "condition-boundary/scan/substr-prefix", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE SUBSTR(SongName, 1, 1) = 'A'`},
	{Label: "condition-boundary/scan/lower-key", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE LOWER(SongName) = 'a'`},
	{Label: "condition-boundary/scan/foldable-constant", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE SongName = UPPER('a')`},
	{Label: "condition-boundary/scan/or-seekable-prefixes", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE STARTS_WITH(SongName, 'A') OR STARTS_WITH(SongName, 'B')`},
	{Label: "condition-boundary/scan/seek-plus-residual", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE STARTS_WITH(SongName, 'A') AND STRPOS(SongName, 'x') > 0`},
	{Label: "condition-boundary/scan/like-prefix-with-residual", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE SongName LIKE 'A_B%'`},
	{Label: "condition-boundary/scan/or-seekable-and-residual", SQL: `SELECT SongName FROM Songs@{FORCE_INDEX=SongsBySongName} WHERE STARTS_WITH(SongName, 'A') OR ENDS_WITH(SongName, 'B')`},
	{Label: "condition-boundary/timestamp-key/pushdown-true", SQL: `@{ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=TRUE}
SELECT Payload FROM CommitTimestampKeys WHERE CommitTs > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 12 HOUR)`},
	{Label: "condition-boundary/timestamp-key/pushdown-false", SQL: `@{ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=FALSE}
SELECT Payload FROM CommitTimestampKeys WHERE CommitTs > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 12 HOUR)`},

	{Label: "condition-boundary/join/hash-equality", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "condition-boundary/join/hash-two-equalities", SQL: `SELECT a.AlbumTitle, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId`},
	{Label: "condition-boundary/join/hash-cast-equality", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON CAST(a.SingerId AS STRING) = CAST(s.SingerId AS STRING)`},
	{Label: "condition-boundary/join/hash-arithmetic-equality", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON a.SingerId + 1 = s.SingerId`},
	{Label: "condition-boundary/join/hash-lower-equality", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON LOWER(a.AlbumTitle) = LOWER(s.SongName)`},
	{Label: "condition-boundary/join/hash-null-safe-equality", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON a.AlbumTitle IS NOT DISTINCT FROM s.SongName`},
	{Label: "condition-boundary/join/hash-key-plus-inequality", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON a.SingerId = s.SingerId AND a.AlbumId < s.AlbumId`},
	{Label: "condition-boundary/join/hash-key-plus-starts-with", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON a.SingerId = s.SingerId AND STARTS_WITH(s.SongName, a.AlbumTitle)`},
	{Label: "condition-boundary/join/hash-key-plus-strpos", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON a.SingerId = s.SingerId AND STRPOS(s.SongName, a.AlbumTitle) > 0`},
	{Label: "condition-boundary/join/hash-key-plus-like-contains", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON a.SingerId = s.SingerId AND s.SongName LIKE CONCAT('%', a.AlbumTitle, '%')`},
	{Label: "condition-boundary/join/hash-inequality-only", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON a.AlbumId < s.AlbumId`},
	{Label: "condition-boundary/join/hash-or-equalities", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON a.SingerId = s.SingerId OR a.AlbumId = s.AlbumId`},
	{Label: "condition-boundary/join/hash-key-plus-one-side-seek", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs@{FORCE_INDEX=SongsBySongName} AS s ON a.SingerId = s.SingerId AND STARTS_WITH(s.SongName, 'A')`},
	{Label: "condition-boundary/join/merge-lower-equality", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=MERGE_JOIN} Songs AS s ON LOWER(a.AlbumTitle) = LOWER(s.SongName)`},
	{Label: "condition-boundary/join/merge-key-plus-inequality", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=MERGE_JOIN} Songs AS s ON a.SingerId = s.SingerId AND a.AlbumId < s.AlbumId`},
	{Label: "condition-boundary/join/apply-equality", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "condition-boundary/join/apply-key-plus-range", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} Songs AS s ON a.SingerId = s.SingerId AND a.AlbumId < s.AlbumId`},
}
