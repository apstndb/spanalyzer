package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestBuildQueryMatrixCases(t *testing.T) {
	got := buildQueryMatrixCases("matrix", `SELECT {{.Projection.SQL}} FROM {{.Table.SQL}}`, queryMatrixAxis{
		Name: "Projection",
		Values: []queryMatrixAxisValue{
			{Label: "one", Fields: map[string]string{"SQL": "1"}},
			{Label: "two", Fields: map[string]string{"SQL": "2"}},
		},
	}, queryMatrixAxis{
		Name: "Table",
		Values: []queryMatrixAxisValue{
			{Label: "singers", Fields: map[string]string{"SQL": "Singers"}},
		},
	})
	if gotLen, wantLen := len(got), 2; gotLen != wantLen {
		t.Fatalf("len(buildQueryMatrixCases()) = %d, want %d", gotLen, wantLen)
	}
	if gotLabel, wantLabel := got[0].Label, "matrix/one/singers"; gotLabel != wantLabel {
		t.Fatalf("first label = %q, want %q", gotLabel, wantLabel)
	}
	if gotSQL, wantSQL := got[0].SQL, "SELECT 1 FROM Singers"; gotSQL != wantSQL {
		t.Fatalf("first SQL = %q, want %q", gotSQL, wantSQL)
	}
	if gotLabel, wantLabel := got[1].Label, "matrix/two/singers"; gotLabel != wantLabel {
		t.Fatalf("second label = %q, want %q", gotLabel, wantLabel)
	}
	if gotSQL, wantSQL := got[1].SQL, "SELECT 2 FROM Singers"; gotSQL != wantSQL {
		t.Fatalf("second SQL = %q, want %q", gotSQL, wantSQL)
	}
}

func TestLoadQueriesSQLFileSplitsStatementsAndPreservesFormatting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queries.sql")
	input := `@{OPTIMIZER_VERSION=1}
SELECT SingerId, AlbumId, TrackId
FROM Songs
WHERE REGEXP_CONTAINS(SongName, "^A.*");

INSERT INTO Singers (SingerId, FirstName)
VALUES (1, "Alice");
`
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	queries, err := loadQueries("docs", nil, []string{path})
	if err != nil {
		t.Fatalf("loadQueries() error = %v", err)
	}
	if got, want := len(queries), 2; got != want {
		t.Fatalf("len(queries) = %d, want %d", got, want)
	}
	if got, want := queries[0].Label, path+"#1"; got != want {
		t.Fatalf("queries[0].Label = %q, want %q", got, want)
	}
	if got := queries[0].SQL; !strings.Contains(got, "\nFROM Songs\n") {
		t.Fatalf("queries[0].SQL did not preserve formatting: %q", got)
	}
	if got, want := queries[0].effectivePlanMode(), planModeReadOnly; got != want {
		t.Fatalf("queries[0] plan mode = %q, want %q", got, want)
	}
	if got, want := queries[1].Label, path+"#2"; got != want {
		t.Fatalf("queries[1].Label = %q, want %q", got, want)
	}
	if got, want := queries[1].effectivePlanMode(), planModeReadWrite; got != want {
		t.Fatalf("queries[1] plan mode = %q, want %q", got, want)
	}
}

func TestLoadQueriesSQLFileAllowsSetOperationHintsNewerThanParser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "set-operations.sql")
	input := `SELECT SingerId FROM Singers
INTERSECT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} DISTINCT
SELECT SingerId FROM Albums;

SELECT SingerId FROM Singers
EXCEPT @{JOIN_METHOD=HASH_JOIN} ALL
SELECT SingerId FROM Albums;

-- The DML prefix must survive parser fallback too.
@{OPTIMIZER_VERSION=8}
INSERT INTO Singers (SingerId)
SELECT SingerId FROM Singers
INTERSECT @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} DISTINCT
SELECT SingerId FROM Albums;
`
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	queries, err := loadQueries("docs", nil, []string{path})
	if err != nil {
		t.Fatalf("loadQueries() error = %v", err)
	}
	if got, want := len(queries), 3; got != want {
		t.Fatalf("len(queries) = %d, want %d", got, want)
	}
	if got := queries[0].SQL; !strings.Contains(got, "INTERSECT @{JOIN_METHOD=APPLY_JOIN") {
		t.Fatalf("queries[0].SQL lost the set-operation hint: %q", got)
	}
	if got := queries[1].SQL; !strings.Contains(got, "EXCEPT @{JOIN_METHOD=HASH_JOIN} ALL") {
		t.Fatalf("queries[1].SQL lost the set-operation hint: %q", got)
	}
	for _, query := range queries[:2] {
		if got, want := query.effectivePlanMode(), planModeReadOnly; got != want {
			t.Errorf("%s plan mode = %q, want %q", query.Label, got, want)
		}
	}
	if got, want := queries[2].effectivePlanMode(), planModeReadWrite; got != want {
		t.Errorf("queries[2] plan mode = %q, want %q", got, want)
	}
}

func TestLoadQueriesDML(t *testing.T) {
	queries, err := loadQueries("dml", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "dml", err)
	}
	if len(queries) == 0 {
		t.Fatal("loadQueries(\"dml\") returned no queries")
	}
	seen := map[string]bool{}
	for _, query := range queries {
		seen[query.Label] = true
		if got, want := query.effectivePlanMode(), planModeReadWrite; got != want {
			t.Fatalf("%s plan mode = %q, want %q", query.Label, got, want)
		}
	}
	for _, label := range []string{
		"dml/insert-values",
		"dml/insert-ignore",
		"dml/insert-or-ignore",
		"dml/insert-or-update",
		"dml/insert-on-conflict-do-nothing",
		"dml/insert-on-conflict-do-update",
		"dml/update-literal",
		"dml/delete-where",
	} {
		if !seen[label] {
			t.Fatalf("loadQueries(\"dml\") missing %s", label)
		}
	}
}

func TestLoadDDLsDMLIncludesDMLOnlyObjects(t *testing.T) {
	ddls, err := loadDDLs("dml", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "dml", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{
		"ALTER TABLE Singers ADD COLUMN Status",
		"CREATE UNIQUE INDEX UniqueIndex_SingerName",
		"CREATE TABLE AckworthSingers",
		"CREATE TABLE Fans",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("loadDDLs(\"dml\") missing %q in:\n%s", want, joined)
		}
	}
}

func TestLoadDDLsOptimizerGapsIncludesDedicatedObjects(t *testing.T) {
	ddls, err := loadDDLs("optimizer_gaps", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "optimizer_gaps", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{
		"CREATE TABLE Venues",
		"CREATE TABLE FKCustomers",
		"CREATE TABLE FKOrders",
		"REFERENCES FKCustomers",
		"NOT ENFORCED",
		"CREATE TABLE AckworthSingers",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("loadDDLs(\"optimizer_gaps\") missing %q in:\n%s", want, joined)
		}
	}
}

func TestLoadDDLsJoinEliminationIncludesControlledSchemas(t *testing.T) {
	ddls, err := loadDDLs("join_elimination", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "join_elimination", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{
		"INTERLEAVE IN PARENT Singers",
		"CREATE TABLE Concerts",
		"CREATE TABLE FkAlbums",
		"CREATE TABLE FkAlbumsNotEnforced",
		"CREATE TABLE FkAlbumsNotEnforcedNonLeadingPK",
		"REFERENCES FkSingers",
		"CREATE TABLE FKOrders",
		"CREATE TABLE FKOrdersEnforced",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("loadDDLs(\"join_elimination\") missing %q in:\n%s", want, joined)
		}
	}
}

func TestLoadDDLsDocsIncludesTimestampPushdownPrerequisites(t *testing.T) {
	ddls, err := loadDDLs("docs", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "docs", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{
		"SingerInfo BYTES(MAX)",
		"ModificationTime TIMESTAMP OPTIONS (allow_commit_timestamp = true)",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("loadDDLs(\"docs\") missing %q in:\n%s", want, joined)
		}
	}
}

func TestLoadQueriesDocsIncludesIndependentFullOuterJoinProbes(t *testing.T) {
	queries, err := loadQueries("docs", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "docs", err)
	}
	seen := map[string]string{}
	for _, query := range queries {
		seen[query.Label] = query.SQL
	}
	for _, label := range []string{
		"binary/hash-join-full-outer",
		"binary/merge-join-full-outer",
	} {
		if seen[label] == "" {
			t.Fatalf("loadQueries(\"docs\") missing %s", label)
		}
		if !strings.Contains(seen[label], "FULL JOIN") || !strings.Contains(seen[label], "Concerts") {
			t.Fatalf("%s does not use the independent-table FULL JOIN shape: %s", label, seen[label])
		}
	}
}

func TestLoadQueriesJoinEliminationIncludesControls(t *testing.T) {
	queries, err := loadQueries("join_elimination", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "join_elimination", err)
	}
	seen := map[string]string{}
	for _, query := range queries {
		seen[query.Label] = query.SQL
	}
	for _, label := range []string{
		"join-elimination/interleave",
		"join-elimination/interleave-hash-join-hint",
		"join-elimination/no-constraint-control",
		"join-elimination/enforced-fk-leading-key",
		"join-elimination/unenforced-fk-leading-key-true",
		"join-elimination/unenforced-fk-leading-key-default",
		"join-elimination/unenforced-fk-leading-key-child-first-true",
		"join-elimination/unenforced-fk-leading-key-true-hash-join-hint",
		"join-elimination/unenforced-fk-leading-key-false",
		"join-elimination/unenforced-fk-in-pk-nonleading-true",
		"join-elimination/enforced-fk-outside-pk",
		"join-elimination/unenforced-fk-outside-pk-true",
		"join-elimination/unenforced-fk-outside-pk-false",
	} {
		if seen[label] == "" {
			t.Fatalf("loadQueries(\"join_elimination\") missing %s", label)
		}
	}
	if !strings.Contains(seen["join-elimination/unenforced-fk-leading-key-true"], "USE_UNENFORCED_FOREIGN_KEY=TRUE") {
		t.Fatal("leading-key TRUE probe is missing USE_UNENFORCED_FOREIGN_KEY=TRUE")
	}
	if !strings.Contains(seen["join-elimination/unenforced-fk-leading-key-true-hash-join-hint"], "JOIN_METHOD=HASH_JOIN") {
		t.Fatal("leading-key TRUE hint probe is missing JOIN_METHOD=HASH_JOIN")
	}
	if !strings.Contains(seen["join-elimination/unenforced-fk-outside-pk-false"], "USE_UNENFORCED_FOREIGN_KEY=FALSE") {
		t.Fatal("outside-PK FALSE control is missing USE_UNENFORCED_FOREIGN_KEY=FALSE")
	}
}

func TestLoadQueriesPlanVocabInferenceIncludesHypothesesAndControls(t *testing.T) {
	t.Parallel()

	queries, err := loadQueries("planvocab_inference", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "planvocab_inference", err)
	}
	seen := map[string]string{}
	for _, query := range queries {
		seen[query.Label] = query.SQL
	}
	for _, label := range []string{
		"planvocab-inference/dca-order/default",
		"planvocab-inference/dca-order/no-distributed-merge",
		"planvocab-inference/merge-right-outer",
		"planvocab-inference/merge-left-outer-control",
		"planvocab-inference/merge-one-to-one",
		"planvocab-inference/merge-one-to-many-control",
		"planvocab-inference/offset-v1",
		"planvocab-inference/offset-v3",
		"planvocab-inference/offset-v8",
		"planvocab-inference/offset-control",
		"planvocab-inference/hash-left-outer-residual-build-left",
		"planvocab-inference/hash-left-outer-residual-build-right",
		"planvocab-inference/hash-left-outer-control",
		"planvocab-inference/minor-sort-values",
		"planvocab-inference/minor-sort-limit-values",
		"planvocab-inference/aggregate-hash-repeated-key-agg",
		"planvocab-inference/aggregate-stream-repeated-key-agg",
	} {
		if seen[label] == "" {
			t.Errorf("loadQueries(\"planvocab_inference\") missing %q", label)
		}
	}
	if got, want := len(queries), len(seen); got != want {
		t.Fatalf("planvocab inference queries include duplicate labels: queries=%d unique=%d", got, want)
	}
}

