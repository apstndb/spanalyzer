//go:build integration && omni

package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanemuboost"
)

const setOperationEquivalenceDDL = `
CREATE TABLE LeftRows (
  Id INT64 NOT NULL,
  K INT64,
  V STRING(20),
) PRIMARY KEY (Id);

CREATE TABLE RightRows (
  Id INT64 NOT NULL,
  K INT64,
  V STRING(20),
) PRIMARY KEY (Id);

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

func TestIntegrationSetOperationRewriteEquivalenceOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	image := strings.TrimSpace(os.Getenv("SPANALYZER_OMNI_IMAGE"))
	if image == "" {
		t.Fatal("set SPANALYZER_OMNI_IMAGE to the pinned Spanner Omni image under test")
	}
	ddls, err := parseBuiltInDDLs("set-operation-equivalence-schema.sql", setOperationEquivalenceDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	runtime := spanemuboost.NewLazyRuntime(
		spanemuboost.BackendOmni,
		spanemuboost.WithContainerImage(image),
	)
	t.Cleanup(func() { _ = runtime.Close() })
	clients, err := spanemuboost.OpenClients(t.Context(), runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(ddls),
	)
	if err != nil {
		t.Fatalf("OpenClients() error = %v", err)
	}
	t.Cleanup(func() { _ = clients.Close() })

	mutations := []*spanner.Mutation{
		spanner.Insert("LeftRows", []string{"Id", "K", "V"}, []any{int64(1), nil, "x"}),
		spanner.Insert("LeftRows", []string{"Id", "K", "V"}, []any{int64(2), nil, "x"}),
		spanner.Insert("LeftRows", []string{"Id", "K", "V"}, []any{int64(3), int64(1), nil}),
		spanner.Insert("LeftRows", []string{"Id", "K", "V"}, []any{int64(4), int64(1), nil}),
		spanner.Insert("LeftRows", []string{"Id", "K", "V"}, []any{int64(5), int64(1), "a"}),
		spanner.Insert("LeftRows", []string{"Id", "K", "V"}, []any{int64(6), int64(2), "b"}),
		spanner.Insert("LeftRows", []string{"Id", "K", "V"}, []any{int64(7), int64(3), nil}),
		spanner.Insert("RightRows", []string{"Id", "K", "V"}, []any{int64(1), nil, "x"}),
		spanner.Insert("RightRows", []string{"Id", "K", "V"}, []any{int64(2), int64(1), nil}),
		spanner.Insert("RightRows", []string{"Id", "K", "V"}, []any{int64(3), int64(1), nil}),
		spanner.Insert("RightRows", []string{"Id", "K", "V"}, []any{int64(4), int64(1), "z"}),
		spanner.Insert("RightRows", []string{"Id", "K", "V"}, []any{int64(5), int64(4), "q"}),
		spanner.Insert("SetOpR", []string{"Id", "K"}, []any{int64(1), int64(1)}),
		spanner.Insert("SetOpR", []string{"Id", "K"}, []any{int64(2), int64(1)}),
		spanner.Insert("SetOpR", []string{"Id", "K"}, []any{int64(3), int64(2)}),
		spanner.Insert("SetOpR", []string{"Id", "K"}, []any{int64(4), int64(3)}),
		spanner.Insert("SetOpS", []string{"Id", "K"}, []any{int64(1), int64(1)}),
		spanner.Insert("SetOpS", []string{"Id", "K"}, []any{int64(2), int64(2)}),
		spanner.Insert("SetOpS", []string{"Id", "K"}, []any{int64(3), int64(2)}),
		spanner.Insert("SetOpS", []string{"Id", "K"}, []any{int64(4), int64(4)}),
		spanner.Insert("SetOpT", []string{"Id", "K"}, []any{int64(1), int64(1)}),
		spanner.Insert("SetOpT", []string{"Id", "K"}, []any{int64(2), int64(4)}),
	}
	if _, err := clients.Client.Apply(t.Context(), mutations); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	nullableRewrites := []struct {
		name       string
		directSQL  string
		rewriteSQL string
		want       []string
	}{
		{
			name:       "intersect-distinct",
			directSQL:  `SELECT K, V FROM (SELECT K, V FROM LeftRows INTERSECT DISTINCT SELECT K, V FROM RightRows) ORDER BY K, V`,
			rewriteSQL: `SELECT DISTINCT l.K, l.V FROM LeftRows AS l WHERE EXISTS (SELECT 1 FROM RightRows AS r WHERE r.K IS NOT DISTINCT FROM l.K AND r.V IS NOT DISTINCT FROM l.V) ORDER BY l.K, l.V`,
			want:       []string{"NULL:x", "1:NULL"},
		},
		{
			name:       "except-distinct",
			directSQL:  `SELECT K, V FROM (SELECT K, V FROM LeftRows EXCEPT DISTINCT SELECT K, V FROM RightRows) ORDER BY K, V`,
			rewriteSQL: `SELECT DISTINCT l.K, l.V FROM LeftRows AS l WHERE NOT EXISTS (SELECT 1 FROM RightRows AS r WHERE r.K IS NOT DISTINCT FROM l.K AND r.V IS NOT DISTINCT FROM l.V) ORDER BY l.K, l.V`,
			want:       []string{"1:a", "2:b", "3:NULL"},
		},
	}
	for _, rewrite := range nullableRewrites {
		assertNullablePairQueryEqual(t, clients.Client, rewrite.name, rewrite.directSQL, rewrite.rewriteSQL, rewrite.want)
	}
	for version := 1; version <= 8; version++ {
		for _, rewrite := range nullableRewrites {
			directPlan, err := analyzePlan(t.Context(), clients.Client, queryCase{
				Label: rewrite.name + "/direct",
				SQL:   withOptimizerVersionStatementHint(rewrite.directSQL, version),
			})
			if err != nil {
				t.Fatalf("AnalyzeQuery(%s direct, v%d) error = %v", rewrite.name, version, err)
			}
			rewritePlan, err := analyzePlan(t.Context(), clients.Client, queryCase{
				Label: rewrite.name + "/rewrite",
				SQL:   withOptimizerVersionStatementHint(rewrite.rewriteSQL, version),
			})
			if err != nil {
				t.Fatalf("AnalyzeQuery(%s rewrite, v%d) error = %v", rewrite.name, version, err)
			}
			if got, want := countPlanNodes(directPlan, "Aggregate", ""), 4; got != want {
				t.Errorf("%s direct v%d physical Aggregate count = %d, want %d", rewrite.name, version, got, want)
			}
			if got, want := countPlanNodes(rewritePlan, "Aggregate", ""), 2; got != want {
				t.Errorf("%s NULL-safe rewrite v%d physical Aggregate count = %d, want %d", rewrite.name, version, got, want)
			}
		}
	}
	assertInt64QueryEqual(t, clients.Client, "three-way-intersect-distinct",
		`SELECT K FROM (SELECT K FROM SetOpR INTERSECT DISTINCT SELECT K FROM SetOpS INTERSECT DISTINCT SELECT K FROM SetOpT) ORDER BY K`,
		`SELECT DISTINCT r.K FROM SetOpR AS r WHERE EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} (SELECT 1 FROM SetOpS AS s WHERE s.K = r.K) AND EXISTS @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} (SELECT 1 FROM SetOpT AS t WHERE t.K = r.K) ORDER BY r.K`,
		[]int64{1},
	)
	assertInt64QueryEqual(t, clients.Client, "three-way-except-distinct",
		`SELECT K FROM (SELECT K FROM SetOpR EXCEPT DISTINCT SELECT K FROM SetOpS EXCEPT DISTINCT SELECT K FROM SetOpT) ORDER BY K`,
		`SELECT DISTINCT r.K FROM SetOpR AS r WHERE NOT EXISTS @{JOIN_METHOD=HASH_JOIN, HASH_JOIN_BUILD_SIDE=BUILD_LEFT} (SELECT 1 FROM SetOpS AS s WHERE s.K = r.K) AND NOT EXISTS @{JOIN_METHOD=APPLY_JOIN, BATCH_MODE=FALSE} (SELECT 1 FROM SetOpT AS t WHERE t.K = r.K) ORDER BY r.K`,
		[]int64{3},
	)
}

func assertNullablePairQueryEqual(t *testing.T, client *spanner.Client, name, directSQL, rewriteSQL string, want []string) {
	t.Helper()
	direct := readNullablePairs(t, client, directSQL)
	rewrite := readNullablePairs(t, client, rewriteSQL)
	if !slices.Equal(direct, want) || !slices.Equal(rewrite, want) {
		t.Fatalf("%s results: direct=%v rewrite=%v want=%v", name, direct, rewrite, want)
	}
}

func readNullablePairs(t *testing.T, client *spanner.Client, sql string) []string {
	t.Helper()
	var got []string
	err := client.Single().Query(t.Context(), spanner.Statement{SQL: sql}).Do(func(row *spanner.Row) error {
		var key spanner.NullInt64
		var value spanner.NullString
		if err := row.Columns(&key, &value); err != nil {
			return err
		}
		keyText := "NULL"
		if key.Valid {
			keyText = fmt.Sprint(key.Int64)
		}
		valueText := "NULL"
		if value.Valid {
			valueText = value.StringVal
		}
		got = append(got, keyText+":"+valueText)
		return nil
	})
	if err != nil {
		t.Fatalf("query failed: %v\nSQL: %s", err, sql)
	}
	return got
}

func assertInt64QueryEqual(t *testing.T, client *spanner.Client, name, directSQL, rewriteSQL string, want []int64) {
	t.Helper()
	direct := readInt64s(t, client, directSQL)
	rewrite := readInt64s(t, client, rewriteSQL)
	if !slices.Equal(direct, want) || !slices.Equal(rewrite, want) {
		t.Fatalf("%s results: direct=%v rewrite=%v want=%v", name, direct, rewrite, want)
	}
}

func readInt64s(t *testing.T, client *spanner.Client, sql string) []int64 {
	t.Helper()
	var got []int64
	err := client.Single().Query(t.Context(), spanner.Statement{SQL: sql}).Do(func(row *spanner.Row) error {
		var value int64
		if err := row.Columns(&value); err != nil {
			return err
		}
		got = append(got, value)
		return nil
	})
	if err != nil {
		t.Fatalf("query failed: %v\nSQL: %s", err, sql)
	}
	return got
}
