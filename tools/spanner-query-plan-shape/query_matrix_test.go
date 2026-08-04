package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
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
		"hint-position/rejected/pipe-finish",
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
		"distinct/index-prefix/groupby-scan-true-control",
		"distinct/base-table/groupby-scan-false-control",
		"distinct/rewrite-group-by-stream",
		"set-operation/union-distinct/rewrite-group-by-hash",
	} {
		if _, ok := seen[label]; !ok {
			t.Errorf("loadQueries(\"set_operation_distinct\") missing %q", label)
		}
	}
	if got, want := len(queries), 48; got != want {
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
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Version, "v0alpha1"; got != want {
		t.Errorf("expectation version = %q, want %q", got, want)
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
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("loadDDLs(\"full_text_search\") missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "MusicGraph") || strings.Contains(joined, "CREATE TABLE Singers") {
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
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("loadDDLs(\"vector_search\") missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "MusicGraph") || strings.Contains(joined, "CREATE TABLE Singers") {
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
	if !strings.Contains(seen["full-text-search/numeric-array-any"], "ARRAY_INCLUDES_ANY") {
		t.Fatalf("numeric array query missing ARRAY_INCLUDES_ANY(): %s", seen["full-text-search/numeric-array-any"])
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
	if !strings.Contains(seen["vector-search/ann-back-join"], "Body") {
		t.Fatalf("back-join probe missing non-stored Body projection: %s", seen["vector-search/ann-back-join"])
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
	got := expandOptimizerVersionMatrix([]queryCase{
		{Label: "plain", SQL: "SELECT 1"},
		{Label: "hinted", SQL: "@{JOIN_METHOD=APPLY_JOIN, OPTIMIZER_VERSION=5} SELECT 1"},
	})
	if gotLen, wantLen := len(got), 16; gotLen != wantLen {
		t.Fatalf("len(expandOptimizerVersionMatrix()) = %d, want %d", gotLen, wantLen)
	}
	if gotLabel, wantLabel := got[0].Label, "optimizer-version/v1/plain"; gotLabel != wantLabel {
		t.Fatalf("first label = %q, want %q", gotLabel, wantLabel)
	}
	if gotSQL, wantSQL := got[0].SQL, "@{OPTIMIZER_VERSION=1}\nSELECT 1"; gotSQL != wantSQL {
		t.Fatalf("first SQL = %q, want %q", gotSQL, wantSQL)
	}
	if gotLabel, wantLabel := got[15].Label, "optimizer-version/v8/hinted"; gotLabel != wantLabel {
		t.Fatalf("last label = %q, want %q", gotLabel, wantLabel)
	}
	if gotSQL, wantSQL := got[15].SQL, "@{OPTIMIZER_VERSION=8, JOIN_METHOD=APPLY_JOIN}\nSELECT 1"; gotSQL != wantSQL {
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
	got := expandAllowDistributedMergeMatrix([]queryCase{
		{Label: "plain", SQL: "SELECT 1"},
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