func TestLoadQueriesHintPositionAuditIncludesMissingPositionsAndRejections(t *testing.T) {
	queries, err := loadQueries("hint_position_audit", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "hint_position_audit", err)
	}
	seen := map[string]queryCase{}
	for _, query := range queries {
		seen[query.Label] = query
	}
	for _, label := range []string{
		"hint-position/accepted/select",
		"hint-position/accepted/order-by",
		"hint-position/accepted/set-operation-first",
		"hint-position/rejected/set-operation-second-same-level",
		"hint-position/accepted/sql-exists",
		"hint-position/accepted/in-subquery",
		"hint-position/rejected/in-value-list",
		"hint-position/rejected/in-unnest",
		"hint-position/accepted/like-any-subquery",
		"hint-position/accepted/like-some-subquery",
		"hint-position/accepted/like-all-subquery",
		"hint-position/accepted/like-some-multi-hint-subquery",
		"hint-position/accepted/like-all-multi-hint-subquery",
		"hint-position/rejected/like-any-value-list",
		"hint-position/rejected/like-some-value-list",
		"hint-position/rejected/like-all-value-list",
		"hint-position/accepted/quantified-subquery",
		"hint-position/accepted/quantified-some-subquery",
		"hint-position/accepted/quantified-all-subquery",
		"hint-position/accepted/quantified-some-multi-hint-subquery",
		"hint-position/accepted/quantified-all-multi-hint-subquery",
		"hint-position/rejected/quantified-value-list",
		"hint-position/rejected/quantified-some-value-list",
		"hint-position/rejected/quantified-all-value-list",
		"hint-position/accepted/window-partition",
		"hint-position/accepted/tvf",
		"hint-position/accepted/gql-return",
		"hint-position/accepted/gql-order-by",
		"hint-position/accepted/gql-with",
		"hint-position/accepted/gql-value",
		"hint-position/accepted/gql-exists",
		"hint-position/rejected/gql-set-operation",
		"hint-position/rejected/gql-subpath-leading",
		"hint-position/rejected/gql-between-edges",
		"hint-position/accepted/pipe-log-unsupported",
		"hint-position/versioned/pipe-finish",
		"hint-position/accepted/insert-target",
		"hint-position/accepted/dml-statement-pdml",
	} {
		if _, ok := seen[label]; !ok {
			t.Errorf("loadQueries(\"hint_position_audit\") missing %q", label)
		}
	}
	if got, want := len(queries), len(seen); got != want {
		t.Fatalf("hint-position queries include duplicate labels: queries=%d unique=%d", got, want)
	}
	for _, auditCase := range hintPositionAuditCases {
		if strings.HasPrefix(auditCase.Query.Label, "hint-position/versioned/") {
			continue
		}
		prefix := "hint-position/" + string(auditCase.Expectation) + "/"
		if !strings.HasPrefix(auditCase.Query.Label, prefix) {
			t.Errorf("case label %q does not match expectation %q", auditCase.Query.Label, auditCase.Expectation)
		}
	}
	for _, label := range []string{
		"hint-position/accepted/insert-target",
		"hint-position/accepted/dml-statement-pdml",
	} {
		if got := seen[label].effectivePlanMode(); got != planModeReadWrite {
			t.Errorf("%s plan mode = %q, want %q", label, got, planModeReadWrite)
		}
	}
}

func TestLoadQueriesHintPositionCombinationsIncludesPlanEffectsAndControls(t *testing.T) {
	queries, err := loadQueries("hint_position_combinations", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "hint_position_combinations", err)
	}
	seen := make(map[string]queryCase, len(queries))
	for _, query := range queries {
		seen[query.Label] = query
	}
	expectedLabels := []string{
		"hint-combination/sql-exists/hash-build-left-one-pass",
		"hint-combination/sql-exists/hash-build-right-one-pass",
		"hint-combination/sql-exists/apply-batch-true",
		"hint-combination/sql-exists/apply-batch-false",
		"hint-combination/in-subquery/hash-build-left-one-pass",
		"hint-combination/in-subquery/hash-build-right-one-pass",
		"hint-combination/in-subquery/apply-batch-true",
		"hint-combination/in-subquery/apply-batch-false",
		"hint-combination/statement-plus-sql-exists/hash-build-left-one-pass",
		"hint-combination/set-operation/hash-one-pass-control",
		"hint-combination/scalar-sql-exists/hash-build-left-one-pass-control",
		"hint-combination/scalar-in-subquery/hash-build-left-one-pass-control",
		"hint-combination/gql-exists/hash-build-left-one-pass-control",
	}
	for _, label := range expectedLabels {
		if _, ok := seen[label]; !ok {
			t.Errorf("loadQueries(\"hint_position_combinations\") missing %q", label)
		}
	}
	if got, want := len(queries), len(expectedLabels); got != want {
		t.Fatalf("hint-position combination query count = %d, want %d", got, want)
	}
	if got, want := len(queries), len(seen); got != want {
		t.Fatalf("hint-position combination queries include duplicate labels: queries=%d unique=%d", got, want)
	}
	for _, label := range []string{
		"hint-combination/sql-exists/hash-build-left-one-pass",
		"hint-combination/in-subquery/hash-build-left-one-pass",
	} {
		for _, assignment := range []string{
			"JOIN_METHOD=HASH_JOIN",
			"HASH_JOIN_BUILD_SIDE=BUILD_LEFT",
			"HASH_JOIN_EXECUTION=ONE_PASS",
		} {
			if !strings.Contains(seen[label].SQL, assignment) {
				t.Errorf("%s missing %q", label, assignment)
			}
		}
	}

	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "hint_position_combination_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Version, "v0alpha1"; got != want {
		t.Errorf("expectation version = %q, want %q", got, want)
	}
	if got, want := len(manifest.Queries), 9; got != want {
		t.Fatalf("expectation query count = %d, want %d", got, want)
	}
	manifestLabels := make(map[string]struct{}, len(manifest.Queries))
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("expectation label %q is absent from the built-in case", expectation.Label)
		}
		if _, duplicate := manifestLabels[expectation.Label]; duplicate {
			t.Errorf("expectation label %q is duplicated", expectation.Label)
		}
		manifestLabels[expectation.Label] = struct{}{}
		if len(expectation.Patterns) == 0 {
			t.Errorf("expectation label %q has no operator patterns", expectation.Label)
		}
	}
	for _, controlLabel := range []string{
		"hint-combination/set-operation/hash-one-pass-control",
		"hint-combination/scalar-sql-exists/hash-build-left-one-pass-control",
		"hint-combination/scalar-in-subquery/hash-build-left-one-pass-control",
		"hint-combination/gql-exists/hash-build-left-one-pass-control",
	} {
		if _, ok := manifestLabels[controlLabel]; ok {
			t.Errorf("syntax-only control %q must not have a plan-effect expectation", controlLabel)
		}
	}
}

