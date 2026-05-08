package main

import "strings"

const docsDDL = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(1024),
  LastName STRING(1024),
  BirthDate DATE,
  SingerInfo BYTES(MAX),
  ReleaseDate DATE,
  ModificationTime TIMESTAMP OPTIONS (allow_commit_timestamp = true),
) PRIMARY KEY(SingerId);

CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName);
CREATE INDEX SingersByLastName ON Singers(LastName) STORING (FirstName);

CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  AlbumTitle STRING(MAX),
  MarketingBudget INT64,
  ReleaseDate DATE,
) PRIMARY KEY(SingerId, AlbumId),
  INTERLEAVE IN PARENT Singers ON DELETE CASCADE;

CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle);
CREATE INDEX AlbumsByAlbumTitle2 ON Albums(AlbumTitle) STORING (MarketingBudget);
CREATE INDEX AlbumsByReleaseDateTitleDesc ON Albums(ReleaseDate, AlbumTitle DESC);

CREATE TABLE Songs (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  TrackId INT64 NOT NULL,
  SongName STRING(MAX),
  Duration INT64,
  SongGenre STRING(25),
) PRIMARY KEY(SingerId, AlbumId, TrackId),
  INTERLEAVE IN PARENT Albums ON DELETE CASCADE;

CREATE INDEX SongsBySingerAlbumSongNameDesc ON Songs(SingerId, AlbumId, SongName DESC),
  INTERLEAVE IN Albums;
CREATE INDEX SongsBySongName ON Songs(SongName);

CREATE TABLE Concerts (
  VenueId INT64 NOT NULL,
  SingerId INT64 NOT NULL,
  ConcertDate DATE NOT NULL,
  BeginTime TIMESTAMP,
  EndTime TIMESTAMP,
  TicketPrices ARRAY<INT64>,
) PRIMARY KEY(VenueId, SingerId, ConcertDate);

CREATE TABLE Collaborations (
  SingerId INT64 NOT NULL,
  FeaturingSingerId INT64 NOT NULL,
  AlbumTitle STRING(MAX) NOT NULL,
) PRIMARY KEY(SingerId, FeaturingSingerId, AlbumTitle);

CREATE OR REPLACE PROPERTY GRAPH MusicGraph
  NODE TABLES(
    Singers
      KEY(SingerId)
      LABEL Singers PROPERTIES(
        BirthDate,
        FirstName,
        LastName,
        SingerId,
        SingerInfo)
  )
  EDGE TABLES(
    Collaborations AS CollabWith
      KEY(SingerId, FeaturingSingerId, AlbumTitle)
      SOURCE KEY(SingerId) REFERENCES Singers(SingerId)
      DESTINATION KEY(FeaturingSingerId) REFERENCES Singers(SingerId)
      LABEL CollabWith PROPERTIES(
        AlbumTitle,
        FeaturingSingerId,
        SingerId)
  );
`

const dmlDDL = docsDDL + `
ALTER TABLE Singers ADD COLUMN Status STRING(1024) DEFAULT ("active");
ALTER TABLE Singers ADD COLUMN LastUpdated TIMESTAMP DEFAULT (PENDING_COMMIT_TIMESTAMP())
  ON UPDATE (PENDING_COMMIT_TIMESTAMP())
  OPTIONS (allow_commit_timestamp = true);
CREATE UNIQUE INDEX UniqueIndex_SingerName ON Singers(FirstName, LastName);

CREATE TABLE AckworthSingers (
  SingerId INT64 NOT NULL,
  FirstName STRING(1024),
  LastName STRING(1024),
  BirthDate DATE,
) PRIMARY KEY(SingerId);

CREATE TABLE Fans (
  FanId STRING(36) DEFAULT (GENERATE_UUID()),
  FirstName STRING(1024),
  LastName STRING(1024),
) PRIMARY KEY(FanId);
`

const optimizerGapsDDL = dmlDDL + `
CREATE TABLE Venues (
  VenueId INT64 NOT NULL,
  VenueName STRING(MAX),
) PRIMARY KEY(VenueId);

CREATE TABLE FKCustomers (
  CustomerId INT64 NOT NULL,
  CustomerName STRING(MAX) NOT NULL,
) PRIMARY KEY(CustomerId);

CREATE TABLE FKOrders (
  OrderId INT64 NOT NULL,
  CustomerId INT64 NOT NULL,
  Quantity INT64 NOT NULL,
  ProductId INT64 NOT NULL,
  CONSTRAINT FK_CustomerOrder FOREIGN KEY (CustomerId)
    REFERENCES FKCustomers (CustomerId) NOT ENFORCED,
) PRIMARY KEY(OrderId);
`

const changeStreamTVFDDL = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(1024),
) PRIMARY KEY(SingerId);

CREATE CHANGE STREAM EverythingStream
FOR ALL;
`

