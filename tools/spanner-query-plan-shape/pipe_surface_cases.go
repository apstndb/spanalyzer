package main

// pipeSurfaceQueries covers every pipe operator listed by the Spanner pipe
// syntax reference. These queries use non-constant inputs where useful so the
// resulting plans retain the operation instead of folding it away.
var pipeSurfaceQueries = []queryCase{
	{Label: "pipe-surface/accepted/select", SQL: `FROM Singers |> SELECT SingerId, FirstName`},
	{Label: "pipe-surface/accepted/extend", SQL: `FROM Singers |> EXTEND LENGTH(FirstName) AS name_length`},
	{Label: "pipe-surface/accepted/set", SQL: `FROM Singers |> SET FirstName = UPPER(FirstName)`},
	{Label: "pipe-surface/accepted/drop", SQL: `FROM Singers |> DROP LastName`},
	{Label: "pipe-surface/accepted/rename", SQL: `FROM Singers |> RENAME FirstName AS GivenName`},
	{Label: "pipe-surface/accepted/as", SQL: `FROM Singers |> AS singer |> SELECT singer.SingerId`},
	{Label: "pipe-surface/accepted/where", SQL: `FROM Singers |> WHERE SingerId > 0`},
	{Label: "pipe-surface/accepted/aggregate", SQL: `FROM Albums |> AGGREGATE COUNT(*) AS album_count GROUP BY SingerId`},
	{Label: "pipe-surface/accepted/join", SQL: `FROM Singers |> AS singer |> JOIN Concerts AS concert ON singer.SingerId = concert.SingerId |> SELECT singer.SingerId, concert.ConcertDate`},
	{Label: "pipe-surface/accepted/order-by", SQL: `FROM Singers |> ORDER BY FirstName`},
	{Label: "pipe-surface/accepted/limit", SQL: `FROM Singers |> LIMIT 2`},
	{Label: "pipe-surface/accepted/union", SQL: `SELECT SingerId FROM Singers |> UNION ALL (SELECT SingerId FROM Albums)`},
	{Label: "pipe-surface/accepted/intersect", SQL: `SELECT SingerId FROM Singers |> INTERSECT DISTINCT (SELECT SingerId FROM Albums)`},
	{Label: "pipe-surface/accepted/except", SQL: `SELECT SingerId FROM Singers |> EXCEPT DISTINCT (SELECT SingerId FROM Albums)`},
	{Label: "pipe-surface/accepted/tablesample", SQL: `FROM Singers |> TABLESAMPLE BERNOULLI (1 PERCENT)`},
}
