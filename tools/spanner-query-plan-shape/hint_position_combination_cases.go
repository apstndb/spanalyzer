package main

// hintPositionCombinationQueries separates plan-visible hint effects from
// combinations that are merely syntax-accepted. The planvocab expectation
// file covers the WHERE EXISTS/IN and statement-plus-predicate cases whose
// selected join family and orientation are visible in Omni PLAN output. The
// remaining cases stay in the raw-plan stream as controls without claiming
// that every accepted key changed the plan.
var hintPositionCombinationQueries = []queryCase{
	{
		Label: "hint-combination/sql-exists/hash-build-left-one-pass",
		SQL:   `SELECT SingerId FROM Singers WHERE EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT 1 FROM Albums WHERE Albums.SingerId = Singers.SingerId)`,
	},
	{
		Label: "hint-combination/sql-exists/hash-build-right-one-pass",
		SQL:   `SELECT SingerId FROM Singers WHERE EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT 1 FROM Albums WHERE Albums.SingerId = Singers.SingerId)`,
	},
	{
		Label: "hint-combination/sql-exists/apply-batch-true",
		SQL:   `SELECT SingerId FROM Singers WHERE EXISTS @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} (SELECT 1 FROM Albums WHERE Albums.SingerId = Singers.SingerId)`,
	},
	{
		Label: "hint-combination/sql-exists/apply-batch-false",
		SQL:   `SELECT SingerId FROM Singers WHERE EXISTS @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} (SELECT 1 FROM Albums WHERE Albums.SingerId = Singers.SingerId)`,
	},
	{
		Label: "hint-combination/in-subquery/hash-build-left-one-pass",
		SQL:   `SELECT SingerId FROM Singers WHERE SingerId IN @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT SingerId FROM Albums)`,
	},
	{
		Label: "hint-combination/in-subquery/hash-build-right-one-pass",
		SQL:   `SELECT SingerId FROM Singers WHERE SingerId IN @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT SingerId FROM Albums)`,
	},
	{
		Label: "hint-combination/in-subquery/apply-batch-true",
		SQL:   `SELECT SingerId FROM Singers WHERE SingerId IN @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} (SELECT SingerId FROM Albums)`,
	},
	{
		Label: "hint-combination/in-subquery/apply-batch-false",
		SQL:   `SELECT SingerId FROM Singers WHERE SingerId IN @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} (SELECT SingerId FROM Albums)`,
	},
	{
		Label: "hint-combination/statement-plus-sql-exists/hash-build-left-one-pass",
		SQL:   `@{OPTIMIZER_VERSION=8, FORCE_JOIN_ORDER=TRUE} SELECT SingerId FROM Singers WHERE EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT 1 FROM Albums WHERE Albums.SingerId = Singers.SingerId)`,
	},
	{
		Label: "hint-combination/set-operation/hash-one-pass-control",
		SQL:   `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS} ALL SELECT SingerId FROM Singers`,
	},
	{
		Label: "hint-combination/scalar-sql-exists/hash-build-left-one-pass-control",
		SQL:   `SELECT EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT 1 FROM Albums WHERE Albums.SingerId = Singers.SingerId) FROM Singers`,
	},
	{
		Label: "hint-combination/scalar-in-subquery/hash-build-left-one-pass-control",
		SQL:   `SELECT SingerId IN @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} (SELECT SingerId FROM Albums) FROM Singers`,
	},
	{
		Label: "hint-combination/gql-exists/hash-build-left-one-pass-control",
		SQL:   `GRAPH MusicGraph MATCH (p:Singers) RETURN EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT, HASH_JOIN_EXECUTION=ONE_PASS} { MATCH (q:Singers) RETURN q.SingerId } AS v`,
	},
}
