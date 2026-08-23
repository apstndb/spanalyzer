package main

var optimizerV9Queries = []queryCase{
	{
		Label: "optimizer-v9/dca-input-column-pruning/v8-control",
		SQL: `@{OPTIMIZER_VERSION=8}
SELECT s.FirstName, c.ConcertDate
FROM Concerts AS c JOIN Singers AS s ON s.SingerId = c.SingerId`,
	},
	{
		Label: "optimizer-v9/dca-input-column-pruning/v9",
		SQL: `@{OPTIMIZER_VERSION=9}
SELECT s.FirstName, c.ConcertDate
FROM Concerts AS c JOIN Singers AS s ON s.SingerId = c.SingerId`,
	},
	{
		Label: "optimizer-v9/dca-input-column-pruning-limit/v9",
		SQL: `@{OPTIMIZER_VERSION=9}
SELECT s.FirstName, c.ConcertDate
FROM Concerts AS c JOIN Singers AS s ON s.SingerId = c.SingerId
LIMIT 5`,
	},
	{
		Label: "optimizer-v9/index-union-aggressiveness/v9",
		SQL: `@{OPTIMIZER_VERSION=9}
SELECT s.SingerId
FROM Singers AS s
WHERE s.FirstName = 'Alice' OR s.LastName = 'Smith'`,
	},
	{
		Label:    "optimizer-v9/dml-delete-then-return/v8-control",
		SQL:      `@{OPTIMIZER_VERSION=8} DELETE FROM Singers WHERE FirstName = 'Melissa' THEN RETURN * EXCEPT (LastUpdated)`,
		PlanMode: planModeReadWrite,
	},
	{
		Label:    "optimizer-v9/dml-delete-then-return/v9",
		SQL:      `@{OPTIMIZER_VERSION=9} DELETE FROM Singers WHERE FirstName = 'Melissa' THEN RETURN * EXCEPT (LastUpdated)`,
		PlanMode: planModeReadWrite,
	},
	{
		Label: "optimizer-v9/unsupported/v10",
		SQL:   `@{OPTIMIZER_VERSION=10} SELECT 1 AS value`,
	},
}