func TestLoadQueriesSetOperationDistinctIncludesEffectsAndControls(t *testing.T) {
	queries, err := loadQueries("set_operation_distinct", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "set_operation_distinct", err)
	}
	seen := make(map[string]queryCase, len(queries))
	for _, query := range queries {
		if _, duplicate := seen[query.Label]; duplicate {
			t.Errorf("duplicate query label %q", query.Label)
		}
		seen[query.Label] = query
	}
	for _, label := range []string{
		"set-operation/intersect-distinct/hash",
		"set-operation/intersect-distinct/apply-batch-true",
		"set-operation/intersect-distinct/force-join-order",
		"set-operation/intersect-distinct/three-input-force-join-order",
		"set-operation/except-distinct/merge",
		"set-operation/intersect-all/apply-batch-false",
		"set-operation/except-all/hash",
		"set-operation/except-all/hash-one-pass",
		"set-operation/except-all/build-left-unsupported",
		"set-operation/intersect-distinct/push-broadcast",
		"set-operation/intersect-distinct/apply-batch-true-execution-row",
		"set-operation/intersect-all/apply-batch-true",
		"set-operation/intersect-all/factorized-both",
		"set-operation/input-shape/except-all-reversed",
		"set-operation/input-shape/intersect-distinct-multi-column",
		"set-operation/input-shape/mixed-parenthesized",
		"set-operation/intersect-distinct/rewrite-exists/hash-build-right",
		"set-operation/intersect-distinct/rewrite-exists/group-stream",
		"set-operation/intersect-distinct/rewrite-exists/multi-predicate-mixed-join-methods",
		"set-operation/intersect-distinct/multi-operator-second-hint-unsupported",
		"set-operation/except-distinct/rewrite-not-exists/apply-batch-true",
		"set-operation/except-distinct/rewrite-not-exists/group-hash",
		"set-operation/except-distinct/rewrite-not-exists/multi-predicate-mixed-join-methods",
		"set-operation/except-distinct/multi-operator-second-hint-unsupported",
		"distinct/index-prefix/groupby-scan-true-control",
		"distinct/base-table/groupby-scan-false-control",
		"distinct/rewrite-group-by-stream",
		"set-operation/union-distinct/rewrite-group-by-hash",
	} {
		if _, ok := seen[label]; !ok {
			t.Errorf("loadQueries(\"set_operation_distinct\") missing %q", label)
		}
	}
	if got, want := len(queries), 116; got != want {
		t.Fatalf("set-operation/distinct query count = %d, want %d", got, want)
	}

	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "set_operation_distinct_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
		ExpectedQueryErrors []struct {
			Label    string `json:"label"`
			Contains string `json:"contains"`
		} `json:"expected_query_errors"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Version, "v0alpha1"; got != want {
		t.Errorf("expectation version = %q, want %q", got, want)
	}
	if got, want := len(manifest.Queries), 47; got != want {
		t.Errorf("set-operation positive expectations = %d, want %d", got, want)
	}
	if got, want := len(manifest.ExpectedQueryErrors), 17; got != want {
		t.Errorf("set-operation error expectations = %d, want %d", got, want)
	}
	manifestLabels := make(map[string]struct{}, len(manifest.Queries))
	patternCount := 0
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("expectation label %q is absent from the built-in case", expectation.Label)
		}
		if _, duplicate := manifestLabels[expectation.Label]; duplicate {
			t.Errorf("expectation label %q is duplicated", expectation.Label)
		}
		manifestLabels[expectation.Label] = struct{}{}
		if len(expectation.Patterns) == 0 {
			t.Errorf("expectation label %q has no operator patterns", expectation.Label)
		}
		patternCount += len(expectation.Patterns)
	}
	if got, want := patternCount, 77; got != want {
		t.Errorf("set-operation operator patterns = %d, want %d", got, want)
	}
	for _, controlLabel := range []string{
		"set-operation/union-all/hash-control",
		"set-operation/union-all/apply-batch-true-control",
		"set-operation/union-distinct/hash-control",
		"set-operation/union-distinct/group-hash-unsupported",
		"set-operation/intersect-distinct/build-left-unsupported",
		"set-operation/except-distinct/build-right-unsupported",
		"distinct/index-prefix/groupby-scan-true-control",
		"distinct/group-stream-unsupported",
	} {
		if _, ok := manifestLabels[controlLabel]; ok {
			t.Errorf("acceptance-only control %q must not have a plan-effect expectation", controlLabel)
		}
	}
	expectedErrorLabels := make(map[string]struct{}, len(manifest.ExpectedQueryErrors))
	for _, expectation := range manifest.ExpectedQueryErrors {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("expected query-error label %q is absent from the built-in case", expectation.Label)
		}
		if expectation.Contains == "" {
			t.Errorf("expected query-error label %q has empty matching text", expectation.Label)
		}
		if _, duplicate := expectedErrorLabels[expectation.Label]; duplicate {
			t.Errorf("expected query-error label %q is duplicated", expectation.Label)
		}
		if _, duplicate := manifestLabels[expectation.Label]; duplicate {
			t.Errorf("expected query-error label %q also has positive plan expectations", expectation.Label)
		}
		expectedErrorLabels[expectation.Label] = struct{}{}
	}
	for _, label := range []string{
		"set-operation/union-distinct/group-hash-unsupported",
		"set-operation/union-distinct/group-stream-unsupported",
		"set-operation/intersect-distinct/build-left-unsupported",
		"set-operation/intersect-distinct/build-right-unsupported",
		"set-operation/intersect-distinct/factorized-both-control",
		"set-operation/except-distinct/build-left-unsupported",
		"set-operation/except-distinct/build-right-unsupported",
		"set-operation/except-distinct/factorized-both-control",
		"set-operation/intersect-all/build-left-unsupported",
		"set-operation/intersect-all/build-right-unsupported",
		"set-operation/except-all/build-left-unsupported",
		"set-operation/except-all/build-right-unsupported",
		"set-operation/except-all/factorized-both-control",
		"set-operation/intersect-distinct/multi-operator-second-hint-unsupported",
		"set-operation/except-distinct/multi-operator-second-hint-unsupported",
		"distinct/group-hash-unsupported",
		"distinct/group-stream-unsupported",
	} {
		if _, ok := expectedErrorLabels[label]; !ok {
			t.Errorf("expected query-error manifest is missing %q", label)
		}
	}

	// These cases are still submitted to Omni and observed by planvocab, but
	// intentionally have no retained positive or error contract. Pinning the
	// exact set prevents a newly added case from silently landing in that gap.
	intentionallyUnasserted := make(map[string]struct{})
	for _, label := range []string{
		"distinct/base-table/groupby-scan-false-control",
		"distinct/base-table/groupby-scan-true-control",
		"distinct/index-prefix/groupby-scan-false-control",
		"distinct/index-prefix/groupby-scan-true-control",
		"set-operation/except-all/default",
		"set-operation/except-all/force-join-order",
		"set-operation/except-all/hash-multi-pass",
		"set-operation/except-all/hash-one-pass",
		"set-operation/except-distinct/apply-batch-true-execution-batch",
		"set-operation/except-distinct/apply-batch-true-execution-row",
		"set-operation/except-distinct/default",
		"set-operation/except-distinct/force-join-order",
		"set-operation/except-distinct/hash-multi-pass",
		"set-operation/except-distinct/hash-one-pass",
		"set-operation/except-distinct/push-broadcast",
		"set-operation/input-shape/except-all-both-duplicate",
		"set-operation/input-shape/except-all-multi-column",
		"set-operation/input-shape/except-all-parenthesized-right",
		"set-operation/input-shape/except-all-reversed",
		"set-operation/input-shape/except-all-three-input",
		"set-operation/input-shape/except-distinct-reversed",
		"set-operation/input-shape/except-distinct-three-input",
		"set-operation/input-shape/intersect-all-both-duplicate",
		"set-operation/input-shape/intersect-all-reversed",
		"set-operation/input-shape/intersect-distinct-multi-column",
		"set-operation/input-shape/intersect-distinct-reversed",
		"set-operation/input-shape/mixed-parenthesized",
		"set-operation/input-shape/union-distinct-multi-column",
		"set-operation/input-shape/union-distinct-three-input",
		"set-operation/intersect-all/default",
		"set-operation/intersect-all/force-join-order",
		"set-operation/intersect-all/hash-multi-pass",
		"set-operation/intersect-all/hash-one-pass",
		"set-operation/intersect-distinct/apply-batch-true-execution-batch",
		"set-operation/intersect-distinct/apply-batch-true-execution-row",
		"set-operation/intersect-distinct/default",
		"set-operation/intersect-distinct/hash-multi-pass",
		"set-operation/intersect-distinct/hash-one-pass",
		"set-operation/intersect-distinct/push-broadcast",
		"set-operation/intersect-distinct/three-input-default",
		"set-operation/intersect-distinct/three-input-force-join-order",
		"set-operation/union-all/apply-batch-false-control",
		"set-operation/union-all/apply-batch-true-control",
		"set-operation/union-all/default",
		"set-operation/union-all/hash-control",
		"set-operation/union-all/merge-control",
		"set-operation/union-all/push-broadcast-control",
		"set-operation/union-distinct/apply-batch-false-control",
		"set-operation/union-distinct/apply-batch-true-control",
		"set-operation/union-distinct/hash-control",
		"set-operation/union-distinct/merge-control",
		"set-operation/union-distinct/push-broadcast-control",
	} {
		intentionallyUnasserted[label] = struct{}{}
	}
	for label := range seen {
		if _, ok := manifestLabels[label]; ok {
			continue
		}
		if _, ok := expectedErrorLabels[label]; ok {
			continue
		}
		if _, ok := intentionallyUnasserted[label]; !ok {
			t.Errorf("set-operation case %q has no plan, error, or intentional-unasserted classification", label)
		}
		delete(intentionallyUnasserted, label)
	}
	for label := range intentionallyUnasserted {
		t.Errorf("intentional-unasserted set-operation label %q is absent from the built-in cases", label)
	}
}

func TestLoadDDLsSetOperationDistinctIncludesIndependentThreeWayInputs(t *testing.T) {
	ddls, err := loadDDLs("set_operation_distinct", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "set_operation_distinct", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, table := range []string{"SetOpR", "SetOpS", "SetOpT"} {
		if !strings.Contains(joined, "CREATE TABLE "+table) {
			t.Errorf("set-operation schema is missing independent table %q", table)
		}
	}
}

func TestLoadQueriesFactorizedModeIncludesEffectsAndEligibilityControls(t *testing.T) {
	queries, err := loadQueries("factorized_mode", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "factorized_mode", err)
	}
	if got, want := len(queries), 15; got != want {
		t.Fatalf("factorized query count = %d, want %d", got, want)
	}
	seen := make(map[string]struct{}, len(queries))
	for _, query := range queries {
		if _, duplicate := seen[query.Label]; duplicate {
			t.Errorf("duplicate factorized query label %q", query.Label)
		}
		seen[query.Label] = struct{}{}
	}
	for _, label := range []string{
		"factorized/control/default",
		"factorized/effect/left",
		"factorized/effect/right",
		"factorized/effect/both",
		"factorized/version/v1-hash-both-effect",
		"factorized/version/v8-hash-both-effect",
		"factorized/version/v4-left-accepted-no-visible-effect",
		"factorized/version/v5-left-effect",
		"factorized/version/v4-join-key-only-both-accepted-no-visible-effect",
		"factorized/version/v4-non-equality-both-accepted-no-visible-effect",
		"factorized/version/v8-unsupported/join-key-only-left",
		"factorized/version/v8-unsupported/join-key-only-right",
		"factorized/version/v8-unsupported/join-key-only-both",
		"factorized/version/v8-unsupported/left-outer-both",
		"factorized/version/v8-unsupported/non-equality-both",
	} {
		if _, ok := seen[label]; !ok {
			t.Errorf("factorized case is missing %q", label)
		}
	}

	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "factorized_mode_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
		ExpectedQueryErrors []struct {
			Label    string `json:"label"`
			Contains string `json:"contains"`
		} `json:"expected_query_errors"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Version, "v0alpha1"; got != want {
		t.Errorf("factorized expectation version = %q, want %q", got, want)
	}
	if got, want := len(manifest.Queries), 10; got != want {
		t.Errorf("factorized positive expectations = %d, want %d", got, want)
	}
	if got, want := len(manifest.ExpectedQueryErrors), 5; got != want {
		t.Errorf("factorized error expectations = %d, want %d", got, want)
	}
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("factorized expectation label %q is absent from the built-in case", expectation.Label)
		}
		if len(expectation.Patterns) == 0 {
			t.Errorf("factorized expectation label %q has no operator patterns", expectation.Label)
		}
	}
	for _, expectation := range manifest.ExpectedQueryErrors {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("factorized expected-error label %q is absent from the built-in case", expectation.Label)
		}
		if expectation.Contains == "" {
			t.Errorf("factorized expected-error label %q has no matching text", expectation.Label)
		}
	}

	ddls, err := loadDDLs("factorized_mode", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "factorized_mode", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, table := range []string{"Albums", "Songs"} {
		if !strings.Contains(joined, "CREATE TABLE "+table) {
			t.Errorf("factorized schema is missing table %q", table)
		}
	}
}

func TestLoadQueriesGQLSurfaceIncludesBroadAcceptanceAndCapabilityControls(t *testing.T) {
	queries, err := loadQueries("gql_surface", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "gql_surface", err)
	}
	if got, want := len(queries), 125; got != want {
		t.Fatalf("GQL surface query count = %d, want %d", got, want)
	}
	seen := make(map[string]struct{}, len(queries))
	for _, query := range queries {
		if !strings.HasPrefix(query.Label, "gql-surface/") {
			t.Errorf("GQL surface label %q has wrong prefix", query.Label)
		}
		if _, duplicate := seen[query.Label]; duplicate {
			t.Errorf("duplicate GQL surface query label %q", query.Label)
		}
		seen[query.Label] = struct{}{}
	}
	for _, label := range []string{
		"gql-surface/linear/for-with-offset",
		"gql-surface/call/optional-correlated",
		"gql-surface/call/uncorrelated-aggregate",
		"gql-surface/subquery/correlated-exists",
		"gql-surface/subquery/correlated-value",
		"gql-surface/subquery/correlated-array",
		"gql-surface/subquery/correlated-in",
		"gql-surface/subquery/correlated-not-in",
		"gql-surface/subquery/exists-match-body",
		"gql-surface/subquery/exists-pattern-body",
		"gql-surface/subquery/exists-pattern-filter",
		"gql-surface/bridge/dml-update-recursive-in",
		"gql-surface/set-operation/intersect-distinct-nontrivial",
		"gql-surface/set-operation/intersect-all-nontrivial",
		"gql-surface/set-operation/reordered-intersect-distinct",
		"gql-surface/set-operation/three-arm-union-all",
		"gql-surface/set-operation/full-union-all-omni",
		"gql-surface/search/any-cheapest-bounded",
		"gql-surface/search/any-cheapest-property-cost",
		"gql-surface/mode/acyclic-quantified",
		"gql-surface/mode/acyclic-fixed",
		"gql-surface/path/functions",
		"gql-surface/path/recursive-materialization",
		"gql-surface/path/construct",
		"gql-surface/path/concatenate",
		"gql-surface/path/quantified-subpath-where",
		"gql-surface/path/zero-hop",
		"gql-surface/path/two-quantified-segments",
		"gql-surface/element/functions",
		"gql-surface/element/predicates",
		"gql-surface/pattern/label-or",
		"gql-surface/pattern/label-and",
		"gql-surface/pattern/label-and-single-control",
		"gql-surface/direction/any",
		"gql-surface/graph-table/tablesample-bernoulli",
		"gql-surface/linear/tablesample-bernoulli",
		"gql-surface/with/group-order-limit",
		"gql-surface/with/group-filter",
		"gql-surface/with/implicit-grouping",
		"gql-surface/pagination/primitive-order-by-collate-limit",
		"gql-surface/aggregate/horizontal-count-distinct",
		"gql-surface/factorized/quantified-left",
		"gql-surface/match/optional",
		"gql-surface/linear/next-two-stage-traversal",
		gqlSurfaceISFirstReturnLabel,
		gqlSurfaceISFirstFilterLabel,
		gqlSurfaceISFirstEdgeOneHopLabel,
		gqlSurfaceISFirstQuantifiedLabel,
		gqlSurfaceISFirstBeforeNextLabel,
		gqlSurfaceISFirstBeforeNextOrderedLabel,
		"gql-surface/aggregate/horizontal-string-agg-ordered",
		"gql-surface/aggregate/horizontal-string-agg-unordered-control",
		"gql-surface/unsupported/horizontal-string-agg-return",
		"gql-surface/aggregate/horizontal-array-concat-agg-ordered",
		"gql-surface/unsupported/vertical-array-concat-agg-group-variable",
		"gql-surface/unsupported/gql-row-number",
		"gql-surface/unsupported/sql-is-first-control",
		"gql-surface/unsupported/pagerank-requires-export",
		"gql-surface/unsupported/gql-native-qualify",
		"gql-surface/unsupported/sql-qualify-control",
		"gql-surface/unsupported/any-path-count",
		"gql-surface/unsupported/simple-path",
		"gql-surface/unsupported/unbounded-quantifier",
		"gql-surface/unsupported/tablesample-repeatable",
		"gql-surface/unsupported/malformed-label-or",
		"gql-surface/unsupported/all-cheapest",
		"gql-surface/unsupported/pagerank-call-per-requires-export",
		"gql-surface/unsupported/final-graph-element",
		"gql-surface/unsupported/mixed-set-operation-kinds",
		"gql-surface/unsupported/export-data-graph-shape",
	} {
		if _, ok := seen[label]; !ok {
			t.Errorf("GQL surface case is missing %q", label)
		}
	}

	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "gql_surface_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
		ExpectedQueryErrors []struct {
			Label    string `json:"label"`
			Contains string `json:"contains"`
		} `json:"expected_query_errors"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Version, "v0alpha1"; got != want {
		t.Errorf("GQL surface expectation version = %q, want %q", got, want)
	}
	if got, want := len(manifest.Queries), 100; got != want {
		t.Errorf("GQL surface positive expectations = %d, want %d", got, want)
	}
	if got, want := len(manifest.ExpectedQueryErrors), 25; got != want {
		t.Errorf("GQL surface error expectations = %d, want %d", got, want)
	}
	patternCount := 0
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("GQL surface expectation label %q is absent from the built-in case", expectation.Label)
		}
		if len(expectation.Patterns) == 0 {
			t.Errorf("GQL surface expectation label %q has no operator patterns", expectation.Label)
		}
		patternCount += len(expectation.Patterns)
	}
	if got, want := patternCount, 208; got != want {
		t.Errorf("GQL surface operator patterns = %d, want %d", got, want)
	}
	for _, expectation := range manifest.ExpectedQueryErrors {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("GQL surface expected-error label %q is absent from the built-in case", expectation.Label)
		}
		if expectation.Contains == "" {
			t.Errorf("GQL surface expected-error label %q has no matching text", expectation.Label)
		}
	}

	ddls, err := loadDDLs("gql_surface", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "gql_surface", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{"CREATE TABLE Singers", "CREATE TABLE Collaborations", "PROPERTY GRAPH MusicGraph", "PROPERTY GRAPH LabelGraph"} {
		if !strings.Contains(joined, want) {
			t.Errorf("GQL surface schema is missing %q", want)
		}
	}
}

func TestLoadQueriesGQLHintSurfaceIncludesVersionBoundaries(t *testing.T) {
	queries, err := loadQueries("gql_hint_surface", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "gql_hint_surface", err)
	}
	if got, want := len(queries), 50; got != want {
		t.Fatalf("GQL hint surface query count = %d, want %d", got, want)
	}
	seen := make(map[string]struct{}, len(queries))
	for _, query := range queries {
		if !strings.HasPrefix(query.Label, "gql-hint/") {
			t.Errorf("GQL hint surface label %q has wrong prefix", query.Label)
		}
		if _, duplicate := seen[query.Label]; duplicate {
			t.Errorf("duplicate GQL hint surface query label %q", query.Label)
		}
		seen[query.Label] = struct{}{}
	}
	for _, label := range []string{
		"gql-hint/factorized/quantified-both",
		"gql-hint/factorized/nonlinear-both",
		"gql-hint/set-operation/direct-hint-unsupported",
		"gql-hint/set-operation/statement-hash",
		"gql-hint/set-operation/statement-merge",
		"gql-hint/set-operation/statement-push",
		"gql-hint/set-operation/arm-mixed-hash-apply",
		"gql-hint/subquery/exists-hash-multi-pass-accepted-no-effect",
		"gql-hint/subquery/exists-hash-one-pass-accepted-no-effect",
		"gql-hint/element/node-force-base-table",
		"gql-hint/element/node-force-secondary-index-seekable-key-size-1",
		"gql-hint/element/node-force-secondary-index-scan-columnar",
		"gql-hint/element/node-force-secondary-index-scan-no-columnar",
		"gql-hint/element/node-index-strategy-force-index-union",
		"gql-hint/element/node-force-secondary-index-groupby-scan-true",
		"gql-hint/element/edge-scan-batch",
		"gql-hint/element/traversal-hash-build-right",
		"gql-hint/element/traversal-apply-batch-false",
		"gql-hint/element/node-to-edge-hash-build-right",
		"gql-hint/element/subpath-to-edge-apply-batch-false",
		"gql-hint/element/between-path-patterns-hash",
		"gql-hint/runtime-extension/subpath-to-node-hash",
		"gql-hint/runtime-extension/subpath-to-subpath-hash",
		"gql-hint/match/second-merge",
		"gql-hint/match/second-push",
		"gql-hint/graph-table/node-scan-batch",
		"gql-hint/version/traversal-push",
	} {
		if _, ok := seen[label]; !ok {
			t.Errorf("GQL hint surface case is missing %q", label)
		}
	}

	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "gql_hint_surface_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
		ExpectedQueryErrors []struct {
			Label    string `json:"label"`
			Contains string `json:"contains"`
		} `json:"expected_query_errors"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Version, "v0alpha1"; got != want {
		t.Errorf("GQL hint surface expectation version = %q, want %q", got, want)
	}
	if got, want := len(manifest.Queries), 47; got != want {
		t.Errorf("GQL hint surface positive expectations = %d, want %d", got, want)
	}
	if got, want := len(manifest.ExpectedQueryErrors), 3; got != want {
		t.Errorf("GQL hint surface error expectations = %d, want %d", got, want)
	}
	patternCount := 0
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("GQL hint expectation label %q is absent from the built-in case", expectation.Label)
		}
		if len(expectation.Patterns) == 0 {
			t.Errorf("GQL hint expectation label %q has no operator patterns", expectation.Label)
		}
		patternCount += len(expectation.Patterns)
	}
	if got, want := patternCount, 93; got != want {
		t.Errorf("GQL hint operator patterns = %d, want %d", got, want)
	}
	for _, expectation := range manifest.ExpectedQueryErrors {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("GQL hint expected-error label %q is absent from the built-in case", expectation.Label)
		}
		if expectation.Contains == "" {
			t.Errorf("GQL hint expected-error label %q has no matching text", expectation.Label)
		}
	}
}

