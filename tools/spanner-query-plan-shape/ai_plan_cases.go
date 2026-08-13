package main

var aiPlanQueries = []queryCase{
	{
		Label: "ai-plan/classify-projection",
		SQL:   `SELECT SongName, AI.CLASSIFY(CONCAT('Classify song-title length: ', SongName), ['short', 'long', 'other']) AS category FROM Songs@{FORCE_INDEX=_BASE_TABLE} WHERE SingerId = 1 LIMIT 10`,
	},
	{
		Label: "ai-plan/classify-case-control",
		SQL:   `SELECT SongName, CASE WHEN LENGTH(SongName) < 10 THEN 'short' ELSE 'long' END AS category FROM Songs@{FORCE_INDEX=_BASE_TABLE} WHERE SingerId = 1 LIMIT 10`,
	},
	{
		Label: "ai-plan/if-filter",
		SQL:   `SELECT SingerId, AlbumId, TrackId, SongName FROM Songs@{FORCE_INDEX=_BASE_TABLE} WHERE SingerId = 1 AND AI.IF(CONCAT('The following is a love song: ', SongName))`,
	},
	{
		Label: "ai-plan/if-scalar-filter-control",
		SQL:   `SELECT SingerId, AlbumId, TrackId, SongName FROM Songs@{FORCE_INDEX=_BASE_TABLE} WHERE SingerId = 1 AND LENGTH(SongName) > 0`,
	},
	{
		Label: "ai-plan/score-order-limit",
		SQL:   `SELECT SongName, AI.SCORE(CONCAT('On a scale from 1 to 10, rate this song title: ', SongName)) AS score FROM Songs@{FORCE_INDEX=_BASE_TABLE} WHERE SingerId = 1 ORDER BY score DESC LIMIT 10`,
	},
	{
		Label: "ai-plan/score-scalar-order-limit-control",
		SQL:   `SELECT SongName, CAST(LENGTH(SongName) AS FLOAT64) AS score FROM Songs@{FORCE_INDEX=_BASE_TABLE} WHERE SingerId = 1 ORDER BY score DESC LIMIT 10`,
	},
}