var docsQueries = []queryCase{
	{
		Label: "execution-plans/simple-scan",
		SQL:   `SELECT s.SongName FROM Songs AS s`,
	},
	{
		Label: "execution-plans/aggregate",
		SQL:   `SELECT s.SingerId, COUNT(*) AS SongCount FROM Songs AS s WHERE s.SingerId < 100 GROUP BY s.SingerId`,
	},
	{
		Label: "execution-plans/join",
		SQL:   `SELECT al.AlbumTitle, so.SongName FROM Albums AS al, Songs AS so WHERE al.SingerId = so.SingerId AND al.AlbumId = so.AlbumId`,
	},
	{
		Label: "execution-plans/index-with-back-join",
		SQL:   `SELECT s.SongName, s.Duration FROM Songs@{FORCE_INDEX=SongsBySongName} AS s WHERE STARTS_WITH(s.SongName, "B")`,
	},
	{
		Label: "execution-plans/index-only",
		SQL:   `SELECT s.SongName FROM Songs@{FORCE_INDEX=SongsBySongName} AS s WHERE STARTS_WITH(s.SongName, "B")`,
	},
	{
		Label: "leaf/array-unnest",
		SQL:   `SELECT a, b FROM UNNEST([1,2,3]) a WITH OFFSET b`,
	},
	{
		Label: "leaf/generate-relation",
		SQL:   `SELECT 1 + 2 AS Result`,
	},
	{
		Label: "leaf/empty-relation",
		SQL:   `SELECT * FROM Albums LIMIT 0`,
	},
	{
		Label: "leaf/filter-scan-index",
		SQL:   `SELECT s.LastName FROM Singers@{FORCE_INDEX=SingersByFirstLastName} AS s WHERE s.FirstName = 'Catalina'`,
	},
	{
		Label: "leaf/filter-scan-table",
		SQL:   `SELECT LastName FROM Singers WHERE SingerId = 1`,
	},
	{
		Label: "unary/aggregate",
		SQL:   `SELECT s.SingerId, AVG(s.Duration) AS average, COUNT(*) AS count FROM Songs AS s GROUP BY SingerId`,
	},
	{
		Label: "unary/compute-struct",
		SQL:   `SELECT FirstName, ARRAY(SELECT AS STRUCT song.SongName, song.SongGenre FROM Songs AS song WHERE song.SingerId = singer.SingerId) FROM Singers AS singer WHERE singer.SingerId = 1`,
	},
	{
		Label: "unary/filter",
		SQL:   `SELECT s.LastName FROM (SELECT s.LastName FROM Singers AS s LIMIT 3) s WHERE s.LastName LIKE 'Rich%'`,
	},
	{
		Label: "unary/limit",
		SQL:   `SELECT s.SongName FROM Songs AS s LIMIT 3`,
	},
	{
		Label: "unary/tablesample-bernoulli",
		SQL:   `SELECT s.SongName FROM Songs AS s TABLESAMPLE BERNOULLI (10 PERCENT)`,
	},
	{
		Label: "unary/tablesample-reservoir",
		SQL:   `SELECT s.SongName FROM Songs AS s TABLESAMPLE RESERVOIR (2 ROWS)`,
	},
	{
		Label: "unary/row-to-datablock",
		SQL:   `SELECT BirthDate FROM Singers`,
	},
	{
		Label: "unary/serialize-result-array",
		SQL:   `SELECT ARRAY(SELECT AS STRUCT so.SongName, so.SongGenre FROM Songs AS so WHERE so.SingerId = s.SingerId) FROM Singers AS s`,
	},
	{
		Label: "unary/sort",
		SQL:   `SELECT s.SongGenre FROM Songs AS s ORDER BY SongGenre`,
	},
	{
		Label: "unary/sort-limit",
		SQL:   `SELECT s.SongGenre FROM Songs AS s ORDER BY SongGenre LIMIT 3`,
	},
	{
		Label: "unary/minor-sort-order-by-partial-key",
		SQL:   `SELECT SingerId, AlbumTitle FROM Albums ORDER BY SingerId, AlbumTitle`,
	},
	{
		Label: "unary/minor-sort-limit-order-by-partial-key",
		SQL:   `SELECT SingerId, AlbumTitle FROM Albums WHERE SingerId > 0 ORDER BY SingerId, AlbumTitle LIMIT 3`,
	},
	{
		Label: "unary/minor-sort-stream-aggregate",
		SQL:   `SELECT SingerId, SongGenre FROM Songs GROUP@{GROUP_METHOD=STREAM_GROUP} BY SingerId, SongGenre`,
	},
	{
		Label: "binary/cross-apply",
		SQL:   `SELECT si.FirstName, (SELECT so.SongName FROM Songs AS so WHERE so.SingerId = si.SingerId LIMIT 1) FROM Singers AS si`,
	},
	{
		Label: "binary/in-subquery",
		SQL:   `SELECT FirstName, LastName FROM Singers WHERE SingerId IN (SELECT SingerId FROM Albums)`,
	},
	{
		Label: "binary/not-in-subquery",
		SQL:   `SELECT FirstName, LastName FROM Singers WHERE SingerId NOT IN (SELECT SingerId FROM Albums)`,
	},
	{
		Label: "binary/hash-join",
		SQL:   `SELECT a.AlbumTitle, s.SongName FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN} Songs AS s ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId`,
	},
	{
		Label: "binary/merge-join",
		SQL:   `SELECT a.AlbumTitle, s.SongName FROM Albums AS a JOIN@{JOIN_METHOD=MERGE_JOIN} Songs AS s ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId`,
	},
	{
		Label: "binary/merge-join-with-sort",
		SQL:   `SELECT a.AlbumTitle, s.SongName FROM Albums AS a JOIN@{JOIN_METHOD=MERGE_JOIN} Songs AS s ON a.AlbumId = s.AlbumId`,
	},
	{
		Label: "binary/recursive-union-graph",
		SQL:   `GRAPH MusicGraph MATCH (singer:Singers {singerId:42})-[c:CollabWith]->{1,2}(featured:Singers) RETURN singer.SingerId AS singer, featured.SingerId AS featured`,
	},
	{
		Label: "n-ary/union-all",
		SQL:   `SELECT 1 a, 2 b UNION ALL SELECT 3 a, 4 b UNION ALL SELECT 5 a, 6 b`,
	},
	{
		Label: "n-ary/union-all-different-names",
		SQL:   `SELECT 1 a, 2 b UNION ALL SELECT 3 c, 4 e`,
	},
	{
		Label: "distributed/distributed-union",
		SQL:   `SELECT s.SongName, s.SongGenre FROM Songs AS s WHERE s.SingerId = 2 AND s.SongGenre = 'ROCK'`,
	},
	{
		Label: "distributed/distributed-apply",
		SQL:   `SELECT AlbumTitle FROM Songs JOIN Albums ON Albums.AlbumId = Songs.AlbumId`,
	},
	{
		Label: "distributed/distributed-merge-union",
		SQL:   `SELECT s.SongGenre FROM Songs AS s ORDER BY SongGenre`,
	},
	{
		Label: "distributed/push-broadcast-hash-join",
		SQL:   `SELECT a.AlbumTitle, s.SongName FROM Albums AS a JOIN@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} Songs AS s ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId`,
	},
	{
		Label: "scalar-subquery/conditional",
		SQL:   `SELECT FirstName, IF(FirstName = 'Alice', (SELECT COUNT(*) FROM Songs WHERE Duration > 300), 0) FROM Singers`,
	},
	{
		Label: "scalar-subquery/filter",
		SQL:   `SELECT * FROM Songs WHERE Duration = (SELECT MAX(Duration) FROM Songs)`,
	},
	{
		Label: "array-subquery",
		SQL:   `SELECT a.AlbumId, ARRAY(SELECT ConcertDate FROM Concerts WHERE Concerts.SingerId = a.SingerId) FROM Albums AS a`,
	},
	{
		Label: "struct-constructor",
		SQL:   `SELECT IF(TRUE, STRUCT(1 AS A, 1 AS B), STRUCT(2 AS A, 2 AS B)).A`,
	},
	{
		Label: "cte/spool-build",
		SQL:   `WITH CTE AS (SELECT 1 AS PK, "foo" AS col) SELECT * FROM CTE c1 JOIN CTE c2 USING (PK)`,
	},
	{
		Label: "best-practices/index-auto",
		SQL:   `SELECT s.SingerId FROM Singers AS s WHERE s.LastName = 'Smith'`,
	},
	{
		Label: "best-practices/force-index",
		SQL:   `SELECT s.SingerId FROM Singers@{FORCE_INDEX=SingersByLastName} AS s WHERE s.LastName = 'Smith'`,
	},
	{
		Label: "best-practices/force-index-covering",
		SQL:   `SELECT s.SingerId, s.FirstName FROM Singers@{FORCE_INDEX=SingersByLastName} AS s WHERE s.LastName = 'Smith'`,
	},
	{
		Label: "best-practices/order-by-auto-index",
		SQL:   `SELECT a.AlbumTitle, a.ReleaseDate FROM Albums AS a ORDER BY a.ReleaseDate, a.AlbumTitle DESC`,
	},
	{
		Label: "best-practices/order-by-index",
		SQL:   `SELECT a.AlbumTitle, a.ReleaseDate FROM Albums@{FORCE_INDEX=AlbumsByReleaseDateTitleDesc} AS a ORDER BY a.ReleaseDate, a.AlbumTitle DESC`,
	},
	{
		Label: "best-practices/order-by-desc-limit-back-join-optimizer-version-5",
		SQL:   `@{OPTIMIZER_VERSION=5} SELECT * FROM Songs@{FORCE_INDEX=SongsBySongName} ORDER BY SongName DESC LIMIT 1`,
	},
	{
		Label: "best-practices/order-by-primary-key",
		SQL:   `SELECT * FROM Singers ORDER BY SingerId`,
	},
	{
		Label: "best-practices/no-order-by",
		SQL:   `SELECT * FROM Singers`,
	},
	{
		Label: "best-practices/like",
		SQL:   `SELECT a.AlbumTitle FROM Albums a WHERE a.AlbumTitle LIKE 'Blue%'`,
	},
	{
		Label: "best-practices/starts-with",
		SQL:   `SELECT a.AlbumTitle FROM Albums a WHERE STARTS_WITH(a.AlbumTitle, 'Blue')`,
	},
}

