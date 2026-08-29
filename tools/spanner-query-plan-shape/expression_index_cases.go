package main

// Keep expression-index DDL raw until the pinned memefish AST can represent an
// index key as either a column name or a scalar expression.
var expressionIndexDDLs = []string{
	`CREATE TABLE ExpressionIndexVenues (
  Id INT64 NOT NULL,
  VenueName STRING(MAX),
  VenueData JSON
) PRIMARY KEY (Id)`,
	`CREATE INDEX ExpressionIndexVenuesByCity
ON ExpressionIndexVenues((JSON_VALUE(VenueData.address.city)))`,
	`CREATE INDEX ExpressionIndexVenuesByNameState
ON ExpressionIndexVenues(VenueName, (JSON_VALUE(VenueData.address.state)))`,
}

var expressionIndexQueries = []queryCase{
	{
		Label: "expression-index/auto-city",
		SQL: `SELECT Id
FROM ExpressionIndexVenues
WHERE JSON_VALUE(VenueData.address.city) = "Seattle"`,
	},
	{
		Label: "expression-index/force-city",
		SQL: `SELECT Id
FROM ExpressionIndexVenues@{FORCE_INDEX=ExpressionIndexVenuesByCity}
WHERE JSON_VALUE(VenueData.address.city) = "Seattle"`,
	},
	{
		Label: "expression-index/composite-name-state",
		SQL: `SELECT Id
FROM ExpressionIndexVenues@{FORCE_INDEX=ExpressionIndexVenuesByNameState}
WHERE VenueName = "Venue"
  AND JSON_VALUE(VenueData.address.state) = "WA"`,
	},
	{
		Label: "expression-index/base-table-control",
		SQL: `SELECT Id
FROM ExpressionIndexVenues@{FORCE_INDEX=_BASE_TABLE}
WHERE JSON_VALUE(VenueData.address.city) = "Seattle"`,
	},
}
