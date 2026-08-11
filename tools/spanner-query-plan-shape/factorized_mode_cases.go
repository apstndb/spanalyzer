package main

// factorizedModeQueries keeps ordinary-join FACTORIZED_MODE eligibility and
// plan-visible effects separate from the set-operation controls. The selected
// payload columns matter: a factorized side must retain non-join output that
// can be reconstructed after deduplicating the join key.
var factorizedModeQueries = []queryCase{
	{Label: "factorized/control/default", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/effect/left", SQL: `SELECT a.AlbumId, s.SingerId FROM Albums AS a JOIN@{FACTORIZED_MODE=FACTORIZE_LEFT} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/effect/right", SQL: `SELECT a.SingerId, s.TrackId FROM Albums AS a JOIN@{FACTORIZED_MODE=FACTORIZE_RIGHT} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/effect/both", SQL: `SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{FACTORIZED_MODE=FACTORIZE_BOTH} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/version/v1-hash-both-effect", SQL: `@{OPTIMIZER_VERSION=1} SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN, FACTORIZED_MODE=FACTORIZE_BOTH} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/version/v8-hash-both-effect", SQL: `@{OPTIMIZER_VERSION=8} SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{JOIN_METHOD=HASH_JOIN, FACTORIZED_MODE=FACTORIZE_BOTH} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/version/v4-left-accepted-no-visible-effect", SQL: `@{OPTIMIZER_VERSION=4} SELECT a.AlbumId, s.SingerId FROM Albums AS a JOIN@{FACTORIZED_MODE=FACTORIZE_LEFT} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/version/v5-left-effect", SQL: `@{OPTIMIZER_VERSION=5} SELECT a.AlbumId, s.SingerId FROM Albums AS a JOIN@{FACTORIZED_MODE=FACTORIZE_LEFT} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/version/v4-join-key-only-both-accepted-no-visible-effect", SQL: `@{OPTIMIZER_VERSION=4} SELECT a.SingerId FROM Albums AS a JOIN@{FACTORIZED_MODE=FACTORIZE_BOTH} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/version/v4-non-equality-both-accepted-no-visible-effect", SQL: `@{OPTIMIZER_VERSION=4} SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{FACTORIZED_MODE=FACTORIZE_BOTH} Songs AS s ON a.SingerId < s.SingerId`},
	{Label: "factorized/version/v8-unsupported/join-key-only-left", SQL: `@{OPTIMIZER_VERSION=8} SELECT a.SingerId FROM Albums AS a JOIN@{FACTORIZED_MODE=FACTORIZE_LEFT} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/version/v8-unsupported/join-key-only-right", SQL: `@{OPTIMIZER_VERSION=8} SELECT a.SingerId FROM Albums AS a JOIN@{FACTORIZED_MODE=FACTORIZE_RIGHT} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/version/v8-unsupported/join-key-only-both", SQL: `@{OPTIMIZER_VERSION=8} SELECT a.SingerId FROM Albums AS a JOIN@{FACTORIZED_MODE=FACTORIZE_BOTH} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/version/v8-unsupported/left-outer-both", SQL: `@{OPTIMIZER_VERSION=8} SELECT a.AlbumId, s.TrackId FROM Albums AS a LEFT JOIN@{FACTORIZED_MODE=FACTORIZE_BOTH} Songs AS s ON a.SingerId = s.SingerId`},
	{Label: "factorized/version/v8-unsupported/non-equality-both", SQL: `@{OPTIMIZER_VERSION=8} SELECT a.AlbumId, s.TrackId FROM Albums AS a JOIN@{FACTORIZED_MODE=FACTORIZE_BOTH} Songs AS s ON a.SingerId < s.SingerId`},
}