var statementHintQueryMatrixQueries = buildStatementHintQueryMatrixQueries()
var functionHintQueries = buildFunctionHintQueries()
var optimizerUnhintedCandidateQueries = buildOptimizerUnhintedCandidateQueries()

var optimizerGapQueries = []queryCase{
	{
		Label: "optimizer-gaps/v8/with-large-in-join-order-limit",
		SQL: `WITH CandidateSingers AS (
  SELECT SingerId, FirstName
  FROM Singers
  WHERE SingerId IN (1, 2, 3, 5, 8, 13, 21, 34, 55, 89, 144, 233, 377, 610, 987, 1597)
)
SELECT c.FirstName, a.AlbumTitle
FROM CandidateSingers AS c
JOIN Albums AS a USING (SingerId)
WHERE a.ReleaseDate >= DATE '2020-01-01'
ORDER BY a.ReleaseDate, a.AlbumTitle
LIMIT 10`,
	},
	{
		Label: "optimizer-gaps/v8/large-in-list",
		SQL:   `SELECT SingerId, FirstName FROM Singers WHERE SingerId IN (1, 2, 3, 5, 8, 13, 21, 34, 55, 89, 144, 233, 377, 610, 987, 1597, 2584, 4181, 6765, 10946, 17711, 28657, 46368, 75025)`,
	},
	{
		Label: "optimizer-gaps/v8/use-unenforced-foreign-key-true",
		SQL: `@{USE_UNENFORCED_FOREIGN_KEY=TRUE}
SELECT o.CustomerId
FROM FKOrders AS o
JOIN FKCustomers AS c ON c.CustomerId = o.CustomerId`,
	},
	{
		Label: "optimizer-gaps/v8/use-unenforced-foreign-key-false",
		SQL: `@{USE_UNENFORCED_FOREIGN_KEY=FALSE}
SELECT o.CustomerId
FROM FKOrders AS o
JOIN FKCustomers AS c ON c.CustomerId = o.CustomerId`,
	},
	{
		Label: "optimizer-gaps/v7/unhinted-index-union-candidate",
		SQL:   `SELECT s.SingerId FROM Singers AS s WHERE s.FirstName = 'Alice' OR s.LastName = 'Smith'`,
	},
	{
		Label: "optimizer-gaps/v7/partial-key-seek-candidate",
		SQL:   `SELECT a.AlbumTitle FROM Albums@{FORCE_INDEX=AlbumsByReleaseDateTitleDesc} AS a WHERE a.ReleaseDate >= DATE '2020-01-01' ORDER BY a.ReleaseDate, a.AlbumTitle DESC`,
	},
	{
		Label:    "optimizer-gaps/v6/dml-insert-select-filter",
		SQL:      `INSERT INTO Singers (SingerId, FirstName, LastName) SELECT SingerId, FirstName, LastName FROM AckworthSingers WHERE SingerId < 100`,
		PlanMode: planModeReadWrite,
	},
	{
		Label: "optimizer-gaps/v6/full-outer-join-predicate-limit",
		SQL: `SELECT a.SingerId, a.AlbumTitle, s.SongName
FROM Albums AS a
FULL OUTER JOIN Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
WHERE a.ReleaseDate >= DATE '2020-01-01' OR s.Duration > 180
LIMIT 10`,
	},
	{
		Label: "optimizer-gaps/v4/interleaved-secondary-index-join",
		SQL: `SELECT s.SingerId, s.SongName
FROM Songs@{FORCE_INDEX=SongsBySingerAlbumSongNameDesc} AS s
JOIN Albums AS a
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
WHERE s.SingerId = 1 AND s.AlbumId = 1`,
	},
	{
		Label: "optimizer-gaps/v3/group-by-prefix-no-extra-column",
		SQL:   `SELECT s.SingerId, s.AlbumId FROM Songs AS s GROUP BY s.SingerId, s.AlbumId`,
	},
	{
		Label: "optimizer-gaps/v3/sorted-limit-cross-apply",
		SQL: `SELECT a2.*
FROM Albums@{FORCE_INDEX=_BASE_TABLE} AS a1
JOIN Albums@{FORCE_INDEX=_BASE_TABLE} AS a2 USING(SingerId)
ORDER BY a1.AlbumId
LIMIT 2`,
	},
	{
		Label: "optimizer-gaps/v3/push-computation-through-join",
		SQL: `SELECT
  t.ConcertDate,
  (SELECT COUNT(*) FROM UNNEST(t.TicketPrices) AS p WHERE p > 10) AS expensive_tickets,
  u.VenueName
FROM Concerts AS t
JOIN Venues AS u ON t.VenueId = u.VenueId
ORDER BY expensive_tickets
LIMIT 2`,
	},
	{
		Label: "optimizer-gaps/v2/regexp-contains-prefix",
		SQL:   `SELECT a.AlbumTitle FROM Albums AS a WHERE REGEXP_CONTAINS(a.AlbumTitle, '^Blue')`,
	},
	{
		Label: "optimizer-gaps/v2/regexp-contains-prefix-forced-index",
		SQL: `SELECT SingerId, AlbumId, TrackId
FROM Songs@{FORCE_INDEX=SongsBySongName}
WHERE REGEXP_CONTAINS(SongName, "^A.*")`,
	},
	{
		Label: "optimizer-gaps/v2/like-prefix-forced-index",
		SQL: `SELECT SingerId, AlbumId, TrackId
FROM Songs@{FORCE_INDEX=SongsBySongName}
WHERE SongName LIKE "A%z"`,
	},
	{
		Label: "optimizer-gaps/v2/group-by-scan-prefix",
		SQL:   `SELECT s.SingerId, s.AlbumId FROM Songs AS s GROUP BY s.SingerId, s.AlbumId`,
	},
}

