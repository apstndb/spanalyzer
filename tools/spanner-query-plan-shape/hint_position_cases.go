package main

type hintPositionExpectation string

const (
	hintPositionAccepted hintPositionExpectation = "accepted"
	hintPositionRejected hintPositionExpectation = "rejected"
)

type hintPositionAuditCase struct {
	Query       queryCase
	Expectation hintPositionExpectation
}

// hintPositionAuditCases covers the hint positions that were not already
// exercised by the statement, table, join, group, graph, function, and DML
// matrices. Synthetic unknown hints intentionally turn successful parsing into
// a later Unsupported hint error when no official key is valid at a position.
var hintPositionAuditCases = []hintPositionAuditCase{
	{queryCase{Label: "hint-position/accepted/select", SQL: `SELECT @{a=1} SingerId FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/order-by", SQL: `SELECT SingerId FROM Singers ORDER @{a=1} BY SingerId`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/set-operation-first", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=HASH_JOIN} ALL SELECT SingerId FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/rejected/set-operation-second-same-level", SQL: `SELECT SingerId FROM Singers UNION ALL SELECT SingerId FROM Singers UNION @{JOIN_METHOD=HASH_JOIN} ALL SELECT SingerId FROM Singers`}, hintPositionRejected},
	{queryCase{Label: "hint-position/accepted/sql-exists", SQL: `SELECT EXISTS @{JOIN_METHOD=HASH_JOIN} (SELECT 1 FROM Albums WHERE Albums.SingerId = Singers.SingerId) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/in-subquery", SQL: `SELECT SingerId IN @{JOIN_METHOD=HASH_JOIN} (SELECT SingerId FROM Albums) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/rejected/in-value-list", SQL: `SELECT SingerId IN @{a=1} (1, 2) FROM Singers`}, hintPositionRejected},
	{queryCase{Label: "hint-position/rejected/in-unnest", SQL: `SELECT SingerId IN @{a=1} UNNEST([1, 2]) FROM Singers`}, hintPositionRejected},
	{queryCase{Label: "hint-position/accepted/like-any-subquery", SQL: `SELECT FirstName LIKE ANY @{JOIN_METHOD=HASH_JOIN} (SELECT AlbumTitle FROM Albums) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/like-some-subquery", SQL: `SELECT FirstName LIKE SOME @{JOIN_METHOD=HASH_JOIN} (SELECT AlbumTitle FROM Albums) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/like-all-subquery", SQL: `SELECT FirstName LIKE ALL @{JOIN_METHOD=HASH_JOIN} (SELECT AlbumTitle FROM Albums) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/like-some-multi-hint-subquery", SQL: `SELECT FirstName LIKE SOME @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT AlbumTitle FROM Albums) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/like-all-multi-hint-subquery", SQL: `SELECT FirstName LIKE ALL @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT AlbumTitle FROM Albums) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/rejected/like-any-value-list", SQL: `SELECT FirstName LIKE ANY @{a=1} ('A', 'B') FROM Singers`}, hintPositionRejected},
	{queryCase{Label: "hint-position/rejected/like-some-value-list", SQL: `SELECT FirstName LIKE SOME @{a=1} ('A', 'B') FROM Singers`}, hintPositionRejected},
	{queryCase{Label: "hint-position/rejected/like-all-value-list", SQL: `SELECT FirstName LIKE ALL @{a=1} ('A', 'B') FROM Singers`}, hintPositionRejected},
	{queryCase{Label: "hint-position/accepted/quantified-subquery", SQL: `SELECT SingerId = ANY @{JOIN_METHOD=HASH_JOIN} (SELECT SingerId FROM Albums) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/quantified-some-subquery", SQL: `SELECT SingerId = SOME @{JOIN_METHOD=HASH_JOIN} (SELECT SingerId FROM Albums) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/quantified-all-subquery", SQL: `SELECT SingerId = ALL @{JOIN_METHOD=HASH_JOIN} (SELECT SingerId FROM Albums) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/quantified-some-multi-hint-subquery", SQL: `SELECT SingerId = SOME @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT SingerId FROM Albums) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/quantified-all-multi-hint-subquery", SQL: `SELECT SingerId = ALL @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT SingerId FROM Albums) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/rejected/quantified-value-list", SQL: `SELECT SingerId = ANY @{a=1} (1, 2) FROM Singers`}, hintPositionRejected},
	{queryCase{Label: "hint-position/rejected/quantified-some-value-list", SQL: `SELECT SingerId = SOME @{a=1} (1, 2) FROM Singers`}, hintPositionRejected},
	{queryCase{Label: "hint-position/rejected/quantified-all-value-list", SQL: `SELECT SingerId = ALL @{a=1} (1, 2) FROM Singers`}, hintPositionRejected},
	{queryCase{Label: "hint-position/accepted/window-partition", SQL: `SELECT ROW_NUMBER() OVER (PARTITION @{a=1} BY SingerId) FROM Singers`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/tvf", SQL: `SELECT * FROM MissingTVF() @{a=1}`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/gql-return", SQL: `GRAPH MusicGraph MATCH (p:Singers) RETURN @{a=1} p.SingerId`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/gql-order-by", SQL: `GRAPH MusicGraph MATCH (p:Singers) RETURN p.SingerId ORDER @{a=1} BY p.SingerId`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/gql-with", SQL: `GRAPH MusicGraph MATCH (p:Singers) WITH @{a=1} p RETURN p.SingerId`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/gql-value", SQL: `GRAPH MusicGraph MATCH (p:Singers) RETURN VALUE @{a=1} { MATCH (q:Singers) RETURN q.SingerId LIMIT 1 } AS v`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/gql-exists", SQL: `GRAPH MusicGraph MATCH (p:Singers) RETURN EXISTS @{JOIN_METHOD=HASH_JOIN} { MATCH (q:Singers) RETURN q.SingerId } AS v`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/rejected/gql-set-operation", SQL: `GRAPH MusicGraph MATCH (p:Singers) RETURN p.SingerId AS id UNION @{a=1} ALL MATCH (q:Singers) RETURN q.SingerId AS id`}, hintPositionRejected},
	{queryCase{Label: "hint-position/rejected/gql-subpath-leading", SQL: `GRAPH MusicGraph MATCH ( @{a=1} (p:Singers) ) RETURN p.SingerId`}, hintPositionRejected},
	{queryCase{Label: "hint-position/rejected/gql-between-edges", SQL: `GRAPH MusicGraph MATCH (a:Singers)-[e:CollabWith]->@{a=1}-[f:CollabWith]->(b:Singers) RETURN a.SingerId`}, hintPositionRejected},
	{queryCase{Label: "hint-position/accepted/pipe-log-unsupported", SQL: `SELECT * FROM Singers |> LOG @{a=1}`}, hintPositionAccepted},
	{queryCase{Label: "hint-position/versioned/pipe-finish", SQL: `SELECT * FROM Singers |> FINISH @{a=1}`}, hintPositionRejected},
	{queryCase{Label: "hint-position/accepted/insert-target", SQL: `INSERT INTO Singers @{a=1} (SingerId, FirstName) SELECT SingerId, FirstName FROM Singers WHERE FALSE`, PlanMode: planModeReadWrite}, hintPositionAccepted},
	{queryCase{Label: "hint-position/accepted/dml-statement-pdml", SQL: `@{PDML_MAX_PARALLELISM=1} UPDATE Singers SET FirstName = FirstName WHERE FALSE`, PlanMode: planModeReadWrite}, hintPositionAccepted},
}

func hintPositionAuditQueries() []queryCase {
	queries := make([]queryCase, 0, len(hintPositionAuditCases))
	for _, auditCase := range hintPositionAuditCases {
		queries = append(queries, auditCase.Query)
	}
	return queries
}
