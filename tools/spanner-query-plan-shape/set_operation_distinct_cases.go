package main

const setOperationDistinctDDL = docsDDL + `
CREATE TABLE SetOpR (
  Id INT64 NOT NULL,
  K INT64 NOT NULL,
) PRIMARY KEY (Id);

CREATE TABLE SetOpS (
  Id INT64 NOT NULL,
  K INT64 NOT NULL,
) PRIMARY KEY (Id);

CREATE TABLE SetOpT (
  Id INT64 NOT NULL,
  K INT64 NOT NULL,
) PRIMARY KEY (Id);
`

// setOperationDistinctQueries keeps set-operator join selection, duplicate
// elimination, and multiplicity restoration in one reproducible matrix.
// Unsupported-hint cases are intentional controls and require
// --continue-on-error. Positive plan-shape claims live in the companion
// planvocab expectation manifest.
var setOperationDistinctQueries = []queryCase{
	{Label: "set-operation/union-all/default", SQL: `SELECT SingerId FROM Singers UNION ALL SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-all/hash-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=HASH_JOIN} ALL SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-all/apply-batch-true-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} ALL SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-all/apply-batch-false-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} ALL SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-all/merge-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=MERGE_JOIN} ALL SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-all/push-broadcast-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} ALL SELECT SingerId FROM Albums`},

	{Label: "set-operation/union-distinct/default", SQL: `SELECT SingerId FROM Singers UNION DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-distinct/hash-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=HASH_JOIN} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-distinct/apply-batch-true-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-distinct/apply-batch-false-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-distinct/merge-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=MERGE_JOIN} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-distinct/push-broadcast-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-distinct/group-hash-unsupported", SQL: `SELECT SingerId FROM Singers UNION @{GROUP_METHOD=HASH_GROUP} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-distinct/group-stream-unsupported", SQL: `SELECT SingerId FROM Singers UNION @{GROUP_METHOD=STREAM_GROUP} DISTINCT SELECT SingerId FROM Albums`},

	{Label: "set-operation/intersect-distinct/default", SQL: `SELECT SingerId FROM Singers INTERSECT DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-distinct/hash", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=HASH_JOIN} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-distinct/apply-batch-true", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-distinct/apply-batch-false", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-distinct/merge", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=MERGE_JOIN} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-distinct/force-join-order", SQL: `SELECT SingerId FROM Singers INTERSECT @{FORCE_JOIN_ORDER=TRUE} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-distinct/three-input-default", SQL: `SELECT SingerId FROM Singers INTERSECT DISTINCT SELECT SingerId FROM Albums INTERSECT DISTINCT SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-distinct/three-input-force-join-order", SQL: `@{FORCE_JOIN_ORDER=TRUE} SELECT SingerId FROM Singers INTERSECT DISTINCT SELECT SingerId FROM Albums INTERSECT DISTINCT SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-distinct/build-left-unsupported", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-distinct/build-right-unsupported", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-distinct/push-broadcast", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-distinct/hash-one-pass", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-distinct/hash-multi-pass", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=MULTI_PASS} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-distinct/factorized-both-control", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=HASH_JOIN, FACTORIZED_MODE=FACTORIZE_BOTH} DISTINCT SELECT SingerId FROM Albums`},
	{
		Label: "set-operation/intersect-distinct/apply-batch-true-execution-row",
		SQL:   `@{EXECUTION_METHOD=ROW} SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} DISTINCT SELECT SingerId FROM Albums`,
	},
	{
		Label: "set-operation/intersect-distinct/apply-batch-true-execution-batch",
		SQL:   `@{EXECUTION_METHOD=BATCH} SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} DISTINCT SELECT SingerId FROM Albums`,
	},
	{Label: "set-operation/intersect-distinct/scan-row", SQL: `@{SCAN_METHOD=ROW} SELECT SingerId FROM Singers INTERSECT DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-distinct/scan-batch", SQL: `@{SCAN_METHOD=BATCH} SELECT SingerId FROM Singers INTERSECT DISTINCT SELECT SingerId FROM Albums`},

	{Label: "set-operation/except-distinct/default", SQL: `SELECT SingerId FROM Singers EXCEPT DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/hash", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=HASH_JOIN} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/apply-batch-true", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/apply-batch-false", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/merge", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=MERGE_JOIN} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/force-join-order", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE, FORCE_JOIN_ORDER=TRUE} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/build-left-unsupported", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/build-right-unsupported", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/push-broadcast", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/hash-one-pass", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/hash-multi-pass", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=MULTI_PASS} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/factorized-both-control", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=HASH_JOIN, FACTORIZED_MODE=FACTORIZE_BOTH} DISTINCT SELECT SingerId FROM Albums`},
	{
		Label: "set-operation/except-distinct/apply-batch-true-execution-row",
		SQL:   `@{EXECUTION_METHOD=ROW} SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} DISTINCT SELECT SingerId FROM Albums`,
	},
	{
		Label: "set-operation/except-distinct/apply-batch-true-execution-batch",
		SQL:   `@{EXECUTION_METHOD=BATCH} SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} DISTINCT SELECT SingerId FROM Albums`,
	},

	{Label: "set-operation/intersect-all/default", SQL: `SELECT SingerId FROM Albums INTERSECT ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-all/hash", SQL: `SELECT SingerId FROM Albums INTERSECT @{JOIN_METHOD=HASH_JOIN} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-all/merge", SQL: `SELECT SingerId FROM Albums INTERSECT @{JOIN_METHOD=MERGE_JOIN} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-all/push-broadcast", SQL: `SELECT SingerId FROM Albums INTERSECT @{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-all/apply-batch-true", SQL: `SELECT SingerId FROM Albums INTERSECT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-all/apply-batch-false", SQL: `SELECT SingerId FROM Albums INTERSECT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-all/hash-one-pass", SQL: `SELECT SingerId FROM Albums INTERSECT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-all/hash-multi-pass", SQL: `SELECT SingerId FROM Albums INTERSECT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=MULTI_PASS} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-all/build-left-unsupported", SQL: `SELECT SingerId FROM Albums INTERSECT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-all/build-right-unsupported", SQL: `SELECT SingerId FROM Albums INTERSECT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-all/force-join-order", SQL: `SELECT SingerId FROM Albums INTERSECT @{FORCE_JOIN_ORDER=TRUE} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/intersect-all/factorized-both", SQL: `SELECT SingerId FROM Albums INTERSECT @{JOIN_METHOD=HASH_JOIN, FACTORIZED_MODE=FACTORIZE_BOTH} ALL SELECT SingerId FROM Songs`},

	{Label: "set-operation/except-all/default", SQL: `SELECT SingerId FROM Albums EXCEPT ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/except-all/hash", SQL: `SELECT SingerId FROM Albums EXCEPT @{JOIN_METHOD=HASH_JOIN} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/except-all/merge", SQL: `SELECT SingerId FROM Albums EXCEPT @{JOIN_METHOD=MERGE_JOIN} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/except-all/push-broadcast", SQL: `SELECT SingerId FROM Albums EXCEPT @{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/except-all/apply-batch-true", SQL: `SELECT SingerId FROM Albums EXCEPT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/except-all/apply-batch-false", SQL: `SELECT SingerId FROM Albums EXCEPT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/except-all/hash-one-pass", SQL: `SELECT SingerId FROM Albums EXCEPT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=ONE_PASS} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/except-all/hash-multi-pass", SQL: `SELECT SingerId FROM Albums EXCEPT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_EXECUTION=MULTI_PASS} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/except-all/build-left-unsupported", SQL: `SELECT SingerId FROM Albums EXCEPT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/except-all/build-right-unsupported", SQL: `SELECT SingerId FROM Albums EXCEPT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/except-all/force-join-order", SQL: `SELECT SingerId FROM Albums EXCEPT @{FORCE_JOIN_ORDER=TRUE} ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/except-all/factorized-both-control", SQL: `SELECT SingerId FROM Albums EXCEPT @{JOIN_METHOD=HASH_JOIN, FACTORIZED_MODE=FACTORIZE_BOTH} ALL SELECT SingerId FROM Songs`},

	{Label: "set-operation/input-shape/intersect-distinct-reversed", SQL: `SELECT SingerId FROM Albums INTERSECT DISTINCT SELECT SingerId FROM Singers`},
	{Label: "set-operation/input-shape/except-distinct-reversed", SQL: `SELECT SingerId FROM Albums EXCEPT DISTINCT SELECT SingerId FROM Singers`},
	{Label: "set-operation/input-shape/intersect-all-reversed", SQL: `SELECT SingerId FROM Albums INTERSECT ALL SELECT SingerId FROM Singers`},
	{Label: "set-operation/input-shape/except-all-reversed", SQL: `SELECT SingerId FROM Albums EXCEPT ALL SELECT SingerId FROM Singers`},
	{Label: "set-operation/input-shape/intersect-distinct-both-duplicate", SQL: `SELECT SingerId FROM Albums INTERSECT DISTINCT SELECT SingerId FROM Songs`},
	{Label: "set-operation/input-shape/except-distinct-both-duplicate", SQL: `SELECT SingerId FROM Albums EXCEPT DISTINCT SELECT SingerId FROM Songs`},
	{Label: "set-operation/input-shape/intersect-all-both-duplicate", SQL: `SELECT SingerId FROM Albums INTERSECT ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/input-shape/except-all-both-duplicate", SQL: `SELECT SingerId FROM Albums EXCEPT ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/input-shape/union-distinct-multi-column", SQL: `SELECT SingerId, AlbumId FROM Albums UNION DISTINCT SELECT SingerId, AlbumId FROM Songs`},
	{Label: "set-operation/input-shape/intersect-distinct-multi-column", SQL: `SELECT SingerId, AlbumId FROM Albums INTERSECT DISTINCT SELECT SingerId, AlbumId FROM Songs`},
	{Label: "set-operation/input-shape/except-all-multi-column", SQL: `SELECT SingerId, AlbumId FROM Albums EXCEPT ALL SELECT SingerId, AlbumId FROM Songs`},
	{Label: "set-operation/input-shape/union-distinct-three-input", SQL: `SELECT SingerId FROM Singers UNION DISTINCT SELECT SingerId FROM Albums UNION DISTINCT SELECT SingerId FROM Songs`},
	{Label: "set-operation/input-shape/except-distinct-three-input", SQL: `SELECT SingerId FROM Singers EXCEPT DISTINCT SELECT SingerId FROM Albums EXCEPT DISTINCT SELECT SingerId FROM Songs`},
	{Label: "set-operation/input-shape/except-all-three-input", SQL: `SELECT SingerId FROM Singers EXCEPT ALL SELECT SingerId FROM Albums EXCEPT ALL SELECT SingerId FROM Songs`},
	{Label: "set-operation/input-shape/mixed-parenthesized", SQL: `(SELECT SingerId FROM Singers UNION ALL SELECT SingerId FROM Albums) INTERSECT DISTINCT SELECT SingerId FROM Songs`},
	{Label: "set-operation/input-shape/except-all-parenthesized-right", SQL: `SELECT SingerId FROM Singers EXCEPT ALL (SELECT SingerId FROM Albums EXCEPT ALL SELECT SingerId FROM Songs)`},

	{Label: "set-operation/intersect-distinct/rewrite-exists/default", SQL: `SELECT DISTINCT a.SingerId FROM Albums AS a WHERE EXISTS (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId)`},
	{Label: "set-operation/intersect-distinct/rewrite-exists/hash-build-left", SQL: `SELECT DISTINCT a.SingerId FROM Albums AS a WHERE EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId)`},
	{Label: "set-operation/intersect-distinct/rewrite-exists/hash-build-right", SQL: `SELECT DISTINCT a.SingerId FROM Albums AS a WHERE EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT} (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId)`},
	{Label: "set-operation/intersect-distinct/rewrite-exists/apply-batch-true", SQL: `SELECT DISTINCT a.SingerId FROM Albums AS a WHERE EXISTS @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId)`},
	{Label: "set-operation/intersect-distinct/rewrite-exists/push-broadcast", SQL: `SELECT DISTINCT a.SingerId FROM Albums AS a WHERE EXISTS @{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId)`},
	{Label: "set-operation/intersect-distinct/rewrite-exists/group-hash", SQL: `SELECT a.SingerId FROM Albums AS a WHERE EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId) GROUP @{GROUP_METHOD=HASH_GROUP} BY a.SingerId`},
	{Label: "set-operation/intersect-distinct/rewrite-exists/group-stream", SQL: `SELECT a.SingerId FROM Albums AS a WHERE EXISTS @{JOIN_METHOD=MERGE_JOIN} (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId) GROUP @{GROUP_METHOD=STREAM_GROUP} BY a.SingerId`},
	{Label: "set-operation/intersect-distinct/rewrite-exists/multi-predicate-mixed-join-methods", SQL: `SELECT DISTINCT r.K FROM SetOpR AS r WHERE EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} (SELECT 1 FROM SetOpS AS s WHERE s.K = r.K) AND EXISTS @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} (SELECT 1 FROM SetOpT AS t WHERE t.K = r.K)`},
	{Label: "set-operation/intersect-distinct/multi-operator-second-hint-unsupported", SQL: `SELECT K FROM SetOpR INTERSECT @{JOIN_METHOD=HASH_JOIN} DISTINCT SELECT K FROM SetOpS INTERSECT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} DISTINCT SELECT K FROM SetOpT`},

	{Label: "set-operation/except-distinct/rewrite-not-exists/default", SQL: `SELECT DISTINCT a.SingerId FROM Albums AS a WHERE NOT EXISTS (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId)`},
	{Label: "set-operation/except-distinct/rewrite-not-exists/hash-build-left", SQL: `SELECT DISTINCT a.SingerId FROM Albums AS a WHERE NOT EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId)`},
	{Label: "set-operation/except-distinct/rewrite-not-exists/hash-build-right", SQL: `SELECT DISTINCT a.SingerId FROM Albums AS a WHERE NOT EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT} (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId)`},
	{Label: "set-operation/except-distinct/rewrite-not-exists/apply-batch-true", SQL: `SELECT DISTINCT a.SingerId FROM Albums AS a WHERE NOT EXISTS @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId)`},
	{Label: "set-operation/except-distinct/rewrite-not-exists/push-broadcast", SQL: `SELECT DISTINCT a.SingerId FROM Albums AS a WHERE NOT EXISTS @{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId)`},
	{Label: "set-operation/except-distinct/rewrite-not-exists/group-hash", SQL: `SELECT a.SingerId FROM Albums AS a WHERE NOT EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId) GROUP @{GROUP_METHOD=HASH_GROUP} BY a.SingerId`},
	{Label: "set-operation/except-distinct/rewrite-not-exists/group-stream", SQL: `SELECT a.SingerId FROM Albums AS a WHERE NOT EXISTS @{JOIN_METHOD=MERGE_JOIN} (SELECT 1 FROM Songs AS s WHERE s.SingerId = a.SingerId) GROUP @{GROUP_METHOD=STREAM_GROUP} BY a.SingerId`},
	{Label: "set-operation/except-distinct/rewrite-not-exists/multi-predicate-mixed-join-methods", SQL: `SELECT DISTINCT r.K FROM SetOpR AS r WHERE NOT EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} (SELECT 1 FROM SetOpS AS s WHERE s.K = r.K) AND NOT EXISTS @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} (SELECT 1 FROM SetOpT AS t WHERE t.K = r.K)`},
	{Label: "set-operation/except-distinct/multi-operator-second-hint-unsupported", SQL: `SELECT K FROM SetOpR EXCEPT @{JOIN_METHOD=HASH_JOIN} DISTINCT SELECT K FROM SetOpS EXCEPT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} DISTINCT SELECT K FROM SetOpT`},

	{Label: "distinct/index-prefix/default", SQL: `SELECT DISTINCT FirstName FROM Singers@{FORCE_INDEX=SingersByFirstLastName}`},
	{Label: "distinct/index-prefix/groupby-scan-true-control", SQL: `SELECT DISTINCT FirstName FROM Singers@{FORCE_INDEX=SingersByFirstLastName, GROUPBY_SCAN_OPTIMIZATION=TRUE}`},
	{Label: "distinct/index-prefix/groupby-scan-false-control", SQL: `SELECT DISTINCT FirstName FROM Singers@{FORCE_INDEX=SingersByFirstLastName, GROUPBY_SCAN_OPTIMIZATION=FALSE}`},
	{Label: "distinct/base-table/default", SQL: `SELECT DISTINCT BirthDate FROM Singers@{FORCE_INDEX=_BASE_TABLE}`},
	{Label: "distinct/base-table/groupby-scan-true-control", SQL: `SELECT DISTINCT BirthDate FROM Singers@{FORCE_INDEX=_BASE_TABLE, GROUPBY_SCAN_OPTIMIZATION=TRUE}`},
	{Label: "distinct/base-table/groupby-scan-false-control", SQL: `SELECT DISTINCT BirthDate FROM Singers@{FORCE_INDEX=_BASE_TABLE, GROUPBY_SCAN_OPTIMIZATION=FALSE}`},
	{Label: "distinct/group-hash-unsupported", SQL: `SELECT @{GROUP_METHOD=HASH_GROUP} DISTINCT FirstName FROM Singers`},
	{Label: "distinct/group-stream-unsupported", SQL: `SELECT @{GROUP_METHOD=STREAM_GROUP} DISTINCT FirstName FROM Singers`},

	{Label: "distinct/rewrite-group-by-hash", SQL: `SELECT FirstName FROM Singers@{FORCE_INDEX=SingersByFirstLastName} GROUP@{GROUP_METHOD=HASH_GROUP} BY FirstName`},
	{Label: "distinct/rewrite-group-by-stream", SQL: `SELECT FirstName FROM Singers@{FORCE_INDEX=SingersByFirstLastName} GROUP@{GROUP_METHOD=STREAM_GROUP} BY FirstName`},
	{Label: "set-operation/union-distinct/rewrite-group-by-hash", SQL: `SELECT SingerId FROM (SELECT SingerId FROM Singers UNION ALL SELECT SingerId FROM Albums) GROUP@{GROUP_METHOD=HASH_GROUP} BY SingerId`},
	{Label: "set-operation/union-distinct/rewrite-group-by-stream", SQL: `SELECT SingerId FROM (SELECT SingerId FROM Singers UNION ALL SELECT SingerId FROM Albums) GROUP@{GROUP_METHOD=STREAM_GROUP} BY SingerId`},
}