func buildOptimizerUnhintedCandidateQueries() []queryCase {
	inputs := append([]queryCase{}, docsQueries...)
	inputs = append(inputs, optimizerGapQueries...)
	out := make([]queryCase, 0, len(inputs))
	seenSQL := map[string]bool{}
	for _, query := range inputs {
		sql := stripGoogleSQLHints(query.SQL)
		key := strings.Join(strings.Fields(sql), " ")
		if seenSQL[key] {
			continue
		}
		seenSQL[key] = true
		out = append(out, queryCase{
			Label:    "optimizer-unhinted-candidates/" + query.Label,
			SQL:      sql,
			PlanMode: query.PlanMode,
		})
	}
	return out
}

func buildFunctionHintQueries() []queryCase {
	hintAxis := queryMatrixAxis{
		Name: "Hint",
		Values: []queryMatrixAxisValue{
			{Label: "default_inline", Fields: map[string]string{"SQL": ""}},
			{Label: "disable_inline_false", Fields: map[string]string{"SQL": " @{DISABLE_INLINE=FALSE}"}},
			{Label: "disable_inline_true", Fields: map[string]string{"SQL": " @{DISABLE_INLINE=TRUE}"}},
		},
	}
	return buildQueryMatrixCases("function-hint", `
SELECT
  SUBSTRING(CAST(x AS STRING), 2, 5) AS w,
  SUBSTRING(CAST(x AS STRING), 3, 7) AS y
FROM (
  SELECT SHA512(s.SingerInfo){{.Hint.SQL}} AS x
  FROM Singers AS s
)
`, hintAxis)
}

type statementHintVariant struct {
	Label       string
	Assignments []string
}

func buildStatementHintQueryMatrixQueries() []queryCase {
	var out []queryCase
	for _, hint := range documentedStatementHintVariants() {
		for _, query := range docsQueries {
			out = append(out, queryCase{
				Label:    "statement-hint-query-matrix/" + hint.Label + "/" + query.Label,
				SQL:      withStatementHintAssignments(query.SQL, hint.Assignments...),
				PlanMode: query.PlanMode,
			})
		}
	}
	return out
}