func TestLoadQueriesGoogleSQLSurfaceIncludesPlansAndRuntimeCapabilityErrors(t *testing.T) {
	queries, err := loadQueries("google_sql_surface", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "google_sql_surface", err)
	}
	if got, want := len(queries), 57; got != want {
		t.Fatalf("GoogleSQL surface query count = %d, want %d", got, want)
	}
	seen := make(map[string]struct{}, len(queries))
	for _, query := range queries {
		if !strings.HasPrefix(query.Label, "google-sql-surface/accepted/") &&
			!strings.HasPrefix(query.Label, "google-sql-surface/unsupported/") {
			t.Errorf("GoogleSQL surface label %q has wrong prefix", query.Label)
		}
		if _, duplicate := seen[query.Label]; duplicate {
			t.Errorf("duplicate GoogleSQL surface query label %q", query.Label)
		}
		seen[query.Label] = struct{}{}
	}

	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "google_sql_surface_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
		ExpectedQueryErrors []struct {
			Label    string `json:"label"`
			Contains string `json:"contains"`
		} `json:"expected_query_errors"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Version, "v0alpha1"; got != want {
		t.Errorf("GoogleSQL surface expectation version = %q, want %q", got, want)
	}
	if got, want := len(manifest.Queries), 35; got != want {
		t.Errorf("GoogleSQL surface positive expectations = %d, want %d", got, want)
	}
	if got, want := len(manifest.ExpectedQueryErrors), 22; got != want {
		t.Errorf("GoogleSQL surface error expectations = %d, want %d", got, want)
	}
	patternCount := 0
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("GoogleSQL surface expectation label %q is absent from the built-in case", expectation.Label)
		}
		if len(expectation.Patterns) == 0 {
			t.Errorf("GoogleSQL surface expectation label %q has no operator patterns", expectation.Label)
		}
		patternCount += len(expectation.Patterns)
	}
	if got, want := patternCount, 74; got != want {
		t.Errorf("GoogleSQL surface operator patterns = %d, want %d", got, want)
	}
	for _, expectation := range manifest.ExpectedQueryErrors {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("GoogleSQL surface expected-error label %q is absent from the built-in case", expectation.Label)
		}
		if expectation.Contains == "" {
			t.Errorf("GoogleSQL surface expected-error label %q has no matching text", expectation.Label)
		}
	}
}

func TestLoadQueriesGoogleSQLProtoSurface(t *testing.T) {
	queries, err := loadQueries("google_sql_proto_surface", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "google_sql_proto_surface", err)
	}
	if got, want := len(queries), 18; got != want {
		t.Fatalf("GoogleSQL proto surface query count = %d, want %d", got, want)
	}
	seen := make(map[string]bool, len(queries))
	accepted, unsupported := 0, 0
	for _, query := range queries {
		switch {
		case strings.HasPrefix(query.Label, "google-sql-proto-surface/accepted/"):
			accepted++
		case strings.HasPrefix(query.Label, "google-sql-proto-surface/unsupported/"):
			unsupported++
		default:
			t.Errorf("GoogleSQL proto surface label %q has wrong prefix", query.Label)
		}
		if seen[query.Label] {
			t.Errorf("duplicate GoogleSQL proto surface label %q", query.Label)
		}
		seen[query.Label] = true
	}
	if got, want := len(seen), len(queries); got != want {
		t.Fatalf("unique GoogleSQL proto surface labels = %d, want %d", got, want)
	}
	if accepted != 11 || unsupported != 7 {
		t.Fatalf("GoogleSQL proto surface classes = accepted %d, unsupported %d; want 11/7", accepted, unsupported)
	}

	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "google_sql_proto_surface_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
		ExpectedQueryErrors []struct {
			Label    string `json:"label"`
			Contains string `json:"contains"`
		} `json:"expected_query_errors"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Version, "v0alpha1"; got != want {
		t.Errorf("GoogleSQL proto expectation version = %q, want %q", got, want)
	}
	if got, want := len(manifest.Queries), 11; got != want {
		t.Errorf("GoogleSQL proto positive expectations = %d, want %d", got, want)
	}
	if got, want := len(manifest.ExpectedQueryErrors), 7; got != want {
		t.Errorf("GoogleSQL proto error expectations = %d, want %d", got, want)
	}
	manifestLabels := make(map[string]struct{}, len(queries))
	patternCount := 0
	for _, expectation := range manifest.Queries {
		if !seen[expectation.Label] {
			t.Errorf("GoogleSQL proto expectation label %q is absent from the built-in case", expectation.Label)
		}
		if _, duplicate := manifestLabels[expectation.Label]; duplicate {
			t.Errorf("GoogleSQL proto expectation label %q is duplicated", expectation.Label)
		}
		manifestLabels[expectation.Label] = struct{}{}
		if len(expectation.Patterns) == 0 {
			t.Errorf("GoogleSQL proto expectation label %q has no operator patterns", expectation.Label)
		}
		patternCount += len(expectation.Patterns)
	}
	if got, want := patternCount, 28; got != want {
		t.Errorf("GoogleSQL proto operator patterns = %d, want %d", got, want)
	}
	for _, expectation := range manifest.ExpectedQueryErrors {
		if !seen[expectation.Label] {
			t.Errorf("GoogleSQL proto expected-error label %q is absent from the built-in case", expectation.Label)
		}
		if expectation.Contains == "" {
			t.Errorf("GoogleSQL proto expected-error label %q has no matching text", expectation.Label)
		}
		if _, duplicate := manifestLabels[expectation.Label]; duplicate {
			t.Errorf("GoogleSQL proto expected-error label %q is duplicated", expectation.Label)
		}
		manifestLabels[expectation.Label] = struct{}{}
	}
	if got, want := len(manifestLabels), len(queries); got != want {
		t.Errorf("GoogleSQL proto expectation coverage = %d labels, want %d", got, want)
	}
}

func TestLoadDDLsGoogleSQLProtoSurface(t *testing.T) {
	ddls, err := loadDDLs("google_sql_proto_surface", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "google_sql_proto_surface", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{"CREATE PROTO BUNDLE", "examples.shipping.Order", "examples.user.User", "examples.user.User.UserType", "CREATE TABLE Orders", "CREATE TABLE ProtoUsers"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("loadDDLs(\"google_sql_proto_surface\") missing %q in:\n%s", want, joined)
		}
	}
}

