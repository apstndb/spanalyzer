package main

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

	{Label: "set-operation/union-distinct/default", SQL: `SELECT SingerId FROM Singers UNION DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-distinct/hash-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=HASH_JOIN} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-distinct/apply-batch-true-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/union-distinct/apply-batch-false-control", SQL: `SELECT SingerId FROM Singers UNION @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} DISTINCT SELECT SingerId FROM Albums`},
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

	{Label: "set-operation/except-distinct/default", SQL: `SELECT SingerId FROM Singers EXCEPT DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/hash", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=HASH_JOIN} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/apply-batch-true", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=TRUE} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/apply-batch-false", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/merge", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=MERGE_JOIN} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/force-join-order", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE, FORCE_JOIN_ORDER=TRUE} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/build-left-unsupported", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} DISTINCT SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-distinct/build-right-unsupported", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_RIGHT} DISTINCT SELECT SingerId FROM Albums`},

	{Label: "set-operation/intersect-all/default", SQL: `SELECT SingerId FROM Singers INTERSECT ALL SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-all/hash-control", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=HASH_JOIN} ALL SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-all/merge-control", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=MERGE_JOIN} ALL SELECT SingerId FROM Albums`},
	{Label: "set-operation/intersect-all/apply-batch-false", SQL: `SELECT SingerId FROM Singers INTERSECT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} ALL SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-all/default", SQL: `SELECT SingerId FROM Singers EXCEPT ALL SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-all/hash", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=HASH_JOIN} ALL SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-all/merge", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=MERGE_JOIN} ALL SELECT SingerId FROM Albums`},
	{Label: "set-operation/except-all/apply-batch-false", SQL: `SELECT SingerId FROM Singers EXCEPT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} ALL SELECT SingerId FROM Albums`},

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