func documentedStatementHintVariants() []statementHintVariant {
	return []statementHintVariant{
		{Label: "use_additional_parallelism_true", Assignments: []string{"USE_ADDITIONAL_PARALLELISM=TRUE"}},
		{Label: "use_additional_parallelism_false", Assignments: []string{"USE_ADDITIONAL_PARALLELISM=FALSE"}},
		{Label: "optimizer_version_default", Assignments: []string{"OPTIMIZER_VERSION=default_version"}},
		{Label: "optimizer_version_latest", Assignments: []string{"OPTIMIZER_VERSION=latest_version"}},
		{Label: "optimizer_statistics_package_latest", Assignments: []string{"OPTIMIZER_STATISTICS_PACKAGE=latest"}},
		{Label: "allow_distributed_merge_true", Assignments: []string{"ALLOW_DISTRIBUTED_MERGE=TRUE"}},
		{Label: "allow_distributed_merge_false", Assignments: []string{"ALLOW_DISTRIBUTED_MERGE=FALSE"}},
		{Label: "lock_scanned_ranges_exclusive", Assignments: []string{"LOCK_SCANNED_RANGES=exclusive"}},
		{Label: "lock_scanned_ranges_shared", Assignments: []string{"LOCK_SCANNED_RANGES=shared"}},
		{Label: "scan_method_batch", Assignments: []string{"SCAN_METHOD=BATCH"}},
		{Label: "scan_method_row", Assignments: []string{"SCAN_METHOD=ROW"}},
		{Label: "scan_method_columnar", Assignments: []string{"SCAN_METHOD=COLUMNAR"}},
		{Label: "scan_method_no_columnar", Assignments: []string{"SCAN_METHOD=NO_COLUMNAR"}},
		{Label: "execution_method_batch", Assignments: []string{"EXECUTION_METHOD=BATCH"}},
		{Label: "execution_method_row", Assignments: []string{"EXECUTION_METHOD=ROW"}},
		{Label: "use_unenforced_foreign_key_true", Assignments: []string{"USE_UNENFORCED_FOREIGN_KEY=TRUE"}},
		{Label: "use_unenforced_foreign_key_false", Assignments: []string{"USE_UNENFORCED_FOREIGN_KEY=FALSE"}},
		{Label: "allow_timestamp_predicate_pushdown_true", Assignments: []string{"ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=TRUE"}},
		{Label: "allow_timestamp_predicate_pushdown_false", Assignments: []string{"ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=FALSE"}},
		{Label: "force_index_base_table", Assignments: []string{"FORCE_INDEX=_BASE_TABLE"}},
		{Label: "groupby_scan_optimization_true", Assignments: []string{"GROUPBY_SCAN_OPTIMIZATION=TRUE"}},
		{Label: "groupby_scan_optimization_false", Assignments: []string{"GROUPBY_SCAN_OPTIMIZATION=FALSE"}},
		{Label: "index_strategy_force_index_union", Assignments: []string{"INDEX_STRATEGY=FORCE_INDEX_UNION"}},
		{Label: "seekable_key_size_zero", Assignments: []string{"FORCE_INDEX=_BASE_TABLE", "SEEKABLE_KEY_SIZE=0"}},
		{Label: "seekable_key_size_one", Assignments: []string{"FORCE_INDEX=_BASE_TABLE", "SEEKABLE_KEY_SIZE=1"}},
		{Label: "seekable_key_size_two", Assignments: []string{"FORCE_INDEX=_BASE_TABLE", "SEEKABLE_KEY_SIZE=2"}},
		{Label: "force_join_order_true", Assignments: []string{"FORCE_JOIN_ORDER=TRUE"}},
		{Label: "force_join_order_false", Assignments: []string{"FORCE_JOIN_ORDER=FALSE"}},
		{Label: "join_method_hash_join", Assignments: []string{"JOIN_METHOD=HASH_JOIN"}},
		{Label: "join_method_apply_join", Assignments: []string{"JOIN_METHOD=APPLY_JOIN"}},
		{Label: "join_method_merge_join", Assignments: []string{"JOIN_METHOD=MERGE_JOIN"}},
		{Label: "join_method_push_broadcast_hash_join", Assignments: []string{"JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN"}},
		{Label: "hash_join_build_left", Assignments: []string{"JOIN_METHOD=HASH_JOIN", "HASH_JOIN_BUILD_SIDE=BUILD_LEFT"}},
		{Label: "hash_join_build_right", Assignments: []string{"JOIN_METHOD=HASH_JOIN", "HASH_JOIN_BUILD_SIDE=BUILD_RIGHT"}},
		{Label: "apply_join_batch_true", Assignments: []string{"JOIN_METHOD=APPLY_JOIN", "BATCH_MODE=TRUE"}},
		{Label: "apply_join_batch_false", Assignments: []string{"JOIN_METHOD=APPLY_JOIN", "BATCH_MODE=FALSE"}},
		{Label: "hash_join_execution_multi_pass", Assignments: []string{"JOIN_METHOD=HASH_JOIN", "HASH_JOIN_EXECUTION=MULTI_PASS"}},
		{Label: "hash_join_execution_one_pass", Assignments: []string{"JOIN_METHOD=HASH_JOIN", "HASH_JOIN_EXECUTION=ONE_PASS"}},
		{Label: "group_method_hash_group", Assignments: []string{"GROUP_METHOD=HASH_GROUP"}},
		{Label: "group_method_stream_group", Assignments: []string{"GROUP_METHOD=STREAM_GROUP"}},
	}
}

var tvfQueries = []queryCase{
	{
		Label: "tvf/change-stream",
		SQL:   `SELECT ChangeRecord FROM READ_EverythingStream (start_timestamp => TIMESTAMP "2026-05-06T00:00:00Z")`,
	},
}

var lockHintQueries = []queryCase{
	{
		Label:    "lock-hints/read-only/shared-point-read",
		SQL:      `@{LOCK_SCANNED_RANGES=shared} SELECT FirstName FROM Singers WHERE SingerId = 1`,
		PlanMode: planModeReadOnly,
	},
	{
		Label:    "lock-hints/read-only/exclusive-point-read",
		SQL:      `@{LOCK_SCANNED_RANGES=exclusive} SELECT FirstName FROM Singers WHERE SingerId = 1`,
		PlanMode: planModeReadOnly,
	},
	{
		Label:    "lock-hints/read-write/shared-point-read",
		SQL:      `@{LOCK_SCANNED_RANGES=shared} SELECT FirstName FROM Singers WHERE SingerId = 1`,
		PlanMode: planModeReadWrite,
	},
	{
		Label:    "lock-hints/read-write/exclusive-point-read",
		SQL:      `@{LOCK_SCANNED_RANGES=exclusive} SELECT FirstName FROM Singers WHERE SingerId = 1`,
		PlanMode: planModeReadWrite,
	},
	{
		Label:    "lock-hints/read-write/exclusive-full-scan",
		SQL:      `@{LOCK_SCANNED_RANGES=exclusive} SELECT FirstName FROM Singers`,
		PlanMode: planModeReadWrite,
	},
}