func TestLoadQueriesConditionBoundaries(t *testing.T) {
	queries, err := loadQueries("condition_boundaries", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "condition_boundaries", err)
	}
	if got, want := len(queries), 38; got != want {
		t.Fatalf("condition boundary query count = %d, want %d", got, want)
	}
	seen := make(map[string]struct{}, len(queries))
	parameterized := 0
	for _, query := range queries {
		if !strings.HasPrefix(query.Label, "condition-boundary/scan/") &&
			!strings.HasPrefix(query.Label, "condition-boundary/timestamp-key/") &&
			!strings.HasPrefix(query.Label, "condition-boundary/join/") {
			t.Errorf("condition boundary label %q has wrong prefix", query.Label)
		}
		if _, duplicate := seen[query.Label]; duplicate {
			t.Errorf("duplicate condition boundary label %q", query.Label)
		}
		seen[query.Label] = struct{}{}
		if len(query.Params) != 0 {
			parameterized++
		}
	}
	if got, want := parameterized, 2; got != want {
		t.Errorf("parameterized condition boundary cases = %d, want %d", got, want)
	}
	ddls, err := loadDDLs("condition_boundaries", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "condition_boundaries", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{"CREATE TABLE Songs", "CREATE INDEX SongsBySongName", "CREATE TABLE Albums", "CREATE TABLE CommitTimestampKeys"} {
		if !strings.Contains(joined, want) {
			t.Errorf("condition boundary DDL is missing %q", want)
		}
	}

	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "condition_boundary_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
		ExpectedQueryErrors []json.RawMessage `json:"expected_query_errors"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Version, "v0alpha1"; got != want {
		t.Errorf("condition boundary expectation version = %q, want %q", got, want)
	}
	if got, want := len(manifest.Queries), len(queries); got != want {
		t.Errorf("condition boundary positive expectations = %d, want %d", got, want)
	}
	if got := len(manifest.ExpectedQueryErrors); got != 0 {
		t.Errorf("condition boundary error expectations = %d, want 0", got)
	}
	manifestLabels := make(map[string]struct{}, len(manifest.Queries))
	patternCount := 0
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("condition boundary expectation label %q is absent from the built-in case", expectation.Label)
		}
		if _, duplicate := manifestLabels[expectation.Label]; duplicate {
			t.Errorf("condition boundary expectation label %q is duplicated", expectation.Label)
		}
		manifestLabels[expectation.Label] = struct{}{}
		if len(expectation.Patterns) == 0 {
			t.Errorf("condition boundary expectation label %q has no operator patterns", expectation.Label)
		}
		patternCount += len(expectation.Patterns)
	}
	if got, want := patternCount, 62; got != want {
		t.Errorf("condition boundary operator patterns = %d, want %d", got, want)
	}
}

func TestLoadQueriesAggregateFunctions(t *testing.T) {
	queries, err := loadQueries("aggregate_functions", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "aggregate_functions", err)
	}
	if got, want := len(queries), 31; got != want {
		t.Fatalf("aggregate function query count = %d, want %d", got, want)
	}
	seen := make(map[string]struct{}, len(queries))
	for _, query := range queries {
		if !strings.HasPrefix(query.Label, "aggregate-function/") {
			t.Errorf("aggregate function label %q has wrong prefix", query.Label)
		}
		if _, duplicate := seen[query.Label]; duplicate {
			t.Errorf("duplicate aggregate function label %q", query.Label)
		}
		seen[query.Label] = struct{}{}
	}
	for _, label := range []string{
		"aggregate-function/general/any-value",
		"aggregate-function/general/array-agg",
		"aggregate-function/general/array-concat-agg",
		"aggregate-function/general/avg",
		"aggregate-function/general/bit-and",
		"aggregate-function/general/bit-or",
		"aggregate-function/general/bit-xor",
		"aggregate-function/general/count",
		"aggregate-function/general/countif",
		"aggregate-function/general/logical-and",
		"aggregate-function/general/logical-or",
		"aggregate-function/general/max",
		"aggregate-function/general/min",
		"aggregate-function/statistical/stddev",
		"aggregate-function/statistical/stddev-samp",
		"aggregate-function/general/string-agg",
		"aggregate-function/general/sum",
		"aggregate-function/statistical/var-samp",
		"aggregate-function/statistical/variance",
	} {
		if _, ok := seen[label]; !ok {
			t.Errorf("documented aggregate function case %q is missing", label)
		}
	}
	ddls, err := loadDDLs("aggregate_functions", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "aggregate_functions", err)
	}
	if joined := strings.Join(ddls, "\n"); !strings.Contains(joined, "CREATE TABLE Songs") {
		t.Error("aggregate function DDL is missing Songs")
	}

	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "aggregate_function_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
		ExpectedQueryErrors []struct {
			Label    string `json:"label"`
			Contains string `json:"contains"`
		} `json:"expected_query_errors"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Version, "v0alpha1"; got != want {
		t.Errorf("aggregate function expectation version = %q, want %q", got, want)
	}
	if got, want := len(manifest.Queries), 29; got != want {
		t.Errorf("aggregate function positive expectations = %d, want %d", got, want)
	}
	if got, want := len(manifest.ExpectedQueryErrors), 2; got != want {
		t.Errorf("aggregate function error expectations = %d, want %d", got, want)
	}
	manifestLabels := make(map[string]struct{}, len(queries))
	patternCount := 0
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("aggregate function expectation label %q is absent from the built-in case", expectation.Label)
		}
		if _, duplicate := manifestLabels[expectation.Label]; duplicate {
			t.Errorf("aggregate function expectation label %q is duplicated", expectation.Label)
		}
		manifestLabels[expectation.Label] = struct{}{}
		if len(expectation.Patterns) == 0 {
			t.Errorf("aggregate function expectation label %q has no operator patterns", expectation.Label)
		}
		patternCount += len(expectation.Patterns)
	}
	if got, want := patternCount, 62; got != want {
		t.Errorf("aggregate function operator patterns = %d, want %d", got, want)
	}
	for _, expectation := range manifest.ExpectedQueryErrors {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("aggregate function error label %q is absent from the built-in case", expectation.Label)
		}
		if _, duplicate := manifestLabels[expectation.Label]; duplicate {
			t.Errorf("aggregate function expectation label %q is duplicated", expectation.Label)
		}
		manifestLabels[expectation.Label] = struct{}{}
		if expectation.Contains == "" {
			t.Errorf("aggregate function error label %q has empty error text", expectation.Label)
		}
	}
	if got, want := len(manifestLabels), len(queries); got != want {
		t.Errorf("aggregate function expectation coverage = %d labels, want %d", got, want)
	}
}

func TestLoadFileDescriptorSet(t *testing.T) {
	const path = "../../testdata/protos/order_descriptors.pb"
	set, err := loadFileDescriptorSet([]string{path, path})
	if err != nil {
		t.Fatalf("loadFileDescriptorSet() error = %v", err)
	}
	if set == nil {
		t.Fatal("loadFileDescriptorSet() = nil, want descriptor set")
	}
	if got, want := len(set.File), 1; got != want {
		t.Fatalf("descriptor file count = %d, want %d", got, want)
	}
	if got, want := set.File[0].GetName(), "order_protos.proto"; got != want {
		t.Fatalf("descriptor file name = %q, want %q", got, want)
	}

	set, err = loadFileDescriptorSet([]string{
		path,
		"../../testdata/protos/complex/complex_descriptors.pb",
	})
	if err != nil {
		t.Fatalf("loadFileDescriptorSet(two fixtures) error = %v", err)
	}
	gotNames := make([]string, 0, len(set.File))
	for _, file := range set.File {
		gotNames = append(gotNames, file.GetName())
	}
	wantNames := []string{
		"order_protos.proto",
		"testdata/protos/complex/order_ext.proto",
		"testdata/protos/complex/user.proto",
	}
	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("merged descriptor names = %v, want %v", gotNames, wantNames)
	}
	if _, err := protodesc.NewFiles(set); err != nil {
		t.Fatalf("protodesc.NewFiles(merged descriptor set) error = %v", err)
	}

	set, err = loadFileDescriptorSet(nil)
	if err != nil {
		t.Fatalf("loadFileDescriptorSet(nil) error = %v", err)
	}
	if set != nil {
		t.Fatalf("loadFileDescriptorSet(nil) = %v, want nil", set)
	}
}

func TestLoadFileDescriptorSetRejectsConflictingFiles(t *testing.T) {
	writeSet := func(name, packageName string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), name+".pb")
		raw, err := proto.Marshal(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("shared.proto"),
			Package: proto.String(packageName),
		}}})
		if err != nil {
			t.Fatalf("proto.Marshal() error = %v", err)
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}
		return path
	}

	_, err := loadFileDescriptorSet([]string{
		writeSet("first", "example.first"),
		writeSet("second", "example.second"),
	})
	if err == nil || !strings.Contains(err.Error(), "conflicting proto descriptor") {
		t.Fatalf("loadFileDescriptorSet() error = %v, want conflicting descriptor error", err)
	}
}

