package main

const gqlSetPropagationStrictError = "GQL set operation requires all input queries to have identical column names"

var gqlSetPropagationQueries = []queryCase{
	{
		Label: "gql-set-propagation/full",
		SQL: `GRAPH MusicGraph
RETURN 1 AS left_only, 10 AS shared
FULL UNION ALL
RETURN 2 AS right_only, 20 AS shared`,
	},
	{
		Label: "gql-set-propagation/full-outer",
		SQL: `GRAPH MusicGraph
RETURN 1 AS left_only, 10 AS shared
FULL OUTER UNION ALL
RETURN 2 AS right_only, 20 AS shared`,
	},
	{
		Label: "gql-set-propagation/outer",
		SQL: `GRAPH MusicGraph
RETURN 1 AS left_only, 10 AS shared
OUTER UNION ALL
RETURN 2 AS right_only, 20 AS shared`,
	},
	{
		Label: "gql-set-propagation/inner",
		SQL: `GRAPH MusicGraph
RETURN 1 AS left_only, 10 AS shared
INNER UNION ALL
RETURN 2 AS right_only, 20 AS shared`,
	},
	{
		Label: "gql-set-propagation/left",
		SQL: `GRAPH MusicGraph
RETURN 1 AS left_only, 10 AS shared
LEFT UNION ALL
RETURN 2 AS right_only, 20 AS shared`,
	},
	{
		Label: "gql-set-propagation/left-outer",
		SQL: `GRAPH MusicGraph
RETURN 1 AS left_only, 10 AS shared
LEFT OUTER UNION ALL
RETURN 2 AS right_only, 20 AS shared`,
	},
	{
		Label: "gql-set-propagation/strict-control",
		SQL: `GRAPH MusicGraph
RETURN 1 AS left_only, 10 AS shared
UNION ALL
RETURN 2 AS right_only, 20 AS shared`,
	},
}