var dmlQueries = []queryCase{
	dmlCase(
		"dml/insert-values",
		`INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (1, 'Marc', 'Richards')`,
	),
	dmlCase(
		"dml/insert-default",
		`INSERT INTO Singers (SingerId, Status) VALUES (2, DEFAULT)`,
	),
	dmlCase(
		"dml/insert-select",
		`INSERT INTO Singers (SingerId, FirstName, LastName) SELECT SingerId, FirstName, LastName FROM AckworthSingers`,
	),
	dmlCase(
		"dml/insert-select-unnest",
		`INSERT INTO Singers (SingerId, FirstName, LastName) SELECT * FROM UNNEST([(4, 'Lea', 'Martin'), (5, 'David', 'Lomond')])`,
	),
	dmlCase(
		"dml/insert-subquery",
		`INSERT INTO Singers (SingerId, FirstName) VALUES (6, (SELECT FirstName FROM AckworthSingers WHERE SingerId = 6))`,
	),
	dmlCase(
		"dml/insert-assert-rows-modified",
		`INSERT INTO Singers (SingerId, FirstName) VALUES (7, 'Asserted') ASSERT_ROWS_MODIFIED 1`,
	),
	dmlCase(
		"dml/insert-ignore",
		`INSERT IGNORE INTO Singers (SingerId, FirstName) VALUES (8, 'Ignored')`,
	),
	dmlCase(
		"dml/insert-or-ignore",
		`INSERT OR IGNORE INTO Singers (SingerId, FirstName) VALUES (9, 'Ignored')`,
	),
	dmlCase(
		"dml/insert-or-update",
		`INSERT OR UPDATE INTO Singers (SingerId, Status) VALUES (10, 'inactive')`,
	),
	dmlCase(
		"dml/insert-or-update-then-return-with-action",
		`INSERT OR UPDATE Singers (SingerId, FirstName, LastName) VALUES (11, 'Melissa', 'Gartner') THEN RETURN WITH ACTION SingerId, FirstName || ' ' || LastName AS FullName`,
	),
	dmlCase(
		"dml/insert-on-conflict-do-nothing",
		`INSERT INTO Singers (SingerId, FirstName) VALUES (12, 'John') ON CONFLICT DO NOTHING`,
	),
	dmlCase(
		"dml/insert-on-conflict-target-do-nothing",
		`INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (13, 'John', 'Smith') ON CONFLICT(SingerId) DO NOTHING`,
	),
	dmlCase(
		"dml/insert-on-conflict-unique-do-nothing",
		`INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (14, 'Jane', 'Smith') ON CONFLICT ON UNIQUE CONSTRAINT UniqueIndex_SingerName DO NOTHING`,
	),
	dmlCase(
		"dml/insert-on-conflict-do-update",
		`INSERT INTO Singers (SingerId, FirstName, LastName, Status) VALUES (15, 'Adele', 'Adkins', 'active') ON CONFLICT(SingerId) DO UPDATE SET Status = EXCLUDED.Status`,
	),
	dmlCase(
		"dml/insert-on-conflict-do-update-where",
		`INSERT INTO Singers (SingerId, FirstName, LastName, Status) VALUES (16, 'Adele', 'Adkins', 'active') ON CONFLICT ON UNIQUE CONSTRAINT UniqueIndex_SingerName DO UPDATE SET Status = EXCLUDED.Status WHERE Singers.Status IS DISTINCT FROM EXCLUDED.Status`,
	),
	dmlCase(
		"dml/insert-on-conflict-select",
		`INSERT INTO Singers (SingerId, FirstName, LastName) (SELECT SingerId, FirstName, LastName FROM AckworthSingers) ON CONFLICT(SingerId) DO NOTHING`,
	),
	dmlCase(
		"dml/insert-then-return",
		`INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (17, 'Russell', 'Morales') THEN RETURN SingerId, FirstName || ' ' || LastName AS FullName`,
	),
	dmlCase(
		"dml/insert-fans-default-key-then-return",
		`INSERT INTO Fans (FirstName, LastName) VALUES ('Melissa', 'Garcia') THEN RETURN FanId`,
	),
	dmlCase(
		"dml/update-literal",
		`UPDATE Singers SET BirthDate = DATE '1990-10-10', SingerInfo = b'nationality:USA' WHERE FirstName = 'Marc' AND LastName = 'Richards'`,
	),
	dmlCase(
		"dml/update-default",
		`UPDATE Singers SET Status = DEFAULT WHERE SingerId = 1`,
	),
	dmlCase(
		"dml/update-on-update-column",
		`UPDATE Singers SET Status = 'inactive' WHERE SingerId = 10`,
	),
	dmlCase(
		"dml/update-array",
		`UPDATE Concerts SET TicketPrices = [25, 50, 100] WHERE VenueId = 1`,
	),
	dmlCase(
		"dml/update-subquery",
		`UPDATE Singers SET Status = 'active' WHERE SingerId IN (SELECT SingerId FROM Albums)`,
	),
	dmlCase(
		"dml/update-force-index",
		`UPDATE Singers@{FORCE_INDEX=SingersByFirstLastName} SET Status = 'inactive' WHERE FirstName = 'Marc' AND LastName = 'Richards'`,
	),
	dmlCase(
		"dml/update-then-return",
		`UPDATE Singers SET BirthDate = DATE '1990-10-10' WHERE FirstName = 'Russell' THEN RETURN SingerId, EXTRACT(YEAR FROM BirthDate) AS year`,
	),
	dmlCase(
		"dml/delete-where",
		`DELETE FROM Singers WHERE FirstName = 'Alice'`,
	),
	dmlCase(
		"dml/delete-subquery",
		`DELETE FROM Singers WHERE FirstName NOT IN (SELECT FirstName FROM AckworthSingers)`,
	),
	dmlCase(
		"dml/delete-force-index",
		`DELETE FROM Singers@{FORCE_INDEX=SingersByFirstLastName} WHERE FirstName = 'Alice'`,
	),
	dmlCase(
		"dml/delete-then-return",
		`DELETE FROM Singers WHERE FirstName = 'Melissa' THEN RETURN * EXCEPT (LastUpdated)`,
	),
}

func dmlCase(label, sql string) queryCase {
	return queryCase{
		Label:    label,
		SQL:      sql,
		PlanMode: planModeReadWrite,
	}
}

var joinHintMatrixAxis = queryMatrixAxis{
	Name: "Hint",
	Values: []queryMatrixAxisValue{
		{Label: "force_join_order_true", Fields: map[string]string{"SQL": "FORCE_JOIN_ORDER=TRUE"}},
		{Label: "join_method_hash_join", Fields: map[string]string{"SQL": "JOIN_METHOD=HASH_JOIN"}},
		{Label: "join_method_apply_join", Fields: map[string]string{"SQL": "JOIN_METHOD=APPLY_JOIN"}},
		{Label: "join_method_merge_join", Fields: map[string]string{"SQL": "JOIN_METHOD=MERGE_JOIN"}},
		{Label: "join_method_push_broadcast_hash_join", Fields: map[string]string{"SQL": "JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN"}},
		{Label: "hash_join_build_left", Fields: map[string]string{"SQL": "JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT"}},
		{Label: "hash_join_build_right", Fields: map[string]string{"SQL": "JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT"}},
		{Label: "hash_join_execution_multi_pass", Fields: map[string]string{"SQL": "JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=MULTI_PASS"}},
		{Label: "hash_join_execution_one_pass", Fields: map[string]string{"SQL": "JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS"}},
		{Label: "apply_join_batch_true", Fields: map[string]string{"SQL": "JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE"}},
		{Label: "apply_join_batch_false", Fields: map[string]string{"SQL": "JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE"}},
		{Label: "legacy_join_type_apply_join", Fields: map[string]string{"SQL": "JOIN_TYPE=APPLY_JOIN"}},
	},
}

var joinMatrixQueries = buildJoinMatrixQueries()
var subqueryJoinHintMatrixQueries = buildSubqueryJoinHintMatrixQueries()
var hintMatrixQueries = buildHintMatrixQueries()