func TestLoadDDLsFullTextSearchUsesDedicatedSchema(t *testing.T) {
	ddls, err := loadDDLs("full_text_search", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "full_text_search", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{
		"CREATE TABLE SearchAlbums",
		"TOKENLIST AS (TOKENIZE_FULLTEXT",
		"TOKENLIST AS (TOKENIZE_SUBSTRING",
		"CREATE SEARCH INDEX SearchAlbumsTitleIndex",
		"CREATE SEARCH INDEX SearchAlbumsTitleSubstringIndex",
		"CREATE SEARCH INDEX SearchAlbumsTitleRatingIndex",
		"CREATE SEARCH INDEX SearchAlbumsMixedIndex",
		"CREATE SEARCH INDEX SearchAlbumsFacetIndex",
		"CREATE SEARCH INDEX SearchAlbumsPhoneticIndex",
		"ArtistNameSoundex STRING(MAX) AS (LOWER(SOUNDEX(ArtistName)))",
		"CREATE SEARCH INDEX SearchSingersByBio",
		"CREATE PROPERTY GRAPH SearchMusicGraph",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("loadDDLs(\"full_text_search\") missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "CREATE PROPERTY GRAPH MusicGraph") || strings.Contains(joined, "CREATE TABLE Singers") {
		t.Fatalf("full_text_search schema unexpectedly includes docs schema objects:\n%s", joined)
	}
}

func TestLoadDDLsJSONSearchUsesDedicatedSchema(t *testing.T) {
	ddls, err := loadDDLs("json_search", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "json_search", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{
		"CREATE TABLE JSONSearchDocuments",
		"TOKENIZE_JSON(Metadata)",
		"CREATE SEARCH INDEX JSONSearchDocumentsByMetadata",
		"CREATE SEARCH INDEX JSONSearchDocumentsByTitleMetadata",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("loadDDLs(\"json_search\") missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "MusicGraph") || strings.Contains(joined, "CREATE TABLE Singers") {
		t.Fatalf("json_search schema unexpectedly includes docs schema objects:\n%s", joined)
	}
}

func TestLoadDDLsVectorSearchUsesDedicatedSchema(t *testing.T) {
	ddls, err := loadDDLs("vector_search", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "vector_search", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{
		"CREATE TABLE VectorDocuments",
		"ARRAY<FLOAT32>(vector_length=>3)",
		"CREATE VECTOR INDEX VectorDocumentsByEmbedding",
		"ON VectorDocuments(Embedding, TenantId)",
		"CREATE VECTOR INDEX TechVectorDocumentsByEmbedding",
		"WHERE TechOnly IS NOT NULL",
		"CREATE VECTOR INDEX VectorDocumentsByDotProduct",
		"OPTIONS (distance_type = 'DOT_PRODUCT')",
		"CREATE VECTOR INDEX VectorDocumentsByEuclidean",
		"OPTIONS (distance_type = 'EUCLIDEAN')",
		"CREATE TABLE VectorRelated",
		"CREATE PROPERTY GRAPH VectorGraph",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("loadDDLs(\"vector_search\") missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "CREATE PROPERTY GRAPH MusicGraph") || strings.Contains(joined, "CREATE TABLE Singers") {
		t.Fatalf("vector_search schema unexpectedly includes docs schema objects:\n%s", joined)
	}
}

func TestLoadQueriesOptimizerGaps(t *testing.T) {
	queries, err := loadQueries("optimizer_gaps", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "optimizer_gaps", err)
	}
	seen := map[string]queryCase{}
	for _, query := range queries {
		seen[query.Label] = query
	}
	for _, label := range []string{
		"optimizer-gaps/v8/with-large-in-join-order-limit",
		"optimizer-gaps/v8/use-unenforced-foreign-key-true",
		"optimizer-gaps/v7/unhinted-index-union-candidate",
		"optimizer-gaps/v6/dml-insert-select-filter",
		"optimizer-gaps/v6/full-outer-join-predicate-limit",
		"optimizer-gaps/v3/sorted-limit-cross-apply",
		"optimizer-gaps/v3/push-computation-through-join",
		"optimizer-gaps/v2/regexp-contains-prefix",
		"optimizer-gaps/v2/regexp-contains-prefix-forced-index",
		"optimizer-gaps/v2/like-prefix-forced-index",
	} {
		if _, ok := seen[label]; !ok {
			t.Fatalf("loadQueries(\"optimizer_gaps\") missing %s", label)
		}
	}
	if got, want := seen["optimizer-gaps/v6/dml-insert-select-filter"].effectivePlanMode(), planModeReadWrite; got != want {
		t.Fatalf("DML optimizer gap plan mode = %q, want %q", got, want)
	}
	if !strings.Contains(seen["optimizer-gaps/v8/with-large-in-join-order-limit"].SQL, "WITH CandidateSingers") {
		t.Fatalf("v8 CTE probe missing WITH clause: %s", seen["optimizer-gaps/v8/with-large-in-join-order-limit"].SQL)
	}
	if !strings.Contains(seen["optimizer-gaps/v8/use-unenforced-foreign-key-true"].SQL, "USE_UNENFORCED_FOREIGN_KEY=TRUE") {
		t.Fatalf("FK true probe missing hint: %s", seen["optimizer-gaps/v8/use-unenforced-foreign-key-true"].SQL)
	}
}

func TestLoadQueriesOptimizerUnhintedCandidates(t *testing.T) {
	queries, err := loadQueries("optimizer_unhinted_candidates", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "optimizer_unhinted_candidates", err)
	}
	if len(queries) == 0 {
		t.Fatal("loadQueries(\"optimizer_unhinted_candidates\") returned no queries")
	}
	seen := map[string]queryCase{}
	for _, query := range queries {
		seen[query.Label] = query
		if strings.Contains(query.SQL, "@{") {
			t.Fatalf("%s still contains a hint: %s", query.Label, query.SQL)
		}
	}
	for _, label := range []string{
		"optimizer-unhinted-candidates/execution-plans/index-with-back-join",
		"optimizer-unhinted-candidates/binary/hash-join",
		"optimizer-unhinted-candidates/binary/hash-join-full-outer",
		"optimizer-unhinted-candidates/unary/minor-sort-stream-aggregate",
		"optimizer-unhinted-candidates/optimizer-gaps/v2/regexp-contains-prefix-forced-index",
		"optimizer-unhinted-candidates/optimizer-gaps/v6/dml-insert-select-filter",
	} {
		if _, ok := seen[label]; !ok {
			t.Fatalf("loadQueries(\"optimizer_unhinted_candidates\") missing %s", label)
		}
	}
	if got := seen["optimizer-unhinted-candidates/unary/minor-sort-stream-aggregate"].SQL; !strings.Contains(got, "GROUP BY") || strings.Contains(got, "GROUP@") {
		t.Fatalf("group hint was not stripped cleanly: %s", got)
	}
	if got, want := seen["optimizer-unhinted-candidates/optimizer-gaps/v6/dml-insert-select-filter"].effectivePlanMode(), planModeReadWrite; got != want {
		t.Fatalf("DML unhinted optimizer candidate plan mode = %q, want %q", got, want)
	}
}

func TestStripGoogleSQLHints(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{
			name: "statement",
			sql:  "@{OPTIMIZER_VERSION=5} SELECT 1",
			want: "SELECT 1",
		},
		{
			name: "table_join_group_and_function",
			sql:  `SELECT SHA512(SingerInfo) @{DISABLE_INLINE=TRUE} FROM Singers@{FORCE_INDEX=_BASE_TABLE} JOIN@{JOIN_METHOD=HASH_JOIN} Albums USING(SingerId) GROUP@{GROUP_METHOD=STREAM_GROUP} BY SingerId`,
			want: `SELECT SHA512(SingerInfo) FROM Singers JOIN Albums USING(SingerId) GROUP BY SingerId`,
		},
		{
			name: "string_literal",
			sql:  `SELECT "@{not_a_hint}" AS s, '@{also_not_a_hint}' AS t`,
			want: `SELECT "@{not_a_hint}" AS s, '@{also_not_a_hint}' AS t`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripGoogleSQLHints(tt.sql); got != tt.want {
				t.Fatalf("stripGoogleSQLHints() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGroupOptimizerVersionShapes(t *testing.T) {
	shapes := []optimizerVersionShape{
		{version: 1, shape: "A"},
		{version: 2, shape: "A"},
		{version: 3, shape: "B"},
		{version: 4, shape: "B"},
		{version: 5, shape: "A"},
	}
	got := groupOptimizerVersionShapes(shapes)
	want := []optimizerVersionShapeGroup{
		{label: "v1-v2", shape: "A"},
		{label: "v3-v4", shape: "B"},
		{label: "v5", shape: "A"},
	}
	if len(got) != len(want) {
		t.Fatalf("len(groupOptimizerVersionShapes()) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("group %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestLoadQueriesFullTextSearch(t *testing.T) {
	queries, err := loadQueries("full_text_search", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "full_text_search", err)
	}
	seen := map[string]string{}
	for _, query := range queries {
		seen[query.Label] = query.SQL
	}
	for _, label := range []string{
		"full-text-search/search",
		"full-text-search/force-index",
		"full-text-search/snippet",
		"full-text-search/score-order",
		"full-text-search/enhanced-query",
		"full-text-search/enhanced-query-control",
		"full-text-search/enhanced-query-required-hint",
		"full-text-search/enhanced-query-timeout-hint",
		"full-text-search/phonetic-composition",
		"full-text-search/phonetic-search-only-control",
		"full-text-search/substring",
		"full-text-search/multi-column-conjunction",
		"full-text-search/multi-column-disjunction",
		"full-text-search/multi-column-negation",
		"full-text-search/tokenlist-concat",
		"full-text-search/partitioned-ordered-index",
		"full-text-search/numeric-array-any",
		"full-text-search/numeric-array-all",
		"full-text-search/mixed-accelerated",
		"full-text-search/mixed-stored-filter",
		"full-text-search/mixed-back-join",
		"full-text-search/gql-search-traversal",
		"full-text-search/facet-single-count",
		"full-text-search/facet-search-only-control",
		"full-text-search/facet-multiple",
		"full-text-search/facet-result-page-control",
	} {
		if seen[label] == "" {
			t.Fatalf("loadQueries(\"full_text_search\") missing %s", label)
		}
	}
	if !strings.Contains(seen["full-text-search/search"], "SEARCH(") {
		t.Fatalf("search query missing SEARCH(): %s", seen["full-text-search/search"])
	}
	if !strings.Contains(seen["full-text-search/score-order"], "SCORE(") {
		t.Fatalf("score query missing SCORE(): %s", seen["full-text-search/score-order"])
	}
	if !strings.Contains(seen["full-text-search/enhanced-query"], "enhance_query => TRUE") {
		t.Fatalf("enhanced-query probe missing named argument: %s", seen["full-text-search/enhanced-query"])
	}
	if !strings.Contains(seen["full-text-search/enhanced-query-required-hint"], "require_enhance_query=true") ||
		!strings.Contains(seen["full-text-search/enhanced-query-timeout-hint"], "enhance_query_timeout_ms=200") {
		t.Fatal("enhanced-query statement hint probes missing expected hint")
	}
	if !strings.Contains(seen["full-text-search/phonetic-composition"], `LOWER(SOUNDEX("stefan"))`) {
		t.Fatalf("phonetic composition probe missing SOUNDEX(): %s", seen["full-text-search/phonetic-composition"])
	}
	if !strings.Contains(seen["full-text-search/numeric-array-any"], "ARRAY_INCLUDES_ANY") {
		t.Fatalf("numeric array query missing ARRAY_INCLUDES_ANY(): %s", seen["full-text-search/numeric-array-any"])
	}
	if !strings.Contains(seen["full-text-search/gql-search-traversal"], "GRAPH SearchMusicGraph") {
		t.Fatalf("GQL search traversal probe missing graph query: %s", seen["full-text-search/gql-search-traversal"])
	}
}

func TestFacetSearchExpectationManifestMatchesCases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "facet_search_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
		ExpectedQueryErrors []json.RawMessage `json:"expected_query_errors"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "v0alpha1" || len(manifest.Queries) != 4 || len(manifest.ExpectedQueryErrors) != 0 {
		t.Fatalf("facet manifest summary = version %q, %d queries, %d errors", manifest.Version, len(manifest.Queries), len(manifest.ExpectedQueryErrors))
	}
	queries, err := loadQueries("full_text_search", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]struct{}, len(queries))
	for _, query := range queries {
		seen[query.Label] = struct{}{}
	}
	patterns := 0
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("facet manifest label %q absent from full-text selector", expectation.Label)
		}
		if len(expectation.Patterns) == 0 {
			t.Errorf("facet manifest label %q has no patterns", expectation.Label)
		}
		patterns += len(expectation.Patterns)
	}
	if patterns != 14 {
		t.Errorf("facet manifest pattern count = %d, want 14", patterns)
	}
}

func TestFullTextResidualExpectationManifestMatchesCases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "full_text_residual_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
		ExpectedQueryErrors []json.RawMessage `json:"expected_query_errors"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "v0alpha1" || len(manifest.Queries) != 6 || len(manifest.ExpectedQueryErrors) != 0 {
		t.Fatalf("full-text residual manifest summary = version %q, %d queries, %d errors", manifest.Version, len(manifest.Queries), len(manifest.ExpectedQueryErrors))
	}
	queries, err := loadQueries("full_text_search", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]struct{}, len(queries))
	for _, query := range queries {
		seen[query.Label] = struct{}{}
	}
	patterns := 0
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("full-text residual manifest label %q absent from full-text selector", expectation.Label)
		}
		if len(expectation.Patterns) == 0 {
			t.Errorf("full-text residual manifest label %q has no patterns", expectation.Label)
		}
		patterns += len(expectation.Patterns)
	}
	if patterns != 17 {
		t.Errorf("full-text residual manifest pattern count = %d, want 17", patterns)
	}
}

func TestLoadQueriesJSONSearch(t *testing.T) {
	queries, err := loadQueries("json_search", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "json_search", err)
	}
	seen := map[string]string{}
	for _, query := range queries {
		seen[query.Label] = query.SQL
	}
	for _, label := range []string{
		"json-search/containment-auto-index",
		"json-search/containment-force-index",
		"json-search/containment-base-table",
		"json-search/nested-array-containment",
		"json-search/key-existence",
		"json-search/array-path-existence",
		"json-search/conjunction",
		"json-search/disjunction-and-negation",
		"json-search/mixed-full-text-json",
		"json-search/stored-residual-filter",
		"json-search/non-covering-back-join",
	} {
		if seen[label] == "" {
			t.Fatalf("loadQueries(\"json_search\") missing %s", label)
		}
	}
	if !strings.Contains(seen["json-search/containment-force-index"], "JSON_CONTAINS(") {
		t.Fatalf("containment probe missing JSON_CONTAINS(): %s", seen["json-search/containment-force-index"])
	}
	if !strings.Contains(seen["json-search/mixed-full-text-json"], "SEARCH(") {
		t.Fatalf("mixed probe missing SEARCH(): %s", seen["json-search/mixed-full-text-json"])
	}
}

func TestLoadQueriesVectorSearch(t *testing.T) {
	queries, err := loadQueries("vector_search", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "vector_search", err)
	}
	seen := map[string]string{}
	for _, query := range queries {
		seen[query.Label] = query.SQL
	}
	for _, label := range []string{
		"vector-search/exact-knn-base-table",
		"vector-search/ann-auto-index",
		"vector-search/ann-extra-key-filter",
		"vector-search/ann-stored-filter",
		"vector-search/ann-back-join",
		"vector-search/ann-filtered-index",
		"vector-search/ann-dot-product",
		"vector-search/ann-euclidean-distance",
		"vector-search/ann-gql-next-traversal",
		"vector-search/hybrid-rrf",
		"vector-search/hybrid-rrf-exact-control",
	} {
		if seen[label] == "" {
			t.Fatalf("loadQueries(\"vector_search\") missing %s", label)
		}
	}
	if !strings.Contains(seen["vector-search/exact-knn-base-table"], "COSINE_DISTANCE(") {
		t.Fatalf("KNN probe missing COSINE_DISTANCE(): %s", seen["vector-search/exact-knn-base-table"])
	}
	if !strings.Contains(seen["vector-search/ann-auto-index"], "APPROX_COSINE_DISTANCE(") {
		t.Fatalf("ANN probe missing APPROX_COSINE_DISTANCE(): %s", seen["vector-search/ann-auto-index"])
	}
	if !strings.Contains(seen["vector-search/ann-dot-product"], "APPROX_DOT_PRODUCT(") {
		t.Fatalf("ANN probe missing APPROX_DOT_PRODUCT(): %s", seen["vector-search/ann-dot-product"])
	}
	if !strings.Contains(seen["vector-search/ann-euclidean-distance"], "APPROX_EUCLIDEAN_DISTANCE(") {
		t.Fatalf("ANN probe missing APPROX_EUCLIDEAN_DISTANCE(): %s", seen["vector-search/ann-euclidean-distance"])
	}
	if !strings.Contains(seen["vector-search/ann-back-join"], "Body") {
		t.Fatalf("back-join probe missing non-stored Body projection: %s", seen["vector-search/ann-back-join"])
	}
	if !strings.Contains(seen["vector-search/ann-gql-next-traversal"], "NEXT") {
		t.Fatalf("ANN GQL traversal probe missing NEXT: %s", seen["vector-search/ann-gql-next-traversal"])
	}
	hybrid := seen["vector-search/hybrid-rrf"]
	for _, want := range []string{"UNION ALL", "SEARCH(", "SCORE(", "APPROX_DOT_PRODUCT("} {
		if !strings.Contains(hybrid, want) {
			t.Errorf("hybrid RRF probe missing %q: %s", want, hybrid)
		}
	}
	control := seen["vector-search/hybrid-rrf-exact-control"]
	for _, want := range []string{"_BASE_TABLE", "DOT_PRODUCT("} {
		if !strings.Contains(control, want) {
			t.Errorf("hybrid RRF exact control missing %q: %s", want, control)
		}
	}
	if strings.Contains(control, "APPROX_DOT_PRODUCT(") {
		t.Errorf("hybrid RRF exact control unexpectedly uses ANN: %s", control)
	}
}

func TestSearchGraphExpectationManifestMatchesBuiltInCases(t *testing.T) {
	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "search_graph_expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Queries []struct {
			Label    string            `json:"label"`
			Patterns []json.RawMessage `json:"patterns"`
		} `json:"queries"`
		ExpectedQueryErrors []json.RawMessage `json:"expected_query_errors"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Version, "v0alpha1"; got != want {
		t.Errorf("search graph expectation version = %q, want %q", got, want)
	}
	if got, want := len(manifest.Queries), 3; got != want {
		t.Fatalf("search graph positive expectations = %d, want %d", got, want)
	}
	if got := len(manifest.ExpectedQueryErrors); got != 0 {
		t.Errorf("search graph error expectations = %d, want 0", got)
	}
	patternCount := 0
	queries, err := loadQueries("search_graph", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ddls, err := loadDDLs("search_graph", nil)
	if err != nil {
		t.Fatal(err)
	}
	joinedDDLs := strings.Join(ddls, "\n")
	for _, want := range []string{"CREATE TABLE SearchAlbums", "CREATE TABLE VectorDocuments", "CREATE PROPERTY GRAPH SearchMusicGraph", "CREATE PROPERTY GRAPH VectorGraph"} {
		if !strings.Contains(joinedDDLs, want) {
			t.Errorf("search_graph schema missing %q", want)
		}
	}
	seen := make(map[string]struct{}, len(queries))
	for _, query := range queries {
		seen[query.Label] = struct{}{}
	}
	for _, expectation := range manifest.Queries {
		if _, ok := seen[expectation.Label]; !ok {
			t.Errorf("search graph expectation label %q is absent from the built-in cases", expectation.Label)
		}
		if len(expectation.Patterns) == 0 {
			t.Errorf("search graph expectation label %q has no operator patterns", expectation.Label)
		}
		patternCount += len(expectation.Patterns)
	}
	if got, want := patternCount, 14; got != want {
		t.Errorf("search graph operator patterns = %d, want %d", got, want)
	}
}

func TestLoadQueriesTVF(t *testing.T) {
	queries, err := loadQueries("tvf", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "tvf", err)
	}
	if got, want := len(queries), 1; got != want {
		t.Fatalf("len(loadQueries(\"tvf\")) = %d, want %d", got, want)
	}
	if got, want := queries[0].Label, "tvf/change-stream"; got != want {
		t.Fatalf("TVF query label = %q, want %q", got, want)
	}
	if !strings.Contains(queries[0].SQL, "READ_EverythingStream") {
		t.Fatalf("TVF query SQL missing READ_EverythingStream: %s", queries[0].SQL)
	}
}

func TestLoadQueriesFunctionHint(t *testing.T) {
	queries, err := loadQueries("function_hint", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "function_hint", err)
	}
	if got, want := len(queries), 3; got != want {
		t.Fatalf("len(loadQueries(\"function_hint\")) = %d, want %d", got, want)
	}
	seen := map[string]string{}
	for _, query := range queries {
		seen[query.Label] = query.SQL
		if !strings.Contains(query.SQL, "SHA512(s.SingerInfo)") {
			t.Fatalf("%s SQL missing SHA512 probe: %s", query.Label, query.SQL)
		}
	}
	if strings.Contains(seen["function-hint/default_inline"], "DISABLE_INLINE") {
		t.Fatalf("default inline query unexpectedly has DISABLE_INLINE: %s", seen["function-hint/default_inline"])
	}
	if !strings.Contains(seen["function-hint/disable_inline_false"], "@{DISABLE_INLINE=FALSE}") {
		t.Fatalf("disable_inline_false query missing function hint: %s", seen["function-hint/disable_inline_false"])
	}
	if !strings.Contains(seen["function-hint/disable_inline_true"], "@{DISABLE_INLINE=TRUE}") {
		t.Fatalf("disable_inline_true query missing function hint: %s", seen["function-hint/disable_inline_true"])
	}
}

func TestLoadQueriesCTE(t *testing.T) {
	queries, err := loadQueries("cte", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "cte", err)
	}
	seen := map[string]string{}
	for _, query := range queries {
		seen[query.Label] = query.SQL
	}
	for _, label := range []string{
		"cte/constant-single-reference",
		"cte/constant-repeated-reference",
		"cte/deterministic-function-single-reference",
		"cte/deterministic-function-repeated-reference",
		"cte/current-timestamp-single-reference",
		"cte/current-timestamp-repeated-reference",
		"cte/table-single-reference",
		"cte/table-repeated-reference",
	} {
		if seen[label] == "" {
			t.Fatalf("loadQueries(\"cte\") missing %s", label)
		}
	}
	if !strings.Contains(seen["cte/deterministic-function-single-reference"], "SHA256") {
		t.Fatalf("deterministic CTE query missing SHA256: %s", seen["cte/deterministic-function-single-reference"])
	}
	if !strings.Contains(seen["cte/current-timestamp-single-reference"], "CURRENT_TIMESTAMP") {
		t.Fatalf("current timestamp CTE query missing CURRENT_TIMESTAMP: %s", seen["cte/current-timestamp-single-reference"])
	}
	if !strings.Contains(seen["cte/table-single-reference"], "FROM Singers") {
		t.Fatalf("table CTE query missing Singers reference: %s", seen["cte/table-single-reference"])
	}
}

func TestLoadDDLsTVFIncludesChangeStream(t *testing.T) {
	ddls, err := loadDDLs("tvf", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "tvf", err)
	}
	joined := strings.Join(ddls, "\n")
	for _, want := range []string{
		"CREATE TABLE Singers",
		"CREATE CHANGE STREAM EverythingStream",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("loadDDLs(\"tvf\") missing %q in:\n%s", want, joined)
		}
	}
}

func TestExpandOptimizerVersionMatrixUsesStatementHints(t *testing.T) {
	params := map[string]interface{}{"prefix": "A"}
	got := expandOptimizerVersionMatrix([]queryCase{
		{Label: "plain", SQL: "SELECT 1", Params: params},
		{Label: "hinted", SQL: "@{JOIN_METHOD=APPLY_JOIN, OPTIMIZER_VERSION=5} SELECT 1"},
	})
	if gotLen, wantLen := len(got), optimizerVersionCount*2; gotLen != wantLen {
		t.Fatalf("len(expandOptimizerVersionMatrix()) = %d, want %d", gotLen, wantLen)
	}
	if gotLabel, wantLabel := got[0].Label, "optimizer-version/v1/plain"; gotLabel != wantLabel {
		t.Fatalf("first label = %q, want %q", gotLabel, wantLabel)
	}
	if gotSQL, wantSQL := got[0].SQL, "@{OPTIMIZER_VERSION=1}\nSELECT 1"; gotSQL != wantSQL {
		t.Fatalf("first SQL = %q, want %q", gotSQL, wantSQL)
	}
	if gotParam, wantParam := got[0].Params["prefix"], "A"; gotParam != wantParam {
		t.Fatalf("first params[prefix] = %v, want %v", gotParam, wantParam)
	}
	last := len(got) - 1
	if gotLabel, wantLabel := got[last].Label, "optimizer-version/v9/hinted"; gotLabel != wantLabel {
		t.Fatalf("last label = %q, want %q", gotLabel, wantLabel)
	}
	if gotSQL, wantSQL := got[last].SQL, "@{OPTIMIZER_VERSION=9, JOIN_METHOD=APPLY_JOIN}\nSELECT 1"; gotSQL != wantSQL {
		t.Fatalf("last SQL = %q, want %q", gotSQL, wantSQL)
	}
}

func TestLoadQueriesStatementHintQueryMatrix(t *testing.T) {
	queries, err := loadQueries("statement_hint_query_matrix", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "statement_hint_query_matrix", err)
	}
	wantLen := len(documentedStatementHintVariants()) * len(docsQueries)
	if gotLen := len(queries); gotLen != wantLen {
		t.Fatalf("len(loadQueries(\"statement_hint_query_matrix\")) = %d, want %d", gotLen, wantLen)
	}
	var sawMultiAssignment bool
	var sawOptimizerReplacement bool
	for _, query := range queries {
		switch query.Label {
		case "statement-hint-query-matrix/hash_join_build_left/execution-plans/join":
			sawMultiAssignment = true
			if !strings.HasPrefix(query.SQL, "@{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT}\n") {
				t.Fatalf("multi-assignment query SQL = %q", query.SQL)
			}
		case "statement-hint-query-matrix/optimizer_version_latest/best-practices/order-by-desc-limit-back-join-optimizer-version-5":
			sawOptimizerReplacement = true
			if !strings.HasPrefix(query.SQL, "@{OPTIMIZER_VERSION=latest_version}\n") {
				t.Fatalf("optimizer replacement query SQL = %q", query.SQL)
			}
			if strings.Contains(query.SQL, "OPTIMIZER_VERSION=5") {
				t.Fatalf("optimizer replacement query retained old optimizer version: %q", query.SQL)
			}
		}
	}
	if !sawMultiAssignment {
		t.Fatal("statement hint query matrix missing hash_join_build_left/execution-plans/join")
	}
	if !sawOptimizerReplacement {
		t.Fatal("statement hint query matrix missing optimizer replacement case")
	}
}

func TestExpandAllowDistributedMergeMatrixUsesStatementHints(t *testing.T) {
	params := map[string]interface{}{"prefix": "A"}
	got := expandAllowDistributedMergeMatrix([]queryCase{
		{Label: "plain", SQL: "SELECT 1", Params: params},
		{Label: "hinted", SQL: "@{JOIN_METHOD=APPLY_JOIN, ALLOW_DISTRIBUTED_MERGE=TRUE} SELECT 1"},
	})
	if gotLen, wantLen := len(got), 6; gotLen != wantLen {
		t.Fatalf("len(expandAllowDistributedMergeMatrix()) = %d, want %d", gotLen, wantLen)
	}
	if gotLabel, wantLabel := got[0].Label, "allow-distributed-merge/default/plain"; gotLabel != wantLabel {
		t.Fatalf("first label = %q, want %q", gotLabel, wantLabel)
	}
	if gotSQL, wantSQL := got[0].SQL, "SELECT 1"; gotSQL != wantSQL {
		t.Fatalf("first SQL = %q, want %q", gotSQL, wantSQL)
	}
	if gotParam, wantParam := got[0].Params["prefix"], "A"; gotParam != wantParam {
		t.Fatalf("first params[prefix] = %v, want %v", gotParam, wantParam)
	}
	if gotLabel, wantLabel := got[2].Label, "allow-distributed-merge/false/plain"; gotLabel != wantLabel {
		t.Fatalf("third label = %q, want %q", gotLabel, wantLabel)
	}
	if gotSQL, wantSQL := got[2].SQL, "@{ALLOW_DISTRIBUTED_MERGE=FALSE}\nSELECT 1"; gotSQL != wantSQL {
		t.Fatalf("third SQL = %q, want %q", gotSQL, wantSQL)
	}
	if gotLabel, wantLabel := got[5].Label, "allow-distributed-merge/false/hinted"; gotLabel != wantLabel {
		t.Fatalf("last label = %q, want %q", gotLabel, wantLabel)
	}
	if gotSQL, wantSQL := got[5].SQL, "@{ALLOW_DISTRIBUTED_MERGE=FALSE, JOIN_METHOD=APPLY_JOIN}\nSELECT 1"; gotSQL != wantSQL {
		t.Fatalf("last SQL = %q, want %q", gotSQL, wantSQL)
	}
}

func TestPrintPlanCompactMetadata(t *testing.T) {
	plan := &spannerpb.QueryPlan{
		PlanNodes: []*spannerpb.PlanNode{
			{
				Index:       0,
				DisplayName: "Distributed Union",
				Metadata: mustStruct(t, map[string]interface{}{
					"preserve_subquery_order": true,
				}),
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{Type: "Input", ChildIndex: 1},
				},
			},
			{
				Index:       1,
				DisplayName: "Filter Scan",
				Metadata: mustStruct(t, map[string]interface{}{
					"seekable_key_size": float64(0),
				}),
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{Type: "Input", ChildIndex: 2},
					{Type: "Residual Condition", ChildIndex: 3},
				},
			},
			{
				Index:       2,
				DisplayName: "Scan",
				Metadata: mustStruct(t, map[string]interface{}{
					"Full scan":   true,
					"scan_method": "Automatic",
				}),
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{Type: "Timestamp Condition", ChildIndex: 4},
				},
			},
			{Index: 3, DisplayName: "Function"},
			{Index: 4, DisplayName: "Function"},
		},
	}

	var stdout bytes.Buffer
	if err := printPlanCompactDFSMetadata(&stdout, queryCase{Label: "timestamp"}, plan); err != nil {
		t.Fatalf("printPlanCompactDFSMetadata() error = %v", err)
	}
	want := "timestamp: Distributed Union{preserve_subquery_order=true} > Filter Scan{seekable_key_size=0; Function[Residual Condition]} > Scan{full_scan=true, scan_method=Automatic; Function[Timestamp Condition]}\n"
	if got := stdout.String(); got != want {
		t.Fatalf("printPlanCompactDFSMetadata() = %q, want %q", got, want)
	}
}

func TestCompactMetadataOperatorKeepsSameNameWithDifferentAnnotations(t *testing.T) {
	plan := &spannerpb.QueryPlan{
		PlanNodes: []*spannerpb.PlanNode{
			{
				Index:       0,
				DisplayName: "Scan",
				Metadata: mustStruct(t, map[string]interface{}{
					"scan_method": "Automatic",
				}),
			},
			{
				Index:       1,
				DisplayName: "Scan",
				Metadata: mustStruct(t, map[string]interface{}{
					"scan_method": "Row",
				}),
			},
			{
				Index:       2,
				DisplayName: "Scan",
				Metadata: mustStruct(t, map[string]interface{}{
					"scan_method": "Row",
				}),
			},
		},
	}

	var stdout bytes.Buffer
	if err := printPlanCompactDFSMetadata(&stdout, queryCase{Label: "scans"}, plan); err != nil {
		t.Fatalf("printPlanCompactDFSMetadata() error = %v", err)
	}
	want := "scans: Scan{scan_method=Automatic} > Scan{scan_method=Row}\n"
	if got := stdout.String(); got != want {
		t.Fatalf("printPlanCompactDFSMetadata() = %q, want %q", got, want)
	}
}

func TestPrintPlanCompactDFSUsesChildLinks(t *testing.T) {
	plan := &spannerpb.QueryPlan{
		PlanNodes: []*spannerpb.PlanNode{
			{Index: 0, DisplayName: "Serialize Result", ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "Input", ChildIndex: 2},
			}},
			{Index: 1, DisplayName: "Scan"},
			{Index: 2, DisplayName: "Filter Scan", ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "Input", ChildIndex: 1},
			}},
		},
	}

	var stdout bytes.Buffer
	if err := printPlanCompactDFS(&stdout, queryCase{Label: "dfs"}, plan); err != nil {
		t.Fatalf("printPlanCompactDFS() error = %v", err)
	}
	want := "dfs: Serialize Result > Filter Scan > Scan\n"
	if got := stdout.String(); got != want {
		t.Fatalf("printPlanCompactDFS() = %q, want %q", got, want)
	}
}

func TestPrintPlanCompactTreeShowsBranching(t *testing.T) {
	plan := &spannerpb.QueryPlan{
		PlanNodes: []*spannerpb.PlanNode{
			{
				Index:       0,
				DisplayName: "Hash Join",
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{Type: "Build", ChildIndex: 1},
					{Type: "Probe", ChildIndex: 2},
				},
			},
			{Index: 1, DisplayName: "Scan"},
			{
				Index:       2,
				DisplayName: "Filter Scan",
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{Type: "Input", ChildIndex: 3},
					{Type: "Residual Condition", ChildIndex: 4},
				},
			},
			{Index: 3, DisplayName: "Scan"},
			{Index: 4, DisplayName: "Function"},
		},
	}

	var stdout bytes.Buffer
	if err := printPlanCompactTree(&stdout, queryCase{Label: "join"}, plan, false, false); err != nil {
		t.Fatalf("printPlanCompactTree() error = %v", err)
	}
	want := "join: Hash Join(-[Build]-> Scan, -[Probe]-> Filter Scan -[Input]-> Scan)\n"
	if got := stdout.String(); got != want {
		t.Fatalf("printPlanCompactTree() = %q, want %q", got, want)
	}
}

func TestPrintPlanCompactTreeCanIncludeIndexes(t *testing.T) {
	plan := &spannerpb.QueryPlan{
		PlanNodes: []*spannerpb.PlanNode{
			{
				Index:       0,
				DisplayName: "Cross Apply",
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{Type: "Input", ChildIndex: 1},
					{Type: "Map", ChildIndex: 2},
				},
			},
			{Index: 1, DisplayName: "Scan"},
			{Index: 2, DisplayName: "Filter Scan", ChildLinks: []*spannerpb.PlanNode_ChildLink{
				{Type: "Input", ChildIndex: 3},
			}},
			{Index: 3, DisplayName: "Scan"},
		},
	}

	var stdout bytes.Buffer
	if err := printPlanCompactTree(&stdout, queryCase{Label: "apply"}, plan, false, true); err != nil {
		t.Fatalf("printPlanCompactTree() error = %v", err)
	}
	want := "apply: 0:Cross Apply(-[Input]-> 1:Scan, -[Map]-> 2:Filter Scan -[Input]-> 3:Scan)\n"
	if got := stdout.String(); got != want {
		t.Fatalf("printPlanCompactTree() = %q, want %q", got, want)
	}
}

func TestPrintPlanCompactTreeMetadataUsesAnnotations(t *testing.T) {
	plan := &spannerpb.QueryPlan{
		PlanNodes: []*spannerpb.PlanNode{
			{
				Index:       0,
				DisplayName: "Scan",
				Metadata: mustStruct(t, map[string]interface{}{
					"scan_type":   "SearchIndexScan",
					"scan_target": "SearchAlbumsTitleIndex",
				}),
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{Type: "Search Predicate", ChildIndex: 1},
				},
			},
			{Index: 1, DisplayName: "Search Predicate"},
		},
	}

	var stdout bytes.Buffer
	if err := printPlanCompactTree(&stdout, queryCase{Label: "search"}, plan, true, false); err != nil {
		t.Fatalf("printPlanCompactTree() error = %v", err)
	}
	want := "search: Scan{scan_target=SearchAlbumsTitleIndex, scan_type=SearchIndexScan; Search Predicate}\n"
	if got := stdout.String(); got != want {
		t.Fatalf("printPlanCompactTree() = %q, want %q", got, want)
	}
}

func TestCompactHiddenScalarChildAnnotations(t *testing.T) {
	plan := &spannerpb.QueryPlan{
		PlanNodes: []*spannerpb.PlanNode{
			{
				Index:       0,
				DisplayName: "Scan",
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{Type: "Residual Condition", ChildIndex: 1},
					{Type: "Timestamp Condition", ChildIndex: 2},
					{Type: "Residual Condition", ChildIndex: 3},
					{Type: "Search Predicate", ChildIndex: 4},
					{Type: "Output", ChildIndex: 5},
					{Type: "Input", ChildIndex: 6},
				},
			},
			{Index: 1, DisplayName: "Function"},
			{Index: 2, DisplayName: "Function"},
			{Index: 3, DisplayName: "Function"},
			{Index: 4, DisplayName: "Search Predicate"},
			{Index: 5},
			{Index: 6, Kind: spannerpb.PlanNode_RELATIONAL, DisplayName: "Scan"},
		},
	}

	got := compactHiddenScalarChildAnnotations(plan.GetPlanNodes()[0], compactTreeNodesByIndex(plan))
	want := []string{
		"Function[Residual Condition, Timestamp Condition]",
		"Search Predicate",
		"Unknown[Output]",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("compactHiddenScalarChildAnnotations() = %#v, want %#v", got, want)
	}
}

func TestPrintPlanCompactTreeHidesNonScalarExpressionNodes(t *testing.T) {
	plan := &spannerpb.QueryPlan{
		PlanNodes: []*spannerpb.PlanNode{
			{
				Index:       0,
				Kind:        spannerpb.PlanNode_RELATIONAL,
				DisplayName: "Serialize Result",
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{ChildIndex: 1},
				},
			},
			{
				Index:       1,
				Kind:        spannerpb.PlanNode_RELATIONAL,
				DisplayName: "Array Unnest",
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{ChildIndex: 2},
				},
			},
			{
				Index:       2,
				Kind:        spannerpb.PlanNode_SCALAR,
				DisplayName: "Array Constructor",
			},
		},
	}

	var stdout bytes.Buffer
	if err := printPlanCompactTree(&stdout, queryCase{Label: "array"}, plan, true, false); err != nil {
		t.Fatalf("printPlanCompactTree() error = %v", err)
	}
	want := "array: Serialize Result -> Array Unnest{Array Constructor}\n"
	if got := stdout.String(); got != want {
		t.Fatalf("printPlanCompactTree() = %q, want %q", got, want)
	}
}

func TestPrintPlanCompactTreeSuppressesScalarOnlyDescendants(t *testing.T) {
	plan := &spannerpb.QueryPlan{
		PlanNodes: []*spannerpb.PlanNode{
			{
				Index:       0,
				DisplayName: "Scan",
				Metadata: mustStruct(t, map[string]interface{}{
					"scan_type": "SearchIndexScan",
				}),
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{Type: "Search Predicate", ChildIndex: 1},
				},
			},
			{
				Index:       1,
				DisplayName: "Function",
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{ChildIndex: 2},
				},
			},
			{Index: 2, DisplayName: "Search Predicate"},
		},
	}

	var stdout bytes.Buffer
	if err := printPlanCompactTree(&stdout, queryCase{Label: "search"}, plan, true, false); err != nil {
		t.Fatalf("printPlanCompactTree() error = %v", err)
	}
	want := "search: Scan{scan_type=SearchIndexScan; Function[Search Predicate]}\n"
	if got := stdout.String(); got != want {
		t.Fatalf("printPlanCompactTree() = %q, want %q", got, want)
	}
}

func TestPrintPlanCompactTreeOmitsAnnotatedLinkWhenSameChildIsRendered(t *testing.T) {
	plan := &spannerpb.QueryPlan{
		PlanNodes: []*spannerpb.PlanNode{
			{
				Index:       0,
				Kind:        spannerpb.PlanNode_RELATIONAL,
				DisplayName: "Serialize Result",
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{ChildIndex: 1},
					{Type: "Scalar", ChildIndex: 1},
				},
			},
			{
				Index:       1,
				Kind:        spannerpb.PlanNode_SCALAR,
				DisplayName: "Array Subquery",
				ChildLinks: []*spannerpb.PlanNode_ChildLink{
					{ChildIndex: 2},
				},
			},
			{
				Index:       2,
				Kind:        spannerpb.PlanNode_RELATIONAL,
				DisplayName: "Scan",
			},
		},
	}

	var stdout bytes.Buffer
	if err := printPlanCompactTree(&stdout, queryCase{Label: "array"}, plan, true, false); err != nil {
		t.Fatalf("printPlanCompactTree() error = %v", err)
	}
	want := "array: Serialize Result -[Scalar]-> Array Subquery -> Scan\n"
	if got := stdout.String(); got != want {
		t.Fatalf("printPlanCompactTree() = %q, want %q", got, want)
	}
}

func TestPrintPlanJSONIncludesScalarOperators(t *testing.T) {
	plan := &spannerpb.QueryPlan{
		PlanNodes: []*spannerpb.PlanNode{
			{
				Index:       0,
				DisplayName: "Function",
				ShortRepresentation: &spannerpb.PlanNode_ShortRepresentation{
					Description: "SHA512($SingerInfo)",
				},
			},
		},
	}

	var stdout bytes.Buffer
	if err := printPlanJSON(&stdout, queryCase{Label: "function-hint", SQL: "SELECT SHA512(SingerInfo) FROM Singers"}, plan); err != nil {
		t.Fatalf("printPlanJSON() error = %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		`"query_label": "function-hint"`,
		`"display_name": "Function"`,
		`"description": "SHA512($SingerInfo)"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("printPlanJSON() missing %q in:\n%s", want, got)
		}
	}
}

