package main

// planVocabInferenceQueries combine catalog components that are individually
// known but whose correlation or multiplicity was not previously proved by a
// checked-in case. Controls stay adjacent to the inferred shapes so a planner
// change can distinguish absence from a different operator choice.
var planVocabInferenceQueries = []queryCase{
	{
		Label: "planvocab-inference/dca-order/default",
		SQL:   `@{OPTIMIZER_VERSION=5} SELECT * FROM Songs@{FORCE_INDEX=SongsBySongName} ORDER BY SongName DESC LIMIT 1`,
	},
	{
		Label: "planvocab-inference/dca-order/no-distributed-merge",
		SQL:   `@{OPTIMIZER_VERSION=5, ALLOW_DISTRIBUTED_MERGE=FALSE} SELECT * FROM Songs@{FORCE_INDEX=SongsBySongName} ORDER BY SongName DESC LIMIT 1`,
	},
	{
		Label: "planvocab-inference/merge-right-outer",
		SQL: `SELECT s.FirstName, c.VenueId
FROM Singers AS s RIGHT JOIN@{JOIN_METHOD=MERGE_JOIN} Concerts AS c
ON s.SingerId = c.SingerId`,
	},
	{
		Label: "planvocab-inference/merge-left-outer-control",
		SQL: `SELECT s.FirstName, c.VenueId
FROM Concerts AS c LEFT JOIN@{JOIN_METHOD=MERGE_JOIN} Singers AS s
ON s.SingerId = c.SingerId`,
	},
	{
		Label: "planvocab-inference/merge-one-to-one",
		SQL: `SELECT s.FirstName, f.FirstName
FROM Singers AS s JOIN@{JOIN_METHOD=MERGE_JOIN} FkSingers AS f
ON s.SingerId = f.SingerId`,
	},
	{
		Label: "planvocab-inference/merge-one-to-many-control",
		SQL: `SELECT s.FirstName, a.AlbumTitle
FROM FkSingers AS s JOIN@{JOIN_METHOD=MERGE_JOIN} FkAlbums AS a
ON s.SingerId = a.SingerId`,
	},
	{
		Label: "planvocab-inference/offset-v1",
		SQL:   `@{OPTIMIZER_VERSION=1} SELECT SongGenre FROM Songs ORDER BY SongGenre LIMIT 3 OFFSET 2`,
	},
	{
		Label: "planvocab-inference/offset-v3",
		SQL:   `@{OPTIMIZER_VERSION=3} SELECT SongGenre FROM Songs ORDER BY SongGenre LIMIT 3 OFFSET 2`,
	},
	{
		Label: "planvocab-inference/offset-v8",
		SQL:   `@{OPTIMIZER_VERSION=8} SELECT SongGenre FROM Songs ORDER BY SongGenre LIMIT 3 OFFSET 2`,
	},
	{
		Label: "planvocab-inference/offset-control",
		SQL:   `@{OPTIMIZER_VERSION=8} SELECT SongGenre FROM Songs ORDER BY SongGenre LIMIT 3`,
	},
	{
		Label: "planvocab-inference/hash-left-outer-residual-build-left",
		SQL: `SELECT s.FirstName, c.VenueId
FROM Singers AS s LEFT JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} Concerts AS c
ON s.SingerId = c.SingerId AND c.VenueId > s.SingerId`,
	},
	{
		Label: "planvocab-inference/hash-left-outer-residual-build-right",
		SQL: `SELECT s.FirstName, c.VenueId
FROM Singers AS s LEFT JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT} Concerts AS c
ON s.SingerId = c.SingerId AND c.VenueId > s.SingerId`,
	},
	{
		Label: "planvocab-inference/hash-left-outer-control",
		SQL: `SELECT s.FirstName, c.VenueId
FROM Singers AS s LEFT JOIN@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT} Concerts AS c
ON s.SingerId = c.SingerId`,
	},
	{
		Label: "planvocab-inference/minor-sort-values",
		SQL:   `SELECT SingerId, AlbumTitle, MarketingBudget, ReleaseDate FROM Albums ORDER BY SingerId, AlbumTitle`,
	},
	{
		Label: "planvocab-inference/minor-sort-limit-values",
		SQL:   `SELECT SingerId, AlbumTitle, MarketingBudget, ReleaseDate FROM Albums WHERE SingerId > 0 ORDER BY SingerId, AlbumTitle LIMIT 3`,
	},
	{
		Label: "planvocab-inference/aggregate-hash-repeated-key-agg",
		SQL: `SELECT SingerId, AlbumId, COUNT(*) AS TrackCount, AVG(Duration) AS AverageDuration
FROM Songs GROUP@{GROUP_METHOD=HASH_GROUP} BY SingerId, AlbumId`,
	},
	{
		Label: "planvocab-inference/aggregate-stream-repeated-key-agg",
		SQL: `SELECT SingerId, AlbumId, COUNT(*) AS TrackCount, AVG(Duration) AS AverageDuration
FROM Songs GROUP@{GROUP_METHOD=STREAM_GROUP} BY SingerId, AlbumId`,
	},
}