func buildJoinMatrixQueries() []queryCase {
	joinAxis := queryMatrixAxis{
		Name: "Join",
		Values: []queryMatrixAxisValue{
			{Label: "inner", Fields: map[string]string{"SQL": "INNER JOIN"}},
			{Label: "left", Fields: map[string]string{"SQL": "LEFT JOIN"}},
			{Label: "right", Fields: map[string]string{"SQL": "RIGHT JOIN"}},
		},
	}
	return buildQueryMatrixCases("join-matrix", `
SELECT a.AlbumTitle, s.SongName
FROM Albums AS a {{.Join.SQL}}@{ {{- .Hint.SQL -}} } Songs AS s
ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
`, joinAxis, joinHintMatrixAxis)
}

func buildSubqueryJoinHintMatrixQueries() []queryCase {
	formAxis := queryMatrixAxis{
		Name: "Form",
		Values: []queryMatrixAxisValue{
			{Label: "in", Fields: map[string]string{"Predicate": "s.SingerId IN (SELECT a.SingerId FROM Albums AS a)"}},
			{Label: "exists", Fields: map[string]string{"Predicate": "EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)"}},
			{Label: "not_in", Fields: map[string]string{"Predicate": "s.SingerId NOT IN (SELECT a.SingerId FROM Albums AS a)"}},
			{Label: "not_exists", Fields: map[string]string{"Predicate": "NOT EXISTS (SELECT 1 FROM Albums AS a WHERE a.SingerId = s.SingerId)"}},
		},
	}
	return buildQueryMatrixCases("subquery-join-hint-matrix", `
@{ {{- .Hint.SQL -}} }
SELECT s.SingerId, s.FirstName
FROM Singers AS s
WHERE {{.Form.Predicate}}
`, formAxis, joinHintMatrixAxis)
}

func buildHintMatrixQueries() []queryCase {
	var out []queryCase
	out = append(out, buildStatementHintMatrixQueries()...)
	out = append(out, buildTableHintMatrixQueries()...)
	out = append(out, buildGroupHintMatrixQueries()...)
	out = append(out, buildGraphHintMatrixQueries()...)
	return out
}

func buildStatementHintMatrixQueries() []queryCase {
	hintAxis := queryMatrixAxis{
		Name: "Hint",
		Values: []queryMatrixAxisValue{
			{Label: "use_additional_parallelism_true", Fields: map[string]string{"SQL": "USE_ADDITIONAL_PARALLELISM=TRUE", "Query": "scan"}},
			{Label: "use_additional_parallelism_false", Fields: map[string]string{"SQL": "USE_ADDITIONAL_PARALLELISM=FALSE", "Query": "scan"}},
			{Label: "optimizer_version_default", Fields: map[string]string{"SQL": "OPTIMIZER_VERSION=default_version", "Query": "scan"}},
			{Label: "optimizer_version_latest", Fields: map[string]string{"SQL": "OPTIMIZER_VERSION=latest_version", "Query": "scan"}},
			{Label: "optimizer_statistics_package_latest", Fields: map[string]string{"SQL": "OPTIMIZER_STATISTICS_PACKAGE=latest", "Query": "scan"}},
			{Label: "allow_distributed_merge_true", Fields: map[string]string{"SQL": "ALLOW_DISTRIBUTED_MERGE=TRUE", "Query": "order_by"}},
			{Label: "allow_distributed_merge_false", Fields: map[string]string{"SQL": "ALLOW_DISTRIBUTED_MERGE=FALSE", "Query": "order_by"}},
			{Label: "lock_scanned_ranges_exclusive", Fields: map[string]string{"SQL": "LOCK_SCANNED_RANGES=exclusive", "Query": "scan"}},
			{Label: "lock_scanned_ranges_shared", Fields: map[string]string{"SQL": "LOCK_SCANNED_RANGES=shared", "Query": "scan"}},
			{Label: "scan_method_batch", Fields: map[string]string{"SQL": "SCAN_METHOD=BATCH", "Query": "scan"}},
			{Label: "scan_method_row", Fields: map[string]string{"SQL": "SCAN_METHOD=ROW", "Query": "scan"}},
			{Label: "scan_method_columnar", Fields: map[string]string{"SQL": "SCAN_METHOD=COLUMNAR", "Query": "scan"}},
			{Label: "scan_method_no_columnar", Fields: map[string]string{"SQL": "SCAN_METHOD=NO_COLUMNAR", "Query": "scan"}},
			{Label: "execution_method_batch", Fields: map[string]string{"SQL": "EXECUTION_METHOD=BATCH", "Query": "scan"}},
			{Label: "execution_method_row", Fields: map[string]string{"SQL": "EXECUTION_METHOD=ROW", "Query": "scan"}},
			{Label: "use_unenforced_foreign_key_true", Fields: map[string]string{"SQL": "USE_UNENFORCED_FOREIGN_KEY=TRUE", "Query": "join"}},
			{Label: "use_unenforced_foreign_key_false", Fields: map[string]string{"SQL": "USE_UNENFORCED_FOREIGN_KEY=FALSE", "Query": "join"}},
			{Label: "allow_timestamp_predicate_pushdown_true", Fields: map[string]string{"SQL": "ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=TRUE", "Query": "timestamp"}},
			{Label: "allow_timestamp_predicate_pushdown_false", Fields: map[string]string{"SQL": "ALLOW_TIMESTAMP_PREDICATE_PUSHDOWN=FALSE", "Query": "timestamp"}},
		},
	}
	return buildQueryMatrixCases("hint-matrix/statement", `
@{ {{- .Hint.SQL -}} }
{{- if eq .Hint.Query "scan" }}
SELECT s.SingerId, s.FirstName FROM Singers AS s WHERE s.SingerId >= 0
{{- else if eq .Hint.Query "order_by" }}
SELECT a.AlbumTitle, a.ReleaseDate FROM Albums AS a ORDER BY a.ReleaseDate, a.AlbumTitle DESC
{{- else if eq .Hint.Query "join" }}
SELECT a.AlbumTitle, s.SongName FROM Albums AS a JOIN Songs AS s ON a.SingerId = s.SingerId AND a.AlbumId = s.AlbumId
{{- else if eq .Hint.Query "timestamp" }}
SELECT s.SingerInfo FROM Singers AS s WHERE s.ModificationTime > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 12 HOUR)
{{- end }}
`, hintAxis)
}