func TestPrintPlanYAMLIncludesScalarOperators(t *testing.T) {
	plan := &spannerpb.QueryPlan{
		PlanNodes: []*spannerpb.PlanNode{
			{
				Index:       0,
				DisplayName: "Function",
				ShortRepresentation: &spannerpb.PlanNode_ShortRepresentation{
					Description: "SHA512($SingerInfo)",
				},
			},
		},
	}

	var stdout bytes.Buffer
	if err := printPlanYAML(&stdout, queryCase{Label: "function-hint", SQL: "SELECT SHA512(SingerInfo) FROM Singers"}, plan); err != nil {
		t.Fatalf("printPlanYAML() error = %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"query_label: function-hint",
		"display_name: Function",
		"description: SHA512($SingerInfo)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("printPlanYAML() missing %q in:\n%s", want, got)
		}
	}
}

func mustStruct(t *testing.T, fields map[string]interface{}) *structpb.Struct {
	t.Helper()
	st, err := structpb.NewStruct(fields)
	if err != nil {
		t.Fatalf("structpb.NewStruct() error = %v", err)
	}
	return st
}

func TestIsDMLStatement(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{
			name: "select",
			sql:  "SELECT * FROM Singers",
		},
		{
			name: "insert",
			sql:  "INSERT INTO Singers (SingerId) VALUES (1)",
			want: true,
		},
		{
			name: "update",
			sql:  "UPDATE Singers SET FirstName = 'A' WHERE SingerId = 1",
			want: true,
		},
		{
			name: "delete",
			sql:  "DELETE FROM Singers WHERE SingerId = 1",
			want: true,
		},
		{
			name: "statement hint before update",
			sql:  "@{PDML_MAX_PARALLELISM=10} UPDATE Singers SET FirstName = 'A' WHERE SingerId = 1",
			want: true,
		},
		{
			name: "cte select",
			sql:  "WITH CTE AS (SELECT 1) SELECT * FROM CTE",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDMLStatement(tt.sql); got != tt.want {
				t.Fatalf("isDMLStatement(%q) = %t, want %t", tt.sql, got, tt.want)
			}
		})
	}
}
