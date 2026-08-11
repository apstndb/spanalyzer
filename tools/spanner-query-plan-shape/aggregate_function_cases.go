package main

// aggregateFunctionQueries compares the documented Spanner GoogleSQL
// aggregate functions with the physical expressions attached to Aggregate
// operators through Agg child links. It also retains modifier, grouping-only,
// and unsupported-function controls that expose optimizer lowering boundaries.
var aggregateFunctionQueries = []queryCase{
	{Label: "aggregate-function/general/any-value", SQL: `SELECT ANY_VALUE(SongName) FROM Songs`},
	{Label: "aggregate-function/general/array-agg", SQL: `SELECT ARRAY_AGG(SongName) FROM Songs`},
	{Label: "aggregate-function/general/array-concat-agg", SQL: `SELECT ARRAY_CONCAT_AGG([SongName]) FROM Songs`},
	{Label: "aggregate-function/general/avg", SQL: `SELECT AVG(Duration) FROM Songs`},
	{Label: "aggregate-function/general/bit-and", SQL: `SELECT BIT_AND(Duration) FROM Songs`},
	{Label: "aggregate-function/general/bit-or", SQL: `SELECT BIT_OR(Duration) FROM Songs`},
	{Label: "aggregate-function/general/bit-xor", SQL: `SELECT BIT_XOR(Duration) FROM Songs`},
	{Label: "aggregate-function/general/count", SQL: `SELECT COUNT(SongName) FROM Songs`},
	{Label: "aggregate-function/general/countif", SQL: `SELECT COUNTIF(Duration > 0) FROM Songs`},
	{Label: "aggregate-function/general/logical-and", SQL: `SELECT LOGICAL_AND(Duration > 0) FROM Songs`},
	{Label: "aggregate-function/general/logical-or", SQL: `SELECT LOGICAL_OR(Duration > 0) FROM Songs`},
	{Label: "aggregate-function/general/max", SQL: `SELECT MAX(Duration) FROM Songs`},
	{Label: "aggregate-function/general/min", SQL: `SELECT MIN(Duration) FROM Songs`},
	{Label: "aggregate-function/statistical/stddev", SQL: `SELECT STDDEV(Duration) FROM Songs`},
	{Label: "aggregate-function/statistical/stddev-samp", SQL: `SELECT STDDEV_SAMP(Duration) FROM Songs`},
	{Label: "aggregate-function/general/string-agg", SQL: `SELECT STRING_AGG(SongName) FROM Songs`},
	{Label: "aggregate-function/general/sum", SQL: `SELECT SUM(Duration) FROM Songs`},
	{Label: "aggregate-function/statistical/var-samp", SQL: `SELECT VAR_SAMP(Duration) FROM Songs`},
	{Label: "aggregate-function/statistical/variance", SQL: `SELECT VARIANCE(Duration) FROM Songs`},

	{Label: "aggregate-function/modifier/any-value-having-max", SQL: `SELECT ANY_VALUE(SongName HAVING MAX Duration) FROM Songs`},
	{Label: "aggregate-function/modifier/any-value-having-min", SQL: `SELECT ANY_VALUE(SongName HAVING MIN Duration) FROM Songs`},
	{Label: "aggregate-function/modifier/count-star", SQL: `SELECT COUNT(*) FROM Songs`},
	{Label: "aggregate-function/modifier/count-distinct", SQL: `SELECT COUNT(DISTINCT SongName) FROM Songs`},
	{Label: "aggregate-function/modifier/array-agg-distinct-ordered-limited", SQL: `SELECT ARRAY_AGG(DISTINCT SongName IGNORE NULLS ORDER BY SongName LIMIT 5) FROM Songs`},
	{Label: "aggregate-function/modifier/avg-distinct", SQL: `SELECT AVG(DISTINCT Duration) FROM Songs`},
	{Label: "aggregate-function/modifier/string-agg-distinct-ordered-limited", SQL: `SELECT STRING_AGG(DISTINCT SongName, ',' ORDER BY SongName LIMIT 5) FROM Songs`},
	{Label: "aggregate-function/control/grouped-count-sum", SQL: `SELECT SingerId, COUNT(*), SUM(Duration) FROM Songs GROUP BY SingerId`},
	{Label: "aggregate-function/control/group-by-without-agg", SQL: `SELECT SingerId FROM Songs GROUP BY SingerId`},
	{Label: "aggregate-function/control/select-distinct-without-agg", SQL: `SELECT DISTINCT SongName FROM Songs`},
	{Label: "aggregate-function/unsupported/approx-count-distinct", SQL: `SELECT APPROX_COUNT_DISTINCT(SongName) FROM Songs`},
	{Label: "aggregate-function/unsupported/corr", SQL: `SELECT CORR(Duration, TrackId) FROM Songs`},
}