func buildTableHintMatrixQueries() []queryCase {
	hintAxis := queryMatrixAxis{
		Name: "Hint",
		Values: []queryMatrixAxisValue{
			{Label: "force_index_base_table", Fields: map[string]string{"Table": "Singers", "Hint": "FORCE_INDEX=_BASE_TABLE", "Query": "last_name"}},
			{Label: "force_index_secondary", Fields: map[string]string{"Table": "Singers", "Hint": "FORCE_INDEX=SingersByLastName", "Query": "last_name"}},
			{Label: "groupby_scan_optimization_true", Fields: map[string]string{"Table": "Songs", "Hint": "GROUPBY_SCAN_OPTIMIZATION=TRUE", "Query": "group_by"}},
			{Label: "groupby_scan_optimization_false", Fields: map[string]string{"Table": "Songs", "Hint": "GROUPBY_SCAN_OPTIMIZATION=FALSE", "Query": "group_by"}},
			{Label: "scan_method_batch", Fields: map[string]string{"Table": "Singers", "Hint": "SCAN_METHOD=BATCH", "Query": "last_name"}},
			{Label: "scan_method_row", Fields: map[string]string{"Table": "Singers", "Hint": "SCAN_METHOD=ROW", "Query": "last_name"}},
			{Label: "scan_method_columnar", Fields: map[string]string{"Table": "Singers", "Hint": "SCAN_METHOD=COLUMNAR", "Query": "last_name"}},
			{Label: "scan_method_no_columnar", Fields: map[string]string{"Table": "Singers", "Hint": "SCAN_METHOD=NO_COLUMNAR", "Query": "last_name"}},
			{Label: "index_strategy_force_index_union", Fields: map[string]string{"Table": "Singers", "Hint": "INDEX_STRATEGY=FORCE_INDEX_UNION", "Query": "index_union"}},
			{Label: "seekable_key_size_zero", Fields: map[string]string{"Table": "Albums", "Hint": "SEEKABLE_KEY_SIZE=0, FORCE_INDEX=_BASE_TABLE", "Query": "album_pk"}},
			{Label: "seekable_key_size_one", Fields: map[string]string{"Table": "Albums", "Hint": "SEEKABLE_KEY_SIZE=1, FORCE_INDEX=_BASE_TABLE", "Query": "album_pk"}},
			{Label: "seekable_key_size_two", Fields: map[string]string{"Table": "Albums", "Hint": "SEEKABLE_KEY_SIZE=2, FORCE_INDEX=_BASE_TABLE", "Query": "album_pk"}},
		},
	}
	return buildQueryMatrixCases("hint-matrix/table", `
{{- if eq .Hint.Query "last_name" }}
SELECT s.SingerId, s.FirstName
FROM {{.Hint.Table}}@{ {{- .Hint.Hint -}} } AS s
WHERE s.LastName = 'Smith'
{{- else if eq .Hint.Query "group_by" }}
SELECT s.SingerId
FROM {{.Hint.Table}}@{ {{- .Hint.Hint -}} } AS s
GROUP BY s.SingerId
{{- else if eq .Hint.Query "index_union" }}
SELECT s.SingerId
FROM {{.Hint.Table}}@{ {{- .Hint.Hint -}} } AS s
WHERE s.FirstName = 'Alice' OR s.LastName = 'Smith'
{{- else if eq .Hint.Query "album_pk" }}
SELECT *
FROM {{.Hint.Table}}@{ {{- .Hint.Hint -}} } AS s
WHERE s.SingerId = 1 AND s.AlbumId = 1
{{- end }}
`, hintAxis)
}

func buildGroupHintMatrixQueries() []queryCase {
	hintAxis := queryMatrixAxis{
		Name: "Hint",
		Values: []queryMatrixAxisValue{
			{Label: "hash_group", Fields: map[string]string{"SQL": "GROUP_METHOD=HASH_GROUP"}},
			{Label: "stream_group", Fields: map[string]string{"SQL": "GROUP_METHOD=STREAM_GROUP"}},
		},
	}
	return buildQueryMatrixCases("hint-matrix/group", `
SELECT s.SingerId, COUNT(*) AS SongCount
FROM Songs AS s
GROUP@{ {{- .Hint.SQL -}} } BY s.SingerId
`, hintAxis)
}

func buildGraphHintMatrixQueries() []queryCase {
	hintAxis := queryMatrixAxis{
		Name: "Hint",
		Values: []queryMatrixAxisValue{
			{Label: "match_join_method_apply_join", Fields: map[string]string{"Query": "match", "SQL": "JOIN_METHOD=APPLY_JOIN"}},
			{Label: "match_join_method_hash_join", Fields: map[string]string{"Query": "match", "SQL": "JOIN_METHOD=HASH_JOIN"}},
			{Label: "traversal_join_method_apply_join", Fields: map[string]string{"Query": "traversal", "SQL": "JOIN_METHOD=APPLY_JOIN"}},
			{Label: "traversal_join_method_hash_join", Fields: map[string]string{"Query": "traversal", "SQL": "JOIN_METHOD=HASH_JOIN"}},
			{Label: "element_force_index_base_table", Fields: map[string]string{"Query": "element", "SQL": "FORCE_INDEX=_BASE_TABLE"}},
			{Label: "group_hash_group", Fields: map[string]string{"Query": "group", "SQL": "GROUP_METHOD=HASH_GROUP"}},
			{Label: "group_stream_group", Fields: map[string]string{"Query": "group", "SQL": "GROUP_METHOD=STREAM_GROUP"}},
		},
	}
	return buildQueryMatrixCases("hint-matrix/graph", `
{{- if eq .Hint.Query "match" }}
GRAPH MusicGraph
MATCH @{ {{- .Hint.SQL -}} } (s:Singers)-[c:CollabWith]->(featured:Singers)
RETURN featured.SingerId AS featured
{{- else if eq .Hint.Query "traversal" }}
GRAPH MusicGraph
MATCH (s:Singers)-[c:CollabWith]->@{ {{- .Hint.SQL -}} }(featured:Singers)
RETURN featured.SingerId AS featured
{{- else if eq .Hint.Query "element" }}
GRAPH MusicGraph
MATCH (s:Singers)-[@{ {{- .Hint.SQL -}} } :CollabWith]->(featured:Singers)
RETURN featured.SingerId AS featured
{{- else if eq .Hint.Query "group" }}
GRAPH MusicGraph
MATCH (s:Singers)-[c:CollabWith]->(featured:Singers)
RETURN s.SingerId AS singer, COUNT(*) AS featured_count
GROUP@{ {{- .Hint.SQL -}} } BY s.SingerId
{{- end }}
`, hintAxis)
}
